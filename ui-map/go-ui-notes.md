# Go UI notes — build order, native concerns, open questions

How to rebuild this map in Go with MyGo + Lobe MyGo, and everything that needs a
non-obvious approach because the source is a browser app.

---

## 1. Build order

Build bottom-up; each stage is testable on its own (goldens or unit assertions).

1. **Tokens** — port `design-system.md` §1–7 into `internal/ui/theme` token
   family. Already started: `internal/ui/theme/graphite.go` (D-017 Graphite
   Console) and the `theme.Adapter`. Decide whether to keep Graphite or render
   Bruno's palettes exactly; if exact parity is the goal, `light`/`dark` are the
   two token sets to port first, presets as data (§9).
2. **Primitives** (`components/*`): Button, text field shell, Select/Dropdown,
   Toggle, Checkbox, RadioButton, Tabs/Segmented, Modal, Toast, Tooltip,
   StatusBadge/CountBadge, StatusDot, Spinner, ListGroup, ContextMenu.
   - **Important:** Lobe's `field.Input` and `action.Button` paint a background
     on a container node they do not return on this MyGo version, so controls
     render with MyGo's near-white default surface (reproducible with Lobe's own
     `theme.Apply`; pre-existing, not caused by the theme port). Wrap them on an
     app-owned, token-painted surface — see `internal/ui/widgets/controls.go`
     `FieldInput` / `Button`. Route every call site through the wrapper.
3. **Composites** (`widgets.md`): key/value table, code editor (single/multi),
   Markdown viewer + rich-text editor, tree, splitter.
4. **Shell**: `AppTitleBar`, `StatusBar`, body frame, sidebar frame
   (`screens/app-shell.md`, `screens/sidebar.md`).
5. **Screens**, in the request loop order: request tabs → request pane (+
   sub-tabs) → response pane → environments → settings/dialogs.

The theme adapter's `Apply` must install a theme that Lobe's `TokensOf` can
resolve (via `theme.NewConfig`), or nested widgets fall back to Lobe defaults.

---

## 2. Browser-only behaviour → native approach

| Source behaviour                         | Go approach                                                                                                                                         |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| CSS layout (flexbox/grid, gap, position) | MyGo layout: `Row`/`Column`/`Box`, `Gap`, `Grow`, `Fill`, explicit widths; no CSS. Re-implement measured sizes from each screen's dimension tables. |
| `-webkit-app-region: drag` title bar     | MyGo custom title bar with `DragWindow()`; honour per-OS window-control insets (macOS 70 px left, 0 in fullscreen).                                 |
| CodeMirror                               | Custom editor widget (see `widgets.md`): syntax highlight, gutter, variable spans. No CodeMirror in Go.                                             |
| `contenteditable` / TipTap rich text     | Native rich-text editor or markdown-with-preview; must keep bold/italic/headings/lists/links/code.                                                  |
| tippy.js popovers (dropdowns, tooltips)  | MyGo popover/overlay layer (`Portal` equivalent) with viewport clamping.                                                                            |
| react-hot-toast                          | `feedback.ToastHost` (already in repo).                                                                                                             |
| HTML `<select>` / native scroll          | Lobe `field.Select` and MyGo scroll containers; custom scrollbar (`scrollbar.color`, width ~6–8).                                                   |
| Copy to clipboard, open external URL     | Platform layer (`internal/platform`), not IPC.                                                                                                      |
| Drag-and-drop (rows, sidebar, files)     | MyGo drag events; drop indicator using `dragAndDrop.border` `2px solid` + `hoverBg`.                                                                |
| Responsive overflow (tab strip chevron)  | MyGo measure/overflow; keep the reserved chevron width to keep layout stable.                                                                       |
| Resizable panes                          | `layout.RootView` splitters already exist; keep 12 px drag bars and min sizes from the screen files.                                                |

Two components exist only to keep panes alive across React re-renders
(`AppPreviewKeepAlive`, `HeightBoundContainer`, `Portal`, `ErrorCapture`/
boundary). In Go they are not needed as components — MyGo keeps widgets alive
between frames. `Portal` maps to a top-layer overlay host; `ErrorCapture` maps to
recover in the frame loop that shows a toast instead of a blank window.

---

## 3. Scope notes (deferred from v1)

Per `docs/research/unresolved-questions.md` §UQ-D6 and the roadmap, these
screens exist in Bruno but are **not** in the first Go build; their map entries
are intentionally one-line:

- AI assist: `AIAssist`, `AiChatSidebar`, `SettingsAiAssist`.
- OpenAPI sync: `ApiSpecPanel`, `ApiSpecTab`, `OpenAPISpecTab`, `OpenAPISyncTab`,
  `Sidebar/ApiSpecs/*`.
- Mock server: `MockServer`, `Sidebar/Sections/MockServersSection`.
- Git pane: `Git` (source-control UI).
- gRPC and WebSocket request/response panes (HTTP is the v1 request loop).
- `Devtools`, `AppWebviewPane` developer surfaces.

The UI must not be built in a way that forbids adding them later.

---

## 4. Token reconciliation (resolved this pass)

- **DELETE dark** is `#e06552` (palette `darken/lighten` in
  `themes/dark/dark.js`), matching `design-system.md` §1.5 and the repo test
  `internal/ui/theme/theme_test.go:57`. The earlier `tokens.json` value came from
  a wrong `lighten` model and is fixed.
- **`requestTabs.bg`** dark is `#26292b`; **`requestTabs.icon.color`** dark is
  `#9f9f9f`; **`requestTabs.icon.hoverColor`** `#cccccc`; **`hoverBg`**
  `#1e1e1e`. `design-system.md` §6.3 now matches.

---

## 5. Resolved & remaining questions

Most of the first pass's open questions are now answered from source and folded
into the screen/component files. Resolved items are kept as a record so a
builder does not re-open them.

### Resolved (with source)

- **`Cmd/Ctrl J` layout toggle** — bound after all: `bindAction('changeLayout', ...)`
  at `providers/Hotkeys/index.js:386`; key mapping `command+bind+j` /
  `ctrl+bind+j`, name "Change Orientation" (`providers/Hotkeys/keyMappings.js:43`).
  It flips `preferences.layout.responsePaneOrientation` horizontal<->vertical.
- **`HttpMethodSelector` caret** — `showCaret` defaults `false` and no call site
  passes `true` (`RequestPane/QueryUrl/index.js:407`,
  `Sidebar/NewRequest/index.js:502`). The URL bar method control deliberately has
  no caret; render it that way.
- **`input.background` (Zoom select)** — a genuine source bug: the token does not
  exist (only `input.bg`), so `Preferences/Display/Zoom/StyledWrapper.js:25`
  renders a transparent background. Go port: use `input.bg`.
- **Import option labels** — verbatim in `screens/import-export.md`
  (Folder arrangement Tags/Paths, Naming requests
  Summary/operationId/Description/Path, Set authentication at collection level,
  Check for Spec Updates, plus their info-tip text).
- **Empty/no-collection state** — it is `WorkspaceOverview`
  (`RequestTabPanel/index.js:40`), not `EmptyAppState`. Copy in
  `screens/welcome.md`. `EmptyAppState` is the deferred App/webview surface.
- **Toast** — no general Toast component; transient messages use
  `react-hot-toast`. Only `Toast/ImportIssuesToast` exists. See
  `components/misc-primitives.md` (Toast).
- **ColorPicker geometry** — preset grid (`#CE4F3B #2E8A54 #346AB2 #C77A0F
#B83D7F #8D44B2`, 20x20 swatches, `No color` with `IconBan`), a six-stop
  discrete slider (`#D85D43 #F4BB74 #61DCB1 #7EBDF2 #D48ADE #B491E5`). See
  `components/misc-primitives.md`.
- **RichTextEditor toolbar** — Bold, Italic, Strikethrough, Inline code, Bullet
  list, Numbered list, Task list, Quote, Code block, Link, Table, Undo, Redo +
  "More formatting options". See `components/misc-primitives.md`.
- **Select / SettingsInput sizing** — `INPUT_SIZES` (`sm` 0.15x0.4 rem, `md`
  0.45 rem) in `components/select.md`; SettingsInput is a `w-24 h-8` row field
  in `components/text-input.md`.
- **Request Docs empty state** — none: the tab shows an action bar only
  (`RequestPane/HttpRequestPane/index.js:149-153`). See `screens/request-docs.md`.
- **Auth mode labels** — `AWS Sig v4`, `Basic Auth`, `Bearer Token`,
  `Digest Auth`, `NTLM Auth`, `OAuth 1.0`, `OAuth 2.0`, `WSSE Auth`,
  `API Key`, `Inherit`, `No Auth`
  (`components/RequestPane/Auth/AuthMode/index.js:28-89`).
- **OAuth2 token placement** — the menu is `Header` / `URL` (not "Headers").
- **Assertion operators** — only `eq`/`neq` are relabelled `equals`/`notEquals`;
  every other operator shows its raw id (`gt`, `startsWith`, ...)
  (`components/RequestPane/Assertions/AssertionOperator/index.js:72-82`). This
  is source-accurate; keep it or normalise all — decide once and document.
- **Secret masking** — one character everywhere: `*` per character, newlines
  preserved, reveal via `SecretEyeButton` (`utils/common/masked-editor.js:65`,
  `MultiLineEditor/index.js:271`, `InheritedVariableRow/index.js:8-9`). The
  "three masking styles" note this section used to carry was wrong; see
  `decisions.md` R-5.

### Decisions taken (see `decisions.md`)

The former "decide" items are now decided and recorded in
[`decisions.md`](decisions.md): sidebar verbs (R-1), request-tab hover (R-2),
preview tabs (R-3), assertion operator labels (R-4), secret masking (R-5),
`input.background` (R-6), global shortcuts (R-7).

### Remaining (needs a fresh capture, not a decision)

- **Screenshot vs source drift.** `refs/side_bar_bruno.png`,
  `folder_bruno.png`, `workspace_switcher_bruno.png` are from a build with a
  top-level tab bar and a `Mock Servers` section the pinned commit renders
  differently. The map follows source at `60d7517`. Needs: captures from a
  build at that commit.
- **Sidebar indent geometry** at depth >= 1 is flex-derived; only the 16 px
  indent cell and 1 px guide are certain. Needs: a pixel check against a fresh
  capture.
- **Preferences / environments surfaces have no captures**; those specs are
  source-derived only. Needs: capture if those screens get a visual pass.

## 6. Verification plan

- **Citations and links are machine-checked.** `python3 ui-map/_tools/verify.py`
  (run from the repo root) asserts every checklist target and internal doc link
  exists, every citation resolves to a real file, and no cited line is past
  end-of-file. It exits non-zero on failure, so it can gate a change. Current
  state: 1,565 citations, 0 unresolved, 0 past EOF.
- Token claims are machine-verified: re-run
  `cd bruno && node ../ui-map/_tools/dump-theme.mjs` and diff `tokens.json`.
- Each screen file's dimension/state tables are checkable with a golden capture
  at 1440x900 (and at 100%/200% DPI). Use the repo's capture harness
  (`internal/ui/app/design_capture_test.go`, opt-in via `CAPTURE_DESIGN=1`) and
  the `UPDATE_GOLDEN` convention.
- Copy claims cite source `file:line`; grep the string to confirm before
  building a label.
