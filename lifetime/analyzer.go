package lifetime

import (
	"fmt"
	"sort"
	"strings"

	"jayess-go/ast"
)

type Finding struct {
	Line    int
	Column  int
	Message string
}

type Report struct {
	EscapesDetected bool
	Findings        []Finding
}

type Analyzer struct {
	findings     []Finding
	seen         map[findingKey]bool
	functions    map[string]bool
	externs      map[string]bool
	classes      map[string]bool
}

var knownRuntimeCalls = map[string]bool{
	"print":        true,
	"sleep":        true,
	"readLine":     true,
	"readKey":      true,
	"sleepAsync":   true,
	"setTimeout":   true,
	"clearTimeout": true,
}

type findingKey struct {
	line    int
	column  int
	message string
}

func New() *Analyzer {
	return &Analyzer{
		seen:      map[findingKey]bool{},
		functions: map[string]bool{},
		externs:   map[string]bool{},
		classes:   map[string]bool{},
	}
}

func (a *Analyzer) Analyze(program *ast.Program) Report {
	globals := map[string]bool{}
	for _, global := range program.Globals {
		globals[global.Name] = true
	}
	for _, extern := range program.ExternFunctions {
		a.externs[extern.Name] = true
	}
	for _, fn := range program.Functions {
		a.functions[fn.Name] = true
	}
	for _, classDecl := range program.Classes {
		a.classes[classDecl.Name] = true
	}
	for _, fn := range program.Functions {
		a.analyzeFunction(fn.Params, fn.Body, globals)
	}
	for _, classDecl := range program.Classes {
		for _, member := range classDecl.Members {
			method, ok := member.(*ast.ClassMethodDecl)
			if !ok {
				continue
			}
			a.analyzeFunction(method.Params, method.Body, globals)
		}
	}
	sort.Slice(a.findings, func(i, j int) bool {
		if a.findings[i].Line != a.findings[j].Line {
			return a.findings[i].Line < a.findings[j].Line
		}
		if a.findings[i].Column != a.findings[j].Column {
			return a.findings[i].Column < a.findings[j].Column
		}
		return a.findings[i].Message < a.findings[j].Message
	})
	return Report{
		EscapesDetected: len(a.findings) > 0,
		Findings:        append([]Finding{}, a.findings...),
	}
}

func (a *Analyzer) analyzeFunction(params []ast.Parameter, body []ast.Statement, globals map[string]bool) {
	locals := map[string]bool{}
	for _, param := range params {
		if param.Name != "" {
			locals[param.Name] = true
		}
		collectPatternNames(param.Pattern, locals)
	}
	collectDeclaredNames(body, locals)
	a.analyzeStatements(body, locals, globals)
}

func collectDeclaredNames(statements []ast.Statement, locals map[string]bool) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			locals[stmt.Name] = true
		case *ast.DestructuringDecl:
			collectPatternNames(stmt.Pattern, locals)
		case *ast.IfStatement:
			collectDeclaredNames(stmt.Consequence, locals)
			collectDeclaredNames(stmt.Alternative, locals)
		case *ast.WhileStatement:
			collectDeclaredNames(stmt.Body, locals)
		case *ast.DoWhileStatement:
			collectDeclaredNames(stmt.Body, locals)
		case *ast.ForStatement:
			if stmt.Init != nil {
				collectDeclaredNames([]ast.Statement{stmt.Init}, locals)
			}
			collectDeclaredNames(stmt.Body, locals)
		case *ast.ForOfStatement:
			locals[stmt.Name] = true
			collectDeclaredNames(stmt.Body, locals)
		case *ast.ForInStatement:
			locals[stmt.Name] = true
			collectDeclaredNames(stmt.Body, locals)
		case *ast.BlockStatement:
			collectDeclaredNames(stmt.Body, locals)
		case *ast.SwitchStatement:
			for _, switchCase := range stmt.Cases {
				collectDeclaredNames(switchCase.Consequent, locals)
			}
			collectDeclaredNames(stmt.Default, locals)
		case *ast.TryStatement:
			if stmt.CatchName != "" {
				locals[stmt.CatchName] = true
			}
			collectDeclaredNames(stmt.TryBody, locals)
			collectDeclaredNames(stmt.CatchBody, locals)
			collectDeclaredNames(stmt.FinallyBody, locals)
		case *ast.LabeledStatement:
			collectDeclaredNames([]ast.Statement{stmt.Statement}, locals)
		}
	}
}

func collectPatternNames(pattern ast.Pattern, locals map[string]bool) {
	switch pattern := pattern.(type) {
	case *ast.IdentifierPattern:
		locals[pattern.Name] = true
	case *ast.ObjectPattern:
		for _, property := range pattern.Properties {
			collectPatternNames(property.Pattern, locals)
		}
		if pattern.Rest != "" {
			locals[pattern.Rest] = true
		}
	case *ast.ArrayPattern:
		for _, element := range pattern.Elements {
			collectPatternNames(element.Pattern, locals)
		}
	}
}

func (a *Analyzer) analyzeStatements(statements []ast.Statement, locals map[string]bool, globals map[string]bool) {
	for _, stmt := range statements {
		a.analyzeStatement(stmt, locals, globals)
	}
}

func (a *Analyzer) analyzeStatement(stmt ast.Statement, locals map[string]bool, globals map[string]bool) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		a.analyzeExpression(stmt.Value, locals)
	case *ast.DestructuringDecl:
		a.analyzeExpression(stmt.Value, locals)
	case *ast.AssignmentStatement:
		a.analyzeExpression(stmt.Target, locals)
		a.analyzeExpression(stmt.Value, locals)
		switch escapeTargetKind(stmt.Target, globals) {
		case assignmentEscapesGlobalState:
			for _, name := range retainedLocals(stmt.Value, locals) {
				a.addFinding(stmt.Value, "local %s escapes via assignment to global state", name)
			}
		case assignmentEscapesOuterScope:
			for _, name := range retainedLocals(stmt.Value, locals) {
				a.addFinding(stmt.Value, "local %s escapes via assignment to outer scope", name)
			}
		}
		switch classifyStorageTarget(stmt.Target) {
		case storageTargetObject:
			for _, name := range retainedLocals(stmt.Value, locals) {
				a.addFinding(stmt.Value, "local %s escapes via object storage", name)
			}
		case storageTargetArray:
			for _, name := range retainedLocals(stmt.Value, locals) {
				a.addFinding(stmt.Value, "local %s escapes via array storage", name)
			}
		}
	case *ast.DestructuringAssignment:
		a.analyzeExpression(stmt.Value, locals)
	case *ast.ReturnStatement:
		if stmt.Value != nil {
			a.analyzeExpression(stmt.Value, locals)
			for _, name := range retainedLocals(stmt.Value, locals) {
				a.addFinding(stmt.Value, "local %s escapes via return", name)
			}
		}
	case *ast.ExpressionStatement:
		a.analyzeExpression(stmt.Expression, locals)
	case *ast.DeleteStatement:
		a.analyzeExpression(stmt.Target, locals)
	case *ast.ThrowStatement:
		a.analyzeExpression(stmt.Value, locals)
	case *ast.IfStatement:
		a.analyzeExpression(stmt.Condition, locals)
		a.analyzeStatements(stmt.Consequence, locals, globals)
		a.analyzeStatements(stmt.Alternative, locals, globals)
	case *ast.WhileStatement:
		a.analyzeExpression(stmt.Condition, locals)
		a.analyzeStatements(stmt.Body, locals, globals)
	case *ast.DoWhileStatement:
		a.analyzeStatements(stmt.Body, locals, globals)
		a.analyzeExpression(stmt.Condition, locals)
	case *ast.ForStatement:
		if stmt.Init != nil {
			a.analyzeStatement(stmt.Init, locals, globals)
		}
		if stmt.Condition != nil {
			a.analyzeExpression(stmt.Condition, locals)
		}
		if stmt.Update != nil {
			a.analyzeStatement(stmt.Update, locals, globals)
		}
		a.analyzeStatements(stmt.Body, locals, globals)
	case *ast.ForOfStatement:
		a.analyzeExpression(stmt.Iterable, locals)
		a.analyzeStatements(stmt.Body, locals, globals)
	case *ast.ForInStatement:
		a.analyzeExpression(stmt.Iterable, locals)
		a.analyzeStatements(stmt.Body, locals, globals)
	case *ast.SwitchStatement:
		a.analyzeExpression(stmt.Discriminant, locals)
		for _, switchCase := range stmt.Cases {
			a.analyzeExpression(switchCase.Test, locals)
			a.analyzeStatements(switchCase.Consequent, locals, globals)
		}
		a.analyzeStatements(stmt.Default, locals, globals)
	case *ast.BlockStatement:
		a.analyzeStatements(stmt.Body, locals, globals)
	case *ast.LabeledStatement:
		a.analyzeStatement(stmt.Statement, locals, globals)
	case *ast.TryStatement:
		a.analyzeStatements(stmt.TryBody, locals, globals)
		a.analyzeStatements(stmt.CatchBody, locals, globals)
		a.analyzeStatements(stmt.FinallyBody, locals, globals)
	}
}

func (a *Analyzer) analyzeExpression(expr ast.Expression, locals map[string]bool) {
	switch expr := expr.(type) {
	case nil:
		return
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			a.analyzeExpression(property.KeyExpr, locals)
			a.analyzeExpression(property.Value, locals)
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			a.analyzeExpression(element, locals)
		}
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			a.analyzeExpression(value, locals)
		}
	case *ast.SpreadExpression:
		a.analyzeExpression(expr.Value, locals)
	case *ast.ClosureExpression:
		for _, name := range retainedLocals(expr.Environment, locals) {
			a.addFinding(expr, "local %s escapes via closure capture", name)
		}
		a.analyzeExpression(expr.Environment, locals)
	case *ast.BinaryExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.ComparisonExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.LogicalExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.NullishCoalesceExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.CommaExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.ConditionalExpression:
		a.analyzeExpression(expr.Condition, locals)
		a.analyzeExpression(expr.Consequent, locals)
		a.analyzeExpression(expr.Alternative, locals)
	case *ast.UnaryExpression:
		a.analyzeExpression(expr.Right, locals)
	case *ast.TypeofExpression:
		a.analyzeExpression(expr.Value, locals)
	case *ast.InstanceofExpression:
		a.analyzeExpression(expr.Left, locals)
		a.analyzeExpression(expr.Right, locals)
	case *ast.IndexExpression:
		a.analyzeExpression(expr.Target, locals)
		a.analyzeExpression(expr.Index, locals)
	case *ast.MemberExpression:
		a.analyzeExpression(expr.Target, locals)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			a.analyzeExpression(arg, locals)
		}
		if a.callRequiresConservativeEscape(expr.Callee) {
			for _, arg := range expr.Arguments {
				for _, name := range retainedLocals(arg, locals) {
					a.addFinding(arg, "local %s escapes via call to unknown or external function", name)
				}
			}
		}
	case *ast.InvokeExpression:
		a.analyzeExpression(expr.Callee, locals)
		for _, arg := range expr.Arguments {
			a.analyzeExpression(arg, locals)
		}
		for _, arg := range expr.Arguments {
			for _, name := range retainedLocals(arg, locals) {
				a.addFinding(arg, "local %s escapes via call to unknown or external function", name)
			}
		}
	case *ast.NewExpression:
		a.analyzeExpression(expr.Callee, locals)
		for _, arg := range expr.Arguments {
			a.analyzeExpression(arg, locals)
		}
		if a.newRequiresConservativeEscape(expr.Callee) {
			for _, arg := range expr.Arguments {
				for _, name := range retainedLocals(arg, locals) {
					a.addFinding(arg, "local %s escapes via call to unknown or external function", name)
				}
			}
		}
	case *ast.AwaitExpression:
		a.analyzeExpression(expr.Value, locals)
	case *ast.YieldExpression:
		a.analyzeExpression(expr.Value, locals)
	}
}

func retainedLocals(expr ast.Expression, locals map[string]bool) []string {
	names := map[string]bool{}
	collectRetainedLocals(expr, locals, names)
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func collectRetainedLocals(expr ast.Expression, locals map[string]bool, names map[string]bool) {
	switch expr := expr.(type) {
	case nil:
		return
	case *ast.Identifier:
		if localName := normalizedLocalName(expr.Name, locals); localName != "" {
			names[localName] = true
		}
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Spread {
				continue
			}
			collectRetainedLocals(property.Value, locals, names)
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if spread, ok := element.(*ast.SpreadExpression); ok {
				_ = spread
				continue
			}
			collectRetainedLocals(element, locals, names)
		}
	case *ast.ClosureExpression:
		collectRetainedLocals(expr.Environment, locals, names)
	case *ast.MemberExpression:
		if envName := capturedEnvName(expr.Target, expr.Property); envName != "" {
			names[normalizedDisplayName(envName)] = true
			return
		}
		collectRetainedLocals(expr.Target, locals, names)
	case *ast.IndexExpression:
		if envName := capturedEnvIndexName(expr.Target, expr.Index); envName != "" {
			names[normalizedDisplayName(envName)] = true
			return
		}
		collectRetainedLocals(expr.Target, locals, names)
		collectRetainedLocals(expr.Index, locals, names)
	case *ast.CommaExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.ConditionalExpression:
		collectRetainedLocals(expr.Condition, locals, names)
		collectRetainedLocals(expr.Consequent, locals, names)
		collectRetainedLocals(expr.Alternative, locals, names)
	case *ast.NullishCoalesceExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.LogicalExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.BinaryExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.ComparisonExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.UnaryExpression:
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.TypeofExpression:
		collectRetainedLocals(expr.Value, locals, names)
	case *ast.InstanceofExpression:
		collectRetainedLocals(expr.Left, locals, names)
		collectRetainedLocals(expr.Right, locals, names)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			collectRetainedLocals(arg, locals, names)
		}
	case *ast.InvokeExpression:
		collectRetainedLocals(expr.Callee, locals, names)
		for _, arg := range expr.Arguments {
			collectRetainedLocals(arg, locals, names)
		}
	case *ast.NewExpression:
		collectRetainedLocals(expr.Callee, locals, names)
		for _, arg := range expr.Arguments {
			collectRetainedLocals(arg, locals, names)
		}
	case *ast.AwaitExpression:
		collectRetainedLocals(expr.Value, locals, names)
	case *ast.YieldExpression:
		collectRetainedLocals(expr.Value, locals, names)
	}
}

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

type assignmentEscapeKind int

const (
	assignmentEscapesNone assignmentEscapeKind = iota
	assignmentEscapesGlobalState
	assignmentEscapesOuterScope
)

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

type storageTargetKind int

const (
	storageTargetNone storageTargetKind = iota
	storageTargetObject
	storageTargetArray
)

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

func (a *Analyzer) addFinding(node any, format string, args ...any) {
	pos := ast.PositionOf(node)
	finding := Finding{
		Line:    pos.Line,
		Column:  pos.Column,
		Message: sprintf(format, args...),
	}
	key := findingKey{line: finding.Line, column: finding.Column, message: finding.Message}
	if a.seen[key] {
		return
	}
	a.seen[key] = true
	a.findings = append(a.findings, finding)
}

func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
