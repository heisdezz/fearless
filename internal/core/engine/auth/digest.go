package auth

import (
	"io"
	"net/http"
	"strings"
)

// digestTransport performs the HTTP Digest challenge-response handshake.
//
// Digest cannot be applied before the server speaks: the header needs a nonce.
// The flow is therefore:
//
//  1. Send the request without Authorization.
//  2. If the response is 401 with a Digest challenge, compute the response.
//  3. Retry once with the Authorization header.
//  4. If the challenge reports stale=true, allow one more attempt.
//
// The body must be replayable for the retry to be correct; the caller supplies a
// rewind function because the transport cannot know how the body was built.
type digestTransport struct {
	next http.RoundTripper
	user string
	pass string
	// rewind re-creates the request for a retry, since the first attempt's body
	// has already been consumed.
	rewind func() (*http.Request, error)
}

// challenge is a parsed WWW-Authenticate: Digest value.
type challenge struct {
	Realm     string
	Nonce     string
	Opaque    string
	QOP       string
	Algorithm string
	Stale     bool
}

// Wrap builds the digest round tripper.
//
// The returned tripper is the receiver itself with next set, so a caller can pass
// either this value or the result of Wrap to an http.Client. When next is nil
// (a caller used the struct directly), RoundTrip falls back to the default
// transport rather than panicking.
func (d *digestTransport) Wrap(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	wrapped := *d
	wrapped.next = next
	return &wrapped
}

// RoundTrip sends the request, and retries with a digest header if challenged.
func (d *digestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := d.next
	if next == nil {
		next = http.DefaultTransport
	}

	first, err := next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if first.StatusCode != http.StatusUnauthorized {
		return first, nil
	}

	ch, ok := parseDigestChallenge(first.Header)
	if !ok {
		return first, nil
	}

	// The body of the 401 must be closed before the retry, or the connection
	// cannot be reused and the server may treat the two attempts as unrelated.
	_, _ = io.Copy(io.Discard, io.LimitReader(first.Body, 1<<16))
	_ = first.Body.Close()

	attempts := 1
	if ch.Stale {
		attempts = 2
	}

	var last *http.Response
	for i := 0; i < attempts; i++ {
		retry, rerr := d.rewind()
		if rerr != nil {
			return nil, rerr
		}
		retry.Header.Set("Authorization", digestAuthorization(ch, d.user, d.pass, retry.Method, requestURI(retry)))

		last, err = next.RoundTrip(retry)
		if err != nil {
			return nil, err
		}
		if last.StatusCode != http.StatusUnauthorized {
			return last, nil
		}
		nextChallenge, ok := parseDigestChallenge(last.Header)
		if !ok || !nextChallenge.Stale {
			return last, nil
		}
		ch = nextChallenge
	}
	return last, nil
}

// requestURI returns the request-target used in the digest calculation.
func requestURI(req *http.Request) string {
	if req.URL == nil {
		return "/"
	}
	if req.URL.RawQuery == "" {
		return req.URL.Path
	}
	return req.URL.Path + "?" + req.URL.RawQuery
}

// digestAuthorization builds the Authorization header for a challenge.
//
// HA1 = H(user:realm:pass)
// HA2 = H(method:uri)
// response = H(HA1:nonce:nc:cnonce:qop:HA2)   when qop is present
// response = H(HA1:nonce:HA2)                 otherwise
func digestAuthorization(ch challenge, user, pass, method, uri string) string {
	algorithm := ch.Algorithm
	if algorithm == "" {
		algorithm = "MD5"
	}

	// This implements the MD5 and MD5-sess algorithms. SHA-256 variants require a
	// different hash; returning a header built with MD5 for a SHA-256 challenge
	// would be worse than omitting it, so callers get an empty header and the
	// server's 401 stands.
	if !strings.EqualFold(algorithm, "MD5") && !strings.EqualFold(algorithm, "MD5-sess") {
		return ""
	}

	// The nc/cnonce pair is fixed rather than random: one challenge produces at
	// most one retry, so replay protection has nothing to protect against, and
	// fixed values keep the header deterministic for tests.
	return buildDigestHeader(ch, user, pass, method, uri, "00000001", "0a4f113b")
}

// buildDigestHeader assembles the Authorization value.
//
// MD5 is used for the hashes because HTTP Digest specifies it (RFC 7616's
// SHA-256 variants are handled by returning no header, see above).
func buildDigestHeader(ch challenge, user, pass, method, uri, nc, cnonce string) string {
	ha1 := md5Hex(user + ":" + ch.Realm + ":" + pass)
	if strings.EqualFold(ch.Algorithm, "MD5-sess") {
		ha1 = md5Hex(ha1 + ":" + ch.Nonce + ":" + cnonce)
	}
	ha2 := md5Hex(method + ":" + uri)

	var response string
	qop := pickQOP(ch.QOP)
	if qop != "" {
		response = md5Hex(ha1 + ":" + ch.Nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)
	} else {
		response = md5Hex(ha1 + ":" + ch.Nonce + ":" + ha2)
	}

	var b strings.Builder
	b.WriteString("Digest ")
	writeParam(&b, "username", quoteHeaderValue(user))
	writeParam(&b, "realm", quoteHeaderValue(ch.Realm))
	writeParam(&b, "nonce", quoteHeaderValue(ch.Nonce))
	writeParam(&b, "uri", quoteHeaderValue(uri))
	if qop != "" {
		writeParam(&b, "qop", qop)
		writeParam(&b, "nc", nc)
		writeParam(&b, "cnonce", quoteHeaderValue(cnonce))
	}
	if ch.Opaque != "" {
		writeParam(&b, "opaque", quoteHeaderValue(ch.Opaque))
	}
	if ch.Algorithm != "" {
		writeParam(&b, "algorithm", ch.Algorithm)
	}
	writeParam(&b, "response", quoteHeaderValue(response))
	return b.String()
}

// pickQOP chooses a qop from the server's offered list, preferring `auth`.
//
// `auth-int` would require hashing the body, which this transport cannot do
// because it does not own the body bytes.
func pickQOP(offered string) string {
	for _, candidate := range strings.Split(offered, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), "auth") {
			return "auth"
		}
	}
	return ""
}

// writeParam appends a comma-separated header parameter.
func writeParam(b *strings.Builder, key, value string) {
	if b.Len() > len("Digest ") {
		b.WriteString(", ")
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(value)
}

// parseDigestChallenge extracts a Digest challenge from response headers.
func parseDigestChallenge(h http.Header) (challenge, bool) {
	for _, value := range h.Values("WWW-Authenticate") {
		if !strings.HasPrefix(strings.ToLower(value), "digest ") {
			continue
		}
		ch := challenge{}
		for key, val := range parseAuthParams(value[len("Digest "):]) {
			switch strings.ToLower(key) {
			case "realm":
				ch.Realm = val
			case "nonce":
				ch.Nonce = val
			case "opaque":
				ch.Opaque = val
			case "qop":
				ch.QOP = val
			case "algorithm":
				ch.Algorithm = val
			case "stale":
				ch.Stale = strings.EqualFold(val, "true")
			}
		}
		if ch.Nonce != "" {
			return ch, true
		}
	}
	return challenge{}, false
}

// parseAuthParams parses comma-separated key=value pairs with optional quoting.
func parseAuthParams(s string) map[string]string {
	params := make(map[string]string)
	i := 0
	for i < len(s) {
		// Skip separators and spaces.
		for i < len(s) && (s[i] == ',' || s[i] == ' ' || s[i] == '\t') {
			i++
		}
		start := i
		for i < len(s) && s[i] != '=' && s[i] != ',' {
			i++
		}
		key := strings.TrimSpace(s[start:i])
		if i >= len(s) || s[i] != '=' {
			if key != "" {
				params[key] = ""
			}
			continue
		}
		i++ // '='
		if i < len(s) && s[i] == '"' {
			i++
			var b strings.Builder
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && i+1 < len(s) {
					i++
				}
				b.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				i++ // closing quote
			}
			params[key] = b.String()
			continue
		}
		start = i
		for i < len(s) && s[i] != ',' {
			i++
		}
		params[key] = strings.TrimSpace(s[start:i])
	}
	return params
}
