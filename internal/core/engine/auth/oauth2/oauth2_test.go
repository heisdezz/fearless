package oauth2

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"fearless/internal/core/model"
)

// tokenServer is a fake token endpoint that records what it received.
type tokenServer struct {
	mu           sync.Mutex
	forms        []url.Values
	authHeaders  []string
	status       int
	body         string
	requestCount int
}

func (s *tokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestCount++

	if err := r.ParseForm(); err == nil {
		s.forms = append(s.forms, r.PostForm)
	}
	s.authHeaders = append(s.authHeaders, r.Header.Get("Authorization"))

	if s.status != 0 {
		w.WriteHeader(s.status)
	}
	_, _ = w.Write([]byte(s.body))
}

func (s *tokenServer) lastForm() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.forms) == 0 {
		return url.Values{}
	}
	return s.forms[len(s.forms)-1]
}

func (s *tokenServer) lastAuthHeader() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.authHeaders) == 0 {
		return ""
	}
	return s.authHeaders[len(s.authHeaders)-1]
}

func (s *tokenServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requestCount
}

// newServer starts a token server and returns it with its URL.
func newServer(t *testing.T, body string, status int) (*tokenServer, string) {
	t.Helper()
	srv := &tokenServer{body: body, status: status}
	httpServer := httptest.NewServer(srv)
	t.Cleanup(httpServer.Close)
	return srv, httpServer.URL
}

// clientFor builds a client with a pinned clock.
func clientFor(now int64) *Client {
	c := &Client{Now: func() int64 { return now }, Cache: NewMemoryCache()}
	return c
}

// TestClientCredentialsGrant covers the machine-to-machine flow.
func TestClientCredentialsGrant(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"at-1","token_type":"Bearer","expires_in":3600,"scope":"read"}`, 200)
	c := clientFor(1_700_000_000_000)

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
		ClientSecret:   "csecret",
		Scope:          "read write",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "at-1" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.CreatedAt != 1_700_000_000_000 {
		t.Errorf("created at = %d", token.CreatedAt)
	}

	form := srv.lastForm()
	if form.Get("grant_type") != "client_credentials" {
		t.Errorf("grant_type = %q", form.Get("grant_type"))
	}
	if form.Get("scope") != "read write" {
		t.Errorf("scope = %q", form.Get("scope"))
	}

	// The default placement is basic auth, so the credentials must not also be in
	// the body.
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("cid:csecret"))
	if got := srv.lastAuthHeader(); got != wantAuth {
		t.Errorf("authorization = %q, want %q", got, wantAuth)
	}
	if form.Get("client_id") != "" || form.Get("client_secret") != "" {
		t.Errorf("basic placement must not put credentials in the body: %v", form)
	}
}

// TestClientCredentialsBodyPlacement covers the alternative placement.
func TestClientCredentialsBodyPlacement(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"at"}`, 200)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:            model.GrantClientCredentials,
		AccessTokenURL:       url,
		ClientID:             "cid",
		ClientSecret:         "csecret",
		CredentialsPlacement: "body",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	form := srv.lastForm()
	if form.Get("client_id") != "cid" || form.Get("client_secret") != "csecret" {
		t.Errorf("form = %v", form)
	}
	if got := srv.lastAuthHeader(); got != "" {
		t.Errorf("body placement must not set Authorization, got %q", got)
	}
}

// TestPublicClientOmitsEmptySecret keeps "" out of the body.
func TestPublicClientOmitsEmptySecret(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"at"}`, 200)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:            model.GrantClientCredentials,
		AccessTokenURL:       url,
		ClientID:             "public",
		CredentialsPlacement: "body",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	form := srv.lastForm()
	if _, present := form["client_secret"]; present {
		t.Errorf("an empty secret must be omitted: %v", form)
	}
}

// TestTokenIsCached is the whole point of the cache: a second request reuses it.
func TestTokenIsCached(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"at","expires_in":3600}`, 200)
	c := clientFor(1_700_000_000_000)
	settings := Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
		ClientSecret:   "cs",
	}

	if _, err := c.Token(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Token(context.Background(), settings); err != nil {
		t.Fatal(err)
	}

	if got := srv.count(); got != 1 {
		t.Errorf("token endpoint called %d times, want 1", got)
	}
}

// TestExpiredTokenIsRefreshed covers the acceptance criterion for refresh.
func TestExpiredTokenIsRefreshed(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"fresh","expires_in":3600}`, 200)

	cache := NewMemoryCache()
	cache.Put("default", Token{
		AccessToken:  "stale",
		RefreshToken: "rt-1",
		ExpiresIn:    60,
		CreatedAt:    1000, // expired long ago
	})

	now := int64(1_700_000_000_000)
	c := &Client{Now: func() int64 { return now }, Cache: cache}

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
		ClientSecret:   "cs",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "fresh" {
		t.Errorf("access token = %q, want the refreshed one", token.AccessToken)
	}

	// The refresh used the stored refresh token, and kept it when the response
	// omitted one.
	form := srv.lastForm()
	if form.Get("grant_type") != "refresh_token" {
		t.Fatalf("grant_type = %q, want refresh_token", form.Get("grant_type"))
	}
	if form.Get("refresh_token") != "rt-1" {
		t.Errorf("refresh_token = %q", form.Get("refresh_token"))
	}
	if token.RefreshToken != "rt-1" {
		t.Errorf("refresh token = %q, want the original preserved", token.RefreshToken)
	}
}

// TestRefreshFailureFallsBackToFetch covers a dead refresh token.
func TestRefreshFailureFallsBackToFetch(t *testing.T) {
	var mu sync.Mutex
	var grants []string

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		grants = append(grants, r.PostForm.Get("grant_type"))
		mu.Unlock()

		if r.PostForm.Get("grant_type") == "refresh_token" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"new","expires_in":3600}`))
	}))
	defer httpServer.Close()

	cache := NewMemoryCache()
	cache.Put("default", Token{AccessToken: "stale", RefreshToken: "rt", ExpiresIn: 1, CreatedAt: 1})
	c := &Client{Now: func() int64 { return 1_700_000_000_000 }, Cache: cache}

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: httpServer.URL,
		ClientID:       "cid",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "new" {
		t.Errorf("access token = %q", token.AccessToken)
	}

	// Both grants were attempted, refresh first.
	if len(grants) != 2 || grants[0] != "refresh_token" || grants[1] != "client_credentials" {
		t.Errorf("grants = %v", grants)
	}
}

// TestAutoFetchDisabledReturnsExpiredToken matches Bruno: the user asked for the
// stored value even if stale.
func TestAutoFetchDisabledReturnsExpiredToken(t *testing.T) {
	cache := NewMemoryCache()
	cache.Put("default", Token{AccessToken: "stale", ExpiresIn: 1, CreatedAt: 1})

	c := &Client{Now: func() int64 { return 1_700_000_000_000 }, Cache: cache}

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: "http://unused.invalid",
		ClientID:       "cid",
		FetchDisabled:  true,
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "stale" {
		t.Errorf("access token = %q, want the cached one", token.AccessToken)
	}
}

// TestNoTokenAndNoAutoFetchReports keeps a silent empty header from happening.
func TestNoTokenAndNoAutoFetchReports(t *testing.T) {
	c := &Client{Now: func() int64 { return 1 }, Cache: NewMemoryCache()}

	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: "http://unused.invalid",
		ClientID:       "cid",
		FetchDisabled:  true,
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "automatic fetching is disabled") {
		t.Errorf("err = %v", err)
	}
}

// TestTokenWithoutExpiryIsTreatedAsValid matches Bruno's precedence.
func TestTokenWithoutExpiryIsTreatedAsValid(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"at"}`, 200)
	c := clientFor(1_700_000_000_000)
	settings := Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	}

	if _, err := c.Token(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	// Advanced far past any plausible expiry; still cached, because the server
	// gave no lifetime to expire against.
	c.Now = func() int64 { return 1_800_000_000_000 }
	if _, err := c.Token(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if got := srv.count(); got != 1 {
		t.Errorf("endpoint called %d times, want 1", got)
	}
}

// TestTokenExpiryArithmetic covers Expired and ExpiresAt.
func TestTokenExpiryArithmetic(t *testing.T) {
	token := Token{AccessToken: "a", ExpiresIn: 60, CreatedAt: 1000}
	if token.ExpiresAt() != 61000 {
		t.Errorf("expiresAt = %d", token.ExpiresAt())
	}
	if token.Expired(60000) {
		t.Error("should be valid before expiry")
	}
	if !token.Expired(61001) {
		t.Error("should be expired after expiry")
	}

	// No lifetime information means never expired.
	noExpiry := Token{AccessToken: "a", CreatedAt: 1000}
	if noExpiry.Expired(9_999_999_999) {
		t.Error("a token with no expires_in must not be considered expired")
	}
	if noExpiry.ExpiresAt() != 0 {
		t.Errorf("expiresAt = %d, want 0", noExpiry.ExpiresAt())
	}
}

// TestTokenSourceSelectsIDToken covers tokenSource = id_token.
func TestTokenSourceSelectsIDToken(t *testing.T) {
	token := Token{AccessToken: "at", IDToken: "id"}
	if got := token.Value(""); got != "at" {
		t.Errorf("default = %q", got)
	}
	if got := token.Value("access_token"); got != "at" {
		t.Errorf("access_token = %q", got)
	}
	if got := token.Value("id_token"); got != "id" {
		t.Errorf("id_token = %q", got)
	}
}

// TestPasswordGrant covers the resource owner grant.
func TestPasswordGrant(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"pw-token"}`, 200)
	c := clientFor(1)

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantPassword,
		AccessTokenURL: url,
		ClientID:       "cid",
		Username:       "alice",
		Password:       "s3cret",
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "pw-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}

	form := srv.lastForm()
	if form.Get("grant_type") != "password" {
		t.Errorf("grant_type = %q", form.Get("grant_type"))
	}
	if form.Get("username") != "alice" || form.Get("password") != "s3cret" {
		t.Errorf("credentials = %v", form)
	}
}

// TestAuthorizationCodeWithPKCE is the end-to-end flow without a WebView.
func TestAuthorizationCodeWithPKCE(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"code-token","refresh_token":"rt"}`, 200)

	var authorizeURL, expectedState string
	authorizer := AuthorizerFunc(func(_ context.Context, u, state string) (string, error) {
		authorizeURL = u
		expectedState = state
		return "auth-code-1", nil
	})

	c := &Client{
		Now:       func() int64 { return 1_700_000_000_000 },
		Cache:     NewMemoryCache(),
		Authorize: authorizer,
	}

	token, err := c.Token(context.Background(), Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth.example.com/authorize",
		AccessTokenURL:   url,
		ClientID:         "cid",
		ClientSecret:     "cs",
		Scope:            "openid profile",
		State:            "user-state",
		CallbackURL:      "http://127.0.0.1:8080/callback",
		PKCE:             true,
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token.AccessToken != "code-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}

	// The authorization URL carries every required parameter.
	parsed, err := url_parse(authorizeURL)
	if err != nil {
		t.Fatalf("authorize URL %q: %v", authorizeURL, err)
	}
	query := parsed.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "cid",
		"redirect_uri":          "http://127.0.0.1:8080/callback",
		"scope":                 "openid profile",
		"code_challenge_method": "S256",
	}
	for key, want := range checks {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if query.Get("code_challenge") == "" {
		t.Error("PKCE was requested but no challenge was sent")
	}
	if query.Get("state") != expectedState {
		t.Errorf("state = %q, want the value handed to the authorizer", query.Get("state"))
	}
	// The user's state is preserved as a prefix so their own handler still sees it.
	if !strings.HasPrefix(expectedState, "user-state") {
		t.Errorf("state %q should keep the configured prefix", expectedState)
	}

	// The token exchange sends the verifier and the code.
	form := srv.lastForm()
	if form.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", form.Get("grant_type"))
	}
	if form.Get("code") != "auth-code-1" {
		t.Errorf("code = %q", form.Get("code"))
	}
	if form.Get("redirect_uri") != "http://127.0.0.1:8080/callback" {
		t.Errorf("redirect_uri = %q", form.Get("redirect_uri"))
	}
	if form.Get("code_verifier") == "" {
		t.Error("PKCE was requested but no verifier was sent")
	}

	// The verifier must hash to the challenge, which is what the server checks.
	if got := challengeFor(form.Get("code_verifier")); got != query.Get("code_challenge") {
		t.Errorf("verifier does not match the challenge: %q vs %q", got, query.Get("code_challenge"))
	}
}

// TestAuthorizationCodeWithoutPKCE covers the non-PKCE flow.
func TestAuthorizationCodeWithoutPKCE(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"t"}`, 200)
	c := &Client{
		Now:       func() int64 { return 1 },
		Cache:     NewMemoryCache(),
		Authorize: AuthorizerFunc(func(context.Context, string, string) (string, error) { return "code", nil }),
	}

	if _, err := c.Token(context.Background(), Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth/authorize",
		AccessTokenURL:   url,
		ClientID:         "cid",
	}); err != nil {
		t.Fatal(err)
	}

	form := srv.lastForm()
	if _, present := form["code_verifier"]; present {
		t.Errorf("no verifier should be sent without PKCE: %v", form)
	}
}

// TestAuthorizationCodeRequiresAuthorizer keeps the CLI failure explicit.
func TestAuthorizationCodeRequiresAuthorizer(t *testing.T) {
	c := clientFor(1)
	_, err := c.Token(context.Background(), Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth/authorize",
		AccessTokenURL:   "https://auth/token",
		ClientID:         "cid",
	})
	if err == nil || !strings.Contains(err.Error(), "authorizer") {
		t.Fatalf("err = %v", err)
	}
}

// TestAuthorizationAuthorizerError propagates a cancelled browser flow.
func TestAuthorizationAuthorizerError(t *testing.T) {
	c := &Client{
		Now:   func() int64 { return 1 },
		Cache: NewMemoryCache(),
		Authorize: AuthorizerFunc(func(context.Context, string, string) (string, error) {
			return "", errors.New("user closed the browser")
		}),
	}
	_, err := c.Token(context.Background(), Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth/authorize",
		AccessTokenURL:   "https://auth/token",
		ClientID:         "cid",
	})
	if err == nil || !strings.Contains(err.Error(), "user closed the browser") {
		t.Fatalf("err = %v", err)
	}
}

// TestAuthorizeURLIsPrintable is the CLI acceptance criterion: the URL can be
// produced without a browser.
func TestAuthorizeURLIsPrintable(t *testing.T) {
	c := clientFor(1)
	settings := Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth.example.com/authorize",
		ClientID:         "cid",
		Scope:            "read",
		CallbackURL:      "http://127.0.0.1:9876/callback",
	}

	authorizeURL, err := c.AuthorizeURL(settings, "state-1")
	if err != nil {
		t.Fatalf("url: %v", err)
	}
	parsed, err := url_parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse %q: %v", authorizeURL, err)
	}
	if parsed.Query().Get("state") != "state-1" {
		t.Errorf("state = %q", parsed.Query().Get("state"))
	}
	if !strings.HasPrefix(authorizeURL, "https://auth.example.com/authorize?") {
		t.Errorf("url = %q", authorizeURL)
	}
}

// TestProviderErrorIsReported carries the server's explanation.
func TestProviderErrorIsReported(t *testing.T) {
	_, url := newServer(t, `{"error":"invalid_client","error_description":"client not found"}`, 401)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "bad",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "invalid_client") || !strings.Contains(err.Error(), "client not found") {
		t.Errorf("err = %v", err)
	}
}

// TestNonJSONResponseIsReported covers an HTML error page.
func TestNonJSONResponseIsReported(t *testing.T) {
	_, url := newServer(t, `<html>Bad Gateway</html>`, 502)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v", err)
	}
}

// TestMissingAccessTokenReports covers a 200 with no token.
func TestMissingAccessTokenReports(t *testing.T) {
	_, url := newServer(t, `{"token_type":"Bearer"}`, 200)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	})
	if !errors.Is(err, ErrNoAccessToken) {
		t.Fatalf("err = %v", err)
	}
}

// TestMissingConfigIsReported covers each required field.
func TestMissingConfigIsReported(t *testing.T) {
	c := clientFor(1)
	ctx := context.Background()

	cases := []struct {
		name     string
		settings Settings
		want     string
	}{
		{"no grant", Settings{}, "grant type"},
		{"client credentials without url", Settings{GrantType: model.GrantClientCredentials, ClientID: "x"}, "access token URL"},
		{"client credentials without id", Settings{GrantType: model.GrantClientCredentials, AccessTokenURL: "http://x"}, "client ID"},
		{"password without url", Settings{GrantType: model.GrantPassword, Username: "u"}, "access token URL"},
		{"password without username", Settings{GrantType: model.GrantPassword, AccessTokenURL: "http://x"}, "username"},
		{"auth code without url", Settings{GrantType: model.GrantAuthorizationCode, AccessTokenURL: "http://x"}, "authorization URL"},
	}
	for _, tc := range cases {
		_, err := c.Token(ctx, tc.settings)
		if err == nil {
			t.Errorf("%s: expected an error", tc.name)
			continue
		}
		if !errors.Is(err, ErrMissingConfig) {
			t.Errorf("%s: err = %v, want ErrMissingConfig", tc.name, err)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: message %q should mention %q", tc.name, err, tc.want)
		}
	}
}

// TestImplicitGrantIsRejected keeps a misconfiguration explicit.
func TestImplicitGrantIsRejected(t *testing.T) {
	c := clientFor(1)
	_, err := c.Token(context.Background(), Settings{GrantType: model.GrantImplicit})
	if !errors.Is(err, ErrUnsupportedGrant) {
		t.Fatalf("err = %v", err)
	}
}

// TestUnknownGrantIsRejected covers a typo'd grant.
func TestUnknownGrantIsRejected(t *testing.T) {
	c := clientFor(1)
	_, err := c.Token(context.Background(), Settings{GrantType: "magic"})
	if !errors.Is(err, ErrUnsupportedGrant) {
		t.Fatalf("err = %v", err)
	}
}

// TestAdditionalParameters covers the three destinations.
func TestAdditionalParameters(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"t"}`, 200)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:            model.GrantClientCredentials,
		AccessTokenURL:       url,
		ClientID:             "cid",
		CredentialsPlacement: "body",
		AdditionalParams: &model.OAuth2AdditionalParams{
			AccessTokenRequest: model.AKVSet{
				Body:        []model.AKV{{Name: strp("audience"), Value: strp("api://x"), Enabled: true}},
				Headers:     []model.AKV{{Name: strp("X-Tenant"), Value: strp("t1"), Enabled: true}},
				QueryParams: []model.AKV{{Name: strp("resource"), Value: strp("r1"), Enabled: true}},
			},
		},
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	form := srv.lastForm()
	if form.Get("audience") != "api://x" {
		t.Errorf("audience = %q", form.Get("audience"))
	}
}

// TestDisabledAdditionalParameterIsSkipped covers the enabled flag.
func TestDisabledAdditionalParameterIsSkipped(t *testing.T) {
	srv, url := newServer(t, `{"access_token":"t"}`, 200)
	c := clientFor(1)

	_, err := c.Token(context.Background(), Settings{
		GrantType:            model.GrantClientCredentials,
		AccessTokenURL:       url,
		ClientID:             "cid",
		CredentialsPlacement: "body",
		AdditionalParams: &model.OAuth2AdditionalParams{
			AccessTokenRequest: model.AKVSet{
				Body: []model.AKV{{Name: strp("audience"), Value: strp("x"), Enabled: false}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := srv.lastForm()["audience"]; present {
		t.Error("a disabled parameter was sent")
	}
}

// TestSettingsFromAppliesDefaults covers the one place defaults live.
func TestSettingsFromAppliesDefaults(t *testing.T) {
	auth := &model.Auth{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: strp("https://token"),
		ClientID:       strp("cid"),
	}
	s := SettingsFrom(auth)

	if s.CredentialsPlacement != "basic_auth_header" {
		t.Errorf("placement = %q", s.CredentialsPlacement)
	}
	if s.CredentialsID != DefaultCredentialsID {
		t.Errorf("credentials id = %q", s.CredentialsID)
	}
	if s.TokenSource != "access_token" {
		t.Errorf("token source = %q", s.TokenSource)
	}
	// Both auto flags default to true, matching the UI defaults.
	if !s.AutoFetchToken || !s.AutoRefreshToken {
		t.Errorf("auto flags = %v %v", s.AutoFetchToken, s.AutoRefreshToken)
	}
}

// TestSettingsFromHonoursExplicitFalse covers a deliberately disabled flag.
func TestSettingsFromHonoursExplicitFalse(t *testing.T) {
	no := false
	auth := &model.Auth{
		GrantType:      model.GrantClientCredentials,
		AutoFetchToken: &no,
	}
	if SettingsFrom(auth).AutoFetchToken {
		t.Error("an explicit false must survive the defaulting")
	}
}

// TestSeparateCredentialsIDsCacheIndependently is why the key is the ID.
func TestSeparateCredentialsIDsCacheIndependently(t *testing.T) {
	first, url := newServer(t, `{"access_token":"first"}`, 200)
	cache := NewMemoryCache()
	c := &Client{Now: func() int64 { return 1 }, Cache: cache}

	base := Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	}

	a := base
	a.CredentialsID = "team-a"
	if _, err := c.Token(context.Background(), a); err != nil {
		t.Fatal(err)
	}

	b := base
	b.CredentialsID = "team-b"
	if _, err := c.Token(context.Background(), b); err != nil {
		t.Fatal(err)
	}

	if got := first.count(); got != 2 {
		t.Errorf("endpoint called %d times, want 2 (one per credentials id)", got)
	}
}

// TestConcurrentTokenRequestsDoNotRace exercises the cache under -race.
func TestConcurrentTokenRequestsDoNotRace(t *testing.T) {
	_, url := newServer(t, `{"access_token":"t","expires_in":3600}`, 200)
	cache := NewMemoryCache()
	c := &Client{Now: func() int64 { return 1 }, Cache: cache}

	settings := Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Token(context.Background(), settings); err != nil {
				t.Errorf("token: %v", err)
			}
		}()
	}
	wg.Wait()

	if token, ok := cache.Get("default"); !ok || token.AccessToken != "t" {
		t.Errorf("cache = %+v, %v", token, ok)
	}
}

// TestMemoryCacheOperations covers the cache contract.
func TestMemoryCacheOperations(t *testing.T) {
	cache := NewMemoryCache()

	if _, ok := cache.Get("missing"); ok {
		t.Error("an empty cache returned a token")
	}

	cache.Put("a", Token{AccessToken: "one"})
	if token, ok := cache.Get("a"); !ok || token.AccessToken != "one" {
		t.Errorf("get = %+v, %v", token, ok)
	}

	cache.Put("a", Token{AccessToken: "two"})
	if token, _ := cache.Get("a"); token.AccessToken != "two" {
		t.Errorf("put did not replace: %+v", token)
	}

	cache.Delete("a")
	if _, ok := cache.Get("a"); ok {
		t.Error("delete did not remove the token")
	}
	// Deleting an absent key is not an error.
	cache.Delete("absent")
}

// TestPKCEPairIsValid checks the verifier and challenge lengths and the hash.
func TestPKCEPairIsValid(t *testing.T) {
	for i := 0; i < 20; i++ {
		pair := NewPKCEPair()

		// RFC 7636 §4.1: 43–128 characters from the unreserved set.
		if len(pair.Verifier) < 43 || len(pair.Verifier) > 128 {
			t.Fatalf("verifier length = %d", len(pair.Verifier))
		}
		if strings.ContainsAny(pair.Verifier, "+/=") {
			t.Fatalf("verifier is not base64url: %q", pair.Verifier)
		}
		if pair.Method != "S256" {
			t.Fatalf("method = %q", pair.Method)
		}
		if challengeFor(pair.Verifier) != pair.Challenge {
			t.Fatalf("challenge does not hash from the verifier")
		}
	}

	// Two pairs must differ; a fixed verifier would defeat PKCE entirely.
	if NewPKCEPair().Verifier == NewPKCEPair().Verifier {
		t.Error("two verifiers were identical")
	}
}

// TestNewStateIsRandomised covers the CSRF defence.
func TestNewStateIsRandomised(t *testing.T) {
	withoutUser := NewState("")
	if len(withoutUser) < 16 {
		t.Errorf("nonce too short: %q", withoutUser)
	}
	if NewState("") == NewState("") {
		t.Error("two states were identical")
	}

	withUser := NewState("my-prefix")
	if !strings.HasPrefix(withUser, "my-prefix") {
		t.Errorf("state = %q, want the user prefix kept", withUser)
	}
	if withUser == "my-prefix" {
		t.Error("the user state was used verbatim, so the check would be guessable")
	}
}

// --- helpers -------------------------------------------------------------

// AuthorizerFunc adapts a function to Authorizer.
type AuthorizerFunc func(ctx context.Context, authorizeURL, expectedState string) (string, error)

// Authorize implements Authorizer.
func (f AuthorizerFunc) Authorize(ctx context.Context, authorizeURL, expectedState string) (string, error) {
	return f(ctx, authorizeURL, expectedState)
}

func strp(s string) *string { return &s }

// challengeFor computes the S256 challenge for a verifier.
func challengeFor(verifier string) string {
	return ChallengeFor(verifier)
}

// url_parse parses a URL, kept as a helper so the test does not shadow the
// `url` variable name it also uses for server addresses.
func url_parse(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
