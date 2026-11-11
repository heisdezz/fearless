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

| Source behaviour | Go approach |
| --- | --- |
| CSS layout (flexbox/grid, gap, position) | MyGo layout: `Row`/`Column`/`Box`, `Gap`, `Grow`, `Fill`, explicit widths; no CSS. Re-implement measured sizes from each screen's dimension tables. |
| `-webkit-app-region: drag` title bar | MyGo custom title bar with `DragWindow()`; honour per-OS window-control insets (macOS 70 px left, 0 in fullscreen). |
| CodeMirror | Custom editor widget (see `widgets.md`): syntax highlight, gutter, variable spans. No CodeMirror in Go. |
| `contenteditable` / TipTap rich text | Native rich-text editor or markdown-with-preview; must keep bold/italic/headings/lists/links/code. |
| tippy.js popovers (dropdowns, tooltips) | MyGo popover/overlay layer (`Portal` equivalent) with viewport clamping. |
| react-hot-toast | `feedback.ToastHost` (already in repo). |
| HTML `<select>` / native scroll | Lobe `field.Select` and MyGo scroll containers; custom scrollbar (`scrollbar.color`, width ~6–8). |
| Copy to clipboard, open external URL | Platform layer (`internal/platform`), not IPC. |
| Drag-and-drop (rows, sidebar, files) | MyGo drag events; drop indicator using `dragAndDrop.border` `2px solid` + `hoverBg`. |
| Responsive overflow (tab strip chevron) | MyGo measure/overflow; keep the reserved chevron width to keep layout stable. |
| Resizable panes | `layout.RootView` splitters already exist; keep 12 px drag bars and min sizes from the screen files. |

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

## 5. Open questions (rolled up from every file)

### Shell & sidebar
- **Screenshot vs source drift.** `refs/side_bar_bruno.png`,
  `folder_bruno.png`, `workspace_switcher_bruno.png` are from a build with a
  top-level tab bar and a `Mock Servers` section that the pinned commit renders
  differently (per-collection tabs; mock servers behind a beta flag). The map
  follows source at `60d7517`. Before locking visuals, capture fresh shots of the
  pinned commit via `npm run dev:electron`.
- **Sidebar indent geometry** at depth ≥ 1 is flex-derived; needs a pixel check
  against a fresh capture. Only the 16 px indent cell and 1 px guide are certain.
- **`MethodBadge` unused by sidebar** at this commit: sidebar uses a
  `min-width: 34px` text slot. Choose the badge or the slot explicitly.
- **Light hover == selection** (`#eaeaea` both): only the 3 px accent rail
  distinguishes them. Intentional in source; decide whether to keep.
- **Request tab hover/preview**: no hover token and no preview colour token;
  preview is italics only. Do not invent colours.

### Request / response
- `⌘/Ctrl J` (`changeLayout`) is declared in `providers/Hotkeys/keyMappings.js`
  but no binding was found — verify before shipping the shortcut.
- `HttpMethodSelector` defaults `showCaret=false`; the URL bar shows no method
  caret even though other selectors do. Intent unclear.
- `ResponsiveTabs` reserves 60 px for an 18 px chevron — reproduce or
  right-size; call it out.
- `EditableTable` sets a `drag-over` class with no drop indicator and ignores
  `dragAndDrop.*`; implement an indicator explicitly.
- GraphQL is a separate request type, not an HTTP body mode.
- Auth copy is inconsistent in source: `NTLM Auth` (menu) vs `NTLM` (selected);
  `AWS Sig v4` vs `AWS Sig V4`; `Query Param` vs `Query Params`; `Header` vs
  `Headers` (OAuth2 token placement); EdgeGrid "Advanced Settings" renders
  unconditionally despite a basic/advanced split. Pick one per surface and
  document it.
- Assertion operators show raw ids (`gt`, `startsWith`) except `eq`/`neq`
  (`equals`/`notEquals`).
- Request Docs tab has no empty-state placeholder.

### Settings & environments
- Preferences has **ten** tabs (General, Themes, Display, Proxy, Client
  Certificates, Keybindings, AI, Cache, Support, Beta) — no Request/Font/Privacy/
  About tabs; request behaviour is inside General, font inside Display, privacy
  inside AI › Security (`screens/preferences.md`).
- `SecuritySettings` is the per-collection JS-sandbox selector, not a
  Preferences tab.
- Missing `input.background` token used by the Zoom select
  (`Display/Zoom/StyledWrapper.js:25`).
- Secret masking differs across three surfaces (CodeMirror bullets /
  `********` / per-char `*`) — unify.
- Client-cert path wording conflicts (absolute vs relative); folder `Test` vs
  collection `Tests`; Expr tooltip drift (Template Literal vs expression).
- `refs/workspace_switcher_bruno.png` is a workspace switcher, not an
  environment surface; no environment screenshots exist, so
  `screens/environments.md` is source-derived only.

### Import / export & docs
- Import option labels live in per-option components; read them at build time.
- `EmptyAppState` / `WorkspaceHome` copy not exhaustively captured.
- `ResponseExample` should be `QueryResponse` minus live actions.

### Primitives
- Select trigger height/padding follow `InputWrapper` — confirm once.
- `SettingsInput` vs `TextInput` visual delta not read line-by-line.
- Toast visuals and ColorPicker geometry still need a source read.
- RichTextEditor toolbar contents not enumerated.

---

## 6. Verification plan

- Each screen file's dimension/state tables are checkable with a golden capture
  at 1440×900 (and at 100 %/200 % DPI). Use the repo's existing capture harness
  (`internal/ui/app/zz_design_test.go`) and the `UPDATE_GOLDEN` convention.
- Token claims are machine-verified: re-run
  `cd bruno && node ../ui-map/_tools/dump-theme.mjs` and diff `tokens.json`.
- Copy claims cite source `file:line`; grep the string to confirm before
  building a label.
