package llvmbackend

import (
	"runtime"

	"jayess-go/target"
)

type RelocationModel string

const (
	RelocDefault RelocationModel = "default"
	RelocPIC     RelocationModel = "pic"
	RelocStatic  RelocationModel = "static"
)

type CodeModel string

const (
	CodeModelDefault CodeModel = "default"
	CodeModelSmall   CodeModel = "small"
	CodeModelLarge   CodeModel = "large"
)

type TargetConfig struct {
	Name            string
	Triple          string
	CPU             string
	Features        []string
	RelocationModel RelocationModel
	CodeModel       CodeModel
}

func HostTargetConfig() (TargetConfig, bool) {
	spec, ok := target.LookupOSArch(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return TargetConfig{}, false
	}
	return TargetConfigFor(spec.Name)
}

func TargetConfigFor(name string) (TargetConfig, bool) {
	spec, ok := target.Lookup(name)
	if !ok {
		return TargetConfig{}, false
	}
	return TargetConfig{
		Name:            spec.Name,
		Triple:          spec.Triple,
		CPU:             "generic",
		RelocationModel: RelocDefault,
		CodeModel:       CodeModelDefault,
	}, true
}

func WithCPU(config TargetConfig, cpu string) TargetConfig {
	config.CPU = cpu
	return config
}

func WithFeatures(config TargetConfig, features ...string) TargetConfig {
	config.Features = append([]string{}, features...)
	return config
}
