package events

import (
	"context"
	"errors"
)

// Scopes identify which command failed, so the UI can react differently to, say,
// a failed save versus a failed send.
const (
	ScopeSend        = "send"
	ScopeSave        = "save"
	ScopeLoad        = "load"
	ScopeDelete      = "delete"
	ScopeMove        = "move"
	ScopeImport      = "import"
	ScopeExport      = "export"
	ScopeRunner      = "runner"
	ScopeRefresh     = "refresh"
	ScopeCollection  = "collection"
	ScopeEnvironment = "environment"
	ScopeSettings    = "settings"
	ScopeOAuth       = "oauth"
	ScopeWatcher     = "watcher"
	ScopeScript      = "script"
	ScopeInternal    = "internal"
)

// Error classifications, coarser than Go errors because the UI needs a handful of
// distinct treatments rather than a hierarchy.
const (
	// ErrorCancelled is a user-initiated abort. It is not a failure, and the UI
	// shows "cancelled" rather than an error.
	ErrorCancelled = "cancelled"
	// ErrorTimeout is a deadline exceeded.
	ErrorTimeout = "timeout"
	// ErrorNetwork covers DNS, connection, TLS handshake and transport failures.
	ErrorNetwork = "network"
	// ErrorValidation covers rejected input, such as an unresolved variable that
	// must abort.
	ErrorValidation = "validation"
	// ErrorNotFound covers a missing file, item or collection.
	ErrorNotFound = "notfound"
	// ErrorConflict covers a concurrent-change or lock conflict.
	ErrorConflict = "conflict"
	// ErrorPermission covers an access denial.
	ErrorPermission = "permission"
	// ErrorUnknown is the fallback.
	ErrorUnknown = "unknown"
)

// ErrorOccurred is a terminal event for a failed execution, and the way any
// command reports a non-request failure.
//
// Cancellation surfaces here with ErrorCancelled so that every RequestQueued has
// exactly one terminal event: a cancelled send is not a silent end.
//
// It is a value type like every other event, which keeps a presenter's switch
// uniform. The wrapped error is held in Err but is not part of the value's
// identity: two events carrying different error values are still comparable, and
// copyable, as long as Msg is the field consumers read.
type ErrorOccurred struct {
	// UID is the execution this failure belongs to, or "" for commands that are
	// not request executions (a failed save, for example).
	UID string
	// ItemUID and CollectionUID identify the leaf and its tree, when known.
	ItemUID       string
	CollectionUID string
	// Scope is one of the ScopeXxx constants.
	Scope string
	// Class is one of the ErrorXxx constants.
	Class string
	// Msg is a user-facing message. It must never contain a secret.
	Msg string
	// Details is optional diagnostic text (a server response excerpt, a stack)
	// shown behind a disclosure.
	Details string
	// Err is the original error, when one exists. Consumers should prefer Msg;
	// Err is for callers that need errors.Is/As, such as the CLI choosing an exit
	// code.
	Err error
}

// Kind implements Event.
func (ErrorOccurred) Kind() Kind { return KindErrorOccurred }

// Error makes ErrorOccurred usable as an error itself.
func (e ErrorOccurred) Error() string { return e.Msg }

// Unwrap exposes the original error to errors.Is and errors.As.
func (e ErrorOccurred) Unwrap() error { return e.Err }

// Cancelled reports whether the failure was a user abort.
func (e ErrorOccurred) Cancelled() bool { return e.Class == ErrorCancelled }

// NewError builds an ErrorOccurred from an error, classifying it.
//
// Classification is deliberately conservative: anything not recognised becomes
// ErrorUnknown rather than being guessed, so the UI shows an honest "something
// went wrong" instead of a misleading category.
func NewError(scope string, err error) ErrorOccurred {
	ev := ErrorOccurred{Scope: scope, Class: Classify(err), Err: err}
	if err != nil {
		ev.Msg = err.Error()
	}
	return ev
}

// Classify maps an error to one of the ErrorXxx classes.
//
// Wrap the error with context or the classification will not see through it;
// errors.Is handles wrapping, so fmt.Errorf("%w") is safe.
func Classify(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled):
		return ErrorCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorTimeout
	}
	return ErrorUnknown
}

// WithMessage replaces the user-facing message, keeping the wrapped error.
func (e ErrorOccurred) WithMessage(msg string) ErrorOccurred {
	e.Msg = msg
	return e
}

// WithDetails attaches diagnostic text.
func (e ErrorOccurred) WithDetails(details string) ErrorOccurred {
	e.Details = details
	return e
}

// WithRequest attaches the execution and item identity.
func (e ErrorOccurred) WithRequest(uid, itemUID, collectionUID string) ErrorOccurred {
	e.UID = uid
	e.ItemUID = itemUID
	e.CollectionUID = collectionUID
	return e
}
