package extension

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HTTPHook allows intercepting and mutating outgoing HTTP requests and incoming responses.
type HTTPHook interface {
	Name() string
	BeforeRequest(ctx context.Context, req *http.Request) error
	AfterResponse(ctx context.Context, req *http.Request, res *http.Response) error
}

// HookRegistry manages registered HTTP hooks.
type HookRegistry struct {
	mu    sync.RWMutex
	hooks []HTTPHook
}

// NewHookRegistry creates an initialized HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// Register adds an HTTP hook. If a hook with the same name exists, it is replaced.
func (r *HookRegistry) Register(hook HTTPHook) {
	if hook == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.hooks {
		if h.Name() == hook.Name() {
			r.hooks[i] = hook
			return
		}
	}
	r.hooks = append(r.hooks, hook)
}

// Unregister removes an HTTP hook by name.
func (r *HookRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var remaining []HTTPHook
	for _, h := range r.hooks {
		if h.Name() != name {
			remaining = append(remaining, h)
		}
	}
	r.hooks = remaining
}

// Get returns the hook matching name, or nil.
func (r *HookRegistry) Get(name string) HTTPHook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.hooks {
		if h.Name() == name {
			return h
		}
	}
	return nil
}

// All returns a slice of all registered hooks.
func (r *HookRegistry) All() []HTTPHook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]HTTPHook, len(r.hooks))
	copy(out, r.hooks)
	return out
}

// RunBeforeRequest executes BeforeRequest on all registered hooks in order.
func (r *HookRegistry) RunBeforeRequest(ctx context.Context, req *http.Request) error {
	r.mu.RLock()
	hooks := make([]HTTPHook, len(r.hooks))
	copy(hooks, r.hooks)
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.BeforeRequest(ctx, req); err != nil {
			return fmt.Errorf("hook %s failed before request: %w", h.Name(), err)
		}
	}
	return nil
}

// RunAfterResponse executes AfterResponse on all registered hooks in reverse order.
func (r *HookRegistry) RunAfterResponse(ctx context.Context, req *http.Request, res *http.Response) error {
	r.mu.RLock()
	hooks := make([]HTTPHook, len(r.hooks))
	copy(hooks, r.hooks)
	r.mu.RUnlock()

	for i := len(hooks) - 1; i >= 0; i-- {
		h := hooks[i]
		if err := h.AfterResponse(ctx, req, res); err != nil {
			return fmt.Errorf("hook %s failed after response: %w", h.Name(), err)
		}
	}
	return nil
}

// HeaderInjectionHook is an internal implementation of HTTPHook that injects static headers.
type HeaderInjectionHook struct {
	name    string
	headers map[string]string
}

// NewHeaderInjectionHook creates a hook that injects headers into each request.
func NewHeaderInjectionHook(name string, headers map[string]string) *HeaderInjectionHook {
	return &HeaderInjectionHook{
		name:    name,
		headers: headers,
	}
}

func (h *HeaderInjectionHook) Name() string { return h.name }

func (h *HeaderInjectionHook) BeforeRequest(ctx context.Context, req *http.Request) error {
	if req == nil {
		return nil
	}
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return nil
}

func (h *HeaderInjectionHook) AfterResponse(ctx context.Context, req *http.Request, res *http.Response) error {
	return nil
}

// TimingHook measures and records request-to-response duration.
type TimingHook struct {
	name      string
	starts    sync.Map
	Durations sync.Map
}

// NewTimingHook creates an initialized TimingHook.
func NewTimingHook(name string) *TimingHook {
	return &TimingHook{name: name}
}

func (h *TimingHook) Name() string { return h.name }

func (h *TimingHook) BeforeRequest(ctx context.Context, req *http.Request) error {
	if req != nil {
		h.starts.Store(req, time.Now())
	}
	return nil
}

func (h *TimingHook) AfterResponse(ctx context.Context, req *http.Request, res *http.Response) error {
	if req != nil {
		if val, ok := h.starts.LoadAndDelete(req); ok {
			startTime := val.(time.Time)
			h.Durations.Store(req, time.Since(startTime))
		}
	}
	return nil
}
