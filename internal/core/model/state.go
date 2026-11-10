package model

// TabKind distinguishes what a tab shows.
type TabKind string

const (
	TabRequest  TabKind = "request"
	TabFolder   TabKind = "folder"
	TabWelcome  TabKind = "welcome"
	TabSettings TabKind = "settings"
)

// PaneOrientation is how the request and response panes are arranged.
type PaneOrientation string

const (
	// Stacked puts the request above the response.
	Stacked PaneOrientation = "stacked"
	// SideBySide puts the request left and the response right.
	SideBySide PaneOrientation = "sideBySide"
)

// RequestSubTab names one of the request editor tabs.
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

// ResponseSubTab names one of the response viewer tabs.
type ResponseSubTab string

const (
	SubTabResponse    ResponseSubTab = "response"
	SubTabRespHeaders ResponseSubTab = "responseHeaders"
	SubTabTimeline    ResponseSubTab = "timeline"
	SubTabTestResults ResponseSubTab = "testResults"
)

// Tab is one open tab.
//
// Tab content state (which sub-tab is showing, orientation) lives here rather
// than in the domain model: it is presentation state that must never be read by
// the engine.
type Tab struct {
	UID           string  `json:"uid"`
	Kind          TabKind `json:"kind"`
	CollectionUID string  `json:"collectionUid,omitempty"`
	ItemUID       string  `json:"itemUid,omitempty"`
	Title         string  `json:"title"`
	// Transient marks a scratch request that has never been saved.
	Transient bool `json:"transient,omitempty"`

	// Presentation-only state.
	SubTab         RequestSubTab   `json:"subTab,omitempty"`
	ResponseSubTab ResponseSubTab  `json:"responseSubTab,omitempty"`
	Orientation    PaneOrientation `json:"orientation,omitempty"`
	ResponseFormat string          `json:"responseFormat,omitempty"` // pretty | raw | preview
}

// WorkspaceState is the immutable snapshot the UI renders.
//
// Snapshots are produced by the state actor and handed to presenters. Readers
// must never mutate them; the actor owns the writable copy.
type WorkspaceState struct {
	Collections []*Collection `json:"collections"`
	Tabs        []*Tab        `json:"tabs"`
	ActiveTab   string        `json:"activeTab,omitempty"`
	Settings    Settings      `json:"settings"`

	// Responses holds the current result per request UID. Bodies live behind
	// BodyRef; nothing here carries a large payload.
	Responses map[string]*Result `json:"responses,omitempty"`
	// Pending marks requests currently in flight.
	Pending map[string]bool `json:"pending,omitempty"`

	// SidebarCollapsed and layout ratios are presentation state persisted with
	// the session.
	SidebarWidth     float32 `json:"sidebarWidth,omitempty"`
	SidebarCollapsed bool    `json:"sidebarCollapsed,omitempty"`
	ConsoleOpen      bool    `json:"consoleOpen,omitempty"`
	ConsoleHeight    float32 `json:"consoleHeight,omitempty"`
}

// FindCollection returns the collection with the given UID.
func (s *WorkspaceState) FindCollection(uid string) *Collection {
	if s == nil {
		return nil
	}
	for _, c := range s.Collections {
		if c.UID == uid {
			return c
		}
	}
	return nil
}

// FindTab returns the tab with the given UID.
func (s *WorkspaceState) FindTab(uid string) *Tab {
	if s == nil {
		return nil
	}
	for _, t := range s.Tabs {
		if t.UID == uid {
			return t
		}
	}
	return nil
}

// ActiveTabState returns the active tab, or nil.
func (s *WorkspaceState) ActiveTabState() *Tab {
	if s == nil {
		return nil
	}
	return s.FindTab(s.ActiveTab)
}

// ResponseFor returns the stored result for an item, if any.
func (s *WorkspaceState) ResponseFor(itemUID string) *Result {
	if s == nil || s.Responses == nil {
		return nil
	}
	return s.Responses[itemUID]
}

// IsPending reports whether the item has a request in flight.
func (s *WorkspaceState) IsPending(itemUID string) bool {
	if s == nil || s.Pending == nil {
		return false
	}
	return s.Pending[itemUID]
}
