package yamlcodec

import (
	"errors"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// ParseCollection reads `opencollection.yml`.
func (c *Codec) ParseCollection(data []byte) (*model.CollectionConfig, error) {
	var doc collectionDoc
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	cfg := &model.CollectionConfig{
		Version: doc.OpenCollection,
		Name:    doc.Info.Name,
		Type:    "collection",
		Format:  model.FormatYML,
	}
	if doc.Info.Version != "" {
		cfg.Info = &model.CollectionInfo{
			Name:        doc.Info.Name,
			Description: doc.Info.Description,
			Version:     doc.Info.Version,
		}
	}
	// Collection-level runtime blocks carry variables and scripts.
	if doc.Runtime != nil {
		if vb := varBlocksFromRuntime(doc.Runtime); vb != nil {
			cfg.Vars = vb.PreRequest
		}
		if sc := scriptsFromRuntime(doc.Runtime); sc != nil {
			cfg.Script = sc
		}
	}
	return cfg, nil
}

// WriteCollection serialises a collection config.
func (c *Codec) WriteCollection(cfg *model.CollectionConfig) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("yamlcodec: nil collection config")
	}
	version := cfg.Version
	if version == "" {
		version = "1.0.0"
	}
	doc := collectionDoc{
		OpenCollection: version,
		Info: infoDoc{
			Name: cfg.Name,
		},
	}
	if cfg.Info != nil {
		doc.Info.Version = cfg.Info.Version
		doc.Info.Description = cfg.Info.Description
	}
	if len(cfg.Vars) > 0 || cfg.Script != nil {
		rt := &runtimeDoc{}
		for _, v := range cfg.Vars {
			rt.Variables = append(rt.Variables, variableToDoc(v))
		}
		if cfg.Script != nil {
			if cfg.Script.Req != nil {
				rt.Scripts = append(rt.Scripts, scriptDoc{Type: "pre-request", Code: *cfg.Script.Req})
			}
			if cfg.Script.Res != nil {
				rt.Scripts = append(rt.Scripts, scriptDoc{Type: "post-response", Code: *cfg.Script.Res})
			}
		}
		doc.Runtime = rt
	}
	return encode(&doc)
}

// ParseRequest reads a request file.
func (c *Codec) ParseRequest(data []byte) (*model.Request, error) {
	var doc requestDoc
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	req, err := requestFromDoc(&doc)
	if err != nil {
		return nil, err
	}
	// Tests live in the runtime scripts in this dialect.
	if doc.Runtime != nil {
		if t := testsFromRuntime(doc.Runtime); t != nil {
			req.Tests = t
		}
	}
	return req, nil
}

// WriteRequest serialises a request.
func (c *Codec) WriteRequest(req *model.Request) ([]byte, error) {
	if req == nil {
		return nil, errors.New("yamlcodec: nil request")
	}
	return encode(requestToDoc(req))
}

// ParseFolder reads a folder file.
func (c *Codec) ParseFolder(data []byte) (*model.Item, error) {
	var doc folderDoc
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	item := &model.Item{
		Type: model.ItemFolder,
		Name: doc.Info.Name,
		Seq:  doc.Info.Seq,
		Tags: append([]string(nil), doc.Info.Tags...),
	}
	// A folder's inherited settings live under a `request:` key in this dialect.
	if doc.Request != nil {
		if doc.Request.Auth != nil {
			item.Auth = authFromDoc(doc.Request.Auth)
		}
		for i, kv := range doc.Request.Headers {
			item.Headers = append(item.Headers, keyValue{uid: rowUID("header", i, kv.Name), kv: kv}.model())
		}
	}
	if doc.Runtime != nil {
		if vb := varBlocksFromRuntime(doc.Runtime); vb != nil {
			item.Vars = vb.PreRequest
		}
		if sc := scriptsFromRuntime(doc.Runtime); sc != nil {
			item.Script = sc
		}
	}
	return item, nil
}

// WriteFolder serialises a folder file.
func (c *Codec) WriteFolder(folder *model.Item) ([]byte, error) {
	if folder == nil {
		return nil, errors.New("yamlcodec: nil folder")
	}
	doc := folderDoc{Info: infoDoc{Name: folder.Name, Type: "folder", Seq: folder.Seq, Tags: folder.Tags}}
	if folder.Auth != nil || len(folder.Headers) > 0 {
		req := &httpDoc{}
		if folder.Auth != nil {
			req.Auth = authToDoc(folder.Auth)
		}
		for _, kv := range folder.Headers {
			req.Headers = append(req.Headers, kvDoc{
				Name:     kv.NameOrEmpty(),
				Value:    kv.ValueOrEmpty(),
				Disabled: !kv.Enabled,
			})
		}
		doc.Request = req
	}
	if len(folder.Vars) > 0 || folder.Script != nil {
		rt := &runtimeDoc{}
		for _, v := range folder.Vars {
			rt.Variables = append(rt.Variables, variableToDoc(v))
		}
		if folder.Script != nil {
			if folder.Script.Req != nil {
				rt.Scripts = append(rt.Scripts, scriptDoc{Type: "pre-request", Code: *folder.Script.Req})
			}
			if folder.Script.Res != nil {
				rt.Scripts = append(rt.Scripts, scriptDoc{Type: "post-response", Code: *folder.Script.Res})
			}
		}
		doc.Runtime = rt
	}
	return encode(&doc)
}

// ParseEnvironment reads an environment file.
func (c *Codec) ParseEnvironment(data []byte) (*model.Environment, error) {
	var doc environmentDoc
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	env := &model.Environment{
		Name:    doc.Name,
		Extends: doc.Extends,
		Color:   doc.Color,
		Secrets: append([]string(nil), doc.Secrets...),
		Format:  model.FormatYML,
	}

	secret := make(map[string]bool, len(doc.Secrets))
	for _, s := range doc.Secrets {
		secret[s] = true
	}

	for i, v := range doc.Variables {
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
		if secret[v.Name] {
			out.Secret = true
		}
		env.Variables = append(env.Variables, out)
	}
	return env, nil
}

// WriteEnvironment serialises an environment file.
func (c *Codec) WriteEnvironment(env *model.Environment) ([]byte, error) {
	if env == nil {
		return nil, errors.New("yamlcodec: nil environment")
	}
	doc := environmentDoc{
		Name:    env.Name,
		Extends: env.Extends,
		Color:   env.Color,
	}
	for _, v := range env.Variables {
		doc.Variables = append(doc.Variables, variableToDoc(v))
	}
	doc.Secrets = append(doc.Secrets, env.Secrets...)
	return encode(&doc)
}

// FormatVersion returns the opencollection version this codec writes.
func FormatVersion() string { return "1.0.0" }

// parseSeq parses a sequence number, tolerating an empty value.
func parseSeq(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
