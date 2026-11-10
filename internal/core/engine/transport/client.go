package transport

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"fearless/internal/core/engine/proxy"
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

	client := &http.Client{
		Transport: tr,
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

	// The caller has already resolved which proxy to use.
	if opts.Proxy != nil {
		if err := proxy.ConfigureTransport(tr, opts.Proxy); err != nil {
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
func newTrace(tl *timeline) *httptrace.ClientTrace {
	rec := tl.trace

	send := func(ev traceEvent) {
		select {
		case rec <- ev:
		default:
			// Dropping a phase beats blocking a connection goroutine.
		}
	}

	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			send(traceEvent{begin: true, phase: modelPhaseDNS})
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			send(traceEvent{phase: modelPhaseDNS})
		},
		ConnectStart: func(string, string) {
			send(traceEvent{begin: true, phase: modelPhaseConnect})
		},
		ConnectDone: func(_, _ string, err error) {
			send(traceEvent{phase: modelPhaseConnect, detail: errText(err)})
		},
		TLSHandshakeStart: func() {
			send(traceEvent{begin: true, phase: modelPhaseTLS})
		},
		TLSHandshakeDone: func(_ tlsState, err error) {
			send(traceEvent{phase: modelPhaseTLS, detail: errText(err)})
		},
		GotFirstResponseByte: func() {
			send(traceEvent{phase: modelPhaseWaiting, detail: "first byte"})
		},
	}
}

// String renders the options for diagnostics and log lines.
func (o Options) String() string {
	return fmt.Sprintf("transport{redirects:%v max:%d proxy:%v timeout:%dms}",
		o.FollowRedirects, o.MaxRedirects, o.Proxy != nil, o.TimeoutMs)
}
