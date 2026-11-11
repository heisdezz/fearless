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
