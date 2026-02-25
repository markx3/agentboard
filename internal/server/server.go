package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markx3/agentboard/internal/auth"
	"github.com/markx3/agentboard/internal/board"
)

type Server struct {
	hub          *Hub
	addr         string
	listener     net.Listener
	tunnelActive bool
}

func newUpgrader(tunnelActive bool) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if tunnelActive {
				return true // GitHub auth is the real gate
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser clients
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := u.Hostname()
			return host == "127.0.0.1" || host == "localhost" || host == "::1"
		},
	}
}

func New(svc board.Service, host string, port int) *Server {
	hub := NewHub(svc)
	addr := fmt.Sprintf("%s:%d", host, port)
	return &Server{
		hub:  hub,
		addr: addr,
	}
}

// SetListener sets an external listener (e.g. ngrok) for the server to use.
func (s *Server) SetListener(ln net.Listener) {
	s.listener = ln
	s.tunnelActive = true
}

// Hub returns the server's Hub for client count access.
func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run(ctx)

	upgrader := newUpgrader(s.tunnelActive)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWS(ctx, w, r, upgrader)
	})

	// Use pre-set listener (e.g. ngrok) or create a local one
	if s.listener == nil {
		var err error
		s.listener, err = net.Listen("tcp", s.addr)
		if err != nil {
			return fmt.Errorf("listening on %s: %w", s.addr, err)
		}
	}

	log.Printf("WebSocket server listening on %s", s.listener.Addr().String())

	srv := &http.Server{Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	if err := srv.Serve(s.listener); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

func (s *Server) handleWS(ctx context.Context, w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	// First message should be auth token
	var authMsg struct {
		Token string `json:"token"`
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err := conn.ReadJSON(&authMsg); err != nil {
		conn.WriteJSON(map[string]string{"error": "expected auth message"})
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{}) // clear deadline

	username, err := auth.VerifyTokenString(ctx, authMsg.Token)
	if err != nil {
		log.Printf("auth failed: %v", err)
		conn.WriteJSON(map[string]string{"error": "authentication failed"})
		conn.Close()
		return
	}

	client := newClient(s.hub, conn, username)
	s.hub.register <- client

	go client.writePump(ctx)
	go client.readPump(ctx)
}
