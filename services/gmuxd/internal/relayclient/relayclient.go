// Package relayclient connects gmuxd to gmux-relayd over an outbound
// WebSocket and serves gmux HTTP/WebSocket traffic through that connection.
package relayclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gmuxapp/gmux/packages/relayproto"
	"nhooyr.io/websocket"
)

const (
	maxHTTPBody  = 16 << 20
	pingInterval = 15 * time.Second
	pingTimeout  = 10 * time.Second
)

// Config controls the outbound relay connection.
type Config struct {
	URL      string
	Token    string
	LocalURL string
}

// Run reconnects until ctx is canceled.
func Run(ctx context.Context, cfg Config) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := runOnce(ctx, cfg); err != nil && ctx.Err() == nil {
			log.Printf("relay: disconnected: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func runOnce(ctx context.Context, cfg Config) error {
	reqHeader := http.Header{}
	reqHeader.Set("Authorization", "Bearer "+cfg.Token)
	conn, _, err := websocket.Dial(ctx, cfg.URL, &websocket.DialOptions{HTTPHeader: reqHeader})
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(32 << 20)

	log.Printf("relay: connected to %s", cfg.URL)
	c := &client{
		conn:     conn,
		localURL: strings.TrimRight(cfg.LocalURL, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		ws: map[string]*websocket.Conn{},
	}

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 2)
	go func() { errc <- c.readLoop(connCtx) }()
	go func() { errc <- c.pingLoop(connCtx) }()

	err = <-errc
	cancel()
	c.closeAllWS("relay disconnected")
	return err
}

type client struct {
	conn       *websocket.Conn
	localURL   string
	httpClient *http.Client

	sendMu sync.Mutex
	mu     sync.Mutex
	ws     map[string]*websocket.Conn
}

func (c *client) pingLoop(ctx context.Context) error {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		c.sendMu.Lock()
		err := c.conn.Ping(pingCtx)
		c.sendMu.Unlock()
		cancel()
		if err != nil {
			return fmt.Errorf("relay ping failed: %w", err)
		}
	}
}

func (c *client) readLoop(ctx context.Context) error {
	for {
		typ, data, err := c.conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageBinary {
			continue
		}
		f, err := relayproto.Unmarshal(data)
		if err != nil {
			log.Printf("relay: bad frame: %v", err)
			continue
		}
		switch f.Type {
		case relayproto.TypeHTTPReq:
			go c.handleHTTP(ctx, f)
		case relayproto.TypeWSOpen:
			go c.handleWSOpen(ctx, f)
		case relayproto.TypeWSData:
			c.handleWSData(ctx, f)
		case relayproto.TypeWSClose:
			c.closeWS(f.ID, f.Error)
		}
	}
}

func (c *client) send(ctx context.Context, f relayproto.Frame) error {
	b, err := relayproto.Marshal(f)
	if err != nil {
		return err
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.conn.Write(ctx, websocket.MessageBinary, b)
}

func (c *client) handleHTTP(ctx context.Context, f relayproto.Frame) {
	if f.Method == "" || f.Path == "" {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadRequest, Body: []byte("bad relay request")})
		return
	}

	target, err := c.targetURL(f.Path)
	if err != nil {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadRequest, Body: []byte(err.Error())})
		return
	}
	req, err := http.NewRequestWithContext(ctx, f.Method, target, bytes.NewReader(f.Body))
	if err != nil {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadRequest, Body: []byte(err.Error())})
		return
	}
	req.Header = cloneHeader(f.Header)
	dropHopHeaders(req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadGateway, Body: []byte(err.Error())})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPBody+1))
	if err != nil {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadGateway, Body: []byte(err.Error())})
		return
	}
	if len(body) > maxHTTPBody {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: http.StatusBadGateway, Body: []byte("relay response too large")})
		return
	}

	_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeHTTPResp, ID: f.ID, Status: resp.StatusCode, Header: cloneHeader(resp.Header), Body: body})
}

func (c *client) handleWSOpen(ctx context.Context, f relayproto.Frame) {
	target, err := c.targetURL(f.Path)
	if err != nil {
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeWSResult, ID: f.ID, Status: http.StatusBadRequest, Error: err.Error()})
		return
	}
	wsURL := httpToWS(target)
	header := cloneHeader(f.Header)
	dropHopHeaders(header)
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		status := http.StatusBadGateway
		if resp != nil {
			status = resp.StatusCode
		}
		_ = c.send(ctx, relayproto.Frame{Type: relayproto.TypeWSResult, ID: f.ID, Status: status, Error: err.Error()})
		return
	}
	conn.SetReadLimit(4 << 20)
	c.mu.Lock()
	c.ws[f.ID] = conn
	c.mu.Unlock()
	if err := c.send(ctx, relayproto.Frame{Type: relayproto.TypeWSResult, ID: f.ID, Status: http.StatusSwitchingProtocols}); err != nil {
		c.closeWS(f.ID, "relay send failed")
		return
	}

	go func() {
		defer func() {
			c.closeWS(f.ID, "")
			_ = c.send(context.Background(), relayproto.Frame{Type: relayproto.TypeWSClose, ID: f.ID})
		}()
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if err := c.send(ctx, relayproto.Frame{Type: relayproto.TypeWSData, ID: f.ID, MessageType: int(typ), Data: data}); err != nil {
				return
			}
		}
	}()
}

func (c *client) handleWSData(ctx context.Context, f relayproto.Frame) {
	c.mu.Lock()
	conn := c.ws[f.ID]
	c.mu.Unlock()
	if conn == nil {
		return
	}
	if err := conn.Write(ctx, websocket.MessageType(f.MessageType), f.Data); err != nil {
		c.closeWS(f.ID, err.Error())
	}
}

func (c *client) closeAllWS(reason string) {
	c.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(c.ws))
	for id, conn := range c.ws {
		delete(c.ws, id)
		conns = append(conns, conn)
	}
	c.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close(websocket.StatusTryAgainLater, reason)
	}
}

func (c *client) closeWS(id, reason string) {
	c.mu.Lock()
	conn := c.ws[id]
	delete(c.ws, id)
	c.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, reason)
	}
}

func (c *client) targetURL(path string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("bad path %q", path)
	}
	u, err := url.Parse(c.localURL + path)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("bad local url scheme %q", u.Scheme)
	}
	return u.String(), nil
}

func httpToWS(s string) string {
	if strings.HasPrefix(s, "https://") {
		return "wss://" + strings.TrimPrefix(s, "https://")
	}
	return "ws://" + strings.TrimPrefix(s, "http://")
}

func cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, vv := range h {
		cp := make([]string, len(vv))
		copy(cp, vv)
		out[k] = cp
	}
	return out
}

func dropHopHeaders(h http.Header) {
	for _, k := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"} {
		h.Del(k)
	}
}
