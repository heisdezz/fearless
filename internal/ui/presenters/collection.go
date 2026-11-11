package presenters

import (
	"context"
	"errors"
	"strings"
	"sync"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// TreeNode represents a node in the collection sidebar tree.
type TreeNode struct {
	UID           string
	Name          string
	Type          model.ItemType
	Method        string
	Path          string
	Level         int
	IsFolder      bool
	IsExpanded    bool
	IsSelected    bool
	IsDirty       bool
	CollectionUID string
	Item          *model.Item
	Children      []*TreeNode
}

// CollectionLoaderPort is the consumer port to open collections from disk.
type CollectionLoaderPort interface {
	Load(ctx context.Context, path string) (string, error)
}

// ItemCreatorPort is the consumer port to create requests or folders.
type ItemCreatorPort interface {
	CreateItem(ctx context.Context, colUID, parentFolderUID, name string, itemType model.ItemType, method, url string) (*model.Item, error)
}

// ItemDeleterPort is the consumer port to delete items.
type ItemDeleterPort interface {
	DeleteItem(ctx context.Context, colUID, itemUID string) error
}

// ItemMoverPort is the consumer port to move or reorder items.
type ItemMoverPort interface {
	MoveItem(ctx context.Context, colUID, itemUID, targetFolderUID string, targetSeq *int) error
}

// CollectionPresenter coordinates the sidebar collection explorer view-model.
type CollectionPresenter struct {
	mu              sync.RWMutex
	collections     []*model.Collection
	expandedFolders map[string]bool
	selectedUID     string
	filterQuery     string
	loader          CollectionLoaderPort
	creator         ItemCreatorPort
	deleter         ItemDeleterPort
	mover           ItemMoverPort
	onChanged       func()
}

// NewCollectionPresenter creates an initialized CollectionPresenter.
func NewCollectionPresenter(
	loader CollectionLoaderPort,
	creator ItemCreatorPort,
	deleter ItemDeleterPort,
	mover ItemMoverPort,
) *CollectionPresenter {
	return &CollectionPresenter{
		expandedFolders: make(map[string]bool),
		loader:          loader,
		creator:         creator,
		deleter:         deleter,
		mover:           mover,
	}
}

// SetCollections updates the list of loaded collections.
func (p *CollectionPresenter) SetCollections(cols []*model.Collection) {
	p.mu.Lock()
	p.collections = cols
	// By default, newly added collections are expanded.
	for _, c := range cols {
		if _, ok := p.expandedFolders[c.UID]; !ok {
			p.expandedFolders[c.UID] = true
		}
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// Collections returns the open collections.
func (p *CollectionPresenter) Collections() []*model.Collection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*model.Collection, len(p.collections))
	copy(out, p.collections)
	return out
}

// FindCollection returns the collection matching uid, or nil.
func (p *CollectionPresenter) FindCollection(uid string) *model.Collection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, c := range p.collections {
		if c.UID == uid {
			return c
		}
	}
	return nil
}

// FindItem searches for an item across all loaded collections.
func (p *CollectionPresenter) FindItem(uid string) (*model.Item, *model.Collection) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, c := range p.collections {
		if it := c.FindItem(uid); it != nil {
			return it, c
		}
	}
	return nil, nil
}

// SelectedUID returns the currently highlighted item or collection UID.
func (p *CollectionPresenter) SelectedUID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedUID
}

// Select sets the selected item or folder UID.
func (p *CollectionPresenter) Select(uid string) {
	p.mu.Lock()
	p.selectedUID = uid
	p.mu.Unlock()
	p.notifyChanged()
}

// ClearSelection clears the active selection.
func (p *CollectionPresenter) ClearSelection() {
	p.mu.Lock()
	p.selectedUID = ""
	p.mu.Unlock()
	p.notifyChanged()
}

// IsExpanded reports whether a folder or collection is expanded.
func (p *CollectionPresenter) IsExpanded(uid string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.expandedFolders[uid]
}

// SetExpanded sets the expansion state for a folder or collection.
func (p *CollectionPresenter) SetExpanded(uid string, expanded bool) {
	p.mu.Lock()
	p.expandedFolders[uid] = expanded
	p.mu.Unlock()
	p.notifyChanged()
}

// ToggleExpanded flips the expansion state of a folder or collection.
func (p *CollectionPresenter) ToggleExpanded(uid string) {
	p.mu.Lock()
	p.expandedFolders[uid] = !p.expandedFolders[uid]
	p.mu.Unlock()
	p.notifyChanged()
}

// Filter returns the active search query.
func (p *CollectionPresenter) Filter() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filterQuery
}

// SetFilter updates the search filter.
func (p *CollectionPresenter) SetFilter(query string) {
	p.mu.Lock()
	p.filterQuery = strings.TrimSpace(query)
	p.mu.Unlock()
	p.notifyChanged()
}

// CanMove checks if moving itemUID to targetFolderUID within collectionUID is valid.
func (p *CollectionPresenter) CanMove(collectionUID, itemUID, targetFolderUID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var targetCol *model.Collection
	for _, c := range p.collections {
		if c.UID == collectionUID {
			targetCol = c
			break
		}
	}
	if targetCol == nil {
		return false
	}

	item := targetCol.FindItem(itemUID)
	if item == nil {
		return false
	}

	// If moving to root folder (empty UID), valid as long as it's not already at root without seq change
	if targetFolderUID == "" {
		return true
	}

	// Target folder must exist and be a folder
	targetFolder := targetCol.FindItem(targetFolderUID)
	if targetFolder == nil || !targetFolder.IsFolder() {
		return false
	}

	// Cannot move into itself
	if targetFolderUID == itemUID {
		return false
	}

	// If item is a folder, cannot move into any descendant
	if item.IsFolder() && isDescendant(item, targetFolderUID) {
		return false
	}

	return true
}

func isDescendant(parent *model.Item, targetUID string) bool {
	for _, child := range parent.Children {
		if child.UID == targetUID {
			return true
		}
		if child.IsFolder() && isDescendant(child, targetUID) {
			return true
		}
	}
	return false
}

// VisibleNodes returns the flat ordered list of visible tree nodes for rendering.
func (p *CollectionPresenter) VisibleNodes() []*TreeNode {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*TreeNode
	q := strings.ToLower(p.filterQuery)

	for _, col := range p.collections {
		colNode := &TreeNode{
			UID:           col.UID,
			Name:          col.Name,
			Type:          model.ItemFolder,
			Path:          col.Path,
			Level:         0,
			IsFolder:      true,
			IsExpanded:    p.expandedFolders[col.UID] || q != "",
			IsSelected:    col.UID == p.selectedUID,
			IsDirty:       col.Dirty,
			CollectionUID: col.UID,
		}

		childFiltered := p.buildItemNodes(col.Items, col.UID, 1, q)

		// When filtering, only include collection if it or its children match
		if q != "" {
			colMatches := strings.Contains(strings.ToLower(col.Name), q)
			if colMatches || len(childFiltered) > 0 {
				colNode.IsExpanded = true
				result = append(result, colNode)
				result = append(result, childFiltered...)
			}
		} else {
			result = append(result, colNode)
			if colNode.IsExpanded {
				result = append(result, childFiltered...)
			}
		}
	}

	return result
}

func (p *CollectionPresenter) buildItemNodes(items []*model.Item, colUID string, level int, filterQuery string) []*TreeNode {
	var nodes []*TreeNode

	for _, item := range items {
		if item == nil {
			continue
		}

		method := ""
		url := ""
		if req := item.Effective(); req != nil {
			method = req.Method
			url = req.URL
		}

		node := &TreeNode{
			UID:           item.UID,
			Name:          item.Name,
			Type:          item.Type,
			Method:        method,
			Path:          item.Path,
			Level:         level,
			IsFolder:      item.IsFolder(),
			IsExpanded:    p.expandedFolders[item.UID],
			IsSelected:    item.UID == p.selectedUID,
			IsDirty:       item.Dirty(),
			CollectionUID: colUID,
			Item:          item,
		}

		if item.IsFolder() {
			childNodes := p.buildItemNodes(item.Children, colUID, level+1, filterQuery)
			node.Children = childNodes

			if filterQuery != "" {
				nameMatches := strings.Contains(strings.ToLower(item.Name), filterQuery)
				if nameMatches || len(childNodes) > 0 {
					node.IsExpanded = true
					nodes = append(nodes, node)
					nodes = append(nodes, childNodes...)
				}
			} else {
				nodes = append(nodes, node)
				if node.IsExpanded {
					nodes = append(nodes, childNodes...)
				}
			}
		} else {
			// Request node
			if filterQuery != "" {
				nameMatch := strings.Contains(strings.ToLower(item.Name), filterQuery)
				methodMatch := strings.Contains(strings.ToLower(method), filterQuery)
				urlMatch := strings.Contains(strings.ToLower(url), filterQuery)
				if nameMatch || methodMatch || urlMatch {
					nodes = append(nodes, node)
				}
			} else {
				nodes = append(nodes, node)
			}
		}
	}

	return nodes
}

// CreateRequest dispatches creation of a new request item.
func (p *CollectionPresenter) CreateRequest(ctx context.Context, colUID, parentFolderUID, name, method, url string) (*model.Item, error) {
	if p.creator == nil {
		return nil, nil
	}
	item, err := p.creator.CreateItem(ctx, colUID, parentFolderUID, name, model.ItemHTTPRequest, method, url)
	if err == nil && parentFolderUID != "" {
		p.SetExpanded(parentFolderUID, true)
	}
	p.notifyChanged()
	return item, err
}

// CreateFolder dispatches creation of a new folder item.
func (p *CollectionPresenter) CreateFolder(ctx context.Context, colUID, parentFolderUID, name string) (*model.Item, error) {
	if p.creator == nil {
		return nil, nil
	}
	item, err := p.creator.CreateItem(ctx, colUID, parentFolderUID, name, model.ItemFolder, "", "")
	if err == nil && parentFolderUID != "" {
		p.SetExpanded(parentFolderUID, true)
	}
	p.notifyChanged()
	return item, err
}

// DeleteItem dispatches item deletion.
func (p *CollectionPresenter) DeleteItem(ctx context.Context, colUID, itemUID string) error {
	if p.deleter == nil {
		return nil
	}
	err := p.deleter.DeleteItem(ctx, colUID, itemUID)
	if err == nil {
		p.mu.Lock()
		delete(p.expandedFolders, itemUID)
		if p.selectedUID == itemUID {
			p.selectedUID = ""
		}
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// MoveItem dispatches item move or reordering.
func (p *CollectionPresenter) MoveItem(ctx context.Context, colUID, itemUID, targetFolderUID string, targetSeq *int) error {
	if p.mover == nil {
		return nil
	}
	if !p.CanMove(colUID, itemUID, targetFolderUID) {
		return errors.New("invalid move operation")
	}
	err := p.mover.MoveItem(ctx, colUID, itemUID, targetFolderUID, targetSeq)
	if err == nil && targetFolderUID != "" {
		p.SetExpanded(targetFolderUID, true)
	}
	p.notifyChanged()
	return err
}

// LoadCollection dispatches loading a collection directory.
func (p *CollectionPresenter) LoadCollection(ctx context.Context, path string) (string, error) {
	if p.loader == nil {
		return "", nil
	}
	uid, err := p.loader.Load(ctx, path)
	p.notifyChanged()
	return uid, err
}

// HandleEvent processes collection events.
func (p *CollectionPresenter) HandleEvent(ev events.Event) {
	switch e := ev.(type) {
	case events.CollectionChanged:
		p.mu.Lock()
		for _, removedUID := range e.Change.Removed {
			delete(p.expandedFolders, removedUID)
			if p.selectedUID == removedUID {
				p.selectedUID = ""
			}
		}
		p.mu.Unlock()
		p.notifyChanged()
	}
}

// OnChanged registers a callback hook for invalidation.
func (p *CollectionPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *CollectionPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
