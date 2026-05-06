package llvmbackend

import (
	"fmt"
	"strconv"
)

type localSlot struct {
	Name string
}

func (emitter *ExpressionEmitter) LoadLocal(name string) (string, error) {
	slot, exists := emitter.locals[name]
	if !exists {
		return "", fmt.Errorf("undefined emitted local %s", name)
	}
	result := emitter.nextValueName()
	emitter.body = append(emitter.body, result+" = load "+runtimeValueIRType+", "+runtimeValueIRType+"* "+slot.Name)
	return result, nil
}

func (emitter *ExpressionEmitter) StoreLocal(name string, value string) error {
	slot, exists := emitter.locals[name]
	if !exists {
		return fmt.Errorf("assignment to undefined emitted local %s", name)
	}
	if value == "" {
		return fmt.Errorf("local %s must not be stored from an empty value", name)
	}
	emitter.body = append(emitter.body, "store "+runtimeValueIRType+" "+value+", "+runtimeValueIRType+"* "+slot.Name)
	return nil
}

func (emitter *ExpressionEmitter) nextLocalSlot() localSlot {
	slot := localSlot{Name: "%local." + strconv.Itoa(emitter.slotIndex)}
	emitter.slotIndex++
	return slot
}
