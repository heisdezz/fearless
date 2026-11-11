package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"fearless/internal/core/engine/auth"
	"fearless/internal/core/engine/build"
	enginetls "fearless/internal/core/engine/tls"
	"fearless/internal/core/engine/transport"
	"fearless/internal/core/model"
	"fearless/internal/core/template"
	"fearless/internal/core/vars"
)

// runPipeline executes the full 11-phase request lifecycle documented in
// flows/request-lifecycle.md.
//
// Invariants:
// 1. Exactly one of (*model.Result, nil) or (nil, error) is returned.
// 2. Phase order strictly conforms to flows/request-lifecycle.md.
// 3. Unresolved variables are reported to sink.
func runPipeline(ctx context.Context, in Input) (*model.Result, error) {
	// 0. Pre-flight input validation
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ensureOne(nil, err)
	}
	if in.Item == nil || in.Item.Request == nil {
		return ensureOne(nil, errors.New("engine: missing item or request"))
	}
	if in.Sink == nil {
		in.Sink = NopSink{}
	}

	// 1. Pre-Request Variables (vars:pre-request declarative block)
	if in.Item.Request.Vars != nil && len(in.Item.Request.Vars.PreRequest) > 0 {
		tplRes := template.PreRequest(in.Item.Request.Vars.PreRequest, in.Scopes, vars.InterpolateOptions{})
		if in.Scopes.Request == nil {
			in.Scopes.Request = make(map[string]any)
		}
		for k, v := range tplRes.AsMap() {
			in.Scopes.Request[k] = v
		}
		if len(tplRes.Unresolved) > 0 {
			in.Sink.Unresolved(tplRes.Unresolved)
		}
	}

	// 2. Pre-Request Scripts / Hooks
	for _, hook := range in.Hooks.PreRequest {
		if hook == nil {
			continue
		}
		if err := hook.RunPreRequest(ctx, in.Item, &in.Scopes, in.Sink); err != nil {
			return ensureOne(nil, err)
		}
		if err := ctx.Err(); err != nil {
			return ensureOne(nil, err)
		}
	}

	// 3. Request Preparation & Synthesis (merge tree, interpolate vars, render body, calculate headers)
	buildIn := build.BuildInput{
		Item:           in.Item,
		Collection:     in.Collection,
		Variables:      in.Scopes.Merge(),
		CollectionPath: in.CollectionPath,
		Settings:       &in.Settings,
	}
	built, err := build.Build(ctx, buildIn)
	if err != nil {
		return ensureOne(nil, err)
	}
	if len(built.UnresolvedVars) > 0 {
		in.Sink.Unresolved(built.UnresolvedVars)
	}

	// 4. Configure Request & Auth Providers
	chain := auth.ChainFor(in.Collection, in.Ancestors, in.Item)
	authDeps := auth.Deps{
		Interpolate: func(s string) string {
			return vars.Interpolate(s, in.Scopes.Merge(), vars.InterpolateOptions{})
		},
		Scopes:         in.Scopes,
		CollectionPath: in.CollectionPath,
		Now:            func() int64 { return time.Now().Unix() },
		Getenv:         os.Getenv,
		Rewind: func() (*http.Request, error) {
			b, berr := build.Build(ctx, buildIn)
			if berr != nil {
				return nil, berr
			}
			return b.HTTPRequest, nil
		},
	}

	authPrep, err := auth.Apply(ctx, chain, authDeps)
	if err != nil {
		return ensureOne(nil, err)
	}

	for _, h := range authPrep.Headers {
		built.HTTPRequest.Header.Set(h.Name, h.Value)
		built.Headers.Set(h.Name, h.Value)
	}
	if len(authPrep.QueryParams) > 0 {
		q := built.HTTPRequest.URL.Query()
		for _, qp := range authPrep.QueryParams {
			q.Add(qp.Name, qp.Value)
		}
		built.HTTPRequest.URL.RawQuery = q.Encode()
	}
	// A provider may replace the payload, as OAuth1 does for body placement where
	// the signed parameters become the form body.
	if authPrep.Body != nil {
		built.Body = authPrep.Body
		built.ContentLength = int64(len(authPrep.Body))
		built.HTTPRequest.Body = io.NopCloser(bytes.NewReader(authPrep.Body))
		built.HTTPRequest.ContentLength = built.ContentLength
		built.HTTPRequest.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(authPrep.Body)), nil
		}
		built.HTTPRequest.Header.Set("Content-Length", strconv.Itoa(len(authPrep.Body)))
		built.Headers.Set("Content-Length", strconv.Itoa(len(authPrep.Body)))
	}

	// Resolve timeout
	timeoutMs := 0
	if in.Settings.Timeout != nil && !in.Settings.Timeout.Inherit {
		timeoutMs = in.Settings.Timeout.Millis
	} else if in.GlobalPreferences.Request.TimeoutMs > 0 {
		timeoutMs = in.GlobalPreferences.Request.TimeoutMs
	}

	// Resolve redirects
	followRedirects := in.Settings.FollowingRedirects()
	maxRedirects := in.Settings.MaxRedirectCount()
	if !followRedirects {
		maxRedirects = 0
	}

	// Resolve forwarding auth header
	forwardAuth := in.Settings.ForwardsAuthHeader()

	// Resolve User-Agent if not explicitly set
	if built.HTTPRequest.Header.Get("User-Agent") == "" {
		ua := in.UserAgent
		if ua == "" {
			ua = transport.DefaultUserAgent
		}
		built.HTTPRequest.Header.Set("User-Agent", ua)
		built.Headers.Set("User-Agent", ua)
	}

	// 5. TLS Resolution (CA bundles, client certificates, verification precedence)
	tlsOpts := in.TLS
	if tlsOpts.TargetURL == "" && built.HTTPRequest != nil && built.HTTPRequest.URL != nil {
		tlsOpts.TargetURL = built.HTTPRequest.URL.String()
	}
	if tlsOpts.CollectionPath == "" {
		tlsOpts.CollectionPath = in.CollectionPath
	}

	// 5.1 Verification precedence: Request > Collection > Preferences > default true
	var colVerify *bool
	if in.Collection != nil && in.Collection.Config.Security != nil {
		colVerify = in.Collection.Config.Security.VerifyTLS
	}
	var prefVerify *bool
	if in.GlobalPreferences.TLS.VerifyServerCertificates {
		v := true
		prefVerify = &v
	} else if in.GlobalPreferences.Request.TimeoutMs > 0 || in.GlobalPreferences.TLS.CACertFile != "" {
		v := in.GlobalPreferences.TLS.VerifyServerCertificates
		prefVerify = &v
	}

	effectiveVerify := enginetls.ResolveVerifyTLS(tlsOpts.VerifyServerCertificates, colVerify, prefVerify)
	tlsOpts.VerifyServerCertificates = &effectiveVerify

	// 5.2 Client certificates: Collection takes precedence over Global
	if tlsOpts.CollectionCerts == nil && in.Collection != nil && in.Collection.Config.Security != nil && in.Collection.Config.Security.ClientCertificates != nil {
		tlsOpts.CollectionCerts = enginetls.FromModelClientCertificates(in.Collection.Config.Security.ClientCertificates)
	}
	if tlsOpts.GlobalCerts == nil && in.GlobalPreferences.TLS.ClientCertificates != nil {
		tlsOpts.GlobalCerts = enginetls.FromModelClientCertificates(in.GlobalPreferences.TLS.ClientCertificates)
	}

	// 5.3 CA configuration from preferences if not explicitly set
	if tlsOpts.CA.CustomFilePath == "" && in.GlobalPreferences.TLS.CACertFile != "" {
		tlsOpts.CA.CustomFilePath = in.GlobalPreferences.TLS.CACertFile
	}

	tlsRes, err := enginetls.Resolve(tlsOpts)
	if err != nil {
		return ensureOne(nil, err)
	}

	var wrapTransport func(http.RoundTripper) http.RoundTripper
	if authPrep.Transport != nil {
		wrapTransport = authPrep.Transport.Wrap
	}

	// 6. Transmission (emit Queued snapshot, execute via transport)
	sent := sentSnapshot(built)
	in.Sink.Queued(sent)

	transOpts := transport.Options{
		FollowRedirects:    followRedirects,
		MaxRedirects:       maxRedirects,
		TimeoutMs:          timeoutMs,
		ForwardAuthHeader:  forwardAuth,
		Proxy:              in.ProxyURL,
		TLS:                tlsOpts,
		TLSConfig:          tlsRes.TLSConfig,
		WrapTransport:      wrapTransport,
		Jar:                in.Jar,
		Spooler:            in.Spooler,
		Sink:               &transportSinkBridge{sink: in.Sink},
		UserAgent:          in.UserAgent,
		DisableCompression: in.DisableCompression,
	}

	res, execErr := transport.Execute(ctx, built.HTTPRequest, transOpts)
	if execErr != nil {
		return ensureOne(nil, execErr)
	}
	if err := ctx.Err(); err != nil {
		return ensureOne(nil, err)
	}

	// 6. Post-Response Pipeline (only when a response status is received)
	if res != nil && res.Status > 0 {
		// 6.1 Post-Response Variables ($res.body... etc.)
		if in.Item.Request.Vars != nil && len(in.Item.Request.Vars.PostResponse) > 0 {
			tplResp := TemplateResponseFromResult(res)
			postRes := template.PostResponse(in.Item.Request.Vars.PostResponse, tplResp, in.Scopes, vars.InterpolateOptions{})
			if in.Scopes.Request == nil {
				in.Scopes.Request = make(map[string]any)
			}
			for k, v := range postRes.AsMap() {
				in.Scopes.Request[k] = v
			}
			if len(postRes.Unresolved) > 0 {
				in.Sink.Unresolved(postRes.Unresolved)
			}
		}

		// 6.2 Post-Response Scripts / Hooks
		for _, hook := range in.Hooks.PostResponse {
			if hook == nil {
				continue
			}
			if err := hook.RunPostResponse(ctx, in.Item, res, &in.Scopes, in.Sink); err != nil {
				in.Sink.ScriptOutput("error", err.Error())
			}
		}

		// 6.3 Assertions
		for _, hook := range in.Hooks.Assertions {
			if hook == nil {
				continue
			}
			if _, err := hook.RunAssertions(ctx, in.Item, res, &in.Scopes, in.Sink); err != nil {
				in.Sink.ScriptOutput("error", err.Error())
			}
		}

		// 6.4 Tests
		for _, hook := range in.Hooks.Tests {
			if hook == nil {
				continue
			}
			if _, err := hook.RunTests(ctx, in.Item, res, &in.Scopes, in.Sink); err != nil {
				in.Sink.ScriptOutput("error", err.Error())
			}
		}
	}

	return ensureOne(res, nil)
}

// sentSnapshot constructs the SentRequest structure for UI and audit logs.
func sentSnapshot(built *build.BuiltRequest) model.SentRequest {
	sent := model.SentRequest{
		Method: built.Method,
		URL:    built.HTTPRequest.URL.String(),
		Body:   built.Body,
	}

	// Preserve exact header entries
	seen := make(map[string]bool)
	for _, entry := range built.HeaderEntries {
		n, val := entry.Name, entry.Value
		sent.Headers = append(sent.Headers, model.KeyValue{
			Name:    &n,
			Value:   &val,
			Enabled: true,
		})
		seen[n] = true
	}

	// Add any extra headers from built.HTTPRequest that were not in HeaderEntries (e.g. auth)
	for name, values := range built.HTTPRequest.Header {
		if !seen[name] {
			for _, v := range values {
				n, val := name, v
				sent.Headers = append(sent.Headers, model.KeyValue{
					Name:    &n,
					Value:   &val,
					Enabled: true,
				})
			}
		}
	}

	return sent
}

// ensureOne guarantees that exactly one of (*model.Result, error) is returned.
func ensureOne(res *model.Result, err error) (*model.Result, error) {
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("engine: internal error: nil result")
	}
	return res, nil
}
