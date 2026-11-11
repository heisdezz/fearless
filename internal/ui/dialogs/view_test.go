package dialogs

import (
	"context"
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
)

// render draws the dialog for one frame inside a headless tester.
func render(t *testing.T, q *Queue) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		View(c, q)
	}, 700, 520)
}

// TestViewDrawsNothingWhenClosed keeps the shell from painting an empty modal.
func TestViewDrawsNothingWhenClosed(t *testing.T) {
	q := NewQueue()
	tt := render(t, q)
	// Frame must not panic or request a dialog surface for a closed queue.
	tt.Frame()
}

// TestViewShowsConfirmation covers the base case: a title, a message and the
// two buttons.
func TestViewShowsConfirmation(t *testing.T) {
	q := NewQueue()
	if _, err := q.Confirm("Delete request", "This cannot be undone.", "Delete", nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	tt := render(t, q)
	tt.Frame()

	for _, want := range []string{"Delete request", "This cannot be undone.", "Delete", "Cancel"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q; texts = %v", want, tt.Texts())
		}
	}
	q.Cancel()
}

// TestViewShowsFailedActionError covers the error surface a failed action
// produces.
func TestViewShowsFailedActionError(t *testing.T) {
	q := NewQueue()
	if _, err := q.Confirm("Save", "Save it?", "Save", func(context.Context, map[string]string) error {
		return errors.New("permission denied")
	}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	q.OK()
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	tt := render(t, q)
	tt.Frame()
	if !tt.HasText("permission denied") {
		t.Errorf("the error is not visible; texts = %v", tt.Texts())
	}
}

// TestViewPromptShowsFields covers the prompt form's labels.
func TestViewPromptShowsFields(t *testing.T) {
	q := NewQueue()
	if _, err := q.Prompt("Variables", []string{"apiToken", "tenant"}, nil); err != nil {
		t.Fatalf("prompt: %v", err)
	}
	q.SetValue("apiToken", "abc")

	tt := render(t, q)
	tt.Frame()
	for _, want := range []string{"Variables", "apiToken", "tenant"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q; texts = %v", want, tt.Texts())
		}
	}
}

// TestViewSavePickerOffersCollections covers the target list rendering.
func TestViewSavePickerOffersCollections(t *testing.T) {
	q := NewQueue()
	if _, err := q.SaveTransientModal(SaveTransient{
		URL: "https://api.example.com/users",
		Targets: []SaveTarget{
			{UID: "c1", Name: "Todo API"},
		},
	}, func(context.Context, string, string) error { return nil }); err != nil {
		t.Fatalf("SaveTransientModal: %v", err)
	}

	tt := render(t, q)
	tt.Frame()
	if !tt.HasText("Collection") {
		t.Errorf("the collection field is missing; texts = %v", tt.Texts())
	}
	// The URL is shown so the user knows which request is being saved.
	if !tt.HasText("https://api.example.com/users") {
		t.Errorf("the request URL is not shown; texts = %v", tt.Texts())
	}
}

// TestViewClickingOKRunsTheAction covers the wiring from a real click through
// Lobe's Modal into the queue.
func TestViewClickingOKRunsTheAction(t *testing.T) {
	q := NewQueue()
	ran := make(chan struct{}, 1)
	w, err := q.Confirm("Go", "Proceed?", "Proceed", func(context.Context, map[string]string) error {
		ran <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	tt := render(t, q)
	tt.Frame()
	if err := tt.Click("Proceed"); err != nil {
		t.Fatalf("click: %v", err)
	}
	tt.Frame()

	select {
	case <-ran:
	default:
		t.Fatal("the action did not run after clicking the primary button")
	}

	// The dialog closes once the action succeeds; one more frame settles it.
	tt.Frame()
	deadline := 0
	for q.Active() && deadline < 100 {
		tt.Frame()
		deadline++
	}
	if q.Active() {
		t.Error("the dialog stayed open after a successful action")
	}
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
}

// TestViewFailedActionStaysOnScreen is the view-level twin of the state-machine
// acceptance test: a real click on a failing action leaves the modal up.
func TestViewFailedActionStaysOnScreen(t *testing.T) {
	q := NewQueue()
	if _, err := q.Confirm("Go", "Proceed?", "Proceed", func(context.Context, map[string]string) error {
		return errors.New("nope")
	}); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	tt := render(t, q)
	tt.Frame()
	if err := tt.Click("Proceed"); err != nil {
		t.Fatalf("click: %v", err)
	}
	waitFor(t, "the failed phase", func() bool { return q.Phase() == PhaseFailed })

	tt.Frame()
	if !q.Active() {
		t.Fatal("the modal closed on a failed action")
	}
	if !tt.HasText("nope") {
		t.Errorf("the error is not on screen; texts = %v", tt.Texts())
	}
	// The modal is still drawn after the failure.
	if !tt.HasText("Go") {
		t.Errorf("the title vanished; texts = %v", tt.Texts())
	}
}

// TestActionsFooter covers the shell-provides-its-own-chrome variant.
func TestActionsFooter(t *testing.T) {
	q := NewQueue()
	if _, err := q.Confirm("Go", "", "", nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		Actions(c, q)
	}, 400, 120)
	tt.Frame()
	if !tt.HasText("Cancel") || !tt.HasText("OK") {
		t.Fatalf("footer texts = %v", tt.Texts())
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("click: %v", err)
	}
	tt.Frame()
	if q.Active() {
		t.Error("Cancel in the footer did not close the dialog")
	}
}
