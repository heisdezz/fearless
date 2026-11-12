package presenters

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
	return p.findCollectionLocked(uid)
}

func (p *CollectionPresenter) findCollectionLocked(uid string) *model.Collection {
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

// SetFilter updates the filter search query.
func (p *CollectionPresenter) SetFilter(query string) {
	p.mu.Lock()
	p.filterQuery = query
	p.mu.Unlock()
	p.notifyChanged()
}

// Filter returns the current filter search query.
func (p *CollectionPresenter) Filter() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filterQuery
}

// CanMove checks if a source item can be moved to a target folder.
// An item cannot be moved into its own descendant.
func (p *CollectionPresenter) CanMove(colUID, sourceItemUID, targetFolderUID string) bool {
	if sourceItemUID == targetFolderUID {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	col := p.findCollectionLocked(colUID)
	if col == nil {
		return false
	}

	src := col.FindItem(sourceItemUID)
	if src == nil {
		return false
	}

	if targetFolderUID != "" {
		target := col.FindItem(targetFolderUID)
		if target == nil || !target.IsFolder() {
			return false
		}
	}

	if !src.IsFolder() {
		return true
	}

	if targetFolderUID == "" {
		return true
	}

	// Check if target is a descendant of source
	var isDescendant func(parent *model.Item, targetUID string) bool
	isDescendant = func(parent *model.Item, targetUID string) bool {
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

	return !isDescendant(src, targetFolderUID)
}

// BuildTree returns the hierarchical forest of TreeNode items for all loaded collections.
func (p *CollectionPresenter) BuildTree() []*TreeNode {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var roots []*TreeNode
	for _, col := range p.collections {
		colNode := &TreeNode{
			UID:           col.UID,
			Name:          col.Name,
			Type:          model.ItemFolder,
			Path:          col.Path,
			Level:         0,
			IsFolder:      true,
			IsExpanded:    p.expandedFolders[col.UID],
			IsSelected:    p.selectedUID == col.UID,
			CollectionUID: col.UID,
			Children:      p.buildItemNodes(col.Items, col.UID, 1),
		}
		roots = append(roots, colNode)
	}

	return roots
}

func (p *CollectionPresenter) buildItemNodes(items []*model.Item, colUID string, level int) []*TreeNode {
	var nodes []*TreeNode
	for _, it := range items {
		node := &TreeNode{
			UID:           it.UID,
			Name:          it.Name,
			Type:          it.Type,
			Level:         level,
			IsFolder:      it.IsFolder(),
			IsExpanded:    p.expandedFolders[it.UID],
			IsSelected:    p.selectedUID == it.UID,
			IsDirty:       it.Dirty(),
			CollectionUID: colUID,
			Item:          it,
		}

		if it.Request != nil {
			node.Method = it.Request.Method
			node.Path = it.Request.URL
		}

		if it.IsFolder() {
			node.Children = p.buildItemNodes(it.Children, colUID, level+1)
		}

		nodes = append(nodes, node)
	}
	return nodes
}

// VisibleNodes flattens the tree considering expansion state and the filter query.
func (p *CollectionPresenter) VisibleNodes() []*TreeNode {
	tree := p.BuildTree()
	var flat []*TreeNode
	for _, root := range tree {
		flat = append(flat, p.flattenVisible(root)...)
	}
	return flat
}

func (p *CollectionPresenter) flattenVisible(node *TreeNode) []*TreeNode {
	var nodes []*TreeNode

	filterQuery := strings.ToLower(strings.TrimSpace(p.Filter()))
	isFiltering := filterQuery != ""

	if isFiltering {
		// When filtering, return matching nodes and their ancestors.
		return p.filterNode(node, filterQuery)
	}

	nodes = append(nodes, node)
	if node.IsFolder && node.IsExpanded {
		for _, child := range node.Children {
			nodes = append(nodes, p.flattenVisible(child)...)
		}
	}
	return nodes
}

func (p *CollectionPresenter) filterNode(node *TreeNode, filterQuery string) []*TreeNode {
	var nodes []*TreeNode
	nameMatch := strings.Contains(strings.ToLower(node.Name), filterQuery)

	if node.IsFolder {
		var matchedChildren []*TreeNode
		for _, child := range node.Children {
			matchedChildren = append(matchedChildren, p.filterNode(child, filterQuery)...)
		}
		if nameMatch || len(matchedChildren) > 0 {
			copyNode := *node
			copyNode.IsExpanded = true
			copyNode.Children = matchedChildren
			nodes = append(nodes, &copyNode)
			nodes = append(nodes, matchedChildren...)
		}
	} else {
		methodMatch := strings.Contains(strings.ToLower(node.Method), filterQuery)
		url := ""
		if node.Item != nil && node.Item.Request != nil {
			url = node.Item.Request.URL
		}
		urlMatch := strings.Contains(strings.ToLower(url), filterQuery)
		if nameMatch || methodMatch || urlMatch {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// CreateCollection creates a new collection and adds it to the presenter.
func (p *CollectionPresenter) CreateCollection(ctx context.Context, name, path string, format model.CollectionFormat) (*model.Collection, error) {
	p.mu.Lock()
	if name == "" {
		name = "Untitled Collection"
	}
	if format == "" {
		format = model.FormatBru
	}
	uid := fmt.Sprintf("col-%d", time.Now().UnixNano())
	col := &model.Collection{
		UID:  uid,
		Name: name,
		Path: path,
		Config: model.CollectionConfig{
			Version: "1",
			Name:    name,
			Type:    "collection",
			Format:  format,
		},
	}
	p.collections = append(p.collections, col)
	p.expandedFolders[uid] = true
	p.selectedUID = uid
	p.mu.Unlock()
	p.notifyChanged()
	return col, nil
}

func (p *CollectionPresenter) resolveColUIDLocked(colUID, parentFolderUID string) (string, string) {
	if colUID == "" && parentFolderUID != "" {
		for _, c := range p.collections {
			if c.UID == parentFolderUID {
				return c.UID, ""
			}
			if it := c.FindItem(parentFolderUID); it != nil {
				return c.UID, parentFolderUID
			}
		}
	}
	if colUID == "" && len(p.collections) > 0 {
		if p.selectedUID != "" {
			for _, c := range p.collections {
				if c.UID == p.selectedUID {
					return c.UID, parentFolderUID
				}
				if it := c.FindItem(p.selectedUID); it != nil {
					if it.IsFolder() && parentFolderUID == "" {
						return c.UID, it.UID
					}
					return c.UID, parentFolderUID
				}
			}
		}
		return p.collections[0].UID, parentFolderUID
	}
	if colUID == "" {
		// Create default collection if none exists
		uid := fmt.Sprintf("col-%d", time.Now().UnixNano())
		c := &model.Collection{
			UID:  uid,
			Name: "Default Collection",
			Config: model.CollectionConfig{
				Version: "1",
				Name:    "Default Collection",
				Type:    "collection",
				Format:  model.FormatBru,
			},
		}
		p.collections = append(p.collections, c)
		p.expandedFolders[uid] = true
		return uid, parentFolderUID
	}
	return colUID, parentFolderUID
}

// CreateRequest dispatches creation of a new request item.
func (p *CollectionPresenter) CreateRequest(ctx context.Context, colUID, parentFolderUID, name, method, url string) (*model.Item, error) {
	p.mu.Lock()
	colUID, parentFolderUID = p.resolveColUIDLocked(colUID, parentFolderUID)
	creator := p.creator
	p.mu.Unlock()

	if name == "" {
		name = "Untitled Request"
	}
	if method == "" {
		method = "GET"
	}

	if creator != nil {
		item, err := creator.CreateItem(ctx, colUID, parentFolderUID, name, model.ItemHTTPRequest, method, url)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		col := p.findCollectionLocked(colUID)
		if col != nil && item != nil && col.FindItem(item.UID) == nil {
			if parentFolderUID != "" {
				if parent := col.FindItem(parentFolderUID); parent != nil && parent.IsFolder() {
					parent.Children = append(parent.Children, item)
				} else {
					col.Items = append(col.Items, item)
				}
			} else {
				col.Items = append(col.Items, item)
			}
		}
		if item != nil {
			p.selectedUID = item.UID
		}
		if parentFolderUID != "" {
			p.expandedFolders[parentFolderUID] = true
		}
		p.mu.Unlock()
		p.notifyChanged()
		return item, nil
	}

	// In-memory creation fallback
	p.mu.Lock()
	col := p.findCollectionLocked(colUID)
	if col == nil {
		p.mu.Unlock()
		return nil, errors.New("collection not found")
	}
	item := &model.Item{
		UID:  fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Type: model.ItemHTTPRequest,
		Name: name,
		Request: &model.Request{
			Method: method,
			URL:    url,
		},
	}
	if parentFolderUID != "" {
		if parent := col.FindItem(parentFolderUID); parent != nil && parent.IsFolder() {
			parent.Children = append(parent.Children, item)
			p.expandedFolders[parentFolderUID] = true
		} else {
			col.Items = append(col.Items, item)
		}
	} else {
		col.Items = append(col.Items, item)
	}
	p.selectedUID = item.UID
	p.mu.Unlock()
	p.notifyChanged()
	return item, nil
}

// CreateFolder dispatches creation of a new folder item.
func (p *CollectionPresenter) CreateFolder(ctx context.Context, colUID, parentFolderUID, name string) (*model.Item, error) {
	p.mu.Lock()
	colUID, parentFolderUID = p.resolveColUIDLocked(colUID, parentFolderUID)
	creator := p.creator
	p.mu.Unlock()

	if name == "" {
		name = "New Folder"
	}

	if creator != nil {
		item, err := creator.CreateItem(ctx, colUID, parentFolderUID, name, model.ItemFolder, "", "")
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		col := p.findCollectionLocked(colUID)
		if col != nil && item != nil && col.FindItem(item.UID) == nil {
			if parentFolderUID != "" {
				if parent := col.FindItem(parentFolderUID); parent != nil && parent.IsFolder() {
					parent.Children = append(parent.Children, item)
				} else {
					col.Items = append(col.Items, item)
				}
			} else {
				col.Items = append(col.Items, item)
			}
		}
		if item != nil {
			p.selectedUID = item.UID
			p.expandedFolders[item.UID] = true
		}
		if parentFolderUID != "" {
			p.expandedFolders[parentFolderUID] = true
		}
		p.mu.Unlock()
		p.notifyChanged()
		return item, nil
	}

	// In-memory creation fallback
	p.mu.Lock()
	col := p.findCollectionLocked(colUID)
	if col == nil {
		p.mu.Unlock()
		return nil, errors.New("collection not found")
	}
	folder := &model.Item{
		UID:  fmt.Sprintf("folder-%d", time.Now().UnixNano()),
		Type: model.ItemFolder,
		Name: name,
	}
	if parentFolderUID != "" {
		if parent := col.FindItem(parentFolderUID); parent != nil && parent.IsFolder() {
			parent.Children = append(parent.Children, folder)
			p.expandedFolders[parentFolderUID] = true
		} else {
			col.Items = append(col.Items, folder)
		}
	} else {
		col.Items = append(col.Items, folder)
	}
	p.selectedUID = folder.UID
	p.expandedFolders[folder.UID] = true
	p.mu.Unlock()
	p.notifyChanged()
	return folder, nil
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
