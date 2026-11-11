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

## Toast (`components/Toast`, react-hot-toast)

Transient message bottom-right. Types: success (green), error (danger),
loading (spinner), blank (neutral). Auto-dismiss ~4 s; stackable; each has a
close affordance. Copy is per-call. `feedback.ToastHost` is the Go port.

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

- `ColorBadge`: a swatch + label chip (theme previews, env colour dots).
- `ColorPicker`: popover with hue/saturation area + hex input (custom scenes).
- `ColorRange`: a gradient/hue slider used inside ColorPicker.
  Uses `colors.accent` and the theme palette.

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

## Open Questions

- Several small components (Toast visuals, ColorPicker geometry) were not
  read line-by-line; refine from source when building each.
