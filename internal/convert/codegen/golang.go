package codegen

import (
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

func init() {
	Register(TargetGo, GenerateGo)
}

// GenerateGo produces idiomatic Go net/http code from a model.Request.
func GenerateGo(req *model.Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request cannot be nil")
	}

	targetURL := synthesizeURL(req.URL, req.Params)
	method := req.Method
	if method == "" {
		method = "GET"
	}

	imports := map[string]bool{
		"fmt":      true,
		"net/http": true,
		"io":       true,
	}

	var bodySetup strings.Builder
	var bodyArg = "nil"

	if req.Body != nil {
		switch req.Body.Mode {
		case model.BodyJSON:
			if req.Body.JSON != nil && *req.Body.JSON != "" {
				imports["strings"] = true
				bodySetup.WriteString(fmt.Sprintf("\tpayload := strings.NewReader(%s)\n", quoteGoString(*req.Body.JSON)))
				bodyArg = "payload"
			}
		case model.BodyText:
			if req.Body.Text != nil && *req.Body.Text != "" {
				imports["strings"] = true
				bodySetup.WriteString(fmt.Sprintf("\tpayload := strings.NewReader(%s)\n", quoteGoString(*req.Body.Text)))
				bodyArg = "payload"
			}
		case model.BodyXML:
			if req.Body.XML != nil && *req.Body.XML != "" {
				imports["strings"] = true
				bodySetup.WriteString(fmt.Sprintf("\tpayload := strings.NewReader(%s)\n", quoteGoString(*req.Body.XML)))
				bodyArg = "payload"
			}
		case model.BodySparql:
			if req.Body.Sparql != nil && *req.Body.Sparql != "" {
				imports["strings"] = true
				bodySetup.WriteString(fmt.Sprintf("\tpayload := strings.NewReader(%s)\n", quoteGoString(*req.Body.Sparql)))
				bodyArg = "payload"
			}
		case model.BodyFormURLEncoded:
			imports["net/url"] = true
			imports["strings"] = true
			bodySetup.WriteString("\tformData := url.Values{}\n")
			for _, kv := range req.Body.FormURLEncoded {
				if kv.Enabled && kv.NameOrEmpty() != "" {
					bodySetup.WriteString(fmt.Sprintf("\tformData.Set(%q, %q)\n", kv.NameOrEmpty(), kv.ValueOrEmpty()))
				}
			}
			bodySetup.WriteString("\tpayload := strings.NewReader(formData.Encode())\n")
			bodyArg = "payload"
		case model.BodyFile:
			for _, f := range req.Body.File {
				if f.Selected && f.FilePath != nil && *f.FilePath != "" {
					imports["os"] = true
					bodySetup.WriteString(fmt.Sprintf("\tfile, err := os.Open(%q)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tdefer file.Close()\n", *f.FilePath))
					bodyArg = "file"
					break
				}
			}
		}
	}

	var headerSetup strings.Builder
	hasContentType := false
	hasAuthHeader := false
	for _, h := range req.LiveHeaders() {
		k := h.NameOrEmpty()
		v := h.ValueOrEmpty()
		if strings.EqualFold(k, "Content-Type") {
			hasContentType = true
		}
		if strings.EqualFold(k, "Authorization") {
			hasAuthHeader = true
		}
		headerSetup.WriteString(fmt.Sprintf("\treq.Header.Set(%q, %q)\n", k, v))
	}

	if req.Body != nil && !hasContentType {
		switch req.Body.Mode {
		case model.BodyJSON:
			headerSetup.WriteString("\treq.Header.Set(\"Content-Type\", \"application/json\")\n")
		case model.BodyXML:
			headerSetup.WriteString("\treq.Header.Set(\"Content-Type\", \"application/xml\")\n")
		case model.BodyFormURLEncoded:
			headerSetup.WriteString("\treq.Header.Set(\"Content-Type\", \"application/x-www-form-urlencoded\")\n")
		}
	}

	var authSetup strings.Builder
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
			authSetup.WriteString(fmt.Sprintf("\treq.SetBasicAuth(%q, %q)\n", u, p))
		case model.AuthBearer:
			if !hasAuthHeader && req.Auth.Token != nil && *req.Auth.Token != "" {
				authSetup.WriteString(fmt.Sprintf("\treq.Header.Set(\"Authorization\", \"Bearer \" + %q)\n", *req.Auth.Token))
			}
		}
	}

	// Format imports
	var importBuf strings.Builder
	importBuf.WriteString("import (\n")
	for imp := range imports {
		importBuf.WriteString(fmt.Sprintf("\t%q\n", imp))
	}
	importBuf.WriteString(")\n\n")

	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString(importBuf.String())
	b.WriteString("func main() {\n")
	b.WriteString(fmt.Sprintf("\turl := %q\n", targetURL))
	if bodySetup.Len() > 0 {
		b.WriteString(bodySetup.String())
	}
	b.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(%q, url, %s)\n", method, bodyArg))
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")

	if headerSetup.Len() > 0 {
		b.WriteString(headerSetup.String())
	}
	if authSetup.Len() > 0 {
		b.WriteString(authSetup.String())
	}

	b.WriteString("\n\tclient := &http.Client{}\n")
	b.WriteString("\tres, err := client.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tdefer res.Body.Close()\n\n")
	b.WriteString("\tbody, err := io.ReadAll(res.Body)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tfmt.Println(string(body))\n")
	b.WriteString("}\n")

	return b.String(), nil
}

func quoteGoString(s string) string {
	if !strings.Contains(s, "`") && strings.Contains(s, "\n") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}
