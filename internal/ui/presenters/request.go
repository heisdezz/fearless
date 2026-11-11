package presenters

import (
	"context"
	"strings"
	"sync"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// RequestSubTab selects an active editor tab in the request pane.
type RequestSubTab string

const (
	SubTabParams   RequestSubTab = "params"
	SubTabBody     RequestSubTab = "body"
	SubTabHeaders  RequestSubTab = "headers"
	SubTabAuth     RequestSubTab = "auth"
	SubTabVars     RequestSubTab = "vars"
	SubTabScript   RequestSubTab = "script"
	SubTabAssert   RequestSubTab = "assert"
	SubTabTests    RequestSubTab = "tests"
	SubTabDocs     RequestSubTab = "docs"
	SubTabSettings RequestSubTab = "settings"
)

// AllRequestSubTabs lists all request sub-tabs in standard UI order.
var AllRequestSubTabs = []RequestSubTab{
	SubTabParams,
	SubTabBody,
	SubTabHeaders,
	SubTabAuth,
	SubTabVars,
	SubTabScript,
	SubTabAssert,
	SubTabTests,
	SubTabDocs,
	SubTabSettings,
}

// RequestSenderPort is the consumer port required by RequestPresenter to dispatch requests.
type RequestSenderPort interface {
	Send(ctx context.Context, itemUID string) (string, error)
}

// RequestCancellerPort is the consumer port required by RequestPresenter to cancel requests.
type RequestCancellerPort interface {
	Cancel(ctx context.Context, requestUID string) error
}

// ItemSaverPort is the consumer port required by RequestPresenter to persist items.
type ItemSaverPort interface {
	Save(ctx context.Context, item *model.Item) error
}

// RequestPresenter coordinates the request editing view-model.
type RequestPresenter struct {
	mu           sync.RWMutex
	item         *model.Item
	activeSubTab RequestSubTab
	activeReqUID string
	sender       RequestSenderPort
	canceller    RequestCancellerPort
	saver        ItemSaverPort
	onChanged    func()
	inFlight     bool
}

// NewRequestPresenter creates an initialized RequestPresenter.
func NewRequestPresenter(sender RequestSenderPort, canceller RequestCancellerPort, saver ItemSaverPort) *RequestPresenter {
	return &RequestPresenter{
		activeSubTab: SubTabParams,
		sender:       sender,
		canceller:    canceller,
		saver:        saver,
	}
}

// SetItem binds a model.Item to the presenter.
func (p *RequestPresenter) SetItem(item *model.Item) {
	p.mu.Lock()
	p.item = item
	p.mu.Unlock()
	p.notifyChanged()
}

// Item returns the currently bound model.Item.
func (p *RequestPresenter) Item() *model.Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.item
}

// EffectiveRequest returns the draft or saved request being edited.
func (p *RequestPresenter) EffectiveRequest() *model.Request {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.item == nil {
		return nil
	}
	return p.item.Effective()
}

// ActiveSubTab returns the currently selected sub-tab.
func (p *RequestPresenter) ActiveSubTab() RequestSubTab {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeSubTab
}

// SelectSubTab switches the active editor sub-tab.
func (p *RequestPresenter) SelectSubTab(tab RequestSubTab) {
	p.mu.Lock()
	p.activeSubTab = tab
	p.mu.Unlock()
	p.notifyChanged()
}

// IsDirty reports whether the current item has uncommitted edits.
func (p *RequestPresenter) IsDirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.item != nil && p.item.Dirty()
}

// InFlight reports whether a request execution is currently in-flight.
func (p *RequestPresenter) InFlight() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.inFlight
}

// OnChanged registers a callback triggered when presenter state changes.
func (p *RequestPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *RequestPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func (p *RequestPresenter) ensureDraft() *model.Request {
	if p.item == nil {
		return nil
	}
	if p.item.Draft == nil {
		if p.item.Request != nil {
			p.item.Draft = p.item.Request.Clone()
		} else {
			p.item.Draft = &model.Request{
				Method: "GET",
			}
		}
	}
	return p.item.Draft
}

// SetMethod updates the HTTP method on the draft request.
func (p *RequestPresenter) SetMethod(method string) {
	p.mu.Lock()
	req := p.ensureDraft()
	if req != nil {
		req.Method = strings.ToUpper(strings.TrimSpace(method))
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// SetURL updates the target URL on the draft request.
func (p *RequestPresenter) SetURL(rawURL string) {
	p.mu.Lock()
	req := p.ensureDraft()
	if req != nil {
		req.URL = strings.TrimSpace(rawURL)
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// SetBodyMode updates the body mode on the draft request.
func (p *RequestPresenter) SetBodyMode(mode model.BodyMode) {
	p.mu.Lock()
	req := p.ensureDraft()
	if req != nil {
		if req.Body == nil {
			req.Body = &model.Body{}
		}
		req.Body.Mode = mode
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// SetJSONBody updates the raw JSON body on the draft request.
func (p *RequestPresenter) SetJSONBody(jsonStr string) {
	p.mu.Lock()
	req := p.ensureDraft()
	if req != nil {
		if req.Body == nil {
			req.Body = &model.Body{Mode: model.BodyJSON}
		}
		req.Body.JSON = &jsonStr
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// SetTextBody updates the raw text body on the draft request.
func (p *RequestPresenter) SetTextBody(textStr string) {
	p.mu.Lock()
	req := p.ensureDraft()
	if req != nil {
		if req.Body == nil {
			req.Body = &model.Body{Mode: model.BodyText}
		}
		req.Body.Text = &textStr
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// Save promotes draft edits to saved state and invokes ItemSaverPort.
func (p *RequestPresenter) Save(ctx context.Context) error {
	p.mu.Lock()
	if p.item == nil {
		p.mu.Unlock()
		return nil
	}

	if p.item.Draft != nil {
		p.item.Request = p.item.Draft
		p.item.Draft = nil
	}
	itemToSave := p.item
	saver := p.saver
	p.mu.Unlock()

	var err error
	if saver != nil {
		err = saver.Save(ctx, itemToSave)
	}
	p.notifyChanged()
	return err
}

// Revert discards any uncommitted draft edits.
func (p *RequestPresenter) Revert() {
	p.mu.Lock()
	if p.item != nil {
		p.item.Draft = nil
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// Send dispatches the current item via RequestSenderPort.
func (p *RequestPresenter) Send(ctx context.Context) (string, error) {
	p.mu.Lock()
	if p.item == nil || p.sender == nil {
		p.mu.Unlock()
		return "", nil
	}
	itemUID := p.item.UID
	sender := p.sender
	p.inFlight = true
	p.mu.Unlock()

	p.notifyChanged()

	reqUID, err := sender.Send(ctx, itemUID)
	p.mu.Lock()
	if err != nil {
		p.inFlight = false
	} else {
		p.activeReqUID = reqUID
	}
	p.mu.Unlock()

	p.notifyChanged()
	return reqUID, err
}

// Cancel terminates in-flight execution via RequestCancellerPort.
func (p *RequestPresenter) Cancel(ctx context.Context) error {
	p.mu.Lock()
	reqUID := p.activeReqUID
	canceller := p.canceller
	p.mu.Unlock()

	if canceller == nil || reqUID == "" {
		return nil
	}

	return canceller.Cancel(ctx, reqUID)
}

// HandleEvent updates presenter in-flight state based on lifecycle events.
func (p *RequestPresenter) HandleEvent(ev events.Event) {
	switch e := ev.(type) {
	case events.RequestQueued:
		p.mu.Lock()
		if p.item != nil && e.ItemUID == p.item.UID {
			p.inFlight = true
			p.activeReqUID = e.UID
		}
		p.mu.Unlock()
		p.notifyChanged()

	case events.ResponseReceived:
		p.mu.Lock()
		if p.activeReqUID == e.UID {
			p.inFlight = false
			p.activeReqUID = ""
		}
		p.mu.Unlock()
		p.notifyChanged()

	case events.ErrorOccurred:
		p.mu.Lock()
		if p.activeReqUID == e.UID {
			p.inFlight = false
			p.activeReqUID = ""
		}
		p.mu.Unlock()
		p.notifyChanged()
	}
}
