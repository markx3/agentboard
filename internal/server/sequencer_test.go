package server_test

import (
	"sync"
	"testing"

	"github.com/markx3/agentboard/internal/server"
)

func TestSequencerMonotonic(t *testing.T) {
	seq := server.NewSequencer()

	prev := seq.Next()
	for i := 0; i < 100; i++ {
		next := seq.Next()
		if next <= prev {
			t.Errorf("sequence not monotonic: %d <= %d", next, prev)
		}
		prev = next
	}
}

func TestSequencerConcurrent(t *testing.T) {
	seq := server.NewSequencer()
	var wg sync.WaitGroup
	results := make(chan int64, 1000)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				results <- seq.Next()
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[int64]bool)
	for val := range results {
		if seen[val] {
			t.Errorf("duplicate sequence number: %d", val)
		}
		seen[val] = true
	}

	if len(seen) != 1000 {
		t.Errorf("got %d unique numbers, want 1000", len(seen))
	}
}

func TestSequencerCurrent(t *testing.T) {
	seq := server.NewSequencer()

	if seq.Current() != 0 {
		t.Errorf("initial current: got %d, want 0", seq.Current())
	}

	seq.Next()
	seq.Next()
	seq.Next()

	if seq.Current() != 3 {
		t.Errorf("after 3 Next(): got %d, want 3", seq.Current())
	}
}
