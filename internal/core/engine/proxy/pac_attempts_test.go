package proxy

import (
	"errors"
	"testing"
)

// TestAttemptsOfPreservesRankOrder is the unit half of B-020: PAC's return value
// is a ranked list of attempts, and the resolver must not collapse it to one.
func TestAttemptsOfPreservesRankOrder(t *testing.T) {
	got, err := AttemptsOf(PACResult{
		Raw: "PROXY a.example:3128; PROXY b.example:3128; DIRECT",
		Attempts: []PACDirective{
			{Kind: PACHTTP, Host: "a.example", Port: "3128"},
			{Kind: PACHTTP, Host: "b.example", Port: "3128"},
			{Kind: PACDirect},
		},
	})
	if err != nil {
		t.Fatalf("AttemptsOf: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d attempts, want 3: %v", len(got), got)
	}
	if got[0] == nil || got[0].Host != "a.example:3128" {
		t.Errorf("attempt 0 = %v, want a.example:3128", got[0])
	}
	if got[1] == nil || got[1].Host != "b.example:3128" {
		t.Errorf("attempt 1 = %v, want b.example:3128", got[1])
	}
	// DIRECT is a nil URL: "connect directly", distinct from "no attempt".
	if got[2] != nil {
		t.Errorf("attempt 2 = %v, want nil for DIRECT", got[2])
	}
}

// TestAttemptsOfEmptyIsAnError distinguishes "the script named nothing" from a
// DIRECT attempt, so a broken PAC file is not silently treated as direct.
func TestAttemptsOfEmptyIsAnError(t *testing.T) {
	_, err := AttemptsOf(PACResult{})
	if !errors.Is(err, ErrPACEmptyResult) {
		t.Fatalf("err = %v, want ErrPACEmptyResult", err)
	}
}

// TestAttemptsOfRejectsUnsupportedScheme keeps a PAC file asking for something
// this client cannot do visible, rather than skipping it silently.
func TestAttemptsOfRejectsUnsupportedScheme(t *testing.T) {
	_, err := AttemptsOf(PACResult{
		Attempts: []PACDirective{{Kind: PACDirectiveKind(-1), Host: "x"}},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown directive kind")
	}
}
