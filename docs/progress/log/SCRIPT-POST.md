# SCRIPT-POST — assertions and the test harness

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/engine/post/`
**Reference:** `bruno-js/src/runtime/assert-runtime.js`, `bruno-js/src/utils.js`

## What this covers

The declarative half of the post-response phase. `vars:post-response` is already
evaluated natively by `TPL` (D-010), so this package owns the other piece: the
`assert` blocks, which compare a JavaScript expression against an expected value
using one of 28 operators.

The left-hand expression is **not** evaluated here. It is JavaScript — `res.status`
is the common case but `bru.getVar("x").length > 3` is legal — so the caller
supplies an `Expression` function. That keeps the package free of a script
runtime while still owning the operator semantics, which are the part with a
specification behind them.

## The operator table

**28 operators: 16 binary plus 12 unary.** The package brief says 28 and the
source agrees; an intermediate draft of this note claimed 27 and was wrong.

| Binary (16) | Unary (12) |
| --- | --- |
| `eq` `neq` `gt` `gte` `lt` `lte` `in` `notIn` | `isEmpty` `isNotEmpty` `isNull` `isUndefined` |
| `contains` `notContains` `length` `matches` `notMatches` | `isDefined` `isTruthy` `isFalsy` `isJson` |
| `startsWith` `endsWith` `between` | `isNumber` `isString` `isBoolean` `isArray` |

`TestOperatorListIsComplete` pins the count and every name, so the table cannot
drift back.

### Value-column grammar

The assertion's value column holds the operator *and* its operand, separated by
the first space. Three details come straight from `parseAssertionOperator` and are
each easy to get wrong:

- **An unrecognised leading word is not an unknown operator.** `assert { status:
  200 }` is legal equality shorthand; `200` is not an operator, so the whole
  string becomes an `eq` operand. Treating it as an error would reject the
  commonest shorthand.
- **A unary operator ignores trailing text.** `isJson extra` is `isJson`, not a
  parse failure.
- **An empty value is `eq ""`**, not an error.

### Operand typing

Operands are untyped text, and `eq 200` only works because the text is coerced —
the same conversion `evaluateJsTemplateLiteral` performs: `true`/`false`/`null`/
`undefined`, quoted text (which stays text even when numeric), a numeric string
within JavaScript's safe-integer range, and otherwise the text itself.

Above 2^53 the text is kept, because converting it would lose digits — Bruno does
the same after issue #1000, where a long numeric identifier was silently altered.

### Forgiving structured operands

`in` accepts `a,b` and `[a, b]`; `between` takes two comma-separated values;
`matches` accepts `^v\d+` and `/^v\d+/`. Both spellings appear in the wild and
`evaluateRhsOperand` normalises them, so this does too.

## Two places the implementation is stricter than a naive port

**`"200"` does not equal `200`.** chai's `equal` is strict, and the operand `200`
parses to a *number*. A string body must not satisfy a numeric assertion just
because it looks numeric. Numeric *representations* still compare equal
(`float64(200)`, `int(200)`, `int64(200)`), because a JS number exports in more
than one Go type.

**`isNumber("1")` is false.** It is a type check; coercing would make it useless
for distinguishing a stringly-typed body from a real number. Relational
operators (`gt`, `between`) *do* coerce numeric strings, because JavaScript does:
`"6" > 5` is true.

## Results

`Run` reports every enabled assertion in source order. A failure does not stop
the run, and **an expression that throws does not discard the results already
collected** — the acceptance criterion. A thrown `bru.*` call in assertion three
leaves assertions one and two intact and reports three's error.

That matters because the user fixing assertions wants the whole list, and because
a runner summarising a folder must not lose the passing assertions of a request
whose last assertion scripted something broken.

Each result carries the expression, operator, rendered actual and expected, and
an explanatory error such as `expected 200 to equal 404`.

## Known gap: `.bru` line mapping

The acceptance criterion asks that errors carry file and line. **They do not,
and cannot yet**: `model.Request.Assertions` is `[]KeyValue`, which has no
position field, and the `.bru` mapper discards the AST positions when it builds
the model. `post` therefore reports the expression text, not a line number.

Fixing this needs `model.KeyValue` (or an assertions-specific type) to carry a
source position, which is `MODEL`/`FORMAT-BRU` territory. Recorded rather than
faked with a line number of 0.

## Verification

```
go vet ./internal/core/engine/post/          # clean
go test -race -cover ./internal/core/engine/post/
ok  ...  coverage: 96.5% of statements
```

Coverage: the operator count and full name set; the value-column grammar (10
cases); every operator on a passing and, where it can fail, a failing case; the
loose-equality matrix (19 cases) including the string-vs-number rule; `asFloat`
for all 13 numeric types plus the string and `json.Number` fallbacks;
`lengthOf` for strings, arrays, objects, scalars and nil; `textOf`; every
`literal` form including the unsafe-integer case; all 28 failure descriptions
being non-empty and non-trivially different from the operator name; the
type-error paths for `gt`, `length`, `between`, `in` and `matches`; interpolation
in operands; an invalid pattern failing rather than passing; partial results
preserved when an expression throws; and a missing evaluator reporting instead of
panicking.

## Sources

- `bruno-js/src/runtime/assert-runtime.js` — `parseAssertionOperator`,
  `evaluateRhsOperand`, `runAssertions`, the `json`/`match`/`jsonBody` chai plugins
- `bruno-js/src/utils.js` — `evaluateJsTemplateLiteral`
- `bruno-app/src/components/RequestPane/Assertions/index.js` — the editor's operator list
