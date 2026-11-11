package request

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

type mockSender struct {
	sentItemUID string
	sendCount   int
}

func (m *mockSender) Send(ctx context.Context, itemUID string) (string, error) {
	m.sentItemUID = itemUID
	m.sendCount++
	return "req-exec-1", nil
}

type mockCanceller struct {
	cancelledReqUID string
	cancelCount     int
}

func (m *mockCanceller) Cancel(ctx context.Context, requestUID string) error {
	m.cancelledReqUID = requestUID
	m.cancelCount++
	return nil
}

type mockSaver struct {
	savedItem  *model.Item
	savedCount int
}

func (m *mockSaver) Save(ctx context.Context, item *model.Item) error {
	m.savedItem = item
	m.savedCount++
	return nil
}

func strPtr(s string) *string {
	return &s
}

func createTestItem() *model.Item {
	return &model.Item{
		UID:  "item-123",
		Name: "Get Users",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/users",
			Headers: []model.KeyValue{
				{Name: strPtr("Accept"), Value: strPtr("application/json"), Enabled: true},
			},
			Params: []model.Param{
				{KeyValue: model.KeyValue{Name: strPtr("page"), Value: strPtr("1"), Enabled: true}, Type: model.ParamQuery},
				{KeyValue: model.KeyValue{Name: strPtr("userId"), Value: strPtr("42"), Enabled: true}, Type: model.ParamPath},
			},
			Body: &model.Body{
				Mode: model.BodyJSON,
				JSON: strPtr(`{"name": "Alice"}`),
			},
			Auth: &model.Auth{
				Mode:     model.AuthBasic,
				Username: strPtr("admin"),
				Password: strPtr("secret"),
			},
			Script: &model.Scripts{
				Req: strPtr("console.log('pre-req');"),
			},
			Vars: &model.VarBlocks{
				PreRequest: []model.Variable{
					{Name: strPtr("v1"), Value: "val1", Enabled: true},
				},
				PostResponse: []model.Variable{
					{Name: strPtr("v2"), Value: "val2", Enabled: true},
				},
			},
			Assertions: []model.KeyValue{
				{Name: strPtr("res.status"), Value: strPtr("200"), Enabled: true},
			},
			Tests: strPtr("test('ok', function() {});"),
			Docs:  strPtr("# Documentation"),
		},
	}
}

func TestRequestScreen_EmptyState(t *testing.T) {
	view := func(c *ui.Context) {
		RequestScreen(c, nil)
	}

	tt := ui.NewTester(view, 800, 600)
	tt.Frame()

	if !tt.HasText("No Request Selected") {
		t.Error("expected 'No Request Selected' empty state text")
	}

	nodes := tt.AccessAll()
	if len(nodes) == 0 {
		t.Error("expected accessibility nodes for empty state")
	}
}

func TestRequestScreen_RenderAndInteract(t *testing.T) {
	sender := &mockSender{}
	canceller := &mockCanceller{}
	saver := &mockSaver{}

	p := presenters.NewRequestPresenter(sender, canceller, saver)
	item := createTestItem()
	p.SetItem(item)

	st := &RequestScreenState{}

	view := func(c *ui.Context) {
		RequestScreen(c, p, st)
	}

	tt := ui.NewTester(view, 1000, 700)
	tt.Frame()

	// Verify top bar elements
	if !tt.HasText("Send") {
		t.Error("expected 'Send' button")
	}
	if !tt.HasText("Save") {
		t.Error("expected 'Save' button")
	}

	// Verify sub-tab labels
	for _, tab := range []string{"Params", "Body", "Headers", "Auth", "Vars", "Script", "Assert", "Tests", "Docs"} {
		if !tt.HasText(tab) {
			t.Errorf("expected sub-tab %q to be visible", tab)
		}
	}

	// Click Send button
	err := tt.Click("Send")
	if err != nil {
		t.Fatalf("failed to click Send: %v", err)
	}
	tt.Frame()

	if sender.sendCount != 1 || sender.sentItemUID != "item-123" {
		t.Errorf("expected sender to be called once with item-123, got count=%d, uid=%q",
			sender.sendCount, sender.sentItemUID)
	}

	// While in flight, Send button turns into Cancel
	if !tt.HasText("Cancel") {
		t.Error("expected 'Cancel' button while request is in-flight")
	}

	// Click Cancel button
	err = tt.Click("Cancel")
	if err != nil {
		t.Fatalf("failed to click Cancel: %v", err)
	}

	if canceller.cancelCount != 1 || canceller.cancelledReqUID != "req-exec-1" {
		t.Errorf("expected canceller to be called once with req-exec-1, got count=%d, uid=%q",
			canceller.cancelCount, canceller.cancelledReqUID)
	}

	// Switch sub-tabs and verify rendering of each
	p.SelectSubTab(presenters.SubTabBody)
	tt.Frame()
	if !tt.HasText("Body Mode:") {
		t.Error("expected Body Mode selector when SubTabBody is active")
	}

	p.SelectSubTab(presenters.SubTabHeaders)
	tt.Frame()
	if !tt.HasText("Headers") {
		t.Error("expected Headers title when SubTabHeaders is active")
	}

	p.SelectSubTab(presenters.SubTabAuth)
	tt.Frame()
	if !tt.HasText("Auth Type:") {
		t.Error("expected Auth Type selector when SubTabAuth is active")
	}

	p.SelectSubTab(presenters.SubTabVars)
	tt.Frame()
	if !tt.HasText("Pre-Request Variables") || !tt.HasText("Post-Response Variables") {
		t.Error("expected Variables sections when SubTabVars is active")
	}

	p.SelectSubTab(presenters.SubTabScript)
	tt.Frame()
	if !tt.HasText("Pre-Request Script (runs before request is sent)") {
		t.Error("expected Pre-Request Script section when SubTabScript is active")
	}

	p.SelectSubTab(presenters.SubTabAssert)
	tt.Frame()
	if !tt.HasText("Response Assertions") {
		t.Error("expected Response Assertions section when SubTabAssert is active")
	}

	p.SelectSubTab(presenters.SubTabTests)
	tt.Frame()
	if !tt.HasText("Tests (runs after response is received)") {
		t.Error("expected Tests section when SubTabTests is active")
	}

	p.SelectSubTab(presenters.SubTabDocs)
	tt.Frame()
	if !tt.HasText("Documentation (Markdown)") {
		t.Error("expected Documentation section when SubTabDocs is active")
	}

	// Verify accessibility
	nodes := tt.AccessAll()
	if len(nodes) == 0 {
		t.Error("expected accessible elements in RequestScreen")
	}
}

func TestRequestScreen_BodyModes(t *testing.T) {
	modes := []struct {
		mode    model.BodyMode
		snippet string
	}{
		{model.BodyNone, "This request does not have a body."},
		{model.BodyText, "plain text payload"},
		{model.BodyXML, "<request>"},
		{model.BodyFormURLEncoded, "Field Name"},
		{model.BodyMultipartForm, "Part Name"},
	}

	for _, m := range modes {
		p := presenters.NewRequestPresenter(nil, nil, nil)
		item := createTestItem()
		item.Request.Body = &model.Body{
			Mode: m.mode,
			FormURLEncoded: []model.KeyValue{
				{Name: strPtr("formKey"), Value: strPtr("formVal"), Enabled: true},
			},
			MultipartForm: []model.MultipartEntry{
				{Name: strPtr("part1"), Value: strPtr("val1"), Enabled: true},
			},
		}
		p.SetItem(item)
		p.SelectSubTab(presenters.SubTabBody)

		st := &RequestScreenState{}
		tt := ui.NewTester(func(c *ui.Context) {
			RequestScreen(c, p, st)
		}, 800, 600)
		tt.Frame()

		if !tt.HasText("Body Mode:") {
			t.Errorf("mode %s: expected Body Mode selector", m.mode)
		}
	}
}

func TestRequestScreen_AuthModes(t *testing.T) {
	modes := []struct {
		mode      model.AuthMode
		wantToken string
	}{
		{model.AuthInherit, "Inheriting authentication"},
		{model.AuthNone, "This request does not use any authentication."},
		{model.AuthBearer, "Bearer token"},
		{model.AuthAPIKey, "X-API-Key"},
		{model.AuthDigest, "Username"},
	}

	for _, m := range modes {
		p := presenters.NewRequestPresenter(nil, nil, nil)
		item := createTestItem()
		item.Request.Auth = &model.Auth{
			Mode:      m.mode,
			Username:  strPtr("user1"),
			Password:  strPtr("pass1"),
			Token:     strPtr("tok123"),
			Key:       strPtr("api-key-header"),
			Value:     strPtr("secret-val"),
			Placement: model.APIKeyHeader,
		}
		p.SetItem(item)
		p.SelectSubTab(presenters.SubTabAuth)

		tt := ui.NewTester(func(c *ui.Context) {
			RequestScreen(c, p)
		}, 800, 600)
		tt.Frame()

		if !tt.HasText("Auth Type:") {
			t.Errorf("mode %s: expected Auth Type selector", m.mode)
		}
	}
}

func TestRequestScreen_SaveAndDirty(t *testing.T) {
	saver := &mockSaver{}
	p := presenters.NewRequestPresenter(nil, nil, saver)
	item := createTestItem()
	p.SetItem(item)

	// Mutate draft to make it dirty
	p.SetURL("https://api.example.com/updated")

	if !p.IsDirty() {
		t.Fatal("expected presenter to report dirty")
	}

	view := func(c *ui.Context) {
		RequestScreen(c, p)
	}

	tt := ui.NewTester(view, 800, 600)
	tt.Frame()

	// Should show "Save *" for dirty request
	if !tt.HasText("Save *") {
		t.Error("expected 'Save *' button text when request is dirty")
	}

	// Click Save *
	err := tt.Click("Save *")
	if err != nil {
		t.Fatalf("failed to click Save: %v", err)
	}

	if saver.savedCount != 1 {
		t.Errorf("expected saver to be called once, got %d", saver.savedCount)
	}
	if p.IsDirty() {
		t.Error("expected presenter to no longer be dirty after saving")
	}
}

func TestRequestScreen_Callbacks(t *testing.T) {
	p := presenters.NewRequestPresenter(nil, nil, nil)
	item := createTestItem()
	p.SetItem(item)

	p.SelectSubTab(presenters.SubTabParams)
	tt := ui.NewTester(func(c *ui.Context) {
		RequestScreen(c, p)
	}, 800, 600)
	tt.Frame()

	// Add Row in Params
	err := tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("click + Add Row in params: %v", err)
	}
	tt.Frame()

	// Switch to Vars
	p.SelectSubTab(presenters.SubTabVars)
	tt.Frame()
	err = tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("click + Add Row in vars: %v", err)
	}
	tt.Frame()

	// Switch to Multipart Form Body
	item.Request.Body = &model.Body{
		Mode: model.BodyMultipartForm,
		MultipartForm: []model.MultipartEntry{
			{Name: strPtr("file"), Value: strPtr("a.txt"), Enabled: true},
		},
	}
	p.SelectSubTab(presenters.SubTabBody)
	tt.Frame()
	err = tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("click + Add Row in multipart: %v", err)
	}
	tt.Frame()
}
