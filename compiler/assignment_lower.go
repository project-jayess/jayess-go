package compiler

import (
	"fmt"

	"jayess-go/ast"
)

type assignmentLowerer struct{}

func lowerAssignments(program *ast.Program) (*ast.Program, error) {
	l := &assignmentLowerer{}
	out := &ast.Program{
		Globals:         append([]*ast.VariableDecl{}, program.Globals...),
		ExternFunctions: append([]*ast.ExternFunctionDecl{}, program.ExternFunctions...),
	}

	for _, fn := range program.Functions {
		body, err := l.lowerStatements(fn.Body)
		if err != nil {
			return nil, err
		}
		cloned := *fn
		cloned.Body = body
		out.Functions = append(out.Functions, &cloned)
	}

	for _, classDecl := range program.Classes {
		cloned := *classDecl
		cloned.Members = nil
		for _, member := range classDecl.Members {
			if method, ok := member.(*ast.ClassMethodDecl); ok {
				rewritten := *method
				body, err := l.lowerStatements(method.Body)
				if err != nil {
					return nil, err
				}
				rewritten.Body = body
				cloned.Members = append(cloned.Members, &rewritten)
				continue
			}
			cloned.Members = append(cloned.Members, member)
		}
		out.Classes = append(out.Classes, &cloned)
	}

	return out, nil
}

func (l *assignmentLowerer) lowerStatements(statements []ast.Statement) ([]ast.Statement, error) {
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

func (l *assignmentLowerer) lowerStatement(stmt ast.Statement) ([]ast.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.AssignmentStatement:
		return l.lowerAssignment(stmt)
	case *ast.IfStatement:
		consequence, err := l.lowerStatements(stmt.Consequence)
		if err != nil {
			return nil, err
		}
		alternative, err := l.lowerStatements(stmt.Alternative)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.IfStatement{Condition: stmt.Condition, Consequence: consequence, Alternative: alternative}}, nil
	case *ast.WhileStatement:
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.WhileStatement{Condition: stmt.Condition, Body: body}}, nil
	case *ast.ForStatement:
		var init ast.Statement
		var err error
		if stmt.Init != nil {
			lowered, err := l.lowerStatement(stmt.Init)
			if err != nil {
				return nil, err
			}
			if len(lowered) > 1 {
				return nil, fmt.Errorf("compound assignments are not supported in for-loop init")
			}
			if len(lowered) == 1 {
				init = lowered[0]
			}
		}
		var update ast.Statement
		if stmt.Update != nil {
			lowered, err := l.lowerStatement(stmt.Update)
			if err != nil {
				return nil, err
			}
			if len(lowered) > 1 {
				return nil, fmt.Errorf("compound assignments are not supported in for-loop update")
			}
			if len(lowered) == 1 {
				update = lowered[0]
			}
		}
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ForStatement{Init: init, Condition: stmt.Condition, Update: update, Body: body}}, nil
	case *ast.ForOfStatement:
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ForOfStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: stmt.Iterable, Body: body}}, nil
	case *ast.ForInStatement:
		body, err := l.lowerStatements(stmt.Body)
		if err != nil {
			return nil, err
		}
		return []ast.Statement{&ast.ForInStatement{Kind: stmt.Kind, Name: stmt.Name, Iterable: stmt.Iterable, Body: body}}, nil
	case *ast.SwitchStatement:
		out := &ast.SwitchStatement{Discriminant: stmt.Discriminant}
		for _, switchCase := range stmt.Cases {
			consequent, err := l.lowerStatements(switchCase.Consequent)
			if err != nil {
				return nil, err
			}
			out.Cases = append(out.Cases, ast.SwitchCase{Test: switchCase.Test, Consequent: consequent})
		}
		defaultBody, err := l.lowerStatements(stmt.Default)
		if err != nil {
			return nil, err
		}
		out.Default = defaultBody
		return []ast.Statement{out}, nil
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

func (l *assignmentLowerer) lowerAssignment(stmt *ast.AssignmentStatement) ([]ast.Statement, error) {
	if stmt.Operator == "" || stmt.Operator == ast.AssignmentAssign {
		return []ast.Statement{&ast.AssignmentStatement{Target: stmt.Target, Operator: ast.AssignmentAssign, Value: stmt.Value}}, nil
	}

	readTarget := cloneAssignmentTarget(stmt.Target)
	if readTarget == nil {
		return nil, fmt.Errorf("unsupported assignment target")
	}

	switch stmt.Operator {
	case ast.AssignmentAddAssign:
		return []ast.Statement{&ast.AssignmentStatement{
			Target:   stmt.Target,
			Operator: ast.AssignmentAssign,
			Value:    &ast.BinaryExpression{Operator: ast.OperatorAdd, Left: readTarget, Right: stmt.Value},
		}}, nil
	case ast.AssignmentSubAssign:
		return []ast.Statement{&ast.AssignmentStatement{
			Target:   stmt.Target,
			Operator: ast.AssignmentAssign,
			Value:    &ast.BinaryExpression{Operator: ast.OperatorSub, Left: readTarget, Right: stmt.Value},
		}}, nil
	case ast.AssignmentMulAssign:
		return []ast.Statement{&ast.AssignmentStatement{
			Target:   stmt.Target,
			Operator: ast.AssignmentAssign,
			Value:    &ast.BinaryExpression{Operator: ast.OperatorMul, Left: readTarget, Right: stmt.Value},
		}}, nil
	case ast.AssignmentDivAssign:
		return []ast.Statement{&ast.AssignmentStatement{
			Target:   stmt.Target,
			Operator: ast.AssignmentAssign,
			Value:    &ast.BinaryExpression{Operator: ast.OperatorDiv, Left: readTarget, Right: stmt.Value},
		}}, nil
	case ast.AssignmentNullishAssign:
		condition := &ast.LogicalExpression{
			Operator: ast.OperatorOr,
			Left: &ast.ComparisonExpression{
				Operator: ast.OperatorStrictEq,
				Left:     readTarget,
				Right:    &ast.UndefinedLiteral{},
			},
			Right: &ast.ComparisonExpression{
				Operator: ast.OperatorEq,
				Left:     cloneAssignmentTarget(stmt.Target),
				Right:    &ast.NullLiteral{},
			},
		}
		return []ast.Statement{&ast.IfStatement{
			Condition: condition,
			Consequence: []ast.Statement{
				&ast.AssignmentStatement{Target: stmt.Target, Operator: ast.AssignmentAssign, Value: stmt.Value},
			},
		}}, nil
	case ast.AssignmentOrAssign:
		return []ast.Statement{&ast.IfStatement{
			Condition: &ast.UnaryExpression{Operator: ast.OperatorNot, Right: readTarget},
			Consequence: []ast.Statement{
				&ast.AssignmentStatement{Target: stmt.Target, Operator: ast.AssignmentAssign, Value: stmt.Value},
			},
		}}, nil
	case ast.AssignmentAndAssign:
		return []ast.Statement{&ast.IfStatement{
			Condition: readTarget,
			Consequence: []ast.Statement{
				&ast.AssignmentStatement{Target: stmt.Target, Operator: ast.AssignmentAssign, Value: stmt.Value},
			},
		}}, nil
	default:
		return nil, fmt.Errorf("unsupported assignment operator")
	}
}

func cloneAssignmentTarget(expr ast.Expression) ast.Expression {
	switch expr := expr.(type) {
	case *ast.Identifier:
		return &ast.Identifier{Name: expr.Name}
	case *ast.MemberExpression:
		target := cloneAssignmentTarget(expr.Target)
		if target == nil {
			return nil
		}
		return &ast.MemberExpression{Target: target, Property: expr.Property, Private: expr.Private, Optional: expr.Optional}
	case *ast.IndexExpression:
		target := cloneAssignmentTarget(expr.Target)
		index := cloneAssignmentTarget(expr.Index)
		if target == nil || index == nil {
			return nil
		}
		return &ast.IndexExpression{Target: target, Index: index, Optional: expr.Optional}
	case *ast.NumberLiteral, *ast.StringLiteral, *ast.BooleanLiteral, *ast.NullLiteral, *ast.UndefinedLiteral, *ast.ThisExpression, *ast.SuperExpression, *ast.NewTargetExpression, *ast.BoundSuperExpression:
		return expr
	default:
		return expr
	}
}
