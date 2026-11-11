# Screen — Request pane (container)

The request pane is the left (or top) half of an open request tab. It owns the URL
bar and the sub-tab strip; each sub-tab's body is specified in a sibling file
(`request-params.md`, `request-body.md`, `request-auth.md`,
`request-subtabs.md`).

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source. Item 4 of **Open Questions** records the one
dimension I could not read from source.

---

## 1. Component layering

```text
RequestTabPanel/                     components/RequestTabPanel/index.js:70
├─ .query-url-wrapper                components/RequestTabPanel/index.js:708   (pt-3 pb-4 px-4)
│   └─ QueryUrl  (HTTP/GraphQL)      components/RequestTabPanel/index.js:640-648  → RequestPane/QueryUrl/index.js:31
│      ├─ HttpMethodSelector          components/RequestPane/QueryUrl/index.js:407
│      ├─ url-input-group
│      │   ├─ SingleLineEditor        components/RequestPane/QueryUrl/index.js:414
│      │   └─ #request-actions        components/RequestPane/QueryUrl/index.js:428
│      │       ├─ IconCode (Generate Code)   :437
│      │       └─ IconDeviceFloppy (Save)    :450
│      └− SendButton                  components/RequestPane/QueryUrl/index.js:461
├─ .main  (flex row | flex-col)      components/RequestTabPanel/index.js:711
│   ├─ .request-pane                 components/RequestTabPanel/index.js:721   width from getRequestPaneStyle()
│   │   └─ px-4 → HttpRequestPane    components/RequestTabPanel/index.js:722-724
│   ├─ .dragbar-wrapper               components/RequestTabPanel/index.js:729-735  (1px line inside 12px hit area)
│   └─ .response-pane                components/RequestTabPanel/index.js:746     (out of scope: response-pane.md)
└─ CollapsedPanelIndicator           components/RequestTabPanel/index.js:712-744 (when a pane is collapsed)

HttpRequestPane                       RequestPane/HttpRequestPane/index.js:53
├─ ResponsiveTabs                     components/RequestTabPanel/index.js:25-37 TAB_CONFIG, :170
│   ├─ visible tabs (div role=tab)    ui/ResponsiveTabs/index.js:200-216
│   ├─ overflow "»" MenuDropdown      ui/ResponsiveTabs/index.js:263-274
│   └─ rightContent (per-tab extras)  components/RequestTabPanel/index.js:138-166
└─ <section class="flex w-full flex-1 mt-4">
    └─ HeightBoundContainer
        └─ TAB_PANELS[tab]            components/RequestTabPanel/index.js:39-51
```

GraphQL requests use `GraphQLRequestPane` (own tab set: Query, Headers, Auth,
Vars, Script, Assert, Tests, Docs, Settings — `RequestPane/GraphQLRequestPane/index.js:39-49`).
gRPC/WebSocket panes exist (`components/RequestPane/GrpcRequestPane/index.js:103-128`,
`components/RequestPane/WSRequestPane/index.js:126-146`) but are out of scope.

---

## 2. Dimensions

| Element | Value | Source |
| --- | --- | --- |
| Request pane min width (horizontal layout) | `350px` | `RequestTabPanel/index.js:55`, applied `:698` |
| Response pane min width | `490px` | `RequestTabPanel/index.js:56` |
| Request pane min height (vertical layout) | `150px` | `RequestTabPanel/index.js:57`, applied `:693-694` |
| Response pane min height (vertical layout) | `150px` | `RequestTabPanel/index.js:58` |
| Default request pane height (vertical) | `380px` | `hooks/useTabPaneBoundaries/index.js:13` |
| Default request pane width | `(screenWidth − sidebarWidth) / 2.2` | `hooks/useTabPaneBoundaries/index.js:16,22` |
| Dragbar hit area, horizontal layout | width `12px`, line `1px` solid | `RequestTabPanel/StyledWrapper.js:43-54` |
| Dragbar hit area, vertical layout | height `12px`, line `1px` solid, `padding: 0 1rem` | `RequestTabPanel/StyledWrapper.js:81-92` |
| Dragbar line colour / hover | `requestTabPanel.dragbar.border` / `.activeBorder` | `components/RequestTabPanel/StyledWrapper.js:54,59,92,98` |
| `.main` bottom padding | `1rem` (`padding-bottom: 1rem`) | `RequestTabPanel/StyledWrapper.js:20-22` |
| URL bar wrapper padding | `pt-3` (12px), `pb-4` (16px), `px-4` (16px) | `RequestTabPanel/index.js:708` |
| **URL bar height** | `2.1rem` (33.6px at 16px root) | `RequestPane/QueryUrl/StyledWrapper.js:4` |
| URL input container radius | `border.radius.base` (6px) | `components/RequestPane/QueryUrl/StyledWrapper.js:8` |
| URL container border | `requestTabPanel.url.border` → `solid 1px #e5e5e5` L / `solid 1px #333333` D | `components/RequestPane/QueryUrl/StyledWrapper.js:7`, `_tools/tokens.json` |
| URL container background | `requestTabPanel.url.bg` → `#ffffff` L / `#1a1a1a` D | `HttpMethodSelector/StyledWrapper.js:44`, tokens |
| URL action icon size | `20px`, `strokeWidth={1.5}`, gap `gap-3` (12px), `mx-2` (8px) | `components/RequestPane/QueryUrl/index.js:428,437,453` |
| SendButton min width | `min-width: 4.1rem` (65.6px), `margin-left: 0.5rem` | `components/RequestPane/SendButton/StyledWrapper.js:6`, `components/RequestTabPanel/index.js:7` |
| Method span max width | `15ch`, truncated with ellipsis | `HttpMethodSelector/StyledWrapper.js:57` |
| Method span font | `font.size.sm` (12px), weight 500, line-height 1.5 | `HttpMethodSelector/StyledWrapper.js:61-63` |
| Method selector inner margin | `2px`, radius `border.radius.sm` (4px) | `HttpMethodSelector/StyledWrapper.js:20-21` |
| Sub-tab strip → body gap | `mt-4` (16px) | `RequestPane/HttpRequestPane/index.js:179` |
| Tab item | `padding: 6px 0`, `border-bottom: 2px solid`, `margin-right: 1.2rem`, font `font.size.sm` | `ui/ResponsiveTabs/StyledWrapper.js:27-36` |
| Tab active underline | `tabs.active.border` → `#d5882a` L / `#e7b85f` D | `ResponsiveTabs/StyledWrapper.js:54`, tokens |
| Overflow chevron icon | `18px`, `strokeWidth={2}`; reserved width `60px` in the fit calculation | `ui/ResponsiveTabs/index.js:7,271` |

Colour tokens used: `text`/`text.base` `#343434` L / `#cccccc` D (body text),
`colors.text.subtext0` `#9b9b9b` L / `#999` D (inactive tab),
`tabs.active.color` `#343434` L / `#cccccc` D,
`draftColor` `#cc7b1b` both themes (unsaved save icon),
`textLink` `#346ab2` L / `#8bc2f9` D ("+ Add Custom" in the method menu),
`primary.solid` / `button2.color.primary.bg` `#d37f17` L / `#e4ae49` D (Send),
`button2.color.primary.text` `#ffffff` L / `#000000` D.

---

## 3. Sub-tab bar — order, labels, indicators

Order and verbatim labels, `RequestPane/HttpRequestPane/index.js:25-37`:

| # | key | Label | Panel component | Right-side content | Indicator |
| --- | --- | --- | --- | --- | --- |
| 1 | `params` | `Params` | `QueryParams` | — | `<sup>` count of enabled params (`:108`) |
| 2 | `body` | `Body` | `RequestBody` | `RequestBodyMode` selector (`:140-146`) | `StatusDot` when `body.mode !== 'none'` (`:109`) |
| 3 | `headers` | `Headers` | `RequestHeaders` | — | `<sup>` count of enabled headers (`:110`) |
| 4 | `auth` | `Auth` | `Auth` | — | `StatusDot` when effective auth exists (`:111`) |
| 5 | `vars` | `Vars` | `Vars` | — | `<sup>` count of enabled pre+post vars (`:87,112`) |
| 6 | `script` | `Script` | `Script` | `TabBarAiAssist` (`:155-163`) | `StatusDot` (error variant on script error) (`:113`) |
| 7 | `assert` | `Assert` | `Assertions` | — | `<sup>` count of enabled assertions (`:114`) |
| 8 | `tests` | `Tests` | `Tests` | `TabBarAiAssist` (`:155-163`) | `StatusDot` (`:115`) |
| 9 | `docs` | `Docs` | `Documentation` | `DocsAction` + `TabBarAiAssist` (`:147-154`) | `StatusDot` when docs non-empty (`:116`) |
| 10 | `app` | `App` | `AppCodeEditor` | `TabBarAiAssist` | `StatusDot` when app code non-empty (`:117`) |
| 11 | `settings` | `Settings` | `Settings` | — | `StatusDot` when tags exist (`:118`) |

The **App** tab (key `app`) is only rendered when `item.app.enabled === true`
(`components/RequestPane/HttpRequestPane/index.js:79,124`). A restored `app` tab while apps are
disabled falls back to `params` (`:81`).

**Default tab.** `getDefaultRequestPaneTab` (`utils/collections/index.js:1217-1230`):

- If request has **no enabled params** and `body.mode` is set and not `'none'` →
  `'body'`. Otherwise → `'params'`.
- gRPC/WS requests → `'body'`; GraphQL → `'query'`
  (`providers/ReduxStore/slices/tabs.js:139-145`).

**Visual rules** (`ui/ResponsiveTabs/StyledWrapper.js`): inactive tab colour
`colors.text.subtext0`, hover raises colour to `tabs.active.color`
(`:47-49`), active tab sets `font-weight` = `tabs.active.fontWeight` = **400**
(`:52`, token) and a 2px bottom border in `tabs.active.border`. Count
indicators are rendered as plain `<sup>` elements, not pills — `sup { line-height: 1; margin-left: 0 }`
(`:75-81`). (A `.tab-count` pill style exists at `:61-73` but is not used by
this tab strip.)

**Overflow.** Tabs are measured off-screen (`ui/ResponsiveTabs/index.js:246-256`).
A tab moves to the overflow dropdown when `currentWidth + tabWidth > availableWidth`
where `availableWidth = containerWidth − rightContentWidth − 60` and
`tabWidth = measuredWidth + 20` (`:7-10,52-68`). If the active tab would be
hidden it is swapped back in and the last visible tab moves to overflow
(`:71-82`). The overflow trigger is `IconChevronsRight` inside a
`MenuDropdown` (placement `bottom-start`) (`:263-274`); the active item is
highlighted via `selectedItemId` (`:268`). Re-measure runs 20 ms after a tab
change (150 ms for `delayedTabs`, which is `['body']` — `HttpRequestPane:176`).

**Empty/error states** (`HttpRequestPane/index.js`): missing tab context →
`An error occurred!` (`:135`); unknown key → `404 | Not found` (`:131`).

---

## 4. URL bar (`QueryUrl`)

### 4.1 Layout

```text
[ METHOD ▾ ] │ [ ________________________________________ Enter URL or paste a cURL request  </>  💾 ] [ Send ]
  1.2rem-ish     ← SingleLineEditor (grows) →        mx-2 gap-3   20px icons     ml-2  4.1rem min
```

### 4.2 Method selector (`HttpMethodSelector`)

| Aspect | Value | Source |
| --- | --- | --- |
| Standard methods, in menu order | `GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD, TRACE, CONNECT` | `HttpMethodSelector/index.js:7` |
| Extra menu item (last) | `+ Add Custom` — styled `font-normal mt-1 text-link` | `components/RequestTabPanel/index.js:120-125` |
| Dropdown placement | `bottom-start` | `components/RequestTabPanel/index.js:180` |
| Trigger | `<button>` with `<span>` method text + optional caret | `components/RequestTabPanel/index.js:13-31` |
| Caret | `IconCaretDown` 14px; hidden when `showCaret` is false (the request pane passes no `showCaret`) | `components/RequestTabPanel/index.js:28,33` |
| Method colour | `theme.request.methods[method.toLowerCase()]` (design-system §1.5) | `components/RequestTabPanel/index.js:41-47` |
| Hover / open background | `color-mix(in srgb, currentColor 15%, transparent)` | `components/RequestTabPanel/StyledWrapper.js:23-26` |
| Custom-mode input | uppercase on each keystroke, width `calc(Nch + 1rem)`, max `calc(16ch + 1rem)`, font 12px | `components/RequestTabPanel/index.js:51-54,142-161` |
| Unknown custom method | keeps the typed verb; empty falls back to `GET` | `components/RequestTabPanel/index.js:63-74,99-104` |
| Blur keeps last value | `selectedMethodRef` | `components/RequestTabPanel/index.js:63-74` |

### 4.3 URL editor

- `SingleLineEditor` with placeholder **"Enter URL or paste a cURL request"**
  (`components/RequestPane/QueryUrl/index.js:417`), `highlightPathParams`, `showNewlineArrow`
  (`:424-426`).
- Leading/trailing whitespace is trimmed on change; cursor restored when
  trimmed (`:59-76`).
- `Enter` sends, `Alt-Enter` sends, `Cmd/Ctrl-F` is suppressed, `Tab`/`Shift-Tab`
  disabled (`components/SingleLineEditor/index.js:74-89`).
- Pasting text matching `/^\s*curl\s/i` triggers cURL import (HTTP:
  `components/RequestPane/QueryUrl/index.js:178-397`; GraphQL request type: `:98-176`).
- Transient new requests auto-focus the editor if the URL is empty
  (`:44-48`).

### 4.4 Right-side icons

| Icon | Tooltip text (verbatim) | Behaviour | Source |
| --- | --- | --- | --- |
| `IconCode` | `Generate Code` | Opens `GenerateCodeItem` modal; toast `URL is required` if no URL | `components/RequestPane/QueryUrl/index.js:429-439,89-96` |
| `IconDeviceFloppy` | `Save (Cmd + S)` on macOS, `Save (Ctrl + S)` elsewhere | Saves only when `hasRequestChanges(item)`; icon colour `draftColor` when dirty, `requestTabs.icon.color` when clean; cursor `pointer`/`default` | `components/RequestPane/QueryUrl/index.js:37,42,440-457` |

Icon colour when clean is `requestTabs.icon.color` = `#9b9b9b` L / `#999` D
(design-system §6.3).

### 4.5 Send button

| State | Label | Variant / colour | Source |
| --- | --- | --- | --- |
| Idle | `Send` | `filled`, `color="primary"`, size `sm` | `components/RequestPane/SendButton/index.js:8-18` |
| Sending (`requestState` ∈ `queued`, `sending`) or streaming response running | `Cancel` | `outline`, `color="primary"`, size `sm` | `components/RequestPane/QueryUrl/index.js:39,462`, `SendButton:11,15,17` |
| `data-action` attribute | `send` / `cancel` | — | `components/RequestPane/SendButton/index.js:14` |

`Button` size `sm`: `padding: 0.375rem 0.75rem`, `font-size: font.size.sm`
(12px), `gap: 0.375rem`, radius `border.radius.sm` (4px)
(`ui/Button/StyledWrapper.js:31-43,91-93`). Send/Cancel fills the whole
`2.1rem` bar height (`components/RequestPane/SendButton/StyledWrapper.js:14-17`).

Cancel dispatches `cancelRequest` with `preventDefault` + `stopPropagation`
(`components/RequestPane/QueryUrl/index.js:398-402`).

### 4.6 Container states

| State | Appearance | Source |
| --- | --- | --- |
| Default | bordered box, `input`-like radius 6px | `QueryUrl/StyledWrapper.js` |
| Focus | **no** custom focus ring — border stays `requestTabPanel.url.border` | `components/RequestTabPanel/StyledWrapper.js:6-11` (only border is declared) |
| Dragging panes | whole tab panel gets `cursor: col-resize` / `row-resize`, plus a fixed full-screen overlay at `z-index: 9999` | `RequestTabPanel/StyledWrapper.js:4-10`, `components/RequestTabPanel/index.js:761-771` |

---

## 5. Interactions

| Interaction | Result | Source |
| --- | --- | --- |
| Click a tab | `updateRequestPaneTab({ uid, requestPaneTab })` | `components/RequestPane/HttpRequestPane/index.js:90-95` |
| Click `»` overflow | `MenuDropdown` listing overflow tabs; selecting hides the menu | `ui/ResponsiveTabs/index.js:37-43,227-240` |
| Click Send | `sendRequest(item, collection.uid)`; on rejection shows `NetworkError` toast for 5000 ms | `RequestTabPanel/index.js:595-618` |
| Send while already sending/queued | no-op | `RequestTabPanel/index.js:612` |
| Drag dragbar | resizes request pane; release clamps to `MIN_*_PANE_*` | `RequestTabPanel/index.js:183-249` |
| Collapse request/response | `CollapsedPanelIndicator` strip; drag past `MIN_LEFT_PANE_WIDTH / 2` (or height/2) re-expands | `RequestTabPanel/index.js:712-744` |
| Request changed | Save icon turns `draftColor`; clicking saves | `components/RequestPane/QueryUrl/index.js:442-456` |

---

## 6. Keyboard shortcuts

Source of truth: `providers/Hotkeys/keyMappings.js` (defaults, user-overridable
via `app.preferences.keyBindings`; disabled when
`preferences.keybindingsEnabled === false` — `hooks/useKeybinding/index.js:24-27`).

| Action | macOS | Windows/Linux | Name | Source |
| --- | --- | --- | --- | --- |
| `sendRequest` | `Cmd + Enter` (`command+bind+enter`) | `Ctrl + Enter` | Send Request | `keyMappings.js:42`; bound in `RequestTabPanel/index.js:88-93` |
| `save` | `Cmd + S` | `Ctrl + S` | Save | `keyMappings.js:9`; also the URL editor's `onSave` tooltip `QueryUrl:37` |
| `prettify` | `Alt + Shift + F` | `Alt + Shift + F` | Prettify | `keyMappings.js:44`; JSON/XML body only, and only when focus is inside the body editor (`RequestPane/RequestBody/RequestBodyMode/index.js:96-100`) |
| `changeLayout` | `Cmd + J` | `Ctrl + J` | Change Orientation | `keyMappings.js:43` |
| `closeTab` | `Cmd + W` | `Ctrl + W` | Close Tab | `keyMappings.js:7` |
| `newRequest` | `Cmd + N` | `Ctrl + N` | New Request | `keyMappings.js:58` |
| `triggerAutocomplete` | `Alt + Space` | `Ctrl + Space` | Trigger Autocomplete | `keyMappings.js:50` |

Editor-local keys inside the pane: `Enter` = send the request (single-line
editors), `Tab` = two spaces / indent (block editors), `Shift-Tab` =
indent less, `Ctrl/Space` = show completion hints
(`components/SingleLineEditor/index.js:74-89`,
`components/CodeEditor/index.js:198-218`).

Display formatting (for a Go port's tooltips): macOS uses `⌘ ⌃ ⌥ ⇧` and `↩`
for Enter; Windows `Ctrl Alt Shift Win`; Linux shows `Super` in place of `Win`
(`keyMappings.js:100-166`).

---

## 7. Verbatim copy index

| String | Where |
| --- | --- |
| `Params` `Body` `Headers` `Auth` `Vars` `Script` `Assert` `Tests` `Docs` `App` `Settings` | `components/RequestPane/HttpRequestPane/index.js:26-36` |
| `Enter URL or paste a cURL request` | `components/RequestPane/QueryUrl/index.js:417` |
| `+ Add Custom` | `HttpMethodSelector/index.js:122` |
| `Generate Code` | `components/RequestPane/QueryUrl/index.js:429` |
| `Save (Cmd + S)` / `Save (Ctrl + S)` | `components/RequestPane/QueryUrl/index.js:37,440` |
| `Send` / `Cancel` | `components/RequestPane/SendButton/index.js:17` |
| `URL is required` | `components/RequestPane/QueryUrl/index.js:94` |
| `Invalid cURL command` | `components/RequestPane/QueryUrl/index.js:115,202` |
| `cURL command imported successfully` | `components/RequestPane/QueryUrl/index.js:390` |
| `GraphQL query imported successfully` | `components/RequestPane/QueryUrl/index.js:170` |
| `Failed to parse cURL command` / `Failed to parse GraphQL query` | `components/RequestPane/QueryUrl/index.js:393,174` |
| `An error occurred!` | `components/RequestPane/HttpRequestPane/index.js:135` |
| `404 \| Not found` | `components/RequestPane/HttpRequestPane/index.js:131` |
| `Please enter a valid gRPC server URL` / `Please select a gRPC method` / `Please enter a valid WebSocket URL` | `RequestTabPanel/index.js:599,604,609` |

i18n: `src/i18n/translation/en.json` contains only app-wide strings
(`COMMON`, `WELCOME`, `SUPPORT` — `i18n/translation/en.json:1-40`). All
request-pane labels listed above are hard-coded English in components.

---

## 8. Screenshots

- `refs/api_screen_bruno.png` — full request pane: method `GET`, URL with
  `{{baseUrl}}` variable highlighting, the two right-side icons, `Send`,
  sub-tab strip `Params ² | Body | Headers | Auth | Vars | Script | Assert …` with
  the `»` overflow, `Query` table, `Bulk Edit` link and collapsed Path section.
- `refs/main_section_bruno.png` — collection-level Headers tab for comparison of
  the same tab-strip styling.

---

## 9. Out of scope here

Mentioned only: AI assist (`TabBarAiAssist`), App tab / App preview
(`AppCodeEditor`), OpenAPI sync/spec tabs, mock-server tabs, gRPC and
WebSocket request panes (`RequestTabPanel/index.js:564-578`,
`components/RequestPane/GrpcRequestPane`, `WSRequestPane`).

---

## 10. Open Questions

1. **RESOLVED — `changeLayout` is bound globally.** Not in `RequestTabPanel`;
   it lives in the hotkeys provider: `bindAction('changeLayout', ...)` at
   `providers/Hotkeys/index.js:386`, mapping `Cmd/Ctrl J`
   (`keyMappings.js:43`), toggling `responsePaneOrientation`
   horizontal<->vertical. Implement it as a global shortcut.
2. **RESOLVED — `showCaret` default.** Confirmed: every call site omits it, so the URL bar shows no caret by design. `HttpMethodSelector` defaults `showCaret = false`
   (`components/RequestTabPanel/index.js:33`) and `QueryUrl` does not pass it, so no caret should render
   in the request URL bar — the screenshot shows only the verb. Confirm this is
   intended (the caret is visible in other method selectors such as the new-request
   dialog).
3. **Sub-tab overflow width reservation** is a fixed `60px` (`ui/ResponsiveTabs/index.js:7`)
   even though the chevron trigger is `18px` wide plus padding; the constant also
   has `GAP_BETWEEN_LEFT_AND_RIGHT_CONTENT = 80` used only by the expandable-right-content
   path (`:10`). Reproduce the arithmetic as-is rather than "improving" it.
4. **Send button border radius.** `Button` `size="sm"` maps to radius
   `border.radius.sm` (4px) and the container is 6px (`components/RequestPane/QueryUrl/StyledWrapper.js:8`),
   so the Send button sits inside a differently-rounded bar. The screenshot is
   consistent with a ~4px button radius, but I could not measure it exactly.
5. **`/` in the URL editor** may be highlighted as a path-param boundary via
   `highlightPathParams`; the exact overlay styling lives in the CodeMirror
   mode (`utils/common/codemirror`) and is not specified here.
