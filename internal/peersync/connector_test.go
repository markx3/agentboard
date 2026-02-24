package peersync

import "testing"

func TestBuildWSURL(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{
			name: "https URL to wss",
			addr: "https://abc.ngrok-free.app",
			want: "wss://abc.ngrok-free.app/ws",
		},
		{
			name: "http URL to ws",
			addr: "http://localhost:8080",
			want: "ws://localhost:8080/ws",
		},
		{
			name: "bare ngrok host",
			addr: "abc.ngrok-free.app",
			want: "wss://abc.ngrok-free.app/ws",
		},
		{
			name: "bare local address",
			addr: "127.0.0.1:8080",
			want: "ws://127.0.0.1:8080/ws",
		},
		{
			name: "bare localhost with port",
			addr: "localhost:9090",
			want: "ws://localhost:9090/ws",
		},
		{
			name: "https with trailing slash",
			addr: "https://abc.ngrok-free.app/",
			want: "wss://abc.ngrok-free.app/ws",
		},
		{
			name: "wss URL passthrough",
			addr: "wss://abc.ngrok-free.app",
			want: "wss://abc.ngrok-free.app/ws",
		},
		{
			name: "ws URL passthrough",
			addr: "ws://localhost:8080",
			want: "ws://localhost:8080/ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWSURL(tt.addr)
			if got != tt.want {
				t.Errorf("buildWSURL(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}
