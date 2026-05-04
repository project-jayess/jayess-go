package semantic

type scope struct {
	parent  *scope
	symbols map[string]binding
}

type binding struct {
	mutable       bool
	constructable bool
	builtin       bool
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent, symbols: map[string]binding{}}
}

func (s *scope) declare(name string) bool {
	return s.declareBinding(name, binding{mutable: true})
}

func (s *scope) declareConst(name string) bool {
	return s.declareBinding(name, binding{mutable: false})
}

func (s *scope) declareConstructable(name string) bool {
	return s.declareBinding(name, binding{mutable: true, constructable: true})
}

func (s *scope) declareBuiltin(name string) bool {
	return s.declareBinding(name, binding{mutable: true, builtin: true})
}

func (s *scope) declareBuiltinConst(name string) bool {
	return s.declareBinding(name, binding{builtin: true})
}

func (s *scope) declareBuiltinConstructable(name string) bool {
	return s.declareBinding(name, binding{mutable: true, constructable: true, builtin: true})
}

func (s *scope) declareImported(name string) bool {
	if existing, exists := s.symbols[name]; exists && existing.builtin {
		s.symbols[name] = binding{constructable: true}
		return true
	}
	return s.declareBinding(name, binding{constructable: true})
}

func (s *scope) declareBinding(name string, bind binding) bool {
	if _, exists := s.symbols[name]; exists {
		return false
	}
	s.symbols[name] = bind
	return true
}

func (s *scope) lookup(name string) bool {
	_, ok := s.resolve(name)
	return ok
}

func (s *scope) hasLocal(name string) bool {
	_, ok := s.symbols[name]
	return ok
}

func (s *scope) resolve(name string) (binding, bool) {
	for current := s; current != nil; current = current.parent {
		if bind, exists := current.symbols[name]; exists {
			return bind, true
		}
	}
	return binding{}, false
}
