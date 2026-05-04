package semantic

import "jayess-go/ast"

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(program *ast.Program) error {
	scope := newRootScope()
	context := rootContext()
	for _, statement := range program.Statements {
		if err := analyzeStatement(scope, context, statement); err != nil {
			return err
		}
	}
	return nil
}
