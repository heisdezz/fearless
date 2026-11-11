package environment

import (
	"errors"
	"testing"

	"fearless/internal/core/model"
)

// env builds an environment with the given variables, all enabled.
func env(name, extends string, vars map[string]any) *model.Environment {
	e := &model.Environment{UID: "uid-" + name, Name: name, Extends: extends}
	for k, v := range vars {
		e.Variables = append(e.Variables, model.Variable{
			Name:    strp(k),
			Value:   v,
			Enabled: true,
		})
	}
	return e
}

func strp(s string) *string { return &s }

// TestResolveChainOrder covers the basic inheritance: the nearest ancestor wins.
func TestResolveChainOrder(t *testing.T) {
	root := env("root", "", map[string]any{"host": "root-host", "a": 1})
	mid := env("mid", "root", map[string]any{"host": "mid-host", "b": 2})
	leaf := env("leaf", "mid", map[string]any{"c": 3})

	res, err := Resolve([]*model.Environment{root, mid, leaf}, leaf)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	values := res.Values()
	if values["host"] != "mid-host" {
		t.Errorf("host = %v, want the nearest ancestor's value", values["host"])
	}
	if values["a"] != 1 || values["b"] != 2 {
		t.Errorf("ancestor values missing: %v", values)
	}
	if _, ok := values["c"]; ok {
		t.Error("the target's own variables are not inherited")
	}

	wantChain := []string{"root", "mid", "leaf"}
	if len(res.Chain) != len(wantChain) {
		t.Fatalf("chain = %v, want %v", res.Chain, wantChain)
	}
	for i := range wantChain {
		if res.Chain[i] != wantChain[i] {
			t.Fatalf("chain = %v, want %v", res.Chain, wantChain)
		}
	}
}

// TestResolveChildDeletesInherited is the precedence rule that catches people
// out: defining a name in the child removes the inherited entry entirely.
func TestResolveChildDeletesInherited(t *testing.T) {
	parent := env("parent", "", map[string]any{"host": "parent-host"})
	child := env("child", "parent", map[string]any{"host": "child-host"})

	res, err := Resolve([]*model.Environment{parent, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(res.Inherited) != 0 {
		t.Errorf("inherited = %+v, want the shadowed entry removed", res.Inherited)
	}
	if v, ok := child.VariableValue("host"); !ok || v != "child-host" {
		t.Errorf("child value = %v, %v", v, ok)
	}
}

// TestResolveDisabledParentVariableIsSkipped covers `enabled: false`.
func TestResolveDisabledParentVariableIsSkipped(t *testing.T) {
	parent := &model.Environment{
		UID:  "p",
		Name: "parent",
		Variables: []model.Variable{
			{Name: strp("on"), Value: "yes", Enabled: true},
			{Name: strp("off"), Value: "no", Enabled: false},
		},
	}
	child := env("child", "parent", nil)

	res, err := Resolve([]*model.Environment{parent, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	values := res.Values()
	if _, ok := values["off"]; ok {
		t.Error("a disabled variable must not be inherited")
	}
	if values["on"] != "yes" {
		t.Errorf("on = %v", values["on"])
	}
}

// TestResolveProvenance records which ancestor supplied a value.
func TestResolveProvenance(t *testing.T) {
	parent := env("parent", "", map[string]any{"shared": "p", "onlyParent": "p"})
	child := env("child", "parent", nil)

	res, err := Resolve([]*model.Environment{parent, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, v := range res.Inherited {
		if v.From == nil {
			t.Fatalf("%s has no provenance", v.Name())
		}
		if v.From.Name != "parent" || v.From.UID != "uid-parent" {
			t.Errorf("%s came from %+v", v.Name(), v.From)
		}
	}
}

// TestResolveCycleIsRejected covers a two-node and a three-node cycle.
func TestResolveCycleIsRejected(t *testing.T) {
	a := env("a", "b", nil)
	b := env("b", "a", nil)

	_, err := Resolve([]*model.Environment{a, b}, a)
	if !errors.Is(err, ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}

	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("err type = %T", err)
	}
	// The path begins and ends with the same name.
	if len(cycleErr.Path) < 2 || cycleErr.Path[0] != cycleErr.Path[len(cycleErr.Path)-1] {
		t.Errorf("cycle path = %v", cycleErr.Path)
	}

	// A self-reference is the degenerate case.
	self := env("s", "s", nil)
	if _, err := Resolve([]*model.Environment{self}, self); !errors.Is(err, ErrCycle) {
		t.Errorf("self-cycle err = %v", err)
	}

	// Three nodes closing the loop.
	x := env("x", "y", nil)
	y := env("y", "z", nil)
	z := env("z", "x", nil)
	if _, err := Resolve([]*model.Environment{x, y, z}, x); !errors.Is(err, ErrCycle) {
		t.Errorf("three-node cycle err = %v", err)
	}
}

// TestResolveMissingParentIsRejected reports an unresolvable reference.
func TestResolveMissingParentIsRejected(t *testing.T) {
	child := env("child", "nonexistent", nil)

	_, err := Resolve([]*model.Environment{child}, child)
	if !errors.Is(err, ErrMissingParent) {
		t.Fatalf("err = %v, want ErrMissingParent", err)
	}

	var missing *MissingParentError
	if !errors.As(err, &missing) {
		t.Fatalf("err type = %T", err)
	}
	if missing.Environment != "child" || missing.Parent != "nonexistent" {
		t.Errorf("error = %+v", missing)
	}
}

// TestResolveNilTarget is a programming-error guard.
func TestResolveNilTarget(t *testing.T) {
	if _, err := Resolve(nil, nil); err == nil {
		t.Fatal("expected an error")
	}
}

// TestResolveSecretsKeptSeparate verifies that secrets and plaintext are tracked
// in separate maps, as Bruno does.
func TestResolveSecretsKeptSeparate(t *testing.T) {
	parent := &model.Environment{
		UID:  "p",
		Name: "parent",
		Variables: []model.Variable{
			{Name: strp("token"), Value: "", Enabled: true, Secret: true},
			{Name: strp("host"), Value: "p-host", Enabled: true},
		},
	}
	child := env("child", "parent", nil)

	res, err := Resolve([]*model.Environment{parent, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Secrets do not leak into the value map, because the file holds no value.
	if _, ok := res.Values()["token"]; ok {
		t.Error("a secret must not appear in Values")
	}
	names := res.SecretNames()
	if len(names) != 1 || names[0] != "token" {
		t.Errorf("secret names = %v", names)
	}
}

// TestSecretNamesAreDeduplicatedAndOrdered covers the target's own declarations.
func TestSecretNamesAreDeduplicatedAndOrdered(t *testing.T) {
	parent := &model.Environment{
		UID:       "p",
		Name:      "parent",
		Variables: []model.Variable{{Name: strp("shared"), Enabled: true, Secret: true}},
	}
	child := &model.Environment{
		UID:     "c",
		Name:    "child",
		Extends: "parent",
		// The target may declare secrets in `vars:secret [ … ]` without a value.
		Secrets: []string{"shared", "own", "own"},
	}

	res, err := Resolve([]*model.Environment{parent, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := res.SecretNames()
	want := []string{"shared", "own"}
	if len(got) != len(want) {
		t.Fatalf("secret names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("secret names = %v, want %v", got, want)
		}
	}
}

// TestInheritableExcludesDescendants covers the picker's cycle prevention.
func TestInheritableExcludesDescendants(t *testing.T) {
	root := env("root", "", nil)
	mid := env("mid", "root", nil)
	leaf := env("leaf", "mid", nil)
	other := env("other", "", nil)

	got := Inheritable([]*model.Environment{root, mid, leaf, other}, root)
	names := make(map[string]bool)
	for _, e := range got {
		names[e.Name] = true
	}

	if names["root"] {
		t.Error("an environment cannot extend itself")
	}
	if names["mid"] || names["leaf"] {
		t.Error("descendants must be excluded: they would close a cycle")
	}
	if !names["other"] {
		t.Error("an unrelated environment is inheritable")
	}
}

// TestInheritableNilTargetListsAll covers the create-environment case.
func TestInheritableNilTargetListsAll(t *testing.T) {
	a := env("a", "", nil)
	b := env("b", "", nil)
	if got := Inheritable([]*model.Environment{a, b}, nil); len(got) != 2 {
		t.Errorf("got %d environments, want 2", len(got))
	}
}

// TestSelect covers selection by UID.
func TestSelect(t *testing.T) {
	a := env("a", "", nil)
	b := env("b", "", nil)
	list := []*model.Environment{a, b}

	if got := Select(list, "uid-b"); got != b {
		t.Errorf("selected %v", got)
	}
	// An empty UID means "no environment", not "the first one".
	if got := Select(list, ""); got != nil {
		t.Errorf("empty uid selected %v", got)
	}
	if got := Select(list, "missing"); got != nil {
		t.Errorf("missing uid selected %v", got)
	}
	if got := Select(nil, "uid-a"); got != nil {
		t.Errorf("nil list selected %v", got)
	}
}

// TestResolveDuplicateNameUsesFirstDefinition matches Bruno's Array.find.
func TestResolveDuplicateNameUsesFirstDefinition(t *testing.T) {
	first := &model.Environment{UID: "1", Name: "dup", Variables: []model.Variable{
		{Name: strp("v"), Value: "first", Enabled: true},
	}}
	second := &model.Environment{UID: "2", Name: "dup", Variables: []model.Variable{
		{Name: strp("v"), Value: "second", Enabled: true},
	}}
	child := env("child", "dup", nil)

	res, err := Resolve([]*model.Environment{first, second, child}, child)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got := res.Values()["v"]; got != "first" {
		t.Errorf("v = %v, want the first definition", got)
	}
}
