# FORMAT-CODE — Codec interface and detection

Owner history: antigravity (claimed 2026-11-10T22:30Z, moved to InReview 2026-11-10T22:40Z)

## Entries

### 2026-11-10T22:30Z — antigravity

- Did: Claimed `FORMAT-CODE`. Reviewing `interface-design.md` §7, `collection-format.md`, and acceptance criteria.
- Tests: not run yet.
- Notes: Implementing `Codec` interface in `internal/core/format/codec.go`, registry in `internal/core/format/registry.go`, format detection in `internal/core/format/detect.go`.

### 2026-11-10T22:40Z — antigravity

- Did: Implemented format codec contract, registry, and dialect detection:
  - `codec.go`: Defined `Codec` interface (`Name`, `Detect`, `ParseRequest`, `WriteRequest`, `ParseCollection`, `WriteCollection`, `ParseFolder`, `WriteFolder`, `ParseEnvironment`, `WriteEnvironment`) plus sentinel errors `ErrUnknownFormat` and `ErrNoCollection`.
  - `registry.go`: Thread-safe registry (`Register`, `Get`, `For`, `Registered`, `Reset`) supporting dynamic codec registration with duplicate and nil guards.
  - `detect.go`: Deterministic format detector (`Detect`, `DetectFormat`, `FileExists`, `DetectFile`). Tested deterministic precedence (`FormatBru` prioritised over `FormatYML` if both exist in same root directory).
  - `detect_test.go`: Unit tests for registry, panic handling, dialect detection for bru and yml, deterministic resolution for multi-format directory, error propagation, dynamic extension without modifying the format package, and concurrent race-free access.
- Tests: `go test -race -v -cover ./internal/core/format` -> PASS (98.4% statement coverage). `go vet ./internal/core/format` -> clean. `gofmt -l` -> clean.
- Notes: Ready for review. `FORMAT-BRU` and `FORMAT-YAML` can now implement `format.Codec` and call `format.Register` in their `init()` functions.
