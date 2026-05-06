package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeSetPropertySymbol    = "jayess_value_set_property"
	runtimeSetIndexSymbol       = "jayess_value_set_index"
	runtimeGetPropertySymbol    = "jayess_value_get_property"
	runtimeGetIndexSymbol       = "jayess_value_get_index"
	runtimeDeletePropertySymbol = "jayess_value_delete_property"
	runtimeDeleteIndexSymbol    = "jayess_value_delete_index"
)

type assignmentTarget interface {
	Load() (string, error)
	Store(value string) error
	Delete() (string, error)
}

type localAssignmentTarget struct {
	emitter *ExpressionEmitter
	name    string
}

func (target localAssignmentTarget) Load() (string, error) {
	return target.emitter.LoadLocal(target.name)
}

func (target localAssignmentTarget) Store(value string) error {
	return target.emitter.StoreLocal(target.name, value)
}

func (target localAssignmentTarget) Delete() (string, error) {
	return "", fmt.Errorf("unsupported runtime delete local target %s", target.name)
}

type runtimeSetAssignmentTarget struct {
	emitter *ExpressionEmitter
	getter  string
	setter  string
	deleter string
	object  string
	key     string
}

func (target runtimeSetAssignmentTarget) Load() (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: target.object},
		{IRType: runtimeValueIRType, Value: target.key},
	}
	target.emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(target.getter, runtimeValueIRType, args)})
	result := target.emitter.nextValueName()
	target.emitter.body = append(target.emitter.body, RuntimeCall(result, runtimeValueIRType, target.getter, args))
	return result, nil
}

func (target runtimeSetAssignmentTarget) Store(value string) error {
	if value == "" {
		return fmt.Errorf("runtime assignment target must not store an empty value")
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: target.object},
		{IRType: runtimeValueIRType, Value: target.key},
		{IRType: runtimeValueIRType, Value: value},
	}
	target.emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(target.setter, "void", args)})
	target.emitter.body = append(target.emitter.body, RuntimeVoidCall(target.setter, args))
	return nil
}

func (target runtimeSetAssignmentTarget) Delete() (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: target.object},
		{IRType: runtimeValueIRType, Value: target.key},
	}
	target.emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(target.deleter, runtimeValueIRType, args)})
	result := target.emitter.nextValueName()
	target.emitter.body = append(target.emitter.body, RuntimeCall(result, runtimeValueIRType, target.deleter, args))
	return result, nil
}

func (emitter *ExpressionEmitter) ResolveAssignmentTarget(expression ast.Expression) (assignmentTarget, error) {
	switch target := expression.(type) {
	case *ast.Identifier:
		if !emitter.HasLocal(target.Name) {
			return nil, fmt.Errorf("assignment to undefined emitted local %s", target.Name)
		}
		return localAssignmentTarget{emitter: emitter, name: target.Name}, nil
	case *ast.MemberExpression:
		if target.Optional {
			return nil, fmt.Errorf("unsupported runtime optional member assignment target")
		}
		if target.Private {
			return nil, fmt.Errorf("unsupported runtime private member assignment target")
		}
		object, err := emitter.EmitExpression(target.Target)
		if err != nil {
			return nil, err
		}
		key, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: target.Property})
		if err != nil {
			return nil, err
		}
		return runtimeSetAssignmentTarget{
			emitter: emitter,
			getter:  runtimeGetPropertySymbol,
			setter:  runtimeSetPropertySymbol,
			deleter: runtimeDeletePropertySymbol,
			object:  object,
			key:     key,
		}, nil
	case *ast.IndexExpression:
		if target.Optional {
			return nil, fmt.Errorf("unsupported runtime optional index assignment target")
		}
		object, err := emitter.EmitExpression(target.Target)
		if err != nil {
			return nil, err
		}
		key, err := emitter.EmitExpression(target.Index)
		if err != nil {
			return nil, err
		}
		return runtimeSetAssignmentTarget{
			emitter: emitter,
			getter:  runtimeGetIndexSymbol,
			setter:  runtimeSetIndexSymbol,
			deleter: runtimeDeleteIndexSymbol,
			object:  object,
			key:     key,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime assignment target %T", expression)
	}
}
