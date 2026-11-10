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
// Digest cannot set a header up front: the response depends on a nonce the
// server has not sent yet. Prepare therefore returns a TransportProvider that
// performs the 401 challenge/retry dance, which is the whole mechanism.
type digestProvider struct{}

func (digestProvider) Mode() model.AuthMode { return model.AuthDigest }

// Prepare supplies a transport that signs the request after a challenge.
//
// The credentials are interpolated here, before the transport runs, so a
// `{{password}}` reference works. The rewind function rebuilds the request from
// the pipeline's replayable body; when the caller supplied one, retries are
// possible, and when it did not, the first (unauthenticated) response is returned
// unchanged rather than failing — which is exactly what a server that rejects
// Digest would produce anyway.
func (digestProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	user := interpolate(deps, model.DerefString(cfg.Username))
	pass := interpolate(deps, model.DerefString(cfg.Password))

	return Prepared{
		Transport: &digestTransport{
			user:   user,
			pass:   pass,
			rewind: deps.Rewind,
		},
	}, nil
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
