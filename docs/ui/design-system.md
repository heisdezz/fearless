# Design System

Status: VERIFIED against `bruno/packages/bruno-app/src/themes/` at commit `60d7517` and machine-verified via `ui-map/_tools/dump-theme.mjs` (output: `ui-map/_tools/tokens.json`). Cross-referenced with the full Bruno reconstruction specification in `ui-map/design-system.md`.

---

## 1. Foundations

| Concern        | Choice                                                                                                                                                  | Source                                                            |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Styling engine | **styled-components** (CSS-in-JS) + a small Tailwind setup                                                                                              | `src/globalStyles.js` (`createGlobalStyle`), `tailwind.config.js` |
| Tailwind role  | Minimal: `darkMode: ['class']`, `content: ['./src/**/*.{js,jsx}']`, no custom theme extension. Most styling is styled-components reading `props.theme`. | `tailwind.config.js`                                              |
| Font           | Inter                                                                                                                                                   | `@fontsource/inter` in `bruno-app/package.json`                   |
| Icons          | Tabler Icons v1                                                                                                                                         | `@tabler/icons`                                                   |
| Theming        | ~20 themes, light + dark + presets                                                                                                                      | `src/themes/index.js`                                             |
| Base font size | `0.8125rem` (13 px)                                                                                                                                     | `themes/light/light.js` `font.size`                               |

**Design tokens are a TypeScript/JS object validated by JSON Schema**, visible in `themes/schema/oss.js` and previewed in `themes/DesignSystem/DesignSystem.stories.jsx` (Storybook).

Source of truth:

- Light theme default: `themes/light/light.js` (default export `lightTheme`, palette at `:2`).
- Dark theme default: `themes/dark/dark.js` (default export `darkTheme`, palette at `:3`).
- Schema validation: `themes/schema/oss.js`.
- Machine-verifiable token dump: `ui-map/_tools/dump-theme.mjs` -> `ui-map/_tools/tokens.json`.

---

## 2. Concrete Theme Tokens (VERIFIED)

Every colour below is resolved to its concrete hex or rgba value from the pinned source. "L" = Light default, "D" = Dark default. Values keeping `rgba(...)` are genuinely translucent in source and retain their alpha channel.

### 2.1 Brand, text, background

| Token                    | L         | D         | Use                               |
| ------------------------ | --------- | --------- | --------------------------------- |
| `primary.solid`          | `#d37f17` | `#e4ae49` | Primary buttons, active accent    |
| `primary.text`           | `#c07a26` | `#e7b85f` | Accent text, links on chrome      |
| `primary.strong`         | `#d5882a` | `#e7b85f` | Active tab underline, bold accent |
| `primary.subtle`         | `#dc9741` | `#e1a433` | Subtle accent                     |
| `accents.primary`        | `#d37f17` | `#e4ae49` | Alias of `primary.solid`          |
| `brand`                  | `#d37f17` | `#e4ae49` | Alias of `primary.solid`          |
| `text` / `text.base`     | `#343434` | `#cccccc` | Default body text                 |
| `textLink`               | `#346ab2` | `#8bc2f9` | Hyperlinks                        |
| `draftColor`             | `#cc7b1b` | `#cc7b1b` | Unsaved/draft marker (same both)  |
| `bg` / `background.base` | `#ffffff` | `#1a1a1a` | Window/app ground                 |
| `background.mantle`      | `#f8f8f8` | `#222224` | Sidebar, secondary panels         |
| `background.crust`       | `#f6f6f6` | `#333333` | Tab strip, tertiary panels        |
| `background.surface0`    | `#f1f1f1` | `#26292b` | Raised rows, modal title bar      |
| `background.surface1`    | `#eaeaea` | `#444444` | Hover face, striped row           |
| `background.surface2`    | `#e5e5e5` | `#666666` | Strongest neutral face            |
| `colors.accent`          | `#b96f1d` | `#d9a342` | System control accent             |

Source: `light.js:1-75`, `dark.js:1-70`.

### 2.2 Text ramp

| Token                            | L         | D         | Usage                            |
| -------------------------------- | --------- | --------- | -------------------------------- |
| `colors.text.white`              | `#ffffff` | `#cccccc` | High-contrast text on dark fills |
| `colors.text.subtext2`           | `#666666` | `#bbbbbb` | Secondary labels, badges         |
| `colors.text.subtext1` (`muted`) | `#838383` | `#aaaaaa` | Muted descriptions               |
| `colors.text.subtext0`           | `#9b9b9b` | `#999999` | Placeholders, inactive icons     |
| `colors.text.danger`             | `#ce4f3b` | `#da462f` | Error text, destructive labels   |
| `colors.text.warning`            | `#c67a10` | `#f6ab79` | Warning text, attention badges   |
| `colors.text.green`              | `#2e8a54` | `#73e89a` | Success text, 2xx meters         |
| `colors.text.purple`             | `#8d43b1` | `#d691ed` | Scripts, special tokens          |
| `colors.text.yellow`             | `#bb931b` | `#fad075` | Highlights                       |
| `colors.bg.danger`               | `#ce4f3b` | `#da462f` | Danger badge backgrounds         |

Note: `colors.text.muted` is explicitly aliased to `subtext1`.

### 2.3 Borders and overlays

| Token              | L         | D         | Usage                         |
| ------------------ | --------- | --------- | ----------------------------- |
| `border.border0`   | `#efefef` | `#2a2a2a` | Faintest dividers             |
| `border.border1`   | `#e5e5e5` | `#333333` | Standard element borders      |
| `border.border2`   | `#cccccc` | `#444444` | Inputs, active card borders   |
| `border.border3`   | `#b3b3b3` | `#555555` | Strongest interactive outline |
| `border.subtle`    | `#e5e5e5` | `#333333` | Subtle panels and separators  |
| `overlay.overlay0` | `#c0c0c0` | `#444444` | Light overlay tint            |
| `overlay.overlay1` | `#b0b0b0` | `#555555` | Medium overlay tint           |
| `overlay.overlay2` | `#8b8b8b` | `#666666` | Strong overlay tint           |

Hairlines are strictly 1 px.

### 2.4 Status colours

Each status provides `background` (15 % alpha tint), `text`, and `border`:

| Status  | L bg / text / border                           | D bg / text / border                            |
| ------- | ---------------------------------------------- | ----------------------------------------------- |
| info    | `rgba(52,106,178,.15)` / `#346ab2` / `#346ab2` | `rgba(139,194,249,.15)` / `#8bc2f9` / `#8bc2f9` |
| success | `rgba(46,138,84,.15)` / `#2e8a54` / `#2e8a54`  | `rgba(115,232,154,.15)` / `#73e89a` / `#73e89a` |
| warning | `rgba(198,122,16,.15)` / `#c67a10` / `#c67a10` | `rgba(246,171,121,.15)` / `#f6ab79` / `#f6ab79` |
| danger  | `rgba(206,79,59,.15)` / `#ce4f3b` / `#ce4f3b`  | `rgba(218,70,47,.15)` / `#da462f` / `#da462f`   |

### 2.5 HTTP method colours (`request.methods`)

Per-method colours are **tokens**, so HTTP verbs render consistently across themes and screens:

| Method                  | L                               | D         | Notes                            |
| ----------------------- | ------------------------------- | --------- | -------------------------------- |
| GET                     | `#2e8a54`                       | `#73e89a` | Emerald / mint green             |
| POST                    | `#8d43b1`                       | `#79c8f6` | Purple (L) / Indigo sky-blue (D) |
| PUT                     | `#c67a10`                       | `#f6ab79` | Amber / warm orange              |
| DELETE                  | `#ce4f3b`                       | `#e06552` | Crimson / coral red              |
| PATCH                   | `#8d43b1`                       | `#f6ab79` | Purple (L) / Warm orange (D)     |
| OPTIONS                 | `#2e8a87`                       | `#52e0c9` | Teal / cyan                      |
| HEAD                    | `#3089a6`                       | `#7ddff2` | Sky blue / light cyan            |
| Fallback (unknown verb) | `#838383` (`colors.text.muted`) | `#aaaaaa` | Muted neutral                    |

_(Verified against `themes/light/light.js`, `themes/dark/dark.js`, and asserted by `internal/ui/theme/theme_test.go:57`)_.

---

## 3. Typography & Spacing

### 3.1 Font sizes

| Token            | Value (rem) | px (at 16 px base) | Usage                               |
| ---------------- | ----------- | ------------------ | ----------------------------------- |
| `font.size.xs`   | `0.6875rem` | 11                 | Request method labels, fine badges  |
| `font.size.sm`   | `0.75rem`   | 12                 | Status bar, table secondary text    |
| `font.size.base` | `0.8125rem` | **13 (default)**   | Standard UI text, labels, inputs    |
| `font.size.md`   | `0.875rem`  | 14                 | Modal titles, section titles        |
| `font.size.lg`   | `1rem`      | 16                 | Hero text, collection headings      |
| `font.size.xl`   | `1.125rem`  | 18                 | Welcome banners, large modal titles |

- **UI face:** Inter (`@fontsource/inter`).
- **Monospace face:** `SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace` for code editors, numeric meters, status codes, URLs, and timing.
- **Weights:** 400 (regular body), 500 (medium, buttons/labels), 600 (semibold, section headings, badges).

### 3.2 Spacing scale

Observed 4-based usage rhythm across components:

- `2px`: tight padding inside badges, chips
- `4px`: icon-to-label gap, small padding
- `6px`: row gaps, table cell padding
- `8px`: control padding, button padding
- `10px`: section padding
- `12px`: panel padding, pane dividers
- `16px`: large section gaps
- `24px`: major vertical rhythm

### 3.3 Border radius

| Token                | Value  | Usage                          |
| -------------------- | ------ | ------------------------------ |
| `border.radius.sm`   | `4px`  | Badges, tags, small keycaps    |
| `border.radius.base` | `6px`  | Inputs, buttons, dropdown rows |
| `border.radius.md`   | `8px`  | Cards, popovers                |
| `border.radius.lg`   | `10px` | Dialogs, modals                |
| `border.radius.xl`   | `12px` | Outer frames                   |

### 3.4 Shadows

| Token       | Light                                                   | Dark                                                  |
| ----------- | ------------------------------------------------------- | ----------------------------------------------------- |
| `shadow.sm` | `0 1px 3px rgba(0,0,0,.12), 0 0 0 1px rgba(0,0,0,.05)`  | `0 1px 3px rgba(0,0,0,.5), 0 0 0 1px rgba(0,0,0,.3)`  |
| `shadow.md` | `0 2px 8px rgba(0,0,0,.14), 0 0 0 1px rgba(0,0,0,.06)`  | `0 2px 8px rgba(0,0,0,.6), 0 0 0 1px rgba(0,0,0,.4)`  |
| `shadow.lg` | `0 2px 12px rgba(0,0,0,.15), 0 0 0 1px rgba(0,0,0,.05)` | `0 2px 12px rgba(0,0,0,.7), 0 0 0 1px rgba(0,0,0,.4)` |

---

## 4. Component Surfaces (VERIFIED Token Groups)

### 4.1 Input tokens

| Token                       | L         | D                     | Notes                                      |
| --------------------------- | --------- | --------------------- | ------------------------------------------ |
| `input.bg`                  | `#ffffff` | `transparent`         | Dark inputs are transparent over container |
| `input.border`              | `#cccccc` | `#444444`             | Standard border                            |
| `input.focusBorder`         | `#8b8b8b` | `rgba(228,174,73,.8)` | Accent focus ring                          |
| `input.placeholder.color`   | `#b0b0b0` | `#aaaaaa`             | Muted hint text                            |
| `input.placeholder.opacity` | `0.8`     | `0.6`                 | Opacity multiplier                         |

_(Note: Decision R-6 establishes that `input.background` in source was a bug; `input.bg` is canonical)_.

### 4.2 Modal tokens

| Token                    | L         | D         |
| ------------------------ | --------- | --------- |
| `modal.title.bg`         | `#f1f1f1` | `#1a1a1a` |
| `modal.title.color`      | `#343434` | `#cccccc` |
| `modal.body.bg`          | `#ffffff` | `#222224` |
| `modal.backdrop.opacity` | `0.4`     | `0.4`     |

### 4.3 Sidebar, dropdown, and request tabs

| Token                                  | L         | D         | Notes                          |
| -------------------------------------- | --------- | --------- | ------------------------------ |
| `sidebar.bg`                           | `#f8f8f8` | `#222224` | Frame sidebar surface          |
| `sidebar.color`                        | `#343434` | `#cccccc` | Main sidebar text              |
| `sidebar.muted`                        | `#838383` | `#aaaaaa` | Secondary sidebar text         |
| `sidebar.dragbar.border`               | `#e5e5e5` | `#666666` | Splitter hairline              |
| `sidebar.collection.item.bg`           | `#eaeaea` | `#444444` | Active item background         |
| `sidebar.collection.item.indentBorder` | `#e5e5e5` | `#333333` | Tree hierarchy vertical line   |
| `dropdown.bg`                          | `#ffffff` | `#222224` | Popover face                   |
| `dropdown.hoverBg`                     | `#f6f6f6` | `#333333` | Selected / hovered item        |
| `dropdown.separator`                   | `#e5e5e5` | `#333333` | Menu item divider              |
| `dropdown.selectedColor`               | `#c07a26` | `#e7b85f` | Active check / highlight       |
| `requestTabs.bg`                       | `#f6f6f6` | `#26292b` | Tab strip ground               |
| `requestTabs.bottomBorder`             | `#efefef` | `#2a2a2a` | Tab strip baseline             |
| `requestTabs.icon.color`               | `#9b9b9b` | `#9f9f9f` | Close icon neutral             |
| `requestTabs.icon.hoverColor`          | `#343434` | `#cccccc` | Close icon active              |
| `requestTabs.icon.hoverBg`             | `#eaeaea` | `#1e1e1e` | Close icon circular hover disc |

### 4.4 Response pane tokens

| Token                               | L                      | D                   |
| ----------------------------------- | ---------------------- | ------------------- |
| `requestTabPanel.responseStatus`    | `#838383`              | `#cccccc`           |
| `requestTabPanel.responseOk`        | `#2e8a54`              | `#73e89a`           |
| `requestTabPanel.responseError`     | `#ce4f3b`              | `#da462f`           |
| `requestTabPanel.responsePending`   | `#346ab2`              | `#8bc2f9`           |
| `requestTabPanel.responseOverlayBg` | `rgba(255,255,255,.6)` | `rgba(26,26,26,.8)` |

---

## 5. Reusable Component Inventory (`src/ui`)

| Primitive                                               | Notes                                                      | Reference Spec                         |
| ------------------------------------------------------- | ---------------------------------------------------------- | -------------------------------------- |
| `Button`                                                | Sizes xs (20), sm (24), md (30), lg (38); variants         | `ui-map/components/button.md`          |
| `Select`                                                | Custom dropdown with chevron, search filter                | `ui-map/components/select.md`          |
| `MenuDropdown`                                          | Row context menus, items with icons & shortcuts            | `ui-map/components/select.md`          |
| `ContextMenu`                                           | Right-click context menus with coordinates                 | `ui-map/components/select.md`          |
| `Modal`                                                 | Focus trap (`useFocusTrap`), Esc close, portal host        | `ui-map/screens/sidebar-dialogs.md`    |
| `Toggle`, `Checkbox`, `RadioButton`                     | Form controls with animated switches                       | `ui-map/components/toggle.md`          |
| `TextInput`, `MaskedInput`, `FieldInput`                | Single/multiline with secret masking (R-5: 1 `*` per char) | `ui-map/components/text-input.md`      |
| `Tabs` / `ResponsiveTabs`                               | Overflow chevron, R-2 no hover tint, R-3 italic preview    | `ui-map/components/tabs.md`            |
| `RequestMethod` vs `MethodBadge`                        | R-1: Sidebar uses `RequestMethod` (11px, 34px fixed slot)  | `ui-map/components/method-badge.md`    |
| `StatusBadge`, `StatusDot`, `CountBadge`, `Spinner`     | Status & activity indicators                               | `ui-map/components/status-*.md`        |
| `ColorPicker`, `ColorBadge`, `CodeSnippet`, `Accordion` | Misc primitives                                            | `ui-map/components/misc-primitives.md` |

---

## 6. Resolved Reconstruction Decisions (R-1 through R-7)

From `ui-map/decisions.md`:

- **R-1 (Sidebar Verbs):** Sidebar rows use `RequestMethod` (`11px`, 34 px slot, verbs > 5 characters truncated to 3: DEL, OPT; protocol tags for grpc/ws/gql), not `MethodBadge`.
- **R-2 (Tab Hover):** Request tabs have no background hover tint; only the close button has a hover disc.
- **R-3 (Preview Tabs):** Preview/transient tabs are styled with italic text only; no preview color token.
- **R-4 (Assertion Operators):** Operators retain raw identifiers (`equals`, `notEquals`, `contains`, `matches`, `startsWith`, `endsWith`, `length`, `empty`, `null`, `type`, etc.).
- **R-5 (Secret Masking):** Secret masking displays one `*` per character everywhere in the UI.
- **R-6 (Input Background Token):** `input.background` in source was a theme schema omission; `input.bg` is the canonical token.
- **R-7 (Global Shortcuts):** `Cmd/Ctrl + J` layout toggling is a global hotkey bound in `providers/Hotkeys/index.js:386`.

---

## 7. Go Implementation Architecture

### 7.1 Graphite Console World & Token Plumbing (D-017, D-018)

The Go port builds upon the **Graphite Console** runtime token system (`internal/ui/theme`):

- **Dark-first appearance (D-018):** Dark mode is the primary shipped look; light mode is a complete companion.
- **Palette and elevations:** Ground `#141518`, seam-only elevation (never drop shadows), hairline borders `#24262B` at 1 px, single accent `#4D8DFF` earned by primary actions.
- **Per-method HTTP colors:** Retained as a first-class lookup table (`theme.MethodColorFor(verb)`), matching Bruno's exact token tables.

### 7.2 Upstream Control Shimming (D-019)

Upstream Lobe/MyGo components (`field.Input`, `field.Select`, `action.Button`) paint backgrounds on a detached container node that is not returned, causing controls to paint on MyGo's default near-white surface.

**Application Rule (D-019):**
No screen calls raw Lobe controls directly. Screens import `internal/ui/widgets/controls.go`:

- `widgets.FieldInput`
- `widgets.FieldSelect`
- `widgets.Button`

These controls are painted on app-owned, token-styled surfaces with borderless inner Lobe controls.

### 7.3 Token Bridge API

```go
package theme

import (
    "image/color"
    "github.com/egoist/mygo/ui"
    lobeTheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// MethodColorFor returns the Bruno per-method color for a given HTTP verb.
func MethodColorFor(method string, dark bool) color.NRGBA

// TokensOf resolves the active world tokens for the current context.
func TokensOf(c *ui.Context) *lobeTheme.Tokens
```

All screens read tokens via `theme.TokensOf(c)` or `theme.MethodColorFor(method, dark)`. Hard-coded colors are prohibited.
