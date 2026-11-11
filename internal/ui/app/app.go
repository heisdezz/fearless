package app

import (
	"context"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/platform"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/layout"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/navigation"
	"fearless/internal/ui/screens/request"
	"fearless/internal/ui/screens/response"
	"fearless/internal/ui/screens/settings"
	"fearless/internal/ui/screens/shell"
	"fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// ViewKind represents the active top-level screen view.
type ViewKind int

const (
	ViewKindMain ViewKind = iota
	ViewKindPreferences
	ViewKindCollectionSettings
)

// App manages the fearless GUI lifecycle, window, root view composition,
// keyboard routing, dialog queue, and thread-safe UI updates.
type App struct {
	mu               sync.Mutex
	Window           *mygo.Window
	Services         platform.Services
	Router           *KeybindingRouter
	DialogQueue      *dialogs.Queue
	LayoutState      layout.LayoutState
	ActiveView       ViewKind
	activeConsoleTab int

	// Presenters
	SidebarPresenter     *presenters.SidebarPresenter
	CollectionPresenter  *presenters.CollectionPresenter
	EnvironmentPresenter *presenters.EnvironmentPresenter
	RequestPresenter     *presenters.RequestPresenter
	ResponsePresenter    *presenters.ResponsePresenter
	TabsPresenter        *presenters.TabsPresenter
	PreferencesPresenter *presenters.PreferencesPresenter
	SettingsPresenter    *presenters.SettingsPresenter

	// Screens
	ShellScreen             *shell.Screen
	NavScreen               *navigation.Screen
	RequestScreenState      *request.RequestScreenState
	CollectionSettingsState *settings.CollectionSettingsState
}

// Option configures an App instance.
type Option func(*App)

// WithServices sets the platform services provider.
func WithServices(s platform.Services) Option {
	return func(a *App) {
		a.Services = s
	}
}

// WithRouter sets a custom keybinding router.
func WithRouter(r *KeybindingRouter) Option {
	return func(a *App) {
		a.Router = r
	}
}

// WithDialogQueue sets the active modal dialog queue.
func WithDialogQueue(q *dialogs.Queue) Option {
	return func(a *App) {
		a.DialogQueue = q
	}
}

// New creates and initializes a fearless GUI Application with default components.
func New(opts ...Option) *App {
	a := &App{
		LayoutState:             layout.DefaultLayoutState(),
		DialogQueue:             dialogs.NewQueue(),
		RequestScreenState:      &request.RequestScreenState{},
		CollectionSettingsState: &settings.CollectionSettingsState{},
	}

	for _, opt := range opts {
		opt(a)
	}

	if a.Router == nil {
		a.Router = DefaultKeybindingRouter()
	}

	a.initPresenters()
	a.initScreens()
	a.wireKeybindingHandlers()

	return a
}

func (a *App) initPresenters() {
	if a.CollectionPresenter == nil {
		a.CollectionPresenter = presenters.NewCollectionPresenter(nil, nil, nil, nil)
	}
	if a.EnvironmentPresenter == nil {
		a.EnvironmentPresenter = presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
	}
	if a.SidebarPresenter == nil {
		a.SidebarPresenter = presenters.NewSidebarPresenter(a.CollectionPresenter, a.EnvironmentPresenter)
	}
	if a.RequestPresenter == nil {
		a.RequestPresenter = presenters.NewRequestPresenter(nil, nil, nil)
	}
	if a.ResponsePresenter == nil {
		a.ResponsePresenter = presenters.NewResponsePresenter()
	}
	if a.TabsPresenter == nil {
		a.TabsPresenter = presenters.NewTabsPresenter()
	}
	if a.PreferencesPresenter == nil {
		a.PreferencesPresenter = presenters.NewPreferencesPresenter(model.Settings{}, nil, nil, nil)
	}
	if a.SettingsPresenter == nil {
		col := &model.Collection{
			UID:  "default-collection",
			Name: "Fearless Collection",
		}
		a.SettingsPresenter = presenters.NewCollectionSettingsPresenter(col, nil)
	}
}

func (a *App) initScreens() {
	navActions := navigation.Actions{
		OnSelectSection: func(sec presenters.SidebarSection) {
			if a.SidebarPresenter != nil {
				a.SidebarPresenter.SelectSection(sec)
			}
		},
		OnToggleCollapse: func() {
			a.ToggleSidebar()
		},
		OnFilterChanged: func(q string) {
			if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
				a.SidebarPresenter.Collections().SetFilter(q)
			}
		},
		OnSelectNode: func(uid string) {
			if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
				a.SidebarPresenter.Collections().Select(uid)
			}
		},
		OnToggleNode: func(uid string) {
			if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
				a.SidebarPresenter.Collections().ToggleExpanded(uid)
			}
		},
		OnOpenCollection: func() {
			a.ShowImport()
		},
		OnCreateRequest: func(parentUID string) {
			a.NewRequest(parentUID)
		},
		OnCreateFolder: func(parentUID string) {
			a.NewFolder(parentUID)
		},
		OnSelectEnvironment: func(envName string) {
			if a.EnvironmentPresenter != nil {
				_ = a.EnvironmentPresenter.SelectEnvironment(context.Background(), envName)
			}
		},
		OnManageEnvironments: func() {
			a.ShowPreferences()
		},
	}

	a.NavScreen = navigation.NewScreen(a.SidebarPresenter, navActions)

	shellActions := shell.Actions{
		OnMenu: func() {
			a.mu.Lock()
			win := a.Window
			a.mu.Unlock()
			if win != nil {
				menu := BuildMenu(a.MenuActions(), a.Router)
				menu.Popup(win)
			}
		},
		OnWorkspace: func() {},
		OnSettings: func() {
			a.TogglePreferences()
		},
		OnEnvironment: func() {
			a.ShowPreferences()
		},
		OnConsoleTab: func(tab int) {
			a.mu.Lock()
			a.activeConsoleTab = tab
			a.mu.Unlock()
		},
		OnConsoleClose: func() {
			a.mu.Lock()
			a.LayoutState.ConsoleOpen = false
			a.mu.Unlock()
		},
	}

	a.ShellScreen = &shell.Screen{
		State:   &a.LayoutState,
		Budget:  layout.DefaultRegionBudget(),
		Actions: shellActions,
		Sidebar: func(c *ui.Context, nav lobe.SideNavState) {
			if a.NavScreen != nil {
				a.NavScreen.Sidebar(c, nav)
			}
		},
		CollectionHeader: func(c *ui.Context) {
			if a.NavScreen != nil {
				a.NavScreen.CollectionHeader(c)
			}
		},
		RequestPane: func(c *ui.Context) {
			if a.RequestPresenter != nil {
				request.RequestScreen(c, a.RequestPresenter, a.RequestScreenState)
			}
		},
		ResponsePane: func(c *ui.Context) {
			if a.ResponsePresenter != nil {
				response.Screen(c, response.Props{Presenter: a.ResponsePresenter})
			}
		},
	}
}

func (a *App) wireKeybindingHandlers() {
	if a.Router == nil {
		return
	}

	a.Router.SetHandler(ActionSendRequest, func() {
		a.SendActiveRequest()
	})
	a.Router.SetHandler(ActionCancelRequest, func() {
		a.CancelActiveRequest()
	})
	a.Router.SetHandler(ActionSave, func() {
		a.SaveActiveRequest()
	})
	a.Router.SetHandler(ActionNewRequest, func() {
		a.NewRequest("")
	})
	a.Router.SetHandler(ActionCloseTab, func() {
		a.CloseActiveTab()
	})
	a.Router.SetHandler(ActionPreferences, func() {
		a.TogglePreferences()
	})
	a.Router.SetHandler(ActionToggleSidebar, func() {
		a.ToggleSidebar()
	})
	a.Router.SetHandler(ActionToggleConsole, func() {
		a.ToggleConsole()
	})
	a.Router.SetHandler(ActionImport, func() {
		a.ShowImport()
	})
}

// MenuActions returns the bound actions for window and application menus.
func (a *App) MenuActions() MenuActions {
	return MenuActions{
		OnNewRequest: func() {
			a.NewRequest("")
		},
		OnNewCollection: func() {
			a.NewFolder("")
		},
		OnImportCollection: func() {
			a.ShowImport()
		},
		OnSave: func() {
			a.SaveActiveRequest()
		},
		OnCloseTab: func() {
			a.CloseActiveTab()
		},
		OnPreferences: func() {
			a.ShowPreferences()
		},
		OnQuit: func() {
			mygo.App.Quit()
		},
		OnSendRequest: func() {
			a.SendActiveRequest()
		},
		OnCancelRequest: func() {
			a.CancelActiveRequest()
		},
		OnToggleSidebar: func() {
			a.ToggleSidebar()
		},
		OnToggleConsole: func() {
			a.ToggleConsole()
		},
		OnAbout: func() {
			mygo.App.ShowAboutPanel(mygo.AboutPanelOptions{
				ApplicationName: "fearless",
				Version:         "1.0.0",
			})
		},
		OnDocumentation: func() {
			browser := a.Services.BrowserOr()
			_ = browser.Open(context.Background(), "https://github.com/usebruno/bruno")
		},
	}
}

// SafeUpdate marshals a UI mutation callback onto the MyGo UI loop via Window.Update.
// If the window is not yet created (headless or startup), it executes synchronously.
func (a *App) SafeUpdate(fn func()) {
	a.mu.Lock()
	win := a.Window
	a.mu.Unlock()

	if win != nil {
		win.Update(fn)
	} else {
		a.mu.Lock()
		defer a.mu.Unlock()
		fn()
	}
}

// WrapUpload ensures that any field.Upload callbacks marshal their state changes
// onto the UI loop through Window.Update.
func (a *App) WrapUpload(p field.UploadProps) field.UploadProps {
	wrapped := p
	origOnFiles := p.OnFiles
	if origOnFiles != nil {
		wrapped.OnFiles = func(files []field.File) {
			a.SafeUpdate(func() {
				origOnFiles(files)
			})
		}
	}
	return wrapped
}

// SendActiveRequest triggers sending the request currently loaded in RequestPresenter.
func (a *App) SendActiveRequest() {
	if a.RequestPresenter != nil {
		a.SafeUpdate(func() {
			_, _ = a.RequestPresenter.Send(context.Background())
		})
	}
}

// CancelActiveRequest cancels any in-flight request.
func (a *App) CancelActiveRequest() {
	if a.RequestPresenter != nil {
		a.SafeUpdate(func() {
			_ = a.RequestPresenter.Cancel(context.Background())
		})
	}
}

// SaveActiveRequest saves the currently active request.
func (a *App) SaveActiveRequest() {
	if a.RequestPresenter != nil {
		a.SafeUpdate(func() {
			_ = a.RequestPresenter.Save(context.Background())
		})
	}
}

// NewRequest opens a creation dialog or prompt for a new request.
func (a *App) NewRequest(parentUID string) {
	if a.DialogQueue == nil {
		return
	}
	_, _ = a.DialogQueue.Prompt("New Request", []string{"Request Name"}, func(_ context.Context, vals map[string]string) error {
		name := vals["Request Name"]
		if name == "" {
			name = "Untitled Request"
		}
		a.SafeUpdate(func() {
			if a.CollectionPresenter != nil {
				_, _ = a.CollectionPresenter.CreateRequest(context.Background(), "", parentUID, name, "GET", "")
			}
		})
		return nil
	})
}

// NewFolder opens a prompt for a new folder.
func (a *App) NewFolder(parentUID string) {
	if a.DialogQueue == nil {
		return
	}
	_, _ = a.DialogQueue.Prompt("New Folder", []string{"Folder Name"}, func(_ context.Context, vals map[string]string) error {
		name := vals["Folder Name"]
		if name == "" {
			name = "New Folder"
		}
		a.SafeUpdate(func() {
			if a.CollectionPresenter != nil {
				_, _ = a.CollectionPresenter.CreateFolder(context.Background(), "", parentUID, name)
			}
		})
		return nil
	})
}

// CloseActiveTab closes the currently selected tab.
func (a *App) CloseActiveTab() {
	a.SafeUpdate(func() {
		if a.TabsPresenter != nil {
			if act := a.TabsPresenter.ActiveTab(); act != nil && act.UID != "" {
				a.TabsPresenter.CloseTab(act.UID)
			}
		}
	})
}

// ToggleSidebar switches the sidebar expanded state.
func (a *App) ToggleSidebar() {
	a.SafeUpdate(func() {
		a.LayoutState.SidebarExpanded = !a.LayoutState.SidebarExpanded
	})
}

// ToggleConsole toggles the visibility of the bottom console dock.
func (a *App) ToggleConsole() {
	a.SafeUpdate(func() {
		a.LayoutState.ConsoleOpen = !a.LayoutState.ConsoleOpen
	})
}

// ShowPreferences displays the preferences screen.
func (a *App) ShowPreferences() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindPreferences
	})
}

// ShowCollectionSettings displays the collection settings screen.
func (a *App) ShowCollectionSettings() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindCollectionSettings
	})
}

// ShowMain switches back to the main request/workspace view.
func (a *App) ShowMain() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindMain
	})
}

// TogglePreferences toggles between preferences and main view.
func (a *App) TogglePreferences() {
	a.SafeUpdate(func() {
		if a.ActiveView == ViewKindPreferences {
			a.ActiveView = ViewKindMain
		} else {
			a.ActiveView = ViewKindPreferences
		}
	})
}

// ShowImport opens the import collection modal flow.
func (a *App) ShowImport() {
	if a.DialogQueue == nil {
		return
	}
	_, _ = a.DialogQueue.ImportModal(func(ctx context.Context, imp *dialogs.Imported) error {
		a.SafeUpdate(func() {
			if a.CollectionPresenter != nil && imp != nil && imp.Collection != nil {
				cols := a.CollectionPresenter.Collections()
				a.CollectionPresenter.SetCollections(append(cols, imp.Collection))
			}
		})
		return nil
	})
}

// RootView renders the primary application frame.
// Exactly per packages.md UI-APP specification:
// - theme.Apply is called once
// - feedback.ToastHost is mounted once
// - keybindings are routed
// - active dialogs or screen regions are composed
func (a *App) RootView(c *ui.Context) {
	// 1. theme.Apply once
	theme.Apply(c)

	// 2. Route active keybindings
	if a.Router != nil {
		a.Router.Route(c)
	}

	tokens := lobetheme.TokensOf(c)

	// 3. Render root frame with toast host mounted once
	ui.Column(c).Fill().Background(tokens.BgLayout).Children(func() {
		// Active Screen View
		switch a.ActiveView {
		case ViewKindPreferences:
			a.renderPreferencesView(c, tokens)
		case ViewKindCollectionSettings:
			a.renderCollectionSettingsView(c, tokens)
		default:
			a.renderMainView(c)
		}

		// 4. Modal dialogs overlay if active
		if a.DialogQueue != nil {
			dialogs.View(c, a.DialogQueue)
		}

		// 5. Toast host mounted once at the root view
		feedback.ToastHost(c, feedback.ToastHostProps{})
	})
}

func (a *App) renderMainView(c *ui.Context) {
	if a.ShellScreen != nil {
		// Update shell snapshot from live presenters
		colName := "fearless"
		if a.CollectionPresenter != nil {
			if cols := a.CollectionPresenter.Collections(); len(cols) > 0 && cols[0] != nil {
				colName = cols[0].Name
			}
		}
		envName := ""
		if a.EnvironmentPresenter != nil {
			envName = a.EnvironmentPresenter.ActiveEnvironmentName()
		}

		// The status bar shows facts, never decoration (PRODUCT.md): the
		// request count of the mounted collections, the engine state and
		// the dirty flag. An empty bar reads as dead space, and Bruno's
		// bar is always populated.
		requestCount, unsaved := 0, false
		if a.CollectionPresenter != nil {
			for _, col := range a.CollectionPresenter.Collections() {
				if col == nil {
					continue
				}
				requestCount += countRequestItems(col.Items)
				unsaved = unsaved || col.Dirty
			}
		}
		status := make([]shell.StatusEntry, 0, 2)
		if a.RequestPresenter != nil && a.RequestPresenter.InFlight() {
			status = append(status, shell.StatusEntry{Label: "Sending…"})
		} else {
			status = append(status, shell.StatusEntry{Label: "Ready", Tooltip: "No in-flight requests"})
		}
		if unsaved {
			status = append(status, shell.StatusEntry{
				Label:   "Unsaved changes",
				Tooltip: "A mounted collection has uncommitted changes",
			})
		}

		a.ShellScreen.Snapshot = shell.Snapshot{
			AppName:          "fearless",
			CollectionName:   colName,
			WorkspaceName:    "Default Workspace",
			ShowMenu:         true,
			RequestCount:     requestCount,
			EnvironmentName:  envName,
			Status:           status,
			ActiveConsoleTab: a.activeConsoleTab,
		}

		a.ShellScreen.View(c)
	}
}

// countRequestItems counts request items in a collection tree, recursing
// through folders; inert entries (no saved request) don't count.
func countRequestItems(items []*model.Item) int {
	n := 0
	for _, it := range items {
		if it == nil {
			continue
		}
		if it.IsFolder() {
			n += countRequestItems(it.Children)
		} else if it.Request != nil {
			n++
		}
	}
	return n
}

func (a *App) renderPreferencesView(c *ui.Context, tokens *lobetheme.Tokens) {
	ui.Column(c).Fill().Children(func() {
		// Navigation Bar
		ui.Row(c).Height(40).FillWidth().Padding(8, 16).AlignItems(ui.Center).Background(tokens.BgContainer).Children(func() {
			widgets.Button(c, "← Back to Workspace", widgets.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				a.ShowMain()
			})

			ui.Box(c).Width(16)
			ui.Text(c, "Application Preferences").TextColor(tokens.TextHeading)
		})

		// Body
		ui.Box(c).Fill().Children(func() {
			if a.PreferencesPresenter != nil {
				settings.PreferencesScreen(c, a.PreferencesPresenter)
			}
		})
	})
}

func (a *App) renderCollectionSettingsView(c *ui.Context, tokens *lobetheme.Tokens) {
	ui.Column(c).Fill().Children(func() {
		// Navigation Bar
		ui.Row(c).Height(40).FillWidth().Padding(8, 16).AlignItems(ui.Center).Background(tokens.BgContainer).Children(func() {
			widgets.Button(c, "← Back to Workspace", widgets.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				a.ShowMain()
			})

			ui.Box(c).Width(16)
			ui.Text(c, "Collection Settings").TextColor(tokens.TextHeading)
		})

		// Body
		ui.Box(c).Fill().Children(func() {
			if a.SettingsPresenter != nil {
				settings.CollectionSettingsScreen(
					c,
					a.SettingsPresenter,
					a.CollectionSettingsState,
				)
			}
		})
	})
}
