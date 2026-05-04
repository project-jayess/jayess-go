package escape

type scope struct {
	parent *scope
	names  map[string]bool
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent, names: map[string]bool{}}
}

func (s *scope) declare(name string) {
	if name != "" {
		s.names[name] = true
	}
}

func (s *scope) hasLocal(name string) bool {
	return s != nil && s.names[name]
}

func (s *scope) hasOuter(name string) bool {
	if s == nil {
		return false
	}
	for current := s.parent; current != nil; current = current.parent {
		if current.names[name] {
			return true
		}
	}
	return false
}
