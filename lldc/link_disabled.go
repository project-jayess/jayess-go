//go:build !jayess_lld || !cgo

package lldc

import "fmt"

func Available() bool {
	return false
}

func BackendName() string {
	return "external-clang"
}

func Link(request LinkRequest) error {
	return fmt.Errorf("internal lld linker is not enabled in this build")
}
