package scripting

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/dop251/goja"

	"fearless/internal/core/model"
)

// buildReq constructs the `req` object.
//
// In Bruno `req` is a live view over the request being executed: setHeader in a
// pre-request script changes what is sent. The same holds here, mutating
// State.Request in place.
//
// Source: bruno-js/src/bruno-request.js
func (b *Bindings) buildReq() (goja.Value, error) {
	return b.object(map[string]any{
		"getUrl": func() string { return b.requestURL() },
		"setUrl": func(call goja.FunctionCall) goja.Value {
			if req := b.state.Request; req != nil {
				req.URL = call.Argument(0).String()
			}
			b.state.RequestURL = call.Argument(0).String()
			return goja.Undefined()
		},
		"getHost": func() string {
			if u, err := url.Parse(b.requestURL()); err == nil {
				return u.Host
			}
			return ""
		},
		"getPath": func() string {
			if u, err := url.Parse(b.requestURL()); err == nil {
				return u.Path
			}
			return ""
		},
		"getQueryString": func() string {
			if u, err := url.Parse(b.requestURL()); err == nil {
				return u.RawQuery
			}
			return ""
		},
		"getMethod": func() string {
			if req := b.state.Request; req != nil {
				return req.Method
			}
			return ""
		},
		"setMethod": func(call goja.FunctionCall) goja.Value {
			if req := b.state.Request; req != nil {
				req.Method = strings.ToUpper(call.Argument(0).String())
			}
			return goja.Undefined()
		},
		"getAuthMode": func() string {
			if req := b.state.Request; req != nil && req.Auth != nil {
				return string(req.Auth.Mode)
			}
			return "none"
		},
		"getHeaders": func() goja.Value { return b.vm.ToValue(b.headerMap()) },
		"getHeader": func(call goja.FunctionCall) goja.Value {
			return b.vm.ToValue(lookupHeader(b.headerMap(), call.Argument(0).String()))
		},
		"setHeader": func(call goja.FunctionCall) goja.Value {
			b.mutateHeader(call.Argument(0).String(), func(kv *model.KeyValue) {
				value := call.Argument(1).String()
				kv.Value = &value
				kv.Enabled = true
			})
			return goja.Undefined()
		},
		"deleteHeader": func(call goja.FunctionCall) goja.Value {
			b.removeHeaders([]string{call.Argument(0).String()})
			return goja.Undefined()
		},
		"setHeaders": func(call goja.FunctionCall) goja.Value {
			// Replaces the whole set, as Bruno does, rather than merging.
			if req := b.state.Request; req != nil {
				req.Headers = nil
			}
			for name, value := range b.exportMap(call.Argument(0)) {
				text := asString(value)
				b.mutateHeader(name, func(kv *model.KeyValue) {
					kv.Value = &text
					kv.Enabled = true
				})
			}
			return goja.Undefined()
		},
		"deleteHeaders": func(call goja.FunctionCall) goja.Value {
			var names []string
			if values, ok := call.Argument(0).Export().([]any); ok {
				for _, v := range values {
					names = append(names, asString(v))
				}
			}
			b.removeHeaders(names)
			return goja.Undefined()
		},
		"hasJSONContentType": func() bool {
			contentType, _ := lookupHeader(b.headerMap(), "content-type").(string)
			return strings.Contains(strings.ToLower(contentType), "application/json")
		},
		"getBody": func(call goja.FunctionCall) goja.Value {
			return b.vm.ToValue(b.requestBody(call))
		},
		"setBody": func(call goja.FunctionCall) goja.Value {
			b.setRequestBody(call)
			return goja.Undefined()
		},
		"setMaxRedirects": func(call goja.FunctionCall) goja.Value {
			// Carried on the request settings by the caller; recorded here so a
			// script can express it without reaching into the engine.
			b.state.maxRedirects = int(call.Argument(0).ToInteger())
			return goja.Undefined()
		},
		"getTimeout": func() int { return b.state.timeoutMs },
		"setTimeout": func(call goja.FunctionCall) goja.Value {
			b.state.timeoutMs = int(call.Argument(0).ToInteger())
			return goja.Undefined()
		},
		"disableParsingResponseJson": func() goja.Value {
			b.state.disableJSONParsing = true
			return goja.Undefined()
		},
		"getExecutionMode": func() string { return b.buildBruMode() },
		"getName": func() string {
			if b.state.Request != nil {
				return b.state.Request.Name
			}
			return ""
		},
		"getPathParams": func() goja.Value { return b.vm.ToValue([]any{}) },
		"getTags": func() goja.Value {
			if b.state.Request != nil {
				return b.vm.ToValue(b.state.Request.Tags)
			}
			return b.vm.ToValue([]string{})
		},
		"onFail": func(call goja.FunctionCall) goja.Value {
			// The handler is stored and invoked by the host when the request fails,
			// so a script can rewrite the failure into a clearer message.
			b.state.failHandler = call.Argument(0)
			return goja.Undefined()
		},
	})
}

// buildBruMode returns the execution mode for req.getExecutionMode.
func (b *Bindings) buildBruMode() string {
	if b.state.ExecutionMode == "" {
		return "standalone"
	}
	return b.state.ExecutionMode
}

// requestURL returns the effective URL, preferring a script-set override.
func (b *Bindings) requestURL() string {
	if b.state.RequestURL != "" {
		return b.state.RequestURL
	}
	if req := b.state.Request; req != nil {
		return req.URL
	}
	return ""
}

// headerMap returns the enabled headers as a name→value map.
func (b *Bindings) headerMap() map[string]any {
	out := map[string]any{}
	if req := b.state.Request; req != nil {
		for _, kv := range req.Headers {
			if !kv.Enabled {
				continue
			}
			out[kv.NameOrEmpty()] = kv.ValueOrEmpty()
		}
	}
	return out
}

// lookupHeader finds a header case-insensitively.
func lookupHeader(headers map[string]any, name string) any {
	if v, ok := headers[name]; ok {
		return v
	}
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return nil
}

// mutateHeader creates or updates a header row.
func (b *Bindings) mutateHeader(name string, mutate func(*model.KeyValue)) {
	if name == "" {
		b.throw("scripting: a header name is required")
	}
	req := b.ensureRequest()
	for i := range req.Headers {
		if strings.EqualFold(req.Headers[i].NameOrEmpty(), name) {
			mutate(&req.Headers[i])
			return
		}
	}
	trimmed := name
	kv := model.KeyValue{Name: &trimmed, Enabled: true}
	mutate(&kv)
	req.Headers = append(req.Headers, kv)
}

// removeHeaders deletes headers by name.
func (b *Bindings) removeHeaders(names []string) {
	req := b.ensureRequest()
	kept := req.Headers[:0]
	for _, kv := range req.Headers {
		remove := false
		for _, name := range names {
			if strings.EqualFold(kv.NameOrEmpty(), name) {
				remove = true
				break
			}
		}
		if !remove {
			kept = append(kept, kv)
		}
	}
	req.Headers = kept
}

// ensureRequest creates a request when the script runs outside one.
func (b *Bindings) ensureRequest() *model.Request {
	if b.state.Request == nil {
		b.state.Request = &model.Request{}
	}
	return b.state.Request
}

// toAnySlice widens a value into a slice for appending.
func toAnySlice(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{v}
}

// requestBody returns the body, parsed as JSON unless raw is requested.
//
// Source: bruno-js/src/bruno-request.js getBody
func (b *Bindings) requestBody(call goja.FunctionCall) any {
	raw := false
	if opts := b.exportMap(call.Argument(0)); opts != nil {
		if v, ok := opts["raw"]; ok {
			raw = strings.EqualFold(asString(v), "true")
		}
	}

	req := b.state.Request
	if req == nil || req.Body == nil {
		return nil
	}

	text := model.DerefString(req.Body.JSON)
	if text == "" {
		text = model.DerefString(req.Body.Text)
	}
	if raw {
		return text
	}
	if text == "" {
		return nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		// Not JSON: return the text rather than nothing, so a script that set a
		// plain body can still read it back.
		return text
	}
	return parsed
}

// setRequestBody replaces the body.
//
// Source: bruno-js/src/bruno-request.js setBody
func (b *Bindings) setRequestBody(call goja.FunctionCall) {
	value := call.Argument(0)
	req := b.ensureRequest()

	if req.Body == nil {
		req.Body = &model.Body{}
	}

	raw := false
	if opts := b.exportMap(call.Argument(1)); opts != nil {
		if v, ok := opts["raw"]; ok {
			raw = strings.EqualFold(asString(v), "true")
		}
	}

	var text string
	switch export := exportJS(value).(type) {
	case string:
		text = export
	default:
		if encoded, err := json.Marshal(export); err == nil {
			text = string(encoded)
		}
	}

	if raw {
		req.Body.Mode = model.BodyText
		req.Body.Text = &text
		return
	}
	req.Body.Mode = model.BodyJSON
	req.Body.JSON = &text
}

// buildRes constructs the `res` object.
//
// Source: bruno-js/src/bruno-response.js
func (b *Bindings) buildRes() (goja.Value, error) {
	return b.object(map[string]any{
		"getStatus": func() int {
			if b.state.Result != nil {
				return b.state.Result.Status
			}
			return 0
		},
		"getStatusText": func() string {
			if b.state.Result != nil {
				return b.state.Result.StatusText
			}
			return ""
		},
		"getHeader": func(call goja.FunctionCall) goja.Value {
			return b.vm.ToValue(b.responseHeader(call.Argument(0).String()))
		},
		"getHeaders": func() goja.Value {
			out := map[string]any{}
			if b.state.Result != nil {
				for _, kv := range b.state.Result.Headers {
					name := kv.NameOrEmpty()
					if existing, ok := out[name]; ok {
						out[name] = append(toAnySlice(existing), kv.ValueOrEmpty())
						continue
					}
					out[name] = kv.ValueOrEmpty()
				}
			}
			return b.vm.ToValue(out)
		},
		"getBody": func(call goja.FunctionCall) goja.Value {
			return b.vm.ToValue(b.responseBody(call))
		},
		"getResponseTime": func() int64 {
			if b.state.Result != nil {
				return b.state.Result.Duration.Milliseconds()
			}
			return 0
		},
		"getUrl": func() string { return b.requestURL() },
		"getSize": func() int64 {
			if b.state.Result != nil {
				return b.state.Result.Body.Size
			}
			return 0
		},
		"getDataBuffer": func() goja.Value {
			// The body bytes as an array of numbers, which is what a Node Buffer
			// exports as through this accessor.
			data := b.state.BodyBytes
			values := make([]any, len(data))
			for i, c := range data {
				values[i] = int(c)
			}
			return b.vm.ToValue(values)
		},
		"setBody": func(call goja.FunctionCall) goja.Value {
			if b.state.Result != nil {
				b.state.ResponseText = call.Argument(0).String()
			}
			return goja.Undefined()
		},
	})
}

// responseHeader reads a response header case-insensitively.
func (b *Bindings) responseHeader(name string) any {
	if b.state.Result == nil {
		return nil
	}
	if values := b.state.Result.Headers.Values(name); len(values) > 0 {
		if len(values) == 1 {
			return values[0]
		}
		return values
	}
	return nil
}

// responseBody returns the body, parsed as JSON unless raw is requested.
//
// Source: bruno-js/src/bruno-response.js getBody
func (b *Bindings) responseBody(call goja.FunctionCall) any {
	raw := false
	if opts := b.exportMap(call.Argument(0)); opts != nil {
		if v, ok := opts["raw"]; ok {
			raw = strings.EqualFold(asString(v), "true")
		}
	}

	text := b.state.ResponseText
	if raw {
		return text
	}
	if text == "" {
		return nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return text
	}
	return parsed
}
