package vars

import (
	"reflect"
	"regexp"
	"testing"
)

func TestInterpolateBasic(t *testing.T) {
	out := Interpolate("Hello, my name is {{user.name}} and I am {{user.age}} years old", map[string]any{
		"user.name": "Bruno",
		"user":      map[string]any{"age": 4},
	}, InterpolateOptions{})
	if out != "Hello, my name is Bruno and I am 4 years old" {
		t.Fatalf("got %q", out)
	}
}

func TestInterpolateMissingKeepsPlaceholder(t *testing.T) {
	var unresolved []string
	out := Interpolate("Hello {{name}} {{missing}}", map[string]any{"name": "Bruno"}, InterpolateOptions{
		OnUnresolved: func(n string) { unresolved = append(unresolved, n) },
	})
	if out != "Hello Bruno {{missing}}" {
		t.Fatalf("got %q", out)
	}
	if !reflect.DeepEqual(unresolved, []string{"missing"}) {
		t.Fatalf("unresolved=%v", unresolved)
	}
}

func TestInterpolateRecursiveAndCycle(t *testing.T) {
	out := Interpolate("{{a}}", map[string]any{"a": "{{b}}", "b": "{{c}}", "c": "ok"}, InterpolateOptions{})
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
	cycle := Interpolate("{{x}}", map[string]any{"x": "{{x}}"}, InterpolateOptions{})
	if cycle != "{{x}}" {
		t.Fatalf("cycle got %q", cycle)
	}
}

func TestInterpolateObjectStringify(t *testing.T) {
	out := Interpolate("{{obj}}", map[string]any{"obj": map[string]any{"a": 1}}, InterpolateOptions{})
	if out != "{\"a\":1}" && out != "{\"a\":1.0}" {
		// encoding/json may render numbers differently depending on input type
		if out != `{"a":1}` {
			t.Fatalf("got %q", out)
		}
	}
}

func TestInterpolateJSONStringEscaping(t *testing.T) {
	out := Interpolate("{{v}}", map[string]any{"v": "a\n\"b\""}, InterpolateOptions{EscapeJSON: true})
	if out != `a\n\"b\"` {
		t.Fatalf("got %q", out)
	}
}

func TestLookupPaths(t *testing.T) {
	vars := map[string]any{
		"user": map[string]any{"name": "Bruno", "arr": []any{"x", "y"}},
	}
	if v, ok := Lookup(vars, "user.name"); !ok || v != "Bruno" {
		t.Fatalf("lookup user.name = %v %v", v, ok)
	}
	if v, ok := Lookup(vars, "user.arr[1]"); !ok || v != "y" {
		t.Fatalf("lookup arr[1] = %v %v", v, ok)
	}
}

func TestPromptExtraction(t *testing.T) {
	got := ExtractPromptVariables(map[string]any{
		"a": "Hello {{?Name}} and {{?Name}}",
		"b": []any{"{{?Other}}"},
	})
	want := []string{"Name", "Other"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestScopesPrecedence(t *testing.T) {
	s := Scopes{
		Global:      map[string]any{"k": "g", "g": "1"},
		Collection:  map[string]any{"k": "c"},
		Environment: map[string]any{"k": "e"},
		Folder:      map[string]any{"k": "f"},
		Request:     map[string]any{"k": "r"},
		OAuth2:      map[string]any{"k": "o"},
		Runtime:     map[string]any{"k": "rt"},
		Prompt:      map[string]any{"k": "p"},
		ProcessEnv:  map[string]string{"HOME": "/x"},
	}
	m := s.Merge()
	if m["k"] != "p" {
		t.Fatalf("precedence got %v", m["k"])
	}
	if Interpolate("{{process.env.HOME}}", m, InterpolateOptions{}) != "/x" {
		t.Fatalf("process.env lookup failed: %v", Interpolate("{{process.env.HOME}}", m, InterpolateOptions{}))
	}
}

func TestMockISO(t *testing.T) {
	out := Interpolate("{{$isoTimestamp}}", map[string]any{}, InterpolateOptions{})
	ok, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T`, out)
	if !ok {
		t.Fatalf("not iso-ish: %q", out)
	}
}
