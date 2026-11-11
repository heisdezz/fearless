package transport

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"fearless/internal/core/engine/proxy"
	"fearless/internal/core/model"
)

// errText renders an error for a timeline detail.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// buildTLSConfig assembles the TLS configuration for a request.
//
// This package applies only the two verification switches it owns. Loading a CA
// bundle or a client certificate is the TLS package's job: the ENGINE builds the
// full *tls.Config through it and passes the result in Options, so the two
// concerns do not duplicate each other.
func buildTLSConfig(opts Options) *tls.Config {
	if opts.TLSConfig != nil {
		return opts.TLSConfig.Clone()
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if opts.TLS.VerifyServerCertificates != nil && !*opts.TLS.VerifyServerCertificates {
		cfg.InsecureSkipVerify = true
	}
	if opts.TLS.InsecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	return cfg
}

// tlsState mirrors crypto/tls.ConnectionState for the trace callback, so hop.go
// does not need to import crypto/tls.
type tlsState = tls.ConnectionState

// newClient builds the http.Client for one execution.
func newClient(opts Options, tl *timeline) (*http.Client, error) {
	tr, err := newTransport(opts, tl)
	if err != nil {
		return nil, err
	}

	var rt http.RoundTripper = tr
	if opts.WrapTransport != nil {
		rt = opts.WrapTransport(rt)
	}

	client := &http.Client{
		// WrapTransport, when supplied, wraps the round tripper once for the whole
		// client: that is the seam Digest and NTLM use for their handshakes (B-006).
		Transport: rt,
		// Redirects are followed by Execute, not by net/http: each hop must be
		// recorded, re-authenticated and given the jar's cookies for the new host.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if opts.TimeoutMs > 0 {
		client.Timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
	}
	return client, nil
}

// newTransport builds the *http.Transport for one execution.
func newTransport(opts Options, _ *timeline) (*http.Transport, error) {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    opts.DisableCompression,
		TLSClientConfig:       buildTLSConfig(opts),
	}

	// The caller has already resolved which proxy to use. ProxyAttempts carries the
	// full PAC chain in rank order, so a caller that supplies only the chain still
	// gets its first attempt configured here.
	proxyURL := opts.Proxy
	if proxyURL == nil && len(opts.ProxyAttempts) > 0 {
		proxyURL = opts.ProxyAttempts[0]
	}
	if proxyURL != nil {
		if err := proxy.ConfigureTransport(tr, proxyURL); err != nil {
			return nil, err
		}
	}
	return tr, nil
}

// newTrace builds the httptrace client trace for one hop.
//
// Concurrency note: httptrace callbacks run on the connection's own goroutines
// while client.Do is in flight, so they cannot write to the timeline directly.
// Each callback sends to a channel, which the hop drains once Do returns; that
// serialises the writes and preserves their order.
// traceBuffer sizes the trace channel.
//
// It must hold every event one hop can emit, because the hop drains the channel
// only after the request completes: DNS, connect and TLS in begin/end pairs, then
// the write and the first-byte pair. At 16, a hop that resolved DNS and opened a
// fresh connection filled the buffer before the first-byte event, which was then
// dropped — leaving `waiting` absent rather than merely duplicated.
const traceBuffer = 32

func newTrace(tl *timeline) *httptrace.ClientTrace {
	rec := tl.trace

	send := func(ev traceEvent) {
		select {
		case rec <- ev:
		default:
			// Dropping a phase beats blocking a connection goroutine, but it must
			// not happen in practice: traceBuffer is sized for the worst-case hop,
			// so reaching this means the capacity assumption is wrong.
		}
	}

	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			send(traceEvent{begin: true, phase: modelPhaseDNS, at: time.Now()})
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			send(traceEvent{phase: modelPhaseDNS, at: time.Now()})
		},
		ConnectStart: func(string, string) {
			send(traceEvent{begin: true, phase: modelPhaseConnect, at: time.Now()})
		},
		ConnectDone: func(_, _ string, err error) {
			send(traceEvent{phase: modelPhaseConnect, detail: errText(err), at: time.Now()})
		},
		TLSHandshakeStart: func() {
			send(traceEvent{begin: true, phase: modelPhaseTLS, at: time.Now()})
		},
		TLSHandshakeDone: func(_ tlsState, err error) {
			send(traceEvent{phase: modelPhaseTLS, detail: errText(err), at: time.Now()})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			// The write finished, so the client is now polling for the response.
			// `waiting` opens here rather than before Do so its duration measures
			// the wait itself, and the first-byte hook below is guaranteed to run
			// after this, so it always finds the phase open.
			// Both stamps are taken here, on the connection goroutine: drain runs
			// after Do returns, so stamping there would collapse every duration.
			now := time.Now()
			send(traceEvent{phase: model.PhaseRequestSent, detail: errText(info.Err), at: now})
			send(traceEvent{begin: true, phase: modelPhaseWaiting, at: now})
		},
		GotFirstResponseByte: func() {
			// The first response byte ends the wait. Both phases are ended here,
			// in the one place that observes the transition: ending them in doHop
			// as well recorded a second, zero-duration `waiting` entry and left
			// `request-sent` measuring the whole round trip.
			send(traceEvent{phase: modelPhaseWaiting, detail: "first byte", at: time.Now()})
		},
	}
}

// String renders the options for diagnostics and log lines.
func (o Options) String() string {
	return fmt.Sprintf("transport{redirects:%v max:%d proxy:%v timeout:%dms}",
		o.FollowRedirects, o.MaxRedirects, o.Proxy != nil, o.TimeoutMs)
}
