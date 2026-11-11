# Screens — request docs & response example

Mode: **Read / Operate**.

Sources: `components/Documentation/` (`index.js`, `DocsEditor/`, `DocsAction/`,
`useDocsEditingState`), `components/ResponseExample/`.

---

## Request Docs tab

Reached from the request sub-tab strip: **`Docs`**
(`components/RequestPane/HttpRequestPane/index.js:34`). When the Docs tab is
active the right side of the sub-tab bar shows a `DocsAction` (edit/save
controls) plus the AI-assist control (`:149-153`).

- Sections seen in `WorkspaceHome/WorkspaceDocs` (same docs editor family):
  **`Documentation`**, **`Project overview`**, **`Setup instructions`**,
  **`Key workflows`** — these are the workspace-level docs placeholders; the
  request Docs tab uses the same `DocsEditor` on the request's own doc.
- Editing state via `useDocsEditingState`; save/cancel surfaced by `DocsAction`.
- **Empty state:** the request Docs tab has **no placeholder** — an empty doc
  renders an empty editor. (Collection settings _does_ seed a placeholder via
  `documentationPlaceholder`, `components/CollectionSettings/Docs/index.js:79,91`;
  the request tab does not.)

## Response example

`ResponseExample` (35 files) renders a saved example response — the same body
viewer as the live response pane, in a static/read-only mode. Layout mirrors
`QueryResponse` (see `screens/response-viewer.md`) without the live actions
(no Send, no timing from a fresh run; the saved example keeps its recorded
status/time/size).

## Open Questions

None outstanding: the Docs empty state and action bar are documented above.
