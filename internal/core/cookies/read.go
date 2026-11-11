package cookies

import (
	"errors"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// mapScheme maps a request scheme onto the scheme the jar sees. WSS is a
// trustworthy origin like HTTPS and WS behaves like plain HTTP
// (logic/cookies.md §3); every other scheme — file, ftp, data — has no
// participation in the jar. A file URL is trustworthy too, but it carries
// no host, so no cookie can ever match it.
func mapScheme(s string) (string, bool) {
	switch s {
	case "https", "wss":
		return "https", true
	case "http", "ws":
		return "http", true
	default:
		return "", false
	}
}

// CookiesFor returns the cookies a request to u would carry: expired ones
// filtered (§2), domain/path/secure rules applied (§3), in RFC 6265 §5.4
// order — longest path first, then oldest first. It errors only on a nil
// URL. Origins outside the HTTP family return no cookies.
func (j *Jar) CookiesFor(u *url.URL) ([]model.Cookie, error) {
	if u == nil {
		return nil, errors.New("cookies: nil request URL")
	}
	if _, ok := mapScheme(u.Scheme); !ok {
		return nil, nil
	}
	host := indexHost(u)
	if host == "" {
		return nil, nil
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	https := u.Scheme == "https" || u.Scheme == "wss"

	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.send {
		return nil, nil // §7: sendCookies = false sends nothing
	}
	now := time.Now()
	var out []model.Cookie
	for _, c := range j.index {
		if c.ExpiredAt(now) || !cookieMatches(c, host, path, https) {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, k int) bool {
		return lessRFC6265(out[i], out[k])
	})
	return out, nil
}

// HeaderFor is the Cookie header value for u: "name=value; name2=value2",
// with no leading or trailing separator (§10). It is empty when nothing
// matches or when the send toggle is off (§7).
func (j *Jar) HeaderFor(u *url.URL) (string, error) {
	cs, err := j.CookiesFor(u)
	if err != nil {
		return "", err
	}
	return cookieLine(cs), nil
}

// MergeHeaderFor combines an explicit Cookie header with the jar's cookies
// for u (§6). Jar cookies win per name — the reference implementation
// spreads the jar last ({...parseCookies(existing), ...parseCookies(jar)}) —
// so a jar cookie overrides the same name the user typed, while explicit
// names keep their position and jar-only names are appended. With the send
// toggle off, or with an empty jar, the explicit header passes through
// unchanged.
func (j *Jar) MergeHeaderFor(u *url.URL, explicit string) (string, error) {
	jarLine, err := j.HeaderFor(u)
	if err != nil {
		return "", err
	}
	if jarLine == "" {
		return explicit, nil
	}
	if strings.TrimSpace(explicit) == "" {
		return jarLine, nil
	}
	return mergeCookieLines(explicit, jarLine), nil
}

// Domains groups the stored cookies by domain for the cookie manager, in
// Bruno's shape: sorted domains, expired cookies filtered out, domains left
// without a live cookie omitted (getDomainsWithCookies).
func (j *Jar) Domains() ([]events.DomainCookies, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	byDomain := make(map[string][]model.Cookie)
	for _, c := range j.index {
		if c.Domain == "" || c.ExpiredAt(now) {
			continue
		}
		byDomain[c.Domain] = append(byDomain[c.Domain], c)
	}
	domains := make([]string, 0, len(byDomain))
	for d := range byDomain {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	out := make([]events.DomainCookies, 0, len(domains))
	for _, d := range domains {
		cs := byDomain[d]
		sort.Slice(cs, func(i, k int) bool {
			if cs[i].Key != cs[k].Key {
				return cs[i].Key < cs[k].Key
			}
			return cs[i].Path < cs[k].Path
		})
		out = append(out, events.DomainCookies{Domain: d, Cookies: cs})
	}
	return out, nil
}

// cookieLine renders cookies as a Cookie header value.
func cookieLine(cs []model.Cookie) string {
	if len(cs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range cs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(c.Key)
		b.WriteByte('=')
		b.WriteString(c.Value)
	}
	return b.String()
}

// mergeCookieLines implements §6: jar cookies override an explicit header
// per name. Mirroring the reference's object spread, explicit names keep
// their first position (with the jar's value substituted) and jar-only names
// are appended.
func mergeCookieLines(explicit, jarLine string) string {
	var (
		order  []string
		values = make(map[string]string)
	)
	add := func(line string) {
		for _, seg := range strings.Split(line, ";") {
			name, value, ok := strings.Cut(seg, "=")
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			// The reference keeps a segment without '=' as a name with an
			// empty value (rest.join('=') of an empty rest).
			if ok {
				value = strings.TrimSpace(value)
			} else {
				value = ""
			}
			if _, seen := values[name]; !seen {
				order = append(order, name)
			}
			values[name] = value
		}
	}
	add(explicit)
	add(jarLine)
	var b strings.Builder
	for i, name := range order {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(values[name])
	}
	return b.String()
}

// cookieMatches applies the RFC 6265 domain, path and secure rules to one
// index entry. The rules mirror net/http/cookiejar entry.shouldSend byte for
// byte; the secure check looks at the cookie's own domain, as the jar's
// secureMatch does, so loopback hosts (and *.localhost) can carry Secure
// cookies over plain HTTP (§3).
func cookieMatches(c model.Cookie, host, path string, https bool) bool {
	if c.HostOnly {
		if c.Domain != host {
			return false
		}
	} else if c.Domain != host && !hasDotSuffix(host, c.Domain) {
		return false
	}
	if !pathMatch(c.Path, path) {
		return false
	}
	if c.Secure && !https && !secureDomain(c.Domain) {
		return false
	}
	return true
}

// pathMatch is RFC 6265 §5.1.4, byte for byte from cookiejar.
//
// An empty cookie path cannot occur through the public API — every write path
// normalises it to "/" — but the guard is kept because indexing at
// `cookiePath[len(cookiePath)-1]` would otherwise panic on a caller that skips
// that normalisation. An empty path matches nothing.
func pathMatch(cookiePath, requestPath string) bool {
	if cookiePath == "" {
		return false
	}
	if cookiePath == requestPath {
		return true
	}
	if !strings.HasPrefix(requestPath, cookiePath) {
		return false
	}
	if cookiePath[len(cookiePath)-1] == '/' {
		return true // "/any/" matches "/any/path"
	}
	return requestPath[len(cookiePath)] == '/'
}

// hasDotSuffix reports whether s ends in "."+suffix (cookiejar's version).
func hasDotSuffix(s, suffix string) bool {
	return len(s) > len(suffix) && s[len(s)-len(suffix)-1] == '.' && s[len(s)-len(suffix):] == suffix
}

// secureDomain reports whether a Secure cookie may be sent to this cookie
// domain over plain HTTP: localhost by name (any label ending in
// ".localhost", case-insensitively) or a loopback address — cookiejar's
// secureMatch rule.
func secureDomain(domain string) bool {
	host := strings.TrimSuffix(domain, ".")
	if i := strings.LastIndex(host, "."); i >= 0 {
		host = host[i+1:]
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip, err := netip.ParseAddr(domain); err == nil {
		return ip.IsLoopback()
	}
	return false
}

// lessRFC6265 orders cookies for the wire like cookiejar does: RFC 6265
// §5.4 — longest path first, then earliest creation, with name and path as
// deterministic tie-breaks.
func lessRFC6265(a, b model.Cookie) bool {
	if len(a.Path) != len(b.Path) {
		return len(a.Path) > len(b.Path)
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	if a.Domain != b.Domain {
		return a.Domain < b.Domain
	}
	return a.Key < b.Key
}
