package tunnel

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.ngrok.com/ngrok/v2"
)

// Listen creates an ngrok tunnel and returns a net.Listener.
// It reads the authtoken from the NGROK_AUTHTOKEN environment variable.
func Listen(ctx context.Context) (net.Listener, error) {
	token := os.Getenv("NGROK_AUTHTOKEN")
	if token == "" {
		return nil, fmt.Errorf("--tunnel requires NGROK_AUTHTOKEN environment variable\nGet yours at https://dashboard.ngrok.com/get-started/your-authtoken")
	}

	ln, err := ngrok.Listen(ctx)
	if err != nil {
		// Sanitize error to avoid leaking authtoken
		msg := strings.ReplaceAll(err.Error(), token, "[REDACTED]")
		return nil, fmt.Errorf("creating ngrok tunnel: %s", msg)
	}
	return ln, nil
}

// URLFromListener extracts the public URL from an ngrok listener.
// Returns empty string if the listener is not an ngrok listener.
func URLFromListener(ln net.Listener) string {
	type urlProvider interface {
		URL() string
	}
	if u, ok := ln.(urlProvider); ok {
		return u.URL()
	}
	return ""
}
