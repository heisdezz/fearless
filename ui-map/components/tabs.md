# Component — tabs & segmented control

Sources: `src/ui/Tabs/`, `src/ui/ResponsiveTabs/`, `src/ui/SegmentedControl/`,
`components/Tabs/`, `components/Tab/`. Request tabs are specified in
`screens/request-tabs.md`; this file covers the generic primitives and the
request sub-tab strip look.

---

## Tabs (generic)

Underline-style tab strip. Active tab: `tabs.active.color` (`text`) with a
`2px` bottom border in `tabs.active.border` (`primary.strong`: `#d5882a`
light / `#e7b85f` dark), `font-weight 400`; inactive: `text.subtext1`. Item
spacing `tabs.marginRight` = `1.2rem` (19.2 px). Secondary tone pair
(`tabs.secondary`): active bg `surface1`, inactive bg `surface0`.

(`themes/light/light.js` `tabs`.)

## ResponsiveTabs

Tab strip that overflows into a chevron menu. Reserves a fixed `60px` for the
chevron even though the glyph is ~18 px (flagged in request-pane Open
Questions). Overflow items move into a dropdown rather than scrolling off-screen
(the docs' accessibility floor: "tab strips never overflow off-screen").

## SegmentedControl

Bordered segmented button group; one active segment. Active segment bg
`surface1`, text `text`; inactive bg `surface0`, text `subtext1`. Used for the
response layout toggle, body-mode pickers and similar binary/ternary choices.
Props resolve options to `{value,label}`; keyboard Left/Right moves selection.

## Request sub-tab strip (visual)

The request editor's sub-tabs (Params, Body, Headers, Auth, Vars, Script,
Assert, Tests, Docs, Settings) render as the generic underline tabs. Exact
active/inactive colours and order are in `screens/request-pane.md` and
`screens/request-subtabs.md`.

## Interactions

| Key | Tabs | Segmented |
| --- | --- | --- |
| ← `⌘⇧[` / `⌘⇧]` | previous/next request tab | — |
| ←/→ | — | move segment |
| Home/End | first/last tab | first/last |

Shortcuts for request tabs are listed in `screens/request-tabs.md`.

## Open Questions

- ResponsiveTabs 60 px reserve vs 18 px glyph — reproduce the reserve to keep
  layout stable, or size to content; choose in the Go port.
