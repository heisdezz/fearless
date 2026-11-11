package cookies

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// AddRaw parses one Set-Cookie line as received from u's response and stores
// it (§2, addCookieToJar).
//
// Errors: a parse failure, a cookie the underlying jar refuses (public-suffix
// domain, host/domain mismatch, malformed domain attribute) and a
// non-compliant __Host- cookie (§4) all return an error and store nothing.
// Everything else is a no-op returning nil: disabling the store toggle (§7),
// a URL outside the HTTP family (http, https, ws, wss) and a URL without a
// host — there is simply nowhere for a cookie to live.
func (j *Jar) AddRaw(setCookieHeader string, u *url.URL) error {
	if u == nil {
		return errors.New("cookies: nil request URL")
	}
	mapped, ok := mapScheme(u.Scheme)
	if !ok {
		return nil
	}
	host := indexHost(u)
	if host == "" {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.store {
		return nil // §7: storeCookies = false leaves the jar unchanged
	}
	hc, err := parseSetCookie(setCookieHeader)
	if err != nil {
		return err
	}
	path := hc.Path
	if path == "" || path[0] != '/' {
		path = defaultPath(u.Path)
	}
	hc.Path = path
	if strings.HasPrefix(hc.Name, "__Host-") && (!hc.Secure || path != "/" || hc.Domain != "") {
		return fmt.Errorf("%w: __Host- cookie %q requires Secure, Path=/ and no Domain attribute",
			ErrRejected, hc.Name)
	}

	// Resolve the effective domain and validate the request host against a
	// Domain attribute exactly as the jar's domainAndType does, so the index
	// can never hold a cookie the jar would not.
	var (
		effDomain string
		hostOnly  bool
		storeURL  *url.URL
	)
	switch {
	case hc.Domain == "":
		effDomain, hostOnly = host, true
		storeURL = &url.URL{Scheme: mapped, Host: effDomain, Path: "/"}
	case isIPHost(host):
		// The jar compares IP hosts against the raw attribute and stores
		// accepted ones as host-only (domainAndType).
		if hc.Domain != host {
			return fmt.Errorf("%w: cookie %q domain %q does not match host %q",
				ErrRejected, hc.Name, hc.Domain, host)
		}
		effDomain, hostOnly = host, true
		storeURL = &url.URL{Scheme: mapped, Host: effDomain, Path: "/"}
	default:
		effDomain = normDomain(hc.Domain)
		if effDomain != host && !hasDotSuffix(host, effDomain) {
			return fmt.Errorf("%w: cookie %q domain %q does not match host %q",
				ErrRejected, hc.Name, effDomain, host)
		}
		// A Domain attribute must be validated against the *request* host,
		// so the jar gets the original URL to check against.
		storeURL = &url.URL{Scheme: mapped, Host: u.Host, Path: "/"}
	}

	now := time.Now()
	exp, remove := wireExpiry(hc.MaxAge, hc.Expires, now)
	key := indexKey(effDomain, path, hc.Name)

	if remove {
		// An expired line deletes: the jar gets it as sent (MaxAge < 0 or a
		// past Expires) and derives the same identity the entry was stored
		// under; the index loses its key.
		j.jar.SetCookies(storeURL, []*http.Cookie{hc})
		if _, ok := j.index[key]; ok {
			delete(j.index, key)
			j.scheduleSaveLocked()
		}
		return nil
	}

	// Store with the absolute instant so jar and index agree on expiry.
	stored := *hc
	stored.MaxAge = 0
	stored.Expires = exp
	j.jar.SetCookies(storeURL, []*http.Cookie{&stored})

	probe := &url.URL{Scheme: "https", Host: effDomain, Path: path}
	if !j.jarContains(probe, hc.Name, hc.Value) {
		return fmt.Errorf("%w: cookie %q (domain %q path %q)",
			ErrRejected, hc.Name, effDomain, path)
	}
	mc := model.Cookie{
		Key:       hc.Name,
		Value:     hc.Value,
		Domain:    effDomain,
		Path:      path,
		ExpiresAt: exp,
		MaxAge:    hc.MaxAge,
		Secure:    hc.Secure,
		HTTPOnly:  hc.HttpOnly,
		SameSite:  sameSiteString(hc.SameSite),
		HostOnly:  hostOnly,
		CreatedAt: now,
	}
	if old, ok := j.index[key]; ok && !old.CreatedAt.IsZero() {
		mc.CreatedAt = old.CreatedAt // the jar preserves creation on replace
	}
	j.index[key] = mc
	j.scheduleSaveLocked()
	return nil
}

// AddForDomain stores cookies edited in the cookies manager (§10). The
// group's domain is the fallback for cookies that do not carry one; cookies
// the jar refuses or that break the __Host- rules of §4 are skipped and not
// counted. The number of stored cookies is returned.
func (j *Jar) AddForDomain(d events.DomainCookies) (int, error) {
	group := normDomain(d.Domain)
	if len(d.Cookies) > 0 && group == "" {
		usable := false
		for _, c := range d.Cookies {
			if normDomain(c.Domain) != "" {
				usable = true
				break
			}
		}
		if !usable {
			return 0, errors.New("cookies: AddForDomain: domain required")
		}
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.store {
		return 0, nil // §7
	}
	now := time.Now()
	added := 0
	changed := false
	for _, c := range d.Cookies {
		if c.Key == "" {
			continue
		}
		domain := normDomain(c.Domain)
		if domain == "" {
			domain = group
		}
		if domain == "" {
			continue
		}
		path := c.Path
		if path == "" || path[0] != '/' {
			path = "/"
		}
		if strings.HasPrefix(c.Key, "__Host-") && (!c.Secure || path != "/" || !c.HostOnly) {
			continue // §4: Secure, Path=/ and host-only (no Domain attribute)
		}
		c.Domain, c.Path = domain, path

		exp, remove := modelExpiry(c, now)
		key := indexKey(domain, path, c.Key)
		if remove {
			j.removeFromJar(domain, path, c.Key)
			if _, ok := j.index[key]; ok {
				delete(j.index, key)
				changed = true
			}
			continue
		}
		if !exp.IsZero() {
			c.ExpiresAt = exp
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = now
		}
		if !j.acceptJar(c) {
			continue // the jar refused it (public suffix, malformed domain…)
		}
		if old, ok := j.index[key]; ok && !old.CreatedAt.IsZero() {
			c.CreatedAt = old.CreatedAt
		}
		j.index[key] = c
		added++
		changed = true
	}
	if changed {
		j.scheduleSaveLocked()
	}
	return added, nil
}

// Delete removes one cookie by domain, path and name (§10). Deleting a
// cookie that is not there is not an error: the operation is idempotent, as
// in the reference implementation.
func (j *Jar) Delete(domain, path, key string) error {
	d := normDomain(domain)
	if d == "" {
		return errors.New("cookies: Delete: empty domain")
	}
	if key == "" {
		return errors.New("cookies: Delete: empty cookie name")
	}
	if path == "" {
		path = "/"
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	k := indexKey(d, path, key)
	if _, ok := j.index[k]; !ok {
		return nil
	}
	delete(j.index, k)
	j.removeFromJar(d, path, key)
	j.scheduleSaveLocked()
	return nil
}

// DeleteForDomain removes every cookie of one domain (§10).
func (j *Jar) DeleteForDomain(domain string) error {
	d := normDomain(domain)
	if d == "" {
		return errors.New("cookies: DeleteForDomain: empty domain")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	changed := false
	for k, c := range j.index {
		if c.Domain != d {
			continue
		}
		delete(j.index, k)
		j.removeFromJar(c.Domain, c.Path, c.Key)
		changed = true
	}
	if changed {
		j.scheduleSaveLocked()
	}
	return nil
}

// acceptJar writes c into the validation jar using a URL built from the
// cookie's own domain and path, and reports whether the jar kept it. Used
// where there is no request URL to validate against (reload, UI edits).
func (j *Jar) acceptJar(c model.Cookie) bool {
	u := &url.URL{Scheme: "https", Host: c.Domain, Path: c.Path}
	hc := &http.Cookie{
		Name:     c.Key,
		Value:    c.Value,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
		SameSite: sameSiteMode(c.SameSite),
	}
	if !c.HostOnly {
		hc.Domain = c.Domain
	}
	if exp, _ := modelExpiry(c, time.Now()); !exp.IsZero() {
		hc.Expires = exp
	}
	j.jar.SetCookies(u, []*http.Cookie{hc})
	return j.jarContains(u, c.Key, c.Value)
}

// removeFromJar asks the jar to drop an entry. The URL is built from the
// cookie's own domain and path so the jar derives the same identity
// (domain, path, name) the entry was stored under.
func (j *Jar) removeFromJar(domain, path, key string) {
	u := &url.URL{Scheme: "https", Host: domain, Path: path}
	j.jar.SetCookies(u, []*http.Cookie{{Name: key, Path: path, MaxAge: -1}})
}

// jarContains reports whether the jar holds name=value for u. The jar's
// Cookies method returns only name and value, which is exactly what the
// write-time probe needs.
func (j *Jar) jarContains(u *url.URL, name, value string) bool {
	for _, got := range j.jar.Cookies(u) {
		if got.Name == name && got.Value == value {
			return true
		}
	}
	return false
}

// parseSetCookie parses one Set-Cookie line, preferring the strict standard
// parser and falling back to a permissive parser for the irregular lines the
// reference implementation accepts (Bruno parses with tough-cookie's
// { loose: true }).
func parseSetCookie(line string) (*http.Cookie, error) {
	if c, err := http.ParseSetCookie(line); err == nil {
		return c, nil
	}
	return looseParseSetCookie(line)
}

// looseParseSetCookie is the { loose: true } approximation: the first
// name=value pair is taken as the cookie (names that are not RFC tokens are
// accepted) and the remaining segments are scanned for known attributes,
// skipping anything malformed instead of failing the whole line.
func looseParseSetCookie(line string) (*http.Cookie, error) {
	parts := strings.Split(line, ";")
	name, value, ok := strings.Cut(strings.TrimSpace(parts[0]), "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return nil, fmt.Errorf("cookies: malformed Set-Cookie line %q", line)
	}
	c := &http.Cookie{
		Name:  name,
		Value: strings.TrimSpace(value),
		Raw:   line,
	}
	for _, seg := range parts[1:] {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		attr, val, _ := strings.Cut(seg, "=")
		val = strings.TrimSpace(val)
		switch strings.ToLower(strings.TrimSpace(attr)) {
		case "secure":
			c.Secure = true
		case "httponly":
			c.HttpOnly = true
		case "domain":
			c.Domain = val
		case "path":
			c.Path = val
		case "max-age":
			if secs, err := strconv.Atoi(val); err == nil {
				if secs <= 0 {
					secs = -1
				}
				c.MaxAge = secs
			}
		case "expires":
			if t, err := time.Parse(time.RFC1123, val); err == nil {
				c.Expires = t.UTC()
			} else if t, err := time.Parse("Mon, 02-Jan-2006 15:04:05 MST", val); err == nil {
				c.Expires = t.UTC()
			}
		case "samesite":
			switch strings.ToLower(val) {
			case "lax":
				c.SameSite = http.SameSiteLaxMode
			case "strict":
				c.SameSite = http.SameSiteStrictMode
			case "none":
				c.SameSite = http.SameSiteNoneMode
			}
		}
	}
	return c, nil
}

// isIPHost reports whether host is an IP literal, like the jar's isIP.
func isIPHost(host string) bool {
	_, err := netip.ParseAddr(host)
	return err == nil
}
