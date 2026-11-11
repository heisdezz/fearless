# PLATFORM — OS concerns behind interfaces

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/platform/`
**Reference:** MyGo v0.8.4 `Secrets`, `WatchFiles`

## What the acceptance criterion actually asks for

> every OS-specific concern is behind an interface and has a headless fake for CI.

That is the deliverable: **interfaces and fakes**, with toolkit-free code in this
package. The MyGo implementations belong above this layer, where the GUI already
lives, because a package that wraps MyGo is a package that must import it — and
importing it here would defeat the purpose of having this package at all.

## The five concerns

| Interface | Purpose | MyGo support |
| --- | --- | --- |
| `Dialogs` | open/save file and folder prompts | **none found** |
| `Secrets` | OS credential store | `mygo.Secrets` |
| `Browser` | open a URL for OAuth2 | **none found** |
| `Loopback` | OAuth2 redirect receiver | not needed (pure Go) |
| `WatcherFactory` | file-change notifications | `mygo.WatchFiles` |
| `WindowState` | remember window geometry | not needed (file) |

### Gaps in MyGo to flag

- **No "open this URL in the system browser".** MyGo has `Application.OnOpenURL`,
  which *receives* URLs the OS hands to the app — the opposite direction. Opening
  a URL will therefore need a per-OS implementation of the `Browser` interface
  (`xdg-open` / `open` / `rundll32`) or an addition to MyGo. `Browser` is declared
  here so the OAuth2 flow can be written against it now.
- **No file pickers.** MyGo has no dialog API in the version pinned here, so
  `Dialogs` will need either a MyGo addition or a per-OS implementation.
  `openFile`/`saveFile` on the web backend exist; a native one does not.

## Design decisions

### Degrading defaults, not nils

`Services` fields are optional. Every accessor returns a no-op rather than nil:

- `DialogsOr()` → `NoDialogs`, which **cancels** every prompt. A headless runner
  that reaches a picker has a bug; surfacing it as a cancellation is better than
  a hang or a nil dereference.
- `SecretsOr()` → `UnavailableSecrets`, reporting
  `errors.Is(err, errors.ErrUnsupported)` and `Available() == false`.
- `BrowserOr()` / `WatchersOr()` → report `ErrUnsupported`.
- `WindowStateOr()` → `MemoryWindowState`, so a window still reopens at its last
  size within a session even when nothing persists it.

### `ErrUnsupported` is `errors.ErrUnsupported`

MyGo documents that every credential-store method fails with an error satisfying
`errors.Is(err, errors.ErrUnsupported)` when there is no keyring — which is the
normal state on a Linux server or minimal desktop. Aliasing the standard sentinel
means one check works for the app and for MyGo, instead of two parallel notions of
"unsupported".

Callers **must** handle it: refusing to start because a keyring is absent would
make the CLI runner unusable in CI.

### Cancellation is distinct from failure

`ErrCancelled` is separate from any error the platform reports, so "the user
dismissed the picker" never looks like "the filesystem said no". A test pins both
directions.

### Loopback is implemented, not stubbed

It is pure Go (`net/http` on `127.0.0.1`), so there is no reason to leave it to
the GUI layer, and the OAuth2 flow's correctness depends on details worth
testing:

- **`NewLoopbackAt` takes the path at construction.** The route is registered
  when the server starts, so a settable `Path` afterwards would leave the handler
  unreachable — a bug found while writing the test, not theorised.
- **A fixed port is honoured exactly**, because providers compare the redirect
  URI as a string. `RedirectURI()` uses `127.0.0.1` literally rather than
  `localhost`, which may resolve to IPv6 first and fail against an IPv4 listener.
- **Only the first callback counts.** A second request gets `409` and the first
  authorization code is what `Wait` returns, so a stray tab cannot replace the
  code being awaited.
- **`Wait` ends on context cancellation or listener close**, so an abandoned
  browser flow does not hang the caller forever.

### `Geometry.Valid`

A stored geometry can be corrupt or come from a monitor that is no longer
attached. A caller should check `Valid()` before applying it; non-positive
dimensions are the failure that matters.

### Watcher events are hints

MyGo documents that its events say what *changed*, not every step: a file written
twice is one write. That contract is restated on the interface so a caller
re-reads rather than assuming it saw each save.

## Verification

```
go vet ./internal/platform/                  # clean
go test -race -cover ./internal/platform/
ok  ...  coverage: 89.7% of statements
```

Coverage: every degraded default returning the documented value (six concerns,
including `Available()` and both directions of the cancellation/failure
distinction); `ErrUnsupported` matching the standard sentinel; the `Geometry`
validity table; `Query.Get` including multiple values and a nil query; an
injected fake being reached with its options intact; and eleven loopback tests
covering the happy path, an error redirect, a second request getting `409` with
the first code preserved, context cancellation, close unblocking `Wait`, a
rebound fixed port, a custom callback path, `Listen` returning the URI, the
factory plugging into `Services`, and an occupied port failing.

Not tested: the MyGo implementations, because none are delivered here.

## Open item

`scripts/check-deps.sh` does not list `internal/platform` as a layering root
(**B-010**), for the same reason as `internal/history` (**B-009**). The graph is
clean today; both are one-line additions owned by `INFRA`.

## Sources

- `mygo@v0.8.4-.../secrets.go` — `SecretsModule`, `Service`, `MaxSecretSize`,
  the `errors.ErrUnsupported` contract
- `mygo@v0.8.4-.../watch.go` — `WatchFiles`, `FileEvent`, the hint semantics
- `mygo@v0.8.4-.../app.go:550` — `Application.OnOpenURL`, which is inbound only
