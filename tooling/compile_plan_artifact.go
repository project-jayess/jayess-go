package tooling

import "jayess-go/llvmbackend"

func artifactForEmit(emit EmitKind) llvmbackend.ArtifactKind {
	switch emit {
	case EmitLLVMIR:
		return llvmbackend.LLVMIRArtifact
	case EmitBitcode:
		return llvmbackend.BitcodeArtifact
	case EmitObject:
		return llvmbackend.ObjectArtifact
	case EmitStatic:
		return llvmbackend.StaticLibArtifact
	case EmitShared:
		return llvmbackend.SharedLibArtifact
	case EmitNative:
		return llvmbackend.ExecutableArtifact
	case EmitDist:
		return llvmbackend.ExecutableArtifact
	default:
		return ""
	}
}
