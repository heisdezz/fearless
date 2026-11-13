package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/screens/environment"
	"fearless/internal/ui/screens/navigation"
	"fearless/internal/ui/screens/settings"
	"fearless/internal/ui/screens/shell"
	"fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

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
