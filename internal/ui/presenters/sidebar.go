package presenters

import (
	"sync"

	"fearless/internal/core/events"
)

// SidebarSection identifies a section of the sidebar. Only Collections is left:
// the History and Runner sections were placeholders with no backing data.
type SidebarSection string

const (
	SidebarSectionCollections SidebarSection = "collections"
)

// AllSidebarSections lists all available sidebar sections in standard order.
var AllSidebarSections = []SidebarSection{
	SidebarSectionCollections,
}

const (
	DefaultSidebarWidth float32 = 260.0
	MinSidebarWidth     float32 = 180.0
	MaxSidebarWidth     float32 = 600.0
)

// SidebarPresenter coordinates sidebar navigation, section switching, and sizing view-models.
type SidebarPresenter struct {
	mu            sync.RWMutex
	width         float32
	collapsed     bool
	activeSection SidebarSection
	collections   *CollectionPresenter
	environments  *EnvironmentPresenter
	onChanged     func()
}

// NewSidebarPresenter creates an initialized SidebarPresenter.
func NewSidebarPresenter(col *CollectionPresenter, env *EnvironmentPresenter) *SidebarPresenter {
	return &SidebarPresenter{
		width:         DefaultSidebarWidth,
		activeSection: SidebarSectionCollections,
		collections:   col,
		environments:  env,
	}
}

// Width returns the sidebar width in DIPs.
func (p *SidebarPresenter) Width() float32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.width
}

// SetWidth updates the sidebar width, clamped within Min and Max boundaries.
func (p *SidebarPresenter) SetWidth(w float32) {
	p.mu.Lock()
	if w < MinSidebarWidth {
		w = MinSidebarWidth
	} else if w > MaxSidebarWidth {
		w = MaxSidebarWidth
	}
	p.width = w
	p.mu.Unlock()
	p.notifyChanged()
}

// IsCollapsed reports whether the sidebar is collapsed into an icon rail.
func (p *SidebarPresenter) IsCollapsed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.collapsed
}

// SetCollapsed sets the collapsed/rail state.
func (p *SidebarPresenter) SetCollapsed(collapsed bool) {
	p.mu.Lock()
	p.collapsed = collapsed
	p.mu.Unlock()
	p.notifyChanged()
}

// ToggleCollapsed toggles between expanded and rail modes.
func (p *SidebarPresenter) ToggleCollapsed() {
	p.mu.Lock()
	p.collapsed = !p.collapsed
	p.mu.Unlock()
	p.notifyChanged()
}

// ActiveSection returns the selected sidebar section.
func (p *SidebarPresenter) ActiveSection() SidebarSection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeSection
}

// SelectSection switches the active sidebar section.
func (p *SidebarPresenter) SelectSection(s SidebarSection) {
	p.mu.Lock()
	p.activeSection = s
	p.mu.Unlock()
	p.notifyChanged()
}

// Collections returns the nested CollectionPresenter.
func (p *SidebarPresenter) Collections() *CollectionPresenter {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.collections
}

// Environments returns the nested EnvironmentPresenter.
func (p *SidebarPresenter) Environments() *EnvironmentPresenter {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.environments
}

// HandleEvent forwards events to child presenters.
func (p *SidebarPresenter) HandleEvent(ev events.Event) {
	if p.collections != nil {
		p.collections.HandleEvent(ev)
	}
	p.notifyChanged()
}

// OnChanged registers an invalidation callback hook.
func (p *SidebarPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *SidebarPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
