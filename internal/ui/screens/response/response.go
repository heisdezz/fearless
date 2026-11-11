// Package response renders the response viewer.
//
// The screen is a pure function of a presenter snapshot: it reads
// ResponsePresenter, draws it, and reports clicks through callbacks. It never
// mutates domain state inside a View closure, which is the acceptance criterion
// for every UI-SCREEN-* package (docs/progress/packages.md L237).
//
// Layout, top to bottom:
//
//	status row   200 OK · 45 ms · 512 B · (error text when failed)
//	sub-tabs     Body | Headers | Timeline | Tests
//	pane         the selected view
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
		return ui.Box(c).FillWidth().Height(DefaultMinHeight).Background(tk.BgContainer)
	}

	minHeight := props.MinHeight
	if minHeight <= 0 {
		minHeight = DefaultMinHeight
	}

	return ui.Column(c).
		FillWidth().
		Height(minHeight).
		Background(tk.BgContainer).
		Children(func() {
			StatusRow(c, p)
			SubTabs(c, p, props.OnSelectTab)
			ui.Column(c).FillWidth().Padding(8).Children(func() {
				Pane(c, p)
			})
		})
}

// StatusRow renders the status summary line.
//
// It is always drawn, even when idle, so the pane does not jump vertically as a
// request starts: an empty line reserves the space.
func StatusRow(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	state := p.State()
	status := p.FormattedStatus()

	return ui.Row(c).
		FillWidth().
		Height(32).
		AlignItems(ui.Center).
		Gap(12).
		Padding(0, 10).
		Border(1, tk.BorderSecondary).
		Children(func() {
			switch state {
			case presenters.ResponseStateLoading:
				// The size grows as chunks arrive, so it is the honest indicator
				// that something is still streaming.
				ui.Text(c, "Sending…").FontSize(12).TextColor(tk.TextSecondary)
				if size := p.FormattedSize(); size != "" {
					ui.Text(c, size).FontSize(12).TextColor(tk.TextTertiary)
				}

			case presenters.ResponseStateError:
				ui.Text(c, "Error").FontSize(12).TextColor(tk.Error).FontWeight(600)
				if message := p.ErrorMessage(); message != "" {
					ui.Text(c, message).FontSize(12).TextColor(tk.Error)
				}

			case presenters.ResponseStateCompleted:
				if status != "" {
					ui.Text(c, status).
						FontSize(12).
						FontWeight(600).
						TextColor(statusColor(tk, p.StatusCodeCategory()))
				}
				if d := p.FormattedDuration(); d != "" {
					ui.Text(c, d).FontSize(12).TextColor(tk.TextSecondary)
				}
				if s := p.FormattedSize(); s != "" {
					ui.Text(c, s).FontSize(12).TextColor(tk.TextSecondary)
				}
				// An error alongside a completed exchange (4xx/5xx, or a failed
				// assertion) belongs on the same line as the status.
				if message := p.ErrorMessage(); message != "" {
					ui.Text(c, message).FontSize(12).TextColor(tk.Error)
				}

			default:
				ui.Text(c, "").FontSize(12)
			}
		})
}

// SubTabs renders the Body/Headers/Timeline/Tests strip.
func SubTabs(c *ui.Context, p *presenters.ResponsePresenter, onSelect func(presenters.ResponseSubTab)) ui.Element {
	tk := lobetheme.TokensOf(c)

	active := p.ActiveSubTab()

	return ui.Row(c).
		FillWidth().
		Height(32).
		AlignItems(ui.Center).
		Gap(4).
		Padding(0, 8).
		Children(func() {
			for _, tab := range presenters.AllResponseSubTabs {
				tab := tab
				isActive := tab == active

				label := tabLabel(tab)
				if count := tabCount(p, tab); count > 0 {
					label = fmt.Sprintf("%s (%d)", label, count)
				}

				color := tk.TextSecondary
				bg := tk.BgContainer
				if isActive {
					color = tk.Text
					bg = tk.BgElevated
				}

				ui.Row(c).
					AlignItems(ui.Center).
					Padding(4, 10).
					Background(bg).
					Border(1, borderFor(tk, isActive)).
					Children(func() {
						ui.Text(c, label).FontSize(12).TextColor(color)
					}).
					OnClick(func() {
						if onSelect != nil {
							onSelect(tab)
						}
					})
			}
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
// A spilled body is reported rather than read: the screen has no spooler, and
// showing a large body would defeat the reason it was spilled.
func BodyPane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).FillWidth().Children(func() {
		text := p.BodyText()
		if strings.TrimSpace(text) == "" {
			ui.Text(c, emptyBodyMessage(p)).FontSize(12).TextColor(tk.TextTertiary)
			return
		}
		ui.Text(c, text).FontSize(12).TextColor(tk.Text)
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
			ui.Text(c, "No response headers.").FontSize(12).TextColor(tk.TextTertiary)
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
				ui.Text(c, name).FontSize(12).TextColor(tk.TextSecondary).Width(220)
				ui.Text(c, value).FontSize(12).TextColor(tk.Text)
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
			ui.Text(c, "No timeline recorded.").FontSize(12).TextColor(tk.TextTertiary)
		})
	}

	return ui.Column(c).FillWidth().Gap(2).Children(func() {
		for _, entry := range result.Timeline {
			entry := entry
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, string(entry.Phase)).
					FontSize(12).TextColor(tk.TextSecondary).Width(140)
				ui.Text(c, formatDuration(entry.Duration)).
					FontSize(12).TextColor(tk.Text).Width(90)
				if entry.Detail != "" {
					ui.Text(c, entry.Detail).FontSize(12).TextColor(tk.TextTertiary)
				}
			})
		}

		// A total that does not match the sum of the phases is a signal worth
		// showing: it means time passed outside every recorded phase.
		if total := result.Timeline.Total(); total > 0 {
			ui.Row(c).FillWidth().Gap(8).Padding(6, 0, 0, 0).Children(func() {
				ui.Text(c, "total").FontSize(12).TextColor(tk.TextSecondary).Width(140)
				ui.Text(c, formatDuration(total)).FontSize(12).TextColor(tk.Text)
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
		ui.Text(c, "No tests recorded for this request.").FontSize(12).TextColor(tk.TextTertiary)
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

// borderFor returns the border colour for a tab.
func borderFor(tk *lobetheme.Tokens, active bool) ui.Color {
	if active {
		return tk.Primary
	}
	return tk.BorderSecondary
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
