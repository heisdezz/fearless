# Assertions & Tests

Status: VERIFIED unless marked.

Bruno has **two** test mechanisms:

1. **`assert { }` blocks** — declarative `lhs : operator rhs`, evaluated without writing JS.
2. **`tests { }` blocks** — arbitrary JS with `test()` and chai `expect`.

Both feed the same Tests tab.

---

## 1. Assert block syntax

```bru
assert {
  res.status: 200
  res.body.message: eq Authentication successful
  res.headers.content-type: contains application/json
  res.responseTime: lt 1000
  res.body.items: isArray
}
```

The RHS string is parsed as `<operator> <operand>`; **bare values default to `eq`**.

Source: `bruno-lang/v2/src/bruToJson.js` (`assertdictionary`), `bruno-js/src/runtime/assert-runtime.js` `parseAssertionOperator` (line ~235).

---

## 2. Operator reference (VERIFIED, 28 operators)

| Category | Operators |
|---|---|
| Equality | `eq`, `neq` |
| Comparison | `gt`, `gte`, `lt`, `lte` |
| Membership | `in`, `notIn` |
| String | `contains`, `notContains`, `startsWith`, `endsWith`, `matches`, `notMatches` |
| Size | `length` |
| Range | `between` |
| Emptiness/null | `isEmpty`, `isNotEmpty`, `isNull`, `isUndefined`, `isDefined` |
| Truthiness | `isTruthy`, `isFalsy` |
| Type checks | `isJson`, `isNumber`, `isString`, `isBoolean`, `isArray` |

Unary operators: `isEmpty`, `isNotEmpty`, `isNull`, `isUndefined`, `isDefined`, `isTruthy`, `isFalsy`, `isJson`, `isNumber`, `isString`, `isBoolean`, `isArray`.

---

## 3. Operand parsing rules

| Operator | RHS accepted forms | Normalization |
|---|---|---|
| `in` / `notIn` | `a,b` or `[a, b]` (brackets stripped) | split on `,`, each trimmed, `{{var}}` interpolated, evaluated as a JS template literal |
| `between` | `min,max` | exactly two values; `.within(min,max)` |
| `matches` / `notMatches` | `^[a-Z]` or `/^[a-Z]/` (slashes stripped) | compiled with `new RegExp(...)` |
| everything else | free text | interpolated, then evaluated as a JS template literal |

Consequence: RHS values are **JS template literals**, so `"{{token}}"` and `${…}` semantics apply.

---

## 4. LHS evaluation

```js
lhs = evaluateJsExpressionBasedOnRuntime(lhsExpr, context, runtime)
```

- Typical expressions: `res.status`, `res.body.id`, `res.headers['x-request-id']`, `res.responseTime`, `bru.getVar('x')`.
- QuickJS evaluates in the sandbox; nodevm uses the local evaluator.

---

## 5. Result shape

```js
{
  status: 'pass' | 'fail',
  lhsExpr: 'res.status',
  rhsExpr: '200',
  operator: 'eq',
  rhsOperand: 200,
  error?: 'expected 201 to equal 200'
}
```

Rendered by `ResponsePane/TestResults`. Failing assertions do **not** stop execution and do not fail the request (they are reported).

---

## 6. Tests block

```js
test('user id is present', () => {
  expect(res.getBody()).to.have.property('id');
  expect(res.getHeader('content-type')).to.include('json');
});

test('async check', async () => {
  const v = await bru.getVar('token');
  expect(v).to.be.a('string');
});
```

- `test(name, fn)` — synchronous or async.
- `expect` / `assert` from chai (+ `chai-string` extras like `startWith`).
- Any uncaught error fails that test; subsequent tests still run.
- `bru.setVar` inside tests persists variables.

Assertion results from `assert {}` are merged into the same list so ordering is preserved (assertions are appended by `createBruTestResultMethods`).

---

## 7. Execution ordering

```text
vars:post-response  →  script:post-response  →  assert { }  →  tests { }
```

Tests and assertions run **after** post-response scripts, so scripts can mutate `res` (`res.setBody(...)`) before checks.

---

## 8. Failure semantics matrix

| Situation | Request considered | UI |
|---|---|---|
| Assertion fails | Successful execution | Red row in Tests |
| Test fails | Successful execution | Red row |
| Script throws | Error for that phase | Error card + partial results retained |
| `expect` used outside `test()` | Fails the whole script | Error card |

Runner exit codes: the collection runner treats any failing test/assertion as a run failure (see `bruno-cli` reporters).

---

## 9. Go implementation (PROPOSED)

Implement the operator table natively (it is only 28 cases) so assertions do not require an interpreter, and evaluate expressions with the script engine only:

```go
type Assertion struct {
    Expression string
    Operator   string
    Operand    string
}

func Eval(a Assertion, scope Scope, eval ExprEvaluator) Result
```

Reuse an `ExprEvaluator` interface backed by goja for `lhsExpr` and for RHS template-literal cases. Add a compatibility test suite by replaying Bruno's `asserts` fixture collection (`packages/bruno-tests/collection/asserts/`).
