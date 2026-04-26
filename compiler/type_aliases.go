package compiler

import (
	"fmt"
	"strings"

	"jayess-go/ast"
	"jayess-go/typesys"
)

type aliasResolutionSpec struct {
	params []ast.TypeParameter
	target string
}

func resolveTypeAliases(program *ast.Program) error {
	if len(program.TypeAliases) == 0 {
		return nil
	}
	raw := map[string]aliasResolutionSpec{}
	for _, alias := range program.TypeAliases {
		if alias == nil {
			continue
		}
		if _, exists := raw[alias.Name]; exists {
			return fmt.Errorf("duplicate type alias %s", alias.Name)
		}
		raw[alias.Name] = aliasResolutionSpec{params: alias.TypeParams, target: alias.Target}
	}
	resolved := map[string]string{}
	visiting := map[string]bool{}
	var resolve func(string) (string, error)
	var rewriteType func(string, map[string]*typesys.Expr) (string, error)
	resolve = func(name string) (string, error) {
		if normalizedBuiltinType(name) != "" {
			return normalizedBuiltinType(name), nil
		}
		if value, ok := resolved[name]; ok {
			return value, nil
		}
		spec, ok := raw[name]
		if !ok {
			return "", fmt.Errorf("unknown type alias %s", name)
		}
		if len(spec.params) > 0 {
			return "", fmt.Errorf("generic type alias %s requires type arguments", name)
		}
		if visiting[name] {
			return "", fmt.Errorf("type alias cycle detected involving %s", name)
		}
		visiting[name] = true
		value, err := rewriteType(spec.target, nil)
		delete(visiting, name)
		if err != nil {
			return "", err
		}
		resolved[name] = value
		return value, nil
	}
	rewriteType = func(name string, scope map[string]*typesys.Expr) (string, error) {
		if name == "" {
			return "", nil
		}
		expr, err := typesys.Parse(name)
		if err != nil {
			return "", err
		}
		rewritten, err := instantiateAliasExpr(expr, raw, scope, visiting, resolve)
		if err != nil {
			return "", err
		}
		if rewritten.Kind == typesys.KindAny {
			return "", nil
		}
		return rewritten.String(), nil
	}
	rewrite := func(name string) (string, error) {
		return rewriteType(name, nil)
	}
	for name, spec := range raw {
		if len(spec.params) > 0 {
			continue
		}
		if _, err := resolve(name); err != nil {
			return err
		}
	}
	for _, global := range program.Globals {
		if global == nil {
			continue
		}
		if err := rewriteStatementTypeAnnotations(global, rewrite); err != nil {
			return err
		}
	}
	for _, ext := range program.ExternFunctions {
		if ext == nil {
			continue
		}
		for i := range ext.Params {
			rewritten, err := rewrite(ext.Params[i].TypeAnnotation)
			if err != nil {
				return err
			}
			ext.Params[i].TypeAnnotation = rewritten
		}
	}
	for _, fn := range program.Functions {
		if fn == nil {
			continue
		}
		if err := rewriteFunctionTypeAnnotations(fn, rewrite); err != nil {
			return err
		}
	}
	for _, classDecl := range program.Classes {
		if classDecl == nil {
			continue
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				rewritten, err := rewrite(member.TypeAnnotation)
				if err != nil {
					return err
				}
				member.TypeAnnotation = rewritten
				if err := rewriteExpressionTypeAnnotations(member.Initializer, rewrite); err != nil {
					return err
				}
			case *ast.ClassMethodDecl:
				for i := range member.Params {
					rewritten, err := rewrite(member.Params[i].TypeAnnotation)
					if err != nil {
						return err
					}
					member.Params[i].TypeAnnotation = rewritten
					if err := rewriteExpressionTypeAnnotations(member.Params[i].Default, rewrite); err != nil {
						return err
					}
				}
				if err := rewriteStatementsTypeAnnotations(member.Body, rewrite); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func instantiateAliasExpr(expr *typesys.Expr, aliases map[string]aliasResolutionSpec, scope map[string]*typesys.Expr, visiting map[string]bool, resolveSimple func(string) (string, error)) (*typesys.Expr, error) {
	if expr == nil {
		return &typesys.Expr{Kind: typesys.KindAny}, nil
	}
	switch expr.Kind {
	case typesys.KindAny:
		return &typesys.Expr{Kind: typesys.KindAny}, nil
	case typesys.KindLiteral:
		return &typesys.Expr{Kind: typesys.KindLiteral, Name: expr.Name}, nil
	case typesys.KindSimple:
		if scope != nil {
			if bound, ok := scope[expr.Name]; ok {
				return bound, nil
			}
		}
		if builtin := normalizedBuiltinType(expr.Name); builtin != "" {
			return typesys.Parse(builtin)
		}
		if spec, ok := aliases[expr.Name]; ok {
			if len(spec.params) > 0 {
				return nil, fmt.Errorf("generic type alias %s requires type arguments", expr.Name)
			}
			resolved, err := resolveSimple(expr.Name)
			if err != nil {
				return nil, err
			}
			return typesys.Parse(resolved)
		}
		return nil, fmt.Errorf("unknown type alias %s", expr.Name)
	case typesys.KindApplication:
		spec, ok := aliases[expr.Name]
		if !ok {
			return nil, fmt.Errorf("unknown type alias %s", expr.Name)
		}
		if len(spec.params) != len(expr.TypeArgs) {
			return nil, fmt.Errorf("type alias %s expects %d type arguments", expr.Name, len(spec.params))
		}
		key := expr.String()
		if visiting[key] {
			return nil, fmt.Errorf("type alias cycle detected involving %s", expr.Name)
		}
		visiting[key] = true
		defer delete(visiting, key)
		bindings := map[string]*typesys.Expr{}
		for i, param := range spec.params {
			arg, err := instantiateAliasExpr(expr.TypeArgs[i], aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			if param.Constraint != "" {
				constraint, err := instantiateConstraintExpr(param.Constraint, aliases, bindings, visiting, resolveSimple)
				if err != nil {
					return nil, err
				}
				if !typeExprAssignable(constraint, arg) {
					return nil, fmt.Errorf("type argument %s does not satisfy constraint %s for %s", arg.String(), constraint.String(), param.Name)
				}
			}
			bindings[param.Name] = arg
		}
		targetExpr, err := typesys.Parse(spec.target)
		if err != nil {
			return nil, err
		}
		return instantiateAliasExpr(targetExpr, aliases, bindings, visiting, resolveSimple)
	case typesys.KindUnion, typesys.KindIntersection, typesys.KindTuple:
		out := &typesys.Expr{Kind: expr.Kind, Elements: make([]*typesys.Expr, len(expr.Elements))}
		for i, element := range expr.Elements {
			rewritten, err := instantiateAliasExpr(element, aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			out.Elements[i] = rewritten
		}
		return out, nil
	case typesys.KindObject:
		out := &typesys.Expr{Kind: typesys.KindObject, Properties: make([]typesys.Property, len(expr.Properties)), IndexSignatures: make([]typesys.IndexSignature, len(expr.IndexSignatures))}
		for i, property := range expr.Properties {
			rewritten, err := instantiateAliasExpr(property.Type, aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			out.Properties[i] = property
			out.Properties[i].Type = rewritten
		}
		for i, signature := range expr.IndexSignatures {
			keyType, err := instantiateAliasExpr(signature.KeyType, aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			valueType, err := instantiateAliasExpr(signature.ValueType, aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			out.IndexSignatures[i] = signature
			out.IndexSignatures[i].KeyType = keyType
			out.IndexSignatures[i].ValueType = valueType
		}
		return out, nil
	case typesys.KindFunction:
		out := &typesys.Expr{Kind: typesys.KindFunction, Params: make([]*typesys.Expr, len(expr.Params))}
		for i, param := range expr.Params {
			rewritten, err := instantiateAliasExpr(param, aliases, scope, visiting, resolveSimple)
			if err != nil {
				return nil, err
			}
			out.Params[i] = rewritten
		}
		rewrittenReturn, err := instantiateAliasExpr(expr.Return, aliases, scope, visiting, resolveSimple)
		if err != nil {
			return nil, err
		}
		out.Return = rewrittenReturn
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type expression kind %d", expr.Kind)
	}
}

func instantiateConstraintExpr(text string, aliases map[string]aliasResolutionSpec, scope map[string]*typesys.Expr, visiting map[string]bool, resolveSimple func(string) (string, error)) (*typesys.Expr, error) {
	expr, err := typesys.Parse(text)
	if err != nil {
		return nil, err
	}
	return instantiateAliasExpr(expr, aliases, scope, visiting, resolveSimple)
}

func typeExprAssignable(expected *typesys.Expr, actual *typesys.Expr) bool {
	if expected == nil || expected.Kind == typesys.KindAny {
		return true
	}
	if actual == nil {
		return false
	}
	expectedText := strings.TrimSpace(expected.String())
	actualText := strings.TrimSpace(actual.String())
	if expectedText == "" {
		return true
	}
	if actualText == "" {
		return expectedText == ""
	}
	if expectedText == "unknown" {
		return true
	}
	if expectedText == "void" {
		return actualText == "void" || actualText == "undefined"
	}
	switch expected.Kind {
	case typesys.KindLiteral:
		switch expected.Name {
		case "true", "false":
			return actualText == expected.Name || actualText == "boolean"
		default:
			if strings.HasPrefix(expected.Name, "\"") {
				return actualText == expected.Name || actualText == "string"
			}
			return actualText == expected.Name || actualText == "number"
		}
	case typesys.KindUnion:
		for _, member := range expected.Elements {
			if typeExprAssignable(member, actual) {
				return true
			}
		}
		return false
	case typesys.KindIntersection:
		for _, member := range expected.Elements {
			if !typeExprAssignable(member, actual) {
				return false
			}
		}
		return true
	case typesys.KindTuple:
		return actual.Kind == typesys.KindTuple || actualText == "array" || actualText == expectedText
	case typesys.KindObject:
		return actual.Kind == typesys.KindObject || actualText == "object" || actualText == expectedText
	case typesys.KindFunction:
		return actual.Kind == typesys.KindFunction || actualText == "function" || actualText == expectedText
	default:
		return expectedText == actualText
	}
}

func eraseCastExpressions(program *ast.Program) error {
	for _, global := range program.Globals {
		if global == nil {
			continue
		}
		value, err := eraseCastExpression(global.Value)
		if err != nil {
			return err
		}
		global.Value = value
	}
	for _, fn := range program.Functions {
		if fn == nil {
			continue
		}
		if err := eraseStatementCasts(fn.Body); err != nil {
			return err
		}
	}
	for _, classDecl := range program.Classes {
		if classDecl == nil {
			continue
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				value, err := eraseCastExpression(member.Initializer)
				if err != nil {
					return err
				}
				member.Initializer = value
			case *ast.ClassMethodDecl:
				if err := eraseStatementCasts(member.Body); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func normalizedBuiltinType(name string) string {
	expr, err := typesys.Parse(name)
	if err != nil || expr == nil {
		return ""
	}
	switch expr.Kind {
	case typesys.KindAny:
		return ""
	case typesys.KindLiteral, typesys.KindUnion, typesys.KindIntersection, typesys.KindTuple, typesys.KindObject, typesys.KindFunction:
		return expr.String()
	case typesys.KindSimple:
		switch expr.String() {
		case "number", "bigint", "string", "boolean", "symbol", "object", "array", "function", "void", "null", "undefined", "unknown", "never":
			return expr.String()
		default:
			return ""
		}
	default:
		return ""
	}
}

func rewriteFunctionTypeAnnotations(fn *ast.FunctionDecl, rewrite func(string) (string, error)) error {
	for i := range fn.Params {
		rewritten, err := rewrite(fn.Params[i].TypeAnnotation)
		if err != nil {
			return err
		}
		fn.Params[i].TypeAnnotation = rewritten
		if err := rewriteExpressionTypeAnnotations(fn.Params[i].Default, rewrite); err != nil {
			return err
		}
	}
	rewritten, err := rewrite(fn.ReturnType)
	if err != nil {
		return err
	}
	fn.ReturnType = rewritten
	return rewriteStatementsTypeAnnotations(fn.Body, rewrite)
}

func rewriteStatementTypeAnnotations(stmt ast.Statement, rewrite func(string) (string, error)) error {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		rewritten, err := rewrite(stmt.TypeAnnotation)
		if err != nil {
			return err
		}
		stmt.TypeAnnotation = rewritten
		return rewriteExpressionTypeAnnotations(stmt.Value, rewrite)
	case *ast.AssignmentStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Target, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(stmt.Value, rewrite)
	case *ast.ReturnStatement:
		return rewriteExpressionTypeAnnotations(stmt.Value, rewrite)
	case *ast.ExpressionStatement:
		return rewriteExpressionTypeAnnotations(stmt.Expression, rewrite)
	case *ast.DeleteStatement:
		return rewriteExpressionTypeAnnotations(stmt.Target, rewrite)
	case *ast.ThrowStatement:
		return rewriteExpressionTypeAnnotations(stmt.Value, rewrite)
	case *ast.IfStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Condition, rewrite); err != nil {
			return err
		}
		if err := rewriteStatementsTypeAnnotations(stmt.Consequence, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(stmt.Alternative, rewrite)
	case *ast.WhileStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Condition, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(stmt.Body, rewrite)
	case *ast.DoWhileStatement:
		if err := rewriteStatementsTypeAnnotations(stmt.Body, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(stmt.Condition, rewrite)
	case *ast.ForStatement:
		if stmt.Init != nil {
			if err := rewriteStatementTypeAnnotations(stmt.Init, rewrite); err != nil {
				return err
			}
		}
		if err := rewriteExpressionTypeAnnotations(stmt.Condition, rewrite); err != nil {
			return err
		}
		if stmt.Update != nil {
			if err := rewriteStatementTypeAnnotations(stmt.Update, rewrite); err != nil {
				return err
			}
		}
		return rewriteStatementsTypeAnnotations(stmt.Body, rewrite)
	case *ast.ForOfStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Iterable, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(stmt.Body, rewrite)
	case *ast.ForInStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Iterable, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(stmt.Body, rewrite)
	case *ast.SwitchStatement:
		if err := rewriteExpressionTypeAnnotations(stmt.Discriminant, rewrite); err != nil {
			return err
		}
		for i := range stmt.Cases {
			if err := rewriteExpressionTypeAnnotations(stmt.Cases[i].Test, rewrite); err != nil {
				return err
			}
			if err := rewriteStatementsTypeAnnotations(stmt.Cases[i].Consequent, rewrite); err != nil {
				return err
			}
		}
		return rewriteStatementsTypeAnnotations(stmt.Default, rewrite)
	case *ast.BlockStatement:
		return rewriteStatementsTypeAnnotations(stmt.Body, rewrite)
	case *ast.LabeledStatement:
		return rewriteStatementTypeAnnotations(stmt.Statement, rewrite)
	case *ast.TryStatement:
		if err := rewriteStatementsTypeAnnotations(stmt.TryBody, rewrite); err != nil {
			return err
		}
		if err := rewriteStatementsTypeAnnotations(stmt.CatchBody, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(stmt.FinallyBody, rewrite)
	default:
		return nil
	}
}

func rewriteStatementsTypeAnnotations(statements []ast.Statement, rewrite func(string) (string, error)) error {
	for _, stmt := range statements {
		if err := rewriteStatementTypeAnnotations(stmt, rewrite); err != nil {
			return err
		}
	}
	return nil
}

func rewriteExpressionTypeAnnotations(expr ast.Expression, rewrite func(string) (string, error)) error {
	switch expr := expr.(type) {
	case nil:
		return nil
	case *ast.CastExpression:
		rewritten, err := rewrite(expr.TypeAnnotation)
		if err != nil {
			return err
		}
		expr.TypeAnnotation = rewritten
		return rewriteExpressionTypeAnnotations(expr.Value, rewrite)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			if err := rewriteExpressionTypeAnnotations(arg, rewrite); err != nil {
				return err
			}
		}
	case *ast.InvokeExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Callee, rewrite); err != nil {
			return err
		}
		for _, arg := range expr.Arguments {
			if err := rewriteExpressionTypeAnnotations(arg, rewrite); err != nil {
				return err
			}
		}
	case *ast.NewExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Callee, rewrite); err != nil {
			return err
		}
		for _, arg := range expr.Arguments {
			if err := rewriteExpressionTypeAnnotations(arg, rewrite); err != nil {
				return err
			}
		}
	case *ast.ClosureExpression:
		return rewriteExpressionTypeAnnotations(expr.Environment, rewrite)
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Computed {
				if err := rewriteExpressionTypeAnnotations(property.KeyExpr, rewrite); err != nil {
					return err
				}
			}
			if err := rewriteExpressionTypeAnnotations(property.Value, rewrite); err != nil {
				return err
			}
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if err := rewriteExpressionTypeAnnotations(element, rewrite); err != nil {
				return err
			}
		}
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			if err := rewriteExpressionTypeAnnotations(value, rewrite); err != nil {
				return err
			}
		}
	case *ast.SpreadExpression:
		return rewriteExpressionTypeAnnotations(expr.Value, rewrite)
	case *ast.BinaryExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.ComparisonExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.LogicalExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.NullishCoalesceExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.CommaExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.ConditionalExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Condition, rewrite); err != nil {
			return err
		}
		if err := rewriteExpressionTypeAnnotations(expr.Consequent, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Alternative, rewrite)
	case *ast.UnaryExpression:
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.TypeofExpression:
		return rewriteExpressionTypeAnnotations(expr.Value, rewrite)
	case *ast.TypeCheckExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Value, rewrite); err != nil {
			return err
		}
		rewritten, err := typesys.RewriteAliases(expr.TypeAnnotation, rewrite)
		if err != nil {
			return err
		}
		expr.TypeAnnotation = rewritten
		return nil
	case *ast.AwaitExpression:
		return rewriteExpressionTypeAnnotations(expr.Value, rewrite)
	case *ast.YieldExpression:
		return rewriteExpressionTypeAnnotations(expr.Value, rewrite)
	case *ast.InstanceofExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Left, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Right, rewrite)
	case *ast.IndexExpression:
		if err := rewriteExpressionTypeAnnotations(expr.Target, rewrite); err != nil {
			return err
		}
		return rewriteExpressionTypeAnnotations(expr.Index, rewrite)
	case *ast.MemberExpression:
		return rewriteExpressionTypeAnnotations(expr.Target, rewrite)
	case *ast.FunctionExpression:
		for i := range expr.Params {
			rewritten, err := rewrite(expr.Params[i].TypeAnnotation)
			if err != nil {
				return err
			}
			expr.Params[i].TypeAnnotation = rewritten
			if err := rewriteExpressionTypeAnnotations(expr.Params[i].Default, rewrite); err != nil {
				return err
			}
		}
		rewritten, err := rewrite(expr.ReturnType)
		if err != nil {
			return err
		}
		expr.ReturnType = rewritten
		if err := rewriteExpressionTypeAnnotations(expr.ExpressionBody, rewrite); err != nil {
			return err
		}
		return rewriteStatementsTypeAnnotations(expr.Body, rewrite)
	}
	return nil
}

func eraseStatementCasts(statements []ast.Statement) error {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			value, err := eraseCastExpression(stmt.Value)
			if err != nil {
				return err
			}
			stmt.Value = value
		case *ast.AssignmentStatement:
			target, err := eraseCastExpression(stmt.Target)
			if err != nil {
				return err
			}
			value, err := eraseCastExpression(stmt.Value)
			if err != nil {
				return err
			}
			stmt.Target = target
			stmt.Value = value
		case *ast.ReturnStatement:
			value, err := eraseCastExpression(stmt.Value)
			if err != nil {
				return err
			}
			stmt.Value = value
		case *ast.ExpressionStatement:
			value, err := eraseCastExpression(stmt.Expression)
			if err != nil {
				return err
			}
			stmt.Expression = value
		case *ast.DeleteStatement:
			target, err := eraseCastExpression(stmt.Target)
			if err != nil {
				return err
			}
			stmt.Target = target
		case *ast.ThrowStatement:
			value, err := eraseCastExpression(stmt.Value)
			if err != nil {
				return err
			}
			stmt.Value = value
		case *ast.IfStatement:
			condition, err := eraseCastExpression(stmt.Condition)
			if err != nil {
				return err
			}
			stmt.Condition = condition
			if err := eraseStatementCasts(stmt.Consequence); err != nil {
				return err
			}
			if err := eraseStatementCasts(stmt.Alternative); err != nil {
				return err
			}
		case *ast.WhileStatement:
			condition, err := eraseCastExpression(stmt.Condition)
			if err != nil {
				return err
			}
			stmt.Condition = condition
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
		case *ast.DoWhileStatement:
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
			condition, err := eraseCastExpression(stmt.Condition)
			if err != nil {
				return err
			}
			stmt.Condition = condition
		case *ast.ForStatement:
			if stmt.Init != nil {
				if err := eraseStatementCasts([]ast.Statement{stmt.Init}); err != nil {
					return err
				}
			}
			condition, err := eraseCastExpression(stmt.Condition)
			if err != nil {
				return err
			}
			stmt.Condition = condition
			if stmt.Update != nil {
				if err := eraseStatementCasts([]ast.Statement{stmt.Update}); err != nil {
					return err
				}
			}
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
		case *ast.ForOfStatement:
			iterable, err := eraseCastExpression(stmt.Iterable)
			if err != nil {
				return err
			}
			stmt.Iterable = iterable
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
		case *ast.ForInStatement:
			iterable, err := eraseCastExpression(stmt.Iterable)
			if err != nil {
				return err
			}
			stmt.Iterable = iterable
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
		case *ast.SwitchStatement:
			discriminant, err := eraseCastExpression(stmt.Discriminant)
			if err != nil {
				return err
			}
			stmt.Discriminant = discriminant
			for i := range stmt.Cases {
				test, err := eraseCastExpression(stmt.Cases[i].Test)
				if err != nil {
					return err
				}
				stmt.Cases[i].Test = test
				if err := eraseStatementCasts(stmt.Cases[i].Consequent); err != nil {
					return err
				}
			}
			if err := eraseStatementCasts(stmt.Default); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := eraseStatementCasts(stmt.Body); err != nil {
				return err
			}
		case *ast.LabeledStatement:
			if err := eraseStatementCasts([]ast.Statement{stmt.Statement}); err != nil {
				return err
			}
		case *ast.TryStatement:
			if err := eraseStatementCasts(stmt.TryBody); err != nil {
				return err
			}
			if err := eraseStatementCasts(stmt.CatchBody); err != nil {
				return err
			}
			if err := eraseStatementCasts(stmt.FinallyBody); err != nil {
				return err
			}
		}
	}
	return nil
}

func eraseCastExpression(expr ast.Expression) (ast.Expression, error) {
	switch expr := expr.(type) {
	case nil:
		return nil, nil
	case *ast.CastExpression:
		return eraseCastExpression(expr.Value)
	case *ast.CallExpression:
		for i, arg := range expr.Arguments {
			rewritten, err := eraseCastExpression(arg)
			if err != nil {
				return nil, err
			}
			expr.Arguments[i] = rewritten
		}
		return expr, nil
	case *ast.InvokeExpression:
		callee, err := eraseCastExpression(expr.Callee)
		if err != nil {
			return nil, err
		}
		expr.Callee = callee
		for i, arg := range expr.Arguments {
			rewritten, err := eraseCastExpression(arg)
			if err != nil {
				return nil, err
			}
			expr.Arguments[i] = rewritten
		}
		return expr, nil
	case *ast.NewExpression:
		callee, err := eraseCastExpression(expr.Callee)
		if err != nil {
			return nil, err
		}
		expr.Callee = callee
		for i, arg := range expr.Arguments {
			rewritten, err := eraseCastExpression(arg)
			if err != nil {
				return nil, err
			}
			expr.Arguments[i] = rewritten
		}
		return expr, nil
	case *ast.ClosureExpression:
		environment, err := eraseCastExpression(expr.Environment)
		if err != nil {
			return nil, err
		}
		expr.Environment = environment
		return expr, nil
	case *ast.ObjectLiteral:
		for i, property := range expr.Properties {
			if property.Computed {
				keyExpr, err := eraseCastExpression(property.KeyExpr)
				if err != nil {
					return nil, err
				}
				property.KeyExpr = keyExpr
			}
			value, err := eraseCastExpression(property.Value)
			if err != nil {
				return nil, err
			}
			property.Value = value
			expr.Properties[i] = property
		}
		return expr, nil
	case *ast.ArrayLiteral:
		for i, element := range expr.Elements {
			rewritten, err := eraseCastExpression(element)
			if err != nil {
				return nil, err
			}
			expr.Elements[i] = rewritten
		}
		return expr, nil
	case *ast.TemplateLiteral:
		for i, value := range expr.Values {
			rewritten, err := eraseCastExpression(value)
			if err != nil {
				return nil, err
			}
			expr.Values[i] = rewritten
		}
		return expr, nil
	case *ast.SpreadExpression:
		value, err := eraseCastExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		expr.Value = value
		return expr, nil
	case *ast.BinaryExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.ComparisonExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.LogicalExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.NullishCoalesceExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.CommaExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.ConditionalExpression:
		condition, err := eraseCastExpression(expr.Condition)
		if err != nil {
			return nil, err
		}
		consequent, err := eraseCastExpression(expr.Consequent)
		if err != nil {
			return nil, err
		}
		alternative, err := eraseCastExpression(expr.Alternative)
		if err != nil {
			return nil, err
		}
		expr.Condition = condition
		expr.Consequent = consequent
		expr.Alternative = alternative
		return expr, nil
	case *ast.UnaryExpression:
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Right = right
		return expr, nil
	case *ast.TypeofExpression:
		value, err := eraseCastExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		expr.Value = value
		return expr, nil
	case *ast.TypeCheckExpression:
		value, err := eraseCastExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		expr.Value = value
		return expr, nil
	case *ast.AwaitExpression:
		value, err := eraseCastExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		expr.Value = value
		return expr, nil
	case *ast.YieldExpression:
		value, err := eraseCastExpression(expr.Value)
		if err != nil {
			return nil, err
		}
		expr.Value = value
		return expr, nil
	case *ast.InstanceofExpression:
		left, err := eraseCastExpression(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := eraseCastExpression(expr.Right)
		if err != nil {
			return nil, err
		}
		expr.Left = left
		expr.Right = right
		return expr, nil
	case *ast.IndexExpression:
		target, err := eraseCastExpression(expr.Target)
		if err != nil {
			return nil, err
		}
		index, err := eraseCastExpression(expr.Index)
		if err != nil {
			return nil, err
		}
		expr.Target = target
		expr.Index = index
		return expr, nil
	case *ast.MemberExpression:
		target, err := eraseCastExpression(expr.Target)
		if err != nil {
			return nil, err
		}
		expr.Target = target
		return expr, nil
	case *ast.FunctionExpression:
		value, err := eraseCastExpression(expr.ExpressionBody)
		if err != nil {
			return nil, err
		}
		expr.ExpressionBody = value
		if err := eraseStatementCasts(expr.Body); err != nil {
			return nil, err
		}
		return expr, nil
	default:
		return expr, nil
	}
}
