package collection

import (
	"sort"
	"strings"

	"fearless/internal/core/model"
)

// SortItems sorts a slice of items in-place using Bruno's sortByNameThenSequence algorithm:
//  1. Items are first sorted alphabetically by Name (case-insensitive).
//  2. Items with a valid positive Seq (> 0) are placed at 1-based index (Seq - 1).
//  3. Equal sequences or multiple items at the same index preserve alphabetical stability.
func SortItems(items []*model.Item) {
	if len(items) <= 1 {
		return
	}

	sorted := SortByNameThenSequence(items)
	copy(items, sorted)
}

// SortByNameThenSequence returns a new sorted slice of items matching Bruno's ordering.
func SortByNameThenSequence(items []*model.Item) []*model.Item {
	if len(items) <= 1 {
		res := make([]*model.Item, len(items))
		copy(res, items)
		return res
	}

	// 1. Initial alphabetical sort by Name
	alphabetical := make([]*model.Item, len(items))
	copy(alphabetical, items)
	sort.SliceStable(alphabetical, func(i, j int) bool {
		ni := strings.ToLower(alphabetical[i].Name)
		nj := strings.ToLower(alphabetical[j].Name)
		if ni != nj {
			return ni < nj
		}
		return alphabetical[i].Name < alphabetical[j].Name
	})

	// 2. Separate into items without valid seq and with valid seq
	var withoutSeq []*model.Item
	var withSeq []*model.Item

	for _, it := range alphabetical {
		if it.Seq != nil && *it.Seq > 0 {
			withSeq = append(withSeq, it)
		} else {
			withoutSeq = append(withoutSeq, it)
		}
	}

	// Sort withSeq by Seq ascending, stable
	sort.SliceStable(withSeq, func(i, j int) bool {
		return *withSeq[i].Seq < *withSeq[j].Seq
	})

	// 3. Insert items with seq at position (seq - 1)
	result := make([]*model.Item, len(withoutSeq))
	copy(result, withoutSeq)

	for _, item := range withSeq {
		pos := *item.Seq - 1
		if pos < 0 {
			pos = 0
		}
		if pos >= len(result) {
			result = append(result, item)
		} else {
			// Insert at pos
			result = append(result[:pos], append([]*model.Item{item}, result[pos:]...)...)
		}
	}

	return result
}
