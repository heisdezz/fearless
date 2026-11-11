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

### 7.2 Baseline MyGo `ui` widgets (fallback layer)

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

### 7.2.1 Layering: MyGo floor, Lobe components on top (PROPOSED)

The table in §7.2 is the **floor**: what the plain MyGo `ui` package offers. The application does not build its look from that floor directly. It uses **Lobe MyGo** (`github.com/lobehub/lobe-mygo`), a component library drawn on MyGo's native UI, and falls back to the `ui` primitives only where Lobe has no component.

| Layer                | Package                                                                        | Role in this app                                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| Toolkit              | `github.com/egoist/mygo/ui`                                                    | Layout (`Column`/`Row`/`Box`/`Split`/`SplitVertical`/`SplitPercent`), drawing, input, IME, `ui/editor`, `ui/richtext`, `ui.Tester` |
| Design tokens        | `lobe-mygo/pkg/theme`                                                          | All colour, radius, font-size and control-height tokens                                                                            |
| Components           | `lobe-mygo/pkg/{action,field,choice,display,feedback,overlay,nav,layout,code}` | Buttons, inputs, menus, modals, tabs, side nav, lists, alerts, toasts                                                              |
| Icons                | `lobe-mygo/pkg/lucide`                                                         | All icons (2,300+ Lucide glyphs as Go functions)                                                                                   |
| App-specific widgets | ours                                                                           | Method badge, key/value editors, code editor with variable completion, tree with drag-reorder                                      |

Source: `github.com/lobehub/lobe-mygo` README (“Lobe UI components for MyGo native UI, drawn on the GPU in pure Go”)
Relevant symbols: package list in the module README

Lobe is MIT-licensed, pure Go, needs Go ≥ 1.27 (the `go.mod` here declares 1.27.3), and states it “brings the Lobe UI design language to desktop apps written with MyGo's native UI: no HTML, no JavaScript and no webview”. It is therefore compatible with the Go-only constraint: it is not a webview wrapper, it is a component set that paints through MyGo.

### 7.2.2 Theme tokens come from `lobe/theme`, not from a hand-written adapter

This replaces rule (1)–(5) of §7.1 as the _primary_ mechanism. The Bruno-shaped struct in §7.1 is retained **only as an import format** for users who bring Bruno's own theme JSON across; the runtime token source is `theme.TokensOf(c)`.

```go
import (
    "github.com/egoist/mygo/ui"
    "github.com/lobehub/lobe-mygo/pkg/theme"
)

func view(c *ui.Context) {
    theme.Apply(c)                    // light/dark follows the system appearance
    tk := theme.TokensOf(c)           // copy of the active Tokens
    ui.Column(c).Background(tk.BgContainer).Children(func() {
        ui.Text(c, "Body").TextColor(tk.Text)
    })
}
```

API (VERIFIED from the package docs):

| Symbol                                      | Signature                                                            | Use here                                                                                                                             |
| ------------------------------------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `theme.Apply`                               | `func Apply(c *ui.Context)`                                          | Called once at the top of each view; selects light or dark from system appearance                                                    |
| `theme.TokensOf`                            | `func TokensOf(c *ui.Context) *Tokens`                               | Read the active token set in any view                                                                                                |
| `theme.Light` / `theme.Dark`                | `func() *Tokens`                                                     | Preset token sets                                                                                                                    |
| `theme.Default`                             | `func() *Config`                                                     | Lobe UI's own tokens (what `Apply` uses)                                                                                             |
| `theme.NewConfig`                           | `func NewConfig(light, dark Tokens) *Config`                         | Brand override; docs say “keep the Config for the life of the app rather than making one for each frame: every Config is remembered” |
| `(*Config).Apply`                           | `func (cfg *Config) Apply(c *ui.Context)`                            | Apply a branded config                                                                                                               |
| `(*Config).Theme`                           | `func (cfg *Config) Theme(dark bool) *ui.Theme`                      | Theme for a `ui.Themed` subtree                                                                                                      |
| `(*Tokens).Color`                           | `func (t *Tokens) Color(name string) (ui.Color, bool)`               | Look up a token **by string name**                                                                                                   |
| `theme.ColorTokenNames`                     | `func ColorTokenNames() []string`                                    | Enumerate valid names — useful for a theme-inspector settings page                                                                   |
| `(*Tokens).Elevate`                         | `func (t *Tokens) Elevate(e ui.Element, level Elevation) ui.Element` | Lobe's shadow + 1 px ring; `theme.BoxShadow`, `BoxShadowSecondary`, `BoxShadowTertiary`                                              |
| `(*Tokens).FillBorder` / `FillBorderStrong` | `func() ui.Color`                                                    | Derived border colours                                                                                                               |
| `theme.Mix`                                 | `func Mix(c, o ui.Color, t float32) ui.Color`                        | Blend for hover states                                                                                                               |
| `theme.Size`                                | `Middle`, `Small`, `Large`                                           | Control sizing enum used by nearly every component prop                                                                              |

`Tokens` is a flat struct of ~200 `ui.Color` fields plus scale fields. The groups that matter for this application:

| Group    | Fields (examples)                                                                                                                                                        | Maps to Bruno token                        |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------ |
| Surfaces | `BgBase`, `BgLayout`, `BgContainer`, `BgElevated`, `BgContainerSecondary`, `BgContainerDisabled`                                                                         | `background.base/mantle/crust/surface0..2` |
| Text     | `Text`, `TextSecondary`, `TextTertiary`, `TextQuaternary`, `TextDescription`, `TextPlaceholder`, `TextDisabled`, `TextHeading`, `TextLabel`                              | `text`, `colors.text.subtext0..2`          |
| Borders  | `Border`, `BorderSecondary`, `BorderBg`, `BorderDisabled`, `Fill`, `FillSecondary`, `FillTertiary`, `FillQuaternary`, `Split`                                            | `border.border0..3`, `border.subtle`       |
| Brand    | `Primary`, `PrimaryHover`, `PrimaryActive`, `PrimaryBorder`, `PrimaryFill*`, `PrimaryText*`, `PrimaryForeground`                                                         | `primary.solid/text/strong/subtle`         |
| Status   | `Success*`, `Warning*`, `Error*`, `Info*` (each with `…Active`, `…Hover`, `…Bg`, `…Border`, `…Fill`, `…FillSecondary/Tertiary/Quaternary`, `…Text`)                      | `status.{success,warning,danger,info}`     |
| Accents  | `Blue`, `Cyan`, `Geekblue`, `Gold`, `Green`, `Lime`, `Magenta`, `Orange`, `Purple`, `Red`, `Volcano`, `Yellow`, `Gray` (+ `…Active`, `…FillTertiary`, `…FillQuaternary`) | accent usage, tag palettes                 |
| Scale    | `BorderRadius`, `BorderRadiusLG/SM/XS`, `ControlHeight`, `ControlHeightLG/SM/XS`, `FontSize`, `FontSizeSM/LG/XL`, `FontSizeHeading1..5`, `FontWeightStrong`, `LineWidth` | `font.size.*`, `border.radius.*`           |

Consequences for the earlier rules in §7.1:

1. **Do not hand-roll alpha backgrounds.** Bruno's `rgba(intent, 0.15)` status backgrounds already exist as `SuccessBg`, `ErrorBg`, `WarningBg`, `InfoBg` and their `…Hover` counterparts. Use them; only fall back to `theme.Mix` for a colour Lobe does not define.
2. **Control density is a token pair.** `ControlHeight`/`FontSize` give the default 32 px / 13 px pair that matches Bruno's density; `theme.Small` and `theme.Large` select 24 px and 40 px variants through the components' `Size` prop. This satisfies §6 rule 4 (preserve Bruno's density) without hard-coding numbers.
3. **Named lookup beats struct fields** where a colour comes from persisted user settings or a `.bru` `color:` rule: `tk.Color(name)` with names from `theme.ColorTokenNames()` avoids a `switch` over 200 fields.
4. **Per-method HTTP colours still have no token.** Lobe's palette has `Green`, `Blue`, `Orange`, `Red`, `Purple`, `Cyan`, `Gold` — enough to _derive_ Bruno's light/dark method colours, but there is no `request.methods.*` equivalent. Keep our own `Methods map[string]ui.Color` in the theme adapter and read from it in the method badge (§7.2.3).

VERIFIED: the full `Tokens` field list and the signatures above are transcribed from the `pkg/theme` package documentation of `lobe-mygo` (commit `5e5fea5`, docs dated 2026-11-10).
Source: `github.com/lobehub/lobe-mygo` → `pkg/theme` (`apply.go`, `shadow.go`, `size.go`, `theme.go`, `tokens.go`, `tokens_gen.go`)
Relevant symbols: `Apply`, `TokensOf`, `Config`, `Tokens`, `Elevation`, `Size`, `ColorTokenNames`, `Mix`

### 7.2.3 Typography, spacing and iconography in the Lobe layer

§7.4's rules still hold; these are the additions that come with Lobe.

| Concern      | Rule                                                                                                                                                                                                                                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Text styles  | Prefer `display.Text(c, s, display.TextProps{…})` over raw `ui.Text`: it carries Lobe's type ramp (`Heading` 1–5, `Strong`, `Italic`, `Code`, `Mark`, `Disabled`, `Rows` truncation, `Shiny` pending state) and status `Type` (`TextSecondary`, `TextSuccess`, `TextWarning`, `TextDanger`, `TextInfo`) |
| Code text    | `display.TextProps{Code: true}` for inline code; `code.Highlighter` for blocks (§7.2.4)                                                                                                                                                                                                                 |
| Size enum    | Every component with a `Size` field takes `theme.Middle`/`Small`/`Large`; do not invent pixel sizes                                                                                                                                                                                                     |
| Icons        | `display.Icon(c, svg, display.IconProps{Size: display.IconSmall\|IconMiddle\|IconLarge, Spin: bool})`; `IconInherit` follows the surrounding font size                                                                                                                                                  |
| Icons source | `lobe-mygo/pkg/lucide` — Lucide glyphs as Go functions. pkg.go.dev withholds the symbol list for licence reasons, so take individual names from the catalog code under `catalog/*/` or the Lucide set                                                                                                   |
| Keycaps      | `display.Hotkey(c, "mod+shift+k", display.HotkeyProps{Platform: display.HotkeyAuto})` renders platform-correct glyphs (⌘ on Apple, Ctrl elsewhere)                                                                                                                                                      |
| Key parsing  | `display.HotkeyBinding("mod+k", display.HotkeyAuto) (ui.Modifiers, ui.Key, bool)` and `display.FormatHotkey(mods, key, platform) (string, bool)` bridge our keybinding table to `Context.Shortcut` and back                                                                                             |
| Spacing      | Layout components take `Gap` fields (e.g. `layout.GridProps.Gap` defaults to 14 DIPs, `SortableListProps.Gap` to 8, `AccordionProps.Gap`); prefer these over manual padding                                                                                                                             |
| Shadows      | Use `tk.Elevate(el, theme.BoxShadow)` for dialogs, `BoxShadowSecondary` for popovers and menus, `BoxShadowTertiary` for tooltips — this is Lobe UI's own convention from the docs                                                                                                                       |

**Accessibility is improved by the Lobe layer, not reduced.** §7.3 asked every interactive element to declare a `Role` and `Label`; Lobe components already take a `Label`/`Title` prop for this (`action.ActionIconProps.Title` “names the button for assistive technology and shows as its tooltip”; `overlay.Tooltip` “describes anchor to assistive technology”). Where we build a custom widget, we still set `Role`/`Label` ourselves. The `Spin` variant `theme.Size` and `display.Skeleton` give us loading states that are announced rather than blank.

### 7.2.4 Code rendering

Two distinct needs, two mechanisms:

| Need                                                                 | Mechanism                                                                                            | Notes                                                                                                                                                |
| -------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Read-only, highlighted** (response body, generated code, snippets) | `code.Highlighter(c, src, code.HighlighterProps{Language: "json", Wrap: false, FullFeatured: true})` | Handles copy button (`NoCopy`), language label, collapse toggle, optional `Actions`; `LanguageName` normalises `"ts"`→`"TypeScript"`                 |
| **Editable** (request body, scripts, tests)                          | `ui/editor` from MyGo with `code.Theme(tk)`                                                          | `code.Theme` returns Lobe's syntax colours for the active tokens; `code.Spans(ct, lang, src)` yields `[]ui.Span` if we need to paint spans ourselves |
| Single line                                                          | `code.Snippet`                                                                                       | Prefix support (`"$"`) is excluded from what Copy copies                                                                                             |

The important consequence: **there is no Lobe code editor.** `pkg/code` highlights and displays; it does not accept edits. The editable editor remains a MyGo `ui/editor` instance that we configure with `code.Theme(tk)` so its colours match Lobe. See `ui/request-editor.md` and the "custom implementation" list in `ui/components.md` §6.

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
