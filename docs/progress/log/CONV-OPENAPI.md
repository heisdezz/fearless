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
