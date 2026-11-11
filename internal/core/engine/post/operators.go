// Package post evaluates the declarative parts of Bruno's post-response phase.
//
// It covers two things the request file owns rather than a script:
//
//   - `vars:post-response` blocks, which derive variables from the response
//   - `assert` blocks, which compare an expression against an expected value
//     using one of 27 operators
//
// Both are evaluated natively. A `{{var}}` reference is resolved by package
// vars; the assertion's left-hand expression is a JavaScript expression and is
// evaluated by the caller's script runtime, because it can be anything from
// `res.status` to a function call.
//
// Source: bruno-js/src/runtime/assert-runtime.js
package post

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Unary operators take no operand. All others are binary.
//
// Source: bruno-js/src/runtime/assert-runtime.js unaryOperators
var unaryOperators = map[string]bool{
	"isEmpty":     true,
	"isNotEmpty":  true,
	"isNull":      true,
	"isUndefined": true,
	"isDefined":   true,
	"isTruthy":    true,
	"isFalsy":     true,
	"isJson":      true,
	"isNumber":    true,
	"isString":    true,
	"isBoolean":   true,
	"isArray":     true,
}

// binaryOperators are the remaining operators, in the order Bruno declares them.
//
// The count is 16 binary plus 12 unary = 28, matching the operator table in
// assert-runtime.js. TestOperatorListIsComplete pins it.
var binaryOperators = []string{
	"eq", "neq", "gt", "gte", "lt", "lte", "in", "notIn",
	"contains", "notContains", "length", "matches", "notMatches",
	"startsWith", "endsWith", "between",
}

// Operators lists every supported operator, binary first then unary, in the
// order Bruno's documentation comments present them.
func Operators() []string {
	out := make([]string, 0, len(binaryOperators)+len(unaryOperators))
	out = append(out, binaryOperators...)
	for _, name := range []string{
		"isEmpty", "isNotEmpty", "isNull", "isUndefined", "isDefined",
		"isTruthy", "isFalsy", "isJson", "isNumber", "isString",
		"isBoolean", "isArray",
	} {
		out = append(out, name)
	}
	return out
}

// IsUnary reports whether an operator takes no operand.
func IsUnary(operator string) bool { return unaryOperators[operator] }

// Known reports whether an operator is supported.
func Known(operator string) bool {
	if unaryOperators[operator] {
		return true
	}
	for _, name := range binaryOperators {
		if name == operator {
			return true
		}
	}
	return false
}

// Specification is a parsed `assert` row.
type Specification struct {
	// Expression is the left-hand JavaScript expression, e.g. `res.status`.
	Expression string
	// Operator is the parsed operator name.
	Operator string
	// Operand is the right-hand operand text with the operator removed. Empty
	// for unary operators.
	Operand string
	// Raw is the original value column, kept for reporting.
	Raw string
}

// ParseSpecification splits an assertion's value column into operator and
// operand.
//
// The value column holds both, separated by the first space: `eq 200`,
// `isTruthy`, `contains foo bar`. An unrecognised leading word is *not* treated
// as an unknown operator — Bruno falls back to equality against the whole
// string, because `assert { status: 200 }` is a legal shorthand for equality and
// the word `200` is not an operator.
//
// Source: bruno-js/src/runtime/assert-runtime.js parseAssertionOperator
func ParseSpecification(expression, value string) Specification {
	spec := Specification{Expression: expression, Raw: value}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		// An empty value means "equal to the empty string", which is what a
		// user writing `assert { done: eq }` would get if they meant something
		// else and never noticed.
		spec.Operator = "eq"
		return spec
	}

	operator, rest, _ := strings.Cut(trimmed, " ")
	if IsUnary(operator) {
		spec.Operator = operator
		return spec
	}
	if !Known(operator) {
		spec.Operator = "eq"
		spec.Operand = trimmed
		return spec
	}
	spec.Operator = operator
	spec.Operand = strings.TrimSpace(rest)
	return spec
}

// UnaryPredicate decides a unary operator.
type UnaryPredicate func(actual any) bool

// Outcome is the result of evaluating one assertion.
type Outcome struct {
	// Passed reports whether the assertion held.
	Passed bool
	// Actual is the rendered left-hand value, for the report.
	Actual string
	// Expected is the rendered operand, for the report.
	Expected string
	// Error explains the failure, when the comparison itself could not run.
	Error string
}

// Evaluate applies a specification to an already-evaluated left-hand value.
//
// `rhs` is the operand resolved by the caller: a list for `in`/`notIn`, a pair
// for `between`, a pattern string for `matches`, and any value otherwise.
//
// The left value is the *exported* JavaScript value, so a number arrives as
// float64 or int64 depending on how the runtime exported it; comparisons coerce
// rather than type-switch, matching chai's loose equality.
func Evaluate(spec Specification, actual, rhs any) Outcome {
	out := Outcome{Actual: Render(actual), Expected: Render(rhs)}

	if !Known(spec.Operator) {
		out.Error = fmt.Sprintf("unknown assertion operator %q", spec.Operator)
		return out
	}

	if IsUnary(spec.Operator) {
		out.Passed = evaluateUnary(spec.Operator, actual)
		if !out.Passed {
			out.Error = fmt.Sprintf("%s failed", describeUnary(spec.Operator))
		}
		return out
	}

	passed, err := evaluateBinary(spec.Operator, actual, rhs)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Passed = passed
	if !passed {
		out.Error = fmt.Sprintf("expected %s %s %s", out.Actual, describeBinary(spec.Operator), out.Expected)
	}
	return out
}

// evaluateUnary applies a unary operator.
func evaluateUnary(operator string, actual any) bool {
	switch operator {
	case "isEmpty":
		return lengthOf(actual) == 0
	case "isNotEmpty":
		return lengthOf(actual) != 0
	case "isNull":
		return actual == nil
	case "isUndefined", "isDefined":
		// Both values arrive as exported Go values, so a JavaScript `undefined`
		// is indistinguishable from an absent lookup. The caller decides by
		// passing a sentinel; see EvaluateExpression's contract.
		_, isUndefined := actual.(Undefined)
		if operator == "isUndefined" {
			return isUndefined
		}
		return !isUndefined
	case "isTruthy":
		return truthy(actual)
	case "isFalsy":
		return !truthy(actual)
	case "isJson":
		return isJSON(actual)
	case "isNumber":
		return isNumeric(actual)
	case "isString":
		_, ok := actual.(string)
		return ok
	case "isBoolean":
		_, ok := actual.(bool)
		return ok
	case "isArray":
		_, ok := actual.([]any)
		return ok
	}
	return false
}

// Undefined marks a JavaScript `undefined` result.
//
// A Go nil cannot carry the distinction: `res.headers["x"]` on a missing header
// is `undefined`, while `res.getBody()` on an empty body may be `null`, and
// `isUndefined` versus `isNull` must tell them apart.
type Undefined struct{}

// evaluateBinary applies a binary operator.
func evaluateBinary(operator string, actual, rhs any) (bool, error) {
	switch operator {
	case "eq":
		return looseEqual(actual, rhs), nil
	case "neq":
		return !looseEqual(actual, rhs), nil
	case "gt", "gte", "lt", "lte":
		left, ok := asFloat(actual)
		if !ok {
			return false, fmt.Errorf("%s needs a number, got %T", operator, actual)
		}
		right, ok := asFloat(rhs)
		if !ok {
			return false, fmt.Errorf("%s needs a numeric operand, got %T", operator, rhs)
		}
		switch operator {
		case "gt":
			return left > right, nil
		case "gte":
			return left >= right, nil
		case "lt":
			return left < right, nil
		default:
			return left <= right, nil
		}
	case "in", "notIn":
		list, ok := rhs.([]any)
		if !ok {
			return false, fmt.Errorf("%s needs a comma-separated list", operator)
		}
		found := false
		for _, candidate := range list {
			if looseEqual(actual, candidate) {
				found = true
				break
			}
		}
		if operator == "notIn" {
			return !found, nil
		}
		return found, nil
	case "contains", "notContains":
		found := contains(actual, rhs)
		if operator == "notContains" {
			return !found, nil
		}
		return found, nil
	case "length":
		// The operand is a number, and JS coerces the character count too.
		want, ok := asFloat(rhs)
		if !ok {
			return false, fmt.Errorf("length needs a numeric operand, got %T", rhs)
		}
		// A non-collection has length 0 rather than an error, matching chai's
		// `lengthOf` on a value with no length property.
		return lengthOf(actual) == int(want), nil
	case "matches", "notMatches":
		pattern, ok := rhs.(string)
		if !ok {
			return false, fmt.Errorf("%s needs a pattern", operator)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		matched := re.MatchString(textOf(actual))
		if operator == "notMatches" {
			return !matched, nil
		}
		return matched, nil
	case "startsWith":
		return strings.HasPrefix(textOf(actual), textOf(rhs)), nil
	case "endsWith":
		return strings.HasSuffix(textOf(actual), textOf(rhs)), nil
	case "between":
		pair, ok := rhs.([]any)
		if !ok || len(pair) != 2 {
			return false, fmt.Errorf("between needs exactly two operands")
		}
		value, ok := asFloat(actual)
		if !ok {
			return false, fmt.Errorf("between needs a number, got %T", actual)
		}
		low, lowOK := asFloat(pair[0])
		high, highOK := asFloat(pair[1])
		if !lowOK || !highOK {
			return false, fmt.Errorf("between needs numeric bounds")
		}
		// Inclusive on both ends, matching chai's `within`: `between 1,5`
		// passes for 1 and for 5.
		return value >= low && value <= high, nil
	}
	return false, fmt.Errorf("unhandled operator %q", operator)
}

// looseEqual compares two exported values.
//
// Numeric representations of the same number are equal, because a JavaScript
// number exports as float64 or int64 depending on the runtime path and
// `expect(res.status).to.equal(200)` must keep working. A string almost never
// equals a number: chai's `equal` is strict, and treating `"200"` as 200 would
// make a stringly-typed response body pass a numeric assertion.
func looseEqual(actual, rhs any) bool {
	if actual == nil || rhs == nil {
		return actual == nil && rhs == nil
	}

	// Strings and booleans compare only within their own kind, and must be
	// checked before the numeric path so a numeric string cannot fall through.
	if left, ok := actual.(string); ok {
		right, ok := rhs.(string)
		return ok && left == right
	}
	if left, ok := actual.(bool); ok {
		right, ok := rhs.(bool)
		return ok && left == right
	}
	if _, isString := rhs.(string); isString {
		return false
	}
	if _, isBool := rhs.(bool); isBool {
		return false
	}

	if left, ok := asFloat(actual); ok {
		if right, ok := asFloat(rhs); ok {
			return left == right
		}
		return false
	}

	// Structurally compare the rest, so an object compares to an equal object.
	leftJSON, errLeft := json.Marshal(actual)
	rightJSON, errRight := json.Marshal(rhs)
	if errLeft != nil || errRight != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}

// contains reports substring or element membership.
func contains(actual, rhs any) bool {
	if list, ok := actual.([]any); ok {
		for _, item := range list {
			if looseEqual(item, rhs) {
				return true
			}
		}
		return false
	}
	if obj, ok := actual.(map[string]any); ok {
		// `contains` on an object checks keys, which is how a header map is
		// probed.
		_, present := obj[textOf(rhs)]
		return present
	}
	return strings.Contains(textOf(actual), textOf(rhs))
}

// lengthOf returns a string or array length, or 0.
func lengthOf(actual any) int {
	switch value := actual.(type) {
	case string:
		return len(value)
	case []any:
		return len(value)
	case map[string]any:
		return len(value)
	}
	return 0
}

// isJSON reports whether a value is an object or array.
//
// Source: bruno-js/src/runtime/assert-runtime.js (the custom `json` chai property)
func isJSON(actual any) bool {
	switch actual.(type) {
	case map[string]any, []any:
		return true
	}
	return false
}

// truthy applies JavaScript truthiness.
func truthy(actual any) bool {
	switch value := actual.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case Undefined:
		return false
	}
	if number, ok := asFloat(actual); ok {
		return number != 0 && !math.IsNaN(number)
	}
	return true
}

// isNumeric reports whether a value is a number.
//
// A numeric *string* is not one. `isNumber` is a type check, so coercing here
// would report `"1"` as a number and make the assertion useless for telling a
// stringly-typed body from a real number.
func isNumeric(v any) bool {
	switch v.(type) {
	case float64, float32, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	}
	return false
}

// asFloat converts a value to float64 for relational comparison.
//
// A numeric string is accepted because JavaScript coerces it for `>` and `<`:
// `"6" > 5` is true. Equality is checked before this is reached, so the coercion
// never makes `"200"` equal the number 200.
func asFloat(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint8:
		return float64(value), true
	case uint16:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	case json.Number:
		f, err := value.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case string:
		// JavaScript coerces a numeric string for relational operators, so
		// `gt 5` against "6" passes. Equality does not, which looseEqual
		// already handles by checking strings first.
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// textOf renders a value as its string form for substring and pattern tests.
func textOf(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case Undefined:
		return "undefined"
	}
	return Render(v)
}

// Render produces a stable human-readable form for reports.
//
// Strings are quoted so an empty string is visible in a failure message, and
// objects render as JSON so a comparison failure shows what was compared.
func Render(v any) string {
	switch value := v.(type) {
	case nil:
		return "null"
	case Undefined:
		return "undefined"
	case string:
		return strconv.Quote(value)
	}
	if encoded, err := json.Marshal(v); err == nil {
		return string(encoded)
	}
	return fmt.Sprint(v)
}

// describeUnary returns the human description of a unary operator.
func describeUnary(operator string) string {
	switch operator {
	case "isEmpty":
		return "be empty"
	case "isNotEmpty":
		return "not be empty"
	case "isNull":
		return "be null"
	case "isUndefined":
		return "be undefined"
	case "isDefined":
		return "be defined"
	case "isTruthy":
		return "be truthy"
	case "isFalsy":
		return "be falsy"
	case "isJson":
		return "be JSON"
	case "isNumber":
		return "be a number"
	case "isString":
		return "be a string"
	case "isBoolean":
		return "be a boolean"
	case "isArray":
		return "be an array"
	}
	return operator
}

// describeBinary returns the human description of a binary operator.
func describeBinary(operator string) string {
	switch operator {
	case "eq":
		return "to equal"
	case "neq":
		return "not to equal"
	case "gt":
		return "to be greater than"
	case "gte":
		return "to be at least"
	case "lt":
		return "to be less than"
	case "lte":
		return "to be at most"
	case "in":
		return "to be one of"
	case "notIn":
		return "not to be one of"
	case "contains":
		return "to contain"
	case "notContains":
		return "not to contain"
	case "length":
		return "to have length"
	case "matches":
		return "to match"
	case "notMatches":
		return "not to match"
	case "startsWith":
		return "to start with"
	case "endsWith":
		return "to end with"
	case "between":
		return "to be between"
	}
	return operator
}
