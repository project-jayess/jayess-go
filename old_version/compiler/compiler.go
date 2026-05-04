package compiler

import (
	"fmt"
	"strings"

	"jayess-go/ast"
	"jayess-go/codegen"
	"jayess-go/ir"
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
	LifetimeReport     lifetime.Report
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

	lifetimeReport := sanitizeLifetimeReport(lifetime.New().Analyze(program))
	warnings = append(warnings, lifetimeWarnings(lifetimeReport, sourcePath)...)
	warnings, err = applyWarningPolicy(warnings, opts.WarningPolicy, opts.AllowedWarningCategories)
	if err != nil {
		return nil, err
	}

	module, err := lowering.Lower(program)
	if err != nil {
		return nil, fmt.Errorf("lowering failed: %w", err)
	}
	module.SourcePath = sourcePath
	module.Classes = append(module.Classes, classIR...)
	module.LifetimeEligible = mapLifetimeEligibleLocals(lifetimeReport)
	module.EligibleParams = mapLifetimeEligibleParams(lifetimeReport)
	applyEligibleParams(module)

	llvmIR, err := codegen.NewLLVMIRGenerator().Generate(module, opts.TargetTriple)
	if err != nil {
		return nil, fmt.Errorf("LLVM IR generation failed: %w", err)
	}

	return &Result{LLVMIR: llvmIR, Warnings: warnings, LifetimeReport: lifetimeReport}, nil
}

func mapLifetimeEligibleLocals(report lifetime.Report) []ir.LocalLifetimeClassification {
	out := make([]ir.LocalLifetimeClassification, 0, len(report.Eligible))
	for _, item := range report.Eligible {
		out = append(out, ir.LocalLifetimeClassification{
			Function: item.Function,
			Name:     item.Name,
			Line:     item.Line,
			Column:   item.Column,
			Kind:     ir.DeclarationKind(item.Kind),
			InLoop:   item.InLoop,
		})
	}
	return out
}

func mapLifetimeEligibleParams(report lifetime.Report) []ir.ParameterLifetimeClassification {
	out := make([]ir.ParameterLifetimeClassification, 0, len(report.EligibleParams))
	for _, item := range report.EligibleParams {
		out = append(out, ir.ParameterLifetimeClassification{
			Function: item.Function,
			Name:     item.Name,
		})
	}
	return out
}

func applyEligibleParams(module *ir.Module) {
	if module == nil || len(module.EligibleParams) == 0 {
		return
	}
	eligible := map[string]map[string]bool{}
	for _, item := range module.EligibleParams {
		names := eligible[item.Function]
		if names == nil {
			names = map[string]bool{}
			eligible[item.Function] = names
		}
		names[item.Name] = true
	}
	for fi := range module.Functions {
		names := eligible[module.Functions[fi].Name]
		if len(names) == 0 {
			continue
		}
		for pi := range module.Functions[fi].Params {
			if names[module.Functions[fi].Params[pi].Name] {
				module.Functions[fi].Params[pi].CleanupEligible = true
			}
		}
	}
}
