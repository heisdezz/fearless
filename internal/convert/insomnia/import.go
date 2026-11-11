package insomnia

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"

	"fearless/internal/core/model"
)

func newUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ImportResult contains the imported collection and warnings.
type ImportResult struct {
	Collection *model.Collection
	Warnings   []string
}

// Import parses an Insomnia export (JSON or YAML) into a fearless model.Collection.
func Import(data []byte) (*ImportResult, error) {
	var exp ExportJSON
	if err := json.Unmarshal(data, &exp); err != nil {
		// Try YAML
		if yErr := yaml.Unmarshal(data, &exp); yErr != nil {
			return nil, fmt.Errorf("parsing Insomnia export as JSON (%v) or YAML (%w)", err, yErr)
		}
	}

	if len(exp.Resources) == 0 {
		return nil, fmt.Errorf("no resources found in Insomnia export")
	}

	var warnings []string

	// Index resources by ID and by ParentID
	resByID := make(map[string]ResourceJSON)
	childrenOf := make(map[string][]ResourceJSON)

	var workspace *ResourceJSON
	var baseEnv *ResourceJSON
	var subEnvs []ResourceJSON

	for _, r := range exp.Resources {
		resByID[r.ID] = r
		childrenOf[r.ParentID] = append(childrenOf[r.ParentID], r)

		if r.Type == "workspace" && workspace == nil {
			cp := r
			workspace = &cp
		}
	}

	collName := "Imported Insomnia Collection"
	collDesc := ""
	workspaceID := ""
	if workspace != nil {
		if workspace.Name != "" {
			collName = workspace.Name
		}
		collDesc = workspace.Description
		workspaceID = workspace.ID
	}

	// Find base environment and sub-environments
	for _, r := range exp.Resources {
		if r.Type == "environment" {
			if r.ParentID == workspaceID || r.ParentID == "" {
				cp := r
				baseEnv = &cp
			} else {
				subEnvs = append(subEnvs, r)
			}
		}
	}

	// Collection Config
	cfg := model.CollectionConfig{
		Name:    collName,
		Type:    "collection",
		Version: "1",
		Format:  model.FormatBru,
	}
	if collDesc != "" {
		cfg.Docs = &collDesc
	}

	// Extract base environment variables into collection vars
	if baseEnv != nil && len(baseEnv.Data) > 0 {
		for k, v := range baseEnv.Data {
			key := k
			val := v
			cfg.Vars = append(cfg.Vars, model.Variable{
				UID:     newUID(),
				Name:    &key,
				Value:   val,
				Enabled: true,
			})
		}
	}

	// Convert sub-environments into model.Environment
	var environments []*model.Environment
	for _, se := range subEnvs {
		envName := se.Name
		if envName == "" {
			envName = "Environment"
		}
		env := &model.Environment{
			UID:    newUID(),
			Name:   envName,
			Color:  se.Color,
			Format: model.FormatBru,
		}
		for k, v := range se.Data {
			key := k
			val := v
			env.Variables = append(env.Variables, model.Variable{
				UID:     newUID(),
				Name:    &key,
				Value:   val,
				Enabled: true,
			})
			if se.IsPrivate {
				env.Secrets = append(env.Secrets, key)
			}
		}
		environments = append(environments, env)
	}

	// Recursively build items
	seqCounter := 1
	var rootItems []ResourceJSON
	if workspaceID != "" {
		rootItems = childrenOf[workspaceID]
	} else {
		rootItems = childrenOf[""]
	}

	items, itemWarnings := buildItemTree(rootItems, childrenOf, &seqCounter)
	warnings = append(warnings, itemWarnings...)

	coll := &model.Collection{
		UID:          newUID(),
		Name:         collName,
		Config:       cfg,
		Items:        items,
		Environments: environments,
	}

	return &ImportResult{
		Collection: coll,
		Warnings:   warnings,
	}, nil
}

func buildItemTree(resources []ResourceJSON, childrenOf map[string][]ResourceJSON, seqCounter *int) ([]*model.Item, []string) {
	var items []*model.Item
	var warnings []string

	for _, r := range resources {
		switch r.Type {
		case "request_group":
			seq := *seqCounter
			*seqCounter++
			folderName := r.Name
			if folderName == "" {
				folderName = "Folder"
			}
			folder := &model.Item{
				UID:  newUID(),
				Type: model.ItemFolder,
				Name: folderName,
				Seq:  &seq,
			}
			if r.Description != "" {
				folder.Request = &model.Request{Docs: &r.Description}
			}

			// Folder vars
			if len(r.Environment) > 0 {
				for k, v := range r.Environment {
					key := k
					val := v
					folder.Vars = append(folder.Vars, model.Variable{
						UID:     newUID(),
						Name:    &key,
						Value:   val,
						Enabled: true,
					})
				}
			}

			children, childWarnings := buildItemTree(childrenOf[r.ID], childrenOf, seqCounter)
			folder.Children = children
			warnings = append(warnings, childWarnings...)
			items = append(items, folder)

		case "request":
			seq := *seqCounter
			*seqCounter++
			item, w := convertRequest(r, seq)
			warnings = append(warnings, w...)
			items = append(items, item)

		case "cookie_jar":
			warnings = append(warnings, "Insomnia cookie_jar resource dropped (cookies handled in session)")
		}
	}

	return items, warnings
}

func convertRequest(r ResourceJSON, seq int) (*model.Item, []string) {
	var warnings []string
	name := r.Name
	if name == "" {
		name = "Untitled Request"
	}

	method := strings.ToUpper(r.Method)
	if method == "" {
		method = "GET"
	}

	itemType := model.ItemHTTPRequest
	if r.Body != nil && strings.Contains(strings.ToLower(r.Body.MimeType), "graphql") {
		itemType = model.ItemGraphQLRequest
	}

	baseURL, queryParams := parseInsomniaURL(r.URL, r.Parameters)

	// Headers
	var headers []model.KeyValue
	for _, h := range r.Headers {
		k := h.Name
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
	if r.Authentication != nil && !r.Authentication.Disabled {
		a, w := convertAuth(r.Authentication)
		auth = a
		warnings = append(warnings, w...)
	}

	// Body
	body, bodyWarnings := convertBody(r.Body)
	warnings = append(warnings, bodyWarnings...)

	// Settings
	var followRedirects *bool
	if r.SettingFollowRedirects == "on" {
		t := true
		followRedirects = &t
	} else if r.SettingFollowRedirects == "off" {
		f := false
		followRedirects = &f
	}

	req := &model.Request{
		URL:     baseURL,
		Method:  method,
		Headers: headers,
		Params:  queryParams,
		Auth:    auth,
		Body:    body,
		Settings: model.ItemSettings{
			FollowRedirects: followRedirects,
		},
	}
	if r.Description != "" {
		req.Docs = &r.Description
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

func parseInsomniaURL(rawURL string, extraParams []ParameterJSON) (string, []model.Param) {
	rawURL = strings.TrimSpace(rawURL)
	var queryParams []model.Param

	baseURL := rawURL
	if qIdx := strings.Index(rawURL, "?"); qIdx != -1 {
		baseURL = rawURL[:qIdx]
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
	}

	// Append structured parameters
	for _, p := range extraParams {
		k := p.Name
		v := p.Value
		queryParams = append(queryParams, model.Param{
			Type: model.ParamQuery,
			KeyValue: model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: !p.Disabled,
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
	mime := strings.ToLower(b.MimeType)

	if strings.Contains(mime, "json") {
		text := b.Text
		return &model.Body{
			Mode: model.BodyJSON,
			JSON: &text,
		}, nil
	}

	if strings.Contains(mime, "graphql") {
		var gql struct {
			Query     string `json:"query"`
			Variables any    `json:"variables"`
		}
		q := b.Text
		var vars string
		if json.Unmarshal([]byte(b.Text), &gql) == nil {
			q = gql.Query
			if gql.Variables != nil {
				if vb, err := json.Marshal(gql.Variables); err == nil {
					vars = string(vb)
				}
			}
		}
		return &model.Body{
			Mode: model.BodyGraphQL,
			GraphQL: &model.GraphQLBody{
				Query:     &q,
				Variables: &vars,
			},
		}, nil
	}

	if strings.Contains(mime, "x-www-form-urlencoded") {
		var kvs []model.KeyValue
		for _, p := range b.Params {
			k := p.Name
			v := p.Value
			kvs = append(kvs, model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: !p.Disabled,
			})
		}
		return &model.Body{
			Mode:           model.BodyFormURLEncoded,
			FormURLEncoded: kvs,
		}, nil
	}

	if strings.Contains(mime, "multipart/form-data") {
		var entries []model.MultipartEntry
		for _, p := range b.Params {
			name := p.Name
			if p.Type == "file" || p.FileName != "" {
				fileVal := "@file(" + p.FileName + ")"
				entries = append(entries, model.MultipartEntry{
					UID:     newUID(),
					Type:    model.MultipartFile,
					Name:    &name,
					Value:   &fileVal,
					Enabled: !p.Disabled,
				})
			} else {
				val := p.Value
				entries = append(entries, model.MultipartEntry{
					UID:     newUID(),
					Type:    model.MultipartText,
					Name:    &name,
					Value:   &val,
					Enabled: !p.Disabled,
				})
			}
		}
		return &model.Body{
			Mode:          model.BodyMultipartForm,
			MultipartForm: entries,
		}, nil
	}

	if strings.Contains(mime, "xml") {
		text := b.Text
		return &model.Body{
			Mode: model.BodyXML,
			XML:  &text,
		}, nil
	}

	if b.FileName != "" {
		fPath := b.FileName
		return &model.Body{
			Mode: model.BodyFile,
			File: []model.FileEntry{
				{
					UID:      newUID(),
					FilePath: &fPath,
					Selected: true,
				},
			},
		}, nil
	}

	if b.Text != "" {
		text := b.Text
		return &model.Body{
			Mode: model.BodyText,
			Text: &text,
		}, nil
	}

	return nil, warnings
}

func convertAuth(a *AuthJSON) (*model.Auth, []string) {
	if a == nil || a.Disabled {
		return nil, nil
	}

	var warnings []string
	switch strings.ToLower(a.Type) {
	case "basic":
		u := a.Username
		p := a.Password
		return &model.Auth{
			Mode:     model.AuthBasic,
			Username: &u,
			Password: &p,
		}, nil

	case "bearer":
		t := a.Token
		return &model.Auth{
			Mode:  model.AuthBearer,
			Token: &t,
		}, nil

	case "apikey":
		k := a.Key
		v := a.Value
		placement := model.APIKeyHeader
		if strings.EqualFold(a.AddTo, "queryParams") {
			placement = model.APIKeyQueryParams
		}
		return &model.Auth{
			Mode:      model.AuthAPIKey,
			Key:       &k,
			Value:     &v,
			Placement: placement,
		}, nil

	case "digest":
		u := a.Username
		p := a.Password
		return &model.Auth{
			Mode:     model.AuthDigest,
			Username: &u,
			Password: &p,
		}, nil

	case "ntlm":
		u := a.Username
		p := a.Password
		d := a.Domain
		return &model.Auth{
			Mode:     model.AuthNTLM,
			Username: &u,
			Password: &p,
			Domain:   &d,
		}, nil

	case "iam", "awsv4":
		ak := a.AccessKeyID
		sk := a.SecretAccessKey
		tok := a.SessionToken
		reg := a.Region
		svc := a.Service
		return &model.Auth{
			Mode:            model.AuthAWSV4,
			AccessKeyID:     &ak,
			SecretAccessKey: &sk,
			SessionToken:    &tok,
			Region:          &reg,
			Service:         &svc,
		}, nil

	case "oauth2":
		if a.AccessToken != "" {
			tok := a.AccessToken
			return &model.Auth{
				Mode:  model.AuthBearer,
				Token: &tok,
			}, nil
		}
		return &model.Auth{
			Mode: model.AuthOAuth2,
		}, nil

	default:
		warnings = append(warnings, fmt.Sprintf("Unsupported Insomnia auth type '%s' dropped", a.Type))
		return nil, warnings
	}
}
