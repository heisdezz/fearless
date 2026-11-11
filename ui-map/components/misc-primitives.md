# Components — non-table primitives (modal, toast, tooltip, menu, checkbox, radio, tag, color, snippet, list-group, error-banner, dropdown extras, variable scope)

Small shared primitives. Each entry: what it is, the tokens it uses, and its
states. Feature screens reference these.

---

## Modal (`components/Modal`)

Centred dialog with title bar + body + optional footer (Cancel / OK).

| Part                | Token                                                                               |
| ------------------- | ----------------------------------------------------------------------------------- |
| title bar bg / text | `modal.title.bg` (`surface0`→light `#f1f1f1`, dark `#1a1a1a`) / `modal.title.color` |
| body bg / text      | `modal.body.bg` (light `#ffffff`, dark `#222224`) / `modal.body.color`              |
| backdrop            | `modal.backdrop.opacity` `0.4`                                                      |
| radius / shadow     | `border.radius.md`–`lg` / `shadow.md`                                               |

Props: `title`, `size` (`sm|md|lg|xl`), `hideFooter`, `handleCancel`,
`handleConfirm`, `customHeader`, `confirmText`/`cancelText`. Behaviour: traps
focus, restores focus on close, `Esc` cancels, backdrop click cancels. Esc when
there are unsaved changes uses the confirm-close dialogs (see
`screens/request-tabs.md`).

Ctrl/Cmd+Enter confirms in prompt modals (see `dialogs` in repo:
`internal/ui/dialogs/view.go`).

## Toast

Bruno has **no general toast component** at this commit. Transient messages use
`react-hot-toast` directly (`toast.success` / `toast.error` / `toast.loading`),
positioned bottom-right, auto-dismiss ~4 s. The only `components/Toast` entry is
`Toast/ImportIssuesToast`, a bespoke persistent notice shown after an import
when some items failed — it lists the issues and closes on user action.

Go port: use `feedback.ToastHost` for the transient messages; render the import
issues notice as a modal/card (follow `components/Toast/ImportIssuesToast`).

## Tooltip — `ToolHint` & `InfoTip`

- `ToolHint` (`components/ToolHint`): one-line hover tip (tippy). Props
  `text`, `toolhintId`, `place`, `offset`. Delay ~150 ms, fade.
- `InfoTip` (`components/InfoTip`): richer hover card with a title + body;
  bg `infoTip.bg` (light `white`), border `surface1`, shadow
  `0 4px 12px rgba(0,0,0,.15)`.
- `Help` (see `screens/about.md`) is the click-to-open popover variant.

## Menu / context menu

- `ui/ContextMenu` (`components/ContextMenu`): right-click menu; items with
  icon, label, optional shortcut, separator, danger variant. Keyboard: ↑/↓,
  Enter activate, Esc close. Position clamped to viewport.
- `ui/MenuDropdown`: anchored menu with checkable items (workspace, theme).
- `SetAsVariableMenu`, `SetAsVariablePopover`: the "turn this value into a
  variable" context action over a field/table cell (see
  `screens/request-params.md`).

## Checkbox (`src/ui/Checkbox`)

Square check, radius 2–3, checked colour `primary.solid`, indeterminate state.
Used in tables to enable/disable a row (`screens/request-params.md`).
`examples.checkbox.color` tints the check on example rows.

## RadioButton (`components/RadioButton`)

Round radio group; selected dot `primary.solid`. Used in auth/body mode groups
alongside Select/Segmented.

## TagList / TagPicker (`components/TagList`, `components/TagPicker`)

Chips for tags. `TagPicker` = input + suggestion dropdown, `TagList` = readonly
removable chips. Colour hash per tag (stable hue) or theme accent. Used in
request Settings and runner include/exclude (`screens/runner.md`).

## ColorBadge / ColorPicker / ColorRange

- `ColorBadge`: a small swatch + optional label chip (theme previews, env colour
  dots). Reads the colour directly; falls back to a neutral swatch.
- `ColorPicker` (`components/ColorPicker/index.js`): a `Dropdown` anchored bottom-start.
  Default trigger is the current colour swatch, `title="Change color"`
  (`:109-113`). Content (`:115-160`):
  - Row 1 — a **`No color`** control (`IconBan` 20/1.5) plus six preset swatches,
    each `w-5 h-5` (20×20) rounded, `hover:scale-110`, selected gets a 2 px
    `border-current`. Presets in order: `#CE4F3B`, `#2E8A54`, `#346AB2`,
    `#C77A0F`, `#B83D7F`, `#8D44B2` (`:9-16`).
  - Row 2 — a round custom-colour chip (`w-5 h-5 rounded-full`) and a
    `ColorRange` slider (`:143-153`).
  - Picker body padding `p-2` (8 px), swatch gap `1.5` (6 px).
- `ColorRange` (`components/ColorRange`): a **discrete** slider interpolating
  across a fixed six-stop sequence `#D85D43`, `#F4BB74`, `#61DCB1`, `#7EBDF2`,
  `#D48ADE`, `#B491E5` (`index.js:18`); position 0–100 maps linearly across the
  stops (`interpolateColor`, `:40-60`). It is not an HSL hue slider.

A Go port reproduces the preset grid, the `No color` option, and the six-stop
slider; do **not** substitute a free-form hue/saturation picker.

## CodeSnippet (`components/CodeSnippet`)

Monospace block with a copy button and optional language label; syntax colours
from `codemirror.tokens` where highlighted. Used for "Copy as cURL" etc.

## ListGroup (`src/ui/ListGroup`)

Stacked list container with dividers (`border.border0`), hover rows
(`plainGrid.hoverBg`), selectable rows. Backbone of several modals.

## Accordion (`components/Accordion`)

`src/ui/ListGroup`-based collapsible sections. `Accordion` + `Accordion.Item` +
`Accordion.Header`; `defaultIndex` sets the open section. The header row is a
button (caret + label), the body expands below. Used by Cookies (one section
per domain) and any grouped settings. Keyboard: `Enter`/`Space` toggles the
focused header.

## ErrorBanner (`src/ui/ErrorBanner`) & Errors (`components/Errors`)

Inline error strip: danger tint bg `rgba(danger,.15)`, danger text, danger left
border, dismiss (`IconX`). `components/Errors` shows request/response error
lists; see `screens/response-pane.md` `ResponseErrorsCard`.

## StatusDot (`components/StatusDot`)

Small filled circle (6–8 px) indicating state: success/warning/danger/muted
from `status.*.text`. See `components/status-dot.md`.

## VariableScopeIcon / VariableScopeList

Scope icon (folder/collection/environment/request/global/runtime) and the
scope switcher list. Used by VariablesEditor and variable autocomplete. See
`screens/environments.md`.

## Spinner (`components/Spinner`)

Indeterminate ring spinner, inherits `currentColor`. See
`components/spinner.md`.

## RichTextEditor toolbar (Docs editing)

`src/ui/RichTextEditor` is TipTap-based. The toolbar
(`ui/RichTextEditor/components/EditorToolbar/index.js`) renders buttons from
`utils/editorToolbarConfig.js`; each button's `aria-label`, tooltip and label
is its tooltip string. Verbatim, in order:

| Group    | Buttons (tooltip = label)                                                       |
| -------- | ------------------------------------------------------------------------------- |
| Inline   | **Bold**, **Italic**, **Strikethrough**, **Inline code**                        |
| Block    | **Bullet list**, **Numbered list**, **Task list**, **Quote**, **Code block**    |
| Insert   | **Link**, **Table** (with a `Table` options menu, `aria-label="Table options"`) |
| History  | **Undo**, **Redo**                                                              |
| Overflow | `IconDots` → `aria-label="More formatting options"` (`:128-131`)                |

Toolbar icons use Tabler at 16/1.5 (carets 14/1.5 filled). Link edit popover
fields: **`Text`** (placeholder `Text to display`) and **`URL`** (placeholder
`https://example.com`) (`ui/RichTextEditor/components/EditorLinkEditPopover/index.js:78-99`).
A tooltip host `ToolHint` is mounted once with id `editor-toolbar-tooltip`
(`:136`).

## Open Questions

None outstanding for the primitives in this file.
