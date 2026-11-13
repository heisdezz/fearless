package app

import (
	"context"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/engine"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// requestSender dispatches the active request through the request engine and
// publishes its events on a bus.
//
// The request presenter holds an item and asks a sender to execute it; it knows
// nothing about HTTP, environments or the event bus. This adapter is the
// engine side: it hands the item and its collection to commands.RequestService,
// which resolves variables, runs the request and publishes RequestQueued and
// ResponseReceived.
//
// The UI owns its collections in memory rather than in a bus store, so the
// target is passed straight to the command instead of being looked up by UID.
type requestSender struct {
	svc        *commands.RequestService
	collection func() *model.Collection
	item       func() *model.Item
	env        func() string
}

var _ presenters.RequestSenderPort = (*requestSender)(nil)

// ensureRequestServices lazily builds the bus and the request service the send
// path needs. They are created on first use so a test App with no engine still
// constructs cheaply.
func (a *App) ensureRequestServices() {
	if a.bus == nil {
		a.bus = bus.New()
	}
	if a.requests == nil {
		a.requests = commands.NewRequestService(engine.New(), nil, a.bus, nil, nil)
	}
}

// subscribeResponseEvents forwards execution events to every open tab's
// presenters on its own goroutine, marshalling each update onto the UI loop.
//
// Every pane is fed, not just the active one: a request sent from one tab keeps
// running while another is selected, and its response must land in its own tab
// rather than being dropped. Each presenter ignores events for a UID it is not
// tracking, so every pane seeing every event is harmless.
func (a *App) subscribeResponseEvents() {
	if a.bus == nil {
		return
	}
	ch, cancel := a.bus.Subscribe()
	go func() {
		defer cancel()
		for ev := range ch {
			a.SafeUpdate(func() {
				for _, p := range a.panes {
					if p.response != nil {
						p.response.HandleEvent(ev)
					}
					if p.request != nil {
						p.request.HandleEvent(ev)
					}
				}
			})
		}
	}()
}

// activeCollection returns the collection the request pane is editing: the
// one the environment presenter is bound to, or the first mounted collection.
func (a *App) activeCollection() *model.Collection {
	if a.EnvironmentPresenter != nil {
		if col := a.EnvironmentPresenter.Collection(); col != nil {
			return col
		}
	}
	if a.CollectionPresenter != nil {
		if cols := a.CollectionPresenter.Collections(); len(cols) > 0 {
			return cols[0]
		}
	}
	return nil
}

// Send executes the presenter's active item. The request UID is allocated by the
// service, which publishes RequestQueued before and ResponseReceived after, so
// the response presenter sees the result without a return value.
func (s *requestSender) Send(ctx context.Context, itemUID string) (string, error) {
	if s.svc == nil {
		return "", nil
	}
	col := s.collection()
	item := s.item()
	if col == nil || item == nil {
		return "", nil
	}

	env := ""
	if s.env != nil {
		env = s.env()
	}

	return s.svc.Send(ctx, commands.SendRequestCmd{
		ItemUID:       itemUID,
		CollectionUID: col.UID,
		Env:           env,
		Item:          item,
		Collection:    col,
	})
}
