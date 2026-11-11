package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCSVData(t *testing.T) {
	csvContent := `username,email,role
alice,alice@example.com,admin
bob,bob@example.com,user
charlie,charlie@example.com,guest
`
	rows, err := ParseData(strings.NewReader(csvContent), "csv")
	if err != nil {
		t.Fatalf("parse CSV failed: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0]["username"] != "alice" || rows[0]["role"] != "admin" {
		t.Errorf("row 0 mismatch: %v", rows[0])
	}
	if rows[1]["username"] != "bob" || rows[1]["role"] != "user" {
		t.Errorf("row 1 mismatch: %v", rows[1])
	}
	if rows[2]["username"] != "charlie" || rows[2]["role"] != "guest" {
		t.Errorf("row 2 mismatch: %v", rows[2])
	}
}

func TestParseJSONData(t *testing.T) {
	jsonContent := `[
  {"id": 1, "name": "item 1"},
  {"id": 2, "name": "item 2"}
]`
	rows, err := ParseData(strings.NewReader(jsonContent), "json")
	if err != nil {
		t.Fatalf("parse JSON failed: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "item 1" {
		t.Errorf("row 0 mismatch: %v", rows[0])
	}
	if rows[1]["name"] != "item 2" {
		t.Errorf("row 1 mismatch: %v", rows[1])
	}

	// Single JSON object
	singleJSON := `{"single": true}`
	rows, err = ParseData(strings.NewReader(singleJSON), "json")
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected 1 row for single object, got %d (err: %v)", len(rows), err)
	}
}

func TestLoadDataFile(t *testing.T) {
	tmpDir := t.TempDir()

	csvFile := filepath.Join(tmpDir, "data.csv")
	if err := os.WriteFile(csvFile, []byte("key,val\nk1,v1\nk2,v2\n"), 0644); err != nil {
		t.Fatalf("write csv failed: %v", err)
	}

	df, err := LoadDataFile(csvFile)
	if err != nil {
		t.Fatalf("LoadDataFile CSV failed: %v", err)
	}
	if len(df.Rows) != 2 || df.Rows[0]["key"] != "k1" {
		t.Errorf("loaded CSV rows mismatch: %v", df.Rows)
	}

	jsonFile := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(jsonFile, []byte(`[{"foo": "bar"}]`), 0644); err != nil {
		t.Fatalf("write json failed: %v", err)
	}

	dfJSON, err := LoadDataFile(jsonFile)
	if err != nil {
		t.Fatalf("LoadDataFile JSON failed: %v", err)
	}
	if len(dfJSON.Rows) != 1 || dfJSON.Rows[0]["foo"] != "bar" {
		t.Errorf("loaded JSON rows mismatch: %v", dfJSON.Rows)
	}

	// Empty path
	nilDF, err := LoadDataFile("")
	if err != nil || nilDF != nil {
		t.Errorf("expected nil for empty path: %v, %v", nilDF, err)
	}

	// Nonexistent file
	if _, err := LoadDataFile(filepath.Join(tmpDir, "missing.csv")); err == nil {
		t.Error("expected error for nonexistent data file")
	}
}
