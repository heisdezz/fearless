# UI Map — Bruno UI reconstruction specification for a Go GUI

Status: IN PROGRESS. This directory is a framework-agnostic description of
Bruno's interface as it ships, written to rebuild it in Go with MyGo + Lobe
MyGo. It covers **only the UI**; backend, file IO, request execution and
parsing are out of scope.

**Reference target:** `bruno/` at commit `60d7517d0545ef82c032c10dcbb6a3c815d246cf`
(`bruno/packages/bruno-app`). All paths in this map are relative to
`bruno/packages/bruno-app/src/` unless stated otherwise.

**Screenshots:** `refs/*.png` (16 captures of the real app). Each screen file
names the captures that show it.

---

## How to read this map

| File               | Holds                                                                                                   |
| ------------------ | ------------------------------------------------------------------------------------------------------- |
| `overview.md`      | Package structure, how to run Bruno to view it, component index, checklist                              |
| `design-system.md` | Exact tokens per theme: colour, type, spacing, radius, border, shadow, motion                           |
| `screens/*.md`     | One file per screen or dialog: ASCII layout, dimensions, states, verbatim copy, interactions, shortcuts |
| `components/*.md`  | One file per reusable component: visual spec, variants, states                                          |
| `widgets.md`       | Rich-widget requirements ranked MUST-HAVE / NICE-TO-HAVE, editor syntax colours                         |
| `go-ui-notes.md`   | Build order, browser-only behaviour needing a native approach, open questions                           |

## Checklist — every `src/components/*` entry appears in the map

Legend: `S` = covered by a screen file, `C` = its own component file,
`W` = covered in `widgets.md`, `—` = out of UI scope (documented why).

| Component                   | Where                                                                               |
| --------------------------- | ----------------------------------------------------------------------------------- |
| Accordion                   | components/misc-primitives.md                                                            |
| AIAssist                    | — (deferred: AI is out of v1 scope, `docs/research/unresolved-questions.md` §UQ-D6) |
| AiChatSidebar               | — (deferred: AI)                                                                    |
| ApiSpecPanel                | — (deferred: OpenAPI sync is out of v1 scope)                                       |
| ApiSpecTab                  | — (deferred: OpenAPI)                                                               |
| AppPreviewKeepAlive         | go-ui-notes.md (browser-only pane retention)                                        |
| AppTitleBar                 | screens/app-shell.md                                                                |
| AppView                     | screens/app-shell.md                                                                |
| AppWebviewPane              | — (out of scope: webview pane)                                                      |
| BodyModeSelector            | components/select.md (a Select)                                                     |
| Bruno                       | screens/about.md                                                                    |
| BrunoSupport                | screens/about.md                                                                    |
| BulkEditor                  | widgets.md                                                                          |
| ChangelogTab                | screens/about.md                                                                    |
| CodeEditor                  | widgets.md                                                                          |
| CodeMirrorSearch            | widgets.md                                                                          |
| CodeSnippet                 | components/misc-primitives.md                                                          |
| CollectionApp               | screens/app-shell.md                                                                |
| CollectionSettings          | screens/collection-settings.md                                                      |
| ColorBadge                  | components/misc-primitives.md                                                           |
| ColorPicker                 | components/misc-primitives.md                                                          |
| ColorRange                  | components/misc-primitives.md                                                          |
| Cookies                     | screens/cookies.md                                                                  |
| CreateTransientRequest      | screens/request-tabs.md                                                             |
| CreateUntitledRequest       | screens/request-tabs.md                                                             |
| DataTypeSelector            | components/select.md                                                                |
| DeprecationWarning          | components/status-badge.md                                                          |
| Devtools                    | — (out of scope: developer tools)                                                   |
| DndTargetErrorGuard         | go-ui-notes.md (drag-and-drop guard)                                                |
| Documentation               | screens/request-docs.md                                                             |
| Dropdown                    | components/select.md                                                              |
| EditableTable               | widgets.md                                                                          |
| EmptyAppState               | screens/welcome.md                                                                  |
| Environments                | screens/environments.md                                                             |
| EnvironmentVariablesTable   | screens/environments.md                                                             |
| ErrorCapture                | go-ui-notes.md (error boundary)                                                     |
| Errors                      | components/misc-primitives.md                                                          |
| FileEditor                  | screens/request-body.md (file body)                                                 |
| FilePickerEditor            | components/text-input.md                                                            |
| FilterDropdown              | components/select.md                                                              |
| FolderSettings              | screens/folder-settings.md                                                          |
| Git                         | — (out of v1 UI scope: source control pane; see go-ui-notes.md)                     |
| GlobalSearchModal           | screens/global-search.md                                                            |
| Help                        | screens/about.md                                                                     |
| Icons                       | design-system.md (§ icons)                                                          |
| IncludeExcludeTags          | screens/runner.md                                                                   |
| InfoTip                     | components/misc-primitives.md                                                               |
| InheritableSettingsInput    | components/text-input.md                                                            |
| ManageWorkspace             | screens/workspace-switcher.md                                                       |
| MarkDown                    | widgets.md                                                                          |
| MigrateCollectionToYmlModal | screens/collection-settings.md                                                      |
| MockServer                  | — (deferred: mock server out of v1 scope)                                           |
| Modal                       | components/misc-primitives.md                                                                 |
| ModeSwitch                  | components/toggle.md                                                                |
| MultiLineEditor             | widgets.md                                                                          |
| MultipartFileChipsCell      | widgets.md (multipart cells)                                                        |
| Notifications               | screens/notifications.md                                                            |
| OpenAPISpecTab              | — (deferred: OpenAPI)                                                               |
| OpenAPISyncTab              | — (deferred: OpenAPI)                                                               |
| PathDisplay                 | components/text-input.md                                                            |
| Portal                      | go-ui-notes.md (overlay hosting)                                                    |
| Preferences                 | screens/preferences.md                                                              |
| RadioButton                 | components/misc-primitives.md                                                          |
| ReorderTable                | widgets.md                                                                          |
| RequestPane                 | screens/request-pane.md                                                             |
| RequestTabPanel             | screens/request-pane.md                                                             |
| RequestTabs                 | screens/request-tabs.md                                                             |
| ResponseExample             | screens/request-docs.md (§ Response example)                                                         |
| ResponsePane                | screens/response-pane.md                                                            |
| RunnerResults               | screens/runner.md                                                                   |
| SaveTransientRequest        | screens/request-tabs.md                                                             |
| SearchInput                 | components/text-input.md                                                            |
| SecuritySettings            | screens/preferences.md                                                              |
| SelectionFooter             | screens/runner.md                                                                   |
| SelectionList               | screens/runner.md                                                                   |
| SensitiveFieldWarning       | components/text-input.md                                                            |
| SetAsVariableMenu           | components/select.md                                                              |
| SetAsVariablePopover        | components/select.md                                                              |
| SettingsAiAssist            | — (deferred: AI)                                                                    |
| SettingsInput               | components/text-input.md                                                            |
| ShareCollection             | screens/collection-settings.md                                                      |
| Sidebar                     | screens/sidebar.md                                                                  |
| SingleLineEditor            | widgets.md                                                                          |
| SkippedPathsWarning         | screens/import-export.md                                                            |
| Spinner                     | components/spinner.md                                                               |
| StatusBar                   | screens/app-shell.md                                                                |
| StatusDot                   | components/status-dot.md                                                            |
| StopWatch                   | components/status-badge.md                                                          |
| Tab                         | components/tabs.md                                                                  |
| Table                       | widgets.md                                                                          |
| Tabs                        | components/tabs.md                                                                  |
| TagList                     | components/misc-primitives.md                                                                   |
| TagPicker                   | components/misc-primitives.md                                                                   |
| Toast                       | components/misc-primitives.md                                                                 |
| ToggleSwitch                | components/toggle.md                                                                |
| ToolHint                    | components/misc-primitives.md                                                               |
| TruncatedText               | components/text-input.md                                                            |
| VariableScopeIcon           | components/misc-primitives.md                                                        |
| VariableScopeList           | components/misc-primitives.md                                                        |
| VariablesEditor             | screens/environments.md                                                             |
| VarValueCell                | screens/environments.md                                                             |
| WelcomeModal                | screens/welcome.md                                                                  |
| WorkspaceHome               | screens/welcome.md                                                                  |
| WorkspaceSidebar            | screens/workspace-switcher.md                                                       |

Also mapped although they are sub-entries rather than top-level folders:
`src/ui/*` primitives (Button, InputWrapper, ListGroup, ResponsiveTabs,
RichTextEditor, SegmentedControl, Select, MaskedInput, MenuDropdown, etc.) →
`components/*`.

---

## Package structure

Bruno is a Lerna monorepo (`bruno/package.json`, `bruno/lerna.json`). Only one
package is in scope for this map:

```text
packages/bruno-app/
├── package.json          # dev = rsbuild dev, storybook = storybook dev
├── rsbuild.config.js     # web build config
└── src/
    ├── pages/            # route/entry shells: Bruno/, Main.js, index.js
    ├── components/       # 104 top-level UI feature folders (the bulk of the UI)
    ├── ui/               # ~25 shared primitives (Button, Select, Toggle, …)
    ├── themes/           # token sets: light/, dark/, DesignSystem/, schema/
    ├── styles/           # global styles, editor theme, tailwind entry
    ├── i18n/             # translation catalogue (labels resolve here)
    ├── providers/        # React context providers (theme, redux, toast)
    ├── selectors/, utils/, hooks/
    └── assets/           # images, logos, fonts used by screens
```

Sources: `packages/bruno-app/package.json:6-14`,
`packages/bruno-app/src/` directory listing.

**Component families** (from `src/components/`, counts are files per folder):

| Family        | Folders                                                                                                  | Purpose in the map                                                                       |
| ------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| Shell         | `AppTitleBar` (5), `AppView` (6), `CollectionApp` (2), `StatusBar` (4), `WorkspaceSidebar` (2)           | `screens/app-shell.md`                                                                   |
| Sidebar       | `Sidebar` (163 incl. sub-folders)                                                                        | `screens/sidebar.md`                                                                     |
| Tabs          | `RequestTabs` (21), `Tab` (1), `Tabs` (3)                                                                | `screens/request-tabs.md`, `components/tabs.md`                                          |
| Request       | `RequestPane` (140 incl. 60 sub-folders), `RequestTabPanel` (15)                                         | `screens/request-pane.md` + per-sub-tab files                                            |
| Response      | `ResponsePane` (130 incl. sub-folders)                                                                   | `screens/response-pane.md`                                                               |
| Environments  | `Environments` (58), `EnvironmentVariablesTable` (8), `VariablesEditor` (13)                             | `screens/environments.md`                                                                |
| Settings      | `Preferences` (42), `CollectionSettings` (53), `FolderSettings` (20)                                     | `screens/preferences.md`, `screens/collection-settings.md`, `screens/folder-settings.md` |
| Runner        | `RunnerResults` (9), `SelectionList` (3), `SelectionFooter` (1), `IncludeExcludeTags` (3)                | `screens/runner.md`                                                                      |
| Import/export | `Sidebar/ImportCollection*`, `Sidebar/BulkImportCollectionLocation`, `SkippedPathsWarning` (2)           | `screens/import-export.md`                                                               |
| Editors       | `CodeEditor` (8), `SingleLineEditor` (2), `MultiLineEditor` (3), `MarkDown` (1), `CodeMirrorSearch` (11) | `widgets.md`                                                                             |
| Tables        | `Table` (2), `EditableTable` (4), `ReorderTable` (1), `BulkEditor` (1)                                   | `widgets.md`                                                                             |
| Primitives    | `src/ui/*` (25)                                                                                          | `components/*`                                                                           |

**Out of scope in this pass** (documented, not specified in detail): AI assist
(`AIAssist`, `AiChatSidebar`, `SettingsAiAssist`), OpenAPI sync/spec
(`ApiSpecPanel`, `ApiSpecTab`, `OpenAPISpecTab`, `OpenAPISyncTab`), mock server
(`MockServer`), Git pane (`Git`), devtools (`Devtools`), webview pane
(`AppWebviewPane`). These are deferred from v1 per
`docs/research/unresolved-questions.md` §UQ-D6 and the roadmap.

---

## How to run Bruno to view it

From `bruno/` (the verified scripts):

```bash
npm install                     # once, hoisted workspace install
npm run dev:web                 # rsbuild dev server for bruno-app (browser)
npm run dev:electron            # full desktop app (Electron + bruno-app)
npm run storybook               # Storybook on :6006 — component/theme previews
```

- `npm run dev:web` runs `rsbuild dev` in `packages/bruno-app`
  (`packages/bruno-app/package.json:7`). The browser build has no Electron
  APIs; use it for pure layout/visual checks only.
- `npm run dev:electron` (`bruno/package.json:67`) runs the real desktop app —
  **use this for interaction and chrome checks** (title bar, native menus,
  custom title bar insets).
- Storybook (`bruno/package.json:69`) publishes `themes/DesignSystem` and
  `ui/*` stories: the fastest way to see every theme's tokens
  (`themes/DesignSystem/DesignSystem.stories.jsx`, `themes/PaletteViewer/`).

Screenshots already captured from a running instance live in `refs/`. Prefer
those over re-running when a state is already covered; capture a new one only
for a state `refs/` lacks.

---

## Component index

See the Checklist table above: it is the authoritative index of every
`src/components/*` folder and where it is specified. The `src/ui/*` primitives
are indexed by `components/` file in this table:

| Primitive (`src/ui/`)                                         | Component file                               |
| ------------------------------------------------------------- | -------------------------------------------- |
| `Button`                                                      | components/button.md                         |
| `Checkbox`                                                    | components/misc-primitives.md                       |
| `InputWrapper`, `TextInput`, `MaskedInput`, `MaskedTextfield` | components/text-input.md                     |
| `Select`, `Dropdown`, `MenuDropdown`                          | components/select.md, components/select.md |
| `Toggle`                                                      | components/toggle.md                         |
| `SegmentedControl`, `ResponsiveTabs`, `Tabs`                  | components/tabs.md                           |
| `ListGroup`                                                   | components/misc-primitives.md                     |
| `MethodBadge`                                                 | components/method-badge.md                   |
| `StatusBadge`, `CountBadge`                                   | components/status-badge.md                   |
| `ErrorBanner`                                                 | components/misc-primitives.md                   |
| `ContextMenu`                                                 | components/misc-primitives.md                   |
| `RichTextEditor`                                              | widgets.md                                   |
| `HeightBoundContainer`, `Portal`                              | go-ui-notes.md                               |
