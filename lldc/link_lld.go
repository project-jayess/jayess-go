//go:build jayess_lld && cgo

package lldc

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../refs/llvm-project/lld/include -I${SRCDIR}/../refs/llvm-project/llvm/include -I${SRCDIR}/../refs/llvm-project/build/include
#cgo LDFLAGS: -L${SRCDIR}/../refs/llvm-project/build/lib -llldELF -llldCOFF -llldMachO -llldCommon
#include <stdlib.h>
#include "lld_shim.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"
)

func Available() bool {
	return true
}

func BackendName() string {
	return "lld-cpp-shim"
}

func Link(request LinkRequest) error {
	args, free := lldArgs(request)
	defer free()
	var message *C.char
	ret := C.jayess_lld_link((**C.char)(unsafe.Pointer(&args[0])), C.int(len(args)), &message)
	if message != nil {
		defer C.jayess_lld_free_message(message)
	}
	if ret != 0 {
		detail := C.GoString(message)
		if detail == "" {
			return fmt.Errorf("internal lld link failed with exit code %d", int(ret))
		}
		return fmt.Errorf("internal lld link failed with exit code %d: %s", int(ret), detail)
	}
	return nil
}

func lldArgs(request LinkRequest) ([]*C.char, func()) {
	values := lldArgValues(request)
	args := make([]*C.char, 0, len(values))
	for _, value := range values {
		args = append(args, C.CString(value))
	}
	return args, func() {
		for _, arg := range args {
			C.free(unsafe.Pointer(arg))
		}
	}
}

func lldArgValues(request LinkRequest) []string {
	switch targetFlavor(request.TargetTriple) {
	case "darwin":
		return machoArgs(request)
	case "windows":
		return coffArgs(request)
	default:
		return elfArgs(request)
	}
}

func elfArgs(request LinkRequest) []string {
	args := []string{"ld.lld", "-shared", request.ObjectPath, "-o", request.OutputPath}
	args = append(args, request.ExtraObjectFiles...)
	args = append(args, request.LinkFlags...)
	return args
}

func machoArgs(request LinkRequest) []string {
	args := []string{"ld64.lld", "-dylib", request.ObjectPath, "-o", request.OutputPath}
	args = append(args, request.ExtraObjectFiles...)
	args = append(args, request.LinkFlags...)
	return args
}

func coffArgs(request LinkRequest) []string {
	args := []string{"lld-link", "/dll", request.ObjectPath, "/out:" + request.OutputPath}
	args = append(args, request.ExtraObjectFiles...)
	args = append(args, request.LinkFlags...)
	return args
}

func targetFlavor(triple string) string {
	if strings.Contains(triple, "apple-darwin") {
		return "darwin"
	}
	if strings.Contains(triple, "windows") || strings.Contains(triple, "msvc") || strings.Contains(triple, "mingw") {
		return "windows"
	}
	if runtime.GOOS == "darwin" && triple == "" {
		return "darwin"
	}
	if runtime.GOOS == "windows" && triple == "" {
		return "windows"
	}
	return "elf"
}
