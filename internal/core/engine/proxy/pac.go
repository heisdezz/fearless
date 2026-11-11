package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// PAC (Proxy Auto-Configuration) support.
//
// A PAC document defines
//
//	function FindProxyForURL(url, host) { return "PROXY proxy.example:8080"; }
//
// and returns a list of attempts, best first:
//
//	"PROXY host:port"    an HTTP proxy
//	"HTTPS host:port"    an HTTP proxy reached over TLS
//	"SOCKS5 host:port"   a SOCKS5 proxy
//	"SOCKS host:port"    a SOCKS4 proxy, same as SOCKS5 for our purposes
//	"DIRECT"             no proxy
//
// The acceptance criterion (docs/progress/packages.md, `PROXY`) asks for the
// restricted goja globals: dnsResolve, isInNet, shExpMatch, myIpAddress,
// dnsDomainIs. Those are the "safe subset" from the original Netscape
// specification, and they are installed on a purpose-built runtime that has
// none of the scripting sandbox's power — no require, no timers, no fetch, no
// console. A PAC file fetched from a network location is untrusted input.
//
// Source: Netscape "Proxy Auto-Config" specification; bruno-app's proxy
// resolution defers PAC to the OS, so this is the first full implementation.

// PACResult is the parsed outcome of FindProxyForURL.
type PACResult struct {
	// Raw is the string the script returned.
	Raw string
	// Attempts are the parsed directives, in order.
	Attempts []PACDirective
}

// DirectTransfer reports the script's first preference was DIRECT, before any
// proxy is tried. Which proxy to *use* is still the caller's decision: see
// Resolve's docs for why several attempts cannot be expressed as one URL.
func (r PACResult) DirectTransfer() bool {
	return len(r.Attempts) == 0 || r.Attempts[0].Kind == PACDirect
}

// PACDirectiveKind classifies one PAC return token.
type PACDirectiveKind int

const (
	// PACDirect is the DIRECT directive: connect without a proxy.
	PACDirect PACDirectiveKind = iota
	// PACHTTP is PROXY: a plain HTTP proxy.
	PACHTTP
	// PACHTTPS is HTTPS: an HTTP proxy reached over TLS.
	PACHTTPS
	// PACSOCKS5 is SOCKS5 or SOCKS: a SOCKS proxy.
	PACSOCKS5
)

// PACDirective is one parsed return token.
type PACDirective struct {
	Kind PACDirectiveKind
	// Host and Port are empty for PACDirect.
	Host string
	Port string
}

// URL renders the directive as a proxy URL, or nil for DIRECT.
func (d PACDirective) URL() *url.URL {
	if d.Kind == PACDirect {
		return nil
	}
	scheme := "http"
	switch d.Kind {
	case PACHTTPS:
		scheme = "https"
	case PACSOCKS5:
		scheme = "socks5"
	}
	host := d.Host
	if d.Port != "" {
		host = net.JoinHostPort(d.Host, d.Port)
	}
	return &url.URL{Scheme: scheme, Host: host}
}

// String renders the directive back to PAC syntax, for logs.
func (d PACDirective) String() string {
	switch d.Kind {
	case PACDirect:
		return "DIRECT"
	case PACHTTPS:
		return "HTTPS " + net.JoinHostPort(d.Host, d.Port)
	case PACSOCKS5:
		return "SOCKS5 " + net.JoinHostPort(d.Host, d.Port)
	}
	return "PROXY " + net.JoinHostPort(d.Host, d.Port)
}

// PACErrors.
var (
	// ErrPACNoFunction means the document parsed but defines no
	// FindProxyForURL. It is an error rather than a "direct" result because a
	// broken PAC file silently disabling proxying is a security decision the
	// user should see.
	ErrPACNoFunction = errors.New("proxy: the PAC document defines no FindProxyForURL")
	// ErrPACEmptyResult means the function returned nothing usable.
	ErrPACEmptyResult = errors.New("proxy: FindProxyForURL returned no proxy directive")
	// ErrPACTimeout means evaluation exceeded the deadline.
	ErrPACTimeout = errors.New("proxy: PAC evaluation timed out")
)

// PACEvaluator evaluates one PAC document.
//
// A goja runtime is single-goroutine state and the script may be called for
// every request, so the evaluator creates a fresh runtime per evaluation. That
// costs one script compile per request; PAC documents are small and the cost is
// dwarfed by the DNS lookups the globals perform. A caller that needs to avoid
// even that should cache at a higher level, which is what PACResolver's
// document cache does.
type PACEvaluator struct {
	// Source is the PAC document.
	Source string
	// Timeout bounds evaluation. Zero means DefaultPACTimeout.
	Timeout time.Duration
	// LookupHost resolves a name to IPs. Nil means net.LookupHost.
	LookupHost func(ctx context.Context, host string) ([]string, error)
	// MyIP returns this machine's address, for myIpAddress(). Nil means detect.
	MyIP func() string
}

// DefaultPACTimeout bounds a single FindProxyForURL call. It is short because
// evaluation happens inline in request setup; a PAC file that needs more than
// this is pathological, and DNS lookups are already bounded by the context.
const DefaultPACTimeout = 2 * time.Second

// Evaluate runs FindProxyForURL for one URL on its own runtime and context.
func (e *PACEvaluator) Evaluate(ctx context.Context, target *url.URL) (PACResult, error) {
	if target == nil {
		return PACResult{}, errors.New("proxy: PAC evaluation needs a target URL")
	}
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = DefaultPACTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	vm, err := newPACRuntime(ctx, e)
	if err != nil {
		return PACResult{}, err
	}

	if _, err := vm.RunString(e.Source); err != nil {
		return PACResult{}, fmt.Errorf("proxy: loading PAC document: %w", err)
	}

	fn, ok := goja.AssertFunction(vm.Get("FindProxyForURL"))
	if !ok {
		return PACResult{}, ErrPACNoFunction
	}

	stop := watchInterrupt(ctx, vm)
	defer stop()

	host := target.Hostname()
	// The first argument is the URL string, not a context object. Every PAC file
	// in practice treats it as a string (`url.substring(0, 5) == "http:"`), and
	// the Netscape spec's own examples assume that.
	value, err := fn(goja.Undefined(), vm.ToValue(target.String()), vm.ToValue(host))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return PACResult{}, PACTimeoutError(ctxErr)
		}
		return PACResult{}, fmt.Errorf("proxy: FindProxyForURL: %w", err)
	}

	raw := strings.TrimSpace(value.String())
	if raw == "" {
		return PACResult{}, ErrPACEmptyResult
	}
	attempts, err := ParsePACResult(raw)
	if err != nil {
		return PACResult{}, err
	}
	return PACResult{Raw: raw, Attempts: attempts}, nil
}

// PACTimeoutError wraps a context cause as ErrPACTimeout.
func PACTimeoutError(cause error) error {
	return fmt.Errorf("%w: %v", ErrPACTimeout, cause)
}

// watchInterrupt stops a runaway script when ctx is done, returning a stop
// function the caller must call. goja has no context support of its own, so
// the interrupt is the only way to break a synchronous loop.
func watchInterrupt(ctx context.Context, vm *goja.Runtime) func() {
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		select {
		case <-ctx.Done():
			vm.Interrupt(ErrPACTimeout.Error())
		case <-done:
		}
	}()
	return func() {
		close(done)
		<-finished
		// A stale interrupt would abort the next call; the runtime is
		// per-evaluation so this is belt and braces.
		vm.ClearInterrupt()
	}
}

// newPACRuntime builds the restricted runtime.
//
// It is a fresh goja runtime with exactly the PAC helper functions installed and
// nothing else: no require, no timers, no fetch, no console. A PAC document can
// come from a network location and is therefore untrusted input, so it must not
// inherit the scripting sandbox's capabilities.
func newPACRuntime(ctx context.Context, e *PACEvaluator) (*goja.Runtime, error) {
	vm := goja.New()
	vm.SetMaxCallStackSize(1024)

	lookupHost := e.LookupHost
	if lookupHost == nil {
		lookupHost = net.DefaultResolver.LookupHost
	}
	myIP := e.MyIP
	if myIP == nil {
		myIP = detectMyIP
	}

	gl := pacGlobals{vm: vm, ctx: ctx, lookupHost: lookupHost, myIP: myIP}
	if err := gl.install(); err != nil {
		return nil, err
	}
	return vm, nil
}

// pacGlobals installs the safe subset. Each function is a plain Go func; goja
// converts them on Set.
type pacGlobals struct {
	vm         *goja.Runtime
	ctx        context.Context
	lookupHost func(context.Context, string) ([]string, error)
	myIP       func() string
}

// install sets the helpers. Functions that cannot resolve a name return false
// or an empty string rather than throwing: a PAC script is expected to tolerate
// a failed DNS lookup and fall back, and Bruno's own behaviour is to continue
// rather than abort the request.
func (g *pacGlobals) install() error {
	set := func(name string, fn any) error {
		if err := g.vm.Set(name, fn); err != nil {
			return fmt.Errorf("proxy: installing PAC global %s: %w", name, err)
		}
		return nil
	}

	helpers := []struct {
		name string
		fn   any
	}{
		{"dnsDomainIs", g.dnsDomainIs},
		{"shExpMatch", shExpMatch},
		{"isPlainHostName", isPlainHostName},
		{"isInNet", g.isInNet},
		{"dnsResolve", g.dnsResolve},
		{"myIpAddress", func() string { return g.myIP() }},
		{"localHostOrDomainIs", localHostOrDomainIs},
		{"dnsDomainLevels", dnsDomainLevels},
		{"weekdayRange", weekdayRange},
		{"dateRange", dateRange},
		{"timeRange", timeRange},
		{"alert", func(string) {}},
	}
	for _, h := range helpers {
		if err := set(h.name, h.fn); err != nil {
			return err
		}
	}
	return nil
}

// dnsDomainIs reports whether host is inside domain, comparing the tail.
func (g *pacGlobals) dnsDomainIs(host, domain string) bool {
	return strings.HasSuffix(strings.ToLower(host), strings.ToLower(domain))
}

// shExpMatch is the shell-style matcher PAC files use for host patterns.
// Matching is case-insensitive and supports the two wildcards PAC files
// actually use: * (any run) and ? (one character). It does not use path.Match
// because that refuses to let * cross a '/' and these patterns are hosts and
// URLs, where * must cross everything.
func shExpMatch(text, pattern string) bool {
	return globMatch(strings.ToLower(text), strings.ToLower(pattern))
}

// globMatch matches text against a pattern of literals, * and ? by walking both
// with backtracking on *. It runs in O(len(text)*len(pattern)) worst case, which
// is irrelevant for the strings involved.
func globMatch(text, pattern string) bool {
	t, p := 0, 0
	star, resume := -1, 0
	for t < len(text) {
		switch {
		case p < len(pattern) && (pattern[p] == '?' || pattern[p] == text[t]):
			t++
			p++
		case p < len(pattern) && pattern[p] == '*':
			star = p
			resume = t
			p++
		case star >= 0:
			// Let the star swallow one more character and retry.
			resume++
			t = resume
			p = star + 1
		default:
			return false
		}
	}
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}
	return p == len(pattern)
}

// isPlainHostName is true when the host has no dots.
func isPlainHostName(host string) bool {
	return !strings.Contains(host, ".")
}

// dnsDomainLevels counts the dots in a host.
func dnsDomainLevels(host string) int {
	return strings.Count(host, ".")
}

// localHostOrDomainIs is true when host is name, or name with the parts of host
// beyond name's own labels appended.
func localHostOrDomainIs(host, hostdom string) bool {
	if strings.EqualFold(host, hostdom) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(hostdom), strings.ToLower(host)+".")
}

// dnsResolve returns the first address for host, or "" when it does not resolve.
func (g *pacGlobals) dnsResolve(host string) string {
	addrs, err := g.lookupHost(g.ctx, host)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// isInNet is true when the resolved address of host is inside pattern/netmask.
// A name that does not resolve is not in any net, per the spec.
func (g *pacGlobals) isInNet(host, pattern, mask string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		resolved := g.dnsResolve(host)
		if resolved == "" {
			return false
		}
		ip = net.ParseIP(resolved)
	}
	want := net.ParseIP(pattern)
	maskIP := net.ParseIP(mask)
	if ip == nil || want == nil || maskIP == nil {
		return false
	}

	ip4, want4, mask4 := ip.To4(), want.To4(), maskIP.To4()
	if ip4 != nil && want4 != nil && mask4 != nil {
		return ip4.Mask(net.IPMask(mask4)).Equal(want4.Mask(net.IPMask(mask4)))
	}

	ip16, want16, mask16 := ip.To16(), want.To16(), maskIP.To16()
	if ip16 == nil || want16 == nil || mask16 == nil {
		return false
	}
	return ip16.Mask(net.IPMask(mask16)).Equal(want16.Mask(net.IPMask(mask16)))
}

// weekdayRange implements the day predicate. Only the single-day and
// two-day-range forms are supported; the optional GMT argument is accepted and
// ignored, because PAC callers use local time and a UTC switch would change
// which day a proxy is chosen — a surprising effect for a rarely used feature.
func weekdayRange(a string, rest ...goja.Value) bool {
	args := packArgs(rest)
	now := time.Now()
	want := func(s string) (time.Weekday, bool) {
		switch strings.ToUpper(s) {
		case "SUN":
			return time.Sunday, true
		case "MON":
			return time.Monday, true
		case "TUE":
			return time.Tuesday, true
		case "WED":
			return time.Wednesday, true
		case "THU":
			return time.Thursday, true
		case "FRI":
			return time.Friday, true
		case "SAT":
			return time.Saturday, true
		}
		return 0, false
	}
	from, ok := want(a)
	if !ok {
		return false
	}
	if len(args) == 0 || strings.EqualFold(args[len(args)-1], "GMT") {
		return now.Weekday() == from
	}
	to, ok := want(args[0])
	if !ok {
		return false
	}
	d := now.Weekday()
	if from <= to {
		return d >= from && d <= to
	}
	// A wrapping range such as SAT-SUN.
	return d >= from || d <= to
}

// dateRange implements the day-of-month predicate: a single day, a range, or
// nothing (always true).
func dateRange(args ...goja.Value) bool {
	vals := packArgs(args)
	if len(vals) == 0 {
		return true
	}
	nums := make([]int, 0, len(vals))
	for _, v := range vals {
		if strings.EqualFold(v, "GMT") {
			continue
		}
		n, ok := atoiSafe(v)
		if !ok {
			return false
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return true
	}
	day := time.Now().Day()
	if len(nums) == 1 {
		return day == nums[0]
	}
	lo, hi := nums[0], nums[1]
	if lo > hi {
		return day >= lo || day <= hi
	}
	return day >= lo && day <= hi
}

// timeRange implements the hour/minute/second predicate. Arguments come in
// pairs or a single hour; a trailing GMT is accepted and ignored for the same
// reason as weekdayRange.
func timeRange(args ...goja.Value) bool {
	vals := packArgs(args)
	nums := make([]int, 0, len(vals))
	for _, v := range vals {
		if strings.EqualFold(v, "GMT") {
			continue
		}
		n, ok := atoiSafe(v)
		if !ok {
			return false
		}
		nums = append(nums, n)
	}
	now := time.Now()
	// Compare as seconds since midnight so an hour-only range covers the whole
	// hour, which is what a PAC author expects.
	cur := now.Hour()*3600 + now.Minute()*60 + now.Second()
	if len(nums) == 0 {
		return true
	}
	if len(nums) == 1 {
		lo := nums[0] * 3600
		return cur >= lo && cur < lo+3600
	}
	lo := nums[0] * 3600
	hi := nums[1]*3600 + 3599
	if lo > hi {
		return cur >= lo || cur <= hi
	}
	return cur >= lo && cur <= hi
}

// packArgs converts goja's variadic arguments into strings for arity-agnostic
// predicates.
func packArgs(rest []goja.Value) []string {
	out := make([]string, 0, len(rest))
	for _, v := range rest {
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			continue
		}
		out = append(out, v.String())
	}
	return out
}

// atoiSafe parses a decimal integer, reporting failure instead of returning 0
// so a malformed range does not silently match midnight.
func atoiSafe(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// detectMyIP returns a non-loopback IPv4 address for this machine, or
// "127.0.0.1" when none can be found. It does not open a connection: a UDP
// "dial" only picks a route and sends nothing.
func detectMyIP() string {
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}
			if ip4 := ipNet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

// ParsePACResult parses a FindProxyForURL return value into directives.
//
// The grammar is semicolon-separated tokens; unknown tokens are skipped rather
// than failing the request, because the convention is that a client tries what
// it understands. A result with no usable token is an error.
func ParsePACResult(raw string) ([]PACDirective, error) {
	var out []PACDirective
	for _, token := range strings.Split(raw, ";") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		fields := strings.Fields(token)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "DIRECT":
			out = append(out, PACDirective{Kind: PACDirect})
		case "PROXY", "HTTPS", "SOCKS5", "SOCKS":
			if len(fields) < 2 {
				continue
			}
			host, port := splitHostPort(fields[1])
			if host == "" {
				continue
			}
			kind := PACHTTP
			switch strings.ToUpper(fields[0]) {
			case "HTTPS":
				kind = PACHTTPS
			case "SOCKS5", "SOCKS":
				kind = PACSOCKS5
			}
			out = append(out, PACDirective{Kind: kind, Host: host, Port: port})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrPACEmptyResult, raw)
	}
	return out, nil
}

// splitHostPort splits "host:port", tolerating IPv6 brackets and a missing
// port.
func splitHostPort(s string) (host, port string) {
	if h, p, err := net.SplitHostPort(s); err == nil {
		return h, p
	}
	// No port, or an unbracketed IPv6 literal.
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return strings.Trim(s, "[]"), ""
	}
	return s, ""
}

// PACResolver fetches a PAC document once and evaluates it per request.
//
// The document is fetched lazily and cached for CacheTTL. A fetch failure is
// cached for a much shorter time so a transient network error does not pin the
// app to "no proxy" for the whole TTL.
type PACResolver struct {
	// URL is the PAC document location, http(s) or file.
	URL string
	// Client fetches the document. Nil means a client with a short timeout.
	Client *http.Client
	// CacheTTL is how long a fetched document is reused. Zero means
	// DefaultPACCacheTTL.
	CacheTTL time.Duration
	// Now allows tests to control the clock.
	Now func() time.Time

	mu       sync.Mutex
	doc      string
	fetched  time.Time
	fetchErr error
}

// DefaultPACCacheTTL matches the interval most browsers use between PAC
// refetches.
const DefaultPACCacheTTL = 15 * time.Minute

// pacFetchTimeout bounds loading the document itself.
const pacFetchTimeout = 10 * time.Second

// ErrPACNoURL means the resolver was used without a document location.
var ErrPACNoURL = errors.New("proxy: PAC mode needs a PAC URL")

// fetch returns the document, loading it if the cache is cold or stale.
func (p *PACResolver) fetch(ctx context.Context) (string, error) {
	ttl := p.CacheTTL
	if ttl <= 0 {
		ttl = DefaultPACCacheTTL
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.doc != "" && now().Sub(p.fetched) < ttl {
		return p.doc, nil
	}
	if p.fetchErr != nil && now().Sub(p.fetched) < min(ttl, time.Minute) {
		return "", p.fetchErr
	}

	doc, err := p.load(ctx)
	p.fetched = now()
	if err != nil {
		p.fetchErr = err
		return "", err
	}
	p.doc, p.fetchErr = doc, nil
	return doc, nil
}

// load reads the document from its URL.
func (p *PACResolver) load(ctx context.Context) (string, error) {
	if strings.TrimSpace(p.URL) == "" {
		return "", ErrPACNoURL
	}
	ctx, cancel := context.WithTimeout(ctx, pacFetchTimeout)
	defer cancel()

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: pacFetchTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return "", fmt.Errorf("proxy: PAC request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("proxy: fetching PAC document: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy: fetching PAC document: %s", resp.Status)
	}
	// A PAC file is small; cap it so a misconfigured URL cannot stream forever.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPACBytes))
	if err != nil {
		return "", fmt.Errorf("proxy: reading PAC document: %w", err)
	}
	return string(body), nil
}

// maxPACBytes caps a PAC document at 1 MiB. Real ones are a few kilobytes.
const maxPACBytes = 1 << 20

// Evaluate loads (or reuses) the document and runs FindProxyForURL for target.
// It is the entry point Resolve uses in PAC mode.
func (p *PACResolver) Evaluate(ctx context.Context, target *url.URL, eval PACEvaluator) (PACResult, error) {
	doc, err := p.fetch(ctx)
	if err != nil {
		return PACResult{}, err
	}
	eval.Source = doc
	return eval.Evaluate(ctx, target)
}

// Invalidate drops the cached document so the next evaluation refetches it.
// The settings screen calls it when the PAC URL changes.
func (p *PACResolver) Invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.doc, p.fetchErr = "", nil
	p.fetched = time.Time{}
}
