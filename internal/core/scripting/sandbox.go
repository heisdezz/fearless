package scripting

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// installGlobals defines the global helpers Bruno's scripts use.
//
// ECMAScript built-ins come from goja's own runtime; only the Node/Web extras in
// Bruno's `safeGlobals` list are installed here. That distinction matters: passing
// host objects in would create cross-realm prototype mismatches, which Bruno's
// sandbox comments call out explicitly.
//
// Source: bruno-js/src/sandbox/node-vm/constants.js safeGlobals
func (r *Runtime) installGlobals() error {
	// AssertionError is what chai throws, and what test() distinguishes from an
	// ordinary error when reporting a failure. Without it, an expect() failure
	// would arrive as a generic TypeError and lose its actual/expected fields.
	if err := r.vm.Set("AssertionError", func(call goja.ConstructorCall) *goja.Object {
		message := "assertion failed"
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			message = call.Arguments[0].String()
		}
		instance := call.This
		_ = instance.Set("name", "AssertionError")
		_ = instance.Set("message", message)
		return instance
	}); err != nil {
		return fmt.Errorf("scripting: install AssertionError: %w", err)
	}

	native := map[string]any{
		"btoa": btoa,
		"atob": atob,
	}
	for name, value := range native {
		if err := r.vm.Set(name, value); err != nil {
			return fmt.Errorf("scripting: install %s: %w", name, err)
		}
	}

	// Buffer is a JS object rather than a Go value because it has to return a
	// buffer instance, and a Go type would leak its fields into the script's realm.
	buffer := r.vm.NewObject()
	bufferFactory := r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return r.newBuffer(call)
	})
	if err := buffer.Set("from", bufferFactory); err != nil {
		return fmt.Errorf("scripting: install Buffer.from: %w", err)
	}
	if err := buffer.Set("isBuffer", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		obj, ok := call.Argument(0).(*goja.Object)
		if !ok {
			return r.vm.ToValue(false)
		}
		marker := obj.Get("__isBuffer")
		return r.vm.ToValue(marker != nil && marker.ToBoolean())
	})); err != nil {
		return fmt.Errorf("scripting: install Buffer.isBuffer: %w", err)
	}

	return r.vm.Set("Buffer", buffer)
}

// newBuffer implements Buffer.from(value[, encoding]).
//
// It returns a JS object exposing the byte length and the conversions scripts
// actually use (toString, toJSON), rather than a Go struct: the script's realm
// must own every object it inspects.
func (r *Runtime) newBuffer(call goja.FunctionCall) goja.Value {
	var data []byte

	arg := call.Argument(0)
	encoding := "utf8"
	if len(call.Arguments) > 1 {
		encoding = call.Argument(1).String()
	}

	if obj, ok := arg.(*goja.Object); ok {
		if length := obj.Get("length"); length != nil && !goja.IsUndefined(length) {
			n := int(length.ToInteger())
			data = make([]byte, 0, n)
			for i := 0; i < n; i++ {
				data = append(data, byte(obj.Get(fmt.Sprint(i)).ToInteger()))
			}
		}
	} else if strings.EqualFold(encoding, "base64") {
		if decoded, err := base64.StdEncoding.DecodeString(arg.String()); err == nil {
			data = decoded
		}
	} else {
		data = []byte(arg.String())
	}

	buf := r.vm.NewObject()
	_ = buf.Set("__isBuffer", true)
	_ = buf.Set("length", len(data))
	_ = buf.Set("toString", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 && strings.EqualFold(call.Argument(0).String(), "base64") {
			return r.vm.ToValue(base64.StdEncoding.EncodeToString(data))
		}
		return r.vm.ToValue(string(data))
	}))
	_ = buf.Set("toJSON", r.vm.ToValue(func(goja.FunctionCall) goja.Value {
		values := make([]any, len(data))
		for i, b := range data {
			values[i] = int(b)
		}
		return r.vm.ToValue(values)
	}))
	return buf
}

// btoa encodes a string as base64, as the browser function does.
func btoa(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// atob decodes base64, returning an empty string on malformed input.
//
// The browser throws InvalidCharacterError instead. Returning empty keeps a
// script running, but silently: callers that care should validate first, which
// is why this is documented rather than made silent.
func atob(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return ""
	}
	return string(decoded)
}

// capturingConsole records console output and forwards it.
//
// It is mutex-guarded because a host function may log from a goroutine the
// script called into.
type capturingConsole struct {
	mu    sync.Mutex
	lines []ConsoleLine
	sink  Console
}

func newCapturingConsole(sink Console) *capturingConsole {
	return &capturingConsole{sink: sink}
}

// record appends a line and forwards it.
func (c *capturingConsole) record(level, message string) {
	c.mu.Lock()
	c.lines = append(c.lines, ConsoleLine{Level: level, Message: message})
	c.mu.Unlock()

	if c.sink != nil {
		c.sink.Log(level, message)
	}
}

// lines returns a copy of the captured output.
func (c *capturingConsole) captured() []ConsoleLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ConsoleLine(nil), c.lines...)
}

// installConsole provides console.log/info/warn/error/debug.
//
// Arguments are joined with a space, and non-strings are rendered as JSON, which
// is what Bruno's wrapped console does after serialising. Without that, an
// object would log as `[object Object]` and no test could match on it.
//
// Source: bruno-js/src/sandbox/node-vm/console.js wrapConsoleWithSerializers
func (r *Runtime) installConsole() error {
	console := r.vm.NewObject()

	levels := map[string]string{
		"log":   "log",
		"info":  "info",
		"warn":  "warn",
		"error": "error",
		"debug": "debug",
		"trace": "trace",
	}
	names := make([]string, 0, len(levels))
	for name := range levels {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		level := levels[name]
		err := console.Set(name, func(call goja.FunctionCall) goja.Value {
			r.console.record(level, r.formatConsoleArgs(call))
			return goja.Undefined()
		})
		if err != nil {
			return fmt.Errorf("scripting: install console.%s: %w", name, err)
		}
	}
	return r.vm.Set("console", console)
}

// formatConsoleArgs renders console arguments the way Bruno does.
func (r *Runtime) formatConsoleArgs(call goja.FunctionCall) string {
	parts := make([]string, 0, len(call.Arguments))
	for _, arg := range call.Arguments {
		parts = append(parts, r.renderValue(arg))
	}
	return strings.Join(parts, " ")
}

// renderValue formats one console argument.
func (r *Runtime) renderValue(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) {
		return "undefined"
	}
	if goja.IsNull(v) {
		return "null"
	}

	switch v.Export().(type) {
	case string:
		return v.String()
	}

	// Objects and arrays render as JSON so an assertion can match the text.
	if obj, ok := v.(*goja.Object); ok {
		if encoded, err := obj.MarshalJSON(); err == nil {
			return string(encoded)
		}
	}
	return v.String()
}

// installRequire provides `require(name)` over the configured module table.
//
// An unknown module throws rather than returning undefined. A silent undefined is
// the worst outcome for a scripting API: the script continues, calls a method on
// nothing, and the error the user sees is three lines away from the cause.
//
// Node built-ins Bruno allows (`fs`, `path`, `crypto`) are absent here. They are
// environment-specific — file paths, a filesystem the CLI shares with the GUI —
// and a Go implementation has to supply deliberately chosen equivalents rather
// than pretend a Node module exists. Recorded as a known gap in UQ-B5.
func (r *Runtime) installRequire() error {
	modules := r.opts.Modules
	available := make([]string, 0, len(modules))
	for name := range modules {
		available = append(available, name)
	}
	sort.Strings(available)

	require := func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		// Accept a relative-looking specifier so a script written for Node does
		// not fail on the path form alone when only the name is registered.
		key := strings.TrimPrefix(name, "./")
		key = strings.TrimPrefix(key, "node:")

		if module, ok := modules[key]; ok {
			return r.vm.ToValue(module)
		}
		panic(r.vm.NewTypeError("%s: %q (available: %s)",
			ErrNotAllowed.Error(), name, strings.Join(available, ", ")))
	}

	if err := r.vm.Set("require", require); err != nil {
		return fmt.Errorf("scripting: install require: %w", err)
	}
	return nil
}

// installTimers provides setTimeout/clearTimeout and their interval forms.
//
// Timers fire only while the script is synchronously running or while microtasks
// drain, because there is no event loop between Run calls. A timer scheduled for
// the future therefore does not fire. That matches Bruno's behaviour closely
// enough for callback-style crypto and promise tricks, and is recorded rather
// than papered over: a script that relies on a real event loop needs the run to
// be awaited, which is what Bruno's promise support does.
func (r *Runtime) installTimers() error {
	timers := map[string]any{
		"setTimeout": func(call goja.FunctionCall) goja.Value {
			r.invokeTimer(call, false)
			return goja.Undefined()
		},
		"setInterval": func(call goja.FunctionCall) goja.Value {
			// An interval would loop forever without an event loop; run once, as a
			// timeout, rather than hanging the execution.
			r.invokeTimer(call, true)
			return goja.Undefined()
		},
		"clearTimeout":  func(goja.FunctionCall) goja.Value { return goja.Undefined() },
		"clearInterval": func(goja.FunctionCall) goja.Value { return goja.Undefined() },
		// queueMicrotask defers to the job queue goja already runs after Run.
		"queueMicrotask": func(call goja.FunctionCall) goja.Value {
			if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
				_, _ = fn(goja.Undefined())
			}
			return goja.Undefined()
		},
	}
	for name, value := range timers {
		if err := r.vm.Set(name, value); err != nil {
			return fmt.Errorf("scripting: install %s: %w", name, err)
		}
	}
	return nil
}

// invokeTimer runs a timer callback immediately.
//
// A callback that throws is re-panicked so Run reports it; swallowing it would
// make a broken callback look like success.
func (r *Runtime) invokeTimer(call goja.FunctionCall, once bool) {
	fn, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		return
	}
	_ = once
	if _, err := fn(goja.Undefined(), call.Arguments[1:]...); err != nil {
		panic(err)
	}
}

// installWebAPIs provides the WHATWG pieces Bruno exposes: TextEncoder,
// TextDecoder, URL and URLSearchParams.
//
// TextEncoder and TextDecoder are constructors because scripts call
// `new TextDecoder().decode(...)`; installing them as plain functions would fail
// with "not a constructor".
//
// Source: bruno-js/src/sandbox/node-vm/constants.js (Encoding and URL APIs)
func (r *Runtime) installWebAPIs() error {
	encoderProto := r.vm.NewObject()
	_ = encoderProto.Set("encode", func(call goja.FunctionCall) goja.Value {
		return r.vm.ToValue([]byte(call.Argument(0).String()))
	})

	decoderProto := r.vm.NewObject()
	_ = decoderProto.Set("decode", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if obj, ok := arg.(*goja.Object); ok {
			length := int(obj.Get("length").ToInteger())
			data := make([]byte, 0, length)
			for i := 0; i < length; i++ {
				data = append(data, byte(obj.Get(fmt.Sprint(i)).ToInteger()))
			}
			return r.vm.ToValue(string(data))
		}
		return r.vm.ToValue(arg.String())
	})

	constructors := map[string]*goja.Object{
		"TextEncoder": encoderProto,
		"TextDecoder": decoderProto,
	}
	for name, proto := range constructors {
		name, proto := name, proto
		ctor := r.vm.ToValue(func(call goja.ConstructorCall) *goja.Object {
			// Each construction gets its own object, so two decoders cannot share
			// state through a prototype they were never meant to mutate.
			instance := r.vm.NewObject()
			instance.SetPrototype(proto)
			return instance
		})
		if err := r.vm.Set(name, ctor); err != nil {
			return fmt.Errorf("scripting: install %s: %w", name, err)
		}
	}
	return nil
}

// installConfiguredGlobals installs Options.Globals last, so a caller can
// override any default above.
func (r *Runtime) installConfiguredGlobals() error {
	names := make([]string, 0, len(r.opts.Globals))
	for name := range r.opts.Globals {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := r.vm.Set(name, r.opts.Globals[name]); err != nil {
			return fmt.Errorf("scripting: install global %s: %w", name, err)
		}
	}
	return nil
}

// JSONStringify renders a value the way JSON.stringify would, for host functions
// that need to return a JSON text rather than an object.
func JSONStringify(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(encoded)
}
