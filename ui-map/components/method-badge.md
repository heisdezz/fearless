# Component — MethodBadge

Source: `src/ui/MethodBadge/` (`index.js`, `StyledWrapper.js`). Port:
`internal/ui/widgets/method_badge.go` + `internal/ui/theme/theme.go`
`MethodColor`.

---

## Behaviour

Shows an HTTP verb coloured by the theme's `request.methods` table. The method
is normalised to lowercase for the colour lookup and uppercased for display;
a missing method falls back to `colors.text.muted` and displays `GET`
(`index.js:4-6`).

| Input | Colour (light / dark) |
| --- | --- |
| GET | `#2e8a54` / `#73e89a` |
| POST | `#8d43b1` / `#79c8f6` |
| PUT | `#c67a10` / `#f6ab79` |
| DELETE | `#ce4f3b` / `#e06552` |
| PATCH | `#8d43b1` / `#f6ab79` |
| OPTIONS | `#2e8a87` / `#52e0c9` |
| HEAD | `#3089a6` / `#7ddff2` |
| unknown (TRACE, custom) | `#838383` / `#aaa` |

(`design-system.md` §1.5.)

## Sizes

| Size | Style |
| --- | --- |
| `md` (default) | inline-block, `font.size.xs` (11 px), weight 600, uppercase, fixed width **52 px**, left-aligned, no background |
| `sm` | 9 px monospace, weight 600, `padding .05rem .25rem`, radius 3 px, uppercase — a small tinted pill |

(`StyledWrapper.js:12-33`.)

## Notes

- The badge is **text-only by default** (`md`), not a filled pill. This is what
  keeps the sidebar rows reading like a list, not a wall of coloured chips.
- Sidebar at the pinned commit uses a `min-width: 34px` text slot instead of
  this component (see `screens/sidebar.md` §Open Questions 4); the request
  editor and tabs use the badge.
