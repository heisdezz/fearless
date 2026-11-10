package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"fearless/internal/core/model"
	"fearless/internal/core/template"
)

// TemplateResponseFromResult builds a template.Response from a model.Result.
// This allows template.PostResponse to evaluate $res.body, $res.status, $res.headers, etc.
func TemplateResponseFromResult(res *model.Result) template.Response {
	if res == nil {
		return template.Response{}
	}

	headers := make(map[string]string)
	for _, h := range res.Headers {
		k := strings.ToLower(h.NameOrEmpty())
		if k != "" {
			if _, exists := headers[k]; !exists {
				headers[k] = h.ValueOrEmpty()
			}
		}
	}

	var rawBytes []byte
	if len(res.Body.Inline) > 0 {
		rawBytes = res.Body.Inline
	} else if res.Body.Path != "" {
		if data, err := os.ReadFile(res.Body.Path); err == nil {
			rawBytes = data
		}
	}

	var body any = string(rawBytes)
	trimmed := bytes.TrimSpace(rawBytes)
	ct := strings.ToLower(res.Body.ContentType)
	if (strings.Contains(ct, "json") || bytes.HasPrefix(trimmed, []byte("{")) || bytes.HasPrefix(trimmed, []byte("["))) && len(rawBytes) > 0 {
		var decoded any
		if err := json.Unmarshal(rawBytes, &decoded); err == nil {
			body = decoded
		}
	}

	return template.Response{
		Status:       res.Status,
		Headers:      headers,
		Body:         body,
		ResponseTime: res.Duration.Milliseconds(),
	}
}
