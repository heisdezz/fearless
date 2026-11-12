package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestCaptureStoragePrompt writes the first-run dialog to testdata for review.
//
//	CAPTURE_DESIGN=1 go test ./internal/ui/app/ -run TestCaptureStoragePrompt
func TestCaptureStoragePrompt(t *testing.T) {
	if os.Getenv("CAPTURE_DESIGN") == "" {
		t.Skip("set CAPTURE_DESIGN=1 to write the storage-prompt capture")
	}

	for _, dark := range []bool{true, false} {
		a := New()
		a.promptStorageLocationIfNeeded()
		a.DialogQueue.SetValue("location", "/home/user/fearless")

		tt := ui.NewTester(a.RootView, 1280, 800)
		tt.SetDark(dark)
		// The dialog fades and scales in; render past its enter animation so the
		// capture shows the settled modal, not the first animated frame.
		tt.Frame()
		tt.Advance(400 * time.Millisecond)

		name := "firstrun-light"
		if dark {
			name = "firstrun-dark"
		}
		dir := filepath.Join("testdata", "design")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
}
