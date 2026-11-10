# Response Parser

Status: VERIFIED unless marked.

---

## 1. Where parsing happens

| Layer | Function | Output |
|---|---|---|
| Main process | `parseDataFromResponse(response, disableParsingResponseJson)` | `{ data, dataBuffer }` |
| Main process | `parseDataFromRequest(request)` | `{ data, dataBuffer }` (for the "sent" view) |
| IPC boundary | `sendNetworkRequest` | `dataBuffer` → base64 string |
| Renderer | `formatResponse(data, dataBufferBase64, mode, filter, threshold)` | display string |
| Renderer | `useInitialResponseFormat` | initial format + view tab |

Sources: `packages/bruno-electron/src/utils/common.js` (lines 106–151), `packages/bruno-app/src/utils/common/index.js` (line 249).

---

## 2. `parseDataFromResponse` algorithm

```js
const charsetMatch = /charset=([^()<>@,;:"/[\]?.=\s]*)/i.exec(headers['content-type'] || '');
const charsetValue = charsetMatch?.[1];
const dataBuffer = Buffer.from(response.data);

let data = iconv.encodingExists(charsetValue)
  ? iconv.decode(dataBuffer, charsetValue)
  : iconv.decode(dataBuffer, 'utf-8');

try {
  data = data.replace(/^\uFEFF/, '');         // strip BOM (ZWNBSP)
  if (!disableParsingResponseJson) data = JSON.parse(data);
} catch { /* keep string */ }

return { data, dataBuffer };
```

Notes:

1. **Charset-aware decoding** via `iconv-lite`, falling back to UTF-8. This matters for `charset=iso-8859-1`, `shift_jis`, etc.
2. **BOM stripping** before JSON parse.
3. JSON parsing is **opportunistic** — any text that parses becomes an object, regardless of Content-Type. A `text/plain` body containing `{"a":1}` becomes an object.
4. `disableParsingResponseJson` (set by `req.disableParsingResponseJson()` in scripts) prevents step 3.
5. The raw bytes are always retained in `dataBuffer`.

**Go note:** use `golang.org/x/text/encoding` + `charset.NewReaderLabel` for charset detection; fall back to raw bytes for display.

---

## 3. Response headers

- Kept as received (`response.headers`), including `content-encoding` even though axios decompressed.
- Duplicate headers are joined by axios (Node) into comma-separated strings — a known lossy point for `set-cookie` (handled separately by the cookie jar before this point).
- `sentHeaders` is attached to `error.response` so post-response scripts can inspect what was actually sent.

---

## 4. Request body parsing (for the "request sent" view)

`parseDataFromRequest`:

| Case | Rendered |
|---|---|
| `mode === 'file'` | `'<request body redacted>'` |
| multipart `FormData` with `_originalMultipartData` | Reconstructed textual multipart via `formatMultipartData(...)` with the real boundary |
| string | as-is |
| object | `safeStringifyJSON` |
| empty | `{ data: null, dataBuffer: null }` |

`formatMultipartData` produces a readable (not wire-exact) representation including `value: [File: name]` placeholders for file parts.

---

## 5. Size and timing

| Value | Computation |
|---|---|
| `size` | `Buffer.byteLength(response.dataBuffer)` — post-decompression |
| `duration` | `measureResponseTime(config.metadata)` — from `startHop` to last byte (or stream close) |
| `url` | From the final `response.request` (`protocol + '//' + host + path`) |

---

## 6. Renderer formatting (`formatResponse`)

| Format | Behavior |
|---|---|
| `json` | JSONPath filter → pretty print (`fast-json-format`) from raw text when possible; falls back to `safeStringifyJSON(data)`; above 50 MB only parsed `data` is stringified |
| `xml` | Pretty print |
| `html` | Raw text (preview mode sanitizes) |
| `raw` | Decoded string |
| `hex` | Hex dump |
| `base64` | Base64 of the buffer |

Buffer-level thresholds: 10 MB (skip formatting + show warning) and 50 MB (never decode the whole buffer to a string).

---

## 7. Streaming responses (SSE)

```js
isResponseStream = hasStreamHeaders(response.headers);
if (isResponseStream) axiosDataStream = response.data;
// on 'close':
response.data = buildResponseBodyFromStreamChunks(sseChunks, response.headers, disableJson)
response.dataBuffer = …
runPostScripts()
```

- Post scripts run only after the stream closes.
- `sseChunks` are also returned for the UI.

---

## 8. Known limitations

- Whole body is buffered for non-stream responses (no size cap).
- `dataBuffer` is base64-transported, adding ~33 % memory overhead.
- JSON auto-parse can surprise users when a text body happens to be valid JSON.
- Header duplication is flattened.

### Go implementation (PROPOSED)

```go
type Body struct {
    Bytes    []byte      // nil if spilled
    TempFile string      // spill target above threshold
    Size     int64
}

type ParsedResponse struct {
    Status   int
    Proto    string
    Headers  map[string][]string   // preserve multiplicity
    Body     Body
    Text     string                // charset-decoded, BOM-stripped
    JSON     any                   // parsed opportunistically
    Timing   Timing
    URL      *url.URL
    Redirects []RedirectHop
}
```

Keep the "opportunistic JSON" behavior behind a setting for parity.
