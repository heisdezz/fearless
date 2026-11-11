# Screens — about, help & changelog

Mode: **Read**.

Sources: `components/Bruno` (logo/wordmark), `components/BrunoSupport`,
`components/ChangelogTab`, `components/Help`.

---

## About

Reachable from Preferences (see `screens/preferences.md`). Shows the
`Bruno` wordmark and version. `useApp().version` renders as `v{version}` in the
status bar (`StatusBar/index.js:169-171`); the About surface shows the same
version plus support links via `BrunoSupport`.

## Help

`Help` (`components/Help/index.js:49`) is a small **popover** helper used
throughout the app, not a full screen: an icon (default `'question'`,
overridable via `iconComponent`, size default 14) that opens a panel
(default width 200, default placement `right`). Content is arbitrary children.

Use it as the single tooltip-with-rich-content primitive. Plain one-line tips
use `ToolHint` (see `components/tooltip.md`).

## Changelog

`ChangelogTab` renders release notes as a tab. Content is fetched, not static;
a Go port renders the same markdown (see `widgets.md` § Markdown).

## Open Questions

- About screen exact layout not captured; it is a Preferences tab, so build it
  with the Preferences shell (`screens/preferences.md`).
