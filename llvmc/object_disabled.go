//go:build !jayess_llvmc || !cgo

package llvmc

import "fmt"

func Available() bool {
	return false
}

func BackendName() string {
	return "external-toolchain"
}

func EmitObject(request ObjectRequest) error {
	return fmt.Errorf("LLVM C API object emitter is not enabled in this build")
}
