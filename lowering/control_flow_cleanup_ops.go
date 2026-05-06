package lowering

import "jayess-go/ast"

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
