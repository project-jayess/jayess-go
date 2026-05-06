package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/lifetime"
)

type StatementEmitter struct {
	expressions      *ExpressionEmitter
	returned         bool
	termination      statementTermination
	terminationLabel string
	exits            []structuredExit
	pendingLabels    []string
	returnSlot       localSlot
	returnLabel      string
	hasReturnTarget  bool
	throwSlot        localSlot
	throwLabel       string
	hasThrowTarget   bool
	throwHandlers    []string
	lifetimePlan     *lifetime.Plan
	cleanupScopes    [][]lifetimeCleanup
}

type statementTermination string

const (
	statementTerminationNone     statementTermination = ""
	statementTerminationReturn   statementTermination = "return"
	statementTerminationBreak    statementTermination = "break"
	statementTerminationContinue statementTermination = "continue"
	statementTerminationThrow    statementTermination = "throw"
)

func NewStatementEmitter() *StatementEmitter {
	return &StatementEmitter{expressions: NewExpressionEmitter()}
}

func NewStatementEmitterWithLifetimePlan(plan lifetime.Plan) *StatementEmitter {
	emitter := NewStatementEmitter()
	emitter.lifetimePlan = &plan
	emitter.expressions.SetLifetimePlan(&plan)
	return emitter
}

func LowerRuntimeStatementFunction(name string, statements []ast.Statement) (Function, []Declaration, []Global, error) {
	return lowerRuntimeStatementFunctionWithEmitter(name, statements, NewStatementEmitter())
}

func LowerRuntimeStatementFunctionWithLifetimePlan(name string, statements []ast.Statement, plan lifetime.Plan) (Function, []Declaration, []Global, error) {
	return lowerRuntimeStatementFunctionWithEmitter(name, statements, NewStatementEmitterWithLifetimePlan(plan))
}

func LowerRuntimeProgramFunction(name string, program *ast.Program) (Function, []Declaration, []Global, error) {
	if program == nil {
		return Function{}, nil, nil, fmt.Errorf("runtime program must not be nil")
	}
	plan := lifetime.BuildScopeExitPlan(program)
	return LowerRuntimeStatementFunctionWithLifetimePlan(name, program.Statements, plan)
}

func lowerRuntimeStatementFunctionWithEmitter(name string, statements []ast.Statement, emitter *StatementEmitter) (Function, []Declaration, []Global, error) {
	if name == "" {
		return Function{}, nil, nil, fmt.Errorf("runtime statement function name must not be empty")
	}
	if err := emitter.EmitStatements(statements); err != nil {
		return Function{}, nil, nil, err
	}
	body := emitter.Body()
	if emitter.hasReturnTarget {
		body = emitter.appendReturnTarget(body)
	}
	if emitter.hasThrowTarget {
		body = emitter.appendThrowTarget(body)
	}
	if !emitter.hasReturnTarget && !emitter.hasThrowTarget && !emitter.Returned() {
		body = append(body, "ret "+runtimeValueIRType+" undef")
	}
	return Function{
		Name:       name,
		ReturnType: runtimeValueIRType,
		Body:       body,
	}, emitter.Declarations(), emitter.Globals(), nil
}

func (emitter *StatementEmitter) EmitStatements(statements []ast.Statement) error {
	for _, statement := range statements {
		if emitter.returned {
			return fmt.Errorf("unreachable statement after lowered return")
		}
		if err := emitter.EmitStatement(statement); err != nil {
			return err
		}
	}
	return nil
}

func (emitter *StatementEmitter) EmitStatement(statement ast.Statement) (err error) {
	defer func() {
		err = diagnosticError(statement, err)
	}()
	switch stmt := statement.(type) {
	case *ast.EmptyStatement:
		return nil
	case *ast.VariableDecl:
		return emitter.emitVariableDeclaration(stmt)
	case *ast.FunctionDecl:
		return emitter.emitFunctionDeclaration(stmt)
	case *ast.ClassDecl:
		return emitter.emitClassDeclaration(stmt)
	case *ast.ImportDecl:
		return emitter.emitImportDeclaration(stmt)
	case *ast.ExportDecl:
		return emitter.emitExportDeclaration(stmt)
	case *ast.AssignmentStatement:
		return emitter.emitAssignmentStatement(stmt)
	case *ast.BlockStatement:
		return emitter.emitBlockStatement(stmt)
	case *ast.IfStatement:
		return emitter.emitIfStatement(stmt)
	case *ast.WhileStatement:
		return emitter.emitWhileStatement(stmt)
	case *ast.DoWhileStatement:
		return emitter.emitDoWhileStatement(stmt)
	case *ast.ForStatement:
		return emitter.emitForStatement(stmt)
	case *ast.ForInStatement:
		return emitter.emitForInStatement(stmt)
	case *ast.ForOfStatement:
		return emitter.emitForOfStatement(stmt)
	case *ast.SwitchStatement:
		return emitter.emitSwitchStatement(stmt)
	case *ast.LabeledStatement:
		return emitter.emitLabeledStatement(stmt)
	case *ast.BreakStatement:
		return emitter.emitBreak(stmt.Label)
	case *ast.ContinueStatement:
		return emitter.emitContinue(stmt.Label)
	case *ast.ExpressionStatement:
		_, err := emitter.expressions.EmitExpression(stmt.Expression)
		return err
	case *ast.ReturnStatement:
		return emitter.emitReturnStatement(stmt)
	case *ast.ThrowStatement:
		return emitter.emitThrowStatement(stmt)
	case *ast.TryStatement:
		return emitter.emitTryStatement(stmt)
	default:
		return fmt.Errorf("unsupported runtime statement %T", statement)
	}
}

func bindingName(pattern ast.BindingPattern) (string, bool) {
	name, ok := pattern.(*ast.BindingName)
	if !ok || name.Name == "" {
		return "", false
	}
	return name.Name, true
}

func (emitter *StatementEmitter) Body() []string {
	return emitter.expressions.Body()
}

func (emitter *StatementEmitter) Declarations() []Declaration {
	return emitter.expressions.Declarations()
}

func (emitter *StatementEmitter) Globals() []Global {
	return emitter.expressions.Globals()
}

func (emitter *StatementEmitter) Returned() bool {
	return emitter.returned
}
