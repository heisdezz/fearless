package app

import (
	"testing"

	"fearless/internal/core/model"
)

// envTestApp is an app with one open collection containing a folder and a
// request, so selection can land on each and the binding can be checked.
func envTestApp(t *testing.T) (*App, *model.Collection) {
	t.Helper()
	a := New()
	col := &model.Collection{
		UID:  "col-1",
		Name: "Cren API",
		Path: t.TempDir(),
		Items: []*model.Item{
			{
				UID:  "folder-1",
				Name: "auth",
				Type: model.ItemFolder,
				Children: []*model.Item{{
					UID:     "req-1",
					Name:    "Login",
					Request: &model.Request{Method: "POST", URL: "{{baseUrl}}/api/login"},
				}},
			},
		},
	}
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	return a, col
}

// TestSelectingRequestBindsEnvironment: the environment editor reads from the
// collection bound to its presenter. Selecting anything inside a collection
// must bind that collection, or the editor reports "No collection selected"
// while the collection is plainly open.
func TestSelectingRequestBindsEnvironment(t *testing.T) {
	a, col := envTestApp(t)
	if got := a.EnvironmentPresenter.Collection(); got != nil {
		t.Fatalf("collection bound before any selection: %v", got)
	}

	a.bindEnvironment("req-1")

	if got := a.EnvironmentPresenter.Collection(); got != col {
		t.Fatalf("bound collection = %v, want the open collection", got)
	}
}

// TestSelectingCollectionRowBindsEnvironment: clicking the collection row
// itself, rather than a request in it, has to bind it too.
func TestSelectingCollectionRowBindsEnvironment(t *testing.T) {
	a, col := envTestApp(t)
	a.bindEnvironment("col-1")
	if got := a.EnvironmentPresenter.Collection(); got != col {
		t.Fatalf("bound collection = %v, want the open collection", got)
	}
}

// TestLoadedCollectionBindsEnvironment: opening a collection (or restoring one
// at startup) binds it without requiring a click first.
func TestLoadedCollectionBindsEnvironment(t *testing.T) {
	a := New()
	col := &model.Collection{UID: "col-1", Name: "Cren API", Path: "/tmp/cren"}
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	a.bindEnvironment("col-1")
	if got := a.EnvironmentPresenter.Collection(); got != col {
		t.Fatalf("bound collection = %v, want the open collection", got)
	}
}

// TestBoundCollectionCanCreateAnEnvironment: the reported bug was that no
// environment could be made. Once bound, the editor's create path works and
// reports the new environment, so the list has something to show.
func TestBoundCollectionCanCreateAnEnvironment(t *testing.T) {
	a, _ := envTestApp(t)
	a.bindEnvironment("req-1")

	if envs := a.EnvironmentPresenter.Environments(); len(envs) != 0 {
		t.Fatalf("collection starts with %d environments, want 0", len(envs))
	}

	if _, err := a.EnvironmentPresenter.CreateEnvironment(t.Context(), "dev"); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	envs := a.EnvironmentPresenter.Environments()
	if len(envs) != 1 || envs[0].Name != "dev" {
		t.Fatalf("environments = %v, want one named dev", namesOf(envs))
	}
}

func namesOf(envs []*model.Environment) []string {
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		if e != nil {
			out = append(out, e.Name)
		}
	}
	return out
}
