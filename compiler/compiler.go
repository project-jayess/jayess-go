package compiler

import (
	"fmt"

	"jayess-go/codegen"
	"jayess-go/lexer"
	"jayess-go/lifetime"
	"jayess-go/lowering"
	"jayess-go/parser"
	"jayess-go/semantic"
)

type Options struct {
	TargetTriple string
}

type Result struct {
	LLVMIR        []byte
	NativeImports []string
}

func Compile(source string, opts Options) (*Result, error) {
	return compileLoadedSource(source, opts)
}

func CompilePath(inputPath string, opts Options) (*Result, error) {
	bundle, err := loadSourceTree(inputPath)
	if err != nil {
		return nil, fmt.Errorf("load sources: %w", err)
	}
	result, err := compileLoadedSource(bundle.Source, opts)
	if err != nil {
		return nil, err
	}
	result.NativeImports = append(result.NativeImports, bundle.NativeImports...)
	return result, nil
}

func compileLoadedSource(source string, opts Options) (*Result, error) {
	rewrittenSource, err := transpileClasses(source)
	if err != nil {
		return nil, fmt.Errorf("class transpilation failed: %w", err)
	}

	l := lexer.New(rewrittenSource)
	p := parser.New(l)

	program, err := p.ParseProgram()
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	if err := semantic.New().Analyze(program); err != nil {
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	_ = lifetime.New().Analyze(program)

	module, err := lowering.Lower(program)
	if err != nil {
		return nil, fmt.Errorf("lowering failed: %w", err)
	}

	llvmIR, err := codegen.NewLLVMIRGenerator().Generate(module, opts.TargetTriple)
	if err != nil {
		return nil, fmt.Errorf("LLVM IR generation failed: %w", err)
	}

	return &Result{LLVMIR: llvmIR}, nil
}
