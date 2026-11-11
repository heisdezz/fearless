package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

func strPtr(s string) *string {
	return &s
}

func TestMethodBadge(t *testing.T) {
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "CUSTOM"} {
		view := func(c *ui.Context) {
			ui.Column(c).Children(func() {
				MethodBadge(c, method)
				MethodBadge(c, method, MethodBadgeProps{Size: 14, PaddingX: 8, PaddingY: 4, Radius: 5})
			})
		}

		tt := ui.NewTester(view, 300, 200)
		tt.Frame()

		if !tt.HasText(method) {
			t.Errorf("expected MethodBadge to display text %q", method)
		}
	}

	// Empty method defaults to GET
	tt := ui.NewTester(func(c *ui.Context) {
		MethodBadge(c, "")
	}, 200, 100)
	tt.Frame()
	if !tt.HasText("GET") {
		t.Error("expected empty method to default to GET")
	}
}

func TestKeyValueGrid(t *testing.T) {
	changed := false
	rows := []model.KeyValue{
		{Name: strPtr("Authorization"), Value: strPtr("Bearer token123"), Enabled: true},
		{Name: strPtr("Accept"), Value: strPtr("application/json"), Enabled: false, Description: strPtr("Content negotiation")},
	}

	view := func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			KeyValueGrid(c, &rows, KeyValueGridProps{
				ShowDescription: true,
				OnChanged: func() {
					changed = true
				},
			})
		})
	}

	tt := ui.NewTester(view, 800, 400)
	tt.Frame()

	// Verify header titles
	if !tt.HasText("KEY") || !tt.HasText("VALUE") || !tt.HasText("DESCRIPTION") {
		t.Error("expected table headers KEY, VALUE, DESCRIPTION to be visible")
	}

	// Verify accessibility and input values
	nodes := tt.AccessAll()
	if len(nodes) == 0 {
		t.Error("expected accessible elements in KeyValueGrid")
	}
	foundAuth := false
	foundToken := false
	foundAccept := false
	foundDesc := false
	for _, a := range nodes {
		if a.Value == "Authorization" {
			foundAuth = true
		}
		if a.Value == "Bearer token123" {
			foundToken = true
		}
		if a.Value == "Accept" {
			foundAccept = true
		}
		if a.Value == "Content negotiation" {
			foundDesc = true
		}
	}
	if !foundAuth || !foundToken || !foundAccept || !foundDesc {
		t.Errorf("expected row values in accessibility tree: auth=%v, token=%v, accept=%v, desc=%v",
			foundAuth, foundToken, foundAccept, foundDesc)
	}

	// Test Add Row button click
	err := tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("failed to click '+ Add Row': %v", err)
	}
	tt.Frame()

	if len(rows) != 3 {
		t.Errorf("expected 3 rows after adding row, got %d", len(rows))
	}
	if !changed {
		t.Error("expected OnChanged to be called when row is added")
	}

	// Test Delete row button click ("✕")
	changed = false
	err = tt.Click("✕")
	if err != nil {
		t.Fatalf("failed to click delete button: %v", err)
	}
	tt.Frame()

	if len(rows) != 2 {
		t.Errorf("expected 2 rows after deleting row, got %d", len(rows))
	}
	if !changed {
		t.Error("expected OnChanged to be called when row is deleted")
	}
}

func TestRequestTabBar(t *testing.T) {
	selectedUID := ""
	closedUID := ""
	newTabClicked := false

	tabs := []*presenters.TabItem{
		{UID: "tab-1", Title: "Get Users", Method: "GET", IsDirty: false},
		{UID: "tab-2", Title: "Create User", Method: "POST", IsDirty: true},
	}

	view := func(c *ui.Context) {
		RequestTabBar(c, tabs, "tab-1", TabBarProps{
			OnSelect: func(uid string) {
				selectedUID = uid
			},
			OnClose: func(uid string) {
				closedUID = uid
			},
			OnNewTab: func() {
				newTabClicked = true
			},
		})
	}

	tt := ui.NewTester(view, 800, 100)
	tt.Frame()

	// Verify texts
	if !tt.HasText("Get Users") || !tt.HasText("Create User") {
		t.Error("expected tab titles to be visible")
	}
	if !tt.HasText("+") {
		t.Error("expected new tab '+' button to be visible")
	}

	// Click tab-2 to select
	err := tt.Click("Create User")
	if err != nil {
		t.Fatalf("failed to click tab: %v", err)
	}
	if selectedUID != "tab-2" {
		t.Errorf("expected selectedUID to be 'tab-2', got %q", selectedUID)
	}

	// Click close button on tab
	err = tt.Click("✕")
	if err != nil {
		t.Fatalf("failed to click tab close: %v", err)
	}
	if closedUID != "tab-1" {
		t.Errorf("expected closedUID to be 'tab-1', got %q", closedUID)
	}

	// Click new tab "+" button
	err = tt.Click("+")
	if err != nil {
		t.Fatalf("failed to click '+': %v", err)
	}
	if !newTabClicked {
		t.Error("expected newTabClicked to be true")
	}
}

func TestCollectionTreeRow(t *testing.T) {
	toggledUID := ""
	selectedUID := ""

	folderNode := &presenters.TreeNode{
		UID:        "folder-1",
		Name:       "Auth Endpoints",
		IsFolder:   true,
		IsExpanded: false,
		Level:      0,
	}

	reqNode := &presenters.TreeNode{
		UID:        "req-1",
		Name:       "Login",
		Method:     "POST",
		IsFolder:   false,
		IsSelected: true,
		IsDirty:    true,
		Level:      1,
	}

	view := func(c *ui.Context) {
		ui.Column(c).Children(func() {
			CollectionTreeRow(c, folderNode, TreeRowProps{
				OnToggle: func(uid string) {
					toggledUID = uid
				},
				OnSelect: func(uid string) {
					selectedUID = uid
				},
			})
			CollectionTreeRow(c, reqNode, TreeRowProps{
				OnToggle: func(uid string) {
					toggledUID = uid
				},
				OnSelect: func(uid string) {
					selectedUID = uid
				},
			})
			CollectionTreeRow(c, nil, TreeRowProps{})
		})
	}

	tt := ui.NewTester(view, 400, 300)
	tt.Frame()

	if !tt.HasText("Auth Endpoints") || !tt.HasText("Login") {
		t.Error("expected node names to be rendered")
	}
	if !tt.HasText("▸") {
		t.Error("expected collapsed chevron '▸' for folder")
	}

	// Click folder chevron to toggle
	err := tt.Click("▸")
	if err != nil {
		t.Fatalf("failed to click folder chevron: %v", err)
	}
	if toggledUID != "folder-1" {
		t.Errorf("expected toggledUID to be 'folder-1', got %q", toggledUID)
	}

	// Click request row to select
	err = tt.Click("Login")
	if err != nil {
		t.Fatalf("failed to click request row: %v", err)
	}
	if selectedUID != "req-1" {
		t.Errorf("expected selectedUID to be 'req-1', got %q", selectedUID)
	}
}

func TestVariableInput(t *testing.T) {
	inputVal := "https://api.example.com/{{env"
	changedVal := ""

	view := func(c *ui.Context) {
		VariableInput(c, &inputVal, VariableInputProps{
			Placeholder: "Enter URL",
			Variables:   []string{"baseUrl", "apiKey"},
			OnChanged: func(newVal string) {
				changedVal = newVal
			},
		})
	}

	tt := ui.NewTester(view, 400, 200)
	tt.Frame()

	// Initially no "{{", suggestions shouldn't show
	if tt.HasText("Available Variables:") {
		t.Error("suggestions should not show when input doesn't end with '{{'")
	}

	// Trigger suggestions with ending "{{"
	inputVal = "https://api.example.com/{{"
	tt.Frame()

	if !tt.HasText("Available Variables:") {
		t.Error("expected 'Available Variables:' dropdown when ending with '{{'")
	}
	if !tt.HasText("{{baseUrl}}") || !tt.HasText("{{apiKey}}") {
		t.Error("expected variable suggestions to be visible")
	}

	// Click suggestion
	err := tt.Click("{{baseUrl}}")
	if err != nil {
		t.Fatalf("failed to click suggestion: %v", err)
	}

	expectedResult := "https://api.example.com/{{baseUrl}}"
	if inputVal != expectedResult {
		t.Errorf("expected input to become %q, got %q", expectedResult, inputVal)
	}
	if changedVal != expectedResult {
		t.Errorf("expected changedVal to be %q, got %q", expectedResult, changedVal)
	}
}

func TestSplitter(t *testing.T) {
	ratio := float32(0.4)

	view := func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Splitter(c, &ratio, SplitHorizontal, func(c *ui.Context) {
				ui.Text(c, "Left Pane")
			}, func(c *ui.Context) {
				ui.Text(c, "Right Pane")
			})
		})
	}

	tt := ui.NewTester(view, 600, 400)
	tt.Frame()

	if !tt.HasText("Left Pane") || !tt.HasText("Right Pane") {
		t.Error("expected both split pane contents to be rendered")
	}

	// Test vertical split with default nil ratio
	ttVert := ui.NewTester(func(c *ui.Context) {
		Splitter(c, nil, SplitVertical, func(c *ui.Context) {
			ui.Text(c, "Top Pane")
		}, func(c *ui.Context) {
			ui.Text(c, "Bottom Pane")
		})
	}, 600, 400)
	ttVert.Frame()

	if !ttVert.HasText("Top Pane") || !ttVert.HasText("Bottom Pane") {
		t.Error("expected top and bottom vertical split panes to be rendered")
	}
}

func TestVirtualizedList(t *testing.T) {
	renderedIndices := make(map[int]bool)

	view := func(c *ui.Context) {
		VirtualizedList(c, VirtualizedListProps{
			ItemCount:     1000,
			ItemHeight:    30,
			VisibleHeight: 150,
			ScrollOffset:  300, // starting at row 10
			Buffer:        2,
			RenderItem: func(c *ui.Context, index int) {
				renderedIndices[index] = true
				ui.Text(c, "Item")
			},
		})
	}

	tt := ui.NewTester(view, 400, 300)
	tt.Frame()

	// Offset 300 / 30 = 10. Buffer 2 => startIndex = 8.
	// VisibleCount = 150/30 + 1 + 4 = 10 => endIndex = 18.
	// Rows rendered should be roughly 8..17. Far row 500 should NOT be rendered!
	if renderedIndices[500] {
		t.Error("virtualized list rendered item 500 which is out of view")
	}
	if !renderedIndices[10] {
		t.Error("virtualized list should have rendered visible item 10")
	}

	// Empty list / invalid height
	ttEmpty := ui.NewTester(func(c *ui.Context) {
		VirtualizedList(c, VirtualizedListProps{ItemCount: 0, ItemHeight: 30})
	}, 200, 100)
	ttEmpty.Frame()
}

func TestFreeTextSelect(t *testing.T) {
	val := "app"
	selectedVal := ""
	options := []string{"application/json", "application/xml", "text/plain", "multipart/form-data"}

	view := func(c *ui.Context) {
		FreeTextSelect(c, &val, FreeTextSelectProps{
			Placeholder: "Content-Type",
			Options:     options,
			Size:        lobetheme.Small,
			OnSelect: func(v string) {
				selectedVal = v
			},
		})
	}

	tt := ui.NewTester(view, 400, 300)
	tt.Frame()

	// Should show matching dropdown items ("application/json", "application/xml")
	if !tt.HasText("application/json") || !tt.HasText("application/xml") {
		t.Error("expected matching options in dropdown")
	}
	// "text/plain" should not match "app"
	if tt.HasText("text/plain") {
		t.Error("non-matching option 'text/plain' should not be displayed in dropdown")
	}

	// Click an option
	err := tt.Click("application/json")
	if err != nil {
		t.Fatalf("failed to click option: %v", err)
	}

	if val != "application/json" {
		t.Errorf("expected val to be 'application/json', got %q", val)
	}
	if selectedVal != "application/json" {
		t.Errorf("expected selectedVal to be 'application/json', got %q", selectedVal)
	}
}

func TestCodeEditor(t *testing.T) {
	var state *editor.State
	view := func(c *ui.Context) {
		CodeEditor(c, &state, `{"key": "value"}`, CodeEditorProps{
			Language:  "json",
			Variables: []string{"token", "id"},
		})
	}

	tt := ui.NewTester(view, 500, 300)
	tt.Frame()

	if state == nil {
		t.Fatal("expected CodeEditor to initialize editor state")
	}
	if state.Buffer.String() != `{"key": "value"}` {
		t.Errorf("unexpected buffer content: %q", state.Buffer.String())
	}
	if state.OnCompletion == nil {
		t.Error("expected OnCompletion handler to be configured for variables")
	}

	// Trigger completion
	state.OnCompletion(editor.CompletionRequest{})
}
