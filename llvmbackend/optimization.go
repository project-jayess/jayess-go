package llvmbackend

type OptLevel string

const (
	OptO0 OptLevel = "O0"
	OptO1 OptLevel = "O1"
	OptO2 OptLevel = "O2"
	OptO3 OptLevel = "O3"
	OptOz OptLevel = "Oz"
)

type OptimizationPipeline struct {
	Level         OptLevel
	DebugFriendly bool
	VerifyIR      bool
	Passes        []string
}

func OptimizationPipelineFor(level OptLevel) OptimizationPipeline {
	if level == OptO0 {
		return OptimizationPipeline{
			Level:         OptO0,
			DebugFriendly: true,
			VerifyIR:      true,
			Passes:        []string{"verify"},
		}
	}
	return OptimizationPipeline{
		Level:    level,
		VerifyIR: true,
		Passes:   []string{"verify", "mem2reg", "instcombine", "simplifycfg"},
	}
}

func OptLevels() []OptLevel {
	return []OptLevel{OptO0, OptO1, OptO2, OptO3, OptOz}
}
