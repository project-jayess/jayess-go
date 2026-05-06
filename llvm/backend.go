package llvm

import (
	"fmt"

	"jayess-go/lldc"
	"jayess-go/llvmc"
)

type ObjectEmissionRequest struct {
	Module       *Module
	TargetTriple string
	OutputPath   string
}

type LinkRequest struct {
	ObjectPath       string
	ExtraObjectFiles []string
	OutputPath       string
	TargetTriple     string
	Shared           bool
	LinkFlags        []string
}

func ObjectEmitterAvailable() bool {
	return llvmc.Available()
}

func LinkerAvailable() bool {
	return lldc.Available()
}

func EmitObject(request ObjectEmissionRequest) error {
	if request.Module == nil {
		return fmt.Errorf("LLVM object emission requires a module")
	}
	return llvmc.EmitObject(llvmc.ObjectRequest{
		IR:           request.Module.String(),
		TargetTriple: request.TargetTriple,
		OutputPath:   request.OutputPath,
	})
}

func Link(request LinkRequest) error {
	return lldc.Link(lldc.LinkRequest{
		ObjectPath:       request.ObjectPath,
		ExtraObjectFiles: append([]string(nil), request.ExtraObjectFiles...),
		OutputPath:       request.OutputPath,
		TargetTriple:     request.TargetTriple,
		Shared:           request.Shared,
		LinkFlags:        append([]string(nil), request.LinkFlags...),
	})
}
