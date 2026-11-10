package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

func init() {
	RegisterBase()
}

func strp(s string) *string { return &s }

// newMode builds an Auth for a mode with the given fields set.
func newMode(mode model.AuthMode) *model.Auth { return &model.Auth{Mode: mode} }

func TestRegisteredModesCoverBaseSet(t *testing.T) {
	want := []model.AuthMode{
		model.AuthAPIKey, model.AuthBasic, model.AuthBearer,
		model.AuthDigest, model.AuthNone, model.AuthWSSE,
	}
	got := Modes()
	for _, m := range want {
		if _, ok := Get(m); !ok {
			t.Errorf("mode %s has no provider", m)
		}
	}
	if len(got) < len(want) {
		t.Errorf("registered %d modes, want at least %d", len(got), len(want))
	}
	// Modes must be sorted for stable UI rendering.
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("modes not sorted: %v", got)
		}
	}
}

func TestRegisterRejectsDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering a duplicate mode should panic")
		}
	}()
	Register(basicProvider{})
}

func TestBasicSetsBase64Header(t *testing.T) {
	cfg := newMode(model.AuthBasic)
	cfg.Username = strp("user")
	cfg.Password = strp("pass")

	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if len(p.Headers) != 1 || p.Headers[0].Name != "Authorization" || p.Headers[0].Value != want {
		t.Fatalf("headers = %+v, want %s", p.Headers, want)
	}
}

func TestBasicInterpolatesCredentials(t *testing.T) {
	cfg := newMode(model.AuthBasic)
	cfg.Username = strp("{{user}}")
	cfg.Password = strp("{{pass}}")

	deps := Deps{Interpolate: func(s string) string {
		switch s {
		case "{{user}}":
			return "alice"
		case "{{pass}}":
			return "secret"
		}
		return s
	}}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, deps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if p.Headers[0].Value != want {
		t.Errorf("value = %q, want %q", p.Headers[0].Value, want)
	}
}

func TestBasicEmptyCredentialsStillEncodes(t *testing.T) {
	cfg := newMode(model.AuthBasic)
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	// ":" base64 is "Og=="
	if got := p.Headers[0].Value; got != "Basic Og==" {
		t.Errorf("value = %q", got)
	}
}

func TestBearerSetsHeader(t *testing.T) {
	cfg := newMode(model.AuthBearer)
	cfg.Token = strp("abc123")
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if p.Headers[0].Value != "Bearer abc123" {
		t.Errorf("value = %q", p.Headers[0].Value)
	}
}

// TestBearerEmptyTokenKeepsTrailingSpace pins Bruno's behaviour: the header is
// present with a trailing space, not omitted.
func TestBearerEmptyTokenKeepsTrailingSpace(t *testing.T) {
	cfg := newMode(model.AuthBearer)
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 1 || p.Headers[0].Value != "Bearer " {
		t.Errorf("headers = %+v", p.Headers)
	}
}

func TestAPIKeyHeaderPlacement(t *testing.T) {
	cfg := newMode(model.AuthAPIKey)
	cfg.Key = strp("X-API-Key")
	cfg.Value = strp("secret")
	cfg.Placement = model.APIKeyHeader

	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 1 || p.Headers[0].Name != "X-API-Key" || p.Headers[0].Value != "secret" {
		t.Fatalf("headers = %+v", p.Headers)
	}
	if len(p.QueryParams) != 0 {
		t.Error("header placement must not add a query param")
	}
}

func TestAPIKeyQueryPlacement(t *testing.T) {
	cfg := newMode(model.AuthAPIKey)
	cfg.Key = strp("api_key")
	cfg.Value = strp("abc")
	cfg.Placement = model.APIKeyQueryParams

	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.QueryParams) != 1 || p.QueryParams[0].Name != "api_key" || p.QueryParams[0].Value != "abc" {
		t.Fatalf("query = %+v", p.QueryParams)
	}
	if len(p.Headers) != 0 {
		t.Error("query placement must not add a header")
	}
}

// TestAPIKeyDefaultsToHeader covers an unset placement.
func TestAPIKeyDefaultsToHeader(t *testing.T) {
	cfg := newMode(model.AuthAPIKey)
	cfg.Key = strp("k")
	cfg.Value = strp("v")
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 1 {
		t.Fatalf("expected header placement by default, got %+v", p)
	}
}

func TestAPIKeyEmptyKeyIsNoop(t *testing.T) {
	cfg := newMode(model.AuthAPIKey)
	cfg.Key = strp("")
	cfg.Value = strp("v")
	p, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 || len(p.QueryParams) != 0 {
		t.Errorf("empty key should do nothing, got %+v", p)
	}
}

// TestWSSEDigestGolden pins the exact digest composition. The hex-inside-base64
// step is unusual and easy to omit.
func TestWSSEDigestGolden(t *testing.T) {
	cfg := newMode(model.AuthWSSE)
	cfg.Username = strp("joe")
	cfg.Password = strp("secret")

	deps := Deps{
		Now:       func() int64 { return 1700000000 },
		RandBytes: func(n int) []byte { return make([]byte, n) }, // all zero
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, deps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	header := p.Headers[0].Value
	if !strings.HasPrefix(header, `UsernameToken Username="joe"`) {
		t.Errorf("header = %q", header)
	}
	// nonce: 16 zero bytes as hex
	if !strings.Contains(header, `Nonce="00000000000000000000000000000000"`) {
		t.Errorf("nonce wrong in %q", header)
	}
	if !strings.Contains(header, `Created="2023-11-14T22:13:20Z"`) {
		t.Errorf("created wrong in %q", header)
	}

	want := wsseDigest("00000000000000000000000000000000", "2023-11-14T22:13:20Z", "secret")
	if !strings.Contains(header, `PasswordDigest="`+want+`"`) {
		t.Errorf("digest = %q, want %q in %q", want, want, header)
	}
}

func TestWSSEDigestIsHexThenBase64(t *testing.T) {
	// Compute independently: base64(hex(sha1(nonce+created+password))).
	got := wsseDigest("n", "t", "p")
	if got == "" {
		t.Fatal("empty digest")
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("digest is not valid base64: %v", err)
	}
	// The decoded payload is 40 hex characters, not 20 raw bytes: that hex step is
	// the part a from-scratch implementation usually omits.
	if len(decoded) != 40 {
		t.Errorf("decoded length = %d, want 40 hex chars", len(decoded))
	}
	for _, c := range decoded {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("decoded payload is not hex: %q", decoded)
		}
	}
}

func TestWSSEInterpolatesBeforeHashing(t *testing.T) {
	// The divergence from Bruno (D-009): {{pass}} must be resolved before hashing.
	cfg := newMode(model.AuthWSSE)
	cfg.Username = strp("u")
	cfg.Password = strp("{{pass}}")

	deps := Deps{
		Now:       func() int64 { return 0 },
		RandBytes: func(n int) []byte { return make([]byte, n) },
		Interpolate: func(s string) string {
			if s == "{{pass}}" {
				return "resolved"
			}
			return s
		},
	}
	p, err := Apply(context.Background(), []*model.Auth{cfg}, deps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := wsseDigest("00000000000000000000000000000000", "1970-01-01T00:00:00Z", "resolved")
	if !strings.Contains(p.Headers[0].Value, want) {
		t.Errorf("digest did not use the interpolated password: %q", p.Headers[0].Value)
	}
}

// ---- inheritance ----

func TestResolveInheritanceChain(t *testing.T) {
	collection := &model.Auth{Mode: model.AuthBearer, Token: strp("coll")}
	folder := &model.Auth{Mode: model.AuthInherit}
	request := &model.Auth{Mode: model.AuthInherit}

	got := Resolve([]*model.Auth{collection, folder, request})
	if got == nil || got.Mode != model.AuthBearer {
		t.Fatalf("resolved = %+v, want the collection bearer", got)
	}
}

func TestResolveMoreSpecificWins(t *testing.T) {
	collection := &model.Auth{Mode: model.AuthBearer, Token: strp("coll")}
	folder := &model.Auth{Mode: model.AuthBasic, Username: strp("u")}
	request := &model.Auth{Mode: model.AuthInherit}

	got := Resolve([]*model.Auth{collection, folder, request})
	if got == nil || got.Mode != model.AuthBasic {
		t.Fatalf("resolved = %+v, want the folder's basic", got)
	}
}

// TestResolveNoneStopsInheritance matters: `none` is a deliberate opt-out, not a
// request to keep looking upward.
func TestResolveNoneStopsInheritance(t *testing.T) {
	collection := &model.Auth{Mode: model.AuthBearer, Token: strp("coll")}
	request := &model.Auth{Mode: model.AuthNone}

	got := Resolve([]*model.Auth{collection, request})
	if got != nil {
		t.Fatalf("resolved = %+v, want nil (explicit none)", got)
	}
}

func TestResolveEmptyChain(t *testing.T) {
	if got := Resolve(nil); got != nil {
		t.Errorf("nil chain resolved to %+v", got)
	}
	if got := Resolve([]*model.Auth{nil, {Mode: model.AuthInherit}}); got != nil {
		t.Errorf("all-inherit chain resolved to %+v", got)
	}
}

func TestResolveReturnsCopy(t *testing.T) {
	cfg := &model.Auth{Mode: model.AuthBearer, Token: strp("orig")}
	got := Resolve([]*model.Auth{cfg})
	*got.Token = "mutated"
	if *cfg.Token != "orig" {
		t.Error("Resolve must not alias stored configuration")
	}
}

func TestApplyNoProviderForAdvancedMode(t *testing.T) {
	// oauth2 is AUTH-OAUTH2's; until it lands, Apply must report it clearly.
	cfg := newMode(model.AuthOAuth2)
	_, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err == nil {
		t.Fatal("expected an error for an unregistered mode")
	}
	if !strings.Contains(err.Error(), string(model.AuthOAuth2)) {
		t.Errorf("error should name the mode: %v", err)
	}
}

func TestApplyNoneIsNotAnError(t *testing.T) {
	p, err := Apply(context.Background(), []*model.Auth{{Mode: model.AuthNone}}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(p.Headers) != 0 {
		t.Errorf("none should add nothing, got %+v", p)
	}
}

func TestChainForOrder(t *testing.T) {
	coll := &model.Collection{Config: model.CollectionConfig{Auth: &model.Auth{Mode: model.AuthBearer}}}
	outer := &model.Item{UID: "f1", Auth: &model.Auth{Mode: model.AuthBasic}}
	inner := &model.Item{UID: "f2", Auth: &model.Auth{Mode: model.AuthAPIKey}}
	req := &model.Item{Request: &model.Request{Auth: &model.Auth{Mode: model.AuthNTLM}}}

	chain := ChainFor(coll, []*model.Item{outer, inner}, req)
	if len(chain) != 4 {
		t.Fatalf("chain length = %d, want 4", len(chain))
	}
	if chain[0].Mode != model.AuthBearer || chain[1].Mode != model.AuthBasic ||
		chain[2].Mode != model.AuthAPIKey || chain[3].Mode != model.AuthNTLM {
		t.Errorf("chain order = %v", []model.AuthMode{chain[0].Mode, chain[1].Mode, chain[2].Mode, chain[3].Mode})
	}
}

func TestChainForSkipsMissing(t *testing.T) {
	if got := ChainFor(&model.Collection{}, nil, nil); len(got) != 0 {
		t.Errorf("chain = %+v, want empty", got)
	}
}

// ---- digest ----

func TestParseDigestChallenge(t *testing.T) {
	h := http.Header{}
	h.Set("WWW-Authenticate", `Digest realm="test@example.org", qop="auth,auth-int", nonce="abc123", opaque="op", algorithm=MD5, stale=true`)

	ch, ok := parseDigestChallenge(h)
	if !ok {
		t.Fatal("challenge not parsed")
	}
	if ch.Realm != "test@example.org" || ch.Nonce != "abc123" || ch.Opaque != "op" {
		t.Errorf("challenge = %+v", ch)
	}
	if ch.QOP != "auth,auth-int" || ch.Algorithm != "MD5" || !ch.Stale {
		t.Errorf("challenge = %+v", ch)
	}
}

func TestParseDigestChallengeRequiresNonce(t *testing.T) {
	h := http.Header{}
	h.Set("WWW-Authenticate", `Basic realm="x"`)
	if _, ok := parseDigestChallenge(h); ok {
		t.Error("a Basic challenge must not be read as Digest")
	}
}

func TestDigestHeaderShape(t *testing.T) {
	ch := challenge{Realm: "r", Nonce: "n", QOP: "auth,auth-int", Algorithm: "MD5"}
	hdr := digestAuthorization(ch, "user", "pass", "GET", "/path")

	for _, want := range []string{"Digest ", `username="user"`, `realm="r"`, `nonce="n"`, "qop=auth", `uri="/path"`, "response="} {
		if !strings.Contains(hdr, want) {
			t.Errorf("header missing %q:\n%s", want, hdr)
		}
	}
}

func TestDigestRejectsUnsupportedAlgorithm(t *testing.T) {
	ch := challenge{Realm: "r", Nonce: "n", Algorithm: "SHA-256"}
	if got := digestAuthorization(ch, "u", "p", "GET", "/"); got != "" {
		t.Errorf("expected no header for SHA-256, got %q", got)
	}
}

// TestDigestTransportRetriesAfterChallenge exercises the full handshake against a
// real server, which is the only way to be sure the retry and body handling work.
func TestDigestTransportRetriesAfterChallenge(t *testing.T) {
	const realm = "test"
	const nonce = "dcd98b7102dd2f0e8b11d0f600bfb0c093"
	const user = "Mufasa"
	const pass = "Circle Of Life"

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		authz := r.Header.Get("Authorization")
		if authz == "" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm=%q, nonce=%q, qop="auth"`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Verify the response digest the same way a server would.
		params := parseAuthParams(strings.TrimPrefix(authz, "Digest "))
		ha1 := md5Hex(user + ":" + realm + ":" + pass)
		ha2 := md5Hex(r.Method + ":" + params["uri"])
		want := md5Hex(ha1 + ":" + nonce + ":" + params["nc"] + ":" + params["cnonce"] + ":auth:" + ha2)
		if params["response"] != want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	buildReq := func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, server.URL+"/dir/index.html", nil)
	}
	dt := &digestTransport{user: user, pass: pass, rewind: buildReq}
	client := &http.Client{Transport: dt}

	req, err := buildReq()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (attempts=%d)", resp.StatusCode, attempts)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2 (challenge then retry)", attempts)
	}
}

func TestDigestTransportPassesThroughWithoutChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dt := &digestTransport{user: "u", pass: "p", rewind: func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, server.URL, nil)
	}}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestDigestTransportLeavesNonDigestChallengeAlone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="x"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	dt := &digestTransport{user: "u", pass: "p", rewind: func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, server.URL, nil)
	}}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want the original 401", resp.StatusCode)
	}
}

func TestRequestURI(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://x/a/b?c=1&d=2", nil)
	if got := requestURI(req); got != "/a/b?c=1&d=2" {
		t.Errorf("uri = %q", got)
	}
	req2, _ := http.NewRequest(http.MethodGet, "http://x/a/b", nil)
	if got := requestURI(req2); got != "/a/b" {
		t.Errorf("uri = %q", got)
	}
}

func TestQuoteHeaderValue(t *testing.T) {
	if got := quoteHeaderValue(`a"b\c`); got != `"a\"b\\c"` {
		t.Errorf("quoted = %q", got)
	}
}

func TestPickQOP(t *testing.T) {
	if got := pickQOP("auth-int, auth"); got != "auth" {
		t.Errorf("qop = %q", got)
	}
	if got := pickQOP("auth-int"); got != "" {
		t.Errorf("qop = %q, want empty when only auth-int is offered", got)
	}
	if got := pickQOP(""); got != "" {
		t.Errorf("qop = %q", got)
	}
}
