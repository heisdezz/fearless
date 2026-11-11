package insomnia

import (
	"encoding/json"
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

// Export converts a fearless model.Collection into Insomnia v4 export JSON bytes.
func Export(coll *model.Collection) ([]byte, error) {
	if coll == nil {
		return nil, fmt.Errorf("collection cannot be nil")
	}

	workspaceID := "wrk_" + coll.UID
	baseEnvID := "env_" + coll.UID

	exp := ExportJSON{
		Type:         "export",
		ExportFormat: 4,
		ExportDate:   "2026-11-11T00:00:00.000Z",
		ExportSource: "fearless",
		Resources: []ResourceJSON{
			{
				ID:          workspaceID,
				Type:        "workspace",
				Name:        coll.Name,
				Description: docString(coll.Config.Docs),
			},
		},
	}

	// Base environment
	baseEnvData := make(map[string]any)
	for _, v := range coll.Config.Vars {
		if v.Enabled {
			baseEnvData[v.NameOrEmpty()] = v.Value
		}
	}
	exp.Resources = append(exp.Resources, ResourceJSON{
		ID:       baseEnvID,
		Type:     "environment",
		ParentID: workspaceID,
		Name:     "Base Environment",
		Data:     baseEnvData,
	})

	// Sub-environments
	for _, env := range coll.Environments {
		if env == nil {
			continue
		}
		data := make(map[string]any)
		for _, v := range env.Variables {
			if v.Enabled {
				data[v.NameOrEmpty()] = v.Value
			}
		}
		exp.Resources = append(exp.Resources, ResourceJSON{
			ID:        "env_" + env.UID,
			Type:      "environment",
			ParentID:  baseEnvID,
			Name:      env.Name,
			Color:     env.Color,
			Data:      data,
			IsPrivate: len(env.Secrets) > 0,
		})
	}

	// Items (folders & requests)
	for _, it := range coll.Items {
		exportItemResources(it, workspaceID, &exp.Resources)
	}

	return json.MarshalIndent(exp, "", "  ")
}

func exportItemResources(it *model.Item, parentID string, resources *[]ResourceJSON) {
	if it == nil {
		return
	}

	if it.IsFolder() {
		folderID := "fld_" + it.UID
		folderRes := ResourceJSON{
			ID:          folderID,
			Type:        "request_group",
			ParentID:    parentID,
			Name:        it.Name,
			Description: docString(requestDocs(it.Request)),
		}
		if len(it.Vars) > 0 {
			folderRes.Environment = make(map[string]any)
			for _, v := range it.Vars {
				if v.Enabled {
					folderRes.Environment[v.NameOrEmpty()] = v.Value
				}
			}
		}
		*resources = append(*resources, folderRes)

		for _, child := range it.Children {
			exportItemResources(child, folderID, resources)
		}
	} else {
		req := it.Effective()
		if req == nil {
			return
		}
		reqID := "req_" + it.UID

		var params []ParameterJSON
		for _, p := range req.QueryParams() {
			params = append(params, ParameterJSON{
				Name:     p.NameOrEmpty(),
				Value:    p.ValueOrEmpty(),
				Disabled: !p.Enabled,
			})
		}

		var headers []HeaderJSON
		for _, h := range req.LiveHeaders() {
			headers = append(headers, HeaderJSON{
				Name:     h.NameOrEmpty(),
				Value:    h.ValueOrEmpty(),
				Disabled: !h.Enabled,
			})
		}

		body := exportBody(req.Body)
		auth := exportAuth(req.Auth)

		followRedirects := "global"
		if req.Settings.FollowRedirects != nil {
			if *req.Settings.FollowRedirects {
				followRedirects = "on"
			} else {
				followRedirects = "off"
			}
		}

		reqRes := ResourceJSON{
			ID:                     reqID,
			Type:                   "request",
			ParentID:               parentID,
			Name:                   it.Name,
			Description:            docString(req.Docs),
			URL:                    req.URL,
			Method:                 req.Method,
			Parameters:             params,
			Headers:                headers,
			Body:                   body,
			Authentication:         auth,
			SettingFollowRedirects: followRedirects,
		}
		*resources = append(*resources, reqRes)
	}
}

func exportBody(b *model.Body) *BodyJSON {
	if b == nil || b.IsEmpty() {
		return nil
	}

	switch b.Mode {
	case model.BodyJSON:
		txt := ""
		if b.JSON != nil {
			txt = *b.JSON
		}
		return &BodyJSON{
			MimeType: "application/json",
			Text:     txt,
		}
	case model.BodyGraphQL:
		txt := ""
		if b.GraphQL != nil && b.GraphQL.Query != nil {
			payload := map[string]any{
				"query": *b.GraphQL.Query,
			}
			if b.GraphQL.Variables != nil {
				var v any
				if json.Unmarshal([]byte(*b.GraphQL.Variables), &v) == nil {
					payload["variables"] = v
				} else {
					payload["variables"] = *b.GraphQL.Variables
				}
			}
			if bytes, err := json.Marshal(payload); err == nil {
				txt = string(bytes)
			}
		}
		return &BodyJSON{
			MimeType: "application/graphql",
			Text:     txt,
		}
	case model.BodyFormURLEncoded:
		var params []BodyParamJSON
		for _, kv := range b.FormURLEncoded {
			params = append(params, BodyParamJSON{
				Name:     kv.NameOrEmpty(),
				Value:    kv.ValueOrEmpty(),
				Disabled: !kv.Enabled,
			})
		}
		return &BodyJSON{
			MimeType: "application/x-www-form-urlencoded",
			Params:   params,
		}
	case model.BodyMultipartForm:
		var params []BodyParamJSON
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
				params = append(params, BodyParamJSON{
					Name:     name,
					Type:     "file",
					FileName: filePath,
					Disabled: !part.Enabled,
				})
			} else {
				params = append(params, BodyParamJSON{
					Name:     name,
					Type:     "text",
					Value:    val,
					Disabled: !part.Enabled,
				})
			}
		}
		return &BodyJSON{
			MimeType: "multipart/form-data",
			Params:   params,
		}
	case model.BodyXML:
		txt := ""
		if b.XML != nil {
			txt = *b.XML
		}
		return &BodyJSON{
			MimeType: "application/xml",
			Text:     txt,
		}
	case model.BodyFile:
		for _, f := range b.File {
			if f.Selected && f.FilePath != nil {
				return &BodyJSON{
					FileName: *f.FilePath,
				}
			}
		}
	case model.BodyText, model.BodySparql:
		txt := ""
		if b.Mode == model.BodyText && b.Text != nil {
			txt = *b.Text
		} else if b.Mode == model.BodySparql && b.Sparql != nil {
			txt = *b.Sparql
		}
		return &BodyJSON{
			MimeType: "text/plain",
			Text:     txt,
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
			Type:     "basic",
			Username: u,
			Password: p,
		}
	case model.AuthBearer:
		var t string
		if a.Token != nil {
			t = *a.Token
		}
		return &AuthJSON{
			Type:  "bearer",
			Token: t,
		}
	case model.AuthAPIKey:
		var k, v string
		if a.Key != nil {
			k = *a.Key
		}
		if a.Value != nil {
			v = *a.Value
		}
		addTo := "header"
		if a.Placement == model.APIKeyQueryParams {
			addTo = "queryParams"
		}
		return &AuthJSON{
			Type:  "apikey",
			Key:   k,
			Value: v,
			AddTo: addTo,
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
			Type:     "digest",
			Username: u,
			Password: p,
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
			Type:     "ntlm",
			Username: u,
			Password: p,
			Domain:   d,
		}
	case model.AuthAWSV4:
		var ak, sk, tok, reg, svc string
		if a.AccessKeyID != nil {
			ak = *a.AccessKeyID
		}
		if a.SecretAccessKey != nil {
			sk = *a.SecretAccessKey
		}
		if a.SessionToken != nil {
			tok = *a.SessionToken
		}
		if a.Region != nil {
			reg = *a.Region
		}
		if a.Service != nil {
			svc = *a.Service
		}
		return &AuthJSON{
			Type:            "iam",
			AccessKeyID:     ak,
			SecretAccessKey: sk,
			SessionToken:    tok,
			Region:          reg,
			Service:         svc,
		}
	}

	return nil
}

func docString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func requestDocs(req *model.Request) *string {
	if req == nil {
		return nil
	}
	return req.Docs
}
