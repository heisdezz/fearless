package presenters

import (
	"testing"

	"fearless/internal/core/model"
)

func TestTabsPresenter_OpenAndSelect(t *testing.T) {
	p := NewTabsPresenter()

	changedCount := 0
	p.OnChanged(func() {
		changedCount++
	})

	if p.ActiveTab() != nil {
		t.Errorf("expected no active tab initially")
	}
	if len(p.Tabs()) != 0 {
		t.Errorf("expected empty tabs list initially")
	}

	// Nil open
	p.OpenTab(nil)
	if len(p.Tabs()) != 0 {
		t.Errorf("opening nil item should not add tab")
	}

	// Open first item
	item1 := &model.Item{
		UID:  "item-1",
		Name: "Get Users",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/users",
		},
	}
	p.OpenTab(item1)

	if len(p.Tabs()) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(p.Tabs()))
	}
	active := p.ActiveTab()
	if active == nil || active.UID != "item-1" {
		t.Fatalf("expected active tab item-1, got %v", active)
	}
	if active.Title != "Get Users" || active.Method != "GET" {
		t.Errorf("tab fields mismatch: %+v", active)
	}
	if !active.IsActive {
		t.Errorf("expected active tab to have IsActive=true")
	}

	// Open second item
	item2 := &model.Item{
		UID:  "item-2",
		Name: "Create User",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "POST",
		},
	}
	p.OpenTab(item2)

	if len(p.Tabs()) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(p.Tabs()))
	}
	if p.ActiveTab().UID != "item-2" {
		t.Fatalf("expected active tab to be item-2, got %s", p.ActiveTab().UID)
	}
	if p.Tabs()[0].IsActive || !p.Tabs()[1].IsActive {
		t.Errorf("tab active flags incorrect: tab0=%v, tab1=%v", p.Tabs()[0].IsActive, p.Tabs()[1].IsActive)
	}

	// Re-opening item-1 switches to it without adding duplicate
	p.OpenTab(item1)
	if len(p.Tabs()) != 2 {
		t.Fatalf("expected 2 tabs after reopening item-1, got %d", len(p.Tabs()))
	}
	if p.ActiveTab().UID != "item-1" {
		t.Fatalf("expected active tab to be item-1, got %s", p.ActiveTab().UID)
	}

	// SelectTab
	p.SelectTab("item-2")
	if p.ActiveTab().UID != "item-2" {
		t.Fatalf("expected active tab to be item-2 after SelectTab, got %s", p.ActiveTab().UID)
	}

	// Open untitled & transient item
	itemTransient := &model.Item{
		UID:  "transient_123",
		Name: "",
		Request: &model.Request{
			Method: "PUT",
		},
	}
	p.OpenTab(itemTransient)
	if len(p.Tabs()) != 3 {
		t.Fatalf("expected 3 tabs, got %d", len(p.Tabs()))
	}
	transientTab := p.ActiveTab()
	if transientTab.Title != "Untitled" {
		t.Errorf("expected 'Untitled', got %s", transientTab.Title)
	}
	if !transientTab.IsTransient {
		t.Errorf("expected IsTransient=true for transient UID")
	}
}

func TestTabsPresenter_CloseAndReorder(t *testing.T) {
	p := NewTabsPresenter()

	item1 := &model.Item{UID: "tab-1", Name: "Tab 1"}
	item2 := &model.Item{UID: "tab-2", Name: "Tab 2"}
	item3 := &model.Item{UID: "tab-3", Name: "Tab 3"}

	p.OpenTab(item1)
	p.OpenTab(item2)
	p.OpenTab(item3)

	// Currently tab-3 is active.
	// Reorder tab-0 ("tab-1") to index 2 (end): order becomes [tab-2, tab-3, tab-1]
	p.Reorder(0, 2)
	tabs := p.Tabs()
	if tabs[0].UID != "tab-2" || tabs[1].UID != "tab-3" || tabs[2].UID != "tab-1" {
		t.Fatalf("unexpected order after Reorder(0, 2): %s, %s, %s", tabs[0].UID, tabs[1].UID, tabs[2].UID)
	}

	// Reorder out of range should be no-op
	p.Reorder(-1, 2)
	p.Reorder(0, 5)
	p.Reorder(1, 1)

	// Close inactive tab (tab-1, index 2)
	p.CloseTab("tab-1")
	if len(p.Tabs()) != 2 {
		t.Fatalf("expected 2 tabs after closing tab-1, got %d", len(p.Tabs()))
	}
	if p.ActiveTab().UID != "tab-3" {
		t.Errorf("active tab should remain tab-3, got %s", p.ActiveTab().UID)
	}

	// Close active tab (tab-3, at index 1, which is the last tab) -> should activate tab-2
	p.CloseTab("tab-3")
	if len(p.Tabs()) != 1 {
		t.Fatalf("expected 1 tab after closing tab-3, got %d", len(p.Tabs()))
	}
	if p.ActiveTab().UID != "tab-2" {
		t.Errorf("expected active tab to fall back to tab-2, got %s", p.ActiveTab().UID)
	}

	// Close nonexistent tab
	p.CloseTab("nonexistent")
	if len(p.Tabs()) != 1 {
		t.Fatalf("closing nonexistent tab should be no-op")
	}

	// Close remaining tab
	p.CloseTab("tab-2")
	if len(p.Tabs()) != 0 {
		t.Fatalf("expected 0 tabs after closing tab-2")
	}
	if p.ActiveTab() != nil {
		t.Fatalf("expected nil active tab after closing all tabs")
	}
}

func TestTabsPresenter_CloseOthersAndCloseAll(t *testing.T) {
	p := NewTabsPresenter()

	item1 := &model.Item{UID: "tab-1", Name: "Tab 1"}
	item2 := &model.Item{UID: "tab-2", Name: "Tab 2"}
	item3 := &model.Item{UID: "tab-3", Name: "Tab 3"}

	p.OpenTab(item1)
	p.OpenTab(item2)
	p.OpenTab(item3)

	p.CloseOthers("tab-2")
	if len(p.Tabs()) != 1 {
		t.Fatalf("expected 1 tab after CloseOthers, got %d", len(p.Tabs()))
	}
	if p.ActiveTab().UID != "tab-2" {
		t.Errorf("expected active tab to be tab-2, got %s", p.ActiveTab().UID)
	}

	p.OpenTab(item1)
	if len(p.Tabs()) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(p.Tabs()))
	}

	p.CloseAll()
	if len(p.Tabs()) != 0 {
		t.Fatalf("expected 0 tabs after CloseAll, got %d", len(p.Tabs()))
	}
	if p.ActiveTab() != nil {
		t.Fatalf("expected nil active tab after CloseAll")
	}
}
