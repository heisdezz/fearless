package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"fearless/internal/core/model"
)

// deviceServer serves the device authorization and token endpoints.
type deviceServer struct {
	mu    sync.Mutex
	polls int
	// succeedAfter is how many polls return authorization_pending first.
	succeedAfter int
	// slowDown makes the first poll return slow_down.
	slowDown bool
	// omitInterval leaves `interval` out of the authorization response.
	omitInterval bool
}

func (s *deviceServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	switch r.URL.Path {
	case "/device":
		s.mu.Lock()
		omitInterval := s.omitInterval
		s.mu.Unlock()

		body := `{"device_code":"dc-1","user_code":"ABCD-EFGH",` +
			`"verification_uri":"https://verify.example.com",` +
			`"verification_uri_complete":"https://verify.example.com?c=ABCD-EFGH"`
		if !omitInterval {
			body += `,"interval":1,"expires_in":600`
		}
		body += "}"
		_, _ = w.Write([]byte(body))

	case "/token":
		s.mu.Lock()
		s.polls++
		polls := s.polls
		slowDown := s.slowDown
		succeedAfter := s.succeedAfter
		s.mu.Unlock()

		if r.PostForm.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"unsupported_grant_type"}`))
			return
		}

		if slowDown && polls == 1 {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"slow_down"}`))
			return
		}
		if polls <= succeedAfter {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"device-token","expires_in":3600}`))
	}
}

// newDeviceServer starts a device server.
func newDeviceServer(t *testing.T, srv *deviceServer) string {
	t.Helper()
	httpServer := httptest.NewServer(srv)
	t.Cleanup(httpServer.Close)
	return httpServer.URL
}

// TestDeviceGrantPollsUntilApproved covers the RFC 8628 happy path.
func TestDeviceGrantPollsUntilApproved(t *testing.T) {
	srv := &deviceServer{succeedAfter: 2}
	base := newDeviceServer(t, srv)

	var promptedURI, promptedCode string
	c := &Client{
		Now:   func() int64 { return 1_700_000_000_000 },
		Cache: NewMemoryCache(),
		PromptDevice: func(uri, code string) bool {
			promptedURI, promptedCode = uri, code
			return true
		},
	}

	token, err := c.FetchDevice(context.Background(), Settings{
		ClientID:               "cid",
		AccessTokenURL:         base + "/token",
		DeviceAuthorizationURL: base + "/device",
		Scope:                  "read",
	})
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if token.AccessToken != "device-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if promptedCode != "ABCD-EFGH" {
		t.Errorf("user code = %q", promptedCode)
	}
	// The complete verification URI is preferred, because it saves the user
	// typing the code.
	if !strings.Contains(promptedURI, "c=ABCD-EFGH") {
		t.Errorf("verification URI = %q", promptedURI)
	}

	// The first two polls were pending, the third succeeded.
	if srv.polls != 3 {
		t.Errorf("polls = %d, want 3", srv.polls)
	}

	// And the token is cached like any other.
	if cached, ok := c.cache().Get(DefaultCredentialsID); !ok || cached.AccessToken != "device-token" {
		t.Errorf("cache = %+v, %v", cached, ok)
	}
}

// TestDeviceGrantSlowDownKeepsPolling covers the second retryable error.
func TestDeviceGrantSlowDownKeepsPolling(t *testing.T) {
	srv := &deviceServer{slowDown: true, succeedAfter: 1}
	base := newDeviceServer(t, srv)

	c := &Client{
		Now:          func() int64 { return 1 },
		Cache:        NewMemoryCache(),
		PromptDevice: func(string, string) bool { return true },
	}

	token, err := c.FetchDevice(context.Background(), Settings{
		ClientID:               "cid",
		AccessTokenURL:         base + "/token",
		DeviceAuthorizationURL: base + "/device",
	})
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if token.AccessToken != "device-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}
}

// TestDeviceGrantCancel covers a user declining the prompt.
func TestDeviceGrantCancel(t *testing.T) {
	base := newDeviceServer(t, &deviceServer{})
	c := &Client{
		Now:          func() int64 { return 1 },
		Cache:        NewMemoryCache(),
		PromptDevice: func(string, string) bool { return false },
	}

	_, err := c.FetchDevice(context.Background(), Settings{
		ClientID:               "cid",
		AccessTokenURL:         base + "/token",
		DeviceAuthorizationURL: base + "/device",
	})
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err = %v", err)
	}
}

// TestDeviceGrantTerminalErrorStopsPolling covers invalid_grant and friends.
func TestDeviceGrantTerminalErrorStopsPolling(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device" {
			_, _ = w.Write([]byte(`{"device_code":"dc","user_code":"UC","verification_uri":"https://v"}`))
			return
		}
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"access_denied","error_description":"user refused"}`))
	}))
	defer httpServer.Close()

	c := &Client{Now: func() int64 { return 1 }, Cache: NewMemoryCache(), PromptDevice: func(string, string) bool { return true }}

	_, err := c.FetchDevice(context.Background(), Settings{
		ClientID:               "cid",
		AccessTokenURL:         httpServer.URL + "/token",
		DeviceAuthorizationURL: httpServer.URL + "/device",
	})
	if err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("err = %v", err)
	}
}

// TestDeviceGrantContextCancellation stops a poll loop the user abandoned.
func TestDeviceGrantContextCancellation(t *testing.T) {
	// A server that only ever reports pending, so the loop must be stopped by the
	// context rather than by a token.
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device" {
			_, _ = w.Write([]byte(`{"device_code":"dc","user_code":"UC","verification_uri":"https://v"}`))
			return
		}
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	}))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := &Client{Now: func() int64 { return 1 }, Cache: NewMemoryCache(), PromptDevice: func(string, string) bool { return true }}

	_, err := c.FetchDevice(ctx, Settings{
		ClientID:               "cid",
		AccessTokenURL:         httpServer.URL + "/token",
		DeviceAuthorizationURL: httpServer.URL + "/device",
	})
	if err == nil {
		t.Fatal("expected the context to end the loop")
	}
}

// TestDeviceGrantIncompleteResponseIsReported covers a server that omits the
// codes.
func TestDeviceGrantIncompleteResponseIsReported(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"verification_uri":"https://v"}`))
	}))
	defer httpServer.Close()

	c := &Client{Now: func() int64 { return 1 }, Cache: NewMemoryCache()}
	_, err := c.FetchDevice(context.Background(), Settings{
		ClientID:               "cid",
		AccessTokenURL:         httpServer.URL + "/token",
		DeviceAuthorizationURL: httpServer.URL + "/device",
	})
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("err = %v", err)
	}
}

// TestDeviceGrantDefaultsAuthorizationURL covers DeviceAuthorizationURL being
// unset, which falls back to the token URL.
//
// Both endpoints are the same URL here, so the server discriminates on the grant
// type in the body rather than the path.
func TestDeviceGrantDefaultsAuthorizationURL(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") == "urn:ietf:params:oauth:grant-type:device_code" {
			_, _ = w.Write([]byte(`{"access_token":"device-token"}`))
			return
		}
		_, _ = w.Write([]byte(`{"device_code":"dc","user_code":"UC","verification_uri":"https://v"}`))
	}))
	defer httpServer.Close()

	c := &Client{Now: func() int64 { return 1 }, Cache: NewMemoryCache(), PromptDevice: func(string, string) bool { return true }}

	token, err := c.FetchDevice(context.Background(), Settings{
		ClientID:       "cid",
		AccessTokenURL: httpServer.URL,
	})
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if token.AccessToken != "device-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}
}

// TestQueryParametersAreAppendedToTheTokenURL covers the query destination.
func TestQueryParametersAreAppendedToTheTokenURL(t *testing.T) {
	var gotQuery string
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"access_token":"t"}`))
	}))
	defer httpServer.Close()

	c := clientFor(1)
	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: httpServer.URL + "/token",
		ClientID:       "cid",
		AdditionalParams: &model.OAuth2AdditionalParams{
			AccessTokenRequest: model.AKVSet{
				QueryParams: []model.AKV{{Name: strp("resource"), Value: strp("r1"), Enabled: true}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "resource=r1") {
		t.Errorf("query = %q", gotQuery)
	}
}

// TestHeaderParametersAreSent covers the header destination.
func TestHeaderParametersAreSent(t *testing.T) {
	var gotHeader string
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Tenant")
		_, _ = w.Write([]byte(`{"access_token":"t"}`))
	}))
	defer httpServer.Close()

	c := clientFor(1)
	_, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: httpServer.URL,
		ClientID:       "cid",
		AdditionalParams: &model.OAuth2AdditionalParams{
			AccessTokenRequest: model.AKVSet{
				Headers: []model.AKV{{Name: strp("X-Tenant"), Value: strp("t1"), Enabled: true}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != "t1" {
		t.Errorf("header = %q", gotHeader)
	}
}

// TestRefreshUsesConfiguredRefreshURL covers a separate refresh endpoint.
func TestRefreshUsesConfiguredRefreshURL(t *testing.T) {
	var refreshHit bool
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/refresh" {
			refreshHit = true
		}
		_, _ = w.Write([]byte(`{"access_token":"new"}`))
	}))
	defer httpServer.Close()

	cache := NewMemoryCache()
	cache.Put("default", Token{AccessToken: "old", RefreshToken: "rt", ExpiresIn: 1, CreatedAt: 1})
	c := &Client{Now: func() int64 { return 1_700_000_000_000 }, Cache: cache}

	_, err := c.Token(context.Background(), Settings{
		GrantType:       model.GrantClientCredentials,
		AccessTokenURL:  httpServer.URL + "/token",
		RefreshTokenURL: httpServer.URL + "/refresh",
		ClientID:        "cid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !refreshHit {
		t.Error("the configured refresh URL was not used")
	}
}

// TestRefreshWithoutTokenReports covers a cache entry with no refresh token.
func TestRefreshWithoutTokenReports(t *testing.T) {
	c := clientFor(1)
	_, err := c.Refresh(context.Background(), Settings{}, Token{AccessToken: "a"})
	if err == nil || !strings.Contains(err.Error(), "no refresh token") {
		t.Fatalf("err = %v", err)
	}
}

// TestRefreshWithoutURLReports covers a missing endpoint.
func TestRefreshWithoutURLReports(t *testing.T) {
	c := clientFor(1)
	_, err := c.Refresh(context.Background(), Settings{}, Token{AccessToken: "a", RefreshToken: "rt"})
	if err == nil || !strings.Contains(err.Error(), "refresh URL") {
		t.Fatalf("err = %v", err)
	}
}

// TestExpiresInAsStringIsParsed covers a provider sending the lifetime quoted.
func TestExpiresInAsStringIsParsed(t *testing.T) {
	_, url := newServer(t, `{"access_token":"t","expires_in":"7200"}`, 200)
	c := clientFor(1_700_000_000_000)

	token, err := c.Token(context.Background(), Settings{
		GrantType:      model.GrantClientCredentials,
		AccessTokenURL: url,
		ClientID:       "cid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if token.ExpiresIn != 7200 {
		t.Errorf("expires_in = %d, want 7200", token.ExpiresIn)
	}
}

// TestAuthCodeAdditionalAuthorizationParams covers the authorize-URL extras.
func TestAuthCodeAdditionalAuthorizationParams(t *testing.T) {
	var authorizeURL string
	c := &Client{
		Now:   func() int64 { return 1 },
		Cache: NewMemoryCache(),
		Authorize: AuthorizerFunc(func(_ context.Context, u, _ string) (string, error) {
			authorizeURL = u
			return "", context.Canceled
		}),
	}

	_, _ = c.Token(context.Background(), Settings{
		GrantType:        model.GrantAuthorizationCode,
		AuthorizationURL: "https://auth/authorize",
		AccessTokenURL:   "https://auth/token",
		ClientID:         "cid",
		AdditionalParams: &model.OAuth2AdditionalParams{
			AuthorizationRequest: model.AKVSet{
				QueryParams: []model.AKV{{Name: strp("prompt"), Value: strp("consent"), Enabled: true}},
			},
		},
	})

	if !strings.Contains(authorizeURL, "prompt=consent") {
		t.Errorf("authorize URL = %q", authorizeURL)
	}
}

// TestAuthorizeURLRejectsInvalidBase covers a broken authorization URL.
func TestAuthorizeURLRejectsInvalidBase(t *testing.T) {
	c := clientFor(1)
	if _, err := c.AuthorizeURL(Settings{AuthorizationURL: "://bad"}, "s"); err == nil {
		t.Fatal("expected an error")
	}
}
