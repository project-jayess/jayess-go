package semantic

import (
	"strings"

	"jayess-go/ast"
)

func analyzeImportDeclaration(scope *scope, stmt *ast.ImportDecl) error {
	if err := validateModuleSource(stmt, stmt.Source); err != nil {
		return err
	}
	for _, specifier := range stmt.Specifiers {
		if !scope.declareImported(specifier.Local) {
			return errorAt(stmt, "duplicate declaration %s", specifier.Local)
		}
	}
	return nil
}

func analyzeExportDeclaration(scope *scope, context controlContext, stmt *ast.ExportDecl) error {
	if stmt.Declaration != nil {
		return analyzeStatement(scope, context, stmt.Declaration)
	}
	if stmt.Value != nil {
		return analyzeExpression(scope, stmt.Value)
	}
	if stmt.Source != "" {
		return validateModuleSource(stmt, stmt.Source)
	}
	for _, specifier := range stmt.Specifiers {
		if !scope.lookup(specifier.Local) {
			return errorAt(stmt, "export of %s before declaration", specifier.Local)
		}
	}
	return nil
}

func validateModuleSource(node ast.Node, source string) error {
	if source == "" {
		return errorAt(node, "unsupported empty module source")
	}
	if strings.Contains(source, "\\") {
		return errorAt(node, "unsupported module source %q; use / as the module path separator", source)
	}
	if isRelativeModuleSource(source) || isPackageModuleSource(source) {
		return nil
	}
	return errorAt(node, "unsupported module source %q; expected relative or package specifier", source)
}

func isRelativeModuleSource(source string) bool {
	return strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../")
}

func isPackageModuleSource(source string) bool {
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || isSchemeLikeModuleSource(source) {
		return false
	}
	if hasInvalidPackageModuleSegment(source) {
		return false
	}
	if strings.HasPrefix(source, "@") {
		return isScopedPackageModuleSource(source)
	}
	return true
}

func hasInvalidPackageModuleSegment(source string) bool {
	for _, segment := range strings.Split(source, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func isScopedPackageModuleSource(source string) bool {
	rest := strings.TrimPrefix(source, "@")
	slash := strings.IndexByte(rest, '/')
	return slash > 0 && slash < len(rest)-1
}

func isSchemeLikeModuleSource(source string) bool {
	colon := strings.IndexByte(source, ':')
	if colon <= 0 {
		return false
	}
	for i := 0; i < colon; i++ {
		ch := source[i]
		if isModuleSourceSchemeChar(ch, i == 0) {
			continue
		}
		return false
	}
	return true
}

func isModuleSourceSchemeChar(ch byte, first bool) bool {
	if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' {
		return true
	}
	if first {
		return false
	}
	return ch >= '0' && ch <= '9' || ch == '+' || ch == '-' || ch == '.'
}
