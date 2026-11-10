package model

import "time"

// Cookie is one cookie as the UI and events see it.
//
// This is a value type in the model, not a wrapper around net/http's cookie
// type, because it must be serialisable (persisted, shown in the UI, carried by
// events) and because the UI edits cookies that are not yet attached to a jar.
//
// Ownership note: the type lives here because events.CookiesUpdated and the
// settings layer reference it; the jar implementation and its policy belong to
// the COOKIES package (internal/core/cookies).
type Cookie struct {
	// Key is the cookie name. Bruno calls it "key" in its UI and API.
	Key string `json:"key"`
	// Value is the cookie value.
	Value string `json:"value"`
	// Domain is the domain the cookie applies to, without a leading dot.
	Domain string `json:"domain"`
	// Path is the path prefix, "/" by default.
	Path string `json:"path"`
	// ExpiresAt is the expiry instant. Session cookies use the zero value.
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	// MaxAge is the lifetime in seconds as sent; 0 is a session cookie, negative
	// means delete.
	MaxAge int `json:"maxAge,omitempty"`
	// Secure restricts the cookie to HTTPS origins.
	Secure bool `json:"secure"`
	// HTTPOnly hides the cookie from script.
	HTTPOnly bool `json:"httpOnly"`
	// SameSite is "Strict", "Lax" or "None"; "" leaves it unspecified.
	SameSite string `json:"sameSite,omitempty"`
	// HostOnly reports whether the cookie was set without a Domain attribute and
	// therefore applies only to the exact host.
	HostOnly bool `json:"hostOnly,omitempty"`
	// CreatedAt is when the jar first stored it.
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// Session reports whether the cookie expires with the session.
func (c Cookie) Session() bool { return c.ExpiresAt.IsZero() && c.MaxAge == 0 }

// ExpiredAt reports whether the cookie has expired by the given instant.
func (c Cookie) ExpiredAt(now time.Time) bool {
	if c.MaxAge < 0 {
		return true
	}
	if c.ExpiresAt.IsZero() {
		return false
	}
	return !c.ExpiresAt.After(now)
}
