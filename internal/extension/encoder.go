package extension

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// BodyEncoder defines encoding and decoding for a specific payload format.
type BodyEncoder interface {
	Name() string
	ContentType() string
	Encode(v any) ([]byte, error)
	Decode(data []byte, target any) error
}

// BodyEncoderRegistry manages registered body encoders.
type BodyEncoderRegistry struct {
	mu       sync.RWMutex
	encoders map[string]BodyEncoder
}

// NewBodyEncoderRegistry creates an initialized BodyEncoderRegistry with default encoders registered.
func NewBodyEncoderRegistry() *BodyEncoderRegistry {
	r := &BodyEncoderRegistry{
		encoders: make(map[string]BodyEncoder),
	}
	r.Register(&JSONBodyEncoder{})
	r.Register(&FormURLEncodedBodyEncoder{})
	r.Register(&XMLBodyEncoder{})
	return r
}

// Register registers or replaces a BodyEncoder.
func (r *BodyEncoderRegistry) Register(encoder BodyEncoder) {
	if encoder == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.encoders[strings.ToLower(encoder.Name())] = encoder
}

// Get returns the encoder by name.
func (r *BodyEncoderRegistry) Get(name string) BodyEncoder {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.encoders[strings.ToLower(name)]
}

// ForContentType returns an encoder matching the media type in the Content-Type header.
func (r *BodyEncoderRegistry) ForContentType(contentType string) BodyEncoder {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ct := strings.ToLower(contentType)
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}

	for _, enc := range r.encoders {
		encCT := strings.ToLower(enc.ContentType())
		if idx := strings.Index(encCT, ";"); idx >= 0 {
			encCT = strings.TrimSpace(encCT[:idx])
		}
		if ct == encCT {
			return enc
		}
	}
	return nil
}

// All returns all registered encoders.
func (r *BodyEncoderRegistry) All() []BodyEncoder {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]BodyEncoder, 0, len(r.encoders))
	for _, enc := range r.encoders {
		out = append(out, enc)
	}
	return out
}

// JSONBodyEncoder implements BodyEncoder for JSON.
type JSONBodyEncoder struct{}

func (e *JSONBodyEncoder) Name() string        { return "json" }
func (e *JSONBodyEncoder) ContentType() string { return "application/json" }

func (e *JSONBodyEncoder) Encode(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	if s, ok := v.(string); ok {
		return []byte(s), nil
	}
	return json.Marshal(v)
}

func (e *JSONBodyEncoder) Decode(data []byte, target any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

// FormURLEncodedBodyEncoder implements BodyEncoder for form-urlencoded.
type FormURLEncodedBodyEncoder struct{}

func (e *FormURLEncodedBodyEncoder) Name() string        { return "form-urlencoded" }
func (e *FormURLEncodedBodyEncoder) ContentType() string { return "application/x-www-form-urlencoded" }

func (e *FormURLEncodedBodyEncoder) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case string:
		return []byte(val), nil
	case []byte:
		return val, nil
	case map[string]string:
		values := url.Values{}
		for k, vStr := range val {
			values.Set(k, vStr)
		}
		return []byte(values.Encode()), nil
	case map[string]any:
		values := url.Values{}
		for k, vAny := range val {
			values.Set(k, fmt.Sprintf("%v", vAny))
		}
		return []byte(values.Encode()), nil
	case url.Values:
		return []byte(val.Encode()), nil
	default:
		return nil, fmt.Errorf("form-urlencoded: unsupported type %T", v)
	}
}

func (e *FormURLEncodedBodyEncoder) Decode(data []byte, target any) error {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	switch ptr := target.(type) {
	case *map[string]string:
		m := make(map[string]string, len(values))
		for k := range values {
			m[k] = values.Get(k)
		}
		*ptr = m
		return nil
	case *map[string]any:
		m := make(map[string]any, len(values))
		for k := range values {
			m[k] = values.Get(k)
		}
		*ptr = m
		return nil
	case *url.Values:
		*ptr = values
		return nil
	default:
		return fmt.Errorf("form-urlencoded: target must be *map[string]string or *url.Values")
	}
}

// XMLBodyEncoder implements BodyEncoder for XML.
type XMLBodyEncoder struct{}

func (e *XMLBodyEncoder) Name() string        { return "xml" }
func (e *XMLBodyEncoder) ContentType() string { return "application/xml" }

func (e *XMLBodyEncoder) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	if s, ok := v.(string); ok {
		return []byte(s), nil
	}
	return xml.Marshal(v)
}

func (e *XMLBodyEncoder) Decode(data []byte, target any) error {
	if len(data) == 0 {
		return nil
	}
	return xml.Unmarshal(data, target)
}
