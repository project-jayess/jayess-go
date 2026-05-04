package target

type targetSpec struct {
	name   string
	goos   string
	goarch string
	triple string
}

var supportedTargets = []targetSpec{
	{name: "linux-x64", goos: "linux", goarch: "amd64", triple: "x86_64-pc-linux-gnu"},
	{name: "linux-arm64", goos: "linux", goarch: "arm64", triple: "aarch64-unknown-linux-gnu"},
	{name: "darwin-x64", goos: "darwin", goarch: "amd64", triple: "x86_64-apple-darwin"},
	{name: "darwin-arm64", goos: "darwin", goarch: "arm64", triple: "arm64-apple-darwin"},
	{name: "windows-x64", goos: "windows", goarch: "amd64", triple: "x86_64-pc-windows-msvc"},
}

func findTargetByName(name string) (targetSpec, bool) {
	for _, spec := range supportedTargets {
		if spec.name == name {
			return spec, true
		}
	}
	return targetSpec{}, false
}

func findTargetByOSArch(goos, goarch string) (targetSpec, bool) {
	for _, spec := range supportedTargets {
		if spec.goos == goos && spec.goarch == goarch {
			return spec, true
		}
	}
	return targetSpec{}, false
}
