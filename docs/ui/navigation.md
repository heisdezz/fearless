# Navigation

Status: VERIFIED against `bruno/packages/bruno-app/src/` at commit `60d7517` and cross-referenced with `ui-map/screens/*`.

---

## 1. Top-Level Navigation Model

Bruno operates as a document-style workspace without URL routing:

1. **Tabs Strip:** Primary container for viewing requests, runners, and editors.
2. **Sidebar Tree:** Hierarchical collection explorer driving tab creation and focus.
3. **Modals & Dialogs:** Dedicated overlays for creation, import, preferences, cookies, and search.

```text
App Window
  ├── AppTitleBar (Workspace switcher dropdown)
  ├── Body
  │    ├── Sidebar (Collections tree, search, actions)
  │    └── Workspace Canvas
  │         ├── RequestTabs (Tab strip)
  │         └── Panes (Request / Response split)
  └── StatusBar (Preferences, search, cookies, devtools triggers)
```

---

## 2. Request Tabs Specification (`ui-map/screens/request-tabs.md`)

- **Strip Height:** `34px`.
- **Background:** `requestTabs.bg` (`#f6f6f6` L / `#26292b` D).
- **Tab Anatomy:**
  - Method indicator: Compact `MethodBadge` (`GET`, `POST`, etc.) with theme colors.
  - Title: Item name (13 px).
  - Dirty dot: Amber circle (`draftColor` `#cc7b1b`) when unsaved draft changes exist.
  - Close button: 16 px icon with circular hover background (`requestTabs.icon.hoverBg` `#1e1e1e` D).
- **Decision R-2 (Tab Hover):** Request tabs have **no background hover tint**; only the close button highlights on hover.
- **Decision R-3 (Preview Tabs):** Transient / preview tabs (opened via single click in sidebar) are styled with **italic text only**; no custom color token is used.

### Tab Shortcuts & Switching

| Shortcut               | Action                                     | Scope             |
| ---------------------- | ------------------------------------------ | ----------------- |
| **⌘/Ctrl + W**         | Close active tab                           | Active collection |
| **⌘/Ctrl + Shift + T** | Reopen last closed tab (`reopenClosedTab`) | Global            |
| **⌘/Ctrl + 1..8**      | Switch to tab index 1–8                    | Active collection |
| **Drag & Drop**        | Reorder tabs horizontally                  | Tab strip         |

---

## 3. Dedicated Navigation Screens

### 3.1 Global Search Modal (`ui-map/screens/global-search.md`)

- Opened via **⌘/Ctrl + K** or status bar search button.
- Modal width: `600px`.
- Real-time search across: Request names, URLs, HTTP headers, request body contents, and scripts across all mounted collections.
- Arrow key navigation with Enter to open result in a tab.

### 3.2 Collection Runner Screen (`ui-map/screens/runner.md`)

- Dedicated runner tab opened from collection context menu.
- Configuration: Select folder, select environment, iteration count, inter-request delay (ms), data file runner (`.json`/`.csv`).
- Live execution view: Progress bar, real-time passing/failing status chips, response timing charts, and exportable run summary.

### 3.3 Cookies Management Modal (`ui-map/screens/cookies.md`)

- Opened from status bar `Cookies` button.
- Inspects and modifies the global cookie jar (`internal/core/cookies`).
- Domain list on the left, cookie details table on the right with secret attributes (HttpOnly, Secure, SameSite, Expiry).

### 3.4 Workspace Switcher (`ui-map/screens/workspace-switcher.md`)

- Opened from title bar workspace trigger.
- Switch between multi-collection workspace projects, pin favorite workspaces, create new workspaces, or clone remote repositories.

---

## 4. Global Hotkey Routing (Decision R-7)

> [!IMPORTANT]
> **Decision R-7:** Core layout and navigation shortcuts are bound globally in the root event loop (`providers/Hotkeys/index.js:386`):
>
> - **⌘/Ctrl + J:** Toggle request/response pane layout between Stacked (vertical) and Side-by-Side (horizontal).
> - **⌘/Ctrl + `\`:** Toggle sidebar expand / collapse.
> - **⌘/Ctrl + Enter:** Send active request (or Cancel if in-flight).
> - **⌘/Ctrl + S:** Save active request draft.
