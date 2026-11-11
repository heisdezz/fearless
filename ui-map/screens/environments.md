# Screen — Environments

All paths are relative to `bruno/packages/bruno-app/src/` unless prefixed with
`design-system.md`.

Status: VERIFIED against source. Anything not confirmable from the repo is
marked **UNVERIFIED** inline and repeated in §13 Open Questions.

Scope:

- `components/EnvironmentVariablesTable/**` (8 files)
- `components/VariablesEditor/**` (13 files) — the read-only "Variables" viewer tab
- `components/VarValueCell` — compact value cell wrapper
- `components/Environments/**` (58 files) — `EnvironmentSelector`,
  `EnvironmentSettings` (`EnvironmentList`, `EnvironmentDetails`), create /
  copy / delete / import / export modals, `.env` file editor, global environments

Screenshot: `refs/workspace_switcher_bruno.png` is the **workspace switcher**, not
an environment surface — it is cited only because the brief listed it; the
`Global` tab of the selector is where global environments live, and those are
owned by the workspace (`components/WorkspaceHome/WorkspaceEnvironments`,
`components/Environments/GlobalEnvironmentSettings/index.js:11`).

No screenshot in `refs/` shows the environment editor; every table below is
source-derived.

---

## 1. Three distinct surfaces

| Surface                     | Component                          | When it appears                                                                     | Read/write                                                 |
| --------------------------- | ---------------------------------- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| 1. Selector dropdown        | `Environments/EnvironmentSelector` | Always mounted in the collection toolbar                                            | writes the _active_ env selection                          |
| 2. Environment settings tab | `Environments/EnvironmentSettings` | Tab type `environment-settings` (uid `` `${collectionUid}-environment-settings` ``) | edits env names, colours, variables, secrets, `.env` files |
| 3. Variables viewer tab     | `VariablesEditor`                  | Tab type `variables` (read-only view of currently resolved runtime + env variables) | read-only                                                  |

Sources: `EnvironmentSelector/index.js:271-353`; `EnvironmentSettings/index.js:8-54`;
`VariablesEditor/index.js:198-281`; tab types from
`EnvironmentSelector/index.js:231-241,261-269` and
`CollectionSettings/Overview/Info/index.js:107-130`.

---

## 2. Environment selector (trigger)

`Environments/EnvironmentSelector/index.js`.

### 2.1 Layout

```text
 Toolbar:  …  [ 🗄 Prod │ 🌐 Shared ▾ ]     ← both environments
                ↑ collection badge  ↑ global badge + caret
 Toolbar:  …  [ 🗄 Staging ▾ ]               ← only collection env
 Toolbar:  …  [ No Environment ▾ ]           ← nothing selected (dashed)
```

### 2.2 Trigger variants — `:108-182`

| State           | Markup                                                                                                                           | Source               |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------- |
| None            | `<span class="env-text-inactive">No Environment</span>` + `IconCaretDown` 12 px; trigger gets `.no-environments` (dashed border) | `:112-123`           |
| Collection only | env badge: `IconDatabase` 14 px + name + caret (12 px)                                                                           | `:126-140`           |
| Global only     | `IconWorld` 14 px + name + caret                                                                                                 | `:143-157`           |
| Both            | collection badge (left radius `0.3rem 0 0 0.3rem`) + 1 px separator + global badge (right radius) + caret                        | `:160-181`, `:43-65` |

Badge background = env colour at **12 % alpha**: `transparentize(1 - 0.12, color)`
(`:31-38`); with no colour → `transparent`. Badge padding
`0.25rem 0.5rem` (or `0.25rem 0.3rem 0.25rem 0.5rem` for the outer half when
only one env) (`:54-58`). Icon and text inherit the env colour when set
(`:85-99`).

Trigger chrome (`EnvironmentSelector/StyledWrapper.js:4-54`):

| Property       | Value                                                                                                                                    |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Height         | `24px`                                                                                                                                   |
| Padding        | `0.25rem 0.3rem 0.25rem 0.5rem`                                                                                                          |
| Radius         | `border.radius.base` (6px)                                                                                                               |
| Background     | `app.collection.toolbar.environmentSelector.bg` (`#ffffff` L / `#1a1a1a` D)                                                              |
| Border         | `1px solid …environmentSelector.border` (`#e5e5e5` L / `#444` D)                                                                         |
| Hover          | border → `…hoverBorder` (`#cccccc` L / `#666` D), bg → `…hoverBg`                                                                        |
| Caret          | `…caret` (`#b0b0b0` L / `#aaa` D), `fill` same, `margin-left .25rem`                                                                     |
| Icon           | `…icon` (`#c07a26` L / `#e7b85f` D), `margin-right .25rem`                                                                               |
| Text           | `…text` (`#343434` L / `#cccccc` D)                                                                                                      |
| Separator      | `…separator` (`#e5e5e5` L / `#444` D)                                                                                                    |
| No-environment | text `#838383` L / `#aaa` D, bg `#ffffff` L / `#1a1a1a` D, **dashed** border `#cccccc` L / `#444` D, hover border `#b0b0b0` L / `#666` D |
| Inactive text  | `colors.text.muted`, `font.size.sm` (12px)                                                                                               |
| Transition     | `all 0.15s ease`                                                                                                                         |

Token values: design-system §6.6 / `tokens.json` →
`app.collection.toolbar.environmentSelector.*`.

Name truncation: `max-w-24` + tooltip after 1 s when the name is ≥ 7 chars
(`:90-100`).

### 2.3 Dropdown — `:271-318`

`Dropdown` (tippy) `placement="bottom-end"`, `popperOptions={{ strategy:'fixed' }}`,
trigger `click` (`components/Dropdown/index.js:36-38,57`).

Panel (`EnvironmentSelector/StyledWrapper.js:57-225`):

| Property    | Value                                                                            | Source                                |
| ----------- | -------------------------------------------------------------------------------- | ------------------------------------- |
| Width       | `max(longest env name length × 8px, 12rem)` — JS-computed `width` prop           | `:70-77,58`, `StyledWrapper.js:57-60` |
| Min / max   | `min-width: 12rem`, `max-width: 650px !important`                                | `StyledWrapper.js:59-60`              |
| Height      | `min-height: 15.5rem`, `max-height: 75vh`                                        | `:61-62`                              |
| Font        | `font.size.base`                                                                 | `:63`                                 |
| List scroll | `max-height: calc(75vh - 8rem)`, `padding-bottom: 2.625rem` (room for Configure) | `:124-129`                            |

Tab headers (`:282-301`): two buttons, `font.size.sm`, idle
`colors.text.sub0`, active `tabs.active.color` with 2 px bottom border
`tabs.active.border`; `margin-right: 1.25rem`; icon 16 px.

| #   | id           | Label        | Icon                 |
| --- | ------------ | ------------ | -------------------- |
| 1   | `collection` | `Collection` | `IconDatabase` 16 px |
| 2   | `global`     | `Global`     | `IconWorld` 16 px    |

### 2.4 List content — `EnvironmentListContent/index.js`

**When the active tab has ≥ 1 environment** (`:66-132`):

| Element            | Verbatim / detail                                                                                                                                   | Source                               |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| Search             | `SearchInput` placeholder **`Search environments...`**, height 30 px, `text-xs`, icon 13 px, sticky at top (`padding: 0 4px 6px`, bg `dropdown.bg`) | `:69-83`, `StyledWrapper.js:131-140` |
| No Environment row | leading spacer `w-2` (aligned with the 8 px colour dot) then `No Environment`; active when no env selected                                          | `:84-91`                             |
| Env row            | `ColorBadge size={8}` + name (`truncate`); active row `.dropdown-item-active`; tooltip (right, 1 s, `max-width:200px`) when name ≥ 90 chars         | `:108-121`                           |
| No results         | italic `No results found`, centred, `text-xs opacity-50`                                                                                            | `:103-106`                           |
| Configure footer   | pinned bottom bar, top border `dropdown.separator`, button `IconSettings` 16 px + `Configure`                                                       | `:125-130`, `StyledWrapper.js:68-90` |

`dropdown-item` chrome comes from `components/Dropdown/StyledWrapper.js:56-71`
(`gap .5rem`, `padding .275rem .625rem`, radius 6px, `font.size.sm`).

Selection toasts (`EnvironmentSelector/index.js:221-228`):
`Environment changed to {name}` / `No Environments are active now` /
`An error occurred while selecting the environment`.

**When the active tab has zero environments** (`:133-148`):

| Element                   | Verbatim                                                                                                                                                                 | Source                                 |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------- |
| Title (`h3`, 16 px / 500) | `Ready to get started?`                                                                                                                                                  | `:135`, `StyledWrapper.js:158-164`     |
| Body                      | collection: `Create your first environment to begin working with your collection.` · global: `Create your first global environment to begin working across collections.` | `:23-26,136`                           |
| Buttons                   | `Create` (`IconPlus` 16 px) and `Import` (`IconDownload` 16 px); full width, `1px solid text`, transparent, `padding .5rem 1rem`, radius .375rem                         | `:137-146`, `StyledWrapper.js:182-204` |

Empty-state container `max-width:20rem`, `min-height:12.5rem` (`StyledWrapper.js:147-157`).

Keyboard: typing any printable character focuses the search input; `Escape`
clears the query then blurs (`:20-54`). Query resets when the dropdown closes
(`EnvironmentSelector/index.js:276`).

Create / Import from the dropdown open modals and then **open the settings tab**
(`:321-351`).

---

## 3. Environment settings tab

`Environments/EnvironmentSettings/index.js` → `EnvironmentList/index.js` →
`EnvironmentDetails/index.js` → `EnvironmentVariables` →
`components/EnvironmentVariablesTable`.

### 3.1 Layout

```text
┌ Environments ───────────────────────────────────────────────────────────┐
│ ┌ sidebar 240px ──────────┐ ┌ details ────────────────────────────────┐ │
│ │ ▸ Environments      + ↓ ↑│ │ 🟥 Staging  [colour]      Inherits ☰ ⤓ ✎ ⧉ 🗑│ │
│ │   [🔍 Search environments...]│ │                         ╰ Save All │ │
│ │   ( ) No …  (o) Staging   │ │ Variables⁽⁴⁾  Secrets⁽¹⁾        🔍      │ │
│ │   ( ) Prod                │ │ ┌─────────────────────────────────────┐ │ │
│ │ ▸ .env Files (2)     +    │ │ │☐ Inherited Variables (2)          ▾ │ │ │
│ │                           │ │ │☑ base_url   …   Description  ↗      │ │ │
│ │                           │ │ │☐ Variables (4)                    ▾ │ │ │
│ │                           │ │ │☑ token      …   Description     👁 🗑│ │ │
│ │                           │ │ └─────────────────────────────────────┘ │ │
│ │                           │ │ [ Save ] [ Reset ]                       │ │
│ └───────────────────────────┘ └─────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

Container: `height/width 100%`, two columns (`EnvironmentList/StyledWrapper.js:6-15`).

| Region         | Value                                                                                                    | Source                                           |
| -------------- | -------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| Sidebar        | `width:240px`, fixed; inner `.sections-container` `padding:8px`, `border-right:1px solid border.border0` | `EnvironmentList/StyledWrapper.js:28-30,109-115` |
| Section header | `padding-left:6px`, `padding-right:3px`, `padding-block:4px`, radius 6px                                 | `:118-123`                                       |
| Row            | `height:1.6rem`, `padding:0 8px`, `font-size:13px`, radius 6px                                           | `:155-167`                                       |
| Detail header  | `padding: 9px 20px 8px 20px`, flex, space-between                                                        | `EnvironmentDetails/StyledWrapper.js:10-17`      |
| Detail title   | `font.size.base`, weight 500                                                                             | `:18-23`                                         |
| Detail body    | `padding: 0 20px 20px 20px`                                                                              | `:176`                                           |

### 3.2 Sidebar — `EnvironmentList/index.js:703-1021`

Two `CollapsibleSection`s (`components/Environments/CollapsibleSection`):
chevron `IconChevronRight` 14 px (rotates when expanded), title, optional count
badge, action buttons on the right (`CollapsibleSection/index.js:19-41`).

#### Section 1 — `Environments` (`expanded` by default)

| Action button (`title`) | Icon                 | Effect            | Source     |
| ----------------------- | -------------------- | ----------------- | ---------- |
| `Create environment`    | `IconPlus` 14 px     | inline create row | `:716-727` |
| `Import environment`    | `IconDownload` 14 px | import modal      | `:728-739` |
| `Export environment`    | `IconUpload` 14 px   | export modal      | `:740-750` |

Search field (`:754-779`): `IconSearch` 13 px + input placeholder
**`Search environments...`**, height 28 px, radius 6px, `1px solid border1`,
focus border `colors.accent`; clear button (`title="Clear search"`,
`IconX` 13 px) when non-empty.

Environment row (`:786-871`):

| Part            | Detail                                                                                                                                 | Source     |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Colour dot      | `ColorBadge size={8}`                                                                                                                  | `:839`     |
| Name            | `.environment-name`                                                                                                                    | `:840`     |
| Row menu        | `RowActionsMenu` (`aria-label="More actions"`), hidden for multi-select                                                                | `:842-852` |
| Active marker   | checkmark (`title="Active environment"`) when active; otherwise an activate button (`title="Activate environment"`, `IconCheck` 16 px) | `:853-866` |
| Activated toast | `Environment "{name}" activated` / `Failed to activate environment`                                                                    | `:215-218` |
| States          | `active` (open in detail), `activated` (collection's active env), `is-selected` (multi-select), `renaming`                             | `:793-798` |

Row interactions:

- **single click** → open in the detail pane (blocked while unsaved changes are
  pending → unsaved-changes modal) (`:173-185`).
- **double click** → inline rename (`:201-209`).
- **right click** → context menu (`:800`).
- **multi-select + Ctrl/Cmd+A** → select all; **Delete/Backspace** → delete
  (`useEnvironmentShortcuts`, `Common/EnvironmentSelection/useEnvironmentShortcuts/index.js:37-47`);
  `Escape` clears selection (`:28`).

Inline rename (`:803-836`): input `.environment-name-input` + save (`title="Save"`,
`IconCheck` 14 px) and cancel (`title="Cancel"`, `IconX` 14 px). `Enter` saves,
`Escape` cancels, clicking outside saves (`:300-316,351`). Success toast
`Environment renamed!` (`:335`).

Inline create (`:874-910`): placeholder **`Environment name...`**, same
save/cancel buttons (`Save` has `data-testid="env-create-save"`), `Enter` creates,
`Escape` cancels, outside click cancels (`:261,300-316`). Success toast
`Environment created!` (`:266`).

Create-from-search row (`:920-941`): when the search text is non-empty and no
row matches, a button **`Create "{searchText}"`** (plus `IconPlus` 14 px) is
shown. Errors render under it (`:912,935-939`).

Empty states:

- `No environments` (`data-testid="collection-env-empty"`) when the filtered
  list is empty and not creating (`:914-918`).
- `No .env files` inside the `.env` section (`:1014-1018`).
- Whole detail pane `No Environments` + `Create Environment` / `Import Environment`
  buttons when the collection has no environments at all (`:640-653`).

Name validation (`validateEnvironmentName`, `:222-240` and
`EnvironmentDetails/index.js:64-90`):

| Condition                   | Message                                                                                                                                                                                                                |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Empty                       | `Name is required`                                                                                                                                                                                                     |
| < 1 char                    | `Must be at least 1 character` (details pane only)                                                                                                                                                                     |
| > 255 chars                 | `Must be 255 characters or less` (details pane only)                                                                                                                                                                   |
| Invalid filename characters | `validateNameError`, i.e. `Name cannot be empty.` / `Name cannot exceed 255 characters.` / `Name cannot be a reserved device name.` / _special characters_ message (`packages/bruno-common/src/utils/naming.ts:50-60`) |
| Duplicate                   | `Environment already exists`                                                                                                                                                                                           |

#### Section 2 — `.env Files` (collapsed by default, badge = count)

| Part          | Detail                                                                                                                                      | Source                      |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Title         | `.env Files` + count badge                                                                                                                  | `:945-950`                  |
| Action        | `Create .env file` (`IconPlus` 14 px)                                                                                                       | `:951-960`                  |
| File row      | filename only                                                                                                                               | `:963-974`                  |
| Inline create | input prefilled **`.env`**; characters cannot be deleted before index 4; `Enter` create, `Escape` cancel, `Backspace` blocked at offset ≤ 4 | `:383-398,464-477,976-1010` |
| Validation    | `Name is required`, `File name must start with .env`, `Invalid file name`, `File already exists`                                            | `:408-428`                  |
| Toasts        | `{name} file created!`, `Failed to create .env file`, `{name} file deleted!`, `Failed to delete .env file`                                  | `:439,448,482,498`          |

### 3.3 Detail header — `EnvironmentDetails/index.js:204-269`

| Part          | Detail                                                                                                                                                                                         | Source                                                          |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| Title         | env name, `data-testid="env-details-title"`                                                                                                                                                    | `:243`                                                          |
| Colour        | `ColorPicker` (`title="Change color"`); presets `#CE4F3B`, `#2E8A54`, `#346AB2`, `#C77A0F`, `#B83D7F`, `#8D44B2`; also `No color` and `Custom color` (range `#D85D43 → #F4BB91 → … → #B491E5`) | `components/ColorPicker/index.js:9-18,108-159`                  |
| Rename        | `ActionIcon label="Rename"` → replaces title with `.title-input` (`font-size:15px`, weight 500, padding `4px 8px`, radius 5px, bg `sidebar.collection.item.hoverBg`) plus save/cancel          | `:208-240,257-258`; `EnvironmentDetails/StyledWrapper.js:22-50` |
| Inherits from | `InheritsFrom` — menu `Inherit variables from`, items `No Environment` + inheritable envs; active label `Inherits from: {name}`; `ActionIcon label="Inherit variables from"`                   | `:250-255`; `Common/InheritsFrom/index.js:23-62`                |
| Save All      | `ActionIcon label="Save All"` (`IconDeviceFloppy` 15 px) — dispatches `environment-save-all`                                                                                                   | `:256-258,191-193`                                              |
| Copy          | `ActionIcon label="Copy"` → `CopyEnvironment` modal                                                                                                                                            | `:262-264`                                                      |
| Delete        | `ActionIcon label="Delete"` (`IconTrash` 15 px, hover danger) → delete modal                                                                                                                   | `:265-267`                                                      |

Inheritance toasts (`:176-189`): `Inheriting variables from {name}` /
`Stopped inheriting variables` / `An error occurred while saving the inherited environment`.

Inheritance warnings (`Common/EnvironmentInheritanceWarning/index.js:8-38`),
rendered directly under the header (`EnvironmentDetails/index.js:271`):

| Case           | Verbatim                                          |
| -------------- | ------------------------------------------------- |
| Missing parent | `Referenced parent environment not found: {name}` |
| Cycle          | `Circular inheritance: A → B → A`                 |

### 3.4 Variables / Secrets tabs — `hooks/useEnvironmentTabs/index.js:37-40`

| #   | key         | Label       | Count superscript                               |
| --- | ----------- | ----------- | ----------------------------------------------- |
| 1   | `variables` | `Variables` | own non-secret + inherited non-secret, when > 0 |
| 2   | `secrets`   | `Secrets`   | own secret + inherited secret, when > 0         |

The superscript gets class `unsaved` when that tab has un-saved draft changes
(`:24-35`; style in `EnvironmentDetails/StyledWrapper.js:106`). Selected tab is
stored in tab state (`EnvironmentDetails/index.js:30-31`); default
`variables` (`EnvironmentList/index.js:56`).

Search (right of the tabs, `EnvironmentDetails/index.js:278-317`): collapsed to
an `ActionIcon label="Search"`; expands to an input with placeholder
**`Search variables...`** or **`Search secrets...`** (`:286`), clear button
`title="Clear search"` (`:297-307`); blurs + collapses when emptied
(`:166-170`). 300 ms debounce before filtering (`EnvironmentList/index.js:88`).

---

## 4. EnvironmentVariablesTable (shared variables grid)

`components/EnvironmentVariablesTable/index.js`; styles `…/StyledWrapper.js`.
Used by the environment settings tab (collection scope).

### 4.1 Layout

```text
┌ table-container (radius 8px, 1px border0, height animated) ──────────────┐
│   [x]  Name  ⇅⇩     Value         Description            (25/40 px cols) │
│ ┌───────────────────────────────────────────────────────────────────────┐│
│ │ ☐  Inherited Variables (2)                                       ▾   ││
│ │ ☑  base_url             https://…      desc            ↗           ││
│ │ ☐  Variables (4)                                                 ▾   ││
│ │⠿☑  token                ••••••   👁   desc                🗑          ││
│ │ ⠿☑  host                 {{x}}         desc                🗑          ││
│ │    [ new blank row ]                                                 ││
│ └───────────────────────────────────────────────────────────────────────┘│
│ [ Save ]  [ Reset ]                                                      │
└──────────────────────────────────────────────────────────────────────────┘
```

| Property              | Value                                                                                                                | Source                                             |
| --------------------- | -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Container             | `overflow-y:auto`, radius **8px**, `1px solid border.border0`                                                        | `EnvironmentVariablesTable/StyledWrapper.js:16-21` |
| Height transition     | `height 75ms cubic-bezier(0,1.12,.84,.64)` (disabled while measuring)                                                | `:20-25`                                           |
| Table font            | `12px`; cell padding `2px 10px`                                                                                      | `:27-36`                                           |
| Checkbox column       | `25px`                                                                                                               | `:3,37-40`                                         |
| Actions column        | `40px`                                                                                                               | `:4,42-44`                                         |
| Default column widths | `[20, 45, 35]` percent                                                                                               | `index.js:127-128`                                 |
| Min column width      | `80`                                                                                                                 | `:47`                                              |
| Row height            | `35` (`MIN_ROW_HEIGHT`)                                                                                              | `:48`                                              |
| Min table height      | `70` (`MIN_H = 35*2`)                                                                                                | `:46`                                              |
| Header                | bg `sidebar.bg`, colour `table.thead.color`, `font.size.base`, `padding 5px 10px`, `1px border0` grid lines          | `StyledWrapper.js:47-82`                           |
| Resize handle         | 3 px, `col-resize`, visible bg `colors.accent` on hover/drag                                                         | `:67-80`                                           |
| Checkbox              | 14×14, `accent-color: colors.accent`                                                                                 | `:160-167`                                         |
| Drag handle           | hidden until row hover; `IconGripVertical` 14 px / `IconMinusVertical` 14 px swap on hover; opacity transition `.1s` | `:214-234`, `index.js:1072-1081`                   |
| Dragging source       | `opacity: .4`                                                                                                        | `:232-234`                                         |
| No-results            | `No results found for “{query}”` (curly quotes) centred, `padding 24px`, `font.size.sm`                              | `index.js:978-981`, `StyledWrapper.js:138-143`     |

### 4.2 Columns — `index.js:993-1022`

| #   | Column        | Content                                                                                                                       | Notes | Source                 |
| --- | ------------- | ----------------------------------------------------------------------------------------------------------------------------- | ----- | ---------------------- |
| 1   | _(checkbox)_  | enabled checkbox (hidden for the trailing blank row)                                                                          |       | `index.js:1071-1091`   |
| 2   | `Name`        | text input, placeholder `Name`; sortable header (cycles Default → A-Z → Z-A); error icon `IconAlertCircle` 20 px with tooltip |       | `:993-1009,1093-1121`  |
| 3   | `Value`       | `EnvVarValueCell` (below)                                                                                                     |       | `:1010-1018,1122-1137` |
| 4   | `Description` | `MultiLineEditor`, placeholder `Description` (blank row only)                                                                 |       | `:1019,1138-1163`      |
| 5   | _(actions)_   | trash button `IconTrash` 18 px                                                                                                |       | `:1164-1170`           |

Sort tooltips: `Default order`, `Sorted A-Z`, `Sorted Z-A`
(`hooks/useSortCycle/index.js:17-21`).

Validation messages (`index.js:425-473`): `Name cannot be empty`;
invalid characters → `Variable contains invalid characters. Must only contain alphanumeric characters, "-", "_", "."`
(`utils/common/variables.js:31-32`); duplicate secret → `Secret names must be unique`
(`utils/environments.js:147`).

### 4.3 Secret value display (masked + reveal)

| Behaviour        | Detail                                                                                                                                                                         | Source                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| Mask             | Rows with `secret: true` render the `MultiLineEditor` with `isSecret` and `hideSecretEye`; text is masked by CodeMirror                                                        | `index.js:178-210`                                                       |
| Reveal control   | `SecretEyeButton` (`IconEyeOff` while masked, `IconEye` while revealed; 18 px, stroke 2) in `trailingContent`, `data-testid="secret-reveal-toggle"`                            | `index.js:163-171`; `components/MultiLineEditor/SecretEyeButton.js:4-12` |
| State            | `masked` mirrors `showAsSecret = secret && !isLastEmptyRow`; toggling calls the editor's `toggleVisibleSecret()` and the editor reports back via `onMaskChange`                | `index.js:155-160,166-170,187`                                           |
| Inherited secret | Masked with the same character repeated (`*` per non-newline char) in `InheritedVariableRow`; its own eye toggles local state (`data-testid="inherited-secret-reveal-toggle"`) | `InheritedVariableRow/index.js:8-9,12-15,39-45`                          |

`VarValueCell` (compact behaviour) — `components/VarValueCell/index.js`:

| Rule              | Value                                                                                                                                                                | Source                             |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| Compact threshold | container width `< 150px`                                                                                                                                            | `:4,18-23`                         |
| Compact           | type selector overlays the cell, `opacity 0` → `1` on hover (`transition .15s`), bg `bg`, `padding 0 4px`, radius 4px; always visible when a type warning is present | `:27-34`; `StyledWrapper.js:13-42` |
| Non-compact       | type selector rendered inline with `gap .5rem`                                                                                                                       | `:37-65`                           |

### 4.4 Section rows (inherited vs own)

Shown only when the environment inherits from another (`index.js:953-970`).

| Order | Section label         | Count     | Behaviour               | Source                                                          |
| ----- | --------------------- | --------- | ----------------------- | --------------------------------------------------------------- | -------------------- |
| 1     | `Inherited {Variables | Secrets}` | matching inherited rows | collapsible, chevron `IconChevronDown`/`IconChevronRight` 14 px | `:952-961,1029-1046` |
| 2     | `{Variables           | Secrets}` | own named rows          | collapsible                                                     | `:962-969`           |

Section toggle: `font.size.base`, weight 600, `padding 8px 10px`, gap 6 px,
count in `colors.text.muted` (`StyledWrapper.js:109-123`).

Inherited rows (`InheritedVariableRow/index.js:24-62`): checkbox **checked &
disabled**, name, value (`valueToString(value, 2)`, masked if secret), inline
data type badge (`variable.dataType || 'string'`), description, and a source
button (`title="Inherited from {name}"`, `IconArrowUpRight` 16 px,
`data-testid="inherited-source"`) that **switches the detail pane to the parent
environment** (`:17-22`). Rows are `colors.text.muted` (`StyledWrapper.js:125-127`).

### 4.5 Drag-to-reorder

| Rule               | Detail                                                       | Source         |
| ------------------ | ------------------------------------------------------------ | -------------- |
| Enabled            | only when sort mode is `default` **and** no search is active | `index.js:333` |
| Handle             | appears on row hover in the checkbox column, `cursor: grab`  | `:1072-1081`   |
| Reorder            | within the active tab's subset only (`reorderWithinSubset`)  | `:499-505`     |
| Trailing blank row | cannot be dragged                                            | `:94,1072`     |

### 4.6 Footer

| Button  | Style                                                                                              | Behaviour                                                                         | Source                                                       |
| ------- | -------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| `Save`  | `.submit`: `padding 6px 16px`, `font.size.sm`, radius 6px, bg `brand`, text `bg`; hover opacity .9 | saves only the active tab's rows; if nothing changed → toast `No changes to save` | `index.js:1181-1183`, `:675-693`; `StyledWrapper.js:190-203` |
| `Reset` | `.reset`: transparent, text `brand`                                                                | `handleReset`                                                                     | `:1184-1186`; `StyledWrapper.js:205-212`                     |

Footer bar: `padding 12px 2px`, bg `bg`, gap 8 px (`StyledWrapper.js:182-188`).

Draft / autosave: edits are staged to `environmentsDraft` 300 ms after the last
change (`index.js:572-592`); closing with unsaved changes shows the
unsaved-changes modal (§6.4).

Save shortcut: `Cmd/Ctrl+S` dispatches `environment-save`; `Cmd/Ctrl+Shift+S`
dispatches `environment-save-all` (`index.js:883-898`).

---

## 5. VariablesEditor (read-only variables tab)

`components/VariablesEditor/**`. Tab type `variables`.

### 5.1 Layout

```text
┌ VariablesEditor ───────────────────────────────────────┬ drawer (resizable) ┐
│ ▸ ⚡ Runtime Variables 12                                │  obj  (Runtime)   ✕ │
│ ▸ 🗄 Environment Variables 8            Staging        │ {  … JSON …      } │
│   (table: Name | Value | actions)                      │                    │
└────────────────────────────────────────────────────────┴────────────────────┘
```

| Region              | Value                                                                                                              | Source                                                     |
| ------------------- | ------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------- |
| Scroll area         | `padding: 0 1rem 1rem`                                                                                             | `VariablesEditor/StyledWrapper.js:17-23`                   |
| Section header      | sticky, height **38px** (`SECTION_HEADER_HEIGHT`), gap 8 px, `padding 0 8px`, bg `bg`, hover `background.surface0` | `constants.js:7`; `VariablesSection/StyledWrapper.js:5-26` |
| Chevron             | `IconChevronRight` 14 px, rotates 90° when expanded, `transition transform .2s ease`, `colors.text.muted`          | `VariablesSection/index.js:15`; `StyledWrapper.js:28-36`   |
| Section icon        | 16 px, `primary.text`                                                                                              | `StyledWrapper.js:38-41`                                   |
| Title               | weight 600                                                                                                         | `:43-45`                                                   |
| Count               | `colors.text.muted`, tabular numerals                                                                              | `:47-50`                                                   |
| Subtitle (env name) | pushed right (`margin-left:auto`), `font.size.sm`, muted                                                           | `:52-56`                                                   |
| Drawer              | width persisted, default **400px**, min **280px**, max `min(container, 60%)`                                       | `constants.js:9-13`                                        |
| Drawer handle       | 6 px hit area at `left:-3px`, 1 px line `sidebar.dragbar.border`, hover `activeBorder`                             | `StyledWrapper.js:36-58`                                   |

### 5.2 Sections

| #   | Icon           | Title                                                 | Count                    | Rows source                      | Empty text                                                                   | Source             |
| --- | -------------- | ----------------------------------------------------- | ------------------------ | -------------------------------- | ---------------------------------------------------------------------------- | ------------------ |
| 1   | `IconBolt`     | `Runtime Variables`                                   | `runtimeVariables` count | enabled/sorted name              | `No runtime variables found`                                                 | `index.js:202-226` |
| 2   | `IconDatabase` | `Environment Variables` + active env name as subtitle | env vars count           | enabled env vars, sorted by name | `No environment selected` (no active env) / `No environment variables found` | `index.js:228-257` |

Empty/muted text: `.muted text-xs px-2 py-1` (`index.js:224,238,255`).

### 5.3 Table (`VariablesTable/index.js`)

| Column  | Width | Content             | Source   |
| ------- | ----- | ------------------- | -------- |
| `Name`  | 35 %  | read-only, sortable | `:19-27` |
| `Value` | 65 %  | `VariableValue`     | `:28-46` |

`EditableTable` is used with `showCheckbox={false}`, `showDelete={false}`,
`showAddRow={false}` (`:63-70`).

`VariableValue` (`VariableValue/index.js`):

| Element                | Detail                                                                                                                                                                                | Source                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| Secret mask            | `'********'` (8 asterisks) when `secret && !revealed`                                                                                                                                 | `:29,36-39`                              |
| Object/array           | rendered as expandable JSON in a `MultiLineEditor`, `max-height 120px` (`OBJECT_CELL_MAX_HEIGHT`)                                                                                     | `:30-34,68-78`; `constants.js:1`         |
| Editor font            | mono stack `ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace`, **12px**, transparent bg                                                         | `VariableValue/StyledWrapper.js:5,38-46` |
| Open object            | `IconArrowsDiagonal` 15 px, `title="Open in drawer"`, `aria-label="Open object in drawer"`                                                                                            | `:89-100`                                |
| Reveal secret          | `IconEye`/`IconEyeOff` 15 px, `title` `Show value`/`Hide value`                                                                                                                       | `:101-114`                               |
| Copy                   | `IconCopy` 15 px → `IconCheck` 15 px for 1200 ms (`COPY_FEEDBACK_MS`), `title="Copy value"`                                                                                           | `:115-125`; `constants.js:37`            |
| Row actions visibility | `opacity 0`, pointer-events none; visible when `.is-pinned` (drawer open for that var) or `:focus-visible`; buttons `padding 2px`, radius 4px, `colors.text.muted`, `transition .12s` | `StyledWrapper.js:58-81`                 |

Drawer (`VariableDetailsDrawer/index.js`): title = variable name + `(Runtime)` or
`(Environment)` badge, close button `title="Close"` (`IconX` 16 px), JSON
`CodeEditor` read-only (`:32-60`). Drawer opens only for object/array values and
closes automatically when a secret is re-masked (`index.js:143-158`).

---

## 6. Modals

### 6.1 Create Environment (from selector)

`Environments/EnvironmentSettings/CreateEnvironment/index.js`.

`Modal size="md" title="Create Environment" confirmText="Create"`
(`:61-68`). Field label **`Environment Name`** (`:71-73`), full-width
text input. Validation: `Must be at least 1 character`,
`Must be 255 characters or less`, `Name is required`,
`Environment already exists` (`:24-34`). Success toast
`Environment created in collection` (`:38`); failure
`An error occurred while creating the environment` (`:45`).

### 6.2 Copy Environment

`EnvironmentSettings/CopyEnvironment/index.js`. `Modal size="sm"
title="Copy Environment" confirmText="Copy"` (`:46-53`). Label
**`New Environment Name`** (`:56-58`), default value `"{name} - Copy"` (`:16`).
Validation: `must be at least 1 character`, `must be 50 characters or less`,
`name is required` (lower-case, `:19-23`). Toasts: `Environment created in collection`,
`An error occurred while created the environment` (sic, `:27,30`).

### 6.3 Delete environments

`Common/DeleteEnvironmentsModal/index.js`.

| Condition           | Title                      | Body                                                                                                          | Source      |
| ------------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------- | ----------- |
| 1 env               | `Delete environment?`      | `This permanently deletes {name} and its file from disk. This cannot be undone.`                              | `:14-16,87` |
| N envs              | `Delete {n} environments?` | `This permanently deletes the environment files from disk. This cannot be undone.`                            | `:17,87`    |
| Active env included | —                          | warning: `{name} is your active environment. Deleting it leaves the collection with no environment selected.` | `:97-105`   |
| Global variant      | same titles                | `… from this workspace …` (`scopeLabel = workspace`)                                                          | `:20-29`    |

Confirm button `Delete` / `Delete ({n})`, `confirmButtonColor="danger"` (`:88-91`).
Toasts: `Failed to delete "{name}"` per failure, `{n} environment(s) deleted`
(`:68,76`).

### 6.4 Unsaved changes

`Environments/ConfirmCloseEnvironment/index.js`.

`Modal size="md" title="Unsaved changes"`, no footer; body:
`IconAlertTriangle` 32 px + **`Hold on...`** (`:26-29`) then
`You have unsaved changes in {collection environment settings | global environment settings | .env file}.`
(`:8-13,30-32`). Actions: `Don't Save` (danger), `Cancel` (secondary ghost),
`Save` (`:34-48`).

### 6.5 Import environments

`Common/ImportEnvironmentModal/**`. Two steps.

| Step    | Copy                                                                                                                                                                                                                                                                                                                                                                                                               | Source                                                |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| Upload  | Title `Import environments` (collection) / `Import global environments`; dropzone `IconFileImport` 64 px; title `Import your environments`, on drag-over `Drop your environment files here`; subtitle `Drag & drop JSON files or click to browse. Supports both Bruno and Postman formats.`                                                                                                                        | `index.js:25`; `UploadStep/index.js:44,68-75`         |
| Review  | Custom header: title + `{n} found`; search placeholder `Filter by name`; groups `New` and `Already exists`; per-duplicate resolution `New` (`Import as a new environment`) / `Replace` (`Replace existing`); invalid group; warnings `{n} already exists with the same name`, `{n} could not be read and will be skipped`; no-match text `No environments match your filter`; footer `{selected}/{total} selected` | `ReviewStep/index.js:28-35,100-200`; `utils.js:22-25` |
| Confirm | `Import ({n})` or `Import`; `Cancel`                                                                                                                                                                                                                                                                                                                                                                               | `ReviewStep/index.js:95-98`                           |

### 6.6 Export environments

`Common/ExportEnvironmentModal/index.js`: `Modal title="Export Environments"`
(`:192`) with a target-location control (`placeholder="Select a target location"`,
`:302`). Per-environment selection is pre-seeded from the context-menu target
(`EnvironmentList/index.js:662-669`). Full step list **UNVERIFIED** (file not
read end-to-end).

### 6.7 Delete `.env` file

`EnvironmentSettings/DeleteDotEnvFile/index.js`: `Modal size="sm"
title="Delete {filename} File" confirmText="Delete"` (danger);
body `Are you sure you want to delete {filename} file?` (`:15-24`).

---

## 7. `.env` file editor

`Environments/DotEnvFileEditor/**` + `DotEnvFileDetails`.

| Part           | Detail                                                                                                                                                                       | Source                                              |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| Details header | filename title; when the file exists: view toggle `Table` / `Raw` (`role="group"` `aria-label="View mode"`) and delete button `title="Delete .env file"` (`IconTrash` 15 px) | `DotEnvFileDetails/index.js:28-57`                  |
| Empty state    | `IconFileOff` 48 px + `No .env File` + `Add a variable below to create a .env file in this location.`                                                                        | `DotEnvFileEditor/DotEnvEmptyState.js:4-13`         |
| Table view     | columns with placeholders `Name` / `Value`; `Save` (idle) / `Saving...`                                                                                                      | `DotEnvFileEditor/DotEnvTableView.js:62,77,121-123` |
| Raw view       | plain-text editor; `Save` / `Saving...`                                                                                                                                      | `DotEnvFileEditor/DotEnvRawView.js:31-33`           |

---

## 8. Global environments

`GlobalEnvironmentSettings/index.js` renders `WorkspaceEnvironments`
(`:11`) — global environments belong to the **workspace**, not the collection.
Selector `Global` tab lists them (`EnvironmentSelector/index.js:194-198,309`),
selection dispatches `selectGlobalEnvironment` (`:219`). Toolbar can therefore
show a collection env, a global env, both, or neither (§2.2).

Deep workspace-environment UI is out of scope here; only the collection-side
integration is specified. Create/import from the `Global` tab use
`WorkspaceHome/WorkspaceEnvironments/CreateEnvironment` and the same
`ImportEnvironmentModal type="global"` (`EnvironmentSelector/index.js:321-334`).

---

## 9. Environment colour dots — summary

| Location               | Component                                                                 | Size       | Source                                                           |
| ---------------------- | ------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------- |
| Selector trigger       | icon + text tinted by env colour; badge bg = colour @ 12 %                | 14 px icon | `EnvironmentSelector/index.js:31-38,82-103`                      |
| Dropdown rows          | `ColorBadge size={8}`                                                     | 8 px       | `EnvironmentListContent/index.js:118`                            |
| Settings list rows     | `ColorBadge size={8}`                                                     | 8 px       | `EnvironmentList/index.js:839`                                   |
| Settings detail header | `ColorPicker` (`ColorBadge size={8}` as its icon; `title="Change color"`) | 8 px       | `EnvironmentDetails/index.js:244`; `ColorPicker/index.js:75-109` |

`ColorBadge` renders a solid `border-radius:9999px` circle with
`background-color: color || transparent` (`components/ColorBadge/index.js:4-19`).

---

## 10. Shortcuts

| Scope                | Keys                                   | Action                               | Source                                                               |
| -------------------- | -------------------------------------- | ------------------------------------ | -------------------------------------------------------------------- |
| Selector dropdown    | any printable key                      | focus search                         | `EnvironmentListContent/index.js:30-54`                              |
| Selector dropdown    | `Escape`                               | clear search, else blur              | `:20-28`                                                             |
| Env list (settings)  | `Ctrl/Cmd+A`                           | select all (closes row menu)         | `Common/EnvironmentSelection/useEnvironmentShortcuts/index.js:37,43` |
| Env list             | `Delete` (win) / `Cmd+Backspace` (mac) | delete target env(s)                 | `:38-47`; hint text in context menu                                  |
| Env list             | `Escape`                               | clear selection                      | `:28`                                                                |
| Variables grid       | `Cmd/Ctrl+S`                           | `environment-save` → save active tab | `EnvironmentVariablesTable/index.js:883-898`                         |
| Variables grid       | `Cmd/Ctrl+Shift+S`                     | `environment-save-all`               | `:887-892`                                                           |
| Variables grid       | `Enter` in Name                        | mark touched (validate)              | `:668-673`                                                           |
| Inline rename/create | `Enter` / `Escape`                     | save / cancel                        | `:300-316`                                                           |
| Global app           | `Cmd/Ctrl+E`                           | `Edit Environment`                   | `providers/Hotkeys/keyMappings.js:57`                                |

---

## 11. Verbatim copy index (English)

| String                                                                                      | Where                     | Source                                                                   |
| ------------------------------------------------------------------------------------------- | ------------------------- | ------------------------------------------------------------------------ |
| `No Environment`                                                                            | trigger + dropdown row    | `EnvironmentSelector/index.js:119`; `EnvironmentListContent/index.js:90` |
| `Search environments...`                                                                    | both search inputs        | `EnvironmentListContent/index.js:72`; `EnvironmentList/index.js:759`     |
| `No results found`                                                                          | selector dropdown filter  | `EnvironmentListContent/index.js:105`                                    |
| `Configure`                                                                                 | selector footer           | `:128`                                                                   |
| `Ready to get started?`                                                                     | selector empty state      | `:135`                                                                   |
| `Create` / `Import`                                                                         | selector empty state      | `:140,144`                                                               |
| `Collection` / `Global`                                                                     | selector tabs             | `EnvironmentSelector/index.js:19-20`                                     |
| `Environments` / `.env Files`                                                               | sidebar sections          | `EnvironmentList/index.js:707,946`                                       |
| `No environments` / `No .env files`                                                         | sidebar empties           | `:916,1016`                                                              |
| `Environment name...`                                                                       | inline create placeholder | `:884`                                                                   |
| `Create "{q}"`                                                                              | create-from-search        | `:931`                                                                   |
| `Name is required` / `Environment already exists`                                           | validation                | `:224,236`                                                               |
| `Variables` / `Secrets`                                                                     | detail tabs               | `hooks/useEnvironmentTabs/index.js:38-39`                                |
| `Search variables...` / `Search secrets...`                                                 | detail search             | `EnvironmentDetails/index.js:286`                                        |
| `Inherit variables from` / `Inherits from: {name}` / `No Environment`                       | inheritance control       | `Common/InheritsFrom/index.js:23,27,57`                                  |
| `No changes to save`                                                                        | toast                     | `EnvironmentVariablesTable/index.js:691`                                 |
| `No results found for “{q}”`                                                                | grid filter empty         | `:980`                                                                   |
| `Name` / `Value` / `Description`                                                            | grid placeholders         | `:1107,184,1144`                                                         |
| `Runtime Variables` / `Environment Variables`                                               | viewer sections           | `VariablesEditor/index.js:204,230`                                       |
| `No runtime variables found` / `No environment variables found` / `No environment selected` | viewer empties            | `:224,255,238`                                                           |
| `Show value` / `Hide value` / `Copy value` / `Open in drawer`                               | viewer row actions        | `VariableValue/index.js:94,106-107,119`                                  |
| `No .env File` / `Add a variable below to create a .env file in this location.`             | `.env` empty              | `DotEnvFileEditor/DotEnvEmptyState.js:8-11`                              |
| `Hold on...` / `You have unsaved changes in {scope}.`                                       | unsaved modal             | `ConfirmCloseEnvironment/index.js:28,31`                                 |
| `Import your environments` / `Drop your environment files here`                             | import dropzone           | `UploadStep/index.js:70,71`                                              |
| `Drag & drop JSON files or click to browse. Supports both Bruno and Postman formats.`       | import subtitle           | `UploadStep/index.js:73-74`                                              |

No `i18n` lookups exist anywhere in these components — all copy is inline English
(contrast: `Preferences/Support`, which uses `i18n/translation/en.json`).

---

## 12. Interaction summary

| Interaction                | Result                                                               |
| -------------------------- | -------------------------------------------------------------------- |
| Pick env in selector       | Sets active collection or global env; toast; dropdown closes         |
| `No Environment` row       | Clears the active env for that tab                                   |
| `Configure`                | Opens `environment-settings` / `global-environment-settings` tab     |
| Click env in settings list | Loads detail; blocked when the current env has unsaved changes       |
| Double-click env           | Inline rename                                                        |
| Activate button            | Sets this env as the collection's active env                         |
| Colour swatch              | Open colour picker (presets + custom range)                          |
| `Inherits from`            | Choose parent env → inherited variables appear in a separate section |
| Inherited row source icon  | Jump to the parent env's detail                                      |
| Value eye                  | Mask/unmask secret                                                   |
| Drag handle                | Reorder rows within the active tab (default sort only)               |
| `Save` / `Reset`           | Persist active tab / revert                                          |
| Delete                     | Confirm modal → permanent delete from disk                           |

---

## 13. Open Questions

1. **`SearchInput` default `iconSize`** is not passed by the selector list
   (`EnvironmentListContent/index.js:70-82` passes `iconSize={13}` — verified);
   the settings-list search uses a hand-rolled input instead
   (`EnvironmentList/index.js:754-779`), so the two search fields differ
   visually. The port should unify them.
2. **Secret masking differs per surface**: the editor masks via CodeMirror's
   secret mode (bullet glyph defined in `MultiLineEditor`), the viewer uses
   eight literal `*` (`VariableValue/index.js:37`), and inherited rows repeat
   `*` per character (`InheritedVariableRow/index.js:8-9`). Pick one convention
   for the Go port and note it in the design system.
3. **`Name cannot be empty` vs `Name is required`** — two validators
   (`EnvironmentVariablesTable/index.js:433` vs `EnvironmentList/index.js:224`).
4. **Filter semantics differ**: the selector filters by env name substring;
   the variables grid searches `name` + value + description
   (`EnvironmentVariablesTable/index.js:133-137`).
5. **Environment colour palette** is hard-coded in `ColorPicker`
   (`#CE4F3B …`), not part of the theme token set. Decide whether the port
   tokenises it.
6. **`EnvironmentSelector` dropdown width** is computed as `maxNameLength × 8px`
   (`EnvironmentSelector/index.js:70-77`) — a heuristic that breaks with wide
   glyphs; clamp in the port.
7. **`refs/workspace_switcher_bruno.png` is not an environment screenshot.**
   The workspace switcher popover (`My Workspace`, `test ✓`, `Workspaces`,
   `Create workspace`, `Open workspace`, `Import workspace`, `Manage workspaces`)
   is documented elsewhere (app shell); it was requested by the brief but does
   not depict environments.
8. **Global environment settings** are owned by `WorkspaceHome` — this file only
   covers the collection-side hooks into them. A full global-environment spec
   should be added to a `workspace.md` screen file.
9. **`ExportEnvironmentModal` step detail** is not fully specified here
   (**UNVERIFIED** beyond the title/selector/Preselection).
10. **Sorting persistence keys** include the active tab uid
    (`persisted::${activeTabUid}::…`, `EnvironmentVariablesTable/index.js:282-284,330-331`)
    — clarify whether the Go port should scope scroll/sort per tab or per env.
