package lowering

import (
	"jayess-go/ast"
	"jayess-go/ir"
)

func LowerClasses(program *ast.Program) []ir.ClassDecl {
	classes := make([]ir.ClassDecl, 0, len(program.Classes))
	for _, classDecl := range program.Classes {
		lowered := ir.ClassDecl{
			Name:       classDecl.Name,
			SuperClass: classDecl.SuperClass,
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				lowered.Fields = append(lowered.Fields, ir.ClassField{
					Name:           member.Name,
					Private:        member.Private,
					Static:         member.Static,
					HasInitializer: member.Initializer != nil,
				})
			case *ast.ClassMethodDecl:
				lowered.Methods = append(lowered.Methods, ir.ClassMethod{
					Name:          member.Name,
					Private:       member.Private,
					Static:        member.Static,
					IsConstructor: member.IsConstructor,
					IsGetter:      member.IsGetter,
					IsSetter:      member.IsSetter,
					ParamCount:    len(member.Params),
				})
			}
		}
		classes = append(classes, lowered)
	}
	return classes
}
