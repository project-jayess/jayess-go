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
	tempCounter     int
	labelCounter    int
	slots           map[string]variableSlot
	stringRefs      map[string]string
	loopStack       []loopLabels
	functions       map[string]ir.Function
	externs         map[string]ir.ExternFunction
	globals         map[string]ir.ValueKind
	classNames      map[string]bool
	functionName    string
	isMain          bool
	exceptionTarget string
}

type loopLabels struct {
	continueTarget string
	end            string
}

func validateClassLayout(module *ir.Module) error {
	functions := map[string]ir.Function{}
	for _, fn := range module.Functions {
		functions[fn.Name] = fn
	}
	globals := map[string]bool{}
	for _, global := range module.Globals {
		globals[global.Name] = true
	}
	classes := map[string]ir.ClassDecl{}
	for _, classDecl := range module.Classes {
		classes[classDecl.Name] = classDecl
	}
	for _, classDecl := range module.Classes {
		if classDecl.SuperClass != "" {
			if _, ok := classes[classDecl.SuperClass]; !ok {
				return fmt.Errorf("codegen class validation failed: class %s extends unknown class %s", classDecl.Name, classDecl.SuperClass)
			}
		}
		if _, ok := functions[classDecl.Name]; !ok {
			return fmt.Errorf("codegen class validation failed: missing lowered constructor for class %s", classDecl.Name)
		}
		for _, field := range classDecl.Fields {
			if !field.Static {
				continue
			}
			name := classStaticSymbol(classDecl.Name, field.Name, field.Private)
			if !globals[name] {
				return fmt.Errorf("codegen class validation failed: missing lowered static field %s for class %s", field.Name, classDecl.Name)
			}
		}
		for _, method := range classDecl.Methods {
			if method.IsConstructor {
				continue
			}
			name := classMethodSymbol(classDecl.Name, method.Name, method.Private, method.Static)
			fn, ok := functions[name]
			if !ok {
				return fmt.Errorf("codegen class validation failed: missing lowered method %s for class %s", method.Name, classDecl.Name)
			}
			expectedParams := method.ParamCount
			if !method.Static {
				expectedParams++
			}
			if len(fn.Params) != expectedParams {
				return fmt.Errorf("codegen class validation failed: lowered method %s for class %s has %d params, expected %d", method.Name, classDecl.Name, len(fn.Params), expectedParams)
			}
		}
	}
	return nil
}

func emitClassMetadata(buf *bytes.Buffer, classes []ir.ClassDecl) {
	if len(classes) == 0 {
		return
	}
	fmt.Fprintf(buf, "; class metadata\n")
	for _, classDecl := range classes {
		super := "none"
		if classDecl.SuperClass != "" {
			super = classDecl.SuperClass
		}
		fmt.Fprintf(buf, "; class %s extends %s\n", classDecl.Name, super)
		fmt.Fprintf(buf, ";   fields=%d methods=%d\n", len(classDecl.Fields), len(classDecl.Methods))
		for _, field := range classDecl.Fields {
			kind := "instance"
			if field.Static {
				kind = "static"
			}
			if field.Private {
				kind += " private"
			}
			initFlag := "noinit"
			if field.HasInitializer {
				initFlag = "init"
			}
			fmt.Fprintf(buf, ";   field %s [%s %s]\n", field.Name, kind, initFlag)
		}
		for _, method := range classDecl.Methods {
			kind := "instance"
			if method.Static {
				kind = "static"
			}
			if method.Private {
				kind += " private"
			}
			if method.IsConstructor {
				kind = "constructor"
			}
			fmt.Fprintf(buf, ";   method %s [%s params=%d]\n", method.Name, kind, method.ParamCount)
		}
	}
	buf.WriteString("\n")
}

func classMethodSymbol(className, methodName string, private, static bool) string {
	if static {
		return classStaticSymbol(className, methodName, private)
	}
	if private {
		return fmt.Sprintf("%s__private__%s", className, methodName)
	}
	return fmt.Sprintf("%s__%s", className, methodName)
}

func classStaticSymbol(className, name string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, name)
	}
	return fmt.Sprintf("%s__%s", className, name)
}

func (g *LLVMIRGenerator) Generate(module *ir.Module, targetTriple string) ([]byte, error) {
	if targetTriple == "" {
		return nil, fmt.Errorf("target triple is required")
	}
	if err := validateClassLayout(module); err != nil {
		return nil, err
	}

	stringsPool := collectStrings(module)
	stringRefs := buildStringRefMap(stringsPool)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "; jayess module\n")
	fmt.Fprintf(&buf, "target triple = %q\n\n", targetTriple)
	emitClassMetadata(&buf, module.Classes)

	for i, value := range stringsPool {
		encoded, length := encodeLLVMString(value)
		fmt.Fprintf(&buf, "@.str.%d = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n", i, length+1, encoded)
	}
	if len(stringsPool) > 0 {
		fmt.Fprintln(&buf)
	}

	globalKinds := map[string]ir.ValueKind{}
	classNames := buildClassNames(module.Classes)
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
	fmt.Fprintf(&buf, "declare void @jayess_print_many(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_console_log(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_console_warn(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_console_error(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_cwd()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_env(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_argv()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_platform()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_arch()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_exit(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_join(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_normalize(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_resolve(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_relative(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_parse(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_is_absolute(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_format(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_sep()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_delimiter()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_basename(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_dirname(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_path_extname(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_read_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_write_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_exists(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_read_dir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_stat(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_mkdir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_remove(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_copy_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_copy_dir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_rename(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_line(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_key(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_sleep_ms(i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_make_args(i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_args_get(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_args_length(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_object_new()\n")
	fmt.Fprintf(&buf, "declare void @jayess_object_set_value(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_object_get(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_object_delete(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_object_keys(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_member(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_get_member(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_delete_member(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_keys(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_rest(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_values(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_entries(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_assign(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_has_own(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_number_is_nan(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_number_is_finite(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_string_from_char_code(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_array_is_array(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_array_from(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_array_of(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_object_from_entries(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_map_new()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_set_new()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_date_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_date_now()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_regexp_new(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_json_stringify(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_json_parse(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_iterable_values(ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_floor(double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_ceil(double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_round(double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_min(double, double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_max(double, double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_abs(double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_pow(double, double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_sqrt(double)\n")
	fmt.Fprintf(&buf, "declare double @jayess_math_random()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_new()\n")
	fmt.Fprintf(&buf, "declare void @jayess_array_set_value(ptr, i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_get(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_array_length(ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_array_push_value(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_pop_value(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_shift_value(ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_array_unshift_value(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_array_slice_values(ptr, i32, i32, i1)\n")
	fmt.Fprintf(&buf, "declare void @jayess_array_append_array(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_index(ptr, i32, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_get_index(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_dynamic_index(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_get_dynamic_index(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_delete_dynamic_index(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_value_array_length(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_array_push(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_array_pop(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_array_shift(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_array_unshift(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_array_slice(ptr, i32, i32, i1)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_string(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_stringify(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_template_string(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_concat_values(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_set_computed_member(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_number(double)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_bool(i1)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_object(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_array(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_args(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_function(ptr, ptr, ptr, ptr, i32, i1)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_null()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_undefined()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_ptr(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_env(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bind(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_bound_this(ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_value_to_number(ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_value_function_param_count(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_function_has_rest(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_is_nullish(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_args_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_merge_bound_args(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_throw(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_has_exception()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_take_exception()\n")
	fmt.Fprintf(&buf, "declare void @jayess_report_uncaught_exception()\n")
	fmt.Fprintf(&buf, "declare void @jayess_push_this(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_pop_this()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_current_this()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_typeof(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_class_name(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_instanceof(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_is_truthy(ptr)\n\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_as_array(ptr)\n\n")

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

	if err := g.emitGlobalInit(&buf, module.Globals, stringRefs, functionsByName, externsByName, globalKinds, classNames); err != nil {
		return nil, err
	}

	for _, fn := range module.Functions {
		if err := g.emitFunction(&buf, fn, stringRefs, functionsByName, externsByName, globalKinds, classNames); err != nil {
			return nil, err
		}
	}

	if err := g.emitEntryWrapper(&buf, findMain(module.Functions)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *LLVMIRGenerator) emitFunction(buf *bytes.Buffer, fn ir.Function, stringRefs map[string]string, functionsByName map[string]ir.Function, externsByName map[string]ir.ExternFunction, globalKinds map[string]ir.ValueKind, classNames map[string]bool) error {
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
		slots:        map[string]variableSlot{},
		stringRefs:   stringRefs,
		functions:    functionsByName,
		externs:      externsByName,
		globals:      globalKinds,
		classNames:   classNames,
		functionName: fn.Name,
		isMain:       fn.Name == "main",
	}
	uncaughtLabel := state.nextLabel("throw.uncaught")
	state.exceptionTarget = uncaughtLabel

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
	fmt.Fprintf(buf, "%s:\n", uncaughtLabel)
	if state.isMain {
		buf.WriteString("  ret double 0.000000\n")
	} else {
		buf.WriteString("  %tmp.throw = call ptr @jayess_value_undefined()\n")
		buf.WriteString("  ret ptr %tmp.throw\n")
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
	buf.WriteString("  %thrown = call i1 @jayess_has_exception()\n")
	buf.WriteString("  br i1 %thrown, label %uncaught, label %exit.ok\n")
	buf.WriteString("uncaught:\n")
	buf.WriteString("  call void @jayess_report_uncaught_exception()\n")
	buf.WriteString("  ret i32 1\n")
	buf.WriteString("exit.ok:\n")
	buf.WriteString("  %exit = fptosi double %result to i32\n")
	buf.WriteString("  ret i32 %exit\n")
	buf.WriteString("}\n")
	return nil
}

func (g *LLVMIRGenerator) emitGlobalInit(buf *bytes.Buffer, globals []ir.VariableDecl, stringRefs map[string]string, functionsByName map[string]ir.Function, externsByName map[string]ir.ExternFunction, globalKinds map[string]ir.ValueKind, classNames map[string]bool) error {
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
		classNames: classNames,
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
			storageKind := value.kind
			if stmt.Kind == ir.DeclarationVar {
				storageKind = ir.ValueDynamic
			}
			typ := llvmStorageType(storageKind)
			fmt.Fprintf(buf, "  %s = alloca %s\n", slot, typ)
			storeRef := value.ref
			if storageKind == ir.ValueDynamic {
				boxed, err := g.emitBoxedValue(buf, state, value)
				if err != nil {
					return false, err
				}
				storeRef = boxed
			}
			fmt.Fprintf(buf, "  store %s %s, ptr %s\n", typ, storeRef, slot)
			state.slots[stmt.Name] = variableSlot{kind: storageKind, ptr: slot}
		case *ir.AssignmentStatement:
			if err := g.emitAssignment(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.ExpressionStatement:
			if _, err := g.emitExpression(buf, state, stmt.Expression); err != nil {
				return false, err
			}
		case *ir.DeleteStatement:
			if err := g.emitDelete(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.ThrowStatement:
			if err := g.emitThrow(buf, state, stmt); err != nil {
				return false, err
			}
			return true, nil
		case *ir.TryStatement:
			if err := g.emitTry(buf, state, stmt); err != nil {
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
		if state.exceptionTarget != "" {
			g.emitExceptionCheck(buf, state, state.exceptionTarget)
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

func (g *LLVMIRGenerator) emitThrow(buf *bytes.Buffer, state *functionState, stmt *ir.ThrowStatement) error {
	value, err := g.emitExpression(buf, state, stmt.Value)
	if err != nil {
		return err
	}
	boxed, err := g.emitBoxedValue(buf, state, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "  call void @jayess_throw(ptr %s)\n", boxed)
	if state.exceptionTarget == "" {
		return fmt.Errorf("throw used without an exception target")
	}
	fmt.Fprintf(buf, "  br label %%%s\n", state.exceptionTarget)
	return nil
}

func (g *LLVMIRGenerator) emitTry(buf *bytes.Buffer, state *functionState, stmt *ir.TryStatement) error {
	tryLabel := state.nextLabel("try.body")
	endLabel := state.nextLabel("try.end")
	finallyLabel := ""
	catchLabel := ""
	tryExceptionTarget := state.exceptionTarget
	if len(stmt.FinallyBody) > 0 {
		finallyLabel = state.nextLabel("try.finally")
		tryExceptionTarget = finallyLabel
	}
	if len(stmt.CatchBody) > 0 {
		catchLabel = state.nextLabel("try.catch")
		tryExceptionTarget = catchLabel
	}
	if len(stmt.CatchBody) == 0 && len(stmt.FinallyBody) > 0 {
		tryExceptionTarget = finallyLabel
	}

	fmt.Fprintf(buf, "  br label %%%s\n", tryLabel)
	fmt.Fprintf(buf, "%s:\n", tryLabel)
	outerTarget := state.exceptionTarget
	state.exceptionTarget = tryExceptionTarget
	tryTerminated, err := g.emitStatements(buf, state, stmt.TryBody)
	if err != nil {
		return err
	}
	state.exceptionTarget = outerTarget
	if !tryTerminated {
		if finallyLabel != "" {
			fmt.Fprintf(buf, "  br label %%%s\n", finallyLabel)
		} else {
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		}
	}

	if catchLabel != "" {
		fmt.Fprintf(buf, "%s:\n", catchLabel)
		catchSlot := ""
		if stmt.CatchName != "" {
			catchSlot = state.nextTemp()
			fmt.Fprintf(buf, "  %s = alloca ptr\n", catchSlot)
			caught := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_take_exception()\n", caught)
			fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", caught, catchSlot)
			state.slots[stmt.CatchName] = variableSlot{kind: ir.ValueDynamic, ptr: catchSlot}
		} else {
			discard := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_take_exception()\n", discard)
		}
		catchTarget := outerTarget
		if finallyLabel != "" {
			catchTarget = finallyLabel
		}
		state.exceptionTarget = catchTarget
		catchTerminated, err := g.emitStatements(buf, state, stmt.CatchBody)
		if stmt.CatchName != "" {
			delete(state.slots, stmt.CatchName)
		}
		if err != nil {
			return err
		}
		state.exceptionTarget = outerTarget
		if !catchTerminated {
			if finallyLabel != "" {
				fmt.Fprintf(buf, "  br label %%%s\n", finallyLabel)
			} else {
				fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
			}
		}
	}

	if finallyLabel != "" {
		fmt.Fprintf(buf, "%s:\n", finallyLabel)
		state.exceptionTarget = outerTarget
		finallyTerminated, err := g.emitStatements(buf, state, stmt.FinallyBody)
		if err != nil {
			return err
		}
		state.exceptionTarget = outerTarget
		if !finallyTerminated {
			if outerTarget != "" {
				g.emitExceptionCheck(buf, state, outerTarget)
			}
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		}
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
			storeRef := value.ref
			if slot.kind == ir.ValueDynamic {
				boxed, err := g.emitBoxedValue(buf, state, value)
				if err != nil {
					return err
				}
				storeRef = boxed
			}
			fmt.Fprintf(buf, "  store %s %s, ptr %s\n", llvmStorageType(slot.kind), storeRef, slot.ptr)
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
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return err
		}
		if indexValue.kind == ir.ValueString {
			if arrayValue.kind == ir.ValueObject {
				fmt.Fprintf(buf, "  call void @jayess_object_set_value(ptr %s, ptr %s, ptr %s)\n", arrayValue.ref, indexValue.ref, boxed)
			} else {
				fmt.Fprintf(buf, "  call void @jayess_value_set_member(ptr %s, ptr %s, ptr %s)\n", arrayValue.ref, indexValue.ref, boxed)
			}
			return nil
		}
		if indexValue.kind == ir.ValueDynamic {
			fmt.Fprintf(buf, "  call void @jayess_value_set_dynamic_index(ptr %s, ptr %s, ptr %s)\n", arrayValue.ref, indexValue.ref, boxed)
			return nil
		}
		indexRef, err := g.emitNumberOperand(buf, state, indexValue)
		if err != nil {
			return err
		}
		indexInt := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", indexInt, indexRef)
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

func (g *LLVMIRGenerator) emitDelete(buf *bytes.Buffer, state *functionState, stmt *ir.DeleteStatement) error {
	switch target := stmt.Target.(type) {
	case *ir.MemberExpression:
		objectValue, err := g.emitExpression(buf, state, target.Target)
		if err != nil {
			return err
		}
		if objectValue.kind == ir.ValueObject {
			fmt.Fprintf(buf, "  call void @jayess_object_delete(ptr %s, ptr %s)\n", objectValue.ref, state.stringRefs[target.Property])
		} else {
			fmt.Fprintf(buf, "  call void @jayess_value_delete_member(ptr %s, ptr %s)\n", objectValue.ref, state.stringRefs[target.Property])
		}
		return nil
	case *ir.IndexExpression:
		objectValue, err := g.emitExpression(buf, state, target.Target)
		if err != nil {
			return err
		}
		indexValue, err := g.emitExpression(buf, state, target.Index)
		if err != nil {
			return err
		}
		if indexValue.kind == ir.ValueString {
			if objectValue.kind == ir.ValueObject {
				fmt.Fprintf(buf, "  call void @jayess_object_delete(ptr %s, ptr %s)\n", objectValue.ref, indexValue.ref)
			} else {
				fmt.Fprintf(buf, "  call void @jayess_value_delete_member(ptr %s, ptr %s)\n", objectValue.ref, indexValue.ref)
			}
			return nil
		}
		fmt.Fprintf(buf, "  call void @jayess_value_delete_dynamic_index(ptr %s, ptr %s)\n", objectValue.ref, indexValue.ref)
		return nil
	default:
		return fmt.Errorf("unsupported delete target")
	}
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
			if property.Computed {
				keyValue, err := g.emitExpression(buf, state, property.KeyExpr)
				if err != nil {
					return emittedValue{}, err
				}
				boxedKey, err := g.emitBoxedValue(buf, state, keyValue)
				if err != nil {
					return emittedValue{}, err
				}
				boxedObject := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxedObject, tmp)
				fmt.Fprintf(buf, "  call void @jayess_value_set_computed_member(ptr %s, ptr %s, ptr %s)\n", boxedObject, boxedKey, boxed)
			} else {
				fmt.Fprintf(buf, "  call void @jayess_object_set_value(ptr %s, ptr %s, ptr %s)\n", tmp, state.stringRefs[property.Key], boxed)
			}
		}
		return emittedValue{kind: ir.ValueObject, ref: tmp}, nil
	case *ir.ArrayLiteral:
		tmp, err := g.emitArrayRefFromExpressions(buf, state, expr.Elements)
		if err != nil {
			return emittedValue{}, err
		}
		return emittedValue{kind: ir.ValueArray, ref: tmp}, nil
	case *ir.TemplateLiteral:
		partsArray := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", partsArray)
		for i, part := range expr.Parts {
			boxedPart := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_string(ptr %s)\n", boxedPart, state.stringRefs[part])
			fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 %d, ptr %s)\n", partsArray, i, boxedPart)
		}
		valuesArray := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", valuesArray)
		for i, valueExpr := range expr.Values {
			value, err := g.emitExpression(buf, state, valueExpr)
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 %d, ptr %s)\n", valuesArray, i, boxed)
		}
		partsBoxed := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", partsBoxed, partsArray)
		valuesBoxed := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", valuesBoxed, valuesArray)
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_template_string(ptr %s, ptr %s)\n", tmp, partsBoxed, valuesBoxed)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case *ir.SpreadExpression:
		return emittedValue{}, fmt.Errorf("spread expressions are only valid inside arrays and call arguments")
	case *ir.FunctionValue:
		envRef := "null"
		if expr.Environment != nil {
			envValue, err := g.emitExpression(buf, state, expr.Environment)
			if err != nil {
				return emittedValue{}, err
			}
			boxedEnv, err := g.emitBoxedValue(buf, state, envValue)
			if err != nil {
				return emittedValue{}, err
			}
			envRef = boxedEnv
		}
		classRef := "null"
		if state.classNames[expr.Name] {
			classRef = state.stringRefs[expr.Name]
		}
		paramCount, hasRest := functionMetadata(expr.Name, state.functions, state.externs)
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr %s, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, expr.Name, envRef, state.stringRefs[expr.Name], classRef, paramCount, hasRest)
		return emittedValue{kind: ir.ValueFunction, ref: tmp}, nil
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
		if expr.Operator == ir.OperatorAdd && (left.kind == ir.ValueString || right.kind == ir.ValueString) {
			leftBoxed, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			rightBoxed, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_concat_values(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			return emittedValue{kind: ir.ValueString, ref: tmp}, nil
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
	case *ir.NullishCoalesceExpression:
		left, err := g.emitExpression(buf, state, expr.Left)
		if err != nil {
			return emittedValue{}, err
		}
		resultPtr := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
		useRightLabel := state.nextLabel("nullish.right")
		endLabel := state.nextLabel("nullish.end")
		leftNullish, err := g.emitNullishCheck(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", leftNullish, useRightLabel, endLabel)
		boxedLeft, err := g.emitBoxedValue(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxedLeft, resultPtr)
		fmt.Fprintf(buf, "%s:\n", useRightLabel)
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		boxedRight, err := g.emitBoxedValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxedRight, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", endLabel)
		result := state.nextTemp()
		fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", result, resultPtr)
		return emittedValue{kind: ir.ValueDynamic, ref: result}, nil
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
	case *ir.NewTargetExpression:
		if state.classNames[state.functionName] {
			paramCount, hasRest := functionMetadata(state.functionName, state.functions, state.externs)
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr null, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, state.functionName, state.stringRefs[state.functionName], state.stringRefs[state.functionName], paramCount, hasRest)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case *ir.TypeofExpression:
		return g.emitTypeof(buf, state, expr)
	case *ir.InstanceofExpression:
		return g.emitInstanceof(buf, state, expr)
	case *ir.LogicalExpression:
		return g.emitLogical(buf, state, expr)
	case *ir.ComparisonExpression:
		return g.emitComparison(buf, state, expr)
	case *ir.IndexExpression:
		target, err := g.emitExpression(buf, state, expr.Target)
		if err != nil {
			return emittedValue{}, err
		}
		if expr.Optional {
			if nullish, err := g.emitNullishCheck(buf, state, target); err == nil {
				nilLabel := state.nextLabel("optidx.nil")
				valLabel := state.nextLabel("optidx.val")
				endLabel := state.nextLabel("optidx.end")
				resultPtr := state.nextTemp()
				fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
				fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", nullish, nilLabel, valLabel)
				fmt.Fprintf(buf, "%s:\n", nilLabel)
				undef := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
				fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
				fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
				fmt.Fprintf(buf, "%s:\n", valLabel)
				index, err := g.emitExpression(buf, state, expr.Index)
				if err != nil {
					return emittedValue{}, err
				}
				res, err := g.emitIndexAccess(buf, state, target, index)
				if err != nil {
					return emittedValue{}, err
				}
				boxed, err := g.emitBoxedValue(buf, state, res)
				if err != nil {
					return emittedValue{}, err
				}
				fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxed, resultPtr)
				fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
				fmt.Fprintf(buf, "%s:\n", endLabel)
				out := state.nextTemp()
				fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", out, resultPtr)
				return emittedValue{kind: ir.ValueDynamic, ref: out}, nil
			}
		}
		index, err := g.emitExpression(buf, state, expr.Index)
		if err != nil {
			return emittedValue{}, err
		}
		return g.emitIndexAccess(buf, state, target, index)
	case *ir.MemberExpression:
		target, err := g.emitExpression(buf, state, expr.Target)
		if err != nil {
			return emittedValue{}, err
		}
		if expr.Optional {
			nullish, err := g.emitNullishCheck(buf, state, target)
			if err != nil {
				return emittedValue{}, err
			}
			nilLabel := state.nextLabel("optmem.nil")
			valLabel := state.nextLabel("optmem.val")
			endLabel := state.nextLabel("optmem.end")
			resultPtr := state.nextTemp()
			fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
			fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", nullish, nilLabel, valLabel)
			fmt.Fprintf(buf, "%s:\n", nilLabel)
			undef := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
			fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
			fmt.Fprintf(buf, "%s:\n", valLabel)
			res, err := g.emitMemberAccess(buf, state, target, expr.Property)
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, res)
			if err != nil {
				return emittedValue{}, err
			}
			fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxed, resultPtr)
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
			fmt.Fprintf(buf, "%s:\n", endLabel)
			out := state.nextTemp()
			fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", out, resultPtr)
			return emittedValue{kind: ir.ValueDynamic, ref: out}, nil
		}
		return g.emitMemberAccess(buf, state, target, expr.Property)
	case *ir.CallExpression:
		return g.emitCall(buf, state, expr)
	case *ir.InvokeExpression:
		return g.emitInvoke(buf, state, expr)
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

func (g *LLVMIRGenerator) emitTypeof(buf *bytes.Buffer, state *functionState, expr *ir.TypeofExpression) (emittedValue, error) {
	value, err := g.emitExpression(buf, state, expr.Value)
	if err != nil {
		return emittedValue{}, err
	}
	switch value.kind {
	case ir.ValueNumber:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["number"]}, nil
	case ir.ValueBoolean:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["boolean"]}, nil
	case ir.ValueString:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["string"]}, nil
	case ir.ValueFunction:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["function"]}, nil
	case ir.ValueObject, ir.ValueArray, ir.ValueArgsArray, ir.ValueNull:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["object"]}, nil
	case ir.ValueUndefined:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["undefined"]}, nil
	case ir.ValueDynamic:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_typeof(ptr %s)\n", tmp, value.ref)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	default:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["undefined"]}, nil
	}
}

func (g *LLVMIRGenerator) emitInstanceof(buf *bytes.Buffer, state *functionState, expr *ir.InstanceofExpression) (emittedValue, error) {
	left, err := g.emitExpression(buf, state, expr.Left)
	if err != nil {
		return emittedValue{}, err
	}
	leftBoxed, err := g.emitBoxedValue(buf, state, left)
	if err != nil {
		return emittedValue{}, err
	}

	classNameRef := ""
	if expr.ClassName != "" {
		classNameRef = state.stringRefs[expr.ClassName]
	} else {
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		rightBoxed, err := g.emitBoxedValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		classNameRef = state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_class_name(ptr %s)\n", classNameRef, rightBoxed)
	}
	tmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_value_instanceof(ptr %s, ptr %s)\n", tmp, leftBoxed, classNameRef)
	return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
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
	if expr.Operator == ir.OperatorEq || expr.Operator == ir.OperatorNe || expr.Operator == ir.OperatorStrictEq || expr.Operator == ir.OperatorStrictNe {
		if left.kind == ir.ValueDynamic || right.kind == ir.ValueDynamic || left.kind == ir.ValueFunction || right.kind == ir.ValueFunction {
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
			if expr.Operator == ir.OperatorNe || expr.Operator == ir.OperatorStrictNe {
				neg := state.nextTemp()
				fmt.Fprintf(buf, "  %s = xor i1 %s, true\n", neg, tmp)
				return emittedValue{kind: ir.ValueBoolean, ref: neg}, nil
			}
			return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
		}
	}
	if (left.kind == ir.ValueString && right.kind == ir.ValueString) && (expr.Operator == ir.OperatorEq || expr.Operator == ir.OperatorNe || expr.Operator == ir.OperatorStrictEq || expr.Operator == ir.OperatorStrictNe) {
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_string_eq(ptr %s, ptr %s)\n", tmp, left.ref, right.ref)
		if expr.Operator == ir.OperatorNe || expr.Operator == ir.OperatorStrictNe {
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
		if len(call.Arguments) == 1 {
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
			case ir.ValueDynamic, ir.ValueFunction:
				fmt.Fprintf(buf, "  call void @jayess_print_value(ptr %s)\n", arg.ref)
			default:
				fmt.Fprintf(buf, "  call void @jayess_print_string(ptr %s)\n", arg.ref)
			}
			return emittedValue{}, nil
		}
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  call void @jayess_print_many(ptr %s)\n", argsBoxed)
		return emittedValue{}, nil
	case "__jayess_console_log", "__jayess_console_warn", "__jayess_console_error":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  call void @%s(ptr %s)\n", call.Callee, argsBoxed)
		return emittedValue{}, nil
	case "__jayess_process_cwd":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_cwd()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_env":
		name, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedName, err := g.emitBoxedValue(buf, state, name)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_env(ptr %s)\n", tmp, boxedName)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_argv":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_argv()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_platform":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_platform()\n", tmp)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_process_arch":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_arch()\n", tmp)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_process_exit":
		codeValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedCode, err := g.emitBoxedValue(buf, state, codeValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_exit(ptr %s)\n", tmp, boxedCode)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_join":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_join(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_normalize":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_normalize(ptr %s)\n", tmp, boxedPath)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_resolve":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_resolve(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_relative":
		fromValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		toValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedFrom, err := g.emitBoxedValue(buf, state, fromValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedTo, err := g.emitBoxedValue(buf, state, toValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_relative(ptr %s, ptr %s)\n", tmp, boxedFrom, boxedTo)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_parse":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_parse(ptr %s)\n", tmp, boxedPath)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_is_absolute":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_is_absolute(ptr %s)\n", tmp, boxedPath)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_format":
		partsValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedParts, err := g.emitBoxedValue(buf, state, partsValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_format(ptr %s)\n", tmp, boxedParts)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_sep":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_sep()\n", tmp)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_delimiter":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_delimiter()\n", tmp)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_path_basename", "__jayess_path_dirname", "__jayess_path_extname":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__")
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s)\n", tmp, runtimeName, boxedPath)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_fs_read_file":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		encodingRef := "null"
		if len(call.Arguments) > 1 {
			encodingValue, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			encodingRef, err = g.emitBoxedValue(buf, state, encodingValue)
			if err != nil {
				return emittedValue{}, err
			}
		} else {
			encodingRef, err = g.emitBoxedValue(buf, state, emittedValue{kind: ir.ValueUndefined})
			if err != nil {
				return emittedValue{}, err
			}
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_read_file(ptr %s, ptr %s)\n", tmp, boxedPath, encodingRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_write_file":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		contentValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedContent, err := g.emitBoxedValue(buf, state, contentValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_write_file(ptr %s, ptr %s)\n", tmp, boxedPath, boxedContent)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_exists":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_exists(ptr %s)\n", tmp, boxedPath)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_remove":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		optionsRef := "null"
		if len(call.Arguments) > 1 {
			optionsValue, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			optionsRef, err = g.emitBoxedValue(buf, state, optionsValue)
			if err != nil {
				return emittedValue{}, err
			}
		} else {
			optionsRef, err = g.emitBoxedValue(buf, state, emittedValue{kind: ir.ValueUndefined})
			if err != nil {
				return emittedValue{}, err
			}
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_remove(ptr %s, ptr %s)\n", tmp, boxedPath, optionsRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_read_dir":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		optionsRef := "null"
		if len(call.Arguments) > 1 {
			optionsValue, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			boxedOptions, err := g.emitBoxedValue(buf, state, optionsValue)
			if err != nil {
				return emittedValue{}, err
			}
			optionsRef = boxedOptions
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_read_dir(ptr %s, ptr %s)\n", tmp, boxedPath, optionsRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_stat":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_stat(ptr %s)\n", tmp, boxedPath)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_mkdir":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		optionsRef := "null"
		if len(call.Arguments) > 1 {
			optionsValue, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			boxedOptions, err := g.emitBoxedValue(buf, state, optionsValue)
			if err != nil {
				return emittedValue{}, err
			}
			optionsRef = boxedOptions
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_mkdir(ptr %s, ptr %s)\n", tmp, boxedPath, optionsRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_copy_file", "__jayess_fs_copy_dir", "__jayess_fs_rename":
		fromValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		toValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedFrom, err := g.emitBoxedValue(buf, state, fromValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedTo, err := g.emitBoxedValue(buf, state, toValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__")
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedFrom, boxedTo)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
	case "__jayess_apply":
		return g.emitApply(buf, state, call)
	case "__jayess_bind":
		return g.emitBind(buf, state, call)
	case "__jayess_array_push":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		value, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedValue, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		if target.kind == ir.ValueArray {
			length := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_array_push_value(ptr %s, ptr %s)\n", length, target.ref, boxedValue)
			number := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", number, length)
			return emittedValue{kind: ir.ValueNumber, ref: number}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_array_push(ptr %s, ptr %s)\n", tmp, boxedTarget, boxedValue)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_pop":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		if target.kind == ir.ValueArray {
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_array_pop_value(ptr %s)\n", tmp, target.ref)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_array_pop(ptr %s)\n", tmp, boxedTarget)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_shift":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		if target.kind == ir.ValueArray {
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_array_shift_value(ptr %s)\n", tmp, target.ref)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_array_shift(ptr %s)\n", tmp, boxedTarget)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_unshift":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		value, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedValue, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		if target.kind == ir.ValueArray {
			length := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_array_unshift_value(ptr %s, ptr %s)\n", length, target.ref, boxedValue)
			number := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", number, length)
			return emittedValue{kind: ir.ValueNumber, ref: number}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_array_unshift(ptr %s, ptr %s)\n", tmp, boxedTarget, boxedValue)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_slice":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		start, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		end, err := g.emitExpression(buf, state, call.Arguments[2])
		if err != nil {
			return emittedValue{}, err
		}
		startRef, err := g.emitNumberOperand(buf, state, start)
		if err != nil {
			return emittedValue{}, err
		}
		startInt := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", startInt, startRef)
		endInt := "0"
		hasEnd := "false"
		if end.kind != ir.ValueUndefined {
			endRef, err := g.emitNumberOperand(buf, state, end)
			if err != nil {
				return emittedValue{}, err
			}
			endInt = state.nextTemp()
			fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", endInt, endRef)
			hasEnd = "true"
		}
		if target.kind == ir.ValueArray {
			slice := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_array_slice_values(ptr %s, i32 %s, i32 %s, i1 %s)\n", slice, target.ref, startInt, endInt, hasEnd)
			return emittedValue{kind: ir.ValueArray, ref: slice}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_array_slice(ptr %s, i32 %s, i32 %s, i1 %s)\n", tmp, boxedTarget, startInt, endInt, hasEnd)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_keys":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		if target.kind == ir.ValueObject {
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_object_keys(ptr %s)\n", tmp, target.ref)
			return emittedValue{kind: ir.ValueArray, ref: tmp}, nil
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_keys(ptr %s)\n", tmp, boxedTarget)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_values":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_values(ptr %s)\n", tmp, boxedTarget)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_entries":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_entries(ptr %s)\n", tmp, boxedTarget)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_assign":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		source, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		boxedSource, err := g.emitBoxedValue(buf, state, source)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_assign(ptr %s, ptr %s)\n", tmp, boxedTarget, boxedSource)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_has_own":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		key, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		boxedKey, err := g.emitBoxedValue(buf, state, key)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_has_own(ptr %s, ptr %s)\n", tmp, boxedTarget, boxedKey)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_from_entries":
		entries, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedEntries, err := g.emitBoxedValue(buf, state, entries)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_object_from_entries(ptr %s)\n", tmp, boxedEntries)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_object_rest":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		excluded, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		boxedExcluded, err := g.emitBoxedValue(buf, state, excluded)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_rest(ptr %s, ptr %s)\n", tmp, boxedTarget, boxedExcluded)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_map_new":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_map_new()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_set_new":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_set_new()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_date_new":
		var argRef string
		if len(call.Arguments) == 0 {
			argRef = state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", argRef)
		} else {
			value, err := g.emitExpression(buf, state, call.Arguments[0])
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			argRef = boxed
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_date_new(ptr %s)\n", tmp, argRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_date_now":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_date_now()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_regexp_new":
		patternRef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", patternRef)
		flagsRef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", flagsRef)
		if len(call.Arguments) > 0 {
			value, err := g.emitExpression(buf, state, call.Arguments[0])
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			patternRef = boxed
		}
		if len(call.Arguments) > 1 {
			value, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			boxed, err := g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
			flagsRef = boxed
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_regexp_new(ptr %s, ptr %s)\n", tmp, patternRef, flagsRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_json_stringify":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_json_stringify(ptr %s)\n", tmp, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_json_parse":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_json_parse(ptr %s)\n", tmp, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_iter_values":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_iterable_values(ptr %s)\n", tmp, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_math_floor", "__jayess_math_ceil", "__jayess_math_round", "__jayess_math_abs", "__jayess_math_sqrt":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		number, err := g.emitNumberOperand(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @%s(double %s)\n", tmp, call.Callee, number)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_math_min", "__jayess_math_max", "__jayess_math_pow":
		left, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		right, err := g.emitExpression(buf, state, call.Arguments[1])
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
		fmt.Fprintf(buf, "  %s = call double @%s(double %s, double %s)\n", tmp, call.Callee, leftRef, rightRef)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_math_random":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_math_random()\n", tmp)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_number_is_nan", "__jayess_number_is_finite":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		if call.Callee == "__jayess_number_is_nan" {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_std_number_is_nan(ptr %s)\n", tmp, boxed)
		} else {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_std_number_is_finite(ptr %s)\n", tmp, boxed)
		}
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_string_from_char_code":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_string_from_char_code(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueString, ref: tmp}, nil
	case "__jayess_array_is_array":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_array_is_array(ptr %s)\n", tmp, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_from":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_array_from(ptr %s)\n", tmp, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_of":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_array_of(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_array_for_each":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		callback, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		itemsRef, err := g.emitArrayLikeValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		callbackBoxed, err := g.emitBoxedValue(buf, state, callback)
		if err != nil {
			return emittedValue{}, err
		}
		if err := g.emitForEachCall(buf, state, itemsRef, callbackBoxed); err != nil {
			return emittedValue{}, err
		}
		return emittedValue{kind: ir.ValueUndefined, ref: ""}, nil
	case "__jayess_array_map":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		callback, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		itemsRef, err := g.emitArrayLikeValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		callbackBoxed, err := g.emitBoxedValue(buf, state, callback)
		if err != nil {
			return emittedValue{}, err
		}
		return g.emitArrayMapCall(buf, state, itemsRef, callbackBoxed)
	case "__jayess_array_filter":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		callback, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		itemsRef, err := g.emitArrayLikeValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		callbackBoxed, err := g.emitBoxedValue(buf, state, callback)
		if err != nil {
			return emittedValue{}, err
		}
		return g.emitArrayFilterCall(buf, state, itemsRef, callbackBoxed)
	case "__jayess_array_find":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		callback, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		itemsRef, err := g.emitArrayLikeValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		callbackBoxed, err := g.emitBoxedValue(buf, state, callback)
		if err != nil {
			return emittedValue{}, err
		}
		return g.emitArrayFindCall(buf, state, itemsRef, callbackBoxed)
	case "__jayess_current_this":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_current_this()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		if hasSpreadIRArguments(call.Arguments) {
			calleeValue, err := g.emitNamedFunctionValue(buf, state, call.Callee)
			if err != nil {
				return emittedValue{}, err
			}
			argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
			if err != nil {
				return emittedValue{}, err
			}
			undefThis := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefThis)
			return g.emitApplyFromValues(buf, state, calleeValue.ref, undefThis, emittedValue{kind: ir.ValueDynamic, ref: argsBoxed})
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
		undefThis := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefThis)
		return g.emitDirectJayessCallWithThis(buf, state, call.Callee, fn, undefThis, args)
	}
}

func (g *LLVMIRGenerator) emitInvoke(buf *bytes.Buffer, state *functionState, call *ir.InvokeExpression) (emittedValue, error) {
	callee, err := g.emitExpression(buf, state, call.Callee)
	if err != nil {
		return emittedValue{}, err
	}
	if call.Optional {
		nullish, err := g.emitNullishCheck(buf, state, callee)
		if err != nil {
			return emittedValue{}, err
		}
		nilLabel := state.nextLabel("optcall.nil")
		callLabel := state.nextLabel("optcall.call")
		endLabel := state.nextLabel("optcall.end")
		resultPtr := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", nullish, nilLabel, callLabel)
		fmt.Fprintf(buf, "%s:\n", nilLabel)
		undef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", callLabel)
		boxed, err := g.emitBoxedValue(buf, state, callee)
		if err != nil {
			return emittedValue{}, err
		}
		directArgsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		mergedArgs := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_merge_bound_args(ptr %s, ptr %s)\n", mergedArgs, boxed, directArgsBoxed)
		boundThis := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_this(ptr %s)\n", boundThis, boxed)
		result, err := g.emitApplyFromValues(buf, state, boxed, boundThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", result.ref, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", endLabel)
		out := state.nextTemp()
		fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", out, resultPtr)
		return emittedValue{kind: ir.ValueDynamic, ref: out}, nil
	}
	boxed, err := g.emitBoxedValue(buf, state, callee)
	if err != nil {
		return emittedValue{}, err
	}

	directArgsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
	if err != nil {
		return emittedValue{}, err
	}
	mergedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_merge_bound_args(ptr %s, ptr %s)\n", mergedArgs, boxed, directArgsBoxed)
	boundThis := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_this(ptr %s)\n", boundThis, boxed)
	return g.emitApplyFromValues(buf, state, boxed, boundThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
}

func (g *LLVMIRGenerator) emitApply(buf *bytes.Buffer, state *functionState, call *ir.CallExpression) (emittedValue, error) {
	callee, err := g.emitExpression(buf, state, call.Arguments[0])
	if err != nil {
		return emittedValue{}, err
	}
	argsValue, err := g.emitExpression(buf, state, call.Arguments[2])
	if err != nil {
		return emittedValue{}, err
	}

	thisValue, err := g.emitExpression(buf, state, call.Arguments[1])
	if err != nil {
		return emittedValue{}, err
	}
	boxedCallee, err := g.emitBoxedValue(buf, state, callee)
	if err != nil {
		return emittedValue{}, err
	}
	boxedArgs, err := g.emitArrayLikeValue(buf, state, argsValue)
	if err != nil {
		return emittedValue{}, err
	}
	boxedThis, err := g.emitBoxedValue(buf, state, thisValue)
	if err != nil {
		return emittedValue{}, err
	}
	mergedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_merge_bound_args(ptr %s, ptr %s)\n", mergedArgs, boxedCallee, boxedArgs)
	return g.emitApplyFromValues(buf, state, boxedCallee, boxedThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
}

func (g *LLVMIRGenerator) emitBind(buf *bytes.Buffer, state *functionState, call *ir.CallExpression) (emittedValue, error) {
	callee, err := g.emitExpression(buf, state, call.Arguments[0])
	if err != nil {
		return emittedValue{}, err
	}
	boundThis, err := g.emitExpression(buf, state, call.Arguments[1])
	if err != nil {
		return emittedValue{}, err
	}
	boundArgs, err := g.emitExpression(buf, state, call.Arguments[2])
	if err != nil {
		return emittedValue{}, err
	}
	boxedCallee, err := g.emitBoxedValue(buf, state, callee)
	if err != nil {
		return emittedValue{}, err
	}
	boxedThis, err := g.emitBoxedValue(buf, state, boundThis)
	if err != nil {
		return emittedValue{}, err
	}
	boxedArgs, err := g.emitArrayLikeValue(buf, state, boundArgs)
	if err != nil {
		return emittedValue{}, err
	}
	tmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bind(ptr %s, ptr %s, ptr %s)\n", tmp, boxedCallee, boxedThis, boxedArgs)
	return emittedValue{kind: ir.ValueFunction, ref: tmp}, nil
}

func (g *LLVMIRGenerator) emitForEachCall(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string) error {
	indexPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca i32\n", indexPtr)
	fmt.Fprintf(buf, "  store i32 0, ptr %s\n", indexPtr)
	condLabel := state.nextLabel("foreach.cond")
	bodyLabel := state.nextLabel("foreach.body")
	endLabel := state.nextLabel("foreach.end")
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	lenRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", lenRef, itemsRef)
	index := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load i32, ptr %s\n", index, indexPtr)
	cmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp slt i32 %s, %s\n", cmp, index, lenRef)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cmp, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	item := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %s)\n", item, itemsRef, index)
	argsArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
	fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, item)
	boxedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
	undefinedThis := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
	if _, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: boxedArgs}); err != nil {
		return err
	}
	nextIndex := state.nextTemp()
	fmt.Fprintf(buf, "  %s = add i32 %s, 1\n", nextIndex, index)
	fmt.Fprintf(buf, "  store i32 %s, ptr %s\n", nextIndex, indexPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitArrayMapCall(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string) (emittedValue, error) {
	resultArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", resultArray)
	if err := g.emitArrayCallbackLoop(buf, state, itemsRef, callbackRef, func(item string) error {
		argsArray := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
		fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, item)
		boxedArgs := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
		undefinedThis := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
		result, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: boxedArgs})
		if err != nil {
			return err
		}
		boxedResult, err := g.emitBoxedValue(buf, state, result)
		if err != nil {
			return err
		}
		fmt.Fprintf(buf, "  call i32 @jayess_array_push_value(ptr %s, ptr %s)\n", resultArray, boxedResult)
		return nil
	}); err != nil {
		return emittedValue{}, err
	}
	return emittedValue{kind: ir.ValueArray, ref: resultArray}, nil
}

func (g *LLVMIRGenerator) emitArrayFilterCall(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string) (emittedValue, error) {
	resultArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", resultArray)
	if err := g.emitArrayCallbackLoop(buf, state, itemsRef, callbackRef, func(item string) error {
		argsArray := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
		fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, item)
		boxedArgs := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
		undefinedThis := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
		result, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: boxedArgs})
		if err != nil {
			return err
		}
		cond, err := g.emitTruthyFromValue(buf, state, result)
		if err != nil {
			return err
		}
		thenLabel := state.nextLabel("array.filter.then")
		endLabel := state.nextLabel("array.filter.end")
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, thenLabel, endLabel)
		fmt.Fprintf(buf, "%s:\n", thenLabel)
		fmt.Fprintf(buf, "  call i32 @jayess_array_push_value(ptr %s, ptr %s)\n", resultArray, item)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", endLabel)
		return nil
	}); err != nil {
		return emittedValue{}, err
	}
	return emittedValue{kind: ir.ValueArray, ref: resultArray}, nil
}

func (g *LLVMIRGenerator) emitArrayFindCall(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string) (emittedValue, error) {
	lenRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", lenRef, itemsRef)
	indexPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca i32\n", indexPtr)
	fmt.Fprintf(buf, "  store i32 0, ptr %s\n", indexPtr)
	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
	undef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
	condLabel := state.nextLabel("array.find.cond")
	bodyLabel := state.nextLabel("array.find.body")
	matchLabel := state.nextLabel("array.find.match")
	nextLabel := state.nextLabel("array.find.next")
	endLabel := state.nextLabel("array.find.end")
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	index := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load i32, ptr %s\n", index, indexPtr)
	hasNext := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp slt i32 %s, %s\n", hasNext, index, lenRef)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasNext, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	item := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %s)\n", item, itemsRef, index)
	argsArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
	fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, item)
	boxedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
	undefinedThis := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
	result, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: boxedArgs})
	if err != nil {
		return emittedValue{}, err
	}
	cond, err := g.emitTruthyFromValue(buf, state, result)
	if err != nil {
		return emittedValue{}, err
	}
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, matchLabel, nextLabel)
	fmt.Fprintf(buf, "%s:\n", matchLabel)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", item, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	fmt.Fprintf(buf, "%s:\n", nextLabel)
	nextIndex := state.nextTemp()
	fmt.Fprintf(buf, "  %s = add i32 %s, 1\n", nextIndex, index)
	fmt.Fprintf(buf, "  store i32 %s, ptr %s\n", nextIndex, indexPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	out := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", out, resultPtr)
	return emittedValue{kind: ir.ValueDynamic, ref: out}, nil
}

func (g *LLVMIRGenerator) emitArrayCallbackLoop(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string, body func(item string) error) error {
	lenRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", lenRef, itemsRef)
	indexPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca i32\n", indexPtr)
	fmt.Fprintf(buf, "  store i32 0, ptr %s\n", indexPtr)
	condLabel := state.nextLabel("array.loop.cond")
	bodyLabel := state.nextLabel("array.loop.body")
	endLabel := state.nextLabel("array.loop.end")
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	index := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load i32, ptr %s\n", index, indexPtr)
	hasNext := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp slt i32 %s, %s\n", hasNext, index, lenRef)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasNext, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	item := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %s)\n", item, itemsRef, index)
	if err := body(item); err != nil {
		return err
	}
	nextIndex := state.nextTemp()
	fmt.Fprintf(buf, "  %s = add i32 %s, 1\n", nextIndex, index)
	fmt.Fprintf(buf, "  store i32 %s, ptr %s\n", nextIndex, indexPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitApplyFromValues(buf *bytes.Buffer, state *functionState, boxedCallee string, thisRef string, argsValue emittedValue) (emittedValue, error) {
	fnPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_ptr(ptr %s)\n", fnPtr, boxedCallee)
	envPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_env(ptr %s)\n", envPtr, boxedCallee)
	paramCountRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_function_param_count(ptr %s)\n", paramCountRef, boxedCallee)
	hasRestRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_value_function_has_rest(ptr %s)\n", hasRestRef, boxedCallee)

	lenRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", lenRef, argsValue.ref)

	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
	endLabel := state.nextLabel("apply.end")
	defaultLabel := state.nextLabel("apply.default")

	caseLabels := make([]string, 9)
	checkLabels := make([]string, 9)
	for i := 0; i <= 8; i++ {
		caseLabels[i] = state.nextLabel(fmt.Sprintf("apply.%d", i))
		checkLabels[i] = state.nextLabel(fmt.Sprintf("apply.check.%d", i))
	}

	fmt.Fprintf(buf, "  br label %%%s\n", checkLabels[0])
	for i := 0; i <= 8; i++ {
		next := defaultLabel
		if i < 8 {
			next = checkLabels[i+1]
		}
		fmt.Fprintf(buf, "%s:\n", checkLabels[i])
		match := state.nextTemp()
		fmt.Fprintf(buf, "  %s = icmp eq i32 %s, %d\n", match, lenRef, i)
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", match, caseLabels[i], next)
	}

	for i := 0; i <= 8; i++ {
		fmt.Fprintf(buf, "%s:\n", caseLabels[i])
		var args []string
		for index := 0; index < i; index++ {
			argRef, err := g.emitApplyArgumentAt(buf, state, argsValue, index)
			if err != nil {
				return emittedValue{}, err
			}
			args = append(args, fmt.Sprintf("ptr %s", argRef))
		}
		if err := g.emitIndirectApplyCase(buf, state, fnPtr, envPtr, thisRef, paramCountRef, hasRestRef, args, resultPtr, endLabel); err != nil {
			return emittedValue{}, err
		}
	}

	fmt.Fprintf(buf, "%s:\n", defaultLabel)
	undef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	result := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", result, resultPtr)
	return emittedValue{kind: ir.ValueDynamic, ref: result}, nil
}

func (g *LLVMIRGenerator) emitIndirectApplyCase(buf *bytes.Buffer, state *functionState, fnPtr, envPtr, thisRef, paramCountRef, hasRestRef string, args []string, resultPtr, endLabel string) error {
	fixedLabel := state.nextLabel("apply.fixed")
	restLabel := state.nextLabel("apply.rest")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasRestRef, restLabel, fixedLabel)

	fmt.Fprintf(buf, "%s:\n", fixedLabel)
	if err := g.emitApplySignatureDispatch(buf, state, fnPtr, envPtr, thisRef, paramCountRef, false, args, resultPtr, endLabel); err != nil {
		return err
	}

	fmt.Fprintf(buf, "%s:\n", restLabel)
	if err := g.emitApplySignatureDispatch(buf, state, fnPtr, envPtr, thisRef, paramCountRef, true, args, resultPtr, endLabel); err != nil {
		return err
	}
	return nil
}

func (g *LLVMIRGenerator) emitApplySignatureDispatch(buf *bytes.Buffer, state *functionState, fnPtr, envPtr, thisRef, paramCountRef string, hasRest bool, args []string, resultPtr, endLabel string) error {
	defaultLabel := state.nextLabel("apply.sig.default")
	maxParams := 8
	for paramCount := 0; paramCount <= maxParams; paramCount++ {
		checkLabel := state.nextLabel(fmt.Sprintf("apply.sig.check.%d", paramCount))
		matchLabel := state.nextLabel(fmt.Sprintf("apply.sig.match.%d", paramCount))
		nextLabel := defaultLabel
		if paramCount < maxParams {
			nextLabel = state.nextLabel(fmt.Sprintf("apply.sig.next.%d", paramCount))
		}
		fmt.Fprintf(buf, "  br label %%%s\n", checkLabel)
		fmt.Fprintf(buf, "%s:\n", checkLabel)
		match := state.nextTemp()
		fmt.Fprintf(buf, "  %s = icmp eq i32 %s, %d\n", match, paramCountRef, paramCount)
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", match, matchLabel, nextLabel)
		fmt.Fprintf(buf, "%s:\n", matchLabel)
		callArgs, err := g.emitPreparedArguments(buf, state, args, paramCount, hasRest)
		if err != nil {
			return err
		}
		if err := g.emitIndirectCallIntoResult(buf, state, fnPtr, envPtr, thisRef, callArgs, resultPtr, endLabel); err != nil {
			return err
		}
		if paramCount < maxParams {
			fmt.Fprintf(buf, "%s:\n", nextLabel)
		}
	}
	fmt.Fprintf(buf, "%s:\n", defaultLabel)
	undef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitPreparedArguments(buf *bytes.Buffer, state *functionState, args []string, paramCount int, hasRest bool) ([]string, error) {
	if !hasRest {
		out := make([]string, 0, paramCount)
		for i := 0; i < paramCount; i++ {
			if i < len(args) {
				out = append(out, args[i])
				continue
			}
			undef := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
			out = append(out, fmt.Sprintf("ptr %s", undef))
		}
		return out, nil
	}
	if paramCount == 0 {
		return nil, nil
	}
	fixedCount := paramCount - 1
	out := make([]string, 0, paramCount)
	for i := 0; i < fixedCount; i++ {
		if i < len(args) {
			out = append(out, args[i])
			continue
		}
		undef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
		out = append(out, fmt.Sprintf("ptr %s", undef))
	}
	restArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", restArray)
	if fixedCount < len(args) {
		for _, arg := range args[fixedCount:] {
			fmt.Fprintf(buf, "  call i32 @jayess_array_push_value(ptr %s, %s)\n", restArray, arg)
		}
	}
	out = append(out, fmt.Sprintf("ptr %s", restArray))
	return out, nil
}

func (g *LLVMIRGenerator) emitDirectJayessCallWithThis(buf *bytes.Buffer, state *functionState, callee string, fn ir.Function, thisRef string, args []string) (emittedValue, error) {
	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
	fmt.Fprintf(buf, "  call void @jayess_push_this(ptr %s)\n", thisRef)
	callLabel := state.nextLabel("direct.call")
	doneLabel := state.nextLabel("direct.done")
	fmt.Fprintf(buf, "  br label %%%s\n", callLabel)
	fmt.Fprintf(buf, "%s:\n", callLabel)
	result := state.nextTemp()
	callArgs, err := g.emitDirectCallArguments(buf, state, fn, args)
	if err != nil {
		return emittedValue{}, err
	}
	if len(callArgs) == 0 {
		fmt.Fprintf(buf, "  %s = call ptr @%s()\n", result, callee)
	} else {
		fmt.Fprintf(buf, "  %s = call ptr @%s(%s)\n", result, callee, strings.Join(callArgs, ", "))
	}
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", result, resultPtr)
	fmt.Fprintf(buf, "  call void @jayess_pop_this()\n")
	fmt.Fprintf(buf, "  br label %%%s\n", doneLabel)
	fmt.Fprintf(buf, "%s:\n", doneLabel)
	final := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", final, resultPtr)
	return emittedValue{kind: ir.ValueDynamic, ref: final}, nil
}

func (g *LLVMIRGenerator) emitDirectCallArguments(buf *bytes.Buffer, state *functionState, fn ir.Function, args []string) ([]string, error) {
	if len(fn.Params) == 0 {
		return nil, nil
	}
	if !fn.Params[len(fn.Params)-1].Rest {
		out := append([]string{}, args...)
		for len(out) < len(fn.Params) {
			undef := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
			out = append(out, fmt.Sprintf("ptr %s", undef))
		}
		return out, nil
	}
	fixedCount := len(fn.Params) - 1
	if len(args) < fixedCount {
		return nil, fmt.Errorf("rest-parameter call is missing required arguments")
	}
	out := append([]string{}, args[:fixedCount]...)
	restArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", restArray)
	for _, arg := range args[fixedCount:] {
		fmt.Fprintf(buf, "  call i32 @jayess_array_push_value(ptr %s, %s)\n", restArray, arg)
	}
	out = append(out, fmt.Sprintf("ptr %s", restArray))
	return out, nil
}

func (g *LLVMIRGenerator) emitNamedFunctionValue(buf *bytes.Buffer, state *functionState, name string) (emittedValue, error) {
	classRef := "null"
	if state.classNames[name] {
		classRef = state.stringRefs[name]
	}
	paramCount, hasRest := functionMetadata(name, state.functions, state.externs)
	tmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr null, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, name, state.stringRefs[name], classRef, paramCount, hasRest)
	return emittedValue{kind: ir.ValueFunction, ref: tmp}, nil
}

func (g *LLVMIRGenerator) emitArrayRefFromExpressions(buf *bytes.Buffer, state *functionState, expressions []ir.Expression) (string, error) {
	arrayRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", arrayRef)
	for _, expr := range expressions {
		if spread, ok := expr.(*ir.SpreadExpression); ok {
			spreadValue, err := g.emitExpression(buf, state, spread.Value)
			if err != nil {
				return "", err
			}
			if err := g.emitAppendArrayLikeValue(buf, state, arrayRef, spreadValue); err != nil {
				return "", err
			}
			continue
		}
		value, err := g.emitExpression(buf, state, expr)
		if err != nil {
			return "", err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(buf, "  call i32 @jayess_array_push_value(ptr %s, ptr %s)\n", arrayRef, boxed)
	}
	return arrayRef, nil
}

func (g *LLVMIRGenerator) emitAppendArrayLikeValue(buf *bytes.Buffer, state *functionState, arrayRef string, value emittedValue) error {
	switch value.kind {
	case ir.ValueArray:
		fmt.Fprintf(buf, "  call void @jayess_array_append_array(ptr %s, ptr %s)\n", arrayRef, value.ref)
		return nil
	case ir.ValueArgsArray:
		boxedArgs := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_args(ptr %s)\n", boxedArgs, value.ref)
		rawArgs := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_as_array(ptr %s)\n", rawArgs, boxedArgs)
		fmt.Fprintf(buf, "  call void @jayess_array_append_array(ptr %s, ptr %s)\n", arrayRef, rawArgs)
		return nil
	case ir.ValueDynamic:
		rawArray := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_as_array(ptr %s)\n", rawArray, value.ref)
		fmt.Fprintf(buf, "  call void @jayess_array_append_array(ptr %s, ptr %s)\n", arrayRef, rawArray)
		return nil
	default:
		return fmt.Errorf("spread expects an array-like value")
	}
}

func (g *LLVMIRGenerator) emitBoxedArrayFromExpressions(buf *bytes.Buffer, state *functionState, expressions []ir.Expression) (string, error) {
	arrayRef, err := g.emitArrayRefFromExpressions(buf, state, expressions)
	if err != nil {
		return "", err
	}
	boxedArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArray, arrayRef)
	return boxedArray, nil
}

func (g *LLVMIRGenerator) emitNullishCheck(buf *bytes.Buffer, state *functionState, value emittedValue) (string, error) {
	switch value.kind {
	case ir.ValueDynamic:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_value_is_nullish(ptr %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueNull, ir.ValueUndefined:
		return "true", nil
	default:
		return "false", nil
	}
}

func (g *LLVMIRGenerator) emitIndexAccess(buf *bytes.Buffer, state *functionState, target, index emittedValue) (emittedValue, error) {
	if index.kind == ir.ValueString {
		tmp := state.nextTemp()
		if target.kind == ir.ValueObject {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_object_get(ptr %s, ptr %s)\n", tmp, target.ref, index.ref)
		} else {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, target.ref, index.ref)
		}
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	}
	if index.kind == ir.ValueDynamic {
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_dynamic_index(ptr %s, ptr %s)\n", tmp, target.ref, index.ref)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
}

func (g *LLVMIRGenerator) emitMemberAccess(buf *bytes.Buffer, state *functionState, target emittedValue, property string) (emittedValue, error) {
	if property == "length" {
		switch target.kind {
		case ir.ValueArray:
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_array_length(ptr %s)\n", tmp, target.ref)
			num := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", num, tmp)
			return emittedValue{kind: ir.ValueNumber, ref: num}, nil
		case ir.ValueArgsArray:
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_args_length(ptr %s)\n", tmp, target.ref)
			num := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", num, tmp)
			return emittedValue{kind: ir.ValueNumber, ref: num}, nil
		case ir.ValueDynamic:
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", tmp, target.ref)
			num := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", num, tmp)
			return emittedValue{kind: ir.ValueNumber, ref: num}, nil
		case ir.ValueString:
			boxed, err := g.emitBoxedValue(buf, state, target)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", tmp, boxed)
			num := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", num, tmp)
			return emittedValue{kind: ir.ValueNumber, ref: num}, nil
		}
	}
	tmp := state.nextTemp()
	if target.kind == ir.ValueObject {
		fmt.Fprintf(buf, "  %s = call ptr @jayess_object_get(ptr %s, ptr %s)\n", tmp, target.ref, state.stringRefs[property])
	} else if target.kind == ir.ValueDynamic {
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, target.ref, state.stringRefs[property])
	} else {
		boxed, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, boxed, state.stringRefs[property])
	}
	return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
}

func (g *LLVMIRGenerator) emitArrayLikeValue(buf *bytes.Buffer, state *functionState, value emittedValue) (string, error) {
	switch value.kind {
	case ir.ValueArray:
		boxed := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxed, value.ref)
		return boxed, nil
	case ir.ValueArgsArray:
		boxed := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_args(ptr %s)\n", boxed, value.ref)
		return boxed, nil
	case ir.ValueDynamic:
		return value.ref, nil
	default:
		return "", fmt.Errorf("expected array-like value, got %s", value.kind)
	}
}

func (g *LLVMIRGenerator) emitApplyArgumentAt(buf *bytes.Buffer, state *functionState, source emittedValue, index int) (string, error) {
	arg := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %d)\n", arg, source.ref, index)
	return arg, nil
}

func (g *LLVMIRGenerator) emitIndirectCallIntoResult(buf *bytes.Buffer, state *functionState, fnPtr, envPtr, thisRef string, args []string, resultPtr, endLabel string) error {
	hasFn := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp ne ptr %s, null\n", hasFn, fnPtr)
	callLabel := state.nextLabel("apply.call")
	failLabel := state.nextLabel("apply.fail")
	envLabel := state.nextLabel("apply.env")
	noEnvLabel := state.nextLabel("apply.noenv")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasFn, callLabel, failLabel)
	fmt.Fprintf(buf, "%s:\n", callLabel)
	fmt.Fprintf(buf, "  call void @jayess_push_this(ptr %s)\n", thisRef)
	hasEnv := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp ne ptr %s, null\n", hasEnv, envPtr)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasEnv, envLabel, noEnvLabel)
	fmt.Fprintf(buf, "%s:\n", envLabel)
	envArgs := append([]string{fmt.Sprintf("ptr %s", envPtr)}, args...)
	envResult := state.nextTemp()
	if len(envArgs) == 0 {
		fmt.Fprintf(buf, "  %s = call ptr %s()\n", envResult, fnPtr)
	} else {
		fmt.Fprintf(buf, "  %s = call ptr %s(%s)\n", envResult, fnPtr, strings.Join(envArgs, ", "))
	}
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", envResult, resultPtr)
	fmt.Fprintf(buf, "  call void @jayess_pop_this()\n")
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	fmt.Fprintf(buf, "%s:\n", noEnvLabel)
	noEnvResult := state.nextTemp()
	if len(args) == 0 {
		fmt.Fprintf(buf, "  %s = call ptr %s()\n", noEnvResult, fnPtr)
	} else {
		fmt.Fprintf(buf, "  %s = call ptr %s(%s)\n", noEnvResult, fnPtr, strings.Join(args, ", "))
	}
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", noEnvResult, resultPtr)
	fmt.Fprintf(buf, "  call void @jayess_pop_this()\n")
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	fmt.Fprintf(buf, "%s:\n", failLabel)
	failResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", failResult)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", failResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitCondition(buf *bytes.Buffer, state *functionState, expr ir.Expression) (string, error) {
	value, err := g.emitExpression(buf, state, expr)
	if err != nil {
		return "", err
	}
	if state.exceptionTarget != "" {
		g.emitExceptionCheck(buf, state, state.exceptionTarget)
	}
	return g.emitTruthyFromValue(buf, state, value)
}

func (g *LLVMIRGenerator) emitExceptionCheck(buf *bytes.Buffer, state *functionState, target string) {
	continueLabel := state.nextLabel("throw.cont")
	hasException := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_has_exception()\n", hasException)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasException, target, continueLabel)
	fmt.Fprintf(buf, "%s:\n", continueLabel)
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
	case ir.ValueFunction:
		return "true", nil
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
	case ir.ValueFunction:
		return value.ref, nil
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
	case ir.OperatorStrictEq:
		return "oeq"
	case ir.OperatorNe:
		return "one"
	case ir.OperatorStrictNe:
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

func buildClassNames(classes []ir.ClassDecl) map[string]bool {
	names := map[string]bool{}
	for _, classDecl := range classes {
		names[classDecl.Name] = true
	}
	return names
}

func functionMetadata(name string, functions map[string]ir.Function, externs map[string]ir.ExternFunction) (int, string) {
	if fn, ok := functions[name]; ok {
		hasRest := "false"
		if len(fn.Params) > 0 && fn.Params[len(fn.Params)-1].Rest {
			hasRest = "true"
		}
		return len(fn.Params), hasRest
	}
	if fn, ok := externs[name]; ok {
		hasRest := "false"
		if len(fn.Params) > 0 && fn.Params[len(fn.Params)-1].Rest {
			hasRest = "true"
		}
		return len(fn.Params), hasRest
	}
	return 0, "false"
}

func hasSpreadIRArguments(arguments []ir.Expression) bool {
	for _, arg := range arguments {
		if _, ok := arg.(*ir.SpreadExpression); ok {
			return true
		}
	}
	return false
}

func collectStrings(module *ir.Module) []string {
	seen := map[string]bool{}
	var out []string
	addString("__jayess_class", seen, &out)
	addString("undefined", seen, &out)
	addString("object", seen, &out)
	addString("boolean", seen, &out)
	addString("number", seen, &out)
	addString("string", seen, &out)
	addString("function", seen, &out)
	for _, classDecl := range module.Classes {
		addString(classDecl.Name, seen, &out)
	}
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
	case *ir.DeleteStatement:
		collectStringsFromExpression(stmt.Target, seen, out)
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
	case *ir.FunctionValue:
		addString(expr.Name, seen, out)
		if expr.Environment != nil {
			collectStringsFromExpression(expr.Environment, seen, out)
		}
	case *ir.ObjectLiteral:
		for _, property := range expr.Properties {
			if property.Computed {
				collectStringsFromExpression(property.KeyExpr, seen, out)
			} else {
				addString(property.Key, seen, out)
			}
			collectStringsFromExpression(property.Value, seen, out)
		}
	case *ir.ArrayLiteral:
		for _, element := range expr.Elements {
			collectStringsFromExpression(element, seen, out)
		}
	case *ir.TemplateLiteral:
		for _, part := range expr.Parts {
			addString(part, seen, out)
		}
		for _, value := range expr.Values {
			collectStringsFromExpression(value, seen, out)
		}
	case *ir.MemberExpression:
		addString(expr.Property, seen, out)
		collectStringsFromExpression(expr.Target, seen, out)
	case *ir.BinaryExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.TypeofExpression:
		addString("undefined", seen, out)
		addString("object", seen, out)
		addString("boolean", seen, out)
		addString("number", seen, out)
		addString("string", seen, out)
		addString("function", seen, out)
		collectStringsFromExpression(expr.Value, seen, out)
	case *ir.InstanceofExpression:
		addString("__jayess_class", seen, out)
		if expr.ClassName != "" {
			addString(expr.ClassName, seen, out)
		}
		collectStringsFromExpression(expr.Left, seen, out)
		if expr.Right != nil {
			collectStringsFromExpression(expr.Right, seen, out)
		}
	case *ir.ComparisonExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.LogicalExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.NullishCoalesceExpression:
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
	case *ir.InvokeExpression:
		collectStringsFromExpression(expr.Callee, seen, out)
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
