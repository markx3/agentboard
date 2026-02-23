package peersync

import (
	"context"
	"fmt"
	"log"
	"net"
	"path/filepath"

	boardpkg "github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

type Role int

const (
	RoleClient Role = iota
	RoleLeader
)

func (r Role) String() string {
	switch r {
	case RoleLeader:
		return "leader"
	default:
		return "client"
	}
}

type PeerState struct {
	Role       Role
	Server     *server.Server
	Connector  *Connector
	ServerAddr string
}

// StartOrConnect tries to connect to an existing server, or becomes the leader.
func StartOrConnect(ctx context.Context, connectAddr string, token string) (*PeerState, error) {
	// Explicit connect address takes priority
	if connectAddr != "" {
		conn := NewConnector(connectAddr, token)
		if err := conn.Connect(ctx); err != nil {
			return nil, fmt.Errorf("connecting to %s: %w", connectAddr, err)
		}
		return &PeerState{
			Role:       RoleClient,
			Connector:  conn,
			ServerAddr: connectAddr,
		}, nil
	}

	// Try to discover existing server
	info, err := ReadServerInfo()
	if err == nil {
		conn := NewConnector(info.Addr, token)
		if err := conn.Connect(ctx); err == nil {
			return &PeerState{
				Role:       RoleClient,
				Connector:  conn,
				ServerAddr: info.Addr,
			}, nil
		}
		log.Printf("stale server at %s, becoming leader", info.Addr)
	}

	// Become leader
	dbPath := filepath.Join(".agentboard", "board.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	svc := boardpkg.NewLocalService(database)
	srv := server.New(svc, "127.0.0.1", 0)

	go func() {
		if err := srv.Start(ctx); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// Wait a moment for listener to be ready, then write server info
	// The address will be available once Start begins listening
	addr := waitForAddr(srv)
	if addr == "" {
		return nil, fmt.Errorf("server failed to start")
	}

	if err := WriteServerInfo(addr); err != nil {
		log.Printf("warning: could not write server info: %v", err)
	}

	return &PeerState{
		Role:       RoleLeader,
		Server:     srv,
		ServerAddr: addr,
	}, nil
}

func waitForAddr(srv *server.Server) string {
	// Poll for addr (the listener needs a moment to bind)
	for i := 0; i < 50; i++ {
		addr := srv.Addr()
		if addr != "" && addr != ":0" {
			_, port, _ := net.SplitHostPort(addr)
			if port != "0" && port != "" {
				return addr
			}
		}
		// Small sleep via channel
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			// ~10ms delay
		}()
		<-ch
	}
	return ""
}
