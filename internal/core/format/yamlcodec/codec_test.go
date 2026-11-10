package yamlcodec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

func parseReq(t *testing.T, src string) *model.Request {
	t.Helper()
	req, err := New().ParseRequest([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return req
}

func TestParseMinimalRequest(t *testing.T) {
	src := `info:
  name: request_1
  type: http
  seq: 1

http:
  method: GET
  url: https://api.example.com/one
`
	req := parseReq(t, src)
	if req.Name != "request_1" {
		t.Errorf("name = %q", req.Name)
	}
	if req.Method != "GET" || req.URL != "https://api.example.com/one" {
		t.Errorf("method/url = %s %s", req.Method, req.URL)
	}
	if req.Seq == nil || *req.Seq != 1 {
		t.Errorf("seq = %v", req.Seq)
	}
}

func TestParseJSONBody(t *testing.T) {
	src := `info:
  name: Create User
  type: http
  seq: 1

http:
  method: POST
  url: https://api.example.com/users
  body:
    mode: json
    json: |
      {
        "name": "John Doe"
      }
`
	req := parseReq(t, src)
	if req.Body == nil || req.Body.Mode != model.BodyJSON {
		t.Fatalf("body = %+v", req.Body)
	}
	body := model.DerefString(req.Body.JSON)
	if !strings.Contains(body, `"name": "John Doe"`) {
		t.Errorf("json = %q", body)
	}
	// The YAML block scalar keeps its trailing newline; that is content, not noise.
	if !strings.HasSuffix(body, "\n") {
		t.Errorf("block scalar newline lost: %q", body)
	}
}

func TestParseFormURLEncoded(t *testing.T) {
	src := `info:
  name: form
http:
  method: POST
  url: '{{host}}'
  body:
    type: form-urlencoded
    data:
      - name: a
        value: '1'
      - name: b
        value: '2'
        disabled: true
`
	req := parseReq(t, src)
	if req.Body.Mode != model.BodyFormURLEncoded {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	if len(req.Body.FormURLEncoded) != 2 {
		t.Fatalf("fields = %d", len(req.Body.FormURLEncoded))
	}
	if req.Body.FormURLEncoded[0].NameOrEmpty() != "a" {
		t.Errorf("field 0 = %+v", req.Body.FormURLEncoded[0])
	}
	if req.Body.FormURLEncoded[1].Enabled {
		t.Error("disabled flag lost")
	}
}

func TestParseMultipart(t *testing.T) {
	src := `info:
  name: multipart
http:
  method: POST
  url: x
  body:
    type: multipart-form
    data:
      - name: mp_str
        type: text
        value: '{{mp_str}}'
      - name: upload
        type: file
        value: ./a.png
        contentType: image/png
`
	req := parseReq(t, src)
	if req.Body.Mode != model.BodyMultipartForm {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	if len(req.Body.MultipartForm) != 2 {
		t.Fatalf("parts = %d", len(req.Body.MultipartForm))
	}
	if req.Body.MultipartForm[0].Type != model.MultipartText {
		t.Errorf("part 0 type = %v", req.Body.MultipartForm[0].Type)
	}
	if req.Body.MultipartForm[1].Type != model.MultipartFile {
		t.Errorf("part 1 type = %v", req.Body.MultipartForm[1].Type)
	}
	if got := model.DerefString(req.Body.MultipartForm[1].ContentType); got != "image/png" {
		t.Errorf("contentType = %q", got)
	}
}

func TestParseAuthDigest(t *testing.T) {
	src := `info:
  name: request-level
http:
  method: GET
  url: https://x
  auth:
    type: digest
    username: "user"
    password: "{{digestPw}}"
`
	req := parseReq(t, src)
	if req.Auth == nil || req.Auth.Mode != model.AuthDigest {
		t.Fatalf("auth = %+v", req.Auth)
	}
	if model.DerefString(req.Auth.Username) != "user" {
		t.Errorf("username = %q", model.DerefString(req.Auth.Username))
	}
	if model.DerefString(req.Auth.Password) != "{{digestPw}}" {
		t.Errorf("password = %q", model.DerefString(req.Auth.Password))
	}
}

func TestParseTypedRuntimeVariables(t *testing.T) {
	src := `info:
  name: v
runtime:
  variables:
    - name: s
      value: plain
    - name: n
      value:
        type: number
        data: "42"
    - name: b
      value:
        type: boolean
        data: "true"
  scripts:
    - type: tests
      code: |-
        test('x', () => {});
`
	req := parseReq(t, src)
	if req.Vars == nil || len(req.Vars.PreRequest) != 3 {
		t.Fatalf("vars = %+v", req.Vars)
	}
	if got := req.Vars.PreRequest[1]; got.DataType != "number" || got.Value != "42" {
		t.Errorf("typed var = %+v", got)
	}
	if got := req.Vars.PreRequest[2]; got.DataType != "boolean" {
		t.Errorf("typed var = %+v", got)
	}
	// `tests` runtime scripts map onto the Tests field, not onto a script hook.
	if req.Tests == nil || !strings.Contains(*req.Tests, "test('x'") {
		t.Errorf("tests = %v", req.Tests)
	}
}

func TestParseScripts(t *testing.T) {
	src := `info:
  name: s
runtime:
  scripts:
    - type: pre-request
      code: bru.setVar('a', 1);
    - type: post-response
      code: console.log(res.getStatus());
`
	req := parseReq(t, src)
	if req.Script == nil {
		t.Fatal("nil scripts")
	}
	if got := model.DerefString(req.Script.Req); !strings.Contains(got, "setVar") {
		t.Errorf("pre = %q", got)
	}
	if got := model.DerefString(req.Script.Res); !strings.Contains(got, "getStatus") {
		t.Errorf("post = %q", got)
	}
	if req.Tests != nil {
		t.Error("a post-response script must not become the tests block")
	}
}

func TestParseSettings(t *testing.T) {
	src := `info:
  name: s
settings:
  encodeUrl: true
  timeout: 0
  followRedirects: false
  maxRedirects: 3
`
	req := parseReq(t, src)
	if !req.Settings.EncodesURL() {
		t.Error("encodeUrl not applied")
	}
	if req.Settings.FollowingRedirects() {
		t.Error("followRedirects not applied")
	}
	if req.Settings.MaxRedirectCount() != 3 {
		t.Errorf("maxRedirects = %d", req.Settings.MaxRedirectCount())
	}
	if req.Settings.Timeout == nil || req.Settings.Timeout.Millis != 0 {
		t.Errorf("timeout = %+v", req.Settings.Timeout)
	}
}

func TestParseHeadersAndParams(t *testing.T) {
	src := `info:
  name: h
http:
  method: GET
  url: x
  headers:
    - name: content-type
      value: application/json
    - name: x-debug
      value: '1'
      disabled: true
  params:
    - name: q
      value: search
      type: query
    - name: id
      value: '7'
      type: path
`
	req := parseReq(t, src)
	if len(req.Headers) != 2 {
		t.Fatalf("headers = %d", len(req.Headers))
	}
	if req.Headers[1].Enabled {
		t.Error("disabled header lost its flag")
	}
	if len(req.QueryParams()) != 1 {
		t.Errorf("query params = %d", len(req.QueryParams()))
	}
	if len(req.PathParams()) != 1 {
		t.Errorf("path params = %d", len(req.PathParams()))
	}
}

func TestParseFolder(t *testing.T) {
	src := `info:
  name: folder_1
  type: folder
  seq: 1
`
	item, err := New().ParseFolder([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if item.Type != model.ItemFolder || item.Name != "folder_1" {
		t.Errorf("folder = %+v", item)
	}
	if item.Seq == nil || *item.Seq != 1 {
		t.Errorf("seq = %v", item.Seq)
	}
}

func TestParseCollection(t *testing.T) {
	src := `opencollection: "1.0.0"
info:
  name: ymlcollection
  version: "3.4.0"
`
	cfg, err := New().ParseCollection([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Name != "ymlcollection" || cfg.Version != "1.0.0" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.Format != model.FormatYML {
		t.Errorf("format = %v", cfg.Format)
	}
	if cfg.Info == nil || cfg.Info.Version != "3.4.0" {
		t.Errorf("info = %+v", cfg.Info)
	}
}

func TestParseEnvironment(t *testing.T) {
	src := `name: Local
variables:
  - name: host
    value: http://localhost:8080
  - name: retries
    value:
      type: number
      data: "3"
secrets:
  - token
extends: Base
color: "#ff0000"
`
	env, err := New().ParseEnvironment([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.Name != "Local" || env.Extends != "Base" || env.Color != "#ff0000" {
		t.Errorf("env = %+v", env)
	}
	if v, ok := env.VariableValue("host"); !ok || v != "http://localhost:8080" {
		t.Errorf("host = %v %v", v, ok)
	}
	if len(env.Secrets) != 1 || env.Secrets[0] != "token" {
		t.Errorf("secrets = %v", env.Secrets)
	}
	for _, v := range env.Variables {
		if v.NameOrEmpty() == "token" && !v.Secret {
			t.Error("secret variable not flagged")
		}
	}
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	if ok, err := New().Detect(dir); err != nil || ok {
		t.Fatalf("empty dir: ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencollection.yml"), []byte("opencollection: 1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := New().Detect(dir); err != nil || !ok {
		t.Fatalf("with marker: ok=%v err=%v", ok, err)
	}
}

// TestWriteRoundTrip checks model -> YAML -> model for the fields that matter.
func TestWriteRoundTrip(t *testing.T) {
	jsonBody := "{\n  \"a\": 1\n}\n"
	user, pass := "u", "p"
	seq := 2
	req := &model.Request{
		Name:   "Create User",
		Method: "POST",
		URL:    "https://api.example.com/users",
		Seq:    &seq,
		Headers: []model.KeyValue{
			{Name: strPtr("content-type"), Value: strPtr("application/json"), Enabled: true},
			{Name: strPtr("x-off"), Value: strPtr("1"), Enabled: false},
		},
		Body: &model.Body{Mode: model.BodyJSON, JSON: &jsonBody},
		Auth: &model.Auth{Mode: model.AuthBasic, Username: &user, Password: &pass},
		Vars: &model.VarBlocks{PreRequest: []model.Variable{
			{Name: strPtr("host"), Value: "http://x", Enabled: true, DataType: "string"},
		}},
	}

	out, err := New().WriteRequest(req)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := New().ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}

	if back.Method != req.Method || back.URL != req.URL || back.Name != req.Name {
		t.Errorf("identity lost: %+v", back)
	}
	if len(back.Headers) != 2 || back.Headers[1].Enabled {
		t.Errorf("headers = %+v", back.Headers)
	}
	if got := model.DerefString(back.Body.JSON); got != jsonBody {
		t.Errorf("body = %q, want %q", got, jsonBody)
	}
	if back.Auth == nil || back.Auth.Mode != model.AuthBasic {
		t.Errorf("auth = %+v", back.Auth)
	}
	if back.Vars == nil || len(back.Vars.PreRequest) != 1 {
		t.Errorf("vars = %+v", back.Vars)
	}
}

func TestWriteTypedVariableRoundTrip(t *testing.T) {
	env := &model.Environment{
		Name: "E",
		Variables: []model.Variable{
			{Name: strPtr("n"), Value: "42", Enabled: true, DataType: "number"},
			{Name: strPtr("s"), Value: "text", Enabled: true, DataType: "string"},
		},
	}
	out, err := New().WriteEnvironment(env)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := New().ParseEnvironment(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}
	if len(back.Variables) != 2 {
		t.Fatalf("vars = %d", len(back.Variables))
	}
	if back.Variables[0].DataType != "number" || back.Variables[0].Value != "42" {
		t.Errorf("typed var = %+v", back.Variables[0])
	}
	if back.Variables[1].DataType != "string" {
		t.Errorf("plain var = %+v", back.Variables[1])
	}
}

// TestParseUpstreamYMLCorpus parses every checked-in .yml fixture.
func TestParseUpstreamYMLCorpus(t *testing.T) {
	root := "../../../../bruno/packages"
	if _, err := os.Stat(root); err != nil {
		t.Skip("upstream clone not present")
	}
	var n, failed int
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yml") || strings.Contains(path, "node_modules") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		n++
		switch base {
		case "opencollection.yml":
			if _, err := New().ParseCollection(data); err != nil {
				failed++
				t.Logf("collection FAIL %s: %v", path, err)
			}
		case "folder.yml":
			if _, err := New().ParseFolder(data); err != nil {
				failed++
				t.Logf("folder FAIL %s: %v", path, err)
			}
		default:
			if _, err := New().ParseRequest(data); err != nil {
				failed++
				t.Logf("request FAIL %s: %v", path, err)
			}
		}
		return nil
	})
	t.Logf("parsed %d yml files, %d failures", n, failed)
	if n == 0 {
		t.Skip("no yml fixtures")
	}
}
