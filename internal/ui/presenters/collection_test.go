package presenters

import (
	"context"
	"errors"
	"testing"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

type fakeColLoader struct {
	path string
	err  error
}

func (f *fakeColLoader) Load(ctx context.Context, path string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.path = path
	return "col-loaded-uid", nil
}

type fakeItemCreator struct {
	createdItem *model.Item
	err         error
}

func (f *fakeItemCreator) CreateItem(ctx context.Context, colUID, parentFolderUID, name string, itemType model.ItemType, method, url string) (*model.Item, error) {
	if f.err != nil {
		return nil, f.err
	}
	it := &model.Item{
		UID:  "new-item-uid",
		Name: name,
		Type: itemType,
	}
	if itemType == model.ItemHTTPRequest {
		it.Request = &model.Request{
			Method: method,
			URL:    url,
		}
	}
	f.createdItem = it
	return it, nil
}

type fakeItemDeleter struct {
	deletedColUID  string
	deletedItemUID string
	err            error
}

func (f *fakeItemDeleter) DeleteItem(ctx context.Context, colUID, itemUID string) error {
	if f.err != nil {
		return f.err
	}
	f.deletedColUID = colUID
	f.deletedItemUID = itemUID
	return nil
}

type fakeItemMover struct {
	colUID    string
	itemUID   string
	targetUID string
	targetSeq *int
	err       error
}

func (f *fakeItemMover) MoveItem(ctx context.Context, colUID, itemUID, targetFolderUID string, targetSeq *int) error {
	if f.err != nil {
		return f.err
	}
	f.colUID = colUID
	f.itemUID = itemUID
	f.targetUID = targetFolderUID
	f.targetSeq = targetSeq
	return nil
}

func buildTestCollection() *model.Collection {
	req1 := &model.Item{
		UID:  "req-1",
		Name: "List Users",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/users",
		},
	}
	req2 := &model.Item{
		UID:  "req-2",
		Name: "Create Order",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "POST",
			URL:    "https://api.example.com/orders",
		},
	}
	subFolder := &model.Item{
		UID:      "subfolder-1",
		Name:     "Admin Subfolder",
		Type:     model.ItemFolder,
		Children: []*model.Item{req2},
	}
	parentFolder := &model.Item{
		UID:      "folder-1",
		Name:     "Admin API",
		Type:     model.ItemFolder,
		Children: []*model.Item{subFolder},
	}

	return &model.Collection{
		UID:   "col-1",
		Name:  "Example API",
		Path:  "/path/to/col-1",
		Items: []*model.Item{req1, parentFolder},
	}
}

func TestCollectionPresenter_BasicsAndExpansion(t *testing.T) {
	loader := &fakeColLoader{}
	creator := &fakeItemCreator{}
	deleter := &fakeItemDeleter{}
	mover := &fakeItemMover{}

	p := NewCollectionPresenter(loader, creator, deleter, mover)

	changed := 0
	p.OnChanged(func() {
		changed++
	})

	col := buildTestCollection()
	p.SetCollections([]*model.Collection{col})

	if len(p.Collections()) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(p.Collections()))
	}
	if p.FindCollection("col-1") != col {
		t.Errorf("FindCollection mismatch")
	}
	if p.FindCollection("nonexistent") != nil {
		t.Errorf("expected nil for nonexistent collection")
	}

	it, c := p.FindItem("req-2")
	if it == nil || it.Name != "Create Order" || c != col {
		t.Errorf("FindItem req-2 failed: %v, %v", it, c)
	}

	// Collection is expanded by default
	if !p.IsExpanded("col-1") {
		t.Errorf("expected col-1 to be expanded initially")
	}

	p.ToggleExpanded("col-1")
	if p.IsExpanded("col-1") {
		t.Errorf("expected col-1 to be collapsed after toggle")
	}

	p.SetExpanded("col-1", true)
	if !p.IsExpanded("col-1") {
		t.Errorf("expected col-1 to be expanded after SetExpanded")
	}

	// Selection
	p.Select("req-1")
	if p.SelectedUID() != "req-1" {
		t.Errorf("expected selected UID req-1, got %s", p.SelectedUID())
	}
	p.ClearSelection()
	if p.SelectedUID() != "" {
		t.Errorf("expected empty selection after ClearSelection")
	}
}

func TestCollectionPresenter_VisibleNodes(t *testing.T) {
	p := NewCollectionPresenter(nil, nil, nil, nil)
	col := buildTestCollection()
	p.SetCollections([]*model.Collection{col})

	// Initial: col-1 expanded, folder-1 collapsed.
	// Visible nodes: col-1 (level 0), req-1 (level 1), folder-1 (level 1)
	nodes := p.VisibleNodes()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 visible nodes, got %d", len(nodes))
	}
	if nodes[0].UID != "col-1" || nodes[1].UID != "req-1" || nodes[2].UID != "folder-1" {
		t.Errorf("unexpected nodes order: %s, %s, %s", nodes[0].UID, nodes[1].UID, nodes[2].UID)
	}

	// Expand folder-1
	p.SetExpanded("folder-1", true)
	// Visible: col-1, req-1, folder-1, subfolder-1
	nodes = p.VisibleNodes()
	if len(nodes) != 4 {
		t.Fatalf("expected 4 visible nodes, got %d", len(nodes))
	}

	// Expand subfolder-1
	p.SetExpanded("subfolder-1", true)
	// Visible: col-1, req-1, folder-1, subfolder-1, req-2
	nodes = p.VisibleNodes()
	if len(nodes) != 5 {
		t.Fatalf("expected 5 visible nodes, got %d", len(nodes))
	}

	// Filter query: "orders" (matches req-2)
	p.SetFilter("orders")
	if p.Filter() != "orders" {
		t.Errorf("Filter mismatch: %s", p.Filter())
	}
	nodes = p.VisibleNodes()
	// Should auto-expand ancestors: col-1, folder-1, subfolder-1, req-2
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes when filtered by 'orders', got %d", len(nodes))
	}
	if nodes[len(nodes)-1].UID != "req-2" {
		t.Errorf("expected last node to be req-2, got %s", nodes[len(nodes)-1].UID)
	}

	// Filter by method: "GET" (matches req-1)
	p.SetFilter("GET")
	nodes = p.VisibleNodes()
	if len(nodes) != 2 { // col-1, req-1
		t.Fatalf("expected 2 nodes when filtered by 'GET', got %d", len(nodes))
	}

	// Filter with no matches
	p.SetFilter("nonexistent_search_term")
	nodes = p.VisibleNodes()
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes when filtered with no match, got %d", len(nodes))
	}
}

func TestCollectionPresenter_CanMove(t *testing.T) {
	p := NewCollectionPresenter(nil, nil, nil, nil)
	col := buildTestCollection()
	p.SetCollections([]*model.Collection{col})

	// Valid moves
	if !p.CanMove("col-1", "req-1", "folder-1") {
		t.Error("expected valid move req-1 into folder-1")
	}
	if !p.CanMove("col-1", "req-2", "") {
		t.Error("expected valid move req-2 into root")
	}

	// Non-existent collection or item
	if p.CanMove("nonexistent", "req-1", "folder-1") {
		t.Error("expected invalid move for nonexistent collection")
	}
	if p.CanMove("col-1", "nonexistent", "folder-1") {
		t.Error("expected invalid move for nonexistent item")
	}

	// Moving to nonexistent folder
	if p.CanMove("col-1", "req-1", "nonexistent-folder") {
		t.Error("expected invalid move to nonexistent folder")
	}

	// Moving to a non-folder target
	if p.CanMove("col-1", "req-1", "req-2") {
		t.Error("expected invalid move targeting a request item")
	}

	// Moving folder into itself
	if p.CanMove("col-1", "folder-1", "folder-1") {
		t.Error("expected invalid move of folder into itself")
	}

	// Moving folder into its own descendant (folder-1 into subfolder-1)
	if p.CanMove("col-1", "folder-1", "subfolder-1") {
		t.Error("expected invalid move of folder into descendant")
	}
}

func TestCollectionPresenter_ActionsAndEvents(t *testing.T) {
	loader := &fakeColLoader{}
	creator := &fakeItemCreator{}
	deleter := &fakeItemDeleter{}
	mover := &fakeItemMover{}

	p := NewCollectionPresenter(loader, creator, deleter, mover)
	col := buildTestCollection()
	p.SetCollections([]*model.Collection{col})

	// CreateRequest
	item, err := p.CreateRequest(context.Background(), "col-1", "folder-1", "New Req", "POST", "https://api.example.com")
	if err != nil || item == nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}
	if !p.IsExpanded("folder-1") {
		t.Error("expected parent folder to expand on item creation")
	}

	// CreateFolder
	folderItem, err := p.CreateFolder(context.Background(), "col-1", "folder-1", "New Sub")
	if err != nil || folderItem == nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// MoveItem
	seq := 2
	if err := p.MoveItem(context.Background(), "col-1", "req-1", "folder-1", &seq); err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}
	if mover.targetUID != "folder-1" || *mover.targetSeq != 2 {
		t.Errorf("mover args mismatch: target=%s, seq=%v", mover.targetUID, mover.targetSeq)
	}

	// MoveItem validation error
	if err := p.MoveItem(context.Background(), "col-1", "folder-1", "subfolder-1", nil); err == nil {
		t.Error("expected error moving folder into descendant")
	}

	// DeleteItem
	p.Select("req-1")
	p.SetExpanded("req-1", true)
	if err := p.DeleteItem(context.Background(), "col-1", "req-1"); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	if p.SelectedUID() != "" {
		t.Error("expected selected UID to clear after deletion")
	}
	if p.IsExpanded("req-1") {
		t.Error("expected expanded state to clear after deletion")
	}

	// LoadCollection
	uid, err := p.LoadCollection(context.Background(), "/path/to/col")
	if err != nil || uid != "col-loaded-uid" {
		t.Fatalf("LoadCollection failed: %v, %s", err, uid)
	}

	// HandleEvent CollectionChanged
	p.Select("req-2")
	p.SetExpanded("req-2", true)
	p.HandleEvent(events.CollectionChanged{
		CollectionUID: "col-1",
		Change: events.TreeChange{
			Origin:  events.OriginExternal,
			Removed: []string{"req-2"},
		},
	})
	if p.SelectedUID() != "" {
		t.Error("expected selected UID to clear after CollectionChanged removed")
	}
	if p.IsExpanded("req-2") {
		t.Error("expected expanded state to clear after CollectionChanged removed")
	}

	// Error path tests for Creator / Deleter / Loader
	creator.err = errors.New("creator error")
	if _, err := p.CreateRequest(context.Background(), "col-1", "", "name", "GET", ""); err == nil {
		t.Error("expected error from creator")
	}

	deleter.err = errors.New("deleter error")
	if err := p.DeleteItem(context.Background(), "col-1", "some-uid"); err == nil {
		t.Error("expected error from deleter")
	}

	loader.err = errors.New("loader error")
	if _, err := p.LoadCollection(context.Background(), "/bad/path"); err == nil {
		t.Error("expected error from loader")
	}
}
