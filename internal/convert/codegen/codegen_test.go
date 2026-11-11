package codegen

import (
	"strings"
	"testing"

	"fearless/internal/convert/curl"
	"fearless/internal/core/model"
)

func strPtr(s string) *string {
	return &s
}

func sampleRequest() *model.Request {
	jsonPayload := `{"name":"alice","role":"developer"}`
	return &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Params: []model.Param{
			{
				Type: model.ParamQuery,
				KeyValue: model.KeyValue{
					Name:    strPtr("active"),
					Value:   strPtr("true"),
					Enabled: true,
				},
			},
		},
		Headers: []model.KeyValue{
			{
				Name:    strPtr("X-Custom-Token"),
				Value:   strPtr("custom-token-val"),
				Enabled: true,
			},
		},
		Auth: &model.Auth{
			Mode:     model.AuthBasic,
			Username: strPtr("admin"),
			Password: strPtr("secret"),
		},
		Body: &model.Body{
			Mode: model.BodyJSON,
			JSON: &jsonPayload,
		},
	}
}

func TestRegistryAndListTargets(t *testing.T) {
	targets := ListTargets()
	if len(targets) != 5 {
		t.Fatalf("expected 5 targets, got %d", len(targets))
	}

	// Unknown target
	req := sampleRequest()
	_, err := Generate("unknown-target", req)
	if err == nil {
		t.Error("expected error on unknown target")
	}

	// Nil request
	_, err = Generate(TargetCurl, nil)
	if err == nil {
		t.Error("expected error on nil request")
	}
}

func TestGenerateCurl(t *testing.T) {
	req := sampleRequest()
	code, err := Generate(TargetCurl, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(code, "curl") {
		t.Errorf("expected curl command, got: %s", code)
	}
	if !strings.Contains(code, "-X POST") {
		t.Errorf("missing method in curl snippet: %s", code)
	}
	if !strings.Contains(code, "https://api.example.com/users?active=true") {
		t.Errorf("missing synthesized url in curl snippet: %s", code)
	}

	// Byte-for-byte / parse verification: import back the generated curl
	imported, err := curl.Import(code)
	if err != nil {
		t.Fatalf("failed to re-import generated curl: %v", err)
	}
	if imported.Request.Method != req.Method {
		t.Errorf("method mismatch: %s vs %s", imported.Request.Method, req.Method)
	}
	if imported.Request.URL != req.URL {
		t.Errorf("url mismatch: %s vs %s", imported.Request.URL, req.URL)
	}
}

func TestGenerateGo(t *testing.T) {
	req := sampleRequest()
	code, err := Generate(TargetGo, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"package main",
		"http.NewRequest(\"POST\", url, payload)",
		"req.SetBasicAuth(\"admin\", \"secret\")",
		"req.Header.Set(\"X-Custom-Token\", \"custom-token-val\")",
		"client.Do(req)",
	}
	for _, part := range expectedParts {
		if !strings.Contains(code, part) {
			t.Errorf("missing Go code part: %s in\n%s", part, code)
		}
	}

	// Go with Bearer auth and FormURLEncoded body
	token := "bearer-token"
	formReq := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/login",
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: &token,
		},
		Body: &model.Body{
			Mode: model.BodyFormURLEncoded,
			FormURLEncoded: []model.KeyValue{
				{
					Name:    strPtr("user"),
					Value:   strPtr("alice"),
					Enabled: true,
				},
			},
		},
	}
	code, err = GenerateGo(formReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Authorization") || !strings.Contains(code, "formData.Set(\"user\", \"alice\")") {
		t.Errorf("form code mismatch: %s", code)
	}

	// Go with Text, XML, Sparql, File bodies
	txt := "plain-txt"
	code, _ = GenerateGo(&model.Request{Method: "POST", URL: "http://x.com", Body: &model.Body{Mode: model.BodyText, Text: &txt}})
	if !strings.Contains(code, "plain-txt") {
		t.Errorf("text mismatch: %s", code)
	}

	xml := "<xml/>"
	code, _ = GenerateGo(&model.Request{Method: "POST", URL: "http://x.com", Body: &model.Body{Mode: model.BodyXML, XML: &xml}})
	if !strings.Contains(code, "<xml/>") {
		t.Errorf("xml mismatch: %s", code)
	}

	sparql := "SELECT"
	code, _ = GenerateGo(&model.Request{Method: "POST", URL: "http://x.com", Body: &model.Body{Mode: model.BodySparql, Sparql: &sparql}})
	if !strings.Contains(code, "SELECT") {
		t.Errorf("sparql mismatch: %s", code)
	}

	fpath := "/tmp/file"
	code, _ = GenerateGo(&model.Request{Method: "POST", URL: "http://x.com", Body: &model.Body{Mode: model.BodyFile, File: []model.FileEntry{{FilePath: &fpath, Selected: true}}}})
	if !strings.Contains(code, "/tmp/file") {
		t.Errorf("file mismatch: %s", code)
	}
}

func TestGenerateJSFetch(t *testing.T) {
	req := sampleRequest()
	code, err := Generate(TargetJSFetch, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "fetch(url, options)") {
		t.Errorf("missing fetch call in: %s", code)
	}
	if !strings.Contains(code, "POST") {
		t.Errorf("missing method in: %s", code)
	}
	if !strings.Contains(code, "Basic") {
		t.Errorf("missing basic auth in: %s", code)
	}
	if !strings.Contains(code, "JSON.stringify") {
		t.Errorf("missing JSON.stringify in: %s", code)
	}

	// Bearer auth & URLSearchParams
	token := "token-val"
	formReq := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/form",
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: &token,
		},
		Body: &model.Body{
			Mode: model.BodyFormURLEncoded,
			FormURLEncoded: []model.KeyValue{
				{
					Name:    strPtr("foo"),
					Value:   strPtr("bar"),
					Enabled: true,
				},
			},
		},
	}
	code, err = GenerateJSFetch(formReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "URLSearchParams") || !strings.Contains(code, "Bearer token-val") {
		t.Errorf("fetch form mismatch: %s", code)
	}

	// Text & XML
	xml := "<data/>"
	code, _ = GenerateJSFetch(&model.Request{Method: "PUT", URL: "http://x.com", Body: &model.Body{Mode: model.BodyXML, XML: &xml}})
	if !strings.Contains(code, "<data/>") {
		t.Errorf("xml mismatch: %s", code)
	}
}

func TestGenerateJSAxios(t *testing.T) {
	req := sampleRequest()
	code, err := Generate(TargetJSAxios, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "require('axios')") {
		t.Errorf("missing axios require in: %s", code)
	}
	if !strings.Contains(code, "axios(config)") {
		t.Errorf("missing axios call in: %s", code)
	}
	if !strings.Contains(code, "username: \"admin\"") {
		t.Errorf("missing basic auth username in: %s", code)
	}

	// Bearer auth
	token := "token-val"
	reqBearer := &model.Request{
		Method: "GET",
		URL:    "https://api.example.com/me",
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: &token,
		},
	}
	code, _ = GenerateJSAxios(reqBearer)
	if !strings.Contains(code, "Bearer token-val") {
		t.Errorf("missing bearer auth in: %s", code)
	}
}

func TestGeneratePython(t *testing.T) {
	req := sampleRequest()
	code, err := Generate(TargetPythonRequests, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "import requests") {
		t.Errorf("missing import requests in: %s", code)
	}
	if !strings.Contains(code, "requests.request(") {
		t.Errorf("missing requests.request in: %s", code)
	}
	if !strings.Contains(code, `auth = ("admin", "secret")`) {
		t.Errorf("missing auth tuple in: %s", code)
	}
	if !strings.Contains(code, "payload = ") {
		t.Errorf("missing payload in: %s", code)
	}

	// Python FormURLEncoded
	formReq := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com/login",
		Body: &model.Body{
			Mode: model.BodyFormURLEncoded,
			FormURLEncoded: []model.KeyValue{
				{
					Name:    strPtr("user"),
					Value:   strPtr("bob"),
					Enabled: true,
				},
			},
		},
	}
	code, _ = GeneratePython(formReq)
	if !strings.Contains(code, `"user": "bob"`) {
		t.Errorf("missing form data in python code: %s", code)
	}

	// Python Bearer auth
	token := "py-token"
	bearerReq := &model.Request{
		Method: "GET",
		URL:    "https://api.example.com/me",
		Auth: &model.Auth{
			Mode:  model.AuthBearer,
			Token: &token,
		},
	}
	code, _ = GeneratePython(bearerReq)
	if !strings.Contains(code, "Bearer py-token") {
		t.Errorf("missing bearer token in python code: %s", code)
	}
}
