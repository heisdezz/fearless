# UI-DIALOGS

**Agent:** zed-agent-1
**Scope:** `internal/ui/dialogs/**`
**State:** InReview

Modal flows that sit above the screens: confirmations, the prompt-variable
form, import, and save-transient.

## Acceptance criterion

> `feedback.Modal` closes only after the action succeeds.

That sentence is the whole design. Lobe's `feedback.Modal`
(`lobe-mygo/pkg/feedback/modal.go`) already has the right shape: it returns
`true` in the frame OK is clicked and *leaves `*open` to the caller\*. So the
rule is enforced once, in the state machine, instead of being repeated in every
flow that shows a dialog.

## Split

The package has two halves that never import each other's concerns:

| Half          | Files                                 | Imports Lobe/MyGo?            | Tested                       |
| ------------- | ------------------------------------- | ----------------------------- | ---------------------------- |
| State machine | `dialogs.go`, `flows.go`, `prompt.go` | no (only `convert/*`, `vars`) | plain `go test`, no renderer |
| Views         | `view.go`                             | yes                           | headless `ui.Tester`         |

`internal/ui/dialogs/*_test.go` — `dialogs_test.go` (state machine),
`flows_test.go` (import/prompt/save flows), `view_test.go` (render + click
wiring), `import_test.go` (all four converters, labels, truncation).

## The state machine

`Queue` is the single owner of dialog state. One dialog at a time; opening a
second supersedes the first with `ErrSuperseded` rather than stacking modals.

```
PhaseClosed ──open──▶ PhaseOpen ──OK──▶ PhaseRunning ──raw nil──▶ PhaseClosed
                         ▲                   │
                         │                   └──raw error──▶ PhaseFailed
                         └──────────── SetValue clears the stale error ──┘
```

- `PhaseFailed` keeps the dialog **open** with the error visible, so the user can
  retry or cancel. `Busy()` is false in that phase, which is what makes a retry
  possible — the view binds OK's disabled state to `Busy()`.
- `PhaseRunning` makes `Busy()` true, so a double click cannot start a second
  action; the second `OK()` finds `Busy` and returns.
- `SetValue` is ignored while running, so late keystrokes cannot change the
  values an in-flight action already captured.
- `Cancel` during `PhaseRunning` closes the dialog, resolves the future with
  `ErrSuperseded`, cancels the action's context, and _discards_ the action's
  eventual result rather than reopening a closed dialog.

### Concurrency

Every field is guarded by `Queue.mu`. The action goroutine writes `phase` when it
finishes while the UI goroutine reads it to draw, so the lock is not optional.
`notify()` copies the listener under the lock and calls it with **no** lock held,
because a redraw reads the queue and a lock held across that read would deadlock.
`TestListenerMayReadQueue` pins that.

A mutex is not reentrant in Go, so `notify` must never be called from a method
that still holds the lock. Each mutator therefore unlocks before notifying; the
`*Locked` helpers (`busyLocked`, `closeLocked`, `supersedeLocked`, `copyValues`)
exist for that reason.

## The flows

### Prompt variables

`{{?label}}` is Bruno's "ask me before sending" syntax.
`vars.ExtractPromptVariablesFromString` already extracts it;
`SendRequestCmd.PromptVars` (`internal/app/commands/send_request.go`) already
merges the answers into the prompt scope.

`PromptLabels(*requestView)` walks the URL, then the body, then each **enabled**
key/value pair, deduping in first-appearance order. A disabled header containing
`{{?x}}` does not pop a dialog — matching what is actually interpolated at send
time. Plain `{{name}}` is not a prompt and is ignored.

`AskPrompt(ctx, labels)` blocks the caller until the user sends or cancels, and
also closes the dialog when `ctx` is done: a cancelled send must not leave a
question on screen (`TestAskPromptContextCancellationClosesTheDialog`).

`requestView`/`requestPart` are declared in this package rather than importing
`model`, so the scan is testable from a literal and the dependency stays
one-directional. `RequestView(url, body, parts)` builds one.

### Import

`ParseImport(src, text)` dispatches to the four existing converters
(`convert/curl`, `convert/postman`, `convert/insomnia`, `convert/openapi`) and
returns at most one of a `Request` (cURL) or a `Collection` (the other three).

Auto-detection is a heuristic for the picker, not a parser: a wrong guess still
fails loudly in the converter rather than importing something unexpected.
Detection compares **whitespace-stripped** text, because a real Insomnia export
is pretty-printed (`"_type": "export"`, with spaces) and an initial check that
assumed minified JSON missed it — caught by
`TestParseImportAutoDetectsJSONFormats/insomnia pretty-printed`.

### Save transient

A request created in a scratch tab has no home on disk. `SaveTransientModal`
asks for a name and a destination collection; the name is pre-filled from the
URL's last path segment (or the host, for a bare host) by `suggestedName`, and
the collection starts **unset** so the user makes a deliberate choice. The
picker is `widgets.FreeTextSelect`, so a user may also type a collection name.

`TargetUID` maps a picked label back to a UID; an unknown label passes through
unchanged so the save command reports it rather than silently no-op'ing.

`ErrNoSaveTarget` is returned without opening a dialog when there are no
collections: a picker with no options cannot be satisfied, so an empty modal
would be worse than an error.

## What the view does

`View(c, q)` is the **single** call site the shell needs, which makes "only one
dialog at a time" structural rather than a convention. It reads everything from
`q` each frame and reports clicks back; it holds no state of its own.

`Actions(c, q)` is the footer for a shell that draws its own chrome instead of
using `feedback.Modal`.

## Not delivered

- **Wiring into `UI-APP`.** `UI-DIALOGS` owns the dialogs, not the window. The
  shell must call `View(c, q)` once, and the command goroutines must call
  `AskPrompt` / `SaveTransientModal` / `ImportModal` and block on the `*Wait`.
- **A native file picker for import.** The import dialog takes a paste. Reading a
  file is `PLATFORM.dialog`'s job; the two compose but neither owns the other.
- **Multi-file import and drag-and-drop.** A dropped file needs the platform
  dialog plus a batch variant of `ImportModal`.
- **Nested confirmation stacks.** Deliberately not implemented: superseding
  matches Bruno, where a command that opens a dialog interrupts whatever was
  asking.

## Verification

```
go test -race -cover ./internal/ui/dialogs/      ok  91.8 % of statements
go build ./...                                   clean
go vet ./...                                     clean
gofmt -l internal/ui/dialogs/                    empty
sh scripts/check-deps.sh                         dependency-policy: ok
go test ./...                                    whole tree green
```

### 2026-11-11T04:35Z — antigravity-2 (reviewer)
- Review: Verified UI-DIALOGS implementation against packages.md criteria.
  - Close-on-success verified: dialog stays open on failure and closes only when action succeeds.
  - Prompt variable modal, import/export flows (cURL, Postman, Insomnia, OpenAPI), and save-transient-request verified.
  - Tests: 36 tests, 91.6% statement coverage, -race clean.
- Confirmed: Done.
