# Screen — app shell (title bar, body frame, status bar)

Mode: **Operate**. The frame every other surface sits in.

Sources: `components/AppTitleBar/` (`index.js`, `StyledWrapper.js`),
`components/StatusBar/` (`index.js`, `StyledWrapper.js`), `components/AppView/`,
`components/CollectionApp/`, `pages/`. Screenshot: `refs/main_app_bruno.png`,
`refs/main_section_bruno.png`, `refs/workspace_switcher_bruno.png`.

---

## Layout tree

```text
┌──────────────────────────────────────────────────────────────────────────┐
│ AppTitleBar (36 px, drag region)                                          │
│  [workspace ▾]                    brun  or  o   (center)   [icons][win]   │
├──────────────┬───────────────────────────────────────────────────────────┤
│              │ RequestTabs                                               │
│  Sidebar     ├───────────────────────────────────────────────────────────┤
│  (§sidebar)  │ RequestPane                    │ ResponsePane              │
│              │  (§request-pane)               │  (§response-pane)         │
│              │         ── drag bar 12 px ──                                │
├──────────────┴───────────────────────────────────────────────────────────┤
│ StatusBar (24 px)                                                         │
└──────────────────────────────────────────────────────────────────────────┘
```

Full-window: title bar → (sidebar | main column) → status bar. The main column
is `RequestTabs` on top, then the request/response split (`RequestTabPanel`,
`components/RequestTabPanel/index.js:673-700`).

---

## AppTitleBar

**Height:** `36px` (`StyledWrapper.js:4`). **Ground:** `sidebar.bg`
(light `#f8f8f8`, dark `#222224`). Custom-drawn chrome, `-webkit-app-region:
drag`; interactive children opt out with `no-drag` (`StyledWrapper.js:6,24-30`).

| Region | Contents | Spec |
| --- | --- | --- |
| Left | Workspace name dropdown trigger: optional pin icon, name, chevron | padding `5px 10px`, radius `6px`, gap `6px`, hover `sidebar.collection.item.hoverBg`, name `13px/500` `sidebar.color`, max-width `180px` with ellipsis (`StyledWrapper.js:44-86`) |
| Center | Bruno wordmark | absolute-centred, `13px/600`, `letter-spacing .5px`, `text` colour, non-interactive (`StyledWrapper.js:89-102`) |
| Right | Home, pin/unpin workspace, new workspace, import, settings, sidebar toggle, console toggle, response-layout toggle, then OS window controls on Win/Linux (`index.js:1-2`, `:36-44`) | 16 px Tabler icons, `strokeWidth 1.5`; min/max/close only when `isWindows || isLinux` (`index.js:36`) |
| Window-control inset | macOS traffic lights | left padding `70px`; `6px` in fullscreen (`StyledWrapper.js:17,21`) |

Window controls on Windows/Linux: minimize (`IconMinus`), maximize/restore
(`IconSquare`/`IconCopy`), close (`IconX`) — `index.js:1`.

Workspace dropdown (left): list of workspaces with pin toggle
(`toggleWorkspacePin`), rename, delete, create, import
(`WorkspaceSidebar/CreateWorkspace`, `WorkspaceSidebar/ImportWorkspace`).
See `components/menu-dropdown.md`.

Copy: untitled workspace shows **"Untitled Workspace"** (`index.js:31-34`).

---

## StatusBar

**Height:** `1.5rem` = 24 px (`StyledWrapper.js:7`). **Ground:** `sidebar.bg`,
**top border** `1px statusBar.border` (light `#E9E9E9`), **text**
`statusBar.color` (`rgb(100,100,100)`), **font-size** `font.size.sm` (12 px),
`user-select: none` (`StyledWrapper.js:9-14`).

Left group (right-to-left visually: buttons then far-left) — buttons render
right-aligned within their section; practical order in source:

| Control | Icon | Tooltip (verbatim) | Action |
| --- | --- | --- | --- |
| Preferences | `IconSettings` | `Preferences` | opens preferences tab (`index.js:29-36,68-86`) |
| Theme | `IconPalette` | — (ThemeDropdown) | theme picker popover (`index.js:90-97`) |
| Notifications | — (`Notifications`) | `Notifications` | notifications list (`index.js:99-104`) |
| GitHub | `IconBrandGithub` | `GitHub Repository` | opens `https://github.com/usebruno/bruno` (`index.js:106-115`) |
| Global Search | `IconSearch` + label `Search` | — | triggers `globalSearch` binding (`index.js:122-134`) |
| Cookies | `IconCookie` + label `Cookies` | — | opens Cookies modal (`index.js:136-147`) |
| Dev Tools | `IconTool` + label `Dev Tools` | — | opens console; shows error count (`index.js:149-165`) |
| Version | — | — | `v{version}` (`index.js:169-171`) |

Icon size 16, `strokeWidth 1.5`. Buttons: `padding: 0 4px`, group gap `2px`
(`StyledWrapper.js:26-42`).

Invalid/error state: Dev Tools button gets `has-errors`; error count badge
`.error-count-inline` — `10px/500`, `colors.text.danger` on
`colors.bg.danger` at 20 % alpha, `padding 1px 4px`, radius `4px`
(`StyledWrapper.js:56-63`).

Divider between action groups: `1px × 16px`, `sidebar.dragbar`, opacity `.4`
(`StyledWrapper.js:65-70`).

---

## States

| State | Behaviour |
| --- | --- |
| Normal | every region visible per layout state |
| Sidebar collapsed | main column expands; sidebar toggle icon reflects state (`toggleSidebarCollapse`, `index.js:9`) |
| Fullscreen (macOS) | title-bar left padding drops `70px → 6px`, `titlebar-left` margin-left `10px → 0` (`StyledWrapper.js:19-28,36-38`) |
| Windows/Linux | window controls shown; macOS hides them (`index.js:36`) |
| Console open with errors | Dev Tools button `has-errors`, inline count |

---

## Interactions & shortcuts

- Title-bar drag moves the window; all buttons/dropdowns are `no-drag`.
- Status-bar buttons are keyboard-focusable (`tabIndex={0}`) with
  `aria-label` on each (`index.js:71-84`, etc.).
- `⌘/Ctrl + K` opens global search (via `globalSearch` binding, triggered in
  `index.js:46-51`). See `screens/global-search.md`.

## Open Questions

- `refs/side_bar_bruno.png` shows a top-level tab bar layout that the pinned
  commit renders as workspace dropdown + request tabs; the screenshot is from
  a different build. Source is authoritative here.
