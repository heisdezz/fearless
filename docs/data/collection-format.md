# Collection Format

Status: VERIFIED unless marked.

Bruno supports **two on-disk formats** selected per collection: `bru` (default/historical) and `yml` (OpenCollection, the migration target). This document describes the directory model common to both; format-specific syntax is in `bru-format.md` and the YAML sections here.

---

## 1. Directory layout

```text
<collection-root>/
├── bruno.json                 # bru-format collection config
│   # or opencollection.yml    # yml-format collection config
├── collection.bru             # bru-format root (headers, auth, vars, scripts, tests, docs)
│   # or collection.yml
├── .env                       # optional; → process.env.* (collection scope)
├── environments/
│   ├── Local.bru              # or Local.yml
│   └── Prod.bru
├── folder-a/
│   ├── folder.bru             # or folder.yml
│   ├── request-one.bru        # or request-one.yml
│   ├── request-one/           # optional sibling dir holding examples/ (INFERRED from example UID scheme)
│   └── nested-folder/
│       └── …
└── folder-b/
```

Resolution rules (VERIFIED, `utils/collection.js` `FORMAT_CONFIG`):

```js
bru: { ext: '.bru', collectionFile: 'collection.bru', folderFile: 'folder.bru' }
yml: { ext: '.yml', collectionFile: 'opencollection.yml', folderFile: 'folder.yml' }
```

- A file without an extension is normalized by appending the format's extension (`hasRequestExtension`).
- Ignored when watching: `node_modules`, `.git`.
- Files > 2.5 MB are parsed with large text blocks redacted (placeholders) to bound memory; restored on write.

---

## 2. Collection config — `bruno.json`

```jsonc
{
  "version": "1",
  "name": "my-collection",
  "type": "collection",

  "proxy": {
    "inherit": true, // true → use app preferences
    "disabled": false, // optional
    "config": {
      "protocol": "http",
      "hostname": "proxy.local",
      "port": 4000,
      "auth": { "username": "u", "password": "p", "disabled": true },
      "bypassProxy": "localhost,.internal",
    },
  },

  "scripts": {
    "runtime": "quickjs", // or "nodevm"
    "moduleWhitelist": ["crypto", "buffer", "form-data"],
    "additionalContextRoots": ["../shared-lib"],
    "flow": "sandwich", // script merge order
  },

  "clientCertificates": {
    "enabled": true,
    "certs": [
      {
        "domain": "*.example.com",
        "type": "cert",
        "certFilePath": "certs/client.crt",
        "keyFilePath": "certs/client.key",
        "passphrase": "",
        "disabled": false,
      },
    ],
  },

  "presets": { "requestType": "http", "requestUrl": "http://localhost:6000" },
}
```

Validated by `configSchema` (Yup) in `bruno-electron/src/app/collections.js` line 19: `name` (≤256 chars, required), `type` (must be `'collection'`), optional `version`, `presets{requestType, requestUrl, defaultEnvironment}`, `opencollection` (for yml collections), and `openapi[]` sync entries (`sourceUrl`, `lastSyncDate`, `specHash`, `groupBy: tags|path`, `autoCheck`, `autoCheckInterval`). No `noUnknown()` is applied, so extra keys (proxy, scripts, clientCertificates) pass validation and are consumed by other code paths.

### `opencollection.yml` (yml format)

```yaml
opencollection: "1.0.0"
info:
  name: my-collection
# runtime/extensions blocks hold the yml equivalents of scripts/proxy/certs
```

Verified fixture: `packages/bruno-tests/workspaces/onfail/onfail-collection/opencollection.yml`.

---

## 3. Root / folder configuration

| Concept            | `.bru`                                                | `.yml`                          |
| ------------------ | ----------------------------------------------------- | ------------------------------- |
| Name, seq          | `meta { name, seq, type }`                            | `info: { name, seq, type }`     |
| Shared headers     | `headers { … }`                                       | `http.headers` (folder)         |
| Shared query       | `query { … }`                                         | `http.params`                   |
| Shared auth        | `auth { mode }` + `auth:<mode> { … }`                 | `http.auth`                     |
| Pre-request vars   | `vars:pre-request { … }`                              | `variables`                     |
| Post-response vars | `vars:post-response { … }`                            | `actions`                       |
| Scripts            | `script:pre-request`, `script:post-response`, `tests` | `runtime.scripts[]` with `type` |
| Assertions         | `assert { … }`                                        | `runtime.assertions`            |
| Docs               | `docs { … }`                                          | `info.description`              |

YAML script `type` values (VERIFIED, `formats/yml/common/scripts.ts`):

```text
before-request | after-response | tests
grpc:before-call-start | grpc:before-message-send | grpc:after-message-receive | grpc:after-call-end
```

---

## 4. Item model (canonical)

From `bruno-schema-types/src/collection/item.ts`:

```ts
interface Item {
  uid: string;
  type:
    | "http-request"
    | "graphql-request"
    | "folder"
    | "js"
    | "app"
    | "grpc-request"
    | "ws-request";
  seq?: number | null;
  name: string;
  description?: string | null;
  tags?: string[] | null;
  request?: HttpRequest | GrpcRequest | WsRequest | null;
  settings?: HttpItemSettings | WebSocketItemSettings | null;
  app?: { code?: string; enabled?: boolean } | null;
  fileContent?: string | null;
  root?: FolderRoot | null; // set on folders & collection root
  items?: Item[] | null; // children
  examples?: Example[] | null;
  filename?: string | null;
  pathname?: string | null;
}
```

`uid` is **not stored on disk** for requests — it is derived from the pathname (`getRequestUid` in `cache/requestUids.js`), so renaming a file changes its UID (and open tabs must remap).

---

## 5. Ordering (`seq`)

- `meta.seq` defaults to `1` when absent.
- Items are sorted by name alphabetically, then by `seq` for equal names (VERIFIED: `sortByNameThenSequence`).
- Reordering rewrites `seq` across the affected folder via `renderer:resequence-items`.

---

## 6. Environment files

`environments/<Name>.bru` or `.yml`. See `environment-format.md`. Secret values live in an encrypted app store, not in these files.

---

## 7. Examples

Stored inside the request file as `example { … }` blocks (bru) or an `examples` array (yml). They contain a saved request variant and/or response.

---

## 8. Encoding and newlines

- Files are UTF-8.
- The parser accepts `\r\n` and `\n` (`nl = "\r"? "\n"` in the grammar).
- `normalizeLineEndings` is used when comparing scripts/tests (`@usebruno/common`).

---

## 9. Round-trip fidelity

| Direction               | Guarantee                                                                                                    |
| ----------------------- | ------------------------------------------------------------------------------------------------------------ |
| `.bru` → model → `.bru` | Preserves unknown-free structure; comments in the DSL are not represented in the model                       |
| `.yml` → model → `.yml` | Field order normalized by the stringifier; `encodeUrl`, `followRedirects` default to `true` when unspecified |
| Cross-format            | `yml` ⇄ `bru` conversions exist via the model; `graphql` body handled by the GraphQL-specific stringifier    |

---

## 10. Go implementation (PROPOSED)

```go
package collection

type Collection struct {
    Root     string          // absolute path
    Format   Format          // Bru | YAML
    Config   Config          // bruno.json / opencollection.yml
    RootItem *FolderRoot     // collection.bru equivalent
    Items    []*Item         // ordered
    Envs     []*Environment
}

type Item struct {
    UID      string
    Type     ItemType
    Name     string
    Seq      int
    Tags     []string
    Request  *Request        // nil for folders
    Root     *FolderRoot     // non-nil for folders
    Children []*Item
    Examples []*Example
    Path     string
}

type Codec interface {
    Ext() string
    ParseRequest([]byte) (*Item, error)
    WriteRequest(*Item) ([]byte, error)
    ParseFolder([]byte) (*FolderRoot, error)
    WriteFolder(*FolderRoot) ([]byte, error)
    ParseEnvironment([]byte) (*Environment, error)
    WriteEnvironment(*Environment) ([]byte, error)
}
```

Preserve: format autodetection by the presence of `opencollection.yml`, UID-from-path derivation, `seq` ordering, size-based redaction.
