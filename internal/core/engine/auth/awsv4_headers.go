package auth

import "strings"

// signedStandardHeaders is the allow list of non-x-amz headers that take part in
// a SigV4 signature.
//
// It mirrors the AWS SDK's requiredSignedHeaders set: headers that determine what
// the service returns or how it interprets the body, plus the conditional and
// range headers. Signing a broader set would break when a proxy injects
// something; signing a narrower one silently drops a header the caller set, and
// AWS then rejects the request.
var signedStandardHeaders = map[string]bool{
	"cache-control":       true,
	"content-disposition": true,
	"content-encoding":    true,
	"content-language":    true,
	"content-length":      true,
	"content-md5":         true,
	"content-type":        true,
	"expires":             true,
	"if-match":            true,
	"if-modified-since":   true,
	"if-none-match":       true,
	"if-unmodified-since": true,
	"range":               true,
}

// isSignedHeader reports whether a lower-case header name participates in the
// signature.
//
// Every x-amz-* header is signed (including x-amz-meta-*, x-amz-object-lock-* and
// x-amz-security-token), as are the standard headers in signedStandardHeaders.
// Host is handled separately because it has a fallback.
func isSignedHeader(lower string) bool {
	if strings.HasPrefix(lower, "x-amz-") {
		return true
	}
	return signedStandardHeaders[lower]
}
