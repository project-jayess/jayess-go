package llvm

import (
	"fmt"
	"strings"
)

type Function struct {
	name       string
	returnType Type
	blocks     []BasicBlock
}

func (function *Function) AppendBlock(name string) (*BasicBlock, error) {
	if name == "" {
		return nil, fmt.Errorf("LLVM basic block name must not be empty")
	}
	block := BasicBlock{name: name}
	function.blocks = append(function.blocks, block)
	return &function.blocks[len(function.blocks)-1], nil
}

func (function Function) String() string {
	var builder strings.Builder
	builder.WriteString("define ")
	builder.WriteString(function.returnType.String())
	builder.WriteString(" @")
	builder.WriteString(function.name)
	builder.WriteString("() {\n")
	for index := range function.blocks {
		builder.WriteString(function.blocks[index].String())
	}
	builder.WriteString("}")
	return builder.String()
}
