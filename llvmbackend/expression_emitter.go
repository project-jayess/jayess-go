package llvmbackend

import (
	"fmt"
	"strconv"

	"jayess-go/ast"
	"jayess-go/lifetime"
)

type ExpressionEmitter struct {
	valueIndex  int
	stringIndex int
	slotIndex   int
	blockIndex  int

	locals       map[string]localSlot
	declarations []Declaration
	declared     map[string]struct{}
	globals      []Global
	body         []string
	scopes       []map[string]scopedLocal
	lifetimePlan *lifetime.Plan
}

func NewExpressionEmitter() *ExpressionEmitter {
	return &ExpressionEmitter{locals: map[string]localSlot{}, declared: map[string]struct{}{}}
}

func (emitter *ExpressionEmitter) SetLifetimePlan(plan *lifetime.Plan) {
	emitter.lifetimePlan = plan
}

func LowerRuntimeExpressionFunction(name string, expression ast.Expression) (Function, []Declaration, []Global, error) {
	if name == "" {
		return Function{}, nil, nil, fmt.Errorf("runtime expression function name must not be empty")
	}
	emitter := NewExpressionEmitter()
	value, err := emitter.EmitExpression(expression)
	if err != nil {
		return Function{}, nil, nil, err
	}
	body := append(emitter.Body(), "ret "+runtimeValueIRType+" "+value)
	return Function{
		Name:       name,
		ReturnType: runtimeValueIRType,
		Body:       body,
	}, emitter.Declarations(), emitter.Globals(), nil
}

func (emitter *ExpressionEmitter) EmitExpression(expression ast.Expression) (result string, err error) {
	defer func() {
		err = diagnosticError(expression, err)
	}()
	if comma, ok := expression.(*ast.CommaExpression); ok {
		return emitter.emitCommaExpression(comma)
	}
	if unary, ok := expression.(*ast.UnaryExpression); ok {
		return emitter.emitUnaryExpression(unary)
	}
	if update, ok := expression.(*ast.UpdateExpression); ok {
		return emitter.emitUpdateExpression(update)
	}
	if typeof, ok := expression.(*ast.TypeofExpression); ok {
		return emitter.emitTypeofExpression(typeof)
	}
	if conditional, ok := expression.(*ast.ConditionalExpression); ok {
		return emitter.emitConditionalExpression(conditional)
	}
	if nullish, ok := expression.(*ast.NullishCoalesceExpression); ok {
		return emitter.emitNullishCoalesceExpression(nullish)
	}
	if logical, ok := expression.(*ast.LogicalExpression); ok {
		return emitter.emitLogicalExpression(logical)
	}
	if binary, ok := expression.(*ast.BinaryExpression); ok {
		return emitter.emitBinaryExpression(binary)
	}
	if comparison, ok := expression.(*ast.ComparisonExpression); ok {
		return emitter.emitComparisonExpression(comparison)
	}
	if instanceOf, ok := expression.(*ast.InstanceofExpression); ok {
		return emitter.emitInstanceofExpression(instanceOf)
	}
	if object, ok := expression.(*ast.ObjectLiteral); ok {
		return emitter.emitObjectLiteral(object)
	}
	if array, ok := expression.(*ast.ArrayLiteral); ok {
		return emitter.emitArrayLiteral(array)
	}
	if member, ok := expression.(*ast.MemberExpression); ok {
		return emitter.emitMemberExpression(member)
	}
	if index, ok := expression.(*ast.IndexExpression); ok {
		return emitter.emitIndexExpression(index)
	}
	if call, ok := expression.(*ast.CallExpression); ok {
		return emitter.emitCallExpression(call)
	}
	if invoke, ok := expression.(*ast.InvokeExpression); ok {
		return emitter.emitInvokeExpression(invoke)
	}
	if newExpression, ok := expression.(*ast.NewExpression); ok {
		return emitter.emitNewExpression(newExpression)
	}
	if thisExpression, ok := expression.(*ast.ThisExpression); ok {
		return emitter.emitThisExpression(thisExpression)
	}
	if superExpression, ok := expression.(*ast.SuperExpression); ok {
		return emitter.emitSuperExpression(superExpression)
	}
	if newTarget, ok := expression.(*ast.NewTargetExpression); ok {
		return emitter.emitNewTargetExpression(newTarget)
	}
	if function, ok := expression.(*ast.FunctionExpression); ok {
		return emitter.emitFunctionExpression(function)
	}
	if identifier, ok := expression.(*ast.Identifier); ok {
		return emitter.LoadLocal(identifier.Name)
	}
	result = emitter.nextValueName()
	lowered, err := LowerASTRuntimeLiteral(result, expression, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.globals = append(emitter.globals, lowered.Globals...)
	emitter.stringIndex += len(lowered.Globals)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func (emitter *ExpressionEmitter) BindLocal(name string, value string) error {
	if name == "" {
		return fmt.Errorf("local name must not be empty")
	}
	if value == "" {
		return fmt.Errorf("local %s must not be bound to an empty value", name)
	}
	if emitter.locals == nil {
		emitter.locals = map[string]localSlot{}
	}
	slot := emitter.nextLocalSlot()
	emitter.locals[name] = slot
	emitter.body = append(emitter.body,
		slot.Name+" = alloca "+runtimeValueIRType,
		"store "+runtimeValueIRType+" "+value+", "+runtimeValueIRType+"* "+slot.Name,
	)
	return nil
}

func (emitter *ExpressionEmitter) HasLocal(name string) bool {
	_, exists := emitter.locals[name]
	return exists
}

func (emitter *ExpressionEmitter) Body() []string {
	return append([]string{}, emitter.body...)
}

func (emitter *ExpressionEmitter) Declarations() []Declaration {
	return append([]Declaration{}, emitter.declarations...)
}

func (emitter *ExpressionEmitter) Globals() []Global {
	return append([]Global{}, emitter.globals...)
}

func (emitter *ExpressionEmitter) nextValueName() string {
	name := "%v" + strconv.Itoa(emitter.valueIndex)
	emitter.valueIndex++
	return name
}

func (emitter *ExpressionEmitter) nextBlockLabel(prefix string) string {
	var builder BasicBlockBuilder
	builder.next = emitter.blockIndex
	label := builder.NewLabel(prefix)
	emitter.blockIndex = builder.next
	return label
}

func (emitter *ExpressionEmitter) addDeclarations(declarations []Declaration) {
	if emitter.declared == nil {
		emitter.declared = map[string]struct{}{}
	}
	for _, declaration := range declarations {
		key := declaration.IRType + " @" + declaration.Name
		if _, exists := emitter.declared[key]; exists {
			continue
		}
		emitter.declared[key] = struct{}{}
		emitter.declarations = append(emitter.declarations, declaration)
	}
}
