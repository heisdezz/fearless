package yamlcodec

import (
	"fmt"

	"fearless/internal/core/model"
)

// uidPrefix keeps generated row UIDs distinguishable from `.bru` ones.
const uidPrefix = "yml-"

// rowUID builds a deterministic UID for a row.
func rowUID(kind string, index int, name string) string {
	return fmt.Sprintf("%s%s-%d-%x", uidPrefix, kind, index, len(name))
}

// requestFromDoc maps a parsed request document onto the model.
func requestFromDoc(doc *requestDoc) (*model.Request, error) {
	req := &model.Request{
		Method: "GET",
		Name:   doc.Info.Name,
		Seq:    doc.Info.Seq,
		Tags:   append([]string(nil), doc.Info.Tags...),
	}

	if h := doc.HTTP; h != nil {
		if h.Method != "" {
			req.Method = h.Method
		}
		req.URL = h.URL

		for i, kv := range h.Headers {
			req.Headers = append(req.Headers, keyValue{uid: rowUID("header", i, kv.Name), kv: kv}.model())
		}
		for i, p := range h.Params {
			typ := model.ParamQuery
			if p.Type == string(model.ParamPath) {
				typ = model.ParamPath
			}
			req.Params = append(req.Params, model.Param{
				KeyValue: model.KeyValue{
					UID:     rowUID("param", i, p.Name),
					Name:    strPtr(p.Name),
					Value:   strPtr(p.Value),
					Enabled: !p.Disabled,
				},
				Type: typ,
			})
		}
		if h.Auth != nil {
			req.Auth = authFromDoc(h.Auth)
		}
		req.Body = bodyFromDoc(h.Body)
	}

	if doc.Runtime != nil {
		req.Vars = varBlocksFromRuntime(doc.Runtime)
		req.Script = scriptsFromRuntime(doc.Runtime)
	}

	if doc.Settings != nil {
		req.Settings = settingsFromDoc(doc.Settings)
	}

	if doc.Docs != "" {
		docs := doc.Docs
		req.Docs = &docs
	}

	for _, a := range doc.Assertions {
		value := a.Value
		if a.Operator != "" {
			value = a.Operator
			if a.Value != "" {
				value = a.Operator + " " + a.Value
			}
		}
		req.Assertions = append(req.Assertions, model.KeyValue{
			UID:     rowUID("assert", len(req.Assertions), a.Expression),
			Name:    strPtr(a.Expression),
			Value:   strPtr(value),
			Enabled: true,
		})
	}

	return req, nil
}

// keyValue adapts a kvDoc to a model row.
type keyValue struct {
	uid string
	kv  kvDoc
}

// model converts the adapter to a model row.
func (k keyValue) model() model.KeyValue {
	out := model.KeyValue{
		UID:     k.uid,
		Name:    strPtr(k.kv.Name),
		Value:   strPtr(k.kv.Value),
		Enabled: !k.kv.Disabled,
	}
	if k.kv.Description != "" {
		out.Description = strPtr(k.kv.Description)
	}
	return out
}

// bodyFromDoc maps the body section, handling both `mode:` and `type:` shapes.
func bodyFromDoc(b *bodyDoc) *model.Body {
	if b == nil {
		return &model.Body{Mode: model.BodyNone}
	}
	out := &model.Body{Mode: model.BodyNone}

	// `type:` wins for form-like bodies because that is how Bruno writes them.
	kind := b.Type
	if kind == "" {
		kind = b.Mode
	}

	switch kind {
	case "json":
		out.Mode = model.BodyJSON
		out.JSON = strPtr(firstNonEmpty(b.JSON, b.dataText()))
	case "text":
		out.Mode = model.BodyText
		out.Text = strPtr(firstNonEmpty(b.Text, b.dataText()))
	case "xml":
		out.Mode = model.BodyXML
		out.XML = strPtr(firstNonEmpty(b.XML, b.dataText()))
	case "sparql":
		out.Mode = model.BodySparql
		out.Sparql = strPtr(firstNonEmpty(b.Sparql, b.dataText()))
	case "graphql":
		out.Mode = model.BodyGraphQL
		gql := &model.GraphQLBody{Query: strPtr(firstNonEmpty(b.dataText(), deref(graphQLQuery(b))))}
		if b.GraphQL != nil && b.GraphQL.Variables != "" {
			gql.Variables = strPtr(b.GraphQL.Variables)
		}
		out.GraphQL = gql
	case "form-urlencoded":
		out.Mode = model.BodyFormURLEncoded
		for i, d := range b.dataRows() {
			out.FormURLEncoded = append(out.FormURLEncoded, keyValue{uid: rowUID("form", i, d.Name), kv: d}.model())
		}
	case "multipart-form":
		out.Mode = model.BodyMultipartForm
		for i, d := range b.dataRows() {
			partType := model.MultipartText
			if d.Type == "file" {
				partType = model.MultipartFile
			}
			part := model.MultipartEntry{
				UID:     rowUID("part", i, d.Name),
				Name:    strPtr(d.Name),
				Value:   strPtr(d.Value),
				Type:    partType,
				Enabled: !d.Disabled,
			}
			if d.ContentType != "" {
				part.ContentType = strPtr(d.ContentType)
			}
			out.MultipartForm = append(out.MultipartForm, part)
		}
	case "file":
		out.Mode = model.BodyFile
		for _, f := range b.File {
			out.File = append(out.File, model.FileEntry{
				UID:         rowUID("file", 0, f.FilePath),
				FilePath:    strPtr(f.FilePath),
				ContentType: strPtrOrNil(f.ContentType),
				Selected:    f.Selected,
			})
		}
	case "none", "":
		out.Mode = model.BodyNone
	default:
		// An unrecognised mode is preserved as none rather than guessed at.
		out.Mode = model.BodyNone
	}
	return out
}

// authFromDoc maps the auth section.
func authFromDoc(a *authDoc) *model.Auth {
	out := &model.Auth{Mode: model.AuthMode(a.Type)}
	if out.Mode == "" {
		out.Mode = model.AuthInherit
	}
	out.Username = a.Username
	out.Password = a.Password
	out.Token = a.Token
	out.Domain = a.Domain
	out.Key = a.Key
	out.Value = a.Value
	if a.Placement != "" {
		out.Placement = model.APIKeyPlacement(a.Placement)
	}
	out.AccessKeyID = a.AccessKeyID
	out.SecretAccessKey = a.SecretAccessKey
	out.SessionToken = a.SessionToken
	out.Service = a.Service
	out.Region = a.Region
	if a.GrantType != "" {
		out.GrantType = model.OAuth2GrantType(a.GrantType)
	}
	out.AuthorizationURL = a.AuthorizationURL
	out.AccessTokenURL = a.AccessTokenURL
	out.RefreshTokenURL = a.RefreshTokenURL
	out.CallbackURL = a.CallbackURL
	out.ClientID = a.ClientID
	out.ClientSecret = a.ClientSecret
	out.Scope = a.Scope
	out.State = a.State
	out.PKCE = a.PKCE
	out.CredentialsPlacement = a.CredentialsPlacement
	out.CredentialsID = a.CredentialsID
	out.TokenPlacement = a.TokenPlacement
	out.TokenHeaderPrefix = a.TokenHeaderPrefix
	out.TokenQueryKey = a.TokenQueryKey
	out.ConsumerKey = a.ConsumerKey
	out.ConsumerSecret = a.ConsumerSecret
	out.AccessToken = a.AccessToken
	out.AccessTokenSecret = a.AccessTokenSecret
	out.SignatureMethod = a.SignatureMethod
	return out
}

// scriptsFromRuntime extracts the pre-request and post-response scripts.
func scriptsFromRuntime(r *runtimeDoc) *model.Scripts {
	if r == nil {
		return nil
	}
	var sc model.Scripts
	found := false
	for _, s := range r.Scripts {
		code := s.Code
		switch s.Type {
		case "pre-request", "before-request":
			sc.Req = &code
			found = true
		case "post-response", "after-response", "tests", "test":
			// Bruno emits its `tests` block as a runtime script of type tests.
			if s.Type == "tests" || s.Type == "test" {
				continue
			}
			sc.Res = &code
			found = true
		}
	}
	if !found {
		return nil
	}
	return &sc
}

// testsFromRuntime extracts the tests block from the runtime scripts.
func testsFromRuntime(r *runtimeDoc) *string {
	if r == nil {
		return nil
	}
	for _, s := range r.Scripts {
		if s.Type == "tests" || s.Type == "test" {
			code := s.Code
			return &code
		}
	}
	return nil
}

// varBlocksFromRuntime maps runtime variables onto the pre-request block.
//
// OpenCollection has no pre/post split for runtime variables: they are all
// pre-request, so that is where they go.
func varBlocksFromRuntime(r *runtimeDoc) *model.VarBlocks {
	if r == nil || len(r.Variables) == 0 {
		return nil
	}
	vb := &model.VarBlocks{}
	for i, v := range r.Variables {
		out := model.Variable{
			UID:      rowUID("var", i, v.Name),
			Name:     strPtr(v.Name),
			Enabled:  !v.Disabled,
			DataType: "string",
		}
		if v.Value != nil {
			out.Value = v.Value.String()
			if v.Value.Type != "" {
				out.DataType = v.Value.Type
			}
		}
		if v.Description != "" {
			out.Description = strPtr(v.Description)
		}
		vb.PreRequest = append(vb.PreRequest, out)
	}
	return vb
}

// settingsFromDoc maps the settings section.
func settingsFromDoc(s *settingsDoc) model.ItemSettings {
	out := model.ItemSettings{
		EncodeURL:                  s.EncodeURL,
		FollowRedirects:            s.FollowRedirects,
		MaxRedirects:               s.MaxRedirects,
		ForwardAuthorizationHeader: s.ForwardAuthorizationHeader,
	}
	if s.Timeout != nil {
		t := model.TimeoutMillis(*s.Timeout)
		out.Timeout = &t
	}
	if s.OmitHeaders != nil && *s.OmitHeaders != "" {
		out.OmitHeaders = splitCSV(*s.OmitHeaders)
	}
	return out
}

// splitCSV splits a comma-separated list, trimming each item.
func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			item := trimSpace(s[start:i])
			if item != "" {
				out = append(out, item)
			}
			start = i + 1
		}
	}
	return out
}

// trimSpace trims ASCII whitespace.
func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// graphQLQuery returns the nested graphql block's query, if any.
func graphQLQuery(b *bodyDoc) *string {
	if b.GraphQL == nil || b.GraphQL.Query == "" {
		return nil
	}
	return &b.GraphQL.Query
}

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// strPtrOrNil returns nil for an empty string.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
