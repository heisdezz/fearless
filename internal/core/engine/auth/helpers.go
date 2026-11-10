package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// defaults fill in the injectable deps so a provider never has to nil-check.
const (
	defaultNonceBytes = 16
)

// interpolate resolves a configured value.
//
// Providers only call this on values that came from configuration. A value the
// provider computed itself (a base64 blob, a digest, a timestamp) is never
// interpolated, because doing so would corrupt it when it happens to contain
// braces.
func interpolate(deps Deps, s string) string {
	if deps.Interpolate == nil || s == "" {
		return s
	}
	return deps.Interpolate(s)
}

// nowUnix returns the current unix time, honouring the injected clock.
func nowUnix(deps Deps) int64 {
	if deps.Now != nil {
		return deps.Now()
	}
	return time.Now().Unix()
}

// randomBytes returns n random bytes, honouring the injected source.
//
// A provider that needs randomness must use this rather than crypto/rand
// directly, so tests can pin the output and assert exact header values.
func randomBytes(deps Deps, n int) []byte {
	if deps.RandBytes != nil {
		return deps.RandBytes(n)
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing is unrecoverable; an all-zero nonce is worse than
		// a panic because it silently weakens the digest.
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return buf
}

// hexEncode renders bytes as lower-case hex.
func hexEncode(b []byte) string { return hex.EncodeToString(b) }

// rfc3339 renders a unix time as RFC 3339 UTC, the format WSSE expects.
func rfc3339(unix int64) string {
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}

// wsseDigest computes base64(hex(sha1(nonce + created + password))).
//
// The hex-inside-base64 composition is Bruno's, and it is unusual enough that a
// from-scratch implementation routinely gets it wrong by omitting the hex step.
func wsseDigest(nonce, created, password string) string {
	sum := sha1.Sum([]byte(nonce + created + password))
	return base64.StdEncoding.EncodeToString([]byte(hexEncode(sum[:])))
}

// md5Hex is used by digest auth.
func md5Hex(s string) string {
	return hex.EncodeToString(md5Sum(s))
}

// quoteHeaderValue wraps a value in double quotes, escaping embedded quotes and
// backslashes, for the header syntax of digest auth.
func quoteHeaderValue(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		if r == '"' || r == '\\' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

// jsonMarshal is a thin alias so providers do not each import encoding/json.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
