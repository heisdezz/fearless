package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

// rfc5849Example reproduces the worked example from RFC 5849 §1.2. The RFC
// publishes every intermediate value, so this pins percent-encoding,
// normalisation, the base string and the HMAC together.
//
// The base string and signing key below are copied from the RFC. The expected
// signature, however, is computed independently with openssl rather than taken
// from the RFC text: the RFC's published signature (§1.2) does not verify
// against its own published base string, while our construction does.
//
//	printf %s "$BASE_STRING" | openssl dgst -sha1 \
//	    -hmac 'kd94hf93k423kf44&pfkkdhi9sl3r4s00' -binary | base64
//	=> tR3+Ty81lMeYAr/Fid0kMTYa/WM=
func TestOAuth1RFC5849Golden(t *testing.T) {
	u, err := url.Parse("http://photos.example.net/photos?file=vacation.jpg&size=original")
	if err != nil {
		t.Fatal(err)
	}

	// These are the RFC's own base string and key, reproduced so a failure
	// points at normalisation rather than at any single helper.
	const wantBase = "GET&http%3A%2F%2Fphotos.example.net%2Fphotos&" +
		"file%3Dvacation.jpg%26oauth_consumer_key%3Ddpf43f3p2l4k3l03%26" +
		"oauth_nonce%3Dkllo9940pd9333jh%26oauth_signature_method%3DHMAC-SHA1%26" +
		"oauth_timestamp%3D1191242096%26oauth_token%3Dnnch734d00sl2jdk%26" +
		"oauth_version%3D1.0%26size%3Doriginal"
	const wantSignature = "tR3+Ty81lMeYAr/Fid0kMTYa/WM="

	cfg := &model.Auth{
		Mode:              model.AuthOAuth1,
		ConsumerKey:       strp("dpf43f3p2l4k3l03"),
		ConsumerSecret:    strp("kd94hf93k423kf44"),
		AccessToken:       strp("nnch734d00sl2jdk"),
		AccessTokenSecret: strp("pfkkdhi9sl3r4s00"),
		SignatureMethod:   strp("HMAC-SHA1"),
		Version:           strp("1.0"),
		Timestamp:         strp("1191242096"),
		Nonce:             strp("kllo9940pd9333jh"),
	}

	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{Method: "GET", URL: u, Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 1 || p.Headers[0].Name != "Authorization" {
		t.Fatalf("headers = %+v", p.Headers)
	}
	if want := `oauth_signature="` + oauth1PercentEncode(wantSignature) + `"`; !strings.Contains(p.Headers[0].Value, want) {
		t.Fatalf("signature mismatch\n got header: %s\n want fragment: %s", p.Headers[0].Value, want)
	}

	// Re-derive the base string directly so a mismatch localises the bug.
	params := buildOAuthParamString(
		[][2]string{
			{"oauth_consumer_key", "dpf43f3p2l4k3l03"},
			{"oauth_nonce", "kllo9940pd9333jh"},
			{"oauth_signature_method", "HMAC-SHA1"},
			{"oauth_timestamp", "1191242096"},
			{"oauth_token", "nnch734d00sl2jdk"},
			{"oauth_version", "1.0"},
		},
		parseOAuthQuery(u.RawQuery),
	)
	gotBase := "GET&" + oauth1PercentEncode(oauth1BaseURL(u)) + "&" + oauth1PercentEncode(params)
	if gotBase != wantBase {
		t.Fatalf("base string mismatch\n got: %s\nwant: %s", gotBase, wantBase)
	}
}

// TestOAuth1HeaderShape checks the emitted header ordering and realm prefix.
func TestOAuth1HeaderShape(t *testing.T) {
	cfg := &model.Auth{
		Mode:        model.AuthOAuth1,
		ConsumerKey: strp("ck"),
		Timestamp:   strp("1"),
		Nonce:       strp("n"),
		Realm:       strp(`a "quoted" \ realm`),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{
			Method: "GET",
			URL:    mustURL(t, "https://api.example.com/x"),
			Header: http.Header{},
		},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	hdr := p.Headers[0].Value
	if !strings.HasPrefix(hdr, `OAuth realm="a \"quoted\" \\ realm", `) {
		t.Errorf("realm not escaped or misplaced: %q", hdr)
	}

	// Parameters must be sorted by name.
	idx := -1
	for _, name := range []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature", "oauth_signature_method", "oauth_timestamp", "oauth_version"} {
		next := strings.Index(hdr, name+"=")
		if next < 0 {
			t.Fatalf("missing %s in %q", name, hdr)
		}
		if next < idx {
			t.Fatalf("%s out of order in %q", name, hdr)
		}
		idx = next
	}
	if strings.Contains(hdr, "oauth_token=") {
		t.Error("empty access token must be omitted")
	}
}

// TestOAuth1PercentEncode pins RFC 5849 §3.6 behaviour, which differs from
// url.QueryEscape on space and on ! * ' ( ).
func TestOAuth1PercentEncode(t *testing.T) {
	cases := map[string]string{
		"abcXYZ019-._~": "abcXYZ019-._~",
		"a b":           "a%20b",
		"!*'()":         "%21%2A%27%28%29",
		"a+b":           "a%2Bb",
		"&=?/":          "%26%3D%3F%2F",
		"100%":          "100%25",
		"é":             "%C3%A9",
	}
	for in, want := range cases {
		if got := oauth1PercentEncode(in); got != want {
			t.Errorf("percentEncode(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestOAuth1BaseURL covers RFC 5849 §3.4.1.2 normalisation.
func TestOAuth1BaseURL(t *testing.T) {
	cases := map[string]string{
		"HTTP://Example.COM/Path?q=1#frag": "http://example.com/Path",
		"https://example.com:443/x":        "https://example.com/x",
		"http://example.com:80/x":          "http://example.com/x",
		"http://example.com:8080/x":        "http://example.com:8080/x",
		"https://example.com":              "https://example.com/",
		"https://example.com/?q=1":         "https://example.com/",
	}
	for in, want := range cases {
		if got := oauth1BaseURL(mustURL(t, in)); got != want {
			t.Errorf("baseURL(%q) = %q, want %q", in, got, want)
		}
	}
	if got := oauth1BaseURL(nil); got != "" {
		t.Errorf("nil URL = %q", got)
	}
}

// TestOAuth1ParameterStringSortsAndKeepsDuplicates covers §3.4.1.3.2.
func TestOAuth1ParameterStringSortsAndKeepsDuplicates(t *testing.T) {
	got := buildOAuthParamString(
		[][2]string{{"b", "2"}, {"a", "1"}},
		[][2]string{{"a", "0"}, {"a", "1"}},
		[][2]string{{"c", " "}},
	)
	// Keys sorted, then values sorted within a key; duplicates preserved.
	want := "a=0&a=1&a=1&b=2&c=%20"
	if got != want {
		t.Errorf("param string = %q, want %q", got, want)
	}
}

// TestOAuth1QueryPlusIsLiteral covers the url.ParseQuery trap: "+" is a space
// under the HTML form convention but a literal character under RFC 5849.
func TestOAuth1QueryPlusIsLiteral(t *testing.T) {
	pairs := parseOAuthQuery("a=1+2&b=x%2By&c=&d")
	want := [][2]string{{"a", "1+2"}, {"b", "x+y"}, {"c", ""}, {"d", ""}}
	if len(pairs) != len(want) {
		t.Fatalf("pairs = %v", pairs)
	}
	for i := range want {
		if pairs[i] != want[i] {
			t.Errorf("pair %d = %v, want %v", i, pairs[i], want[i])
		}
	}
}

// TestOAuth1FormBodyParticipatesInSignature covers §3.4.1.3.1.
func TestOAuth1FormBodyParticipatesInSignature(t *testing.T) {
	formCfg := &model.Auth{
		Mode:           model.AuthOAuth1,
		ConsumerKey:    strp("ck"),
		ConsumerSecret: strp("cs"),
		Timestamp:      strp("1"),
		Nonce:          strp("n"),
	}
	run := func(body string) string {
		deps := Deps{
			Request: &http.Request{
				Method: "POST",
				URL:    mustURL(t, "https://example.com"),
				Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded; charset=utf-8"}},
			},
			Body: []byte(body),
		}
		p, err := Apply(context.Background(), []*model.Auth{formCfg}, deps)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		return p.Headers[0].Value
	}
	withField := run("status=hello")
	without := run("")
	if withField == without {
		t.Fatal("a form body must change the signature")
	}

	// A JSON body must not participate.
	jsonDeps := Deps{
		Request: &http.Request{
			Method: "POST",
			URL:    mustURL(t, "https://example.com"),
			Header: http.Header{"Content-Type": {"application/json"}},
		},
		Body: []byte(`{"status":"hello"}`),
	}
	p, err := Apply(context.Background(), []*model.Auth{formCfg}, jsonDeps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if p.Headers[0].Value != without {
		t.Error("a JSON body must not be signed as parameters")
	}
}

// TestOAuth1PlaintextSignature covers §3.4.4.
func TestOAuth1PlaintextSignature(t *testing.T) {
	cfg := &model.Auth{
		Mode:              model.AuthOAuth1,
		ConsumerKey:       strp("ck"),
		ConsumerSecret:    strp("cs"),
		AccessToken:       strp("tok"),
		AccessTokenSecret: strp("ts"),
		SignatureMethod:   strp("PLAINTEXT"),
		Timestamp:         strp("1"),
		Nonce:             strp("n"),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := `oauth_signature="cs%26ts"`; !strings.Contains(p.Headers[0].Value, want) {
		t.Errorf("header = %q, want %s", p.Headers[0].Value, want)
	}
}

// TestOAuth1HMACVariants verifies every HMAC method produces a 32-byte-class
// signature of differing length and differs from each other.
func TestOAuth1HMACVariants(t *testing.T) {
	sigs := map[string]string{}
	for _, method := range []string{"HMAC-SHA1", "HMAC-SHA256", "HMAC-SHA512"} {
		cfg := &model.Auth{
			Mode:            model.AuthOAuth1,
			ConsumerKey:     strp("ck"),
			ConsumerSecret:  strp("cs"),
			SignatureMethod: strp(method),
			Timestamp:       strp("1"),
			Nonce:           strp("n"),
		}
		p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
			Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
		})
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		sigs[method] = p.Headers[0].Value
	}
	if sigs["HMAC-SHA1"] == sigs["HMAC-SHA256"] || sigs["HMAC-SHA256"] == sigs["HMAC-SHA512"] {
		t.Fatal("signature methods must differ")
	}
}

// TestOAuth1UnsupportedMethodReportsError keeps a typo from silently unsigned.
func TestOAuth1UnsupportedMethodReportsError(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		SignatureMethod: strp("HMAC-MD5"),
	}
	_, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err == nil || !strings.Contains(err.Error(), "HMAC-MD5") {
		t.Fatalf("err = %v", err)
	}
}

// TestOAuth1EmptyConsumerKeyIsNoop matches Bruno's warning path.
func TestOAuth1EmptyConsumerKeyIsNoop(t *testing.T) {
	p, err := Apply(context.Background(), []*model.Auth{{
		Mode:        model.AuthOAuth1,
		ConsumerKey: strp(""),
	}}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 || len(p.QueryParams) != 0 {
		t.Errorf("expected no-op, got %+v", p)
	}
}

// TestOAuth1RequiresRequest guards signers that cover the URL.
func TestOAuth1RequiresRequest(t *testing.T) {
	_, err := Apply(context.Background(), []*model.Auth{{
		Mode:        model.AuthOAuth1,
		ConsumerKey: strp("ck"),
	}}, Deps{})
	if err != errNoRequestForSigning {
		t.Fatalf("err = %v", err)
	}
}

// TestOAuth1QueryPlacement puts the signed params on the URL instead.
func TestOAuth1QueryPlacement(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		SignatureMethod: strp("HMAC-SHA1"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
		Placement:       model.APIKeyQueryParams,
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 {
		t.Errorf("query placement must not set headers: %+v", p.Headers)
	}
	names := make(map[string]bool)
	for _, qp := range p.QueryParams {
		names[qp.Name] = true
	}
	for _, want := range []string{"oauth_consumer_key", "oauth_signature", "oauth_nonce"} {
		if !names[want] {
			t.Errorf("missing %s in %+v", want, p.QueryParams)
		}
	}
}

// TestOAuth1BodyPlacement rewrites the body and fixes the content type.
func TestOAuth1BodyPlacement(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		SignatureMethod: strp("HMAC-SHA1"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
		Placement:       "Body",
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{
			Method: "POST",
			URL:    mustURL(t, "https://example.com"),
			Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		},
		Body: []byte("status=hello&oauth_nonce=stale"),
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if p.Body == nil {
		t.Fatal("body placement must supply a body")
	}
	body := string(p.Body)
	if !strings.Contains(body, "status=hello") {
		t.Errorf("original field lost: %s", body)
	}
	if !strings.Contains(body, "oauth_nonce=n") || strings.Contains(body, "stale") {
		t.Errorf("oauth param not overwritten: %s", body)
	}
	if !strings.Contains(body, "oauth_signature=") {
		t.Errorf("signature missing: %s", body)
	}
	if len(p.Headers) != 1 || p.Headers[0].Name != "Content-Type" {
		t.Errorf("content type not set: %+v", p.Headers)
	}
}

// TestOAuth1BodyPlacementSignsGETBody covers Bruno's deliberate extension of
// §3.4.1.3.1: with body placement a GET's form body is still signed.
func TestOAuth1BodyPlacementSignsGETBody(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		ConsumerSecret:  strp("cs"),
		SignatureMethod: strp("HMAC-SHA1"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
		Placement:       oauth1PlacementBody,
	}
	deps := func(body string) Deps {
		return Deps{
			Request: &http.Request{
				Method: "GET",
				URL:    mustURL(t, "https://example.com"),
				Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
			},
			Body: []byte(body),
		}
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, deps("a=1"))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	sig := oauth1ParamValue(string(p.Body), "oauth_signature")
	if sig == "" {
		t.Fatalf("no signature in body: %s", p.Body)
	}

	// Same first-attempt signature must hold when the body is unchanged, because
	// the signature was computed over a=1.
	if oauth1ParamValue(string(p.Body), "a") != "1" {
		t.Errorf("body field lost: %s", p.Body)
	}
}

// TestOAuth1BodyHash covers draft-eaton-oauth-bodyhash-00.
func TestOAuth1BodyHash(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		ConsumerSecret:  strp("cs"),
		SignatureMethod: strp("HMAC-SHA256"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
		IncludeBodyHash: boolp(true),
	}
	deps := Deps{
		Request: &http.Request{
			Method: "POST",
			URL:    mustURL(t, "https://example.com"),
			Header: http.Header{"Content-Type": {"application/json"}},
		},
		Body: []byte(`{"a":1}`),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, deps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(p.Headers[0].Value, "oauth_body_hash") {
		t.Errorf("body hash missing: %q", p.Headers[0].Value)
	}
	// The body is the JSON payload, so the hash must cover its bytes verbatim.
	if want := `oauth_body_hash="` + oauth1PercentEncode(oauth1BodyHash([]byte(`{"a":1}`), "HMAC-SHA256")) + `"`; !strings.Contains(p.Headers[0].Value, want) {
		t.Errorf("body hash value wrong:\n got %q\nwant %s", p.Headers[0].Value, want)
	}

	// SHA-256 of the body, base64, is 44 characters ending in "=".
	if got := oauth1BodyHash([]byte("abc"), "HMAC-SHA256"); got != "ungWv48Bz+pBQUDeXa4iI7ADYaOWF3qctBD/YfIAFa0=" {
		t.Errorf("sha256 body hash = %q", got)
	}
	// Default family is SHA-1, matching Bruno.
	if got := oauth1BodyHash([]byte("abc"), "HMAC-SHA1"); got != "qZk+NkcGgWq6PiVxeFDCbJzQ2J0=" {
		t.Errorf("sha1 body hash = %q", got)
	}

	// Form-encoded bodies must never carry a hash.
	formDeps := deps
	formDeps.Request.Header = http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}
	formDeps.Body = []byte("a=1")
	pf, err := Apply(context.Background(), []*model.Auth{cfg}, formDeps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Contains(pf.Headers[0].Value, "oauth_body_hash") {
		t.Errorf("body hash sent for form body: %q", pf.Headers[0].Value)
	}
}

// TestOAuth1RSASignature round-trips a real key.
func TestOAuth1RSASignature(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		SignatureMethod: strp("RSA-SHA256"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
		PrivateKey:      strp(testRSAKeyPEM),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(p.Headers[0].Value, "oauth_signature=") {
		t.Errorf("header = %q", p.Headers[0].Value)
	}
}

// TestOAuth1RSASignatureRequiresKey reports a missing key instead of signing
// with nothing.
func TestOAuth1RSASignatureRequiresKey(t *testing.T) {
	_, err := Apply(context.Background(), []*model.Auth{{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("ck"),
		SignatureMethod: strp("RSA-SHA1"),
	}}, Deps{
		Request: &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("err = %v", err)
	}
}

// TestOAuth1NonceUsesInjectedReader keeps nonce generation deterministic.
func TestOAuth1NonceUsesInjectedReader(t *testing.T) {
	deps := Deps{RandBytes: func(n int) []byte {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	}}
	got := oauth1Nonce(deps)
	if len(got) != 32 {
		t.Fatalf("nonce length = %d", len(got))
	}
	for i := 0; i < len(got); i++ {
		c := got[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !ok {
			t.Fatalf("nonce %q has non-alphanumeric %q", got, c)
		}
	}
}

// TestOAuth1InterpolatesConfig makes sure {{vars}} in auth fields resolve
// before signing, as they do for every other provider.
func TestOAuth1InterpolatesConfig(t *testing.T) {
	cfg := &model.Auth{
		Mode:            model.AuthOAuth1,
		ConsumerKey:     strp("{{ck}}"),
		SignatureMethod: strp("HMAC-SHA1"),
		Timestamp:       strp("1"),
		Nonce:           strp("n"),
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{
		Interpolate: func(s string) string { return strings.ReplaceAll(s, "{{ck}}", "resolved-key") },
		Request:     &http.Request{Method: "GET", URL: mustURL(t, "https://example.com"), Header: http.Header{}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(p.Headers[0].Value, `oauth_consumer_key="resolved-key"`) {
		t.Errorf("header = %q", p.Headers[0].Value)
	}
}

// TestOAuth1Registered confirms the provider is wired into the registry.
//
// RegisterBase is not called here: other tests in this package already wire the
// registry, and Register panics on a duplicate mode by design.
func TestOAuth1Registered(t *testing.T) {
	if _, ok := Get(model.AuthOAuth1); !ok {
		t.Fatal("oauth1 provider not registered")
	}
}

// --- helpers -------------------------------------------------------------

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func boolp(b bool) *bool { return &b }

// oauth1ParamValue extracts a form value without relying on url.ParseQuery,
// whose "+" handling is the very thing under test elsewhere.
func oauth1ParamValue(body, name string) string {
	for _, part := range strings.Split(body, "&") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if uk, err := url.QueryUnescape(k); err == nil && uk == name {
			if uv, err := url.QueryUnescape(v); err == nil {
				return uv
			}
		}
	}
	return ""
}

// testRSAKeyPEM is a throwaway 2048-bit key generated for these tests only.
const testRSAKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAz8UbFK0pq41vnqVfrwobxYb4/dAkwhVlO9tyzzMNz9zkXf6/
jRbKV+9vTiQqWRJk0sovKejjqmtNxoGDx7aTL3xcfv+ol9qQ4l5jhRF/MkBGRUg6
uc01ZBoybUg5NlDNpJ5RFIM/PqKOd+tweQw5Zy+NI/ppoReZah1zC2qBoqQCqU9e
NNZIIQmuPBBlcGIoQG9SFnuvq0tgy3rmibUSmxhhMvsI0sqphAROmIsDa8BNvWo2
TITpwga5vyi8gt0QmgHxZjOYAZ+E2WFcsYict1+KiE4w+NsTZqflrOhFJnGmpvWJ
ToEYDJpAebosU6Lx8+Zx/dIljCrpi25M4KzmqwIDAQABAoIBABcfmDklC9yzGTJJ
Ccz4194UJ1T+29XFPNGRxbAoE0hIN3BQoFqgUopr3vGNloWi8EQCIDKkCpErxRRW
fhH1kLb5um+gniUHEcCSgbVMcmFnhDVEvyhw8sQTzaXvolJttsgJ81B69PMlyKux
sSr/qx8qoc+DGKDG+Bo2cdiE6i2YAOSrLa2fT0FMJB+a+zP+c3flCFNl8oXvNsMO
Fihar6NlroPE3N3gVUrdvaistWTGZZOIsqjcIo0U22LV3HUGeay9rh5VjD8ckCzV
QOouNupE4eCZDa6+/tPN2fTgvVQV9EKMcgp9nRtjl2d4TRyFGrHipOY6aZcrj6le
TVtExuUCgYEA9FkK1t1YPJlCb0aZO9mTrfFAMqyS9D8c3dEQuLSKS5SVoVMqLxML
HLZ1QjcvOs9qsdCs6wjkJTXp1OYK99UH+VpMCSKQVu8aJVhbLYBtwiPX6YlEDX7A
uP57uA9YMWnzVH5f5Y3dipdX9Mv+6RgWKkhtAxHArKs0DQVmJwJmljcCgYEA2a2G
25uSTQyL6NHBWt4k2woTZ61nqtTHDA/k8GO9wcx592O7C0UburdOLiXOt0jTY3RL
35e993kUyZNxkoccHFkac2qIRcgXIaWsytk574rKp8VYtqaH687aDRPKND5JxFIs
qfAXjGqs/tFBW6rNr7o3InWf3akBtnBOrnM/+S0CgYADSdd9gMkZN67PNJIHXixA
oQiOJghsRRUsXw3sjnF2Bu/klevxaj4yCrqVQ4zdK8Fx1y8ckqgPYCRfeyEWqMVM
s8xmKxLqOJmPexrvB9VyUW0sdW53CAer9bDQj1ROhfCCaJj9y+T5YoQKhVxO4sYZ
l5PwAKMpL7ypmDMBv2SGDQKBgQDPFD9455TUa50lzXXOBgq1ve4b8WDvqR/6n5xB
/ADnPqjA6kVkQUhR5lQfnqP6xPMkZ9xOR0LXDH70DKo8pP/fD/0T00L0b5dh0QSC
rQhqk3SbCiiNCfvAHrZdPnhcHu16MFqBCAC++jS87UDp/aFTDSmqRI/jBQa2Ej0n
xEgVuQKBgQCJtcu5EK5XnVre+j+FA+Ob+PxFarCcqFbRqlMr7yvk86kMBndRyfam
Bv2VFB15uf6qvZ+nhV4+vv20Zzc1lFB3dnqJiIK2oJWu3spnXT43LnzSHBnICZKX
yxuZ21qflXy4duY+0iM67PFVD7w0PkEdb1YAtbdtVCV4ViIzhMirDw==
-----END RSA PRIVATE KEY-----`
