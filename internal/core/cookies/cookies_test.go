package cookies

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func newJar(t *testing.T, opts ...Option) *Jar {
	t.Helper()
	j, err := New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = j.Close() })
	return j
}

func addRaw(t *testing.T, j *Jar, line, rawURL string) {
	t.Helper()
	if err := j.AddRaw(line, mustURL(t, rawURL)); err != nil {
		t.Fatalf("AddRaw(%q, %q): %v", line, rawURL, err)
	}
}

func header(t *testing.T, j *Jar, rawURL string) string {
	t.Helper()
	s, err := j.HeaderFor(mustURL(t, rawURL))
	if err != nil {
		t.Fatalf("HeaderFor(%q): %v", rawURL, err)
	}
	return s
}

func TestAddRawAndHeaderFor(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "a=1", "https://example.com/dir/page")
	addRaw(t, j, "v=x=y", "https://example.com/dir/page") // value keeps its '='

	got := header(t, j, "https://example.com/dir/page")
	if !strings.Contains(got, "a=1") || !strings.Contains(got, "v=x=y") {
		t.Fatalf("HeaderFor = %q, want both cookies", got)
	}
	if got := header(t, j, "https://example.com/other"); got != "" {
		t.Fatalf("default path /dir must not apply to /other, got %q", got)
	}
}

func TestDomainRules(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "d=1; Domain=example.com", "https://www.example.com/x")
	addRaw(t, j, "h=1", "https://www.example.com/")

	for _, u := range []string{"https://www.example.com/", "https://api.example.com/", "https://example.com/"} {
		if got := header(t, j, u); !strings.Contains(got, "d=1") {
			t.Errorf("domain cookie must reach %s, got %q", u, got)
		}
	}
	if got := header(t, j, "https://example.org/"); strings.Contains(got, "d=1") {
		t.Errorf("domain cookie leaked to example.org: %q", got)
	}
	if got := header(t, j, "https://api.example.com/"); strings.Contains(got, "h=1") {
		t.Errorf("host-only cookie leaked to a subdomain: %q", got)
	}
	if got := header(t, j, "https://www.example.com/"); !strings.Contains(got, "h=1") {
		t.Errorf("host-only cookie missing on its own host: %q", got)
	}
}

// TestPathMatchEmpty guards the one input pathMatch cannot tolerate. The public
// API normalises an empty path to "/" on every write, so this is defence in
// depth against a future caller that skips it: it used to panic on
// cookiePath[len(cookiePath)-1].
func TestPathMatchEmpty(t *testing.T) {
	if pathMatch("", "/anything") {
		t.Error("an empty cookie path matched")
	}
	if pathMatch("", "") {
		t.Error("an empty cookie path matched the empty request path")
	}
}

func TestPathRules(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "p=1; Path=/api", "https://h.test/")
	addRaw(t, j, "q=1", "https://h.test/dir/page") // default path /dir

	cases := []struct {
		url  string
		want string
	}{
		{"https://h.test/api/v1", "p=1"},
		{"https://h.test/api", "p=1"},
		{"https://h.test/apix", ""}, // prefix must end on a boundary
		{"https://h.test/", ""},
		{"https://h.test/dir", "q=1"},
		{"https://h.test/dir/x", "q=1"},
	}
	for _, tc := range cases {
		got := header(t, j, tc.url)
		if tc.want == "" {
			if got != "" {
				t.Errorf("HeaderFor(%q) = %q, want empty", tc.url, got)
			}
			continue
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("HeaderFor(%q) = %q, want it to contain %q", tc.url, got, tc.want)
		}
	}
}

func TestSecureAndTrustworthyOrigins(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "s=1; Secure", "https://secure.test/")
	addRaw(t, j, "s2=1; Secure", "http://127.0.0.1/")
	addRaw(t, j, "s3=1; Secure", "http://localhost/")
	addRaw(t, j, "s4=1; Secure", "wss://ws.test/")

	if got := header(t, j, "https://secure.test/"); !strings.Contains(got, "s=1") {
		t.Errorf("https must carry Secure cookies, got %q", got)
	}
	if got := header(t, j, "http://secure.test/"); got != "" {
		t.Errorf("plain http must not carry Secure cookies, got %q", got)
	}
	if got := header(t, j, "http://127.0.0.1/"); !strings.Contains(got, "s2=1") {
		t.Errorf("loopback is trustworthy over http (§3), got %q", got)
	}
	if got := header(t, j, "http://localhost/"); !strings.Contains(got, "s3=1") {
		t.Errorf("localhost is trustworthy over http (§3), got %q", got)
	}
	if got := header(t, j, "wss://ws.test/"); !strings.Contains(got, "s4=1") {
		t.Errorf("wss is trustworthy like https (§3), got %q", got)
	}
	if got := header(t, j, "ws://ws.test/"); got != "" {
		t.Errorf("ws behaves like plain http, got %q", got)
	}
}

func TestSchemeParticipation(t *testing.T) {
	j := newJar(t)
	if err := j.AddRaw("f=1", mustURL(t, "file:///tmp/x")); err != nil {
		t.Fatalf("file URL must be a no-op, got %v", err)
	}
	if err := j.AddRaw("f=1", mustURL(t, "ftp://files.test/")); err != nil {
		t.Fatalf("ftp URL must be a no-op, got %v", err)
	}
	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 0 {
		t.Fatalf("nothing may be stored for non-HTTP origins, got %+v", ds)
	}
	cs, err := j.CookiesFor(mustURL(t, "file:///tmp/x"))
	if err != nil || len(cs) != 0 {
		t.Fatalf("CookiesFor(file) = %v, %v; want empty, nil", cs, err)
	}
}

func TestExpiryAndRemoval(t *testing.T) {
	j := newJar(t)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC1123)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC1123)

	addRaw(t, j, "e=1; Expires="+future, "https://x.test/")
	if got := header(t, j, "https://x.test/"); !strings.Contains(got, "e=1") {
		t.Fatalf("future expiry must be visible, got %q", got)
	}

	addRaw(t, j, "e=2; Max-Age=-1", "https://x.test/")
	if got := header(t, j, "https://x.test/"); got != "" {
		t.Errorf("Max-Age=-1 must delete, got %q", got)
	}

	addRaw(t, j, "e=3", "https://x.test/")
	addRaw(t, j, "e=3; Max-Age=0", "https://x.test/") // parser maps 0 to -1
	if got := header(t, j, "https://x.test/"); got != "" {
		t.Errorf("Max-Age=0 must delete, got %q", got)
	}

	addRaw(t, j, "e=4", "https://x.test/")
	addRaw(t, j, "e=4; Expires="+past, "https://x.test/")
	if got := header(t, j, "https://x.test/"); got != "" {
		t.Errorf("past Expires must delete, got %q", got)
	}
}

func TestHostPrefixRules(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		url     string
		wantErr bool
	}{
		{"missing secure", "__Host-TOKEN=abc", "https://h.test/", true},
		{"missing path", "__Host-TOKEN=abc; Secure", "https://h.test/dir/page", true},
		{"domain attribute", "__Host-TOKEN=abc; Secure; Path=/; Domain=h.test", "https://h.test/", true},
		{"compliant", "__Host-TOKEN=abc; Secure; Path=/", "https://h.test/", false},
		{"compliant default path", "__Host-TOKEN=abc; Secure", "https://h.test/", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			j := newJar(t)
			err := j.AddRaw(tc.line, mustURL(t, tc.url))
			if tc.wantErr {
				if !errors.Is(err, ErrRejected) {
					t.Fatalf("AddRaw = %v, want ErrRejected", err)
				}
				ds, derr := j.Domains()
				if derr != nil {
					t.Fatalf("Domains: %v", derr)
				}
				if len(ds) != 0 {
					t.Fatalf("rejected cookie must not be stored, got %+v", ds)
				}
				return
			}
			if err != nil {
				t.Fatalf("AddRaw: %v", err)
			}
			if got := header(t, j, tc.url); !strings.Contains(got, "__Host-TOKEN=abc") {
				t.Fatalf("compliant __Host- cookie must be stored, got %q", got)
			}
		})
	}
}

func TestJarRejections(t *testing.T) {
	j := newJar(t)
	cases := []struct {
		name string
		line string
		url  string
	}{
		{"public suffix domain", "a=1; Domain=co.uk", "http://x.co.uk/"},
		{"public suffix com", "a=1; Domain=com", "http://example.com/"},
		{"domain mismatch", "a=1; Domain=other.com", "http://www.example.com/"},
		{"ip domain mismatch", "a=1; Domain=10.0.0.1", "http://127.0.0.1/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := j.AddRaw(tc.line, mustURL(t, tc.url))
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("AddRaw = %v, want ErrRejected", err)
			}
		})
	}
	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 0 {
		t.Fatalf("no rejected cookie may be stored, got %+v", ds)
	}
}

func TestIPHostCookies(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "a=1; Domain=127.0.0.1", "http://127.0.0.1:8080/")
	if got := header(t, j, "http://127.0.0.1:9999/"); !strings.Contains(got, "a=1") {
		t.Fatalf("IP cookies ignore the port, got %q", got)
	}
	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 1 || len(ds[0].Cookies) != 1 {
		t.Fatalf("Domains = %+v, want one cookie", ds)
	}
	if !ds[0].Cookies[0].HostOnly {
		t.Error("a cookie for an IP host is host-only")
	}
}

func TestMergeHeaderJarWins(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "b=9", "https://m.test/")
	addRaw(t, j, "c=3", "https://m.test/")
	u := mustURL(t, "https://m.test/")

	got, err := j.MergeHeaderFor(u, "a=1; b=2")
	if err != nil {
		t.Fatalf("MergeHeaderFor: %v", err)
	}
	// §6: jar wins per name, explicit names keep their position, jar-only
	// names append (reference: {...parseCookies(existing), ...parseCookies(jar)}).
	if want := "a=1; b=9; c=3"; got != want {
		t.Fatalf("MergeHeaderFor = %q, want %q", got, want)
	}

	// Send toggle off: the explicit header passes through untouched.
	j.SetSendCookies(false)
	got, err = j.MergeHeaderFor(u, "a=1; b=2")
	if err != nil {
		t.Fatalf("MergeHeaderFor: %v", err)
	}
	if want := "a=1; b=2"; got != want {
		t.Fatalf("send-off merge = %q, want %q", got, want)
	}
	j.SetSendCookies(true)

	// Empty jar: explicit passes through, even unparseable.
	empty := newJar(t)
	got, err = empty.MergeHeaderFor(u, "not a cookie header")
	if err != nil {
		t.Fatalf("MergeHeaderFor: %v", err)
	}
	if want := "not a cookie header"; got != want {
		t.Fatalf("empty-jar merge = %q, want %q", got, want)
	}
}

func TestDomainsExport(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "b=1", "https://one.test/")
	addRaw(t, j, "a=2", "https://one.test/")
	addRaw(t, j, "c=3", "https://two.test/")

	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 2 {
		t.Fatalf("Domains len = %d, want 2", len(ds))
	}
	if ds[0].Domain != "one.test" || ds[1].Domain != "two.test" {
		t.Errorf("domains not sorted: %q, %q", ds[0].Domain, ds[1].Domain)
	}
	if len(ds[0].Cookies) != 2 || ds[0].Cookies[0].Key != "a" || ds[0].Cookies[1].Key != "b" {
		t.Errorf("cookies not sorted by name: %+v", ds[0].Cookies)
	}
}

func TestDelete(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "k=1; Path=/api", "https://d.test/")
	addRaw(t, j, "k=2", "https://d.test/")

	if err := j.Delete("d.test", "/api", "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got := header(t, j, "https://d.test/api/x")
	if strings.Contains(got, "k=1;") || strings.HasSuffix(got, "k=1") {
		t.Errorf("k=1 at /api must be gone, got %q", got)
	}
	if !strings.Contains(got, "k=2") {
		t.Errorf("k=2 at / must survive, got %q", got)
	}
	if err := j.Delete("d.test", "/api", "k"); err != nil {
		t.Errorf("second delete must be a no-op, got %v", err)
	}
	if err := j.Delete("other.test", "/", "k"); err != nil {
		t.Errorf("unknown cookie delete must be a no-op, got %v", err)
	}
	if err := j.Delete("", "/", "k"); err == nil {
		t.Error("empty domain must be an error")
	}
}

func TestDeleteForDomain(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "a=1", "https://one.test/")
	addRaw(t, j, "b=1", "https://two.test/")

	if err := j.DeleteForDomain("one.test"); err != nil {
		t.Fatalf("DeleteForDomain: %v", err)
	}
	if got := header(t, j, "https://one.test/"); got != "" {
		t.Errorf("one.test must be empty, got %q", got)
	}
	if got := header(t, j, "https://two.test/"); !strings.Contains(got, "b=1") {
		t.Errorf("two.test must survive, got %q", got)
	}
}

func TestAddForDomain(t *testing.T) {
	j := newJar(t)
	added, err := j.AddForDomain(events.DomainCookies{
		Domain: "example.com",
		Cookies: []model.Cookie{
			{Key: "a", Value: "1"}, // group domain is the fallback
			{Key: "b", Value: "2", Domain: "other.org", HostOnly: true},
		},
	})
	if err != nil {
		t.Fatalf("AddForDomain: %v", err)
	}
	if added != 2 {
		t.Fatalf("added = %d, want 2", added)
	}
	if got := header(t, j, "https://example.com/"); !strings.Contains(got, "a=1") {
		t.Errorf("fallback-domain cookie missing: %q", got)
	}
	if got := header(t, j, "https://sub.example.com/"); !strings.Contains(got, "a=1") {
		t.Errorf("UI cookies default to domain cookies: %q", got)
	}
	if got := header(t, j, "https://other.org/"); !strings.Contains(got, "b=2") {
		t.Errorf("host-only UI cookie missing: %q", got)
	}
	if got := header(t, j, "https://x.other.org/"); strings.Contains(got, "b=2") {
		t.Errorf("host-only UI cookie leaked to a subdomain: %q", got)
	}

	// __Host- rules apply to UI edits too: skipped unless Secure, Path=/ and
	// host-only.
	added, err = j.AddForDomain(events.DomainCookies{
		Domain: "h.test",
		Cookies: []model.Cookie{
			{Key: "__Host-X", Value: "1", Secure: true, Path: "/"},
		},
	})
	if err != nil || added != 0 {
		t.Fatalf("non-host-only __Host- must be skipped, got added=%d err=%v", added, err)
	}
	added, err = j.AddForDomain(events.DomainCookies{
		Domain: "h.test",
		Cookies: []model.Cookie{
			{Key: "__Host-X", Value: "1", Secure: true, Path: "/", HostOnly: true},
		},
	})
	if err != nil || added != 1 {
		t.Fatalf("compliant __Host- must be stored, got added=%d err=%v", added, err)
	}

	// MaxAge < 0 deletes (§10), and deletions are not counted as additions.
	added, err = j.AddForDomain(events.DomainCookies{
		Domain: "example.com",
		Cookies: []model.Cookie{
			{Key: "a", Value: "x", MaxAge: -1},
		},
	})
	if err != nil || added != 0 {
		t.Fatalf("MaxAge=-1: added=%d err=%v, want 0, nil", added, err)
	}
	if got := header(t, j, "https://example.com/"); strings.Contains(got, "a=") {
		t.Errorf("MaxAge=-1 must delete, got %q", got)
	}

	if _, err := j.AddForDomain(events.DomainCookies{
		Cookies: []model.Cookie{{Key: "x", Value: "1"}},
	}); err == nil {
		t.Error("missing domain must be an error")
	}
}

func TestToggles(t *testing.T) {
	j := newJar(t)

	j.SetStoreCookies(false)
	if err := j.AddRaw("a=1", mustURL(t, "https://t.test/")); err != nil {
		t.Fatalf("store-off AddRaw must be a nil no-op, got %v", err)
	}
	if err := j.AddRaw("not a cookie", mustURL(t, "https://t.test/")); err != nil {
		t.Fatalf("store-off AddRaw must not even parse, got %v", err)
	}
	ds, _ := j.Domains()
	if len(ds) != 0 {
		t.Fatalf("store-off must leave the jar unchanged, got %+v", ds)
	}
	j.SetStoreCookies(true)
	addRaw(t, j, "a=1", "https://t.test/")

	j.SetSendCookies(false)
	if got := header(t, j, "https://t.test/"); got != "" {
		t.Errorf("send-off HeaderFor = %q, want empty", got)
	}
	ds, _ = j.Domains()
	if len(ds) != 1 {
		t.Errorf("send-off must not hide cookies from the UI, got %+v", ds)
	}
	j.SetSendCookies(true)
	if got := header(t, j, "https://t.test/"); !strings.Contains(got, "a=1") {
		t.Errorf("send back on: got %q", got)
	}
}

func TestLooseParse(t *testing.T) {
	j := newJar(t)
	// The reference parses with tough-cookie { loose: true }: names that are
	// not RFC tokens are accepted.
	if err := j.AddRaw("bad name=1", mustURL(t, "https://l.test/")); err != nil {
		t.Fatalf("loose AddRaw: %v", err)
	}
	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 1 || ds[0].Cookies[0].Key != "bad name" {
		t.Fatalf("loose cookie not stored: %+v", ds)
	}
	// No '=' at all is unrecoverable in both parsers.
	if err := j.AddRaw("noequals", mustURL(t, "https://l.test/")); err == nil {
		t.Fatal("line without '=' must be an error")
	}
}

func TestHeaderOrderRFC6265(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "first=1", "https://o.test/base/page")
	addRaw(t, j, "second=1", "https://o.test/base/page") // same path: creation order
	addRaw(t, j, "long=1; Path=/base/deep", "https://o.test/")

	// RFC 6265 §5.4: longest path first, then oldest first.
	if got, want := header(t, j, "https://o.test/base/deep/x"), "long=1; first=1; second=1"; got != want {
		t.Fatalf("HeaderFor = %q, want %q", got, want)
	}
}

func TestCreatedAtPreservedOnReplace(t *testing.T) {
	j := newJar(t)
	addRaw(t, j, "k=1", "https://c.test/")
	first, err := j.Domains()
	if err != nil || len(first) != 1 || len(first[0].Cookies) != 1 {
		t.Fatalf("Domains = %+v, %v", first, err)
	}
	created := first[0].Cookies[0].CreatedAt
	if created.IsZero() {
		t.Fatal("CreatedAt must be set on store")
	}

	addRaw(t, j, "k=2", "https://c.test/")
	second, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if !second[0].Cookies[0].CreatedAt.Equal(created) {
		t.Errorf("replace must keep CreatedAt: %v != %v", second[0].Cookies[0].CreatedAt, created)
	}
}

// memPersist is an in-memory Persistence for the tests.
type memPersist struct {
	mu      sync.Mutex
	initial []model.Cookie
	loadErr error
	saveErr error
	saved   [][]model.Cookie
}

func (m *memPersist) Save(cookies []model.Cookie) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, append([]model.Cookie(nil), cookies...))
	return nil
}

func (m *memPersist) Load() ([]model.Cookie, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return append([]model.Cookie(nil), m.initial...), nil
}

func (m *memPersist) saveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.saved)
}

func (m *memPersist) lastSave() []model.Cookie {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.saved) == 0 {
		return nil
	}
	return m.saved[len(m.saved)-1]
}

func TestPersistenceLoadFiltersExpired(t *testing.T) {
	p := &memPersist{initial: []model.Cookie{
		{Key: "old", Domain: "example.com", Path: "/", ExpiresAt: time.Now().Add(-time.Hour)},
		{Key: "live", Domain: "example.com", Path: "/"},
	}}
	j := newJar(t, WithPersistence(p, 0))

	ds, err := j.Domains()
	if err != nil {
		t.Fatalf("Domains: %v", err)
	}
	if len(ds) != 1 || len(ds[0].Cookies) != 1 || ds[0].Cookies[0].Key != "live" {
		t.Fatalf("load must filter expired cookies, got %+v", ds)
	}
	if got := header(t, j, "https://example.com/"); !strings.Contains(got, "live") {
		t.Fatalf("restored cookie missing from header: %q", got)
	}
	if p.saveCount() != 0 {
		t.Fatalf("load must not trigger a save, got %d", p.saveCount())
	}
}

func TestPersistenceSynchronousSave(t *testing.T) {
	p := &memPersist{}
	j := newJar(t, WithPersistence(p, 0)) // debounce <= 0: save inline
	addRaw(t, j, "a=1", "https://example.com/")

	if p.saveCount() != 1 {
		t.Fatalf("synchronous saves = %d, want 1", p.saveCount())
	}
	last := p.lastSave()
	if len(last) != 1 || last[0].Key != "a" || last[0].Domain != "example.com" {
		t.Fatalf("snapshot = %+v, want the stored cookie", last)
	}
	// A session cookie round-trips with a zero ExpiresAt.
	if !last[0].ExpiresAt.IsZero() || last[0].MaxAge != 0 {
		t.Errorf("session cookie persisted with expiry %+v maxAge %d", last[0].ExpiresAt, last[0].MaxAge)
	}
}

func TestPersistenceCloseFlushesPendingSave(t *testing.T) {
	p := &memPersist{}
	j, err := New(WithPersistence(p, time.Hour)) // long debounce: nothing inline
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	addRaw(t, j, "a=1", "https://example.com/")
	if p.saveCount() != 0 {
		t.Fatalf("debounced save fired early: %d", p.saveCount())
	}
	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if p.saveCount() != 1 {
		t.Fatalf("Close must flush, saves = %d", p.saveCount())
	}
}

func TestPersistenceExplicitPersist(t *testing.T) {
	p := &memPersist{}
	j := newJar(t, WithPersistence(p, time.Hour))
	addRaw(t, j, "a=1", "https://example.com/")
	if err := j.Persist(); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if p.saveCount() != 1 {
		t.Fatalf("Persist must save, saves = %d", p.saveCount())
	}
	// Nothing pending: Persist is a no-op afterwards.
	if err := j.Persist(); err != nil {
		t.Fatalf("second Persist: %v", err)
	}
	if p.saveCount() != 1 {
		t.Fatalf("idle Persist must not save, saves = %d", p.saveCount())
	}
}

func TestPersistenceLoadError(t *testing.T) {
	p := &memPersist{loadErr: errors.New("corrupt")}
	if _, err := New(WithPersistence(p, 0)); err == nil {
		t.Fatal("a failing Load must fail construction")
	}
}

func TestNilURLIsAnError(t *testing.T) {
	j := newJar(t)
	if err := j.AddRaw("a=1", nil); err == nil {
		t.Error("AddRaw(nil) must error")
	}
	if _, err := j.CookiesFor(nil); err == nil {
		t.Error("CookiesFor(nil) must error")
	}
	if _, err := j.HeaderFor(nil); err == nil {
		t.Error("HeaderFor(nil) must error")
	}
	if _, err := j.MergeHeaderFor(nil, "x"); err == nil {
		t.Error("MergeHeaderFor(nil) must error")
	}
}

func TestConcurrentAccess(t *testing.T) {
	j := newJar(t)
	u := mustURL(t, "https://example.com/dir/page")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("c%d", i)
			_ = j.AddRaw(name+"=1", u)
			_, _ = j.CookiesFor(u)
			_, _ = j.HeaderFor(u)
			_, _ = j.Domains()
			_, _ = j.MergeHeaderFor(u, "x=1")
			_ = j.Delete("example.com", "/dir", name)
		}(i)
	}
	wg.Wait()
}
