# UI-LAYOUT — Layout system

Owner history: kilo (claimed 2026-11-11T04:35Z)

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