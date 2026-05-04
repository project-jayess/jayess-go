package escape

// Report contains conservative escape-analysis decisions by binding name.
type Report struct {
	escaped map[string]bool
	aliases map[string]map[string]bool
}

func newReport() *Report {
	return &Report{
		escaped: map[string]bool{},
		aliases: map[string]map[string]bool{},
	}
}

// Escapes reports whether a binding is treated as escaping.
func (r *Report) Escapes(name string) bool {
	if r == nil {
		return false
	}
	return r.escaped[name]
}

// EligibleForScopeCleanup reports whether a binding can use scope-based cleanup.
func (r *Report) EligibleForScopeCleanup(name string) bool {
	return r != nil && name != "" && !r.Escapes(name)
}

// MustSurviveScopeExit reports whether a binding must not be destroyed at scope exit.
func (r *Report) MustSurviveScopeExit(name string) bool {
	return r != nil && name != "" && r.Escapes(name)
}

func (r *Report) markEscaping(name string) {
	r.markEscapingSeen(name, map[string]bool{})
}

func (r *Report) markEscapingSeen(name string, seen map[string]bool) {
	if name == "" {
		return
	}
	if seen[name] {
		return
	}
	seen[name] = true
	r.escaped[name] = true
	for alias := range r.aliases[name] {
		r.markEscapingSeen(alias, seen)
	}
}

func (r *Report) addAlias(first string, second string) {
	if first == "" || second == "" || first == second {
		return
	}
	if r.aliases[first] == nil {
		r.aliases[first] = map[string]bool{}
	}
	if r.aliases[second] == nil {
		r.aliases[second] = map[string]bool{}
	}
	r.aliases[first][second] = true
	r.aliases[second][first] = true
	if r.Escapes(first) || r.Escapes(second) {
		r.markEscaping(first)
		r.markEscaping(second)
	}
}
