package lowering

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/ir"
)

func Lower(program *ast.Program) (*ir.Module, error) {
	module := &ir.Module{}
	globalSymbols := map[string]ir.ValueKind{}
	for _, global := range program.Globals {
		value, err := lowerExpression(global.Value, globalSymbols)
		if err != nil {
			return nil, err
		}
		module.Globals = append(module.Globals, ir.VariableDecl{
			Visibility: lowerVisibility(global.Visibility),
			Kind:       lowerDeclarationKind(global.Kind),
			Name:       global.Name,
			Value:      value,
		})
		globalSymbols[global.Name] = ir.ValueDynamic
	}
	for _, fn := range program.ExternFunctions {
		symbolName := fn.NativeSymbol
		if symbolName == "" {
			symbolName = fn.Name
		}
		lowered := ir.ExternFunction{Name: fn.Name, SymbolName: symbolName, Variadic: fn.Variadic}
		for _, param := range fn.Params {
			lowered.Params = append(lowered.Params, ir.Parameter{Name: param.Name, Kind: ir.ValueDynamic})
		}
		module.ExternFunctions = append(module.ExternFunctions, lowered)
	}
	for _, fn := range program.Functions {
		lowered, err := lowerFunction(fn, globalSymbols)
		if err != nil {
			return nil, err
		}
		module.Functions = append(module.Functions, lowered)
	}
	return module, nil
}

func lowerFunction(fn *ast.FunctionDecl, globals map[string]ir.ValueKind) (ir.Function, error) {
	result := ir.Function{Visibility: lowerVisibility(fn.Visibility), Name: fn.Name}

	symbols := cloneKinds(globals)
	for _, param := range fn.Params {
		kind := ir.ValueDynamic
		if fn.Name == "main" {
			kind = ir.ValueArgsArray
		}
		result.Params = append(result.Params, ir.Parameter{Name: param.Name, Kind: kind})
		symbols[param.Name] = kind
	}

	body, err := lowerStatements(fn.Body, symbols)
	if err != nil {
		return ir.Function{}, err
	}
	result.Body = body
	return result, nil
}

func lowerStatements(statements []ast.Statement, symbols map[string]ir.ValueKind) ([]ir.Statement, error) {
	var out []ir.Statement
	local := cloneKinds(symbols)
	for _, stmt := range statements {
		lowered, err := lowerStatement(stmt, local)
		if err != nil {
			return nil, err
		}
		if decl, ok := lowered.(*ir.VariableDecl); ok {
			local[decl.Name] = inferIRKind(decl.Value)
		}
		out = append(out, lowered)
	}
	return out, nil
}

func lowerStatement(stmt ast.Statement, symbols map[string]ir.ValueKind) (ir.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := lowerExpression(stmt.Value, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.VariableDecl{
			Visibility: lowerVisibility(stmt.Visibility),
			Kind:       lowerDeclarationKind(stmt.Kind),
			Name:       stmt.Name,
			Value:      value,
		}, nil
	case *ast.AssignmentStatement:
		target, err := lowerExpression(stmt.Target, symbols)
		if err != nil {
			return nil, err
		}
		value, err := lowerExpression(stmt.Value, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.AssignmentStatement{Target: target, Value: value}, nil
	case *ast.ReturnStatement:
		value, err := lowerExpression(stmt.Value, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.ReturnStatement{Value: value}, nil
	case *ast.ExpressionStatement:
		value, err := lowerExpression(stmt.Expression, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.ExpressionStatement{Expression: value}, nil
	case *ast.IfStatement:
		condition, err := lowerExpression(stmt.Condition, symbols)
		if err != nil {
			return nil, err
		}
		consequence, err := lowerStatements(stmt.Consequence, symbols)
		if err != nil {
			return nil, err
		}
		alternative, err := lowerStatements(stmt.Alternative, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := lowerExpression(stmt.Condition, symbols)
		if err != nil {
			return nil, err
		}
		body, err := lowerStatements(stmt.Body, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.WhileStatement{Condition: condition, Body: body}, nil
	case *ast.ForStatement:
		var init ir.Statement
		var condition ir.Expression
		var update ir.Statement
		var err error

		loopSymbols := cloneKinds(symbols)
		if stmt.Init != nil {
			init, err = lowerStatement(stmt.Init, loopSymbols)
			if err != nil {
				return nil, err
			}
			if decl, ok := init.(*ir.VariableDecl); ok {
				loopSymbols[decl.Name] = inferIRKind(decl.Value)
			}
		}
		if stmt.Condition != nil {
			condition, err = lowerExpression(stmt.Condition, loopSymbols)
			if err != nil {
				return nil, err
			}
		}
		body, err := lowerStatements(stmt.Body, loopSymbols)
		if err != nil {
			return nil, err
		}
		if stmt.Update != nil {
			update, err = lowerStatement(stmt.Update, loopSymbols)
			if err != nil {
				return nil, err
			}
		}
		return &ir.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.BreakStatement:
		return &ir.BreakStatement{}, nil
	case *ast.ContinueStatement:
		return &ir.ContinueStatement{}, nil
	default:
		return nil, fmt.Errorf("unsupported statement in lowering")
	}
}

func lowerExpression(expr ast.Expression, symbols map[string]ir.ValueKind) (ir.Expression, error) {
	switch expr := expr.(type) {
	case *ast.NumberLiteral:
		return &ir.NumberLiteral{Value: expr.Value}, nil
	case *ast.BooleanLiteral:
		return &ir.BooleanLiteral{Value: expr.Value}, nil
	case *ast.NullLiteral:
		return &ir.NullLiteral{}, nil
	case *ast.UndefinedLiteral:
		return &ir.UndefinedLiteral{}, nil
	case *ast.StringLiteral:
		return &ir.StringLiteral{Value: expr.Value}, nil
	case *ast.ObjectLiteral:
		literal := &ir.ObjectLiteral{}
		for _, property := range expr.Properties {
			value, err := lowerExpression(property.Value, symbols)
			if err != nil {
				return nil, err
			}
			literal.Properties = append(literal.Properties, ir.ObjectProperty{Key: property.Key, Value: value})
		}
		return literal, nil
	case *ast.ArrayLiteral:
		literal := &ir.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := lowerExpression(element, symbols)
			if err != nil {
				return nil, err
			}
			literal.Elements = append(literal.Elements, value)
		}
		return literal, nil
	case *ast.Identifier:
		return &ir.VariableRef{Name: expr.Name, Kind: symbols[expr.Name]}, nil
	case *ast.BinaryExpression:
		left, err := lowerExpression(expr.Left, symbols)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.BinaryExpression{Operator: lowerOperator(expr.Operator), Left: left, Right: right}, nil
	case *ast.UnaryExpression:
		right, err := lowerExpression(expr.Right, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.UnaryExpression{Operator: ir.OperatorNot, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := lowerExpression(expr.Left, symbols)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols)
		if err != nil {
			return nil, err
		}
		op := ir.OperatorAnd
		if expr.Operator == ast.OperatorOr {
			op = ir.OperatorOr
		}
		return &ir.LogicalExpression{Operator: op, Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := lowerExpression(expr.Left, symbols)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.ComparisonExpression{Operator: lowerComparisonOperator(expr.Operator), Left: left, Right: right}, nil
	case *ast.IndexExpression:
		target, err := lowerExpression(expr.Target, symbols)
		if err != nil {
			return nil, err
		}
		index, err := lowerExpression(expr.Index, symbols)
		if err != nil {
			return nil, err
		}
		kind := ir.ValueString
		if variable, ok := target.(*ir.VariableRef); ok && variable.Kind == ir.ValueArray {
			kind = ir.ValueDynamic
		}
		if variable, ok := target.(*ir.VariableRef); ok && variable.Kind == ir.ValueDynamic {
			kind = ir.ValueDynamic
		}
		return &ir.IndexExpression{Target: target, Index: index, Kind: kind}, nil
	case *ast.MemberExpression:
		target, err := lowerExpression(expr.Target, symbols)
		if err != nil {
			return nil, err
		}
		return &ir.MemberExpression{Target: target, Property: expr.Property, Kind: ir.ValueDynamic}, nil
	case *ast.CallExpression:
		call := &ir.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			lowered, err := lowerExpression(arg, symbols)
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, lowered)
		}
		switch expr.Callee {
		case "readLine", "readKey":
			call.Kind = ir.ValueString
		case "print", "sleep":
			call.Kind = ""
		default:
			call.Kind = ir.ValueDynamic
		}
		return call, nil
	default:
		return nil, fmt.Errorf("unsupported expression in lowering")
	}
}

func inferIRKind(expr ir.Expression) ir.ValueKind {
	switch expr := expr.(type) {
	case *ir.NumberLiteral, *ir.BinaryExpression:
		return ir.ValueNumber
	case *ir.BooleanLiteral, *ir.ComparisonExpression:
		return ir.ValueBoolean
	case *ir.NullLiteral, *ir.UndefinedLiteral:
		return ir.ValueDynamic
	case *ir.UnaryExpression, *ir.LogicalExpression:
		return ir.ValueBoolean
	case *ir.StringLiteral:
		return ir.ValueString
	case *ir.IndexExpression:
		return expr.Kind
	case *ir.ArrayLiteral:
		return ir.ValueArray
	case *ir.ObjectLiteral:
		return ir.ValueObject
	case *ir.MemberExpression:
		return ir.ValueDynamic
	case *ir.VariableRef:
		return expr.Kind
	case *ir.CallExpression:
		return expr.Kind
	default:
		return ""
	}
}

func lowerVisibility(visibility ast.Visibility) ir.Visibility {
	if visibility == ast.VisibilityPrivate {
		return ir.VisibilityPrivate
	}
	return ir.VisibilityPublic
}

func lowerDeclarationKind(kind ast.DeclarationKind) ir.DeclarationKind {
	switch kind {
	case ast.DeclarationConst:
		return ir.DeclarationConst
	case ast.DeclarationLet:
		return ir.DeclarationLet
	default:
		return ir.DeclarationVar
	}
}

func lowerOperator(op ast.BinaryOperator) ir.BinaryOperator {
	switch op {
	case ast.OperatorAdd:
		return ir.OperatorAdd
	case ast.OperatorSub:
		return ir.OperatorSub
	case ast.OperatorMul:
		return ir.OperatorMul
	default:
		return ir.OperatorDiv
	}
}

func lowerComparisonOperator(op ast.ComparisonOperator) ir.ComparisonOperator {
	switch op {
	case ast.OperatorEq:
		return ir.OperatorEq
	case ast.OperatorNe:
		return ir.OperatorNe
	case ast.OperatorLt:
		return ir.OperatorLt
	case ast.OperatorLte:
		return ir.OperatorLte
	case ast.OperatorGt:
		return ir.OperatorGt
	default:
		return ir.OperatorGte
	}
}

func cloneKinds(input map[string]ir.ValueKind) map[string]ir.ValueKind {
	out := make(map[string]ir.ValueKind, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
