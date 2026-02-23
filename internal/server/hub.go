package server

import (
	"context"
	"encoding/json"
	"log"

	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type clientMessage struct {
	client  *Client
	message Message
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	incoming   chan clientMessage
	sequencer  *Sequencer
	service    board.Service
}

func NewHub(svc board.Service) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		incoming:   make(chan clientMessage, 256),
		sequencer:  NewSequencer(),
		service:    svc,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			return

		case client := <-h.register:
			h.clients[client] = true
			log.Printf("peer joined: %s (%d total)", client.username, len(h.clients))

			// Send full state to new client
			h.sendFullSync(ctx, client)

			// Notify others
			h.broadcastExcept(client, MsgPeerJoin, PeerPayload{Username: client.username})

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("peer left: %s (%d remaining)", client.username, len(h.clients))
				h.broadcastAll(MsgPeerLeave, PeerPayload{Username: client.username})
			}

		case cm := <-h.incoming:
			h.handleMessage(ctx, cm)
		}
	}
}

func (h *Hub) handleMessage(ctx context.Context, cm clientMessage) {
	msg := cm.message

	switch msg.Type {
	case MsgTaskCreate:
		var p TaskCreatePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		task, err := h.service.CreateTask(ctx, p.Title, p.Description)
		if err != nil {
			h.sendReject(cm.client, err.Error())
			return
		}
		seq := h.sequencer.Next()
		msg.Seq = seq
		msg.Payload = mustMarshal(task)
		h.broadcastAllRaw(msg)

	case MsgTaskMove:
		var p TaskMovePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		if !db.TaskStatus(p.ToColumn).Valid() {
			h.sendReject(cm.client, "invalid status")
			return
		}
		if err := h.service.MoveTask(ctx, p.TaskID, db.TaskStatus(p.ToColumn)); err != nil {
			h.sendReject(cm.client, err.Error())
			return
		}
		seq := h.sequencer.Next()
		msg.Seq = seq
		h.broadcastAllRaw(msg)

	case MsgTaskDelete:
		var p TaskDeletePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		if err := h.service.DeleteTask(ctx, p.TaskID); err != nil {
			h.sendReject(cm.client, err.Error())
			return
		}
		seq := h.sequencer.Next()
		msg.Seq = seq
		h.broadcastAllRaw(msg)

	case MsgTaskClaim:
		var p TaskClaimPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		if err := h.service.ClaimTask(ctx, p.TaskID, cm.client.username); err != nil {
			h.sendReject(cm.client, err.Error())
			return
		}
		seq := h.sequencer.Next()
		msg.Seq = seq
		h.broadcastAllRaw(msg)

	case MsgTaskUnclaim:
		var p TaskUnclaimPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		if err := h.service.UnclaimTask(ctx, p.TaskID); err != nil {
			h.sendReject(cm.client, err.Error())
			return
		}
		seq := h.sequencer.Next()
		msg.Seq = seq
		h.broadcastAllRaw(msg)

	case MsgPing:
		ack, _ := json.Marshal(Message{Type: MsgPong, Sender: "server"})
		cm.client.send <- ack
	}
}

func (h *Hub) sendFullSync(ctx context.Context, client *Client) {
	tasks, err := h.service.ListTasks(ctx)
	if err != nil {
		log.Printf("failed to get tasks for sync: %v", err)
		return
	}
	msg, err := NewMessage(MsgSyncFull, "server", tasks)
	if err != nil {
		log.Printf("failed to create sync message: %v", err)
		return
	}
	msg.Seq = h.sequencer.Current()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal sync message: %v", err)
		return
	}
	client.send <- data
}

func (h *Hub) sendReject(client *Client, reason string) {
	msg, err := NewMessage(MsgSyncReject, "server", SyncRejectPayload{Reason: reason})
	if err != nil {
		log.Printf("failed to create reject message: %v", err)
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal reject message: %v", err)
		return
	}
	client.send <- data
}

func (h *Hub) broadcastAll(msgType string, payload interface{}) {
	msg, err := NewMessage(msgType, "server", payload)
	if err != nil {
		log.Printf("failed to create broadcast message: %v", err)
		return
	}
	msg.Seq = h.sequencer.Next()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal broadcast message: %v", err)
		return
	}
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) broadcastExcept(except *Client, msgType string, payload interface{}) {
	msg, err := NewMessage(msgType, "server", payload)
	if err != nil {
		log.Printf("failed to create broadcast message: %v", err)
		return
	}
	msg.Seq = h.sequencer.Next()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal broadcast message: %v", err)
		return
	}
	for client := range h.clients {
		if client == except {
			continue
		}
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) broadcastAllRaw(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal raw broadcast: %v", err)
		return
	}
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

