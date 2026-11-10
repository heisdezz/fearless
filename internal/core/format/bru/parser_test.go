package bru

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseFile is a helper for fixture-driven tests.
func parseFile(t *testing.T, path string) *Document {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

func TestParseBasicRequest(t *testing.T) {
	doc := parseFile(t, "../../../../testdata/corpus/bru/basic/request_1.bru")

	meta, ok := doc.FindBlock("meta")
	if !ok {
		t.Fatal("no meta block")
	}
	if got := meta.Pairs[0]; got.Key != "name" || got.Value != "request_1" {
		t.Errorf("meta name = %q / %q", got.Key, got.Value)
	}

	// The method block's name carries the verb.
	get, ok := doc.FindBlock("get")
	if !ok {
		t.Fatal("no get block")
	}
	var url, body, auth string
	for _, p := range get.Pairs {
		switch p.Key {
		case "url":
			url = p.Value
		case "body":
			body = p.Value
		case "auth":
			auth = p.Value
		}
	}
	if url != "https://api.example.com/one" {
		t.Errorf("url = %q", url)
	}
	if body != "none" || auth != "none" {
		t.Errorf("body/auth = %q/%q", body, auth)
	}
}

func TestParseFolderFile(t *testing.T) {
	doc := parseFile(t, "../../../../testdata/corpus/bru/basic/folder_1/folder.bru")
	if _, ok := doc.FindBlock("meta"); !ok {
		t.Fatal("no meta block")
	}
	// A folder file has no method block; the mapper relies on that.
	for name := range methodBlocks {
		if _, ok := doc.FindBlock(name); ok {
			t.Errorf("folder file unexpectedly has a %s block", name)
		}
	}
}

func TestParseEnvironmentExtensions(t *testing.T) {
	doc := parseFile(t, "../../../../testdata/corpus/environments/Local.bru")

	vars, ok := doc.FindBlock("vars")
	if !ok {
		t.Fatal("no vars block")
	}
	if len(vars.Pairs) < 5 {
		t.Fatalf("only %d vars parsed", len(vars.Pairs))
	}

	// An annotated typed variable.
	var found bool
	for i, p := range vars.Pairs {
		if p.Key == "typed_env_num" {
			found = true
			if p.Value != "300" {
				t.Errorf("typed_env_num = %q", p.Value)
			}
			// The @number annotation precedes the pair.
			prev := vars.Pairs[i-1]
			if len(prev.Annotations) == 0 || prev.Annotations[0].Name != "number" {
				// Annotations may attach to the annotated pair itself.
				if _, ok := p.Annotation("number"); !ok {
					t.Errorf("@number not attached near typed_env_num: prev %+v, self %+v",
						prev.Annotations, p.Annotations)
				}
			}
		}
	}
	if !found {
		t.Fatal("typed_env_num not parsed")
	}

	// vars:secret is a list block.
	secret, ok := doc.FindBlock("vars:secret")
	if !ok {
		t.Fatal("no vars:secret block")
	}
	if secret.Kind != ListBlock {
		t.Fatalf("vars:secret kind = %v", secret.Kind)
	}
	if len(secret.Items) == 0 {
		t.Fatal("vars:secret has no items")
	}
	if secret.Items[0] != "github_client_secret" {
		t.Errorf("first secret = %q", secret.Items[0])
	}
}

func TestParseJSONBodyAsText(t *testing.T) {
	src := `meta {
  name: body test
}

post {
  url: http://x
  body: json
}

body:json {
  {
    "a": {"nested": true},
    "b": "}"
  }
}
`
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	body, ok := doc.FindBlock("body:json")
	if !ok {
		t.Fatal("no body:json")
	}
	if body.Kind != TextBlock {
		t.Fatalf("body:json kind = %v, want text", body.Kind)
	}
	if !strings.Contains(body.Text, `"nested": true`) {
		t.Errorf("body text lost content:\n%s", body.Text)
	}
	// The closing brace must be the block terminator, not part of the body.
	if strings.HasSuffix(strings.TrimSpace(body.Text), "}\n}") {
		t.Error("terminator leaked into the body")
	}
}

func TestParseMultipartWithFileAnnotation(t *testing.T) {
	src := `post {
  url: http://x
  body: multipartForm
}

body:multipart-form {
  sample: @file(bruno.png) @contentType(image/png)
  foo: {"bar":"baz"} @contentType(application/json--test)
}
`
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	block, ok := doc.FindBlock("body:multipart-form")
	if !ok {
		t.Fatal("no multipart block")
	}
	if block.Kind != DictBlock {
		t.Fatalf("multipart should be a dictionary, got %v", block.Kind)
	}
	if len(block.Pairs) != 2 {
		t.Fatalf("got %d parts, want 2", len(block.Pairs))
	}
	// The value is the raw text including annotations; the mapper extracts them.
	if !strings.Contains(block.Pairs[0].Value, "@file(bruno.png)") {
		t.Errorf("file annotation lost: %q", block.Pairs[0].Value)
	}
}

func TestParseDisabledAndQuotedKeys(t *testing.T) {
	src := `headers {
  content-type: application/json
  ~x-debug: 1
  "key:with:colons": value
}
`
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h, _ := doc.FindBlock("headers")
	if len(h.Pairs) != 3 {
		t.Fatalf("got %d pairs", len(h.Pairs))
	}
	if h.Pairs[1].Key != "x-debug" || !h.Pairs[1].Disabled {
		t.Errorf("disabled pair = %+v", h.Pairs[1])
	}
	if h.Pairs[2].Key != "key:with:colons" || !h.Pairs[2].Quoted {
		t.Errorf("quoted key = %+v", h.Pairs[2])
	}
	if h.Pairs[2].Value != "value" {
		t.Errorf("quoted key value = %q", h.Pairs[2].Value)
	}
}

func TestParseMultilineValueDedent(t *testing.T) {
	src := `vars {
  obj: '''
    {"scope":"env"}
  '''
}
`
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Bruno's multilinetextblock strips exactly four characters per line and keeps
	// the newline that follows the opening '''. Verified against
	// bruToJson.js (`line.slice(4)`, no leading-newline removal).
	v, _ := doc.FindBlock("vars")
	if got := v.Pairs[0].Value; got != "\n{\"scope\":\"env\"}\n" {
		t.Errorf("multiline value = %q", got)
	}
}

func TestParseCRLF(t *testing.T) {
	src := "meta {\r\n  name: crlf\r\n}\r\n"
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m, _ := doc.FindBlock("meta"); m == nil || m.Pairs[0].Value != "crlf" {
		t.Errorf("CRLF parse failed: %+v", m)
	}
}

func TestParseValuesContainColons(t *testing.T) {
	src := `get {
  url: https://example.test:8443/a:b?c=d
}
`
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, _ := doc.FindBlock("get")
	if got := g.Pairs[0].Value; got != "https://example.test:8443/a:b?c=d" {
		t.Errorf("value = %q", got)
	}
}

func TestParseEmptyDictionary(t *testing.T) {
	src := "headers {\n}\n"
	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h, _ := doc.FindBlock("headers")
	if h == nil || len(h.Pairs) != 0 {
		t.Errorf("empty dictionary = %+v", h)
	}
}

func TestParseExtendsBothForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want []string
	}{
		{"colon", "extends: Base\n", []string{"Base"}},
		{"list", "extends [\n  Shared,\n  \"Parent With Spaces\"\n]\n", []string{"Shared", "Parent With Spaces"}},
		{"color", "color: #ff0000\n", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if tc.name == "color" {
				if doc.Color != "#ff0000" {
					t.Errorf("color = %q", doc.Color)
				}
				return
			}
			if len(doc.Extends) != len(tc.want) {
				t.Fatalf("extends = %v, want %v", doc.Extends, tc.want)
			}
			for i := range tc.want {
				if doc.Extends[i] != tc.want[i] {
					t.Errorf("extends[%d] = %q, want %q", i, doc.Extends[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseErrorsCarryPosition(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"unterminated block", "meta {\n  name: x\n"},
		{"missing colon", "meta {\n  name x\n}\n"},
		{"unterminated multiline", "vars {\n  a: '''\n   x\n}\n"},
		{"unterminated list", "tags [\n a\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.src))
			if err == nil {
				t.Fatal("expected an error")
			}
			se, ok := err.(*SyntaxError)
			if !ok {
				t.Fatalf("error is %T, want *SyntaxError", err)
			}
			if se.Pos.Line < 1 {
				t.Errorf("error has no line: %+v", se.Pos)
			}
			if !strings.Contains(err.Error(), ":") {
				t.Errorf("error text lacks a position: %v", err)
			}
		})
	}
}

// TestParseCorpus parses every .bru file in the vendored corpus. It is the
// cheapest broad check that the grammar is right.
func TestParseCorpus(t *testing.T) {
	root := "../../../../testdata/corpus"
	var count int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".bru") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := Parse(data); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if count == 0 {
		t.Skip("no corpus fixtures present")
	}
	t.Logf("parsed %d fixtures", count)
}
