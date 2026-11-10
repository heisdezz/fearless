package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fearless/internal/core/model"
)

// TestDigestProviderSuppliesWorkingTransport is the AUTH-DIGEST acceptance test
// end to end: Apply must return a transport that authenticates against a
// challenge server. A provider that only recorded config would fail here.
func TestDigestProviderSuppliesWorkingTransport(t *testing.T) {
	const realm = "restricted"
	const nonce = "7d5b1c3e9f0a2b4c6d8e0f1a3b5c7d9e"
	const user = "alice"
	const pass = "s3cret"

	var unauthenticatedSeen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			unauthenticatedSeen++
			w.Header().Set("WWW-Authenticate",
				fmt.Sprintf(`Digest realm=%q, nonce=%q, qop="auth"`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		params := parseAuthParams(strings.TrimPrefix(authz, "Digest "))
		ha1 := md5Hex(user + ":" + realm + ":" + pass)
		ha2 := md5Hex(r.Method + ":" + params["uri"])
		want := md5Hex(ha1 + ":" + nonce + ":" + params["nc"] + ":" + params["cnonce"] + ":auth:" + ha2)
		if params["response"] != want {
			t.Errorf("server-side digest mismatch\n got %s\nwant %s", params["response"], want)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "authenticated")
	}))
	defer server.Close()

	cfg := &model.Auth{
		Mode:     model.AuthDigest,
		Username: strp(user),
		Password: strp(pass),
	}

	rewind := func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, server.URL+"/secret", nil)
	}
	prepared, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{Rewind: rewind})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if prepared.Transport == nil {
		t.Fatal("digest must supply a TransportProvider")
	}

	client := &http.Client{Transport: prepared.Transport.Wrap(http.DefaultTransport)}
	req, _ := rewind()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "authenticated" {
		t.Errorf("body = %q", body)
	}
	if unauthenticatedSeen != 1 {
		t.Errorf("unauthenticated attempts = %d, want exactly 1", unauthenticatedSeen)
	}
}

// TestDigestProviderWithoutRewindIsSafe covers the case where the pipeline could
// not supply a replayable body: the 401 must be returned, not a panic.
func TestDigestProviderWithoutRewindIsSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Digest realm="r", nonce="n", qop="auth"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cfg := &model.Auth{Mode: model.AuthDigest, Username: strp("u"), Password: strp("p")}
	prepared, err := Apply(context.Background(), []*model.Auth{cfg}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if prepared.Transport == nil {
		t.Fatal("expected a transport even without rewind")
	}

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := prepared.Transport.Wrap(http.DefaultTransport).RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want the original 401", resp.StatusCode)
	}
}

// TestDigestProviderInterpolatesCredentials checks that a {{var}} password is
// resolved before the digest is computed.
func TestDigestProviderInterpolatesCredentials(t *testing.T) {
	const realm = "r"
	const nonce = "n"
	const resolved = "resolved-pass"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm=%q, nonce=%q, qop="auth"`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		params := parseAuthParams(strings.TrimPrefix(authz, "Digest "))
		ha1 := md5Hex("user:" + realm + ":" + resolved)
		ha2 := md5Hex(r.Method + ":" + params["uri"])
		want := md5Hex(ha1 + ":" + nonce + ":" + params["nc"] + ":" + params["cnonce"] + ":auth:" + ha2)
		if params["response"] != want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &model.Auth{Mode: model.AuthDigest, Username: strp("user"), Password: strp("{{pw}}")}
	deps := Deps{
		Interpolate: func(s string) string {
			if s == "{{pw}}" {
				return resolved
			}
			return s
		},
		Rewind: func() (*http.Request, error) {
			return http.NewRequest(http.MethodGet, server.URL+"/x", nil)
		},
	}
	prepared, err := Apply(context.Background(), []*model.Auth{cfg}, deps)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	client := &http.Client{Transport: prepared.Transport.Wrap(http.DefaultTransport)}
	req, _ := deps.Rewind()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (interpolated password rejected)", resp.StatusCode)
	}
}
