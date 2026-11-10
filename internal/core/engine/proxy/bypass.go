package proxy

import (
	"net"
	"net/url"
	"strings"
)

// MatchBypass reports whether targetURL matches any pattern in patterns,
// adhering to curl's NO_PROXY semantics.
//
// Supported patterns:
//   - "*" matches every host.
//   - Domain suffixes (e.g. ".example.com" or "example.com") match the domain and subdomains.
//   - IP addresses and CIDR subnets (e.g. "127.0.0.1", "10.0.0.0/8", "::1", "fe80::/10").
//   - Specific port qualifications (e.g. "example.com:8080", "127.0.0.1:3000").
//   - "localhost" matches localhost, 127.0.0.1, and ::1.
func MatchBypass(patterns []string, targetURL *url.URL) bool {
	if targetURL == nil || len(patterns) == 0 {
		return false
	}

	targetHost := targetURL.Hostname()
	if targetHost == "" {
		return false
	}
	targetPort := targetURL.Port()
	if targetPort == "" {
		switch strings.ToLower(targetURL.Scheme) {
		case "http":
			targetPort = "80"
		case "https":
			targetPort = "443"
		}
	}

	targetIP := net.ParseIP(targetHost)

	for _, raw := range patterns {
		// Split multiple comma-separated or space-separated items if present
		fields := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\t'
		})
		for _, pattern := range fields {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if pattern == "*" {
				return true
			}
			if matchSinglePattern(pattern, targetHost, targetPort, targetIP) {
				return true
			}
		}
	}

	return false
}

func matchSinglePattern(pattern, targetHost, targetPort string, targetIP net.IP) bool {
	pattern = strings.ToLower(pattern)
	targetHostLower := strings.ToLower(targetHost)

	// Check if pattern has a port specified (e.g. "foo.com:8080" or "10.0.0.1:8080")
	// Note: IPv6 CIDRs may contain colons, so only split if not a CIDR and valid port.
	var patternHost, patternPort string
	if !strings.Contains(pattern, "/") {
		if h, p, err := net.SplitHostPort(pattern); err == nil {
			patternHost = h
			patternPort = p
		} else {
			patternHost = pattern
		}
	} else {
		patternHost = pattern
	}

	if patternPort != "" && patternPort != targetPort {
		return false
	}

	// 1. CIDR matching: e.g. 192.168.0.0/16, 10.0.0.0/8, fe80::/10
	if strings.Contains(patternHost, "/") {
		_, ipNet, err := net.ParseCIDR(patternHost)
		if err == nil && targetIP != nil {
			if ipNet.Contains(targetIP) {
				return true
			}
		}
		return false
	}

	// 2. IP address exact match
	if patIP := net.ParseIP(patternHost); patIP != nil {
		if targetIP != nil && targetIP.Equal(patIP) {
			return true
		}
		// Special case: 127.0.0.1 and ::1 match target host "localhost"
		if targetHostLower == "localhost" && (patIP.IsLoopback() || patIP.Equal(net.IPv4(127, 0, 0, 1))) {
			return true
		}
		return false
	}

	// 3. Localhost alias
	if patternHost == "localhost" {
		if targetHostLower == "localhost" {
			return true
		}
		if targetIP != nil && targetIP.IsLoopback() {
			return true
		}
	}

	// 4. Domain / hostname suffix matching (curl rules)
	patClean := strings.TrimPrefix(patternHost, ".")

	if targetHostLower == patClean {
		return true
	}

	if strings.HasSuffix(targetHostLower, "."+patClean) {
		return true
	}

	return false
}
