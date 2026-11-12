package app

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/app/commands"
	"fearless/internal/core/collection"
	"fearless/internal/core/model"
	"fearless/internal/platform"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/layout"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/environment"
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
	// ViewKindEnvironment edits the active collection's variables.
	ViewKindEnvironment
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
	titleMenuOpen    bool

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

	// preferences persists settings; nil runs on defaults (see
	// WithPreferencesService).
	preferences *commands.PreferencesService

	// urlField keeps the URL bar's highlighting/edit state across frames.
	urlField *widgets.TemplateField

	// environmentState holds the environment editor's transient row state.
	environmentState *environment.State

	// sessionPath overrides the default session.json location for tests.
	sessionPath string
}

// Option configures an App instance.
type Option func(*App)

// WithPreferencesService lets the app persist settings — notably the first-run
// storage location. Without it the app runs on defaults and cannot remember a
// folder, so the storage prompt cannot complete.
func WithPreferencesService(svc *commands.PreferencesService) Option {
	return func(a *App) {
		a.preferences = svc
	}
}

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
		environmentState:        &environment.State{},
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
		// The presenter edits a draft and hands it to a saver; the store is the
		// file side. Without these ports, creating, saving and deleting an
		// environment are all silent no-ops — the editor looks empty and
		// "+ New environment" does nothing.
		store := &environmentStore{collection: func() *model.Collection {
			if a.EnvironmentPresenter == nil {
				return nil
			}
			return a.EnvironmentPresenter.Collection()
		}}
		a.EnvironmentPresenter = presenters.NewEnvironmentPresenter(store, store, store, store)
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
		store := preferencesStore{svc: a.preferences}
		a.PreferencesPresenter = presenters.NewPreferencesPresenter(model.DefaultSettings(), store, store, nil)
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
				// The environment editor and selector show the collection the
				// user is working in, so selection drives that binding.
				a.bindEnvironment(uid)
				item, _ := a.SidebarPresenter.Collections().FindItem(uid)
				if item != nil {
					if item.IsFolder() {
						a.SidebarPresenter.Collections().ToggleExpanded(uid)
					} else {
						if a.RequestPresenter != nil {
							a.RequestPresenter.SetItem(item)
						}
						if a.TabsPresenter != nil {
							a.TabsPresenter.OpenTab(item)
						}
					}
				} else {
					for _, col := range a.SidebarPresenter.Collections().Collections() {
						if col.UID == uid {
							a.SidebarPresenter.Collections().ToggleExpanded(uid)
							break
						}
					}
				}
			}
		},
		OnToggleNode: func(uid string) {
			if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
				a.SidebarPresenter.Collections().ToggleExpanded(uid)
			}
		},
		OnOpenCollection: func() {
			a.OpenCollection()
		},
		OnCreateCollection: func() {
			a.CreateCollection()
		},
		OnImportCollection: func() {
			a.ShowImport()
		},
		OnCreateRequest: func(parentUID string) {
			a.NewRequest(parentUID)
		},
		OnCreateFolder: func(parentUID string) {
			a.NewFolder(parentUID)
		},
		OnDeleteItem: func(uid string) {
			if a.CollectionPresenter != nil {
				_ = a.CollectionPresenter.DeleteItem(context.Background(), "", uid)
			}
		},
		OnSelectEnvironment: func(envName string) {
			if a.EnvironmentPresenter != nil {
				_ = a.EnvironmentPresenter.SelectEnvironment(context.Background(), envName)
			}
		},
		OnManageEnvironments: func() {
			a.ShowEnvironment()
		},
		OnRunCollection: func(colUID string) {
			if a.SidebarPresenter != nil {
				a.SidebarPresenter.SelectSection(presenters.SidebarSectionRunner)
			}
		},
		OnCollectionSettings: func(colUID string) {
			a.ShowCollectionSettings()
		},
		OnCloneCollection: func(colUID string) {
			a.CreateCollection()
		},
		OnRenameItem: func(uid string) {
			if a.DialogQueue != nil {
				_, _ = a.DialogQueue.Prompt("Rename", []string{"New Name"}, func(_ context.Context, vals map[string]string) error {
					newName := vals["New Name"]
					if newName != "" && a.CollectionPresenter != nil {
						item, _ := a.CollectionPresenter.FindItem(uid)
						if item != nil {
							item.Name = newName
						}
					}
					return nil
				})
			}
		},
		OnRevealItem: func(uid string) {
			if a.CollectionPresenter != nil {
				cols := a.CollectionPresenter.Collections()
				if len(cols) > 0 && a.Services.Browser != nil {
					_ = a.Services.Browser.Open(context.Background(), cols[0].Path)
				}
			}
		},
		OnOpenTerminal: func(uid string) {},
		OnCreateMockServer: func() {
			if a.DialogQueue != nil {
				_, _ = a.DialogQueue.Prompt("Create Mock Server (Beta)", []string{"Port"}, func(_ context.Context, vals map[string]string) error {
					return nil
				})
			}
		},
		OnCollapseAll: func(colUID string) {
			if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
				nodes := a.SidebarPresenter.Collections().VisibleNodes()
				for _, n := range nodes {
					if n != nil && n.IsFolder && n.IsExpanded {
						a.SidebarPresenter.Collections().SetExpanded(n.UID, false)
					}
				}
			}
		},
	}

	a.NavScreen = navigation.NewScreen(a.SidebarPresenter, navActions)

	shellActions := shell.Actions{
		OnMenuItem: a.handleMenuKey,
		OnMenu: func() {
			a.titleMenuOpen = !a.titleMenuOpen
		},
		OnWorkspace: func() {},
		OnSettings: func() {
			a.TogglePreferences()
		},
		OnSelectEnvironment: func(name string) {
			if a.EnvironmentPresenter != nil {
				_ = a.EnvironmentPresenter.SelectEnvironment(context.Background(), name)
			}
		},
		OnManageEnvironment: func() {
			a.ShowEnvironment()
		},
		OnRun: func() {
			// No runner backend is wired into the GUI yet, so Run leads to
			// the place where running happens: the runner section of the
			// sidebar. A button that does nothing is worse than one that
			// moves the user closer to the action.
			if a.SidebarPresenter != nil {
				a.SidebarPresenter.SelectSection(presenters.SidebarSectionRunner)
			}
		},
		OnStop: func() {
			a.CancelActiveRequest()
		},
		// The window's own controls: the window has no native title bar,
		// so the app draws the buttons that minimize, maximize and close
		// it in its title strip.
		OnMinimize: func() {
			if win := a.window(); win != nil {
				win.Minimize()
			}
		},
		OnToggleMaximize: func() {
			if win := a.window(); win != nil {
				win.ToggleMaximize()
			}
		},
		OnClose: func() {
			if win := a.window(); win != nil {
				win.Close()
			}
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
		// No collection-header region: the title strip carries the
		// collection name, the environment selector and the run control,
		// and a second strip under it would answer the same question
		// twice.
		RequestPane: func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				if a.TabsPresenter != nil && len(a.TabsPresenter.Tabs()) > 0 {
					activeUID := ""
					if act := a.TabsPresenter.ActiveTab(); act != nil {
						activeUID = act.UID
					}
					widgets.RequestTabBar(c, a.TabsPresenter.Tabs(), activeUID, widgets.TabBarProps{
						OnSelect: func(uid string) {
							if a.TabsPresenter != nil {
								a.TabsPresenter.SelectTab(uid)
							}
							if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
								a.SidebarPresenter.Collections().Select(uid)
								item, _ := a.SidebarPresenter.Collections().FindItem(uid)
								if item != nil && a.RequestPresenter != nil {
									a.RequestPresenter.SetItem(item)
								}
							}
						},
						OnClose: func(uid string) {
							if a.TabsPresenter != nil {
								a.TabsPresenter.CloseTab(uid)
								if act := a.TabsPresenter.ActiveTab(); act != nil {
									if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
										a.SidebarPresenter.Collections().Select(act.UID)
										item, _ := a.SidebarPresenter.Collections().FindItem(act.UID)
										if item != nil && a.RequestPresenter != nil {
											a.RequestPresenter.SetItem(item)
										}
									}
								} else {
									if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
										a.SidebarPresenter.Collections().ClearSelection()
									}
									if a.RequestPresenter != nil {
										a.RequestPresenter.SetItem(nil)
									}
								}
							}
						},
						OnNewTab: func() {
							a.NewRequest("")
						},
					})
				}
				if a.RequestPresenter != nil {
					ui.Box(c).Grow(1).FillWidth().Children(func() {
						opts := request.Options{
							Resolves:       a.resolvesVariable,
							OnOpenVariable: a.EditVariable,
							Field:          a.urlField,
						}
						request.RequestScreenWith(c, a.RequestPresenter, opts, a.RequestScreenState)
					})
				}
			})
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
	a.Router.SetHandler(ActionChangeLayout, func() {
		a.ToggleLayout()
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
			a.CreateCollection()
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
		OnChangeLayout: func() {
			a.ToggleLayout()
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

func (a *App) buildLobeMenuItems() []overlay.MenuItem {
	return []overlay.MenuItem{
		{Key: "new-request", Label: "New Request", Icon: lucide.Plus(), Hotkey: "mod+n"},
		{Key: "new-collection", Label: "New Collection", Icon: lucide.FolderPlus(), Hotkey: "mod+shift+n"},
		{Key: "import-collection", Label: "Import Collection...", Icon: lucide.Upload(), Hotkey: "mod+o"},
		{Divider: true},
		{Key: "save", Label: "Save", Icon: lucide.Save(), Hotkey: "mod+s"},
		{Key: "close-tab", Label: "Close Tab", Icon: lucide.X(), Hotkey: "mod+w"},
		{Divider: true},
		{Key: "send-request", Label: "Send Request", Icon: lucide.Send(), Hotkey: "mod+enter"},
		{Key: "cancel-request", Label: "Cancel Request", Icon: lucide.Square(), Hotkey: "escape"},
		{Divider: true},
		{Key: "toggle-sidebar", Label: "Toggle Sidebar", Icon: lucide.Sidebar(), Hotkey: "mod+\\"},
		{Key: "toggle-console", Label: "Toggle Console", Icon: lucide.Terminal(), Hotkey: "mod+t"},
		{Key: "change-layout", Label: "Change Orientation", Icon: lucide.Columns(), Hotkey: "mod+j"},
		{Divider: true},
		{Key: "preferences", Label: "Preferences...", Icon: lucide.Settings(), Hotkey: "mod+,"},
		{Divider: true},
		{Key: "documentation", Label: "Documentation", Icon: lucide.BookOpen()},
		{Key: "about", Label: "About fearless", Icon: lucide.Info()},
		{Divider: true},
		{Key: "quit", Label: "Quit fearless", Icon: lucide.LogOut(), Danger: true, Hotkey: "mod+q"},
	}
}

func (a *App) handleMenuKey(key string) {
	actions := a.MenuActions()
	switch key {
	case "new-request":
		if actions.OnNewRequest != nil {
			actions.OnNewRequest()
		}
	case "new-collection":
		if actions.OnNewCollection != nil {
			actions.OnNewCollection()
		}
	case "import-collection":
		if actions.OnImportCollection != nil {
			actions.OnImportCollection()
		}
	case "save":
		if actions.OnSave != nil {
			actions.OnSave()
		}
	case "close-tab":
		if actions.OnCloseTab != nil {
			actions.OnCloseTab()
		}
	case "send-request":
		if actions.OnSendRequest != nil {
			actions.OnSendRequest()
		}
	case "cancel-request":
		if actions.OnCancelRequest != nil {
			actions.OnCancelRequest()
		}
	case "toggle-sidebar":
		if actions.OnToggleSidebar != nil {
			actions.OnToggleSidebar()
		}
	case "toggle-console":
		if actions.OnToggleConsole != nil {
			actions.OnToggleConsole()
		}
	case "change-layout":
		if actions.OnChangeLayout != nil {
			actions.OnChangeLayout()
		}
	case "preferences":
		if actions.OnPreferences != nil {
			actions.OnPreferences()
		}
	case "documentation":
		if actions.OnDocumentation != nil {
			actions.OnDocumentation()
		}
	case "about":
		if actions.OnAbout != nil {
			actions.OnAbout()
		}
	case "quit":
		if actions.OnQuit != nil {
			actions.OnQuit()
		}
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
		method := vals["Method"]
		if method == "" {
			method = "GET"
		}
		reqURL := vals["URL"]
		a.SafeUpdate(func() {
			if a.CollectionPresenter != nil {
				item, _ := a.CollectionPresenter.CreateRequest(context.Background(), "", parentUID, name, method, reqURL)
				if item != nil {
					if a.RequestPresenter != nil {
						a.RequestPresenter.SetItem(item)
					}
					if a.TabsPresenter != nil {
						a.TabsPresenter.OpenTab(item)
					}
				}
			}
		})
		return nil
	})
}

// CreateCollection opens the modal to create a new collection.
func (a *App) CreateCollection() {
	if a.DialogQueue == nil {
		return
	}
	_, _ = a.DialogQueue.CreateCollectionModal(func(ctx context.Context, name, location string, format model.CollectionFormat) error {
		a.SafeUpdate(func() {
			if a.CollectionPresenter != nil {
				col, _ := a.CollectionPresenter.CreateCollection(ctx, name, location, format)
				if col != nil {
					a.CollectionPresenter.Select(col.UID)
				}
			}
		})
		return nil
	})
}

// OpenCollection opens a directory picker or prompt to load an existing collection.
func (a *App) OpenCollection() {
	if a.Services.Dialogs != nil {
		dir, err := a.Services.Dialogs.OpenDirectory(context.Background(), platform.OpenOptions{
			Title: "Open Collection",
		})
		if err == nil && dir != "" {
			a.loadCollectionDir(dir)
			return
		}
		if errors.Is(err, platform.ErrCancelled) {
			return
		}
	}
	if a.DialogQueue != nil {
		_, _ = a.DialogQueue.Prompt("Open Collection", []string{"Folder Path"}, func(_ context.Context, vals map[string]string) error {
			path := strings.TrimSpace(vals["Folder Path"])
			if path == "" {
				return errors.New("folder path is required")
			}
			a.loadCollectionDir(path)
			return nil
		})
	}
}

func (a *App) loadCollectionDir(dir string) {
	a.SafeUpdate(func() {
		if a.CollectionPresenter != nil {
			col, err := collection.Load(dir)
			if err == nil && col != nil {
				cols := a.CollectionPresenter.Collections()
				for _, existing := range cols {
					if existing.Path == col.Path || existing.UID == col.UID {
						a.CollectionPresenter.Select(existing.UID)
						a.bindEnvironment(existing.UID)
						return
					}
				}
				a.CollectionPresenter.SetCollections(append(cols, col))
				a.CollectionPresenter.Select(col.UID)
				a.bindEnvironment(col.UID)
			}
		}
	})
	a.persistSession()
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
				if nextAct := a.TabsPresenter.ActiveTab(); nextAct != nil {
					if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
						a.SidebarPresenter.Collections().Select(nextAct.UID)
						item, _ := a.SidebarPresenter.Collections().FindItem(nextAct.UID)
						if item != nil && a.RequestPresenter != nil {
							a.RequestPresenter.SetItem(item)
						}
					}
				} else {
					if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
						a.SidebarPresenter.Collections().ClearSelection()
					}
					if a.RequestPresenter != nil {
						a.RequestPresenter.SetItem(nil)
					}
				}
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

// ToggleLayout switches the request and response pane orientation between stacked and side-by-side (Decision R-7).
func (a *App) ToggleLayout() {
	a.SafeUpdate(func() {
		if a.LayoutState.RequestResponseOrientation == layout.OrientationStacked {
			a.LayoutState.RequestResponseOrientation = layout.OrientationSideBySide
		} else {
			a.LayoutState.RequestResponseOrientation = layout.OrientationStacked
		}
	})
}

// ToggleConsole toggles the visibility of the bottom console dock.
func (a *App) ToggleConsole() {
	a.SafeUpdate(func() {
		a.LayoutState.ConsoleOpen = !a.LayoutState.ConsoleOpen
	})
}

// resolvesVariable reports whether name is defined for the active collection:
// in the active environment, or in the collection's own variables.
//
// It backs the URL bar's `{{variable}}` colouring, so an undefined name reads
// as a mistake before the request is sent.
func (a *App) resolvesVariable(name string) bool {
	if name == "" {
		return false
	}
	if a.EnvironmentPresenter != nil {
		if _, ok := a.EnvironmentPresenter.VariableValue(name); ok {
			return true
		}
	}
	return false
}

// ShowEnvironment displays the environment editor for the active collection.
func (a *App) ShowEnvironment() {
	a.SafeUpdate(func() {
		if a.environmentState != nil {
			a.environmentState.Focus, a.environmentState.Back = "", nil
		}
		a.ActiveView = ViewKindEnvironment
	})
}

// EditVariable opens the environment editor on one variable, for a
// `{{variable}}` double-clicked in the request URL bar.
//
// A name the environment does not have yet is created for the user to fill in,
// so one gesture covers both editing a variable and making a missing one. The
// editor offers Back, so the user returns to the request they came from.
func (a *App) EditVariable(name string) {
	a.SafeUpdate(func() {
		p := a.EnvironmentPresenter
		if p == nil || p.Collection() == nil {
			return
		}
		// The variable is defined somewhere else — a collection or folder
		// variable, say — so there is nothing for the environment table to
		// create or highlight.
		if a.resolvesVariable(name) && !environmentDefines(p.EffectiveEnvironment(), name) {
			return
		}
		env := p.EffectiveEnvironment()
		if env == nil {
			// No environment is being edited: edit the one in use, or make one.
			env = p.ActiveEnvironment()
			if env == nil {
				created, err := p.CreateEnvironment(context.Background(), environment.NextEnvironmentName(p.Environments()))
				if err != nil || created == nil {
					return
				}
				env = created
			}
			p.SetEditingEnvironment(env.Name)
		}
		if a.environmentState != nil {
			a.environmentState.Focus = name
			a.environmentState.Back = a.ShowMain
		}
		a.ActiveView = ViewKindEnvironment
	})
}

// environmentDefines reports whether env itself has a variable called name.
func environmentDefines(env *model.Environment, name string) bool {
	if env == nil {
		return false
	}
	for _, v := range env.Variables {
		if v.NameOrEmpty() == name {
			return true
		}
	}
	return false
}

// ShowPreferences displays the application preferences screen.
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
		if imp == nil || imp.Collection == nil {
			return nil
		}
		// An import only exists in memory until it is written into the storage
		// folder: the app opens collections by path, so without this the
		// collection is gone on the next launch.
		dir, err := a.storeImportedCollection(imp.Collection)
		if err != nil {
			return err
		}
		if dir != "" {
			a.loadCollectionDir(dir)
		}
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
		case ViewKindEnvironment:
			a.renderEnvironmentView(c)
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
	// The sidebar draws from a plain snapshot, so it is re-read from the
	// presenters before the frame: the tree, the selected node and the
	// environments have to be the ones the app holds now, not the ones it
	// held when the screen was built.
	if a.NavScreen != nil && a.SidebarPresenter != nil {
		a.NavScreen.Snapshot = navigation.TakeSnapshot(a.SidebarPresenter)
	}

	if a.ShellScreen != nil {
		// Update shell snapshot from live presenters
		colName := "fearless"
		if a.CollectionPresenter != nil {
			if cols := a.CollectionPresenter.Collections(); len(cols) > 0 && cols[0] != nil {
				colName = cols[0].Name
			}
		}
		var envNames []string
		envName := ""
		if a.EnvironmentPresenter != nil {
			for _, env := range a.EnvironmentPresenter.Environments() {
				if env != nil {
					envNames = append(envNames, env.Name)
				}
			}
			envName = a.EnvironmentPresenter.ActiveEnvironmentName()
		}
		inFlight := false
		if a.RequestPresenter != nil {
			inFlight = a.RequestPresenter.InFlight()
		}
		// The window's state, for the title strip's window control: the
		// glyph says whether a click maximizes or restores.
		maximized := false
		if win := a.window(); win != nil {
			maximized = win.IsMaximized()
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
			MenuOpen:         &a.titleMenuOpen,
			MenuItems:        a.buildLobeMenuItems(),
			RequestCount:     requestCount,
			EnvironmentName:  envName,
			Environments:     envNames,
			InFlight:         inFlight,
			Maximized:        maximized,
			Status:           status,
			ActiveConsoleTab: a.activeConsoleTab,
		}

		a.ShellScreen.View(c)
	}
}

// window returns the main window, or nil before the window exists.
func (a *App) window() *mygo.Window {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Window
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

// renderEnvironmentView shows the environment editor over the workspace, with
// a back bar matching collection settings.
func (a *App) renderEnvironmentView(c *ui.Context) {
	tokens := lobetheme.TokensOf(c)
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(40).FillWidth().Padding(8, 16).AlignItems(ui.Center).Background(tokens.BgContainer).Children(func() {
			widgets.Button(c, "← Back to Workspace", widgets.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() { a.ShowMain() })
			ui.Box(c).Width(16)
			ui.Text(c, "Environments").TextColor(tokens.TextHeading)
		})
		ui.Box(c).Fill().Children(func() {
			environment.Screen(c, a.EnvironmentPresenter, a.environmentState)
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
