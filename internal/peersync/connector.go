package peersync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

type Connector struct {
	addr     string
	token    string
	mu       sync.Mutex
	conn     *websocket.Conn
	Messages chan server.Message
	done     chan struct{}
}

func NewConnector(addr, token string) *Connector {
	return &Connector{
		addr:     addr,
		token:    token,
		Messages: make(chan server.Message, 256),
		done:     make(chan struct{}),
	}
}

func (c *Connector) Connect(ctx context.Context) error {
	wsURL := buildWSURL(c.addr)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.addr, err)
	}

	// Send auth token
	authMsg := struct {
		Token string `json:"token"`
	}{Token: c.token}
	if err := conn.WriteJSON(authMsg); err != nil {
		conn.Close()
		return fmt.Errorf("sending auth: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start read pump
	go c.readPump(ctx)

	return nil
}

func (c *Connector) readPump(ctx context.Context) {
	defer func() {
		close(c.done)
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	// Close connection when context is cancelled to unblock ReadMessage
	go func() {
		<-ctx.Done()
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("connection lost: %v", err)
			}
			return
		}

		var msg server.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		select {
		case c.Messages <- msg:
		case <-ctx.Done():
			return
		}
	}
}

func (c *Connector) Send(msg server.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Connector) Done() <-chan struct{} {
	return c.done
}

// buildWSURL converts an address to a WebSocket URL.
//
// Supported formats:
//   - "https://abc.ngrok-free.app" → "wss://abc.ngrok-free.app/ws"
//   - "http://localhost:8080"      → "ws://localhost:8080/ws"
//   - "abc.ngrok-free.app"         → "wss://abc.ngrok-free.app/ws"
//   - "127.0.0.1:8080"            → "ws://127.0.0.1:8080/ws"
func buildWSURL(addr string) string {
	// If it has a scheme, parse and convert
	if strings.HasPrefix(addr, "https://") || strings.HasPrefix(addr, "http://") {
		u, err := url.Parse(addr)
		if err == nil {
			scheme := "ws"
			if u.Scheme == "https" {
				scheme = "wss"
			}
			path := u.Path
			if path == "" || path == "/" {
				path = "/ws"
			}
			return fmt.Sprintf("%s://%s%s", scheme, u.Host, path)
		}
	}

	// If it has a wss:// or ws:// scheme already, just ensure /ws path
	if strings.HasPrefix(addr, "wss://") || strings.HasPrefix(addr, "ws://") {
		u, err := url.Parse(addr)
		if err == nil {
			path := u.Path
			if path == "" || path == "/" {
				path = "/ws"
			}
			return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)
		}
	}

	// Bare host — detect known tunnel domains for wss, otherwise ws
	if needsWSS(addr) {
		return fmt.Sprintf("wss://%s/ws", addr)
	}
	return fmt.Sprintf("ws://%s/ws", addr)
}

// needsWSS returns true if the address looks like a tunnel/HTTPS domain.
func needsWSS(addr string) bool {
	host := addr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return strings.Contains(host, "ngrok") ||
		strings.Contains(host, ".app") ||
		strings.Contains(host, ".io")
}
