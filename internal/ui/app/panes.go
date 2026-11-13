package app

import (
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/request"
	"fearless/internal/ui/widgets"
)

// newRequestPresenter builds a request presenter wired to the app's send path.
//
// Each tab gets its own, and each takes its own item as it is created, so the
// presenter's in-flight flag and draft belong to one tab rather than to
// whichever tab was last selected.
func (a *App) newRequestPresenter() *presenters.RequestPresenter {
	a.ensureRequestServices()
	var p *presenters.RequestPresenter
	p = presenters.NewRequestPresenter(
		&requestSender{
			svc:        a.requests,
			item:       func() *model.Item { return p.Item() },
			collection: func() *model.Collection { return a.activeCollection() },
			env: func() string {
				if a.EnvironmentPresenter == nil {
					return ""
				}
				return a.EnvironmentPresenter.ActiveEnvironmentName()
			},
		},
		nil,
		nil,
	)
	return p
}

// requestPane is everything one tab owns: the request editor's presenter, the
// response viewer's, the URL bar's edit state and the request screen's editor
// state.
//
// Each tab gets its own pane so one request cannot interfere with another: a
// send in one tab no longer leaves another tab stuck on "Cancel", and a response
// in one tab no longer overwrites the response shown in another. The presenters
// are cheap value objects, so a pane per open tab costs nothing worth sharing.
type requestPane struct {
	request  *presenters.RequestPresenter
	response *presenters.ResponsePresenter
	url      *widgets.TemplateField
	state    *request.RequestScreenState
}

// paneFor returns the pane for a tab, creating it on first use. itemUID is the
// tab's identity, which TabsPresenter sets to the item's UID.
func (a *App) paneFor(itemUID string) *requestPane {
	if a.panes == nil {
		a.panes = make(map[string]*requestPane)
	}
	if p, ok := a.panes[itemUID]; ok {
		return p
	}
	p := &requestPane{
		request:  a.newRequestPresenter(),
		response: presenters.NewResponsePresenter(),
		url:      &widgets.TemplateField{},
		state:    &request.RequestScreenState{},
	}
	// Tie the response pane to this tab's request, so it adopts only that
	// item's executions. A pane keyed "" (the headless seed) stays unbound.
	if itemUID != "" {
		p.response.BindItem(itemUID)
	}
	a.panes[itemUID] = p
	return p
}

// setActiveItem points the app at item's tab: a.RequestPresenter and
// a.ResponsePresenter become that tab's, so every pane, action and status read
// sees one tab's state and never another's.
//
// Passing nil unbinds the request presenter and leaves the response presenter
// alone, so an empty selection does not discard the last tab's pane.
func (a *App) setActiveItem(item *model.Item) {
	if item == nil {
		if a.RequestPresenter != nil {
			a.RequestPresenter.SetItem(nil)
		}
		return
	}
	p := a.paneFor(item.UID)
	p.request.SetItem(item)
	a.RequestPresenter = p.request
	a.ResponsePresenter = p.response
}

// paneOf returns the pane belonging to item, creating it if needed.
func (a *App) paneOf(item *model.Item) *requestPane {
	if item == nil {
		return nil
	}
	return a.paneFor(item.UID)
}

// findItem resolves an item UID to the item, searching every open collection.
// The tab strip identifies a tab by its item UID, so this is how a tab click
// reaches the item again.
func (a *App) findItem(uid string) *model.Item {
	if uid == "" || a.CollectionPresenter == nil {
		return nil
	}
	for _, col := range a.CollectionPresenter.Collections() {
		if col == nil {
			continue
		}
		if item := col.FindItem(uid); item != nil {
			return item
		}
	}
	return nil
}

// dropPane forgets a closed tab's pane.
func (a *App) dropPane(itemUID string) {
	delete(a.panes, itemUID)
}
