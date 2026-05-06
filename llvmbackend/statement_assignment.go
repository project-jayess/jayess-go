package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *StatementEmitter) emitAssignmentStatement(statement *ast.AssignmentStatement) error {
	if statement.Operator != ast.AssignmentAssign {
		return fmt.Errorf("unsupported runtime assignment operator %s", statement.Operator)
	}
	target, err := emitter.expressions.ResolveAssignmentTarget(statement.Target)
	if err != nil {
		return err
	}
	value, err := emitter.expressions.EmitExpression(statement.Value)
	if err != nil {
		return err
	}
	return target.Store(value)
}
