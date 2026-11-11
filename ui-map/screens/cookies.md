# Screen — cookies

Mode: **Operate**. Modal listing stored cookies grouped by domain, editable.

Source: `components/Cookies/` (`index.js`, `StyledWrapper.js`). Opened from the
status bar (`components/StatusBar/index.js:136-147`). `role="dialog"`, `aria-modal`,
`aria-labelledby="cookies-title"`, `aria-describedby="cookies-description"`.

---

## Layout tree

```text
┌──────────── Cookies  [Search by domain…]  [+ Add Cookie] ────────────┐
│ ▾ api.example.com  (3 cookies)              [+ Add]                  │
│     name      value        domain   path   …   [edit] [delete]       │
│ ▸ auth.example.com (1 cookie)                                        │
└──────────────────────────────────────────────────────────────────────┘
```

- Modal size `xl` (`components/Cookies/index.js:126`), `title="Cookies"`, `hideFooter` (no OK/Cancel
  bar; header carries a custom toolbar, `components/Cookies/index.js:127-152`).
- Header: heading `Cookies` (`text-xs font-medium`), search input
  `placeholder="Search by domain"` (auto-focused), **`Add Cookie`** button with
  `IconCirclePlus` (16, stroke 1.5) (`components/Cookies/index.js:134-151`).
- Body: `Accordion` of domains, default first open (`components/Cookies/index.js:196`). Each
  header shows the domain, a count **`({n} cookie|/cookies)`**, and an add
  button scoped to that domain (`components/Cookies/index.js:199-215`).

## States & copy (verbatim)

| State | Copy |
| --- | --- |
| No cookies at all | `IconCookieOff` 48 + **`No cookies found`** + **`Add cookies to get started`** + `Add Cookie` button (`components/Cookies/index.js:161-177`) |
| Search matches nothing | `IconSearch` 48 + **`No search results`** + **`Try a different search term`** (`components/Cookies/index.js:179-188`) |
| Delete confirm | modal `Delete Cookie` (`components/Cookies/index.js:40`) |
| Clear-domain confirm | modal `Clear Domain Cookies` (`components/Cookies/index.js:15`), warning icon `IconAlertTriangle` 32 |

## Interactions

- Row actions per cookie: edit (`IconEdit`), delete (`IconTrash`).
- Domain header: add cookie under that domain, clear all cookies of the domain.
- Search narrows by domain; empty-domain list auto-collapses.
