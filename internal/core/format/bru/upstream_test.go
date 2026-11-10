package bru

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseBrunoUpstreamCorpus runs the parser over every `.bru` file in the
// Bruno clone, which is the broadest conformance check available.
//
// Files in the legacy v1 dialect (`type http-request`, `key value` with no colon)
// are counted separately rather than treated as failures: they are a different
// grammar that Bruno stopped writing. The v2 dialect must parse with zero
// failures.
func TestParseBrunoUpstreamCorpus(t *testing.T) {
	root := "../../../../bruno/packages"
	if _, err := os.Stat(root); err != nil {
		t.Skip("upstream clone not present")
	}

	var total, v2Parsed, v1Skipped int
	var v2Failures []string

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".bru") {
			return nil
		}
		if strings.Contains(path, "node_modules") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		total++

		if isV1Dialect(data) {
			v1Skipped++
			return nil
		}

		if _, err := Parse(data); err != nil {
			v2Failures = append(v2Failures, path+": "+err.Error())
			return nil
		}
		v2Parsed++
		return nil
	})

	t.Logf("corpus: %d files total, %d parsed as v2, %d skipped as v1, %d v2 failures",
		total, v2Parsed, v1Skipped, len(v2Failures))

	for _, f := range v2Failures {
		t.Errorf("v2 parse failure: %s", f)
	}
	if v2Parsed == 0 {
		t.Skip("no v2 fixtures found")
	}
	// Guard against a regression that silently treats everything as v1.
	if v1Skipped > total/4 {
		t.Errorf("%d of %d files classified as v1; the detector is probably wrong",
			v1Skipped, total)
	}
}

// isV1Dialect reports whether the source is in Bruno's legacy v1 format.
//
// v1 files begin with bare `type http-request` lines and use `/block` closers
// instead of braces; detecting them up front keeps the v2 parser honest.
func isV1Dialect(src []byte) bool {
	text := string(src)
	if strings.Contains(text, "\n/params") || strings.Contains(text, "\n/headers") ||
		strings.Contains(text, "\n/vars") || strings.Contains(text, "\n/body") {
		return true
	}
	first := strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	return first == "type http-request" || first == "name" || strings.HasPrefix(first, "name ")
}
