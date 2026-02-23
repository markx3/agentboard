package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/marcosfelipeeipper/agentboard/internal/auth"
	"github.com/marcosfelipeeipper/agentboard/internal/board"
)

type Server struct {
	hub      *Hub
	addr     string
	listener net.Listener
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local network use
	},
}

func New(svc board.Service, host string, port int) *Server {
	hub := NewHub(svc)
	addr := fmt.Sprintf("%s:%d", host, port)
	return &Server{
		hub:  hub,
		addr: addr,
	}
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWS(ctx, w, r)
	})

	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.addr, err)
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

func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) handleWS(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	// First message should be auth token
	var authMsg struct {
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&authMsg); err != nil {
		conn.WriteJSON(map[string]string{"error": "expected auth message"})
		conn.Close()
		return
	}

	username, err := auth.VerifyTokenString(ctx, authMsg.Token)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": "authentication failed: " + err.Error()})
		conn.Close()
		return
	}

	client := newClient(s.hub, conn, username)
	s.hub.register <- client

	go client.writePump(ctx)
	go client.readPump(ctx)
}
