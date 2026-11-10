# Extension Architecture (as it actually exists)

Status: VERIFIED.

**Headline:** Bruno has **no plugin system**. There is no plugin manifest, no third-party code loading, no hook registry. Extensibility is achieved through six concrete mechanisms, documented below.

---

## 1. Mechanisms inventory

| Mechanism | What it extends | Runtime boundary |
|---|---|---|
| **1. Scripts** (`script:pre-request`, `script:post-response`, `tests`, `vars:post-response`) | Any request behavior: headers, URL, body, variables, auth headers, extra requests | Sandbox (QuickJS or Node VM) |
| **2. `bru`/`req`/`res` API** | The host surface exposed to scripts | Host objects proxied into the sandbox |
| **3. Sandbox library shims** | `require()`-able libraries (axios, uuid, nano id, jwt, path, crypto-utils; bundled lodash/moment/chai…) | Compiled into the app |
| **4. Auth providers** | Authentication schemes | Compiled into `prepare-request.js` + `configureRequest` + `.bru` grammar |
| **5. Scripted requests** | Calling other requests / raw HTTP from scripts | `bru.runRequest`, `bru.sendRequest` |
| **6. Integrations** | Vault secrets, AWS credential chain, git, mock server, AI | Compiled into `bruno-electron` |

---

## 2. Mechanism 1–3: the scripting surface

```bru
script:pre-request {
  const token = await bru.sendRequest({
    url: 'https://auth.example.com/token',
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    data: 'grant_type=client_credentials&client_id=' + bru.getEnvVar('id')
  });
  req.setHeader('authorization', 'Bearer ' + token.data.access_token);
  bru.setVar('accessToken', token.data.access_token);
}
```

Extension points in practice:

| Capability | API |
|---|---|
| Mutate outgoing request | `req.setUrl/setMethod/setHeader/deleteHeader/setBody/setTimeout/setMaxRedirects` |
| Read/modify response pre-assertions | `res.setBody` |
| Variables at every scope | `bru.*Var` family |
| Cross-request orchestration | `bru.runRequest(path)`, `bru.setNextRequest` |
| Outbound HTTP | `bru.sendRequest` |
| Credential invalidation | `bru.deleteOauth2Credential()` |
| Interpolation | `bru.interpolate(str)` |

Boundary controls:

- `nodevm` module access limited by `bruno.json → scripts.moduleWhitelist`.
- Local requires limited to `additionalContextRoots`.
- `bru.sendRequest` URL validation (`url-validation.ts`) blocks metadata/link-local targets.
- No filesystem access from QuickJS.

---

## 3. Mechanism 4: auth providers (hard-coded)

Adding a provider requires changes in **four** places (VERIFIED by tracing `apikey`):

1. Grammar: a new `auth:<name>` rule in `bruno-lang/v2/src/brunoToJson.js` and `jsonToBru.js`.
2. Type: `AuthMode` union + interface in `bruno-schema-types/src/common/auth.ts`.
3. Preparation: case in `setAuthHeaders` (`prepare-request.js`) × 2 copies (collection + request).
4. Execution: case in `configureRequest` (interceptor/adapter/token).
5. UI: form in `components/RequestPane/Auth/*`.

This is a **deliberate anti-extensibility choice** — auth touches signing correctly, so untrusted code must not be able to implement it.

---

## 4. Mechanism 5: scripted requests

See `flows/scripting-flow.md §6`. Two levels:

- `bru.sendRequest(options)` — raw request; no collection inheritance, no scripts.
- `bru.runRequest(path)` — full pipeline on a collection request.

Both produce `scripted-request` timeline entries with request/response data for auditing.

---

## 5. Mechanism 6: external integrations

| Integration | Where | Notes |
|---|---|---|
| HashiCorp Vault | `bruno-requests/src/utils/node-vault.ts` | `vars:externalsecrets:vault { path: … }` in env files |
| AWS credentials | `@aws-sdk/credential-providers` | Env, shared file, profile |
| `.env` files | `store/process-env.js`, `app/dotenv-watcher.js` | Feeds `process.env.*` |
| Git | `ipc/git.js` | Status/diff/commit UI |
| Mock server | `ipc/mock-server/` | Serve routes from a spec |
| AI assistant | `ipc/ai/` | Out of scope for the port |

---

## 6. What cannot be extended

| Aspect | Status |
|---|---|
| Adding a body mode | Not possible without core changes |
| Adding an export/import format | Not possible without recompiling `bruno-converters` |
| Adding a scripting runtime | Only `quickjs` / `nodevm` |
| Adding a response viewer | Not possible |
| UI extensions | None (no slots, no custom panels) |

---

## 7. Assessment

**Strengths.** The scripting surface is genuinely powerful: with `req` mutation plus `bru.sendRequest`, users can implement token exchange, HMAC signing, pagination loops, and data-driven flows without touching the app.

**Weaknesses.**

1. **Auth providers are closed** — the most requested extension class (custom signing schemes) requires a fork.
2. **No hooks for transport** (custom `DNS`, custom `RoundTripper`-like middleware, custom body encoders).
3. **No declarative extension distribution** (no way to share a "plugin" with a team other than copying a collection).
4. **Scripts are per-request/per-folder**, so cross-cutting behavior must be duplicated or placed in `collection.bru` scripts.

---

## 8. Recommended Go extensibility (summary — details in `plugins/extensibility-proposal.md`)

1. Keep scripts as the primary extension mechanism (parity, zero friction).
2. Add **in-process, compile-time** interfaces: `AuthProvider`, `BodyEncoder`, `TransportMiddleware`, `Importer`, `Exporter`, `HTTPHook`.
3. Keep the **out-of-process/Untrusted** surface restricted to the script sandbox.
4. Do **not** add dynamic plugin loading (`.so`/WASM) in v1; it complicates signing and sandbox guarantees.
5. Future: optional WASM plugins for importers/exporters only (no network, no secrets).
