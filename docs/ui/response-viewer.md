# Response Viewer

Status: VERIFIED unless marked.

Root: `packages/bruno-app/src/components/ResponsePane/`

---

## 1. Layout

```text
┌───────────────────────────────────────────────────────────────┐
│ [Response] [Headers n] [Timeline] [Tests •]     200 · 42 ms · 1.2 KB · ⋯ │
├───────────────────────────────────────────────────────────────┤
│  format bar: [JSON ▾]   view: [Editor | Preview]   [filter 🔍]  │
│                                                               │
│  body view                                                    │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

| Piece | Component | Notes |
|---|---|---|
| Tabs | `ResponsiveTabs` | Collapses on narrow widths |
| Status | `StatusCode` | Colored by class (2xx green, 3xx amber, 4xx/5xx red) |
| Time | `ResponseTime`, `ResponseStopWatch` (live while sending) | |
| Size | `ResponseSize` | From `response.size` or decoded buffer |
| Actions | `ResponsePaneActions` | Copy, download, save-to-file, bookmark, clear; collapses below 135 px |
| Body | `QueryResult` | |
| Headers | `ResponseHeaders` | Count badge |
| Timeline | `Timeline` | Network trace |
| Tests | `TestResults` | Tests + assertions + pre/post script tests |

Source: `ResponsePane/index.js` `allTabs` memo (lines 122–148).

---

## 2. Body viewer (`QueryResult`)

### Formats

| Group | Options |
|---|---|
| Structured (preview) | `json`, `html`, `xml`, `javascript` |
| Raw (byte) | `raw`, `hex`, `base64` |

`useInitialResponseFormat(dataBuffer, headers)` picks a starting format from `content-type`, and an initial **view tab** (`editor` | `preview`).

### Rendering modes

| Format | `previewMode` | Renderer |
|---|---|---|
| `json` | `preview-json` | Pretty JSON in CodeMirror; tree collapse in preview |
| `html` | `preview-web` | **Sanitized** HTML (DOMPurify) |
| `xml` | `preview-xml` | Formatted XML |
| `javascript` | `preview-web` | Sandboxed preview |
| `raw` | `preview-text` | Plain |
| `base64`/`hex` | content sniffing → `preview-image`, `preview-pdf`, or text | |

### Large-response protection (three thresholds — VERIFIED)

| Threshold | Location | Behavior |
|---|---|---|
| **10 MB** | `QueryResult/index.js` line 125 (`isLargeResponse`) | Body not formatted; `LargeResponseWarning` shown until user opts in |
| **50 MB** | `utils/common/index.js` `LARGE_BUFFER_THRESHOLD` | `formatResponse` skips decoding the buffer to a string; JSON path renders from parsed `data` only |
| **2.5 MB** | Main-process watcher (collections) | Unrelated to responses; included to avoid confusion |

Also `promisifyStream` in main buffers the whole body for non-stream responses, so a 500 MB download is fully in memory **on both sides** today. See `research/limitations.md`.

### JSON filter

`QueryResultFilter` applies a JSONPath-like expression (`applyJSONPathFilter`) to the parsed body before display, with an expandable input at the bottom.

---

## 3. Headers tab

Table of `name: value`, raw toggle, copy-all. Includes hop-by-hop headers as the server sent them (e.g. `content-encoding: gzip` remains, because axios decompresses transparently).

---

## 4. Timeline tab

Renders `response.timeline` entries: separators, request line, sent headers, response line, response headers, timings, errors, redirect notes, and any nested `scripted-request` entries from `bru.runRequest`/`bru.sendRequest`.

`utils/timeline/index.js` → `sentHeadersFromTimeline` reconstructs the outgoing header set from the trace.

---

## 5. Tests tab

`TestResultsLabel` aggregates:

```text
item.testResults              (tests block)
item.assertionResults         (assert block)
item.preRequestTestResults    (test() in pre-request script)
item.postResponseTestResults  (test() in post-response script)
```

Each row shows description, pass/fail, expected vs actual, and error text. A failed assertion shows `lhsExpr`, operator, `rhsExpr`, and values.

---

## 6. States

| State | Rendering |
|---|---|
| No request yet | `Placeholder` |
| Queued / sending | `Overlay` with spinner; stopwatch running |
| Cancelled | `response === null` → placeholder with "request cancelled" |
| Transport error | `NetworkError` (message + retry) |
| Skipped (runner, tags) | `SkippedRequest` |
| Stream/WS | `WsResponsePane/WSMessagesList` when `response.stream` |
| gRPC | `GrpcResponsePane` |
| Too large | `LargeResponseWarning` |

---

## 7. Actions

| Action | Notes |
|---|---|
| Copy body | Respects current format |
| Download / Save to file | Streams from main (`renderer:save-response-to-file`) |
| Bookmark → response example | Creates an `example` block |
| Clear | Removes response from Redux |
| Set as variable | Select text → `SetAsVariableMenu` → creates a variable in a chosen scope |

---

## 8. Go port guidance (PROPOSED)

1. Keep the three-tier large-body handling but move to **streaming preview**: render incrementally, spool to disk above N MB.
2. Implement JSON path filtering and raw/hex/base64 toggles — cheap and expected.
3. Sanitize HTML previews in a webview with JS disabled by default.
4. Preserve the format-per-tab memory behavior.
