# UI-MAP — Bruno UI reconstruction specification

Owner history: zed-design (claimed 2026-11-11T16:30Z)

## Entries

### 2026-11-11T16:30Z — zed-design

- Did: wrote `ui-map/`, a UI-only specification for rebuilding Bruno's interface
  in the Go port, from the pinned source (`bruno/` at `60d7517`).
  - `overview.md` — package structure, how to run Bruno (`dev:web`,
    `dev:electron`, `storybook`), and a checklist indexing all 104
    `src/components/*` folders to a doc target.
  - `design-system.md` — light/dark tokens resolved to concrete hex by
    `_tools/dump-theme.mjs` (output `_tools/tokens.json`); type, radius,
    shadow, editor syntax, motion.
  - `screens/` (22 files) — app shell, sidebar, sidebar dialogs, request tabs,
    request pane + params/body/auth/sub-tabs, response pane + viewer, runner,
    environments, preferences, collection/folder settings, global search,
    cookies, notifications, welcome, workspace switcher, import/export,
    request docs, about.
  - `components/` (10 files) — button, select/dropdown, toggle, tabs, text
    inputs, status badge, status dot, spinner, method badge, misc primitives.
  - `widgets.md` — MUST/NICE tiers for the editor, tables, markdown, trees.
  - `go-ui-notes.md` — build order, browser-only → native mapping, deferred
    scope, and the rolled-up open questions.
- Tests: not applicable (documentation). Token values cross-checked against
  `internal/ui/theme/theme_test.go` (method colours match).
- Notes: corrected the `lighten` model in `dump-theme.mjs` mid-pass (polished
  `lighten` adds to HSL lightness, not toward white), which fixed the dark
  DELETE token to `#e06552` — matching the repo's own test. Where `refs/`
  captures disagree with source (different build), source wins and the drift is
  recorded in `go-ui-notes.md` §5.

### 2026-11-11T17:35Z — zed-design

- Did: closed the map's flagged gaps against source and folded answers into the
  screen/component files.
  - Resolved with citations: `changeLayout` is globally bound (`Cmd/Ctrl J`,
    `providers/Hotkeys/index.js:386`); URL-bar method caret is off by design;
    `input.background` is a real missing-token bug (use `input.bg`); import
    option labels/descriptions verbatim; the no-collection surface is
    `WorkspaceOverview`, not `EmptyAppState`; Toast is react-hot-toast plus
    `ImportIssuesToast`; ColorPicker preset/slider geometry; RichTextEditor
    toolbar buttons; `INPUT_SIZES`; SettingsInput `w-24 h-8`; request Docs has
    no empty state; auth mode labels; OAuth2 placement Header/URL; assertion
    operator labels; secret masking.
  - `go-ui-notes.md` section 5 rewritten as "Resolved & remaining": resolved
    items keep their sources; the remaining few are genuine product decisions
    or need a fresh capture of the pinned commit.
- Tests: docs only; `go build ./...` and `go test ./...` still green.
- Notes: the remaining open items are decisions (badge vs text slot, hover
  tokens, operator labelling, secret-mask style) or capture-dependent (sidebar
  indent, screenshot drift), not further reading.
