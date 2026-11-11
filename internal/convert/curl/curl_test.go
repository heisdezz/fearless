package curl

import (
	"strings"
	"testing"

	"fearless/internal/core/model"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "curl https://api.example.com",
			expected: []string{"https://api.example.com"},
		},
		{
			name:     "with quotes and line breaks",
			input:    "curl \\\n  -X POST \\\n  'https://api.example.com' \\\n  -H 'Content-Type: application/json'",
			expected: []string{"-X", "POST", "https://api.example.com", "-H", "Content-Type: application/json"},
		},
		{
			name:     "double quotes with escapes",
			input:    `curl "https://example.com" -d "{\"foo\": \"bar\"}"`,
			expected: []string{"https://example.com", "-d", `{"foo": "bar"}`},
		},
		{
			name:     "inline attached values",
			input:    `curl -XPOST -HContent-Type:text/plain -uuser:pass https://example.com`,
			expected: []string{"-XPOST", "-HContent-Type:text/plain", "-uuser:pass", "https://example.com"},
		},
		{
			name:     "long flags with equals",
			input:    `curl --request=PUT --header="Accept: text/html" --url=https://example.com`,
			expected: []string{"--request=PUT", "--header=Accept: text/html", "--url=https://example.com"},
		},
		{
			name:     "comments and whitespace",
			input:    "# comment line\ncurl https://example.com # trailing comment\n",
			expected: []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("length mismatch: got %v (len %d), want %v (len %d)", got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("token %d mismatch: got %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error on empty command")
	}

	_, err = Parse("curl -X GET")
	if err == nil {
		t.Fatal("expected error on missing URL")
	}
}

func TestParseMethodsAndURLs(t *testing.T) {
	// Simple GET
	res, err := Import("curl https://api.example.com/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Method != "GET" {
		t.Errorf("expected GET, got %s", res.Request.Method)
	}
	if res.Request.URL != "https://api.example.com/users" {
		t.Errorf("expected URL https://api.example.com/users, got %s", res.Request.URL)
	}
	if res.Item.Name != "users" {
		t.Errorf("expected item name 'users', got %s", res.Item.Name)
	}

	// Explicit DELETE
	res, err = Import("curl -X DELETE https://api.example.com/users/42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Method != "DELETE" {
		t.Errorf("expected DELETE, got %s", res.Request.Method)
	}

	// HEAD with -I
	res, err = Import("curl -I https://api.example.com/status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Method != "HEAD" {
		t.Errorf("expected HEAD, got %s", res.Request.Method)
	}
}

func TestParseQueryParams(t *testing.T) {
	cmd := "curl 'https://api.example.com/search?query=go+lang&page=2&filter=active'"
	res, err := Import(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Request.URL != "https://api.example.com/search" {
		t.Errorf("expected base URL without query, got %s", res.Request.URL)
	}
	params := res.Request.QueryParams()
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	if params[0].NameOrEmpty() != "query" || params[0].ValueOrEmpty() != "go lang" {
		t.Errorf("param 0 mismatch: %s=%s", params[0].NameOrEmpty(), params[0].ValueOrEmpty())
	}
	if params[1].NameOrEmpty() != "page" || params[1].ValueOrEmpty() != "2" {
		t.Errorf("param 1 mismatch: %s=%s", params[1].NameOrEmpty(), params[1].ValueOrEmpty())
	}
	if params[2].NameOrEmpty() != "filter" || params[2].ValueOrEmpty() != "active" {
		t.Errorf("param 2 mismatch: %s=%s", params[2].NameOrEmpty(), params[2].ValueOrEmpty())
	}
}

func TestParseGetWithDataChunks(t *testing.T) {
	cmd := "curl -G -d 'a=1' -d 'b=2' https://api.example.com/calc"
	res, err := Import(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Method != "GET" {
		t.Errorf("expected GET, got %s", res.Request.Method)
	}
	params := res.Request.QueryParams()
	if len(params) != 2 {
		t.Fatalf("expected 2 params from -G -d, got %d", len(params))
	}
	if params[0].NameOrEmpty() != "a" || params[0].ValueOrEmpty() != "1" {
		t.Errorf("param 0 mismatch: %s=%s", params[0].NameOrEmpty(), params[0].ValueOrEmpty())
	}
}

func TestParseHeadersAndSpecialHeaders(t *testing.T) {
	cmd := `curl https://api.example.com \
	  -H "X-API-Key: secret123" \
	  -H "Empty-Header;" \
	  -H "User-Agent: MyTestAgent/1.0" \
	  -A "OverridingAgent" \
	  -e "https://ref.example.com" \
	  -b "session=abc; theme=dark"`

	res, err := Import(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	headers := res.Request.LiveHeaders()
	foundHeader := func(name string) (string, bool) {
		for _, h := range headers {
			if strings.EqualFold(h.NameOrEmpty(), name) {
				return h.ValueOrEmpty(), true
			}
		}
		return "", false
	}

	if v, ok := foundHeader("X-API-Key"); !ok || v != "secret123" {
		t.Errorf("expected X-API-Key: secret123, got %s", v)
	}
	if v, ok := foundHeader("Empty-Header"); !ok || v != "" {
		t.Errorf("expected Empty-Header: '', got %s", v)
	}
	if v, ok := foundHeader("Referer"); !ok || v != "https://ref.example.com" {
		t.Errorf("expected Referer, got %s", v)
	}
	if v, ok := foundHeader("Cookie"); !ok || v != "session=abc; theme=dark" {
		t.Errorf("expected Cookie, got %s", v)
	}
}

func TestParseAuthTypes(t *testing.T) {
	// Basic via -u
	res, err := Import("curl -u user:pass https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthBasic {
		t.Fatalf("expected AuthBasic, got %v", res.Request.Auth)
	}
	if *res.Request.Auth.Username != "user" || *res.Request.Auth.Password != "pass" {
		t.Errorf("credentials mismatch: %s:%s", *res.Request.Auth.Username, *res.Request.Auth.Password)
	}

	// Basic via header
	res, err = Import("curl -H 'Authorization: Basic dXNlcjpwYXNz' https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthBasic {
		t.Fatalf("expected AuthBasic from header, got %v", res.Request.Auth)
	}
	if *res.Request.Auth.Username != "user" || *res.Request.Auth.Password != "pass" {
		t.Errorf("credentials mismatch: %s:%s", *res.Request.Auth.Username, *res.Request.Auth.Password)
	}

	// Bearer via header
	res, err = Import("curl -H 'Authorization: Bearer secret-token' https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthBearer {
		t.Fatalf("expected AuthBearer, got %v", res.Request.Auth)
	}
	if *res.Request.Auth.Token != "secret-token" {
		t.Errorf("token mismatch: %s", *res.Request.Auth.Token)
	}

	// Bearer via --oauth2-bearer
	res, err = Import("curl --oauth2-bearer direct-token https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthBearer || *res.Request.Auth.Token != "direct-token" {
		t.Errorf("bearer mismatch: %v", res.Request.Auth)
	}

	// Digest via --digest
	res, err = Import("curl --digest -u user:pass https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthDigest {
		t.Errorf("expected AuthDigest, got %v", res.Request.Auth)
	}

	// NTLM via --ntlm
	res, err = Import("curl --ntlm -u user:pass https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthNTLM {
		t.Errorf("expected AuthNTLM, got %v", res.Request.Auth)
	}

	// AWSV4 via --aws-sigv4
	res, err = Import("curl --aws-sigv4 aws:amz:us-east-1:s3 https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Auth == nil || res.Request.Auth.Mode != model.AuthAWSV4 {
		t.Errorf("expected AuthAWSV4, got %v", res.Request.Auth)
	}
}

func TestParseBodyModes(t *testing.T) {
	// JSON
	res, err := Import(`curl -X POST https://api.example.com -d '{"name":"alice","age":30}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body == nil || res.Request.Body.Mode != model.BodyJSON {
		t.Fatalf("expected BodyModeJSON, got %v", res.Request.Body)
	}
	if *res.Request.Body.JSON != `{"name":"alice","age":30}` {
		t.Errorf("JSON mismatch: %s", *res.Request.Body.JSON)
	}

	// GraphQL
	res, err = Import(`curl -X POST https://api.example.com/graphql -H "Content-Type: application/json" -d '{"query":"query { user { id } }","variables":{"x":1}}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body.GraphQL == nil || *res.Request.Body.GraphQL.Query != "query { user { id } }" {
		t.Errorf("GraphQL query mismatch: %v", res.Request.Body.GraphQL)
	}

	// XML
	res, err = Import(`curl -X POST https://api.example.com -d '<note><to>Bob</to></note>'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body.Mode != model.BodyXML || *res.Request.Body.XML != "<note><to>Bob</to></note>" {
		t.Errorf("XML mismatch: %v", res.Request.Body)
	}

	// Form URL-Encoded
	res, err = Import(`curl https://api.example.com --data-urlencode 'user=bob' --data-urlencode 'action=login'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body.Mode != model.BodyFormURLEncoded {
		t.Fatalf("expected BodyModeFormURLEncoded, got %v", res.Request.Body.Mode)
	}
	if len(res.Request.Body.FormURLEncoded) != 2 {
		t.Fatalf("expected 2 form entries, got %d", len(res.Request.Body.FormURLEncoded))
	}
	if res.Request.Body.FormURLEncoded[0].NameOrEmpty() != "user" || res.Request.Body.FormURLEncoded[0].ValueOrEmpty() != "bob" {
		t.Errorf("form 0 mismatch: %v", res.Request.Body.FormURLEncoded[0])
	}

	// Multipart Form
	res, err = Import(`curl -F 'title=profile' -F 'avatar=@/path/to/img.png;type=image/png' https://api.example.com/upload`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body.Mode != model.BodyMultipartForm {
		t.Fatalf("expected BodyModeMultipartForm, got %v", res.Request.Body.Mode)
	}
	if len(res.Request.Body.MultipartForm) != 2 {
		t.Fatalf("expected 2 multipart entries, got %d", len(res.Request.Body.MultipartForm))
	}
	if res.Request.Body.MultipartForm[0].Type != model.MultipartText || *res.Request.Body.MultipartForm[0].Value != "profile" {
		t.Errorf("text part mismatch: %v", res.Request.Body.MultipartForm[0])
	}
	filePart := res.Request.Body.MultipartForm[1]
	if filePart.Type != model.MultipartFile || *filePart.Value != "@file(/path/to/img.png)" || *filePart.ContentType != "image/png" {
		t.Errorf("file part mismatch: %v", filePart)
	}

	// File Binary
	res, err = Import(`curl --data-binary '@doc.pdf' https://api.example.com/upload`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.Body.Mode != model.BodyFile || len(res.Request.Body.File) != 1 || *res.Request.Body.File[0].FilePath != "doc.pdf" {
		t.Errorf("file binary mismatch: %v", res.Request.Body)
	}
}

func TestDroppedConstructsAndWarnings(t *testing.T) {
	cmd := `curl --compressed -k -L --proxy "http://127.0.0.1:8080" -m 30 --cacert "/etc/ssl/ca.pem" https://api.example.com`
	res, err := Import(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Warnings) < 4 {
		t.Errorf("expected multiple warnings, got: %v", res.Warnings)
	}

	hasWarning := func(substr string) bool {
		for _, w := range res.Warnings {
			if strings.Contains(strings.ToLower(w), strings.ToLower(substr)) {
				return true
			}
		}
		return false
	}

	if !hasWarning("compressed") {
		t.Error("missing warning for --compressed")
	}
	if !hasWarning("insecure") {
		t.Error("missing warning for -k")
	}
	if !hasWarning("proxy") {
		t.Error("missing warning for --proxy")
	}
	if !hasWarning("max-time") {
		t.Error("missing warning for -m")
	}
	if !hasWarning("cacert") {
		t.Error("missing warning for --cacert")
	}

	if res.Request.Settings.FollowRedirects == nil || !*res.Request.Settings.FollowRedirects {
		t.Error("expected FollowRedirects to be true from -L")
	}
}

func TestExportAndRoundTrip(t *testing.T) {
	jsonPayload := `{"hello":"world","count":42}`
	req := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/items",
		Params: []model.Param{
			{
				Type: model.ParamQuery,
				KeyValue: model.KeyValue{
					Name:    strPtr("page"),
					Value:   strPtr("1"),
					Enabled: true,
				},
			},
		},
		Headers: []model.KeyValue{
			{
				Name:    strPtr("X-Custom-Header"),
				Value:   strPtr("test-value"),
				Enabled: true,
			},
		},
		Auth: &model.Auth{
			Mode:     model.AuthBasic,
			Username: strPtr("admin"),
			Password: strPtr("secret123"),
		},
		Body: &model.Body{
			Mode: model.BodyJSON,
			JSON: &jsonPayload,
		},
	}

	cmd, err := Export(req)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Verify command contains expected flags
	if !strings.Contains(cmd, "-X POST") {
		t.Errorf("expected -X POST in exported cmd: %s", cmd)
	}
	if !strings.Contains(cmd, "page=1") {
		t.Errorf("expected page=1 in exported cmd: %s", cmd)
	}
	if !strings.Contains(cmd, "X-Custom-Header: test-value") {
		t.Errorf("expected custom header in exported cmd: %s", cmd)
	}
	if !strings.Contains(cmd, "-u 'admin:secret123'") {
		t.Errorf("expected -u admin:secret123 in exported cmd: %s", cmd)
	}
	if !strings.Contains(cmd, jsonPayload) {
		t.Errorf("expected json payload in exported cmd: %s", cmd)
	}

	// Re-import and assert fidelity
	imported, err := Import(cmd)
	if err != nil {
		t.Fatalf("re-import error: %v", err)
	}

	if imported.Request.Method != req.Method {
		t.Errorf("method round-trip mismatch: got %s, want %s", imported.Request.Method, req.Method)
	}
	if imported.Request.URL != req.URL {
		t.Errorf("URL round-trip mismatch: got %s, want %s", imported.Request.URL, req.URL)
	}
	if len(imported.Request.QueryParams()) != 1 || imported.Request.QueryParams()[0].ValueOrEmpty() != "1" {
		t.Errorf("params round-trip mismatch: %v", imported.Request.QueryParams())
	}
	if imported.Request.Auth == nil || imported.Request.Auth.Mode != model.AuthBasic || *imported.Request.Auth.Username != "admin" {
		t.Errorf("auth round-trip mismatch: %v", imported.Request.Auth)
	}
	if imported.Request.Body == nil || imported.Request.Body.Mode != model.BodyJSON || *imported.Request.Body.JSON != jsonPayload {
		t.Errorf("body round-trip mismatch: %v", imported.Request.Body)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestImportConvenienceFunctions(t *testing.T) {
	// ImportRequest success
	req, warnings, err := ImportRequest("curl https://api.example.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "GET" || req.URL != "https://api.example.com/test" {
		t.Errorf("request mismatch: %v", req)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	// ImportRequest error
	_, _, err = ImportRequest("")
	if err == nil {
		t.Error("expected error on empty command")
	}

	// ImportItem success
	item, _, err := ImportItem("curl -X PUT https://api.example.com/items/5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Type != model.ItemHTTPRequest || item.Request.Method != "PUT" {
		t.Errorf("item mismatch: %v", item)
	}

	// ImportItem error
	_, _, err = ImportItem("")
	if err == nil {
		t.Error("expected error on empty command")
	}
}

func TestExportAllModesAndAuth(t *testing.T) {
	// Nil request
	_, err := Export(nil)
	if err == nil {
		t.Error("expected error on nil request")
	}

	// Single line option
	req := &model.Request{
		Method: "GET",
		URL:    "https://api.example.com/ping",
	}
	single, err := ExportWithOptions(req, ExportOptions{Multiline: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(single, "\n") {
		t.Errorf("expected single line output, got %q", single)
	}

	// FollowRedirects
	tTrue := true
	req.Settings.FollowRedirects = &tTrue
	cmd, _ := Export(req)
	if !strings.Contains(cmd, "-L") {
		t.Errorf("expected -L for follow redirects: %s", cmd)
	}

	// Text body
	textPayload := "raw plain text"
	reqText := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/text",
		Body: &model.Body{
			Mode: model.BodyText,
			Text: &textPayload,
		},
	}
	cmd, _ = Export(reqText)
	if !strings.Contains(cmd, "raw plain text") {
		t.Errorf("expected plain text in cmd: %s", cmd)
	}

	// XML body
	xmlPayload := "<data>123</data>"
	reqXML := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/xml",
		Body: &model.Body{
			Mode: model.BodyXML,
			XML:  &xmlPayload,
		},
	}
	cmd, _ = Export(reqXML)
	if !strings.Contains(cmd, "application/xml") || !strings.Contains(cmd, xmlPayload) {
		t.Errorf("expected xml in cmd: %s", cmd)
	}

	// Sparql body
	sparqlPayload := "SELECT * WHERE { ?s ?p ?o }"
	reqSparql := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/sparql",
		Body: &model.Body{
			Mode:   model.BodySparql,
			Sparql: &sparqlPayload,
		},
	}
	cmd, _ = Export(reqSparql)
	if !strings.Contains(cmd, sparqlPayload) {
		t.Errorf("expected sparql in cmd: %s", cmd)
	}

	// GraphQL body
	gqlQuery := "query GetUser { user { id } }"
	gqlVars := `{"id": 10}`
	reqGQL := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/graphql",
		Body: &model.Body{
			Mode: model.BodyGraphQL,
			GraphQL: &model.GraphQLBody{
				Query:     &gqlQuery,
				Variables: &gqlVars,
			},
		},
	}
	cmd, _ = Export(reqGQL)
	if !strings.Contains(cmd, "GetUser") {
		t.Errorf("expected graphql in cmd: %s", cmd)
	}

	// Form URL-Encoded
	reqForm := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/form",
		Body: &model.Body{
			Mode: model.BodyFormURLEncoded,
			FormURLEncoded: []model.KeyValue{
				{
					Name:    strPtr("username"),
					Value:   strPtr("user1"),
					Enabled: true,
				},
				{
					Name:    strPtr("pass"),
					Value:   strPtr("pass1"),
					Enabled: true,
				},
			},
		},
	}
	cmd, _ = Export(reqForm)
	if !strings.Contains(cmd, "--data-urlencode 'username=user1'") {
		t.Errorf("expected form urlencode in cmd: %s", cmd)
	}

	// Multipart Form
	reqMulti := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/multi",
		Body: &model.Body{
			Mode: model.BodyMultipartForm,
			MultipartForm: []model.MultipartEntry{
				{
					Type:    model.MultipartText,
					Name:    strPtr("comment"),
					Value:   strPtr("my comment"),
					Enabled: true,
				},
				{
					Type:        model.MultipartFile,
					Name:        strPtr("attachment"),
					Value:       strPtr("@file(/tmp/file.pdf)"),
					ContentType: strPtr("application/pdf"),
					Enabled:     true,
				},
			},
		},
	}
	cmd, _ = Export(reqMulti)
	if !strings.Contains(cmd, "-F 'comment=my comment'") || !strings.Contains(cmd, "-F 'attachment=@/tmp/file.pdf;type=application/pdf'") {
		t.Errorf("expected multipart parts in cmd: %s", cmd)
	}

	// File body
	filePath := "/tmp/binary.dat"
	reqFile := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/binary",
		Body: &model.Body{
			Mode: model.BodyFile,
			File: []model.FileEntry{
				{
					FilePath: &filePath,
					Selected: true,
				},
			},
		},
	}
	cmd, _ = Export(reqFile)
	if !strings.Contains(cmd, "--data-binary '@/tmp/binary.dat'") {
		t.Errorf("expected data-binary in cmd: %s", cmd)
	}

	// Auth modes: Bearer, Digest, NTLM, AWSV4
	token := "bearer-token-123"
	reqAuth := &model.Request{
		Method: "GET",
		URL:    "https://api.example.com/auth",
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: &token,
		},
	}
	cmd, _ = Export(reqAuth)
	if !strings.Contains(cmd, "Authorization: Bearer bearer-token-123") {
		t.Errorf("expected bearer token in cmd: %s", cmd)
	}

	reqAuth.Auth = &model.Auth{
		Mode:     model.AuthDigest,
		Username: strPtr("duser"),
		Password: strPtr("dpass"),
	}
	cmd, _ = Export(reqAuth)
	if !strings.Contains(cmd, "--digest") || !strings.Contains(cmd, "-u 'duser:dpass'") {
		t.Errorf("expected digest in cmd: %s", cmd)
	}

	reqAuth.Auth = &model.Auth{
		Mode:     model.AuthNTLM,
		Username: strPtr("nuser"),
		Password: strPtr("npass"),
	}
	cmd, _ = Export(reqAuth)
	if !strings.Contains(cmd, "--ntlm") || !strings.Contains(cmd, "-u 'nuser:npass'") {
		t.Errorf("expected ntlm in cmd: %s", cmd)
	}

	reqAuth.Auth = &model.Auth{
		Mode: model.AuthAWSV4,
	}
	cmd, _ = Export(reqAuth)
	if !strings.Contains(cmd, "--aws-sigv4") {
		t.Errorf("expected aws-sigv4 in cmd: %s", cmd)
	}
}

func TestMoreParserFlags(t *testing.T) {
	cmd := `curl --connect-timeout 15 --retry 3 --resolve "example.com:443:127.0.0.1" --cert "/tmp/c.pem" --key "/tmp/k.pem" -o "/tmp/out.txt" -c "/tmp/jar.txt" --url https://example.com/api`
	res, err := Import(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.URL != "https://example.com/api" {
		t.Errorf("URL mismatch: %s", res.Request.URL)
	}
	if len(res.Warnings) < 7 {
		t.Errorf("expected at least 7 warnings, got %d: %v", len(res.Warnings), res.Warnings)
	}
}
