# Decisions — resolved product/source choices for the Go port

Where Bruno's source is inconsistent or ambiguous, the map records a decision so
an implementer does not have to guess. Each entry states the choice, the
evidence behind it, and what to do instead of the alternative.

These are **map decisions** (how to render Bruno's intent in Go). They are not
product decisions about Fearless itself; those live in
`docs/progress/decisions.md`.

---

## R-1. Sidebar request verbs use `RequestMethod`, not `MethodBadge`

**Decision:** the sidebar renders a compact verb glyph via a sidebar-local
component (`RequestMethod`), not the shared `MethodBadge`.

**Evidence:** `components/Sidebar/Collections/Collection/CollectionItem/RequestMethod/index.js`
and its `StyledWrapper.js:3-8`:
- `font.size.xs` (11 px), `align-self: stretch`, `min-width: 34px`,
  `flex-shrink: 0`, text nudged `top: 1px`;
- the verb is **truncated to 3 characters when longer than 5** (`:15-17`), so
  `OPTIONS` renders `OPT`, `DELETE` renders `DEL`;
- non-HTTP items show `grpc`, `ws`, `gql` instead of a verb (`:10-13`);
- colour comes from `request.methods.<verb>` classes, with `method-grpc`,
  `method-ws`, `method-graphql` for the other types.

`MethodBadge` (`ui/MethodBadge`) is the request-editor/tab badge: fixed 52 px
and always uppercases the full verb. Do not use it in the sidebar — it would
either clip long verbs or widen every row.

**Consequence:** two verb renderers exist on purpose. Share only the colour
table (`internal/ui/theme` `MethodColor`).

---

## R-2. Request tabs get no hover background

**Decision:** do not add a hover tint to request tabs. Hover changes only the
close button's appearance.

**Evidence:** `components/RequestTabs/StyledWrapper.js` has no `:hover` rule on
the tab itself; the only hover rules are on the collection-header icons
(`:189-196`, using `background.surface0`) and the overflow name fade
(`:83,:98`). There is no `requestTabs.hoverBg` token in the theme.

**Consequence:** if a hover affordance is wanted for discoverability, it is an
addition to Bruno's look, not a port of it — record it as a Fearless decision
and add a token, rather than inventing an unrecorded colour.

---

## R-3. Preview tabs are italic only

**Decision:** a preview (unpinned) tab differs by **italics on the label** only.
No separate colour, background, or border.

**Evidence:** `tab.preview` drives an italics class; there is no preview colour
in `themes/*`. The draft/unsaved marker is a separate dot in `draftColor`
(`#cc7b1b`, both themes).

**Consequence:** do not introduce a preview colour token.

---

## R-4. Assertion operators show their ids, except `eq`/`neq`

**Decision:** render operator labels exactly as source does — `equals` and
`notEquals` for the two, and the raw id for every other operator (`gt`, `gte`,
`lt`, `lte`, `in`, `notIn`, `contains`, `notContains`, `length`, `matches`,
`notMatches`, `startsWith`, `endsWith`, `between`, `isEmpty`, `isNotEmpty`,
`isNull`, `isUndefined`, `isDefined`, `isTruthy`, `isFalsy`, `isJson`,
`isNumber`, `isString`, `isBoolean`, `isArray`).

**Evidence:** `components/RequestPane/Assertions/AssertionOperator/index.js:72-83`
(`getLabel` special-cases only `eq`/`neq`). The human-readable names exist only
as a comment block (`:3-29`) and are not wired to the UI.

**Consequence:** a Go port that "fixes" the labels diverges from what a Bruno
user sees and from saved `.bru` round-trips. If readability is wanted, do it as
a deliberate, recorded improvement (add a display-name map) — not silently.

---

## R-5. Secret values mask with `*`, one per character

**Decision:** mask secret values with a literal `*` per character, with an eye
toggle to reveal. Use the same mask on every surface.

**Evidence:** `utils/common/masked-editor.js:65` — `constructor(editor, maskChar = '*')`
and `MultiLineEditor` constructs it as `new MaskedEditor(this.editor, '*')`
(`components/MultiLineEditor/index.js:271`). The inherited-variable row uses the same character:
`InheritedVariableRow/index.js:8-9` (`const MASK_CHARACTER = '*'`,
`value.replace(/[^\n]/g, MASK_CHARACTER)`).

**Consequence:** the earlier note in `go-ui-notes.md` claiming "three masking
styles (CodeMirror bullets / `********` / per-char `*`)" was **wrong** — there
is one character. Mask per character, preserving line breaks, and reuse
`SecretEyeButton` behaviour (masked by default; reveal toggles the whole cell).

---

## R-6. `input.background` is a bug, not a token to add

**Decision:** the Go port uses `input.bg` wherever Bruno's source says
`input.background`.

**Evidence:** `themes/light/light.js` / `dark.js` define `input.{bg,border,focusBorder,placeholder}`
only. `components/Preferences/Display/Zoom/StyledWrapper.js:25` reads
`theme.input.background`, which is `undefined`, so that select renders with no
background — a latent source bug.

**Consequence:** do not add an `input.background` alias to the token set; just
use `input.bg`. If exact visual parity with the shipped bug is ever required,
say so explicitly.

---

## R-7. Shortcuts live in the global hotkey provider

**Decision:** `Cmd/Ctrl J` (Change Orientation) and every other binding are
global, handled once, not per-pane.

**Evidence:** `providers/Hotkeys/index.js:386` binds `changeLayout`; the mapping
is `providers/Hotkeys/keyMappings.js:43`. The request pane does not own it.

**Consequence:** the Go keybinding router registers these at app level (see
`internal/ui/app/keybindings.go`), mirroring the existing structure.

---

## R-8. Sidebar `MethodBadge` vs text slot — resolved by R-1

Superseded: R-1 explains the sidebar's `RequestMethod` and why it is not
`MethodBadge`. There is no open slot/badge question.

---

## Still genuinely open (need a capture, not a decision)

- Sidebar indent geometry at depth ≥ 1 (flex-derived).
- `refs/` captures are from a different build than the pinned commit.
- Preferences / environments surfaces have no captures.

These need screenshots from a build at `60d7517`, not more source reading.
