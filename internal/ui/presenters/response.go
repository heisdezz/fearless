package presenters

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// ResponseSubTab selects an active tab in the response pane.
type ResponseSubTab string

const (
	RespSubTabBody     ResponseSubTab = "body"
	RespSubTabHeaders  ResponseSubTab = "headers"
	RespSubTabTimeline ResponseSubTab = "timeline"
	RespSubTabTests    ResponseSubTab = "tests"
)

// AllResponseSubTabs lists all response tabs in standard UI order.
var AllResponseSubTabs = []ResponseSubTab{
	RespSubTabBody,
	RespSubTabHeaders,
	RespSubTabTimeline,
	RespSubTabTests,
}

// ResponseState represents the lifecycle status of the response viewer.
type ResponseState string

const (
	ResponseStateIdle      ResponseState = "idle"
	ResponseStateLoading   ResponseState = "loading"
	ResponseStateCompleted ResponseState = "completed"
	ResponseStateError     ResponseState = "error"
)

// ResponsePresenter coordinates the response viewer view-model.
type ResponsePresenter struct {
	mu           sync.RWMutex
	state        ResponseState
	activeSubTab ResponseSubTab
	activeReqUID string
	// itemUID is the request this pane belongs to. A pane only adopts the
	// executions of that item, so one tab's send never lands in another.
	itemUID    string
	result     *model.Result
	errMessage string
	streamBody strings.Builder
	streamSize int64
	onChanged  func()
}

// NewResponsePresenter creates an initialized ResponsePresenter.
func NewResponsePresenter() *ResponsePresenter {
	return &ResponsePresenter{
		state:        ResponseStateIdle,
		activeSubTab: RespSubTabBody,
	}
}

// BindItem ties the presenter to the request it shows. Only that request's
// executions are adopted; an empty UID keeps the old accept-everything
// behaviour, which a standalone presenter (tests, previews) still wants.
func (p *ResponsePresenter) BindItem(itemUID string) {
	p.mu.Lock()
	if p.itemUID == itemUID {
		p.mu.Unlock()
		return
	}
	p.itemUID = itemUID
	// Switching to another request starts from a clean pane rather than
	// showing the previous request's response.
	p.state = ResponseStateIdle
	p.activeReqUID = ""
	p.result = nil
	p.errMessage = ""
	p.streamBody.Reset()
	p.streamSize = 0
	p.mu.Unlock()
	p.notifyChanged()
}

// State returns the current lifecycle state (idle, loading, completed, error).
func (p *ResponsePresenter) State() ResponseState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Result returns the completed or in-progress model.Result.
func (p *ResponsePresenter) Result() *model.Result {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.result
}

// ErrorMessage returns the error description if in error state.
func (p *ResponsePresenter) ErrorMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errMessage
}

// ActiveSubTab returns the currently selected response tab.
func (p *ResponsePresenter) ActiveSubTab() ResponseSubTab {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeSubTab
}

// SelectSubTab switches the active response tab.
func (p *ResponsePresenter) SelectSubTab(tab ResponseSubTab) {
	p.mu.Lock()
	p.activeSubTab = tab
	p.mu.Unlock()
	p.notifyChanged()
}

// FormattedStatus returns "200 OK" or "Error" for the header bar.
func (p *ResponsePresenter) FormattedStatus() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.result == nil {
		if p.state == ResponseStateError {
			return "Error"
		}
		return ""
	}
	if p.result.StatusText != "" {
		return fmt.Sprintf("%d %s", p.result.Status, p.result.StatusText)
	}
	return fmt.Sprintf("%d", p.result.Status)
}

// StatusCodeCategory returns "success", "redirect", "client_error", "server_error" or "".
func (p *ResponsePresenter) StatusCodeCategory() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.result == nil {
		return ""
	}
	code := p.result.Status
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client_error"
	case code >= 500:
		return "server_error"
	default:
		return ""
	}
}

// FormattedDuration returns a human-friendly duration string (e.g. "45 ms", "1.23 s").
func (p *ResponsePresenter) FormattedDuration() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.result == nil {
		return ""
	}
	d := p.result.Duration
	if d < time.Millisecond {
		return "< 1 ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}

// FormattedSize returns a human-friendly byte size string (e.g. "512 B", "12.4 KB", "3.1 MB").
func (p *ResponsePresenter) FormattedSize() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var size int64
	if p.result != nil {
		size = p.result.Body.Size
	} else if p.streamSize > 0 {
		size = p.streamSize
	} else {
		return ""
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), suffix)
}

// BodyText returns the response body as a string.
func (p *ResponsePresenter) BodyText() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.result != nil {
		if len(p.result.Body.Inline) > 0 {
			return string(p.result.Body.Inline)
		}
		if p.result.Body.Spilled() {
			return fmt.Sprintf("[Body spilled to disk: %s]", p.result.Body.Path)
		}
	}
	return p.streamBody.String()
}

// OnChanged registers an invalidation callback hook.
func (p *ResponsePresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *ResponsePresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

// Reset clears the presenter back to idle state.
func (p *ResponsePresenter) Reset() {
	p.mu.Lock()
	p.state = ResponseStateIdle
	p.activeReqUID = ""
	p.result = nil
	p.errMessage = ""
	p.streamBody.Reset()
	p.streamSize = 0
	p.mu.Unlock()
	p.notifyChanged()
}

// HandleEvent processes events from the bus to drive the response view.
func (p *ResponsePresenter) HandleEvent(ev events.Event) {
	switch e := ev.(type) {
	case events.RequestQueued:
		p.mu.Lock()
		// Adopt only this pane's own request. Without the filter every open
		// tab latched onto whichever request was queued last, so a send in one
		// tab put every other tab in loading state and then showed that tab's
		// response.
		if p.itemUID != "" && e.ItemUID != p.itemUID {
			p.mu.Unlock()
			return
		}
		p.activeReqUID = e.UID
		p.state = ResponseStateLoading
		p.result = nil
		p.errMessage = ""
		p.streamBody.Reset()
		p.streamSize = 0
		p.mu.Unlock()
		p.notifyChanged()

	case events.ResponseChunk:
		p.mu.Lock()
		if p.activeReqUID == e.UID {
			p.streamBody.Write(e.Data)
			p.streamSize += int64(len(e.Data))
		}
		p.mu.Unlock()
		p.notifyChanged()

	case events.ResponseReceived:
		p.mu.Lock()
		if p.activeReqUID == e.UID {
			res := e.Received
			p.result = &res
			p.state = ResponseStateCompleted
			p.activeReqUID = ""
		}
		p.mu.Unlock()
		p.notifyChanged()

	case events.ErrorOccurred:
		p.mu.Lock()
		if p.activeReqUID == e.UID {
			p.state = ResponseStateError
			p.errMessage = e.Msg
			if p.errMessage == "" && e.Err != nil {
				p.errMessage = e.Err.Error()
			}
			p.activeReqUID = ""
		}
		p.mu.Unlock()
		p.notifyChanged()
	}
}
