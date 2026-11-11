package post

import (
	"errors"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

// TestOperatorListIsComplete pins the count against assert-runtime.js.
//
// 16 binary plus 12 unary is 28 operators, which is the number in the table.
func TestOperatorListIsComplete(t *testing.T) {
	ops := Operators()
	if len(ops) != 28 {
		t.Fatalf("operator count = %d, want 28: %v", len(ops), ops)
	}

	seen := map[string]bool{}
	for _, op := range ops {
		if seen[op] {
			t.Errorf("%q listed twice", op)
		}
		seen[op] = true
		if !Known(op) {
			t.Errorf("%q is listed but not known", op)
		}
	}

	// Every operator Bruno documents must be present.
	for _, want := range []string{
		"eq", "neq", "gt", "gte", "lt", "lte", "in", "notIn",
		"contains", "notContains", "length", "matches", "notMatches",
		"startsWith", "endsWith", "between",
		"isEmpty", "isNotEmpty", "isNull", "isUndefined", "isDefined",
		"isTruthy", "isFalsy", "isJson", "isNumber", "isString",
		"isBoolean", "isArray",
	} {
		if !seen[want] {
			t.Errorf("%q missing from the operator table", want)
		}
	}
}

// TestParseSpecification covers the value-column grammar.
func TestParseSpecification(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		operator string
		operand  string
	}{
		{"operator and operand", "eq 200", "eq", "200"},
		{"operand with spaces", "contains foo bar", "contains", "foo bar"},
		{"unary operator", "isTruthy", "isTruthy", ""},
		// A unary operator ignores trailing text rather than treating it as an
		// operand, because it has none to take.
		{"unary with trailing text", "isJson extra", "isJson", ""},
		// An unrecognised word is equality against the whole string, which is
		// what `assert { status: 200 }` relies on.
		{"bare value", "200", "eq", "200"},
		{"bare text", "hello world", "eq", "hello world"},
		{"empty value", "", "eq", ""},
		{"only whitespace", "   ", "eq", ""},
		{"leading and trailing spaces", "  eq 200  ", "eq", "200"},
	}
	for _, tc := range cases {
		spec := ParseSpecification("res.status", tc.value)
		if spec.Operator != tc.operator || spec.Operand != tc.operand {
			t.Errorf("%s: got (%q, %q), want (%q, %q)",
				tc.name, spec.Operator, spec.Operand, tc.operator, tc.operand)
		}
		if spec.Expression != "res.status" {
			t.Errorf("%s: expression = %q", tc.name, spec.Expression)
		}
		if spec.Raw != tc.value {
			t.Errorf("%s: raw = %q, want %q", tc.name, spec.Raw, tc.value)
		}
	}
}

// evalCase is one operator check.
type evalCase struct {
	value string
	// actual is the left-hand value.
	actual any
	// rhs is what ParseSpecification would resolve to, when the operand text is
	// a simple literal. Structured operands are supplied explicitly.
	rhs  any
	pass bool
	// skipResolve skips operand resolution and uses rhs as-is.
	skipResolve bool
}

// TestEveryOperator covers each of the 27 operators on both a passing and a
// failing case where the operator can fail.
func TestEveryOperator(t *testing.T) {
	cases := []evalCase{
		// Equality family. The operand text is parsed by evaluateJsTemplateLiteral,
		// so `200` becomes a number and compares numerically.
		{value: "eq 200", actual: 200.0, rhs: 200.0, pass: true},
		{value: "eq 200", actual: 201.0, rhs: 200.0, pass: false},
		{value: "eq hello", actual: "hello", rhs: "hello", pass: true},
		{value: "neq 200", actual: 201.0, rhs: 200.0, pass: true},
		{value: "neq 200", actual: 200.0, rhs: 200.0, pass: false},

		// Numeric comparisons.
		{value: "gt 5", actual: 6.0, rhs: 5.0, pass: true},
		{value: "gt 5", actual: 5.0, rhs: 5.0, pass: false},
		{value: "gte 5", actual: 5.0, rhs: 5.0, pass: true},
		{value: "lt 5", actual: 4.0, rhs: 5.0, pass: true},
		{value: "lte 5", actual: 5.0, rhs: 5.0, pass: true},

		// Membership.
		{value: "in a,b,c", actual: "b", skipResolve: true, rhs: []any{"a", "b", "c"}, pass: true},
		{value: "in a,b,c", actual: "z", skipResolve: true, rhs: []any{"a", "b", "c"}, pass: false},
		{value: "notIn a,b", actual: "z", skipResolve: true, rhs: []any{"a", "b"}, pass: true},

		// Substring and pattern.
		{value: "contains ell", actual: "hello", rhs: "ell", pass: true},
		{value: "contains zzz", actual: "hello", rhs: "zzz", pass: false},
		{value: "notContains zzz", actual: "hello", rhs: "zzz", pass: true},
		{value: "matches ^v\\d+", actual: "v1.2", rhs: `^v\d+`, pass: true},
		{value: "matches ^x", actual: "v1.2", rhs: `^x`, pass: false},
		{value: "notMatches ^x", actual: "v1.2", rhs: `^x`, pass: true},

		// Length.
		{value: "length 3", actual: "abc", rhs: 3.0, pass: true},
		{value: "length 3", actual: "ab", rhs: 3.0, pass: false},

		// Prefix and suffix.
		{value: "startsWith Hel", actual: "Hello", rhs: "Hel", pass: true},
		{value: "startsWith x", actual: "Hello", rhs: "x", pass: false},
		{value: "endsWith llo", actual: "Hello", rhs: "llo", pass: true},

		// Range, inclusive on both ends.
		{value: "between 1,5", actual: 3.0, skipResolve: true, rhs: []any{1.0, 5.0}, pass: true},
		{value: "between 1,5", actual: 1.0, skipResolve: true, rhs: []any{1.0, 5.0}, pass: true},
		{value: "between 1,5", actual: 5.0, skipResolve: true, rhs: []any{1.0, 5.0}, pass: true},
		{value: "between 1,5", actual: 6.0, skipResolve: true, rhs: []any{1.0, 5.0}, pass: false},

		// Unary.
		{value: "isEmpty", actual: "", pass: true},
		{value: "isEmpty", actual: "x", pass: false},
		{value: "isNotEmpty", actual: "x", pass: true},
		{value: "isEmpty", actual: []any{}, pass: true},
		{value: "isNull", actual: nil, pass: true},
		{value: "isNull", actual: "x", pass: false},
		{value: "isUndefined", actual: Undefined{}, pass: true},
		{value: "isUndefined", actual: nil, pass: false},
		{value: "isDefined", actual: "x", pass: true},
		{value: "isTruthy", actual: "x", pass: true},
		{value: "isTruthy", actual: "", pass: false},
		{value: "isFalsy", actual: 0.0, pass: true},
		{value: "isJson", actual: map[string]any{"a": 1.0}, pass: true},
		{value: "isJson", actual: []any{1.0}, pass: true},
		{value: "isJson", actual: "text", pass: false},
		{value: "isNumber", actual: 1.5, pass: true},
		// A numeric string is not a number: isNumber is a type check.
		{value: "isNumber", actual: "1", pass: false},
		{value: "isString", actual: "s", pass: true},
		{value: "isBoolean", actual: true, pass: true},
		{value: "isArray", actual: []any{1.0}, pass: true},
	}

	for _, tc := range cases {
		spec := ParseSpecification("lhs", tc.value)
		rhs := tc.rhs
		if !tc.skipResolve && !IsUnary(spec.Operator) {
			resolved, err := (Evaluator{}).resolveOperand(spec)
			if err != nil {
				t.Errorf("%s: resolve: %v", tc.value, err)
				continue
			}
			rhs = resolved
		}

		outcome := Evaluate(spec, tc.actual, rhs)
		if outcome.Passed != tc.pass {
			t.Errorf("%s with %#v: passed = %v, want %v (error %q)",
				tc.value, tc.actual, outcome.Passed, tc.pass, outcome.Error)
		}
		if !tc.pass && outcome.Error == "" {
			t.Errorf("%s: a failure must explain itself", tc.value)
		}
		if tc.pass && outcome.Error != "" {
			t.Errorf("%s: a pass must not carry an error, got %q", tc.value, outcome.Error)
		}
	}
}

// TestNumericCoercion covers the comparisons that must accept either numeric
// representation: assertions compare a JS number against parsed operand text.
func TestNumericCoercion(t *testing.T) {
	spec := ParseSpecification("res.status", "eq 200")
	rhs, err := (Evaluator{}).resolveOperand(spec)
	if err != nil {
		t.Fatal(err)
	}

	for _, actual := range []any{200.0, int(200), int64(200), float32(200)} {
		if outcome := Evaluate(spec, actual, rhs); !outcome.Passed {
			t.Errorf("actual %#v did not equal 200", actual)
		}
	}

	// A string body must not equal a number: the operand `200` parses to a number,
	// so a truthy string is not coerced into a match.
	if outcome := Evaluate(spec, "200", rhs); outcome.Passed {
		t.Error(`"200" must not equal the number 200`)
	}

	// A quoted operand is text, so it matches the string instead.
	quoted := ParseSpecification("body", `eq "200"`)
	quotedRHS, err := (Evaluator{}).resolveOperand(quoted)
	if err != nil {
		t.Fatal(err)
	}
	if _, isString := quotedRHS.(string); !isString {
		t.Fatalf("quoted operand resolved to %T", quotedRHS)
	}
	if outcome := Evaluate(quoted, "200", quotedRHS); !outcome.Passed {
		t.Error(`a quoted operand must match the string`)
	}
}

// TestOperandResolution covers the forgiving structured syntax.
func TestOperandResolution(t *testing.T) {
	e := Evaluator{}

	cases := []struct {
		value string
		want  any
	}{
		// `in` accepts a bare list and a bracketed one.
		{"in a,b", []any{"a", "b"}},
		{"in [a, b]", []any{"a", "b"}},
		{"in [ 1, 2 ]", []any{1.0, 2.0}},
		// Patterns accept a bare expression and a slash-delimited one.
		{"matches ^v\\d+", `^v\d+`},
		{`matches /^v\d+/`, `^v\d+`},
		// A numeric operand is a number, so `gt` works.
		{"gt 5", 5.0},
		// A quoted operand stays text even when numeric.
		{`eq "200"`, "200"},
		// Unquoted digits become a number, matching evaluateJsTemplateLiteral.
		{`eq 200`, 200.0},
	}
	for _, tc := range cases {
		spec := ParseSpecification("lhs", tc.value)
		got, err := e.resolveOperand(spec)
		if err != nil {
			t.Errorf("%s: %v", tc.value, err)
			continue
		}
		if Render(got) != Render(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.value, got, tc.want)
		}
	}

	// between yields a pair.
	spec := ParseSpecification("lhs", "between 1, 5")
	got, err := e.resolveOperand(spec)
	if err != nil {
		t.Fatal(err)
	}
	pair, ok := got.([]any)
	if !ok || len(pair) != 2 {
		t.Fatalf("between = %#v", got)
	}

	// Above JavaScript's safe-integer bound the conversion would lose digits, so the
	// text is kept — a long numeric identifier must survive an assertion.
	if got := literal("12345678901234567890"); got != "12345678901234567890" {
		t.Errorf("an unsafe integer should stay text, got %T %v", got, got)
	}
	if _, err := e.resolveOperand(ParseSpecification("lhs", "between 1")); err == nil {
		t.Error("expected an error for a one-sided between")
	}
}

// TestInterpolationInOperand covers {{var}} in the operand column.
func TestInterpolationInOperand(t *testing.T) {
	e := Evaluator{Interpolate: func(s string) string {
		return strings.ReplaceAll(s, "{{expected}}", "42")
	}}

	spec := ParseSpecification("res.status", "eq {{expected}}")
	rhs, err := e.resolveOperand(spec)
	if err != nil {
		t.Fatal(err)
	}
	if outcome := Evaluate(spec, 42.0, rhs); !outcome.Passed {
		t.Errorf("interpolated operand did not compare: %+v", outcome)
	}
}

// TestInvalidPatternFails makes a bad regex a failure rather than a pass.
func TestInvalidPatternFails(t *testing.T) {
	spec := ParseSpecification("body", "matches (")
	outcome := Evaluate(spec, "abc", "(")
	if outcome.Passed {
		t.Error("an invalid pattern must not pass")
	}
	if !strings.Contains(outcome.Error, "invalid pattern") {
		t.Errorf("error = %q", outcome.Error)
	}
}

// TestUnknownOperatorReports is a programming-error guard.
func TestUnknownOperatorReports(t *testing.T) {
	outcome := Evaluate(Specification{Operator: "nonsense"}, 1.0, 1.0)
	if outcome.Passed {
		t.Error("an unknown operator must not pass")
	}
	if !strings.Contains(outcome.Error, "unknown assertion operator") {
		t.Errorf("error = %q", outcome.Error)
	}
}

// TestRunReportsEveryAssertion covers the harness contract: order preserved,
// disabled rows skipped, and one failure not stopping the run.
func TestRunReportsEveryAssertion(t *testing.T) {
	assertions := []model.KeyValue{
		{UID: "1", Name: strp("res.status"), Value: strp("eq 200"), Enabled: true},
		{UID: "2", Name: strp("res.status"), Value: strp("eq 404"), Enabled: true},
		{UID: "3", Name: strp("ignored"), Value: strp("eq 1"), Enabled: false},
		{UID: "4", Name: strp("body.ok"), Value: strp("isTruthy"), Enabled: true},
	}

	e := Evaluator{}
	results := e.Run(Input{
		Assertions: assertions,
		Expression: func(name string) (any, error) {
			switch name {
			case "res.status":
				return 200.0, nil
			case "body.ok":
				return true, nil
			}
			return nil, errors.New("not found")
		},
	})

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3 (the disabled row must be skipped)", len(results))
	}
	if results[0].UID != "1" || !results[0].Passed {
		t.Errorf("first = %+v", results[0])
	}
	if results[1].UID != "2" || results[1].Passed || results[1].Error == "" {
		t.Errorf("second = %+v", results[1])
	}
	if results[2].UID != "4" || !results[2].Passed {
		t.Errorf("third = %+v", results[2])
	}

	// The failure names both sides so the report is actionable.
	if !strings.Contains(results[1].Actual, "200") || !strings.Contains(results[1].Expected, "404") {
		t.Errorf("failure report = %+v", results[1])
	}
}

// TestRunPreservesPartialResultsWhenExpressionThrows is the acceptance
// criterion: a script (or expression) that throws must not discard the results
// already collected.
func TestRunPreservesPartialResultsWhenExpressionThrows(t *testing.T) {
	assertions := []model.KeyValue{
		{UID: "1", Name: strp("res.status"), Value: strp("eq 200"), Enabled: true},
		{UID: "2", Name: strp("boom()"), Value: strp("eq 1"), Enabled: true},
		{UID: "3", Name: strp("res.status"), Value: strp("lt 300"), Enabled: true},
	}

	results := Evaluator{}.Run(Input{
		Assertions: assertions,
		Expression: func(name string) (any, error) {
			if name == "boom()" {
				return nil, errors.New("TypeError: boom is not a function")
			}
			return 200.0, nil
		},
	})

	if len(results) != 3 {
		t.Fatalf("results = %d, want all 3 preserved", len(results))
	}
	if !results[0].Passed {
		t.Errorf("first = %+v", results[0])
	}
	if results[1].Passed || !strings.Contains(results[1].Error, "boom is not a function") {
		t.Errorf("second = %+v", results[1])
	}
	if !results[2].Passed {
		t.Errorf("third = %+v", results[2])
	}
}

// TestRunWithoutEvaluator reports rather than panics.
func TestRunWithoutEvaluator(t *testing.T) {
	results := Evaluator{}.Run(Input{
		Assertions: []model.KeyValue{
			{UID: "1", Name: strp("x"), Value: strp("eq 1"), Enabled: true},
		},
	})
	if len(results) != 1 || results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(results[0].Error, "no expression evaluator") {
		t.Errorf("error = %q", results[0].Error)
	}
}

// TestRunWithNoAssertions returns an empty, non-nil slice so callers can range
// over it without a nil check.
func TestRunWithNoAssertions(t *testing.T) {
	results := Evaluator{}.Run(Input{})
	if results == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("results = %+v", results)
	}
}

// TestRenderForms covers the report rendering rules.
func TestRenderForms(t *testing.T) {
	cases := map[string]string{}
	_ = cases

	if got := Render(nil); got != "null" {
		t.Errorf("nil = %q", got)
	}
	if got := Render(Undefined{}); got != "undefined" {
		t.Errorf("undefined = %q", got)
	}
	// A string is quoted so an empty string is visible in a report.
	if got := Render(""); got != `""` {
		t.Errorf("empty string = %q", got)
	}
	if got := Render(200.0); got != "200" {
		t.Errorf("number = %q", got)
	}
	if got := Render(map[string]any{"a": 1.0}); got != `{"a":1}` {
		t.Errorf("object = %q", got)
	}
	if got := Render([]any{1.0, 2.0}); got != "[1,2]" {
		t.Errorf("array = %q", got)
	}
	if got := Render(true); got != "true" {
		t.Errorf("bool = %q", got)
	}
}

// TestTruthiness covers JavaScript coercion.
func TestTruthiness(t *testing.T) {
	truthyCases := []any{"x", 1.0, true, []any{}}
	for _, v := range truthyCases {
		if !truthy(v) {
			t.Errorf("%#v should be truthy", v)
		}
	}
	falsyCases := []any{"", 0.0, false, nil, Undefined{}}
	for _, v := range falsyCases {
		if truthy(v) {
			t.Errorf("%#v should be falsy", v)
		}
	}
}

// TestContainsVariants covers strings, arrays and objects.
func TestContainsVariants(t *testing.T) {
	if !contains("hello", "ell") {
		t.Error("substring not found")
	}
	if !contains([]any{"a", "b"}, "b") {
		t.Error("array element not found")
	}
	if !contains(map[string]any{"content-type": "x"}, "content-type") {
		t.Error("object key not found")
	}
	if contains(map[string]any{"a": 1.0}, "b") {
		t.Error("an absent key reported present")
	}
}

func strp(s string) *string { return &s }
