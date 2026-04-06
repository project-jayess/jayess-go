package compiler

import (
	"fmt"

	"jayess-go/ast"
)

type functionExprLowerer struct {
	counter   int
	generated []*ast.FunctionDecl
	globals   map[string]bool
}

type captureSet struct {
	names      []string
	hasThis    bool
	hasSuper   bool
	nameLookup map[string]bool
}

type superCaptureContext struct {
	ownerClass string
	baseClass  string
	isStatic   bool
}

func lowerFunctionExpressions(program *ast.Program) (*ast.Program, error) {
	l := &functionExprLowerer{
		globals: map[string]bool{
			"print":    true,
			"readLine": true,
			"readKey":  true,
			"sleep":    true,
			"Map":      true,
			"Set":      true,
			"Date":     true,
			"JSON":     true,
			"Math":     true,
			"Object":   true,
			"RegExp":   true,
			"Number":   true,
			"String":   true,
			"Array":    true,
			"console":  true,
			"process":  true,
			"path":     true,
			"fs":       true,
		},
	}

	for _, global := range program.Globals {
		l.globals[global.Name] = true
	}
	for _, ext := range program.ExternFunctions {
		l.globals[ext.Name] = true
	}
	for _, fn := range program.Functions {
		l.globals[fn.Name] = true
	}
	for _, classDecl := range program.Classes {
		l.globals[classDecl.Name] = true
	}

	for _, global := range program.Globals {
		value, err := l.rewriteExpression(global.Value, l.globals, nil)
		if err != nil {
			return nil, err
		}
		global.Value = value
	}
	for _, fn := range program.Functions {
		params, err := l.rewriteParameters(fn.Params, l.globals, nil)
		if err != nil {
			return nil, err
		}
		fn.Params = params
		scope := l.functionScope(fn.Params, fn.Body)
		scope["this"] = true
		body, err := l.rewriteStatements(fn.Body, scope, nil)
		if err != nil {
			return nil, err
		}
		fn.Body = ensureFunctionReturns(body)
	}
	for _, classDecl := range program.Classes {
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				if member.Initializer == nil {
					continue
				}
				value, err := l.rewriteExpression(member.Initializer, l.globals, nil)
				if err != nil {
					return nil, err
				}
				member.Initializer = value
			case *ast.ClassMethodDecl:
				params, err := l.rewriteParameters(member.Params, l.globals, &superCaptureContext{
					ownerClass: classDecl.Name,
					baseClass:  classDecl.SuperClass,
					isStatic:   member.Static,
				})
				if err != nil {
					return nil, err
				}
				member.Params = params
				scope := l.functionScope(member.Params, member.Body)
				scope["this"] = true
				scope["super"] = true
				body, err := l.rewriteStatements(member.Body, scope, &superCaptureContext{
					ownerClass: classDecl.Name,
					baseClass:  classDecl.SuperClass,
					isStatic:   member.Static,
				})
				if err != nil {
					return nil, err
				}
				member.Body = ensureFunctionReturns(body)
			}
		}
	}

	program.Functions = append(program.Functions, l.generated...)
	return program, nil
}

func (l *functionExprLowerer) rewriteParameters(params []ast.Parameter, scope map[string]bool, superCtx *superCaptureContext) ([]ast.Parameter, error) {
	out := make([]ast.Parameter, 0, len(params))
	for _, param := range params {
		rewritten := ast.Parameter{Name: param.Name, Rest: param.Rest}
		if param.Default != nil {
			value, err := l.rewriteExpression(param.Default, scope, superCtx)
			if err != nil {
				return nil, err
			}
			rewritten.Default = value
		}
		out = append(out, rewritten)
	}
	return out, nil
}

func (l *functionExprLowerer) functionScope(params []ast.Parameter, body []ast.Statement) map[string]bool {
	scope := make(map[string]bool, len(l.globals)+len(params)+8)
	for name := range l.globals {
		scope[name] = true
	}
	for _, param := range params {
		scope[param.Name] = true
	}
	collectDeclaredNames(body, scope)
	return scope
}

func collectDeclaredNames(statements []ast.Statement, scope map[string]bool) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			scope[stmt.Name] = true
		case *ast.IfStatement:
			collectDeclaredNames(stmt.Consequence, scope)
			collectDeclaredNames(stmt.Alternative, scope)
		case *ast.WhileStatement:
			collectDeclaredNames(stmt.Body, scope)
		case *ast.ForStatement:
			if decl, ok := stmt.Init.(*ast.VariableDecl); ok {
				scope[decl.Name] = true
			}
			collectDeclaredNames(stmt.Body, scope)
		}
	}
}

func (l *functionExprLowerer) rewriteStatements(statements []ast.Statement, scope map[string]bool, superCtx *superCaptureContext) ([]ast.Statement, error) {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		rewritten, err := l.rewriteStatement(stmt, scope, superCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, rewritten)
	}
	return out, nil
}

func (l *functionExprLowerer) rewriteStatement(stmt ast.Statement, scope map[string]bool, superCtx *superCaptureContext) (ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := l.rewriteExpression(stmt.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.VariableDecl{Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, Value: value}, nil
	case *ast.AssignmentStatement:
		target, err := l.rewriteExpression(stmt.Target, scope, superCtx)
		if err != nil {
			return nil, err
		}
		value, err := l.rewriteExpression(stmt.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.AssignmentStatement{Target: target, Value: value}, nil
	case *ast.ReturnStatement:
		value, err := l.rewriteExpression(stmt.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ReturnStatement{Value: value}, nil
	case *ast.ExpressionStatement:
		expr, err := l.rewriteExpression(stmt.Expression, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{Expression: expr}, nil
	case *ast.DeleteStatement:
		target, err := l.rewriteExpression(stmt.Target, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.DeleteStatement{Target: target}, nil
	case *ast.ThrowStatement:
		value, err := l.rewriteExpression(stmt.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ThrowStatement{Value: value}, nil
	case *ast.TryStatement:
		tryBody, err := l.rewriteStatements(stmt.TryBody, scope, superCtx)
		if err != nil {
			return nil, err
		}
		catchBody, err := l.rewriteStatements(stmt.CatchBody, scope, superCtx)
		if err != nil {
			return nil, err
		}
		finallyBody, err := l.rewriteStatements(stmt.FinallyBody, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.TryStatement{TryBody: tryBody, CatchName: stmt.CatchName, CatchBody: catchBody, FinallyBody: finallyBody}, nil
	case *ast.IfStatement:
		condition, err := l.rewriteExpression(stmt.Condition, scope, superCtx)
		if err != nil {
			return nil, err
		}
		consequence, err := l.rewriteStatements(stmt.Consequence, scope, superCtx)
		if err != nil {
			return nil, err
		}
		alternative, err := l.rewriteStatements(stmt.Alternative, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := l.rewriteExpression(stmt.Condition, scope, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteStatements(stmt.Body, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.WhileStatement{Condition: condition, Body: body}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var condition ast.Expression
		var update ast.Statement
		var err error
		loopScope := cloneDefined(scope)
		if stmt.Init != nil {
			init, err = l.rewriteStatement(stmt.Init, loopScope, superCtx)
			if err != nil {
				return nil, err
			}
			if decl, ok := init.(*ast.VariableDecl); ok {
				loopScope[decl.Name] = true
			}
		}
		if stmt.Condition != nil {
			condition, err = l.rewriteExpression(stmt.Condition, loopScope, superCtx)
			if err != nil {
				return nil, err
			}
		}
		if stmt.Update != nil {
			update, err = l.rewriteStatement(stmt.Update, loopScope, superCtx)
			if err != nil {
				return nil, err
			}
		}
		body, err := l.rewriteStatements(stmt.Body, loopScope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.ForOfStatement:
		iterable, err := l.rewriteExpression(stmt.Iterable, scope, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteStatements(stmt.Body, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForOfStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.ForInStatement:
		iterable, err := l.rewriteExpression(stmt.Iterable, scope, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteStatements(stmt.Body, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForInStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.SwitchStatement:
		discriminant, err := l.rewriteExpression(stmt.Discriminant, scope, superCtx)
		if err != nil {
			return nil, err
		}
		out := &ast.SwitchStatement{Discriminant: discriminant}
		for _, switchCase := range stmt.Cases {
			test, err := l.rewriteExpression(switchCase.Test, scope, superCtx)
			if err != nil {
				return nil, err
			}
			consequent, err := l.rewriteStatements(switchCase.Consequent, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		}
		defaultBody, err := l.rewriteStatements(stmt.Default, scope, superCtx)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return out, nil
	case *ast.BreakStatement, *ast.ContinueStatement:
		return stmt, nil
	default:
		return nil, fmt.Errorf("unsupported statement in function expression lowering")
	}
}

func (l *functionExprLowerer) rewriteExpression(expr ast.Expression, scope map[string]bool, superCtx *superCaptureContext) (ast.Expression, error) {
	switch expr := expr.(type) {
	case *ast.NumberLiteral, *ast.BooleanLiteral, *ast.NullLiteral, *ast.UndefinedLiteral, *ast.StringLiteral:
		return expr, nil
	case *ast.Identifier:
		return expr, nil
	case *ast.ThisExpression, *ast.SuperExpression, *ast.NewTargetExpression:
		return expr, nil
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{}
		for _, property := range expr.Properties {
			var keyExpr ast.Expression
			if property.Computed {
				var err error
				keyExpr, err = l.rewriteExpression(property.KeyExpr, scope, superCtx)
				if err != nil {
					return nil, err
				}
			}
			value, err := l.rewriteExpression(property.Value, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Properties = append(out.Properties, ast.ObjectProperty{Key: property.Key, KeyExpr: keyExpr, Value: value, Computed: property.Computed})
		}
		return out, nil
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := l.rewriteExpression(element, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Elements = append(out.Elements, value)
		}
		return out, nil
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{Parts: append([]string{}, expr.Parts...)}
		for _, valueExpr := range expr.Values {
			value, err := l.rewriteExpression(valueExpr, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := l.rewriteExpression(expr.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{Value: value}, nil
	case *ast.FunctionExpression:
		return l.hoistFunctionExpression(expr, scope, superCtx)
	case *ast.ClosureExpression:
		env, err := l.rewriteExpression(expr.Environment, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ClosureExpression{FunctionName: expr.FunctionName, Environment: env}, nil
	case *ast.BinaryExpression:
		left, err := l.rewriteExpression(expr.Left, scope, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.NullishCoalesceExpression:
		left, err := l.rewriteExpression(expr.Left, scope, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.NullishCoalesceExpression{Left: left, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := l.rewriteExpression(expr.Value, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{Value: value}, nil
	case *ast.InstanceofExpression:
		left, err := l.rewriteExpression(expr.Left, scope, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.InstanceofExpression{Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := l.rewriteExpression(expr.Left, scope, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := l.rewriteExpression(expr.Left, scope, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.LogicalExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.UnaryExpression:
		right, err := l.rewriteExpression(expr.Right, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: expr.Operator, Right: right}, nil
	case *ast.IndexExpression:
		target, err := l.rewriteExpression(expr.Target, scope, superCtx)
		if err != nil {
			return nil, err
		}
		index, err := l.rewriteExpression(expr.Index, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpression{Target: target, Index: index, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		target, err := l.rewriteExpression(expr.Target, scope, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}, nil
	case *ast.CallExpression:
		out := &ast.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			value, err := l.rewriteExpression(arg, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.InvokeExpression:
		callee, err := l.rewriteExpression(expr.Callee, scope, superCtx)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := l.rewriteExpression(arg, scope, superCtx)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.InvokeExpression{Callee: callee, Arguments: args, Optional: expr.Optional}, nil
	case *ast.NewExpression:
		callee, err := l.rewriteExpression(expr.Callee, scope, superCtx)
		if err != nil {
			return nil, err
		}
		out := &ast.NewExpression{Callee: callee}
		for _, arg := range expr.Arguments {
			value, err := l.rewriteExpression(arg, scope, superCtx)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported expression in function expression lowering")
	}
}

func (l *functionExprLowerer) hoistFunctionExpression(expr *ast.FunctionExpression, outerScope map[string]bool, superCtx *superCaptureContext) (ast.Expression, error) {
	name := fmt.Sprintf("__jayess_lambda_%d", l.counter)
	l.counter++
	l.globals[name] = true

	localScope := l.functionScope(expr.Params, expr.Body)
	localScope["__env"] = true
	if !expr.IsArrowFunction {
		localScope["this"] = true
	}

	captures := analyzeCaptures(expr, localScope, outerScope)
	if captures.hasSuper && superCtx == nil {
		return nil, fmt.Errorf("closures capturing super are not supported yet")
	}

	rewriterScope := cloneDefined(localScope)
	for _, name := range captures.names {
		rewriterScope[name] = true
	}
	if captures.hasThis {
		rewriterScope["this"] = true
	}
	if len(captures.names) > 0 || captures.hasThis {
		rewriterScope["__env"] = true
	}

	var body []ast.Statement
	if expr.ExpressionBody != nil {
		value, err := l.rewriteCapturedExpression(expr.ExpressionBody, rewriterScope, captures, expr.IsArrowFunction, superCtx)
		if err != nil {
			return nil, err
		}
		body = []ast.Statement{&ast.ReturnStatement{Value: value}}
	} else {
		rewritten, err := l.rewriteCapturedStatements(expr.Body, rewriterScope, captures, expr.IsArrowFunction, superCtx)
		if err != nil {
			return nil, err
		}
		body = ensureFunctionReturns(rewritten)
	}

	params, err := l.rewriteParameters(expr.Params, outerScope, superCtx)
	if err != nil {
		return nil, err
	}
	if len(captures.names) > 0 || captures.hasThis || captures.hasSuper {
		params = append([]ast.Parameter{{Name: "__env"}}, params...)
	}
	l.generated = append(l.generated, &ast.FunctionDecl{
		Visibility: ast.VisibilityPublic,
		Name:       name,
		Params:     params,
		Body:       body,
	})

	if len(captures.names) == 0 && !captures.hasThis && !captures.hasSuper {
		return &ast.Identifier{Name: name}, nil
	}
	return &ast.ClosureExpression{FunctionName: name, Environment: buildClosureEnvironment(captures, superCtx)}, nil
}

func analyzeCaptures(expr *ast.FunctionExpression, localScope map[string]bool, outerScope map[string]bool) captureSet {
	set := captureSet{nameLookup: map[string]bool{}}
	if expr.ExpressionBody != nil {
		collectCapturesFromExpression(expr.ExpressionBody, localScope, outerScope, &set)
	} else {
		collectCapturesFromStatements(expr.Body, localScope, outerScope, &set)
	}
	return set
}

func collectCapturesFromStatements(statements []ast.Statement, localScope map[string]bool, outerScope map[string]bool, captures *captureSet) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			collectCapturesFromExpression(stmt.Value, localScope, outerScope, captures)
		case *ast.AssignmentStatement:
			collectCapturesFromExpression(stmt.Target, localScope, outerScope, captures)
			collectCapturesFromExpression(stmt.Value, localScope, outerScope, captures)
		case *ast.ReturnStatement:
			collectCapturesFromExpression(stmt.Value, localScope, outerScope, captures)
		case *ast.ExpressionStatement:
			collectCapturesFromExpression(stmt.Expression, localScope, outerScope, captures)
		case *ast.DeleteStatement:
			collectCapturesFromExpression(stmt.Target, localScope, outerScope, captures)
		case *ast.IfStatement:
			collectCapturesFromExpression(stmt.Condition, localScope, outerScope, captures)
			collectCapturesFromStatements(stmt.Consequence, localScope, outerScope, captures)
			collectCapturesFromStatements(stmt.Alternative, localScope, outerScope, captures)
		case *ast.WhileStatement:
			collectCapturesFromExpression(stmt.Condition, localScope, outerScope, captures)
			collectCapturesFromStatements(stmt.Body, localScope, outerScope, captures)
		case *ast.ForStatement:
			if stmt.Init != nil {
				collectCapturesFromStatements([]ast.Statement{stmt.Init}, localScope, outerScope, captures)
			}
			if stmt.Condition != nil {
				collectCapturesFromExpression(stmt.Condition, localScope, outerScope, captures)
			}
			if stmt.Update != nil {
				collectCapturesFromStatements([]ast.Statement{stmt.Update}, localScope, outerScope, captures)
			}
			collectCapturesFromStatements(stmt.Body, localScope, outerScope, captures)
		case *ast.ForOfStatement:
			collectCapturesFromExpression(stmt.Iterable, localScope, outerScope, captures)
			collectCapturesFromStatements(stmt.Body, localScope, outerScope, captures)
		case *ast.ForInStatement:
			collectCapturesFromExpression(stmt.Iterable, localScope, outerScope, captures)
			collectCapturesFromStatements(stmt.Body, localScope, outerScope, captures)
		case *ast.SwitchStatement:
			collectCapturesFromExpression(stmt.Discriminant, localScope, outerScope, captures)
			for _, switchCase := range stmt.Cases {
				collectCapturesFromExpression(switchCase.Test, localScope, outerScope, captures)
				collectCapturesFromStatements(switchCase.Consequent, localScope, outerScope, captures)
			}
			collectCapturesFromStatements(stmt.Default, localScope, outerScope, captures)
		}
	}
}

func collectCapturesFromExpression(expr ast.Expression, localScope map[string]bool, outerScope map[string]bool, captures *captureSet) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if !localScope[expr.Name] && outerScope[expr.Name] {
			addCaptureName(captures, expr.Name)
		}
	case *ast.ThisExpression:
		if !localScope["this"] && outerScope["this"] {
			captures.hasThis = true
		}
	case *ast.SuperExpression:
		if !localScope["super"] && outerScope["super"] {
			captures.hasSuper = true
		}
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Computed {
				collectCapturesFromExpression(property.KeyExpr, localScope, outerScope, captures)
			}
			collectCapturesFromExpression(property.Value, localScope, outerScope, captures)
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			collectCapturesFromExpression(element, localScope, outerScope, captures)
		}
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			collectCapturesFromExpression(value, localScope, outerScope, captures)
		}
	case *ast.SpreadExpression:
		collectCapturesFromExpression(expr.Value, localScope, outerScope, captures)
	case *ast.FunctionExpression:
		nestedLocal := cloneDefined(localScope)
		for _, param := range expr.Params {
			nestedLocal[param.Name] = true
		}
		collectDeclaredNames(expr.Body, nestedLocal)
		if expr.ExpressionBody != nil {
			collectCapturesFromExpression(expr.ExpressionBody, nestedLocal, outerScope, captures)
		} else {
			collectCapturesFromStatements(expr.Body, nestedLocal, outerScope, captures)
		}
		return
	case *ast.ClosureExpression:
		collectCapturesFromExpression(expr.Environment, localScope, outerScope, captures)
	case *ast.BinaryExpression:
		collectCapturesFromExpression(expr.Left, localScope, outerScope, captures)
		collectCapturesFromExpression(expr.Right, localScope, outerScope, captures)
	case *ast.TypeofExpression:
		collectCapturesFromExpression(expr.Value, localScope, outerScope, captures)
	case *ast.InstanceofExpression:
		collectCapturesFromExpression(expr.Left, localScope, outerScope, captures)
		collectCapturesFromExpression(expr.Right, localScope, outerScope, captures)
	case *ast.ComparisonExpression:
		collectCapturesFromExpression(expr.Left, localScope, outerScope, captures)
		collectCapturesFromExpression(expr.Right, localScope, outerScope, captures)
	case *ast.LogicalExpression:
		collectCapturesFromExpression(expr.Left, localScope, outerScope, captures)
		collectCapturesFromExpression(expr.Right, localScope, outerScope, captures)
	case *ast.UnaryExpression:
		collectCapturesFromExpression(expr.Right, localScope, outerScope, captures)
	case *ast.IndexExpression:
		collectCapturesFromExpression(expr.Target, localScope, outerScope, captures)
		collectCapturesFromExpression(expr.Index, localScope, outerScope, captures)
	case *ast.MemberExpression:
		collectCapturesFromExpression(expr.Target, localScope, outerScope, captures)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			collectCapturesFromExpression(arg, localScope, outerScope, captures)
		}
	case *ast.InvokeExpression:
		collectCapturesFromExpression(expr.Callee, localScope, outerScope, captures)
		for _, arg := range expr.Arguments {
			collectCapturesFromExpression(arg, localScope, outerScope, captures)
		}
	case *ast.NewExpression:
		collectCapturesFromExpression(expr.Callee, localScope, outerScope, captures)
		for _, arg := range expr.Arguments {
			collectCapturesFromExpression(arg, localScope, outerScope, captures)
		}
	}
}

func addCaptureName(captures *captureSet, name string) {
	if captures.nameLookup[name] {
		return
	}
	captures.nameLookup[name] = true
	captures.names = append(captures.names, name)
}

func buildClosureEnvironment(captures captureSet, superCtx *superCaptureContext) ast.Expression {
	properties := make([]ast.ObjectProperty, 0, len(captures.names)+3)
	for _, name := range captures.names {
		properties = append(properties, ast.ObjectProperty{Key: name, Value: &ast.Identifier{Name: name}})
	}
	if captures.hasThis {
		properties = append(properties, ast.ObjectProperty{Key: "__this", Value: &ast.ThisExpression{}})
	}
	if captures.hasSuper && superCtx != nil {
		properties = append(properties, ast.ObjectProperty{Key: "__super_owner", Value: &ast.StringLiteral{Value: superCtx.ownerClass}})
		properties = append(properties, ast.ObjectProperty{Key: "__super_base", Value: &ast.StringLiteral{Value: superCtx.baseClass}})
		if !superCtx.isStatic {
			properties = append(properties, ast.ObjectProperty{Key: "__super_receiver", Value: &ast.ThisExpression{}})
		}
	}
	return &ast.ObjectLiteral{Properties: properties}
}

func (l *functionExprLowerer) rewriteCapturedStatements(statements []ast.Statement, scope map[string]bool, captures captureSet, allowLexicalThis bool, superCtx *superCaptureContext) ([]ast.Statement, error) {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		rewritten, err := l.rewriteCapturedStatement(stmt, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, rewritten)
	}
	return out, nil
}

func (l *functionExprLowerer) rewriteCapturedStatement(stmt ast.Statement, scope map[string]bool, captures captureSet, allowLexicalThis bool, superCtx *superCaptureContext) (ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := l.rewriteCapturedExpression(stmt.Value, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.VariableDecl{Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, Value: value}, nil
	case *ast.AssignmentStatement:
		target, err := l.rewriteCapturedExpression(stmt.Target, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		value, err := l.rewriteCapturedExpression(stmt.Value, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.AssignmentStatement{Target: target, Value: value}, nil
	case *ast.ReturnStatement:
		value, err := l.rewriteCapturedExpression(stmt.Value, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ReturnStatement{Value: value}, nil
	case *ast.ExpressionStatement:
		expr, err := l.rewriteCapturedExpression(stmt.Expression, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{Expression: expr}, nil
	case *ast.DeleteStatement:
		target, err := l.rewriteCapturedExpression(stmt.Target, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.DeleteStatement{Target: target}, nil
	case *ast.IfStatement:
		condition, err := l.rewriteCapturedExpression(stmt.Condition, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		consequence, err := l.rewriteCapturedStatements(stmt.Consequence, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		alternative, err := l.rewriteCapturedStatements(stmt.Alternative, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := l.rewriteCapturedExpression(stmt.Condition, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteCapturedStatements(stmt.Body, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.WhileStatement{Condition: condition, Body: body}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var condition ast.Expression
		var update ast.Statement
		var err error
		if stmt.Init != nil {
			init, err = l.rewriteCapturedStatement(stmt.Init, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
		}
		if stmt.Condition != nil {
			condition, err = l.rewriteCapturedExpression(stmt.Condition, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
		}
		if stmt.Update != nil {
			update, err = l.rewriteCapturedStatement(stmt.Update, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
		}
		body, err := l.rewriteCapturedStatements(stmt.Body, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.ForOfStatement:
		iterable, err := l.rewriteCapturedExpression(stmt.Iterable, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteCapturedStatements(stmt.Body, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForOfStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.ForInStatement:
		iterable, err := l.rewriteCapturedExpression(stmt.Iterable, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		body, err := l.rewriteCapturedStatements(stmt.Body, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ForInStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.SwitchStatement:
		discriminant, err := l.rewriteCapturedExpression(stmt.Discriminant, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		out := &ast.SwitchStatement{Discriminant: discriminant}
		for _, switchCase := range stmt.Cases {
			test, err := l.rewriteCapturedExpression(switchCase.Test, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			consequent, err := l.rewriteCapturedStatements(switchCase.Consequent, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		}
		defaultBody, err := l.rewriteCapturedStatements(stmt.Default, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return out, nil
	case *ast.BreakStatement, *ast.ContinueStatement:
		return stmt, nil
	default:
		return nil, fmt.Errorf("unsupported statement in closure rewriting")
	}
}

func (l *functionExprLowerer) rewriteCapturedExpression(expr ast.Expression, scope map[string]bool, captures captureSet, allowLexicalThis bool, superCtx *superCaptureContext) (ast.Expression, error) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if captures.nameLookup[expr.Name] {
			return &ast.MemberExpression{Target: &ast.Identifier{Name: "__env"}, Property: expr.Name}, nil
		}
		return l.rewriteExpression(expr, scope, superCtx)
	case *ast.ThisExpression:
		if allowLexicalThis && captures.hasThis {
			return &ast.MemberExpression{Target: &ast.Identifier{Name: "__env"}, Property: "__this"}, nil
		}
		return expr, nil
	case *ast.SuperExpression:
		if captures.hasSuper && superCtx != nil {
			var receiver ast.Expression
			if !superCtx.isStatic {
				receiver = &ast.MemberExpression{Target: &ast.Identifier{Name: "__env"}, Property: "__super_receiver"}
			}
			return &ast.BoundSuperExpression{
				OwnerClass: superCtx.ownerClass,
				BaseClass:  superCtx.baseClass,
				IsStatic:   superCtx.isStatic,
				Receiver:   receiver,
			}, nil
		}
		return expr, nil
	case *ast.FunctionExpression:
		hoisted, err := l.hoistFunctionExpression(expr, scope, superCtx)
		if err != nil {
			return nil, err
		}
		if closure, ok := hoisted.(*ast.ClosureExpression); ok {
			env, err := l.rewriteCapturedExpression(closure.Environment, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			return &ast.ClosureExpression{FunctionName: closure.FunctionName, Environment: env}, nil
		}
		return hoisted, nil
	case *ast.ClosureExpression:
		env, err := l.rewriteCapturedExpression(expr.Environment, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ClosureExpression{FunctionName: expr.FunctionName, Environment: env}, nil
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{}
		for _, property := range expr.Properties {
			var keyExpr ast.Expression
			if property.Computed {
				var err error
				keyExpr, err = l.rewriteCapturedExpression(property.KeyExpr, scope, captures, allowLexicalThis, superCtx)
				if err != nil {
					return nil, err
				}
			}
			value, err := l.rewriteCapturedExpression(property.Value, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Properties = append(out.Properties, ast.ObjectProperty{Key: property.Key, KeyExpr: keyExpr, Value: value, Computed: property.Computed})
		}
		return out, nil
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := l.rewriteCapturedExpression(element, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Elements = append(out.Elements, value)
		}
		return out, nil
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{Parts: append([]string{}, expr.Parts...)}
		for _, valueExpr := range expr.Values {
			value, err := l.rewriteCapturedExpression(valueExpr, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := l.rewriteCapturedExpression(expr.Value, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{Value: value}, nil
	case *ast.BinaryExpression:
		left, err := l.rewriteCapturedExpression(expr.Left, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteCapturedExpression(expr.Right, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := l.rewriteCapturedExpression(expr.Value, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{Value: value}, nil
	case *ast.NewTargetExpression:
		return expr, nil
	case *ast.InstanceofExpression:
		left, err := l.rewriteCapturedExpression(expr.Left, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteCapturedExpression(expr.Right, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.InstanceofExpression{Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := l.rewriteCapturedExpression(expr.Left, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteCapturedExpression(expr.Right, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := l.rewriteCapturedExpression(expr.Left, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		right, err := l.rewriteCapturedExpression(expr.Right, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.LogicalExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.UnaryExpression:
		right, err := l.rewriteCapturedExpression(expr.Right, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: expr.Operator, Right: right}, nil
	case *ast.IndexExpression:
		target, err := l.rewriteCapturedExpression(expr.Target, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		index, err := l.rewriteCapturedExpression(expr.Index, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpression{Target: target, Index: index, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		target, err := l.rewriteCapturedExpression(expr.Target, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}, nil
	case *ast.CallExpression:
		out := &ast.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			value, err := l.rewriteCapturedExpression(arg, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.InvokeExpression:
		callee, err := l.rewriteCapturedExpression(expr.Callee, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := l.rewriteCapturedExpression(arg, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.InvokeExpression{Callee: callee, Arguments: args, Optional: expr.Optional}, nil
	case *ast.NewExpression:
		callee, err := l.rewriteCapturedExpression(expr.Callee, scope, captures, allowLexicalThis, superCtx)
		if err != nil {
			return nil, err
		}
		out := &ast.NewExpression{Callee: callee}
		for _, arg := range expr.Arguments {
			value, err := l.rewriteCapturedExpression(arg, scope, captures, allowLexicalThis, superCtx)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	default:
		return l.rewriteExpression(expr, scope, superCtx)
	}
}

func ensureFunctionReturns(body []ast.Statement) []ast.Statement {
	if len(body) == 0 {
		return []ast.Statement{&ast.ReturnStatement{Value: &ast.UndefinedLiteral{}}}
	}
	if _, ok := body[len(body)-1].(*ast.ReturnStatement); ok {
		return body
	}
	out := append([]ast.Statement{}, body...)
	out = append(out, &ast.ReturnStatement{Value: &ast.UndefinedLiteral{}})
	return out
}

func cloneDefined(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
