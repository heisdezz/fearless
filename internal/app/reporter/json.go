package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"fearless/internal/runner"
)

// JSONReporter serializes a RunReport into indented JSON format.
type JSONReporter struct {
	writer   io.Writer
	filePath string
}

// NewJSONReporter writes JSON to the provided io.Writer.
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{writer: w}
}

// NewJSONFileReporter writes JSON to a file at the given path.
func NewJSONFileReporter(filePath string) *JSONReporter {
	return &JSONReporter{filePath: filePath}
}

// Report outputs the JSON representation of the report.
func (r *JSONReporter) Report(report *runner.RunReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("reporter: marshal json: %w", err)
	}

	if r.filePath != "" {
		if err := os.WriteFile(r.filePath, data, 0644); err != nil {
			return fmt.Errorf("reporter: write json to %s: %w", r.filePath, err)
		}
	}

	if r.writer != nil {
		_, err = r.writer.Write(data)
		return err
	}

	return nil
}

// GenerateJSON returns the report formatted as JSON bytes.
func GenerateJSON(report *runner.RunReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
