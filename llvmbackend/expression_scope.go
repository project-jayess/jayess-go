package llvmbackend

import "fmt"

type scopedLocal struct {
	slot   localSlot
	exists bool
}

func (emitter *ExpressionEmitter) PushScope() {
	emitter.scopes = append(emitter.scopes, map[string]scopedLocal{})
}

func (emitter *ExpressionEmitter) PopScope() error {
	if len(emitter.scopes) == 0 {
		return fmt.Errorf("cannot pop runtime local scope from empty stack")
	}
	scope := emitter.scopes[len(emitter.scopes)-1]
	emitter.scopes = emitter.scopes[:len(emitter.scopes)-1]
	for name, previous := range scope {
		if previous.exists {
			emitter.locals[name] = previous.slot
			continue
		}
		delete(emitter.locals, name)
	}
	return nil
}

func (emitter *ExpressionEmitter) DeclareLocal(name string, value string) error {
	emitter.recordScopedLocal(name)
	return emitter.BindLocal(name, value)
}

func (emitter *ExpressionEmitter) AssignLocal(name string, value string) error {
	if !emitter.HasLocal(name) {
		return fmt.Errorf("assignment to undefined emitted local %s", name)
	}
	return emitter.StoreLocal(name, value)
}

func (emitter *ExpressionEmitter) recordScopedLocal(name string) {
	if len(emitter.scopes) == 0 || name == "" {
		return
	}
	scope := emitter.scopes[len(emitter.scopes)-1]
	if _, recorded := scope[name]; recorded {
		return
	}
	slot, exists := emitter.locals[name]
	scope[name] = scopedLocal{slot: slot, exists: exists}
}
