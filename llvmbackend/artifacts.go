package llvmbackend

type ArtifactKind string

const (
	LLVMIRArtifact     ArtifactKind = "llvm-ir"
	BitcodeArtifact    ArtifactKind = "bitcode"
	ObjectArtifact     ArtifactKind = "object"
	ExecutableArtifact ArtifactKind = "executable"
	StaticLibArtifact  ArtifactKind = "static-library"
	SharedLibArtifact  ArtifactKind = "shared-library"
)

func ArtifactKinds() []ArtifactKind {
	return []ArtifactKind{
		LLVMIRArtifact,
		BitcodeArtifact,
		ObjectArtifact,
		ExecutableArtifact,
		StaticLibArtifact,
		SharedLibArtifact,
	}
}

func SupportsArtifact(kind ArtifactKind) bool {
	for _, artifact := range ArtifactKinds() {
		if artifact == kind {
			return true
		}
	}
	return false
}

func SharedLibraryName(platform string, base string) string {
	switch platform {
	case "linux":
		return "lib" + base + ".so"
	case "darwin":
		return "lib" + base + ".dylib"
	case "windows":
		return base + ".dll"
	default:
		return "lib" + base + ".so"
	}
}
