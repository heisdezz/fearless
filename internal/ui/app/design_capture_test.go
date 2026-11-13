package app

import (
	"errors"
	"hash/crc32"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/events"
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
//
// The fixture is a working desk, not an empty window: a collection with a
// folder of requests, an environment to run against, a request open and a
// response already back. A capture of an empty app shows chrome and nothing
// else, which is not what a reviewer has to judge.
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
		Body: &model.Body{
			Mode: model.BodyJSON,
			JSON: strp(`{"name":"ada","email":"ada@example.com"}`),
		},
	}
	listReq := &model.Request{
		Name:   "List users",
		Method: "GET",
		URL:    "{{baseUrl}}/v1/users?page=1",
	}
	healthReq := &model.Request{
		Name:   "Health",
		Method: "GET",
		URL:    "{{baseUrl}}/health",
	}

	col := &model.Collection{
		UID: "col-1",
		// The name says the data is sample data: a capture that presents
		// fixture rows as a real collection is a claim the fixture does not
		// back.
		Name:              "Payments API (sample)",
		ActiveEnvironment: "Staging",
		Environments: []*model.Environment{
			{Name: "Staging"},
			{Name: "Production"},
		},
		Items: []*model.Item{
			{
				UID:  "folder-1",
				Name: "Users",
				Type: model.ItemFolder,
				Children: []*model.Item{
					{UID: "r1", Name: "Create user", Type: model.ItemHTTPRequest, Request: req},
					{UID: "r2", Name: "List users", Type: model.ItemHTTPRequest, Request: listReq},
				},
			},
			{UID: "r3", Name: "Health", Type: model.ItemHTTPRequest, Request: healthReq},
		},
	}

	loaded.CollectionPresenter.SetCollections([]*model.Collection{col})
	loaded.EnvironmentPresenter.SetCollection(col)
	loaded.CollectionPresenter.SetExpanded("folder-1", true)
	loaded.CollectionPresenter.Select("r1")
	loaded.RequestPresenter.SetItem(&model.Item{UID: "r1", Name: "Create user", Type: model.ItemHTTPRequest, Request: req})

	// A response already back, so the status seam carries a real line:
	// 200 OK · 412 ms · 2.4 kB.
	loaded.ResponsePresenter.HandleEvent(events.RequestQueued{UID: "r1"})
	loaded.ResponsePresenter.HandleEvent(events.ResponseReceived{
		UID: "r1",
		Received: model.Result{
			Status:     200,
			StatusText: "OK",
			Duration:   412 * time.Millisecond,
			Headers: model.ResponseHeaders{
				{Name: strp("Content-Type"), Value: strp("application/json"), Enabled: true},
				{Name: strp("X-Request-Id"), Value: strp("01HQ7Z9K3M"), Enabled: true},
				{Name: strp("Cache-Control"), Value: strp("no-store"), Enabled: true},
			},
			Body: model.BodyRef{Size: 2412, Inline: []byte(`{"id":"usr_42","name":"ada"}`)},
			Timeline: model.Timeline{
				{Phase: model.PhaseConnect, Duration: 12 * time.Millisecond},
				{Phase: model.PhaseRequestSent, Duration: 38 * time.Millisecond},
				{Phase: model.PhaseDownload, Duration: 362 * time.Millisecond},
			},
		},
	})

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
		// Let the modal's entrance animation finish, so the capture shows the
		// dialog open rather than mid-fade.
		tt.Advance(600 * time.Millisecond)
		tt.Frame()
		path := filepath.Join(outDir, name+".png")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		if err := embedProvenance(path, captureProvenance); err != nil {
			t.Fatal(err)
		}
	}
	render("empty-dark", empty, 1440, 900, true)
	render("request-dark", loaded, 1440, 900, true)
	render("request-light", loaded, 1440, 900, false)

	// The New Request dialog, over the empty workspace: the form a user meets
	// before any request exists.
	composing := New()
	composing.NewRequest("")
	render("new-request-dark", composing, 900, 620, true)
}

// captureProvenance is the record every capture carries inside itself: how
// the frame was produced and what its data is.
const captureProvenance = "Captured by the app's own headless renderer: " +
	"CAPTURE_DESIGN=1 go test ./internal/ui/app/ -run TestCaptureDesign " +
	"(internal/ui/app/design_capture_test.go, 1440x900). Drawn with the " +
	"Graphite Console world in internal/ui/theme/graphite.go. Every name, " +
	"URL and figure in the frame is synthetic fixture data."

// embedProvenance writes the capture's provenance into the PNG as a tEXt
// chunk, so a raster that ships carries its own origin: the file says what
// made it, without a side file that can drift away from it.
//
// The Impeccable toolchain's embed-prompt command is not available here, and
// the capture regenerates these files, so the record has to travel with the
// writer.
func embedProvenance(path, provenance string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	const signature = "\x89PNG\r\n\x1a\n"
	if len(raw) < len(signature) || string(raw[:len(signature)]) != signature {
		return errors.New("capture: not a PNG")
	}

	// The IHDR chunk is always first: 8 bytes of signature, then a 4-byte
	// length, the 4-byte type, 13 bytes of header and 4 bytes of CRC.
	const ihdrEnd = 8 + 4 + 4 + 13 + 4
	if len(raw) < ihdrEnd {
		return errors.New("capture: truncated PNG")
	}

	data := append([]byte("Provenance\x00"), []byte(provenance)...)
	chunk := make([]byte, 0, len(data)+12)
	chunk = append(chunk, byte(len(data)>>24), byte(len(data)>>16), byte(len(data)>>8), byte(len(data)))
	chunk = append(chunk, 't', 'E', 'X', 't')
	chunk = append(chunk, data...)
	crc := crc32.ChecksumIEEE(chunk[4:])
	chunk = append(chunk, byte(crc>>24), byte(crc>>16), byte(crc>>8), byte(crc))

	out := make([]byte, 0, len(raw)+len(chunk))
	out = append(out, raw[:ihdrEnd]...)
	out = append(out, chunk...)
	out = append(out, raw[ihdrEnd:]...)

	return os.WriteFile(path, out, 0o644)
}
