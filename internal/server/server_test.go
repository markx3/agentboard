package server

import (
	"net/http"
	"testing"
)

func TestNewUpgraderOriginCheck(t *testing.T) {
	tests := []struct {
		name         string
		tunnelActive bool
		origin       string
		want         bool
	}{
		{"local no origin", false, "", true},
		{"local localhost", false, "http://localhost:3000", true},
		{"local 127.0.0.1", false, "http://127.0.0.1:3000", true},
		{"local ::1", false, "http://[::1]:3000", true},
		{"local remote rejected", false, "https://evil.com", false},
		{"tunnel no origin", true, "", true},
		{"tunnel localhost", true, "http://localhost:3000", true},
		{"tunnel remote allowed", true, "https://abc.ngrok-free.app", true},
		{"tunnel any origin", true, "https://evil.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgrader := newUpgrader(tt.tunnelActive)
			r, _ := http.NewRequest("GET", "/ws", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			got := upgrader.CheckOrigin(r)
			if got != tt.want {
				t.Errorf("CheckOrigin(tunnel=%v, origin=%q) = %v, want %v",
					tt.tunnelActive, tt.origin, got, tt.want)
			}
		})
	}
}

func TestHubClientCount(t *testing.T) {
	h := &Hub{
		clients: make(map[*Client]bool),
	}

	if got := h.ClientCount(); got != 0 {
		t.Errorf("initial ClientCount() = %d, want 0", got)
	}

	h.clientCount.Store(5)
	if got := h.ClientCount(); got != 5 {
		t.Errorf("ClientCount() = %d, want 5", got)
	}
}
