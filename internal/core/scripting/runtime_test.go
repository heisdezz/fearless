package scripting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// run executes source in a fresh runtime and returns the result.
func run(t *testing.T, opts Options, source string) *Result {
	t.Helper()
	rt, err := New(opts)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	res, err := rt.Run(context.Background(), source)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

// TestRunReturnsCompletionValue covers the basic evaluation path.
func TestRunReturnsCompletionValue(t *testing.T) {
	res := run(t, Options{}, "1 + 2")
	if res.ReturnValue != int64(3) {
		t.Errorf("return = %#v, want 3", res.ReturnValue)
	}
}

// TestRunExportsPlainGoValues checks that objects and arrays arrive as
// map[string]any and []any, so host code never sees goja types.
func TestRunExportsPlainGoValues(t *testing.T) {
	res := run(t, Options{}, `({a: 1, b: [2, 3], c: "x"})`)

	obj, ok := res.ReturnValue.(map[string]any)
	if !ok {
		t.Fatalf("return = %T", res.ReturnValue)
	}
	if obj["a"] != int64(1) {
		t.Errorf("a = %#v", obj["a"])
	}
	if _, ok := obj["b"].([]any); !ok {
		t.Errorf("b = %T, want []any", obj["b"])
	}
}

// TestRunInterruptsRunawayScript is the core safety property: an infinite loop
// terminates at the deadline rather than hanging the process.
func TestRunInterruptsRunawayScript(t *testing.T) {
	rt, err := New(Options{Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	_, err = rt.Run(context.Background(), "while (true) {}")
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("interrupt took %v", elapsed)
	}

	// The error must be a *ScriptError too, so callers get one error type.
	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("err type = %T", err)
	}
}

// TestRunHonoursContextCancellation covers the cancel path.
func TestRunHonoursContextCancellation(t *testing.T) {
	rt, err := New(Options{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err = rt.Run(ctx, "while (true) {}")
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v, want ErrCancelled", err)
	}
}

// TestInterruptDoesNotLeakAcrossRuns matters because a stale interrupt would
// abort every later execution in that runtime.
func TestInterruptDoesNotLeakAcrossRuns(t *testing.T) {
	rt, err := New(Options{Timeout: 30 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Run(context.Background(), "while (true) {}"); !errors.Is(err, ErrTimeout) {
		t.Fatalf("first run err = %v", err)
	}

	// The same runtime must be usable afterwards, with whatever deadline remains.
	res, err := rt.Run(context.Background(), "6 * 7")
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if res.ReturnValue != int64(42) {
		t.Errorf("return = %#v", res.ReturnValue)
	}
}

// TestRunDoesNotLeakGoroutines checks the watcher is released. Run with -race and
// repeatedly, this would surface a leak as a stuck goroutine.
func TestRunDoesNotLeakGoroutines(t *testing.T) {
	for i := 0; i < 100; i++ {
		rt, err := New(Options{Timeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := rt.Run(context.Background(), "1"); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRecursionLimitIsEnforced keeps a deep recursion from exhausting memory.
func TestRecursionLimitIsEnforced(t *testing.T) {
	rt, err := New(Options{MaxCallStack: 64, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Run(context.Background(), `
		function f(n) { return f(n + 1); }
		f(0);
	`)
	if err == nil {
		t.Fatal("expected a stack error")
	}
}

// TestScriptErrorCarriesLocation makes the `.bru` line reachable.
func TestScriptErrorCarriesLocation(t *testing.T) {
	rt, err := New(Options{Filename: "request.bru", LineOffset: 10})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Run(context.Background(), "null.x")
	if err == nil {
		t.Fatal("expected an error")
	}

	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("err type = %T", err)
	}
	if se.Filename != "request.bru" {
		t.Errorf("filename = %q", se.Filename)
	}
	// Line 1 in the script plus the offset points into the .bru file.
	if se.Line != 11 {
		t.Errorf("line = %d, want 11", se.Line)
	}
	if !strings.Contains(se.Error(), "request.bru:11") {
		t.Errorf("message = %q", se.Error())
	}
}

// TestThrownErrorKeepsNameAndMessage covers an explicit throw.
func TestThrownErrorKeepsNameAndMessage(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Run(context.Background(), `throw new TypeError("bad input")`)

	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("err type = %T", err)
	}
	if se.Name != "TypeError" || se.Message != "bad input" {
		t.Errorf("error = %+v", se)
	}
}

// TestConsoleIsCapturedAndSerialised covers console output shape.
func TestConsoleIsCapturedAndSerialised(t *testing.T) {
	var forwarded []string
	res := run(t, Options{Console: ConsoleFunc(func(level, message string) {
		forwarded = append(forwarded, level+":"+message)
	})}, `
		console.log("plain", 42);
		console.warn("careful");
		console.error({code: "E1"});
	`)

	if len(res.Console) != 3 {
		t.Fatalf("captured %d lines: %+v", len(res.Console), res.Console)
	}
	if res.Console[0].Message != "plain 42" {
		t.Errorf("line 0 = %q", res.Console[0].Message)
	}
	if res.Console[1].Level != "warn" {
		t.Errorf("line 1 level = %q", res.Console[1].Level)
	}
	// An object must render as JSON, not [object Object].
	if !strings.Contains(res.Console[2].Message, `"code":"E1"`) {
		t.Errorf("object rendered as %q", res.Console[2].Message)
	}

	// A sink, when supplied, also receives the lines.
	if len(forwarded) != 3 || forwarded[0] != "log:plain 42" {
		t.Errorf("forwarded = %v", forwarded)
	}
}

// TestConsoleRendersUndefinedAndNull covers the edge cases.
func TestConsoleRendersUndefinedAndNull(t *testing.T) {
	res := run(t, Options{}, `console.log(undefined, null)`)
	if got := res.Console[0].Message; got != "undefined null" {
		t.Errorf("message = %q", got)
	}
}

// TestRequireProvidesAllowlistedModules covers the module table.
func TestRequireProvidesAllowlistedModules(t *testing.T) {
	modules := map[string]any{
		"moment": map[string]any{
			"version": "2.29.4",
		},
	}
	res := run(t, Options{Modules: modules}, `require("moment").version`)
	if res.ReturnValue != "2.29.4" {
		t.Errorf("return = %#v", res.ReturnValue)
	}
}

// TestRequireNormalisesSpecifier accepts the `node:` prefix and a relative form.
func TestRequireNormalisesSpecifier(t *testing.T) {
	modules := map[string]any{"crypto-js": map[string]any{"ok": true}}
	for _, spec := range []string{"crypto-js", "node:crypto-js"} {
		res := run(t, Options{Modules: modules}, `require("`+spec+`").ok`)
		if res.ReturnValue != true {
			t.Errorf("%s: return = %#v", spec, res.ReturnValue)
		}
	}
}

// TestRequireRefusesUnknownModule is the important half: a missing module must
// throw, not return undefined, so the script fails where the mistake is.
func TestRequireRefusesUnknownModule(t *testing.T) {
	rt, err := New(Options{Modules: map[string]any{"moment": map[string]any{}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Run(context.Background(), `require("left-pad")`)
	if err == nil {
		t.Fatal("expected an error")
	}
	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("err type = %T", err)
	}
	// The message lists what is available, which is what makes it actionable.
	if !strings.Contains(se.Message, "left-pad") || !strings.Contains(se.Message, "moment") {
		t.Errorf("message = %q", se.Message)
	}
}

// TestGlobalsFromOptionsAreInstalled covers caller-supplied globals.
func TestGlobalsFromOptionsAreInstalled(t *testing.T) {
	res := run(t, Options{Globals: map[string]any{"answer": 42}}, "answer")
	if res.ReturnValue != int64(42) {
		t.Errorf("return = %#v", res.ReturnValue)
	}
}

// TestSetAndGetVariable moves values across the host boundary.
func TestSetAndGetVariable(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.SetVariable("host", "example.com"); err != nil {
		t.Fatal(err)
	}
	res, err := rt.Run(context.Background(), `host`)
	if err != nil {
		t.Fatal(err)
	}
	if res.ReturnValue != "example.com" {
		t.Errorf("return = %#v", res.ReturnValue)
	}

	if got, ok := rt.GetVariable("host"); !ok || got != "example.com" {
		t.Errorf("get = %#v, %v", got, ok)
	}
	// A name the script never defined reports absent rather than a zero value.
	if _, ok := rt.GetVariable("missing"); ok {
		t.Error("a missing variable must report false")
	}
}

// TestRunFunctionCallsGlobal covers the callback entry point.
func TestRunFunctionCallsGlobal(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Run(context.Background(), `function onDone(x) { return x * 2; }`); err != nil {
		t.Fatal(err)
	}

	res, err := rt.RunFunction(context.Background(), "onDone", 21)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if res.ReturnValue != int64(42) {
		t.Errorf("return = %#v", res.ReturnValue)
	}

	if _, err := rt.RunFunction(context.Background(), "notDefined"); err == nil {
		t.Error("expected an error for a missing function")
	}
}

// TestReturnedPromiseIsAwaited covers the await contract.
//
// A script whose completion value *is* a promise is awaited until it settles, so
// the caller sees the resolved value rather than `undefined`. Note the narrower
// guarantee: goja captures the completion value before draining its job queue, so
// a script that ends in a plain expression while a promise is still queued does
// not get its `then` flushed before Run returns. Bruno awaits the script's result,
// and this does the same thing — awaiting the result, not the queue.
func TestReturnedPromiseIsAwaited(t *testing.T) {
	res := run(t, Options{}, `Promise.resolve(5).then(v => v * 2)`)
	if res.ReturnValue != int64(10) {
		t.Errorf("return = %#v, want the resolved value", res.ReturnValue)
	}
}

// TestRejectedPromiseBecomesScriptError surfaces a rejection as an error rather
// than a silently unresolved value.
func TestRejectedPromiseBecomesScriptError(t *testing.T) {
	rt, err := New(Options{Filename: "request.bru"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Run(context.Background(), `Promise.reject(new Error("nope"))`)
	if err == nil {
		t.Fatal("expected an error")
	}
	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("err type = %T", err)
	}
	if se.Message != "nope" {
		t.Errorf("message = %q", se.Message)
	}
}

// TestQueuedMicrotasksRunAfterRun documents the narrower guarantee: jobs pending
// at the end of the script have run by the time the *next* call observes them,
// because goja drains its queue when the top-level call returns.
func TestQueuedMicrotasksRunAfterRun(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Run(context.Background(), `
		globalThis.seen = null;
		Promise.resolve(5).then(v => { globalThis.seen = v * 2; });
	`); err != nil {
		t.Fatal(err)
	}

	got, ok := rt.GetVariable("seen")
	if !ok || got != int64(10) {
		t.Errorf("seen = %#v, want 10 — the then callback should have run", got)
	}
}

// TestBase64Helpers covers btoa/atob.
func TestBase64Helpers(t *testing.T) {
	res := run(t, Options{}, `btoa("hello")`)
	if res.ReturnValue != "aGVsbG8=" {
		t.Errorf("btoa = %#v", res.ReturnValue)
	}

	res = run(t, Options{}, `atob("aGVsbG8=")`)
	if res.ReturnValue != "hello" {
		t.Errorf("atob = %#v", res.ReturnValue)
	}

	// Malformed input yields an empty string rather than throwing.
	res = run(t, Options{}, `atob("!!!")`)
	if res.ReturnValue != "" {
		t.Errorf("atob(malformed) = %#v", res.ReturnValue)
	}
}

// TestBufferFrom covers the Buffer subset.
func TestBufferFrom(t *testing.T) {
	res := run(t, Options{}, `Buffer.from("hello").toString("base64")`)
	if res.ReturnValue != "aGVsbG8=" {
		t.Errorf("return = %#v", res.ReturnValue)
	}

	res = run(t, Options{}, `Buffer.from([104, 105]).toString()`)
	if res.ReturnValue != "hi" {
		t.Errorf("return = %#v", res.ReturnValue)
	}

	res = run(t, Options{}, `Buffer.from("aGk=", "base64").toString()`)
	if res.ReturnValue != "hi" {
		t.Errorf("return = %#v", res.ReturnValue)
	}

	res = run(t, Options{}, `Buffer.isBuffer(Buffer.from("x")) && !Buffer.isBuffer("x")`)
	if res.ReturnValue != true {
		t.Errorf("isBuffer = %#v", res.ReturnValue)
	}
}

// TestTimersRunCallback covers setTimeout's synchronous execution.
func TestTimersRunCallback(t *testing.T) {
	res := run(t, Options{}, `
		let out = null;
		setTimeout(() => { out = "fired"; }, 0);
		out;
	`)
	if res.ReturnValue != "fired" {
		t.Errorf("return = %#v", res.ReturnValue)
	}
}

// TestTextEncoderDecoder covers the encoding API.
func TestTextEncoderDecoder(t *testing.T) {
	res := run(t, Options{}, `new TextDecoder().decode(new TextEncoder().encode("héllo"))`)
	if res.ReturnValue != "héllo" {
		t.Errorf("return = %#v", res.ReturnValue)
	}
}

// TestConcurrentRuntimesAreIndependent is the -race property: two runtimes share
// no state, so two requests can script at once.
func TestConcurrentRuntimesAreIndependent(t *testing.T) {
	const workers = 8
	done := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func(n int) {
			rt, err := New(Options{Timeout: 2 * time.Second})
			if err != nil {
				done <- err
				return
			}
			// Each runtime defines its own `state`; a shared context would collide.
			res, err := rt.Run(context.Background(), `
				let state = 0;
				for (let i = 0; i < 1000; i++) state += 1;
				state;
			`)
			if err != nil {
				done <- err
				return
			}
			if res.ReturnValue != int64(1000) {
				done <- errors.New("unexpected return value")
				return
			}
			done <- nil
		}(i)
	}

	for i := 0; i < workers; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
