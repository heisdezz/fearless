package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fearless/internal/app/commands"
	"fearless/internal/core/model"
)

type fakeFolderRunner struct {
	calls        []commands.RunFolderCmd
	summaryToRet *commands.RunnerSummary
	errToRet     error
}

func (f *fakeFolderRunner) RunFolder(ctx context.Context, cmd commands.RunFolderCmd) (*commands.RunnerSummary, error) {
	f.calls = append(f.calls, cmd)
	if f.errToRet != nil {
		return nil, f.errToRet
	}
	if f.summaryToRet != nil {
		return f.summaryToRet, nil
	}
	// Default fake summary
	return &commands.RunnerSummary{
		RunnerUID:     cmd.RunnerUID,
		CollectionUID: cmd.CollectionUID,
		Total:         2,
		Passed:        2,
		Results: []*model.Result{
			{
				Status:   200,
				Duration: 15 * time.Millisecond,
				Sent:     model.SentRequest{Method: "GET", URL: "https://api.example.com/1"},
			},
			{
				Status:   201,
				Duration: 25 * time.Millisecond,
				Sent:     model.SentRequest{Method: "POST", URL: "https://api.example.com/2"},
			},
		},
	}, nil
}

func (f *fakeFolderRunner) CancelRunner(runnerUID string) bool {
	return true
}

func TestRunner_MultiIteration(t *testing.T) {
	fr := &fakeFolderRunner{}
	r := New(fr)

	col := &model.Collection{
		UID:  "col-1",
		Name: "Test Col",
	}

	opts := Options{
		Collection: col,
		Iterations: 3,
		DelayMs:    5,
		Recursive:  true,
	}

	report, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(fr.calls) != 3 {
		t.Fatalf("expected 3 runner calls for 3 iterations, got %d", len(fr.calls))
	}
	if len(report.Iterations) != 3 {
		t.Fatalf("expected 3 iteration reports, got %d", len(report.Iterations))
	}
	if report.Total != 6 || report.Passed != 6 || report.Failed != 0 {
		t.Errorf("totals mismatch: %+v", report)
	}
	if !report.Success() {
		t.Error("expected Success() == true")
	}
	if report.ExitCode() != 0 {
		t.Errorf("expected exit code 0, got %d", report.ExitCode())
	}
}

func TestRunner_ParameterizedDataFile(t *testing.T) {
	fr := &fakeFolderRunner{}
	r := New(fr)

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "users.csv")
	csvContent := "userId,action\n101,view\n102,update\n"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("write csv failed: %v", err)
	}

	opts := Options{
		CollectionUID: "col-param",
		DataFilePath:  csvPath,
	}

	report, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run with data file failed: %v", err)
	}

	if len(fr.calls) != 2 {
		t.Fatalf("expected 2 iterations matching data file rows, got %d", len(fr.calls))
	}
	if fr.calls[0].RuntimeVars["userId"] != "101" {
		t.Errorf("iteration 1 var mismatch: %v", fr.calls[0].RuntimeVars)
	}
	if fr.calls[1].RuntimeVars["userId"] != "102" {
		t.Errorf("iteration 2 var mismatch: %v", fr.calls[1].RuntimeVars)
	}
	if len(report.Iterations) != 2 {
		t.Fatalf("expected 2 iteration reports, got %d", len(report.Iterations))
	}
}

func TestRunner_BailOnFailure(t *testing.T) {
	fr := &fakeFolderRunner{
		summaryToRet: &commands.RunnerSummary{
			Total:  1,
			Failed: 1,
			Results: []*model.Result{
				{
					Status: 500,
					Err:    "Internal Server Error",
					Sent:   model.SentRequest{Method: "GET", URL: "https://api.example.com/fail"},
				},
			},
		},
	}
	r := New(fr)

	opts := Options{
		CollectionUID: "col-bail",
		Iterations:    5,
		Bail:          true,
	}

	report, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should stop after first iteration because of Bail
	if len(fr.calls) != 1 {
		t.Errorf("expected Bail to stop after 1 call, got %d", len(fr.calls))
	}
	if report.Success() {
		t.Error("expected Success() == false on failure")
	}
	if report.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", report.ExitCode())
	}
}

func TestRunner_ErrorAndCancellation(t *testing.T) {
	// Nil runner error
	nilRunner := New(nil)
	if _, err := nilRunner.Run(context.Background(), Options{}); err == nil {
		t.Error("expected error for nil folderRunner")
	}

	// Folder runner error
	fr := &fakeFolderRunner{errToRet: errors.New("engine dead")}
	r := New(fr)
	if _, err := r.Run(context.Background(), Options{Iterations: 1}); err == nil {
		t.Error("expected error when folderRunner fails")
	}

	// Cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := r.Run(ctx, Options{Iterations: 2})
	if err != nil {
		t.Fatalf("unexpected error on cancelled run: %v", err)
	}
	if len(report.Iterations) != 0 {
		t.Errorf("expected 0 iterations run with cancelled context, got %d", len(report.Iterations))
	}
}
