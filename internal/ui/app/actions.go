package app

import (
	"context"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/layout"
)

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

// NewRequest opens the dialog that composes a request: its type, name, method
// and URL.
func (a *App) NewRequest(parentUID string) {
	if a.DialogQueue == nil {
		return
	}
	_, _ = a.DialogQueue.NewRequestModal(func(_ context.Context, vals dialogs.NewRequestValues) error {
		a.SafeUpdate(func() {
			if a.CollectionPresenter == nil {
				return
			}
			method := vals.Method
			switch vals.Kind {
			case dialogs.RequestKindGraphQL:
				method = "POST"
			case dialogs.RequestKindGRPC, dialogs.RequestKindWebSocket:
				method = "GET"
			}
			item, _ := a.CollectionPresenter.CreateRequest(context.Background(), "", parentUID, vals.Name, method, vals.URL)
			if item == nil {
				return
			}
			if itemType := itemTypeForKind(vals.Kind); itemType != "" {
				item.Type = itemType
			}
			if a.TabsPresenter != nil {
				a.TabsPresenter.OpenTab(item)
			}
			a.setActiveItem(item)
		})
		return nil
	})
}

// itemTypeForKind maps the New Request dialog's choice onto the model.
func itemTypeForKind(kind dialogs.RequestKind) model.ItemType {
	switch kind {
	case dialogs.RequestKindGraphQL:
		return model.ItemGraphQLRequest
	case dialogs.RequestKindGRPC:
		return model.ItemGRPCRequest
	case dialogs.RequestKindWebSocket:
		return model.ItemWSRequest
	}
	return model.ItemHTTPRequest
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

// OpenCollection opens the import dialog.
//
// "Open" and "Open Collection" both land here: the import dialog is the one
// place that accepts a folder, a file (Bruno, Postman, Insomnia, OpenAPI), a
// URL, a git repository or a pasted document, so routing them at a folder-only
// picker would refuse the formats the dialog already handles.
func (a *App) OpenCollection() {
	a.ShowImport()
}

func (a *App) loadCollectionDir(dir string) {
	// The session is written inside the update, not after it: SafeUpdate posts
	// the callback to the UI loop and returns, so persisting on the next line
	// would read the presenter before the collection was added and record an
	// empty session.
	a.SafeUpdate(func() {
		added := false
		if a.CollectionPresenter != nil {
			col, err := collection.Load(dir)
			if err == nil && col != nil {
				cols := a.CollectionPresenter.Collections()
				for _, existing := range cols {
					if existing.Path == col.Path || existing.UID == col.UID {
						a.CollectionPresenter.Select(existing.UID)
						a.bindEnvironment(existing.UID)
						added = true
						break
					}
				}
				if !added {
					a.CollectionPresenter.SetCollections(append(cols, col))
					a.CollectionPresenter.Select(col.UID)
					a.bindEnvironment(col.UID)
					added = true
				}
			}
		}
		if added {
			a.persistSession()
		}
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
				a.dropPane(act.UID)
				if nextAct := a.TabsPresenter.ActiveTab(); nextAct != nil {
					if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
						a.SidebarPresenter.Collections().Select(nextAct.UID)
					}
					a.setActiveItem(a.findItem(nextAct.UID))
				} else {
					if a.SidebarPresenter != nil && a.SidebarPresenter.Collections() != nil {
						a.SidebarPresenter.Collections().ClearSelection()
					}
					a.setActiveItem(nil)
				}
			}
		}
	})
}

// ToggleSidebar resets the sidebar to its default width. The sidebar keeps a
// minimum width rather than collapsing, so there is no collapsed state to
// toggle; the shortcut and menu item still work as a "reset width" command.
func (a *App) ToggleSidebar() {
	a.SafeUpdate(func() {
		a.LayoutState.SidebarWidth = layout.DefaultLayoutState().SidebarWidth
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
