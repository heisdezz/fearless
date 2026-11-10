# Navigation

Status: VERIFIED unless marked.

---

## 1. Navigation model

Bruno has **no URL-based routing** for collections. The app is a single workspace view where navigation is:

1. **Tabs** — the only way to view a request (like a code editor).
2. **Sidebar** — the tree that feeds tabs.
3. **Menus & hotkeys** — commands that open modals or tabs.

Route-level pages just switch top-level mode:

```text
pages/index.js → Main
  ├─ Welcome / onboarding
  ├─ ManageWorkspace  (state.app.showManageWorkspacePage)
  └─ Bruno            (collections view)
```

Source: `packages/bruno-app/src/pages/Bruno/index.js` (`showManageWorkspacePage` selector), `pages/Main.js`

---

## 2. Tabs

### Tab anatomy

| Part           | Detail                                                                            |
| -------------- | --------------------------------------------------------------------------------- |
| Request tab    | Method badge + name + optional protocol icon + dirty dot + close                  |
| Collection tab | Collection name + header; opens collection settings / runner                      |
| Example tab    | Example name, distinct icon                                                       |
| Special tabs   | `ApiSpecTab`, `OpenAPISpecTab`, `OpenAPISyncTab`, `MockServerTab`, `ChangelogTab` |

Components: `RequestTabs/index.js`, `RequestTabs/RequestTab/index.js` (924 lines), `RequestTabs/CollectionHeader/index.js` (821 lines), `RequestTabs/ExampleTab`.

### Tab behaviors

| Behavior                 | Detail                                                                        |
| ------------------------ | ----------------------------------------------------------------------------- |
| Single instance per item | Opening an already-open item focuses it (`focusTab`)                          |
| Reorder                  | Drag & drop within the collection group                                       |
| Scoped switching         | ⌘1–⌘8 jump to tab position **within the active collection**                   |
| Reopen closed            | `reopenClosedTab` thunk                                                       |
| Dirty indicator          | From `draftDetectMiddleware`                                                  |
| Restore on restart       | Persisted via snapshot + `slices/tabs` restore logic (`tabs-restore.spec.js`) |
| Request state badge      | Spinner while queued/sending; error dot on failure                            |

Source: `providers/Hotkeys/index.js` (`switchToTab1..8`, `reopenClosedTab`), `providers/ReduxStore/slices/tabs.js`

### Per-tab persisted UI state

Each tab object carries:

```text
uid, collectionUid, type ('request'|'collection'|'example'|'api-spec'|…),
requestPaneTab, responsePaneTab, responseFormat, responseViewTab,
requestPaneOrientation, preview …
```

This is why switching tabs restores exactly the sub-tab and response format you left.

---

## 3. Sidebar navigation

```text
Workspace selector
├── Collections (each expandable tree)
│   ├── Folder
│   │   ├── Request
│   │   └── Folder …
│   ├── Environments (section)
│   ├── Mock servers / API specs (sections)
│   └── Git status (section)
├── Import
├── New request / folder / collection
└── Search (sidebar filter + global search)
```

Source: `components/Sidebar/index.js`, `SidebarContent.js`, `SidebarSection`, `Sidebar/Sections/*`

### Sidebar interactions

| Interaction       | Behavior                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Click             | Open item in a tab                                                                                                        |
| Right-click       | Context menu (rename, clone, delete, copy path, run, share…)                                                              |
| Drag              | Move/reorder; cross-collection moves blocked with a cursor (`dragBlockedCursor.js`)                                       |
| Keyboard          | `Escape` clears selection (`useClearSidebarSelectionOnEscape`), search filters, arrow keys per `useSidebarSelectionClick` |
| Auto-reveal       | Active tab's item is revealed when switching tabs (`useSidebarAutoReveal`)                                                |
| Sections collapse | Persisted (`usePersistedState`, `useExpandedRows`)                                                                        |

---

## 4. Global search

`components/GlobalSearchModal` + `utils/collections/search.js` + `fast-fuzzy`:

- Opens with ⌘/Ctrl + K.
- Searches request names, URLs, methods, folder names, and (optionally) request contents.
- Results grouped by collection; Enter opens the item.

Sidebar-local search (⌘/Ctrl + F while sidebar focused) filters the tree in place.

---

## 5. Collection header (context control center)

`RequestTabs/CollectionHeader` sits above the request tabs and contains:

- Environment selector (`Environments/EnvironmentSelector`)
- Collection settings, folder settings
- Runner launcher
- Cookies manager
- Share / git actions
- Docs / overview toggle

It is per-collection, so multiple open collections each keep their own active environment.

---

## 6. Keyboard-driven navigation summary

| Action             | Shortcut                                             |
| ------------------ | ---------------------------------------------------- |
| Tab 1–8            | ⌘/Ctrl + 1..8                                        |
| Close tab          | ⌘/Ctrl + W                                           |
| Next/previous tab  | no default binding (cycle via tab-position bindings) |
| Global search      | ⌘/Ctrl + K                                           |
| Sidebar search     | ⌘/Ctrl + F                                           |
| New request        | ⌘/Ctrl + N                                           |
| Edit environment   | ⌘/Ctrl + E                                           |
| Import collection  | ⌘/Ctrl + O                                           |
| Preferences        | ⌘/Ctrl + ,                                           |
| Toggle sidebar     | ⌘/Ctrl + \                                           |
| Change orientation | ⌘/Ctrl + J                                           |

Exact bindings: `providers/Hotkeys/keyMappings.js`.

---

## 7. Go port guidance (PROPOSED)

- Model navigation as a `NavigationStack` of tabs in core state with stable item UIDs (Bruno's `getRequestUid` path-hash trick is worth copying so tabs survive external edits).
- Keep per-tab sub-state; users rely on it.
- Implement the tab strip with overflow dropdown from the start.
