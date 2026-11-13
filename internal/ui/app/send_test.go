package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// TestSendActiveRequestReachesResponsePresenter covers the send path end to
// end: a click on Send runs the item through the engine against a test server,
// the result is published on the bus, and the response presenter lands on the
// completed state. Without the request sender wired, Send is a silent no-op and
// this test never sees a result.
func TestSendActiveRequestReachesResponsePresenter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"pong"}`))
	}))
	defer srv.Close()

	a := New()

	col := &model.Collection{
		UID:  "col-send",
		Name: "Send Test",
		Environments: []*model.Environment{
			{Name: "local", Variables: []model.Variable{
				{Name: strPtr("baseUrl"), Value: srv.URL, Enabled: true},
			}},
		},
		ActiveEnvironment: "local",
		Items: []*model.Item{
			{
				UID:  "item-send",
				Name: "Ping",
				Type: model.ItemHTTPRequest,
				Request: &model.Request{
					Method: "GET",
					URL:    "{{baseUrl}}/ping",
				},
			},
		},
	}

	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	a.EnvironmentPresenter.SetCollection(col)

	item := col.FindItem("item-send")
	if item == nil {
		t.Fatal("test item missing from collection")
	}
	a.RequestPresenter.SetItem(item)

	a.SendActiveRequest()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if a.ResponsePresenter.State() == presenters.ResponseStateCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("response never completed; state=%q err=%q",
				a.ResponsePresenter.State(), a.ResponsePresenter.ErrorMessage())
		}
		time.Sleep(10 * time.Millisecond)
	}

	res := a.ResponsePresenter.Result()
	if res == nil {
		t.Fatal("completed with nil result")
	}
	if res.Status != http.StatusCreated {
		t.Errorf("status = %d, want %d", res.Status, http.StatusCreated)
	}
	if got := string(res.Body.Inline); got != `{"message":"pong"}` {
		t.Errorf("body = %q, want the server payload", got)
	}
	// The request URL was a variable, so a resolved URL proves the environment
	// variables reached the engine.
	if got := res.Sent.URL; got != srv.URL+"/ping" {
		t.Errorf("sent URL = %q, want %q (variable not resolved)", got, srv.URL+"/ping")
	}

	// The request presenter must see the same event, or the Send button stays
	// on "Cancel" after the response is already showing.
	inFlightDeadline := time.Now().Add(2 * time.Second)
	for a.RequestPresenter.InFlight() {
		if time.Now().After(inFlightDeadline) {
			t.Fatal("request presenter still reports in-flight after the response completed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func strPtr(s string) *string { return &s }

// TestSendActiveRequestNoItemIsSafe documents that Send without a loaded item
// does nothing and does not panic.
func TestSendActiveRequestNoItemIsSafe(t *testing.T) {
	a := New()
	a.RequestPresenter.SetItem(nil)
	a.SendActiveRequest()

	if a.ResponsePresenter.State() != presenters.ResponseStateIdle {
		t.Errorf("expected idle state, got %q", a.ResponsePresenter.State())
	}
}
