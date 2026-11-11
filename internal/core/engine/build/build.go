package build

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// DefaultBuilder provides the default request preparation and synthesis implementation.
type DefaultBuilder struct{}

// New creates a new DefaultBuilder.
func New() *DefaultBuilder {
	return &DefaultBuilder{}
}

// Build executes the request preparation pipeline matching flows/request-lifecycle.md.
func (b *DefaultBuilder) Build(ctx context.Context, in BuildInput) (*BuiltRequest, error) {
	return Build(ctx, in)
}

// Build compiles and synthesizes an executable request from a collection item and variables.
func Build(ctx context.Context, in BuildInput) (*BuiltRequest, error) {
	if in.Item == nil {
		return nil, fmt.Errorf("build: nil item")
	}

	var unresolved []string
	onUnresolved := func(name string) {
		unresolved = append(unresolved, name)
	}

	interpOpts := vars.InterpolateOptions{
		EscapeJSON:   false,
		OnUnresolved: onUnresolved,
	}

	scopeVars := in.Variables
	if scopeVars == nil {
		scopeVars = make(map[string]any)
	}

	// 1. Merge tree: collection -> folders -> request
	merged := MergeRequest(in.Collection, in.Item)
	req := merged.Request

	// 2. Determine base directory for resolving relative file bodies
	basePath := in.CollectionPath
	if basePath == "" && in.Collection != nil {
		basePath = in.Collection.Path
	}

	// 3. Process Auth headers
	authHeaders := resolveAuthHeaders(merged.Auth, scopeVars, interpOpts)

	// 4. Interpolate and assemble Headers
	headers := make(http.Header)
	var headerEntries []HeaderEntry
	var explicitContentType string

	addHeader := func(name, value string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if strings.EqualFold(name, "content-type") {
			explicitContentType = value
		}
		headers.Add(name, value)
		headerEntries = append(headerEntries, HeaderEntry{Name: name, Value: value})
	}

	// Add merged headers
	for _, kv := range merged.Headers {
		if !kv.Live() {
			continue
		}
		name := vars.Interpolate(kv.NameOrEmpty(), scopeVars, interpOpts)
		val := vars.Interpolate(kv.ValueOrEmpty(), scopeVars, interpOpts)
		addHeader(name, val)
	}

	// Add auth headers
	for _, ah := range authHeaders {
		addHeader(ah.Name, ah.Value)
	}

	// 5. Interpolate URL
	rawURL := vars.Interpolate(req.URL, scopeVars, interpOpts)

	// Interpolate path parameters
	var interpolatedPathParams []model.Param
	for _, p := range req.Params {
		if p.Type == model.ParamPath && p.Live() {
			cp := p
			name := vars.Interpolate(p.NameOrEmpty(), scopeVars, interpOpts)
			val := vars.Interpolate(p.ValueOrEmpty(), scopeVars, interpOpts)
			cp.Name = &name
			cp.Value = &val
			interpolatedPathParams = append(interpolatedPathParams, cp)
		}
	}

	// Interpolate query parameters
	var interpolatedQueryParams []model.Param
	for _, p := range req.Params {
		if p.Type == model.ParamQuery && p.Live() {
			cp := p
			name := vars.Interpolate(p.NameOrEmpty(), scopeVars, interpOpts)
			val := vars.Interpolate(p.ValueOrEmpty(), scopeVars, interpOpts)
			cp.Name = &name
			cp.Value = &val
			interpolatedQueryParams = append(interpolatedQueryParams, cp)
		}
	}

	// Handle API Key query placement if configured
	if merged.Auth != nil && merged.Auth.Mode == model.AuthAPIKey {
		placement := string(merged.Auth.Placement)
		if strings.EqualFold(placement, "query") || strings.EqualFold(placement, "params") || strings.EqualFold(placement, "queryparams") || merged.Auth.Placement == model.APIKeyQueryParams {
			rawKey := ""
			if merged.Auth.Key != nil {
				rawKey = *merged.Auth.Key
			}
			rawVal := ""
			if merged.Auth.Value != nil {
				rawVal = *merged.Auth.Value
			}
			kName := vars.Interpolate(rawKey, scopeVars, interpOpts)
			kVal := vars.Interpolate(rawVal, scopeVars, interpOpts)
			if kName != "" {
				interpolatedQueryParams = append(interpolatedQueryParams, model.Param{
					Type: model.ParamQuery,
					KeyValue: model.KeyValue{
						Name:    &kName,
						Value:   &kVal,
						Enabled: true,
					},
				})
			}
		}
	}

	// 6. Synthesize URL
	encodeURLSetting := false
	if in.Settings != nil && in.Settings.EncodeURL != nil {
		encodeURLSetting = *in.Settings.EncodeURL
	} else if merged.Settings.EncodeURL != nil {
		encodeURLSetting = *merged.Settings.EncodeURL
	}

	parsedURL, finalURLStr, err := SynthesizeURL(rawURL, interpolatedPathParams, interpolatedQueryParams, encodeURLSetting)
	if err != nil {
		return nil, err
	}

	// 7. Render Body
	bodyBytes, determinedContentType, contentLength, err := RenderBody(
		req.Body,
		scopeVars,
		basePath,
		explicitContentType,
		onUnresolved,
	)
	if err != nil {
		return nil, err
	}

	// 8. Finalize Content-Type & Content-Length headers
	if determinedContentType != "" {
		headers.Set("Content-Type", determinedContentType)
	}

	var bodyReader io.ReadCloser
	if bodyBytes != nil {
		bodyReader = io.NopCloser(bytes.NewReader(bodyBytes))
		headers.Set("Content-Length", fmt.Sprintf("%d", contentLength))
	}

	// 9. Construct net/http.Request
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	var httpReqBody io.Reader
	if bodyBytes != nil {
		httpReqBody = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, finalURLStr, httpReqBody)
	if err != nil {
		return nil, fmt.Errorf("build: create http.Request: %w", err)
	}

	httpReq.Header = headers.Clone()
	if hostVal := headers.Get("Host"); hostVal != "" {
		httpReq.Host = hostVal
	}
	httpReq.ContentLength = contentLength

	return &BuiltRequest{
		Method:         method,
		URL:            parsedURL,
		RawURL:         finalURLStr,
		Headers:        headers,
		HeaderEntries:  headerEntries,
		Body:           bodyBytes,
		BodyReader:     bodyReader,
		ContentLength:  contentLength,
		ContentType:    determinedContentType,
		UnresolvedVars: unresolved,
		HTTPRequest:    httpReq,
	}, nil
}

func resolveAuthHeaders(auth *model.Auth, scopeVars map[string]any, interpOpts vars.InterpolateOptions) []HeaderEntry {
	if auth == nil || auth.Mode == "" || auth.Mode == model.AuthNone || auth.Mode == model.AuthInherit {
		return nil
	}

	var entries []HeaderEntry

	strOrEmpty := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	switch auth.Mode {
	case model.AuthBasic:
		user := vars.Interpolate(strOrEmpty(auth.Username), scopeVars, interpOpts)
		pass := vars.Interpolate(strOrEmpty(auth.Password), scopeVars, interpOpts)
		authStr := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		entries = append(entries, HeaderEntry{
			Name:  "Authorization",
			Value: "Basic " + authStr,
		})

	case model.AuthBearer:
		token := vars.Interpolate(strOrEmpty(auth.Token), scopeVars, interpOpts)
		entries = append(entries, HeaderEntry{
			Name:  "Authorization",
			Value: "Bearer " + token,
		})

	case model.AuthAPIKey:
		placement := string(auth.Placement)
		if placement == "" || strings.EqualFold(placement, "header") || auth.Placement == model.APIKeyHeader {
			name := vars.Interpolate(strOrEmpty(auth.Key), scopeVars, interpOpts)
			val := vars.Interpolate(strOrEmpty(auth.Value), scopeVars, interpOpts)
			if name != "" {
				entries = append(entries, HeaderEntry{Name: name, Value: val})
			}
		}

	case model.AuthWSSE:
		user := vars.Interpolate(strOrEmpty(auth.Username), scopeVars, interpOpts)
		pass := vars.Interpolate(strOrEmpty(auth.Password), scopeVars, interpOpts)
		nonce := fmt.Sprintf("%d", time.Now().UnixNano())
		created := time.Now().UTC().Format(time.RFC3339)

		hasher := sha1.New()
		hasher.Write([]byte(nonce + created + pass))
		digest := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(hasher.Sum(nil))))

		wsseHeader := fmt.Sprintf(
			`UsernameToken Username="%s", PasswordDigest="%s", Nonce="%s", Created="%s"`,
			user, digest, base64.StdEncoding.EncodeToString([]byte(nonce)), created,
		)
		entries = append(entries, HeaderEntry{
			Name:  "X-WSSE",
			Value: wsseHeader,
		})
		entries = append(entries, HeaderEntry{
			Name:  "Authorization",
			Value: `WSSE profile="UsernameToken"`,
		})
	}

	return entries
}
