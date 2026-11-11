package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
)

// TestCaptureDesign renders the assembled app in the Graphite world and writes
// the frames to testdata/design/. It is a design-review tool, not an
// acceptance test: it asserts only that each frame renders, and it writes
// files, so it stays behind a flag.
//
//	CAPTURE_DESIGN=1 go test ./internal/ui/app/ -run TestCaptureDesign
//
// The frames are the evidence the visual pass is judged on (D-018: dark is the
// shipped appearance, light is the legible companion). They are committed so a
// reviewer can see what the build looked like without running it.
func TestCaptureDesign(t *testing.T) {
	if os.Getenv("CAPTURE_DESIGN") == "" {
		t.Skip("set CAPTURE_DESIGN=1 to write testdata/design/*.png")
	}

	strp := func(s string) *string { return &s }

	empty := New()
	loaded := New()
	req := &model.Request{
		Name:   "Create user",
		Method: "POST",
		URL:    "{{baseUrl}}/v1/users",
		Params: []model.Param{{Name: strp("page"), Value: strp("1"), Type: model.ParamQuery, Enabled: true}},
		Headers: []model.KeyValue{
			{Name: strp("Content-Type"), Value: strp("application/json"), Enabled: true},
			{Name: strp("Authorization"), Value: strp("Bearer {{token}}"), Enabled: true},
			{Name: strp("Accept"), Value: strp("application/json"), Enabled: true},
		},
	}
	loaded.RequestPresenter.SetItem(&model.Item{UID: "r1", Name: "Create user", Type: model.ItemHTTPRequest, Request: req})

	outDir := filepath.Join("testdata", "design")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	render := func(name string, a *App, w, h int, dark bool) {
		tt := ui.NewTester(a.RootView, w, h)
		// SetDark settles a frame of its own, so the appearance is already the
		// one asked for before the capture frame runs.
		tt.SetDark(dark)
		tt.Frame()
		f, err := os.Create(filepath.Join(outDir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	render("empty-dark", empty, 1440, 900, true)
	render("request-dark", loaded, 1440, 900, true)
	render("request-light", loaded, 1440, 900, false)
}
