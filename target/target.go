package target

import (
	"fmt"
	"runtime"
)

func DefaultTriple() (string, error) {
	return FromOSArch(runtime.GOOS, runtime.GOARCH)
}

func FromName(name string) (string, error) {
	switch name {
	case "", "host":
		return DefaultTriple()
	case "linux-x64":
		return "x86_64-pc-linux-gnu", nil
	case "linux-arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "darwin-x64":
		return "x86_64-apple-darwin", nil
	case "darwin-arm64":
		return "arm64-apple-darwin", nil
	case "windows-x64":
		return "x86_64-pc-windows-msvc", nil
	default:
		return "", fmt.Errorf("unsupported target %q", name)
	}
}

func FromOSArch(goos, goarch string) (string, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "x86_64-pc-linux-gnu", nil
	case goos == "linux" && goarch == "arm64":
		return "aarch64-unknown-linux-gnu", nil
	case goos == "darwin" && goarch == "amd64":
		return "x86_64-apple-darwin", nil
	case goos == "darwin" && goarch == "arm64":
		return "arm64-apple-darwin", nil
	case goos == "windows" && goarch == "amd64":
		return "x86_64-pc-windows-msvc", nil
	default:
		return "", fmt.Errorf("unsupported host platform %s/%s", goos, goarch)
	}
}
