package model

import (
	"encoding/json"
	"testing"
)

func str(s string) *string { return &s }
func iptr(i int) *int      { return &i }

func TestTimeoutJSONUnion(t *testing.T) {
	cases := []struct {
		name string
		in   Timeout
		json string
	}{
		{"inherit", InheritTimeout, `"inherit"`},
		{"millis", TimeoutMillis(5000), `5000`},
		{"zero", TimeoutMillis(0), `0`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.json {
				t.Fatalf("marshal = %s, want %s", got, tc.json)
			}
			var back Timeout
			if err := json.Unmarshal(got, &back); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if back != tc.in {
				t.Fatalf("round-trip = %+v, want %+v", back, tc.in)
			}
		})
	}
}

func TestTimeoutRejectsBadValue(t *testing.T) {
	var out Timeout
	if err := json.Unmarshal([]byte(`"forever"`), &out); err == nil {
		t.Fatal("expected an error for an unknown keyword")
	}
}

func TestTimeoutNullIsUnset(t *testing.T) {
	var out Timeout
	if err := json.Unmarshal([]byte(`null`), &out); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !out.IsZero() {
		t.Fatalf("null should decode to the zero value, got %+v", out)
	}
}

func TestItemSettingsDefaults(t *testing.T) {
	var s ItemSettings
	if !s.FollowingRedirects() {
		t.Error("redirects should default to on")
	}
	if got := s.MaxRedirectCount(); got != DefaultMaxRedirects {
		t.Errorf("max redirects = %d, want %d", got, DefaultMaxRedirects)
	}
	if !s.ForwardsAuthHeader() {
		t.Error("forwarding the auth header should default to on")
	}
	if s.EncodesURL() {
		t.Error("URL encoding should default to off")
	}

	off := false
	s.FollowRedirects = &off
	if s.FollowingRedirects() {
		t.Error("explicit false must override the default")
	}
}

func TestRequestCloneIsDeep(t *testing.T) {
	orig := &Request{
		URL:    "https://example.test/a",
		Method: "GET",
		Headers: []KeyValue{
			{
				UID:         "h1",
				Name:        str("X-Test"),
				Value:       str("one"),
				Enabled:     true,
				Annotations: []Annotation{{Name: "a", Value: "b"}},
			},
		},
		Params: []Param{{
			KeyValue: KeyValue{UID: "p1", Name: str("q"), Value: str("1"), Enabled: true},
			Type:     ParamQuery,
		}},
		Body:   &Body{Mode: BodyJSON, JSON: str(`{"a":1}`)},
		Auth:   &Auth{Mode: AuthBearer, Token: str("tok")},
		Script: &Scripts{Req: str("req()"), Res: str("res()")},
		Tests:  str("test('x', () => {})"),
		Tags:   []string{"smoke"},
	}

	cp := orig.Clone()

	// Mutating the copy must not affect the original.
	*cp.Headers[0].Value = "two"
	cp.Headers[0].Annotations[0].Value = "changed"
	*cp.Params[0].Value = "2"
	*cp.Body.JSON = `{"a":2}`
	*cp.Auth.Token = "other"
	*cp.Script.Req = "changed()"
	*cp.Tests = "changed"
	cp.Tags[0] = "changed"

	if got := orig.Headers[0].ValueOrEmpty(); got != "one" {
		t.Errorf("header value leaked: %q", got)
	}
	if got := orig.Headers[0].Annotations[0].Value; got != "b" {
		t.Errorf("annotation leaked: %q", got)
	}
	if got := orig.Params[0].ValueOrEmpty(); got != "1" {
		t.Errorf("param leaked: %q", got)
	}
	if got := DerefString(orig.Body.JSON); got != `{"a":1}` {
		t.Errorf("body leaked: %q", got)
	}
	if got := DerefString(orig.Auth.Token); got != "tok" {
		t.Errorf("auth leaked: %q", got)
	}
	if got := DerefString(orig.Script.Req); got != "req()" {
		t.Errorf("script leaked: %q", got)
	}
	if got := DerefString(orig.Tests); got != "test('x', () => {})" {
		t.Errorf("tests leaked: %q", got)
	}
	if got := orig.Tags[0]; got != "smoke" {
		t.Errorf("tags leaked: %q", got)
	}
}

func TestRequestCloneNil(t *testing.T) {
	var r *Request
	if got := r.Clone(); got != nil {
		t.Fatalf("cloning nil should give nil, got %+v", got)
	}
}

func TestRequestLiveFilters(t *testing.T) {
	r := &Request{
		Headers: []KeyValue{
			{Name: str("A"), Value: str("1"), Enabled: true},
			{Name: str("B"), Value: str("2"), Enabled: false},
			{Name: str(""), Value: str("3"), Enabled: true},
		},
		Params: []Param{
			{KeyValue: KeyValue{Name: str("q"), Enabled: true}, Type: ParamQuery},
			{KeyValue: KeyValue{Name: str("id"), Enabled: true}, Type: ParamPath},
			{KeyValue: KeyValue{Name: str("off"), Enabled: false}, Type: ParamQuery},
		},
	}
	if got := len(r.LiveHeaders()); got != 1 {
		t.Errorf("live headers = %d, want 1", got)
	}
	if got := len(r.QueryParams()); got != 1 {
		t.Errorf("query params = %d, want 1", got)
	}
	if got := len(r.PathParams()); got != 1 {
		t.Errorf("path params = %d, want 1", got)
	}
}

func TestDraftDuality(t *testing.T) {
	saved := &Request{URL: "https://example.test/saved"}
	item := &Item{UID: "i1", Type: ItemHTTPRequest, Request: saved}

	if item.Dirty() {
		t.Fatal("a fresh item is not dirty")
	}
	if item.Effective() != saved {
		t.Fatal("with no draft, Effective should be the saved request")
	}

	draft := saved.Clone()
	draft.URL = "https://example.test/draft"
	item.Draft = draft

	if !item.Dirty() {
		t.Fatal("item with a draft is dirty")
	}
	if got := item.Effective().URL; got != "https://example.test/draft" {
		t.Errorf("Effective = %q, want the draft", got)
	}
	if saved.URL != "https://example.test/saved" {
		t.Error("saved request must not be mutated by the draft")
	}
}

func TestCollectionFindItemAndAncestors(t *testing.T) {
	root := &Collection{
		Items: []*Item{
			{
				UID: "folder", Type: ItemFolder, Name: "Folder",
				Children: []*Item{
					{
						UID: "nested", Type: ItemFolder, Name: "Nested",
						Children: []*Item{
							{UID: "req", Type: ItemHTTPRequest, Name: "Req"},
						},
					},
				},
			},
			{UID: "top", Type: ItemHTTPRequest, Name: "Top"},
		},
	}

	if got := root.FindItem("req"); got == nil || got.Name != "Req" {
		t.Fatalf("FindItem(req) = %+v", got)
	}
	if got := root.FindItem("missing"); got != nil {
		t.Fatalf("FindItem(missing) = %+v, want nil", got)
	}

	anc := root.Ancestors("req")
	if len(anc) != 2 {
		t.Fatalf("ancestors = %d, want 2", len(anc))
	}
	if anc[0].UID != "folder" || anc[1].UID != "nested" {
		t.Errorf("ancestor order = %s, %s; want outermost first", anc[0].UID, anc[1].UID)
	}
	if got := root.Ancestors("top"); len(got) != 0 {
		t.Errorf("top-level item should have no ancestors, got %d", len(got))
	}
}

func TestCollectionWalkVisitsEveryItem(t *testing.T) {
	root := &Collection{Items: []*Item{
		{UID: "a", Children: []*Item{{UID: "b"}, {UID: "c"}}},
		{UID: "d"},
	}}
	var seen []string
	if err := root.Walk(func(it *Item) error {
		seen = append(seen, it.UID)
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	want := []string{"a", "b", "c", "d"}
	if len(seen) != len(want) {
		t.Fatalf("visited %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("visited %v, want %v (depth-first, parents first)", seen, want)
		}
	}
}

func TestAuthModeValidity(t *testing.T) {
	for _, m := range AllAuthModes {
		if !m.Valid() {
			t.Errorf("mode %q should be valid", m)
		}
	}
	if AuthMode("carrier-pigeon").Valid() {
		t.Error("an unknown mode must not validate")
	}
	var a *Auth
	if got := a.EffectiveMode(); got != AuthInherit {
		t.Errorf("nil auth effective mode = %q, want inherit", got)
	}
	if got := (&Auth{}).EffectiveMode(); got != AuthInherit {
		t.Errorf("empty mode should normalise to inherit, got %q", got)
	}
}

func TestBodySelectedFile(t *testing.T) {
	var b *Body
	if b.SelectedFile() != nil {
		t.Error("nil body has no file")
	}
	b = &Body{File: []FileEntry{{UID: "f1"}, {UID: "f2", Selected: true}}}
	got := b.SelectedFile()
	if got == nil || got.UID != "f2" {
		t.Fatalf("SelectedFile = %+v, want f2", got)
	}
	if !(&Body{Mode: BodyNone}).IsEmpty() {
		t.Error("mode none should be empty")
	}
	if (&Body{Mode: BodyJSON}).IsEmpty() {
		t.Error("json mode is not empty")
	}
}

func TestResponseHeadersPreserveDuplicates(t *testing.T) {
	h := ResponseHeaders{
		{Name: str("Set-Cookie"), Value: str("a=1")},
		{Name: str("set-cookie"), Value: str("b=2")},
		{Name: str("Content-Type"), Value: str("application/json")},
	}
	cookies := h.Values("SET-COOKIE")
	if len(cookies) != 2 {
		t.Fatalf("expected both Set-Cookie values (case-insensitive), got %v", cookies)
	}
	if cookies[0] != "a=1" || cookies[1] != "b=2" {
		t.Errorf("duplicates were reordered: %v", cookies)
	}
}

func TestSentRequestHeaderLookup(t *testing.T) {
	s := &SentRequest{Headers: []KeyValue{
		{Name: str("Content-Type"), Value: str("application/json")},
	}}
	if v, ok := s.Header("content-type"); !ok || v != "application/json" {
		t.Errorf("case-insensitive lookup failed: %q %v", v, ok)
	}
	if _, ok := s.Header("X-Missing"); ok {
		t.Error("missing header reported as present")
	}
}

func TestResultStatusClass(t *testing.T) {
	cases := map[int]int{200: 2, 204: 2, 301: 3, 404: 4, 500: 5, 0: 0, 99: 0}
	for status, want := range cases {
		r := &Result{Status: status}
		if got := r.StatusClass(); got != want {
			t.Errorf("status %d → class %d, want %d", status, got, want)
		}
	}
	if !(&Result{Err: "boom"}).Failed() {
		t.Error("a result with an error should report Failed")
	}
	if (&Result{Status: 200}).Failed() {
		t.Error("a clean result is not Failed")
	}
}

func TestTimelineTotal(t *testing.T) {
	var tl Timeline
	if got := tl.Total(); got != 0 {
		t.Errorf("empty timeline total = %v", got)
	}
}

func TestDefaultSettingsAreValid(t *testing.T) {
	s := DefaultSettings()
	if s.SchemaVersion != CurrentSettingsSchemaVersion {
		t.Errorf("schema version = %d", s.SchemaVersion)
	}
	if s.Request.MaxResponseBytes <= s.Request.InlineBodyBytes {
		t.Error("the spill threshold must be below the hard cap")
	}
	if !s.TLS.VerifyServerCertificates {
		t.Error("certificate verification must default to on")
	}
	if s.Appearance.Zoom != 100 {
		t.Errorf("zoom = %d", s.Appearance.Zoom)
	}
}

func TestEnvironmentVariableValue(t *testing.T) {
	env := &Environment{Variables: []Variable{
		{Name: str("host"), Value: "example.test", Enabled: true},
		{Name: str("off"), Value: "no", Enabled: false},
	}}
	if v, ok := env.VariableValue("host"); !ok || v != "example.test" {
		t.Errorf("VariableValue(host) = %v, %v", v, ok)
	}
	if _, ok := env.VariableValue("off"); ok {
		t.Error("a disabled variable must not resolve")
	}
	var nilEnv *Environment
	if _, ok := nilEnv.VariableValue("x"); ok {
		t.Error("nil environment must not resolve anything")
	}
}

func TestCollectionFormatValid(t *testing.T) {
	for _, f := range []CollectionFormat{FormatBru, FormatYML} {
		if !f.Valid() {
			t.Errorf("%q should be valid", f)
		}
	}
	if CollectionFormat("toml").Valid() {
		t.Error("an unknown format must not validate")
	}
}

func TestItemTypeIsRequest(t *testing.T) {
	requests := []ItemType{ItemHTTPRequest, ItemGraphQLRequest, ItemGRPCRequest, ItemWSRequest}
	for _, it := range requests {
		if !it.IsRequest() {
			t.Errorf("%q should be a request", it)
		}
	}
	for _, it := range []ItemType{ItemFolder, ItemJS, ItemApp} {
		if it.IsRequest() {
			t.Errorf("%q should not be a request", it)
		}
	}
}

func TestWorkspaceStateLookups(t *testing.T) {
	var nilState *WorkspaceState
	if nilState.FindCollection("x") != nil || nilState.FindTab("x") != nil || nilState.ActiveTabState() != nil {
		t.Error("nil state must return nil from every lookup")
	}
	if nilState.IsPending("x") || nilState.ResponseFor("x") != nil {
		t.Error("nil state must report nothing pending")
	}

	s := &WorkspaceState{
		Collections: []*Collection{{UID: "c1"}},
		Tabs:        []*Tab{{UID: "t1"}, {UID: "t2"}},
		ActiveTab:   "t2",
		Pending:     map[string]bool{"i1": true},
		Responses:   map[string]*Result{"i1": {Status: 201}},
	}
	if s.FindCollection("c1") == nil {
		t.Error("collection not found")
	}
	if got := s.ActiveTabState(); got == nil || got.UID != "t2" {
		t.Errorf("ActiveTabState = %+v", got)
	}
	if !s.IsPending("i1") {
		t.Error("i1 should be pending")
	}
	if got := s.ResponseFor("i1"); got == nil || got.Status != 201 {
		t.Errorf("ResponseFor = %+v", got)
	}
}
