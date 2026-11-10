# Log — `FORMAT-YAML`

Owner history: zed-agent-1 (claimed 2026-11-11T00:30Z, moved to InReview 2026-11-11T01:05Z)

Package: `internal/core/format/yamlcodec/**`. Depends on `internal/core/model` and `internal/core/format`.

Note on the package name: it is `yamlcodec`, not `yaml`, so the file can import `gopkg.in/yaml.v3` under its own name.

---

## Entries

### 2026-11-11T01:05Z — zed-agent-1

- **Did:** Implemented the OpenCollection YAML dialect.
  - `codec.go` — `format.Codec` implementation, detection via `opencollection.yml`, an encoder wrapper fixing two-space indentation.
  - `doc.go` — YAML document structs mirroring Bruno's schema for collections, requests, folders and environments.
  - `value.go` — `yamlValue`, which accepts a scalar **or** a `{type, data}` mapping for the same field.
  - `mapper.go` — YAML document → model for requests, bodies, auth, runtime variables, scripts and settings.
  - `writer.go` — model → YAML document.
  - `methods.go` — the `Codec` methods.
- **Tests:** 24 test functions. `go test -race` passes; **86.5 % statement coverage**; `gofmt`/`go vet` clean.
- **Conformance:** **64 of 64** `.yml` fixtures in the Bruno clone parse with zero failures (collections, folders and requests).

### The format's awkward part: three body shapes for one concept

Bruno writes request bodies in three different shapes, and all three appear in its own fixtures:

```yaml
# shape 1 — mode + named payload key
body:
  mode: json
  json: |
    {}

# shape 2 — type + scalar data
body:
  type: json
  data: |-
    {}

# shape 3 — type + row list
body:
  type: form-urlencoded
  data:
    - name: a
      value: b
```

`bodyDoc.Data` is therefore a `yaml.Node` **value** rather than `[]kvDoc` or `string`, with `dataText()` and `dataRows()` accessors. Consequences worth knowing:

- It must be a value, not `*yaml.Node`: the decoder writes the node in place, and a pointer field rejects node kinds other than the one it was allocated for (it failed to decode a sequence into a pre-allocated scalar node).
- The writer builds sequence nodes by hand (`rowsNode`), because a struct field cannot be both a string and a list.

### Other notes

- Auth field names are identical in both dialects, so the mapping is a straight translation. Only `type:` is used here where `.bru` uses the block name.
- Tests are runtime scripts of `type: tests` in this dialect, not a separate block. `testsFromRuntime` pulls them out so `model.Request.Tests` means the same thing in both dialects — this is the one place the two formats differ structurally.
- An unrecognised body mode maps to `BodyNone` rather than being guessed, so an unknown future mode cannot silently produce a wrong body.
- `Register()` follows the same explicit-wiring rule as the `.bru` codec (no `init()`).

- **Open:** awaiting review. Both codecs now implement `format.Codec`, so `COLLECTION` is unblocked.
