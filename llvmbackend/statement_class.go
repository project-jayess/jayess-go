package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeClassDefineConstructorSymbol   = "jayess_class_define_constructor"
	runtimeClassDefineAccessorSymbol      = "jayess_class_define_accessor"
	runtimeClassDefineFieldSymbol         = "jayess_class_define_field"
	runtimeClassDefineMethodSymbol        = "jayess_class_define_method"
	runtimeClassDefinePrivateFieldSymbol  = "jayess_class_define_private_field"
	runtimeClassDefinePrivateMethodSymbol = "jayess_class_define_private_method"
	runtimeClassDefineStaticBlockSymbol   = "jayess_class_define_static_block"
	runtimeClassDefineStaticFieldSymbol   = "jayess_class_define_static_field"
	runtimeClassRunStaticBlocksSymbol     = "jayess_class_run_static_blocks"
	runtimeClassExtendsSymbol             = "jayess_class_extends"
	runtimeClassNewSymbol                 = "jayess_class_new"
)

func (emitter *StatementEmitter) emitClassDeclaration(statement *ast.ClassDecl) error {
	if statement == nil || statement.Name == "" {
		return fmt.Errorf("runtime class declaration must have a name")
	}
	name, err := emitter.expressions.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: statement.Name})
	if err != nil {
		return err
	}
	classValue, err := emitter.expressions.emitRuntimeValueCall(runtimeClassNewSymbol, []RuntimeCallArg{{IRType: runtimeValueIRType, Value: name}})
	if err != nil {
		return err
	}
	if statement.SuperClass != nil {
		superClass, err := emitter.expressions.EmitExpression(statement.SuperClass)
		if err != nil {
			return err
		}
		classValue, err = emitter.expressions.emitRuntimeValueCall(runtimeClassExtendsSymbol, []RuntimeCallArg{
			{IRType: runtimeValueIRType, Value: classValue},
			{IRType: runtimeValueIRType, Value: superClass},
		})
		if err != nil {
			return err
		}
	}
	if err := emitter.expressions.DeclareLocal(statement.Name, classValue); err != nil {
		return err
	}
	for _, member := range statement.Members {
		if err := emitter.emitClassMember(classValue, member); err != nil {
			return err
		}
	}
	emitter.emitClassNoArgVoidCall(runtimeClassRunStaticBlocksSymbol, classValue)
	return nil
}

func (emitter *StatementEmitter) emitClassMember(classValue string, member ast.ClassMember) error {
	if member.Constructor {
		constructor, err := emitter.expressions.emitRuntimeFunctionNew()
		if err != nil {
			return err
		}
		emitter.emitClassVoidCall(runtimeClassDefineConstructorSymbol, classValue, constructor)
		return nil
	}
	if member.StaticBlock {
		block, err := emitter.expressions.emitRuntimeFunctionNew()
		if err != nil {
			return err
		}
		emitter.emitClassVoidCall(runtimeClassDefineStaticBlockSymbol, classValue, block)
		return nil
	}
	key, err := emitter.emitClassMemberKey(member)
	if err != nil {
		return err
	}
	if member.Field {
		value := "undef"
		if member.Value != nil {
			value, err = emitter.expressions.EmitExpression(member.Value)
			if err != nil {
				return err
			}
		} else {
			value, err = emitter.emitRuntimeUndefined()
			if err != nil {
				return err
			}
		}
		symbol := runtimeClassDefineFieldSymbol
		if member.Static {
			symbol = runtimeClassDefineStaticFieldSymbol
		}
		if member.Private {
			symbol = runtimeClassDefinePrivateFieldSymbol
		}
		emitter.emitClassKeyValueVoidCall(symbol, classValue, key, value)
		return nil
	}
	if member.Getter || member.Setter {
		getter, setter, err := emitter.emitClassAccessorFunctions(member)
		if err != nil {
			return err
		}
		emitter.emitClassAccessorVoidCall(runtimeClassDefineAccessorSymbol, classValue, key, getter, setter)
		return nil
	}
	method, err := emitter.expressions.emitRuntimeFunctionNew()
	if err != nil {
		return err
	}
	symbol := runtimeClassDefineMethodSymbol
	if member.Private {
		symbol = runtimeClassDefinePrivateMethodSymbol
	}
	emitter.emitClassKeyValueVoidCall(symbol, classValue, key, method)
	return nil
}

func (emitter *StatementEmitter) emitClassMemberKey(member ast.ClassMember) (string, error) {
	if member.Computed {
		if member.KeyExpr == nil {
			return "", fmt.Errorf("computed class member key must have an expression")
		}
		return emitter.expressions.EmitExpression(member.KeyExpr)
	}
	return emitter.expressions.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: member.Name})
}

func (emitter *StatementEmitter) emitClassAccessorFunctions(member ast.ClassMember) (string, string, error) {
	missing, err := emitter.emitRuntimeUndefined()
	if err != nil {
		return "", "", err
	}
	fn, err := emitter.expressions.emitRuntimeFunctionNew()
	if err != nil {
		return "", "", err
	}
	if member.Getter {
		return fn, missing, nil
	}
	return missing, fn, nil
}

func (emitter *StatementEmitter) emitClassVoidCall(symbol string, classValue string, value string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: classValue},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
}

func (emitter *StatementEmitter) emitClassKeyValueVoidCall(symbol string, classValue string, key string, value string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: classValue},
		{IRType: runtimeValueIRType, Value: key},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
}

func (emitter *StatementEmitter) emitClassAccessorVoidCall(symbol string, classValue string, key string, getter string, setter string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: classValue},
		{IRType: runtimeValueIRType, Value: key},
		{IRType: runtimeValueIRType, Value: getter},
		{IRType: runtimeValueIRType, Value: setter},
	}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
}

func (emitter *StatementEmitter) emitClassNoArgVoidCall(symbol string, classValue string) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: classValue}}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
}
