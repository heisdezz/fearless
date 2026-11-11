package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"fearless/internal/core/model"
)

// edgegridProvider implements Akamai EdgeGrid EG1-HMAC-SHA256 signing.
//
// The signature covers a tab-joined "data to sign":
//
//	METHOD \t scheme \t host \t path+query \t signed-headers \t content-hash \t auth-header
//
// followed by a double HMAC-SHA256: the signing key is
// base64(HMAC(timestamp, clientSecret)) *as a string*, and the signature is
// base64(HMAC(dataToSign, thatString)). Getting the second step wrong — using the
// raw digest as the key instead of its base64 text — is the usual bug, so the
// intermediate values are exposed in edgeGridSignResult for tests.
//
// Source: bruno-requests/src/auth/edgegrid-helper.js signEdgeGridRequest
// Source: bruno-common/src/utils/edgegrid.ts
// Spec: https://techdocs.akamai.com/developer/docs/authenticate-with-edgegrid
type edgegridProvider struct{}

func (edgegridProvider) Mode() model.AuthMode { return model.AuthEdgeGrid }

const (
	// edgeGridDefaultMaxBodySize is Akamai's 128 KB body-hash window.
	//
	// Source: bruno-common/src/utils/edgegrid.ts MAX_BODY_SIZE_DEFAULT
	edgeGridDefaultMaxBodySize = 131072

	// edgeGridTimestampLayout is `YYYYMMDDTHH:MM:SS+0000`.
	//
	// Not RFC 3339: Akamai expects the literal `+0000` offset rather than `Z`.
	// Source: bruno-common/src/utils/edgegrid.ts makeEdgeGridTimestamp
	edgeGridTimestampLayout = "20060102T15:04:05-0700"

	edgeGridScheme = "EG1-HMAC-SHA256"
)

// Prepare signs the request.
//
// All three credentials are required; a partial configuration contributes
// nothing and the request goes out unsigned, matching Bruno's validation (which
// throws and then rejects the request — the harness treats it as a failed send).
func (edgegridProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	accessToken := interpolate(deps, model.DerefString(cfg.AccessToken))
	clientToken := interpolate(deps, model.DerefString(cfg.EdgeGridClientToken))
	clientSecret := interpolate(deps, model.DerefString(cfg.EdgeGridClientSecret))

	if !edgeGridPresent(accessToken) || !edgeGridPresent(clientToken) || !edgeGridPresent(clientSecret) {
		return Prepared{}, nil
	}
	if deps.Request == nil {
		return Prepared{}, errNoRequestForSigning
	}

	timestamp := interpolate(deps, model.DerefString(cfg.Timestamp))
	if !edgeGridPresent(timestamp) {
		// Akamai compares this against server time with a tolerance, so it must
		// be real time in UTC rather than a monotonic-clock reading.
		timestamp = edgeGridNow(deps).UTC().Format(edgeGridTimestampLayout)
	}
	nonce := interpolate(deps, model.DerefString(cfg.Nonce))
	if !edgeGridPresent(nonce) {
		nonce = edgeGridNonce(deps)
	}

	maxBodySize := edgeGridDefaultMaxBodySize
	if cfg.EdgeGridMaxBodySize != nil && *cfg.EdgeGridMaxBodySize > 0 {
		maxBodySize = *cfg.EdgeGridMaxBodySize
	}

	res, err := edgeGridSign(edgeGridInput{
		Method:        deps.Request.Method,
		RequestURL:    deps.Request.URL,
		Headers:       deps.Request.Header,
		Body:          deps.Body,
		AccessToken:   accessToken,
		ClientToken:   clientToken,
		ClientSecret:  clientSecret,
		Timestamp:     timestamp,
		Nonce:         nonce,
		HeadersToSign: interpolate(deps, model.DerefString(cfg.EdgeGridHeadersToSign)),
		BaseURL:       interpolate(deps, model.DerefString(cfg.EdgeGridBaseURL)),
		MaxBodySize:   maxBodySize,
	})
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Headers: []Header{{Name: "Authorization", Value: res.Authorization}}}, nil
}

// edgeGridPresent reports whether a credential is usable.
//
// The literal string "undefined" counts as absent: Bruno's check exists because
// an unresolved template can stringify to that, and signing it would produce a
// header no server accepts.
//
// Source: bruno-common/src/utils/edgegrid.ts isStrPresent
func edgeGridPresent(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed != "" && trimmed != "undefined"
}

// edgeGridNow returns the current time, injectable for deterministic tests.
func edgeGridNow(deps Deps) time.Time {
	return time.Unix(nowUnix(deps), 0).UTC()
}

// edgeGridNonce returns a UUID v4, which is what Akamai's reference client sends.
//
// The version and variant bits are set so the value is a valid UUID; a random
// hex string would also work on the wire, but matching the reference keeps
// captured traffic comparable.
func edgeGridNonce(deps Deps) string {
	var b [16]byte
	copy(b[:], randomBytes(deps, 16))
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// edgeGridInput is everything needed to sign.
type edgeGridInput struct {
	Method        string
	RequestURL    *url.URL
	Headers       map[string][]string
	Body          []byte
	AccessToken   string
	ClientToken   string
	ClientSecret  string
	Timestamp     string
	Nonce         string
	HeadersToSign string
	BaseURL       string
	MaxBodySize   int
}

// edgeGridSignResult keeps the intermediates so tests can check each step.
type edgeGridSignResult struct {
	// Authorization is the complete header value.
	Authorization string
	// AuthHeader is the prefix, which is also the last field of the data to sign.
	AuthHeader string
	// DataToSign is the tab-joined string.
	DataToSign string
	// SigningKey is base64(HMAC(timestamp, clientSecret)).
	SigningKey string
	// Signature is base64(HMAC(dataToSign, SigningKey)).
	Signature string
}

// edgeGridSign produces the Authorization header.
func edgeGridSign(in edgeGridInput) (edgeGridSignResult, error) {
	if in.RequestURL == nil {
		return edgeGridSignResult{}, errNoRequestForSigning
	}

	scheme, host, pathAndQuery, err := edgeGridURLToSign(in.RequestURL, in.BaseURL)
	if err != nil {
		return edgeGridSignResult{}, err
	}

	authHeader := fmt.Sprintf("%s client_token=%s;access_token=%s;timestamp=%s;nonce=%s;",
		edgeGridScheme, in.ClientToken, in.AccessToken, in.Timestamp, in.Nonce)

	signedHeaders := edgeGridCanonicalizeHeaders(in.HeadersToSign, in.Headers)
	contentHash := edgeGridContentHash(in.Method, in.Body, in.MaxBodySize)

	dataToSign := strings.Join([]string{
		strings.ToUpper(in.Method),
		scheme,
		host,
		pathAndQuery,
		signedHeaders,
		contentHash,
		authHeader,
	}, "\t")

	signingKey := edgeGridHMAC(in.Timestamp, in.ClientSecret)
	signature := edgeGridHMAC(dataToSign, signingKey)

	return edgeGridSignResult{
		Authorization: authHeader + "signature=" + signature,
		AuthHeader:    authHeader,
		DataToSign:    dataToSign,
		SigningKey:    signingKey,
		Signature:     signature,
	}, nil
}

// edgeGridSchemePrefix matches a leading `scheme://`.
var edgeGridSchemePrefix = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

// edgeGridURLToSign returns the scheme, host and path+query that get signed.
//
// A configured base URL replaces the request's scheme and host while keeping the
// request's path and query, which is how a collection targets the Akamai staging
// network while keeping friendly URLs in the request.
//
// A scheme-less base URL ("localhost:6000") is completed with the request's
// scheme. Without that, `url.Parse` reads "localhost:" as a scheme and leaves an
// empty host — a silent signature mismatch rather than an error.
func edgeGridURLToSign(requestURL *url.URL, baseURL string) (scheme, host, pathAndQuery string, err error) {
	target := requestURL

	if trimmed := strings.TrimSpace(baseURL); trimmed != "" {
		normalized := trimmed
		if !edgeGridSchemePrefix.MatchString(normalized) {
			normalized = requestURL.Scheme + "://" + normalized
		}
		base, perr := url.Parse(normalized)
		if perr != nil {
			return "", "", "", fmt.Errorf("auth: EdgeGrid baseURL %q: %w", baseURL, perr)
		}
		if base.Host == "" {
			return "", "", "", fmt.Errorf("auth: EdgeGrid baseURL %q has no host", baseURL)
		}
		// Keep the request's path and query; take only authority from the base.
		copied := *requestURL
		copied.Scheme = base.Scheme
		copied.Host = base.Host
		target = &copied
	}

	scheme = strings.ToLower(target.Scheme)
	host = target.Host
	if host == "" {
		return "", "", "", fmt.Errorf("auth: EdgeGrid cannot sign a URL with no host")
	}

	pathAndQuery = target.EscapedPath()
	if pathAndQuery == "" {
		pathAndQuery = "/"
	}
	if target.RawQuery != "" {
		pathAndQuery += "?" + target.RawQuery
	}
	return scheme, host, pathAndQuery, nil
}

// edgeGridCanonicalizeHeaders renders the headers named in HeadersToSign.
//
// Format is `name:value` per header, tab-joined, name lowercased, value trimmed
// with internal whitespace collapsed to single spaces.
//
// Note the ordering rule: the *configuration* order is kept, not alphabetical
// order. An alphabetical sort would produce a different signature from Akamai's.
// Headers listed but absent from the request are skipped, so a signature never
// claims to cover something that is not sent.
//
// Source: bruno-common/src/utils/edgegrid.ts canonicalizeHeaders
func edgeGridCanonicalizeHeaders(headersToSign string, headers map[string][]string) string {
	if !edgeGridPresent(headersToSign) {
		return ""
	}

	lookup := make(map[string]string, len(headers))
	for name, values := range headers {
		if len(values) == 0 {
			continue
		}
		// HTTP header names are case-insensitive, so `Accept` and `accept` are the
		// same header. net/http only guarantees one canonical spelling, but a
		// directly-assigned map may hold both; joining keeps the value complete
		// rather than letting iteration order drop one.
		lower := strings.ToLower(name)
		joined := strings.Join(values, ", ")
		if existing, exists := lookup[lower]; exists {
			lookup[lower] = existing + ", " + joined
			continue
		}
		lookup[lower] = joined
	}

	var parts []string
	for _, name := range strings.Split(headersToSign, ",") {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		value, ok := lookup[key]
		if !ok {
			continue
		}
		parts = append(parts, key+":"+edgeGridCollapseWhitespace(strings.TrimSpace(value)))
	}
	return strings.Join(parts, "\t")
}

// edgeGridWhitespace matches a run of whitespace.
var edgeGridWhitespace = regexp.MustCompile(`\s+`)

// edgeGridCollapseWhitespace replaces whitespace runs with a single space.
func edgeGridCollapseWhitespace(s string) string {
	return edgeGridWhitespace.ReplaceAllString(s, " ")
}

// edgeGridContentHash hashes a POST body, truncated to MaxBodySize.
//
// Only POST is hashed — Akamai's rule, not an oversight. A GET with a body gets an
// empty hash, and every non-POST method likewise.
//
// Source: bruno-common/src/utils/edgegrid.ts makeContentHash
func edgeGridContentHash(method string, body []byte, maxBodySize int) string {
	if !strings.EqualFold(method, "POST") || len(body) == 0 {
		return ""
	}
	limit := maxBodySize
	if limit <= 0 {
		limit = edgeGridDefaultMaxBodySize
	}
	if len(body) > limit {
		body = body[:limit]
	}
	sum := sha256.Sum256(body)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// edgeGridHMAC returns base64(HMAC-SHA256(data, key)) with both sides as UTF-8
// text.
//
// The signing key step passes a base64 *string* as the key, not the decoded
// bytes; that is what Akamai's reference client does and it changes the result.
func edgeGridHMAC(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
