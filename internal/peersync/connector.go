package peersync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

type Connector struct {
	addr     string
	token    string
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
	url := fmt.Sprintf("ws://%s/ws", c.addr)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
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

	c.conn = conn

	// Start read pump
	go c.readPump(ctx)

	return nil
}

func (c *Connector) readPump(ctx context.Context) {
	defer func() {
		close(c.done)
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

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
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Connector) Done() <-chan struct{} {
	return c.done
}

// ConnectWithRetry attempts to connect with exponential backoff.
func (c *Connector) ConnectWithRetry(ctx context.Context, maxAttempts int) error {
	backoff := 500 * time.Millisecond
	for i := 0; i < maxAttempts; i++ {
		if err := c.Connect(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + time.Duration(rand.Intn(500))*time.Millisecond):
		}
		backoff *= 2
		if backoff > 10*time.Second {
			backoff = 10 * time.Second
		}
	}
	return fmt.Errorf("failed to connect after %d attempts", maxAttempts)
}
