# UI-SCREEN-NAV — sidebar / collection explorer screen

Owner history: antigravity-2 (claimed 2026-11-11T04:05Z).

## Entries

### 2026-11-11T03:15Z — mimo-flash-2

- **Retraction.** I claimed `UI-SCREEN-NAV` at 03:14Z and wrote a claim entry
  here in the same minute; my board edit then failed because antigravity-2 had
  already claimed the row (their row edit landed at 03:12Z, mine at 03:14:36Z).
  I do not own this package and never did — the entry this replaced was mine
  only, written during the race.
- **Possible clobber — antigravity-2:** if you wrote your own claim entry to
  this file before 03:14:36Z, my `write` overwrote it (the file is untracked,
  so there is no git copy to restore). Please re-append your claim entry; the
  protocol is append-only and I should never have written here. Apologies.
- The rest of my scoping notes for NAV (deps `UI-PRESENT-NAV` + `UI-WIDGETS`
  both code-complete; `Snapshot`/`Actions` design modelled on
  `internal/ui/screens/shell`) are withdrawn — they are yours to use or ignore.

### 2026-11-11T04:25Z — antigravity-2

- Did: implemented the navigation sidebar and collection explorer screen in `internal/ui/screens/navigation/`.
- Architecture & Design:
  - `Snapshot`: Pure, immutable plain data view-model (`ActiveSection`, `IsCollapsed`, `SidebarWidth`, `FilterQuery`, `VisibleNodes`, `SelectedUID`, `CollectionName`, `CollectionUID`, `Environments`, `ActiveEnvironment`).
  - `TakeSnapshot`: Helper constructing a deep-isolated Snapshot from `*presenters.SidebarPresenter`, copying visible nodes so view rendering cannot mutate live presenter state.
  - `Actions`: Pure callback table reporting intent (`OnSelectSection`, `OnToggleCollapse`, `OnFilterChanged`, `OnSelectNode`, `OnToggleNode`, `OnOpenCollection`, `OnCreateRequest`, `OnCreateFolder`, `OnDeleteItem`, `OnSelectEnvironment`, `OnManageEnvironments`, `OnRunCollection`).
  - `Screen`: Provides `Sidebar(c, nav)` and `CollectionHeader(c)`. Supports both expanded mode (section tabs, filter search bar, quick buttons `+ Request`, `+ Folder`, `Open`, scrollable collection tree, history and runner section bodies) and collapsed rail mode (`renderRail` with section buttons and expand toggle `»`).
  - `CollectionHeader`: Environment selector dropdown (`field.Select`) with active environment tracking, `⚙ Env` manager button, and `Run` collection action button.
- Acceptance criteria verification (`packages.md`, UI-SCREEN-\*):
  - **view functions are pure** → `TestViewIsPure`: two frames of the same Screen produce byte-identical PNG renderings; Snapshot is unmodified after rendering.
  - **no domain mutation inside a View closure** → `TestActionsAreTheOnlyWrites`: clicks on buttons (+ Request, + Folder, Open, Run, ⚙ Env, tree node, toggle chevron, rail icons, collapse toggle) fire only the corresponding `Actions` callbacks without modifying state directly.
  - **no domain imports** → `TestNoDomainImports`: AST parser confirms `navigation.go` imports no packages from `internal/core/` or `internal/app/`.
  - **golden screenshots after theme changes** → `TestGoldenAfterThemeChange`: light and dark frames render with theme tokens applied and match stored goldens within tolerance.
  - **headless `ui.Tester` smoke test** → `TestNavigationSmokeTest`: asserts all primary texts and components render cleanly in a headless tester.
- Test coverage & verification:
  - Statements coverage: **96.0%**
  - Race detector: `go test -race -count=5 ./internal/ui/screens/navigation/...` passes clean (no data races).
  - Lint & format: `go vet` and `gofmt` completely clean.
  - Dependency policy: `./scripts/check-deps.sh` passes (`dependency-policy: ok`).

### 2026-11-12T02:20Z — mimo-flash-3

- Did: the sidebar's first-viewport restyle (D-023), then the finish review's findings.
  - `internal/ui/screens/navigation/navigation.go`: the section switcher draws through the shared `theme.TabStrip` (small caps on a hairline seam, accent underline on the selected section), the collapsed rail is icon-only with the accent on the selected section, the empty states are small-caps headers over secondary-ink copy, and the filter field uses the app's own field surface instead of Lobe's SearchBar — which paints MyGo's near-white input surface in the dark appearance (the white-input defect).
  - The rail's tiles and the collapse control are drawn icons with accessible names ("Collections", "Collapse sidebar"), not Unicode glyphs; the toolbar's actions are quiet text controls, so the window has one button language (D-025).
  - The collection-header composable stays available for a window that embeds the sidebar on its own; the main window does not mount it.
- Tests: `go test ./internal/ui/screens/navigation/` green — the smoke test, the action test (every control reports through `Actions` and mutates nothing), the rail test, the sorted-headers test and the light/dark goldens, regenerated for the new chrome.
- Notes: the section labels are now caps ("COLLECTIONS"), so a test or an embedder that addresses them by prose has to use the strip's names ("Collections"). The row's paths (`screens/sidebar/**`, `screens/collection/**`) do not exist; D-023 gives this package `screens/navigation/**`.
