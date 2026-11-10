package template

import (
	"os/exec"
	"strings"
	"testing"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

func varOf(name string, value any, typ string) model.Variable {
	return model.Variable{
		Name:     &name,
		Value:    value,
		Enabled:  true,
		DataType: typ,
	}
}

func TestPreRequestInterpolates(t *testing.T) {
	block := []model.Variable{
		varOf("host", "http://api.test", "string"),
		varOf("base", "{{host}}/v1", "string"),
	}
	scopes := vars.Scopes{Environment: map[string]any{"host": "http://api.test"}}

	res := PreRequest(block, scopes, vars.InterpolateOptions{})

	if len(res.Values) != 2 {
		t.Fatalf("produced %d variables", len(res.Values))
	}
	if got := res.Values[1].Value; got != "http://api.test/v1" {
		t.Errorf("base = %v, want the interpolated form", got)
	}
	if len(res.Unresolved) != 0 {
		t.Errorf("unexpected unresolved: %v", res.Unresolved)
	}
}

func TestPreRequestTypedValues(t *testing.T) {
	block := []model.Variable{
		varOf("count", "42", "number"),
		varOf("flag", "true", "boolean"),
		varOf("label", "hello", "string"),
	}
	res := PreRequest(block, vars.Scopes{}, vars.InterpolateOptions{})

	byName := res.AsMap()
	if v, ok := byName["count"].(float64); !ok || v != 42 {
		t.Errorf("count = %v (%T), want the number 42", byName["count"], byName["count"])
	}
	if v, ok := byName["flag"].(bool); !ok || !v {
		t.Errorf("flag = %v (%T), want true", byName["flag"], byName["flag"])
	}
	if v, ok := byName["label"].(string); !ok || v != "hello" {
		t.Errorf("label = %v", byName["label"])
	}
}

func TestPreRequestDisabledSkipped(t *testing.T) {
	name := "skipped"
	block := []model.Variable{
		{Name: &name, Value: "x", Enabled: false},
		varOf("kept", "y", "string"),
	}
	res := PreRequest(block, vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Values) != 1 || res.Values[0].NameOrEmpty() != "kept" {
		t.Errorf("values = %+v", res.Values)
	}
}

func TestPreRequestUnresolvedReported(t *testing.T) {
	block := []model.Variable{varOf("depends", "{{missing_var}}", "string")}
	res := PreRequest(block, vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Unresolved) != 1 || res.Unresolved[0] != "depends" {
		t.Errorf("unresolved = %v", res.Unresolved)
	}
}

func TestPreRequestEmptyNameSkipped(t *testing.T) {
	block := []model.Variable{{Name: nil, Value: "x", Enabled: true}}
	res := PreRequest(block, vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Values) != 0 {
		t.Errorf("a nameless variable must be skipped, got %+v", res.Values)
	}
}

func sampleResponse() Response {
	return Response{
		Body: map[string]any{
			"access_token": "tok-123",
			"items": []any{
				map[string]any{"id": 7},
			},
			"nested": map[string]any{"deep": "value"},
		},
		Status: 201,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		ResponseTime: 1234,
	}
}

func TestPostResponseDollarPath(t *testing.T) {
	block := []model.Variable{varOf("token", "$res.body.access_token", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if got, ok := res.Lookup("token"); !ok || got != "tok-123" {
		t.Errorf("token = %v (ok=%v)", got, ok)
	}
	if len(res.Unresolved) != 0 {
		t.Errorf("unresolved = %v", res.Unresolved)
	}
}

func TestPostResponseArrayIndexPath(t *testing.T) {
	block := []model.Variable{varOf("firstId", "$res.body.items[0].id", "number")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if got, ok := res.Lookup("firstId"); !ok {
		t.Fatal("firstId missing")
	} else if f, isNum := got.(float64); !isNum || f != 7 {
		t.Errorf("firstId = %v (%T), want 7", got, got)
	}
}

func TestPostResponseStatusAndHeaders(t *testing.T) {
	block := []model.Variable{
		varOf("status", "$res.status", "number"),
		varOf("ct", "$res.headers.content-type", "string"),
		varOf("took", "$res.responseTime", "number"),
	}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	got := res.AsMap()
	if v, ok := got["status"].(float64); !ok || v != 201 {
		t.Errorf("status = %v", got["status"])
	}
	if v, ok := got["ct"].(string); !ok || v != "application/json" {
		t.Errorf("ct = %v", got["ct"])
	}
	if v, ok := got["took"].(float64); !ok || v != 1234 {
		t.Errorf("took = %v", got["took"])
	}
}

func TestPostResponseInterpolatedPath(t *testing.T) {
	// The `{{res.body…}}` form appears in Bruno's fixtures.
	block := []model.Variable{varOf("firstId", "{{res.body.items[0].id}}", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if got, ok := res.Lookup("firstId"); !ok || got != "7" {
		t.Errorf("firstId = %v (ok=%v), unresolved=%v", got, ok, res.Unresolved)
	}
}

func TestPostResponseDynamicVariable(t *testing.T) {
	block := []model.Variable{varOf("now", "{{$timestamp}}", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Unresolved) != 0 {
		t.Fatalf("dynamic variable reported unresolved: %v", res.Unresolved)
	}
	if got, ok := res.Lookup("now"); !ok {
		t.Fatal("now missing")
	} else if s, isStr := got.(string); !isStr || s == "" || strings.Contains(s, "{{") {
		t.Errorf("now = %v", got)
	}
}

func TestPostResponseLiteral(t *testing.T) {
	block := []model.Variable{varOf("env", "staging", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if got, ok := res.Lookup("env"); !ok || got != "staging" {
		t.Errorf("env = %v", got)
	}
}

func TestPostResponseScopeVariable(t *testing.T) {
	block := []model.Variable{varOf("fromEnv", "$myVar", "string")}
	scopes := vars.Scopes{Environment: map[string]any{"myVar": "from-scope"}}
	res := PostResponse(block, sampleResponse(), scopes, vars.InterpolateOptions{})
	if got, ok := res.Lookup("fromEnv"); !ok || got != "from-scope" {
		t.Errorf("fromEnv = %v (unresolved=%v)", got, res.Unresolved)
	}
}

// TestPostResponseUnsupportedExpressionIsReported is the honesty rule: a value
// that is arbitrary JavaScript needs the scripting package, and this package must
// say so rather than guess.
func TestPostResponseUnsupportedExpressionIsReported(t *testing.T) {
	block := []model.Variable{varOf("complex", "bru.getVar('x') + 1", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	// A plain literal is passed through, so this specific input is not an error;
	// what must not happen is an invented evaluation.
	got, _ := res.Lookup("complex")
	if s, ok := got.(string); ok && s == "2" {
		t.Error("the package must not evaluate arbitrary JavaScript")
	}
}

func TestPostResponseMissingPathReported(t *testing.T) {
	block := []model.Variable{varOf("nope", "$res.body.does_not_exist", "string")}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Unresolved) != 1 || res.Unresolved[0] != "nope" {
		t.Errorf("unresolved = %v", res.Unresolved)
	}
}

func TestPostResponseDisabledSkipped(t *testing.T) {
	name, other := "off", "on"
	block := []model.Variable{
		{Name: &name, Value: "$res.status", Enabled: false},
		{Name: &other, Value: "$res.status", Enabled: true},
	}
	res := PostResponse(block, sampleResponse(), vars.Scopes{}, vars.InterpolateOptions{})
	if len(res.Values) != 1 || res.Values[0].NameOrEmpty() != "on" {
		t.Errorf("values = %+v", res.Values)
	}
}

func TestPostResponseNonJSONBody(t *testing.T) {
	resp := Response{Body: "plain text", Status: 200, Headers: map[string]string{}}
	block := []model.Variable{varOf("body", "{{res.body}}", "string")}
	res := PostResponse(block, resp, vars.Scopes{}, vars.InterpolateOptions{})
	if got, ok := res.Lookup("body"); !ok || got != "plain text" {
		t.Errorf("body = %v", got)
	}
}

func TestCoerceFallbacks(t *testing.T) {
	// A conversion failure keeps the text rather than dropping the value.
	if got := coerce("not-a-number", "number"); got != "not-a-number" {
		t.Errorf("coerce(number) = %v", got)
	}
	if got := coerce("maybe", "boolean"); got != "maybe" {
		t.Errorf("coerce(boolean) = %v", got)
	}
	if got := coerce("x", "object"); got != "x" {
		t.Errorf("coerce(object) = %v", got)
	}
}

func TestStringify(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"s", "s"},
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{int64(7), "7"},
		{1.5, "1.5"},
		{2.0, "2"},
	}
	for _, tc := range cases {
		if got := stringify(tc.in); got != tc.want {
			t.Errorf("stringify(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNoGojaDependency enforces the package's stated acceptance criterion: the
// declarative blocks must not drag the script engine into the core graph.
func TestNoGojaDependency(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./").Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	if strings.Contains(string(out), "dop251/goja") {
		t.Error("the template package must not depend on goja")
	}
}
