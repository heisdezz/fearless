package dialogs

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestStorageLocationModalIsRequired: the first-run dialog must not be
// dismissable — the app has nowhere to write until it is answered.
func TestStorageLocationModalIsRequired(t *testing.T) {
	q := NewQueue()

	var got string
	wait, err := q.StorageLocationModal(func(ctx context.Context, location string) error {
		got = location
		return nil
	})
	if err != nil {
		t.Fatalf("StorageLocationModal: %v", err)
	}

	if q.Kind() != KindStorageLocation {
		t.Fatalf("kind = %v, want KindStorageLocation", q.Kind())
	}
	if !q.Required() {
		t.Error("storage dialog must be required")
	}

	// Cancel is ignored: the only way out is to answer.
	q.Cancel()
	if q.Phase() != PhaseOpen {
		t.Fatalf("after Cancel, phase = %v, want PhaseOpen", q.Phase())
	}

	// OK is blocked until a path is given.
	q.OK()
	if q.Phase() != PhaseOpen {
		t.Fatalf("OK with an empty path must not close; phase = %v", q.Phase())
	}

	q.SetValue("location", "/tmp/collections")
	q.OK()

	// The action runs on a goroutine; wait for it to close the dialog.
	deadline := time.Now().Add(2 * time.Second)
	for wait.Done() == false {
		if time.Now().After(deadline) {
			t.Fatal("dialog never completed after a successful choice")
		}
		time.Sleep(time.Millisecond)
	}

	values, err := wait.Result()
	if err != nil {
		t.Fatalf("Result err = %v", err)
	}
	if values["location"] != "/tmp/collections" {
		t.Fatalf("result location = %q", values["location"])
	}
	if got != "/tmp/collections" {
		t.Fatalf("onChoose got %q, want /tmp/collections", got)
	}
	if q.Active() {
		t.Error("dialog should be closed after a successful choice")
	}
}

// TestStorageLocationModalKeepsOpenOnError: a folder that cannot be used leaves
// the dialog up so the user can pick another.
func TestStorageLocationModalKeepsOpenOnError(t *testing.T) {
	q := NewQueue()

	_, err := q.StorageLocationModal(func(ctx context.Context, location string) error {
		return errors.New("cannot use that folder")
	})
	if err != nil {
		t.Fatalf("StorageLocationModal: %v", err)
	}

	q.SetValue("location", "/nope")
	q.OK()

	// The action runs on a goroutine; wait for the failure to land.
	deadline := time.Now().Add(2 * time.Second)
	for q.Phase() != PhaseFailed {
		if time.Now().After(deadline) {
			t.Fatalf("phase = %v, want PhaseFailed", q.Phase())
		}
		time.Sleep(time.Millisecond)
	}

	if msg := q.Message(); msg == "" {
		t.Error("a failed folder choice should show its error in the dialog")
	}
	if !q.Active() {
		t.Error("a failed choice must not close the dialog")
	}
}

// TestBrowseFillsLocation: the Browse button writes the picker's choice into
// the path field, and a cancelled pick changes nothing.
func TestBrowseFillsLocation(t *testing.T) {
	q := NewQueue()

	next := "/home/user/collections"
	q.SetPathPicker(func() (string, error) { return next, nil })
	if !q.CanBrowse() {
		t.Fatal("CanBrowse should be true once a picker is installed")
	}

	if _, err := q.StorageLocationModal(func(context.Context, string) error { return nil }); err != nil {
		t.Fatalf("StorageLocationModal: %v", err)
	}

	q.Browse()
	if got := q.Value("location"); got != next {
		t.Fatalf("after Browse, location = %q, want %q", got, next)
	}

	// A cancelled pick leaves the value alone.
	q.SetPathPicker(func() (string, error) { return "", errors.New("cancelled") })
	q.Browse()
	if got := q.Value("location"); got != next {
		t.Fatalf("cancelled Browse changed location to %q", got)
	}
}

// TestOnlyStorageDialogIsRequired: the required flag must not leak onto other
// dialogs.
func TestOnlyStorageDialogIsRequired(t *testing.T) {
	q := NewQueue()
	if _, err := q.ImportModal(func(context.Context, *Imported) error { return nil }); err != nil {
		t.Fatalf("ImportModal: %v", err)
	}
	if q.Required() {
		t.Error("import dialog must remain cancellable")
	}
	q.Cancel()
	if q.Active() {
		t.Error("import dialog should close on Cancel")
	}
}
