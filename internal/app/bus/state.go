package bus

import (
	"errors"

	"fearless/internal/core/model"
)

// errClosed is returned by Apply after the store has been closed.
var errClosed = errors.New("bus: store is closed")

// IsClosed reports whether err means the store was closed.
func IsClosed(err error) bool { return errors.Is(err, errClosed) }

// cloneState deep-copies a workspace state.
//
// Deep copying is what makes snapshots immutable. Every mutable field that a
// presenter could reach must be copied here; adding a new field to
// WorkspaceState without extending this function would silently reintroduce
// aliasing, so state_clone_test.go asserts that mutating a snapshot never
// affects the store.
func cloneState(in *model.WorkspaceState) *model.WorkspaceState {
	if in == nil {
		return nil
	}
	out := *in

	if in.Collections != nil {
		out.Collections = make([]*model.Collection, len(in.Collections))
		for i, c := range in.Collections {
			out.Collections[i] = cloneCollection(c)
		}
	}
	if in.Tabs != nil {
		out.Tabs = make([]*model.Tab, len(in.Tabs))
		for i, t := range in.Tabs {
			if t == nil {
				continue
			}
			tab := *t
			out.Tabs[i] = &tab
		}
	}
	if in.Responses != nil {
		out.Responses = make(map[string]*model.Result, len(in.Responses))
		for k, v := range in.Responses {
			out.Responses[k] = cloneResult(v)
		}
	}
	if in.Pending != nil {
		out.Pending = make(map[string]bool, len(in.Pending))
		for k, v := range in.Pending {
			out.Pending[k] = v
		}
	}
	return &out
}

// cloneCollection copies a collection and its whole item tree.
func cloneCollection(in *model.Collection) *model.Collection {
	if in == nil {
		return nil
	}
	out := *in
	out.Items = cloneItems(in.Items)
	if in.Environments != nil {
		out.Environments = make([]*model.Environment, len(in.Environments))
		for i, e := range in.Environments {
			out.Environments[i] = cloneEnvironment(e)
		}
	}
	return &out
}

// cloneItems copies an item slice depth-first.
func cloneItems(in []*model.Item) []*model.Item {
	if in == nil {
		return nil
	}
	out := make([]*model.Item, len(in))
	for i, it := range in {
		out[i] = cloneItem(it)
	}
	return out
}

// cloneItem copies one item, its requests and its subtree.
func cloneItem(in *model.Item) *model.Item {
	if in == nil {
		return nil
	}
	out := *in
	out.Request = in.Request.Clone()
	out.Draft = in.Draft.Clone()
	out.Children = cloneItems(in.Children)
	if in.Headers != nil {
		out.Headers = make([]model.KeyValue, len(in.Headers))
		copy(out.Headers, in.Headers)
	}
	if in.Vars != nil {
		out.Vars = make([]model.Variable, len(in.Vars))
		copy(out.Vars, in.Vars)
	}
	if in.Tags != nil {
		out.Tags = append([]string(nil), in.Tags...)
	}
	if in.Script != nil {
		cp := *in.Script
		out.Script = &cp
	}
	if in.Examples != nil {
		out.Examples = make([]*model.Example, len(in.Examples))
		for i, ex := range in.Examples {
			if ex == nil {
				continue
			}
			c := *ex
			c.Request = ex.Request.Clone()
			c.Response = cloneResult(ex.Response)
			out.Examples[i] = &c
		}
	}
	return &out
}

// cloneEnvironment copies an environment and its variables.
func cloneEnvironment(in *model.Environment) *model.Environment {
	if in == nil {
		return nil
	}
	out := *in
	if in.Variables != nil {
		out.Variables = make([]model.Variable, len(in.Variables))
		copy(out.Variables, in.Variables)
	}
	if in.Secrets != nil {
		out.Secrets = append([]string(nil), in.Secrets...)
	}
	return &out
}

// cloneResult copies a result.
//
// The body is a BodyRef. Its Inline bytes are copied because a caller could
// otherwise mutate the store's copy of a small body; a spilled body is
// immutable on disk and is shared by reference.
func cloneResult(in *model.Result) *model.Result {
	if in == nil {
		return nil
	}
	out := *in
	if in.Body.Inline != nil {
		out.Body.Inline = append([]byte(nil), in.Body.Inline...)
	}
	if in.Headers != nil {
		out.Headers = make([]model.KeyValue, len(in.Headers))
		copy(out.Headers, in.Headers)
	}
	if in.Timeline != nil {
		out.Timeline = make(model.Timeline, len(in.Timeline))
		copy(out.Timeline, in.Timeline)
	}
	if in.Redirects != nil {
		out.Redirects = append([]model.Redirect(nil), in.Redirects...)
	}
	if in.Sent.Headers != nil {
		out.Sent.Headers = make([]model.KeyValue, len(in.Sent.Headers))
		copy(out.Sent.Headers, in.Sent.Headers)
	}
	if in.Sent.Body != nil {
		out.Sent.Body = append([]byte(nil), in.Sent.Body...)
	}
	return &out
}
