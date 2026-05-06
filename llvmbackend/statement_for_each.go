package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

const (
	runtimeForInIteratorSymbol = "jayess_for_in_iterator"
	runtimeForOfIteratorSymbol = "jayess_for_of_iterator"
	runtimeIteratorNextSymbol  = "jayess_iterator_next"
	runtimeIteratorDoneSymbol  = "jayess_iterator_done"
	runtimeIteratorValueSymbol = "jayess_iterator_value"
)

func (emitter *StatementEmitter) emitForInStatement(statement *ast.ForInStatement) error {
	source, err := emitter.expressions.EmitExpression(statement.Object)
	if err != nil {
		return err
	}
	return emitter.emitForEachLoop(statement, "for.in", runtimeForInIteratorSymbol, source, statement.Pattern, statement.Target, statement.Body)
}

func (emitter *StatementEmitter) emitForOfStatement(statement *ast.ForOfStatement) error {
	if statement.Await {
		return fmt.Errorf("unsupported runtime for await...of lowering")
	}
	source, err := emitter.expressions.EmitExpression(statement.Iterable)
	if err != nil {
		return err
	}
	return emitter.emitForEachLoop(statement, "for.of", runtimeForOfIteratorSymbol, source, statement.Pattern, statement.Target, statement.Body)
}

func (emitter *StatementEmitter) emitForEachLoop(node ast.Node, prefix string, iteratorSymbol string, source string, pattern ast.BindingPattern, target ast.Expression, statements []ast.Statement) error {
	assign, err := emitter.prepareForEachTarget(node, pattern, target)
	if err != nil {
		return err
	}

	iterator, err := emitter.emitIteratorCreate(iteratorSymbol, source)
	if err != nil {
		return err
	}
	conditionLabel := emitter.expressions.nextBlockLabel(prefix + ".cond")
	bodyLabel := emitter.expressions.nextBlockLabel(prefix + ".body")
	endLabel := emitter.expressions.nextBlockLabel(prefix + ".end")

	emitter.pushStructuredExit(structuredExit{kind: structuredExitLoop, breakLabel: endLabel, continueLabel: conditionLabel})
	body, err := emitter.captureScopedStatements(statements)
	emitter.popStructuredExit()
	if err != nil {
		return err
	}

	emitter.expressions.body = append(emitter.expressions.body,
		"br label %"+conditionLabel,
		conditionLabel+":",
	)
	next, err := emitter.emitIteratorNext(iterator)
	if err != nil {
		return err
	}
	done, err := emitter.emitIteratorDone(next)
	if err != nil {
		return err
	}
	emitter.expressions.body = append(emitter.expressions.body,
		"br i1 "+done+", label %"+endLabel+", label %"+bodyLabel,
		bodyLabel+":",
	)
	value, err := emitter.emitIteratorValue(next)
	if err != nil {
		return err
	}
	if err := assign(value); err != nil {
		return err
	}
	emitter.expressions.body = append(emitter.expressions.body, body.lines...)
	if !body.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+conditionLabel)
	}
	emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
	return nil
}

func (emitter *StatementEmitter) prepareForEachTarget(node ast.Node, pattern ast.BindingPattern, target ast.Expression) (func(string) error, error) {
	if pattern != nil {
		if err := emitter.declareBindingTargets(pattern, "undef"); err != nil {
			return nil, err
		}
		emitter.registerDeclarationLifetime(node, pattern)
		return func(value string) error {
			return emitter.emitDestructureToPattern(pattern, value)
		}, nil
	}
	assignment, err := emitter.expressions.ResolveAssignmentTarget(target)
	if err != nil {
		return nil, err
	}
	return assignment.Store, nil
}

func (emitter *StatementEmitter) emitIteratorCreate(symbol string, source string) (string, error) {
	return emitter.expressions.emitRuntimeUnaryValue(symbol, source)
}

func (emitter *StatementEmitter) emitIteratorNext(iterator string) (string, error) {
	return emitter.expressions.emitRuntimeUnaryValue(runtimeIteratorNextSymbol, iterator)
}

func (emitter *StatementEmitter) emitIteratorValue(next string) (string, error) {
	return emitter.expressions.emitRuntimeUnaryValue(runtimeIteratorValueSymbol, next)
}

func (emitter *StatementEmitter) emitIteratorDone(next string) (string, error) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: next}}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeIteratorDoneSymbol, "i1", args)})
	result := emitter.expressions.nextValueName()
	emitter.expressions.body = append(emitter.expressions.body, RuntimeCall(result, "i1", runtimeIteratorDoneSymbol, args))
	return result, nil
}
