package auth

import (
	"context"
	"encoding/base64"

	"fearless/internal/core/model"
)

// basicProvider implements HTTP Basic authentication (RFC 7617).
type basicProvider struct{}

func (basicProvider) Mode() model.AuthMode { return model.AuthBasic }

// Prepare sets the Authorization header to Basic base64(user:pass).
//
// Bruno encodes UTF-8 (Node's Buffer.from) even though RFC 7617 nominally uses
// ISO-8859-1. We match Bruno: servers expecting Latin-1 are the exception, and a
// divergence here would silently break existing collections.
func (basicProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	user := interpolate(deps, model.DerefString(cfg.Username))
	pass := interpolate(deps, model.DerefString(cfg.Password))

	encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return Prepared{
		Headers: []Header{{Name: "Authorization", Value: "Basic " + encoded}},
	}, nil
}

// digestProvider implements HTTP Digest authentication.
//
// The credentials are the same as Basic; the difference is that the header is
// computed after a 401 challenge, which requires a transport wrapper. That dance
// lives in digest.go (AUTH-DIGEST); this provider only records the config.
type digestConfigProvider struct{}

func (digestConfigProvider) Mode() model.AuthMode { return model.AuthDigest }

// Prepare records nothing: digest cannot be applied before the server supplies a
// nonce. The transport provider reads the config from RequestContext instead.
func (digestConfigProvider) Prepare(_ context.Context, _ *model.Auth, _ Deps) (Prepared, error) {
	return Prepared{}, nil
}

// wsseProvider implements WSSE UsernameToken (used by some SOAP APIs).
type wsseProvider struct{}

func (wsseProvider) Mode() model.AuthMode { return model.AuthWSSE }

// Prepare builds the X-WSSE header.
//
//	nonce  = 16 random bytes as hex
//	ts     = RFC 3339 timestamp
//	digest = base64( hex( sha1( nonce + ts + password ) ) )
//
// Bruno computes the digest *before* interpolation, so `{{var}}` in a password is
// hashed literally. That is almost certainly a bug, so this implementation
// interpolates first and the divergence is recorded in decisions D-009.
func (wsseProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	user := interpolate(deps, model.DerefString(cfg.Username))
	pass := interpolate(deps, model.DerefString(cfg.Password))

	nonce := hexEncode(randomBytes(deps, 16))
	created := rfc3339(nowUnix(deps))

	digest := wsseDigest(nonce, created, pass)
	value := `UsernameToken Username="` + user + `", PasswordDigest="` + digest +
		`", Nonce="` + nonce + `", Created="` + created + `"`

	return Prepared{
		Headers: []Header{{Name: "X-WSSE", Value: value}},
	}, nil
}
