package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return u
}

// evalPAC runs FindProxyForURL for target against source.
func evalPAC(t *testing.T, source, target string) PACResult {
	t.Helper()
	e := &PACEvaluator{Source: source}
	res, err := e.Evaluate(context.Background(), mustURL(t, target))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	return res
}

// TestPACSpecExamples runs the example documents from the Netscape PAC
// specification, which is the closest thing to a golden vector this format has.
func TestPACSpecExamples(t *testing.T) {
	// The spec's canonical example: proxied for everything but local hosts.
	const specExample = `
function FindProxyForURL(url, host) {
    if (isPlainHostName(host) || dnsDomainIs(host, ".example.com"))
        return "DIRECT";
    return "PROXY w3proxy.example.com:8080; DIRECT";
}
`
	// A local host is direct.
	if got := evalPAC(t, specExample, "http://intranet/"); !got.DirectTransfer() {
		t.Errorf("isPlainHostName: raw = %q, want DIRECT first", got.Raw)
	}
	// An internal domain is direct.
	if got := evalPAC(t, specExample, "http://docs.example.com/"); !got.DirectTransfer() {
		t.Errorf("dnsDomainIs: raw = %q, want DIRECT first", got.Raw)
	}
	// Anything else goes through the proxy, then direct.
	got := evalPAC(t, specExample, "https://api.other.org/x")
	if got.DirectTransfer() {
		t.Fatalf("raw = %q, want the proxy first", got.Raw)
	}
	if len(got.Attempts) != 2 {
		t.Fatalf("attempts = %v, want two", got.Attempts)
	}
	if got.Attempts[0].Kind != PACHTTP || got.Attempts[0].Host != "w3proxy.example.com" || got.Attempts[0].Port != "8080" {
		t.Errorf("first attempt = %+v", got.Attempts[0])
	}
	if got.Attempts[1].Kind != PACDirect {
		t.Errorf("second attempt = %+v, want DIRECT", got.Attempts[1])
	}
}

// TestPACLocalSubnetExample covers the second spec example, which uses
// isInNet with the DNS lookup stubbed.
func TestPACLocalSubnetExample(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    if (isInNet(host, "198.95.0.0", "255.255.0.0"))
        return "DIRECT";
    return "PROXY proxy:80";
}
`
	e := &PACEvaluator{
		Source: source,
		LookupHost: func(_ context.Context, host string) ([]string, error) {
			switch host {
			case "inside.corp":
				return []string{"198.95.249.79"}, nil
			case "outside.corp":
				return []string{"203.0.113.9"}, nil
			}
			return nil, errors.New("no such host")
		},
	}

	res, err := e.Evaluate(context.Background(), mustURL(t, "http://inside.corp/"))
	if err != nil {
		t.Fatalf("inside: %v", err)
	}
	if !res.DirectTransfer() {
		t.Errorf("an in-subnet host got %q, want DIRECT", res.Raw)
	}

	res, err = e.Evaluate(context.Background(), mustURL(t, "http://outside.corp/"))
	if err != nil {
		t.Fatalf("outside: %v", err)
	}
	if res.DirectTransfer() {
		t.Errorf("an out-of-subnet host got %q, want the proxy", res.Raw)
	}
}

// TestPACIsInNetDirectIP covers the literal-address branch, where no lookup
// happens.
func TestPACIsInNetDirectIP(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    if (isInNet(host, "10.0.0.0", "255.0.0.0")) return "DIRECT";
    return "PROXY p:1";
}
`
	lookups := 0
	e := &PACEvaluator{
		Source: source,
		LookupHost: func(context.Context, string) ([]string, error) {
			lookups++
			return []string{"1.2.3.4"}, nil
		},
	}
	res, err := e.Evaluate(context.Background(), mustURL(t, "http://10.1.2.3/"))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !res.DirectTransfer() {
		t.Errorf("raw = %q, want DIRECT", res.Raw)
	}
	if lookups != 0 {
		t.Errorf("isInNet did %d DNS lookups for a literal IP, want 0", lookups)
	}
}

// TestPACShExpMatch pins the host patterns PAC files use most.
func TestPACShExpMatch(t *testing.T) {
	cases := []struct {
		text, pattern string
		want          bool
	}{
		{"www.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"foo.bar", "foo.*", true},
		{"foo.bar", "foo.b?r", true},
		{"foo.bar", "foo.b?z", false},
		{"abc", "*", true},
		{"", "*", true},
		{"ABC", "abc", true}, // case-insensitive
		{"a/b/c", "a*c", true},
	}
	for _, tc := range cases {
		if got := shExpMatch(tc.text, tc.pattern); got != tc.want {
			t.Errorf("shExpMatch(%q, %q) = %v, want %v", tc.text, tc.pattern, got, tc.want)
		}
	}
}

// TestPACHelperFunctions covers the remaining spec helpers directly.
func TestPACHelperFunctions(t *testing.T) {
	if !isPlainHostName("intranet") {
		t.Error("a dotless host is not plain")
	}
	if isPlainHostName("a.b") {
		t.Error("a dotted host is plain")
	}
	if got := dnsDomainLevels("a.b.c"); got != 2 {
		t.Errorf("dnsDomainLevels = %d, want 2", got)
	}
	if !localHostOrDomainIs("www", "www.example.com") {
		t.Error("localHostOrDomainIs prefix failed")
	}
	if !localHostOrDomainIs("www.example.com", "www.example.com") {
		t.Error("localHostOrDomainIs exact failed")
	}
	if localHostOrDomainIs("mail", "www.example.com") {
		t.Error("localHostOrDomainIs matched a non-prefix")
	}
}

// TestPACDnsResolve covers the lookup global, including a failed lookup, which
// must yield "" rather than throwing.
func TestPACDnsResolve(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    var ip = dnsResolve(host);
    if (ip == "192.0.2.1") return "PROXY found:1";
    return "PROXY missing:2";
}
`
	e := &PACEvaluator{
		Source: source,
		LookupHost: func(_ context.Context, host string) ([]string, error) {
			if host == "known.test" {
				return []string{"192.0.2.1"}, nil
			}
			return nil, errors.New("nxdomain")
		},
	}
	res, err := e.Evaluate(context.Background(), mustURL(t, "http://known.test/"))
	if err != nil {
		t.Fatalf("known: %v", err)
	}
	if res.Attempts[0].Host != "found" {
		t.Errorf("raw = %q, want the dnsResolve branch", res.Raw)
	}

	res, err = e.Evaluate(context.Background(), mustURL(t, "http://unknown.test/"))
	if err != nil {
		t.Fatalf("unknown: %v", err)
	}
	if res.Attempts[0].Host != "missing" {
		t.Errorf("raw = %q, want the fallback branch", res.Raw)
	}
}

// TestPACMyIPAddress covers the self-address global.
func TestPACMyIPAddress(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    return "PROXY " + myIpAddress() + ":8080";
}
`
	e := &PACEvaluator{Source: source, MyIP: func() string { return "203.0.113.7" }}
	res, err := e.Evaluate(context.Background(), mustURL(t, "http://x/"))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Attempts[0].Host != "203.0.113.7" {
		t.Errorf("host = %q, want the injected address", res.Attempts[0].Host)
	}
}

// TestPACReceivesURLString pins the calling convention: the first argument is
// the URL as a string, because real PAC files slice it.
func TestPACReceivesURLString(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    if (url.substring(0, 5) == "https") return "PROXY secure:443";
    return "PROXY plain:80";
}
`
	res := evalPAC(t, source, "https://api.example.com/x")
	if res.Attempts[0].Host != "secure" {
		t.Errorf("raw = %q, want the https branch", res.Raw)
	}
	res = evalPAC(t, source, "http://api.example.com/x")
	if res.Attempts[0].Host != "plain" {
		t.Errorf("raw = %q, want the http branch", res.Raw)
	}
}

// TestPACParseResult covers the return-value grammar.
func TestPACParseResult(t *testing.T) {
	cases := []struct {
		raw   string
		want  []PACDirective
		fails bool
	}{
		{"DIRECT", []PACDirective{{Kind: PACDirect}}, false},
		{"PROXY a:1", []PACDirective{{Kind: PACHTTP, Host: "a", Port: "1"}}, false},
		{"PROXY a:1; DIRECT", []PACDirective{
			{Kind: PACHTTP, Host: "a", Port: "1"},
			{Kind: PACDirect},
		}, false},
		{"HTTPS tls:443", []PACDirective{{Kind: PACHTTPS, Host: "tls", Port: "443"}}, false},
		{"SOCKS5 s:1080", []PACDirective{{Kind: PACSOCKS5, Host: "s", Port: "1080"}}, false},
		{"SOCKS s:1080", []PACDirective{{Kind: PACSOCKS5, Host: "s", Port: "1080"}}, false},
		// Unknown directives are skipped, per the convention that a client tries
		// what it understands.
		{"NONEXISTENT x; PROXY p:1", []PACDirective{{Kind: PACHTTP, Host: "p", Port: "1"}}, false},
		// A missing port is tolerated.
		{"PROXY p", []PACDirective{{Kind: PACHTTP, Host: "p"}}, false},
		// Nothing usable is an error.
		{"", nil, true},
		{"   ", nil, true},
		{"NONSENSE", nil, true},
	}
	for _, tc := range cases {
		got, err := ParsePACResult(tc.raw)
		if tc.fails {
			if err == nil {
				t.Errorf("ParsePACResult(%q) succeeded, want an error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePACResult(%q): %v", tc.raw, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("ParsePACResult(%q) = %v, want %v", tc.raw, got, tc.want)
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("ParsePACResult(%q)[%d] = %+v, want %+v", tc.raw, i, got[i], tc.want[i])
			}
		}
	}
}

// TestPACDirectiveURL covers the URL rendering used by the resolver.
func TestPACDirectiveURL(t *testing.T) {
	if u := (PACDirective{Kind: PACDirect}).URL(); u != nil {
		t.Errorf("DIRECT rendered %v, want nil", u)
	}
	cases := map[PACDirectiveKind]string{
		PACHTTP:   "http",
		PACHTTPS:  "https",
		PACSOCKS5: "socks5",
	}
	for kind, scheme := range cases {
		d := PACDirective{Kind: kind, Host: "h", Port: "9"}
		u := d.URL()
		if u == nil || u.Scheme != scheme || u.Host != "h:9" {
			t.Errorf("%s rendered %v", d, u)
		}
	}
}

// TestPACMissingFunction covers a document with no FindProxyForURL: it must be
// an error, not a silent disable.
func TestPACMissingFunction(t *testing.T) {
	e := &PACEvaluator{Source: "var x = 1;"}
	_, err := e.Evaluate(context.Background(), mustURL(t, "http://a/"))
	if !errors.Is(err, ErrPACNoFunction) {
		t.Fatalf("err = %v, want ErrPACNoFunction", err)
	}
}

// TestPACSyntaxError covers a document that does not parse.
func TestPACSyntaxError(t *testing.T) {
	e := &PACEvaluator{Source: "function FindProxyForURL(url, host) {"}
	if _, err := e.Evaluate(context.Background(), mustURL(t, "http://a/")); err == nil {
		t.Fatal("a truncated document evaluated")
	}
}

// TestPACScriptThrow covers an exception thrown from the function.
func TestPACScriptThrow(t *testing.T) {
	e := &PACEvaluator{Source: `
function FindProxyForURL(url, host) { throw new Error("bad"); }
`}
	_, err := e.Evaluate(context.Background(), mustURL(t, "http://a/"))
	if err == nil {
		t.Fatal("a throwing script returned a result")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("err = %v, want the thrown message", err)
	}
}

// TestPACTimeout covers the deadline: an infinite loop must be interrupted
// rather than hanging request setup forever.
func TestPACTimeout(t *testing.T) {
	e := &PACEvaluator{
		Source: `
function FindProxyForURL(url, host) { while (true) {} }
`,
		Timeout: 50 * time.Millisecond,
	}
	start := time.Now()
	_, err := e.Evaluate(context.Background(), mustURL(t, "http://a/"))
	if !errors.Is(err, ErrPACTimeout) {
		t.Fatalf("err = %v, want ErrPACTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("interruption took %v", elapsed)
	}
}

// TestPACEmptyReturn covers a function that returns nothing usable.
func TestPACEmptyReturn(t *testing.T) {
	e := &PACEvaluator{Source: `function FindProxyForURL() { return ""; }`}
	_, err := e.Evaluate(context.Background(), mustURL(t, "http://a/"))
	if !errors.Is(err, ErrPACEmptyResult) {
		t.Fatalf("err = %v, want ErrPACEmptyResult", err)
	}
}

// TestPACSandboxHasNoScriptingPower pins the restriction: a PAC document must
// not reach require, fetch, timers or console, because it can come from the
// network.
func TestPACSandboxHasNoScriptingPower(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    var banned = ["require", "fetch", "setTimeout", "setInterval", "console", "XMLHttpRequest", "process"];
    var hits = [];
    for (var i = 0; i < banned.length; i++) {
        if (typeof this[banned[i]] !== "undefined") hits.push(banned[i]);
    }
    return hits.length ? "PROXY leaked:1" : "PROXY clean:1";
}
`
	res := evalPAC(t, source, "http://a/")
	if res.Attempts[0].Host != "clean" {
		t.Errorf("raw = %q: the PAC sandbox exposed scripting globals", res.Raw)
	}
}

// TestPACResolverCaches pins the fetch-once behaviour and the invalidation.
func TestPACResolverCaches(t *testing.T) {
	fetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches++
		_, _ = w.Write([]byte(`function FindProxyForURL(){ return "PROXY cached:1"; }`))
	}))
	defer srv.Close()

	r := &PACResolver{URL: srv.URL}
	for i := 0; i < 3; i++ {
		res, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{})
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if res.Attempts[0].Host != "cached" {
			t.Fatalf("raw = %q", res.Raw)
		}
	}
	if fetches != 1 {
		t.Errorf("fetched %d times, want 1 with a warm cache", fetches)
	}

	r.Invalidate()
	if _, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{}); err != nil {
		t.Fatalf("after invalidate: %v", err)
	}
	if fetches != 2 {
		t.Errorf("fetched %d times after invalidate, want 2", fetches)
	}
}

// TestPACResolverRefetchesWhenStale covers the TTL.
func TestPACResolverRefetchesWhenStale(t *testing.T) {
	fetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches++
		_, _ = w.Write([]byte(`function FindProxyForURL(){ return "PROXY p:1"; }`))
	}))
	defer srv.Close()

	now := time.Now()
	r := &PACResolver{
		URL:      srv.URL,
		CacheTTL: time.Minute,
		Now:      func() time.Time { return now },
	}
	if _, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{}); err != nil {
		t.Fatalf("first: %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{}); err != nil {
		t.Fatalf("second: %v", err)
	}
	if fetches != 2 {
		t.Errorf("fetched %d times across a TTL expiry, want 2", fetches)
	}
}

// TestPACResolverErrors covers the fetch failure modes.
func TestPACResolverErrors(t *testing.T) {
	r := &PACResolver{}
	if _, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{}); !errors.Is(err, ErrPACNoURL) {
		t.Errorf("no URL: err = %v, want ErrPACNoURL", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()
	r = &PACResolver{URL: srv.URL}
	if _, err := r.Evaluate(context.Background(), mustURL(t, "http://a/"), PACEvaluator{}); err == nil {
		t.Error("a 404 PAC document was accepted")
	}
}

// TestResolvePACMode covers the resolver integration: PAC mode picks the
// script's first proxy.
func TestResolvePACMode(t *testing.T) {
	src := stubPAC{result: PACResult{
		Raw: "PROXY pacproxy:3128; DIRECT",
		Attempts: []PACDirective{
			{Kind: PACHTTP, Host: "pacproxy", Port: "3128"},
			{Kind: PACDirect},
		},
	}}
	u, err := Resolve(ResolutionInput{
		TargetURL:   mustURL(t, "https://api.example.com/x"),
		AppSettings: &model.ProxySettings{Mode: model.ProxyPAC, PACURL: "http://cfg/proxy.pac"},
		PACResolver: src,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if u == nil || u.Host != "pacproxy:3128" || u.Scheme != "http" {
		t.Fatalf("proxy = %v, want http://pacproxy:3128", u)
	}
}

// TestResolvePACDirect covers a script that answers DIRECT.
func TestResolvePACDirect(t *testing.T) {
	src := stubPAC{result: PACResult{
		Raw:      "DIRECT",
		Attempts: []PACDirective{{Kind: PACDirect}},
	}}
	u, err := Resolve(ResolutionInput{
		TargetURL:   mustURL(t, "https://internal.corp/x"),
		AppSettings: &model.ProxySettings{Mode: model.ProxyPAC},
		PACResolver: src,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if u != nil {
		t.Errorf("proxy = %v, want a direct connection", u)
	}
}

// TestResolvePACBypassWins keeps an explicit bypass ahead of the script.
func TestResolvePACBypassWins(t *testing.T) {
	called := false
	src := stubPAC{
		result: PACResult{Raw: "PROXY should-not-be-used:1", Attempts: []PACDirective{{Kind: PACHTTP, Host: "should-not-be-used", Port: "1"}}},
		onEval: func() { called = true },
	}
	u, err := Resolve(ResolutionInput{
		TargetURL: mustURL(t, "https://intranet.local/x"),
		AppSettings: &model.ProxySettings{
			Mode: model.ProxyPAC,
			// curl-style NO_PROXY semantics: a bare domain suffix matches the
			// domain and its subdomains. "*.local" is not a supported form.
			Bypass: []string{"intranet.local"},
		},
		PACResolver: src,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if u != nil {
		t.Errorf("proxy = %v, want direct", u)
	}
	if called {
		t.Error("the PAC script ran for a bypassed host")
	}
}

// TestResolvePACMissingResolver covers a configuration error surfacing instead
// of a silent fall-through to the environment.
func TestResolvePACMissingResolver(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://env:8080")
	_, err := Resolve(ResolutionInput{
		TargetURL:   mustURL(t, "https://api.example.com/x"),
		AppSettings: &model.ProxySettings{Mode: model.ProxyPAC},
	})
	if !errors.Is(err, ErrPACNoSource) {
		t.Fatalf("err = %v, want ErrPACNoSource", err)
	}
}

// TestResolvePACPropagatesErrors keeps a broken PAC document from becoming a
// silent direct connection.
func TestResolvePACPropagatesErrors(t *testing.T) {
	src := stubPAC{err: errors.New("fetch failed")}
	_, err := Resolve(ResolutionInput{
		TargetURL:   mustURL(t, "https://api.example.com/x"),
		AppSettings: &model.ProxySettings{Mode: model.ProxyPAC},
		PACResolver: src,
	})
	if err == nil || !strings.Contains(err.Error(), "fetch failed") {
		t.Fatalf("err = %v, want the fetch error", err)
	}
}

// TestResolvePACEndToEnd goes through the real evaluator and resolver, which is
// the path production takes.
func TestResolvePACEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`
function FindProxyForURL(url, host) {
    if (shExpMatch(host, "*.internal")) return "DIRECT";
    return "SOCKS5 socks.example:1080";
}
`))
	}))
	defer srv.Close()

	r := &PACResolver{URL: srv.URL}
	input := func(target string) ResolutionInput {
		return ResolutionInput{
			TargetURL:   mustURL(t, target),
			AppSettings: &model.ProxySettings{Mode: model.ProxyPAC, PACURL: srv.URL},
			PACResolver: NewPACSource(r),
		}
	}

	u, err := Resolve(input("http://db.internal/x"))
	if err != nil {
		t.Fatalf("internal: %v", err)
	}
	if u != nil {
		t.Errorf("internal proxy = %v, want direct", u)
	}

	u, err = Resolve(input("https://api.public/x"))
	if err != nil {
		t.Fatalf("public: %v", err)
	}
	if u == nil || u.Scheme != "socks5" || u.Host != "socks.example:1080" {
		t.Errorf("public proxy = %v, want socks5://socks.example:1080", u)
	}
}

// stubPAC is a PACSource with fixed behaviour.
type stubPAC struct {
	result PACResult
	err    error
	onEval func()
}

func (s stubPAC) Evaluate(context.Context, *url.URL) (PACResult, error) {
	if s.onEval != nil {
		s.onEval()
	}
	return s.result, s.err
}
