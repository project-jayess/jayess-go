package lowering

import (
	"fmt"
	"sync/atomic"

	"jayess-go/ast"
	"jayess-go/ir"
)

var lowerTempCounter uint64

func Lower(program *ast.Program) (*ir.Module, error) {
	module := &ir.Module{}
	functionKinds := map[string]bool{}
	globalSymbols := map[string]ir.ValueKind{}
	for _, global := range program.Globals {
		value, err := lowerExpression(global.Value, globalSymbols, functionKinds)
		if err != nil {
			return nil, err
		}
		module.Globals = append(module.Globals, ir.VariableDecl{
			Visibility: lowerVisibility(global.Visibility),
			Kind:       lowerDeclarationKind(global.Kind),
			Name:       global.Name,
			Value:      value,
		})
		globalSymbols[global.Name] = ir.ValueDynamic
	}
	for _, fn := range program.ExternFunctions {
		functionKinds[fn.Name] = true
		symbolName := fn.NativeSymbol
		if symbolName == "" {
			symbolName = fn.Name
		}
		lowered := ir.ExternFunction{Name: fn.Name, SymbolName: symbolName, Variadic: fn.Variadic}
		for _, param := range fn.Params {
			kind := ir.ValueDynamic
			if param.Rest {
				kind = ir.ValueArray
			}
			lowered.Params = append(lowered.Params, ir.Parameter{Name: param.Name, Kind: kind, Rest: param.Rest})
		}
		module.ExternFunctions = append(module.ExternFunctions, lowered)
	}
	for _, fn := range program.Functions {
		functionKinds[fn.Name] = true
	}
	for _, fn := range program.Functions {
		lowered, err := lowerFunction(fn, globalSymbols, functionKinds)
		if err != nil {
			return nil, err
		}
		module.Functions = append(module.Functions, lowered)
	}
	return module, nil
}

func lowerFunction(fn *ast.FunctionDecl, globals map[string]ir.ValueKind, functions map[string]bool) (ir.Function, error) {
	result := ir.Function{Visibility: lowerVisibility(fn.Visibility), Name: fn.Name}

	symbols := cloneKinds(globals)
	for _, param := range fn.Params {
		kind := ir.ValueDynamic
		if fn.Name == "main" {
			kind = ir.ValueArgsArray
		} else if param.Rest {
			kind = ir.ValueArray
		}
		var defaultExpr ir.Expression
		if param.Default != nil {
			var err error
			defaultExpr, err = lowerExpression(param.Default, symbols, functions)
			if err != nil {
				return ir.Function{}, err
			}
		}
		result.Params = append(result.Params, ir.Parameter{Name: param.Name, Kind: kind, Rest: param.Rest, Default: defaultExpr})
		symbols[param.Name] = kind
	}

	for _, param := range result.Params {
		if param.Default == nil {
			continue
		}
		result.Body = append(result.Body, &ir.IfStatement{
			Condition: &ir.ComparisonExpression{
				Operator: ir.OperatorStrictEq,
				Left:     &ir.VariableRef{Name: param.Name, Kind: param.Kind},
				Right:    &ir.UndefinedLiteral{},
			},
			Consequence: []ir.Statement{
				&ir.AssignmentStatement{
					Target: &ir.VariableRef{Name: param.Name, Kind: param.Kind},
					Value:  param.Default,
				},
			},
		})
	}

	body, err := lowerStatements(fn.Body, symbols, functions)
	if err != nil {
		return ir.Function{}, err
	}
	result.Body = append(result.Body, body...)
	return result, nil
}

func lowerStatements(statements []ast.Statement, symbols map[string]ir.ValueKind, functions map[string]bool) ([]ir.Statement, error) {
	var out []ir.Statement
	local := cloneKinds(symbols)
	for _, stmt := range statements {
		lowered, err := lowerStatement(stmt, local, functions)
		if err != nil {
			return nil, err
		}
		if decl, ok := lowered.(*ir.VariableDecl); ok {
			if decl.Kind == ir.DeclarationVar {
				local[decl.Name] = ir.ValueDynamic
			} else {
				local[decl.Name] = inferIRKind(decl.Value)
			}
		}
		out = append(out, lowered)
	}
	return out, nil
}

func lowerStatement(stmt ast.Statement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		value, err := lowerExpression(stmt.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.VariableDecl{
			Visibility: lowerVisibility(stmt.Visibility),
			Kind:       lowerDeclarationKind(stmt.Kind),
			Name:       stmt.Name,
			Value:      value,
		}, nil
	case *ast.AssignmentStatement:
		target, err := lowerExpression(stmt.Target, symbols, functions)
		if err != nil {
			return nil, err
		}
		value, err := lowerExpression(stmt.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.AssignmentStatement{Target: target, Value: value}, nil
	case *ast.ReturnStatement:
		value, err := lowerExpression(stmt.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.ReturnStatement{Value: value}, nil
	case *ast.ExpressionStatement:
		value, err := lowerExpression(stmt.Expression, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.ExpressionStatement{Expression: value}, nil
	case *ast.DeleteStatement:
		target, err := lowerExpression(stmt.Target, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.DeleteStatement{Target: target}, nil
	case *ast.ThrowStatement:
		value, err := lowerExpression(stmt.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.ThrowStatement{Value: value}, nil
	case *ast.TryStatement:
		tryBody, err := lowerStatements(stmt.TryBody, symbols, functions)
		if err != nil {
			return nil, err
		}
		catchSymbols := cloneKinds(symbols)
		if stmt.CatchName != "" {
			catchSymbols[stmt.CatchName] = ir.ValueDynamic
		}
		catchBody, err := lowerStatements(stmt.CatchBody, catchSymbols, functions)
		if err != nil {
			return nil, err
		}
		finallyBody, err := lowerStatements(stmt.FinallyBody, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.TryStatement{TryBody: tryBody, CatchName: stmt.CatchName, CatchBody: catchBody, FinallyBody: finallyBody}, nil
	case *ast.IfStatement:
		condition, err := lowerExpression(stmt.Condition, symbols, functions)
		if err != nil {
			return nil, err
		}
		consequence, err := lowerStatements(stmt.Consequence, symbols, functions)
		if err != nil {
			return nil, err
		}
		alternative, err := lowerStatements(stmt.Alternative, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
	case *ast.WhileStatement:
		condition, err := lowerExpression(stmt.Condition, symbols, functions)
		if err != nil {
			return nil, err
		}
		body, err := lowerStatements(stmt.Body, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.WhileStatement{Condition: condition, Body: body}, nil
	case *ast.ForStatement:
		var init ir.Statement
		var condition ir.Expression
		var update ir.Statement
		var err error

		loopSymbols := cloneKinds(symbols)
		if stmt.Init != nil {
			init, err = lowerStatement(stmt.Init, loopSymbols, functions)
			if err != nil {
				return nil, err
			}
			if decl, ok := init.(*ir.VariableDecl); ok {
				if decl.Kind == ir.DeclarationVar {
					loopSymbols[decl.Name] = ir.ValueDynamic
				} else {
					loopSymbols[decl.Name] = inferIRKind(decl.Value)
				}
			}
		}
		if stmt.Condition != nil {
			condition, err = lowerExpression(stmt.Condition, loopSymbols, functions)
			if err != nil {
				return nil, err
			}
		}
		body, err := lowerStatements(stmt.Body, loopSymbols, functions)
		if err != nil {
			return nil, err
		}
		if stmt.Update != nil {
			update, err = lowerStatement(stmt.Update, loopSymbols, functions)
			if err != nil {
				return nil, err
			}
		}
		return &ir.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
	case *ast.ForOfStatement:
		return lowerForOfStatement(stmt, symbols, functions)
	case *ast.ForInStatement:
		return lowerForInStatement(stmt, symbols, functions)
	case *ast.SwitchStatement:
		return lowerSwitchStatement(stmt, symbols, functions)
	case *ast.BreakStatement:
		return &ir.BreakStatement{}, nil
	case *ast.ContinueStatement:
		return &ir.ContinueStatement{}, nil
	default:
		return nil, fmt.Errorf("unsupported statement in lowering")
	}
}

func lowerExpression(expr ast.Expression, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Expression, error) {
	switch expr := expr.(type) {
	case *ast.NumberLiteral:
		return &ir.NumberLiteral{Value: expr.Value}, nil
	case *ast.BooleanLiteral:
		return &ir.BooleanLiteral{Value: expr.Value}, nil
	case *ast.NullLiteral:
		return &ir.NullLiteral{}, nil
	case *ast.UndefinedLiteral:
		return &ir.UndefinedLiteral{}, nil
	case *ast.ThisExpression:
		return &ir.CallExpression{Callee: "__jayess_current_this", Kind: ir.ValueDynamic}, nil
	case *ast.NewTargetExpression:
		return &ir.NewTargetExpression{Kind: ir.ValueDynamic}, nil
	case *ast.AwaitExpression:
		value, err := lowerExpression(expr.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.CallExpression{Callee: "__jayess_await", Arguments: []ir.Expression{value}, Kind: ir.ValueDynamic}, nil
	case *ast.StringLiteral:
		return &ir.StringLiteral{Value: expr.Value}, nil
	case *ast.ObjectLiteral:
		literal := &ir.ObjectLiteral{}
		for _, property := range expr.Properties {
			value, err := lowerExpression(property.Value, symbols, functions)
			if err != nil {
				return nil, err
			}
			lowered := ir.ObjectProperty{Key: property.Key, Value: value, Computed: property.Computed}
			if property.Computed {
				keyExpr, err := lowerExpression(property.KeyExpr, symbols, functions)
				if err != nil {
					return nil, err
				}
				lowered.KeyExpr = keyExpr
			}
			literal.Properties = append(literal.Properties, lowered)
		}
		return literal, nil
	case *ast.ArrayLiteral:
		literal := &ir.ArrayLiteral{}
		for _, element := range expr.Elements {
			value, err := lowerExpression(element, symbols, functions)
			if err != nil {
				return nil, err
			}
			literal.Elements = append(literal.Elements, value)
		}
		return literal, nil
	case *ast.TemplateLiteral:
		out := &ir.TemplateLiteral{Parts: append([]string{}, expr.Parts...)}
		for _, value := range expr.Values {
			lowered, err := lowerExpression(value, symbols, functions)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, lowered)
		}
		return out, nil
	case *ast.SpreadExpression:
		value, err := lowerExpression(expr.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.SpreadExpression{Value: value}, nil
	case *ast.Identifier:
		if kind, ok := symbols[expr.Name]; ok {
			return &ir.VariableRef{Name: expr.Name, Kind: kind}, nil
		}
		if functions[expr.Name] {
			return &ir.FunctionValue{Name: expr.Name}, nil
		}
		return nil, fmt.Errorf("unknown identifier %s", expr.Name)
	case *ast.ClosureExpression:
		var environment ir.Expression
		if expr.Environment != nil {
			var err error
			environment, err = lowerExpression(expr.Environment, symbols, functions)
			if err != nil {
				return nil, err
			}
		}
		return &ir.FunctionValue{Name: expr.FunctionName, Environment: environment}, nil
	case *ast.BinaryExpression:
		left, err := lowerExpression(expr.Left, symbols, functions)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.BinaryExpression{Operator: lowerOperator(expr.Operator), Left: left, Right: right}, nil
	case *ast.NullishCoalesceExpression:
		left, err := lowerExpression(expr.Left, symbols, functions)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.NullishCoalesceExpression{Left: left, Right: right, Kind: ir.ValueDynamic}, nil
	case *ast.UnaryExpression:
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.UnaryExpression{Operator: ir.OperatorNot, Right: right}, nil
	case *ast.TypeofExpression:
		value, err := lowerExpression(expr.Value, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.TypeofExpression{Value: value}, nil
	case *ast.InstanceofExpression:
		left, err := lowerExpression(expr.Left, symbols, functions)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		className := ""
		if ident, ok := expr.Right.(*ast.Identifier); ok {
			className = ident.Name
		}
		return &ir.InstanceofExpression{Left: left, Right: right, ClassName: className}, nil
	case *ast.LogicalExpression:
		left, err := lowerExpression(expr.Left, symbols, functions)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		op := ir.OperatorAnd
		if expr.Operator == ast.OperatorOr {
			op = ir.OperatorOr
		}
		return &ir.LogicalExpression{Operator: op, Left: left, Right: right}, nil
	case *ast.ComparisonExpression:
		left, err := lowerExpression(expr.Left, symbols, functions)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpression(expr.Right, symbols, functions)
		if err != nil {
			return nil, err
		}
		return &ir.ComparisonExpression{Operator: lowerComparisonOperator(expr.Operator), Left: left, Right: right}, nil
	case *ast.IndexExpression:
		target, err := lowerExpression(expr.Target, symbols, functions)
		if err != nil {
			return nil, err
		}
		index, err := lowerExpression(expr.Index, symbols, functions)
		if err != nil {
			return nil, err
		}
		kind := ir.ValueDynamic
		if variable, ok := target.(*ir.VariableRef); ok && variable.Kind == ir.ValueArgsArray {
			kind = ir.ValueString
		}
		return &ir.IndexExpression{Target: target, Index: index, Kind: kind, Optional: expr.Optional}, nil
	case *ast.MemberExpression:
		target, err := lowerExpression(expr.Target, symbols, functions)
		if err != nil {
			return nil, err
		}
		kind := ir.ValueDynamic
		if expr.Property == "length" {
			switch inferIRKind(target) {
			case ir.ValueArray, ir.ValueArgsArray:
				kind = ir.ValueNumber
			}
		}
		return &ir.MemberExpression{Target: target, Property: expr.Property, Kind: kind, Optional: expr.Optional}, nil
	case *ast.CallExpression:
		if kind, ok := symbols[expr.Callee]; ok && (kind == ir.ValueFunction || kind == ir.ValueDynamic) {
			callee := &ir.VariableRef{Name: expr.Callee, Kind: kind}
			invoke := &ir.InvokeExpression{Callee: callee, Kind: ir.ValueDynamic}
			for _, arg := range expr.Arguments {
				lowered, err := lowerExpression(arg, symbols, functions)
				if err != nil {
					return nil, err
				}
				invoke.Arguments = append(invoke.Arguments, lowered)
			}
			return invoke, nil
		}
		call := &ir.CallExpression{Callee: expr.Callee}
		for _, arg := range expr.Arguments {
			lowered, err := lowerExpression(arg, symbols, functions)
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, lowered)
		}
		switch expr.Callee {
		case "readLine", "readKey":
			call.Kind = ir.ValueString
		case "compile", "compileFile":
			call.Kind = ir.ValueDynamic
		case "__jayess_array_push":
			call.Kind = ir.ValueNumber
		case "__jayess_array_pop":
			call.Kind = ir.ValueDynamic
		case "__jayess_array_shift":
			call.Kind = ir.ValueDynamic
		case "__jayess_array_unshift":
			call.Kind = ir.ValueNumber
		case "__jayess_array_slice":
			call.Kind = ir.ValueDynamic
		case "__jayess_array_map", "__jayess_array_filter", "__jayess_array_find":
			call.Kind = ir.ValueDynamic
		case "__jayess_array_is_array", "__jayess_array_from", "__jayess_array_of":
			call.Kind = ir.ValueDynamic
		case "__jayess_object_keys":
			call.Kind = ir.ValueDynamic
		case "__jayess_object_values", "__jayess_object_entries", "__jayess_object_assign", "__jayess_object_has_own", "__jayess_object_from_entries":
			call.Kind = ir.ValueDynamic
		case "__jayess_object_rest":
			call.Kind = ir.ValueDynamic
		case "__jayess_std_map_new", "__jayess_std_set_new", "__jayess_std_date_new", "__jayess_std_regexp_new", "__jayess_std_date_now", "__jayess_std_json_stringify", "__jayess_std_json_parse", "__jayess_iter_values", "__jayess_std_error_new", "__jayess_std_aggregate_error_new", "__jayess_std_array_buffer_new", "__jayess_std_uint8_array_new", "__jayess_std_data_view_new", "__jayess_std_uint8_array_from_string", "__jayess_std_uint8_array_concat", "__jayess_std_iterator_from", "__jayess_std_promise_resolve", "__jayess_std_promise_reject", "__jayess_std_promise_all", "__jayess_std_promise_race", "__jayess_std_promise_all_settled", "__jayess_std_promise_any", "__jayess_timers_sleep", "__jayess_timers_set_timeout", "__jayess_timers_clear_timeout", "__jayess_await":
			call.Kind = ir.ValueDynamic
		case "__jayess_std_uint8_array_equals":
			call.Kind = ir.ValueDynamic
		case "__jayess_std_uint8_array_compare":
			call.Kind = ir.ValueNumber
		case "__jayess_math_floor", "__jayess_math_ceil", "__jayess_math_round", "__jayess_math_min", "__jayess_math_max", "__jayess_math_abs", "__jayess_math_pow", "__jayess_math_sqrt", "__jayess_math_random":
			call.Kind = ir.ValueNumber
		case "__jayess_number_is_nan", "__jayess_number_is_finite":
			call.Kind = ir.ValueDynamic
		case "__jayess_string_from_char_code":
			call.Kind = ir.ValueString
		case "__jayess_array_for_each":
			call.Kind = ir.ValueUndefined
		case "__jayess_console_log", "__jayess_console_warn", "__jayess_console_error":
			call.Kind = ir.ValueUndefined
		case "__jayess_process_cwd", "__jayess_process_env", "__jayess_process_argv", "__jayess_process_exit", "__jayess_fs_read_file", "__jayess_fs_read_file_async", "__jayess_fs_write_file", "__jayess_fs_write_file_async", "__jayess_fs_create_read_stream", "__jayess_fs_create_write_stream", "__jayess_fs_exists", "__jayess_fs_read_dir", "__jayess_fs_stat", "__jayess_fs_mkdir", "__jayess_fs_remove", "__jayess_fs_copy_file", "__jayess_fs_rename", "__jayess_path_parse", "__jayess_path_is_absolute", "__jayess_url_parse", "__jayess_querystring_parse", "__jayess_dns_lookup", "__jayess_dns_lookup_all", "__jayess_dns_reverse", "__jayess_net_connect", "__jayess_net_listen", "__jayess_http_parse_request", "__jayess_http_parse_response", "__jayess_http_request", "__jayess_http_request_stream", "__jayess_http_request_stream_async", "__jayess_http_get", "__jayess_http_get_stream", "__jayess_http_get_stream_async", "__jayess_http_request_async", "__jayess_http_get_async", "__jayess_https_request", "__jayess_https_request_stream", "__jayess_https_request_stream_async", "__jayess_https_get", "__jayess_https_get_stream", "__jayess_https_get_stream_async", "__jayess_https_request_async", "__jayess_https_get_async":
			call.Kind = ir.ValueDynamic
		case "__jayess_process_thread_pool_size":
			call.Kind = ir.ValueNumber
		case "__jayess_net_is_ip":
			call.Kind = ir.ValueNumber
		case "__jayess_tls_is_available", "__jayess_https_is_available", "__jayess_tls_connect":
			call.Kind = ir.ValueDynamic
		case "__jayess_process_platform", "__jayess_path_join", "__jayess_path_normalize", "__jayess_path_resolve", "__jayess_path_relative", "__jayess_path_format", "__jayess_path_basename", "__jayess_path_dirname", "__jayess_path_extname", "__jayess_url_format", "__jayess_querystring_stringify", "__jayess_http_format_request", "__jayess_http_format_response":
			call.Kind = ir.ValueString
		case "__jayess_tls_backend", "__jayess_https_backend":
			call.Kind = ir.ValueString
		case "print", "sleep", "sleepAsync", "setTimeout", "clearTimeout":
			call.Kind = ""
		default:
			call.Kind = ir.ValueDynamic
		}
		return call, nil
	case *ast.InvokeExpression:
		callee, err := lowerExpression(expr.Callee, symbols, functions)
		if err != nil {
			return nil, err
		}
		invoke := &ir.InvokeExpression{Callee: callee, Kind: ir.ValueDynamic, Optional: expr.Optional}
		for _, arg := range expr.Arguments {
			lowered, err := lowerExpression(arg, symbols, functions)
			if err != nil {
				return nil, err
			}
			invoke.Arguments = append(invoke.Arguments, lowered)
		}
		return invoke, nil
	default:
		return nil, fmt.Errorf("unsupported expression in lowering")
	}
}

func inferIRKind(expr ir.Expression) ir.ValueKind {
	switch expr := expr.(type) {
	case *ir.NumberLiteral:
		return ir.ValueNumber
	case *ir.BinaryExpression:
		if expr.Operator == ir.OperatorAdd && (inferIRKind(expr.Left) == ir.ValueString || inferIRKind(expr.Right) == ir.ValueString) {
			return ir.ValueString
		}
		return ir.ValueNumber
	case *ir.BooleanLiteral, *ir.ComparisonExpression:
		return ir.ValueBoolean
	case *ir.NullLiteral, *ir.UndefinedLiteral:
		return ir.ValueDynamic
	case *ir.UnaryExpression, *ir.LogicalExpression:
		return ir.ValueBoolean
	case *ir.NullishCoalesceExpression:
		return expr.Kind
	case *ir.TypeofExpression:
		return ir.ValueString
	case *ir.NewTargetExpression:
		return expr.Kind
	case *ir.InstanceofExpression:
		return ir.ValueBoolean
	case *ir.StringLiteral:
		return ir.ValueString
	case *ir.IndexExpression:
		return expr.Kind
	case *ir.ArrayLiteral:
		return ir.ValueArray
	case *ir.TemplateLiteral:
		return ir.ValueString
	case *ir.SpreadExpression:
		return inferIRKind(expr.Value)
	case *ir.ObjectLiteral:
		return ir.ValueObject
	case *ir.FunctionValue:
		return ir.ValueFunction
	case *ir.MemberExpression:
		return ir.ValueDynamic
	case *ir.VariableRef:
		return expr.Kind
	case *ir.CallExpression:
		return expr.Kind
	case *ir.InvokeExpression:
		return expr.Kind
	default:
		return ""
	}
}

func lowerVisibility(visibility ast.Visibility) ir.Visibility {
	if visibility == ast.VisibilityPrivate {
		return ir.VisibilityPrivate
	}
	return ir.VisibilityPublic
}

func lowerDeclarationKind(kind ast.DeclarationKind) ir.DeclarationKind {
	switch kind {
	case ast.DeclarationConst:
		return ir.DeclarationConst
	case ast.DeclarationLet:
		return ir.DeclarationLet
	default:
		return ir.DeclarationVar
	}
}

func lowerOperator(op ast.BinaryOperator) ir.BinaryOperator {
	switch op {
	case ast.OperatorAdd:
		return ir.OperatorAdd
	case ast.OperatorSub:
		return ir.OperatorSub
	case ast.OperatorMul:
		return ir.OperatorMul
	default:
		return ir.OperatorDiv
	}
}

func lowerComparisonOperator(op ast.ComparisonOperator) ir.ComparisonOperator {
	switch op {
	case ast.OperatorEq:
		return ir.OperatorEq
	case ast.OperatorNe:
		return ir.OperatorNe
	case ast.OperatorStrictEq:
		return ir.OperatorStrictEq
	case ast.OperatorStrictNe:
		return ir.OperatorStrictNe
	case ast.OperatorLt:
		return ir.OperatorLt
	case ast.OperatorLte:
		return ir.OperatorLte
	case ast.OperatorGt:
		return ir.OperatorGt
	default:
		return ir.OperatorGte
	}
}

func cloneKinds(input map[string]ir.ValueKind) map[string]ir.ValueKind {
	out := make(map[string]ir.ValueKind, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func nextLowerTemp(prefix string) string {
	id := atomic.AddUint64(&lowerTempCounter, 1)
	return fmt.Sprintf("__jayess_%s_%d", prefix, id)
}

func lowerForOfStatement(stmt *ast.ForOfStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	itemsName := nextLowerTemp("items")
	indexName := nextLowerTemp("index")
	elementDecl := &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: stmt.Kind, Name: stmt.Name, Value: &ast.IndexExpression{
		Target: &ast.Identifier{Name: itemsName},
		Index:  &ast.Identifier{Name: indexName},
	}}
	update := &ast.AssignmentStatement{
		Target: &ast.Identifier{Name: indexName},
		Value: &ast.BinaryExpression{
			Operator: ast.OperatorAdd,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.NumberLiteral{Value: 1},
		},
	}
	loop := &ast.ForStatement{
		Init: &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: indexName, Value: &ast.NumberLiteral{Value: 0}},
		Condition: &ast.ComparisonExpression{
			Operator: ast.OperatorLt,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.MemberExpression{Target: &ast.Identifier{Name: itemsName}, Property: "length"},
		},
		Update: update,
		Body:   append([]ast.Statement{elementDecl}, stmt.Body...),
	}
	statements := []ast.Statement{
		&ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: itemsName, Value: &ast.CallExpression{Callee: "__jayess_iter_values", Arguments: []ast.Expression{stmt.Iterable}}},
		loop,
	}
	lowered, err := lowerStatements(statements, symbols, functions)
	if err != nil {
		return nil, err
	}
	return &ir.IfStatement{Condition: &ir.BooleanLiteral{Value: true}, Consequence: lowered}, nil
}

func lowerForInStatement(stmt *ast.ForInStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	keysName := nextLowerTemp("keys")
	indexName := nextLowerTemp("index")
	keyDecl := &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: stmt.Kind, Name: stmt.Name, Value: &ast.IndexExpression{
		Target: &ast.Identifier{Name: keysName},
		Index:  &ast.Identifier{Name: indexName},
	}}
	update := &ast.AssignmentStatement{
		Target: &ast.Identifier{Name: indexName},
		Value: &ast.BinaryExpression{
			Operator: ast.OperatorAdd,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.NumberLiteral{Value: 1},
		},
	}
	loop := &ast.ForStatement{
		Init: &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: indexName, Value: &ast.NumberLiteral{Value: 0}},
		Condition: &ast.ComparisonExpression{
			Operator: ast.OperatorLt,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.MemberExpression{Target: &ast.Identifier{Name: keysName}, Property: "length"},
		},
		Update: update,
		Body:   append([]ast.Statement{keyDecl}, stmt.Body...),
	}
	statements := []ast.Statement{
		&ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: keysName, Value: &ast.CallExpression{Callee: "__jayess_object_keys", Arguments: []ast.Expression{stmt.Iterable}}},
		loop,
	}
	lowered, err := lowerStatements(statements, symbols, functions)
	if err != nil {
		return nil, err
	}
	return &ir.IfStatement{Condition: &ir.BooleanLiteral{Value: true}, Consequence: lowered}, nil
}

func lowerSwitchStatement(stmt *ast.SwitchStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	switchName := nextLowerTemp("switch")
	var current ast.Statement = &ast.IfStatement{Condition: &ast.BooleanLiteral{Value: true}, Consequence: stmt.Default}
	for i := len(stmt.Cases) - 1; i >= 0; i-- {
		switchCase := stmt.Cases[i]
		current = &ast.IfStatement{
			Condition: &ast.ComparisonExpression{
				Operator: ast.OperatorEq,
				Left:     &ast.Identifier{Name: switchName},
				Right:    switchCase.Test,
			},
			Consequence: switchCase.Consequent,
			Alternative: []ast.Statement{current},
		}
	}
	statements := []ast.Statement{
		&ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationConst, Name: switchName, Value: stmt.Discriminant},
		current,
	}
	lowered, err := lowerStatements(statements, symbols, functions)
	if err != nil {
		return nil, err
	}
	return &ir.IfStatement{Condition: &ir.BooleanLiteral{Value: true}, Consequence: lowered}, nil
}
