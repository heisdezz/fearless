package reporter

import (
	"fearless/internal/runner"
)

// Reporter defines the interface for generating execution reports.
type Reporter interface {
	Report(report *runner.RunReport) error
}
