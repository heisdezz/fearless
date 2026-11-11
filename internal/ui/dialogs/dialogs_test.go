package dialogs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// waitFor polls until cond holds or the deadline passes. It keeps the state
// machine tests free of arbitrary sleeps while still accommodating the goroutine
// that runs an action.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestConfirmClosesOnlyOnSuccess is the package's acceptance criterion: a modal
// closes after its action succeeds.
func TestConfirmClosesOnlyOnSuccess(t *testing.T) {
	q := NewQueue()
	ran := false
	w, err := q.Confirm("Delete", "Delete this request?", "Delete", func(context.Context, map[string]string) error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	if !q.Active() || q.Phase() != PhaseOpen {
		t.Fatalf("phase = %s, want open", q.Phase())
	}
	if q.Kind() != KindConfirm || q.Title() != "Delete" || q.OKText() != "Delete" {
		t.Errorf("dialog = %s/%q/%q", q.Kind(), q.Title(), q.OKText())
	}

	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	if !ran {
		t.Error("the action did not run")
	}
	if q.Active() {
		t.Error("dialog still active after a successful action")
	}
	if q.Phase() != PhaseClosed {
		t.Errorf("phase = %s, want closed", q.Phase())
	}
}

// TestConfirmStaysOpenOnFailure pins the other half: a failed action keeps the
// dialog up with the error visible so the user can retry.
func TestConfirmStaysOpenOnFailure(t *testing.T) {
	q := NewQueue()
	boom := errors.New("disk is full")
	w, err := q.Confirm("Save", "Save it?", "Save", func(context.Context, map[string]string) error {
		return boom
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	q.OK()
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	if !q.Active() {
		t.Fatal("dialog closed despite a failed action")
	}
	if got := q.Message(); !strings.Contains(got, "disk is full") {
		t.Errorf("message = %q, want the error text", got)
	}
	if w.Done() {
		t.Error("the future resolved even though the dialog is still open")
	}

	// Cancel now succeeds in backing out.
	q.Cancel()
	if _, err := w.Result(); !errors.Is(err, ErrSuperseded) {
		t.Fatalf("result err = %v, want ErrSuperseded", err)
	}
	if q.Active() {
		t.Error("dialog still active after cancel")
	}
}

// TestRetryAfterFailure covers the retry path the failure state exists for.
func TestRetryAfterFailure(t *testing.T) {
	q := NewQueue()
	var calls int
	w, err := q.Confirm("Go", "Try?", "Try", func(context.Context, map[string]string) error {
		calls++
		if calls == 1 {
			return errors.New("first attempt failed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	q.OK()
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	// A retry must be allowed: Busy is false in PhaseFailed.
	if q.Busy() {
		t.Fatal("Busy is true in the failed phase, so the user cannot retry")
	}
	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("retry result: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if q.Active() {
		t.Error("dialog still active after the retry succeeded")
	}
}

// TestOKIsIdempotentWhileRunning makes a double click harmless.
func TestOKIsIdempotentWhileRunning(t *testing.T) {
	q := NewQueue()
	release := make(chan struct{})
	var calls int32
	var mu sync.Mutex
	w, err := q.Confirm("Go", "", "", func(context.Context, map[string]string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release
		return nil
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	q.OK()
	// Busy must be true while the action runs, which is what a view binds the
	// disabled state to.
	if !q.Busy() {
		t.Error("Busy is false while the action runs")
	}
	q.OK() // second press: must not start a second action
	close(release)

	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("action ran %d times, want 1", calls)
	}
}

// TestPromptCollectsValues covers the prompt flow that feeds PromptVars.
func TestPromptCollectsValues(t *testing.T) {
	q := NewQueue()
	var got map[string]string
	w, err := q.Prompt("Variables", []string{"token", "user"}, func(_ context.Context, values map[string]string) error {
		got = values
		return nil
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if q.Kind() != KindPrompt {
		t.Errorf("kind = %s", q.Kind())
	}
	fields := q.Fields()
	if len(fields) != 2 || fields[0].Key != "token" || fields[1].Key != "user" {
		t.Fatalf("fields = %+v", fields)
	}
	// Required-and-empty means OK is unavailable until both are filled.
	if !q.Busy() {
		t.Error("Busy is false with empty required fields")
	}

	q.SetValue("token", "abc")
	if !q.Busy() {
		t.Error("Busy is false with one required field still empty")
	}
	q.SetValue("user", "  ") // whitespace is not a value
	if !q.Busy() {
		t.Error("a whitespace-only required value satisfied Busy")
	}
	q.SetValue("user", "ada")
	if q.Busy() {
		t.Error("Busy is true with both fields filled")
	}

	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	if got["token"] != "abc" || got["user"] != "ada" {
		t.Errorf("values = %+v", got)
	}
}

// TestPromptCancelSupersedes covers a user backing out of a variable prompt.
func TestPromptCancelSupersedes(t *testing.T) {
	q := NewQueue()
	w, err := q.Prompt("Variables", []string{"token"}, nil)
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	q.SetValue("token", "abc")
	q.Cancel()

	values, err := w.Result()
	if !errors.Is(err, ErrSuperseded) {
		t.Fatalf("err = %v, want ErrSuperseded", err)
	}
	if values != nil {
		t.Errorf("values = %+v, want nil after cancel", values)
	}
}

// TestSecondDialogSupersedesFirst covers the one-at-a-time rule: opening a
// dialog interrupts whatever was asking.
func TestSecondDialogSupersedesFirst(t *testing.T) {
	q := NewQueue()
	first, err := q.Confirm("First", "", "", nil)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := q.Confirm("Second", "", "", nil)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if _, err := first.Result(); !errors.Is(err, ErrSuperseded) {
		t.Fatalf("first err = %v, want ErrSuperseded", err)
	}
	if q.Title() != "Second" {
		t.Errorf("title = %q, want Second", q.Title())
	}
	if second.Done() {
		t.Error("the second dialog is already done")
	}
	q.Cancel()
}

// TestOpenRejectedWhileRunning covers the misuse guard: a second dialog cannot
// displace one whose action is mid-flight.
func TestOpenRejectedWhileRunning(t *testing.T) {
	q := NewQueue()
	block := make(chan struct{})
	defer close(block)
	if _, err := q.Confirm("Busy", "", "", func(context.Context, map[string]string) error {
		<-block
		return nil
	}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	q.OK()
	waitFor(t, "the running phase", func() bool { return q.Phase() == PhaseRunning })

	if _, err := q.Confirm("Other", "", "", nil); err == nil {
		t.Fatal("open succeeded while another action was running")
	}
	if q.Title() != "Busy" {
		t.Errorf("title = %q, want Busy: the running dialog was disturbed", q.Title())
	}
}

// TestCancelWhileRunningDiscardsResult covers a cancel that races an in-flight
// action: the dialog closes, the caller is told, and the late success does not
// reopen it.
func TestCancelWhileRunningDiscardsResult(t *testing.T) {
	q := NewQueue()
	started := make(chan struct{})
	proceed := make(chan struct{})
	w, err := q.Confirm("Go", "", "", func(ctx context.Context, _ map[string]string) error {
		close(started)
		<-proceed
		return ctx.Err()
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	q.OK()
	<-started
	q.Cancel()

	if _, err := w.Result(); !errors.Is(err, ErrSuperseded) {
		t.Fatalf("err = %v, want ErrSuperseded", err)
	}
	if q.Active() {
		t.Error("dialog active after cancel")
	}

	close(proceed)
	// Give the action goroutine a chance to finish; the dialog must stay closed.
	time.Sleep(10 * time.Millisecond)
	if q.Active() {
		t.Error("the late action result reopened a closed dialog")
	}
	if q.Phase() != PhaseClosed {
		t.Errorf("phase = %s, want closed", q.Phase())
	}
}

// TestSetValueIgnoredWhileRunning keeps an in-flight action's inputs stable.
func TestSetValueIgnoredWhileRunning(t *testing.T) {
	q := NewQueue()
	block := make(chan struct{})
	defer close(block)
	if _, err := q.Prompt("Vars", []string{"a"}, func(context.Context, map[string]string) error {
		<-block
		return nil
	}); err != nil {
		t.Fatalf("prompt: %v", err)
	}
	q.SetValue("a", "first")
	q.OK()
	waitFor(t, "the running phase", func() bool { return q.Phase() == PhaseRunning })

	q.SetValue("a", "second")
	if got := q.Value("a"); got != "first" {
		t.Errorf("value = %q, want first: input changed mid-action", got)
	}
}

// TestEditingClearsStaleError covers the UX of a failed form: typing dismisses
// the error so it is not left hanging over corrected input.
func TestEditingClearsStaleError(t *testing.T) {
	q := NewQueue()
	if _, err := q.Prompt("Vars", []string{"a"}, func(context.Context, map[string]string) error {
		return errors.New("no good")
	}); err != nil {
		t.Fatalf("prompt: %v", err)
	}
	q.SetValue("a", "x")
	q.OK()
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	q.SetValue("a", "y")
	if q.Phase() != PhaseOpen {
		t.Errorf("phase = %s, want open after editing", q.Phase())
	}
	if msg := q.Message(); msg != "" {
		t.Errorf("message = %q, want empty after editing", msg)
	}
}

// TestOnChangedFires covers the redraw hook the shell registers.
func TestOnChangedFires(t *testing.T) {
	q := NewQueue()
	var n int
	var mu sync.Mutex
	q.OnChanged(func() {
		mu.Lock()
		n++
		mu.Unlock()
	})

	if _, err := q.Confirm("A", "", "", nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	q.Cancel()

	mu.Lock()
	defer mu.Unlock()
	if n < 2 {
		t.Errorf("listener fired %d times, want at least 2", n)
	}
}

// TestListenerMayReadQueue catches the reentrancy bug a lock held across the
// notify would cause: the listener calls back into the queue.
func TestListenerMayReadQueue(t *testing.T) {
	q := NewQueue()
	done := make(chan struct{})
	q.OnChanged(func() {
		// Reading from inside the callback must not deadlock.
		_ = q.Phase()
		_ = q.Title()
		_ = q.Busy()
	})
	go func() {
		if _, err := q.Confirm("A", "", "", nil); err != nil {
			t.Errorf("confirm: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock: the listener's read blocked inside notify")
	}
}

// TestZeroWaitReportsSuperseded covers the nil-safe zero value.
func TestZeroWaitReportsSuperseded(t *testing.T) {
	var w *Wait
	if !w.Done() {
		t.Error("a nil Wait reports not done")
	}
	if _, err := w.Result(); !errors.Is(err, ErrSuperseded) {
		t.Errorf("err = %v", err)
	}
}

// TestDoneDoesNotConsumeResult keeps Done and Result consistent.
func TestDoneDoesNotConsumeResult(t *testing.T) {
	q := NewQueue()
	w, err := q.Confirm("A", "", "", nil)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	q.OK()

	deadline := time.Now().Add(2 * time.Second)
	for !w.Done() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !w.Done() {
		t.Fatal("never became done")
	}
	// Result must still work after a peek.
	if _, err := w.Result(); err != nil {
		t.Fatalf("result after Done: %v", err)
	}
	// Repeated peeks stay consistent.
	if !w.Done() {
		t.Error("Done went back to false")
	}
}
