package post

import (
	"fmt"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// Evaluator resolves operands and runs a set of assertions.
type Evaluator struct {
	// Interpolate resolves {{var}} in an operand.
	Interpolate func(string) string
}

// Input is everything one assertion run needs.
type Input struct {
	// Assertions are the request's `assert` rows.
	Assertions []model.KeyValue
	// Expression evaluates the left-hand side, given the assertion's name.
	//
	// It is supplied by the caller because the expression is JavaScript and may
	// call into `bru`, `req` or `res`; this package deliberately does not own a
	// script runtime.
	Expression func(string) (any, error)
}

// Run evaluates every enabled assertion, preserving source order.
//
// A failing assertion does not stop the run: every row is reported, because a
// user fixing assertions wants the whole list, not the first problem. An
// expression that cannot be evaluated at all is reported as a failed row whose
// error explains why, rather than aborting.
//
// Source: bruno-js/src/runtime/assert-runtime.js runAssertions
func (e Evaluator) Run(in Input) []model.AssertionResult {
	results := make([]model.AssertionResult, 0, len(in.Assertions))

	for _, kv := range in.Assertions {
		if !kv.Enabled {
			continue
		}
		results = append(results, e.runOne(kv, in.Expression))
	}
	return results
}

// runOne evaluates a single assertion row.
func (e Evaluator) runOne(kv model.KeyValue, evalExpression func(string) (any, error)) model.AssertionResult {
	name := kv.NameOrEmpty()
	spec := ParseSpecification(name, kv.ValueOrEmpty())

	result := model.AssertionResult{
		UID:        kv.UID,
		Expression: name,
		Operator:   spec.Operator,
		Expected:   spec.Operand,
	}

	if evalExpression == nil {
		result.Error = "no expression evaluator supplied"
		return result
	}

	actual, err := evalExpression(name)
	if err != nil {
		result.Actual = ""
		result.Error = fmt.Sprintf("cannot evaluate %q: %v", name, err)
		return result
	}

	rhs, err := e.resolveOperand(spec)
	if err != nil {
		result.Actual = Render(actual)
		result.Error = err.Error()
		return result
	}

	outcome := Evaluate(spec, actual, rhs)
	result.Actual = outcome.Actual
	if spec.Operand != "" {
		// The rendered operand is more useful than the raw text when the
		// operand was a {{var}}.
		result.Expected = outcome.Expected
	}
	result.Passed = outcome.Passed
	result.Error = outcome.Error
	return result
}

// resolveOperand turns the operand text into a typed value.
//
// Three operators take structured operands, and their syntax is forgiving in a
// specific way: `in` accepts `a,b` or `[a, b]`, `between` takes two comma-
// separated values, and `matches` accepts `/pattern/` or `pattern`. Reproducing
// that tolerance matters because collections in the wild use both spellings.
//
// Source: bruno-js/src/runtime/assert-runtime.js evaluateRhsOperand
func (e Evaluator) resolveOperand(spec Specification) (any, error) {
	if IsUnary(spec.Operator) {
		return nil, nil
	}

	operand := e.interpolate(spec.Operand)

	switch spec.Operator {
	case "in", "notIn":
		trimmed := strings.TrimSpace(operand)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			trimmed = trimmed[1 : len(trimmed)-1]
		}
		var out []any
		for _, part := range strings.Split(trimmed, ",") {
			out = append(out, literal(strings.TrimSpace(part)))
		}
		return out, nil

	case "between":
		parts := strings.Split(operand, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("between needs two comma-separated values, got %q", spec.Operand)
		}
		return []any{
			literal(strings.TrimSpace(parts[0])),
			literal(strings.TrimSpace(parts[1])),
		}, nil

	case "matches", "notMatches":
		trimmed := strings.TrimSpace(operand)
		if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "/") && strings.HasSuffix(trimmed, "/") {
			trimmed = trimmed[1 : len(trimmed)-1]
		}
		return trimmed, nil

	default:
		return literal(operand), nil
	}
}

// interpolate resolves {{var}} in an operand.
func (e Evaluator) interpolate(s string) string {
	if e.Interpolate == nil {
		return s
	}
	return e.Interpolate(s)
}

// literal converts operand text into a typed value.
//
// This mirrors Bruno's evaluateJsTemplateLiteral, and the mirroring is what makes
// `eq 200` work: the operand is untyped text, so it is coerced the way a template
// literal would be. Accepted forms:
//
//   - true, false, null, undefined
//   - double- or single-quoted text, which stays text even when numeric
//   - a numeric string, up to JavaScript's safe-integer bound; beyond that the
//     text is kept because the conversion would lose the value
//   - anything else stays text
//
// Source: bruno-js/src/utils.js evaluateJsTemplateLiteral
func literal(text string) any {
	trimmed := strings.TrimSpace(text)
	switch trimmed {
	case "":
		return ""
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	case "undefined":
		return Undefined{}
	}

	// Quoted text is text on purpose, even if it looks numeric.
	if len(trimmed) >= 2 {
		if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			return trimmed[1 : len(trimmed)-1]
		}
		if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			return trimmed[1 : len(trimmed)-1]
		}
	}

	if number, ok := parseSafeNumber(trimmed); ok {
		return number
	}

	return trimmed
}

// parseSafeNumber parses a numeric string within JavaScript's safe-integer range.
func parseSafeNumber(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	// Above 2^53 the conversion is lossy, so the text is kept instead; Bruno does
	// the same to avoid corrupting a long identifier that happens to be digits.
	const maxSafeInteger = 1<<53 - 1
	if f > maxSafeInteger || f < -maxSafeInteger {
		return 0, false
	}
	return f, true
}
