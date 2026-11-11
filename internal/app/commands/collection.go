package commands

import (
	"context"
	"errors"
	"sync"

	"fearless/internal/app/bus"
	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

var (
	// ErrCollectionNotFound is returned when a collection UID cannot be resolved.
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrItemNotFound is returned when an item UID cannot be resolved.
	ErrItemNotFound = errors.New("item not found")
	// ErrInvalidInput is returned when required command fields are missing or invalid.
	ErrInvalidInput = errors.New("invalid command input")
)

// Consumer interfaces declared per interface-design.md §2 for presenters and UI.

// CollectionLoader loads a collection tree from disk into the application workspace.
type CollectionLoader interface {
	Execute(ctx context.Context, cmd LoadCollectionCmd) (collectionUID string, err error)
}

// CollectionReloader reloads an existing collection from disk, reconciling external changes.
type CollectionReloader interface {
	Execute(ctx context.Context, cmd ReloadCollectionCmd) error
}

// ItemSaver commits draft edits to saved request and writes to disk.
type ItemSaver interface {
	Execute(ctx context.Context, cmd SaveItemCmd) error
}

// ItemDeleter deletes an item from disk and removes it from the workspace state.
type ItemDeleter interface {
	Execute(ctx context.Context, cmd DeleteItemCmd) error
}

// ItemCreator creates a new request or folder item on disk and adds it to the workspace.
type ItemCreator interface {
	Execute(ctx context.Context, cmd NewItemCmd) (*model.Item, error)
}

// ItemMover moves an item to a new parent folder or re-sequences it within its folder.
type ItemMover interface {
	Execute(ctx context.Context, cmd MoveItemCmd) error
}

// CollectionCloser removes a collection and its tabs from the active workspace.
type CollectionCloser interface {
	Execute(ctx context.Context, cmd CloseCollectionCmd) error
}

// Command payloads.

// LoadCollectionCmd specifies the collection directory path to open.
type LoadCollectionCmd struct {
	Path string
}

// ReloadCollectionCmd specifies which collection to reload from disk.
type ReloadCollectionCmd struct {
	CollectionUID string
}

// SaveItemCmd specifies the item to commit and write to disk.
type SaveItemCmd struct {
	CollectionUID string
	ItemUID       string
	Format        model.CollectionFormat
}

// DeleteItemCmd specifies the item to delete from disk and workspace.
type DeleteItemCmd struct {
	CollectionUID string
	ItemUID       string
}

// NewItemCmd specifies the parameters for creating a new item.
type NewItemCmd struct {
	CollectionUID   string
	ParentFolderUID string
	Name            string
	Type            model.ItemType // model.ItemHTTPRequest or model.ItemFolder
	Method          string         // GET, POST, etc. (for request items)
	URL             string         // initial URL
	Format          model.CollectionFormat
}

// MoveItemCmd specifies the item to move or reorder.
type MoveItemCmd struct {
	CollectionUID   string
	ItemUID         string
	TargetFolderUID string // empty string means root of collection
	TargetSeq       int    // target sequence number (>0)
}

// CloseCollectionCmd specifies the collection to close.
type CloseCollectionCmd struct {
	CollectionUID string
}

// CollectionService coordinates collection loading, persistence, watching, and mutations.
type CollectionService struct {
	store    *bus.Store
	bus      *bus.Bus
	watchers map[string]*collection.Watcher
	mu       sync.Mutex
}

// NewCollectionService returns an initialized CollectionService.
func NewCollectionService(store *bus.Store, b *bus.Bus) *CollectionService {
	return &CollectionService{
		store:    store,
		bus:      b,
		watchers: make(map[string]*collection.Watcher),
	}
}

// Close terminates all active filesystem watchers.
func (s *CollectionService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for uid, w := range s.watchers {
		_ = w.Close()
		delete(s.watchers, uid)
	}
	return nil
}

// publish emits an event over the event bus if available.
func (s *CollectionService) publish(ev events.Event) {
	if s.bus != nil {
		s.bus.Publish(ev)
	}
}
