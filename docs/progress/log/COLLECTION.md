# COLLECTION — Tree loading, watching, and reconcile

Owner history: antigravity (claimed 2026-11-10T23:28Z, moved to InReview 2026-11-10T23:55Z)

## Entries

### 2026-11-10T23:28Z — antigravity

- Did: Claimed `COLLECTION`. Reviewed `docs/flows/collection-lifecycle.md`, `docs/data/collection-format.md`, `packages.md` acceptance criteria, and `internal/core/model/collection.go`.
- Tests: not run yet.
- Notes: Implementing collection tree loading (`LoadCollection`), `seq` sorting, atomic save / update, file watching with self-write suppression (`fsnotify` or polling with suppress window), external edit merge without losing active drafts, UID remapping on rename, and 1,000-request load benchmark in `internal/core/collection/`.

### 2026-11-10T23:55Z — antigravity

- Did: Implemented collection tree loading, ordering, atomic persistence, self-write suppressed watching, and external edit reconciliation:
  - `uid.go`: `GenerateUID` computing deterministic, stable SHA-256 path hashes so items survive reloads across restarts.
  - `sort.go`: `SortByNameThenSequence` and in-place `SortItems` implementing Bruno's exact ordering rules (valid positive sequence insertion at `seq - 1`, alphabetical fallback, sequence collision preservation).
  - `tree.go`: `LoadCollection` / `Load` auto-detecting dialect format (`format.Detect`), parsing `bruno.json` or `opencollection.yml`, root config, `environments/`, and recursively scanning folders and request items while respecting ignore patterns.
  - `save.go`: `SaveItem` promoting in-progress `item.Draft` to `item.Request` and clearing `Draft`, atomic writes via `persistence.WriteAtomic` with self-write suppression. Also implemented `SaveEnvironment`, `SaveCollectionConfig`, `ResequenceItems`, and `DeleteItem`.
  - `merge.go`: `Reconcile` merging fresh disk reads into in-memory collections, ensuring active user drafts are preserved across external reloads, deleted files remove items, renamed files remap UIDs without losing drafts, and active environment selections are retained.
  - `watcher.go`: `Watcher` providing recursive directory monitoring via `github.com/fsnotify/fsnotify`, self-write suppression (`SuppressPath` / `IsSuppressed`), temporary file filtering, and debounced change notifications.
  - `collection_test.go`: 12 test functions validating basic corpus loading, sort ordering, draft preservation across external edits, deletion removal, rename remapping, draft promotion on save, self-write suppression, and error handling.
  - `benchmark_test.go`: Benchmark generating 1,000 requests across 10 folders.
- Benchmark:
  - `Test1000RequestCollectionLoadTime`: 1,000 requests loaded in **83.6 ms** (SLA: < 2 s; ~24x faster than SLA).
  - `BenchmarkLoad1000Requests`: **22.0 ms/op** (4,621,602 B/op, 40,204 allocs/op).
- Tests: `go test -race -v -cover ./internal/core/collection` -> PASS (83.2% statement coverage). `go vet ./internal/core/collection/...` -> clean. `gofmt -l` -> clean. `./scripts/check-deps.sh` -> dependency-policy: ok.
- Notes: Ready for review. Meets all acceptance criteria in `packages.md`.
