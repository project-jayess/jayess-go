package llvmbackend

func SharedLibraryNameForTarget(target TargetConfig, base string) string {
	return SharedLibraryName(targetPlatform(target), base)
}
