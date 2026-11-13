package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fearless/internal/convert/postman"
	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

// TestPostmanImportKeepsCollectionVariables is the regression for the import
// round trip dropping collection variables.
//
// The Postman import puts its `variable` entries into the collection config.
// Writing that collection to disk and reading it back used to lose them, so a
// request like `{{baseUrl}}/api/auth/login` reached net/http with `{{baseUrl}}`
// still in it and failed with "unsupported protocol scheme ”".
func TestPostmanImportKeepsCollectionVariables(t *testing.T) {
	src := filepath.Join("..", "..", "..", "testdata", "collection", "Cren.postman_collection.json")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	res, err := postman.Import(raw)
	if err != nil {
		t.Fatalf("postman import: %v", err)
	}
	if len(res.Collection.Config.Vars) == 0 {
		t.Fatal("import produced no collection variables")
	}

	dir, err := collection.CopyModelIntoStorage(res.Collection, t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	reloaded, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Config.Vars) != len(res.Collection.Config.Vars) {
		t.Fatalf("collection variables did not survive the round trip: %d before, %d after",
			len(res.Collection.Config.Vars), len(reloaded.Config.Vars))
	}
	for i, v := range res.Collection.Config.Vars {
		if got := reloaded.Config.Vars[i].NameOrEmpty(); got != v.NameOrEmpty() {
			t.Errorf("variable %d: name %q, want %q", i, got, v.NameOrEmpty())
		}
	}
}

// TestSendResolvesCollectionVariableFromDisk covers the whole path the user hit:
// import a Postman collection, reload it from disk, point its baseUrl at a real
// server and send. Before the fix the variable was gone after the reload.
func TestSendResolvesCollectionVariableFromDisk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	imported := &model.Collection{
		Name:   "Cren",
		Config: model.CollectionConfig{Format: model.FormatBru},
		Items: []*model.Item{{
			UID:  "r1",
			Name: "Log in as a member",
			Type: model.ItemHTTPRequest,
			Request: &model.Request{
				Method: "POST",
				URL:    "{{baseUrl}}/api/auth/login",
			},
		}},
	}
	name := "baseUrl"
	// The import's placeholder: a bare path, with no scheme. The user replaces
	// it with the real host, which must persist to disk.
	imported.Config.Vars = []model.Variable{{Name: &name, Value: "/", Enabled: true}}

	dir, err := collection.CopyModelIntoStorage(imported, t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	reloaded, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Config.Vars) != 1 {
		t.Fatalf("baseUrl did not survive the reload: %+v", reloaded.Config.Vars)
	}

	// The user sets the host, as the UI's variable editor would.
	reloaded.Config.Vars[0].Value = srv.URL
	if err := collection.SaveCollectionConfig(reloaded); err != nil {
		t.Fatalf("save config: %v", err)
	}
	again, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload after save: %v", err)
	}
	if got := again.Config.Vars[0].Value; got != srv.URL {
		t.Fatalf("saved baseUrl = %v, want %s", got, srv.URL)
	}

	a := New()
	a.CollectionPresenter.SetCollections([]*model.Collection{again})
	a.EnvironmentPresenter.SetCollection(again)
	item := findRequestByName(again, "Log in as a member")
	if item == nil {
		t.Fatal("request missing after reload")
	}
	a.RequestPresenter.SetItem(item)

	a.SendActiveRequest()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if a.ResponsePresenter.State() == "completed" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	res := a.ResponsePresenter.Result()
	if res == nil {
		t.Fatalf("no response; state=%q err=%q", a.ResponsePresenter.State(), a.ResponsePresenter.ErrorMessage())
	}
	if res.Sent.URL != srv.URL+"/api/auth/login" {
		t.Errorf("sent URL = %q, want the resolved base URL", res.Sent.URL)
	}
	if res.Status != http.StatusOK {
		t.Errorf("status = %d, want %d", res.Status, http.StatusOK)
	}
}

// findRequestByName walks a collection for the first request with name.
func findRequestByName(col *model.Collection, name string) *model.Item {
	var found *model.Item
	var walk func(items []*model.Item)
	walk = func(items []*model.Item) {
		for _, it := range items {
			if it == nil || found != nil {
				continue
			}
			if it.IsFolder() {
				walk(it.Children)
				continue
			}
			if it.Name == name {
				found = it
			}
		}
	}
	walk(col.Items)
	return found
}
