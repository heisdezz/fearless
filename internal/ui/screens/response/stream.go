// Package response renders the response viewer, including streaming and SSE views.
package response

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/engine"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/theme"
)

// StreamPaneProps configures the streaming response pane.
type StreamPaneProps struct {
	Messages    []engine.Message
	Progress    engine.Snapshot
	IsStreaming bool
	OnCancel    func()
}

// IsEventStream reports whether the presenter contains a Server-Sent Events stream.
func IsEventStream(p *presenters.ResponsePresenter) bool {
	if p == nil {
		return false
	}
	res := p.Result()
	if res != nil {
		for _, h := range res.Headers {
			if strings.EqualFold(h.NameOrEmpty(), "content-type") &&
				strings.Contains(strings.ToLower(h.ValueOrEmpty()), "text/event-stream") {
				return true
			}
		}
	}
	text := strings.TrimSpace(p.BodyText())
	return strings.HasPrefix(text, "data:") || strings.HasPrefix(text, "event:") || strings.HasPrefix(text, ":")
}

// ParseSSE parses an SSE text payload into discrete engine.Message items.
func ParseSSE(text string) []engine.Message {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var msgs []engine.Message
	h := engine.HandlerFunc(func(e engine.Event) {
		if e.Kind == engine.KindMessage {
			msgs = append(msgs, e.Message)
		}
	})
	parser := engine.NewParser(h, int64(len(text)))
	_ = parser.Write([]byte(text))
	_ = parser.Close()
	return msgs
}

// SSEStreamPane adapts ResponsePresenter to StreamPane.
func SSEStreamPane(c *ui.Context, p *presenters.ResponsePresenter) ui.Element {
	text := p.BodyText()
	msgs := ParseSSE(text)
	snap := engine.Snapshot{
		Received: int64(len(text)),
		Done:     p.State() == presenters.ResponseStateCompleted,
	}
	if p.Result() != nil && p.Result().Body.Size > 0 {
		snap.Expected = p.Result().Body.Size
		if snap.Expected > 0 {
			snap.Fraction = float64(snap.Received) / float64(snap.Expected)
		}
	}
	return StreamPane(c, StreamPaneProps{
		Messages:    msgs,
		Progress:    snap,
		IsStreaming: p.State() == presenters.ResponseStateLoading,
	})
}

// StreamPane renders the incremental stream and Server-Sent Events pane.
func StreamPane(c *ui.Context, props StreamPaneProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		// 1. Stream Header Bar
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Padding(4, 0).Children(func() {
			// Status Tag
			if props.IsStreaming {
				ui.Box(c).Background(tk.Primary).Radius(4).Padding(2, 6).Children(func() {
					ui.Text(c, "STREAMING").FontSize(10).FontWeight(700).TextColor(tk.BgContainer)
				})
			} else {
				ui.Box(c).Background(tk.BgElevated).Border(1, tk.Border).Radius(4).Padding(2, 6).Children(func() {
					ui.Text(c, "STREAM ENDED").FontSize(10).FontWeight(600).TextColor(tk.TextSecondary)
				})
			}

			// Message Count
			theme.DataText(c, fmt.Sprintf("%d events", len(props.Messages))).
				FontSize(theme.DataSize).
				TextColor(tk.Text)

			// Received Bytes
			if props.Progress.Received > 0 {
				theme.DataText(c, formatBytes(props.Progress.Received)).
					FontSize(theme.DataSize).
					TextColor(tk.TextSecondary)
			}

			// Transfer Rate
			if props.Progress.BytesPerSecond > 0 {
				theme.DataText(c, fmt.Sprintf("%.1f KB/s", props.Progress.BytesPerSecond/1024)).
					FontSize(theme.DataSize).
					TextColor(tk.TextTertiary)
			}

			// Spacer
			ui.Box(c).Grow(1)

			// Cancel affordance
			if props.IsStreaming && props.OnCancel != nil {
				ui.Box(c).
					Background(tk.ErrorBg).
					Border(1, tk.Error).
					Radius(4).
					Padding(2, 8).
					Children(func() {
						ui.Text(c, "Cancel Stream").FontSize(11).TextColor(tk.Error)
					}).
					OnClick(props.OnCancel)
			}
		})

		// 2. Progress Bar
		StreamProgressBar(c, props.Progress)

		// 3. Message List
		if len(props.Messages) == 0 {
			ui.Box(c).Padding(16, 0).Children(func() {
				ui.Text(c, "Waiting for stream events…").
					FontSize(theme.DataSize).
					TextColor(tk.TextTertiary)
			})
			return
		}

		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, m := range props.Messages {
				StreamMessageCard(c, m)
			}
		})
	})
}

// StreamProgressBar renders an accurate transfer progress track.
func StreamProgressBar(c *ui.Context, snap engine.Snapshot) ui.Element {
	tk := lobetheme.TokensOf(c)

	fraction := snap.Fraction
	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}

	return ui.Box(c).
		FillWidth().
		Height(4).
		Background(tk.BgElevated).
		Radius(2).
		Children(func() {
			if fraction > 0 {
				pct := float32(fraction * 100)
				ui.Box(c).
					FillHeight().
					Width(pct).
					Background(tk.Primary).
					Radius(2)
			}
		})
}

// StreamMessageCard renders a single SSE discrete message card.
func StreamMessageCard(c *ui.Context, msg engine.Message) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).
		FillWidth().
		Background(tk.BgElevated).
		Border(1, tk.Border).
		Radius(4).
		Padding(6, 8).
		Gap(4).
		Children(func() {
			// Card Header
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				// Sequence #
				theme.DataText(c, fmt.Sprintf("#%d", msg.Seq)).
					FontSize(11).
					TextColor(tk.TextTertiary)

				// Event Name Badge
				evtName := msg.Event
				if evtName == "" {
					evtName = "message"
				}
				ui.Box(c).
					Background(tk.PrimaryBg).
					Radius(3).
					Padding(1, 4).
					Children(func() {
						theme.DataText(c, evtName).
							FontSize(11).
							FontWeight(600).
							TextColor(tk.Primary)
					})

				// ID
				if msg.ID != "" {
					theme.DataText(c, fmt.Sprintf("id: %s", msg.ID)).
						FontSize(11).
						TextColor(tk.TextSecondary)
				}

				// Retry
				if msg.Retry > 0 {
					theme.DataText(c, fmt.Sprintf("retry: %s", msg.Retry)).
						FontSize(11).
						TextColor(tk.TextTertiary)
				}
			})

			// Data Payload
			if msg.Data != "" {
				ui.Box(c).FillWidth().Padding(4, 0).Children(func() {
					theme.DataText(c, msg.Data).
						FontSize(theme.DataSize).
						TextColor(tk.Text)
				})
			}
		})
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), suffix)
}
