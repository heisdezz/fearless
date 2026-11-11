package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// PACSource supplies a FindProxyForURL evaluation for one target.
//
// It is an interface so a caller can hand in a cached resolver (the production
// path) or a literal document in a test without either knowing the other
// exists.
type PACSource interface {
	Evaluate(ctx context.Context, target *url.URL) (PACResult, error)
}

// ErrPACUnsupported means the PAC document answered with something this layer
// cannot express as a single proxy URL.
//
// PAC's return value is a *ranked list of attempts* — "PROXY a:1; PROXY b:2;
// DIRECT" means "try a, then b, then go direct". A url.URL is one choice, and
// http.Transport's Proxy func is free to return http.ErrUseLastResponse-style
// sentinels but cannot say "and if this one fails, try the next". So the first
// proxy attempt is returned and the rest are dropped; when the first attempt is
// DIRECT, the caller gets a direct connection.
//
// That is a real behavioural gap against a full PAC client, and it is recorded
// in docs/progress/blockers.md (B-020) rather than papered over. Closing it
// needs a proxy selector that can retry across attempts, which is an
// ENGINE/TRANSPORT concern, not a resolver one.
var ErrPACUnsupported = errors.New("proxy: PAC returned a fallback list; only the first attempt is used")

// resolveFromPAC evaluates the configured PAC document and converts the first
// attempt into a proxy URL.
//
// Bypass entries in the settings are applied before the script: a host the user
// listed must skip the proxy whether or not the PAC file agrees, because the
// bypass list is an explicit per-app instruction and the PAC file is a
// network-level policy.
func resolveFromPAC(input ResolutionInput) (*url.URL, error) {
	settings := input.AppSettings
	if settings == nil {
		return nil, nil
	}
	if MatchBypass(settings.Bypass, input.TargetURL) {
		return nil, nil
	}

	if input.PACResolver == nil {
		// A PAC mode with nothing to evaluate is a configuration error. Say so
		// rather than quietly going direct, which would hide a broken setup.
		return nil, ErrPACNoSource
	}

	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := input.PACResolver.Evaluate(ctx, input.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("pac: %w", err)
	}
	if len(result.Attempts) == 0 {
		return nil, ErrPACEmptyResult
	}

	first := result.Attempts[0]
	if first.Kind == PACDirect {
		return nil, nil
	}
	u := first.URL()
	if u == nil {
		return nil, nil
	}
	switch u.Scheme {
	case "http", "https", "socks5":
		return u, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrPACUnsupported, first)
	}
}

// ErrPACNoSource means PAC mode was selected but no resolver was supplied.
var ErrPACNoSource = errors.New("proxy: PAC mode is on but no PAC resolver is configured")

// NewPACSource wraps a PACResolver (or any other implementation) as a
// PACSource. It exists so the common case reads as a one-liner.
func NewPACSource(r *PACResolver) PACSource { return pacSource{r} }

// pacSource adapts PACResolver, which needs the evaluator, to PACSource, which
// does not.
type pacSource struct{ r *PACResolver }

// Evaluate runs the cached document, with the default evaluator.
func (s pacSource) Evaluate(ctx context.Context, target *url.URL) (PACResult, error) {
	if s.r == nil {
		return PACResult{}, ErrPACNoSource
	}
	return s.r.Evaluate(ctx, target, PACEvaluator{})
}
