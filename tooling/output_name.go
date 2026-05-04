package tooling

import "jayess-go/llvmbackend"

func DefaultOutputName(emit EmitKind, target llvmbackend.TargetConfig, base string) string {
	switch emit {
	case EmitShared:
		return llvmbackend.SharedLibraryNameForTarget(target, base)
	default:
		return base
	}
}
