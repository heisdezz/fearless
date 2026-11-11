package scripting

import (
	"fmt"
	"sort"
	"sync"

	"github.com/dop251/goja"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// State is the script-visible view of one execution.
//
// It is plain data rather than a reference to the engine: a script mutates the
// scopes through the API bindings, and the caller reads the deltas afterwards.
// That keeps this package free of engine imports, so a script can run in the CLI
// runner exactly as it does in the GUI.
type State struct {
	// Scopes is the live variable set. Scripts read and write it.
	Scopes *vars.Scopes
	// Request is the request being executed, or nil in a pre-request context
	// before one exists.
	Request *model.Request
	// Result is the completed response, or nil in a pre-request script.
	Result *model.Result
	// RequestURL is the request URL after interpolation.
	RequestURL string
	// CollectionName and CollectionPath identify the collection.
	CollectionName string
	CollectionPath string
	// ExecutionMode is "runner" or "standalone".
	ExecutionMode string
	// EnvName is the active environment's display name.
	EnvName string
	// IsSafeMode reports whether mutating APIs are permitted.
	IsSafeMode bool
	// Interpolate resolves {{var}} against Scopes.
	Interpolate func(string) string
	// ResponseText is the decoded response body, for res.getBody.
	ResponseText string
	// BodyBytes is the raw response body, for res.getDataBuffer.
	BodyBytes []byte

	// Overrides a script may apply through req.*. The host reads these when it
	// builds the request, so a script can change behaviour without the engine
	// being imported here.
	maxRedirects       int
	timeoutMs          int
	disableJSONParsing bool
	failHandler        goja.Value
}

// Control collects the runner directives a script may set.
//
// Bruno exposes these on `bru.runner`; the runner reads them after the script
// finishes rather than acting on them mid-execution.
type Control struct {
	SkipRequest   bool
	StopExecution bool
	NextRequest   string
	// Scheduled sleeps requested through bru.sleep, accumulated so the host can
	// honour them without blocking the script's turn.
	SleepMillis int
}

// TestCollector accumulates `test()` results.
type TestCollector struct {
	mu      sync.Mutex
	results []model.TestResult
}

// Add appends a result.
func (c *TestCollector) Add(result model.TestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, result)
}

// Results returns a copy of the collected results.
func (c *TestCollector) Results() []model.TestResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]model.TestResult(nil), c.results...)
}

// Bindings is the API surface registered into a runtime.
type Bindings struct {
	state   *State
	control *Control
	tests   *TestCollector
	vm      *goja.Runtime
	// scripted records nested requests so the caller can emit events for them.
	scripted []ScriptedCall
}

// ScriptedCall records a nested request a script issued.
type ScriptedCall struct {
	Source string
	Method string
	URL    string
}

// Install registers bru, req, res, test and expect on rt.
//
// Every object is built from Go closures, so a script cannot reach a Go pointer
// or mutate host state except through the documented methods.
func Install(rt *Runtime, state *State, control *Control, tests *TestCollector) (*Bindings, error) {
	if rt == nil || state == nil {
		return nil, fmt.Errorf("scripting: Install needs a runtime and a state")
	}
	if control == nil {
		control = &Control{}
	}
	if tests == nil {
		tests = &TestCollector{}
	}

	b := &Bindings{state: state, control: control, tests: tests, vm: rt.VM()}

	installers := map[string]func() (goja.Value, error){
		"bru":    b.buildBru,
		"req":    b.buildReq,
		"res":    b.buildRes,
		"test":   b.buildTest,
		"expect": b.buildExpect,
	}
	// Deterministic order keeps a failure reproducible when two installs conflict.
	names := make([]string, 0, len(installers))
	for name := range installers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value, err := installers[name]()
		if err != nil {
			return nil, fmt.Errorf("scripting: install %s: %w", name, err)
		}
		if err := b.vm.Set(name, value); err != nil {
			return nil, fmt.Errorf("scripting: install %s: %w", name, err)
		}
	}
	return b, nil
}

// ScriptedCalls returns the nested requests the script issued.
func (b *Bindings) ScriptedCalls() []ScriptedCall {
	return append([]ScriptedCall(nil), b.scripted...)
}

// object builds a JS object from a method table.
//
// Values are normalised to native functions where they are Go funcs. goja only
// recognises the exact shapes `func(goja.FunctionCall) goja.Value` and
// `func(goja.FunctionCall, *goja.Runtime) goja.Value`; anything else — including
// `func(goja.FunctionCall) string` — becomes a Go function object whose first
// argument is converted from the call, which fails at the first call site with
// "could not convert … to goja.FunctionCall".
func (b *Bindings) object(methods map[string]any) (*goja.Object, error) {
	obj := b.vm.NewObject()
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := obj.Set(name, b.native(methods[name])); err != nil {
			return nil, fmt.Errorf("set %s: %w", name, err)
		}
	}
	return obj, nil
}

// native converts a method value into something goja treats as a JS function.
//
// goja only recognises the exact shapes `func(goja.FunctionCall) goja.Value` and
// `func(goja.FunctionCall, *goja.Runtime) goja.Value`. Any other signature —
// `bool`, `string`, no return — is turned into a Go function object whose first
// argument is converted from the call, which fails at the call site with
// "could not convert … to goja.FunctionCall". This rewrites those variants into
// the recognised shape so the method tables can stay plainly typed.
func (b *Bindings) native(value any) any {
	switch fn := value.(type) {
	case func(goja.FunctionCall) goja.Value:
		return fn
	case func(goja.FunctionCall) any:
		return func(call goja.FunctionCall) goja.Value { return b.vm.ToValue(fn(call)) }
	case func(goja.FunctionCall) bool:
		return func(call goja.FunctionCall) goja.Value { return b.vm.ToValue(fn(call)) }
	case func(goja.FunctionCall) string:
		return func(call goja.FunctionCall) goja.Value { return b.vm.ToValue(fn(call)) }
	case func(goja.FunctionCall) int:
		return func(call goja.FunctionCall) goja.Value { return b.vm.ToValue(fn(call)) }
	case func(goja.FunctionCall) int64:
		return func(call goja.FunctionCall) goja.Value { return b.vm.ToValue(fn(call)) }
	case func(goja.FunctionCall):
		return func(call goja.FunctionCall) goja.Value {
			fn(call)
			return goja.Undefined()
		}
	case func() goja.Value:
		return func(goja.FunctionCall) goja.Value { return fn() }
	case func() string:
		return func(goja.FunctionCall) goja.Value { return b.vm.ToValue(fn()) }
	case func() int:
		return func(goja.FunctionCall) goja.Value { return b.vm.ToValue(fn()) }
	case func() int64:
		return func(goja.FunctionCall) goja.Value { return b.vm.ToValue(fn()) }
	case func() bool:
		return func(goja.FunctionCall) goja.Value { return b.vm.ToValue(fn()) }
	case func():
		return func(goja.FunctionCall) goja.Value {
			fn()
			return goja.Undefined()
		}
	}
	return value
}

// throw raises a JS exception from Go code.
func (b *Bindings) throw(format string, args ...any) {
	panic(b.vm.NewTypeError(fmt.Sprintf(format, args...)))
}

// interpolate resolves variables through the state's resolver.
func (b *Bindings) interpolate(s string) string {
	if b.state.Interpolate != nil {
		return b.state.Interpolate(s)
	}
	if b.state.Scopes != nil {
		return vars.InterpolateWithScopes(s, *b.state.Scopes, vars.InterpolateOptions{})
	}
	return s
}

// scope returns the map for a scope name, creating it on demand.
func (b *Bindings) scope(name string) map[string]any {
	s := b.state.Scopes
	if s == nil {
		b.state.Scopes = &vars.Scopes{}
		s = b.state.Scopes
	}
	switch name {
	case "env":
		if s.Environment == nil {
			s.Environment = map[string]any{}
		}
		return s.Environment
	case "global":
		if s.Global == nil {
			s.Global = map[string]any{}
		}
		return s.Global
	case "collection":
		if s.Collection == nil {
			s.Collection = map[string]any{}
		}
		return s.Collection
	case "folder":
		if s.Folder == nil {
			s.Folder = map[string]any{}
		}
		return s.Folder
	case "request":
		if s.Request == nil {
			s.Request = map[string]any{}
		}
		return s.Request
	case "runtime":
		if s.Runtime == nil {
			s.Runtime = map[string]any{}
		}
		return s.Runtime
	default:
		b.throw("scripting: unknown variable scope %q", name)
		return nil
	}
}

// merged returns the fully resolved variable map.
func (b *Bindings) merged() map[string]any {
	if b.state.Scopes == nil {
		return map[string]any{}
	}
	return b.state.Scopes.Merge()
}

// exportMap converts a JS object to map[string]any.
func (b *Bindings) exportMap(value goja.Value) map[string]any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	exported, ok := value.Export().(map[string]any)
	if !ok {
		return nil
	}
	return exported
}

// asString renders a value the way a template would.
func asString(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

// throw raises a JS exception from Go code.
