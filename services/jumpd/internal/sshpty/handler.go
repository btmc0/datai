package sshpty

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

const (
	handlerWriteTimeout = 3 * time.Second
	readBufSize         = 32 * 1024
)

// ServerInfo contains the connection details for a remote server.
// The handler caller is responsible for looking these up (e.g. from a DB).
type ServerInfo struct {
	Host       string
	Port       int
	User       string
	PrivateKey []byte // decrypted PEM
}

// ServerResolver looks up a server by ID and returns its connection info.
// Returns an error if the server is not found or the caller has no access.
type ServerResolver func(ctx context.Context, serverID string) (*ServerInfo, error)

// initMessage is the first JSON message the client sends after connecting.
type initMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Rows     int    `json:"rows"`
	Cols     int    `json:"cols"`
	Cmd      string `json:"cmd"`
}

// clientMessage is a subsequent message from the client.
type clientMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Rows int    `json:"rows"`
	Cols int    `json:"cols"`
}

// Handler returns an http.HandlerFunc that bridges a browser WebSocket
// to a remote SSH PTY session.
//
// Protocol:
//  1. Client sends init message: {"type":"init","server_id":"...","rows":24,"cols":80,"cmd":"pi"}
//  2. Client sends data:         {"type":"data","data":"ls\n"}
//  3. Client sends resize:       {"type":"resize","rows":40,"cols":120}
//  4. Server sends raw PTY output as binary WebSocket messages.
func Handler(resolve ServerResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("sshpty: ws accept: %v", err)
			return
		}
		conn.SetReadLimit(256 * 1024)

		ctx := r.Context()

		// Read init message.
		_, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("sshpty: read init: %v", err)
			conn.Close(websocket.StatusProtocolError, "expected init message")
			return
		}

		var init initMessage
		if err := json.Unmarshal(data, &init); err != nil || init.Type != "init" || init.ServerID == "" {
			conn.Close(websocket.StatusProtocolError, "invalid init message")
			return
		}
		if init.Rows <= 0 {
			init.Rows = 24
		}
		if init.Cols <= 0 {
			init.Cols = 80
		}

		// Resolve server connection details.
		info, err := resolve(ctx, init.ServerID)
		if err != nil {
			msg := fmt.Sprintf("server not found: %v", err)
			log.Printf("sshpty: %s", msg)
			conn.Close(websocket.StatusInternalError, msg)
			return
		}

		// Dial SSH.
		sess, err := Dial(info.Host, info.Port, info.User, info.PrivateKey)
		if err != nil {
			log.Printf("sshpty: dial %s: %v", init.ServerID, err)
			conn.Close(websocket.StatusInternalError, "ssh dial failed")
			return
		}

		// Allocate PTY and start command.
		if err := sess.RequestPTY("xterm-256color", init.Rows, init.Cols); err != nil {
			sess.Close()
			log.Printf("sshpty: pty: %v", err)
			conn.Close(websocket.StatusInternalError, "pty request failed")
			return
		}

		if init.Cmd != "" {
			err = sess.Start(init.Cmd)
		} else {
			err = sess.Shell()
		}
		if err != nil {
			sess.Close()
			log.Printf("sshpty: start: %v", err)
			conn.Close(websocket.StatusInternalError, "command start failed")
			return
		}

		log.Printf("sshpty: connected %s -> %s@%s:%d", init.ServerID, info.User, info.Host, info.Port)

		proxyCtx, proxyCancel := context.WithCancel(ctx)

		var wg sync.WaitGroup
		wg.Add(2)

		// SSH stdout → WebSocket (binary).
		go func() {
			defer wg.Done()
			defer proxyCancel()
			buf := make([]byte, readBufSize)
			for {
				n, err := sess.Read(buf)
				if n > 0 {
					writeCtx, cancel := context.WithTimeout(proxyCtx, handlerWriteTimeout)
					werr := conn.Write(writeCtx, websocket.MessageBinary, buf[:n])
					cancel()
					if werr != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()

		// WebSocket → SSH stdin.
		go func() {
			defer wg.Done()
			defer proxyCancel()
			for {
				_, raw, err := conn.Read(proxyCtx)
				if err != nil {
					return
				}

				var msg clientMessage
				if err := json.Unmarshal(raw, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case "data":
					if _, err := sess.Write([]byte(msg.Data)); err != nil {
						return
					}
				case "resize":
					if msg.Rows > 0 && msg.Cols > 0 {
						_ = sess.Resize(msg.Rows, msg.Cols)
					}
				}
			}
		}()

		// Wait for SSH command to exit, then signal proxy goroutines.
		go func() {
			_ = sess.Wait()
			proxyCancel()
		}()

		wg.Wait()

		sess.Close()
		conn.Close(websocket.StatusNormalClosure, "session ended")
		log.Printf("sshpty: disconnected %s", init.ServerID)
	}
}

// writeWS writes a WebSocket message with a timeout.
func writeWS(ctx context.Context, conn *websocket.Conn, typ websocket.MessageType, data []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, handlerWriteTimeout)
	defer cancel()
	return conn.Write(writeCtx, typ, data)
}

// BroadcastResize sends a terminal_resize event as a text message so the
// frontend can update its size state. Mirrors the protocol used by wsproxy.
func BroadcastResize(ctx context.Context, conn *websocket.Conn, rows, cols int) error {
	msg, _ := json.Marshal(map[string]any{
		"type": "terminal_resize",
		"rows": rows,
		"cols": cols,
	})
	return writeWS(ctx, conn, websocket.MessageText, msg)
}

// readPump is a helper that copies from an io.Reader into a channel.
// Used internally by the handler's SSH→WS goroutine.
func readPump(r io.Reader, ch chan<- []byte, done <-chan struct{}) {
	buf := make([]byte, readBufSize)
	for {
		select {
		case <-done:
			return
		default:
		}
		n, err := r.Read(buf)
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, buf[:n])
			select {
			case ch <- cp:
			case <-done:
				return
			}
		}
		if err != nil {
			return
		}
	}
}
