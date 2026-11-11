# Component — Toggle & Switch

Source: `src/ui/Toggle/index.js`, `components/ToggleSwitch/`, `components/ModeSwitch/`.
Port: `internal/ui/widgets/controls.go` (`Toggle` shims), plus
`choice.Switch` from Lobe.

---

## Toggle (`src/ui/Toggle`)

Controlled switch with optional label + description. `checked` drives colour:
on = `primary.solid`, off = `background.surface2` (`index.js:54`).

| Size | Track | Thumb | Off→On translate |
| --- | --- | --- | --- |
| `small` (default) | `h-4 w-8` (16×32) | `h-3 w-3` (12) | `translate-x-1` → `translate-x-4` |
| `medium` | `h-5 w-9` (20×36) | `h-3 w-3` (12) | `→ translate-x-5` |
| `large` | `h-6 w-11` (24×44) | `h-4 w-4` (16) | `→ translate-x-6` |

(`index.js:16-31`.) Label `text-xs font-medium`; description `text-xs`
muted. Layout: label block left, switch right, `justify-between`.

Disabled: `disabled` attribute, pointer-events off, reduced contrast.

## ToggleSwitch (`components/ToggleSwitch`)

Feature-local switch used by request Settings sub-tab; same visual family as
Toggle, adds an inline label row. Exact paddings live in its component.

## ModeSwitch (`components/ModeSwitch`)

A two-position segmented toggle (e.g. on/off with named sides). See its
StyledWrapper for the concrete width and active face.

## Accessibility

- Rendered as a real `<button type="button">`; use `role="switch"` +
  `aria-checked` in the Go port.
- Keyboard: `Space`/`Enter` toggles.
