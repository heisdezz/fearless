package openapi

// Spec represents an OpenAPI 3.x or Swagger 2.0 document.
type Spec struct {
	OpenAPI string         `json:"openapi,omitempty" yaml:"openapi,omitempty"`
	Swagger string         `json:"swagger,omitempty" yaml:"swagger,omitempty"`
	Info    InfoObject     `json:"info" yaml:"info"`
	Servers []ServerObject `json:"servers,omitempty" yaml:"servers,omitempty"`

	// Swagger 2.0 fields
	Host                string                          `json:"host,omitempty" yaml:"host,omitempty"`
	BasePath            string                          `json:"basePath,omitempty" yaml:"basePath,omitempty"`
	Schemes             []string                        `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	SecurityDefinitions map[string]SecuritySchemeObject `json:"securityDefinitions,omitempty" yaml:"securityDefinitions,omitempty"`

	Paths      map[string]PathItemObject `json:"paths" yaml:"paths"`
	Components *ComponentsObject         `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []map[string][]string     `json:"security,omitempty" yaml:"security,omitempty"`
}

// InfoObject holds title, description and version.
type InfoObject struct {
	Title          string `json:"title" yaml:"title"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	Version        string `json:"version" yaml:"version"`
	TermsOfService string `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
}

// ServerObject represents a server URL and variables.
type ServerObject struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItemObject represents available operations on a single path.
type PathItemObject struct {
	Summary     string            `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  []ParameterObject `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Get         *OperationObject  `json:"get,omitempty" yaml:"get,omitempty"`
	Post        *OperationObject  `json:"post,omitempty" yaml:"post,omitempty"`
	Put         *OperationObject  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete      *OperationObject  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch       *OperationObject  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options     *OperationObject  `json:"options,omitempty" yaml:"options,omitempty"`
	Head        *OperationObject  `json:"head,omitempty" yaml:"head,omitempty"`
}

// Operations returns all defined HTTP operations and their method names.
func (p PathItemObject) Operations() map[string]*OperationObject {
	ops := make(map[string]*OperationObject)
	if p.Get != nil {
		ops["GET"] = p.Get
	}
	if p.Post != nil {
		ops["POST"] = p.Post
	}
	if p.Put != nil {
		ops["PUT"] = p.Put
	}
	if p.Delete != nil {
		ops["DELETE"] = p.Delete
	}
	if p.Patch != nil {
		ops["PATCH"] = p.Patch
	}
	if p.Options != nil {
		ops["OPTIONS"] = p.Options
	}
	if p.Head != nil {
		ops["HEAD"] = p.Head
	}
	return ops
}

// OperationObject describes a single API operation on a path.
type OperationObject struct {
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []ParameterObject     `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBodyObject    `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// ParameterObject describes a single query, header or path parameter.
type ParameterObject struct {
	Name        string `json:"name" yaml:"name"`
	In          string `json:"in" yaml:"in"` // "query", "header", "path", "formData", "body"
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
	Example     any    `json:"example,omitempty" yaml:"example,omitempty"`
	Schema      any    `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// RequestBodyObject describes an OpenAPI 3.x request body.
type RequestBodyObject struct {
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                       `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaTypeObject `json:"content" yaml:"content"`
}

// MediaTypeObject provides schema and examples for a media type.
type MediaTypeObject struct {
	Schema  any `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
}

// ComponentsObject holds reusable schemas and security schemes.
type ComponentsObject struct {
	SecuritySchemes map[string]SecuritySchemeObject `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Schemas         map[string]any                  `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// SecuritySchemeObject defines a security scheme.
type SecuritySchemeObject struct {
	Type        string `json:"type" yaml:"type"` // "apiKey", "http", "oauth2", "basic"
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	In          string `json:"in,omitempty" yaml:"in,omitempty"`         // "header", "query"
	Scheme      string `json:"scheme,omitempty" yaml:"scheme,omitempty"` // "basic", "bearer"
}
