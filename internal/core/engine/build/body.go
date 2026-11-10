package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// RenderBody renders the request body into wire bytes, determines the appropriate Content-Type,
// and computes the exact Content-Length.
func RenderBody(
	body *model.Body,
	scopeVars map[string]any,
	basePath string,
	explicitContentType string,
	onUnresolved func(name string),
) (data []byte, contentType string, contentLength int64, err error) {
	if body == nil || body.Mode == model.BodyNone || body.Mode == "" {
		// mode none: no body, Content-Type is suppressed
		return nil, "", 0, nil
	}

	interpOpts := vars.InterpolateOptions{
		EscapeJSON:   false,
		OnUnresolved: onUnresolved,
	}

	switch body.Mode {
	case model.BodyJSON:
		raw := ""
		if body.JSON != nil {
			raw = *body.JSON
		}
		decommented := StripComments(raw)
		jsonOpts := interpOpts
		jsonOpts.EscapeJSON = true
		interpolated := vars.Interpolate(decommented, scopeVars, jsonOpts)
		data = []byte(interpolated)
		contentType = "application/json"

	case model.BodyText:
		raw := ""
		if body.Text != nil {
			raw = *body.Text
		}
		interpolated := vars.Interpolate(raw, scopeVars, interpOpts)
		data = []byte(interpolated)
		contentType = "text/plain"

	case model.BodyXML:
		raw := ""
		if body.XML != nil {
			raw = *body.XML
		}
		interpolated := vars.Interpolate(raw, scopeVars, interpOpts)
		data = []byte(interpolated)
		contentType = "application/xml"

	case model.BodySparql:
		raw := ""
		if body.Sparql != nil {
			raw = *body.Sparql
		}
		interpolated := vars.Interpolate(raw, scopeVars, interpOpts)
		data = []byte(interpolated)
		contentType = "application/sparql-query"

	case model.BodyGraphQL:
		if body.GraphQL == nil {
			return nil, "", 0, nil
		}
		rawQuery := ""
		if body.GraphQL.Query != nil {
			rawQuery = *body.GraphQL.Query
		}
		query := vars.Interpolate(rawQuery, scopeVars, interpOpts)
		payload := map[string]any{
			"query": query,
		}

		if body.GraphQL.Variables != nil && strings.TrimSpace(*body.GraphQL.Variables) != "" {
			varsRaw := StripComments(*body.GraphQL.Variables)
			varsInterp := vars.Interpolate(varsRaw, scopeVars, interpOpts)

			var parsedVars map[string]any
			if err := json.Unmarshal([]byte(varsInterp), &parsedVars); err != nil {
				return nil, "", 0, fmt.Errorf("Failed to parse GraphQL variables: %w", err)
			}
			payload["variables"] = parsedVars
		}

		data, err = json.Marshal(payload)
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to marshal GraphQL body: %w", err)
		}
		contentType = "application/json"

	case model.BodyFormURLEncoded:
		values := url.Values{}
		for _, kv := range body.FormURLEncoded {
			if kv.Live() {
				name := vars.Interpolate(kv.NameOrEmpty(), scopeVars, interpOpts)
				val := vars.Interpolate(kv.ValueOrEmpty(), scopeVars, interpOpts)
				values.Add(name, val)
			}
		}
		data = []byte(values.Encode())
		contentType = "application/x-www-form-urlencoded"

	case model.BodyMultipartForm:
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Check if user already supplied boundary in explicit Content-Type
		if explicitContentType != "" {
			if idx := strings.Index(explicitContentType, "boundary="); idx != -1 {
				boundary := explicitContentType[idx+9:]
				if endIdx := strings.Index(boundary, ";"); endIdx != -1 {
					boundary = boundary[:endIdx]
				}
				boundary = strings.Trim(boundary, `"'`)
				if boundary != "" {
					_ = writer.SetBoundary(boundary)
				}
			}
		}

		for _, p := range body.MultipartForm {
			if !p.Enabled {
				continue
			}
			nameRaw := ""
			if p.Name != nil {
				nameRaw = *p.Name
			}
			name := vars.Interpolate(nameRaw, scopeVars, interpOpts)

			valRaw := ""
			if p.Value != nil {
				valRaw = *p.Value
			}

			if p.Type == model.MultipartFile {
				// File part
				filePath := vars.Interpolate(valRaw, scopeVars, interpOpts)
				if !filepath.IsAbs(filePath) && basePath != "" {
					filePath = filepath.Join(basePath, filePath)
				}
				fileBytes, err := os.ReadFile(filePath)
				if err != nil {
					return nil, "", 0, fmt.Errorf("read multipart file %s: %w", filePath, err)
				}

				h := make(textproto.MIMEHeader)
				filename := filepath.Base(filePath)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, name, filename))

				partContentType := "application/octet-stream"
				if p.ContentType != nil && *p.ContentType != "" {
					partContentType = *p.ContentType
				}
				h.Set("Content-Type", partContentType)

				part, err := writer.CreatePart(h)
				if err != nil {
					return nil, "", 0, fmt.Errorf("create multipart part: %w", err)
				}
				if _, err := part.Write(fileBytes); err != nil {
					return nil, "", 0, fmt.Errorf("write multipart part: %w", err)
				}
			} else {
				// Text part
				val := vars.Interpolate(valRaw, scopeVars, interpOpts)
				if err := writer.WriteField(name, val); err != nil {
					return nil, "", 0, fmt.Errorf("write multipart field %s: %w", name, err)
				}
			}
		}

		if err := writer.Close(); err != nil {
			return nil, "", 0, fmt.Errorf("close multipart writer: %w", err)
		}
		data = buf.Bytes()
		contentType = writer.FormDataContentType()

	case model.BodyFile:
		selected := body.SelectedFile()
		if selected == nil || selected.FilePath == nil {
			return nil, "", 0, nil
		}
		filePath := vars.Interpolate(*selected.FilePath, scopeVars, interpOpts)
		if !filepath.IsAbs(filePath) && basePath != "" {
			filePath = filepath.Join(basePath, filePath)
		}
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			// Per Bruno request-builder spec: error reading file logs and continues without body
			return nil, "", 0, nil
		}
		data = fileBytes
		if selected.ContentType != nil && *selected.ContentType != "" {
			contentType = *selected.ContentType
		} else {
			contentType = "application/octet-stream"
		}
	}

	// Precedence: explicit Content-Type from headers always wins!
	if explicitContentType != "" {
		contentType = explicitContentType
	}

	contentLength = int64(len(data))
	return data, contentType, contentLength, nil
}
