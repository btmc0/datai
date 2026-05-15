// Package relayproto defines gmux-specific relay frames exchanged between
// gmuxd and gmux-relayd over a single authenticated WebSocket.
package relayproto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	TypeHTTPReq  = "http_request"
	TypeHTTPResp = "http_response"
	TypeWSOpen   = "ws_open"
	TypeWSResult = "ws_open_result"
	TypeWSData   = "ws_data"
	TypeWSClose  = "ws_close"
)

var wireMagic = [4]byte{'g', 'm', 'r', 1}

const (
	wireHTTPReq byte = iota + 1
	wireHTTPResp
	wireWSOpen
	wireWSResult
	wireWSData
	wireWSClose
)

// Frame is intentionally gmux-specific. It carries HTTP requests/responses and
// WebSocket messages only; it is not a generic TCP tunneling protocol.
type Frame struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`

	// HTTP request fields.
	Method string      `json:"method,omitempty"`
	Path   string      `json:"path,omitempty"`
	Header http.Header `json:"header,omitempty"`
	Body   []byte      `json:"body,omitempty"`

	// HTTP response / WebSocket open result fields.
	Status int    `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`

	// WebSocket data fields. MessageType uses nhooyr.io/websocket constants:
	// 1 = text, 2 = binary.
	MessageType int    `json:"message_type,omitempty"`
	Data        []byte `json:"data,omitempty"`
}

// Marshal encodes a relay frame for websocket binary transport.
func Marshal(f Frame) ([]byte, error) {
	typeCode, err := typeToWire(f.Type)
	if err != nil {
		return nil, err
	}
	if f.Status < 0 {
		return nil, fmt.Errorf("negative status %d", f.Status)
	}
	if f.MessageType < 0 || f.MessageType > 255 {
		return nil, fmt.Errorf("message type %d out of range", f.MessageType)
	}

	var headerBytes []byte
	if len(f.Header) > 0 {
		var err error
		headerBytes, err = json.Marshal(f.Header)
		if err != nil {
			return nil, fmt.Errorf("marshal header: %w", err)
		}
	}

	var w writer
	w.buf = make([]byte, 0, encodedSizeHint(f, len(headerBytes)))
	w.raw(wireMagic[:])
	w.byte(typeCode)
	w.string(f.ID)
	w.string(f.Method)
	w.string(f.Path)
	w.bytes(headerBytes)
	w.bytes(f.Body)
	w.uint32(uint32(f.Status))
	w.string(f.Error)
	w.byte(byte(f.MessageType))
	w.bytes(f.Data)
	return w.buf, nil
}

// Unmarshal decodes a websocket binary relay frame.
func Unmarshal(data []byte) (Frame, error) {
	r := reader{data: data}
	magic, err := r.raw(len(wireMagic))
	if err != nil {
		return Frame{}, err
	}
	if string(magic) != string(wireMagic[:]) {
		return Frame{}, fmt.Errorf("bad relay frame magic")
	}

	typeCode, err := r.byte()
	if err != nil {
		return Frame{}, err
	}
	typ, err := wireToType(typeCode)
	if err != nil {
		return Frame{}, err
	}
	id, err := r.string()
	if err != nil {
		return Frame{}, err
	}
	method, err := r.string()
	if err != nil {
		return Frame{}, err
	}
	path, err := r.string()
	if err != nil {
		return Frame{}, err
	}
	headerBytes, err := r.bytes()
	if err != nil {
		return Frame{}, err
	}
	body, err := r.bytes()
	if err != nil {
		return Frame{}, err
	}
	status, err := r.uint32()
	if err != nil {
		return Frame{}, err
	}
	errorText, err := r.string()
	if err != nil {
		return Frame{}, err
	}
	messageType, err := r.byte()
	if err != nil {
		return Frame{}, err
	}
	frameData, err := r.bytes()
	if err != nil {
		return Frame{}, err
	}
	if r.remaining() != 0 {
		return Frame{}, fmt.Errorf("relay frame has %d trailing bytes", r.remaining())
	}

	var header http.Header
	if len(headerBytes) > 0 {
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			return Frame{}, fmt.Errorf("unmarshal header: %w", err)
		}
	}

	return Frame{
		Type:        typ,
		ID:          id,
		Method:      method,
		Path:        path,
		Header:      header,
		Body:        body,
		Status:      int(status),
		Error:       errorText,
		MessageType: int(messageType),
		Data:        frameData,
	}, nil
}

func typeToWire(typ string) (byte, error) {
	switch typ {
	case TypeHTTPReq:
		return wireHTTPReq, nil
	case TypeHTTPResp:
		return wireHTTPResp, nil
	case TypeWSOpen:
		return wireWSOpen, nil
	case TypeWSResult:
		return wireWSResult, nil
	case TypeWSData:
		return wireWSData, nil
	case TypeWSClose:
		return wireWSClose, nil
	default:
		return 0, fmt.Errorf("unknown relay frame type %q", typ)
	}
}

func encodedSizeHint(f Frame, headerLen int) int {
	const lenPrefix = 4
	return len(wireMagic) + 1 +
		lenPrefix + len(f.ID) +
		lenPrefix + len(f.Method) +
		lenPrefix + len(f.Path) +
		lenPrefix + headerLen +
		lenPrefix + len(f.Body) +
		4 +
		lenPrefix + len(f.Error) +
		1 +
		lenPrefix + len(f.Data)
}

func wireToType(code byte) (string, error) {
	switch code {
	case wireHTTPReq:
		return TypeHTTPReq, nil
	case wireHTTPResp:
		return TypeHTTPResp, nil
	case wireWSOpen:
		return TypeWSOpen, nil
	case wireWSResult:
		return TypeWSResult, nil
	case wireWSData:
		return TypeWSData, nil
	case wireWSClose:
		return TypeWSClose, nil
	default:
		return "", fmt.Errorf("unknown relay frame type code %d", code)
	}
}

type writer struct{ buf []byte }

func (w *writer) byte(v byte) {
	w.buf = append(w.buf, v)
}

func (w *writer) uint32(v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *writer) string(v string) {
	w.bytes([]byte(v))
}

func (w *writer) raw(v []byte) {
	w.buf = append(w.buf, v...)
}

func (w *writer) bytes(v []byte) {
	w.uint32(uint32(len(v)))
	w.raw(v)
}

type reader struct {
	data []byte
	off  int
}

func (r *reader) remaining() int {
	return len(r.data) - r.off
}

func (r *reader) byte() (byte, error) {
	if r.remaining() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	v := r.data[r.off]
	r.off++
	return v, nil
}

func (r *reader) uint32() (uint32, error) {
	if r.remaining() < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint32(r.data[r.off:])
	r.off += 4
	return v, nil
}

func (r *reader) string() (string, error) {
	b, err := r.bytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *reader) raw(n int) ([]byte, error) {
	if r.remaining() < n {
		return nil, io.ErrUnexpectedEOF
	}
	if n == 0 {
		return nil, nil
	}
	v := r.data[r.off : r.off+n]
	r.off += n
	return v, nil
}

func (r *reader) bytes() ([]byte, error) {
	n, err := r.uint32()
	if err != nil {
		return nil, err
	}
	if uint32(r.remaining()) < n {
		return nil, io.ErrUnexpectedEOF
	}
	return r.raw(int(n))
}
