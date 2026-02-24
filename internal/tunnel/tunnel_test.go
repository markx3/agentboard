package tunnel

import (
	"os"
	"testing"
)

func TestListenMissingAuthtoken(t *testing.T) {
	// Ensure NGROK_AUTHTOKEN is unset
	orig := os.Getenv("NGROK_AUTHTOKEN")
	os.Unsetenv("NGROK_AUTHTOKEN")
	defer func() {
		if orig != "" {
			os.Setenv("NGROK_AUTHTOKEN", orig)
		}
	}()

	_, err := Listen(t.Context())
	if err == nil {
		t.Fatal("expected error when NGROK_AUTHTOKEN is missing")
	}

	want := "--tunnel requires NGROK_AUTHTOKEN"
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestURLFromListenerNonNgrok(t *testing.T) {
	// A regular net.Listener wouldn't have URL() method
	url := URLFromListener(nil)
	if url != "" {
		t.Errorf("URLFromListener(nil) = %q, want empty", url)
	}
}
