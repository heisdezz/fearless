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
