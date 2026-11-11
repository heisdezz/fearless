# Screen — Request tabs

Components: `components/RequestTabs` (tab bar, `RequestTab`, `ExampleTab`,
`CollectionHeader`, `RequestTabLoading`, `DraggableTab`, special tabs), `components/Tab`,
`components/Tabs` (shared pill tabs), and the tab-adjacent request creators
(`CreateUntitledRequest`, `CreateTransientRequest`, `SaveTransientRequest`).

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source. Values are exact; unverifiable items are marked
**inference** or listed in Open Questions.

Screenshots: `refs/folder_bruno.png` (collection header + tab strip),
`refs/collection_main_view_bruno.png`.

---

## 1. Layout overview

```text
┌ collection header (CollectionHeader) ─────────────────────────────────────────┐
│ [📁] Sandbox ▾   ⋮                 [Req|App|File]  Runner  Safe   ⋯  [No Env ▾] │  py-2 px-4
├ tab strip (RequestTabs) ───────────────────────────────────────────────────────┤
│ ‹  ┌ Collection ┐ ┌ due…x┐ ┌ GET users ┐  [+]   ›                              │
│    └────────────┘ └──────┘ └───────────┘                                      │
└────────────────────────────────────────────── 1px bottomBorder ───────────────┘
```

- The strip renders only when the active collection has at least one tab
  (`RequestTabs/index.js:98-100,125-201`).
- The strip is scoped to one collection: tabs are filtered by `activeTab.collectionUid`
  (`RequestTabs/index.js:28-29`; `makeSelectTabsForCollection`).
- `CollectionHeader` renders above the strip when an active collection exists
  (`RequestTabs/index.js:127-132`).
- The strip has a `1px` full-width line pinned to the bottom in
  `requestTabs.bottomBorder` (`#efefef` L / `#444444` D) behind the tabs (`RequestTabs/StyledWrapper.js:6-15`).

---

## 2. Tab bar dimensions (`components/RequestTabs/StyledWrapper.js`)

| Aspect | Value | Source |
| --- | --- | --- |
| Strip row layout | `flex items-center gap-2 pl-2` | `RequestTabs/index.js:133` |
| List padding | `0 3px` | `StyledWrapper.js:38` |
| List alignment | `display: flex; align-items: flex-end` | `:40-41` |
| Tab min / max width | `min-width: 80px`, `max-width: 180px` | `:52-53` |
| Tab vertical padding | `6px 0` | `:62` |
| Tab gap | `margin-right: 3px` (last child `4px`) | `:58,79-81` |
| Tab bottom margin | `3px` (active: `-2px`) | `:64,119` |
| Tab font | `0.8125rem` (13 px) | `:56` |
| Label padding | `px-2` on the tab container | `RequestTab/index.js:629` |
| Method label | `0.6875rem` (11 px), `letter-spacing: 0.02em` | `RequestTab/StyledWrapper.js:16-20` |
| Tab name | `0.8125rem`, ellipsis, `padding-right: 2px` (italic-safe) | `RequestTab/StyledWrapper.js:22-31` |
| Scroll container max width | `screenWidth − effectiveSidebarWidth − 150` | `RequestTabs/index.js:102-103,145` |
| Scroll behaviour | `overflow-x: auto; overflow-y: clip`, scrollbars hidden | `StyledWrapper.js:21-35` |
| Chevron scroll step | `120px`, `behavior: 'smooth'` | `RequestTabs/index.js:105-117` |
| Chevron button | `ActionIcon size="lg"` (28 px), `IconChevronLeft`/`IconChevronRight` `18px` stroke `1.5`, `margin-bottom: 3px`, `aria-label` `Left Chevron` / `Right Chevron`; hidden when no overflow | `:134-138,186-190`; `ui/ActionIcon/StyledWrapper.js` `sizeMap.lg = 28` |
| Overflow detection | `tabs.scrollWidth > scroller.clientWidth + 1` via `ResizeObserver` | `:54-68` |

Tab shape in the light/dark surface: inactive tabs sit on `requestTabs.bg` with
`border.radius.base` (`6px`) and transparent border; the active tab is a raised card
(`8px 8px 0 0`) that merges with the content surface (see §3).

---

## 3. Tab visual states

| State | Background | Border | Text | Source |
| --- | --- | --- | --- | --- |
| Inactive | `requestTabs.bg` (`#f6f6f6` L / `#26292b` D) | transparent, radius `6px` | `requestTabs.color` (`#343434` L / `#cccccc` D) | `StyledWrapper.js:72-77,59`; tokens `requestTabs.bg/color` |
| Active | `bg` (`#ffffff` L / `#1a1a1a` D) | `1px requestTabs.bottomBorder`, bottom border = `bg`, radius `8px 8px 0 0`, `z-index: 1`, `margin-bottom: -2px`, `padding-bottom: 12px` | inherits `requestTabs.color` | `StyledWrapper.js:113-120` |
| Active corner fillers | `::before` / `::after` `8 × 8` squares with `6px` inner radius and `3px` shadow in `bg`, bordered with `bottomBorder` | — | — | `:122-148` |
| Preview (ephemeral) | same as normal, label italic (`tab.preview ? 'italic' : ''`) | — | — | `RequestTab/index.js:678,384`; `ExampleTab/index.js:145` |
| Hover (tab) | no background change (only the close-gradient appears) | — | — | `StyledWrapper.js` has no `li:hover` background rule |
| Overflowing name | mask fade over last `12px` (idle) / `8px` (hover) | — | — | `:83-111` |
| Short tab (unused, "post mvp") | `32px` wide, icon `20px`, hover background `background.surface0` | radius `6px` | `text` | `:151-195` |

Close-button gradient (see §4) supplies the only hover feedback on the trailing edge.

---

## 4. Close button, draft indicator, gradient

`GradientCloseButton` (`RequestTab/GradientCloseButton/index.js:6-27`,
`StyledWrapper.js`):

| Aspect | Value | Source |
| --- | --- | --- |
| Container | absolute right, `width: 44px`, full height, `padding-right: 4px` | `:9-17` |
| Gradient | `linear-gradient(90deg, transparent 0%, <tab bg> 40%)`; active rows use `theme.bg` | `:19-35` (CSS vars set from `requestTabs.bg` / `bg`) |
| Visibility | `opacity: 0` by default; `opacity: 1` on `li:hover` (when closable) or when `.has-changes` | `:25-41` |
| Icon box | `22 × 22`, radius `border.radius.base` (`6px`), hover background `requestTabs.icon.hoverBg` (`#eaeaea` L / `#1e1e1e` D) | `:43-60`; tokens `requestTabs.icon.hoverBg` |
| Close icon | `CloseTabIcon` (solid ✕ path), `12 × 12`, colour `requestTabs.icon.color` (`#9b9b9b` L / `#9f9f9f` D), hover `requestTabs.icon.hoverColor` (`#343434` L / `#cccccc` D), transition `color 0.12s ease` | `:68-73`; `RequestTab/CloseTabIcon.js:1-10` |
| Draft indicator | filled circle SVG `8 × 8` (`width="8" height="16" viewBox="0 0 8 8"`, `r=3`), fill `theme.draftColor` = `#cc7b1b` in both themes; CSS size `8 × 8` | `RequestTab/DraftTabIcon.js:3-18`; `GradientCloseButton/StyledWrapper.js:75-78`; tokens `draftColor` |
| Draft vs close | draft icon shows when unfocused (`has-changes` and not hovering) or when the tab is not closable; close icon replaces it on hover for closable tabs | `:80-121` |
| Non-closable tabs | `.no-close` — no close icon; draft dot stays visible on hover | `:62-66,111-121` |
| Test ids | container `request-tab-close-icon` (closable) / `request-tab-draft-icon`; inner dot `tab-draft-icon` | `GradientCloseButton/index.js:14,16` |

Saving/in-flight state: there is no spinner inside a request tab; a tab whose item is not
yet resolvable shows the loading variant (`RequestTabLoading`, §8).

---

## 5. Tab content variants

| Variant | Structure | Source |
| --- | --- | --- |
| Request (HTTP/GraphQL/gRPC/WS) | `[METHOD] [name]` + gradient close | `RequestTab/index.js:692-729` |
| App | `IconAppWindow 14px` in place of the method, `aria-label="App"` | `:692-695` |
| Response example | `ExampleIcon 14px` in `requestTabs.example.iconColor` (`#666666` L / `#b0b0b0` D) + name, italic when preview | `ExampleTab/index.js:144-163`; `RequestTab/StyledWrapper.js:33-35` |
| Special tabs | `14px` icon + label, icon in `primary.text` (`.special-tab-icon`) | `RequestTab/SpecialTab.js:7-132`; `RequestTabs/StyledWrapper.js:199-201` |
| Mock response | `MockResponseTab` | `RequestTab/index.js:598-602` |
| Loading | name only, from the tab snapshot | `RequestTabLoading/index.js:42-51` |
| Not found | delayed 300 ms, then `IconAlertTriangle 18px` (`text-yellow-600` `#ca8a04`) + `Not Found` | `RequestTabNotFound.js:58-86` |

Method label colour: `getMethodColor` maps `METHOD` lower-cased against
`theme.request.methods` plus `theme.request` (so `grpc`, `ws`, `gql` also resolve)
(`RequestTab/index.js:166-172`). The uppercase label is rendered with that colour inline
(`:697`).

### 5.1 Special tab labels (verbatim)

| `tab.type` | Icon | Label | Source |
| --- | --- | --- | --- |
| `collection-settings` | `IconSettings` | `Collection` (or collection name in §5.2 context) | `SpecialTab.js:10-17` |
| `collection-overview` | `IconSettings` | `Overview` | `:18-25` |
| `folder-settings` | `IconFolder` | folder name or `Folder` | `:26-33` |
| `variables` | `IconVariable` | `Variables` | `:34-41` |
| `collection-runner` | `IconRun` | `Runner` | `:42-49` |
| `environment-settings` | `IconDatabase` | `Environments` | `:50-57` |
| `global-environment-settings` | `IconWorld` | `Global Environments` | `:58-65` |
| `preferences` | `IconSettings` | `Preferences` | `:66-73` |
| `workspaceOverview` | `IconHome` | `Overview` (not closable) | `:74-81`; `RequestTab/index.js:570-571` |
| `workspaceEnvironments` | `IconWorld` | `Environments` (not closable) | `:82-89`; `RequestTab/index.js:572-573` |
| `openapi-sync` | `OpenAPISyncIcon` | `OpenAPI` | `:90-97` |
| `openapi-spec` | `IconFileCode` | `API Spec` | `:98-105` |
| `api-spec` | `IconFileCode` | tab name or `API Spec` | `:106-113` |
| `mock-server` | `IconServer2` | tab name or `Mock Server` + `Beta` badge | `:114-122` |
| `changelog` | `IconConfetti` | `What's New` | `:123-130` |

`specialTabs` list: `RequestTab/index.js:199-215`. Deferred types (OpenAPI, mock server,
API spec) are listed for completeness only.

### 5.2 Which collection tab is shown

`CollectionHeader` shows `Workspace` and `Collections` group labels inside its switcher
(`CollectionHeader/index.js:603-652`); implementation detail of that header is out of
this file's scope (see `screens/app-shell.md`). Only the tab strip is specified here.

---

## 6. Overflow scrolling

| Behaviour | Detail | Source |
| --- | --- | --- |
| Chevrons | left/right appear only when the list overflows | `RequestTabs/index.js:51-68,134-138,186-190` |
| Scroll amount | `±120px` per click, smooth | `:105-117` |
| Locate active tab | `scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' })` on active tab change | `:70-75` |
| Per-tab overflow | each tab measures its own name (`scrollWidth > clientWidth`) and reports `has-overflow` for the mask fade | `RequestTab/index.js:96-122`; `RequestTabs/StyledWrapper.js:83-111` |

---

## 7. Interactions on a tab

| Input | Result | Source |
| --- | --- | --- |
| Single click | focus tab; clicking the already-active tab reveals it in the sidebar | `RequestTabs/index.js:84-96` |
| Double click | make the tab permanent (`makeTabPermanent`) — leaves preview style | `RequestTab/index.js:680`; `ExampleTab/index.js:147` |
| Middle click (`button === 1`) | close without prompt when no changes; opens `ConfirmRequestClose` when there are changes | `RequestTab/index.js:141-164,682-690`; `ExampleTab/index.js:79-89,149-157` |
| Close button | closes immediately when clean; otherwise confirm dialog | `RequestTab/index.js:717-729` |
| Right click | opens the tab context menu at the label rect | `RequestTab/index.js:134-138`; `RequestTabMenu` `:740-745,897-907` |
| Drag tab | reorders within the collection; both dragged and hovered tab fade to `opacity: 0` | `RequestTabs/DraggableTab.js:7-46` |
| Special tab close | collection/folder/env/api-spec settings ask before discarding drafts; others close directly | `RequestTab/index.js:174-197,343-371` |

---

## 8. Tab states reference

| State | Trigger | Visual | Source |
| --- | --- | --- | --- |
| Active | `tab.uid === activeTabUid` | raised card §3 | `RequestTabs/index.js:77-82`; `StyledWrapper.js:113-149` |
| Inactive/preview | `tab.preview` truthy | italic name | `RequestTab/index.js:678` |
| Unsaved changes | `hasRequestChanges(item)` | draft dot replaces ✕ while not hovered | `RequestTab/index.js:88`; `GradientCloseButton/StyledWrapper.js:90-99` |
| Loading | `tab.name && isItemsLoading` (collection mounting) | `RequestTabLoading`: name + close only | `RequestTab/index.js:604-626`; `RequestTabLoading/index.js:42-51` |
| Not found | item missing after 300 ms | warning icon + `Not Found` + draft dot | `RequestTabNotFound.js:58-86` |
| Example | `tab.type === 'response-example'` | example icon + name | `RequestTab/index.js:586-596` |
| Drag | dragging / hover during drag | `opacity: 0` | `DraggableTab.js:40` |

---

## 9. Tab context menu (`RequestTab/index.js:840-895`)

Opened by right-click. Order and labels verbatim:

| # | id | Label | Shortcut hint | Disabled when | Source |
| --- | --- | --- | --- | --- | --- |
| 1 | `new-request` | `New Request` | `newRequest` if the shortcut is effective | — | `:841-847` |
| 2 | `clone-request` | `Clone Request` | — | — | `:848-852` |
| 3 | `revert-changes` | `Revert Changes` | — | no draft on this tab | `:853-858` |
| 4 | `close` | `Close` | `closeTab` if this tab is active | — | `:859-865` |
| 5 | `close-others` | `Close Others` | — | only one tab | `:866-871` |
| 6 | `close-left` | `Close to the Left` | — | first tab | `:872-877` |
| 7 | `close-right` | `Close to the Right` | — | last tab | `:878-883` |
| 8 | `close-saved` | `Close Saved` | — | (none) | `:884-888` |
| 9 | `close-all` | `Close All` | `closeAllTabs` | — | `:889-894` |

Shortcut hints are omitted (empty string) when the shortcut would act on a different tab
(`:844-845,862-863`). `Close`, `Close Others`, `Close to the Left/Right`, `Close All`
silently save unsaved request drafts before closing (`:756-812`); `Close All` re-saves
api-spec drafts too (`:795-796`). `New Request` opens `NewRequest` targeted at the tab's
folder (`resolveNewRequestTarget`, `:329-334,373-379`).

---

## 10. Confirm-close dialogs

`ConfirmRequestClose` (`RequestTab/ConfirmRequestClose/index.js`),
`ConfirmFolderClose` (`ConfirmFolderClose/index.js`), `ConfirmCollectionClose`
(`ConfirmCollectionClose/index.js`). All three share identical chrome:

| Aspect | Value | Source |
| --- | --- | --- |
| Size / title | `md`, `Unsaved changes` | `ConfirmRequestClose/index.js:10-13` (same in the other two) |
| Escape / outside click | both disabled | `:15-16` |
| Fade timeout | `150 ms` | `:17` |
| Footer | hidden; buttons built in body | `:23` |
| Header line | `IconAlertTriangle 32px` (`text-yellow-600` `#ca8a04`) + `Hold on..` in `text-lg font-medium` | `:26-27` |
| Buttons | `Don't Save` (`danger`), `Cancel` (`secondary` `ghost`), `Save` (`primary`), `gap-2` | `:35-44` |

Copy per variant (verbatim):

| Variant | Body copy | Source |
| --- | --- | --- |
| `ConfirmRequestClose` | `You have unsaved changes in {type} {name}.` — `type` is `request` or `example`, name in `font-medium` | `:30`; `ExampleTab/index.js:121` |
| `ConfirmFolderClose` | `You have unsaved changes in {name} folder settings.` | `ConfirmFolderClose/index.js:30` |
| `ConfirmCollectionClose` | `You have unsaved changes in {name} collection settings.` | `ConfirmCollectionClose/index.js:30` |

`Don't Save` deletes the draft (and closes a WS connection for WS requests) before
closing; `Save` saves then closes; transient requests open `SaveTransientRequest` instead
(`RequestTab/index.js:630-673`).

Related flow: `Close All` / the `closeAllTabs` shortcut opens the save-all request prompt
(`providers/Hotkeys/index.js:170-190`) before closing.

---

## 11. Tab-adjacent creators

### 11.1 `CreateTransientRequest` (`components/CreateTransientRequest/index.js`)

Anchored to the right of the tab list (`RequestTabs/index.js:182-184`).

| Aspect | Value | Source |
| --- | --- | --- |
| Trigger | `ActionIcon size="lg"`, `IconPlus 18px` stroke `1.5`, `aria-label="New Transient Request"`, `margin-bottom: 3px` | `:159-169` |
| Left click | creates the collection's preset request type immediately | `:146-148` |
| Right click | opens the dropdown (`placement="bottom-end"`) | `:150-178` |
| Menu items | `HTTP` (`IconApi`), `GraphQL` (`IconBrandGraphql`), `gRPC` (`IconCode`), `WebSocket` (`IconPlugConnected`) — icons `16px` stroke `2` | `:179-202` |
| Naming | `generateTransientRequestName` → `Untitled N` (`utils/collections/index.js:2457-2469`) | `:45-46` |
| Defaults | HTTP `GET`, GraphQL `POST` with empty graphql body, WS `ws`, gRPC | `:42-124` |
| Errors | `An error occurred while adding the request` | `:59` etc. |

Menu item order here is HTTP, GraphQL, gRPC, WebSocket (different from `CreateUntitledRequest`).

### 11.2 `CreateUntitledRequest` (`components/CreateUntitledRequest/index.js`)

| Aspect | Value | Source |
| --- | --- | --- |
| Trigger | `ActionIcon size="sm"` (22 px) with `IconPlus 16px` stroke `2` | `:145-147` |
| Menu | `MenuDropdown`, `autoFocusFirstOption`, default placement `bottom` (configurable) | `:140-148` |
| Items | `HTTP`, `GraphQL`, `WebSocket`, `gRPC` (icons as above) | `:108-133` |
| Naming | `generateUniqueRequestName(collection, 'Untitled', itemUid)` | `:17` etc. |
| Success | `New request created!` | `:32` etc. |

Not mounted by `RequestTabs` at this commit; kept in the map because it is listed in the
checklist (`ui-map/overview.md:59`).

### 11.3 `SaveTransientRequest` (`components/SaveTransientRequest/index.js`)

| Aspect | Value | Source |
| --- | --- | --- |
| Modal | `sm`, title `Select Collection` (collection picking) or `Save Request`; `hideFooter` (custom body) | `:391-399` |
| Request name label | `Request name` | `:403-405` |
| Collections label | `Save to Collections` / `Select a collection to save to` | `:424` |
| Search placeholder | `Search for folder` | `:613` |
| Empty collections | `No Collections Yet` + `Collections help you organize your requests. Create your first one to save this request.` + button `New collection` | `:580-591` |
| Empty folders | `No folders found` (searching) / `No folders available` + button `New Folder` | `:771-781` |
| New folder block | label `New Folder name` / `New Folder name (in bruno)`, placeholder `Untitled new folder`, toggles `Show filesystem name` / `Hide filesystem name` | `:639-762` |
| New collection block | `Collection name` (`Enter collection name`), `Location` (`Select location`) + `Browse`, `File Format` | `:471-575` |
| Errors | `Request name is required`, `Folder name is required`, `Collection name is required`, `Location is required`; success `Request saved successfully` / `New folder created!` | `:193,306,353,361,271,321` |

The multi-modal container offers discard-all (`SaveTransientRequest/Container/index.js:25-35`).

---

## 12. Shared tab primitives

### 12.1 `components/Tabs` (`Tabs/index.js`, `Tabs/StyledWrapper.js`)

Pill-style tab list used inside panes (e.g. Script sub-tabs), not the request tab bar.
Included because the checklist routes both here.

| Aspect | Value | Source |
| --- | --- | --- |
| Trigger padding / radius | `8px`, `4px` | `Tabs/StyledWrapper.js:29-30` |
| Font | `0.75rem` (12 px) | `:31` |
| Gaps | `0.25rem` between triggers | `:11,19` |
| Inactive colour / hover bg | `tabs.secondary.inactive.color` (`#838383` L / `#aaa` D) / `tabs.secondary.inactive.bg` (`#f1f1f1` L / `#26292b` D) | `:36-41` |
| Active bg / colour | `tabs.secondary.active.bg` (`#eaeaea` L / `#26292b` D) / `tabs.secondary.active.color` | `:43-46` |
| Transition | `all 0.15s ease` | `:37` |
| Overflow trigger | `IconChevronsRight 16px` stroke `2`, padding `6px`, `aria-label="More tabs"`, items portalled to `document.body` | `Tabs/index.js:16,79-96`; `StyledWrapper.js:49-51` |
| Measurement | hidden clone row (`.tabs-list-measure`, `visibility: hidden`) decides which triggers overflow | `Tabs/index.js:54-75`; `StyledWrapper.js:14-22` |

### 12.2 `components/Tab` (`Tab/index.js`)

Single tab element: `role="tab"`, `aria-selected`, `tabIndex 0`, activates on click,
`Enter` or `Space` (`:10-15,18-30`). `count > 0` renders a `<sup>` count with `ml-1 font-medium`
(`:29`). Styling comes from the host (`.tab`, `.active`) — no local style file.

---

## 13. Shortcuts (tab-related, complete)

Default bindings from `providers/Hotkeys/keyMappings.js`; each may be overridden by
`preferences.keyBindings` (`keyMappings.js:213-236`).

| Action | Mac | Windows/Linux | Implemented by | Source |
| --- | --- | --- | --- | --- |
| `closeTab` | `⌘ W` | `Ctrl W` | active tab only; draft-aware | `keyMappings.js:7`; `RequestTab/index.js:236-280` |
| `closeAllTabs` | `⌘ ⇧ W` | `Ctrl ⇧ W` | opens save-all prompt | `keyMappings.js:8`; `Hotkeys/index.js:170-190` |
| `reopenLastClosedTab` | `⌘ ⇧ T` | `Ctrl ⇧ T` | current collection | `keyMappings.js:11`; `Hotkeys/index.js:192-207` |
| `switchToTabAtPosition` | `⌘ 1` … `⌘ 8` | `Ctrl 1` … `Ctrl 8` | collection-scoped | `keyMappings.js:12-25`; `Hotkeys/index.js:140-168` |
| `switchToLastTab` | `⌘ 9` | `Ctrl 9` | last tab in collection | `keyMappings.js:13` |
| `switchToPreviousTab` | `⇧ ⌘ [` | `⇧ Ctrl [` | wraps around | `keyMappings.js:14`; `Hotkeys/index.js:108-122` |
| `switchToNextTab` | `⇧ ⌘ ]` | `⇧ Ctrl ]` | wraps around | `keyMappings.js:15`; `Hotkeys/index.js:124-138` |
| `moveTabLeft` | `⌘ [` | `Ctrl [` | reorders tab | `keyMappings.js:16`; `Hotkeys/index.js:333-346` |
| `moveTabRight` | `⌘ ]` | `Ctrl ]` | reorders tab | `keyMappings.js:17`; `Hotkeys/index.js:348-361` |
| `save` | `⌘ S` | `Ctrl S` | tab-type aware save | `keyMappings.js:9`; `RequestTab/index.js:285-327` |
| `saveAllTabs` | `⌘ ⇧ S` | `Ctrl ⇧ S` | saves collection/folder/request drafts | `keyMappings.js:10`; `Hotkeys/index.js:209-265` |
| `newRequest` | `⌘ N` | `Ctrl N` | tab handler only when active tab and no sidebar focus | `keyMappings.js:58`; `RequestTab/index.js:336-341` |
| `editEnvironment` | `⌘ E` | `Ctrl E` | opens environment tab | `keyMappings.js:57`; `Hotkeys/index.js:69-93` |
| `openTerminal` | `⌘ T` | `Ctrl T` | context-aware (sidebar focus → folder; settings tab → path) | `keyMappings.js:78`; `Hotkeys/index.js:280-331` |

`closeTab` and `save` are bound inside each `RequestTab` but enabled only for the active tab
(`RequestTab/index.js:280,327`). `save` is a no-op for tabs without drafts
(`:317-325`).

Modifier display: `⌘ ⌃ ⌥ ⇧` on macOS; `Ctrl`, `Alt`, `Shift`, `Win` on Windows; `Super`
instead of `Win` on Linux (`keyMappings.js:100-119`).

---

## 14. Copy reference (i18n)

All tab labels above are literal JSX text. `src/i18n/translation/en.json` has no tab keys
(`i18n/translation/en.json:1-34`), and `RequestTabs` does not import `useTranslation`.

---

## 15. Interactions summary

1. Selecting a tab focuses it; selecting the already-active tab reveals it in the sidebar
   (`RequestTabs/index.js:84-96`).
2. Middle-click or ✕ closes; drafts route to the confirm dialog, transient requests route to
   `SaveTransientRequest` (`RequestTab/index.js:630-673,717-729`).
3. Dragging reorders tabs inside the active collection only (`DraggableTab.js:7-29`).
4. Overflow shows chevrons; clicking scrolls `120px` (`RequestTabs/index.js:105-117`).
5. Right-click menu offers close variants; multi-close silently saves drafts
   (`RequestTab/index.js:756-838`).
6. The strip disappears when the collection has no tabs (`RequestTabs/index.js:98-100`).

---

## 16. Open Questions

1. **Screenshot vs source tab styling.** `refs/folder_bruno.png` shows pill-shaped tabs
   floating with gaps on a dark surface; source renders `8px 8px 0 0` tabs that merge into
   the content surface with corner fillers (`StyledWrapper.js:113-149`). The capture may be
   from a different theme preset (e.g. cyberdream/dracula) where `bg` and `requestTabs.bg`
   differ more strongly. This file follows source; a visual diff is needed before locking
   the Go layout.
2. **Hover feedback.** Source gives no background change on tab hover (`StyledWrapper.js`
   has no `li:hover` rule); only the close gradient appears. If the Go port wants a hover
   tint, it must pick one explicitly — the design system has no `requestTabs.hoverBg` token.
3. **Preview tabs.** `tab.preview` drives italics only. No separate `preview` colour token
   exists; a Go port should not invent one.
4. **`CreateUntitledRequest`** is not mounted by `RequestTabs` at this commit (no import in
   the tree). Its menu order (HTTP, GraphQL, WebSocket, gRPC) therefore has no live
   placement; recorded for checklist completeness.
5. **`CollectionHeader` scope.** The collection header (workspace switcher, Runner, sandbox
   mode, environment selector) is specified in `screens/app-shell.md`; this file only states
   that it renders above the strip.
6. **Multiple collections in one strip.** Source scopes the strip to the active tab's
   collection (`RequestTabs/index.js:28-29`), while some earlier builds render every open
   collection as one continuous strip. `refs/folder_bruno.png` shows a single collection.
7. **`draftColor`** is `#cc7b1b` in both themes (tokens `draftColor`); if a preset overrides
   it, the dot follows the preset.
8. **Token drift.** `requestTabs.icon.color` resolves to `#9b9b9b` (L) / `#9f9f9f` (D) in
   `tokens.json`, while `design-system.md` §6.3 quotes `#999` for dark; `requestTabs.bg` is
   quoted as `#1e1e1e` in design-system but resolves to `#26292b`. This file uses the
   resolved `tokens.json` values.
