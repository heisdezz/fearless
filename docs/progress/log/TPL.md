# Log — `TPL`

Owner history: zed-agent-1 (claimed 2026-11-11T02:25Z, moved to InReview 2026-11-11T02:50Z)

Package: `internal/core/template/**`. Depends on `model` and `vars`. **Does not** depend on goja.

---

## Entries

### 2026-11-11T02:50Z — zed-agent-1

- **Did:** Implemented the declarative variable blocks.
  - `PreRequest` — interpolates each enabled variable against scopes, honours `@number` / `@boolean` / `@object` annotations, and reports names whose placeholders survived.
  - `PostResponse` — evaluates `$res.<path>`, `{{res.<path>}}`, `{{$dynamic}}` and plain literals against a `Response` value.
  - `Result` — produced values, `Lookup`, `AsMap`, and an `Unresolved` list.
- **Tests:** 18 test functions plus a dependency assertion. `go test -race` passes; **89.5 % statement coverage**; `gofmt`/`go vet` clean.
- **Acceptance criteria:**
  1. *Block semantics match `flows/scripting-flow.md`* — pre-request interpolates; post-response resolves against the response. See the caveat below, which corrects the doc.
  2. *No goja import* — enforced by `TestNoGojaDependency`, which runs `go list -deps` and fails if `dop251/goja` appears. Verified: the package's dependency graph contains zero goja packages.

### What the fixtures actually contain (and why it changed the design)

`packages.md` scoped `TPL` as "kept separate from JS scripting". Inspecting Bruno's own collections showed **two** live forms for `vars:post-response`:

```bru
vars:post-response {
  token: $res.body.access_token        # an expression
  firstId: {{res.body.items[0].id}}    # a template
  now: {{$timestamp}}                  # a dynamic variable
  ~transactionId: $res.body.transactionId
}
```

The `{{…}}` forms need no engine, and `$res.body.access_token` is a dotted path that `vars.Lookup` already resolves — so both common cases are handled natively. That is a strictly better outcome than either "needs goja" or "only handles templates", and it keeps `VARS` and `BUILD` free of the engine.

The one form that genuinely needs JavaScript is an arbitrary expression such as `bru.getVar('x') + 1`. `PostResponse` does **not** attempt it: the value is reported in `Unresolved` so `SCRIPT-POST` can hand it to the engine. `TestPostResponseUnsupportedExpressionIsReported` exists to stop a future contributor from "helpfully" adding a partial evaluator that would produce quietly wrong variables.

### Correction to `docs/flows/scripting-flow.md`

That document describes both blocks as running through `VarsRuntime` in JS. That is accurate for Bruno and for genuine expressions, but it is not the only correct implementation: the declarative subset is well-defined without an engine. Recorded as D-010 rather than editing the analysis doc, since the doc describes Bruno and remains true.

### Bugs found while testing

- **Unresolved detection was dead code.** The first version compared the interpolated value against the original, which never fires when the *input* is a placeholder. It now checks whether the source contained a placeholder that survived interpolation. Without the fix, a typo'd variable silently produced an empty string.

### Not in this package

- Arbitrary JavaScript in either block — `SCRIPT-POST`.
- JSON parsing of the response body. `PostResponse` takes `Response.Body` already decoded; extracting it from a `BodyRef` is `ENGINE`'s job, which is why the type is `any`.
- `vars:externalsecrets:vault` (Vault-sourced secrets). It parses in `FORMAT-BRU` but has no consumer yet.

- **Open:** awaiting review. `ENGINE` and `SCRIPT-POST` can call `template.PreRequest` / `template.PostResponse` directly.
