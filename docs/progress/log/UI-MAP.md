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

### 2026-11-11T18:10Z — zed-design

- Did: made the map self-verifying and fixed every broken citation it exposed.
  - Added `ui-map/_tools/verify.py` (checks checklist targets, internal doc
    links, citation file resolution and past-EOF) and
    `ui-map/_tools/qualify.py` (rewrites bare `index.js`/`StyledWrapper.js`
    citations to src-relative paths per file).
  - Qualified ~1,200 abbreviated citations and corrected genuine wrong-file
    citations found by the check (RequestBody caret, Preferences/ProxySettings
    vs Keybindings, CollectionSettings vs ShareCollection/Migrate, sidebar row
    styles, RequestTabLoading/NotFound, response-pane section, Sidebar create/
    packages rows).
  - Fixed 2 broken internal doc links.
- Tests: `python3 ui-map/_tools/verify.py` -> 1,565 citations, 0 unresolved,
  0 past EOF. `go build ./...` and `go test ./...` green.
- Notes: the map now states it is COMPLETE for the pinned commit, with only
  capture-dependent items open. Agents can run verify.py after edits.

### 2026-11-11T18:40Z — zed-design

- Did: resolved the map's remaining "decide" items and recorded them as
  `ui-map/decisions.md` (R-1..R-7), so implementers are not blocked on them.
  - R-1 sidebar verbs use `RequestMethod` (11px, 34px slot, >5-char verbs
    truncated to 3, grpc/ws/gql labels), not `MethodBadge`.
  - R-2 no request-tab hover tint (source has none; only the close gradient).
  - R-3 preview tabs are italic only; no preview colour token.
  - R-4 assertion operators keep raw ids except eq/neq -> equals/notEquals.
  - R-5 secret masking is one `*` per character everywhere (corrects an
    earlier wrong "three styles" note).
  - R-6 `input.background` is a source bug; use `input.bg`.
  - R-7 shortcuts are global (Cmd/Ctrl J at providers/Hotkeys/index.js:386).
- Tests: `python3 ui-map/_tools/verify.py` -> 1,577 citations, 0 unresolved,
  0 past EOF. `go build ./...` and `go test ./...` green.
- Notes: only capture-dependent items remain open (sidebar indent depth,
  screenshot drift, uncaptured Preferences/environments surfaces).

### 2026-11-11T19:05Z — zed-design

- Did: made the parallel-agent protocol cover this package and enforced it.
  - Registered `UI-MAP` in `ownership.md` (`ui-map/**`, `refs/**`) — the map and
    its captures were previously owned by no row (D-021).
  - Added `scripts/check-ownership.sh`: `--pkg <ID>` fails on a file outside
    that package's row, no argument fails on unowned files, `--range` checks
    commits. §6 shared files are always allowed.
  - Documented the check as protocol step §9 in `docs/progress/README.md`.
- Tests: checker verified in both directions — `--pkg UI-MAP` passes on UI-MAP
  work, `--pkg UI-THEME` fails on it, and an unowned path is reported. The
  checker also flagged my own edit to `ownership.md`, which is the intended
  behaviour for a read-only contract file.
  `python3 ui-map/_tools/verify.py`: 1,577 citations, 0 unresolved, 0 past EOF.
- Notes: `ownership.md` is read-only under §6; the edit is authorised by D-021,
  and any future top-level doc/asset tree must add a row and pass the checker.
