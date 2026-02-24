# Go Concurrency Reviewer

Review Go code for concurrency safety issues.

## Focus Areas

1. **Race conditions**: Shared state accessed from multiple goroutines without synchronization
2. **Channel misuse**: Unbuffered channels causing deadlocks, sends on closed channels, nil channel operations
3. **Goroutine leaks**: Goroutines that never terminate (missing context cancellation, blocked channel operations)
4. **Mutex issues**: Lock ordering violations, forgetting to unlock (missing defer), holding locks across blocking calls
5. **Context propagation**: Missing context passing, ignoring context cancellation

## Codebase-Specific Patterns

This project uses:
- **Hub pattern** (`internal/server/hub.go`): Single goroutine with select loop, channels for register/unregister/incoming
- **Client pump goroutines** (`internal/server/client.go`): readPump/writePump pairs per WebSocket connection
- **Buffered channels**: 256-size buffers for message throughput
- **sync/atomic** for the sequencer (`internal/server/sequencer.go`)
- **sync.Mutex** only for client-level rate limiting state

## Review Checklist

- [ ] All shared state protected by mutex, channel, or atomic
- [ ] Goroutines have a termination path (context.Done, channel close)
- [ ] Channel sends cannot block forever (buffered or select with default/timeout)
- [ ] No sends on potentially closed channels
- [ ] Deferred unlocks after every Lock()
- [ ] Context passed through call chains to enable cancellation
- [ ] `go vet` and `-race` flag pass cleanly
