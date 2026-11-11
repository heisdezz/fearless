package extension

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

// VariableResolver provides dynamic or external variable resolution.
type VariableResolver interface {
	Prefix() string
	Resolve(ctx context.Context, key string) (any, bool, error)
}

// VariableResolverRegistry manages variable resolvers by their namespace prefix.
type VariableResolverRegistry struct {
	mu        sync.RWMutex
	resolvers map[string]VariableResolver
}

// NewVariableResolverRegistry creates a registry preloaded with standard resolvers.
func NewVariableResolverRegistry() *VariableResolverRegistry {
	r := &VariableResolverRegistry{
		resolvers: make(map[string]VariableResolver),
	}
	r.Register(&DynamicVarResolver{})
	r.Register(&EnvVarResolver{prefix: "env"})
	r.Register(&EnvVarResolver{prefix: "process.env"})
	return r
}

// Register registers or replaces a VariableResolver.
func (r *VariableResolverRegistry) Register(resolver VariableResolver) {
	if resolver == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvers[strings.ToLower(resolver.Prefix())] = resolver
}

// Unregister removes a resolver by prefix.
func (r *VariableResolverRegistry) Unregister(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.resolvers, strings.ToLower(prefix))
}

// Get returns the resolver matching prefix.
func (r *VariableResolverRegistry) Get(prefix string) VariableResolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resolvers[strings.ToLower(prefix)]
}

// Resolve attempts to resolve expression across registered namespaces.
func (r *VariableResolverRegistry) Resolve(ctx context.Context, expression string) (any, bool, error) {
	expr := strings.TrimSpace(expression)
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Dynamic variables starting with "$"
	if strings.HasPrefix(expr, "$") {
		if res, ok := r.resolvers["$"]; ok {
			return res.Resolve(ctx, expr)
		}
	}

	// 2. Namespaced expressions: "prefix.key"
	if dotIdx := strings.Index(expr, "."); dotIdx > 0 {
		prefix := strings.ToLower(expr[:dotIdx])
		key := expr[dotIdx+1:]
		if res, ok := r.resolvers[prefix]; ok {
			return res.Resolve(ctx, key)
		}

		// Also check compound prefixes like "process.env"
		for p, res := range r.resolvers {
			if strings.HasPrefix(strings.ToLower(expr), p+".") {
				subKey := expr[len(p)+1:]
				return res.Resolve(ctx, subKey)
			}
		}
	}

	return nil, false, nil
}

// DynamicVarResolver provides Bruno/Postman compatible dynamic variables.
type DynamicVarResolver struct{}

func (d *DynamicVarResolver) Prefix() string { return "$" }

func (d *DynamicVarResolver) Resolve(ctx context.Context, key string) (any, bool, error) {
	k := strings.TrimPrefix(key, "$")
	switch strings.ToLower(k) {
	case "guid":
		uuid, err := newUUID()
		return uuid, err == nil, err
	case "timestamp":
		return fmt.Sprintf("%d", time.Now().Unix()), true, nil
	case "isotimestamp":
		return time.Now().UTC().Format(time.RFC3339), true, nil
	case "randomint":
		n, err := rand.Int(rand.Reader, big.NewInt(1000))
		if err != nil {
			return 42, true, nil
		}
		return int(n.Int64() + 1), true, nil
	}
	return nil, false, nil
}

func newUUID() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// EnvVarResolver resolves environment variables from host OS.
type EnvVarResolver struct {
	prefix string
}

func (e *EnvVarResolver) Prefix() string { return e.prefix }

func (e *EnvVarResolver) Resolve(ctx context.Context, key string) (any, bool, error) {
	val, ok := os.LookupEnv(key)
	return val, ok, nil
}

// MapVarResolver provides static map variable lookup.
type MapVarResolver struct {
	prefix string
	vars   map[string]any
}

// NewMapVarResolver creates a resolver backed by an in-memory map.
func NewMapVarResolver(prefix string, vars map[string]any) *MapVarResolver {
	return &MapVarResolver{
		prefix: prefix,
		vars:   vars,
	}
}

func (m *MapVarResolver) Prefix() string { return m.prefix }

func (m *MapVarResolver) Resolve(ctx context.Context, key string) (any, bool, error) {
	if m.vars == nil {
		return nil, false, nil
	}
	val, ok := m.vars[key]
	return val, ok, nil
}
