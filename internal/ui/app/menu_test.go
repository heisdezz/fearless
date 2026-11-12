package app

import (
	"testing"
)

func TestBuildMenu_StructureAndCallbacks(t *testing.T) {
	var (
		newReqCalled     bool
		newColCalled     bool
		importCalled     bool
		saveCalled       bool
		closeTabCalled   bool
		prefCalled       bool
		sendReqCalled    bool
		cancelReqCalled  bool
		toggleSideCalled bool
		changeLayCalled  bool
		toggleConsCalled bool
		zoomInCalled     bool
		zoomOutCalled    bool
		resetZoomCalled  bool
		toggleFullCalled bool
		aboutCalled      bool
		docCalled        bool
	)

	actions := MenuActions{
		OnNewRequest: func() {
			newReqCalled = true
		},
		OnNewCollection: func() {
			newColCalled = true
		},
		OnImportCollection: func() {
			importCalled = true
		},
		OnSave: func() {
			saveCalled = true
		},
		OnCloseTab: func() {
			closeTabCalled = true
		},
		OnPreferences: func() {
			prefCalled = true
		},
		OnSendRequest: func() {
			sendReqCalled = true
		},
		OnCancelRequest: func() {
			cancelReqCalled = true
		},
		OnToggleSidebar: func() {
			toggleSideCalled = true
		},
		OnChangeLayout: func() {
			changeLayCalled = true
		},
		OnToggleConsole: func() {
			toggleConsCalled = true
		},
		OnZoomIn: func() {
			zoomInCalled = true
		},
		OnZoomOut: func() {
			zoomOutCalled = true
		},
		OnResetZoom: func() {
			resetZoomCalled = true
		},
		OnToggleFullScreen: func() {
			toggleFullCalled = true
		},
		OnAbout: func() {
			aboutCalled = true
		},
		OnDocumentation: func() {
			docCalled = true
		},
	}

	router := DefaultKeybindingRouter()
	menu := BuildMenu(actions, router)
	if menu == nil {
		t.Fatal("expected non-nil menu")
	}

	topItems := menu.Items()
	if len(topItems) != 5 {
		t.Fatalf("expected 5 top-level menus (File, Edit, View, Request, Help), got %d", len(topItems))
	}

	expectedMenus := []string{"File", "Edit", "View", "Request", "Help"}
	for i, expected := range expectedMenus {
		if topItems[i].Label != expected {
			t.Errorf("expected menu %d to be %q, got %q", i, expected, topItems[i].Label)
		}
	}

	// Trigger callbacks in File menu
	fileMenu := topItems[0].Submenu
	for _, it := range fileMenu {
		if it.Click != nil {
			it.Click(it, nil)
		}
	}
	if !newReqCalled {
		t.Error("OnNewRequest not triggered")
	}
	if !newColCalled {
		t.Error("OnNewCollection not triggered")
	}
	if !importCalled {
		t.Error("OnImportCollection not triggered")
	}
	if !saveCalled {
		t.Error("OnSave not triggered")
	}
	if !closeTabCalled {
		t.Error("OnCloseTab not triggered")
	}
	if !prefCalled {
		t.Error("OnPreferences not triggered")
	}

	// Trigger callbacks in View menu
	viewMenu := topItems[2].Submenu
	for _, it := range viewMenu {
		if it.Click != nil {
			it.Click(it, nil)
		}
	}
	if !toggleSideCalled {
		t.Error("OnToggleSidebar not triggered")
	}
	if !changeLayCalled {
		t.Error("OnChangeLayout not triggered")
	}
	if !toggleConsCalled {
		t.Error("OnToggleConsole not triggered")
	}
	if !zoomInCalled {
		t.Error("OnZoomIn not triggered")
	}
	if !zoomOutCalled {
		t.Error("OnZoomOut not triggered")
	}
	if !resetZoomCalled {
		t.Error("OnResetZoom not triggered")
	}
	if !toggleFullCalled {
		t.Error("OnToggleFullScreen not triggered")
	}

	// Trigger callbacks in Request menu
	reqMenu := topItems[3].Submenu
	for _, it := range reqMenu {
		if it.Click != nil {
			it.Click(it, nil)
		}
	}
	if !sendReqCalled {
		t.Error("OnSendRequest not triggered")
	}
	if !cancelReqCalled {
		t.Error("OnCancelRequest not triggered")
	}

	// Trigger callbacks in Help menu
	helpMenu := topItems[4].Submenu
	for _, it := range helpMenu {
		if it.Click != nil {
			it.Click(it, nil)
		}
	}
	if !docCalled {
		t.Error("OnDocumentation not triggered")
	}
	if !aboutCalled {
		t.Error("OnAbout not triggered")
	}
}
