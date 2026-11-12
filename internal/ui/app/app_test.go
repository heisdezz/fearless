package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/core/model"
	"fearless/internal/platform"
	"fearless/internal/ui/dialogs"
	"fearless/internal/ui/presenters"
)

// Mock sender for request presenter testing
type mockRequestSender struct {
	sent bool
}

func (m *mockRequestSender) Send(ctx context.Context, itemUID string) (string, error) {
	m.sent = true
	return "req-1", nil
}

type mockRequestCanceller struct {
	cancelled bool
}

func (m *mockRequestCanceller) Cancel(ctx context.Context, requestUID string) error {
	m.cancelled = true
	return nil
}

type mockItemSaver struct {
	saved bool
}

func (m *mockItemSaver) Save(ctx context.Context, item *model.Item) error {
	m.saved = true
	return nil
}

func TestApp_InitializationAndOptions(t *testing.T) {
	router := NewKeybindingRouter()
	q := dialogs.NewQueue()
	services := platform.Services{}

	a := New(
		WithRouter(router),
		WithDialogQueue(q),
		WithServices(services),
	)

	if a.Router != router {
		t.Error("WithRouter option was not applied")
	}
	if a.DialogQueue != q {
		t.Error("WithDialogQueue option was not applied")
	}
	if a.ShellScreen == nil {
		t.Error("ShellScreen not initialized")
	}
	if a.NavScreen == nil {
		t.Error("NavScreen not initialized")
	}
	if a.RequestPresenter == nil {
		t.Error("RequestPresenter not initialized")
	}
	if a.ResponsePresenter == nil {
		t.Error("ResponsePresenter not initialized")
	}
	if a.SidebarPresenter == nil {
		t.Error("SidebarPresenter not initialized")
	}
}

func TestApp_SafeUpdate(t *testing.T) {
	a := New()

	// 1. Without window (synchronous fallback)
	var executed bool
	a.SafeUpdate(func() {
		executed = true
	})
	if !executed {
		t.Error("SafeUpdate failed to run callback when window is nil")
	}

	// 2. From background goroutines concurrent test
	var wg sync.WaitGroup
	count := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.SafeUpdate(func() {
				count++
			})
		}()
	}
	wg.Wait()
	if count != 20 {
		t.Errorf("expected count 20 from concurrent SafeUpdate, got %d", count)
	}
}

func TestApp_WrapUpload(t *testing.T) {
	a := New()

	var receivedFiles []field.File
	props := field.UploadProps{
		Multiple: true,
		OnFiles: func(files []field.File) {
			receivedFiles = files
		},
	}

	wrapped := a.WrapUpload(props)
	if wrapped.OnFiles == nil {
		t.Fatal("expected non-nil wrapped OnFiles")
	}

	testFiles := []field.File{
		{Name: "test.json", Path: "/tmp/test.json", Size: 42},
	}

	// Invoke from background goroutine
	done := make(chan struct{})
	go func() {
		wrapped.OnFiles(testFiles)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for wrapped OnFiles")
	}

	if len(receivedFiles) != 1 || receivedFiles[0].Name != "test.json" {
		t.Errorf("unexpected files received: %v", receivedFiles)
	}
}

func TestApp_ActionsAndNavigation(t *testing.T) {
	a := New()

	// Initial view
	if a.ActiveView != ViewKindMain {
		t.Errorf("expected ViewKindMain, got %v", a.ActiveView)
	}

	// Show preferences
	a.ShowPreferences()
	if a.ActiveView != ViewKindPreferences {
		t.Errorf("expected ViewKindPreferences, got %v", a.ActiveView)
	}

	// Toggle preferences
	a.TogglePreferences()
	if a.ActiveView != ViewKindMain {
		t.Errorf("expected ViewKindMain after toggle, got %v", a.ActiveView)
	}

	// Show collection settings
	a.ShowCollectionSettings()
	if a.ActiveView != ViewKindCollectionSettings {
		t.Errorf("expected ViewKindCollectionSettings, got %v", a.ActiveView)
	}

	// Show main
	a.ShowMain()
	if a.ActiveView != ViewKindMain {
		t.Errorf("expected ViewKindMain, got %v", a.ActiveView)
	}

	// Toggle sidebar
	origSidebar := a.LayoutState.SidebarExpanded
	a.ToggleSidebar()
	if a.LayoutState.SidebarExpanded == origSidebar {
		t.Error("expected SidebarExpanded to toggle")
	}

	// Toggle console
	origConsole := a.LayoutState.ConsoleOpen
	a.ToggleConsole()
	if a.LayoutState.ConsoleOpen == origConsole {
		t.Error("expected ConsoleOpen to toggle")
	}

	// Toggle layout (stacked <-> side-by-side)
	origOrientation := a.LayoutState.RequestResponseOrientation
	a.ToggleLayout()
	if a.LayoutState.RequestResponseOrientation == origOrientation {
		t.Error("expected RequestResponseOrientation to toggle")
	}
	a.ToggleLayout()
	if a.LayoutState.RequestResponseOrientation != origOrientation {
		t.Error("expected RequestResponseOrientation to toggle back")
	}
}

func TestApp_RequestLifecycleActions(t *testing.T) {
	sender := &mockRequestSender{}
	canceller := &mockRequestCanceller{}
	saver := &mockItemSaver{}

	reqPres := presenters.NewRequestPresenter(sender, canceller, saver)
	item := &model.Item{
		UID:  "item-123",
		Name: "Test Req",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://example.com/api",
		},
	}
	reqPres.SetItem(item)

	a := New()
	a.RequestPresenter = reqPres

	// Send
	a.SendActiveRequest()
	if !sender.sent {
		t.Error("SendActiveRequest did not invoke sender")
	}

	// Cancel
	a.CancelActiveRequest()
	if !canceller.cancelled {
		t.Error("CancelActiveRequest did not invoke canceller")
	}

	// Save
	reqPres.SetMethod("POST")
	a.SaveActiveRequest()
	if !saver.saved {
		t.Error("SaveActiveRequest did not invoke saver")
	}

	// New Request & New Folder via DialogQueue
	a.NewRequest("parent-1")
	if !a.DialogQueue.Active() {
		t.Error("NewRequest did not open dialog prompt")
	}
	a.DialogQueue.OK() // triggers submit handler
	a.DialogQueue.Cancel()

	a.NewFolder("parent-1")
	if !a.DialogQueue.Active() {
		t.Error("NewFolder did not open dialog prompt")
	}
	a.DialogQueue.OK() // triggers submit handler
	a.DialogQueue.Cancel()

	// Show Import
	a.ShowImport()
	if !a.DialogQueue.Active() {
		t.Error("ShowImport did not open import dialog")
	}
	a.DialogQueue.Cancel()

	// Close tab
	a.TabsPresenter.OpenTab(item)
	if a.TabsPresenter.ActiveTab() == nil {
		t.Fatal("expected active tab")
	}
	a.CloseActiveTab()
	if a.TabsPresenter.ActiveTab() != nil {
		t.Error("expected active tab to be closed")
	}
}

func TestApp_MenuActions(t *testing.T) {
	a := New()
	actions := a.MenuActions()

	// Test each menu action execution
	actions.OnNewRequest()
	a.DialogQueue.Cancel()

	actions.OnNewCollection()
	a.DialogQueue.Cancel()

	actions.OnImportCollection()
	a.DialogQueue.Cancel()

	actions.OnSave()
	actions.OnCloseTab()
	actions.OnSendRequest()
	actions.OnCancelRequest()
	actions.OnToggleSidebar()
	actions.OnToggleConsole()
	actions.OnPreferences()
	actions.OnDocumentation()
}

func TestApp_ScreenCallbacks(t *testing.T) {
	a := New()

	// Trigger shell actions
	a.ShellScreen.Actions.OnMenu()
	a.ShellScreen.Actions.OnWorkspace()
	a.ShellScreen.Actions.OnSettings()
	a.ShellScreen.Actions.OnSelectEnvironment("dev")
	a.ShellScreen.Actions.OnManageEnvironment()
	a.ShellScreen.Actions.OnRun()
	a.ShellScreen.Actions.OnStop()
	a.ShellScreen.Actions.OnConsoleTab(1)
	if a.activeConsoleTab != 1 {
		t.Errorf("expected console tab 1, got %d", a.activeConsoleTab)
	}
	a.ShellScreen.Actions.OnConsoleClose()
	if a.LayoutState.ConsoleOpen {
		t.Error("expected console closed")
	}

	// Trigger shell injected region functions
	tester := ui.NewTester(func(c *ui.Context) {
		a.ShellScreen.Sidebar(c, lobe.SideNavState{})
		a.ShellScreen.CollectionHeader(c)
		a.ShellScreen.RequestPane(c)
		a.ShellScreen.ResponsePane(c)
	}, 1280, 800)
	tester.Frame()

	// Trigger keybinding router handlers
	for _, b := range a.Router.Bindings() {
		if b.Handler != nil {
			b.Handler()
		}
	}
	a.DialogQueue.Cancel()
}

func TestApp_RootView_HeadlessSmoke(t *testing.T) {
	a := New()

	// 1. Smoke test: RootView in Main view
	tester := ui.NewTester(func(c *ui.Context) {
		a.RootView(c)
	}, 1280, 800)
	tester.Frame()

	// 2. Smoke test: RootView in Preferences view
	a.ShowPreferences()
	tester.Frame()

	// 3. Smoke test: RootView in CollectionSettings view
	a.ShowCollectionSettings()
	tester.Frame()

	// 4. Smoke test: RootView with active Dialog
	a.ShowMain()
	_, _ = a.DialogQueue.Confirm("Confirm Action", "Are you sure?", "Proceed", func(ctx context.Context, vals map[string]string) error {
		return nil
	})
	tester.Frame()
	a.DialogQueue.Cancel()
	tester.Frame()
}
