package llvmbackend

type DebugInfoKind string

const (
	DebugInfoNone  DebugInfoKind = "none"
	DebugInfoDWARF DebugInfoKind = "dwarf"
)

type DebugConfig struct {
	Kind                    DebugInfoKind
	PreserveSourceLocations bool
	PreserveFunctionNames   bool
	CrashMapping            bool
}

func DefaultDebugConfig(enabled bool) DebugConfig {
	if !enabled {
		return DebugConfig{Kind: DebugInfoNone}
	}
	return DebugConfig{
		Kind:                    DebugInfoDWARF,
		PreserveSourceLocations: true,
		PreserveFunctionNames:   true,
		CrashMapping:            true,
	}
}
