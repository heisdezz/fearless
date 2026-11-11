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

`Sidebar/ImportSettings/` exposes, as labelled rows with dropdowns:

| Option row | Choices (source components) |
| --- | --- |
| Folder arrangement | `FolderArrangementOption` |
| Request name source | `RequestNameSourceOption` |
| Auth placement | `AuthPlacementOption` |
| Check for spec updates | `CheckForSpecUpdatesOption` |
| Advanced | `ImportAdvancedSection` (collapsible) |

`ImportOptionsMenu` groups these; each row is an `ImportSettingRow` with an
`ImportSettingDropdown`. Collect exact labels/values from those components when
building (each mounts its own option list).

## Warnings & reports

| Component | Purpose |
| --- | --- |
| `SkippedPathsWarning` | warns that some files were skipped during import |
| `PostmanPackageReport` | summary after importing a Postman package |
| `ImportCollection/FullscreenLoader` | full-screen progress while importing |

## Export

Collection export lives in `screens/collection-settings.md` (Share) and the
collection row context menu (`Sidebar`, see `sidebar.md`).

## Open Questions

- Exact label strings for each import option are in the per-option components,
  not captured verbatim here; read them at build time.
- Bulk import (`BulkImportCollectionLocation`) reuses the same location picker.
