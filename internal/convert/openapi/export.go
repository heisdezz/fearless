package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"fearless/internal/core/model"
)

// Export converts a fearless model.Collection into an OpenAPI 3.0.3 specification JSON.
func Export(coll *model.Collection) ([]byte, error) {
	if coll == nil {
		return nil, fmt.Errorf("collection cannot be nil")
	}

	docDesc := ""
	if coll.Config.Docs != nil {
		docDesc = *coll.Config.Docs
	}

	spec := Spec{
		OpenAPI: "3.0.3",
		Info: InfoObject{
			Title:       coll.Name,
			Description: docDesc,
			Version:     "1.0.0",
		},
		Paths: make(map[string]PathItemObject),
	}

	// Server URL from baseUrl variable
	for _, v := range coll.Config.Vars {
		if strings.EqualFold(v.NameOrEmpty(), "baseUrl") {
			spec.Servers = append(spec.Servers, ServerObject{
				URL: fmt.Sprintf("%v", v.Value),
			})
			break
		}
	}

	for _, it := range coll.Items {
		exportItemOperations(it, nil, spec.Paths)
	}

	return json.MarshalIndent(spec, "", "  ")
}

func exportItemOperations(it *model.Item, tags []string, paths map[string]PathItemObject) {
	if it == nil {
		return
	}

	if it.IsFolder() {
		folderTags := append([]string(nil), tags...)
		folderTags = append(folderTags, it.Name)
		for _, child := range it.Children {
			exportItemOperations(child, folderTags, paths)
		}
		return
	}

	req := it.Effective()
	if req == nil {
		return
	}

	pathStr := normalizePath(req.URL)
	pathItem, exists := paths[pathStr]
	if !exists {
		pathItem = PathItemObject{}
	}

	op := exportOperation(it, req, tags)

	switch strings.ToUpper(req.Method) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "OPTIONS":
		pathItem.Options = op
	case "HEAD":
		pathItem.Head = op
	default:
		pathItem.Get = op
	}

	paths[pathStr] = pathItem
}

func exportOperation(it *model.Item, req *model.Request, tags []string) *OperationObject {
	desc := ""
	if req.Docs != nil {
		desc = *req.Docs
	}

	op := &OperationObject{
		Tags:        tags,
		Summary:     it.Name,
		Description: desc,
	}

	// Query Parameters
	for _, p := range req.QueryParams() {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:     p.NameOrEmpty(),
			In:       "query",
			Required: p.Enabled,
			Example:  p.ValueOrEmpty(),
		})
	}

	// Path Parameters
	for _, p := range req.PathParams() {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:     p.NameOrEmpty(),
			In:       "path",
			Required: true,
			Example:  p.ValueOrEmpty(),
		})
	}

	// Headers
	for _, h := range req.LiveHeaders() {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:     h.NameOrEmpty(),
			In:       "header",
			Required: h.Enabled,
			Example:  h.ValueOrEmpty(),
		})
	}

	// Request Body
	if req.Body != nil && !req.Body.IsEmpty() {
		switch req.Body.Mode {
		case model.BodyJSON:
			var example any
			if req.Body.JSON != nil {
				_ = json.Unmarshal([]byte(*req.Body.JSON), &example)
			}
			op.RequestBody = &RequestBodyObject{
				Content: map[string]MediaTypeObject{
					"application/json": {
						Example: example,
					},
				},
			}
		case model.BodyFormURLEncoded:
			op.RequestBody = &RequestBodyObject{
				Content: map[string]MediaTypeObject{
					"application/x-www-form-urlencoded": {},
				},
			}
		case model.BodyMultipartForm:
			op.RequestBody = &RequestBodyObject{
				Content: map[string]MediaTypeObject{
					"multipart/form-data": {},
				},
			}
		}
	}

	return op
}

func normalizePath(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "{{baseUrl}}") {
		path := strings.TrimPrefix(rawURL, "{{baseUrl}}")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if idx := strings.Index(path, "?"); idx != -1 {
			path = path[:idx]
		}
		return path
	}

	// If scheme present
	if strings.Contains(rawURL, "://") {
		parts := strings.SplitN(rawURL, "://", 2)
		rest := parts[1]
		firstSlash := strings.Index(rest, "/")
		if firstSlash != -1 {
			path := rest[firstSlash:]
			if idx := strings.Index(path, "?"); idx != -1 {
				path = path[:idx]
			}
			return path
		}
		return "/"
	}

	if !strings.HasPrefix(rawURL, "/") {
		rawURL = "/" + rawURL
	}
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	return rawURL
}
