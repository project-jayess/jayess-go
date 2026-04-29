package semantic

import (
	"fmt"
	"strconv"
	"strings"

	"jayess-go/ast"
	"jayess-go/typesys"
)

type symbol struct {
	kind    string
	mutable bool
}

type functionSignature struct {
	name       string
	nativeName string
	paramCount int
	minArgs    int
	hasRest    bool
	isMain     bool
	isExtern   bool
	isAsync    bool
	variadic   bool
	paramTypes []string
	returnType string
}

type Analyzer struct{}

type DiagnosticError struct {
	Line    int
	Column  int
	Message string
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	if e.Line > 0 {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

func New() *Analyzer {
	return &Analyzer{}
}

func errorAt(node any, format string, args ...any) error {
	pos := ast.PositionOf(node)
	return &DiagnosticError{
		Line:    pos.Line,
		Column:  pos.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

type classInfo struct {
	name                 string
	superClass           string
	hasConstructor       bool
	instanceFields       map[string]bool
	privateFields        map[string]bool
	staticFields         map[string]bool
	privateStaticFields  map[string]bool
	instanceMethods      map[string]bool
	instanceGetters      map[string]bool
	instanceSetters      map[string]bool
	privateMethods       map[string]bool
	staticMethods        map[string]bool
	staticGetters        map[string]bool
	staticSetters        map[string]bool
	privateStaticMethods map[string]bool
}

type classContext struct {
	info          *classInfo
	classes       map[string]*classInfo
	inConstructor bool
	inStatic      bool
}

func (a *Analyzer) Analyze(program *ast.Program) error {
	if len(program.Functions) == 0 {
		return errorAt(program, "program must declare at least one function")
	}

	globalSymbols := map[string]symbol{}
	availableFunctions := map[string]functionSignature{}
	for _, fn := range program.ExternFunctions {
		availableFunctions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isExtern: true, variadic: fn.Variadic, paramTypes: parameterTypes(fn.Params)}
	}
	for _, fn := range program.Functions {
		availableFunctions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isMain: fn.Name == "main", isAsync: fn.IsAsync, paramTypes: parameterTypes(fn.Params), returnType: normalizeTypeAnnotation(fn.ReturnType)}
	}
	for _, global := range program.Globals {
		if global.Kind != ast.DeclarationConst && global.Kind != ast.DeclarationVar {
			return errorAt(global, "top-level variables must use var or const")
		}
		if err := validateVariableAnnotation(global); err != nil {
			return err
		}
		if _, exists := globalSymbols[global.Name]; exists {
			return errorAt(global, "duplicate global %s", global.Name)
		}
		kind, err := inferExpressionKind(global.Value, globalSymbols, availableFunctions)
		if err != nil {
			return err
		}
		if expected := normalizeTypeAnnotation(global.TypeAnnotation); expected != "" && !isExpressionAssignableToType(expected, global.Value, kind, globalSymbols, availableFunctions) {
			return errorAt(global, "cannot initialize %s variable %s with %s", expected, global.Name, kind)
		}
		if !isRuntimeValueKind(kind) {
			return errorAt(global, "global %s must be a runtime value", global.Name)
		}
		globalKind := "dynamic"
		if global.TypeAnnotation != "" {
			globalKind = normalizeTypeAnnotation(global.TypeAnnotation)
		}
		globalSymbols[global.Name] = symbol{kind: globalKind, mutable: global.Kind == ast.DeclarationVar}
	}

	seenMain := false
	functions := map[string]functionSignature{}
	for _, fn := range program.ExternFunctions {
		if _, exists := globalSymbols[fn.Name]; exists {
			return errorAt(fn, "name %s is already used by a global", fn.Name)
		}
		if _, exists := functions[fn.Name]; exists {
			return errorAt(fn, "duplicate function %s", fn.Name)
		}
		functions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isExtern: true, variadic: fn.Variadic, paramTypes: parameterTypes(fn.Params)}
	}
	for _, fn := range program.Functions {
		if _, exists := globalSymbols[fn.Name]; exists {
			return errorAt(fn, "name %s is already used by a global", fn.Name)
		}
		if _, exists := functions[fn.Name]; exists {
			return errorAt(fn, "duplicate function %s", fn.Name)
		}
		functions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isMain: fn.Name == "main", isAsync: fn.IsAsync, paramTypes: parameterTypes(fn.Params), returnType: normalizeTypeAnnotation(fn.ReturnType)}
		if fn.Name == "main" {
			seenMain = true
		}
	}
	for _, fn := range program.Functions {
		if err := validateFunction(fn, functions, globalSymbols); err != nil {
			return err
		}
	}
	if !seenMain {
		return errorAt(program, "entrypoint function main was not found")
	}
	return nil
}

func (a *Analyzer) AnalyzeClasses(program *ast.Program) error {
	if len(program.Classes) == 0 {
		return nil
	}

	globalSymbols := map[string]symbol{}
	for _, global := range program.Globals {
		kind := "dynamic"
		if global.TypeAnnotation != "" {
			kind = normalizeTypeAnnotation(global.TypeAnnotation)
		}
		globalSymbols[global.Name] = symbol{kind: kind, mutable: global.Kind == ast.DeclarationVar}
	}
	functions := map[string]functionSignature{}
	for _, fn := range program.ExternFunctions {
		functions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isExtern: true, variadic: fn.Variadic, paramTypes: parameterTypes(fn.Params)}
	}
	for _, fn := range program.Functions {
		functions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isMain: fn.Name == "main", isAsync: fn.IsAsync, paramTypes: parameterTypes(fn.Params), returnType: normalizeTypeAnnotation(fn.ReturnType)}
	}

	classes := map[string]*classInfo{}
	taken := map[string]string{}
	for _, global := range program.Globals {
		taken[global.Name] = "global"
	}
	for _, fn := range program.Functions {
		taken[fn.Name] = "function"
	}
	for _, classDecl := range program.Classes {
		if prev, ok := taken[classDecl.Name]; ok {
			return errorAt(classDecl, "class %s conflicts with %s %s", classDecl.Name, prev, classDecl.Name)
		}
		taken[classDecl.Name] = "class"
		if _, exists := classes[classDecl.Name]; exists {
			return errorAt(classDecl, "duplicate class %s", classDecl.Name)
		}
		info := &classInfo{
			name:                 classDecl.Name,
			superClass:           classDecl.SuperClass,
			instanceFields:       map[string]bool{},
			privateFields:        map[string]bool{},
			staticFields:         map[string]bool{},
			privateStaticFields:  map[string]bool{},
			instanceMethods:      map[string]bool{},
			instanceGetters:      map[string]bool{},
			instanceSetters:      map[string]bool{},
			privateMethods:       map[string]bool{},
			staticMethods:        map[string]bool{},
			staticGetters:        map[string]bool{},
			staticSetters:        map[string]bool{},
			privateStaticMethods: map[string]bool{},
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				if member.TypeAnnotation != "" && !isSupportedTypeAnnotation(member.TypeAnnotation) {
					return errorAt(member, "field %s has unsupported type annotation %s", member.Name, member.TypeAnnotation)
				}
				if normalizeTypeAnnotation(member.TypeAnnotation) == "void" {
					return errorAt(member, "field %s cannot be annotated as void", member.Name)
				}
				if member.Static {
					if member.Private {
						if info.privateStaticFields[member.Name] {
							return errorAt(member, "duplicate private static field #%s in class %s", member.Name, classDecl.Name)
						}
						info.privateStaticFields[member.Name] = true
					} else {
						if info.staticFields[member.Name] {
							return errorAt(member, "duplicate static field %s in class %s", member.Name, classDecl.Name)
						}
						info.staticFields[member.Name] = true
					}
				} else if member.Private {
					if info.privateFields[member.Name] {
						return errorAt(member, "duplicate private field #%s in class %s", member.Name, classDecl.Name)
					}
					info.privateFields[member.Name] = true
				} else {
					if info.instanceFields[member.Name] {
						return errorAt(member, "duplicate field %s in class %s", member.Name, classDecl.Name)
					}
					info.instanceFields[member.Name] = true
				}
			case *ast.ClassMethodDecl:
				if member.IsConstructor {
					if info.hasConstructor {
						return errorAt(member, "duplicate constructor in class %s", classDecl.Name)
					}
					info.hasConstructor = true
					continue
				}
				if member.IsGetter || member.IsSetter {
					if member.Private {
						return errorAt(member, "private getters and setters are not supported yet")
					}
					if member.IsGetter && member.IsSetter {
						return errorAt(member, "class accessor %s cannot be both getter and setter", member.Name)
					}
					if member.IsGetter && len(member.Params) != 0 {
						return errorAt(member, "getter %s must not declare parameters", member.Name)
					}
					if member.IsSetter && len(member.Params) != 1 {
						return errorAt(member, "setter %s must declare exactly one parameter", member.Name)
					}
					if member.Static {
						if member.IsGetter {
							if info.staticGetters[member.Name] {
								return errorAt(member, "duplicate static getter %s in class %s", member.Name, classDecl.Name)
							}
							info.staticGetters[member.Name] = true
						} else {
							if info.staticSetters[member.Name] {
								return errorAt(member, "duplicate static setter %s in class %s", member.Name, classDecl.Name)
							}
							info.staticSetters[member.Name] = true
						}
					} else {
						if member.IsGetter {
							if info.instanceGetters[member.Name] {
								return errorAt(member, "duplicate getter %s in class %s", member.Name, classDecl.Name)
							}
							info.instanceGetters[member.Name] = true
						} else {
							if info.instanceSetters[member.Name] {
								return errorAt(member, "duplicate setter %s in class %s", member.Name, classDecl.Name)
							}
							info.instanceSetters[member.Name] = true
						}
					}
					continue
				}
				switch {
				case member.Static && member.Private:
					if info.privateStaticMethods[member.Name] {
						return errorAt(member, "duplicate private static method #%s in class %s", member.Name, classDecl.Name)
					}
					info.privateStaticMethods[member.Name] = true
				case member.Static:
					if info.staticMethods[member.Name] {
						return errorAt(member, "duplicate static method %s in class %s", member.Name, classDecl.Name)
					}
					info.staticMethods[member.Name] = true
				case member.Private:
					if info.privateMethods[member.Name] {
						return errorAt(member, "duplicate private method #%s in class %s", member.Name, classDecl.Name)
					}
					info.privateMethods[member.Name] = true
				default:
					if info.instanceMethods[member.Name] {
						return errorAt(member, "duplicate method %s in class %s", member.Name, classDecl.Name)
					}
					info.instanceMethods[member.Name] = true
				}
			}
		}
		classes[classDecl.Name] = info
	}

	for _, classDecl := range program.Classes {
		info := classes[classDecl.Name]
		if info.superClass == "" {
			continue
		}
		if _, ok := classes[info.superClass]; !ok {
			return errorAt(classDecl, "class %s extends unknown class %s", info.name, info.superClass)
		}
	}
	for _, info := range classes {
		seen := map[string]bool{info.name: true}
		next := info.superClass
		for next != "" {
			if seen[next] {
				return errorAt(program, "inheritance cycle detected involving class %s", info.name)
			}
			seen[next] = true
			next = classes[next].superClass
		}
	}

	for _, classDecl := range program.Classes {
		info := classes[classDecl.Name]
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
				if member.Initializer != nil {
					if expected := normalizeTypeAnnotation(member.TypeAnnotation); expected != "" {
						kind, err := inferExpressionKind(member.Initializer, globalSymbols, functions)
						if err != nil {
							return err
						}
						if !isExpressionAssignableToType(expected, member.Initializer, kind, globalSymbols, functions) {
							return errorAt(member, "cannot initialize %s field %s with %s", expected, member.Name, kind)
						}
					}
					if err := a.validateClassExpression(member.Initializer, &classContext{
						info:     info,
						classes:  classes,
						inStatic: member.Static,
					}); err != nil {
						return err
					}
				}
			case *ast.ClassMethodDecl:
				ctx := &classContext{
					info:          info,
					classes:       classes,
					inConstructor: member.IsConstructor,
					inStatic:      member.Static,
				}
				if err := a.validateClassStatements(member.Body, ctx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateFunction(fn *ast.FunctionDecl, functions map[string]functionSignature, globals map[string]symbol) error {
	if len(fn.Body) == 0 {
		return errorAt(fn, "function %s must contain at least one statement", fn.Name)
	}
	if err := validateParameterList(fn.Params); err != nil {
		return errorAt(fn, "function %s: %v", fn.Name, err)
	}
	if fn.ReturnType != "" && !isSupportedTypeAnnotation(fn.ReturnType) {
		return errorAt(fn, "function %s has unsupported return type annotation %s", fn.Name, fn.ReturnType)
	}
	if fn.Name == "main" && len(fn.Params) > 1 {
		return errorAt(fn, "main supports at most one parameter: args")
	}
	if fn.Name == "main" && fn.IsAsync {
		return errorAt(fn, "main cannot be async")
	}

	symbols := cloneSymbols(globals)
	if fn.Name == "main" && len(fn.Params) == 1 {
		if expected := normalizeTypeAnnotation(fn.Params[0].TypeAnnotation); expected != "" && expected != "array" && expected != "dynamic" {
			return errorAt(fn, "main args parameter must be annotated as array or dynamic")
		}
		symbols[fn.Params[0].Name] = symbol{kind: "args_array", mutable: false}
	}
	if fn.Name != "main" {
		for _, param := range fn.Params {
			kind := normalizeTypeAnnotation(param.TypeAnnotation)
			if kind == "" {
				kind = "dynamic"
			}
			symbols[param.Name] = symbol{kind: kind, mutable: true}
		}
	}
	for _, param := range fn.Params {
		if param.Default != nil {
			kind, err := inferExpressionKind(param.Default, symbols, functions)
			if err != nil {
				return err
			}
			if expected := normalizeTypeAnnotation(param.TypeAnnotation); expected != "" && !isExpressionAssignableToType(expected, param.Default, kind, symbols, functions) {
				return errorAt(param.Default, "default value for parameter %s must be %s, got %s", param.Name, expected, kind)
			}
		}
	}

	if err := validateStatementsWithReturn(fn.Body[:len(fn.Body)-1], symbols, false, functions, normalizeTypeAnnotation(fn.ReturnType)); err != nil {
		return err
	}

	switch last := fn.Body[len(fn.Body)-1].(type) {
	case *ast.ReturnStatement:
		kind, err := inferExpressionKind(last.Value, symbols, functions)
		if err != nil {
			return err
		}
		if expected := normalizeTypeAnnotation(fn.ReturnType); expected != "" && !isExpressionAssignableToType(expected, last.Value, kind, symbols, functions) {
			return errorAt(last, "function %s must return %s, got %s", fn.Name, expected, kind)
		}
		if fn.Name == "main" && !isRuntimeValueKind(kind) {
			return errorAt(last, "function %s must return a number-like value", fn.Name)
		}
		if fn.Name != "main" && !isRuntimeValueKind(kind) {
			return errorAt(last, "function %s must return a runtime value", fn.Name)
		}
	case *ast.ThrowStatement:
		if normalizeTypeAnnotation(fn.ReturnType) != "never" {
			return errorAt(fn, "function %s must terminate with a return statement", fn.Name)
		}
		kind, err := inferExpressionKind(last.Value, symbols, functions)
		if err != nil {
			return err
		}
		if !isRuntimeValueKind(kind) {
			return errorAt(last, "throw expects a runtime-compatible value")
		}
	default:
		return errorAt(fn, "function %s must terminate with a return statement", fn.Name)
	}
	return nil
}

func validateStatements(statements []ast.Statement, symbols map[string]symbol, inLoop bool, functions map[string]functionSignature) error {
	return validateStatementsWithReturn(statements, symbols, inLoop, functions, "")
}

type controlTarget struct {
	label                string
	allowsUnlabeledBreak bool
	allowsLabeledBreak   bool
	allowsContinue       bool
}

func validateStatementsWithReturn(statements []ast.Statement, symbols map[string]symbol, inLoop bool, functions map[string]functionSignature, expectedReturn string) error {
	var controls []controlTarget
	if inLoop {
		controls = append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true})
	}
	return validateStatementsWithReturnContext(statements, symbols, controls, functions, expectedReturn)
}

func validateStatementsWithReturnContext(statements []ast.Statement, symbols map[string]symbol, controls []controlTarget, functions map[string]functionSignature, expectedReturn string) error {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			if err := validateVariableAnnotation(stmt); err != nil {
				return err
			}
			kind, err := inferExpressionKind(stmt.Value, symbols, functions)
			if err != nil {
				return err
			}
			if expected := normalizeTypeAnnotation(stmt.TypeAnnotation); expected != "" {
				if !isExpressionAssignableToType(expected, stmt.Value, kind, symbols, functions) {
					return errorAt(stmt, "cannot initialize %s variable %s with %s", expected, stmt.Name, kind)
				}
				kind = expected
			} else if stmt.Kind == ast.DeclarationVar {
				kind = "dynamic"
			}
			symbols[stmt.Name] = symbol{kind: kind, mutable: stmt.Kind != ast.DeclarationConst}
		case *ast.AssignmentStatement:
			if err := validateAssignment(stmt, symbols, functions); err != nil {
				return err
			}
		case *ast.DeleteStatement:
			if err := validateDelete(stmt, symbols, functions); err != nil {
				return err
			}
		case *ast.ThrowStatement:
			kind, err := inferExpressionKind(stmt.Value, symbols, functions)
			if err != nil {
				return err
			}
			if !isRuntimeValueKind(kind) {
				return errorAt(stmt, "throw expects a runtime-compatible value")
			}
		case *ast.TryStatement:
			trySymbols := cloneSymbols(symbols)
			if err := validateStatementsWithReturnContext(stmt.TryBody, trySymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
			catchSymbols := cloneSymbols(symbols)
			if stmt.CatchName != "" {
				catchSymbols[stmt.CatchName] = symbol{kind: "dynamic", mutable: true}
			}
			if err := validateStatementsWithReturnContext(stmt.CatchBody, catchSymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
			finallySymbols := cloneSymbols(symbols)
			if err := validateStatementsWithReturnContext(stmt.FinallyBody, finallySymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
		case *ast.ReturnStatement:
			kind, err := inferExpressionKind(stmt.Value, symbols, functions)
			if err != nil {
				return err
			}
			if expectedReturn != "" && !isExpressionAssignableToType(expectedReturn, stmt.Value, kind, symbols, functions) {
				return errorAt(stmt, "return expects %s, got %s", expectedReturn, kind)
			}
		case *ast.ExpressionStatement:
			if _, err := inferExpressionKind(stmt.Expression, symbols, functions); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := validateCondition(stmt.Condition, symbols, functions); err != nil {
				return err
			}
			consequenceSymbols := cloneSymbols(symbols)
			applyConditionNarrowing(stmt.Condition, consequenceSymbols, functions, true)
			if err := validateStatementsWithReturnContext(stmt.Consequence, consequenceSymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
			alternativeSymbols := cloneSymbols(symbols)
			applyConditionNarrowing(stmt.Condition, alternativeSymbols, functions, false)
			if err := validateStatementsWithReturnContext(stmt.Alternative, alternativeSymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
		case *ast.BlockStatement:
			blockSymbols := cloneSymbols(symbols)
			if err := validateStatementsWithReturnContext(stmt.Body, blockSymbols, controls, functions, expectedReturn); err != nil {
				return err
			}
		case *ast.WhileStatement:
			if err := validateCondition(stmt.Condition, symbols, functions); err != nil {
				return err
			}
			bodySymbols := cloneSymbols(symbols)
			if err := validateStatementsWithReturnContext(stmt.Body, bodySymbols, append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true}), functions, expectedReturn); err != nil {
				return err
			}
		case *ast.DoWhileStatement:
			bodySymbols := cloneSymbols(symbols)
			if err := validateStatementsWithReturnContext(stmt.Body, bodySymbols, append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true}), functions, expectedReturn); err != nil {
				return err
			}
			if err := validateCondition(stmt.Condition, symbols, functions); err != nil {
				return err
			}
		case *ast.ForStatement:
			loopSymbols := cloneSymbols(symbols)
			if stmt.Init != nil {
				if err := validateLoopStatement(stmt.Init, loopSymbols, functions); err != nil {
					return err
				}
			}
			if stmt.Condition != nil {
				if err := validateCondition(stmt.Condition, loopSymbols, functions); err != nil {
					return err
				}
			}
			bodySymbols := cloneSymbols(loopSymbols)
			if err := validateStatementsWithReturnContext(stmt.Body, bodySymbols, append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true}), functions, expectedReturn); err != nil {
				return err
			}
			if stmt.Update != nil {
				if err := validateLoopStatement(stmt.Update, bodySymbols, functions); err != nil {
					return err
				}
			}
		case *ast.ForOfStatement:
			iterKind, err := inferExpressionKind(stmt.Iterable, symbols, functions)
			if err != nil {
				return err
			}
			if iterKind != "array" && iterKind != "dynamic" && iterKind != "args_array" && iterKind != "string" && iterKind != "object" && iterKind != "function" {
				return errorAt(stmt, "for...of expects an iterable value")
			}
			loopSymbols := cloneSymbols(symbols)
			bindingKind := "dynamic"
			if stmt.Kind == ast.DeclarationConst {
				bindingKind = "dynamic"
			}
			loopSymbols[stmt.Name] = symbol{kind: bindingKind, mutable: stmt.Kind != ast.DeclarationConst}
			if err := validateStatementsWithReturnContext(stmt.Body, loopSymbols, append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true}), functions, expectedReturn); err != nil {
				return err
			}
		case *ast.ForInStatement:
			iterKind, err := inferExpressionKind(stmt.Iterable, symbols, functions)
			if err != nil {
				return err
			}
			if iterKind != "object" && iterKind != "dynamic" && iterKind != "function" {
				return errorAt(stmt, "for...in expects an object-like iterable")
			}
			loopSymbols := cloneSymbols(symbols)
			loopSymbols[stmt.Name] = symbol{kind: "string", mutable: stmt.Kind != ast.DeclarationConst}
			if err := validateStatementsWithReturnContext(stmt.Body, loopSymbols, append(controls, controlTarget{allowsUnlabeledBreak: true, allowsContinue: true}), functions, expectedReturn); err != nil {
				return err
			}
		case *ast.SwitchStatement:
			if _, err := inferExpressionKind(stmt.Discriminant, symbols, functions); err != nil {
				return err
			}
			switchControls := append(controls, controlTarget{allowsUnlabeledBreak: true})
			excludedLiterals := []string{}
			for _, switchCase := range stmt.Cases {
				if _, err := inferExpressionKind(switchCase.Test, symbols, functions); err != nil {
					return err
				}
				caseSymbols := cloneSymbols(symbols)
				applySwitchNarrowing(stmt.Discriminant, switchCase.Test, caseSymbols)
				if literal, ok := literalTypeFromExpression(switchCase.Test); ok {
					excludedLiterals = append(excludedLiterals, literal)
				}
				if err := validateStatementsWithReturnContext(switchCase.Consequent, caseSymbols, switchControls, functions, expectedReturn); err != nil {
					return err
				}
			}
			defaultSymbols := cloneSymbols(symbols)
			applyDefaultSwitchNarrowing(stmt.Discriminant, excludedLiterals, defaultSymbols)
			if err := validateStatementsWithReturnContext(stmt.Default, defaultSymbols, switchControls, functions, expectedReturn); err != nil {
				return err
			}
		case *ast.LabeledStatement:
			labelControls := append(controls, controlTarget{
				label:              stmt.Label,
				allowsLabeledBreak: true,
				allowsContinue:     isIterationStatement(stmt.Statement),
			})
			if err := validateStatementsWithReturnContext([]ast.Statement{stmt.Statement}, cloneSymbols(symbols), labelControls, functions, expectedReturn); err != nil {
				return err
			}
		case *ast.BreakStatement:
			if !canBreak(controls, stmt.Label) {
				if stmt.Label != "" {
					return errorAt(stmt, "unknown break label %s", stmt.Label)
				}
				return errorAt(stmt, "break is only valid inside loops and switch statements")
			}
		case *ast.ContinueStatement:
			if !canContinue(controls, stmt.Label) {
				if stmt.Label != "" {
					return errorAt(stmt, "unknown continue label %s", stmt.Label)
				}
				return errorAt(stmt, "continue is only valid inside loops")
			}
		default:
			return errorAt(stmt, "unsupported statement")
		}
	}
	return nil
}

func isIterationStatement(stmt ast.Statement) bool {
	switch stmt.(type) {
	case *ast.WhileStatement, *ast.DoWhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement:
		return true
	default:
		return false
	}
}

func canBreak(controls []controlTarget, label string) bool {
	for i := len(controls) - 1; i >= 0; i-- {
		if label == "" {
			if controls[i].allowsUnlabeledBreak {
				return true
			}
			continue
		}
		if controls[i].label == label && controls[i].allowsLabeledBreak {
			return true
		}
	}
	return false
}

func canContinue(controls []controlTarget, label string) bool {
	for i := len(controls) - 1; i >= 0; i-- {
		if label == "" {
			if controls[i].label == "" && controls[i].allowsContinue {
				return true
			}
			continue
		}
		if controls[i].label == label && controls[i].allowsContinue {
			return true
		}
	}
	return false
}

func validateCondition(expr ast.Expression, symbols map[string]symbol, functions map[string]functionSignature) error {
	kind, err := inferExpressionKind(expr, symbols, functions)
	if err != nil {
		return err
	}
	switch kind {
	case "number", "boolean", "string", "args_array", "array", "object", "dynamic":
		return nil
	default:
		return errorAt(expr, "value of type %s cannot be used as a condition", kind)
	}
}

func validateAssignment(stmt *ast.AssignmentStatement, symbols map[string]symbol, functions map[string]functionSignature) error {
	valueKind, err := inferExpressionKind(stmt.Value, symbols, functions)
	if err != nil {
		return err
	}
	switch target := stmt.Target.(type) {
	case *ast.Identifier:
		current, ok := symbols[target.Name]
		if !ok {
			return errorAt(target, "unknown identifier %s", target.Name)
		}
		if !current.mutable {
			return errorAt(target, "cannot reassign const %s", target.Name)
		}
		if current.kind == "dynamic" {
			if !isRuntimeValueKind(valueKind) {
				return errorAt(stmt, "cannot assign %s to %s", valueKind, current.kind)
			}
			return nil
		}
		if current.kind == "unknown" {
			if !isRuntimeValueKind(valueKind) && valueKind != "null" && valueKind != "undefined" && valueKind != "unknown" {
				return errorAt(stmt, "cannot assign %s to %s", valueKind, current.kind)
			}
			return nil
		}
		if current.kind != valueKind {
			return errorAt(stmt, "cannot assign %s to %s", valueKind, current.kind)
		}
		return nil
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		targetKind = runtimeShapeKind(targetKind)
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return errorAt(target, "member assignment requires an object target")
		}
		if !isRuntimeValueKind(valueKind) {
			return errorAt(stmt, "object properties currently support runtime values")
		}
		if ident, ok := target.Target.(*ast.Identifier); ok {
			if current, exists := symbols[ident.Name]; exists {
				if readonly, valueType := structuredMemberAssignmentRule(current.kind, target.Property); readonly {
					return errorAt(target, "cannot assign to readonly property %s", target.Property)
				} else if valueType != "" && !isExpressionAssignableToType(valueType, stmt.Value, valueKind, symbols, functions) {
					return errorAt(stmt, "cannot assign %s to %s", valueKind, valueType)
				}
			}
		}
		return nil
	case *ast.IndexExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		targetKind = runtimeShapeKind(targetKind)
		indexKind, err := inferExpressionKind(target.Index, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind == "array" {
			if indexKind != "number" {
				return errorAt(target, "array index must be a number")
			}
		} else if targetKind == "object" || targetKind == "function" {
			if indexKind != "string" && indexKind != "symbol" {
				return errorAt(target, "object property index must be a string or symbol")
			}
		} else if targetKind == "dynamic" {
			if indexKind != "number" && indexKind != "string" && indexKind != "symbol" && indexKind != "dynamic" {
				return errorAt(target, "dynamic index must be a number-like, string-like, or symbol value")
			}
		} else {
			return errorAt(target, "index assignment requires an array or object target")
		}
		if !isRuntimeValueKind(valueKind) {
			return errorAt(stmt, "indexed values currently support runtime values")
		}
		if ident, ok := target.Target.(*ast.Identifier); ok {
			if current, exists := symbols[ident.Name]; exists {
				readonly, expectedValueType := structuredIndexAssignmentRule(current.kind, indexKind)
				if readonly {
					return errorAt(target, "cannot assign through readonly index signature")
				}
				if expectedValueType != "" && !isExpressionAssignableToType(expectedValueType, stmt.Value, valueKind, symbols, functions) {
					return errorAt(stmt, "cannot assign %s to %s", valueKind, expectedValueType)
				}
			}
		}
		return nil
	default:
		return errorAt(stmt, "unsupported assignment target")
	}
}

func validateDelete(stmt *ast.DeleteStatement, symbols map[string]symbol, functions map[string]functionSignature) error {
	switch target := stmt.Target.(type) {
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return errorAt(target, "delete requires an object target")
		}
		return nil
	case *ast.IndexExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		indexKind, err := inferExpressionKind(target.Index, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return errorAt(target, "delete requires an object target")
		}
		if indexKind != "string" && indexKind != "dynamic" {
			return errorAt(target, "delete requires a string property key")
		}
		return nil
	default:
		return errorAt(stmt, "delete requires an object member or string-keyed property access")
	}
}

func inferExpressionKind(expr ast.Expression, symbols map[string]symbol, functions map[string]functionSignature) (string, error) {
	switch expr := expr.(type) {
	case nil:
		return "undefined", nil
	case *ast.NumberLiteral:
		return "number", nil
	case *ast.BigIntLiteral:
		return "bigint", nil
	case *ast.BooleanLiteral:
		return "boolean", nil
	case *ast.NullLiteral:
		return "null", nil
	case *ast.UndefinedLiteral:
		return "undefined", nil
	case *ast.ThisExpression:
		return "dynamic", nil
	case *ast.NewTargetExpression:
		return "dynamic", nil
	case *ast.AwaitExpression:
		if _, err := inferExpressionKind(expr.Value, symbols, functions); err != nil {
			return "", err
		}
		return "dynamic", nil
	case *ast.StringLiteral:
		return "string", nil
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Spread {
				kind, err := inferExpressionKind(property.Value, symbols, functions)
				if err != nil {
					return "", err
				}
				if kind != "object" && kind != "dynamic" && kind != "function" {
					return "", errorAt(expr, "object spread expects an object-like value")
				}
				continue
			}
			if property.Computed {
				keyKind, err := inferExpressionKind(property.KeyExpr, symbols, functions)
				if err != nil {
					return "", err
				}
				if keyKind != "string" && keyKind != "symbol" && keyKind != "dynamic" {
					return "", errorAt(expr, "computed object keys must be string-like or symbol values")
				}
			}
			if property.Getter && property.Setter {
				return "", errorAt(expr, "object accessor %s cannot be both getter and setter", property.Key)
			}
			if fn, ok := property.Value.(*ast.FunctionExpression); ok {
				if property.Getter && len(fn.Params) != 0 {
					return "", errorAt(expr, "object getter %s must not declare parameters", property.Key)
				}
				if property.Setter && len(fn.Params) != 1 {
					return "", errorAt(expr, "object setter %s must declare exactly one parameter", property.Key)
				}
			}
			kind, err := inferExpressionKind(property.Value, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", errorAt(expr, "object literal values currently support runtime values")
			}
		}
		return "object", nil
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if spread, ok := element.(*ast.SpreadExpression); ok {
				kind, err := inferExpressionKind(spread.Value, symbols, functions)
				if err != nil {
					return "", err
				}
				if kind != "array" && kind != "args_array" && kind != "dynamic" {
					return "", errorAt(spread, "array spread expects an array-like value")
				}
				continue
			}
			kind, err := inferExpressionKind(element, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", errorAt(expr, "array literal elements currently support runtime values")
			}
		}
		return "array", nil
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			kind, err := inferExpressionKind(value, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", errorAt(expr, "template interpolation expects runtime-compatible values")
			}
		}
		return "string", nil
	case *ast.SpreadExpression:
		return "", errorAt(expr, "spread expressions are only supported in array literals and call arguments")
	case *ast.Identifier:
		if value, ok := symbols[expr.Name]; ok {
			return value.kind, nil
		}
		if _, ok := functions[expr.Name]; ok {
			return "function", nil
		}
		return "", errorAt(expr, "unknown identifier %s", expr.Name)
	case *ast.ClosureExpression:
		return "function", nil
	case *ast.CastExpression:
		if _, err := inferExpressionKind(expr.Value, symbols, functions); err != nil {
			return "", err
		}
		if !isSupportedTypeAnnotation(expr.TypeAnnotation) {
			return "", errorAt(expr, "unsupported cast target %s", expr.TypeAnnotation)
		}
		kind := normalizeTypeAnnotation(expr.TypeAnnotation)
		if kind == "" {
			return "dynamic", nil
		}
		if kind == "never" {
			return "", errorAt(expr, "casts to never are not supported")
		}
		return kind, nil
	case *ast.BinaryExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		switch expr.Operator {
		case ast.OperatorBitAnd, ast.OperatorBitOr, ast.OperatorBitXor:
			return inferBitwiseBinaryKind(expr, leftKind, rightKind)
		case ast.OperatorShl, ast.OperatorShr, ast.OperatorUShr:
			return inferShiftExpressionKind(expr, leftKind, rightKind)
		}
		if expr.Operator == ast.OperatorAdd {
			if leftKind == "string" || rightKind == "string" {
				if isRuntimeValueKind(leftKind) && isRuntimeValueKind(rightKind) {
					return "string", nil
				}
				return "", errorAt(expr, "operator %s expects runtime-compatible string operands", expr.Operator)
			}
		}
		if (leftKind != "number" && leftKind != "dynamic") || (rightKind != "number" && rightKind != "dynamic") {
			return "", errorAt(expr, "operator %s expects number operands", expr.Operator)
		}
		return "number", nil
	case *ast.NullishCoalesceExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		if !isRuntimeValueKind(leftKind) || !isRuntimeValueKind(rightKind) {
			return "", errorAt(expr, "operator ?? expects runtime-compatible operands")
		}
		if leftKind == rightKind && leftKind != "dynamic" {
			return leftKind, nil
		}
		return "dynamic", nil
	case *ast.CommaExpression:
		if _, err := inferExpressionKind(expr.Left, symbols, functions); err != nil {
			return "", err
		}
		return inferExpressionKind(expr.Right, symbols, functions)
	case *ast.ConditionalExpression:
		conditionKind, err := inferExpressionKind(expr.Condition, symbols, functions)
		if err != nil {
			return "", err
		}
		if !isTruthyKind(conditionKind) {
			return "", errorAt(expr, "conditional operator expects a truthy-compatible condition")
		}
		consequentKind, err := inferExpressionKind(expr.Consequent, symbols, functions)
		if err != nil {
			return "", err
		}
		alternativeKind, err := inferExpressionKind(expr.Alternative, symbols, functions)
		if err != nil {
			return "", err
		}
		if !isRuntimeValueKind(consequentKind) || !isRuntimeValueKind(alternativeKind) {
			return "", errorAt(expr, "conditional operator expects runtime-compatible branch values")
		}
		if consequentKind == alternativeKind && consequentKind != "dynamic" {
			return consequentKind, nil
		}
		return "dynamic", nil
	case *ast.UnaryExpression:
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		switch expr.Operator {
		case ast.OperatorNot:
			if !isTruthyKind(rightKind) {
				return "", errorAt(expr, "operator ! expects a truthy-compatible operand")
			}
			return "boolean", nil
		case ast.OperatorBitNot:
			switch rightKind {
			case "number":
				return "number", nil
			case "bigint":
				return "bigint", nil
			case "dynamic":
				return "dynamic", nil
			default:
				return "", errorAt(expr, "operator ~ expects number or bigint operands")
			}
		default:
			return "", errorAt(expr, "unsupported unary operator")
		}
	case *ast.TypeofExpression:
		if _, err := inferExpressionKind(expr.Value, symbols, functions); err != nil {
			return "", err
		}
		return "string", nil
	case *ast.TypeCheckExpression:
		if _, err := inferExpressionKind(expr.Value, symbols, functions); err != nil {
			return "", err
		}
		if !isSupportedTypeAnnotation(expr.TypeAnnotation) {
			return "", errorAt(expr, "unsupported type annotation %q in runtime type check", expr.TypeAnnotation)
		}
		return "boolean", nil
	case *ast.InstanceofExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		if leftKind != "object" && leftKind != "array" && leftKind != "dynamic" {
			return "", errorAt(expr, "instanceof expects an object-like left operand")
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		if rightKind != "function" && rightKind != "dynamic" {
			return "", errorAt(expr, "instanceof expects a function-like right operand")
		}
		return "boolean", nil
	case *ast.LogicalExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		if !isTruthyKind(leftKind) || !isTruthyKind(rightKind) {
			return "", errorAt(expr, "logical operator %s expects truthy-compatible operands", expr.Operator)
		}
		return "boolean", nil
	case *ast.ComparisonExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		switch expr.Operator {
		case ast.OperatorEq, ast.OperatorNe, ast.OperatorStrictEq, ast.OperatorStrictNe:
			if leftKind == "args_array" || rightKind == "args_array" {
				return "", errorAt(expr, "operator %s does not support args arrays directly", expr.Operator)
			}
			if !((leftKind == rightKind) || (leftKind == "dynamic" || rightKind == "dynamic")) {
				return "", errorAt(expr, "operator %s expects comparable operands", expr.Operator)
			}
		default:
			if (leftKind != "number" && leftKind != "dynamic") || (rightKind != "number" && rightKind != "dynamic") {
				return "", errorAt(expr, "operator %s expects number operands", expr.Operator)
			}
		}
		return "boolean", nil
	case *ast.IndexExpression:
		targetKind, err := inferExpressionKind(expr.Target, symbols, functions)
		if err != nil {
			return "", err
		}
		targetKind = runtimeShapeKind(targetKind)
		indexKind, err := inferExpressionKind(expr.Index, symbols, functions)
		if err != nil {
			return "", err
		}
		switch targetKind {
		case "args_array":
			if indexKind != "number" {
				return "", errorAt(expr, "array index must be a number")
			}
			return "string", nil
		case "array":
			if indexKind != "number" {
				return "", errorAt(expr, "array index must be a number")
			}
			return "dynamic", nil
		case "object", "function":
			if indexKind != "string" && indexKind != "symbol" {
				return "", errorAt(expr, "object property index must be a string or symbol")
			}
			return "dynamic", nil
		case "dynamic":
			if indexKind != "number" && indexKind != "string" && indexKind != "symbol" && indexKind != "dynamic" {
				return "", errorAt(expr, "dynamic index must be a number-like, string-like, or symbol value")
			}
			return "dynamic", nil
		default:
			return "", errorAt(expr, "indexing is currently only supported for args, arrays, and objects")
		}
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(expr.Target, symbols, functions)
		if err != nil {
			return "", err
		}
		if propertyType := structuredMemberType(targetKind, expr.Property); propertyType != "" {
			return propertyType, nil
		}
		targetKind = runtimeShapeKind(targetKind)
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" && targetKind != "array" && targetKind != "args_array" && targetKind != "string" {
			if expr.Property == "length" && (targetKind == "array" || targetKind == "args_array" || targetKind == "string") {
				return "number", nil
			}
			return "", errorAt(expr, "member access requires an object target")
		}
		if expr.Property == "length" && (targetKind == "array" || targetKind == "args_array" || targetKind == "string") {
			return "number", nil
		}
		return "dynamic", nil
	case *ast.InvokeExpression:
		if ident, ok := expr.Callee.(*ast.Identifier); ok {
			if fn, exists := functions[ident.Name]; exists {
				return validateFunctionArguments(ident.Name, fn, expr.Arguments, symbols, functions, expr)
			}
		}
		calleeKind, err := inferExpressionKind(expr.Callee, symbols, functions)
		if err != nil {
			return "", err
		}
		calleeKind = runtimeShapeKind(calleeKind)
		if calleeKind != "function" && calleeKind != "dynamic" {
			return "", errorAt(expr, "invocation requires a function-like value")
		}
		for _, arg := range expr.Arguments {
			kind, err := inferExpressionKind(arg, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", errorAt(expr, "invocation expects runtime-compatible arguments")
			}
		}
		return "dynamic", nil
	case *ast.CallExpression:
		return validateCallExpression(expr, symbols, functions)
	case *ast.NewExpression:
		for _, arg := range expr.Arguments {
			if _, err := inferExpressionKind(arg, symbols, functions); err != nil {
				return "", err
			}
		}
		return "dynamic", nil
	default:
		return "", errorAt(expr, "unsupported expression")
	}
}

func validateCallExpression(call *ast.CallExpression, symbols map[string]symbol, functions map[string]functionSignature) (string, error) {
	switch call.Callee {
	case "print":
		if len(call.Arguments) == 0 {
			return "", errorAt(call, "print expects at least 1 argument")
		}
		for _, arg := range call.Arguments {
			kind, err := inferExpressionKind(arg, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isPrintableKind(kind) {
				return "", errorAt(call, "print expects printable values")
			}
		}
		return "void", nil
	case "__jayess_console_log", "__jayess_console_warn", "__jayess_console_error":
		for _, arg := range call.Arguments {
			kind, err := inferExpressionKind(arg, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isPrintableKind(kind) {
				return "", errorAt(call, "%s expects printable values", call.Callee)
			}
		}
		return "void", nil
	case "__jayess_process_cwd":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_cwd expects 0 arguments")
		}
		return "dynamic", nil
	case "__jayess_process_env":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_process_env expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_process_argv":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_argv expects 0 arguments")
		}
		return "dynamic", nil
	case "__jayess_process_platform":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_platform expects 0 arguments")
		}
		return "string", nil
	case "__jayess_process_arch":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_arch expects 0 arguments")
		}
		return "string", nil
	case "__jayess_process_tmpdir":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_tmpdir expects 0 arguments")
		}
		return "string", nil
	case "__jayess_process_hostname":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_hostname expects 0 arguments")
		}
		return "string", nil
	case "__jayess_process_uptime":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_uptime expects 0 arguments")
		}
		return "number", nil
	case "__jayess_process_hrtime":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_hrtime expects 0 arguments")
		}
		return "number", nil
	case "__jayess_process_cpu_info", "__jayess_process_memory_info", "__jayess_process_user_info":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_process_thread_pool_size":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_process_thread_pool_size expects 0 arguments")
		}
		return "number", nil
	case "__jayess_process_exit":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_process_exit expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_path_join":
		return "string", nil
	case "__jayess_path_normalize":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_path_normalize expects 1 argument")
		}
		return "string", nil
	case "__jayess_path_resolve":
		return "string", nil
	case "__jayess_path_relative":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_path_relative expects 2 arguments")
		}
		return "string", nil
	case "__jayess_path_parse":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_path_parse expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_path_is_absolute":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_path_is_absolute expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_path_format":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_path_format expects 1 argument")
		}
		return "string", nil
	case "__jayess_path_sep", "__jayess_path_delimiter":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "string", nil
	case "__jayess_tls_is_available", "__jayess_https_is_available":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_tls_backend", "__jayess_https_backend":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "string", nil
	case "__jayess_tls_connect":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_tls_create_server":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_https_create_server":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_path_basename", "__jayess_path_dirname", "__jayess_path_extname":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "string", nil
	case "__jayess_url_parse", "__jayess_querystring_parse":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_dns_lookup", "__jayess_dns_lookup_all", "__jayess_dns_reverse", "__jayess_dns_set_resolver":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_dns_clear_resolver":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_child_process_exec", "__jayess_child_process_spawn", "__jayess_child_process_kill", "__jayess_worker_create":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_atomics_load":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_atomics_load expects 2 arguments")
		}
		return "number", nil
	case "__jayess_atomics_store", "__jayess_atomics_add", "__jayess_atomics_sub", "__jayess_atomics_and", "__jayess_atomics_or", "__jayess_atomics_xor", "__jayess_atomics_exchange":
		if len(call.Arguments) != 3 {
			return "", errorAt(call, "%s expects 3 arguments", call.Callee)
		}
		return "number", nil
	case "__jayess_atomics_compareExchange":
		if len(call.Arguments) != 4 {
			return "", errorAt(call, "__jayess_atomics_compareExchange expects 4 arguments")
		}
		return "number", nil
	case "__jayess_crypto_random_bytes":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_crypto_random_bytes expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_crypto_hash":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_crypto_hash expects 2 arguments")
		}
		return "string", nil
	case "__jayess_crypto_hmac":
		if len(call.Arguments) != 3 {
			return "", errorAt(call, "__jayess_crypto_hmac expects 3 arguments")
		}
		return "string", nil
	case "__jayess_crypto_secure_compare":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_crypto_secure_compare expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_crypto_encrypt", "__jayess_crypto_decrypt":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_crypto_generate_key_pair", "__jayess_crypto_public_encrypt", "__jayess_crypto_private_decrypt", "__jayess_crypto_sign", "__jayess_crypto_verify":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_compression_gzip", "__jayess_compression_gunzip", "__jayess_compression_deflate", "__jayess_compression_inflate", "__jayess_compression_brotli", "__jayess_compression_unbrotli":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_compression_create_gzip_stream", "__jayess_compression_create_gunzip_stream", "__jayess_compression_create_deflate_stream", "__jayess_compression_create_inflate_stream", "__jayess_compression_create_brotli_stream", "__jayess_compression_create_unbrotli_stream":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_http_parse_request", "__jayess_http_parse_response", "__jayess_http_request", "__jayess_http_create_server", "__jayess_http_request_stream", "__jayess_http_request_stream_async", "__jayess_http_get", "__jayess_http_get_stream", "__jayess_http_get_stream_async", "__jayess_http_request_async", "__jayess_http_get_async", "__jayess_https_request", "__jayess_https_request_stream", "__jayess_https_request_stream_async", "__jayess_https_get", "__jayess_https_get_stream", "__jayess_https_get_stream_async", "__jayess_https_request_async", "__jayess_https_get_async":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_net_is_ip":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_net_is_ip expects 1 argument")
		}
		return "number", nil
	case "__jayess_net_create_datagram_socket":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_net_create_datagram_socket expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_net_connect":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_net_connect expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_net_listen":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_net_listen expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_url_format", "__jayess_querystring_stringify":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "string", nil
	case "__jayess_http_format_request", "__jayess_http_format_response":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "string", nil
	case "__jayess_fs_read_file":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_read_file expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_read_file_async":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_read_file_async expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_write_file":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_write_file expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_append_file":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_append_file expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_write_file_async":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_write_file_async expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_create_read_stream", "__jayess_fs_create_write_stream":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_fs_exists":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_fs_exists expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_fs_remove":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 1 or 2 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_fs_read_dir":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_read_dir expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_stat":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_fs_stat expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_fs_mkdir":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_mkdir expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_copy_file", "__jayess_fs_copy_dir", "__jayess_fs_rename", "__jayess_fs_symlink":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_fs_watch":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_fs_watch expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_timers_sleep":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_timers_sleep expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_process_on_signal", "__jayess_process_once_signal":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_process_off_signal":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_process_off_signal expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_process_raise":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_process_raise expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_timers_set_timeout":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_timers_set_timeout expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_timers_clear_timeout":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_timers_clear_timeout expects 1 argument")
		}
		return "dynamic", nil
	case "readLine", "readKey":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if kind != "string" {
			return "", errorAt(call, "%s expects a string prompt", call.Callee)
		}
		return "string", nil
	case "sleep":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "sleep expects 1 argument")
		}
		kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if kind != "number" && kind != "dynamic" {
			return "", errorAt(call, "sleep expects a number argument")
		}
		return "void", nil
	case "compile", "compileFile":
		if len(call.Arguments) < 1 || len(call.Arguments) > 2 {
			return "", errorAt(call, "%s expects 1 or 2 arguments", call.Callee)
		}
		sourceKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if sourceKind != "string" && sourceKind != "dynamic" {
			if call.Callee == "compileFile" {
				return "", errorAt(call, "compileFile expects a string input path")
			}
			return "", errorAt(call, "compile expects a string source argument")
		}
		if len(call.Arguments) == 2 {
			outputKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
			if err != nil {
				return "", err
			}
			if outputKind != "string" && outputKind != "object" && outputKind != "dynamic" {
				return "", errorAt(call, "%s expects a string output path or options object", call.Callee)
			}
		}
		return "dynamic", nil
	case "sleepAsync":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "sleepAsync expects 1 or 2 arguments")
		}
		delayKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if delayKind != "number" && delayKind != "dynamic" {
			return "", errorAt(call, "sleepAsync expects a number delay")
		}
		return "dynamic", nil
	case "setTimeout":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "setTimeout expects 2 arguments")
		}
		callbackKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if callbackKind != "function" && callbackKind != "dynamic" {
			return "", errorAt(call, "setTimeout expects a function callback")
		}
		delayKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if delayKind != "number" && delayKind != "dynamic" {
			return "", errorAt(call, "setTimeout expects a number delay")
		}
		return "number", nil
	case "clearTimeout":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "clearTimeout expects 1 argument")
		}
		idKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if idKind != "number" && idKind != "dynamic" {
			return "", errorAt(call, "clearTimeout expects a number timer id")
		}
		return "undefined", nil
	case "__jayess_apply":
		if len(call.Arguments) != 3 {
			return "", errorAt(call, "__jayess_apply expects 3 arguments")
		}
		calleeKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if calleeKind != "function" && calleeKind != "dynamic" {
			return "", errorAt(call, "__jayess_apply expects a function-like value")
		}
		if _, err := inferExpressionKind(call.Arguments[1], symbols, functions); err != nil {
			return "", err
		}
		argsKind, err := inferExpressionKind(call.Arguments[2], symbols, functions)
		if err != nil {
			return "", err
		}
		if argsKind != "array" && argsKind != "args_array" && argsKind != "dynamic" {
			return "", errorAt(call, "__jayess_apply expects an array-like third argument")
		}
		return "dynamic", nil
	case "__jayess_bind":
		if len(call.Arguments) != 3 {
			return "", errorAt(call, "__jayess_bind expects 3 arguments")
		}
		calleeKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if calleeKind != "function" && calleeKind != "dynamic" {
			return "", errorAt(call, "__jayess_bind expects a function-like value")
		}
		if _, err := inferExpressionKind(call.Arguments[1], symbols, functions); err != nil {
			return "", err
		}
		argsKind, err := inferExpressionKind(call.Arguments[2], symbols, functions)
		if err != nil {
			return "", err
		}
		if argsKind != "array" && argsKind != "dynamic" {
			return "", errorAt(call, "__jayess_bind expects an array-like third argument")
		}
		return "function", nil
	case "__jayess_constructor_return":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_constructor_return expects 2 arguments")
		}
		selfKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if selfKind != "object" && selfKind != "dynamic" {
			return "", errorAt(call, "__jayess_constructor_return expects an object-like first argument")
		}
		return inferExpressionKind(call.Arguments[1], symbols, functions)
	case "__jayess_array_push":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_array_push expects 2 arguments")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" {
			return "", errorAt(call, "__jayess_array_push expects an array-like first argument")
		}
		valueKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if !isRuntimeValueKind(valueKind) {
			return "", errorAt(call, "__jayess_array_push expects a runtime-compatible value")
		}
		return "number", nil
	case "__jayess_array_pop":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_array_pop expects 1 argument")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" {
			return "", errorAt(call, "__jayess_array_pop expects an array-like first argument")
		}
		return "dynamic", nil
	case "__jayess_array_shift":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_array_shift expects 1 argument")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" {
			return "", errorAt(call, "__jayess_array_shift expects an array-like first argument")
		}
		return "dynamic", nil
	case "__jayess_array_unshift":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_array_unshift expects 2 arguments")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" {
			return "", errorAt(call, "__jayess_array_unshift expects an array-like first argument")
		}
		valueKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if !isRuntimeValueKind(valueKind) {
			return "", errorAt(call, "__jayess_array_unshift expects a runtime-compatible value")
		}
		return "number", nil
	case "__jayess_array_slice":
		if len(call.Arguments) != 3 {
			return "", errorAt(call, "__jayess_array_slice expects 3 arguments")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" && targetKind != "args_array" {
			return "", errorAt(call, "__jayess_array_slice expects an array-like first argument")
		}
		return "dynamic", nil
	case "__jayess_object_keys":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_object_keys expects 1 argument")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return "", errorAt(call, "__jayess_object_keys expects an object-like argument")
		}
		return "dynamic", nil
	case "__jayess_object_values", "__jayess_object_entries":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return "", errorAt(call, "%s expects an object-like argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_object_symbols":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_object_symbols expects 1 argument")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return "", errorAt(call, "__jayess_object_symbols expects an object-like argument")
		}
		return "dynamic", nil
	case "__jayess_object_from_entries":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_object_from_entries expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_object_assign":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_object_assign expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_object_has_own":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_object_has_own expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_object_rest":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_object_rest expects 2 arguments")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "object" && targetKind != "dynamic" {
			return "", errorAt(call, "__jayess_object_rest expects an object-like first argument")
		}
		keysKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if keysKind != "array" && keysKind != "dynamic" {
			return "", errorAt(call, "__jayess_object_rest expects an array-like second argument")
		}
		return "dynamic", nil
	case "__jayess_std_map_new", "__jayess_std_set_new", "__jayess_std_weak_map_new", "__jayess_std_weak_set_new":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_std_symbol", "Symbol":
		if len(call.Arguments) > 1 {
			return "", errorAt(call, "%s expects at most 1 argument", call.Callee)
		}
		if len(call.Arguments) == 1 {
			kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", errorAt(call, "%s expects a runtime-compatible description", call.Callee)
			}
		}
		return "symbol", nil
	case "__jayess_std_symbol_for":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_std_symbol_for expects 1 argument")
		}
		if _, err := inferExpressionKind(call.Arguments[0], symbols, functions); err != nil {
			return "", err
		}
		return "symbol", nil
	case "__jayess_std_symbol_key_for":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_std_symbol_key_for expects 1 argument")
		}
		if _, err := inferExpressionKind(call.Arguments[0], symbols, functions); err != nil {
			return "", err
		}
		return "dynamic", nil
	case "__jayess_std_symbol_iterator", "__jayess_std_symbol_async_iterator", "__jayess_std_symbol_to_string_tag", "__jayess_std_symbol_has_instance", "__jayess_std_symbol_species", "__jayess_std_symbol_match", "__jayess_std_symbol_replace", "__jayess_std_symbol_search", "__jayess_std_symbol_split", "__jayess_std_symbol_to_primitive":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "symbol", nil
	case "__jayess_std_date_new":
		if len(call.Arguments) > 1 {
			return "", errorAt(call, "__jayess_std_date_new expects at most 1 argument")
		}
		return "dynamic", nil
	case "__jayess_std_regexp_new":
		if len(call.Arguments) > 2 {
			return "", errorAt(call, "__jayess_std_regexp_new expects at most 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_std_error_new":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_std_error_new expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_std_aggregate_error_new":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_std_aggregate_error_new expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_std_array_buffer_new", "__jayess_shared_array_buffer_new", "__jayess_std_int8_array_new", "__jayess_std_uint8_array_new", "__jayess_std_uint16_array_new", "__jayess_std_int16_array_new", "__jayess_std_uint32_array_new", "__jayess_std_int32_array_new", "__jayess_std_float32_array_new", "__jayess_std_float64_array_new", "__jayess_std_data_view_new", "__jayess_std_iterator_from", "__jayess_std_async_iterator_from", "__jayess_std_promise_resolve", "__jayess_std_promise_reject", "__jayess_std_promise_all", "__jayess_std_promise_race", "__jayess_std_promise_all_settled", "__jayess_std_promise_any", "__jayess_await":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_std_uint8_array_from_string":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_std_uint8_array_from_string expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_std_uint8_array_concat":
		return "dynamic", nil
	case "__jayess_std_uint8_array_equals":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_std_uint8_array_equals expects 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_std_uint8_array_compare":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_std_uint8_array_compare expects 2 arguments")
		}
		return "number", nil
	case "__jayess_std_date_now":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_std_date_now expects 0 arguments")
		}
		return "number", nil
	case "__jayess_std_json_stringify":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_std_json_stringify expects 1 argument")
		}
		return "string", nil
	case "__jayess_std_json_parse":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_std_json_parse expects 1 argument")
		}
		textKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if textKind != "string" && textKind != "dynamic" {
			return "", errorAt(call, "__jayess_std_json_parse expects a string-like argument")
		}
		return "dynamic", nil
	case "__jayess_iter_values":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_iter_values expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_number_is_nan", "__jayess_number_is_finite":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "dynamic", nil
	case "__jayess_string_from_char_code":
		return "string", nil
	case "__jayess_array_is_array":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_array_is_array expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_array_from":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "__jayess_array_from expects 1 argument")
		}
		return "dynamic", nil
	case "__jayess_array_of":
		return "dynamic", nil
	case "__jayess_array_map", "__jayess_array_filter", "__jayess_array_find":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" && targetKind != "args_array" {
			return "", errorAt(call, "%s expects an array-like first argument", call.Callee)
		}
		callbackKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if callbackKind != "function" && callbackKind != "dynamic" {
			return "", errorAt(call, "%s expects a function-like callback", call.Callee)
		}
		if call.Callee == "__jayess_array_find" {
			return "dynamic", nil
		}
		return "dynamic", nil
	case "__jayess_math_floor", "__jayess_math_ceil", "__jayess_math_round", "__jayess_math_abs", "__jayess_math_sqrt":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "number", nil
	case "__jayess_math_min", "__jayess_math_max", "__jayess_math_pow":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
		}
		return "number", nil
	case "__jayess_math_random":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_math_random expects 0 arguments")
		}
		return "number", nil
	case "__jayess_array_for_each":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_array_for_each expects 2 arguments")
		}
		targetKind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "array" && targetKind != "dynamic" && targetKind != "args_array" {
			return "", errorAt(call, "__jayess_array_for_each expects an array-like first argument")
		}
		callbackKind, err := inferExpressionKind(call.Arguments[1], symbols, functions)
		if err != nil {
			return "", err
		}
		if callbackKind != "function" && callbackKind != "dynamic" {
			return "", errorAt(call, "__jayess_array_for_each expects a function-like callback")
		}
		return "undefined", nil
	case "__jayess_current_this":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "__jayess_current_this expects 0 arguments")
		}
		return "dynamic", nil
	default:
		fn, ok := functions[call.Callee]
		if !ok {
			if symbol, exists := symbols[call.Callee]; exists && (symbol.kind == "function" || symbol.kind == "dynamic") {
				for _, arg := range call.Arguments {
					kind, err := inferExpressionKind(arg, symbols, functions)
					if err != nil {
						return "", err
					}
					if !isRuntimeValueKind(kind) {
						return "", errorAt(call, "invocation expects runtime-compatible arguments")
					}
				}
				return "dynamic", nil
			}
			return "", errorAt(call, "unknown function %s", call.Callee)
		}
		return validateFunctionArguments(call.Callee, fn, call.Arguments, symbols, functions, call)
	}
}

func validateFunctionArguments(name string, fn functionSignature, arguments []ast.Expression, symbols map[string]symbol, functions map[string]functionSignature, node any) (string, error) {
	minArgs := fn.minArgs
	if !fn.variadic && !hasSpreadArguments(arguments) {
		if fn.hasRest {
			if len(arguments) < minArgs {
				return "", errorAt(node, "function %s expects at least %d arguments", name, minArgs)
			}
		} else if len(arguments) < minArgs || len(arguments) > fn.paramCount {
			return "", errorAt(node, "function %s expects %d arguments", name, fn.paramCount)
		}
	}
	for index, arg := range arguments {
		if spread, ok := arg.(*ast.SpreadExpression); ok {
			kind, err := inferExpressionKind(spread.Value, symbols, functions)
			if err != nil {
				return "", err
			}
			if kind != "array" && kind != "args_array" && kind != "dynamic" {
				return "", errorAt(spread, "spread arguments expect an array-like value")
			}
			continue
		}
		kind, err := inferExpressionKind(arg, symbols, functions)
		if err != nil {
			return "", err
		}
		if !isRuntimeValueKind(kind) {
			return "", errorAt(node, "function %s expects runtime-compatible arguments", name)
		}
		if index < len(fn.paramTypes) && fn.paramTypes[index] != "" && !isAssignableTo(fn.paramTypes[index], kind) {
			return "", errorAt(arg, "argument %d for %s expects %s, got %s", index+1, name, fn.paramTypes[index], kind)
		}
	}
	if fn.isExtern {
		return "dynamic", nil
	}
	if fn.isAsync {
		return "dynamic", nil
	}
	if fn.returnType != "" {
		return fn.returnType, nil
	}
	return "dynamic", nil
}

func validateLoopStatement(stmt ast.Statement, symbols map[string]symbol, functions map[string]functionSignature) error {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		if err := validateVariableAnnotation(stmt); err != nil {
			return err
		}
		kind, err := inferExpressionKind(stmt.Value, symbols, functions)
		if err != nil {
			return err
		}
		if expected := normalizeTypeAnnotation(stmt.TypeAnnotation); expected != "" {
			if !isExpressionAssignableToType(expected, stmt.Value, kind, symbols, functions) {
				return errorAt(stmt, "cannot initialize %s variable %s with %s", expected, stmt.Name, kind)
			}
			kind = expected
		} else if stmt.Kind == ast.DeclarationVar {
			kind = "dynamic"
		}
		symbols[stmt.Name] = symbol{kind: kind, mutable: stmt.Kind != ast.DeclarationConst}
		return nil
	case *ast.AssignmentStatement:
		return validateAssignment(stmt, symbols, functions)
	case *ast.DeleteStatement:
		return validateDelete(stmt, symbols, functions)
	case *ast.ExpressionStatement:
		_, err := inferExpressionKind(stmt.Expression, symbols, functions)
		return err
	default:
		return errorAt(stmt, "unsupported for-loop clause")
	}
}

func isRuntimeValueKind(kind string) bool {
	switch kind {
	case "string", "number", "bigint", "boolean", "symbol", "object", "array", "dynamic", "null", "undefined", "unknown":
		return true
	case "function":
		return true
	default:
		return false
	}
}

func isPrintableKind(kind string) bool {
	switch kind {
	case "string", "number", "bigint", "boolean", "symbol", "dynamic", "array", "object", "args_array", "function", "null", "undefined":
		return true
	default:
		return false
	}
}

func isTruthyKind(kind string) bool {
	switch kind {
	case "number", "bigint", "boolean", "string", "symbol", "args_array", "array", "object", "dynamic", "function":
		return true
	default:
		return false
	}
}

func inferBitwiseBinaryKind(expr ast.Expression, leftKind string, rightKind string) (string, error) {
	if leftKind == "number" && rightKind == "bigint" || leftKind == "bigint" && rightKind == "number" {
		return "", errorAt(expr, "cannot mix number and bigint in bitwise expressions")
	}
	if leftKind == "bigint" && rightKind == "bigint" {
		return "bigint", nil
	}
	if leftKind == "number" && rightKind == "number" {
		return "number", nil
	}
	if leftKind == "dynamic" || rightKind == "dynamic" {
		if leftKind == "bigint" || rightKind == "bigint" {
			return "dynamic", nil
		}
		if isBitwiseNumberLike(leftKind) && isBitwiseNumberLike(rightKind) {
			return "number", nil
		}
		return "dynamic", nil
	}
	if isBitwiseNumberLike(leftKind) && isBitwiseNumberLike(rightKind) {
		return "number", nil
	}
	return "", errorAt(expr, "operator expects number or bigint operands")
}

func inferShiftExpressionKind(expr ast.Expression, leftKind string, rightKind string) (string, error) {
	if exprNode, ok := expr.(*ast.BinaryExpression); ok && exprNode.Operator == ast.OperatorUShr {
		if leftKind == "bigint" || rightKind == "bigint" {
			return "", errorAt(expr, "operator >>> does not support bigint operands")
		}
	}
	if leftKind == "number" && rightKind == "bigint" || leftKind == "bigint" && rightKind == "number" {
		return "", errorAt(expr, "cannot mix number and bigint in bitwise expressions")
	}
	if leftKind == "bigint" && rightKind == "bigint" {
		return "bigint", nil
	}
	if leftKind == "number" && rightKind == "number" {
		return "number", nil
	}
	if leftKind == "dynamic" || rightKind == "dynamic" {
		if leftKind == "bigint" || rightKind == "bigint" {
			return "dynamic", nil
		}
		if isBitwiseNumberLike(leftKind) && isBitwiseNumberLike(rightKind) {
			return "number", nil
		}
		return "dynamic", nil
	}
	if isBitwiseNumberLike(leftKind) && isBitwiseNumberLike(rightKind) {
		return "number", nil
	}
	return "", errorAt(expr, "shift operators expect number or bigint operands")
}

func isBitwiseNumberLike(kind string) bool {
	switch kind {
	case "number", "boolean", "dynamic":
		return true
	default:
		return false
	}
}

func isExpressionAssignableToType(expected string, expr ast.Expression, actualKind string, symbols map[string]symbol, functions map[string]functionSignature) bool {
	if actualKind == "dynamic" {
		if specific := structuredExpressionType(expr, symbols, functions); specific != "" {
			actualKind = specific
		}
	}
	if !isAssignableTo(expected, actualKind) {
		return false
	}
	structured, err := typesys.Parse(expected)
	if err != nil {
		return true
	}
	switch structured.Kind {
	case typesys.KindLiteral:
		return matchesLiteralType(structured.Name, expr)
	case typesys.KindUnion:
		for _, member := range structured.Elements {
			if isExpressionAssignableToType(member.String(), expr, actualKind, symbols, functions) {
				return true
			}
		}
		return false
	case typesys.KindIntersection:
		for _, member := range structured.Elements {
			if !isExpressionAssignableToType(member.String(), expr, actualKind, symbols, functions) {
				return false
			}
		}
		return true
	case typesys.KindTuple:
		array, ok := expr.(*ast.ArrayLiteral)
		if !ok {
			return true
		}
		if len(array.Elements) != len(structured.Elements) {
			return false
		}
		for i, element := range array.Elements {
			kind, err := inferExpressionKind(element, symbols, functions)
			if err != nil || !isExpressionAssignableToType(structured.Elements[i].String(), element, kind, symbols, functions) {
				return false
			}
		}
	case typesys.KindObject:
		object, ok := expr.(*ast.ObjectLiteral)
		if !ok {
			return true
		}
		for _, property := range structured.Properties {
			found := false
			for _, actual := range object.Properties {
				if actual.Computed || actual.Spread || actual.Key != property.Name {
					continue
				}
				found = true
				kind, err := inferExpressionKind(actual.Value, symbols, functions)
				if err != nil || !isExpressionAssignableToType(property.Type.String(), actual.Value, kind, symbols, functions) {
					return false
				}
				break
			}
			if !found && !property.Optional {
				return false
			}
		}
		for _, actual := range object.Properties {
			if actual.Computed || actual.Spread {
				continue
			}
			matchedProperty := false
			for _, property := range structured.Properties {
				if property.Name == actual.Key {
					matchedProperty = true
					break
				}
			}
			if matchedProperty {
				continue
			}
			if len(structured.IndexSignatures) == 0 {
				continue
			}
			indexKind := "string"
			valid := false
			for _, signature := range structured.IndexSignatures {
				if isAssignableTo(signature.KeyType.String(), indexKind) {
					kind, err := inferExpressionKind(actual.Value, symbols, functions)
					if err == nil && isExpressionAssignableToType(signature.ValueType.String(), actual.Value, kind, symbols, functions) {
						valid = true
						break
					}
				}
			}
			if !valid {
				return false
			}
		}
	case typesys.KindFunction:
		return actualKind == "function" || actualKind == expected
	}
	return true
}

func runtimeShapeKind(kind string) string {
	if kind == "dynamic" {
		return "dynamic"
	}
	normalized := normalizeTypeAnnotation(kind)
	if normalized == "" {
		return kind
	}
	expr, err := typesys.Parse(normalized)
	if err != nil {
		return normalized
	}
	switch expr.Kind {
	case typesys.KindLiteral:
		switch expr.Name {
		case "true", "false":
			return "boolean"
		default:
			if strings.HasPrefix(expr.Name, "\"") {
				return "string"
			}
			return "number"
		}
	case typesys.KindUnion:
		if len(expr.Elements) == 0 {
			return normalized
		}
		shape := runtimeShapeKind(expr.Elements[0].String())
		for _, element := range expr.Elements[1:] {
			if runtimeShapeKind(element.String()) != shape {
				return normalized
			}
		}
		return shape
	case typesys.KindIntersection:
		if len(expr.Elements) == 0 {
			return normalized
		}
		shape := runtimeShapeKind(expr.Elements[0].String())
		for _, element := range expr.Elements[1:] {
			if runtimeShapeKind(element.String()) != shape {
				return normalized
			}
		}
		return shape
	case typesys.KindTuple:
		return "array"
	case typesys.KindObject:
		return "object"
	case typesys.KindFunction:
		return "function"
	default:
		return normalized
	}
}

func applyConditionNarrowing(condition ast.Expression, symbols map[string]symbol, functions map[string]functionSignature, truthy bool) {
	if typeCheck, ok := condition.(*ast.TypeCheckExpression); ok {
		if truthy {
			applyTypeCheckNarrowing(typeCheck, symbols)
		}
		return
	}
	comparison, ok := condition.(*ast.ComparisonExpression)
	if !ok {
		return
	}
	switch comparison.Operator {
	case ast.OperatorEq, ast.OperatorStrictEq:
		applyComparisonNarrowing(comparison, symbols, functions, truthy)
	case ast.OperatorNe, ast.OperatorStrictNe:
		applyComparisonNarrowing(comparison, symbols, functions, !truthy)
	}
}

func applyTypeCheckNarrowing(typeCheck *ast.TypeCheckExpression, symbols map[string]symbol) {
	target, ok := typeCheck.Value.(*ast.Identifier)
	if !ok {
		return
	}
	current, ok := symbols[target.Name]
	if !ok {
		return
	}
	narrowed := normalizeTypeAnnotation(typeCheck.TypeAnnotation)
	if narrowed == "" {
		return
	}
	symbols[target.Name] = symbol{kind: narrowed, mutable: current.mutable}
}

func applyComparisonNarrowing(comparison *ast.ComparisonExpression, symbols map[string]symbol, functions map[string]functionSignature, equalMatch bool) {
	leftMember, leftOK := comparison.Left.(*ast.MemberExpression)
	rightLiteral, rightLiteralOK := literalTypeFromExpression(comparison.Right)
	if leftOK && rightLiteralOK {
		applyMemberDiscriminatorNarrowing(leftMember, rightLiteral, symbols, equalMatch)
		return
	}
	rightMember, rightOK := comparison.Right.(*ast.MemberExpression)
	leftLiteral, leftLiteralOK := literalTypeFromExpression(comparison.Left)
	if rightOK && leftLiteralOK {
		applyMemberDiscriminatorNarrowing(rightMember, leftLiteral, symbols, equalMatch)
	}
}

func applySwitchNarrowing(discriminant ast.Expression, test ast.Expression, symbols map[string]symbol) {
	member, ok := discriminant.(*ast.MemberExpression)
	if !ok {
		return
	}
	literal, ok := literalTypeFromExpression(test)
	if !ok {
		return
	}
	applyMemberDiscriminatorNarrowing(member, literal, symbols, true)
}

func applyDefaultSwitchNarrowing(discriminant ast.Expression, excluded []string, symbols map[string]symbol) {
	member, ok := discriminant.(*ast.MemberExpression)
	if !ok {
		return
	}
	target, ok := member.Target.(*ast.Identifier)
	if !ok {
		return
	}
	current, ok := symbols[target.Name]
	if !ok {
		return
	}
	narrowed := excludeUnionDiscriminatorLiterals(current.kind, member.Property, excluded)
	if narrowed != "" {
		symbols[target.Name] = symbol{kind: narrowed, mutable: current.mutable}
	}
}

func applyMemberDiscriminatorNarrowing(member *ast.MemberExpression, literal string, symbols map[string]symbol, equalMatch bool) {
	target, ok := member.Target.(*ast.Identifier)
	if !ok {
		return
	}
	current, ok := symbols[target.Name]
	if !ok {
		return
	}
	narrowed := narrowUnionByDiscriminator(current.kind, member.Property, literal, equalMatch)
	if narrowed == "" {
		return
	}
	symbols[target.Name] = symbol{kind: narrowed, mutable: current.mutable}
}

func narrowUnionByDiscriminator(typeName string, property string, literal string, equalMatch bool) string {
	expr, err := typesys.Parse(normalizeTypeAnnotation(typeName))
	if err != nil || expr == nil || expr.Kind != typesys.KindUnion {
		return ""
	}
	matched := []string{}
	for _, element := range expr.Elements {
		discriminatorType := structuredMemberType(element.String(), property)
		if discriminatorType == "" {
			if !equalMatch {
				matched = append(matched, element.String())
			}
			continue
		}
		isMatch := normalizeTypeAnnotation(discriminatorType) == normalizeTypeAnnotation(literal)
		if isMatch == equalMatch {
			matched = append(matched, element.String())
		}
	}
	if len(matched) == 0 {
		return ""
	}
	if len(matched) == 1 {
		return matched[0]
	}
	return strings.Join(matched, "|")
}

func excludeUnionDiscriminatorLiterals(typeName string, property string, excluded []string) string {
	current := normalizeTypeAnnotation(typeName)
	for _, literal := range excluded {
		narrowed := narrowUnionByDiscriminator(current, property, literal, false)
		if narrowed == "" {
			return current
		}
		current = narrowed
	}
	return current
}

func literalTypeFromExpression(expr ast.Expression) (string, bool) {
	switch value := expr.(type) {
	case *ast.StringLiteral:
		return strconv.Quote(value.Value), true
	case *ast.NumberLiteral:
		return strconv.FormatFloat(value.Value, 'f', -1, 64), true
	case *ast.BooleanLiteral:
		if value.Value {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func structuredExpressionType(expr ast.Expression, symbols map[string]symbol, functions map[string]functionSignature) string {
	switch expr := expr.(type) {
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(expr.Target, symbols, functions)
		if err != nil {
			return ""
		}
		return structuredMemberType(targetKind, expr.Property)
	case *ast.IndexExpression:
		targetKind, err := inferExpressionKind(expr.Target, symbols, functions)
		if err != nil {
			return ""
		}
		indexKind, err := inferExpressionKind(expr.Index, symbols, functions)
		if err != nil {
			return ""
		}
		_, valueType := structuredIndexAssignmentRule(targetKind, indexKind)
		return valueType
	default:
		return ""
	}
}

func matchesLiteralType(expected string, expr ast.Expression) bool {
	switch value := expr.(type) {
	case *ast.StringLiteral:
		return expected == strconv.Quote(value.Value)
	case *ast.NumberLiteral:
		return expected == strconv.FormatFloat(value.Value, 'f', -1, 64)
	case *ast.BooleanLiteral:
		if value.Value {
			return expected == "true"
		}
		return expected == "false"
	default:
		return false
	}
}

func structuredMemberAssignmentRule(typeName string, property string) (readonly bool, valueType string) {
	expr, err := typesys.Parse(normalizeTypeAnnotation(typeName))
	if err != nil || expr == nil {
		return false, ""
	}
	if expr.Kind == typesys.KindIntersection {
		readonlyAny := false
		var valueTypes []string
		for _, element := range expr.Elements {
			readonly, valueType := structuredMemberAssignmentRule(element.String(), property)
			if valueType != "" {
				readonlyAny = readonlyAny || readonly
				valueTypes = append(valueTypes, valueType)
			}
		}
		if len(valueTypes) == 0 {
			return false, ""
		}
		return readonlyAny, strings.Join(valueTypes, "&")
	}
	if expr.Kind != typesys.KindObject {
		return false, ""
	}
	for _, item := range expr.Properties {
		if item.Name == property {
			return item.Readonly, item.Type.String()
		}
	}
	for _, signature := range expr.IndexSignatures {
		if isAssignableTo(signature.KeyType.String(), "string") {
			return signature.Readonly, signature.ValueType.String()
		}
	}
	return false, ""
}

func structuredMemberType(typeName string, property string) string {
	expr, err := typesys.Parse(normalizeTypeAnnotation(typeName))
	if err != nil || expr == nil {
		return ""
	}
	if expr.Kind == typesys.KindIntersection {
		found := []string{}
		for _, element := range expr.Elements {
			if valueType := structuredMemberType(element.String(), property); valueType != "" {
				found = append(found, valueType)
			}
		}
		if len(found) == 0 {
			return ""
		}
		return strings.Join(found, "&")
	}
	if expr.Kind != typesys.KindObject {
		return ""
	}
	for _, item := range expr.Properties {
		if item.Name == property {
			return item.Type.String()
		}
	}
	for _, signature := range expr.IndexSignatures {
		if isAssignableTo(signature.KeyType.String(), "string") {
			return signature.ValueType.String()
		}
	}
	return ""
}

func structuredIndexAssignmentRule(typeName string, indexKind string) (readonly bool, valueType string) {
	expr, err := typesys.Parse(normalizeTypeAnnotation(typeName))
	if err != nil || expr == nil {
		return false, ""
	}
	if expr.Kind == typesys.KindIntersection {
		readonlyAny := false
		var valueTypes []string
		for _, element := range expr.Elements {
			readonly, valueType := structuredIndexAssignmentRule(element.String(), indexKind)
			if valueType != "" {
				readonlyAny = readonlyAny || readonly
				valueTypes = append(valueTypes, valueType)
			}
		}
		if len(valueTypes) == 0 {
			return false, ""
		}
		return readonlyAny, strings.Join(valueTypes, "&")
	}
	if expr.Kind != typesys.KindObject {
		return false, ""
	}
	for _, signature := range expr.IndexSignatures {
		if isAssignableTo(signature.KeyType.String(), indexKind) {
			return signature.Readonly, signature.ValueType.String()
		}
	}
	return false, ""
}

func isBuiltinConstructor(name string) bool {
	switch name {
	case "Map", "Set", "WeakMap", "WeakSet", "Date", "RegExp", "Error", "TypeError", "AggregateError", "ArrayBuffer", "SharedArrayBuffer", "Int8Array", "Uint8Array", "Uint16Array", "Int16Array", "Uint32Array", "Int32Array", "Float32Array", "Float64Array", "DataView":
		return true
	default:
		return false
	}
}

func hasSpreadArguments(arguments []ast.Expression) bool {
	for _, arg := range arguments {
		if _, ok := arg.(*ast.SpreadExpression); ok {
			return true
		}
	}
	return false
}

func (a *Analyzer) validateClassStatements(statements []ast.Statement, ctx *classContext) error {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			if err := a.validateClassExpression(stmt.Value, ctx); err != nil {
				return err
			}
		case *ast.AssignmentStatement:
			if err := a.validateClassExpression(stmt.Target, ctx); err != nil {
				return err
			}
			if err := a.validateClassExpression(stmt.Value, ctx); err != nil {
				return err
			}
		case *ast.DeleteStatement:
			if err := a.validateClassExpression(stmt.Target, ctx); err != nil {
				return err
			}
		case *ast.ReturnStatement:
			if err := a.validateClassExpression(stmt.Value, ctx); err != nil {
				return err
			}
		case *ast.ExpressionStatement:
			if err := a.validateClassExpression(stmt.Expression, ctx); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.validateClassExpression(stmt.Condition, ctx); err != nil {
				return err
			}
			if err := a.validateClassStatements(stmt.Consequence, ctx); err != nil {
				return err
			}
			if err := a.validateClassStatements(stmt.Alternative, ctx); err != nil {
				return err
			}
		case *ast.WhileStatement:
			if err := a.validateClassExpression(stmt.Condition, ctx); err != nil {
				return err
			}
			if err := a.validateClassStatements(stmt.Body, ctx); err != nil {
				return err
			}
		case *ast.ForStatement:
			if stmt.Init != nil {
				if err := a.validateClassStatements([]ast.Statement{stmt.Init}, ctx); err != nil {
					return err
				}
			}
			if stmt.Condition != nil {
				if err := a.validateClassExpression(stmt.Condition, ctx); err != nil {
					return err
				}
			}
			if stmt.Update != nil {
				if err := a.validateClassStatements([]ast.Statement{stmt.Update}, ctx); err != nil {
					return err
				}
			}
			if err := a.validateClassStatements(stmt.Body, ctx); err != nil {
				return err
			}
		case *ast.ForOfStatement:
			if err := a.validateClassExpression(stmt.Iterable, ctx); err != nil {
				return err
			}
			if err := a.validateClassStatements(stmt.Body, ctx); err != nil {
				return err
			}
		case *ast.ForInStatement:
			if err := a.validateClassExpression(stmt.Iterable, ctx); err != nil {
				return err
			}
			if err := a.validateClassStatements(stmt.Body, ctx); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.validateClassStatements(stmt.Body, ctx); err != nil {
				return err
			}
		case *ast.SwitchStatement:
			if err := a.validateClassExpression(stmt.Discriminant, ctx); err != nil {
				return err
			}
			for _, switchCase := range stmt.Cases {
				if err := a.validateClassExpression(switchCase.Test, ctx); err != nil {
					return err
				}
				if err := a.validateClassStatements(switchCase.Consequent, ctx); err != nil {
					return err
				}
			}
			if err := a.validateClassStatements(stmt.Default, ctx); err != nil {
				return err
			}
		case *ast.LabeledStatement:
			if err := a.validateClassStatements([]ast.Statement{stmt.Statement}, ctx); err != nil {
				return err
			}
		case *ast.BreakStatement, *ast.ContinueStatement:
		default:
			return errorAt(stmt, "unsupported class statement")
		}
	}
	return nil
}

func (a *Analyzer) validateClassExpression(expr ast.Expression, ctx *classContext) error {
	switch expr := expr.(type) {
	case *ast.NumberLiteral, *ast.BigIntLiteral, *ast.BooleanLiteral, *ast.NullLiteral, *ast.UndefinedLiteral, *ast.StringLiteral, *ast.Identifier, *ast.NewTargetExpression:
		return nil
	case *ast.AwaitExpression:
		return a.validateClassExpression(expr.Value, ctx)
	case *ast.ThisExpression:
		return nil
	case *ast.SuperExpression:
		if ctx.info.superClass == "" {
			return errorAt(expr, "super is only valid in derived class %s", ctx.info.name)
		}
		return nil
	case *ast.BoundSuperExpression:
		if expr.BaseClass == "" {
			return errorAt(expr, "super is only valid in derived class %s", ctx.info.name)
		}
		if expr.Receiver != nil {
			return a.validateClassExpression(expr.Receiver, ctx)
		}
		return nil
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Spread {
				if err := a.validateClassExpression(property.Value, ctx); err != nil {
					return err
				}
				continue
			}
			if property.Computed {
				if err := a.validateClassExpression(property.KeyExpr, ctx); err != nil {
					return err
				}
			}
			if property.Getter && property.Setter {
				return errorAt(expr, "object accessor %s cannot be both getter and setter", property.Key)
			}
			if fn, ok := property.Value.(*ast.FunctionExpression); ok {
				if property.Getter && len(fn.Params) != 0 {
					return errorAt(expr, "object getter %s must not declare parameters", property.Key)
				}
				if property.Setter && len(fn.Params) != 1 {
					return errorAt(expr, "object setter %s must declare exactly one parameter", property.Key)
				}
			}
			if err := a.validateClassExpression(property.Value, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if err := a.validateClassExpression(element, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			if err := a.validateClassExpression(value, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.SpreadExpression:
		return errorAt(expr, "spread expressions are not supported yet")
	case *ast.BinaryExpression:
		if err := a.validateClassExpression(expr.Left, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.TypeofExpression:
		return a.validateClassExpression(expr.Value, ctx)
	case *ast.TypeCheckExpression:
		if err := a.validateClassExpression(expr.Value, ctx); err != nil {
			return err
		}
		if !isSupportedTypeAnnotation(expr.TypeAnnotation) {
			return errorAt(expr, "unsupported type annotation %q in runtime type check", expr.TypeAnnotation)
		}
		return nil
	case *ast.InstanceofExpression:
		if err := a.validateClassExpression(expr.Left, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.ComparisonExpression:
		if err := a.validateClassExpression(expr.Left, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.LogicalExpression:
		if err := a.validateClassExpression(expr.Left, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.ConditionalExpression:
		if err := a.validateClassExpression(expr.Condition, ctx); err != nil {
			return err
		}
		if err := a.validateClassExpression(expr.Consequent, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Alternative, ctx)
	case *ast.CommaExpression:
		if err := a.validateClassExpression(expr.Left, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.UnaryExpression:
		return a.validateClassExpression(expr.Right, ctx)
	case *ast.IndexExpression:
		if err := a.validateClassExpression(expr.Target, ctx); err != nil {
			return err
		}
		return a.validateClassExpression(expr.Index, ctx)
	case *ast.MemberExpression:
		return a.validateClassMemberExpression(expr, ctx)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			if err := a.validateClassExpression(arg, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.ClosureExpression:
		return a.validateClassExpression(expr.Environment, ctx)
	case *ast.CastExpression:
		return a.validateClassExpression(expr.Value, ctx)
	case *ast.InvokeExpression:
		if err := a.validateClassExpression(expr.Callee, ctx); err != nil {
			return err
		}
		if _, ok := expr.Callee.(*ast.SuperExpression); ok {
			if ctx.info.superClass == "" || ctx.inStatic || !ctx.inConstructor {
				return errorAt(expr, "super() is only valid inside derived constructors")
			}
		}
		for _, arg := range expr.Arguments {
			if err := a.validateClassExpression(arg, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.NewExpression:
		ident, ok := expr.Callee.(*ast.Identifier)
		if !ok {
			return errorAt(expr, "dynamic constructors are not supported")
		}
		if isBuiltinConstructor(ident.Name) {
			for _, arg := range expr.Arguments {
				if err := a.validateClassExpression(arg, ctx); err != nil {
					return err
				}
			}
			return nil
		}
		if _, ok := ctx.classes[ident.Name]; !ok {
			return errorAt(expr, "unknown class %s", ident.Name)
		}
		for _, arg := range expr.Arguments {
			if err := a.validateClassExpression(arg, ctx); err != nil {
				return err
			}
		}
		return nil
	default:
		return errorAt(expr, "unsupported class expression")
	}
}

func (a *Analyzer) validateClassMemberExpression(expr *ast.MemberExpression, ctx *classContext) error {
	switch target := expr.Target.(type) {
	case *ast.ThisExpression:
		if expr.Private {
			if ctx.inStatic {
				if !ctx.info.privateStaticFields[expr.Property] && !ctx.info.privateStaticMethods[expr.Property] {
					return errorAt(expr, "unknown private static member #%s on class %s", expr.Property, ctx.info.name)
				}
			} else {
				if !ctx.info.privateFields[expr.Property] && !ctx.info.privateMethods[expr.Property] {
					return errorAt(expr, "unknown private member #%s on class %s", expr.Property, ctx.info.name)
				}
			}
		}
		return nil
	case *ast.SuperExpression:
		if ctx.info.superClass == "" {
			return errorAt(expr, "super is only valid in derived class %s", ctx.info.name)
		}
		if expr.Private {
			return errorAt(expr, "private super access is not supported")
		}
		return nil
	default:
		if err := a.validateClassExpression(target, ctx); err != nil {
			return err
		}
		if expr.Private {
			return errorAt(expr, "private fields are only accessible through this.#name inside the declaring class")
		}
		return nil
	}
}
