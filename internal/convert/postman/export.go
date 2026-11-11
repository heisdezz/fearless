package postman

import (
	"encoding/json"
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

// Export converts a fearless model.Collection into Postman v2.1 collection JSON bytes.
func Export(coll *model.Collection) ([]byte, error) {
	if coll == nil {
		return nil, fmt.Errorf("collection cannot be nil")
	}

	pColl := CollectionJSON{
		Info: InfoJSON{
			PostmanID: coll.UID,
			Name:      coll.Name,
			Schema:    "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
			Version:   coll.Config.Version,
		},
	}

	if coll.Config.Docs != nil {
		pColl.Info.Description = *coll.Config.Docs
	}

	// Variables
	for _, v := range coll.Config.Vars {
		pColl.Variable = append(pColl.Variable, VariableJSON{
			Key:      v.NameOrEmpty(),
			Value:    v.Value,
			Disabled: !v.Enabled,
		})
	}

	// Collection Auth
	if coll.Config.Auth != nil {
		pColl.Auth = exportAuth(coll.Config.Auth)
	}

	// Collection Scripts
	if coll.Config.Script != nil {
		pColl.Event = exportEvents(coll.Config.Script, nil)
	}

	// Items
	pColl.Item = exportItems(coll.Items)

	return json.MarshalIndent(pColl, "", "  ")
}

// ExportEnvironment converts a fearless model.Environment into Postman environment JSON bytes.
func ExportEnvironment(env *model.Environment) ([]byte, error) {
	if env == nil {
		return nil, fmt.Errorf("environment cannot be nil")
	}

	secretSet := make(map[string]bool)
	for _, s := range env.Secrets {
		secretSet[s] = true
	}

	pEnv := EnvironmentJSON{
		ID:   env.UID,
		Name: env.Name,
	}

	for _, v := range env.Variables {
		varType := "default"
		if secretSet[v.NameOrEmpty()] {
			varType = "secret"
		}
		pEnv.Values = append(pEnv.Values, EnvironmentValJSON{
			Key:     v.NameOrEmpty(),
			Value:   fmt.Sprintf("%v", v.Value),
			Type:    varType,
			Enabled: v.Enabled,
		})
	}

	return json.MarshalIndent(pEnv, "", "  ")
}

func exportItems(items []*model.Item) []ItemJSON {
	var result []ItemJSON
	for _, it := range items {
		if it == nil {
			continue
		}

		if it.IsFolder() {
			folderJSON := ItemJSON{
				ID:   it.UID,
				Name: it.Name,
				Item: exportItems(it.Children),
			}
			if it.Auth != nil {
				folderJSON.Auth = exportAuth(it.Auth)
			}
			if it.Script != nil {
				folderJSON.Event = exportEvents(it.Script, nil)
			}
			result = append(result, folderJSON)
		} else {
			req := it.Effective()
			if req == nil {
				continue
			}
			itemJSON := ItemJSON{
				ID:      it.UID,
				Name:    it.Name,
				Request: exportRequest(req),
			}
			if req.Script != nil || req.Tests != nil {
				itemJSON.Event = exportEvents(req.Script, req.Tests)
			}
			result = append(result, itemJSON)
		}
	}
	return result
}

func exportRequest(req *model.Request) *RequestJSON {
	r := &RequestJSON{
		Method: req.Method,
	}

	if req.Docs != nil {
		r.Description = *req.Docs
	}

	// URL
	var queryJSON []QueryJSON
	for _, p := range req.QueryParams() {
		queryJSON = append(queryJSON, QueryJSON{
			Key:      p.NameOrEmpty(),
			Value:    p.ValueOrEmpty(),
			Disabled: !p.Enabled,
		})
	}

	var varJSON []VariableJSON
	for _, p := range req.PathParams() {
		varJSON = append(varJSON, VariableJSON{
			Key:      p.NameOrEmpty(),
			Value:    p.ValueOrEmpty(),
			Disabled: !p.Enabled,
		})
	}

	u := URLJSON{
		Raw:      req.URL,
		Query:    queryJSON,
		Variable: varJSON,
	}
	r.URL = u

	// Headers
	for _, h := range req.LiveHeaders() {
		r.Header = append(r.Header, HeaderJSON{
			Key:      h.NameOrEmpty(),
			Value:    h.ValueOrEmpty(),
			Disabled: !h.Enabled,
		})
	}

	// Auth
	if req.Auth != nil {
		r.Auth = exportAuth(req.Auth)
	}

	// Body
	if req.Body != nil {
		r.Body = exportBody(req.Body)
	}

	return r
}

func exportBody(b *model.Body) *BodyJSON {
	if b == nil || b.IsEmpty() {
		return nil
	}

	switch b.Mode {
	case model.BodyJSON:
		raw := ""
		if b.JSON != nil {
			raw = *b.JSON
		}
		return &BodyJSON{
			Mode: "raw",
			Raw:  raw,
			Options: &BodyOptionsJSON{
				Raw: &RawOptionsJSON{
					Language: "json",
				},
			},
		}
	case model.BodyXML:
		raw := ""
		if b.XML != nil {
			raw = *b.XML
		}
		return &BodyJSON{
			Mode: "raw",
			Raw:  raw,
			Options: &BodyOptionsJSON{
				Raw: &RawOptionsJSON{
					Language: "xml",
				},
			},
		}
	case model.BodyText, model.BodySparql:
		raw := ""
		if b.Mode == model.BodyText && b.Text != nil {
			raw = *b.Text
		} else if b.Mode == model.BodySparql && b.Sparql != nil {
			raw = *b.Sparql
		}
		return &BodyJSON{
			Mode: "raw",
			Raw:  raw,
		}
	case model.BodyFormURLEncoded:
		var items []URLEncodeJSON
		for _, kv := range b.FormURLEncoded {
			items = append(items, URLEncodeJSON{
				Key:      kv.NameOrEmpty(),
				Value:    kv.ValueOrEmpty(),
				Disabled: !kv.Enabled,
			})
		}
		return &BodyJSON{
			Mode:       "urlencoded",
			URLEncoded: items,
		}
	case model.BodyMultipartForm:
		var items []FormDataJSON
		for _, part := range b.MultipartForm {
			name := ""
			if part.Name != nil {
				name = *part.Name
			}
			val := ""
			if part.Value != nil {
				val = *part.Value
			}
			if part.Type == model.MultipartFile {
				filePath := val
				if strings.HasPrefix(val, "@file(") && strings.HasSuffix(val, ")") {
					filePath = strings.TrimSuffix(strings.TrimPrefix(val, "@file("), ")")
				}
				ct := ""
				if part.ContentType != nil {
					ct = *part.ContentType
				}
				items = append(items, FormDataJSON{
					Key:         name,
					Type:        "file",
					Src:         filePath,
					ContentType: ct,
					Disabled:    !part.Enabled,
				})
			} else {
				items = append(items, FormDataJSON{
					Key:      name,
					Type:     "text",
					Value:    val,
					Disabled: !part.Enabled,
				})
			}
		}
		return &BodyJSON{
			Mode:     "formdata",
			FormData: items,
		}
	case model.BodyGraphQL:
		if b.GraphQL != nil {
			q := ""
			if b.GraphQL.Query != nil {
				q = *b.GraphQL.Query
			}
			v := ""
			if b.GraphQL.Variables != nil {
				v = *b.GraphQL.Variables
			}
			return &BodyJSON{
				Mode: "graphql",
				GraphQL: &GraphQLJSON{
					Query:     q,
					Variables: v,
				},
			}
		}
	case model.BodyFile:
		for _, f := range b.File {
			if f.Selected && f.FilePath != nil {
				return &BodyJSON{
					Mode: "file",
					File: &FileJSON{
						Src: *f.FilePath,
					},
				}
			}
		}
	}

	return nil
}

func exportAuth(a *model.Auth) *AuthJSON {
	if a == nil || a.Mode == model.AuthNone || a.Mode == model.AuthInherit {
		return nil
	}

	switch a.Mode {
	case model.AuthBasic:
		var u, p string
		if a.Username != nil {
			u = *a.Username
		}
		if a.Password != nil {
			p = *a.Password
		}
		return &AuthJSON{
			Type: "basic",
			Basic: []AuthParamJSON{
				{Key: "username", Value: u, Type: "string"},
				{Key: "password", Value: p, Type: "string"},
			},
		}
	case model.AuthBearer:
		var t string
		if a.Token != nil {
			t = *a.Token
		}
		return &AuthJSON{
			Type: "bearer",
			Bearer: []AuthParamJSON{
				{Key: "token", Value: t, Type: "string"},
			},
		}
	case model.AuthAPIKey:
		var k, v string
		if a.Key != nil {
			k = *a.Key
		}
		if a.Value != nil {
			v = *a.Value
		}
		in := "header"
		if a.Placement == model.APIKeyQueryParams {
			in = "query"
		}
		return &AuthJSON{
			Type: "apikey",
			APIKey: []AuthParamJSON{
				{Key: "key", Value: k, Type: "string"},
				{Key: "value", Value: v, Type: "string"},
				{Key: "in", Value: in, Type: "string"},
			},
		}
	case model.AuthDigest:
		var u, p string
		if a.Username != nil {
			u = *a.Username
		}
		if a.Password != nil {
			p = *a.Password
		}
		return &AuthJSON{
			Type: "digest",
			Digest: []AuthParamJSON{
				{Key: "username", Value: u, Type: "string"},
				{Key: "password", Value: p, Type: "string"},
			},
		}
	case model.AuthNTLM:
		var u, p, d string
		if a.Username != nil {
			u = *a.Username
		}
		if a.Password != nil {
			p = *a.Password
		}
		if a.Domain != nil {
			d = *a.Domain
		}
		return &AuthJSON{
			Type: "ntlm",
			NTLM: []AuthParamJSON{
				{Key: "username", Value: u, Type: "string"},
				{Key: "password", Value: p, Type: "string"},
				{Key: "domain", Value: d, Type: "string"},
			},
		}
	case model.AuthAWSV4:
		var ak, sk, reg, svc string
		if a.AccessKeyID != nil {
			ak = *a.AccessKeyID
		}
		if a.SecretAccessKey != nil {
			sk = *a.SecretAccessKey
		}
		if a.Region != nil {
			reg = *a.Region
		}
		if a.Service != nil {
			svc = *a.Service
		}
		return &AuthJSON{
			Type: "awsv4",
			AWSV4: []AuthParamJSON{
				{Key: "accessKey", Value: ak, Type: "string"},
				{Key: "secretKey", Value: sk, Type: "string"},
				{Key: "region", Value: reg, Type: "string"},
				{Key: "service", Value: svc, Type: "string"},
			},
		}
	}

	return nil
}

func exportEvents(scripts *model.Scripts, tests *string) []EventJSON {
	var events []EventJSON
	if scripts != nil && scripts.Req != nil && *scripts.Req != "" {
		events = append(events, EventJSON{
			Listen: "prerequest",
			Script: ScriptJSON{
				Type: "text/javascript",
				Exec: strings.Split(*scripts.Req, "\n"),
			},
		})
	}
	if scripts != nil && scripts.Res != nil && *scripts.Res != "" {
		events = append(events, EventJSON{
			Listen: "test",
			Script: ScriptJSON{
				Type: "text/javascript",
				Exec: strings.Split(*scripts.Res, "\n"),
			},
		})
	} else if tests != nil && *tests != "" {
		events = append(events, EventJSON{
			Listen: "test",
			Script: ScriptJSON{
				Type: "text/javascript",
				Exec: strings.Split(*tests, "\n"),
			},
		})
	}
	return events
}
