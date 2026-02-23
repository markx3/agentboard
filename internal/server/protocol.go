package server

import "encoding/json"

// Message is the wire protocol envelope for all WebSocket communication.
type Message struct {
	Type    string          `json:"type"`
	Seq     int64           `json:"seq"`
	Sender  string          `json:"sender"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Message types
const (
	MsgSyncFull    = "sync.full"
	MsgSyncAck     = "sync.ack"
	MsgSyncReject  = "sync.reject"
	MsgTaskCreate  = "task.create"
	MsgTaskUpdate  = "task.update"
	MsgTaskMove    = "task.move"
	MsgTaskDelete  = "task.delete"
	MsgTaskClaim   = "task.claim"
	MsgTaskUnclaim = "task.unclaim"
	MsgAgentStatus = "task.agent_status"
	MsgCommentAdd  = "comment.add"
	MsgPeerJoin    = "peer.join"
	MsgPeerLeave   = "peer.leave"
	MsgLeaderPromote = "leader.promote"
	MsgPing        = "ping"
	MsgPong        = "pong"
)

// Payload types for typed access

type TaskCreatePayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TaskMovePayload struct {
	TaskID     string `json:"task_id"`
	FromColumn string `json:"from_column"`
	ToColumn   string `json:"to_column"`
}

type TaskDeletePayload struct {
	TaskID string `json:"task_id"`
}

type TaskClaimPayload struct {
	TaskID   string `json:"task_id"`
	Assignee string `json:"assignee"`
}

type TaskUnclaimPayload struct {
	TaskID string `json:"task_id"`
}

type AgentStatusPayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"` // idle, active, error
}

type CommentAddPayload struct {
	TaskID string `json:"task_id"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

type PeerPayload struct {
	Username string `json:"username"`
}

type SyncRejectPayload struct {
	Reason string `json:"reason"`
}

func NewMessage(msgType string, sender string, payload interface{}) (Message, error) {
	var raw json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return Message{}, err
		}
		raw = data
	}
	return Message{
		Type:    msgType,
		Sender:  sender,
		Payload: raw,
	}, nil
}
