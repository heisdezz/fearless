# Settings

Status: VERIFIED against `bruno/packages/bruno-app/src/components/Preferences/` and `CollectionSettings/` at commit `60d7517`, cross-referenced with `ui-map/screens/preferences.md`, `collection-settings.md`, and `folder-settings.md`.

---

## 1. Setting Hierarchy

Settings operate at three explicit levels:

| Level                   | Access Point                                     | Storage Target                       | Scope                  |
| ----------------------- | ------------------------------------------------ | ------------------------------------ | ---------------------- |
| **App Preferences**     | Preferences modal (**⌘/Ctrl + ,**) or status bar | Local app store (`preferences.json`) | Global to application  |
| **Collection Settings** | Collection context menu → Settings               | `bruno.json` in collection root      | Entire collection tree |
| **Folder Settings**     | Folder context menu → Settings                   | `folder.bru` in directory            | Subtree of requests    |
| **Request Settings**    | Request editor → Settings tab                    | `request.bru` metadata               | Individual request     |

---

## 2. App Preferences Specification (`ui-map/screens/preferences.md`)

Modal dialog width: `720px`, body height: `520px`. Left sidebar with tabs, right content panel:

### 2.1 Tab Inventory and Controls

| Tab                     | Key Controls                                                                                                                                                                                                                                                                                                                                                                                                                                               | Engine Impact                                    |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| **General**             | - SSL/TLS certificate verification toggle<br/>- Custom CA certificate picker (adds to root pool)<br/>- Store cookies automatically (`shouldStoreCookies`)<br/>- Send cookies automatically (`shouldSendCookies`)<br/>- Authorize OAuth2 in system browser toggle<br/>- Enable auto-save toggle and delay (`autoSaveDelay`, default 1000 ms)<br/>- Request timeout (`request.timeout`, default 0 = unlimited)<br/>- Default collection location path picker | Direct transport, cookie jar, and disk IO impact |
| **Themes**              | - Mode selector: System / Light / Dark<br/>- Theme presets grid (Light, Dark, Catppuccin, Nord, Dracula, Gruvbox, VS Code)                                                                                                                                                                                                                                                                                                                                 | Swaps active token adapter                       |
| **Display**             | - Font size slider / dropdown (11–18 px)<br/>- Interface zoom scale (80 % – 150 %)<br/>- Code editor font family string                                                                                                                                                                                                                                                                                                                                    | Re-scales DIPs in UI frame                       |
| **Proxy**               | - Mode: `off` (direct), `on` (manual), `system` (OS settings), `pac` (script)<br/>- Protocol (`http`/`https`), Hostname, Port<br/>- Auth: Username & masked password<br/>- Bypass list (comma-separated domains)<br/>- PAC script URL / file path                                                                                                                                                                                                          | Configures HTTP engine transport agents          |
| **Client Certificates** | - List of domains and certificate pairs (`cert` + `key` or `.pfx`)<br/>- Passphrase inputs (masked with `*` per R-5)                                                                                                                                                                                                                                                                                                                                       | TLS mutual authentication (mTLS)                 |
| **Keybindings**         | - Searchable table of all actions and current shortcuts<br/>- Custom shortcut capture input<br/>- Reset to defaults button                                                                                                                                                                                                                                                                                                                                 | Global and view shortcut routing                 |
| **Support & About**     | - Fearless version string, commit hash, build date<br/>- Links to repository, issue tracker, and documentation                                                                                                                                                                                                                                                                                                                                             | Information only                                 |

---

## 3. Collection and Folder Settings

### 3.1 Collection Settings (`ui-map/screens/collection-settings.md`)

- **Shared Headers:** Key-value table inherited by all requests in the collection.
- **Shared Auth:** Default authentication provider for requests set to `mode: inherit`.
- **Shared Variables & Scripts:** Pre-request and post-response logic applied globally across the collection.
- **Proxy Override:** Per-collection proxy configuration overriding global preferences.
- **Client Certificates:** Collection-specific mTLS configurations stored in `bruno.json`.

### 3.2 Folder Settings (`ui-map/screens/folder-settings.md`)

- Folder-level headers, auth, variables, and scripts stored in `folder.bru`.
- Supports nested inheritance chains where a child folder overrides a parent folder's configuration.

---

## 4. Control Dimensions & Shimming (D-019)

- Form inputs use `SettingsInput` geometry: height `32px` (`2rem`), width `96px` (`w-24`) for numeric inputs or full width for strings.
- Checkboxes and toggles render through `internal/ui/widgets/controls.go` to maintain dark-theme background integrity.
