# Screen — Folder Settings

All paths are relative to `bruno/packages/bruno-app/src/` unless prefixed with
`design-system.md`.

Status: VERIFIED against source. Unconfirmable points are marked **UNVERIFIED**
inline and repeated in §10 Open Questions.

Scope: `components/FolderSettings/**` (20 files), the shared
`components/TagList`, `components/RequestPane/Auth/*` editors it reuses, and
`components/BulkEditor`.

Screenshot reference: `refs/vars_bruno.png` (dark theme, folder Vars tab) shows
the tab strip, the `Pre Request` / `Post Response` captions, the table header
row and the `Save` button exactly as specified here.

---

## 1. Layout

```text
┌ Folder Settings tab ─────────────────────────────────────────────────┐
│ Headers⁽²⁾  Script•  Test•  Vars⁽²⁾  Auth•  Docs  Settings•          │ ← tab strip (wraps)
│                                            [ Edit ]  [ ✨ AI Assist ] │ ← right slot (Docs / AI tabs)
│──────────────────────────────────────────────────────────────────────│
│                                                                       │
│   <active tab panel>                                                  │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

| Region             | Value                                                                                                                                                                                                                                                                                    | Source                                  |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| Root               | `flex flex-col h-full overflow-auto` → inner `flex flex-col h-full relative px-4 py-4` (**16 px padding**)                                                                                                                                                                               | `FolderSettings/index.js:91-92`         |
| Tab bar row        | `flex items-start justify-between gap-4`                                                                                                                                                                                                                                                 | `:93`                                   |
| Tab strip          | `flex flex-wrap items-center tabs`                                                                                                                                                                                                                                                       | `:94`                                   |
| Tab item           | identical chrome to Collection Settings: `padding 6px 0`, bottom 2 px transparent → `tabs.active.border`, margin-right `tabs.marginRight` (`1.2rem`), idle `colors.text.subtext0`, hover + active `tabs.active.color`, active weight = `tabs.active.fontWeight` (**400**), no focus ring | `FolderSettings/StyledWrapper.js:11-39` |
| Content            | `folder-settings-content flex mt-4 flex-1 min-h-0 overflow-auto`                                                                                                                                                                                                                         | `:130`                                  |
| Count superscript  | `<sup class="ml-1 font-medium">` on Headers / Vars                                                                                                                                                                                                                                       | `:97,109`                               |
| Dot                | `<StatusDot />` (Script / Test / Auth / Settings)                                                                                                                                                                                                                                        | `:101,105,113,120`                      |
| Markdown           | `.markdown-body { height:auto; overflow-y:visible }`                                                                                                                                                                                                                                     | `StyledWrapper.js:6-9`                  |
| Muted helper class | `.muted → colors.text.muted`                                                                                                                                                                                                                                                             | `StyledWrapper.js:51-53`                |

### 1.1 Tab strip (exact order) — `FolderSettings/index.js:95-121`

| #   | id         | Label                                                      | Indicator condition    | Source               |
| --- | ---------- | ---------------------------------------------------------- | ---------------------- | -------------------- |
| 1   | `headers`  | `Headers` + count superscript when > 0                     | enabled header count   | `:95-98`, `:33-34`   |
| 2   | `script`   | `Script` + dot when any script body exists                 | `:99-102`, `:30`       |
| 3   | `test`     | `Test` — **singular** (`folder-settings-tab-test`)         | dot when tests exist   | `:103-106`, `:31`    |
| 4   | `vars`     | `Vars` + count superscript when > 0                        | enabled req + res vars | `:107-110`, `:36-38` |
| 5   | `auth`     | `Auth` + dot when effective auth exists (own or inherited) | `:111-114`, `:42-46`   |
| 6   | `docs`     | `Docs`                                                     | never                  | `:115-117`           |
| 7   | `settings` | `Settings` + dot when the folder has tags                  | `:118-121`, `:40`      |

Default tab: **`headers`** (`:23`). Selection stored per folder in
`collection.folderLevelSettingsSelectedTab[folder.uid]` (`:24-27`, `:48-56`).

Note the plural/singular drift: collection tab is `Tests`
(`CollectionSettings/index.js:137`), folder tab is `Test` (`:104`).

### 1.2 Right slot — `FolderSettings/index.js:123-129`

`AI_TABS = ['script','test','docs']` (`:19`).

| Tab                      | Controls                                     | Source                                                       |
| ------------------------ | -------------------------------------------- | ------------------------------------------------------------ |
| `docs`                   | `DocsAction` → `Edit` / `Preview` button     | `:125`; `components/Documentation/DocsAction/index.js:12-22` |
| `script`, `test`, `docs` | `SettingsAiAssist` — **AI, out of v1 scope** | `:126`                                                       |

---

## 2. Tab — Headers

`FolderSettings/Headers/index.js`; styles `Headers/StyledWrapper.js`.

Intro (verbatim): **`Request headers that will be sent with every request inside this folder.`** (`:138-140` show in bulk mode, `:153-155` normal).

| Column        | Key           | Width  | Placeholder   | Editor                                                           | Source                     |
| ------------- | ------------- | ------ | ------------- | ---------------------------------------------------------------- | -------------------------- |
| checkbox      | `enabled`     | —      | —             | native checkbox                                                  | `EditableTable`            |
| `Name`        | `name`        | 20 %   | `Name`        | `SingleLineEditor` + HTTP-header autocomplete, newlines stripped | `:84-106`                  |
| `Value`       | `value`       | flex   | `Value`       | `SingleLineEditor` + `MimeTypes` autocomplete                    | `:108-124`                 |
| `Description` | `description` | `70px` | `Description` | `createDescriptionColumn`                                        | `:125`, `StyledWrapper.js` |

Validation: `Header name cannot contain spaces or newlines`,
`Header value cannot contain newlines` (same `getRowError` as collection,
`Headers/index.js:61-75` pattern).

Footer: **`Bulk Edit`** text link right-aligned (`:170-174`); `Save`
`Button size="sm"` at `mt-6` (`:175-179`). Bulk mode: intro line + `BulkEditor`
(200 px plain CodeMirror) + link **`Key/Value Edit`**
(`components/BulkEditor/index.js:76-92`).

Table chrome identical to the collection Headers table (border
`table.border`, thead `table.thead.color`, `6px 10px` cell padding).

Empty state: none (blank add row).

---

## 3. Tab — Script

`FolderSettings/Script/index.js`.

Intro (verbatim): **`Pre and post-request scripts that will run before and after any request inside this folder is sent.`** (`:110-112`).

| Element  | Detail                                                                                                             | Source             |
| -------- | ------------------------------------------------------------------------------------------------------------------ | ------------------ |
| Sub-tabs | `Pre Request` / `Post Response` (`components/Tabs`)                                                                | `:114-126`         |
| Dot      | `StatusDot` per sub-tab when its script is non-empty; `type="error"` when any child request reports a script error | `:116-126,105-106` |
| Editor   | `CodeEditor mode="javascript"`, pre: `showHintsFor ['req','bru']`, post: `['req','res','bru']`                     | `:131-167`         |
| Save     | `Button size="sm"` at `mt-12`                                                                                      | `:172-178`         |

Default sub-tab logic: same as collection — pre-request when present, else
post-response (`:29-34` in the collection equivalent; folder uses the identical
pattern at `Script/index.js` top). No placeholder text.

---

## 4. Tab — Test

`FolderSettings/Tests/index.js`.

Intro (verbatim): **`These tests will run any time a request in this folder is sent.`** (`:42`).
Single `CodeEditor mode="javascript"` (`showHintsFor ['req','res','bru']`) + `Save`
at `mt-6` (`:44-64`). No placeholder, no empty state.

---

## 5. Tab — Vars

`FolderSettings/Vars/index.js`; table `Vars/VarsTable/index.js`.

| #   | Caption (verbatim) | `varType`  | Data path                        | Source                |
| --- | ------------------ | ---------- | -------------------------------- | --------------------- |
| 1   | `Pre Request`      | `request`  | `root.request.vars.req`          | `Vars/index.js:25-26` |
| 2   | `Post Response`    | `response` | `root.request.vars.res`          | `Vars/index.js:29-30` |
| 3   | —                  | —          | `Save` `Button size="sm"` `mt-6` | `Vars/index.js:32-36` |

Columns (`Vars/VarsTable/index.js:74-128`):

| Column                                            | Placeholder      | Notes                                                                                                                                                                                                                     | Source           |
| ------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| checkbox                                          | —                | enabled                                                                                                                                                                                                                   | `EditableTable`  |
| `Name`                                            | `Name`           | `width:25%`, sortable, reorderable                                                                                                                                                                                        | `:75-83`         |
| `Value` (request) / `Expr` (response) + `InfoTip` | `Value` / `Expr` | Response tooltip: **`You can write any valid JS expression here`** — note this differs from the **collection** copy (`…JS Template Literal here`, `CollectionSettings/Vars/VarsTable/index.js:81`). A real inconsistency. | `:84-104`, `:88` |
| `Description`                                     | `Description`    | `createDescriptionColumn({ item: folder, nameFromRowIndex:true })`                                                                                                                                                        | `:66-72,127`     |

`renderTypeSelector` → `DataTypeSelector` for request vars only (`:107-120`);
`SensitiveWarningPopover` when used in a sensitive field (`:121-123`).
Name error: `Variable contains invalid characters. Must only contain alphanumeric characters, "-", "_", "."` (`:57-64`).

`defaultRow = { name:'', value:'', description:'', ...(response → { local:false }) }` (`:130-135`).
Table ids `folder-vars`; test ids `folder-vars-req` / `folder-vars-res`
(`:138-155`). Sorting key `folder-vars-sort::{folderUid}::{varType}` (`:147`).

---

## 6. Tab — Auth

`FolderSettings/Auth/index.js`.

Intro (verbatim): **`Configures authentication for the entire folder. This applies to all requests using the Inherit option in the Auth tab.`** (`:202-205`).

Selector block: `AuthMode` then — **only when mode is `inherit`** — an
`InheritedAuthSourceLabel` line (`:206-211`).

### 6.1 Mode menu — `FolderSettings/AuthMode/index.js:25-92`

Same 12 items and order as the request Auth tab (i.e. **includes `Inherit`**
here, unlike the collection):

`AWS Sig v4`, `Basic Auth`, `Bearer Token`, `Digest Auth`, `NTLM Auth`,
`OAuth 1.0`, `OAuth 2.0`, `WSSE Auth`, `API Key`, `Akamai EdgeGrid` + `Beta`,
`Inherit`, `No Auth`.

`StatusBadge status="info" size="xs"` on Akamai EdgeGrid (`:72-81`),
`ariaLabel='Akamai EdgeGrid (Beta)'`. MenuDropdown `bottom-start`, `appendTo` body,
`hideOnReferenceClip`, `showTickMark` (`:97-109`).

### 6.2 Per-mode fields

Folder auth reuses the **request** Auth editors
(`components/RequestPane/Auth/*`) with `updateAuth={updateFolderAuth}` and
`save` bound to `saveFolderRoot` (`FolderSettings/Auth/index.js:73-198`).
Therefore field labels/order are exactly those in `screens/request-auth.md` §3:

| Mode              | Fields                                                                                                                                                | Wired via                    | Source            |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | ----------------- |
| `basic`           | `Username`, `Password`⛨                                                                                                                               | `RequestPane/Auth/BasicAuth` | `:75-85`          |
| `bearer`          | `Token`⛨                                                                                                                                              | `BearerAuth`                 | `:86-96`          |
| `digest`          | `Username`, `Password`⛨                                                                                                                               | `DigestAuth`                 | `:97-107`         |
| `ntlm`            | `Username`, `Password`⛨, `Domain`                                                                                                                     | `NTLMAuth`                   | `:108-118`        |
| `oauth1`          | OAuth 1 fields (see request-auth §3)                                                                                                                  | `OAuth1`                     | `:119-129`        |
| `wsse`            | `Username`, `Password`⛨                                                                                                                               | `WsseAuth`                   | `:130-140`        |
| `apikey`          | `Key`, `Value`⛨, `Add To`                                                                                                                             | `ApiKeyAuth`                 | `:141-151`        |
| `awsv4`           | `Access Key ID`, `Secret Access Key`⛨, `Session Token`⛨, `Service`, `Region`, `Profile Name`                                                          | `AwsV4Auth`                  | `:152-162`        |
| `oauth2`          | `GrantTypeSelector` (`Password Credentials`, `Authorization Code`, `Implicit`, `Client Credentials`) + grant body; default grant `authorization_code` | inline                       | `:163-175`, `:35` |
| `inherit`         | `InheritedAuth` panel (read-only source auth)                                                                                                         | `InheritedAuth`              | `:176-178`        |
| `none`            | nothing                                                                                                                                               | returns `null`               | `:192-193`        |
| `akamai-edgegrid` | EdgeGrid fields (shared with collection, §5.2 there)                                                                                                  | `EdgeGridAuth`               | `:179-191`        |

Unhandled grant: `<div>TBD</div>` (`:47`).

Save: `Button size="sm"` at `mt-6` (`:213-217`).

---

## 7. Tab — Docs

`FolderSettings/Documentation/index.js`.

No intro paragraph; no heading inside the panel — the `Docs` tab label is the
title. Rendered:

| Element       | Detail                                                                      | Source                              |
| ------------- | --------------------------------------------------------------------------- | ----------------------------------- |
| Body          | `DocsEditor` (rich-text preview or edit; markdown mode toggle in edit mode) | `:42-54`                            |
| Edit toggle   | `Edit` / `Preview` in the tab-bar right slot (§1.2)                         | `FolderSettings/index.js:125`       |
| Save          | `Button size="sm"` at `mt-6`, **only while editing**                        | `:56-62`                            |
| Empty preview | **no** custom placeholder (unlike collection docs)                          | `:42-54` (no `emptyPreviewContent`) |

`if (!folder) return null` (`:36-38`).

---

## 8. Tab — Settings

`FolderSettings/Settings/index.js` → `Settings/Tags/index.js` →
`components/TagList`.

Intro (verbatim): **`Configure settings for this folder.`** (`Settings/index.js:8`).

Section heading: `IconTag` 16 px + `Tags` (`text-xs font-medium`) (`:11-14`).

### 8.1 Controls — `components/TagList/index.js`

| Part                   | Detail                                                                                                                                                                                         | Source     |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Input                  | `SingleLineEditor` with `className="border border-gray-500/50 px-2"`, placeholder **`e.g. Production ready`**, autocomplete from other items' tags, `showHintsOnClick`, `{{var}}` autocomplete | `:55-69`   |
| Add                    | `Enter`/run → validated then `addFolderTag` and input cleared                                                                                                                                  | `:22-51`   |
| Chip                   | `IconTag` 12 px + text + remove `IconX` 12 px (`title="Remove tag"`)                                                                                                                           | `:71-90`   |
| Inherited toggle       | chevron + **`{n} Inherited from parent`**                                                                                                                                                      | `:91-108`  |
| Inherited chip tooltip | **`Inherited from folder "{name}"`** with `IconFolder` 12 px                                                                                                                                   | `:110-124` |
| Error text             | `text-xs text-red-500` under the input                                                                                                                                                         | `:70`      |

Validation error strings (verbatim, `:26-41`):

| Condition                  | Message                                                            |
| -------------------------- | ------------------------------------------------------------------ |
| BRU format invalid chars   | `Tags in BRU format must only contain letters, numbers, "-", "_".` |
| YML/other invalid chars    | `Tags must only contain letters, numbers, spaces, "-", "_"`        |
| Duplicate own tag          | `Tag "{text}" already exists`                                      |
| Same tag already inherited | `Tag "{text}" is already inherited from folder "{folderName}"`     |

Format regex: bru `/^[\p{L}\p{N}_-]+$/u`; otherwise
`/^[\p{L}\p{N}_-](?:[\p{L}\p{N}_\s-]*[\p{L}\p{N}_-])?$/u` (`:12`).
Tags cascade from parent folders (`getInheritedTagSourcesForItem`, `:14-15`).

No save button in this tab — tag add/remove persists via `saveFolderRoot` on
`onSave` of the editor (`:30-32,42`). No empty state text (the chip list is
simply empty).

---

## 9. Interaction summary

| Interaction                | Result                                                                           | Source                                                                                     |
| -------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Switch tab                 | Redux `updatedFolderSettingsSelectedTab`                                         | `FolderSettings/index.js:48-56`                                                            |
| `Save` in any tab          | `saveFolderRoot(collection.uid, folder.uid)`                                     | `Headers:59`, `Vars/index.js:16`, `Auth:57-59`, `Tests:32`, `Script:100-102`, `Tags:30-32` |
| Header/Vars row add/delete | `EditableTable` trailing blank row / trash                                       | `EditableTable`                                                                            |
| Vars reorder               | Drag when sort is `Default order` (rows carry a grip handle)                     | `Vars/VarsTable/index.js:53-55`                                                            |
| Tags                       | `Enter` adds; chip × removes; inherited tags read-only                           | `TagList`                                                                                  |
| Folder auth `inherit`      | Shows `InheritedAuthSourceLabel` naming the source (collection or parent folder) | `Auth/index.js:68-71,208-210`                                                              |

Folder settings **inherit downward**: headers/vars/scripts/tests/auth set here
apply to every request in the folder and its sub-folders (copy in each tab intro).

---

## 10. Open Questions

1. **`Test` (folder) vs `Tests` (collection)** tab label — confirm the port keeps
   the singular for folders (source does).
2. **`Expr` tooltip wording differs** between collection ("JS Template Literal")
   and folder ("JS expression"). Which is authoritative? Template Literal is
   technically correct for `{{ }}` expressions; the collection copy is likely the
   intended one.
3. **No `Overview` tab** for folders (unlike collections) — confirm that is
   intentional in the Go port's information architecture.
4. **Tags tab has no explicit Save** — persistence relies on the editor's
   `onSave`. Verify `addFolderTag`/`deleteFolderTag` actually persist without an
   explicit save, or add a Save button in the port.
5. **Right slot AI button** is deferred; only `Edit` (Docs) is in v1 scope.
6. **Inherited-auth source label copy** is owned by
   `components/RequestPane/Auth/InheritedAuth` — see `screens/request-auth.md` §3.2;
   not duplicated here.
7. **OAuth2 grant default** is `authorization_code` (`Auth/index.js:35`) while the
   collection uses `{}` and shows `TBD` for unknown grants
   (`CollectionSettings/Auth/OAuth2/index.js:21,37`). Harmonise in the port.
