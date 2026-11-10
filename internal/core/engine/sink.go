package engine

import (
	"sync"

	"fearless/internal/core/engine/transport"
	"fearless/internal/core/model"
)

// EventSink receives progressive updates while a request executes.
//
// It is an interface rather than a channel so the engine stays synchronous and
// easily testable: Do returns when execution completes, and the calling layer
// decides what becomes an application bus event.
type EventSink interface {
	// Queued receives the snapshot of the request as queued for transmission.
	Queued(sent model.SentRequest)
	// Chunk receives a fragment of the response body as it arrives from transport.
	Chunk(data []byte)
	// Phase records a completed timeline phase (DNS, Connect, TLS, FirstByte, etc.).
	Phase(entry model.TimelineEntry)
	// ScriptOutput records text logged by pre/post scripts.
	ScriptOutput(level, text string)
	// Unresolved records any unresolved variable references.
	Unresolved(names []string)
}

// NopSink discards progressive events.
type NopSink struct{}

func (NopSink) Queued(model.SentRequest)    {}
func (NopSink) Chunk([]byte)                {}
func (NopSink) Phase(model.TimelineEntry)   {}
func (NopSink) ScriptOutput(string, string) {}
func (NopSink) Unresolved([]string)         {}

// RecordingSink records progressive events in memory for testing and assertion.
type RecordingSink struct {
	mu             sync.Mutex
	SentRequests   []model.SentRequest
	Chunks         [][]byte
	Phases         []model.TimelineEntry
	ScriptLogs     []ScriptLogEntry
	UnresolvedVars [][]string
}

// ScriptLogEntry records one script log message.
type ScriptLogEntry struct {
	Level string
	Text  string
}

func (s *RecordingSink) Queued(sent model.SentRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SentRequests = append(s.SentRequests, sent)
}

func (s *RecordingSink) Chunk(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.Chunks = append(s.Chunks, cp)
}

func (s *RecordingSink) Phase(entry model.TimelineEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Phases = append(s.Phases, entry)
}

func (s *RecordingSink) ScriptOutput(level, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ScriptLogs = append(s.ScriptLogs, ScriptLogEntry{Level: level, Text: text})
}

func (s *RecordingSink) Unresolved(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]string, len(names))
	copy(cp, names)
	s.UnresolvedVars = append(s.UnresolvedVars, cp)
}

// transportSinkBridge adapts an EventSink to the transport.Sink interface.
type transportSinkBridge struct {
	sink EventSink
}

var _ transport.Sink = (*transportSinkBridge)(nil)

func (b *transportSinkBridge) Chunk(data []byte) {
	if b != nil && b.sink != nil {
		b.sink.Chunk(data)
	}
}

func (b *transportSinkBridge) Phase(entry model.TimelineEntry) {
	if b != nil && b.sink != nil {
		b.sink.Phase(entry)
	}
}
