package bru

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// writer builds `.bru` text.
//
// The output is semantics-preserving rather than byte-exact: block order and
// key order are fixed by Bruno's own serialiser, so a file that this writer
// produces is stable across saves but need not match the original byte for byte.
// Round-trip tests assert parse(write(parse(x))) == parse(x) rather than byte
// equality, which is the property that actually matters.
type writer struct {
	b strings.Builder
}

// line writes an indented line.
func (w *writer) line(indent int, format string, args ...any) {
	w.b.WriteString(strings.Repeat("  ", indent))
	if len(args) == 0 {
		w.b.WriteString(format)
	} else {
		fmt.Fprintf(&w.b, format, args...)
	}
	w.b.WriteByte('\n')
}

// blank writes a separating blank line.
func (w *writer) blank() { w.b.WriteByte('\n') }

// block opens a named block.
func (w *writer) open(indent int, name string) { w.line(indent, "%s {", name) }

// closeBlock writes the closing brace at the given indent.
//
// A closing brace must be at column 0 for a top-level block: that is the rule
// the parser relies on to find text-block terminators.
func (w *writer) closeBlock(indent int) { w.line(indent, "}") }

// String returns the accumulated text.
func (w *writer) String() string { return w.b.String() }

// quoteKey quotes a key when it contains characters that would break parsing.
func quoteKey(k string) string {
	if strings.ContainsAny(k, `:"{} `) {
		return `"` + strings.ReplaceAll(k, `"`, `\"`) + `"`
	}
	return k
}

// valueString renders a scalar value, wrapping multiline values in ”'.
func valueString(v string) string {
	if !strings.ContainsAny(v, "\n\r") {
		return v
	}
	var b strings.Builder
	b.WriteString("'''\n")
	for _, line := range strings.Split(strings.ReplaceAll(v, "\r\n", "\n"), "\n") {
		b.WriteString("  ") // two-space indent inside a pair
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("  '''")
	return b.String()
}

// writePair writes one `key: value` row, with `~` when disabled and any
// annotations on their own line above.
func (w *writer) writePair(indent int, p model.KeyValue) {
	for _, ann := range p.Annotations {
		if ann.Value == "" {
			w.line(indent, "@%s", ann.Name)
		} else {
			w.line(indent, "@%s('%s')", ann.Name, ann.Value)
		}
	}
	if p.Description != nil && *p.Description != "" {
		w.line(indent, "@description('%s')", strings.ReplaceAll(*p.Description, "'", `\'`))
	}
	prefix := ""
	if !p.Enabled {
		prefix = "~"
	}
	w.line(indent, "%s%s: %s", prefix, quoteKey(p.NameOrEmpty()), valueString(p.ValueOrEmpty()))
}

// writeVariable writes one variable row, including its type annotation.
func (w *writer) writeVariable(indent int, v model.Variable) {
	switch v.DataType {
	case "number", "boolean", "object":
		w.line(indent, "@%s", v.DataType)
	}
	if v.Description != nil && *v.Description != "" {
		w.line(indent, "@description('%s')", strings.ReplaceAll(*v.Description, "'", `\'`))
	}
	prefix := ""
	if !v.Enabled {
		prefix = "~"
	}
	w.line(indent, "%s%s: %s", prefix, quoteKey(v.NameOrEmpty()), valueString(variableValueString(v.Value)))
}

// variableValueString renders a variable's value.
//
// Typed values are written as their JSON form, which is what Bruno does for
// objects; strings are written bare.
func variableValueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// writeRequest serialises a request into `.bru` text.
//
// Block order follows Bruno's jsonToBru: meta, method, params, headers, auth,
// body, script, tests, vars, assertions, settings, docs.
func writeRequest(req *model.Request) string {
	w := &writer{}

	// meta
	if req.Name != "" || req.Seq != nil || len(req.Tags) > 0 {
		w.open(0, "meta")
		if req.Name != "" {
			w.line(1, "name: %s", req.Name)
		}
		w.line(1, "type: http")
		if req.Seq != nil {
			w.line(1, "seq: %d", *req.Seq)
		}
		if len(req.Tags) > 0 {
			w.line(1, "tags: [")
			for _, t := range req.Tags {
				w.line(2, "%s", t)
			}
			w.line(1, "]")
		}
		w.closeBlock(0)
		w.blank()
	}

	// method block
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}
	lower := strings.ToLower(method)
	if _, isStandard := methodBlocks[lower]; isStandard {
		w.line(0, "%s {", lower)
	} else {
		w.open(0, "http")
		w.line(1, "method: %s", method)
	}
	w.line(1, "url: %s", req.URL)
	if req.Body != nil && req.Body.Mode != model.BodyNone && req.Body.Mode != "" {
		w.line(1, "body: %s", req.Body.Mode)
	}
	if req.Auth != nil && req.Auth.Mode != "" {
		w.line(1, "auth: %s", req.Auth.Mode)
	}
	w.closeBlock(0)
	w.blank()

	// params
	for _, group := range []struct {
		name string
		typ  model.ParamType
	}{
		{"params:query", model.ParamQuery},
		{"params:path", model.ParamPath},
	} {
		var rows []model.KeyValue
		for _, p := range req.Params {
			if p.Type == group.typ {
				rows = append(rows, p.KeyValue)
			}
		}
		if len(rows) == 0 {
			continue
		}
		w.open(0, group.name)
		for _, r := range rows {
			w.writePair(1, r)
		}
		w.closeBlock(0)
		w.blank()
	}

	// headers
	if len(req.Headers) > 0 {
		w.open(0, "headers")
		for _, h := range req.Headers {
			w.writePair(1, h)
		}
		w.closeBlock(0)
		w.blank()
	}

	// auth
	writeAuth(w, req.Auth)

	// body
	writeBody(w, req.Body)

	// scripts
	if req.Script != nil {
		if req.Script.Req != nil {
			writeTextBlock(w, "script:pre-request", *req.Script.Req)
		}
		if req.Script.Res != nil {
			writeTextBlock(w, "script:post-response", *req.Script.Res)
		}
	}

	// tests
	if req.Tests != nil && strings.TrimSpace(*req.Tests) != "" {
		writeTextBlock(w, "tests", *req.Tests)
	}

	// vars
	if req.Vars != nil {
		if len(req.Vars.PreRequest) > 0 {
			w.open(0, "vars:pre-request")
			for _, v := range req.Vars.PreRequest {
				w.writeVariable(1, v)
			}
			w.closeBlock(0)
			w.blank()
		}
		if len(req.Vars.PostResponse) > 0 {
			w.open(0, "vars:post-response")
			for _, v := range req.Vars.PostResponse {
				w.writeVariable(1, v)
			}
			w.closeBlock(0)
			w.blank()
		}
	}

	// assertions
	if len(req.Assertions) > 0 {
		w.open(0, "assert")
		for _, a := range req.Assertions {
			w.line(1, "%s: %s", a.NameOrEmpty(), a.ValueOrEmpty())
		}
		w.closeBlock(0)
		w.blank()
	}

	// settings
	writeSettings(w, req.Settings)

	// docs
	if req.Docs != nil && strings.TrimSpace(*req.Docs) != "" {
		writeTextBlock(w, "docs", *req.Docs)
	}

	return strings.TrimRight(w.String(), "\n") + "\n"
}

// writeTextBlock writes a raw-text block.
//
// Every content line is indented by two spaces and the closing brace goes at
// column 0. The indent is not cosmetic: the parser outdents text bodies by two
// spaces (outdentString(str, 2)), so writing unindented lines would strip two
// real characters from the body on every save. Blank lines stay blank.
//
// A trailing newline is preserved: the parser produces text that ends just before
// the terminator's newline, and some fixtures genuinely end with a blank line, so
// a plain TrimRight would lose information on the second round trip.
func writeTextBlock(w *writer, name, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	w.line(0, "%s {", name)
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			w.b.WriteString("  ")
		}
		w.b.WriteString(line)
		w.b.WriteByte('\n')
	}
	w.line(0, "}")
	w.blank()
}

// writeAuth writes the mode marker and the mode's field block.
func writeAuth(w *writer, auth *model.Auth) {
	if auth == nil || auth.Mode == "" {
		return
	}
	w.open(0, "auth")
	w.line(1, "mode: %s", auth.Mode)
	w.closeBlock(0)
	w.blank()

	fields := authFields(auth)
	if len(fields) == 0 {
		return
	}
	w.open(0, "auth:"+string(auth.Mode))
	for _, f := range fields {
		w.line(1, "%s: %s", f.key, f.value)
	}
	w.closeBlock(0)
	w.blank()
}

// field is one serialised auth key/value pair.
type field struct{ key, value string }

// authFields lists the non-empty fields for the configured mode.
//
// Only the active mode's fields are written; Bruno keeps other modes' values in
// the same object for the editor's convenience but does not persist them.
func authFields(a *model.Auth) []field {
	var out []field
	add := func(k string, v *string) {
		if v != nil && *v != "" {
			out = append(out, field{k, *v})
		}
	}
	addBool := func(k string, v *bool) {
		if v != nil {
			out = append(out, field{k, strconv.FormatBool(*v)})
		}
	}
	addInt := func(k string, v *int) {
		if v != nil {
			out = append(out, field{k, strconv.Itoa(*v)})
		}
	}

	switch a.Mode {
	case model.AuthBasic, model.AuthDigest:
		add("username", a.Username)
		add("password", a.Password)
	case model.AuthWSSE:
		add("username", a.Username)
		add("password", a.Password)
	case model.AuthNTLM:
		add("username", a.Username)
		add("password", a.Password)
		add("domain", a.Domain)
	case model.AuthBearer:
		add("token", a.Token)
	case model.AuthAPIKey:
		add("key", a.Key)
		add("value", a.Value)
		if a.Placement != "" {
			out = append(out, field{"placement", string(a.Placement)})
		}
	case model.AuthAWSV4:
		add("accessKeyId", a.AccessKeyID)
		add("secretAccessKey", a.SecretAccessKey)
		add("sessionToken", a.SessionToken)
		add("service", a.Service)
		add("region", a.Region)
		add("profileName", a.ProfileName)
	case model.AuthOAuth1:
		add("consumerKey", a.ConsumerKey)
		add("consumerSecret", a.ConsumerSecret)
		add("accessToken", a.AccessToken)
		add("accessTokenSecret", a.AccessTokenSecret)
		add("callbackUrl", a.CallbackURL)
		add("verifier", a.Verifier)
		add("signatureMethod", a.SignatureMethod)
		add("privateKey", a.PrivateKey)
		add("privateKeyType", a.PrivateKeyType)
		add("timestamp", a.Timestamp)
		add("nonce", a.Nonce)
		add("version", a.Version)
		add("realm", a.Realm)
		addBool("includeBodyHash", a.IncludeBodyHash)
	case model.AuthOAuth2:
		if a.GrantType != "" {
			out = append(out, field{"grantType", string(a.GrantType)})
		}
		add("username", a.Username)
		add("password", a.Password)
		add("callbackUrl", a.CallbackURL)
		add("authorizationUrl", a.AuthorizationURL)
		add("accessTokenUrl", a.AccessTokenURL)
		add("refreshTokenUrl", a.RefreshTokenURL)
		add("clientId", a.ClientID)
		add("clientSecret", a.ClientSecret)
		add("scope", a.Scope)
		add("state", a.State)
		addBool("pkce", a.PKCE)
		add("credentialsPlacement", a.CredentialsPlacement)
		add("credentialsId", a.CredentialsID)
		add("tokenPlacement", a.TokenPlacement)
		add("tokenHeaderPrefix", a.TokenHeaderPrefix)
		add("tokenQueryKey", a.TokenQueryKey)
		addBool("autoRefreshToken", a.AutoRefreshToken)
		addBool("autoFetchToken", a.AutoFetchToken)
		add("tokenSource", a.TokenSource)
	case model.AuthEdgeGrid:
		add("clientToken", a.EdgeGridClientToken)
		add("clientSecret", a.EdgeGridClientSecret)
		add("baseURL", a.EdgeGridBaseURL)
		add("headersToSign", a.EdgeGridHeadersToSign)
		addInt("maxBodySize", a.EdgeGridMaxBodySize)
	}
	return out
}

// writeBody writes the body block matching the configured mode.
func writeBody(w *writer, body *model.Body) {
	if body == nil {
		return
	}
	switch body.Mode {
	case model.BodyJSON:
		writeTextBlock(w, "body:json", model.DerefString(body.JSON))
	case model.BodyText:
		writeTextBlock(w, "body:text", model.DerefString(body.Text))
	case model.BodyXML:
		writeTextBlock(w, "body:xml", model.DerefString(body.XML))
	case model.BodySparql:
		writeTextBlock(w, "body:sparql", model.DerefString(body.Sparql))
	case model.BodyGraphQL:
		if body.GraphQL != nil {
			writeTextBlock(w, "body:graphql", model.DerefString(body.GraphQL.Query))
			writeTextBlock(w, "body:graphql:vars", model.DerefString(body.GraphQL.Variables))
		}
	case model.BodyFormURLEncoded:
		if len(body.FormURLEncoded) == 0 {
			return
		}
		w.open(0, "body:form-urlencoded")
		for _, kv := range body.FormURLEncoded {
			w.writePair(1, kv)
		}
		w.closeBlock(0)
		w.blank()
	case model.BodyMultipartForm:
		if len(body.MultipartForm) == 0 {
			return
		}
		w.open(0, "body:multipart-form")
		for _, part := range body.MultipartForm {
			w.writeMultipart(1, part)
		}
		w.closeBlock(0)
		w.blank()
	case model.BodyFile:
		if len(body.File) == 0 {
			return
		}
		w.open(0, "body:file")
		for _, f := range body.File {
			path := model.DerefString(f.FilePath)
			w.line(1, "%s%s: %s", selectedPrefix(f), quoteKey("file"), path)
			if f.ContentType != nil && *f.ContentType != "" {
				w.line(1, "content-type: %s", *f.ContentType)
			}
		}
		w.closeBlock(0)
		w.blank()
	}
}

// selectedPrefix marks an unselected file row with `~`, matching how the editor
// represents "this file is not the one being sent".
func selectedPrefix(f model.FileEntry) string {
	if f.Selected {
		return ""
	}
	return "~"
}

// writeMultipart writes one multipart row with its inline annotations.
func (w *writer) writeMultipart(indent int, part model.MultipartEntry) {
	prefix := ""
	if !part.Enabled {
		prefix = "~"
	}
	name := quoteKey(model.DerefString(part.Name))
	value := model.DerefString(part.Value)

	var suffix string
	if part.Type == model.MultipartFile {
		suffix = fmt.Sprintf(" @file(%s)", value)
	}
	if part.ContentType != nil && *part.ContentType != "" {
		suffix += fmt.Sprintf(" @contentType(%s)", *part.ContentType)
	}
	if part.Type == model.MultipartFile && part.ContentType == nil {
		w.line(indent, "%s%s: %s", prefix, name, suffix)
		return
	}
	w.line(indent, "%s%s: %s%s", prefix, name, valueString(value), suffix)
}

// writeSettings writes the settings block, omitting unset fields.
func writeSettings(w *writer, s model.ItemSettings) {
	var lines []string
	if s.EncodeURL != nil {
		lines = append(lines, fmt.Sprintf("encodeUrl: %t", *s.EncodeURL))
	}
	if s.FollowRedirects != nil {
		lines = append(lines, fmt.Sprintf("followRedirects: %t", *s.FollowRedirects))
	}
	if s.MaxRedirects != nil {
		lines = append(lines, fmt.Sprintf("maxRedirects: %d", *s.MaxRedirects))
	}
	if s.Timeout != nil {
		lines = append(lines, "timeout: "+s.Timeout.String())
	}
	if s.ForwardAuthorizationHeader != nil {
		lines = append(lines, fmt.Sprintf("forwardAuthorizationHeader: %t", *s.ForwardAuthorizationHeader))
	}
	if s.DisableParsingResponseJSON != nil {
		lines = append(lines, fmt.Sprintf("disableParsingResponseJson: %t", *s.DisableParsingResponseJSON))
	}
	if len(s.OmitHeaders) > 0 {
		sorted := append([]string(nil), s.OmitHeaders...)
		sort.Strings(sorted)
		lines = append(lines, "omitHeaders: "+strings.Join(sorted, ","))
	}
	if len(lines) == 0 {
		return
	}
	w.open(0, "settings")
	for _, l := range lines {
		w.line(1, "%s", l)
	}
	w.closeBlock(0)
	w.blank()
}

// writeFolder serialises a folder file.
func writeFolder(item *model.Item) string {
	w := &writer{}
	w.open(0, "meta")
	if item.Name != "" {
		w.line(1, "name: %s", item.Name)
	}
	if item.Seq != nil {
		w.line(1, "seq: %d", *item.Seq)
	}
	if len(item.Tags) > 0 {
		w.line(1, "tags: [")
		for _, t := range item.Tags {
			w.line(2, "%s", t)
		}
		w.line(1, "]")
	}
	w.closeBlock(0)
	w.blank()

	if item.Auth != nil {
		writeAuth(w, item.Auth)
	}
	if len(item.Headers) > 0 {
		w.open(0, "headers")
		for _, h := range item.Headers {
			w.writePair(1, h)
		}
		w.closeBlock(0)
		w.blank()
	}
	if len(item.Vars) > 0 {
		w.open(0, "vars:pre-request")
		for _, v := range item.Vars {
			w.writeVariable(1, v)
		}
		w.closeBlock(0)
		w.blank()
	}
	if item.Script != nil {
		if item.Script.Req != nil {
			writeTextBlock(w, "script:pre-request", *item.Script.Req)
		}
		if item.Script.Res != nil {
			writeTextBlock(w, "script:post-response", *item.Script.Res)
		}
	}
	return strings.TrimRight(w.String(), "\n") + "\n"
}

// writeEnvironment serialises an environment file.
//
// The order matches Bruno: vars, vars:secret, then the top-level extends and
// color rules.
func writeEnvironment(env *model.Environment) string {
	w := &writer{}

	if len(env.Variables) > 0 {
		w.open(0, "vars")
		for _, v := range env.Variables {
			w.writeVariable(1, v)
		}
		w.closeBlock(0)
		w.blank()
	}

	if len(env.Secrets) > 0 {
		w.line(0, "vars:secret [")
		for i, s := range env.Secrets {
			sep := ","
			if i == len(env.Secrets)-1 {
				sep = ""
			}
			w.line(1, "%s%s", s, sep)
		}
		w.line(0, "]")
		w.blank()
	}

	if env.Extends != "" {
		w.line(0, "extends: %s", env.Extends)
	}
	if env.Color != "" {
		w.line(0, "color: %s", env.Color)
	}

	return strings.TrimRight(w.String(), "\n") + "\n"
}
