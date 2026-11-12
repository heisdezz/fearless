package dialogs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"fearless/internal/convert/curl"
	"fearless/internal/convert/insomnia"
	"fearless/internal/convert/openapi"
	"fearless/internal/convert/postman"
	"fearless/internal/core/collection"
	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// ImportSource is the document format an import dialog accepts.
type ImportSource int

const (
	// ImportAuto sniffs the document. A leading '{' or '[' is JSON, which is then
	// tested for Postman, Insomnia or OpenAPI; anything else is treated as cURL.
	ImportAuto ImportSource = iota
	// ImportCurl is a cURL command line.
	ImportCurl
	// ImportPostman is a Postman v2.0/v2.1 collection JSON.
	ImportPostman
	// ImportInsomnia is an Insomnia v4/v5 export, JSON or YAML.
	ImportInsomnia
	// ImportOpenAPI is an OpenAPI 3 / Swagger 2 document, JSON or YAML.
	ImportOpenAPI
)

// String names the source for the picker.
func (s ImportSource) String() string {
	switch s {
	case ImportCurl:
		return "cURL"
	case ImportPostman:
		return "Postman"
	case ImportInsomnia:
		return "Insomnia"
	case ImportOpenAPI:
		return "OpenAPI"
	}
	return "Auto-detect"
}

// Imported is the result of reading a pasted document.
//
// Exactly one of Request or Collection is set: a cURL command yields a single
// request, the other three yield a whole collection.
type Imported struct {
	// Source is the format that was detected or chosen.
	Source ImportSource
	// Request is set for a single-request import.
	Request *model.Request
	// Collection is set for a collection import.
	Collection *model.Collection
	// Warnings are things the converter dropped or adapted.
	Warnings []string
}

// ErrImportEmpty rejects a document with no requests, which is more often a
// wrong-format paste than a genuinely empty collection.
var ErrImportEmpty = errors.New("dialogs: the document contained no requests")

// ErrImportUnknown reports that auto-detection could not place the document.
var ErrImportUnknown = errors.New("dialogs: could not tell what format this is")

// ParseImport converts text, file path, or URL into a request or collection.
func ParseImport(src ImportSource, text string) (*Imported, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrImportEmpty
	}

	// 1. Filesystem path check (directory or file)
	if fi, err := os.Stat(text); err == nil {
		if fi.IsDir() {
			col, err := collection.Load(text)
			if err != nil {
				return nil, fmt.Errorf("dialogs: loading collection: %w", err)
			}
			if col == nil {
				return nil, ErrImportEmpty
			}
			return &Imported{Source: ImportAuto, Collection: col}, nil
		}
		fileBytes, err := os.ReadFile(text)
		if err != nil {
			return nil, fmt.Errorf("dialogs: reading file: %w", err)
		}
		if strings.HasSuffix(strings.ToLower(text), ".bru") {
			codec, ok := format.Get(model.FormatBru)
			if ok {
				req, err := codec.ParseRequest(fileBytes)
				if err == nil && req != nil {
					return &Imported{Source: ImportAuto, Request: req}, nil
				}
			}
		}
		return parseDocumentContent(src, string(fileBytes))
	}

	// 2. URL fetch check
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(text)
		if err == nil {
			defer resp.Body.Close()
			bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
			if readErr == nil && len(bodyBytes) > 0 {
				if imp, err := parseDocumentContent(src, string(bodyBytes)); err == nil && imp != nil {
					return imp, nil
				}
			}
		}
		// If remote fetch was not a multi-request specification, import URL as single GET request
		return &Imported{
			Source: ImportCurl,
			Request: &model.Request{
				Method: "GET",
				URL:    text,
			},
		}, nil
	}

	// 3. Raw .bru text check
	if strings.Contains(text, "meta {") && (strings.Contains(text, "get {") || strings.Contains(text, "post {") || strings.Contains(text, "put {") || strings.Contains(text, "delete {") || strings.Contains(text, "http {")) {
		codec, ok := format.Get(model.FormatBru)
		if ok {
			req, err := codec.ParseRequest([]byte(text))
			if err == nil && req != nil {
				return &Imported{Source: ImportAuto, Request: req}, nil
			}
		}
	}

	return parseDocumentContent(src, text)
}

func parseDocumentContent(src ImportSource, text string) (*Imported, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrImportEmpty
	}

	src = detectImportSource(src, text)

	switch src {
	case ImportCurl:
		res, err := curl.Import(text)
		if err != nil {
			return nil, fmt.Errorf("dialogs: parsing cURL: %w", err)
		}
		if res == nil || res.Request == nil {
			return nil, ErrImportEmpty
		}
		return &Imported{Source: ImportCurl, Request: res.Request, Warnings: res.Warnings}, nil

	case ImportPostman:
		res, err := postman.Import([]byte(text))
		if err != nil {
			return nil, fmt.Errorf("dialogs: parsing Postman collection: %w", err)
		}
		if res == nil || res.Collection == nil {
			return nil, ErrImportEmpty
		}
		return &Imported{Source: ImportPostman, Collection: res.Collection, Warnings: res.Warnings}, nil

	case ImportInsomnia:
		res, err := insomnia.Import([]byte(text))
		if err != nil {
			return nil, fmt.Errorf("dialogs: parsing Insomnia export: %w", err)
		}
		if res == nil {
			return nil, ErrImportEmpty
		}
		return &Imported{Source: ImportInsomnia, Collection: res.Collection, Warnings: res.Warnings}, nil

	case ImportOpenAPI:
		res, err := openapi.Import([]byte(text))
		if err != nil {
			return nil, fmt.Errorf("dialogs: parsing OpenAPI document: %w", err)
		}
		if res == nil || res.Collection == nil {
			return nil, ErrImportEmpty
		}
		return &Imported{Source: ImportOpenAPI, Collection: res.Collection, Warnings: res.Warnings}, nil
	}

	return nil, ErrImportUnknown
}

// detectImportSource resolves ImportAuto by looking at the text. It is a
// heuristic for the picker, not a parser: a wrong guess still fails loudly in
// ParseImport rather than importing something unexpected.
func detectImportSource(src ImportSource, text string) ImportSource {
	if src != ImportAuto {
		return src
	}
	trimmed := strings.TrimLeft(text, " \t\r\n\ufeff")
	if strings.HasPrefix(trimmed, "curl ") || strings.HasPrefix(trimmed, "curl\t") {
		return ImportCurl
	}
	lower := strings.ToLower(text)
	// Format markers survive both minified and pretty-printed JSON, so compare
	// whitespace-stripped text rather than exact substrings.
	tight := stripSpace(lower)
	switch {
	case strings.Contains(tight, `"_postman_id"`) || strings.Contains(tight, "getpostman.com"):
		return ImportPostman
	case strings.Contains(tight, `"_type":"export"`) || strings.Contains(tight, "_type:export"):
		return ImportInsomnia
	case strings.Contains(tight, `"openapi"`) || strings.Contains(tight, `"swagger"`) ||
		strings.Contains(tight, "openapi:") || strings.Contains(tight, "swagger:"):
		return ImportOpenAPI
	}
	// JSON that is none of the above, or YAML with no marker, is not something we
	// can place.
	return ImportAuto
}

// stripSpace removes spaces and tabs. Newlines are kept so a YAML key at the
// start of a line is not accidentally glued to the value above it.
func stripSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// CreateCollectionModal opens the modal to create a new collection.
func (q *Queue) CreateCollectionModal(onCreate func(ctx context.Context, name, location string, format model.CollectionFormat) error) (*Wait, error) {
	fields := []Field{
		{Key: "name", Label: "Name", Placeholder: "Collection Name", Required: true},
		{Key: "location", Label: "Location", Placeholder: "Folder path on filesystem", Required: true},
		{Key: "format", Label: "File Format", Placeholder: "bru or yml"},
	}
	ch, err := q.open(KindCreateCollection, "Create Collection", "", "Create", fields, func(ctx context.Context, values map[string]string) error {
		name := strings.TrimSpace(values["name"])
		if name == "" {
			return errors.New("collection name is required")
		}
		loc := strings.TrimSpace(values["location"])
		if loc == "" {
			return errors.New("location is required")
		}
		fmtVal := strings.ToLower(strings.TrimSpace(values["format"]))
		fmtChoice := model.FormatBru
		if fmtVal == "yml" || fmtVal == "yaml" || fmtVal == "opencollection" {
			fmtChoice = model.FormatYML
		}
		return onCreate(ctx, name, loc, fmtChoice)
	})
	if err != nil {
		return newWait(nil), err
	}
	q.notify()
	return newWait(ch), nil
}

// StorageLocationModal asks, on first run, which folder collections are saved
// in. It is the app's one uncancellable dialog: until a folder is chosen there
// is nowhere to write, so it ignores cancel and keeps asking.
//
// onChoose runs on the action goroutine; returning an error keeps the dialog
// open with the message so the user can pick a different folder.
func (q *Queue) StorageLocationModal(onChoose func(ctx context.Context, location string) error) (*Wait, error) {
	ch, err := q.open(KindStorageLocation, "Choose where to save your work", "Fearless keeps collections as plain files on disk. Pick a folder for new collections and workspaces.", "Continue", []Field{
		{Key: "location", Label: "Folder", Placeholder: "Folder path on filesystem", Required: true},
	}, func(ctx context.Context, values map[string]string) error {
		loc := strings.TrimSpace(values["location"])
		if loc == "" {
			return errors.New("choose a folder")
		}
		return onChoose(ctx, loc)
	})
	if err != nil {
		return newWait(nil), err
	}
	q.notify()
	return newWait(ch), nil
}

// ImportModal opens the import dialog. onImported receives the parsed document
// and runs on the action goroutine; its error keeps the dialog open so the user
// can fix the paste.
func (q *Queue) ImportModal(onImported func(context.Context, *Imported) error) (*Wait, error) {
	ch, err := q.open(KindImport, "Import Collection", "", "Import", []Field{{
		Key:         "document",
		Label:       "File path, URL, cURL, or raw specification",
		Placeholder: "Enter file path, URL, or paste cURL / Postman / Insomnia / OpenAPI...",
		Multiline:   true,
		Required:    true,
	}}, func(ctx context.Context, values map[string]string) error {
		imp, err := ParseImport(ImportAuto, values["document"])
		if err != nil {
			return err
		}
		// The dialog closes on success, so warnings are only visible if the
		// handler itself fails; passing them along lets it report them its own way.
		return onImported(ctx, imp)
	})
	if err != nil {
		return newWait(nil), err
	}
	q.notify()
	return newWait(ch), nil
}

// SaveTransient describes a request that has no home on disk yet, so it must be
// filed into a collection before it can be saved.
type SaveTransient struct {
	// Name is the request's proposed name, pre-filled in the field.
	Name string
	// URL is shown read-only so the user knows which request is being saved.
	URL string
	// Targets lists the collections that can receive it, by name and UID.
	Targets []SaveTarget
}

// SaveTarget is one collection choice.
type SaveTarget struct {
	// UID identifies the collection.
	UID string
	// Name is what the picker shows.
	Name string
}

// ErrNoSaveTarget reports that there is no collection to save into — the user
// must open one first. The dialog shows this rather than an empty picker.
var ErrNoSaveTarget = errors.New("dialogs: open a collection before saving this request")

// SaveTransientModal asks for a name and a destination collection. onSave runs
// on the action goroutine with the chosen collection UID and name.
//
// If st has no targets, the dialog is not opened at all and the error is
// returned, because a picker with no options cannot be satisfied.
func (q *Queue) SaveTransientModal(st SaveTransient, onSave func(ctx context.Context, collectionUID, name string) error) (*Wait, error) {
	if len(st.Targets) == 0 {
		return newWait(nil), ErrNoSaveTarget
	}
	if st.Name == "" {
		st.Name = suggestedName(st.URL)
	}

	fields := []Field{
		{Key: "name", Label: "Name", Placeholder: "request name", Required: true},
		{Key: "collection", Label: "Collection", Placeholder: "choose a collection", Required: true},
	}
	ch, err := q.open(KindSaveTransient, "Save request", st.URL, "Save", fields, func(ctx context.Context, values map[string]string) error {
		return onSave(ctx, q.TargetUID(values["collection"]), strings.TrimSpace(values["name"]))
	})
	if err != nil {
		return newWait(nil), err
	}
	// Targets must be recorded after open, because open resets per-dialog state.
	q.SetTargets(st.Targets)
	// Pre-fill: the name is editable, the collection starts unset so the user has
	// to make a deliberate choice.
	q.SetValueConcrete("name", st.Name)
	q.notify()
	return newWait(ch), nil
}

// SetValueConcrete is SetValue for code, not for keystrokes: it ignores the
// PhaseRunning guard so a pre-filled default can be set immediately after open.
func (q *Queue) SetValueConcrete(key, value string) {
	if q.values == nil {
		q.values = map[string]string{}
	}
	q.values[key] = value
	q.notify()
}

// suggestedName derives a request name from a URL, matching what Bruno does when
// it creates a request from a scratch tab: the last path segment, or the host
// when the path is empty.
func suggestedName(rawURL string) string {
	s := strings.TrimSpace(rawURL)
	if s == "" {
		return "Untitled"
	}
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimRight(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		seg := s[i+1:]
		// A bare host (no path) leaves the scheme with no slash after it.
		if seg != "" && !strings.Contains(seg, "://") {
			return seg
		}
	}
	// Fall back to the host.
	rest := s
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return "Untitled"
	}
	return rest
}
