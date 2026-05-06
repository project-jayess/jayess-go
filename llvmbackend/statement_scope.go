package llvmbackend

import "jayess-go/ast"

type lexicalScopeKind string

const (
	lexicalScopeBlock    lexicalScopeKind = "block"
	lexicalScopeLoop     lexicalScopeKind = "loop"
	lexicalScopeFunction lexicalScopeKind = "function"
	lexicalScopeCatch    lexicalScopeKind = "catch"
)

func (emitter *StatementEmitter) enterLexicalScope(kind lexicalScopeKind) {
	emitter.expressions.PushScope()
	emitter.cleanupScopes = append(emitter.cleanupScopes, nil)
}

func (emitter *StatementEmitter) leaveLexicalScope(emitNormalCleanups bool) error {
	if emitNormalCleanups {
		emitter.emitCurrentScopeCleanups()
	}
	if len(emitter.cleanupScopes) != 0 {
		emitter.cleanupScopes = emitter.cleanupScopes[:len(emitter.cleanupScopes)-1]
	}
	return emitter.expressions.PopScope()
}

func (emitter *StatementEmitter) emitScopedStatements(kind lexicalScopeKind, statements []ast.Statement) error {
	emitter.enterLexicalScope(kind)
	err := emitter.EmitStatements(statements)
	popErr := emitter.leaveLexicalScope(!emitter.returned)
	if err != nil {
		return err
	}
	return popErr
}
