# Screen — Auth tab

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source, except where noted in Open Questions.

---

## 1. Layout

```text
┌ Auth ────────────────────────────────────────────────────────────────┐
│ [ No Auth ▾ ]                                     ← AuthMode selector, 12px font, bordered pill
│ Auth inherited from Collection:  [ Basic Auth ↗ ]  ← only when mode = inherit
│                                                                      │
│ Username                                                              │  ← label + 22px compact editor, max width 400px
│ ┌──────────────────────────────────────────┐                          │
│ │                                          │                          │
│ └──────────────────────────────────────────┘                          │
│ Password  🔒 ...                                                      │
│ ...                                                                   │
└──────────────────────────────────────────────────────────────────────┘
```

Parent: `RequestPane/Auth/index.js:40-52` — column flow, `gap-2`, `mb-4`
between selector block and fields; the scroll container is the tab itself
(`overflow-auto`, `:42`).

Selector chrome (`RequestPane/Auth/AuthMode/StyledWrapper.js`):
`font.size.sm` (12px), `padding: 0.2rem 0`, `border-radius: 3px`,
`1px solid input.border` (`#cccccc` L / `#444444` D), `input.bg`
(`#ffffff` L / `transparent` D); label text `primary.text`, horizontal padding
`0 0.5rem`; caret `IconCaretDown` 14px in `colors.text.muted` (`:11-24`,
`AuthMode/index.js:103-110`).

Compact field rows (`Auth/*/index.js`, e.g. `BasicAuth:52-65`):
`label.block.mb-1` then `.single-line-editor-wrapper` with
`padding: 0.15rem 0.4rem`, radius `3px`, `1px solid input.border`,
`input.bg`, `max-width: 400px` (OAuth styling, `OAuth2/StyledWrapper.js:7-13`);
`SingleLineEditor isCompact` → height `1.375rem` (22px),
CodeMirror font `font.size.base` (13px) (`components/SingleLineEditor/StyledWrapper.js:5,20`).

---

## 2. Auth mode menu

Items in displayed order (`RequestPane/Auth/AuthMode/index.js:31-91`) with the
label shown after selection (`humanizeRequestAuthMode`,
`utils/collections/index.js:1016-1067`):

| #   | id                | Menu label                                                               | Trigger label after select |
| --- | ----------------- | ------------------------------------------------------------------------ | -------------------------- |
| 1   | `awsv4`           | `AWS Sig v4`                                                             | `AWS Sig V4`               |
| 2   | `basic`           | `Basic Auth`                                                             | `Basic Auth`               |
| 3   | `bearer`          | `Bearer Token`                                                           | `Bearer Token`             |
| 4   | `digest`          | `Digest Auth`                                                            | `Digest Auth`              |
| 5   | `ntlm`            | `NTLM Auth`                                                              | `NTLM`                     |
| 6   | `oauth1`          | `OAuth 1.0`                                                              | `OAuth 1.0`                |
| 7   | `oauth2`          | `OAuth 2.0`                                                              | `OAuth 2.0`                |
| 8   | `wsse`            | `WSSE Auth`                                                              | `WSSE Auth`                |
| 9   | `apikey`          | `API Key`                                                                | `API Key`                  |
| 10  | `akamai-edgegrid` | `Akamai EdgeGrid` + `Beta` badge (`StatusBadge status="info" size="xs"`) | `Akamai EdgeGrid`          |
| 11  | `inherit`         | `Inherit`                                                                | `Inherit`                  |
| 12  | `none`            | `No Auth`                                                                | `No Auth`                  |

Note the case difference between the menu item `AWS Sig v4` and the trigger
`AWS Sig V4` (`AuthMode/index.js:33` vs `utils/collections/index.js:1023`).
Dropdown: placement `bottom-start`, `appendTo` `document.body`,
`hideOnReferenceClip`, tick mark enabled (`AuthMode/index.js:93-101`).
The Beta badge uses `status.info` — bg `rgba(52,106,178,.15)` L /
`rgba(139,194,249,.15)` D, text/border `#346ab2` L / `#8bc2f9` D
(design-system §1.4).

---

## 3. Per-mode fields

Every field is a `SingleLineEditor` (secret fields mask to `*` and get a
`SensitiveFieldWarning` button when the value looks like a plain secret).
`Enter` sends the request, `Cmd/Ctrl+S` saves (`SingleLineEditor/index.js:74-89`,
`keyMappings.js:9,42`).

### 3.1 No Auth — `none`

Body: plain text `No Auth` (`RequestPane/Auth/AuthFields/index.js:21-23`).

### 3.2 Inherit — `inherit`

- Source label (verbatim): `Auth inherited from <name>:` where `<name>` is
  wrapped in a highlighted span (`InheritedAuth/index.js:98-102`).
- Mode button after it: mode label + `IconExternalLink` 14px, `aria-label`
  `Open <Mode> auth in <name>`; clicking opens the folder/collection settings
  on the `auth` tab (`InheritedAuth/index.js:103-113,66-94`).
- Fields of the inherited mode are rendered **read-only** below, with
  `pb-4` (`InheritedAuth/index.js:131-147`).
- Unsupported inherited mode: message
  `<Mode> is not supported by <protocol>. Using no auth instead.` or
  `<Mode> is not supported. Using no auth instead.` (`InheritedAuth/index.js:26-33`).

Screenshot: `refs/auth_bruno.png` shows `Inherit` with
`Auth inherited from Collection:  No Auth`.

### 3.3 API Key — `apikey`

| Order | Label    | Input    | Notes                                                                                                                                             | Source                                             |
| ----- | -------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| 1     | `Key`    | text     | —                                                                                                                                                 | `ApiKeyAuth/index.js:70-83`                        |
| 2     | `Value`  | secret   | masked                                                                                                                                            | `:85-100`                                          |
| 3     | `Add To` | dropdown | options `Header` / `Query Param`; default `header` auto-set on mount. Trigger shows `Header` or `Query Params` (`humanizeRequestAPIKeyPlacement`) | `:102-125`, `utils/collections/index.js:1069-1082` |

No placeholders. Missing `placement` is initialised to `header`
(`ApiKeyAuth/index.js:53-67`).

### 3.4 Basic Auth — `basic`

| Label      | Input  | Notes  | Source                     |
| ---------- | ------ | ------ | -------------------------- |
| `Username` | text   | —      | `BasicAuth/index.js:52-65` |
| `Password` | secret | masked | `:67-79`                   |

### 3.5 Bearer Token — `bearer`

| Label   | Input  | Notes  | Source                      |
| ------- | ------ | ------ | --------------------------- |
| `Token` | secret | masked | `BearerAuth/index.js:39-54` |

### 3.6 Digest Auth — `digest`

| Label      | Input  | Notes  | Source                      |
| ---------- | ------ | ------ | --------------------------- |
| `Username` | text   | —      | `DigestAuth/index.js:52-65` |
| `Password` | secret | masked | `:67-79`                    |

### 3.7 NTLM — `ntlm`

| Label      | Input  | Notes  | Source                    |
| ---------- | ------ | ------ | ------------------------- |
| `Username` | text   | —      | `NTLMAuth/index.js:70-83` |
| `Password` | secret | masked | `:85-100`                 |
| `Domain`   | text   | —      | `:102-115`                |

### 3.8 AWS Sig V4 — `awsv4`

| Order | Label               | Input  | Notes  | Source                       |
| ----- | ------------------- | ------ | ------ | ---------------------------- |
| 1     | `Access Key ID`     | text   | —      | `AwsV4Auth/index.js:133-146` |
| 2     | `Secret Access Key` | secret | masked | `:148-164`                   |
| 3     | `Session Token`     | secret | masked | `:166-181`                   |
| 4     | `Service`           | text   | —      | `:183-196`                   |
| 5     | `Region`            | text   | —      | `:198-211`                   |
| 6     | `Profile Name`      | text   | —      | `:213-226`                   |

### 3.9 WSSE — `wsse`

| Label      | Input  | Notes  | Source                    |
| ---------- | ------ | ------ | ------------------------- |
| `Username` | text   | —      | `WsseAuth/index.js:53-66` |
| `Password` | secret | masked | `:68-80`                  |

### 3.10 Akamai EdgeGrid (Beta) — `akamai-edgegrid`

Field config at `EdgeGridAuth/index.tsx:44-70`; rendered by
`renderField` (`:110-146`).

| Group    | Label             | Input                                                      | Tooltip (verbatim)                                                                                                |
| -------- | ----------------- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Basic    | `Access Token`    | secret                                                     | —                                                                                                                 |
| Basic    | `Client Token`    | secret                                                     | —                                                                                                                 |
| Basic    | `Client Secret`   | secret                                                     | —                                                                                                                 |
| Advanced | `Base URL`        | text (pre-filled with the request URL when empty — `:114`) | `Defaults to the request URL if not specified.`                                                                   |
| Advanced | `Nonce`           | text                                                       | `A unique nonce is required per request. Defaults to an auto-generated UUID v4 if not provided.`                  |
| Advanced | `Timestamp`       | text                                                       | `UTC timestamp of when the request is signed (yyyyMMddTHH:mm:ss+0000). Defaults to current time if not provided.` |
| Advanced | `Headers to Sign` | text                                                       | `Comma-separated list of headers to include in the signature.`                                                    |
| Advanced | `Max Body Size`   | text (numeric; invalid → null)                             | `Maximum message body size to include in the signature, in bytes. Defaults to 131072.`                            |

Group header text `Advanced Settings` with `IconAdjustmentsHorizontal` 16px
(`EdgeGridAuth/index.tsx:149-155`). Tooltips are rendered as an
`IconInfoCircle` 16px after the label (`:116-122`). Labels: weight 500
(`EdgeGridAuth/StyledWrapper.js:9`), group header weight 600 (`:36`).

### 3.11 OAuth 1.0 — `oauth1`

Sections with icons (no collapse except Advanced). Source: `OAuth1/index.js`.

| Section                                                                    | Order | Label               | Control                  | Notes                                                                                                                                                         |
| -------------------------------------------------------------------------- | ----- | ------------------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Configuration` (`IconSettings` 14px)                                      | 1     | `Consumer Key`      | text                     | `:126-138`                                                                                                                                                    |
|                                                                            | 2     | `Consumer Secret`   | secret                   | hidden when Signature Method starts with `RSA-` (`:142`)                                                                                                      |
|                                                                            | 3     | `Token`             | text                     | `:164-176`                                                                                                                                                    |
|                                                                            | 4     | `Token Secret`      | secret                   | `:181-195`                                                                                                                                                    |
| `Signature` (`IconShieldLock`)                                             | 5     | `Signature Method`  | dropdown                 | `HMAC-SHA1` (default display), `HMAC-SHA256`, `HMAC-SHA512`, `RSA-SHA1`, `RSA-SHA256`, `RSA-SHA512`, `PLAINTEXT` (`:15-23`)                                   |
|                                                                            | 6     | `Private Key`       | multi-line secret editor | only when signature starts with `RSA-`; `Upload File` button (title `Select File`); file chip shows basename with clear `×` (title `Clear file`) (`:231-280`) |
|                                                                            | 7     | `Add Params To`     | dropdown                 | `Header` (default), `Query Params`, `Body` (`:25-29`)                                                                                                         |
|                                                                            | 8     | (hint)              | —                        | when `Body`: `Body placement requires a form-urlencoded body. Non-form payloads will be replaced with OAuth parameters.` (`:304-311`)                         |
|                                                                            | 9     | `Include Body Hash` | checkbox                 | `:313-335`                                                                                                                                                    |
| `Advanced` (`IconAdjustmentsHorizontal`, collapsible, chevron rotates 90°) | 10    | `Callback URL`      | text                     | `:359-372`                                                                                                                                                    |
|                                                                            | 11    | `Verifier`          | text                     | `:376-389`                                                                                                                                                    |
|                                                                            | 12    | `Timestamp`         | text                     | `:393-406`                                                                                                                                                    |
|                                                                            | 13    | `Nonce`             | text                     | `:410-423`                                                                                                                                                    |
|                                                                            | 14    | `Version`           | text                     | `:427-440`                                                                                                                                                    |
|                                                                            | 15    | `Realm`             | text                     | `:444-457`                                                                                                                                                    |

Labels are `min-w-[140px]`, `font.size.sm`, colour `colors.text.subtext1`
(`OAuth1/StyledWrapper.js:9-12`); section labels in `text`.
Uploading a file as a typed value is rejected with toast
`File references should be added using the "Upload File" button below`
(`OAuth1/index.js:62-68`).

### 3.12 OAuth 2.0 — `oauth2`

First control is the **Grant Type** selector (`OAuth2/GrantTypeSelector/index.js:58-90`),
with `Grant Type` section label + `IconKey`:

| id                   | Label                  | Default                                          |
| -------------------- | ---------------------- | ------------------------------------------------ |
| `password`           | `Password Credentials` |                                                  |
| `authorization_code` | `Authorization Code`   | ✔ auto-initialised when no grant type (`:34-60`) |
| `implicit`           | `Implicit`             |                                                  |
| `client_credentials` | `Client Credentials`   |                                                  |

Unknown grant type renders the body text `TBD` (`OAuth2/index.js:41`).

#### 3.12.1 Authorization Code (default)

Source `OAuth2/AuthorizationCode/index.js`; labels all `min-w-[140px]`.

| Section                                               | Label                                      | Control                                                                                                                             | Notes                                                                                                                                |
| ----------------------------------------------------- | ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| (top)                                                 | token viewer                               | collapsible readers `Access Token`, `Refresh Token`, `ID Token` (`Oauth2TokenViewer/index.js:143-145`)                              | copy tooltip `Copy token`; JWT payload decoded under `Decoded Payload`; expiry pill `Expires in …` / `Expired` (`:50-125`)           |
| `Configuration` (`IconSettings`)                      | `Callback URL`                             | text                                                                                                                                | placeholder `https://oauth.usebruno.com/callback` **only when** the "Use system browser" preference is on (`:157-174`)               |
|                                                       | (unlabelled)                               | checkbox                                                                                                                            | `Use system browser for OAuth` (`:175-195`)                                                                                          |
|                                                       | `Authorization URL`                        | text                                                                                                                                | `inputsConfig.js:3-6`                                                                                                                |
|                                                       | `Access Token URL`                         | text                                                                                                                                | `:7-10`                                                                                                                              |
|                                                       | `Client ID`                                | text                                                                                                                                | `:11-14`                                                                                                                             |
|                                                       | `Client Secret`                            | secret                                                                                                                              | `:15-19`                                                                                                                             |
|                                                       | `Scope`                                    | text                                                                                                                                | `:20-23`                                                                                                                             |
|                                                       | `State` (?)                                | text                                                                                                                                | tooltip: `If left empty, Bruno automatically generates a secure random value to help protect against CSRF attacks.` (`:24-28`)       |
|                                                       | `Add Credentials to`                       | dropdown                                                                                                                            | `Request Body` (default), `Basic Auth Header` (`:231-249`)                                                                           |
|                                                       | `Use PKCE`                                 | checkbox                                                                                                                            | `:250-259`                                                                                                                           |
| `Token` (`IconKey`)                                   | `Token Source`                             | dropdown                                                                                                                            | `Access Token`, `ID Token` (`:268-286`)                                                                                              |
|                                                       | `Token ID`                                 | text                                                                                                                                | `:287-302`                                                                                                                           |
|                                                       | `Add token to`                             | dropdown                                                                                                                            | options `Header`, `URL`; trigger displays `Headers` or `URL` (`:303-321`)                                                            |
|                                                       | `Header Prefix` (when Header)              | text                                                                                                                                | `:325-341`                                                                                                                           |
|                                                       | `Query Param Key` (when URL)               | text                                                                                                                                | `:343-360`                                                                                                                           |
| `Advanced Settings` (`IconAdjustmentsHorizontal`)     | `Refresh Token URL`                        | text                                                                                                                                | `:361-384`                                                                                                                           |
| `Settings` (`IconSettings`)                           | `Automatically fetch token if not found`   | checkbox + (?)                                                                                                                      | tooltip `Automatically fetch a new token when you try to access a resource and don't have one.` (`:393-411`)                         |
|                                                       | `Auto refresh token (with refresh URL)`    | checkbox + (?)                                                                                                                      | disabled until Refresh Token URL set; tooltip `Automatically refresh your token using the refresh URL when it expires.` (`:413-431`) |
| `Additional Parameters` (`IconAdjustmentsHorizontal`) | tabs `Authorization` / `Token` / `Refresh` | table `Key` \| `Value` \| `Send In` \| ☑ 🗑, plus `Add Parameter`                                                                   | `AdditionalParams/index.js:169-300`                                                                                                  |
| (footer buttons)                                      | —                                          | `Get Access Token`, `Refresh Token` (only when a refresh token exists), `Cancel Authorization` (system-browser mode), `Clear Cache` | `Oauth2ActionButtons/index.js:170-203`                                                                                               |

#### 3.12.2 Password Credentials

`OAuth2/PasswordCredentials/index.js` with `PasswordCredentials/inputsConfig.js`:

| Label              | Input  |
| ------------------ | ------ |
| `Access Token URL` | text   |
| `Username`         | text   |
| `Password`         | secret |
| `Client ID`        | text   |
| `Client Secret`    | secret |
| `Scope`            | text   |

Then the shared blocks: `Add Credentials to`, `Token` (`Token Source`, `Token ID`,
`Add token to`, `Header Prefix` / `Query Param Key`), `Refresh Token URL`,
`Settings` (`Automatically fetch token if not found`,
`Auto refresh token (with refresh URL)`), `Additional Parameters`, action
buttons (`PasswordCredentials/index.js:86,116-300`).

#### 3.12.3 Client Credentials

`OAuth2/ClientCredentials/inputsConfig.js`: `Access Token URL`, `Client ID`,
`Client Secret` (secret), `Scope`; then the same token/settings blocks
(`ClientCredentials/index.js:82-292`).

#### 3.12.4 Implicit

`OAuth2/Implicit/inputsConfig.js`: `Authorization URL`, `Client ID`, `Scope`,
`State`. Plus `Callback URL` (+ system-browser checkbox, `Implicit/index.js:117-153`),
`Token` block with `Add Token to` options `Headers` / `URL` (`:226-232`),
`Header Prefix` or `URL Query Key` (`:247,264`), settings checkbox
`Auto fetch token` (`:298`).

OAuth2 field styling: label `font.size.base`; editor box max 400px, radius 3px,
`input.border` / `input.bg`; checkbox `accent-color: primary.solid`
(`OAuth2/StyledWrapper.js`).

---

## 4. States

| State                           | Detail                                                                               | Source                                                      |
| ------------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------- |
| Auth tab indicator              | `StatusDot` when effective auth is configured                                        | `HttpRequestPane/index.js:98-111`                           |
| Secret field                    | rendered as `*` until revealed (mask editor)                                         | `SingleLineEditor/index.js:39,140-150`                      |
| Sensitive-value warning         | warning popover rendered next to secret fields when a literal secret is detected     | `SensitiveFieldWarning/index.js:5-29`                       |
| Read-only (inherited / example) | editors `readOnly`, dropdowns `disabled`, cursor `not-allowed`                       | `InheritedAuth:131-147`, `OAuth2/AuthorizationCode:169,183` |
| Auto-refresh disabled           | checkbox + label dimmed (`opacity-50`, `text-gray-500`)                              | `AuthorizationCode/index.js:419-422`                        |
| OAuth error                     | token fetch errors surface as toast **and** as an error response (`showOauth2Error`) | `Oauth2ActionButtons/index.js:50-64`                        |

## 5. Interactions & shortcuts

| Interaction           | Result                                                                                                                                                                                         | Source                                           |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------ |
| Change mode           | `updateRequestAuthMode`                                                                                                                                                                        | `AuthMode/index.js:19-30`                        |
| Edit field            | `updateAuth({ mode, content })` on each keystroke                                                                                                                                              | e.g. `BasicAuth/index.js:24-49`                  |
| `Enter` in field      | send request                                                                                                                                                                                   | `SingleLineEditor/index.js:75`                   |
| `Cmd/Ctrl + S`        | save                                                                                                                                                                                           | `keyMappings.js:9`                               |
| `Get Access Token`    | `fetchOauth2Credentials`; toasts `Token fetched successfully!` / `Token refreshed successfully!` / `Cleared cache successfully` / `Authorization cancelled` / `Failed to cancel authorization` | `Oauth2ActionButtons/index.js:69-166`            |
| Inherited source link | opens `<collection                                                                                                                                                                             | folder> settings → auth` tab and focuses the row | `InheritedAuth/index.js:66-94` |

## 6. Verbatim copy index

Mode labels (§2), field labels (§3), tooltips (§3.10, §3.12.1), hint (§3.11),
`Auth inherited from` (§3.2), `Advanced Settings`, `Get Access Token`,
`Refresh Token`, `Clear Cache`, `Cancel Authorization`, `Add Parameter`,
`Configuration`, `Signature`, `Advanced`, `Token`, `Settings`,
`Grant Type`, `No Auth`.

Screenshot: `refs/auth_bruno.png` (inherit state on a folder).

---

## 7. Open Questions

1. **EdgeGrid "Advanced Settings"** is rendered unconditionally in the current
   source (`EdgeGridAuth/index.tsx:147-159`) — the `BASIC_FIELDS` /
   `ADVANCED_FIELDS` split suggests a collapse was intended. Treat as
   always-expanded unless the running build shows otherwise.
2. **OAuth2 `Add token to` trigger shows `Headers` (plural) while the menu
   option is `Header`** (`AuthorizationCode/index.js:309,316`). Also
   `Implicit` uses menu label `Headers` (`Implicit/index.js:231`). Copy is
   inconsistent in source — pick one deliberately.
3. **Auth mode trigger label vs menu label** differ for NTLM (`NTLM Auth` vs
   `NTLM`) and AWS (`AWS Sig v4` vs `AWS Sig V4`). Both strings are in source;
   a Go port must choose per-surface.
4. **API Key "Add To" trigger** says `Query Params` (plural) while the menu
   option is `Query Param` (`ApiKeyAuth/index.js:117-121`).
5. **Password Credentials** repeats the Authorization Code strings
   (`Automatically fetch token if not found`, `Auto refresh token (with refresh URL)`)
   with identical tooltip text (`PasswordCredentials/index.js:278-284,300-306`),
   but the tooltip bubble is hard-coded `bg-gray-700 text-white` instead of the
   theme-driven style used elsewhere. Flagged as a styling inconsistency, not a
   copy one.
6. **Sensitive-field detection** rules live in `hooks/useDetectSensitiveField`
   and were not specified here; the warning button's own copy is out of scope.
