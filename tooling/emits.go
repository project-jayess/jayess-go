package tooling

type EmitKind string

const (
	EmitLLVMIR  EmitKind = "llvm"
	EmitBitcode EmitKind = "bc"
	EmitObject  EmitKind = "obj"
	EmitStatic  EmitKind = "lib"
	EmitShared  EmitKind = "shared"
	EmitNative  EmitKind = "exe"
	EmitDist    EmitKind = "dist"
)

func EmitKinds() []EmitKind {
	return []EmitKind{EmitLLVMIR, EmitBitcode, EmitObject, EmitStatic, EmitShared, EmitNative, EmitDist}
}

func HasEmitKind(kind EmitKind) bool {
	for _, emitKind := range EmitKinds() {
		if emitKind == kind {
			return true
		}
	}
	return false
}
