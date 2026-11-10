package vars

import (
	"strconv"
	"strings"
)

// Lookup resolves a dotted/bracket path like "a.b[0].c" or "data['user.name']"
// from vars. It supports both direct key hits and nested traversal.
func Lookup(vars map[string]any, path string) (any, bool) {
	if vars == nil {
		return nil, false
	}
	if v, ok := vars[path]; ok {
		return v, true
	}
	tokens := tokenizePath(path)
	if len(tokens) == 0 {
		return nil, false
	}
	var cur any = vars
	for _, tok := range tokens {
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[tok]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false
			}
			cur = c[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

func tokenizePath(path string) []string {
	var out []string
	i := 0
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
		case '[':
			j := i + 1
			if j < len(path) && (path[j] == '\'' || path[j] == '"') {
				q := path[j]
				j++
				start := j
				for j < len(path) && path[j] != q {
					j++
				}
				out = append(out, path[start:j])
				if j < len(path) {
					j++ // quote
				}
				if j < len(path) && path[j] == ']' {
					j++
				}
				i = j
			} else {
				start := j
				for j < len(path) && path[j] != ']' {
					j++
				}
				out = append(out, path[start:j])
				if j < len(path) {
					j++
				}
				i = j
			}
		default:
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start < i {
				out = append(out, path[start:i])
			}
		}
	}
	return out
}

func hasKey(m map[string]any, k string) bool {
	_, ok := m[k]
	return ok
}

// GetString is a convenience helper.
func GetString(vars map[string]any, path string) (string, bool) {
	v, ok := Lookup(vars, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

var _ = strings.TrimSpace
