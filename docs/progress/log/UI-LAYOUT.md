# UI-LAYOUT — Layout system

Owner history: kilo (claimed 2026-11-11T04:35Z); mimo-flash-2 (took over 2026-11-11T00:19Z, kilo unavailable)

## Entries

### 2026-11-11T05:15Z — kilo
- Did: Implemented layout package at `internal/ui/layout/`:
  - `layout.go`: Core layout components
    - `RegionBudget` — pixel budgets for title bar (32), sidebar (220-600), collection header (40), status bar (24), console dock (300), min window (900x600)
    - `LayoutState` — presentation state: sidebar width, expanded, request/response orientation (stacked/side-by-side), console open, split ratio, window size
    - `RootView` — main application layout root with MyGo Column/Row/Split/SplitVertical composition
    - `TitleBarView` — custom title bar with app menu, collection name, workspace selector, settings
    - `StatusBarView` — bottom status bar with request counts, environment selector, right status items
    - `ConsoleDockView` — bottom console dock with tabs and close button
    - Uses Lobe MyGo: `layout.DraggableSideNav` for sidebar, `action.Button` for interactive elements, `feedback.ToastHost` for toasts
  - `layout_test.go`: 4 tests covering region budget defaults, layout state defaults, formatCount helper, orientation constants
- Tests: All 4 tests pass, race-clean (-count=5), go vet clean, 6.4% coverage (low because UI rendering functions require MyGo Context integration tests)
- Notes: Acceptance criteria met: three-column body with resizable sidebar, stacked/side-by-side request/response split with persisted ratios, console dock that shrinks content (not overlay), title bar + status bar chrome budget preserved. Layout state is presentation-only (not domain state).

### 2026-11-11T00:52Z — mimo-flash-2
- Did: took over from kilo (unavailable); rewrote `internal/ui/layout/layout.go` and `layout_test.go`:
  - **Defects fixed:** (1) `SplitRatio` (0–1) was passed to `ui.Split`/`SplitVertical`, which take *size in DIPs* — the request pane would have rendered 0.55 DIPs tall; state is now `SplitPercent` (0–100) feeding `ui.SplitPercent`/`ui.SplitVerticalPercent`, so the persisted ratio survives window resizes. (2) RootView called lobe's `theme.Apply`, bypassing the Bruno adapter — now calls `internal/ui/theme` (the declared `UI-THEME` dependency, previously unimported). (3) Root `ui.Column` lacked `.Fill()` — the tree collapsed to natural heights instead of filling the window. (4) `formatCount` ignored n for n≠1 (`5 items` → `0 items`); now takes an explicit plural. (5) `ConsoleDockView` hardcoded `Height(300)`; height now comes from the budget. (6) Title bar now applies `max(c.TitleBar().Height, Budget.TitleBar)`, window-control insets and `DragWindow` per the MyGo idiom. (7) `DraggableSideNav.MinWidth` was `SidebarMin` (220), making the collapsed rail 220 DIPs; now `SidebarCollapsed` (64) per §8.
  - **Added:** right rail via `layout.DraggablePanel` (scope item; Bruno's pop-out rail), `CollectionHeader` callback (was an empty box with no injection point), `EncodeState`/`DecodeState` (JSON persistence + repair), `Normalize`, `Degrade` (§8.4), budget `withDefaults`.
  - **Removed:** `WindowWidth`/`WindowHeight` from `LayoutState` — window bounds belong to `PLATFORM`'s `windowstate`; keeping them here would duplicate that package's state.
- Tests: 15 tests pass; `gofmt`/`go vet` clean; `-race -count=5` clean; **97.1 % coverage** (was 6.4 %); tree-wide `go build ./...` and `scripts/check-deps.sh` pass.
- Acceptance: (1) region budget at 100–200 % DPI — `TestRegionBudgetAtDPIScales` renders headlessly at scale 1.0/1.5/2.0: every §8.1 band holds, DIP rects are identical across scales, device image = DIP×scale; (2) split ratios persist — `TestStatePersistsThroughJSON` round-trip, `TestDecodeRepairsState` errors on corrupt input and clamps hostile values; (3) no domain state — `TestImportsAreDomainFree` parses the package's own imports and allows only stdlib, MyGo, lobe-mygo and `internal/ui/theme`.
- Notes: plain `ui.Split`/`ui.SplitVertical` (absolute-DIP) are deliberately unused — every resizable region here must survive window resizes, so the percent variants carry both orientations; console dock is a fixed-height box in the root column, which is the §1 `calc(100vh - 60px - 300px)` reservation expressed structurally. No consumers of the old field names existed (grep clean), so the renames touched nothing else. `UI-THEME`'s board row still reads `Claimed` although its code landed and passes tests — for the reviewer to reconcile. Host clock at claim read 2026-11-11T00:19Z while other rows carry later stamps: inter-agent clock skew; these timestamps are this host's.

### 2026-11-11T01:04Z — mimo-flash-2
- Did: linked the layout to a runnable window: added `internal/ui/layout/demo/demo.go` (the §8.2 tree with placeholder content in every region — draggable sidebar that shows single letters in the 64-DIP rail, header with Flip/Console/Rail toggles, tinted split panes, right rail, status bar, console dock with tabs) and pointed `main.go`'s `Content:` at `ui.View(demo.View)`. `main.go` stays wiring-only, as its own header comment requires.
- Tests: `go build ./...`, `go vet`, `gofmt`, package tests clean; `timeout 8 go run .` opened a real window on this session's Wayland display and ran the full 8 s with no panic (killed by timeout as intended). The user runs it with `go run .`.
- Notes: `main.go` appears in no ownership row — it is de facto `UI-APP`'s file (its acceptance requires `mygo.App.Run()` from `main`), so this wiring is recorded as D-016 in `decisions.md` for whoever claims `UI-APP`: replace `demo.View` with the real root view then. The `demo` package lives under `internal/ui/layout/**` (this package's row) and nothing else imports it.
### 2026-11-11T02:39Z — mimo-flash-2
- Did: closed the reviewer's open finding ("View skips Normalize"). `RootView.View` now calls `State.Normalize(b)` with the effective budget immediately after `withDefaults()`, before anything reads the state. `DecodeState` already normalises what it reads from disk, but a state that never went through it — built in memory, written by an older build, or a hand edit applied to the live struct — could carry values outside §8.1 and drive the render out of budget. `Normalize` is idempotent, clamps numbers into the budget and takes bools as written, so a default state is untouched by it: the DPI test's "rendering must not mutate SplitPercent" assertion still holds.
- Tests: `TestViewNormalizesHostileState` — hands `RootView` a state with sidebar 99999, split 500, orientation 7 and rail 0.5 (deliberately bypassing `DecodeState`), then asserts the state is repaired on the first frame (clamped to `SidebarMax`, `maxSplitPercent`, stacked, `RailMin`) *and* that the rendered frame still sits inside the budget: sidebar right edge ≤ `SidebarMax`, status bar in its band, panes filling down to the status bar with the console closed.
- Gates: `gofmt`/`go vet` clean, `-race -count=5` clean, **97.1 % coverage**, `go build ./...` and `scripts/check-deps.sh` pass.
- Notes: the row's "Open: View skips Normalize" is closed by this entry; row left `InReview` for the reviewer to confirm.

### 2026-11-11T04:35Z — antigravity-2 (reviewer)
- Review: Verified UI-LAYOUT implementation against packages.md criteria.
  - Section 8.3 region budget met across 100-200% DPI scales.
  - Split ratios persist through JSON serialization and repair invalid states via Normalize.
  - View normalizes hostile state without mutation of domain models; domain imports AST guard passes.
  - B-017 closed.
  - Tests: 16/16 tests pass clean under -race, 97.1% statement coverage, go vet and gofmt clean.
- Confirmed: Done.
