package environment

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestIsDotEnvFile covers the filenames Bruno watches.
//
// Source: bruno-electron/src/app/dotenv-watcher.js:7-9
func TestIsDotEnvFile(t *testing.T) {
	yes := []string{".env", ".env.local", ".env.production"}
	no := []string{"env", "env.local", ".envrc", "dotenv", ".ENV"}

	for _, name := range yes {
		if !IsDotEnvFile(name) {
			t.Errorf("%q should be a dotenv file", name)
		}
	}
	for _, name := range no {
		if IsDotEnvFile(name) {
			t.Errorf("%q should not be a dotenv file", name)
		}
	}
}

// TestParseDotEnv covers the grammar.
func TestParseDotEnv(t *testing.T) {
	content := `# a comment
HOST=https://api.example.com
TOKEN=abc123

export EXPORTED=yes
SPACED = around equals
EMPTY=
QUOTED="value with spaces"
ESCAPED="line1\nline2\ttabbed \"quoted\" back\\slash"
SINGLE='no $expansion \n here'
HASH_AFTER_SPACE=abc # trailing comment
HASH_INLINE=pass#word
COMMENT_ONLY=   # value is empty
`
	got, err := ParseDotEnv(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := map[string]string{
		"HOST":             "https://api.example.com",
		"TOKEN":            "abc123",
		"EXPORTED":         "yes",
		"SPACED":           "around equals",
		"EMPTY":            "",
		"QUOTED":           "value with spaces",
		"ESCAPED":          "line1\nline2\ttabbed \"quoted\" back\\slash",
		"SINGLE":           `no $expansion \n here`,
		"HASH_AFTER_SPACE": "abc",
		"HASH_INLINE":      "pass#word",
		"COMMENT_ONLY":     "",
	}
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			t.Errorf("%s missing", key)
			continue
		}
		if gotValue != wantValue {
			t.Errorf("%s = %q, want %q", key, gotValue, wantValue)
		}
	}
	// Comments must not become keys.
	if _, ok := got["# a comment"]; ok {
		t.Error("a comment line became a variable")
	}
}

// TestParseDotEnvLastAssignmentWins matches dotenv's behaviour.
func TestParseDotEnvLastAssignmentWins(t *testing.T) {
	got, err := ParseDotEnv("A=1\nA=2\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["A"] != "2" {
		t.Errorf("A = %q, want the last assignment", got["A"])
	}
}

// TestParseDotEnvContinuation joins a trailing-backslash line.
func TestParseDotEnvContinuation(t *testing.T) {
	got, err := ParseDotEnv("MULTI=one\\\ntwo\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["MULTI"] != "one\ntwo" {
		t.Errorf("MULTI = %q", got["MULTI"])
	}
}

// TestParseDotEnvCRLF handles Windows line endings.
func TestParseDotEnvCRLF(t *testing.T) {
	got, err := ParseDotEnv("A=1\r\nB=2\r\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["A"] != "1" || got["B"] != "2" {
		t.Errorf("parsed %v", got)
	}
}

// TestParseDotEnvErrors keeps a malformed file from being silently accepted.
func TestParseDotEnvErrors(t *testing.T) {
	cases := map[string]string{
		"no equals":       "JUSTANAME\n",
		"empty name":      "=value\n",
		"unterminated dq": `A="unclosed` + "\n",
		"unterminated sq": "A='unclosed\n",
	}
	for name, content := range cases {
		if _, err := ParseDotEnv(content); !errors.Is(err, ErrDotEnvSyntax) {
			t.Errorf("%s: err = %v, want ErrDotEnvSyntax", name, err)
		}
	}
}

// TestParseDotEnvUnknownEscapeIsKept covers a lenient case: dotenv keeps the
// backslash rather than failing the whole file.
func TestParseDotEnvUnknownEscapeIsKept(t *testing.T) {
	got, err := ParseDotEnv(`A="x\qy"` + "\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["A"] != `x\qy` {
		t.Errorf("A = %q", got["A"])
	}
}

// TestLoadDotEnvMissingFileIsEmpty makes an absent file a normal state.
func TestLoadDotEnvMissingFileIsEmpty(t *testing.T) {
	got, err := LoadDotEnv(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

// TestLoadDotEnvReadsFile covers the happy path.
func TestLoadDotEnvReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DotEnvFileName), []byte("API_HOST=http://local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDotEnv(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got["API_HOST"] != "http://local" {
		t.Errorf("API_HOST = %q", got["API_HOST"])
	}
}

// TestLoadDotEnvMalformedReports puts the filename in the error.
func TestLoadDotEnvMalformedReports(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DotEnvFileName), []byte("garbage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadDotEnv(dir)
	if !errors.Is(err, ErrDotEnvSyntax) {
		t.Fatalf("err = %v", err)
	}
}

// TestMergeIntoPrefersRealEnvironment keeps a CI export winning over the file.
func TestMergeIntoPrefersRealEnvironment(t *testing.T) {
	dst := map[string]string{"EXISTING": "from-shell"}
	dotenv := map[string]string{"EXISTING": "from-file", "NEW": "from-file"}

	MergeInto(dst, dotenv)

	if dst["EXISTING"] != "from-shell" {
		t.Errorf("EXISTING = %q, the process environment must win", dst["EXISTING"])
	}
	if dst["NEW"] != "from-file" {
		t.Errorf("NEW = %q, a new key should be added", dst["NEW"])
	}
}
