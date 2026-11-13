package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fearless/internal/core/model"
)

// TestTabsSendIndependently covers per-tab isolation: sending in one tab must
// not put another tab in flight, and each tab must keep its own response.
//
// Before panes, every tab shared one request presenter and one response
// presenter, so one tab's send left every other tab showing "Cancel" and its
// response overwrote whatever another tab had received.
func TestTabsSendIndependently(t *testing.T) {
	slowRelease := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-slowRelease
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow"))
	}))
	defer slow.Close()

	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("fast"))
	}))
	defer fast.Close()

	a := New()
	col := &model.Collection{
		UID: "c1", Name: "Tabs",
		Items: []*model.Item{
			{UID: "r-slow", Name: "Slow", Type: model.ItemHTTPRequest,
				Request: &model.Request{Method: "GET", URL: slow.URL}},
			{UID: "r-fast", Name: "Fast", Type: model.ItemHTTPRequest,
				Request: &model.Request{Method: "GET", URL: fast.URL}},
		},
	}
	a.CollectionPresenter.SetCollections([]*model.Collection{col})
	a.EnvironmentPresenter.SetCollection(col)

	slowItem, fastItem := col.Items[0], col.Items[1]

	// Open the slow tab and send it.
	a.TabsPresenter.OpenTab(slowItem)
	a.setActiveItem(slowItem)
	slowPane := a.paneFor("r-slow")

	a.SendActiveRequest()
	waitUntil(t, "slow request to be in flight", func() bool {
		return slowPane.request.InFlight()
	})

	// While it is in flight, switch to the fast tab: it must not be in flight.
	a.TabsPresenter.OpenTab(fastItem)
	a.setActiveItem(fastItem)
	fastPane := a.paneFor("r-fast")

	if fastPane.request.InFlight() {
		t.Fatal("the fast tab reports in-flight because another tab is sending")
	}
	if a.RequestPresenter == slowPane.request {
		t.Fatal("the active request presenter did not switch to the fast tab")
	}

	// Send the fast one while the slow one is still running.
	a.SendActiveRequest()
	waitUntil(t, "fast response", func() bool {
		return fastPane.response.State() == "completed"
	})
	if got := fastPane.response.Result(); got == nil || got.Status != http.StatusCreated {
		t.Fatalf("fast tab result = %+v", got)
	}
	if !slowPane.request.InFlight() {
		t.Error("the slow tab should still be in flight while the fast one finished")
	}

	// Release the slow one; both tabs keep their own result.
	close(slowRelease)
	waitUntil(t, "slow response", func() bool {
		return slowPane.response.State() == "completed"
	})
	if got := slowPane.response.Result(); got == nil || got.Status != http.StatusOK {
		t.Fatalf("slow tab result = %+v", got)
	}
	if got := fastPane.response.Result(); got == nil || got.Status != http.StatusCreated {
		t.Errorf("fast tab result was overwritten: %+v", got)
	}

	// Both tabs are idle once their own request finished.
	if slowPane.request.InFlight() || fastPane.request.InFlight() {
		t.Error("both tabs should be idle after their requests completed")
	}
}

func waitUntil(t *testing.T, what string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
