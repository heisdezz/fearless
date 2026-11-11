package environment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fearless/internal/core/format"
	"fearless/internal/core/format/bru"
	"fearless/internal/core/format/yamlcodec"
	"fearless/internal/core/model"
)

// registerCodecs wires the real codecs once per test binary.
func registerCodecs() {
	if _, ok := format.Get(model.FormatBru); !ok {
		format.Register(bru.New())
	}
	if _, ok := format.Get(model.FormatYML); !ok {
		format.Register(yamlcodec.New())
	}
}

func TestMain(m *testing.M) {
	registerCodecs()
	os.Exit(m.Run())
}

// writeEnv writes an environment file into dir/environments.
func writeEnv(t *testing.T, dir, filename, content string) {
	t.Helper()
	envDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoaderReadsBruEnvironments covers the .bru dialect end to end.
func TestLoaderReadsBruEnvironments(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "Prod.bru", "vars {\n  host: https://prod.example.com\n  token: abc\n}\n")
	writeEnv(t, dir, "Local.bru", "vars {\n  host: http://localhost:8080\n}\n")
	// A non-environment file in the same directory must be ignored.
	writeEnv(t, dir, "notes.txt", "ignore me")
	writeEnv(t, dir, "nested", "")

	// `nested` is a file, not a directory, so it is skipped by the extension check.
	got, err := Loader{Format: model.FormatBru, Root: dir}.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loaded %d environments, want 2", len(got))
	}

	// Sorted case-insensitively, so "Local" precedes "Prod".
	if got[0].Name != "Local" || got[1].Name != "Prod" {
		t.Errorf("order = %s, %s", got[0].Name, got[1].Name)
	}
	if got[1].Filename != "Prod.bru" {
		t.Errorf("filename = %q", got[1].Filename)
	}
	if got[1].Format != model.FormatBru {
		t.Errorf("format = %q", got[1].Format)
	}
	if v, ok := got[1].VariableValue("host"); !ok || v != "https://prod.example.com" {
		t.Errorf("host = %v, %v", v, ok)
	}
}

// TestLoaderReadsYAMLEnvironments covers the yml dialect.
func TestLoaderReadsYAMLEnvironments(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "Test.yml", `name: Test
variables:
  - name: envVar
    value: original
    enabled: true
    secret: false
`)
	// A .bru file in a yml collection must not be picked up.
	writeEnv(t, dir, "Stray.bru", "vars {\n  x: 1\n}\n")

	got, err := Loader{Format: model.FormatYML, Root: dir}.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Test" {
		t.Fatalf("environments = %+v", got)
	}
	if v, ok := got[0].VariableValue("envVar"); !ok || v != "original" {
		t.Errorf("envVar = %v, %v", v, ok)
	}
}

// TestLoaderFallsBackToFilename covers a file with no name line.
func TestLoaderFallsBackToFilename(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "Staging.bru", "vars {\n  a: 1\n}\n")

	got, err := Loader{Format: model.FormatBru, Root: dir}.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("loaded %d", len(got))
	}
	if got[0].Name != "Staging" {
		t.Errorf("name = %q, want the filename stem", got[0].Name)
	}
}

// TestLoaderMissingDirectory reports the absence distinctly.
func TestLoaderMissingDirectory(t *testing.T) {
	_, err := Loader{Format: model.FormatBru, Root: t.TempDir()}.Load()
	if !errors.Is(err, ErrNoEnvironments) {
		t.Fatalf("err = %v, want ErrNoEnvironments", err)
	}
}

// TestLoaderUnknownFormat reports a format with no codec.
func TestLoaderUnknownFormat(t *testing.T) {
	_, err := Loader{Format: model.CollectionFormat("nonsense"), Root: t.TempDir()}.Load()
	if !errors.Is(err, format.ErrUnknownFormat) {
		t.Fatalf("err = %v, want ErrUnknownFormat", err)
	}
}

// TestLoaderMalformedFileFails pins the decision that one bad file fails the
// load rather than being silently skipped.
func TestLoaderMalformedFileFails(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "Good.bru", "vars {\n  a: 1\n}\n")
	writeEnv(t, dir, "Bad.bru", "this is not a bru environment {\n  unclosed: [\n")

	if _, err := (Loader{Format: model.FormatBru, Root: dir}).Load(); err == nil {
		t.Fatal("expected an error for a malformed file")
	}
}

// TestLoaderSaveRoundTrip writes and re-reads an environment.
func TestLoaderSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	loader := Loader{Format: model.FormatBru, Root: dir}

	env := env("Staging", "", map[string]any{"host": "https://staging.example.com"})
	path, err := loader.Save(env)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if filepath.Base(path) != "Staging.bru" {
		t.Errorf("wrote %q", path)
	}

	got, err := loader.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Staging" {
		t.Fatalf("reloaded %+v", got)
	}
	if v, ok := got[0].VariableValue("host"); !ok || v != "https://staging.example.com" {
		t.Errorf("host = %v, %v", v, ok)
	}
}

// TestSaveSanitisesFilename keeps a display name from escaping the directory.
func TestSaveSanitisesFilename(t *testing.T) {
	dir := t.TempDir()
	loader := Loader{Format: model.FormatBru, Root: dir}

	path, err := loader.Save(env("Prod/US", "", map[string]any{"a": 1}))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// The file must live directly in environments/, not in a subdirectory.
	if got, want := filepath.Dir(path), filepath.Join(dir, DirName); got != want {
		t.Errorf("wrote to %q, want %q", got, want)
	}
	if strings.ContainsRune(filepath.Base(path), '/') {
		t.Errorf("filename %q still contains a separator", filepath.Base(path))
	}
}

// TestSaveRequiresName refuses an environment that cannot be named.
func TestSaveRequiresName(t *testing.T) {
	loader := Loader{Format: model.FormatBru, Root: t.TempDir()}
	if _, err := loader.Save(&model.Environment{Name: "   "}); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := loader.Save(nil); err == nil {
		t.Fatal("expected an error for a nil environment")
	}
}

// TestRemoveIsConfinedToEnvironmentsDir stops a crafted filename escaping.
func TestRemoveIsConfinedToEnvironmentsDir(t *testing.T) {
	dir := t.TempDir()
	// A file outside the environments directory that must survive.
	outside := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := Loader{Format: model.FormatBru, Root: dir}
	if err := loader.Remove(&model.Environment{Filename: "../../keep.txt"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("a traversal filename deleted a file outside the collection")
	}
}

// TestRemoveMissingFileIsNotAnError makes deletion idempotent.
func TestRemoveMissingFileIsNotAnError(t *testing.T) {
	loader := Loader{Format: model.FormatBru, Root: t.TempDir()}
	if err := loader.Remove(&model.Environment{Filename: "gone.bru"}); err != nil {
		t.Errorf("err = %v", err)
	}
	if err := loader.Remove(nil); err != nil {
		t.Errorf("nil err = %v", err)
	}
}

// TestLoadAndResolve covers the combined path.
func TestLoadAndResolve(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "Parent.bru", "vars {\n  host: parent\n}\n")
	writeEnv(t, dir, "Child.bru", "vars {\n  extra: child-value\n}\n")

	loader := Loader{Format: model.FormatBru, Root: dir}

	// No uid selected: environments load, nothing resolves.
	all, res, err := loader.LoadAndResolve("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(all) != 2 || res != nil {
		t.Errorf("all = %d, res = %v", len(all), res)
	}

	// A uid that exists resolves.
	uid := ""
	for _, e := range all {
		if e.Name == "Child" {
			uid = e.UID
		}
	}
	if uid == "" {
		t.Skip("the codec did not assign a UID")
	}
	_, res, err = loader.LoadAndResolve(uid)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res == nil || res.Environment.Name != "Child" {
		t.Errorf("res = %+v", res)
	}
}

// TestReadCappedRejectsOversizedFile guards the size limit.
func TestReadCappedRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.bru")
	big := make([]byte, MaxFileBytes+10)
	if err := os.WriteFile(path, big, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readCapped(path); err == nil {
		t.Fatal("expected a size error")
	}
}
