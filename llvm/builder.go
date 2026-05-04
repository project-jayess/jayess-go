package llvm

import "fmt"

type Builder struct {
	block *BasicBlock
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (builder *Builder) PositionAtEnd(block *BasicBlock) {
	builder.block = block
}

func (builder *Builder) BuildRet(value Constant) error {
	if builder.block == nil {
		return fmt.Errorf("LLVM builder has no insertion block")
	}
	return builder.block.Return(value)
}

func (builder *Builder) BuildRetVoid() error {
	if builder.block == nil {
		return fmt.Errorf("LLVM builder has no insertion block")
	}
	return builder.block.ReturnVoid()
}
