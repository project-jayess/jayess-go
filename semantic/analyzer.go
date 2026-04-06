package semantic

import (
	"fmt"

	"jayess-go/ast"
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
	variadic   bool
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
	privateMethods       map[string]bool
	staticMethods        map[string]bool
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
		availableFunctions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isExtern: true, variadic: fn.Variadic}
	}
	for _, fn := range program.Functions {
		availableFunctions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isMain: fn.Name == "main"}
	}
	for _, global := range program.Globals {
		if global.Kind != ast.DeclarationConst && global.Kind != ast.DeclarationVar {
			return errorAt(global, "top-level variables must use var or const")
		}
		if _, exists := globalSymbols[global.Name]; exists {
			return errorAt(global, "duplicate global %s", global.Name)
		}
		kind, err := inferExpressionKind(global.Value, globalSymbols, availableFunctions)
		if err != nil {
			return err
		}
		if !isRuntimeValueKind(kind) {
			return errorAt(global, "global %s must be a runtime value", global.Name)
		}
		globalSymbols[global.Name] = symbol{kind: "dynamic", mutable: global.Kind == ast.DeclarationVar}
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
		functions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isExtern: true, variadic: fn.Variadic}
	}
	for _, fn := range program.Functions {
		if _, exists := globalSymbols[fn.Name]; exists {
			return errorAt(fn, "name %s is already used by a global", fn.Name)
		}
		if _, exists := functions[fn.Name]; exists {
			return errorAt(fn, "duplicate function %s", fn.Name)
		}
		functions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), minArgs: minRequiredParams(fn.Params), hasRest: hasRestParam(fn.Params), isMain: fn.Name == "main"}
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
			privateMethods:       map[string]bool{},
			staticMethods:        map[string]bool{},
			privateStaticMethods: map[string]bool{},
		}
		for _, member := range classDecl.Members {
			switch member := member.(type) {
			case *ast.ClassFieldDecl:
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
	if fn.Name == "main" && len(fn.Params) > 1 {
		return errorAt(fn, "main supports at most one parameter: args")
	}

	symbols := cloneSymbols(globals)
	if fn.Name == "main" && len(fn.Params) == 1 {
		symbols[fn.Params[0].Name] = symbol{kind: "args_array", mutable: false}
	}
	if fn.Name != "main" {
		for _, param := range fn.Params {
			symbols[param.Name] = symbol{kind: "dynamic", mutable: true}
		}
	}
	for _, param := range fn.Params {
		if param.Default != nil {
			if _, err := inferExpressionKind(param.Default, symbols, functions); err != nil {
				return err
			}
		}
	}

	if err := validateStatements(fn.Body[:len(fn.Body)-1], symbols, false, functions); err != nil {
		return err
	}

	lastReturn, ok := fn.Body[len(fn.Body)-1].(*ast.ReturnStatement)
	if !ok {
		return errorAt(fn, "function %s must terminate with a return statement", fn.Name)
	}
	kind, err := inferExpressionKind(lastReturn.Value, symbols, functions)
	if err != nil {
		return err
	}
	if fn.Name == "main" && kind != "number" && kind != "dynamic" {
		return errorAt(lastReturn, "function %s must return a number-like value", fn.Name)
	}
	if fn.Name != "main" && !isRuntimeValueKind(kind) {
		return errorAt(lastReturn, "function %s must return a runtime value", fn.Name)
	}
	return nil
}

func validateStatements(statements []ast.Statement, symbols map[string]symbol, inLoop bool, functions map[string]functionSignature) error {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.VariableDecl:
			kind, err := inferExpressionKind(stmt.Value, symbols, functions)
			if err != nil {
				return err
			}
			if stmt.Kind == ast.DeclarationVar {
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
			if err := validateStatements(stmt.TryBody, trySymbols, inLoop, functions); err != nil {
				return err
			}
			catchSymbols := cloneSymbols(symbols)
			if stmt.CatchName != "" {
				catchSymbols[stmt.CatchName] = symbol{kind: "dynamic", mutable: true}
			}
			if err := validateStatements(stmt.CatchBody, catchSymbols, inLoop, functions); err != nil {
				return err
			}
			finallySymbols := cloneSymbols(symbols)
			if err := validateStatements(stmt.FinallyBody, finallySymbols, inLoop, functions); err != nil {
				return err
			}
		case *ast.ReturnStatement:
			if _, err := inferExpressionKind(stmt.Value, symbols, functions); err != nil {
				return err
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
			if err := validateStatements(stmt.Consequence, consequenceSymbols, inLoop, functions); err != nil {
				return err
			}
			alternativeSymbols := cloneSymbols(symbols)
			if err := validateStatements(stmt.Alternative, alternativeSymbols, inLoop, functions); err != nil {
				return err
			}
		case *ast.WhileStatement:
			if err := validateCondition(stmt.Condition, symbols, functions); err != nil {
				return err
			}
			bodySymbols := cloneSymbols(symbols)
			if err := validateStatements(stmt.Body, bodySymbols, true, functions); err != nil {
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
			if err := validateStatements(stmt.Body, bodySymbols, true, functions); err != nil {
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
			if iterKind != "array" && iterKind != "dynamic" && iterKind != "args_array" {
				return errorAt(stmt, "for...of expects an array-like iterable")
			}
			loopSymbols := cloneSymbols(symbols)
			bindingKind := "dynamic"
			if stmt.Kind == ast.DeclarationConst {
				bindingKind = "dynamic"
			}
			loopSymbols[stmt.Name] = symbol{kind: bindingKind, mutable: stmt.Kind != ast.DeclarationConst}
			if err := validateStatements(stmt.Body, loopSymbols, true, functions); err != nil {
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
			if err := validateStatements(stmt.Body, loopSymbols, true, functions); err != nil {
				return err
			}
		case *ast.SwitchStatement:
			if _, err := inferExpressionKind(stmt.Discriminant, symbols, functions); err != nil {
				return err
			}
			for _, switchCase := range stmt.Cases {
				if _, err := inferExpressionKind(switchCase.Test, symbols, functions); err != nil {
					return err
				}
				caseSymbols := cloneSymbols(symbols)
				if err := validateStatements(switchCase.Consequent, caseSymbols, inLoop, functions); err != nil {
					return err
				}
			}
			defaultSymbols := cloneSymbols(symbols)
			if err := validateStatements(stmt.Default, defaultSymbols, inLoop, functions); err != nil {
				return err
			}
		case *ast.BreakStatement, *ast.ContinueStatement:
			if !inLoop {
				return errorAt(stmt, "break and continue are only valid inside loops")
			}
		default:
			return errorAt(stmt, "unsupported statement")
		}
	}
	return nil
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
		if current.kind != valueKind {
			return errorAt(stmt, "cannot assign %s to %s", valueKind, current.kind)
		}
		return nil
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind != "object" && targetKind != "dynamic" && targetKind != "function" {
			return errorAt(target, "member assignment requires an object target")
		}
		if !isRuntimeValueKind(valueKind) {
			return errorAt(stmt, "object properties currently support runtime values")
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
		if targetKind == "array" {
			if indexKind != "number" {
				return errorAt(target, "array index must be a number")
			}
		} else if targetKind == "object" || targetKind == "function" {
			if indexKind != "string" {
				return errorAt(target, "object property index must be a string")
			}
		} else if targetKind == "dynamic" {
			if indexKind != "number" && indexKind != "string" && indexKind != "dynamic" {
				return errorAt(target, "dynamic index must be a number-like or string-like value")
			}
		} else {
			return errorAt(target, "index assignment requires an array or object target")
		}
		if !isRuntimeValueKind(valueKind) {
			return errorAt(stmt, "indexed values currently support runtime values")
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
	case *ast.NumberLiteral:
		return "number", nil
	case *ast.BooleanLiteral:
		return "boolean", nil
	case *ast.NullLiteral, *ast.UndefinedLiteral:
		return "dynamic", nil
	case *ast.ThisExpression:
		return "dynamic", nil
	case *ast.NewTargetExpression:
		return "dynamic", nil
	case *ast.StringLiteral:
		return "string", nil
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Computed {
				keyKind, err := inferExpressionKind(property.KeyExpr, symbols, functions)
				if err != nil {
					return "", err
				}
				if keyKind != "string" && keyKind != "dynamic" {
					return "", errorAt(expr, "computed object keys must be string-like")
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
	case *ast.BinaryExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
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
		default:
			return "", errorAt(expr, "unsupported unary operator")
		}
	case *ast.TypeofExpression:
		if _, err := inferExpressionKind(expr.Value, symbols, functions); err != nil {
			return "", err
		}
		return "string", nil
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
		case ast.OperatorEq, ast.OperatorNe:
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
			if indexKind != "string" {
				return "", errorAt(expr, "object property index must be a string")
			}
			return "dynamic", nil
		case "dynamic":
			if indexKind != "number" && indexKind != "string" && indexKind != "dynamic" {
				return "", errorAt(expr, "dynamic index must be a number-like or string-like value")
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
		calleeKind, err := inferExpressionKind(expr.Callee, symbols, functions)
		if err != nil {
			return "", err
		}
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
	case "__jayess_path_basename", "__jayess_path_dirname", "__jayess_path_extname":
		if len(call.Arguments) != 1 {
			return "", errorAt(call, "%s expects 1 argument", call.Callee)
		}
		return "string", nil
	case "__jayess_fs_read_file":
		if len(call.Arguments) != 1 && len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_read_file expects 1 or 2 arguments")
		}
		return "dynamic", nil
	case "__jayess_fs_write_file":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "__jayess_fs_write_file expects 2 arguments")
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
	case "__jayess_fs_copy_file", "__jayess_fs_copy_dir", "__jayess_fs_rename":
		if len(call.Arguments) != 2 {
			return "", errorAt(call, "%s expects 2 arguments", call.Callee)
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
	case "__jayess_std_map_new", "__jayess_std_set_new":
		if len(call.Arguments) != 0 {
			return "", errorAt(call, "%s expects 0 arguments", call.Callee)
		}
		return "dynamic", nil
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
		minArgs := fn.minArgs
		if !fn.variadic && !hasSpreadArguments(call.Arguments) {
			if fn.hasRest {
				if len(call.Arguments) < minArgs {
					return "", errorAt(call, "function %s expects at least %d arguments", call.Callee, minArgs)
				}
			} else if len(call.Arguments) < minArgs || len(call.Arguments) > fn.paramCount {
				return "", errorAt(call, "function %s expects %d arguments", call.Callee, fn.paramCount)
			}
		}
		for _, arg := range call.Arguments {
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
				return "", errorAt(call, "function %s expects runtime-compatible arguments", call.Callee)
			}
		}
		if fn.isExtern {
			return "dynamic", nil
		}
		return "dynamic", nil
	}
}

func validateLoopStatement(stmt ast.Statement, symbols map[string]symbol, functions map[string]functionSignature) error {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		kind, err := inferExpressionKind(stmt.Value, symbols, functions)
		if err != nil {
			return err
		}
		if stmt.Kind == ast.DeclarationVar {
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
	case "string", "number", "boolean", "object", "array", "dynamic":
		return true
	case "function":
		return true
	default:
		return false
	}
}

func isPrintableKind(kind string) bool {
	switch kind {
	case "string", "number", "boolean", "dynamic", "array", "object", "args_array", "function":
		return true
	default:
		return false
	}
}

func isTruthyKind(kind string) bool {
	switch kind {
	case "number", "boolean", "string", "args_array", "array", "object", "dynamic", "function":
		return true
	default:
		return false
	}
}

func cloneSymbols(input map[string]symbol) map[string]symbol {
	out := make(map[string]symbol, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func hasRestParam(params []ast.Parameter) bool {
	return len(params) > 0 && params[len(params)-1].Rest
}

func minRequiredParams(params []ast.Parameter) int {
	count := 0
	for _, param := range params {
		if param.Rest {
			return count
		}
		if param.Default == nil {
			count++
		}
	}
	return count
}

func validateParameterList(params []ast.Parameter) error {
	seenDefault := false
	for i, param := range params {
		if param.Rest && i != len(params)-1 {
			return fmt.Errorf("rest parameter must be last")
		}
		if param.Rest && param.Default != nil {
			return fmt.Errorf("rest parameter cannot have a default value")
		}
		if param.Default != nil {
			seenDefault = true
		} else if seenDefault && !param.Rest {
			return fmt.Errorf("parameters without defaults cannot follow parameters with defaults")
		}
	}
	return nil
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
		case *ast.BreakStatement, *ast.ContinueStatement:
		default:
			return errorAt(stmt, "unsupported class statement")
		}
	}
	return nil
}

func (a *Analyzer) validateClassExpression(expr ast.Expression, ctx *classContext) error {
	switch expr := expr.(type) {
	case *ast.NumberLiteral, *ast.BooleanLiteral, *ast.NullLiteral, *ast.UndefinedLiteral, *ast.StringLiteral, *ast.Identifier, *ast.NewTargetExpression:
		return nil
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
			if property.Computed {
				if err := a.validateClassExpression(property.KeyExpr, ctx); err != nil {
					return err
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
