# Request Schema

Status: VERIFIED. Canonical TypeScript source: `packages/bruno-schema-types/src/requests/http.ts`, `common/*.ts`, `collection/item.ts`.

---

## 1. HTTP request

```ts
interface HttpRequest {
  url: string;
  method: string; // upper-case by convention; custom methods allowed
  headers: KeyValue[];
  params: HttpRequestParam[]; // query + path
  auth?: Auth | null;
  body?: HttpRequestBody | null;
  script?: HTTPScripts | null; // { req?, res? }
  vars?: { req: Variables; res: Variables } | null;
  assertions?: KeyValue[] | null;
  tests?: string | null;
  docs?: string | null;
}
```

---

## 2. Key/value

```ts
interface KeyValue {
  uid: string;
  name?: string | null;
  value?: string | null;
  description?: string | null;
  enabled?: boolean;
  annotations?: Annotation[] | null;
}
```

Used for headers, query/path params, form bodies, assertions and cookies in the UI model.

**Sorted nowhere, ordered everywhere.** The order of `headers` and `params` is the user's order and is preserved to disk and to the wire.

---

## 3. Params

```ts
interface HttpRequestParam extends KeyValue {
  type: "query" | "path";
}
```

- `query`: mirrored from the URL query string by the reducer.
- `path`: derived from `:name` segments in the URL.

---

## 4. Body

```ts
type HttpRequestBodyMode =
  | "none"
  | "json"
  | "text"
  | "xml"
  | "formUrlEncoded"
  | "multipartForm"
  | "graphql"
  | "sparql"
  | "file";

interface HttpRequestBody {
  mode: HttpRequestBodyMode;
  json?: string | null;
  text?: string | null;
  xml?: string | null;
  sparql?: string | null;
  formUrlEncoded?: KeyValue[] | null;
  multipartForm?: MultipartForm | null;
  graphql?: { query?: string | null; variables?: string | null } | null;
  file?: FileList | null;
}
```

`MultipartFormEntry`:

```ts
interface MultipartFormEntry {
  uid: string;
  type: "file" | "text";
  name?: string | null;
  value?: string | string[] | null; // string[] for multi-file
  description?: string | null;
  contentType?: string | null; // per-part type
  enabled?: boolean;
  annotations?: Annotation[];
}
```

`FileEntry`:

```ts
interface FileEntry {
  uid: string;
  filePath?: string | null; // relative to the collection dir
  contentType?: string | null;
  description?: string | null;
  selected: boolean; // exactly one should be selected
  annotations?: Annotation[];
}
```

---

## 5. Variables

```ts
interface Variable {
  uid: string;
  name?: string | null;
  value?: string | number | boolean | Record<string, unknown> | null;
  description?: string | null;
  enabled?: boolean;
  local?: boolean;
  dataType?: "string" | "number" | "boolean" | "object";
  annotations?: Annotation[] | null;
}
```

---

## 6. Scripts

```ts
interface HTTPScripts {
  req?: string | null;
  res?: string | null;
}
```

gRPC adds `beforeCallStart`, `beforeMessageSend`, `afterMessageReceive`, `afterCallEnd`.

---

## 7. Auth

See `data/bru-format.md §8` for the field list per mode, and `logic/authentication.md` for semantics. Discriminator: `mode: AuthMode`.

---

## 8. Item settings

```ts
interface HttpItemSettings {
  encodeUrl?: boolean | null; // default false
  followRedirects?: boolean | null; // default true
  maxRedirects?: number | null; // default 5
  timeout?: number | "inherit" | null; // inherit → global preference
  forwardAuthorizationHeader?: boolean | null; // default true
  omitHeaders?: string[] | null; // suppress transport defaults
}
```

---

## 9. Wrapper item

```ts
interface Item {
  uid: string; // derived from pathname in Bruno, not persisted for requests
  type:
    | "http-request"
    | "graphql-request"
    | "folder"
    | "grpc-request"
    | "ws-request"
    | "js"
    | "app";
  seq?: number | null;
  name: string;
  description?: string | null;
  tags?: string[] | null;
  request?: HttpRequest | null;
  settings?: HttpItemSettings | null;
  app?: { code?: string | null; enabled?: boolean | null } | null;
  root?: FolderRoot | null; // folders
  items?: Item[] | null; // folders
  examples?: Example[] | null;
  filename?: string | null;
  pathname?: string | null;
}
```

---

## 10. Runtime-only fields (added during execution)

Not persisted; attached by `prepareRequest` / `runRequest`:

```text
collectionVariables, folderVariables, requestVariables,
globalEnvironmentVariables, oauth2CredentialVariables, promptVariables,
pathParams, tags, testsMetadata, __explicitHeaderNames,
__bruno__executionMode, responseType, signal,
basicAuth | awsv4config | digestConfig | ntlmConfig | oauth1config | oauth2,
apiKeyHeaderName, apiKeyAuthValueForQueryParams, edgeGridConfig,
certsAndProxyConfig, _originalMultipartData, collectionPath
```

**Go note:** keep these out of the persisted struct entirely. Use a `CompiledRequest` type.

---

## 11. Go structs (PROPOSED)

```go
type Request struct {
    URL        string        `json:"url"`
    Method     string        `json:"method"`
    Headers    []KeyValue    `json:"headers"`
    Params     []Param       `json:"params"`
    Auth       *Auth         `json:"auth,omitempty"`
    Body       *Body         `json:"body,omitempty"`
    Script     *Scripts      `json:"script,omitempty"`
    Vars       *Vars         `json:"vars,omitempty"`
    Assertions []KeyValue    `json:"assertions,omitempty"`
    Tests      string        `json:"tests,omitempty"`
    Docs       string        `json:"docs,omitempty"`
}

type KeyValue struct {
    UID         string `json:"uid,omitempty"` // omit on disk; generate on load
    Name        string `json:"name"`
    Value       string `json:"value"`
    Description string `json:"description,omitempty"`
    Enabled     bool   `json:"enabled"`
}

type Param struct {
    KeyValue
    Type string `json:"type"` // query|path
}
```

Use pointer/slice presence to distinguish "absent" from "empty" where the serializer does (e.g. `auth`).
