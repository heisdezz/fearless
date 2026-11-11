package reporter

import (
	"fmt"
	"io"

	"fearless/internal/runner"
)

// CLIReporter writes concise human-readable output to terminal.
type CLIReporter struct {
	writer io.Writer
}

// NewCLIReporter creates a CLIReporter.
func NewCLIReporter(w io.Writer) *CLIReporter {
	return &CLIReporter{writer: w}
}

// Report outputs a terminal summary of the run.
func (r *CLIReporter) Report(report *runner.RunReport) error {
	if report == nil || r.writer == nil {
		return nil
	}

	fmt.Fprintf(r.writer, "Collection: %s (%s)\n", report.CollectionName, report.Duration)
	fmt.Fprintf(r.writer, "Iterations: %d\n", len(report.Iterations))

	for _, it := range report.Iterations {
		fmt.Fprintf(r.writer, "\nIteration %d:\n", it.Iteration)
		for _, item := range it.Items {
			mark := "✓"
			if item.Failed {
				mark = "✗"
			}
			fmt.Fprintf(r.writer, "  %s [%s] %s (%d, %s)\n",
				mark, item.Method, item.ItemName, item.Status, item.Duration)
			if item.Error != "" {
				fmt.Fprintf(r.writer, "    Error: %s\n", item.Error)
			}
		}
	}

	fmt.Fprintf(r.writer, "\nSummary: %d total, %d passed, %d failed, %d skipped (%s)\n",
		report.Total, report.Passed, report.Failed, report.Skipped, report.Duration)

	return nil
}
