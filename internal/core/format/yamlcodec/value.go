package yamlcodec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML accepts either a scalar or a `{type, data}` mapping.
//
// Bruno writes plain strings for untyped variables and nests a type for typed
// ones, so the decoder has to accept both shapes for the same field.
func (v *yamlValue) UnmarshalYAML(node *yaml.Node) error {
	v.Raw = nil

	switch node.Kind {
	case yaml.ScalarNode:
		s := node.Value
		v.Scalar = &s
		return nil
	case yaml.MappingNode:
		var typed struct {
			Type string `yaml:"type"`
			Data string `yaml:"data"`
		}
		if err := node.Decode(&typed); err != nil {
			return fmt.Errorf("yamlcodec: typed value: %w", err)
		}
		v.Type = typed.Type
		v.Data = typed.Data
		return nil
	case 0:
		return nil
	default:
		// Keep anything unexpected (a sequence, say) so writing back does not
		// silently drop it.
		var raw any
		if err := node.Decode(&raw); err != nil {
			return err
		}
		v.Raw = raw
		return nil
	}
}

// MarshalYAML writes a scalar for untyped values and a `{type, data}` mapping for
// typed ones, matching the input shape.
func (v yamlValue) MarshalYAML() (any, error) {
	if v.Type != "" {
		return map[string]string{"type": v.Type, "data": v.Data}, nil
	}
	if v.Scalar != nil {
		return *v.Scalar, nil
	}
	if v.Raw != nil {
		return v.Raw, nil
	}
	return "", nil
}

// String returns the value in its plain form, whatever shape it was stored as.
func (v yamlValue) String() string {
	if v.Type != "" {
		return v.Data
	}
	if v.Scalar != nil {
		return *v.Scalar
	}
	return ""
}
