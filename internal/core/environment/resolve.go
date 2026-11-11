// Package environment loads, resolves and selects Bruno environment files.
//
// It owns the file-system side of environments: finding `environments/*.bru`
// and `environments/*.yml`, parsing them through the format codecs, following
// `extends` chains, and merging declared secrets.
//
// Precedence *between* scopes lives in package vars; this package produces the
// environment map and records where each variable came from, so the UI can show
// provenance.
//
// Source: bruno-common/src/utils/environment-inheritance.ts
// Source: bruno-electron/src/ipc/network/interpolate-vars.js
package environment

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"fearless/internal/core/model"
)

// Errors reported when an `extends` chain cannot be followed.
//
// They are distinct because the UI shows them differently: a missing parent is a
// fixable reference, a cycle is a structural mistake in the files themselves.
var (
	// ErrMissingParent reports an `extends` naming an environment that does not
	// exist in the collection.
	ErrMissingParent = errors.New("environment: inherited environment not found")

	// ErrCycle reports an `extends` chain that closes on itself.
	ErrCycle = errors.New("environment: cyclic environment inheritance")
)

// MissingParentError names the environment that could not be found.
type MissingParentError struct {
	// Environment is the environment whose `extends` could not be resolved.
	Environment string
	// Parent is the name it referred to.
	Parent string
}

func (e *MissingParentError) Error() string {
	return fmt.Sprintf("%s: %q extends %q, which does not exist", ErrMissingParent, e.Environment, e.Parent)
}

func (e *MissingParentError) Unwrap() error { return ErrMissingParent }

// CycleError describes a chain that closes on itself.
type CycleError struct {
	// Path is the cycle, beginning at the first repeated name and ending with it
	// again, so a cycle through a and b reads as [a b a].
	Path []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("%s: %s", ErrCycle, strings.Join(e.Path, " -> "))
}

func (e *CycleError) Unwrap() error { return ErrCycle }

// InheritedFrom records where an inherited variable came from.
//
// It is the Go equivalent of Bruno's `inheritedFrom: { name, uid }`, kept so the
// UI can label an inherited row with its origin.
type InheritedFrom struct {
	Name string
	UID  string
}

// ResolvedVariable is a variable together with its provenance.
type ResolvedVariable struct {
	Variable model.Variable
	// From is nil for the environment's own variables and set for inherited ones.
	From *InheritedFrom
}

// Name returns the variable name.
func (r ResolvedVariable) Name() string { return r.Variable.NameOrEmpty() }

// Secret reports whether the value lives in the secret store rather than on disk.
func (r ResolvedVariable) Secret() bool { return r.Variable.Secret }

// Resolution is the flattened result of following an `extends` chain.
type Resolution struct {
	// Environment is the environment that was resolved.
	Environment *model.Environment
	// Inherited holds enabled variables from ancestors, nearest ancestor last,
	// minus anything a descendant defines. Non-secrets come first, then secrets.
	Inherited []ResolvedVariable
	// Chain lists the names walked, root ancestor first, ending with the target.
	Chain []string
}

// Values returns the non-secret inherited variables as a lookup map.
//
// Secret values are not on disk, so the caller merges them from the secret store
// separately; including their empty placeholders here would shadow a real value
// supplied later.
func (r Resolution) Values() map[string]any {
	out := make(map[string]any, len(r.Inherited))
	for _, v := range r.Inherited {
		if v.Secret() {
			continue
		}
		out[v.Name()] = v.Variable.Value
	}
	return out
}

// SecretNames lists the secret variables declared by the environment and its
// ancestors, without duplicates, in resolution order.
//
// Bruno keeps the names in `vars:secret [ … ]` and the values outside the
// collection, so a file only ever says which names *need* a secret.
func (r Resolution) SecretNames() []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, v := range r.Inherited {
		if v.Secret() {
			add(v.Name())
		}
	}
	if r.Environment != nil {
		for _, name := range r.Environment.Secrets {
			add(name)
		}
	}
	return names
}

// Resolve follows `extends` from target through its ancestors.
//
// The chain is walked parent-ward and each ancestor's enabled variables are
// collected. A variable defined by a descendant removes the ancestor's entry of
// the same name: the child wins, and defining a name deliberately deletes the
// inherited one rather than leaving it to be overridden later.
//
// A missing parent or a cycle is an error, not a partial result. Bruno's renderer
// returns the ancestors it found alongside the problem so the UI can warn
// without blocking a send; here the caller gets both — the error says what went
// wrong, and errors.Is distinguishes the two cases. This is stricter on purpose:
// silently sending with a half-resolved environment is how a request reaches
// production with the wrong host.
//
// Source: bruno-common/src/utils/environment-inheritance.ts
// resolveEnvironmentInheritance
func Resolve(environments []*model.Environment, target *model.Environment) (Resolution, error) {
	if target == nil {
		return Resolution{}, errors.New("environment: nil environment")
	}

	byName := indexByName(environments)

	var ancestors []*model.Environment
	walked := []string{target.Name}
	seen := map[string]bool{target.Name: true}
	current := target

	for current.Extends != "" {
		parentName := strings.TrimSpace(current.Extends)
		parent, ok := byName[parentName]
		if !ok {
			return Resolution{}, &MissingParentError{Environment: current.Name, Parent: parentName}
		}
		if seen[parentName] {
			start := indexOf(walked, parentName)
			path := append(append([]string{}, walked[start:]...), parentName)
			return Resolution{}, &CycleError{Path: path}
		}
		seen[parentName] = true
		walked = append(walked, parentName)
		ancestors = append(ancestors, parent)
		current = parent
	}

	// Root ancestor first, so nearer ancestors override farther ones.
	reverse(ancestors)

	// Secrets and non-secrets are tracked separately, so a plaintext variable
	// cannot shadow a secret of the same name by ordering alone.
	nonSecrets := make(map[string]ResolvedVariable)
	secrets := make(map[string]ResolvedVariable)
	order := make([]string, 0, len(ancestors))

	for _, ancestor := range ancestors {
		from := &InheritedFrom{Name: ancestor.Name, UID: ancestor.UID}
		for _, v := range ancestor.Variables {
			if !v.Enabled {
				continue
			}
			name := v.NameOrEmpty()
			if name == "" {
				continue
			}
			entry := ResolvedVariable{Variable: v, From: from}
			if v.Secret {
				secrets[name] = entry
				continue
			}
			if _, exists := nonSecrets[name]; !exists {
				order = append(order, name)
			}
			nonSecrets[name] = entry
		}
	}

	// A name the child defines stops being inherited at all.
	for _, v := range target.Variables {
		if !v.Enabled {
			continue
		}
		name := v.NameOrEmpty()
		delete(nonSecrets, name)
		delete(secrets, name)
	}

	inherited := make([]ResolvedVariable, 0, len(nonSecrets)+len(secrets))
	for _, name := range order {
		if v, ok := nonSecrets[name]; ok {
			inherited = append(inherited, v)
		}
	}
	// Secrets follow non-secrets, sorted, so the result is deterministic.
	secretNames := make([]string, 0, len(secrets))
	for name := range secrets {
		secretNames = append(secretNames, name)
	}
	sort.Strings(secretNames)
	for _, name := range secretNames {
		inherited = append(inherited, secrets[name])
	}

	chain := make([]string, 0, len(ancestors)+1)
	for _, a := range ancestors {
		chain = append(chain, a.Name)
	}
	chain = append(chain, target.Name)

	return Resolution{Environment: target, Inherited: inherited, Chain: chain}, nil
}

// Inheritable lists the environments target may extend.
//
// Everything is allowed except the target itself and the environments that
// already extend it, because choosing one of those would close a cycle. The UI
// uses this to populate its picker.
//
// Source: bruno-common/src/utils/environment-inheritance.ts getInheritableEnvironments
func Inheritable(environments []*model.Environment, target *model.Environment) []*model.Environment {
	if target == nil {
		return append([]*model.Environment(nil), environments...)
	}

	out := make([]*model.Environment, 0, len(environments))
	for _, env := range environments {
		if env == nil || env == target || env.Name == target.Name || env.UID == target.UID {
			continue
		}
		if reaches(environments, env, target.Name) {
			continue
		}
		out = append(out, env)
	}
	return out
}

// reaches reports whether env's ancestor chain contains the named environment.
//
// It stops on a repeated name as well as on a missing parent, so a broken chain
// elsewhere in the collection cannot loop here.
func reaches(environments []*model.Environment, env *model.Environment, name string) bool {
	byName := indexByName(environments)
	seen := map[string]bool{env.Name: true}
	current := env
	for current != nil && current.Extends != "" {
		parentName := strings.TrimSpace(current.Extends)
		if parentName == name {
			return true
		}
		if seen[parentName] {
			return false
		}
		seen[parentName] = true
		current = byName[parentName]
	}
	return false
}

// Select returns the environment whose UID matches, or nil.
//
// Selection is by UID rather than name: two environments may share a display
// name across formats, and a rename must not silently move the active selection.
func Select(environments []*model.Environment, uid string) *model.Environment {
	if uid == "" {
		return nil
	}
	for _, env := range environments {
		if env != nil && env.UID == uid {
			return env
		}
	}
	return nil
}

// indexByName maps environment names to environments.
//
// The first definition wins when names collide, matching Bruno's `Array.find`, so
// a duplicate name resolves the same way in both implementations.
func indexByName(environments []*model.Environment) map[string]*model.Environment {
	byName := make(map[string]*model.Environment, len(environments))
	for _, env := range environments {
		if env == nil || env.Name == "" {
			continue
		}
		if _, exists := byName[env.Name]; !exists {
			byName[env.Name] = env
		}
	}
	return byName
}

func indexOf(list []string, want string) int {
	for i, s := range list {
		if s == want {
			return i
		}
	}
	return 0
}

func reverse[T any](list []T) {
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}
