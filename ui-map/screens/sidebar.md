# Screen — Sidebar

Components: `components/Sidebar` (container, sections, rows) and its sub-trees.
Related screens: `screens/sidebar-dialogs.md` (New Request, New Folder, Import, Open,
Create Collection, Clone Git, Close Workspace, Postman report), `screens/request-tabs.md`
(the tab bar that mirrors sidebar selection).

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source at `60d7517`, except the items listed under
**Open Questions**. Colours and sizes are copied verbatim; no value is rounded.

Screenshots: `refs/side_bar_bruno.png`, `refs/sidebar_plus_options.png`,
`refs/sidebar_search_bar_open.png`, `refs/workspace_switcher_bruno.png` (header context only).

---

## 1. Layout overview

```text
┌ aside.sidebar  width = leftSidebarWidth (220–600 px) ────────────┐ ┃   ← drag handle 6 px
│ ┌ accordion-section-wrapper  #collections ────────────────────┐ │ ┃
│ │ [▾] [▣] Collections                    [🔍] [+] [⋮]          │ │ ┃   section header 28 px
│ │──────────────────────────────────────────────────────────────│ │ ┃
│ │ (search field — only when search toggled on)                │ │ ┃
│ │ (inline creator — only while creating a collection)         │ │ ┃
│ │ ▸ Collection A                        count/badge  [⋮]      │ │ ┃   row 1.6 rem = 26 px
│ │   │ ▾ folder                                [⋮]            │ │ ┃   +16 px indent per depth
│ │   │   │ GET  Request name                          [⋮]      │ │ ┃
│ │   │   │   │ 📄 example response                [⋮]          │ │ ┃
│ │   │   │ + Add request                                       │ │ ┃   empty-cta row
│ │ ▸ GitRemoteCollectionRow (ghost)              [⋮]           │ │ ┃
│ └──────────────────────────────────────────────────────────────┘ │ ┃
│ ┌ accordion-section-wrapper  #api-specs  (collapsed) ─────────┐ │ ┃
│ │ [▸] [</>] API Specs                              [+]        │ │ ┃
│ └──────────────────────────────────────────────────────────────┘ │ ┃
│ ┌ accordion-section-wrapper  #mock-servers (beta, collapsed) ─┐ │ ┃
│ │ [▸] [🖥] Mock Servers                             [+]        │ │ ┃
│ └──────────────────────────────────────────────────────────────┘ │ ┃
└──────────────────────────────────────────────────────────────────┘ ┃
```

- Three sections in fixed order: `collections`, `api-specs`, and — only when the beta flag
  `MOCK_SERVER` is on — `mock-servers` (`components/Sidebar/index.js:21-41`).
- Only `collections` is expanded by default (`defaultExpanded: ['collections']`,
  `components/Sidebar/index.js:122`; default parameter `components/Sidebar/SidebarAccordionContext.js:13`).
- Expanded section grows (`flex: 1 1 0%`); collapsed sections are `flex: 0 0 auto` and, when
  a section above them is expanded, are pushed to the bottom (`pinned-to-bottom`,
  `components/Sidebar/SidebarContent.js:41-48`, `components/Sidebar/StyledWrapper.js:132-144`).
- Section separator: `1px` top border in `sidebar.collection.item.hoverBg` for every wrapper
  except the first (`components/Sidebar/StyledWrapper.js:116-118`).

---

## 2. Container (`components/Sidebar/index.js`, `StyledWrapper.js`)

| Aspect | Value | Source |
| --- | --- | --- |
| Min / max width | `220` / `600` px | `components/Sidebar/index.js:16-17` |
| Width source | `state.app.leftSidebarWidth`, persisted to `localStorage` under `SIDEBAR_WIDTH_KEY` | `index.js:8,42,79-90` |
| Resize increment guard | ignore moves < `3` px | `index.js:70` |
| Width transition | `width 0.2s ease-in-out` (disabled while dragging) | `index.js:124` |
| Collapsed width | `0` | `index.js:61` |
| Background | `sidebar.bg` → `#f8f8f8` L / `#222224` D | `StyledWrapper.js:51`; tokens `tokens.json` `sidebar.bg` |
| Text colour | `sidebar.color` → `#343434` L / `#cccccc` D | `StyledWrapper.js:5` |
| Drag handle | width `6px`, offset `right: -3px`, `cursor: col-resize`, `transition: opacity 0.2s ease` | `StyledWrapper.js:148-157` |
| Drag handle line | `2px` box, `border-left: solid 1px sidebar.dragbar.border` (`#e5e5e5` L / `#333333` D) | `StyledWrapper.js:159-163`; tokens `sidebar.dragbar.border` |
| Drag handle hover line | `1px` `sidebar.dragbar.activeBorder` (`#e5e5e5` L / `#444444` D) | `StyledWrapper.js:165-167` |
| Section content scroll | `overflow-y: auto`, `overflow-x: hidden` | `SidebarSection/StyledWrapper.js:105-113` |

The drag handle is only rendered when the sidebar is not collapsed (`index.js:138`).

---

## 3. Section header (`components/Sidebar/SidebarSection/`)

| Aspect | Value | Source |
| --- | --- | --- |
| Height | `28px` (`min-height: 28px; height: 28px`) | `SidebarSection/StyledWrapper.js:33-34` |
| Padding | `6px 4px 6px 8px` | `:32` |
| Header gap | `16px` | `:31` |
| Left cluster gap | `6px` | `:43` |
| Bottom border | `1px solid transparent` (becomes `sidebar.collection.item.hoverBg` when section expanded) | `:38`, `components/Sidebar/StyledWrapper.js:71-73` |
| Background transition | `background-color 0.15s ease` | `:36` |
| Title | `12px`, weight `600`, `sidebar.color`, single line + ellipsis | `:88-95` |
| Icon wrapper | `24 × 24px` | `:66-73` |
| Section icon | `14px`, stroke `1.5`, `sidebar.muted` (`#838383` L / `#aaa` D) | `SidebarSection/index.js:62`; `StyledWrapper.js:79-86` |
| Caret (icon swap) | `IconChevronRight` `12px` collapsed → `IconChevronDown` `12px` expanded, stroke `1.5` | `SidebarSection/index.js:56-60` |
| Caret visibility | hidden by default; only shown on header-left hover, which also hides the section icon and paints the caret `dropdown.hoverBg` | `StyledWrapper.js:49-63,75-77` |
| Action icons | `ActionIcon` `size="sm"` (22 px button), gap `1px` | `StyledWrapper.js:97-102`; `ui/ActionIcon/StyledWrapper.js` `sizeMap.sm = 22` |

### 3.1 Section titles and header actions (verbatim)

| Section | id | Title | Icon | Header actions | Source |
| --- | --- | --- | --- | --- | --- |
| Collections | `collections` | `Collections` | `IconBox` | `Search requests`, `Add new collection`, `More actions` | `Sections/CollectionsSection/index.js:294-332,405-410` |
| API Specs (deferred) | `api-specs` | `API Specs` | `IconFileCode` | `Add new API Spec` | `Sections/ApiSpecsSection/index.js:43-57,66-70` |
| Mock Servers (deferred, beta) | `mock-servers` | `Mock Servers` | `IconServer` | `Create mock server` | `Sections/MockServersSection/index.js:11-19,26-31` |

Deferred sections (`ApiSpecs`, `MockServers` trees) are out of v1 UI scope per
`ui-map/overview.md:188-193`; only their header contract is specified here.

### 3.2 Collections header menus

`Add new collection` (`IconPlus`, 14 px, stroke 1.5) opens `MenuDropdown` `placement="bottom-end"`.
Order (`Sections/CollectionsSection/index.js:239-265`):

| # | id | Left icon | Label (verbatim) | Shortcut hint | Action |
| --- | --- | --- | --- | --- | --- |
| 1 | `create` | `IconPlus` | `Create collection` | — | inline creator (`index.js:244-246`) |
| 2 | `open` | `IconFolder` | `Open collection` | — | `setIsOpeningCollection(true)` (`:252-254`) |
| 3 | `import` | `IconDownload` | `Import collection` | `importCollection` binding (`Cmd/Ctrl+O`) | open Import modal (`:260-263`) |

`More actions` (`IconDotsVertical`, 14 px) opens `placement="bottom-end"` (`:267-292`):

| # | id | Left icon | Label (verbatim) | Source |
| --- | --- | --- | --- | --- |
| 1 | `sort` | rotates by state | `Sort A-Z` → `Sort Z-A` → `Clear sort` | `:155-175,267-275` |
| 2 | `close-all` | `IconSquareX` | `Close all` | `:276-283` |
| 3 | `open-in-terminal` | `IconTerminal2` | `Open in Terminal` | `:284-291` |

Sort cycle: `default → alphabetical → reverseAlphabetical → default`
(`:136-153`); icon per state `IconSortAscendingLetters` / `IconSortDescendingLetters` /
`IconArrowsSort` (`:155-164`).

`Search requests` (`IconSearch`, 14 px) toggles `state.app.showSidebarSearch`
(`:132-134,296-302`).

---

## 4. Rows

### 4.1 Shared row metrics

| Aspect | Value | Source |
| --- | --- | --- |
| Row height | `1.6rem` (= `25.6px` at a 16 px root; Virtuoso default item height `26`) | `SidebarRowStyles/index.js:4`; `Collections/CollectionItem/CollectionItemRow/StyledWrapper.js:24`; `Collections/index.js:141` |
| Cursor / selection | `cursor: pointer`, `user-select: none` | `SidebarRowStyles/index.js:5-6` |
| Left border | `4px solid transparent` (collection rows) / `3px solid transparent` (item rows) | `SidebarRowStyles/index.js:7`; `CollectionItemRow/StyledWrapper.js:28` |
| Indent unit | `16px` per depth level, each cell is an `.indent-block` of `width: 16px; min-width: 16px; height: 100%` | `CollectionItemRow/index.jsx:409,764-776`; `ExampleItem/index.jsx:73,236-246` |
| Indent guide | `border-right: 1px solid sidebar.collection.item.indentBorder` (`#e5e5e5` L / `#26292b` D) | `CollectionItemRow/StyledWrapper.js:19-21` |
| Content padding-left | `8px` on the inner flex row | `CollectionItemRow/index.jsx:779` |
| Collection row padding-left | `4px` (+ row's `py-1`, i.e. 4 px vertical) | `CollectionRow/StyledWrapper.js:6`; `CollectionRow/index.jsx:375` |
| Method/min-width slot | `RequestMethod` box: `min-width: 34px`, `flex-shrink: 0`, `align-self: stretch` | `CollectionItem/RequestMethod/StyledWrapper.js:5-9` |
| Menu icon | `IconDots` `18px` (item rows) / `18px` (collection, example rows `22px`), hidden until hover/focus | `CollectionItemRow/index.jsx:828-830`; `CollectionRow/index.jsx:619-621`; `ExampleItem/index.jsx:265` |
| Menu trigger colour | `sidebar.dropdownIcon.color` (`#666666` L / `#cccccc` D) | `CollectionItemRow/StyledWrapper.js:5-6`; tokens `sidebar.dropdownIcon.color` |

### 4.2 Indentation per depth (from the flat row model)

Rows are flattened with an explicit `depth` (`utils/collections/flattenSidebarTree.js`):

| Row kind | `depth` | Left offset (visual) | Source |
| --- | --- | --- | --- |
| Collection header | `0` | `4px` padding only | `flattenSidebarTree.js:199-209`; `CollectionRow/StyledWrapper.js:6` |
| Git ghost row | `0` | `4px` padding | `flattenSidebarTree.js:299-307`; `GitRemoteCollectionRow/StyledWrapper.js:10` |
| Top-level folder / request / app | `1` | one 16 px indent block + 8 px | `flattenSidebarTree.js:228-232` |
| Nested folder / request | `depth` | `depth × 16px` + 8 px | `flattenSidebarTree.js:90-92` |
| Example child of a request | parent depth + 1 | `depth × 16px` + 8 px (+ 16 px icon spacer) | `flattenSidebarTree.js:157-171`; `ExampleItem/index.jsx:251` |
| Empty CTA row | folder depth + 1 (root: `1`) | `depth × 16px` + 8 px, plus `ml-1` on the link | `flattenSidebarTree.js:97-109,241-251`; `EmptyCtaRow/index.jsx:21-34` |

A dash-tree implementation should compute left padding as
`rowLeftPadding = 4px (collection) | depth × 16px + 8px (item/example/cta)` with a
`1px` vertical guide per indent cell. **Marked inference:** the exact pixel where the
first indent guide sits depends on the flex layout; only the `16px` cell width and
`1px` guide colour are asserted in source.

### 4.3 Icons

| Row | Leading element | Size / stroke | Source |
| --- | --- | --- | --- |
| Collection | `IconChevronRight`, rotates `rotateZ(90deg)` when expanded | `16px`, strokeWidth `2`, colour literal `rgb(160 160 160)` | `CollectionRow/index.jsx:591-601`; `CollectionRow/StyledWrapper.js:9-11` |
| Folder | `IconChevronRight` rotating `90deg` when expanded (`.rotate-90`) | `16px`, strokeWidth `2`, `rgb(160 160 160)` | `CollectionItemRow/index.jsx:784-795,297-299` |
| Request with examples | `IconChevronRight` in a `16px` box, rotates when examples expanded | `16px`, strokeWidth `2`, `rgb(160 160 160)` | `CollectionItemRow/index.jsx:796-808` |
| Request without examples | empty `16px` box is *not* rendered (only the icon slot of `RequestMethod`) | — | `CollectionItemRow/index.jsx:796-808` |
| App | `IconAppWindow` | `16px`, strokeWidth `1.5`, `mr-2` | `CollectionItemIcon/index.js:18-20` |
| Errored item | `IconAlertCircle` in `colors.text.danger` | `18px`, strokeWidth `1.5` | `CollectionItemIcon/index.js:6-8`; `CollectionItemIcon/StyledWrapper.js:7-9` |
| Loading item | `IconLoader2`, spinning | `18px`, strokeWidth `1.5` | `CollectionItemIcon/index.js:10-12` |
| Partially loaded item | `IconAlertTriangle` in `colors.text.yellow` | `18px`, strokeWidth `1.5` | `CollectionItemIcon/index.js:14-16`; `CollectionItemIcon/StyledWrapper.js:4-6` |
| Example | `ExampleIcon` in `sidebar.collection.item.example.iconColor` (`#666666` L / `#cccccc` D) | `16px` | `ExampleItem/index.jsx:252`; `ExampleItem/StyledWrapper.js:57-59` |
| Git ghost | `IconBrandGit` in `sidebar.muted` | `14px`, strokeWidth `2` | `GitRemoteCollectionRow/index.js:80-82`; `GitRemoteCollectionRow/StyledWrapper.js:14-20` |
| Collection spinner | `IconLoader2`, spinning, right-aligned | `18px`, strokeWidth `1.5` | `CollectionRow/index.jsx:606` |

### 4.4 Method badge usage

Sidebar request rows do **not** use `ui/MethodBadge`. They render
`CollectionItem/RequestMethod`, a text run inside a `min-width: 34px` slot:

| Aspect | Value | Source |
| --- | --- | --- |
| Font size | `font.size.xs` = `0.6875rem` (11 px) | `RequestMethod/StyledWrapper.js:4` |
| Transform | `uppercase` | `RequestMethod/index.js:48-50` |
| Colour per verb | `request.methods.get/post/put/delete/patch/options/head` | `RequestMethod/StyledWrapper.js:16-36` |
| gRPC / WS / GraphQL | `request.grpc` / `request.ws` / `request.gql`, text `grpc` / `ws` / `gql` | `RequestMethod/index.js:11-18`; `StyledWrapper.js:37-45` |
| Long verbs | truncated to first 3 characters (`method.substring(0, 3)`) | `RequestMethod/index.js:15-17` |
| Unknown verb | no colour class → inherits row text colour (**inference**; no fallback rule exists in this component) | `RequestMethod/index.js:20-34` |

Concrete colours (tokens `tokens.json` `request.methods` / `request.*`):

| Token | L | D |
| --- | --- | --- |
| `request.methods.get` | `#2e8a54` | `#73e89a` |
| `request.methods.post` | `#8d43b1` | `#79c8f6` |
| `request.methods.put` | `#c67a10` | `#f6ab79` |
| `request.methods.delete` | `#ce4f3b` | `#dd5540` |
| `request.methods.patch` | `#8d43b1` | `#f6ab79` |
| `request.methods.options` | `#2e8a87` | `#52e0c9` |
| `request.methods.head` | `#3089a6` | `#7ddff2` |
| `request.grpc` | `#3f48a6` | `#52e0c9` |
| `request.ws` | `#c67a10` | `#f6ab79` |
| `request.gql` | `#b83d7e` | `#e496dd` |

Note: `design-system.md` §1.5 lists DELETE dark as `#e06552`; `tokens.json` resolves it to
`#dd5540`. This file uses the resolved token value.

`ui/MethodBadge` exists (`ui/MethodBadge/index.js`, `StyledWrapper.js`) with `md` (52 px slot,
`font.size.xs`, weight `600`, uppercase, left aligned) and `sm` (9 px, monospace,
`0.05rem 0.25rem`, radius `3px`) variants, but it is **not** mounted by the sidebar.

### 4.5 Row states (exact values)

Applies to item rows (`CollectionItemRow/StyledWrapper.js:107-149`) and collection rows
(`SidebarRowStyles/index.js:13-38`, `Collections/StyledWrapper.js:8-48`).

| State | Background | Border | Other | Source |
| --- | --- | --- | --- | --- |
| Default | transparent | `3px` left transparent | menu hidden | `CollectionItemRow/StyledWrapper.js:28` |
| Hover | `sidebar.collection.item.hoverBg` (`#eaeaea` L / `#222224` D) | — | menu becomes visible; menu background forced transparent | `CollectionItemRow/StyledWrapper.js:108-117` |
| Drop-hover target | `sidebar.collection.item.hoverBg` | — | `.item-hovered` shares the hover rule | `:109-110` |
| Focus-within | same as hover | — | — | `SidebarRowStyles/index.js:13-21` |
| Keyboard focus | unchanged | `1px` top + `1px` bottom `sidebar.collection.item.focusBorder` (`#cccccc` L / `#444444` D), `outline: none` | — | `CollectionItemRow/StyledWrapper.js:145-149`; `SidebarRowStyles/index.js:34-38` |
| Selected in tab (active) | `sidebar.collection.item.bg` (`#eaeaea` L / `#26292b` D); hover forced to same | `3px` left `primary.solid` (`#d37f17` L / `#e4ae49` D) | - item: `.item-focused-in-tab`; - collection: `.collection-focused-in-tab` | `CollectionItemRow/StyledWrapper.js:133-143`; `Collections/StyledWrapper.js:17-20` |
| Collection selected (active) | `sidebar.collection.item.bg`; `box-shadow: inset 3px 0 0 0 primary.solid`; `padding-left: 8px`; left border width forced 0 | — | avoids mitered corners with drop borders | `Collections/StyledWrapper.js:24-28` |
| Multi-selected `collection-item-selected` | `rgba(primary.text, 0.1)` (hover `rgba(primary.text, 0.16)`) | `3px` left `primary.solid` | — | `Collections/StyledWrapper.js:8-20` (L `#c07a26`/D `#e7b85f` at 10 % / 16 %) |
| Multi-selected collection | `rgba(primary.text, 0.1)` / hover `0.16` | `inset 3px 0 0 0 primary.solid` | — | `Collections/StyledWrapper.js:8-28` |
| Keyboard focus + selected | `rgba(primary.text, 0.1)` (hover `0.16`) | as selected | `.collection-keyboard-focused` / `.item-keyboard-focused` | `Collections/StyledWrapper.js:31-38` |
| Tab focus + selected | `rgba(primary.text, 0.16)` (hover also `0.16`) | as selected | `.collection-focused-in-tab` / `.item-focused-in-tab` on selected row | `Collections/StyledWrapper.js:41-48` |
| Drag disabled | cursor `not-allowed` while pressed | — | `.drag-disabled:active` | `CollectionItemRow/StyledWrapper.js:35-37`; `CollectionRow/StyledWrapper.js:17-19` |
| Ghost (git) row | `sidebar.muted` text at `opacity: 0.7`; hover `sidebar.collection.item.hoverBg` at `opacity: 0.9` | — | — | `GitRemoteCollectionRow/StyledWrapper.js:11-12,33-42` |
| Empty CTA row | text `sidebar.muted`, `font.size.sm`, link `textLink` | — | `+ Add request` in `#346ab2` L / `#8bc2f9` D | `EmptyCtaRow/StyledWrapper.js:4-14` |

Additional legacy classes present but not used by the v1 states above:
`.item-target` background `#ccc3` and `.item-seperator` `3px` bar `#ccc3`
(`CollectionItemRow/StyledWrapper.js:119-131`).

### 4.6 Expand / collapse caret

| Behaviour | Detail | Source |
| --- | --- | --- |
| Collection | single click anywhere on the row expands a collapsed collection and opens the `collection-settings` tab; clicking the chevron only toggles, without opening the tab | `CollectionRow/index.jsx:149-177` |
| Folder | chevron toggles folder; chevron click and double-click are stopped so the row handler does not fire | `CollectionItemRow/index.jsx:367-382` |
| Request examples | chevron toggles the example list; only rendered when the request has examples and no search text | `CollectionItemRow/index.jsx:384-394,796` |
| Rotation | `.rotate-90 { transform: rotateZ(90deg) }` | `CollectionRow/StyledWrapper.js:9-11`; `CollectionItemRow/StyledWrapper.js:97-99` |
| Search override | while a search query is active, all collections/folders render expanded | `CollectionRow/index.jsx:142-147`; `flattenSidebarTree.js:86,214` |
| Collapse gesture | chevron double-click on collection is swallowed | `CollectionRow/index.jsx:191-194` |

### 4.7 Search (`Collections/CollectionSearch`)

| Aspect | Value | Source |
| --- | --- | --- |
| Trigger | header `Search requests` button, or `sidebarSearch` binding (`Ctrl/Cmd+F`) unless focus is inside `.CodeMirror` | `Sidebar/index.js:51-56`; `Sections/CollectionsSection/index.js:132-134` |
| Wrapper margins | `4px 10px 8px 10px` | `CollectionSearch/StyledWrapper.js:4` |
| Input height | `32px`, padding `0 32px 0 32px`, font `12px`, radius `6px`, background `sidebar.collection.item.hoverBg` | `:16-24` |
| Placeholder | `Search requests...` in `sidebar.muted` | `CollectionSearch/index.js:12`; `StyledWrapper.js:28-30` |
| Placeholder opacity | default input placeholder opacity is **not** applied here (no `opacity` rule) | `StyledWrapper.js:28-30` |
| Hover / focus | border `input.border` (`#cccccc` L / `#444444` D); focus also swaps background to `input.bg` (`#ffffff` L / `transparent` D) | `:32-39` |
| Search icon | `IconSearch` `14px` stroke `1.5`, absolute at `left: 10px`, `sidebar.muted` | `CollectionSearch/index.js:7`; `StyledWrapper.js:7-14` |
| Clear icon | `IconX` `14px` in an `18 × 18` box, radius `4px`, only when text non-empty | `CollectionSearch/index.js:22-26`; `StyledWrapper.js:42-61` |
| Debounce | `300 ms`, empty queries skip debounce | `Collections/index.js:24` |
| Matching | case-insensitive on lower-cased input; folders expand to show matching descendants | `CollectionSearch/index.js:20`; `flattenSidebarTree.js:85-88` |
| No results | collection-level rows are filtered out one by one; **no** empty-state message is rendered by the list | `CollectionRow/index.jsx:540-544`; `CollectionItemRow/index.jsx:701-711` |

Screenshot: `refs/sidebar_search_bar_open.png`.

### 4.8 Row badge: invalid files

| Aspect | Value | Source |
| --- | --- | --- |
| Trigger | `CountBadge variant="danger"` shown when `count > 0` | `InvalidFilesBadge/index.js:28-32` |
| Placement | after name, `flex-shrink-0 mx-1`, radius `9999px` | `InvalidFilesBadge/index.js:29`; `StyledWrapper.js:7-9` |
| Tooltip | `You have N invalid file(s).` + button `See here`, placed `bottom-start` | `InvalidFilesBadge/index.js:44-48` |
| Action | opens collection settings → overview and scrolls to the not-loaded list | `InvalidFilesBadge/index.js:20-26` |

---

## 5. Collections list (`Collections/`)

### 5.1 Empty state

When there are no entries the section body shows `CreateOrOpenCollection`
(`Collections/index.js:101-114`):

```text
No collections found.
Create or Open Collection.
```

- `No collections found.` then, on a second line (`mt-2`), `Create` (link) ` or ` `Open` (link) ` Collection.`
  (`CreateOrOpenCollection/index.js:37-40`).
- Text `text-xs`, centred, `colors.text.muted`; links underlined in `textLink`
  (`CreateOrOpenCollection/index.js:21-31`; `StyledWrapper.js:4`).
- `Create` starts the inline creator; `Open` sets `isOpeningCollection` (`:16-18,29`).

### 5.2 InlineCollectionCreator

Rendered in place of the list while `state.app.isCreatingCollection` is true
(`Collections/index.js:104-110,122-128`).

| Aspect | Value | Source |
| --- | --- | --- |
| Row | height `1.6rem`, padding-left `8px`, padding-right `4px`, gap `4px` | `InlineCollectionCreator/StyledWrapper.js:4-11` |
| Input wrapper | border `1px input.border`, radius `3px`, background `input.bg`; focus-within border `input.focusBorder` | `:13-25` |
| Input | `13px`, padding `1px 4px`, transparent, `color: text` | `:27-36` |
| Default value | `Untitled Collection` (replaced by an IPC-suggested unique name when a default location exists) | `InlineCollectionCreator/index.js:44-51,131` |
| Cog button | `20px` wide, `IconSettings` `13px` stroke `1.5`, opacity `0.5` → `1` on hover, tooltip `Advanced options` | `:139-151`; `StyledWrapper.js:38-54` |
| Save button | `20 × 20`, radius `3px`, `IconCheck` `14px` stroke `2`, `colors.text.green`, tooltip `Create` | `:154-163`; `StyledWrapper.js:63-81` |
| Cancel button | same box, `IconX` `14px` stroke `2`, `colors.text.danger`, tooltip `Cancel` | `:164-173`; `StyledWrapper.js:83-85` |
| Hover background of action buttons | `sidebar.collection.item.hoverBg` | `StyledWrapper.js:75-77` |

Validation toasts (verbatim): `Collection name is required`
(`InlineCollectionCreator/index.js:69`), `validateNameError(name)` (`:75`),
`Please set a default location in Preferences > General` (`:83`), success `Collection created!` (`:92`),
failure `An error occurred while creating the collection` + IPC message (`:95`).

Behaviours: `Enter` creates, `Escape` cancels, click outside attempts create (and cancels
silently when the name is empty/blank) (`:53-56,101-120`).

### 5.3 GitRemoteCollectionRow

A placeholder for a git-backed collection whose folder is missing; it is never expanded
(`flattenSidebarTree.js:295-310`).

| Aspect | Value | Source |
| --- | --- | --- |
| Height | `1.6rem`, padding-left `4px` | `GitRemoteCollectionRow/StyledWrapper.js:4-10` |
| Name padding | `6px` left, ellipsis | `:22-27` |
| Tooltip | `` `${entry.name} — click to clone from ${entry.remote}` `` | `GitRemoteCollectionRow/index.js:76` |
| Click | opens Clone Git Repository modal pre-filled with the remote | `:17,73-74` |
| Right click | opens the row menu | `:29-32` |

Row menu (`:34-53`): `Clone from Git` (`IconBrandGit`), `Copy Git URL` (`IconCopy`,
toast `Git URL copied` / `Failed to copy URL`), `Remove Git Remote` (`IconUnlink`).

---

## 6. Context menus — exact order

### 6.1 Collection row (`Collection/CollectionRow/index.jsx:386-538`)

| # | id | Icon | Label (verbatim) | Shortcut hint | Condition |
| --- | --- | --- | --- | --- | --- |
| 1 | `new-request` | `IconFilePlus` | `New Request` | `newRequest` (`Cmd/Ctrl+N`) | always |
| 2 | `new-folder` | `IconFolderPlus` | `New Folder` | — | always |
| 3 | `new-app` | `IconAppWindow` | `New App` | — | always |
| 4 | `run` | `IconPlayerPlay` | `Run` | — | always |
| 5 | `clone` | `IconCopy` | `Clone` | `cloneItem` (`Cmd/Ctrl+D`) | always |
| 6 | `sync-openapi` | `OpenAPISyncIcon` | `OpenAPI` | — | always |
| 7 | `paste` | `IconClipboard` | `Paste` | `pasteItem` (`Cmd/Ctrl+V`) | only when `hasCopiedItems` |
| 8 | `rename` | `IconEdit` | `Rename` | `renameItem` (`Cmd/Ctrl+R`) | always |
| 9 | `share` | `IconShare` | `Share` | — | always |
| 10 | `generate-docs` | `IconBook` | `Generate Docs` | — | always |
| 11 | `collapse` | `IconFoldDown` | `Collapse` | — | always |
| 12 | `show-in-folder` | `IconFolder` | `Reveal in Finder` / `Reveal in File Explorer` / `Reveal in File Manager` | — | always (platform-dependent) |
| 13 | `create-mock-server` | `IconServer` | `Create Mock server` + `StatusBadge` `Beta` | — | beta `MOCK_SERVER` only |
| — | `divider-1` | — | divider | — | always |
| 14 | `settings` | `IconSettings` | `Settings` | — | always |
| 15 | `terminal` | `IconTerminal2` | `Open in Terminal` | `openTerminal` (`Cmd/Ctrl+T`) | always |
| 16 | `move-to-workspace` | `IconFileArrowRight` | `Move into Workspace` | — | only when the collection path is outside the workspace |
| 17 | `remove` | `IconX` | `Remove` | — | always |

Reveal label resolution: `utils/common/platform.js` (`getRevealInFolderLabel`).

### 6.2 Folder row (`Collection/CollectionItem/CollectionItemRow/index.jsx:412-568`)

| # | id | Icon | Label (verbatim) | Shortcut hint |
| --- | --- | --- | --- | --- |
| 1 | `new-request` | `IconFilePlus` | `New Request` | `newRequest` |
| 2 | `new-folder` | `IconFolderPlus` | `New Folder` | — |
| 3 | `new-app` | `IconAppWindow` | `New App` | — |
| 4 | `run` | `IconPlayerPlay` | `Run` | — |
| 5 | `clone` | `IconCopy` | `Clone` | `cloneItem` |
| 6 | `copy` | `IconCopy` | `Copy` | `copyItem` (`Cmd/Ctrl+C`) |
| 7 | `paste` | `IconClipboard` | `Paste` | `pasteItem` — only for folders with copied items |
| 8 | `rename` | `IconEdit` | `Rename` | `renameItem` |
| 9 | `show-in-folder` | `IconFolder` | platform reveal label | — |
| 10 | `ignore` | `IconEyeOff` | `Ignore` | — |
| — | `separator-1` | — | divider | — |
| 11 | `info` | `IconInfoCircle` | `Info` | — |
| 12 | `settings` | `IconSettings` | `Settings` | — |
| 13 | `open-terminal` | `IconTerminal2` | `Open in Terminal` | `openTerminal` |
| 14 | `delete` | `IconTrash` | `Delete` (danger class) | — |

Items 1–4 exist only for folders; `clone` only when `isCloneable` (folder, request or app).

### 6.3 Request row (`CollectionItemRow/index.jsx:445-565`)

| # | id | Icon | Label (verbatim) | Shortcut hint | Condition |
| --- | --- | --- | --- | --- | --- |
| 1 | `clone` | `IconCopy` | `Clone` | `cloneItem` | `isCloneable` |
| 2 | `copy` | `IconCopy` | `Copy` | `copyItem` | always |
| 3 | `rename` | `IconEdit` | `Rename` | `renameItem` | always |
| 4 | `run` | `IconPlayerPlay` | `Run` | — | non-HTTP/GraphQL requests (gRPC, WS) |
| 5 | `generate-code` | `IconCode` | `Generate Code` | — | HTTP and GraphQL requests |
| 6 | `create-example` | `ExampleIcon` | `Create Example` | — | HTTP requests |
| 7 | `show-in-folder` | `IconFolder` | platform reveal label | — | always |
| — | `separator-1` | — | divider | — | always |
| 8 | `info` | `IconInfoCircle` | `Info` | — | always |
| 9 | `delete` | `IconTrash` | `Delete` (danger class) | — | always |

`Delete` styling: `colors.text.danger`; hover `colors.bg.danger` background with white text
(`CollectionItemRow/StyledWrapper.js:156-162`).

### 6.4 Example row (`CollectionItem/ExampleItem/index.jsx:171-204`)

`Rename` (`IconEdit`), `Clone` (`IconCopy`), `Generate Code` (`IconCode`), divider, `Delete` (`IconTrash`).

### 6.5 Menu surface

`Dropdown` popover: `min-width: 160px`, `font.size.sm`, `dropdown.bg` (`#ffffff` L / `#222224` D),
`dropdown.border`, radius `border.radius.base` (`6px`), padding `0.25rem`
(`components/Dropdown/StyledWrapper.js:4-21`). Sidebar menus use `placement="bottom-start"`
and render into the accordion container (`CollectionRow/index.jsx:611-617`;
`CollectionItemRow/index.jsx:820-826`).

Multi-selection replaces the row menu with `BulkActionsMenu`
(`CollectionRow/index.jsx:196-210`; `CollectionItemRow/index.jsx:397-407`;
`Collections/BulkActionsMenu/index.js`).

---

## 7. Drag and drop visuals

| Aspect | Value | Source |
| --- | --- | --- |
| Indicator line / border | `dragAndDrop.border` (`#8b8b8b` L / `#666666` D), `dragAndDrop.borderStyle` `2px solid`, transition `all 0.1s ease` | tokens `dragAndDrop.*`; `CollectionItemRow/StyledWrapper.js:40-50` |
| Line above target | row `::before` `2px` full-width line, `opacity: 1` | `CollectionItemRow/StyledWrapper.js:52-54,70-75` |
| Line below target | row `::after` `2px` full-width line | `:56-58,77-82` |
| Drop-inside target | background `dragAndDrop.hoverBg` (`rgba(139,139,139,.05)` L / `rgba(102,102,102,.08)` D) plus a `2px` inset border | `:61-68,85-95` |
| Collection-inside target | border `dragAndDrop.border`, background `dragAndDrop.hoverBg`, transition | `CollectionRow/StyledWrapper.js:27-31` |
| Collection-above indicator | `border-top` `2px` line, `margin-top: -2px`, transparent background | `CollectionRow/StyledWrapper.js:33-39` |
| Collection-below indicator | `border-bottom` `2px` line, `margin-bottom: -2px` | `CollectionRow/StyledWrapper.js:41-47` |
| Collection hovered line | top + bottom `2px` (`border-bottom` transparent) | `CollectionRow/StyledWrapper.js:12-15` |
| Drag preview | floating label `p-2`, inner box `border rounded-md px-2 py-1`, background `sidebar.collection.item.hoverBg`, border `border.border2`, icon `16px`; multi-drag label `"N folders, M requests and K apps"` with `and` before the last part | `CollectionItemDragPreview/index.js:23-52,98-104`; `StyledWrapper.js:4-7` |
| Drag leaving rows | getEmptyImage preview hides the native ghost | `CollectionRow/index.jsx:372`; `CollectionItemRow/index.jsx:289` |
| Disabled drag cursor | `dnd-blocked-cursor` class after a `4px` move threshold | `utils/dragBlockedCursor.js:9-31` |
| Blocked combinations | examples never drag alone; collections cannot be dragged with their own children; examples can drag only together with their parent request | `Collections/index.js:68-78`; `ExampleItem/index.jsx:53-63` |

---

## 8. Keyboard and selection

| Input | Effect | Source |
| --- | --- | --- |
| Click | select row; collection click also opens settings tab; request/folder click opens its tab | `useSidebarSelectionClick/index.js:12-31`; `CollectionRow/index.jsx:149-177` |
| `Cmd`/`Ctrl` + click | toggle multi-selection | `useSidebarSelectionClick/index.js:12-20` |
| `Shift` + click | range selection over visible rows | `:22-27` |
| Double click | make the tab permanent (leave preview state) | `CollectionRow/index.jsx:179-181`; `CollectionItemRow/index.jsx:574-576` |
| Click on empty list background | clear selection | `Collections/index.js:95-99` |
| `Tab` focus | row gains keyboard-focus border, sets `focusedSidebarPath` | `CollectionRow/index.jsx:267-275`; `CollectionItemRow/index.jsx:690-699` |
| `Enter` / `Space` on section icon | toggle section | `SidebarSection/index.js:49-53` |

Row-level shortcuts, active only while that row has keyboard focus
(`CollectionRow/index.jsx:246-265`; `CollectionItemRow/index.jsx:145-169`):

| Action | Mac | Windows/Linux | Scope |
| --- | --- | --- | --- |
| `newRequest` | `⌘ N` | `Ctrl N` | collection row; folder row only |
| `cloneItem` | `⌘ D` | `Ctrl D` | collection row, item row |
| `renameItem` | `⌘ R` | `Ctrl R` | collection row, item row |
| `copyItem` | `⌘ C` | `Ctrl C` | item row |
| `pasteItem` | `⌘ V` | `Ctrl V` | collection row, item row |

Default bindings: `providers/Hotkeys/keyMappings.js:28-60`. Users may override them
(`getMergedKeyBindings`, `keyMappings.js:213-236`).

Global/section shortcuts: `sidebarSearch` `⌘/Ctrl+F` (`keyMappings.js:31`),
`importCollection` `⌘/Ctrl+O` (`:56`), `collapseSidebar` `⌘/Ctrl+\` (`:36`,
handler `providers/Hotkeys/index.js:267-278`).

---

## 9. Copy reference

| Location | Text (verbatim) | Source |
| --- | --- | --- |
| Section title | `Collections` | `Sections/CollectionsSection/index.js:407` |
| Section title | `API Specs` | `Sections/ApiSpecsSection/index.js:68` |
| Section title | `Mock Servers` | `Sections/MockServersSection/index.js:28` |
| Header tooltip | `Search requests` | `Sections/CollectionsSection/index.js:298` |
| Header tooltip | `Add new collection` | `:310` |
| Header tooltip | `More actions` | `:322` |
| Header tooltip | `Add new API Spec` | `Sections/ApiSpecsSection/index.js:51` |
| Header tooltip | `Create mock server` | `Sections/MockServersSection/index.js:13` |
| Empty list | `No collections found.` | `CreateOrOpenCollection/index.js:37` |
| Empty list | `Create or Open Collection.` | `:39` |
| Search placeholder | `Search requests...` | `CollectionSearch/index.js:12` |
| Inline creator default | `Untitled Collection` | `InlineCollectionCreator/index.js:131` |
| Inline creator tooltips | `Advanced options`, `Create`, `Cancel` | `:147,159,169` |
| Empty folder CTA | `+ Add request` | `EmptyCtaRow/index.jsx:34` |
| Git ghost tooltip | `{name} — click to clone from {remote}` | `GitRemoteCollectionRow/index.js:76` |
| Invalid files tooltip | `You have {n} invalid file(s).` + `See here` | `InvalidFilesBadge/index.js:44-48` |

No sidebar copy is routed through `i18n`: `src/i18n/translation/en.json` contains only
`COMMON`, `WELCOME`, `SUPPORT` keys (34 lines, `i18n/translation/en.json:1-34`) and the only
`useTranslation` consumer is `components/Preferences/Support`. All labels above are literal
strings in the cited source lines.

---

## 10. Interactions

1. Dragging the right edge expands the sidebar between 220 and 600 px; the width is committed
   on mouse-up and stored in `localStorage` (`components/Sidebar/index.js:66-105`).
2. Clicking a section header toggles it; only one expanded section fills the body, several
   expanded sections split the body equally (`SidebarContent.js:26-51`; `Collections/index.js` styles in `StyledWrapper.js`).
3. Clicking any header action auto-expands the section first (`SidebarSection/index.js:66-78`).
4. `+ Add request` opens the HTTP / GraphQL / gRPC / WebSocket menu
   (`utils/collections/emptyStateRequest.js:53-81`).
5. `Remove` / `Close all` open `RemoveCollections`, which blocks on unsaved drafts
   (`Collections/Collection/RemoveCollections/index.js:27-31`).
6. Dragging a request onto another request shows above/inside/below indicators based on the
   pointer position (`CollectionItemRow/index.jsx:192-198,305-314`).
7. Dragging a collection over a collection shows only the above line; dropping an item over a
   collection shows the full highlight (`CollectionRow/index.jsx:297-308,374-384`).
8. After a successful move, the multi-selection is cleared (`CollectionRow/index.jsx:337-339,356-358`).
9. The active tab scrolls into view with `behavior: 'smooth'` when a sidebar row is revealed
   (`hooks/useSidebarAutoReveal` used at `Collections/index.js:90`).

---

## 11. Open Questions

1. **Indent geometry.** Only the `16px` cell width and the `1px` guide colour are asserted.
   The exact alignment of the caret, method slot and text at depth ≥ 1 depends on flex
   sizing (`CollectionItemRow/index.jsx:764-817`); a pixel-perfect Go layout needs a visual
   check against `refs/side_bar_bruno.png`.
2. **Screenshot vs source: workspace switcher.** `refs/workspace_switcher_bruno.png` shows a
   `My Workspace` menu with `Create workspace`, `Open workspace`, `Import workspace`,
   `Manage workspaces`, but the current `RequestTabs/CollectionHeader` switcher renders
   `Workspace` and `Collections` groups only (`CollectionHeader/index.js:603-652`). The
   workspace switcher belongs to `screens/workspace-switcher.md`; flagged here because the
   capture is taken over the sidebar.
3. **Desktop capture vs source header.** `refs/side_bar_bruno.png` shows top tabs plus a
   `Mock Servers` section; source renders tabs per collection ("Moved to post mvp",
   `RequestTabs/index.js:139-144,191-198`) and gates Mock Servers behind a beta flag
   (`Sidebar/index.js:20,33-38`). The capture is therefore from an older/newer build than the
   pinned commit; this file follows source.
4. **`MethodBadge` in the sidebar.** The component exists but is unused by the sidebar
   (`ui/MethodBadge/index.js` has no sidebar import). If the Go port wants the badge look,
   the 52 px slot and `font.size.xs`/weight `600` are available; otherwise use the
   `min-width: 34px` text slot described in §4.4.
5. **Delete dark token.** `design-system.md` §1.5 says DELETE dark `#e06552`, `tokens.json`
   resolves `#dd5540`. Source of truth is the palette (`themes/dark/dark.js`); documentation
   should be reconciled.
6. **Light theme hover equals selection.** In light both `sidebar.collection.item.bg` and
   `sidebar.collection.item.hoverBg` resolve to `#eaeaea`, so hover and active are visually
   identical except for the `3px` accent rail. This is source-accurate, not a spec bug.
