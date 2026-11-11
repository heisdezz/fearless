package oauth2

import (
	"strings"
)

// Token placement values.
//
// Bruno's OAuth2 editor offers these, with `header` as the default:
//
//	header      Authorization: <prefix><token>
//	queryparams ?<key>=<token>
//	body        appended to a form body (not supported here; see Apply)
const (
	PlacementHeader      = "header"
	PlacementQueryParams = "queryparams"
	PlacementBody        = "body"

	// DefaultHeaderPrefix is what a token's placement uses when the prefix is
	// unset, and what the token_type implies when the server reports Bearer.
	DefaultHeaderPrefix = "Bearer"
	// DefaultQueryKey is the query parameter name used when none is configured.
	DefaultQueryKey = "access_token"
)

// Placement is where a token is attached to a request.
type Placement struct {
	// In is one of the Placement values.
	In string
	// HeaderPrefix is the scheme, e.g. "Bearer".
	HeaderPrefix string
	// QueryKey is the parameter name for the query placement.
	QueryKey string
}

// PlacementFrom reads the placement from the model's fields.
//
// Source: bruno-electron/src/utils/oauth2.js (tokenPlacement, tokenHeaderPrefix,
// tokenQueryKey)
func PlacementFrom(tokenPlacement, tokenHeaderPrefix, tokenQueryKey, tokenType string) Placement {
	p := Placement{
		In:           tokenPlacement,
		HeaderPrefix: tokenHeaderPrefix,
		QueryKey:     tokenQueryKey,
	}

	if p.In == "" {
		p.In = PlacementHeader
	}
	if p.QueryKey == "" {
		p.QueryKey = DefaultQueryKey
	}
	if p.HeaderPrefix == "" {
		// The server's token_type is the honest default, because a server issuing
		// DPoP or MAC expects that scheme rather than Bearer.
		p.HeaderPrefix = tokenType
	}
	if p.HeaderPrefix == "" {
		p.HeaderPrefix = DefaultHeaderPrefix
	}
	return p
}

// Apply attaches the token, returning a header or a query pair.
//
// The caller decides what to do with the result: an empty Header means nothing
// goes in the header, and an empty QueryKey means nothing goes in the query.
//
// The `body` placement is reported as unsupported rather than silently doing
// nothing. Appending a token to a request body means rewriting an
// already-serialised payload — including a multipart one — which is a transport
// concern, and it is rare enough that a clear failure is better than a wrong
// request.
func (p Placement) Apply(token string, tokenType string) (header string, queryKey string) {
	switch p.In {
	case PlacementQueryParams:
		return "", p.QueryKey
	case PlacementBody:
		return "", ""
	default:
		prefix := strings.TrimSpace(p.HeaderPrefix)
		if prefix == "" {
			prefix = strings.TrimSpace(tokenType)
		}
		if prefix == "" {
			prefix = DefaultHeaderPrefix
		}
		// An explicitly blank prefix means a bare token, so it must not acquire a
		// leading space. Distinguishing this from "unset" matters because some
		// servers expect the raw token.
		if p.HeaderPrefix != "" && strings.TrimSpace(p.HeaderPrefix) == "" {
			return token, ""
		}
		return prefix + " " + token, ""
	}
}

// SupportsBody reports whether the body placement can be applied.
//
// It exists so a caller can validate configuration up front instead of failing
// at send time.
func (p Placement) SupportsBody() bool { return p.In != PlacementBody }
