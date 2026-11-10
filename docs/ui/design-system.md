# Design System

Status: VERIFIED unless marked.

---

## 1. Foundations

| Concern        | Choice                                                                                                                                                  | Source                                                            |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Styling engine | **styled-components** (CSS-in-JS) + a small Tailwind setup                                                                                              | `src/globalStyles.js` (`createGlobalStyle`), `tailwind.config.js` |
| Tailwind role  | Minimal: `darkMode: ['class']`, `content: ['./src/**/*.{js,jsx}']`, no custom theme extension. Most styling is styled-components reading `props.theme`. | `tailwind.config.js`                                              |
| Font           | Inter                                                                                                                                                   | `@fontsource/inter` in `bruno-app/package.json`                   |
| Icons          | Tabler Icons v1                                                                                                                                         | `@tabler/icons`                                                   |
| Theming        | ~20 themes, light + dark + accents                                                                                                                      | `src/themes/index.js`                                             |
| Base font size | `0.8125rem` (13 px)                                                                                                                                     | `themes/light/light.js` `font.size`                               |

**Design tokens are a TypeScript/JS object validated by JSON Schema**, visible in `themes/schema/oss.js` and previewed in `themes/DesignSystem/DesignSystem.stories.jsx` (Storybook).

---

## 2. Token structure (VERIFIED keys)

```text
theme = {
  mode: 'light' | 'dark',
  brand, text, textLink, draftColor, bg,

  primary:  { solid, text, strong, subtle },
  accents:  { primary },

  background: { base, mantle, crust, surface0, surface1, surface2 },

  status: { info|success|warning|danger: { background, text, border } },

  overlay: { overlay0, overlay1, overlay2 },

  font: { size: { xs:11, sm:12, base:13, md:14, lg:16, xl:18 (px) } },

  shadow: { sm, md, lg },

  border: { radius: { sm:4, base:6, md:8, lg:10, xl:12 }, border0..border3, subtle },

  colors: { text: {white, green, danger, warning, muted, purple, yellow, subtext0..2},
            bg: { danger }, accent },

  input: { bg, border, focusBorder, placeholder },

  request: { methods: { get, post, put, delete, patch, options, head }, grpc, ws, gql },
  ws: { activeMessage: { label } },
  codemirror: { gutter: { bg }, border, … },
  sidebar/table/tabs/… (component-specific groups)
}
```

Notable: **per-method colors are tokens** (`request.methods.*`) so HTTP verbs render consistently across themes. `MethodBadge` reads the normalized lowercase method to pick its style.

Semantic mapping:

| Intent  | Palette slot                 | Usage                              |
| ------- | ---------------------------- | ---------------------------------- |
| brand   | `hsl(33, 80%, 46%)` (orange) | Primary buttons, active states     |
| info    | BLUE                         | Hints, links                       |
| success | GREEN                        | 2xx meters, passing tests          |
| warning | ORANGE                       | Unresolved vars, 3xx               |
| danger  | RED                          | Errors, 4xx/5xx, failures          |
| draft   | `draftColor`                 | Modified tabs / unsaved indicators |

Themes shipped: `light`, `dark`, `light/dark-monochrome`, `light/dark-pastel`, Catppuccin (latte/frappe/macchiato/mocha), Gruvbox (6 variants), Nord, Dracula, VS Code light/dark, Cyberdream light/dark.

---

## 3. Component library (`src/ui`)

| Primitive                                                                       | Notes                                                          |
| ------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| `Button`                                                                        | Variants via styled-components; has a Storybook story          |
| `Select`                                                                        | Custom dropdown (not native)                                   |
| `MenuDropdown`                                                                  | Row context menus, items with icons & shortcuts                |
| `ContextMenu`                                                                   | Right-click menus                                              |
| `Modal`                                                                         | Portal-based, focus trap (`hooks/useFocusTrap`), Esc close     |
| `Toggle`, `Checkbox`, `RadioButton`                                             | Form controls                                                  |
| `TextInput`, `MaskedInput`, `MaskedTextfield`, `InputWrapper`                   | Inputs with validation states                                  |
| `Tabs` / `ResponsiveTabs`                                                       | Overflow-aware tab strip (collapses to a `…` menu when narrow) |
| `MethodBadge`, `StatusBadge`, `StatusDot`, `CountBadge`                         | Status/indicator primitives                                    |
| `ListGroup`, `SegmentedControl`, `ActionIcon`, `HeightBoundContainer`, `Portal` | Layout helpers                                                 |

Source: `src/ui/*`

Design rules implied by the primitives:

- Every interactive element has hover/focus/disabled states driven by theme tokens.
- Focus is always visible; modals trap focus and restore it.
- Tab strips never overflow horizontally — they collapse.

---

## 4. Editor components

| Component           | Purpose                                                                                     |
| ------------------- | ------------------------------------------------------------------------------------------- |
| `SingleLineEditor`  | One-line CodeMirror (URL, header values) with variable autocomplete                         |
| `MultiLineEditor`   | Multi-line CodeMirror (bodies, scripts, tests) with syntax modes                            |
| `CodeEditor`        | Extensible CodeMirror wrapper used by docs/response views, with search (`CodeMirrorSearch`) |
| `CodeMirror*` utils | Mode loading per format, lint (`@prantlf/jsonlint`), theme mapping (`codemirror.*` tokens)  |

Editor behaviors (VERIFIED from component props):

- `{{variable}}` autocomplete (see `ui/interaction-patterns.md`).
- Clickable variable chips and "set as variable" from selected response text (`SetAsVariableMenu`, `SetAsVariablePopover`).
- Error decorations from scripts (`useFocusErrorLine`).
- Persisted editor state (`CodeEditor/state-persistence.js`).

---

## 5. Accessibility (observed level)

| Aspect         | State                                                                                                  |
| -------------- | ------------------------------------------------------------------------------------------------------ |
| Keyboard       | Strong: full keybinding system, focus trap, Esc-to-clear sidebar selection                             |
| ARIA           | Mixed — `role="alert"` used in banner; many custom widgets rely on keyboard rather than ARIA           |
| Contrast       | Tokens define paired text/bg per theme; `status.*` uses 15 % alpha backgrounds with full-strength text |
| Reduced motion | Not observed in the analyzed files (UNKNOWN)                                                           |

---

## 6. Go port implications (PROPOSED)

1. Formalize the token tree above as Go structs (`Theme`) loaded from JSON, because the existing themes are already _data_. This lets you import Bruno themes later.
2. Keep **per-method colors** as tokens; users expect the same visual language.
3. Do not attempt to replicate styled-components. A Go UI toolkit needs a small design-token layer + widget set; this doc's token table is the contract.
4. 13 px base font with 11/12/14/16/18 px scale is a deliberate density choice — preserve it or the app will feel unlike Bruno.

---

## 7. Go-native design system (PROPOSED — Go-only constraint)

Sections 1–6 document Bruno's web design system (styled-components + Tailwind) as **source evidence**. The Go application must reproduce the _tokens and interaction language_ with native widgets only. No CSS, no HTML, no styled-components, no Tailwind at runtime.

### 7.1 Token parity

The token tree in §2 is already data. In Go it becomes a serializable struct.**Target toolkit: MyGo native UI.** `ui.Theme` already exposes a narrower set of fields than Bruno's tree, so the architecture is a **two-stage adapter**: Bruno-shaped JSON → Go struct (below) → `*ui.Theme` mapping. Fields with no `ui.Theme` slot (per-method colors, extra surfaces, custom syntax palettes) are held by our struct and applied through `ui.Element.Background/TextColor` and `ui.Themed` rather than being lost.

```go
package theme

type Theme struct {
    Name     string `json:"name"`
    Mode     string `json:"mode"` // light|dark
    Brand    color.NRGBA
    Primary  struct{ Solid, Text, Strong, Subtle color.NRGBA }
    Background struct{ Base, Mantle, Crust, Surface0, Surface1, Surface2 color.NRGBA }
    Status   map[string]StatusColors // info|success|warning|danger
    Font     struct{ XS, SM, Base, MD, LG, XL int } // 11,12,13,14,16,18
    Radius   struct{ SM, Base, MD, LG, XL int }    // 4,6,8,10,12
    Border   struct{ Border0, Border1, Border2, Border3 color.NRGBA }
    Methods  map[string]color.NRGBA // get|post|put|delete|patch|options|head
}
```

Rules:

1. **Colors are `color.NRGBA`** in the Bruno-shaped struct; converted to `ui.Color` (`ui.RGB`/`ui.RGBA`/`ui.Hex`) at the `ui.Theme` boundary.
2. **Alpha-derived status backgrounds** (`rgba(intent, 0.15)`) must be precomputed with `Color.Over`/`Color.Mix` — MyGo composites differently from CSS.
3. **Font sizes are integers in px**; `ui.Theme.FontSize` carries the base (13), and larger/smaller variants are applied per element with `Element.FontSize`.
4. **Per-method colors remain tokens** in our struct and drive `MethodBadge` explicitly — `ui.Theme` has no method palette.
5. Tokens are read-only after load; switching themes swaps the snapshot and calls `Context.SetTheme` once.
6. **Spacing and radius** map to `ui.Theme.Spacing` (base unit) and `ui.Theme.Radius`; our struct's `Space(n)`/`Rem(n)` helpers mirror `Theme.Space`/`Theme.Rem`.
7. **Accessibility preferences** (`ReduceMotion`, `HighContrast`, `TextScale`, accent color) are read from `Context.Preferences()` and must feed the theme adapter, not be ignored.

### 7.2 Widgets replace components

The component inventory in `ui/components.md` maps to **MyGo `ui` widgets**; nothing is drawn from scratch except badges and the token adapter.

| Bruno primitive                                         | MyGo widget                                                                                              |
| ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `Button` (variants, sizes)                              | `ui.Button`, `ui.PrimaryButton`, `ui.ButtonBase` (custom look) + `Element.Disabled`                      |
| `Select`, `MenuDropdown`, `ContextMenu`                 | `ui.Select`/`SelectBase`, `ui.Menu`/`ContextMenu`/`PopoverMenu`, `ui.MenuButton`, `ui.PopoverMenuButton` |
| `Modal` / `Portal`                                      | `ui.Modal`, `ui.DialogBase`, `ui.AlertDialog`, `ui.Overlay`                                              |
| `Tabs` / `ResponsiveTabs`                               | `ui.Tabs`/`TabsBase`; the overflow `…` menu is ours (a `MenuButton` when labels exceed the width)        |
| `Toggle`, `Checkbox`, `RadioButton`                     | `ui.Switch`, `ui.Checkbox`, `ui.Radio`, `ui.Toggle`                                                      |
| `MethodBadge`, `StatusBadge`, `StatusDot`, `CountBadge` | `ui.Badge` + `Element.Background`/`TextColor` with token colors                                          |
| `SingleLineEditor`, `MultiLineEditor`, `CodeEditor`     | `ui.TextInput`/`TextInputBuffer`, `ui.TextArea`/`TextAreaBuffer`, **`ui/editor`** for code               |
| Table (`EditableTable`, `ReorderTable`)                 | `ui.Table` + `ListState{Key, Selected, Selection, Reorder, Columns, Sort, CellSelect}`                   |
| Collection tree                                         | `ui.Outline`/`OutlineTable` + `OutlineState` (virtualized)                                               |
| Variable autocomplete                                   | `ui.Autocomplete`, `ui.TokenField`                                                                       |
| Toasts / tooltips                                       | `ui.Toast` / `Context.Toast`, `Element.Tooltip`                                                          |
| Docs tab (tiptap)                                       | `ui/richtext` editor or `ui/markdown` viewer                                                             |
| Response body (HTML/PDF)                                | optional **response-only** `ui.WebView` — never for app chrome                                           |

### 7.3 Accessibility (target state)

Bruno's accessibility was "mixed" (§5). MyGo provides the primitives to do better: every element can carry `Role`, `Label`, `Value`, `Description` and ARIA-style states (`Checked`, `Selected`, `Expanded`, `Invalid`, `Busy`, `Range`), and the test harness asserts them (`ui.Tester.Access`, `AccessByRole`).

- Every interactive element declares a `Role` and an accessible `Label`; no custom-painted control without one.
- Full keyboard operation: Tab order, arrow navigation in trees/tables/tabs (`FocusGroup`), visible focus ring (theme `Focus`, `FocusRingWidth`, `FocusRingOffset`).
- Honour `Context.Preferences()`: `ReduceMotion`, `HighContrast`, `TextScale`.
- Every custom control gets an accessibility assertion in the widget test suite.

### 7.4 Typography and code rendering

- **UI font:** Inter, bundled via `ui.RegisterFont` and set with `Theme.Font`; `"system-ui"` as the fallback.
- **Code font:** `Element.Font("monospace")`, configurable, with a bundled coding font registered through `ui.RegisterFont`.
- **DPI:** MyGo works in DIPs (`ui.Rect`, `Context.Size`); the display scale is applied by the toolkit. Test at 100 %, 125 %, 150 %, 200 % with `ui.Tester.SetScale`.
- **Minimum window size:** the layout in `ui/application-layout.md` assumes ≥ ~900 px; document the graceful degradation path.
