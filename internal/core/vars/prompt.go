package vars

import "regexp"

var promptRe = regexp.MustCompile(`\{\{\?([^{}\s](?:[^{}]*[^{}\s])?)\}\}`)

// ExtractPromptVariables returns prompt variable labels in order of first appearance.
func ExtractPromptVariablesFromString(s string) []string {
	matches := promptRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// ExtractPromptVariables walks arbitrary values (string/map/slice) and collects prompt labels.
func ExtractPromptVariables(v any) []string {
	seen := map[string]bool{}
	var out []string
	var walk func(any)
	walk = func(x any) {
		switch vv := x.(type) {
		case string:
			for _, p := range ExtractPromptVariablesFromString(vv) {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		case []any:
			for _, it := range vv {
				walk(it)
			}
		case map[string]any:
			for _, it := range vv {
				walk(it)
			}
		}
	}
	walk(v)
	return out
}
