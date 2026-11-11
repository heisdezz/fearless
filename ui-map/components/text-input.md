# Component — text inputs (text field, search, masked, textarea)

Sources: `src/ui/InputWrapper/`, `src/ui/TextInput/`, `src/ui/MaskedInput/`,
`src/ui/MaskedTextfield/`, `components/SearchInput/`, `components/SettingsInput/`,
`components/InheritableSettingsInput/`, `components/FilePickerEditor/`,
`components/PathDisplay/`, `components/TruncatedText/`,
`components/SensitiveFieldWarning/`.

Port: `internal/ui/widgets/controls.go` `FieldInput` (wraps Lobe `field.Input`
on an app-owned surface — see the file comment for why).

---

## Field shell (`InputWrapper`)

The common box: bg `input.bg` (light `#ffffff`, dark `transparent`), border
`1px input.border` (light `#cccccc`, dark `#444444`), radius `border.radius.base`
(6), focus border `input.focusBorder` (light `#8b8b8b`, dark
`rgba(228,174,73,.8)`), placeholder `input.placeholder.color` at
`input.placeholder.opacity` (light `#b0b0b0`/0.8, dark `#aaa`/0.6).

**Dark inputs are transparent on a panel**, not white — a Go port must paint the
field on the panel ground (that is exactly the D-017 white-input fix).

## TextInput

Single-line, optional prefix/suffix, error text below. Height follows size
(`sm` 24, `base` 32, `lg` 40 px — see `components/button.md` size rhythm).

## SearchInput (`components/SearchInput`)

Text field with a leading `IconSearch` and a clear (`IconX`) when non-empty.
Sidebar filter uses this; placeholder **`Filter requests...`** appears in the
sidebar (see `screens/sidebar.md`).

## Masked / password (`MaskedInput`, `MaskedTextfield`)

Reveal toggle (eye icon), value masked by default. Used for tokens/secret
fields. Pair with `SensitiveFieldWarning`.

## SensitiveFieldWarning

Inline caution next to a field that will be stored in plaintext in the
collection (e.g. secrets in a committed environment). Icon `IconAlertTriangle`
+ short copy; see `screens/environments.md` § secret variables.

## Textarea

Multi-line field shell; `MinRows`/`MaxRows` clamp. Used by modal prompts and
the script/body fallback (see `widgets.md`).

## SettingsInput / InheritableSettingsInput

Settings-styled field. `InheritableSettingsInput` shows a "placeholders from
parent" hint and a reset-to-inherited affordance (folder/collection settings;
see `screens/collection-settings.md`).

## FilePickerEditor / PathDisplay

`FilePickerEditor`: read-only path + Browse button (file or folder). `PathDisplay`
shortens long paths with a middle ellipsis and full path on hover.

## TruncatedText

Single-line clamp with ellipsis and a tooltip carrying the full value; used in
sidebar rows and tables.

## Open Questions

- `SettingsInput` vs `TextInput` exact visual delta (padding/border) not read
  line-by-line; both sit on the field shell above.
