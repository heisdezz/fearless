package yamlcodec

import (
	"fmt"
	"strconv"

	"fearless/internal/core/model"

	"gopkg.in/yaml.v3"
)

// requestToDoc maps a model request onto the YAML document shape.
func requestToDoc(req *model.Request) *requestDoc {
	doc := &requestDoc{
		Info: infoDoc{
			Name: req.Name,
			Type: "http",
			Seq:  req.Seq,
			Tags: req.Tags,
		},
	}
	if doc.Info.Type == "" {
		doc.Info.Type = "http"
	}

	h := &httpDoc{Method: req.Method, URL: req.URL}
	if h.Method == "" {
		h.Method = "GET"
	}

	for _, kv := range req.Headers {
		h.Headers = append(h.Headers, kvDoc{
			Name:        kv.NameOrEmpty(),
			Value:       kv.ValueOrEmpty(),
			Description: deref(kv.Description),
			Disabled:    !kv.Enabled,
		})
	}
	for _, p := range req.Params {
		h.Params = append(h.Params, paramDoc{
			Name:     deref(p.Name),
			Value:    p.ValueOrEmpty(),
			Type:     string(p.Type),
			Disabled: !p.Enabled,
		})
	}
	if req.Auth != nil && req.Auth.Mode != "" {
		h.Auth = authToDoc(req.Auth)
	}
	h.Body = bodyToDoc(req.Body)
	doc.HTTP = h

	// Runtime: variables, then scripts, then tests.
	rt := &runtimeDoc{}
	if req.Vars != nil {
		for _, v := range req.Vars.PreRequest {
			rt.Variables = append(rt.Variables, variableToDoc(v))
		}
		for _, v := range req.Vars.PostResponse {
			rt.Variables = append(rt.Variables, variableToDoc(v))
		}
	}
	if req.Script != nil {
		if req.Script.Req != nil {
			rt.Scripts = append(rt.Scripts, scriptDoc{Type: "pre-request", Code: *req.Script.Req})
		}
		if req.Script.Res != nil {
			rt.Scripts = append(rt.Scripts, scriptDoc{Type: "post-response", Code: *req.Script.Res})
		}
	}
	if req.Tests != nil && *req.Tests != "" {
		rt.Scripts = append(rt.Scripts, scriptDoc{Type: "tests", Code: *req.Tests})
	}
	if len(rt.Variables) > 0 || len(rt.Scripts) > 0 {
		doc.Runtime = rt
	}

	doc.Settings = settingsToDoc(req.Settings)
	doc.Docs = deref(req.Docs)

	for _, a := range req.Assertions {
		doc.Assertions = append(doc.Assertions, assertDoc{
			Expression: a.NameOrEmpty(),
			Value:      a.ValueOrEmpty(),
		})
	}
	return doc
}

// variableToDoc maps a model variable onto the YAML shape.
func variableToDoc(v model.Variable) runtimeVarDoc {
	out := runtimeVarDoc{
		Name:     v.NameOrEmpty(),
		Disabled: !v.Enabled,
	}
	if v.Description != nil {
		out.Description = *v.Description
	}
	text := valueString(v.Value)
	switch v.DataType {
	case "", "string":
		out.Value = &yamlValue{Scalar: &text}
	default:
		out.Value = &yamlValue{Type: v.DataType, Data: text}
	}
	return out
}

// valueString renders a variable value as text.
func valueString(v any) string {
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

// authToDoc maps the auth union onto the YAML shape.
func authToDoc(a *model.Auth) *authDoc {
	doc := &authDoc{Type: string(a.Mode)}
	doc.Username = a.Username
	doc.Password = a.Password
	doc.Token = a.Token
	doc.Domain = a.Domain
	doc.Key = a.Key
	doc.Value = a.Value
	if a.Placement != "" {
		doc.Placement = string(a.Placement)
	}
	doc.AccessKeyID = a.AccessKeyID
	doc.SecretAccessKey = a.SecretAccessKey
	doc.SessionToken = a.SessionToken
	doc.Service = a.Service
	doc.Region = a.Region
	if a.GrantType != "" {
		doc.GrantType = string(a.GrantType)
	}
	doc.AuthorizationURL = a.AuthorizationURL
	doc.AccessTokenURL = a.AccessTokenURL
	doc.RefreshTokenURL = a.RefreshTokenURL
	doc.CallbackURL = a.CallbackURL
	doc.ClientID = a.ClientID
	doc.ClientSecret = a.ClientSecret
	doc.Scope = a.Scope
	doc.State = a.State
	doc.PKCE = a.PKCE
	doc.CredentialsPlacement = a.CredentialsPlacement
	doc.CredentialsID = a.CredentialsID
	doc.TokenPlacement = a.TokenPlacement
	doc.TokenHeaderPrefix = a.TokenHeaderPrefix
	doc.TokenQueryKey = a.TokenQueryKey
	doc.ConsumerKey = a.ConsumerKey
	doc.ConsumerSecret = a.ConsumerSecret
	doc.AccessToken = a.AccessToken
	doc.AccessTokenSecret = a.AccessTokenSecret
	doc.SignatureMethod = a.SignatureMethod
	return doc
}

// bodyToDoc maps the body union onto the YAML shape.
func bodyToDoc(b *model.Body) *bodyDoc {
	if b == nil || b.Mode == model.BodyNone || b.Mode == "" {
		return nil
	}
	switch b.Mode {
	case model.BodyJSON:
		return &bodyDoc{Mode: "json", JSON: deref(b.JSON)}
	case model.BodyText:
		return &bodyDoc{Mode: "text", Text: deref(b.Text)}
	case model.BodyXML:
		return &bodyDoc{Mode: "xml", XML: deref(b.XML)}
	case model.BodySparql:
		return &bodyDoc{Mode: "sparql", Sparql: deref(b.Sparql)}
	case model.BodyGraphQL:
		doc := &bodyDoc{Mode: "graphql", GraphQL: &graphqlDoc{}}
		if b.GraphQL != nil {
			doc.GraphQL.Query = deref(b.GraphQL.Query)
			doc.GraphQL.Variables = deref(b.GraphQL.Variables)
		}
		return doc
	case model.BodyFormURLEncoded:
		doc := &bodyDoc{Type: "form-urlencoded"}
		var rows []kvDoc
		for _, kv := range b.FormURLEncoded {
			rows = append(rows, kvDoc{
				Name:     kv.NameOrEmpty(),
				Value:    kv.ValueOrEmpty(),
				Disabled: !kv.Enabled,
			})
		}
		doc.Data = rowsNode(rows)
		return doc
	case model.BodyMultipartForm:
		doc := &bodyDoc{Type: "multipart-form"}
		var rows []kvDoc
		for _, p := range b.MultipartForm {
			rows = append(rows, kvDoc{
				Name:        deref(p.Name),
				Value:       deref(p.Value),
				Type:        string(p.Type),
				ContentType: deref(p.ContentType),
				Disabled:    !p.Enabled,
			})
		}
		doc.Data = rowsNode(rows)
		return doc
	case model.BodyFile:
		doc := &bodyDoc{Type: "file"}
		for _, f := range b.File {
			doc.File = append(doc.File, fileDoc{
				FilePath:    deref(f.FilePath),
				ContentType: deref(f.ContentType),
				Selected:    f.Selected,
			})
		}
		return doc
	}
	return nil
}

// settingsToDoc maps settings, omitting unset fields.
func settingsToDoc(s model.ItemSettings) *settingsDoc {
	doc := &settingsDoc{
		EncodeURL:                  s.EncodeURL,
		FollowRedirects:            s.FollowRedirects,
		MaxRedirects:               s.MaxRedirects,
		ForwardAuthorizationHeader: s.ForwardAuthorizationHeader,
	}
	if s.Timeout != nil && !s.Timeout.Inherit {
		ms := s.Timeout.Millis
		doc.Timeout = &ms
	}
	if len(s.OmitHeaders) > 0 {
		joined := ""
		for i, h := range s.OmitHeaders {
			if i > 0 {
				joined += ","
			}
			joined += h
		}
		doc.OmitHeaders = &joined
	}
	if doc.EncodeURL == nil && doc.FollowRedirects == nil && doc.MaxRedirects == nil &&
		doc.Timeout == nil && doc.ForwardAuthorizationHeader == nil && doc.OmitHeaders == nil {
		return nil
	}
	return doc
}

// deref returns *p or "" when p is nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// rowsNode builds a YAML sequence node from rows.
//
// The body struct stores Data as a *yaml.Node so it can carry either a scalar or
// a sequence, so the writer constructs the node explicitly.
func rowsNode(rows []kvDoc) yaml.Node {
	node := yaml.Node{Kind: yaml.SequenceNode}
	for _, r := range rows {
		row := &yaml.Node{Kind: yaml.MappingNode}
		appendScalar(row, "name", r.Name)
		if r.Value != "" {
			appendScalar(row, "value", r.Value)
		}
		if r.Type != "" {
			appendScalar(row, "type", r.Type)
		}
		if r.ContentType != "" {
			appendScalar(row, "contentType", r.ContentType)
		}
		if r.Disabled {
			appendScalar(row, "disabled", "true")
		}
		node.Content = append(node.Content, row)
	}
	return node
}

// appendScalar adds a key/value pair to a mapping node.
func appendScalar(m *yaml.Node, key, value string) {
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}
