package runner

import (
	"context"
	"fmt"
	"time"

	"fearless/internal/app/commands"
	"fearless/internal/core/model"
)

// Options specifies configuration for an execution run.
type Options struct {
	Collection    *model.Collection
	CollectionUID string
	FolderUID     string
	Env           string
	Recursive     bool
	DelayMs       int
	Bail          bool
	FilterTags    []string
	Iterations    int
	DataFilePath  string
	RuntimeVars   map[string]any
}

// Runner coordinates multi-iteration and parameterized collection runs.
type Runner struct {
	folderRunner commands.FolderRunner
}

// New creates an initialized Runner.
func New(folderRunner commands.FolderRunner) *Runner {
	return &Runner{
		folderRunner: folderRunner,
	}
}

// Run executes a collection or folder according to the specified options.
func (r *Runner) Run(ctx context.Context, opts Options) (*RunReport, error) {
	if r.folderRunner == nil {
		return nil, fmt.Errorf("runner: folder runner is required")
	}

	colUID := opts.CollectionUID
	colName := "Collection"
	if opts.Collection != nil {
		colUID = opts.Collection.UID
		if opts.Collection.Name != "" {
			colName = opts.Collection.Name
		}
	}

	// 1. Load data file if specified
	var dataRows []map[string]any
	if opts.DataFilePath != "" {
		df, err := LoadDataFile(opts.DataFilePath)
		if err != nil {
			return nil, err
		}
		if df != nil {
			dataRows = df.Rows
		}
	}

	// 2. Determine number of iterations
	iterations := opts.Iterations
	if iterations <= 0 {
		if len(dataRows) > 0 {
			iterations = len(dataRows)
		} else {
			iterations = 1
		}
	}

	report := &RunReport{
		RunnerUID:      fmt.Sprintf("run-%d", time.Now().UnixNano()),
		CollectionUID:  colUID,
		CollectionName: colName,
		StartTime:      time.Now(),
	}

	// 3. Execution loop over iterations
	for iter := 1; iter <= iterations; iter++ {
		if ctx.Err() != nil {
			break
		}

		// Compose runtime variables for this iteration
		iterVars := make(map[string]any)
		for k, v := range opts.RuntimeVars {
			iterVars[k] = v
		}

		var rowData map[string]any
		if len(dataRows) > 0 {
			rowData = dataRows[(iter-1)%len(dataRows)]
			for k, v := range rowData {
				iterVars[k] = v
			}
			iterVars["data"] = rowData
		}

		cmd := commands.RunFolderCmd{
			RunnerUID:     fmt.Sprintf("%s-it%d", report.RunnerUID, iter),
			CollectionUID: colUID,
			FolderUID:     opts.FolderUID,
			Env:           opts.Env,
			RuntimeVars:   iterVars,
			Recursive:     opts.Recursive,
			DelayMs:       opts.DelayMs,
			Bail:          opts.Bail,
			FilterTags:    opts.FilterTags,
			Collection:    opts.Collection,
		}

		iterStart := time.Now()
		summary, err := r.folderRunner.RunFolder(ctx, cmd)
		iterDuration := time.Since(iterStart)

		if err != nil && summary == nil {
			return nil, fmt.Errorf("runner: iteration %d failed: %w", iter, err)
		}

		iterReport := &IterationReport{
			Iteration: iter,
			Variables: rowData,
			Duration:  iterDuration,
		}

		if summary != nil {
			iterReport.Total = summary.Total
			iterReport.Passed = summary.Passed
			iterReport.Failed = summary.Failed
			iterReport.Skipped = summary.Skipped

			for _, res := range summary.Results {
				if res == nil {
					continue
				}

				name := res.Sent.URL
				itemUID := ""
				method := res.Sent.Method
				if method == "" {
					method = "GET"
				}

				itemRep := &ItemExecutionReport{
					ItemUID:    itemUID,
					ItemName:   name,
					Method:     method,
					URL:        res.Sent.URL,
					Status:     res.Status,
					StatusText: res.StatusText,
					Duration:   res.Duration,
					Failed:     res.Failed() || res.Status >= 400,
					Error:      res.Err,
				}
				iterReport.Items = append(iterReport.Items, itemRep)
			}
		}

		report.Iterations = append(report.Iterations, iterReport)
		report.Total += iterReport.Total
		report.Passed += iterReport.Passed
		report.Failed += iterReport.Failed
		report.Skipped += iterReport.Skipped

		if opts.Bail && iterReport.Failed > 0 {
			break
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	return report, nil
}
