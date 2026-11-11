package build_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/engine/build"
	"fearless/internal/core/model"
)

func stringPtr(s string) *string { return &s }

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line comments",
			input: "{\n  // comment here\n  \"key\": \"value\"\n}",
			want:  "{\n  \n  \"key\": \"value\"\n}",
		},
		{
			name:  "multi line comments",
			input: "{\n  /* multi\n  line */\n  \"key\": \"value\"\n}",
			want:  "{\n  \n  \"key\": \"value\"\n}",
		},
		{
			name:  "preserves slashes inside strings",
			input: `{"url": "https://example.com/api", "note": "keep // and /* */"}`,
			want:  `{"url": "https://example.com/api", "note": "keep // and /* */"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := build.StripComments(tc.input)
			if got != tc.want {
				t.Errorf("StripComments() =\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

func TestSynthesizeURL(t *testing.T) {
	pathParams := []model.Param{
		{
			Type: model.ParamPath,
			KeyValue: model.KeyValue{
				Name:    stringPtr("userId"),
				Value:   stringPtr("42"),
				Enabled: true,
			},
		},
		{
			Type: model.ParamPath,
			KeyValue: model.KeyValue{
				Name:    stringPtr("orderId"),
				Value:   stringPtr("1001"),
				Enabled: true,
			},
		},
	}

	queryParams := []model.Param{
		{
			Type: model.ParamQuery,
			KeyValue: model.KeyValue{
				Name:    stringPtr("filter"),
				Value:   stringPtr("active"),
				Enabled: true,
			},
		},
	}

	// 1. Path param substitution & scheme prepend
	u, raw, err := build.SynthesizeURL("localhost:8080/users/:userId/orders", pathParams, nil, false)
	if err != nil {
		t.Fatalf("SynthesizeURL failed: %v", err)
	}
	if u.Scheme != "http" || u.Host != "localhost:8080" || u.Path != "/users/42/orders" {
		t.Errorf("unexpected synthesized URL: %s (raw: %s)", u.String(), raw)
	}

	// 2. OData parameter substitution
	u, _, err = build.SynthesizeURL("https://api.example.com/Orders(:orderId)", pathParams, nil, false)
	if err != nil {
		t.Fatalf("SynthesizeURL OData failed: %v", err)
	}
	if u.Path != "/Orders(1001)" {
		t.Errorf("OData substitution failed: %s", u.Path)
	}

	// 3. Unresolved path parameter stays literal
	u, _, err = build.SynthesizeURL("https://api.example.com/items/:unresolved", pathParams, nil, false)
	if err != nil {
		t.Fatalf("SynthesizeURL literal failed: %v", err)
	}
	if u.Path != "/items/:unresolved" {
		t.Errorf("expected literal unresolved parameter: %s", u.Path)
	}

	// 4. Query param merging
	u, _, err = build.SynthesizeURL("https://api.example.com/search?q=test", nil, queryParams, false)
	if err != nil {
		t.Fatalf("SynthesizeURL query failed: %v", err)
	}
	if !strings.Contains(u.RawQuery, "q=test") || !strings.Contains(u.RawQuery, "filter=active") {
		t.Errorf("query merging failed: %s", u.RawQuery)
	}

	// 5. Invalid URL format error
	_, _, err = build.SynthesizeURL("", nil, nil, false)
	if err == nil {
		t.Errorf("expected error for empty URL")
	}
}

func TestBodyVariantsAndContentHeaders(t *testing.T) {
	ctx := context.Background()

	// 1. Body Mode: None
	t.Run("BodyNone", func(t *testing.T) {
		item := &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    "https://api.example.com/ping",
				Body:   &model.Body{Mode: model.BodyNone},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if res.Body != nil {
			t.Errorf("expected nil body for BodyNone")
		}
		if res.ContentLength != 0 {
			t.Errorf("expected ContentLength 0, got %d", res.ContentLength)
		}
		if res.Headers.Get("Content-Type") != "" {
			t.Errorf("Content-Type should be suppressed for BodyNone, got %q", res.Headers.Get("Content-Type"))
		}
	})

	// 2. Body Mode: JSON with comments and interpolation
	t.Run("BodyJSON", func(t *testing.T) {
		jsonText := `// Config comment
{
  /* Multi
     line */
  "name": "{{user.name}}",
  "quote": "{{quote}}"
}`
		varsMap := map[string]any{
			"user.name": "Alice",
			"quote":     "hello \"world\"\nline2",
		}

		item := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/users",
				Body: &model.Body{
					Mode: model.BodyJSON,
					JSON: stringPtr(jsonText),
				},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{
			Item:      item,
			Variables: varsMap,
		})
		if err != nil {
			t.Fatalf("Build JSON failed: %v", err)
		}

		if res.ContentType != "application/json" {
			t.Errorf("ContentType = %q, want application/json", res.ContentType)
		}
		if res.ContentLength != int64(len(res.Body)) {
			t.Errorf("ContentLength mismatch: %d vs %d", res.ContentLength, len(res.Body))
		}
		if !strings.Contains(string(res.Body), `"name": "Alice"`) {
			t.Errorf("JSON interpolation failed: %s", string(res.Body))
		}
		// Slashes and newlines must be safely escaped in JSON mode
		if !strings.Contains(string(res.Body), `hello \"world\"`) {
			t.Errorf("JSON string escaping failed: %s", string(res.Body))
		}
	})

	// 3. Body Mode: Text, XML, Sparql
	t.Run("BodyTextXMLSparql", func(t *testing.T) {
		itemText := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/echo",
				Body: &model.Body{
					Mode: model.BodyText,
					Text: stringPtr("plain text {{var}}"),
				},
			},
		}
		resText, err := build.Build(ctx, build.BuildInput{
			Item:      itemText,
			Variables: map[string]any{"var": "val"},
		})
		if err != nil || resText.ContentType != "text/plain" || string(resText.Body) != "plain text val" {
			t.Errorf("text body failed: %+v %v", resText, err)
		}

		itemXML := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/xml",
				Body: &model.Body{
					Mode: model.BodyXML,
					XML:  stringPtr("<root>{{var}}</root>"),
				},
			},
		}
		resXML, err := build.Build(ctx, build.BuildInput{
			Item:      itemXML,
			Variables: map[string]any{"var": "ok"},
		})
		if err != nil || resXML.ContentType != "application/xml" || string(resXML.Body) != "<root>ok</root>" {
			t.Errorf("xml body failed: %+v %v", resXML, err)
		}

		itemSparql := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/sparql",
				Body: &model.Body{
					Mode:   model.BodySparql,
					Sparql: stringPtr("SELECT * WHERE { ?s ?p ?o }"),
				},
			},
		}
		resSparql, err := build.Build(ctx, build.BuildInput{Item: itemSparql})
		if err != nil || resSparql.ContentType != "application/sparql-query" {
			t.Errorf("sparql body failed: %+v %v", resSparql, err)
		}
	})

	// 4. Body Mode: GraphQL
	t.Run("BodyGraphQL", func(t *testing.T) {
		query := "query GetUser($id: ID!) { user(id: $id) { name } }"
		variables := "{\n  // comment\n  \"id\": {{userId}}\n}"

		item := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/graphql",
				Body: &model.Body{
					Mode: model.BodyGraphQL,
					GraphQL: &model.GraphQLBody{
						Query:     stringPtr(query),
						Variables: stringPtr(variables),
					},
				},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{
			Item:      item,
			Variables: map[string]any{"userId": 101},
		})
		if err != nil {
			t.Fatalf("GraphQL build failed: %v", err)
		}
		if res.ContentType != "application/json" {
			t.Errorf("GraphQL ContentType = %q, want application/json", res.ContentType)
		}
		if !strings.Contains(string(res.Body), `"query":`) || !strings.Contains(string(res.Body), `"variables":{"id":101}`) {
			t.Errorf("GraphQL payload unexpected: %s", string(res.Body))
		}
	})

	// 5. Body Mode: FormUrlEncoded
	t.Run("BodyFormURLEncoded", func(t *testing.T) {
		item := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/login",
				Body: &model.Body{
					Mode: model.BodyFormURLEncoded,
					FormURLEncoded: []model.KeyValue{
						{
							Name:    stringPtr("user"),
							Value:   stringPtr("alice & bob"),
							Enabled: true,
						},
						{
							Name:    stringPtr("pass"),
							Value:   stringPtr("secret 123"),
							Enabled: true,
						},
					},
				},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("FormURLEncoded failed: %v", err)
		}
		if res.ContentType != "application/x-www-form-urlencoded" {
			t.Errorf("ContentType = %q", res.ContentType)
		}
		if !strings.Contains(string(res.Body), "user=alice+%26+bob") || !strings.Contains(string(res.Body), "pass=secret+123") {
			t.Errorf("FormURLEncoded payload unexpected: %s", string(res.Body))
		}
	})

	// 6. Body Mode: MultipartForm
	t.Run("BodyMultipartForm", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFilePath := filepath.Join(tmpDir, "upload.txt")
		if err := os.WriteFile(testFilePath, []byte("file content here"), 0o644); err != nil {
			t.Fatalf("failed to write upload file: %v", err)
		}

		item := &model.Item{
			Request: &model.Request{
				Method: "POST",
				URL:    "https://api.example.com/upload",
				Body: &model.Body{
					Mode: model.BodyMultipartForm,
					MultipartForm: []model.MultipartEntry{
						{
							Type:    model.MultipartText,
							Name:    stringPtr("field1"),
							Value:   stringPtr("hello {{name}}"),
							Enabled: true,
						},
						{
							Type:        model.MultipartFile,
							Name:        stringPtr("doc"),
							Value:       stringPtr(testFilePath),
							ContentType: stringPtr("text/plain"),
							Enabled:     true,
						},
					},
				},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{
			Item:      item,
			Variables: map[string]any{"name": "world"},
		})
		if err != nil {
			t.Fatalf("Multipart build failed: %v", err)
		}

		if !strings.HasPrefix(res.ContentType, "multipart/form-data; boundary=") {
			t.Errorf("unexpected multipart Content-Type: %s", res.ContentType)
		}
		bodyStr := string(res.Body)
		if !strings.Contains(bodyStr, "name=\"field1\"") || !strings.Contains(bodyStr, "hello world") {
			t.Errorf("missing text part in multipart body: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "filename=\"upload.txt\"") || !strings.Contains(bodyStr, "file content here") {
			t.Errorf("missing file part in multipart body: %s", bodyStr)
		}
	})

	// 7. Body Mode: File
	t.Run("BodyFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		binaryPath := filepath.Join(tmpDir, "binary.bin")
		fileBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		if err := os.WriteFile(binaryPath, fileBytes, 0o644); err != nil {
			t.Fatalf("failed to write binary file: %v", err)
		}

		item := &model.Item{
			Request: &model.Request{
				Method: "PUT",
				URL:    "https://api.example.com/blob",
				Body: &model.Body{
					Mode: model.BodyFile,
					File: []model.FileEntry{
						{
							FilePath:    stringPtr(binaryPath),
							ContentType: stringPtr("application/octet-stream"),
							Selected:    true,
						},
					},
				},
			},
		}

		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("File body build failed: %v", err)
		}

		if res.ContentType != "application/octet-stream" {
			t.Errorf("ContentType = %q", res.ContentType)
		}
		if res.ContentLength != int64(len(fileBytes)) {
			t.Errorf("ContentLength = %d, want %d", res.ContentLength, len(fileBytes))
		}
	})
}

func TestExplicitContentTypeWins(t *testing.T) {
	ctx := context.Background()

	item := &model.Item{
		Request: &model.Request{
			Method: "POST",
			URL:    "https://api.example.com/custom",
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("Content-Type"),
					Value:   stringPtr("application/vnd.api+json"),
					Enabled: true,
				},
			},
			Body: &model.Body{
				Mode: model.BodyJSON,
				JSON: stringPtr(`{"id": 1}`),
			},
		},
	}

	res, err := build.Build(ctx, build.BuildInput{Item: item})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.ContentType != "application/vnd.api+json" {
		t.Errorf("explicit Content-Type did not take precedence! got %q", res.ContentType)
	}
	if res.Headers.Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("header Content-Type = %q", res.Headers.Get("Content-Type"))
	}
}

func TestNestedInterpolationAndMocks(t *testing.T) {
	ctx := context.Background()

	varsMap := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "id": 10},
			map[string]any{"name": "Bob", "id": 20},
		},
		"host": "api.service.local",
	}

	rawJSON := `{
  "userId": {{users[0].id}},
  "userName": "{{users[0].name}}",
  "requestId": "{{$guid}}",
  "timestamp": {{$timestamp}},
  "missing": "{{unresolvedVar}}"
}`

	item := &model.Item{
		Request: &model.Request{
			Method: "POST",
			URL:    "https://{{host}}/events",
			Body: &model.Body{
				Mode: model.BodyJSON,
				JSON: stringPtr(rawJSON),
			},
		},
	}

	res, err := build.Build(ctx, build.BuildInput{
		Item:      item,
		Variables: varsMap,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.URL.Host != "api.service.local" {
		t.Errorf("URL host interpolation failed: %s", res.URL.Host)
	}

	bodyStr := string(res.Body)
	if !strings.Contains(bodyStr, `"userId": 10`) {
		t.Errorf("nested array lookup failed: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"userName": "Alice"`) {
		t.Errorf("nested property lookup failed: %s", bodyStr)
	}
	// Check $guid mock: should be replaced with uuid string, not literal {{$guid}}
	if strings.Contains(bodyStr, `{{$guid}}`) {
		t.Errorf("mock {{$guid}} was not expanded: %s", bodyStr)
	}
	// Check $timestamp mock
	if strings.Contains(bodyStr, `{{$timestamp}}`) {
		t.Errorf("mock {{$timestamp}} was not expanded: %s", bodyStr)
	}
	// Check unresolved variable: must be tracked and remain literal in text
	if !strings.Contains(bodyStr, `{{unresolvedVar}}`) {
		t.Errorf("unresolved variable should remain literal: %s", bodyStr)
	}
	foundUnresolved := false
	for _, u := range res.UnresolvedVars {
		if u == "unresolvedVar" {
			foundUnresolved = true
			break
		}
	}
	if !foundUnresolved {
		t.Errorf("unresolved variable not reported in UnresolvedVars: %v", res.UnresolvedVars)
	}
}

func TestTreeMergeInheritance(t *testing.T) {
	ctx := context.Background()

	parentFolder := &model.Item{
		UID:  "folder-1",
		Type: model.ItemFolder,
		Name: "Auth Folder",
		Headers: []model.KeyValue{
			{
				Name:    stringPtr("X-Folder-Header"),
				Value:   stringPtr("FolderVal"),
				Enabled: true,
			},
			{
				Name:    stringPtr("X-Overridden"),
				Value:   stringPtr("FolderOld"),
				Enabled: true,
			},
		},
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: stringPtr("folder-token-xyz"),
		},
	}

	reqItem := &model.Item{
		UID:  "req-1",
		Type: model.ItemHTTPRequest,
		Name: "Child Request",
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/data",
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("X-Overridden"),
					Value:   stringPtr("RequestNew"),
					Enabled: true,
				},
				{
					Name:    stringPtr("X-Request-Only"),
					Value:   stringPtr("ReqVal"),
					Enabled: true,
				},
			},
			Auth: &model.Auth{
				Mode: model.AuthInherit, // Inherits from parent folder!
			},
		},
	}
	parentFolder.Children = []*model.Item{reqItem}

	col := &model.Collection{
		UID:   "col-1",
		Name:  "Test Col",
		Items: []*model.Item{parentFolder},
	}

	res, err := build.Build(ctx, build.BuildInput{
		Item:       reqItem,
		Collection: col,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 1. Header inheritance & override
	if res.Headers.Get("X-Folder-Header") != "FolderVal" {
		t.Errorf("folder header not inherited: %s", res.Headers.Get("X-Folder-Header"))
	}
	if res.Headers.Get("X-Overridden") != "RequestNew" {
		t.Errorf("request header did not override folder header: %s", res.Headers.Get("X-Overridden"))
	}
	if res.Headers.Get("X-Request-Only") != "ReqVal" {
		t.Errorf("request-only header missing: %s", res.Headers.Get("X-Request-Only"))
	}

	// 2. Auth inheritance: Bearer token from parent folder
	authHeader := res.Headers.Get("Authorization")
	if authHeader != "Bearer folder-token-xyz" {
		t.Errorf("inherited Bearer auth failed: got %q", authHeader)
	}
}

func TestAuthModes(t *testing.T) {
	ctx := context.Background()

	// 1. Basic Auth
	t.Run("BasicAuth", func(t *testing.T) {
		item := &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    "https://api.example.com/basic",
				Auth: &model.Auth{
					Mode:     model.AuthBasic,
					Username: stringPtr("admin"),
					Password: stringPtr("pass123"),
				},
			},
		}
		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("Build basic auth failed: %v", err)
		}
		authH := res.Headers.Get("Authorization")
		if !strings.HasPrefix(authH, "Basic ") {
			t.Errorf("expected Basic auth header, got %q", authH)
		}
	})

	// 2. API Key Header vs Query Placement
	t.Run("APIKeyPlacement", func(t *testing.T) {
		itemHeader := &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    "https://api.example.com/v1",
				Auth: &model.Auth{
					Mode:      model.AuthAPIKey,
					Key:       stringPtr("X-API-KEY"),
					Value:     stringPtr("secret-key-1"),
					Placement: model.APIKeyHeader,
				},
			},
		}
		resHeader, err := build.Build(ctx, build.BuildInput{Item: itemHeader})
		if err != nil || resHeader.Headers.Get("X-API-KEY") != "secret-key-1" {
			t.Errorf("API key header placement failed: %+v", resHeader)
		}

		itemQuery := &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    "https://api.example.com/v1?search=all",
				Auth: &model.Auth{
					Mode:      model.AuthAPIKey,
					Key:       stringPtr("api_key"),
					Value:     stringPtr("secret-key-2"),
					Placement: model.APIKeyQueryParams,
				},
			},
		}
		resQuery, err := build.Build(ctx, build.BuildInput{Item: itemQuery})
		if err != nil {
			t.Fatalf("API key query placement build failed: %v", err)
		}
		if !strings.Contains(resQuery.URL.RawQuery, "api_key=secret-key-2") {
			t.Errorf("API key query placement missing in query: %s", resQuery.URL.RawQuery)
		}
	})

	// 3. WSSE Auth
	t.Run("WSSEAuth", func(t *testing.T) {
		item := &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    "https://api.example.com/wsse",
				Auth: &model.Auth{
					Mode:     model.AuthWSSE,
					Username: stringPtr("wsse-user"),
					Password: stringPtr("wsse-pass"),
				},
			},
		}
		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("WSSE build failed: %v", err)
		}
		if res.Headers.Get("X-WSSE") == "" {
			t.Errorf("missing X-WSSE header")
		}
		if res.Headers.Get("Authorization") != `WSSE profile="UsernameToken"` {
			t.Errorf("unexpected Authorization header for WSSE: %s", res.Headers.Get("Authorization"))
		}
	})
}

func TestHTTPRequestObject(t *testing.T) {
	ctx := context.Background()

	item := &model.Item{
		Request: &model.Request{
			Method: "POST",
			URL:    "https://api.example.com/submit",
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("X-Custom"),
					Value:   stringPtr("custom-val"),
					Enabled: true,
				},
			},
			Body: &model.Body{
				Mode: model.BodyJSON,
				JSON: stringPtr(`{"msg": "hi"}`),
			},
		},
	}

	builder := build.New()
	res, err := builder.Build(ctx, build.BuildInput{Item: item})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	httpReq := res.HTTPRequest
	if httpReq == nil {
		t.Fatalf("HTTPRequest is nil")
	}

	if httpReq.Method != http.MethodPost {
		t.Errorf("httpReq.Method = %s; want POST", httpReq.Method)
	}
	if httpReq.URL.String() != "https://api.example.com/submit" {
		t.Errorf("httpReq.URL = %s", httpReq.URL.String())
	}
	if httpReq.Header.Get("X-Custom") != "custom-val" {
		t.Errorf("httpReq header X-Custom missing")
	}
	if httpReq.ContentLength != int64(len(res.Body)) {
		t.Errorf("httpReq.ContentLength = %d, want %d", httpReq.ContentLength, len(res.Body))
	}

	bodyRead, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("read httpReq.Body failed: %v", err)
	}
	if string(bodyRead) != string(res.Body) {
		t.Errorf("body mismatch: %s vs %s", string(bodyRead), string(res.Body))
	}
}

func TestHeaderDeletionUQ_B1(t *testing.T) {
	ctx := context.Background()

	parentFolder := &model.Item{
		UID:  "folder-del",
		Type: model.ItemFolder,
		Headers: []model.KeyValue{
			{
				Name:    stringPtr("X-Keep"),
				Value:   stringPtr("keep-me"),
				Enabled: true,
			},
			{
				Name:    stringPtr("X-To-Delete"),
				Value:   stringPtr("delete-me"),
				Enabled: true,
			},
		},
	}

	reqItem := &model.Item{
		UID:  "req-del",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/test",
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("X-To-Delete"),
					Value:   stringPtr(""), // Explicit empty value deletes per UQ-B1
					Enabled: true,
				},
			},
		},
	}
	parentFolder.Children = []*model.Item{reqItem}
	col := &model.Collection{
		UID:   "col-del",
		Items: []*model.Item{parentFolder},
	}

	res, err := build.Build(ctx, build.BuildInput{
		Item:       reqItem,
		Collection: col,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.Headers.Get("X-Keep") != "keep-me" {
		t.Errorf("X-Keep header should be retained: %s", res.Headers.Get("X-Keep"))
	}
	if got := res.Headers.Get("X-To-Delete"); got != "" {
		t.Errorf("X-To-Delete should have been deleted by child empty string, got %q", got)
	}
}

func TestCollectionRootInheritance(t *testing.T) {
	ctx := context.Background()

	col := &model.Collection{
		UID:  "col-root",
		Name: "Root Collection",
		Config: model.CollectionConfig{
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("X-Col-Header"),
					Value:   stringPtr("col-val"),
					Enabled: true,
				},
			},
			Auth: &model.Auth{
				Mode:  model.AuthBearer,
				Token: stringPtr("col-bearer-token"),
			},
		},
	}

	reqItem := &model.Item{
		UID:  "req-inherit-col",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/check",
			Auth: &model.Auth{
				Mode: model.AuthInherit,
			},
		},
	}
	col.Items = []*model.Item{reqItem}

	res, err := build.Build(ctx, build.BuildInput{
		Item:       reqItem,
		Collection: col,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.Headers.Get("X-Col-Header") != "col-val" {
		t.Errorf("Collection root header not inherited: %s", res.Headers.Get("X-Col-Header"))
	}
	if res.Headers.Get("Authorization") != "Bearer col-bearer-token" {
		t.Errorf("Collection root auth not inherited: %s", res.Headers.Get("Authorization"))
	}
}

func TestQueryParamsPreserveDuplicatesAndOrder(t *testing.T) {
	ctx := context.Background()

	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/items?base=0",
			Params: []model.Param{
				{
					Type: model.ParamQuery,
					KeyValue: model.KeyValue{
						Name:    stringPtr("tag"),
						Value:   stringPtr("alpha"),
						Enabled: true,
					},
				},
				{
					Type: model.ParamQuery,
					KeyValue: model.KeyValue{
						Name:    stringPtr("tag"),
						Value:   stringPtr("beta"),
						Enabled: true,
					},
				},
				{
					Type: model.ParamQuery,
					KeyValue: model.KeyValue{
						Name:    stringPtr("sort"),
						Value:   stringPtr("desc"),
						Enabled: true,
					},
				},
			},
		},
	}

	res, err := build.Build(ctx, build.BuildInput{Item: item})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expectedQuery := "base=0&tag=alpha&tag=beta&sort=desc"
	if res.URL.RawQuery != expectedQuery {
		t.Errorf("Query mismatch:\n got:  %s\n want: %s", res.URL.RawQuery, expectedQuery)
	}
}

func TestFormURLEncodedOrderPreserved(t *testing.T) {
	ctx := context.Background()

	item := &model.Item{
		Request: &model.Request{
			Method: "POST",
			URL:    "https://api.example.com/submit",
			Body: &model.Body{
				Mode: model.BodyFormURLEncoded,
				FormURLEncoded: []model.KeyValue{
					{
						Name:    stringPtr("z_last"),
						Value:   stringPtr("3"),
						Enabled: true,
					},
					{
						Name:    stringPtr("a_first"),
						Value:   stringPtr("1"),
						Enabled: true,
					},
					{
						Name:    stringPtr("m_mid"),
						Value:   stringPtr("2"),
						Enabled: true,
					},
				},
			},
		},
	}

	res, err := build.Build(ctx, build.BuildInput{Item: item})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expectedBody := "z_last=3&a_first=1&m_mid=2"
	if string(res.Body) != expectedBody {
		t.Errorf("FormURLEncoded order was sorted or destroyed:\n got:  %s\n want: %s", string(res.Body), expectedBody)
	}
}

func TestHostHeaderSetsHTTPRequestHost(t *testing.T) {
	ctx := context.Background()

	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    "https://127.0.0.1:8080/v1",
			Headers: []model.KeyValue{
				{
					Name:    stringPtr("Host"),
					Value:   stringPtr("custom.virtualhost.org"),
					Enabled: true,
				},
			},
		},
	}

	res, err := build.Build(ctx, build.BuildInput{Item: item})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.HTTPRequest.Host != "custom.virtualhost.org" {
		t.Errorf("HTTPRequest.Host = %q, want custom.virtualhost.org", res.HTTPRequest.Host)
	}
}

// TestAuthHeaderReplacesUserHeader pins the rule that a header the auth mode
// generates replaces a same-named header the user set, rather than joining it.
//
// http.Header.Add appends and net/http writes one line per value, so appending
// put two Authorization headers on the wire with the user's stale value first.
// Bruno assigns to its header object, so its generated value is the only one
// sent; this matches that.
func TestAuthHeaderReplacesUserHeader(t *testing.T) {
	ctx := context.Background()

	buildWith := func(t *testing.T, auth *model.Auth, headers ...model.KeyValue) *build.BuiltRequest {
		t.Helper()
		item := &model.Item{
			Request: &model.Request{
				Method:  "GET",
				URL:     "https://api.example.com/x",
				Headers: headers,
				Auth:    auth,
			},
		}
		res, err := build.Build(ctx, build.BuildInput{Item: item})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return res
	}
	userAuth := model.KeyValue{
		Name: stringPtr("Authorization"), Value: stringPtr("Bearer USER-SET"), Enabled: true,
	}

	t.Run("Bearer", func(t *testing.T) {
		res := buildWith(t,
			&model.Auth{Mode: model.AuthBearer, Token: stringPtr("GENERATED")},
			userAuth)
		got := res.Headers.Values("Authorization")
		if len(got) != 1 {
			t.Fatalf("Authorization = %v, want exactly one value", got)
		}
		if got[0] != "Bearer GENERATED" {
			t.Errorf("Authorization = %q, want the generated value", got[0])
		}
	})

	t.Run("Basic", func(t *testing.T) {
		res := buildWith(t,
			&model.Auth{Mode: model.AuthBasic, Username: stringPtr("u"), Password: stringPtr("p")},
			userAuth)
		got := res.Headers.Values("Authorization")
		if len(got) != 1 {
			t.Fatalf("Authorization = %v, want exactly one value", got)
		}
		if !strings.HasPrefix(got[0], "Basic ") {
			t.Errorf("Authorization = %q, want the generated Basic value", got[0])
		}
	})

	t.Run("WSSE", func(t *testing.T) {
		// WSSE generates both X-WSSE and Authorization; the user set both names.
		res := buildWith(t,
			&model.Auth{Mode: model.AuthWSSE, Username: stringPtr("u"), Password: stringPtr("p")},
			userAuth,
			model.KeyValue{Name: stringPtr("X-WSSE"), Value: stringPtr("stale"), Enabled: true})
		if got := res.Headers.Values("Authorization"); len(got) != 1 ||
			got[0] != `WSSE profile="UsernameToken"` {
			t.Errorf("Authorization = %v, want one WSSE profile line", got)
		}
		if got := res.Headers.Values("X-WSSE"); len(got) != 1 || got[0] == "stale" {
			t.Errorf("X-WSSE = %v, want one generated value", got)
		}
	})

	t.Run("APIKeyCustomName", func(t *testing.T) {
		// The generated header name is user-chosen, so replacement must key on
		// whatever name the auth produced, not on Authorization.
		res := buildWith(t,
			&model.Auth{
				Mode:      model.AuthAPIKey,
				Key:       stringPtr("X-API-KEY"),
				Value:     stringPtr("fresh"),
				Placement: model.APIKeyHeader,
			},
			model.KeyValue{Name: stringPtr("X-API-KEY"), Value: stringPtr("stale"), Enabled: true})
		got := res.Headers.Values("X-API-KEY")
		if len(got) != 1 {
			t.Fatalf("X-API-KEY = %v, want exactly one value", got)
		}
		if got[0] != "fresh" {
			t.Errorf("X-API-KEY = %q, want the generated value", got[0])
		}
	})

	t.Run("UnrelatedUserHeadersSurvive", func(t *testing.T) {
		// Replacement must be narrow: only the auth-managed name is touched.
		res := buildWith(t,
			&model.Auth{Mode: model.AuthBearer, Token: stringPtr("GENERATED")},
			userAuth,
			model.KeyValue{Name: stringPtr("X-Trace"), Value: stringPtr("abc"), Enabled: true})
		if got := res.Headers.Get("X-Trace"); got != "abc" {
			t.Errorf("X-Trace = %q, want abc: an unrelated header was dropped", got)
		}
	})

	t.Run("DuplicatesCollapse", func(t *testing.T) {
		// Two user entries under the same name (different casing) collapse to one
		// generated value, and HeaderEntries stays in sync with the map.
		res := buildWith(t,
			&model.Auth{Mode: model.AuthBearer, Token: stringPtr("GENERATED")},
			model.KeyValue{Name: stringPtr("Authorization"), Value: stringPtr("one"), Enabled: true},
			model.KeyValue{Name: stringPtr("authorization"), Value: stringPtr("two"), Enabled: true})
		if got := res.Headers.Values("Authorization"); len(got) != 1 || got[0] != "Bearer GENERATED" {
			t.Errorf("Authorization = %v, want one generated value", got)
		}
		var authEntries int
		for _, e := range res.HeaderEntries {
			if strings.EqualFold(e.Name, "Authorization") {
				authEntries++
				if e.Value != "Bearer GENERATED" {
					t.Errorf("HeaderEntries has stale value %q", e.Value)
				}
			}
		}
		if authEntries != 1 {
			t.Errorf("HeaderEntries has %d Authorization entries, want 1", authEntries)
		}
	})
}
