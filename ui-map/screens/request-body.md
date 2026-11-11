# Screen — Body tab and body modes

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source.

---

## 1. Layout

```text
Tab strip ..........  Body
                      └ rightContent: [ No Body ▾ ] [ Prettify ]     ← RequestPane/RequestBody/RequestBodyMode/index.js:111-127
────────────────────────────────────────────────────────────────
Body area (tab content, HeightBoundContainer)
 ├─ No Body            → literal text "No Body"                     RequestBody/index.js:101
 ├─ Multipart Form     → Key | Value | Content-Type | Description   request-params.md §2.1
 ├─ Form URL Encoded   → Key | Value | Description                  request-params.md §2.1
 ├─ File / Binary      → File | Content-Type | Selected | Description | 🗑   + Add File
 ├─ JSON / XML / TEXT / SPARQL → full-height editor (CodeMirror)     RequestBody/index.js:39-79
```

GraphQL requests do not use this tab; they use the `Query` tab
(`RequestPane/GraphQLRequestPane/index.js:219-274`) with an optional
`Query Builder` side panel (default width `320px`, drag range `200–600px`
and ≤50% of the pane — `:67,107-110`) and a collapsible `Variables` panel
(default height `150px`, min `80px`, ≤60% — `:68,116-123`).

---

## 2. Mode selector — listed order and labels

`RequestBodyMode` (`RequestPane/RequestBody/RequestBodyMode/index.js:23-46`)
renders a `MenuDropdown` with three groups (`groupStyle="select"`,
`showGroupDividers={false}`, placement `bottom-end`, `:114-121`):

| Group | id | Label (verbatim) | Icon (Tabler) |
| --- | --- | --- | --- |
| Form | `multipartForm` | `Multipart Form` | `IconForms` |
| Form | `formUrlEncoded` | `Form URL Encoded` | `IconForms` |
| Raw | `json` | `JSON` | `IconBraces` |
| Raw | `xml` | `XML` | `IconCode` |
| Raw | `text` | `TEXT` | `IconFileText` |
| Raw | `sparql` | `SPARQL` | `IconDatabase` |
| Other | `file` | `File / Binary` | `IconFile` |
| Other | `none` | `No Body` | `IconX` |

Trigger label is `humanizeRequestBodyMode(bodyMode)`
(`utils/collections/index.js:980-1013`) — identical strings to the table above,
with `No Body` as the fallback for an unknown/empty mode.

Trigger chrome (`RequestBodyMode/StyledWrapper.js`):

| Property | Value | Source |
| --- | --- | --- |
| Text colour | `primary.text` (`#c07a26` L / `#e7b85f` D) | `:12` |
| Padding | `pl-3` (12px), `py-1` (4px) | `index.js:122` |
| Radius | `3px` | `:8` |
| Caret | `IconCaretDown` 14px, `strokeWidth 2`, `rgb(140,140,140)` filled, `ml-1` | `index.js:123`, `StyledWrapper.js:16-19` |
| Background | transparent | `:7` |

The same component is exported standalone as
`components/BodyModeSelector/index.js` (defaults shown at `:5-22`); inside the
request pane the in-place `RequestBodyMode` is used.

`Prettify` button: rendered only for `json` and `xml`
(`RequestBodyMode/index.js:125-127`), `ml-2`. Failures raise toasts
`Unable to prettify. Invalid JSON format.` /
`Unable to prettify. Invalid XML format.` (`:77,90`). Works for the whole tab
body when focus is inside it (`:96-100`) and is also bound to `Alt+Shift+F`
(`keyMappings.js:44`).

---

## 3. Per-mode editor

| Mode | Editor | Mode string passed to CodeMirror | Source |
| --- | --- | --- | --- |
| `json` | `CodeEditor`, variable highlighting on | `application/ld+json` | `RequestBody/index.js:40,64-78` |
| `text` | `CodeEditor` | `application/text` | `:41` |
| `xml` | `CodeEditor` | `application/xml` | `:42` |
| `sparql` | `CodeEditor` | `application/sparql-query` | `:43` |
| `file` | `FileBody` table | — | `:82-84` |
| `formUrlEncoded` | `FormUrlEncodedParams` table | — | `:86-88` |
| `multipartForm` | `MultipartFormParams` table | — | `:90-92` |
| `none` | plain `div`, text `No Body` | — | `:94-99` |
| `graphql` | not handled by `RequestBody`; `GraphQLRequestPane` provides Query editor + Variables | — | `RequestBody/index.js:39-99` (no graphql branch) |

Editor CSS class: `REQUEST_BODY_EDITOR_CLASS` is attached to the wrapper for
`json/xml/text/sparql` (`RequestBody/index.js:57`) and to the GraphQL query
container (`GraphQLRequestPane/index.js:222`); the prettify binding uses
`e.target.closest('.${REQUEST_BODY_EDITOR_CLASS}')` to scope itself.

Font family / size come from preferences (`font.codeFont`, `font.codeFontSize`,
`RequestBody/index.js:68-69`). Scroll position persists per mode per request
(`request-body-${bodyMode}-scroll-${uid}`, `:21`).

### 3.1 File / Binary table (`FileBody`)

Columns `File`, `Content-Type`, `Selected`, `Description`, (trash) with widths
`25% / 20% / 10% / 35% / 70px` (`FileBody/StyledWrapper.js:14-32`).

| Element | Detail | Source |
| --- | --- | --- |
| File cell | `FilePickerEditor` single-file, display `labelAndIcon` | `FileBody/index.js:94-108` |
| Content-Type | `SingleLineEditor`, placeholder `Auto`, cleared when the file changes | `:109-134`, `:40` |
| Selected | native radio (`name="selected"`), `tabIndex="-1"` | `:135-147` |
| Description | `MultiLineEditor`, placeholder `Description` | `:148-167` |
| Delete | `IconTrash` 20px | `:168-175` |
| Add row | text link `+ Add File` | `:180-185` |
| Empty state | header row only, no body rows | `:88-177` |

---

## 4. States

| State | Detail | Source |
| --- | --- | --- |
| Body tab indicator | `StatusDot` next to `Body` when `mode !== 'none'` | `RequestPane/HttpRequestPane/index.js:109` |
| Default tab | if no enabled params and mode ≠ none, the Body tab opens by default | `utils/collections/index.js:1226-1228` |
| Switching mode | does not clear other modes' content (each mode has its own field) | `RequestBody/index.js:65-69` |

## 5. Interactions & shortcuts

| Interaction | Result | Source |
| --- | --- | --- |
| Open mode menu | `MenuDropdown`, active mode highlighted via `selectedItemId` | `RequestBodyMode/index.js:118` |
| Select mode | `updateRequestBodyMode({ mode })` | `:56-64` |
| `Prettify` | pretty-prints JSON (2-space) or XML (`collapseContent: true`) | `:67-93` |
| `Alt + Shift + F` | same as Prettify, scoped to the body editor | `keyMappings.js:44`, `RequestBodyMode/index.js:96-100` |
| `Cmd/Ctrl + Enter` | send request | `keyMappings.js:42` |
| `Cmd/Ctrl + S` | save | `keyMappings.js:9` |
| `{{` in value cells | variable autocomplete popup | `FormUrlEncodedParams/index.js:72`, `MultipartFormParams/index.js:204` |

## 6. Verbatim copy index

`Multipart Form`, `Form URL Encoded`, `JSON`, `XML`, `TEXT`, `SPARQL`,
`File / Binary`, `No Body`, `Prettify`, `Auto`, `+ Add File`,
`Unable to prettify. Invalid JSON format.`,
`Unable to prettify. Invalid XML format.` — sources cited above.

Screenshots: `refs/api_screen_bruno.png` shows the tab strip with `Body` but
the pane is on `Params`; no dedicated body-mode screenshot exists in `refs/`.

---

## 7. Open Questions

1. **GraphQL is not a body mode in the HTTP pane.** It is a separate request
   type (`graphql-request`) with its own pane. If the Go port exposes a
   `graphql` body mode the UI diverges from Bruno.
2. **`RequestBody` `mode` values are camelCase** (`multipartForm`,
   `formUrlEncoded`) while the displayed labels are sentence case; persisted
   file formats use the same camelCase keys, so a Go port must not rename them.
3. **Prettify for `sparql`/`text`** is not offered (button only for json/xml),
   even though the binding exists globally. Confirm whether a Go port should
   keep this restriction.
4. **Vendor-mode mapping**: `application/ld+json` / `application/text` are used
   as CodeMirror mode identifiers (`RequestBody/index.js:46-51`). The
   corresponding Go editor (MyGo `Editor`, which takes a code language name
   rather than a MIME type) needs an explicit mapping table — not specified
   here.
5. **Empty-state "No Body"** has no visual container or helper text; only the
   literal string. Confirm before adding any illustration.
