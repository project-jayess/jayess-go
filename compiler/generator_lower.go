package compiler

import (
	"fmt"

	"jayess-go/ast"
)

type generatorLowerer struct {
	counter int
}

func lowerGenerators(program *ast.Program) (*ast.Program, error) {
	l := &generatorLowerer{}
	for _, fn := range program.Functions {
		var (
			body []ast.Statement
			err  error
		)
		if fn.IsGenerator {
			body, err = l.lowerGeneratorBody(fn.Body, fn.IsAsync)
		} else {
			body, err = l.lowerStatements(fn.Body, false)
		}
		if err != nil {
			return nil, err
		}
		fn.Body = body
		if fn.IsGenerator && fn.IsAsync {
			fn.IsAsync = false
		}
		fn.IsGenerator = false
	}
	return program, nil
}

func (l *generatorLowerer) lowerGeneratorBody(body []ast.Statement, isAsync bool) ([]ast.Statement, error) {
	valuesName := fmt.Sprintf("__jayess_generator_values_%d", l.counter)
	l.counter++
	rewritten, err := l.lowerStatements(body, true)
	if err != nil {
		return nil, err
	}
	out := []ast.Statement{
		&ast.VariableDecl{
			Visibility: ast.VisibilityPublic,
			Kind:       ast.DeclarationVar,
			Name:       valuesName,
			Value:      &ast.ArrayLiteral{},
		},
	}
	out = append(out, l.rewriteGeneratorReturns(rewritten, valuesName, isAsync)...)
	if !generatorBodyReturns(out) {
		out = append(out, generatorReturn(valuesName, isAsync))
	}
	return out, nil
}

func (l *generatorLowerer) lowerStatements(statements []ast.Statement, inGenerator bool) ([]ast.Statement, error) {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		rewritten, err := l.lowerStatement(stmt, inGenerator)
		if err != nil {
			return nil, err
		}
		out = append(out, rewritten)
	}
	return out, nil
}

func (l *generatorLowerer) lowerStatement(stmt ast.Statement, inGenerator bool) (ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := l.lowerExpression(stmt.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.VariableDecl{BaseNode: stmt.BaseNode, Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, TypeAnnotation: stmt.TypeAnnotation, Value: value}, nil
	case *ast.AssignmentStatement:
		target, err := l.lowerExpression(stmt.Target, inGenerator)
		if err != nil {
			return nil, err
		}
		value, err := l.lowerExpression(stmt.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.AssignmentStatement{BaseNode: stmt.BaseNode, Target: target, Operator: stmt.Operator, Value: value}, nil
	case *ast.ReturnStatement:
		if stmt.Value == nil {
			return &ast.ReturnStatement{BaseNode: stmt.BaseNode}, nil
		}
		value, err := l.lowerExpression(stmt.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ReturnStatement{BaseNode: stmt.BaseNode, Value: value}, nil
	case *ast.ExpressionStatement:
		value, err := l.lowerExpression(stmt.Expression, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{BaseNode: stmt.BaseNode, Expression: value}, nil
	case *ast.DeleteStatement:
		target, err := l.lowerExpression(stmt.Target, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.DeleteStatement{BaseNode: stmt.BaseNode, Target: target}, nil
	case *ast.ThrowStatement:
		value, err := l.lowerExpression(stmt.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ThrowStatement{BaseNode: stmt.BaseNode, Value: value}, nil
	case *ast.IfStatement:
		condition, err := l.lowerExpression(stmt.Condition, inGenerator)
		if err != nil {
			return nil, err
		}
		consequence, err := l.lowerStatements(stmt.Consequence, inGenerator)
		if err != nil {
			return nil, err
		}
		alternative, err := l.lowerStatements(stmt.Alternative, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.IfStatement{BaseNode: stmt.BaseNode, Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := l.lowerExpression(stmt.Condition, inGenerator)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.WhileStatement{BaseNode: stmt.BaseNode, Condition: condition, Body: body}, nil
	case *ast.DoWhileStatement:
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		condition, err := l.lowerExpression(stmt.Condition, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.DoWhileStatement{BaseNode: stmt.BaseNode, Body: body, Condition: condition}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var update ast.Statement
		var condition ast.Expression
		var err error
		if stmt.Init != nil {
			init, err = l.lowerStatement(stmt.Init, inGenerator)
			if err != nil {
				return nil, err
			}
		}
		if stmt.Condition != nil {
			condition, err = l.lowerExpression(stmt.Condition, inGenerator)
			if err != nil {
				return nil, err
			}
		}
		if stmt.Update != nil {
			update, err = l.lowerStatement(stmt.Update, inGenerator)
			if err != nil {
				return nil, err
			}
		}
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ForStatement{BaseNode: stmt.BaseNode, Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.ForOfStatement:
		iterable, err := l.lowerExpression(stmt.Iterable, inGenerator)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ForOfStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.ForInStatement:
		iterable, err := l.lowerExpression(stmt.Iterable, inGenerator)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ForInStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.SwitchStatement:
		discriminant, err := l.lowerExpression(stmt.Discriminant, inGenerator)
		if err != nil {
			return nil, err
		}
		out := &ast.SwitchStatement{BaseNode: stmt.BaseNode, Discriminant: discriminant}
		for _, switchCase := range stmt.Cases {
			test, err := l.lowerExpression(switchCase.Test, inGenerator)
			if err != nil {
				return nil, err
			}
			consequent, err := l.lowerStatements(switchCase.Consequent, inGenerator)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		}
		defaultBody, err := l.lowerStatements(stmt.Default, inGenerator)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return out, nil
	case *ast.BlockStatement:
		body, err := l.lowerStatements(stmt.Body, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.BlockStatement{BaseNode: stmt.BaseNode, Body: body}, nil
	case *ast.LabeledStatement:
		rewritten, err := l.lowerStatement(stmt.Statement, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.LabeledStatement{BaseNode: stmt.BaseNode, Label: stmt.Label, Statement: rewritten}, nil
	case *ast.TryStatement:
		tryBody, err := l.lowerStatements(stmt.TryBody, inGenerator)
		if err != nil {
			return nil, err
		}
		catchBody, err := l.lowerStatements(stmt.CatchBody, inGenerator)
		if err != nil {
			return nil, err
		}
		finallyBody, err := l.lowerStatements(stmt.FinallyBody, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.TryStatement{BaseNode: stmt.BaseNode, TryBody: tryBody, CatchName: stmt.CatchName, CatchBody: catchBody, FinallyBody: finallyBody}, nil
	case *ast.BreakStatement, *ast.ContinueStatement:
		return stmt, nil
	default:
		return stmt, nil
	}
}

func (l *generatorLowerer) lowerExpression(expr ast.Expression, inGenerator bool) (ast.Expression, error) {
	switch expr := expr.(type) {
	case nil:
		return nil, nil
	case *ast.FunctionExpression:
		body, err := l.lowerStatements(expr.Body, expr.IsGenerator)
		if err != nil {
			return nil, err
		}
		out := &ast.FunctionExpression{
			BaseNode:        expr.BaseNode,
			Params:          expr.Params,
			ReturnType:      expr.ReturnType,
			IsAsync:         expr.IsAsync,
			IsGenerator:     false,
			Body:            body,
			IsArrowFunction: expr.IsArrowFunction,
		}
		if expr.IsGenerator {
			out.Body, err = l.lowerGeneratorBody(expr.Body, expr.IsAsync)
			if err != nil {
				return nil, err
			}
			out.IsAsync = false
			return out, nil
		}
		if expr.ExpressionBody != nil {
			value, err := l.lowerExpression(expr.ExpressionBody, expr.IsGenerator)
			if err != nil {
				return nil, err
			}
			out.ExpressionBody = value
		}
		return out, nil
	case *ast.YieldExpression:
		if !inGenerator {
			return nil, fmt.Errorf("yield is only valid inside generator functions")
		}
		value, err := l.lowerExpression(expr.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.YieldExpression{BaseNode: expr.BaseNode, Value: value}, nil
	case *ast.CallExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg, inGenerator)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.CallExpression{BaseNode: expr.BaseNode, Callee: expr.Callee, Arguments: args}, nil
	case *ast.InvokeExpression:
		callee, err := l.lowerExpression(expr.Callee, inGenerator)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg, inGenerator)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.InvokeExpression{BaseNode: expr.BaseNode, Callee: callee, Arguments: args, Optional: expr.Optional}, nil
	case *ast.NewExpression:
		callee, err := l.lowerExpression(expr.Callee, inGenerator)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg, inGenerator)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.NewExpression{BaseNode: expr.BaseNode, Callee: callee, Arguments: args}, nil
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{BaseNode: expr.BaseNode}
		for _, property := range expr.Properties {
			rewritten := property
			if property.Computed {
				value, err := l.lowerExpression(property.KeyExpr, inGenerator)
				if err != nil {
					return nil, err
				}
				rewritten.KeyExpr = value
			}
			value, err := l.lowerExpression(property.Value, inGenerator)
			if err != nil {
				return nil, err
			}
			rewritten.Value = value
			out.Properties = append(out.Properties, rewritten)
		}
		return out, nil
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{BaseNode: expr.BaseNode}
		for _, element := range expr.Elements {
			value, err := l.lowerExpression(element, inGenerator)
			if err != nil {
				return nil, err
			}
			out.Elements = append(out.Elements, value)
		}
		return out, nil
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{BaseNode: expr.BaseNode, Parts: append([]string{}, expr.Parts...)}
		for _, valueExpr := range expr.Values {
			value, err := l.lowerExpression(valueExpr, inGenerator)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := l.lowerExpression(expr.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{BaseNode: expr.BaseNode, Value: value}, nil
	case *ast.BinaryExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.NullishCoalesceExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.NullishCoalesceExpression{BaseNode: expr.BaseNode, Left: left, Right: right}, nil
	case *ast.CommaExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.CommaExpression{BaseNode: expr.BaseNode, Left: left, Right: right}, nil
	case *ast.ConditionalExpression:
		condition, err := l.lowerExpression(expr.Condition, inGenerator)
		if err != nil {
			return nil, err
		}
		consequent, err := l.lowerExpression(expr.Consequent, inGenerator)
		if err != nil {
			return nil, err
		}
		alternative, err := l.lowerExpression(expr.Alternative, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ConditionalExpression{BaseNode: expr.BaseNode, Condition: condition, Consequent: consequent, Alternative: alternative}, nil
	case *ast.UnaryExpression:
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := l.lowerExpression(expr.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{BaseNode: expr.BaseNode, Value: value}, nil
	case *ast.TypeCheckExpression:
		value, err := l.lowerExpression(expr.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.TypeCheckExpression{BaseNode: expr.BaseNode, Value: value, TypeAnnotation: expr.TypeAnnotation}, nil
	case *ast.InstanceofExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.InstanceofExpression{BaseNode: expr.BaseNode, Left: left, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.LogicalExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := l.lowerExpression(expr.Left, inGenerator)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.IndexExpression:
		target, err := l.lowerExpression(expr.Target, inGenerator)
		if err != nil {
			return nil, err
		}
		index, err := l.lowerExpression(expr.Index, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpression{BaseNode: expr.BaseNode, Target: target, Index: index, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		target, err := l.lowerExpression(expr.Target, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{BaseNode: expr.BaseNode, Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}, nil
	case *ast.ClosureExpression:
		env, err := l.lowerExpression(expr.Environment, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.ClosureExpression{BaseNode: expr.BaseNode, FunctionName: expr.FunctionName, Environment: env}, nil
	case *ast.CastExpression:
		value, err := l.lowerExpression(expr.Value, inGenerator)
		if err != nil {
			return nil, err
		}
		return &ast.CastExpression{BaseNode: expr.BaseNode, Value: value, TypeAnnotation: expr.TypeAnnotation}, nil
	default:
		return expr, nil
	}
}

func (l *generatorLowerer) rewriteGeneratorReturns(statements []ast.Statement, valuesName string, isAsync bool) []ast.Statement {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.ReturnStatement:
			out = append(out, generatorReturn(valuesName, isAsync))
		case *ast.ExpressionStatement:
			out = append(out, &ast.ExpressionStatement{BaseNode: stmt.BaseNode, Expression: l.rewriteYieldExpression(stmt.Expression, valuesName)})
		case *ast.VariableDecl:
			out = append(out, &ast.VariableDecl{BaseNode: stmt.BaseNode, Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, TypeAnnotation: stmt.TypeAnnotation, Value: l.rewriteYieldExpression(stmt.Value, valuesName)})
		case *ast.AssignmentStatement:
			out = append(out, &ast.AssignmentStatement{BaseNode: stmt.BaseNode, Target: l.rewriteYieldExpression(stmt.Target, valuesName), Operator: stmt.Operator, Value: l.rewriteYieldExpression(stmt.Value, valuesName)})
		case *ast.DeleteStatement:
			out = append(out, &ast.DeleteStatement{BaseNode: stmt.BaseNode, Target: l.rewriteYieldExpression(stmt.Target, valuesName)})
		case *ast.ThrowStatement:
			out = append(out, &ast.ThrowStatement{BaseNode: stmt.BaseNode, Value: l.rewriteYieldExpression(stmt.Value, valuesName)})
		case *ast.IfStatement:
			out = append(out, &ast.IfStatement{
				BaseNode:    stmt.BaseNode,
				Condition:   l.rewriteYieldExpression(stmt.Condition, valuesName),
				Consequence: l.rewriteGeneratorReturns(stmt.Consequence, valuesName, isAsync),
				Alternative: l.rewriteGeneratorReturns(stmt.Alternative, valuesName, isAsync),
			})
		case *ast.WhileStatement:
			out = append(out, &ast.WhileStatement{BaseNode: stmt.BaseNode, Condition: l.rewriteYieldExpression(stmt.Condition, valuesName), Body: l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync)})
		case *ast.DoWhileStatement:
			out = append(out, &ast.DoWhileStatement{BaseNode: stmt.BaseNode, Body: l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync), Condition: l.rewriteYieldExpression(stmt.Condition, valuesName)})
		case *ast.ForStatement:
			var init ast.Statement
			var update ast.Statement
			if stmt.Init != nil {
				init = l.rewriteGeneratorReturns([]ast.Statement{stmt.Init}, valuesName, isAsync)[0]
			}
			if stmt.Update != nil {
				update = l.rewriteGeneratorReturns([]ast.Statement{stmt.Update}, valuesName, isAsync)[0]
			}
			out = append(out, &ast.ForStatement{
				BaseNode:  stmt.BaseNode,
				Init:      init,
				Condition: l.rewriteYieldExpression(stmt.Condition, valuesName),
				Update:    update,
				Body:      l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync),
			})
		case *ast.ForOfStatement:
			out = append(out, &ast.ForOfStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: l.rewriteYieldExpression(stmt.Iterable, valuesName), Body: l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync)})
		case *ast.ForInStatement:
			out = append(out, &ast.ForInStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: l.rewriteYieldExpression(stmt.Iterable, valuesName), Body: l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync)})
		case *ast.SwitchStatement:
			rewritten := &ast.SwitchStatement{BaseNode: stmt.BaseNode, Discriminant: l.rewriteYieldExpression(stmt.Discriminant, valuesName), Default: l.rewriteGeneratorReturns(stmt.Default, valuesName, isAsync)}
			for _, switchCase := range stmt.Cases {
				rewritten.Cases = append(rewritten.Cases, ast.SwitchCase{Test: l.rewriteYieldExpression(switchCase.Test, valuesName), Consequent: l.rewriteGeneratorReturns(switchCase.Consequent, valuesName, isAsync)})
			}
			out = append(out, rewritten)
		case *ast.BlockStatement:
			out = append(out, &ast.BlockStatement{BaseNode: stmt.BaseNode, Body: l.rewriteGeneratorReturns(stmt.Body, valuesName, isAsync)})
		case *ast.LabeledStatement:
			rewritten := l.rewriteGeneratorReturns([]ast.Statement{stmt.Statement}, valuesName, isAsync)
			out = append(out, &ast.LabeledStatement{BaseNode: stmt.BaseNode, Label: stmt.Label, Statement: rewritten[0]})
		case *ast.TryStatement:
			out = append(out, &ast.TryStatement{
				BaseNode:    stmt.BaseNode,
				TryBody:     l.rewriteGeneratorReturns(stmt.TryBody, valuesName, isAsync),
				CatchName:   stmt.CatchName,
				CatchBody:   l.rewriteGeneratorReturns(stmt.CatchBody, valuesName, isAsync),
				FinallyBody: l.rewriteGeneratorReturns(stmt.FinallyBody, valuesName, isAsync),
			})
		default:
			out = append(out, stmt)
		}
	}
	return out
}

func (l *generatorLowerer) rewriteYieldExpression(expr ast.Expression, valuesName string) ast.Expression {
	switch expr := expr.(type) {
	case nil:
		return nil
	case *ast.YieldExpression:
		value := expr.Value
		if value == nil {
			value = &ast.UndefinedLiteral{}
		}
		return &ast.CommaExpression{
			BaseNode: expr.BaseNode,
			Left: &ast.CallExpression{
				BaseNode:  expr.BaseNode,
				Callee:    "__jayess_array_push",
				Arguments: []ast.Expression{&ast.Identifier{BaseNode: expr.BaseNode, Name: valuesName}, l.rewriteYieldExpression(value, valuesName)},
			},
			Right: &ast.UndefinedLiteral{BaseNode: expr.BaseNode},
		}
	case *ast.CallExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			args = append(args, l.rewriteYieldExpression(arg, valuesName))
		}
		return &ast.CallExpression{BaseNode: expr.BaseNode, Callee: expr.Callee, Arguments: args}
	case *ast.InvokeExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			args = append(args, l.rewriteYieldExpression(arg, valuesName))
		}
		return &ast.InvokeExpression{BaseNode: expr.BaseNode, Callee: l.rewriteYieldExpression(expr.Callee, valuesName), Arguments: args, Optional: expr.Optional}
	case *ast.NewExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			args = append(args, l.rewriteYieldExpression(arg, valuesName))
		}
		return &ast.NewExpression{BaseNode: expr.BaseNode, Callee: l.rewriteYieldExpression(expr.Callee, valuesName), Arguments: args}
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{BaseNode: expr.BaseNode}
		for _, property := range expr.Properties {
			rewritten := property
			if property.Computed {
				rewritten.KeyExpr = l.rewriteYieldExpression(property.KeyExpr, valuesName)
			}
			rewritten.Value = l.rewriteYieldExpression(property.Value, valuesName)
			out.Properties = append(out.Properties, rewritten)
		}
		return out
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{BaseNode: expr.BaseNode}
		for _, element := range expr.Elements {
			out.Elements = append(out.Elements, l.rewriteYieldExpression(element, valuesName))
		}
		return out
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{BaseNode: expr.BaseNode, Parts: append([]string{}, expr.Parts...)}
		for _, value := range expr.Values {
			out.Values = append(out.Values, l.rewriteYieldExpression(value, valuesName))
		}
		return out
	case *ast.SpreadExpression:
		return &ast.SpreadExpression{BaseNode: expr.BaseNode, Value: l.rewriteYieldExpression(expr.Value, valuesName)}
	case *ast.BinaryExpression:
		return &ast.BinaryExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.NullishCoalesceExpression:
		return &ast.NullishCoalesceExpression{BaseNode: expr.BaseNode, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.CommaExpression:
		return &ast.CommaExpression{BaseNode: expr.BaseNode, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.ConditionalExpression:
		return &ast.ConditionalExpression{BaseNode: expr.BaseNode, Condition: l.rewriteYieldExpression(expr.Condition, valuesName), Consequent: l.rewriteYieldExpression(expr.Consequent, valuesName), Alternative: l.rewriteYieldExpression(expr.Alternative, valuesName)}
	case *ast.UnaryExpression:
		return &ast.UnaryExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.TypeofExpression:
		return &ast.TypeofExpression{BaseNode: expr.BaseNode, Value: l.rewriteYieldExpression(expr.Value, valuesName)}
	case *ast.TypeCheckExpression:
		return &ast.TypeCheckExpression{BaseNode: expr.BaseNode, Value: l.rewriteYieldExpression(expr.Value, valuesName), TypeAnnotation: expr.TypeAnnotation}
	case *ast.InstanceofExpression:
		return &ast.InstanceofExpression{BaseNode: expr.BaseNode, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.LogicalExpression:
		return &ast.LogicalExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.ComparisonExpression:
		return &ast.ComparisonExpression{BaseNode: expr.BaseNode, Operator: expr.Operator, Left: l.rewriteYieldExpression(expr.Left, valuesName), Right: l.rewriteYieldExpression(expr.Right, valuesName)}
	case *ast.IndexExpression:
		return &ast.IndexExpression{BaseNode: expr.BaseNode, Target: l.rewriteYieldExpression(expr.Target, valuesName), Index: l.rewriteYieldExpression(expr.Index, valuesName), Optional: expr.Optional}
	case *ast.MemberExpression:
		return &ast.MemberExpression{BaseNode: expr.BaseNode, Target: l.rewriteYieldExpression(expr.Target, valuesName), Property: expr.Property, Private: expr.Private, Optional: expr.Optional}
	case *ast.AwaitExpression:
		return &ast.AwaitExpression{BaseNode: expr.BaseNode, Value: l.rewriteYieldExpression(expr.Value, valuesName)}
	case *ast.ClosureExpression:
		return &ast.ClosureExpression{BaseNode: expr.BaseNode, FunctionName: expr.FunctionName, Environment: l.rewriteYieldExpression(expr.Environment, valuesName)}
	case *ast.CastExpression:
		return &ast.CastExpression{BaseNode: expr.BaseNode, Value: l.rewriteYieldExpression(expr.Value, valuesName), TypeAnnotation: expr.TypeAnnotation}
	default:
		return expr
	}
}

func generatorReturn(valuesName string, isAsync bool) ast.Statement {
	callee := "__jayess_std_iterator_from"
	if isAsync {
		callee = "__jayess_std_async_iterator_from"
	}
	return &ast.ReturnStatement{
		Value: &ast.CallExpression{
			Callee:    callee,
			Arguments: []ast.Expression{&ast.Identifier{Name: valuesName}},
		},
	}
}

func generatorBodyReturns(statements []ast.Statement) bool {
	if len(statements) == 0 {
		return false
	}
	_, ok := statements[len(statements)-1].(*ast.ReturnStatement)
	return ok
}
