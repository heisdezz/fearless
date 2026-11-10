// Package yamlcodec implements OpenCollection: Bruno's YAML collection dialect.
//
// The dialect keeps the same information as `.bru` but in a nested document
// rather than flat blocks. A request file looks like:
//
//	info:
//	  name: request_1
//	  type: http
//	  seq: 1
//	http:
//	  method: GET
//	  url: https://api.example.com/one
//
// Both dialects map onto the same `model` types, so the rest of the application
// never needs to know which one a collection uses. See
// docs/data/collection-format.md for the reference.
//
// The package is named yamlcodec because `yaml` would shadow the import of
// gopkg.in/yaml.v3 throughout the file.
package yamlcodec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fearless/internal/core/format"
	"fearless/internal/core/model"

	"gopkg.in/yaml.v3"
)

// Codec implements format.Codec for OpenCollection YAML.
type Codec struct{}

// New returns the OpenCollection codec.
func New() *Codec { return &Codec{} }

var _ format.Codec = (*Codec)(nil)

// collectionConfigFile is the marker identifying a YAML collection.
const collectionConfigFile = "opencollection.yml"

// Name reports the dialect.
func (c *Codec) Name() model.CollectionFormat { return model.FormatYML }

// Detect reports whether dir holds an OpenCollection collection.
func (c *Codec) Detect(dir string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, collectionConfigFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// Register installs the codec. Called from an explicit wiring point rather than
// init(), keeping the dependency graph predictable.
func Register() { format.Register(New()) }

// decode unmarshals YAML into v with a friendlier error message.
func decode(data []byte, v any) error {
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("yamlcodec: %w", err)
	}
	return nil
}

// encode marshals v with two-space indentation, matching Bruno's output.
func encode(v any) ([]byte, error) {
	var sb writeBuffer
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("yamlcodec: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("yamlcodec: %w", err)
	}
	return sb.b, nil
}

// writeBuffer collects encoder output.
type writeBuffer struct{ b []byte }

func (w *writeBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
