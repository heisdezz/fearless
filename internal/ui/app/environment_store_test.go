package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

// newBruCollection creates a minimal .bru collection on disk and returns its
// directory: a bruno.json plus an empty top-level request.
func newBruCollection(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bruno.json"),
		[]byte(`{"version":"1","name":"Cren API","type":"collection"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Login.bru"),
		[]byte("meta {\n  name: Login\n  type: http\n  seq: 1\n}\n\npost {\n  url: {{baseUrl}}/api/login\n  body: none\n  auth: none\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// loadInto opens dir the way the app does and binds the environment presenter.
func loadInto(t *testing.T, a *App, dir string) *model.Collection {
	t.Helper()
	a.loadCollectionDir(dir)
	cols := a.CollectionPresenter.Collections()
	if len(cols) != 1 {
		t.Fatalf("%d collections open, want 1", len(cols))
	}
	a.bindEnvironment(cols[0].UID)
	return cols[0]
}

// TestCreatingEnvironmentWritesTheFile: creating an environment from the editor
// writes environments/<Name>.bru inside the collection, so the environment
// survives a reload and can be picked in the environment dropdown.
func TestCreatingEnvironmentWritesTheFile(t *testing.T) {
	dir := newBruCollection(t)
	a := New()
	loadInto(t, a, dir)

	if _, err := a.EnvironmentPresenter.CreateEnvironment(t.Context(), "dev"); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "environments", "dev.bru")); err != nil {
		t.Fatalf("environment file not written: %v", err)
	}
	reloaded, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Environments) != 1 || reloaded.Environments[0].Name != "dev" {
		t.Fatalf("reloaded environments = %v, want [dev]", namesOf(reloaded.Environments))
	}
}

// TestSavingEnvironmentPersistsAVariable: the value typed in the editor reaches
// the file, so it is there after a restart.
func TestSavingEnvironmentPersistsAVariable(t *testing.T) {
	dir := newBruCollection(t)
	if err := os.WriteFile(filepath.Join(dir, "environments", "dev.bru"),
		[]byte("vars {\n  baseUrl: http://old\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	loadInto(t, a, dir)
	a.EnvironmentPresenter.SetEditingEnvironment("dev")

	a.EnvironmentPresenter.SetVariable("baseUrl", "http://localhost:8080", false, true)
	if err := a.EnvironmentPresenter.Save(t.Context()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "environments", "dev.bru"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if want := "http://localhost:8080"; !strings.Contains(string(data), want) {
		t.Fatalf("saved environment does not contain %q:\n%s", want, data)
	}
}

// TestDeletingEnvironmentRemovesTheFile: deleting from the editor removes the
// file, not just the row.
func TestDeletingEnvironmentRemovesTheFile(t *testing.T) {
	dir := newBruCollection(t)
	if err := os.WriteFile(filepath.Join(dir, "environments", "dev.bru"),
		[]byte("vars {\n  baseUrl: http://x\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	loadInto(t, a, dir)

	if err := a.EnvironmentPresenter.DeleteEnvironment(t.Context(), "dev"); err != nil {
		t.Fatalf("DeleteEnvironment: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "environments", "dev.bru")); !os.IsNotExist(err) {
		t.Fatalf("environment file still present: %v", err)
	}
}

// TestSwitchingActiveEnvironmentSticks: picking an environment in the dropdown
// sets the collection's active environment, which is what request variables
// resolve against.
func TestSwitchingActiveEnvironmentSticks(t *testing.T) {
	dir := newBruCollection(t)
	if err := os.WriteFile(filepath.Join(dir, "environments", "dev.bru"),
		[]byte("vars {\n  baseUrl: http://dev\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	col := loadInto(t, a, dir)

	if err := a.EnvironmentPresenter.SelectEnvironment(t.Context(), "dev"); err != nil {
		t.Fatalf("SelectEnvironment: %v", err)
	}
	if col.ActiveEnvironment != "dev" {
		t.Fatalf("active environment = %q, want dev", col.ActiveEnvironment)
	}
	if v, ok := a.EnvironmentPresenter.VariableValue("baseUrl"); !ok || v != "http://dev" {
		t.Fatalf("VariableValue(baseUrl) = %q, %v; want http://dev, true", v, ok)
	}
}

// TestInMemoryCollectionStillEditsEnvironments: an imported collection with no
// folder (or a storage folder not chosen yet) has no file to write. The editor
// must still accept edits rather than silently dropping them.
func TestInMemoryCollectionStillEditsEnvironments(t *testing.T) {
	a, col := envTestApp(t)
	col.Path = "" // memory-only, like an import before first run is answered
	a.bindEnvironment(col.UID)

	if _, err := a.EnvironmentPresenter.CreateEnvironment(t.Context(), "dev"); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	if envs := a.EnvironmentPresenter.Environments(); len(envs) != 1 {
		t.Fatalf("environments = %v, want one", namesOf(envs))
	}
}
