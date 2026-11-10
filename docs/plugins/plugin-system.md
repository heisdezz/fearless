# Plugin System

Status: **VERIFIED — there is no plugin system.** This document records the evidence and describes the closest equivalents, so that the Go port does not invent a system that does not exist.

---

## 1. Evidence

| Search | Result |
|---|---|
| `plugin`, `extension`, `addon` directories | None in any package |
| Dynamic `import()` / `require()` of user-provided modules | Only the script sandbox's `require` (whitelisted) |
| Manifest files (`plugin.json`, `bruno-plugin.*`) | None |
| Hook registry / event bus for third-party code | None |
| npm packages declared as "bruno plugins" | None referenced in code |

The only `require` that loads user code is the Node VM CJS loader (`bruno-js/src/sandbox/node-vm/cjs-loader.js`), restricted by `moduleWhitelist` and `additionalContextRoots`.

---

## 2. Closest equivalents

### 2.1 Collection-level scripts = "collection plugin"

`collection.bru` can contain `script:pre-request` / `script:post-response`, applied to every request in the collection (subject to `scripts.flow`). Combined with `bru.runRequest`, this is how teams implement shared behavior.

### 2.2 Library shims = "bundled extensions"

The QuickJS sandbox bundles lodash, moment, chai, crypto-js, cheerio, ajv, tv4, jsonwebtoken, json-query, xml2js, uuid, nanoid, yaml, etc. (`bruno-js/package.json` + `sandbox/bundle-libraries.js`).

### 2.3 Auth providers = "built-in plugins"

A fixed set (see `plugins/authentication-providers.md`) with no registration API.

### 2.4 Converters = "build-time plugins"

`bruno-converters` is a library, not a registry.

---

## 3. Why this matters for the port

If the Go implementation invents a plugin ABI, it is **not** achieving Bruno parity; it is adding scope. That can be valuable, but it must be a conscious product decision with these consequences:

| Consequence | Mitigation |
|---|---|
| Security review burden (loading arbitrary code) | Compile-time interfaces first; WASM only for pure importers |
| Cross-platform binary compatibility | Avoid `.so`/`.dll`; prefer in-tree registration |
| Versioning of plugin API | Version the interfaces explicitly from day one |
| User expectation mismatch (Bruno users have no plugins) | Teach it in docs; do not require plugins for any core feature |

---

## 4. Recommended scope for v1

```text
v1 (parity):    scripts only (no plugin ABI)
v1.1:           compile-time registration interfaces (in-tree extensions)
v2 (opt-in):    WASM importers/exporters; auth providers remain in-tree
```

Rationale: scripts already cover ~90 % of real extension needs (signing, orchestration, transforms). The remaining needs are about distribution and sharing, which is better solved by a *signed script bundle with a version and a manifest* than by arbitrary code loading.

---

## 5. Proposed "extension bundle" format (OPTIONAL)

```yaml
# .bruno-ext.yml (in a collection)
name: acme-auth
version: 1.0.0
scripts:
  pre-request: |-
    // shared signing logic
requires:
  secrets: [ACME_CLIENT_SECRET]
```

This is a **declarative artifact distributed inside a collection** — no code loading, git-friendly, reviewable. (PROPOSED; not present in Bruno.)
