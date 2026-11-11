package presenters

import (
	"context"
	"errors"
	"testing"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

type fakeSender struct {
	sentUID string
	err     error
}

func (f *fakeSender) Send(ctx context.Context, itemUID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.sentUID = itemUID
	return "req-test-123", nil
}

type fakeCanceller struct {
	cancelledUID string
}

func (f *fakeCanceller) Cancel(ctx context.Context, requestUID string) error {
	f.cancelledUID = requestUID
	return nil
}

type fakeSaver struct {
	savedItem *model.Item
}

func (f *fakeSaver) Save(ctx context.Context, item *model.Item) error {
	f.savedItem = item
	return nil
}

func TestRequestPresenterEditingAndDirty(t *testing.T) {
	saver := &fakeSaver{}
	sender := &fakeSender{}
	canceller := &fakeCanceller{}

	p := NewRequestPresenter(sender, canceller, saver)

	changedCalled := 0
	p.OnChanged(func() {
		changedCalled++
	})

	item := &model.Item{
		UID:  "item-1",
		Name: "Get Users",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com/users",
		},
	}

	p.SetItem(item)
	if p.Item() != item {
		t.Errorf("item mismatch: %v", p.Item())
	}
	if p.IsDirty() {
		t.Error("new item should not be dirty")
	}

	// Change SubTab
	p.SelectSubTab(SubTabHeaders)
	if p.ActiveSubTab() != SubTabHeaders {
		t.Errorf("expected SubTabHeaders, got %s", p.ActiveSubTab())
	}

	// Modify Method
	p.SetMethod("post")
	if !p.IsDirty() {
		t.Error("item should be dirty after method change")
	}
	if p.EffectiveRequest().Method != "POST" {
		t.Errorf("expected method POST, got %s", p.EffectiveRequest().Method)
	}
	// Original saved request should remain unchanged
	if item.Request.Method != "GET" {
		t.Errorf("saved request mutated before save: %s", item.Request.Method)
	}

	// Modify URL
	p.SetURL("https://api.example.com/create")
	if p.EffectiveRequest().URL != "https://api.example.com/create" {
		t.Errorf("URL mismatch: %s", p.EffectiveRequest().URL)
	}

	// Modify Body
	p.SetBodyMode(model.BodyJSON)
	p.SetJSONBody(`{"name":"bob"}`)
	if p.EffectiveRequest().Body.Mode != model.BodyJSON || *p.EffectiveRequest().Body.JSON != `{"name":"bob"}` {
		t.Errorf("body mismatch: %v", p.EffectiveRequest().Body)
	}

	p.SetTextBody("hello text")
	if *p.EffectiveRequest().Body.Text != "hello text" {
		t.Errorf("text body mismatch: %v", p.EffectiveRequest().Body)
	}

	// Save
	if err := p.Save(context.Background()); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if p.IsDirty() {
		t.Error("item should not be dirty after save")
	}
	if item.Request.Method != "POST" || item.Request.URL != "https://api.example.com/create" {
		t.Errorf("saved request mismatch: %v", item.Request)
	}
	if saver.savedItem != item {
		t.Errorf("saver not called with item: %v", saver.savedItem)
	}

	// Revert test
	p.SetMethod("PUT")
	if !p.IsDirty() {
		t.Error("item should be dirty after PUT")
	}
	p.Revert()
	if p.IsDirty() {
		t.Error("item should be clean after revert")
	}
	if p.EffectiveRequest().Method != "POST" {
		t.Errorf("revert failed to restore POST: %s", p.EffectiveRequest().Method)
	}

	if changedCalled == 0 {
		t.Error("onChanged was never called")
	}
}

func TestRequestPresenterSendAndCancel(t *testing.T) {
	sender := &fakeSender{}
	canceller := &fakeCanceller{}
	p := NewRequestPresenter(sender, canceller, nil)

	item := &model.Item{
		UID:  "item-send",
		Type: model.ItemHTTPRequest,
		Request: &model.Request{
			Method: "GET",
			URL:    "https://api.example.com",
		},
	}
	p.SetItem(item)

	reqUID, err := p.Send(context.Background())
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	if reqUID != "req-test-123" {
		t.Errorf("reqUID mismatch: %s", reqUID)
	}
	if !p.InFlight() {
		t.Error("expected inFlight to be true after send")
	}

	// Cancel
	if err := p.Cancel(context.Background()); err != nil {
		t.Fatalf("cancel error: %v", err)
	}
	if canceller.cancelledUID != "req-test-123" {
		t.Errorf("cancelledUID mismatch: %s", canceller.cancelledUID)
	}

	// Event handling
	p.HandleEvent(events.ResponseReceived{UID: "req-test-123"})
	if p.InFlight() {
		t.Error("expected inFlight false after ResponseReceived")
	}

	// Error event handling
	p.HandleEvent(events.RequestQueued{ItemUID: "item-send", UID: "req-err"})
	if !p.InFlight() {
		t.Error("expected inFlight true after RequestQueued")
	}
	p.HandleEvent(events.ErrorOccurred{UID: "req-err", Err: errors.New("boom")})
	if p.InFlight() {
		t.Error("expected inFlight false after ErrorOccurred")
	}
}
