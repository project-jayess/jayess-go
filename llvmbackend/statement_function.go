package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *StatementEmitter) emitFunctionDeclaration(statement *ast.FunctionDecl) error {
	if statement == nil || statement.Name == "" {
		return fmt.Errorf("runtime function declaration must have a name")
	}
	value, err := emitter.expressions.emitRuntimeFunctionNew()
	if err != nil {
		return err
	}
	return emitter.expressions.DeclareLocal(statement.Name, value)
}
