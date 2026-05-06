package semantic

import "jayess-go/ast"

func analyzeBreakStatement(context controlContext, stmt *ast.BreakStatement) error {
	if stmt.Label != "" {
		if _, ok := context.findLabel(stmt.Label); !ok {
			return errorAt(stmt, "unknown label %s", stmt.Label)
		}
		return nil
	}
	if !context.inLoop && !context.inSwitch {
		return errorAt(stmt, "break outside loop or switch")
	}
	return nil
}

func analyzeContinueStatement(context controlContext, stmt *ast.ContinueStatement) error {
	if stmt.Label != "" {
		label, ok := context.findLabel(stmt.Label)
		if !ok {
			return errorAt(stmt, "unknown label %s", stmt.Label)
		}
		if !label.allowsContinue {
			return errorAt(stmt, "continue target %s is not a loop", stmt.Label)
		}
		return nil
	}
	if !context.inLoop {
		return errorAt(stmt, "continue outside loop")
	}
	return nil
}
