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
	isMain     bool
	isExtern   bool
	variadic   bool
}

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(program *ast.Program) error {
	if len(program.Functions) == 0 {
		return fmt.Errorf("program must declare at least one function")
	}

	globalSymbols := map[string]symbol{}
	for _, global := range program.Globals {
		if global.Kind != ast.DeclarationConst && global.Kind != ast.DeclarationVar {
			return fmt.Errorf("top-level variables must use var or const")
		}
		if _, exists := globalSymbols[global.Name]; exists {
			return fmt.Errorf("duplicate global %s", global.Name)
		}
		kind, err := inferExpressionKind(global.Value, globalSymbols, nil)
		if err != nil {
			return err
		}
		if !isRuntimeValueKind(kind) {
			return fmt.Errorf("global %s must be a runtime value", global.Name)
		}
		globalSymbols[global.Name] = symbol{kind: "dynamic", mutable: global.Kind == ast.DeclarationVar}
	}

	seenMain := false
	functions := map[string]functionSignature{}
	for _, fn := range program.ExternFunctions {
		if _, exists := globalSymbols[fn.Name]; exists {
			return fmt.Errorf("name %s is already used by a global", fn.Name)
		}
		if _, exists := functions[fn.Name]; exists {
			return fmt.Errorf("duplicate function %s", fn.Name)
		}
		functions[fn.Name] = functionSignature{name: fn.Name, nativeName: fn.NativeSymbol, paramCount: len(fn.Params), isExtern: true, variadic: fn.Variadic}
	}
	for _, fn := range program.Functions {
		if _, exists := globalSymbols[fn.Name]; exists {
			return fmt.Errorf("name %s is already used by a global", fn.Name)
		}
		if _, exists := functions[fn.Name]; exists {
			return fmt.Errorf("duplicate function %s", fn.Name)
		}
		functions[fn.Name] = functionSignature{name: fn.Name, paramCount: len(fn.Params), isMain: fn.Name == "main"}
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
		return fmt.Errorf("entrypoint function main was not found")
	}
	return nil
}

func validateFunction(fn *ast.FunctionDecl, functions map[string]functionSignature, globals map[string]symbol) error {
	if len(fn.Body) == 0 {
		return fmt.Errorf("function %s must contain at least one statement", fn.Name)
	}
	if fn.Name == "main" && len(fn.Params) > 1 {
		return fmt.Errorf("main supports at most one parameter: args")
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

	if err := validateStatements(fn.Body[:len(fn.Body)-1], symbols, false, functions); err != nil {
		return err
	}

	lastReturn, ok := fn.Body[len(fn.Body)-1].(*ast.ReturnStatement)
	if !ok {
		return fmt.Errorf("function %s must terminate with a return statement", fn.Name)
	}
	kind, err := inferExpressionKind(lastReturn.Value, symbols, functions)
	if err != nil {
		return err
	}
	if fn.Name == "main" && kind != "number" && kind != "dynamic" {
		return fmt.Errorf("function %s must return a number-like value", fn.Name)
	}
	if fn.Name != "main" && !isRuntimeValueKind(kind) {
		return fmt.Errorf("function %s must return a runtime value", fn.Name)
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
			symbols[stmt.Name] = symbol{kind: kind, mutable: stmt.Kind != ast.DeclarationConst}
		case *ast.AssignmentStatement:
			if err := validateAssignment(stmt, symbols, functions); err != nil {
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
		case *ast.BreakStatement, *ast.ContinueStatement:
			if !inLoop {
				return fmt.Errorf("break and continue are only valid inside loops")
			}
		default:
			return fmt.Errorf("unsupported statement")
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
		return fmt.Errorf("value of type %s cannot be used as a condition", kind)
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
			return fmt.Errorf("unknown identifier %s", target.Name)
		}
		if !current.mutable {
			return fmt.Errorf("cannot reassign const %s", target.Name)
		}
		if current.kind == "dynamic" {
			if !isRuntimeValueKind(valueKind) {
				return fmt.Errorf("cannot assign %s to %s", valueKind, current.kind)
			}
			return nil
		}
		if current.kind != valueKind {
			return fmt.Errorf("cannot assign %s to %s", valueKind, current.kind)
		}
		return nil
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind != "object" && targetKind != "dynamic" {
			return fmt.Errorf("member assignment requires an object target")
		}
		if !isRuntimeValueKind(valueKind) {
			return fmt.Errorf("object properties currently support runtime values")
		}
		return nil
	case *ast.IndexExpression:
		targetKind, err := inferExpressionKind(target.Target, symbols, functions)
		if err != nil {
			return err
		}
		if targetKind != "array" && targetKind != "dynamic" {
			return fmt.Errorf("index assignment requires an array target")
		}
		if !isRuntimeValueKind(valueKind) {
			return fmt.Errorf("array elements currently support runtime values")
		}
		return nil
	default:
		return fmt.Errorf("unsupported assignment target")
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
	case *ast.StringLiteral:
		return "string", nil
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			kind, err := inferExpressionKind(property.Value, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", fmt.Errorf("object literal values currently support runtime values")
			}
		}
		return "object", nil
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			kind, err := inferExpressionKind(element, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", fmt.Errorf("array literal elements currently support runtime values")
			}
		}
		return "array", nil
	case *ast.Identifier:
		value, ok := symbols[expr.Name]
		if !ok {
			return "", fmt.Errorf("unknown identifier %s", expr.Name)
		}
		return value.kind, nil
	case *ast.BinaryExpression:
		leftKind, err := inferExpressionKind(expr.Left, symbols, functions)
		if err != nil {
			return "", err
		}
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		if (leftKind != "number" && leftKind != "dynamic") || (rightKind != "number" && rightKind != "dynamic") {
			return "", fmt.Errorf("operator %s expects number operands", expr.Operator)
		}
		return "number", nil
	case *ast.UnaryExpression:
		rightKind, err := inferExpressionKind(expr.Right, symbols, functions)
		if err != nil {
			return "", err
		}
		switch expr.Operator {
		case ast.OperatorNot:
			if !isTruthyKind(rightKind) {
				return "", fmt.Errorf("operator ! expects a truthy-compatible operand")
			}
			return "boolean", nil
		default:
			return "", fmt.Errorf("unsupported unary operator")
		}
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
			return "", fmt.Errorf("logical operator %s expects truthy-compatible operands", expr.Operator)
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
				return "", fmt.Errorf("operator %s does not support args arrays directly", expr.Operator)
			}
			if !((leftKind == rightKind) || (leftKind == "dynamic" || rightKind == "dynamic")) {
				return "", fmt.Errorf("operator %s expects comparable operands", expr.Operator)
			}
		default:
			if (leftKind != "number" && leftKind != "dynamic") || (rightKind != "number" && rightKind != "dynamic") {
				return "", fmt.Errorf("operator %s expects number operands", expr.Operator)
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
		if targetKind != "args_array" && targetKind != "array" && targetKind != "dynamic" {
			return "", fmt.Errorf("indexing is currently only supported for args and arrays")
		}
		if indexKind != "number" {
			return "", fmt.Errorf("array index must be a number")
		}
		if targetKind == "array" || targetKind == "dynamic" {
			return "dynamic", nil
		}
		return "string", nil
	case *ast.MemberExpression:
		targetKind, err := inferExpressionKind(expr.Target, symbols, functions)
		if err != nil {
			return "", err
		}
		if targetKind != "object" && targetKind != "dynamic" {
			return "", fmt.Errorf("member access requires an object target")
		}
		return "dynamic", nil
	case *ast.CallExpression:
		return validateCallExpression(expr, symbols, functions)
	default:
		return "", fmt.Errorf("unsupported expression")
	}
}

func validateCallExpression(call *ast.CallExpression, symbols map[string]symbol, functions map[string]functionSignature) (string, error) {
	switch call.Callee {
	case "print":
		if len(call.Arguments) != 1 {
			return "", fmt.Errorf("print expects 1 argument")
		}
		kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if !isPrintableKind(kind) {
			return "", fmt.Errorf("print expects a printable value")
		}
		return "void", nil
	case "readLine", "readKey":
		if len(call.Arguments) != 1 {
			return "", fmt.Errorf("%s expects 1 argument", call.Callee)
		}
		kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if kind != "string" {
			return "", fmt.Errorf("%s expects a string prompt", call.Callee)
		}
		return "string", nil
	case "sleep":
		if len(call.Arguments) != 1 {
			return "", fmt.Errorf("sleep expects 1 argument")
		}
		kind, err := inferExpressionKind(call.Arguments[0], symbols, functions)
		if err != nil {
			return "", err
		}
		if kind != "number" && kind != "dynamic" {
			return "", fmt.Errorf("sleep expects a number argument")
		}
		return "void", nil
	default:
		fn, ok := functions[call.Callee]
		if !ok {
			return "", fmt.Errorf("unknown function %s", call.Callee)
		}
		if !fn.variadic && len(call.Arguments) != fn.paramCount {
			return "", fmt.Errorf("function %s expects %d arguments", call.Callee, fn.paramCount)
		}
		for _, arg := range call.Arguments {
			kind, err := inferExpressionKind(arg, symbols, functions)
			if err != nil {
				return "", err
			}
			if !isRuntimeValueKind(kind) {
				return "", fmt.Errorf("function %s expects runtime-compatible arguments", call.Callee)
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
		symbols[stmt.Name] = symbol{kind: kind, mutable: stmt.Kind != ast.DeclarationConst}
		return nil
	case *ast.AssignmentStatement:
		return validateAssignment(stmt, symbols, functions)
	case *ast.ExpressionStatement:
		_, err := inferExpressionKind(stmt.Expression, symbols, functions)
		return err
	default:
		return fmt.Errorf("unsupported for-loop clause")
	}
}

func isRuntimeValueKind(kind string) bool {
	switch kind {
	case "string", "number", "boolean", "object", "array", "dynamic":
		return true
	default:
		return false
	}
}

func isPrintableKind(kind string) bool {
	switch kind {
	case "string", "number", "boolean", "dynamic", "array", "object", "args_array":
		return true
	default:
		return false
	}
}

func isTruthyKind(kind string) bool {
	switch kind {
	case "number", "boolean", "string", "args_array", "array", "object", "dynamic":
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
