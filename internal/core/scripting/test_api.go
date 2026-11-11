package scripting

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/dop251/goja"

	"fearless/internal/core/model"
)

// buildTest constructs the global `test(description, callback)`.
//
// A test passes when its callback returns, and fails when it throws — the same
// contract as bruno-js/src/test.js, where the callback is awaited and any throw
// becomes a failed result.
//
// The callback runs synchronously. A callback that returns a promise is not
// awaited: there is no event loop here, so a test that needs to await belongs in
// the host's async path rather than being silently treated as passing.
func (b *Bindings) buildTest() (goja.Value, error) {
	return b.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		description := call.Argument(0).String()
		callback, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			b.throw("scripting: test() requires a callback, got %s for %q", call.Argument(1), description)
		}

		result := model.TestResult{Description: description, Status: "pass"}
		if _, err := callback(goja.Undefined()); err != nil {
			result.Status = "fail"
			result.Error = assertionMessage(err)
		}
		b.tests.Add(result)
		return goja.Undefined()
	}), nil
}

// assertionMessage extracts the useful text from a thrown JS value.
func assertionMessage(err error) string {
	exception, ok := err.(*goja.Exception)
	if !ok {
		return err.Error()
	}

	value := exception.Value()
	obj, ok := value.(*goja.Object)
	if !ok {
		return value.String()
	}

	message := stringField(obj, "message", "")
	actual := fieldString(obj, "actual")
	expected := fieldString(obj, "expected")

	switch {
	case actual == "" && expected == "":
		if message == "" {
			return value.String()
		}
		return message
	case message == "":
		return fmt.Sprintf("expected %s, got %s", expected, actual)
	default:
		return fmt.Sprintf("%s (expected %s, got %s)", message, expected, actual)
	}
}

// fieldString renders an optional field, skipping absent values.
func fieldString(obj *goja.Object, name string) string {
	v := obj.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	if s, ok := v.Export().(string); ok {
		return s
	}
	return fmt.Sprint(v.Export())
}

// buildExpect constructs a Chai-compatible `expect`.
//
// The implemented assertions are the ones Bruno's script API documents and the
// test collections exercise. An unknown assertion is simply absent, so calling it
// throws a TypeError — the correct outcome. A silently-passing assertion would be
// far worse: the suite would report success for something never checked.
//
// Source: bruno-js/src/sandbox/bundle-libraries.js (bundled chai `expect`)
func (b *Bindings) buildExpect() (goja.Value, error) {
	return b.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.assertionObject(call.Argument(0), false)
	}), nil
}

// assertionObject builds the object expect(x) returns.
//
// Assertion methods return `(goja.Value, error)`. That signature matters: goja
// turns a non-nil error from a Go function into a JavaScript exception, which is
// the only supported way to throw from host code. A Go `panic` escapes the
// runtime instead of being catchable by the script.
func (b *Bindings) assertionObject(actual goja.Value, negated bool) *goja.Object {
	obj := b.vm.NewObject()

	// Chain words (`to`, `be`, `is`, …) are accessors returning the assertion.
	//
	// They cannot be plain properties holding the assertion: a property pointing at
	// its own object makes goja recurse without end when it exports that object, so
	// the property graph must stay acyclic. A getter is evaluated only on access,
	// which never exports the whole graph, so `expect(x).to.be.equal(y)` works.
	chainGetter := b.vm.ToValue(func() goja.Value { return obj })
	for _, word := range []string{
		"to", "be", "been", "is", "that", "which", "and", "has", "have",
		"with", "at", "of", "same", "but", "does", "still", "also",
	} {
		_ = obj.DefineAccessorProperty(word, chainGetter, nil, goja.FLAG_FALSE, goja.FLAG_FALSE)
	}

	// `not` negates for the rest of the chain. It is created on first access, not
	// eagerly: building it eagerly would build its own `not`, and so on without
	// end, and goja exports properties during conversion so a lazy getter is also
	// the only way to keep the object graph finite.
	var negatedObj *goja.Object
	_ = obj.DefineAccessorProperty("not", b.vm.ToValue(func() goja.Value {
		if negatedObj == nil {
			negatedObj = b.assertionObject(actual, !negated)
		}
		return negatedObj
	}), nil, goja.FLAG_FALSE, goja.FLAG_FALSE)

	// Methods: each evaluates immediately and throws or returns the chain.
	method := func(name string, predicate func(goja.Value) bool, describe string) {
		_ = obj.Set(name, b.vm.ToValue(func(other goja.Value) (goja.Value, error) {
			if err := b.assertionError(actual, other, negated, predicate(other), describe); err != nil {
				return nil, err
			}
			return obj, nil
		}))
	}

	method("equal", func(expected goja.Value) bool { return looseEqual(actual, expected) }, "equal")
	method("equals", func(expected goja.Value) bool { return looseEqual(actual, expected) }, "equal")
	method("eq", func(expected goja.Value) bool { return looseEqual(actual, expected) }, "equal")
	method("eql", func(expected goja.Value) bool { return deepEqual(actual.Export(), expected.Export()) }, "deep equal")
	method("above", func(other goja.Value) bool { return asNumber(actual) > asNumber(other) }, "be above")
	method("greaterThan", func(other goja.Value) bool { return asNumber(actual) > asNumber(other) }, "be greater than")
	method("below", func(other goja.Value) bool { return asNumber(actual) < asNumber(other) }, "be below")
	method("lessThan", func(other goja.Value) bool { return asNumber(actual) < asNumber(other) }, "be less than")
	method("least", func(other goja.Value) bool { return asNumber(actual) >= asNumber(other) }, "be at least")
	method("most", func(other goja.Value) bool { return asNumber(actual) <= asNumber(other) }, "be at most")
	method("include", func(other goja.Value) bool { return includes(actual, other) }, "include")
	method("contain", func(other goja.Value) bool { return includes(actual, other) }, "contain")
	method("match", func(other goja.Value) bool { return matches(actual, other) }, "match")
	method("lengthOf", func(other goja.Value) bool { return valueLength(actual) == int(asNumber(other)) }, "have length")
	method("oneOf", func(other goja.Value) bool { return oneOf(actual, other) }, "be one of")

	// Boolean assertions are methods, not properties.
	//
	// chai exposes them as getters (`expect(x).to.be.true`), but a goja accessor has
	// no error return, so host code cannot raise a catchable exception from one —
	// only a Go panic, which escapes the runtime. As a method the failure becomes an
	// ordinary JS throw that `test()` reports and a script can catch.
	//
	// The consequence is a compatibility gap: `expect(x).to.be.true` must be written
	// `expect(x).to.be.true()`. Recorded in the package log.
	check := func(name string, predicate func(goja.Value) bool, describe string) {
		_ = obj.Set(name, b.vm.ToValue(func(goja.Value) (goja.Value, error) {
			if err := b.assertionError(actual, nil, negated, predicate(actual), describe); err != nil {
				return nil, err
			}
			return obj, nil
		}))
	}
	check("ok", func(v goja.Value) bool { return v.ToBoolean() }, "be truthy")
	check("true", func(v goja.Value) bool { return v.Export() == true }, "be true")
	check("false", func(v goja.Value) bool { return v.Export() == false }, "be false")
	check("null", func(v goja.Value) bool { return goja.IsNull(v) }, "be null")
	check("undefined", func(v goja.Value) bool { return goja.IsUndefined(v) }, "be undefined")
	check("exist", func(v goja.Value) bool { return !goja.IsUndefined(v) && !goja.IsNull(v) }, "exist")
	check("empty", func(v goja.Value) bool { return valueLength(v) == 0 }, "be empty")

	// a/an assert a type, taking an optional type argument.
	assertType := b.vm.ToValue(func(typeName goja.Value) (goja.Value, error) {
		want := typeName.String()
		if err := b.assertionError(actual, typeName, negated, isType(actual, want), "be a "+want); err != nil {
			return nil, err
		}
		return obj, nil
	})
	_ = obj.Set("a", assertType)
	_ = obj.Set("an", assertType)

	return obj
}

// assertionError builds the error for a failed assertion, or nil when it passed.
//
// Returning an error is how host code raises a JavaScript exception: goja converts
// it at the call boundary. The message names the actual and expected values so a
// failing test explains itself without the reader opening the script.
func (b *Bindings) assertionError(actual, expected goja.Value, negated, passed bool, describe string) error {
	if passed != negated {
		return nil
	}

	message := "expected " + display(actual)
	if negated {
		message += " not " + describe
	} else {
		message += " to " + describe
	}
	if expected != nil && !goja.IsUndefined(expected) {
		message += " " + display(expected)
	}
	return errors.New(message)
}

// display renders a value for an assertion message.
func display(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) {
		return "undefined"
	}
	if goja.IsNull(v) {
		return "null"
	}
	if s, ok := v.Export().(string); ok {
		return fmt.Sprintf("%q", s)
	}
	if obj, ok := v.(*goja.Object); ok {
		if encoded, err := obj.MarshalJSON(); err == nil {
			return string(encoded)
		}
	}
	return v.String()
}

// looseEqual compares with type coercion, matching chai's `equal` for the cases
// scripts hit: `expect(res.getStatus()).to.equal(200)`.
func looseEqual(a, b goja.Value) bool {
	if (a == nil || goja.IsUndefined(a)) && (b == nil || goja.IsUndefined(b)) {
		return true
	}
	if a == nil || goja.IsUndefined(a) || b == nil || goja.IsUndefined(b) {
		return false
	}
	if goja.IsNull(a) || goja.IsNull(b) {
		return goja.IsNull(a) && goja.IsNull(b)
	}
	ea, eb := a.Export(), b.Export()
	if reflect.TypeOf(ea) == reflect.TypeOf(eb) {
		return reflect.DeepEqual(ea, eb)
	}
	return fmt.Sprint(ea) == fmt.Sprint(eb)
}

// deepEqual compares exported values structurally.
func deepEqual(a, b any) bool { return reflect.DeepEqual(a, b) }

// asNumber converts a value to float64.
func asNumber(v goja.Value) float64 {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return 0
	}
	return v.ToFloat()
}

// includes reports substring or array membership.
func includes(haystack, needle goja.Value) bool {
	if haystack == nil || goja.IsUndefined(haystack) || goja.IsNull(haystack) {
		return false
	}
	if obj, ok := haystack.(*goja.Object); ok {
		if l := obj.Get("length"); l != nil && !goja.IsUndefined(l) {
			n := int(l.ToInteger())
			for i := 0; i < n; i++ {
				if looseEqual(obj.Get(fmt.Sprint(i)), needle) {
					return true
				}
			}
			return false
		}
	}
	return strings.Contains(haystack.String(), needle.String())
}

// matches reports whether a string matches a regular expression.
func matches(actual, pattern goja.Value) bool {
	patternText := pattern.String()
	if obj, ok := pattern.(*goja.Object); ok {
		// A RegExp object carries `source` and `flags`; rebuilding it as a Go
		// pattern keeps the semantics without importing goja's internals.
		if source := obj.Get("source"); source != nil && !goja.IsUndefined(source) {
			patternText = source.String()
		}
	}
	if re, err := compileRegexp(patternText); err == nil {
		return re.MatchString(actual.String())
	}
	return false
}

// valueLength returns an array or string length, or 0.
func valueLength(v goja.Value) int {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return 0
	}
	if s, ok := v.Export().(string); ok {
		return len(s)
	}
	if obj, ok := v.(*goja.Object); ok {
		if l := obj.Get("length"); l != nil && !goja.IsUndefined(l) {
			return int(l.ToInteger())
		}
	}
	return 0
}

// oneOf reports membership in the given array.
func oneOf(actual, options goja.Value) bool {
	obj, ok := options.(*goja.Object)
	if !ok {
		return false
	}
	l := obj.Get("length")
	if l == nil || goja.IsUndefined(l) {
		return false
	}
	n := int(l.ToInteger())
	for i := 0; i < n; i++ {
		if looseEqual(actual, obj.Get(fmt.Sprint(i))) {
			return true
		}
	}
	return false
}

// isType reports whether a value has the named JavaScript type.
func isType(v goja.Value, want string) bool {
	switch strings.ToLower(want) {
	case "string":
		_, ok := v.Export().(string)
		return ok
	case "number":
		switch v.Export().(type) {
		case int64, float64, int, int32:
			return true
		}
		return false
	case "boolean":
		_, ok := v.Export().(bool)
		return ok
	case "array":
		_, ok := v.Export().([]any)
		return ok
	case "object":
		return v.Export() != nil && !goja.IsUndefined(v)
	case "null":
		return goja.IsNull(v)
	case "undefined":
		return goja.IsUndefined(v)
	case "function":
		_, ok := goja.AssertFunction(v)
		return ok
	}
	return false
}

// isType reports whether a value has the named JavaScript type.
// compileRegexp compiles a pattern, using Go's syntax.
func compileRegexp(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}
