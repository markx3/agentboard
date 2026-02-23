package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 30 * time.Second
	pingPeriod     = 15 * time.Second
	maxMessageSize = 64 * 1024 // 64KB
	rateLimitPerMin = 60
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
	joinedAt time.Time

	mu          sync.Mutex
	msgCount    int
	lastMinute  time.Time
}

func newClient(hub *Hub, conn *websocket.Conn, username string) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		username:   username,
		joinedAt:   time.Now(),
		lastMinute: time.Now(),
	}
}

func (c *Client) rateLimited() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.lastMinute) > time.Minute {
		c.msgCount = 0
		c.lastMinute = now
	}
	c.msgCount++
	return c.msgCount > rateLimitPerMin
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Close connection when context is cancelled to unblock ReadMessage
	go func() {
		<-ctx.Done()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("client %s read error: %v", c.username, err)
			}
			return
		}

		if c.rateLimited() {
			reject, _ := json.Marshal(Message{
				Type:    MsgSyncReject,
				Payload: mustMarshal(SyncRejectPayload{Reason: "rate limited"}),
			})
			c.send <- reject
			continue
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		msg.Sender = c.username

		c.hub.incoming <- clientMessage{client: c, message: msg}
	}
}

func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshal: %v", err))
	}
	return data
}
