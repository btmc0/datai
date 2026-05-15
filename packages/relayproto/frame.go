// Package relayproto defines gmux-specific relay frames exchanged between
// gmuxd and gmux-relayd over a single authenticated WebSocket.
package relayproto

import "net/http"

const (
	TypeHTTPReq  = "http_request"
	TypeHTTPResp = "http_response"
	TypeWSOpen   = "ws_open"
	TypeWSResult = "ws_open_result"
	TypeWSData   = "ws_data"
	TypeWSClose  = "ws_close"
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
