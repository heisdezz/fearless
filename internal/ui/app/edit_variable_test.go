package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
)

func strp(s string) *string { return &s }

// urlBarApp is an app showing one request whose URL uses baseUrl, and an
// environment that defines it.
func urlBarApp(t *testing.T) *App {
	t.Helper()
	col := &model.Collection{
		UID:               "c1",
		Name:              "Demo",
		ActiveEnvironment: "dev",
		Environments: []*model.Environment{{
			UID:  "e1",
			Name: "dev",
			Variables: []model.Variable{
				{Name: strp("baseUrl"), Value: "https://api.example.com", Enabled: true},
			},
		}},
	}
	a := New()
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	a.EnvironmentPresenter.SetCollection(col)
	a.RequestPresenter.SetItem(&model.Item{
		UID:  "r1",
		Name: "Login",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Name:   "Login",
			Method: "GET",
			URL:    "{{baseUrl}}/api/login",
		},
	})
	return a
}

// TestEditVariable_OpensTheEditorOnThatVariable: a `{{variable}}` opened from
// the URL bar brings the environment editor up with that variable in hand and a
// way back to the request.
func TestEditVariable_OpensTheEditorOnThatVariable(t *testing.T) {
	a := urlBarApp(t)

	a.EditVariable("baseUrl")

	if a.ActiveView != ViewKindEnvironment {
		t.Fatalf("active view = %v, want the environment editor", a.ActiveView)
	}
	if got := a.environmentState.Focus; got != "baseUrl" {
		t.Errorf("focused variable = %q, want baseUrl", got)
	}
	if a.environmentState.Back == nil {
		t.Error("the editor opened from a variable must offer a way back")
	}

	// The request that opened it is still the one shown, so Back returns to it.
	a.environmentState.Back()
	if a.ActiveView != ViewKindMain {
		t.Errorf("after Back the view = %v, want the main view", a.ActiveView)
	}
}

// TestEditVariable_CreatesAMissingVariable: an undefined `{{variable}}` has
// nothing to edit yet, so opening it adds it to the environment draft, ready to
// fill in, instead of failing.
func TestEditVariable_CreatesAMissingVariable(t *testing.T) {
	a := urlBarApp(t)

	a.EditVariable("apiToken")
	if got := a.environmentState.Focus; got != "apiToken" {
		t.Errorf("focused variable = %q, want apiToken", got)
	}

	// The editor adds the variable as it draws the table, so it has a row to
	// highlight.
	tt := ui.NewTester(func(c *ui.Context) { a.RootView(c) }, 1280, 800)
	tt.Frame()

	env := a.EnvironmentPresenter.EffectiveEnvironment()
	if env == nil {
		t.Fatal("no effective environment")
	}
	if !environmentDefines(env, "apiToken") {
		t.Fatalf("apiToken was not added; variables: %v", env.Variables)
	}
}

// TestEnvironmentEditorShowsFocus: the editor highlights the variable it was
// opened for once, and creates the missing one so it has a row to highlight.
func TestEnvironmentEditorShowsFocus(t *testing.T) {
	a := urlBarApp(t)
	a.EditVariable("apiToken")

	tt := ui.NewTester(func(c *ui.Context) { a.RootView(c) }, 1280, 800)
	tt.Frame()

	if !tt.HasText("Environments") || !tt.HasText("Back") {
		t.Fatalf("the environment editor did not open; texts: %v", tt.Texts())
	}
	if !environmentDefines(a.EnvironmentPresenter.EffectiveEnvironment(), "apiToken") {
		t.Error("the variable to edit was not added to the table")
	}
	if a.environmentState.Focus != "" {
		t.Errorf("the focus should be consumed after one frame, still %q", a.environmentState.Focus)
	}
}
