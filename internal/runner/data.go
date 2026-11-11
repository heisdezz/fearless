package runner

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DataFile represents loaded iteration dataset from a CSV or JSON file.
type DataFile struct {
	Path string
	Rows []map[string]any
}

// LoadDataFile loads records from a CSV or JSON file.
func LoadDataFile(path string) (*DataFile, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("runner: read data file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return parseCSVData(path, string(data))
	case ".json":
		return parseJSONData(path, data)
	default:
		// Attempt JSON first, then CSV
		if df, errJSON := parseJSONData(path, data); errJSON == nil {
			return df, nil
		}
		return parseCSVData(path, string(data))
	}
}

// ParseData parses records from a reader given format ("csv" or "json").
func ParseData(r io.Reader, format string) ([]map[string]any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "csv":
		df, err := parseCSVData("", string(data))
		if err != nil {
			return nil, err
		}
		return df.Rows, nil
	case "json":
		df, err := parseJSONData("", data)
		if err != nil {
			return nil, err
		}
		return df.Rows, nil
	default:
		if df, errJSON := parseJSONData("", data); errJSON == nil {
			return df.Rows, nil
		}
		df, errCSV := parseCSVData("", string(data))
		if errCSV != nil {
			return nil, fmt.Errorf("runner: unrecognized data format %q", format)
		}
		return df.Rows, nil
	}
}

func parseCSVData(path, content string) (*DataFile, error) {
	r := csv.NewReader(strings.NewReader(content))
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("runner: parse csv: %w", err)
	}
	if len(records) == 0 {
		return &DataFile{Path: path, Rows: nil}, nil
	}

	headers := records[0]
	var rows []map[string]any
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		record := records[rowIdx]
		if len(record) == 0 || (len(record) == 1 && record[0] == "") {
			continue // skip empty rows
		}
		rowMap := make(map[string]any, len(headers))
		for colIdx, h := range headers {
			key := strings.TrimSpace(h)
			if colIdx < len(record) {
				rowMap[key] = record[colIdx]
			} else {
				rowMap[key] = ""
			}
		}
		rows = append(rows, rowMap)
	}

	return &DataFile{Path: path, Rows: rows}, nil
}

func parseJSONData(path string, data []byte) (*DataFile, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("runner: parse json: %w", err)
	}

	var rows []map[string]any
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if obj, ok := item.(map[string]any); ok {
				rows = append(rows, obj)
			}
		}
	case map[string]any:
		rows = append(rows, v)
	default:
		return nil, fmt.Errorf("runner: json data must be array of objects or single object")
	}

	return &DataFile{Path: path, Rows: rows}, nil
}
