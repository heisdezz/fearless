package model

// Body is the request body union.
//
// Bruno keeps every mode's content in the same object and selects one by Mode,
// so switching modes in the UI does not lose the other modes' text. We keep the
// same shape: all payload fields may be populated, and Mode says which applies.
type Body struct {
	Mode BodyMode `json:"mode" yaml:"mode"`

	JSON   *string `json:"json,omitempty" yaml:"json,omitempty"`
	Text   *string `json:"text,omitempty" yaml:"text,omitempty"`
	XML    *string `json:"xml,omitempty" yaml:"xml,omitempty"`
	Sparql *string `json:"sparql,omitempty" yaml:"sparql,omitempty"`

	FormURLEncoded []KeyValue       `json:"formUrlEncoded,omitempty" yaml:"formUrlEncoded,omitempty"`
	MultipartForm  []MultipartEntry `json:"multipartForm,omitempty" yaml:"multipartForm,omitempty"`
	GraphQL        *GraphQLBody     `json:"graphql,omitempty" yaml:"graphql,omitempty"`
	File           []FileEntry      `json:"file,omitempty" yaml:"file,omitempty"`
}

// GraphQLBody is the GraphQL payload: a query document plus a JSON variables
// string. Variables stay a string because they are edited as text and may be
// mid-edit invalid JSON.
type GraphQLBody struct {
	Query     *string `json:"query,omitempty" yaml:"query,omitempty"`
	Variables *string `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// MultipartEntry is one part of a multipart body.
//
// Value holds the resolved content for a text part, or for a file part the
// `@file(path)` / `@contentType(type)` annotation text before interpolation.
// See docs/data/bru-format.md for the annotation syntax.
type MultipartEntry struct {
	UID         string        `json:"uid" yaml:"uid"`
	Type        MultipartType `json:"type" yaml:"type"`
	Name        *string       `json:"name" yaml:"name"`
	Value       *string       `json:"value" yaml:"value"`
	ContentType *string       `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Description *string       `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	Annotations []Annotation  `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// FileEntry is a selectable file for a `file` body mode. Exactly one entry
// should be Selected; the parser tolerates none, and the builder then sends an
// empty body.
type FileEntry struct {
	UID         string       `json:"uid" yaml:"uid"`
	FilePath    *string      `json:"filePath,omitempty" yaml:"filePath,omitempty"`
	ContentType *string      `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Description *string      `json:"description,omitempty" yaml:"description,omitempty"`
	Selected    bool         `json:"selected" yaml:"selected"`
	Annotations []Annotation `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// IsEmpty reports whether the body would contribute no bytes.
func (b *Body) IsEmpty() bool {
	if b == nil {
		return true
	}
	return b.Mode == BodyNone || b.Mode == ""
}

// SelectedFile returns the chosen file entry, if any.
func (b *Body) SelectedFile() *FileEntry {
	if b == nil {
		return nil
	}
	for i := range b.File {
		if b.File[i].Selected {
			return &b.File[i]
		}
	}
	return nil
}
