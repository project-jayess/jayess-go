package dist

import (
	"fmt"
	"os"
	"path/filepath"
)

type Result struct {
	Plan           Plan
	CopiedTools    []string
	CopiedLibs     []string
	CopiedLicenses []string
	ArchivePath    string
	ChecksumPath   string
	Diagnostics    []string
	CompilerBuilt  bool
}

func Create(config Config) (Result, error) {
	normalized, platform, err := NormalizeConfig(config)
	if err != nil {
		return Result{}, err
	}
	plan, err := BuildPlan(normalized)
	if err != nil {
		return Result{}, err
	}
	result := Result{Plan: plan}
	if err := os.RemoveAll(plan.Root); err != nil {
		return result, err
	}
	if err := os.MkdirAll(plan.Root, 0o755); err != nil {
		return result, err
	}
	if normalized.BuildCompiler {
		if err := buildCompiler(normalized, platform, plan.CompilerPath); err != nil {
			return result, err
		}
		result.CompilerBuilt = true
	}
	tools, diagnostics, err := copyLLVMTools(normalized.LLVMBuildDir, plan.ToolBinDir, normalized.Tools)
	if err != nil {
		return result, err
	}
	result.CopiedTools = tools
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	if normalized.StrictTools && len(diagnostics) > 0 {
		return result, fmt.Errorf("required bundled tools are missing")
	}
	libs, err := copyLLVMLibraries(normalized.LLVMBuildDir, plan.ToolLibDir)
	if err != nil {
		return result, err
	}
	result.CopiedLibs = libs
	licenses, licenseDiagnostics, err := copyLicenses(normalized.SourceRoot, plan.LicenseDir)
	if err != nil {
		return result, err
	}
	result.CopiedLicenses = licenses
	result.Diagnostics = append(result.Diagnostics, licenseDiagnostics...)
	if err := writeManifest(normalized, result); err != nil {
		return result, err
	}
	if normalized.Archive {
		archivePath, checksumPath, err := writeArchive(plan)
		if err != nil {
			return result, fmt.Errorf("archive distribution: %w", err)
		}
		result.ArchivePath = archivePath
		result.ChecksumPath = checksumPath
	}
	return result, nil
}

func llvmBuildBinDir(buildDir string) string {
	return filepath.Join(buildDir, "bin")
}

func llvmBuildLibDir(buildDir string) string {
	return filepath.Join(buildDir, "lib")
}
