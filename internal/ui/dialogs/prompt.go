package dialogs

import (
	"context"
	"fmt"
	"strings"

	"fearless/internal/core/vars"
)

// PromptValues is the resolution of a request's `{{?label}}` variables.
//
// Bruno asks for these immediately before sending, one dialog for the whole
// batch, then merges the answers into the prompt scope, which sits above runtime
// and below process in the precedence order (internal/core/vars/scopes.go).
type PromptValues struct {
	// Labels are the variables asked for, in first-appearance order.
	Labels []string
	// Values maps each label to the text the user typed.
	Values map[string]string
}

// Values returns the map SendRequestCmd.PromptVars expects. It is nil when
// there was nothing to ask, which the send path treats as "no prompt scope".
func (p PromptValues) Scope() map[string]string {
	if len(p.Labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(p.Values))
	for k, v := range p.Values {
		out[k] = v
	}
	return out
}

// Empty reports that no variable needed asking.
func (p PromptValues) Empty() bool { return len(p.Labels) == 0 }

// PromptLabels collects the prompt variables across the request's fields.
//
// It walks only the parts of the request that are interpolated at send time, so
// a `{{?name}}` sitting in a disabled header does not pop a dialog. The order is
// stable and duplicate-free: vars.ExtractPromptVariables already dedupes, and
// requests are walked in a fixed order.
//
// Source: bruno-app/src/utils/collections/index.js, promptVariables
func PromptLabels(r *requestView) []string {
	if r == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(labels []string) {
		for _, l := range labels {
			if seen[l] {
				continue
			}
			seen[l] = true
			out = append(out, l)
		}
	}

	add(vars.ExtractPromptVariablesFromString(r.URL))
	add(vars.ExtractPromptVariablesFromString(r.Body))
	for _, kv := range r.Parts {
		if !kv.Enabled {
			continue
		}
		add(vars.ExtractPromptVariablesFromString(kv.Name))
		add(vars.ExtractPromptVariablesFromString(kv.Value))
	}
	return out
}

// AskPrompt opens the prompt dialog for labels and blocks until the user sends
// or cancels. It returns ErrSuperseded for a cancel, so the caller skips the
// send rather than sending with an empty prompt scope.
func (q *Queue) AskPrompt(ctx context.Context, labels []string) (map[string]string, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	w, err := q.Prompt("Variables", labels, nil)
	if err != nil {
		return nil, err
	}

	// The shell must close the dialog on cancellation of the caller, not only on
	// a click, or a cancelled send would leave a question on screen.
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			q.Cancel()
		case <-stop:
		}
	}()

	values, err := w.Result()
	close(stop)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// ValidatePrompt reports the first required label left blank. OK should already
// be disabled, so this is a belt-and-braces check for callers that bypass the
// view.
func ValidatePrompt(p PromptValues) error {
	for _, l := range p.Labels {
		if strings.TrimSpace(p.Values[l]) == "" {
			return fmt.Errorf("dialogs: %q has no value", l)
		}
	}
	return nil
}

// SetWarnings records adaptations to show inside the active dialog, such as the
// fields a converter dropped. The dialog stays open, so this is advisory text,
// not a failure.
func (q *Queue) SetWarnings(ws []string) {
	q.mu.Lock()
	q.warnings = append([]string(nil), ws...)
	q.mu.Unlock()
	q.notify()
}

// Warnings returns the recorded list.
func (q *Queue) Warnings() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]string(nil), q.warnings...)
}

// SetTargets records the collections a save dialog may choose from. The labels
// are what the picker shows; SetValue is still keyed by label, and the caller
// maps a label back to a UID.
func (q *Queue) SetTargets(targets []SaveTarget) {
	q.mu.Lock()
	q.targets = append([]SaveTarget(nil), targets...)
	q.mu.Unlock()
	q.notify()
}

// TargetLabels returns the picker options.
func (q *Queue) TargetLabels() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]string, 0, len(q.targets))
	for _, t := range q.targets {
		out = append(out, t.Name)
	}
	return out
}

// TargetUID maps a picked label back to its collection UID. It returns the input
// unchanged when the label is unknown, so a user who types a collection name
// that does not exist gets a clear error from the save command rather than a
// silent no-op.
func (q *Queue) TargetUID(label string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, t := range q.targets {
		if t.Name == label || t.UID == label {
			return t.UID
		}
	}
	return label
}

// requestView is the slice of a request that the prompt scan needs. It is
// declared here, rather than importing model, so the scan can be tested with a
// literal and so the dependency stays one-directional.
type requestView struct {
	URL   string
	Body  string
	Parts []requestPart
}

// requestPart is one key/value pair scanned for prompt variables.
type requestPart struct {
	Name    string
	Value   string
	Enabled bool
}

// RequestView builds the scan input from its parts.
func RequestView(url, body string, parts []requestPart) *requestView {
	return &requestView{URL: url, Body: body, Parts: parts}
}

// RequestPart is the exported part type, so callers can build a slice without
// naming an unexported type.
type RequestPart = requestPart
