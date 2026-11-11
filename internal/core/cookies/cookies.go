// Package cookies implements the application's single global cookie jar.
//
// There is one Jar per application instance and no per-collection or
// per-tab jar (docs/logic/cookies.md §1). The package has two moving parts:
//
//   - The index — a map keyed by domain, path and name — is the read
//     authority. CookiesFor, HeaderFor, Domains and every mutation serve
//     from it, applying the domain, path and secure rules of RFC 6265
//     exactly as the wrapped jar does.
//   - The wrapped net/http/cookiejar jar is the write-time validation
//     authority. Every store is round-tripped through it: if the jar
//     refuses the cookie (public-suffix domain, host/domain mismatch,
//     malformed domain attribute) the write is rejected with an error and
//     nothing changes. Go's jar cannot back enumeration — its Cookies
//     method returns only name and value — which is why the index exists
//     alongside it.
//
// Persistence is injected through the Persistence interface (§10); this
// package never imports a store. Writes are persisted with a debounce:
// DefaultDebounce matches Bruno's CookiesStore DEBOUNCE_MS = 5000 (§5).
//
// The toggles of §7 default to Bruno's defaults: SendCookies and
// StoreCookies are both enabled.
package cookies

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"fearless/internal/core/model"
)

// DefaultDebounce is how long a mutation waits before the snapshot is
// handed to the Persistence implementation.
const DefaultDebounce = 5 * time.Second

// Persistence is the injected store half of the jar (logic/cookies.md §10).
// PERSIST implements it; this package only ever sees the interface.
type Persistence interface {
	// Save replaces the persisted snapshot with cookies.
	Save(cookies []model.Cookie) error
	// Load returns the persisted cookies. The jar, not the store, filters
	// expired ones (§7).
	Load() ([]model.Cookie, error)
}

// ErrRejected reports that a cookie was not stored: the underlying jar
// refused it (public-suffix domain, host/domain mismatch, malformed domain
// attribute) or it breaks the __Host- rules of §4.
var ErrRejected = errors.New("cookies: rejected")

// Option configures a Jar at construction time.
type Option func(*Jar)

// WithPersistence injects the store and sets the write debounce. A debounce
// of zero or less saves synchronously after every mutation, which is what
// the tests use; production code leaves it at DefaultDebounce.
func WithPersistence(p Persistence, debounce time.Duration) Option {
	return func(j *Jar) {
		j.persist = p
		j.debounce = debounce
	}
}

// Jar is the single global cookie jar. All methods are safe for concurrent
// use.
type Jar struct {
	mu       sync.Mutex
	jar      *cookiejar.Jar
	index    map[string]model.Cookie
	send     bool
	store    bool
	persist  Persistence
	debounce time.Duration
	timer    *time.Timer
	dirty    bool
	saveErr  error
	closed   bool
}

// New builds the jar. With WithPersistence it loads the persisted cookies
// immediately; cookies that have expired by timestamp are dropped on load
// (§7). A cookie the jar refuses on reload is skipped rather than failing
// startup.
func New(opts ...Option) (*Jar, error) {
	jj, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("cookies: create jar: %w", err)
	}
	j := &Jar{
		jar:      jj,
		index:    make(map[string]model.Cookie),
		send:     true,
		store:    true,
		debounce: DefaultDebounce,
	}
	for _, opt := range opts {
		opt(j)
	}
	if j.persist != nil {
		loaded, err := j.persist.Load()
		if err != nil {
			return nil, fmt.Errorf("cookies: load persisted cookies: %w", err)
		}
		j.restore(loaded)
	}
	return j, nil
}

// SetSendCookies toggles whether request injection sees the jar (§7). It
// does not hide stored cookies from the UI: Domains keeps reporting them.
func (j *Jar) SetSendCookies(v bool) {
	j.mu.Lock()
	j.send = v
	j.mu.Unlock()
}

// SetStoreCookies toggles whether Set-Cookie lines are stored (§7). Loading
// from persistence at startup is independent of this setting.
func (j *Jar) SetStoreCookies(v bool) {
	j.mu.Lock()
	j.store = v
	j.mu.Unlock()
}

// Persist flushes a pending snapshot to the store immediately (§10).
func (j *Jar) Persist() error {
	return j.save()
}

// Close stops the debounce timer and flushes a pending snapshot. The jar
// stays usable in memory afterwards; it just stops persisting.
func (j *Jar) Close() error {
	j.mu.Lock()
	if j.timer != nil {
		j.timer.Stop()
		j.timer = nil
	}
	alreadyClosed := j.closed
	j.closed = true
	j.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	return j.save()
}

// SaveErr reports the last save failure, if any. Asynchronous debounce
// saves cannot return their error to the mutating caller, so failures are
// recorded here and retried by the next mutation or by Close.
func (j *Jar) SaveErr() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.saveErr
}

// restore folds persisted cookies into the jar. Expired cookies are
// filtered by timestamp (§7); everything else must also pass the jar's
// validation or it is skipped.
func (j *Jar) restore(loaded []model.Cookie) {
	now := time.Now()
	for _, c := range loaded {
		if c.Key == "" {
			continue
		}
		c.Domain = normDomain(c.Domain)
		if c.Path == "" || c.Path[0] != '/' {
			c.Path = "/"
		}
		if c.Domain == "" {
			continue
		}
		exp, remove := modelExpiry(c, now)
		if remove {
			continue
		}
		if c.ExpiresAt.IsZero() {
			c.ExpiresAt = exp // MaxAge-derived lifetime, materialised once
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = now
		}
		if !j.acceptJar(c) {
			continue
		}
		j.index[indexKey(c.Domain, c.Path, c.Key)] = c
	}
}

// save hands a snapshot to the Persistence implementation if the index
// changed since the last save. It must not be called with j.mu held.
func (j *Jar) save() error {
	j.mu.Lock()
	if !j.dirty {
		err := j.saveErr
		j.mu.Unlock()
		return err
	}
	snap := j.snapshotLocked()
	p := j.persist
	j.dirty = false
	j.mu.Unlock()
	if p == nil {
		return nil
	}
	err := p.Save(snap)
	j.mu.Lock()
	if err != nil {
		j.dirty = true // retried by the next mutation or Close
		j.saveErr = err
	} else {
		j.saveErr = nil
	}
	j.mu.Unlock()
	return err
}

// scheduleSaveLocked records that the index changed and (re)arms the
// debounce timer. It must be called with j.mu held; in synchronous mode it
// drops the lock around the save so the store may call back into the jar.
func (j *Jar) scheduleSaveLocked() {
	if j.persist == nil || j.closed {
		return
	}
	j.dirty = true
	if j.debounce <= 0 {
		j.mu.Unlock()
		_ = j.save()
		j.mu.Lock()
		return
	}
	if j.timer != nil {
		j.timer.Stop()
	}
	j.timer = time.AfterFunc(j.debounce, func() { _ = j.save() })
}

func (j *Jar) snapshotLocked() []model.Cookie {
	out := make([]model.Cookie, 0, len(j.index))
	for _, c := range j.index {
		out = append(out, c)
	}
	sort.Slice(out, func(i, k int) bool {
		if out[i].Domain != out[k].Domain {
			return out[i].Domain < out[k].Domain
		}
		if out[i].Path != out[k].Path {
			return out[i].Path < out[k].Path
		}
		return out[i].Key < out[k].Key
	})
	return out
}

// indexKey is the identity of a cookie in the index: domain (lowercase, no
// leading dot), path and name, exactly the triple the jar uses for an entry.
func indexKey(domain, path, name string) string {
	return domain + "\x00" + path + "\x00" + name
}

// normDomain canonicalises a domain the way the index stores it: trimmed,
// one leading dot removed (RFC 6265 §5.2.3), lowercased.
func normDomain(domain string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(domain), "."))
}

// indexHost is the request host as the index sees it: no port, no trailing
// dot, lowercased.
func indexHost(u *url.URL) string {
	return strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
}

// defaultPath mirrors net/http/cookiejar's defaultPath: the parent of the
// request path, "/" at the root (RFC 6265 §5.1.4).
func defaultPath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		return "/"
	}
	i := strings.LastIndex(path, "/")
	if i == 0 {
		return "/"
	}
	return path[:i]
}

// wireExpiry mirrors net/http/cookiejar's newEntry for a parsed Set-Cookie:
// MaxAge takes precedence over Expires; MaxAge < 0 or a past Expires means
// the line replaces (deletes) any existing entry.
func wireExpiry(maxAge int, expires time.Time, now time.Time) (time.Time, bool) {
	switch {
	case maxAge < 0:
		return time.Time{}, true
	case maxAge > 0:
		return now.Add(time.Duration(maxAge) * time.Second), false
	case expires.IsZero():
		return time.Time{}, false
	case !expires.After(now):
		return time.Time{}, true
	default:
		return expires, false
	}
}

// modelExpiry derives a model cookie's expiry. MaxAge < 0 means delete; an
// explicit ExpiresAt wins over MaxAge — the jar is given the absolute
// instant so reloads never extend a lifetime — and MaxAge > 0 without an
// ExpiresAt is materialised as now + MaxAge. The boolean reports that the
// cookie must be removed instead of stored.
func modelExpiry(c model.Cookie, now time.Time) (time.Time, bool) {
	if c.MaxAge < 0 {
		return time.Time{}, true
	}
	if !c.ExpiresAt.IsZero() {
		if !c.ExpiresAt.After(now) {
			return time.Time{}, true
		}
		return c.ExpiresAt, false
	}
	if c.MaxAge > 0 {
		return now.Add(time.Duration(c.MaxAge) * time.Second), false
	}
	return time.Time{}, false
}

// sameSiteString maps an http.SameSite onto model.Cookie's representation.
func sameSiteString(m http.SameSite) string {
	switch m {
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return ""
	}
}

// sameSiteMode maps model.Cookie's representation back onto http.SameSite.
// Unknown values are treated as unspecified, like the strict parser does.
func sameSiteMode(s string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}
