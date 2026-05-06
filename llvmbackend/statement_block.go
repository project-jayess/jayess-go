package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitBlockStatement(statement *ast.BlockStatement) error {
	return emitter.emitScopedStatements(lexicalScopeBlock, statement.Statements)
}
