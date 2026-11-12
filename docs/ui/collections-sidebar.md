# Collections Sidebar

Status: VERIFIED against `bruno/packages/bruno-app/src/components/Sidebar/` at commit `60d7517` and cross-referenced with `ui-map/screens/sidebar.md` and `ui-map/screens/sidebar-dialogs.md`.

---

## 1. Structure and Dimensions

```text
┌ aside.sidebar (width 220–600 px, default 248 px) ────────────────┐ ┃   ← drag handle 6 px
│ ┌ accordion-section-wrapper #collections ─────────────────────┐ │ ┃
│ │ [▾] [▣] Collections                    [🔍] [+] [⋮]          │ │ ┃   section header 28 px
│ │──────────────────────────────────────────────────────────────│ │ ┃
│ │ (search field — only when search toggled on: ⌘F)             │ │ ┃   input 28 px
│ │ ▸ Collection A                        count/env-badge [⋮]    │ │ ┃   row 26 px (1.6 rem)
│ │   │ ▾ folder                                [⋮]             │ │ ┃   +16 px indent per depth
│ │   │   │ GET  Request name                          [⋮]       │ │ ┃   verb slot 34 px (R-1)
│ │   │   │   │ 📄 example response                [⋮]           │ │ ┃
│ │   │   │ + Add request                                        │ │ ┃   empty-cta row
│ └──────────────────────────────────────────────────────────────┘ │ ┃
│ ┌ accordion-section-wrapper #api-specs (collapsed) ───────────┐ │ ┃
│ │ [▸] [</>] API Specs                              [+]         │ │ ┃
│ └──────────────────────────────────────────────────────────────┘ │ ┃
└──────────────────────────────────────────────────────────────────┘ ┃
```

### Container Metrics

| Aspect             | Concrete Value                                                                       | Source Citation                               |
| ------------------ | ------------------------------------------------------------------------------------ | --------------------------------------------- |
| **Width range**    | Min `220px`, Max `600px`, default `248px`                                            | `components/Sidebar/index.js:16-17`           |
| **Persistence**    | Stored in preferences under `SIDEBAR_WIDTH_KEY`                                      | `components/Sidebar/index.js:8, 42, 79-90`    |
| **Resize guard**   | Ignores drag movements < `3px`                                                       | `components/Sidebar/index.js:70`              |
| **Background**     | `sidebar.bg` → `#f8f8f8` L / `#222224` D                                             | `components/Sidebar/StyledWrapper.js:51`      |
| **Text colour**    | `sidebar.color` → `#343434` L / `#cccccc` D                                          | `components/Sidebar/StyledWrapper.js:5`       |
| **Drag handle**    | Width `6px`, offset `right: -3px`, `col-resize` cursor                               | `components/Sidebar/StyledWrapper.js:148-157` |
| **Split hairline** | 1 px border `sidebar.dragbar.border` (`#e5e5e5` L / `#333333` D; active `#444444` D) | `components/Sidebar/StyledWrapper.js:159-167` |

---

## 2. Section Headers (`components/Sidebar/SidebarSection/`)

Three sections appear in fixed order: `collections`, `api-specs` (deferred), and `mock-servers` (deferred). `collections` is expanded by default (`defaultExpanded: ['collections']`).

| Element            | Specification                                                           | Source Citation                             |
| ------------------ | ----------------------------------------------------------------------- | ------------------------------------------- |
| **Height**         | `28px` (`min-height: 28px; height: 28px`)                               | `SidebarSection/StyledWrapper.js:33-34`     |
| **Padding**        | `6px 4px 6px 8px`, header gap `16px`, left cluster gap `6px`            | `SidebarSection/StyledWrapper.js:31-32, 43` |
| **Title**          | `12px`, weight `600`, single line with ellipsis                         | `SidebarSection/StyledWrapper.js:88-95`     |
| **Section icon**   | `14px`, stroke `1.5`, color `sidebar.muted` (`#838383` L / `#aaaaaa` D) | `SidebarSection/index.js:62`                |
| **Caret toggle**   | `12px` `IconChevronRight` (collapsed) → `IconChevronDown` (expanded)    | `SidebarSection/index.js:56-60`             |
| **Action buttons** | 22 px square (`ActionIcon size="sm"`), gap `1px`                        | `Sidebar/StyledWrapper.js:97-102`           |

Header actions for Collections: `Search requests` (`IconSearch`), `Add new collection` (`IconPlus`), `More actions` (`IconDotsVertical`).

---

## 3. Tree Rows and Node Anatomy

### Row Dimensions and Indentation

- **Row height:** `1.6rem` = 26 px.
- **Indentation:** `16px` left padding added per depth level (`depth * 16px`).
- **Hierarchy line:** 1 px vertical border `sidebar.collection.item.indentBorder` (`#e5e5e5` L / `#333333` D).
- **Active state:** Background `sidebar.collection.item.bg` (`#eaeaea` L / `#444444` D).
- **Hover state:** Background `sidebar.collection.item.hoverBg` (`#f1f1f1` L / `#26292b` D).

### Verb Rendering: Decision R-1 (`RequestMethod`)

> [!IMPORTANT]
> **Decision R-1:** Sidebar request rows do **not** use `MethodBadge` (which carries borders and background fills). Instead, they render `RequestMethod`:
>
> - Font size: `11px` (`font.size.xs`), bold (`font-weight: 600`).
> - Fixed slot width: `34px` (`RequestMethod/StyledWrapper.js`), right-aligned text.
> - Truncation rule: HTTP verbs longer than 5 characters are truncated to 3 letters (e.g. `DELETE` → `DEL`, `OPTIONS` → `OPT`).
> - Special protocols: `grpc` (blue), `ws` (amber), `gql` (magenta).

---

## 4. Context Menu Actions (Complete Inventory)

Right-clicking an item opens `ui/ContextMenu`:

| Target Node    | Verbatim Actions                                                                    | Behavior                                            |
| -------------- | ----------------------------------------------------------------------------------- | --------------------------------------------------- |
| **Collection** | New Request, New Folder, Run, Run with Tags, Rename, Export, Clone, Close, Settings | Opens modals in `ui-map/screens/sidebar-dialogs.md` |
| **Folder**     | New Request, New Folder, Run, Rename, Clone, Delete, Settings                       | Subtree operations                                  |
| **Request**    | Rename, Clone, Duplicate, Delete, Copy Path, Generate Code, Set As Variable         | Request management                                  |

---

## 5. Drag and Drop Interaction

- **Reorder within folder:** Mouse drag initiates row reorder (`useMouseRowDrag`).
- **Drop target indicator:** 2 px solid line in `dragAndDrop.border` (`#d37f17` L / `#e4ae49` D) with `hoverBg`.
- **Cross-collection drag:** Blocked by cursor (`dragBlockedCursor.js`) with red no-entry indicator.
- **Persistence:** Dropping triggers sequence renumbering (`resequence-items`), updating `seq` in `.bru` metadata.

---

## 6. Performance in Large Collections

- **Chunked rendering:** Virtualized row reveal (`useChunkedReveal`, `useDeferredLoading`).
- **Lazy expansion:** Subtree items are instantiated only when parent folder is expanded (`useExpandedRows`).
- **Auto-reveal:** Switching tabs automatically scrolls the sidebar to reveal and highlight the active request (`useSidebarAutoReveal`).
- **Debounced search:** 200 ms debounce with `fast-fuzzy` scoring across collection names and request paths.

---

## 7. Go Implementation Mapping

- **Sidebar Root:** MyGo `ui.Column` with width clamped to `[220, 600]` DIPs.
- **Tree View:** Implemented using MyGo `ui.Outline` with virtual row recycling to handle collections with >5,000 requests smoothly.
- **Method Verbs:** Custom `widgets.RequestMethod` renderer observing R-1 (11 px text, 34 px right-aligned cell, 3-char truncation).
- **Splitter:** Drag handle with 6 DIP hit target and 1 DIP visible border line.
