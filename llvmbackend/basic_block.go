package llvmbackend

import (
	"fmt"
	"strconv"
	"strings"
)

type BasicBlockBuilder struct {
	next   int
	blocks []basicBlock
	open   *basicBlock
}

type basicBlock struct {
	Label      string
	Body       []string
	Terminator string
}

type PhiIncoming struct {
	Value string
	Label string
}

func (builder *BasicBlockBuilder) NewLabel(prefix string) string {
	if prefix == "" {
		prefix = "block"
	}
	label := sanitizeBlockLabel(prefix) + "." + strconv.Itoa(builder.next)
	builder.next++
	return label
}

func (builder *BasicBlockBuilder) Begin(label string) error {
	if label == "" {
		return fmt.Errorf("basic block label must not be empty")
	}
	if builder.open != nil && builder.open.Terminator == "" {
		return fmt.Errorf("basic block %s has no terminator", builder.open.Label)
	}
	builder.blocks = append(builder.blocks, basicBlock{Label: label})
	builder.open = &builder.blocks[len(builder.blocks)-1]
	return nil
}

func (builder *BasicBlockBuilder) Emit(line string) error {
	if builder.open == nil {
		return fmt.Errorf("cannot emit into unopened basic block")
	}
	if builder.open.Terminator != "" {
		return fmt.Errorf("cannot emit after terminator in basic block %s", builder.open.Label)
	}
	builder.open.Body = append(builder.open.Body, line)
	return nil
}

func (builder *BasicBlockBuilder) Branch(label string) error {
	return builder.terminate("br label %" + label)
}

func (builder *BasicBlockBuilder) ConditionalBranch(condition string, trueLabel string, falseLabel string) error {
	return builder.terminate("br i1 " + condition + ", label %" + trueLabel + ", label %" + falseLabel)
}

func (builder *BasicBlockBuilder) Return(valueType string, value string) error {
	return builder.terminate("ret " + valueType + " " + value)
}

func (builder *BasicBlockBuilder) Phi(result string, valueType string, incoming []PhiIncoming) error {
	if len(incoming) == 0 {
		return fmt.Errorf("phi node must have incoming values")
	}
	parts := make([]string, 0, len(incoming))
	for _, item := range incoming {
		if item.Value == "" || item.Label == "" {
			return fmt.Errorf("phi incoming value and label must not be empty")
		}
		parts = append(parts, "[ "+item.Value+", %"+item.Label+" ]")
	}
	return builder.Emit(result + " = phi " + valueType + " " + strings.Join(parts, ", "))
}

func (builder *BasicBlockBuilder) Lines() ([]string, error) {
	if builder.open != nil && builder.open.Terminator == "" {
		return nil, fmt.Errorf("basic block %s has no terminator", builder.open.Label)
	}
	lines := []string{}
	for _, block := range builder.blocks {
		lines = append(lines, block.Label+":")
		lines = append(lines, block.Body...)
		lines = append(lines, block.Terminator)
	}
	return lines, nil
}

func (builder *BasicBlockBuilder) terminate(line string) error {
	if builder.open == nil {
		return fmt.Errorf("cannot terminate unopened basic block")
	}
	if builder.open.Terminator != "" {
		return fmt.Errorf("basic block %s already has a terminator", builder.open.Label)
	}
	builder.open.Terminator = line
	return nil
}

func sanitizeBlockLabel(label string) string {
	var builder strings.Builder
	for _, ch := range label {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '.' {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteByte('_')
	}
	return builder.String()
}
