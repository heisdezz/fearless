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
