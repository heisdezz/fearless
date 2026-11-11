# Screen — Preferences

All paths are relative to `bruno/packages/bruno-app/src/` unless prefixed with
`design-system.md` (that file is `ui-map/design-system.md`).

Status: VERIFIED against source. Points that could not be confirmed inside the
repository are marked **UNVERIFIED** inline and repeated under §10 Open Questions.

Scope: `components/Preferences/**` (42 files) plus the shared pieces it renders
(`components/Preferences/SettingsLayout/*`, `components/ListGroup`,
`components/ToggleSwitch`, `components/Modal`) and the key-binding data in
`providers/Hotkeys/keyMappings.js`.

> **Terminology note (important for the port).** Bruno's "Preferences" screen is
> NOT the tab set the brief assumed. Source has ten tabs — **General, Themes,
> Display, Proxy, Client Certificates, Keybindings, AI, Cache, Support, Beta**
> (`components/Preferences/index.js:94-133`). There is no "Request", "Font",
> "Privacy", "Shortcuts" or "About" tab as such: request behaviour lives inside
> **General**, font lives inside **Display**, privacy lives inside **AI ›
> Security**, shortcuts are **Keybindings**, and there is no About tab at all.
> `SecuritySettings` is not a Preferences tab — it is the per-collection
> JavaScript-sandbox selector (`components/SecuritySettings/JsSandboxMode/index.js`),
> documented in `collection-settings.md` §7.5.

---

## 1. Layout

```text
┌ Preferences (tab type "preferences") ───────────────────────────────┐
│ ┌ tablist ─────┐  ┌ tab-panel (scrolls) ──────────────────────────┐ │
│ │ ⚙ General    │  │ GENERAL SETTINGS                              │ │
│ │ 🎨 Themes    │  │ Certificates                                   │ │
│ │ 🖥 Display   │  │  [x] SSL/TLS Certificate Verification          │ │
│ │ ◔ Proxy     │  │  [ ] Use Custom CA Certificate                 │ │
│ │ ▤ Client…   │  │        [Select File]                           │ │
│ │ ⌨ Keybind…  │  │        [ ] Keep Default CA Certificates        │ │
│ │ ✨ AI        │  │ ───────────────────────────────────────────────│ │
│ │ ▣ Cache     │  │ Cookies & Authorization                        │ │
│ │ ? Support   │  │  [x] Store Cookies automatically               │ │
│ │ B Beta      │  │  …                                             │ │
│ └──────────────┘  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

| Region | Value | Source |
| --- | --- | --- |
| Root | `h-full`, flex row, `gap-2` | `components/Preferences/index.js:91-92` |
| Tab rail | flex column, `padding: 12px`, `min-width: 180px` | `components/Preferences/StyledWrapper.js:4-6` |
| Tab item | `display:flex`, `align-items:center`, `gap: 8px`, full width, `padding: 6px 10px`, `border: none`, radius `border.radius.sm` (4px) | `components/Preferences/StyledWrapper.js:8-15` |
| Tab item idle text | `colors.text.muted` (`#838383` L / `#aaa` D) | `components/Preferences/StyledWrapper.js:16` |
| Tab item transition | `background-color 0.15s ease` | `components/Preferences/StyledWrapper.js:18` |
| Tab item active | text `text` (`#343434` L / `#cccccc` D), background `tabs.secondary.active.bg` (`#eaeaea` L / `#26292b` D) | `components/Preferences/StyledWrapper.js:29-36` |
| Tab icon | Tabler, `size={16}`, `strokeWidth={1.5}` | `components/Preferences/index.js:95` (all tabs identical) |
| Panel | `flex-grow`, `padding: 12px`, `padding-bottom: 0`, `overflow-y:auto`, `max-height:100%` | `components/Preferences/index.js:135-140`, `StyledWrapper.js:40-47` |
| Checkbox/radio accent | `workspace.accent` (`#b96f1d` L / `#d9a342` D) | `components/Preferences/StyledWrapper.js:49-52` |
| `.textbox` (shared input) | `line-height:1.5`, `padding:0.45rem`, radius 4px, bg `input.bg`, border `1px solid input.border`, text `text`; focus `1px solid input.focusBorder`; disabled `opacity:.5`, `cursor:not-allowed` | `components/Preferences/StyledWrapper.js:55-72` |
| `.section-header` | `font.size.sm` (12px), `colors.text.muted`, weight 500, `margin: 6px 0 8px`, uppercase, `letter-spacing: 0.5px` | `components/Preferences/StyledWrapper.js:73-80` |
| Default tab | `general` | `providers/ReduxStore/slices/app.js:37` |

### 1.1 Shared settings primitives (`components/Preferences/SettingsLayout/`)

`SettingsGroup({title, description, children})` — `SettingsLayout/SettingsGroup/index.js:4-12`

| Part | Style | Source |
| --- | --- | --- |
| Wrapper | column, `padding-top: 1rem`, `border-top: 1px solid border.border1` (`#e5e5e5` L / `#333` D); first group: no top padding, no border | `SettingsLayout/SettingsGroup/StyledWrapper.js:5-12` |
| Title (`h3`) | `font.size.base` (13px), weight 600, `text` | `SettingsLayout/SettingsGroup/StyledWrapper.js:14-19` |
| Description (`p`) | `font.size.xs` (11px), `colors.text.muted`, `margin: 0.25rem 0 0 0` | `SettingsLayout/SettingsGroup/StyledWrapper.js:21-25` |
| Body | column, `gap: 0.5rem`, `margin-top: 0.75rem` | `SettingsLayout/SettingsGroup/StyledWrapper.js:27-36` |

`CheckboxSetting({id, name, label, description, checked, onChange, disabled, children})` —
`SettingsLayout/CheckboxSetting/index.js:5-45`

| Part | Style | Source |
| --- | --- | --- |
| Control row | flex, `gap: 0.5rem` | `SettingsLayout/CheckboxSetting/StyledWrapper.js:7-10` |
| Input | native `type=checkbox`, class `mousetrap` | `CheckboxSetting/index.js:22-32` |
| Label | `font.size.base`, `text`, `cursor:pointer` | `SettingsLayout/CheckboxSetting/StyledWrapper.js:18-23` |
| Description | `font.size.xs`, `colors.text.muted`, `margin: 0.125rem 0 0 1.5rem` | `SettingsLayout/CheckboxSetting/StyledWrapper.js:25-29` |
| Children slot | column, `gap:0.5rem`, `margin: 0.3rem 0 0 1.5rem` (indented under the checkbox) | `SettingsLayout/CheckboxSetting/StyledWrapper.js:31-36` |
| Disabled | label `colors.text.muted`, `cursor:not-allowed`; checkbox `opacity:.5` | `SettingsLayout/CheckboxSetting/StyledWrapper.js:38-48` |

`SettingsField({label, htmlFor, hint, error, disabled, children})` —
`SettingsLayout/SettingsField/index.js:5-18`; label `font.size.base`/`text`
(`SettingsField/StyledWrapper.js:8-12`), hint `font.size.xs`/`colors.text.muted`
(`:14-18`), control row flex `gap:.5rem` `margin-top:.375rem` (`:20-25`),
error `font.size.xs` `colors.text.danger` `margin-top:.25rem` (`:27-31`),
disabled label → `colors.text.muted` (`:33-35`).

---

## 2. Tabs (exact order)

`components/Preferences/index.js:93-134`. Order is fixed; there is no reordering.

| # | id | Label (verbatim) | Icon (`@tabler/icons`) | Panel component | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | `general` | `General` | `IconSettings` | `General` (`components/Preferences/General/index.js`) | default tab |
| 2 | `themes` | `Themes` | `IconPalette` | `Themes` (`components/Preferences/Themes/index.js`) | |
| 3 | `display` | `Display` | `IconBrowser` | `Display` (`components/Preferences/Display/index.js`) | |
| 4 | `proxy` | `Proxy` | `IconUserCircle` | `ProxySettings` (`components/Preferences/ProxySettings/index.js`) | |
| 5 | `clientCert` | `Client Certificates` | `IconCertificate` | `ClientCertSettings` (`components/Preferences/ClientCertSettings/index.js`) | |
| 6 | `keybindings` | `Keybindings` | `IconKeyboard` | `Keybindings` (`components/Preferences/Keybindings/index.js`) | |
| 7 | `ai` | `AI` | `IconSparkles` (local icon component, `components/Icons/IconSparkles`) | `AI` (`components/Preferences/AI/index.js`) | **deferred** |
| 8 | `cache` | `Cache` | `IconDatabase` | `Cache` (`components/Preferences/Cache/index.js`) | |
| 9 | `support` | `Support` | `IconZoomQuestion` | `Support` (`components/Preferences/Support/index.js`) | |
| 10 | `beta` | `Beta` | `IconSquareLetterB` | `Beta` (`components/Preferences/Beta/index.js`) | |

Panel container is `<section role="tabpanel" id="${tab}-panel" aria-labelledby="${tab}-tab">`
(`components/Preferences/index.js:135-140`) — but the tab elements themselves have
no `id`/`aria-controls` (`:94-133`), so the ARIA wiring is one-sided. Worth
fixing in the port (add `id={`${tab}-tab`}`).

**AI tab — deferred.** As requested, one line only: *AI exposes three sub-tabs —
`Configuration`, `Autocomplete`, `Security` (`components/Preferences/AI/index.js:320,331,342`)
— containing provider cards, model selection, and redaction toggles; it is out
of v1 scope (see `overview.md` §"Out of scope").*

---

## 3. Open entry points and shortcuts

| Entry | Behaviour | Source |
| --- | --- | --- |
| Menu bar › File › Preferences | Sends `renderer:open-preferences` to the renderer; shortcut label shown `Ctrl+,` | `components/AppTitleBar/AppMenu/index.js:24-27` |
| Status bar gear button | Opens Preferences; `ToolHint` text `Preferences`; `aria-label="Open Preferences"` | `components/StatusBar/index.js:34,73-79` |
| Keyboard `Open Preferences` | Binding `command+,` (mac) / `ctrl+,` (win, linux) | `providers/Hotkeys/keyMappings.js:84` |
| Changelog links | `PREFERENCE_LINKS` map (currently empty) dispatches `updateActivePreferencesTab` then opens a `preferences` tab | `components/ChangelogTab/index.js:9-34` |

Preferences is a **tab** (`type: 'preferences'`, uid `` `${collectionUid}-preferences` `` or
`'preferences'`), not a modal (`components/ChangelogTab/index.js:27-33`).

---

## 4. Tab 1 — General

`components/Preferences/General/index.js`; styles `General/StyledWrapper.js`.

Heading: `.section-header` text **`General Settings`** (`:191`).
Form: `max-width: 40rem`, column, `gap: 1rem` (`General/StyledWrapper.js:11-16`).

### 4.1 Control table

Defaults come from `initialValues` with lodash `get(default)` fallbacks
(`General/index.js:66-86`). "Persistence" — General auto-saves on a 500 ms debounce
(`:133-152`) **and** on submit; there is no explicit Save button.

| # | Group | Label (verbatim) | Control | Default | Notes / validation | Source |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Certificates | `SSL/TLS Certificate Verification` | checkbox | `preferences.request.sslVerification` (no fallback — **undefined if unset**) | id `sslVerification` | `:22,194-200` |
| 2 | Certificates | `Use Custom CA Certificate` | checkbox | `false` (`request.customCaCertificate.enabled`) | When checked (and a file is chosen) unlocks row 4 | `:69,201-207` |
| 3 | Certificates | `Select File` | button, `IconUpload` 14px + label | — | Hidden `<input type=file>` id `caCertFilePath`; when a path exists the row is replaced by a chip showing `basename(path)` + a trash `IconTrash` 14px button (`aria-label="Remove custom CA certificate"`); wrapper `opacity:.5` when disabled; button style `padding:0.25rem 0.5rem`, radius 4px, `1px solid input.border`, hover border `input.focusBorder` | `:208-246`, `General/StyledWrapper.js:18-70` |
| 4 | Certificates | `Keep Default CA Certificates` | checkbox (nested under #2) | `true` (`request.keepDefaultCaCertificates.enabled`) | Disabled unless `customCaCertificate.enabled && filePath` | `:73,181,247-254` |
| 5 | Cookies & Authorization | `Store Cookies automatically` | checkbox | `true` | | `:76,259-265` |
| 6 | Cookies & Authorization | `Send Cookies automatically` | checkbox | `true` | | `:77,266-272` |
| 7 | Cookies & Authorization | `Use System Browser for OAuth2 Authorization` | checkbox | `false` | | `:83,273-279` |
| 8 | Auto Save | `Enable Auto Save` | checkbox | `false` (`autoSave.enabled`) | | `:79,282-289` |
| 9 | Auto Save | `Auto Save Delay (ms)` | text input (numeric), class `textbox numeric-input` (`max-width: 14rem`) | `1000` (`autoSave.interval`) | Disabled when #8 off. Errors: `Save Delay must be a number`, `Save Delay must be at least 500ms`, `Save Delay is required when Auto Save is enabled` | `:80,290-310`, `:45-58`, `General/StyledWrapper.js:72-75` |
| 10 | Requests | `Request Timeout (ms)` | text input (numeric) | `preferences.request.timeout` (no fallback) | Errors: `Request Timeout must be a number`, `Request Timeout must be greater than or equal to 0` | `:75,313-329`, `:32-42` |
| 11 | Default Location | *(no per-field label)*; group title `Default Location`, group description `Used as the default location for new workspaces and collections` | read-only text input (`aria-label="Default Location"`, placeholder `Click to browse for default location`) + `Browse` text-link button | `general.defaultLocation` ?? `''` | Clicking the input **or** `Browse` opens the platform directory picker; `defaultLocation` max 1024. Browse button: `font.size.sm`, `textLink`, underline on hover | `:62,331-362`, `General/StyledWrapper.js:77-101` |

Numeric field errors are gated on `touched` so they do not flash while typing
(`General/index.js:184-187`).

Non-verbatim strings (toast): `Failed to update preferences` (`:127`),
`Failed to browse for a default location` (`:175`).

### 4.2 Layout

```text
GENERAL SETTINGS
┌ Certificates ─────────────────────────────────────────────┐
│ [x] SSL/TLS Certificate Verification                       │
│ [x] Use Custom CA Certificate                              │
│       ┌ Select File ┐            (or)  ca.crt  🗑           │
│       [ ] Keep Default CA Certificates                     │
├ Cookies & Authorization ───────────────────────────────────┤
│ [x] Store Cookies automatically                            │
│ [x] Send Cookies automatically                             │
│ [ ] Use System Browser for OAuth2 Authorization            │
├ Auto Save ─────────────────────────────────────────────────┤
│ [ ] Enable Auto Save                                       │
│ Auto Save Delay (ms)                                       │
│ [ 1000            ]                                        │
├ Requests ──────────────────────────────────────────────────┤
│ Request Timeout (ms)                                       │
│ [                  ]                                       │
├ Default Location ──────────────────────────────────────────┤
│ Used as the default location for new workspaces and        │
│ collections                                                │
│ [ Click to browse for default location        ] Browse      │
└────────────────────────────────────────────────────────────┘
```

No help tooltip / InfoTip anywhere in General. No empty state (always shown).

---

## 5. Tab 2 — Themes

`components/Preferences/Themes/index.js`; styles `Themes/StyledWrapper.js`.

Heading: `.section-header` **`Appearance`** (`:81`).

### 5.1 Mode selector

Three buttons (`:51-55`, `:84-102`); each icon `size={16}` `strokeWidth={1.5}`,
class `theme-mode-option`, active class `selected`.

| # | id / key | Label | Icon |
| --- | --- | --- | --- |
| 1 | `light` | `Light` | `IconBrightnessUp` |
| 2 | `dark` | `Dark` | `IconMoon` |
| 3 | `system` | `System` | `IconDeviceDesktop` |

### 5.2 Variant cards

`renderThemeVariants` (`:61-75`) renders a section label + a row of
`theme-variant-card`s. Label text and shown set depends on mode:

| Mode | Section label(s) | Grid contents | Source |
| --- | --- | --- | --- |
| `light` | `Light Theme` | `getLightThemes()` | `:106-110` |
| `dark` | `Dark Theme` | `getDarkThemes()` | `:112-116` |
| `system` | `Light Theme`, divider, `Dark Theme` | both lists | `:118-124` |

Each card (`:27-36`) = preview swatch + `theme-variant-name`. The preview
(`:8-25`) paints `background.base` as card background, a `sidebar.bg` column,
and three `rgba(brand, 0.5)` bars (widths 100 %, 60 %, 70 %); border
`1px solid rgba(brand, 0.5)`. Theme catalogue and token values are in
`design-system.md` §9 — do not duplicate palette data here.

Selected state = `.selected` class (`:31`); the stylesheet selector is not in the
shared token table, so its exact chrome is **VERIFIED only as the class name**
(`Themes/index.js:31`) — see `Themes/StyledWrapper.js` when building. **UNVERIFIED**
exact selected-card outline: the doc does not quote a token for it.

State: all three modes always clickable; selecting applies immediately
(`setStoredTheme`, `:57-59`). No save button, no empty state.

---

## 6. Tab 3 — Display

`components/Preferences/Display/index.js`. Heading: `.section-header` **`Display`** (`:8`).
Two blocks in a column with `gap-10` (`:9`): **Font** (width `fit-content`) then
**Zoom** (full width).

> The brief's "Font" tab maps here.

### 6.1 Code Editor Font (`Display/Font/index.js`)

| # | Label (verbatim) | Control | Default | Width | Source |
| --- | --- | --- | --- | --- | --- |
| 1 | `Code Editor Font` | free-text input, class `textbox`, `mt-2` | `'default'` (`font.codeFont`) | 4/5 of row | `:14,66-76` |
| 2 | `Font Size` | `type=number`, `inputMode=numeric`, class `textbox` | `'13'` (`font.codeFontSize`) | 1/5 of row | `:15,79-88` |

Behaviour: both values clamp via `Math.max(1, Math.min(value, 32))` (`:21-25`)
— a numeric **font size** range 1–32 px, not a percentage. Auto-saves after
500 ms via `savePreferences` (`:44-60`); toast on failure `Failed to save preferences`
(`:37`). Note `Font` is rendered with a `close` prop from `Display` that it never
reads. No placeholder, no help text, no validation error message.

### 6.2 Interface Zoom (`Display/Zoom/index.js`)

| # | Label (verbatim) | Control | Default | Source |
| --- | --- | --- | --- | --- |
| 3 | `Interface Zoom` | custom select (div + caret), options below | `100%` (`display.zoomPercentage`, default `100`) | `:26,36,91-113` |

Options, verbatim (`:12-24`): `50%`, `60%`, `70%`, `80%`, `90%`, `100%`,
`110%`, `120%`, `130%`, `140%`, `150%`.

Select chrome: `height: 35.89px` (literal, **quoted exactly as in source**),
`padding: 0 0.5rem`, radius 4px, border `1px solid input.border`, bg
`input.background` (**this token does not exist — see Open Questions**),
font-size `0.875rem`, `line-height:1.5`, transition `all 0.15s ease`
(`Display/Zoom/StyledWrapper.js:17-36`). Caret `IconChevronDown` 14 px coloured
`input.border` (`:46-51`). Menu: width `80px`, `margin-top: .25rem`, radius 4px,
shadow `0 4px 6px -1px rgba(0,0,0,.1), 0 2px 4px -1px rgba(0,0,0,.06)`,
`max-height:200px`, scrollbar hidden; option `padding:.5rem .75rem`, font 14 px,
hover background `input.border`; selected option background
`${input.focusBorder}22` (8-digit hex alpha); check icon `IconCheck` 12 px in
`textLink` (`:53-100`).

Reset button: `Button size="sm" color="secondary" variant="ghost" icon={<IconReload/>}`
rendered only when value ≠ 100 (`:114-122`). No label/tooltip — icon only.

### 6.3 Layout

```text
DISPLAY
Code Editor Font                              Font Size
[ default                                  ]  [ 13   ]
Interface Zoom
[ 100% ▾ ]  ⟳ (appears only when not 100%)
```

### 6.4 Theme radio (orphan)

`Display/Theme/index.js` renders three radios `Light` / `Dark` / `System`
(`:27-73`) but is **not imported** by `Display/index.js` (compare `:2-3`). The
Theme choice moved to the Themes tab. Do **not** build the radio in the port;
it is documented only so nobody wonders what the dead file is.

---

## 7. Tab 4 — Proxy

`components/Preferences/ProxySettings/index.js`; styles `ProxySettings/StyledWrapper.js`.
Heading: `.section-header` **`Proxy Settings`** (`:136`).

Label column: `.settings-label { width: 100px }` (`ProxySettings/StyledWrapper.js:9-11`).
Local `.textbox` override: radius **3px** (`:19`), `padding: 0.15rem 0.45rem` (`:15`).

### 7.1 Mode (radio group)

| # | Label | value | Sets | Source |
| --- | --- | --- | --- | --- |
| 1 | `Off` | `off` | `disabled = true` | `:143-156` |
| 2 | `On` | `manual` | `disabled = false`, `source = 'manual'` | `:157-171` |
| 3 | `System Proxy` | `inherit` | `disabled = false`, `source = 'inherit'` | `:172-186` |
| 4 | `PAC` | `pac` | `disabled = false`, `source = 'pac'` | `:187-201` |

Default mode: derived from prefs — `off` if `proxy.disabled`, else `pac` /
`inherit` / `manual` by `proxy.source`; `source` default `'manual'`
(`:113-118`, `:57`). The radio group's row label reads `Mode` (`:139-141`).

### 7.2 `manual` branch

| # | Label | Control | Options | Default | Source |
| --- | --- | --- | --- | --- | --- |
| 5 | `Protocol` | radios | `HTTP`, `HTTPS`, `SOCKS4`, `SOCKS5` | `http` | `:211-261`, `:62` |
| 6 | `Hostname` | text | — | `''` | `:262-281` |
| 7 | `Port` | `type=number` | 1–65535 | `0` (falls back to `0`, renders `0`) | `:282-301`, `:64`, `:39-44` |
| 8 | `Auth` | **inverted** checkbox (`disabled: !checked`) | — | auth enabled (`auth.disabled` false) | `:302-316`, `:66` |
| 9 | `Username` | text | — | `''` | `:319-337` |
| 10 | `Password` | `type=password`/`text` + eye toggle; width `13.2rem`, height `2.25rem`, inner `10.5rem` | `IconEye`/`IconEyeOff` 18 px | `''` | `:338-366` |
| 11 | `Proxy Bypass` | text | — | `''` | `:368-387` |

Validation (schema `:18-52`): PAC URL error `Specify a valid PAC URL` (http/https/file
only, max 2048); port `Specify port between 1 and 65535`; field max lengths 1024.
Field errors are rendered inline `ml-3 text-red-500` (`:279,299,336,365,386`).

### 7.3 `inherit` branch — System Proxy panel

`ProxySettings/SystemProxy/index.js`. Rendered below the Mode row (`:204-208`).

| Element | Verbatim text | Source |
| --- | --- | --- |
| Title (`h2.text-xs`) | `System Proxy` + spinner `IconLoader2` 16 px while fetching | `:42-44` |
| Description | `Below values are sourced from your system proxy settings.` | `:45-47` |
| Error box | `Error loading system proxy settings: {error}` (danger tint bg/border, `width: fit-content`) | `:51-57`, `StyledWrapper.js:93-101` |
| Source | `Proxy source:` + value | `:60-68` |
| Info | `These values cannot be directly updated in Bruno. Please refer to your OS documentation to update these.` | `:70-72` |
| Values (`100px` label) | `http_proxy`, `https_proxy`, `no_proxy`, `pac_url` — each `-` when empty, value in `colors.text.purple` at `opacity .8` | `:74-98`, `StyledWrapper.js:115-118` |
| Action link | `Refresh` with `IconRefresh` 14 px; toast `PAC cache refreshed` / `Failed to refresh PAC cache` | `:99-107`; `ProxySettings/index.js:106-110` |
| Timestamp | `Last refreshed: {formatProxyTimestamp(lastRefreshedAt)}` | `:108-116` |

### 7.4 `pac` branch

| Element | Detail | Source |
| --- | --- | --- |
| Row label | `PAC` (100 px) | `:394` |
| Segmented toggle | `URL` / `File`; track radius `border.radius.base` (6px), `border 1px input.border`, `margin-right:12px`; button `height:34px`, `padding:.1rem .6rem`, `font.size.sm`, weight 500, idle `colors.text.muted`; active bg `button.secondary.bg`, text `button.secondary.color`; transition `background .12s, color .12s` | `:395-416`, `StyledWrapper.js:29-58` |
| URL input | `placeholder="https://example.com/proxy.pac"`, width `265px` | `:417-430`, `StyledWrapper.js:60-62` |
| File button | shows `Select File` or `decodeURIComponent(basename)` | `:431-450` |
| Hint | `Enter the URL to your PAC file` (url mode) / `Supports .pac files for automatic proxy configuration` (file mode); `font.size.xs`, `colors.text.muted`, `margin-top:4px`, `padding-left:100px` | `:455-459`, `StyledWrapper.js:73-78` |
| Refetch link | `Refetch` + `IconRefresh` 14 px, only when a source exists; toast `PAC cache refreshed` | `:460-468` |
| Picker failure | toast `Failed to open file picker` | `:443` |

PAC-mode is not auto-saved until a source exists (`:123-132`). Initial input mode
is `file` when `pac.source` starts with `file://` (`:119-121`).

### 7.5 Layout

```text
PROXY SETTINGS
Mode            ( ) Off   (•) On   ( ) System Proxy   ( ) PAC
                ── On selected ──
Protocol        (•) HTTP  ( ) HTTPS  ( ) SOCKS4  ( ) SOCKS5
Hostname        [______________________]
Port            [______]
Auth            [x]
Username        [______________________]
Password        [**********]  👁
Proxy Bypass    [______________________]
```

No empty state. No help tooltip. Failure toast `Failed to save preferences` (`:89`).

---

## 8. Tab 5 — Client Certificates

`components/Preferences/ClientCertSettings/index.js`; styles `…/StyledWrapper.js`.
(`ClientCertSettings` is the Preferences-level certificate store; it is **not**
`SecuritySettings`.)

Header block: `h1` **`Client Certificates`** (`font-medium`, `0.9375rem` = 15 px)
+ muted line **`Add client certificates to be used for specific domains.`**
(`font.size.xs`) (`:231-232`).

### 8.1 List (`components/ListGroup`)

| Part | Value | Source |
| --- | --- | --- |
| Container | `ListGroup` with `maxWidth` default `800px`, `margin-top: 20px` (`mt-5`) | `ClientCertSettings/index.js:234-235`; `components/ListGroup/index.js:49,62` |
| Empty state icon | `IconCertificate` 24 px `strokeWidth={1.2}` | `:238-242` |
| Empty state title | `No client certificates` | `:240` |
| Empty state text | `Certificates added here are sent automatically with requests to their matching domains.` | `:241` |
| Add button (empty state & footer) | `Add Certificate` + `IconPlus` 15 px, `Button size="sm"` | `:243-248`; `ListGroup/index.js:54-59,81-86` |
| Footer extra | `Save` button (`Button size="sm"`) | `:262-265` |
| Row leading | `IconCertificate` 20 px in `colors.text.yellow` | `:252`, `StyledWrapper.js:17-19` |
| Row actions | `ToggleSwitch size="2xs"` (title `Disable certificate` / `Enable certificate`) + `ActionIcon label="Remove certificate"` (`IconTrash` 16 px, hover `colors.text.danger`) | `:255-268` |
| Disabled row | `ListGroup.Item disabled` dims leading + body only, controls stay live | `ListGroup/index.js:14,18` |

Row body fields (`CertField`, label column `80px`, gap `0.75rem`, `0.8125rem` —
`StyledWrapper.js:21-37`):

| Field label | Source |
| --- | --- |
| `Host` | `:271` |
| `Cert File` (type `cert`) | `:276` |
| `Key File` (type `cert`) | `:277` |
| `PFX File` (type `pfx`) | `:273` |
| `Passphrase` — masked `••••••••` (8 U+2022) until revealed; toggle `ActionIcon` `label` `Show passphrase`/`Hide passphrase` with `IconEye`/`IconEyeOff` 14 px; rendered only when a passphrase exists | `:280-299` |

### 8.2 "Add Client Certificate" modal

`Modal size="md" title="Add Client Certificate" confirmText="Add"
dataTestId="add-client-cert-modal"` (`:305-312`). Intro (verbatim):
**`The certificate and key files are stored as absolute paths and apply to every collection.`** (`:313-315`).

| # | Label (verbatim) | Control | Options / default | Source |
| --- | --- | --- | --- | --- |
| 1 | `Domain` | text, placeholder `example.org`, inline animated protocol placeholder (`https://`, `grpcs://`, `wss://`) at `width:60px`, input `padding-left:60px`, width `300px` | empty | `:318-345`, `StyledWrapper.js:12-14,95-141` |
| 2 | `Type` | segmented (`role=radiogroup`), button `Cert & Key` / `PFX` | `cert` | `:346-370`, `StyledWrapper.js:40-65` |
| 3 | `Cert file` (cert) / `PFX file` (pfx) | hidden file input + `Choose file` button (`Button size="xs" variant="outline"`, `IconUpload` 13 px); selected → chip `IconFile` + basename + remove `IconX` (`label="Remove file"`) | empty | `:28-67`, `:371-408` |
| 4 | `Key file` (cert only) | same as #3 | empty | `:384-394` |
| 5 | `Passphrase` | `type=password`/`text` + eye toggle; box `300px × 1.70rem` | empty | `:409-439` |

Validation messages: `Domain is required` (`:113`); `certFilePath is a required field`
(`:117`), `keyFilePath is a required field` (`:121`), `pfxFilePath is a required field`
(`:126`). Type switch clears the other fields (`:177-188`).

Protocol placeholder animation: 9 s loop, staggered 0/3/6 s (`StyledWrapper.js:116-141`).

Pending-file chip: label `Remove file`; max filename width `260px` truncate
(`ClientCertSettings/index.js:46-51`).

---

## 9. Tab 6 — Keybindings

`components/Preferences/Keybindings/index.js`; styles `Keybindings/StyledWrapper.js`.

### 9.1 Header

`.section-header` text **`Keybindings`** (`:766-767`) with a right-aligned action
group (`:769-785`):

| Control | Detail | Source |
| --- | --- | --- |
| Enable toggle | `ToggleSwitch size="2xs"`, `activeColor={theme.primary.solid}`, default on (`keybindingsEnabled !== false`) | `:770-775`, `:271` |
| Divider | `1px × 18px`, `input.border`, `opacity .9` | `StyledWrapper.js:32-37` |
| Reset all | text button `Reset Default`; disabled when no custom keybindings; success toast `All shortcuts have been reset to default` | `:777-784`, `:547-555` |

When disabled, the table is dimmed: `opacity: .45`, `pointer-events:none`,
`user-select:none` (`StyledWrapper.js:54-58`) and rows cannot be edited
(`:558`).

### 9.2 Table

Columns (thead, `padding 5px 10px`, bg `table.striped`, colour
`table.thead.color`): **`Command`**, **`Keybinding`** (`:793-796`).
Column widths 35 % / 45 % (`StyledWrapper.js:88-96`). Row height `30px`
(`:101`), hover background `background.surface0` (`:111-114`).

Section heading rows (`font-weight:700`, `padding 6px 10px`) come from
`KEY_BINDING_SECTIONS` in order (`providers/Hotkeys/keyMappings.js:3-88`):

| # | Section heading | Rows (name → mac / windows binding) | Source |
| --- | --- | --- | --- |
| 1 | `Tabs` | `Close Tab` ⌘W / Ctrl+W · `Close All Tabs` ⌘⇧W · `Save` ⌘S · `Save All Tabs` ⌘⇧S · `Reopen Last Closed Tab` ⌘⇧T · `Switch to Tab at Position` ⌘1 – ⌘8 *(read-only, rendered as range)* · `Switch to Last Tab` ⌘9 · `Switch to Previous Tab` ⇧⌘[ · `Switch to Next Tab` ⇧⌘] · `Move Tab Left` ⌘[ · `Move Tab Right` ⌘] | `keyMappings.js:5-26` |
| 2 | `Sidebar` | `Search Sidebar` ⌘F · `Copy Item` ⌘C · `Paste Item` ⌘V · `Clone Item` ⌘D · `Rename Item` ⌘R · `Collapse Sidebar` ⌘\ | `:29-37` |
| 3 | `Requests` | `Send Request` ⌘↵ · `Change Orientation` ⌘J · `Prettify` ⌥⇧F | `:40-45` |
| 4 | `Editor` | `Trigger Autocomplete` ⌥Space / Ctrl+Space | `:48-51` |
| 5 | `Collections & Environment` | `Import Collection` ⌘O · `Edit Environment` ⌘E · `New Request` ⌘N | `:54-59` |
| 6 | `Search` | `Global Search` ⌘K | `:62-65` |
| 7 | `View` | `Zoom In` ⌘= · `Zoom Out` ⌘- · `Reset Zoom` ⌘0 | `:68-73` |
| 8 | `Developer Tool` | `Open in Terminal` ⌘T | `:76-79` |
| 9 | `Others` | `Open Preferences` ⌘, · `Close Bruno` ⌘Q / Ctrl+Shift+Q | `:82-87` |

Windows column uses `Ctrl` for `command`; Linux is display-only and labels the
`command` token `Super` (`keyMappings.js:93-119`). Keycaps: `min-width:20px`,
`height:20px`, `padding:2px`, radius 3px, `1px solid input.border`, bg
`background.base`, `10px`/500 text (`StyledWrapper.js:236-250`).

### 9.3 Row states

| State | Trigger | Visual | Source |
| --- | --- | --- | --- |
| Idle | — | keycaps | `:761` |
| Hover + customisable | `isHovered && !dirty && !editing && !readOnly` | pencil `IconPencil` 14 px, `title="Customize keys"` | `:815,902-910` |
| Customised (dirty) | row value ≠ default | `IconReload` 14 px reset button, `title="Reset to default"` | `:816,889-900` |
| Read-only | binding `readOnly` (only `Switch to Tab at Position`) | lock `IconLock` 14 px, `title="Reserved shortcut"`, `tabIndex=-1`, `aria-disabled` | `:817,912-921,846-849` |
| Editing | row clicked | `outline: 1px solid status.warning.border`, radius 4px, full width, `padding 0 8px`, blinking caret 1 px `1s step-end infinite` | `StyledWrapper.js:200-207,282-295` |
| Editing + error | invalid combo held | `outline: 1px solid status.danger.border`, `IconAlertCircle` 14 px + tooltip | `:209-213,861-878` |
| Success flash | saved | row bg `status.success.background` for 800 ms + `IconCircleCheck` 14 px | `:252-255,262-266,598-605` |

Empty state text: `No key bindings available` (`:940`) — only if every section
filtered out (never in practice).

### 9.4 Editing rules and verbatim error messages

| Rule | Message (verbatim) | Source |
| --- | --- | --- |
| Empty | `Shortcut can’t be empty.` (U+2019 right single quote in `validateCombo`) / `Shortcut can't be empty.` (ASCII apostrophe in the Backspace handler — **inconsistent**, keep both) | `:422`, `:661` |
| Only modifiers | `Add a non-modifier key (e.g. Ctrl + K).` | `:428` |
| No allowed modifier | mac: `macOS shortcuts must include at least one modifier (command/alt/shift/ctrl).` · win: `Windows shortcuts must include at least one modifier (ctrl/alt/shift).` | `:436-438` |
| Two non-modifiers | `Only one non-modifier key allowed (e.g. Cmd + Shift + K).` | `:446` |
| OS reserved | `This shortcut is reserved by the OS.` | `:453` |
| Duplicate | `This shortcut is already in use.` | `:460` |

Reserved sets: mac — ⌘H, ⌥⌘H, ⌃⌘F, ⇧⌘Q, ⌥⌘D, ⌘M, ⌘Tab, ⌘Space, ⌃⌘Q, ⇧⌘3/4/5,
⌥⌘Esc, ⌘Z, ⇧⌘Z, ⌥⌘Z, ⌥⌘I, F11, F12 (`:161-184`); windows — Alt+Tab,
Alt+Shift+Tab, Alt+F4, Alt+Esc, Alt+Space, Ctrl+Alt+Del, Ctrl+Shift+Esc, F1, F11,
F12, Ctrl+Z, Ctrl+Y, Ctrl+Shift+Z, Ctrl+Shift+I (`:185-204`). Function keys
F1–F12 are allowed without a modifier (`:92-96`). Max 3 keys per combo (`:683`).
Lock keys ignored (`:225`).

Interactions: click row → record (`:831`); commit on first keyup of a valid combo
(`:727-730`); `Backspace`/`Delete` clears draft (`:656-664`); `Enter` is recorded
as a key, not a commit; `Blur` commits, or cancels when in error (`:852-858`);
clicking another row tries to commit the current one (`:559-570`).
Tooltip (error) uses `react-tooltip` `place="bottom-start"`, `opacity 1`,
`className="tooltip-mod tooltip-mod--error"` (danger text) (`:868-878`).

Container: `max-width: 80%`, `max-height: calc(100% - 30px)`, scrollbars hidden
(`StyledWrapper.js:3-16`).

---

## 10. Tab 7 — AI *(deferred)*

See §2 and `overview.md` "Out of scope". One line for the build: **do not
implement in v1**; show the tab only if the Go port ships an AI backend.

---

## 11. Tab 8 — Cache

`components/Preferences/Cache/index.js`; styles `Cache/StyledWrapper.js`.

Section title **`Cache`** — uppercase, `font.size.sm`, `letter-spacing .05em`,
`colors.text.muted`, `margin-bottom:.75rem` (`:65`, `StyledWrapper.js:6-12`).

Two cards; each card radius `border.radius.md` (8px), `border 1px border1`,
`margin-bottom:1rem` (`StyledWrapper.js:14-19`). Header `padding .75rem 1rem`,
bg `background.surface0`, bottom border `1px border1` (`:21-29`); body
`padding .875rem 1rem`, align-items flex-end (`:54-60`).

| # | Title | Badge | Toggle default | Description (verbatim) | Extra | Source |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `File cache` | `Beta` (`status.info` tint, `padding 1px 6px`, radius 4px, `font.size.xs`, weight 500) | **off** (`cache.file.enabled` false) | `Loads your workspace faster by caching opened collections. Bruno refreshes the cache when your collection changes. Clearing it won't affect your original files.` | Size line: `Cache size` + `<strong>{formatSize(fileCacheSize)}</strong>`, `—` when null. `ActionIcon label="Clear cache"` (`IconEraser` 16 px, hover danger), disabled when size is falsy. Toasts `File cache cleared` / `Failed to clear file cache` | `:67-100`, `StyledWrapper.js:42-52,74-84` |
| 2 | `SSL session cache` | — | **off** (`cache.sslSession.enabled` false) | `Reuses TLS sessions and connections across requests for faster handshakes. Disable to create a fresh connection for every request.` | `ActionIcon label="Clear cache"`; toasts `SSL session cache cleared` / `Failed to clear SSL session cache`. Disabling the toggle also clears the agent cache | `:102-130`, `:37-46,57-61` |

Toggle: `ToggleSwitch size="2xs" activeColor={theme.primary.solid}`
(`:73-79,107-113`). Failure toast: `Failed to update cache preferences` (`:26`).
Titles `font.size.md` (14px) weight 600 (`StyledWrapper.js:37-40`); descriptions
`font.size.base`, `colors.text.muted`, `line-height 1.5` (`:67-72`); size line
`colors.text.subtext2`, `<strong>` weight 600 `text` (`:74-84`).

---

## 12. Tab 9 — Support

`components/Preferences/Support/index.js`; strings resolved via `react-i18next`
(`useTranslation`, `:72`) from `i18n/translation/en.json`.

Container `max-width: 40rem`, gap `1.25rem` (`Support/StyledWrapper.js:3-10`).
Heading `.section-header` **`Support`** (`:76`).

| Group (key → English) | Card row: label → English · description → English · URL | i18n source |
| --- | --- | --- |
| `SUPPORT.HELP` → **Help** | `COMMON.DOCUMENTATION` → **Documentation** · `SUPPORT.DOCUMENTATION_DESCRIPTION` → **Guides, references and examples** · `https://docs.usebruno.com` | `en.json:25`, `:4`, `:27` |
| | `COMMON.REPORT_ISSUES` → **Report Issues** · `SUPPORT.REPORT_ISSUES_DESCRIPTION` → **File a bug or request a feature** · `https://github.com/usebruno/bruno/issues` | `en.json:5`, `:28` |
| `SUPPORT.COMMUNITY` → **Community** | `COMMON.GITHUB` → **GitHub** · `SUPPORT.GITHUB_DESCRIPTION` → **Source, releases and discussions** · `https://github.com/usebruno/bruno` | `en.json:26`, `:6`, `:30` |
| | `COMMON.DISCORD` → **Discord** · `SUPPORT.DISCORD_DESCRIPTION` → **Ask questions, talk to the team** · `https://discord.com/invite/KgcZUncpjq` | `en.json:7`, `:29` |
| | `COMMON.TWITTER` → **Twitter** · `SUPPORT.TWITTER_DESCRIPTION` → **Release notes and announcements** · `https://twitter.com/use_bruno` | `en.json:8`, `:31` |
| | `COMMON.LINKEDIN` → **LinkedIn** · `SUPPORT.LINKEDIN_DESCRIPTION` → **Company news and updates** · `https://in.linkedin.com/company/usebruno` | `en.json:9`, `:32` |

Row chrome: `gap .75rem`, `padding .75rem .875rem`, hover bg
`background.mantle`, divider between rows `1px border1`; icon tile `26×26`,
radius 8px, bg `background.surface0`, hover `surface1`; label `font.size.base`;
description `font.size.xs` `colors.text.muted`; trailing `IconExternalLink`;
`target="_blank" rel="noreferrer"` (`Support/index.js:84-107`,
`Support/StyledWrapper.js:24-99`). Icons: `IconBook`, `IconSpeakerphone`,
`IconBrandGithub`, `IconBrandDiscord`, `IconBrandTwitter`, `IconBrandLinkedin`
(all 16 px, `:1-9,95-96`). Group headings `font.size.sm` weight 500
`colors.text.muted` (`StyledWrapper.js:17-22`).

---

## 13. Tab 10 — Beta

`components/Preferences/Beta/index.js`; styles `Beta/StyledWrapper.js`.

Heading `.section-header` **`Beta Features`** (`:190`). Intro paragraph
(verbatim, `text-gray-500`): **`Beta features are experimental previews that may
change before full release. Try them and share feedback.`** (`:193-195`).

Data-driven from `BETA_FEATURES` (`:45-82`):

| # | Title | Description (verbatim) | Toggle? | Links | Source |
| --- | --- | --- | --- | --- | --- |
| 1 | `AI Assistant` | `Generate scripts, tests, and documentation directly from the request tab. Includes contextual chat, scripting autocomplete, and support for OpenAI, Anthropic, and OpenAI-compatible providers using your own API key.` | no | `Go to AI settings` → `ai` · `View docs` → `https://link.usebruno.com/docs/ai` | `:47-53` |
| 2 | `File cache` | `Speeds up how quickly your collections open by keeping a local cache on disk. Turn it on or clear it anytime from the Cache settings.` | no | `Go to Cache settings` → `cache` | `:55-60` |
| 3 | `Akamai EdgeGrid` | `Sign requests with the Akamai EdgeGrid authentication scheme. Select it from the Auth type dropdown on any request, folder, or collection.` | no | `View docs` → `…/docs/auth` | `:62-67` |
| 4 | `Mock Server` | `Run a local mock server using response examples defined in your collection. Serve mock API responses for frontend development without a real backend.` | **yes**, default **on** | `View docs` → `…/docs/mock-server` | `:69-75` |
| 5 | `gRPC Scripting` | `Write scripts for gRPC requests. Inspect messages, metadata, and status across the call lifecycle.` | no | `View docs` → `…/docs/grpc-scripting` | `:77-81` |

Action button `Go to …` uses `IconArrowRight` 14 px; docs link `View docs`
`IconExternalLink` 14 px; both `font.size.sm` in `textLink`, underline on hover;
gap `1.25rem`, `margin-top:.625rem` (`:219-243`, `StyledWrapper.js:34-52`).
Rows `padding 1rem 0` + bottom `1px border1`, last row no border (`:13-20`).
Mock-server toggle `ToggleSwitch size="xs"` (`:210-216`) — turning it off with
unsaved mock responses opens the "Unsaved changes" modal (`Beta/useMockServerBetaToggle/index.js:37-43`;
modal copy in `components/RequestTabs/RequestTab/ConfirmRequestClose`). Empty
state (only when array emptied): `No beta features are currently available`
(`:248-252`), italic. Auto-saves after 500 ms; failure toast `Failed to update
beta preferences` (`:143`).

---

## 14. Cross-tab interaction summary

| Interaction | Result | Source |
| --- | --- | --- |
| Click left tab | Switch panel, remember last tab per window in Redux | `Preferences/index.js:36-44` |
| Type in General / Proxy / Font / Beta | Debounced save after 500 ms | `General/index.js:133-152`, `ProxySettings/index.js:99-132`, `Display/Font/index.js:44-60`, `Beta/index.js:150-170` |
| Escape / close | Preferences is a tab — closed like any other tab (`Ctrl/Cmd+W`) | `keyMappings.js:7` |
| `Cmd/Ctrl+S` | Saves the focused tab's form where the form supports submit (General, Proxy) | `General/index.js:192`, `ProxySettings/index.js:137` |

Divergence: only **General** and **Proxy** are `<form>`s; Themes/Display/
Cache/Support/Beta have no submit path. In the Go port, give every settings tab
an explicit save or make autosave uniform.

---

## 15. Open Questions

1. **`input.background` token is missing.** `Display/Zoom/StyledWrapper.js:25,60`
   reads `theme.input.background`, but the theme defines only
   `input.bg` (`themes/light/light.js:196-204`). The zoom select therefore falls
   back to the browser default background. The port should use `input.bg`.
2. **Mode radio in Display is dead code** (`Display/Theme/index.js`) while the
   Themes tab implements mode with three buttons. Confirm with product which
   control is canonical — I chose to document Themes and explicitly exclude the
   radio.
3. **Apostrophe inconsistency** in the Keybindings empty-shortcut message
   (curly vs straight, `Keybindings/index.js:422` vs `:661`). Pick one.
4. **Reset Default labels**: row-level tooltip says `Reset to default`
   (`Keybindings/index.js:896`); header button says `Reset Default` (`:783`).
   Decide whether to normalise to "Reset to Default".
5. **Client-cert file-path wording drift**: preferences modal says *absolute
   paths* and "apply to every collection" (`ClientCertSettings/index.js:314`),
   collection modal says *paths relative to the collection*
   (`CollectionSettings/ClientCertSettings/index.js:333`).
   Verify which one is true before writing copy in the port.
6. **No About tab exists.** If the port wants version/build info, the only
   sources are the status bar (`components/StatusBar`) and the changelog tab
   (`components/ChangelogTab`). Not covered here.
7. **Zoom select height `35.89px`** is a literal from source; treat as legacy.
8. General has **no placeholders, hints or tooltips**; if the port adds help
   text it will be new copy, not a port of existing copy.
