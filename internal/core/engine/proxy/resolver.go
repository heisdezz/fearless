package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"fearless/internal/core/model"

	xproxy "golang.org/x/net/proxy"
)

// ResolutionInput collects proxy configurations from all four scopes:
// Request > Collection > App preferences > Environment.
type ResolutionInput struct {
	// TargetURL is the destination URL being requested.
	TargetURL *url.URL

	// RequestProxy is the optional per-request proxy override.
	RequestProxy *Config

	// CollectionProxy is the optional collection-level proxy configuration.
	CollectionProxy *CollectionProxy

	// AppSettings is the global proxy configuration from application preferences.
	AppSettings *model.ProxySettings

	// PACResolver evaluates PAC mode. It is required only when AppSettings.Mode
	// is ProxyPAC; a nil resolver there is an error rather than a silent fall
	// through to the environment, because silently ignoring a PAC file would
	// send traffic through the wrong route.
	PACResolver PACSource

	// Context bounds PAC fetching and evaluation. Nil means Background.
	Context context.Context

	// LookupEnv allows overriding os.LookupEnv for deterministic testing.
	LookupEnv func(string) (string, bool)
}

// Resolve determines the effective proxy URL for targetURL according to the
// precedence order in flows/request-lifecycle.md §3:
//
//	Request > Collection > App Preferences > Environment
//
// Returns (nil, nil) if a direct connection should be used (e.g. proxy disabled,
// bypassed, or none configured).
func Resolve(input ResolutionInput) (*url.URL, error) {
	if input.TargetURL == nil {
		return nil, errors.New("target URL must not be nil")
	}

	lookup := input.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}

	// 1. Request scope
	if input.RequestProxy != nil {
		if input.RequestProxy.Disabled {
			return nil, nil // Request explicitly disables proxy
		}
		if MatchBypass(input.RequestProxy.Bypass, input.TargetURL) {
			return nil, nil // Bypassed at request level
		}
		u, err := input.RequestProxy.URL()
		if err != nil {
			return nil, fmt.Errorf("request proxy: %w", err)
		}
		if u != nil {
			return u, nil
		}
		// If RequestProxy was present but empty without host/disabled, fall through
	}

	// 2. Collection scope
	if input.CollectionProxy != nil {
		if input.CollectionProxy.Disabled {
			return nil, nil // Collection explicitly disables proxy
		}
		if !input.CollectionProxy.Inherit && input.CollectionProxy.Config != nil {
			cfg := input.CollectionProxy.Config
			if cfg.Disabled {
				return nil, nil
			}
			if MatchBypass(cfg.Bypass, input.TargetURL) {
				return nil, nil // Bypassed at collection level
			}
			u, err := cfg.URL()
			if err != nil {
				return nil, fmt.Errorf("collection proxy: %w", err)
			}
			if u != nil {
				return u, nil
			}
		}
		// If Inherit is true (or empty config), fall through to App scope
	}

	// 3. Application Preferences scope
	if input.AppSettings != nil {
		switch input.AppSettings.Mode {
		case model.ProxyOff:
			return nil, nil // Globally turned off
		case model.ProxyManual:
			cfg := FromModelSettings(input.AppSettings)
			if cfg != nil && !cfg.Disabled {
				if MatchBypass(cfg.Bypass, input.TargetURL) {
					return nil, nil // Bypassed at app level
				}
				u, err := cfg.URL()
				if err != nil {
					return nil, fmt.Errorf("app manual proxy: %w", err)
				}
				if u != nil {
					return u, nil
				}
			}
			return nil, nil
		case model.ProxySystem:
			// Fall through to system environment variables
		case model.ProxyPAC:
			return resolveFromPAC(input)
		}
	}

	// 4. Environment scope (HTTP_PROXY / HTTPS_PROXY / ALL_PROXY / NO_PROXY)
	return resolveFromEnvironment(input.TargetURL, lookup)
}

func resolveFromEnvironment(targetURL *url.URL, lookup func(string) (string, bool)) (*url.URL, error) {
	// Check NO_PROXY / no_proxy
	noProxy, ok := lookup("NO_PROXY")
	if !ok {
		noProxy, ok = lookup("no_proxy")
	}
	if ok && strings.TrimSpace(noProxy) != "" {
		if MatchBypass([]string{noProxy}, targetURL) {
			return nil, nil
		}
	}

	// Look up proxy URL based on target scheme
	var proxyEnv string
	scheme := strings.ToLower(targetURL.Scheme)

	switch scheme {
	case "https":
		for _, key := range []string{"HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"} {
			if val, ok := lookup(key); ok && strings.TrimSpace(val) != "" {
				proxyEnv = val
				break
			}
		}
	case "http":
		for _, key := range []string{"HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
			if val, ok := lookup(key); ok && strings.TrimSpace(val) != "" {
				proxyEnv = val
				break
			}
		}
	default:
		for _, key := range []string{"ALL_PROXY", "all_proxy"} {
			if val, ok := lookup(key); ok && strings.TrimSpace(val) != "" {
				proxyEnv = val
				break
			}
		}
	}

	if strings.TrimSpace(proxyEnv) == "" {
		return nil, nil
	}

	proxyEnv = strings.TrimSpace(proxyEnv)
	if !strings.Contains(proxyEnv, "://") {
		proxyEnv = "http://" + proxyEnv
	}

	u, err := url.Parse(proxyEnv)
	if err != nil {
		return nil, fmt.Errorf("parse env proxy %q: %w", proxyEnv, err)
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5", "socks5h":
		return u, nil
	default:
		return nil, fmt.Errorf("unsupported env proxy scheme %q", u.Scheme)
	}
}

// ConfigureTransport configures http.Transport with the appropriate proxy settings:
//   - http/https: uses standard Transport.Proxy
//   - socks5/socks5h: uses golang.org/x/net/proxy Dialer
//   - nil: clears proxy and restores direct dialing
func ConfigureTransport(tr *http.Transport, proxyURL *url.URL) error {
	if tr == nil {
		return errors.New("cannot configure nil http.Transport")
	}

	if proxyURL == nil {
		tr.Proxy = nil
		tr.DialContext = nil
		return nil
	}

	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https":
		tr.Proxy = http.ProxyURL(proxyURL)
		tr.DialContext = nil
	case "socks5", "socks5h":
		dialer, err := xproxy.FromURL(proxyURL, xproxy.Direct)
		if err != nil {
			return fmt.Errorf("create socks5 dialer: %w", err)
		}
		tr.Proxy = nil
		if contextDialer, ok := dialer.(xproxy.ContextDialer); ok {
			tr.DialContext = contextDialer.DialContext
		} else {
			tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
	default:
		return fmt.Errorf("unsupported proxy protocol: %q", proxyURL.Scheme)
	}

	return nil
}
