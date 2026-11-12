package widgets

import "strings"

// VariableSpan is one `{{name}}` occurrence in a template string, with the byte
// range it occupies so the caller can highlight or hit-test it.
type VariableSpan struct {
	// Name is the variable without the braces, trimmed.
	Name string
	// Start and End bound the whole `{{name}}` occurrence, braces included.
	Start, End int
}

// ParseVariables finds every `{{...}}` occurrence in s, in order.
//
// The match is deliberately forgiving, as Bruno's is: whitespace inside the
// braces is trimmed, an empty or unterminated `{{` is not a variable, and a
// nested `{{` restarts the scan rather than swallowing the rest of the line.
func ParseVariables(s string) []VariableSpan {
	var out []VariableSpan
	i := 0
	for i < len(s)-1 {
		if s[i] != '{' || s[i+1] != '{' {
			i++
			continue
		}
		start := i
		j := i + 2
		closed := false
		for j < len(s)-1 {
			if s[j] == '{' && s[j+1] == '{' {
				// A new opening before a close: abandon this one.
				break
			}
			if s[j] == '}' && s[j+1] == '}' {
				closed = true
				break
			}
			j++
		}
		if !closed {
			i = start + 2
			continue
		}
		name := strings.TrimSpace(s[start+2 : j])
		if name != "" {
			out = append(out, VariableSpan{Name: name, Start: start, End: j + 2})
		}
		i = j + 2
	}
	return out
}

// VariableAt returns the variable whose span contains byte offset off.
func VariableAt(spans []VariableSpan, off int) (VariableSpan, bool) {
	for _, sp := range spans {
		if off >= sp.Start && off < sp.End {
			return sp, true
		}
	}
	return VariableSpan{}, false
}
