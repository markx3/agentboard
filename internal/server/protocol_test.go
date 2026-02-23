package server_test

import (
	"encoding/json"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

func TestNewMessage(t *testing.T) {
	msg, err := server.NewMessage(server.MsgTaskCreate, "alice", server.TaskCreatePayload{
		Title:       "New Task",
		Description: "A new task",
	})
	if err != nil {
		t.Fatalf("creating message: %v", err)
	}

	if msg.Type != server.MsgTaskCreate {
		t.Errorf("got type %q, want %q", msg.Type, server.MsgTaskCreate)
	}
	if msg.Sender != "alice" {
		t.Errorf("got sender %q, want %q", msg.Sender, "alice")
	}

	// Verify payload is valid JSON
	var payload server.TaskCreatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshaling payload: %v", err)
	}
	if payload.Title != "New Task" {
		t.Errorf("got title %q, want %q", payload.Title, "New Task")
	}
}

func TestMessageRoundTrip(t *testing.T) {
	original, _ := server.NewMessage(server.MsgTaskMove, "bob", server.TaskMovePayload{
		TaskID:     "abc-123",
		FromColumn: "backlog",
		ToColumn:   "planning",
	})
	original.Seq = 42

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	var decoded server.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type mismatch: %q != %q", decoded.Type, original.Type)
	}
	if decoded.Seq != 42 {
		t.Errorf("seq mismatch: %d != 42", decoded.Seq)
	}
	if decoded.Sender != "bob" {
		t.Errorf("sender mismatch: %q != %q", decoded.Sender, "bob")
	}
}
