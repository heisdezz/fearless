package collection_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

func stringPtr(s string) *string { return &s }

// Helper to create a temporary test collection on disk
func createTestCollection(t *testing.T) (string, *model.Collection) {
	t.Helper()
	dir := t.TempDir()

	// 1. bruno.json
	brunoJSON := `{
  "version": "1",
  "name": "test-collection",
  "type": "collection",
  "ignore": ["node_modules", ".git"]
}`
	if err := os.WriteFile(filepath.Join(dir, "bruno.json"), []byte(brunoJSON), 0o644); err != nil {
		t.Fatalf("failed to write bruno.json: %v", err)
	}

	// 2. environments/Local.bru
	envsDir := filepath.Join(dir, "environments")
	if err := os.MkdirAll(envsDir, 0o755); err != nil {
		t.Fatalf("failed to create envs dir: %v", err)
	}
	localEnv := `vars {
  host: http://localhost:3000
}`
	if err := os.WriteFile(filepath.Join(envsDir, "Local.bru"), []byte(localEnv), 0o644); err != nil {
		t.Fatalf("failed to write Local.bru: %v", err)
	}

	// 3. Root request: get_users.bru
	req1 := `meta {
  name: Get Users
  type: http
  seq: 1
}

get {
  url: {{host}}/users
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "get_users.bru"), []byte(req1), 0o644); err != nil {
		t.Fatalf("failed to write get_users.bru: %v", err)
	}

	// 4. Subfolder: users/
	usersDir := filepath.Join(dir, "users")
	if err := os.MkdirAll(usersDir, 0o755); err != nil {
		t.Fatalf("failed to create users dir: %v", err)
	}
	folderBru := `meta {
  name: Users Folder
  seq: 2
}`
	if err := os.WriteFile(filepath.Join(usersDir, "folder.bru"), []byte(folderBru), 0o644); err != nil {
		t.Fatalf("failed to write folder.bru: %v", err)
	}

	req2 := `meta {
  name: Create User
  type: http
  seq: 1
}

post {
  url: {{host}}/users
  body: json
  auth: inherit
}

body:json {
  { "username": "alice" }
}`
	if err := os.WriteFile(filepath.Join(usersDir, "create_user.bru"), []byte(req2), 0o644); err != nil {
		t.Fatalf("failed to write create_user.bru: %v", err)
	}

	col, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("collection.Load failed: %v", err)
	}
	return dir, col
}

func TestLoadCorpusBasic(t *testing.T) {
	corpusPath := "../../../testdata/corpus/bru/basic"
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		t.Skip("corpus basic fixture not found, skipping")
	}

	col, err := collection.Load(corpusPath)
	if err != nil {
		t.Fatalf("Load corpus failed: %v", err)
	}

	if col.Name != "collection" {
		t.Errorf("col.Name = %q, want collection", col.Name)
	}
	if len(col.Items) == 0 {
		t.Fatalf("expected non-empty collection items")
	}

	// Verify items hierarchy
	var foundFolder bool
	var foundRequests int
	for _, it := range col.Items {
		if it.IsFolder() {
			foundFolder = true
			if len(it.Children) == 0 {
				t.Errorf("folder %s expected to have children", it.Name)
			}
		} else {
			foundRequests++
		}
	}
	if !foundFolder {
		t.Errorf("expected at least one folder in basic corpus")
	}
	if foundRequests < 2 {
		t.Errorf("expected at least 2 root requests in basic corpus, got %d", foundRequests)
	}
}

func TestLoadCorpusYAMLBasic(t *testing.T) {
	corpusPath := "../../../testdata/corpus/yml/basic"
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		t.Skip("corpus yml basic fixture not found, skipping")
	}

	col, err := collection.Load(corpusPath)
	if err != nil {
		t.Fatalf("Load YAML corpus failed: %v", err)
	}

	if col.Config.Format != model.FormatYML {
		t.Errorf("col.Config.Format = %q, want %q", col.Config.Format, model.FormatYML)
	}
	if len(col.Items) == 0 {
		t.Fatalf("expected non-empty collection items")
	}

	var foundFolder bool
	var foundRequests int
	for _, it := range col.Items {
		if it.IsFolder() {
			foundFolder = true
		} else {
			foundRequests++
		}
	}
	if !foundFolder {
		t.Errorf("expected at least one folder in yml basic corpus")
	}
	if foundRequests < 2 {
		t.Errorf("expected at least 2 root requests in yml basic corpus, got %d", foundRequests)
	}
}

func TestSortByNameThenSequence(t *testing.T) {
	seq1 := 1
	seq2 := 2
	seq5 := 5

	items := []*model.Item{
		{Name: "Gamma"},
		{Name: "Alpha", Seq: &seq2},
		{Name: "Beta", Seq: &seq1},
		{Name: "Delta", Seq: &seq5},
		{Name: "Epsilon"},
	}

	collection.SortItems(items)

	// Expected order:
	// 1. Beta (seq: 1 -> position 0)
	// 2. Alpha (seq: 2 -> position 1)
	// 3. Epsilon (alphabetical)
	// 4. Gamma (alphabetical)
	// 5. Delta (seq: 5 -> position 4)
	if items[0].Name != "Beta" {
		t.Errorf("items[0] = %s; want Beta", items[0].Name)
	}
	if items[1].Name != "Alpha" {
		t.Errorf("items[1] = %s; want Alpha", items[1].Name)
	}
	if items[2].Name != "Epsilon" {
		t.Errorf("items[2] = %s; want Epsilon", items[2].Name)
	}
	if items[3].Name != "Gamma" {
		t.Errorf("items[3] = %s; want Gamma", items[3].Name)
	}
	if items[4].Name != "Delta" {
		t.Errorf("items[4] = %s; want Delta", items[4].Name)
	}
}

func TestReconcilePreservesDrafts(t *testing.T) {
	dir, col := createTestCollection(t)

	// User opens and edits Get Users request creating an in-memory draft
	var getUsersItem *model.Item
	for _, it := range col.Items {
		if it.Name == "Get Users" {
			getUsersItem = it
			break
		}
	}
	if getUsersItem == nil {
		t.Fatalf("could not find Get Users item")
	}

	draftReq := getUsersItem.Request.Clone()
	draftReq.URL = "http://localhost:3000/users/modified-in-draft"
	getUsersItem.Draft = draftReq

	if !getUsersItem.Dirty() {
		t.Fatalf("expected item to be dirty with draft")
	}

	// External modification on disk: an external tool or git edits get_users.bru
	updatedOnDisk := `meta {
  name: Get Users
  type: http
  seq: 1
}

get {
  url: {{host}}/users/updated-externally
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "get_users.bru"), []byte(updatedOnDisk), 0o644); err != nil {
		t.Fatalf("failed to update file on disk: %v", err)
	}

	// Reload from disk
	fresh, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Reconcile
	reconciled := collection.Reconcile(col, fresh)

	// Assert:
	// 1. Saved Request was updated to disk content: http://localhost:3000/users/updated-externally
	// 2. Draft was NOT lost: http://localhost:3000/users/modified-in-draft
	reconciledItem := reconciled.FindItem(getUsersItem.UID)
	if reconciledItem == nil {
		t.Fatalf("reconciled item not found")
	}

	if reconciledItem.Request.URL != "{{host}}/users/updated-externally" {
		t.Errorf("saved Request URL = %q; want external update", reconciledItem.Request.URL)
	}

	if reconciledItem.Draft == nil {
		t.Fatalf("Draft was lost during reconcile!")
	}
	if reconciledItem.Draft.URL != "http://localhost:3000/users/modified-in-draft" {
		t.Errorf("Draft URL = %q; want preserved draft", reconciledItem.Draft.URL)
	}
	if !reconciled.Dirty {
		t.Errorf("expected reconciled collection to be marked dirty")
	}
}

func TestReconcileRemovesDeletedFiles(t *testing.T) {
	dir, col := createTestCollection(t)

	// Delete get_users.bru externally on disk
	if err := os.Remove(filepath.Join(dir, "get_users.bru")); err != nil {
		t.Fatalf("failed to remove get_users.bru: %v", err)
	}

	fresh, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	reconciled := collection.Reconcile(col, fresh)

	// Assert get_users is removed from tree
	for _, it := range reconciled.Items {
		if it.Name == "Get Users" {
			t.Fatalf("deleted item still present in reconciled tree!")
		}
	}
}

func TestReconcileRemapsRenamedFiles(t *testing.T) {
	dir, col := createTestCollection(t)

	// User had draft on Get Users
	var getUsersItem *model.Item
	for _, it := range col.Items {
		if it.Name == "Get Users" {
			getUsersItem = it
			break
		}
	}
	draft := getUsersItem.Request.Clone()
	draft.URL = "http://localhost:3000/draft-on-renamed"
	getUsersItem.Draft = draft

	// External rename: get_users.bru -> get_all_users.bru
	oldPath := filepath.Join(dir, "get_users.bru")
	newPath := filepath.Join(dir, "get_all_users.bru")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("failed to rename file: %v", err)
	}

	fresh, err := collection.Load(dir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	reconciled := collection.Reconcile(col, fresh)

	// Reconciled item should have preserved the draft
	var foundRenamed *model.Item
	for _, it := range reconciled.Items {
		if it.Name == "Get Users" {
			foundRenamed = it
			break
		}
	}
	if foundRenamed == nil {
		t.Fatalf("could not find renamed item")
	}
	if foundRenamed.Draft == nil || foundRenamed.Draft.URL != "http://localhost:3000/draft-on-renamed" {
		t.Errorf("draft was not remapped to renamed item: %+v", foundRenamed)
	}
}

func TestSaveItemPromotesDraft(t *testing.T) {
	_, col := createTestCollection(t)

	item := col.Items[0]
	draft := item.Request.Clone()
	draft.URL = "http://localhost:3000/new-promoted-url"
	item.Draft = draft

	if err := collection.SaveItem(col, item); err != nil {
		t.Fatalf("SaveItem failed: %v", err)
	}

	if item.Draft != nil {
		t.Errorf("item.Draft was not cleared after save")
	}
	if item.Request.URL != "http://localhost:3000/new-promoted-url" {
		t.Errorf("item.Request.URL = %q, want promoted url", item.Request.URL)
	}

	// Verify on disk
	reloaded, err := collection.Load(col.Path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	reloadedItem := reloaded.FindItem(item.UID)
	if reloadedItem.Request.URL != "http://localhost:3000/new-promoted-url" {
		t.Errorf("reloaded URL = %q; want saved content", reloadedItem.Request.URL)
	}
}

func TestSaveFolderItem(t *testing.T) {
	_, col := createTestCollection(t)

	var folderItem *model.Item
	for _, it := range col.Items {
		if it.IsFolder() {
			folderItem = it
			break
		}
	}
	if folderItem == nil {
		t.Fatalf("folder item not found")
	}

	folderItem.Name = "Renamed Folder"
	if err := collection.SaveItem(col, folderItem); err != nil {
		t.Fatalf("SaveItem on folder failed: %v", err)
	}

	reloaded, err := collection.Load(col.Path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	reloadedFolder := reloaded.FindItem(folderItem.UID)
	if reloadedFolder.Name != "Renamed Folder" {
		t.Errorf("reloaded folder name = %q; want Renamed Folder", reloadedFolder.Name)
	}
}

func TestSaveEnvironmentAndConfig(t *testing.T) {
	_, col := createTestCollection(t)

	// Save environment
	env := col.Environments[0]
	env.Variables = append(env.Variables, model.Variable{
		Name:    stringPtr("apiKey"),
		Value:   "secret-123",
		Enabled: true,
	})

	if err := collection.SaveEnvironment(col, env); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	// Save collection config
	col.Config.Name = "Renamed Collection"
	if err := collection.SaveCollectionConfig(col); err != nil {
		t.Fatalf("SaveCollectionConfig failed: %v", err)
	}

	reloaded, err := collection.Load(col.Path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if reloaded.Name != "Renamed Collection" {
		t.Errorf("reloaded Name = %q, want Renamed Collection", reloaded.Name)
	}
	if len(reloaded.Environments) == 0 {
		t.Fatalf("reloaded environments empty")
	}
	val, ok := reloaded.Environments[0].VariableValue("apiKey")
	if !ok || val != "secret-123" {
		t.Errorf("environment variable apiKey = %v, want secret-123", val)
	}
}

func TestSelfWriteSuppressionAndWatcher(t *testing.T) {
	dir, col := createTestCollection(t)

	var changeCount int32
	watcher, err := collection.NewWatcher(col, collection.WatcherOptions{
		DebounceDelay: 20 * time.Millisecond,
		OnChange: func(reloaded *model.Collection, err error) {
			atomic.AddInt32(&changeCount, 1)
		},
	})
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer watcher.Close()

	if watcher.Current() == nil {
		t.Errorf("watcher.Current() returned nil")
	}

	// 1. Self-write via SaveItem: MUST BE SUPPRESSED!
	item := col.Items[0]
	item.Request.URL = "http://localhost:3000/self-write"
	if err := collection.SaveItem(col, item); err != nil {
		t.Fatalf("SaveItem failed: %v", err)
	}

	// Wait beyond debounce delay
	time.Sleep(100 * time.Millisecond)

	if count := atomic.LoadInt32(&changeCount); count != 0 {
		t.Fatalf("Self-write was NOT suppressed! changeCount = %d, want 0", count)
	}

	// 2. External write: MUST TRIGGER OnChange!
	externalReq := `meta {
  name: External Request
  type: http
  seq: 99
}

get {
  url: http://external.test/api
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "external.bru"), []byte(externalReq), 0o644); err != nil {
		t.Fatalf("failed to write external file: %v", err)
	}

	// Wait for debounce and notification
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&changeCount) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if count := atomic.LoadInt32(&changeCount); count == 0 {
		t.Fatalf("External write did not trigger OnChange! changeCount = 0")
	}
}

func TestDeleteItem(t *testing.T) {
	_, col := createTestCollection(t)

	item := col.Items[0]
	path := item.Path

	if err := collection.DeleteItem(col, item); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists on disk after DeleteItem: %s", path)
	}

	if col.FindItem(item.UID) != nil {
		t.Errorf("item still found in collection after DeleteItem")
	}
}

func TestResequenceItems(t *testing.T) {
	_, col := createTestCollection(t)

	items := col.Items
	if err := collection.ResequenceItems(col, items); err != nil {
		t.Fatalf("ResequenceItems failed: %v", err)
	}

	for i, it := range items {
		expected := i + 1
		if it.Seq == nil || *it.Seq != expected {
			t.Errorf("item %s seq = %v; want %d", it.Name, it.Seq, expected)
		}
	}
}

func TestFindItemByPath(t *testing.T) {
	_, col := createTestCollection(t)

	targetItem := col.Items[0]
	found := collection.FindItemByPath(col, targetItem.Path)
	if found == nil || found.UID != targetItem.UID {
		t.Errorf("FindItemByPath failed, got: %+v", found)
	}

	if collection.FindItemByPath(col, "/nonexistent/path") != nil {
		t.Errorf("expected nil for nonexistent path")
	}
	if collection.FindItemByPath(nil, targetItem.Path) != nil {
		t.Errorf("expected nil for nil collection")
	}
}

func TestUIDAndHelperEdgeCases(t *testing.T) {
	if uid := collection.GenerateUID(""); uid != "" {
		t.Errorf("GenerateUID(\"\") = %q, want empty", uid)
	}
	if uid := collection.GenerateUID("."); uid != "" {
		t.Errorf("GenerateUID(\".\") = %q, want empty", uid)
	}

	// Reconcile nil checks
	dummyCol := &model.Collection{Name: "dummy"}
	if collection.Reconcile(nil, dummyCol) != dummyCol {
		t.Errorf("expected dummyCol on nil current")
	}
	if collection.Reconcile(dummyCol, nil) != dummyCol {
		t.Errorf("expected dummyCol on nil incoming")
	}

	// Save nil checks
	if err := collection.SaveItem(nil, nil); err == nil {
		t.Errorf("expected error on nil SaveItem")
	}
	if err := collection.SaveEnvironment(nil, nil); err == nil {
		t.Errorf("expected error on nil SaveEnvironment")
	}
	if err := collection.SaveCollectionConfig(nil); err == nil {
		t.Errorf("expected error on nil SaveCollectionConfig")
	}
	if err := collection.DeleteItem(nil, nil); err == nil {
		t.Errorf("expected error on nil DeleteItem")
	}
	if _, err := collection.NewWatcher(nil, collection.WatcherOptions{}); err == nil {
		t.Errorf("expected error on nil NewWatcher")
	}
}
