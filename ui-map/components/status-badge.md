# Component — StatusBadge & CountBadge

Source: `src/ui/StatusBadge/`, `src/ui/CountBadge/`.

---

## StatusBadge (`StatusBadge/index.js`)

Generic pill/badge themed by status.

| Prop                          | Values (default first)                          | Notes                                |
| ----------------------------- | ----------------------------------------------- | ------------------------------------ |
| `status`                      | `muted`, `danger`, `warning`, `info`, `success` | maps to `status.<key>` tokens        |
| `variant`                     | `light`, `filled`, `outline`, `ghost`           | `light` = 15 % tint bg + status text |
| `size`                        | `sm`, `xs`, `md`                                |                                      |
| `radius`                      | theme radius key (`sm`…`xl`) or CSS value       | default `border.radius.sm` (4)       |
| `leftSection`, `rightSection` | ReactNode                                       | e.g. icon, Help tooltip              |

Colour mapping (`design-system.md` §1.4): `light` = `status.<k>.background`

- `status.<k>.text`; `outline` = transparent + `status.<k>.border`;
  `filled` = `status.<k>.text` solid bg + inverse text; `ghost` = text only.
  `muted` uses `overlay`/`subtext` neutrals.

Usage examples in source docs: `<StatusBadge status="danger">Error</StatusBadge>`,
`<StatusBadge status="info" variant="outline" radius="xl">v2.1</StatusBadge>`
(`StatusBadge/index.js:20-23`).

## CountBadge (`CountBadge/`)

Small numeric counter used on tabs/sidebar rows (e.g. unread notifications,
result counts). Circular or pill depending on digits; uses `primary.solid`
background with inverse text when active, neutral fill otherwise. Read the
component for exact min-width and padding.

## Related

`DeprecationWarning` (`components/DeprecationWarning`) uses the
`deprecationWarning.*` tokens: bg `rgba(217,31,17,.1)`, border same, icon
`#d91f11` (light) / `#fa5343` (dark), text `#343434`/`#b8b8b8`.
`StopWatch` and the response status/time/size badges are specified in
`screens/response-pane.md`.
