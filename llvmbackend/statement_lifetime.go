package llvmbackend

import "jayess-go/ast"

const (
	runtimeValueReleaseSymbol = "jayess_value_release"
	runtimeValueRetainSymbol  = "jayess_value_retain"
)

type lifetimeCleanup struct {
	Binding string
}

func (emitter *StatementEmitter) registerDeclarationLifetime(node ast.Node, pattern ast.BindingPattern) {
	if emitter.lifetimePlan == nil {
		return
	}
	pos := ast.PositionOf(node)
	for _, name := range bindingNames(pattern) {
		if emitter.hasLifetimeCleanup(name, pos) {
			emitter.pushLifetimeCleanup(name)
		}
		if emitter.hasExtendedLifetime(name, pos) {
			emitter.emitLifetimeRetain(name)
		}
	}
}

func (emitter *StatementEmitter) pushLifetimeCleanup(name string) {
	if len(emitter.cleanupScopes) == 0 || name == "" {
		return
	}
	index := len(emitter.cleanupScopes) - 1
	emitter.cleanupScopes[index] = append(emitter.cleanupScopes[index], lifetimeCleanup{Binding: name})
}

func (emitter *StatementEmitter) emitCurrentScopeCleanups() {
	if len(emitter.cleanupScopes) == 0 {
		return
	}
	scope := emitter.cleanupScopes[len(emitter.cleanupScopes)-1]
	emitter.emitCleanupList(scope)
}

func (emitter *StatementEmitter) emitActiveCleanups() {
	emitter.emitCleanupsUntil(0)
}

func (emitter *StatementEmitter) emitCleanupsUntil(boundary int) {
	for index := len(emitter.cleanupScopes) - 1; index >= boundary && index >= 0; index-- {
		emitter.emitCleanupList(emitter.cleanupScopes[index])
	}
}

func (emitter *StatementEmitter) emitCleanupList(cleanups []lifetimeCleanup) {
	for index := len(cleanups) - 1; index >= 0; index-- {
		value, err := emitter.expressions.LoadLocal(cleanups[index].Binding)
		if err != nil {
			continue
		}
		emitter.emitLifetimeCall(runtimeValueReleaseSymbol, value)
	}
}

func (emitter *StatementEmitter) emitLifetimeRetain(name string) {
	value, err := emitter.expressions.LoadLocal(name)
	if err != nil {
		return
	}
	emitter.emitLifetimeCall(runtimeValueRetainSymbol, value)
}

func (emitter *StatementEmitter) emitLifetimeCall(symbol string, value string) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: value}}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
}

func (emitter *StatementEmitter) hasLifetimeCleanup(name string, pos ast.SourcePos) bool {
	for _, cleanup := range emitter.lifetimePlan.ScopeExitCleanups {
		if sameLifetimeBinding(cleanup.Binding, cleanup.Line, cleanup.Column, name, pos) {
			return true
		}
	}
	return false
}

func (emitter *StatementEmitter) hasExtendedLifetime(name string, pos ast.SourcePos) bool {
	for _, extended := range emitter.lifetimePlan.ExtendedLifetimes {
		if sameLifetimeBinding(extended.Binding, extended.Line, extended.Column, name, pos) {
			return true
		}
	}
	return false
}

func sameLifetimeBinding(binding string, line int, column int, name string, pos ast.SourcePos) bool {
	return binding == name && line == pos.Line && column == pos.Column
}

func bindingNames(pattern ast.BindingPattern) []string {
	seen := map[string]bool{}
	collectBindingNames(pattern, seen)
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

func collectBindingNames(pattern ast.BindingPattern, names map[string]bool) {
	switch pattern := pattern.(type) {
	case *ast.BindingName:
		if pattern.Name != "" {
			names[pattern.Name] = true
		}
	case *ast.BindingDefault:
		collectBindingNames(pattern.Pattern, names)
	case *ast.BindingRest:
		collectBindingNames(pattern.Pattern, names)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			collectBindingNames(element, names)
		}
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			collectBindingNames(property.Pattern, names)
		}
	}
}
