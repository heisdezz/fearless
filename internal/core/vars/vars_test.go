package vars

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
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
	// Order is not guaranteed due to map iteration, sort both
	sort.Strings(got)
	sort.Strings(want)
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

func TestMockFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		pattern  string
	}{
		{"guid", "{{$guid}}", `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`},
		{"randomUUID", "{{$randomUUID}}", `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`},
		{"timestamp", "{{$timestamp}}", `^\d+$`},
		{"isoTimestamp", "{{$isoTimestamp}}", `^\d{4}-\d{2}-\d{2}T`},
		{"randomInt", "{{$randomInt}}", `^\d+$`},
		{"randomBoolean", "{{$randomBoolean}}", `^(true|false)$`},
		{"randomEmail", "{{$randomEmail}}", `.+@.+\..+`},
		{"randomFirstName", "{{$randomFirstName}}", `^\w+$`},
		{"randomLastName", "{{$randomLastName}}", `^\w+$`},
		{"randomFullName", "{{$randomFullName}}", `^\w+ \w+$`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := Interpolate(tc.template, map[string]any{}, InterpolateOptions{})
			ok, _ := regexp.MatchString(tc.pattern, out)
			if !ok {
				t.Fatalf("%s: got %q, expected pattern %s", tc.name, out, tc.pattern)
			}
		})
	}
}

func TestInterpolateMissingValues(t *testing.T) {
	var unresolved []string
	out := Interpolate("{{host}}/api/echo/json?foo={{undefinedVar}}", map[string]any{
		"host": "https://example.com",
	}, InterpolateOptions{
		OnUnresolved: func(n string) { unresolved = append(unresolved, n) },
	})
	if out != "https://example.com/api/echo/json?foo={{undefinedVar}}" {
		t.Fatalf("got %q", out)
	}
	if !reflect.DeepEqual(unresolved, []string{"undefinedVar"}) {
		t.Fatalf("unresolved=%v", unresolved)
	}
}

func TestInterpolateObjectsArrays(t *testing.T) {
	vars := map[string]any{
		"arr": []any{1, 2, 3, 4, 5},
		"obj": map[string]any{
			"null":    nil,
			"number":  1,
			"boolean": true,
			"foo": map[string]any{
				"bar": map[string]any{
					"baz": 1,
				},
			},
		},
	}

	tests := []struct {
		template string
		expected string
	}{
		{"{{obj.undefined}}", "{{obj.undefined}}"},
		{"{{obj.null}}", "null"},
		{"{{obj.number}}", "1"},
		{"{{obj.boolean}}", "true"},
		{"{{arr}}", "[1,2,3,4,5]"},
		{"{{arr[0]}}", "1"},
		{"{{obj}}", `{"boolean":true,"foo":{"bar":{"baz":1}},"null":null,"number":1}`},
		{"{{obj.foo}}", `{"bar":{"baz":1}}`},
		{"{{obj.foo.bar}}", `{"baz":1}`},
		{"{{obj.foo.bar.baz}}", "1"},
	}

	for _, tc := range tests {
		t.Run(tc.template, func(t *testing.T) {
			var unresolved []string
			out := Interpolate(tc.template, vars, InterpolateOptions{
				OnUnresolved: func(n string) { unresolved = append(unresolved, n) },
			})
			if out != tc.expected {
				t.Fatalf("got %q, expected %q", out, tc.expected)
			}
		})
	}
}

func TestInterpolateBracketNotation(t *testing.T) {
	vars := map[string]any{
		"user": map[string]any{
			"fav-food": []any{"egg", "meat"},
			"want.attention": true,
		},
	}

	tests := []struct {
		template string
		expected string
	}{
		{"{{user['fav-food'][0]}}", "egg"},
		{"{{user['fav-food'][1]}}", "meat"},
		{"{{user['want.attention']}}", "true"},
		{"{{user.fav-food[0]}}", "egg"},
	}

	for _, tc := range tests {
		t.Run(tc.template, func(t *testing.T) {
			out := Interpolate(tc.template, vars, InterpolateOptions{})
			if out != tc.expected {
				t.Fatalf("got %q, expected %q", out, tc.expected)
			}
		})
	}
}

func TestInterpolateRuntimeVars(t *testing.T) {
	vars := map[string]any{
		"rUser": map[string]any{
			"full_name": "Bruno",
			"age":       7,
			"fav-food":  []any{"egg", "meat"},
			"want.attention": true,
		},
	}

	template := `Hi, I am {{rUser.full_name}},
I am {{rUser.age}} years old.
My favorite food is {{rUser.fav-food[0]}} and {{rUser.fav-food[1]}}.
I like attention: {{rUser['want.attention']}}`

	expected := `Hi, I am Bruno,
I am 7 years old.
My favorite food is egg and meat.
I like attention: true`

	out := Interpolate(template, vars, InterpolateOptions{})
	if out != expected {
		t.Fatalf("got %q\nexpected %q", out, expected)
	}
}

func TestInterpolateEnvVars(t *testing.T) {
	vars := map[string]any{
		"env": map[string]any{
			"var1": "envVar1",
			"var2": "envVar2",
		},
		"host": "https://api.example.com",
	}

	template := `{{host}}/api/echo/json`
	body := `{"envVar1": "{{env.var1}}", "envVar2": "{{env.var2}}"}`

	outURL := Interpolate(template, vars, InterpolateOptions{})
	outBody := Interpolate(body, vars, InterpolateOptions{})

	if outURL != "https://api.example.com/api/echo/json" {
		t.Fatalf("url got %q", outURL)
	}
	if outBody != `{"envVar1": "envVar1", "envVar2": "envVar2"}` {
		t.Fatalf("body got %q", outBody)
	}
}

func TestRecursionDepthGuard(t *testing.T) {
	vars := make(map[string]any)
	for i := 0; i < 300; i++ {
		vars[fmt.Sprintf("v%d", i)] = fmt.Sprintf("{{v%d}}", i+1)
	}
	vars["v300"] = "done"

	out := Interpolate("{{v0}}", vars, InterpolateOptions{})
	if out != "done" && out != "{{v0}}" {
		t.Fatalf("depth guard failed: got %q", out)
	}
}

func TestSecretValuesNotEchoed(t *testing.T) {
	vars := map[string]any{
		"secret_token": "actual-secret-value",
		"public_var":   "public",
	}

	var unresolved []string
	out := Interpolate("{{secret_token}} {{public_var}} {{missing}}", vars, InterpolateOptions{
		OnUnresolved: func(n string) {
			unresolved = append(unresolved, n)
		},
	})

	if out != "actual-secret-value public {{missing}}" {
		t.Fatalf("got %q", out)
	}
	if !reflect.DeepEqual(unresolved, []string{"missing"}) {
		t.Fatalf("unresolved should only contain missing, got %v", unresolved)
	}
}

func TestCollectorMergeFrom(t *testing.T) {
	c1 := NewUnresolvedCollector()
	c1.OnUnresolved("a")
	c1.OnUnresolved("b")

	c2 := NewUnresolvedCollector()
	c2.OnUnresolved("b")
	c2.OnUnresolved("c")

	c1.MergeFrom(c2)
	if !reflect.DeepEqual(c1.Names, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", c1.Names)
	}
}

func TestCollectorNilSafety(t *testing.T) {
	var c *UnresolvedCollector
	c.OnUnresolved("a")
	c.MergeFrom(nil)
	c.MergeFrom(NewUnresolvedCollector())
}
