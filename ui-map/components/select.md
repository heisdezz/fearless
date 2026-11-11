# Component — Select & Dropdown

Two related primitives: `src/ui/Select/` (form select) and
`components/Dropdown/` (menu/anchor popover used by Select and menus).

---

## Select (`src/ui/Select/index.js`)

Trigger is an `InputWrapper` (field shell) with a caret (`IconCaretDown`),
optional left section, optional clear (`IconX`), optional loading
(`IconLoader2`). Popup is a tippy-based `Dropdown` whose width matches the
trigger (`sameWidthModifier`, `ui/Select/index.js:33-45`).

| Prop                                     | Default                | Notes                                                        |
| ---------------------------------------- | ---------------------- | ------------------------------------------------------------ |
| `data`                                   | —                      | `string[]` or `{value,label,disabled?}[]` (`ui/Select/index.js:20-31`) |
| `value` / `onChange`                     | —                      | controlled, value is a string                                |
| `placeholder`                            | **`Select...`**        | shown when empty                                             |
| `searchable`                             | false                  | type-to-filter                                               |
| `nothingFoundMessage`                    | **`No options found`** | search empty state                                           |
| `clearable`                              | false                  | clear button when a value is set                             |
| `allowDeselect`                          | true                   | click selected option to clear                               |
| `maxDropdownHeight`                      | 250 px                 |                                                              |
| `label`, `description`, `error`          | —                      | rendered above/below trigger                                 |
| `loading`, `leftSection`, `rightSection` | —                      | adornments                                                   |

## Trigger shell (shared with text inputs)

Select uses the same `InputWrapper` shell as text inputs, sized from
`src/ui/InputWrapper/constants.js` `INPUT_SIZES`:

| Size           | padding                        | font-size    | radius            |
| -------------- | ------------------------------ | ------------ | ----------------- |
| `sm`           | `0.15rem 0.4rem` (≈2.4×6.4 px) | `xs` (11 px) | `radius.sm` (4)   |
| `md` (default) | `0.45rem` (≈7.2 px)            | `sm` (12 px) | `radius.base` (6) |

The trigger is `width: 100%`, ellipsised, placeholder at opacity `0.5`; when
open the border becomes `input.focusBorder` (`input.focusBorder`).

Keyboard: `ArrowDown/ArrowUp/Home/End` move the active option, `Enter` selects,
`Escape` closes (`ui/Select/index.js:8-18`). Active option keys wrap cyclically.

## Dropdown (`components/Dropdown/`)

Anchor-positioned popover (tippy). Used for menus, the theme picker, workspace
switcher, and Select's popup. Tokens: `dropdown.bg`, `dropdown.hoverBg`,
`dropdown.separator` (`border1`), `dropdown.selectedColor` (`primary.text`),
`dropdown.mutedText` (`subtext0`). Shadow `dropdown.shadow`
(`0 0px 3px rgba(0,0,0,.12), 0 0 0 1px rgba(0,0,0,.05)`), border `none`
(`themes/light/light.js` `dropdown`).

Variants: `src/ui/MenuDropdown` (menu with icons, separators, checkmarks),
`components/FilterDropdown` (multi-select filters), `components/Dropdown` (raw).

## Interaction summary

| Key      | Select           | Menu           |
| -------- | ---------------- | -------------- |
| ↓/↑      | next/prev option | next/prev item |
| Home/End | first/last       | first/last     |
| Enter    | choose           | activate       |
| Esc      | close            | close          |

## Open Questions

None outstanding: the trigger shell and sizing are in § Trigger shell above.
