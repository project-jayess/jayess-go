package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"jayess-go/tooling"
)

type stringListFlag []string

func (flagValue *stringListFlag) String() string {
	if flagValue == nil {
		return ""
	}
	return strings.Join(*flagValue, ",")
}

func (flagValue *stringListFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*flagValue = append(*flagValue, part)
		}
	}
	return nil
}

func parseCLI(args []string) (cliConfig, error) {
	cfg := defaultCLIConfig()
	if len(args) > 0 && args[0] == "package" {
		cfg.emit = tooling.EmitDist
		args = args[1:]
	} else if len(args) > 0 && args[0] == "compile" {
		args = args[1:]
	}

	var emit string
	features := stringListFlag{}
	flags := flag.NewFlagSet("jayess", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&emit, "emit", string(cfg.emit), "output kind: llvm, bc, obj, lib, shared, exe, or dist")
	flags.StringVar(&cfg.targetName, "target", "host", "target name such as linux-x64, macos-arm64, windows-x64, or host")
	flags.StringVar(&cfg.outputPath, "o", "", "output file path")
	flags.StringVar(&cfg.outputPath, "output", "", "output file path")
	flags.StringVar(&cfg.executablePath, "executable", "", "compiled executable path for --emit=dist")
	flags.StringVar(&cfg.optimizationLevel, "opt", "", "optimization level: O0, O1, O2, O3, or Oz")
	flags.StringVar(&cfg.targetCPU, "cpu", "", "target CPU name")
	flags.Var(&features, "feature", "target feature; repeatable or comma-separated")
	flags.StringVar(&cfg.relocationModel, "reloc", "", "relocation model: pic, pie, or static")
	flags.StringVar(&cfg.codeModel, "code-model", "", "code model: small, medium, large, or kernel")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, fmt.Errorf("usage: %s", usage())
	}

	cfg.emit = tooling.EmitKind(emit)
	if !tooling.HasEmitKind(cfg.emit) {
		return cliConfig{}, fmt.Errorf("unsupported emit kind %q", emit)
	}
	cfg.targetFeatures = append([]string{}, features...)

	remaining := flags.Args()
	if len(remaining) != 1 {
		return cliConfig{}, fmt.Errorf("usage: %s", usage())
	}
	cfg.inputPath = remaining[0]
	return cfg, nil
}

func usage() string {
	return "jayess [compile|package] [--target=<name>] [--emit=llvm|bc|obj|lib|shared|exe|dist] [--executable path] [-o output] <input.js>"
}
