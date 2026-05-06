package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"jayess-go/dist"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("jayess-dist", flag.ContinueOnError)
	platform := flags.String("platform", "", "target platform such as linux-x64")
	version := flags.String("version", dist.DefaultVersion, "package version")
	out := flags.String("out", dist.DefaultOutDir, "distribution output directory")
	sourceRoot := flags.String("source-root", dist.DefaultSourceRoot, "Jayess repository root")
	llvmBuildDir := flags.String("llvm-build-dir", "", "LLVM build directory containing bin and lib")
	archive := flags.Bool("archive", true, "write a compressed archive")
	buildCompiler := flags.Bool("build-compiler", true, "build the Jayess compiler into the package")
	strictTools := flags.Bool("strict-tools", true, "fail when requested LLVM tools are missing")
	tags := flags.String("tags", "jayess_llvmc jayess_lld", "Go build tags for the compiler")
	tools := flags.String("tools", strings.Join(dist.DefaultTools(), ","), "comma-separated LLVM tools to copy")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := dist.Create(dist.Config{
		Platform:      *platform,
		Version:       *version,
		OutDir:        *out,
		SourceRoot:    *sourceRoot,
		LLVMBuildDir:  *llvmBuildDir,
		Archive:       *archive,
		BuildCompiler: *buildCompiler,
		StrictTools:   *strictTools,
		GoTags:        splitList(*tags),
		Tools:         splitList(*tools),
	})
	if err != nil {
		return err
	}
	fmt.Println("dist:", result.Plan.Root)
	if result.ArchivePath != "" {
		fmt.Println("archive:", result.ArchivePath)
		fmt.Println("sha256:", result.ChecksumPath)
	}
	for _, diagnostic := range result.Diagnostics {
		fmt.Println("warning:", diagnostic)
	}
	return nil
}

func splitList(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		for _, field := range strings.Fields(part) {
			if field != "" {
				result = append(result, field)
			}
		}
	}
	return result
}
