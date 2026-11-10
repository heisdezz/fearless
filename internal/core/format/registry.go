package format

import (
	"fmt"
	"sort"
	"sync"

	"fearless/internal/core/model"
)

var (
	registryMu sync.RWMutex
	codecs     = make(map[model.CollectionFormat]Codec)
)

// Register adds a codec to the format registry.
// It panics if codec is nil or if a codec for the given format is already registered.
func Register(c Codec) {
	if c == nil {
		panic("format: cannot register nil Codec")
	}
	name := c.Name()
	if !name.Valid() && name != "" {
		// allow custom formats if needed, but validate non-empty
	}
	if name == "" {
		panic("format: cannot register Codec with empty name")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := codecs[name]; exists {
		panic(fmt.Sprintf("format: codec for %q already registered", name))
	}
	codecs[name] = c
}

// Get returns the codec registered for format, or (nil, false) if not found.
func Get(format model.CollectionFormat) (Codec, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	c, ok := codecs[format]
	return c, ok
}

// For returns the codec for format or ErrUnknownFormat.
func For(format model.CollectionFormat) (Codec, error) {
	c, ok := Get(format)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, format)
	}
	return c, nil
}

// Registered returns all registered format names in sorted order.
func Registered() []model.CollectionFormat {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]model.CollectionFormat, 0, len(codecs))
	for name := range codecs {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

// Reset clears the codec registry. Used primarily for unit tests.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	codecs = make(map[model.CollectionFormat]Codec)
}
