package model

// Scripts holds the two script hooks attached to a request.
type Scripts struct {
	Req *string `json:"req,omitempty" yaml:"req,omitempty"`
	Res *string `json:"res,omitempty" yaml:"res,omitempty"`
}

// VarBlocks holds the declarative variable blocks (`vars:pre-request` and
// `vars:post-response`), distinct from JS scripts.
type VarBlocks struct {
	PreRequest   []Variable `json:"req,omitempty" yaml:"req,omitempty"`
	PostResponse []Variable `json:"res,omitempty" yaml:"res,omitempty"`
}

// Request is the canonical HTTP request model, shared by both dialects.
//
// Headers and params are ordered slices, never maps: the user's order is
// preserved to disk and to the wire, and duplicates are legal.
type Request struct {
	URL        string       `json:"url" yaml:"url"`
	Method     string       `json:"method" yaml:"method"`
	Headers    []KeyValue   `json:"headers" yaml:"headers"`
	Params     []Param      `json:"params" yaml:"params"`
	Auth       *Auth        `json:"auth,omitempty" yaml:"auth,omitempty"`
	Body       *Body        `json:"body,omitempty" yaml:"body,omitempty"`
	Script     *Scripts     `json:"script,omitempty" yaml:"script,omitempty"`
	Vars       *VarBlocks   `json:"vars,omitempty" yaml:"vars,omitempty"`
	Assertions []KeyValue   `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Tests      *string      `json:"tests,omitempty" yaml:"tests,omitempty"`
	Docs       *string      `json:"docs,omitempty" yaml:"docs,omitempty"`
	Settings   ItemSettings `json:"settings" yaml:"settings"`
	Name       string       `json:"name,omitempty" yaml:"name,omitempty"`
	Seq        *int         `json:"seq,omitempty" yaml:"seq,omitempty"`
	Tags       []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// QueryParams returns the enabled query parameters, in order.
func (r *Request) QueryParams() []Param {
	if r == nil {
		return nil
	}
	out := make([]Param, 0, len(r.Params))
	for _, p := range r.Params {
		if p.Type == ParamQuery && p.Live() {
			out = append(out, p)
		}
	}
	return out
}

// PathParams returns the enabled path parameters, in order.
func (r *Request) PathParams() []Param {
	if r == nil {
		return nil
	}
	out := make([]Param, 0, len(r.Params))
	for _, p := range r.Params {
		if p.Type == ParamPath && p.Live() {
			out = append(out, p)
		}
	}
	return out
}

// LiveHeaders returns the enabled, named headers in their stored order.
func (r *Request) LiveHeaders() []KeyValue {
	if r == nil {
		return nil
	}
	out := make([]KeyValue, 0, len(r.Headers))
	for _, h := range r.Headers {
		if h.Live() {
			out = append(out, h)
		}
	}
	return out
}

// Clone returns a deep copy. Used by the draft model so a draft can be edited
// without mutating the saved request.
func (r *Request) Clone() *Request {
	if r == nil {
		return nil
	}
	cp := *r
	cp.Headers = cloneKeyValues(r.Headers)
	cp.Params = cloneParams(r.Params)
	cp.Assertions = cloneKeyValues(r.Assertions)
	cp.Tags = append([]string(nil), r.Tags...)
	cp.Body = cloneBody(r.Body)
	cp.Auth = cloneAuth(r.Auth)
	if r.Script != nil {
		s := *r.Script
		s.Req = cloneStringPtr(r.Script.Req)
		s.Res = cloneStringPtr(r.Script.Res)
		cp.Script = &s
	}
	cp.Tests = cloneStringPtr(r.Tests)
	cp.Docs = cloneStringPtr(r.Docs)
	cp.Name = r.Name
	cp.Seq = cloneIntPtr(r.Seq)
	return &cp
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneKeyValues(in []KeyValue) []KeyValue {
	if in == nil {
		return nil
	}
	out := make([]KeyValue, len(in))
	for i, kv := range in {
		out[i] = kv
		out[i].Name = cloneStringPtr(kv.Name)
		out[i].Value = cloneStringPtr(kv.Value)
		out[i].Description = cloneStringPtr(kv.Description)
		out[i].Annotations = append([]Annotation(nil), kv.Annotations...)
	}
	return out
}

func cloneParams(in []Param) []Param {
	if in == nil {
		return nil
	}
	out := make([]Param, len(in))
	for i, p := range in {
		out[i] = p
		out[i].Name = cloneStringPtr(p.Name)
		out[i].Value = cloneStringPtr(p.Value)
		out[i].Description = cloneStringPtr(p.Description)
		out[i].Annotations = append([]Annotation(nil), p.Annotations...)
	}
	return out
}

func cloneBody(b *Body) *Body {
	if b == nil {
		return nil
	}
	cp := *b
	cp.JSON = cloneStringPtr(b.JSON)
	cp.Text = cloneStringPtr(b.Text)
	cp.XML = cloneStringPtr(b.XML)
	cp.Sparql = cloneStringPtr(b.Sparql)
	cp.FormURLEncoded = cloneKeyValues(b.FormURLEncoded)
	if b.GraphQL != nil {
		g := *b.GraphQL
		g.Query = cloneStringPtr(b.GraphQL.Query)
		g.Variables = cloneStringPtr(b.GraphQL.Variables)
		cp.GraphQL = &g
	}
	if b.MultipartForm != nil {
		cp.MultipartForm = make([]MultipartEntry, len(b.MultipartForm))
		for i, m := range b.MultipartForm {
			cp.MultipartForm[i] = m
			cp.MultipartForm[i].Name = cloneStringPtr(m.Name)
			cp.MultipartForm[i].Value = cloneStringPtr(m.Value)
			cp.MultipartForm[i].ContentType = cloneStringPtr(m.ContentType)
			cp.MultipartForm[i].Description = cloneStringPtr(m.Description)
			cp.MultipartForm[i].Annotations = append([]Annotation(nil), m.Annotations...)
		}
	}
	if b.File != nil {
		cp.File = make([]FileEntry, len(b.File))
		for i, f := range b.File {
			cp.File[i] = f
			cp.File[i].FilePath = cloneStringPtr(f.FilePath)
			cp.File[i].ContentType = cloneStringPtr(f.ContentType)
			cp.File[i].Description = cloneStringPtr(f.Description)
			cp.File[i].Annotations = append([]Annotation(nil), f.Annotations...)
		}
	}
	return &cp
}

func cloneAuth(a *Auth) *Auth {
	if a == nil {
		return nil
	}
	cp := *a
	for _, p := range []**string{
		&cp.Username, &cp.Password, &cp.Domain, &cp.Token, &cp.Key, &cp.Value,
		&cp.AccessKeyID, &cp.SecretAccessKey, &cp.SessionToken, &cp.Service,
		&cp.Region, &cp.ProfileName, &cp.ConsumerKey, &cp.ConsumerSecret,
		&cp.AccessToken, &cp.AccessTokenSecret, &cp.CallbackURL, &cp.Verifier,
		&cp.SignatureMethod, &cp.PrivateKey, &cp.PrivateKeyType, &cp.Timestamp,
		&cp.Nonce, &cp.Version, &cp.Realm, &cp.AuthorizationURL, &cp.AccessTokenURL,
		&cp.RefreshTokenURL, &cp.ClientID, &cp.ClientSecret, &cp.Scope, &cp.State,
		&cp.CredentialsPlacement, &cp.CredentialsID, &cp.TokenPlacement,
		&cp.TokenHeaderPrefix, &cp.TokenQueryKey, &cp.TokenSource,
		&cp.EdgeGridClientToken, &cp.EdgeGridClientSecret,
		&cp.EdgeGridBaseURL, &cp.EdgeGridHeadersToSign,
	} {
		*p = cloneStringPtr(*p)
	}
	cp.IncludeBodyHash = cloneBoolPtr(a.IncludeBodyHash)
	cp.PKCE = cloneBoolPtr(a.PKCE)
	cp.AutoRefreshToken = cloneBoolPtr(a.AutoRefreshToken)
	cp.AutoFetchToken = cloneBoolPtr(a.AutoFetchToken)
	cp.EdgeGridMaxBodySize = cloneIntPtr(a.EdgeGridMaxBodySize)
	// AdditionalParams is shared by reference intentionally: it is replaced, not
	// mutated, by the editor, and cloning it is not on a hot path.
	return &cp
}

// Item is a node in a collection tree: a request, a folder, or an inert entry.
//
// Saved and Draft model Bruno's draft duality: Draft is nil when the item has no
// uncommitted edits, and non-nil while the user is editing. Saving promotes
// Draft to Saved and clears Draft.
type Item struct {
	UID      string   `json:"uid" yaml:"uid"`
	Type     ItemType `json:"type" yaml:"type"`
	Name     string   `json:"name" yaml:"name"`
	Path     string   `json:"path,omitempty" yaml:"path,omitempty"`
	Seq      *int     `json:"seq,omitempty" yaml:"seq,omitempty"`
	Filename string   `json:"filename,omitempty" yaml:"filename,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Request holds the saved request for request items.
	Request *Request `json:"request,omitempty" yaml:"request,omitempty"`
	// Draft holds in-progress edits; nil when clean.
	Draft *Request `json:"draft,omitempty" yaml:"draft,omitempty"`

	// Folder-level configuration, applied to descendants.
	Auth    *Auth      `json:"auth,omitempty" yaml:"auth,omitempty"`
	Headers []KeyValue `json:"headers,omitempty" yaml:"headers,omitempty"`
	Vars    []Variable `json:"vars,omitempty" yaml:"vars,omitempty"`
	Script  *Scripts   `json:"script,omitempty" yaml:"script,omitempty"`

	// Children is populated for folders.
	Children []*Item `json:"children,omitempty" yaml:"children,omitempty"`

	// Examples are saved responses attached to a request.
	Examples []*Example `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// IsFolder reports whether the item is a folder.
func (i *Item) IsFolder() bool { return i != nil && i.Type == ItemFolder }

// Effective returns the request the editor is working on: the draft when one
// exists, otherwise the saved request.
func (i *Item) Effective() *Request {
	if i == nil {
		return nil
	}
	if i.Draft != nil {
		return i.Draft
	}
	return i.Request
}

// Dirty reports whether the item has uncommitted edits.
func (i *Item) Dirty() bool { return i != nil && i.Draft != nil }

// Example is a saved response attached to a request.
type Example struct {
	UID      string   `json:"uid" yaml:"uid"`
	Name     string   `json:"name" yaml:"name"`
	Response *Result  `json:"response,omitempty" yaml:"response,omitempty"`
	Request  *Request `json:"request,omitempty" yaml:"request,omitempty"`
}
