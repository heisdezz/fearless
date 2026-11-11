package presenters

import (
	"testing"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func TestSidebarPresenter_WidthAndSizing(t *testing.T) {
	colPresenter := NewCollectionPresenter(nil, nil, nil, nil)
	envPresenter := NewEnvironmentPresenter(nil, nil, nil, nil)
	p := NewSidebarPresenter(colPresenter, envPresenter)

	changed := 0
	p.OnChanged(func() {
		changed++
	})

	if p.Width() != DefaultSidebarWidth {
		t.Errorf("expected default width %v, got %v", DefaultSidebarWidth, p.Width())
	}
	if p.IsCollapsed() {
		t.Error("expected sidebar not collapsed initially")
	}

	// Sizing within bounds
	p.SetWidth(320.0)
	if p.Width() != 320.0 {
		t.Errorf("expected width 320, got %v", p.Width())
	}

	// Clamp below Min
	p.SetWidth(50.0)
	if p.Width() != MinSidebarWidth {
		t.Errorf("expected min clamp %v, got %v", MinSidebarWidth, p.Width())
	}

	// Clamp above Max
	p.SetWidth(1000.0)
	if p.Width() != MaxSidebarWidth {
		t.Errorf("expected max clamp %v, got %v", MaxSidebarWidth, p.Width())
	}

	// Collapse toggling
	p.ToggleCollapsed()
	if !p.IsCollapsed() {
		t.Error("expected collapsed after toggle")
	}
	p.SetCollapsed(false)
	if p.IsCollapsed() {
		t.Error("expected not collapsed after SetCollapsed(false)")
	}

	if changed == 0 {
		t.Error("onChanged was never called")
	}
}

func TestSidebarPresenter_SectionsAndAccessors(t *testing.T) {
	colPresenter := NewCollectionPresenter(nil, nil, nil, nil)
	envPresenter := NewEnvironmentPresenter(nil, nil, nil, nil)
	p := NewSidebarPresenter(colPresenter, envPresenter)

	if p.ActiveSection() != SidebarSectionCollections {
		t.Errorf("expected default section collections, got %s", p.ActiveSection())
	}
	if p.Collections() != colPresenter {
		t.Error("Collections() mismatch")
	}
	if p.Environments() != envPresenter {
		t.Error("Environments() mismatch")
	}

	for _, s := range AllSidebarSections {
		p.SelectSection(s)
		if p.ActiveSection() != s {
			t.Errorf("expected section %s, got %s", s, p.ActiveSection())
		}
	}
}

func TestSidebarPresenter_HandleEvent(t *testing.T) {
	colPresenter := NewCollectionPresenter(nil, nil, nil, nil)
	p := NewSidebarPresenter(colPresenter, nil)

	col := &model.Collection{
		UID:  "col-sb",
		Name: "Sidebar Collection",
		Items: []*model.Item{
			{UID: "item-sb-1", Name: "Req 1"},
		},
	}
	colPresenter.SetCollections([]*model.Collection{col})
	colPresenter.Select("item-sb-1")
	colPresenter.SetExpanded("item-sb-1", true)

	p.HandleEvent(events.CollectionChanged{
		CollectionUID: "col-sb",
		Change: events.TreeChange{
			Origin:  events.OriginExternal,
			Removed: []string{"item-sb-1"},
		},
	})

	if colPresenter.SelectedUID() != "" {
		t.Error("expected child presenter selection to clear via HandleEvent")
	}
	if colPresenter.IsExpanded("item-sb-1") {
		t.Error("expected child presenter expansion to clear via HandleEvent")
	}
}
