package vars

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var (
	placeholderRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)
	mockRe        = regexp.MustCompile(`\{\{\$(\w+)\}\}`)
	jsonSpecial   = regexp.MustCompile(`[\\\n\r\t"]`)
)

// InterpolateOptions configures interpolation behavior.
type InterpolateOptions struct {
	EscapeJSON   bool
	OnUnresolved func(name string)
}

// Interpolate replaces {{path}} placeholders in s using vars.
// It applies {{$faker}} mock expansion first, resolves variables recursively
// with cycle protection and memoization, and reports unresolved names.
func Interpolate(s string, vars map[string]any, opts InterpolateOptions) string {
	if s == "" {
		return s
	}
	prepared := prepareMock(s, opts.EscapeJSON)
	if vars == nil {
		return prepared
	}
	preparedVars := prepareMockObj(vars, false)
	return replace(prepared, preparedVars, opts, map[string]bool{}, map[string]string{}, 0)
}

const maxDepth = 256

func prepareMock(s string, escape bool) string {
	return mockRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := mockRe.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		v, ok := MockValue(sub[1])
		if !ok {
			return match
		}
		if escape {
			return escapeJSONString(v)
		}
		return v
	})
}

func prepareMockObj(obj map[string]any, escape bool) map[string]any {
	if obj == nil {
		return nil
	}
	out := make(map[string]any, len(obj))
	for k, v := range obj {
		switch vv := v.(type) {
		case string:
			out[k] = prepareMock(vv, escape)
		case map[string]any:
			out[k] = prepareMockObj(vv, escape)
		default:
			out[k] = v
		}
	}
	return out
}

func escapeJSONString(s string) string {
	if !jsonSpecial.MatchString(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func replace(s string, vars map[string]any, opts InterpolateOptions, visited map[string]bool, results map[string]string, depth int) string {
	if depth > maxDepth {
		return s
	}
	for {
		matchFound := false
		out := placeholderRe.ReplaceAllStringFunc(s, func(match string) string {
			if cached, ok := results[match]; ok {
				return cached
			}
			sub := placeholderRe.FindStringSubmatch(match)
			if len(sub) != 2 {
				return match
			}
			key := sub[1]
			val, ok := Lookup(vars, key)
			var rep string
			if !ok {
				if opts.OnUnresolved != nil && !visited[match] {
					opts.OnUnresolved(key)
				}
				visited[match] = true
				results[match] = match
				return match
			}
			rep = stringify(val)
			if opts.EscapeJSON {
				if _, okStr := val.(string); okStr {
					rep = escapeJSONString(rep)
				}
			}

			if placeholderRe.MatchString(rep) && !visited[match] {
				visited[match] = true
				res := replace(rep, vars, opts, visited, results, depth+1)
				results[match] = res
				matchFound = true
				return res
			}
			visited[match] = true
			results[match] = rep
			matchFound = true
			return rep
		})
		if out == s {
			return out
		}
		s = out
		if !matchFound {
			return s
		}
	}
}

func stringify(v any) string {
	switch vv := v.(type) {
	case nil:
		return "null"
	case string:
		return vv
	case bool:
		if vv {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case int:
		return strconv.Itoa(vv)
	case int64:
		return strconv.FormatInt(vv, 10)
	case json.Number:
		return vv.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
