# CONV-OPENAPI — OpenAPI and Swagger specification importer and exporter

Owner history: antigravity (claimed 2026-11-11T01:50Z)

## Entries

### 2026-11-11T01:50Z — antigravity

- Did: Claimed `CONV-OPENAPI`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md`.
- Tests: not run yet.
- Notes: Implementing OpenAPI 3.x and Swagger 2 importer and exporter in `internal/convert/openapi/`:
  - Parsing JSON and YAML specifications.
  - Supporting OpenAPI 3.0, 3.1 and Swagger 2.0.
  - Server URL extraction into `baseUrl` collection variable.
  - Tag-based or path-based folder grouping.
  - Operations translation: method, URL, summary/operationId naming, description docs.
  - Parameters translation: query, path, headers.
  - Request body translation: JSON (schema/examples), form-urlencoded, multipart form-data, and Swagger 2 body/formData parameters.
  - Security schemes translation: HTTP basic, HTTP bearer, APIKey (header/query), OAuth2.
  - Exporting `model.Collection` to OpenAPI 3.0 specification.
  - Reporting dropped constructs.

### 2026-11-11T01:52Z — antigravity

- Did: Completed implementation of `CONV-OPENAPI`:
  - `internal/convert/openapi/schema.go`:
    - Full OpenAPI 3.x and Swagger 2.0 schema types with JSON and YAML annotations (`Spec`, `PathItemObject`, `OperationObject`, `ParameterObject`, `RequestBodyObject`, `SecuritySchemeObject`, etc.).
  - `internal/convert/openapi/import.go`:
    - `Import` and `ImportWithOptions` supporting JSON and YAML specifications.
    - Base URL extraction from `servers[0].url` or Swagger 2 `schemes` + `host` + `basePath` stored in `baseUrl` collection variable.
    - Folder grouping configurable by tags (`GroupByTags`) or flat (`GroupByNone`).
    - Operations conversion preserving path, methods (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD), and parameters (`query`, `path`, `header`).
    - Request body handling: OpenAPI 3 content negotiation (JSON with example formatting, form-urlencoded, multipart form-data, XML) and Swagger 2 body/formData parameters.
    - Security schemes conversion: HTTP Bearer, HTTP Basic, API Key (in header or query), and OAuth2.
  - `internal/convert/openapi/export.go`:
    - `Export` converting `model.Collection` back into valid OpenAPI 3.0.3 specification JSON with tags, paths, operations, parameters, and bodies.
  - Unit tests in `internal/convert/openapi/openapi_test.go`:
    - Tested OpenAPI 3.0 JSON import with full tag grouping and all parameter types.
    - Tested Swagger 2.0 YAML import with base URL synthesis and security definitions.
    - Tested grouping options (`GroupByNone`).
    - Tested export and round-trip fidelity.
- Tests: `go test -race -cover ./internal/convert/openapi/...` passes with 94.1% statement coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
