# Screen — global search (⌘/Ctrl K)

Mode: **Operate**. A command-palette style modal over the whole app.

Source: `components/GlobalSearchModal/` (`index.js`). It searches collections,
folders, requests and documentation.

---

## Layout tree

```text
        ┌──────────────── command-k modal ─────────────────┐
        │ [🔍] Search collections, requests, or docs...  [×]│
        ├───────────────────────────────────────────────────┤
        │ [icon] Result title                               │
        │        collection · folder                        │
        │ [icon] Result title                               │
        │ …                                                 │
        └───────────────────────────────────────────────────┘
```

## Spec

- Input: `placeholder="Search collections, requests, or documentation..."`
  (`components/GlobalSearchModal/index.js:396`), 20 px search icon, clear button (`IconX` 16) when a query
  exists (`components/GlobalSearchModal/index.js:416-424`). Search input is a `role="combobox"` with
  `aria-autocomplete="list"` (`components/GlobalSearchModal/index.js:406-414`).
- Result icons by type (`components/GlobalSearchModal/index.js:357-363`), 18 px stroke 1.5:

  | Type | Icon |
  | --- | --- |
  | documentation | `IconBook` |
  | collection | `IconBox` |
  | folder | `IconFolder` |
  | request | `IconFileText` |

- Results list: `role="listbox"`, `aria-label="Search results"` (`components/GlobalSearchModal/index.js:428`).

## States & copy

| State | Copy |
| --- | --- |
| Empty query | no list; hint text (sr-only) "Search through collections, requests, folders, and documentation. Use arrow keys to navigate results and Enter to select." |
| Results found | `"{n} result(s) found"` announced via `aria-live` (`components/GlobalSearchModal/index.js:385-388`) |
| No results | **`No results found for "{query}".`** then "The item might not exist yet, or its collection isn't mounted. Press **Enter** here (or open it from the sidebar) to mount the collection automatically." (`components/GlobalSearchModal/index.js:434-440`) |
| Pressing Enter on a no-result query | mounts the collection automatically (per that hint copy) |

## Interactions

| Key | Action |
| --- | --- |
| `⌘/Ctrl K` | open (status bar Search button also opens, `components/StatusBar/index.js:46-51`) |
| Arrow ↑/↓ | move selection (`handleKeyNavigation`) |
| `Enter` | open selected result / mount collection |
| `Esc` | close |
| click `×` | clear query (`aria-label="Clear search query"`) |

## Open Questions

- Result row layout (secondary "collection · folder" line) is inferred from
  the icon map and rendering helpers; verify exact secondary-line order against
  `index.js` render functions when building.
