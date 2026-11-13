package app

import (
	"context"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/ui/layout"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/navigation"
	"fearless/internal/ui/screens/request"
	"fearless/internal/ui/screens/response"
	"fearless/internal/ui/screens/shell"
	"fearless/internal/ui/widgets"
)

// initScreens wires the navigation shell and request/response panes to the
// app's presenters and actions.
func (a *App) initScreens() {
	navActions := navigation.Actions{
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
						if a.TabsPresenter != nil {
							a.TabsPresenter.OpenTab(item)
						}
						a.setActiveItem(item)
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
			a.SendActiveRequest()
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
			a.SendActiveRequest()
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
							}
							a.setActiveItem(a.findItem(uid))
						},
						OnClose: func(uid string) {
							if a.TabsPresenter != nil {
								a.TabsPresenter.CloseTab(uid)
								a.dropPane(uid)
								if act := a.TabsPresenter.ActiveTab(); act != nil {
									if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
										a.SidebarPresenter.Collections().Select(act.UID)
									}
									a.setActiveItem(a.findItem(act.UID))
								} else {
									if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
										a.SidebarPresenter.Collections().ClearSelection()
									}
									a.setActiveItem(nil)
								}
							}
						},
						OnNewTab: func() {
							a.NewRequest("")
						},
					})
				}
				if a.RequestPresenter != nil {
					pane := a.paneOf(a.RequestPresenter.Item())
					urlField := a.urlField
					screenState := a.RequestScreenState
					if pane != nil {
						urlField = pane.url
						screenState = pane.state
					}
					ui.Box(c).Grow(1).FillWidth().Children(func() {
						opts := request.Options{
							Resolves:       a.resolvesVariable,
							OnOpenVariable: a.EditVariable,
							Field:          urlField,
						}
						request.RequestScreenWith(c, a.RequestPresenter, opts, screenState)
					})
				}
			})
		},
		ResponsePane: func(c *ui.Context) {
			if a.ResponsePresenter != nil {
				response.Screen(c, response.Props{
					Presenter: a.ResponsePresenter,
					OnSelectTab: func(tab presenters.ResponseSubTab) {
						a.ResponsePresenter.SelectSubTab(tab)
					},
				})
			}
		},
	}
}
