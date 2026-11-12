package navigation

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

const (
	testW = 1280
	testH = 800
)

type actionRecorder struct {
	sections       []presenters.SidebarSection
	collapseToggle int
	filters        []string
	selectedNodes  []string
	toggledNodes   []string
	openCollection int
	createRequest  []string
	createFolder   []string
	deleteItem     []string
	environments   []string
	manageEnvs     int
	runCollection  []string
}

func testScreen(rec *actionRecorder) *Screen {
	snap := Snapshot{
		ActiveSection:     presenters.SidebarSectionCollections,
		SidebarWidth:      260,
		CollectionName:    "My API",
		CollectionUID:     "col-1",
		Environments:      []string{"Staging", "Production"},
		ActiveEnvironment: "Staging",
		VisibleNodes: []*presenters.TreeNode{
			{
				UID:           "col-1",
				Name:          "My API",
				Level:         0,
				IsFolder:      true,
				IsExpanded:    true,
				CollectionUID: "col-1",
			},
			{
				UID:           "folder-1",
				Name:          "Users",
				Level:         1,
				IsFolder:      true,
				IsExpanded:    true,
				CollectionUID: "col-1",
			},
			{
				UID:           "req-1",
				Name:          "Get Users",
				Method:        "GET",
				Level:         2,
				IsFolder:      false,
				CollectionUID: "col-1",
			},
			{
				UID:           "req-2",
				Name:          "Create User",
				Method:        "POST",
				Level:         2,
				IsFolder:      false,
				IsDirty:       true,
				CollectionUID: "col-1",
			},
		},
	}

	actions := Actions{
		OnSelectSection: func(s presenters.SidebarSection) {
			rec.sections = append(rec.sections, s)
		},
		OnToggleCollapse: func() {
			rec.collapseToggle++
		},
		OnFilterChanged: func(q string) {
			rec.filters = append(rec.filters, q)
		},
		OnSelectNode: func(uid string) {
			rec.selectedNodes = append(rec.selectedNodes, uid)
		},
		OnToggleNode: func(uid string) {
			rec.toggledNodes = append(rec.toggledNodes, uid)
		},
		OnOpenCollection: func() {
			rec.openCollection++
		},
		OnCreateRequest: func(parentUID string) {
			rec.createRequest = append(rec.createRequest, parentUID)
		},
		OnCreateFolder: func(parentUID string) {
			rec.createFolder = append(rec.createFolder, parentUID)
		},
		OnDeleteItem: func(uid string) {
			rec.deleteItem = append(rec.deleteItem, uid)
		},
		OnSelectEnvironment: func(env string) {
			rec.environments = append(rec.environments, env)
		},
		OnManageEnvironments: func() {
			rec.manageEnvs++
		},
		OnRunCollection: func(colUID string) {
			rec.runCollection = append(rec.runCollection, colUID)
		},
	}

	return &Screen{
		Snapshot: snap,
		Actions:  actions,
	}
}

func newTester(s *Screen) *ui.Tester {
	tt := ui.NewTester(s.View, testW, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	return tt
}

// TestNavigationSmokeTest verifies headless rendering of sidebar explorer and header.
func TestNavigationSmokeTest(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	tt := newTester(s)

	// The section switcher is set in small caps; the tree and the toolbar
	// keep their prose.
	expectedTexts := []string{
		"COLLECTIONS", "HISTORY", "RUNNER",
		"+ Request", "+ Folder", "Open",
		"Users", "Get Users", "Create User",
	}

	for _, want := range expectedTexts {
		if !tt.HasText(want) {
			t.Errorf("navigation missing expected text %q; texts: %v", want, tt.Texts())
		}
	}
}

// TestCollectionHeaderStrip covers the optional environment strip: the main
// window does not mount it — the title strip owns the environment now — but
// it stays available for a window that embeds the sidebar on its own.
func TestCollectionHeaderStrip(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	tt := ui.NewTester(func(c *ui.Context) {
		s.CollectionHeader(c)
	}, testW, 200)
	tt.Frame()

	for _, want := range []string{"My API", "ENVIRONMENT", "Staging", "Environment settings", "Run"} {
		if !tt.HasText(want) {
			t.Errorf("collection header missing expected text %q; texts: %v", want, tt.Texts())
		}
	}

	if err := tt.Click("Environment settings"); err != nil {
		t.Fatalf("click the environment settings button: %v", err)
	}
	if rec.manageEnvs != 1 {
		t.Errorf("OnManageEnvironments: got %d, want 1", rec.manageEnvs)
	}

	if err := tt.Click("Run"); err != nil {
		t.Fatalf("click Run: %v", err)
	}
	if len(rec.runCollection) != 1 || rec.runCollection[0] != "col-1" {
		t.Errorf("OnRunCollection: got %v, want ['col-1']", rec.runCollection)
	}
}

// TestViewIsPure asserts that rendering never mutates the Snapshot and two frames are identical.
func TestViewIsPure(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	snapBefore := s.Snapshot
	tt := ui.NewTester(s.View, testW, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	first := encodePNG(t, tt.Image())

	tt.Frame()
	second := encodePNG(t, tt.Image())

	if !reflect.DeepEqual(s.Snapshot, snapBefore) {
		t.Errorf("Snapshot was mutated during rendering")
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("Two subsequent frames with unchanged data yielded different visuals")
	}
}

// TestActionsAreTheOnlyWrites verifies that interactive clicks correctly fire action callbacks.
func TestActionsAreTheOnlyWrites(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	tt := newTester(s)

	// Click section switcher to History
	if err := tt.Click("HISTORY"); err != nil {
		t.Fatalf("click HISTORY: %v", err)
	}
	if len(rec.sections) == 0 || rec.sections[len(rec.sections)-1] != presenters.SidebarSectionHistory {
		t.Errorf("expected OnSelectSection history, got %v", rec.sections)
	}

	// Click section switcher to Runner
	if err := tt.Click("RUNNER"); err != nil {
		t.Fatalf("click RUNNER: %v", err)
	}
	if len(rec.sections) == 0 || rec.sections[len(rec.sections)-1] != presenters.SidebarSectionRunner {
		t.Errorf("expected OnSelectSection runner, got %v", rec.sections)
	}

	// Click action buttons
	if err := tt.Click("+ Request"); err != nil {
		t.Fatalf("click + Request: %v", err)
	}
	if len(rec.createRequest) == 0 {
		t.Errorf("expected OnCreateRequest callback")
	}

	if err := tt.Click("+ Folder"); err != nil {
		t.Fatalf("click + Folder: %v", err)
	}
	if len(rec.createFolder) == 0 {
		t.Errorf("expected OnCreateFolder callback")
	}

	if err := tt.Click("Open"); err != nil {
		t.Fatalf("click Open: %v", err)
	}
	if rec.openCollection == 0 {
		t.Errorf("expected OnOpenCollection callback")
	}

	if err := tt.Click("Collapse sidebar"); err != nil {
		t.Fatalf("click Collapse sidebar: %v", err)
	}
	if rec.collapseToggle == 0 {
		t.Errorf("expected OnToggleCollapse callback")
	}
}

// TestRailCollapsedMode asserts proper icons and expand button when sidebar is collapsed.
func TestRailCollapsedMode(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	s.Snapshot.IsCollapsed = true

	tt := ui.NewTester(func(c *ui.Context) {
		s.Sidebar(c, lobe.SideNavState{})
	}, 64, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()

	// Text headers should NOT be visible in rail mode
	if tt.HasText("COLLECTIONS") {
		t.Errorf("COLLECTIONS label unexpectedly visible in rail mode")
	}

	// Tooltips / Labels for icon buttons should be registered
	if !tt.HasText("Collections") || !tt.HasText("History") || !tt.HasText("Runner") {
		t.Errorf("expected rail buttons to have labels for tooltips; texts: %v", tt.Texts())
	}

	// Clicking expand button fires OnToggleCollapse
	if err := tt.Click("Expand sidebar"); err != nil {
		t.Fatalf("click expand button in rail: %v", err)
	}
	if rec.collapseToggle != 1 {
		t.Errorf("expected OnToggleCollapse to be called once, got %d", rec.collapseToggle)
	}
}

// TestTakeSnapshotAndNewScreen verifies snapshot creation from real presenters.
func TestTakeSnapshotAndNewScreen(t *testing.T) {
	col := &model.Collection{
		UID:  "col-1",
		Name: "Test Collection",
		ActiveEnvironment: "Dev",
		Items: []*model.Item{
			{
				UID:  "req-1",
				Name: "First Req",
				Request: &model.Request{
					Method: "GET",
					URL:    "https://api.test",
				},
			},
		},
	}

	col.Environments = []*model.Environment{
		{UID: "env-1", Name: "Dev"},
	}
	colPres := presenters.NewCollectionPresenter(nil, nil, nil, nil)
	colPres.SetCollections([]*model.Collection{col})
	envPres := presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
	envPres.SetCollection(col)
	_ = envPres.SelectEnvironment(context.Background(), "Dev")
	sidebarPres := presenters.NewSidebarPresenter(colPres, envPres)

	snap := TakeSnapshot(sidebarPres)
	if snap.ActiveSection != presenters.SidebarSectionCollections {
		t.Errorf("expected ActiveSection collections, got %v", snap.ActiveSection)
	}
	if snap.CollectionName != "Test Collection" {
		t.Errorf("expected CollectionName Test Collection, got %q", snap.CollectionName)
	}
	if snap.ActiveEnvironment != "Dev" {
		t.Errorf("expected ActiveEnvironment Dev, got %q", snap.ActiveEnvironment)
	}
	if len(snap.VisibleNodes) == 0 {
		t.Errorf("expected visible nodes from collection presenter")
	}

	// NewScreen constructor
	screen := NewScreen(sidebarPres, Actions{})
	if screen == nil || screen.Snapshot.CollectionName != "Test Collection" {
		t.Fatal("NewScreen failed to populate snapshot")
	}
}

// TestEmptyCollectionState verifies empty state placeholder.
func TestEmptyCollectionState(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	s.Snapshot.VisibleNodes = nil // empty

	tt := newTester(s)
	if !tt.HasText("NO REQUESTS FOUND") {
		t.Errorf("expected empty state title, texts: %v", tt.Texts())
	}
	if !tt.HasText("Open an existing collection or create a new one") {
		t.Errorf("expected empty state hint, texts: %v", tt.Texts())
	}
}

// TestHistoryAndRunnerSections verifies secondary sections.
func TestHistoryAndRunnerSections(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	// History section
	s.Snapshot.ActiveSection = presenters.SidebarSectionHistory
	tt := newTester(s)
	if !tt.HasText("REQUEST HISTORY") {
		t.Errorf("expected Request History section title, texts: %v", tt.Texts())
	}

	// Runner section
	s.Snapshot.ActiveSection = presenters.SidebarSectionRunner
	tt = newTester(s)
	if !tt.HasText("COLLECTION RUNNER") {
		t.Errorf("expected Collection Runner section title, texts: %v", tt.Texts())
	}
}

// TestNilStateRendersDefaults asserts that nil presenter produces safe defaults.
func TestNilStateRendersDefaults(t *testing.T) {
	snap := TakeSnapshot(nil)
	if snap.ActiveSection != presenters.SidebarSectionCollections {
		t.Errorf("nil presenter should default to collections section, got %v", snap.ActiveSection)
	}

	s := &Screen{Snapshot: snap}
	tt := newTester(s)
	if !tt.HasText("COLLECTIONS") {
		t.Errorf("expected COLLECTIONS switcher tab on nil snapshot")
	}
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b strings.Builder
	// In-memory PNG encode to verify frame equality
	pr, pw, _ := os.Pipe()
	go func() {
		_ = png.Encode(pw, img)
		_ = pw.Close()
	}()
	buf := make([]byte, 1024)
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return []byte(b.String())
}

// TestNoDomainImports ensures ui/screens/navigation does not import internal/core/model or internal/app.
func TestNoDomainImports(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue // test files are allowed domain fixtures
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if strings.Contains(content, "\"fearless/internal/core/model\"") {
			t.Errorf("file %s illegally imports internal/core/model", f)
		}
		if strings.Contains(content, "\"fearless/internal/app/") {
			t.Errorf("file %s illegally imports internal/app/*", f)
		}
	}
}

// TestGoldenAfterThemeChange covers the dark and light goldens.
func TestGoldenAfterThemeChange(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	// Dark Mode Golden
	ttDark := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(lobetheme.Dark().Theme())
		s.View(c)
	}, testW, testH)
	ttDark.Frame()
	ttDark.Advance(600 * time.Millisecond)
	ttDark.Frame()
	compareGolden(t, "navigation-dark", ttDark.Image())

	// Light Mode Golden
	ttLight := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(lobetheme.Light().Theme())
		s.View(c)
	}, testW, testH)
	ttLight.Frame()
	ttLight.Advance(600 * time.Millisecond)
	ttLight.Frame()
	compareGolden(t, "navigation-light", ttLight.Image())
}

const goldenTolerance = 0.05

func compareGolden(t *testing.T, name string, got image.Image) {
	t.Helper()
	goldenPath := filepath.Join("testdata", name+".png")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll("testdata", 0755)
		out, err := os.Create(goldenPath)
		if err != nil {
			t.Fatalf("create golden %s: %v", goldenPath, err)
		}
		defer out.Close()
		if err := png.Encode(out, got); err != nil {
			t.Fatalf("encode golden %s: %v", goldenPath, err)
		}
		t.Logf("wrote %s", goldenPath)
		return
	}
	f, err := os.Open(goldenPath)
	if os.IsNotExist(err) {
		_ = os.MkdirAll("testdata", 0755)
		out, err := os.Create(goldenPath)
		if err != nil {
			t.Fatalf("create golden %s: %v", goldenPath, err)
		}
		defer out.Close()
		if err := png.Encode(out, got); err != nil {
			t.Fatalf("encode golden %s: %v", goldenPath, err)
		}
		t.Logf("wrote initial golden %s", goldenPath)
		return
	} else if err != nil {
		t.Fatalf("open golden %s: %v", goldenPath, err)
	}
	defer f.Close()

	want, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode golden %s: %v", goldenPath, err)
	}

	b := got.Bounds()
	if b != want.Bounds() {
		t.Fatalf("golden %s bounds mismatch: got %v, want %v", name, b, want.Bounds())
	}

	diffCount := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c1 := color.RGBAModel.Convert(got.At(x, y)).(color.RGBA)
			c2 := color.RGBAModel.Convert(want.At(x, y)).(color.RGBA)
			if c1 != c2 {
				diffCount++
			}
		}
	}

	total := float64(b.Dx() * b.Dy())
	ratio := float64(diffCount) / total
	if ratio > goldenTolerance {
		t.Errorf("golden %s diff ratio %.4f exceeds tolerance %.4f (%d differing pixels)",
			name, ratio, goldenTolerance, diffCount)
	}
}

func TestSidebar_HeaderMenuAndCollectionContextMenu(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	// 1. Initial render: has Collections title; Mock Servers dock is closed by default
	tt := newTester(s)
	if !tt.HasText("Collections") {
		t.Fatalf("expected header title Collections, texts: %v", tt.Texts())
	}
	if s.MockServersOpen {
		t.Fatalf("expected MockServersOpen to default to false")
	}
	if !tt.HasText("Mock Servers") {
		t.Fatalf("expected collapsed Mock Servers header label, texts: %v", tt.Texts())
	}
	if tt.HasText("No mock servers configured") {
		t.Fatalf("expected mock servers body to be hidden when collapsed")
	}

	// Open Mock Servers dock
	s.MockServersOpen = true
	tt.Frame()
	if !s.MockServersOpen {
		t.Fatalf("expected MockServersOpen to be true")
	}
	if !tt.HasText("Mock Servers") || !tt.HasText("No mock servers configured") {
		t.Fatalf("expected expanded Mock Servers dock content, texts: %v", tt.Texts())
	}

	// Close Mock Servers dock
	s.MockServersOpen = false
	tt.Frame()
	if s.MockServersOpen {
		t.Fatalf("expected MockServersOpen to be false again")
	}
	if tt.HasText("No mock servers configured") {
		t.Fatalf("expected mock servers body to be hidden after collapse")
	}

	// 2. Open header dropdown menu (sidebar.png)
	s.HeaderMenuOpen = true
	tt.Frame()
	if !tt.HasText("Create collection") || !tt.HasText("Open collection") || !tt.HasText("Import collection") {
		t.Fatalf("expected header dropdown menu items, texts: %v", tt.Texts())
	}

	// 3. Open collection row context menu (collection_with_menu_opened.png)
	s.HeaderMenuOpen = false
	s.OpenMenuUID = "col-1"
	tt.Frame()
	expectedMenuTexts := []string{
		"New Request",
		"New Folder",
		"Run",
		"Clone",
		"Rename",
		"Collapse",
		"Reveal in File Manager",
		"Create Mock server",
		"Beta",
		"Settings",
		"Open in Terminal",
		"Remove",
	}
	for _, expected := range expectedMenuTexts {
		if !tt.HasText(expected) {
			t.Errorf("expected collection context menu to contain %q, texts: %v", expected, tt.Texts())
		}
	}
}
