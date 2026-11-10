package bru

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

func TestRequestMethodAndURL(t *testing.T) {
	for block, want := range map[string]string{
		"get": "GET", "post": "POST", "put": "PUT", "delete": "DELETE",
		"patch": "PATCH", "options": "OPTIONS", "head": "HEAD",
	} {
		src := block + " {\n  url: http://x/y\n}\n"
		req, err := ParseRequestBytes(t, src)
		if err != nil {
			t.Fatalf("%s: %v", block, err)
		}
		if req.Method != want {
			t.Errorf("%s → method %q, want %q", block, req.Method, want)
		}
		if req.URL != "http://x/y" {
			t.Errorf("%s → url %q", block, req.URL)
		}
	}
}

func TestRequestCustomMethod(t *testing.T) {
	req, err := ParseRequestBytes(t, "http {\n  method: PURGE\n  url: http://x\n}\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Method != "PURGE" {
		t.Errorf("method = %q, want PURGE", req.Method)
	}
}

func TestRequestDefaultMethodWhenAbsent(t *testing.T) {
	req, err := ParseRequestBytes(t, "meta {\n  name: n\n}\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("method = %q, want the GET default", req.Method)
	}
}

func TestRequestMetaAndTags(t *testing.T) {
	src := `meta {
  name: Get Users
  type: http
  seq: 3
  tags: [
    smoke
    regression
  ]
}
get {
  url: http://x
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Name != "Get Users" {
		t.Errorf("name = %q", req.Name)
	}
	if req.Seq == nil || *req.Seq != 3 {
		t.Errorf("seq = %v", req.Seq)
	}
	if len(req.Tags) != 2 || req.Tags[0] != "smoke" || req.Tags[1] != "regression" {
		t.Errorf("tags = %v", req.Tags)
	}
}

func TestRequestParamsSeparateQueryAndPath(t *testing.T) {
	src := `get {
  url: http://x/:id
}
params:query {
  page: 1
  ~draft: true
}
params:path {
  id: 42
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	query := req.QueryParams()
	if len(query) != 1 {
		t.Fatalf("enabled query params = %d, want 1 (the disabled one is excluded)", len(query))
	}
	if query[0].NameOrEmpty() != "page" || query[0].ValueOrEmpty() != "1" {
		t.Errorf("query = %+v", query[0])
	}
	path := req.PathParams()
	if len(path) != 1 || path[0].NameOrEmpty() != "id" {
		t.Fatalf("path params = %+v", path)
	}
	// Disabled rows must still be present in the model so the editor can
	// re-enable them, exactly as Bruno preserves them.
	if len(req.Params) != 3 {
		t.Errorf("stored params = %d, want 3 including the disabled row", len(req.Params))
	}
}

func TestRequestHeadersDisabledPreserved(t *testing.T) {
	src := `get {
  url: http://x
}
headers {
  content-type: application/json
  ~x-debug: 1
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(req.Headers) != 2 {
		t.Fatalf("headers = %d, want 2", len(req.Headers))
	}
	if got := len(req.LiveHeaders()); got != 1 {
		t.Errorf("live headers = %d, want 1", got)
	}
	if req.Headers[1].Enabled {
		t.Error("the ~ prefix should mark the row disabled")
	}
}

func TestRequestBodyModes(t *testing.T) {
	cases := []struct {
		name string
		src  string
		mode model.BodyMode
		get  func(*model.Body) string
		want string
	}{
		{
			"json",
			"post {\n url: http://x\n}\nbody:json {\n  {\"a\":1}\n}\n",
			model.BodyJSON,
			func(b *model.Body) string { return model.DerefString(b.JSON) },
			`{"a":1}`,
		},
		{
			"text",
			"post {\n url: http://x\n}\nbody:text {\n  hello\n}\n",
			model.BodyText,
			func(b *model.Body) string { return model.DerefString(b.Text) },
			"hello",
		},
		{
			"xml",
			"post {\n url: http://x\n}\nbody:xml {\n  <a/>\n}\n",
			model.BodyXML,
			func(b *model.Body) string { return model.DerefString(b.XML) },
			"<a/>",
		},
		{
			"sparql",
			"post {\n url: http://x\n}\nbody:sparql {\n  SELECT * {}\n}\n",
			model.BodySparql,
			func(b *model.Body) string { return model.DerefString(b.Sparql) },
			"SELECT * {}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := ParseRequestBytes(t, tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if req.Body == nil || req.Body.Mode != tc.mode {
				t.Fatalf("mode = %v, want %v", req.Body, tc.mode)
			}
			if got := tc.get(req.Body); got != tc.want {
				t.Errorf("payload = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequestBodyFormURLEncoded(t *testing.T) {
	src := `post {
  url: http://x
}
body:form-urlencoded {
  a: 1
  ~b: 2
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Body.Mode != model.BodyFormURLEncoded {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	if len(req.Body.FormURLEncoded) != 2 {
		t.Fatalf("fields = %d", len(req.Body.FormURLEncoded))
	}
	if req.Body.FormURLEncoded[1].Enabled {
		t.Error("the ~ row should be disabled")
	}
}

func TestRequestBodyMultipartAnnotations(t *testing.T) {
	src := `post {
  url: http://x
}
body:multipart-form {
  sample: @file(bruno.png) @contentType(image/png)
  foo: {"bar":"baz"} @contentType(application/json--test)
  plain: just text
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Body.Mode != model.BodyMultipartForm {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	parts := req.Body.MultipartForm
	if len(parts) != 3 {
		t.Fatalf("parts = %d", len(parts))
	}

	// File part: the path comes from @file, the type from @contentType.
	if parts[0].Type != model.MultipartFile {
		t.Errorf("part 0 type = %v, want file", parts[0].Type)
	}
	if got := model.DerefString(parts[0].Value); got != "bruno.png" {
		t.Errorf("part 0 path = %q", got)
	}
	if got := model.DerefString(parts[0].ContentType); got != "image/png" {
		t.Errorf("part 0 contentType = %q", got)
	}

	// Text part with a JSON body and a custom type.
	if parts[1].Type != model.MultipartText {
		t.Errorf("part 1 type = %v, want text", parts[1].Type)
	}
	if got := model.DerefString(parts[1].Value); got != `{"bar":"baz"}` {
		t.Errorf("part 1 value = %q (annotations must be stripped)", got)
	}
	if got := model.DerefString(parts[1].ContentType); got != "application/json--test" {
		t.Errorf("part 1 contentType = %q", got)
	}

	// Plain text part, no annotations.
	if parts[2].Type != model.MultipartText || model.DerefString(parts[2].Value) != "just text" {
		t.Errorf("part 2 = %+v", parts[2])
	}
}

func TestRequestBodyGraphQLWithVariables(t *testing.T) {
	src := `post {
  url: http://x
}
body:graphql {
  query { users { id } }
}
body:graphql:vars {
  {"limit": 10}
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Body.Mode != model.BodyGraphQL {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	if req.Body.GraphQL == nil {
		t.Fatal("nil graphql body")
	}
	if got := model.DerefString(req.Body.GraphQL.Query); !strings.Contains(got, "users") {
		t.Errorf("query = %q", got)
	}
	if got := model.DerefString(req.Body.GraphQL.Variables); got != `{"limit": 10}` {
		t.Errorf("variables = %q", got)
	}
}

func TestRequestAuthBearer(t *testing.T) {
	src := `get {
  url: http://x
  auth: bearer
}
auth {
  mode: bearer
}
auth:bearer {
  token: {{tok}}
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Auth == nil || req.Auth.Mode != model.AuthBearer {
		t.Fatalf("auth = %+v", req.Auth)
	}
	if got := model.DerefString(req.Auth.Token); got != "{{tok}}" {
		t.Errorf("token = %q", got)
	}
}

func TestRequestAuthModes(t *testing.T) {
	cases := []struct {
		block string
		mode  model.AuthMode
		check func(*model.Auth) bool
	}{
		{"auth:basic {\n username: u\n password: p\n}", model.AuthBasic,
			func(a *model.Auth) bool {
				return model.DerefString(a.Username) == "u" && model.DerefString(a.Password) == "p"
			}},
		{"auth:apikey {\n key: k\n value: v\n placement: queryparams\n}", model.AuthAPIKey,
			func(a *model.Auth) bool { return a.Placement == model.APIKeyQueryParams }},
		{"auth:ntlm {\n username: u\n domain: d\n}", model.AuthNTLM,
			func(a *model.Auth) bool { return model.DerefString(a.Domain) == "d" }},
		{"auth:awsv4 {\n accessKeyId: ak\n region: eu-west-1\n}", model.AuthAWSV4,
			func(a *model.Auth) bool { return model.DerefString(a.Region) == "eu-west-1" }},
	}
	for _, tc := range cases {
		t.Run(string(tc.mode), func(t *testing.T) {
			src := "auth {\n  mode: " + string(tc.mode) + "\n}\n" + tc.block + "\n"
			req, err := ParseRequestBytes(t, src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if req.Auth == nil || req.Auth.Mode != tc.mode {
				t.Fatalf("auth mode = %+v, want %v", req.Auth, tc.mode)
			}
			if !tc.check(req.Auth) {
				t.Errorf("auth fields not mapped: %+v", req.Auth)
			}
		})
	}
}

func TestRequestAuthAbsentIsNil(t *testing.T) {
	req, err := ParseRequestBytes(t, "get {\n url: http://x\n}\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Auth != nil {
		t.Errorf("auth = %+v, want nil when no auth block exists", req.Auth)
	}
}

func TestRequestScriptsAndTests(t *testing.T) {
	src := `get {
  url: http://x
}
script:pre-request {
  bru.setVar('a', 1);
}
script:post-response {
  console.log(res.getStatus());
}
tests {
  test('ok', () => {});
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Script == nil {
		t.Fatal("nil scripts")
	}
	if got := model.DerefString(req.Script.Req); !strings.Contains(got, "setVar") {
		t.Errorf("pre-request = %q", got)
	}
	if got := model.DerefString(req.Script.Res); !strings.Contains(got, "getStatus") {
		t.Errorf("post-response = %q", got)
	}
	if req.Tests == nil || !strings.Contains(*req.Tests, "test(") {
		t.Errorf("tests = %v", req.Tests)
	}
}

func TestRequestAssertions(t *testing.T) {
	src := `get {
  url: http://x
}
assert {
  res.status: 200
  res.body.items: isArray
  res.responseTime: lt 1000
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(req.Assertions) != 3 {
		t.Fatalf("assertions = %d", len(req.Assertions))
	}
	if req.Assertions[0].NameOrEmpty() != "res.status" || req.Assertions[0].ValueOrEmpty() != "200" {
		t.Errorf("assertion 0 = %+v", req.Assertions[0])
	}
	if req.Assertions[2].ValueOrEmpty() != "lt 1000" {
		t.Errorf("assertion 2 = %+v", req.Assertions[2])
	}
}

func TestRequestVarsPostResponseIsExpression(t *testing.T) {
	src := `get {
  url: http://x
}
vars:post-response {
  token: res.body.access_token
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Vars == nil || len(req.Vars.PostResponse) != 1 {
		t.Fatalf("vars = %+v", req.Vars)
	}
	if got := req.Vars.PostResponse[0].Value.(string); got != "res.body.access_token" {
		t.Errorf("value = %q", got)
	}
}

func TestRequestSettings(t *testing.T) {
	src := `get {
  url: http://x
}
settings {
  encodeUrl: true
  followRedirects: false
  maxRedirects: 10
  timeout: 5000
  forwardAuthorizationHeader: false
}
`
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	s := req.Settings
	if !s.EncodesURL() {
		t.Error("encodeUrl not applied")
	}
	if s.FollowingRedirects() {
		t.Error("followRedirects not applied")
	}
	if s.MaxRedirectCount() != 10 {
		t.Errorf("maxRedirects = %d", s.MaxRedirectCount())
	}
	if s.Timeout == nil || s.Timeout.Inherit || s.Timeout.Millis != 5000 {
		t.Errorf("timeout = %+v", s.Timeout)
	}
	if s.ForwardsAuthHeader() {
		t.Error("forwardAuthorizationHeader not applied")
	}
}

func TestRequestSettingsBlankMaxRedirectsMeansUnset(t *testing.T) {
	src := "get {\n url: http://x\n}\nsettings {\n  maxRedirects:\n}\n"
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Settings.MaxRedirects != nil {
		t.Errorf("blank maxRedirects should stay unset, got %d", *req.Settings.MaxRedirects)
	}
	if got := req.Settings.MaxRedirectCount(); got != model.DefaultMaxRedirects {
		t.Errorf("default maxRedirects = %d", got)
	}
}

func TestRequestSettingsTimeoutInherit(t *testing.T) {
	src := "get {\n url: http://x\n}\nsettings {\n  timeout: inherit\n}\n"
	req, err := ParseRequestBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Settings.Timeout == nil || !req.Settings.Timeout.Inherit {
		t.Errorf("timeout = %+v, want inherit", req.Settings.Timeout)
	}
}

func TestFolderParsing(t *testing.T) {
	src := `meta {
  name: digest
  seq: 14
}
auth {
  mode: inherit
}
headers {
  x-shared: 1
}
`
	item, err := ParseFolderBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if item.Type != model.ItemFolder || item.Name != "digest" {
		t.Errorf("folder = %+v", item)
	}
	if item.Seq == nil || *item.Seq != 14 {
		t.Errorf("seq = %v", item.Seq)
	}
	if item.Auth == nil || item.Auth.Mode != model.AuthInherit {
		t.Errorf("auth = %+v", item.Auth)
	}
	if len(item.Headers) != 1 {
		t.Errorf("headers = %d", len(item.Headers))
	}
}

func TestEnvironmentParsing(t *testing.T) {
	src := `vars {
  host: http://localhost:8080
  @number
  retries: 3
  multiline: '''
    line1
    line2
  '''
}
vars:secret [
  clientSecret
]
extends: Base-Environment
color: #ff0000
`
	env, err := ParseEnvironmentBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.Extends != "Base-Environment" {
		t.Errorf("extends = %q", env.Extends)
	}
	if env.Color != "#ff0000" {
		t.Errorf("color = %q", env.Color)
	}
	if v, ok := env.VariableValue("host"); !ok || v != "http://localhost:8080" {
		t.Errorf("host = %v %v", v, ok)
	}
	// Typed variable.
	var retries *model.Variable
	for i := range env.Variables {
		if env.Variables[i].NameOrEmpty() == "retries" {
			retries = &env.Variables[i]
		}
	}
	if retries == nil || retries.DataType != "number" {
		t.Errorf("retries variable = %+v", retries)
	}
	// Secret marked.
	if len(env.Secrets) != 1 || env.Secrets[0] != "clientSecret" {
		t.Errorf("secrets = %v", env.Secrets)
	}
	for _, v := range env.Variables {
		if v.NameOrEmpty() == "clientSecret" && !v.Secret {
			t.Error("clientSecret should be flagged as secret")
		}
	}
}

func TestEnvironmentExtendsList(t *testing.T) {
	src := "vars {\n  a: 1\n}\nextends [\n  Shared,\n  \"Parent With Spaces\"\n]\n"
	env, err := ParseEnvironmentBytes(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.Extends != "Shared" {
		t.Errorf("extends = %q, want the first parent", env.Extends)
	}
}

// TestMapperOverUpstreamCorpus maps every v2 fixture in the Bruno clone.
func TestMapperOverUpstreamCorpus(t *testing.T) {
	root := "../../../../bruno/packages"
	var n, failed int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".bru") || strings.Contains(path, "node_modules") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || isV1Dialect(data) {
			return nil
		}
		n++
		doc, perr := Parse(data)
		if perr != nil {
			failed++
			return nil
		}
		if _, merr := doc.Request(); merr != nil {
			failed++
			t.Logf("MAP FAIL %s: %v", path, merr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	t.Logf("mapped %d v2 files, %d failures", n, failed)
	if n == 0 {
		t.Skip("upstream clone not present")
	}
}

// ParseRequestBytes parses a request from a string literal.
func ParseRequestBytes(t *testing.T, src string) (*model.Request, error) {
	t.Helper()
	doc, err := Parse([]byte(src))
	if err != nil {
		return nil, err
	}
	return doc.Request()
}

// ParseFolderBytes parses a folder file from a string literal.
func ParseFolderBytes(t *testing.T, src string) (*model.Item, error) {
	t.Helper()
	return (&Codec{}).ParseFolder([]byte(src))
}

// ParseEnvironmentBytes parses an environment file from a string literal.
func ParseEnvironmentBytes(t *testing.T, src string) (*model.Environment, error) {
	t.Helper()
	return (&Codec{}).ParseEnvironment([]byte(src))
}
