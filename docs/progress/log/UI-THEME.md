# UI-THEME — design tokens, `theme.Apply` call point, per-method colour map, Bruno theme import

Owner history: mimo-flash-2 (claimed 2026-11-11T01:10Z, confirmed 2026-11-11T04:35Z); zed-design (claimed 2026-11-11T04:24Z)

## Entries

### 2026-11-11T04:24Z — zed-design

- Did: claimed the package for the D-017 Graphite Console redesign. No code changed yet.
- Tests: not run.
- Notes: reopening a `Done` package is deliberate: the product owner asked for a
  full visual redesign, and the world is carried by this package's tokens. The
  two findings driving the work are (a) no Graphite Console token world exists —
  the app renders Lobe's defaults, and (b) dark mode paints light input surfaces
  in the assembled app (`internal/ui/app`), which the package's own 98 % test
  coverage did not catch because the tests measure `MethodColor` and the token
  map, not a frame painted by a component that changed the frame theme itself.

### 2026-11-11T05:40Z — zed-design

- Did: closed the white-input defect (D-017) and its cousin, the theme that a
  repainting widget could not resolve.
  - Root cause of the white surfaces is upstream: Lobe's `field.Input`,
    `field.Select` and `action.Button` paint their theme background on a
    container node they do not return, so what actually paints keeps MyGo's
    near-white default. Reproduces with Lobe's own `theme.Apply` alone.
  - Second bug: `Adapter.Apply` built an ad-hoc `ui.Theme` that Lobe's
    `TokensOf` could not map back to tokens, so widgets asking for tokens got
    Lobe's light defaults. `Apply` now installs `config().Theme(dark)` from a
    `theme.NewConfig` pair; `buildConfig` finishes each mode's theme through
    `worldTheme` so the world's accent survives a repaint (D-019).
  - Added `internal/ui/widgets/controls.go` (`FieldInput`, `FieldSelect`,
    `Button`) and routed every screen through them.
- Tests: `go test ./...` green. `internal/ui/theme` tests updated to assert the
  world's tokens for the unimported mode (the old assertions encoded Lobe's
  defaults). Shell golden regenerated (`shell-light/shell-dark`) — geometry
  unchanged, pixels now Graphite; the test's real assertions still pass.
- Notes: design captures moved to `internal/ui/app/design_capture_test.go`,
  opt-in via `CAPTURE_DESIGN=1`, output in `internal/ui/app/testdata/design/`.

### 2026-11-11T23:16Z — mimo-flash-3

- Did: took over the claim (zed-design stale since 04:24Z) to build the
  direction contract's first viewport (surface brief
  `.impeccable/briefs/internal-ui-app.md`, FINISH line undischarged). Scope
  this pass: the type system the world lacks — small-caps panel headers
  (10 px, 0.08 em tracking) and monospaced tabular data — plus the token-side
  support the composition needs. The composition moves themselves (30 DIP
  title strip, 248 px sidebar, env+run into the strip) are claimed under
  `UI-LAYOUT` on the same timestamp.
- Tests: not run yet; entry closes with the pass's own test run.
- Notes: concurrent agent `antigravity` is executing the same user request on
  the documentation side (`docs/ui/*.md`, recorded as D-022) — disjoint file
  sets, no shared paths. `docs/ui` documents Bruno's reference geometry; the
  build follows the brief's contract (30/248/24), and the documenter
  reconciles at DESIGN.md time.

### 2026-11-12T01:20Z — antigravity

- Did: completed Phase 2 first-viewport visual restyle across the request screen and chrome surfaces.
  - Applied `uitheme.DataGroup` to the request screen's target URL bar for monospace with tabular figures.
  - Refined sub-tab panel section titles (`Query Parameters`, `Path Parameters`, `Headers`, `Pre-Request Variables`, `Post-Response Variables`, `Pre-Request Script`, `Response Assertions`, `Tests`, `Documentation`) to use small-caps `uitheme.PanelHeader` (10 px, 0.8 DIP tracking, TextSecondary ink).
  - Verified primary `Send` action button alignment and electric blue styling (`action.Primary`).
  - Generated and verified design captures in `internal/ui/app/testdata/design/` (`empty-dark.png`, `request-dark.png`, `request-light.png`).
- Tests: `CAPTURE_DESIGN=1 go test -v ./internal/ui/app -run TestCaptureDesign` green; full `go test ./...` tree-wide green; `python3 ui-map/_tools/verify.py` passes (1577 citations, 0 unresolved, 0 past-EOF).

### 2026-11-12T02:20Z — mimo-flash-3

- Did: closed the D-017 type world and the finish review's findings on it.
  - `internal/ui/theme/type.go`: the shared type world — small-caps panel headers (10 px, 0.8 DIP tracking), the monospaced data face with tabular figures, the title size, and the `PanelHeader`/`Header`/`DataText`/`DataGroup` helpers every screen calls (D-017: no screen restyles itself).
  - `internal/ui/theme/tabs.go` (new): **one** section-tab strip for the whole window — small-caps labels on a hairline seam, a 2 DIP accent underline on the selected tab, the figure in the data face, the tab named by its own label. The sidebar switcher, the request column's sections and the response pane's sub-tabs all draw through it (D-025).
  - `internal/ui/theme/graphite.go`: the placeholder ink is its own token now, lifted clear of 4.5:1 against both the ground and the panel (dark `#868A92`, light `#646870`). The brief's three pinned text levels are untouched; the placeholder was never one of them, and it sat at 3.4:1.
- Tests: `go test ./internal/ui/...` green; the world's strip is exercised through the response screen's seam and tab tests (`TestStatusSeamIsOneDataRow`, `TestActiveTabCarriesTheAccentUnderline`).
- Notes: the finish review (see the UI-APP entry) found a second blue — the request column's strip came from a component library and painted its own underline colour — and a strip that drifts is a strip that grows a second accent; hence `theme.TabStrip` rather than three copies. `internal/ui/screens/response/**` is being written by the concurrent agent for the SSE stream feature: its `SubTabs` was replaced with a hand-rolled copy and re-routed through the shared strip, recorded as B-026.
