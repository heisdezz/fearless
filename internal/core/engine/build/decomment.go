package build

import (
	"strings"
)

// StripComments removes single-line (// ...) and multi-line (/* ... */) comments
// from JSON/JavaScript text while preserving strings containing slashes.
func StripComments(input string) string {
	if input == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(input))

	inString := false
	var stringDelimiter byte
	inSingleLineComment := false
	inMultiLineComment := false

	n := len(input)
	for i := 0; i < n; i++ {
		ch := input[i]

		if inSingleLineComment {
			if ch == '\n' {
				inSingleLineComment = false
				sb.WriteByte('\n')
			}
			continue
		}

		if inMultiLineComment {
			if ch == '*' && i+1 < n && input[i+1] == '/' {
				inMultiLineComment = false
				i++ // skip '/'
			}
			continue
		}

		if inString {
			sb.WriteByte(ch)
			if ch == '\\' && i+1 < n {
				i++
				sb.WriteByte(input[i])
			} else if ch == stringDelimiter {
				inString = false
			}
			continue
		}

		// Not in string or comment
		if ch == '"' || ch == '\'' || ch == '`' {
			inString = true
			stringDelimiter = ch
			sb.WriteByte(ch)
			continue
		}

		if ch == '/' && i+1 < n {
			next := input[i+1]
			if next == '/' {
				inSingleLineComment = true
				i++
				continue
			} else if next == '*' {
				inMultiLineComment = true
				i++
				continue
			}
		}

		sb.WriteByte(ch)
	}

	return sb.String()
}
