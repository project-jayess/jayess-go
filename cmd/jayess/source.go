package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/lowering"
	"jayess-go/parser"
)

type loweredInput struct {
	IR      string
	Program *ast.Program
}

func lowerInputToIR(inputPath string, target llvmbackend.TargetConfig) (string, error) {
	input, err := lowerInput(inputPath, target)
	if err != nil {
		return "", err
	}
	return input.IR, nil
}

func lowerInput(inputPath string, target llvmbackend.TargetConfig) (loweredInput, error) {
	if filepath.Ext(inputPath) != ".js" {
		return loweredInput{}, fmt.Errorf("Jayess source file must use .js extension")
	}
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return loweredInput{}, fmt.Errorf("read input: %w", err)
	}
	program, err := parser.New(lexer.New(string(source))).ParseProgram()
	if err != nil {
		return loweredInput{}, fmt.Errorf("parse input: %w", err)
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:       moduleName(inputPath),
		Target:     target,
		Statements: jayessEntryStatements(program),
	})
	if err != nil {
		module = lowerFoldedReturnCodeProgram(inputPath, target, program)
	}
	return loweredInput{IR: llvmbackend.EmitLLVMIR(module), Program: program}, nil
}

func lowerFoldedReturnCodeProgram(inputPath string, target llvmbackend.TargetConfig, program *ast.Program) llvmbackend.Module {
	returnCode, _ := lowering.MainReturnCode(program)
	return llvmbackend.LowerJayessProgram(llvmbackend.JayessProgram{
		Name:       moduleName(inputPath),
		Target:     target,
		ReturnCode: returnCode,
	})
}

func jayessEntryStatements(program *ast.Program) []ast.Statement {
	if program == nil {
		return nil
	}
	for _, statement := range program.Statements {
		function, ok := statement.(*ast.FunctionDecl)
		if ok && function.Name == "main" {
			return function.Body
		}
	}
	return program.Statements
}

func moduleName(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "main"
	}
	return name
}
