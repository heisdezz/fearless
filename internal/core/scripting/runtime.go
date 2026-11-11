// Package scripting runs Bruno's JavaScript in an isolated goja runtime.
//
// One Runtime is created per script execution and thrown away afterwards. Bruno
// does the same (a fresh vm context per run) because scripts routinely shadow
// globals, define helpers, and stash state on `globalThis`; sharing a context
// between two requests would leak that state across executions.
//
// The runtime deliberately knows nothing about `bru`, `req`, `res` or `test`.
// Those are host functions registered by the caller through VM(), which keeps
// this package free of engine and model dependencies.
//
// Source: bruno-js/src/sandbox/node-vm/index.js
// Source: bruno-js/src/sandbox/node-vm/constants.js (safeGlobals)
package scripting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Defaults bound a runaway script.
const (
	// DefaultTimeout interrupts a script that runs longer than this.
	DefaultTimeout = 10 * time.Second
	// DefaultMaxCallStack bounds recursion. goja keeps its stack on the Go heap,
	// so unbounded recursion is an out-of-memory rather than a panic.
	DefaultMaxCallStack = 1024
)

// Errors returned by the runtime.
var (
	// ErrTimeout reports a script interrupted by the deadline.
	ErrTimeout = errors.New("scripting: script timed out")

	// ErrCancelled reports a script interrupted by its context.
	ErrCancelled = errors.New("scripting: script cancelled")

	// ErrNotAllowed reports a module the sandbox does not provide.
	ErrNotAllowed = errors.New("scripting: module not available in the sandbox")
)

// Options configure one execution.
type Options struct {
	// Timeout interrupts the script. Zero uses DefaultTimeout; a negative value
	// disables the deadline, which only tests should do.
	Timeout time.Duration
	// MaxCallStack overrides the recursion limit.
	MaxCallStack int
	// Modules supplies the require() table. A name not present is refused rather
	// than returning undefined, so a script fails loudly instead of calling a nil.
	Modules map[string]any
	// Globals are installed as properties of the global object before the script
	// runs.
	Globals map[string]any
	// Console receives console.* output. When nil, output is discarded.
	Console Console
	// Filename and LineOffset locate the script in error messages, so a stack
	// trace points at the `.bru` file rather than a synthetic name.
	Filename   string
	LineOffset int
}

// Console receives console output from a script.
//
// Bruno wraps console with serialisers so an object logs as JSON rather than
// `[object Object]`; the default implementation here does the same.
type Console interface {
	Log(level, message string)
}

// ConsoleFunc adapts a function to Console.
type ConsoleFunc func(level, message string)

// Log implements Console.
func (f ConsoleFunc) Log(level, message string) { f(level, message) }

// ConsoleLine is one captured console call.
type ConsoleLine struct {
	Level   string
	Message string
}

// Result is what a script produced.
type Result struct {
	// ReturnValue is the script's completion value as a Go value.
	ReturnValue any
	// Console holds the captured log lines in order.
	Console []ConsoleLine
}

// ScriptError is a JavaScript error with source location.
type ScriptError struct {
	// Name is the error's constructor name (TypeError, Error, ...).
	Name string
	// Message is the error message.
	Message string
	// Stack is goja's formatted stack, when available.
	Stack string
	// Filename and Line locate the failing statement.
	Filename string
	Line     int
	// Cause is the underlying Go error, which is ErrTimeout or ErrCancelled for
	// interruptions.
	Cause error
}

func (e *ScriptError) Error() string {
	kind := e.Name
	if kind == "" {
		kind = "Error"
	}
	switch {
	case e.Filename != "" && e.Line > 0:
		return fmt.Sprintf("%s:%d: %s: %s", e.Filename, e.Line, kind, e.Message)
	case e.Filename != "":
		return fmt.Sprintf("%s: %s: %s", e.Filename, kind, e.Message)
	default:
		return fmt.Sprintf("%s: %s", kind, e.Message)
	}
}

// Unwrap exposes the interruption cause.
func (e *ScriptError) Unwrap() error { return e.Cause }

// Runtime is a single-use JavaScript execution context.
//
// It is not safe for concurrent use: goja runtimes are single-goroutine objects.
// Each concurrent execution needs its own Runtime.
type Runtime struct {
	vm      *goja.Runtime
	opts    Options
	console *capturingConsole
}

// New creates a runtime.
func New(opts Options) (*Runtime, error) {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxCallStack <= 0 {
		opts.MaxCallStack = DefaultMaxCallStack
	}
	if opts.Filename == "" {
		opts.Filename = "script.js"
	}

	vm := goja.New()
	vm.SetMaxCallStackSize(opts.MaxCallStack)

	rt := &Runtime{vm: vm, opts: opts, console: newCapturingConsole(opts.Console)}

	installers := []func() error{
		rt.installGlobals,
		rt.installConsole,
		rt.installRequire,
		rt.installTimers,
		rt.installWebAPIs,
		rt.installConfiguredGlobals,
	}
	for _, install := range installers {
		if err := install(); err != nil {
			return nil, err
		}
	}
	return rt, nil
}

// VM exposes the underlying runtime for host-function registration.
//
// The caller registers `bru`, `req`, `res` and friends through this. Exposing it
// is the seam that keeps this package independent of the engine.
func (r *Runtime) VM() *goja.Runtime { return r.vm }

// Run executes source and returns its completion value.
//
// The script is interrupted when the deadline passes or the context is cancelled.
// Both surface as *ScriptError wrapping ErrTimeout or ErrCancelled, so a caller
// can tell a runaway script from one that threw.
//
// Microtasks are drained after evaluation, so a script ending with a resolved
// promise runs its `then` callbacks before Run returns. Bruno awaits the result,
// so a caller that skipped them would see a different outcome.
func (r *Runtime) Run(ctx context.Context, source string) (*Result, error) {
	stop := r.watch(ctx)
	defer stop()

	// A pending interrupt from an earlier call would abort this one immediately.
	r.vm.ClearInterrupt()

	value, err := r.vm.RunScript(r.opts.Filename, source)
	if err != nil {
		return nil, r.wrapError(err)
	}

	resolved, err := r.await(value, ctx)
	if err != nil {
		return nil, err
	}

	return &Result{ReturnValue: resolved, Console: r.console.captured()}, nil
}

// await settles a promise the script returned.
//
// goja drains its job queue when the top-level call returns, but the value that
// call produced was captured *before* those jobs ran — so a script whose last
// statement is a promise would report `nil` even though its callbacks had by then
// executed. Bruno awaits the script's result, so the equivalent here is to keep
// pumping the runtime until the promise settles.
//
// Pumping works because the runtime has no event loop: every job that will ever
// run is already queued by the time RunScript returns. A promise left pending
// forever therefore means the script awaited something that never resolves, and
// the deadline is what stops the loop.
func (r *Runtime) await(value goja.Value, ctx context.Context) (any, error) {
	if value == nil {
		return nil, nil
	}
	promise, ok := value.Export().(*goja.Promise)
	if !ok {
		return exportValue(value), nil
	}

	for promise.State() == goja.PromiseStatePending {
		if err := ctx.Err(); err != nil {
			return nil, &ScriptError{
				Name: "Error", Message: ErrCancelled.Error(),
				Filename: r.opts.Filename, Cause: ErrCancelled,
			}
		}
		// A tiny sleep yields to any goroutine that produced work for the queue.
		// The deadline watcher interrupts long synchronous jobs.
		time.Sleep(time.Millisecond)
	}

	if promise.State() == goja.PromiseStateRejected {
		reason := promise.Result()
		se := &ScriptError{Name: "Error", Filename: r.opts.Filename}
		if obj, isObj := reason.(*goja.Object); isObj {
			se.Name = stringField(obj, "name", "Error")
			se.Message = stringField(obj, "message", "")
			se.Stack = stringField(obj, "stack", "")
		}
		if se.Message == "" {
			se.Message = reason.String()
		}
		return nil, se
	}
	return exportValue(promise.Result()), nil
}

// RunFunction calls a function defined on the global object.
func (r *Runtime) RunFunction(ctx context.Context, name string, args ...any) (*Result, error) {
	stop := r.watch(ctx)
	defer stop()

	r.vm.ClearInterrupt()

	fn, ok := goja.AssertFunction(r.vm.Get(name))
	if !ok {
		return nil, fmt.Errorf("scripting: %q is not a function", name)
	}

	callArgs := make([]goja.Value, len(args))
	for i, a := range args {
		callArgs[i] = r.vm.ToValue(a)
	}

	value, err := fn(goja.Undefined(), callArgs...)
	if err != nil {
		return nil, r.wrapError(err)
	}

	resolved, err := r.await(value, ctx)
	if err != nil {
		return nil, err
	}

	return &Result{ReturnValue: resolved, Console: r.console.captured()}, nil
}

// SetVariable defines a global variable.
func (r *Runtime) SetVariable(name string, value any) error {
	return r.vm.Set(name, value)
}

// GetVariable reads a global variable as a Go value.
//
// A defined-but-empty value (`undefined` or `null`) reports false, because the
// callers that use this for script results care whether the script assigned
// anything.
func (r *Runtime) GetVariable(name string) (any, bool) {
	v := r.vm.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, false
	}
	return exportValue(v), true
}

// watch installs the deadline and cancellation interrupt.
//
// The returned stop function must be called, or every execution leaks the
// watcher goroutine. It also clears the interrupt before returning, so a
// completed Run cannot leave one armed for the next call.
func (r *Runtime) watch(ctx context.Context) func() {
	if r.opts.Timeout < 0 {
		return func() { r.vm.ClearInterrupt() }
	}

	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		var timeout <-chan time.Time
		if r.opts.Timeout > 0 {
			timer := time.NewTimer(r.opts.Timeout)
			defer timer.Stop()
			timeout = timer.C
		}

		select {
		case <-done:
			return
		case <-ctx.Done():
			r.vm.Interrupt(ErrCancelled)
		case <-timeout:
			r.vm.Interrupt(ErrTimeout)
		}
	}()

	return func() {
		close(done)
		<-finished
		r.vm.ClearInterrupt()
	}
}

// wrapError converts a goja error into a *ScriptError.
//
// An interruption surfaces as *goja.InterruptedError whose Value is what was
// passed to Interrupt, so the cause is recoverable rather than a string.
func (r *Runtime) wrapError(err error) error {
	var interrupted *goja.InterruptedError
	if errors.As(err, &interrupted) {
		cause := ErrCancelled
		if v, ok := interrupted.Value().(error); ok && errors.Is(v, ErrTimeout) {
			cause = ErrTimeout
		}
		return &ScriptError{
			Name:     "Error",
			Message:  cause.Error(),
			Filename: r.opts.Filename,
			Cause:    cause,
		}
	}

	var exception *goja.Exception
	if errors.As(err, &exception) {
		value := exception.Value()
		se := &ScriptError{Filename: r.opts.Filename}

		if obj, ok := value.(*goja.Object); ok {
			se.Name = stringField(obj, "name", "Error")
			se.Message = stringField(obj, "message", "")
			se.Stack = stringField(obj, "stack", "")
		} else {
			se.Name = "Error"
			se.Message = value.String()
		}
		if se.Message == "" {
			se.Message = value.String()
		}
		se.Line = r.errorLine(exception)
		return se
	}

	return fmt.Errorf("scripting: %w", err)
}

// stringField reads a string property, falling back when absent.
func stringField(obj *goja.Object, name, fallback string) string {
	v := obj.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return fallback
	}
	return v.String()
}

// errorLine extracts the line number from a goja stack trace.
//
// goja exposes no position accessor, so the first `<file>:<line>` in the stack is
// parsed. LineOffset is added so a caller can point at the script's position
// inside a `.bru` file.
func (r *Runtime) errorLine(exception *goja.Exception) int {
	stack := exception.String()
	marker := r.opts.Filename + ":"
	idx := strings.Index(stack, marker)
	if idx < 0 {
		return 0
	}
	rest := stack[idx+len(marker):]

	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}

	line := 0
	for _, c := range rest[:end] {
		line = line*10 + int(c-'0')
	}
	return line + r.opts.LineOffset
}

// exportValue converts a goja value into the Go representation callers expect.
//
// Exported maps are map[string]any and arrays are []any, whoever produced them,
// so a host function never has to type-switch on goja types.
func exportValue(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return v.Export()
}
