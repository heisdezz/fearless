package codegen

import (
	"encoding/json"
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

func init() {
	Register(TargetPythonRequests, GeneratePython)
}

// GeneratePython produces Python requests code from a model.Request.
func GeneratePython(req *model.Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request cannot be nil")
	}

	targetURL := synthesizeURL(req.URL, req.Params)
	method := req.Method
	if method == "" {
		method = "GET"
	}

	headers := make(map[string]string)
	for _, h := range req.LiveHeaders() {
		headers[h.NameOrEmpty()] = h.ValueOrEmpty()
	}

	var authStr string
	if req.Auth != nil {
		switch req.Auth.Mode {
		case model.AuthBasic:
			var u, p string
			if req.Auth.Username != nil {
				u = *req.Auth.Username
			}
			if req.Auth.Password != nil {
				p = *req.Auth.Password
			}
			authStr = fmt.Sprintf("auth = (%q, %q)\n", u, p)
		case model.AuthBearer:
			if req.Auth.Token != nil && *req.Auth.Token != "" {
				headers["Authorization"] = "Bearer " + *req.Auth.Token
			}
		}
	}

	var payloadSetup strings.Builder
	var payloadArg = "payload"
	hasPayload := false

	if req.Body != nil {
		switch req.Body.Mode {
		case model.BodyJSON:
			if req.Body.JSON != nil && *req.Body.JSON != "" {
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				hasPayload = true
				var parsed any
				if json.Unmarshal([]byte(*req.Body.JSON), &parsed) == nil {
					compact, _ := json.Marshal(parsed)
					payloadSetup.WriteString(fmt.Sprintf("payload = %q\n", string(compact)))
				} else {
					payloadSetup.WriteString(fmt.Sprintf("payload = %q\n", *req.Body.JSON))
				}
			}
		case model.BodyFormURLEncoded:
			if _, ok := headers["Content-Type"]; !ok {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}
			hasPayload = true
			payloadSetup.WriteString("payload = {\n")
			for _, kv := range req.Body.FormURLEncoded {
				if kv.Enabled && kv.NameOrEmpty() != "" {
					payloadSetup.WriteString(fmt.Sprintf("  %q: %q,\n", kv.NameOrEmpty(), kv.ValueOrEmpty()))
				}
			}
			payloadSetup.WriteString("}\n")
		case model.BodyText, model.BodyXML, model.BodySparql:
			var val string
			if req.Body.Mode == model.BodyText && req.Body.Text != nil {
				val = *req.Body.Text
			} else if req.Body.Mode == model.BodyXML && req.Body.XML != nil {
				val = *req.Body.XML
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/xml"
				}
			} else if req.Body.Mode == model.BodySparql && req.Body.Sparql != nil {
				val = *req.Body.Sparql
			}
			if val != "" {
				hasPayload = true
				payloadSetup.WriteString(fmt.Sprintf("payload = %q\n", val))
			}
		}
	}

	var b strings.Builder
	b.WriteString("import requests\n\n")
	b.WriteString(fmt.Sprintf("url = %q\n\n", targetURL))

	if hasPayload {
		b.WriteString(payloadSetup.String())
		b.WriteString("\n")
	}

	if len(headers) > 0 {
		b.WriteString("headers = {\n")
		for k, v := range headers {
			b.WriteString(fmt.Sprintf("  %q: %q,\n", k, v))
		}
		b.WriteString("}\n\n")
	}

	if authStr != "" {
		b.WriteString(authStr)
		b.WriteString("\n")
	}

	var callArgs []string
	callArgs = append(callArgs, fmt.Sprintf("%q", strings.ToUpper(method)), "url")
	if len(headers) > 0 {
		callArgs = append(callArgs, "headers=headers")
	}
	if hasPayload {
		callArgs = append(callArgs, fmt.Sprintf("data=%s", payloadArg))
	}
	if authStr != "" {
		callArgs = append(callArgs, "auth=auth")
	}

	b.WriteString(fmt.Sprintf("response = requests.request(%s)\n\n", strings.Join(callArgs, ", ")))
	b.WriteString("print(response.text)\n")

	return b.String(), nil
}
