package reporter

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fearless/internal/runner"
)

func buildSampleReport() *runner.RunReport {
	return &runner.RunReport{
		RunnerUID:      "run-sample-123",
		CollectionUID:  "col-1",
		CollectionName: "Petstore API",
		Total:          3,
		Passed:         2,
		Failed:         1,
		Skipped:        0,
		Duration:       150 * time.Millisecond,
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(150 * time.Millisecond),
		Iterations: []*runner.IterationReport{
			{
				Iteration: 1,
				Total:     3,
				Passed:    2,
				Failed:    1,
				Duration:  150 * time.Millisecond,
				Items: []*runner.ItemExecutionReport{
					{
						ItemUID:    "req-1",
						ItemName:   "Get Pets",
						Method:     "GET",
						URL:        "https://api.petstore.local/pets",
						Status:     200,
						StatusText: "OK",
						Duration:   40 * time.Millisecond,
					},
					{
						ItemUID:    "req-2",
						ItemName:   "Create Pet",
						Method:     "POST",
						URL:        "https://api.petstore.local/pets",
						Status:     201,
						StatusText: "Created",
						Duration:   60 * time.Millisecond,
					},
					{
						ItemUID:    "req-3",
						ItemName:   "Delete Pet",
						Method:     "DELETE",
						URL:        "https://api.petstore.local/pets/1",
						Status:     500,
						StatusText: "Internal Server Error",
						Duration:   50 * time.Millisecond,
						Failed:     true,
						Error:      "Internal server error",
					},
				},
			},
		},
	}
}

func TestJSONReporter(t *testing.T) {
	rep := buildSampleReport()
	var buf bytes.Buffer
	reporter := NewJSONReporter(&buf)

	if err := reporter.Report(rep); err != nil {
		t.Fatalf("JSONReporter Report failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal json failed: %v", err)
	}
	if parsed["collectionName"] != "Petstore API" {
		t.Errorf("collectionName mismatch: %v", parsed["collectionName"])
	}

	// File reporter
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.json")
	fileReporter := NewJSONFileReporter(outPath)
	if err := fileReporter.Report(rep); err != nil {
		t.Fatalf("file reporter failed: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file does not exist: %v", err)
	}
}

func TestJUnitReporter(t *testing.T) {
	rep := buildSampleReport()
	var buf bytes.Buffer
	reporter := NewJUnitReporter(&buf)

	if err := reporter.Report(rep); err != nil {
		t.Fatalf("JUnitReporter Report failed: %v", err)
	}

	xmlStr := buf.String()
	if !strings.Contains(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing XML header")
	}
	if !strings.Contains(xmlStr, `<testsuites`) || !strings.Contains(xmlStr, `<testcase`) {
		t.Errorf("missing junit tags: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<failure`) {
		t.Errorf("missing failure element: %s", xmlStr)
	}

	// Valid XML unmarshal check
	var ts jUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &ts); err != nil {
		t.Fatalf("xml unmarshal failed: %v", err)
	}
	if ts.Tests != 3 || ts.Failures != 1 {
		t.Errorf("junit stats mismatch: tests=%d, failures=%d", ts.Tests, ts.Failures)
	}

	// File reporter
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "junit.xml")
	fileReporter := NewJUnitFileReporter(outPath)
	if err := fileReporter.Report(rep); err != nil {
		t.Fatalf("file reporter failed: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file does not exist: %v", err)
	}

	// Nil report
	data, err := GenerateJUnit(nil)
	if err != nil || len(data) == 0 {
		t.Errorf("nil report handling failed: %v", err)
	}
}

func TestHTMLReporter(t *testing.T) {
	rep := buildSampleReport()
	var buf bytes.Buffer
	reporter := NewHTMLReporter(&buf)

	if err := reporter.Report(rep); err != nil {
		t.Fatalf("HTMLReporter Report failed: %v", err)
	}

	htmlStr := buf.String()
	if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE html")
	}
	if !strings.Contains(htmlStr, "Petstore API") {
		t.Error("missing collection name in HTML")
	}
	if !strings.Contains(htmlStr, "badge-GET") || !strings.Contains(htmlStr, "badge-POST") {
		t.Error("missing method badges in HTML")
	}
	if !strings.Contains(htmlStr, "Internal server error") {
		t.Error("missing failure message in HTML")
	}

	// File reporter
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.html")
	fileReporter := NewHTMLFileReporter(outPath)
	if err := fileReporter.Report(rep); err != nil {
		t.Fatalf("file reporter failed: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file does not exist: %v", err)
	}

	// Nil report
	data, err := GenerateHTML(nil)
	if err != nil || !strings.Contains(string(data), "Empty Report") {
		t.Errorf("nil report handling failed: %v", err)
	}
}

func TestCLIReporter(t *testing.T) {
	rep := buildSampleReport()
	var buf bytes.Buffer
	reporter := NewCLIReporter(&buf)

	if err := reporter.Report(rep); err != nil {
		t.Fatalf("CLIReporter Report failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Collection: Petstore API") {
		t.Errorf("missing header in CLI report: %s", output)
	}
	if !strings.Contains(output, "Iteration 1:") {
		t.Errorf("missing iteration in CLI report: %s", output)
	}
	if !strings.Contains(output, "Summary: 3 total, 2 passed, 1 failed, 0 skipped") {
		t.Errorf("missing summary in CLI report: %s", output)
	}

	// Nil report no-op
	if err := reporter.Report(nil); err != nil {
		t.Errorf("nil report error: %v", err)
	}
}
