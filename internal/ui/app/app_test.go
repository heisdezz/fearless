package app

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/core/model"
	"fearless/internal/platform"
	"fearless/internal/ui/dialogs"
)

func TestApp_Lifecycle(t *testing.T) {
	// Construct App with custom router, queue, services
	router := DefaultKeybindingRouter()
	queue := dialogs.NewQueue()
	services := platform.Services{}

	a := New(
		WithRouter(router),
		WithDialogQueue(queue),
		WithServices(services),
	)

	if a.Router != router {
		t.Error("WithRouter option failed")
	}
	if a.DialogQueue != queue {
		t.Error("WithDialogQueue option failed")
	}
	if a.LayoutState.SidebarWidth != 248 {
		t.Errorf("expected default sidebar width 248, got %f", a.LayoutState.SidebarWidth)
	}

	// Verify all presenters are initialized
	if a.CollectionPresenter == nil {
		t.Error("CollectionPresenter is nil")
	}
	if a.EnvironmentPresenter == nil {
		t.Error("EnvironmentPresenter is nil")
	}
	if a.SidebarPresenter == nil {
		t.Error("SidebarPresenter is nil")
	}
	if a.RequestPresenter == nil {
		t.Error("RequestPresenter is nil")
	}
	if a.ResponsePresenter == nil {
		t.Error("ResponsePresenter is nil")
	}
	if a.TabsPresenter == nil {
		t.Error("TabsPresenter is nil")
	}
	if a.PreferencesPresenter == nil {
		t.Error("PreferencesPresenter is nil")
	}
	if a.SettingsPresenter == nil {
		t.Error("SettingsPresenter is nil")
	}

	// Verify screens
	if a.ShellScreen == nil {
		t.Error("ShellScreen is nil")
	}
	if a.NavScreen == nil {
		t.Error("NavScreen is nil")
	}
}

func TestApp_ViewNavigation(t *testing.T) {
	a := New()

	if a.ActiveView != ViewKindMain {
		t.Errorf("initial view: got %v, want ViewKindMain", a.ActiveView)
	}

	a.ShowPreferences()
	if a.ActiveView != ViewKindPreferences {
		t.Errorf("after ShowPreferences: got %v, want ViewKindPreferences", a.ActiveView)
	}

	a.TogglePreferences()
	if a.ActiveView != ViewKindMain {
		t.Errorf("after TogglePreferences: got %v, want ViewKindMain", a.ActiveView)
	}

	a.TogglePreferences()
	if a.ActiveView != ViewKindPreferences {
		t.Errorf("after second TogglePreferences: got %v, want ViewKindPreferences", a.ActiveView)
	}

	a.ShowCollectionSettings()
	if a.ActiveView != ViewKindCollectionSettings {
		t.Errorf("after ShowCollectionSettings: got %v, want ViewKindCollectionSettings", a.ActiveView)
	}

	a.ShowMain()
	if a.ActiveView != ViewKindMain {
		t.Errorf("after ShowMain: got %v, want ViewKindMain", a.ActiveView)
	}
}

func TestApp_LayoutToggles(t *testing.T) {
	a := New()

	// Sidebar expanded toggle
	initSidebar := a.LayoutState.SidebarExpanded
	a.ToggleSidebar()
	if a.LayoutState.SidebarExpanded == initSidebar {
		t.Error("ToggleSidebar did not flip SidebarExpanded")
	}

	// Orientation toggle
	initOrient := a.LayoutState.RequestResponseOrientation
	a.ToggleLayout()
	if a.LayoutState.RequestResponseOrientation == initOrient {
		t.Error("ToggleLayout did not flip orientation")
	}
	a.ToggleLayout()
	if a.LayoutState.RequestResponseOrientation != initOrient {
		t.Error("ToggleLayout second call did not restore initial orientation")
	}

	// Console toggle
	initConsole := a.LayoutState.ConsoleOpen
	a.ToggleConsole()
	if a.LayoutState.ConsoleOpen == initConsole {
		t.Error("ToggleConsole did not flip ConsoleOpen")
	}
}

func TestApp_ActionsAndDialogs(t *testing.T) {
	a := New()

	// Set up model item for request
	col := &model.Collection{
		UID:  "col-1",
		Name: "Test Collection",
		Items: []*model.Item{
			{
				UID:  "req-1",
				Name: "Get Users",
				Type: model.ItemHTTPRequest,
				Request: &model.Request{
					Name:   "Get Users",
					Method: "GET",
					URL:    "https://api.example.com/users",
				},
			},
		},
	}
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	item := col.Items[0]

	// Send request smoke
	a.RequestPresenter.SetItem(item)
	a.SendActiveRequest()
	a.CancelActiveRequest()
	a.SaveActiveRequest()

	waitFor := func(desc string, fn func() bool) {
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if fn() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timeout waiting for %s", desc)
	}

	// Create Collection via DialogQueue
	a.CreateCollection()
	if !a.DialogQueue.Active() {
		t.Error("CreateCollection did not open dialog")
	}
	a.DialogQueue.SetValue("name", "New Col")
	a.DialogQueue.SetValue("location", "/tmp")
	a.DialogQueue.SetValue("format", "bruno")
	a.DialogQueue.OK()

	waitFor("create collection to finish", func() bool {
		return !a.DialogQueue.Active() && len(a.CollectionPresenter.Collections()) >= 2
	})
	cols := a.CollectionPresenter.Collections()
	var foundNew bool
	for _, c := range cols {
		if c.Name == "New Col" {
			foundNew = true
			break
		}
	}
	if !foundNew {
		t.Error("New Col not found in collection presenter after creation")
	}

	// Open Collection via prompt fallback
	a.OpenCollection()
	if !a.DialogQueue.Active() {
		t.Error("OpenCollection did not open dialog prompt")
	}
	a.DialogQueue.SetValue("Folder Path", "")
	a.DialogQueue.OK() // triggers validation error, stays active
	if !a.DialogQueue.Active() {
		t.Error("empty Folder Path should keep dialog active")
	}
	a.DialogQueue.SetValue("Folder Path", "/nonexistent/test/path")
	a.DialogQueue.OK()
	a.DialogQueue.Cancel()

	// New Request via DialogQueue
	a.NewRequest("")
	if !a.DialogQueue.Active() {
		t.Error("NewRequest did not open dialog prompt")
	}
	a.DialogQueue.SetValue("Request Name", "My Test Request")
	a.DialogQueue.OK()

	waitFor("new request to finish", func() bool {
		return !a.DialogQueue.Active() && a.RequestPresenter.Item() != nil && a.RequestPresenter.Item().Name == "My Test Request"
	})
	if a.TabsPresenter.ActiveTab() == nil || a.TabsPresenter.ActiveTab().Title != "My Test Request" {
		t.Errorf("request was not opened in TabsPresenter: %v", a.TabsPresenter.ActiveTab())
	}

	a.NewFolder("")
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
	a.TabsPresenter.CloseAll()
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

func TestApp_SelectNode_LoadsRequestAndOpensTab(t *testing.T) {
	a := New()

	reqItem := &model.Item{
		UID:  "nested-req-1",
		Name: "Nested Get Users",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Name:   "Nested Get Users",
			Method: "GET",
			URL:    "https://api.example.com/nested/users",
		},
	}
	folderItem := &model.Item{
		UID:      "folder-nested",
		Name:     "Nested Folder",
		Type:     model.ItemFolder,
		Children: []*model.Item{reqItem},
	}
	col := &model.Collection{
		UID:   "col-nested",
		Name:  "Imported Nested API",
		Items: []*model.Item{folderItem},
	}

	a.CollectionPresenter.SetCollections([]*model.Collection{col})

	// 1. Click on folder node: should toggle expanded
	if a.CollectionPresenter.IsExpanded("folder-nested") {
		t.Error("expected folder to start collapsed")
	}
	a.NavScreen.Actions.OnSelectNode("folder-nested")
	if !a.CollectionPresenter.IsExpanded("folder-nested") {
		t.Error("expected folder to be expanded after OnSelectNode")
	}

	// 2. Click on request node: should select node, set item in RequestPresenter, and open tab
	a.NavScreen.Actions.OnSelectNode("nested-req-1")
	if a.CollectionPresenter.SelectedUID() != "nested-req-1" {
		t.Errorf("expected selectedUID nested-req-1, got %q", a.CollectionPresenter.SelectedUID())
	}
	if a.RequestPresenter.Item() != reqItem {
		t.Errorf("expected RequestPresenter item %v, got %v", reqItem, a.RequestPresenter.Item())
	}
	if a.TabsPresenter.ActiveTab() == nil || a.TabsPresenter.ActiveTab().UID != "nested-req-1" {
		t.Errorf("expected active tab nested-req-1, got %v", a.TabsPresenter.ActiveTab())
	}

	// 3. Render RequestPane: tab strip and request screen render cleanly
	tt := ui.NewTester(func(c *ui.Context) {
		a.ShellScreen.RequestPane(c)
	}, 800, 600)
	tt.Frame()

	if !tt.HasText("Nested Get Users") {
		t.Errorf("expected RequestPane to render tab with title, texts: %v", tt.Texts())
	}
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

func TestApp_CrenPostmanCollection_ImportAndSelect(t *testing.T) {
	fixturePath := "../../../test_collection/Cren.postman_collection.json"
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skipf("skipping: fixture not found at %s", fixturePath)
	}

	imported, err := dialogs.ParseImport(dialogs.ImportAuto, fixturePath)
	if err != nil {
		t.Fatalf("ParseImport failed on Cren collection: %v", err)
	}
	if imported.Collection == nil {
		t.Fatalf("expected imported Collection, got nil")
	}
	if imported.Collection.Name != "Cren" {
		t.Errorf("expected collection name 'Cren', got %q", imported.Collection.Name)
	}

	a := New()
	a.CollectionPresenter.SetCollections([]*model.Collection{imported.Collection})

	// Verify the collection presenter holds the imported collection
	cols := a.CollectionPresenter.Collections()
	if len(cols) == 0 {
		t.Fatal("expected at least 1 collection in presenter")
	}

	// Target request item from deeply nested structure:
	// Cren -> api -> get -> member -> Auth Controller get Members Get
	var targetItem *model.Item
	var findItemByName func(items []*model.Item, name string) *model.Item
	findItemByName = func(items []*model.Item, name string) *model.Item {
		for _, it := range items {
			if it.Name == name {
				return it
			}
			if len(it.Children) > 0 {
				if found := findItemByName(it.Children, name); found != nil {
					return found
				}
			}
		}
		return nil
	}

	targetItem = findItemByName(imported.Collection.Items, "Auth Controller get Members Get")
	if targetItem == nil {
		t.Fatal("target item 'Auth Controller get Members Get' not found in imported collection")
	}

	// 1. Select the nested node via NavScreen Actions
	a.NavScreen.Actions.OnSelectNode(targetItem.UID)

	// 2. Assert CollectionPresenter selected UID
	if a.CollectionPresenter.SelectedUID() != targetItem.UID {
		t.Errorf("selected UID = %q, want %q", a.CollectionPresenter.SelectedUID(), targetItem.UID)
	}

	// 3. Assert RequestPresenter is populated with targetItem
	if a.RequestPresenter.Item() != targetItem {
		t.Errorf("RequestPresenter item = %v, want targetItem %v", a.RequestPresenter.Item(), targetItem)
	}
	req := a.RequestPresenter.EffectiveRequest()
	if req == nil {
		t.Fatal("expected EffectiveRequest, got nil")
	}
	if req.Method != "GET" {
		t.Errorf("RequestPresenter method = %q, want GET", req.Method)
	}
	if req.URL != "{{baseUrl}}/api/get/member" {
		t.Errorf("RequestPresenter URL = %q, want {{baseUrl}}/api/get/member", req.URL)
	}

	// 4. Assert TabsPresenter opened the tab
	activeTab := a.TabsPresenter.ActiveTab()
	if activeTab == nil {
		t.Fatal("expected active tab, got nil")
	}
	if activeTab.UID != targetItem.UID {
		t.Errorf("active tab UID = %q, want %q", activeTab.UID, targetItem.UID)
	}
	if activeTab.Title != "Auth Controller get Members Get" {
		t.Errorf("active tab Title = %q, want 'Auth Controller get Members Get'", activeTab.Title)
	}

	// 5. Verify headless render of RootView and RequestPane with tab bar
	tester := ui.NewTester(func(c *ui.Context) {
		a.RootView(c)
	}, 1280, 800)
	tester.Frame()

	// 6. Verify tab title rendered in RequestPane
	rpTester := ui.NewTester(func(c *ui.Context) {
		a.ShellScreen.RequestPane(c)
	}, 800, 600)
	rpTester.Frame()
	if !rpTester.HasText("Auth Controller get Members Get") {
		t.Errorf("expected RequestPane to render tab with title, texts: %v", rpTester.Texts())
	}
}

func TestApp_LobeDropdownMenu(t *testing.T) {
	a := New()
	items := a.buildLobeMenuItems()
	if len(items) == 0 {
		t.Fatal("expected non-empty menu items")
	}

	// Verify key items exist
	keys := make(map[string]bool)
	for _, it := range items {
		if !it.Divider {
			keys[it.Key] = true
		}
	}
	expectedKeys := []string{
		"new-request", "new-collection", "import-collection",
		"save", "close-tab", "send-request", "cancel-request",
		"toggle-sidebar", "toggle-console", "change-layout",
		"preferences", "documentation", "about", "quit",
	}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected menu item with key %q", k)
		}
	}

	// Test handleMenuKey
	initialSidebar := a.LayoutState.SidebarExpanded
	a.handleMenuKey("toggle-sidebar")
	if a.LayoutState.SidebarExpanded == initialSidebar {
		t.Error("expected toggle-sidebar to flip sidebar state")
	}

	initialLayout := a.LayoutState.RequestResponseOrientation
	a.handleMenuKey("change-layout")
	if a.LayoutState.RequestResponseOrientation == initialLayout {
		t.Error("expected change-layout to flip layout orientation")
	}

	initialConsole := a.LayoutState.ConsoleOpen
	a.handleMenuKey("toggle-console")
	if a.LayoutState.ConsoleOpen == initialConsole {
		t.Error("expected toggle-console to flip console state")
	}

	// Verify rendering dropdown menu when titleMenuOpen is true
	a.titleMenuOpen = true
	tt := ui.NewTester(func(c *ui.Context) {
		a.RootView(c)
	}, 1280, 800)
	tt.Frame()

	for _, label := range []string{"New Request", "New Collection", "Save", "Toggle Sidebar"} {
		if !tt.HasText(label) {
			t.Errorf("expected dropdown menu to render text %q, texts: %v", label, tt.Texts())
		}
	}
}

