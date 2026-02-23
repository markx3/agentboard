package server

import "sync/atomic"

// Sequencer assigns monotonically increasing sequence numbers to messages.
// First-write-wins conflict resolution.
type Sequencer struct {
	seq atomic.Int64
}

func NewSequencer() *Sequencer {
	return &Sequencer{}
}

func (s *Sequencer) Next() int64 {
	return s.seq.Add(1)
}

func (s *Sequencer) Current() int64 {
	return s.seq.Load()
}
