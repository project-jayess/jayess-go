package compiler

import (
	"fmt"
	"sort"

	"jayess-go/ast"
)

type loweredClassInfo struct {
	name                 string
	base                 string
	constructor          *ast.ClassMethodDecl
	methods              map[string]int
	privateMethods       map[string]bool
	staticMethods        map[string]int
	privateStaticMethods map[string]bool
	staticFields         map[string]bool
	privateStaticFields  map[string]bool
	instanceFields       []*ast.ClassFieldDecl
	privateFields        []*ast.ClassFieldDecl
}

type classRewriteContext struct {
	info       *loweredClassInfo
	classes    map[string]*loweredClassInfo
	isStatic   bool
	dispatches map[dispatchSignature]bool
}

type dispatchSignature struct {
	method   string
	argCount int
}

type callBinding struct {
	callee         string
	receiver       ast.Expression
	dispatchMethod string
}

func lowerClasses(program *ast.Program) (*ast.Program, error) {
	classes := map[string]*loweredClassInfo{}
	dispatches := map[dispatchSignature]bool{}
	for _, classDecl := range program.Classes {
		info, err := collectClassInfo(classDecl)
		if err != nil {
			return nil, err
		}
		classes[classDecl.Name] = info
	}

	out := &ast.Program{
		ExternFunctions: append([]*ast.ExternFunctionDecl{}, program.ExternFunctions...),
	}

	globalBindings := map[string]string{}
	globalCalls := map[string]callBinding{}
	for _, classDecl := range program.Classes {
		globalBindings[classDecl.Name] = classDecl.Name
	}

	for _, classDecl := range program.Classes {
		info := classes[classDecl.Name]
		for _, field := range info.instanceFields {
			_ = field
		}
		for _, field := range info.privateFields {
			_ = field
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				if member.Static {
					value := member.Initializer
					if value == nil {
						value = &ast.UndefinedLiteral{}
					}
					rewritten, err := rewriteExpression(value, globalBindings, globalCalls, nil, classes)
					if err != nil {
						return nil, err
					}
					out.Globals = append(out.Globals, &ast.VariableDecl{
						Visibility: ast.VisibilityPublic,
						Kind:       ast.DeclarationVar,
						Name:       staticMemberSymbol(classDecl.Name, member.Name, member.Private),
						Value:      rewritten,
					})
				}
			}
		}

		constructor, err := emitClassConstructor(info, classes, dispatches)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, constructor)

		for _, member := range classDecl.Members {
			method, ok := member.(*ast.ClassMethodDecl)
			if !ok || method.IsConstructor {
				continue
			}
			emitted, err := emitClassMethod(info, method, classes, dispatches)
			if err != nil {
				return nil, err
			}
			out.Functions = append(out.Functions, emitted)
		}
	}

	for _, global := range program.Globals {
		rewritten, err := rewriteExpression(global.Value, globalBindings, globalCalls, nil, classes)
		if err != nil {
			return nil, err
		}
		if binding, ok, err := inferCallBinding(global.Value, globalBindings, globalCalls, nil, classes); err != nil {
			return nil, err
		} else if ok {
			globalCalls[global.Name] = binding
			rewritten = &ast.UndefinedLiteral{}
		}
		out.Globals = append(out.Globals, &ast.VariableDecl{
			Visibility: global.Visibility,
			Kind:       global.Kind,
			Name:       global.Name,
			Value:      rewritten,
		})
		if className := inferClassBinding(rewritten, globalBindings, classes); className != "" {
			globalBindings[global.Name] = className
		}
	}

	for _, fn := range program.Functions {
		rewritten, err := rewriteFunction(fn, globalBindings, globalCalls, classes, dispatches)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, rewritten)
	}
	for _, helper := range emitDispatchHelpers(classes, dispatches) {
		out.Functions = append(out.Functions, helper)
	}

	return out, nil
}

func collectClassInfo(classDecl *ast.ClassDecl) (*loweredClassInfo, error) {
	info := &loweredClassInfo{
		name:                 classDecl.Name,
		base:                 classDecl.SuperClass,
		methods:              map[string]int{},
		privateMethods:       map[string]bool{},
		staticMethods:        map[string]int{},
		privateStaticMethods: map[string]bool{},
		staticFields:         map[string]bool{},
		privateStaticFields:  map[string]bool{},
	}

	for _, member := range classDecl.Members {
		switch member := member.(type) {
		case *ast.ClassFieldDecl:
			if member.Static {
				if member.Private {
					info.privateStaticFields[member.Name] = true
				} else {
					info.staticFields[member.Name] = true
				}
			} else if member.Private {
				info.privateFields = append(info.privateFields, member)
			} else {
				info.instanceFields = append(info.instanceFields, member)
			}
		case *ast.ClassMethodDecl:
			if member.IsConstructor {
				if info.constructor != nil {
					return nil, fmt.Errorf("class %s declares multiple constructors", classDecl.Name)
				}
				info.constructor = member
				continue
			}
			switch {
			case member.Static && member.Private:
				info.privateStaticMethods[member.Name] = true
			case member.Static:
				info.staticMethods[member.Name] = len(member.Params)
			case member.Private:
				info.privateMethods[member.Name] = true
			default:
				info.methods[member.Name] = len(member.Params)
			}
		}
	}

	return info, nil
}

func emitClassConstructor(info *loweredClassInfo, classes map[string]*loweredClassInfo, dispatches map[dispatchSignature]bool) (*ast.FunctionDecl, error) {
	if info.constructor == nil {
		body := []ast.Statement{
			&ast.VariableDecl{
				Visibility: ast.VisibilityPublic,
				Kind:       ast.DeclarationVar,
				Name:       "__self",
				Value:      constructorInitialValue(info),
			},
		}
		if info.base != "" {
			body = append(body, implicitSuperInit(info.base))
		}
		body = append(body, setClassTagStatement(info.name))
		body = append(body, setClassMarkerStatements(info, classes)...)
		body = append(body, instanceFieldInitializers(info)...)
		body = append(body, &ast.ReturnStatement{Value: &ast.Identifier{Name: "__self"}})
		return &ast.FunctionDecl{
			Visibility: ast.VisibilityPublic,
			Name:       info.name,
			Body:       body,
		}, nil
	}

	ctx := &classRewriteContext{info: info, classes: classes, dispatches: dispatches}
	body := []ast.Statement{
		&ast.VariableDecl{
			Visibility: ast.VisibilityPublic,
			Kind:       ast.DeclarationVar,
			Name:       "__self",
			Value:      constructorInitialValue(info),
		},
	}

	superIndex := firstDirectSuperCallIndex(info.constructor.Body)
	if info.base == "" {
		body = append(body, setClassTagStatement(info.name))
		body = append(body, instanceFieldInitializers(info)...)
		rewritten, err := rewriteStatements(info.constructor.Body, map[string]string{}, map[string]callBinding{}, ctx, classes)
		if err != nil {
			return nil, err
		}
		body = append(body, rewritten...)
	} else if superIndex >= 0 {
		before, err := rewriteStatements(info.constructor.Body[:superIndex+1], map[string]string{}, map[string]callBinding{}, ctx, classes)
		if err != nil {
			return nil, err
		}
		after, err := rewriteStatements(info.constructor.Body[superIndex+1:], map[string]string{}, map[string]callBinding{}, ctx, classes)
		if err != nil {
			return nil, err
		}
		body = append(body, before...)
		body = append(body, setClassTagStatement(info.name))
		body = append(body, setClassMarkerStatements(info, classes)...)
		body = append(body, instanceFieldInitializers(info)...)
		body = append(body, after...)
	} else {
		body = append(body, implicitSuperInit(info.base))
		body = append(body, setClassTagStatement(info.name))
		body = append(body, setClassMarkerStatements(info, classes)...)
		body = append(body, instanceFieldInitializers(info)...)
		rewritten, err := rewriteStatements(info.constructor.Body, map[string]string{}, map[string]callBinding{}, ctx, classes)
		if err != nil {
			return nil, err
		}
		body = append(body, rewritten...)
	}

	body = append(body, &ast.ReturnStatement{Value: &ast.Identifier{Name: "__self"}})
	params, err := rewriteParameters(info.constructor.Params, map[string]string{}, map[string]callBinding{}, ctx, classes)
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDecl{
		Visibility: ast.VisibilityPublic,
		Name:       info.name,
		Params:     params,
		Body:       body,
	}, nil
}

func emitClassMethod(info *loweredClassInfo, method *ast.ClassMethodDecl, classes map[string]*loweredClassInfo, dispatches map[dispatchSignature]bool) (*ast.FunctionDecl, error) {
	ctx := &classRewriteContext{info: info, classes: classes, isStatic: method.Static, dispatches: dispatches}
	bindings := map[string]string{}
	params, err := rewriteParameters(method.Params, bindings, map[string]callBinding{}, ctx, classes)
	if err != nil {
		return nil, err
	}
	if !method.Static {
		params = append([]ast.Parameter{{Name: "__self"}}, params...)
	}
	body, err := rewriteStatements(method.Body, bindings, map[string]callBinding{}, ctx, classes)
	if err != nil {
		return nil, err
	}
	name := methodSymbol(info.name, method.Name, method.Private)
	if method.Static {
		name = staticMemberSymbol(info.name, method.Name, method.Private)
	}
	return &ast.FunctionDecl{
		Visibility: ast.VisibilityPublic,
		Name:       name,
		Params:     params,
		Body:       body,
	}, nil
}

func rewriteFunction(fn *ast.FunctionDecl, globalBindings map[string]string, globalCalls map[string]callBinding, classes map[string]*loweredClassInfo, dispatches map[dispatchSignature]bool) (*ast.FunctionDecl, error) {
	bindings := cloneBindings(globalBindings)
	calls := cloneCallBindings(globalCalls)
	params, err := rewriteParameters(fn.Params, bindings, calls, &classRewriteContext{classes: classes, dispatches: dispatches}, classes)
	if err != nil {
		return nil, err
	}
	body, err := rewriteStatements(fn.Body, bindings, calls, &classRewriteContext{classes: classes, dispatches: dispatches}, classes)
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDecl{
		Visibility: fn.Visibility,
		Name:       fn.Name,
		Params:     params,
		Body:       body,
	}, nil
}

func rewriteParameters(params []ast.Parameter, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) ([]ast.Parameter, error) {
	out := make([]ast.Parameter, 0, len(params))
	for _, param := range params {
		rewritten := ast.Parameter{Name: param.Name, Rest: param.Rest}
		if param.Default != nil {
			value, err := rewriteExpression(param.Default, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			rewritten.Default = value
		}
		out = append(out, rewritten)
	}
	return out, nil
}

func rewriteStatements(statements []ast.Statement, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) ([]ast.Statement, error) {
	local := cloneBindings(bindings)
	localCalls := cloneCallBindings(callBindings)
	var out []ast.Statement
	for _, original := range statements {
		rewritten, err := rewriteStatement(original, local, localCalls, ctx, classes)
		if err != nil {
			return nil, err
		}
		out = append(out, rewritten)
		switch rewrittenStmt := rewritten.(type) {
		case *ast.VariableDecl:
			if className := inferClassBinding(rewrittenStmt.Value, local, classes); className != "" {
				local[rewrittenStmt.Name] = className
			} else {
				delete(local, rewrittenStmt.Name)
			}
			if originalDecl, ok := original.(*ast.VariableDecl); ok {
				if binding, ok, err := inferCallBinding(originalDecl.Value, local, localCalls, ctx, classes); err != nil {
					return nil, err
				} else if ok {
					localCalls[rewrittenStmt.Name] = binding
				} else {
					delete(localCalls, rewrittenStmt.Name)
				}
			}
		case *ast.AssignmentStatement:
			if ident, ok := rewrittenStmt.Target.(*ast.Identifier); ok {
				if className := inferClassBinding(rewrittenStmt.Value, local, classes); className != "" {
					local[ident.Name] = className
				} else {
					delete(local, ident.Name)
				}
				if originalAssign, ok := original.(*ast.AssignmentStatement); ok {
					if binding, ok, err := inferCallBinding(originalAssign.Value, local, localCalls, ctx, classes); err != nil {
						return nil, err
					} else if ok {
						localCalls[ident.Name] = binding
					} else {
						delete(localCalls, ident.Name)
					}
				}
			}
		}
	}
	return out, nil
}

func rewriteStatement(stmt ast.Statement, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		if _, ok, err := inferCallBinding(stmt.Value, bindings, callBindings, ctx, classes); err != nil {
			return nil, err
		} else if ok {
			return &ast.VariableDecl{Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, Value: &ast.UndefinedLiteral{}}, nil
		}
		value, err := rewriteExpression(stmt.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.VariableDecl{Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, Value: value}, nil
	case *ast.AssignmentStatement:
		target, err := rewriteExpression(stmt.Target, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		if _, ok, err := inferCallBinding(stmt.Value, bindings, callBindings, ctx, classes); err != nil {
			return nil, err
		} else if ok {
			return &ast.AssignmentStatement{Target: target, Value: &ast.UndefinedLiteral{}}, nil
		}
		value, err := rewriteExpression(stmt.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.AssignmentStatement{Target: target, Value: value}, nil
	case *ast.ReturnStatement:
		value, err := rewriteExpression(stmt.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ReturnStatement{Value: value}, nil
	case *ast.ExpressionStatement:
		expr, err := rewriteExpression(stmt.Expression, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ExpressionStatement{Expression: expr}, nil
	case *ast.DeleteStatement:
		target, err := rewriteExpression(stmt.Target, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.DeleteStatement{Target: target}, nil
	case *ast.ThrowStatement:
		value, err := rewriteExpression(stmt.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ThrowStatement{Value: value}, nil
	case *ast.TryStatement:
		tryBody, err := rewriteStatements(stmt.TryBody, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		catchBody, err := rewriteStatements(stmt.CatchBody, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		finallyBody, err := rewriteStatements(stmt.FinallyBody, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.TryStatement{TryBody: tryBody, CatchName: stmt.CatchName, CatchBody: catchBody, FinallyBody: finallyBody}, nil
	case *ast.IfStatement:
		condition, err := rewriteExpression(stmt.Condition, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		consequence, err := rewriteStatements(stmt.Consequence, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		alternative, err := rewriteStatements(stmt.Alternative, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := rewriteExpression(stmt.Condition, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		body, err := rewriteStatements(stmt.Body, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.WhileStatement{Condition: condition, Body: body}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var condition ast.Expression
		var update ast.Statement
		var err error
		loopBindings := cloneBindings(bindings)
		if stmt.Init != nil {
			init, err = rewriteStatement(stmt.Init, loopBindings, cloneCallBindings(callBindings), ctx, classes)
			if err != nil {
				return nil, err
			}
			if decl, ok := init.(*ast.VariableDecl); ok {
				if className := inferClassBinding(decl.Value, loopBindings, classes); className != "" {
					loopBindings[decl.Name] = className
				}
			}
		}
		if stmt.Condition != nil {
			condition, err = rewriteExpression(stmt.Condition, loopBindings, cloneCallBindings(callBindings), ctx, classes)
			if err != nil {
				return nil, err
			}
		}
		body, err := rewriteStatements(stmt.Body, loopBindings, cloneCallBindings(callBindings), ctx, classes)
		if err != nil {
			return nil, err
		}
		if stmt.Update != nil {
			update, err = rewriteStatement(stmt.Update, loopBindings, cloneCallBindings(callBindings), ctx, classes)
			if err != nil {
				return nil, err
			}
		}
		return &ast.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.ForOfStatement:
		iterable, err := rewriteExpression(stmt.Iterable, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		body, err := rewriteStatements(stmt.Body, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ForOfStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.ForInStatement:
		iterable, err := rewriteExpression(stmt.Iterable, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		body, err := rewriteStatements(stmt.Body, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ForInStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}, nil
	case *ast.SwitchStatement:
		discriminant, err := rewriteExpression(stmt.Discriminant, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		out := &ast.SwitchStatement{Discriminant: discriminant}
		for _, switchCase := range stmt.Cases {
			test, err := rewriteExpression(switchCase.Test, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			consequent, err := rewriteStatements(switchCase.Consequent, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		}
		defaultBody, err := rewriteStatements(stmt.Default, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return out, nil
	case *ast.BreakStatement, *ast.ContinueStatement:
		return stmt, nil
	default:
		return nil, fmt.Errorf("unsupported statement during class lowering")
	}
}

func rewriteExpression(expr ast.Expression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Expression, error) {
	switch expr := expr.(type) {
	case *ast.NumberLiteral, *ast.BooleanLiteral, *ast.NullLiteral, *ast.UndefinedLiteral, *ast.StringLiteral:
		return expr, nil
	case *ast.Identifier:
		return expr, nil
	case *ast.ThisExpression:
		if ctx == nil || ctx.info == nil {
			return expr, nil
		}
		if ctx.isStatic {
			return &ast.Identifier{Name: ctx.info.name}, nil
		}
		return &ast.Identifier{Name: "__self"}, nil
	case *ast.SuperExpression:
		if ctx == nil || ctx.info == nil || ctx.info.base == "" {
			return nil, fmt.Errorf("super is only valid inside derived class methods")
		}
		return expr, nil
	case *ast.NewTargetExpression:
		return expr, nil
	case *ast.BoundSuperExpression:
		return expr, nil
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{}
		for _, property := range expr.Properties {
			var keyExpr ast.Expression
			if property.Computed {
				var err error
				keyExpr, err = rewriteExpression(property.KeyExpr, bindings, callBindings, ctx, classes)
				if err != nil {
					return nil, err
				}
			}
			value, err := rewriteExpression(property.Value, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			out.Properties = append(out.Properties, ast.ObjectProperty{Key: property.Key, KeyExpr: keyExpr, Value: value, Computed: property.Computed})
		}
		return out, nil
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := rewriteExpression(element, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			out.Elements = append(out.Elements, value)
		}
		return out, nil
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{Parts: append([]string{}, expr.Parts...)}
		for _, valueExpr := range expr.Values {
			value, err := rewriteExpression(valueExpr, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := rewriteExpression(expr.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{Value: value}, nil
	case *ast.BinaryExpression:
		left, err := rewriteExpression(expr.Left, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := rewriteExpression(expr.Value, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{Value: value}, nil
	case *ast.InstanceofExpression:
		left, err := rewriteExpression(expr.Left, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.InstanceofExpression{Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := rewriteExpression(expr.Left, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := rewriteExpression(expr.Left, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.LogicalExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.NullishCoalesceExpression:
		left, err := rewriteExpression(expr.Left, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.NullishCoalesceExpression{Left: left, Right: right}, nil
	case *ast.UnaryExpression:
		right, err := rewriteExpression(expr.Right, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: expr.Operator, Right: right}, nil
	case *ast.IndexExpression:
		target, err := rewriteExpression(expr.Target, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		index, err := rewriteExpression(expr.Index, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpression{Target: target, Index: index, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		return rewriteMemberExpression(expr, bindings, callBindings, ctx, classes)
	case *ast.CallExpression:
		if binding, ok := callBindings[expr.Callee]; ok {
			args := make([]ast.Expression, 0, len(expr.Arguments)+1)
			for _, arg := range expr.Arguments {
				value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
				if err != nil {
					return nil, err
				}
				args = append(args, value)
			}
			if binding.dispatchMethod != "" {
				receiver := cloneExpression(binding.receiver)
				if receiver != nil {
					var err error
					receiver, err = rewriteExpression(receiver, bindings, callBindings, ctx, classes)
					if err != nil {
						return nil, err
					}
				}
				return buildInstanceDispatchCall(binding.dispatchMethod, receiver, args, ctx.dispatches), nil
			}
			if binding.receiver != nil {
				receiver := cloneExpression(binding.receiver)
				receiver, err := rewriteExpression(receiver, bindings, callBindings, ctx, classes)
				if err != nil {
					return nil, err
				}
				args = append([]ast.Expression{receiver}, args...)
			}
			return &ast.CallExpression{Callee: binding.callee, Arguments: args}, nil
		}
		out := &ast.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.InvokeExpression:
		return rewriteInvokeExpression(expr, bindings, callBindings, ctx, classes)
	case *ast.NewExpression:
		return rewriteNewExpression(expr, bindings, callBindings, ctx, classes)
	case *ast.ClosureExpression:
		env, err := rewriteExpression(expr.Environment, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.ClosureExpression{FunctionName: expr.FunctionName, Environment: env}, nil
	default:
		return nil, fmt.Errorf("unsupported expression during class lowering")
	}
}

func rewriteNewExpression(expr *ast.NewExpression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Expression, error) {
	callee, err := rewriteExpression(expr.Callee, bindings, callBindings, ctx, classes)
	if err != nil {
		return nil, err
	}
	ident, ok := callee.(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("dynamic constructors are not supported")
	}
	switch ident.Name {
	case "Map":
		if len(expr.Arguments) != 0 {
			return nil, fmt.Errorf("Map constructor expects 0 arguments")
		}
		return &ast.CallExpression{Callee: "__jayess_std_map_new"}, nil
	case "Set":
		if len(expr.Arguments) != 0 {
			return nil, fmt.Errorf("Set constructor expects 0 arguments")
		}
		return &ast.CallExpression{Callee: "__jayess_std_set_new"}, nil
	case "Date":
		call := &ast.CallExpression{Callee: "__jayess_std_date_new"}
		if len(expr.Arguments) > 1 {
			return nil, fmt.Errorf("Date constructor expects at most 1 argument")
		}
		for _, arg := range expr.Arguments {
			value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, value)
		}
		return call, nil
	case "RegExp":
		call := &ast.CallExpression{Callee: "__jayess_std_regexp_new"}
		if len(expr.Arguments) > 2 {
			return nil, fmt.Errorf("RegExp constructor expects at most 2 arguments")
		}
		for _, arg := range expr.Arguments {
			value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, value)
		}
		return call, nil
	}
	if _, ok := classes[ident.Name]; !ok {
		return nil, fmt.Errorf("unknown class %s", ident.Name)
	}
	call := &ast.CallExpression{Callee: ident.Name}
	for _, arg := range expr.Arguments {
		value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		call.Arguments = append(call.Arguments, value)
	}
	return call, nil
}

func rewriteInvokeExpression(expr *ast.InvokeExpression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Expression, error) {
	switch callee := expr.Callee.(type) {
	case *ast.MemberExpression:
		return rewriteMemberInvoke(callee, expr.Arguments, bindings, callBindings, ctx, classes)
	case *ast.SuperExpression:
		if ctx == nil || ctx.info == nil || ctx.isStatic || ctx.info.base == "" {
			return nil, fmt.Errorf("super() is only valid inside derived constructors")
		}
		call := &ast.CallExpression{Callee: ctx.info.base}
		for _, arg := range expr.Arguments {
			value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, value)
		}
		return call, nil
	default:
		rewrittenCallee, err := rewriteExpression(expr.Callee, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		return &ast.InvokeExpression{Callee: rewrittenCallee, Arguments: args, Optional: expr.Optional}, nil
	}
}

func rewriteMemberInvoke(member *ast.MemberExpression, arguments []ast.Expression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Expression, error) {
	args := make([]ast.Expression, 0, len(arguments))
	for _, arg := range arguments {
		value, err := rewriteExpression(arg, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		args = append(args, value)
	}
	callee, err := rewriteMemberExpression(member, bindings, callBindings, ctx, classes)
	if err != nil {
		return nil, err
	}

	if !member.Private {
		if targetIdent, ok := member.Target.(*ast.Identifier); ok {
			switch targetIdent.Name {
			case "Date":
				if member.Property == "now" {
					if len(args) != 0 {
						return nil, fmt.Errorf("Date.now expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_std_date_now"}, nil
				}
			case "JSON":
				switch member.Property {
				case "stringify":
					if len(args) != 1 {
						return nil, fmt.Errorf("JSON.stringify expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_std_json_stringify", Arguments: []ast.Expression{args[0]}}, nil
				case "parse":
					if len(args) != 1 {
						return nil, fmt.Errorf("JSON.parse expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_std_json_parse", Arguments: []ast.Expression{args[0]}}, nil
				}
			case "Math":
				switch member.Property {
				case "floor", "ceil", "round", "abs", "sqrt":
					if len(args) != 1 {
						return nil, fmt.Errorf("Math.%s expects exactly 1 argument", member.Property)
					}
					return &ast.CallExpression{Callee: "__jayess_math_" + member.Property, Arguments: []ast.Expression{args[0]}}, nil
				case "min", "max", "pow":
					if len(args) != 2 {
						return nil, fmt.Errorf("Math.%s expects exactly 2 arguments", member.Property)
					}
					return &ast.CallExpression{Callee: "__jayess_math_" + member.Property, Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "random":
					if len(args) != 0 {
						return nil, fmt.Errorf("Math.random expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_math_random"}, nil
				}
			case "Object":
				switch member.Property {
				case "keys":
					if len(args) != 1 {
						return nil, fmt.Errorf("Object.keys expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_object_keys", Arguments: []ast.Expression{args[0]}}, nil
				case "values":
					if len(args) != 1 {
						return nil, fmt.Errorf("Object.values expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_object_values", Arguments: []ast.Expression{args[0]}}, nil
				case "entries":
					if len(args) != 1 {
						return nil, fmt.Errorf("Object.entries expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_object_entries", Arguments: []ast.Expression{args[0]}}, nil
				case "assign":
					if len(args) != 2 {
						return nil, fmt.Errorf("Object.assign expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_object_assign", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "hasOwn":
					if len(args) != 2 {
						return nil, fmt.Errorf("Object.hasOwn expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_object_has_own", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "fromEntries":
					if len(args) != 1 {
						return nil, fmt.Errorf("Object.fromEntries expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_object_from_entries", Arguments: []ast.Expression{args[0]}}, nil
				}
			case "Number":
				switch member.Property {
				case "isNaN":
					if len(args) != 1 {
						return nil, fmt.Errorf("Number.isNaN expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_number_is_nan", Arguments: []ast.Expression{args[0]}}, nil
				case "isFinite":
					if len(args) != 1 {
						return nil, fmt.Errorf("Number.isFinite expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_number_is_finite", Arguments: []ast.Expression{args[0]}}, nil
				}
			case "String":
				if member.Property == "fromCharCode" {
					return &ast.CallExpression{Callee: "__jayess_string_from_char_code", Arguments: args}, nil
				}
			case "Array":
				switch member.Property {
				case "isArray":
					if len(args) != 1 {
						return nil, fmt.Errorf("Array.isArray expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_array_is_array", Arguments: []ast.Expression{args[0]}}, nil
				case "from":
					if len(args) != 1 {
						return nil, fmt.Errorf("Array.from expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_array_from", Arguments: []ast.Expression{args[0]}}, nil
				case "of":
					return &ast.CallExpression{Callee: "__jayess_array_of", Arguments: args}, nil
				}
			case "console":
				switch member.Property {
				case "log":
					return &ast.CallExpression{Callee: "__jayess_console_log", Arguments: args}, nil
				case "warn":
					return &ast.CallExpression{Callee: "__jayess_console_warn", Arguments: args}, nil
				case "error":
					return &ast.CallExpression{Callee: "__jayess_console_error", Arguments: args}, nil
				}
			case "process":
				switch member.Property {
				case "cwd":
					if len(args) != 0 {
						return nil, fmt.Errorf("process.cwd expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_process_cwd"}, nil
				case "env":
					if len(args) != 1 {
						return nil, fmt.Errorf("process.env expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_process_env", Arguments: []ast.Expression{args[0]}}, nil
				case "argv":
					if len(args) != 0 {
						return nil, fmt.Errorf("process.argv expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_process_argv"}, nil
				case "platform":
					if len(args) != 0 {
						return nil, fmt.Errorf("process.platform expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_process_platform"}, nil
				case "arch":
					if len(args) != 0 {
						return nil, fmt.Errorf("process.arch expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_process_arch"}, nil
				case "exit":
					if len(args) != 1 {
						return nil, fmt.Errorf("process.exit expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_process_exit", Arguments: []ast.Expression{args[0]}}, nil
				}
			case "path":
				switch member.Property {
				case "join":
					return &ast.CallExpression{Callee: "__jayess_path_join", Arguments: args}, nil
				case "normalize":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.normalize expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_normalize", Arguments: []ast.Expression{args[0]}}, nil
				case "resolve":
					return &ast.CallExpression{Callee: "__jayess_path_resolve", Arguments: args}, nil
				case "relative":
					if len(args) != 2 {
						return nil, fmt.Errorf("path.relative expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_path_relative", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "parse":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.parse expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_parse", Arguments: []ast.Expression{args[0]}}, nil
				case "isAbsolute":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.isAbsolute expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_is_absolute", Arguments: []ast.Expression{args[0]}}, nil
				case "format":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.format expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_format", Arguments: []ast.Expression{args[0]}}, nil
				case "sep":
					if len(args) != 0 {
						return nil, fmt.Errorf("path.sep expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_path_sep"}, nil
				case "delimiter":
					if len(args) != 0 {
						return nil, fmt.Errorf("path.delimiter expects no arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_path_delimiter"}, nil
				case "basename":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.basename expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_basename", Arguments: []ast.Expression{args[0]}}, nil
				case "dirname":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.dirname expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_dirname", Arguments: []ast.Expression{args[0]}}, nil
				case "extname":
					if len(args) != 1 {
						return nil, fmt.Errorf("path.extname expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_path_extname", Arguments: []ast.Expression{args[0]}}, nil
				}
			case "fs":
				switch member.Property {
				case "readFile":
					if len(args) != 1 && len(args) != 2 {
						return nil, fmt.Errorf("fs.readFile expects 1 or 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_read_file", Arguments: args}, nil
				case "writeFile":
					if len(args) != 2 {
						return nil, fmt.Errorf("fs.writeFile expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_write_file", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "exists":
					if len(args) != 1 {
						return nil, fmt.Errorf("fs.exists expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_exists", Arguments: []ast.Expression{args[0]}}, nil
				case "readDir":
					if len(args) != 1 && len(args) != 2 {
						return nil, fmt.Errorf("fs.readDir expects 1 or 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_read_dir", Arguments: args}, nil
				case "stat":
					if len(args) != 1 {
						return nil, fmt.Errorf("fs.stat expects exactly 1 argument")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_stat", Arguments: []ast.Expression{args[0]}}, nil
				case "mkdir":
					if len(args) != 1 && len(args) != 2 {
						return nil, fmt.Errorf("fs.mkdir expects 1 or 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_mkdir", Arguments: args}, nil
				case "remove":
					if len(args) != 1 && len(args) != 2 {
						return nil, fmt.Errorf("fs.remove expects 1 or 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_remove", Arguments: args}, nil
				case "copyFile":
					if len(args) != 2 {
						return nil, fmt.Errorf("fs.copyFile expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_copy_file", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "copyDir":
					if len(args) != 2 {
						return nil, fmt.Errorf("fs.copyDir expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_copy_dir", Arguments: []ast.Expression{args[0], args[1]}}, nil
				case "rename":
					if len(args) != 2 {
						return nil, fmt.Errorf("fs.rename expects exactly 2 arguments")
					}
					return &ast.CallExpression{Callee: "__jayess_fs_rename", Arguments: []ast.Expression{args[0], args[1]}}, nil
				}
			}
		}
		switch member.Property {
		case "call":
			thisArg := ast.Expression(&ast.UndefinedLiteral{})
			callArgs := args
			if len(args) > 0 {
				thisArg = args[0]
				callArgs = args[1:]
			}
			callArray := &ast.ArrayLiteral{Elements: append([]ast.Expression{}, callArgs...)}
			return &ast.CallExpression{Callee: "__jayess_apply", Arguments: []ast.Expression{callee, thisArg, callArray}}, nil
		case "apply":
			if len(args) != 2 {
				return nil, fmt.Errorf("apply expects exactly 2 arguments")
			}
			return &ast.CallExpression{Callee: "__jayess_apply", Arguments: []ast.Expression{callee, args[0], args[1]}}, nil
		case "bind":
			thisArg := ast.Expression(&ast.UndefinedLiteral{})
			if len(args) > 0 {
				thisArg = args[0]
			}
			boundArgs := &ast.ArrayLiteral{}
			if len(args) > 1 {
				boundArgs.Elements = append(boundArgs.Elements, args[1:]...)
			}
			return &ast.CallExpression{Callee: "__jayess_bind", Arguments: []ast.Expression{callee, thisArg, boundArgs}}, nil
		case "push":
			if len(args) != 1 {
				return nil, fmt.Errorf("push expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_push", Arguments: []ast.Expression{target, args[0]}}, nil
		case "pop":
			if len(args) != 0 {
				return nil, fmt.Errorf("pop expects no arguments")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_pop", Arguments: []ast.Expression{target}}, nil
		case "shift":
			if len(args) != 0 {
				return nil, fmt.Errorf("shift expects no arguments")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_shift", Arguments: []ast.Expression{target}}, nil
		case "unshift":
			if len(args) != 1 {
				return nil, fmt.Errorf("unshift expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_unshift", Arguments: []ast.Expression{target, args[0]}}, nil
		case "slice":
			if len(args) > 2 {
				return nil, fmt.Errorf("slice expects at most 2 arguments")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			start := ast.Expression(&ast.NumberLiteral{Value: 0})
			end := ast.Expression(&ast.UndefinedLiteral{})
			if len(args) > 0 {
				start = args[0]
			}
			if len(args) > 1 {
				end = args[1]
			}
			return &ast.CallExpression{Callee: "__jayess_array_slice", Arguments: []ast.Expression{target, start, end}}, nil
		case "forEach":
			if len(args) != 1 {
				return nil, fmt.Errorf("forEach expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_for_each", Arguments: []ast.Expression{target, args[0]}}, nil
		case "map":
			if len(args) != 1 {
				return nil, fmt.Errorf("map expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_map", Arguments: []ast.Expression{target, args[0]}}, nil
		case "filter":
			if len(args) != 1 {
				return nil, fmt.Errorf("filter expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_filter", Arguments: []ast.Expression{target, args[0]}}, nil
		case "find":
			if len(args) != 1 {
				return nil, fmt.Errorf("find expects exactly 1 argument")
			}
			target, err := rewriteExpression(member.Target, bindings, callBindings, ctx, classes)
			if err != nil {
				return nil, err
			}
			return &ast.CallExpression{Callee: "__jayess_array_find", Arguments: []ast.Expression{target, args[0]}}, nil
		}
	}

	return &ast.InvokeExpression{Callee: callee, Arguments: args, Optional: member.Optional}, nil
}

func rewriteMemberExpression(expr *ast.MemberExpression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (ast.Expression, error) {
	switch target := expr.Target.(type) {
	case *ast.ThisExpression:
		if ctx == nil || ctx.info == nil {
			return &ast.MemberExpression{Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}, nil
		}
		if ctx.isStatic {
			if expr.Private {
				if !ctx.info.privateStaticFields[expr.Property] {
					if ctx.info.privateStaticMethods[expr.Property] {
						return &ast.Identifier{Name: staticMemberSymbol(ctx.info.name, expr.Property, true)}, nil
					}
					return nil, fmt.Errorf("unknown private static field #%s on %s", expr.Property, ctx.info.name)
				}
				return &ast.Identifier{Name: staticMemberSymbol(ctx.info.name, expr.Property, true)}, nil
			}
			if owner := lookupStaticFieldOwnerAST(classes, ctx.info.name, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			if owner := lookupStaticMethodOwnerAST(classes, ctx.info.name, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			return nil, fmt.Errorf("unknown static property %s on %s", expr.Property, ctx.info.name)
		}
		targetExpr := ast.Expression(&ast.Identifier{Name: "__self"})
		property := expr.Property
		if expr.Private {
			if !hasPrivateFieldAST(ctx.info, expr.Property) {
				if ctx.info.privateMethods[expr.Property] {
					return &ast.ClosureExpression{
						FunctionName: methodSymbol(ctx.info.name, expr.Property, true),
						Environment:  &ast.Identifier{Name: "__self"},
					}, nil
				}
				return nil, fmt.Errorf("unknown private field #%s on %s", expr.Property, ctx.info.name)
			}
			property = privateFieldStorage(ctx.info.name, expr.Property)
		} else if owner := lookupInstanceMethodOwnerAST(classes, ctx.info.name, expr.Property); owner != "" {
			sig := dispatchSignature{method: expr.Property, argCount: classes[owner].methods[expr.Property]}
			if ctx.dispatches != nil {
				ctx.dispatches[sig] = true
			}
			return &ast.ClosureExpression{
				FunctionName: dispatchFunctionName(sig),
				Environment:  &ast.Identifier{Name: "__self"},
			}, nil
		}
		return &ast.MemberExpression{Target: targetExpr, Property: property, Optional: expr.Optional}, nil
	case *ast.SuperExpression:
		if ctx == nil || ctx.info == nil || ctx.info.base == "" {
			return nil, fmt.Errorf("super is only valid inside derived class methods")
		}
		if expr.Private {
			return nil, fmt.Errorf("private super access is not supported")
		}
		if ctx.isStatic {
			if owner := lookupStaticFieldOwnerAST(classes, ctx.info.base, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			if owner := lookupStaticMethodOwnerAST(classes, ctx.info.base, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			return nil, fmt.Errorf("unknown static super property %s", expr.Property)
		}
		if owner := lookupInstanceMethodOwnerAST(classes, ctx.info.base, expr.Property); owner != "" {
			return &ast.ClosureExpression{
				FunctionName: methodSymbol(owner, expr.Property, false),
				Environment:  &ast.Identifier{Name: "__self"},
			}, nil
		}
		return &ast.MemberExpression{Target: &ast.Identifier{Name: "__self"}, Property: expr.Property, Optional: expr.Optional}, nil
	case *ast.BoundSuperExpression:
		if target.BaseClass == "" {
			return nil, fmt.Errorf("bound super is only valid for derived classes")
		}
		if expr.Private {
			return nil, fmt.Errorf("private super access is not supported")
		}
		if target.IsStatic {
			if owner := lookupStaticFieldOwnerAST(classes, target.BaseClass, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			if owner := lookupStaticMethodOwnerAST(classes, target.BaseClass, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			return nil, fmt.Errorf("unknown static super property %s", expr.Property)
		}
		if owner := lookupInstanceMethodOwnerAST(classes, target.BaseClass, expr.Property); owner != "" {
			return &ast.ClosureExpression{
				FunctionName: methodSymbol(owner, expr.Property, false),
				Environment:  target.Receiver,
			}, nil
		}
		receiver, err := rewriteExpression(target.Receiver, bindings, callBindings, ctx, classes)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{Target: receiver, Property: expr.Property, Optional: expr.Optional}, nil
	case *ast.Identifier:
		if target.Name == "process" {
			switch expr.Property {
			case "arch":
				return &ast.CallExpression{Callee: "__jayess_process_arch"}, nil
			}
		}
		if target.Name == "path" {
			switch expr.Property {
			case "sep":
				return &ast.CallExpression{Callee: "__jayess_path_sep"}, nil
			case "delimiter":
				return &ast.CallExpression{Callee: "__jayess_path_delimiter"}, nil
			}
		}
		if _, ok := classes[target.Name]; ok {
			if expr.Private {
				return nil, fmt.Errorf("private static members are not accessible outside classes")
			}
			if owner := lookupStaticFieldOwnerAST(classes, target.Name, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
			if owner := lookupStaticMethodOwnerAST(classes, target.Name, expr.Property); owner != "" {
				return &ast.Identifier{Name: staticMemberSymbol(owner, expr.Property, false)}, nil
			}
		}
		if className, ok := bindings[target.Name]; ok {
			if expr.Private {
				return nil, fmt.Errorf("private fields are only accessible through this.#name inside the declaring class")
			}
			if owner := lookupInstanceMethodOwnerAST(classes, className, expr.Property); owner != "" {
				sig := dispatchSignature{method: expr.Property, argCount: classes[owner].methods[expr.Property]}
				if ctx != nil && ctx.dispatches != nil {
					ctx.dispatches[sig] = true
				}
				return &ast.ClosureExpression{
					FunctionName: dispatchFunctionName(sig),
					Environment:  &ast.Identifier{Name: target.Name},
				}, nil
			}
		}
	}

	target, err := rewriteExpression(expr.Target, bindings, callBindings, ctx, classes)
	if err != nil {
		return nil, err
	}
	if expr.Private {
		return nil, fmt.Errorf("private fields are only accessible through this.#name inside the declaring class")
	}
	return &ast.MemberExpression{Target: target, Property: expr.Property, Optional: expr.Optional}, nil
}

func constructorInitialValue(info *loweredClassInfo) ast.Expression {
	if info.base != "" {
		return &ast.UndefinedLiteral{}
	}
	return &ast.ObjectLiteral{}
}

func implicitSuperInit(base string) ast.Statement {
	return &ast.IfStatement{
		Condition: &ast.ComparisonExpression{
			Operator: ast.OperatorEq,
			Left:     &ast.Identifier{Name: "__self"},
			Right:    &ast.UndefinedLiteral{},
		},
		Consequence: []ast.Statement{
			&ast.AssignmentStatement{
				Target: &ast.Identifier{Name: "__self"},
				Value:  &ast.CallExpression{Callee: base},
			},
		},
	}
}

func instanceFieldInitializers(info *loweredClassInfo) []ast.Statement {
	var out []ast.Statement
	for _, field := range info.instanceFields {
		value := field.Initializer
		if value == nil {
			value = &ast.UndefinedLiteral{}
		}
		out = append(out, &ast.AssignmentStatement{
			Target: &ast.MemberExpression{Target: &ast.Identifier{Name: "__self"}, Property: field.Name},
			Value:  value,
		})
	}
	for _, field := range info.privateFields {
		value := field.Initializer
		if value == nil {
			value = &ast.UndefinedLiteral{}
		}
		out = append(out, &ast.AssignmentStatement{
			Target: &ast.MemberExpression{Target: &ast.Identifier{Name: "__self"}, Property: privateFieldStorage(info.name, field.Name)},
			Value:  value,
		})
	}
	return out
}

func firstDirectSuperCallIndex(statements []ast.Statement) int {
	for i, stmt := range statements {
		exprStmt, ok := stmt.(*ast.ExpressionStatement)
		if !ok {
			continue
		}
		invoke, ok := exprStmt.Expression.(*ast.InvokeExpression)
		if !ok {
			continue
		}
		if _, ok := invoke.Callee.(*ast.SuperExpression); ok {
			return i
		}
	}
	return -1
}

func inferClassBinding(expr ast.Expression, bindings map[string]string, classes map[string]*loweredClassInfo) string {
	switch expr := expr.(type) {
	case *ast.CallExpression:
		if _, ok := classes[expr.Callee]; ok {
			return expr.Callee
		}
	case *ast.Identifier:
		if className, ok := bindings[expr.Name]; ok {
			return className
		}
	}
	return ""
}

func inferCallBinding(expr ast.Expression, bindings map[string]string, callBindings map[string]callBinding, ctx *classRewriteContext, classes map[string]*loweredClassInfo) (callBinding, bool, error) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		binding, ok := callBindings[expr.Name]
		return binding, ok, nil
	}
	return callBinding{}, false, nil
}

func lookupInstanceMethodOwnerAST(classes map[string]*loweredClassInfo, className, methodName string) string {
	info, ok := classes[className]
	if !ok {
		return ""
	}
	if _, ok := info.methods[methodName]; ok {
		return className
	}
	if info.base == "" {
		return ""
	}
	return lookupInstanceMethodOwnerAST(classes, info.base, methodName)
}

func lookupStaticMethodOwnerAST(classes map[string]*loweredClassInfo, className, methodName string) string {
	info, ok := classes[className]
	if !ok {
		return ""
	}
	if _, ok := info.staticMethods[methodName]; ok {
		return className
	}
	if info.base == "" {
		return ""
	}
	return lookupStaticMethodOwnerAST(classes, info.base, methodName)
}

func lookupStaticFieldOwnerAST(classes map[string]*loweredClassInfo, className, fieldName string) string {
	info, ok := classes[className]
	if !ok {
		return ""
	}
	if info.staticFields[fieldName] {
		return className
	}
	if info.base == "" {
		return ""
	}
	return lookupStaticFieldOwnerAST(classes, info.base, fieldName)
}

func hasPrivateFieldAST(info *loweredClassInfo, fieldName string) bool {
	for _, field := range info.privateFields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

func cloneBindings(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func cloneCallBindings(input map[string]callBinding) map[string]callBinding {
	out := make(map[string]callBinding, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func cloneExpression(expr ast.Expression) ast.Expression {
	switch expr := expr.(type) {
	case *ast.Identifier:
		return &ast.Identifier{Name: expr.Name}
	default:
		return expr
	}
}

func setClassTagStatement(className string) ast.Statement {
	return &ast.AssignmentStatement{
		Target: &ast.MemberExpression{Target: &ast.Identifier{Name: "__self"}, Property: "__jayess_class"},
		Value:  &ast.StringLiteral{Value: className},
	}
}

func setClassMarkerStatements(info *loweredClassInfo, classes map[string]*loweredClassInfo) []ast.Statement {
	var out []ast.Statement
	for _, className := range classLineage(info.name, classes) {
		out = append(out, &ast.AssignmentStatement{
			Target: &ast.MemberExpression{
				Target:   &ast.Identifier{Name: "__self"},
				Property: fmt.Sprintf("__jayess_is_%s", className),
			},
			Value: &ast.BooleanLiteral{Value: true},
		})
	}
	return out
}

func buildInstanceDispatchCall(method string, receiver ast.Expression, args []ast.Expression, dispatches map[dispatchSignature]bool) ast.Expression {
	sig := dispatchSignature{method: method, argCount: len(args)}
	if dispatches != nil {
		dispatches[sig] = true
	}
	callArgs := []ast.Expression{receiver}
	callArgs = append(callArgs, args...)
	return &ast.CallExpression{
		Callee:    dispatchFunctionName(sig),
		Arguments: callArgs,
	}
}

func dispatchFunctionName(sig dispatchSignature) string {
	return fmt.Sprintf("__jayess_dispatch__%s__%d", sig.method, sig.argCount)
}

func emitDispatchHelpers(classes map[string]*loweredClassInfo, dispatches map[dispatchSignature]bool) []*ast.FunctionDecl {
	var out []*ast.FunctionDecl
	for sig := range dispatches {
		params := []ast.Parameter{{Name: "__receiver"}}
		args := []ast.Expression{&ast.Identifier{Name: "__receiver"}}
		for i := 0; i < sig.argCount; i++ {
			name := fmt.Sprintf("arg%d", i)
			params = append(params, ast.Parameter{Name: name})
			args = append(args, &ast.Identifier{Name: name})
		}

		body := []ast.Statement{}
		for _, className := range classOrder(classes) {
			owner := lookupInstanceMethodOwnerAST(classes, className, sig.method)
			if owner == "" {
				continue
			}
			if classes[owner].methods[sig.method] != sig.argCount {
				continue
			}
			body = append(body, &ast.IfStatement{
				Condition: &ast.ComparisonExpression{
					Operator: ast.OperatorEq,
					Left:     &ast.MemberExpression{Target: &ast.Identifier{Name: "__receiver"}, Property: "__jayess_class"},
					Right:    &ast.StringLiteral{Value: className},
				},
				Consequence: []ast.Statement{
					&ast.ReturnStatement{
						Value: &ast.CallExpression{
							Callee:    methodSymbol(owner, sig.method, false),
							Arguments: args,
						},
					},
				},
			})
		}
		body = append(body, &ast.ReturnStatement{Value: &ast.UndefinedLiteral{}})
		out = append(out, &ast.FunctionDecl{
			Visibility: ast.VisibilityPublic,
			Name:       dispatchFunctionName(sig),
			Params:     params,
			Body:       body,
		})
	}
	return out
}

func classOrder(classes map[string]*loweredClassInfo) []string {
	keys := make([]string, 0, len(classes))
	for name := range classes {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	names := make([]string, 0, len(classes))
	seen := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if info := classes[name]; info != nil && info.base != "" {
			visit(info.base)
		}
		names = append(names, name)
	}
	for _, name := range keys {
		visit(name)
	}
	return names
}

func classLineage(name string, classes map[string]*loweredClassInfo) []string {
	var out []string
	current := name
	for current != "" {
		out = append(out, current)
		info := classes[current]
		if info == nil {
			break
		}
		current = info.base
	}
	return out
}

func methodSymbol(className, methodName string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, methodName)
	}
	return fmt.Sprintf("%s__%s", className, methodName)
}

func staticMemberSymbol(className, name string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, name)
	}
	return fmt.Sprintf("%s__%s", className, name)
}

func privateFieldStorage(className, fieldName string) string {
	return fmt.Sprintf("__jayess_private__%s__%s", className, fieldName)
}
