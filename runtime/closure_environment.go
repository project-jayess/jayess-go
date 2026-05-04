package runtime

type ClosureEnvironment struct {
	slots map[string]Value
	order []string
}

func NewClosureEnvironment() *ClosureEnvironment {
	return &ClosureEnvironment{slots: map[string]Value{}}
}

func NewClosureEnvironmentFromCaptures(captures map[string]Value) *ClosureEnvironment {
	environment := NewClosureEnvironment()
	for name, value := range captures {
		environment.Set(name, value)
	}
	return environment
}

func (environment *ClosureEnvironment) Set(name string, value Value) {
	if environment.slots == nil {
		environment.slots = map[string]Value{}
	}
	if _, exists := environment.slots[name]; !exists {
		environment.order = append(environment.order, name)
	}
	environment.slots[name] = value
}

func (environment *ClosureEnvironment) Get(name string) (Value, bool) {
	if environment == nil || environment.slots == nil {
		return Undefined(), false
	}
	value, exists := environment.slots[name]
	if !exists {
		return Undefined(), false
	}
	return value, true
}

func (environment *ClosureEnvironment) Has(name string) bool {
	if environment == nil || environment.slots == nil {
		return false
	}
	_, exists := environment.slots[name]
	return exists
}

func (environment *ClosureEnvironment) Names() []string {
	if environment == nil {
		return nil
	}
	names := make([]string, 0, len(environment.order))
	for _, name := range environment.order {
		if environment.Has(name) {
			names = append(names, name)
		}
	}
	return names
}

func (environment *ClosureEnvironment) Clone() *ClosureEnvironment {
	clone := NewClosureEnvironment()
	for _, name := range environment.Names() {
		value, _ := environment.Get(name)
		clone.Set(name, value)
	}
	return clone
}
