package llvmbackend

import "jayess-go/ast"

func (emitter *ExpressionEmitter) emitConditionalExpression(expression *ast.ConditionalExpression) (string, error) {
	conditionValue, err := emitter.EmitExpression(expression.Condition)
	if err != nil {
		return "", err
	}
	condition, err := emitter.EmitTruthiness(conditionValue)
	if err != nil {
		return "", err
	}
	return emitter.emitBranchValue("conditional", condition, expression.Consequent, expression.Alternative)
}
