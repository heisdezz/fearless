package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fearless/internal/core/cookies"
)

// TestCookiesSentAndStored proves the jar is actually wired: a Set-Cookie from
// the first request must be sent back on the next one.
func TestCookiesSentAndStored(t *testing.T) {
	jar, err := cookies.New()
	if err != nil {
		t.Fatalf("jar: %v", err)
	}
	defer jar.Close()

	var secondRequestCookie string
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		secondRequestCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	o := opts()
	o.Jar = jar

	// First request stores the cookie.
	if res, _ := Execute(context.Background(), get(t, server.URL+"/set"), o); res.Err != "" {
		t.Fatalf("first request: %s", res.Err)
	}

	// Second request must carry it.
	if res, _ := Execute(context.Background(), get(t, server.URL+"/read"), o); res.Err != "" {
		t.Fatalf("second request: %s", res.Err)
	}
	if secondRequestCookie == "" {
		t.Fatal("the jar's cookie was not sent on the second request")
	}
	if !contains(secondRequestCookie, "session=abc") {
		t.Errorf("Cookie header = %q, want it to include session=abc", secondRequestCookie)
	}
}

// TestCookiesFollowRedirect checks that a cookie set on a redirecting hop is sent
// to the target, which is the case a naive implementation misses.
func TestCookiesFollowRedirect(t *testing.T) {
	jar, err := cookies.New()
	if err != nil {
		t.Fatalf("jar: %v", err)
	}
	defer jar.Close()

	var targetCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.SetCookie(w, &http.Cookie{Name: "hop", Value: "1", Path: "/"})
			w.Header().Set("Location", "/target")
			w.WriteHeader(http.StatusFound)
			return
		}
		targetCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	o := opts()
	o.Jar = jar

	res, _ := Execute(context.Background(), get(t, server.URL+"/start"), o)
	if res.Err != "" {
		t.Fatalf("error: %s", res.Err)
	}
	if !contains(targetCookie, "hop=1") {
		t.Errorf("target Cookie = %q, want the redirect hop's cookie", targetCookie)
	}
}

func TestMergeCookieHeader(t *testing.T) {
	cases := []struct {
		explicit, fromJar, want string
	}{
		{"", "a=1", "a=1"},
		{"a=1", "", "a=1"},
		{"a=1", "a=2", "a=1"},                // explicit wins
		{"a=1", "b=2", "a=1; b=2"},           // disjoint merge
		{"a=1; b=2", "a=2; b=3", "a=1; b=2"}, // all explicit
	}
	for _, tc := range cases {
		if got := mergeCookieHeader(tc.explicit, tc.fromJar); got != tc.want {
			t.Errorf("merge(%q,%q) = %q, want %q", tc.explicit, tc.fromJar, got, tc.want)
		}
	}
}

// contains is a small helper so the test does not depend on strings for one call.
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

// indexOf returns the first index of needle in haystack, or -1.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
