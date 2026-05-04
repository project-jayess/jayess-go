package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*f = append(*f, part)
		}
	}
	return nil
}

type cliConfig struct {
	mode                     string
	emit                     string
	optimizationLevel        string
	targetCPU                string
	targetFeatures           stringListFlag
	relocationModel          string
	codeModel                string
	targetName               string
	output                   string
	warningPolicy            string
	allowedWarningCategories stringListFlag
	inputPath                string
	initPath                 string
	programArgs              []string
}

func parseCLI(args []string) (cliConfig, error) {
	cfg := cliConfig{
		mode:          "compile",
		targetName:    "host",
		warningPolicy: "default",
	}
	if len(args) > 0 && (args[0] == "compile" || args[0] == "run" || args[0] == "init" || args[0] == "test") {
		cfg.mode = args[0]
		args = args[1:]
	}
	if cfg.mode == "init" {
		if len(args) > 1 {
			return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
		}
		if len(args) == 1 {
			cfg.initPath = args[0]
		}
		return cfg, nil
	}

	flags := flag.NewFlagSet("jayess", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.emit, "emit", "", "output kind: llvm, obj, or exe")
	flags.StringVar(&cfg.optimizationLevel, "opt", "", "optimization level: O0, O1, O2, O3, or Oz")
	flags.StringVar(&cfg.targetCPU, "cpu", "", "target CPU name passed to clang")
	flags.Var(&cfg.targetFeatures, "feature", "target feature passed to clang; repeatable or comma-separated, such as +sse2 or -avx")
	flags.StringVar(&cfg.relocationModel, "reloc", "", "relocation model: pic, pie, or static")
	flags.StringVar(&cfg.codeModel, "code-model", "", "code model: small, medium, large, or kernel")
	flags.StringVar(&cfg.targetName, "target", "host", "target name such as windows-x64 or darwin-arm64")
	flags.StringVar(&cfg.output, "o", "", "output file path")
	flags.StringVar(&cfg.warningPolicy, "warnings", "default", "warning policy: default, none, or error")
	flags.Var(&cfg.allowedWarningCategories, "allow-warning", "warning category to allow when --warnings=error; repeatable or comma-separated")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	remaining := flags.Args()
	if cfg.mode == "test" && len(remaining) == 0 {
		cfg.inputPath = "."
		return cfg, nil
	}
	if len(remaining) == 0 {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	cfg.inputPath = remaining[0]
	if cfg.mode == "run" {
		cfg.programArgs = remaining[1:]
	} else if len(remaining) != 1 {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	return cfg, nil
}

func usageForMode(mode string) string {
	switch mode {
	case "init":
		return "jayess init [directory]"
	case "run":
		return "jayess run [--target=<name>] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [-o output] <input.js> [args...]"
	case "test":
		return "jayess test [--target=<name>] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [path|file.test.js]"
	default:
		return "jayess [--target=<name>] [--emit=llvm|bc|obj|lib|shared|exe] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [-o output] <input.js>"
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
	case "obj":
		return filepath.Join("build", base+".o")
	case "lib":
		return filepath.Join("build", "lib"+base+".a")
	case "shared":
		switch runtime.GOOS {
		case "windows":
			return filepath.Join("build", base+".dll")
		case "darwin":
			return filepath.Join("build", "lib"+base+".dylib")
		default:
			return filepath.Join("build", "lib"+base+".so")
		}
	case "bc":
		return filepath.Join("build", base+".bc")
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

func isSupportedOptimizationLevel(level string) bool {
	switch level {
	case "O0", "O1", "O2", "O3", "Oz":
		return true
	default:
		return false
	}
}

func isSupportedRelocationModel(model string) bool {
	switch model {
	case "pic", "pie", "static":
		return true
	default:
		return false
	}
}

func isSupportedCodeModel(model string) bool {
	switch model {
	case "small", "medium", "large", "kernel":
		return true
	default:
		return false
	}
}
