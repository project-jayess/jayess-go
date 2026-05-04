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
	default:
		spec, ok := findTargetByName(name)
		if !ok {
			return "", fmt.Errorf("unsupported target %q", name)
		}
		return spec.triple, nil
	}
}

func FromOSArch(goos, goarch string) (string, error) {
	spec, ok := findTargetByOSArch(goos, goarch)
	if !ok {
		return "", fmt.Errorf("unsupported host platform %s/%s", goos, goarch)
	}
	return spec.triple, nil
}
