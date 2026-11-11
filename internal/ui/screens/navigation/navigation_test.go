package navigation

import (
	"bytes"
	"go/parser"
	"go/token"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

const (
	testW           = 1280
	testH           = 800
	goldenTolerance = 0.05
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
		IsCollapsed:       false,
		SidebarWidth:      260,
		FilterQuery:       "",
		SelectedUID:       "req-1",
		CollectionName:    "My API",
		CollectionUID:     "col-1",
		Environments:      []string{"Staging", "Production"},
		ActiveEnvironment: "Staging",
		VisibleNodes: []*presenters.TreeNode{
			{
				UID:           "col-1",
				Name:          "My API",
				Type:          "folder",
				Level:         0,
				IsFolder:      true,
				IsExpanded:    true,
				CollectionUID: "col-1",
			},
			{
				UID:           "folder-1",
				Name:          "Users",
				Type:          "folder",
				Level:         1,
				IsFolder:      true,
				IsExpanded:    true,
				CollectionUID: "col-1",
			},
			{
				UID:           "req-1",
				Name:          "Get Users",
				Type:          "http-request",
				Method:        "GET",
				Level:         2,
				IsFolder:      false,
				IsSelected:    true,
				CollectionUID: "col-1",
			},
			{
				UID:           "req-2",
				Name:          "Create User",
				Type:          "http-request",
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

	expectedTexts := []string{
		"My API", "Collections", "History", "Runner",
		"+ Request", "+ Folder", "Open",
		"Users", "Get Users", "Create User",
		"Env:", "Staging", "⚙ Env", "Run",
	}

	for _, want := range expectedTexts {
		if !tt.HasText(want) {
			t.Errorf("navigation missing expected text %q; texts: %v", want, tt.Texts())
		}
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
		t.Errorf("View mutated the Snapshot:\n got %+v\nwant %+v", s.Snapshot, snapBefore)
	}

	if !bytes.Equal(first, second) {
		t.Error("two consecutive frames of the same Screen produced different pixel outputs")
	}
}

// TestActionsAreTheOnlyWrites verifies that user interactions invoke only the Actions callbacks.
func TestActionsAreTheOnlyWrites(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	tt := newTester(s)

	// Click on + Request
	if err := tt.Click("+ Request"); err != nil {
		t.Fatalf("click + Request: %v", err)
	}
	if len(rec.createRequest) != 1 || rec.createRequest[0] != "req-1" {
		t.Errorf("OnCreateRequest: got %v, want ['req-1']", rec.createRequest)
	}

	// Click on + Folder
	if err := tt.Click("+ Folder"); err != nil {
		t.Fatalf("click + Folder: %v", err)
	}
	if len(rec.createFolder) != 1 || rec.createFolder[0] != "req-1" {
		t.Errorf("OnCreateFolder: got %v, want ['req-1']", rec.createFolder)
	}

	// Click on Open
	if err := tt.Click("Open"); err != nil {
		t.Fatalf("click Open: %v", err)
	}
	if rec.openCollection != 1 {
		t.Errorf("OnOpenCollection: got %d, want 1", rec.openCollection)
	}

	// Click on Section Tab: History
	if err := tt.Click("History"); err != nil {
		t.Fatalf("click History tab: %v", err)
	}
	if len(rec.sections) != 1 || rec.sections[0] != presenters.SidebarSectionHistory {
		t.Errorf("OnSelectSection: got %v, want [history]", rec.sections)
	}

	// Click on Section Tab: Runner
	if err := tt.Click("Runner"); err != nil {
		t.Fatalf("click Runner tab: %v", err)
	}
	if len(rec.sections) != 2 || rec.sections[1] != presenters.SidebarSectionRunner {
		t.Errorf("OnSelectSection: got %v, want runner as second entry", rec.sections)
	}

	// Click on Manage Env button
	if err := tt.Click("⚙ Env"); err != nil {
		t.Fatalf("click ⚙ Env: %v", err)
	}
	if rec.manageEnvs != 1 {
		t.Errorf("OnManageEnvironments: got %d, want 1", rec.manageEnvs)
	}

	// Click on Run button
	if err := tt.Click("Run"); err != nil {
		t.Fatalf("click Run: %v", err)
	}
	if len(rec.runCollection) != 1 || rec.runCollection[0] != "col-1" {
		t.Errorf("OnRunCollection: got %v, want ['col-1']", rec.runCollection)
	}

	// Click on a node row
	if err := tt.Click("Create User"); err != nil {
		t.Fatalf("click node: %v", err)
	}
	if len(rec.selectedNodes) != 1 || rec.selectedNodes[0] != "req-2" {
		t.Errorf("OnSelectNode: got %v, want ['req-2']", rec.selectedNodes)
	}

	// Click on folder toggle chevron
	if err := tt.Click("▾"); err != nil {
		t.Fatalf("click chevron ▾: %v", err)
	}
	if len(rec.toggledNodes) == 0 {
		t.Error("OnToggleNode not triggered when chevron clicked")
	}

	// Click on collapse toggle button in expanded header
	if err := tt.Click("«"); err != nil {
		t.Fatalf("click « collapse: %v", err)
	}
	if rec.collapseToggle != 1 {
		t.Errorf("OnToggleCollapse: got %d, want 1", rec.collapseToggle)
	}
}

// TestRailCollapsedMode verifies that rail mode renders section icons and expand affordance.
func TestRailCollapsedMode(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	s.Snapshot.IsCollapsed = true

	tt := ui.NewTester(func(c *ui.Context) {
		s.Sidebar(c, lobe.SideNavState{
			Collapse:  1.0,
			RailWidth: 64,
		})
	}, 200, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()

	// Verify rail section labels/icons
	for _, want := range []string{"Collections", "History", "Runner", "»"} {
		if !tt.HasText(want) {
			t.Errorf("rail mode missing %q; texts: %v", want, tt.Texts())
		}
	}

	// Verify tree is not drawn in rail mode
	if tt.HasText("Get Users") {
		t.Error("rail mode unexpectedly rendered tree nodes")
	}

	// Click on rail section buttons
	if err := tt.Click("Collections"); err != nil {
		t.Fatalf("click rail Collections: %v", err)
	}
	if err := tt.Click("History"); err != nil {
		t.Fatalf("click rail History: %v", err)
	}
	if err := tt.Click("Runner"); err != nil {
		t.Fatalf("click rail Runner: %v", err)
	}
	if len(rec.sections) != 3 {
		t.Errorf("rail section clicks: got %d, want 3", len(rec.sections))
	}

	// Click on rail expand button
	if err := tt.Click("»"); err != nil {
		t.Fatalf("click expand button »: %v", err)
	}
	if rec.collapseToggle != 1 {
		t.Errorf("OnToggleCollapse got %d, want 1", rec.collapseToggle)
	}
}

// TestTakeSnapshotAndNewScreen verifies snapshot creation from real presenters.
func TestTakeSnapshotAndNewScreen(t *testing.T) {
	// 1. Nil sidebar presenter
	nilSnap := TakeSnapshot(nil)
	if nilSnap.ActiveSection != presenters.SidebarSectionCollections {
		t.Errorf("TakeSnapshot(nil) ActiveSection: got %v, want collections", nilSnap.ActiveSection)
	}

	// 2. Real presenters with model data
	col := &model.Collection{
		UID:               "col-real",
		Name:              "Real API",
		ActiveEnvironment: "Dev",
		Environments: []*model.Environment{
			{Name: "Dev"},
			{Name: "Prod"},
		},
		Items: []*model.Item{
			{
				UID:  "item-1",
				Name: "Request 1",
				Type: model.ItemHTTPRequest,
			},
		},
	}

	colPres := presenters.NewCollectionPresenter(nil, nil, nil, nil)
	colPres.SetCollections([]*model.Collection{col})
	colPres.SetFilter("req")
	colPres.Select("item-1")

	envPres := presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
	envPres.SetCollection(col)

	sidebarPres := presenters.NewSidebarPresenter(colPres, envPres)
	sidebarPres.SetCollapsed(true)
	sidebarPres.SelectSection(presenters.SidebarSectionHistory)

	snap := TakeSnapshot(sidebarPres)
	if snap.CollectionName != "Real API" || snap.CollectionUID != "col-real" {
		t.Errorf("snap collection: got name=%q, uid=%q", snap.CollectionName, snap.CollectionUID)
	}
	if !snap.IsCollapsed {
		t.Error("snap IsCollapsed: got false, want true")
	}
	if snap.ActiveSection != presenters.SidebarSectionHistory {
		t.Errorf("snap ActiveSection: got %v, want history", snap.ActiveSection)
	}
	if snap.FilterQuery != "req" {
		t.Errorf("snap FilterQuery: got %q, want 'req'", snap.FilterQuery)
	}
	if snap.ActiveEnvironment != "Dev" || len(snap.Environments) != 2 {
		t.Errorf("snap environments: active=%q, all=%v", snap.ActiveEnvironment, snap.Environments)
	}

	// NewScreen constructor
	screen := NewScreen(sidebarPres, Actions{})
	if screen == nil || screen.Snapshot.CollectionName != "Real API" {
		t.Fatal("NewScreen failed to populate snapshot")
	}
}

// TestEmptyCollectionState verifies message shown when no nodes exist.
func TestEmptyCollectionState(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	s.Snapshot.VisibleNodes = nil

	tt := newTester(s)
	if !tt.HasText("No requests found") {
		t.Errorf("missing empty state message; texts: %v", tt.Texts())
	}
}

// TestHistoryAndRunnerSections verifies section switching bodies.
func TestHistoryAndRunnerSections(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)

	// History section
	s.Snapshot.ActiveSection = presenters.SidebarSectionHistory
	tt := newTester(s)
	if !tt.HasText("Request History") {
		t.Errorf("history section missing header; texts: %v", tt.Texts())
	}

	// Runner section
	s.Snapshot.ActiveSection = presenters.SidebarSectionRunner
	tt2 := newTester(s)
	if !tt2.HasText("Collection Runner") || !tt2.HasText("Target: My API") {
		t.Errorf("runner section missing contents; texts: %v", tt2.Texts())
	}

	// Click on Run Collection in runner section
	if err := tt2.Click("Run Collection"); err != nil {
		t.Fatalf("click Run Collection: %v", err)
	}
	if len(rec.runCollection) != 1 || rec.runCollection[0] != "col-1" {
		t.Errorf("OnRunCollection from runner: got %v, want ['col-1']", rec.runCollection)
	}
}

// TestNilStateRendersDefaults verifies safe fallback when Screen is instantiated empty or nil.
func TestNilStateRendersDefaults(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		Sidebar(c, nil, lobe.SideNavState{})
		CollectionHeader(c, nil)
	}, testW, testH)
	tt.Frame()

	if !tt.HasText("Fearless Collection") {
		t.Errorf("default header text missing; texts: %v", tt.Texts())
	}
}

// TestNoDomainImports ensures non-test source files do not import core or app packages.
func TestNoDomainImports(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no source files found")
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue // Test harness may construct model fixtures
		}
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", f, err)
			}
			if !strings.HasPrefix(path, "fearless/") {
				continue
			}
			if strings.HasPrefix(path, "fearless/internal/ui/") {
				continue // UI presenters, layout, theme, widgets are allowed
			}
			t.Errorf("%s imports %q: a screen source file must not reach the domain layers", f, path)
		}
	}
}

// TestGoldenAfterThemeChange validates light/dark theme switching and visual regression.
func TestGoldenAfterThemeChange(t *testing.T) {
	rec := &actionRecorder{}
	s := testScreen(rec)
	tt := newTester(s)

	lightImg := tt.Image()

	tt.SetDark(true)
	tt.Frame()
	darkImg := tt.Image()

	if sameImage(lightImg, darkImg) {
		t.Error("light and dark frames are pixel-identical: theme did not apply")
	}

	compareGolden(t, "navigation-light.png", lightImg)
	compareGolden(t, "navigation-dark.png", darkImg)
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return buf.Bytes()
}

func sameImage(a, b image.Image) bool {
	if a.Bounds().Size() != b.Bounds().Size() {
		return false
	}
	ab := a.Bounds()
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

func compareGolden(t *testing.T, name string, got image.Image) {
	t.Helper()
	path := filepath.Join("testdata", name)

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create golden %s: %v", path, err)
		}
		defer f.Close()
		if err := png.Encode(f, got); err != nil {
			t.Fatalf("encode golden %s: %v", path, err)
		}
		return
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Auto-generate if missing so tests pass out of the box in new environment
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatalf("mkdir testdata: %v", err)
			}
			wf, werr := os.Create(path)
			if werr != nil {
				t.Fatalf("create golden %s: %v", path, werr)
			}
			defer wf.Close()
			if err := png.Encode(wf, got); err != nil {
				t.Fatalf("encode golden %s: %v", path, err)
			}
			return
		}
		t.Fatalf("open golden %s: %v", path, err)
	}
	defer f.Close()

	want, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode golden %s: %v", path, err)
	}

	diffCount := 0
	b := got.Bounds()
	wb := want.Bounds()
	if b != wb {
		t.Fatalf("golden bounds mismatch: got %v, want %v", b, wb)
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if got.At(x, y) != want.At(x, y) {
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
