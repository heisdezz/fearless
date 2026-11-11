package postman

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"fearless/internal/core/model"
)

func newUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ImportResult contains the parsed collection and any dropped/warning notes.
type ImportResult struct {
	Collection *model.Collection
	Warnings   []string
}

// Import converts a Postman v2.0 or v2.1 collection JSON into a fearless model.Collection.
func Import(data []byte) (*ImportResult, error) {
	var pColl CollectionJSON
	if err := json.Unmarshal(data, &pColl); err != nil {
		return nil, fmt.Errorf("parsing Postman collection JSON: %w", err)
	}

	name := pColl.Info.Name
	if strings.TrimSpace(name) == "" {
		name = "Imported Postman Collection"
	}

	var warnings []string

	// Collection Config
	cfg := model.CollectionConfig{
		Name:    name,
		Type:    "collection",
		Version: "1",
		Format:  model.FormatBru,
	}

	if desc := pColl.Info.Description; desc != "" {
		cfg.Docs = &desc
	}

	// Collection-level variables
	for _, v := range pColl.Variable {
		name := v.Key
		val := v.Value
		cfg.Vars = append(cfg.Vars, model.Variable{
			UID:     newUID(),
			Name:    &name,
			Value:   val,
			Enabled: !v.Disabled,
		})
	}

	// Collection-level auth
	if pColl.Auth != nil {
		auth, w := convertAuth(pColl.Auth)
		cfg.Auth = auth
		warnings = append(warnings, w...)
	}

	// Collection-level scripts
	cfg.Script = convertEvents(pColl.Event)

	// Items
	seqCounter := 1
	items, itemWarnings := convertItems(pColl.Item, &seqCounter)
	warnings = append(warnings, itemWarnings...)

	coll := &model.Collection{
		UID:    newUID(),
		Name:   name,
		Config: cfg,
		Items:  items,
	}

	return &ImportResult{
		Collection: coll,
		Warnings:   warnings,
	}, nil
}

// ImportEnvironment converts Postman environment JSON into a fearless model.Environment.
func ImportEnvironment(data []byte) (*model.Environment, error) {
	var envJSON EnvironmentJSON
	if err := json.Unmarshal(data, &envJSON); err != nil {
		return nil, fmt.Errorf("parsing Postman environment JSON: %w", err)
	}

	name := envJSON.Name
	if strings.TrimSpace(name) == "" {
		name = "Postman Environment"
	}

	env := &model.Environment{
		UID:    newUID(),
		Name:   name,
		Format: model.FormatBru,
	}

	for _, v := range envJSON.Values {
		k := v.Key
		val := v.Value
		env.Variables = append(env.Variables, model.Variable{
			UID:     newUID(),
			Name:    &k,
			Value:   val,
			Enabled: v.Enabled,
		})
		if strings.EqualFold(v.Type, "secret") {
			env.Secrets = append(env.Secrets, k)
		}
	}

	return env, nil
}

func convertItems(items []ItemJSON, seqCounter *int) ([]*model.Item, []string) {
	var result []*model.Item
	var warnings []string

	for _, it := range items {
		seq := *seqCounter
		*seqCounter++

		name := it.Name
		if name == "" {
			name = "Untitled"
		}

		desc := parseDescription(it.Description)

		if it.Request == nil && len(it.Item) > 0 {
			// Folder
			folder := &model.Item{
				UID:  newUID(),
				Type: model.ItemFolder,
				Name: name,
				Seq:  &seq,
			}
			if desc != "" {
				folder.Request = &model.Request{Docs: &desc}
			}
			if it.Auth != nil {
				a, w := convertAuth(it.Auth)
				folder.Auth = a
				warnings = append(warnings, w...)
			}
			folder.Script = convertEvents(it.Event)

			children, childWarnings := convertItems(it.Item, seqCounter)
			folder.Children = children
			warnings = append(warnings, childWarnings...)
			result = append(result, folder)
		} else if it.Request != nil {
			// Request
			item, w := convertRequestItem(it, seq, name, desc)
			warnings = append(warnings, w...)
			result = append(result, item)
		}
	}

	return result, warnings
}

func convertRequestItem(it ItemJSON, seq int, name string, desc string) (*model.Item, []string) {
	var warnings []string
	pReq := it.Request

	method := strings.ToUpper(pReq.Method)
	if method == "" {
		method = "GET"
	}

	itemType := model.ItemHTTPRequest
	if pReq.Body != nil && pReq.Body.Mode == "graphql" {
		itemType = model.ItemGraphQLRequest
	}

	reqURL, params := parsePostmanURL(pReq.URL)

	// Headers
	var headers []model.KeyValue
	for _, h := range pReq.Header {
		k := h.Key
		v := h.Value
		headers = append(headers, model.KeyValue{
			UID:     newUID(),
			Name:    &k,
			Value:   &v,
			Enabled: !h.Disabled,
		})
	}

	// Auth
	var auth *model.Auth
	if pReq.Auth != nil {
		a, w := convertAuth(pReq.Auth)
		auth = a
		warnings = append(warnings, w...)
	}

	// Body
	body, bodyWarnings := convertBody(pReq.Body)
	warnings = append(warnings, bodyWarnings...)

	// Scripts & Tests
	scripts, tests := parseRequestEvents(it.Event)

	req := &model.Request{
		URL:     reqURL,
		Method:  method,
		Headers: headers,
		Params:  params,
		Auth:    auth,
		Body:    body,
		Script:  scripts,
		Tests:   tests,
	}

	if desc != "" {
		req.Docs = &desc
	}

	item := &model.Item{
		UID:     newUID(),
		Type:    itemType,
		Name:    name,
		Seq:     &seq,
		Request: req,
	}

	return item, warnings
}

func parsePostmanURL(rawURL any) (string, []model.Param) {
	if rawURL == nil {
		return "", nil
	}

	var params []model.Param

	switch v := rawURL.(type) {
	case string:
		uStr := strings.TrimSpace(v)
		baseURL, queryParams := extractURLAndParams(uStr)
		return baseURL, queryParams
	case map[string]any:
		// Object URL
		var raw string
		if r, ok := v["raw"].(string); ok {
			raw = strings.TrimSpace(r)
		}

		if queries, ok := v["query"].([]any); ok {
			for _, q := range queries {
				if qm, ok := q.(map[string]any); ok {
					k, _ := qm["key"].(string)
					val, _ := qm["value"].(string)
					disabled, _ := qm["disabled"].(bool)
					params = append(params, model.Param{
						Type: model.ParamQuery,
						KeyValue: model.KeyValue{
							UID:     newUID(),
							Name:    &k,
							Value:   &val,
							Enabled: !disabled,
						},
					})
				}
			}
		}

		if vars, ok := v["variable"].([]any); ok {
			for _, vr := range vars {
				if vm, ok := vr.(map[string]any); ok {
					k, _ := vm["key"].(string)
					val, _ := vm["value"].(string)
					disabled, _ := vm["disabled"].(bool)
					params = append(params, model.Param{
						Type: model.ParamPath,
						KeyValue: model.KeyValue{
							UID:     newUID(),
							Name:    &k,
							Value:   &val,
							Enabled: !disabled,
						},
					})
				}
			}
		}

		if raw != "" {
			baseURL, inlineParams := extractURLAndParams(raw)
			if len(params) == 0 {
				return baseURL, inlineParams
			}
			return baseURL, params
		}
	}

	return "", params
}

func extractURLAndParams(rawURL string) (string, []model.Param) {
	rawURL = strings.TrimSpace(rawURL)
	var queryParams []model.Param

	qIdx := strings.Index(rawURL, "?")
	if qIdx == -1 {
		return rawURL, nil
	}

	baseURL := rawURL[:qIdx]
	queryString := rawURL[qIdx+1:]

	if fIdx := strings.Index(queryString, "#"); fIdx != -1 {
		queryString = queryString[:fIdx]
	}

	pairs := strings.Split(queryString, "&")
	for _, p := range pairs {
		if p == "" {
			continue
		}
		var k, v string
		if strings.Contains(p, "=") {
			parts := strings.SplitN(p, "=", 2)
			k, _ = url.QueryUnescape(parts[0])
			v, _ = url.QueryUnescape(parts[1])
		} else {
			k, _ = url.QueryUnescape(p)
			v = ""
		}
		queryParams = append(queryParams, model.Param{
			Type: model.ParamQuery,
			KeyValue: model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: true,
			},
		})
	}

	return baseURL, queryParams
}

func convertBody(b *BodyJSON) (*model.Body, []string) {
	if b == nil {
		return nil, nil
	}

	var warnings []string
	switch strings.ToLower(b.Mode) {
	case "raw":
		lang := ""
		if b.Options != nil && b.Options.Raw != nil {
			lang = strings.ToLower(b.Options.Raw.Language)
		}
		rawText := b.Raw
		if lang == "json" || (lang == "" && (strings.HasPrefix(strings.TrimSpace(rawText), "{") || strings.HasPrefix(strings.TrimSpace(rawText), "["))) {
			return &model.Body{
				Mode: model.BodyJSON,
				JSON: &rawText,
			}, nil
		} else if lang == "xml" || strings.HasPrefix(strings.TrimSpace(rawText), "<") {
			return &model.Body{
				Mode: model.BodyXML,
				XML:  &rawText,
			}, nil
		}
		return &model.Body{
			Mode: model.BodyText,
			Text: &rawText,
		}, nil

	case "urlencoded":
		var kvs []model.KeyValue
		for _, item := range b.URLEncoded {
			k := item.Key
			v := item.Value
			kvs = append(kvs, model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: !item.Disabled,
			})
		}
		return &model.Body{
			Mode:           model.BodyFormURLEncoded,
			FormURLEncoded: kvs,
		}, nil

	case "formdata":
		var entries []model.MultipartEntry
		for _, item := range b.FormData {
			name := item.Key
			if strings.EqualFold(item.Type, "file") {
				src := ""
				switch s := item.Src.(type) {
				case string:
					src = s
				case []any:
					if len(s) > 0 {
						if str, ok := s[0].(string); ok {
							src = str
						}
					}
				}
				val := "@file(" + src + ")"
				var ct *string
				if item.ContentType != "" {
					ct = &item.ContentType
				}
				entries = append(entries, model.MultipartEntry{
					UID:         newUID(),
					Type:        model.MultipartFile,
					Name:        &name,
					Value:       &val,
					ContentType: ct,
					Enabled:     !item.Disabled,
				})
			} else {
				val := item.Value
				entries = append(entries, model.MultipartEntry{
					UID:     newUID(),
					Type:    model.MultipartText,
					Name:    &name,
					Value:   &val,
					Enabled: !item.Disabled,
				})
			}
		}
		return &model.Body{
			Mode:          model.BodyMultipartForm,
			MultipartForm: entries,
		}, nil

	case "file":
		if b.File != nil {
			src := b.File.Src
			return &model.Body{
				Mode: model.BodyFile,
				File: []model.FileEntry{
					{
						UID:      newUID(),
						FilePath: &src,
						Selected: true,
					},
				},
			}, nil
		}

	case "graphql":
		if b.GraphQL != nil {
			q := b.GraphQL.Query
			v := b.GraphQL.Variables
			return &model.Body{
				Mode: model.BodyGraphQL,
				GraphQL: &model.GraphQLBody{
					Query:     &q,
					Variables: &v,
				},
			}, nil
		}

	default:
		if b.Mode != "" {
			warnings = append(warnings, fmt.Sprintf("Unsupported Postman body mode '%s' dropped", b.Mode))
		}
	}

	return nil, warnings
}

func convertAuth(a *AuthJSON) (*model.Auth, []string) {
	if a == nil {
		return nil, nil
	}

	var warnings []string
	getParam := func(params []AuthParamJSON, key string) string {
		for _, p := range params {
			if strings.EqualFold(p.Key, key) {
				return fmt.Sprintf("%v", p.Value)
			}
		}
		return ""
	}

	switch strings.ToLower(a.Type) {
	case "basic":
		u := getParam(a.Basic, "username")
		p := getParam(a.Basic, "password")
		return &model.Auth{
			Mode:     model.AuthBasic,
			Username: &u,
			Password: &p,
		}, nil

	case "bearer":
		t := getParam(a.Bearer, "token")
		return &model.Auth{
			Mode:  model.AuthBearer,
			Token: &t,
		}, nil

	case "apikey":
		k := getParam(a.APIKey, "key")
		v := getParam(a.APIKey, "value")
		in := getParam(a.APIKey, "in")
		placement := model.APIKeyHeader
		if strings.EqualFold(in, "query") {
			placement = model.APIKeyQueryParams
		}
		return &model.Auth{
			Mode:      model.AuthAPIKey,
			Key:       &k,
			Value:     &v,
			Placement: placement,
		}, nil

	case "digest":
		u := getParam(a.Digest, "username")
		p := getParam(a.Digest, "password")
		return &model.Auth{
			Mode:     model.AuthDigest,
			Username: &u,
			Password: &p,
		}, nil

	case "ntlm":
		u := getParam(a.NTLM, "username")
		p := getParam(a.NTLM, "password")
		d := getParam(a.NTLM, "domain")
		return &model.Auth{
			Mode:     model.AuthNTLM,
			Username: &u,
			Password: &p,
			Domain:   &d,
		}, nil

	case "oauth2":
		token := getParam(a.OAuth2, "accessToken")
		if token != "" {
			return &model.Auth{
				Mode:  model.AuthBearer,
				Token: &token,
			}, nil
		}
		return &model.Auth{
			Mode: model.AuthOAuth2,
		}, nil

	case "awsv4":
		ak := getParam(a.AWSV4, "accessKey")
		sk := getParam(a.AWSV4, "secretKey")
		reg := getParam(a.AWSV4, "region")
		svc := getParam(a.AWSV4, "service")
		return &model.Auth{
			Mode:            model.AuthAWSV4,
			AccessKeyID:     &ak,
			SecretAccessKey: &sk,
			Region:          &reg,
			Service:         &svc,
		}, nil

	case "noauth":
		return &model.Auth{
			Mode: model.AuthNone,
		}, nil

	default:
		warnings = append(warnings, fmt.Sprintf("Unsupported Postman auth type '%s' converted to AuthNone", a.Type))
		return &model.Auth{
			Mode: model.AuthNone,
		}, warnings
	}
}

func convertEvents(events []EventJSON) *model.Scripts {
	var reqScript, resScript *string

	for _, ev := range events {
		code := extractScriptCode(ev.Script)
		if code == "" {
			continue
		}
		transpiled := transpileScript(code)

		if strings.EqualFold(ev.Listen, "prerequest") {
			reqScript = &transpiled
		} else if strings.EqualFold(ev.Listen, "test") {
			resScript = &transpiled
		}
	}

	if reqScript == nil && resScript == nil {
		return nil
	}
	return &model.Scripts{
		Req: reqScript,
		Res: resScript,
	}
}

func parseRequestEvents(events []EventJSON) (*model.Scripts, *string) {
	var reqScript *string
	var tests *string

	for _, ev := range events {
		code := extractScriptCode(ev.Script)
		if code == "" {
			continue
		}
		transpiled := transpileScript(code)

		if strings.EqualFold(ev.Listen, "prerequest") {
			reqScript = &transpiled
		} else if strings.EqualFold(ev.Listen, "test") {
			tests = &transpiled
		}
	}

	var scripts *model.Scripts
	if reqScript != nil {
		scripts = &model.Scripts{
			Req: reqScript,
		}
	}

	return scripts, tests
}

func extractScriptCode(s ScriptJSON) string {
	switch v := s.Exec.(type) {
	case string:
		return v
	case []any:
		var lines []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				lines = append(lines, str)
			}
		}
		return strings.Join(lines, "\n")
	case []string:
		return strings.Join(v, "\n")
	}
	return ""
}

func transpileScript(code string) string {
	code = strings.ReplaceAll(code, "pm.environment.set(", "bru.setEnvVar(")
	code = strings.ReplaceAll(code, "pm.environment.get(", "bru.getEnvVar(")
	code = strings.ReplaceAll(code, "pm.variables.set(", "bru.setVar(")
	code = strings.ReplaceAll(code, "pm.variables.get(", "bru.getVar(")
	code = strings.ReplaceAll(code, "pm.collectionVariables.set(", "bru.setVar(")
	code = strings.ReplaceAll(code, "pm.collectionVariables.get(", "bru.getVar(")
	code = strings.ReplaceAll(code, "pm.test(", "test(")
	code = strings.ReplaceAll(code, "pm.expect(", "expect(")
	code = strings.ReplaceAll(code, "pm.response.json()", "res.getBody()")
	return code
}
