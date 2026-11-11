# ADV-HISTORY — history store

**Agent:** zed-agent-1
**State:** InReview (store delivered; UI half blocked, see below)
**Package:** `internal/history/`
**Reference:** D-011 (retention), `flows/history-flow.md`

## Scope split

The package row covers "SQLite history store + UI". **Only the store is
delivered here.** The UI half (`internal/ui/screens/history/**`) depends on
`UI-SCREEN-RESP`, which is unimplemented, and on a presenters layer that does not
exist yet. Building a screen against no presenter is guesswork, so it stays
blocked rather than half-built.

## Why not SQLite

The row says SQLite. This uses **append-only JSONL, one file per collection** plus
an in-memory index, for three reasons:

1. **No driver is in the dependency graph.** Every Go SQLite driver is either CGO
   (`mattn/go-sqlite3`, against the Go-only, cross-compilable goal) or a large
   pure-Go translation (`modernc.org/sqlite`) that would be the heaviest
   dependency in the project. Neither is currently cached, so it would also need
   a network fetch and a `go.mod` change owned by `INFRA`.
2. **The access pattern does not need a query planner.** Rows are appended and
   read newest-first; the only deletion is a prefix prune, which is exactly what
   rewriting a file does.
3. **It is inspectable.** A user can read their history with `cat`, which matters
   for a privacy-sensitive feature; a SQLite file needs tooling.

If a future requirement needs ad-hoc queries or multi-process access, the
interface is small enough to swap the backend without touching callers.

## Entry shape

The metadata row D-011 specifies: timestamp, collection and item UIDs, item name
and path, environment, method, URL, status, duration, size, error, assertion
counts, and tags.

`BodyRef` is a **pointer**, because a body is usually absent: D-011 keeps bodies
only on failure or explicit save, and the spool owns the bytes. Storing the bytes
here would duplicate the spool's content-addressed storage.

`ItemName`, `ItemPath` and `Tags` are copied rather than referenced, so a row
still reads sensibly after the request is renamed or deleted.

### What counts as a failure

`Entry.Failed()` is `Error != "" || Status >= 400`. A 3xx is **not** a failure:
the transport follows redirects, so a final 3xx means it stopped early for
another reason, which the error field already records.

## Retention (D-011)

Both bounds are enforced, age first:

- **Age:** rows older than `RetainDays` (default 30) are dropped. A negative value
  disables the bound entirely, so a user who wants everything can have it.
- **Size:** rows are dropped oldest-first until the total is under `MaxBytes`
  (default 200 MB). The newest rows survive, because those are the ones a user is
  looking at.

Pruning runs inline on every write, and `Prune()` runs it explicitly for the
settings screen. An inline prune failure is swallowed **on purpose**: the row is
already durable, and returning an error would invite a retry that duplicates it.

## Write path

- Appends are serialised behind a mutex and only ever extend a file. No row is
  rewritten on the hot path.
- Reads take a read lock and copy rows out, so a query cannot observe a
  half-written entry.
- Rewrites (delete, clear, prune) go through a temporary file and `os.Rename`, so
  a crash mid-rewrite leaves the previous file intact rather than truncated.
- A malformed line at load is **skipped**. History is a log: a row truncated by a
  crash must not make the whole log unreadable, and refusing to start the
  application is a far worse outcome than losing that one row.

"Write never blocks the UI" is satisfied structurally: the write is a single
`write(2)` of one small line under a mutex held for microseconds, with no
fsync, no re-marshal of existing data, and no file scan. `TestConcurrentWritesDoNotCorrupt`
drives 200 concurrent writes and `TestReadsDuringWritesDoNotRace` runs queries
against a live writer, both under `-race`.

## Path safety

A collection UID becomes a filename, and UIDs come from files on disk. Path
separators and NUL are replaced, and `TestUnsafeCollectionUIDStaysInDirectory`
plants `../../evil` and asserts no file appears outside the store.

## Verification

```
go vet ./internal/history/                   # clean
go test -race -cover ./internal/history/
ok  ...  coverage: 87.5% of statements
```

Coverage: ID and timestamp assignment including collision within one second;
caller-supplied timestamps; newest-first ordering; every query filter
(collection, item, status class, failed, case-insensitive search over method/URL/
name, since/until, limit); limit truncating the tail; the failure classification
table; `Summarize`; persistence across reopen including `BodyRef`; malformed-line
tolerance; age pruning with a reopen proving the file changed; inline pruning on
write; disabled pruning; size pruning dropping oldest while keeping newest; delete
(including unknown ID) and its durability; clear by collection and in full;
200 concurrent writes with unique IDs on disk; concurrent reads against a writer;
all six methods reporting `ErrClosed`; a missing directory; path traversal
rejection; unknown-collection files; and the option defaults.

## Open item

`scripts/check-deps.sh` guards `internal/core`, `internal/app` and
`internal/ui/presenters` as layering roots. **`internal/history` is not a guarded
root**, so nothing mechanically prevents a future change from importing the GUI
into the store. Its dependency graph is clean today (`go list -deps` has no MyGo,
Gio or Fyne). Adding `./internal/history/...` to the roots array is a one-line
`INFRA` change; logged in `blockers.md` rather than edited here.

## Sources

- D-011 in `docs/progress/decisions.md` — retention by age and size, bodies on
  failure or explicit save
- `docs/flows/history-flow.md` — the lifecycle this store's entry shape mirrors
- `internal/core/persistence/spool.go` — `BodyRef` ownership
