package lowering

import (
	"jayess-go/ast"
	"jayess-go/lifetime"
)

type CleanupPath string

const (
	CleanupPathNormal   CleanupPath = "normal"
	CleanupPathReturn   CleanupPath = "return"
	CleanupPathBreak    CleanupPath = "break"
	CleanupPathContinue CleanupPath = "continue"
	CleanupPathThrow    CleanupPath = "throw"
)

type ControlFlowCleanupOp struct {
	CleanupOp
	Path       CleanupPath
	ExitLine   int
	ExitColumn int
}

type cleanupLowerer struct {
	cleanups         []lifetime.Cleanup
	ops              []ControlFlowCleanupOp
	breakBoundary    int
	continueBoundary int
}

// LowerControlFlowCleanupOps emits cleanup operations on normal and abrupt exits.
func LowerControlFlowCleanupOps(program *ast.Program, plan lifetime.Plan) []ControlFlowCleanupOp {
	if program == nil {
		return nil
	}
	lowerer := cleanupLowerer{
		cleanups:         plan.ScopeExitCleanups,
		breakBoundary:    -1,
		continueBoundary: -1,
	}
	lowerer.lowerScope(program.Statements, nil)
	return lowerer.ops
}

func (l *cleanupLowerer) lowerScope(statements []ast.Statement, incoming []CleanupOp) []CleanupOp {
	active := append([]CleanupOp(nil), incoming...)
	scopeStart := len(active)
	for _, statement := range statements {
		var fallsThrough bool
		active, fallsThrough = l.lowerStatement(statement, active)
		if !fallsThrough {
			return incoming
		}
	}
	l.emitCleanups(active[scopeStart:], CleanupPathNormal, ast.SourcePos{})
	return incoming
}

func (l *cleanupLowerer) lowerStatement(statement ast.Statement, active []CleanupOp) ([]CleanupOp, bool) {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		return append(active, l.cleanupOpsForDeclaration(stmt, stmt.Pattern)...), true
	case *ast.FunctionDecl:
		l.lowerScope(stmt.Body, nil)
		return active, true
	case *ast.BlockStatement:
		l.lowerScope(stmt.Statements, active)
		return active, true
	case *ast.IfStatement:
		l.lowerScope(stmt.Consequence, active)
		l.lowerScope(stmt.Alternative, active)
		return active, true
	case *ast.WhileStatement:
		l.lowerLoopBody(stmt.Body, active)
		return active, true
	case *ast.DoWhileStatement:
		l.lowerLoopBody(stmt.Body, active)
		return active, true
	case *ast.ForStatement:
		if stmt.Init != nil {
			var fallsThrough bool
			active, fallsThrough = l.lowerStatement(stmt.Init, active)
			if !fallsThrough {
				return active, false
			}
		}
		l.lowerLoopBody(stmt.Body, active)
		return active, true
	case *ast.ForOfStatement:
		loopActive := append(active, l.cleanupOpsForDeclaration(stmt, stmt.Pattern)...)
		l.lowerLoopBody(stmt.Body, loopActive)
		return active, true
	case *ast.ForInStatement:
		loopActive := append(active, l.cleanupOpsForDeclaration(stmt, stmt.Pattern)...)
		l.lowerLoopBody(stmt.Body, loopActive)
		return active, true
	case *ast.LabeledStatement:
		return l.lowerStatement(stmt.Statement, active)
	case *ast.SwitchStatement:
		lowerBreakBoundary := l.breakBoundary
		l.breakBoundary = len(active)
		for _, switchCase := range stmt.Cases {
			l.lowerScope(switchCase.Consequent, active)
		}
		l.lowerScope(stmt.Default, active)
		l.breakBoundary = lowerBreakBoundary
		return active, true
	case *ast.TryStatement:
		l.lowerScope(stmt.TryBody, active)
		catchActive := append(active, l.cleanupOpsForDeclaration(stmt, stmt.CatchPattern)...)
		l.lowerScope(stmt.CatchBody, catchActive)
		l.lowerScope(stmt.FinallyBody, active)
		return active, true
	case *ast.ReturnStatement:
		l.emitCleanups(active, CleanupPathReturn, ast.PositionOf(stmt))
		return active, false
	case *ast.BreakStatement:
		l.emitCleanupsFromBoundary(active, l.breakBoundary, CleanupPathBreak, ast.PositionOf(stmt))
		return active, false
	case *ast.ContinueStatement:
		l.emitCleanupsFromBoundary(active, l.continueBoundary, CleanupPathContinue, ast.PositionOf(stmt))
		return active, false
	case *ast.ThrowStatement:
		l.emitCleanups(active, CleanupPathThrow, ast.PositionOf(stmt))
		return active, false
	default:
		return active, true
	}
}

func (l *cleanupLowerer) lowerLoopBody(body []ast.Statement, active []CleanupOp) {
	oldBreakBoundary := l.breakBoundary
	oldContinueBoundary := l.continueBoundary
	l.breakBoundary = len(active)
	l.continueBoundary = len(active)
	l.lowerScope(body, active)
	l.breakBoundary = oldBreakBoundary
	l.continueBoundary = oldContinueBoundary
}

func (l *cleanupLowerer) cleanupOpsForDeclaration(node ast.Node, pattern ast.BindingPattern) []CleanupOp {
	pos := ast.PositionOf(node)
	names := cleanupBindingNames(pattern)
	ops := make([]CleanupOp, 0, len(names))
	for _, name := range names {
		for _, cleanup := range l.cleanups {
			if cleanup.Binding == name && cleanup.Line == pos.Line && cleanup.Column == pos.Column {
				ops = append(ops, CleanupOp{
					Binding:    cleanup.Binding,
					Line:       cleanup.Line,
					Column:     cleanup.Column,
					ScopeDepth: cleanup.ScopeDepth,
				})
				break
			}
		}
	}
	return ops
}

func (l *cleanupLowerer) emitCleanupsFromBoundary(active []CleanupOp, boundary int, path CleanupPath, exit ast.SourcePos) {
	if boundary < 0 || boundary > len(active) {
		return
	}
	l.emitCleanups(active[boundary:], path, exit)
}

func (l *cleanupLowerer) emitCleanups(cleanups []CleanupOp, path CleanupPath, exit ast.SourcePos) {
	for i := len(cleanups) - 1; i >= 0; i-- {
		l.ops = append(l.ops, ControlFlowCleanupOp{
			CleanupOp:  cleanups[i],
			Path:       path,
			ExitLine:   exit.Line,
			ExitColumn: exit.Column,
		})
	}
}

func cleanupBindingNames(pattern ast.BindingPattern) []string {
	switch pat := pattern.(type) {
	case *ast.BindingName:
		if pat.Name == "" {
			return nil
		}
		return []string{pat.Name}
	case *ast.BindingDefault:
		return cleanupBindingNames(pat.Pattern)
	case *ast.BindingRest:
		return cleanupBindingNames(pat.Pattern)
	case *ast.ArrayBindingPattern:
		var names []string
		for _, element := range pat.Elements {
			names = append(names, cleanupBindingNames(element)...)
		}
		return names
	case *ast.ObjectBindingPattern:
		var names []string
		for _, property := range pat.Properties {
			names = append(names, cleanupBindingNames(property.Pattern)...)
		}
		return names
	default:
		return nil
	}
}
