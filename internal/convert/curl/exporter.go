package curl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"fearless/internal/core/model"
)

// ExportOptions configures how cURL commands are formatted.
type ExportOptions struct {
	// Multiline separates flags onto new lines with backslash continuations.
	Multiline bool
}

// Export converts a model.Request into a runnable cURL command string.
func Export(req *model.Request) (string, error) {
	return ExportWithOptions(req, ExportOptions{Multiline: true})
}

// ExportWithOptions converts a model.Request into a cURL command string with custom options.
func ExportWithOptions(req *model.Request, opts ExportOptions) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request cannot be nil")
	}

	var parts []string
	parts = append(parts, "curl")

	// Method
	method := req.Method
	if method == "" {
		method = "GET"
	}
	if method != "GET" || (req.Body != nil && !req.Body.IsEmpty()) {
		parts = append(parts, "-X", method)
	}

	// Full URL with query parameters synthesized
	targetURL := synthesizeURL(req.URL, req.Params)
	parts = append(parts, escapeShellArg(targetURL))

	// Settings
	if req.Settings.FollowRedirects != nil && *req.Settings.FollowRedirects {
		parts = append(parts, "-L")
	}

	// Headers
	hasContentType := false
	hasAuthHeader := false
	for _, h := range req.LiveHeaders() {
		name := h.NameOrEmpty()
		val := h.ValueOrEmpty()
		if strings.EqualFold(name, "Content-Type") {
			hasContentType = true
		}
		if strings.EqualFold(name, "Authorization") {
			hasAuthHeader = true
		}
		parts = append(parts, "-H", escapeShellArg(fmt.Sprintf("%s: %s", name, val)))
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
			if u != "" || p != "" {
				parts = append(parts, "-u", escapeShellArg(fmt.Sprintf("%s:%s", u, p)))
			}
		case model.AuthBearer:
			if !hasAuthHeader && req.Auth.Token != nil && *req.Auth.Token != "" {
				parts = append(parts, "-H", escapeShellArg(fmt.Sprintf("Authorization: Bearer %s", *req.Auth.Token)))
			}
		case model.AuthDigest:
			var u, p string
			if req.Auth.Username != nil {
				u = *req.Auth.Username
			}
			if req.Auth.Password != nil {
				p = *req.Auth.Password
			}
			parts = append(parts, "--digest", "-u", escapeShellArg(fmt.Sprintf("%s:%s", u, p)))
		case model.AuthNTLM:
			var u, p string
			if req.Auth.Username != nil {
				u = *req.Auth.Username
			}
			if req.Auth.Password != nil {
				p = *req.Auth.Password
			}
			parts = append(parts, "--ntlm", "-u", escapeShellArg(fmt.Sprintf("%s:%s", u, p)))
		case model.AuthAWSV4:
			parts = append(parts, "--aws-sigv4", escapeShellArg("aws:amz"))
		}
	}

	// Body
	if req.Body != nil {
		switch req.Body.Mode {
		case model.BodyJSON:
			if req.Body.JSON != nil && *req.Body.JSON != "" {
				if !hasContentType {
					parts = append(parts, "-H", escapeShellArg("Content-Type: application/json"))
				}
				parts = append(parts, "--data-raw", escapeShellArg(*req.Body.JSON))
			}
		case model.BodyGraphQL:
			if req.Body.GraphQL != nil && req.Body.GraphQL.Query != nil {
				if !hasContentType {
					parts = append(parts, "-H", escapeShellArg("Content-Type: application/json"))
				}
				payload := map[string]any{
					"query": *req.Body.GraphQL.Query,
				}
				if req.Body.GraphQL.Variables != nil && *req.Body.GraphQL.Variables != "" {
					var v any
					if err := json.Unmarshal([]byte(*req.Body.GraphQL.Variables), &v); err == nil {
						payload["variables"] = v
					} else {
						payload["variables"] = *req.Body.GraphQL.Variables
					}
				}
				if b, err := json.Marshal(payload); err == nil {
					parts = append(parts, "--data-raw", escapeShellArg(string(b)))
				}
			}
		case model.BodyText:
			if req.Body.Text != nil && *req.Body.Text != "" {
				parts = append(parts, "--data-raw", escapeShellArg(*req.Body.Text))
			}
		case model.BodyXML:
			if req.Body.XML != nil && *req.Body.XML != "" {
				if !hasContentType {
					parts = append(parts, "-H", escapeShellArg("Content-Type: application/xml"))
				}
				parts = append(parts, "--data-raw", escapeShellArg(*req.Body.XML))
			}
		case model.BodySparql:
			if req.Body.Sparql != nil && *req.Body.Sparql != "" {
				parts = append(parts, "--data-raw", escapeShellArg(*req.Body.Sparql))
			}
		case model.BodyFormURLEncoded:
			for _, kv := range req.Body.FormURLEncoded {
				if kv.Enabled && kv.NameOrEmpty() != "" {
					entry := fmt.Sprintf("%s=%s", kv.NameOrEmpty(), kv.ValueOrEmpty())
					parts = append(parts, "--data-urlencode", escapeShellArg(entry))
				}
			}
		case model.BodyMultipartForm:
			for _, part := range req.Body.MultipartForm {
				if !part.Enabled || part.Name == nil || *part.Name == "" {
					continue
				}
				name := *part.Name
				val := ""
				if part.Value != nil {
					val = *part.Value
				}
				if part.Type == model.MultipartFile {
					filePath := val
					if strings.HasPrefix(val, "@file(") && strings.HasSuffix(val, ")") {
						filePath = strings.TrimSuffix(strings.TrimPrefix(val, "@file("), ")")
					}
					var formArg string
					if part.ContentType != nil && *part.ContentType != "" {
						formArg = fmt.Sprintf("%s=@%s;type=%s", name, filePath, *part.ContentType)
					} else {
						formArg = fmt.Sprintf("%s=@%s", name, filePath)
					}
					parts = append(parts, "-F", escapeShellArg(formArg))
				} else {
					parts = append(parts, "-F", escapeShellArg(fmt.Sprintf("%s=%s", name, val)))
				}
			}
		case model.BodyFile:
			for _, f := range req.Body.File {
				if f.Selected && f.FilePath != nil && *f.FilePath != "" {
					parts = append(parts, "--data-binary", escapeShellArg(fmt.Sprintf("@%s", *f.FilePath)))
					break
				}
			}
		}
	}

	if !opts.Multiline {
		return strings.Join(parts, " "), nil
	}

	// Format into multiline command with \ continuations
	var b strings.Builder
	b.WriteString("curl")
	i := 1
	for i < len(parts) {
		token := parts[i]
		b.WriteString(" \\\n  ")
		b.WriteString(token)

		// Check if this token is a flag with an argument
		switch token {
		case "-X", "-H", "-u", "-d", "--data-raw", "--data-urlencode", "-F", "--data-binary", "--digest", "--ntlm", "--aws-sigv4":
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
				i++
				b.WriteString(" ")
				b.WriteString(parts[i])
			}
		}
		i++
	}

	return b.String(), nil
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
		// If contains template braces, don't query escape
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

func escapeShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
