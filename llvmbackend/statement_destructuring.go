package llvmbackend

import (
	"fmt"
	"strconv"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeDestructureArrayIndexSymbol = "jayess_destructure_array_index"
	runtimeDestructureArrayRestSymbol  = "jayess_destructure_array_rest"
	runtimeDestructureDefaultSymbol    = "jayess_destructure_default"
	runtimeDestructureObjectRestSymbol = "jayess_destructure_object_rest"
	runtimeDestructurePropertySymbol   = "jayess_destructure_property"
)

func (emitter *StatementEmitter) emitVariableDeclaration(statement *ast.VariableDecl) error {
	value := "undef"
	if statement.Value != nil {
		emitted, err := emitter.expressions.EmitExpression(statement.Value)
		if err != nil {
			return err
		}
		value = emitted
	}
	if err := emitter.emitBindingDeclaration(statement.Pattern, value); err != nil {
		return err
	}
	emitter.registerDeclarationLifetime(statement, statement.Pattern)
	return nil
}

func (emitter *StatementEmitter) emitBindingDeclaration(pattern ast.BindingPattern, value string) error {
	if name, ok := bindingName(pattern); ok {
		return emitter.expressions.DeclareLocal(name, value)
	}
	if value == "undef" {
		undefined, err := emitter.emitRuntimeUndefined()
		if err != nil {
			return err
		}
		value = undefined
	}
	if err := emitter.declareBindingTargets(pattern, "undef"); err != nil {
		return err
	}
	return emitter.emitDestructureToPattern(pattern, value)
}

func (emitter *StatementEmitter) declareBindingTargets(pattern ast.BindingPattern, value string) error {
	switch pattern := pattern.(type) {
	case nil:
		return nil
	case *ast.BindingName:
		return emitter.expressions.DeclareLocal(pattern.Name, value)
	case *ast.BindingDefault:
		return emitter.declareBindingTargets(pattern.Pattern, value)
	case *ast.BindingRest:
		return emitter.declareBindingTargets(pattern.Pattern, value)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			if err := emitter.declareBindingTargets(element, value); err != nil {
				return err
			}
		}
		return nil
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			if err := emitter.declareBindingTargets(property.Pattern, value); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported runtime binding pattern %T", pattern)
	}
}

func (emitter *StatementEmitter) emitDestructureToPattern(pattern ast.BindingPattern, source string) error {
	switch pattern := pattern.(type) {
	case nil:
		return nil
	case *ast.BindingName:
		target, err := emitter.expressions.ResolveAssignmentTarget(&ast.Identifier{Name: pattern.Name})
		if err != nil {
			return err
		}
		return target.Store(source)
	case *ast.BindingDefault:
		fallback, err := emitter.expressions.EmitExpression(pattern.Value)
		if err != nil {
			return err
		}
		value, err := emitter.emitRuntimeDestructureDefault(source, fallback)
		if err != nil {
			return err
		}
		return emitter.emitDestructureToPattern(pattern.Pattern, value)
	case *ast.BindingRest:
		return emitter.emitDestructureToPattern(pattern.Pattern, source)
	case *ast.ArrayBindingPattern:
		return emitter.emitArrayDestructure(pattern, source)
	case *ast.ObjectBindingPattern:
		return emitter.emitObjectDestructure(pattern, source)
	default:
		return fmt.Errorf("unsupported runtime binding pattern %T", pattern)
	}
}

func (emitter *StatementEmitter) emitArrayDestructure(pattern *ast.ArrayBindingPattern, source string) error {
	for index, element := range pattern.Elements {
		if element == nil {
			continue
		}
		if rest, ok := element.(*ast.BindingRest); ok {
			value, err := emitter.emitRuntimeArrayRest(source, index)
			if err != nil {
				return err
			}
			if err := emitter.emitDestructureToPattern(rest.Pattern, value); err != nil {
				return err
			}
			continue
		}
		value, err := emitter.emitRuntimeArrayIndex(source, index, "undef")
		if err != nil {
			return err
		}
		if err := emitter.emitDestructureToPattern(element, value); err != nil {
			return err
		}
	}
	return nil
}

func (emitter *StatementEmitter) emitObjectDestructure(pattern *ast.ObjectBindingPattern, source string) error {
	for _, property := range pattern.Properties {
		if property.Rest {
			value, err := emitter.emitRuntimeObjectRest(source)
			if err != nil {
				return err
			}
			if err := emitter.emitDestructureToPattern(property.Pattern, value); err != nil {
				return err
			}
			continue
		}
		key, err := emitter.emitObjectBindingKey(property)
		if err != nil {
			return err
		}
		value, err := emitter.emitRuntimePropertyDestructure(source, key, "undef")
		if err != nil {
			return err
		}
		if err := emitter.emitDestructureToPattern(property.Pattern, value); err != nil {
			return err
		}
	}
	return nil
}

func (emitter *StatementEmitter) emitObjectBindingKey(property ast.ObjectBindingProperty) (string, error) {
	if property.Computed {
		if property.KeyExpr == nil {
			return "", fmt.Errorf("computed object binding key must have an expression")
		}
		return emitter.expressions.EmitExpression(property.KeyExpr)
	}
	return emitter.expressions.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: property.Key})
}

func (emitter *StatementEmitter) emitRuntimeArrayIndex(source string, index int, fallback string) (string, error) {
	if fallback == "undef" {
		undefined, err := emitter.emitRuntimeUndefined()
		if err != nil {
			return "", err
		}
		fallback = undefined
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: source},
		{IRType: "i32", Value: strconv.Itoa(index)},
		{IRType: runtimeValueIRType, Value: fallback},
	}
	return emitter.emitRuntimeDestructureValue(runtimeDestructureArrayIndexSymbol, args)
}

func (emitter *StatementEmitter) emitRuntimeArrayRest(source string, start int) (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: source},
		{IRType: "i32", Value: strconv.Itoa(start)},
	}
	return emitter.emitRuntimeDestructureValue(runtimeDestructureArrayRestSymbol, args)
}

func (emitter *StatementEmitter) emitRuntimeObjectRest(source string) (string, error) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: source}}
	return emitter.emitRuntimeDestructureValue(runtimeDestructureObjectRestSymbol, args)
}

func (emitter *StatementEmitter) emitRuntimePropertyDestructure(source string, key string, fallback string) (string, error) {
	if fallback == "undef" {
		undefined, err := emitter.emitRuntimeUndefined()
		if err != nil {
			return "", err
		}
		fallback = undefined
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: source},
		{IRType: runtimeValueIRType, Value: key},
		{IRType: runtimeValueIRType, Value: fallback},
	}
	return emitter.emitRuntimeDestructureValue(runtimeDestructurePropertySymbol, args)
}

func (emitter *StatementEmitter) emitRuntimeDestructureDefault(value string, fallback string) (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: value},
		{IRType: runtimeValueIRType, Value: fallback},
	}
	return emitter.emitRuntimeDestructureValue(runtimeDestructureDefaultSymbol, args)
}

func (emitter *StatementEmitter) emitRuntimeUndefined() (string, error) {
	return emitter.expressions.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.UndefinedValue})
}

func (emitter *StatementEmitter) emitRuntimeDestructureValue(symbol string, args []RuntimeCallArg) (string, error) {
	result := emitter.expressions.nextValueName()
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}
