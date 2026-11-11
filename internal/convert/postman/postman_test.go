package postman

import (
	"strings"
	"testing"

	"fearless/internal/core/model"
)

const samplePostmanCollection = `{
  "info": {
    "_postman_id": "12345-abcde",
    "name": "E2E Postman Collection",
    "description": "Full collection documentation",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {
      "key": "baseUrl",
      "value": "https://api.example.com",
      "type": "string"
    },
    {
      "key": "disabledVar",
      "value": "disabled",
      "disabled": true
    }
  ],
  "auth": {
    "type": "bearer",
    "bearer": [
      {
        "key": "token",
        "value": "collection-token",
        "type": "string"
      }
    ]
  },
  "event": [
    {
      "listen": "prerequest",
      "script": {
        "type": "text/javascript",
        "exec": [
          "pm.environment.set('time', Date.now());",
          "pm.collectionVariables.set('runId', 42);"
        ]
      }
    }
  ],
  "item": [
    {
      "name": "Auth",
      "description": "Authentication folder",
      "auth": {
        "type": "basic",
        "basic": [
          { "key": "username", "value": "folder_admin" },
          { "key": "password", "value": "folder_pass" }
        ]
      },
      "item": [
        {
          "name": "Login Request",
          "request": {
            "method": "POST",
            "description": "Logs in the user",
            "url": {
              "raw": "{{baseUrl}}/auth/login?redirect=dashboard",
              "query": [
                { "key": "redirect", "value": "dashboard", "disabled": false }
              ],
              "variable": [
                { "key": "id", "value": "100" }
              ]
            },
            "header": [
              { "key": "Content-Type", "value": "application/json" },
              { "key": "X-Disabled", "value": "ignore", "disabled": true }
            ],
            "body": {
              "mode": "raw",
              "raw": "{\"username\":\"alice\",\"password\":\"secret\"}",
              "options": {
                "raw": { "language": "json" }
              }
            }
          },
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Status is 200', function() {",
                  "  pm.expect(pm.response.json().success).to.be.true;",
                  "});"
                ]
              }
            }
          ]
        }
      ]
    },
    {
      "name": "URL String Request",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/ping?foo=bar"
      }
    },
    {
      "name": "GraphQL Request",
      "request": {
        "method": "POST",
        "url": "https://api.example.com/graphql",
        "body": {
          "mode": "graphql",
          "graphql": {
            "query": "query { me { name } }",
            "variables": "{\"a\": 1}"
          }
        }
      }
    },
    {
      "name": "Form URL-Encoded Request",
      "request": {
        "method": "POST",
        "url": "https://api.example.com/form",
        "body": {
          "mode": "urlencoded",
          "urlencoded": [
            { "key": "grant_type", "value": "password" },
            { "key": "disabled_key", "value": "x", "disabled": true }
          ]
        }
      }
    },
    {
      "name": "Form Data Request",
      "request": {
        "method": "POST",
        "url": "https://api.example.com/upload",
        "body": {
          "mode": "formdata",
          "formdata": [
            { "key": "text_field", "value": "hello", "type": "text" },
            { "key": "file_field", "src": "/path/to/img.png", "type": "file", "contentType": "image/png" }
          ]
        }
      }
    },
    {
      "name": "Binary File Request",
      "request": {
        "method": "PUT",
        "url": "https://api.example.com/blob",
        "body": {
          "mode": "file",
          "file": {
            "src": "blob.bin"
          }
        }
      }
    },
    {
      "name": "All Auth Modes",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/auth-test",
        "auth": {
          "type": "apikey",
          "apikey": [
            { "key": "key", "value": "X-API-KEY" },
            { "key": "value", "value": "12345" },
            { "key": "in", "value": "query" }
          ]
        }
      }
    }
  ]
}`

func TestImportPostmanCollection(t *testing.T) {
	res, err := Import([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	coll := res.Collection
	if coll.Name != "E2E Postman Collection" {
		t.Errorf("collection name mismatch: %s", coll.Name)
	}
	if coll.Config.Docs == nil || *coll.Config.Docs != "Full collection documentation" {
		t.Errorf("collection docs mismatch: %v", coll.Config.Docs)
	}

	// Variables
	if len(coll.Config.Vars) != 2 {
		t.Fatalf("expected 2 collection vars, got %d", len(coll.Config.Vars))
	}
	if coll.Config.Vars[0].NameOrEmpty() != "baseUrl" || !coll.Config.Vars[0].Enabled {
		t.Errorf("var 0 mismatch: %v", coll.Config.Vars[0])
	}
	if coll.Config.Vars[1].Enabled {
		t.Errorf("var 1 should be disabled")
	}

	// Collection Auth
	if coll.Config.Auth == nil || coll.Config.Auth.Mode != model.AuthBearer || *coll.Config.Auth.Token != "collection-token" {
		t.Errorf("collection auth mismatch: %v", coll.Config.Auth)
	}

	// Collection Script
	if coll.Config.Script == nil || coll.Config.Script.Req == nil {
		t.Fatalf("expected prerequest collection script")
	}
	if !strings.Contains(*coll.Config.Script.Req, "bru.setEnvVar('time'") || !strings.Contains(*coll.Config.Script.Req, "bru.setVar('runId'") {
		t.Errorf("script transpilation mismatch: %s", *coll.Config.Script.Req)
	}

	// Items & Folders
	if len(coll.Items) != 7 {
		t.Fatalf("expected 7 top-level items, got %d", len(coll.Items))
	}

	// Folder verification
	folder := coll.Items[0]
	if !folder.IsFolder() || folder.Name != "Auth" {
		t.Fatalf("expected folder 'Auth', got: %v", folder)
	}
	if folder.Auth == nil || folder.Auth.Mode != model.AuthBasic || *folder.Auth.Username != "folder_admin" {
		t.Errorf("folder auth mismatch: %v", folder.Auth)
	}
	if len(folder.Children) != 1 {
		t.Fatalf("expected 1 child in folder, got %d", len(folder.Children))
	}

	// Nested Request
	child := folder.Children[0]
	if child.Type != model.ItemHTTPRequest || child.Name != "Login Request" {
		t.Fatalf("expected request 'Login Request', got: %v", child)
	}
	req := child.Request
	if req.Method != "POST" || req.URL != "{{baseUrl}}/auth/login" {
		t.Errorf("req URL/method mismatch: %s %s", req.Method, req.URL)
	}
	if len(req.QueryParams()) != 1 || req.QueryParams()[0].NameOrEmpty() != "redirect" {
		t.Errorf("query params mismatch: %v", req.QueryParams())
	}
	if len(req.PathParams()) != 1 || req.PathParams()[0].NameOrEmpty() != "id" {
		t.Errorf("path params mismatch: %v", req.PathParams())
	}
	if len(req.LiveHeaders()) != 1 || req.LiveHeaders()[0].NameOrEmpty() != "Content-Type" {
		t.Errorf("headers mismatch: %v", req.LiveHeaders())
	}
	if req.Body == nil || req.Body.Mode != model.BodyJSON || !strings.Contains(*req.Body.JSON, "alice") {
		t.Errorf("body mismatch: %v", req.Body)
	}
	if req.Tests == nil || !strings.Contains(*req.Tests, "test('Status is 200'") || !strings.Contains(*req.Tests, "expect(res.getBody()") {
		t.Errorf("test transpilation mismatch: %v", req.Tests)
	}

	// String URL request
	strReqItem := coll.Items[1]
	if strReqItem.Request.URL != "https://api.example.com/ping" {
		t.Errorf("string url mismatch: %s", strReqItem.Request.URL)
	}
	if len(strReqItem.Request.QueryParams()) != 1 || strReqItem.Request.QueryParams()[0].NameOrEmpty() != "foo" {
		t.Errorf("string query param mismatch: %v", strReqItem.Request.QueryParams())
	}

	// GraphQL
	gqlItem := coll.Items[2]
	if gqlItem.Type != model.ItemGraphQLRequest || gqlItem.Request.Body.Mode != model.BodyGraphQL {
		t.Errorf("graphql item mismatch: %v", gqlItem)
	}

	// Form URL-Encoded
	formItem := coll.Items[3]
	if formItem.Request.Body.Mode != model.BodyFormURLEncoded || len(formItem.Request.Body.FormURLEncoded) != 2 {
		t.Errorf("form urlencoded mismatch: %v", formItem.Request.Body)
	}

	// Form Data
	multiItem := coll.Items[4]
	if multiItem.Request.Body.Mode != model.BodyMultipartForm || len(multiItem.Request.Body.MultipartForm) != 2 {
		t.Errorf("formdata mismatch: %v", multiItem.Request.Body)
	}
	if multiItem.Request.Body.MultipartForm[1].Type != model.MultipartFile || *multiItem.Request.Body.MultipartForm[1].Value != "@file(/path/to/img.png)" {
		t.Errorf("multipart file part mismatch: %v", multiItem.Request.Body.MultipartForm[1])
	}

	// File body
	fileItem := coll.Items[5]
	if fileItem.Request.Body.Mode != model.BodyFile || len(fileItem.Request.Body.File) != 1 || *fileItem.Request.Body.File[0].FilePath != "blob.bin" {
		t.Errorf("file body mismatch: %v", fileItem.Request.Body)
	}

	// APIKey in query
	apiKeyItem := coll.Items[6]
	if apiKeyItem.Request.Auth == nil || apiKeyItem.Request.Auth.Mode != model.AuthAPIKey || apiKeyItem.Request.Auth.Placement != model.APIKeyQueryParams {
		t.Errorf("apikey auth mismatch: %v", apiKeyItem.Request.Auth)
	}
}

func TestImportPostmanAuthVariants(t *testing.T) {
	tests := []struct {
		name     string
		authJSON string
		wantMode model.AuthMode
	}{
		{
			name:     "Digest auth",
			authJSON: `{"type":"digest","digest":[{"key":"username","value":"duser"},{"key":"password","value":"dpass"}]}`,
			wantMode: model.AuthDigest,
		},
		{
			name:     "NTLM auth",
			authJSON: `{"type":"ntlm","ntlm":[{"key":"username","value":"nuser"},{"key":"password","value":"npass"},{"key":"domain","value":"DOMAIN"}]}`,
			wantMode: model.AuthNTLM,
		},
		{
			name:     "OAuth2 with access token",
			authJSON: `{"type":"oauth2","oauth2":[{"key":"accessToken","value":"oauth2-token"}]}`,
			wantMode: model.AuthBearer,
		},
		{
			name:     "AWS SigV4 auth",
			authJSON: `{"type":"awsv4","awsv4":[{"key":"accessKey","value":"AKIA..."},{"key":"secretKey","value":"SECRET"}]}`,
			wantMode: model.AuthAWSV4,
		},
		{
			name:     "No auth",
			authJSON: `{"type":"noauth"}`,
			wantMode: model.AuthNone,
		},
		{
			name:     "Unknown auth",
			authJSON: `{"type":"custom-hawk"}`,
			wantMode: model.AuthNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := `{"info":{"name":"Test"},"item":[{"name":"req","request":{"method":"GET","url":"http://x.com","auth":` + tt.authJSON + `}}]}`
			res, err := Import([]byte(raw))
			if err != nil {
				t.Fatalf("import error: %v", err)
			}
			req := res.Collection.Items[0].Request
			if req.Auth == nil || req.Auth.Mode != tt.wantMode {
				t.Errorf("auth mode mismatch: got %v, want %s", req.Auth, tt.wantMode)
			}
		})
	}
}

func TestImportPostmanEnvironment(t *testing.T) {
	envRaw := `{
  "id": "env-123",
  "name": "Staging Env",
  "values": [
    {
      "key": "host",
      "value": "staging.example.com",
      "enabled": true
    },
    {
      "key": "apiKey",
      "value": "secret-val",
      "type": "secret",
      "enabled": true
    },
    {
      "key": "debug",
      "value": "false",
      "enabled": false
    }
  ]
}`

	env, err := ImportEnvironment([]byte(envRaw))
	if err != nil {
		t.Fatalf("unexpected environment import error: %v", err)
	}

	if env.Name != "Staging Env" {
		t.Errorf("env name mismatch: %s", env.Name)
	}
	if len(env.Variables) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(env.Variables))
	}
	if len(env.Secrets) != 1 || env.Secrets[0] != "apiKey" {
		t.Errorf("secrets mismatch: %v", env.Secrets)
	}

	// Export Environment
	exportedBytes, err := ExportEnvironment(env)
	if err != nil {
		t.Fatalf("unexpected export environment error: %v", err)
	}

	reImported, err := ImportEnvironment(exportedBytes)
	if err != nil {
		t.Fatalf("unexpected re-import error: %v", err)
	}
	if reImported.Name != env.Name || len(reImported.Variables) != len(env.Variables) || len(reImported.Secrets) != 1 {
		t.Errorf("environment roundtrip mismatch: %v", reImported)
	}
}

func TestExportAndRoundTripCollection(t *testing.T) {
	res, err := Import([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("initial import error: %v", err)
	}

	// Export to Postman JSON
	exportedJSON, err := Export(res.Collection)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Re-import exported Postman JSON
	res2, err := Import(exportedJSON)
	if err != nil {
		t.Fatalf("re-import error: %v", err)
	}

	if res2.Collection.Name != res.Collection.Name {
		t.Errorf("name round-trip mismatch: %s vs %s", res2.Collection.Name, res.Collection.Name)
	}
	if len(res2.Collection.Items) != len(res.Collection.Items) {
		t.Errorf("items count mismatch: %d vs %d", len(res2.Collection.Items), len(res.Collection.Items))
	}

	// Error tests for nil
	if _, err := Export(nil); err == nil {
		t.Error("expected error on nil collection export")
	}
	if _, err := ExportEnvironment(nil); err == nil {
		t.Error("expected error on nil environment export")
	}
	if _, err := Import([]byte("invalid json")); err == nil {
		t.Error("expected error on invalid json import")
	}
	if _, err := ImportEnvironment([]byte("invalid json")); err == nil {
		t.Error("expected error on invalid json environment import")
	}
}
