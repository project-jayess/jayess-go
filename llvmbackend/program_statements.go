package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

const runtimeExitCodeSymbol = "jayess_value_to_exit_code"

type JayessStatementProgram struct {
	Name                 string
	Target               TargetConfig
	EntryName            string
	UserEntryName        string
	Statements           []ast.Statement
	ModuleInitialization ModuleInitializationPlan
}

func LowerJayessStatementProgram(program JayessStatementProgram) (Module, error) {
	entry := program.EntryName
	if entry == "" {
		entry = "main"
	}
	userEntry := program.UserEntryName
	if userEntry == "" {
		userEntry = "__jayess_user_main"
	}
	userMain, declarations, globals, err := LowerRuntimeStatementFunction(userEntry, program.Statements)
	if err != nil {
		return Module{}, err
	}

	mainBody := moduleInitializationCalls(program.ModuleInitialization)
	result := "%jayess.result"
	code := "%jayess.exit_code"
	mainBody = append(mainBody,
		result+" = call "+runtimeValueIRType+" @"+userEntry+"()",
		code+" = call i32 @"+runtimeExitCodeSymbol+"("+runtimeValueIRType+" "+result+")",
		"ret i32 "+code,
	)

	declarations = append(moduleInitializationDeclarations(program.ModuleInitialization), declarations...)
	declarations = append(declarations, RuntimeCallDeclaration(runtimeExitCodeSymbol, "i32", []RuntimeCallArg{{IRType: runtimeValueIRType}}))

	if entry == userEntry {
		return Module{}, fmt.Errorf("native entry name must differ from Jayess user entry name")
	}
	return Module{
		Name:         program.Name,
		Target:       program.Target,
		Globals:      globals,
		Declarations: declarations,
		Functions: []Function{
			{Name: entry, ReturnType: "i32", Body: mainBody},
			userMain,
		},
	}, nil
}
