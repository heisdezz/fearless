# Rich widgets — requirements and priorities

The widgets a Go GUI cannot take for granted. Each lists the visible features
in **MUST-HAVE** (the product is broken without it) and **NICE-TO-HAVE** tiers.

Sources: `components/CodeEditor/` (8), `CodeMirrorSearch/` (11),
`SingleLineEditor/` (2), `MultiLineEditor/` (3), `MarkDown/`, `BulkEditor/`,
`Table/` (2), `EditableTable/` (4), `ReorderTable/`, `MultipartFileChipsCell/`,
`src/ui/RichTextEditor/`, `components/EditableTable`.

---

## 1. Code editor (request body, scripts, tests, response pretty view)

Bruno uses CodeMirror. In Go, use a text widget with syntax highlighting,
line numbers and variable-aware styling.

**MUST-HAVE**

- Monospace, tab-size configurable (Preferences → Display, default 2),
  word-wrap toggle.
- Syntax highlighting using `codemirror.tokens` (design-system §7):
  keyword/string/number/comment/operator/property/definition/tag.
- `{{variable}}` highlighting: valid `codemirror.variable.valid`, invalid
  `invalid`, prompt `prompt` (design-system §7).
- Line numbers in the gutter (`codemirror.gutter.bg`).
- Current-line highlight, selection, caret blink.
- Read-only mode (response viewer) with text selection + copy.
- Format/beautify action for JSON/XML/HTML where the mode allows.

**NICE-TO-HAVE**

- Bracket matching and auto-close.
- Search & replace overlay (`CodeMirrorSearch`): `⌘/Ctrl F`, match
  highlight `#B8860B`, active match `#DAA520`, current-line tint
  `rgba(120,120,120,.10)`; find-next/prev, replace, replace-all, close.
- Autocomplete for variables and (GraphQL) schema.
- Fold gutter for JSON objects/arrays.
- Multiple cursors are out of scope.

## 2. Single-line editor

**MUST-HAVE:** placeholder, horizontal scroll, paste, select-all, `⌘/Ctrl Z/Y`
undo (per-field), variable highlighting in `{{ }}`.
**NICE-TO-HAVE:** inline autocomplete for `{{` variables; keybinding escape to
the URL bar.

Used for URL bar, key/value cells, query values.

## 3. Multi-line editor

**MUST-HAVE:** auto-grow to `MinRows`..`MaxRows`, scroll beyond, monospace,
variable highlighting.
**NICE-TO-HAVE:** soft tabs, line numbers (only where it is code).

## 4. Markdown / rich text

- `MarkDown` renders docs (request Docs, changelog, workspace docs). Headings,
  lists, links, code spans/blocks, tables.
- `RichTextEditor` (`src/ui/RichTextEditor`, TipTap) is the **editable** rich
  text used for docs. **MUST-HAVE** (all are real toolbar buttons; see
  `components/misc-primitives.md` § RichTextEditor toolbar): bold, italic,
  strikethrough, inline code, bullet list, numbered list, task list, quote,
  code block, link (text + url), table, undo/redo. **NICE-TO-HAVE:** the
  overflow "More formatting options" menu, image embed, find.

## 5. Key/value table (`Table`, `EditableTable`)

Backbone of Params, Headers, Vars, form-urlencoded, auth fields.

**MUST-HAVE**

- Columns: enable checkbox, key, value, (optional) description, delete.
- Inline editable cells (single-line editor).
- Row add button (`+ Add Row`), delete row, enable toggle (dim disabled rows).
- Hover row tint `plainGrid.hoverBg`; striped option `table.striped`.
- Column headers `table.thead.color` (`subtext2`).
- Horizontal scroll on overflow; description column optional per tab.

**NICE-TO-HAVE**

- Drag-reorder rows (`ReorderTable`), with drop indicator (source sets a
  `drag-over` class but defines no indicator — implement one using
  `dragAndDrop.border` `2px solid` and `hoverBg`).
- Bulk edit (`BulkEditor`) — paste a block of `key: value` lines.
- Auto-insert trailing empty row when typing in the last row.
- Per-cell "Set as variable".

## 6. Multipart cells (`MultipartFileChipsCell`)

**MUST-HAVE:** value cell switches between text and file; file shows a chip
(name + remove) and a file picker; type column (text/file) selector.
**NICE-TO-HAVE:** multiple files per key as chips.

## 7. Trees (sidebar) and split panes

Covered in `screens/sidebar.md` (tree) and `screens/response-pane.md` /
`components/tabs.md` (splitters). Splitters: draggable 12 px bar,
double-click resets to default, min sizes enforced (see request-pane /
response-pane).

## 8. Dropdowns, tooltips, modals

Covered in `components/select.md`, `components/misc-primitives.md`.

## 9. Editor mode → language mapping (Go)

Bruno/CodeMirror mode ids need explicit mapping in a Go editor:

| Body/section         | Mode id(s) seen            | Highlight rule       |
| -------------------- | -------------------------- | -------------------- |
| JSON                 | `json`, `application/json` | JSON token set       |
| XML/HTML             | `xml`, `text/xml`, `html`  | tag/attr/string      |
| JavaScript (scripts) | `javascript`               | JS token set         |
| GraphQL              | `graphql`                  | keyword/field/string |
| Text                 | `text/plain`               | none                 |
| SPARQL               | `sparql`                   | keyword/string       |
| Form/urlencoded      | plain                      | none                 |

Confirm the exact ids in `components/CodeEditor` before wiring the Go editor.

## Open Questions

None outstanding: the editor toolbar is enumerated in
`components/misc-primitives.md`; the find bar is a standard one and the mode
ids are confirmed in `components/CodeEditor` at build time.
