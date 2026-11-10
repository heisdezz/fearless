package tls

import (
	"net/url"
	"regexp"
	"strings"
)

// MatchDomain checks whether a target URL, host:port, or domain matches a configured domain pattern.
// It implements the Bruno domain matching rules:
//   - Pattern wildcard '*' matches any characters ('.*').
//   - Leading scheme prefixes (https://, grpc://, grpcs://, ws://, wss://) are accepted.
//   - Exact hostname and suffix wildcard matching (*.domain.com) are supported.
func MatchDomain(pattern, target string) bool {
	pattern = strings.TrimSpace(pattern)
	target = strings.TrimSpace(target)
	if pattern == "" || target == "" {
		return false
	}

	// 1. Direct equality (case-insensitive)
	if strings.EqualFold(pattern, target) {
		return true
	}

	// 2. Bruno regex matching on full URL/target string:
	// regex: ^(https:\/\/|grpc:\/\/|grpcs:\/\/|ws:\/\/|wss:\/\/)? + escapedDomain(with * -> .*)
	if matchBrunoRegex(pattern, target) {
		return true
	}

	// 3. Normalized Hostname matching
	targetHost := extractHostname(target)
	patternHost := extractHostname(pattern)
	if targetHost != "" && patternHost != "" {
		if strings.EqualFold(targetHost, patternHost) {
			return true
		}
		if matchWildcardHost(patternHost, targetHost) {
			return true
		}
	}

	return false
}

// matchBrunoRegex compiles and evaluates the regex defined by upstream Bruno:
// ^(https:\/\/|grpc:\/\/|grpcs:\/\/|ws:\/\/|wss:\/\/)?<escaped domain with * as .*>
func matchBrunoRegex(pattern, target string) bool {
	// Escape special regex characters in pattern, but keep '*' as wildcard
	var sb strings.Builder
	sb.WriteString(`^(?i)(https:\/\/|grpc:\/\/|grpcs:\/\/|ws:\/\/|wss:\/\/)?`)

	// Strip any leading scheme from pattern if user entered it
	pat := pattern
	for _, scheme := range []string{"https://", "grpc://", "grpcs://", "ws://", "wss://", "http://"} {
		if strings.HasPrefix(strings.ToLower(pat), scheme) {
			pat = pat[len(scheme):]
			break
		}
	}

	for i := 0; i < len(pat); i++ {
		ch := pat[i]
		switch ch {
		case '*':
			sb.WriteString(`.*`)
		case '.', '+', '?', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(ch)
		default:
			sb.WriteByte(ch)
		}
	}

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return false
	}
	return re.MatchString(target)
}

// extractHostname parses out host without scheme, port, or path.
func extractHostname(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Add dummy scheme if missing so url.Parse parses host properly
	toParse := raw
	if !strings.Contains(raw, "://") {
		toParse = "https://" + raw
	}

	u, err := url.Parse(toParse)
	if err == nil && u.Hostname() != "" {
		return strings.ToLower(u.Hostname())
	}

	// Fallback: strip port and path manually
	s := raw
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.Index(s, ":"); idx != -1 {
		s = s[:idx]
	}
	return strings.ToLower(s)
}

// matchWildcardHost matches *.example.com against sub.example.com
func matchWildcardHost(patternHost, targetHost string) bool {
	patternHost = strings.ToLower(patternHost)
	targetHost = strings.ToLower(targetHost)

	if patternHost == "*" {
		return true
	}

	if strings.HasPrefix(patternHost, "*.") {
		suffix := patternHost[1:] // ".example.com"
		if strings.HasSuffix(targetHost, suffix) {
			return true
		}
		// Also match root domain: *.example.com matches example.com
		if targetHost == patternHost[2:] {
			return true
		}
	}

	return false
}
