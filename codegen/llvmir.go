package codegen

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"jayess-go/ir"
)

type LLVMIRGenerator struct{}

func NewLLVMIRGenerator() *LLVMIRGenerator {
	return &LLVMIRGenerator{}
}

type variableSlot struct {
	kind ir.ValueKind
	ptr  string
}

type emittedValue struct {
	kind ir.ValueKind
	ref  string
}

type functionState struct {
	tempCounter  int
	labelCounter int
	slots        map[string]variableSlot
	stringRefs   map[string]string
	loopStack    []loopLabels
	functions    map[string]ir.Function
	externs      map[string]ir.ExternFunction
	globals      map[string]ir.ValueKind
	isMain       bool
}

type loopLabels struct {
	continueTarget string
	end            string
}

func (g *LLVMIRGenerator) Generate(module *ir.Module, targetTriple string) ([]byte, error) {
	if targetTriple == "" {
		return nil, fmt.Errorf("target triple is required")
	}

	stringsPool := collectStrings(module)
	stringRefs := buildStringRefMap(stringsPool)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "; jayess module\n")
	fmt.Fprintf(&buf, "target triple = %q\n\n", targetTriple)

	for i, value := range stringsPool {
		encoded, length := encodeLLVMString(value)
		fmt.Fprintf(&buf, "@.str.%d = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n", i, length+1, encoded)
	}
	if len(stringsPool) > 0 {
		fmt.Fprintln(&buf)
	}

	globalKinds := map[string]ir.ValueKind{}
	for _, global := range module.Globals {
		globalKinds[global.Name] = ir.ValueDynamic
		fmt.Fprintf(&buf, "@jayess_global_%s = internal global ptr null\n", global.Name)
	}
	if len(module.Globals) > 0 {
		fmt.Fprintln(&buf)
	}

	fmt.Fprintf(&buf, "declare void @jayess_print_string(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_number(double)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_bool(i1)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_object(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_array(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_args(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_print_value(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_line(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_key(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_sleep_ms(i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_make_args(i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_args_get(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_object_new()\n")
	fmt.Fprintf(&buf, "declare void @jayess_object_set_value(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_object_get(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_member(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_get_member(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_new()\n")
	fmt.Fprintf(&buf, "declare void @jayess_array_set_value(ptr, i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_get(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_index(ptr, i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_get_index(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_string(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_number(double)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_bool(i1)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_object(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_array(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_null()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_undefined()\n")
	fmt.Fprintf(&buf, "declare double @jayess_value_to_number(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_args_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_is_truthy(ptr)\n\n")

	for _, fn := range module.ExternFunctions {
		fmt.Fprintf(&buf, "declare ptr @%s(", fn.SymbolName)
		if fn.Variadic {
			buf.WriteString("...")
		} else {
			for i := range fn.Params {
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString("ptr")
			}
		}
		buf.WriteString(")\n")
	}
	if len(module.ExternFunctions) > 0 {
		buf.WriteString("\n")
	}

	functionsByName := map[string]ir.Function{}
	for _, fn := range module.Functions {
		functionsByName[fn.Name] = fn
	}
	externsByName := map[string]ir.ExternFunction{}
	for _, fn := range module.ExternFunctions {
		externsByName[fn.Name] = fn
	}

	if err := g.emitGlobalInit(&buf, module.Globals, stringRefs, functionsByName, externsByName, globalKinds); err != nil {
		return nil, err
	}

	for _, fn := range module.Functions {
		if err := g.emitFunction(&buf, fn, stringRefs, functionsByName, externsByName, globalKinds); err != nil {
			return nil, err
		}
	}

	if err := g.emitEntryWrapper(&buf, findMain(module.Functions)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *LLVMIRGenerator) emitFunction(buf *bytes.Buffer, fn ir.Function, stringRefs map[string]string, functionsByName map[string]ir.Function, externsByName map[string]ir.ExternFunction, globalKinds map[string]ir.ValueKind) error {
	headerName := fn.Name
	returnType := "ptr"
	if fn.Name == "main" {
		headerName = "jayess_user_main"
		returnType = "double"
	}
	fmt.Fprintf(buf, "define %s @%s(", returnType, headerName)
	for i, param := range fn.Params {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(buf, "ptr %%%s", param.Name)
	}
	buf.WriteString(") {\n")
	buf.WriteString("entry:\n")

	state := &functionState{
		slots:      map[string]variableSlot{},
		stringRefs: stringRefs,
		functions:  functionsByName,
		externs:    externsByName,
		globals:    globalKinds,
		isMain:     fn.Name == "main",
	}

	for _, param := range fn.Params {
		slot := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", slot)
		fmt.Fprintf(buf, "  store ptr %%%s, ptr %s\n", param.Name, slot)
		state.slots[param.Name] = variableSlot{kind: param.Kind, ptr: slot}
	}

	terminated, err := g.emitStatements(buf, state, fn.Body)
	if err != nil {
		return err
	}
	if !terminated {
		if state.isMain {
			buf.WriteString("  ret double 0.000000\n")
		} else {
			buf.WriteString("  %tmp.default = call ptr @jayess_value_undefined()\n")
			buf.WriteString("  ret ptr %tmp.default\n")
		}
	}

	buf.WriteString("}\n\n")
	return nil
}

func (g *LLVMIRGenerator) emitEntryWrapper(buf *bytes.Buffer, fn ir.Function) error {
	buf.WriteString("define i32 @main(i32 %argc, ptr %argv) {\n")
	buf.WriteString("entry:\n")
	buf.WriteString("  call void @jayess_init_globals()\n")
	if len(fn.Params) == 1 {
		buf.WriteString("  %args = call ptr @jayess_make_args(i32 %argc, ptr %argv)\n")
		buf.WriteString("  %result = call double @jayess_user_main(ptr %args)\n")
	} else {
		buf.WriteString("  %result = call double @jayess_user_main()\n")
	}
	buf.WriteString("  %exit = fptosi double %result to i32\n")
	buf.WriteString("  ret i32 %exit\n")
	buf.WriteString("}\n")
	return nil
}

func (g *LLVMIRGenerator) emitGlobalInit(buf *bytes.Buffer, globals []ir.VariableDecl, stringRefs map[string]string, functionsByName map[string]ir.Function, externsByName map[string]ir.ExternFunction, globalKinds map[string]ir.ValueKind) error {
	buf.WriteString("define void @jayess_init_globals() {\n")
	buf.WriteString("entry:\n")
	if len(globals) == 0 {
		buf.WriteString("  ret void\n")
		buf.WriteString("}\n\n")
		return nil
	}

	state := &functionState{
		slots:      map[string]variableSlot{},
		stringRefs: stringRefs,
		functions:  functionsByName,
		externs:    externsByName,
		globals:    globalKinds,
		isMain:     false,
	}
	for _, global := range globals {
		value, err := g.emitExpression(buf, state, global.Value)
		if err != nil {
			return err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr @jayess_global_%s\n", boxed, global.Name)
	}
	buf.WriteString("  ret void\n")
	buf.WriteString("}\n\n")
	return nil
}

func (g *LLVMIRGenerator) emitStatements(buf *bytes.Buffer, state *functionState, statements []ir.Statement) (bool, error) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ir.VariableDecl:
			value, err := g.emitExpression(buf, state, stmt.Value)
			if err != nil {
				return false, err
			}
			slot := state.nextTemp()
			typ := llvmStorageType(value.kind)
			fmt.Fprintf(buf, "  %s = alloca %s\n", slot, typ)
			fmt.Fprintf(buf, "  store %s %s, ptr %s\n", typ, value.ref, slot)
			state.slots[stmt.Name] = variableSlot{kind: value.kind, ptr: slot}
		case *ir.AssignmentStatement:
			if err := g.emitAssignment(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.ExpressionStatement:
			if _, err := g.emitExpression(buf, state, stmt.Expression); err != nil {
				return false, err
			}
		case *ir.ReturnStatement:
			value, err := g.emitExpression(buf, state, stmt.Value)
			if err != nil {
				return false, err
			}
			if state.isMain {
				numberRef, err := g.emitNumberOperand(buf, state, value)
				if err != nil {
					return false, err
				}
				fmt.Fprintf(buf, "  ret double %s\n", numberRef)
			} else {
				boxed, err := g.emitBoxedValue(buf, state, value)
				if err != nil {
					return false, err
				}
				fmt.Fprintf(buf, "  ret ptr %s\n", boxed)
			}
			return true, nil
		case *ir.IfStatement:
			if err := g.emitIf(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.WhileStatement:
			if err := g.emitWhile(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.ForStatement:
			if err := g.emitFor(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.BreakStatement:
			if len(state.loopStack) == 0 {
				return false, fmt.Errorf("break used outside loop")
			}
			fmt.Fprintf(buf, "  br label %%%s\n", state.loopStack[len(state.loopStack)-1].end)
			return true, nil
		case *ir.ContinueStatement:
			if len(state.loopStack) == 0 {
				return false, fmt.Errorf("continue used outside loop")
			}
			fmt.Fprintf(buf, "  br label %%%s\n", state.loopStack[len(state.loopStack)-1].continueTarget)
			return true, nil
		default:
			return false, fmt.Errorf("unsupported statement")
		}
	}
	return false, nil
}

func (g *LLVMIRGenerator) emitIf(buf *bytes.Buffer, state *functionState, stmt *ir.IfStatement) error {
	cond, err := g.emitCondition(buf, state, stmt.Condition)
	if err != nil {
		return err
	}
	thenLabel := state.nextLabel("if.then")
	elseLabel := state.nextLabel("if.else")
	endLabel := state.nextLabel("if.end")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, thenLabel, elseLabel)
	fmt.Fprintf(buf, "%s:\n", thenLabel)
	thenTerminated, err := g.emitStatements(buf, state, stmt.Consequence)
	if err != nil {
		return err
	}
	if !thenTerminated {
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	}
	fmt.Fprintf(buf, "%s:\n", elseLabel)
	elseTerminated, err := g.emitStatements(buf, state, stmt.Alternative)
	if err != nil {
		return err
	}
	if !elseTerminated {
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	}
	fmt.Fprintf(buf, "%s:\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitWhile(buf *bytes.Buffer, state *functionState, stmt *ir.WhileStatement) error {
	condLabel := state.nextLabel("while.cond")
	bodyLabel := state.nextLabel("while.body")
	endLabel := state.nextLabel("while.end")
	state.loopStack = append(state.loopStack, loopLabels{continueTarget: condLabel, end: endLabel})
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	cond, err := g.emitCondition(buf, state, stmt.Condition)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	terminated, err := g.emitStatements(buf, state, stmt.Body)
	if err != nil {
		return err
	}
	if !terminated {
		fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	}
	fmt.Fprintf(buf, "%s:\n", endLabel)
	state.loopStack = state.loopStack[:len(state.loopStack)-1]
	return nil
}

func (g *LLVMIRGenerator) emitFor(buf *bytes.Buffer, state *functionState, stmt *ir.ForStatement) error {
	if stmt.Init != nil {
		terminated, err := g.emitStatements(buf, state, []ir.Statement{stmt.Init})
		if err != nil {
			return err
		}
		if terminated {
			return fmt.Errorf("for initializer cannot terminate control flow")
		}
	}

	condLabel := state.nextLabel("for.cond")
	bodyLabel := state.nextLabel("for.body")
	updateLabel := state.nextLabel("for.update")
	endLabel := state.nextLabel("for.end")
	continueTarget := condLabel
	if stmt.Update != nil {
		continueTarget = updateLabel
	}

	state.loopStack = append(state.loopStack, loopLabels{continueTarget: continueTarget, end: endLabel})
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	if stmt.Condition != nil {
		cond, err := g.emitCondition(buf, state, stmt.Condition)
		if err != nil {
			return err
		}
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, bodyLabel, endLabel)
	} else {
		fmt.Fprintf(buf, "  br label %%%s\n", bodyLabel)
	}
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	terminated, err := g.emitStatements(buf, state, stmt.Body)
	if err != nil {
		return err
	}
	if !terminated {
		fmt.Fprintf(buf, "  br label %%%s\n", continueTarget)
	}
	if stmt.Update != nil {
		fmt.Fprintf(buf, "%s:\n", updateLabel)
		updateTerminated, err := g.emitStatements(buf, state, []ir.Statement{stmt.Update})
		if err != nil {
			return err
		}
		if updateTerminated {
			return fmt.Errorf("for update cannot terminate control flow")
		}
		fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	}
	fmt.Fprintf(buf, "%s:\n", endLabel)
	state.loopStack = state.loopStack[:len(state.loopStack)-1]
	return nil
}

func (g *LLVMIRGenerator) emitAssignment(buf *bytes.Buffer, state *functionState, stmt *ir.AssignmentStatement) error {
	value, err := g.emitExpression(buf, state, stmt.Value)
	if err != nil {
		return err
	}

	switch target := stmt.Target.(type) {
	case *ir.VariableRef:
		slot, ok := state.slots[target.Name]
		if ok {
			fmt.Fprintf(buf, "  store %s %s, ptr %s\n", llvmStorageType(slot.kind), value.ref, slot.ptr)
			return nil
		}
		if _, ok := state.globals[target.Name]; ok {
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, "  store ptr %s, ptr @jayess_global_%s\n", boxed, target.Name)
			return nil
		}
		return fmt.Errorf("unknown variable %s", target.Name)
	case *ir.MemberExpression:
		objectValue, err := g.emitExpression(buf, state, target.Target)
		if err != nil {
			return err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return err
		}
		if objectValue.kind == ir.ValueObject {
			fmt.Fprintf(buf, "  call void @jayess_object_set_value(ptr %s, ptr %s, ptr %s)\n", objectValue.ref, state.stringRefs[target.Property], boxed)
		} else {
			fmt.Fprintf(buf, "  call void @jayess_value_set_member(ptr %s, ptr %s, ptr %s)\n", objectValue.ref, state.stringRefs[target.Property], boxed)
		}
	case *ir.IndexExpression:
		arrayValue, err := g.emitExpression(buf, state, target.Target)
		if err != nil {
			return err
		}
		indexValue, err := g.emitExpression(buf, state, target.Index)
		if err != nil {
			return err
		}
		indexRef, err := g.emitNumberOperand(buf, state, indexValue)
		if err != nil {
			return err
		}
		indexInt := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", indexInt, indexRef)
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return err
		}
		if arrayValue.kind == ir.ValueArray {
			fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 %s, ptr %s)\n", arrayValue.ref, indexInt, boxed)
		} else {
			fmt.Fprintf(buf, "  call void @jayess_value_set_index(ptr %s, i32 %s, ptr %s)\n", arrayValue.ref, indexInt, boxed)
		}
	default:
		return fmt.Errorf("unsupported assignment target")
	}
	return nil
}

func (g *LLVMIRGenerator) emitExpression(buf *bytes.Buffer, state *functionState, expr ir.Expression) (emittedValue, error) {
	switch expr := expr.(type) {
	case *ir.NumberLiteral:
		return emittedValue{kind: ir.ValueNumber, ref: formatFloat(expr.Value)}, nil
	case *ir.BooleanLiteral:
		if expr.Value {
			return emittedValue{kind: ir.ValueBoolean, ref: "true"}, nil
		}
		return emittedValue{kind: ir.ValueBoolean, ref: "false"}, nil
	case *ir.NullLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_null()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case *ir.UndefinedLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case *ir.StringLiteral:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs[expr.Value]}, nil
	case *ir.ObjectLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_object_new()\n", tmp)
		for _, property := range expr.Properties {
			value, err := g.emitExpression(buf, state, property.Value)
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			fmt.Fprintf(buf, "  call void @jayess_object_set_value(ptr %s, ptr %s, ptr %s)\n", tmp, state.stringRefs[property.Key], boxed)
		}
		return emittedValue{kind: ir.ValueObject, ref: tmp}, nil
	case *ir.ArrayLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", tmp)
		for i, element := range expr.Elements {
			value, err := g.emitExpression(buf, state, element)
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 %d, ptr %s)\n", tmp, i, boxed)
		}
		return emittedValue{kind: ir.ValueArray, ref: tmp}, nil
	case *ir.VariableRef:
		slot, ok := state.slots[expr.Name]
		if ok {
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = load %s, ptr %s\n", tmp, llvmStorageType(slot.kind), slot.ptr)
			return emittedValue{kind: slot.kind, ref: tmp}, nil
		}
		if kind, ok := state.globals[expr.Name]; ok {
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = load ptr, ptr @jayess_global_%s\n", tmp, expr.Name)
			return emittedValue{kind: kind, ref: tmp}, nil
		}
		return emittedValue{}, fmt.Errorf("unknown variable %s", expr.Name)
	case *ir.BinaryExpression:
		left, err := g.emitExpression(buf, state, expr.Left)
		if err != nil {
			return emittedValue{}, err
		}
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		leftRef, err := g.emitNumberOperand(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		rightRef, err := g.emitNumberOperand(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		switch expr.Operator {
		case ir.OperatorAdd:
			fmt.Fprintf(buf, "  %s = fadd double %s, %s\n", tmp, leftRef, rightRef)
		case ir.OperatorSub:
			fmt.Fprintf(buf, "  %s = fsub double %s, %s\n", tmp, leftRef, rightRef)
		case ir.OperatorMul:
			fmt.Fprintf(buf, "  %s = fmul double %s, %s\n", tmp, leftRef, rightRef)
		case ir.OperatorDiv:
			fmt.Fprintf(buf, "  %s = fdiv double %s, %s\n", tmp, leftRef, rightRef)
		}
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case *ir.UnaryExpression:
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		cond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = xor i1 %s, true\n", tmp, cond)
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	case *ir.LogicalExpression:
		return g.emitLogical(buf, state, expr)
	case *ir.ComparisonExpression:
		return g.emitComparison(buf, state, expr)
	case *ir.IndexExpression:
		target, err := g.emitExpression(buf, state, expr.Target)
		if err != nil {
			return emittedValue{}, err
		}
		index, err := g.emitExpression(buf, state, expr.Index)
		if err != nil {
			return emittedValue{}, err
		}
		indexRef, err := g.emitNumberOperand(buf, state, index)
		if err != nil {
			return emittedValue{}, err
		}
		indexInt := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", indexInt, indexRef)
		tmp := state.nextTemp()
		if target.kind == ir.ValueArray {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_array_get(ptr %s, i32 %s)\n", tmp, target.ref, indexInt)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		if target.kind == ir.ValueDynamic {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %s)\n", tmp, target.ref, indexInt)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		fmt.Fprintf(buf, "  %s = call ptr @jayess_args_get(ptr %s, i32 %s)\n", tmp, target.ref, indexInt)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case *ir.MemberExpression:
		target, err := g.emitExpression(buf, state, expr.Target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		if target.kind == ir.ValueObject {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_object_get(ptr %s, ptr %s)\n", tmp, target.ref, state.stringRefs[expr.Property])
		} else {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, target.ref, state.stringRefs[expr.Property])
		}
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case *ir.CallExpression:
		return g.emitCall(buf, state, expr)
	default:
		return emittedValue{}, fmt.Errorf("unsupported expression")
	}
}

func (g *LLVMIRGenerator) emitLogical(buf *bytes.Buffer, state *functionState, expr *ir.LogicalExpression) (emittedValue, error) {
	left, err := g.emitExpression(buf, state, expr.Left)
	if err != nil {
		return emittedValue{}, err
	}
	leftCond, err := g.emitTruthyFromValue(buf, state, left)
	if err != nil {
		return emittedValue{}, err
	}

	rightLabel := state.nextLabel("logic.rhs")
	shortLabel := state.nextLabel("logic.short")
	endLabel := state.nextLabel("logic.end")

	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca i1\n", resultPtr)

	if expr.Operator == ir.OperatorAnd {
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", leftCond, rightLabel, shortLabel)
		fmt.Fprintf(buf, "%s:\n", rightLabel)
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		rightCond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store i1 %s, ptr %s\n", rightCond, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", shortLabel)
		fmt.Fprintf(buf, "  store i1 false, ptr %s\n", resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	} else {
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", leftCond, shortLabel, rightLabel)
		fmt.Fprintf(buf, "%s:\n", shortLabel)
		fmt.Fprintf(buf, "  store i1 true, ptr %s\n", resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", rightLabel)
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		rightCond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store i1 %s, ptr %s\n", rightCond, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	}

	fmt.Fprintf(buf, "%s:\n", endLabel)
	result := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load i1, ptr %s\n", result, resultPtr)
	return emittedValue{kind: ir.ValueBoolean, ref: result}, nil
}

func (g *LLVMIRGenerator) emitComparison(buf *bytes.Buffer, state *functionState, expr *ir.ComparisonExpression) (emittedValue, error) {
	left, err := g.emitExpression(buf, state, expr.Left)
	if err != nil {
		return emittedValue{}, err
	}
	right, err := g.emitExpression(buf, state, expr.Right)
	if err != nil {
		return emittedValue{}, err
	}

	// Bool comparisons stay boolean. Everything else compares as number-like.
	if left.kind == ir.ValueBoolean && right.kind == ir.ValueBoolean {
		tmp := state.nextTemp()
		pred := "eq"
		if expr.Operator == ir.OperatorNe {
			pred = "ne"
		}
		fmt.Fprintf(buf, "  %s = icmp %s i1 %s, %s\n", tmp, pred, left.ref, right.ref)
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	}
	if expr.Operator == ir.OperatorEq || expr.Operator == ir.OperatorNe {
		if left.kind == ir.ValueDynamic || right.kind == ir.ValueDynamic {
			leftBoxed, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			rightBoxed, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i1 @jayess_value_eq(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			if expr.Operator == ir.OperatorNe {
				neg := state.nextTemp()
				fmt.Fprintf(buf, "  %s = xor i1 %s, true\n", neg, tmp)
				return emittedValue{kind: ir.ValueBoolean, ref: neg}, nil
			}
			return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
		}
	}
	if (left.kind == ir.ValueString && right.kind == ir.ValueString) && (expr.Operator == ir.OperatorEq || expr.Operator == ir.OperatorNe) {
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_string_eq(ptr %s, ptr %s)\n", tmp, left.ref, right.ref)
		if expr.Operator == ir.OperatorNe {
			neg := state.nextTemp()
			fmt.Fprintf(buf, "  %s = xor i1 %s, true\n", neg, tmp)
			return emittedValue{kind: ir.ValueBoolean, ref: neg}, nil
		}
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	}

	leftRef, err := g.emitNumberOperand(buf, state, left)
	if err != nil {
		return emittedValue{}, err
	}
	rightRef, err := g.emitNumberOperand(buf, state, right)
	if err != nil {
		return emittedValue{}, err
	}
	tmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = fcmp %s double %s, %s\n", tmp, llvmCmpPredicate(expr.Operator), leftRef, rightRef)
	return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
}

func (g *LLVMIRGenerator) emitCall(buf *bytes.Buffer, state *functionState, call *ir.CallExpression) (emittedValue, error) {
	switch call.Callee {
	case "print":
		arg, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		switch arg.kind {
		case ir.ValueNumber:
			fmt.Fprintf(buf, "  call void @jayess_print_number(double %s)\n", arg.ref)
		case ir.ValueBoolean:
			fmt.Fprintf(buf, "  call void @jayess_print_bool(i1 %s)\n", arg.ref)
		case ir.ValueObject:
			fmt.Fprintf(buf, "  call void @jayess_print_object(ptr %s)\n", arg.ref)
		case ir.ValueArray:
			fmt.Fprintf(buf, "  call void @jayess_print_array(ptr %s)\n", arg.ref)
		case ir.ValueArgsArray:
			fmt.Fprintf(buf, "  call void @jayess_print_args(ptr %s)\n", arg.ref)
		case ir.ValueDynamic:
			fmt.Fprintf(buf, "  call void @jayess_print_value(ptr %s)\n", arg.ref)
		default:
			fmt.Fprintf(buf, "  call void @jayess_print_string(ptr %s)\n", arg.ref)
		}
		return emittedValue{}, nil
	case "readLine":
		prompt, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_read_line(ptr %s)\n", tmp, prompt.ref)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "readKey":
		prompt, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_read_key(ptr %s)\n", tmp, prompt.ref)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "sleep":
		arg, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		numberRef, err := g.emitNumberOperand(buf, state, arg)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", tmp, numberRef)
		fmt.Fprintf(buf, "  call void @jayess_sleep_ms(i32 %s)\n", tmp)
		return emittedValue{}, nil
	default:
		if ext, ok := state.externs[call.Callee]; ok {
			var args []string
			for _, argExpr := range call.Arguments {
				argValue, err := g.emitExpression(buf, state, argExpr)
				if err != nil {
					return emittedValue{}, err
				}
				boxed, err := g.emitBoxedValue(buf, state, argValue)
				if err != nil {
					return emittedValue{}, err
				}
				args = append(args, "ptr "+boxed)
			}
			tmp := state.nextTemp()
			if len(args) == 0 {
				fmt.Fprintf(buf, "  %s = call ptr @%s()\n", tmp, ext.SymbolName)
			} else {
				fmt.Fprintf(buf, "  %s = call ptr @%s(%s)\n", tmp, ext.SymbolName, strings.Join(args, ", "))
			}
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		fn, ok := state.functions[call.Callee]
		if !ok {
			return emittedValue{}, fmt.Errorf("unknown function %s", call.Callee)
		}
		var args []string
		for _, argExpr := range call.Arguments {
			argValue, err := g.emitExpression(buf, state, argExpr)
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, argValue)
			if err != nil {
				return emittedValue{}, err
			}
			args = append(args, fmt.Sprintf("ptr %s", boxed))
		}
		tmp := state.nextTemp()
		if len(fn.Params) == 0 {
			fmt.Fprintf(buf, "  %s = call ptr @%s()\n", tmp, call.Callee)
		} else {
			fmt.Fprintf(buf, "  %s = call ptr @%s(%s)\n", tmp, call.Callee, strings.Join(args, ", "))
		}
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	}
}

func (g *LLVMIRGenerator) emitCondition(buf *bytes.Buffer, state *functionState, expr ir.Expression) (string, error) {
	value, err := g.emitExpression(buf, state, expr)
	if err != nil {
		return "", err
	}
	return g.emitTruthyFromValue(buf, state, value)
}

func (g *LLVMIRGenerator) emitTruthyFromValue(buf *bytes.Buffer, state *functionState, value emittedValue) (string, error) {
	switch value.kind {
	case ir.ValueBoolean:
		return value.ref, nil
	case ir.ValueNumber:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fcmp one double %s, 0.000000\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueString:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_string_is_truthy(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueArgsArray:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_args_is_truthy(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueObject, ir.ValueArray:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = icmp ne ptr %s, null\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueDynamic:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_value_is_truthy(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	default:
		return "", fmt.Errorf("unsupported condition kind %s", value.kind)
	}
}

func (g *LLVMIRGenerator) emitBoxedValue(buf *bytes.Buffer, state *functionState, value emittedValue) (string, error) {
	switch value.kind {
	case ir.ValueDynamic:
		return value.ref, nil
	case ir.ValueString:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_string(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueNumber:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_number(double %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueBoolean:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_bool(i1 %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueObject:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueArray:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueNull:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_null()\n", tmp)
		return tmp, nil
	case ir.ValueUndefined:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", tmp)
		return tmp, nil
	default:
		return "", fmt.Errorf("cannot box value kind %s", value.kind)
	}
}

func (g *LLVMIRGenerator) emitNumberOperand(buf *bytes.Buffer, state *functionState, value emittedValue) (string, error) {
	switch value.kind {
	case ir.ValueNumber:
		return value.ref, nil
	case ir.ValueBoolean:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = uitofp i1 %s to double\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueDynamic:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_value_to_number(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	default:
		return "", fmt.Errorf("value kind %s cannot be used as a number", value.kind)
	}
}

func (s *functionState) nextTemp() string {
	name := fmt.Sprintf("%%tmp.%d", s.tempCounter)
	s.tempCounter++
	return name
}

func (s *functionState) nextLabel(prefix string) string {
	name := fmt.Sprintf("%s.%d", prefix, s.labelCounter)
	s.labelCounter++
	return name
}

func llvmStorageType(kind ir.ValueKind) string {
	switch kind {
	case ir.ValueNumber:
		return "double"
	case ir.ValueBoolean:
		return "i1"
	default:
		return "ptr"
	}
}

func llvmCmpPredicate(op ir.ComparisonOperator) string {
	switch op {
	case ir.OperatorEq:
		return "oeq"
	case ir.OperatorNe:
		return "one"
	case ir.OperatorLt:
		return "olt"
	case ir.OperatorLte:
		return "ole"
	case ir.OperatorGt:
		return "ogt"
	default:
		return "oge"
	}
}

func collectStrings(module *ir.Module) []string {
	seen := map[string]bool{}
	var out []string
	for _, global := range module.Globals {
		collectStringsFromExpression(global.Value, seen, &out)
	}
	for _, fn := range module.Functions {
		for _, stmt := range fn.Body {
			collectStringsFromStatement(stmt, seen, &out)
		}
	}
	return out
}

func collectStringsFromStatement(stmt ir.Statement, seen map[string]bool, out *[]string) {
	switch stmt := stmt.(type) {
	case *ir.VariableDecl:
		collectStringsFromExpression(stmt.Value, seen, out)
	case *ir.AssignmentStatement:
		collectStringsFromExpression(stmt.Target, seen, out)
		collectStringsFromExpression(stmt.Value, seen, out)
	case *ir.ReturnStatement:
		collectStringsFromExpression(stmt.Value, seen, out)
	case *ir.ExpressionStatement:
		collectStringsFromExpression(stmt.Expression, seen, out)
	case *ir.IfStatement:
		collectStringsFromExpression(stmt.Condition, seen, out)
		for _, child := range stmt.Consequence {
			collectStringsFromStatement(child, seen, out)
		}
		for _, child := range stmt.Alternative {
			collectStringsFromStatement(child, seen, out)
		}
	case *ir.WhileStatement:
		collectStringsFromExpression(stmt.Condition, seen, out)
		for _, child := range stmt.Body {
			collectStringsFromStatement(child, seen, out)
		}
	case *ir.ForStatement:
		if stmt.Init != nil {
			collectStringsFromStatement(stmt.Init, seen, out)
		}
		if stmt.Condition != nil {
			collectStringsFromExpression(stmt.Condition, seen, out)
		}
		if stmt.Update != nil {
			collectStringsFromStatement(stmt.Update, seen, out)
		}
		for _, child := range stmt.Body {
			collectStringsFromStatement(child, seen, out)
		}
	}
}

func collectStringsFromExpression(expr ir.Expression, seen map[string]bool, out *[]string) {
	switch expr := expr.(type) {
	case *ir.StringLiteral:
		addString(expr.Value, seen, out)
	case *ir.ObjectLiteral:
		for _, property := range expr.Properties {
			addString(property.Key, seen, out)
			collectStringsFromExpression(property.Value, seen, out)
		}
	case *ir.ArrayLiteral:
		for _, element := range expr.Elements {
			collectStringsFromExpression(element, seen, out)
		}
	case *ir.MemberExpression:
		addString(expr.Property, seen, out)
		collectStringsFromExpression(expr.Target, seen, out)
	case *ir.BinaryExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.ComparisonExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.LogicalExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.UnaryExpression:
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.IndexExpression:
		collectStringsFromExpression(expr.Target, seen, out)
		collectStringsFromExpression(expr.Index, seen, out)
	case *ir.CallExpression:
		for _, arg := range expr.Arguments {
			collectStringsFromExpression(arg, seen, out)
		}
	}
}

func addString(value string, seen map[string]bool, out *[]string) {
	if !seen[value] {
		seen[value] = true
		*out = append(*out, value)
	}
}

func buildStringRefMap(values []string) map[string]string {
	result := map[string]string{}
	for i, value := range values {
		result[value] = fmt.Sprintf("@.str.%d", i)
	}
	return result
}

func findMain(functions []ir.Function) ir.Function {
	for _, fn := range functions {
		if fn.Name == "main" {
			return fn
		}
	}
	return ir.Function{}
}

func formatFloat(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0.0"
	}
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func encodeLLVMString(input string) (string, int) {
	var b strings.Builder
	length := 0
	for _, r := range input {
		switch r {
		case '\\':
			b.WriteString("\\5C")
		case '"':
			b.WriteString("\\22")
		case '\n':
			b.WriteString("\\0A")
		case '\r':
			b.WriteString("\\0D")
		case '\t':
			b.WriteString("\\09")
		default:
			if r < 32 || r > 126 {
				b.WriteString(fmt.Sprintf("\\%02X", r))
			} else {
				b.WriteRune(r)
			}
		}
		length++
	}
	return b.String(), length
}
