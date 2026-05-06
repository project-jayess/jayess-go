package dist

import (
	"fmt"
	"path/filepath"
	"runtime"

	"jayess-go/target"
)

const (
	DefaultOutDir     = "dist"
	DefaultSourceRoot = "."
	DefaultVersion    = "dev"
)

type Config struct {
	Platform      string
	Version       string
	OutDir        string
	SourceRoot    string
	LLVMBuildDir  string
	Archive       bool
	BuildCompiler bool
	StrictTools   bool
	GoTags        []string
	Tools         []string
}

type Platform struct {
	Name   string
	GOOS   string
	GOARCH string
}

func NormalizeConfig(config Config) (Config, Platform, error) {
	if config.Platform == "" {
		spec, ok := target.LookupOSArch(runtime.GOOS, runtime.GOARCH)
		if !ok {
			return Config{}, Platform{}, fmt.Errorf("host platform %s/%s is not supported", runtime.GOOS, runtime.GOARCH)
		}
		config.Platform = spec.Name
	}
	spec, ok := target.Lookup(config.Platform)
	if !ok {
		return Config{}, Platform{}, fmt.Errorf("unsupported distribution platform %q", config.Platform)
	}
	if config.Version == "" {
		config.Version = DefaultVersion
	}
	if config.OutDir == "" {
		config.OutDir = DefaultOutDir
	}
	if config.SourceRoot == "" {
		config.SourceRoot = DefaultSourceRoot
	}
	if config.LLVMBuildDir == "" {
		config.LLVMBuildDir = defaultLLVMBuildDir(config.SourceRoot)
	}
	if len(config.Tools) == 0 {
		config.Tools = DefaultTools()
	}
	return config, Platform{Name: spec.Name, GOOS: spec.GOOS, GOARCH: spec.GOARCH}, nil
}

func DefaultTools() []string {
	return []string{"clang", "clang++", "lld", "ld.lld", "llvm-as", "llc"}
}

func defaultLLVMBuildDir(sourceRoot string) string {
	return filepath.Join(sourceRoot, "refs", "llvm-project", "build")
}
