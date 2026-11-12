// Package response renders the response viewer.
//
// The screen is a pure function of a presenter snapshot: it reads
// ResponsePresenter, draws it, and reports clicks through callbacks. It never
// mutates domain state inside a View closure, which is the acceptance criterion
// for every UI-SCREEN-* package (docs/progress/packages.md L237).
//
// Layout, top to bottom:
//
//	status seam   200 OK · 45 ms · 512 B · (error text when failed)
//	sub-tabs     Body | Headers | Timeline | Tests
//	pane         the selected view
//
// Both chrome rows are hairline seams rather than boxed strips: the status
// seam carries the one colour that means something (the status category) and
// the sub-tab seam carries the accent underline on the active tab. Every
// figure on them is set in the data face — theme.DataText — so codes, timings
// and sizes line up on tabular digits.
//
// Source: bruno-app/src/components/ResponsePane
package response

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/theme"
)

// Props configure the screen.
type Props struct {
	// Presenter is the view-model. It is read, never written, by the render.
	Presenter *presenters.ResponsePresenter
	// OnSelectTab reports a sub-tab click. When nil, tabs still render but do
	// nothing, which keeps the screen usable in a golden test.
	OnSelectTab func(tab presenters.ResponseSubTab)
	// MinHeight bounds the pane so the screen does not collapse inside a splitter.
	MinHeight float32
}

// DefaultMinHeight is the pane height used when Props.MinHeight is zero.
const DefaultMinHeight = 220

// Screen renders the whole response viewer.
func Screen(c *ui.Context, props Props) ui.Element {
	tk := lobetheme.TokensOf(c)

	p := props.Presenter
	if p == nil {
		// A nil presenter is a wiring mistake, not a user-visible state. Rendering
		// an empty pane keeps the surrounding shell intact instead of panicking
		// mid-frame.
		return ui.Box(c).Fill().MinHeight(DefaultMinHeight).Background(tk.BgContainer)
	}

	minHeight := props.MinHeight
	if minHeight <= 0 {
		minHeight = DefaultMinHeight
	}

	// Fill the splitter pane and keep MinHeight only as the floor: the
	// pane's share of the window is larger than DefaultMinHeight at the
	// §8.1 budget, so a fixed Height would strand the rest of the pane as
	// bare background.
	return ui.Column(c).
		Fill().
		MinHeight(minHeight).
		Background(tk.BgContainer).
		Children(func() {
			StatusRow(c, p)
			SubTabs(c, p, props.OnSelectTab)
			ui.Column(c).FillWidth().Grow(1).Padding(8).Children(func() {
				Pane(c, p)
			})
		})
}

// StatusRow renders the status summary as the pane's top seam.
//
// It is always drawn, even when idle, so the pane does not jump vertically as a
// request starts: an empty line reserves the space. The figures are set in the
// data face with dim "·" separators so the line reads as one data row:
// 200 OK · 412 ms · 2.4 kB, the code carrying the semantic colour and the
// rest staying secondary.
func StatusRow(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	state := p.State()
	status := p.FormattedStatus()

	// separator is dimmed against the figures it divides, so the line's
	// structure comes from rhythm rather than from more ink. It is set in
	// the data face like the figures it divides, and the row's own gap
	// gives it its air — the monospaced face draws its middot high and
	// tight, and a seam that reads "200 OK ·412 ms" is not one row.
	separator := func() {
		theme.DataText(c, "·").FontSize(theme.DataSize).TextColor(tk.TextTertiary)
	}

	return ui.Row(c).
		FillWidth().
		Height(26).
		AlignItems(ui.Center).
		Padding(0, 12).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.Border).
		Children(func() {
			switch state {
			case presenters.ResponseStateLoading:
				// A loading pulse stays in the secondary ink: the pane is
				// working, not warning.
				theme.DataText(c, "Loading…").FontSize(theme.DataSize).TextColor(tk.TextSecondary)

				// A streamed partial size reports progress while the request is
				// still in flight, so a large transfer does not look hung.
				if s := p.FormattedSize(); s != "" {
					separator()
					theme.DataText(c, s).FontSize(theme.DataSize).TextColor(tk.TextTertiary)
				}

			case presenters.ResponseStateError:
				theme.DataText(c, status).FontSize(theme.DataSize).FontWeight(600).TextColor(tk.Error)
				if message := p.ErrorMessage(); message != "" {
					separator()
					theme.DataText(c, message).FontSize(theme.DataSize).TextColor(tk.Error)
				}

			case presenters.ResponseStateCompleted:
				if status != "" {
					theme.DataText(c, status).
						FontSize(theme.DataSize).
						FontWeight(600).
						TextColor(statusColor(tk, p.StatusCodeCategory()))
				}
				if d := p.FormattedDuration(); d != "" {
					separator()
					theme.DataText(c, d).FontSize(theme.DataSize).TextColor(tk.TextSecondary)
				}
				if s := p.FormattedSize(); s != "" {
					separator()
					theme.DataText(c, s).FontSize(theme.DataSize).TextColor(tk.TextSecondary)
				}
				// An error alongside a completed exchange (4xx/5xx, or a failed
				// assertion) belongs on the same line as the status.
				if message := p.ErrorMessage(); message != "" {
					separator()
					theme.DataText(c, message).FontSize(theme.DataSize).TextColor(tk.Error)
				}

			default:
				ui.Text(c, "").FontSize(theme.DataSize)
			}
		})
}

// SubTabs renders the Body/Headers/Timeline/Tests strip as a hairline seam.
//
// It is the world's strip, shared with the sidebar's switcher and the request
// column's sections, so the three speak with one voice and the selected tab
// looks the same everywhere in the window. No tab is a chip: the active one
// is marked by a 2 DIP accent underline that sits on the seam, and the labels
// stay secondary until active. The figure beside a label is set in the data
// face, so the numbers in a strip line up on their digits.
func SubTabs(c *ui.Context, p *presenters.ResponsePresenter, onSelect func(presenters.ResponseSubTab)) ui.Element {
	active := p.ActiveSubTab()

	items := make([]theme.TabItem, 0, len(presenters.AllResponseSubTabs))
	activeIndex := 0
	for i, tab := range presenters.AllResponseSubTabs {
		items = append(items, theme.TabItem{
			Label: tabLabel(tab),
			Count: tabCount(p, tab),
		})
		if tab == active {
			activeIndex = i
		}
	}

	return theme.TabStrip(c, theme.TabStripProps{
		Items:  items,
		Active: activeIndex,
		OnSelect: func(i int) {
			if onSelect != nil {
				onSelect(presenters.AllResponseSubTabs[i])
			}
		},
	})
}

// Pane renders the selected sub-tab's content.
func Pane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	switch p.ActiveSubTab() {
	case presenters.RespSubTabHeaders:
		return HeadersPane(c, p)
	case presenters.RespSubTabTimeline:
		return TimelinePane(c, p)
	case presenters.RespSubTabTests:
		return TestsPane(c, p)
	default:
		return BodyPane(c, p)
	}
}

// BodyPane renders the response body.
//
// The body is code, so it is set in the data face: a JSON payload read in
// monospace keeps its indentation and its column alignment legible in a way
// proportional text cannot.
//
// A spilled body is reported rather than read: the screen has no spooler, and
// showing a large body would defeat the reason it was spilled.
// If the response is a Server-Sent Events stream, it is rendered via SSEStreamPane.
func BodyPane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	if IsEventStream(p) {
		return SSEStreamPane(c, p)
	}

	return ui.Column(c).FillWidth().Children(func() {
		text := p.BodyText()
		if strings.TrimSpace(text) == "" {
			ui.Text(c, emptyBodyMessage(p)).FontSize(theme.DataSize).TextColor(tk.TextSecondary)
			return
		}
		theme.DataText(c, text).FontSize(theme.DataSize).TextColor(tk.Text)
	})
}

// emptyBodyMessage explains an empty pane, distinguishing "nothing yet" from
// "nothing came back".
func emptyBodyMessage(p *presenters.ResponsePresenter) string {
	switch p.State() {
	case presenters.ResponseStateLoading:
		return "Waiting for the response…"
	case presenters.ResponseStateCompleted:
		return "The response body is empty."
	case presenters.ResponseStateError:
		return "No response."
	default:
		return "Send a request to see its response."
	}
}

// HeadersPane lists the response headers, sorted by name.
//
// Sorting is deliberate: a map's iteration order is random, and a header list that
// reshuffles on every frame is unreadable.
func HeadersPane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	result := p.Result()
	if result == nil || len(result.Headers) == 0 {
		return ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, "No response headers.").FontSize(theme.DataSize).TextColor(tk.TextSecondary)
		})
	}

	headers := append(model.ResponseHeaders(nil), result.Headers...)
	sort.SliceStable(headers, func(i, j int) bool {
		return strings.ToLower(headers[i].NameOrEmpty()) < strings.ToLower(headers[j].NameOrEmpty())
	})

	return ui.Column(c).FillWidth().Gap(2).Children(func() {
		for _, h := range headers {
			name := h.NameOrEmpty()
			value := h.ValueOrEmpty()
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				theme.DataText(c, name).FontSize(theme.DataSize).TextColor(tk.TextSecondary).Width(220)
				theme.DataText(c, value).FontSize(theme.DataSize).TextColor(tk.Text)
			})
		}
	})
}

// TimelinePane lists the phase timings.
func TimelinePane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	result := p.Result()
	if result == nil || len(result.Timeline) == 0 {
		return ui.Column(c).FillWidth().Children(func() {
			ui.Text(c, "No timeline recorded.").FontSize(theme.DataSize).TextColor(tk.TextSecondary)
		})
	}

	return ui.Column(c).FillWidth().Gap(2).Children(func() {
		for _, entry := range result.Timeline {
			entry := entry
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				theme.DataText(c, string(entry.Phase)).
					FontSize(theme.DataSize).TextColor(tk.TextSecondary).Width(140)
				theme.DataText(c, formatDuration(entry.Duration)).
					FontSize(theme.DataSize).TextColor(tk.Text).Width(90)
				if entry.Detail != "" {
					theme.DataText(c, entry.Detail).FontSize(theme.DataSize).TextColor(tk.TextTertiary)
				}
			})
		}

		// A total that does not match the sum of the phases is a signal worth
		// showing: it means time passed outside every recorded phase.
		if total := result.Timeline.Total(); total > 0 {
			ui.Row(c).FillWidth().Gap(8).Padding(6, 0, 0, 0).Children(func() {
				theme.DataText(c, "total").FontSize(theme.DataSize).TextColor(tk.TextSecondary).Width(140)
				theme.DataText(c, formatDuration(total)).FontSize(theme.DataSize).TextColor(tk.Text)
			})
		}
	})
}

// TestsPane lists assertion and test results.
//
// The presenter does not yet expose these, so the pane reports that rather than
// rendering an empty list that looks like "everything passed".
func TestsPane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).FillWidth().Children(func() {
		_ = p
		ui.Text(c, "No tests recorded for this request.").FontSize(theme.DataSize).TextColor(tk.TextSecondary)
	})
}

// tabLabel returns the display name of a sub-tab.
func tabLabel(tab presenters.ResponseSubTab) string {
	switch tab {
	case presenters.RespSubTabHeaders:
		return "Headers"
	case presenters.RespSubTabTimeline:
		return "Timeline"
	case presenters.RespSubTabTests:
		return "Tests"
	default:
		return "Body"
	}
}

// tabCount returns a count to append to a tab label, or 0 for none.
func tabCount(p *presenters.ResponsePresenter, tab presenters.ResponseSubTab) int {
	result := p.Result()
	if result == nil {
		return 0
	}
	switch tab {
	case presenters.RespSubTabHeaders:
		return len(result.Headers)
	case presenters.RespSubTabTimeline:
		return len(result.Timeline)
	default:
		return 0
	}
}

// statusColor maps a status category to a theme colour.
func statusColor(tk *lobetheme.Tokens, category string) ui.Color {
	switch category {
	case "success":
		return tk.Success
	case "redirect":
		return tk.Warning
	case "client_error", "server_error":
		return tk.Error
	default:
		return tk.Text
	}
}

// formatDuration renders a phase duration compactly.
func formatDuration(d time.Duration) string {
	us := d.Microseconds()
	switch {
	case us < 1000:
		return fmt.Sprintf("%d µs", us)
	case us < 1_000_000:
		return fmt.Sprintf("%.1f ms", float64(us)/1000)
	default:
		return fmt.Sprintf("%.2f s", float64(us)/1_000_000)
	}
}
