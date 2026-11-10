// Package template evaluates Bruno's declarative variable blocks.
//
// A `.bru` request may carry two blocks that produce variables without any
// JavaScript:
//
//	vars:pre-request {
//	  userId: 42
//	  pageSize: 20
//	}
//
//	vars:post-response {
//	  token: $res.body.access_token
//	  firstId: {{res.body.items[0].id}}
//	  now: {{$timestamp}}
//	}
//
// This package exists so the pre-request half needs no script engine at all, and
// so the common post-response forms are handled without one. Bruno evaluates both
// blocks through its JS runtimes; we handle the declarative subset natively and
// leave genuine expressions to the scripting package.
//
// Deliberately import-free of goja: see docs/progress/packages.md (`TPL`) and the
// acceptance criterion "no goja import".
package template

import (
	"fmt"
	"strings"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// Response is the response context available to post-response blocks.
//
// It mirrors the surface users reach through `$res` in Bruno
// (`$res.body`, `$res.status`, `$res.headers`, `$res.responseTime`), but as plain
// data so the package needs no engine.
type Response struct {
	// Body is the parsed response body: a decoded JSON value when the body is
	// JSON, otherwise the raw string.
	Body any
	// Status is the HTTP status code.
	Status int
	// Headers maps lower-case header names to their first value.
	Headers map[string]string
	// ResponseTime is the elapsed time in milliseconds.
	ResponseTime int64
}

// Result is the outcome of evaluating a block.
type Result struct {
	// Values are the variables produced, in source order.
	Values []model.Variable
	// Unresolved lists names or expressions that could not be evaluated. They are
	// reported rather than silently producing an empty string, which is how a
	// typo in a variable block usually manifests.
	Unresolved []string
}

// Lookup returns a produced value by name.
func (r Result) Lookup(name string) (any, bool) {
	for _, v := range r.Values {
		if v.NameOrEmpty() == name {
			return v.Value, true
		}
	}
	return nil, false
}

// AsMap returns the produced values as a plain map, for merging into scopes.
func (r Result) AsMap() map[string]any {
	out := make(map[string]any, len(r.Values))
	for _, v := range r.Values {
		out[v.NameOrEmpty()] = v.Value
	}
	return out
}

// PreRequest evaluates a `vars:pre-request` block.
//
// Values are interpolated against scopes, so `{{host}}/users` resolves, and the
// result is a string. Typed variables keep their declared type: a variable whose
// DataType is number or boolean has its value converted, which is what Bruno does
// via its `@number` / `@boolean` annotations.
func PreRequest(block []model.Variable, scopes vars.Scopes, opts vars.InterpolateOptions) Result {
	var res Result
	for _, v := range block {
		if !v.Enabled {
			continue
		}
		name := v.NameOrEmpty()
		if name == "" {
			continue
		}

		raw := stringify(v.Value)
		interpolated := vars.InterpolateWithScopes(raw, scopes, opts)

		out := v
		out.Value = coerce(interpolated, v.DataType)
		// Report a name when the source contained a placeholder that survived
		// interpolation. The name is reported, never the value: it may be a secret.
		if stillHasPlaceholder(raw) && stillHasPlaceholder(interpolated) {
			res.Unresolved = append(res.Unresolved, name)
		}
		res.Values = append(res.Values, out)
	}
	return res
}

// PostResponse evaluates a `vars:post-response` block against a response.
//
// Three value forms are supported, matching what Bruno's own fixtures contain:
//
//	token: $res.body.access_token          — a $-prefixed path expression
//	id:    {{res.body.items[0].id}}        — an interpolated path expression
//	now:   {{$timestamp}}                  — a dynamic variable
//	plain: literal text                    — used as-is
//
// Anything else is reported in Unresolved rather than guessed at: a value that
// looks like arbitrary JavaScript needs the scripting package, and inventing a
// partial evaluator here would produce quietly wrong variables.
func PostResponse(block []model.Variable, resp Response, scopes vars.Scopes, opts vars.InterpolateOptions) Result {
	var res Result

	// Response fields are exposed to interpolation under the `res.` prefix, so a
	// `{{res.body.x}}` placeholder resolves through the same path lookup as `{{x}}`.
	respScopes := scopes
	respScopes.Request = mergeMaps(respScopes.Request, responseFields(resp))

	for _, v := range block {
		if !v.Enabled {
			continue
		}
		name := v.NameOrEmpty()
		if name == "" {
			continue
		}

		raw := strings.TrimSpace(stringify(v.Value))

		out := v
		value, ok := evalPostValue(raw, resp, respScopes, opts)
		if !ok {
			res.Unresolved = append(res.Unresolved, name)
		}
		out.Value = coerce(value, v.DataType)
		res.Values = append(res.Values, out)
	}
	return res
}

// evalPostValue evaluates one post-response value.
func evalPostValue(raw string, resp Response, scopes vars.Scopes, opts vars.InterpolateOptions) (any, bool) {
	if raw == "" {
		return "", true
	}

	// A $-prefixed path: $res.body.access_token
	if strings.HasPrefix(raw, "$res.") || strings.HasPrefix(raw, "$") {
		if value, ok := resolveResPath(raw, resp); ok {
			return value, true
		}
		// A bare `$var` may name a scope variable.
		if strings.HasPrefix(raw, "$") && !strings.HasPrefix(raw, "$res") {
			if value, ok := vars.Lookup(scopes.Merge(), strings.TrimPrefix(raw, "$")); ok {
				return value, true
			}
		}
		return nil, false
	}

	// An interpolated form: {{res.body.x}} or {{$timestamp}} or a literal mix.
	if strings.Contains(raw, "{{") {
		interpolated := vars.InterpolateWithScopes(raw, scopes, opts)
		if stillHasPlaceholder(interpolated) {
			return nil, false
		}
		return interpolated, true
	}

	// Plain literal.
	return raw, true
}

// resolveResPath resolves `$res.<path>` against the response.
func resolveResPath(expr string, resp Response) (any, bool) {
	path := strings.TrimPrefix(expr, "$")
	path = strings.TrimPrefix(path, "res")

	switch path {
	case "", ".":
		return resp, true
	}
	path = strings.TrimPrefix(path, ".")

	// Fixed fields first, then the body.
	switch {
	case path == "status":
		return resp.Status, true
	case path == "responseTime":
		return resp.ResponseTime, true
	case path == "headers":
		return resp.Headers, true
	case path == "body":
		return resp.Body, true
	case strings.HasPrefix(path, "headers.") || strings.HasPrefix(path, "headers["):
		key := strings.TrimPrefix(path, "headers.")
		key = strings.TrimPrefix(key, "headers[")
		key = strings.Trim(key, "'\"]")
		v, ok := resp.Headers[strings.ToLower(key)]
		return v, ok
	case strings.HasPrefix(path, "body."):
		body, ok := resp.Body.(map[string]any)
		if !ok {
			return nil, false
		}
		return vars.Lookup(body, strings.TrimPrefix(path, "body."))
	}
	return nil, false
}

// responseFields exposes the response under `res.*` for interpolation.
func responseFields(resp Response) map[string]any {
	headers := make(map[string]any, len(resp.Headers))
	for k, v := range resp.Headers {
		headers[k] = v
	}
	return map[string]any{
		"res": map[string]any{
			"body":         resp.Body,
			"status":       resp.Status,
			"headers":      headers,
			"responseTime": resp.ResponseTime,
		},
	}
}

// mergeMaps returns a new map with b layered over a.
func mergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// stringify renders a variable value as text.
func stringify(value any) string {
	switch t := value.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		return trimFloat(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// trimFloat renders a float without a trailing ".0".
func trimFloat(f float64) string {
	s := fmt.Sprintf("%f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// coerce converts a value to the variable's declared type.
//
// A conversion failure keeps the text rather than dropping the variable: a user
// seeing the string is better served than one seeing nothing.
func coerce(value any, dataType string) any {
	s := stringify(value)
	switch dataType {
	case "number":
		var f float64
		if _, err := fmt.Sscanf(s, "%g", &f); err == nil {
			return f
		}
		return value
	case "boolean":
		switch strings.ToLower(s) {
		case "true":
			return true
		case "false":
			return false
		}
		return value
	case "object":
		// An object variable keeps its parsed form when one is available; the
		// caller (BUILD) parses JSON bodies, so passing the value through is right.
		return value
	default:
		return s
	}
}

// stillHasPlaceholder reports whether a string still contains a {{…}} marker.
func stillHasPlaceholder(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}
