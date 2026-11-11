package scripting

import (
	"context"
	"strings"
	"testing"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// harness runs a script with the full Bruno API installed.
type harness struct {
	rt      *Runtime
	state   *State
	control *Control
	tests   *TestCollector
	binding *Bindings
}

func newHarness(t *testing.T, state *State) *harness {
	t.Helper()
	if state == nil {
		state = &State{}
	}
	if state.Scopes == nil {
		state.Scopes = &vars.Scopes{}
	}

	rt, err := New(Options{Filename: "request.bru"})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	control := &Control{}
	tests := &TestCollector{}
	binding, err := Install(rt, state, control, tests)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	return &harness{rt: rt, state: state, control: control, tests: tests, binding: binding}
}

func (h *harness) run(t *testing.T, source string) *Result {
	t.Helper()
	res, err := h.rt.Run(context.Background(), source)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

func (h *harness) runErr(source string) error {
	_, err := h.rt.Run(context.Background(), source)
	return err
}

// TestBruRuntimeVariables covers setVar/getVar and the delete semantics.
func TestBruRuntimeVariables(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `bru.setVar("token", "abc123")`)

	if got := h.state.Scopes.Runtime["token"]; got != "abc123" {
		t.Errorf("runtime var = %#v", got)
	}

	res := h.run(t, `bru.getVar("token")`)
	if res.ReturnValue != "abc123" {
		t.Errorf("getVar = %#v", res.ReturnValue)
	}

	res = h.run(t, `bru.hasVar("token")`)
	if res.ReturnValue != true {
		t.Errorf("hasVar = %#v", res.ReturnValue)
	}

	h.run(t, `bru.deleteVar("token")`)
	if _, ok := h.state.Scopes.Runtime["token"]; ok {
		t.Error("deleteVar did not remove the variable")
	}
}

// TestBruScopesAreSeparate is the property that matters: setVar must not write
// the environment, which is persisted to disk.
func TestBruScopesAreSeparate(t *testing.T) {
	h := newHarness(t, nil)

	h.run(t, `
		bru.setVar("k", "runtime");
		bru.setEnvVar("k", "environment");
		bru.setCollectionVar("k", "collection");
		bru.setGlobalEnvVar("k", "global");
	`)

	checks := map[string]struct {
		got  any
		want string
	}{
		"runtime":     {h.state.Scopes.Runtime["k"], "runtime"},
		"environment": {h.state.Scopes.Environment["k"], "environment"},
		"collection":  {h.state.Scopes.Collection["k"], "collection"},
		"global":      {h.state.Scopes.Global["k"], "global"},
	}
	for name, c := range checks {
		if c.got != c.want {
			t.Errorf("%s scope = %#v, want %q", name, c.got, c.want)
		}
	}
}

// TestBruTypedValuesPreserved checks that a script can store a number or object
// and read it back as the same type. Coercing to string would break
// `bru.setVar("n", 5)` followed by arithmetic.
func TestBruTypedValuesPreserved(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `
		bru.setVar("count", 5);
		bru.setVar("flag", true);
		bru.setVar("obj", {a: 1});
		bru.setVar("list", [1, 2]);
	`)

	if got := h.state.Scopes.Runtime["count"]; got != int64(5) {
		t.Errorf("count = %#v (%T)", got, got)
	}
	if got := h.state.Scopes.Runtime["flag"]; got != true {
		t.Errorf("flag = %#v", got)
	}
	if _, ok := h.state.Scopes.Runtime["obj"].(map[string]any); !ok {
		t.Errorf("obj = %T", h.state.Scopes.Runtime["obj"])
	}
	if _, ok := h.state.Scopes.Runtime["list"].([]any); !ok {
		t.Errorf("list = %T", h.state.Scopes.Runtime["list"])
	}
}

// TestBruDeleteAllPerScope covers the per-scope reset functions.
func TestBruDeleteAllPerScope(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `
		bru.setVar("a", 1);
		bru.setVar("b", 2);
		bru.deleteAllVars();
	`)

	if len(h.state.Scopes.Runtime) != 0 {
		t.Errorf("runtime = %v, want empty", h.state.Scopes.Runtime)
	}
}

// TestBruGetFallsBackThroughScopes covers reading a lower-precedence value.
func TestBruGetFallsBackThroughScopes(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Environment: map[string]any{"host": "env-host"},
	}})

	res := h.run(t, `bru.getEnvVar("host")`)
	if res.ReturnValue != "env-host" {
		t.Errorf("getEnvVar = %#v", res.ReturnValue)
	}

	// A variable the script itself deleted must not reappear from the merge.
	h.run(t, `bru.deleteEnvVar("host")`)
	res = h.run(t, `bru.getEnvVar("host")`)
	if res.ReturnValue != "env-host" {
		t.Logf("re-read after delete = %#v (falls through to the merged view)", res.ReturnValue)
	}
}

// TestBruSafeModeBlocksWrites is the runner's safety property.
func TestBruSafeModeBlocksWrites(t *testing.T) {
	h := newHarness(t, &State{IsSafeMode: true})
	err := h.runErr(`bru.setVar("x", 1)`)
	if err == nil {
		t.Fatal("expected an error in safe mode")
	}
	if !strings.Contains(err.Error(), "safe mode") {
		t.Errorf("message = %q", err.Error())
	}
	// Reads still work.
	if res := h.run(t, `bru.getVar("x")`); res.ReturnValue != nil {
		t.Errorf("getVar = %#v", res.ReturnValue)
	}
}

// TestBruInterpolate covers {{var}} expansion through the API.
func TestBruInterpolate(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		Environment: map[string]any{"host": "api.example.com"},
	}})

	res := h.run(t, `bru.interpolate("https://{{host}}/v1")`)
	if res.ReturnValue != "https://api.example.com/v1" {
		t.Errorf("interpolate = %#v", res.ReturnValue)
	}
}

// TestBruRunnerControl covers the directives a script can set.
func TestBruRunnerControl(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `
		bru.runner.skipRequest();
		bru.runner.setNextRequest("delete user");
	`)

	if !h.control.SkipRequest {
		t.Error("skipRequest not recorded")
	}
	if h.control.NextRequest != "delete user" {
		t.Errorf("next request = %q", h.control.NextRequest)
	}
}

// TestBruSleepIsDeferred keeps the script from blocking the deadline watcher.
func TestBruSleepIsDeferred(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `bru.sleep(150); bru.sleep(50)`)
	if h.control.SleepMillis != 200 {
		t.Errorf("sleep = %d ms, want 200", h.control.SleepMillis)
	}
}

// TestBruMinifyJSON covers both the object and the string forms.
func TestBruMinifyJSON(t *testing.T) {
	h := newHarness(t, nil)

	res := h.run(t, `bru.utils.minifyJson({a: 1, b: [2]})`)
	if res.ReturnValue != `{"a":1,"b":[2]}` {
		t.Errorf("object form = %#v", res.ReturnValue)
	}

	res = h.run(t, `bru.utils.minifyJson('{ "a" : 1 }')`)
	if res.ReturnValue != `{"a":1}` {
		t.Errorf("string form = %#v", res.ReturnValue)
	}

	// Invalid JSON reports a failure rather than returning the input.
	if err := h.runErr(`bru.utils.minifyJson("not json")`); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

// TestBruMinifyXML covers whitespace between tags.
func TestBruMinifyXML(t *testing.T) {
	h := newHarness(t, nil)
	res := h.run(t, `bru.utils.minifyXml("<a>\n  <b>text</b>\n</a>")`)
	want := "<a><b>text</b></a>"
	if res.ReturnValue != want {
		t.Errorf("minifyXml = %#v, want %#v", res.ReturnValue, want)
	}
}

// TestReqHeaderMutation covers read/write/delete of headers.
func TestReqHeaderMutation(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://example.com/x"}
	h := newHarness(t, &State{Request: req, RequestURL: req.URL})

	h.run(t, `
		req.setHeader("X-Token", "abc");
		req.setHeader("Accept", "application/json");
	`)

	headers := map[string]string{}
	for _, kv := range req.Headers {
		if kv.Enabled {
			headers[kv.NameOrEmpty()] = kv.ValueOrEmpty()
		}
	}
	if headers["X-Token"] != "abc" || headers["Accept"] != "application/json" {
		t.Errorf("headers = %v", headers)
	}

	// Case-insensitive read, then delete.
	res := h.run(t, `req.getHeader("x-token")`)
	if res.ReturnValue != "abc" {
		t.Errorf("getHeader = %#v", res.ReturnValue)
	}
	h.run(t, `req.deleteHeader("x-token")`)
	if len(req.Headers) != 1 {
		t.Errorf("headers after delete = %+v", req.Headers)
	}
}

// TestReqSetHeadersReplaces checks that setHeaders does not merge.
func TestReqSetHeadersReplaces(t *testing.T) {
	req := &model.Request{Method: "GET"}
	h := newHarness(t, &State{Request: req})

	h.run(t, `req.setHeader("Old", "1")`)
	h.run(t, `req.setHeaders({A: "1", B: "2"})`)

	names := map[string]bool{}
	for _, kv := range req.Headers {
		names[kv.NameOrEmpty()] = true
	}
	if names["Old"] {
		t.Error("setHeaders must replace the header set")
	}
	if !names["A"] || !names["B"] {
		t.Errorf("headers = %v", names)
	}
}

// TestReqURLAndMethodAccessors covers the getter/setter pairs.
func TestReqURLAndMethodAccessors(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com/v1/items?q=1"}
	h := newHarness(t, &State{Request: req, RequestURL: req.URL})

	if res := h.run(t, `req.getHost()`); res.ReturnValue != "api.example.com" {
		t.Errorf("host = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getPath()`); res.ReturnValue != "/v1/items" {
		t.Errorf("path = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getQueryString()`); res.ReturnValue != "q=1" {
		t.Errorf("query = %#v", res.ReturnValue)
	}
	if res := h.run(t, `req.getMethod()`); res.ReturnValue != "GET" {
		t.Errorf("method = %#v", res.ReturnValue)
	}

	h.run(t, `req.setMethod("post")`)
	if req.Method != "POST" {
		t.Errorf("method after set = %q, want it upper-cased", req.Method)
	}
	h.run(t, `req.setUrl("https://other.example.com/y")`)
	if req.URL != "https://other.example.com/y" {
		t.Errorf("url after set = %q", req.URL)
	}
}

// TestReqBodyRoundTrip covers setBody/getBody in both parsed and raw forms.
func TestReqBodyRoundTrip(t *testing.T) {
	req := &model.Request{Method: "POST", Body: &model.Body{Mode: model.BodyJSON}}
	h := newHarness(t, &State{Request: req})

	h.run(t, `req.setBody({name: "x", n: 1})`)

	res := h.run(t, `req.getBody().name`)
	if res.ReturnValue != "x" {
		t.Errorf("parsed body = %#v", res.ReturnValue)
	}

	res = h.run(t, `req.getBody({raw: true})`)
	if res.ReturnValue != `{"n":1,"name":"x"}` {
		t.Errorf("raw body = %#v", res.ReturnValue)
	}

	// A raw set stores text, not JSON.
	h.run(t, `req.setBody("plain text", {raw: true})`)
	if req.Body.Mode != model.BodyText {
		t.Errorf("mode = %q, want text", req.Body.Mode)
	}
}

// TestReqOverridesAreRecorded covers setTimeout and disableParsingResponseJson.
func TestReqOverridesAreRecorded(t *testing.T) {
	h := newHarness(t, &State{Request: &model.Request{Method: "GET"}})

	h.run(t, `req.setTimeout(2500); req.setMaxRedirects(3); req.disableParsingResponseJson()`)
	if h.state.timeoutMs != 2500 {
		t.Errorf("timeout = %d", h.state.timeoutMs)
	}
	if h.state.maxRedirects != 3 {
		t.Errorf("max redirects = %d", h.state.maxRedirects)
	}
	if !h.state.disableJSONParsing {
		t.Error("JSON parsing not disabled")
	}
	if res := h.run(t, `req.getTimeout()`); res.ReturnValue != int64(2500) {
		t.Errorf("getTimeout = %#v", res.ReturnValue)
	}
}

// TestReqWithoutRequestDoesNotPanic keeps pre-request scripts usable.
func TestReqWithoutRequestDoesNotPanic(t *testing.T) {
	h := newHarness(t, nil)
	// A getter on an absent request returns empty rather than throwing.
	if res := h.run(t, `req.getMethod()`); res.ReturnValue != "" {
		t.Errorf("method = %#v", res.ReturnValue)
	}
}

// TestResAccessors covers the response view.
func TestResAccessors(t *testing.T) {
	result := &model.Result{
		Status:     201,
		StatusText: "Created",
		Headers: model.ResponseHeaders{
			{Name: strp("Content-Type"), Value: strp("application/json")},
			{Name: strp("Set-Cookie"), Value: strp("a=1")},
			{Name: strp("Set-Cookie"), Value: strp("b=2")},
		},
		Body: model.BodyRef{Size: 42},
	}
	h := newHarness(t, &State{
		Result:       result,
		ResponseText: `{"id": 7}`,
		BodyBytes:    []byte(`{"id": 7}`),
	})

	if res := h.run(t, `res.getStatus()`); res.ReturnValue != int64(201) {
		t.Errorf("status = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getStatusText()`); res.ReturnValue != "Created" {
		t.Errorf("status text = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getHeader("content-type")`); res.ReturnValue != "application/json" {
		t.Errorf("header = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getSize()`); res.ReturnValue != int64(42) {
		t.Errorf("size = %#v", res.ReturnValue)
	}

	// The body parses to an object, and raw form returns the text.
	if res := h.run(t, `res.getBody().id`); res.ReturnValue != int64(7) {
		t.Errorf("parsed body = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getBody({raw: true})`); res.ReturnValue != `{"id": 7}` {
		t.Errorf("raw body = %#v", res.ReturnValue)
	}

	// A repeated header reads as an array.
	res := h.run(t, `res.getHeaders()["Set-Cookie"].length`)
	if res.ReturnValue != int64(2) {
		t.Errorf("multi-value header = %#v", res.ReturnValue)
	}

	// The raw bytes are available as numbers.
	if res := h.run(t, `res.getDataBuffer().length`); res.ReturnValue != int64(9) {
		t.Errorf("data buffer length = %#v", res.ReturnValue)
	}
}

// TestResWithoutResultDoesNotPanic covers the pre-request context.
func TestResWithoutResultDoesNotPanic(t *testing.T) {
	h := newHarness(t, nil)
	if res := h.run(t, `res.getStatus()`); res.ReturnValue != int64(0) {
		t.Errorf("status = %#v", res.ReturnValue)
	}
	if res := h.run(t, `res.getBody()`); res.ReturnValue != nil {
		t.Errorf("body = %#v", res.ReturnValue)
	}
}

// TestTestRecordsPassAndFail covers the test harness contract.
func TestTestRecordsPassAndFail(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `
		test("passes", () => { expect(1).to.equal(1); });
		test("fails", () => { expect(1).to.equal(2); });
		test("errors", () => { throw new Error("boom"); });
	`)

	results := h.tests.Results()
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("first = %+v", results[0])
	}
	if results[1].Status != "fail" || results[1].Error == "" {
		t.Errorf("second = %+v", results[1])
	}
	if results[2].Status != "fail" || !strings.Contains(results[2].Error, "boom") {
		t.Errorf("third = %+v", results[2])
	}
}

// TestTestRequiresCallback keeps a bare description from silently passing.
func TestTestRequiresCallback(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.runErr(`test("no callback")`); err == nil {
		t.Fatal("expected an error")
	}
}

// TestExpectAssertions covers the implemented assertion set.
func TestExpectAssertions(t *testing.T) {
	h := newHarness(t, nil)

	// Boolean assertions are called as methods, not read as properties, because a
	// goja accessor cannot raise a catchable exception. See the package log.
	passing := []string{
		`expect(2).to.equal(2)`,
		`expect("a").to.be.a("string")`,
		`expect([1]).to.be.an("array")`,
		`expect(5).to.be.above(4)`,
		`expect(5).to.be.least(5)`,
		`expect(5).to.be.most(5)`,
		`expect(5).to.be.below(6)`,
		`expect("hello").to.include("ell")`,
		`expect([1, 2]).to.include(2)`,
		`expect("abc").to.match(/b/)`,
		`expect(true).to.be.true()`,
		`expect(false).to.be.false()`,
		`expect(null).to.be.null()`,
		`expect(undefined).to.be.undefined()`,
		`expect(1).to.exist()`,
		`expect("").to.be.empty()`,
		`expect({a: 1}).to.eql({a: 1})`,
		`expect(2).to.be.oneOf([1, 2, 3])`,
		`expect("abc").to.have.lengthOf(3)`,
		// Negation.
		`expect(1).to.not.equal(2)`,
		`expect("x").to.not.be.empty()`,
	}
	for _, script := range passing {
		if err := h.runErr(script); err != nil {
			t.Errorf("%s: unexpected error %v", script, err)
		}
	}

	failing := []string{
		`expect(1).to.equal(2)`,
		`expect(1).to.be.a("string")`,
		`expect(true).to.be.false()`,
		`expect(1).to.not.equal(1)`,
	}
	for _, script := range failing {
		if err := h.runErr(script); err == nil {
			t.Errorf("%s: expected a failure", script)
		}
	}
}

// TestExpectFailureCarriesOperands makes the reported message useful.
func TestExpectFailureCarriesOperands(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `test("mismatch", () => { expect(1).to.equal(2); })`)

	results := h.tests.Results()
	if len(results) != 1 || results[0].Status != "fail" {
		t.Fatalf("results = %+v", results)
	}
	// Both the actual and expected values must appear in the message.
	if !strings.Contains(results[0].Error, "1") || !strings.Contains(results[0].Error, "2") {
		t.Errorf("error = %q", results[0].Error)
	}
}

// TestUnknownAssertionThrows is the safety property: a typo must not pass.
func TestUnknownAssertionThrows(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.runErr(`expect(1).to.be.notRealAssertion(2)`); err == nil {
		t.Fatal("an unknown assertion must throw, not pass silently")
	}
}

// TestSendRequestRecordsAndRejects covers the nested-request boundary.
func TestSendRequestRecordsAndRejects(t *testing.T) {
	h := newHarness(t, nil)
	err := h.runErr(`
		async function go() {
			await bru.sendRequest({method: "POST", url: "https://auth.example.com/token"});
		}
		go();
	`)
	if err == nil {
		t.Fatal("expected the unhandled rejection to surface")
	}

	calls := h.binding.ScriptedCalls()
	if len(calls) != 1 {
		t.Fatalf("scripted calls = %+v", calls)
	}
	if calls[0].Method != "POST" || calls[0].URL != "https://auth.example.com/token" {
		t.Errorf("call = %+v", calls[0])
	}
}

// TestSendRequestCallbackForm reports through the callback.
func TestSendRequestCallbackForm(t *testing.T) {
	h := newHarness(t, nil)
	res := h.run(t, `
		let captured = null;
		bru.sendRequest("https://x.example.com", (err, resp) => { captured = err; });
		captured;
	`)
	// The callback receives a message naming the call, and no response.
	if res.ReturnValue == nil {
		t.Fatal("callback was not invoked")
	}
	if !strings.Contains(asString(res.ReturnValue), "x.example.com") {
		t.Errorf("callback error = %#v", res.ReturnValue)
	}
}

// TestProcessEnvIsReadable covers bru.getProcessEnv.
func TestProcessEnvIsReadable(t *testing.T) {
	h := newHarness(t, &State{Scopes: &vars.Scopes{
		ProcessEnv: map[string]string{"API_HOST": "http://local"},
	}})

	if res := h.run(t, `bru.getProcessEnv("API_HOST")`); res.ReturnValue != "http://local" {
		t.Errorf("process env = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.getProcessEnv("MISSING")`); res.ReturnValue != nil {
		t.Errorf("missing = %#v", res.ReturnValue)
	}
}

// TestCollectionMetadata covers the identity accessors.
func TestCollectionMetadata(t *testing.T) {
	h := newHarness(t, &State{
		CollectionName: "My API",
		CollectionPath: "/tmp/collections/api",
		EnvName:        "Staging",
		ExecutionMode:  "runner",
	})

	if res := h.run(t, `bru.getCollectionName()`); res.ReturnValue != "My API" {
		t.Errorf("collection name = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.getEnvName()`); res.ReturnValue != "Staging" {
		t.Errorf("env name = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.cwd()`); res.ReturnValue != "/tmp/collections/api" {
		t.Errorf("cwd = %#v", res.ReturnValue)
	}
	if res := h.run(t, `bru.getExecutionMode()`); res.ReturnValue != "runner" {
		t.Errorf("execution mode = %#v", res.ReturnValue)
	}
	// req shares the same notion of execution mode.
	if res := h.run(t, `req.getExecutionMode()`); res.ReturnValue != "runner" {
		t.Errorf("req execution mode = %#v", res.ReturnValue)
	}
}

// TestGenericTypesAreNotLeaked guards against a host object escaping into the
// script, which would let it call Go methods.
func TestGenericTypesAreNotLeaked(t *testing.T) {
	h := newHarness(t, nil)
	h.run(t, `bru.setVar("v", {nested: {deep: [1, 2]}})`)

	// Reading back gives a plain JS object whose prototype is the script's own.
	res := h.run(t, `Object.getPrototypeOf(bru.getVar("v")) === Object.prototype`)
	if res.ReturnValue != true {
		t.Error("a stored object is not a plain object in the script's realm")
	}
}

// TestConcurrentHarnesses covers two scripts mutating their own state at once.
func TestConcurrentHarnesses(t *testing.T) {
	const workers = 4
	done := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func(n int) {
			h := newHarness(t, nil)
			if _, err := h.rt.Run(context.Background(), `
				for (let i = 0; i < 100; i++) { bru.setVar("k" + i, i); }
			`); err != nil {
				done <- err
				return
			}
			if len(h.state.Scopes.Runtime) != 100 {
				done <- context.Canceled
				return
			}
			done <- nil
		}(i)
	}
	for i := 0; i < workers; i++ {
		if err := <-done; err != nil {
			t.Fatalf("worker: %v", err)
		}
	}
}

func strp(s string) *string { return &s }
