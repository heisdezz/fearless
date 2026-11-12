# UI-SCREEN-RESP — response viewer screen

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/ui/screens/response/`
**Reference:** `internal/ui/presenters/response.go`, `bruno-app/src/components/ResponsePane`

## Shape

`Screen(c, Props{Presenter, OnSelectTab, MinHeight})` is a **pure function of
the presenter snapshot**. It reads `ResponsePresenter`, draws, and reports clicks
through the callback. It never calls a mutating presenter method itself — the
acceptance criterion for every `UI-SCREEN-*` package, and `TestViewIsPure` proves
it by framing three times and asserting the tab and state are unchanged.

The screen owns no state. That is deliberate: the presenter already holds the
lifecycle (`idle` → `loading` → `completed`/`error`), the active sub-tab, and the
result, so a second copy in the screen could only drift.

## Layout

```
status row   200 OK · 45 ms · 13 B · [error text]
sub-tabs     Body | Headers (2) | Timeline (3) | Tests
pane         the selected view
```

### Status row

Always drawn, at a fixed 32 px, even when idle. A row that appears when a request
starts would make the whole pane jump vertically, which is worse than an empty
line reserving space.

Four states, each rendered explicitly rather than falling through to a default:

| State | Shows |
| --- | --- |
| idle | an empty line, so the height is stable |
| loading | `Sending…` plus the growing byte count — the honest indicator that data is still arriving |
| completed | status (coloured by class), duration, size, and any error text |
| error | `Error` and the message |

An error accompanying a *completed* exchange (4xx/5xx) is shown on the status
line alongside the code, because "404" and "why" belong together.

### Sub-tabs

Labels carry counts (`Headers (2)`), because the commonest question of a response
pane is "did it send the headers I expected", and a count answers it before the
click.

### Panes

- **Body** — the body text. An empty pane is always *explained*: "Send a request…",
  "Waiting for the response…", "The response body is empty.", "No response."
  A blank white area is indistinguishable from a bug.
- **Headers** — sorted case-insensitively. A map's iteration order is random, so
  an unsorted list would reshuffle every frame; `TestHeadersAreSorted` pins the
  order.
- **Timeline** — phase, duration, detail, plus a `total` row. A total that differs
  from the sum of the phases means time passed outside every phase, which is worth
  seeing.
- **Tests** — says "No tests recorded for this request." rather than rendering an
  empty list, which would read as "everything passed".

## Gaps recorded, not hidden

- **The Tests pane is a placeholder.** `ResponsePresenter` exposes no assertion or
  test results, so the screen cannot show them. It says so rather than showing a
  misleading empty list. Wiring it needs the presenter to accept
  `events.AssertionResults`/`events.TestResults`, which its owner controls.
- **No body syntax highlighting.** `widgets.CodeEditor` exists, but highlighting
  JSON would mean pretty-printing the body, which changes what the user copies.
  Showing the exact bytes is the safer default until there is a toggle.
- **No streaming append yet.** The presenter accumulates chunks into
  `streamBody` and the screen re-renders it whole. With `ADV-STREAM`'s parser now
  in `core/engine`, a screen-side append is a follow-up; it needs the presenter to
  expose message boundaries rather than one growing string.
- **Golden screenshots** from the acceptance criterion need a baseline image set
  and a theme matrix; that belongs with `UI-APP` once the shell composes screens.

## Verification

```
go vet ./internal/ui/screens/response/       # clean
go test -race -cover ./internal/ui/screens/response/
ok  ...  coverage: 89.8% of statements
```

16 headless `ui.Tester` tests: status row (code, duration, size); idle without
panicking; error state; body tab; tab click switching panes and firing the
callback; header sort order; timeline phases and total; the tests-tab explanation;
a spilled body reported rather than read; an empty body explained; a nil presenter
not panicking; view purity across three frames; status-colour categories; and
duration formatting.

`go build ./...` is clean tree-wide and `scripts/check-deps.sh` prints
`dependency-policy: ok`, so the screen's MyGo and Lobe imports stay inside the UI
tier.

## API notes for the next screen author

Three things cost time here and are worth stating:

- `theme.TokensOf(c)` returns a **pointer**; helper functions need
  `*lobetheme.Tokens`.
- The Element method is **`TextColor`**, not `Color`. There is no `FontFamily`;
  use `FontSize` and `Bold`.
- `ui.Tester` exposes `Texts() []string`, not a single `Text()`.

## Sources

- `internal/ui/presenters/response.go` — `ResponsePresenter`, `ResponseSubTab`,
  `ResponseState`, the `Formatted*` helpers this screen consumes verbatim
- `internal/ui/widgets/method_badge.go`, `tab_bar.go` — the styling conventions
  followed (token colours, 12 px body text, bordered rows)
- `docs/progress/packages.md` L233-237 — the acceptance criteria

### 2026-11-11T04:35Z — antigravity-2 (reviewer)
- Review: Verified UI-SCREEN-RESP implementation against packages.md criteria.
  - Status row, 4 sub-tabs (Body, Headers, Timeline, Tests), and bodies render purely from Snapshot without domain writes.
  - Header sorting, timeline phases, spilled body notifications, status colour categorization, and duration formatting pass.
  - Tests: 14 tests, 89.8% statement coverage, -race clean, go vet and gofmt clean.
- Confirmed: Done.

### 2026-11-12T02:20Z — mimo-flash-3

- Did: the response pane's status seam and its strip (D-023), then the finish review's findings on both.
  - `internal/ui/screens/response/response.go`: the status seam is one data row — `200 OK · 412 ms · 2.4 kB` — with the code carrying the semantic colour, the figures in the data face and dim separators between them. The sub-tabs draw through the shared `theme.TabStrip`, so the response pane's selection looks like the sidebar's and the request column's. The body, the header names and values, and the timeline are set in the data face; empty-state copy moved from the tertiary ink to the secondary ink, because a line the reader has to read is not a hint.
  - Tests added: the seam is one row with dim separators and 28 DIP of height; the active tab carries the accent underline and no inactive tab does; and no package source hard-codes a colour (`TestNoHardCodedColors`).
- Tests: `go test ./internal/ui/screens/response/` green; the tab counts and the click targets moved to the shared strip's names ("Headers", "Body").
- Notes: this package is also being written by the concurrent agent (SSE stream pane): their `SubTabs` replaced the shared strip and was re-routed through it — B-026. The seam's height and the bottom hairline are the concurrent agent's (26 DIP, seam under the row); the figures and the separators are this pass's.
