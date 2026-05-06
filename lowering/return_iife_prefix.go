package lowering

import "jayess-go/ast"

func collectIIFEPrefixBindings(prefix []ast.Statement, selfName string, params []ast.Parameter, bindings returnScope) ([]string, []string, []string, []string, bool) {
	paramNames := iifeParameterNames(params)
	seenLocals := make(map[string]bool)
	var locals []string
	var shadows []string
	var funcLocals []string
	var funcShadows []string
	for _, statement := range iifePrefixFunctionDecls(prefix) {
		decl, ok := statement.(*ast.FunctionDecl)
		if !ok || decl.Name == "" {
			continue
		}
		if paramNames[decl.Name] || seenLocals[decl.Name] {
			continue
		}
		seenLocals[decl.Name] = true
		if bindingKnown(bindings, decl.Name) {
			funcShadows = append(funcShadows, decl.Name)
		} else {
			funcLocals = append(funcLocals, decl.Name)
		}
	}
	for _, statement := range iifePrefixVariableDecls(prefix) {
		decl, ok := statement.(*ast.VariableDecl)
		if !ok {
			continue
		}
		for _, name := range iifeVariableDeclNames(decl) {
			if paramNames[name] {
				continue
			}
			if name == selfName {
				continue
			}
			if seenLocals[name] {
				continue
			}
			seenLocals[name] = true
			if bindingKnown(bindings, name) {
				shadows = append(shadows, name)
			} else {
				locals = append(locals, name)
			}
		}
	}
	return locals, shadows, funcLocals, funcShadows, true
}

func iifeVariableDeclNames(decl *ast.VariableDecl) []string {
	if decl.Name != "" {
		return []string{decl.Name}
	}
	return bindingNames(decl.Pattern)
}

func iifePrefixFunctionDecls(statements []ast.Statement) []ast.Statement {
	var declarations []ast.Statement
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.FunctionDecl:
			declarations = append(declarations, stmt)
		case *ast.BlockStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Statements)...)
		case *ast.IfStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Consequence)...)
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Alternative)...)
		case *ast.WhileStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Body)...)
		case *ast.DoWhileStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Body)...)
		case *ast.ForStatement:
			declarations = append(declarations, iifePrefixFunctionDecls([]ast.Statement{stmt.Init})...)
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Body)...)
		case *ast.ForOfStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Body)...)
		case *ast.ForInStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Body)...)
		case *ast.LabeledStatement:
			declarations = append(declarations, iifePrefixFunctionDecls([]ast.Statement{stmt.Statement})...)
		case *ast.SwitchStatement:
			for _, switchCase := range stmt.Cases {
				declarations = append(declarations, iifePrefixFunctionDecls(switchCase.Consequent)...)
			}
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.Default)...)
		case *ast.TryStatement:
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.TryBody)...)
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.CatchBody)...)
			declarations = append(declarations, iifePrefixFunctionDecls(stmt.FinallyBody)...)
		}
	}
	return declarations
}

func iifePrefixVariableDecls(statements []ast.Statement) []ast.Statement {
	var declarations []ast.Statement
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			declarations = append(declarations, stmt)
		case *ast.BlockStatement:
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Statements)...)
		case *ast.IfStatement:
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Consequence)...)
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Alternative)...)
		case *ast.WhileStatement:
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Body)...)
		case *ast.DoWhileStatement:
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Body)...)
		case *ast.ForStatement:
			declarations = append(declarations, iifePrefixVariableDecls([]ast.Statement{stmt.Init})...)
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Body)...)
		case *ast.ForOfStatement:
			if stmt.Kind == ast.DeclarationVar {
				declarations = append(declarations, &ast.VariableDecl{Name: stmt.Name, Pattern: stmt.Pattern})
			}
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Body)...)
		case *ast.ForInStatement:
			if stmt.Kind == ast.DeclarationVar {
				declarations = append(declarations, &ast.VariableDecl{Name: stmt.Name, Pattern: stmt.Pattern})
			}
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Body)...)
		case *ast.LabeledStatement:
			declarations = append(declarations, iifePrefixVariableDecls([]ast.Statement{stmt.Statement})...)
		case *ast.SwitchStatement:
			for _, switchCase := range stmt.Cases {
				declarations = append(declarations, iifePrefixVariableDecls(switchCase.Consequent)...)
			}
			declarations = append(declarations, iifePrefixVariableDecls(stmt.Default)...)
		case *ast.TryStatement:
			declarations = append(declarations, iifePrefixVariableDecls(stmt.TryBody)...)
			declarations = append(declarations, iifePrefixVariableDecls(stmt.CatchBody)...)
			declarations = append(declarations, iifePrefixVariableDecls(stmt.FinallyBody)...)
		}
	}
	return declarations
}
