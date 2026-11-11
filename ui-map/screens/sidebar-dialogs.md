# Screen — Sidebar dialogs

Modal and popover surfaces launched from the sidebar: New Request, New Folder,
Import Collection (incl. location + settings), Bulk Import, Open Collection,
Create Collection, Clone Git Repository, Close Workspace, Postman Package Report.

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source. Values are exact; unverifiable items are marked
**inference** or listed in Open Questions.

Screenshots: `refs/new_request_modal_bruno.png`, `refs/new_folder_modal_bruno.png`,
`refs/import_collection_bruno.png`.

---

## 1. Shared modal frame (`components/Modal/index.js`, `Modal/StyledWrapper.js`)

Every dialog in this file except the anchored dropdowns is a `Modal` rendered through
`Portal` (`Modal/index.js:152-197` for the shell; consumers wrap with `components/Portal`).

| Aspect                        | Value                                                                                                                             | Source                                            |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------- |
| Position                      | `fixed`, full viewport, `align-items: flex-start`, `z-index: 20`                                                                  | `Modal/StyledWrapper.js:15-27`                    |
| Card margin                   | `margin-top: 50px`, `margin: 3vh 10vw`                                                                                            | `:42-43`                                          |
| Card background               | `modal.body.bg` (`#ffffff` L / `#222224` D)                                                                                       | `:32`; tokens `modal.body.bg`                     |
| Card border / radius / shadow | `1px solid border.border0`; `border.radius.base` (`6px`); `var(--box-shadow-base)`                                                | `:33-44`                                          |
| Size `sm`                     | `min-width: 300px`, `max-width: 500px`                                                                                            | `:46-49`                                          |
| Size `md`                     | `min-width: 500px`, `max-width: 800px`                                                                                            | `:51-54`                                          |
| Size `lg`                     | `min-width: 800px`, `max-width: 1140px`                                                                                           | `:56-59`                                          |
| Header                        | `modal.title.bg` (`#f1f1f1` L / `#1a1a1a` D), `modal.title.color`, `font.size.md`, padding `0.5rem 1rem`                          | `:74-83`; tokens `modal.title.*`                  |
| Header close                  | `IconX` `16px` stroke `1.5` in a `24 × 24` box, opacity `0.7` → `1` on hover with `rgba(modal.title.color, .1)` background        | `Modal/index.js:14-22`; `StyledWrapper.js:91-109` |
| Body padding                  | `px-4 py-4` (`1rem`) unless `noPadding`                                                                                           | `Modal/index.js:26-28`                            |
| Footer                        | padding `1rem`, cancel (`secondary`/`ghost`) then confirm (`primary` by default)                                                  | `Modal/index.js:50-80`                            |
| Backdrop                      | black at `modal.backdrop.opacity` `0.4`, fade-in `0.1s`                                                                           | `StyledWrapper.js:158-170`                        |
| Enter animation               | `fade-and-slide-in-from-top 0.5s forwards cubic-bezier(0.19, 1, 0.22, 1)`                                                         | `StyledWrapper.js:66`                             |
| Exit animation                | `fade-out` / `fade-and-slide-out-from-top 0.5s`, default `closeModalFadeTimeout` `500 ms`                                         | `:7-12`; `Modal/index.js:99,133-136`              |
| Textbox                       | radius `border.radius.sm` (`4px`), border `1px input.border`, padding `0.45rem`, background `input.bg`; focus `input.focusBorder` | `StyledWrapper.js:116-133`                        |
| Radio / checkbox accent       | `primary.solid`                                                                                                                   | `:186-195`                                        |
| Esc                           | closes (`closeModal({type:'esc'})`) unless `disableEscapeKey`                                                                     | `Modal/index.js:117-121`                          |
| Enter                         | confirms when focus is inside the modal, not on a submit button and not disabled                                                  | `:122-127`                                        |
| Focus trap                    | `useFocusTrap(modalRef)`                                                                                                          | `:131`                                            |
| Outside click                 | closes unless `disableCloseOnOutsideClick`                                                                                        | `:186-195`                                        |

**Note on default footer labels.** `Modal` defaults `confirmText` to `Save` and
`cancelText` to `Cancel` (`Modal/index.js:43-44`).

---

## 2. New Request — `components/Sidebar/NewRequest` (modal)

Layout (`refs/new_request_modal_bruno.png`, source `NewRequest/index.js:312-606`):

```text
┌ New Request ───────────────────────────────────────────────── ✕ ┐
│ Type                                                             │
│   ⦿ HTTP          ⦿ gRPC           ⦿ From cURL                    │
│   ○ GraphQL       ○ WebSocket                                    │
│                                                                  │
│ Request Name                                                     │
│ ┌──────────────────────────────────────────────────────────────┐ │
│ │ Request Name                                                 │ │
│ └──────────────────────────────────────────────────────────────┘ │
│ URL                                                              │
│ ┌ GET ▾ ┬─────────────────────────────────────────────────────┐ │
│ │       │ Request URL                                         │ │
│ └───────┴─────────────────────────────────────────────────────┘ │
│ Options ▾                                Cancel   [ Create ]     │
└──────────────────────────────────────────────────────────────────┘
```

| Aspect                   | Value                                                                                                                    | Source                                          |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------- |
| Type                     | Modal `md`, title `New Request`, `hideFooter` (footer built in the form)                                                 | `NewRequest/index.js:315`                       |
| Type radios              | `grid grid-cols-3 gap-2`, each column `flex flex-col gap-2`, radio accent `primary.solid`                                | `:325-405`                                      |
| Radio labels (verbatim)  | `HTTP`, `GraphQL` (col 1); `gRPC`, `WebSocket` (col 2); `From cURL` (col 3)                                              | `:337-400`                                      |
| Default type             | collection preset or `http-request`                                                                                      | `:79-105,112`                                   |
| Request Name label       | `Request Name` (`font-medium`)                                                                                           | `:408-410`                                      |
| Request Name placeholder | `Request Name`                                                                                                           | `:415`                                          |
| Request Name input       | `.textbox mt-2 w-full`, height `2.1rem`, radius `border.radius.base`                                                     | `:411-428`; `NewRequest/StyledWrapper.js:31-34` |
| Name → filename          | filename auto-derived with `sanitizeName` until the user edits it                                                        | `:422-425`                                      |
| File Name (advanced)     | label `File Name` + muted `(on filesystem)`; placeholder `File Name`; suffix `.{collection.format}`                      | `:433-492`                                      |
| URL label                | `URL`                                                                                                                    | `:496-498`                                      |
| Method selector          | `HttpMethodSelector` with caret, left radius `border.radius.base`, height `2.1rem`                                       | `:500-508`; `StyledWrapper.js:4-15`             |
| URL editor               | `SingleLineEditor`, placeholder `Request URL`, height `2.1rem`, right radius `border.radius.base`                        | `:509-533`; `StyledWrapper.js:16-29`            |
| cURL branch              | label `cURL Command`, dropdown `HTTP` / `GraphQL`, textarea placeholder `Enter cURL request here..`, `min-height: 150px` | `:540-578`; `StyledWrapper.js:36-38`            |
| Options control          | text-link `Options` + caret (`textLink` fill)                                                                            | `:301-310,580-594`; `StyledWrapper.js:47-52`    |
| Options item             | `Show Filesystem Name` / `Hide Filesystem Name`                                                                          | `:591`                                          |
| Footer                   | `Cancel` (`secondary`, `ghost`), `Create` (`primary`)                                                                    | `:596-601`                                      |

Validation (`NewRequest/index.js:117-148`), message text verbatim:

| Field         | Rule                             | Message                                                                                    |
| ------------- | -------------------------------- | ------------------------------------------------------------------------------------------ |
| `requestName` | trimmed, min 1                   | `must be at least 1 character`                                                             |
| `requestName` | max 255                          | `must be 255 characters or less`                                                           |
| `requestName` | required                         | `name is required`                                                                         |
| `filename`    | min 1 / max 255 / required       | `must be at least 1 character` / `must be 255 characters or less` / `filename is required` |
| `filename`    | `validateName`                   | `validateNameError(value)` (`utils/common/regex`)                                          |
| `filename`    | not reserved                     | `The file names "collection" and "folder" are reserved in bruno`                           |
| `curlCommand` | required when type = `from-curl` | `curlCommand is required`                                                                  |
| `curlCommand` | parseable curl                   | `Invalid cURL Command`                                                                     |

Errors render as `.text-red-500` directly below the field (`:429-431,488-490,575-577`).
Pasting a `curl` command into the form switches the type to `From cURL`
(`:264-287`). Success toast: `New request created!` (`:166,183,231,248`); failure
`An error occurred while adding the request` (`:169` etc.).

The same modal is reused for ephemeral requests with `isEphemeral` (`:187-210`,
opened from `RequestTabs/index.js:122-124`) and for `NewRequest` triggered from tab and
sidebar shortcuts (see `request-tabs.md`).

---

## 3. New Folder — `components/Sidebar/NewFolder` (modal)

```text
┌ New Folder ─────────────────────────────────────────────── ✕ ┐
│ Folder Name                                                   │
│ ┌───────────────────────────────────────────────────────────┐ │
│ │                                                           │ │
│ └───────────────────────────────────────────────────────────┘ │
│ Options ▾                                  Cancel   [ Create ] │
└───────────────────────────────────────────────────────────────┘
```

| Aspect                      | Value                                                        | Source                  |
| --------------------------- | ------------------------------------------------------------ | ----------------------- |
| Modal                       | `md`, title `New Folder`, `hideFooter`                       | `NewFolder/index.js:88` |
| Field label                 | `Folder Name`                                                | `:90-92`                |
| Input                       | `.textbox mt-2 w-full`, no placeholder attribute             | `:93-109`               |
| Filesystem label (advanced) | `Folder Name` + `(on filesystem)`; placeholder `Folder Name` | `:117-148`              |
| Options item                | `Show Filesystem Name` / `Hide Filesystem Name`              | `:181`                  |
| Footer                      | `Cancel`, `Create`                                           | `:185-192`              |
| Autofocus                   | input focused on mount                                       | `:65-69`                |

Validation (`NewFolder/index.js:33-54`): `folderName` required → `name is required`;
min 1 → `must be at least 1 character`. `directoryName` required → `foldername is required`;
`validateName` → `validateNameError(value)`; at collection root a name containing
`environments` is rejected with
`The folder name "environments" at the root of the collection is reserved in bruno`
(`:46-53`). Success toast `New folder created!` (`:58`).

---

## 4. Import Collection — `components/Sidebar/ImportCollection` (modal)

Title `Import Collection`, `md`, `hideFooter` (`ImportCollection/index.js:42`);
body width `IMPORT_COLLECTION_SELECTION_WIDTH = 600px` (`ImportCollection/StyledWrapper.js:4-8`;
`SelectionList/constants.js:2`).

### 4.1 Tabs

| #   | id       | Icon                   | Label (verbatim) | Source                            |
| --- | -------- | ---------------------- | ---------------- | --------------------------------- |
| 1   | `file`   | `IconFileImport` 18 px | `File`           | `ImportCollection/index.js:46-53` |
| 2   | `github` | `IconBrandGit` 18 px   | `Git Repository` | `:54-61`                          |
| 3   | `url`    | `IconUnlink` 18 px     | `URL`            | `:62-69`                          |

Tab metrics: padding `6px 0`, bottom border `2px transparent`, margin-right `1.25rem`;
inactive colour `var(--color-tab-inactive)`; active colour `tabs.active.color`
(`#343434` L / `#cccccc` D) and `2px` `tabs.active.border` (`#d5882a` L / `#e7b85f` D)
(`ImportCollection/StyledWrapper.js:10-32`; tokens `tabs.active`).

### 4.2 Error banner

When a tab sets an error: `mb-4 p-2 border rounded-md`, background
`status.danger.background` (`rgba(206,79,59,.15)` L / `rgba(218,70,47,.15)` D),
border `status.danger.border`, `text-xs` text in `status.danger.text`, close `IconX` 16 px
(`ImportCollection/index.js:73-98`; tokens `status.danger.*`).

### 4.3 File tab (`ImportCollection/FileTab.js`)

| Aspect         | Value                                                                                                                                                                                                   | Source               |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------- |
| Drop zone      | `border-2 border-dashed rounded-lg p-6`; idle border `#e5e7eb` (`border-gray-200`) / `#374151` (`border-gray-700`) dark; active `#3b82f6` border + `#eff6ff` bg (dark `#60a5fa` / `rgba(30,58,138,.2)`) | `FileTab.js:231-237` |
| Icon           | `IconFileImport` `28px`, grey `#9ca3af` / `#6b7280`                                                                                                                                                     | `:240-243`           |
| Primary copy   | `Drop file(s) to import or` + link `choose file(s)` (underline, `theme.textLink`)                                                                                                                       | `:252-261`           |
| Formats copy   | `Supports Bruno, OpenCollection, Postman, Insomnia, OpenAPI 3.x / Swagger 2.0, WSDL, and ZIP formats`                                                                                                   | `:263-265`           |
| Accepted files | `.json .yaml .yml .wsdl .zip` + MIME list; `multiple`                                                                                                                                                   | `:24-37,244-251`     |
| Errors         | `Cannot mix ZIP files with other file types. Please select either a single ZIP file OR collection files (JSON/YAML)`                                                                                    | `:180`               |
| Errors         | `Multiple ZIP files selected. Please select only one ZIP file at a time for import.`                                                                                                                    | `:185`               |
| Errors (toast) | `The ZIP file is not a valid Bruno collection`, `Import ZIP file failed`, `Import multiple files failed`, `Import collection failed`                                                                    | `:66,68,122,166`     |

### 4.4 Git Repository tab (`ImportCollection/GitHubTab.js`)

| Aspect            | Value                                                    | Source   |
| ----------------- | -------------------------------------------------------- | -------- |
| Input placeholder | `Enter Git repository URL`                               | `:36`    |
| Button            | `Clone` (`filled`, `primary`), disabled when input empty | `:39-48` |
| Error             | `Please enter a valid git repository URL`                | `:13`    |

On submit the modal closes and `Clone Git Repository` opens pre-filled
(`Sections/CollectionsSection/index.js:104-108,390-396`).

### 4.5 URL tab (`ImportCollection/UrlTab.js`)

| Aspect            | Value                                                             | Source   |
| ----------------- | ----------------------------------------------------------------- | -------- |
| Input placeholder | `Enter URL (OpenAPI/Swagger, Postman, or Insomnia specification)` | `:44`    |
| Button            | `Import` (`filled`, `primary`), disabled when input empty         | `:47-57` |
| Error             | `Please enter a valid URL`                                        | `:15`    |
| Error             | `URL import failed. Please check the URL and try again.`          | `:25`    |

### 4.6 Fullscreen loader (`ImportCollection/FullscreenLoader/index.js`)

During fetch/parse the tab is replaced by a fixed overlay:
`bg-white/80 dark:bg-zinc-900/80 backdrop-blur-sm`, centred card `p-8 rounded-lg shadow-lg max-w-md`,
spinner `IconLoader2 h-12 w-12`, heading `text-lg font-medium`, body text
`This may take a moment depending on the collection size` (`:31-38`).
Cycling headings every `2000 ms` (`:5-11,20-23`, verbatim):
`Processing collection...` → `Analyzing requests...` → `Translating scripts...` → `Preparing collection...` → `Almost done...`.

---

## 5. Import Collection (location) — `components/Sidebar/ImportCollectionLocation` (modal)

Title `Import Collection`, `md`, confirm `Import`, cancel default `Cancel`
(`ImportCollectionLocation/index.js:217-228`). `footerLeft` = `ImportOptionsMenu`
(omitted for `bruno-zip`, `:224-228`).

```text
┌ Import Collection ──────────────────────────────────────────── ✕ ┐
│ Name                                                              │
│   <detected collection name>                                      │
│ Location ⓘ                                                        │
│ ┌───────────────────────────────────────────────────────────────┐ │
│ │ /path/to/collections                                          │ │
│ └───────────────────────────────────────────────────────────────┘ │
│ Browse                                                            │
│ [File Format ▾]           (only after "Show file format")         │
│ ── Folder arrangement   Tags ▾ ─────────────────────────────      │
│ ── Check for Spec Updates  [toggle] ──────────────────────────    │
│ ── ▸ Advanced ────────────────────────────────────────────────    │
│ Options ▾                                   Cancel   [ Import ]   │
└───────────────────────────────────────────────────────────────────┘
```

| Field                      | Label / copy (verbatim)                                                                                                                                                                                                                                                            | Source                                                   |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Name                       | `Name`, value = detected name (fallbacks `Collection`, `OpenAPI Collection`, `Postman Collection`, `Insomnia Collection`, `Bruno Collection`, `OpenCollection`, `WSDL Collection`)                                                                                                 | `:32-61,232-235`                                         |
| Location label             | `Location` + help tip `Bruno stores your collections on your computer's filesystem.` / `Choose the location where you want to store this collection.`                                                                                                                              | `:238-244`                                               |
| Location input             | `.textbox` with cursor pointer; clicking opens directory picker                                                                                                                                                                                                                    | `:245-259`                                               |
| Browse link                | `Browse`                                                                                                                                                                                                                                                                           | `:265-273`                                               |
| File Format                | label `File Format`; options `OpenCollection (YAML)` (`yml`), `BRU Format (.bru)` (`bru`)                                                                                                                                                                                          | `:275-300`                                               |
| Options menu item          | `Show file format` / `Hide file format`                                                                                                                                                                                                                                            | `ImportSettings/ImportOptionsMenu/index.js:16-25`        |
| Folder arrangement         | label `Folder arrangement`, desc `Select whether to create folders according to the spec's paths or tags.`, options `Tags` / `Paths`                                                                                                                                               | `ImportSettings/FolderArrangementOption/index.js:4-18`   |
| Check for Spec Updates     | label `Check for Spec Updates`, desc `Stay notified of spec changes and sync your collection with the spec.`; Swagger 2.0 desc `OpenAPI Sync is not supported for Swagger 2.0 specs.` (toggle disabled)                                                                            | `ImportSettings/CheckForSpecUpdatesOption/index.js:4-17` |
| Advanced (OpenAPI)         | toggle chevron + title `Advanced`                                                                                                                                                                                                                                                  | `ImportSettings/ImportAdvancedSection/index.js:6-23`     |
| Naming requests            | label `Naming requests`; info `If an operation doesn't have the selected field, Bruno falls back to summary, operationId, description, then method and path.`; desc `How requests in the generated collection are named.`; options `Summary`, `operationId`, `Description`, `Path` | `ImportSettings/RequestNameSourceOption/index.js:5-23`   |
| Auth placement             | label `Set authentication at collection level`; info `On import, the spec's security scheme is added to the collection. Operations with a different scheme keep their own auth.`; desc `Requests and folders inherit it instead of getting their own copy.`                        | `ImportSettings/AuthPlacementOption/index.js:8-21`       |
| Preserve scripts (Postman) | label `Preserve scripts`, desc `Import Postman scripts without translating them.`                                                                                                                                                                                                  | `ImportCollectionLocation/index.js:320-331`              |
| File format help           | `Choose the file format for storing requests in this collection.` + `OpenCollection (YAML):` `Industry-standard YAML format (.yml files)` + `BRU:` `Bruno's native file format (.bru files)`                                                                                       | `:279-288`                                               |

Conditional visibility (`ImportCollectionLocation/index.js:118-124,303-331`):

| Row                           | Shown for                           |
| ----------------------------- | ----------------------------------- |
| `Folder arrangement`          | OpenAPI                             |
| `Check for Spec Updates`      | OpenAPI loaded from a URL or a file |
| `Advanced` (naming + auth)    | OpenAPI                             |
| `Advanced` (preserve scripts) | Postman                             |
| `Options` / `File Format`     | everything except `bruno-zip`       |

Defaults: arrangement `tags`, format `DEFAULT_COLLECTION_FORMAT`, auth placement
`DEFAULT_AUTH_PLACEMENT = 'request'`, request naming `DEFAULT_REQUEST_NAME_SOURCE = 'summary'`,
preserve scripts off (`:111-117`; `bruno-common/src/openapi-sync/index.ts:3,9,11`).

### 5.1 Settings control styling

| Control                       | Value                                                                                                                                                                                                            | Source                                                                                            |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `ImportSettingRow` layout     | label row `gap-1 font-medium` + info icon; description `text-muted text-xs mt-1`; switch rows are `label` elements with `cursor-pointer`                                                                         | `ImportSettings/ImportSettingRow/index.js:7-30`                                                   |
| `ImportSettingDropdown`       | background `background.surface1` (`#eaeaea` L / `#444444` D), border `1px background.surface2` (`#e5e5e5` L / `#666666` D), radius `4px`, padding `0.3rem 0.6rem`; caret `16px` filled in `colors.text.subtext0` | `ImportSettings/ImportSettingDropdown/StyledWrapper.js:5-13`; `ImportSettingDropdown/index.js:43` |
| `ImportSettingDropdown` width | reserves the widest option width via hidden sizer spans                                                                                                                                                          | `ImportSettingDropdown/index.js:39-41`; `StyledWrapper.js:17-28`                                  |
| `ImportAdvancedSection`       | top border `1px border.border1`, chevron `16px` rotating `90deg` in `0.15s ease`, collapsed content `height: 0`                                                                                                  | `ImportSettings/ImportAdvancedSection/StyledWrapper.js:4-15`                                      |
| Toggle size                   | `ToggleSwitch size="2xs"`                                                                                                                                                                                        | e.g. `AuthPlacementOption/index.js:18`                                                            |

---

## 6. Bulk Import — `components/Sidebar/BulkImportCollectionLocation` (modal)

Opened when the file tab receives multiple files or a bulk set
(`Sections/CollectionsSection/index.js:383-389`). Title `Bulk Import`, `md`
(`BulkImportCollectionLocation/index.js:612-615`).

Phase A — selection (`:786-962`):

| Element                       | Copy (verbatim)                                                                                                                               | Source     |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Collections list              | title `Collections`, search placeholder `Search Collections`, empty `No collections found`, 5 visible rows (`60 px` rows for multiple import) | `:787-803` |
| Environments list (bulk only) | title `Environments`, placeholder `Search Environments`, empty `No environments found`, 4 rows × `30 px`                                      | `:808-825` |
| Assignment heading            | `Environment Assignment`                                                                                                                      | `:827-830` |
| Checkbox                      | `Global Environment` + tip `Environments will be imported and stored as global, accessible across collections.`                               | `:839-845` |
| Checkbox                      | `Duplicate Across Collections` + tip `Each imported collection will receive its own copy of the environments.`                                | `:855-861` |
| Location heading              | `Location`                                                                                                                                    | `:869`     |
| Location placeholder          | `Select a location to save the collection`                                                                                                    | `:874`     |
| Browse                        | `Browse`                                                                                                                                      | `:892-896` |
| File Format                   | same help and options as §5                                                                                                                   | `:899-925` |
| OpenAPI block                 | `ImportAdvancedSection title="OpenAPI Import configuration"` (arrangement, spec updates, naming, auth)                                        | `:927-962` |
| Swagger note                  | `Swagger 2.0 specs are not supported in OpenAPI sync.`                                                                                        | `:944-948` |
| Footer                        | `Import` (disabled when nothing selected) / `Cancel`                                                                                          | `:616-623` |

Phase B — progress (`:632-783`): `Location` + value or `No location selected`;
`Importing Collections (N)` with `(f/N failed)`; rows show `IconLoader2` (blue,
`#3b82f6`), `IconCheck` (green `#22c55e`), `IconX` (red `#ef4444`); failing rows expose
`See error`; skipped counts render `{n} item(s) skipped` with a `Copy` action
(toast `Copied to clipboard`). `Importing Environments (N)` follows the same pattern.
Confirm button becomes `Close`, cancel is hidden (`importStarted`, `:581-585,616-623`).

Error modal: `sm`, title `Error Details`, no footer, pre-formatted red text
(`:593-607`).

Validation: `collectionLocation` required → `Location is required` (`:312-317`).

---

## 7. Open Collection — `components/Sidebar/OpenCollection` (modal)

The native directory picker runs first; if several collections are found a modal appears.

| Aspect           | Value                                                                                                | Source                            |
| ---------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------- |
| Title            | `Open Collection`; confirm `Open`; footer left `{selected} of {total} selected`                      | `OpenCollection/index.js:165-176` |
| Description      | `These collections were found inside your selection. Choose which ones to open.`                     | `:179-181`                        |
| Selection list   | title `Collections`, placeholder `Search Collections`, 8 rows × `60 px`, gap `4`                     | `:184-197`                        |
| Confirm disabled | nothing selected                                                                                     | `:171`                            |
| Toasts           | `Collection added to workspace` / `Collections added to workspace`; `Failed to open N collection(s)` | `:26-35`                          |
| Toasts           | `No Bruno collections found. Couldn't find a bruno.json or opencollection.yml`                       | `:80`                             |
| Toasts           | `No Bruno collections found. {n} skipped, config could not be read`                                  | `:78`                             |
| Toasts           | `Failed to scan N folder(s) for collections`                                                         | `:76,87`                          |
| Toast            | `An error occurred while scanning for collections`                                                   | `:115`                            |
| Fast path        | all picked folders are collections → opens immediately, no modal                                     | `:90-107`                         |

---

## 8. Create Collection — `components/Sidebar/CreateCollection` (modal)

Title `Create Collection`, `md`, `hideFooter` (`CreateCollection/index.js:117`).

| Field                  | Label / copy                                                                                                                                                                                                                       | Source     |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Name                   | `Name`                                                                                                                                                                                                                             | `:120-147` |
| Location               | `Location` + help `Bruno stores your collections on your computer's filesystem.` / `Choose the location where you want to store this collection.`; read-only input                                                                 | `:152-178` |
| Browse                 | `Browse`                                                                                                                                                                                                                           | `:182-188` |
| Folder Name            | `Folder Name` + help `The name of the folder used to store the collection.` / `You can choose a folder name different from your collection's name or one compatible with filesystem rules.`; only shown once the name is non-empty | `:190-244` |
| File Format (advanced) | `File Format`, options `OpenCollection (YAML)`, `BRU Format (.bru)`                                                                                                                                                                | `:246-276` |
| Options menu           | `Show Advanced Options` / `Hide Advanced Options`                                                                                                                                                                                  | `:290`     |
| Footer                 | `Cancel`, `Create`                                                                                                                                                                                                                 | `:294-302` |

Validation (`:46-62`): `collectionName` → `Collection name can't be empty` (min),
`Must be 255 characters or less`, `Collection name is required`; `collectionFolderName` →
`Must be at least 1 character`, `Must be 255 characters or less`, `validateNameError`,
`Folder name is required`; `collectionLocation` → `Location is required`;
`format` → `invalid format` / `Format is required`.
Success `Collection created!` (`:70`); failure `An error occurred while creating the collection`
plus IPC detail (`:73`).

Reached from the header `Create collection` item, the header `Add new collection` inline
creator's advanced cog, and the Welcome modal (`Sections/CollectionsSection/index.js:233-237,343-346`).

---

## 9. Clone Git Repository — `components/Sidebar/CloneGitRespository` (modal)

Title `Clone Git Repository`, `md` (`CloneGitRespository/index.js:283-293`).
If `gitVersion` is missing the modal is replaced by `GitNotFoundModal` (`:277-279`).

Form view (`:295-381`):

| Element                  | Value                                                                                                                                                               | Source     |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Repo header (pre-filled) | git badge box `p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg`, `IconBrandGit h-6 w-6 #8b5cf6`, name `text-sm font-semibold`, URL `text-xs text-muted font-mono` | `:298-311` |
| Repository URL           | label `Git Repository URL`, `.textbox mt-2 w-full`                                                                                                                  | `:314-331` |
| Location                 | label `Location`, read-only input, click to browse                                                                                                                  | `:335-358` |
| Branch                   | `Select` label `Branch`, searchable, placeholder `Loading branches...` / `Repository default`, empty `No branches match your search`, caret `IconChevronDown 12px`  | `:359-373` |
| Branch help              | `Branches could not be listed for this repository. Cloning will use the default branch.`                                                                            | `:374-378` |

Validation (`:143-146`): `Repository URL is required`, `Location is required`.

Progress view (`:382-433`): step list with `IconRefresh` spinning in
`status.warning.text` while running and `IconAlertCircle` in `status.danger.text` on error;
step titles `Cloning repository` → `Cloning successful` / `Cloning failed`, then
`Scanning for Bruno files` → `Scan successful`; log box `.info-box` background
`background.mantle`, border `1px border.border2`, radius `5px`, width `400px`,
`max-height: 150px` (`:56-62,117-134`; `CloneGitRespository/StyledWrapper.js:9-20`).
Scan result: `SkippedPathsWarning` + `SelectionList` (`Collections`, `Search Collections`,
8 × `60 px`) or the warning `No Bruno collections were found in this repository.`
(`:407-432`; warning style `status.warning.*`, `StyledWrapper.js:47-59`).

Footer logic (`:249-292`): confirm text `Clone` → `Close` → `Open`
(`getConfirmText`, `:270-275`); cancel hidden on error or empty scan; footer hidden while
steps run. Toasts: `Repository cloned successfully`, `Repository scanned successfully`.

---

## 10. Close Workspace — `components/Sidebar/CloseWorkspace` (modal)

| Aspect                 | Value                                                                                                                            | Source                          |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| Size / title / confirm | `sm`, `Close Workspace`, `Close`                                                                                                 | `CloseWorkspace/index.js:36-42` |
| Header line            | `IconFolder` 18 px + workspace name (`font-semibold`)                                                                            | `:43-46`                        |
| Path                   | `text-xs mt-1 break-words`                                                                                                       | `:47-49`                        |
| Copy                   | `Are you sure you want to close workspace {name}?`                                                                               | `:50-52`                        |
| Copy                   | `It will still be available in the file system at the above location and can be re-opened later.`                                | `:53-55`                        |
| Toasts                 | `Cannot close the default workspace`, `Workspace closed`, `Workspace not found`, `An error occurred while closing the workspace` | `:16,21,27,31`                  |

---

## 11. Postman Package Report — `components/Sidebar/PostmanPackageReport` (modal)

Shown after importing a Postman collection whose scripts reference npm packages or
unsupported APIs (`Sections/CollectionsSection/index.js:397-404`).

| Aspect                        | Value                                                                                                                                                                                                                                                                                                         | Source                                                           |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| Modal                         | `md`, title `Install packages`; card width forced `600px`                                                                                                                                                                                                                                                     | `PostmanPackageReport/index.js:177-189`; `StyledWrapper.js:8-10` |
| Cancel text                   | `Skip`                                                                                                                                                                                                                                                                                                        | `:181`                                                           |
| Confirm text                  | `Done` / `Installing…` / `Install N package(s)`                                                                                                                                                                                                                                                               | `:165-172`                                                       |
| Section card                  | border `1px border.border2`, radius `border.radius.base`, background `background.mantle`, padding `12px 14px`, stacked `10px` apart                                                                                                                                                                           | `StyledWrapper.js:12-23`                                         |
| Section title                 | uppercase, `font.size.sm`, weight `500`, `colors.text.muted`, letter-spacing `0.04em`, count pill `min-width: 20px; height: 18px; radius 999px`                                                                                                                                                               | `:33-55`                                                         |
| Packages section              | `Packages used in scripts` + count; help `These npm packages are referenced by scripts in your imported collection but aren't installed in this collection's folder.`                                                                                                                                         | `index.js:190-202`                                               |
| Manual command                | label `Or install manually`, command `npm install --save …` prefixed with `$ `, copy button (icon `IconCopy` → `IconCheck`)                                                                                                                                                                                   | `:204-222`; `StyledWrapper.js:185-243`                           |
| Installing                    | `Installing N package(s)…` with spinner                                                                                                                                                                                                                                                                       | `:224-229`                                                       |
| Installed                     | `Installed N package(s) into this collection.`                                                                                                                                                                                                                                                                | `:231-239`                                                       |
| Developer mode block          | title `Scripts use libraries that need Developer Mode`; desc `Your imported scripts call … which need Developer Mode to run.`; trust `Only enable Developer Mode for collections you trust.`; button `Switch to Developer Mode`                                                                               | `:243-268`                                                       |
| Unsupported block             | title `Not supported in Bruno`; help `Postman-specific packages without a Bruno equivalent. Scripts that call these will fail at runtime.`                                                                                                                                                                    | `:271-284`                                                       |
| Post-install (developer mode) | `This collection runs in Developer Mode - your scripts can use these packages right away.`                                                                                                                                                                                                                    | `:286-294`                                                       |
| Post-install (safe mode)      | title `External modules require Developer Mode`; desc `Custom npm packages (such as …) are installed, but this collection is currently running in Safe Mode.`                                                                                                                                                 | `:296-320`                                                       |
| Failure                       | `npm was not found on your PATH. Install Node.js/npm, then retry or run the command manually.`; `npm install timed out. Try running the command manually in a terminal.`; `Could not start npm install. Try running the command manually.`; `npm install failed (exit code N). Try the manual command above.` | `:55-67`                                                         |
| Package chip                  | monospace `font.size.sm`, background `background.base`, border `border.border2`, radius `border.radius.sm`, padding `3px 8px 3px 6px`                                                                                                                                                                         | `StyledWrapper.js:163-174`                                       |

The modal auto-closes when nothing is actionable (`:98-102`).

---

## 12. Anchored popovers referenced from this screen

| Popover                          | Trigger                    | Placement      | Source                                                                   |
| -------------------------------- | -------------------------- | -------------- | ------------------------------------------------------------------------ |
| Collections `Add new collection` | `+` in section header      | `bottom-end`   | `Sections/CollectionsSection/index.js:304-314`                           |
| Collections `More actions`       | `⋮` in section header      | `bottom-end`   | `:316-326`                                                               |
| API Specs add                    | `+` in section header      | `bottom-end`   | `Sections/ApiSpecsSection/index.js:45-55`                                |
| Row actions                      | `⋮` on row, or right-click | `bottom-start` | `CollectionRow/index.jsx:611-617`; `CollectionItemRow/index.jsx:820-826` |
| Empty CTA                        | `+ Add request`            | `bottom-start` | `EmptyCtaRow/index.jsx:27-35`                                            |

Popover surface metrics: min-width `160px`, padding `0.25rem`, radius `6px`, item padding
`0.275rem 0.625rem`, gap `0.5rem`, icon box `16 × 16` at `opacity 0.8`
(`components/Dropdown/StyledWrapper.js:4-86`).

---

## 13. Interactions and shortcuts

| Interaction           | Result                                                               | Source                                                                                    |
| --------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `Esc` in any modal    | closes (except `disableEscapeKey` dialogs, e.g. confirm-close)       | `Modal/index.js:117-121`                                                                  |
| `Enter` in a modal    | confirms (skipped when focus is a submit button or confirm disabled) | `:122-127`                                                                                |
| Click backdrop        | closes (except `disableCloseOnOutsideClick`)                         | `:186-195`                                                                                |
| Import shortcut       | `importCollection` → `⌘/Ctrl+O` opens Import Collection              | `keyMappings.js:56`; `Sections/CollectionsSection/index.js:66-69`                         |
| Inline create confirm | `Enter` / click-outside / green check                                | `InlineCollectionCreator/index.js:101-120`                                                |
| Inline create cancel  | `Esc` / red cross                                                    | `:116-118,164-173`                                                                        |
| New Folder advanced   | `Options ▾` → `Show Filesystem Name`                                 | `NewFolder/index.js:172-183`                                                              |
| New Request options   | `Options ▾` → `Show Filesystem Name`                                 | `NewRequest/index.js:582-593`                                                             |
| New Request paste     | pasting `curl …` switches to `From cURL`                             | `NewRequest/index.js:264-287`                                                             |
| Import location input | click opens directory picker (`browseDirectory`)                     | `ImportCollectionLocation/index.js:182-193,255`                                           |
| Import options menu   | `Options ▾` → `Show/Hide file format`                                | `ImportSettings/ImportOptionsMenu/index.js:16-35`                                         |
| Import advanced       | click `Advanced` toggles; `Esc` returns focus to trigger             | `ImportSettings/ImportAdvancedSection/index.js:11-21`; `ImportOptionsMenu/index.js:10-14` |
| Clone git             | footer goes `Clone` → `Close`/`Open`; `Back` on error                | `CloneGitRespository/index.js:219-275`                                                    |

---

## 14. Copy reference (i18n)

All dialog copy above is literal JSX text. `src/i18n/translation/en.json` has no keys for
these dialogs (`i18n/translation/en.json:1-34`) and no sidebar dialog imports
`useTranslation`; therefore no translated lookup is required.

---

## 15. Open Questions

1. **Visual language drift.** `ImportCollection/FileTab.js` uses raw Tailwind greys
   (`gray-200/700`, `blue-500`, `bg-blue-50`) instead of theme tokens
   (`FileTab.js:231-265`); `FullscreenLoader` similarly uses `zinc` and `white/80`
   (`FullscreenLoader/index.js:31-37`). A Go port must decide whether to map these to
   tokens (`border.border1/border2`, `input.focusBorder`, `requestTabPanel.responsePending`)
   or reproduce the literal greys. Flagged, not resolved.
2. **`refs/new_request_modal_bruno.png` shows a `URL` editor using a plain textbox look**
   with the method dropdown attached on the left; source uses `SingleLineEditor` (CodeMirror)
   (`NewRequest/index.js:509-533`). A Go port should render a text field with vars
   autocomplete; appearance may differ slightly from a CodeMirror input.
3. **`refs/new_folder_modal_bruno.png` input has no visible placeholder**; source agrees
   (no `placeholder` prop, `NewFolder/index.js:93-109`).
4. **`refs/import_collection_bruno.png`** is the only Import capture; it does not show the
   OpenAPI/Postman rows. Those rows are specified from source only.
5. **Selection list internals** (`SelectionList`) are specified in `screens/runner.md`;
   only the row heights/visible counts are repeated here.
6. **`ImportSettingRow` label for the "on filesystem" toggle** uses sentence-case
   `Show Filesystem Name` in New Request/New Folder but `Show file format` (sentence case)
   in Import — inconsistent casing is source-accurate.
7. **`SkippedPathsWarning`** is out of scope of this file (`screens/import-export.md`).
