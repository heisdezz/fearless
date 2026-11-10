# INFRA — Build, CI, dependency policy

Owner history: kilo (claimed 2026-11-10T20:16Z)

## Entries

### 2026-11-10T20:16Z — kilo
- Did: Claimed `INFRA`. Reviewing acceptance criteria and current repo state (`go.mod` pinned, minimal `main.go`, no CI/lint/tooling yet).
- Tests: not run yet.
- Notes: Will add CI workflow, lint/build scripts, dependency guard for core/app GUI imports, and version package within INFRA ownership.

### 2026-11-10T20:24Z — kilo
- Did: Added INFRA-owned files: `.github/workflows/ci.yml` (push/PR, ubuntu/macos/windows, CGO_ENABLED=0; runs dep guard, gofmt, go vet, staticcheck, govulncheck, build, test), `scripts/check-deps.sh` (forbids `github.com/egoist/mygo|github.com/lobehub/lobe-mygo|gioui.org|fyne.io` in `./core/...` and `./app/...` dep graphs; skips when packages absent), `.golangci.yml`, `Makefile`, `.editorconfig`, `internal/version/version.go`. Formatted with `gofmt -w .`. Confirmed pins: `mygo v0.8.4-0.20261108082913-9233663736ca`, `lobe-mygo v0.0.0-20261110162706-5e5fea5ef9e2` (D-004/D-005).
- Tests: `./scripts/check-deps.sh` -> ok (no core/app packages yet). `CGO_ENABLED=0 go vet ./...` -> fails on pre-existing `internal/core/model/auth.go:150:2 ClientSecret redeclared` (outside INFRA ownership; not modified). `CGO_ENABLED=0 go build ./...` / `go test ./...` not completed due to same compile/vet blocker in `internal/core/model`.
- Notes: INFRA acceptance items addressed (CI policy/gates incl. lobe-mygo guard, cross-OS matrix, no-CGO build/test steps, exact GUI pins). Repo currently has non-INFRA compile issue blocking full green vet/build/test; needs `MODEL` owner fix.
