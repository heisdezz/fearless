package navigation

import (
	"fearless/internal/ui/presenters"
)

// Snapshot is an immutable snapshot of presenters data for rendering navigation views.
type Snapshot struct {
	ActiveSection     presenters.SidebarSection
	IsCollapsed       bool
	SidebarWidth      float32
	FilterQuery       string
	VisibleNodes      []*presenters.TreeNode
	SelectedUID       string
	CollectionName    string
	CollectionUID     string
	Environments      []string
	ActiveEnvironment string
}

// Actions captures all intent-based callbacks dispatched from navigation view interactions.
type Actions struct {
	OnFilterChanged      func(string)
	OnSelectNode         func(uid string)
	OnToggleNode         func(uid string)
	OnOpenCollection     func()
	OnCreateCollection   func()
	OnImportCollection   func()
	OnCreateRequest      func(parentUID string)
	OnCreateFolder       func(parentUID string)
	OnDeleteItem         func(uid string)
	OnSelectEnvironment  func(envName string)
	OnManageEnvironments func()
	OnRunCollection      func(colUID string)
	OnCollectionSettings func(colUID string)
	OnCloneCollection    func(colUID string)
	OnRenameItem         func(uid string)
	OnRevealItem         func(uid string)
	OnOpenTerminal       func(uid string)
	OnCollapseAll        func(colUID string)
}

// Screen encapsulates the state and action callbacks for the navigation views.
type Screen struct {
	Snapshot       Snapshot
	Actions        Actions
	HeaderMenuOpen bool
	SearchVisible  bool
	OpenMenuUID    string
}

// NewScreen creates an initialized Screen from a sidebar presenter and command actions.
func NewScreen(sidebar *presenters.SidebarPresenter, actions Actions) *Screen {
	return &Screen{
		Snapshot: TakeSnapshot(sidebar),
		Actions:  actions,
	}
}

// TakeSnapshot creates an immutable plain data Snapshot from the given sidebar presenter.
func TakeSnapshot(sidebar *presenters.SidebarPresenter) Snapshot {
	if sidebar == nil {
		return Snapshot{ActiveSection: presenters.SidebarSectionCollections}
	}

	colPres := sidebar.Collections()
	envPres := sidebar.Environments()

	snap := Snapshot{
		ActiveSection: sidebar.ActiveSection(),
		IsCollapsed:   sidebar.IsCollapsed(),
		SidebarWidth:  sidebar.Width(),
	}

	if colPres != nil {
		snap.FilterQuery = colPres.Filter()
		nodes := colPres.VisibleNodes()
		if len(nodes) > 0 {
			snap.VisibleNodes = make([]*presenters.TreeNode, len(nodes))
			for i, n := range nodes {
				if n != nil {
					cp := *n
					snap.VisibleNodes[i] = &cp
				}
			}
		}
		snap.SelectedUID = colPres.SelectedUID()
		cols := colPres.Collections()
		if len(cols) > 0 && cols[0] != nil {
			snap.CollectionName = cols[0].Name
			snap.CollectionUID = cols[0].UID
		}
	}

	if envPres != nil {
		envs := envPres.Environments()
		for _, env := range envs {
			if env != nil {
				snap.Environments = append(snap.Environments, env.Name)
			}
		}
		snap.ActiveEnvironment = envPres.ActiveEnvironmentName()
	}

	return snap
}
