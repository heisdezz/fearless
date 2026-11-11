# Collection runner — `RunnerResults` + selection components

Status: VERIFIED against `bruno/packages/bruno-app/src/` at the commit in
`overview.md:8`. There is **no runner screenshot** in `refs/` — every value
below is read from source; `refs/*.png` do not cover this screen.

Files in scope: `components/RunnerResults` (with `RunnerFilter`, `RunnerTags`,
`RunConfigurationPanel`, `ResponsePane`), `components/SelectionList`,
`components/SelectionFooter`, `components/IncludeExcludeTags`. The result
viewer reuses `components/ResponsePane/QueryResponse` (see
`screens/response-viewer.md`).

The runner is the root tab body: `RequestTabPanel` renders
`<RunnerResults collection={…} />` (`components/RequestTabPanel/index.js:17`).
It has two mutually exclusive faces: **setup** (before/without results) and
**results**.

---

## 1. Layout tree — setup face

Shown when `runnerResult.items` is empty
(`components/RunnerResults/index.jsx:248-319`).

```text
StyledWrapper (pl-4, h-full)
└─ div.flex (h-full)
   ├─ div.w-1/2.pr-4                                  ← left column
   │  ├─ "Runner"                     IconRun 20/1.5, font-medium, mt-6
   │  ├─ "You have <b>N</b> request(s) in this collection. (Loading...)"
   │  ├─ [danger] "Requests in this collection are still loading."
   │  ├─ .runner-section-title "Timings"              mt-6
   │  ├─ .runner-section: <label>Delay between requests (ms)</label>
   │  │                 <input type=number placeholder="e.g. 5">      mt-2
   │  ├─ .runner-section-title "Filters"              mt-6
   │  ├─ .runner-section  <RunnerTags> = <IncludeExcludeTags>   mt-2 mb-6
   │  └─ div.flex.gap-2                              ← run controls
   │     ├─ Button[primary] "Run N Request(s)"        disabled when none selected / loading
   │     └─ Button[ghost]   "Reset"
   └─ div.run-config-panel.w-1/2.border-l            ← RunConfigurationPanel
```

Copy, verbatim:

| Text                                             | Condition                       | Source              |
| ------------------------------------------------ | ------------------------------- | ------------------- |
| `Runner`                                         | always                          | `index.jsx:253-256` |
| `You have {N} request(s) in this collection.`    | always                          | `:258`              |
| `(Loading...)`                                   | `areItemsLoading(collection)`   | `:259-263`          |
| `Requests in this collection are still loading.` | loading (`danger`)              | `:265`              |
| `Timings`                                        | always                          | `:268`              |
| `Delay between requests (ms)`                    | label                           | `:270`              |
| `e.g. 5`                                         | delay placeholder               | `:274`              |
| `Filters`                                        | always                          | `:286`              |
| `Run {N} Request{s}`                             | run button; plural uses `!== 1` | `:299`              |
| `Reset`                                          | always                          | `:302`              |

Dimensions: `.textbox` = `height:1.875rem`, `padding:.2rem .5rem`,
`font.size.sm`, `radius.sm`, `1px solid input.border`, focus border
`input.focusBorder` (`RunnerResults/StyledWrapper.js:4-27`). Space between
section title and body: `mt-2` (8 px). Left column `.pr-4` (16 px). The config
panel is separated by a `1px` left border coloured
`background.surface1` (`StyledWrapper.js:107-109`).

The whole setup body is `pl-4` (16 px) with no bottom scroll
(`index.jsx:250`).

---

## 2. Layout tree — results face

Shown once `runnerResult.items.length > 0` (`index.jsx:248`, `:321-620`).

```text
StyledWrapper (px-4 pb-4 flex flex-col flex-grow relative overflow-auto)
├─ .runner-toolbar (flex justify-between mb-4 pt-[14px] gap-4 min-w-0)
│  ├─ RunnerFilter  ".filter-bar"
│  │   ├─ "Filter by:"
│  │   └─ All 0 | Passed 0 | Failed 0 | Skipped 0        (or compact select)
│  └─ .runner-actions
│      [running/pending]  Button[filled danger][IconPlayerStop] "Cancel Execution"
│      [ended]            Button[filled secondary][IconRefresh] "Run Again"
│                         Button[filled secondary][IconReload]  "Reset"
├─ div.flex.gap-4.h-[calc(100vh_-_10rem)].overflow-hidden
│  ├─ div.flex.flex-col.w-1/2                       ← results list
│  │  ├─ "Tags:  <green>include, …</green>  <muted>exclude, …</muted>"   (if tags set)
│  │  ├─ [danger] runnerInfo.statusText
│  │  └─ div.overflow-y-auto.flex-1
│  │     └─ result row × N                          (see §3)
│  └─ div.flex.flex-1.w-[50%].overflow-y-auto       ← detail pane
│     ├─ [nothing selected] centred: IconExternalLink 64 + "Click on the status code to view the response"
│     └─ [selected] header row (name + status icon + × Close) + <ResponsePane item=…>
```

- Toolbar: `RunnerResults/index.jsx:324-377`; `pt-[14px]`, `mb-4`, `gap-4`
  (`:324`).
- List ↔ detail split: `w-1/2` each (`:379-383`, `:536`).
- List height is pinned: `h-[calc(100vh_-_10rem)]` (160 px reserved)
  (`:379`).
- List scrolls; auto-scrolls to the newest row until the user scrolls up
  (re-enabled within 15 px of the bottom) (`:133-159`).
- Tags summary line: `Tags:` at `text-xs`, include list
  `colors.text.green`, exclude list `colors.text.muted` (`:383-395`).
- `runnerInfo.statusText` rendered in `danger` (`:396-403`).

### 2.1 Empty detail pane

| Property    | Value                                                     | Source              |
| ----------- | --------------------------------------------------------- | ------------------- |
| Icon        | `IconExternalLink` 64/1.5, `colors.text.subtext0`, `mb-4` | `index.jsx:567-570` |
| Text        | `Click on the status code to view the response`           | `:571-573`          |
| Text colour | `colors.text.subtext1`                                    | `:571-573`          |

### 2.2 Selected detail header

- Left: `displayName` (`mr-2`) + one status icon at 20/1.5 —
  `IconCircleCheck` (`test-success`), `IconCircleX` (`test-failure`),
  `IconCircleOff` (`skipped-request`) (`index.jsx:540-551`).
- Right: close button `p-1 rounded hover-bg-surface`, `title="Close"`,
  `aria-label="Close response view"`, `IconX` 16/1.5 (`:553-560`).
- Below: the runner `ResponsePane` (`:562`), covered in §6.

---

## 3. Result rows

One row per request (`index.jsx:404-534`):

```text
div.item-path  (mt-2)                        data-testid=runner-result-item
├─ div.flex.items-center
│  ├─ status icon 20/1.5                     ← exactly one of the three
│  ├─ displayName  (mr-1 ml-2)               ← danger if failed, skipped-request if skipped
│  ├─ [running]  IconRefresh 18/1.5 animate-spin ml-1
│  ├─ [has status] "200 - OK"                 ← text-xs link, click → open detail
│  └─ [no status]  "(request failed)"         ← danger, text-xs, click → open
├─ [include tags matched]  "Tags: a, b"       (pl-7, text-xs, muted)
├─ [error]  item.error                        (error-message pl-8 pt-2 text-xs)
└─ ul.pl-8
   ├─ Pre-Request tests  (IconCheck/IconX 18/2 + description)
   ├─ Post-Response tests
   ├─ Tests
   └─ Assertions        ({lhsExpr}: {rhsExpr})
```

| Element                 | Spec                                                                                                          | Source                           |
| ----------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| Pass icon               | `IconCircleCheck` 20/1.5, `colors.text.green`                                                                 | `index.jsx:412-413`              |
| Fail icon               | `IconCircleX` 20/1.5, `colors.text.danger`                                                                    | `:418-419`                       |
| Skipped icon            | `IconCircleOff` 20/1.5, `colors.text.muted`                                                                   | `:415-416`                       |
| Pass predicate          | `status !== 'error'` **and** all four statuses `'pass'`                                                       | `:32-38`                         |
| Fail predicate          | `status === 'error'` or any of the four `'fail'`                                                              | `:40-46`                         |
| Row name                | `displayName` = relative folder path + name                                                                   | `:20-24`, `:112`                 |
| Status text             | `{status} - {statusText}`, `text-xs`, `textLink`                                                              | `:430-435`                       |
| Failed placeholder      | `(request failed)`                                                                                            | `:436-438`                       |
| In-flight spinner       | `IconRefresh` 18/1.5 `animate-spin`                                                                           | `:428`                           |
| Error line              | `pl-8 pt-2 text-xs`, `colors.text.danger` via `.error-message` (only when `status == 'error'`)                | `:446`, `StyledWrapper.js:69-75` |
| Test rows (pass)        | `IconCheck` 18/2 `mr-2`, `colors.text.green`                                                                  | `:450-466`                       |
| Test rows (fail)        | `IconX` 18/2 `mr-2`, `colors.text.danger`, then error in `error-message pl-8 text-xs`                         | `:462-464`                       |
| Assertion text          | `{lhsExpr}: {rhsExpr}`                                                                                        | `:509-511`                       |
| Row order inside `<ul>` | Pre-Request → Post-Response → Tests → Assertions (rendered in that order, `assertionResults` always rendered) | `:448-521`                       |

`testStatus` / `assertionStatus` / `preRequestTestStatus` /
`postResponseTestStatus` are derived per row with
`getTestStatus(results)` = `fail` if any result failed, else `pass`, and
**not recomputed** for rows whose status is `error`, `skipped` or `running`
(`index.jsx:115-120`, `:26-30`).

Rows are keyed by `item.uid` and listed in `runnerResult.items` order, which
is the run order (see §5).

---

## 4. Filter — `RunnerResults/RunnerFilter`

```text
.filter-bar (1px solid border.border0; radius.base; max-height 35px)
├─ .filter-label  "Filter by:"     (background.mantle, padding .5rem .75rem)
└─ .filter-buttons                  (gap 1.25rem, padding .5rem .75rem 0)
   ├─ All {{count}}      ── active: 2px underline tabs.active.border
   ├─ Passed {{count}}
   ├─ Failed {{count}}
   └─ Skipped {{count}}
```

| Property              | Value                                                                                                                                                                                          | Source                                                              |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Label                 | `Filter by:` (verbatim, incl. colon)                                                                                                                                                           | `RunnerFilter/index.jsx:29`                                         |
| Filter labels         | `All`, `Passed`, `Failed`, `Skipped`                                                                                                                                                           | `RunnerResults/index.jsx:49-70`                                     |
| Predicates            | All: everything · Passed: `allTestsPassed` · Failed: `anyTestFailed` · Skipped: `status === 'skipped'`                                                                                         | `index.jsx:49-70`                                                   |
| Counts                | recomputed from current `items`                                                                                                                                                                | `index.jsx:235-245`                                                 |
| Count chip            | `padding 2px 4.5px; radius 2px; 1px solid border.border0; background.surface0; font.size.xs; 500`                                                                                              | `RunnerFilter/StyledWrapper.js:150-161`                             |
| Inactive button       | 500, `colors.text.subtext0`, transparent bottom border                                                                                                                                         | `:87-94`                                                            |
| Active button         | `tabs.active.fontWeight` (400), `tabs.active.color`, 2 px bottom border `tabs.active.border`                                                                                                   | `:77-85`                                                            |
| Transition            | `color .15s ease, border-bottom-color .15s ease`                                                                                                                                               | `:68`                                                               |
| Focus ring            | `2px solid tabs.active.border`, offset 2 px                                                                                                                                                    | `:72-75`                                                            |
| Compact variant       | below the breakpoint the buttons hide and a `.filter-select` dropdown (`{label} {count} ⌄`, `IconChevronDown` 14/2) takes over, menu `placement="bottom-start"`, tick in `.filter-check` 14 px | `RunnerFilter/index.jsx:47-63`, `StyledWrapper.js:45-53`, `:97-140` |
| Result list filtering | filtering also narrows the per-row test lists (`resultFilter`) — Passed keeps passes, Failed keeps `fail`/`error`                                                                              | `index.jsx:128-131`                                                 |

Toolbar collapse levels are `['compact', 'tiny']` (`index.jsx:18`): `compact`
switches the filter to the dropdown; `tiny` additionally hides the action
button labels (icon-only, `padding-left/right .5rem`) (`StyledWrapper.js:147-157`).

---

## 5. Run controls & configuration

### 5.1 Run configuration panel — `RunConfigurationPanel`

```text
runner-config-panel (bg, height 100%, flex column, overflow hidden)
├─ .header (py .75rem px 1rem, background.mantle, border-bottom border.border0)
│  ├─ .counter   "{selected} of {enabled} selected"
│  └─ .actions   Button[ghost] "Select All" | "Deselect All"    Button[ghost] "Reset"
└─ .request-list
   └─ .requests-container (padding .5rem)
      └─ .request-item × N  (height 2.5rem, padding .5rem, radius 4px, mb .25rem)
         ├─ .drag-handle  IconGripVertical 16/1.5
         ├─ .checkbox-container > .checkbox (1rem × 1rem, radius 3px)
         ├─ .method  GET/POST/… | GRPC | WS | GQL   (mono, font.size.base, 500, min-width 3rem)
         └─ .request-name  {name}  <span.folder-path>{folderPath}</span>
```

| Element          | Spec                                                                                                                    | Source                                |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| Counter          | `{selectedItems.length} of {enabledCount} selected`                                                                     | `RunConfigurationPanel/index.jsx:401` |
| Select-all label | `Deselect All` when everything selected, else `Select All`                                                              | `:408`                                |
| Reset            | `Reset`, `title="Reset selection and order"`                                                                            | `:412-419`                            |
| Loading          | `Loading requests...` (muted, `font.size.base`, padding `.75rem`)                                                       | `:424-425`, `StyledWrapper.jsx:48-53` |
| Empty            | `No requests found in this collection`                                                                                  | `:427`                                |
| Row selected     | bg `background.mantle`, border `border.border0`, checkbox filled `primary.solid` with `bg`-coloured tick                | `StyledWrapper.jsx:73-85`             |
| Row disabled     | `opacity .4`, `pointer-events:none`, drag handle hidden, checkbox `border.border2` on `surface0`                        | `:87-108`                             |
| Row dragging     | `opacity .5`, `1px dashed sidebar.dragbar`, `scale(.98)`, `shadow.md`, `z-index 5`                                      | `:110-117`                            |
| Drop target      | 2 px bar above/below (`::before`/`::after`) at `dragAndDrop.border`                                                     | `:119-154`                            |
| Checkbox hover   | border `primary.solid`                                                                                                  | `:188-190`                            |
| Method colour    | `request.methods.{verb}`; GRPC `request.grpc`, WS `request.ws`, GQL `request.gql`; fallback `sidebar.muted`             | `:194-241`                            |
| Method text      | verb uppercased, truncated to 3 chars when longer than 5 (`OPTIONS` → `OPT`); GraphQL → `GQL`, gRPC → `GRPC`, WS → `WS` | `index.jsx:35-59`                     |
| Folder path      | `sidebar.muted`, `ml-0.5rem`                                                                                            | `StyledWrapper.jsx:250-254`           |
| Transition       | `transform, background-color, box-shadow .15s ease`                                                                     | `StyledWrapper.jsx:71`                |
| Scrollbar        | 6 px wide, thumb `console.scrollbarThumb`                                                                               | `:35-46`                              |

Behaviour:

- Default selection = every enabled request (`index.jsx:251-257`).
- A saved `runnerConfiguration.requestItemsOrder` re-orders the list, unknown
  uids appended after (`:225-247`).
- Tag filters disable rows that do not match, and deselect them
  (`:15-29`, `:265-298`); de-selection is remembered in
  `pendingReselectRef` so they come back when the tag is removed (`:272-288`).
- Dragging is `react-dnd` with a drag preview suppressed (`:126-128`);
  dropping persists the new order (`:319-334`).

Ordering summary: folders first (by name, then sequence), then requests by
`seq`, depth-first through folders (`index.jsx:185-216`); a saved order
overrides this and drag-and-drop edits that saved order.

### 5.2 Results-face run controls

| State                                                   | Buttons              | Icon                        | Colour/size                                                                                 | Source              |
| ------------------------------------------------------- | -------------------- | --------------------------- | ------------------------------------------------------------------------------------------- | ------------------- |
| Running (`status !== 'ended'` and has `cancelTokenUid`) | `Cancel Execution`   | `IconPlayerStop`            | `Button variant="filled" color="danger" size="sm"`, `title`/`aria-label` `Cancel Execution` | `index.jsx:331-346` |
| Ended                                                   | `Run Again`, `Reset` | `IconRefresh`, `IconReload` | `filled secondary sm`, titles/aria-labels `Run Again`, `Reset`                              | `:347-375`          |
| Idle (neither)                                          | none                 | —                           | —                                                                                           | `:376`              |

`Run Again` re-runs the saved configuration (folder uid, saved delay, saved
selected items) (`:197-217`); `Reset` clears stored exchanges, resets the
runner state and the delay (`:219-228`); `Cancel Execution` dispatches
`cancelRunnerExecution(cancelTokenUid)` (`:230-232`).

### 5.3 Tags — `RunnerTags` → `IncludeExcludeTags`

Filled with the unique tags of the collection (`RunnerTags/index.jsx:8-28`).

```text
IncludeExcludeTags (data-testid=runner-tags)
└─ .tags-columns (flex, gap 1rem)
   ├─ .tags-column  "Include Tags"  <TagPicker placeholder="e.g. Production ready">
   └─ .tags-column  "Exclude Tags"  <TagPicker placeholder="e.g. Work in progress">
```

| Property            | Value                                                                                                 | Source                                        |
| ------------------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| Column labels       | `Include Tags`, `Exclude Tags` (verbatim)                                                             | `IncludeExcludeTags/index.jsx:48`, `:60`      |
| Label style         | `font.size.sm`, 500, `text`                                                                           | `IncludeExcludeTags/StyledWrapper.js:19-24`   |
| Include placeholder | `e.g. Production ready`                                                                               | `index.jsx:56`                                |
| Exclude placeholder | `e.g. Work in progress`                                                                               | `index.jsx:68`                                |
| Columns             | `flex:1` each, `gap .25rem` inside, `1rem` between                                                    | `StyledWrapper.js:4-17`                       |
| Input               | padding `.375rem .5rem`, `font.size.sm`, `border.border1`, `radius.base`, focus `input.focusBorder`   | `components/TagPicker/StyledWrapper.js:13-31` |
| Tag chip            | `height 1.75rem`, `1px solid border.border0`, `radius.base`, `IconTag` + text + remove `IconX`        | `:47-91`                                      |
| Suggestions menu    | fixed, `max-height:10rem`, `dropdown.bg`, `border.border1`, `dropdown.shadow`                         | `:94-120`                                     |
| Validation errors   | `Tag does not exist` / `Tag is already in the include list` / `Tag is already in the exclude list`    | `IncludeExcludeTags/index.jsx:16-27`          |
| Name validation     | `Tags must only contain letters, numbers, spaces, "-", "_"`; duplicate → `Tag "{tag}" already exists` | `TagPicker/index.jsx:54-61`                   |
| Keyboard            | `Enter` add (or the highlighted suggestion), `↑`/`↓` move, `Esc` close                                | `TagPicker/index.jsx:86-114`                  |
| Added tag order     | sorted ascending                                                                                      | `IncludeExcludeTags/index.jsx:34`             |

The same tag set is shown on results (`Tags:` line, §2) and per row when the
row carries any include tag (`index.jsx:433-437`).

---

## 6. Runner detail pane — `RunnerResults/ResponsePane`

A reduced `ResponsePane`: plain tab row (not `ResponsiveTabs`) and no
format selector until inside the body tab.

```text
ResponsePane (flex column, h-full, overflow-auto)
├─ div.tabs  "Response" | "Headers" N | "Timeline" | "{Tests label}"
│            right: [ResponseErrorsIcon] StatusCode ResponseTime ResponseSize
└─ section (flex column, pt-3)
   ├─ ResponseAlertsSection
   └─ tab content
      ├─ Response  → QueryResponse (disableRunEventListener)
      ├─ Headers   → ResponseHeaders
      ├─ Timeline  → RunnerTimeline (no filter bar, no timestamps)
      └─ Tests     → TestResults
```

| Property         | Value                                                                                                                      | Source                                                           |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| Tabs             | `padding 6px 0`, inactive `colors.text.subtext0`, active `tabs.active.*` + 2 px underline, `margin-right tabs.marginRight` | `RunnerResults/ResponsePane/StyledWrapper.js:4-28`               |
| Headers count    | `<sup class="ml-1 font-medium">{headers.length}</sup>`                                                                     | `RunnerResults/ResponsePane/index.js:85-86`                      |
| Status seam      | `StatusCode` → `ResponseTime` → `ResponseSize`, right-aligned                                                              | `:99-104`                                                        |
| Skipped item     | `SkippedRequest` replaces the whole pane                                                                                   | `:60-66`                                                         |
| Empty body       | `(no body)` is **not** shown; `QueryResponse` renders its own empty editor                                                 | `:44-55`                                                         |
| Run send blocked | `disableRunEventListener` — in-editor run shortcuts do nothing                                                             | `:49`                                                            |
| Timeline         | `RunnerTimeline` (main + oauth2 + scripted, `hideTimestamp`)                                                               | `:61-70`, `components/ResponsePane/RunnerTimeline/index.js:5-64` |

---

## 7. `SelectionList` + `SelectionFooter`

`SelectionList` is the generic multi-select list (used by import flows). It
is **not** used by the runner itself (the runner uses
`RunConfigurationPanel`) but is part of this scope's contract.

```text
SelectionList (max-width 720px; width 100%)
├─ .selection-heading   "{title}"  "{count}"      (inline-flex, gap .375rem, mb .5rem)
└─ .selection-panel (border, radius.base, padding .5rem, flex column gap .75rem)
   ├─ .selection-toolbar
   │  ├─ .selection-search  [IconSearch 16/1.5] <input placeholder="Search {title}">
   │  └─ .selection-toggle  [checkbox] "Select all"
   └─ ul.selection-list (max-height = rows×height + gaps; overflow-y auto; scrollbar-gutter stable)
      ├─ [no items]     "{emptyMessage}"                  (italic, muted)
      ├─ [no matches]   "No matching {title.toLowerCase()} found"
      └─ li > label.selection-item  [checkbox]  title / description (2-line clamp)
└─ [showSelectedCount] SelectionFooter  "{n} of {total} selected"
```

| Property           | Value                                                                                                                                        | Source                                             |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Defaults           | `visibleRows=8`, `rowHeight=40`, `rowGap=4`, `emptyMessage='No items found'`, `maxWidth='720px'`                                             | `SelectionList/index.js:22-28`, `constants.js:1-2` |
| List height        | `rowHeight × rows + rowGap × (rows-1)` px; rows = `min(items.length, visibleRows)` or 1 when empty                                           | `index.js:47`, `StyledWrapper.js:4-9`              |
| Count badge        | `min-width/height 1.25rem`, `radius.base`, `surface0` on `border.border1` (dark: `workspace.button.bg` border, `overlay.overlay0` fill), 500 | `StyledWrapper.js:26-42`                           |
| Search placeholder | `Search {title}` when none passed                                                                                                            | `index.js:99`                                      |
| Select-all label   | `Select all` (lowercase “a”), indeterminate when partial                                                                                     | `index.js:102-111`, `:53-57`                       |
| Filtering          | fuzzy match (`fast-fuzzy`) over title + description                                                                                          | `index.js:33-40`                                   |
| Item row           | `grid-template-columns: 1.5rem minmax(0,1fr)`, `gap .375rem`, `padding .25rem 0`                                                             | `StyledWrapper.js:141-153`                         |
| Item title         | `font.size.base`, 600, ellipsis                                                                                                              | `:170-178`                                         |
| Item description   | `font.size.sm`, 500, `subtext1`, 2-line clamp, paths normalised to `/`                                                                       | `:180-192`, `index.js:10`, `:71`                   |
| Empty state        | muted, `font.size.sm`, italic, grid-aligned to the text column                                                                               | `StyledWrapper.js:194-214`                         |

`SelectionFooter` (`components/SelectionFooter/index.js:3-12`) is a styled
`div`: `colors.text.subtext2`, `font.size.base`, 500, `line-height 1.25rem`;
its `<span>` (the number) is `primary.solid`. Used as
`<span>{n}</span> of {total} selected` (`SelectionList/index.js:147-151`).

---

## 8. State matrix

| State                     | Face    | Key visuals                                            |
| ------------------------- | ------- | ------------------------------------------------------ |
| Collection still mounting | setup   | `(Loading...)`, danger loading note, run disabled      |
| No results yet            | setup   | config panel + tag filters + delay                     |
| Running                   | results | in-row spinners, `Cancel Execution` (danger filled)    |
| Ended, all pass           | results | `IconCircleCheck`, green test rows                     |
| Ended, some fail          | results | `IconCircleX`, red rows with muted error lines         |
| Skipped rows              | results | `IconCircleOff`, muted name, `skipped-request`         |
| Request errored           | results | failure icon + `(request failed)` + `item.error`       |
| Filter with zero matches  | results | empty list, no message (list simply renders nothing)   |
| No selection              | results | `Click on the status code to view the response`        |
| Re-running                | results | list kept (`isReRunningRef`) while new items stream in |

---

## 9. Interactions

| Action                  | Trigger                                 | Effect                                                   | Source                                    |
| ----------------------- | --------------------------------------- | -------------------------------------------------------- | ----------------------------------------- |
| Run                     | `Run N Request(s)`                      | persists config, clears exchanges, `runCollectionFolder` | `index.jsx:188-195`                       |
| Reset (setup)           | `Reset` ghost                           | `resetCollectionRunner`, delay cleared                   | `:219-228`                                |
| Cancel                  | `Cancel Execution`                      | `cancelRunnerExecution`                                  | `:230-232`                                |
| Run Again               | results toolbar                         | re-run saved configuration                               | `:197-217`                                |
| Filter results          | chip / dropdown                         | client-side filter                                       | `:76`, `RunnerFilter/index.jsx:23-24`     |
| Open result             | click status code or `(request failed)` | selects the item into the detail pane                    | `index.jsx:423`, `:429`                   |
| Close detail            | `×`                                     | `setSelectedItem(null)`                                  | `:553-560`                                |
| Select/deselect request | checkbox / row click                    | toggles selection, persists order                        | `RunConfigurationPanel/index.jsx:336-361` |
| Select/Deselect all     | header button                           | all enabled uids                                         | `:363-379`                                |
| Reorder                 | drag handle                             | reorders and persists                                    | `:303-334`                                |
| Add/remove tag          | picker                                  | re-evaluates disabled rows                               | `:265-298`                                |
| Delay                   | numeric input                           | integer, stored in `runnerConfiguration.delay`           | `index.jsx:161-168`, `:192`               |

No keyboard shortcut is bound in the runner; the pickers use
`Enter`/`↑`/`↓`/`Esc` (§5.3).

---

## Open Questions

1. **`Run {N} Request{s}` with zero selected.** The button is disabled when
   `selectedRequestItems.length === 0` (`index.jsx:296`), so `Run 0 Request`
   is unreachable — harmless but odd copy.
2. **Filter counts vs. visible rows.** Counts are computed from all items
   (`index.jsx:235-240`) while the list shows only `filteredItems`; the
   `All` chip can therefore display a number larger than the rows on screen
   when another filter is active. Unverified whether this is intentional.
3. **Empty filtered list** has no empty state (`index.jsx:405-534` maps an
   empty array to nothing). A Go port may want `No matching results`.
4. **Test-row ordering** interleaves Pre-Request → Post-Response → Tests →
   Assertions without a visual section header; the same order in
   `TestResults` uses headings. Divergence is real
   (`index.jsx:448-521` vs `ResponsePane/TestResults/index.js:127-161`).
5. **`assertionResults` always rendered.** Runner rows render
   `assertionResults` without a null guard (`RunnerResults/index.jsx:509`),
   while the tests panel hides the section when empty
   (`ResponsePane/TestResults/index.js:57`). A missing array is filtered to
   `[]` by `filterTestResults` (`RunnerResults/index.jsx:128-131`), so it is
   safe but produces no empty-state feedback.
6. **`SelectionFooter` is styled-components + exported as a component**, not
   a plain footer; its props pass straight through to a `div`.
7. **Runner detail `ResponsePane` uses a raw tab list**, so tabs never
   overflow into a dropdown (unlike the main pane). The `Headers` sup uses
   `headers.length` and therefore shows no count when headers arrive as a
   plain object (`RunnerResults/ResponsePane/index.js:33`, `:86`).
