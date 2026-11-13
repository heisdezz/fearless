package bru

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// Codec implements format.Codec for the `.bru` dialect.
type Codec struct{}

// New returns the `.bru` codec.
func New() *Codec { return &Codec{} }

// Compile-time check that we satisfy the interface.
var _ format.Codec = (*Codec)(nil)

// Name reports the dialect.
func (c *Codec) Name() model.CollectionFormat { return model.FormatBru }

// collectionConfigFile is the marker that identifies a `.bru` collection.
const collectionConfigFile = "bruno.json"

// Detect reports whether dir holds a `.bru` collection.
func (c *Codec) Detect(dir string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, collectionConfigFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// collectionJSON is the shape of `bruno.json`.
//
// Only the keys Bruno writes are modelled; unknown keys are tolerated on read
// (Bruno's schema does not reject them) but are not preserved on write.
type collectionJSON struct {
	Version string   `json:"version"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Ignore  []string `json:"ignore,omitempty"`
	Presets *struct {
		RequestType     string `json:"requestType,omitempty"`
		RequestUrl      string `json:"requestUrl,omitempty"`
		FolderName      string `json:"folderName,omitempty"`
		EnvironmentName string `json:"environmentName,omitempty"`
	} `json:"presets,omitempty"`
}

// ParseCollection reads a `bruno.json` file.
func (c *Codec) ParseCollection(data []byte) (*model.CollectionConfig, error) {
	var raw collectionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("bru: parse %s: %w", collectionConfigFile, err)
	}
	cfg := &model.CollectionConfig{
		Version: raw.Version,
		Name:    raw.Name,
		Type:    raw.Type,
		Ignore:  raw.Ignore,
		Format:  model.FormatBru,
	}
	if raw.Presets != nil {
		cfg.Presets = &model.Presets{
			RequestType:     raw.Presets.RequestType,
			RequestUrl:      raw.Presets.RequestUrl,
			FolderName:      raw.Presets.FolderName,
			EnvironmentName: raw.Presets.EnvironmentName,
		}
	}
	return cfg, nil
}

// WriteCollection serialises a collection config as `bruno.json`.
func (c *Codec) WriteCollection(cfg *model.CollectionConfig) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("bru: nil collection config")
	}
	raw := collectionJSON{
		Version: cfg.Version,
		Name:    cfg.Name,
		Type:    cfg.Type,
		Ignore:  cfg.Ignore,
	}
	if cfg.Version == "" {
		raw.Version = "1"
	}
	if cfg.Type == "" {
		raw.Type = "collection"
	}
	if cfg.Presets != nil {
		raw.Presets = &struct {
			RequestType     string `json:"requestType,omitempty"`
			RequestUrl      string `json:"requestUrl,omitempty"`
			FolderName      string `json:"folderName,omitempty"`
			EnvironmentName string `json:"environmentName,omitempty"`
		}{
			RequestType:     cfg.Presets.RequestType,
			RequestUrl:      cfg.Presets.RequestUrl,
			FolderName:      cfg.Presets.FolderName,
			EnvironmentName: cfg.Presets.EnvironmentName,
		}
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// ParseRequest reads a request `.bru` file.
func (c *Codec) ParseRequest(data []byte) (*model.Request, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.Request()
}

// ParseFolder reads a `folder.bru` file.
func (c *Codec) ParseFolder(data []byte) (*model.Item, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	item := &model.Item{Type: model.ItemFolder}
	if meta, ok := doc.FindBlock("meta"); ok {
		for _, p := range meta.Pairs {
			switch p.Key {
			case "name":
				item.Name = p.Value
			case "seq":
				if n, err := strconv.Atoi(strings.TrimSpace(p.Value)); err == nil {
					item.Seq = &n
				}
			case "tags":
				item.Tags = append([]string(nil), p.List...)
			}
		}
	}
	auth, err := doc.Auth()
	if err != nil {
		return nil, err
	}
	item.Auth = auth
	if blk, ok := doc.FindBlock("headers"); ok {
		for i, p := range blk.Pairs {
			item.Headers = append(item.Headers, keyValueFrom(p, "header", i))
		}
	}
	if blk, ok := doc.FindBlock("vars:pre-request"); ok {
		item.Vars = variablesFrom(blk)
	}
	// A plain `vars` block is the collection's own scope: collection.bru holds
	// the variables `{{name}}` resolves against before any environment. It is
	// read into the same field, since the root file has no other use for it.
	if len(item.Vars) == 0 {
		if blk, ok := doc.FindBlock("vars"); ok {
			item.Vars = variablesFrom(blk)
		}
	}
	if sc := buildScripts(doc); sc != nil {
		item.Script = sc
	}
	return item, nil
}

// ParseEnvironment reads an environment `.bru` file.
func (c *Codec) ParseEnvironment(data []byte) (*model.Environment, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	env := &model.Environment{Format: model.FormatBru, Color: doc.Color}

	// Environment files have no meta block: the name is the filename, which the
	// caller knows and the parser does not. `extends` carries the parent.
	if len(doc.Extends) > 0 {
		env.Extends = doc.Extends[0]
	}

	if blk, ok := doc.FindBlock("vars"); ok {
		env.Variables = variablesFrom(blk)
	}
	if blk, ok := doc.FindBlock("vars:secret"); ok {
		env.Secrets = append(env.Secrets, blk.Items...)
	}

	// Mark secret variables so the secret store is consulted for them.
	secret := make(map[string]bool, len(env.Secrets))
	for _, s := range env.Secrets {
		secret[s] = true
	}
	for i := range env.Variables {
		if name := env.Variables[i].NameOrEmpty(); secret[name] {
			env.Variables[i].Secret = true
		}
	}
	return env, nil
}

// WriteRequest serialises a request as `.bru`.
//
// The output is semantics-preserving, not byte-identical to the source: block and
// key order follow Bruno's own serialiser. Round-trip tests assert
// parse(write(parse(x))) == parse(x).
func (c *Codec) WriteRequest(req *model.Request) ([]byte, error) {
	if req == nil {
		return nil, errors.New("bru: nil request")
	}
	return []byte(writeRequest(req)), nil
}

// WriteFolder serialises a folder file.
func (c *Codec) WriteFolder(folder *model.Item) ([]byte, error) {
	if folder == nil {
		return nil, errors.New("bru: nil folder")
	}
	return []byte(writeFolder(folder)), nil
}

// WriteCollectionRoot serialises `collection.bru`, the root file that holds
// collection-level settings. Only the variables block is written: they are the
// values `{{name}}` resolves against at the collection scope, so a collection
// saved without them loses every request's base URL on reload.
func (c *Codec) WriteCollectionRoot(vars []model.Variable) ([]byte, error) {
	w := &writer{}
	if len(vars) > 0 {
		w.open(0, "vars")
		for _, v := range vars {
			w.writeVariable(1, v)
		}
		w.closeBlock(0)
	}
	return []byte(strings.TrimRight(w.String(), "\n") + "\n"), nil
}

// WriteEnvironment serialises an environment file.
func (c *Codec) WriteEnvironment(env *model.Environment) ([]byte, error) {
	if env == nil {
		return nil, errors.New("bru: nil environment")
	}
	return []byte(writeEnvironment(env)), nil
}

// Register installs the codec in the format registry. It is called from an
// explicit wiring point rather than init(), so the dependency graph stays
// predictable (see docs/implementation/package-structure.md §2).
func Register() { format.Register(New()) }
