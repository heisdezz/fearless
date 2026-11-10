package model

// AuthMode selects an authentication strategy.
//
// Inherit means "use the nearest ancestor that sets a mode", resolved during
// request building rather than at parse time.
type AuthMode string

const (
	AuthInherit  AuthMode = "inherit"
	AuthNone     AuthMode = "none"
	AuthBasic    AuthMode = "basic"
	AuthBearer   AuthMode = "bearer"
	AuthDigest   AuthMode = "digest"
	AuthNTLM     AuthMode = "ntlm"
	AuthOAuth1   AuthMode = "oauth1"
	AuthOAuth2   AuthMode = "oauth2"
	AuthWSSE     AuthMode = "wsse"
	AuthAPIKey   AuthMode = "apikey"
	AuthAWSV4    AuthMode = "awsv4"
	AuthEdgeGrid AuthMode = "akamai-edgegrid"
)

// AllAuthModes lists every mode in the order the UI presents them.
var AllAuthModes = []AuthMode{
	AuthInherit, AuthNone, AuthBasic, AuthBearer, AuthDigest, AuthNTLM,
	AuthOAuth1, AuthOAuth2, AuthWSSE, AuthAPIKey, AuthAWSV4, AuthEdgeGrid,
}

// Valid reports whether m is a known mode.
func (m AuthMode) Valid() bool {
	for _, candidate := range AllAuthModes {
		if m == candidate {
			return true
		}
	}
	return false
}

// APIKeyPlacement is where an API key is injected.
type APIKeyPlacement string

const (
	APIKeyHeader      APIKeyPlacement = "header"
	APIKeyQueryParams APIKeyPlacement = "queryparams"
)

// OAuth2GrantType is the grant an OAuth2 config uses.
type OAuth2GrantType string

const (
	GrantAuthorizationCode OAuth2GrantType = "authorization_code"
	GrantClientCredentials OAuth2GrantType = "client_credentials"
	GrantPassword          OAuth2GrantType = "password"
	GrantImplicit          OAuth2GrantType = "implicit"
)

// AKV is a single header/query row used by the "additional parameters"
// sections of OAuth2 configs.
type AKV struct {
	Name    *string `json:"name" yaml:"name"`
	Value   *string `json:"value" yaml:"value"`
	Enabled bool    `json:"enabled" yaml:"enabled"`
}

// OAuth2AdditionalParams holds the extra values sent with each OAuth2 request.
type OAuth2AdditionalParams struct {
	AuthorizationRequest AKVSet `json:"authorizationRequest,omitempty" yaml:"authorizationRequest,omitempty"`
	AccessTokenRequest   AKVSet `json:"accessTokenRequest,omitempty" yaml:"accessTokenRequest,omitempty"`
	RefreshTokenRequest  AKVSet `json:"refreshTokenRequest,omitempty" yaml:"refreshTokenRequest,omitempty"`
}

// AKVSet is a set of headers, query params and (for token requests) body rows.
type AKVSet struct {
	Headers     []AKV `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams []AKV `json:"queryParams,omitempty" yaml:"queryParams,omitempty"`
	Body        []AKV `json:"body,omitempty" yaml:"body,omitempty"`
}

// Auth is the authentication configuration union.
//
// As with Body, Bruno keeps every mode's fields in one object and selects one
// by Mode, so switching modes in the UI preserves the others. Pointer fields
// keep "unset" distinct from "empty string" because an empty client secret is
// a legal, and different, value.
type Auth struct {
	Mode AuthMode `json:"mode" yaml:"mode"`

	// basic, digest, wsse, ntlm, oauth2
	Username *string `json:"username,omitempty" yaml:"username,omitempty"`
	Password *string `json:"password,omitempty" yaml:"password,omitempty"`

	// ntlm
	Domain *string `json:"domain,omitempty" yaml:"domain,omitempty"`

	// bearer
	Token *string `json:"token,omitempty" yaml:"token,omitempty"`

	// apikey
	Key       *string         `json:"key,omitempty" yaml:"key,omitempty"`
	Value     *string         `json:"value,omitempty" yaml:"value,omitempty"`
	Placement APIKeyPlacement `json:"placement,omitempty" yaml:"placement,omitempty"`

	// awsv4
	AccessKeyID     *string `json:"accessKeyId,omitempty" yaml:"accessKeyId,omitempty"`
	SecretAccessKey *string `json:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`
	SessionToken    *string `json:"sessionToken,omitempty" yaml:"sessionToken,omitempty"`
	Service         *string `json:"service,omitempty" yaml:"service,omitempty"`
	Region          *string `json:"region,omitempty" yaml:"region,omitempty"`
	ProfileName     *string `json:"profileName,omitempty" yaml:"profileName,omitempty"`

	// oauth1
	ConsumerKey       *string `json:"consumerKey,omitempty" yaml:"consumerKey,omitempty"`
	ConsumerSecret    *string `json:"consumerSecret,omitempty" yaml:"consumerSecret,omitempty"`
	AccessToken       *string `json:"accessToken,omitempty" yaml:"accessToken,omitempty"`
	AccessTokenSecret *string `json:"accessTokenSecret,omitempty" yaml:"accessTokenSecret,omitempty"`
	CallbackURL       *string `json:"callbackUrl,omitempty" yaml:"callbackUrl,omitempty"`
	Verifier          *string `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	SignatureMethod   *string `json:"signatureMethod,omitempty" yaml:"signatureMethod,omitempty"`
	PrivateKey        *string `json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	PrivateKeyType    *string `json:"privateKeyType,omitempty" yaml:"privateKeyType,omitempty"`
	Timestamp         *string `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Nonce             *string `json:"nonce,omitempty" yaml:"nonce,omitempty"`
	Version           *string `json:"version,omitempty" yaml:"version,omitempty"`
	Realm             *string `json:"realm,omitempty" yaml:"realm,omitempty"`
	IncludeBodyHash   *bool   `json:"includeBodyHash,omitempty" yaml:"includeBodyHash,omitempty"`

	// oauth2
	GrantType            OAuth2GrantType         `json:"grantType,omitempty" yaml:"grantType,omitempty"`
	AuthorizationURL     *string                 `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	AccessTokenURL       *string                 `json:"accessTokenUrl,omitempty" yaml:"accessTokenUrl,omitempty"`
	RefreshTokenURL      *string                 `json:"refreshTokenUrl,omitempty" yaml:"refreshTokenUrl,omitempty"`
	ClientID             *string                 `json:"clientId,omitempty" yaml:"clientId,omitempty"`
	ClientSecret         *string                 `json:"clientSecret,omitempty" yaml:"clientSecret,omitempty"`
	Scope                *string                 `json:"scope,omitempty" yaml:"scope,omitempty"`
	State                *string                 `json:"state,omitempty" yaml:"state,omitempty"`
	PKCE                 *bool                   `json:"pkce,omitempty" yaml:"pkce,omitempty"`
	CredentialsPlacement *string                 `json:"credentialsPlacement,omitempty" yaml:"credentialsPlacement,omitempty"`
	CredentialsID        *string                 `json:"credentialsId,omitempty" yaml:"credentialsId,omitempty"`
	TokenPlacement       *string                 `json:"tokenPlacement,omitempty" yaml:"tokenPlacement,omitempty"`
	TokenHeaderPrefix    *string                 `json:"tokenHeaderPrefix,omitempty" yaml:"tokenHeaderPrefix,omitempty"`
	TokenQueryKey        *string                 `json:"tokenQueryKey,omitempty" yaml:"tokenQueryKey,omitempty"`
	AutoRefreshToken     *bool                   `json:"autoRefreshToken,omitempty" yaml:"autoRefreshToken,omitempty"`
	AutoFetchToken       *bool                   `json:"autoFetchToken,omitempty" yaml:"autoFetchToken,omitempty"`
	TokenSource          *string                 `json:"tokenSource,omitempty" yaml:"tokenSource,omitempty"`
	AdditionalParams     *OAuth2AdditionalParams `json:"additionalParameters,omitempty" yaml:"additionalParameters,omitempty"`

	// akamai-edgegrid. EdgeGrid has its own client secret, kept in a separate
	// field because a request can carry both OAuth2 and EdgeGrid settings
	// (Bruno stores every mode in one object).
	EdgeGridClientToken   *string `json:"clientToken,omitempty" yaml:"clientToken,omitempty"`
	EdgeGridClientSecret  *string `json:"clientSecretEdgeGrid,omitempty" yaml:"clientSecretEdgeGrid,omitempty"`
	EdgeGridBaseURL       *string `json:"baseURL,omitempty" yaml:"baseURL,omitempty"`
	EdgeGridHeadersToSign *string `json:"headersToSign,omitempty" yaml:"headersToSign,omitempty"`
	EdgeGridMaxBodySize   *int    `json:"maxBodySize,omitempty" yaml:"maxBodySize,omitempty"`
}

// IsZero reports whether no auth was configured at all.
func (a *Auth) IsZero() bool {
	return a == nil || (a.Mode == "" && a.Username == nil && a.Token == nil && a.Key == nil)
}

// EffectiveMode returns the mode with "" normalised to inherit.
func (a *Auth) EffectiveMode() AuthMode {
	if a == nil || a.Mode == "" {
		return AuthInherit
	}
	return a.Mode
}

// DerefString returns *p, or "" when p is nil. A small helper used pervasively
// by the builder and provider packages.
func DerefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// DerefBool returns *p, or def when p is nil.
func DerefBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// DerefInt returns *p, or def when p is nil.
func DerefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
