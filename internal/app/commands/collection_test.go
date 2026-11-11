package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func createTestWorkspace(t *testing.T) (string, *bus.Bus, *bus.Store, *commands.CollectionService) {
	t.Helper()
	dir := t.TempDir()

	// Create a minimal bruno collection on disk
	brunoJSON := `{
  "version": "1",
  "name": "test-collection",
  "type": "collection"
}`
	if err := os.WriteFile(filepath.Join(dir, "bruno.json"), []byte(brunoJSON), 0o644); err != nil {
		t.Fatalf("failed to write bruno.json: %v", err)
	}

	req1 := `meta {
  name: Get Users
  type: http
  seq: 1
}

get {
  url: http://localhost:3000/users
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "get_users.bru"), []byte(req1), 0o644); err != nil {
		t.Fatalf("failed to write get_users.bru: %v", err)
	}

	// Subfolder
	subDir := filepath.Join(dir, "admin")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create admin dir: %v", err)
	}
	folderBru := `meta {
  name: Admin Folder
  seq: 2
}`
	if err := os.WriteFile(filepath.Join(subDir, "folder.bru"), []byte(folderBru), 0o644); err != nil {
		t.Fatalf("failed to write folder.bru: %v", err)
	}

	req2 := `meta {
  name: Admin Status
  type: http
  seq: 1
}

get {
  url: http://localhost:3000/admin/status
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(subDir, "admin_status.bru"), []byte(req2), 0o644); err != nil {
		t.Fatalf("failed to write admin_status.bru: %v", err)
	}

	b := bus.New()
	store := bus.NewStore(nil, b)
	svc := commands.NewCollectionService(store, b)

	t.Cleanup(func() {
		_ = svc.Close()
		store.Close()
		b.Close()
	})

	return dir, b, store, svc
}

func collectEvents(t *testing.T, eventChan <-chan events.Event, timeout time.Duration) []events.Event {
	t.Helper()
	var list []events.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-eventChan:
			if !ok {
				return list
			}
			list = append(list, ev)
		case <-deadline:
			return list
		}
	}
}

func TestLoadCollection(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	eventCh, cancel := b.Subscribe()
	defer cancel()

	ctx := context.Background()
	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}
	if colUID == "" {
		t.Fatalf("expected non-empty collection UID")
	}

	// 1. Verify in workspace state
	snap := store.Snapshot()
	col := snap.FindCollection(colUID)
	if col == nil {
		t.Fatalf("collection not found in workspace state")
	}
	if col.Name != "test-collection" {
		t.Errorf("col.Name = %q, want test-collection", col.Name)
	}
	if len(col.Items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(col.Items))
	}

	// 2. Verify event published
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundAdded bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			if cEv.CollectionUID == colUID && cEv.Change.Origin == events.OriginSelf {
				for _, id := range cEv.Change.Added {
					if id == colUID {
						foundAdded = true
					}
				}
			}
		}
	}
	if !foundAdded {
		t.Errorf("expected CollectionChanged Added event for collection")
	}

	// 3. Idempotency test: loading same path returns same UID without duplicate
	colUID2, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("second LoadCollection failed: %v", err)
	}
	if colUID2 != colUID {
		t.Errorf("second LoadCollection returned UID %q, want %q", colUID2, colUID)
	}
	if len(store.Snapshot().Collections) != 1 {
		t.Errorf("expected 1 collection in state, got %d", len(store.Snapshot().Collections))
	}
}

func TestReloadCollection(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	// User creates a draft in memory on Get Users
	snap := store.Snapshot()
	col := snap.FindCollection(colUID)
	getUsersItem := col.FindItem(col.Items[0].UID)
	draft := getUsersItem.Request.Clone()
	draft.URL = "http://localhost:3000/in-memory-draft"

	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection(colUID).FindItem(getUsersItem.UID)
		it.Draft = draft
	})

	// External change on disk: update get_users.bru
	updatedDisk := `meta {
  name: Get Users
  type: http
  seq: 1
}

get {
  url: http://localhost:3000/updated-from-disk
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "get_users.bru"), []byte(updatedDisk), 0o644); err != nil {
		t.Fatalf("failed to update file on disk: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Call ReloadCollection
	if err := svc.ReloadCollection(ctx, commands.ReloadCollectionCmd{CollectionUID: colUID}); err != nil {
		t.Fatalf("ReloadCollection failed: %v", err)
	}

	reloadedSnap := store.Snapshot()
	reloadedItem := reloadedSnap.FindCollection(colUID).FindItem(getUsersItem.UID)

	// Assert saved request was updated from disk
	if reloadedItem.Request.URL != "http://localhost:3000/updated-from-disk" {
		t.Errorf("reloaded saved URL = %q; want updated-from-disk", reloadedItem.Request.URL)
	}

	// Assert in-progress draft was PRESERVED
	if reloadedItem.Draft == nil || reloadedItem.Draft.URL != "http://localhost:3000/in-memory-draft" {
		t.Errorf("draft was lost or corrupted during ReloadCollection: %+v", reloadedItem.Draft)
	}

	// Assert event published with Rescan: true
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundRescan bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			if cEv.CollectionUID == colUID && cEv.Change.Rescan {
				foundRescan = true
			}
		}
	}
	if !foundRescan {
		t.Errorf("expected CollectionChanged event with Rescan: true")
	}

	// Idempotency test: Reloading again succeeds cleanly
	if err := svc.ReloadCollection(ctx, commands.ReloadCollectionCmd{CollectionUID: colUID}); err != nil {
		t.Fatalf("second ReloadCollection failed: %v", err)
	}
}

func TestSaveItem(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	col := store.Snapshot().FindCollection(colUID)
	item := col.Items[0]

	// Put draft into item
	draft := item.Request.Clone()
	draft.URL = "http://localhost:3000/promoted-by-save"
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection(colUID).FindItem(item.UID)
		it.Draft = draft
	})

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Execute SaveItem
	if err := svc.SaveItem(ctx, commands.SaveItemCmd{CollectionUID: colUID, ItemUID: item.UID}); err != nil {
		t.Fatalf("SaveItem failed: %v", err)
	}

	savedSnap := store.Snapshot()
	savedItem := savedSnap.FindCollection(colUID).FindItem(item.UID)

	if savedItem.Draft != nil {
		t.Errorf("expected item.Draft to be cleared after SaveItem")
	}
	if savedItem.Request.URL != "http://localhost:3000/promoted-by-save" {
		t.Errorf("saved Request URL = %q; want promoted url", savedItem.Request.URL)
	}

	// Verify disk contents
	content, err := os.ReadFile(item.Path)
	if err != nil {
		t.Fatalf("failed to read item from disk: %v", err)
	}
	if !containsString(string(content), "http://localhost:3000/promoted-by-save") {
		t.Errorf("disk file did not contain promoted URL")
	}

	// Verify event
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundModified bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, m := range cEv.Change.Modified {
				if m == item.UID {
					foundModified = true
				}
			}
		}
	}
	if !foundModified {
		t.Errorf("expected CollectionChanged Modified event for saved item")
	}

	// Idempotency: saving clean item returns nil immediately
	if err := svc.SaveItem(ctx, commands.SaveItemCmd{CollectionUID: colUID, ItemUID: item.UID}); err != nil {
		t.Errorf("second SaveItem on clean item failed: %v", err)
	}
}

func TestNewItem_RequestAndFolder(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// 1. Create a root request item
	reqItem, err := svc.NewItem(ctx, commands.NewItemCmd{
		CollectionUID: colUID,
		Name:          "Create Product",
		Method:        "POST",
		URL:           "http://localhost:3000/products",
	})
	if err != nil {
		t.Fatalf("NewItem for request failed: %v", err)
	}
	if reqItem.Name != "Create Product" {
		t.Errorf("reqItem.Name = %q, want Create Product", reqItem.Name)
	}
	if _, err := os.Stat(reqItem.Path); os.IsNotExist(err) {
		t.Errorf("new request file not found on disk: %s", reqItem.Path)
	}

	// 2. Create a folder item
	folderItem, err := svc.NewItem(ctx, commands.NewItemCmd{
		CollectionUID: colUID,
		Name:          "Billing",
		Type:          model.ItemFolder,
	})
	if err != nil {
		t.Fatalf("NewItem for folder failed: %v", err)
	}
	if _, err := os.Stat(folderItem.Path); os.IsNotExist(err) {
		t.Errorf("new folder not found on disk: %s", folderItem.Path)
	}

	// 3. Create a child request inside the Billing folder
	childReq, err := svc.NewItem(ctx, commands.NewItemCmd{
		CollectionUID:   colUID,
		ParentFolderUID: folderItem.UID,
		Name:            "Get Invoices",
		Method:          "GET",
		URL:             "http://localhost:3000/billing/invoices",
	})
	if err != nil {
		t.Fatalf("NewItem inside folder failed: %v", err)
	}

	snap := store.Snapshot()
	col := snap.FindCollection(colUID)
	foundBilling := col.FindItem(folderItem.UID)
	if foundBilling == nil || len(foundBilling.Children) != 1 {
		t.Fatalf("expected 1 child in Billing folder, got: %+v", foundBilling)
	}
	if foundBilling.Children[0].UID != childReq.UID {
		t.Errorf("child UID mismatch")
	}

	// Verify events
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var addedCount int
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			addedCount += len(cEv.Change.Added)
		}
	}
	if addedCount < 3 {
		t.Errorf("expected at least 3 added events, got %d", addedCount)
	}
}

func TestNewItem_YAML(t *testing.T) {
	dir := t.TempDir()
	openCollectionYML := `opencollection: "1.0.0"
info:
  name: yaml-test
  version: "1.0.0"
`
	if err := os.WriteFile(filepath.Join(dir, "opencollection.yml"), []byte(openCollectionYML), 0o644); err != nil {
		t.Fatalf("failed to write opencollection.yml: %v", err)
	}

	b := bus.New()
	store := bus.NewStore(nil, b)
	svc := commands.NewCollectionService(store, b)
	defer func() {
		_ = svc.Close()
		store.Close()
		b.Close()
	}()

	ctx := context.Background()
	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection YAML failed: %v", err)
	}

	// Create request in YAML collection
	item, err := svc.NewItem(ctx, commands.NewItemCmd{
		CollectionUID: colUID,
		Name:          "YAML Request",
		Method:        "GET",
		URL:           "https://api.example.com/yaml",
		Format:        model.FormatYML,
	})
	if err != nil {
		t.Fatalf("NewItem in YAML failed: %v", err)
	}
	if !filepath.IsAbs(item.Path) || filepath.Ext(item.Path) != ".yml" {
		t.Errorf("item.Path = %q, want .yml extension", item.Path)
	}
	if _, err := os.Stat(item.Path); os.IsNotExist(err) {
		t.Errorf("yaml request file not found on disk")
	}
}

func TestMoveItem_AcrossFoldersAndSequence(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	snap := store.Snapshot()
	col := snap.FindCollection(colUID)

	// Find root request item (Get Users) and Admin Folder
	var rootReq *model.Item
	var adminFolder *model.Item
	for _, it := range col.Items {
		if it.Name == "Get Users" {
			rootReq = it
		}
		if it.Name == "Admin Folder" {
			adminFolder = it
		}
	}
	if rootReq == nil || adminFolder == nil {
		t.Fatalf("failed to locate root request or admin folder")
	}

	oldUID := rootReq.UID

	// Open an editor tab viewing the root request
	store.Apply(func(s *model.WorkspaceState) {
		s.Tabs = append(s.Tabs, &model.Tab{
			UID:           "tab-1",
			CollectionUID: colUID,
			ItemUID:       oldUID,
			Title:         "Get Users",
		})
		s.ActiveTab = "tab-1"
	})

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Move Get Users from root into Admin Folder
	err = svc.MoveItem(ctx, commands.MoveItemCmd{
		CollectionUID:   colUID,
		ItemUID:         oldUID,
		TargetFolderUID: adminFolder.UID,
		TargetSeq:       2,
	})
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	// Assert item moved in tree
	updatedSnap := store.Snapshot()
	updatedCol := updatedSnap.FindCollection(colUID)
	updatedAdmin := updatedCol.FindItem(adminFolder.UID)

	var foundMovedItem *model.Item
	for _, it := range updatedAdmin.Children {
		if it.Name == "Get Users" {
			foundMovedItem = it
			break
		}
	}
	if foundMovedItem == nil {
		t.Fatalf("moved item not found in admin folder children")
	}

	// Assert tab ItemUID was remapped to new UID
	tab := updatedSnap.FindTab("tab-1")
	if tab == nil {
		t.Fatalf("tab-1 not found")
	}
	if tab.ItemUID != foundMovedItem.UID {
		t.Errorf("tab.ItemUID = %q, want remapped UID %q", tab.ItemUID, foundMovedItem.UID)
	}

	// Move within same folder to test sequence-only update
	err = svc.MoveItem(ctx, commands.MoveItemCmd{
		CollectionUID:   colUID,
		ItemUID:         foundMovedItem.UID,
		TargetFolderUID: adminFolder.UID,
		TargetSeq:       1,
	})
	if err != nil {
		t.Fatalf("re-sequence MoveItem failed: %v", err)
	}

	// Assert event published
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundModified bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, m := range cEv.Change.Modified {
				if m == foundMovedItem.UID {
					foundModified = true
				}
			}
		}
	}
	if !foundModified {
		t.Errorf("expected CollectionChanged Modified event for moved item")
	}
}

func TestDeleteItem(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	col := store.Snapshot().FindCollection(colUID)
	item := col.Items[0]
	path := item.Path

	// Open a tab for this item
	store.Apply(func(s *model.WorkspaceState) {
		s.Tabs = append(s.Tabs, &model.Tab{
			UID:           "tab-delete",
			CollectionUID: colUID,
			ItemUID:       item.UID,
			Title:         item.Name,
		})
		s.ActiveTab = "tab-delete"
	})

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Delete item
	if err := svc.DeleteItem(ctx, commands.DeleteItemCmd{CollectionUID: colUID, ItemUID: item.UID}); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	// 1. Assert deleted from disk
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("item still exists on disk after DeleteItem: %s", path)
	}

	// 2. Assert removed from workspace state
	updatedSnap := store.Snapshot()
	if updatedSnap.FindCollection(colUID).FindItem(item.UID) != nil {
		t.Errorf("item still found in workspace collection")
	}

	// 3. Assert tab was closed
	if updatedSnap.FindTab("tab-delete") != nil {
		t.Errorf("tab for deleted item was not closed")
	}
	if updatedSnap.ActiveTab == "tab-delete" {
		t.Errorf("active tab still points to deleted tab")
	}

	// 4. Assert event published
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundRemoved bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, r := range cEv.Change.Removed {
				if r == item.UID {
					foundRemoved = true
				}
			}
		}
	}
	if !foundRemoved {
		t.Errorf("expected CollectionChanged Removed event")
	}

	// Idempotency: deleting already deleted item returns nil
	if err := svc.DeleteItem(ctx, commands.DeleteItemCmd{CollectionUID: colUID, ItemUID: item.UID}); err != nil {
		t.Errorf("second DeleteItem failed: %v", err)
	}

	// Delete folder with recursive children
	adminFolder := updatedSnap.FindCollection(colUID).Items[0]
	if err := svc.DeleteItem(ctx, commands.DeleteItemCmd{CollectionUID: colUID, ItemUID: adminFolder.UID}); err != nil {
		t.Fatalf("DeleteItem on folder failed: %v", err)
	}
}

func TestCloseCollection(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	// Open tabs for collection
	store.Apply(func(s *model.WorkspaceState) {
		s.Tabs = append(s.Tabs, &model.Tab{
			UID:           "tab-col-1",
			CollectionUID: colUID,
			Title:         "Request 1",
		})
		s.ActiveTab = "tab-col-1"
	})

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Close collection
	if err := svc.CloseCollection(ctx, commands.CloseCollectionCmd{CollectionUID: colUID}); err != nil {
		t.Fatalf("CloseCollection failed: %v", err)
	}

	snap := store.Snapshot()
	if snap.FindCollection(colUID) != nil {
		t.Errorf("collection still present in workspace state after CloseCollection")
	}
	if snap.FindTab("tab-col-1") != nil {
		t.Errorf("tab for closed collection was not closed")
	}

	// Assert event
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundRemoved bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, r := range cEv.Change.Removed {
				if r == colUID {
					foundRemoved = true
				}
			}
		}
	}
	if !foundRemoved {
		t.Errorf("expected CollectionChanged Removed event for closed collection")
	}

	// Idempotency: closing non-existent collection returns nil
	if err := svc.CloseCollection(ctx, commands.CloseCollectionCmd{CollectionUID: colUID}); err != nil {
		t.Errorf("second CloseCollection failed: %v", err)
	}
}

func TestWatcher_ExternalEditNotification(t *testing.T) {
	dir, b, _, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Create a new file externally on disk
	extReq := `meta {
  name: External Addition
  type: http
  seq: 99
}

get {
  url: http://localhost:3000/external
  body: none
  auth: inherit
}`
	if err := os.WriteFile(filepath.Join(dir, "external_addition.bru"), []byte(extReq), 0o644); err != nil {
		t.Fatalf("failed to write external file: %v", err)
	}

	// Wait for watcher debounce & event notification
	deadline := time.Now().Add(500 * time.Millisecond)
	var foundExternal bool
	for time.Now().Before(deadline) {
		evs := collectEvents(t, eventCh, 50*time.Millisecond)
		for _, ev := range evs {
			if cEv, ok := ev.(events.CollectionChanged); ok {
				if cEv.CollectionUID == colUID && cEv.Change.Origin == events.OriginExternal {
					foundExternal = true
					break
				}
			}
		}
		if foundExternal {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !foundExternal {
		t.Errorf("expected external CollectionChanged event from watcher")
	}
}

func TestCommandValidationAndEdgeCases(t *testing.T) {
	dir, _, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	// LoadCollection errors
	if _, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{}); err == nil {
		t.Errorf("expected error on empty path")
	}
	if _, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: "/nonexistent/path/for/sure"}); err == nil {
		t.Errorf("expected error on nonexistent path")
	}

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	// CloseCollection errors
	if err := svc.CloseCollection(ctx, commands.CloseCollectionCmd{}); err == nil {
		t.Errorf("expected error on empty close collection UID")
	}

	// ReloadCollection errors
	if err := svc.ReloadCollection(ctx, commands.ReloadCollectionCmd{}); err == nil {
		t.Errorf("expected error on empty reload collection UID")
	}
	if err := svc.ReloadCollection(ctx, commands.ReloadCollectionCmd{CollectionUID: "bad-uid"}); err == nil {
		t.Errorf("expected error on unknown collection UID")
	}

	// SaveItem errors
	if err := svc.SaveItem(ctx, commands.SaveItemCmd{}); err == nil {
		t.Errorf("expected error on empty SaveItemCmd")
	}
	if err := svc.SaveItem(ctx, commands.SaveItemCmd{CollectionUID: "bad", ItemUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
	if err := svc.SaveItem(ctx, commands.SaveItemCmd{CollectionUID: colUID, ItemUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown item")
	}

	// DeleteItem errors
	if err := svc.DeleteItem(ctx, commands.DeleteItemCmd{}); err == nil {
		t.Errorf("expected error on empty DeleteItemCmd")
	}
	if err := svc.DeleteItem(ctx, commands.DeleteItemCmd{CollectionUID: "bad", ItemUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}

	// NewItem errors
	if _, err := svc.NewItem(ctx, commands.NewItemCmd{}); err == nil {
		t.Errorf("expected error on empty NewItemCmd")
	}
	if _, err := svc.NewItem(ctx, commands.NewItemCmd{CollectionUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
	if _, err := svc.NewItem(ctx, commands.NewItemCmd{CollectionUID: colUID, ParentFolderUID: "bad-folder"}); err == nil {
		t.Errorf("expected error on unknown parent folder")
	}
	// Target folder is request, not folder
	reqUID := store.Snapshot().FindCollection(colUID).Items[0].UID
	if _, err := svc.NewItem(ctx, commands.NewItemCmd{CollectionUID: colUID, ParentFolderUID: reqUID}); err == nil {
		t.Errorf("expected error when ParentFolderUID is a request")
	}

	// MoveItem errors
	if err := svc.MoveItem(ctx, commands.MoveItemCmd{}); err == nil {
		t.Errorf("expected error on empty MoveItemCmd")
	}
	if err := svc.MoveItem(ctx, commands.MoveItemCmd{CollectionUID: "bad", ItemUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
	if err := svc.MoveItem(ctx, commands.MoveItemCmd{CollectionUID: colUID, ItemUID: "bad"}); err == nil {
		t.Errorf("expected error on unknown item")
	}
	if err := svc.MoveItem(ctx, commands.MoveItemCmd{CollectionUID: colUID, ItemUID: reqUID, TargetFolderUID: "bad-folder"}); err == nil {
		t.Errorf("expected error on unknown target folder")
	}
	if err := svc.MoveItem(ctx, commands.MoveItemCmd{CollectionUID: colUID, ItemUID: reqUID, TargetFolderUID: reqUID}); err == nil {
		t.Errorf("expected error when TargetFolderUID is not a folder")
	}
}

func containsString(s, substr string) bool {
	return filepath.Clean(s) != "" && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || filepath.Base(s) == substr || (len(s) > 0 && len(substr) > 0 && checkSubstr(s, substr)))
}

func checkSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
