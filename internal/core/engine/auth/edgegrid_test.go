package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

// edgeGridCreds are fixed credentials for deterministic tests.
const (
	egAccessToken  = "akab-access"
	egClientToken  = "akab-client"
	egClientSecret = "akab-secret"
	egTimestamp    = "20240101T00:00:00+0000"
	egNonce        = "0f8fad5b-d9cb-469f-a165-70867728950e"
)

func egInput(t *testing.T, method, rawURL string, body []byte) edgeGridInput {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return edgeGridInput{
		Method:       method,
		RequestURL:   u,
		Headers:      map[string][]string{},
		Body:         body,
		AccessToken:  egAccessToken,
		ClientToken:  egClientToken,
		ClientSecret: egClientSecret,
		Timestamp:    egTimestamp,
		Nonce:        egNonce,
		MaxBodySize:  edgeGridDefaultMaxBodySize,
	}
}

// TestEdgeGridDataToSignShape pins the exact tab-joined layout.
//
// The order is method, scheme, host, path+query, signed headers, content hash,
// auth header. A single misplaced field changes every signature, so this checks
// the string itself rather than only the final digest.
func TestEdgeGridDataToSignShape(t *testing.T) {
	in := egInput(t, "GET", "https://akab-host.example.com/identity/v1/sites?limit=2", nil)
	res, err := edgeGridSign(in)
	if err != nil {
		t.Fatal(err)
	}

	wantAuthHeader := "EG1-HMAC-SHA256 client_token=akab-client;access_token=akab-access;" +
		"timestamp=20240101T00:00:00+0000;nonce=0f8fad5b-d9cb-469f-a165-70867728950e;"
	if res.AuthHeader != wantAuthHeader {
		t.Errorf("auth header = %q", res.AuthHeader)
	}

	wantFields := []string{
		"GET",
		"https",
		"akab-host.example.com",
		"/identity/v1/sites?limit=2",
		"", // no signed headers
		"", // GET has no content hash
		wantAuthHeader,
	}
	want := strings.Join(wantFields, "\t")
	if res.DataToSign != want {
		t.Errorf("data to sign mismatch\n got %q\nwant %q", res.DataToSign, want)
	}

	// The auth header is the last field, which is easy to get wrong.
	if !strings.HasSuffix(res.DataToSign, "\t"+wantAuthHeader) {
		t.Error("the auth header must be the last field of the data to sign")
	}
}

// TestEdgeGridSigningKeyIsBase64String checks the double-HMAC construction.
//
// The signing key is base64(HMAC(timestamp, secret)) *as text*, used as the key
// for the second HMAC. Using the raw digest instead is the classic mistake, so
// both readings are computed and compared.
func TestEdgeGridSigningKeyIsBase64String(t *testing.T) {
	in := egInput(t, "GET", "https://example.com/", nil)
	res, err := edgeGridSign(in)
	if err != nil {
		t.Fatal(err)
	}

	keyMac := hmac.New(sha256.New, []byte(egClientSecret))
	keyMac.Write([]byte(egTimestamp))
	wantKey := base64.StdEncoding.EncodeToString(keyMac.Sum(nil))
	if res.SigningKey != wantKey {
		t.Fatalf("signing key = %q, want %q", res.SigningKey, wantKey)
	}

	// Correct: key is the base64 text.
	sigMac := hmac.New(sha256.New, []byte(wantKey))
	sigMac.Write([]byte(res.DataToSign))
	wantSig := base64.StdEncoding.EncodeToString(sigMac.Sum(nil))
	if res.Signature != wantSig {
		t.Errorf("signature = %q, want %q", res.Signature, wantSig)
	}

	// Wrong: key is the raw digest. The two must not coincide.
	rawMac := hmac.New(sha256.New, keyMac.Sum(nil))
	rawMac.Write([]byte(res.DataToSign))
	wrongSig := base64.StdEncoding.EncodeToString(rawMac.Sum(nil))
	if res.Signature == wrongSig {
		t.Fatal("the signature must use the base64 key text, not the raw digest")
	}

	if !strings.HasSuffix(res.Authorization, "signature="+res.Signature) {
		t.Errorf("authorization = %q", res.Authorization)
	}
}

// TestEdgeGridKnownVector recomputes the whole chain independently from the fixed
// inputs and compares against the implementation.
func TestEdgeGridKnownVector(t *testing.T) {
	in := egInput(t, "POST", "https://example.com/api/v1/items?q=1", []byte(`{"name":"x"}`))
	in.Headers = map[string][]string{
		"Content-Type": {"application/json"},
		"X-Custom":     {"  spaced   out  "},
	}
	in.HeadersToSign = "Content-Type,X-Custom"

	res, err := edgeGridSign(in)
	if err != nil {
		t.Fatal(err)
	}

	// Content hash: POST bodies are hashed, truncated to the window.
	bodyHash := sha256.Sum256([]byte(`{"name":"x"}`))
	wantHash := base64.StdEncoding.EncodeToString(bodyHash[:])

	wantFields := []string{
		"POST",
		"https",
		"example.com",
		"/api/v1/items?q=1",
		// Config order preserved, values trimmed and whitespace collapsed.
		"content-type:application/json\tx-custom:spaced out",
		wantHash,
		res.AuthHeader,
	}
	if want := strings.Join(wantFields, "\t"); res.DataToSign != want {
		t.Fatalf("data to sign\n got %q\nwant %q", res.DataToSign, want)
	}
}

// TestEdgeGridSignedHeaderOrderIsConfigOrder is the trap: alphabetical sorting
// would look correct and produce a signature Akamai rejects.
func TestEdgeGridSignedHeaderOrderIsConfigOrder(t *testing.T) {
	headers := map[string][]string{
		"Zebra": {"z"},
		"Alpha": {"a"},
	}
	// Config order is Zebra then Alpha; alphabetical would be Alpha then Zebra.
	got := edgeGridCanonicalizeHeaders("Zebra,Alpha", headers)
	want := "zebra:z\talpha:a"
	if got != want {
		t.Errorf("canonical headers = %q, want %q", got, want)
	}
}

// TestEdgeGridCanonicalizeHeadersCases covers the individual rules.
func TestEdgeGridCanonicalizeHeadersCases(t *testing.T) {
	headers := map[string][]string{
		"Accept":      {"text/plain"},
		"X-Multi":     {"one", "two"},
		"Host":        {"api.example.com"},
		"Empty-Value": {""},
	}

	cases := []struct {
		name          string
		headersToSign string
		want          string
	}{
		{"empty config", "", ""},
		{"whitespace config", "   ", ""},
		{"lowercased name", "ACCEPT", "accept:text/plain"},
		{"spaces around comma", "Accept , X-Multi", "accept:text/plain\tx-multi:one, two"},
		{"multi-valued header joined", "X-Multi", "x-multi:one, two"},
		{"absent header skipped", "Accept, Missing", "accept:text/plain"},
		{"empty value kept", "Empty-Value", "empty-value:"},
		{"blank entry ignored", "Accept,,", "accept:text/plain"},
		// Case-insensitive lookup: config says `host`, the map holds `Host`.
		{"case-insensitive match", "host", "host:api.example.com"},
	}
	for _, tc := range cases {
		if got := edgeGridCanonicalizeHeaders(tc.headersToSign, headers); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestEdgeGridContentHashRules covers which methods and bodies are hashed.
func TestEdgeGridContentHashRules(t *testing.T) {
	body := []byte("hello")

	if got := edgeGridContentHash("GET", body, 100); got != "" {
		t.Errorf("GET must not hash a body, got %q", got)
	}
	if got := edgeGridContentHash("PUT", body, 100); got != "" {
		t.Errorf("only POST hashes a body, got %q", got)
	}
	if got := edgeGridContentHash("POST", nil, 100); got != "" {
		t.Errorf("an empty body hashes to the empty string, got %q", got)
	}
	if got := edgeGridContentHash("post", body, 100); got == "" {
		t.Error("the method check is case-insensitive")
	}

	// Truncation: only the first maxBodySize bytes are hashed.
	full := sha256.Sum256([]byte("he"))
	wantTruncated := base64.StdEncoding.EncodeToString(full[:])
	if got := edgeGridContentHash("POST", body, 2); got != wantTruncated {
		t.Errorf("truncated hash = %q, want %q", got, wantTruncated)
	}

	// A non-positive window falls back to the default rather than hashing nothing.
	if got := edgeGridContentHash("POST", body, 0); got == "" {
		t.Error("a zero max body size must fall back to the default")
	}
}

// TestEdgeGridBaseURLOverridesAuthority covers the staging-network case.
func TestEdgeGridBaseURLOverridesAuthority(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{"full base", "https://staging.example.net", "https://staging.example.net/api/x?q=1"},
		// A scheme-less base borrows the request's scheme; without this, `url.Parse`
		// reads "localhost:" as a protocol and leaves the host empty.
		{"scheme-less base", "localhost:6000", "http://localhost:6000/api/x?q=1"},
		{"base with a path is ignored", "https://b.example.com/ignored", "https://b.example.com/api/x?q=1"},
	}
	for _, tc := range cases {
		scheme, host, pathAndQuery, err := edgeGridURLToSign(
			mustParse(t, "http://original.example.com/api/x?q=1"), tc.baseURL)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := scheme + "://" + host + pathAndQuery
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestEdgeGridURLWithoutHostIsRejected avoids signing an empty authority.
func TestEdgeGridURLWithoutHostIsRejected(t *testing.T) {
	if _, _, _, err := edgeGridURLToSign(mustParse(t, "/relative/path"), ""); err == nil {
		t.Fatal("expected an error")
	}
}

// TestEdgeGridEmptyPathBecomesSlash matches HTTP's request-target rule.
func TestEdgeGridEmptyPathBecomesSlash(t *testing.T) {
	_, _, pathAndQuery, err := edgeGridURLToSign(mustParse(t, "https://example.com"), "")
	if err != nil {
		t.Fatal(err)
	}
	if pathAndQuery != "/" {
		t.Errorf("path = %q, want /", pathAndQuery)
	}
}

// TestEdgeGridProviderWiring covers Prepare and the credential guard.
func TestEdgeGridProviderWiring(t *testing.T) {
	if _, ok := Get(model.AuthEdgeGrid); !ok {
		t.Fatal("edgegrid provider not registered")
	}

	// Missing any credential contributes nothing.
	partial := &model.Auth{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp(egAccessToken),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: nil,
	}
	p, err := Apply(context.Background(), []*model.Auth{partial}, Deps{
		Request: &http.Request{Method: "GET", URL: mustParse(t, "https://example.com")},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 {
		t.Errorf("partial credentials must not sign, got %+v", p.Headers)
	}

	// The literal "undefined" is treated as absent, matching Bruno.
	undefined := &model.Auth{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp("undefined"),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: strp(egClientSecret),
	}
	p, err = Apply(context.Background(), []*model.Auth{undefined}, Deps{
		Request: &http.Request{Method: "GET", URL: mustParse(t, "https://example.com")},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 {
		t.Error("the string \"undefined\" must count as absent")
	}
}

// TestEdgeGridProviderProducesHeader covers the happy path through Apply.
func TestEdgeGridProviderProducesHeader(t *testing.T) {
	cfg := &model.Auth{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp(egAccessToken),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: strp(egClientSecret),
		Timestamp:            strp(egTimestamp),
		Nonce:                strp(egNonce),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{
			Method: "GET",
			URL:    mustParse(t, "https://example.com/x"),
			Header: http.Header{},
		},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 1 || p.Headers[0].Name != "Authorization" {
		t.Fatalf("headers = %+v", p.Headers)
	}
	if !strings.HasPrefix(p.Headers[0].Value, "EG1-HMAC-SHA256 client_token=akab-client;") {
		t.Errorf("header = %q", p.Headers[0].Value)
	}
	if !strings.Contains(p.Headers[0].Value, "signature=") {
		t.Errorf("header has no signature: %q", p.Headers[0].Value)
	}
}

// TestEdgeGridProviderDefaultsTimestampAndNonce covers generation.
func TestEdgeGridProviderDefaultsTimestampAndNonce(t *testing.T) {
	cfg := &model.Auth{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp(egAccessToken),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: strp(egClientSecret),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		// 2024-01-01T00:00:00Z
		Now: func() int64 { return 1704067200 },
		RandBytes: func(n int) []byte {
			b := make([]byte, n)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		},
		Request: &http.Request{Method: "GET", URL: mustParse(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	hdr := p.Headers[0].Value

	if !strings.Contains(hdr, "timestamp=20240101T00:00:00+0000") {
		t.Errorf("timestamp format wrong: %q", hdr)
	}
	// A UUID v4 nonce: 8-4-4-4-12 with version 4 and a `8`/`9`/`a`/`b` variant nibble.
	nonce := edgeGridNonceFromHeader(t, hdr)
	if len(nonce) != 36 {
		t.Fatalf("nonce = %q, want a UUID", nonce)
	}
	if nonce[14] != '4' {
		t.Errorf("nonce version nibble = %c, want 4", nonce[14])
	}
	if !strings.ContainsRune("89ab", rune(nonce[19])) {
		t.Errorf("nonce variant nibble = %c", nonce[19])
	}
}

// TestEdgeGridProviderRequiresRequest guards signers that cover the URL.
func TestEdgeGridProviderRequiresRequest(t *testing.T) {
	_, err := Apply(context.Background(), []*model.Auth{{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp(egAccessToken),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: strp(egClientSecret),
	}}, Deps{})
	if err != errNoRequestForSigning {
		t.Fatalf("err = %v", err)
	}
}

// TestEdgeGridProviderInterpolates covers templated credentials.
func TestEdgeGridProviderInterpolates(t *testing.T) {
	p, err := Apply(context.Background(), []*model.Auth{{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp("{{access}}"),
		EdgeGridClientToken:  strp("{{client}}"),
		EdgeGridClientSecret: strp("{{secret}}"),
		Timestamp:            strp(egTimestamp),
		Nonce:                strp(egNonce),
	}}, Deps{
		Interpolate: func(s string) string {
			return strings.NewReplacer(
				"{{access}}", egAccessToken,
				"{{client}}", egClientToken,
				"{{secret}}", egClientSecret,
			).Replace(s)
		},
		Request: &http.Request{Method: "GET", URL: mustParse(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(p.Headers[0].Value, "client_token=akab-client;access_token=akab-access;") {
		t.Errorf("credentials not interpolated: %q", p.Headers[0].Value)
	}
}

// TestEdgeGridMaxBodySizeFromConfig covers the override.
func TestEdgeGridMaxBodySizeFromConfig(t *testing.T) {
	limit := 2
	cfg := &model.Auth{
		Mode:                 model.AuthEdgeGrid,
		AccessToken:          strp(egAccessToken),
		EdgeGridClientToken:  strp(egClientToken),
		EdgeGridClientSecret: strp(egClientSecret),
		Timestamp:            strp(egTimestamp),
		Nonce:                strp(egNonce),
		EdgeGridMaxBodySize:  &limit,
	}
	body := []byte("hello world")
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{
			Method: "POST",
			URL:    mustParse(t, "https://example.com"),
			Header: http.Header{},
		},
		Body: body,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	full := sha256.Sum256(body)
	fullHash := base64.StdEncoding.EncodeToString(full[:])
	truncated := sha256.Sum256(body[:2])
	truncHash := base64.StdEncoding.EncodeToString(truncated[:])

	// The signed data is not in the header, so re-sign and compare.
	res, err := edgeGridSign(edgeGridInput{
		Method: "POST", RequestURL: mustParse(t, "https://example.com"),
		Headers:      map[string][]string{},
		Body:         body,
		AccessToken:  egAccessToken,
		ClientToken:  egClientToken,
		ClientSecret: egClientSecret,
		Timestamp:    egTimestamp,
		Nonce:        egNonce,
		MaxBodySize:  limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.DataToSign, truncHash) {
		t.Errorf("expected the truncated hash %q in %q", truncHash, res.DataToSign)
	}
	if strings.Contains(res.DataToSign, fullHash) {
		t.Error("the signature must cover only the truncated body")
	}
	if p.Headers[0].Value != res.Authorization {
		t.Error("the provider and the signer disagree")
	}
}

// --- helpers -------------------------------------------------------------

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

// edgeGridNonceFromHeader extracts the nonce from an Authorization value.
func edgeGridNonceFromHeader(t *testing.T, hdr string) string {
	t.Helper()
	const marker = "nonce="
	i := strings.Index(hdr, marker)
	if i < 0 {
		t.Fatalf("no nonce in %q", hdr)
	}
	rest := hdr[i+len(marker):]
	if j := strings.IndexByte(rest, ';'); j >= 0 {
		return rest[:j]
	}
	return rest
}
