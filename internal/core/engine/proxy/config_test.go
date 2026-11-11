package proxy

import (
	"testing"

	"fearless/internal/core/model"
)

// TestFromModelSettings covers the model-to-config conversion, including the
// two cases the original tests did not reach: a nil settings value and a
// password.
func TestFromModelSettings(t *testing.T) {
	if cfg := FromModelSettings(nil); cfg == nil || !cfg.Disabled {
		t.Errorf("nil settings = %+v, want disabled", cfg)
	}

	off := FromModelSettings(&model.ProxySettings{Mode: model.ProxyOff})
	if off == nil || !off.Disabled {
		t.Errorf("off = %+v, want disabled", off)
	}

	// A non-manual, non-off mode has no static proxy to describe.
	if cfg := FromModelSettings(&model.ProxySettings{Mode: model.ProxySystem}); cfg != nil {
		t.Errorf("system = %+v, want nil", cfg)
	}

	pwd := "s3cret"
	cfg := FromModelSettings(&model.ProxySettings{
		Mode:     model.ProxyManual,
		Protocol: "http",
		Hostname: "proxy.example",
		Port:     "3128",
		Username: "ada",
		Password: &pwd,
		Bypass:   []string{"localhost"},
	})
	if cfg == nil {
		t.Fatal("manual settings produced nil")
	}
	if cfg.Username != "ada" || cfg.Password != "s3cret" {
		t.Errorf("credentials = %q/%q", cfg.Username, cfg.Password)
	}
	if len(cfg.Bypass) != 1 || cfg.Bypass[0] != "localhost" {
		t.Errorf("bypass = %v", cfg.Bypass)
	}
	u, err := cfg.URL()
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if u.String() != "http://ada:s3cret@proxy.example:3128" {
		t.Errorf("url = %q", u.String())
	}
}

// TestResolveFromEnvironmentSchemeSelection covers the per-scheme variable
// preference and the no-proxy bypass, which are the branches the original tests
// did not reach.
func TestResolveFromEnvironmentSchemeSelection(t *testing.T) {
	lookup := func(env map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) {
			v, ok := env[k]
			return v, ok
		}
	}

	// https prefers HTTPS_PROXY over HTTP_PROXY.
	u, err := resolveFromEnvironment(mustURL(t, "https://a.test/x"), lookup(map[string]string{
		"HTTPS_PROXY": "http://secure:1",
		"HTTP_PROXY":  "http://plain:2",
	}))
	if err != nil {
		t.Fatalf("https: %v", err)
	}
	if u == nil || u.Host != "secure:1" {
		t.Errorf("https proxy = %v, want secure:1", u)
	}

	// http prefers HTTP_PROXY.
	u, err = resolveFromEnvironment(mustURL(t, "http://a.test/x"), lookup(map[string]string{
		"HTTPS_PROXY": "http://secure:1",
		"HTTP_PROXY":  "http://plain:2",
	}))
	if err != nil {
		t.Fatalf("http: %v", err)
	}
	if u == nil || u.Host != "plain:2" {
		t.Errorf("http proxy = %v, want plain:2", u)
	}

	// An unlisted scheme falls back to ALL_PROXY.
	u, err = resolveFromEnvironment(mustURL(t, "ftp://a.test/x"), lookup(map[string]string{
		"ALL_PROXY": "socks5://all:1080",
	}))
	if err != nil {
		t.Fatalf("ftp: %v", err)
	}
	if u == nil || u.Scheme != "socks5" {
		t.Errorf("ftp proxy = %v, want the ALL_PROXY", u)
	}

	// NO_PROXY wins over any proxy variable.
	u, err = resolveFromEnvironment(mustURL(t, "http://internal.test/x"), lookup(map[string]string{
		"HTTP_PROXY": "http://plain:2",
		"NO_PROXY":   "internal.test",
	}))
	if err != nil {
		t.Fatalf("no_proxy: %v", err)
	}
	if u != nil {
		t.Errorf("proxy = %v, want direct for a NO_PROXY host", u)
	}

	// A scheme-less value gets http:// prefixed.
	u, err = resolveFromEnvironment(mustURL(t, "http://a.test/x"), lookup(map[string]string{
		"HTTP_PROXY": "bare:8080",
	}))
	if err != nil {
		t.Fatalf("bare: %v", err)
	}
	if u == nil || u.Scheme != "http" || u.Host != "bare:8080" {
		t.Errorf("proxy = %v, want http://bare:8080", u)
	}

	// An unsupported scheme is an error, not a silent direct connection.
	if _, err := resolveFromEnvironment(mustURL(t, "http://a.test/x"), lookup(map[string]string{
		"HTTP_PROXY": "ftp://nope:21",
	})); err == nil {
		t.Error("an unsupported env proxy scheme was accepted")
	}

	// Nothing configured is direct.
	if u, err := resolveFromEnvironment(mustURL(t, "http://a.test/x"), lookup(nil)); err != nil || u != nil {
		t.Errorf("empty env = %v, %v; want nil, nil", u, err)
	}

	// A whitespace-only value is treated as unset.
	if u, err := resolveFromEnvironment(mustURL(t, "http://a.test/x"), lookup(map[string]string{
		"HTTP_PROXY": "   ",
	})); err != nil || u != nil {
		t.Errorf("blank env = %v, %v; want nil, nil", u, err)
	}
}
