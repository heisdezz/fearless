package environment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DotEnvFileName is the collection-level `.env` file.
//
// Bruno also watches `.env.*` variants (`.env.local`, `.env.production`) and
// exposes them in the UI, but only a plain `.env` actually populates
// `process.env` for a request.
//
// Source: bruno-electron/src/app/dotenv-watcher.js:39-41
const DotEnvFileName = ".env"

// IsDotEnvFile reports whether a base filename is a dotenv file Bruno watches.
//
// Source: bruno-electron/src/app/dotenv-watcher.js:7-9
func IsDotEnvFile(name string) bool {
	return name == DotEnvFileName || strings.HasPrefix(name, DotEnvFileName+".")
}

// ErrDotEnvSyntax reports a malformed `.env` file.
var ErrDotEnvSyntax = errors.New("environment: malformed .env file")

// ParseDotEnv parses dotenv content into a map.
//
// The grammar follows what dotenv files in the wild require, which is what
// Bruno's `parseDotEnv` accepts:
//
//   - `KEY=value`, with optional spaces around `=`
//   - `export KEY=value`
//   - `#` starts a comment, at the start of a line or after a value
//   - double-quoted values honour \n, \r, \t, \\ and \" and keep `#`
//   - single-quoted values are literal
//   - unquoted values are trimmed and may not contain a bare `#`
//   - a trailing backslash joins the next line (multi-line values)
//
// The last assignment of a key wins, matching dotenv's behaviour everywhere.
func ParseDotEnv(content string) (map[string]string, error) {
	out := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			// A line with no `=` is not an assignment. dotenv treats it as an
			// error; so do we, rather than silently dropping a typo.
			return nil, fmt.Errorf("%w: line %d has no assignment: %q", ErrDotEnvSyntax, i+1, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("%w: line %d has an empty name", ErrDotEnvSyntax, i+1)
		}

		// Join continuation lines before interpreting quoting.
		value := rawValue
		for endsWithContinuation(value) && i+1 < len(lines) {
			i++
			value = strings.TrimSuffix(value, "\\") + "\n" + lines[i]
		}

		decoded, err := decodeDotEnvValue(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("line %d (%s): %w", i+1, key, err)
		}
		out[key] = decoded
	}
	return out, nil
}

// endsWithContinuation reports an unquoted trailing backslash.
//
// A backslash inside a quoted value is an escape, not a continuation, so the
// check ignores lines whose value already opened a quote.
func endsWithContinuation(value string) bool {
	trimmed := strings.TrimRight(value, " \t")
	if !strings.HasSuffix(trimmed, "\\") {
		return false
	}
	return !strings.HasPrefix(strings.TrimSpace(value), `"`) &&
		!strings.HasPrefix(strings.TrimSpace(value), `'`)
}

// decodeDotEnvValue interprets the value half of an assignment.
func decodeDotEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	switch value[0] {
	case '"':
		// Find the closing quote, honouring escapes.
		var b strings.Builder
		closed := false
		for i := 1; i < len(value); i++ {
			c := value[i]
			if c == '\\' && i+1 < len(value) {
				i++
				switch value[i] {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				case '"':
					b.WriteByte('"')
				case '\\':
					b.WriteByte('\\')
				default:
					// An unknown escape keeps both characters, which is what
					// dotenv does rather than failing the file.
					b.WriteByte('\\')
					b.WriteByte(value[i])
				}
				continue
			}
			if c == '"' {
				closed = true
				break
			}
			b.WriteByte(c)
		}
		if !closed {
			return "", fmt.Errorf("%w: unterminated double-quoted value", ErrDotEnvSyntax)
		}
		return b.String(), nil

	case '\'':
		end := strings.IndexByte(value[1:], '\'')
		if end < 0 {
			return "", fmt.Errorf("%w: unterminated single-quoted value", ErrDotEnvSyntax)
		}
		return value[1 : 1+end], nil
	}

	// Unquoted: a `#` preceded by whitespace starts a comment.
	if idx := indexUnquotedComment(value); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value), nil
}

// indexUnquotedComment finds a comment marker in an unquoted value.
//
// `#` only starts a comment at the beginning of the value or after whitespace,
// so a value like `pass#word` keeps its `#`.
func indexUnquotedComment(value string) int {
	for i := 0; i < len(value); i++ {
		if value[i] != '#' {
			continue
		}
		if i == 0 {
			return i
		}
		if value[i-1] == ' ' || value[i-1] == '\t' {
			return i
		}
	}
	return -1
}

// LoadDotEnv reads a collection's `.env` file.
//
// A missing file is not an error: many collections have none, and `process.env`
// then simply holds nothing. An unreadable or malformed file is an error,
// because silently ignoring it would make a request use the wrong host.
func LoadDotEnv(collectionRoot string) (map[string]string, error) {
	path := filepath.Join(collectionRoot, DotEnvFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("environment: read %s: %w", path, err)
	}
	vars, err := ParseDotEnv(string(data))
	if err != nil {
		return nil, fmt.Errorf("environment: %s: %w", DotEnvFileName, err)
	}
	return vars, nil
}

// MergeInto merges .env values into the process environment.
//
// Explicit OS variables win over the file: the point of a `.env` is to supply
// what the shell did not, and a CI job that exports a variable expects that
// value to be used.
func MergeInto(dst, dotenv map[string]string) {
	for key, value := range dotenv {
		if _, exists := dst[key]; exists {
			continue
		}
		dst[key] = value
	}
}
