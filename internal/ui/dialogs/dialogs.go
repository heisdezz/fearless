// Package dialogs implements the modal flows that sit above the screens:
// confirmations, the prompt-variable form, import, and save-transient.
//
// The package is split in two halves that are deliberately unrelated:
//
//   - The *state machine* (Queue, Prompt, Confirm, ImportState, SaveState) is
//     pure Go. It decides what is open, what it holds, and whether it may close.
//     It imports no Lobe or MyGo package and is unit-tested without a renderer.
//   - The *views* (ConfirmModal, PromptModal, ImportModal, SaveTransientModal)
//     draw the current state and report clicks upward. They never mutate a
//     command's result and never close a dialog on their own.
//
// The split exists because the acceptance criterion for this package
// (docs/progress/packages.md, `UI-DIALOGS`) is that "feedback.Modal closes only
// after the action succeeds". Lobe's Modal already has the right shape — it
// returns true on OK and leaves *open to the caller — so the rule is enforced
// once, in the state machine, rather than repeated in every view.
package dialogs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Kind identifies which flow a dialog belongs to.
type Kind int

const (
	// KindConfirm asks a yes/no question.
	KindConfirm Kind = iota
	// KindPrompt collects values for prompt variables (`{{?name}}`).
	KindPrompt
	// KindImport converts a pasted document into a request.
	KindImport
	// KindSaveTransient asks where to save a request that has no home yet.
	KindSaveTransient
)

// String is used in tests and logs.
func (k Kind) String() string {
	switch k {
	case KindConfirm:
		return "confirm"
	case KindPrompt:
		return "prompt"
	case KindImport:
		return "import"
	case KindSaveTransient:
		return "save-transient"
	}
	return "unknown"
}

// Phase is where a dialog is in its lifecycle.
//
// The important transition is PhaseRunning → PhaseFailed: a dialog that is
// running an action stays open, and only leaves PhaseRunning for PhaseClosed
// when the action reported success. PhaseFailed keeps it open with the error
// visible so the user can retry or cancel.
type Phase int

const (
	// PhaseClosed means nothing is shown.
	PhaseClosed Phase = iota
	// PhaseOpen means the user is editing or deciding.
	PhaseOpen
	// PhaseRunning means the action is in flight; OK is disabled.
	PhaseRunning
	// PhaseFailed means the action returned an error; the dialog stays open.
	PhaseFailed
)

// String is used in tests.
func (p Phase) String() string {
	switch p {
	case PhaseClosed:
		return "closed"
	case PhaseOpen:
		return "open"
	case PhaseRunning:
		return "running"
	case PhaseFailed:
		return "failed"
	}
	return "unknown"
}

// Action is the work a dialog performs when OK is pressed.
//
// It receives the dialog's own context, which is cancelled when the dialog is
// closed, so a slow import or save does not outlive the window that started it.
// Returning nil is the only way the dialog closes; returning an error switches
// it to PhaseFailed and shows the message.
type Action func(ctx context.Context, values map[string]string) error

// ErrSuperseded is returned by the future of a dialog that was replaced by
// another. Callers awaiting a result use it to distinguish "the user cancelled"
// from "the user opened something else".
var ErrSuperseded = errors.New("dialog superseded by another")

// result carries the outcome of one dialog to whoever awaited it.
type result struct {
	values map[string]string
	err    error
}

// Queue is the single owner of dialog state.
//
// Only one dialog is shown at a time. Opening a second one fails the first with
// ErrSuperseded rather than stacking modals, which matches Bruno: a command that
// opens a dialog interrupts whatever was asking a question.
type Queue struct {
	// mu guards every field below. The action goroutine writes phase when it
	// finishes while the UI goroutine reads it to draw, so the lock is not
	// optional.
	mu     sync.Mutex
	phase  Phase
	values map[string]string

	// kind and the description fields are what the view reads.
	kind      Kind
	title     string
	message   string
	okText    string
	fields    []Field
	action    Action
	actionErr string

	// result is the channel the opener awaits; nil when nobody awaits.
	result chan result
	cancel context.CancelFunc

	// listener is notified on every state change so the shell can redraw.
	listener func()

	// warnings are converter adaptations shown inside the active dialog.
	warnings []string
	// targets are the collections a save dialog may choose from.
	targets []SaveTarget
}

// Field describes one input in a prompt or import dialog.
type Field struct {
	// Key is the map key the value is stored under in values.
	Key string
	// Label is shown above the input.
	Label string
	// Placeholder is shown while the field is empty.
	Placeholder string
	// Multiline renders a text area instead of a single-line input.
	Multiline bool
	// Secret renders the value masked; it is not echoed in error reports.
	Secret bool
	// Required rejects OK while the trimmed value is empty.
	Required bool
}

// NewQueue creates an empty queue.
func NewQueue() *Queue {
	return &Queue{phase: PhaseClosed}
}

// OnChanged registers a redraw callback, replacing any previous one.
func (q *Queue) OnChanged(fn func()) {
	q.mu.Lock()
	q.listener = fn
	q.mu.Unlock()
}

// notify tells the shell to redraw. It is public-safe: it copies the listener
// under the lock and calls it with no lock held, because a redraw may read the
// queue and a lock held across that read would deadlock.
func (q *Queue) notify() {
	q.mu.Lock()
	fn := q.listener
	q.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Active reports whether a dialog is on screen.
func (q *Queue) Active() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.phase != PhaseClosed
}

// Phase returns the current phase.
func (q *Queue) Phase() Phase {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.phase
}

// Kind returns the active dialog's kind. It is meaningless when closed.
func (q *Queue) Kind() Kind {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.kind
}

// Title returns the active dialog's title.
func (q *Queue) Title() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.title
}

// Message returns the explanatory text, or the error after a failure.
func (q *Queue) Message() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.phase == PhaseFailed && q.actionErr != "" {
		return q.actionErr
	}
	return q.message
}

// OKText returns the primary button label, defaulting to "OK".
func (q *Queue) OKText() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.okText != "" {
		return q.okText
	}
	return "OK"
}

// Fields returns the inputs to render. It is a copy, so a redraw that holds the
// slice while the queue changes cannot observe a half-updated one.
func (q *Queue) Fields() []Field {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Field(nil), q.fields...)
}

// Value returns the current text of field key.
func (q *Queue) Value(key string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.values[key]
}

// Values returns a copy of every collected value, so a view cannot mutate the
// queue by writing into the map.
func (q *Queue) Values() map[string]string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return copyValues(q.values)
}

// copyValues is copyValues for a caller already holding the lock.
func copyValues(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Busy reports whether OK should be disabled: an action is running, or a
// required field is still empty.
func (q *Queue) Busy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.busyLocked()
}

// busyLocked is Busy for a caller already holding the lock.
func (q *Queue) busyLocked() bool {
	if q.phase == PhaseRunning {
		return true
	}
	if q.phase != PhaseOpen && q.phase != PhaseFailed {
		return true
	}
	for _, f := range q.fields {
		if f.Required && strings.TrimSpace(q.values[f.Key]) == "" {
			return true
		}
	}
	return false
}

// SetValue records a keystroke. It is ignored while the action runs, so late
// input cannot change the values an in-flight action is already using.
func (q *Queue) SetValue(key, value string) {
	q.mu.Lock()
	if q.phase != PhaseOpen && q.phase != PhaseFailed {
		q.mu.Unlock()
		return
	}
	if q.values == nil {
		q.values = map[string]string{}
	}
	if q.values[key] == value {
		q.mu.Unlock()
		return
	}
	q.values[key] = value
	// A failed action's error is stale once the input changes.
	if q.phase == PhaseFailed {
		q.phase = PhaseOpen
		q.actionErr = ""
	}
	q.mu.Unlock()
	q.notify()
}

// open starts a dialog, superseding any current one, and returns its future.
func (q *Queue) open(kind Kind, title, message, okText string, fields []Field, action Action) (<-chan result, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.phase == PhaseRunning {
		// The running action's context must not be cancelled underneath it by a
		// second open that the first dialog's action would then race with. The
		// shell disables the rest of the window while running, so this is a
		// programming error rather than a user flow.
		return nil, fmt.Errorf("dialogs: cannot open %s while %s is running", kind, q.kind)
	}
	q.supersedeLocked()

	_, cancel := context.WithCancel(context.Background())
	q.cancel = cancel
	q.kind = kind
	q.title = title
	q.message = message
	q.okText = okText
	q.fields = fields
	q.action = action
	q.actionErr = ""
	q.warnings = nil
	q.targets = nil
	q.phase = PhaseOpen
	q.values = make(map[string]string, len(fields))
	for _, f := range fields {
		q.values[f.Key] = ""
	}
	q.result = make(chan result, 1)
	ch := q.result
	return ch, nil
}

// supersedeLocked fails any awaited dialog with ErrSuperseded.
func (q *Queue) supersedeLocked() {
	if q.result != nil {
		sendResult(q.result, result{err: ErrSuperseded})
		q.result = nil
	}
	if q.cancel != nil {
		q.cancel()
		q.cancel = nil
	}
}

// sendResult delivers without blocking; the channel is buffered with room for
// one, and a duplicate delivery would be a bug, so a second send is dropped.
func sendResult(ch chan result, r result) {
	select {
	case ch <- r:
	default:
	}
}

// Await turns a future into a blocking call. It is the form the shell uses from
// command goroutines.
func Await(ch <-chan result) (map[string]string, error) {
	r, ok := <-ch
	if !ok {
		return nil, ErrSuperseded
	}
	return r.values, r.err
}

// Cancel closes the active dialog without running its action. When an action is
// already running, it closes the dialog and cancels the action's context; the
// action's eventual result is discarded rather than reopening a closed dialog.
func (q *Queue) Cancel() {
	q.mu.Lock()
	if q.phase == PhaseClosed {
		q.mu.Unlock()
		return
	}
	q.closeLocked(nil, ErrSuperseded)
	q.mu.Unlock()
	q.notify()
}

// closeLocked resets to PhaseClosed and resolves the future.
func (q *Queue) closeLocked(values map[string]string, err error) {
	ch, cancel := q.result, q.cancel
	q.result, q.cancel = nil, nil
	q.phase = PhaseClosed
	q.actionErr = ""
	q.warnings = nil
	q.targets = nil
	if cancel != nil {
		cancel()
	}
	if ch != nil {
		sendResult(ch, result{values: values, err: err})
	}
}

// OK is pressed. For a confirmation it runs the action. For a form it runs the
// action with the collected values. The dialog closes only if the action
// succeeds; otherwise it stays open in PhaseFailed.
//
// OK is a no-op while Busy, which makes a double click harmless: the second
// press finds PhaseRunning and returns.
func (q *Queue) OK() {
	q.mu.Lock()
	if q.busyLocked() || q.phase != PhaseOpen && q.phase != PhaseFailed {
		q.mu.Unlock()
		return
	}
	values := copyValues(q.values)
	action := q.action
	if action == nil {
		q.closeLocked(values, nil)
		q.mu.Unlock()
		q.notify()
		return
	}

	q.phase = PhaseRunning
	q.actionErr = ""
	if q.cancel != nil {
		q.cancel() // release the idle dialog context
	}
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel
	q.mu.Unlock()
	q.notify()

	go func() {
		err := action(ctx, values)
		q.complete(values, err)
	}()
}

// complete applies the action's outcome. It is the only place a running dialog
// leaves PhaseRunning, which is what keeps the "close after success" rule in one
// place.
func (q *Queue) complete(values map[string]string, err error) {
	q.mu.Lock()
	if q.phase != PhaseRunning {
		// Cancelled while running: the result is discarded, but the context was
		// already cancelled so the action should return shortly.
		q.mu.Unlock()
		return
	}
	if err != nil {
		q.phase = PhaseFailed
		q.actionErr = err.Error()
		q.mu.Unlock()
		q.notify()
		return
	}
	q.closeLocked(values, nil)
	q.mu.Unlock()
	q.notify()
}

// Confirm is the convenience form for a yes/no question. ok is the action run
// on OK.
func (q *Queue) Confirm(title, message, okText string, action Action) (*Wait, error) {
	ch, err := q.open(KindConfirm, title, message, okText, nil, action)
	if err != nil {
		return newWait(nil), err
	}
	q.notify()
	return newWait(ch), nil
}

// Prompt asks for the named variables in order. The caller supplies labels; the
// values are returned keyed by label, which is what SendRequestCmd.PromptVars
// expects (internal/app/commands/send_request.go, PromptVars).
func (q *Queue) Prompt(title string, labels []string, action Action) (*Wait, error) {
	fields := make([]Field, 0, len(labels))
	for _, label := range labels {
		fields = append(fields, Field{
			Key:         label,
			Label:       label,
			Placeholder: "value for " + label,
			Required:    true,
		})
	}
	ch, err := q.open(KindPrompt, title, "", "Send", fields, action)
	if err != nil {
		return newWait(nil), err
	}
	q.notify()
	return newWait(ch), nil
}

// Wait is a dialog's future. The zero value is a closed, already-cancelled
// dialog, so a caller can ignore an error from Confirm/Prompt without a nil
// check. It is passed by pointer so the peek cache below is shared.
type Wait struct {
	ch     <-chan result
	mu     sync.Mutex
	cached *result
	peeked bool
}

// newWait wraps a future channel. A nil channel yields a Wait that reports
// ErrSuperseded immediately, which is what a caller gets when a dialog failed to
// open.
func newWait(ch <-chan result) *Wait { return &Wait{ch: ch} }

// Result blocks until the dialog closes. It returns ErrSuperseded when the user
// cancelled, opened another dialog, or the Wait is nil.
func (w *Wait) Result() (map[string]string, error) {
	if w == nil || w.ch == nil {
		return nil, ErrSuperseded
	}
	if r := w.takeCached(); r != nil {
		return r.values, r.err
	}
	return Await(w.ch)
}

// Done reports whether the dialog has already closed, without consuming the
// result. The first peek caches the outcome so that Result and Done agree even
// when they are called from different goroutines.
func (w *Wait) Done() bool {
	if w == nil || w.ch == nil {
		return true
	}
	if w.takeCached() != nil {
		return true
	}
	select {
	case r := <-w.ch:
		w.mu.Lock()
		w.cached = &r
		w.mu.Unlock()
		return true
	default:
		return false
	}
}

// takeCached returns the peeked result, or nil when nothing was peeked.
func (w *Wait) takeCached() *result {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.cached
}
