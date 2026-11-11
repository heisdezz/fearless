package post

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLooseEqualMatrix covers every comparison branch directly, since the
// operator tests only reach a few of them.
func TestLooseEqualMatrix(t *testing.T) {
	cases := []struct {
		name   string
		left   any
		right  any
		expect bool
	}{
		{"both nil", nil, nil, true},
		{"nil vs value", nil, 1.0, false},
		{"value vs nil", 1.0, nil, false},

		// Numeric representations of one number are equal.
		{"float vs int", 200.0, int(200), true},
		{"int64 vs float32", int64(5), float32(5), true},
		{"uint vs int", uint64(7), 7.0, true},
		{"json number", json.Number("9"), 9.0, true},
		{"different numbers", 1.0, 2.0, false},

		// A string only equals a string.
		{"same strings", "a", "a", true},
		{"different strings", "a", "b", false},
		{"numeric string vs number", "200", 200.0, false},
		{"number vs numeric string", 200.0, "200", false},
		{"string vs bool", "true", true, false},

		// Booleans only equal booleans.
		{"both true", true, true, true},
		{"true vs false", true, false, false},
		{"false vs number", false, 0.0, false},

		// Containers compare structurally.
		{"equal objects", map[string]any{"a": 1.0}, map[string]any{"a": 1.0}, true},
		{"different objects", map[string]any{"a": 1.0}, map[string]any{"a": 2.0}, false},
		{"equal arrays", []any{1.0, 2.0}, []any{1.0, 2.0}, true},
		{"different arrays", []any{1.0}, []any{2.0}, false},
		{"object vs array", map[string]any{"a": 1.0}, []any{1.0}, false},
		// An unserialisable value cannot be compared structurally.
		{"unsupported type", make(chan int), make(chan int), false},
	}

	for _, tc := range cases {
		if got := looseEqual(tc.left, tc.right); got != tc.expect {
			t.Errorf("%s: looseEqual(%#v, %#v) = %v, want %v", tc.name, tc.left, tc.right, got, tc.expect)
		}
	}
}

// TestAsFloatConversions covers every numeric type and the string fallback.
func TestAsFloatConversions(t *testing.T) {
	numeric := []any{
		float64(1.5), float32(1.5), int(1), int8(1), int16(1), int32(1),
		int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1),
		json.Number("1.5"),
	}
	for _, v := range numeric {
		if _, ok := asFloat(v); !ok {
			t.Errorf("%T should convert", v)
		}
	}

	// A numeric string converts, which is what relational operators need.
	if got, ok := asFloat(" 6 "); !ok || got != 6 {
		t.Errorf("numeric string = %v, %v", got, ok)
	}
	// A non-numeric string does not.
	if _, ok := asFloat("abc"); ok {
		t.Error("a non-numeric string must not convert")
	}
	// An invalid json.Number does not.
	if _, ok := asFloat(json.Number("nope")); ok {
		t.Error("an invalid json.Number must not convert")
	}
}

// TestLengthOfVariants covers strings, arrays, objects and scalars.
func TestLengthOfVariants(t *testing.T) {
	cases := map[string]struct {
		value any
		want  int
	}{
		"string":       {"abc", 3},
		"empty string": {"", 0},
		"array":        {[]any{1.0, 2.0}, 2},
		"object":       {map[string]any{"a": 1.0, "b": 2.0}, 2},
		"number":       {1.0, 0},
		"nil":          {nil, 0},
	}
	for name, tc := range cases {
		if got := lengthOf(tc.value); got != tc.want {
			t.Errorf("%s: lengthOf = %d, want %d", name, got, tc.want)
		}
	}

	// The length operator reads the same lengths.
	outcome := Evaluate(ParseSpecification("body", "length 2"), map[string]any{"a": 1.0, "b": 2.0}, 2.0)
	if !outcome.Passed {
		t.Errorf("object length should be 2: %+v", outcome)
	}
}

// TestTextOfVariants covers the substring conversions.
func TestTextOfVariants(t *testing.T) {
	cases := map[string]struct {
		value any
		want  string
	}{
		"string":    {"abc", "abc"},
		"nil":       {nil, ""},
		"undefined": {Undefined{}, "undefined"},
		"number":    {42.0, "42"},
		"bool":      {true, "true"},
	}
	for name, tc := range cases {
		if got := textOf(tc.value); got != tc.want {
			t.Errorf("%s: textOf = %q, want %q", name, got, tc.want)
		}
	}
}

// TestLiteralForms covers every accepted operand form.
func TestLiteralForms(t *testing.T) {
	cases := []struct {
		text string
		want any
	}{
		{"", ""},
		{"   ", ""},
		{"true", true},
		{"false", false},
		{"null", nil},
		{"undefined", Undefined{}},
		{"42", 42.0},
		{"-1.5", -1.5},
		{`"200"`, "200"},
		{`'200'`, "200"},
		{`"has space"`, "has space"},
		{"hello", "hello"},
		{"hello world", "hello world"},
		// Single characters that are not quotes stay text.
		{`"`, `"`},
	}
	for _, tc := range cases {
		got := literal(tc.text)
		if Render(got) != Render(tc.want) {
			t.Errorf("literal(%q) = %#v, want %#v", tc.text, got, tc.want)
		}
	}
}

// TestDescriptionsAreComplete ensures every operator has a readable failure
// message, since the message is what a user sees.
func TestDescriptionsAreComplete(t *testing.T) {
	for _, op := range Operators() {
		var describe string
		if IsUnary(op) {
			describe = describeUnary(op)
		} else {
			describe = describeBinary(op)
		}
		if describe == op {
			t.Errorf("%q has no description, so a failure would read as the bare operator", op)
		}
		if strings.TrimSpace(describe) == "" {
			t.Errorf("%q has an empty description", op)
		}
	}
}

// TestNumericOperatorTypeErrors covers the paths where an operand cannot be
// compared, which must fail rather than panic or pass.
func TestNumericOperatorTypeErrors(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		actual any
		rhs    any
	}{
		{"gt with a non-number", "gt 5", "abc", 5.0},
		{"gt with a non-numeric operand", "gt abc", 1.0, "abc"},
		{"length with a non-numeric operand", "length abc", "x", "abc"},
		{"between with one operand", "between 1", 1.0, []any{1.0}},
		{"between with a non-number", "between 1,5", "x", []any{1.0, 5.0}},
		{"between with non-numeric bounds", "between a,b", 1.0, []any{"a", "b"}},
		{"in with a non-list", "in a", "a", "a"},
		{"matches with a non-string pattern", "matches 1", "x", 1.0},
	}
	for _, tc := range cases {
		outcome := Evaluate(ParseSpecification("lhs", tc.value), tc.actual, tc.rhs)
		if outcome.Passed {
			t.Errorf("%s: must not pass", tc.name)
		}
		if outcome.Error == "" {
			t.Errorf("%s: must explain the failure", tc.name)
		}
	}
}
