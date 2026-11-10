package yamlcodec

import (
	"strings"
	"testing"

	"fearless/internal/core/model"
)

func TestCodecName(t *testing.T) {
	if got := New().Name(); got != model.FormatYML {
		t.Errorf("Name = %v, want yml", got)
	}
}

func TestRegisterAndGet(t *testing.T) {
	// Register would panic if called twice; exercise the codec identity instead.
	if FormatVersion() == "" {
		t.Error("FormatVersion is empty")
	}
}

func TestWriteCollectionRoundTrip(t *testing.T) {
	cfg := &model.CollectionConfig{
		Name:    "my-collection",
		Version: "1.0.0",
		Info:    &model.CollectionInfo{Version: "3.4.0", Description: "desc"},
		Vars: []model.Variable{
			{Name: strPtr("baseUrl"), Value: "http://x", Enabled: true, DataType: "string"},
		},
	}
	out, err := New().WriteCollection(cfg)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "opencollection:") {
		t.Errorf("missing opencollection key:\n%s", text)
	}
	if !strings.Contains(text, "my-collection") {
		t.Errorf("missing name:\n%s", text)
	}

	back, err := New().ParseCollection(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if back.Name != cfg.Name {
		t.Errorf("name = %q", back.Name)
	}
	if back.Info == nil || back.Info.Version != "3.4.0" {
		t.Errorf("info = %+v", back.Info)
	}
	if len(back.Vars) != 1 || back.Vars[0].NameOrEmpty() != "baseUrl" {
		t.Errorf("vars = %+v", back.Vars)
	}
}

func TestWriteCollectionDefaultsVersion(t *testing.T) {
	out, err := New().WriteCollection(&model.CollectionConfig{Name: "c"})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(string(out), "1.0.0") {
		t.Errorf("expected a default version:\n%s", out)
	}
}

func TestWriteAdminErrors(t *testing.T) {
	var c Codec
	if _, err := c.WriteCollection(nil); err == nil {
		t.Error("nil collection should error")
	}
	if _, err := c.WriteRequest(nil); err == nil {
		t.Error("nil request should error")
	}
	if _, err := c.WriteFolder(nil); err == nil {
		t.Error("nil folder should error")
	}
	if _, err := c.WriteEnvironment(nil); err == nil {
		t.Error("nil environment should error")
	}
}

func TestBodyRoundTripAllModes(t *testing.T) {
	cases := []struct {
		name string
		body *model.Body
		get  func(*model.Body) string
		want string
	}{
		{"json", &model.Body{Mode: model.BodyJSON, JSON: strPtr(`{"a":1}`)},
			func(b *model.Body) string { return model.DerefString(b.JSON) }, `{"a":1}`},
		{"text", &model.Body{Mode: model.BodyText, Text: strPtr("hello")},
			func(b *model.Body) string { return model.DerefString(b.Text) }, "hello"},
		{"xml", &model.Body{Mode: model.BodyXML, XML: strPtr("<a/>")},
			func(b *model.Body) string { return model.DerefString(b.XML) }, "<a/>"},
		{"sparql", &model.Body{Mode: model.BodySparql, Sparql: strPtr("SELECT {}")},
			func(b *model.Body) string { return model.DerefString(b.Sparql) }, "SELECT {}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &model.Request{Method: "POST", URL: "http://x", Body: tc.body}
			out, err := New().WriteRequest(req)
			if err != nil {
				t.Fatalf("write: %v", err)
			}
			back, err := New().ParseRequest(out)
			if err != nil {
				t.Fatalf("reparse:\n%s\n%v", out, err)
			}
			if back.Body.Mode != tc.body.Mode {
				t.Fatalf("mode = %v, want %v", back.Body.Mode, tc.body.Mode)
			}
			if got := tc.get(back.Body); got != tc.want {
				t.Errorf("payload = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGraphQLRoundTrip(t *testing.T) {
	req := &model.Request{
		Method: "POST", URL: "http://x",
		Body: &model.Body{Mode: model.BodyGraphQL, GraphQL: &model.GraphQLBody{
			Query:     strPtr("query { users { id } }"),
			Variables: strPtr(`{"limit":10}`),
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
	if back.Body.Mode != model.BodyGraphQL {
		t.Fatalf("mode = %v", back.Body.Mode)
	}
	q := model.DerefString(back.Body.GraphQL.Query)
	if !strings.Contains(q, "users") {
		t.Errorf("query = %q", q)
	}
	if got := model.DerefString(back.Body.GraphQL.Variables); got != `{"limit":10}` {
		t.Errorf("variables = %q", got)
	}
}

func TestGraphQLFromDataTypeShape(t *testing.T) {
	// shape 2: type: graphql with a scalar data payload
	src := `info:
  name: g
http:
  method: POST
  url: x
  body:
    type: graphql
    data: query { users { id } }
`
	req := parseReq(t, src)
	if req.Body.Mode != model.BodyGraphQL {
		t.Fatalf("mode = %v", req.Body.Mode)
	}
	if got := model.DerefString(req.Body.GraphQL.Query); !strings.Contains(got, "users") {
		t.Errorf("query = %q", got)
	}
}

func TestFormAndMultipartRoundTrip(t *testing.T) {
	req := &model.Request{
		Method: "POST", URL: "http://x",
		Body: &model.Body{Mode: model.BodyFormURLEncoded, FormURLEncoded: []model.KeyValue{
			{Name: strPtr("a"), Value: strPtr("1"), Enabled: true},
			{Name: strPtr("b"), Value: strPtr("2"), Enabled: false},
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
	if len(back.Body.FormURLEncoded) != 2 {
		t.Fatalf("fields = %d", len(back.Body.FormURLEncoded))
	}
	if back.Body.FormURLEncoded[1].Enabled {
		t.Error("disabled flag lost")
	}

	req.Body = &model.Body{Mode: model.BodyMultipartForm, MultipartForm: []model.MultipartEntry{
		{Name: strPtr("f"), Value: strPtr("./a.png"), Type: model.MultipartFile, ContentType: strPtr("image/png"), Enabled: true},
		{Name: strPtr("t"), Value: strPtr("text"), Type: model.MultipartText, Enabled: true},
	}}
	out, err = New().WriteRequest(req)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err = New().ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}
	if len(back.Body.MultipartForm) != 2 {
		t.Fatalf("parts = %d", len(back.Body.MultipartForm))
	}
	if back.Body.MultipartForm[0].Type != model.MultipartFile {
		t.Errorf("part 0 type = %v", back.Body.MultipartForm[0].Type)
	}
	if got := model.DerefString(back.Body.MultipartForm[0].ContentType); got != "image/png" {
		t.Errorf("contentType = %q", got)
	}
}

func TestFileBodyRoundTrip(t *testing.T) {
	req := &model.Request{
		Method: "POST", URL: "http://x",
		Body: &model.Body{Mode: model.BodyFile, File: []model.FileEntry{
			{FilePath: strPtr("./data.bin"), Selected: true},
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
	if back.Body.Mode != model.BodyFile || len(back.Body.File) != 1 {
		t.Fatalf("body = %+v", back.Body)
	}
	if got := model.DerefString(back.Body.File[0].FilePath); got != "./data.bin" {
		t.Errorf("path = %q", got)
	}
}

func TestSettingsRoundTripAllFields(t *testing.T) {
	yes, no := true, false
	n := 4
	timeout := model.TimeoutMillis(2500)
	omit := []string{"x-a", "x-b"}
	req := &model.Request{
		Method: "GET", URL: "http://x",
		Settings: model.ItemSettings{
			EncodeURL:                  &yes,
			FollowRedirects:            &no,
			MaxRedirects:               &n,
			Timeout:                    &timeout,
			ForwardAuthorizationHeader: &yes,
			OmitHeaders:                omit,
		},
	}
	out, err := New().WriteRequest(req)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := New().ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}
	if !back.Settings.EncodesURL() {
		t.Error("encodeUrl lost")
	}
	if back.Settings.FollowingRedirects() {
		t.Error("followRedirects lost")
	}
	if back.Settings.MaxRedirectCount() != 4 {
		t.Errorf("maxRedirects = %d", back.Settings.MaxRedirectCount())
	}
	if back.Settings.Timeout == nil || back.Settings.Timeout.Millis != 2500 {
		t.Errorf("timeout = %+v", back.Settings.Timeout)
	}
	if len(back.Settings.OmitHeaders) != 2 {
		t.Errorf("omitHeaders = %v", back.Settings.OmitHeaders)
	}
}

func TestFolderRoundTripWithInheritance(t *testing.T) {
	user := "u"
	folder := &model.Item{
		Type: model.ItemFolder,
		Name: "sub",
		Auth: &model.Auth{Mode: model.AuthBasic, Username: &user},
		Headers: []model.KeyValue{
			{Name: strPtr("x-shared"), Value: strPtr("1"), Enabled: true},
		},
		Vars: []model.Variable{
			{Name: strPtr("v"), Value: "x", Enabled: true, DataType: "string"},
		},
		Script: &model.Scripts{Req: strPtr("bru.setVar('a',1);")},
	}
	out, err := New().WriteFolder(folder)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := New().ParseFolder(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}
	if back.Name != "sub" {
		t.Errorf("name = %q", back.Name)
	}
	if back.Auth == nil || back.Auth.Mode != model.AuthBasic {
		t.Errorf("auth = %+v", back.Auth)
	}
	if len(back.Headers) != 1 {
		t.Errorf("headers = %d", len(back.Headers))
	}
	if len(back.Vars) != 1 {
		t.Errorf("vars = %d", len(back.Vars))
	}
	if back.Script == nil || model.DerefString(back.Script.Req) == "" {
		t.Errorf("script = %+v", back.Script)
	}
}

func TestAssertionsRoundTrip(t *testing.T) {
	req := &model.Request{
		Method: "GET", URL: "http://x",
		Assertions: []model.KeyValue{
			{Name: strPtr("res.status"), Value: strPtr("200"), Enabled: true},
			{Name: strPtr("res.responseTime"), Value: strPtr("lt 1000"), Enabled: true},
		},
	}
	out, err := New().WriteRequest(req)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := New().ParseRequest(out)
	if err != nil {
		t.Fatalf("reparse:\n%s\n%v", out, err)
	}
	if len(back.Assertions) != 2 {
		t.Fatalf("assertions = %d", len(back.Assertions))
	}
	if back.Assertions[0].NameOrEmpty() != "res.status" {
		t.Errorf("assertion 0 = %+v", back.Assertions[0])
	}
}

func TestScalarValueUnmarshalShapes(t *testing.T) {
	// A sequence value is preserved as Raw rather than dropped.
	src := `name: E
variables:
  - name: weird
    value:
      - a
      - b
`
	env, err := New().ParseEnvironment([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(env.Variables) != 1 {
		t.Fatalf("vars = %d", len(env.Variables))
	}
	if env.Variables[0].NameOrEmpty() != "weird" {
		t.Errorf("var = %+v", env.Variables[0])
	}
}

func TestMalformedYAMLErrors(t *testing.T) {
	if _, err := New().ParseRequest([]byte("http: [unclosed\n")); err == nil {
		t.Error("expected a parse error")
	}
}
