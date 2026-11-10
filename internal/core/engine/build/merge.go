package build

import (
	"strings"

	"fearless/internal/core/model"
)

// MergedRequest collects the merged headers, auth, params, and settings for an item
// by walking up the collection tree hierarchy (collection -> folders -> item).
type MergedRequest struct {
	Request  *model.Request
	Headers  []model.KeyValue
	Auth     *model.Auth
	Settings model.ItemSettings
}

// MergeRequest prepares the effective request configuration by layering folder and collection settings.
func MergeRequest(col *model.Collection, item *model.Item) MergedRequest {
	if item == nil {
		return MergedRequest{}
	}

	effectiveReq := item.Effective()
	if effectiveReq == nil {
		effectiveReq = &model.Request{}
	}

	var ancestors []*model.Item
	if col != nil {
		ancestors = col.Ancestors(item.UID)
	}

	// 1. Merge Headers
	// Order: collection -> ancestor folders -> request
	// Case-insensitive header matching (later wins), preserving duplicates with different casing or same name
	var mergedHeaders []model.KeyValue
	headerIndex := make(map[string]int)

	appendOrUpdateHeader := func(kv model.KeyValue) {
		if !kv.Live() {
			return
		}
		name := strings.TrimSpace(kv.NameOrEmpty())
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if idx, exists := headerIndex[key]; exists {
			mergedHeaders[idx] = kv
		} else {
			headerIndex[key] = len(mergedHeaders)
			mergedHeaders = append(mergedHeaders, kv)
		}
	}

	// Ancestor folders headers
	for _, folder := range ancestors {
		for _, h := range folder.Headers {
			appendOrUpdateHeader(h)
		}
	}

	// Request headers
	for _, h := range effectiveReq.Headers {
		appendOrUpdateHeader(h)
	}

	// 2. Merge Auth
	// Precedence: Collection -> ancestor folders -> request
	var effectiveAuth *model.Auth
	for _, folder := range ancestors {
		if folder.Auth != nil && folder.Auth.Mode != "" && folder.Auth.Mode != model.AuthInherit {
			effectiveAuth = folder.Auth
		}
	}

	if effectiveReq.Auth != nil && effectiveReq.Auth.Mode != "" && effectiveReq.Auth.Mode != model.AuthInherit {
		effectiveAuth = effectiveReq.Auth
	}

	// 3. Settings
	settings := effectiveReq.Settings

	return MergedRequest{
		Request:  effectiveReq,
		Headers:  mergedHeaders,
		Auth:     effectiveAuth,
		Settings: settings,
	}
}
