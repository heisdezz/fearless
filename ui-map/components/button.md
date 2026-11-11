# Component — Button

Source: `src/ui/Button/` (`index.js`, `StyledWrapper.js`). The app's one button
primitive; `docs/ui/components.md` §Button and `internal/ui/widgets/controls.go`
are the Go ports.

---

## Props

| Prop | Values | Default |
| --- | --- | --- |
| `size` | `xs`, `sm`, `base`, `md`, `lg` | `base` |
| `variant` | `filled`, `outline`, `ghost` | `filled` |
| `color` | `primary`, `light`, `secondary`, `success`, `warning`, `danger` | `primary` |
| `rounded` | `sm`, `base`, `md`, `lg`, `full` | `base` |
| `fontWeight` | `regular` (400), `medium` (500) | 400 |
| `icon`, `iconPosition` | element, `left` \| `right` | — / `left` |
| `disabled`, `loading`, `fullWidth` | bool | false |

(`index.js:4-17`.)

## Sizes (CSS → px at 16 px root)

| size | padding | font-size | gap | icon |
| --- | --- | --- | --- | --- |
| xs | `0.25rem 0.5rem` (4×8) | 11 px (`font.size.xs`) | 4 | 12 px |
| sm | `0.375rem 0.75rem` (6×12) | 12 px (`sm`) | 6 | 14 px |
| base | `0.5rem 1rem` (8×16) | 12 px (`sm`) | 8 | 16 px |
| md | `0.625rem 1.125rem` (10×18) | 12 px (`sm`) | 8 | 16 px |
| lg | `0.75rem 1.5rem` (12×24) | 13 px (`base`) | 12 | 18 px |

(`StyledWrapper.js:14-94`.) Radius from `border.radius.*`; `full` = 9999 px.

## Colours (from `button2.color.<name>`)

| color | bg | text | border |
| --- | --- | --- | --- |
| primary | `primary.solid` `#d37f17`/`#e4ae49` | `#ffffff` | `primary.solid` |
| light | `rgba(primary.solid,.08)` | `primary.solid` | `rgba(primary.solid,.06)` |
| secondary | `background.mantle` `#f8f8f8`/`#222224` | `text` | `border2` |
| success | `#2e8a54`/`#73e89a` | `#fff` | same as bg |
| warning | `#c67a10`/`#f6ab79` | `#fff` | same |
| danger | `#ce4f3b`/`#da462f` | `#fff` | same |

(`design-system.md`; `themes/light/light.js` `button2`.)

## Variant behaviour

- **filled**: bg/border/text as above. Hover darkens bg+border 3 %, active
  darkens 7 % (`StyledWrapper.js:127-145`).
- **outline**: transparent bg, colour+border = the colour's *display colour*
  (bg for most, text for `secondary`). Hover bg = display colour at 5 %,
  active 10 % (`:148-168`).
- **ghost**: transparent, transparent border. Hover bg = display colour 10 %,
  active 15 % (`:171-190`).
- Focus-visible: `box-shadow 0 0 0 2px` of the display colour at 40 %.
- Disabled: `opacity .7`, `cursor: not-allowed`, pointer-events off.
- Loading: leading spinner (`strokeDasharray 31.4 31.4`, rotating), icon hidden
  while loading (`index.js:49-60`).
- Transition: `all .15s ease` (`StyledWrapper.js:220`).

## Open Questions

- Closed-button (`components/.../CloseButton`) variants are feature-local; see
  their screen files.
