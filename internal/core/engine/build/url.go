package build

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"fearless/internal/core/model"
)

var odataSegmentRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+\([^)]*\)$`)

// SynthesizeURL builds, substitutes path parameters into, and normalizes a target URL.
func SynthesizeURL(rawURL string, pathParams []model.Param, queryParams []model.Param, encodeURL bool) (*url.URL, string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, "", fmt.Errorf("URL cannot be empty")
	}

	// 1. Ensure scheme (default to http:// if missing)
	if !strings.Contains(rawURL, "://") && !strings.HasPrefix(rawURL, "{{") {
		rawURL = "http://" + rawURL
	}

	// 2. Separate query string and fragment to preserve raw query
	var rawQuery string
	var rawFragment string
	baseURL := rawURL

	if idx := strings.Index(baseURL, "#"); idx != -1 {
		rawFragment = baseURL[idx:]
		baseURL = baseURL[:idx]
	}

	if idx := strings.Index(baseURL, "?"); idx != -1 {
		rawQuery = baseURL[idx+1:]
		baseURL = baseURL[:idx]
	}

	// Build map of enabled, non-empty path parameters
	paramMap := make(map[string]string)
	for _, p := range pathParams {
		if p.Type == model.ParamPath && p.Live() {
			name := strings.TrimPrefix(strings.TrimSpace(p.NameOrEmpty()), ":")
			val := p.ValueOrEmpty()
			if name != "" && val != "" {
				paramMap[name] = val
			}
		}
	}

	// 3. Substitute path parameters in pathname
	schemeEnd := strings.Index(baseURL, "://")
	var prefix, pathPart string
	if schemeEnd != -1 {
		prefix = baseURL[:schemeEnd+3]
		rem := baseURL[schemeEnd+3:]
		firstSlash := strings.Index(rem, "/")
		if firstSlash != -1 {
			prefix += rem[:firstSlash]
			pathPart = rem[firstSlash:]
		} else {
			prefix += rem
			pathPart = ""
		}
	} else {
		pathPart = baseURL
	}

	if pathPart != "" && len(paramMap) > 0 {
		segments := strings.Split(pathPart, "/")
		for i, seg := range segments {
			if strings.HasPrefix(seg, ":") {
				name := seg[1:]
				if val, ok := paramMap[name]; ok {
					segments[i] = val
				}
			} else if odataSegmentRe.MatchString(seg) {
				// OData segment: e.g. Orders(:orderId) or Users(key=:id)
				for name, val := range paramMap {
					placeholder := ":" + name
					seg = strings.ReplaceAll(seg, placeholder, val)
				}
				segments[i] = seg
			}
		}
		pathPart = strings.Join(segments, "/")
	}

	// Reassemble path
	assembledURL := prefix + pathPart

	// 4. Append / merge query parameters
	// If queryParams slice contains live query params not present in rawQuery, add them
	var queryParts []string
	if rawQuery != "" {
		queryParts = append(queryParts, rawQuery)
	}

	for _, p := range queryParams {
		if p.Type == model.ParamQuery && p.Live() {
			name := strings.TrimSpace(p.NameOrEmpty())
			if name != "" {
				val := p.ValueOrEmpty()
				queryEntry := url.QueryEscape(name) + "=" + url.QueryEscape(val)
				// Check if already in rawQuery to prevent duplication
				if !strings.Contains(rawQuery, url.QueryEscape(name)+"=") &&
					!strings.Contains(rawQuery, name+"=") {
					queryParts = append(queryParts, queryEntry)
				}
			}
		}
	}

	if len(queryParts) > 0 {
		assembledURL += "?" + strings.Join(queryParts, "&")
	}
	if rawFragment != "" {
		assembledURL += rawFragment
	}

	// 5. Parse into *url.URL
	parsedURL, err := url.Parse(assembledURL)
	if err != nil {
		return nil, assembledURL, fmt.Errorf("Invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, assembledURL, fmt.Errorf("Invalid URL format: missing scheme or host in %q", assembledURL)
	}

	// 6. Optional URL encoding
	if encodeURL {
		parsedURL.RawPath = parsedURL.EscapedPath()
	}

	return parsedURL, assembledURL, nil
}
