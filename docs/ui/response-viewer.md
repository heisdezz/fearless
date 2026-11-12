# Response Viewer

Status: VERIFIED against `bruno/packages/bruno-app/src/components/ResponsePane/` at commit `60d7517` and cross-referenced with `ui-map/screens/response-pane.md` and `ui-map/screens/response-viewer.md`.

---

## 1. Response Pane Structure

```text
┌───────────────────────────────────────────────────────────────────────────┐
│ [Response] [Headers (6)] [Timeline] [Tests (4/4)]  200 OK · 42ms · 1.2 KB  [⋯]│
├───────────────────────────────────────────────────────────────────────────┤
│ Format: [JSON ▾]  View: [Editor | Preview]   [Filter 🔍]                   │
├───────────────────────────────────────────────────────────────────────────┤
│ (Formatted Response Body Viewer / CodeMirror read-only)                   │
│                                                                           │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

### 1.1 Status Strip Metrics

| Element                 | Specification & Sizing           | Theme Token / Behavior                                                                                                                                                |
| ----------------------- | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Response Status**     | `13px/600`, monospace figures    | 2xx: `requestTabPanel.responseOk` (`#2e8a54` L / `#73e89a` D)<br/>3xx: `status.warning.text`<br/>4xx/5xx: `requestTabPanel.responseError` (`#ce4f3b` L / `#da462f` D) |
| **Response Time**       | `12px/400`, monospace figures    | Live stopwatch (`ResponseStopWatch`) ticks during active execution; final elapsed time on completion.                                                                 |
| **Response Size**       | `12px/400`, monospace figures    | Formatted bytes/KB/MB calculated from response buffer length.                                                                                                         |
| **Action Icons**        | 16 px icons, 24 px click targets | Copy, Download file, Save to file, Bookmark, Clear response.                                                                                                          |
| **Responsive Collapse** | Collapses below `135px`          | When pane width drops below `135px` (`RIGHT_CONTENT_EXPANDED_WIDTH = 135`), action buttons collapse into a `⋯` popover menu.                                          |

---

## 2. Body Viewer (`QueryResult`)

### 2.1 Supported Formats and Renderers

| Format       | View Options       | Renderer                                                                                |
| ------------ | ------------------ | --------------------------------------------------------------------------------------- |
| `json`       | Editor / Preview   | Read-only CodeMirror with JSON syntax highlight; collapsible tree structure in preview. |
| `xml`        | Editor / Preview   | XML syntax highlighting with auto-formatting.                                           |
| `html`       | Editor / Preview   | DOMPurify-sanitized web preview in sandbox.                                             |
| `javascript` | Editor / Preview   | Highlighted JS code.                                                                    |
| `raw`        | Text view          | Unformatted plaintext.                                                                  |
| `hex`        | Hex dump           | Byte-level hex viewer with ASCII sidebar.                                               |
| `base64`     | Image / PDF / Text | Content sniffing: renders images directly (`preview-image`) or opens PDF preview.       |

### 2.2 Safety Thresholds for Large Payloads (VERIFIED)

| Threshold | Trigger Location                                   | Protective Action                                                                                                              |
| --------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **10 MB** | `QueryResult/index.js:125` (`isLargeResponse`)     | Disables automatic pretty-printing to prevent main thread lockup; displays `LargeResponseWarning` with manual override button. |
| **50 MB** | `utils/common/index.js` (`LARGE_BUFFER_THRESHOLD`) | Skips decoding raw binary buffer to string; renders summary and requires user to save to disk.                                 |

### 2.3 Interactive JSON Filter

- Filter toggle opens an inline query bar at the bottom of the body.
- Evaluates JSONPath expressions in real time (`applyJSONPathFilter`), filtering the rendered tree/editor to matching nodes.

---

## 3. Headers Tab

- Table displaying all response headers received from the server.
- Shows raw header names and values, preserving hop-by-hop headers and compression status (`content-encoding: gzip`).
- "Copy All Headers" action for quick export.

---

## 4. Timeline Tab (Network Diagnostics)

- Structured event log displaying the complete HTTP/TLS transaction:
  - Connection initiation & DNS resolution timing
  - TLS handshake details (cipher, certificate subject, protocol version)
  - Full HTTP request line and headers sent
  - Response status line and response headers received
  - Redirect history (each hop recorded with target URI and intermediate status)
  - Script-driven requests executed via `bru.sendRequest()` nested within trace

---

## 5. Tests and Assertions Tab

Aggregates test outcomes across all evaluation phases:

- Declarative `assert {}` blocks
- Procedural `tests {}` block scripts
- Pre-request script tests (`test()` calls)
- Post-response script tests (`test()` calls)

Displays total counts (e.g. `4/4 passed`), per-assertion status dots (green check / red cross), expression failure diffs, and thrown script errors.
