# Screen — Vars, Script, Assert, Tests, Docs, Settings

Bodies of the remaining request sub-tabs (tab strip itself: `request-pane.md` §3).

All paths are relative to `bruno/packages/bruno-app/src/`.

Status: VERIFIED against source, except Open Questions #2 (`ModeSwitch` active style).

---

## 1. Layout overview

```text
Vars        ┌ Pre Request ────────────────────────────────┐   EditableTable Name|Value|Description
            └ Post Response ──────────────────────────────┘   EditableTable Name|Expr (?)|Description
Script      [ Pre Request | Post Response ]   ← pill sub-tabs, overflow » if needed
             full-height JS CodeEditor
Assert      EditableTable  Expr | Operator | Value | Description
Tests       full-height JS CodeEditor
Docs        preview (rich text)  ── or ──  edit mode: MenuBar + [Rich Text | Markdown] + editor
Settings    "Configure request settings for this item."  → Tags → toggles → Max Redirects → Timeout
```

Vertical rhythm inside the tab: each tab starts immediately below the strip
(`mt-4` between strip and body — `RequestPane/HttpRequestPane/index.js:179`);
`Vars` uses `mb-3` above each section title (`RequestPane/Vars/index.js:17,21`).

---

## 2. Vars tab — `RequestPane/Vars/index.js`

Two stacked key/value tables, each under a `text-xs` title in
`colors.text.subtext0` (`Vars/StyledWrapper.js:4-6`):

| Section title (verbatim) | Table                   | Value column                      | Source                |
| ------------------------ | ----------------------- | --------------------------------- | --------------------- |
| `Pre Request`            | request-local variables | `Value` (with data-type selector) | `Vars/index.js:17-19` |
| `Post Response`          | response-time variables | `Expr` + InfoTip                  | `Vars/index.js:21-23` |

| Aspect                | Detail                                                                                           | Source                           |
| --------------------- | ------------------------------------------------------------------------------------------------ | -------------------------------- |
| Columns               | `Name` (`20%`, sortable), `Value`/`Expr`, `Description` (`25%`)                                  | `Vars/VarsTable/index.js:79-140` |
| Name placeholder      | `Name`                                                                                           | `:86`                            |
| Value placeholder     | `Value` (Pre Request) / `Expr` (Post Response)                                                   | `:97,110`                        |
| Value editor          | `MultiLineEditor`, `{{` autocomplete                                                             | `:99-115`                        |
| Data-type selector    | only in Pre Request, hidden on the empty add-row                                                 | `:117-133`                       |
| Expr help tooltip     | `You can write any valid JS expression here`                                                     | `:93`                            |
| Row order             | reorderable; sortable by name                                                                    | `:151-156`                       |
| Empty add-row present | yes                                                                                              | `EditableTable/index.js:265-289` |
| Name validation       | `Variable contains invalid characters. Must only contain alphanumeric characters, "-", "_", "."` | `Vars/VarsTable/index.js:66-70`  |

Screenshot: `refs/vars_bruno.png`.

---

## 3. Script tab — `RequestPane/Script/index.js`

**Nested pill sub-tabs** (`components/Tabs`):

| Order | value           | Label           | Indicator                                                                                |
| ----- | --------------- | --------------- | ---------------------------------------------------------------------------------------- |
| 1     | `pre-request`   | `Pre Request`   | `StatusDot` when script non-empty; error variant when `preRequestScriptErrorMessage` set |
| 2     | `post-response` | `Post Response` | same, from `postResponseScriptErrorMessage`                                              |

Sources: `RequestPane/Script/index.js:121-136`.

Pill styling (`components/Tabs/StyledWrapper.js`):

| Property                   | Value                                                                                                 | Source                                             |
| -------------------------- | ----------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Padding                    | `8px`                                                                                                 | `:36`                                              |
| Radius                     | `4px`                                                                                                 | `:35`                                              |
| Font                       | `0.75rem` (12px)                                                                                      | `:37`                                              |
| Gap between pills          | `0.25rem`                                                                                             | `:9`                                               |
| Inactive colour / hover bg | `tabs.secondary.inactive.color` (`#838383` L / `#aaa` D) / `.inactive.bg` (`#f1f1f1` L / `#26292b` D) | `:38-47`                                           |
| Active bg / colour         | `tabs.secondary.active.bg` (`#eaeaea` L / `#26292b` D) / `.active.color`                              | `:46-50`                                           |
| Overflow trigger           | `IconChevronsRight` 16px, `padding: 6px`, `aria-label="More tabs"`                                    | `Tabs/index.js:16,88-98`, `StyledWrapper.js:52-54` |

Editor: `CodeEditor` in `javascript` mode, `showHintsFor` `['req','bru']`
(pre-request) and `['req','res','bru']` (post-response)
(`Script/index.js:143-157,161-175`). Error lines are jumped to automatically
(`useFocusErrorLine`, `:63-76`).

Default sub-tab on first open: `getActiveScriptTab(scriptPaneTab, requestScript)`
returns the previously chosen sub-tab, else **`pre-request` when the request
script is non-empty**, else **`post-response`** (`utils/tabs:15-19`, used at
`RequestPane/Script/index.js:30`).

Screenshot: `refs/scripts_bruno.png` (collection-level equivalent UI).

---

## 4. Assert tab — `RequestPane/Assertions/index.js`

| Column        | Width   | Content                                          | Source                                        |
| ------------- | ------- | ------------------------------------------------ | --------------------------------------------- |
| `Expr`        | `20%`   | single-line editor, placeholder `Expr`           | `:89-95`                                      |
| `Operator`    | `120px` | `<select>` of operators                          | `:96-148`, `AssertionOperator/index.js:72-83` |
| `Value`       | `30%`   | single-line editor; disabled for unary operators | `:149-179`                                    |
| `Description` | `25%`   | multi-line                                       | `descriptionColumn.js:12-27`                  |

Empty add-row default value is `eq ` (with trailing space), so a new assertion
appears as _operator `equals`_ + empty value (`Assertions/index.js:181-186`).

Operator options, in order, with displayed label (`AssertionOperator/index.js:32-84`):

`eq` → `equals`, `neq` → `notEquals`, then raw ids: `gt`, `gte`, `lt`, `lte`,
`in`, `notIn`, `contains`, `notContains`, `length`, `matches`, `notMatches`,
`startsWith`, `endsWith`, `between`, `isEmpty`, `isNotEmpty`, `isNull`,
`isUndefined`, `isDefined`, `isTruthy`, `isFalsy`, `isJson`, `isNumber`,
`isString`, `isBoolean`, `isArray`.

Unary operators (Value cell disabled): `isEmpty`, `isNotEmpty`, `isNull`,
`isUndefined`, `isDefined`, `isTruthy`, `isFalsy`, `isJson`, `isNumber`,
`isString`, `isBoolean`, `isArray` (`Assertions/index.js:22-35`).

Row storage is a single string `<operator> <value>` parsed on read
(`parseAssertionOperator`, `:37-60`).

---

## 5. Tests tab — `RequestPane/Tests/index.js`

Single full-height `CodeEditor`, JS mode, hints `['req','res','bru']`
(`:49-58`). No chrome of its own (no heading, no empty-state text); the AI
assist button is the only right-side control (`HttpRequestPane:155-163`).
`test-script-editor` test id; errors jump to the line (`:37-43`).

---

## 6. Docs tab — `components/Documentation`

| Mode              | Chrome                                                                   | Source                                                     |
| ----------------- | ------------------------------------------------------------------------ | ---------------------------------------------------------- |
| Preview (default) | rendered rich text; **double-click** the body enters edit mode           | `Documentation/DocsEditor/index.js:123-127,176-181`        |
| Edit — rich text  | toolbar (`RichTextEditor.MenuBar`) + mode switch `Rich Text \| Markdown` | `DocsEditor/index.js:148-157`, `ModeSwitch/index.js:10-24` |
| Edit — Markdown   | `CodeEditor` in `gfm` mode, toolbar slot removed                         | `DocsEditor/index.js:159-174`                              |

Right-side control in the tab strip: `DocsAction` → button labelled `Edit`
(icon `IconEdit`) or `Preview` (icon `IconEye`), variant `ghost`, colour
`secondary`, size `sm`, `opacity-70` → `opacity-100` on hover
(`Documentation/DocsAction/index.js:13-26`). Screenshot:
`refs/docs_bruno.png` (shows the `Edit` button at the right of the strip).

Empty preview: no placeholder copy is passed for the request Docs tab
(`emptyPreviewContent` defaults to `''` — `DocsEditor/index.js:116`), so the
preview is blank.

Vertical rhythm: `.docs-tab-strip` row carries the toolbar + mode switch;
mode-switch container is licensed as a bordered segmented control — radius
`border.radius.base` (6px), `1px solid dropdown.separator` (`#e5e5e5` L /
`#333333` D), buttons flat (`border-radius: 0`, no border) with label colour
`colors.text.muted`, active button text `button2.color.primary.bg` and a
12% tint of `dropdown.selectedColor` behind it (`ModeSwitch/StyledWrapper.js:5-38`).
Labels `Rich Text` and `Markdown` (`ModeSwitch/index.js:17,23`).

---

## 7. Settings tab — `RequestPane/Settings/index.js`

Intro line, verbatim: `Configure request settings for this item.`
(`text-xs`, `mb-4`, muted — `:130`).

### 7.1 Tags (heading with `IconTag` 16px, `text-xs font-medium`)

`TagList` chip input (`:133-138`); placeholder `e.g. Production ready`
(`components/TagList/index.js:59`); inherited tags show tooltip
`Inherited from folder "X"` (`:114`).

### 7.2 Toggles

Shared `ToggleSelector` chrome: label `text-xs font-medium`, description
`text-xs` muted below it; switch size `medium` = container `h-5 w-9` (20×36px),
thumb `h-3 w-3` (12px), checked translate `translate-x-5`, track colour
`primary.solid` when on / `background.surface2` when off; thumb white;
`role="switch"` (`RequestPane/Settings/ToggleSelector/index.js:16-72`).

| Order | Label                               | Description (verbatim)                                                                            | Default       | Source                         |
| ----- | ----------------------------------- | ------------------------------------------------------------------------------------------------- | ------------- | ------------------------------ |
| 1     | `URL Encoding`                      | `Automatically encode query parameters in the URL`                                                | off (`false`) | `Settings/index.js:20,145-152` |
| 2     | `Automatically Follow Redirects`    | `Follow HTTP redirects automatically`                                                             | on (`true`)   | `:21,154-161`                  |
| 3     | `Forward Authorization on Redirect` | `Send Authorization and Proxy-Authorization headers when a redirect points to a different origin` | on (`true`)   | `:24,163-170`                  |
| 4     | `Enable App` (HTTP requests only)   | `Show the App tab and app view mode for this request`                                             | off           | `:173-183`                     |

Defaults object: `{ encodeUrl: false, followRedirects: true, maxRedirects: utils.DEFAULT_MAX_REDIRECTS, timeout: 'inherit', forwardAuthorizationHeader: true }`
(`Settings/index.js:17-23`).

### 7.3 Numeric inputs

| Control                    | Label           | Description                                            | Source     |
| -------------------------- | --------------- | ------------------------------------------------------ | ---------- |
| `SettingsInput`            | `Max Redirects` | `Set a limit for the number of redirects to follow`    | `:185-192` |
| `InheritableSettingsInput` | `Timeout (ms)`  | `Set maximum time to wait before aborting the request` | `:194-203` |

Timeout control states (`components/InheritableSettingsInput/index.js`):

| State                           | UI                                                                                                       |
| ------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Inherited (default `'inherit'`) | dropdown button `Inherit` (w-24 h-8, `input.bg`/`input.border`) with menu `Inherit`, `Custom` (`:32-57`) |
| Custom                          | text input (digits only, empty → `0`) with an `×` button, tooltip `Reset to inherit` (`:58-82`)          |

Both inputs also honour `Cmd/Ctrl+S` (save) and `Cmd/Ctrl+Enter` (send) while
focused (`Settings/index.js:112-120`).

---

## 8. States & interactions

| Tab      | Empty state                             | Indicator                                  | Notes                             |
| -------- | --------------------------------------- | ------------------------------------------ | --------------------------------- |
| Vars     | one empty row per table                 | `sup` count of enabled vars (pre + post)   | `HttpRequestPane/index.js:87,112` |
| Script   | empty JS editor                         | `StatusDot`, error variant on script error | `:113`                            |
| Assert   | one empty row, operator pre-set to `eq` | `sup` count                                | `:114`                            |
| Tests    | empty JS editor                         | `StatusDot`, error variant on test error   | `:115`                            |
| Docs     | blank preview                           | `StatusDot` when docs non-empty            | `:116`                            |
| Settings | —                                       | `StatusDot` when tags exist                | `:118`                            |

Shared table interactions (add row, drag reorder, checkbox, delete, column
resize) are specified in `request-params.md` §2.

`StatusDot` rendering: 10px dot, `opacity .8`, red (`text-red-500`) for the
error variant (`components/StatusDot/index.js:3-9`).

---

## 9. Verbatim copy index

`Pre Request`, `Post Response` (Vars sections **and** Script pills),
`Name`, `Value`, `Expr`, `Description`,
`You can write any valid JS expression here`,
`Variable contains invalid characters. Must only contain alphanumeric characters, "-", "_", "."`,
`equals`, `notEquals`,
`Edit`, `Preview`, `Rich Text`, `Markdown`, `More tabs`,
`Configure request settings for this item.`,
`URL Encoding`, `Automatically encode query parameters in the URL`,
`Automatically Follow Redirects`, `Follow HTTP redirects automatically`,
`Forward Authorization on Redirect`, `Send Authorization and Proxy-Authorization headers when a redirect points to a different origin`,
`Enable App`, `Show the App tab and app view mode for this request`,
`Max Redirects`, `Set a limit for the number of redirects to follow`,
`Timeout (ms)`, `Set maximum time to wait before aborting the request`,
`Inherit`, `Custom`, `Reset to inherit`,
`e.g. Production ready`, `Inherited from folder "X"`.

---

## 10. Open Questions

1. **Script pill tab pre-selection rule** is verified
   (`utils/tabs:15-19`): with no stored sub-tab, an existing Pre Request script
   opens `pre-request`, otherwise `post-response`.
2. **Assert operator labels** are raw ids (`gt`, `gte`, `startsWith` …) except
   `eq`/`neq`; source comments describe display names (`greater than`,
   `starts with` …) that are not used (`AssertionOperator/index.js:3-29` vs
   `:72-83`). Which form the Go port should show is a product decision.
3. **Assert add-row pre-filled value `'eq '`** means an untouched new row still
   round-trips a string; verify the persisted JSON shape matches
   (`value: 'eq '`).
4. **Docs empty state** has no explicit placeholder for requests; the
   collection Docs view passes a `documentationPlaceholder`
   (`CollectionSettings/Docs/index.js:79`) — the request tab does not. Confirm
   whether the Go port should mimic the blank preview or reuse a placeholder.
5. **Tags heading icon size** is `16px` but the heading text style is
   `text-xs font-medium`; the heading weight/size differs from the
   `ToggleSelector` labels. Confirm visual intent from a screenshot with tags
   present (none in `refs/`).
