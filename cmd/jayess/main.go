package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"jayess-go/backend"
	"jayess-go/compiler"
	"jayess-go/target"
)

func main() {
	var emit string
	var targetName string
	var output string

	flag.StringVar(&emit, "emit", "", "output kind: llvm or exe")
	flag.StringVar(&targetName, "target", "host", "target name such as windows-x64 or darwin-arm64")
	flag.StringVar(&output, "o", "", "output file path")
	flag.Parse()

	if flag.NArg() != 1 {
		exitf("usage: jayess [--target=<name>] [--emit=llvm|exe] [-o output] <input.js>")
	}

	inputPath := flag.Arg(0)

	targetTriple, err := target.FromName(targetName)
	if err != nil {
		exitf("resolve target: %v", err)
	}

	if emit == "" {
		emit = defaultEmitMode()
	}

	if output == "" {
		output = defaultOutputPath(inputPath, emit)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		exitf("create output directory: %v", err)
	}

	opts := compiler.Options{TargetTriple: targetTriple}

	switch emit {
	case "llvm":
		result, err := compiler.CompilePath(inputPath, opts)
		if err != nil {
			exitf("compile: %v", err)
		}
		if err := os.WriteFile(output, result.LLVMIR, 0o644); err != nil {
			exitf("write output: %v", err)
		}
	case "exe":
		tc, err := backend.DetectToolchain()
		if err != nil {
			exitf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildExecutable(inputPath, opts, output); err != nil {
			exitf("build executable: %v", err)
		}
	default:
		exitf("unsupported emit mode %q", emit)
	}
}

func defaultOutputPath(inputPath, emit string) string {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	switch emit {
	case "exe":
		if runtime.GOOS == "windows" {
			return filepath.Join("build", base+".exe")
		}
		return filepath.Join("build", base)
	default:
		return filepath.Join("build", base+".ll")
	}
}

func defaultEmitMode() string {
	if runtime.GOOS == "windows" {
		return "exe"
	}
	return "llvm"
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
