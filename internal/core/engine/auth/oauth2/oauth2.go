// Package oauth2 obtains, caches and refreshes OAuth2 tokens.
//
// It is a separate package from the auth providers because the flow is
// stateful in a way the others are not: it needs a token endpoint, a cache, a
// refresh policy and — for the authorization-code grant — a browser and a
// loopback listener. Putting that inside `internal/core/engine/auth` would drag
// the platform interfaces into every provider.
//
// Four grants are supported: authorization code (with optional PKCE), client
// credentials, password, and device code. Bruno's own helper implements only the
// last two (bruno-requests/src/auth/oauth2-helper.ts); the authorization-code
// flow lives in bruno-electron/src/utils/oauth2.js and is reproduced from there.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts
// Source: bruno-electron/src/utils/oauth2.js
package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"fearless/internal/core/model"
)

// Errors reported by the token machinery.
var (
	// ErrUnsupportedGrant reports a grant this package does not implement.
	ErrUnsupportedGrant = errors.New("oauth2: unsupported grant type")

	// ErrNoAccessToken reports a token response without one, which is a server
	// error rather than a client one.
	ErrNoAccessToken = errors.New("oauth2: no access token in response")

	// ErrStateMismatch reports a callback whose state does not match the one sent,
	// which is the authorization-code injection defence.
	ErrStateMismatch = errors.New("oauth2: state mismatch in callback")

	// ErrMissingConfig reports a required field that was not set.
	ErrMissingConfig = errors.New("oauth2: missing required configuration")
)

// Token is a token response plus the bookkeeping the cache needs.
type Token struct {
	AccessToken string `json:"access_token"`
	// TokenType is the response's token_type, usually "Bearer".
	TokenType string `json:"token_type,omitempty"`
	// RefreshToken is used to obtain a new access token.
	RefreshToken string `json:"refresh_token,omitempty"`
	// IDToken is the OIDC identity token, selected by tokenSource = "id_token".
	IDToken string `json:"id_token,omitempty"`
	// Scope is what the server granted, which may be narrower than requested.
	Scope string `json:"scope,omitempty"`
	// ExpiresIn is the lifetime the server reported, in seconds.
	ExpiresIn int64 `json:"expires_in,omitempty"`
	// CreatedAt is when the token was obtained, in Unix milliseconds. It is
	// stored because ExpiresIn alone cannot answer "is it still valid".
	//
	// Milliseconds rather than seconds matches Bruno's cache, so an imported
	// credential file keeps working.
	CreatedAt int64 `json:"created_at,omitempty"`
	// Raw is the full response, so a token endpoint's extra fields survive.
	Raw map[string]any `json:"-"`
}

// ExpiresAt returns the absolute expiry in Unix milliseconds, or 0 when the
// server gave no lifetime.
func (t Token) ExpiresAt() int64 {
	if t.ExpiresIn <= 0 || t.CreatedAt <= 0 {
		return 0
	}
	return t.CreatedAt + t.ExpiresIn*1000
}

// Expired reports whether the token has expired at now (Unix milliseconds).
//
// A token with no expiry information is treated as **valid**: Bruno does the
// same, and the alternative — refetching on every request for a server that
// omits expires_in — would be worse.
func (t Token) Expired(now int64) bool {
	expiresAt := t.ExpiresAt()
	if expiresAt == 0 {
		return false
	}
	return now > expiresAt
}

// Value returns the token to send, honouring tokenSource.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts getOAuth2Token
func (t Token) Value(tokenSource string) string {
	if strings.EqualFold(tokenSource, "id_token") {
		return t.IDToken
	}
	return t.AccessToken
}

// Cache stores tokens keyed by credentials ID.
//
// The key is the credentials ID rather than the token URL, so two requests can
// share a token only when the user says they are the same credentials. Bruno
// keys by (url, credentialsId); credentialsId alone is equivalent when it is set,
// and the default id covers the single-credentials case.
type Cache interface {
	Get(credentialsID string) (Token, bool)
	Put(credentialsID string, token Token)
	Delete(credentialsID string)
}

// MemoryCache is an in-memory Cache.
//
// It is safe for concurrent use, because two requests to the same collection can
// run at once and both may need the token.
type MemoryCache struct {
	mu     sync.RWMutex
	tokens map[string]Token
}

// NewMemoryCache creates an empty cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{tokens: make(map[string]Token)}
}

// Get implements Cache.
func (c *MemoryCache) Get(credentialsID string) (Token, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	token, ok := c.tokens[credentialsID]
	return token, ok
}

// Put implements Cache.
func (c *MemoryCache) Put(credentialsID string, token Token) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens[credentialsID] = token
}

// Delete implements Cache.
func (c *MemoryCache) Delete(credentialsID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tokens, credentialsID)
}

// Settings is the resolved OAuth2 configuration for one request.
//
// It is a plain struct rather than *model.Auth because the values have already
// been interpolated by the time they reach here, and because every field needs a
// default that the model's pointers express as nil.
type Settings struct {
	GrantType            model.OAuth2GrantType
	AuthorizationURL     string
	AccessTokenURL       string
	RefreshTokenURL      string
	ClientID             string
	ClientSecret         string
	Scope                string
	State                string
	CallbackURL          string
	PKCE                 bool
	CredentialsPlacement string
	CredentialsID        string
	TokenSource          string
	AutoRefreshToken     bool
	AutoFetchToken       bool
	AdditionalParams     *model.OAuth2AdditionalParams
	// Username and Password are the password grant's credentials.
	Username string
	Password string
	// DeviceAuthorizationURL is the device grant's verification endpoint.
	DeviceAuthorizationURL string

	// FetchDisabled turns off both fetching and refreshing, deliberately.
	//
	// It exists because a bool cannot distinguish "unset" from "false": a
	// hand-built Settings with AutoFetchToken left at zero would otherwise either
	// fetch against the caller's wishes, or (if zero meant off) fetch nothing and
	// produce an empty Authorization header with no explanation. SettingsFrom sets
	// it from AutoFetchToken; anyone else who wants fetching off says so here.
	FetchDisabled bool
}

// DefaultCredentialsID names the cache slot when none was configured.
const DefaultCredentialsID = "default"

// SettingsFrom builds Settings from a model.Auth.
//
// Defaults are applied here and nowhere else, so a caller never has to remember
// that an empty credentials placement means the basic auth header.
func SettingsFrom(auth *model.Auth) Settings {
	s := Settings{
		GrantType:            auth.GrantType,
		AuthorizationURL:     model.DerefString(auth.AuthorizationURL),
		AccessTokenURL:       model.DerefString(auth.AccessTokenURL),
		RefreshTokenURL:      model.DerefString(auth.RefreshTokenURL),
		ClientID:             model.DerefString(auth.ClientID),
		ClientSecret:         model.DerefString(auth.ClientSecret),
		Scope:                model.DerefString(auth.Scope),
		State:                model.DerefString(auth.State),
		CallbackURL:          model.DerefString(auth.CallbackURL),
		PKCE:                 model.DerefBool(auth.PKCE, false),
		CredentialsPlacement: model.DerefString(auth.CredentialsPlacement),
		CredentialsID:        model.DerefString(auth.CredentialsID),
		TokenSource:          model.DerefString(auth.TokenSource),
		Username:             model.DerefString(auth.Username),
		Password:             model.DerefString(auth.Password),
		AdditionalParams:     auth.AdditionalParams,
	}

	if s.CredentialsPlacement == "" {
		s.CredentialsPlacement = "basic_auth_header"
	}
	if s.CredentialsID == "" {
		s.CredentialsID = DefaultCredentialsID
	}
	if s.TokenSource == "" {
		s.TokenSource = "access_token"
	}
	// Both default to true: Bruno's UI ships them enabled, and a client that
	// silently stopped fetching tokens would look like an authorization failure.
	s.AutoFetchToken = model.DerefBool(auth.AutoFetchToken, true)
	s.AutoRefreshToken = model.DerefBool(auth.AutoRefreshToken, true)
	s.FetchDisabled = !s.AutoFetchToken && !s.AutoRefreshToken

	return s
}

// applyDefaults fills in the values that have a single sensible fallback.
//
// Called from Token and Fetch so a Settings built by hand behaves the same as one
// from SettingsFrom. It does not touch AutoFetchToken, which cannot be defaulted
// safely — see wantsFetch.
func (s Settings) applyDefaults() Settings {
	if s.CredentialsPlacement == "" {
		s.CredentialsPlacement = "basic_auth_header"
	}
	if s.CredentialsID == "" {
		s.CredentialsID = DefaultCredentialsID
	}
	if s.TokenSource == "" {
		s.TokenSource = "access_token"
	}
	return s
}

// usesBasicAuth reports whether client credentials go in the Authorization header.
func (s Settings) usesBasicAuth() bool {
	return s.CredentialsPlacement == "basic_auth_header"
}

// usesBody reports whether client credentials go in the form body.
func (s Settings) usesBody() bool {
	return s.CredentialsPlacement == "body" || s.CredentialsPlacement == "credentials_body"
}

// Client is fetches tokens.
type Client struct {
	// HTTP performs token requests. When nil, http.DefaultClient is used.
	HTTP *http.Client
	// Cache stores acquired tokens.
	Cache Cache
	// Now returns Unix milliseconds; injectable for expiry tests.
	Now func() int64
	// Authorize obtains an authorization code for the authorization-code grant.
	// It is nil when that grant is unavailable, which is the CLI case.
	Authorize Authorizer
	// PromptDevice asks the user to visit a URL and enter a code, for the device
	// grant. Returning false cancels.
	PromptDevice func(verificationURI, userCode string) bool
}

// Authorizer runs the browser half of the authorization-code grant.
//
// It is an interface so this package does not import internal/platform: the GUI
// supplies one backed by a system browser and a loopback listener, and the CLI
// supplies one that prints the URL and reads the code from stdin.
type Authorizer interface {
	// Authorize opens the URL, waits for the callback and returns the code.
	// expectedState must match the callback's state parameter.
	Authorize(ctx context.Context, authorizeURL, expectedState string) (code string, err error)
}

// now returns the current Unix milliseconds.
func (c *Client) now() int64 {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UnixMilli()
}

// httpClient returns the HTTP client to use.
func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// cache returns the cache to use.
func (c *Client) cache() Cache {
	if c.Cache != nil {
		return c.Cache
	}
	return defaultCache
}

// defaultCache backs a Client with no cache of its own.
var defaultCache = NewMemoryCache()

// Token returns a usable access token for the settings, fetching or refreshing
// as the configuration allows.
//
// Defaults are applied here rather than only in SettingsFrom, so a caller that
// built a Settings by hand gets the same behaviour. The auto-fetch default in
// particular must not depend on the constructor: a zero value would silently
// disable fetching, and the symptom — an empty Authorization header — is far
// from the cause.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts getOAuth2Token
func (c *Client) Token(ctx context.Context, s Settings) (Token, error) {
	if s.GrantType == "" {
		return Token{}, fmt.Errorf("%w: grant type", ErrMissingConfig)
	}

	// An unset credentials ID and auto-fetch flag can only mean "the default",
	// because there is no way to express "explicitly empty" for either.
	if s.CredentialsID == "" {
		s.CredentialsID = DefaultCredentialsID
	}
	if s.TokenSource == "" {
		s.TokenSource = "access_token"
	}

	cached, haveCached := c.cache().Get(s.CredentialsID)

	if haveCached {
		if !cached.Expired(c.now()) {
			return cached, nil
		}
		// Expired. Refreshing is the right first move when the server issued a
		// refresh token, because it avoids the interactive authorization step
		// entirely.
		if s.wantsFetch() {
			if refreshed, err := c.Refresh(ctx, s, cached); err == nil {
				return refreshed, nil
			}
		}
		if s.explicitlyNoFetch() {
			// The user asked not to refetch, so the expired token is what they
			// get; the server's 401 is the honest outcome.
			return cached, nil
		}
		c.cache().Delete(s.CredentialsID)
	}

	if s.explicitlyNoFetch() {
		return Token{}, errors.New("oauth2: no token and automatic fetching is disabled")
	}

	return c.Fetch(ctx, s)
}

// wantsFetch reports whether a fetch or refresh should be attempted.
func (s Settings) wantsFetch() bool { return !s.FetchDisabled }

// explicitlyNoFetch reports a caller who turned fetching off on purpose.
func (s Settings) explicitlyNoFetch() bool { return s.FetchDisabled }

// Fetch obtains a new token using the configured grant.
func (c *Client) Fetch(ctx context.Context, s Settings) (Token, error) {
	s = s.applyDefaults()

	switch s.GrantType {
	case model.GrantClientCredentials:
		return c.fetchClientCredentials(ctx, s)
	case model.GrantPassword:
		return c.fetchPassword(ctx, s)
	case model.GrantAuthorizationCode:
		return c.fetchAuthorizationCode(ctx, s)
	case model.GrantImplicit:
		return Token{}, fmt.Errorf("%w: %s (implicit tokens come from the redirect, not the token endpoint)", ErrUnsupportedGrant, s.GrantType)
	default:
		return Token{}, fmt.Errorf("%w: %q", ErrUnsupportedGrant, s.GrantType)
	}
}

// Refresh exchanges a refresh token for a new access token.
//
// The refresh URL falls back to the access-token URL, which is what the spec
// assumes and what most providers do. A refresh that fails leaves the cache
// untouched, so a transient failure does not destroy a still-usable token.
func (c *Client) Refresh(ctx context.Context, s Settings, token Token) (Token, error) {
	if token.RefreshToken == "" {
		return Token{}, errors.New("oauth2: no refresh token to use")
	}
	s = s.applyDefaults()

	endpoint := s.RefreshTokenURL
	if endpoint == "" {
		endpoint = s.AccessTokenURL
	}
	if endpoint == "" {
		return Token{}, fmt.Errorf("%w: refresh URL", ErrMissingConfig)
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", token.RefreshToken)
	if s.Scope != "" {
		form.Set("scope", s.Scope)
	}

	headers := c.basicHeaders(s)
	if s.AdditionalParams != nil {
		c.applyAdditional(form, &headers, s.AdditionalParams.RefreshTokenRequest)
	}
	c.applyClientCredentials(form, s)

	endpoint = c.withQueryParams(endpoint, s.AdditionalParams, func(p *model.OAuth2AdditionalParams) model.AKVSet {
		return p.RefreshTokenRequest
	})

	refreshed, err := c.postForm(ctx, endpoint, form, headers)
	if err != nil {
		return Token{}, err
	}

	refreshed = c.normalise(refreshed)
	// A refresh response often omits the refresh token, meaning "keep the one you
	// have"; dropping it would make the next refresh impossible.
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	if refreshed.IDToken == "" {
		refreshed.IDToken = token.IDToken
	}
	c.cache().Put(s.CredentialsID, refreshed)
	return refreshed, nil
}

// normalise fills in CreatedAt so expiry arithmetic works.
func (c *Client) normalise(token Token) Token {
	if token.CreatedAt == 0 {
		token.CreatedAt = c.now()
	}
	return token
}

// postForm posts a form and decodes the token response.
//
// A non-2xx response is an error carrying the server's body, because the useful
// detail (invalid_client, invalid_grant) is in the body and dropping it turns
// every failure into "unauthorized".
func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values, headers map[string]string) (Token, error) {
	req, err := buildFormRequest(ctx, endpoint, form, headers)
	if err != nil {
		return Token{}, err
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("oauth2: token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body)
	if err != nil {
		return Token{}, err
	}

	token, decodeErr := decodeToken(body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if decodeErr == nil {
			if message := tokenError(token.Raw); message != "" {
				return Token{}, fmt.Errorf("oauth2: token endpoint returned %d: %s", resp.StatusCode, message)
			}
		}
		return Token{}, fmt.Errorf("oauth2: token endpoint returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}
	if decodeErr != nil {
		return Token{}, decodeErr
	}
	if token.AccessToken == "" {
		return Token{}, ErrNoAccessToken
	}
	return token, nil
}

// decodeToken parses a token response.
//
// The raw map is kept because providers add fields (scope, id_token, a custom
// audience) that a struct would silently discard, and a caller may need them.
func decodeToken(body []byte) (Token, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Token{}, fmt.Errorf("oauth2: token response is not JSON: %w", err)
	}

	token := Token{Raw: raw}
	token.AccessToken, _ = raw["access_token"].(string)
	token.TokenType, _ = raw["token_type"].(string)
	token.RefreshToken, _ = raw["refresh_token"].(string)
	token.IDToken, _ = raw["id_token"].(string)
	token.Scope, _ = raw["scope"].(string)

	// expires_in is a number but some servers send it as a string, which a
	// struct decode would reject outright.
	switch value := raw["expires_in"].(type) {
	case float64:
		token.ExpiresIn = int64(value)
	case string:
		var seconds int64
		if _, err := fmt.Sscan(value, &seconds); err == nil {
			token.ExpiresIn = seconds
		}
	}
	return token, nil
}

// tokenError extracts an OAuth2 error description from a response body.
func tokenError(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	code, _ := raw["error"].(string)
	description, _ := raw["error_description"].(string)

	if code == "" && description == "" {
		return ""
	}
	if description == "" {
		return code
	}
	if code == "" {
		return description
	}
	return code + ": " + description
}

// truncate shortens a string for an error message.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// applyClientCredentials adds the client ID and secret where the placement says.
//
// With the basic-auth placement they go in the Authorization header instead, which
// basicHeaders already set, so this only fills the form for the body placement.
func (c *Client) applyClientCredentials(form url.Values, s Settings) {
	if s.usesBasicAuth() {
		return
	}
	if s.ClientID != "" {
		form.Set("client_id", s.ClientID)
	}
	// An empty secret is omitted rather than sent blank: a public client has no
	// secret, and sending "" makes some servers reject the request.
	if s.ClientSecret != "" {
		form.Set("client_secret", s.ClientSecret)
	}
}

// basicHeaders returns the headers for a token request, including basic auth
// when the placement asks for it.
//
// The map is always initialised, never nil: additional header parameters are
// merged into it afterwards, and writing to a nil map panics.
func (c *Client) basicHeaders(s Settings) map[string]string {
	headers := map[string]string{}
	if s.usesBasicAuth() {
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(s.ClientID+":"+s.ClientSecret))
	}
	return headers
}

// applyAdditional merges configured extra parameters into a token request.
//
// AKVSet groups parameters by destination, so there is no per-parameter `sendIn`
// to interpret: a header goes in the header map, a query parameter in the URL
// (see withQueryParams), and a body value in the form.
//
// Source: bruno-requests/src/auth/oauth2-helper.ts applyAdditionalParameters
func (c *Client) applyAdditional(form url.Values, headers *map[string]string, set model.AKVSet) {
	for _, param := range set.Body {
		if !param.Enabled {
			continue
		}
		if name := model.DerefString(param.Name); name != "" {
			form.Set(name, model.DerefString(param.Value))
		}
	}
	if headers == nil {
		return
	}
	for _, param := range set.Headers {
		if !param.Enabled {
			continue
		}
		if name := model.DerefString(param.Name); name != "" {
			(*headers)[name] = model.DerefString(param.Value)
		}
	}
}

// withQueryParams appends a set's query parameters to an endpoint URL.
func (c *Client) withQueryParams(endpoint string, params *model.OAuth2AdditionalParams, selectSet func(*model.OAuth2AdditionalParams) model.AKVSet) string {
	if params == nil {
		return endpoint
	}
	set := selectSet(params)
	if len(set.QueryParams) == 0 {
		return endpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsed.Query()
	for _, param := range set.QueryParams {
		if !param.Enabled {
			continue
		}
		if name := model.DerefString(param.Name); name != "" {
			query.Set(name, model.DerefString(param.Value))
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// randomBytes returns n cryptographically random bytes.
func randomBytes(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("oauth2: crypto/rand failed: " + err.Error())
	}
	return buf
}

// PKCEPair is a verifier and its S256 challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
	Method    string
}

// NewPKCEPair creates a PKCE verifier and challenge.
//
// S256 only. `plain` offers no protection against an attacker who can read the
// authorization request, and no provider Bruno targets requires it.
//
// Source: bruno-electron/src/utils/oauth2.js (code_challenge_method S256)
func NewPKCEPair() PKCEPair {
	// 32 random bytes, base64url-encoded, is 43 characters: the minimum the RFC
	// allows and well inside the 128-character maximum.
	verifier := base64.RawURLEncoding.EncodeToString(randomBytes(32))
	return PKCEPair{
		Verifier:  verifier,
		Challenge: ChallengeFor(verifier),
		Method:    "S256",
	}
}

// ChallengeFor returns the S256 challenge for a verifier.
//
// It is exported so a caller can verify a pair it received from elsewhere, and
// so a test can prove the challenge really hashes from the verifier rather than
// being independently random.
func ChallengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewState returns a random state value.
//
// The user's configured state is *appended to* a random nonce rather than
// replaced, which is what Bruno does: the nonce is what makes the check
// meaningful, and the user's text is kept so their own callback handler sees it.
func NewState(userState string) string {
	nonce := base64.RawURLEncoding.EncodeToString(randomBytes(16))
	if strings.TrimSpace(userState) == "" {
		return nonce
	}
	return userState + nonce
}
