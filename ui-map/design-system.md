# Design system — tokens, type, colour, motion

Status: VERIFIED against `bruno/packages/bruno-app/src/themes/`. Every colour
below was resolved from source to a concrete value by
`ui-map/_tools/dump-theme.mjs` (re-run: `cd bruno && node ../ui-map/_tools/dump-theme.mjs`;
output in `ui-map/_tools/tokens.json`). Values that stay as `rgba(...)` are
genuinely translucent in source and must keep their alpha.

Source of truth:

- Light: `themes/light/light.js` (default export `lightTheme`, palette at `:2`).
- Dark: `themes/dark/dark.js` (default export `darkTheme`, palette at `:3`).
- Schema: `themes/schema/oss.js` validates the token shape.
- Themes are consumed through styled-components `props.theme`; a Go port reads
  these tables directly (see `docs/ui/design-system.md` §7 for the mapping to
  Lobe tokens).

**Every other theme is a variant of these two.** `themes/light/` and
`themes/dark/` hold 11 + 10 presets (catppuccin, dracula, gruvbox, nord,
vscode, cyberdream, _-pastel, _-monochrome). They share the same token keys and
differ only in palette values; `light.js`/`dark.js` are the defaults. A Go port
needs these two precise; presets are data (see §9).

---

## 1. Colour — semantic tokens

Both themes, resolved. "L" = light, "D" = dark.

### 1.1 Brand, text, background

| Token                    | L         | D         | Use                               |
| ------------------------ | --------- | --------- | --------------------------------- |
| `primary.solid`          | `#d37f17` | `#e4ae49` | Primary buttons, active accent    |
| `primary.text`           | `#c07a26` | `#e7b85f` | Accent text, links on chrome      |
| `primary.strong`         | `#d5882a` | `#e7b85f` | Active tab underline, bold accent |
| `primary.subtle`         | `#dc9741` | `#e1a433` | Subtle accent                     |
| `accents.primary`        | `#d37f17` | `#e4ae49` | alias of `primary.solid`          |
| `brand`                  | `#d37f17` | `#e4ae49` | alias of `primary.solid`          |
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

Source: `light.js:1-75` (palette), `light.js` `lightTheme.primary/background/text`;
`dark.js:1-70`.

### 1.2 Text ramp

| Token                            | L         | D         |
| -------------------------------- | --------- | --------- |
| `colors.text.white`              | `#ffffff` | `#cccccc` |
| `colors.text.subtext2`           | `#666666` | `#bbb`    |
| `colors.text.subtext1` (`muted`) | `#838383` | `#aaa`    |
| `colors.text.subtext0`           | `#9b9b9b` | `#999`    |
| `colors.text.danger`             | `#ce4f3b` | `#da462f` |
| `colors.text.warning`            | `#c67a10` | `#f6ab79` |
| `colors.text.green`              | `#2e8a54` | `#73e89a` |
| `colors.text.purple`             | `#8d43b1` | `#d691ed` |
| `colors.text.yellow`             | `#bb931b` | `#fad075` |
| `colors.bg.danger`               | `#ce4f3b` | `#da462f` |

Note `colors.text.muted` is aliased to `subtext1`.

### 1.3 Borders, overlays

| Token              | L         | D         |
| ------------------ | --------- | --------- |
| `border.border0`   | `#efefef` | `#2a2a2a` |
| `border.border1`   | `#e5e5e5` | `#333333` |
| `border.border2`   | `#cccccc` | `#444444` |
| `border.border3`   | `#b3b3b3` | `#555555` |
| `border.subtle`    | `#e5e5e5` | `#333333` |
| `overlay.overlay0` | `#c0c0c0` | `#444444` |
| `overlay.overlay1` | `#b0b0b0` | `#555555` |
| `overlay.overlay2` | `#8b8b8b` | `#666666` |

Hairlines are 1 px. `border0` is the faintest divider, `border3` the strongest
interactive outline.

### 1.4 Status colours

Each status has `background` (15 % tint), `text`, `border`.

| Status  | L bg / text / border                           | D bg / text / border                            |
| ------- | ---------------------------------------------- | ----------------------------------------------- |
| info    | `rgba(52,106,178,.15)` / `#346ab2` / `#346ab2` | `rgba(139,194,249,.15)` / `#8bc2f9` / `#8bc2f9` |
| success | `rgba(46,138,84,.15)` / `#2e8a54` / `#2e8a54`  | `rgba(115,232,154,.15)` / `#73e89a` / `#73e89a` |
| warning | `rgba(198,122,16,.15)` / `#c67a10` / `#c67a10` | `rgba(246,171,121,.15)` / `#f6ab79` / `#f6ab79` |
| danger  | `rgba(206,79,59,.15)` / `#ce4f3b` / `#ce4f3b`  | `rgba(218,70,47,.15)` / `#da462f` / `#da462f`   |

### 1.5 HTTP method colours (`request.methods`)

Per-method colours are **tokens**, so verbs read the same across themes. Value
is drawn from the theme palette (`light.js` `request.methods`, `dark.js` same).

| Method                  | L                               | D         |
| ----------------------- | ------------------------------- | --------- |
| GET                     | `#2e8a54`                       | `#73e89a` |
| POST                    | `#8d43b1`                       | `#79c8f6` |
| PUT                     | `#c67a10`                       | `#f6ab79` |
| DELETE                  | `#ce4f3b`                       | `#e06552` |
| PATCH                   | `#8d43b1`                       | `#f6ab79` |
| OPTIONS                 | `#2e8a87`                       | `#52e0c9` |
| HEAD                    | `#3089a6`                       | `#7ddff2` |
| fallback (unknown verb) | `#838383` (`colors.text.muted`) | `#aaa`    |

Note light: POST and PATCH share purple; dark: POST is indigo while PUT and
PATCH share orange. These exact pairs are asserted by
`internal/ui/theme/theme_test.go:57` in this repo.

---

## 2. Typography

| Token            | Value (rem) | px at 16 px root |
| ---------------- | ----------- | ---------------- |
| `font.size.xs`   | `0.6875rem` | 11               |
| `font.size.sm`   | `0.75rem`   | 12               |
| `font.size.base` | `0.8125rem` | **13 (default)** |
| `font.size.md`   | `0.875rem`  | 14               |
| `font.size.lg`   | `1rem`      | 16               |
| `font.size.xl`   | `1.125rem`  | 18               |

- **UI face:** Inter (`@fontsource/inter`, `packages/bruno-app/package.json`).
  Fallback stack is the browser default sans.
- **Mono face:** the code editor and every numeric/code value use the editor's
  monospace stack (CodeMirror default: `SFMono-Regular, Menlo, Consolas,
"Liberation Mono", monospace`). See `widgets.md` § editor.
- Weights observed: 400 body, 500 medium (buttons, labels), 600 bold (section
  headings, method badges read as bold-ish). Tab active uses `fontWeight: 400`
  (`light.js` `tabs.active`).
- Line-height is not a token; components set it locally (typically 1.4–1.5 for
  body, tighter for badges).

---

## 3. Spacing

No spacing scale token exists. Components use literal `px`/`rem`. The observed
set, which a Go port should adopt as a scale (4-based):

| Step | px  | Observed use                              |
| ---- | --- | ----------------------------------------- |
| 1    | 2   | tight gaps inside badges, key/value chips |
| 2    | 4   | icon-to-label gap, small padding          |
| 3    | 6   | row gap, cell padding                     |
| 4    | 8   | control padding, group gap                |
| 5    | 10  | section padding                           |
| 6    | 12  | panel padding                             |
| 8    | 16  | large gaps                                |
| 10   | 24  | section vertical rhythm                   |

These are **derived from usage**, not a token table; verify per component when
building (each component file cites its own paddings). Flag any conflict under
Open Questions in `go-ui-notes.md`.

---

## 4. Radius

| Token                | Value  |
| -------------------- | ------ |
| `border.radius.sm`   | `4px`  |
| `border.radius.base` | `6px`  |
| `border.radius.md`   | `8px`  |
| `border.radius.lg`   | `10px` |
| `border.radius.xl`   | `12px` |

Same in both themes. Inputs use `base` (6), badges use `sm` (4), modals use
`md`–`lg`.

---

## 5. Shadow

| Token       | L                                                       | D                                                     |
| ----------- | ------------------------------------------------------- | ----------------------------------------------------- |
| `shadow.sm` | `0 1px 3px rgba(0,0,0,.12), 0 0 0 1px rgba(0,0,0,.05)`  | `0 1px 3px rgba(0,0,0,.5), 0 0 0 1px rgba(0,0,0,.3)`  |
| `shadow.md` | `0 2px 8px rgba(0,0,0,.14), 0 0 0 1px rgba(0,0,0,.06)`  | `0 2px 8px rgba(0,0,0,.6), 0 0 0 1px rgba(0,0,0,.4)`  |
| `shadow.lg` | `0 2px 12px rgba(0,0,0,.15), 0 0 0 1px rgba(0,0,0,.05)` | `0 2px 12px rgba(0,0,0,.7), 0 0 0 1px rgba(0,0,0,.4)` |

Dropdowns and modals use `sm`/`md`; `infoTip.boxShadow` is
`0 4px 12px rgba(0,0,0,.15)` in light.

---

## 6. Component surfaces (token groups)

These groups map a screen region to concrete values. Copy from
`tokens.json` when building.

### 6.1 Input

| Token                       | L         | D                     |
| --------------------------- | --------- | --------------------- |
| `input.bg`                  | `#ffffff` | `transparent`         |
| `input.border`              | `#cccccc` | `#444444`             |
| `input.focusBorder`         | `#8b8b8b` | `rgba(228,174,73,.8)` |
| `input.placeholder.color`   | `#b0b0b0` | `#aaa`                |
| `input.placeholder.opacity` | `0.8`     | `0.6`                 |

Dark inputs are transparent over their panel with an `#444` border — the
"white input" look comes from the panel, not the input.

### 6.2 Modal

| Token                    | L         | D         |
| ------------------------ | --------- | --------- |
| `modal.title.bg`         | `#f1f1f1` | `#1a1a1a` |
| `modal.title.color`      | `#343434` | `#cccccc` |
| `modal.body.bg`          | `#ffffff` | `#222224` |
| `modal.backdrop.opacity` | `0.4`     | `0.4`     |

### 6.3 Sidebar / dropdown / request tabs

| Token                                  | L         | D         |
| -------------------------------------- | --------- | --------- |
| `sidebar.bg`                           | `#f8f8f8` | `#222224` |
| `sidebar.color`                        | `#343434` | `#cccccc` |
| `sidebar.muted`                        | `#838383` | `#aaa`    |
| `sidebar.dragbar.border`               | `#e5e5e5` | `#666666` |
| `sidebar.collection.item.bg`           | `#eaeaea` | `#444444` |
| `sidebar.collection.item.indentBorder` | `#e5e5e5` | `#333333` |
| `sidebar.dropdownIcon.color`           | `#666666` | `#bbb`    |
| `dropdown.bg`                          | `#ffffff` | `#222224` |
| `dropdown.hoverBg`                     | `#f6f6f6` | `#333333` |
| `dropdown.separator`                   | `#e5e5e5` | `#333333` |
| `dropdown.selectedColor`               | `#c07a26` | `#e7b85f` |
| `requestTabs.bg`                       | `#f6f6f6` | `#26292b` |
| `requestTabs.bottomBorder`             | `#efefef` | `#2a2a2a` |
| `requestTabs.icon.color`               | `#9b9b9b` | `#9f9f9f` |
| `requestTabs.icon.hoverColor`          | `#343434` | `#cccccc` |
| `requestTabs.icon.hoverBg`             | `#eaeaea` | `#1e1e1e` |

### 6.4 Response pane

| Token                               | L                      | D                   |
| ----------------------------------- | ---------------------- | ------------------- |
| `requestTabPanel.responseStatus`    | `#838383`              | `#ccc`              |
| `requestTabPanel.responseOk`        | `#2e8a54`              | `#73e89a`           |
| `requestTabPanel.responseError`     | `#ce4f3b`              | `#da462f`           |
| `requestTabPanel.responsePending`   | `#346ab2`              | `#8bc2f9`           |
| `requestTabPanel.responseOverlayBg` | `rgba(255,255,255,.6)` | `rgba(26,26,26,.8)` |
| `requestTabPanel.card.bg`           | `#ffffff`              | `#222224`           |
| `requestTabPanel.card.border`       | `#e5e5e5`              | `#333333`           |

### 6.5 Table, grid, drag-and-drop

| Token                     | L                       | D                       |
| ------------------------- | ----------------------- | ----------------------- |
| `table.border`            | `#efefef`               | `#2a2a2a`               |
| `table.thead.color`       | `#666666`               | `#bbb`                  |
| `table.striped`           | `#f1f1f1`               | `#1e1e1e`               |
| `table.input.color`       | `#343434`               | `#cccccc`               |
| `plainGrid.hoverBg`       | `#f6f6f6`               | `#444444`               |
| `dragAndDrop.border`      | `#8b8b8b`               | `#666666`               |
| `dragAndDrop.borderStyle` | `2px solid`             | `2px solid`             |
| `dragAndDrop.hoverBg`     | `rgba(139,139,139,.05)` | `rgba(139,139,139,.05)` |
| `scrollbar.color`         | `rgb(152,151,149)`      | `rgb(52,51,49)`         |

### 6.6 Status bar, console, icons

| Token                     | L                  | D                 |
| ------------------------- | ------------------ | ----------------- |
| `statusBar.border`        | `#E9E9E9`          | (see tokens.json) |
| `statusBar.color`         | `rgb(100,100,100)` | (see tokens.json) |
| `console.bg`              | `#f8f9fa`          | (see tokens.json) |
| `examples.buttonColor`    | `#d97706`          | `#d9a342`         |
| `examples.urlBar.bg`      | `#f5f5f5`          | `#292929`         |
| `deprecationWarning.icon` | `#d91f11`          | `#fa5343`         |

The `console.*` and `grpc.*` groups are large and mostly browser-chrome
flavoured; read them from `ui-map/_tools/tokens.json` when a console/runner
panel is built. They are not central to v1.

---

## 7. Editor syntax colours (`codemirror.tokens`)

| Token      | L         | D         |
| ---------- | --------- | --------- |
| definition | `#346ab2` | `#8bc2f9` |
| property   | `#346ab2` | `#8bc2f9` |
| string     | `#965d2c` | `#e6bf89` |
| number     | `#b83d7e` | `#e496dd` |
| atom       | `#b94655` | `#f38172` |
| variable   | `#b83d7e` | `#e496dd` |
| keyword    | `#b94655` | `#f38172` |
| comment    | `#9b9b9b` | `#999`    |
| operator   | `#838383` | `#aaa`    |
| tag        | `#b94655` | `#f38172` |
| tagBracket | `#838383` | `#aaa`    |

Variable highlighting inside the editor (`codemirror.variable`):

| State           | L         | D         |
| --------------- | --------- | --------- |
| valid `{{var}}` | `#2e8a54` | `#0bd592` |
| invalid         | `#ce4f3b` | `#da462f` |
| prompt          | `#346ab2` | `#8bc2f9` |

| Editor chrome                           | L                       | D                       |
| --------------------------------------- | ----------------------- | ----------------------- |
| `codemirror.bg`                         | `#ffffff`               | (`tokens.json`)         |
| `codemirror.gutter.bg`                  | `#ffffff`               | (`tokens.json`)         |
| `codemirror.searchMatch`                | `#B8860B`               | `#B8860B`               |
| `codemirror.searchMatchActive`          | `#DAA520`               | `#DAA520`               |
| `codemirror.searchLineHighlightCurrent` | `rgba(120,120,120,.10)` | `rgba(120,120,120,.10)` |

---

## 8. Icons

- **Set:** Tabler Icons v1 (`@tabler/icons`, `packages/bruno-app/package.json`).
- **Sizes observed:** 12, 14, 16 px inline; 18–20 px in primary controls. Stroke
  2 px is Tabler's default.
- **Colour:** inherited from text tokens; sidebar caret uses
  `sidebar.dropdownIcon.color`; top-bar icons use `text`/`text.muted`.
- A Go port maps the same glyphs to a vector set (MyGo `SVG`). Tabler names
  used by feature area are listed in each component file. Do **not** substitute
  a different icon family; keep Tabler outlines.

---

## 9. Themes and presets

`themes/index.js` registers ~21 presets. Light family: `light`,
`light-pastel`, `light-monochrome`, `catppuccin-latte`, `cyberdream-light`,
`gruvbox` (light), `vscode` (light). Dark family: `dark`, `dark-pastel`,
`dark-monochrome`, `catppuccin-frappe`, `catppuccin-macchiato`,
`catppuccin-mocha`, `cyberdream-dark`, `dracula`, `gruvbox` (dark), `nord`,
`vscode` (dark).

Each preset is the same token tree with different palette values. A Go port
should ship `light` and `dark` as first-class and load presets as data, exactly
as `theme.Adapter` already does for imported Bruno themes
(`internal/ui/theme/theme.go` `mapBrunoToTokens`).

---

## 10. Motion

No motion tokens exist in the theme. Observed values (see per-component files
for the exact ones):

- Tab/button colour transitions: ~`0.1s ease` (e.g. `dragAndDrop.transition`).
- Hover transitions on rows/buttons: 120–160 ms `ease`.
- Modal open/close: fade + slight scale, ~200 ms.
- Tooltips (tippy.js, `@tippyjs/react`): fade 150 ms default.
- Toast enter/exit: slide + fade ~200 ms.

A Go port defines these as constants; MyGo transitions take duration + easing
(`ui.ElementTransition`). Exact per-component values are cited where used.
