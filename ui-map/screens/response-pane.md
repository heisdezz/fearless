# Response pane — `ResponsePane` + `RequestTabPanel` response area

Status: VERIFIED against `bruno/packages/bruno-app/src/` at the commit in
`overview.md:8`. Screenshots: `refs/api_screen_bruno.png` (empty state, dark),
`refs/main_app_bruno.png`. The body viewer, Headers/Timeline/Tests panels are in
`screens/response-viewer.md`; the two tiny indicators are in
`components/spinner.md` and `components/status-dot.md`.

All paths are relative to `bruno/packages/bruno-app/src/` unless prefixed.
Tokens are quoted as `theme.*` paths from
`ui-map/_tools/tokens.json`; resolved values are repeated where a Go port
needs the literal.

---

## 1. Placement in the shell

Vertical/horizontal is the **orientation**, stored in preferences as
`layout.responsePaneOrientation` (`components/ResponsePane/ResponseLayoutToggle/index.js:57`),
default `'horizontal'`.

| Orientation            | Request pane                                                                                         | Response pane                                                                            | Dragbar                                                                                     |
| ---------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `horizontal` (default) | left, fixed width `max(leftPaneWidth, 350px)` (`components/RequestTabPanel/index.js:55`, `:698-700`) | right, `flex-grow`, `min-width: 0` (`components/RequestTabPanel/StyledWrapper.js:16-18`) | vertical bar, `12px` wide, 1 px left border (`components/ResponsePane/StyledWrapper.js:39-62`)                      |
| `vertical`             | top, height `max(topPaneHeight, 150px)`, `width: 100%` (`components/ResponsePane/index.js:57`, `:693-697`)                   | bottom                                                                                   | horizontal bar, `12px` high, `padding: 0 1rem`, 1 px top border (`components/ResponsePane/StyledWrapper.js:81-101`) |

- Pane minimums: left `350px`, right `490px`, top `150px`, bottom `150px`,
  collapse threshold `80px`, expand threshold `100px`, click-expand for the
  response requires `300px` (`components/RequestTabPanel/index.js:55-62`).
- Dragbar hairline: `requestTabPanel.dragbar.border` → light `#e5e5e5`,
  dark `#333333`; on hover `dragbar.activeBorder` → `#cccccc` / `#444444`.
- The response pane is rendered by `renderResponsePane()`
  (`components/RequestTabPanel/index.js:673-683`): gRPC → `GrpcResponsePane`,
  WebSocket → `WSResponsePane`, everything else → `ResponsePane`.
- The response pane's parent section is `section.response-pane.flex-grow.overflow-x-auto`
  (`components/RequestTabPanel/index.js:746`, `RequestTabPanel/StyledWrapper.js:16`).
- Horizontal: `request-pane` gets `padding-bottom: 0.5rem` and `response-pane`
  `padding-top: 0.5rem` in vertical layout only (`components/ResponsePane/StyledWrapper.js:64-79`).
- While dragging, a fixed full-window overlay sets `cursor: col-resize`
  (`row-resize` in vertical) and `user-select: none`
  (`components/RequestTabPanel/index.js:761-771`).

Screenshot `refs/api_screen_bruno.png` shows the horizontal split with the
empty pane on the right.

---

## 2. Layout tree (HTTP/GraphQL, has response)

```text
section.response-pane (/RequestTabPanel:746)
└─ ResponsePane  div.flex.flex-col.h-full.relative   (/ResponsePane/StyledWrapper.js:3)
   ├─ div.px-4                                        (/ResponsePane/index.js:294)
   │  └─ ResponsiveTabs  div.tabs                     (ui/ResponsiveTabs/index.js:243)
   │     ├─ div.flex.items-center                     (tabs left group)
   │     │  ├─ [hidden measure row]
   │     │  ├─ div.tab.response  "Response"
   │     │  ├─ div.tab.headers   "Headers" <sup>N</sup>
   │     │  ├─ div.tab.timeline  "Timeline"
   │     │  ├─ div.tab.tests     TestResultsLabel  ("Tests" + <sup>count</sup>)
   │     │  └─ div.more-tabs  (only when tabs overflow: IconChevronsRight 18)
   │     └─ div.flex.justify-end.items-center (+ .expandable when wide)
   │        ├─ [errors icon]  ResponseErrorsIcon      (only when card is closed)
   │        ├─ div.result-view-tabs                   (only on Response tab, not streaming)
   │        │  └─ QueryResultTypeSelector  [Eye|Brace] Label ⌄
   │        ├─ div.response-pane-status                (data-testid=response-pane-status)
   │        │  ├─ StatusCode   "200 OK" | "200 OK - STREAMING"
   │        │  ├─ ResponseTime | ResponseStopWatch     (stopwatch while streaming)
   │        │  └─ ResponseSize  "1.2KB"
   │        └─ div.response-pane-actions
   │           ├─ ClearTimeline "Clear Timeline"      (only on Timeline tab)
   │           └─ ResponsePaneActions                 (only when response && !error)
   │              ├─ div.actions-dropdown  → ⋯ menu
   │              └─ div.actions-buttons   → copy, bookmark, download, clear, layout
   └─ section.response-pane-content                   (/ResponsePane/index.js:304)
      ├─ Overlay            (isLoading)
      ├─ ResponseAlertsSection
      │  ├─ ResponseErrorsCard
      │  └─ UnresolvedVariablesInfo
      └─ div.response-tab-content                     (data-testid=response-tab-content)
         └─ QueryResult | ResponseHeaders | Timeline | TestResults
```

Container metrics (`components/ResponsePane/StyledWrapper.js`):

| Region                                             | Rule                                                                                             | Line      |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------ | --------- |
| `.response-pane-content`                           | `display:flex; flex-direction:column; flex:1 1 0; min-height:0; padding:0 1rem; margin-top:1rem` | `:52-60`  |
| `.response-tab-content`                            | `flex:1; overflow-y:auto; min-height:0`                                                          | `:62-66`  |
| `.right-side-container`                            | `min-width:0; flex-shrink:1; flex-grow:1`                                                        | `:68-72`  |
| `.response-pane-status` / `.response-pane-actions` | `flex-grow:0; flex-shrink:1; min-width:0`                                                        | `:74-84`  |
| `.result-view-tabs`                                | `display:flex; align-items:center; gap:2px; padding:3px; border-radius:8px`                      | `:94-119` |

### 2.1 Tabs

| Tab        | Label                                            | Indicator                                                          | Source                                     |
| ---------- | ------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------ |
| `response` | `Response`                                       | none                                                               | `components/ResponsePane/index.js:123-128` |
| `headers`  | `Headers`                                        | `<sup class="ml-1 font-medium">{headerCount}</sup>` when count > 0 | `:129-133`                                 |
| `timeline` | `Timeline`                                       | none                                                               | `:134-138`                                 |
| `tests`    | `TestResultsLabel` (renders `Tests` + sup count) | —                                                                  | `:139-151`                                 |

- Tab bar: `gap-6` between left group and right content (`ui/ResponsiveTabs/index.js:243`);
  each tab `padding: 6px 0`, `border-bottom: 2px solid transparent`,
  `margin-right: tabs.marginRight` (`1.2rem`), `font-size: font.size.sm`
  (12 px), inactive `colors.text.subtext0` (`#9b9b9b` / `#999`)
  (`ui/ResponsiveTabs/StyledWrapper.js:23-36`).
- Active tab: `font-weight: tabs.active.fontWeight` (`400`), colour
  `tabs.active.color` (`#343434` / `#cccccc`), underline
  `2px solid tabs.active.border` (`#d5882a` / `#e7b85f`)
  (`ui/ResponsiveTabs/StyledWrapper.js:51-55`; tokens `tokens.json`).
- Hover (inactive): `tabs.active.color` (`:47-49`).
- `sup` inside a tab: `line-height:1; vertical-align:baseline; margin-left:0`
  (`components/ResponsePane/StyledWrapper.js:75-81`).
- Overflow: tabs are measured against container width; the active tab is
  always kept visible by evicting the last visible tab
  (`ui/ResponsiveTabs/index.js:47-82`). The overflow trigger is
  `IconChevronsRight` 18 px, colour `colors.text.subtext0`
  (`components/ResponsePane/index.js:263-274`, `components/ResponsePane/StyledWrapper.js:14-21`); the menu is a
  `MenuDropdown` with `placement="bottom-start"`, active tab marked with
  `selectedItemId` (`components/ResponsePane/index.js:264-269`).
- Right-content "expandable" mode: whether the inline action buttons or the
  ⋯ menu is shown is decided by `ResponsiveTabs` (`rightContentExpandedWidth`
  = `135`, `components/ResponsePane/index.js:32`, `:301`). The gap reserved
  between left tabs and right content is `80px`, the overflow dropdown is
  `60px`, with `20px` hysteresis (`ui/ResponsiveTabs/index.js:7-11`).
  `ResponsePaneActions` consumes this via the `.expandable` class
  (`components/ResponsePane/ResponsePaneActions/StyledWrapper.js:17-28`).

### 2.2 Status seam (right of tabs)

Order is fixed: **StatusCode, ResponseTime|StopWatch, ResponseSize**, then the
actions block (`components/ResponsePane/index.js:266-288`).

| Element             | Text                                                                             | Typography                                                                          | Colour token                                                                                                                                                  | File:line                                                                                          |
| ------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `StatusCode`        | `{status} {statusText \|\| phrase}` and, when streaming, ` - STREAMING` appended | `font.size.sm` (12 px), **600**, `white-space: nowrap`                              | 1xx/2xx `requestTabPanel.responseOk` (`#2e8a54` / `#73e89a`); 3xx/4xx/5xx `requestTabPanel.responseError` (`#ce4f3b` / `#da462f`); no class → inherits `text` | `components/ResponsePane/StatusCode/index.js:7-23`, `components/ResponsePane/StatusCode/StyledWrapper.js:3-15`                                     |
| `ResponseTime`      | `formatDuration(duration)` → `123ms` under 1 s, else `1.23s` (2 dp)              | `font.size.sm`, **500**                                                             | `requestTabPanel.responseStatus` (`#838383` / `#ccc`)                                                                                                         | `components/ResponsePane/ResponseTime/index.js:6-12`, `utils/common/index.js:227-233`, `components/ResponsePane/ResponseTime/StyledWrapper.js:3-7` |
| `ResponseStopWatch` | `12.3s` (1 dp, ticks every 100 ms)                                               | `font.size.sm`, **500**, `text-align:center`, width `chars × 0.4rem` to stop reflow | `requestTabPanel.responseStatus`                                                                                                                              | `components/ResponsePane/ResponseStopWatch/index.js:4-28`, `components/ResponsePane/StyledWrapper.js:3-8`                                          |
| `ResponseSize`      | `formatSize(size, 2)` → `512B` / `1.20KB` / `3.00MB`; `title` = `"1,234,567B"`   | `font.size.sm`, **500**                                                             | `requestTabPanel.responseStatus`                                                                                                                              | `components/ResponsePane/ResponseSize/index.js:5-17`, `utils/common/index.js:206-224`, `components/ResponsePane/StyledWrapper.js:3-7`              |

Spacing: `ResponseTime`, `ResponseStopWatch` and `ResponseSize` all carry
`ml-2` (8 px); StatusCode has no left margin, so the seam begins flush at
the tab bar's right edge. `ResponseTime`/`ResponseSize` render `null` when
the value is not a finite number (`components/ResponsePane/ResponseTime/index.js:7-9`,
`components/ResponsePane/ResponseSize/index.js:6-8`); `ResponseStopWatch` shows `0.0s` for missing
timestamps (`components/ResponsePane/ResponseStopWatch/index.js:18-21`).

Size derivation: `response.size` if numeric, else decoded base64 buffer
length, else `0` (`components/ResponsePane/index.js:105-119`).

`StatusCode` phrase map (`components/ResponsePane/StatusCode/get-status-code-phrase.js:2-66`) — keep
this table as data; unknown codes render `{status} {statusText || phrase}`
with an undefined phrase (`components/ResponsePane/index.js:20`) — see Open Questions #1.

### 2.3 Status seam states

| Pane state                                      | Seam shows                                                                                           | Line                                     |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| Never sent, no timeline                         | no tab bar at all (`Placeholder`)                                                                    | `components/ResponsePane/index.js:219-225`                       |
| Sending/queued, no response                     | no tab bar (`Overlay` only)                                                                          | `components/ResponsePane/index.js:211-217`                       |
| Sending/queued, previous response still present | full bar, `rightContent` is `null` while loading (no format selector, status, actions)               | `components/ResponsePane/index.js:243` (`!isLoading ? … : null`) |
| Streaming                                       | StopWatch replaces ResponseTime; `- STREAMING` suffix; no format selector                            | `components/ResponsePane/index.js:246`, `:267-271`               |
| Response settled                                | StatusCode + ResponseTime + ResponseSize, actions visible                                            | `components/ResponsePane/index.js:243-289`                       |
| Response has `error`                            | status seam still renders; `ResponsePaneActions` hidden (`item?.response && !item?.response?.error`) | `components/ResponsePane/index.js:276-287`                       |
| Skipped                                         | whole pane replaced by `SkippedRequest`                                                              | `components/ResponsePane/index.js:203-209`                       |

---

## 3. Actions block

`ResponsePaneActions` (`components/ResponsePane/ResponsePaneActions/index.js`).
Rendered only for `http-request` and `graphql-request` (`:145-147`).

### 3.1 Wide (`> expandable`): inline icon buttons

Order left→right: **Copy, Bookmark, Download, Clear, LayoutToggle**
(`:160-173`). Each is a 16 px Tabler glyph inside `ui/ActionIcon size="md"`
(`24×24` box, `padding:0`, `border-radius:4px`,
`transition: all 0.15s ease` — `ui/ActionIcon/StyledWrapper.js:3-35`), wrapped
with `ActionIcon className="p-1"` (4 px padding) giving a **32×32** hit box.
Gap between buttons: `2px` (`components/ResponsePane/ResponsePaneActions/index.js:160`).

| Button   | Icon (Tabler)                                                           | Stroke | Tooltip (`title`) verbatim                                                                                                                                                                       | Disabled when                                                                         | File:line                                     |
| -------- | ----------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- | --------------------------------------------- |
| Copy     | `IconCopy` → after click `IconCheck` for 2 s                            | 2      | `Copy response to clipboard`                                                                                                                                                                     | `!data` (also `aria-disabled`, `opacity-50 cursor-not-allowed`)                       | `components/ResponsePane/ResponseCopy/index.js:53-91`                 |
| Bookmark | `IconBookmark`                                                          | 2      | `Save current response as example`; when streaming `Response Examples aren't supported in streaming responses yet.`; when size ≥ 5 MB `Response size exceeds 5MB limit. Cannot save as example.` | HTTP only (hidden for GraphQL); disabled if `response.stream` or `size ≥ 5*1024*1024` | `components/ResponsePane/ResponseBookmark/index.js:17-47`, `:139-167` |
| Download | `IconDownload`                                                          | 2      | `Save response to file`                                                                                                                                                                          | `!response.dataBuffer \|\| response.stream?.running`                                  | `components/ResponsePane/ResponseDownload/index.js:9-59`              |
| Clear    | `IconEraser`                                                            | 2      | `Clear response`                                                                                                                                                                                 | never (`isDisabled: false`)                                                           | `components/ResponsePane/ResponseClear/index.js:25-45`                |
| Layout   | `IconLayoutRows`, or `IconLayoutColumns` when orientation is `vertical` | 2      | `Switch to vertical layout` / `Switch to horizontal layout`                                                                                                                                      | never                                                                                 | `components/ResponsePane/ResponseLayoutToggle/index.js:74-105`        |

Icon colours: `dropdown.iconColor` (`#666` / `#bbb`) on a transparent face; hover
face `workspace.button.bg` (`#f8f8f8` / `#3d3d3d`) and glyph `text`
(`#343434` / `#ccc`) (`components/ResponsePane/ResponseCopy/StyledWrapper.js:3-12`,
`components/ResponsePane/ResponseClear/StyledWrapper.js:3-12`, `ResponseDownload/StyledWrapper.js`,
`components/ResponsePane/ResponseBookmark/StyledWrapper.js:3-13`, `components/ResponsePane/ResponseLayoutToggle/StyledWrapper.js:3-19`).
Disabled: `opacity-50 cursor-not-allowed` on the wrapper
(`components/ResponsePane/ResponseCopy/index.js:80-84`), plus `ActionIcon:disabled { opacity:0.5; cursor:not-allowed }`
(`ui/ActionIcon/StyledWrapper.js:41-44`).

Toasts raised by these actions (copy exactly):

| Toast                                                      | Kind    | Source                          |
| ---------------------------------------------------------- | ------- | ------------------------------- |
| `Response copied to clipboard`                             | success | `components/ResponsePane/ResponseCopy/index.js:39`      |
| `Failed to copy response`                                  | error   | `components/ResponsePane/ResponseCopy/index.js:42`      |
| `Response downloaded to file`                              | success | `components/ResponsePane/ResponseDownload/index.js:29`  |
| `Something went wrong!` (or the IPC error message)         | error   | `components/ResponsePane/ResponseDownload/index.js:34`  |
| `No valid response to save as example`                     | error   | `components/ResponsePane/ResponseBookmark/index.js:51`  |
| `Response size exceeds 5MB limit. Cannot save as example.` | error   | `components/ResponsePane/ResponseBookmark/index.js:58`  |
| `Example "{name}" created successfully`                    | success | `components/ResponsePane/ResponseBookmark/index.js:131` |

Copy semantics: `preview` tab copies the raw text, `editor` tab copies
`formatResponse(data, dataBuffer, selectedFormat, null)`; otherwise
`JSON.stringify(data, null, 2)` (`components/ResponsePane/ResponseCopy/index.js:10-20`).

Bookmark opens `CreateExampleModal` titled `Save Response as Example`
(`components/ResponsePane/ResponseBookmark/index.js:161-167`).

### 3.2 Narrow: `⋯` menu

- Trigger: `div.actions-dropdown` → `MenuDropdown` `placement="bottom-end"`,
  trigger glyph `IconDots` 16 px `strokeWidth 1.5`, `title="More actions"`
  (`components/ResponsePane/ResponsePaneActions/index.js:13-37`, `:151-159`).
- Trigger box: `height 1.25rem` (20 px), `width 1.5rem` (24 px), `1px solid
workspace.border` (`#e5e5e5` / `#444`), `border-radius 4px`, colour
  `dropdown.iconColor`; hover border
  `app.collection.toolbar.environmentSelector.hoverBorder` (`#cccccc` /
  `#666666`) and colour `text` (`:13-27`).
- Items (HTTP): `Copy response`, `Save response`, `Download response`,
  `Clear response`, `Change layout` — each with `leftSection` = the same
  icon as its inline button; `Change layout` also shows the keybinding
  (`getKeybindingDisplayText('changeLayout')`, default `Ctrl + J` on Linux/Windows,
  `⌘ + J` on macOS — `providers/Hotkeys/keyMappings.js:43`).
- GraphQL drops `Save response` and shows the `gqlMenuItems` set
  (`components/ResponsePane/ResponsePaneActions/index.js:56-94`, `:153`).

### 3.3 Clear Timeline (Timeline tab only)

When the active pane tab is `timeline`, the actions slot swaps to
`ClearTimeline` (`components/ResponsePane/index.js:275-276`): a text button
`Clear Timeline`, `title="Clear Timeline"`, `font-size: font.size.base`
(13 px), colour `requestTabPanel.responseStatus`, link colour + underline on
hover (`components/ResponsePane/ClearTimeline/index.js:17-23`, `components/ResponsePane/StyledWrapper.js:3-6`).

---

## 4. Pane states

### 4.1 Loading (`queued` / `sending`)

`isLoading = ['queued','sending'].includes(item.requestState)`
(`components/ResponsePane/index.js:38`).

- No response yet: the pane is only the `Overlay` (`components/ResponsePane/index.js:211-217`).
- Previous response present: overlay covers the content, the tab bar and
  seam keep rendering (`components/ResponsePane/index.js:305`).

```text
div.response-pane-content
└─ Overlay (absolute, z-index 1)
   └─ div.overlay
      ├─ StopWatch       "Loading..." for <250 ms, then "3.4s"
      ├─ IconRefresh 24  (.loading-icon — rotateCounterClockwise 1s linear infinite)
      └─ Button[secondary, sm]  "Cancel Request"
```

| Property        | Value                                                                                                                                              | Source                                                                             |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Backdrop        | `requestTabPanel.responseOverlayBg` — `rgba(255,255,255,.6)` / `rgba(26,26,26,.8)`                                                                 | `components/ResponsePane/Overlay/StyledWrapper.js:8`                                                       |
| Size            | `position:absolute; height:100%; width:calc(100% - 0.75rem); z-index:1`                                                                            | `:4-8`                                                                             |
| Content         | `flex column; align-items:center; padding-top:20%; text-align:center`                                                                              | `:10-19`                                                                           |
| Vertical layout | content centred vertically, `padding:1rem`                                                                                                         | `:28-33`                                                                           |
| Stopwatch block | `margin-bottom:15px`, inline `font-size:20px`, `margin 0 5px`                                                                                      | `components/ResponsePane/Overlay/index.js:21-25`                                                           |
| Spinner         | `IconRefresh size={24}`, animation `rotateCounterClockwise 1s linear infinite` (`0%: scaleY(-1) rotate(360deg)` → `100%: scaleY(-1) rotate(0deg)`) | `components/ResponsePane/Overlay/index.js:26`, `components/ResponsePane/Overlay/StyledWrapper.js:20-23`, `globalStyles.js:180-188` |
| Cancel          | `Button color="secondary" size="sm"`, `className="mt-4"` — label `Cancel Request`                                                                  | `components/ResponsePane/Overlay/index.js:27-34`                                                           |

`StopWatch` (`components/StopWatch/index.js`): renders `Loading...` when
`startTime` is falsy or elapsed < 250 ms, else `{s.toFixed(1)}s`, ticking at
100 ms (`:16-22`).

### 4.2 Cancelled

There is **no dedicated cancelled view**. Cancelling dispatches
`cancelRequest` (`components/ResponsePane/Overlay/index.js:14-16`), the item returns to idle and the
pane falls through to whichever state applies: no response → `Placeholder`
(`components/ResponsePane/index.js:219-225`); a partial/last response → normal pane. A cancelled send
also loses `requestUid`, which the error-card persistence keys on
(`hooks/useResponsePaneErrors/index.js:11-13`).

### 4.3 Empty (`Placeholder`)

Shown when `!item.response && !requestTimeline.length` (`components/ResponsePane/index.js:219-225`),
wrapped in `HeightBoundContainer`.

```text
Placeholder (padding-top:20%, width:100%)
├─ .send-icon   IconSend, size 150 (horizontal) / 80 (vertical), strokeWidth 1, colour background.surface2
└─ two columns (mt-4 horizontal / mt-2 vertical)
   ├─ left  (items-end, px-1): "Send Request" | "New Request" | "Edit Environments"  (px-1 py-2)
   └─ right (px-1):            "Ctrl + Enter" | "Ctrl + N" | "Ctrl + E"              (px-1 py-2)
```

- Icon size: 150 px horizontal, 80 px vertical; wrapper font-size
  200/100 px (`components/ResponsePane/Placeholder/index.js:25`, `:32`).
- Icon colour `background.surface2` (`#e5e5e5` / `#666`
  — `components/ResponsePane/Placeholder/StyledWrapper.js:9-11`).
- Rows: `px-1 py-2` (4 px horizontal, 8 px vertical), labels right-aligned
  in the left column (`components/ResponsePane/index.js:35-49`).
- Shortcuts are user-rebindable and resolved at render
  (`getKeyBindingDisplayTextByOS`, `components/ResponsePane/index.js:17-23`). Defaults
  (`providers/Hotkeys/keyMappings.js:42`, `:58`, `:57`): Send Request
  `Ctrl + Enter` / `⌘ + ↩`; New Request `Ctrl + N` / `⌘ + N`; Edit
  Environments `Ctrl + E` / `⌘ + E`.
- Vertical layout: `padding:1rem; justify-content:center`, no `padding-top:20%`
  (`components/ResponsePane/StyledWrapper.js:13-16`).

`refs/api_screen_bruno.png` captures exactly this state.

### 4.4 Skipped request

When `item.response && item.status === 'skipped'` the whole pane becomes
`SkippedRequest` (`components/ResponsePane/index.js:203-209`):

- `IconCircleOff` 150 px, `strokeWidth 1`, colour `background.surface2`,
  `padding-top:20%`.
- Text `Request skipped`, `margin-top:1rem` (16 px), inline `font-size:25px`,
  centred.

(`components/ResponsePane/SkippedRequest/index.js:5-16`, `components/ResponsePane/StyledWrapper.js:3-9`.)

### 4.5 Error responses (HTTP error body)

`QueryResult` renders the error string in `div.error` (`color:
colors.text.danger`, `white-space: pre-line`) after stripping the prefix
`Error invoking remote method 'send-http-request':`
(`components/ResponsePane/QueryResult/index.js:31-42`, `:184-196`). Empty error → `Something went
wrong`. If the message contains `self signed certificate`, a hint appears
below `mt-6 muted text-xs` (colour `colors.text.muted`):

> You can disable SSL verification in the Preferences.
> To open the Preferences, click on the gear icon in the bottom left corner.

(`components/ResponsePane/QueryResult/index.js:190-195`.)

If a script error exists (`preRequestScriptErrorMessage` /
`postResponseScriptErrorMessage`) the inline error text is suppressed because
`ResponseErrorsCard` already shows it (`components/ResponsePane/QueryResult/index.js:177`, `:186`).

### 4.6 Network error — `NetworkError`

Not a pane state. `NetworkError` is a **toast payload** raised when
`sendRequest` rejects (`components/RequestTabPanel/index.js:612-618`), shown
via `toast.custom` for `5000` ms:

| Property  | Value                                                                                                                         | Source                         |
| --------- | ----------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| Container | `max-w-md w-full bg-white shadow-lg rounded-lg pointer-events-auto flex bg-red-100` (Tailwind defaults; red-100 is `#fee2e2`) | `components/ResponsePane/NetworkError/index.js:5`      |
| Padding   | `p-4` (16 px)                                                                                                                 | `:6`                           |
| Title     | `Network Error`, `font-medium text-red-800` (`#991b1b`)                                                                       | `:9`                           |
| Close     | `Close`, `p-4`, `font-medium`, transparent border, right-rounded                                                              | `:14-19`                       |
| Duration  | `5000` ms                                                                                                                     | `RequestTabPanel/index.js:615` |

Note: the two `bg-*` classes conflict in source (`bg-white` then
`bg-red-100`); the later class wins in the cascade order used by
Tailwind's build — treat the surface as `#fee2e2`.

### 4.7 Large response warning — `LargeResponseWarning`

Triggered inside `QueryResult` when `responseSize > 10 * 1024 * 1024`
(`components/ResponsePane/QueryResult/index.js:125`, `:197-202`).

```text
LargeResponseWarning (flex column, align-items:center)
├─ .warning-container (max-width 480px, margin-top 10%, margin-bottom 1.5rem, centred)
│  ├─ .warning-icon   IconAlertTriangle 45, strokeWidth 2, colour colors.text.yellow
│  ├─ .warning-title  "Large Response Warning"   (500, text, margin-bottom 1rem)
│  └─ .warning-description (colors.text.muted)
│       Handling responses over [10.0MB] could degrade performance.
│       <br>
│       Size of current response: [NN.NMB]
│       <br>  (only when size > MAX_RENDERABLE_RESPONSE_BYTES = 50 MB)
│       Responses over [50.0MB] can only be downloaded.
└─ .warning-actions (flex, gap 0.75rem)
   ├─ Button[secondary sm] IconEye 18/1.5    "View"      title "Show response content"
   ├─ Button[secondary sm] IconDownload 18/1.5 "Download" title "Download response to file"
   └─ Button[secondary sm] IconCopy 18/1.5   "Copy"      title "Copy response to clipboard"
```

| Detail              | Value                                                                                                | Source                                                    |
| ------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Thresholds          | warn above `10 MB`; download-only above `50 MB` (`MAX_RENDERABLE_RESPONSE_BYTES = 50 * 1024 * 1024`) | `components/ResponsePane/QueryResult/index.js:125`, `utils/common/constants.js:6` |
| Icon colour         | `colors.text.yellow` (`#bb931b` / `#fad075`)                                                         | `components/ResponsePane/LargeResponseWarning/StyledWrapper.js:18-21`             |
| Highlight chips     | `padding: 0.125rem 0.375rem; border-radius: 4px; font-size: 0.8rem`                                  | `:32-36`                                                  |
| Current size chip   | `colors.text.danger` on `…danger 15` (8-digit hex alpha)                                             | `:38-41`                                                  |
| Supported size chip | `colors.text.yellow` on `…yellow 15`                                                                 | `:43-46`                                                  |
| View disabled       | when download-only                                                                                   | `components/ResponsePane/LargeResponseWarning/index.js:78`                        |
| Download disabled   | `!response.dataBuffer && !response.storedRequestUid`                                                 | `:90`                                                     |
| Copy disabled       | download-only or `!response.data`                                                                    | `:102`                                                    |

Buttons `View`, `Download`, `Copy` are `ui/Button` `sm` (padding
`0.375rem 0.75rem`, font 12 px, gap `0.375rem` — `ui/Button/StyledWrapper.js:29-43`).
`View` reveals the body (`setShowLargeResponse(true)`), it does not dismiss
the warning.

### 4.8 Unresolved variables — `UnresolvedVariablesInfo`

Rendered by `ResponseAlertsSection` for every non-loading Response-tab view
when `item.unresolvedVariables.length > 0` and the errors card is not
full-pane (`components/ResponsePane/ResponseAlertsSection/index.js:10`, `:27`).

```text
UnresolvedVariablesInfo (role=status)
├─ IconInfoCircle 16/1.5   (.info-icon, colors.text.muted, margin-top .125rem)
├─ .info-message (flex:1, line-height 1.25rem, text)
│   inline form (joined names ≤ 40 chars):
│     This request uses variables that could not be resolved: <code>a</code>, <code>b</code>.
│   compact form (> 40 chars):
│     This request uses [3 variables] that could not be resolved.
└─ ActionIcon[xs] IconX 16/1.5  aria-label "Dismiss unresolved variables info"
```

| Property     | Value                                                                                                                                | Source                                          |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------- |
| Box          | `flex; align-items:flex-start; gap .5rem; padding .75rem`                                                                            | `components/ResponsePane/UnresolvedVariablesInfo/StyledWrapper.js:3-11` |
| Surface      | `background.base`, `1px solid border.subtle`, `radius.base`                                                                          | `:8-10`                                         |
| Threshold    | `MAX_INLINE_NAMES_LENGTH = 40` on `names.join(', ')`                                                                                 | `components/ResponsePane/UnresolvedVariablesInfo/index.js:10`, `:80`    |
| Singular     | `1 variable` / plural `variables`                                                                                                    | `:39`                                           |
| Code chip    | mono, `0.85em`, padding `.15em .35em`, bg `sidebar.bg` in light / transparent in dark, border `border.border0`                       | `components/ResponsePane/StyledWrapper.js:25-33`                        |
| Count link   | `textLink`, underlined, `text-underline-offset:2px`                                                                                  | `:35-44`                                        |
| Popover      | `place=bottom-start`, no arrow, `opacity 1`, clickable; header `Unresolved variables` + copy icon (`Copied` / `Copy variable names`) | `components/ResponsePane/index.js:41-69`, `components/ResponsePane/StyledWrapper.js:46-65`      |
| Popover size | `max-height 11.125rem; min-width 14rem; max-width min(32rem, 100vw-1.875rem)`                                                        | `components/ResponsePane/StyledWrapper.js:67-90`                        |
| Header size  | `font.size.xs`, padding `.375rem .625rem .125rem .875rem`                                                                            | `:57-65`                                        |
| Dismiss      | dispatches `dismissUnresolvedVariables`, persisted per item                                                                          | `components/ResponsePane/index.js:82-84`                                |

### 4.9 Response errors card — `ResponseErrorsCard`

Built by `utils/response-pane-errors/index.js` from the per-script error
fields, not from the response body. Labels come from `SCRIPT_TYPES`
(`:42-50`): `Pre-Request`, `Post-Response`, `Test`, `Before Call Start`,
`Before Message Send`, `After Message Receive`, `After Call End`.
Each entry: `id = scriptType`, `kind = 'script'`, `title = "{label} Script
Error"`, `label = "{label}"`, `preview = first line of message`
(`:63`). With context: `displayMessage = "{errorType}: {message}"`
(`errorType` defaults to `Error`), `location = "{filePath}:{line}"`
(`:69-74`).

```text
ResponseErrorsCard (role=region, aria-label=title)
├─ .card-header  (height 2.375rem; padding 0 .5rem 0 .75rem; gap .5rem)
│  ├─ IconAlertCircle 16/1.5   (.card-title-icon, colors.text.danger)
│  ├─ .card-title   600, ellipsis, colors.text.danger   ← "Pre-Request Script Error" | "3 errors"
│  └─ .card-actions (gap .125rem, margin-left:auto)
│     ├─ single → CopyErrorButton (IconCopy/IconCheck 16/1.5, "Copy error")
│     │  multi  → ActionIcon ("Expand all" | "Collapse all")
│     ├─ ActionIcon ("Expand to full pane" | "Restore height")
│     └─ ActionIcon ("Close")
└─ .card-body (max-height 10rem; overflow-y auto; scrollbar-gutter:stable)
   ├─ single  → ErrorDetails
   └─ multi   → CollapsibleError × N
                 ├─ .error-row-line (sticky, min-height 1.875rem, border-top border.border0)
                 │  ├─ chevron 14 + .error-row-label (500) + .error-row-preview (muted, ellipsis)
                 │  └─ CopyErrorButton
                 └─ ErrorDetails (when expanded, padding .625rem .75rem .625rem 2.25rem)
```

`ErrorDetails` (`ResponseErrorsCard/ErrorDetails/index.js`):

| Row          | Content                                                                                                                                                                          | Source                                                                          |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Source       | `source.label` (`Collection` / `Folder` / `Folder: {name}` / `Request`) uppercase, `font.size.xs`, 600, `letter-spacing .05em`, muted                                            | `:17-24`, `utils/response-pane-errors/index.js:21-40`, `components/ResponsePane/StyledWrapper.js:17-23` |
| Location     | `filePath:line`, mono 11 px, muted; clickable (`navigable`) when a target tab exists, `title="Open {location}"`, trailing `IconExternalLink` 12                                  | `:20-30`, `components/ResponsePane/StyledWrapper.js:25-46`                                              |
| Code         | `CodeSnippet variant="error"` — mono `font.size.xs`, line-height 1.4, `radius.base`, highlight row `danger @10%` with 3 px left border; gutter `min-width 2.5rem`, right-aligned | `:32`, `CodeSnippet/StyledWrapper.js:5-47`                                      |
| Message      | mono `font.size.sm`, 500, `pre-wrap`, `break-word`, `colors.text.danger`                                                                                                         | `:33-35`, `components/ResponsePane/StyledWrapper.js:48-55`                                              |
| Stack toggle | `Show stack trace` / `Hide stack trace` (verbatim `{Show\|Hide} {detailsHeading.toLowerCase()}`, heading `Stack trace`) + chevron 14                                             | `:36-47`, `utils/response-pane-errors/index.js:52-54`                           |
| Stack        | `<pre>` mono `font.size.xs`, line-height 1.4, `break-all`, muted                                                                                                                 | `components/ResponsePane/StyledWrapper.js:77-84`                                                        |

Card chrome (`ResponseErrorsCard/StyledWrapper.js`): surface `background.base`,
`1px solid border.subtle`, `radius.base`; row hover =
`mix(0.04, text, background.base)`; expanded row =
`mix(0.06, text, background.base)` (`CollapsibleError/StyledWrapper.js:16-23`).
Focus ring on rows: `1px solid input.focusBorder`, `outline-offset:-1px`
(`:35-38`).

Behaviour (`hooks/useResponsePaneErrors/index.js`):

- Persisted closed state per item: `localStorage` key
  `response-pane-errors-closed-{uid}` (`:8`).
- Closing hides the card and shows `ResponseErrorsIcon` in the seam
  (`components/ResponsePane/index.js:239-241`); clicking the icon calls
  `reopenCard()` which selects the Response tab and clears closed (`:24-27`).
- Closed/collapsed flags only apply to the current send
  (`sendId = requestUid ?? requestSent.timestamp`) so a re-send re-opens
  (`:12-17`).
- Full-pane mode hides `div.response-tab-content`
  (`components/ResponsePane/index.js:307-310`).
- Leaving the Response tab counts as closed (`:16`).

`ResponseErrorsIcon` (`components/ResponsePane/ResponseErrorsIcon`):
`IconAlertCircle` 16/1.5 + `<sup>` count, `height 1.5rem`, `padding 0 .25rem`,
`gap .125rem`, `radius.sm`, colour `colors.text.danger`, font `font.size.sm`;
tooltip `Show {n} error(s)` via `ToolHint place="bottom"`
(`components/ResponsePane/index.js:12-24`, `components/ResponsePane/StyledWrapper.js:3-12`).

### 4.10 Alert stacking

`ResponseAlertsSection` is a flex column with `gap .5rem`, `margin-bottom
.5rem`; `full-pane` grows and drops the margin
(`components/ResponsePane/ResponseAlertsSection/StyledWrapper.js:3-15`). Order: errors card, then
unresolved-variables info.

---

## 5. State matrix

| #   | `requestState` / data                    | Pane renders                         | Testid present                       |
| --- | ---------------------------------------- | ------------------------------------ | ------------------------------------ |
| 1   | idle, no response, no timeline           | `Placeholder`                        | `response-pane-shortcut-placeholder` |
| 2   | `queued`/`sending`, no previous response | `Overlay` only                       | —                                    |
| 3   | `queued`/`sending`, previous response    | tabs + overlay over previous content | `response-pane-status`               |
| 4   | responded, body small                    | tabs + body viewer                   | `response-tab-content`               |
| 5   | responded, body > 10 MB                  | tabs + `LargeResponseWarning`        | `large-response-view`                |
| 6   | responded with `error` string            | tabs, error text (no actions)        | —                                    |
| 7   | responded, script error(s)               | alerts card over body                | `response-errors-card`               |
| 8   | responded, unresolved vars               | alert below card/body header         | `unresolved-variables-info`          |
| 9   | `status: 'skipped'`                      | `SkippedRequest`                     | —                                    |
| 10  | cancelled                                | falls through to 1 or 4              | —                                    |
| 11  | send rejected                            | `NetworkError` toast (5 s)           | —                                    |
| 12  | TLS self-signed error                    | error text + preferences hint        | —                                    |

---

## 6. Interactions & shortcuts

| Action                         | Trigger                                               | Effect                                                                                                                     | Source                                                                                           |
| ------------------------------ | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Switch pane tab                | click tab / overflow menu item                        | `updateResponsePaneTab`                                                                                                    | `components/ResponsePane/index.js:91-98`                                                         |
| Change layout                  | `ResponseLayoutToggle` click, or `Ctrl`+`J` / `⌘`+`J` | flips `layout.responsePaneOrientation`, persisted via `savePreferences`                                                    | `components/ResponsePane/ResponseLayoutToggle/index.js:54-72`, `providers/Hotkeys/index.js:386-400`, `keyMappings.js:43` |
| Copy                           | click/Enter/Space on Copy                             | clipboard + toast                                                                                                          | `components/ResponsePane/ResponseCopy/index.js:35-98`                                                                    |
| Save as example                | click Bookmark                                        | opens modal                                                                                                                | `components/ResponsePane/ResponseBookmark/index.js:49-71`                                                                |
| Download                       | click Download                                        | IPC `renderer:save-response-to-file`                                                                                       | `components/ResponsePane/ResponseDownload/index.js:20-38`                                                                |
| Clear                          | click Clear                                           | `responseCleared` (response → null)                                                                                        | `components/ResponsePane/ResponseClear/index.js:9-23`                                                                    |
| Clear timeline                 | `Clear Timeline` text button                          | `clearRequestTimeline`                                                                                                     | `components/ResponsePane/ClearTimeline/index.js:9-21`                                                                    |
| Cancel request                 | `Cancel Request` in overlay                           | `cancelRequest`                                                                                                            | `components/ResponsePane/Overlay/index.js:14-16`                                                                         |
| Reveal large response          | `View`                                                | permits formatting; warning removed                                                                                        | `components/ResponsePane/QueryResult/index.js:197-202`                                                                   |
| Close error card               | `Close`                                               | persisted closed + new-send reset                                                                                          | `useResponsePaneErrors/index.js:19-27`                                                           |
| Expand error card              | `Expand to full pane` / `Restore height`              | hides tab content                                                                                                          | `:29`                                                                                            |
| Expand all / collapse all      | card action (multi-error only)                        | all rows                                                                                                                   | `components/ResponsePane/ResponseErrorsCard/index.js:28`, `:56-58`                                                       |
| Toggle stack trace             | `Show stack trace` / `Hide stack trace`               | pre block                                                                                                                  | `ErrorDetails/index.js:38-50`                                                                    |
| Jump to error source           | click file path                                       | opens the collection/folder/request settings tab with the script pane focused (`getErrorSourceTabUid` resolves the target) | `components/ResponsePane/ResponseErrorsCard/ErrorDetails/index.js:20-24`, `utils/response-pane-errors/index.js:6-14`     |
| Dismiss unresolved info        | `×`                                                   | persisted per item                                                                                                         | `components/ResponsePane/UnresolvedVariablesInfo/index.js:82-84`                                                         |
| Context menu "set as variable" | right-click on the tab content                        | `setAsVariable.openFromDomSelection`                                                                                       | `components/ResponsePane/index.js:310`                                                           |

Keyboard: only `Ctrl`/`⌘`+`J` is wired directly. `Alt`+`Shift`+`F` (Prettify)
exists globally (`keyMappings.js:44`) but the response editor is read-only.

Persistence keys (Go port should keep equivalent state):

| Key                                                                                              | What                         | Source                                                                          |
| ------------------------------------------------------------------------------------------------ | ---------------------------- | ------------------------------------------------------------------------------- |
| `tabs[uid].responsePaneTab`                                                                      | active pane tab              | `components/ResponsePane/index.js:100`                                          |
| `tabs[uid].responseFormat` / `.responseViewTab`                                                  | body format + editor/preview | `:54-59`                                                                        |
| `tabs[uid].responseFilter` / `.responseFilterExpanded`                                           | JSONPath/XPath filter        | `:172-175`                                                                      |
| `response-body-scroll-{uid}`                                                                     | body scroll                  | `QueryResultPreview/index.js:37`                                                |
| `response-headers-scroll-{uid}`, `response-tests-scroll-{uid}`, `response-timeline-scroll-{uid}` | panel scroll                 | `components/ResponsePane/ResponseHeaders/index.js:9`, `components/ResponsePane/TestResults/index.js:95`, `components/ResponsePane/Timeline/index.js:52` |

---

## 7. gRPC and WebSocket (deferred)

`GrpcResponsePane` and `WSResponsePane` share this shell (tabs + status seam

- actions) but use their own tabs — gRPC: `Response`, `Metadata`, `Trailers`,
  `Timeline`, test label (`components/ResponsePane/GrpcResponsePane/index.js:65-90`); WS: `Messages`,
  … (`components/ResponsePane/WsResponsePane/index.js:44-50`) — and are not detailed here.

---

## Open Questions

1. **Unknown status code text.** `StatusCode` renders
   `{status} {statusText || phrase}` (`components/ResponsePane/StatusCode/index.js:20`). For a code
   absent from the phrase map and without `statusText`, React prints
   nothing for the expression but the literal survives as
   `"200 undefined"`. Unverified what React actually emits; a Go port should
   print just the number.
2. **1xx rendering.** 1xx and 2xx both map to `text-ok`
   (`components/ResponsePane/StatusCode/index.js:10-11`); 3xx is `text-error` (`:12`). The Timeline
   status badge instead colours 3xx with `colors.text.warning` and 1xx as
   muted (`components/ResponsePane/Timeline/TimelineItem/Common/Status/index.js:9-16`). Pick one;
   this file follows `StatusCode`.
3. **`NetworkError` surface colour.** The class list sets both `bg-white` and
   `bg-red-100` (`components/ResponsePane/NetworkError/index.js:5`); the rendered colour depends on
   Tailwind's cascade order, not read from a built stylesheet here.
4. **`5 MB` vs `5*1024*1024`.** The bookmark tooltip says "5MB"
   (`components/ResponsePane/ResponseBookmark/index.js:23`), the check is 5 MiB (`:35`).
5. **`width: calc(100% - 0.75rem)` on Overlay** (`components/ResponsePane/Overlay/StyledWrapper.js:6`)
   has no cited rationale; with `position:absolute` inside a
   `padding:0 1rem` container it is 12 px narrower than the container. A Go
   port should simply fill the content box.
6. **Size mismatch between panes.** `ResponsePane` decodes the base64 buffer
   for `responseSize` (`components/ResponsePane/index.js:105-119`) while
   `QueryResult` falls back to `floor(dataBuffer.length * 0.75)`
   (`components/ResponsePane/QueryResult/index.js:112-123`). The same response can therefore show two
   different sizes if `response.size` is absent.
7. **Streaming + errors.** While streaming, `rightContent` omits the actions
   block entirely (`components/ResponsePane/index.js:243`) but `ResponseAlertsSection` still renders
   when `!isLoading` (`:235`); unverified whether a stream can have script
   errors before ending.
