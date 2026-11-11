package runner

import (
	"time"

	"fearless/internal/core/model"
)

// ItemExecutionReport holds result details of one request execution in a runner run.
type ItemExecutionReport struct {
	ItemUID     string             `json:"itemUid"`
	ItemName    string             `json:"itemName"`
	Method      string             `json:"method"`
	URL         string             `json:"url"`
	Status      int                `json:"status"`
	StatusText  string             `json:"statusText,omitempty"`
	Duration    time.Duration      `json:"duration"`
	Failed      bool               `json:"failed"`
	Error       string             `json:"error,omitempty"`
	TestResults []model.TestResult `json:"testResults,omitempty"`
}

// IterationReport records results for one run through the request list.
type IterationReport struct {
	Iteration int                    `json:"iteration"`
	Variables map[string]any         `json:"variables,omitempty"`
	Items     []*ItemExecutionReport `json:"items"`
	Total     int                    `json:"total"`
	Passed    int                    `json:"passed"`
	Failed    int                    `json:"failed"`
	Skipped   int                    `json:"skipped"`
	Duration  time.Duration          `json:"duration"`
}

// RunReport contains the complete summary and item outcomes across all iterations.
type RunReport struct {
	RunnerUID      string             `json:"runnerUid"`
	CollectionUID  string             `json:"collectionUid"`
	CollectionName string             `json:"collectionName"`
	Iterations     []*IterationReport `json:"iterations"`
	Total          int                `json:"total"`
	Passed         int                `json:"passed"`
	Failed         int                `json:"failed"`
	Skipped        int                `json:"skipped"`
	Duration       time.Duration      `json:"duration"`
	StartTime      time.Time          `json:"startTime"`
	EndTime        time.Time          `json:"endTime"`
}

// Success reports whether all requests passed without failures.
func (r *RunReport) Success() bool {
	return r != nil && r.Failed == 0
}

// ExitCode returns 0 if all requests and tests passed, 1 otherwise.
func (r *RunReport) ExitCode() int {
	if r.Success() {
		return 0
	}
	return 1
}
