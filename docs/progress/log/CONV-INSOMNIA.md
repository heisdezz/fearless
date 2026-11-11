# CONV-INSOMNIA — Insomnia export importer and converter

Owner history: antigravity (claimed 2026-11-11T01:45Z)

## Entries

### 2026-11-11T01:45Z — antigravity

- Did: Claimed `CONV-INSOMNIA`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md`.
- Tests: not run yet.
- Notes: Implementing Insomnia v4/v5 export importer in `internal/convert/insomnia/`:
  - Parsing Insomnia JSON and YAML exports (`__export_format` 4 or 5).
  - Graph/tree reconstruction linking parentId to workspace roots and request groups (folders).
  - Extracting requests, methods, URLs, query parameters, headers, auth (basic, bearer, apikey, digest, ntlm, awsv4, oauth2).
  - Extracting request bodies (raw json/xml/text, form-urlencoded, multipart formdata with file/text parts, graphql, binary file).
  - Extracting base and sub-environments into `model.Environment` and collection variables.
  - Exporting collections back into Insomnia v4 export JSON.
  - Reporting dropped or unmapped constructs.
