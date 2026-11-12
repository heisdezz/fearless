package dialogs

import (
	"context"
	"errors"
	"fearless/internal/core/model"
	"strings"
	"testing"
	"time"
)

// timeout bounds a test that waits on a goroutine.
func timeout() <-chan time.Time { return time.After(2 * time.Second) }

// TestParseImportCurl covers the single-request import path.
func TestParseImportCurl(t *testing.T) {
	imp, err := ParseImport(ImportAuto, `curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{"name":"ada"}'`)
	if err != nil {
		t.Fatalf("ParseImport: %v", err)
	}
	if imp.Source != ImportCurl {
		t.Errorf("source = %s, want cURL", imp.Source)
	}
	if imp.Request == nil {
		t.Fatal("no request produced")
	}
	if imp.Collection != nil {
		t.Error("a cURL import produced a collection")
	}
	if imp.Request.Method != "POST" {
		t.Errorf("method = %q", imp.Request.Method)
	}
	if !strings.Contains(imp.Request.URL, "api.example.com/users") {
		t.Errorf("url = %q", imp.Request.URL)
	}
}

// TestParseImportAutoDetectsJSONFormats covers the sniffing heuristic.
func TestParseImportAutoDetectsJSONFormats(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want ImportSource
	}{
		{
			name: "postman",
			doc:  `{"info":{"name":"C","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`,
			want: ImportPostman,
		},
		{
			name: "insomnia",
			doc:  `{"_type":"export","__export_format":4,"resources":[]}`,
			want: ImportInsomnia,
		},
		{
			name: "insomnia pretty-printed",
			doc:  "{\n  \"_type\": \"export\",\n  \"__export_format\": 4,\n  \"resources\": []\n}",
			want: ImportInsomnia,
		},
		{
			name: "postman pretty-printed",
			doc:  "{\n  \"info\": {\n    \"name\": \"C\",\n    \"schema\": \"https://schema.getpostman.com/json/collection/v2.1.0/collection.json\"\n  },\n  \"item\": []\n}",
			want: ImportPostman,
		},
		{
			name: "openapi yaml",
			doc:  "openapi: 3.0.0\ninfo:\n  title: T\n  version: '1'\npaths: {}\n",
			want: ImportOpenAPI,
		},
		{
			name: "openapi",
			doc:  `{"openapi":"3.0.0","info":{"title":"T","version":"1"},"paths":{}}`,
			want: ImportOpenAPI,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectImportSource(ImportAuto, tc.doc); got != tc.want {
				t.Errorf("detected %s, want %s", got, tc.want)
			}
		})
	}
}

// TestParseImportRejectsEmpty covers the guard against an empty paste.
func TestParseImportRejectsEmpty(t *testing.T) {
	if _, err := ParseImport(ImportAuto, "   \n\t "); !errors.Is(err, ErrImportEmpty) {
		t.Fatalf("err = %v, want ErrImportEmpty", err)
	}
}

// TestParseImportUnknownDocument covers a JSON document that is none of the
// supported formats: it must fail loudly, not import something surprising.
func TestParseImportUnknownDocument(t *testing.T) {
	_, err := ParseImport(ImportAuto, `{"hello":"world"}`)
	if err == nil {
		t.Fatal("an unrecognized document imported without error")
	}
	if !errors.Is(err, ErrImportUnknown) && !strings.Contains(err.Error(), "could not tell") {
		t.Errorf("err = %v, want an unknown-format error", err)
	}
}

// TestImportModalRunsHandler covers the dialog glue: OK parses and hands the
// result to the handler, then closes.
func TestImportModalRunsHandler(t *testing.T) {
	q := NewQueue()
	var got *Imported
	w, err := q.ImportModal(func(_ context.Context, imp *Imported) error {
		got = imp
		return nil
	})
	if err != nil {
		t.Fatalf("ImportModal: %v", err)
	}
	if !q.Busy() {
		t.Error("Busy is false before a document is pasted")
	}

	q.SetValue("document", "curl https://example.com/ping")
	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	if got == nil || got.Request == nil {
		t.Fatal("the handler did not receive the request")
	}
	if q.Active() {
		t.Error("dialog still active after a successful import")
	}
}

// TestImportModalKeepsOpenOnParseError covers a bad paste: the dialog stays up
// with the parser's message so the user can correct it.
func TestImportModalKeepsOpenOnParseError(t *testing.T) {
	q := NewQueue()
	called := false
	w, err := q.ImportModal(func(context.Context, *Imported) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("ImportModal: %v", err)
	}

	q.SetValue("document", "{not valid at all")
	q.OK()
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	if called {
		t.Error("the handler ran for a document that did not parse")
	}
	if !q.Active() {
		t.Fatal("the dialog closed on a parse error")
	}
	if q.Message() == "" {
		t.Error("no error message shown")
	}
	q.Cancel()
	_ = w
}

// TestSaveTransientRequiresACollection covers the empty-picker guard.
func TestSaveTransientRequiresACollection(t *testing.T) {
	q := NewQueue()
	_, err := q.SaveTransientModal(SaveTransient{Name: "x"}, nil)
	if !errors.Is(err, ErrNoSaveTarget) {
		t.Fatalf("err = %v, want ErrNoSaveTarget", err)
	}
	if q.Active() {
		t.Error("a dialog opened with no targets to choose from")
	}
}

// TestSaveTransientPrefillsNameAndMapsTarget covers the save flow end to end.
func TestSaveTransientPrefillsNameAndMapsTarget(t *testing.T) {
	q := NewQueue()
	var gotUID, gotName string
	w, err := q.SaveTransientModal(SaveTransient{
		URL: "https://api.example.com/v1/users",
		Targets: []SaveTarget{
			{UID: "col-1", Name: "Todo API"},
			{UID: "col-2", Name: "Billing"},
		},
	}, func(_ context.Context, uid, name string) error {
		gotUID, gotName = uid, name
		return nil
	})
	if err != nil {
		t.Fatalf("SaveTransientModal: %v", err)
	}

	if got := q.Value("name"); got != "users" {
		t.Errorf("prefilled name = %q, want users", got)
	}
	labels := q.TargetLabels()
	if len(labels) != 2 || labels[0] != "Todo API" {
		t.Fatalf("labels = %v", labels)
	}
	// The collection starts unset, so OK is unavailable until it is chosen.
	if !q.Busy() {
		t.Error("Busy is false with no collection chosen")
	}
	q.SetValue("collection", "Billing")
	if q.Busy() {
		t.Error("Busy is true with both fields filled")
	}

	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	if gotUID != "col-2" {
		t.Errorf("collection UID = %q, want col-2 (mapped from the label)", gotUID)
	}
	if gotName != "users" {
		t.Errorf("name = %q", gotName)
	}
}

// TestSuggestedName covers the URL-to-name derivation.
func TestSuggestedName(t *testing.T) {
	cases := map[string]string{
		"https://api.example.com/v1/users":   "users",
		"https://api.example.com/v1/users/":  "users",
		"https://api.example.com/v1/users?x": "users",
		"https://api.example.com":            "api.example.com",
		"https://api.example.com/":           "api.example.com",
		"":                                   "Untitled",
		"   ":                                "Untitled",
	}
	for in, want := range cases {
		if got := suggestedName(in); got != want {
			t.Errorf("suggestedName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestPromptLabelsScansEnabledFieldsOnly covers the prompt-variable scan.
func TestPromptLabelsScansEnabledFieldsOnly(t *testing.T) {
	r := RequestView(
		"https://api/{{?host}}/users/{{?id}}?x={{?host}}",
		`{"token":"{{?token}}"}`,
		[]RequestPart{
			{Name: "X-A", Value: "{{?headerVar}}", Enabled: true},
			{Name: "X-B", Value: "{{?disabledVar}}", Enabled: false},
			{Name: "X-C", Value: "{{?token}}", Enabled: true},
		},
	)
	got := PromptLabels(r)
	want := []string{"host", "id", "token", "headerVar"}
	if len(got) != len(want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("labels = %v, want %v (order matters: it is first-appearance)", got, want)
		}
	}
}

// TestPromptLabelsIgnoresOrdinaryVariables keeps `{{name}}` from popping a
// dialog; only `{{?name}}` is a prompt.
func TestPromptLabelsIgnoresOrdinaryVariables(t *testing.T) {
	r := RequestView("https://api/{{host}}/users", "", nil)
	if got := PromptLabels(r); len(got) != 0 {
		t.Errorf("labels = %v, want none", got)
	}
	if got := PromptLabels(nil); got != nil {
		t.Errorf("nil request produced %v", got)
	}
}

// TestPromptValuesScope covers the nil-vs-empty contract the send path relies
// on: no labels means no prompt scope at all.
func TestPromptValuesScope(t *testing.T) {
	if got := (PromptValues{}).Scope(); got != nil {
		t.Errorf("empty prompt produced %v, want nil", got)
	}
	p := PromptValues{Labels: []string{"a"}, Values: map[string]string{"a": "1"}}
	if got := p.Scope(); got["a"] != "1" {
		t.Errorf("scope = %v", got)
	}
}

// TestValidatePrompt covers the belt-and-braces check.
func TestValidatePrompt(t *testing.T) {
	p := PromptValues{
		Labels: []string{"a", "b"},
		Values: map[string]string{"a": "1", "b": " "},
	}
	if err := ValidatePrompt(p); err == nil {
		t.Fatal("a blank value passed validation")
	}
	p.Values["b"] = "2"
	if err := ValidatePrompt(p); err != nil {
		t.Fatalf("valid prompt rejected: %v", err)
	}
}

// TestAskPromptWithNoLabelsSkipsTheDialog keeps a request with no prompt
// variables from flashing a modal.
func TestAskPromptWithNoLabelsSkipsTheDialog(t *testing.T) {
	q := NewQueue()
	values, err := q.AskPrompt(context.Background(), nil)
	if err != nil || values != nil {
		t.Fatalf("values = %v, err = %v; want nil, nil", values, err)
	}
	if q.Active() {
		t.Error("a dialog opened with no labels to ask for")
	}
}

// TestAskPromptCollectsThroughTheQueue covers the blocking form the send path
// uses.
func TestAskPromptCollectsThroughTheQueue(t *testing.T) {
	q := NewQueue()
	done := make(chan map[string]string, 1)
	go func() {
		values, err := q.AskPrompt(context.Background(), []string{"token"})
		if err != nil {
			done <- nil
			return
		}
		done <- values
	}()

	waitFor(t, "the prompt to open", func() bool { return q.Kind() == KindPrompt && q.Phase() == PhaseOpen })
	q.SetValue("token", "s3cret")
	q.OK()

	select {
	case values := <-done:
		if values["token"] != "s3cret" {
			t.Errorf("values = %v", values)
		}
	case <-timeout():
		t.Fatal("AskPrompt never returned")
	}
}

// TestAskPromptContextCancellationClosesTheDialog keeps a cancelled send from
// leaving a question on screen.
func TestAskPromptContextCancellationClosesTheDialog(t *testing.T) {
	q := NewQueue()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := q.AskPrompt(ctx, []string{"token"})
		done <- err
	}()

	waitFor(t, "the prompt to open", func() bool { return q.Kind() == KindPrompt })
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, ErrSuperseded) {
			t.Fatalf("err = %v, want ErrSuperseded", err)
		}
	case <-timeout():
		t.Fatal("AskPrompt ignored the caller's cancellation")
	}
	if q.Active() {
		t.Error("the dialog stayed open after its caller was cancelled")
	}
}

func TestCreateCollectionModal(t *testing.T) {
	q := NewQueue()
	var gotName, gotLoc string
	var gotFmt model.CollectionFormat
	wait, err := q.CreateCollectionModal(func(ctx context.Context, name, loc string, format model.CollectionFormat) error {
		gotName = name
		gotLoc = loc
		gotFmt = format
		return nil
	})
	if err != nil {
		t.Fatalf("CreateCollectionModal error: %v", err)
	}
	if q.Kind() != KindCreateCollection {
		t.Fatalf("kind = %v, want KindCreateCollection", q.Kind())
	}
	q.SetValue("name", "Test Col")
	q.SetValue("location", "/tmp/test")
	q.SetValue("format", "yml")
	q.OK()

	if _, err := wait.Result(); err != nil {
		t.Fatalf("wait.Result error: %v", err)
	}

	if gotName != "Test Col" || gotLoc != "/tmp/test" || gotFmt != model.FormatYML {
		t.Errorf("got (%s, %s, %s), want (Test Col, /tmp/test, %s)", gotName, gotLoc, gotFmt, model.FormatYML)
	}
}

func TestParseImportURLFallback(t *testing.T) {
	imp, err := ParseImport(ImportAuto, "https://api.example.com/health")
	if err != nil {
		t.Fatalf("ParseImport error: %v", err)
	}
	if imp.Request == nil || imp.Request.URL != "https://api.example.com/health" || imp.Request.Method != "GET" {
		t.Errorf("unexpected imported request: %+v", imp.Request)
	}
}
