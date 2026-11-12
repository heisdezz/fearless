package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
)

// TestCaptureEnvironment writes the environment editor and the URL bar's
// variable tokens to testdata for review.
//
//	CAPTURE_DESIGN=1 go test ./internal/ui/app/ -run TestCaptureEnvironment
func TestCaptureEnvironment(t *testing.T) {
	if os.Getenv("CAPTURE_DESIGN") == "" {
		t.Skip("set CAPTURE_DESIGN=1 to write the environment captures")
	}
	dir := filepath.Join("testdata", "design")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	col := &model.Collection{
		UID:               "c1",
		Name:              "Fearless Collection",
		ActiveEnvironment: "dev",
		Environments: []*model.Environment{{
			UID:  "e1",
			Name: "dev",
			Variables: []model.Variable{
				mkVar("baseUrl", "https://api.example.com", false),
				mkVar("token", "s3cr3t-value", true),
				mkVar("userId", "42", false),
			},
		}, {
			UID:  "e2",
			Name: "prod",
			Variables: []model.Variable{
				mkVar("baseUrl", "https://api.example.com", false),
			},
		}},
	}

	// 1. Environment editor.
	a := New()
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	a.EnvironmentPresenter.SetCollection(col)
	a.EnvironmentPresenter.SetEditingEnvironment("dev")
	a.ActiveView = ViewKindEnvironment

	capture(t, filepath.Join(dir, "environment-dark.png"), a.RootView, true)

	// 2. URL bar with defined and undefined variables.
	b := New()
	b.CollectionPresenter.SetCollections([]*model.Collection{col})
	b.EnvironmentPresenter.SetCollection(col)
	b.RequestPresenter.SetItem(&model.Item{
		UID:  "r1",
		Name: "Login",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Name:   "Login",
			Method: "GET",
			URL:    "{{baseUrl}}/api/login?user={{userId}}&k={{missing}}",
		},
	})
	capture(t, filepath.Join(dir, "urlbar-dark.png"), b.RootView, true)
}

func mkVar(name, value string, secret bool) model.Variable {
	n := name
	return model.Variable{Name: &n, Value: value, Enabled: true, Secret: secret}
}

func capture(t *testing.T, path string, view func(*ui.Context), dark bool) {
	t.Helper()
	tt := ui.NewTester(view, 1280, 800)
	tt.SetDark(dark)
	tt.Frame()
	tt.Advance(400 * time.Millisecond)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
