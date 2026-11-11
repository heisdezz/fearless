# UI-WIDGETS — Custom Widgets for Fearless GUI

Owner history: antigravity (claimed 2026-11-11T03:10Z)

## Entries

### 2026-11-11T03:10Z — antigravity

- Did: Claimed `UI-WIDGETS`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/ui/components.md` §6.6.
- Tests: not run yet.
- Notes: Implementing the custom widgets from `ui/components.md` §6.6 in `internal/ui/widgets/`:
  1. `MethodBadge`: HTTP method chip with per-verb styling from `internal/ui/theme`.
  2. `KeyValueGrid`: Editable query/header/variable table with enabled toggle, key/value inputs, row deletion, and auto-append empty row.
  3. `RequestTabBar`: Reorderable tab bar with close buttons, dirty indicators, active tab indicator, and overflow dropdown.
  4. `CollectionTreeRow`: Tree row rendering collection/folder/request item with indentation, toggle chevron, method badge, and actions.
  5. `VariableInput`: Text input with autocomplete suggestions for `{{var}}` placeholders.
  6. `Splitter`: Flexible splitter helpers wrapping `ui.SplitVertical` / `ui.Split` / `ui.SplitPercent`.
  7. `VirtualizedList`: Windowed list helper for efficient rendering of large lists.
  8. `FreeTextSelect`: Combobox allowing selection from preset options or entering custom free-text.

### 2026-11-11T03:38Z — antigravity

- Did: Implemented all 8 custom widgets plus `CodeEditor` with `{{var}}` completion in `internal/ui/widgets/`:
  - `method_badge.go`: `MethodBadge` HTTP verb badge with dynamic token colors from `internal/ui/theme`.
  - `keyvalue_grid.go`: `KeyValueGrid` editable parameter table with enable checkboxes, key/value inputs, delete buttons, and dynamic row addition.
  - `tab_bar.go`: `RequestTabBar` request tab bar with active highlights, dirty dot indicators, and close/new-tab actions.
  - `tree_row.go`: `CollectionTreeRow` hierarchical tree row with indentation, expand/collapse chevrons, method badges, and selection.
  - `variable_input.go`: `VariableInput` input field with `{{` trigger autocomplete dropdown.
  - `splitter.go`: `Splitter` horizontal and vertical resizable percentage splitters.
  - `virtualized_list.go`: `VirtualizedList` windowed virtualized list calculating visible window and spacer boxes.
  - `freetext_select.go`: `FreeTextSelect` editable combobox combining text input with option suggestion list.
  - `code_editor.go`: `CodeEditor` wrapping MyGo's `ui/editor` with syntax highlighting and variable completion.
  - `widgets_test.go`: Comprehensive headless testing with `ui.NewTester` testing all widgets, click interactions, text rendering, and accessibility trees.
- Tests: `go test -race -cover ./internal/ui/widgets/...` (94.2% coverage), `go vet ./internal/ui/widgets/...`, `gofmt`, and `./scripts/check-deps.sh` all passed cleanly.
- Status: Moving `UI-WIDGETS` to `InReview`.
