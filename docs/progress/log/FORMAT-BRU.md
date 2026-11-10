# Log — `FORMAT-BRU`

Owner history: zed-agent-1 (claimed 2026-11-10T23:15Z, moved to InReview 2026-11-10T23:55Z)

Package: `internal/core/format/bru/**`. Depends on `internal/core/model` and `internal/core/format`.

---

## Entries

### 2026-11-10T23:55Z — zed-agent-1

- **Did:** Implemented the `.bru` dialect: parser and model mapper, plus the `Codec` wiring.
  - `ast.go` — `Document`, `Block` (dict/text/list), `Pair`, `Annotation`, `SyntaxError` with line:col.
  - `parser.go` — hand-written recursive-descent parser: blocks, dictionaries, text bodies, `'''` multiline values, annotations, `~` disabled rows, quoted keys, lists, `extends`/`color` top-level rules.
  - `names.go` — block classification (text vs dictionary), method-block table, closed-grammar check.
  - `mapper.go` — `Document` → `model.Request`, including body modes, multipart annotation extraction, scripts, vars, assertions, settings.
  - `auth.go` — `auth { mode }` + `auth:<mode>` → the auth union for all 12 modes.
  - `codec.go` — `format.Codec` implementation for request/folder/environment/collection-config.
- **Tests:** 45 test functions across `parser_test.go`, `mapper_test.go`, `upstream_test.go`. `go test -race` passes; **79.3 % statement coverage**; `gofmt` and `go vet` clean.
- **Conformance:** **357 of 357** v2 `.bru` files in the Bruno clone parse *and* map with zero failures. Four legacy v1 files are detected and skipped (`isV1Dialect`); the test fails if more than a quarter of the corpus is classified v1, so a broken detector cannot silently pass.

### Grammar rules that took empirical work to get right

These are the parts a from-scratch implementation gets wrong, recorded so nobody has to rediscover them:

| Rule | Why it matters |
| --- | --- |
| **Text-block terminator is `nl "}"` with the brace at column 0** | An indented `  }` inside a JS script or JSON body is *content*. My first attempt accepted indented braces and truncated every script containing an object literal. |
| **Terminator selection is backtracking, not greedy-longest** | `textblock = textline (~tagend nl textline)*` combined with the column-0 rule; the first viable terminator wins. Taking the *last* one made block 1 swallow blocks 2–5. |
| **Text bodies are outdented by 2 spaces** (`outdentString(str, 2)`), and the newline after `{` is not part of the body | Verified in `bruToJson.js`. |
| **`'''` multiline values strip exactly 4 characters per line** (`line.slice(4)`), keep the leading newline, and empty short lines | Different rule from text bodies; both are implemented separately. |
| **`assert` is a dictionary, not a text block** | It uses `assertdictionary`, whose keys may contain dots (`res.body.items`) and whose values may contain spaces (`lt 1000`). Classifying it as text silently produced zero assertions. |
| **`tags: [ … ]` is a list-valued pair**, newline-separated | `value = list \| multilinetextblock \| singlelinevalue`. Missing this failed 13 corpus files. |
| **Blank `maxRedirects:` means unset, not 0** | Bruno distinguishes them; `ItemSettings` uses pointers so inheritance still works. |

### Notes

- `Document.Request()` never fails on missing blocks: defaults are method GET, body none, auth nil. This matches Bruno, where a `.bru` file with only `meta` is valid.
- Row UIDs (`rowUID`) are derived from kind+index+name length, so re-parsing the same file yields the same UIDs without the format persisting them. Request UIDs come from the path and are assigned by `COLLECTION`.
- Multipart annotations live inline in the value (`@file(x) @contentType(y)`); `extractAnnotation`/`removeAnnotation` pull them out and the remainder becomes the part's value.
- The closed-grammar check (`knownBlock`) exists but is not yet enforced during parse; enforcing it is a one-line change once the v1 path is decided.

### Not done in this package

- **The writer.** `WriteRequest`, `WriteFolder`, `WriteEnvironment` return a "not implemented yet" error. Writing is a separate concern from parsing (ordering, `~` prefixes, `'''` re-quoting, multiline re-indentation) and is the natural next unit of work. Recorded so `COLLECTION` and the export path are not blocked unexpectedly.
- **gRPC/WebSocket bodies** (`body:grpc`, `body:ws`) parse as text blocks but are not mapped onto the model, which has no fields for them yet.
- **v1 dialect** is detected and skipped, not parsed.

- **Open:** awaiting review. `FORMAT-YAML` and `COLLECTION` are unblocked for reading; anything that writes `.bru` needs the writer first.
