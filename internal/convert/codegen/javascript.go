package codegen

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

func init() {
	Register(TargetJSFetch, GenerateJSFetch)
	Register(TargetJSAxios, GenerateJSAxios)
}

// GenerateJSFetch produces JavaScript Fetch API code from a model.Request.
func GenerateJSFetch(req *model.Request) (string, error) {
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

	// Auth
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
			cred := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", u, p)))
			headers["Authorization"] = "Basic " + cred
		case model.AuthBearer:
			if req.Auth.Token != nil && *req.Auth.Token != "" {
				headers["Authorization"] = "Bearer " + *req.Auth.Token
			}
		}
	}

	// Body
	var bodyExpr string
	if req.Body != nil {
		switch req.Body.Mode {
		case model.BodyJSON:
			if req.Body.JSON != nil && *req.Body.JSON != "" {
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				var parsed any
				if json.Unmarshal([]byte(*req.Body.JSON), &parsed) == nil {
					pretty, _ := json.MarshalIndent(parsed, "  ", "  ")
					bodyExpr = fmt.Sprintf("JSON.stringify(%s)", string(pretty))
				} else {
					bodyExpr = fmt.Sprintf("%q", *req.Body.JSON)
				}
			}
		case model.BodyFormURLEncoded:
			if _, ok := headers["Content-Type"]; !ok {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}
			var entries []string
			for _, kv := range req.Body.FormURLEncoded {
				if kv.Enabled && kv.NameOrEmpty() != "" {
					entries = append(entries, fmt.Sprintf("params.append(%q, %q);", kv.NameOrEmpty(), kv.ValueOrEmpty()))
				}
			}
			if len(entries) > 0 {
				bodyExpr = "params"
			}
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
				bodyExpr = fmt.Sprintf("%q", val)
			}
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("const url = %q;\n", targetURL))

	if req.Body != nil && req.Body.Mode == model.BodyFormURLEncoded {
		b.WriteString("const params = new URLSearchParams();\n")
		for _, kv := range req.Body.FormURLEncoded {
			if kv.Enabled && kv.NameOrEmpty() != "" {
				b.WriteString(fmt.Sprintf("params.append(%q, %q);\n", kv.NameOrEmpty(), kv.ValueOrEmpty()))
			}
		}
	}

	b.WriteString("const options = {\n")
	b.WriteString(fmt.Sprintf("  method: %q,\n", strings.ToUpper(method)))

	if len(headers) > 0 {
		b.WriteString("  headers: {\n")
		for k, v := range headers {
			b.WriteString(fmt.Sprintf("    %q: %q,\n", k, v))
		}
		b.WriteString("  },\n")
	}

	if bodyExpr != "" {
		b.WriteString(fmt.Sprintf("  body: %s,\n", bodyExpr))
	}

	b.WriteString("};\n\n")
	b.WriteString("fetch(url, options)\n")
	b.WriteString("  .then(res => res.text())\n")
	b.WriteString("  .then(data => console.log(data))\n")
	b.WriteString("  .catch(err => console.error(err));\n")

	return b.String(), nil
}

// GenerateJSAxios produces JavaScript Axios code from a model.Request.
func GenerateJSAxios(req *model.Request) (string, error) {
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

	var authBasicStr string
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
			authBasicStr = fmt.Sprintf("  auth: {\n    username: %q,\n    password: %q,\n  },\n", u, p)
		case model.AuthBearer:
			if req.Auth.Token != nil && *req.Auth.Token != "" {
				headers["Authorization"] = "Bearer " + *req.Auth.Token
			}
		}
	}

	var dataExpr string
	if req.Body != nil {
		switch req.Body.Mode {
		case model.BodyJSON:
			if req.Body.JSON != nil && *req.Body.JSON != "" {
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				var parsed any
				if json.Unmarshal([]byte(*req.Body.JSON), &parsed) == nil {
					pretty, _ := json.MarshalIndent(parsed, "  ", "  ")
					dataExpr = string(pretty)
				} else {
					dataExpr = fmt.Sprintf("%q", *req.Body.JSON)
				}
			}
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
				dataExpr = fmt.Sprintf("%q", val)
			}
		}
	}

	var b strings.Builder
	b.WriteString("const axios = require('axios');\n\n")
	b.WriteString("const config = {\n")
	b.WriteString(fmt.Sprintf("  method: %q,\n", strings.ToLower(method)))
	b.WriteString(fmt.Sprintf("  url: %q,\n", targetURL))

	if len(headers) > 0 {
		b.WriteString("  headers: {\n")
		for k, v := range headers {
			b.WriteString(fmt.Sprintf("    %q: %q,\n", k, v))
		}
		b.WriteString("  },\n")
	}

	if authBasicStr != "" {
		b.WriteString(authBasicStr)
	}

	if dataExpr != "" {
		b.WriteString(fmt.Sprintf("  data: %s,\n", dataExpr))
	}

	b.WriteString("};\n\n")
	b.WriteString("axios(config)\n")
	b.WriteString("  .then(response => console.log(JSON.stringify(response.data)))\n")
	b.WriteString("  .catch(error => console.error(error));\n")

	return b.String(), nil
}
