# Screen — Collection Settings

All paths are relative to `bruno/packages/bruno-app/src/` unless prefixed with
`design-system.md`.

Status: VERIFIED against source. Anything not confirmable from the repo is marked
**UNVERIFIED** inline and restated in §11 Open Questions.

Scope: `components/CollectionSettings/**` (53 files), plus
`components/MigrateCollectionToYmlModal`, `components/ShareCollection`, the
collection-level `SecuritySettings` (`components/SecuritySettings/JsSandboxMode`)
and the shared `components/BulkEditor` used inside Headers.

Screenshot reference: `refs/vars_bruno.png` shows the closely related
folder-settings Vars tab in dark theme (tab strip, `Pre Request` /
`Post Response` table captions, `Save` button); the collection Vars tab uses the
same table chrome (`CollectionSettings/Vars/VarsTable/index.js:129-147`).

---

## 1. Layout

```text
┌ Collection Settings tab (whole request-pane area) ────────────────────┐
│ Overview  Headers⁽²⁾  Vars⁽²⁾  Auth•  Script•  Tests•  Presets•  Proxy•  │ ← tab strip, wraps
│ Client Certificates•  Protobuf•            [ Edit ]  [ ✨ AI Assist ]  │ ← right-side actions
│───────────────────────────────────────────────────────────────────────│
│                                                                        │
│   <active tab panel>                                                   │
│                                                                        │
└───────────────────────────────────────────────────────────────────────┘
```

| Region | Value | Source |
| --- | --- | --- |
| Root | `flex flex-col h-full relative px-4 py-4 overflow-hidden` — **16 px padding all round** | `CollectionSettings/index.js:114` |
| Tab bar row | `flex items-start justify-between gap-4` | `:115` |
| Tab strip | `flex flex-wrap items-center tabs` | `:116` |
| Tab item | `padding: 6px 0px`, `border-bottom: 2px solid transparent`, `margin-right: tabs.marginRight` (**`1.2rem`**), colour `colors.text.subtext0`; hover `tabs.active.color`; active `font-weight: tabs.active.fontWeight` (**400**), `tabs.active.color`, `border-bottom: 2px solid tabs.active.border`; no focus outline | `CollectionSettings/StyledWrapper.js:8-36` (values resolved in `design-system.md` §1/§6) |
| Content | `mt-4 flex-1 min-h-0 overflow-auto` | `:164` |
| Count superscript | `<sup class="ml-1 font-medium">` for Headers / Vars | `:122,126` |
| Active-state dot | `<StatusDot />` (12 px dot, see `components/status-dot.md`) | `:130,134,138,142,146,150,154` |
| Radio accent | `primary.solid` | `components/CollectionSettings/StyledWrapper.js:53-56` |
| Markdown body | `height:auto`, `overflow-y:visible` | `components/CollectionSettings/StyledWrapper.js:4-7` |

### 1.1 Tab strip (exact order) — `CollectionSettings/index.js:117-155`

| # | id | Label | Indicator condition | Source |
| --- | --- | --- | --- | --- |
| 1 | `overview` | `Overview` | — | `:117-119` |
| 2 | `headers` | `Headers` + count superscript when > 0 | enabled headers count | `:120-123`, `:43-46` |
| 3 | `vars` | `Vars` + count superscript when > 0 | enabled req + res vars | `:124-127`, `:48-54` |
| 4 | `auth` | `Auth` + dot when `mode !== 'none'` | `:128-131`, `:55-57` |
| 5 | `script` | `Script` + dot when any script body exists | `:132-135`, `:39` |
| 6 | `tests` | `Tests` + dot when tests exist | `:136-139`, `:40` |
| 7 | `presets` | `Presets` + dot when requestType ≠ default or requestUrl ≠ '' | `:140-143`, `:69-70` |
| 8 | `proxy` | `Proxy` + dot when `proxyConfig` non-empty **and** `hostname` set | `:144-147`, `:59-62` |
| 9 | `clientCert` | `Client Certificates` + dot when ≥ 1 cert | `:148-151`, `:63-65` |
| 10 | `protobuf` | `Protobuf` + dot when `protoFiles.length > 0` | `:152-155`, `:66-68` |

The selected tab is stored per collection in Redux
(`collection.settingsSelectedTab`, `:28-36`), so it survives tab switches (the
initial value is **UNVERIFIED** — it comes from the collections slice
`providers/ReduxStore/slices/collections`, not read here).

### 1.2 Right-side actions — `CollectionSettings/index.js:157-162`

Rendered only for `AI_TABS = ['overview', 'script', 'tests']` (`:23`):

| Tab | Control | Source |
| --- | --- | --- |
| `overview` (not editing docs) | `DocsAction` → button `Edit` (`IconEdit`) / `Preview` (`IconEye`), `Button size="sm" color="secondary" variant="ghost"`, `opacity .7` → `1` on hover | `:159`; `components/Documentation/DocsAction/index.js:12-22` |
| all three | `SettingsAiAssist` — **AI feature, out of v1 scope** (`overview.md`) | `:160` |

---

## 2. Tab — Overview

`CollectionSettings/Overview/index.js`; styles `Overview/StyledWrapper.js`,
`Overview/Info/StyledWrapper.js`.

### 2.1 Layout (2/5 info column + 3/5 docs column)

```text
┌───────────────────────────┬──────────────────────────────────────┐
│ 📦 Collection name (18px) │  📄 Documentation          [Edit]     │
│                           │                                      │
│ 📁 Location               │  <rich text / preview>               │
│    /path/to/collection    │                                      │
│ 🏷 Version  Not Set  change│                                      │
│ 🌐 Environments           │                                      │
│    3 collection environments                                      │
│    2 global environments  │                                      │
│ 🔌 Requests               │                                      │
│    12 requests in collection                                      │
│ 🔗 Share                  │                                      │
│    Share Collection       │                                      │
│ 📖 Documentation          │                                      │
│    Generate Docs          │                                      │
│ ⇄ Migration (bru only)    │                                      │
└───────────────────────────┴──────────────────────────────────────┘
```

Grid `grid-cols-5 gap-5 h-full`; left `col-span-2` scrolls, right `col-span-3`
(`components/CollectionSettings/Overview/index.js:10-24`). Collection name row: `IconBox` 20 px + name
`text-lg font-medium` truncated (`:12-17`).

### 2.2 Info rows — `components/CollectionSettings/Overview/Info/index.js:45-188`

Each row = 40 px icon box (`p-3`, radius 8px = `rounded-lg`) + `font-medium`
title + muted value (`:47-59`).

| # | Title | Value / control | Icon + tint | Source |
| --- | --- | --- | --- | --- |
| 1 | `Location` | `collection.pathname` (`text-muted break-all`) | `IconFolder`, `rgba(textLink,.08)` bg, `rgba(textLink,.09)` border, icon `textLink` | `:50-60`, `Info/StyledWrapper.js:18-26` |
| 2 | `Version` | value or italic `Not Set`; whole row clickable → opens `ChangeCollectionVersion` modal; trailing link `change` (`text-link`, underline on row hover); tooltip only when text overflows (`max-width:220px` wrapper; tooltip `max-width:320px`, break-all) | `IconTag`, `rgba(colors.text.yellow,.1)` bg/border, icon `colors.text.yellow` | `:62-94`, `:5-16`, `:55-62` |
| 3 | `Environments` | link 1: `{n} collection environment{s}` → opens tab type `environment-settings`; link 2: `{n} global environment{s}` → tab type `global-environment-settings`; both `text-link hover:underline` | `IconWorld`, `rgba(colors.text.green,.08)` / `.09`, icon green | `:96-136`, `Info/StyledWrapper.js:28-35` |
| 4 | `Requests` | loading → `{total-itemsLoading} out of {total} requests in the collection loaded`; else `{n} request{s} in collection`; if invalid files: ` · {n} invalid` appended | `IconApi`, `rgba(colors.text.purple,.08)` / `.09`, icon purple | `:138-156`, `Info/StyledWrapper.js:37-44` |
| 5 | `Share` | link `Share Collection` → opens `ShareCollection` modal | `IconShare`, textLink tint | `:158-169`, `Info/StyledWrapper.js:46-53` |
| 6 | `Documentation` | link `Generate Docs` → opens `GenerateDocumentation` modal | `IconBook`, `rgba(accents.primary,.08)` / `.09`, icon `accents.primary` | `:171-182`, `Info/StyledWrapper.js:64-71` |
| 7 | `Migration` | see §2.4 (only when `collection.format === 'bru'`) | `IconTransform` 20 px in heading | `components/CollectionSettings/Overview/Migration/index.js:26-66` |

### 2.3 "Following requests were not loaded" card

`Overview/RequestsNotLoaded/index.js`. Rendered only when some items are
`partial && !loading` or have an `error` (`:27-40`).

| Part | Verbatim | Source |
| --- | --- | --- |
| Title | `Following requests were not loaded` with `IconAlertTriangle` 16 px (`warning-icon`) | `:66-69` |
| Columns | `Pathname`, `Size` | `:73-79` |
| Row | relative pathname + `{size} {KB|MB|GB}` (2-dp rounding) | `:82-93`, `:33-36` |
| Click | focuses the existing tab, else opens the request | `:42-62` |
| Card | `w-full card my-2` | `:65` |

### 2.4 Migration card (bru → yml)

| Part | Verbatim | Source |
| --- | --- | --- |
| Heading | `Migration` with `IconTransform` 20 px, `text-lg font-medium` | `components/CollectionSettings/Overview/Migration/index.js:29-32` |
| Title | `Migrate to YML file format` | `:39` |
| Body | `This collection is stored in BRU format. Switch to YML.` + link `Learn More ↗` (U+2197) → `https://blog.usebruno.com/making-yaml-the-default-in-bruno-v3.1` | `:40-51` |
| Button | `Convert to YML` (`Button size="sm" color="primary"`), opens `MigrateCollectionToYmlModal` | `:52-60`, `:16-24` |

### 2.5 Documentation column

`CollectionSettings/Docs/index.js`.

| Part | Detail | Source |
| --- | --- | --- |
| Header | `IconFileText` 20 px + `Documentation` (`text-lg font-medium`) | `:53-57` |
| Preview mode | `DocsEditor` read-only rich text; double-click to edit | `:72-83`; `components/Documentation/DocsEditor/index.js:147-183` |
| Edit mode controls | `Edit`/`Preview` button lives in the tab-bar right slot (§1.2); inside Docs, `Cancel` (`Button color="secondary"`) and `Save` buttons | `components/CollectionSettings/Docs/index.js:59-68` |
| Edit-mode strip | `RichTextEditor.MenuBar` + a `ModeSwitch` (rich ⇄ markdown, `gfm` CodeMirror) | `DocsEditor/index.js:145-161` |
| Empty preview copy | long Markdown placeholder (`Welcome to your collection documentation! …`) shown only in preview when docs are empty | `components/CollectionSettings/Docs/index.js:79,91-113` |
| Narrow width | `StyledWrapper` forces `.markdown-body { height:auto; overflow-y:visible }` | `CollectionSettings/StyledWrapper.js:4-7` |

---

## 3. Tab — Headers

`CollectionSettings/Headers/index.js`; styles `Headers/StyledWrapper.js`.

Intro line (verbatim, `text-xs text-muted`, `mb-4`):
**`Add request headers that will be sent with every request in this collection.`**
(`:151-153`).

### 3.1 Table

| Column | Key | Width | Placeholder | Editor | Source |
| --- | --- | --- | --- | --- | --- |
| checkbox (enabled) | `enabled` | — | — | native checkbox (`accent-color: workspace.accent`) | `EditableTable` | 
| `Name` | `name` | 20 % | `Name` | `SingleLineEditor`, autocomplete from `know-your-http-well` header list, newlines stripped | `:84-102` |
| `Value` | `value` | flex | `Value` | `SingleLineEditor`, autocomplete `MimeTypes` | `:103-118` |
| `Description` | `description` | `70px` (3rd cell) | `Description` | `createDescriptionColumn` | `:119`, `components/CollectionSettings/StyledWrapper.js:27-29` |
| trailing | — | — | — | delete row | `EditableTable` |

Table chrome: `font-weight:500`, `table-layout:fixed`; `thead`/`td`
`1px solid table.border`; `thead` colour `table.thead.color`, `font.size.base`,
non-selectable; `td` padding `6px 10px`; first cell 30 %; transparent-border
text inputs (`:4-53`). `defaultRow = { name:'', value:'', description:'' }` (`:122-126`).

Validation (`getRowError`, `:61-75`): name → `Header name cannot contain spaces or newlines`;
value → `Header value cannot contain newlines`.

### 3.2 Bulk Edit

Toggle link (bottom-right, `text-link`): **`Bulk Edit`** (`:164-168`). In bulk
mode the same intro line is shown, then `components/BulkEditor`: a 200 px
`text/plain` `CodeEditor` and a right-aligned link **`Key/Value Edit`**
(`components/BulkEditor/index.js:76-92`). No placeholder text inside the bulk
editor (blank canvas).

### 3.3 Save

`Save` `Button size="sm"` at `mt-6` (`:168-172`). `Cmd/Ctrl+S` inside any editor
also saves (`onSave={handleSave}` on each editor).

Empty state: none — an empty table renders one blank "add" row from `EditableTable`.

---

## 4. Tab — Vars

`CollectionSettings/Vars/index.js`; table `Vars/VarsTable/index.js`.

Two stacked tables, each with a `text-xs` caption in `colors.text.subtext0`
(`components/CollectionSettings/Vars/StyledWrapper.js:4-6`):

| # | Caption (verbatim) | `varType` | Data path | Source |
| --- | --- | --- | --- | --- |
| 1 | `Pre Request` | `request` | `root.request.vars.req` | `components/CollectionSettings/Vars/index.js:25-26` |
| 2 | `Post Response` | `response` | `root.request.vars.res` | `components/CollectionSettings/Vars/index.js:29-30` |
| 3 | — | — | `Save` `Button size="sm"` at `mt-6` | `components/CollectionSettings/Vars/index.js:32-36` |

### 4.1 Columns

| Column | Placeholder (request / response) | Notes | Source |
| --- | --- | --- | --- |
| checkbox (enabled) | — | `EditableTable` | table |
| `Name` | `Name` / `Name` | `width:25%`, sortable, reorderable (all env rows drag) | `VarsTable/index.js:68-75` |
| `Value` | `Value` / `Expr` | Response column header is `Expr` plus `InfoTip` reading **`You can write any valid JS Template Literal here`** | `VarsTable/index.js:76-118`, `:81` |
| `Description` | `Description` | `createDescriptionColumn({ nameFromRowIndex:true })` | `:60-65,119` |

Value cell = `VarValueCell` wrapping a `MultiLineEditor` (`{{var}}` autocomplete)
with `renderTypeSelector` → `DataTypeSelector` for **request vars only** (`:99-112`)
and a `SensitiveWarningPopover` when the row is used in a sensitive field
(`:113-115`).

Name error (`:51-58`): `Variable contains invalid characters. Must only contain alphanumeric characters, "-", "_", "."`
(`utils/common/variables.js:31-32`; regex `/^[\w-.]*$/`, `utils/common/regex.js:5`).

`defaultRow` = `{ name:'', value:'', description:'', ...(response → { local:false }) }`
(`:122-127`).

Chrome = `VarsTable/StyledWrapper.js` (identical to Headers, plus
`.btn-add-var { font-size: font.size.base }`). Table id `collection-vars`;
test ids `collection-vars-req` / `collection-vars-res` (`:131-146`).

Empty state: none; the trailing blank row is the "add" affordance. Reorder via
row drag when sorting is `Default order`.

---

## 5. Tab — Auth

`CollectionSettings/Auth/index.js`.

Intro (verbatim, `text-xs text-muted mb-4`):
**`Configures authentication for the entire collection. This applies to all requests using the Inherit option in the Auth tab.`**
with `Inherit` and `Auth` in `font-medium` (`:62-65`).

Selector: `AuthMode` (`Auth/AuthMode/index.js`) — bordered pill, `MenuDropdown`
placement `bottom-start`, `appendTo` body, `hideOnReferenceClip`, tick mark
(`:88-104`).

### 5.1 Mode menu — `components/CollectionSettings/Auth/AuthMode/index.js:24-86`

| # | id | Menu label | Trigger label (`humanizeRequestAuthMode`) |
| --- | --- | --- | --- |
| 1 | `awsv4` | `AWS Sig v4` | `AWS Sig V4` |
| 2 | `basic` | `Basic Auth` | `Basic Auth` |
| 3 | `wsse` | `WSSE Auth` | `WSSE Auth` |
| 4 | `bearer` | `Bearer Token` | `Bearer Token` |
| 5 | `digest` | `Digest Auth` | `Digest Auth` |
| 6 | `ntlm` | `NTLM Auth` | `NTLM` |
| 7 | `oauth1` | `OAuth 1.0` | `OAuth 1.0` |
| 8 | `oauth2` | `OAuth 2.0` | `OAuth 2.0` |
| 9 | `apikey` | `API Key` | `API Key` |
| 10 | `akamai-edgegrid` | `Akamai EdgeGrid` + `Beta` badge (`StatusBadge status="info" size="xs"`), `ariaLabel` `Akamai EdgeGrid (Beta)` | `Akamai EdgeGrid` |
| 11 | `none` | `No Auth` | `No Auth` |

**Collection auth has no `Inherit` option** (unlike request/folder auth,
`screens/request-auth.md` §2). Order difference from the request menu is real:
WSSE is 3rd here, 8th on requests.

### 5.2 Per-mode fields

All fields are `SingleLineEditor isCompact`; secrets use `isSecret` + a
`SensitiveFieldWarning` button. Labels use `block mb-1`; rows separated by
`mb-3`.

| Mode | Fields (order) | Notes | Source |
| --- | --- | --- | --- |
| `none` | — | no panel | `components/CollectionSettings/Auth/index.js:25-58` (switch has no `none` case) |
| `basic` | `Username`, `Password`⛨ | | `components/CollectionSettings/Auth/BasicAuth/index.js:45-72` |
| `bearer` | `Token`⛨ | | `components/CollectionSettings/Auth/BearerAuth/index.js:32-46` |
| `digest` | `Username`, `Password`⛨ | | `components/CollectionSettings/Auth/DigestAuth/index.js:46-72` |
| `wsse` | `Username`, `Password`⛨ | | `components/CollectionSettings/Auth/WsseAuth/index.js:47-70` |
| `ntlm` | `Username`, `Password`⛨, `Domain` | | `components/CollectionSettings/Auth/NTLMAuth/index.js:64-95` |
| `apikey` | `Key`, `Value`⛨, `Add To` | `Add To` options `Header` / `Query Params`; missing placement initialised to `header`; trigger at `bottom-end` | `components/CollectionSettings/Auth/ApiKeyAuth/index.js:59-110`, `:46-57`, `:87-108` |
| `awsv4` | `Access Key ID`, `Secret Access Key`⛨, `Session Token`⛨, `Service`, `Region`, `Profile Name` | | `components/CollectionSettings/Auth/AwsV4Auth/index.js:133-213` |
| `oauth1` | `Consumer Key`, `Consumer Secret`⛨, `Token`, `Token Secret`⛨, section `Signature` (`IconShieldLock` 14 px), `Signature Method`, *(conditional)* `Private Key`, `Add Params To`, then collapsible `Callback URL`, `Verifier`, `Timestamp`, `Nonce`, `Version`, `Realm` | `Signature Method` options: `HMAC-SHA1`, `HMAC-SHA256`, `HMAC-SHA512`, `RSA-SHA1`, `RSA-SHA256`, `RSA-SHA512`, `PLAINTEXT` (default label when unset: `HMAC-SHA1`). `Add Params To` options `Header` / `Query Params` / `Body` (default `Header`) | `components/CollectionSettings/Auth/Oauth1/index.js:8-22` (wraps `components/RequestPane/Auth/OAuth1/index.js:16-22,126-444`) |
| `oauth2` | `GrantTypeSelector` + grant-specific body | Grant types in menu order: `Password Credentials`, `Authorization Code`, `Implicit`, `Client Credentials` | `components/CollectionSettings/Auth/OAuth2/index.js:42-51`; `components/RequestPane/Auth/OAuth2/GrantTypeSelector/index.js:80-83` |
| `akamai-edgegrid` | `Access Token`⛨, `Client Token`⛨, `Client Secret`⛨, then `Advanced Settings` (`IconAdjustmentsHorizontal` 16 px): `Base URL`, `Nonce`, `Timestamp`, `Headers to Sign`, `Max Body Size` | Tooltips (verbatim) in table below | `components/CollectionSettings/Auth/EdgeGridAuth/index.tsx:36-68,110-155` |

⛨ = secret (`isSecret` + `SensitiveFieldWarning`).

Field labels use a fixed `min-w-[140px]` in the OAuth1/OAuth2 panels (vs
`block mb-1` in the simple modes) — an inconsistency worth harmonising.

#### OAuth2 field labels by grant (`inputsConfig.js`)

| Grant | Fields (order) | Source |
| --- | --- | --- |
| Authorization Code | `Authorization URL`, `Access Token URL`, `Client ID`, `Client Secret`⛨, `Scope`, `State` | `components/RequestPane/Auth/OAuth2/AuthorizationCode/inputsConfig.js:4-25` |
| Password Credentials | `Access Token URL`, `Username`, `Password`⛨, `Client ID`, `Client Secret`⛨, `Scope` | `…/PasswordCredentials/inputsConfig.js:4-26` |
| Client Credentials | `Access Token URL`, `Client ID`, `Client Secret`⛨, `Scope` | `…/ClientCredentials/inputsConfig.js:4-17` |
| Implicit | `Authorization URL`, `Client ID`, `Scope`, `State` | `…/Implicit/inputsConfig.js:4-16` |

Authorization Code / Client Credentials / Implicit additionally render:
`Add Credentials to` (`Request Body` / `Basic Auth Header`), `Use PKCE`,
`Token Source` (`Access Token` / `ID Token`), `Token ID`, `Add token to`
(`Header` / `URL`), `Header Prefix`, `Query Param Key`, `Refresh Token URL`,
`Automatically fetch token if not found`, `Auto refresh token (with refresh URL)`
(`AuthorizationCode/index.js:232-422`, `ClientCredentials/index.js:112-292`,
`Implicit/index.js:117-231`). These are shared with the request Auth tab — see
`screens/request-auth.md`; do not fork the implementation.

#### EdgeGrid tooltips (verbatim) — `components/CollectionSettings/Auth/EdgeGridAuth/index.tsx:36-62`

| Field | Tooltip |
| --- | --- |
| `Base URL` | `Defaults to the request URL if not specified.` |
| `Nonce` | `A unique nonce is required per request. Defaults to an auto-generated UUID v4 if not provided.` |
| `Timestamp` | `UTC timestamp of when the request is signed (yyyyMMddTHH:mm:ss+0000). Defaults to current time if not provided.` |
| `Headers to Sign` | `Comma-separated list of headers to include in the signature.` |
| `Max Body Size` | `Maximum message body size to include in the signature, in bytes. Defaults to 131072.` |

### 5.3 Save

`Save` `Button size="sm"` at `mt-6` (`components/CollectionSettings/Auth/index.js:70-74`). Mode change applies
immediately to the draft; the button persists.

---

## 6. Tab — Script

`CollectionSettings/Script/index.js`.

Intro (verbatim): **`Write pre and post-request scripts that will run before and after any request in this collection is sent.`** (`:105-107`).

Sub-tabs (`components/Tabs`):
`Pre Request` / `Post Response` (`:111-123`). Each shows a `StatusDot` (`error`
variant when any request reports a script error, else `default`) when its script
is non-empty (`:113-122`, `:99-101`). Default sub-tab: `pre-request` when a
pre-request script exists, else `post-response` (`:29-34`). Editors:
`CodeEditor mode="javascript"`, `showHintsFor ['req','bru']` (pre) /
`['req','res','bru']` (post) (`:135-172`). Save at `mt-12` (`:168-172`).

No placeholder copy inside editors (blank CodeMirror).

---

## 7. Tab — Tests

`CollectionSettings/Tests/index.js`.

Intro (verbatim): **`These tests will run any time a request in this collection is sent.`** (`:41`).
Single `CodeEditor mode="javascript"`, `showHintsFor ['req','res','bru']` (`:43-57`);
`Save` at `mt-6` (`:60-64`). No placeholder, no empty state.

---

## 8. Tab — Presets

`CollectionSettings/Presets/index.js`; items `Presets/constants.jsx`.

| # | Label | Subtitle (verbatim) | Control | Options / default | Source |
| --- | --- | --- | --- | --- | --- |
| 1 | `Default Request Type` | `Selected by default for new requests.` | `SegmentedControl`, `size="sm"` | `HTTP` (`IconApi` 18 px), `GraphQL` (`IconBrandGraphql` 16 px), `gRPC` (`IconCode` 16 px), `WebSocket` (`IconPlugConnected` 16 px); default `DEFAULT_PRESET_REQUEST_TYPE` | `components/CollectionSettings/Presets/index.js:71-82`, `constants.jsx:5-30` |
| 2 | `Default Base URL` | `Pre-fills the URL field for new requests.` | text input, placeholder **`Request URL`** | empty | `:84-101` |
| 3 | `Default Environment` | `Selected when this collection is shared and first opened.` | `MenuDropdown` (`bottom-start`, `sameWidth`), trigger shows name or `None`, caret `IconCaretDown` 14 px | first item `None` then each collection environment by name | `:103-124`, `:47-54` |
| 4 | — | — | `Save` `Button size="sm"` (`type="button"`) | | `:126-131` |

Empty environment list → only `None` in the dropdown (`:47-54`).
`presets` default = `{ requestType: DEFAULT_PRESET_REQUEST_TYPE, requestUrl: '' }`
(`:16`).

---

## 9. Tab — Proxy

`CollectionSettings/ProxySettings/index.js`.

Intro (verbatim): **`Configure proxy settings for this collection.`** (`:200`).

| # | Label | Control | Options / default | Source |
| --- | --- | --- | --- | --- |
| 1 | `Config` + `InfoTip` | radios | `inherit` (default), `enabled`, `disabled` — rendered **lowercase** as `inherit` / `enabled` / `disabled` | `:202-250`, `:17-18,195-196` |
| 2 | `Protocol` *(only when `enabled`)* | radios | `HTTP`, `HTTPS`, `SOCKS4`, `SOCKS5`; default `http` | `:253-303`, `:20` |
| 3 | `Hostname` | `SingleLineEditor isCompact` | empty | `:304-319` |
| 4 | `Port` | `type=number` | empty | `:320-337` |
| 5 | `Auth` | checkbox (`disabled: !checked`) | auth on by default | `:338-348` |
| 6 | `Username` | `SingleLineEditor` | empty | `:350-364` |
| 7 | `Password`⛨ | `SingleLineEditor isSecret` + `SensitiveFieldWarning` | empty | `:366-383` |
| 8 | `Proxy Bypass` | text | empty | `:385-402` |
| 9 | — | `Save` `Button size="sm"` | | `:404-409` |

InfoTip body (`:206-213`), verbatim list:

- `inherit - inherit from global preferences`
- `enabled - use collection-specific proxy config`
- `disabled - disable proxy for this collection`

Validation toasts (`:38-84`): `Hostname must be less than 1024 characters`,
`Port must be a valid number`, `Port must be between 1 and 65535`,
`Username must be less than 1024 characters`, `Password must be less than 1024 characters`,
`Bypass proxy must be less than 1024 characters`.

Initial config: `{ inherit: true, config: { protocol:'http', hostname:'', port:'', auth:{username:'',password:''}, bypassProxy:'' } }` (`:17-29`).

---

## 10. Tab — Client Certificates

`CollectionSettings/ClientCertSettings/index.js` — same component shape as the
Preferences tab.

| Part | Verbatim | Source |
| --- | --- | --- |
| Header | `Client Certificates` (`font-medium`, 15 px) | `:245` |
| Sub-line | `Add client certificates to be used for specific domains.` | `:246` |
| Empty title | `No client certificates` | `:254` |
| Empty text | `Certificates added here are sent automatically with requests to their matching domains.` | `:255` |
| Add button | `Add Certificate` + `IconPlus` 15 px | `:256-260` |
| Footer | `Save` button | `:262-266` |
| Row fields | `Host`, `Cert File`/`PFX File`, `Key File`, optional `Passphrase` (masked `••••••••`) | `:290-318` |
| Row actions | enable toggle (`Enable certificate` / `Disable certificate`), `Remove certificate` | `:274-285` |
| Modal title | `Add Client Certificate`, confirm `Add` | `:326-330` |
| Modal intro | **`The certificate and key files are stored as paths relative to the collection.`** | `:332-334` |
| Modal fields | `Domain` (placeholder `example.org`), `Type` (`Cert & Key` / `PFX`), `Cert file`, `Key file`, `PFX file`, `Passphrase` | `:337-438` |

⚠ Conflict with the Preferences copy (`components/CollectionSettings/ClientCertSettings/index.js:314` says
"absolute paths… every collection"). See Open Questions §11.5.

---

## 11. Tab — Protobuf

`CollectionSettings/Protobuf/index.js`.

### 11.1 Proto Files section (`data-testid="protobuf-proto-files-section"`)

| Part | Verbatim | Source |
| --- | --- | --- |
| Heading label | `Proto Files ({n})` + `IconAlertCircle` 16 px tooltip anchor | `:116-128` |
| Heading tooltip | `Keep your proto files within the collection folder or the corresponding git repository to ensure paths remain valid when sharing the collection.` | `:129-133` |
| Error banner | `Some proto files cannot be found. Use the replace option to update their locations.` (`IconAlertCircle` 14 px) | `:133-138` |
| Columns | `File`, `Path`, `Actions` | `:141-156` |
| Empty | `IconFile` 24 px + `No proto files added` | `:155-164` |
| Row | `IconFile` 16 px, basename, `IconAlertCircle` 12 px when missing; path text; actions: `Replace file` (`IconFileImport` 14 px, only when missing), `Remove file` (`IconTrash` 14 px) | `:165-208` |
| Add | `+ Add Proto File` (`text-link`, `py-3`) — opens file picker `accept=".proto" multiple` | `:213-215`, `:113-116` |

### 11.2 Import Paths section (`data-testid="protobuf-import-paths-section"`)

| Part | Verbatim | Source |
| --- | --- | --- |
| Heading label | `Import Paths ({n})` + tooltip anchor | `:223-236` |
| Heading tooltip | `Add directories that contain proto files to be imported. These paths help resolve import statements in your proto files.` | `:239-243` |
| Error banner | `Some import paths cannot be found at their specified locations.` | `:241-245` |
| Columns | *(checkbox)*, `Directory`, `Path`, `Actions` | `:247-262` |
| Empty | `IconFolder` 24 px + `No import paths added` | `:264-273` |
| Row | checkbox (`Disable this import path` / `Enable this import path`), `IconFolder` 16 px, basename, path; `Replace directory` (`IconFileImport`), `Remove import path` (`IconTrash`) | `:274-326` |
| Add | `+ Add Import Path` | `:331-333` |

`Save` at `mt-6` (`:337-341`).

---

## 12. Modal — MigrateCollectionToYmlModal

`components/MigrateCollectionToYmlModal/index.js`; styles `…/StyledWrapper.js`.
Hosted at **app level** (AppProvider) because the collection is removed from the
store mid-migration (`:102-104`). Two steps, both `Modal size="md"`.

### 12.1 Confirm step (`:397-475`)

| Part | Verbatim | Source |
| --- | --- | --- |
| Title | `Migrate to YML format` | `:401` |
| Body | `This will convert all files in <CollectionName> from .bru format to .yml format.` | `:414-416` |
| "What will happen:" | `What will happen:` heading + 4 bullets (below) | `:441-447` |
| Bullet 1 | `All .bru request files will be converted to .yml` | `:443` |
| Bullet 2 | `Environment files will be converted to YML format` | `:444` |
| Bullet 3 | `bruno.json will be replaced with opencollection.yml` | `:445` |
| Bullet 4 | `Open tabs will be closed and the collection will be reloaded` | `:446` |
| Loading note | `Waiting for the collection to finish loading before migration can start…` | `:448-450` |
| Backup block title | `Backup` (uppercase, `font.size.sm`, weight 500, `colors.text.muted`, `letter-spacing .04em`) | `:454`, `components/CollectionSettings/StyledWrapper.js:19-25` |
| Backup help | `Export this collection as a ZIP archive before migrating, in case you want to restore it later.` | `:456-458` |
| Backup button | `Export Collection` (idle) / `Exporting…`; `Button size="sm" color="secondary" variant="outline"` | `:460-469` |
| Backup success toast | `Collection backup exported` | `:166` |
| Backup failure toast | `Failed to export backup: {message}` | `:169` |
| Confirm | `Migrate` (`primary`); disabled while exporting or until `mountStatus === 'mounted'` | `:403-405,391` |
| Cancel | `Cancel` (Modal default) → closes; confirm is disabled while collection not mounted | `:407` |

Backup block chrome: `1px solid border.border2`, radius `border.radius.base`,
bg `background.mantle`, `padding 12px 14px` (`components/CollectionSettings/StyledWrapper.js:4-9`).

### 12.2 Unsaved-changes step (`:287-389`)

Shown instead of the confirm step when any draft exists (collection, folder,
request, env). `Modal size="md" title="Unsaved changes"`, `hideFooter`,
`disableEscapeKey`, `disableCloseOnOutsideClick`, `closeModalFadeTimeout={150}`
(`:291-300`).

| Part | Verbatim | Source |
| --- | --- | --- |
| Heading | `IconAlertTriangle` 32 px (`status.warning.text`) + `Hold on..` (two dots) | `:301-304` |
| Summary | `You have unsaved changes in {n} item(s). Save or discard them before migrating.` | `:305-308` |
| List (max 5) | `• Collection: {name}` / `• Folder: {name}` / `• Environment: {name}` / `• App: {name}` / `• Request: {filename || name}` | `:272-285,310-316` |
| Overflow | `...{n} additional item(s) not shown` | `:318-323` |
| Transient block | `Transient requests need to be saved individually before migrating.` + one row per transient with `Save` button (`IconDeviceFloppy` 14 px) | `:325-353` |
| Left action | `Discard All` (`color="danger"`) → deletes all drafts then migrates | `:355-365,175-195` |
| Right actions | `Back` (`secondary`, ghost) → returns to confirm | `:366-375,150-153` |
| Primary | `Saving…` while resolving; otherwise `Save All and Migrate` when `>1` item else `Save and Migrate`; disabled while resolving or when transient requests exist (tooltip `Save transient requests individually first`) | `:376-384` |
| Toasts | `Cannot save environment "{name}": invalid variable name(s) — …`, `Cannot save environment "{name}": {message}`, `Failed to save environment "{name}"`, `Failed to save changes` | `:236-259` |

Item list style: transient rows bg `background.surface0`, `1px solid border.border0`,
radius 4px (`components/MigrateCollectionToYmlModal/StyledWrapper.js:68-72`).

### 12.3 Progress state (`isMigrating`)

| Part | Verbatim | Source |
| --- | --- | --- |
| Confirm text | `Cancel` (danger) while migrating; `Cancelling…` while cancelling | `:403-404` |
| Progress bar | `role="progressbar"` 0–100, track 6 px radius 3px `background.mantle`, fill `accents.primary`, transition `width .15s ease` | `:417-437`, `components/MigrateCollectionToYmlModal/StyledWrapper.js:39-57` |
| Phase labels | `Converting files`, `Writing yml files`, `Removing bru files` rendered as `{Phase}: {current}/{total}` | `:37-41,393-395` |
| Fallback label | `Preparing…` | `:395` |
| Cancelling label | `Cancelling and restoring the collection…` | `:435` |

During migration: `hideCancel`, `hideClose`, no outside click, no Escape
(`:408-411`).

---

## 13. Modal — ShareCollection

`components/ShareCollection/index.js`; styles `…/StyledWrapper.js`.
`Modal size="lg" title="Share Collection" hideFooter` — the action button lives
inside the body (`:117-118,212-220`).

### 13.1 Intro

`Bruno uses OpenCollection - An open format for API collections` — `OpenCollection`
is a `textLink` link to `https://opencollection.com`, `text-sm`, `mb-4` (`:119-130`).

### 13.2 Format cards

Group 1 heading **`Bruno Format`** (`:133`), 2-column grid `gap 1rem` (`:134`).

| Card | Title | Description | Feature bullets | Footer | Source |
| --- | --- | --- | --- | --- | --- |
| ZIP (default selected) | `Bruno Collection (ZIP)` + badge `Recommended` | `OpenCollection format organized as folders and files` | `Folder structure with individual .yml files`; `Collaborate with your team via pull requests`; `Extract and open directly in Bruno` | `Best for: Team collaboration, version control, publishing` | `:136-160` |
| YAML | `Single File (YAML)` | `OpenCollection format bundled into one .yml file` | `Everything in a single YAML file`; `Paste in a gist or attach to an issue` | `Best for: Quick sharing as a single file` | `:163-183` |

Group 2 heading **`Other Format`** (`:186`), 2-column grid:

| Card | Name | Description | Source |
| --- | --- | --- | --- |
| Postman | `Postman` | `Export for Postman` (`IconFileExport` 28 px) | `:187-201` |

Card chrome: `2px solid border.border0`, radius 6px, `padding 1rem`,
`min-height:180px`, bg `background.base`, transition `border-color .15s ease`;
hover `border2`; selected `primary.solid` (`components/ShareCollection/StyledWrapper.js:53-71`).
`Recommended` badge: `padding .125rem .5rem`, `11px`, weight 600, radius 4px,
bg `colors.text.warning` (`#c67a10` L / `#f6ab79` D), text white
(`components/ShareCollection/StyledWrapper.js:83-90`) — note the badge is white-on-warning, which is
**low contrast in dark theme**. Title `15px/600`; description `13px`
`colors.text.subtext0`; bullet `13px`; footer italic `12px`
`colors.text.muted` (`components/ShareCollection/StyledWrapper.js:72-126`).

### 13.3 Postman warning

Shown only when Postman is selected **and** the collection contains gRPC or
WebSocket requests (`:203-210`):

> `Note: {gRPC, WebSocket} requests in this collection will not be exported`

Band uses literal `backgroundColor: rgba(251, 191, 36, 0.1)` and colour
`#f59e0b` (`:204-206`) — **hard-coded values, not theme tokens.**

### 13.4 Footer / actions

| Control | Verbatim | Enabled when | Source |
| --- | --- | --- | --- |
| Proceed | `Proceed` / `Exporting...` | not while collection loading or exporting | `:212-220`, `:103` |
| Disabled card state | `opacity-50 cursor-not-allowed` on all cards | collection loading/exporting | `:137,164,189` |

Toasts: `Collection exported successfully` (`:54,65`),
`Failed to export collection: {message}` (`:57,67`).
Postman selection mounts `ExportToPostman` instead of this modal (`:105-113`).

---

## 14. Collection-level `SecuritySettings` (JavaScript Sandbox)

The brief's "SecuritySettings" is **not** part of CollectionSettings. It is a
standalone dropdown mounted in the collection toolbar area —
`components/SecuritySettings/JsSandboxMode/index.js`.

| Part | Detail | Source |
| --- | --- | --- |
| Trigger | `Dropdown` (`placement="bottom-start"`) whose icon is a 14 px `IconShieldCheck` (safe) or `IconCode` (developer) in a `.sandbox-icon` wrapper; `ToolHint` text `Safe Mode` / `Developer Mode` | `:106-114`, `:118` |
| Dropdown header | `JavaScript Sandbox` | `:120` |
| Options | see table | `:10-26` |
| Persistence | `saveCollectionSecurityConfig(uid, { jsSandboxMode })`; failure toast `Failed to update sandbox mode` | `:51-68` |
| Default | `safe` | `:31,34` |
| Escape | closes the dropdown | `:45-49` |

| # | Key | Label | Badge | Description (verbatim) | Warning (verbatim) | Icon |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `safe` | `Safe Mode` | `Recommended` | `JavaScript code is executed in a secure sandbox and cannot access your filesystem or execute system commands.` | — | `IconShieldCheck` 24 px |
| 2 | `developer` | `Developer Mode` | — | `JavaScript code has access to the filesystem, can execute system commands and access sensitive information.` | `Use only if you trust the authors of the collection` | `IconCode` 24 px |

Toolbar badge colours (safe vs developer) are tokenised under
`app.collection.toolbar.sandboxMode` — see `tokens.json` (design-system §6.6
points there for the full group). Option rows carry a hidden radio input
(`name="sandbox-mode"`), `role="menuitemradio"`, `aria-checked` (`:85-97`).

---

## 15. Interaction summary

| Interaction | Result | Source |
| --- | --- | --- |
| Switch settings tab | Redux `updateSettingsSelectedTab`, per collection | `CollectionSettings/index.js:29-36` |
| `Save` in any tab | `saveCollectionSettings(collection.uid)` | e.g. `components/CollectionSettings/Headers/index.js:59`, `components/CollectionSettings/Vars/index.js:16`, `components/CollectionSettings/Auth/index.js:23` |
| `Cmd/Ctrl+S` in a table editor | Saves (each `SingleLineEditor` has `onSave`) | `components/CollectionSettings/Headers/index.js:95,111`; `keyMappings.js:9` |
| Header/Vars rows | Add via trailing blank row; delete via row trash | `components/EditableTable` |
| Vars rows | Drag to reorder when sort = default; column widths resizable & persisted | `VarsTable/index.js:47-49,136-145` |
| Overview version row | Opens `ChangeCollectionVersion` | `Info/index.js:62-94` |
| Overview links | Open environment settings tabs | `Info/index.js:104-133` |
| Share | Opens ShareCollection modal (no API key in current build — purely export) | `ShareCollection/index.js` |

Inheritance coupling: collection Auth feeds request/folder `Inherit`
(`components/CollectionSettings/Auth/index.js:62-65`); collection Vars/Headers/Script/Tests are applied to every
request (copy says so in each tab intro).

---

## 16. Open Questions

1. **`collection.settingsSelectedTab` default** is not set in this component —
   confirm which tab opens first (likely `overview`) by reading
   `providers/ReduxStore/slices/collections`.
2. **`none` mode has no editor panel** (`components/CollectionSettings/Auth/index.js:25-58`), but the intro
   paragraph still shows. Should the port hide the intro for `No Auth`?
3. **`Akamai EdgeGrid` is behind the Beta flag** but is always present in the
   menu (`components/CollectionSettings/Auth/AuthMode/index.js:71-80`). Confirm whether the port gates it.
4. **Collection auth lacks `Inherit`** by design (there is nothing above a
   collection), yet the intro copy references Inherit — that is about *requests*
   inheriting from the collection. Wording is correct but easy to misread.
5. **Client-cert path semantics conflict** (§10 vs Preferences). Resolve
   before porting both strings.
6. **ShareCollection is export-only.** There is no share URL / permission UI in
   this component despite the name; the sidebar/collection-menu share action
   may differ (not verified here).
7. **Postman warning colours** (`rgba(251,191,36,.1)`, `#f59e0b`) bypass the
   theme. The port should map them to `status.warning.*`.
8. **`Recommended` badge contrast** is white on `colors.text.warning` — verify
   against `design-system.md` §1.4 and consider a tokenised pair.
9. **`Hold on..` vs `Hold on...`** — the migration modal uses two dots
   (`MigrateCollectionToYmlModal/index.js:303`); the environment unsaved modal
   uses three (`Environments/ConfirmCloseEnvironment/index.js:28`).
10. **Right-side AI assist** is visible on Overview/Script/Tests; it is deferred
    from v1 (`overview.md`), so the port shows no right-side action except
    `Edit` on Overview.
