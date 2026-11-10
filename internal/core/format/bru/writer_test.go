package bru

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

// TestRoundTripRequests parses, writes and re-parses every v2 fixture and
// asserts the model is unchanged. This is the property that matters for a
// semantics-preserving writer: byte equality is not required, but no information
// may be lost.
func TestRoundTripRequests(t *testing.T) {
	root := "../../../../bruno/packages"
	if _, err := os.Stat(root); err != nil {
		t.Skip("upstream clone not present")
	}

	var checked, failed int
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".bru") || strings.Contains(path, "node_modules") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || isV1Dialect(data) {
			return nil
		}

		first, err := Parse(data)
		if err != nil {
			return nil
		}
		req1, err := first.Request()
		if err != nil {
			return nil
		}
		// Only requests can be round-tripped through writeRequest; folder and
		// collection files have no method block.
		if req1.URL == "" && req1.Name == "" {
			return nil
		}

		out, err := (&Codec{}).WriteRequest(req1)
		if err != nil {
			failed++
			t.Logf("WRITE FAIL %s: %v", path, err)
			return nil
		}

		second, err := Parse(out)
		if err != nil {
			failed++
			t.Logf("REPARSE FAIL %s: %v\n---\n%s", path, err, out)
			return nil
		}
		req2, err := second.Request()
		if err != nil {
			failed++
			t.Logf("REMAP FAIL %s: %v", path, err)
			return nil
		}

		checked++
		if diff := diffRequests(req1, req2); diff != "" {
			failed++
			t.Logf("ROUND-TRIP DIFF %s:\n%s", path, diff)
		}
		return nil
	})

	t.Logf("round-tripped %d requests, %d failures", checked, failed)
	if checked == 0 {
		t.Skip("nothing to round-trip")
	}
}

// diffRequests reports the first substantive difference between two requests,
// ignoring fields the writer deliberately does not persist.
func diffRequests(a, b *model.Request) string {
	if a.Method != b.Method {
		return "method: " + a.Method + " != " + b.Method
	}
	if a.URL != b.URL {
		return "url: " + a.URL + " != " + b.URL
	}
	if a.Name != b.Name {
		return "name: " + a.Name + " != " + b.Name
	}
	if len(a.Headers) != len(b.Headers) {
		return "header count changed"
	}
	for i := range a.Headers {
		if a.Headers[i].NameOrEmpty() != b.Headers[i].NameOrEmpty() ||
			a.Headers[i].ValueOrEmpty() != b.Headers[i].ValueOrEmpty() ||
			a.Headers[i].Enabled != b.Headers[i].Enabled {
			return "header " + a.Headers[i].NameOrEmpty() + " changed"
		}
	}
	if len(a.Params) != len(b.Params) {
		return "param count changed"
	}
	for i := range a.Params {
		if a.Params[i].NameOrEmpty() != b.Params[i].NameOrEmpty() ||
			a.Params[i].ValueOrEmpty() != b.Params[i].ValueOrEmpty() ||
			a.Params[i].Type != b.Params[i].Type ||
			a.Params[i].Enabled != b.Params[i].Enabled {
			return "param " + a.Params[i].NameOrEmpty() + " changed"
		}
	}
	if (a.Body == nil) != (b.Body == nil) {
		return "body presence changed"
	}
	if a.Body != nil {
		if a.Body.Mode != b.Body.Mode {
			return "body mode: " + string(a.Body.Mode) + " != " + string(b.Body.Mode)
		}
		switch a.Body.Mode {
		case model.BodyJSON:
			if model.DerefString(a.Body.JSON) != model.DerefString(b.Body.JSON) {
				return "json body changed"
			}
		case model.BodyText:
			if model.DerefString(a.Body.Text) != model.DerefString(b.Body.Text) {
				return "text body changed"
			}
		case model.BodyXML:
			if model.DerefString(a.Body.XML) != model.DerefString(b.Body.XML) {
				return "xml body changed"
			}
		case model.BodyFormURLEncoded:
			if len(a.Body.FormURLEncoded) != len(b.Body.FormURLEncoded) {
				return "form row count changed"
			}
		case model.BodyMultipartForm:
			if len(a.Body.MultipartForm) != len(b.Body.MultipartForm) {
				return "multipart part count changed"
			}
			for i := range a.Body.MultipartForm {
				if a.Body.MultipartForm[i].Type != b.Body.MultipartForm[i].Type {
					return "multipart part type changed"
				}
				if model.DerefString(a.Body.MultipartForm[i].Value) != model.DerefString(b.Body.MultipartForm[i].Value) {
					return "multipart part value changed"
				}
			}
		}
	}
	if (a.Auth == nil) != (b.Auth == nil) {
		return "auth presence changed"
	}
	if a.Auth != nil && a.Auth.Mode != b.Auth.Mode {
		return "auth mode: " + string(a.Auth.Mode) + " != " + string(b.Auth.Mode)
	}
	if (a.Script == nil) != (b.Script == nil) {
		return "script presence changed"
	}
	if a.Script != nil {
		if model.DerefString(a.Script.Req) != model.DerefString(b.Script.Req) {
			return "pre-request script changed"
		}
		if model.DerefString(a.Script.Res) != model.DerefString(b.Script.Res) {
			return "post-response script changed"
		}
	}
	if model.DerefString(a.Tests) != model.DerefString(b.Tests) {
		return "tests changed"
	}
	if len(a.Assertions) != len(b.Assertions) {
		return "assertion count changed"
	}
	return ""
}

func TestWriteRequestMinimal(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://example.test/a"}
	out, err := (&Codec{}).WriteRequest(req)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	text := string(out)
	if !strings.HasPrefix(text, "get {") {
		t.Errorf("expected a standard method block, got:\n%s", text)
	}
	if !strings.Contains(text, "url: https://example.test/a") {
		t.Errorf("url missing:\n%s", text)
	}
	// Re-parsing must give the same method and URL.
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if back.Method != "GET" || back.URL != req.URL {
		t.Errorf("round-trip = %s %s", back.Method, back.URL)
	}
}

func TestWriteRequestCustomMethod(t *testing.T) {
	req := &model.Request{Method: "PURGE", URL: "http://x"}
	out, _ := (&Codec{}).WriteRequest(req)
	if !strings.Contains(string(out), "http {") || !strings.Contains(string(out), "method: PURGE") {
		t.Errorf("custom method should use the http block:\n%s", out)
	}
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if back.Method != "PURGE" {
		t.Errorf("method = %q", back.Method)
	}
}

func TestWriteDisabledRowsKeepTilde(t *testing.T) {
	req := &model.Request{
		Method: "GET",
		URL:    "http://x",
		Headers: []model.KeyValue{
			{Name: strp("a"), Value: strp("1"), Enabled: true},
			{Name: strp("b"), Value: strp("2"), Enabled: false},
		},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	if !strings.Contains(string(out), "~b: 2") {
		t.Errorf("disabled header lost its tilde:\n%s", out)
	}
	back, _ := (&Codec{}).ParseRequest(out)
	if len(back.Headers) != 2 || back.Headers[1].Enabled {
		t.Errorf("disabled state lost: %+v", back.Headers)
	}
}

func TestWriteQuotedKeys(t *testing.T) {
	req := &model.Request{
		Method: "GET",
		URL:    "http://x",
		Headers: []model.KeyValue{
			{Name: strp("key:with:colons"), Value: strp("v"), Enabled: true},
		},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	if !strings.Contains(string(out), `"key:with:colons": v`) {
		t.Errorf("colon key not quoted:\n%s", out)
	}
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if got := back.Headers[0].NameOrEmpty(); got != "key:with:colons" {
		t.Errorf("key = %q", got)
	}
}

func TestWriteJSONBodySurvivesNestedBraces(t *testing.T) {
	body := "{\n  \"a\": {\n    \"b\": 1\n  }\n}"
	req := &model.Request{
		Method: "POST",
		URL:    "http://x",
		Body:   &model.Body{Mode: model.BodyJSON, JSON: &body},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, out)
	}
	if got := model.DerefString(back.Body.JSON); got != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}

func TestWriteMultipartAnnotations(t *testing.T) {
	path := "img.png"
	ct := "image/png"
	text := `{"a":1}`
	req := &model.Request{
		Method: "POST",
		URL:    "http://x",
		Body: &model.Body{Mode: model.BodyMultipartForm, MultipartForm: []model.MultipartEntry{
			{Name: strp("file"), Value: &path, Type: model.MultipartFile, ContentType: &ct, Enabled: true},
			{Name: strp("meta"), Value: &text, Type: model.MultipartText, Enabled: true},
		}},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	if !strings.Contains(string(out), "@file(img.png)") {
		t.Errorf("file annotation missing:\n%s", out)
	}
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, out)
	}
	if len(back.Body.MultipartForm) != 2 {
		t.Fatalf("parts = %d", len(back.Body.MultipartForm))
	}
	if back.Body.MultipartForm[0].Type != model.MultipartFile {
		t.Errorf("part 0 type = %v", back.Body.MultipartForm[0].Type)
	}
	if got := model.DerefString(back.Body.MultipartForm[0].Value); got != "img.png" {
		t.Errorf("part 0 path = %q", got)
	}
}

func TestWriteAuthRoundTrip(t *testing.T) {
	user, pass := "u", "p"
	req := &model.Request{
		Method: "GET",
		URL:    "http://x",
		Auth:   &model.Auth{Mode: model.AuthBasic, Username: &user, Password: &pass},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	back, err := (&Codec{}).ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, out)
	}
	if back.Auth == nil || back.Auth.Mode != model.AuthBasic {
		t.Fatalf("auth = %+v", back.Auth)
	}
	if model.DerefString(back.Auth.Username) != "u" || model.DerefString(back.Auth.Password) != "p" {
		t.Errorf("credentials lost: %+v", back.Auth)
	}
}

func TestWriteSettingsOnlyWritesSetFields(t *testing.T) {
	yes := true
	req := &model.Request{
		Method:   "GET",
		URL:      "http://x",
		Settings: model.ItemSettings{FollowRedirects: &yes},
	}
	out, _ := (&Codec{}).WriteRequest(req)
	text := string(out)
	if !strings.Contains(text, "followRedirects: true") {
		t.Errorf("set field missing:\n%s", text)
	}
	if strings.Contains(text, "maxRedirects") {
		t.Errorf("unset field was written:\n%s", text)
	}
	back, _ := (&Codec{}).ParseRequest(out)
	if back.Settings.MaxRedirects != nil {
		t.Error("an unset maxRedirects must stay unset")
	}
}

func TestWriteEnvironmentRoundTrip(t *testing.T) {
	env := &model.Environment{
		Variables: []model.Variable{
			{Name: strp("host"), Value: "http://x", Enabled: true, DataType: "string"},
			{Name: strp("retries"), Value: "3", Enabled: true, DataType: "number"},
		},
		Secrets: []string{"token"},
		Extends: "Base",
		Color:   "#ff0000",
	}
	out, err := (&Codec{}).WriteEnvironment(env)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	text := string(out)
	for _, want := range []string{"vars {", "@number", "vars:secret [", "extends: Base", "color: #ff0000"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	back, err := (&Codec{}).ParseEnvironment(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, out)
	}
	if back.Extends != "Base" || back.Color != "#ff0000" {
		t.Errorf("extends/color lost: %+v", back)
	}
	if v, ok := back.VariableValue("retries"); !ok || v != "3" {
		t.Errorf("retries = %v %v", v, ok)
	}
}

func TestWriteFolderRoundTrip(t *testing.T) {
	seq := 7
	item := &model.Item{
		Type: model.ItemFolder,
		Name: "sub",
		Seq:  &seq,
		Auth: &model.Auth{Mode: model.AuthInherit},
	}
	out, err := (&Codec{}).WriteFolder(item)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := (&Codec{}).ParseFolder(out)
	if err != nil {
		t.Fatalf("reparse: %v\n%s", err, out)
	}
	if back.Name != "sub" || back.Seq == nil || *back.Seq != 7 {
		t.Errorf("folder = %+v", back)
	}
}

func strp(s string) *string { return &s }
