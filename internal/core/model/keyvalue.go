package model

// Annotation is a free-form marker attached to a key/value row by the editor
// (for example the "file" marker on a multipart reference). Bruno persists
// these alongside the row; we preserve them so round-trips are lossless.
type Annotation struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value,omitempty" yaml:"value,omitempty,omitempty"`
}

// KeyValue is the shared row shape for headers, query parameters, path
// parameters, form fields, assertions and variables.
//
// Empty-string and absent are distinguished only where Bruno does; name and
// value are nullable in the source schema, so they use *string.
type KeyValue struct {
	UID         string       `json:"uid" yaml:"uid"`
	Name        *string      `json:"name" yaml:"name"`
	Value       *string      `json:"value" yaml:"value"`
	Description *string      `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool         `json:"enabled" yaml:"enabled"`
	Annotations []Annotation `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// NameOrEmpty returns the row's name, or "" when unset.
func (kv KeyValue) NameOrEmpty() string {
	if kv.Name == nil {
		return ""
	}
	return *kv.Name
}

// ValueOrEmpty returns the row's value, or "" when unset.
func (kv KeyValue) ValueOrEmpty() string {
	if kv.Value == nil {
		return ""
	}
	return *kv.Value
}

// Live reports whether the row participates in a request: enabled and named.
func (kv KeyValue) Live() bool {
	return kv.Enabled && kv.NameOrEmpty() != ""
}

// Param is a query or path parameter.
type Param struct {
	KeyValue
	Type ParamType `json:"type" yaml:"type"`
}

// Variable is a scoped variable definition. Value is any because Bruno allows
// numbers, booleans and objects, not only strings.
type Variable struct {
	UID         string       `json:"uid" yaml:"uid"`
	Name        *string      `json:"name" yaml:"name"`
	Value       any          `json:"value" yaml:"value"`
	Description *string      `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool         `json:"enabled" yaml:"enabled"`
	Local       bool         `json:"local,omitempty" yaml:"local,omitempty"`
	DataType    string       `json:"dataType,omitempty" yaml:"dataType,omitempty"`
	Secret      bool         `json:"secret,omitempty" yaml:"secret,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// NameOrEmpty returns the variable name, or "" when unset.
func (v Variable) NameOrEmpty() string {
	if v.Name == nil {
		return ""
	}
	return *v.Name
}

// ItemSettings are per-request transport overrides.
//
// Every field is optional because "unset" means "inherit the collection or
// global default", which differs from an explicit zero value.
type ItemSettings struct {
	EncodeURL                  *bool    `json:"encodeUrl,omitempty" yaml:"encodeUrl,omitempty"`
	FollowRedirects            *bool    `json:"followRedirects,omitempty" yaml:"followRedirects,omitempty"`
	MaxRedirects               *int     `json:"maxRedirects,omitempty" yaml:"maxRedirects,omitempty"`
	Timeout                    *Timeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ForwardAuthorizationHeader *bool    `json:"forwardAuthorizationHeader,omitempty" yaml:"forwardAuthorizationHeader,omitempty"`
	OmitHeaders                []string `json:"omitHeaders,omitempty" yaml:"omitHeaders,omitempty"`
	DisableParsingResponseJSON *bool    `json:"disableParsingResponseJson,omitempty" yaml:"disableParsingResponseJson,omitempty"`
}

// Defaults applied when the corresponding setting is unset, matching Bruno.
const (
	DefaultFollowRedirects            = true
	DefaultMaxRedirects               = 5
	DefaultForwardAuthorizationHeader = true
	DefaultEncodeURL                  = false
)

// FollowingRedirects resolves FollowRedirects against its default.
func (s ItemSettings) FollowingRedirects() bool {
	if s.FollowRedirects == nil {
		return DefaultFollowRedirects
	}
	return *s.FollowRedirects
}

// MaxRedirectCount resolves MaxRedirects against its default.
func (s ItemSettings) MaxRedirectCount() int {
	if s.MaxRedirects == nil {
		return DefaultMaxRedirects
	}
	return *s.MaxRedirects
}

// ForwardsAuthHeader resolves ForwardAuthorizationHeader against its default.
func (s ItemSettings) ForwardsAuthHeader() bool {
	if s.ForwardAuthorizationHeader == nil {
		return DefaultForwardAuthorizationHeader
	}
	return *s.ForwardAuthorizationHeader
}

// EncodesURL resolves EncodeURL against its default.
func (s ItemSettings) EncodesURL() bool {
	if s.EncodeURL == nil {
		return DefaultEncodeURL
	}
	return *s.EncodeURL
}
