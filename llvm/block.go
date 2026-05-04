package llvm

import (
	"fmt"
	"strings"
)

type BasicBlock struct {
	name         string
	instructions []string
	terminated   bool
}

func (block *BasicBlock) Return(value Constant) error {
	if block.terminated {
		return fmt.Errorf("LLVM basic block %q is already terminated", block.name)
	}
	if value.Type().ir == "" {
		return fmt.Errorf("LLVM return value must have a type")
	}
	block.instructions = append(block.instructions, "ret "+value.Type().String()+" "+value.String())
	block.terminated = true
	return nil
}

func (block *BasicBlock) ReturnVoid() error {
	if block.terminated {
		return fmt.Errorf("LLVM basic block %q is already terminated", block.name)
	}
	block.instructions = append(block.instructions, "ret void")
	block.terminated = true
	return nil
}

func (block BasicBlock) String() string {
	var builder strings.Builder
	builder.WriteString(block.name)
	builder.WriteString(":\n")
	for _, instruction := range block.instructions {
		builder.WriteString("  ")
		builder.WriteString(instruction)
		builder.WriteString("\n")
	}
	return builder.String()
}
