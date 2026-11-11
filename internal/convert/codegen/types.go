package codegen

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"fearless/internal/core/model"
)

// Target identifies a target code snippet language or library.
type Target string

const (
	TargetCurl           Target = "curl"
	TargetGo             Target = "go"
	TargetJSFetch        Target = "javascript-fetch"
	TargetJSAxios        Target = "javascript-axios"
	TargetPythonRequests Target = "python-requests"
)

// Generator is a function that produces a code snippet from a model.Request.
type Generator func(req *model.Request) (string, error)

var (
	registryMu sync.RWMutex
	registry   = map[Target]Generator{}
)

// Register associates a Generator with a Target identifier.
func Register(target Target, gen Generator) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[target] = gen
}

// ListTargets returns all registered Target identifiers in standard order.
func ListTargets() []Target {
	return []Target{
		TargetCurl,
		TargetGo,
		TargetJSFetch,
		TargetJSAxios,
		TargetPythonRequests,
	}
}

// Generate produces a code snippet for the specified Target from a model.Request.
func Generate(target Target, req *model.Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request cannot be nil")
	}

	registryMu.RLock()
	gen, ok := registry[target]
	registryMu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unsupported codegen target: %s", target)
	}

	return gen(req)
}

func synthesizeURL(rawURL string, params []model.Param) string {
	rawURL = strings.TrimSpace(rawURL)
	var enabledParams []model.Param
	for _, p := range params {
		if p.Type == model.ParamQuery && p.Live() {
			enabledParams = append(enabledParams, p)
		}
	}

	if len(enabledParams) == 0 {
		return rawURL
	}

	var q strings.Builder
	for idx, p := range enabledParams {
		if idx > 0 {
			q.WriteString("&")
		}
		name := p.NameOrEmpty()
		val := p.ValueOrEmpty()
		if strings.Contains(name, "{{") {
			q.WriteString(name)
		} else {
			q.WriteString(url.QueryEscape(name))
		}
		q.WriteString("=")
		if strings.Contains(val, "{{") {
			q.WriteString(val)
		} else {
			q.WriteString(url.QueryEscape(val))
		}
	}

	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + q.String()
}
