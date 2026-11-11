# Status dot

Status: VERIFIED against `bruno/packages/bruno-app/src/components/StatusDot/`
at the commit in `overview.md:8`. No screenshot in `refs/` shows it in
isolation; `refs/main_section_bruno.png` and `refs/tests_bruno.png` show the
request/folder tab bars where it appears.

`StatusDot` is a 10 px filled circle pinned as a superscript marker next to a
tab label, telling the reader that a section has content (default) or an
error (red).

---

## 1. Visual spec

```text
<sup class="ml-[.125rem] opacity-80 font-medium [text-red-500]">
  <svg width=10 height=10 viewBox="0 0 24 24">   ← components/Icons/Dot
     path: circle, r=5, fill=currentColor, strokeWidth=0
  </svg>
</sup>
```

| Property          | Value                                                                                                                        | Source                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| Element           | `<sup>` (superscript baseline offset)                                                                                        | `components/StatusDot/index.js:4-13`      |
| Dot size          | `width=10` — the icon sets both `width` and `height` from the same prop                                                      | `components/Icons/Dot/index.js:3-11`      |
| Shape             | filled circle: `M12 7a5 5 0 1 1 -4.995 5.217…` with `r = 5` on a `24 × 24` viewBox, `fill="currentColor"`, `strokeWidth="0"` | `components/Icons/Dot/index.js:18`        |
| Rendered diameter | `10 × (10/24) ≈ 4.17 px` (the circle's 5 px radius is in viewBox units)                                                      | derived from `:8-10`, `:18`               |
| Colour (default)  | `currentColor` — inherits the tab's text colour                                                                              | `components/Icons/Dot/index.js:11`, `:18` |
| Colour (error)    | Tailwind `text-red-500` (`#ef4444`)                                                                                          | `components/StatusDot/index.js:6-8`       |
| Opacity           | `opacity-80` (0.8)                                                                                                           | `:6`                                      |
| Weight            | `font-medium` (no visual effect on an SVG; kept for parity)                                                                  | `:6`                                      |
| Left margin       | `ml-[.125rem]` (2 px) from the preceding label                                                                               | `:6`                                      |
| Alignment         | `inline-block` on the SVG                                                                                                    | `components/Icons/Dot/index.js:15`        |
| `data-testid`     | `status-dot` or `status-dot-{dataTestId}`                                                                                    | `components/StatusDot/index.js:9`         |

Note the dot is **not** theme-tinted in the default state: it uses whatever
colour the surrounding text has (`currentColor`), i.e. the tab label colour
(`colors.text.subtext0` for inactive `ui/ResponsiveTabs/StyledWrapper.js:31`).

---

## 2. Variants

| Prop         | Value                 | Meaning              | Colour                   | Source                              |
| ------------ | --------------------- | -------------------- | ------------------------ | ----------------------------------- |
| `type`       | `'default'` (default) | section has content  | `currentColor`           | `components/StatusDot/index.js:4-8` |
| `type`       | `'error'`             | section has an error | `text-red-500` `#ef4444` | `:6-8`                              |
| `dataTestId` | any string            | test id suffix       | —                        | `:4`, `:9`                          |

There is no size prop, no tooltip, no hover state.

---

## 3. Sizes

| Context                               | Effective dot                                  | Notes                                                     |
| ------------------------------------- | ---------------------------------------------- | --------------------------------------------------------- |
| Request-pane tabs (`HttpRequestPane`) | ~4.2 px rendered, sits on the superscript line | `components/RequestPane/HttpRequestPane/index.js:109-118` |
| Folder-settings tabs                  | same                                           | `components/FolderSettings/index.js:101-120`              |
| Collection-settings tabs              | same                                           | `components/CollectionSettings/index.js:130-154`          |
| Script sub-tabs (error variant)       | same, red                                      | `components/CollectionSettings/Script/index.js:114`       |

Because the marker is `<sup>`, it is laid out with the browser's
superscript offset; no explicit `vertical-align`, `top` or `font-size` is
set (`components/StatusDot/index.js:5`). A Go port must decide its own
baseline offset (CSS default `vertical-align: super` with
`font-size: smaller` on the parent).

---

## 4. Where it appears

| Host                   | Shown when                                                        | Source                                                  |
| ---------------------- | ----------------------------------------------------------------- | ------------------------------------------------------- |
| Request tabs: `Params` | count sup, no dot                                                 | `components/RequestPane/HttpRequestPane/index.js:108`   |
| `Body`                 | `body.mode !== 'none'`                                            | `:109`                                                  |
| `Headers`              | count sup, no dot                                                 | `:110`                                                  |
| `Auth`                 | `hasAuth` (dataTestId `auth`)                                     | `:111`                                                  |
| `Vars`                 | count sup, no dot                                                 | `:112`                                                  |
| `Script`               | `script.req \|\| script.res`; red when `hasScriptError`           | `:113`                                                  |
| `Assert`               | count sup, no dot                                                 | `:114`                                                  |
| `Tests`                | `tests?.length > 0`; red when `hasTestError`                      | `:115`                                                  |
| `Docs`                 | `docs?.length > 0`                                                | `:116`                                                  |
| `App`                  | `app?.code?.length > 0` (dataTestId `app`)                        | `:117`                                                  |
| `Settings`             | `tags?.length > 0`                                                | `:118`                                                  |
| Folder settings        | scripts/test/auth/tags present                                    | `components/FolderSettings/index.js:101-120`            |
| Collection settings    | auth/scripts/tests/presets/proxy/client certs/proto files present | `components/CollectionSettings/index.js:130-154`        |
| Script sub-tabs        | pre-request / post-response have errors                           | `components/CollectionSettings/Script/index.js:114-120` |

In the request tab bar the count-bearing tabs (`Params`, `Headers`, `Vars`,
`Assert`) use a `<sup class="font-medium">{n}</sup>` indicator instead of a
dot (`components/RequestPane/HttpRequestPane/index.js:108-114`).

The tabs hosting the dots are `ui/ResponsiveTabs` tabs
(`components/RequestPane/HttpRequestPane/index.js:20`), which render
`{tab.label}{tab.indicator}` (`ui/ResponsiveTabs/index.js:212-213`) — the
dot is therefore the tab's `indicator` node.

---

## 5. States

| State                 | Appearance                                                            |
| --------------------- | --------------------------------------------------------------------- |
| Default (has content) | 80 %-opacity dot in the label colour                                  |
| Error                 | 80 %-opacity dot in `#ef4444`                                         |
| Absent                | the `<sup>` is not rendered at all (`condition && <StatusDot />`)     |
| Hover / focus         | no change (inherits the tab's hover colour because of `currentColor`) |
| Disabled              | no concept                                                            |

---

## 6. Interactions & shortcuts

None. The dot itself is not focusable or clickable; clicking anywhere on the
tab changes tabs (`ui/ResponsiveTabs/index.js:204-211`). No ARIA label is
set, so screen readers announce the tab label only — the dot is decorative.

---

## Open Questions

1. **Rendered size is 10 px box, ~4.2 px dot.** The SVG is 10 px but the
   circle only spans 10/24 of the viewBox before scaling; whether the design
   intent was a 10 px dot or a 10 px icon box is unverified.
2. **Un-themed error colour.** `text-red-500` is a Tailwind literal outside
   the token system; dark themes do not adjust it
   (`components/StatusDot/index.js:7`).
3. **`font-medium` on a `<sup>`** has no effect on the SVG glyph; it may be
   a leftover from an earlier text-based dot.
4. **No tooltip or accessible name**, so the distinction between "has
   content" and "has error" is colour-only.
