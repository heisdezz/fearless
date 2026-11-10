package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"fearless/internal/core/model"
)

// awsv4Provider implements AWS Signature Version 4.
//
// Signing has five steps (AWS IAM User Guide, "Create a signed AWS API
// request"):
//
//  1. canonical request: method, URI, canonical query, canonical headers,
//     signed-header list, payload hash
//  2. hash of the canonical request
//  3. string to sign: algorithm, timestamp, credential scope, that hash
//  4. signing key: HMAC chain over date, region, service, "aws4_request"
//  5. signature: HMAC of the string to sign with the signing key
//
// The result goes in the Authorization header. This provider signs at Prepare
// time rather than wrapping the transport, because SigV4 needs the final body
// bytes and URL — both of which are fixed by the time the pipeline calls it.
type awsv4Provider struct{}

func (awsv4Provider) Mode() model.AuthMode { return model.AuthAWSV4 }

const (
	awsAlgorithm     = "AWS4-HMAC-SHA256"
	awsRequestSuffix = "aws4_request"
	// awsDateLayout is the compact date used in the credential scope.
	awsDateLayout = "20060102"
	// awsTimeLayout is the full timestamp used in X-Amz-Date.
	awsTimeLayout = "20060102T150405Z"
	// emptyPayloadHash is sha256("") — required for requests with no body.
	emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// Prepare signs the request.
//
// The signature is computed over the URL and body the caller supplies through
// Deps.Request, because headers alone are not enough: the host, path and query
// all take part. When no request is available the provider returns an error
// rather than emitting a signature that cannot be correct.
func (awsv4Provider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	accessKey := interpolate(deps, model.DerefString(cfg.AccessKeyID))
	secretKey := interpolate(deps, model.DerefString(cfg.SecretAccessKey))
	sessionToken := interpolate(deps, model.DerefString(cfg.SessionToken))
	region := interpolate(deps, model.DerefString(cfg.Region))
	service := interpolate(deps, model.DerefString(cfg.Service))

	// Without credentials there is nothing to sign; Bruno warns and sends
	// unsigned, and matching that is friendlier than failing the request.
	if accessKey == "" || secretKey == "" {
		return Prepared{}, nil
	}
	if region == "" {
		region = "us-east-1"
	}
	if service == "" {
		service = "execute-api"
	}

	if deps.Request == nil {
		return Prepared{}, errNoRequestForSigning
	}

	now := time.Now().UTC()
	if deps.Now != nil {
		now = time.Unix(deps.Now(), 0).UTC()
	}

	signed, err := signV4(signInput{
		Request:      deps.Request,
		Body:         deps.Body,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
		Region:       region,
		Service:      service,
		Time:         now,
	})
	if err != nil {
		return Prepared{}, err
	}

	// The signature header and X-Amz-* headers must all be set for the request to
	// verify, so they are returned together.
	headers := []Header{
		{Name: "X-Amz-Date", Value: signed.timestamp},
		{Name: "Authorization", Value: signed.authorization},
	}
	if sessionToken != "" {
		headers = append(headers, Header{Name: "X-Amz-Security-Token", Value: sessionToken})
	}
	return Prepared{Headers: headers}, nil
}

// signInput is everything needed to compute a signature.
type signInput struct {
	Request      *http.Request
	Body         []byte
	AccessKey    string
	SecretKey    string
	SessionToken string
	Region       string
	Service      string
	Time         time.Time
	// UnsignedPayload emits UNSIGNED-PAYLOAD instead of hashing the body, which
	// S3 permits for streaming uploads.
	UnsignedPayload bool
}

// signResult is the output of signing.
type signResult struct {
	authorization string
	timestamp     string
}

// signV4 computes the Authorization header value.
func signV4(in signInput) (signResult, error) {
	if in.Request == nil || in.Request.URL == nil {
		return signResult{}, errNoRequestForSigning
	}

	timestamp := in.Time.UTC().Format(awsTimeLayout)
	dateStamp := in.Time.UTC().Format(awsDateLayout)

	payloadHash := emptyPayloadHash
	if in.UnsignedPayload {
		payloadHash = "UNSIGNED-PAYLOAD"
	} else if len(in.Body) > 0 {
		sum := sha256.Sum256(in.Body)
		payloadHash = hex.EncodeToString(sum[:])
	}

	// Headers that take part in the signature, following the AWS SDK's
	// requiredSignedHeaders allow list: the standard headers that affect the
	// response, plus every x-amz-* header. A narrower list would silently drop a
	// header the caller set and produce a signature AWS rejects.
	headers := map[string]string{}
	host := in.Request.Host
	if host == "" {
		host = in.Request.URL.Host
	}
	headers["host"] = host
	for name, values := range in.Request.Header {
		lower := strings.ToLower(name)
		if lower == "authorization" || lower == "host" || !isSignedHeader(lower) {
			continue
		}
		// Each value is normalised separately, then the values are joined with
		// commas — AWS collapses whitespace *within* a value, not across a list.
		headers[lower] = normaliseHeaderValues(values)
	}
	// The timestamp is set on the request, so it is signed.
	headers["x-amz-date"] = timestamp
	if in.SessionToken != "" {
		headers["x-amz-security-token"] = in.SessionToken
	}

	canonicalHeaders, signedHeaders := canonicalizeHeaders(headers)

	// The specification puts exactly one empty line between the canonical headers
	// and the signed-header list. canonicalHeaders already ends with a newline, so
	// the join separator supplies that blank line; adding a second would insert an
	// extra one and change every signature.
	canonicalRequest := strings.Join([]string{
		in.Request.Method,
		canonicalURI(in.Request.URL),
		canonicalQuery(in.Request.URL.RawQuery),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := strings.Join([]string{dateStamp, in.Region, in.Service, awsRequestSuffix}, "/")
	stringToSign := strings.Join([]string{
		awsAlgorithm,
		timestamp,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	key := signingKey(in.SecretKey, dateStamp, in.Region, in.Service)
	signature := hex.EncodeToString(hmacSHA256(key, []byte(stringToSign)))

	authorization := awsAlgorithm +
		" Credential=" + in.AccessKey + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders +
		", Signature=" + signature

	return signResult{authorization: authorization, timestamp: timestamp}, nil
}

// canonicalizeHeaders builds the canonical header block and the signed-header
// list.
//
// Values have already had their whitespace normalised by the caller; this
// function sorts names and joins them. Sorting is by byte order, which for
// lower-case ASCII names is alphabetical.
func canonicalizeHeaders(headers map[string]string) (block, signed string) {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(headers[name])
		b.WriteByte('\n')
	}
	return b.String(), strings.Join(names, ";")
}

// normaliseHeaderValues collapses whitespace in each value and joins them with
// commas.
//
// AWS collapses whitespace inside a value and joins repeated headers with a
// comma; doing both in the other order would leave tabs and runs of spaces that
// AWS's verifier normalises away, producing a signature it rejects.
func normaliseHeaderValues(values []string) string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, collapseSpaces(v))
	}
	return strings.Join(out, ",")
}

// collapseSpaces trims a value and collapses runs of consecutive spaces.
//
// Only spaces are collapsed, not tabs: AWS's stripExcessSpaces looks for
// "  " specifically. Using a generic whitespace split would also fold tabs and
// newlines, producing a canonical header AWS does not compute.
func collapseSpaces(s string) string {
	s = strings.TrimSpace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// canonicalURI returns the URI-encoded path, or "/" when empty.
//
// A URL built with Opaque set carries its request-target there with Path empty;
// that form is what http.Request writes on the wire, so it is what must be
// signed. The AWS SDK derives it as "/" + the segments after the authority,
// which is reproduced exactly here so signatures match its output.
func canonicalURI(u *url.URL) string {
	if u == nil {
		return "/"
	}

	var uri string
	if u.Opaque != "" {
		// Opaque is "//host/path..."; drop the two empty leading segments and the
		// host, keeping the rest as the path.
		segments := strings.Split(u.Opaque, "/")
		if len(segments) > 3 {
			uri = "/" + strings.Join(segments[3:], "/")
		}
	} else {
		uri = u.EscapedPath()
	}
	if uri == "" {
		return "/"
	}
	return uri
}

// canonicalQuery sorts and URI-encodes query parameters.
//
// The AWS SDK re-encodes the query through url.Values.Encode (which sorts by
// key and encodes values) and then replaces "+" with "%20". Reproducing that
// exactly matters: Go's Encode escapes differently from awsURIEncode in a few
// cases, and the signature must match what the SDK produces.
func canonicalQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// A malformed query is signed as given rather than dropped.
		return rawQuery
	}
	// The SDK sorts each key's values before encoding; url.Values.Encode already
	// sorts keys, so this makes the whole thing deterministic.
	for key := range values {
		sort.Strings(values[key])
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

// awsURIEncode percent-encodes a string per the SigV4 rules.
//
// Unreserved characters (A-Z a-z 0-9 - . _ ~) pass through; a space becomes %20,
// not "+"; hex digits are upper case. This is used for path components; the
// query string is encoded through url.Values.Encode in canonicalQuery, because
// that is what the AWS SDK does and the two differ on a few characters.
func awsURIEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			b.WriteByte(c)
		case c == '/':
			if encodeSlash {
				b.WriteString("%2F")
			} else {
				b.WriteByte('/')
			}
		default:
			const hexDigits = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xF])
		}
	}
	return b.String()
}

// signingKey derives the SigV4 signing key.
//
//	kDate    = HMAC("AWS4"+secret, date)
//	kRegion  = HMAC(kDate, region)
//	kService = HMAC(kRegion, service)
//	key      = HMAC(kService, "aws4_request")
func signingKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte(awsRequestSuffix))
}

// hmacSHA256 computes an HMAC-SHA256.
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// sha256Hex returns the lower-case hex SHA-256 of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
