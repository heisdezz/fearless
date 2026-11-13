package app

import (
	"sync"

	"github.com/egoist/mygo"
	"github.com/lobehub/lobe-mygo/pkg/field"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/model"
	"fearless/internal/platform"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/layout"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/environment"
	"fearless/internal/ui/screens/navigation"
	"fearless/internal/ui/screens/request"
	"fearless/internal/ui/screens/settings"
	"fearless/internal/ui/screens/shell"
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

	// panes holds one request/response pane per open tab, keyed by tab UID
	// (the item's UID). RequestPresenter and ResponsePresenter point at the
	// active tab's pane, so the rest of the app keeps reading those two fields.
	panes map[string]*requestPane

	// environmentState holds the environment editor's transient row state.
	environmentState *environment.State

	// sessionPath overrides the default session.json location for tests.
	sessionPath string

	// bus and requests carry the send path: requests publishes execution
	// events on bus, and the response presenter subscribes to bus so a
	// completed request reaches the response pane.
	bus      *bus.Bus
	requests *commands.RequestService
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
	a.wirePresenterInvalidation()
	a.subscribeResponseEvents()

	return a
}

// wirePresenterInvalidation repaints when a presenter changes.
//
// Presenters are the source of truth for what the screen shows, and they have
// no window of their own: without this, a menu command that mutates the model
// (Delete, Collapse, Rename, a new collection) updates the data and leaves the
// frame stale, so the command looks like it did nothing.
func (a *App) wirePresenterInvalidation() {
	invalidate := func() { a.SafeUpdate(func() {}) }
	for _, p := range []interface{ OnChanged(func()) }{
		a.CollectionPresenter,
		a.EnvironmentPresenter,
		a.SidebarPresenter,
		a.RequestPresenter,
		a.ResponsePresenter,
		a.TabsPresenter,
		a.PreferencesPresenter,
		a.SettingsPresenter,
	} {
		if p != nil {
			p.OnChanged(invalidate)
		}
	}
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
	if a.panes == nil {
		a.panes = make(map[string]*requestPane)
	}
	// The request and response presenters always point at some tab's pane, so
	// a headless App has a place to draw before a tab is opened. Opening a tab
	// swaps them for that tab's pane (see setActiveItem).
	if a.RequestPresenter == nil {
		a.RequestPresenter = a.newRequestPresenter()
	}
	if p, ok := a.panes[""]; !ok {
		a.panes[""] = &requestPane{
			request:  a.RequestPresenter,
			response: presenters.NewResponsePresenter(),
			url:      &widgets.TemplateField{},
			state:    &request.RequestScreenState{},
		}
	} else {
		_ = p
	}
	if a.ResponsePresenter == nil {
		a.ResponsePresenter = a.panes[""].response
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

// SafeUpdate marshals a UI mutation callback onto the MyGo UI loop via Window.Update.
// If the window is not yet created (headless or startup), it executes
// synchronously.
//
// The synchronous branch holds no lock while fn runs: a presenter notify inside
// fn can call back into SafeUpdate, and the mutex is not reentrant.
func (a *App) SafeUpdate(fn func()) {
	a.mu.Lock()
	win := a.Window
	a.mu.Unlock()

	if win != nil {
		win.Update(fn)
		return
	}
	fn()
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
