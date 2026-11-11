package reporter

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"fearless/internal/runner"
)

// JUnit XML structs per standard schema.
type jUnitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []jUnitTestSuite `xml:"testsuite"`
}

type jUnitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []jUnitTestCase `xml:"testcase"`
}

type jUnitTestCase struct {
	ClassName string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *jUnitFailure `xml:"failure,omitempty"`
}

type jUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitReporter outputs test outcomes formatted as JUnit XML.
type JUnitReporter struct {
	writer   io.Writer
	filePath string
}

// NewJUnitReporter creates a JUnitReporter writing to w.
func NewJUnitReporter(w io.Writer) *JUnitReporter {
	return &JUnitReporter{writer: w}
}

// NewJUnitFileReporter creates a JUnitReporter writing to filePath.
func NewJUnitFileReporter(filePath string) *JUnitReporter {
	return &JUnitReporter{filePath: filePath}
}

// Report serializes the report into JUnit XML.
func (r *JUnitReporter) Report(report *runner.RunReport) error {
	data, err := GenerateJUnit(report)
	if err != nil {
		return err
	}

	if r.filePath != "" {
		if err := os.WriteFile(r.filePath, data, 0644); err != nil {
			return fmt.Errorf("reporter: write junit xml to %s: %w", r.filePath, err)
		}
	}

	if r.writer != nil {
		_, err = r.writer.Write(data)
		return err
	}

	return nil
}

// GenerateJUnit returns JUnit XML bytes for a report.
func GenerateJUnit(report *runner.RunReport) ([]byte, error) {
	if report == nil {
		return []byte(xml.Header + "<testsuites/>"), nil
	}

	ts := jUnitTestSuites{
		Name:     report.CollectionName,
		Tests:    report.Total,
		Failures: report.Failed,
		Errors:   0,
		Time:     fmt.Sprintf("%.3f", report.Duration.Seconds()),
	}

	for _, it := range report.Iterations {
		suite := jUnitTestSuite{
			Name:     fmt.Sprintf("%s - Iteration %d", report.CollectionName, it.Iteration),
			Tests:    it.Total,
			Failures: it.Failed,
			Errors:   0,
			Time:     fmt.Sprintf("%.3f", it.Duration.Seconds()),
		}

		for _, item := range it.Items {
			tc := jUnitTestCase{
				ClassName: fmt.Sprintf("Iteration %d", it.Iteration),
				Name:      fmt.Sprintf("[%s] %s", item.Method, item.ItemName),
				Time:      fmt.Sprintf("%.3f", item.Duration.Seconds()),
			}

			if item.Failed {
				errMsg := item.Error
				if errMsg == "" {
					errMsg = fmt.Sprintf("HTTP status %d", item.Status)
				}
				tc.Failure = &jUnitFailure{
					Message: errMsg,
					Type:    "AssertionFailure",
					Content: errMsg,
				}
			}

			suite.Cases = append(suite.Cases, tc)
		}

		ts.Suites = append(ts.Suites, suite)
	}

	xmlData, err := xml.MarshalIndent(ts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("reporter: marshal junit xml: %w", err)
	}

	return append([]byte(xml.Header), xmlData...), nil
}
