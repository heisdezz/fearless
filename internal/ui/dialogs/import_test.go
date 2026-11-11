package dialogs

import (
	"strings"
	"testing"
)

// TestParseImportEachFormat drives all four converters through the dialog's
// parser, which is the path a paste actually takes.
func TestParseImportEachFormat(t *testing.T) {
	cases := []struct {
		name       string
		src        ImportSource
		doc        string
		wantSource ImportSource
		wantColl   bool
	}{
		{
			name:       "curl explicit",
			src:        ImportCurl,
			doc:        `curl https://api.example.com/ping`,
			wantSource: ImportCurl,
		},
		{
			name: "postman",
			src:  ImportPostman,
			doc: `{
  "info": {
    "name": "Todo",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "List",
      "request": {
        "method": "GET",
        "url": { "raw": "https://api.example.com/todos", "host": ["api","example","com"], "path": ["todos"] }
      }
    }
  ]
}`,
			wantSource: ImportPostman,
			wantColl:   true,
		},
		{
			name: "insomnia",
			src:  ImportInsomnia,
			doc: `{
  "_type": "export",
  "__export_format": 4,
  "resources": [
    {
      "_id": "req_1",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "Ping",
      "method": "GET",
      "url": "https://api.example.com/ping",
      "body": {},
      "headers": []
    }
  ]
}`,
			wantSource: ImportInsomnia,
			wantColl:   true,
		},
		{
			name: "openapi",
			src:  ImportOpenAPI,
			doc: `{
  "openapi": "3.0.0",
  "info": { "title": "Demo", "version": "1.0" },
  "paths": {
    "/ping": {
      "get": { "operationId": "ping", "responses": { "200": { "description": "ok" } } }
    }
  }
}`,
			wantSource: ImportOpenAPI,
			wantColl:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imp, err := ParseImport(tc.src, tc.doc)
			if err != nil {
				t.Fatalf("ParseImport: %v", err)
			}
			if imp.Source != tc.wantSource {
				t.Errorf("source = %s, want %s", imp.Source, tc.wantSource)
			}
			if tc.wantColl {
				if imp.Collection == nil {
					t.Fatal("no collection produced")
				}
				if imp.Request != nil {
					t.Error("a collection import also produced a bare request")
				}
				return
			}
			if imp.Request == nil {
				t.Fatal("no request produced")
			}
		})
	}
}

// TestParseImportMalformedRejects covers the error branch of each converter: a
// chosen format whose document does not parse must name the format.
func TestParseImportMalformedRejects(t *testing.T) {
	cases := []struct {
		src  ImportSource
		doc  string
		want string
	}{
		{ImportCurl, "curl --this-flag-does-not-exist", "cURL"},
		{ImportPostman, "{", "Postman"},
		{ImportInsomnia, "{", "Insomnia"},
		{ImportOpenAPI, "{", "OpenAPI"},
	}
	for _, tc := range cases {
		_, err := ParseImport(tc.src, tc.doc)
		if err == nil {
			t.Errorf("%s: a malformed document imported without error", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", tc.src, err, tc.want)
		}
	}
}

// TestImportSourceString covers the picker labels.
func TestImportSourceString(t *testing.T) {
	want := map[ImportSource]string{
		ImportAuto:     "Auto-detect",
		ImportCurl:     "cURL",
		ImportPostman:  "Postman",
		ImportInsomnia: "Insomnia",
		ImportOpenAPI:  "OpenAPI",
	}
	for src, s := range want {
		if got := src.String(); got != s {
			t.Errorf("%d.String() = %q, want %q", src, got, s)
		}
	}
}

// TestKindAndPhaseString covers the diagnostic labels.
func TestKindAndPhaseString(t *testing.T) {
	if got := KindPrompt.String(); got != "prompt" {
		t.Errorf("kind = %q", got)
	}
	if got := Kind(99).String(); got != "unknown" {
		t.Errorf("kind = %q", got)
	}
	if got := PhaseRunning.String(); got != "running" {
		t.Errorf("phase = %q", got)
	}
	if got := Phase(99).String(); got != "unknown" {
		t.Errorf("phase = %q", got)
	}
}

// TestJoinWarningsTruncates keeps a long warning list inside the dialog.
func TestJoinWarningsTruncates(t *testing.T) {
	ws := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	got := joinWarnings(ws)
	if !strings.Contains(got, "• a") {
		t.Errorf("missing the first warning: %q", got)
	}
	if strings.Contains(got, "• h") {
		t.Errorf("the list was not truncated: %q", got)
	}
	if !strings.Contains(got, "…") {
		t.Errorf("no truncation marker: %q", got)
	}
	if joinWarnings(nil) != "" {
		t.Error("an empty list produced text")
	}
}

// TestValuesAndEmpty covers the copy accessor and the Empty predicate.
func TestValuesAndEmpty(t *testing.T) {
	q := NewQueue()
	if _, err := q.Prompt("Vars", []string{"a"}, nil); err != nil {
		t.Fatalf("prompt: %v", err)
	}
	q.SetValue("a", "1")

	values := q.Values()
	values["a"] = "mutated"
	if got := q.Value("a"); got != "1" {
		t.Errorf("Value = %q: the caller mutated the queue through a copy", got)
	}

	if !(PromptValues{}).Empty() {
		t.Error("an empty PromptValues is not Empty")
	}
	if (PromptValues{Labels: []string{"a"}}).Empty() {
		t.Error("a non-empty PromptValues is Empty")
	}
}

// TestSetWarningsAndTargets covers the advisory state the views read.
func TestSetWarningsAndTargets(t *testing.T) {
	q := NewQueue()
	if _, err := q.Confirm("A", "", "", nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	q.SetWarnings([]string{"dropped a header"})
	if got := q.Warnings(); len(got) != 1 || got[0] != "dropped a header" {
		t.Errorf("warnings = %v", got)
	}

	q.SetTargets([]SaveTarget{{UID: "c1", Name: "One"}})
	if got := q.TargetLabels(); len(got) != 1 || got[0] != "One" {
		t.Errorf("labels = %v", got)
	}
	if got := q.TargetUID("One"); got != "c1" {
		t.Errorf("TargetUID by name = %q", got)
	}
	if got := q.TargetUID("c1"); got != "c1" {
		t.Errorf("TargetUID by UID = %q", got)
	}
	// An unknown label passes through so the save command reports it.
	if got := q.TargetUID("nonexistent"); got != "nonexistent" {
		t.Errorf("TargetUID(unknown) = %q", got)
	}
}
