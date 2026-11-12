# Request Editor

Status: VERIFIED against `bruno/packages/bruno-app/src/components/RequestPane/` at commit `60d7517` and cross-referenced with `ui-map/screens/request-*.md`.

---

## 1. Sub-Tab Architecture

`HttpRequestPane` hosts the primary sub-tab navigation (`RequestPane/index.js:25-36`):

```text
┌────────────────────────────────────────────────────────────────────────┐
│ [GET ▾] https://api.example.com/v1/users/:id         [Send ▾] [⋯]      │
├────────────────────────────────────────────────────────────────────────┤
│ Params (1) │ Body • │ Headers (2) │ Auth • │ Vars │ Script │ Assert (1) │ Tests │ Docs │ Settings │
├────────────────────────────────────────────────────────────────────────┤
│ (Active sub-tab pane)                                                  │
└────────────────────────────────────────────────────────────────────────┘
```

### 1.1 Tab Indicator Rules (VERIFIED)

| Sub-Tab      | Indicator Type    | Active Condition                                                 |
| ------------ | ----------------- | ---------------------------------------------------------------- |
| **Params**   | `sup` count badge | Enabled query params + path params count > 0                     |
| **Body**     | `StatusDot`       | Body mode ≠ `none`                                               |
| **Headers**  | `sup` count badge | Count of enabled headers > 0                                     |
| **Auth**     | `StatusDot`       | Effective auth mode ≠ `none` (or inherit resolving to non-empty) |
| **Vars**     | `sup` count badge | Count of enabled pre-request + post-response variables > 0       |
| **Script**   | `StatusDot`       | Non-empty script; styled with **error color** if script threw    |
| **Assert**   | `sup` count badge | Count of enabled assertions > 0                                  |
| **Tests**    | `StatusDot`       | Non-empty test script; **error color** on assertion/test failure |
| **Docs**     | `StatusDot`       | Documentation text is non-empty                                  |
| **Settings** | `StatusDot`       | Non-empty tags or non-default timeout/proxy settings             |

---

## 2. Method and URL Bar (`QueryUrl`)

**Container Height:** `32px` (`2rem`). Surface: `input.bg`.

| Control               | Dimension / Spec                                  | Behavior                                                                                                                                                                                                                                 |
| --------------------- | ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Method Selector**   | Default width `72px`, font `12px/600`             | Dropdown of standard HTTP verbs (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`, `TRACE`, `CONNECT`) and Custom Method entry. Verbatim method colors applied from theme tokens. Method caret omitted from URL bar by design. |
| **URL Input**         | Single-line CodeMirror editor                     | `{{variable}}` autocomplete on typing `{{`. Syntax highlights protocol, host, and path parameters (`:id`). Pasting `curl ...` auto-parses into method, headers, and body.                                                                |
| **Send Button**       | Height `30px`, padding `6px 14px`, primary accent | Dispatches request. Transitions to **Cancel** button with spinner while request is queued or in-flight. Hotkey: **⌘/Ctrl + Enter**.                                                                                                      |
| **Secondary Actions** | 30 px ghost button (`⋯`)                          | Save (⌘S), Generate Code snippet modal, Copy URL, Revert Changes.                                                                                                                                                                        |

---

## 3. Sub-Tab Specifications

### 3.1 Params Tab (`ui-map/screens/request-params.md`)

- **Query Params Table:** Key-value grid (`EditableTable`) with: Enabled checkbox, Name, Value, Description, Delete row action, and drag-reorder.
- **Path Params Table:** Automatically derived from `:param` tokens in the URL. Read-only names; values editable.
- **Two-way Sync:** Query params are the source of truth for query string reconstruction. Unchecking a param disables it without deleting its key/value.

### 3.2 Body Tab (`ui-map/screens/request-body.md`)

- **Mode Selector:** Radio / dropdown switching modes:
  - `none`: Empty body
  - `json`: Code editor with JSON syntax highlighting and linting
  - `text` / `xml` / `sparql`: Plain/formatted text editors
  - `form-urlencoded`: Key-value grid
  - `multipart-form`: Key-value grid supporting text fields and file attachments (`FileEditor` with file picker chips)
  - `file`: Raw binary file selection

### 3.3 Headers Tab (`ui-map/screens/request-headers.md`)

- Key-value table with name autocomplete (common HTTP headers).
- **Bulk Edit Mode:** Toggle to raw `Header-Name: Value` multiline text editor and back.

### 3.4 Auth Tab (`ui-map/screens/request-auth.md`)

- **Modes:** `Inherit`, `No Auth`, `AWS Sig v4`, `Basic Auth`, `Bearer Token`, `Digest Auth`, `OAuth 2.0`, `WSSE Auth`, `NTLM`, `Akamai EdgeGrid`.
- **OAuth 2.0 Placement:** Configurable token placement (Header: `Authorization: Bearer <token>` vs URL Query: `?access_token=<token>`).
- **Secret Masking:** Follows Decision R-5 (one `*` per character for secrets and passwords).

### 3.5 Assertions Tab (`ui-map/screens/request-script-assert.md`)

- Expression table: Target expression (e.g. `res.status`, `res.body.id`), Operator dropdown, Expected value.
- **Decision R-4 (Operators):** Assertion operators retain raw identifiers: `equals`, `notEquals`, `contains`, `matches`, `startsWith`, `endsWith`, `length`, `empty`, `null`, `type`.

### 3.6 Docs Tab (`ui-map/screens/request-docs.md`)

- Freeform Markdown editor. Note: Request Docs has **no empty placeholder state**; it renders directly as a clean editor ready for input.
