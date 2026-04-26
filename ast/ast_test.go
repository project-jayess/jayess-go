package ast

import "testing"

func TestPositionOfReturnsSourcePositionForPositionedNodes(t *testing.T) {
	node := &Identifier{
		BaseNode: BaseNode{Pos: SourcePos{Line: 7, Column: 3}},
		Name:     "value",
	}

	pos := PositionOf(node)
	if pos.Line != 7 || pos.Column != 3 {
		t.Fatalf("expected position 7:3, got %d:%d", pos.Line, pos.Column)
	}
}

func TestPositionOfReturnsZeroForNonPositionedValues(t *testing.T) {
	pos := PositionOf(struct{}{})
	if pos.Line != 0 || pos.Column != 0 {
		t.Fatalf("expected zero position, got %d:%d", pos.Line, pos.Column)
	}
}
