package compiler

import (
	"fmt"

	"jayess-go/ast"
)

type destructureLowerer struct {
	counter int
}

func lowerDestructuring(program *ast.Program) (*ast.Program, error) {
	l := &destructureLowerer{}
	out := &ast.Program{
		TypeAliases:     append([]*ast.TypeAliasDecl{}, program.TypeAliases...),
		Globals:         append([]*ast.VariableDecl{}, program.Globals...),
		ExternFunctions: append([]*ast.ExternFunctionDecl{}, program.ExternFunctions...),
	}

	for _, fn := range program.Functions {
		lowered, err := l.lowerFunctionDecl(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, lowered)
	}

	for _, classDecl := range program.Classes {
		cloned := *classDecl
		cloned.Members = nil
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassMethodDecl:
				params, prologue, err := l.lowerParameters(member.Params)
				if err != nil {
					return nil, err
				}
				body, err := l.lowerStatements(member.Body)
				if err != nil {
					return nil, err
				}
				rewritten := *member
				rewritten.Params = params
				rewritten.Body = append(prologue, body...)
				cloned.Members = append(cloned.Members, &rewritten)
			case *ast.ClassFieldDecl:
				rewritten := *member
				if member.Initializer != nil {
					value, err := l.lowerExpression(member.Initializer)
					if err != nil {
						return nil, err
					}
					rewritten.Initializer = value
				}
				cloned.Members = append(cloned.Members, &rewritten)
			default:
				cloned.Members = append(cloned.Members, member)
			}
		}
		out.Classes = append(out.Classes, &cloned)
	}

	return out, nil
}

func (l *destructureLowerer) nextTemp() string {
	name := fmt.Sprintf("__jayess_destructure_%d", l.counter)
	l.counter++
	return name
}

func (l *destructureLowerer) lowerFunctionDecl(fn *ast.FunctionDecl) (*ast.FunctionDecl, error) {
	params, prologue, err := l.lowerParameters(fn.Params)
	if err != nil {
		return nil, err
	}
	body, err := l.lowerStatements(fn.Body)
	if err != nil {
		return nil, err
	}
	cloned := *fn
	cloned.Params = params
	cloned.Body = append(prologue, body...)
	return &cloned, nil
}

func (l *destructureLowerer) lowerParameters(params []ast.Parameter) ([]ast.Parameter, []ast.Statement, error) {
	out := make([]ast.Parameter, 0, len(params))
	var prologue []ast.Statement
	for _, param := range params {
		rewritten := ast.Parameter{Rest: param.Rest, TypeAnnotation: param.TypeAnnotation}
		if param.Default != nil {
			value, err := l.lowerExpression(param.Default)
			if err != nil {
				return nil, nil, err
			}
			rewritten.Default = value
		}
		if param.Pattern == nil {
			rewritten.Name = param.Name
			out = append(out, rewritten)
			continue
		}
		if param.Rest {
			return nil, nil, fmt.Errorf("rest parameter destructuring is not supported yet")
		}
		temp := l.nextTemp()
		rewritten.Name = temp
		out = append(out, rewritten)
		bindings, err := l.bindPattern(param.Pattern, &ast.Identifier{Name: temp}, ast.DeclarationVar)
		if err != nil {
			return nil, nil, err
		}
		prologue = append(prologue, bindings...)
	}
	return out, prologue, nil
}

func (l *destructureLowerer) lowerStatements(statements []ast.Statement) ([]ast.Statement, error) {
	var out []ast.Statement
	for _, stmt := range statements {
		lowered, err := l.lowerStatement(stmt)
		if err != nil {
			return nil, err
		}
		out = append(out, lowered...)
	}
	return out, nil
}

func (l *destructureLowerer) lowerStatement(stmt ast.Statement) ([]ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.VariableDecl{Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, TypeAnnotation: stmt.TypeAnnotation, Value: value}}, nil
	case *ast.DestructuringDecl:
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		if direct, ok, err := l.bindPatternFromLiteral(stmt.Pattern, value, stmt.Kind); err != nil {
			return nil, err
		} else if ok {
			return direct, nil
		}
		temp := l.nextTemp()
		out := []ast.Statement{
			&ast.VariableDecl{
				Visibility: stmt.Visibility,
				Kind:       ast.DeclarationVar,
				Name:       temp,
				Value:      value,
			},
		}
		bindings, err := l.bindPattern(stmt.Pattern, &ast.Identifier{Name: temp}, stmt.Kind)
		if err != nil {
			return nil, err
		}
		return append(out, bindings...), nil
	case *ast.AssignmentStatement:
		target, err := l.lowerExpression(stmt.Target)
		if err != nil {
			return nil, err
		}
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.AssignmentStatement{Target: target, Value: value}}, nil
	case *ast.DestructuringAssignment:
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		if direct, ok, err := l.assignPatternFromLiteral(stmt.Pattern, value); err != nil {
			return nil, err
		} else if ok {
			return direct, nil
		}
		temp := l.nextTemp()
		out := []ast.Statement{
			&ast.VariableDecl{
				Visibility: ast.VisibilityPublic,
				Kind:       ast.DeclarationVar,
				Name:       temp,
				Value:      value,
			},
		}
		bindings, err := l.assignPattern(stmt.Pattern, &ast.Identifier{Name: temp})
		if err != nil {
			return nil, err
		}
		return append(out, bindings...), nil
	case *ast.ReturnStatement:
		if stmt.Value == nil {
			return []ast.Statement{&ast.ReturnStatement{}}, nil
		}
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ReturnStatement{Value: value}}, nil
	case *ast.ExpressionStatement:
		expr, err := l.lowerExpression(stmt.Expression)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ExpressionStatement{Expression: expr}}, nil
	case *ast.DeleteStatement:
		target, err := l.lowerExpression(stmt.Target)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.DeleteStatement{Target: target}}, nil
	case *ast.ThrowStatement:
		value, err := l.lowerExpression(stmt.Value)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ThrowStatement{Value: value}}, nil
	case *ast.IfStatement:
		condition, err := l.lowerExpression(stmt.Condition)
		if err != nil {
			return nil, err
		}
		consequence, err := l.lowerStatements(stmt.Consequence)
		if err != nil {
			return nil, err
		}
		alternative, err := l.lowerStatements(stmt.Alternative)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}}, nil
	case *ast.WhileStatement:
		condition, err := l.lowerExpression(stmt.Condition)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.WhileStatement{Condition: condition, Body: body}}, nil
	case *ast.DoWhileStatement:
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		condition, err := l.lowerExpression(stmt.Condition)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.DoWhileStatement{Body: body, Condition: condition}}, nil
	case *ast.BlockStatement:
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.BlockStatement{Body: body}}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var initPrefix []ast.Statement
		if stmt.Init != nil {
			lowered, err := l.lowerStatement(stmt.Init)
			if err != nil {
				return nil, err
			}
			if len(lowered) > 1 {
				initPrefix = append(initPrefix, lowered...)
			} else if len(lowered) == 1 {
				init = lowered[0]
			}
		}
		var condition ast.Expression
		var err error
		if stmt.Condition != nil {
			condition, err = l.lowerExpression(stmt.Condition)
			if err != nil {
				return nil, err
			}
		}
		var update ast.Statement
		if stmt.Update != nil {
			lowered, err := l.lowerStatement(stmt.Update)
			if err != nil {
				return nil, err
			}
			if len(lowered) > 1 {
				return nil, fmt.Errorf("destructuring is not supported in for-loop update yet")
			}
			if len(lowered) == 1 {
				update = lowered[0]
			}
		}
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		forStmt := &ast.ForStatement{Init: init, Condition: condition, Update: update, Body: body}
		if len(initPrefix) == 0 {
			return []ast.Statement{forStmt}, nil
		}
		blockBody := append(initPrefix, forStmt)
		return []ast.Statement{&ast.BlockStatement{Body: blockBody}}, nil
	case *ast.ForOfStatement:
		iterable, err := l.lowerExpression(stmt.Iterable)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ForOfStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}}, nil
	case *ast.ForInStatement:
		iterable, err := l.lowerExpression(stmt.Iterable)
		if err != nil {
			return nil, err
		}
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ForInStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: iterable, Body: body}}, nil
	case *ast.SwitchStatement:
		discriminant, err := l.lowerExpression(stmt.Discriminant)
		if err != nil {
			return nil, err
		}
		out := &ast.SwitchStatement{Discriminant: discriminant}
		for _, switchCase := range stmt.Cases {
			test, err := l.lowerExpression(switchCase.Test)
			if err != nil {
				return nil, err
			}
			consequent, err := l.lowerStatements(switchCase.Consequent)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		}
		defaultBody, err := l.lowerStatements(stmt.Default)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return []ast.Statement{out}, nil
	case *ast.LabeledStatement:
		lowered, err := l.lowerStatement(stmt.Statement)
		if err != nil {
			return nil, err
		}
		if len(lowered) != 1 {
			return nil, fmt.Errorf("labeled statements cannot lower to multiple statements")
		}
		return []ast.Statement{&ast.LabeledStatement{Label: stmt.Label, Statement: lowered[0]}}, nil
	case *ast.TryStatement:
		tryBody, err := l.lowerStatements(stmt.TryBody)
		if err != nil {
			return nil, err
		}
		catchBody, err := l.lowerStatements(stmt.CatchBody)
		if err != nil {
			return nil, err
		}
		finallyBody, err := l.lowerStatements(stmt.FinallyBody)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.TryStatement{TryBody: tryBody, CatchName: stmt.CatchName, CatchBody: catchBody, FinallyBody: finallyBody}}, nil
	default:
		return []ast.Statement{stmt}, nil
	}
}

func (l *destructureLowerer) lowerExpression(expr ast.Expression) (ast.Expression, error) {
	switch expr := expr.(type) {
	case *ast.ObjectLiteral:
		out := &ast.ObjectLiteral{}
		for _, property := range expr.Properties {
			var keyExpr ast.Expression
			if property.KeyExpr != nil {
				value, err := l.lowerExpression(property.KeyExpr)
				if err != nil {
					return nil, err
				}
				keyExpr = value
			}
			value, err := l.lowerExpression(property.Value)
			if err != nil {
				return nil, err
			}
			out.Properties = append(out.Properties, ast.ObjectProperty{
				Key:      property.Key,
				KeyExpr:  keyExpr,
				Value:    value,
				Computed: property.Computed,
				Spread:   property.Spread,
				Getter:   property.Getter,
				Setter:   property.Setter,
			})
		}
		return out, nil
	case *ast.ArrayLiteral:
		out := &ast.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := l.lowerExpression(element)
			if err != nil {
				return nil, err
			}
			out.Elements = append(out.Elements, value)
		}
		return out, nil
	case *ast.TemplateLiteral:
		out := &ast.TemplateLiteral{Parts: append([]string{}, expr.Parts...)}
		for _, valueExpr := range expr.Values {
			value, err := l.lowerExpression(valueExpr)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := l.lowerExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{Value: value}, nil
	case *ast.BinaryExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := l.lowerExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{Value: value}, nil
	case *ast.TypeCheckExpression:
		value, err := l.lowerExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		return &ast.TypeCheckExpression{Value: value, TypeAnnotation: expr.TypeAnnotation}, nil
	case *ast.InstanceofExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.InstanceofExpression{Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.LogicalExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.LogicalExpression{Operator: expr.Operator, Left: left, Right: right}, nil
	case *ast.NullishCoalesceExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.NullishCoalesceExpression{Left: left, Right: right}, nil
	case *ast.CommaExpression:
		left, err := l.lowerExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.CommaExpression{Left: left, Right: right}, nil
	case *ast.ConditionalExpression:
		condition, err := l.lowerExpression(expr.Condition)
		if err != nil {
			return nil, err
		}
		consequent, err := l.lowerExpression(expr.Consequent)
		if err != nil {
			return nil, err
		}
		alternative, err := l.lowerExpression(expr.Alternative)
		if err != nil {
			return nil, err
		}
		return &ast.ConditionalExpression{Condition: condition, Consequent: consequent, Alternative: alternative}, nil
	case *ast.UnaryExpression:
		right, err := l.lowerExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: expr.Operator, Right: right}, nil
	case *ast.IndexExpression:
		target, err := l.lowerExpression(expr.Target)
		if err != nil {
			return nil, err
		}
		index, err := l.lowerExpression(expr.Index)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpression{Target: target, Index: index, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		target, err := l.lowerExpression(expr.Target)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}, nil
	case *ast.CallExpression:
		out := &ast.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.InvokeExpression:
		callee, err := l.lowerExpression(expr.Callee)
		if err != nil {
			return nil, err
		}
		out := &ast.InvokeExpression{Callee: callee, Optional: expr.Optional}
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.NewExpression:
		callee, err := l.lowerExpression(expr.Callee)
		if err != nil {
			return nil, err
		}
		out := &ast.NewExpression{Callee: callee}
		for _, arg := range expr.Arguments {
			value, err := l.lowerExpression(arg)
			if err != nil {
				return nil, err
			}
			out.Arguments = append(out.Arguments, value)
		}
		return out, nil
	case *ast.FunctionExpression:
		params, prologue, err := l.lowerParameters(expr.Params)
		if err != nil {
			return nil, err
		}
		rewritten := &ast.FunctionExpression{
			BaseNode:        expr.BaseNode,
			Params:          params,
			ReturnType:      expr.ReturnType,
			IsAsync:         expr.IsAsync,
			IsGenerator:     expr.IsGenerator,
			IsArrowFunction: expr.IsArrowFunction,
		}
		if expr.ExpressionBody != nil {
			value, err := l.lowerExpression(expr.ExpressionBody)
			if err != nil {
				return nil, err
			}
			rewritten.ExpressionBody = value
			if len(prologue) > 0 {
				rewritten.ExpressionBody = nil
				rewritten.Body = append(prologue, &ast.ReturnStatement{Value: value})
			}
			return rewritten, nil
		}
		body, err := l.lowerStatements(expr.Body)
		if err != nil {
			return nil, err
		}
		rewritten.Body = append(prologue, body...)
		return rewritten, nil
	default:
		return expr, nil
	}
}

func (l *destructureLowerer) bindPattern(pattern ast.Pattern, source ast.Expression, kind ast.DeclarationKind) ([]ast.Statement, error) {
	switch pattern := pattern.(type) {
	case *ast.IdentifierPattern:
		return []ast.Statement{
			&ast.VariableDecl{
				Visibility: ast.VisibilityPublic,
				Kind:       kind,
				Name:       pattern.Name,
				Value:      source,
			},
		}, nil
	case *ast.ObjectPattern:
		var out []ast.Statement
		for _, property := range pattern.Properties {
			value, prefix, err := l.patternValueWithDefault(
				&ast.MemberExpression{Target: source, Property: property.Key},
				property.Default,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, prefix...)
			nested, err := l.bindPattern(property.Pattern, value, kind)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
		if pattern.Rest != "" {
			excluded := &ast.ArrayLiteral{}
			for _, property := range pattern.Properties {
				excluded.Elements = append(excluded.Elements, &ast.StringLiteral{Value: property.Key})
			}
			out = append(out, &ast.VariableDecl{
				Visibility: ast.VisibilityPublic,
				Kind:       kind,
				Name:       pattern.Rest,
				Value:      &ast.CallExpression{Callee: "__jayess_object_rest", Arguments: []ast.Expression{source, excluded}},
			})
		}
		return out, nil
	case *ast.ArrayPattern:
		var out []ast.Statement
		for index, element := range pattern.Elements {
			if element.Pattern == nil {
				continue
			}
			if element.Rest {
				out = append(out, &ast.VariableDecl{
					Visibility: ast.VisibilityPublic,
					Kind:       kind,
					Name:       element.Pattern.(*ast.IdentifierPattern).Name,
					Value:      &ast.CallExpression{Callee: "__jayess_array_slice", Arguments: []ast.Expression{source, &ast.NumberLiteral{Value: float64(index)}, &ast.UndefinedLiteral{}}},
				})
				continue
			}
			value, prefix, err := l.patternValueWithDefault(
				&ast.IndexExpression{Target: source, Index: &ast.NumberLiteral{Value: float64(index)}},
				element.Default,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, prefix...)
			nested, err := l.bindPattern(element.Pattern, value, kind)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported binding pattern")
	}
}

func (l *destructureLowerer) bindPatternFromLiteral(pattern ast.Pattern, source ast.Expression, kind ast.DeclarationKind) ([]ast.Statement, bool, error) {
	return l.literalPatternStatements(
		pattern,
		source,
		func(name string, value ast.Expression) ast.Statement {
			return &ast.VariableDecl{
				Visibility: ast.VisibilityPublic,
				Kind:       kind,
				Name:       name,
				Value:      value,
			}
		},
		func(pattern ast.Pattern, value ast.Expression) ([]ast.Statement, error) {
			return l.bindPattern(pattern, value, kind)
		},
	)
}

func (l *destructureLowerer) assignPattern(pattern ast.Pattern, source ast.Expression) ([]ast.Statement, error) {
	switch pattern := pattern.(type) {
	case *ast.IdentifierPattern:
		return []ast.Statement{
			&ast.AssignmentStatement{
				Target: &ast.Identifier{Name: pattern.Name},
				Value:  source,
			},
		}, nil
	case *ast.ObjectPattern:
		var out []ast.Statement
		for _, property := range pattern.Properties {
			value, prefix, err := l.patternValueWithDefault(
				&ast.MemberExpression{Target: source, Property: property.Key},
				property.Default,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, prefix...)
			nested, err := l.assignPattern(property.Pattern, value)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
		if pattern.Rest != "" {
			excluded := &ast.ArrayLiteral{}
			for _, property := range pattern.Properties {
				excluded.Elements = append(excluded.Elements, &ast.StringLiteral{Value: property.Key})
			}
			out = append(out, &ast.AssignmentStatement{
				Target: &ast.Identifier{Name: pattern.Rest},
				Value:  &ast.CallExpression{Callee: "__jayess_object_rest", Arguments: []ast.Expression{source, excluded}},
			})
		}
		return out, nil
	case *ast.ArrayPattern:
		var out []ast.Statement
		for index, element := range pattern.Elements {
			if element.Pattern == nil {
				continue
			}
			if element.Rest {
				out = append(out, &ast.AssignmentStatement{
					Target: &ast.Identifier{Name: element.Pattern.(*ast.IdentifierPattern).Name},
					Value:  &ast.CallExpression{Callee: "__jayess_array_slice", Arguments: []ast.Expression{source, &ast.NumberLiteral{Value: float64(index)}, &ast.UndefinedLiteral{}}},
				})
				continue
			}
			value, prefix, err := l.patternValueWithDefault(
				&ast.IndexExpression{Target: source, Index: &ast.NumberLiteral{Value: float64(index)}},
				element.Default,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, prefix...)
			nested, err := l.assignPattern(element.Pattern, value)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported assignment pattern")
	}
}

func (l *destructureLowerer) literalPatternStatements(
	pattern ast.Pattern,
	source ast.Expression,
	emitIdentifier func(name string, value ast.Expression) ast.Statement,
	fallback func(pattern ast.Pattern, value ast.Expression) ([]ast.Statement, error),
) ([]ast.Statement, bool, error) {
	switch pattern := pattern.(type) {
	case *ast.IdentifierPattern:
		return []ast.Statement{
			emitIdentifier(pattern.Name, source),
		}, true, nil
	case *ast.ArrayPattern:
		literal, ok := l.directArrayPatternSource(source)
		if !ok {
			return nil, false, nil
		}
		var out []ast.Statement
		for index, element := range pattern.Elements {
			if element.Pattern == nil {
				continue
			}
			if element.Rest {
				identifier, ok := element.Pattern.(*ast.IdentifierPattern)
				if !ok {
					return nil, false, nil
				}
				rest := &ast.ArrayLiteral{}
				for _, trailing := range literal.Elements[index:] {
					if trailing == nil {
						rest.Elements = append(rest.Elements, &ast.UndefinedLiteral{})
						continue
					}
					rest.Elements = append(rest.Elements, trailing)
				}
				out = append(out, emitIdentifier(
					identifier.Name,
					rest,
				))
				continue
			}
			value := ast.Expression(&ast.UndefinedLiteral{})
			if index < len(literal.Elements) && literal.Elements[index] != nil {
				value = literal.Elements[index]
			}
			resolved, prefix, err := l.patternValueWithDefault(value, element.Default)
			if err != nil {
				return nil, false, err
			}
			out = append(out, prefix...)
			nested, ok, err := l.literalPatternStatements(element.Pattern, resolved, emitIdentifier, fallback)
			if err != nil {
				return nil, false, err
			}
			if !ok {
				nested, err = l.materializeLiteralFallback(element.Pattern, resolved, fallback)
				if err != nil {
					return nil, false, err
				}
			}
			out = append(out, nested...)
		}
		return out, true, nil
	case *ast.ObjectPattern:
		literal, values, ok := l.directObjectPatternSource(source)
		if !ok {
			return nil, false, nil
		}
		var out []ast.Statement
		for _, property := range pattern.Properties {
			value := ast.Expression(&ast.UndefinedLiteral{})
			if resolvedValue, ok := values[property.Key]; ok && resolvedValue != nil {
				value = resolvedValue
			}
			resolved, prefix, err := l.patternValueWithDefault(value, property.Default)
			if err != nil {
				return nil, false, err
			}
			out = append(out, prefix...)
			nested, ok, err := l.literalPatternStatements(property.Pattern, resolved, emitIdentifier, fallback)
			if err != nil {
				return nil, false, err
			}
			if !ok {
				nested, err = l.materializeLiteralFallback(property.Pattern, resolved, fallback)
				if err != nil {
					return nil, false, err
				}
			}
			out = append(out, nested...)
		}
		if pattern.Rest != "" {
			excluded := map[string]struct{}{}
			for _, property := range pattern.Properties {
				excluded[property.Key] = struct{}{}
			}
			rest := &ast.ObjectLiteral{}
			for _, property := range literal.Properties {
				if _, blocked := excluded[property.Key]; blocked {
					continue
				}
				rest.Properties = append(rest.Properties, property)
			}
			out = append(out, emitIdentifier(
				pattern.Rest,
				rest,
			))
		}
		return out, true, nil
	default:
		return nil, false, nil
	}
}

func (l *destructureLowerer) directArrayPatternSource(source ast.Expression) (*ast.ArrayLiteral, bool) {
	literal, ok := source.(*ast.ArrayLiteral)
	if !ok {
		return nil, false
	}
	for _, value := range literal.Elements {
		if _, spread := value.(*ast.SpreadExpression); spread {
			return nil, false
		}
	}
	return literal, true
}

func (l *destructureLowerer) directObjectPatternSource(source ast.Expression) (*ast.ObjectLiteral, map[string]ast.Expression, bool) {
	literal, ok := source.(*ast.ObjectLiteral)
	if !ok {
		return nil, nil, false
	}
	seenKeys := map[string]struct{}{}
	values := map[string]ast.Expression{}
	for _, property := range literal.Properties {
		if property.Computed || property.Spread || property.Getter || property.Setter {
			return nil, nil, false
		}
		if _, exists := seenKeys[property.Key]; exists {
			return nil, nil, false
		}
		seenKeys[property.Key] = struct{}{}
		values[property.Key] = property.Value
	}
	return literal, values, true
}

func (l *destructureLowerer) materializeLiteralFallback(
	pattern ast.Pattern,
	source ast.Expression,
	fallback func(pattern ast.Pattern, value ast.Expression) ([]ast.Statement, error),
) ([]ast.Statement, error) {
	temp := l.nextTemp()
	out := []ast.Statement{
		&ast.VariableDecl{
			Visibility: ast.VisibilityPublic,
			Kind:       ast.DeclarationVar,
			Name:       temp,
			Value:      source,
		},
	}
	nested, err := fallback(pattern, &ast.Identifier{Name: temp})
	if err != nil {
		return nil, err
	}
	return append(out, nested...), nil
}

func (l *destructureLowerer) assignPatternFromLiteral(pattern ast.Pattern, source ast.Expression) ([]ast.Statement, bool, error) {
	return l.literalPatternStatements(
		pattern,
		source,
		func(name string, value ast.Expression) ast.Statement {
			return &ast.AssignmentStatement{
				Target: &ast.Identifier{Name: name},
				Value:  value,
			}
		},
		func(pattern ast.Pattern, value ast.Expression) ([]ast.Statement, error) {
			return l.assignPattern(pattern, value)
		},
	)
}

func (l *destructureLowerer) patternValueWithDefault(source ast.Expression, defaultValue ast.Expression) (ast.Expression, []ast.Statement, error) {
	if defaultValue == nil {
		return source, nil, nil
	}
	loweredDefault, err := l.lowerExpression(defaultValue)
	if err != nil {
		return nil, nil, err
	}
	temp := l.nextTemp()
	out := []ast.Statement{
		&ast.VariableDecl{
			Visibility: ast.VisibilityPublic,
			Kind:       ast.DeclarationVar,
			Name:       temp,
			Value:      source,
		},
		&ast.IfStatement{
			Condition: &ast.ComparisonExpression{
				Operator: ast.OperatorStrictEq,
				Left:     &ast.Identifier{Name: temp},
				Right:    &ast.UndefinedLiteral{},
			},
			Consequence: []ast.Statement{
				&ast.AssignmentStatement{
					Target: &ast.Identifier{Name: temp},
					Value:  loweredDefault,
				},
			},
		},
	}
	return &ast.Identifier{Name: temp}, out, nil
}
