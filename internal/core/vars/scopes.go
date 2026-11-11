package vars

// Scopes holds ordered variable maps for resolution. Later scopes override earlier
// ones when merged, matching flows/environment-resolution.md precedence:
// global -> collection -> environment -> folder -> request -> oauth2 -> runtime -> prompt -> process.
type Scopes struct {
	Global      map[string]any
	Collection  map[string]any
	Environment map[string]any
	Folder      map[string]any
	Request     map[string]any
	OAuth2      map[string]any
	Runtime     map[string]any
	Prompt      map[string]any
	ProcessEnv  map[string]string
}

// Merge produces a single lookup map with precedence applied.
func (s Scopes) Merge() map[string]any {
	out := map[string]any{}
	mergeInto(out, s.Global)
	mergeInto(out, s.Collection)
	mergeInto(out, s.Environment)
	mergeInto(out, s.Folder)
	mergeInto(out, s.Request)
	mergeInto(out, s.OAuth2)
	mergeInto(out, s.Runtime)
	mergeInto(out, s.Prompt)
	if len(s.ProcessEnv) > 0 {
		env := make(map[string]any, len(s.ProcessEnv))
		for k, v := range s.ProcessEnv {
			env[k] = v
		}
		out["process"] = map[string]any{"env": env}
	}
	return out
}

func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

// InterpolateWithScopes interpolates s against merged scopes.
func InterpolateWithScopes(s string, scopes Scopes, opts InterpolateOptions) string {
	return Interpolate(s, scopes.Merge(), opts)
}

// UnresolvedCollector collects unresolved variable names in order of first report.
type UnresolvedCollector struct {
	seen  map[string]bool
	Names []string
}

// NewUnresolvedCollector returns a collector suitable for OnUnresolved.
func NewUnresolvedCollector() *UnresolvedCollector {
	return &UnresolvedCollector{seen: map[string]bool{}}
}

// OnUnresolved records name once.
func (c *UnresolvedCollector) OnUnresolved(name string) {
	if c == nil || c.seen[name] {
		return
	}
	c.seen[name] = true
	c.Names = append(c.Names, name)
}

// MergeFrom merges names from other, preserving uniqueness/order.
func (c *UnresolvedCollector) MergeFrom(other *UnresolvedCollector) {
	if c == nil || other == nil {
		return
	}
	for _, n := range other.Names {
		c.OnUnresolved(n)
	}
}
