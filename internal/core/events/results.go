package events

import "fearless/internal/core/model"

// Phases a test or assertion result can belong to. These match the script hooks
// Bruno exposes, so a result can be attributed to the block that produced it.
const (
	PhasePreRequest   = "pre-request"
	PhasePostResponse = "post-response"
	PhaseTests        = "tests"
)

// TestResults reports the outcome of the `test()` calls for one phase.
//
// The runner publishes one event per phase rather than one per test, so the UI
// can render a phase's results atomically.
type TestResults struct {
	UID string
	// Phase is one of "pre-request", "post-response" or "tests".
	Phase string
	// Results preserve the order the tests ran in.
	Results []model.TestResult
}

// Kind implements Event.
func (TestResults) Kind() Kind { return KindTestResults }

// Summmary counts the results by status.
func (t TestResults) Summary() (passed, failed, skipped int) {
	for _, r := range t.Results {
		switch r.Status {
		case StatusPass:
			passed++
		case StatusFail:
			failed++
		case StatusSkip:
			skipped++
		}
	}
	return passed, failed, skipped
}

// Test statuses.
const (
	StatusPass  = "pass"
	StatusFail  = "fail"
	StatusSkip  = "skip"
	StatusError = "error"
)

// AssertionResults reports the outcome of the `assert {}` block.
type AssertionResults struct {
	UID string
	// Results preserve the order the assertions appear in the source.
	Results []model.AssertionResult
}

// Kind implements Event.
func (AssertionResults) Kind() Kind { return KindAssertionResults }

// AllPassed reports whether every assertion succeeded.
func (a AssertionResults) AllPassed() bool {
	for _, r := range a.Results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// DomainCookies is the cookie set for one domain.
//
// Cookies carry values, so this is the one event whose payload is sensitive.
// Consumers must not log it verbatim; the redaction tests cover it.
type DomainCookies struct {
	Domain  string
	Cookies []model.Cookie
}

// CookiesUpdated publishes the complete cookie state after a change.
//
// It is a full snapshot of the affected domains, not a delta, so a consumer may
// replace its view wholesale and never needs to reconcile additions and
// deletions.
type CookiesUpdated struct {
	// Domains holds every domain whose cookies changed.
	Domains []DomainCookies
}

// Kind implements Event.
func (CookiesUpdated) Kind() Kind { return KindCookiesUpdated }

// CredentialsUpdated announces new or refreshed OAuth2 credentials.
//
// AccessToken is carried because the UI shows its expiry; the refresh token is
// deliberately absent so a leaked log line cannot mint new sessions.
type CredentialsUpdated struct {
	// CredentialsID identifies the OAuth2 configuration.
	CredentialsID string
	// AccessToken is the current bearer value.
	AccessToken string
	// TokenType is usually "Bearer".
	TokenType string
	// ExpiresIn is the token lifetime; zero when the server did not say.
	ExpiresIn int64
	// Scope is the granted scope set.
	Scope string
}

// Kind implements Event.
func (CredentialsUpdated) Kind() Kind { return KindCredentialsUpdated }

// ChangeOrigin explains who changed a collection on disk.
type ChangeOrigin string

const (
	// OriginSelf marks a change this application caused (a save).
	OriginSelf ChangeOrigin = "self"
	// OriginExternal marks a change made outside the application.
	OriginExternal ChangeOrigin = "external"
)

// TreeChange describes what happened to a collection's files.
type TreeChange struct {
	Origin ChangeOrigin
	// Added, Modified and Removed hold item UIDs where known, and file paths
	// otherwise.
	Added    []string
	Modified []string
	Removed  []string
	// Rescan reports that the change was too broad to describe item by item and
	// the consumer should reload the whole tree.
	Rescan bool
}

// CollectionChanged announces a filesystem change to a loaded collection.
//
// The watcher suppresses self-originated changes, so collecting them means the
// UI can ignore its own writes without diffing.
type CollectionChanged struct {
	CollectionUID string
	Change        TreeChange
}

// Kind implements Event.
func (CollectionChanged) Kind() Kind { return KindCollectionChanged }
