package scripting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// TestBruGetAllScopes covers the enumeration helpers, which share copyMap.
func TestBruGetAllScopes(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Environment: map[string]any{"env": "e"},
		Global:      map[string]any{"glob": "g"},
		Runtime:     map[string]any{"run": "r"},
		Collection:  map[string]any{"coll": "c"},
	}})

	cases := map[string]string{
		"bru.getAllEnvVars()":        "env",
		"bru.getAllGlobalEnvVars()":  "glob",
		"bru.getAllVars()":           "run",
		"bru.getAllCollectionVars()": "coll",
	}
	for script, key := range cases {
		res := h.run(t, script+`["`+key+`"]`)
		if res.ReturnValue == nil {
			t.Errorf("%s: missing %q", script, key)
		}
	}

	// getAll returns a copy, so mutating it must not touch the scope.
	h.run(t, `bru.getAllVars()["injected"] = "x"`)
	if _, ok := h.state.Scopes.Runtime["injected"]; ok {
		t.Error("getAllVars returned the live map")
	}
}

// TestBruDeleteAllScopes clears each mutable scope.
func TestBruDeleteAllScopes(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Environment: map[string]any{"a": 1},
		Global:      map[string]any{"a": 1},
		Runtime:     map[string]any{"a": 1},
		Collection:  map[string]any{"a": 1},
	}})

	h.run(t, `
		bru.deleteAllEnvVars();
		bru.deleteAllGlobalEnvVars();
		bru.deleteAllVars();
		bru.deleteAllCollectionVars();
	`)

	for name, scope := range map[string]map[string]any{
		"environment": h.state.Scopes.Environment,
		"global":      h.state.Scopes.Global,
		"runtime":     h.state.Scopes.Runtime,
		"collection":  h.state.Scopes.Collection,
	} {
		if len(scope) != 0 {
			t.Errorf("%s scope = %v, want empty", name, scope)
		}
	}
}

// TestBruDeleteAllBlockedInSafeMode covers the shared guard.
func TestBruDeleteAllBlockedInSafeMode(t *testing.T) {
	h := newHarness(t, &State{IsSafeMode: true, Scopes: &vars.Scopes{
		Runtime: map[string]any{"a": 1},
	}})
	if err := h.runErr(`bru.deleteAllVars()`); err == nil {
		t.Fatal("expected an error in safe mode")
	}
}

// TestBruFolderAndRequestVarsAreReadOnly covers the two read scopes.
func TestBruFolderAndRequestVarsAreReadOnly(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Folder:  map[string]any{"f": "folder-value"},
		Request: map[string]any{"r": "request-value"},
	}})

	if res := h.run(t, `bru.getFolderVar("f")`); res.ReturnValue != "folder-value" {
		t.Errorf("folder var = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.getRequestVar("r")`); res.ReturnValue != "request-value" {
		t.Errorf("request var = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.getFolderVar("missing")`); res.ReturnValue != nil {
		t.Errorf("missing = %#v", res.ReturnValue)
	}
}

// TestBruTopLevelSetNextRequest covers the alias on bru itself.
func TestBruTopLevelSetNextRequest(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `bru.setNextRequest("step 2")`)
	if h.control.NextRequest != "step 2" {
		t.Errorf("next request = %q", h.control.NextRequest)
	}
}

// TestBruStopExecution covers the runner's stop directive.
func TestBruStopExecution(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `bru.runner.stopExecution()`)
	if !h.control.StopExecution {
		t.Error("stopExecution not recorded")
	}
}

// TestBruSetVarRequiresName keeps an empty key from entering the scope.
func TestBruSetVarRequiresName(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.runErr(`bru.setVar("", 1)`); err == nil {
		t.Fatal("expected an error for an empty name")
	}
}

// TestBruNameWithLowerPrecedenceValueIsShadowed covers a scope reading its own
// entry before falling back to the merged view.
func TestBruNameWithLowerPrecedenceValueIsShadowed(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Environment: map[string]any{"host": "env"},
	}})

	h.run(t, `bru.setVar("host", "runtime")`)
	if res := h.run(t, `bru.getVar("host")`); res.ReturnValue != "runtime" {
		t.Errorf("runtime host = %#v", res.ReturnValue)
	}
	// The environment still holds its own value.
	if res := h.run(t, `bru.getEnvVar("host")`); res.ReturnValue != "env" {
		t.Errorf("env host = %#v", res.ReturnValue)
	}
}

// TestMinifyXMLEdgeCases covers the input shapes the minifier must tolerate.
func TestMinifyXMLEdgeCases(t *testing.T) {
	h := newHarness(t, nil)

	cases := []struct {
		name   string
		script string
		want   string
	}{
		{"null input", `bru.utils.minifyXml(null)`, ""},
		{"undefined input", `bru.utils.minifyXml(undefined)`, ""},
		{"already minimal", `bru.utils.minifyXml("<a><b/></a>")`, "<a><b/></a>"},
		{"text whitespace preserved", `bru.utils.minifyXml("<a>hello world</a>")`, "<a>hello world</a>"},
		{"multiple gaps", `bru.utils.minifyXml("<a>  <b/>  \n  <c/>  </a>")`, "<a><b/><c/></a>"},
	}
	for _, tc := range cases {
		res := h.run(t, tc.script)
		if got, _ := res.ReturnValue.(string); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestIsTypeVariants covers the type table used by `to.be.a`.
func TestIsTypeVariants(t *testing.T) {
	h := newHarness(t, nil)

	passing := []string{
		`expect("s").to.be.a("string")`,
		`expect(1).to.be.a("number")`,
		`expect(1.5).to.be.a("number")`,
		`expect(true).to.be.a("boolean")`,
		`expect([1]).to.be.an("array")`,
		`expect({}).to.be.an("object")`,
		`expect(null).to.be.a("null")`,
		`expect(undefined).to.be.a("undefined")`,
		`expect((() => 1)).to.be.a("function")`,
	}
	for _, script := range passing {
		if err := h.runErr(script); err != nil {
			t.Errorf("%s: %v", script, err)
		}
	}

	// An unknown type name never matches, rather than matching everything.
	if err := h.runErr(`expect(1).to.be.a("widget")`); err == nil {
		t.Error("an unknown type name must not pass")
	}
}

// TestValueLengthVariants covers strings, arrays and non-collections.
func TestValueLengthVariants(t *testing.T) {
	h := newHarness(t, nil)

	if err := h.runErr(`expect("abc").to.have.lengthOf(3)`); err != nil {
		t.Errorf("string length: %v", err)
	}
	if err := h.runErr(`expect([1,2,3]).to.have.lengthOf(3)`); err != nil {
		t.Errorf("array length: %v", err)
	}
	// A number has no length, so length 0 is the honest answer.
	if err := h.runErr(`expect(7).to.have.lengthOf(0)`); err != nil {
		t.Errorf("number length: %v", err)
	}
	if err := h.runErr(`expect(7).to.have.lengthOf(3)`); err == nil {
		t.Error("expected a failure")
	}
}

// TestExpectNegationChains covers `.not` with the various assertion families.
func TestExpectNegationChains(t *testing.T) {
	h := newHarness(t, nil)

	passing := []string{
		`expect(5).not.to.equal(6)`,
		`expect(5).to.not.be.above(10)`,
		`expect("abc").to.not.include("zzz")`,
		`expect([1]).to.not.include(9)`,
		`expect(true).to.not.be.false()`,
	}
	for _, script := range passing {
		if err := h.runErr(script); err != nil {
			t.Errorf("%s: %v", script, err)
		}
	}
}

// TestOneOfAndMatchVariants covers the two array/regex assertions.
func TestOneOfAndMatchVariants(t *testing.T) {
	h := newHarness(t, nil)

	if err := h.runErr(`expect("b").to.be.oneOf(["a", "b"])`); err != nil {
		t.Errorf("oneOf: %v", err)
	}
	if err := h.runErr(`expect("z").to.be.oneOf(["a", "b"])`); err == nil {
		t.Error("oneOf should fail for a missing value")
	}
	// A non-array argument cannot contain anything.
	if err := h.runErr(`expect("a").to.be.oneOf("a")`); err == nil {
		t.Error("oneOf with a non-array must fail")
	}

	if err := h.runErr(`expect("v1.2.3").to.match(/^v\d+/)`); err != nil {
		t.Errorf("match: %v", err)
	}
	if err := h.runErr(`expect("abc").to.match(/xyz/)`); err == nil {
		t.Error("a non-matching pattern must fail")
	}
	// An invalid pattern must fail rather than pass.
	if err := h.runErr(`expect("abc").to.match("(")`); err == nil {
		t.Error("an invalid pattern must fail")
	}
}

// TestScriptErrorMessageFormat covers every ScriptError rendering branch.
func TestScriptErrorMessageFormat(t *testing.T) {
	cases := []struct {
		err  *ScriptError
		want string
	}{
		{&ScriptError{Filename: "f.bru", Line: 3, Name: "TypeError", Message: "bad"}, "f.bru:3: TypeError: bad"},
		{&ScriptError{Filename: "f.bru", Name: "TypeError", Message: "bad"}, "f.bru: TypeError: bad"},
		{&ScriptError{Name: "Error", Message: "bad"}, "Error: bad"},
		// Without a filename, the line is omitted rather than rendered as a bare
		// ":1", which would read as a corruption.
		{&ScriptError{Line: 1, Message: "bad"}, "Error: bad"},
	}
	for _, tc := range cases {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}

	// Unwrap exposes the cause for errors.Is.
	wrapped := &ScriptError{Cause: ErrTimeout}
	if !errors.Is(wrapped, ErrTimeout) {
		t.Error("Unwrap did not expose the cause")
	}
}

// TestSendRequestWithoutHandlerDoesNotPanic covers the callback-less form.
func TestSendRequestWithoutHandlerDoesNotPanic(t *testing.T) {
	h := newHarness(t, nil)
	// The rejection is unhandled, which goja reports as a failure.
	err := h.runErr(`bru.sendRequest("https://x/")`)
	if err == nil {
		t.Fatal("expected the rejection to surface")
	}
	if calls := h.binding.ScriptedCalls(); len(calls) != 1 || calls[0].Source != "sendRequest" {
		t.Errorf("calls = %+v", calls)
	}
}

// TestRunRequestSourceIsRecorded distinguishes the two nested-request APIs.
func TestRunRequestSourceIsRecorded(t *testing.T) {
	h := newHarness(t, nil)
	_ = h.runErr(`bru.runRequest("https://y/")`)
	calls := h.binding.ScriptedCalls()
	if len(calls) != 1 || calls[0].Source != "runRequest" {
		t.Errorf("calls = %+v", calls)
	}
}

// TestReqOnFailStoresHandler covers the failure-rewrite hook.
func TestReqOnFailStoresHandler(t *testing.T) {
	h := newHarness(t, &State{Request: &model.Request{Method: "GET"}})
	h.run(t, `req.onFail((err) => { return "handled" })`)
	if h.state.failHandler == nil {
		t.Error("onFail handler not stored")
	}
}

// TestInstallRequiresState keeps a nil state from being dereferenced.
func TestInstallRequiresState(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(rt, nil, nil, nil); err == nil {
		t.Fatal("expected an error for a nil state")
	}
}

// TestInstallDefaultsControlAndCollector covers the nil arguments.
func TestInstallDefaultsControlAndCollector(t *testing.T) {
	rt, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Install(rt, &State{Scopes: &vars.Scopes{}}, nil, nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if b.control == nil || b.tests == nil {
		t.Fatal("Install must supply defaults")
	}
	// And the defaulted objects are live.
	if _, err := rt.Run(context.Background(), `test("x", () => {}); bru.runner.skipRequest()`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(b.tests.Results()) != 1 || !b.control.SkipRequest {
		t.Errorf("defaults not wired: %+v %+v", b.tests.Results(), b.control)
	}
}

// TestGetterWithoutRequestReturnsEmpty covers the nil-request defaults.
func TestGetterWithoutRequestReturnsEmpty(t *testing.T) {
	h := newHarness(t, nil)

	if res := h.run(t, `req.getName()`); res.ReturnValue != "" {
		t.Errorf("name = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getAuthMode()`); res.ReturnValue != "none" {
		t.Errorf("auth mode = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getUrl()`); res.ReturnValue != "" {
		t.Errorf("url = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.hasJSONContentType()`); res.ReturnValue != false {
		t.Errorf("json content type = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getTags().length`); res.ReturnValue != int64(0) {
		t.Errorf("tags = %#v", res.ReturnValue)
	}
}

// TestReqAuthModeReadsConfig covers the auth accessor with a real config.
func TestReqAuthModeReadsConfig(t *testing.T) {
	req := &model.Request{Method: "GET", Auth: &model.Auth{Mode: model.AuthBearer}}
	h := newHarness(t, &State{Request: req})

	if res := h.run(t, `req.getAuthMode()`); res.ReturnValue != "bearer" {
		t.Errorf("auth mode = %#v", res.ReturnValue)
	}
}

// TestResSetBodyRewrite covers res.setBody.
func TestResSetBodyRewrite(t *testing.T) {
	result := &model.Result{Status: 200}
	h := newHarness(t, &State{Result: result, ResponseText: "original"})

	h.run(t, `res.setBody("rewritten")`)
	if h.state.ResponseText != "rewritten" {
		t.Errorf("body = %q", h.state.ResponseText)
	}
	// Without a result, setBody is a no-op rather than an error.
	other := newHarness(t, nil)
	if err := other.runErr(`res.setBody("x")`); err != nil {
		t.Errorf("err = %v", err)
	}
}

// TestResTimeAndURL covers the remaining response accessors.
func TestResTimeAndURL(t *testing.T) {
	h := newHarness(t, &State{
		Result:     &model.Result{Duration: 150 * time.Millisecond},
		RequestURL: "https://api.example.com/x",
	})
	if res := h.run(t, `res.getResponseTime()`); res.ReturnValue != int64(150) {
		t.Errorf("response time = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getUrl()`); res.ReturnValue != "https://api.example.com/x" {
		t.Errorf("url = %#v", res.ReturnValue)
	}
}

// TestTestCaseWithoutMessage covers the message fallback branches.
func TestTestCaseWithoutMessage(t *testing.T) {
	h := newHarness(t, nil)
	// A thrown non-error value still produces a usable message.
	h.run(t, `test("throws a string", () => { throw "plain"; })`)

	results := h.tests.Results()
	if len(results) != 1 || results[0].Status != "fail" {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(results[0].Error, "plain") {
		t.Errorf("error = %q", results[0].Error)
	}
}
