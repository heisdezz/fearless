package insomnia

import (
	"strings"
	"testing"

	"fearless/internal/core/model"
)

const sampleInsomniaJSON = `{
  "_type": "export",
  "__export_format": 4,
  "__export_date": "2026-11-11T00:00:00.000Z",
  "__export_source": "insomnia.desktop.app:v2023.5.8",
  "resources": [
    {
      "_id": "wrk_1",
      "_type": "workspace",
      "name": "My Insomnia API",
      "description": "Collection documentation here"
    },
    {
      "_id": "env_base",
      "_type": "environment",
      "parentId": "wrk_1",
      "name": "Base Environment",
      "data": {
        "baseUrl": "https://api.example.com",
        "version": "v1"
      }
    },
    {
      "_id": "env_staging",
      "_type": "environment",
      "parentId": "env_base",
      "name": "Staging",
      "color": "#ffaa00",
      "isPrivate": true,
      "data": {
        "baseUrl": "https://staging.example.com",
        "secretKey": "topsecret"
      }
    },
    {
      "_id": "fld_1",
      "_type": "request_group",
      "parentId": "wrk_1",
      "name": "Core Service",
      "description": "Core service group",
      "environment": {
        "timeout": 5000
      }
    },
    {
      "_id": "req_1",
      "_type": "request",
      "parentId": "fld_1",
      "name": "Get User",
      "url": "{{baseUrl}}/users/42?details=true",
      "method": "GET",
      "parameters": [
        { "name": "active", "value": "1", "disabled": false }
      ],
      "headers": [
        { "name": "Accept", "value": "application/json" },
        { "name": "X-Disabled", "value": "no", "disabled": true }
      ],
      "authentication": {
        "type": "bearer",
        "token": "bearer-token-123"
      },
      "settingFollowRedirects": "on"
    },
    {
      "_id": "req_2",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "Create Item",
      "url": "{{baseUrl}}/items",
      "method": "POST",
      "body": {
        "mimeType": "application/json",
        "text": "{\"title\":\"Widget\",\"qty\":10}"
      },
      "authentication": {
        "type": "basic",
        "username": "admin",
        "password": "secretpassword"
      }
    },
    {
      "_id": "req_3",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "GraphQL Query",
      "url": "{{baseUrl}}/graphql",
      "method": "POST",
      "body": {
        "mimeType": "application/graphql",
        "text": "{\"query\":\"query { items { id } }\",\"variables\":{\"limit\":5}}"
      }
    },
    {
      "_id": "req_4",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "Form Upload",
      "url": "{{baseUrl}}/upload",
      "method": "POST",
      "body": {
        "mimeType": "multipart/form-data",
        "params": [
          { "name": "desc", "value": "test file", "type": "text" },
          { "name": "file", "type": "file", "fileName": "/path/to/sample.txt" }
        ]
      }
    },
    {
      "_id": "req_5",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "URLEncoded Form",
      "url": "{{baseUrl}}/token",
      "method": "POST",
      "body": {
        "mimeType": "application/x-www-form-urlencoded",
        "params": [
          { "name": "grant_type", "value": "client_credentials" }
        ]
      }
    },
    {
      "_id": "req_6",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "Binary File",
      "url": "{{baseUrl}}/binary",
      "method": "PUT",
      "body": {
        "fileName": "/tmp/firmware.bin"
      }
    },
    {
      "_id": "req_7",
      "_type": "request",
      "parentId": "wrk_1",
      "name": "API Key Query",
      "url": "{{baseUrl}}/key",
      "method": "GET",
      "authentication": {
        "type": "apikey",
        "key": "api_key",
        "value": "xyz123",
        "addTo": "queryParams"
      }
    },
    {
      "_id": "jar_1",
      "_type": "cookie_jar",
      "parentId": "wrk_1"
    }
  ]
}`

func TestImportInsomniaJSON(t *testing.T) {
	res, err := Import([]byte(sampleInsomniaJSON))
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	coll := res.Collection
	if coll.Name != "My Insomnia API" {
		t.Errorf("collection name mismatch: %s", coll.Name)
	}
	if coll.Config.Docs == nil || *coll.Config.Docs != "Collection documentation here" {
		t.Errorf("collection docs mismatch: %v", coll.Config.Docs)
	}

	// Base environment variables converted to collection vars
	if len(coll.Config.Vars) != 2 {
		t.Errorf("expected 2 base env variables, got %d", len(coll.Config.Vars))
	}

	// Sub-environments
	if len(coll.Environments) != 1 {
		t.Fatalf("expected 1 sub-environment, got %d", len(coll.Environments))
	}
	staging := coll.Environments[0]
	if staging.Name != "Staging" || staging.Color != "#ffaa00" {
		t.Errorf("staging environment mismatch: %v", staging)
	}
	if len(staging.Secrets) != 2 { // all keys in private env are marked secret
		t.Errorf("expected 2 secrets in private env, got %d", len(staging.Secrets))
	}

	// Root items: Folder (Core Service) + 6 requests = 7 items
	if len(coll.Items) != 7 {
		t.Fatalf("expected 7 items, got %d", len(coll.Items))
	}

	// Folder verification
	folder := coll.Items[0]
	if !folder.IsFolder() || folder.Name != "Core Service" {
		t.Fatalf("folder mismatch: %v", folder)
	}
	if len(folder.Vars) != 1 || folder.Vars[0].NameOrEmpty() != "timeout" {
		t.Errorf("folder vars mismatch: %v", folder.Vars)
	}
	if len(folder.Children) != 1 {
		t.Fatalf("expected 1 child in folder, got %d", len(folder.Children))
	}

	// Folder Child request (Get User)
	req1 := folder.Children[0].Request
	if req1.Method != "GET" || req1.URL != "{{baseUrl}}/users/42" {
		t.Errorf("req1 URL/method mismatch: %s %s", req1.Method, req1.URL)
	}
	// Query params from URL (?details=true) + parameters array (active=1)
	params := req1.QueryParams()
	if len(params) != 2 {
		t.Fatalf("expected 2 query params, got %d: %v", len(params), params)
	}
	if req1.Auth == nil || req1.Auth.Mode != model.AuthBearer || *req1.Auth.Token != "bearer-token-123" {
		t.Errorf("req1 auth mismatch: %v", req1.Auth)
	}
	if req1.Settings.FollowRedirects == nil || !*req1.Settings.FollowRedirects {
		t.Errorf("req1 follow redirects should be true")
	}

	// JSON request (Create Item)
	req2 := coll.Items[1].Request
	if req2.Body == nil || req2.Body.Mode != model.BodyJSON || !strings.Contains(*req2.Body.JSON, "Widget") {
		t.Errorf("req2 body mismatch: %v", req2.Body)
	}
	if req2.Auth == nil || req2.Auth.Mode != model.AuthBasic || *req2.Auth.Username != "admin" {
		t.Errorf("req2 auth mismatch: %v", req2.Auth)
	}

	// GraphQL request
	item3 := coll.Items[2]
	if item3.Type != model.ItemGraphQLRequest || item3.Request.Body.Mode != model.BodyGraphQL {
		t.Errorf("req3 graphql mismatch: %v", item3)
	}
	if *item3.Request.Body.GraphQL.Query != "query { items { id } }" {
		t.Errorf("graphql query mismatch: %s", *item3.Request.Body.GraphQL.Query)
	}

	// Multipart Form (Form Upload)
	req4 := coll.Items[3].Request
	if req4.Body == nil || req4.Body.Mode != model.BodyMultipartForm || len(req4.Body.MultipartForm) != 2 {
		t.Fatalf("req4 multipart mismatch: %v", req4.Body)
	}
	if req4.Body.MultipartForm[1].Type != model.MultipartFile || *req4.Body.MultipartForm[1].Value != "@file(/path/to/sample.txt)" {
		t.Errorf("req4 multipart file mismatch: %v", req4.Body.MultipartForm[1])
	}

	// Form URLEncoded
	req5 := coll.Items[4].Request
	if req5.Body.Mode != model.BodyFormURLEncoded || len(req5.Body.FormURLEncoded) != 1 {
		t.Errorf("req5 urlencoded mismatch: %v", req5.Body)
	}

	// Binary File
	req6 := coll.Items[5].Request
	if req6.Body.Mode != model.BodyFile || len(req6.Body.File) != 1 || *req6.Body.File[0].FilePath != "/tmp/firmware.bin" {
		t.Errorf("req6 binary file mismatch: %v", req6.Body)
	}

	// APIKey in query
	req7 := coll.Items[6].Request
	if req7.Auth == nil || req7.Auth.Mode != model.AuthAPIKey || req7.Auth.Placement != model.APIKeyQueryParams {
		t.Errorf("req7 apikey mismatch: %v", req7.Auth)
	}

	// Cookie jar warning
	hasCookieWarning := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "cookie_jar") {
			hasCookieWarning = true
			break
		}
	}
	if !hasCookieWarning {
		t.Errorf("expected warning for cookie_jar, got: %v", res.Warnings)
	}
}

func TestImportInsomniaYAML(t *testing.T) {
	yamlData := `
_type: export
__export_format: 4
resources:
  - _id: wrk_yaml
    _type: workspace
    name: YAML Workspace
  - _id: req_yaml
    _type: request
    parentId: wrk_yaml
    name: Ping
    url: https://api.example.com/ping
    method: GET
`
	res, err := Import([]byte(yamlData))
	if err != nil {
		t.Fatalf("YAML import error: %v", err)
	}
	if res.Collection.Name != "YAML Workspace" {
		t.Errorf("name mismatch: %s", res.Collection.Name)
	}
	if len(res.Collection.Items) != 1 || res.Collection.Items[0].Name != "Ping" {
		t.Errorf("items mismatch: %v", res.Collection.Items)
	}
}

func TestImportAuthVariants(t *testing.T) {
	tests := []struct {
		name     string
		authJSON string
		wantMode model.AuthMode
	}{
		{
			name:     "Digest",
			authJSON: `{"type":"digest","username":"du","password":"dp"}`,
			wantMode: model.AuthDigest,
		},
		{
			name:     "NTLM",
			authJSON: `{"type":"ntlm","username":"nu","password":"np","domain":"dom"}`,
			wantMode: model.AuthNTLM,
		},
		{
			name:     "IAM / AWSv4",
			authJSON: `{"type":"iam","accessKeyId":"AKIA","secretAccessKey":"SEC"}`,
			wantMode: model.AuthAWSV4,
		},
		{
			name:     "OAuth2 with access token",
			authJSON: `{"type":"oauth2","accessToken":"tok123"}`,
			wantMode: model.AuthBearer,
		},
		{
			name:     "OAuth2 without token",
			authJSON: `{"type":"oauth2"}`,
			wantMode: model.AuthOAuth2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := `{"_type":"export","__export_format":4,"resources":[{"_id":"w","_type":"workspace"},{"_id":"r","_type":"request","parentId":"w","authentication":` + tt.authJSON + `}]}`
			res, err := Import([]byte(raw))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req := res.Collection.Items[0].Request
			if req.Auth == nil || req.Auth.Mode != tt.wantMode {
				t.Errorf("auth mode mismatch: got %v, want %s", req.Auth, tt.wantMode)
			}
		})
	}
}

func TestExportAndRoundTripInsomnia(t *testing.T) {
	res, err := Import([]byte(sampleInsomniaJSON))
	if err != nil {
		t.Fatalf("initial import error: %v", err)
	}

	// Export
	exportedBytes, err := Export(res.Collection)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Re-import
	res2, err := Import(exportedBytes)
	if err != nil {
		t.Fatalf("re-import error: %v", err)
	}

	if res2.Collection.Name != res.Collection.Name {
		t.Errorf("collection name round-trip mismatch: %s vs %s", res2.Collection.Name, res.Collection.Name)
	}
	if len(res2.Collection.Items) != len(res.Collection.Items) {
		t.Errorf("items round-trip mismatch: %d vs %d", len(res2.Collection.Items), len(res.Collection.Items))
	}

	// Error cases
	if _, err := Export(nil); err == nil {
		t.Error("expected error on nil collection export")
	}
	if _, err := Import([]byte("")); err == nil {
		t.Error("expected error on empty import")
	}
	if _, err := Import([]byte(`{"_type":"export","resources":[]}`)); err == nil {
		t.Error("expected error on empty resources list")
	}
}
