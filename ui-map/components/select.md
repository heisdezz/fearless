# Component — Select & Dropdown

Two related primitives: `src/ui/Select/` (form select) and
`components/Dropdown/` (menu/anchor popover used by Select and menus).

---

## Select (`src/ui/Select/index.js`)

Trigger is an `InputWrapper` (field shell) with a caret (`IconCaretDown`),
optional left section, optional clear (`IconX`), optional loading
(`IconLoader2`). Popup is a tippy-based `Dropdown` whose width matches the
trigger (`sameWidthModifier`, `index.js:33-45`).

| Prop | Default | Notes |
| --- | --- | --- |
| `data` | — | `string[]` or `{value,label,disabled?}[]` (`index.js:20-31`) |
| `value` / `onChange` | — | controlled, value is a string |
| `placeholder` | **`Select...`** | shown when empty |
| `searchable` | false | type-to-filter |
| `nothingFoundMessage` | **`No options found`** | search empty state |
| `clearable` | false | clear button when a value is set |
| `allowDeselect` | true | click selected option to clear |
| `maxDropdownHeight` | 250 px | |
| `label`, `description`, `error` | — | rendered above/below trigger |
| `loading`, `leftSection`, `rightSection` | — | adornments |

Keyboard: `ArrowDown/ArrowUp/Home/End` move the active option, `Enter` selects,
`Escape` closes (`index.js:8-18`). Active option keys wrap cyclically.

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

| Key | Select | Menu |
| --- | --- | --- |
| ↓/↑ | next/prev option | next/prev item |
| Home/End | first/last | first/last |
| Enter | choose | activate |
| Esc | close | close |

## Open Questions

- Select's exact trigger padding/height follow `InputWrapper`; confirm against
  `components/text-input.md` (same shell).
