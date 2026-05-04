//go:build jayess_llvmc && cgo

package llvmc

/*
#cgo CFLAGS: -I${SRCDIR}/../refs/llvm-project/build/include -I${SRCDIR}/../refs/llvm-project/llvm/include
#cgo LDFLAGS: -L${SRCDIR}/../refs/llvm-project/build/lib -Wl,-rpath,${SRCDIR}/../refs/llvm-project/build/lib -lLLVM
#include <stdlib.h>
#include <llvm-c/Core.h>
#include <llvm-c/Target.h>
#include <llvm-c/TargetMachine.h>
#include <llvm-c/IRReader.h>

static void jayessLLVMInitializeTargets(void) {
	LLVMInitializeAllTargetInfos();
	LLVMInitializeAllTargets();
	LLVMInitializeAllTargetMCs();
	LLVMInitializeAllAsmParsers();
	LLVMInitializeAllAsmPrinters();
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func Available() bool {
	return true
}

func BackendName() string {
	return "llvm-c-api"
}

func EmitObject(request ObjectRequest) error {
	if request.IR == "" {
		return fmt.Errorf("missing LLVM IR input")
	}
	if request.TargetTriple == "" {
		return fmt.Errorf("missing LLVM target triple")
	}
	if request.OutputPath == "" {
		return fmt.Errorf("missing object output path")
	}

	C.jayessLLVMInitializeTargets()
	context := C.LLVMContextCreate()
	defer C.LLVMContextDispose(context)

	cIR := C.CString(request.IR)
	defer C.free(unsafe.Pointer(cIR))
	bufferName := C.CString("jayess-module")
	defer C.free(unsafe.Pointer(bufferName))
	buffer := C.LLVMCreateMemoryBufferWithMemoryRangeCopy(cIR, C.size_t(len(request.IR)), bufferName)

	var module C.LLVMModuleRef
	var message *C.char
	if C.LLVMParseIRInContext(context, buffer, &module, &message) != 0 {
		defer C.LLVMDisposeMessage(message)
		return fmt.Errorf("parse LLVM IR: %s", C.GoString(message))
	}
	defer C.LLVMDisposeModule(module)

	triple := C.CString(request.TargetTriple)
	defer C.free(unsafe.Pointer(triple))
	C.LLVMSetTarget(module, triple)

	var target C.LLVMTargetRef
	if C.LLVMGetTargetFromTriple(triple, &target, &message) != 0 {
		defer C.LLVMDisposeMessage(message)
		return fmt.Errorf("resolve LLVM target: %s", C.GoString(message))
	}
	cpu := C.CString("generic")
	defer C.free(unsafe.Pointer(cpu))
	features := C.CString("")
	defer C.free(unsafe.Pointer(features))
	targetMachine := C.LLVMCreateTargetMachine(target, triple, cpu, features, C.LLVMCodeGenLevelDefault, C.LLVMRelocDefault, C.LLVMCodeModelDefault)
	if targetMachine == nil {
		return fmt.Errorf("create LLVM target machine")
	}
	defer C.LLVMDisposeTargetMachine(targetMachine)

	output := C.CString(request.OutputPath)
	defer C.free(unsafe.Pointer(output))
	if C.LLVMTargetMachineEmitToFile(targetMachine, module, output, C.LLVMObjectFile, &message) != 0 {
		defer C.LLVMDisposeMessage(message)
		return fmt.Errorf("emit object file: %s", C.GoString(message))
	}
	return nil
}
