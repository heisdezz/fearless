package proxy

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"fearless/internal/core/model"
)

// Config represents a concrete proxy server configuration.
type Config struct {
	// Protocol is "http", "https", "socks5", or "socks5h". Defaults to "http".
	Protocol string `json:"protocol,omitempty"`
	// Hostname is the proxy host (IP or domain name).
	Hostname string `json:"hostname,omitempty"`
	// Port is the proxy port (numeric string or int string).
	Port string `json:"port,omitempty"`
	// Username is the optional proxy authentication username.
	Username string `json:"username,omitempty"`
	// Password is the optional proxy authentication password.
	Password string `json:"password,omitempty"`
	// Bypass lists CIDRs, hostnames, or domain patterns that bypass this proxy.
	Bypass []string `json:"bypass,omitempty"`
	// Disabled explicitly turns off proxying when true, forcing a direct connection.
	Disabled bool `json:"disabled,omitempty"`
}

// URL constructs a standard *url.URL representing the proxy endpoint.
// Returns nil if c is nil, Disabled is true, or Hostname is empty.
func (c *Config) URL() (*url.URL, error) {
	if c == nil || c.Disabled || strings.TrimSpace(c.Hostname) == "" {
		return nil, nil
	}

	proto := strings.ToLower(strings.TrimSpace(c.Protocol))
	if proto == "" {
		proto = "http"
	}
	switch proto {
	case "http", "https", "socks5", "socks5h":
		// valid
	default:
		return nil, fmt.Errorf("unsupported proxy protocol: %q", c.Protocol)
	}

	host := strings.TrimSpace(c.Hostname)
	port := strings.TrimSpace(c.Port)
	if port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return nil, fmt.Errorf("invalid proxy port: %q", port)
		}
		host = net.JoinHostPort(host, port)
	}

	u := &url.URL{
		Scheme: proto,
		Host:   host,
	}

	if c.Username != "" || c.Password != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.Username, c.Password)
		} else {
			u.User = url.User(c.Username)
		}
	}

	return u, nil
}

// CollectionProxy holds collection-level proxy settings from bruno.json.
type CollectionProxy struct {
	// Inherit indicates that the collection should inherit the application proxy.
	// Default is true when absent in bruno.json.
	Inherit bool `json:"inherit"`
	// Disabled explicitly forces a direct connection for this collection.
	Disabled bool `json:"disabled"`
	// Config holds collection-specific proxy settings when Inherit is false.
	Config *Config `json:"config,omitempty"`
}

// FromModelSettings converts model.ProxySettings to *Config.
func FromModelSettings(s *model.ProxySettings) *Config {
	if s == nil || s.Mode == model.ProxyOff {
		return &Config{Disabled: true}
	}
	if s.Mode != model.ProxyManual {
		return nil
	}
	var pwd string
	if s.Password != nil {
		pwd = *s.Password
	}
	return &Config{
		Protocol: s.Protocol,
		Hostname: s.Hostname,
		Port:     s.Port,
		Username: s.Username,
		Password: pwd,
		Bypass:   s.Bypass,
	}
}
