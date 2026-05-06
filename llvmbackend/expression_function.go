package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/lifetime"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeClosureEnvironmentNewSymbol = "jayess_closure_environment_new"
	runtimeClosureEnvironmentSetSymbol = "jayess_closure_environment_set"
	runtimeFunctionNewSymbol           = "jayess_function_new"
	runtimeFunctionNewClosureSymbol    = "jayess_function_new_with_closure"
)

func (emitter *ExpressionEmitter) emitFunctionExpression(expression *ast.FunctionExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("function expression must not be nil")
	}
	if environment, ok := emitter.closureEnvironmentFor(ast.PositionOf(expression)); ok {
		closure, err := emitter.emitClosureEnvironment(environment)
		if err != nil {
			return "", err
		}
		return emitter.emitRuntimeClosureFunctionNew(closure)
	}
	return emitter.emitRuntimeFunctionNew()
}

func (emitter *ExpressionEmitter) emitRuntimeFunctionNew() (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeFunctionNewSymbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, runtimeFunctionNewSymbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitRuntimeClosureFunctionNew(environment string) (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{{IRType: "i8*", Value: environment}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeFunctionNewClosureSymbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, runtimeFunctionNewClosureSymbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitClosureEnvironment(environment lifetime.ClosureEnvironment) (string, error) {
	closure := emitter.nextValueName()
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeClosureEnvironmentNewSymbol, "i8*", nil)})
	emitter.body = append(emitter.body, RuntimeCall(closure, "i8*", runtimeClosureEnvironmentNewSymbol, nil))
	for _, capture := range environment.Captures {
		value, err := emitter.LoadLocal(capture.Binding)
		if err != nil {
			return "", err
		}
		key, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: capture.Binding})
		if err != nil {
			return "", err
		}
		args := []RuntimeCallArg{
			{IRType: "i8*", Value: closure},
			{IRType: runtimeValueIRType, Value: key},
			{IRType: runtimeValueIRType, Value: value},
		}
		emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeClosureEnvironmentSetSymbol, "void", args)})
		emitter.body = append(emitter.body, RuntimeVoidCall(runtimeClosureEnvironmentSetSymbol, args))
	}
	return closure, nil
}

func (emitter *ExpressionEmitter) closureEnvironmentFor(pos ast.SourcePos) (lifetime.ClosureEnvironment, bool) {
	if emitter.lifetimePlan == nil {
		return lifetime.ClosureEnvironment{}, false
	}
	for _, environment := range emitter.lifetimePlan.ClosureEnvironments {
		if environment.Line == pos.Line && environment.Column == pos.Column {
			return environment, true
		}
	}
	return lifetime.ClosureEnvironment{}, false
}
