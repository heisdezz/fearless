package format

import (
	"errors"

	"fearless/internal/core/model"
)

var (
	// ErrUnknownFormat is returned when a requested format has no registered codec.
	ErrUnknownFormat = errors.New("unknown collection format")

	// ErrNoCollection is returned when no collection config is detected in a directory.
	ErrNoCollection = errors.New("no collection found in directory")
)

// Codec represents an on-disk collection format parser and serializer.
//
// Implementations exist for the `.bru` dialect (FORMAT-BRU) and OpenCollection
// YAML dialect (FORMAT-YAML).
type Codec interface {
	// Name returns the format dialect identifier (e.g. FormatBru, FormatYML).
	Name() model.CollectionFormat

	// Detect reports whether the given directory contains a collection in this dialect.
	Detect(dir string) (bool, error)

	// ParseRequest deserializes request file bytes into a model.Request.
	ParseRequest(data []byte) (*model.Request, error)

	// WriteRequest serializes a model.Request into format-specific bytes.
	WriteRequest(req *model.Request) ([]byte, error)

	// ParseCollection deserializes collection configuration file bytes (e.g. bruno.json or opencollection.yml).
	ParseCollection(data []byte) (*model.CollectionConfig, error)

	// WriteCollection serializes collection configuration into format-specific bytes.
	WriteCollection(cfg *model.CollectionConfig) ([]byte, error)

	// ParseFolder deserializes folder configuration file bytes (e.g. folder.bru or folder.yml).
	ParseFolder(data []byte) (*model.Item, error)

	// WriteFolder serializes a folder Item into format-specific bytes.
	WriteFolder(folder *model.Item) ([]byte, error)

	// ParseEnvironment deserializes environment file bytes into a model.Environment.
	ParseEnvironment(data []byte) (*model.Environment, error)

	// WriteEnvironment serializes a model.Environment into format-specific bytes.
	WriteEnvironment(env *model.Environment) ([]byte, error)
}
