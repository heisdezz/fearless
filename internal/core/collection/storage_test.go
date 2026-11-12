package collection

import (
	"os"
	"path/filepath"
	"testing"

	"fearless/internal/core/model"
)

// makeCollectionDir writes a minimal bru collection on disk.
func makeCollectionDir(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("bruno.json", `{"version":"1","name":"`+name+`","type":"collection"}`)
	write("login.bru", "meta {\n  name: Login\n  type: http\n}\n\nget {\n  url: https://example.com\n  body: none\n}\n")
	return dir
}

// TestCopyIntoStorageIsLoadable: what is copied in must load back, which is the
// whole point — an imported collection has to survive a restart.
func TestCopyIntoStorageIsLoadable(t *testing.T) {
	srcRoot := t.TempDir()
	src := makeCollectionDir(t, srcRoot, "My API")
	storage := t.TempDir()

	dst, err := CopyIntoStorage(src, storage)
	if err != nil {
		t.Fatalf("CopyIntoStorage: %v", err)
	}
	if want := filepath.Join(storage, "My API"); dst != want {
		t.Errorf("dst = %q, want %q", dst, want)
	}

	col, err := Load(dst)
	if err != nil {
		t.Fatalf("Load(copy): %v", err)
	}
	if col.Name != "My API" {
		t.Errorf("name = %q, want My API", col.Name)
	}
	found := false
	for _, it := range col.Items {
		if it.Name == "Login" {
			found = true
		}
	}
	if !found {
		t.Errorf("copied collection has no Login request: %+v", col.Items)
	}

	// The source is untouched.
	if _, err := os.Stat(filepath.Join(src, "bruno.json")); err != nil {
		t.Errorf("source was modified: %v", err)
	}
}

// TestCopyIntoStorageNeverOverwrites: a second import of the same name must not
// clobber the first.
func TestCopyIntoStorageNeverOverwrites(t *testing.T) {
	src := makeCollectionDir(t, t.TempDir(), "Dup")
	storage := t.TempDir()

	first, err := CopyIntoStorage(src, storage)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CopyIntoStorage(src, storage)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("second import reused %q", first)
	}
	if want := filepath.Join(storage, "Dup (2)"); second != want {
		t.Errorf("second = %q, want %q", second, want)
	}
}

// TestCopyModelIntoStorageIsLoadable: a model-only import (cURL, Postman) has
// no source directory, so it must be written from the model and load back.
func TestCopyModelIntoStorageIsLoadable(t *testing.T) {
	storage := t.TempDir()
	url := "https://api.example.com/users"
	col := &model.Collection{
		Name:   "Imported",
		Config: model.CollectionConfig{Version: "1", Name: "Imported", Type: "collection"},
		Items: []*model.Item{{
			UID:  "r1",
			Type: model.ItemHTTPRequest,
			Name: "List users",
			Request: &model.Request{
				Name:   "List users",
				Method: "GET",
				URL:    url,
			},
		}},
	}

	dst, err := CopyModelIntoStorage(col, storage)
	if err != nil {
		t.Fatalf("CopyModelIntoStorage: %v", err)
	}

	loaded, err := Load(dst)
	if err != nil {
		t.Fatalf("Load(model copy): %v", err)
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("loaded %d items, want 1", len(loaded.Items))
	}
	if got := loaded.Items[0].Name; got != "List users" {
		t.Errorf("item name = %q, want List users", got)
	}
	if loaded.Items[0].Request == nil || loaded.Items[0].Request.URL != url {
		t.Errorf("request URL not preserved: %+v", loaded.Items[0].Request)
	}
}

// TestCopyIntoStorageRejectsMissingStorage: no storage folder is an error the
// caller reports, not a silent write to the wrong place.
func TestCopyIntoStorageRejectsMissingStorage(t *testing.T) {
	src := makeCollectionDir(t, t.TempDir(), "X")
	if _, err := CopyIntoStorage(src, ""); err == nil {
		t.Fatal("expected an error with no storage folder")
	}
}

// TestSanitizeDirName keeps a name usable as a folder.
func TestSanitizeDirName(t *testing.T) {
	for in, want := range map[string]string{
		"My API":  "My API",
		"a/b":     "a-b",
		"  . .. ": "Collection",
		"":        "Collection",
	} {
		if got := sanitizeDirName(in); got != want {
			t.Errorf("sanitizeDirName(%q) = %q, want %q", in, got, want)
		}
	}
}
