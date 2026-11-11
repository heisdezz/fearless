package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"fearless/internal/core/model"
)

// Importer defines format detection and import into a model.Collection.
type Importer interface {
	Format() string
	Detect(data []byte) bool
	Import(ctx context.Context, data []byte) (*model.Collection, error)
}

// Exporter defines exporting a model.Collection to a specific format.
type Exporter interface {
	Format() string
	Export(ctx context.Context, col *model.Collection) ([]byte, error)
}

// ConverterRegistry manages collection importers and exporters.
type ConverterRegistry struct {
	mu        sync.RWMutex
	importers map[string]Importer
	exporters map[string]Exporter
}

// NewConverterRegistry creates an initialized ConverterRegistry.
func NewConverterRegistry() *ConverterRegistry {
	r := &ConverterRegistry{
		importers: make(map[string]Importer),
		exporters: make(map[string]Exporter),
	}
	r.RegisterImporter(&JSONCollectionConverter{})
	r.RegisterExporter(&JSONCollectionConverter{})
	return r
}

// RegisterImporter registers or replaces an Importer.
func (r *ConverterRegistry) RegisterImporter(importer Importer) {
	if importer == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.importers[strings.ToLower(importer.Format())] = importer
}

// RegisterExporter registers or replaces an Exporter.
func (r *ConverterRegistry) RegisterExporter(exporter Exporter) {
	if exporter == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exporters[strings.ToLower(exporter.Format())] = exporter
}

// GetImporter returns an importer by format name.
func (r *ConverterRegistry) GetImporter(format string) Importer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.importers[strings.ToLower(format)]
}

// GetExporter returns an exporter by format name.
func (r *ConverterRegistry) GetExporter(format string) Exporter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.exporters[strings.ToLower(format)]
}

// DetectImporter scans registered importers to find one that recognizes the data.
func (r *ConverterRegistry) DetectImporter(data []byte) Importer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, imp := range r.importers {
		if imp.Detect(data) {
			return imp
		}
	}
	return nil
}

// AllImporters returns all registered importers.
func (r *ConverterRegistry) AllImporters() []Importer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Importer, 0, len(r.importers))
	for _, imp := range r.importers {
		out = append(out, imp)
	}
	return out
}

// AllExporters returns all registered exporters.
func (r *ConverterRegistry) AllExporters() []Exporter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Exporter, 0, len(r.exporters))
	for _, exp := range r.exporters {
		out = append(out, exp)
	}
	return out
}

// JSONCollectionConverter implements Importer and Exporter for raw collection JSON.
type JSONCollectionConverter struct{}

func (c *JSONCollectionConverter) Format() string { return "json" }

func (c *JSONCollectionConverter) Detect(data []byte) bool {
	var probe struct {
		Name   string `json:"name"`
		Config struct {
			Version string `json:"version"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &probe); err == nil {
		return probe.Name != ""
	}
	return false
}

func (c *JSONCollectionConverter) Import(ctx context.Context, data []byte) (*model.Collection, error) {
	var col model.Collection
	if err := json.Unmarshal(data, &col); err != nil {
		return nil, fmt.Errorf("json converter: import: %w", err)
	}
	if col.UID == "" {
		col.UID = fmt.Sprintf("col-imported-%s", col.Name)
	}
	return &col, nil
}

func (c *JSONCollectionConverter) Export(ctx context.Context, col *model.Collection) ([]byte, error) {
	if col == nil {
		return nil, fmt.Errorf("json converter: nil collection")
	}
	return json.MarshalIndent(col, "", "  ")
}
