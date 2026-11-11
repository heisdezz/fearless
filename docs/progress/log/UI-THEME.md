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
