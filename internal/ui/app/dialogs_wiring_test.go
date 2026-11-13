package app

import (
	"testing"

	"fearless/internal/ui/dialogs"
)

// TestOpenCollectionOpensImportDialog covers the wiring behind "Open" and
// "Open Collection": both go through OpenCollection, which must reach the import
// dialog — the one place that accepts a folder, a file, a URL or a paste.
func TestOpenCollectionOpensImportDialog(t *testing.T) {
	a := New()
	a.OpenCollection()

	if !a.DialogQueue.Active() {
		t.Fatal("OpenCollection did not open a dialog")
	}
	if got := a.DialogQueue.Kind(); got != dialogs.KindImport {
		t.Errorf("OpenCollection opened %s, want the import dialog", got)
	}
	a.DialogQueue.Cancel()
}

// TestNewRequestDialogIsTheNewRequestKind covers the navigation and toolbar
// entries that create a request.
func TestNewRequestDialogIsTheNewRequestKind(t *testing.T) {
	a := New()
	a.NewRequest("")

	if !a.DialogQueue.Active() {
		t.Fatal("NewRequest did not open a dialog")
	}
	if got := a.DialogQueue.Kind(); got != dialogs.KindNewRequest {
		t.Errorf("NewRequest opened %s, want the new-request dialog", got)
	}
}
