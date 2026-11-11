package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// RunFolder executes a sequence of requests within a collection or folder.
func (s *RunnerService) RunFolder(ctx context.Context, cmd RunFolderCmd) (*RunnerSummary, error) {
	col, err := s.resolveCollection(cmd)
	if err != nil {
		return nil, err
	}

	items, err := s.collectItems(col, cmd.FolderUID, cmd.Recursive, cmd.FilterTags)
	if err != nil {
		return nil, err
	}

	runnerUID := cmd.RunnerUID
	if runnerUID == "" {
		runnerUID = newRunnerUID()
	}

	runCtx, cancel := context.WithCancel(ctx)
	active := &activeRun{
		uid:           runnerUID,
		collectionUID: col.UID,
		cancel:        cancel,
	}

	s.mu.Lock()
	s.active[runnerUID] = active
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.active, runnerUID)
		s.mu.Unlock()
		cancel()
	}()

	summary := &RunnerSummary{
		RunnerUID:     runnerUID,
		CollectionUID: col.UID,
	}
	start := time.Now()

	// Build index of items by Name and UID for setNextRequest jumps
	itemByName := make(map[string]*model.Item)
	itemByUID := make(map[string]*model.Item)
	col.Walk(func(it *model.Item) error {
		if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
			itemByName[it.Name] = it
			itemByUID[it.UID] = it
		}
		return nil
	})

	for i := 0; i < len(items); i++ {
		if runCtx.Err() != nil {
			ev := events.NewError(events.ScopeRunner, runCtx.Err()).
				WithRequest(runnerUID, "", col.UID)
			s.publish(ev)
			break
		}

		active.mu.Lock()
		stopped := active.stopped
		nextTarget := active.nextOverride
		active.nextOverride = ""
		skipCurrent := active.skipNext
		active.skipNext = false
		active.mu.Unlock()

		if stopped {
			break
		}

		// Handle setNextRequest jump
		if nextTarget != "" {
			targetItem := itemByName[nextTarget]
			if targetItem == nil {
				targetItem = itemByUID[nextTarget]
			}
			if targetItem != nil {
				// Jump to target item
				targetIdx := -1
				for idx, it := range items {
					if it.UID == targetItem.UID {
						targetIdx = idx
						break
					}
				}
				if targetIdx >= 0 {
					i = targetIdx
				}
			}
		}

		if skipCurrent {
			summary.Skipped++
			continue
		}

		it := items[i]
		summary.Total++

		// Apply optional inter-request delay
		if cmd.DelayMs > 0 && summary.Total > 1 {
			select {
			case <-time.After(time.Duration(cmd.DelayMs) * time.Millisecond):
			case <-runCtx.Done():
				ev := events.NewError(events.ScopeRunner, runCtx.Err()).
					WithRequest(runnerUID, it.UID, col.UID)
				s.publish(ev)
				break
			}
		}

		sendCmd := SendRequestCmd{
			CollectionUID: col.UID,
			ItemUID:       it.UID,
			Collection:    col,
			Item:          it,
			Env:           cmd.Env,
			RuntimeVars:   cmd.RuntimeVars,
		}

		res, sendErr := s.sender.SendSync(runCtx, sendCmd)
		if sendErr != nil {
			summary.Failed++
			if cmd.Bail {
				break
			}
			continue
		}

		summary.Results = append(summary.Results, res)
		if res.Failed() || res.Status >= 400 {
			summary.Failed++
			if cmd.Bail {
				break
			}
		} else {
			summary.Passed++
		}
	}

	summary.Duration = time.Since(start)
	return summary, nil
}

func (s *RunnerService) resolveCollection(cmd RunFolderCmd) (*model.Collection, error) {
	if cmd.Collection != nil {
		return cmd.Collection, nil
	}
	if s.store == nil {
		return nil, errors.New("commands: store not available")
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return nil, fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}
	return col, nil
}

func (s *RunnerService) collectItems(
	col *model.Collection,
	folderUID string,
	recursive bool,
	filterTags []string,
) ([]*model.Item, error) {
	var candidates []*model.Item

	if folderUID == "" {
		if recursive {
			_ = col.Walk(func(it *model.Item) error {
				if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
					candidates = append(candidates, it)
				}
				return nil
			})
		} else {
			for _, it := range col.Items {
				if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
					candidates = append(candidates, it)
				}
			}
		}
	} else {
		folder := col.FindItem(folderUID)
		if folder == nil {
			return nil, fmt.Errorf("%w: folder %s not found", ErrItemNotFound, folderUID)
		}
		if recursive {
			_ = walkFolder(folder, func(it *model.Item) error {
				if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
					candidates = append(candidates, it)
				}
				return nil
			})
		} else {
			for _, child := range folder.Children {
				if child.Type == model.ItemHTTPRequest || (child.Type == "" && child.Request != nil) {
					candidates = append(candidates, child)
				}
			}
		}
	}

	if len(filterTags) == 0 {
		return candidates, nil
	}

	var filtered []*model.Item
	for _, it := range candidates {
		if hasAnyTag(it.Tags, filterTags) {
			filtered = append(filtered, it)
		}
	}
	return filtered, nil
}

func hasAnyTag(tags []string, filter []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}
	for _, f := range filter {
		if tagSet[f] {
			return true
		}
	}
	return false
}

func walkFolder(it *model.Item, fn func(*model.Item) error) error {
	if it == nil {
		return nil
	}
	if err := fn(it); err != nil {
		return err
	}
	for _, child := range it.Children {
		if err := walkFolder(child, fn); err != nil {
			return err
		}
	}
	return nil
}
