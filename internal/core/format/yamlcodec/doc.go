package yamlcodec

import "gopkg.in/yaml.v3"

// The structs here mirror the OpenCollection YAML schema exactly as Bruno writes
// it. They are deliberately separate from `model`: the model is the application's
// canonical shape, while these track the file format (including its inconsistent
// field names, such as `mode:` for some bodies and `type:` for others).

// collectionDoc is `opencollection.yml`.
type collectionDoc struct {
	OpenCollection string          `yaml:"opencollection"`
	Info           infoDoc         `yaml:"info"`
	Runtime        *runtimeDoc     `yaml:"runtime,omitempty"`
	Settings       *map[string]any `yaml:"settings,omitempty"`
	Extensions     *map[string]any `yaml:"extensions,omitempty"`
}

// infoDoc is the shared `info:` header for collections, folders and requests.
type infoDoc struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type,omitempty"`
	Seq         *int     `yaml:"seq,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// requestDoc is a request file.
type requestDoc struct {
	Info       infoDoc         `yaml:"info"`
	HTTP       *httpDoc        `yaml:"http,omitempty"`
	GRPC       *map[string]any `yaml:"grpc,omitempty"`
	WS         *map[string]any `yaml:"ws,omitempty"`
	Runtime    *runtimeDoc     `yaml:"runtime,omitempty"`
	Settings   *settingsDoc    `yaml:"settings,omitempty"`
	Docs       string          `yaml:"docs,omitempty"`
	Assertions []assertDoc     `yaml:"assertions,omitempty"`
}

// httpDoc is the `http:` section of a request.
type httpDoc struct {
	Method  string     `yaml:"method"`
	URL     string     `yaml:"url"`
	Headers []kvDoc    `yaml:"headers,omitempty"`
	Params  []paramDoc `yaml:"params,omitempty"`
	Auth    *authDoc   `yaml:"auth,omitempty"`
	Body    *bodyDoc   `yaml:"body,omitempty"`
	// TLS and proxy overrides live here in some versions; kept as raw maps so
	// unknown keys survive a read.
	Extra map[string]any `yaml:",inline,omitempty"`
}

// kvDoc is a name/value row with an optional enabled flag.
type kvDoc struct {
	Name        string `yaml:"name"`
	Value       string `yaml:"value,omitempty"`
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type,omitempty"`
	Disabled    bool   `yaml:"disabled,omitempty"`
	ContentType string `yaml:"contentType,omitempty"`
}

// paramDoc is a query or path parameter.
type paramDoc struct {
	Name     string `yaml:"name"`
	Value    string `yaml:"value,omitempty"`
	Type     string `yaml:"type,omitempty"` // query | path
	Disabled bool   `yaml:"disabled,omitempty"`
}

// bodyDoc is the request body.
//
// Bruno writes three shapes for the same concept, and all three occur in the
// wild, so each payload field accepts either a scalar or a list:
//
//	body:
//	  mode: json          # shape 1: mode + payload key
//	  json: |
//	    {}
//
//	body:
//	  type: json          # shape 2: type + data
//	  data: |-
//	    {}
//
//	body:
//	  type: form-urlencoded  # shape 3: type + data as a list of rows
//	  data:
//	    - name: a
//	      value: b
//
// Modelling `data` as yaml.Node keeps both scalar-text and row-list bodies
// parseable without a second decode pass.
type bodyDoc struct {
	Mode string `yaml:"mode,omitempty"`
	Type string `yaml:"type,omitempty"`

	// Single-payload bodies written with a named key (shape 1).
	JSON   string `yaml:"json,omitempty"`
	Text   string `yaml:"text,omitempty"`
	XML    string `yaml:"xml,omitempty"`
	Sparql string `yaml:"sparql,omitempty"`

	// GraphQL is its own nested mapping.
	GraphQL *graphqlDoc `yaml:"graphql,omitempty"`
	// File is a list of file entries.
	File []fileDoc `yaml:"file,omitempty"`

	// Data holds the payload for the `type:` shapes: either a scalar (text body)
	// or a sequence of rows (form and multipart bodies). It is a yaml.Node value,
	// not a pointer: the decoder writes into it directly, and a pointer field
	// rejects node kinds other than the one it was allocated for.
	Data yaml.Node `yaml:"data,omitempty"`
}

// dataText returns the scalar form of Data, or "".
func (b *bodyDoc) dataText() string {
	if b.Data.Kind != yaml.ScalarNode {
		return ""
	}
	return b.Data.Value
}

// dataRows returns the row list in Data, or nil.
func (b *bodyDoc) dataRows() []kvDoc {
	if b.Data.Kind != yaml.SequenceNode {
		return nil
	}
	var rows []kvDoc
	if err := b.Data.Decode(&rows); err != nil {
		return nil
	}
	return rows
}

// graphqlDoc is a GraphQL payload.
type graphqlDoc struct {
	Query     string `yaml:"query,omitempty"`
	Variables string `yaml:"variables,omitempty"`
}

// fileDoc is a file body entry.
type fileDoc struct {
	FilePath    string `yaml:"filePath"`
	ContentType string `yaml:"contentType,omitempty"`
	Selected    bool   `yaml:"selected,omitempty"`
}

// authDoc is the auth section; the field set depends on `type`.
type authDoc struct {
	Type string `yaml:"type"`

	Username  *string `yaml:"username,omitempty"`
	Password  *string `yaml:"password,omitempty"`
	Token     *string `yaml:"token,omitempty"`
	Domain    *string `yaml:"domain,omitempty"`
	Key       *string `yaml:"key,omitempty"`
	Value     *string `yaml:"value,omitempty"`
	Placement string  `yaml:"placement,omitempty"`

	AccessKeyID     *string `yaml:"accessKeyId,omitempty"`
	SecretAccessKey *string `yaml:"secretAccessKey,omitempty"`
	SessionToken    *string `yaml:"sessionToken,omitempty"`
	Service         *string `yaml:"service,omitempty"`
	Region          *string `yaml:"region,omitempty"`

	// OAuth2
	GrantType            string  `yaml:"grantType,omitempty"`
	AuthorizationURL     *string `yaml:"authorizationUrl,omitempty"`
	AccessTokenURL       *string `yaml:"accessTokenUrl,omitempty"`
	RefreshTokenURL      *string `yaml:"refreshTokenUrl,omitempty"`
	CallbackURL          *string `yaml:"callbackUrl,omitempty"`
	ClientID             *string `yaml:"clientId,omitempty"`
	ClientSecret         *string `yaml:"clientSecret,omitempty"`
	Scope                *string `yaml:"scope,omitempty"`
	State                *string `yaml:"state,omitempty"`
	PKCE                 *bool   `yaml:"pkce,omitempty"`
	CredentialsPlacement *string `yaml:"credentialsPlacement,omitempty"`
	CredentialsID        *string `yaml:"credentialsId,omitempty"`
	TokenPlacement       *string `yaml:"tokenPlacement,omitempty"`
	TokenHeaderPrefix    *string `yaml:"tokenHeaderPrefix,omitempty"`
	TokenQueryKey        *string `yaml:"tokenQueryKey,omitempty"`

	// OAuth1
	ConsumerKey       *string `yaml:"consumerKey,omitempty"`
	ConsumerSecret    *string `yaml:"consumerSecret,omitempty"`
	AccessToken       *string `yaml:"accessToken,omitempty"`
	AccessTokenSecret *string `yaml:"accessTokenSecret,omitempty"`
	SignatureMethod   *string `yaml:"signatureMethod,omitempty"`
}

// runtimeDoc holds variables and scripts.
type runtimeDoc struct {
	Variables []runtimeVarDoc `yaml:"variables,omitempty"`
	Scripts   []scriptDoc     `yaml:"scripts,omitempty"`
}

// runtimeVarDoc is a runtime variable; typed values nest a type/data pair.
type runtimeVarDoc struct {
	Name        string     `yaml:"name"`
	Value       *yamlValue `yaml:"value,omitempty"`
	Description string     `yaml:"description,omitempty"`
	Disabled    bool       `yaml:"disabled,omitempty"`
}

// yamlValue accepts either a scalar or a `{type, data}` object.
type yamlValue struct {
	// Scalar holds a plain string value.
	Scalar *string
	// Type and Data hold a typed value.
	Type string
	Data string
	// Raw keeps the original form so a value the codec does not understand is
	// still written back rather than dropped.
	Raw any
}

// scriptDoc is one runtime script.
type scriptDoc struct {
	Type string `yaml:"type"`
	Code string `yaml:"code"`
}

// assertDoc is one assertion.
type assertDoc struct {
	Expression string `yaml:"expression"`
	Operator   string `yaml:"operator,omitempty"`
	Value      string `yaml:"value,omitempty"`
}

// settingsDoc is the per-request settings section.
type settingsDoc struct {
	EncodeURL                  *bool   `yaml:"encodeUrl,omitempty"`
	FollowRedirects            *bool   `yaml:"followRedirects,omitempty"`
	MaxRedirects               *int    `yaml:"maxRedirects,omitempty"`
	Timeout                    *int    `yaml:"timeout,omitempty"`
	ForwardAuthorizationHeader *bool   `yaml:"forwardAuthorizationHeader,omitempty"`
	OmitHeaders                *string `yaml:"omitHeaders,omitempty"`
}

// folderDoc is a folder file: the info header plus inherited auth, headers and
// runtime blocks.
type folderDoc struct {
	Info    infoDoc     `yaml:"info"`
	Request *httpDoc    `yaml:"request,omitempty"`
	Runtime *runtimeDoc `yaml:"runtime,omitempty"`
	Docs    string      `yaml:"docs,omitempty"`
}

// environmentDoc is an environment file.
type environmentDoc struct {
	Name      string          `yaml:"name"`
	Variables []runtimeVarDoc `yaml:"variables,omitempty"`
	Secrets   []string        `yaml:"secrets,omitempty"`
	Extends   string          `yaml:"extends,omitempty"`
	Color     string          `yaml:"color,omitempty"`
}
