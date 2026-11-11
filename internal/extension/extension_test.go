package extension

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
)

type failHook struct{}

func (f *failHook) Name() string { return "fail" }
func (f *failHook) BeforeRequest(ctx context.Context, req *http.Request) error {
	return errors.New("boom")
}
func (f *failHook) AfterResponse(ctx context.Context, req *http.Request, res *http.Response) error {
	return errors.New("boom-after")
}

func TestHookRegistry(t *testing.T) {
	r := NewHookRegistry()

	hdrHook := NewHeaderInjectionHook("auth-hdr", map[string]string{"Authorization": "Bearer secret"})
	timingHook := NewTimingHook("timing")

	r.Register(hdrHook)
	r.Register(timingHook)

	if len(r.All()) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(r.All()))
	}
	if r.Get("auth-hdr") != hdrHook {
		t.Error("Get hook mismatch")
	}

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "https://api.example.com", nil)
	if err := r.RunBeforeRequest(context.Background(), req); err != nil {
		t.Fatalf("RunBeforeRequest failed: %v", err)
	}

	if req.Header.Get("Authorization") != "Bearer secret" {
		t.Errorf("header not injected: %s", req.Header.Get("Authorization"))
	}

	res := &http.Response{StatusCode: 200}
	if err := r.RunAfterResponse(context.Background(), req, res); err != nil {
		t.Fatalf("RunAfterResponse failed: %v", err)
	}

	if dur, ok := timingHook.Durations.Load(req); !ok || dur.(time.Duration) < 0 {
		t.Errorf("timing hook did not record duration: %v", dur)
	}

	// Error propagation
	r.Register(&failHook{})
	if err := r.RunBeforeRequest(context.Background(), req); err == nil {
		t.Error("expected error from failHook BeforeRequest")
	}
	if err := r.RunAfterResponse(context.Background(), req, res); err == nil {
		t.Error("expected error from failHook AfterResponse")
	}

	// Unregister
	r.Unregister("fail")
	if r.Get("fail") != nil {
		t.Error("expected fail hook to be unregistered")
	}
}

type sampleXML struct {
	XMLName xml.Name `xml:"user"`
	Name    string   `xml:"name"`
	Age     int      `xml:"age"`
}

func TestBodyEncoderRegistry(t *testing.T) {
	r := NewBodyEncoderRegistry()

	if len(r.All()) != 3 {
		t.Fatalf("expected 3 default encoders, got %d", len(r.All()))
	}

	// JSON
	jsonEnc := r.Get("json")
	if jsonEnc == nil || jsonEnc.ContentType() != "application/json" {
		t.Fatalf("json encoder mismatch")
	}
	data, err := jsonEnc.Encode(map[string]string{"foo": "bar"})
	if err != nil || !strings.Contains(string(data), `"foo":"bar"`) {
		t.Errorf("json encode mismatch: %s, %v", data, err)
	}
	var target map[string]string
	if err := jsonEnc.Decode(data, &target); err != nil || target["foo"] != "bar" {
		t.Errorf("json decode mismatch: %v, %v", target, err)
	}

	// Form-URL-Encoded
	formEnc := r.ForContentType("application/x-www-form-urlencoded; charset=utf-8")
	if formEnc == nil {
		t.Fatalf("form-urlencoded encoder not found by content type")
	}
	formData, err := formEnc.Encode(map[string]string{"user": "alice", "active": "true"})
	if err != nil {
		t.Fatalf("form encode failed: %v", err)
	}
	var formTarget map[string]string
	if err := formEnc.Decode(formData, &formTarget); err != nil || formTarget["user"] != "alice" {
		t.Errorf("form decode mismatch: %v, %v", formTarget, err)
	}
	var urlValuesTarget url.Values
	if err := formEnc.Decode(formData, &urlValuesTarget); err != nil || urlValuesTarget.Get("user") != "alice" {
		t.Errorf("url.Values decode mismatch: %v, %v", urlValuesTarget, err)
	}

	// XML
	xmlEnc := r.ForContentType("application/xml")
	if xmlEnc == nil {
		t.Fatalf("xml encoder not found by content type")
	}
	xmlData, err := xmlEnc.Encode(sampleXML{Name: "Bob", Age: 30})
	if err != nil {
		t.Fatalf("xml encode failed: %v", err)
	}
	var xmlTarget sampleXML
	if err := xmlEnc.Decode(xmlData, &xmlTarget); err != nil || xmlTarget.Name != "Bob" || xmlTarget.Age != 30 {
		t.Errorf("xml decode mismatch: %v, %v", xmlTarget, err)
	}
}

func TestVariableResolverRegistry(t *testing.T) {
	r := NewVariableResolverRegistry()

	// Dynamic variables
	val, ok, err := r.Resolve(context.Background(), "$guid")
	if !ok || err != nil || len(val.(string)) != 36 {
		t.Errorf("resolve $guid failed: %v, %v, %v", val, ok, err)
	}

	val, ok, err = r.Resolve(context.Background(), "$timestamp")
	if !ok || err != nil || val.(string) == "" {
		t.Errorf("resolve $timestamp failed: %v, %v, %v", val, ok, err)
	}

	val, ok, err = r.Resolve(context.Background(), "$isoTimestamp")
	if !ok || err != nil || !strings.Contains(val.(string), "T") {
		t.Errorf("resolve $isoTimestamp failed: %v, %v, %v", val, ok, err)
	}

	val, ok, err = r.Resolve(context.Background(), "$randomInt")
	if !ok || err != nil || val.(int) <= 0 {
		t.Errorf("resolve $randomInt failed: %v, %v, %v", val, ok, err)
	}

	// Env variable
	os.Setenv("FEARLESS_TEST_VAR", "hello_world")
	defer os.Unsetenv("FEARLESS_TEST_VAR")

	val, ok, err = r.Resolve(context.Background(), "env.FEARLESS_TEST_VAR")
	if !ok || err != nil || val != "hello_world" {
		t.Errorf("resolve env.FEARLESS_TEST_VAR failed: %v, %v, %v", val, ok, err)
	}

	val, ok, err = r.Resolve(context.Background(), "process.env.FEARLESS_TEST_VAR")
	if !ok || err != nil || val != "hello_world" {
		t.Errorf("resolve process.env.FEARLESS_TEST_VAR failed: %v, %v, %v", val, ok, err)
	}

	// Custom Map Resolver
	mapResolver := NewMapVarResolver("custom", map[string]any{"key1": 42})
	r.Register(mapResolver)
	val, ok, err = r.Resolve(context.Background(), "custom.key1")
	if !ok || err != nil || val != 42 {
		t.Errorf("resolve custom.key1 failed: %v, %v, %v", val, ok, err)
	}

	// Unregister
	r.Unregister("custom")
	if r.Get("custom") != nil {
		t.Error("expected custom resolver to be unregistered")
	}
}

func TestConverterRegistry(t *testing.T) {
	r := NewConverterRegistry()

	if len(r.AllImporters()) != 1 || len(r.AllExporters()) != 1 {
		t.Fatalf("expected 1 default importer/exporter, got %d, %d", len(r.AllImporters()), len(r.AllExporters()))
	}

	imp := r.GetImporter("json")
	exp := r.GetExporter("json")
	if imp == nil || exp == nil {
		t.Fatalf("json importer/exporter not found")
	}

	col := &model.Collection{
		UID:  "col-ext-test",
		Name: "Ext Collection",
		Config: model.CollectionConfig{
			Name:    "Ext Collection",
			Version: "1",
		},
	}

	// Export
	exported, err := exp.Export(context.Background(), col)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Detect
	detected := r.DetectImporter(exported)
	if detected == nil || detected.Format() != "json" {
		t.Fatalf("detect importer failed: %v", detected)
	}

	// Import
	imported, err := detected.Import(context.Background(), exported)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported.Name != "Ext Collection" {
		t.Errorf("imported collection mismatch: %s", imported.Name)
	}
}
