# Response viewer — `QueryResponse` / `QueryResult`

Status: VERIFIED against `bruno/packages/bruno-app/src/` at the commit in
`overview.md:8`. Screenshots: `refs/api_screen_bruno.png` (empty pane),
`refs/main_app_bruno.png`. This file covers the four pane tabs' _content_:
body viewer (pretty/preview/raw), format selector, JSONPath/XPath filter,
Headers table, Timeline, Tests and the example response pane. The tab bar,
status seam, actions and pane-level states are in `screens/response-pane.md`.

**Viewer hosts.** The same viewer component is mounted in three places, so a
Go port needs one implementation:

| Host                | Where                                          | Selector shown?                               | File:line                                              |
| ------------------- | ---------------------------------------------- | --------------------------------------------- | ------------------------------------------------------ |
| Main response pane  | ResponsePane → `QueryResult`                   | in the tab bar's right content                | `components/ResponsePane/index.js:249-264`, `:162-176` |
| Timeline entry body | `TimelineItem/Common/Body` → `QueryResponse`   | only for responses, hidden for request bodies | `components/ResponsePane/Timeline/TimelineItem/Common/Body/index.js:26-36`     |
| Runner result       | `RunnerResults/ResponsePane` → `QueryResponse` | yes                                           | `RunnerResults/ResponsePane/index.js:44-55`            |

`QueryResponse` is a thin wrapper: it owns local `selectedFormat` /
`selectedTab` / `filter` state (initially `'raw'` / `'editor'` / `''`)
and renders the type selector in a right-aligned toolbar plus
`QueryResult` (`components/ResponsePane/QueryResponse/index.js:21-31`, `:33-70`). The wrapper is
`border-radius:4px; border:1px solid table.border`; `.result-type-selector`
has `border-bottom:1px solid table.border` (`components/ResponsePane/QueryResponse/StyledWrapper.js:3-13`).

---

## 1. Layout tree

```text
div.response-tab-content          (scroll container, /ResponsePane/StyledWrapper.js:62)
└─ QueryResult  (flex, w-full h-full)
   ├─ [error branch]
   │  └─ div.error            white-space: pre-line, colors.text.danger
   │     └─ div.mt-6.muted.text-xs   (only for self-signed cert errors)
   ├─ [large branch]  LargeResponseWarning          → screens/response-pane.md §4.7
   └─ [normal branch] div.h-full.flex.flex-col
      └─ div.flex-1.relative
         ├─ div.absolute.inset-0        data-testid=response-preview-container
         │  └─ QueryResultPreview  (editor | preview)
         │     ├─ editor  → CodeEditor (read-only)
         │     └─ preview → HtmlPreview | <img> | PDF | <audio> | VideoPreview
         │                 | JsonPreview | TextPreview | XmlPreview | empty-state
         └─ QueryResultFilter  (only JSON editor; absolute bottom-0, full width)
            ├─ input#response-filter       placeholder "$.store.books..author" | "/store/books//author"
            └─ IconFilter 20/1.5  →  IconX 20/1.5
```

`QueryResult/StyledWrapper.js` notes: the CodeMirror host is forced to
`position:absolute; inset:0; height:100%; width:100%` (`:12-18`) so the
editor fills the pane; PDF pages get `margin-top:10px` and a
`1px solid darkgrey` text layer with `5px 5px 5px 1px #ccc` shadow
(`:20-32`); `.muted` = `colors.text.muted`, `.error` = `colors.text.danger`
(`:39-45`).

---

## 2. Format model

`selectedFormat` is one of the ids below (`components/ResponsePane/QueryResult/index.js:11-29`).
`selectedTab` is `'editor'` or `'preview'` (`:101`).

| id           | Label        | Tabler icon           | CodeMirror mode       | Options group            |
| ------------ | ------------ | --------------------- | --------------------- | ------------------------ |
| `json`       | `JSON`       | `IconBraces`          | `application/ld+json` | structured               |
| `html`       | `HTML`       | `IconCode`            | `xml`                 | structured               |
| `xml`        | `XML`        | `IconFileCode`        | `xml`                 | structured               |
| `javascript` | `JavaScript` | `IconBrandJavascript` | `javascript`          | structured               |
| —            | _(divider)_  | —                     | —                     | `divider-structured-raw` |
| `raw`        | `Raw`        | `IconFileText`        | `text/plain`          | raw                      |
| `hex`        | `Hex`        | `IconHexagons`        | `text/plain`          | raw                      |
| `base64`     | `Base64`     | `IconBinaryTree`      | `text/plain`          | raw                      |

Sources: option tables `components/ResponsePane/QueryResult/index.js:12-29`; icon map
`QueryResultTypeSelector/index.jsx:9-17`.

- Binary content types (`image`, `video`, `audio`, `pdf`, `zip`, but **not**
  `svg`) get **only** the raw options (`components/ResponsePane/QueryResult/index.js:61-90`).
  Content type is resolved from the base64 sniff first, then the
  `content-type` header (`:73-78`).
- Default format/tab come from `getDefaultResponseFormat(contentType)`;
  `null` until both the sniff and the header are known
  (`components/ResponsePane/QueryResult/index.js:45-58`; `utils/response`). Until then the pane
  falls back to `raw` / `editor` (`components/ResponsePane/index.js:58-59`).
- The selection is persisted per tab in Redux and reset when `content-type`
  changes (`components/ResponsePane/index.js:61-77`).

### 2.1 Format → preview mode

`components/ResponsePane/QueryResult/index.js:147-167`:

| `selectedFormat`                 | `previewMode`   |
| -------------------------------- | --------------- |
| `html`, `javascript`             | `preview-web`   |
| `json`                           | `preview-json`  |
| `xml`                            | `preview-xml`   |
| `raw`                            | `preview-text`  |
| `base64`/`hex` + sniffed `image` | `preview-image` |
| `base64`/`hex` + sniffed `pdf`   | `preview-pdf`   |
| `base64`/`hex` + sniffed `audio` | `preview-audio` |
| `base64`/`hex` + sniffed `video` | `preview-video` |
| anything else                    | `preview-text`  |

Note the asymmetry: `JSON`/`HTML`/`XML`/`JavaScript` also decide the
**editor** syntax highlighting via `codeMirrorMode` (`:169-174`).

### 2.2 `QueryResultTypeSelector`

```text
[ 🔍Eye | {IconBrace} ] JSON ⌄        ← button (data-testid=format-response-tab)
      └ dropdown (placement bottom-end)
         ├─ header: "Preview"  [ToggleSwitch 2xs]   title "Turn on Preview Mode" / "Turn off Preview Mode"
         ├─ JSON      (brace)
         ├─ HTML      (code)
         ├─ XML       (file-code)
         ├─ JavaScript
         ├─ ──────────
         ├─ Raw
         ├─ Hex
         └─ Base64
```

| Property                  | Value                                                                                           | Source                                                                      |
| ------------------------- | ----------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Button base               | `h-7 rounded-[6px] border px-2 text-xs`, overridden `h-[22px] text-[10px]`                      | `QueryResultTypeSelector/index.jsx:19-39`, `:37`                            |
| Prefix icon               | `IconEye` 14/2 in preview mode, format icon 14/1.5 in editor mode                               | `:77-86`                                                                    |
| Prefix colour             | active tab → `primary.text`; inactive → `colors.text.muted`                                     | `:34`, `components/ResponsePane/StyledWrapper.js:21-27`                                             |
| Caret                     | `IconCaretDown` 12/2, `ml-0.5`, shown only when `isActiveTab`                                   | `:37`                                                                       |
| Caret colour / fill       | `app.collection.toolbar.environmentSelector.caret` (`#b0b0b0` / `#aaa`)                         | `components/ResponsePane/StyledWrapper.js:3-7`                                                      |
| Border                    | `workspace.border` (`#e5e5e5` / `#444`); hover `environmentSelector.hoverBorder`                | `components/ResponsePane/StyledWrapper.js:9-13`, `components/ResponsePane/StyledWrapper.js:101-118` |
| Active-tab border         | `environmentSelector.border` (`#e5e5e5` / `#444`)                                               | `components/ResponsePane/StyledWrapper.js:112-118`                          |
| Divider                   | `dropdown.separator`, `height:1px`, `margin:4px 0`                                              | `components/ResponsePane/StyledWrapper.js:15-19`                                                    |
| Menu header               | `flex; justify-between; gap .75rem; py .35rem; px .6rem`; label `Preview` at `0.8125rem`, muted | `index.jsx:102-104`, `components/ResponsePane/StyledWrapper.js:29-31`                               |
| Selection tick            | `showTickMark` with `selectedItemId={formatValue}`                                              | `index.jsx:124-127`                                                         |
| First click when inactive | selects the editor tab instead of opening the menu                                              | `index.jsx:54-62`                                                           |

The selector's `title` on the toggle is exactly
`Turn on Preview Mode` / `Turn off Preview Mode` (`index.jsx:112-114`).

---

## 3. Body viewer modes

### 3.1 Editor (pretty) — `selectedTab === 'editor'`

Read-only `CodeEditor` (`components/ResponsePane/QueryResult/QueryResultPreview/index.js:60-79`)
with `mode = codeMirrorMode`, `value = formatResponse(data, dataBuffer,
selectedFormat, filter)`, the user's `font.codeFont` / `font.codeFontSize`
and the active display theme. Editor chrome and syntax colours are in
`design-system.md` §7 and `widgets.md`; this screen only needs to know the
data handed to it.

`formatResponse` behaviour (`utils/common/index.js:250+`, prefix at `:250`):

- `data` missing or no buffer/mode → returns `''` (`utils/common/index.js:250-252`).
- Buffers above `LARGE_BUFFER_THRESHOLD = 50 MB` are not decoded
  (`:238`, `:254-264`).
- A `filter` is applied as JSONPath for JSON modes; invalid paths log a
  console warning and fall through to unfiltered output (`:241-247`, `:266-272`).
- XML modes re-serialise pretty-printed unless the buffer is very large
  (`:289-299`); the `filter` argument is only honoured for JSON modes.

Scroll position is persisted per request: `response-body-scroll-{uid}`
(`QueryResultPreview/index.js:37`).

### 3.2 Preview — `selectedTab === 'preview'`

| `previewMode`   | Renderer                                                                                                                                  | Notes                                                                                                                                                                                                                   | Source                                                                     |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `preview-web`   | `HtmlPreview`                                                                                                                             | Electron `<webview>` with injected `<base href="{requestUrl}">`; `disableDialogs=true, javascript=yes`; drag-safe (`pointer-events:none`) while the pane resize is active                                               | `QueryResultPreview/index.js:82-85`, `HtmlPreview.js:33-54`                |
| `preview-image` | `<img src="data:{mime};base64,…">`                                                                                                        | no sizing constraints in source                                                                                                                                                                                         | `QueryResultPreview/index.js:86-88`                                        |
| `preview-pdf`   | `react-pdf`: one `<Page>` per page, `renderAnnotationLayer={false}`; wrapper `height:100%; overflow:auto; max-height:calc(100vh - 220px)` |                                                                                                                                                                                                                         | `:89-99`                                                                   |
| `preview-audio` | `<audio controls>` `mx-auto`                                                                                                              |                                                                                                                                                                                                                         | `:100-104`                                                                 |
| `preview-video` | `VideoPreview` (react-player)                                                                                                             | `controls`, `muted`; shows `Loading video...` until the blob URL exists                                                                                                                                                 | `:105-107`, `VideoPreview.js:17`                                           |
| `preview-json`  | `JsonPreview`                                                                                                                             | react-json-view: `collapsed={1}`, `displayDataTypes=false`, `displayObjectSize=true`, `enableClipboard=true`, `name=false`, theme `rjv-default` (light) / `monokai` (dark), `font-size:12px`, padding 16 px, mono stack | `:108-110`, `JsonPreview.js:53-70`                                         |
| `preview-text`  | `TextPreview`                                                                                                                             | mono `13px`, `p-4`, `whitespace-pre-wrap`, `break-words`, links underlined/clickable; rows revealed in chunks of **200** via `IntersectionObserver` (`rootMargin 200px`)                                                | `:112-114`, `TextPreview.js:53-73`, `hooks/useChunkedReveal/index.js:3-38` |
| `preview-xml`   | `XmlPreview`                                                                                                                              |                                                                                                                                                                                                                         | `:116-118`                                                                 |
| default         | inline empty state                                                                                                                        | see §3.3                                                                                                                                                                                                                | `:120-130`                                                                 |

JSON preview error banners (`JsonPreview.js:31-43`), rendered with
`ui/ErrorBanner`:

| Condition                      | Title                    | Message                                                                    |
| ------------------------------ | ------------------------ | -------------------------------------------------------------------------- |
| string that fails `JSON.parse` | `Cannot preview as JSON` | `Invalid JSON format: {e.message}`                                         |
| null/undefined data            | `Cannot preview as JSON` | `Data is null or undefined. Expected a valid JSON object or array.`        |
| non-object after parsing       | `Cannot preview as JSON` | `Data cannot be rendered as a JSON tree. Expected a JSON object or array.` |
| non-string, non-object input   | `Cannot preview as JSON` | `Invalid input. Expected a JSON object, array, or valid JSON string.`      |

Clicking an `http(s)` value in the JSON tree or a link in text/XML opens
that URL as a new request of the same type (`QueryResultPreview/index.js:54-58`;
`JsonPreview.js:45-51`; `TextPreview.js:25-49`).

### 3.3 "No Preview Available" empty state

Verbatim (`QueryResultPreview/index.js:121-130`):

| Line                                                    | Style                                                                    |
| ------------------------------------------------------- | ------------------------------------------------------------------------ |
| `No Preview Available`                                  | `text-lg font-semibold` (18 px, 600), `text-gray-700 dark:text-gray-200` |
| `Sorry, no preview is available for this content type.` | `text-sm` (14 px), `text-gray-500 dark:text-gray-400`                    |

Centred (`flex flex-col items-center justify-center h-full text-center`),
`p-4`, second line `mb-2`-separated.

### 3.4 Raw

`Raw`, `Hex` and `Base64` all use `preview-text` in preview mode and
`text/plain` highlighting in editor mode (`components/ResponsePane/QueryResult/index.js:12-16`,
`:152`, `:169-174`). `Hex`/`Base64` in preview mode upgrade to an image,
PDF, audio or video player when the sniffed type allows (§2.1).

---

## 4. Filter — `QueryResultFilter`

Shown only when `codeMirrorMode.includes('json') && selectedFormat === 'json'
&& selectedTab === 'editor'` (`components/ResponsePane/QueryResult/index.js:176`, `:222-230`).

```text
div.response-filter (absolute bottom-2 right-0 w-full, flex row items-center gap-2 py-4 px-2, pointer-events-none)
├─ [Collapsed]  IconFilter 20/1.5   (text-gray-500, id=request-filter-icon)
└─ [Expanded]   input#response-filter  "…path…"     IconX 20/1.5
```

| Property         | Value                                                                                            | Source                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| Placement        | `absolute; bottom:0; width:100%` (`StyledWrapper.js`) / `absolute bottom-2 right-0 … py-4 px-2`  | `components/ResponsePane/QueryResult/StyledWrapper.js:47-61`, `QueryResultFilter/index.js:55-57` |
| Input box        | `block ml-14 p-2 py-1 transition-all duration-200 ease-in-out border border-gray-300 rounded-md` | `QueryResultFilter/index.js:70-72`                                       |
| Collapsed        | `w-[0] opacity-0`, `pointer-events-none`                                                         | `:70-72`                                                                 |
| Expanded         | `w-full opacity-100 pointer-events-auto`                                                         | `:70-72`                                                                 |
| JSON placeholder | `$.store.books..author`                                                                          | `:42-45`                                                                 |
| XML placeholder  | `/store/books//author` (used by other viewers of the same component)                             | `:47-49`                                                                 |
| Tooltip          | `Filter with JSONPath` / `Filter with XPath` anchored on `request-filter-icon`                   | `:30-40`, `:58`                                                          |
| Close/reset      | clicking `IconX` clears the filter and resets the input                                          | `:9-22`                                                                  |

Filter text is persisted per tab: `tabs[uid].responseFilter`, plus
`.responseFilterExpanded` (`components/ResponsePane/index.js:172-175`).

---

## 5. Headers tab — `ResponseHeaders`

```text
div.table-wrapper (1px solid table.border, radius 4px, font.size.sm)
└─ table (width 100%, collapse)
   ├─ thead: "Name" | "Value"       (table.thead.color, 500, 12px)
   └─ tbody: zebra rows             (odd rows table.striped)
        Name (normal weight) | Value (break-all)
```

| Property      | Value                                                         | Source                                   |
| ------------- | ------------------------------------------------------------- | ---------------------------------------- |
| Header labels | `Name`, `Value`                                               | `components/ResponsePane/ResponseHeaders/index.js:18-19`         |
| Cell padding  | `4px 8px`; outer left/right borders removed                   | `components/ResponsePane/ResponseHeaders/StyledWrapper.js:25-35` |
| Row borders   | `1px solid table.border`; last row's bottom border removed    | `:26`, `:47-49`                          |
| Zebra         | `tr:nth-child(odd)` → `table.striped` (`#f1f1f1` / `#1e1e1e`) | `:42-45`                                 |
| Sorted?       | No — `Object.entries(headers)` order                          | `components/ResponsePane/index.js:7`                             |
| Empty         | Renders header row only, **no** empty-state text              | `components/ResponsePane/index.js:32`                            |
| Scroll        | persisted `response-headers-scroll-{uid}`                     | `components/ResponsePane/index.js:9`                             |

Tab indicator: `Headers` gains `<sup>` with the header count when > 0
(`components/ResponsePane/index.js:132`).

### 5.1 Timeline entry headers (different component)

Inside a Timeline entry the `Headers` block is a different table
(`components/ResponsePane/Timeline/TimelineItem/Common/Headers/index.js:12-50`):

- Section header button `Headers` + `({count})` in `tl-block-count`
  (11 px, muted, `opacity .65`), chevron 12 toggle (`:19-31`).
- Empty: `No Headers` in `.tl-empty` (12 px muted) (`:34`).
- Table: mono 12 px, `td` padding `5px 10px 5px 0`, key column
  `220px` (min 120, max 280) muted, value in `text`; row hover
  `rgba(text,.03)` (`TimelineItem/StyledWrapper.js:86-116`).

---

## 6. Timeline tab — `Timeline`

```text
Timeline (position:relative; overflow-y:auto; height:100%; flex:1)
├─ [filter bar]  .timeline-filter-bar          ← only if entries exist AND pre/post/oauth kinds present
│  └─ "All N"  "Request N"  "Pre-Request N"  "Post-Response N"  "OAuth N"
└─ .timeline-container
   └─ .timeline-event × N                      ← newest first
      └─ TimelineItem
         ├─ .tl-row  [chev] [status] [method] [url] [badge] [relative-time]
         └─ [expanded] .tl-detail
            ├─ .tl-header   METHOD url  ·  ↗ source-file
            ├─ .tl-tabs     Request | Response | Network
            └─ .tl-panel    ← content per tab
```

### 6.1 Row grid

`grid-template-columns: 14px auto 50px minmax(0,1fr) 96px 100px;
column-gap: 10px; padding: 7px 4px; border-top: 1px solid border.border1;
transition: background-color .08s ease` (`TimelineItem/StyledWrapper.js:9-21`).

| Cell    | Content                                                                                         | Style                                                    | Source                                                    |
| ------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------- | --------------------------------------------------------- |
| chevron | `IconChevronRight` / `IconChevronDown` 14/2                                                     | `colors.text.muted`, `opacity .7`                        | `TimelineItem/index.js:150-152`, `components/ResponsePane/StyledWrapper.js:36-43` |
| status  | numeric code, or `statusText`, or `Error`                                                       | see §6.2                                                 | `components/ResponsePane/index.js:66-76`, `:153-155`                              |
| method  | uppercase verb                                                                                  | `request.methods[verb]` else `text`, `font.size.xs`, 500 | `Common/Method/index.js:4-16`                             |
| url     | raw URL, `title` = URL                                                                          | 13 px, `text`, ellipsis                                  | `components/ResponsePane/index.js:159`, `components/ResponsePane/StyledWrapper.js:64-67`                  |
| badge   | `request` / `oauth2.0` / `sendRequest` / `runRequest`                                           | see §6.3                                                 | `components/ResponsePane/index.js:160-162`, `entryMeta.js:3-8`                    |
| time    | relative, 1 s tick (`Intl.RelativeTimeFormat('en', {numeric:'auto'})`), `title` = locale string | 11 px, muted, right-aligned                              | `Common/Time/index.js:4-48`                               |

Row states: hover `rgba(text,.04)`, expanded `rgba(text,.06)`,
keyboard focus `2px solid textLink` with `outline-offset:-2px`
(`components/ResponsePane/StyledWrapper.js:22-31`). First row has no top border (`:32-34`).
The row is a `role="button" tabIndex=0` that toggles with `Enter`/`Space`
(`TimelineItem/index.js:50-55`, `:141-149`).

### 6.2 Status badge (Timeline variant)

Not the same as `StatusCode` from the seam:

| Code range     | Colour                          | Source                         |
| -------------- | ------------------------------- | ------------------------------ |
| `200–299`      | `requestTabPanel.responseOk`    | `Common/Status/index.js:10-11` |
| `300–399`      | `colors.text.warning`           | `:12-13`                       |
| `400–599`      | `requestTabPanel.responseError` | `:14-15`                       |
| other / string | `colors.text.muted`             | `:9`                           |

Background `rgba(colour,.12)` when the code is known, else `transparent`;
`font-weight:600; font-size:11px; padding:2px 8px; border-radius:3px;
letter-spacing:.02em` (`:18-34`).

### 6.3 Entry badges

| Kind  | Badge label   | Chip label      | Background / colour                | Source                                     |
| ----- | ------------- | --------------- | ---------------------------------- | ------------------------------------------ |
| main  | `request`     | `Request`       | `colors.text.green @14%` / green   | `entryMeta.js:4`, `components/ResponsePane/StyledWrapper.js:85-88` |
| oauth | `oauth2.0`    | `OAuth`         | `textLink @12%` / textLink         | `entryMeta.js:5`, `:89-92`                 |
| pre   | `sendRequest` | `Pre-Request`   | `colors.text.yellow @12%` / yellow | `entryMeta.js:6`, `:93-96`                 |
| post  | `runRequest`  | `Post-Response` | `colors.text.purple @14%` / purple | `entryMeta.js:7`, `:97-100`                |

Badge base: 10 px, 600, `padding 2px 8px`, `border-radius:10px`,
`letter-spacing .02em` (`components/ResponsePane/StyledWrapper.js:75-84`).

### 6.4 Filter chips

- Visibility: the bar renders only when `entries.length > 0` and at least
  one `pre`, `post` or `oauth` entry exists (`components/ResponsePane/Timeline/index.js:71-72`);
  a chip is shown only when its count > 0 (`:70`). Order: `All`, `Request`,
  `Pre-Request`, `Post-Response`, `OAuth` (`entryMeta.js:10-16`).
- Bar: `gap:6px; padding:10px 0; border-bottom:1px solid border.border1;
margin-bottom:4px` (`components/ResponsePane/StyledWrapper.js:13-21`).
- Chip: `padding 4px 10px`, `radius 4px`, transparent, 12 px, 500,
  muted; hover `text` + `rgba(255,255,255,.04)`; active `text` +
  `rgba(255,255,255,.06)`; transition `color .1s, background-color .1s`
  (`:23-47`).
- Count chip: 11 px, 500, muted, `opacity .6`, tabular figures; active chip
  count takes `tabs.active.border` at full opacity (`:49-60`).
- Selecting a filter hides rows of other kinds; if the active chip becomes
  empty it resets to `all` (`components/ResponsePane/Timeline/index.js:74-78`, `:105`).

Ordering: newest first by timestamp (`buildEntries.js:68`). OAuth entries are
expanded into one row per sub-call and anchored to the paired main request's
timestamp; they are dropped when they have no sub-calls
(`buildEntries.js:12-51`).

### 6.5 Expanded detail

- `.tl-detail` `border-top: 1px dashed border.border1; margin-top: 4px`
  (`components/ResponsePane/StyledWrapper.js:102-105`).
- Header: `padding:10px 12px 10px 28px; gap:12px`; `METHOD url` in mono
  13 px with the verb 600 uppercase + 6 px gap; source link mono 11 px,
  `max-width:260px`, muted, `↗` in `textLink`, `title="Open {file}"` when
  navigable (`components/ResponsePane/StyledWrapper.js:107-151`, `components/ResponsePane/index.js:172-189`).
- Tabs: `padding: 0 12px 0 14px; border-bottom:1px solid border.border1`;
  each `padding: 9px 14px`, 12 px, muted; active `tabs.active.color` with a
  2 px underline (left/right 14 px). Labels: `Request`, `Response`, and
  `Network` only when `response.timeline.length > 0`
  (`components/ResponsePane/Timeline/TimelineItem/index.js:132-136`).
- Panel `padding: 12px 12px 14px 28px`; blocks are collapsible with a
  11 px uppercase 600 header, chevron 12, optional `({count})`
  (`components/ResponsePane/StyledWrapper.js:23-24`, `:43-84`).

Response tab content (`TimelineItem/Response/index.js`):

- Meta line: `{code} {statusText}` 13 px 700 coloured by the same ranges as
  §6.2, then `{Math.round(duration)}ms` and `formatSize(size)` (12 px muted)
  (`:24-43`).
- `Headers` block then `Body` block (`:52-70`).
- Body block renders `QueryResponse` in a `h-96` (384 px) scroll area; when
  there is no body it shows `No Body` (`Common/Body/index.js:23-41`).

### 6.6 Clear Timeline

The Timeline tab's actions slot shows `Clear Timeline`
(`components/ResponsePane/index.js:275-276`). See
`screens/response-pane.md` §3.3.

### 6.7 Runner timeline

`RunnerTimeline` reuses `TimelineItem` with `hideTimestamp` and builds
entries from the runner item only (`components/ResponsePane/RunnerTimeline/index.js:8-39`) —
main request, then OAuth debug entries, then scripted entries, sorted
newest-first. There is **no** filter bar.

---

## 7. Tests tab — `TestResults` + `TestResultsLabel`

### 7.1 Tab label

`TestResultsLabel` (`components/ResponsePane/TestResultsLabel/index.js:4-38`):

- No results in any of the four buckets → plain text `Tests`.
- Otherwise `Tests` + `<sup>` where the sup is **failed count in
  `colors.text.danger`** when any failed, else **total count in
  `colors.text.green`** (`:32-36`). Both `ml-1 font-medium`.
- Counts are summed across `testResults`, `assertionResults`,
  `preRequestTestResults`, `postResponseTestResults` (`:26-27`).
- Sup colours come from `components/ResponsePane/StyledWrapper.js:86-92`
  (`.some-tests-failed` / `.all-tests-passed`, both `!important`).

### 7.2 Panel

```text
TestResults (flex column, text)
├─ Pre-Request Tests (n), Passed: p, Failed: f
├─ Post-Response Tests (n), Passed: p, Failed: f
├─ Tests (n), Passed: p, Failed: f
└─ Assertions (n), Passed: p, Failed: f
```

| Property          | Value                                                                                   | Source                              |
| ----------------- | --------------------------------------------------------------------------------------- | ----------------------------------- |
| Empty state       | `No tests found` (plain div, 13 px body)                                                | `components/ResponsePane/TestResults/index.js:121-123`      |
| Section titles    | exactly `Pre-Request Tests`, `Post-Response Tests`, `Tests`, `Assertions`               | `:128`, `:137`, `:146`, `:155`      |
| Section summary   | `{title} ({results.length}), Passed: {passed}, Failed: {failed}`                        | `:72`                               |
| Summary row       | `font-medium`, `hover:bg-gray-500/10`, `rounded`, `py-2`, `mb-4`; chevron 18/1.5 `mr-2` | `:60-70`                            |
| Default expansion | each section expanded iff it has results                                                | `:105-112`                          |
| Row               | `<li class="py-1">` in `ul.ml-5`                                                        | `:76-82`                            |
| Pass icon         | `IconCircleCheck` 14, colour `colors.text.green`, `aria-label="Test passed"`            | `:12-23`                            |
| Fail icon         | `IconCircleX` 14, colour `colors.text.danger`, `aria-label="Test failed"`               | `:12-23`                            |
| Row text          | test → `result.description`; assertion → `{lhsExpr}: {rhsExpr}`                         | `:34-44`                            |
| Error text        | `role="alert"`, `pl-8`, `colors.text.muted`, on a new line                              | `:25-32`                            |
| Section hover     | `sidebar.collection.item.hoverBg`                                                       | `components/ResponsePane/TestResults/StyledWrapper.js:6-13` |
| Transitions       | summary `background-color .2s`; list `all .3s ease`                                     | `:6-8`, `:35-37`                    |
| Scroll            | persisted `response-tests-scroll-{uid}`                                                 | `components/ResponsePane/index.js:95`                       |

Screenshot `refs/tests_bruno.png` shows the _request_ Tests editor, not this
panel — no capture of the response Tests tab exists.

---

## 8. Response example pane — `ResponseExample`

Opened as a tab for a saved example; chrome mirrors the request tab
(`components/ResponseExample/index.js:182-235`).

```text
ResponseExample
├─ ResponseExampleTopBar  (p-4)
│  ├─ view mode:
│  │   h2 "[item.name] / [example.name]"        ← item.name at opacity .6
│  │   description (max 2 lines, "View More"/"View Less")
│  │   buttons: [IconCode]  ·  "Edit Example"  ·  "Try"
│  └─ edit mode:
│      input  "Enter example name"
│      textarea "Enter example description"
│      buttons: "Cancel"  ·  [IconDeviceFloppy] "Save"
└─ main (mt-4, pb-4)
   ├─ request pane (width max(left,300px))
   ├─ dragbar
   └─ response pane
      ├─ tabs: "Response" (or expectedResponseLabel) | "Headers" N  | ["Try Result"]
      │        right: ResponseLayoutToggle + StatusCode | StatusTextInput
      └─ content (HeightBoundContainer)
```

| Element                 | Verbatim copy / spec                                                                                        | Source                                              |
| ----------------------- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| Title                   | `{item.name} / {example.name}` (item name at `opacity-60`)                                                  | `ResponseExampleTopBar/index.js:166-170`            |
| Description toggle      | `View More` / `View Less`                                                                                   | `:118-119`                                          |
| Generate code           | icon-only, `title="Generate Code"`                                                                          | `:126-133`                                          |
| Edit                    | `Edit Example`                                                                                              | `:134-142`                                          |
| Try                     | `Try`, `title="Open this example as a new request and send it"`, `loading` while pending                    | `:143-154`                                          |
| Save (edit)             | `Save`, `padding: 6px 12px`, primary                                                                        | `:84-92`                                            |
| Cancel (edit)           | `Cancel`, secondary                                                                                         | `:77-83`                                            |
| Name placeholder        | `Enter example name`                                                                                        | `:58`                                               |
| Description placeholder | `Enter example description` (3 rows)                                                                        | `:64-69`                                            |
| Tabs                    | label from `expectedResponseLabel`, default `Response`; `Headers` with count; `Try Result` only after a try | `ResponseExampleResponsePane/index.js:83-97`, `:22` |
| Status                  | `StatusCode status=… statusText=…` (read-only); edit mode swaps in `ResponseExampleStatusInput`             | `:113-130`                                          |
| Layout toggle           | `ResponseLayoutToggle` at the right of the tab row                                                          | `:114`                                              |
| Pane minimums           | left `300px`, right `350px`, top/bottom `150px`                                                             | `components/ResponseExample/index.js:13-16`         |
| Keyboard                | `Ctrl`/`⌘`+`S` saves in edit mode                                                                           | `:165-180`                                          |

Toasts: `Failed to try example` (error, on try failure)
(`components/ResponseExample/index.js:145`).

> Example creation itself (the `Save Response as Example` modal opened by
> the bookmark button) is specified with the request-docs/examples screen
> and is out of scope here.

---

## 9. Viewer state matrix

| State              | Trigger                         | What the viewer shows                                |
| ------------------ | ------------------------------- | ---------------------------------------------------- |
| Editor, no content | empty body                      | Empty read-only editor                               |
| Editor, pretty     | JSON/HTML/XML/JS/raw/hex/base64 | Syntax-highlighted text (§3.1)                       |
| Preview web        | `html`, `javascript`            | Sandboxed webview                                    |
| Preview JSON tree  | `json`                          | Collapsible tree or `Cannot preview as JSON` banner  |
| Preview media      | `base64`/`hex` + media sniff    | `<img>` / audio / video / PDF pages                  |
| Preview text       | `raw` and others                | Mono text with clickable http(s) links               |
| No preview         | unknown mode                    | `No Preview Available` (§3.3)                        |
| Error body         | `response.error`                | `div.error` text, `pre-line` (§4.5 of response-pane) |
| Large body         | > 10 MB                         | `LargeResponseWarning`                               |
| Filter open        | JSON + editor                   | Filter input overlay                                 |
| Headers empty      | no headers                      | Header row only                                      |
| Timeline empty     | no entries                      | Nothing rendered (no empty text)                     |
| Tests empty        | no results                      | `No tests found`                                     |
| Example edit       | Edit Example                    | Editable top bar + status input                      |

---

## 10. Interactions

| Action                         | Trigger                                  | Effect                                                                 | Source                                                                               |
| ------------------------------ | ---------------------------------------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Change format                  | pick a menu item                         | `onFormatChange(id)` → persisted `responseFormat`                      | `QueryResultTypeSelector/index.jsx:89-100`, `components/ResponsePane/index.js:79-81` |
| Toggle preview                 | `Preview` switch in menu header          | flips `editor` ↔ `preview`                                             | `QueryResultTypeSelector/index.jsx:105-117`                                          |
| Toggle preview (shortcut)      | first click on selector while inactive   | activates the tab                                                      | `:54-62`                                                                             |
| Expand/collapse timeline entry | click row, `Enter`, `Space`              | toggles; expansion persisted per timestamp (`timeline-{timestamp}`)    | `TimelineItem/index.js:41-55`                                                        |
| Switch entry tab               | click `Request` / `Response` / `Network` | lazy-mounts tabs on first visit                                        | `:61-64`, `:191-203`                                                                 |
| Filter timeline                | chip click                               | client-side kind filter                                                | `components/ResponsePane/Timeline/index.js:87-98`                                                            |
| Open link in body              | click underlined link / JSON value       | sends that URL as a new request (same request type)                    | `QueryResultPreview/index.js:54-58`                                                  |
| Expand test section            | click summary row                        | show/hide rows                                                         | `components/ResponsePane/TestResults/index.js:61-64`                                                         |
| Toggle filter                  | filter icon                              | expand/collapse and clear on close                                     | `QueryResultFilter/index.js:9-22`                                                    |
| Clear timeline                 | `Clear Timeline`                         | wipes `collection.timeline` for this item                              | `components/ResponsePane/ClearTimeline/index.js:9-21`                                                        |
| Jump to source                 | `↗` in timeline header                   | opens the script/tests tab of the owning collection, folder or request | `TimelineItem/index.js:98-130`                                                       |

No keyboard shortcut is bound inside the viewer except the global
`Ctrl`/`⌘`+`S` in the example editor.

---

## Open Questions

1. **XML filter in the main pane.** `QueryResultFilter` supports XPath
   (`QueryResultFilter/index.js:35-37`) but `QueryResult` only mounts it for
   JSON (`components/ResponsePane/QueryResult/index.js:176`). Unclear whether XPath filtering is
   reachable from the response pane at all.
2. **Hex/Base64 in editor mode.** The editor shows the raw text for these
   formats (`text/plain`) while preview upgrades to a media player; whether
   the `Hex` label implies a hex dump of binary bytes is not implemented in
   source (no `hex`-specific formatter found in `utils/common`).
3. **`text-gray-*` colours in `QueryResultPreview` default state** are
   Tailwind palette values, not design tokens, so they do not follow theme
   changes.
4. **`background.elevated`** is referenced by `CodeSnippet`
   (`CodeSnippet/StyledWrapper.js:13`) but is absent from
   `ui-map/_tools/tokens.json`; the rendered value is unverified.
5. **ResponseExample tab persistence** writes to the _active tab_ uid rather
   than the example uid in some paths (`ResponseExampleResponsePane/index.js:33-38`
   uses `exampleUid`, but `activeTab` is read from `focusedTab`) — unverified
   which wins in practice.
6. **Timeline empty state.** When an item has no entries the Timeline tab
   renders an empty container with no message; a Go port may want an empty
   state, but none exists in source.
