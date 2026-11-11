package openapi

import (
	"strings"
	"testing"

	"fearless/internal/core/model"
)

const sampleOpenAPI3JSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "Petstore OpenAPI 3",
    "description": "Sample petstore API in OpenAPI 3",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "https://api.petstore.com/v1"
    }
  ],
  "components": {
    "securitySchemes": {
      "petstore_auth": {
        "type": "oauth2"
      },
      "api_key": {
        "type": "apiKey",
        "name": "api_key",
        "in": "header"
      },
      "bearer_auth": {
        "type": "http",
        "scheme": "bearer"
      }
    }
  },
  "paths": {
    "/pets": {
      "get": {
        "tags": ["Pets"],
        "summary": "List all pets",
        "operationId": "listPets",
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "description": "How many items to return",
            "required": false,
            "default": 20
          }
        ],
        "security": [
          {
            "api_key": []
          }
        ]
      },
      "post": {
        "tags": ["Pets"],
        "summary": "Create a pet",
        "operationId": "createPets",
        "requestBody": {
          "description": "Pet to add",
          "required": true,
          "content": {
            "application/json": {
              "example": {
                "name": "Doggie",
                "tag": "dog"
              }
            }
          }
        },
        "security": [
          {
            "bearer_auth": []
          }
        ]
      }
    },
    "/pets/{petId}": {
      "parameters": [
        {
          "name": "petId",
          "in": "path",
          "required": true,
          "example": "42"
        }
      ],
      "get": {
        "tags": ["Pets"],
        "summary": "Get pet by ID",
        "description": "Returns a single pet"
      }
    },
    "/store/order": {
      "post": {
        "tags": ["Store"],
        "summary": "Place an order",
        "requestBody": {
          "content": {
            "application/x-www-form-urlencoded": {}
          }
        }
      }
    }
  }
}`

const sampleSwagger2YAML = `
swagger: "2.0"
info:
  title: Swagger 2.0 API
  description: Sample Swagger 2 API in YAML
  version: "2.0"
host: api.swagger.example.com
basePath: /api
schemes:
  - https
securityDefinitions:
  basic_auth:
    type: basic
  query_key:
    type: apiKey
    name: token
    in: query
paths:
  /users:
    get:
      summary: Get users
      parameters:
        - name: role
          in: query
          default: admin
      security:
        - basic_auth: []
    post:
      summary: Add user
      parameters:
        - name: user
          in: formData
          required: true
      security:
        - query_key: []
`

func TestImportOpenAPI3JSON(t *testing.T) {
	res, err := Import([]byte(sampleOpenAPI3JSON))
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	coll := res.Collection
	if coll.Name != "Petstore OpenAPI 3" {
		t.Errorf("title mismatch: %s", coll.Name)
	}
	if coll.Config.Docs == nil || *coll.Config.Docs != "Sample petstore API in OpenAPI 3" {
		t.Errorf("description mismatch: %v", coll.Config.Docs)
	}

	// Base URL variable
	if len(coll.Config.Vars) != 1 || coll.Config.Vars[0].NameOrEmpty() != "baseUrl" || coll.Config.Vars[0].Value != "https://api.petstore.com/v1" {
		t.Errorf("baseUrl variable mismatch: %v", coll.Config.Vars)
	}

	// Tag grouping: 2 folders ("Pets", "Store")
	if len(coll.Items) != 2 {
		t.Fatalf("expected 2 folders from tags, got %d", len(coll.Items))
	}

	petsFolder := coll.Items[0]
	if !petsFolder.IsFolder() || petsFolder.Name != "Pets" {
		t.Errorf("folder 0 mismatch: %v", petsFolder)
	}
	// Pets has 3 requests: GET /pets, POST /pets, GET /pets/{petId}
	if len(petsFolder.Children) != 3 {
		t.Fatalf("expected 3 items in Pets folder, got %d", len(petsFolder.Children))
	}

	// GET /pets
	req1 := petsFolder.Children[0].Request
	if req1.Method != "GET" || req1.URL != "{{baseUrl}}/pets" {
		t.Errorf("req1 mismatch: %s %s", req1.Method, req1.URL)
	}
	if len(req1.QueryParams()) != 1 || req1.QueryParams()[0].NameOrEmpty() != "limit" {
		t.Errorf("req1 query params mismatch: %v", req1.QueryParams())
	}
	if req1.Auth == nil || req1.Auth.Mode != model.AuthAPIKey || req1.Auth.Placement != model.APIKeyHeader {
		t.Errorf("req1 apiKey auth mismatch: %v", req1.Auth)
	}

	// POST /pets with JSON body
	req2 := petsFolder.Children[1].Request
	if req2.Method != "POST" || req2.Body == nil || req2.Body.Mode != model.BodyJSON || !strings.Contains(*req2.Body.JSON, "Doggie") {
		t.Errorf("req2 JSON body mismatch: %v", req2.Body)
	}
	if req2.Auth == nil || req2.Auth.Mode != model.AuthBearer {
		t.Errorf("req2 bearer auth mismatch: %v", req2.Auth)
	}

	// GET /pets/{petId} with path param
	req3 := petsFolder.Children[2].Request
	if req3.Method != "GET" || len(req3.PathParams()) != 1 || req3.PathParams()[0].NameOrEmpty() != "petId" {
		t.Errorf("req3 path params mismatch: %v", req3.PathParams())
	}

	// Store folder
	storeFolder := coll.Items[1]
	if !storeFolder.IsFolder() || storeFolder.Name != "Store" || len(storeFolder.Children) != 1 {
		t.Errorf("store folder mismatch: %v", storeFolder)
	}
	storeReq := storeFolder.Children[0].Request
	if storeReq.Body == nil || storeReq.Body.Mode != model.BodyFormURLEncoded {
		t.Errorf("store req body mismatch: %v", storeReq.Body)
	}
}

func TestImportSwagger2YAML(t *testing.T) {
	res, err := Import([]byte(sampleSwagger2YAML))
	if err != nil {
		t.Fatalf("unexpected Swagger 2 import error: %v", err)
	}

	coll := res.Collection
	if coll.Name != "Swagger 2.0 API" {
		t.Errorf("title mismatch: %s", coll.Name)
	}
	if len(coll.Config.Vars) != 1 || coll.Config.Vars[0].Value != "https://api.swagger.example.com/api" {
		t.Errorf("baseUrl variable mismatch: %v", coll.Config.Vars)
	}

	// Swagger sample has no tags, so operations are root items
	if len(coll.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(coll.Items))
	}

	// GET /users with Basic auth
	getReq := coll.Items[0].Request
	if getReq.Method != "GET" || getReq.Auth == nil || getReq.Auth.Mode != model.AuthBasic {
		t.Errorf("get users basic auth mismatch: %v", getReq.Auth)
	}

	// POST /users with Query APIKey
	postReq := coll.Items[1].Request
	if postReq.Method != "POST" || postReq.Auth == nil || postReq.Auth.Mode != model.AuthAPIKey || postReq.Auth.Placement != model.APIKeyQueryParams {
		t.Errorf("post users query apikey mismatch: %v", postReq.Auth)
	}
}

func TestImportOptionsGroupByNone(t *testing.T) {
	res, err := ImportWithOptions([]byte(sampleOpenAPI3JSON), ImportOptions{GroupBy: GroupByNone})
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	// All 4 operations should be root items
	if len(res.Collection.Items) != 4 {
		t.Errorf("expected 4 root items with GroupByNone, got %d", len(res.Collection.Items))
	}
	for _, it := range res.Collection.Items {
		if it.IsFolder() {
			t.Errorf("unexpected folder found with GroupByNone: %s", it.Name)
		}
	}
}

func TestExportAndRoundTripOpenAPI(t *testing.T) {
	res, err := Import([]byte(sampleOpenAPI3JSON))
	if err != nil {
		t.Fatalf("initial import error: %v", err)
	}

	// Export to OpenAPI JSON
	exportedBytes, err := Export(res.Collection)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Re-import exported spec
	res2, err := Import(exportedBytes)
	if err != nil {
		t.Fatalf("re-import error: %v", err)
	}

	if res2.Collection.Name != res.Collection.Name {
		t.Errorf("title round-trip mismatch: %s vs %s", res2.Collection.Name, res.Collection.Name)
	}
	if len(res2.Collection.Items) != len(res.Collection.Items) {
		t.Errorf("items count round-trip mismatch: %d vs %d", len(res2.Collection.Items), len(res.Collection.Items))
	}

	// Error handling tests
	if _, err := Export(nil); err == nil {
		t.Error("expected error on nil collection export")
	}
	if _, err := Import([]byte("")); err == nil {
		t.Error("expected error on empty import")
	}
	if _, err := Import([]byte(`{"foo":"bar"}`)); err == nil {
		t.Error("expected error on invalid document")
	}
}

func TestMoreBodyAndSecurityTypes(t *testing.T) {
	specRaw := `{
  "openapi": "3.0.0",
  "info": { "title": "Advanced Spec", "version": "1.0" },
  "components": {
    "securitySchemes": {
      "basic_auth": { "type": "http", "scheme": "basic" },
      "oauth_sec": { "type": "oauth2" }
    }
  },
  "paths": {
    "/upload": {
      "post": {
        "summary": "Multipart Upload",
        "parameters": [
          { "name": "description", "in": "formData" }
        ],
        "requestBody": {
          "content": {
            "multipart/form-data": {}
          }
        },
        "security": [
          { "basic_auth": [] }
        ]
      }
    },
    "/xml": {
      "put": {
        "summary": "XML Payload",
        "requestBody": {
          "content": {
            "application/xml": {}
          }
        },
        "security": [
          { "oauth_sec": [] }
        ]
      }
    },
    "/methods": {
      "delete": { "summary": "Delete" },
      "patch": { "summary": "Patch" },
      "options": { "summary": "Options" },
      "head": { "summary": "Head" }
    }
  }
}`

	res, err := Import([]byte(specRaw))
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	// Verify operations were generated
	coll := res.Collection
	if len(coll.Items) != 6 {
		t.Fatalf("expected 6 operations, got %d", len(coll.Items))
	}

	// Export with all methods and normalizePath variations
	coll.Items = append(coll.Items, &model.Item{
		UID:  "custom-1",
		Name: "Full URL Request",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://custom.domain.com/v2/items?page=1",
		},
	})
	coll.Items = append(coll.Items, &model.Item{
		UID:  "custom-2",
		Name: "Relative URL Request",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "POST",
			URL:    "items",
		},
	})

	exported, err := Export(coll)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}
	if !strings.Contains(string(exported), "/v2/items") || !strings.Contains(string(exported), "/items") {
		t.Errorf("expected paths in exported spec: %s", string(exported))
	}
}
