package llvmbackend

type PlatformSupport struct {
	TargetName            string
	Executable            bool
	CrossObject           bool
	ObjectLibraryEmission bool
	Diagnostic            string
}

func PlatformSupports() []PlatformSupport {
	return []PlatformSupport{
		{TargetName: "linux-x64", Executable: true, CrossObject: true, ObjectLibraryEmission: true},
		{TargetName: "linux-arm64", Executable: true, CrossObject: true, ObjectLibraryEmission: true},
		{TargetName: "macos-x64", Executable: true, CrossObject: true, ObjectLibraryEmission: true, Diagnostic: "Apple SDK/sysroot is required for executable linking"},
		{TargetName: "macos-arm64", Executable: true, CrossObject: true, ObjectLibraryEmission: true, Diagnostic: "Apple SDK/sysroot is required for executable linking"},
		{TargetName: "windows-x64", Executable: true, CrossObject: true, ObjectLibraryEmission: true, Diagnostic: "Windows SDK and C runtime are required for executable linking"},
	}
}

func PlatformSupportFor(targetName string) (PlatformSupport, bool) {
	for _, support := range PlatformSupports() {
		if support.TargetName == targetName {
			return support, true
		}
	}
	return PlatformSupport{
		TargetName: targetName,
		Diagnostic: "LLVM target support is not configured",
	}, false
}
