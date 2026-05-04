package main

import (
	"fmt"
	"os"

	"jayess-go/lldc"
	"jayess-go/llvmbackend"
	"jayess-go/llvmc"
	"jayess-go/tooling"
)

const sharedLibraryWorkDir = "temp/jayess-build"

func compileSharedLibrary(ir string, plan tooling.CompilePlan) error {
	shared := plan.SharedLibrary
	if !plan.CanBuild() {
		return fmt.Errorf("shared-library plan is not buildable: %v", plan.Diagnostics)
	}
	internalObject := llvmc.Available()
	internalLink := internalObject && lldc.Available()
	var commands []llvmbackend.ToolchainCommand
	if !internalLink {
		var err error
		commands, err = resolveToolchainCommands(sharedLibraryCommands(shared), shared.Target.Name)
		if err != nil {
			return err
		}
	}
	irPath := llvmbackend.SharedLibraryIRPath(shared, sharedLibraryWorkDir)
	if err := writeFile(irPath, []byte(ir)); err != nil {
		return err
	}
	if internalObject {
		objectPath := llvmbackend.SharedLibraryObjectPath(shared, sharedLibraryWorkDir)
		if err := llvmc.EmitObject(llvmc.ObjectRequest{IR: ir, TargetTriple: shared.Target.Triple, OutputPath: objectPath}); err != nil {
			return err
		}
	}
	if err := ensureParentDir(shared.OutputPath); err != nil {
		return err
	}
	if internalLink {
		return linkSharedLibraryInternally(shared)
	}
	for _, command := range commands {
		if err := runToolchainCommand(command); err != nil {
			return err
		}
	}
	if _, err := os.Stat(shared.OutputPath); err != nil {
		return fmt.Errorf("shared library output was not produced: %w", err)
	}
	return nil
}

func linkSharedLibraryInternally(shared llvmbackend.SharedLibraryBuildPlan) error {
	objectPath := llvmbackend.SharedLibraryObjectPath(shared, sharedLibraryWorkDir)
	if err := lldc.Link(lldc.LinkRequest{
		ObjectPath:       objectPath,
		ExtraObjectFiles: shared.ExtraObjectFiles,
		OutputPath:       shared.OutputPath,
		TargetTriple:     shared.Target.Triple,
		Shared:           true,
		LinkFlags:        shared.LinkFlags,
	}); err != nil {
		return err
	}
	if _, err := os.Stat(shared.OutputPath); err != nil {
		return fmt.Errorf("shared library output was not produced: %w", err)
	}
	return nil
}

func sharedLibraryCommands(shared llvmbackend.SharedLibraryBuildPlan) []llvmbackend.ToolchainCommand {
	if llvmc.Available() {
		return []llvmbackend.ToolchainCommand{llvmbackend.SharedLibraryLinkCommand(shared, sharedLibraryWorkDir)}
	}
	return llvmbackend.SharedLibraryToolchainCommands(shared, sharedLibraryWorkDir)
}
