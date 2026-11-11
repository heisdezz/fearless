package curl

import (
	"strings"
	"unicode"
)

// Tokenize splits a command string into shell words, respecting single quotes,
// double quotes, backslash escapes, line continuation backslashes, and comments.
func Tokenize(cmd string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	hasToken := false

	// Normalize CRLF to LF
	cmd = strings.ReplaceAll(cmd, "\r\n", "\n")
	runes := []rune(cmd)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]

		// Handle line continuation: backslash followed by newline
		if r == '\\' && !inSingleQuote && i+1 < n && runes[i+1] == '\n' {
			i++ // skip the newline
			continue
		}

		// Handle comments: # outside quotes at line start (or after space at line start)
		if r == '#' && !inSingleQuote && !inDoubleQuote && !hasToken {
			// Skip until end of line
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}

		if escaped {
			escaped = false
			hasToken = true
			if inDoubleQuote {
				// In double quotes, backslash only escapes ", \, $, `, and newline
				switch r {
				case '"', '\\', '$', '`':
					current.WriteRune(r)
				case 'n':
					// Some pasted strings have literal \n or escaped sequences
					current.WriteRune('\n')
				case 'r':
					current.WriteRune('\r')
				case 't':
					current.WriteRune('\t')
				default:
					current.WriteRune('\\')
					current.WriteRune(r)
				}
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if r == '\\' {
			if inSingleQuote {
				// Look ahead: if followed by ', support forgiving \' in single quotes
				if i+1 < n && runes[i+1] == '\'' {
					current.WriteRune('\'')
					i++
					hasToken = true
					continue
				}
				current.WriteRune('\\')
			} else {
				escaped = true
			}
			continue
		}

		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			hasToken = true
			continue
		}

		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			hasToken = true
			continue
		}

		if !inSingleQuote && !inDoubleQuote {
			if unicode.IsSpace(r) {
				if hasToken {
					tokens = append(tokens, current.String())
					current.Reset()
					hasToken = false
				}
				continue
			}
		}

		current.WriteRune(r)
		hasToken = true
	}

	if hasToken {
		tokens = append(tokens, current.String())
	}

	// Strip leading curl binary if present
	if len(tokens) > 0 {
		first := strings.ToLower(tokens[0])
		if first == "curl" || strings.HasSuffix(first, "/curl") || strings.HasSuffix(first, "\\curl") ||
			first == "curl.exe" || strings.HasSuffix(first, "/curl.exe") || strings.HasSuffix(first, "\\curl.exe") {
			tokens = tokens[1:]
		}
	}

	return tokens, nil
}
