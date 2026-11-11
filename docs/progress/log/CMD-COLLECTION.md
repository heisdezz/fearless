# CMD-COLLECTION — Collection & Item Commands

Owner history: antigravity-2 (claimed 2026-11-11T00:05Z, moved to InReview 2026-11-11T00:15Z)

## Entries

### 2026-11-11T00:05Z — antigravity-2

- Did: Claimed `CMD-COLLECTION`. Reviewed `packages.md` acceptance criteria, `ownership.md` owned paths, `interface-design.md` §2 and §4, `internal/app/bus/`, `internal/core/collection/`, and `internal/core/events/`.
- Tests: not run yet.
- Notes: Scope covers collection and item commands: `LoadCollection`, `SaveItem`, `DeleteItem`, `MoveItem`, `NewItem`, `ReloadCollection`, and collection-level commands. Wires domain operations in `internal/core/collection/` to the application state actor (`bus.Store`) and event bus (`bus.Bus`), publishing typed `events.CollectionChanged` events and maintaining workspace tabs/state idempotently.

### 2026-11-11T00:15Z — antigravity-2

- Did: Implemented collection and item command handlers in `internal/app/commands/`:
  - `collection.go`: `CollectionService` coordinating state mutations and event publication over `bus.Store` and `bus.Bus`. Defined consumer interfaces from `interface-design.md` §2 (`CollectionLoader`, `CollectionReloader`, `ItemSaver`, `ItemDeleter`, `ItemCreator`, `ItemMover`, `CollectionCloser`) and serializable command payloads (`LoadCollectionCmd`, `ReloadCollectionCmd`, `SaveItemCmd`, `DeleteItemCmd`, `NewItemCmd`, `MoveItemCmd`, `CloseCollectionCmd`).
  - `load_collection.go`: `LoadCollection` with path resolution, idempotency checks against already-opened collections, `collection.NewWatcher` attachment with debounced external change dispatch, atomic workspace state update, and `events.CollectionChanged` (Self, Added) event emission. Also implemented `CloseCollection` which terminates the watcher, removes the collection and closes its tabs in `model.WorkspaceState`, and emits `events.CollectionChanged` (Self, Removed).
  - `reload_collection.go`: `ReloadCollection` reloading the tree from disk, reconciling via `collection.Reconcile` to preserve active uncommitted drafts, updating state, and publishing `events.CollectionChanged` (Self, Rescan).
  - `save_item.go`: `SaveItem` promoting in-progress draft edits to `item.Request` and clearing `item.Draft`, writing atomically to disk via `collection.SaveItem` with self-write suppression, updating state, and publishing `events.CollectionChanged` (Self, Modified). Idempotent on clean items.
  - `delete_item.go`: `DeleteItem` removing files or directory trees from disk via `collection.DeleteItem`, recursively removing from `targetCol.Items`, closing any active editor tabs associated with deleted item UIDs, updating active tab, and publishing `events.CollectionChanged` (Self, Removed). Idempotent when deleting already removed items.
  - `new_item.go`: `NewItem` creating new HTTP request or folder items on disk (supporting both `.bru` and `.yml` OpenCollection dialects), generating deterministic path-based UIDs, sorting items into the parent or root folder via `collection.SortItems`, updating workspace state, and publishing `events.CollectionChanged` (Self, Added).
  - `move_item.go`: `MoveItem` handling cross-folder moves via `os.Rename` with self-write suppression on both old and new paths, updating item path and regenerating UID, remapping any open editor tabs in workspace state, handling sequence re-ordering within folders, updating state, and publishing `events.CollectionChanged` (Self, Modified).
  - `collection_test.go`: 10 comprehensive test suites covering opening collections, reloading with draft preservation, saving items, creating requests and folders, cross-folder moves with tab remapping, item deletion with tab closure, collection closure, watcher external change notification, and validation/error edge cases.
- Tests: `go test -race -v -cover ./internal/app/commands` -> PASS (**85.7% statement coverage**). `go vet ./internal/app/commands/...` -> clean. `gofmt -l` -> clean. `./scripts/check-deps.sh` -> dependency-policy: ok.
- Notes: Ready for review. Meets all acceptance criteria in `packages.md`.
