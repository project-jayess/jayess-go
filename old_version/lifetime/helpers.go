package lifetime

import (
	"strings"

	"jayess-go/ast"
)

type assignmentEscapeKind int

const (
	assignmentEscapesNone assignmentEscapeKind = iota
	assignmentEscapesGlobalState
	assignmentEscapesOuterScope
)

type storageTargetKind int

const (
	storageTargetNone storageTargetKind = iota
	storageTargetObject
	storageTargetArray
)

func normalizedLocalName(name string, locals map[string]bool) string {
	if locals[name] {
		return normalizedDisplayName(name)
	}
	normalized := normalizedDisplayName(name)
	if normalized != name && locals[normalized] {
		return normalized
	}
	return ""
}

func normalizedDisplayName(name string) string {
	if strings.HasPrefix(name, "__jayess_cell_") {
		return strings.TrimPrefix(name, "__jayess_cell_")
	}
	return name
}

func capturedEnvName(target ast.Expression, property string) string {
	ident, ok := target.(*ast.Identifier)
	if !ok || ident.Name != "__env" || property == "" {
		return ""
	}
	return property
}

func capturedEnvIndexName(target ast.Expression, index ast.Expression) string {
	ident, ok := target.(*ast.Identifier)
	if !ok || ident.Name != "__env" {
		return ""
	}
	literal, ok := index.(*ast.StringLiteral)
	if !ok || literal.Value == "" {
		return ""
	}
	return literal.Value
}

func escapeTargetKind(target ast.Expression, globals map[string]bool) assignmentEscapeKind {
	root := rootIdentifier(target)
	switch {
	case root == "__env":
		return assignmentEscapesOuterScope
	case root != "" && globals[root]:
		return assignmentEscapesGlobalState
	default:
		return assignmentEscapesNone
	}
}

func classifyStorageTarget(target ast.Expression) storageTargetKind {
	switch target := target.(type) {
	case *ast.MemberExpression:
		return storageTargetObject
	case *ast.IndexExpression:
		if literal, ok := target.Index.(*ast.StringLiteral); ok && literal.Value != "" {
			return storageTargetObject
		}
		return storageTargetArray
	default:
		return storageTargetNone
	}
}

func rootIdentifier(expr ast.Expression) string {
	switch expr := expr.(type) {
	case *ast.Identifier:
		return expr.Name
	case *ast.MemberExpression:
		return rootIdentifier(expr.Target)
	case *ast.IndexExpression:
		return rootIdentifier(expr.Target)
	default:
		return ""
	}
}

func (a *Analyzer) callRequiresConservativeEscape(callee string) bool {
	if callee == "" {
		return true
	}
	if knownRuntimeCalls[callee] {
		return false
	}
	if strings.HasPrefix(callee, "__jayess_") {
		return false
	}
	if a.externs[callee] {
		if a.borrowedExterns[callee] {
			return false
		}
		return true
	}
	if a.functions[callee] || a.classes[callee] {
		return false
	}
	return true
}

func (a *Analyzer) newRequiresConservativeEscape(callee ast.Expression) bool {
	ident, ok := callee.(*ast.Identifier)
	if !ok {
		return true
	}
	if strings.HasPrefix(ident.Name, "__jayess_") {
		return false
	}
	return !a.classes[ident.Name]
}

func (a *Analyzer) returnRequiresConservativeEscape(expr ast.Expression) bool {
	switch expr := expr.(type) {
	case *ast.CallExpression:
		return a.callRequiresConservativeEscape(expr.Callee)
	case *ast.NewExpression:
		return a.newRequiresConservativeEscape(expr.Callee)
	default:
		return true
	}
}
