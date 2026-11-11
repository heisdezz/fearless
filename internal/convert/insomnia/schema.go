package insomnia

// ExportJSON models an Insomnia export (v4 and v5).
type ExportJSON struct {
	Type         string         `json:"_type" yaml:"_type"`
	ExportFormat int            `json:"__export_format" yaml:"__export_format"`
	ExportDate   string         `json:"__export_date,omitempty" yaml:"__export_date,omitempty"`
	ExportSource string         `json:"__export_source,omitempty" yaml:"__export_source,omitempty"`
	Resources    []ResourceJSON `json:"resources" yaml:"resources"`
}

// ResourceJSON represents any resource in an Insomnia export.
type ResourceJSON struct {
	ID          string `json:"_id" yaml:"_id"`
	Type        string `json:"_type" yaml:"_type"` // "workspace", "request_group", "request", "environment", "cookie_jar"
	ParentID    string `json:"parentId,omitempty" yaml:"parentId,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Request specific
	URL                    string          `json:"url,omitempty" yaml:"url,omitempty"`
	Method                 string          `json:"method,omitempty" yaml:"method,omitempty"`
	Body                   *BodyJSON       `json:"body,omitempty" yaml:"body,omitempty"`
	Parameters             []ParameterJSON `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Headers                []HeaderJSON    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Authentication         *AuthJSON       `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	SettingFollowRedirects string          `json:"settingFollowRedirects,omitempty" yaml:"settingFollowRedirects,omitempty"` // "global", "on", "off"

	// Environment specific
	Data      map[string]any `json:"data,omitempty" yaml:"data,omitempty"`
	Color     string         `json:"color,omitempty" yaml:"color,omitempty"`
	IsPrivate bool           `json:"isPrivate,omitempty" yaml:"isPrivate,omitempty"`

	// RequestGroup / Folder environment
	Environment map[string]any `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// BodyJSON represents request payload in Insomnia.
type BodyJSON struct {
	MimeType string          `json:"mimeType,omitempty" yaml:"mimeType,omitempty"`
	Text     string          `json:"text,omitempty" yaml:"text,omitempty"`
	FileName string          `json:"fileName,omitempty" yaml:"fileName,omitempty"`
	Params   []BodyParamJSON `json:"params,omitempty" yaml:"params,omitempty"`
}

// BodyParamJSON is a parameter in form or multipart bodies.
type BodyParamJSON struct {
	Name        string `json:"name" yaml:"name"`
	Value       string `json:"value,omitempty" yaml:"value,omitempty"`
	Disabled    bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"` // "text", "file"
	FileName    string `json:"fileName,omitempty" yaml:"fileName,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ParameterJSON represents a query parameter in Insomnia.
type ParameterJSON struct {
	Name        string `json:"name" yaml:"name"`
	Value       string `json:"value" yaml:"value"`
	Disabled    bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// HeaderJSON represents a request header in Insomnia.
type HeaderJSON struct {
	Name        string `json:"name" yaml:"name"`
	Value       string `json:"value" yaml:"value"`
	Disabled    bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// AuthJSON represents authentication configurations in Insomnia.
type AuthJSON struct {
	Type     string `json:"type,omitempty" yaml:"type,omitempty"` // "basic", "bearer", "apikey", "digest", "ntlm", "iam", "oauth2"
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`

	// Basic & Digest & NTLM
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Domain   string `json:"domain,omitempty" yaml:"domain,omitempty"`

	// Bearer
	Token  string `json:"token,omitempty" yaml:"token,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`

	// API Key
	Key   string `json:"key,omitempty" yaml:"key,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	AddTo string `json:"addTo,omitempty" yaml:"addTo,omitempty"` // "header", "queryParams"

	// AWS IAM
	AccessKeyID     string `json:"accessKeyId,omitempty" yaml:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`
	SessionToken    string `json:"sessionToken,omitempty" yaml:"sessionToken,omitempty"`
	Region          string `json:"region,omitempty" yaml:"region,omitempty"`
	Service         string `json:"service,omitempty" yaml:"service,omitempty"`

	// OAuth2
	AccessToken string `json:"accessToken,omitempty" yaml:"accessToken,omitempty"`
}
