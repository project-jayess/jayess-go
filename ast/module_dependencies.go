package ast

type ModuleDependency struct {
	Source     string
	ReExport   bool
	SideEffect bool
}

func ModuleDependencies(program *Program) []ModuleDependency {
	if program == nil {
		return nil
	}
	var dependencies []ModuleDependency
	for _, statement := range program.Statements {
		switch stmt := statement.(type) {
		case *ImportDecl:
			dependencies = append(dependencies, ModuleDependency{
				Source:     stmt.Source,
				SideEffect: stmt.SideEffect,
			})
		case *ExportDecl:
			if stmt.Source == "" {
				continue
			}
			dependencies = append(dependencies, ModuleDependency{
				Source:   stmt.Source,
				ReExport: true,
			})
		}
	}
	return dependencies
}
