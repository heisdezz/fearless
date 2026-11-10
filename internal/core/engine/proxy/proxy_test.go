package proxy_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"fearless/internal/core/engine/proxy"
	"fearless/internal/core/model"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestConfigURL(t *testing.T) {
	// Standard HTTP
	c1 := &proxy.Config{
		Protocol: "http",
		Hostname: "127.0.0.1",
		Port:     "8080",
	}
	u1, err := c1.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u1.String() != "http://127.0.0.1:8080" {
		t.Fatalf("expected http://127.0.0.1:8080, got %s", u1)
	}

	// HTTPS with auth
	c2 := &proxy.Config{
		Protocol: "https",
		Hostname: "proxy.corp.internal",
		Port:     "8443",
		Username: "alice",
		Password: "secret-password",
	}
	u2, err := c2.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u2.String() != "https://alice:secret-password@proxy.corp.internal:8443" {
		t.Fatalf("expected https://alice:secret-password@proxy.corp.internal:8443, got %s", u2)
	}

	// SOCKS5
	c3 := &proxy.Config{
		Protocol: "socks5",
		Hostname: "localhost",
		Port:     "1080",
	}
	u3, err := c3.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u3.String() != "socks5://localhost:1080" {
		t.Fatalf("expected socks5://localhost:1080, got %s", u3)
	}

	// Disabled returns nil
	c4 := &proxy.Config{
		Protocol: "http",
		Hostname: "127.0.0.1",
		Port:     "8080",
		Disabled: true,
	}
	u4, err := c4.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u4 != nil {
		t.Fatalf("expected nil URL for disabled config")
	}

	// Invalid protocol
	c5 := &proxy.Config{
		Protocol: "ftp",
		Hostname: "127.0.0.1",
		Port:     "21",
	}
	if _, err := c5.URL(); err == nil {
		t.Fatalf("expected error for ftp protocol")
	}
}

func TestBypassCurlSemantics(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		target   string
		expected bool
	}{
		// Wildcard
		{"wildcard matches all", []string{"*"}, "https://api.github.com/v1", true},
		{"wildcard in list", []string{"foo.com", "*"}, "http://192.168.1.10", true},

		// Domain suffix matching
		{"exact domain", []string{"example.com"}, "http://example.com/api", true},
		{"subdomain matches base domain", []string{"example.com"}, "http://api.example.com/test", true},
		{"deep subdomain matches base domain", []string{"example.com"}, "http://a.b.c.example.com/", true},
		{"leading dot in pattern matches subdomain", []string{".example.com"}, "http://api.example.com/", true},
		{"leading dot in pattern matches root", []string{".example.com"}, "http://example.com/", true},
		{"different domain does not match", []string{"example.com"}, "http://notexample.com/", false},
		{"partial prefix does not match", []string{"ample.com"}, "http://example.com/", false},

		// Port-qualified patterns
		{"port matches target port", []string{"example.com:8080"}, "http://example.com:8080/foo", true},
		{"port mismatch does not bypass", []string{"example.com:8080"}, "http://example.com:9000/foo", false},
		{"port mismatch with default http port 80", []string{"example.com:8080"}, "http://example.com/foo", false},
		{"port match with default http port 80", []string{"example.com:80"}, "http://example.com/foo", true},
		{"port match with default https port 443", []string{"example.com:443"}, "https://example.com/foo", true},

		// IP matching
		{"exact IPv4 match", []string{"192.168.1.1"}, "http://192.168.1.1/index", true},
		{"IPv4 mismatch", []string{"192.168.1.1"}, "http://192.168.1.2/index", false},
		{"exact IPv6 match", []string{"::1"}, "http://[::1]:8080/", true},

		// CIDR subnet matching
		{"CIDR /24 match", []string{"192.168.1.0/24"}, "http://192.168.1.42:3000/", true},
		{"CIDR /24 outside", []string{"192.168.1.0/24"}, "http://192.168.2.1:3000/", false},
		{"CIDR /16 match", []string{"10.0.0.0/8"}, "http://10.254.1.20/", true},
		{"IPv6 CIDR match", []string{"fe80::/10"}, "http://[fe80::1]/", true},

		// Localhost alias
		{"localhost matches localhost", []string{"localhost"}, "http://localhost:3000", true},
		{"localhost matches 127.0.0.1", []string{"localhost"}, "http://127.0.0.1:3000", true},
		{"localhost matches ::1 loopback", []string{"localhost"}, "http://[::1]:3000", true},
		{"127.0.0.1 matches localhost", []string{"127.0.0.1"}, "http://localhost:3000", true},

		// Delimiter splitting
		{"comma separated", []string{"foo.com, bar.com, 10.0.0.0/8"}, "http://bar.com/test", true},
		{"space separated", []string{"foo.com bar.com"}, "http://bar.com/test", true},
		{"semicolon separated", []string{"foo.com; bar.com"}, "http://bar.com/test", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := mustParseURL(tc.target)
			got := proxy.MatchBypass(tc.patterns, target)
			if got != tc.expected {
				t.Fatalf("MatchBypass(%v, %s) = %v; expected %v", tc.patterns, tc.target, got, tc.expected)
			}
		})
	}
}

func TestPrecedenceResolution(t *testing.T) {
	target := mustParseURL("https://api.example.com/users")

	reqProxy := &proxy.Config{
		Protocol: "http",
		Hostname: "request-proxy.local",
		Port:     "8001",
	}
	colProxy := &proxy.CollectionProxy{
		Inherit: false,
		Config: &proxy.Config{
			Protocol: "http",
			Hostname: "collection-proxy.local",
			Port:     "8002",
		},
	}
	appSettings := &model.ProxySettings{
		Mode:     model.ProxyManual,
		Protocol: "http",
		Hostname: "app-proxy.local",
		Port:     "8003",
	}
	mockEnv := map[string]string{
		"HTTPS_PROXY": "http://env-proxy.local:8004",
	}
	envLookup := func(k string) (string, bool) {
		v, ok := mockEnv[k]
		return v, ok
	}

	// 1. Request scope wins over all
	u, err := proxy.Resolve(proxy.ResolutionInput{
		TargetURL:       target,
		RequestProxy:    reqProxy,
		CollectionProxy: colProxy,
		AppSettings:     appSettings,
		LookupEnv:       envLookup,
	})
	if err != nil || u == nil || u.Host != "request-proxy.local:8001" {
		t.Fatalf("expected request proxy (8001), got %v (err=%v)", u, err)
	}

	// 2. Request disabled forces direct connection (does not fall through!)
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:       target,
		RequestProxy:    &proxy.Config{Disabled: true},
		CollectionProxy: colProxy,
		AppSettings:     appSettings,
		LookupEnv:       envLookup,
	})
	if err != nil || u != nil {
		t.Fatalf("expected direct (nil) when request proxy is disabled, got %v", u)
	}

	// 3. Request bypassed forces direct connection
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL: target,
		RequestProxy: &proxy.Config{
			Protocol: "http",
			Hostname: "request-proxy.local",
			Bypass:   []string{"example.com"},
		},
		CollectionProxy: colProxy,
		AppSettings:     appSettings,
		LookupEnv:       envLookup,
	})
	if err != nil || u != nil {
		t.Fatalf("expected direct (nil) when request proxy is bypassed, got %v", u)
	}

	// 4. Without request proxy, Collection proxy wins over App and Env
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:       target,
		CollectionProxy: colProxy,
		AppSettings:     appSettings,
		LookupEnv:       envLookup,
	})
	if err != nil || u == nil || u.Host != "collection-proxy.local:8002" {
		t.Fatalf("expected collection proxy (8002), got %v", u)
	}

	// 5. Collection disabled forces direct connection
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:       target,
		CollectionProxy: &proxy.CollectionProxy{Disabled: true},
		AppSettings:     appSettings,
		LookupEnv:       envLookup,
	})
	if err != nil || u != nil {
		t.Fatalf("expected direct (nil) when collection proxy is disabled, got %v", u)
	}

	// 6. Collection with Inherit: true falls through to App
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL: target,
		CollectionProxy: &proxy.CollectionProxy{
			Inherit: true,
			Config:  colProxy.Config, // ignored because inherit is true
		},
		AppSettings: appSettings,
		LookupEnv:   envLookup,
	})
	if err != nil || u == nil || u.Host != "app-proxy.local:8003" {
		t.Fatalf("expected app proxy (8003) via inheritance, got %v", u)
	}

	// 7. App manual proxy wins over Env
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:   target,
		AppSettings: appSettings,
		LookupEnv:   envLookup,
	})
	if err != nil || u == nil || u.Host != "app-proxy.local:8003" {
		t.Fatalf("expected app proxy (8003), got %v", u)
	}

	// 8. App ProxyOff forces direct (ignores env!)
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:   target,
		AppSettings: &model.ProxySettings{Mode: model.ProxyOff},
		LookupEnv:   envLookup,
	})
	if err != nil || u != nil {
		t.Fatalf("expected direct (nil) when app proxy mode is off, got %v", u)
	}

	// 9. App ProxySystem delegates to Environment
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL:   target,
		AppSettings: &model.ProxySettings{Mode: model.ProxySystem},
		LookupEnv:   envLookup,
	})
	if err != nil || u == nil || u.Host != "env-proxy.local:8004" {
		t.Fatalf("expected env proxy (8004) under ProxySystem, got %v", u)
	}

	// 10. Environment NO_PROXY bypasses env proxy
	mockEnvWithNoProxy := map[string]string{
		"HTTPS_PROXY": "http://env-proxy.local:8004",
		"NO_PROXY":    ".example.com",
	}
	u, err = proxy.Resolve(proxy.ResolutionInput{
		TargetURL: target,
		LookupEnv: func(k string) (string, bool) {
			v, ok := mockEnvWithNoProxy[k]
			return v, ok
		},
	})
	if err != nil || u != nil {
		t.Fatalf("expected direct (nil) via NO_PROXY, got %v", u)
	}
}

func TestConfigureTransport(t *testing.T) {
	tr := &http.Transport{}

	// 1. Direct (nil)
	if err := proxy.ConfigureTransport(tr, nil); err != nil {
		t.Fatal(err)
	}
	if tr.Proxy != nil || tr.DialContext != nil {
		t.Fatalf("expected direct transport")
	}

	// 2. HTTP proxy
	httpProxyURL := mustParseURL("http://proxy.local:8080")
	if err := proxy.ConfigureTransport(tr, httpProxyURL); err != nil {
		t.Fatal(err)
	}
	if tr.Proxy == nil {
		t.Fatalf("expected tr.Proxy to be set for HTTP proxy")
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
	resolved, err := tr.Proxy(req)
	if err != nil || resolved == nil || resolved.String() != "http://proxy.local:8080" {
		t.Fatalf("expected http://proxy.local:8080, got %v", resolved)
	}

	// 3. SOCKS5 proxy
	socksURL := mustParseURL("socks5://127.0.0.1:1080")
	if err := proxy.ConfigureTransport(tr, socksURL); err != nil {
		t.Fatal(err)
	}
	if tr.Proxy != nil {
		t.Fatalf("expected tr.Proxy to be nil for SOCKS5")
	}
	if tr.DialContext == nil {
		t.Fatalf("expected tr.DialContext to be configured for SOCKS5")
	}

	// 4. Nil transport error
	if err := proxy.ConfigureTransport(nil, httpProxyURL); err == nil {
		t.Fatalf("expected error for nil transport")
	}

	// 5. Unsupported scheme
	badURL := mustParseURL("gopher://localhost:70")
	if err := proxy.ConfigureTransport(tr, badURL); err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
}

func TestLiveHTTPProxyRoundTrip(t *testing.T) {
	// Start target upstream server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reached", "target")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("target response"))
	}))
	defer targetServer.Close()

	// Start a proxy server that adds X-Proxied: true
	proxyCalled := false
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCalled = true
		// Forward request to target
		outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.RequestURI, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for k, v := range r.Header {
			outReq.Header[k] = v
		}
		resp, err := http.DefaultTransport.RoundTrip(outReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.Header().Set("X-Proxied", "true")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	// Configure client with proxy
	proxyURL := mustParseURL(proxyServer.URL)
	tr := &http.Transport{}
	if err := proxy.ConfigureTransport(tr, proxyURL); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(targetServer.URL)
	if err != nil {
		t.Fatalf("GET via proxy failed: %v", err)
	}
	defer resp.Body.Close()

	if !proxyCalled {
		t.Fatalf("expected proxy server to be called")
	}
	if resp.Header.Get("X-Proxied") != "true" {
		t.Fatalf("expected X-Proxied header from proxy")
	}
	if resp.Header.Get("X-Reached") != "target" {
		t.Fatalf("expected X-Reached header from target")
	}
}
