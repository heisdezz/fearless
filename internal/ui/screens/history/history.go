// Package history provides the UI screen for viewing and replaying request
// execution history recorded by the history store.
package history

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/history"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// Snapshot is the pure-data presentation model representing the history screen state.
type Snapshot struct {
	Entries     []history.Entry
	SelectedID  string
	FilterQuery string
	FailedOnly  bool
	Summary     history.Summary
}

// Actions captures user intent callbacks from the history screen.
type Actions struct {
	OnSelectEntry   func(id string)
	OnDeleteEntry   func(id string)
	OnClearHistory  func()
	OnReplayEntry   func(id string)
	OnFilterChanged func(query string)
	OnToggleFailed  func(failedOnly bool)
}

// Screen is the history screen component.
type Screen struct {
	Snapshot Snapshot
	Actions  Actions
}

// NewScreen creates an initialized Screen.
func NewScreen(snap Snapshot, actions Actions) *Screen {
	return &Screen{
		Snapshot: snap,
		Actions:  actions,
	}
}

// View renders the history screen frame.
func (s *Screen) View(c *ui.Context) ui.Element {
	tk := lobetheme.TokensOf(c)
	snap := s.Snapshot
	actions := s.Actions

	return ui.Column(c).
		Fill().
		Background(tk.BgLayout).
		Padding(16, 20).
		Gap(14).
		Children(func() {
			// 1. Header Toolbar
			s.renderHeader(c, snap, actions)

			// 2. Main content split: List on left, details on right if selected
			var selectedEntry *history.Entry
			if snap.SelectedID != "" {
				for i := range snap.Entries {
					if snap.Entries[i].ID == snap.SelectedID {
						selectedEntry = &snap.Entries[i]
						break
					}
				}
			}

			ui.Row(c).Fill().Grow(1).Gap(16).Children(func() {
				// Entries list (takes full width if no selection, or flex-1 when detail is open)
				s.renderEntriesList(c, snap, actions, selectedEntry != nil)

				// Details side-panel when an entry is selected
				if selectedEntry != nil {
					s.renderEntryDetail(c, selectedEntry, actions)
				}
			})
		})
}

func (s *Screen) renderHeader(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)

	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Text(c, "Request History").
			FontSize(16).
			Bold().
			TextColor(tk.Text)

		// Summary indicators
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(2, 8).Radius(tk.BorderRadius).Background(tk.BgContainer).Children(func() {
			ui.Text(c, fmt.Sprintf("Total: %d", snap.Summary.Total)).
				FontSize(11).
				TextColor(tk.TextSecondary)

			if snap.Summary.Passed > 0 {
				ui.Text(c, fmt.Sprintf("Passed: %d", snap.Summary.Passed)).
					FontSize(11).
					Bold().
					TextColor(tk.Success)
			}
			if snap.Summary.Failed > 0 {
				ui.Text(c, fmt.Sprintf("Failed: %d", snap.Summary.Failed)).
					FontSize(11).
					Bold().
					TextColor(tk.Error)
			}
		})

		ui.Box(c).Grow(1)

		// Clear history button
		if snap.Summary.Total > 0 {
			widgets.Button(c, "Clear History", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Default,
			}).OnClick(func() {
				if actions.OnClearHistory != nil {
					actions.OnClearHistory()
				}
			})
		}
	})

	// Filter & Options Bar
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		q := snap.FilterQuery
		field.SearchBar(c, &q, field.SearchBarProps{
			Placeholder: "Filter history by URL or name...",
			Size:        lobetheme.Small,
			OnSearch: func(query string) {
				if actions.OnFilterChanged != nil {
					actions.OnFilterChanged(query)
				}
			},
		})
		if q != snap.FilterQuery && actions.OnFilterChanged != nil {
			actions.OnFilterChanged(q)
		}

		// Failed Only Toggle
		failedOnly := snap.FailedOnly
		choice.Switch(c, &failedOnly, choice.SwitchProps{
			Size: lobetheme.Small,
		})
		if failedOnly != snap.FailedOnly && actions.OnToggleFailed != nil {
			actions.OnToggleFailed(failedOnly)
		}
		ui.Text(c, "Failed Only").FontSize(12).TextColor(tk.TextSecondary)
	})
}

func (s *Screen) renderEntriesList(c *ui.Context, snap Snapshot, actions Actions, hasDetail bool) {
	tk := lobetheme.TokensOf(c)

	container := ui.Column(c).FillHeight().Background(tk.BgContainer).Radius(tk.BorderRadius).Padding(8)
	if hasDetail {
		container.Width(420).Shrink(0)
	} else {
		container.FillWidth().Grow(1)
	}

	container.Children(func() {
		if len(snap.Entries) == 0 {
			ui.Column(c).
				Fill().
				Grow(1).
				Center().
				Gap(6).
				Children(func() {
					ui.Text(c, "No request history recorded").
						FontSize(13).
						Bold().
						TextColor(tk.TextSecondary)
					ui.Text(c, "Executed requests and responses will be logged here").
						FontSize(11).
						TextColor(tk.TextTertiary)
				})
			return
		}

		// List of history entries
		ui.Scroll(c).Fill().Grow(1).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, entry := range snap.Entries {
					e := entry
					s.renderEntryRow(c, e, snap.SelectedID == e.ID, actions)
				}
			})
		})
	})
}

func (s *Screen) renderEntryRow(c *ui.Context, e history.Entry, isSelected bool, actions Actions) {
	tk := lobetheme.TokensOf(c)

	row := ui.Row(c).
		FillWidth().
		Height(34).
		AlignItems(ui.Center).
		Padding(0, 10).
		Radius(tk.BorderRadius).
		Gap(10).
		Cursor(ui.CursorPointer).
		OnClick(func() {
			if actions.OnSelectEntry != nil {
				actions.OnSelectEntry(e.ID)
			}
		})

	switch {
	case isSelected:
		row.Background(tk.BgElevated)
	case row.Hovered():
		row.Background(tk.BgTextHover)
	}

	row.Children(func() {
		// Method Badge
		widgets.RequestMethod(c, e.Method)

		// URL and Item Name
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			name := e.ItemName
			if name == "" {
				name = e.URL
			}
			ui.Text(c, name).
				SingleLine().
				FontSize(12).
				Bold().
				TextColor(tk.Text)

			if e.ItemName != "" && e.URL != "" {
				ui.Text(c, e.URL).
					SingleLine().
					FontSize(10).
					TextColor(tk.TextSecondary)
			}
		})

		// Status Badge
		statusColor := tk.Success
		if e.Failed() {
			statusColor = tk.Error
		} else if e.Status >= 300 && e.Status < 400 {
			statusColor = tk.Warning
		}

		statusLabel := fmt.Sprintf("%d", e.Status)
		if e.StatusText != "" {
			statusLabel = fmt.Sprintf("%d %s", e.Status, e.StatusText)
		} else if e.Status == 0 && e.Error != "" {
			statusLabel = "Error"
		}

		ui.Box(c).Padding(1, 6).Radius(tk.BorderRadius).Background(tk.BgElevated).Children(func() {
			ui.Text(c, statusLabel).
				FontSize(11).
				Bold().
				TextColor(statusColor)
		})

		// Duration
		if e.DurationMs > 0 {
			ui.Text(c, fmt.Sprintf("%d ms", e.DurationMs)).
				FontSize(11).
				TextColor(tk.TextSecondary)
		}

		// Timestamp
		timeStr := e.ExecutedAt.Format("15:04:05")
		if time.Since(e.ExecutedAt) > 24*time.Hour {
			timeStr = e.ExecutedAt.Format("Jan 02 15:04")
		}
		ui.Text(c, timeStr).
			FontSize(10).
			TextColor(tk.TextTertiary)

		// Replay action button
		if actions.OnReplayEntry != nil {
			widgets.Button(c, "Replay", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Default,
			}).OnClick(func() {
				actions.OnReplayEntry(e.ID)
			})
		}
	})
}

func (s *Screen) renderEntryDetail(c *ui.Context, e *history.Entry, actions Actions) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).
		Fill().
		Grow(1).
		Background(tk.BgContainer).
		Radius(tk.BorderRadius).
		Padding(16).
		Gap(12).
		Children(func() {
			// Detail Header
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				widgets.RequestMethod(c, e.Method)

				ui.Text(c, e.URL).
					FontSize(13).
					Bold().
					TextColor(tk.Text).
					Grow(1)

				if actions.OnDeleteEntry != nil {
					widgets.Button(c, "Delete", widgets.ButtonProps{
						Size:    lobetheme.Small,
						Variant: action.Default,
					}).OnClick(func() {
						actions.OnDeleteEntry(e.ID)
					})
				}

				if actions.OnSelectEntry != nil {
					widgets.IconButton(c, widgets.IconButtonProps{
						Icon:    lucide.X(),
						Label:   "Close entry",
						Tooltip: "Close entry",
						Size:    lobetheme.Small,
						Variant: action.Text,
					}).OnClick(func() {
						actions.OnSelectEntry("")
					})
				}
			})

			// Meta info strip
			ui.Row(c).FillWidth().Gap(14).Padding(8, 12).Radius(tk.BorderRadius).Background(tk.BgElevated).Children(func() {
				ui.Column(c).Children(func() {
					ui.Text(c, "Status").FontSize(10).TextColor(tk.TextSecondary)
					color := tk.Success
					if e.Failed() {
						color = tk.Error
					}
					ui.Text(c, fmt.Sprintf("%d %s", e.Status, e.StatusText)).
						FontSize(12).
						Bold().
						TextColor(color)
				})

				ui.Column(c).Children(func() {
					ui.Text(c, "Duration").FontSize(10).TextColor(tk.TextSecondary)
					ui.Text(c, fmt.Sprintf("%d ms", e.DurationMs)).
						FontSize(12).
						TextColor(tk.Text)
				})

				ui.Column(c).Children(func() {
					ui.Text(c, "Size").FontSize(10).TextColor(tk.TextSecondary)
					ui.Text(c, formatSize(e.Size)).
						FontSize(12).
						TextColor(tk.Text)
				})

				if e.Environment != "" {
					ui.Column(c).Children(func() {
						ui.Text(c, "Environment").FontSize(10).TextColor(tk.TextSecondary)
						ui.Text(c, e.Environment).
							FontSize(12).
							TextColor(tk.Text)
					})
				}
			})

			// Error message if present
			if e.Error != "" {
				ui.Row(c).FillWidth().Padding(8, 12).Radius(tk.BorderRadius).Background(tk.ErrorBg).Children(func() {
					ui.Text(c, "Error: "+e.Error).
						FontSize(12).
						TextColor(tk.Error)
				})
			}

			// Scrollable Headers and Details
			ui.Scroll(c).Fill().Grow(1).Children(func() {
				ui.Column(c).FillWidth().Gap(14).Children(func() {
					// Response Headers
					if len(e.ResponseHeaders) > 0 {
						ui.Text(c, "Response Headers").
							FontSize(12).
							Bold().
							TextColor(tk.Text)

						s.renderHeadersTable(c, e.ResponseHeaders)
					}

					// Request Headers
					if len(e.RequestHeaders) > 0 {
						ui.Text(c, "Request Headers").
							FontSize(12).
							Bold().
							TextColor(tk.Text)

						s.renderHeadersTable(c, e.RequestHeaders)
					}
				})
			})
		})
}

func (s *Screen) renderHeadersTable(c *ui.Context, headers map[string][]string) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).FillWidth().Radius(tk.BorderRadius).Background(tk.BgLayout).Padding(6, 10).Gap(4).Children(func() {
		for key, vals := range headers {
			k := key
			v := strings.Join(vals, ", ")
			ui.Row(c).FillWidth().Gap(10).Children(func() {
				ui.Box(c).Width(140).Children(func() {
					ui.Text(c, k).
						FontSize(11).
						Bold().
						TextColor(tk.TextSecondary)
				})
				ui.Text(c, v).
					FontSize(11).
					TextColor(tk.Text).
					Grow(1)
			})
		}
	})
}

func formatSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f kB", float64(b)/1024.0)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/(1024.0*1024.0))
}

var _ = uitheme.MethodColorFor
