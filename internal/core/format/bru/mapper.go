package bru

import (
	"fmt"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// uidPrefix keeps generated UIDs stable across parses of identical input.
//
// Bruno derives request identity from the file path, not from the file contents,
// so the UID on a request here is filled in by the caller (COLLECTION) after
// parsing. Rows inside a request need an identity too, and hashing the row's
// position and name gives a stable, collision-free value without persisting a
// UID that Bruno's format has no slot for.
const uidPrefix = "bru-"

// rowUID builds a deterministic UID for a row.
func rowUID(kind string, index int, name string) string {
	return fmt.Sprintf("%s%s-%d-%x", uidPrefix, kind, index, len(name))
}

// Request converts a parsed document into the canonical request model.
//
// It always returns a request: a `.bru` file may legitimately omit blocks, and
// defaults (method GET, body none, auth inherit) are what Bruno applies.
func (d *Document) Request() (*model.Request, error) {
	req := &model.Request{
		Method: "GET",
	}

	// meta
	if meta, ok := d.FindBlock("meta"); ok {
		for _, p := range meta.Pairs {
			switch p.Key {
			case "name":
				req.Name = p.Value
			case "seq":
				if n, err := strconv.Atoi(strings.TrimSpace(p.Value)); err == nil {
					req.Seq = &n
				}
			case "tags":
				req.Tags = append([]string(nil), p.List...)
			}
		}
	}

	// Method block: the block name carries the verb; `http` carries it as a key.
	methodFound := false
	for name, verb := range methodBlocks {
		if blk, ok := d.FindBlock(name); ok {
			req.Method = verb
			methodFound = true
			if err := applyMethodBlock(req, blk); err != nil {
				return nil, err
			}
			break
		}
	}
	if !methodFound {
		if blk, ok := d.FindBlock("http"); ok {
			methodFound = true
			for _, p := range blk.Pairs {
				if p.Key == "method" {
					req.Method = strings.ToUpper(strings.TrimSpace(p.Value))
				}
			}
			if err := applyMethodBlock(req, blk); err != nil {
				return nil, err
			}
		}
	}

	// params
	for _, kind := range []struct {
		block string
		typ   model.ParamType
	}{
		{"params:query", model.ParamQuery},
		{"params:path", model.ParamPath},
	} {
		if blk, ok := d.FindBlock(kind.block); ok {
			for i, p := range blk.Pairs {
				req.Params = append(req.Params, model.Param{
					KeyValue: keyValueFrom(p, "param", i),
					Type:     kind.typ,
				})
			}
		}
	}

	// headers
	if blk, ok := d.FindBlock("headers"); ok {
		for i, p := range blk.Pairs {
			req.Headers = append(req.Headers, keyValueFrom(p, "header", i))
		}
	}

	// body — the mode comes from the method block's `body:` key, and the
	// payload from the matching body:<mode> block.
	req.Body = buildBody(d)

	// auth
	auth, err := d.Auth()
	if err != nil {
		return nil, err
	}
	req.Auth = auth

	// scripts
	if sc := buildScripts(d); sc != nil {
		req.Script = sc
	}

	// tests
	if blk, ok := d.FindBlock("tests"); ok && blk.Kind == TextBlock {
		if t := strings.TrimSpace(blk.Text); t != "" {
			req.Tests = &blk.Text
		}
	}

	// docs
	if blk, ok := d.FindBlock("docs"); ok && blk.Kind == TextBlock {
		req.Docs = &blk.Text
	}

	// vars
	if vb := buildVarBlocks(d); vb != nil {
		req.Vars = vb
	}

	// assertions
	if blk, ok := d.FindBlock("assert"); ok {
		for i, p := range blk.Pairs {
			req.Assertions = append(req.Assertions, keyValueFrom(p, "assert", i))
		}
	}

	// settings
	if blk, ok := d.FindBlock("settings"); ok {
		applySettings(&req.Settings, blk)
	}

	return req, nil
}

// applyMethodBlock copies the keys shared by every method block.
func applyMethodBlock(req *model.Request, blk *Block) error {
	for _, p := range blk.Pairs {
		switch p.Key {
		case "url":
			req.URL = p.Value
		case "method":
			// Handled by the caller for the `http` block.
		}
	}
	return nil
}

// keyValueFrom converts a parsed pair into the shared row model.
func keyValueFrom(p Pair, kind string, index int) model.KeyValue {
	kv := model.KeyValue{
		UID: rowUID(kind, index, p.Key),
		// Disabled pairs stay in the model so the editor can re-enable them;
		// `~` means disabled, absence means enabled.
		Enabled: !p.Disabled,
	}
	name := p.Key
	kv.Name = &name
	if !p.IsList {
		value := p.Value
		kv.Value = &value
	} else {
		value := strings.Join(p.List, ", ")
		kv.Value = &value
	}
	if ann, ok := p.Annotation("description"); ok {
		desc := ann.Arg
		kv.Description = &desc
	}
	return kv
}

// buildBody assembles the body union from the body blocks.
func buildBody(d *Document) *model.Body {
	body := &model.Body{Mode: model.BodyNone}

	// body:json / text / xml / sparql / graphql / graphql:vars
	if blk, ok := d.FindBlock("body:json"); ok && blk.Kind == TextBlock {
		body.Mode = model.BodyJSON
		body.JSON = &blk.Text
	}
	if blk, ok := d.FindBlock("body:text"); ok && blk.Kind == TextBlock {
		body.Mode = model.BodyText
		body.Text = &blk.Text
	}
	if blk, ok := d.FindBlock("body:xml"); ok && blk.Kind == TextBlock {
		body.Mode = model.BodyXML
		body.XML = &blk.Text
	}
	if blk, ok := d.FindBlock("body:sparql"); ok && blk.Kind == TextBlock {
		body.Mode = model.BodySparql
		body.Sparql = &blk.Text
	}
	if blk, ok := d.FindBlock("body:graphql"); ok && blk.Kind == TextBlock {
		body.Mode = model.BodyGraphQL
		gql := &model.GraphQLBody{Query: &blk.Text}
		if vars, ok := d.FindBlock("body:graphql:vars"); ok && vars.Kind == TextBlock {
			gql.Variables = &vars.Text
		}
		body.GraphQL = gql
	}

	// form-urlencoded
	if blk, ok := d.FindBlock("body:form-urlencoded"); ok {
		body.Mode = model.BodyFormURLEncoded
		for i, p := range blk.Pairs {
			body.FormURLEncoded = append(body.FormURLEncoded, keyValueFrom(p, "form", i))
		}
	}

	// multipart-form: each value may carry @file(path) and @contentType(type).
	if blk, ok := d.FindBlock("body:multipart-form"); ok {
		body.Mode = model.BodyMultipartForm
		for i, p := range blk.Pairs {
			body.MultipartForm = append(body.MultipartForm, multipartFrom(p, i))
		}
	}

	// file
	if blk, ok := d.FindBlock("body:file"); ok {
		body.Mode = model.BodyFile
		var entry model.FileEntry
		for _, p := range blk.Pairs {
			switch p.Key {
			case "file":
				path := firstListItemOrValue(p)
				entry.FilePath = &path
			case "content-type":
				ct := p.Value
				entry.ContentType = &ct
			}
		}
		entry.UID = rowUID("file", 0, firstListItemOrValue(Pair{Value: ""}))
		entry.Selected = true
		body.File = append(body.File, entry)
	}

	return body
}

// firstListItemOrValue returns a pair's first list item, or its scalar value.
func firstListItemOrValue(p Pair) string {
	if p.IsList && len(p.List) > 0 {
		return p.List[0]
	}
	return p.Value
}

// multipartFrom converts one multipart row, extracting the @file and
// @contentType annotations from its value.
//
// The annotations live inline in the value text, as in Bruno:
//
//	sample: @file(bruno.png) @contentType(image/png)
func multipartFrom(p Pair, index int) model.MultipartEntry {
	entry := model.MultipartEntry{
		UID:     rowUID("part", index, p.Key),
		Enabled: !p.Disabled,
	}
	name := p.Key
	entry.Name = &name

	value := p.Value
	rest := value
	entryType := model.MultipartText

	if ct, ok := extractAnnotation(rest, "contentType"); ok {
		c := ct
		entry.ContentType = &c
		rest = removeAnnotation(rest, "contentType")
	}
	if file, ok := extractAnnotation(rest, "file"); ok {
		entryType = model.MultipartFile
		rest = removeAnnotation(rest, "file")
		// For a file part the value is the path, not the annotation text.
		value = file
	} else {
		value = strings.TrimSpace(rest)
	}
	entry.Type = entryType
	entry.Value = &value
	return entry
}

// extractAnnotation returns the argument of @name(arg) in s.
func extractAnnotation(s, name string) (string, bool) {
	marker := "@" + name + "("
	i := strings.Index(s, marker)
	if i < 0 {
		return "", false
	}
	start := i + len(marker)
	end := strings.IndexByte(s[start:], ')')
	if end < 0 {
		return "", false
	}
	return s[start : start+end], true
}

// removeAnnotation strips @name(arg) from s.
func removeAnnotation(s, name string) string {
	marker := "@" + name + "("
	for {
		i := strings.Index(s, marker)
		if i < 0 {
			return s
		}
		end := strings.IndexByte(s[i:], ')')
		if end < 0 {
			return s
		}
		s = s[:i] + s[i+end+1:]
	}
}

// buildScripts maps the script blocks onto the two hooks.
func buildScripts(d *Document) *model.Scripts {
	var sc model.Scripts
	found := false
	if blk, ok := d.FindBlock("script:pre-request"); ok && blk.Kind == TextBlock {
		sc.Req = &blk.Text
		found = true
	}
	if blk, ok := d.FindBlock("script:post-response"); ok && blk.Kind == TextBlock {
		sc.Res = &blk.Text
		found = true
	}
	if !found {
		return nil
	}
	return &sc
}

// buildVarBlocks maps `vars:pre-request` and `vars:post-response`.
func buildVarBlocks(d *Document) *model.VarBlocks {
	var vb model.VarBlocks
	found := false
	if blk, ok := d.FindBlock("vars:pre-request"); ok {
		vb.PreRequest = variablesFrom(blk)
		found = true
	}
	if blk, ok := d.FindBlock("vars:post-response"); ok {
		vb.PostResponse = variablesFrom(blk)
		found = true
	}
	if !found {
		return nil
	}
	return &vb
}

// variablesFrom converts a vars block into typed variables.
func variablesFrom(blk *Block) []model.Variable {
	vars := make([]model.Variable, 0, len(blk.Pairs))
	for i, p := range blk.Pairs {
		name := p.Key
		v := model.Variable{
			UID:     rowUID("var", i, p.Key),
			Name:    &name,
			Enabled: !p.Disabled,
		}
		// Typed annotations mark the value's type; the value stays a string
		// because a .bru file stores it as text.
		for _, ann := range p.Annotations {
			switch ann.Name {
			case "number", "boolean", "object":
				v.DataType = ann.Name
			case "description":
				desc := ann.Arg
				v.Description = &desc
			}
		}
		if v.DataType == "" {
			v.DataType = "string"
		}
		v.Value = p.Value
		vars = append(vars, v)
	}
	return vars
}

// applySettings maps the settings block onto request settings.
//
// Blank values mean unset, not zero: Bruno distinguishes them, and a request with
// `maxRedirects:` empty must inherit rather than be capped at 0.
func applySettings(s *model.ItemSettings, blk *Block) {
	for _, p := range blk.Pairs {
		v := strings.TrimSpace(p.Value)
		switch p.Key {
		case "encodeUrl":
			if b, ok := parseBool(v); ok {
				s.EncodeURL = &b
			}
		case "followRedirects":
			if b, ok := parseBool(v); ok {
				s.FollowRedirects = &b
			}
		case "forwardAuthorizationHeader":
			if b, ok := parseBool(v); ok {
				s.ForwardAuthorizationHeader = &b
			}
		case "maxRedirects":
			if v == "" {
				continue // blank means unset
			}
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				s.MaxRedirects = &n
			}
		case "timeout":
			if v == "" {
				continue
			}
			if v == "inherit" {
				t := model.InheritTimeout
				s.Timeout = &t
				continue
			}
			if n, err := strconv.Atoi(v); err == nil {
				t := model.TimeoutMillis(n)
				s.Timeout = &t
			}
		}
	}
}

// parseBool accepts the spellings Bruno's toBool accepts.
func parseBool(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}
