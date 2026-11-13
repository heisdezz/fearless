package dialogs

import (
	"context"
	"testing"
)

// TestNewRequestModalCollectsValues covers the dialog's data: the type, name,
// method and URL reach the handler with sensible defaults.
func TestNewRequestModalCollectsValues(t *testing.T) {
	q := NewQueue()
	var got NewRequestValues
	var called bool
	w, err := q.NewRequestModal(func(_ context.Context, v NewRequestValues) error {
		called = true
		got = v
		return nil
	})
	if err != nil {
		t.Fatalf("NewRequestModal: %v", err)
	}
	if q.Kind() != KindNewRequest {
		t.Fatalf("kind = %s, want new-request", q.Kind())
	}

	// The dialog opens on HTTP / GET.
	if q.Value("kind") != string(RequestKindHTTP) {
		t.Errorf("default kind = %q, want http", q.Value("kind"))
	}
	if q.Value("method") != "GET" {
		t.Errorf("default method = %q, want GET", q.Value("method"))
	}

	q.SetValue("kind", string(RequestKindGraphQL))
	q.SetValue("name", "List Users")
	q.SetValue("method", "POST")
	q.SetValue("url", "https://api.test/graphql")
	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}

	if !called {
		t.Fatal("handler was not called")
	}
	if got.Kind != RequestKindGraphQL {
		t.Errorf("kind = %q", got.Kind)
	}
	if got.Name != "List Users" {
		t.Errorf("name = %q", got.Name)
	}
	if got.Method != "POST" {
		t.Errorf("method = %q", got.Method)
	}
	if got.URL != "https://api.test/graphql" {
		t.Errorf("url = %q", got.URL)
	}
}

// TestNewRequestModalDefaultsName covers an empty name: the dialog supplies one
// rather than creating a nameless request.
func TestNewRequestModalDefaultsName(t *testing.T) {
	q := NewQueue()
	var got NewRequestValues
	w, err := q.NewRequestModal(func(_ context.Context, v NewRequestValues) error {
		got = v
		return nil
	})
	if err != nil {
		t.Fatalf("NewRequestModal: %v", err)
	}
	q.OK()
	if _, err := w.Result(); err != nil {
		t.Fatalf("result: %v", err)
	}
	if got.Name != "Untitled Request" {
		t.Errorf("name = %q, want the default", got.Name)
	}
	if got.Method != "GET" || got.Kind != RequestKindHTTP {
		t.Errorf("defaults = %+v", got)
	}
}
