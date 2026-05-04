package main

import "jayess-go/tooling"

type cliConfig struct {
	mode              string
	emit              tooling.EmitKind
	targetName        string
	outputPath        string
	executablePath    string
	inputPath         string
	optimizationLevel string
	targetCPU         string
	targetFeatures    []string
	relocationModel   string
	codeModel         string
}

func defaultCLIConfig() cliConfig {
	return cliConfig{
		mode:       "compile",
		emit:       tooling.EmitLLVMIR,
		targetName: "host",
	}
}
