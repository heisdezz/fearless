// Package auth applies authentication to outgoing requests.
//
// A provider is one file that declares how a mode transforms a request. The
// registry maps modes to providers, so adding a mode is one file plus one
// Register call — contrast Bruno, where a new scheme touches the request
// builder, the transport wrapper, the schema, the UI and the serialiser.
//
// Three kinds of provider exist, distinguished by what they need:
//
//   - Prepare-only (basic, bearer, api-key, wsse): compute headers or query
//     parameters before sending. No transport involvement.
//   - Challenge-response (digest, ntlm): send, read a challenge, retry. These
//     implement TransportProvider.
//   - Token-based (oauth2): fetch or refresh a token, then prepare. These
//     implement TokenProvider.
//
// See docs/logic/authentication.md for the per-mode semantics this implements.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// Prepared is what a provider contributes to a request.
//
// It is separate from *http.Request because auth is applied before the body is
// serialised and before the URL is finalised in some cases; merging at the end
// keeps the order deterministic.
type Prepared struct {
	// Headers to set, in the order added. Later entries replace earlier ones
	// with the same name.
	Headers []Header
	// QueryParams to append to the URL.
	QueryParams []QueryParam
	// Transport, when non-nil, wraps the underlying round tripper. Digest and
	// NTLM use this to retry after a challenge.
	Transport TransportProvider
	// OnFailure, when non-nil, is called if the request fails with a status the
	// provider wants to report specially.
	OnFailure func(status int)
}

// Header is one header a provider set.
type Header struct {
	Name  string
	Value string
}

// QueryParam is one query parameter a provider appended.
type QueryParam struct {
	Name  string
	Value string
}

// Deps are the collaborators a provider may need.
//
// They are injected rather than imported so providers stay testable and the
// package graph stays acyclic.
type Deps struct {
	// Interpolate resolves {{var}} in auth fields. Providers call it on values
	// that came from configuration; values they compute are never interpolated.
	Interpolate func(string) string
	// Scopes is the resolved variable environment, for providers that need to
	// read a token variable.
	Scopes vars.Scopes
	// TokenStore returns a cached OAuth2 token, or false.
	TokenStore func(credentialsID string) (Token, bool)
	// TokenFetcher obtains a new token. Supplied by the oauth2 package.
	TokenFetcher func(ctx context.Context, cfg *model.Auth) (Token, error)
	// CollectionPath is the collection root, for providers that read files
	// relative to it (OAuth1 private keys).
	CollectionPath string
	// Now returns the current time; injectable for deterministic tests.
	Now func() int64
	// RandBytes returns n random bytes; injectable for deterministic tests.
	RandBytes func(n int) []byte
	// Getenv reads a process environment variable.
	Getenv func(string) string
	// Rewind rebuilds the request for a retry after a challenge.
	//
	// Challenge-response schemes (digest, ntlm) consume the request body on the
	// first attempt, so a retry needs a fresh one. The pipeline supplies this; when
	// it is nil, providers that need it fall back to returning the original
	// response unchanged rather than failing.
	Rewind func() (*http.Request, error)
	// Request is the request being prepared, for signers that cover the URL.
	//
	// SigV4 signs the method, path and query as well as headers, so it cannot work
	// from configuration alone. Providers that do not need it ignore it.
	Request *http.Request
	// Body is the exact payload that will be sent.
	//
	// SigV4 hashes it, so it must be the final bytes after every other transform
	// (compression, multipart encoding). A nil Body is treated as empty.
	Body []byte
}

// Token is an obtained OAuth2 token.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    int64 // unix seconds; zero means no expiry
	Scope        string
}

// Expired reports whether the token has expired at now.
func (t Token) Expired(now int64) bool {
	return t.ExpiresAt != 0 && now >= t.ExpiresAt
}

// TransportProvider wraps the transport for challenge-response schemes.
type TransportProvider interface {
	// Wrap returns a round tripper that performs the handshake.
	Wrap(next http.RoundTripper) http.RoundTripper
}

// Provider is the single method every auth mode implements.
type Provider interface {
	// Mode reports which auth mode this provider handles.
	Mode() model.AuthMode
	// Prepare applies the credentials to the request.
	Prepare(ctx context.Context, cfg *model.Auth, deps Deps) (Prepared, error)
}

// registry maps modes to providers.
var (
	registryMu sync.RWMutex
	registry   = make(map[model.AuthMode]Provider)
)

// Register adds a provider. It panics on a nil provider or a duplicate mode,
// because both indicate a wiring bug that would otherwise surface as confusing
// runtime behaviour.
func Register(p Provider) {
	if p == nil {
		panic("auth: cannot register a nil provider")
	}
	mode := p.Mode()
	if mode == "" {
		panic("auth: cannot register a provider with an empty mode")
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[mode]; exists {
		panic(fmt.Sprintf("auth: provider for %q already registered", mode))
	}
	registry[mode] = p
}

// Get returns the provider for a mode.
func Get(mode model.AuthMode) (Provider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[mode]
	return p, ok
}

// Modes returns the registered modes in a stable order.
func Modes() []model.AuthMode {
	registryMu.RLock()
	modes := make([]model.AuthMode, 0, len(registry))
	for m := range registry {
		modes = append(modes, m)
	}
	registryMu.RUnlock()
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	return modes
}

// ErrNoProvider reports a mode with no registered provider.
var ErrNoProvider = errors.New("auth: no provider registered for mode")

// Apply resolves the effective auth mode and prepares the request.
//
// Resolution walks the supplied chain from least to most specific, which is how
// Bruno's `inherit` works: a request that inherits takes the nearest ancestor
// that sets a concrete mode.
//
// A nil chain, or one whose every entry is empty, inherit or none, yields an
// empty Prepared and no error: unauthenticated is a valid outcome, not a
// failure.
func Apply(ctx context.Context, chain []*model.Auth, deps Deps) (Prepared, error) {
	cfg := Resolve(chain)
	if cfg == nil {
		return Prepared{}, nil
	}

	provider, ok := Get(cfg.Mode)
	if !ok {
		if cfg.Mode == model.AuthNone || cfg.Mode == model.AuthInherit {
			return Prepared{}, nil
		}
		return Prepared{}, fmt.Errorf("%w: %s", ErrNoProvider, cfg.Mode)
	}
	return provider.Prepare(ctx, cfg, deps)
}

// Resolve picks the effective auth config from an ordered chain.
//
// The chain is ordered least specific first: collection, then folders outermost
// to innermost, then the request. The last entry with a concrete mode wins.
func Resolve(chain []*model.Auth) *model.Auth {
	var chosen *model.Auth
	for _, cfg := range chain {
		if cfg == nil {
			continue
		}
		switch cfg.EffectiveMode() {
		case model.AuthInherit:
			continue
		case model.AuthNone:
			// `none` is explicit: it stops inheritance rather than passing it on.
			chosen = cfg
		default:
			chosen = cfg
		}
	}
	if chosen == nil || chosen.EffectiveMode() == model.AuthNone {
		return nil
	}
	// Return a deep copy so a provider cannot mutate stored configuration through
	// the returned pointer. A struct copy alone is not enough: the string fields
	// are pointers, and writing through one would reach the caller's original.
	clone := *chosen
	for _, p := range []**string{
		&clone.Username, &clone.Password, &clone.Domain, &clone.Token,
		&clone.Key, &clone.Value, &clone.AccessKeyID, &clone.SecretAccessKey,
		&clone.SessionToken, &clone.Service, &clone.Region, &clone.ProfileName,
		&clone.ConsumerKey, &clone.ConsumerSecret, &clone.AccessToken,
		&clone.AccessTokenSecret, &clone.CallbackURL, &clone.Verifier,
		&clone.SignatureMethod, &clone.PrivateKey, &clone.PrivateKeyType,
		&clone.Timestamp, &clone.Nonce, &clone.Version, &clone.Realm,
		&clone.AuthorizationURL, &clone.AccessTokenURL, &clone.RefreshTokenURL,
		&clone.ClientID, &clone.ClientSecret, &clone.Scope, &clone.State,
		&clone.CredentialsPlacement, &clone.CredentialsID, &clone.TokenPlacement,
		&clone.TokenHeaderPrefix, &clone.TokenQueryKey, &clone.TokenSource,
		&clone.EdgeGridClientToken, &clone.EdgeGridClientSecret,
		&clone.EdgeGridBaseURL, &clone.EdgeGridHeadersToSign,
	} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	return &clone
}
