# Interaction Patterns

Status: VERIFIED against `bruno/packages/bruno-app/src/` at commit `60d7517` and cross-referenced with `ui-map/decisions.md` and `ui-map/go-ui-notes.md`.

---

## 1. Draft Interaction Model

The central editing paradigm: **every edit modifies an in-memory draft (`item.draft`); disk is never touched until explicit save or autosave.**

```javascript
if (!item.draft) {
  item.draft = cloneDeep(item);
}
item.draft.request.url = action.payload.url;
```

### Visual Indicators and State Transitions

| State / Trigger           | Indicator in UI                                   | Behavior / Guard                                                   |
| ------------------------- | ------------------------------------------------- | ------------------------------------------------------------------ |
| **Unsaved Edit**          | Tab dirty dot (`draftColor` `#cc7b1b`)            | `hasRequestChanges(item)` evaluates to true; Save button activates |
| **Manual Save**           | **⌘/Ctrl + S** or Save button click               | Serializes `item.draft` to `.bru` file on disk; clears draft       |
| **Auto-Save**             | Debounced timer (`autoSaveDelay`, default 1000ms) | Triggers save automatically when enabled in Preferences            |
| **Closing Dirty Tab**     | `SaveRequestsModal` / `ConfirmAppClose`           | Prompts user: Save changes, Discard, or Cancel close               |
| **External Modification** | File watcher event                                | If clean, auto-reloads from disk; if dirty, shows collision banner |

---

## 2. Complete Keyboard Shortcut Registry

Hotkeys are registered globally through the keybinding router (R-7). Combinations serialize with `+bind+` across platforms:

| Action Group   | Shortcut (macOS / Linux & Windows)   | Target Action / Flow                                         |
| -------------- | ------------------------------------ | ------------------------------------------------------------ |
| **Requests**   | `⌘ + Enter` / `Ctrl + Enter`         | Send active request (or Cancel if sending)                   |
|                | `⌘ + S` / `Ctrl + S`                 | Save active request draft                                    |
|                | `⌘ + J` / `Ctrl + J`                 | Toggle layout: Stacked vs Side-by-side (R-7)                 |
|                | `Alt + Shift + F`                    | Auto-format / prettify request body                          |
| **Tabs**       | `⌘ + W` / `Ctrl + W`                 | Close active tab                                             |
|                | `⌘ + Shift + T` / `Ctrl + Shift + T` | Reopen last closed tab (`reopenClosedTab`)                   |
|                | `⌘ + 1..8` / `Ctrl + 1..8`           | Switch to tab index 1 through 8                              |
| **Sidebar**    | `⌘ + \` / `Ctrl + \`                 | Toggle sidebar expand / collapse                             |
|                | `⌘ + F` / `Ctrl + F`                 | Focus sidebar search filter input                            |
|                | `Escape`                             | Clear sidebar selection (`useClearSidebarSelectionOnEscape`) |
| **Navigation** | `⌘ + K` / `Ctrl + K`                 | Open Global Search modal                                     |
|                | `⌘ + E` / `Ctrl + E`                 | Open active Environment settings tab                         |
|                | `⌘ + ,` / `Ctrl + ,`                 | Open Preferences modal                                       |
|                | `⌘ + O` / `Ctrl + O`                 | Open / Import Collection wizard                              |
|                | `⌘ + N` / `Ctrl + N`                 | Create new request in active collection                      |
| **Editor**     | `Alt + Space` / `Ctrl + Space`       | Trigger variable autocomplete list                           |

---

## 3. Variable Autocomplete Flow (`{{...}}`)

Triggered inside any editor input (URL bar, headers, body, auth, scripts):

1. Typing `{{` opens a floating completion popup.
2. Computes in-scope variables grouped by hierarchy: Runtime, Request, Folder, Collection, Environment, Global, Process.
3. Displays variable name, resolved value preview, and provenance badge.
4. Arrow keys select; `Enter` or `Tab` completes `{{name}}`; `Escape` closes popup.

---

## 4. Set-as-Variable Flow

Selecting text in response body or headers brings up the context menu:

1. User clicks **Set as variable** (`SetAsVariableMenu`).
2. Popover prompts for variable name (pre-populated from JSONPath or header key).
3. User selects target scope: `Collection`, `Environment`, or `Runtime`.
4. Option to automatically replace selected text in the active request with `{{name}}`.
5. Commits value immediately to target scope.

---

## 5. Secret Masking Standard (Decision R-5)

> [!IMPORTANT]
> **Decision R-5:** All secret fields across the entire interface (environment variables, auth passwords, bearer tokens, client certificate passphrases) strictly display **one `*` per character**.
>
> - Eye toggle icon reveals plaintext on click.
> - Clipboard copy copies the actual unmasked secret.

---

## 6. Drag-and-Drop Feedback

- **Valid Target:** Displays a 2 px solid highlight in `dragAndDrop.border` (`#d37f17` L / `#e4ae49` D) with `hoverBg`.
- **Invalid / Blocked Target:** Displays blocked cursor (`dragBlockedCursor.js`) with red no-drop badge when hovering outside allowed boundaries (e.g. cross-collection moves).
- **Tab Reordering:** Dragging a tab horizontally swaps positions live with animated layout transitions.

---

## 7. Dialog and Modal Patterns (`ui-map/screens/sidebar-dialogs.md`)

- **Focus Trap:** All modals implement `useFocusTrap` — focus stays within modal until dismissed.
- **Escape Key:** Always closes the active modal or dialog without applying changes.
- **Enter Key:** Submits the primary modal action when focused on single-line inputs.
- **Destructive Confirmations:** Deleting requests, folders, or environments requires confirmation with a red primary action button (`status.danger.background`).
