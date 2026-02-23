package server

import "sync"

// Sequencer assigns monotonically increasing sequence numbers to messages.
// First-write-wins conflict resolution.
type Sequencer struct {
	mu  sync.Mutex
	seq int64
}

func NewSequencer() *Sequencer {
	return &Sequencer{}
}

func (s *Sequencer) Next() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Sequencer) Current() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}
