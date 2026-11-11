package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// RunnerController allows scripting or external callers to control runner flow.
type RunnerController interface {
	SetNextRequest(target string)
	SkipRequest()
	StopExecution()
}

// FolderRunner coordinates collection and folder execution runs.
type FolderRunner interface {
	RunFolder(ctx context.Context, cmd RunFolderCmd) (*RunnerSummary, error)
	CancelRunner(runnerUID string) bool
}

// RunFolderCmd defines the parameters for running a collection or folder.
type RunFolderCmd struct {
	RunnerUID     string
	CollectionUID string
	FolderUID     string // empty string runs root of collection
	Env           string
	RuntimeVars   map[string]any
	Recursive     bool
	DelayMs       int
	Bail          bool
	FilterTags    []string

	// Direct model override for headless runs
	Collection *model.Collection
}

// RunnerSummary summarizes the outcome of a folder execution run.
type RunnerSummary struct {
	RunnerUID     string        `json:"runnerUid"`
	CollectionUID string        `json:"collectionUid"`
	Total         int           `json:"total"`
	Passed        int           `json:"passed"`
	Failed        int           `json:"failed"`
	Skipped       int           `json:"skipped"`
	Duration      time.Duration `json:"duration"`
	Results       []*model.Result
}

// RunnerService manages folder runs, runner controllers, and run cancellation.
type RunnerService struct {
	sender RequestSender
	store  *bus.Store
	bus    *bus.Bus

	mu     sync.Mutex
	active map[string]*activeRun
}

type activeRun struct {
	uid           string
	collectionUID string
	cancel        context.CancelFunc
	mu            sync.Mutex
	nextOverride  string
	skipNext      bool
	stopped       bool
}

func (r *activeRun) SetNextRequest(target string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextOverride = target
}

func (r *activeRun) SkipRequest() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skipNext = true
}

func (r *activeRun) StopExecution() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	if r.cancel != nil {
		r.cancel()
	}
}

// NewRunnerService creates a new RunnerService.
func NewRunnerService(sender RequestSender, store *bus.Store, b *bus.Bus) *RunnerService {
	return &RunnerService{
		sender: sender,
		store:  store,
		bus:    b,
		active: make(map[string]*activeRun),
	}
}

// CancelRunner cancels an active collection run and its current in-flight request.
func (s *RunnerService) CancelRunner(runnerUID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run, ok := s.active[runnerUID]; ok {
		run.StopExecution()
		return true
	}
	return false
}

func newRunnerUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

// Controller returns the active RunnerController for a given runner UID, if running.
func (s *RunnerService) Controller(runnerUID string) (RunnerController, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.active[runnerUID]
	return run, ok
}

// ActiveRunners returns the count of runs currently executing.
func (s *RunnerService) ActiveRunners() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.active)
}

func (s *RunnerService) publish(ev events.Event) {
	if s.bus != nil {
		s.bus.Publish(ev)
	}
}

var ErrRunnerStopped = errors.New("runner: execution stopped")
