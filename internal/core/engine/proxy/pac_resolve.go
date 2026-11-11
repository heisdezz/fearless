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

// ErrPACUnsupported means the PAC document answered with a directive this client
// cannot express as a proxy URL.
//
// PAC's return value is a *ranked list of attempts* — "PROXY a:1; PROXY b:2;
// DIRECT" means "try a, then b, then go direct". Resolve returns only the first
// attempt, because a url.URL is one choice; ResolveAttempts returns the whole
// chain, and the transport falls through to the next entry when dialing one
// fails (B-020).
var ErrPACUnsupported = errors.New("proxy: PAC returned a directive this client cannot use")

// resolveFromPAC evaluates the configured PAC document and returns every attempt
// it named, in PAC's own rank order.
//
// Bypass entries in the settings are applied before the script: a host the user
// listed must skip the proxy whether or not the PAC file agrees, because the
// bypass list is an explicit per-app instruction and the PAC file is a
// network-level policy.
func resolveFromPAC(input ResolutionInput) ([]*url.URL, error) {
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
	return AttemptsOf(result)
}

// AttemptsOf converts a PACResult into dial attempts, preserving order.
//
// A DIRECT directive becomes a nil *url.URL so callers can tell "connect
// directly" apart from "there is no attempt at all" (an empty slice). An
// unsupported scheme is an error rather than a silently skipped attempt: a PAC
// file asking for something this client cannot do should be visible.
func AttemptsOf(result PACResult) ([]*url.URL, error) {
	if len(result.Attempts) == 0 {
		return nil, ErrPACEmptyResult
	}
	attempts := make([]*url.URL, 0, len(result.Attempts))
	for _, d := range result.Attempts {
		switch d.Kind {
		case PACDirect:
			attempts = append(attempts, nil)
		case PACHTTP, PACHTTPS, PACSOCKS5:
			attempts = append(attempts, d.URL())
		default:
			// PACDirective.URL defaults an unknown kind to http, which would turn a
			// typo in the parser into a bogus HTTP proxy. Reject it here instead.
			return nil, fmt.Errorf("%w: %s", ErrPACUnsupported, d)
		}
	}
	return attempts, nil
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
