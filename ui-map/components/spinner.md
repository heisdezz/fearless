# Spinner

Status: VERIFIED against `bruno/packages/bruno-app/src/components/Spinner/`
at the commit in `overview.md:8`. No screenshot in `refs/` shows it; every
value below is read from source.

`components/Spinner` is a single rotating ring. It is **not** used anywhere
in the app today (grep over `src/` finds only its own definition
`components/Spinner/index.js:5`); loading states instead use
`IconLoader2` with Tailwind `animate-spin`. The response-pane loading state
uses a third spinner, `IconRefresh` rotating — see §5. A Go port should
implement one spinner primitive and map all three call sites onto it.

---

## 1. Visual spec

```text
<div class="animate-spin">        ← the ring itself
</div>
[optional] <div>{children}</div>  ← label under/next to the ring
```

| Property         | Value                                                                                           | Source                                    |
| ---------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------- |
| Size             | `2rem` × `2rem` (32 × 32 px)                                                                    | `components/Spinner/StyledWrapper.js:5-6` |
| Shape            | `border-radius: 50%`                                                                            | `:9`                                      |
| Ring thickness   | `border: 0.25em solid currentColor` (4 px at the inherited 16 px font, scales with `font-size`) | `:7`                                      |
| Gap              | `border-right-color: transparent` — the ring is three quarters drawn                            | `:8`                                      |
| Rotation         | `animation: spinner-border 0.75s linear infinite`                                               | `:11`                                     |
| Inline alignment | `display: inline-block; vertical-align: text-bottom`                                            | `:4-7`                                    |
| Colour           | `currentColor` — inherits the surrounding text colour                                           | `:7`                                      |
| Children         | rendered in a sibling `div`, no styling (`{children && <div>{children}</div>}`)                 | `components/Spinner/index.js:8-9`         |

Component signature: `Spinner({ size, color, children })` — **`size` and
`color` are accepted but ignored**; the source carries a `// Todo: Size,
Color config support` comment (`components/Spinner/index.js:4-12`). Any Go
port should either implement them or drop them from the API.

---

## 2. Animation

- Name: `spinner-border`, duration `0.75s`, timing `linear`, iteration
  `infinite` (`components/Spinner/StyledWrapper.js:11`).
- **The `@keyframes spinner-border` rule does not exist in the repository.**
  It is not in `src/globalStyles.js` (which only defines `fade-in`,
  `fade-out`, `fade-and-slide-in-from-top`, `fade-and-slide-out-from-top`,
  `rotateClockwise`, `rotateCounterClockwise` — `globalStyles.js:127-188`)
  and not in `src/styles/globals.css`. The class also carries Tailwind
  `animate-spin` (`index.js:8`), whose own keyframes rotate `0 → 360deg`
  over 1 s. Unverified which animation wins in the built bundle.

For a Go port, define the ring rotation explicitly:

| Parameter        | Suggested value (matches the intended `spinner-border`) |
| ---------------- | ------------------------------------------------------- |
| Rotation         | full `360°` turn                                        |
| Duration         | `0.75s`                                                 |
| Easing           | `linear`                                                |
| Repeat           | infinite                                                |
| Transform origin | centre                                                  |

---

## 3. Sizes and states

| Case                      | Value                                                                                                                         | Source                 |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| Only size                 | 32 × 32 px, 4 px ring                                                                                                         | `StyledWrapper.js:5-7` |
| Compact placement         | wrap in a container that sets `font-size` to shrink the ring, since thickness is `em`-based — used nowhere in source          | —                      |
| Default                   | animating, `currentColor`                                                                                                     | —                      |
| With label                | children rendered below the ring, inherits block layout                                                                       | `index.js:9`           |
| Disabled / reduced motion | no source rule; `globalStyles.css` disables all animation only under `.measuring-overflow` (`src/styles/globals.css:189-194`) | —                      |

There are no colour variants — tint via `color` on a parent.

---

## 4. Where a spinner appears in the response/runner surface

| Location                 | Implementation                                                | Source                                                                                                     |
| ------------------------ | ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Response loading overlay | `IconRefresh` 24, `rotateCounterClockwise 1s linear infinite` | `components/ResponsePane/Overlay/index.js:26`, `Overlay/StyledWrapper.js:20-23`, `globalStyles.js:180-188` |
| Runner row (in flight)   | `IconRefresh` 18/1.5 with Tailwind `animate-spin ml-1`        | `components/RunnerResults/index.jsx:418`                                                                   |
| Tab-panel loading        | `IconLoader2` 24/1.5 `animate-spin`                           | `components/RequestTabPanel/RequestTabPanelLoading/index.js:7`                                             |
| Request loading chip     | `IconLoader2` 16/2 `animate-spin`                             | `components/RequestTabPanel/RequestIsLoading/index.js:39`                                                  |
| Async buttons            | `<Button loading>` renders a `.spinner-icon` (14 px at `sm`)  | `ui/Button/StyledWrapper.js:30-33`                                                                         |

`components/Spinner` itself: no call sites.

---

## 5. Interactions & shortcuts

None. The spinner is decorative; no click target, focus target, ARIA role or
live region is set in source. Any accessibility guarantee must be supplied by
the caller (a `role="status"` wrapper), which the app does not currently do
here.

---

## Open Questions

1. **Missing keyframes.** `spinner-border` is referenced but undefined
   (`components/Spinner/StyledWrapper.js:11`); the element also has Tailwind
   `animate-spin`. Unverified which animation actually runs.
2. **Ignored props.** `size` and `color` are accepted and discarded
   (`components/Spinner/index.js:4-12`). A Go port should document its own
   sizing API rather than mirror this.
3. **Dead component.** No consumer exists in `src/`; the mapping should treat
   this file as the canonical visual definition rather than a live one.
4. **Direction mismatch.** The overlay spinner rotates
   counter-clockwise (`scaleY(-1)` + `rotate(360deg→0deg)`,
   `globalStyles.js:180-188`) while `animate-spin` rotates clockwise. Two
   spinners on one screen would spin opposite ways.
