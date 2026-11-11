package auth

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// oauth1Provider implements OAuth 1.0a request signing (RFC 5849).
//
// The algorithm has four parts:
//
//  1. collect parameters: the oauth_* set, the URL's query, and form body fields
//  2. normalise them: percent-encode name and value, sort by name then value,
//     join with "&"
//  3. build the signature base string: METHOD & encoded(base URL) & encoded(params)
//  4. sign it with the chosen method and a key of encoded(consumerSecret)&encoded(tokenSecret)
//
// The result is emitted as an Authorization: OAuth header, or as query
// parameters when the client is configured that way.
type oauth1Provider struct{}

func (oauth1Provider) Mode() model.AuthMode { return model.AuthOAuth1 }

const (
	oauth1DefaultVersion    = "1.0"
	oauth1SignatureHMACSHA1 = "HMAC-SHA1"
)

// Prepare signs the request.
func (oauth1Provider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	consumerKey := interpolate(deps, model.DerefString(cfg.ConsumerKey))
	consumerSecret := interpolate(deps, model.DerefString(cfg.ConsumerSecret))
	token := interpolate(deps, model.DerefString(cfg.AccessToken))
	tokenSecret := interpolate(deps, model.DerefString(cfg.AccessTokenSecret))
	method := interpolate(deps, model.DerefString(cfg.SignatureMethod))
	if method == "" {
		method = oauth1SignatureHMACSHA1
	}
	version := model.DerefString(cfg.Version)
	if version == "" {
		version = oauth1DefaultVersion
	}
	callback := interpolate(deps, model.DerefString(cfg.CallbackURL))
	verifier := interpolate(deps, model.DerefString(cfg.Verifier))
	realm := interpolate(deps, model.DerefString(cfg.Realm))

	// Without a consumer key there is nothing to sign, matching Bruno's warning
	// path: the request goes out unauthenticated and the server explains itself.
	if consumerKey == "" {
		return Prepared{}, nil
	}
	if deps.Request == nil {
		return Prepared{}, errNoRequestForSigning
	}

	// Bruno lets the request pin a timestamp and nonce, which is what reproducible
	// test suites and replay capture need. Configured values win over generated.
	timestamp := interpolate(deps, model.DerefString(cfg.Timestamp))
	if timestamp == "" {
		timestamp = strconv.FormatInt(nowUnix(deps), 10)
	}
	nonce := interpolate(deps, model.DerefString(cfg.Nonce))
	if nonce == "" {
		nonce = oauth1Nonce(deps)
	}

	// Query parameters already on the URL take part in the signature, so they are
	// read from the request rather than from configuration.
	queryParams := parseOAuthQuery(deps.Request.URL.RawQuery)

	// RFC 5849 §3.4.1.3.1: a form-encoded body is signed as parameters. Bruno
	// extends this to GET/HEAD when the OAuth1 params go in the body, because it
	// still transmits a body for those methods.
	placement := oauth1Placement(cfg)
	isForm := oauth1IsFormEncoded(deps.Request.Header.Get("Content-Type"))
	var bodyParams [][2]string
	if isForm && (placement == oauth1PlacementBody || oauth1HasBody(deps.Request.Method)) {
		bodyParams = parseOAuthQuery(string(deps.Body))
	}

	params, err := oauth1Sign(oauth1Input{
		Method:          deps.Request.Method,
		URL:             deps.Request.URL,
		ConsumerKey:     consumerKey,
		ConsumerSecret:  consumerSecret,
		Token:           token,
		TokenSecret:     tokenSecret,
		SignatureMethod: method,
		Version:         version,
		Callback:        callback,
		Verifier:        verifier,
		Timestamp:       timestamp,
		Nonce:           nonce,
		QueryParams:     queryParams,
		BodyParams:      bodyParams,
		Realm:           realm,
		PrivateKeyPEM:   interpolate(deps, model.DerefString(cfg.PrivateKey)),
	})
	if err != nil {
		return Prepared{}, err
	}

	// draft-eaton-oauth-bodyhash-00 §3.2: oauth_body_hash is forbidden for
	// form-encoded bodies, so it only applies when the body is not signed.
	if model.DerefBool(cfg.IncludeBodyHash, false) && !isForm {
		params.pairs = append(params.pairs, [2]string{
			"oauth_body_hash", oauth1BodyHash(deps.Body, method),
		})
	}

	// Placement decides where the signed parameters travel. The header is the
	// RFC default; query and body exist for clients and APIs that cannot read it.
	switch placement {
	case oauth1PlacementQuery:
		qp := make([]QueryParam, 0, len(params.pairs))
		for _, kv := range params.pairs {
			qp = append(qp, QueryParam{Name: kv[0], Value: kv[1]})
		}
		return Prepared{QueryParams: qp}, nil

	case oauth1PlacementBody:
		// A non-form body cannot carry parameters, so it is replaced wholesale and
		// the content type is corrected, as Bruno does. Form params are re-emitted
		// with the oauth_* values merged in.
		body := oauth1RenderedBody(bodyParams, params.pairs)
		return Prepared{
			Headers: []Header{{Name: "Content-Type", Value: oauth1FormContentType}},
			Body:    body,
		}, nil

	default:
		return Prepared{Headers: []Header{{Name: "Authorization", Value: params.header()}}}, nil
	}
}

const (
	oauth1PlacementHeader = "header"
	oauth1PlacementQuery  = "query"
	oauth1PlacementBody   = "body"

	oauth1FormContentType = "application/x-www-form-urlencoded"
)

// oauth1Placement reads Placement, accepting the spelling Bruno uses for the
// OAuth1 editor ("query") as well as the API-key spelling ("queryparams").
func oauth1Placement(cfg *model.Auth) string {
	switch strings.ToLower(strings.TrimSpace(string(cfg.Placement))) {
	case oauth1PlacementQuery, "queryparams", "queryparamsandheaders":
		return oauth1PlacementQuery
	case oauth1PlacementBody:
		return oauth1PlacementBody
	}
	return oauth1PlacementHeader
}

// oauth1HasBody reports whether a method normally carries an entity body.
func oauth1HasBody(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead:
		return false
	}
	return true
}

// oauth1IsFormEncoded reports whether the content type is form urlencoded.
func oauth1IsFormEncoded(contentType string) bool {
	mediaType := contentType
	if i := strings.IndexByte(mediaType, ';'); i >= 0 {
		mediaType = mediaType[:i]
	}
	return strings.EqualFold(strings.TrimSpace(mediaType), oauth1FormContentType)
}

// oauth1RenderedBody serialises body parameters with the oauth_* set merged in.
//
// Existing body fields keep their order and are overwritten in place when an
// oauth parameter shares their name, matching URLSearchParams.set.
func oauth1RenderedBody(bodyParams, oauthParams [][2]string) []byte {
	type pair struct{ k, v string }
	var ordered []pair
	index := make(map[string]int)

	add := func(k, v string) {
		if i, ok := index[k]; ok {
			ordered[i].v = v
			return
		}
		index[k] = len(ordered)
		ordered = append(ordered, pair{k, v})
	}
	for _, kv := range bodyParams {
		add(kv[0], kv[1])
	}
	for _, kv := range oauthParams {
		add(kv[0], kv[1])
	}

	var b strings.Builder
	for i, p := range ordered {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.v))
	}
	return []byte(b.String())
}

// oauth1BodyHash computes the hashed body of draft-eaton-oauth-bodyhash-00.
//
// The digest follows the signature method's hash family, defaulting to SHA-1,
// which is what Bruno does; the body itself is hashed without a key.
func oauth1BodyHash(body []byte, signatureMethod string) string {
	var newHash func() hash.Hash
	switch strings.ToUpper(signatureMethod) {
	case "HMAC-SHA512", "RSA-SHA512":
		newHash = sha512.New
	case "HMAC-SHA256", "RSA-SHA256":
		newHash = sha256.New
	default:
		newHash = sha1.New
	}
	h := newHash()
	h.Write(body)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// oauth1BodyHash computes the hashed body of draft-eaton-oauth-bodyhash-00.
// oauth1Input is everything needed to sign.
type oauth1Input struct {
	Method          string
	URL             *url.URL
	ConsumerKey     string
	ConsumerSecret  string
	Token           string
	TokenSecret     string
	SignatureMethod string
	Version         string
	Callback        string
	Verifier        string
	Timestamp       string
	Nonce           string
	QueryParams     [][2]string
	BodyParams      [][2]string
	// PrivateKeyPEM is used only by the RSA signature methods.
	PrivateKeyPEM string
	// Realm prefixes the Authorization header; it is not signed.
	Realm string
}

// oauth1Result is the signed parameter set.
type oauth1Result struct {
	// pairs holds the oauth_* parameters in a stable order.
	pairs [][2]string
	// realm prefixes the header when set.
	realm string
}

// oauth1QuoteRealm escapes a realm for a quoted-string header value.
func oauth1QuoteRealm(realm string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(realm)
}

// header renders the Authorization value.
//
// Only oauth_* parameters go in the header; query and body parameters were
// signed but travel with the request itself.
func (r oauth1Result) header() string {
	var b strings.Builder
	b.WriteString("OAuth ")
	// RFC 5849 §3.5.1 allows a realm before the parameters; it is excluded from
	// the signature, so it can only be added here.
	if r.realm != "" {
		b.WriteString(`realm="`)
		b.WriteString(oauth1QuoteRealm(r.realm))
		b.WriteString(`", `)
	}
	for i, kv := range r.pairs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(oauth1PercentEncode(kv[0]))
		b.WriteString(`="`)
		b.WriteString(oauth1PercentEncode(kv[1]))
		b.WriteString(`"`)
	}
	return b.String()
}

// oauth1Sign computes the signature and returns the full parameter set.
func oauth1Sign(in oauth1Input) (oauth1Result, error) {
	oauthParams := [][2]string{
		{"oauth_consumer_key", in.ConsumerKey},
		{"oauth_nonce", in.Nonce},
		{"oauth_signature_method", in.SignatureMethod},
		{"oauth_timestamp", in.Timestamp},
		{"oauth_version", in.Version},
	}
	if in.Token != "" {
		oauthParams = append(oauthParams, [2]string{"oauth_token", in.Token})
	}
	if in.Callback != "" {
		oauthParams = append(oauthParams, [2]string{"oauth_callback", in.Callback})
	}
	if in.Verifier != "" {
		oauthParams = append(oauthParams, [2]string{"oauth_verifier", in.Verifier})
	}

	// The signature base string includes oauth params, query params and form body
	// params. Duplicates are preserved: RFC 5849 §3.4.1.3.2 signs every value.
	paramString := buildOAuthParamString(oauthParams, in.QueryParams, in.BodyParams)
	baseURL := oauth1BaseURL(in.URL)
	baseString := strings.ToUpper(in.Method) + "&" +
		oauth1PercentEncode(baseURL) + "&" +
		oauth1PercentEncode(paramString)

	signingKey := oauth1PercentEncode(in.ConsumerSecret) + "&" + oauth1PercentEncode(in.TokenSecret)

	signature, err := oauth1Signature(in.SignatureMethod, baseString, signingKey, in.PrivateKeyPEM)
	if err != nil {
		return oauth1Result{}, err
	}

	oauthParams = append(oauthParams, [2]string{"oauth_signature", signature})
	// Bruno emits the parameters in sorted order, which also keeps the header
	// byte-identical across runs for a pinned timestamp and nonce.
	sort.SliceStable(oauthParams, func(i, j int) bool {
		return oauthParams[i][0] < oauthParams[j][0]
	})
	return oauth1Result{pairs: oauthParams, realm: in.Realm}, nil
}

// buildOAuthParamString normalises and sorts all parameters (RFC 5849 §3.4.1.3.2).
func buildOAuthParamString(sets ...[][2]string) string {
	type pair struct{ k, v string }
	var collected []pair
	for _, set := range sets {
		for _, kv := range set {
			collected = append(collected, pair{oauth1PercentEncode(kv[0]), oauth1PercentEncode(kv[1])})
		}
	}
	sort.Slice(collected, func(i, j int) bool {
		if collected[i].k != collected[j].k {
			return collected[i].k < collected[j].k
		}
		return collected[i].v < collected[j].v
	})
	parts := make([]string, len(collected))
	for i, p := range collected {
		parts[i] = p.k + "=" + p.v
	}
	return strings.Join(parts, "&")
}

// oauth1BaseURL normalises the base string URI (RFC 5849 §3.4.1.2).
//
// Scheme and host are lower-cased, default ports are dropped, and the query and
// fragment are excluded, because those are signed as parameters instead.
func oauth1BaseURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()

	// A default port adds nothing; including it would change the signature.
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	var b strings.Builder
	b.WriteString(scheme)
	b.WriteString("://")
	b.WriteString(host)
	if port != "" {
		b.WriteByte(':')
		b.WriteString(port)
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	b.WriteString(path)
	return b.String()
}

// oauth1Signature computes the signature for a base string.
func oauth1Signature(method, baseString, signingKey, privateKeyPEM string) (string, error) {
	switch strings.ToUpper(method) {
	case "PLAINTEXT":
		// RFC 5849 §3.4.4: the signature is the signing key itself.
		return signingKey, nil

	case "HMAC-SHA1":
		return base64.StdEncoding.EncodeToString(hmacDigest(sha1.New, signingKey, baseString)), nil
	case "HMAC-SHA256":
		return base64.StdEncoding.EncodeToString(hmacDigest(sha256.New, signingKey, baseString)), nil
	case "HMAC-SHA512":
		return base64.StdEncoding.EncodeToString(hmacDigest(sha512.New, signingKey, baseString)), nil

	case "RSA-SHA1", "RSA-SHA256", "RSA-SHA512":
		return oauth1RSASignature(method, baseString, privateKeyPEM)
	}
	return "", fmt.Errorf("auth: unsupported OAuth1 signature method %q", method)
}

// hmacDigest computes an HMAC with the given hash constructor.
func hmacDigest(newHash func() hash.Hash, key, data string) []byte {
	mac := hmac.New(newHash, []byte(key))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// oauth1RSASignature signs the base string with an RSA private key.
func oauth1RSASignature(method, baseString, privateKeyPEM string) (string, error) {
	if privateKeyPEM == "" {
		return "", fmt.Errorf("auth: OAuth1 %s requires a private key", method)
	}
	key, err := parseRSAPrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return "", err
	}

	var h crypto.Hash
	switch strings.ToUpper(method) {
	case "RSA-SHA1":
		h = crypto.SHA1
	case "RSA-SHA256":
		h = crypto.SHA256
	case "RSA-SHA512":
		h = crypto.SHA512
	}

	hasher := h.New()
	hasher.Write([]byte(baseString))
	digest := hasher.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, key, h, digest)
	if err != nil {
		return "", fmt.Errorf("auth: RSA signing failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// parseRSAPrivateKey accepts PKCS#1 and PKCS#8 PEM blocks.
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("auth: no PEM block in private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: cannot parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("auth: private key is not RSA")
	}
	return key, nil
}

// oauth1PercentEncode implements RFC 5849 §3.6.
//
// It is stricter than url.QueryEscape: space becomes %20 (not "+"), and the
// characters ! * ' ( ) are encoded even though Go's QueryEscape leaves them.
func oauth1PercentEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			b.WriteByte(c)
		default:
			const hexDigits = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xF])
		}
	}
	return b.String()
}

// oauth1Nonce returns a random nonce.
//
// RFC 5849 only requires uniqueness per timestamp; 32 alphanumerics matches
// Bruno and gives ample margin. Randomness is injected so tests can pin it.
func oauth1Nonce(deps Deps) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32
	raw := randomBytes(deps, length)
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(alphabet[int(raw[i])%len(alphabet)])
	}
	return b.String()
}

// parseOAuthQuery parses query parameters, treating "+" as a literal character.
//
// url.ParseQuery decodes "+" as a space (the HTML form convention), but RFC 5849
// treats it literally, so the raw string is split by hand.
func parseOAuthQuery(rawQuery string) [][2]string {
	if rawQuery == "" {
		return nil
	}
	var out [][2]string
	for _, part := range strings.Split(rawQuery, "&") {
		if part == "" {
			continue
		}
		name := part
		value := ""
		if i := strings.IndexByte(part, '='); i >= 0 {
			name, value = part[:i], part[i+1:]
		}
		out = append(out, [2]string{unescapeQueryComponent(name), unescapeQueryComponent(value)})
	}
	return out
}

// unescapeQueryComponent decodes percent escapes but leaves "+" alone.
func unescapeQueryComponent(s string) string {
	s = strings.ReplaceAll(s, "+", "%2B")
	if decoded, err := url.PathUnescape(s); err == nil {
		return decoded
	}
	return s
}
