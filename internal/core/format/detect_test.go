package format_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// stubCodec is a test implementation of format.Codec.
type stubCodec struct {
	name              model.CollectionFormat
	configFilename    string
	parsedRequest     *model.Request
	writtenRequest    []byte
	parsedCollection  *model.CollectionConfig
	writtenCollection []byte
	parsedFolder      *model.Item
	writtenFolder     []byte
	parsedEnv         *model.Environment
	writtenEnv        []byte
	detectErr         error
}

func (s *stubCodec) Name() model.CollectionFormat {
	return s.name
}

func (s *stubCodec) Detect(dir string) (bool, error) {
	if s.detectErr != nil {
		return false, s.detectErr
	}
	target := filepath.Join(dir, s.configFilename)
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

func (s *stubCodec) ParseRequest(data []byte) (*model.Request, error) {
	return s.parsedRequest, nil
}

func (s *stubCodec) WriteRequest(req *model.Request) ([]byte, error) {
	return s.writtenRequest, nil
}

func (s *stubCodec) ParseCollection(data []byte) (*model.CollectionConfig, error) {
	return s.parsedCollection, nil
}

func (s *stubCodec) WriteCollection(cfg *model.CollectionConfig) ([]byte, error) {
	return s.writtenCollection, nil
}

func (s *stubCodec) ParseFolder(data []byte) (*model.Item, error) {
	return s.parsedFolder, nil
}

func (s *stubCodec) WriteFolder(folder *model.Item) ([]byte, error) {
	return s.writtenFolder, nil
}

func (s *stubCodec) ParseEnvironment(data []byte) (*model.Environment, error) {
	return s.parsedEnv, nil
}

func (s *stubCodec) WriteEnvironment(env *model.Environment) ([]byte, error) {
	return s.writtenEnv, nil
}

func setupTestCodecs(t *testing.T) (*stubCodec, *stubCodec) {
	t.Helper()
	format.Reset()
	t.Cleanup(format.Reset)

	bruCodec := &stubCodec{
		name:           model.FormatBru,
		configFilename: format.BrunoJSONFilename,
	}
	ymlCodec := &stubCodec{
		name:           model.FormatYML,
		configFilename: format.OpenCollectionYMLFilename,
	}

	format.Register(bruCodec)
	format.Register(ymlCodec)

	return bruCodec, ymlCodec
}

func TestRegistryBasics(t *testing.T) {
	bruCodec, ymlCodec := setupTestCodecs(t)

	// Get
	c, ok := format.Get(model.FormatBru)
	if !ok || c != bruCodec {
		t.Fatalf("expected bruCodec, got %v (ok=%v)", c, ok)
	}
	c, ok = format.Get(model.FormatYML)
	if !ok || c != ymlCodec {
		t.Fatalf("expected ymlCodec, got %v (ok=%v)", c, ok)
	}
	_, ok = format.Get("nonexistent")
	if ok {
		t.Fatalf("expected ok=false for nonexistent format")
	}

	// For
	cf, err := format.For(model.FormatBru)
	if err != nil || cf != bruCodec {
		t.Fatalf("expected bruCodec, got %v, err=%v", cf, err)
	}
	_, err = format.For("nonexistent")
	if !errors.Is(err, format.ErrUnknownFormat) {
		t.Fatalf("expected ErrUnknownFormat, got %v", err)
	}

	// Registered formats list
	names := format.Registered()
	if len(names) != 2 || names[0] != model.FormatBru || names[1] != model.FormatYML {
		t.Fatalf("expected [bru, yml], got %v", names)
	}
}

func TestRegisterPanics(t *testing.T) {
	setupTestCodecs(t)

	// nil codec
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on registering nil Codec")
			}
		}()
		format.Register(nil)
	}()

	// empty name
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on registering Codec with empty name")
			}
		}()
		format.Register(&stubCodec{name: ""})
	}()

	// duplicate registration
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on duplicate registration")
			}
		}()
		format.Register(&stubCodec{name: model.FormatBru})
	}()
}

func TestDetectBruDialect(t *testing.T) {
	setupTestCodecs(t)
	dir := t.TempDir()

	// Write bruno.json
	brunoPath := filepath.Join(dir, format.BrunoJSONFilename)
	if err := os.WriteFile(brunoPath, []byte(`{"version":"1","name":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	codec, err := format.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if codec.Name() != model.FormatBru {
		t.Fatalf("expected format %q, got %q", model.FormatBru, codec.Name())
	}

	fmtName, err := format.DetectFormat(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmtName != model.FormatBru {
		t.Fatalf("expected format %q, got %q", model.FormatBru, fmtName)
	}
}

func TestDetectYMLDialect(t *testing.T) {
	setupTestCodecs(t)
	dir := t.TempDir()

	// Write opencollection.yml
	ymlPath := filepath.Join(dir, format.OpenCollectionYMLFilename)
	if err := os.WriteFile(ymlPath, []byte("info:\n  name: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	codec, err := format.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if codec.Name() != model.FormatYML {
		t.Fatalf("expected format %q, got %q", model.FormatYML, codec.Name())
	}

	fmtName, err := format.DetectFormat(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmtName != model.FormatYML {
		t.Fatalf("expected format %q, got %q", model.FormatYML, fmtName)
	}
}

func TestDetectDeterministicWhenBothExist(t *testing.T) {
	setupTestCodecs(t)
	dir := t.TempDir()

	// Write BOTH bruno.json and opencollection.yml
	if err := os.WriteFile(filepath.Join(dir, format.BrunoJSONFilename), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, format.OpenCollectionYMLFilename), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Invariant: when both are present, detection must be strictly deterministic
	// across repeated invocations. FormatBru has priority over FormatYML.
	for i := 0; i < 50; i++ {
		codec, err := format.Detect(dir)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if codec.Name() != model.FormatBru {
			t.Fatalf("iteration %d: expected FormatBru, got %v", i, codec.Name())
		}
	}
}

func TestDetectNoCollection(t *testing.T) {
	setupTestCodecs(t)
	dir := t.TempDir()

	_, err := format.Detect(dir)
	if !errors.Is(err, format.ErrNoCollection) {
		t.Fatalf("expected ErrNoCollection, got %v", err)
	}

	_, err = format.DetectFormat(dir)
	if !errors.Is(err, format.ErrNoCollection) {
		t.Fatalf("expected ErrNoCollection, got %v", err)
	}
}

func TestDetectErrorPropagation(t *testing.T) {
	format.Reset()
	t.Cleanup(format.Reset)

	failCodec := &stubCodec{
		name:      model.FormatBru,
		detectErr: errors.New("disk failure"),
	}
	format.Register(failCodec)

	_, err := format.Detect(t.TempDir())
	if err == nil || !errors.Is(err, failCodec.detectErr) && err.Error() != "detect format \"bru\": disk failure" {
		t.Fatalf("expected error containing 'disk failure', got %v", err)
	}
}

func TestExtendWithNewCodecRequiresNoPackageChange(t *testing.T) {
	setupTestCodecs(t)

	// Demonstrate acceptance criterion 3:
	// "Adding a codec requires no change to this package beyond a Register call by the implementation."
	customCodec := &stubCodec{
		name:           "custom-format",
		configFilename: "custom.json",
	}
	format.Register(customCodec)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "custom.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	detected, err := format.Detect(dir)
	if err != nil {
		t.Fatalf("failed to detect new custom codec: %v", err)
	}
	if detected.Name() != "custom-format" {
		t.Fatalf("expected custom-format, got %s", detected.Name())
	}
}

func TestFileExistsAndDetectFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")

	if format.FileExists(file) {
		t.Fatalf("expected file to not exist yet")
	}
	if format.DetectFile(dir, "test.txt") {
		t.Fatalf("expected detect to be false")
	}

	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !format.FileExists(file) {
		t.Fatalf("expected file to exist")
	}
	if !format.DetectFile(dir, "test.txt") {
		t.Fatalf("expected detect to be true")
	}

	// Directories are not regular files
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if format.FileExists(subDir) {
		t.Fatalf("directory should not count as file")
	}
}

func TestConcurrentAccess(t *testing.T) {
	setupTestCodecs(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, format.BrunoJSONFilename), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = format.Get(model.FormatBru)
				_, _ = format.For(model.FormatYML)
				_ = format.Registered()
				_, _ = format.Detect(dir)
			}
		}()
	}
	wg.Wait()
}
