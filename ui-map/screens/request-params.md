# Screen — Params, Headers, form params, file body (key/value tables)

Five tab bodies share one table primitive. Everything below is glue over
`components/EditableTable`.

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source, except `drag-over` row styling (Open
Questions #1).

---

## 1. Layout

### 1.1 Params tab (`QueryParams`)

```text
Query                          ← .title text-xs, subtext0
┌──────────────────────────────────────────────────────────────┐
│ ☑ │ Name        │ Value        │ Description      │ 🗑         │  header 35px, sticky
├──────────────────────────────────────────────────────────────┤
│ ☑ │ page        │ 1            │ Page number      │ 🗑         │  row 35px
│ ☑ │ limit       │ 10           │ Number of item   │ 🗑         │
│   │ Name        │ Value        │ Description      │            │  always-present add row
└──────────────────────────────────────────────────────────────┘
                                      Bulk Edit                ← right-aligned link
Path (?)                                                       ← .title + InfoTip
┌──────────────────────────────────────────────────────────────┐
│ │ id          │ 42           │ Description      │            │  read-only name, no checkbox, no delete, no add row
└──────────────────────────────────────────────────────────────┘
```

Source: `RequestPane/QueryParams/index.js:186-239`.

### 1.2 Headers tab (`RequestHeaders`) — HTTP requests only

Two modes of the same table:

- **Collapsed (default):** only request headers, exactly like the Query table.
  Footer left: eye icon + `Show Additional Headers (N)`; right: `Bulk Edit`
  (`RequestPane/RequestHeaders/index.js:556-583`).
- **Expanded:** the table gains two full-width *section* rows
  (`Inherited & Runtime Headers (n)` and `Request Headers (n)`), each a
  collapsible button with chevron (`index.js:255-291`), plus read-only
  italic name/value rows for inherited and Bruno default headers
  (`index.js:437-497`, `RequestHeaders/StyledWrapper.js:74-82`).
  Footer left becomes eye-off icon + `Hide Additional Headers`.

### 1.3 Form-urlencoded / Multipart / File body

```text
Form URL Encoded → Key | Value | Description            (RequestPane/FormUrlEncodedParams/index.js:53-85)
Multipart Form   → Key | Value | Content-Type | Description
                   Value cell = text editor + upload button, or file chips
                   (RequestPane/MultipartFormParams/index.js:180-240)
File / Binary    → File | Content-Type | Selected(radio) | Description | 🗑
                   + Add File                              (RequestPane/FileBody/index.js:70-186)
```

---

## 2. Key-value table behaviour (`components/EditableTable`)

| Aspect | Value | Source |
| --- | --- | --- |
| Row height | `35px` (`ROW_HEIGHT`), header row 35px | `EditableTable/index.js:14`, `StyledWrapper.js:111,120,188-190` |
| Virtualisation | `TableVirtuoso`, `customScrollParent` = nearest overflow-y ancestor | `index.js:21-29,587-602` |
| Table border | `1px solid border.border0` (`#efefef` L / `#2a2a2a` D), radius `border.radius.base` (6px) | `StyledWrapper.js:13-17` |
| Header | background `sidebar.bg` (`#f8f8f8` L / `#222224` D), colour `table.thead.color` (`#666666` L / `#bbb` D), `padding: 5px 10px`, sticky, `z-index: 13` | `StyledWrapper.js:38-55` |
| Cell padding | body cells `1px 10px`; header cells `5px 10px` | `StyledWrapper.js:50,122` |
| Cell borders | right + bottom `1px solid border.border0`; left border never drawn | `StyledWrapper.js:53-54,125-126` |
| Text | `font.size.base` (13px), weight `normal` (overrides the legacy 500 in tab wrappers) | `StyledWrapper.js:34-35` |
| Input text colour | `text` (`#343434` L / `#cccccc` D) | `StyledWrapper.js:239` |
| Input placeholder | `codemirror.placeholder.color` + `.opacity` (`#b0b0b0` @ .75 L; `#a2a2a2` @ .5 D) | `StyledWrapper.js:248-251`, tokens |
| Checkbox | native, `14×14px`, `accent-color: colors.accent` (`#b96f1d` L / `#d9a342` D), first column fixed `25px` | `StyledWrapper.js:254-261,214-226,104-107` |
| Checkbox label | empty for all five tables (default `checkboxLabel = ''`) | `index.js:98`, callers pass none |
| Delete button | `IconTrash` 18px `strokeWidth 1.5`, `padding: 4px`, colour `colors.text.muted`, hover `colors.text.danger`; column width `57px` | `index.js:482-488,562-569`, `StyledWrapper.js:263-278` |
| Add row | always exactly one trailing empty row; typing into it promotes it and appends a new empty row; smooth-scrolls to it when appended | `index.js:265-315` |
| Row hover | no background change on `tr:hover`; the drag handle fades in (`opacity 0 → 1`, `0.1s ease`) | `StyledWrapper.js:280-296` |
| Drag handle | `IconMinusVertical` 14px normally, `IconGripVertical` 14px on row hover; sits at `left: -8px`, vertically centred, `cursor: grab`; hidden on the add row | `index.js:69-77,510-524`, `StyledWrapper.js:280-296` |
| Drag in progress | source row `opacity: 0.4`; a floating preview chip follows the pointer (`infoTip.bg`, `1px` border, `box-shadow`, font `0.75rem`, offset 12px) | `StyledWrapper.js:298-300`, `hooks/useMouseRowDrag/index.js:50-73` |
| Column resize | handle is `5px` wide at `right: -2px`, hover line `1px` in `colors.accent`; min column width `80px`; widths persisted per tab per table | `StyledWrapper.js:75-100`, `index.js:13,173-238` |
| Multi-line cells | a `MultiLineEditor` cell lets the row grow to `max-height: calc(35px * 3)` then scrolls | `StyledWrapper.js:167-186` |
| Single-line CodeMirror in a cell | clipped to `33px`, ellipsis, no scrollbars | `StyledWrapper.js:139-165` |
| Row error | `IconAlertCircle` 20px red at the cell's right, tooltip `max-width: 200px` | `index.js:375-388`, `StyledWrapper.js:228-233` |
| Focus flash | a row brought into view flashes `status.warning.background` for `2.5s` | `StyledWrapper.js:302-320` |
| Sortable columns | header becomes clickable; only `Vars` name column is `sortable: true` | `index.js:455-470`, `QueryParams` none, `Vars/VarsTable:82` |

### 2.1 Per-table columns

| Table | Columns (order) | Widths | Notes | Source |
| --- | --- | --- | --- | --- |
| Query params | `Name`, `Value`, `Description` | Name `20%`, Description `25%`, Value auto | checkbox + drag + delete; reorderable | `QueryParams/index.js:107-135` |
| Path params | `Name` (read-only), `Value`, `Description` | Name `20%` | `showCheckbox={false} showDelete={false} showAddRow={false}`; rendered only if ≥1 path param | `QueryParams/index.js:137-165,224-239` |
| Request headers | `Name`, `Value`, `Description` | Name `20%` | header name autocompletes from `know-your-http-well`; value autocompletes MIME types | `RequestHeaders/index.js:112-118,437-497` |
| Form URL-encoded | `Key`, `Value`, `Description` | Key `20%` | value allows newlines + `{{` autocomplete | `FormUrlEncodedParams/index.js:53-85` |
| Multipart | `Key`, `Value`, `Content-Type`, `Description` | Key `20%`, Value `35%`, Content-Type `20%` | `Content-Type` placeholder `Auto` | `MultipartFormParams/index.js:180-240` |
| File body | `File`, `Content-Type`, `Selected`, `Description`, (trash) | 25% / 20% / 10% / 35% / 70px | not an `EditableTable` — bespoke `<table>` | `FileBody/index.js:70-186`, `FileBody/StyledWrapper.js:14-32` |

Placeholders (verbatim): `Name`, `Value`, `Key`, `Description`, `Expr`, `Auto`.
File body: `Content-Type` placeholder `Auto`, description placeholder
`Description` (`FileBody/index.js:140,171`).

### 2.2 Path params behaviour

- Path params are **derived from the URL**: any `:name` segment in the URL
  becomes a row automatically (InfoTip text in `QueryParams/index.js:207-219`).
- Name column is read-only; only `value` and `description` are editable and go
  through `updatePathParam` (`QueryParams/index.js:71-83,137-165`).
- Empty state: no Path table at all; only the `Path` title and InfoTip remain
  followed by an empty spacer div (`QueryParams/index.js:221-238`).

InfoTip copy (verbatim, `QueryParams/index.js:209-219`): *"Path variables are
automatically added whenever the `:name` template is used in the URL. For
example: https://example.com/v1/users/:id"*.

### 2.3 Headers — extra behaviour

| Behaviour | Detail | Source |
| --- | --- | --- |
| Show/hide additional headers | toggle button text `Show Additional Headers (N)` / `Hide Additional Headers` | `RequestHeaders/index.js:571-581` |
| Section rows | `Inherited & Runtime Headers (N)`, `Request Headers (N)`; chevron 16px; bold (600); 35px tall; background `sidebar.bg` | `index.js:255-291`, `RequestHeaders/StyledWrapper.js:38-60` |
| Inherited/default rows | italic in `colors.text.muted`; inherited checkbox disabled at `opacity .55`; default rows are omittable only when `header.omittable` | `StyledWrapper.js:62-76`, `index.js:404-411` |
| Default-header info tooltip | `Automatically added at runtime` or `Automatically added at runtime. Required by HTTP, cannot be omitted` | `index.js:396-406` |
| Jump to source | external-link icon 16px, tooltip `Open headers in folder "X"` / `Open headers in collection "X"` | `index.js:348-372` |
| Validation | name `/` → `Header name cannot contain spaces or newlines`; value newline → `Header value cannot contain newlines`; newlines stripped on input | `index.js:311-325,455` |
| Bulk Edit | 200px-high plain-text editor; format `name:value` per line, `//` prefix = disabled; toggle back is labelled `Key/Value Edit` | `components/BulkEditor/index.js:91-96`, `utils/common/bulkKeyValueUtils.js:19` |

### 2.4 Multipart value cell

- Multiple files per row: chips with file icon, basename, remove `×`; hover
  tooltips `Remove file` / full path; upload button `title="Select File"`
  (`MultipartFormParams/index.js:205-213`, `MultipartFileChipsCell/index.js:16-38`).
- Overflow: `+{n} more` chip (`MultipartFileChipsCell/index.js:171-173`).
- Content type auto-derived on file add/remove
  (`MultipartFormParams/index.js:97,158`).
- Duplicate file toasts: `"<name>" is already added` / `N files are already added — skipped`
  (`MultipartFormParams/index.js:98-102`).

### 2.5 Bulk Edit toggle

- Link text `Bulk Edit` (all tables) at the bottom-right, `font.size.base`,
  underline on hover (`QueryParams/StyledWrapper.js:41-44`,
  `RequestHeaders/index.js:581`).
- Bar is sticky to the bottom of the scroll area with background `bg` and
  `padding-top: 8px; padding-bottom: 4px` (`QueryParams/StyledWrapper.js:46-52`).
- Bulk editor is `h-[200px]` (`components/BulkEditor/index.js:66-67`).

---

## 3. States

| State | Appearance | Source |
| --- | --- | --- |
| Empty table | one empty editable row showing column placeholders | `EditableTable/index.js:270-272` |
| Row disabled (checkbox off) | row stays fully visible, no dimming | `EditableTable/index.js:525-534` |
| Row error | red alert icon + tooltip (max 200px) | `index.js:375-388` |
| Drag source / drag over | `opacity .4` / handle visible | see §2 |
| Bulk edit mode | table replaced by the plain-text editor + `Key/Value Edit` link | `QueryParams/index.js:169-179` |

---

## 4. Interactions

| Interaction | Result | Source |
| --- | --- | --- |
| Type in last empty row | row committed; new empty row appended; `onChange` dispatched | `EditableTable/index.js:317-334` |
| Clear all values in a row | row is dropped on next change (last row rebuilt) | `index.js:328-331` |
| Toggle checkbox | `enabled` flag on the row (headers: `omitHeaders` for runtime defaults) | `index.js:336-343`, `RequestHeaders/index.js:262-284` |
| Drag handle | reorders within the reorderable rows; `onReorder` with the new uid order | `index.js:352-368` |
| Delete | removes row by uid | `index.js:345-348` |
| Column resize drag | adjusts this and the next column width, min 80px | `index.js:173-238` |
| `Enter` in a cell editor | sends the request (same `onRun` as the Send button) | `SingleLineEditor/index.js:75` |
| `Cmd/Ctrl + S` in a cell | saves the request | `keyMappings.js:9` |

---

## 5. Verbatim copy index

| String | Source |
| --- | --- |
| `Query`, `Path` | `QueryParams/index.js:188,204` |
| `Name`, `Value`, `Description`, `Key` | column definitions, see §2.1 |
| `Bulk Edit` / `Key/Value Edit` | `QueryParams/index.js:194`, `BulkEditor/index.js:93` |
| `Show Additional Headers (N)` / `Hide Additional Headers` | `RequestHeaders/index.js:575-578` |
| `Inherited & Runtime Headers` / `Request Headers` | `RequestHeaders/index.js:193,200` |
| `Automatically added at runtime` / `Automatically added at runtime. Required by HTTP, cannot be omitted` | `RequestHeaders/index.js:396-397` |
| `Header name cannot contain spaces or newlines` / `Header value cannot contain newlines` | `RequestHeaders/index.js:317,321` |
| `Content-Type`, `Auto`, `Selected`, `File`, `+ Add File` | `MultipartFormParams/index.js:220`, `FileBody/index.js:75-91,185` |
| `Select File` (tooltip), `Add files`, `Remove file` | `MultipartFormParams/index.js:212`, `MultipartFileChipsCell/index.js:16,188` |

Screenshots: `refs/api_screen_bruno.png` (Query + Path), `refs/main_section_bruno.png`
(header table with `Bulk Edit`).

---

## 6. Open Questions

1. **`drag-over` styling is not defined.** `EditableTable` sets a `drag-over`
   class on the target row (`index.js:53`) and `StyledWrapper` only uses it to
   reveal the drag handle (`:293-296`). There is no drop-indicator line colour
   in the table stylesheet. Do not invent one; flag for a decision.
2. **`dragAndDrop.*` tokens** (`border`, `borderStyle`, `hoverBg`,
   `transition`) are present in the theme but not referenced by
   `EditableTable`/`useMouseRowDrag`. They may belong to other DnD surfaces.
3. **Border collapse on the table.** `StyledWrapper.js:32` sets
   `border-collapse: collapse` while the outer container also has a 1px border
   and radius 6px (`:14-16`); corner rendering in the real app should be
   compared against `refs/api_screen_bruno.png` before fixing pixel values.
4. **Path-param table shares `request-vars`-style scroll persistence** — the
   Path table reuses the Query table's `initialScroll`
   (`QueryParams/index.js:234`); whether this is intentional is unclear.
5. **Query table in `refs/api_screen_bruno.png` shows a delete column with no
   visible drag handle** (handles only appear on hover) — consistent with
   §2, but drag order is not verifiable from a static screenshot.
