package bru

import (
	"fmt"
	"strconv"
	"strings"
)

// sprintf is a thin alias so parser.go reads cleanly.
func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

// strconvQuote renders a name for an error message.
func strconvQuote(s string) string { return strconv.Quote(s) }

// isTextBlock reports whether a block header holds raw text rather than
// `key: value` pairs.
//
// This distinction comes from Bruno's grammar: the `bodies` and `script` rules
// name text-bearing blocks explicitly, while `meta`, `get`, `headers`, `auth:*`,
// `settings`, `app`, `assert` and the params blocks are dictionaries. Getting this
// wrong is the difference between parsing a JSON body and choking on it.
func isTextBlock(name string) bool {
	switch name {
	case "tests", "docs", "example":
		return true
	}
	// body:json, body:text, body:xml, body:sparql, body:graphql,
	// body:graphql:vars, body:grpc, body:ws.
	// The form/multipart/file bodies are dictionaries in the grammar.
	if strings.HasPrefix(name, "body:") {
		switch name {
		case "body:form-urlencoded", "body:multipart-form", "body:file":
			return false
		}
		return true
	}
	// script:pre-request, script:post-response and the gRPC hooks.
	if strings.HasPrefix(name, "script:") {
		return true
	}
	// `assert` is parsed with assertdictionary, a dictionary variant whose keys
	// may contain dots and whose values may contain spaces.
	return false
}

// methodBlocks are the block names that carry an HTTP method.
var methodBlocks = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"delete":  "DELETE",
	"patch":   "PATCH",
	"options": "OPTIONS",
	"head":    "HEAD",
	"trace":   "TRACE",
	"connect": "CONNECT",
}

// knownBlock reports whether a top-level block name is part of the grammar.
//
// Bruno's grammar is closed: an unknown block is a parse error, which catches
// typos instead of silently dropping data.
func knownBlock(name string) bool {
	if _, ok := methodBlocks[name]; ok {
		return true
	}
	switch name {
	case "meta", "http", "grpc", "ws", "app", "settings", "docs",
		"headers", "vars", "auth", "assert", "tests", "example",
		"params:query", "params:path",
		"body:json", "body:text", "body:xml", "body:sparql", "body:graphql",
		"body:graphql:vars", "body:form-urlencoded", "body:multipart-form",
		"body:file", "body:grpc", "body:ws",
		"vars:pre-request", "vars:post-response", "vars:secret",
		"vars:externalsecrets:vault",
		"script:pre-request", "script:post-response",
		"script:grpc:before-call-start", "script:grpc:before-message-send",
		"script:grpc:after-message-receive", "script:grpc:after-call-end":
		return true
	}
	// auth:<mode> and auth:oauth2:additional_params:…:… blocks.
	if strings.HasPrefix(name, "auth:") {
		return true
	}
	return false
}

// Errors used by the small integer parser in auth.go.
var (
	errEmptyInt = &SyntaxError{Msg: "empty integer"}
	errBadInt   = &SyntaxError{Msg: "invalid integer"}
)
