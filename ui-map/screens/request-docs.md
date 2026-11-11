# Screens — request docs & response example

Mode: **Read / Operate**.

Sources: `components/Documentation/` (`index.js`, `DocsEditor/`, `DocsAction/`,
`useDocsEditingState`), `components/ResponseExample/`.

---

## Request Docs tab

Reached from the request sub-tab strip (see `screens/request-subtabs.md`).
A rich-text/markdown editor for the request's documentation, stored with the
request.

- Sections seen in `WorkspaceHome/WorkspaceDocs` (same docs component family):
  **`Documentation`**, **`Project overview`**, **`Setup instructions`**,
  **`Key workflows`** — these are the workspace-level docs templates; the
  request Docs tab uses the same editor on the request's own doc.
- Editing state via `useDocsEditingState`; save/cancel surfaced by `DocsAction`.
- Empty state: no placeholder is defined (see Open Questions in
  `request-subtabs.md`).

## Response example

`ResponseExample` (35 files) renders a saved example response — the same body
viewer as the live response pane, in a static/read-only mode. Layout mirrors
`QueryResponse` (see `screens/response-viewer.md`) without the live actions
(no Send, no timing from a fresh run; the saved example keeps its recorded
status/time/size).

## Open Questions

- `ResponseExample` internals not fully read; build it as `QueryResponse` minus
  live actions and verify the recorded-metadata display.
