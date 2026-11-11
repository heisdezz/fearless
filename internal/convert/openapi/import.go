package openapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"fearless/internal/core/model"
)

func newUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GroupByOption specifies how items are organized into folders.
type GroupByOption string

const (
	GroupByTags GroupByOption = "tags"
	GroupByPath GroupByOption = "path"
	GroupByNone GroupByOption = "none"
)

// ImportOptions configures OpenAPI import behavior.
type ImportOptions struct {
	GroupBy GroupByOption
}

// ImportResult contains the imported collection and any warnings.
type ImportResult struct {
	Collection *model.Collection
	Warnings   []string
}

// Import parses an OpenAPI 3.x or Swagger 2.0 document into a fearless model.Collection.
func Import(data []byte) (*ImportResult, error) {
	return ImportWithOptions(data, ImportOptions{GroupBy: GroupByTags})
}

// ImportWithOptions parses an OpenAPI document with custom grouping options.
func ImportWithOptions(data []byte, opts ImportOptions) (*ImportResult, error) {
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		if yErr := yaml.Unmarshal(data, &spec); yErr != nil {
			return nil, fmt.Errorf("parsing OpenAPI as JSON (%v) or YAML (%w)", err, yErr)
		}
	}

	if spec.OpenAPI == "" && spec.Swagger == "" && len(spec.Paths) == 0 {
		return nil, fmt.Errorf("invalid OpenAPI/Swagger document: missing version and paths")
	}

	var warnings []string

	title := spec.Info.Title
	if strings.TrimSpace(title) == "" {
		title = "Imported OpenAPI Collection"
	}

	cfg := model.CollectionConfig{
		Name:    title,
		Type:    "collection",
		Version: "1",
		Format:  model.FormatBru,
	}
	if spec.Info.Description != "" {
		desc := spec.Info.Description
		cfg.Docs = &desc
	}

	// Base URL resolution
	baseURL := resolveBaseURL(&spec)
	if baseURL != "" {
		k := "baseUrl"
		v := baseURL
		cfg.Vars = append(cfg.Vars, model.Variable{
			UID:     newUID(),
			Name:    &k,
			Value:   v,
			Enabled: true,
		})
	}

	// Security schemes index
	secSchemes := make(map[string]SecuritySchemeObject)
	if spec.Components != nil && spec.Components.SecuritySchemes != nil {
		for k, s := range spec.Components.SecuritySchemes {
			secSchemes[k] = s
		}
	}
	for k, s := range spec.SecurityDefinitions {
		secSchemes[k] = s
	}

	// Folders index
	foldersByTag := make(map[string]*model.Item)
	var rootItems []*model.Item
	seqCounter := 1

	// Sort paths for stable import order
	var pathKeys []string
	for p := range spec.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, pathStr := range pathKeys {
		pathItem := spec.Paths[pathStr]
		ops := pathItem.Operations()

		// Stable method order
		methodOrder := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
		for _, m := range methodOrder {
			op, ok := ops[m]
			if !ok {
				continue
			}

			item, itemWarnings := convertOperation(pathStr, m, op, pathItem.Parameters, secSchemes, &seqCounter)
			warnings = append(warnings, itemWarnings...)

			// Grouping
			if opts.GroupBy == GroupByTags && len(op.Tags) > 0 {
				tagName := op.Tags[0]
				folder, exists := foldersByTag[tagName]
				if !exists {
					seq := seqCounter
					seqCounter++
					folder = &model.Item{
						UID:  newUID(),
						Type: model.ItemFolder,
						Name: tagName,
						Seq:  &seq,
					}
					foldersByTag[tagName] = folder
					rootItems = append(rootItems, folder)
				}
				folder.Children = append(folder.Children, item)
			} else {
				rootItems = append(rootItems, item)
			}
		}
	}

	coll := &model.Collection{
		UID:    newUID(),
		Name:   title,
		Config: cfg,
		Items:  rootItems,
	}

	return &ImportResult{
		Collection: coll,
		Warnings:   warnings,
	}, nil
}

func convertOperation(
	pathStr string,
	method string,
	op *OperationObject,
	pathParams []ParameterObject,
	secSchemes map[string]SecuritySchemeObject,
	seqCounter *int,
) (*model.Item, []string) {
	var warnings []string
	seq := *seqCounter
	*seqCounter++

	// Determine request name
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", method, pathStr)
	}

	// Target URL: {{baseUrl}} + pathStr
	targetURL := "{{baseUrl}}" + pathStr

	// Merge parameters
	var allParams []ParameterObject
	allParams = append(allParams, pathParams...)
	allParams = append(allParams, op.Parameters...)

	var queryParams []model.Param
	var headers []model.KeyValue
	var formParams []model.KeyValue

	for _, p := range allParams {
		k := p.Name
		v := ""
		if p.Default != nil {
			v = fmt.Sprintf("%v", p.Default)
		} else if p.Example != nil {
			v = fmt.Sprintf("%v", p.Example)
		}

		switch strings.ToLower(p.In) {
		case "query":
			queryParams = append(queryParams, model.Param{
				Type: model.ParamQuery,
				KeyValue: model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &v,
					Enabled: p.Required || v != "",
				},
			})
		case "path":
			queryParams = append(queryParams, model.Param{
				Type: model.ParamPath,
				KeyValue: model.KeyValue{
					UID:     newUID(),
					Name:    &k,
					Value:   &v,
					Enabled: true,
				},
			})
		case "header":
			headers = append(headers, model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: p.Required || v != "",
			})
		case "formdata":
			formParams = append(formParams, model.KeyValue{
				UID:     newUID(),
				Name:    &k,
				Value:   &v,
				Enabled: true,
			})
		case "body":
			// Swagger 2.0 body parameter
		}
	}

	// Body resolution
	body, bodyWarnings := resolveBody(op, formParams)
	warnings = append(warnings, bodyWarnings...)

	// Security resolution
	auth, authWarnings := resolveSecurity(op, secSchemes)
	warnings = append(warnings, authWarnings...)

	req := &model.Request{
		URL:     targetURL,
		Method:  method,
		Headers: headers,
		Params:  queryParams,
		Auth:    auth,
		Body:    body,
	}

	if op.Description != "" {
		req.Docs = &op.Description
	}

	item := &model.Item{
		UID:     newUID(),
		Type:    model.ItemHTTPRequest,
		Name:    name,
		Seq:     &seq,
		Request: req,
	}

	return item, warnings
}

func resolveBody(op *OperationObject, formParams []model.KeyValue) (*model.Body, []string) {
	var warnings []string

	// Check OpenAPI 3.x requestBody
	if op.RequestBody != nil && op.RequestBody.Content != nil {
		for mime, media := range op.RequestBody.Content {
			lower := strings.ToLower(mime)
			if strings.Contains(lower, "json") {
				raw := "{}"
				if media.Example != nil {
					if b, err := json.MarshalIndent(media.Example, "", "  "); err == nil {
						raw = string(b)
					}
				}
				return &model.Body{
					Mode: model.BodyJSON,
					JSON: &raw,
				}, nil
			} else if strings.Contains(lower, "x-www-form-urlencoded") {
				return &model.Body{
					Mode:           model.BodyFormURLEncoded,
					FormURLEncoded: formParams,
				}, nil
			} else if strings.Contains(lower, "multipart/form-data") {
				var parts []model.MultipartEntry
				for _, fp := range formParams {
					k := fp.NameOrEmpty()
					v := fp.ValueOrEmpty()
					parts = append(parts, model.MultipartEntry{
						UID:     newUID(),
						Type:    model.MultipartText,
						Name:    &k,
						Value:   &v,
						Enabled: true,
					})
				}
				return &model.Body{
					Mode:          model.BodyMultipartForm,
					MultipartForm: parts,
				}, nil
			} else if strings.Contains(lower, "xml") {
				raw := "<xml/>"
				return &model.Body{
					Mode: model.BodyXML,
					XML:  &raw,
				}, nil
			}
		}
	}

	// Check Swagger 2.0 formData
	if len(formParams) > 0 {
		return &model.Body{
			Mode:           model.BodyFormURLEncoded,
			FormURLEncoded: formParams,
		}, nil
	}

	return nil, warnings
}

func resolveSecurity(op *OperationObject, secSchemes map[string]SecuritySchemeObject) (*model.Auth, []string) {
	var warnings []string
	if len(op.Security) == 0 {
		return nil, nil
	}

	for _, secReq := range op.Security {
		for schemeName := range secReq {
			scheme, ok := secSchemes[schemeName]
			if !ok {
				continue
			}

			switch strings.ToLower(scheme.Type) {
			case "http":
				if strings.EqualFold(scheme.Scheme, "bearer") {
					tok := ""
					return &model.Auth{
						Mode:  model.AuthBearer,
						Token: &tok,
					}, nil
				} else if strings.EqualFold(scheme.Scheme, "basic") {
					u, p := "", ""
					return &model.Auth{
						Mode:     model.AuthBasic,
						Username: &u,
						Password: &p,
					}, nil
				}
			case "basic":
				u, p := "", ""
				return &model.Auth{
					Mode:     model.AuthBasic,
					Username: &u,
					Password: &p,
				}, nil
			case "apikey":
				k := scheme.Name
				v := ""
				placement := model.APIKeyHeader
				if strings.EqualFold(scheme.In, "query") {
					placement = model.APIKeyQueryParams
				}
				return &model.Auth{
					Mode:      model.AuthAPIKey,
					Key:       &k,
					Value:     &v,
					Placement: placement,
				}, nil
			case "oauth2":
				return &model.Auth{
					Mode: model.AuthOAuth2,
				}, nil
			}
		}
	}

	return nil, warnings
}

func resolveBaseURL(spec *Spec) string {
	if len(spec.Servers) > 0 && spec.Servers[0].URL != "" {
		return strings.TrimRight(spec.Servers[0].URL, "/")
	}
	if spec.Host != "" {
		scheme := "https"
		if len(spec.Schemes) > 0 {
			scheme = spec.Schemes[0]
		}
		base := scheme + "://" + spec.Host
		if spec.BasePath != "" && spec.BasePath != "/" {
			base += spec.BasePath
		}
		return strings.TrimRight(base, "/")
	}
	return ""
}
