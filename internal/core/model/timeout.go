package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Timeout is Bruno's timeout setting: either a number of milliseconds or the
// literal string "inherit", which defers to the global preference.
//
// It is modelled as a union rather than *int because the "inherit" case is not
// a number at all, and collapsing it to 0 would make it indistinguishable from
// "no timeout".
type Timeout struct {
	// Millis is the timeout in milliseconds. Meaningful only when Inherit is false.
	Millis int `json:"-"`
	// Inherit defers to the global request timeout preference.
	Inherit bool `json:"-"`
}

// InheritTimeout is the timeout value meaning "use the global preference".
var InheritTimeout = Timeout{Inherit: true}

// TimeoutMillis returns a fixed timeout in milliseconds.
func TimeoutMillis(ms int) Timeout { return Timeout{Millis: ms} }

// IsZero reports whether the value is unset.
func (t Timeout) IsZero() bool { return !t.Inherit && t.Millis == 0 }

// String renders the value as it appears on disk.
func (t Timeout) String() string {
	if t.Inherit {
		return "inherit"
	}
	return fmt.Sprintf("%d", t.Millis)
}

// MarshalJSON encodes the union in Bruno's representation.
func (t Timeout) MarshalJSON() ([]byte, error) {
	if t.Inherit {
		return json.Marshal("inherit")
	}
	return json.Marshal(t.Millis)
}

// UnmarshalJSON accepts either a number or the string "inherit".
func (t *Timeout) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = Timeout{}
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s != "inherit" {
			return fmt.Errorf("model: invalid timeout value %q", s)
		}
		*t = Timeout{Inherit: true}
		return nil
	}
	var ms int
	if err := json.Unmarshal(data, &ms); err != nil {
		return fmt.Errorf("model: timeout must be a number or \"inherit\": %w", err)
	}
	*t = Timeout{Millis: ms}
	return nil
}

// MarshalYAML renders the value for the OpenCollection dialect.
func (t Timeout) MarshalYAML() (any, error) {
	if t.Inherit {
		return "inherit", nil
	}
	return t.Millis, nil
}

// UnmarshalYAML accepts an int or the string "inherit".
func (t *Timeout) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		if s != "inherit" {
			return fmt.Errorf("model: invalid timeout value %q", s)
		}
		*t = Timeout{Inherit: true}
		return nil
	}
	var ms int
	if err := unmarshal(&ms); err != nil {
		return fmt.Errorf("model: timeout must be a number or \"inherit\": %w", err)
	}
	*t = Timeout{Millis: ms}
	return nil
}
