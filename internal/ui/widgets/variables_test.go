package widgets

import "testing"

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []VariableSpan
	}{
		{"none", "https://api.example.com/users", nil},
		{"one", "{{baseUrl}}/v1/users", []VariableSpan{{Name: "baseUrl", Start: 0, End: 11}}},
		{
			"several",
			"{{proto}}://{{host}}:{{port}}/x",
			[]VariableSpan{
				{Name: "proto", Start: 0, End: 9},
				{Name: "host", Start: 12, End: 20},
				{Name: "port", Start: 21, End: 29},
			},
		},
		{"trimmed", "{{  baseUrl  }}/x", []VariableSpan{{Name: "baseUrl", Start: 0, End: 15}}},
		{"dotted path", "{{user.id}}", []VariableSpan{{Name: "user.id", Start: 0, End: 11}}},
		{"unterminated", "{{baseUrl/x", nil},
		{"empty", "{{}}/x", nil},
		{"only spaces", "{{   }}/x", nil},
		{"nested restart", "{{a{{b}}", []VariableSpan{{Name: "b", Start: 3, End: 8}}},
		{"adjacent", "{{a}}{{b}}", []VariableSpan{{Name: "a", Start: 0, End: 5}, {Name: "b", Start: 5, End: 10}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseVariables(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseVariables(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("span %d = %+v, want %+v", i, got[i], tc.want[i])
				}
				// The span must cover the braces it claims to.
				if got[i].Start < 0 || got[i].End > len(tc.in) || got[i].Start >= got[i].End {
					t.Errorf("span %d out of range for %q: %+v", i, tc.in, got[i])
				}
			}
		})
	}
}

func TestVariableAt(t *testing.T) {
	spans := ParseVariables("{{a}}/x/{{b}}")
	if sp, ok := VariableAt(spans, 2); !ok || sp.Name != "a" {
		t.Errorf("offset 2 should be inside a, got %+v ok=%v", sp, ok)
	}
	if sp, ok := VariableAt(spans, 6); ok {
		t.Errorf("offset 6 is plain text, got %+v", sp)
	}
	if sp, ok := VariableAt(spans, 9); !ok || sp.Name != "b" {
		t.Errorf("offset 9 should be inside b, got %+v ok=%v", sp, ok)
	}
	if _, ok := VariableAt(spans, 100); ok {
		t.Error("out-of-range offset must not match")
	}
}
