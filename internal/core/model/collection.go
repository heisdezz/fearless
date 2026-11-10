package model

// CollectionConfig is the root configuration of a collection, read from
// `bruno.json` (bru dialect) or `opencollection.yml` (yml dialect).
type CollectionConfig struct {
	// Version is the config schema version as written by Bruno.
	Version string `json:"version" yaml:"version"`
	// Name is the collection name.
	Name string `json:"name" yaml:"name"`
	// Type is the collection kind; "collection" for a normal collection.
	Type string `json:"type" yaml:"type"`
	// Ignore lists glob patterns excluded from the collection.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// Format records which dialect was read from disk.
	Format CollectionFormat `json:"-" yaml:"-"`
	// Presets may carry grouping metadata used by the UI.
	Presets *Presets `json:"presets,omitempty" yaml:"presets,omitempty"`
	// Info carries description and contact metadata (yml dialect).
	Info *CollectionInfo `json:"info,omitempty" yaml:"info,omitempty"`
	// Auth is collection-level inherited auth.
	Auth *Auth `json:"auth,omitempty" yaml:"auth,omitempty"`
	// Headers are collection-level inherited headers.
	Headers []KeyValue `json:"headers,omitempty" yaml:"headers,omitempty"`
	// Vars are collection-level variables.
	Vars []Variable `json:"vars,omitempty" yaml:"vars,omitempty"`
	// Script holds collection-level scripts.
	Script *Scripts `json:"script,omitempty" yaml:"script,omitempty"`
	// Docs is the collection README shown in the UI.
	Docs *string `json:"docs,omitempty" yaml:"docs,omitempty"`
	// Security holds TLS and other trust settings.
	Security *CollectionSecurity `json:"security,omitempty" yaml:"security,omitempty"`
}

// Presets is Bruno's optional collection grouping metadata.
type Presets struct {
	RequestType     string `json:"requestType,omitempty" yaml:"requestType,omitempty"`
	RequestUrl      string `json:"requestUrl,omitempty" yaml:"requestUrl,omitempty"`
	FolderName      string `json:"folderName,omitempty" yaml:"folderName,omitempty"`
	EnvironmentName string `json:"environmentName,omitempty" yaml:"environmentName,omitempty"`
}

// CollectionInfo is descriptive metadata for the yml dialect.
type CollectionInfo struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
}

// CollectionSecurity holds collection-scoped TLS settings.
type CollectionSecurity struct {
	// ClientCertificates configures mutual-TLS certificates.
	ClientCertificates *ClientCertificates `json:"clientCertificates,omitempty" yaml:"clientCertificates,omitempty"`
	// VerifyTLS turns server certificate verification off when false.
	VerifyTLS *bool `json:"verifyTls,omitempty" yaml:"verifyTls,omitempty"`
}

// ClientCertificates is a set of domain-scoped client certificates.
type ClientCertificates struct {
	Enabled      bool            `json:"enabled" yaml:"enabled"`
	Certificates []ClientCertRef `json:"certificates,omitempty" yaml:"certificates,omitempty"`
}

// ClientCertRef references a certificate/key pair on disk.
type ClientCertRef struct {
	Domain       string `json:"domain,omitempty" yaml:"domain,omitempty"`
	CertFilePath string `json:"certFilePath" yaml:"certFilePath"`
	KeyFilePath  string `json:"keyFilePath" yaml:"keyFilePath"`
	Passphrase   string `json:"passphrase,omitempty" yaml:"passphrase,omitempty"`
}

// Collection is a loaded collection: its config plus the item tree.
//
// UID is derived from the absolute path so identity survives reloads and file
// renames can be remapped.
type Collection struct {
	UID    string           `json:"uid"`
	Path   string           `json:"path"`
	Name   string           `json:"name"`
	Config CollectionConfig `json:"config"`
	Items  []*Item          `json:"items"`

	// Environments are loaded lazily; Names is always populated.
	Environments []*Environment `json:"environments,omitempty"`
	// ActiveEnvironment names the selected environment, if any.
	ActiveEnvironment string `json:"activeEnvironment,omitempty"`

	// Dirty marks collections with uncommitted changes (UI affordance).
	Dirty bool `json:"dirty,omitempty"`
}

// Environment is a variable set selected before sending.
type Environment struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	// Variables are the named values.
	Variables []Variable `json:"variables"`
	// Color is an optional display colour from the .bru file.
	Color string `json:"color,omitempty"`
	// Extends names a parent environment in yml collections; empty in the bru
	// dialect, which uses a different inheritance field.
	Extends string `json:"extends,omitempty"`
	// Secrets holds variable names whose values live in the secret store.
	Secrets []string `json:"secrets,omitempty"`
	// Format is the dialect the environment was read from.
	Format CollectionFormat `json:"-"`
	// Filename is the file it was read from.
	Filename string `json:"filename,omitempty"`
}

// VariableValue returns the value of the named variable.
func (e *Environment) VariableValue(name string) (any, bool) {
	if e == nil {
		return nil, false
	}
	for _, v := range e.Variables {
		if v.NameOrEmpty() == name && v.Enabled {
			return v.Value, true
		}
	}
	return nil, false
}

// FindItem returns the item with the given UID, searching recursively.
func (c *Collection) FindItem(uid string) *Item {
	if c == nil {
		return nil
	}
	for _, it := range c.Items {
		if found := findItem(it, uid); found != nil {
			return found
		}
	}
	return nil
}

func findItem(it *Item, uid string) *Item {
	if it == nil {
		return nil
	}
	if it.UID == uid {
		return it
	}
	for _, child := range it.Children {
		if found := findItem(child, uid); found != nil {
			return found
		}
	}
	return nil
}

// Walk visits every item depth-first, parents before children.
func (c *Collection) Walk(fn func(*Item) error) error {
	if c == nil {
		return nil
	}
	for _, it := range c.Items {
		if err := walkItem(it, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkItem(it *Item, fn func(*Item) error) error {
	if it == nil {
		return nil
	}
	if err := fn(it); err != nil {
		return err
	}
	for _, child := range it.Children {
		if err := walkItem(child, fn); err != nil {
			return err
		}
	}
	return nil
}

// Ancestors returns the chain of folders containing uid, outermost first. The
// chain is used to merge inherited auth, headers and vars.
func (c *Collection) Ancestors(uid string) []*Item {
	if c == nil {
		return nil
	}
	var path []*Item
	var search func(items []*Item) bool
	search = func(items []*Item) bool {
		for _, it := range items {
			if it == nil {
				continue
			}
			path = append(path, it)
			if it.UID == uid {
				return true
			}
			if search(it.Children) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	search(c.Items)
	if len(path) == 0 {
		return nil
	}
	// The last element is the item itself; ancestors are everything before it.
	return path[:len(path)-1]
}
