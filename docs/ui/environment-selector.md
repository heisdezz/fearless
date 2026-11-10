# Environment Selector

Status: VERIFIED unless marked.

Root: `packages/bruno-app/src/components/Environments/`

---

## 1. Two-tier selector

`EnvironmentSelector` opens a dropdown with two tabs:

```text
[ Collection ]  [ Global ]
```

| Tab | Scope | Storage |
|---|---|---|
| Collection | Per-collection environment | `environments/*.bru` in the collection |
| Global | App-wide environment | `electron-store` `global-environments` |

Each tab lists environments with radio selection, plus actions (create, import, edit, delete, duplicate). Selecting dispatches `selectEnvironment` (collection) or `selectGlobalEnvironment` (global). The current selection is shown as a badge on the collection row in the sidebar.

Source: `Environments/EnvironmentSelector/index.js` (`TABS`, `getEnvBadgeStyle`), `EnvironmentListContent`.

---

## 2. Environment list features

| Feature | Detail |
|---|---|
| Color tags | `environment.color`; badge shows color at 12 % alpha |
| Inheritance | Environments with `extends` show an "inherited" marker and a parent chip |
| Secret indicator | Lock icon; values not shown until revealed |
| Active marker | Radio + check icon |
| Import | `ImportEnvironmentModal` (from `.env`, Postman env JSON, Insomnia) |
| Create | Inline modal/tab |
| Search | Filter within the list |

---

## 3. Environment editor (tab)

Opened via `editEnvironment` (⌘/Ctrl + E) or the selector's edit action. `Environments/EnvironmentSettings` renders:

```text
┌ Name · Color · [Extends ▾] ────────────────────────────┐
│ Variables table                                        │
│  ☑ name | value | type | secret | description          │
│  + Add variable                                        │
├ Secret values are stored encrypted                     │
├ Dotenv file section (optional)                         │
└────────────────────────────────────────────────────────┘
```

| Control | Behavior |
|---|---|
| Variable row | enabled checkbox, name, value (masked if secret), `DataTypeSelector` (string/number/boolean/object), secret toggle, description |
| Secret toggle | Moves the value to the encrypted store; file keeps only the name in `vars:secret` |
| `Extends` | Choose a parent environment; cycles and missing parents are flagged |
| Validation | Duplicate names, invalid characters (`[A-Za-z0-9_.-]`), missing parent shown inline |
| Save | `saveEnvironment` thunk → file write, or global store update |
| Discard | `ConfirmCloseEnvironment` warns on unsaved changes |

`EnvironmentVariablesTable`, `VarValueCell`, `VariableScopeIcon` provide the shared table UX.

---

## 4. Dotenv integration

`DotEnvFileEditor` / `DotEnvFileDetails` manage the collection's `.env` file, whose values become `process.env.*` during interpolation.

Priority (VERIFIED, `bruno-electron/src/store/process-env.js`):

```text
collection .env  >  workspace .env  >  OS process.env
```

The file header comment states this explicitly. Lines are watched by `app/dotenv-watcher.js` so edits apply without reopening.

---

## 5. Global environments editor

`WorkspaceHome/WorkspaceEnvironments/*` and `Preferences`-adjacent UI manage app-level environments. Secret values are encrypted with the same mechanism (`encryptStringSafe`/`decryptStringSafe`).

Workspaces add a **third** source: workspace environments (`store/workspace-environments.js`), selectable per workspace.

---

## 6. Effective value display

When a variable is defined in multiple scopes, the UI shows the winner and its provenance (`VariableScopeList`, `VariableScopeIcon`), matching the precedence in `flows/environment-resolution.md`.

---

## 7. Go port guidance (PROPOSED)

- Keep the two-tab selector; global environments are a real feature, not a duplicate.
- Put the encryption in the core with a pluggable key source (OS keychain preferred).
- Implement `extends` validation (cycle detection) in core so the CLI rejects invalid envs identically.
- Show provenance; it is the main debugging aid for variables.
