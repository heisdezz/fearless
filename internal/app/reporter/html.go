package reporter

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"os"

	"fearless/internal/runner"
)

// HTMLReporter generates a standalone, responsive HTML test report.
type HTMLReporter struct {
	writer   io.Writer
	filePath string
}

// NewHTMLReporter creates an HTMLReporter writing to w.
func NewHTMLReporter(w io.Writer) *HTMLReporter {
	return &HTMLReporter{writer: w}
}

// NewHTMLFileReporter creates an HTMLReporter writing to filePath.
func NewHTMLFileReporter(filePath string) *HTMLReporter {
	return &HTMLReporter{filePath: filePath}
}

// Report outputs the HTML formatted report.
func (r *HTMLReporter) Report(report *runner.RunReport) error {
	data, err := GenerateHTML(report)
	if err != nil {
		return err
	}

	if r.filePath != "" {
		if err := os.WriteFile(r.filePath, data, 0644); err != nil {
			return fmt.Errorf("reporter: write html to %s: %w", r.filePath, err)
		}
	}

	if r.writer != nil {
		_, err = r.writer.Write(data)
		return err
	}

	return nil
}

// GenerateHTML renders the report into an HTML document.
func GenerateHTML(report *runner.RunReport) ([]byte, error) {
	if report == nil {
		return []byte("<!DOCTYPE html><html><body><h1>Empty Report</h1></body></html>"), nil
	}

	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	buf.WriteString("<meta charset=\"UTF-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	buf.WriteString(fmt.Sprintf("<title>%s - Test Report</title>\n", html.EscapeString(report.CollectionName)))
	buf.WriteString("<style>\n")
	buf.WriteString(`
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #f8fafc; margin: 0; padding: 24px; }
.container { max-width: 960px; margin: 0 auto; }
h1 { font-size: 24px; margin-bottom: 8px; }
.meta { color: #94a3b8; font-size: 14px; margin-bottom: 24px; }
.cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
.card { background: #1e293b; border-radius: 8px; padding: 16px; text-align: center; border: 1px solid #334155; }
.card .num { font-size: 28px; font-weight: bold; margin-bottom: 4px; }
.card .lbl { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: #94a3b8; }
.passed { color: #4ade80; }
.failed { color: #f87171; }
.skipped { color: #fbbf24; }
.total { color: #60a5fa; }
.iter-box { background: #1e293b; border: 1px solid #334155; border-radius: 8px; margin-bottom: 16px; overflow: hidden; }
.iter-header { background: #334155; padding: 12px 16px; font-weight: 600; font-size: 15px; display: flex; justify-content: space-between; }
.item-row { padding: 12px 16px; border-bottom: 1px solid #334155; display: flex; align-items: center; justify-content: space-between; font-size: 14px; }
.item-row:last-child { border-bottom: none; }
.badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 600; text-transform: uppercase; margin-right: 8px; }
.badge-GET { background: #0284c7; color: white; }
.badge-POST { background: #16a34a; color: white; }
.badge-PUT { background: #d97706; color: white; }
.badge-DELETE { background: #dc2626; color: white; }
.badge-DEFAULT { background: #64748b; color: white; }
.err-box { margin-top: 8px; padding: 8px 12px; background: #451a1a; border: 1px solid #7f1d1d; border-radius: 4px; color: #fca5a5; font-family: monospace; font-size: 12px; }
`)
	buf.WriteString("</style>\n</head>\n<body>\n<div class=\"container\">\n")

	// Header
	buf.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(report.CollectionName)))
	buf.WriteString(fmt.Sprintf("<div class=\"meta\">Run: %s &bull; Duration: %s</div>\n",
		html.EscapeString(report.RunnerUID), report.Duration.String()))

	// Stat Cards
	buf.WriteString("<div class=\"cards\">\n")
	buf.WriteString(fmt.Sprintf("<div class=\"card\"><div class=\"num total\">%d</div><div class=\"lbl\">Total</div></div>\n", report.Total))
	buf.WriteString(fmt.Sprintf("<div class=\"card\"><div class=\"num passed\">%d</div><div class=\"lbl\">Passed</div></div>\n", report.Passed))
	buf.WriteString(fmt.Sprintf("<div class=\"card\"><div class=\"num failed\">%d</div><div class=\"lbl\">Failed</div></div>\n", report.Failed))
	buf.WriteString(fmt.Sprintf("<div class=\"card\"><div class=\"num skipped\">%d</div><div class=\"lbl\">Skipped</div></div>\n", report.Skipped))
	buf.WriteString("</div>\n")

	// Iterations
	for _, it := range report.Iterations {
		buf.WriteString("<div class=\"iter-box\">\n")
		buf.WriteString(fmt.Sprintf("<div class=\"iter-header\"><span>Iteration %d</span><span>%s</span></div>\n",
			it.Iteration, it.Duration.String()))

		for _, item := range it.Items {
			badgeClass := "badge-DEFAULT"
			switch item.Method {
			case "GET", "POST", "PUT", "DELETE":
				badgeClass = "badge-" + item.Method
			}

			statusStr := fmt.Sprintf("%d", item.Status)
			if item.StatusText != "" {
				statusStr += " " + item.StatusText
			}

			statusClass := "passed"
			if item.Failed {
				statusClass = "failed"
			}

			buf.WriteString("<div class=\"item-row\">\n")
			buf.WriteString("  <div>\n")
			buf.WriteString(fmt.Sprintf("    <span class=\"badge %s\">%s</span><strong>%s</strong>\n",
				badgeClass, html.EscapeString(item.Method), html.EscapeString(item.ItemName)))
			if item.Error != "" {
				buf.WriteString(fmt.Sprintf("    <div class=\"err-box\">%s</div>\n", html.EscapeString(item.Error)))
			}
			buf.WriteString("  </div>\n")
			buf.WriteString(fmt.Sprintf("  <div><span class=\"%s\">%s</span> &bull; <span style=\"color:#94a3b8;\">%s</span></div>\n",
				statusClass, html.EscapeString(statusStr), item.Duration.String()))
			buf.WriteString("</div>\n")
		}
		buf.WriteString("</div>\n")
	}

	buf.WriteString("</div>\n</body>\n</html>\n")
	return buf.Bytes(), nil
}
