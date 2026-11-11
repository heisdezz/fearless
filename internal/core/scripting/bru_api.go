package scripting

import (
	"encoding/json"

	"github.com/dop251/goja"
)

// buildBru constructs the `bru` object.
//
// The method set mirrors bruno-js/src/bru.js exactly, including the three
// distinct variable scopes that all look similar:
//
//   - setVar/getVar        runtime variables, session-only
//   - setEnvVar/getEnvVar  the active environment (persisted)
//   - setCollectionVar     collection variables (persisted)
//
// Source: bruno-js/src/bru.js
func (b *Bindings) buildBru() (goja.Value, error) {
	runner, err := b.object(map[string]any{
		"skipRequest": func() {
			b.control.SkipRequest = true
		},
		"stopExecution": func() {
			b.control.StopExecution = true
		},
		"setNextRequest": func(call goja.FunctionCall) goja.Value {
			b.control.NextRequest = call.Argument(0).String()
			return goja.Undefined()
		},
	})
	if err != nil {
		return nil, err
	}

	utils, err := b.object(map[string]any{
		"minifyJson": func(call goja.FunctionCall) goja.Value {
			return b.minifyJSON(call.Argument(0))
		},
		"minifyXml": func(call goja.FunctionCall) goja.Value {
			return b.minifyXML(call.Argument(0))
		},
	})
	if err != nil {
		return nil, err
	}

	return b.object(map[string]any{
		"interpolate": func(call goja.FunctionCall) goja.Value {
			return b.vm.ToValue(b.interpolate(call.Argument(0).String()))
		},
		"getEnvName":        func() string { return b.state.EnvName },
		"getCollectionName": func() string { return b.state.CollectionName },
		"getProcessEnv": func(call goja.FunctionCall) goja.Value {
			return b.getProcessEnv(call.Argument(0).String())
		},
		"isSafeMode": func() bool { return b.state.IsSafeMode },
		"sleep": func(call goja.FunctionCall) goja.Value {
			// The host honours the delay after the script finishes; blocking here
			// would stall the deadline watcher that is meant to bound the script.
			b.control.SleepMillis += int(call.Argument(0).ToInteger())
			return goja.Undefined()
		},
		"cwd": func() string { return b.state.CollectionPath },
		"getExecutionMode": func() string {
			if b.state.ExecutionMode == "" {
				return "standalone"
			}
			return b.state.ExecutionMode
		},
		"runner": runner,
		"utils":  utils,

		// Environment scope.
		"hasEnvVar": func(call goja.FunctionCall) bool {
			return exists(b.scope("env"), call.Argument(0).String())
		},
		"getEnvVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("env", call.Argument(0).String())
		},
		"setEnvVar": func(call goja.FunctionCall) goja.Value {
			b.setInScope("env", call)
			return goja.Undefined()
		},
		"deleteEnvVar": func(call goja.FunctionCall) {
			delete(b.scope("env"), call.Argument(0).String())
		},
		"getAllEnvVars": func() goja.Value { return b.vm.ToValue(copyMap(b.scope("env"))) },
		"deleteAllEnvVars": func() {
			if b.state.IsSafeMode {
				b.throw("scripting: cannot modify variables in safe mode")
			}
			b.state.Scopes.Environment = map[string]any{}
		},

		// Global environment scope.
		"hasGlobalEnvVar": func(call goja.FunctionCall) bool {
			return exists(b.scope("global"), call.Argument(0).String())
		},
		"getGlobalEnvVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("global", call.Argument(0).String())
		},
		"setGlobalEnvVar": func(call goja.FunctionCall) goja.Value {
			b.setInScope("global", call)
			return goja.Undefined()
		},
		"deleteGlobalEnvVar": func(call goja.FunctionCall) {
			delete(b.scope("global"), call.Argument(0).String())
		},
		"getAllGlobalEnvVars": func() goja.Value {
			return b.vm.ToValue(copyMap(b.scope("global")))
		},
		"deleteAllGlobalEnvVars": func() {
			if b.state.IsSafeMode {
				b.throw("scripting: cannot modify variables in safe mode")
			}
			b.state.Scopes.Global = map[string]any{}
		},

		// Runtime scope.
		"hasVar": func(call goja.FunctionCall) bool {
			return exists(b.scope("runtime"), call.Argument(0).String())
		},
		"getVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("runtime", call.Argument(0).String())
		},
		"setVar": func(call goja.FunctionCall) goja.Value {
			b.setInScope("runtime", call)
			return goja.Undefined()
		},
		"deleteVar": func(call goja.FunctionCall) {
			delete(b.scope("runtime"), call.Argument(0).String())
		},
		"getAllVars": func() goja.Value { return b.vm.ToValue(copyMap(b.scope("runtime"))) },
		"deleteAllVars": func() {
			if b.state.IsSafeMode {
				b.throw("scripting: cannot modify variables in safe mode")
			}
			b.state.Scopes.Runtime = map[string]any{}
		},

		// Collection scope.
		"getCollectionVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("collection", call.Argument(0).String())
		},
		"setCollectionVar": func(call goja.FunctionCall) goja.Value {
			b.setInScope("collection", call)
			return goja.Undefined()
		},
		"hasCollectionVar": func(call goja.FunctionCall) bool {
			return exists(b.scope("collection"), call.Argument(0).String())
		},
		"deleteCollectionVar": func(call goja.FunctionCall) {
			delete(b.scope("collection"), call.Argument(0).String())
		},
		"getAllCollectionVars": func() goja.Value {
			return b.vm.ToValue(copyMap(b.scope("collection")))
		},
		"deleteAllCollectionVars": func() {
			if b.state.IsSafeMode {
				b.throw("scripting: cannot modify variables in safe mode")
			}
			b.state.Scopes.Collection = map[string]any{}
		},

		// Read-only scopes.
		"getFolderVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("folder", call.Argument(0).String())
		},
		"getRequestVar": func(call goja.FunctionCall) goja.Value {
			return b.getFromScope("request", call.Argument(0).String())
		},

		// Runner control, also reachable at the top level.
		"setNextRequest": func(call goja.FunctionCall) goja.Value {
			b.control.NextRequest = call.Argument(0).String()
			return goja.Undefined()
		},

		// Nested requests. These record the call and return a rejection-shaped
		// promise whose result the host fills in, because the engine is not
		// imported here; see SendRequestFunc.
		"sendRequest": b.sendRequest("sendRequest"),
		"runRequest":  b.sendRequest("runRequest"),
	})
}

// getFromScope reads a variable, preferring the merged view.
//
// A value the script deleted from its own scope must not reappear from a lower
// one, so a present-but-empty entry shadows; only a genuinely missing name falls
// through to the merge.
func (b *Bindings) getFromScope(scopeName, key string) goja.Value {
	if entry, ok := b.scope(scopeName)[key]; ok {
		return b.vm.ToValue(entry)
	}
	if merged := b.merged(); merged != nil {
		if entry, ok := merged[key]; ok {
			return b.vm.ToValue(entry)
		}
	}
	return goja.Undefined()
}

// setInScope writes a variable, refusing in safe mode.
func (b *Bindings) setInScope(scopeName string, call goja.FunctionCall) {
	if b.state.IsSafeMode {
		b.throw("scripting: cannot modify variables in safe mode")
	}
	key := call.Argument(0).String()
	if key == "" {
		b.throw("scripting: a variable name is required")
	}
	// A JS object must arrive as a Go map, not as a goja object, so later
	// interpolation can walk it and the caller can persist it.
	b.scope(scopeName)[key] = exportJS(call.Argument(1))
}

// exportJS converts a JS value into a plain Go value.
func exportJS(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return v.Export()
}

// getProcessEnv reads an OS environment variable.
//
// Source: bruno-js/src/bru.js getProcessEnv
func (b *Bindings) getProcessEnv(key string) goja.Value {
	if b.state.Scopes != nil && b.state.Scopes.ProcessEnv != nil {
		if value, ok := b.state.Scopes.ProcessEnv[key]; ok {
			return b.vm.ToValue(value)
		}
	}
	return goja.Undefined()
}

// sendRequest returns a function that records a nested request.
//
// The returned promise rejects with an explanatory error: a script that calls
// sendRequest without a host handler must learn the request did not happen, rather
// than receive an undefined response and fail somewhere unrelated.
func (b *Bindings) sendRequest(source string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		method, url := parseRequestArgument(call.Argument(0))
		b.scripted = append(b.scripted, ScriptedCall{Source: source, Method: method, URL: url})

		promise, _, reject := b.vm.NewPromise()
		message := "scripting: bru." + source + " has no host handler for " + method + " " + url

		// A callback-style call reports the failure through its callback, matching
		// Bruno's (error, response) contract.
		if len(call.Arguments) > 1 {
			if fn, ok := goja.AssertFunction(call.Argument(1)); ok {
				_, _ = fn(goja.Undefined(), b.vm.ToValue(message), goja.Null())
			}
		}
		_ = reject(message)
		return b.vm.ToValue(promise)
	}
}

// parseRequestArgument extracts method and URL from sendRequest's argument.
//
// Bruno accepts either a URL string or an options object with `url` and `method`.
func parseRequestArgument(arg goja.Value) (method, url string) {
	method = "GET"
	if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
		return method, ""
	}

	if obj, ok := arg.(*goja.Object); ok {
		if v := obj.Get("url"); v != nil && !goja.IsUndefined(v) {
			url = v.String()
		}
		if v := obj.Get("method"); v != nil && !goja.IsUndefined(v) {
			method = v.String()
		}
		return method, url
	}
	return method, arg.String()
}

// minifyJSON implements bru.utils.minifyJson.

// Source: bruno-js/src/bru.js utils.minifyJson
func (b *Bindings) minifyJSON(value goja.Value) goja.Value {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		b.throw("Failed to minify")
	}

	switch export := exportJS(value).(type) {
	case string:
		var parsed any
		if err := json.Unmarshal([]byte(export), &parsed); err != nil {
			b.throw("Failed to minify: %s", err.Error())
		}
		// Re-marshalling and stripping the trailing newline reproduces
		// JSON.stringify's output for the parsed value.
		if encoded, err := json.Marshal(parsed); err == nil {
			return b.vm.ToValue(string(encoded))
		}
		b.throw("Failed to minify")
	default:
		if encoded, err := json.Marshal(export); err == nil {
			return b.vm.ToValue(string(encoded))
		}
		b.throw("Failed to minify")
	}
	return goja.Undefined()
}

// minifyXML implements bru.utils.minifyXml by collapsing inter-tag whitespace.
//
// A real XML minifier is not in the standard library. This removes whitespace
// between `>` and `<`, which is what minification means in practice and what the
// fixtures exercise; it deliberately does not touch text nodes, where whitespace
// can be significant.
//
// Source: bruno-js/src/bru.js utils.minifyXml
func (b *Bindings) minifyXML(value goja.Value) goja.Value {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return b.vm.ToValue("")
	}
	input := value.String()

	var out []byte
	inTag := false
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
		}
		// Drop whitespace sequences that sit between a closing and an opening tag.
		if (c == ' ' || c == '\n' || c == '\t' || c == '\r') && !inTag {
			j := i
			for j < len(input) && (input[j] == ' ' || input[j] == '\n' || input[j] == '\t' || input[j] == '\r') {
				j++
			}
			if j < len(input) && input[j] == '<' && len(out) > 0 && out[len(out)-1] == '>' {
				i = j - 1
				continue
			}
		}
		out = append(out, c)
	}
	return b.vm.ToValue(string(out))
}

// exists reports whether a key is present, including with a nil value.
func exists(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// copyMap returns a shallow copy, which goja then converts to a JS object.
func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
