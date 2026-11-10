// Package model holds the domain types of the application.
//
// These are pure data: no I/O, no policy, and no imports from any other
// fearless package. Every other package depends on this one, so it must stay
// acyclic and dependency-free.
//
// Field names and optionality mirror Bruno's schema
// (packages/bruno-schema-types). See docs/data/request-schema.md.
package model

// CollectionFormat identifies the on-disk dialect of a collection.
type CollectionFormat string

const (
	// FormatBru is the `.bru` DSL (bruno.json + *.bru).
	FormatBru CollectionFormat = "bru"
	// FormatYML is OpenCollection (opencollection.yml + *.yml).
	FormatYML CollectionFormat = "yml"
)

// Valid reports whether f is a known dialect.
func (f CollectionFormat) Valid() bool {
	return f == FormatBru || f == FormatYML
}

// ItemType is the kind of entry in a collection tree.
type ItemType string

const (
	ItemHTTPRequest    ItemType = "http-request"
	ItemGraphQLRequest ItemType = "graphql-request"
	ItemFolder         ItemType = "folder"
	ItemGRPCRequest    ItemType = "grpc-request"
	ItemWSRequest      ItemType = "ws-request"
	ItemJS             ItemType = "js"
	ItemApp            ItemType = "app"
)

// IsRequest reports whether t can be sent over the network.
func (t ItemType) IsRequest() bool {
	switch t {
	case ItemHTTPRequest, ItemGraphQLRequest, ItemGRPCRequest, ItemWSRequest:
		return true
	}
	return false
}

// BodyMode is the request body discriminator.
type BodyMode string

const (
	BodyNone           BodyMode = "none"
	BodyJSON           BodyMode = "json"
	BodyText           BodyMode = "text"
	BodyXML            BodyMode = "xml"
	BodySparql         BodyMode = "sparql"
	BodyFormURLEncoded BodyMode = "formUrlEncoded"
	BodyMultipartForm  BodyMode = "multipartForm"
	BodyGraphQL        BodyMode = "graphql"
	BodyFile           BodyMode = "file"
)

// Valid reports whether m is a known body mode.
func (m BodyMode) Valid() bool {
	switch m {
	case BodyNone, BodyJSON, BodyText, BodyXML, BodySparql,
		BodyFormURLEncoded, BodyMultipartForm, BodyGraphQL, BodyFile:
		return true
	}
	return false
}

// ParamType distinguishes query parameters from path parameters.
type ParamType string

const (
	ParamQuery ParamType = "query"
	ParamPath  ParamType = "path"
)

// MultipartType is the kind of a multipart part.
type MultipartType string

const (
	MultipartText MultipartType = "text"
	MultipartFile MultipartType = "file"
)
