package oauth2

import (
	"strings"
	"testing"
)

// TestPlacementDefaults covers the defaulting rules.
func TestPlacementDefaults(t *testing.T) {
	// An empty placement is the header, which is what the editor starts on.
	p := PlacementFrom("", "", "", "")
	if p.In != PlacementHeader {
		t.Errorf("In = %q", p.In)
	}
	// With no configured prefix and no server token_type, Bearer is the default.
	if p.HeaderPrefix != DefaultHeaderPrefix {
		t.Errorf("prefix = %q", p.HeaderPrefix)
	}
	if p.QueryKey != DefaultQueryKey {
		t.Errorf("query key = %q", p.QueryKey)
	}

	// The server's token_type wins over the default prefix, because a server
	// issuing DPoP or MAC expects that scheme.
	fromServer := PlacementFrom("", "", "", "DPoP")
	if fromServer.HeaderPrefix != "DPoP" {
		t.Errorf("prefix = %q, want the token_type", fromServer.HeaderPrefix)
	}

	// A configured prefix wins over both.
	configured := PlacementFrom("", "Token", "", "Bearer")
	if configured.HeaderPrefix != "Token" {
		t.Errorf("prefix = %q", configured.HeaderPrefix)
	}
}

// TestPlacementApplyHeader covers the default placement.
func TestPlacementApplyHeader(t *testing.T) {
	p := PlacementFrom("", "", "", "")
	header, queryKey := p.Apply("tok-1", "Bearer")

	if header != "Bearer tok-1" {
		t.Errorf("header = %q", header)
	}
	if queryKey != "" {
		t.Errorf("query key = %q, want none", queryKey)
	}
}

// TestPlacementApplyQueryParams covers the query placement.
func TestPlacementApplyQueryParams(t *testing.T) {
	p := PlacementFrom(PlacementQueryParams, "", "my_token", "")
	header, queryKey := p.Apply("tok-1", "Bearer")

	if header != "" {
		t.Errorf("header = %q, want none", header)
	}
	if queryKey != "my_token" {
		t.Errorf("query key = %q", queryKey)
	}

	// Without a configured key the default applies.
	def := PlacementFrom(PlacementQueryParams, "", "", "")
	if _, key := def.Apply("t", ""); key != DefaultQueryKey {
		t.Errorf("default query key = %q", key)
	}
}

// TestPlacementApplyBareToken covers a prefix-less configuration.
func TestPlacementApplyBareToken(t *testing.T) {
	p := Placement{In: PlacementHeader, HeaderPrefix: " "}
	header, _ := p.Apply("tok", "")
	// An all-space prefix must not leave a leading space in the header.
	if strings.HasPrefix(header, " ") {
		t.Errorf("header = %q, has a leading space", header)
	}
	if header != "tok" {
		t.Errorf("header = %q, want the bare token", header)
	}
}

// TestPlacementBodyIsUnsupported makes the gap explicit rather than silent.
func TestPlacementBodyIsUnsupported(t *testing.T) {
	p := PlacementFrom(PlacementBody, "", "", "")
	if p.SupportsBody() {
		t.Error("the body placement is not implemented and must report so")
	}

	header, queryKey := p.Apply("tok", "")
	if header != "" || queryKey != "" {
		t.Errorf("body placement produced header=%q queryKey=%q, want neither", header, queryKey)
	}

	// The other placements do support themselves.
	if !PlacementFrom("", "", "", "").SupportsBody() {
		t.Error("header placement should be supported")
	}
}
