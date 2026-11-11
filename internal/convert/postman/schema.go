package postman

import (
	"encoding/json"
)

// CollectionJSON models Postman v2.0 and v2.1 collection files.
type CollectionJSON struct {
	Info                    InfoJSON       `json:"info"`
	Item                    []ItemJSON     `json:"item"`
	Auth                    *AuthJSON      `json:"auth,omitempty"`
	Event                   []EventJSON    `json:"event,omitempty"`
	Variable                []VariableJSON `json:"variable,omitempty"`
	ProtocolProfileBehavior map[string]any `json:"protocolProfileBehavior,omitempty"`
}

// InfoJSON contains collection metadata.
type InfoJSON struct {
	PostmanID   string `json:"_postman_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
	Version     string `json:"version,omitempty"`
}

// ItemJSON is a node in the Postman collection tree: folder or request.
type ItemJSON struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Description any            `json:"description,omitempty"` // string or DescriptionJSON
	Variable    []VariableJSON `json:"variable,omitempty"`
	Event       []EventJSON    `json:"event,omitempty"`
	Request     *RequestJSON   `json:"request,omitempty"` // If present, it's a request; if nil and Item != nil, it's a folder
	Response    []any          `json:"response,omitempty"`
	Auth        *AuthJSON      `json:"auth,omitempty"`
	Item        []ItemJSON     `json:"item,omitempty"` // Children for folders
}

// DescriptionJSON is the structured description object in Postman schema.
type DescriptionJSON struct {
	Content string `json:"content"`
	Type    string `json:"type,omitempty"`
}

// RequestJSON represents an HTTP or GraphQL request.
type RequestJSON struct {
	URL         any          `json:"url"` // string or URLJSON
	Method      string       `json:"method"`
	Description any          `json:"description,omitempty"`
	Header      []HeaderJSON `json:"header,omitempty"`
	Body        *BodyJSON    `json:"body,omitempty"`
	Auth        *AuthJSON    `json:"auth,omitempty"`
}

// URLJSON represents the structured URL object in Postman v2.
type URLJSON struct {
	Raw      string         `json:"raw,omitempty"`
	Protocol string         `json:"protocol,omitempty"`
	Host     any            `json:"host,omitempty"` // []string or string
	Path     any            `json:"path,omitempty"` // []string or string
	Port     string         `json:"port,omitempty"`
	Query    []QueryJSON    `json:"query,omitempty"`
	Hash     string         `json:"hash,omitempty"`
	Variable []VariableJSON `json:"variable,omitempty"`
}

// HeaderJSON represents a single header entry.
type HeaderJSON struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

// QueryJSON represents a single query parameter entry.
type QueryJSON struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// BodyJSON represents request payload configuration.
type BodyJSON struct {
	Mode       string           `json:"mode,omitempty"` // "raw", "urlencoded", "formdata", "file", "graphql"
	Raw        string           `json:"raw,omitempty"`
	URLEncoded []URLEncodeJSON  `json:"urlencoded,omitempty"`
	FormData   []FormDataJSON   `json:"formdata,omitempty"`
	File       *FileJSON        `json:"file,omitempty"`
	GraphQL    *GraphQLJSON     `json:"graphql,omitempty"`
	Options    *BodyOptionsJSON `json:"options,omitempty"`
}

// BodyOptionsJSON holds format-specific options (e.g. language: "json").
type BodyOptionsJSON struct {
	Raw *RawOptionsJSON `json:"raw,omitempty"`
}

// RawOptionsJSON indicates the raw format language.
type RawOptionsJSON struct {
	Language string `json:"language,omitempty"`
}

// URLEncodeJSON is a form urlencoded key-value row.
type URLEncodeJSON struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// FormDataJSON is a multipart form entry.
type FormDataJSON struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Src         any    `json:"src,omitempty"` // string or []string
	Disabled    bool   `json:"disabled,omitempty"`
	Type        string `json:"type,omitempty"` // "text" or "file"
	ContentType string `json:"contentType,omitempty"`
	Description string `json:"description,omitempty"`
}

// FileJSON is a binary file body.
type FileJSON struct {
	Src     string `json:"src,omitempty"`
	Content string `json:"content,omitempty"`
}

// GraphQLJSON is a GraphQL body.
type GraphQLJSON struct {
	Query     string `json:"query,omitempty"`
	Variables string `json:"variables,omitempty"`
}

// AuthJSON holds Postman authentication settings.
type AuthJSON struct {
	Type   string          `json:"type"` // "basic", "bearer", "apikey", "digest", "ntlm", "oauth2", "awsv4"
	Basic  []AuthParamJSON `json:"basic,omitempty"`
	Bearer []AuthParamJSON `json:"bearer,omitempty"`
	APIKey []AuthParamJSON `json:"apikey,omitempty"`
	Digest []AuthParamJSON `json:"digest,omitempty"`
	NTLM   []AuthParamJSON `json:"ntlm,omitempty"`
	OAuth2 []AuthParamJSON `json:"oauth2,omitempty"`
	AWSV4  []AuthParamJSON `json:"awsv4,omitempty"`
}

// AuthParamJSON is a key/value/type entry inside Postman's auth type arrays.
type AuthParamJSON struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
	Type  string `json:"type,omitempty"`
}

// EventJSON represents a script hook (prerequest or test).
type EventJSON struct {
	Listen string     `json:"listen"` // "prerequest" or "test"
	Script ScriptJSON `json:"script"`
}

// ScriptJSON is the script content.
type ScriptJSON struct {
	Type string `json:"type,omitempty"`
	Exec any    `json:"exec,omitempty"` // string or []string
}

// VariableJSON represents a Postman variable.
type VariableJSON struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// EnvironmentJSON models Postman environment export files.
type EnvironmentJSON struct {
	ID     string               `json:"id,omitempty"`
	Name   string               `json:"name"`
	Values []EnvironmentValJSON `json:"values"`
}

// EnvironmentValJSON is an entry in a Postman environment.
type EnvironmentValJSON struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Enabled bool   `json:"enabled"`
}

func parseDescription(desc any) string {
	if desc == nil {
		return ""
	}
	switch v := desc.(type) {
	case string:
		return v
	case map[string]any:
		if c, ok := v["content"].(string); ok {
			return c
		}
	}
	// Try unmarshal into DescriptionJSON
	b, err := json.Marshal(desc)
	if err == nil {
		var dj DescriptionJSON
		if json.Unmarshal(b, &dj) == nil {
			return dj.Content
		}
	}
	return ""
}
