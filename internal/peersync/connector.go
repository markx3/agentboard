package peersync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
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
