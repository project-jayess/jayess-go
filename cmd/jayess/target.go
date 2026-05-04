package main

import (
	"fmt"

	"jayess-go/llvmbackend"
)

func resolveTarget(name string) (llvmbackend.TargetConfig, error) {
	if name == "" || name == "host" {
		target, ok := llvmbackend.HostTargetConfig()
		if !ok {
			return llvmbackend.TargetConfig{}, fmt.Errorf("host target is not supported")
		}
		return target, nil
	}
	target, ok := llvmbackend.TargetConfigFor(name)
	if !ok {
		return llvmbackend.TargetConfig{}, fmt.Errorf("unknown target %q", name)
	}
	return target, nil
}
