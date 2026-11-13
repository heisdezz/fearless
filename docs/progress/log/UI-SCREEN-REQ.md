# UI-SCREEN-REQ — Request Editor Screen

Owner history: antigravity (claimed 2026-11-11T03:47Z)

## Entries

### 2026-11-11T03:47Z — antigravity

- Did: Claimed `UI-SCREEN-REQ`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/ui/components.md`.
- Scope: Request editor screen rendering `RequestPresenter` through Lobe components and custom widgets:
  - Top action row: HTTP Method selector badge, URL input with `{{var}}` autocomplete, Send / Cancel button, Save button.
  - Sub-tabs bar: Params, Body, Headers, Auth, Vars, Script, Assert, Tests, Docs.
  - Sub-panes:
    - Params: Query parameters & path params editable key/value grid.
    - Body: Body mode selector (None, JSON, Text, XML, Form-urlencoded, Multipart) + CodeEditor / KeyValueGrid.
    - Headers: Headers editable key/value grid.
    - Auth: Auth mode selection and inputs.
    - Vars: Pre/Post request variables grid.
    - Script: Pre-request script code editor.
    - Assert / Tests: Post-response test script code editor.
    - Docs: Markdown documentation editor.
- Acceptance: pure view function, headless `ui.Tester` smoke test, no domain mutations in view pass.

### 2026-11-11T04:04Z — antigravity

- Did: Implemented the request editor screen and all sub-tab panes in `internal/ui/screens/request/`:
  - `screen.go`: `RequestScreen` root component with empty-state handling, top URL / HTTP method selector, Send / Cancel toggle, Save dirty indicator, and `nav.Tabs` sub-tab switcher.
  - `params.go`: `renderParamsTab` for Query and Path parameters with editable key-value grids and automatic param rebuilding.
  - `body.go`: `renderBodyTab` with body mode segmented control (`none`, `json`, `text`, `xml`, `formUrlEncoded`, `multipartForm`) and matching editors.
  - `headers.go`: `renderHeadersTab` for request headers grid.
  - `auth.go`: `renderAuthTab` supporting Inherit, None, Basic, Bearer, APIKey, Digest, and OAuth2 modes.
  - `vars.go`: `renderVarsTab` for Pre-Request and Post-Response variables with expression editing.
  - `scripts.go`: `renderScriptTab`, `renderAssertTab`, `renderTestsTab`, and `renderDocsTab` with syntax-highlighted code editors and assertion grids.
  - `screen_test.go`: Complete headless testing with `ui.NewTester`, testing empty state, URL editing, Send / Cancel actions, dirty Save, sub-tab switching, body modes, auth modes, callbacks, and full accessibility tree inspection.
- Tests: `go test -race -cover ./internal/ui/screens/request/...` (87.0% statement coverage), `go vet ./internal/ui/screens/request/...`, `gofmt`, and `./scripts/check-deps.sh` all pass cleanly.
- Status: Moving `UI-SCREEN-REQ` to `InReview`.

### 2026-11-12T14:49Z — zed-agent

- Did: a `{{variable}}` click in the request URL bar opens a variable editor, where the variable can be created or edited.
  - `internal/ui/widgets/template_field.go`: a click on a token opens its variable, as before; a click on the path text still starts editing the field. A variable token is `{{var}}` in the URL, so the click target is the token, not the whole field.
  - `internal/ui/app/app.go`: `OnOpenVariable` now routes to `EditVariable(name)` instead of the generic `ShowEnvironment()`. `EditVariable` opens the environment editor on that name, creating a missing variable in the draft so the table has a row to highlight, and offers Back to return to the request. `ShowEnvironment` clears the focus when opened any other way.
  - `internal/ui/screens/environment/screen.go`: `State.Focus` marks the variable to open and `State.Back` closes the editor; `focusNewVariable` adds a missing name to the draft on the first frame. The row of the focused variable is highlighted once.
  - `internal/ui/screens/request/screen.go`: the `OnOpenVariable` doc now says a click, not a double click.
  - `internal/ui/screens/environment/screen.go`: uses `presenters.NextEnvironmentName`, the rule moved to the presenter by the concurrent pass, instead of a local copy.
- Tests: `internal/ui/widgets/template_field_test.go` (click on a token opens the variable and does not edit; a double click opens it once; a click on the path edits). `internal/ui/app/edit_variable_test.go` (the editor opens on the variable with a Back; a missing variable is created in the draft; the drawn editor shows the focus once). `go test ./internal/ui/...` and `go vet ./internal/ui/...` green; `go build ./...` clean.
- Notes: the earlier single-click-edits / double-click-opens split was dropped by product decision: MyGo gives a click an element handled to no one else, so the field could not also see it, and the field's swap to its input between two clicks lost the double. "A click on a token opens the variable" is the one gesture that holds in both.
