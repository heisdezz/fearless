# Screen — import / export

Mode: **Operate**. Importing collections (Bruno, Postman, OpenAPI, Insomnia,
Postman env) and exporting one.

Sources: `components/Sidebar/ImportCollection/`,
`Sidebar/ImportCollectionLocation`, `Sidebar/ImportSettings/` (and its
sub-options), `Sidebar/BulkImportCollectionLocation`, `SkippedPathsWarning`,
`Sidebar/PostmanPackageReport`. Screenshot: `refs/side_bar_bruno.png`
(sidebar plus menu).

---

## Flow

```text
[pick file type] → [choose import file] → [ImportCollectionLocation: choose folder] → [ImportSettings: options] → import → [PostmanPackageReport / SkippedPathsWarning]
```

## Import settings options

`Sidebar/ImportSettings/` exposes labelled rows, each with a dropdown or a
switch (`ImportSettingRow`: label + info tip + description, `isSwitch` for
toggles; `ImportSettingDropdown` for choices). Verbatim:

| Component                   | Label                                      | Description (verbatim)                                                  | Control                                 | Options                                            |
| --------------------------- | ------------------------------------------ | ----------------------------------------------------------------------- | --------------------------------------- | -------------------------------------------------- |
| `FolderArrangementOption`   | **Folder arrangement**                     | Select whether to create folders according to the spec's paths or tags. | dropdown (`testId=grouping`)            | `Tags` (`tags`), `Paths` (`path`) — default `tags` |
| `RequestNameSourceOption`   | **Naming requests**                        | How requests in the generated collection are named.                     | dropdown (`testId=request-name-source`) | `Summary`, `operationId`, `Description`, `Path`    |
| `AuthPlacementOption`       | **Set authentication at collection level** | Requests and folders inherit it instead of getting their own copy.      | switch (`testId=auth-placement-toggle`) | on = `collection`, off = `request`                 |
| `CheckForSpecUpdatesOption` | **Check for Spec Updates**                 | Stay notified of spec changes and sync your collection with the spec.   | switch                                  | on/off                                             |
| `ImportAdvancedSection`     | (collapsible section)                      | —                                                                       | expander                                | —                                                  |

Info tips (shown from the row's info icon, verbatim):

- Naming requests: _If an operation doesn't have the selected field, Bruno
  falls back to summary, operationId, description, then method and path._
- Auth placement: _On import, the spec's security scheme is added to the
  collection. Operations with a different scheme keep their own auth._
- Check for Spec Updates, Swagger 2.0 variant: _OpenAPI Sync is not supported
  for Swagger 2.0 specs._

`ImportOptionsMenu` groups these rows.

## Warnings & reports

| Component                           | Purpose                                          |
| ----------------------------------- | ------------------------------------------------ |
| `SkippedPathsWarning`               | warns that some files were skipped during import |
| `PostmanPackageReport`              | summary after importing a Postman package        |
| `ImportCollection/FullscreenLoader` | full-screen progress while importing             |

## Export

Collection export lives in `screens/collection-settings.md` (Share) and the
collection row context menu (`Sidebar`, see `sidebar.md`).

## Open Questions

Resolved: the labels above are read verbatim from the per-option components.

- Bulk import (`BulkImportCollectionLocation`) reuses the same location picker.
