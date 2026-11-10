# `.bru` Format Specification

Status: VERIFIED — derived from the `ohm` grammar in `packages/bruno-lang/v2/src/bruToJson.js`, `collectionBruToJson.js`, `envToJson.js`, the writers `jsonToBru.js` / `jsonToCollectionBru.js` / `jsonToEnv.js`, and real fixtures in `packages/bruno-tests/collection/`.

This is the reference a Go parser should be written against.

---

## 1. Grammar (top level)

```text
BruFile = (meta | http | grpc | ws | query | params | headers | metadata
          | auths | bodies | varsandassert | script | tests | app
          | settings | docs | example)*
```

Blocks may appear in any order; the parser collects them. Unknown blocks are a **parse error** (the grammar is closed).

---

## 2. Lexical elements

| Concept                 | Rule                                                                                    |
| ----------------------- | --------------------------------------------------------------------------------------- |
| Newline                 | `\r?\n`                                                                                 |
| Whitespace              | space or tab                                                                            |
| Dictionary              | `{ … }` closed by newline + `}`                                                         |
| List                    | `[` newline items (newline)+ `]`                                                        |
| Single-line value       | everything to end of line, trimmed                                                      |
| Multiline value         | `'''` … `'''`, each line de-indented by **4 spaces**                                    |
| Content-type annotation | `'''…''' @contentType(text/plain)`                                                      |
| Disabled key            | prefix `~` → parses with `enabled: false`                                               |
| Quoted key              | `"key:with:colons": value`                                                              |
| Annotation              | `@name`, `@name(arg)`, args may be `'single'`, `"double"`, unquoted, or a `'''block'''` |

Annotations precede a pair and apply to it:

```bru
headers {
  @description('''Sent to identify the caller''')
  @required
  x-api-key: {{apiKey}}
}
```

Known annotation effects (VERIFIED):

| Annotation                         | Effect                           |
| ---------------------------------- | -------------------------------- |
| `@description('…')`                | Sets `description` on the entry  |
| `@number` / `@boolean` / `@object` | Sets `dataType` on variables     |
| `@contentType(…)`                  | Multipart/file part content type |

---

## 3. `meta` block

```bru
meta {
  name: Get Users
  type: http
  seq: 2
  tags: [
    smoke
    regression
  ]
}
```

| Key    | Type                                  | Default                  |
| ------ | ------------------------------------- | ------------------------ |
| `name` | string                                | — (required in practice) |
| `type` | `http` \| `graphql` \| `grpc` \| `ws` | `http`                   |
| `seq`  | int                                   | `1`                      |
| `tags` | list                                  | —                        |

`tags` is serialized specially by the writer as a bracketed list.

> Note: `meta` values are scalar in the grammar (`key: value`); `tags` is written by the custom writer, and the reader handles it via the list rule.

---

## 4. Method block

```bru
get {
  url: {{host}}/users
  body: json
  auth: inherit
}
```

- One of `get post put patch delete head options trace connect` or `http { method: <custom> }`.
- Keys: `url` (required), `body` (body mode name), `auth` (auth mode name).
- `body` and `auth` are **references** to the corresponding blocks; they are omitted when unused.

---

## 5. `params`

```bru
params:query {
  page: 1
  limit: 20
  ~draft: true
}

params:path {
  id: {{userId}}
}
```

Read as items with `type: 'query' | 'path'`. Disabled rows use `~`.

> In practice the **URL is the source of truth** for query params (see `ui/request-editor.md §2`); the `query` block is the legacy form and is still parsed.

---

## 6. `headers`

```bru
headers {
  content-type: application/json
  authorization: Bearer {{token}}
  ~x-debug: 1
}
```

Order preserved into `headerEntries`; `~` marks disabled.

---

## 7. Bodies

| Block                                      | Mode               |
| ------------------------------------------ | ------------------ |
| `body:json { … }`                          | `json`             |
| `body:text { … }`                          | `text`             |
| `body:xml { … }`                           | `xml`              |
| `body:sparql { … }`                        | `sparql`           |
| `body:graphql { … }`                       | `graphql` query    |
| `body:graphql:vars { … }`                  | graphql variables  |
| `body:form-urlencoded { k: v }`            | formUrlEncoded     |
| `body:multipart-form { k: v }`             | multipartForm      |
| `body:file { file: […], content-type: … }` | file               |
| `body:grpc { … }`                          | gRPC message       |
| `body:ws { … }`                            | WebSocket messages |

Text bodies are **raw text**: no escaping, terminated by the closing `}` on its own line.

Multipart example, copied from the real fixture `packages/bruno-tests/collection/multipart/multipart-mixed-form-data-file.bru`:

```bru
body:multipart-form {
  sample: @file(bruno.png) @contentType(image/png)
  foo: {"bar":"baz"} @contentType(application/json--test)
  multiline: '''
    "multiline-test"
  ''' @contentType(application/json--multiline--test)
}
```

File parts use the `@file(relativePath)` annotation; `@contentType(…)` sets the part's type. A plain `name: value` pair is a text part. Multi-file values and the exact annotation grammar live in `bruno-lang/v2/src/bruToJson.js` (`multipartExtractContentType`) and `bruno-electron/src/utils/form-data.js`.

---

## 8. Auth blocks

```bru
auth {
  mode: bearer
}

auth:bearer {
  token: {{token}}
}

auth:basic {
  username: {{user}}
  password: {{pass}}
}
```

| Block                                                                        | Fields (`bruno-schema-types/src/common/auth.ts`)                                                                                                                                                                                                                                |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `auth:basic`                                                                 | username, password                                                                                                                                                                                                                                                              |
| `auth:bearer`                                                                | token                                                                                                                                                                                                                                                                           |
| `auth:digest`                                                                | username, password                                                                                                                                                                                                                                                              |
| `auth:ntlm`                                                                  | username, password, domain                                                                                                                                                                                                                                                      |
| `auth:awsv4`                                                                 | accessKeyId, secretAccessKey, sessionToken, service, region, profileName                                                                                                                                                                                                        |
| `auth:apikey`                                                                | key, value, placement (`header`/`queryparams`)                                                                                                                                                                                                                                  |
| `auth:wsse`                                                                  | username, password                                                                                                                                                                                                                                                              |
| `auth:oauth1`                                                                | consumerKey, consumerSecret, accessToken, accessTokenSecret, callbackUrl, verifier, signatureMethod, privateKey, privateKeyType, timestamp, nonce, version, realm, placement, includeBodyHash                                                                                   |
| `auth:oauth2`                                                                | grantType, username, password, callbackUrl, authorizationUrl, accessTokenUrl, clientId, clientSecret, scope, state, pkce, credentialsPlacement, credentialsId, tokenPlacement, tokenHeaderPrefix, tokenQueryKey, refreshTokenUrl, autoRefreshToken, autoFetchToken, tokenSource |
| `auth:oauth2:additional_params:auth_req:{headers,queryparams}`               | additional authorization params                                                                                                                                                                                                                                                 |
| `auth:oauth2:additional_params:access_token_req:{headers,queryparams,body}`  | additional token params                                                                                                                                                                                                                                                         |
| `auth:oauth2:additional_params:refresh_token_req:{headers,queryparams,body}` | additional refresh params                                                                                                                                                                                                                                                       |
| `auth:akamai-edgegrid`                                                       | accessToken, clientToken, clientSecret, baseURL, nonce, timestamp, headersToSign, maxBodySize                                                                                                                                                                                   |

`mode` is one of: `inherit`, `none`, `basic`, `bearer`, `digest`, `ntlm`, `oauth1`, `oauth2`, `wsse`, `apikey`, `awsv4`, `akamai-edgegrid`.

---

## 9. Variables and assertions

```bru
vars:pre-request {
  userId: 42
  @number
  pageSize: 20
}

vars:post-response {
  token: res.body.access_token
  firstId: res.body.items[0].id
}

assert {
  res.status: 200
  res.body.items: isArray
  res.responseTime: lt 1000
}
```

- `vars:post-response` values are **JS expressions**, not interpolated templates.
- `assert` uses the `lhs: operator operand` syntax (`logic/assertions-and-tests.md`).

---

## 10. Scripts

```bru
script:pre-request {
  const t = await bru.sendRequest({ url: '…' });
  bru.setVar('tok', t.data.token);
}

script:post-response {
  console.log(res.getStatus());
}

tests {
  test('ok', () => { expect(res.getStatus()).to.equal(200); });
}
```

gRPC hooks: `script:grpc:before-call-start`, `script:grpc:before-message-send`, `script:grpc:after-message-receive`, `script:grpc:after-call-end`.

---

## 11. Docs, settings, app

```bru
docs {
  # Users API

  Returns a paginated list of users.
}

settings {
  encodeUrl: true
  timeout: 0
  followRedirects: true
  maxRedirects: 5
  forwardAuthorizationHeader: true
}

app {
  code: console.log('custom');
  enabled: true
}
```

Settings parsing notes (VERIFIED in `bruToJson.js`):

- `followRedirects`, `forwardAuthorizationHeader` → booleans via `toBool`.
- `maxRedirects` → `Number`, `Math.trunc`, must be `>= 0`; **blank means unset** (not `0`).
- `timeout` → integer or the literal `'inherit'`; default `0`.
- `encodeUrl` → boolean.

---

## 12. Examples

```bru
example {
  name: 200 OK
  request {
    url: {{host}}/users
    method: GET
  }
  response {
    status: 200
    body:json {
      []
    }
  }
}
```

Parsed by `bruno-lang/v2/src/example/bruToJson.js` (a nested mini-grammar).

---

## 13. Collection root file (`collection.bru`)

Grammar: `BruFile = (meta | query | headers | auth | auths | vars | script | tests | docs)*`

Same blocks as requests minus the method/body/params; adds shared `headers`, `query`, `auth`, `vars:pre-request`, scripts and tests that apply to every request in the collection.

---

## 14. Folder file (`folder.bru`)

```bru
meta {
  name: digest
  seq: 14
}

auth {
  mode: inherit
}
```

Same grammar as `collection.bru`.

---

## 15. Environment file (`environments/*.bru`)

```bru
vars {
  host: http://localhost:8080
  @number
  retries: 3
  multiline: '''
    line1
    line2
  '''
}

vars:secret [
  clientSecret,
  clientId
]

vars:externalsecrets:vault {
  path: secret/data/bruno
}

extends: Base-Environment

extends [
  Shared,
  "Parent With Spaces"
]

color: #ff0000
```

`extends` is a top-level rule, not a block: either `extends: <name>` (`extendsname = ":" singlelinechar*`) or a bracketed list `extends [ a, "b c" ]` with comma separators and quoting for names containing `,`, `[`, `]` or `"`. `color:` is a top-level `color: <hex>` line. Both verified in `bruno-lang/v2/src/envToJson.js`.

---

## 16. Parsing edge cases (VERIFIED from grammar)

| Case                        | Rule                                               |
| --------------------------- | -------------------------------------------------- |
| Values containing `:`       | Allowed (value runs to EOL)                        |
| Keys containing `:`         | Must be quoted                                     |
| Empty dictionary            | `headers {\n}` valid                               |
| Trailing spaces             | Trimmed from single-line values                    |
| CRLF                        | Accepted                                           |
| Multiline indentation       | Exactly 4 spaces stripped per line                 |
| Duplicate keys              | Last wins in a map; `headerEntries` preserves both |
| `~` prefix on an empty name | Ignored                                            |

---

## 17. Writing (serialization order)

`jsonToBru` emits blocks in a fixed order: `meta`, method/http, `grpc`, `ws`, `params:query`, `params:path`, `headers`, `metadata`, auth, body, scripts, tests, vars, assertions, settings, app, docs, examples. Disabled items are written with a `~` prefix (`enabled(...)` / `disabled(...)` helpers).

---

## 18. Go parser plan (PROPOSED)

```go
package bru

type Parser struct {
    src []byte
    pos int
    line int
    col  int
}

// Block dispatch
func (p *Parser) parseFile() (*Document, error)
func (p *Parser) parseDict() ([]Pair, error)      // handles ~, quoted keys, annotations, '''values'''
func (p *Parser) parseList() ([]string, error)
func (p *Parser) parseTextBlock() (string, error) // NOTE: our own text block is NOT ohm's remainder grammar
```

Implementation notes:

1. Preserve `Pair{Key, Value, Disabled, Annotations, Line}` so the writer can round-trip.
2. Text blocks are the tricky part: `body { }` terminates at a line whose content is exactly `}` — do **not** trim leading spaces from body lines.
3. Implement a `Document` → `Item` mapping layer separate from the lexer, mirroring `bruToJson`'s semantic actions.
4. Keep a `Format` help type to produce `line:col: message` errors like ohm does.
5. Build a golden-file test suite from `packages/bruno-lang/v2/tests` and `packages/bruno-tests/collection`.
