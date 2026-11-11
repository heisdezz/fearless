# CMD-SETTINGS — Settings and Preferences Commands

Owner history: antigravity-2 (claimed 2026-11-11T01:02Z, moved to InReview 2026-11-11T01:08Z)

## Entries

### 2026-11-11T01:02Z — antigravity-2

- Did: Claimed `CMD-SETTINGS`. Reviewed `packages.md` acceptance criteria, `ownership.md` owned paths (`internal/app/commands/preferences*.go`, `save_preferences.go`, `oauth_cache.go`), `docs/ui/settings.md`, `internal/core/model/settings.go`, `internal/core/model/state.go`, `internal/core/events/`, and `internal/core/persistence/`.
- Tests: not run yet.
- Notes: Scope covers preferences loading and persistence using atomic writes, schema migrations, updating workspace state in `bus.Store`, OAuth2 token cache management/clearing, and emitting relevant events (`CredentialsUpdated`, etc.).

### 2026-11-11T01:08Z — antigravity-2

- Did: Implemented settings and OAuth cache commands in `internal/app/commands/`:
  - `preferences.go`: Defined consumer interfaces (`PreferencesSaver`, `PreferencesLoader`, `OAuthCacheCleaner`) and command payloads (`SavePreferencesCmd`, `LoadPreferencesCmd`, `ClearOAuthCacheCmd`). Implemented `PreferencesService` and Execute-based command adapters. Implemented `MigrateSettings` upgrading legacy/v0 settings (migrating proxy configs and ensuring valid defaults for redirects, zoom, fonts, theme, etc.) and `LoadPreferences` with atomic writeback of upgraded schemas. Implemented automatic corrupt file quarantining via `persistence.QuarantineCorrupt`, publishing `events.ErrorOccurred` with `events.ScopeSettings`, and restoring clean defaults.
  - `save_preferences.go`: Implemented `SavePreferences` with input sanitization/bounds clamping (Zoom, MaxRedirects, MaxResponseBytes, InlineBodyBytes, FontSize, TabSize, Theme, Language, HistoryRetainDays), atomic disk persistence via `persistence.WriteAtomic`, and live workspace state updates in `bus.Store`. Idempotent on repeated calls.
  - `oauth_cache.go`: Defined `OAuthTokenCache` interface with `MemoryOAuthTokenCache` and disk-persisted `FileOAuthTokenCache`. Implemented `ClearOAuth2Cache` (clearing individual credentials IDs or purging all cached tokens, emitting `events.CredentialsUpdated`), token accessors (`GetToken`, `SetToken`, `DeleteToken`), and `TokenStore` function meeting `auth.Deps.TokenStore`.
  - `preferences_test.go`: 10 comprehensive tests covering default initialization, custom saving and reloading, idempotency, bounds sanitization, legacy schema migration with auto-writeback, corrupted file quarantine and error event emission, in-memory and file-backed OAuth caching with live token clearing events, context cancellations, consumer interface adapters, and concurrent multi-goroutine access under race detector.
- Tests: `go test -count=1 -race -cover ./internal/app/commands/...` -> PASS (**85.9% statement coverage**, race-clean). `go vet ./internal/app/commands/...` -> clean. `gofmt -l ./internal/app/commands/` -> clean. `bash scripts/check-deps.sh` -> `dependency-policy: ok`.
- Notes: Ready for review. Completes Option A scope (`CMD-COLLECTION`, `CMD-ENV`, `CMD-SETTINGS`).
