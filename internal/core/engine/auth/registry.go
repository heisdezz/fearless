package auth

import (
	"crypto/md5"

	"fearless/internal/core/model"
)

// md5Sum returns the MD5 digest of s.
//
// MD5 is the required hash for HTTP Digest auth and AWS SigV4's optional body
// checksum; it is not used here for anything security-sensitive beyond what
// those protocols mandate.
func md5Sum(s string) []byte {
	sum := md5.Sum([]byte(s))
	return sum[:]
}

// RegisterBase registers the providers this package owns.
//
// It is called once from an explicit wiring point rather than from init(), so the
// dependency graph stays predictable and tests can register into a fresh
// registry. Calling it twice panics via Register, which catches double wiring.
func RegisterBase() {
	Register(noneProvider{})
	Register(basicProvider{})
	Register(bearerProvider{})
	Register(apiKeyProvider{})
	Register(wsseProvider{})
	// Digest's work happens in the transport, after the server's challenge; the
	// provider supplies that transport. See digest.go.
	Register(digestProvider{})
	Register(awsv4Provider{})
}

// ChainFor builds the auth inheritance chain for one request.
//
// Order is least specific first: collection, then each enclosing folder from
// outermost to innermost, then the request itself. Resolve() then picks the last
// concrete mode, which is what `inherit` means in Bruno.
//
// ancestors must already be ordered outermost-first, as model.Collection.Ancestors
// returns them.
func ChainFor(collection *model.Collection, ancestors []*model.Item, item *model.Item) []*model.Auth {
	var chain []*model.Auth
	if collection != nil && collection.Config.Auth != nil {
		chain = append(chain, collection.Config.Auth)
	}
	for _, folder := range ancestors {
		if folder != nil && folder.Auth != nil {
			chain = append(chain, folder.Auth)
		}
	}
	if item != nil && item.Request != nil && item.Request.Auth != nil {
		chain = append(chain, item.Request.Auth)
	}
	return chain
}
