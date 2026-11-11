# CONV-POSTMAN — Postman collection importer

Owner history: antigravity (claimed 2026-11-11T01:38Z)

## Entries

### 2026-11-11T01:38Z — antigravity

- Did: Claimed `CONV-POSTMAN`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md`.
- Tests: not run yet.
- Notes: Implementing Postman collection importer in `internal/convert/postman/`:
  - Parsing Postman v2.0 and v2.1 collection schemas.
  - Converting collection hierarchy, folders, requests, descriptions, and metadata into `model.Collection` and `model.Item` trees.
  - Converting URLs (string or structured objects), query params, headers, and path variables.
  - Converting auth (basic, bearer, apikey, digest, ntlm, oauth2, awsv4, etc.).
  - Converting request bodies: raw (json, text, xml), urlencoded, formdata, and binary file.
  - Converting pre-request scripts and test assertions into `model.Request.Script` and `Tests`.
  - Converting collection-level variables into `model.CollectionConfig.Vars`.
  - Dropped constructs reporting for unsupported Postman constructs.

### 2026-11-11T01:43Z — antigravity

- Did: Completed implementation of `CONV-POSTMAN`:
  - `internal/convert/postman/schema.go`:
    - Full Postman v2.0/v2.1 schema type definitions for collections, folders, items, requests, headers, URLs, query parameters, body modes, auth types, scripts, events, variables, and environment files.
  - `internal/convert/postman/import.go`:
    - Collection conversion parsing metadata, collection config, docs, collection variables, and collection auth.
    - Recursive item and folder tree mapping with sequential ordering (`seq`).
    - URL parsing handling both string and object formats, extracting query params and path params.
    - Full body conversion: raw JSON, XML, text, form urlencoded, multipart formdata with text parts and `@file(...)` parts with content types, binary files, and GraphQL queries/variables.
    - Auth conversion: basic, bearer, apikey (header/query), digest, ntlm, oauth2 (access token), awsv4, and graceful fallback with warnings for unsupported auth modes.
    - Script transpilation mapping `pm.environment.set/get` -> `bru.setEnvVar/getEnvVar`, `pm.variables.set/get` -> `bru.setVar/getVar`, `pm.collectionVariables.set/get` -> `bru.setVar/getVar`, `pm.test` -> `test`, `pm.expect` -> `expect`, and `pm.response.json()` -> `res.getBody()`.
    - Postman environment importer parsing key/value pairs and marking secret types.
  - `internal/convert/postman/export.go`:
    - Full collection exporter converting `model.Collection` into valid Postman v2.1 JSON.
    - Environment exporter converting `model.Environment` into Postman environment JSON.
  - Unit tests in `internal/convert/postman/postman_test.go`:
    - Full end-to-end import test with folders, nested requests, all body modes, variables, headers, and test scripts.
    - Auth variants test matrix (digest, ntlm, oauth2, awsv4, none, unknown).
    - Environment import, export, and roundtrip test.
    - Collection export and roundtrip fidelity test.
- Tests: `go test -race -cover ./internal/convert/postman/...` passes with 82.2% statement coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
