# UI-SCREEN-SHELL — window chrome screen

Owner history: mimo-flash-2 (claimed 2026-11-11T02:43Z)

## Entries

### 2026-11-11T02:43Z — mimo-flash-2
- Did: claimed the package. It is the window chrome screen — title bar, status row, console dock — per `packages.md` (`UI-SCREEN-SHELL`), ownership `internal/ui/screens/shell/**`.
- Why this one next: of the seven `Ready` rows it is the only one whose building blocks are already code-complete and mine — `internal/ui/layout` already provides `TitleBarView`, `StatusBarView` and `ConsoleDockView`, and `internal/ui/theme` provides the tokens; the other screens (`REQ`, `RESP`, `NAV`) are blocked on `UI-WIDGETS`' code editor, tab bar, collection tree and virtualised lists, which `antigravity` claimed at 03:10Z and whose tree does not compile yet. `UI-APP` waits on every screen plus `PLATFORM`.
- Notes: claimed while `UI-LAYOUT` and `UI-THEME` are still `InReview` — three rows held at once, against the protocol's two-package limit; raised with the user, who directed "move to next free task". Both earlier rows have their review findings closed (mode case-insensitivity, `View` normalizes, B-017) and are waiting on the reviewer. Deps noted as `UI-PRESENT-*` (code-complete, `InReview`) + `UI-WIDGETS`; the shell needs the chrome only, so the widget dependency should reduce to the request tab bar if it touches it at all — to be confirmed while scoping.

### 2026-11-11T03:01Z — mimo-flash-2
- Did: implemented the window chrome screen in `internal/ui/screens/shell/`. Design: a `Screen` struct with a `Snapshot` (plain data — app/collection/workspace names, request/selected counts, environment, status chips, active console tab) and `Actions` (the commands the app owns: menu, workspace, settings, environment, console tab/close, status chip by index). `View` copies both into locals, builds a fresh `layout.RootView` with the chrome regions wired to `layout.TitleBarView`/`StatusBarView`/`ConsoleDockView`, and renders it — so rendering mutates neither the receiver nor anything it reads. The regions the other screens own (`Sidebar`, `CollectionHeader`, `RequestPane`, `ResponsePane`, `RightRail`) are injected function fields, nil-safe, because they are views and not data; the console dock bodies are injected the same way. Status chips with `Index < 0` render as plain labels (Bruno's informational status, no click target); `OnStatus` is addressed by index so the app owns the dispatch.
- Acceptance mapping (`packages.md`, UI-SCREEN-*):
  - **view functions are pure** → `TestViewIsPure`: after a frame the `Snapshot` and `LayoutState` are deep-equal to before, and two frames of the same `Screen` are byte-identical PNGs.
  - **no domain mutation inside a `View` closure** → `TestActionsAreTheOnlyWrites`: clicks on ⚙/environment/status fire exactly the `Actions` counters and change no state (`ConsoleOpen` untouched); the app then applies the change itself and the next frame shows it. `TestNoDomainImports` parses every file's imports and fails on anything outside `fearless/internal/ui/` — the screen cannot reach `internal/core` or `internal/app` at all.
  - **golden screenshots after theme changes** → `TestGoldenAfterThemeChange`: light and dark frames have identical geometry (sentinel rects compared) but differ in pixels (guard against the theme not applying), and each matches `testdata/shell-{light,dark}.png` within a 5 % pixel tolerance; `UPDATE_GOLDEN=1` regenerates.
  - **headless `ui.Tester` smoke test** → `TestShellSmokeTest`: chrome renders headlessly with every region's data present (title, counts, env, injected region sentinels, active dock tab) and the inactive tab's body absent.
- Tests: 7 tests, **100.0 % coverage** of the package, `-race -count=5` clean; `gofmt`/`go vet` clean; `scripts/check-deps.sh` → `dependency-policy: ok`. `TestNilStateRendersDefaults` covers a `Screen` assembled without state (renders one frame of the defaults).
- Notes: build verification is package-scoped (`go build` on shell+theme+layout+presenters+root passes): tree-wide `go build ./...` is red only on `internal/ui/screens/response` (zed-agent-1, `InProgress` — `ui.Text(...).FontSize(12).Color` undefined, `*Tokens` vs `Tokens`), which is not mine and not touched. `internal/ui/screens/request` (antigravity) fails `go vet` on a test file (`model.ItemRequest` undefined) but its non-test code builds. Row moved to `InReview`; I do not set my own packages to `Done`.

### 2026-11-11T04:35Z — antigravity-2 (reviewer)
- Review: Verified UI-SCREEN-SHELL implementation against packages.md criteria.
  - View functions are pure: verified by TestViewIsPure (identical byte output across frames, snapshot unmodified).
  - Actions are the only writes: clicks dispatch intent without state mutation.
  - Goldens: TestGoldenAfterThemeChange passes with light and dark screenshots matching within tolerance.
  - Smoke test: all regions and dock tabs render headlessly.
  - Tests: 100.0% statement coverage, -race -count=5 clean.
- Confirmed: Done.

### 2026-11-12T02:20Z — mimo-flash-3

- Did: the chrome plumbing for the first viewport (D-023).
  - `internal/ui/screens/shell/shell.go`: the shell's snapshot carries the app name, the collection name, the environments and the in-flight state, so the title strip can render them without reaching for a presenter; the status strip carries the counts, the chips and the workspace/settings controls. The regions the other screens own stay injected callbacks — views, not data — and the collection-header region is now `CollectionHeaderRegion`, with a nil-safe `CollectionHeader` method over it (D-024).
  - The shell's actions cover the title strip's controls: menu, environment selection, environment management, run, stop, workspace, settings, console tab and close.
- Tests: `go test ./internal/ui/screens/shell/` green — `TestActionsAreTheOnlyWrites` (clicks on the drawn menu, the environment control, the run control, and a direct environment selection; the stop control while a request is in flight), the smoke test, the geometry test and the light/dark goldens.
- Notes: the shell draws no icon of its own: the menu, the gear and the close control come from `widgets.IconButton`, so the app has one icon system (D-025).
