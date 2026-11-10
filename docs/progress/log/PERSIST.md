# PERSIST — Disk persistence and body spooling

Owner history: antigravity (claimed 2026-11-10T22:45Z, moved to InReview 2026-11-10T23:20Z)

## Entries

### 2026-11-10T22:45Z — antigravity
- Did: Claimed `PERSIST`. Reviewing `packages.md` acceptance criteria, `logic/persistence.md`, and `internal/core/model/result.go` (`BodyRef`).
- Tests: not run yet.
- Notes: Scope covers large body spooling (> 1MB), session temp directory cache, cache eviction policy (`max_cache_bytes`), and atomic file write/rename primitives.

### 2026-11-10T23:20Z — antigravity
- Did: Implemented disk persistence and response body spooling in `internal/core/persistence/`:
  - `atomic.go`: `WriteAtomic`, `WriteAtomicStream`, `QuarantineCorrupt`. Atomic writes write to temp file in same directory, sync (`fsync`), chmod, and atomic rename to guarantee no torn reads or partial writes.
  - `spool.go`: `Spooler`, `NewSpooler`, `Spool`, `SpoolBytes`, `Open`, `ReadBytes`, `Evict`, `Close`. Bodies <= 1 MB remain inline (`Inline != nil`, `Path == ""`). Bodies > 1 MB are spooled to session temp directory backing `model.BodyRef` (`Path != ""`, `Inline == nil`). Identical bodies are content-addressed by sha256 and deduplicated. Eviction policy enforces `MaxCacheBytes` (default 500 MB) by removing oldest files first.
  - `atomic_test.go` & `spool_test.go`: 12 test functions covering atomic writes, concurrent torn-read prevention with 5 readers + active writer, quarantine of corrupt files, inline vs spilled payloads, hash deduplication, synthetic flood eviction obeying `max_cache_bytes`, size cap and truncation, cleanup on close, and race-free concurrent spooling across multiple goroutines.
- Tests: `go test -race -v -cover ./internal/core/persistence` -> PASS (80.5% statement coverage). `go vet ./internal/core/persistence` -> clean. `gofmt -l` -> clean.
- Notes: Ready for review. `TRANSPORT` and `ENGINE` can now use `persistence.Spooler` to spool incoming responses into `model.BodyRef`.
