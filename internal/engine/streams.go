package engine

import "time"

// RegisterStream explicitly registers a stream (called at startup for known inputs).
func (e *Engine) RegisterStream(source, stream string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := source + ":" + stream
	if _, ok := e.streams[key]; !ok {
		e.streams[key] = &StreamInfo{
			Source:   source,
			Stream:   stream,
			LogCount: 0,
			LastSeen: time.Now(),
			Active:   true,
		}
	}
}

// MarkStreamInactive marks a stream as inactive (e.g., file reader finished).
func (e *Engine) MarkStreamInactive(source, stream string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := source + ":" + stream
	if s, ok := e.streams[key]; ok {
		s.Active = false
	}
}
