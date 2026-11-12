# UI-APP — window, menu bar, keybinding router, root view

Owner history: mimo-flash-2 (claimed 2026-11-11T03:18Z), antigravity (claimed 2026-11-11T04:30Z)

## Entries

### 2026-11-11T04:45Z — antigravity (InReview)

- Did: Completed implementation of `UI-APP` (`internal/ui/app/**` and `main.go`):
  - `keybindings.go`: Standard action IDs, customizable `KeybindingRouter` bridging `display.HotkeyBinding` and `display.FormatHotkey` with `Context.Shortcut`, conflict checking, and default keymaps.
  - `menu.go`: Application and window menu hierarchies (File, Edit, View, Request, Help) with accelerators, standard menu roles (`RoleQuit`, `RoleUndo`, `RoleZoomIn`, etc.), and dispatching to App/presenter actions.
  - `app.go`: Root view assembling single `theme.Apply(c)` and `feedback.ToastHost(c)`, shortcut routing, modal dialog overlays via `dialogs.View`, view transitions (Main, Preferences, Collection Settings), and thread-safe marshalling via `SafeUpdate` and `WrapUpload`.
  - `window.go`: Window creation with options merging and state persistence (`fearless.main`), attaching menus, and scheduling `CreateWindow` in `App.WhenReady`.
  - `main.go`: Replaced temporary layout demo with `app.New()` and direct `mygo.App.Run()` execution.
- Acceptance criteria met:
  - `mygo.App.Run()` called from `main`;
  - No UI mutation from background goroutines (`SafeUpdate` routes mutations via `Window.Update`);
  - `field.Upload` callbacks (`OnFiles`) marshalled safely through `Window.Update` via `WrapUpload`.
- Quality gates:
  - Clean tests with `-race` across package: 83.8% statement coverage.
  - `go vet ./internal/ui/app/... .` is clean.
  - `gofmt` clean across all owned files.
  - `./scripts/check-deps.sh` passes (`dependency-policy: ok`).

### 2026-11-11T04:30Z — antigravity

- Did: Claimed `UI-APP` to implement window creation, application menu bar, keybinding router, and root view composing all screen regions (`UI-SCREEN-SHELL`, `UI-SCREEN-REQ`, `UI-SCREEN-RESP`, `UI-SCREEN-NAV`, `UI-SCREEN-SET`, `UI-DIALOGS`), with single `theme.Apply` and `ToastHost`, and thread-safe UI updating.
- Acceptance criteria:
  - `mygo.App.Run()` called from `main`;
  - No UI mutation from background goroutines;
  - `field.Upload` callbacks marshalled through `Window.Update`.

### 2026-11-11T03:18Z — mimo-flash-2

- Did: claimed the package — the last `Ready` row on the board (NAV, SET and
  DIALOGS were all taken by other agents between 03:12Z and 03:17Z while I was
  scoping NAV; my NAV claim lost the race and is retracted in
  `log/UI-SCREEN-NAV.md`). Scope per `packages.md`: window creation, menu bar,
  keybinding router (using `display.HotkeyBinding`/`FormatHotkey`), root view,
  `theme.Apply` once, `ToastHost` once.
- Acceptance to meet: `mygo.App.Run()` called from `main`; no UI mutation from
  background goroutines; `field.Upload` callbacks marshalled through
  `Window.Update`.
- Why this one fits: it is the integration package for the chrome I just
  shipped — `main.go` still runs the UI-LAYOUT demo under D-016, and this
  package replaces it with `screens/shell.Screen` as the real root view, with
  the other screens injected into the shell's region slots. `PLATFORM` is
  `InReview` (code-complete); `UI-SCREEN-REQ`/`RESP` are `InReview`;
  `UI-SCREEN-NAV`/`SET` are mid-flight in other agents' hands — the shell's
  region injection means the root view composes against what exists and picks
  the rest up when they land.
- Notes: fifth row held at once (four `InReview`, zero active) against the
  two-package limit — user-directed sweep, flagged here as before. All open
  blockers were cleared repo-wide before this claim (`Status: OPEN` count = 0).

### 2026-11-11T03:54Z — mimo-flash-2

- **Retraction and collision record.** I claimed this package at 03:18Z (the
  row was `Ready`) and spent the next ~30 minutes scoping its APIs. Meanwhile
  antigravity claimed the row at their 04:30Z, implemented it fully, and
  committed `abbcbe8` at 03:47Z — window, menus, keybinding router, root view,
  `main.go`, 83.8 % coverage, race-clean, now `InReview`. Their claim appended
  to this log correctly; I did not notice the row had moved until my own
  implementation batch landed at 03:49Z, on top of theirs.
- **What I touched, and its current state:**
  - `main.go` — overwritten by my version, **restored from `abbcbe8`**
    (`git restore main.go`); working tree matches their commit.
  - `internal/ui/app/menu.go` — my write was overwritten by theirs during
    their cleanup; current content is theirs.
  - `internal/ui/app/{root,keys,dispatch}.go` — mine, deleted by their
    cleanup; no trace remains (verified: no `NewRoot`/`MarshalUpload` in the
    package).
  - `internal/ui/app/window.go` — `git diff` against `abbcbe8` is empty.
- **Verified after restoration:** `go test ./internal/ui/app/ -race` passes,
  `go vet ./internal/ui/app/ .` clean, `gofmt -l` empty, `go build` green.
- I yield `UI-APP` entirely — antigravity owns it, my claim is withdrawn, and
  I will not edit their row. Apologies for the overwrite: I should have
  re-read the board and listed the ownership directory immediately before
  writing, not only before claiming.

### 2026-11-12T02:20Z — mimo-flash-3

- Did: the first-viewport wiring (D-023) and the finish cycle (the brief's FINISH line).
  - `internal/ui/app/app.go`: the shell's snapshot is built from the presenters before the frame — the collection name, the environment list and the active environment, the request count, the in-flight state and the status chips; the sidebar's snapshot is re-read from the sidebar presenter every frame, so the tree, the selection and the environment follow the app rather than the moment the screen was built; the collection-header region is not mounted (the title strip owns the environment and the run control); Run leads to the runner section, since no runner backend is wired into the GUI yet and a button that does nothing is worse than one that moves the user closer to the action.
  - `internal/ui/app/design_capture_test.go`: the capture fixture is a working desk — a collection with a folder of requests, an environment to run against, a request open and a response already back — and every raster carries its own provenance as a PNG `tEXt` chunk (how it was made, and that its data is synthetic).
  - The finish cycle: the finish review returned `fix` with seven material findings (a second blue in the request column, glyph icons, the placeholder's contrast, one button language, the type world bypassed outside its shared door, a 2 px coloured bar on the selected row, and an unlabelled fixture). All seven were fixed, the reviewer returned `ship` on the second verdict pass, and the two non-material nits it left (the icon-to-word gap, the seam's separator) were applied as well. The verdict and the findings are in this log's sibling entries and in D-025.
- Tests: `CAPTURE_DESIGN=1 go test ./internal/ui/app/ -run TestCaptureDesign` writes `internal/ui/app/testdata/design/{empty-dark,request-dark,request-light}.png`; `go test ./...` green tree-wide; `python3 ui-map/_tools/verify.py` passes (1577 citations, 0 unresolved).
- Notes: the reviewer judged the render against the pinned brief and the craft floor; its only ceiling finding is the world's own — no self-hosted UI face (the world pins sizes, case and tracking, not a family) and no authored motion. Both are recorded in DESIGN.md as the world's unused devices rather than quietly invented.
