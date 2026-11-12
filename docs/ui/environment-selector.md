# Environment Selector

Status: VERIFIED against `bruno/packages/bruno-app/src/components/Environments/` at commit `60d7517` and cross-referenced with `ui-map/screens/environments.md`.

---

## 1. Two-Tier Environment Model

`EnvironmentSelector` renders a split popover menu:

```text
┌────────────────────────────────────────────────────────┐
│ [ Collection Environments ]   [ Global Environments ]  │
├────────────────────────────────────────────────────────┤
│ ◉ Development                    [Inherits: Base] [🔒] │
│ ○ Staging                                         [🔒] │
│ ○ Production                                      [🔒] │
├────────────────────────────────────────────────────────┤
│ [+ New Environment]    [Import Environment]   [⚙ Manage]│
└────────────────────────────────────────────────────────┘
```

| Scope          | Storage Mechanism                                                          | Visibility & Lifecycle                                            |
| -------------- | -------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Collection** | Saved as `.bru` files in `environments/*.bru` within the collection folder | Scoped to active collection; checked into version control         |
| **Global**     | Persisted in app-level store (`global-environments`)                       | Available across all open collections; local to developer machine |

---

## 2. Environment Selector Menu

- **Active Badge:** Active environment appears as a badge on the collection header in the sidebar. Colors derive from `environment.color` rendered at 12 % alpha background (`getEnvBadgeStyle`).
- **Inheritance Marker:** Environments declaring an `extends` parent display an inheritance tag linking to the parent environment.
- **Secret Indicator:** Locked padlock icon indicates variables stored in secret storage.
- **Import Actions:** Postman environment JSON, `.env` files, and Insomnia environments.

---

## 3. Environment Editor Specification

Opened via **⌘/Ctrl + E** or from the selector's configure action.

### 3.1 Variables Grid

| Column          | Component & Behavior        | Spec / Decision Reference                                |
| --------------- | --------------------------- | -------------------------------------------------------- |
| **Enabled**     | Checkbox toggle             | Controls whether variable participates in interpolation  |
| **Name**        | Monospace text field        | Validates against `^[A-Za-z0-9_.-]+$`; flags duplicates  |
| **Value**       | Monospace editor            | Masked with `*` when secret; reveals on hover / focus    |
| **Type**        | `DataTypeSelector` dropdown | `string`, `number`, `boolean`, `object`                  |
| **Secret**      | Lock toggle switch          | Moves value to encrypted store; keeps only key in `.bru` |
| **Description** | Plain text input            | Freeform documentation                                   |

> [!IMPORTANT]
> **Decision R-5 (Secret Masking):** Secret variable values render strictly as **one `*` per character** everywhere in the UI (tables, tooltips, dialogs), matching user expectations.

---

## 4. DotEnv Integration

- Collection `.env` files are monitored via filesystem watcher.
- Values become available during interpolation under `process.env.*`.
- **Precedence Hierarchy (VERIFIED):**
  ```text
  Collection .env  >  Workspace .env  >  Host OS process.env
  ```

---

## 5. Effective Variable Inspector

- Hovering an interpolated `{{variable}}` anywhere in the app displays an inspection popover showing:
  - Effective resolved value (with secret masking if secret)
  - Provenance source (Request runtime, Folder, Collection, Environment, Global, System)
  - Full scope inheritance chain
