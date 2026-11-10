package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/core/cookies"
	"fearless/internal/core/engine"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
	"fearless/internal/core/vars"
)

// RequestSender executes an HTTP request item through the engine.
type RequestSender interface {
	Send(ctx context.Context, cmd SendRequestCmd) (string, error)
	SendSync(ctx context.Context, cmd SendRequestCmd) (*model.Result, error)
}

// RequestCanceller cancels in-flight request executions.
type RequestCanceller interface {
	Cancel(cmd CancelRequestCmd) bool
	InFlight(itemUID string) bool
	ActiveCount() int
}

// SendRequestCmd defines the parameters for sending a request.
type SendRequestCmd struct {
	// RequestUID is optional. If empty, a unique ID is generated.
	RequestUID    string
	CollectionUID string
	ItemUID       string
	Env           string
	RuntimeVars   map[string]any
	PromptVars    map[string]string

	// Direct model overrides (used when collection/item are held outside the workspace store)
	Item        *model.Item
	Collection  *model.Collection
	Settings    *model.ItemSettings
	GlobalPrefs *model.Settings
}

// CancelRequestCmd defines the target of a cancellation.
type CancelRequestCmd struct {
	RequestUID string
	ItemUID    string
}

// RequestService manages request dispatch, cancellation registry, and event publication.
type RequestService struct {
	eng     engine.Engine
	store   *bus.Store
	bus     *bus.Bus
	jar     *cookies.Jar
	spooler *persistence.Spooler

	mu     sync.Mutex
	active map[string]*inFlight
	byItem map[string]string // itemUID -> requestUID
}

type inFlight struct {
	ctx           context.Context
	uid           string
	itemUID       string
	collectionUID string
	cancel        context.CancelFunc
	terminalSent  atomic.Bool
	done          chan struct{}
	result        *model.Result
	err           error
}

// NewRequestService returns a new RequestService.
func NewRequestService(
	eng engine.Engine,
	store *bus.Store,
	b *bus.Bus,
	jar *cookies.Jar,
	spooler *persistence.Spooler,
) *RequestService {
	if eng == nil {
		eng = engine.New()
	}
	return &RequestService{
		eng:     eng,
		store:   store,
		bus:     b,
		jar:     jar,
		spooler: spooler,
		active:  make(map[string]*inFlight),
		byItem:  make(map[string]string),
	}
}

// Send accepts a request for execution, registers it, emits RequestQueued,
// and starts execution in a goroutine. Returns the execution UID immediately.
func (s *RequestService) Send(ctx context.Context, cmd SendRequestCmd) (string, error) {
	inflight, engineIn, err := s.prepare(ctx, cmd)
	if err != nil {
		return "", err
	}

	go s.execute(inflight, engineIn)
	return inflight.uid, nil
}

// SendSync accepts a request, registers it, emits RequestQueued, and waits
// for execution to complete before returning.
func (s *RequestService) SendSync(ctx context.Context, cmd SendRequestCmd) (*model.Result, error) {
	inflight, engineIn, err := s.prepare(ctx, cmd)
	if err != nil {
		return nil, err
	}

	s.execute(inflight, engineIn)
	<-inflight.done
	return inflight.result, inflight.err
}

// prepare resolves item/collection, handles in-flight cancellation for the item,
// allocates request UID, registers in-flight state, and publishes RequestQueued.
func (s *RequestService) prepare(ctx context.Context, cmd SendRequestCmd) (*inFlight, engine.Input, error) {
	col, item, err := s.resolveTarget(cmd)
	if err != nil {
		return nil, engine.Input{}, err
	}

	effectiveItem := *item
	if item.Draft != nil {
		effectiveItem.Request = item.Draft
	}
	if effectiveItem.Request == nil {
		return nil, engine.Input{}, fmt.Errorf("%w: item %s has no request body or draft", ErrInvalidInput, item.Name)
	}

	requestUID := cmd.RequestUID
	if requestUID == "" {
		requestUID = newRequestUID()
	}

	s.mu.Lock()
	// Cancel any existing in-flight request for the same item
	if priorUID, exists := s.byItem[item.UID]; exists {
		if prior, ok := s.active[priorUID]; ok {
			prior.cancel()
		}
	}

	reqCtx, cancel := context.WithCancel(ctx)
	inflight := &inFlight{
		ctx:           reqCtx,
		uid:           requestUID,
		itemUID:       item.UID,
		collectionUID: col.UID,
		cancel:        cancel,
		done:          make(chan struct{}),
	}
	s.active[requestUID] = inflight
	s.byItem[item.UID] = requestUID
	s.mu.Unlock()

	// Publish RequestQueued
	s.publish(events.RequestQueued{
		UID:           requestUID,
		ItemUID:       item.UID,
		CollectionUID: col.UID,
		Environment:   cmd.Env,
		QueuedAt:      time.Now().UTC(),
	})

	// Resolve scopes
	scopes := s.resolveScopes(col, &effectiveItem, cmd)

	// Resolve settings
	settings := effectiveItem.Request.Settings
	if cmd.Settings != nil {
		settings = *cmd.Settings
	}

	globalPrefs := model.DefaultSettings()
	if cmd.GlobalPrefs != nil {
		globalPrefs = *cmd.GlobalPrefs
	} else if s.store != nil {
		snap := s.store.Snapshot()
		globalPrefs = snap.Settings
	}

	engineIn := engine.Input{
		Item:              &effectiveItem,
		Collection:        col,
		Ancestors:         col.Ancestors(item.UID),
		Scopes:            scopes,
		Settings:          settings,
		GlobalPreferences: globalPrefs,
		Sink:              s.newSink(requestUID),
		Jar:               s.jar,
		Spooler:           s.spooler,
		CollectionPath:    col.Path,
	}

	return inflight, engineIn, nil
}

// execute runs the request through the engine and emits exactly one terminal event.
func (s *RequestService) execute(req *inFlight, in engine.Input) {
	defer func() {
		s.mu.Lock()
		delete(s.active, req.uid)
		if s.byItem[req.itemUID] == req.uid {
			delete(s.byItem, req.itemUID)
		}
		s.mu.Unlock()
		req.cancel()
		close(req.done)
	}()

	res, err := s.eng.Do(req.ctx, in)
	req.result = res
	req.err = err

	// Guarantee: exactly one terminal event is published
	if !req.terminalSent.CompareAndSwap(false, true) {
		return
	}

	if err != nil {
		ev := events.NewError(events.ScopeSend, err).
			WithRequest(req.uid, req.itemUID, req.collectionUID)
		s.publish(ev)
		return
	}

	if res != nil && res.Err != "" && res.Status == 0 {
		ev := events.NewError(events.ScopeSend, errors.New(res.Err)).
			WithRequest(req.uid, req.itemUID, req.collectionUID)
		s.publish(ev)
		return
	}

	if res != nil {
		s.publish(events.ResponseReceived{
			UID:      req.uid,
			ItemUID:  req.itemUID,
			Received: *res,
		})
	}
}

func (s *RequestService) resolveTarget(cmd SendRequestCmd) (*model.Collection, *model.Item, error) {
	if cmd.Collection != nil && cmd.Item != nil {
		return cmd.Collection, cmd.Item, nil
	}

	if s.store == nil {
		return nil, nil, errors.New("commands: store not available")
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	item := col.FindItem(cmd.ItemUID)
	if item == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrItemNotFound, cmd.ItemUID)
	}

	return col, item, nil
}

func (s *RequestService) resolveScopes(col *model.Collection, item *model.Item, cmd SendRequestCmd) vars.Scopes {
	sc := vars.Scopes{
		Collection: make(map[string]any),
		Folder:     make(map[string]any),
		Request:    make(map[string]any),
		Runtime:    cmd.RuntimeVars,
		Prompt:     make(map[string]any),
	}

	// Collection variables
	if col != nil && col.Config.Vars != nil {
		for _, v := range col.Config.Vars {
			if v.Enabled && v.NameOrEmpty() != "" {
				sc.Collection[v.NameOrEmpty()] = v.Value
			}
		}
	}

	// Environment variables
	if col != nil && cmd.Env != "" {
		for _, env := range col.Environments {
			if env.Name == cmd.Env || env.UID == cmd.Env {
				sc.Environment = make(map[string]any)
				for _, v := range env.Variables {
					if v.Enabled && v.NameOrEmpty() != "" {
						sc.Environment[v.NameOrEmpty()] = v.Value
					}
				}
				break
			}
		}
	}

	// Folder variables (from outermost to innermost ancestor)
	if col != nil && item != nil {
		ancestors := col.Ancestors(item.UID)
		for _, folder := range ancestors {
			for _, v := range folder.Vars {
				if v.Enabled && v.NameOrEmpty() != "" {
					sc.Folder[v.NameOrEmpty()] = v.Value
				}
			}
		}
	}

	// Prompt variables
	for k, v := range cmd.PromptVars {
		sc.Prompt[k] = v
	}

	return sc
}

func (s *RequestService) publish(ev events.Event) {
	if s.bus != nil {
		s.bus.Publish(ev)
	}
}

// busSink forwards progressive engine events to the event bus.
type busSink struct {
	bus        *bus.Bus
	requestUID string
	chunkSeq   atomic.Uint64
}

func (s *RequestService) newSink(requestUID string) engine.EventSink {
	return &busSink{
		bus:        s.bus,
		requestUID: requestUID,
	}
}

func (b *busSink) Queued(sent model.SentRequest) {
	if b.bus != nil {
		b.bus.Publish(events.RequestSent{
			UID:     b.requestUID,
			Sent:    sent,
			Attempt: 1,
		})
	}
}

func (b *busSink) Chunk(data []byte) {
	if b.bus != nil {
		seq := b.chunkSeq.Add(1)
		b.bus.Publish(events.ResponseChunk{
			UID:  b.requestUID,
			Data: data,
			Seq:  seq,
		})
	}
}

func (b *busSink) Phase(entry model.TimelineEntry) {
	// Progressive phases recorded
}

func (b *busSink) ScriptOutput(level, text string) {
	if b.bus != nil {
		b.bus.Publish(events.ScriptOutput{
			UID:   b.requestUID,
			Level: level,
			Text:  text,
			At:    time.Now().UTC(),
		})
	}
}

func (b *busSink) Unresolved(names []string) {
	if b.bus != nil {
		b.bus.Publish(events.UnresolvedVariables{
			UID:   b.requestUID,
			Names: names,
		})
	}
}

func newRequestUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
