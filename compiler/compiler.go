package compiler

import (
	"errors"
	"fmt"
	"strings"

	"jayess-go/ast"
	"jayess-go/codegen"
	"jayess-go/lexer"
	"jayess-go/lifetime"
	"jayess-go/lowering"
	"jayess-go/parser"
	"jayess-go/semantic"
)

type Options struct {
	TargetTriple             string
	WarningPolicy            string
	AllowedWarningCategories []string
	OptimizationLevel        string
	TargetCPU                string
	TargetFeatures           []string
	RelocationModel          string
	CodeModel                string
}

type Diagnostic struct {
	Severity string
	Category string
	Code     string
	File     string
	Line     int
	Column   int
	Message  string
	Notes    []string
}

type CompileError struct {
	Diagnostic Diagnostic
}

func (e *CompileError) Error() string {
	location := ""
	if e.Diagnostic.File != "" {
		location = e.Diagnostic.File
		if e.Diagnostic.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, e.Diagnostic.Line)
			if e.Diagnostic.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, e.Diagnostic.Column)
			}
		}
		location += ": "
	}
	return fmt.Sprintf("%s%s", location, e.Diagnostic.Message)
}

type Result struct {
	LLVMIR             []byte
	NativeImports      []string
	NativeIncludeDirs  []string
	NativeCompileFlags []string
	NativeLinkFlags    []string
	Warnings           []Diagnostic
}

func Compile(source string, opts Options) (*Result, error) {
	return compileLoadedSource(source, "", nil, opts)
}

func CompilePath(inputPath string, opts Options) (*Result, error) {
	bundle, err := loadSourceTree(inputPath, opts.TargetTriple)
	if err != nil {
		return nil, formatLoaderError(err)
	}
	result, err := compileLoadedSource(bundle.Source, inputPath, bundle.NativeSymbols, opts)
	if err != nil {
		return nil, err
	}
	result.NativeImports = append(result.NativeImports, bundle.NativeImports...)
	result.NativeIncludeDirs = append(result.NativeIncludeDirs, bundle.NativeIncludeDirs...)
	result.NativeCompileFlags = append(result.NativeCompileFlags, bundle.NativeCompileFlags...)
	result.NativeLinkFlags = append(result.NativeLinkFlags, bundle.NativeLinkFlags...)
	return result, nil
}

func formatLoaderError(err error) error {
	var diagnostic *LoaderDiagnosticError
	if errors.As(err, &diagnostic) {
		return &CompileError{Diagnostic: Diagnostic{
			Severity: "error",
			Category: "loader",
			Code:     "JY300",
			File:     diagnostic.File,
			Line:     diagnostic.Line,
			Column:   diagnostic.Column,
			Message:  diagnostic.Message,
			Notes:    diagnostic.Notes,
		}}
	}
	return fmt.Errorf("load sources: %w", err)
}

func compileLoadedSource(source string, sourcePath string, extraExterns []*ast.ExternFunctionDecl, opts Options) (*Result, error) {
	source = strings.ReplaceAll(source, "fs.watch(", "fs.watchSync(")
	l := lexer.New(source)
	p := parser.New(l)

	program, err := p.ParseProgram()
	if err != nil {
		return nil, formatParseError(sourcePath, err)
	}
	if err := resolveTypeAliases(program); err != nil {
		return nil, fmt.Errorf("type alias resolution failed: %w", err)
	}
	warnings := collectWarnings(program, sourcePath)
	program.ExternFunctions = append(extraExterns, program.ExternFunctions...)

	program, err = lowerDestructuring(program)
	if err != nil {
		return nil, fmt.Errorf("destructuring lowering failed: %w", err)
	}

	program, err = lowerAssignments(program)
	if err != nil {
		return nil, fmt.Errorf("assignment lowering failed: %w", err)
	}

	program, err = lowerFunctionExpressions(program)
	if err != nil {
		return nil, fmt.Errorf("function expression lowering failed: %w", err)
	}

	program, err = lowerGenerators(program)
	if err != nil {
		return nil, fmt.Errorf("generator lowering failed: %w", err)
	}

	if err := semantic.New().AnalyzeClasses(program); err != nil {
		return nil, formatAnalysisError(sourcePath, "class semantic analysis failed", err)
	}

	classIR := lowering.LowerClasses(program)

	program, err = lowerClasses(program)
	if err != nil {
		return nil, fmt.Errorf("class lowering failed: %w", err)
	}

	if err := semantic.New().Analyze(program); err != nil {
		return nil, formatAnalysisError(sourcePath, "semantic analysis failed", err)
	}
	if err := eraseCastExpressions(program); err != nil {
		return nil, fmt.Errorf("cast erasure failed: %w", err)
	}

	program, err = lowerAsyncFunctions(program)
	if err != nil {
		return nil, fmt.Errorf("async lowering failed: %w", err)
	}

	program, err = lowerAsyncFunctionExpressions(program)
	if err != nil {
		return nil, fmt.Errorf("async function expression lowering failed: %w", err)
	}

	lifetimeReport := lifetime.New().Analyze(program)
	warnings = append(warnings, lifetimeWarnings(lifetimeReport, sourcePath)...)
	warnings, err = applyWarningPolicy(warnings, opts.WarningPolicy, opts.AllowedWarningCategories)
	if err != nil {
		return nil, err
	}

	module, err := lowering.Lower(program)
	if err != nil {
		return nil, fmt.Errorf("lowering failed: %w", err)
	}
	module.Classes = append(module.Classes, classIR...)

	llvmIR, err := codegen.NewLLVMIRGenerator().Generate(module, opts.TargetTriple)
	if err != nil {
		return nil, fmt.Errorf("LLVM IR generation failed: %w", err)
	}

	return &Result{LLVMIR: llvmIR, Warnings: warnings}, nil
}

func lifetimeWarnings(report lifetime.Report, sourcePath string) []Diagnostic {
	if len(report.Findings) == 0 {
		return nil
	}
	warnings := make([]Diagnostic, 0, len(report.Findings))
	for _, finding := range report.Findings {
		warnings = append(warnings, Diagnostic{
			Severity: "warning",
			Category: "lifetime",
			Code:     "JY400",
			File:     sourcePath,
			Line:     finding.Line,
			Column:   finding.Column,
			Message:  finding.Message,
		})
	}
	return warnings
}

func applyWarningPolicy(warnings []Diagnostic, policy string, allowedCategories []string) ([]Diagnostic, error) {
	switch policy {
	case "", "default":
		return warnings, nil
	case "none":
		return nil, nil
	case "error":
		diagnostic, blockedCount, ok := firstBlockedWarning(warnings, allowedCategories)
		if !ok {
			return warnings, nil
		}
		diagnostic.Severity = "error"
		diagnostic.Notes = append(append([]string{}, diagnostic.Notes...), "warnings are treated as errors")
		if blockedCount > 1 {
			diagnostic.Notes = append(diagnostic.Notes, fmt.Sprintf("%d additional blocked warning(s) not shown", blockedCount-1))
		}
		return nil, &CompileError{Diagnostic: diagnostic}
	default:
		return nil, fmt.Errorf("unsupported warning policy %q; expected default, none, or error", policy)
	}
}

func firstBlockedWarning(warnings []Diagnostic, allowedCategories []string) (Diagnostic, int, bool) {
	allowed := map[string]bool{}
	for _, category := range allowedCategories {
		if category != "" {
			allowed[category] = true
		}
	}
	var first Diagnostic
	count := 0
	for _, warning := range warnings {
		if allowed[warning.Category] {
			continue
		}
		if count == 0 {
			first = warning
		}
		count++
	}
	return first, count, count > 0
}

func formatAnalysisError(sourcePath string, stage string, err error) error {
	var diagnostic *semantic.DiagnosticError
	if errors.As(err, &diagnostic) {
		out := Diagnostic{
			Severity: "error",
			Category: "semantic",
			Code:     "JY200",
			File:     sourcePath,
			Line:     diagnostic.Line,
			Column:   diagnostic.Column,
			Message:  diagnostic.Message,
		}
		if out.File == "" && out.Line == 0 {
			out.Message = fmt.Sprintf("%s: %s", stage, diagnostic.Message)
		}
		return &CompileError{Diagnostic: out}
	}
	if sourcePath != "" {
		return fmt.Errorf("%s: %s: %w", sourcePath, stage, err)
	}
	return fmt.Errorf("%s: %w", stage, err)
}

func formatParseError(sourcePath string, err error) error {
	var lexDiagnostic *lexer.DiagnosticError
	if errors.As(err, &lexDiagnostic) {
		out := Diagnostic{
			Severity: "error",
			Category: "lexer",
			Code:     "JY050",
			File:     sourcePath,
			Line:     lexDiagnostic.Line,
			Column:   lexDiagnostic.Column,
			Message:  lexDiagnostic.Message,
		}
		if out.File == "" && out.Line == 0 {
			out.Message = fmt.Sprintf("lex failed: %s", lexDiagnostic.Message)
		}
		return &CompileError{Diagnostic: out}
	}
	var diagnostic *parser.DiagnosticError
	if errors.As(err, &diagnostic) {
		out := Diagnostic{
			Severity: "error",
			Category: "parse",
			Code:     "JY100",
			File:     sourcePath,
			Line:     diagnostic.Line,
			Column:   diagnostic.Column,
			Message:  diagnostic.Message,
		}
		if out.File == "" && out.Line == 0 {
			out.Message = fmt.Sprintf("parse failed: %s", diagnostic.Message)
		}
		return &CompileError{Diagnostic: out}
	}
	if sourcePath != "" {
		return fmt.Errorf("%s: parse failed: %w", sourcePath, err)
	}
	return fmt.Errorf("parse failed: %w", err)
}

func collectWarnings(program *ast.Program, sourcePath string) []Diagnostic {
	var warnings []Diagnostic
	for _, global := range program.Globals {
		warnings = append(warnings, expressionWarnings(global.Value, sourcePath)...)
	}
	for _, fn := range program.Functions {
		warnings = append(warnings, statementWarnings(fn.Body, sourcePath)...)
	}
	for _, classDecl := range program.Classes {
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				warnings = append(warnings, expressionWarnings(member.Initializer, sourcePath)...)
			case *ast.ClassMethodDecl:
				warnings = append(warnings, statementWarnings(member.Body, sourcePath)...)
			}
		}
	}
	return warnings
}

func statementWarnings(statements []ast.Statement, sourcePath string) []Diagnostic {
	var warnings []Diagnostic
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			warnings = append(warnings, expressionWarnings(stmt.Value, sourcePath)...)
		case *ast.AssignmentStatement:
			warnings = append(warnings, expressionWarnings(stmt.Target, sourcePath)...)
			warnings = append(warnings, expressionWarnings(stmt.Value, sourcePath)...)
		case *ast.ReturnStatement:
			warnings = append(warnings, expressionWarnings(stmt.Value, sourcePath)...)
		case *ast.ExpressionStatement:
			warnings = append(warnings, expressionWarnings(stmt.Expression, sourcePath)...)
		case *ast.DeleteStatement:
			warnings = append(warnings, expressionWarnings(stmt.Target, sourcePath)...)
		case *ast.IfStatement:
			warnings = append(warnings, expressionWarnings(stmt.Condition, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Consequence, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Alternative, sourcePath)...)
		case *ast.WhileStatement:
			warnings = append(warnings, expressionWarnings(stmt.Condition, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
		case *ast.DoWhileStatement:
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
			warnings = append(warnings, expressionWarnings(stmt.Condition, sourcePath)...)
		case *ast.ForStatement:
			if stmt.Init != nil {
				warnings = append(warnings, statementWarnings([]ast.Statement{stmt.Init}, sourcePath)...)
			}
			if stmt.Condition != nil {
				warnings = append(warnings, expressionWarnings(stmt.Condition, sourcePath)...)
			}
			if stmt.Update != nil {
				warnings = append(warnings, statementWarnings([]ast.Statement{stmt.Update}, sourcePath)...)
			}
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
		case *ast.ForOfStatement:
			warnings = append(warnings, expressionWarnings(stmt.Iterable, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
		case *ast.ForInStatement:
			warnings = append(warnings, expressionWarnings(stmt.Iterable, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
		case *ast.SwitchStatement:
			warnings = append(warnings, expressionWarnings(stmt.Discriminant, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.Default, sourcePath)...)
			for _, switchCase := range stmt.Cases {
				warnings = append(warnings, expressionWarnings(switchCase.Test, sourcePath)...)
				warnings = append(warnings, statementWarnings(switchCase.Consequent, sourcePath)...)
			}
		case *ast.BlockStatement:
			warnings = append(warnings, statementWarnings(stmt.Body, sourcePath)...)
		case *ast.LabeledStatement:
			warnings = append(warnings, statementWarnings([]ast.Statement{stmt.Statement}, sourcePath)...)
		case *ast.TryStatement:
			warnings = append(warnings, statementWarnings(stmt.TryBody, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.CatchBody, sourcePath)...)
			warnings = append(warnings, statementWarnings(stmt.FinallyBody, sourcePath)...)
		case *ast.ThrowStatement:
			warnings = append(warnings, expressionWarnings(stmt.Value, sourcePath)...)
		}
	}
	return warnings
}

func expressionWarnings(expr ast.Expression, sourcePath string) []Diagnostic {
	switch expr := expr.(type) {
	case nil:
		return nil
	case *ast.CallExpression:
		var warnings []Diagnostic
		warnings = append(warnings, callDeprecationWarning(expr, sourcePath)...)
		for _, arg := range expr.Arguments {
			warnings = append(warnings, expressionWarnings(arg, sourcePath)...)
		}
		return warnings
	case *ast.InvokeExpression:
		var warnings []Diagnostic
		warnings = append(warnings, expressionWarnings(expr.Callee, sourcePath)...)
		for _, arg := range expr.Arguments {
			warnings = append(warnings, expressionWarnings(arg, sourcePath)...)
		}
		return warnings
	case *ast.NewExpression:
		var warnings []Diagnostic
		warnings = append(warnings, expressionWarnings(expr.Callee, sourcePath)...)
		for _, arg := range expr.Arguments {
			warnings = append(warnings, expressionWarnings(arg, sourcePath)...)
		}
		return warnings
	case *ast.ClosureExpression:
		return expressionWarnings(expr.Environment, sourcePath)
	case *ast.CastExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.ObjectLiteral:
		var warnings []Diagnostic
		for _, property := range expr.Properties {
			if property.Computed {
				warnings = append(warnings, expressionWarnings(property.KeyExpr, sourcePath)...)
			}
			warnings = append(warnings, expressionWarnings(property.Value, sourcePath)...)
		}
		return warnings
	case *ast.ArrayLiteral:
		var warnings []Diagnostic
		for _, element := range expr.Elements {
			warnings = append(warnings, expressionWarnings(element, sourcePath)...)
		}
		return warnings
	case *ast.TemplateLiteral:
		var warnings []Diagnostic
		for _, value := range expr.Values {
			warnings = append(warnings, expressionWarnings(value, sourcePath)...)
		}
		return warnings
	case *ast.SpreadExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.BinaryExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.ComparisonExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.LogicalExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.NullishCoalesceExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.CommaExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.ConditionalExpression:
		warnings := expressionWarnings(expr.Condition, sourcePath)
		warnings = append(warnings, expressionWarnings(expr.Consequent, sourcePath)...)
		return append(warnings, expressionWarnings(expr.Alternative, sourcePath)...)
	case *ast.UnaryExpression:
		return expressionWarnings(expr.Right, sourcePath)
	case *ast.TypeofExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.TypeCheckExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.AwaitExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.YieldExpression:
		return expressionWarnings(expr.Value, sourcePath)
	case *ast.InstanceofExpression:
		warnings := expressionWarnings(expr.Left, sourcePath)
		return append(warnings, expressionWarnings(expr.Right, sourcePath)...)
	case *ast.IndexExpression:
		warnings := expressionWarnings(expr.Target, sourcePath)
		return append(warnings, expressionWarnings(expr.Index, sourcePath)...)
	case *ast.MemberExpression:
		return expressionWarnings(expr.Target, sourcePath)
	case *ast.FunctionExpression:
		var warnings []Diagnostic
		for _, param := range expr.Params {
			warnings = append(warnings, expressionWarnings(param.Default, sourcePath)...)
		}
		warnings = append(warnings, expressionWarnings(expr.ExpressionBody, sourcePath)...)
		warnings = append(warnings, statementWarnings(expr.Body, sourcePath)...)
		return warnings
	}
	return nil
}

func callDeprecationWarning(expr *ast.CallExpression, sourcePath string) []Diagnostic {
	message := ""
	switch expr.Callee {
	case "print":
		message = "'print' is deprecated; use console.log, console.warn, or console.error instead"
	case "sleepAsync":
		message = "'sleepAsync' is a compatibility alias; use timers.sleep instead"
	case "setTimeout":
		message = "'setTimeout' is a compatibility alias; use timers.setTimeout instead"
	case "clearTimeout":
		message = "'clearTimeout' is a compatibility alias; use timers.clearTimeout instead"
	default:
		return nil
	}
	pos := ast.PositionOf(expr)
	return []Diagnostic{{
		Severity: "warning",
		Category: "deprecation",
		Code:     "JY001",
		File:     sourcePath,
		Line:     pos.Line,
		Column:   pos.Column,
		Message:  message,
	}}
}
