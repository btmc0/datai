package relayproto

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	frames := []Frame{
		{
			Type:   TypeHTTPReq,
			ID:     "r1",
			Method: http.MethodPost,
			Path:   "/v1/sessions?tail=1",
			Header: http.Header{"Content-Type": {"application/json"}, "X-Test": {"a", "b"}},
			Body:   []byte(`{"ok":true}`),
		},
		{
			Type:   TypeHTTPResp,
			ID:     "r1",
			Status: http.StatusAccepted,
			Header: http.Header{"Content-Type": {"text/plain"}},
			Body:   []byte("accepted"),
		},
		{
			Type:        TypeWSData,
			ID:          "r2",
			MessageType: 2,
			Data:        []byte{0, 1, 2, 3, 255},
		},
		{
			Type:  TypeWSClose,
			ID:    "r2",
			Error: "closed",
		},
	}

	for _, in := range frames {
		b, err := Marshal(in)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", in.Type, err)
		}
		out, err := Unmarshal(b)
		if err != nil {
			t.Fatalf("Unmarshal(%s): %v", in.Type, err)
		}
		if !reflect.DeepEqual(out, in) {
			t.Errorf("round trip mismatch\nin:  %#v\nout: %#v", in, out)
		}
	}
}

func TestMarshalRejectsUnknownType(t *testing.T) {
	_, err := Marshal(Frame{Type: "wat"})
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestUnmarshalRejectsBadFrames(t *testing.T) {
	valid, err := Marshal(Frame{Type: TypeWSData, ID: "r1", MessageType: 2, Data: []byte("x")})
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string][]byte{
		"empty":     nil,
		"bad magic": append([]byte(nil), valid...),
		"truncated": valid[:len(valid)-1],
		"trailing":  append(append([]byte(nil), valid...), 0),
	}
	cases["bad magic"][2] = 'x'

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Unmarshal(data); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBinaryFrameAvoidsJSONBase64Overhead(t *testing.T) {
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i)
	}
	f := Frame{Type: TypeWSData, ID: "r1", MessageType: 2, Data: payload}

	binaryFrame, err := Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	jsonFrame, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(binaryFrame) >= len(jsonFrame) {
		t.Fatalf("binary frame size = %d, json frame size = %d; want binary smaller", len(binaryFrame), len(jsonFrame))
	}
	if float64(len(jsonFrame))/float64(len(binaryFrame)) < 1.25 {
		t.Fatalf("json/binary size ratio = %.2f, want visible base64 overhead", float64(len(jsonFrame))/float64(len(binaryFrame)))
	}
}
