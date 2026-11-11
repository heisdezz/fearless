package presenters

import (
	"strings"
	"sync"

	"fearless/internal/core/model"
)

// TabItem represents an open tab in the request workspace.
type TabItem struct {
	UID         string
	Title       string
	Method      string
	IsDirty     bool
	IsActive    bool
	IsTransient bool
	Item        *model.Item
}

// TabsPresenter manages the open request tabs view-model.
type TabsPresenter struct {
	mu        sync.RWMutex
	tabs      []*TabItem
	activeUID string
	onChanged func()
}

// NewTabsPresenter creates an initialized TabsPresenter.
func NewTabsPresenter() *TabsPresenter {
	return &TabsPresenter{}
}

// Tabs returns the ordered list of open tabs.
func (p *TabsPresenter) Tabs() []*TabItem {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*TabItem, len(p.tabs))
	copy(out, p.tabs)
	return out
}

// ActiveTab returns the currently selected tab, or nil if none is open.
func (p *TabsPresenter) ActiveTab() *TabItem {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, t := range p.tabs {
		if t.UID == p.activeUID {
			return t
		}
	}
	if len(p.tabs) > 0 {
		return p.tabs[0]
	}
	return nil
}

// OpenTab opens an item as a tab. If already open, it is activated.
func (p *TabsPresenter) OpenTab(item *model.Item) {
	if item == nil {
		return
	}

	p.mu.Lock()
	for _, t := range p.tabs {
		if t.Item != nil && t.Item.UID == item.UID {
			p.activeUID = t.UID
			p.updateActiveStates()
			p.mu.Unlock()
			p.notifyChanged()
			return
		}
	}

	title := item.Name
	if title == "" {
		title = "Untitled"
	}
	method := "GET"
	if req := item.Effective(); req != nil && req.Method != "" {
		method = req.Method
	}

	newTab := &TabItem{
		UID:         item.UID,
		Title:       title,
		Method:      method,
		IsDirty:     item.Dirty(),
		IsActive:    true,
		IsTransient: item.UID == "" || strings.HasPrefix(item.UID, "transient_"),
		Item:        item,
	}

	p.tabs = append(p.tabs, newTab)
	p.activeUID = newTab.UID
	p.updateActiveStates()
	p.mu.Unlock()

	p.notifyChanged()
}

// SelectTab sets the active tab by UID.
func (p *TabsPresenter) SelectTab(uid string) {
	p.mu.Lock()
	p.activeUID = uid
	p.updateActiveStates()
	p.mu.Unlock()
	p.notifyChanged()
}

// CloseTab removes a tab. If it was active, adjacent tab is selected.
func (p *TabsPresenter) CloseTab(uid string) {
	p.mu.Lock()
	idx := -1
	for i, t := range p.tabs {
		if t.UID == uid {
			idx = i
			break
		}
	}

	if idx == -1 {
		p.mu.Unlock()
		return
	}

	p.tabs = append(p.tabs[:idx], p.tabs[idx+1:]...)

	if p.activeUID == uid {
		if len(p.tabs) == 0 {
			p.activeUID = ""
		} else if idx < len(p.tabs) {
			p.activeUID = p.tabs[idx].UID
		} else {
			p.activeUID = p.tabs[len(p.tabs)-1].UID
		}
	}

	p.updateActiveStates()
	p.mu.Unlock()
	p.notifyChanged()
}

// CloseOthers closes all tabs except the specified UID.
func (p *TabsPresenter) CloseOthers(uid string) {
	p.mu.Lock()
	var remaining []*TabItem
	for _, t := range p.tabs {
		if t.UID == uid {
			remaining = append(remaining, t)
		}
	}
	p.tabs = remaining
	p.activeUID = uid
	p.updateActiveStates()
	p.mu.Unlock()
	p.notifyChanged()
}

// CloseAll closes all open tabs.
func (p *TabsPresenter) CloseAll() {
	p.mu.Lock()
	p.tabs = nil
	p.activeUID = ""
	p.mu.Unlock()
	p.notifyChanged()
}

// Reorder moves a tab from fromIndex to toIndex.
func (p *TabsPresenter) Reorder(fromIndex, toIndex int) {
	p.mu.Lock()
	n := len(p.tabs)
	if fromIndex < 0 || fromIndex >= n || toIndex < 0 || toIndex >= n || fromIndex == toIndex {
		p.mu.Unlock()
		return
	}

	item := p.tabs[fromIndex]
	p.tabs = append(p.tabs[:fromIndex], p.tabs[fromIndex+1:]...)

	// Insert at toIndex
	var newTabs []*TabItem
	newTabs = append(newTabs, p.tabs[:toIndex]...)
	newTabs = append(newTabs, item)
	newTabs = append(newTabs, p.tabs[toIndex:]...)
	p.tabs = newTabs

	p.mu.Unlock()
	p.notifyChanged()
}

func (p *TabsPresenter) updateActiveStates() {
	for _, t := range p.tabs {
		t.IsActive = (t.UID == p.activeUID)
		if t.Item != nil {
			t.IsDirty = t.Item.Dirty()
			if req := t.Item.Effective(); req != nil {
				t.Method = req.Method
			}
		}
	}
}

// OnChanged registers an invalidation callback hook.
func (p *TabsPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *TabsPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
