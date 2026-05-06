package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

type stdlibBinding struct {
	Module string
	Member string
}

var stdlibRuntimeSymbols = map[string]map[string]string{
	"childProcess": {
		"spawn":      "jayess_child_process_spawn",
		"exec":       "jayess_child_process_exec",
		"pipe":       "jayess_child_process_pipe",
		"exitStatus": "jayess_child_process_exit_status",
		"signal":     "jayess_child_process_signal",
		"cleanup":    "jayess_child_process_cleanup",
	},
	"fs": {
		"readFile":          "jayess_fs_read_file",
		"writeFile":         "jayess_fs_write_file",
		"appendFile":        "jayess_fs_append_file",
		"deleteFile":        "jayess_fs_delete_file",
		"rename":            "jayess_fs_rename",
		"copyFile":          "jayess_fs_copy_file",
		"stat":              "jayess_fs_stat",
		"chmod":             "jayess_fs_chmod",
		"exists":            "jayess_fs_exists",
		"mkdir":             "jayess_fs_mkdir",
		"mkdirp":            "jayess_fs_mkdirp",
		"rmdir":             "jayess_fs_rmdir",
		"readdir":           "jayess_fs_readdir",
		"walkDir":           "jayess_fs_walk_dir",
		"symlink":           "jayess_fs_symlink",
		"watch":             "jayess_fs_watch",
		"createReadStream":  "jayess_fs_create_read_stream",
		"createWriteStream": "jayess_fs_create_write_stream",
	},
	"process": {
		"cwd":    "jayess_process_cwd",
		"exit":   "jayess_process_exit",
		"hrtime": "jayess_process_hrtime",
		"on":     "jayess_process_on",
	},
	"stream": {
		"readable":   "jayess_stream_readable",
		"writable":   "jayess_stream_writable",
		"duplex":     "jayess_stream_duplex",
		"transform":  "jayess_stream_transform",
		"pipe":       "jayess_stream_pipe",
		"awaitDrain": "jayess_stream_await_drain",
	},
	"terminal": {
		"isTTY":         "jayess_terminal_is_tty",
		"size":          "jayess_terminal_size",
		"supportsColor": "jayess_terminal_supports_color",
	},
}

var stdlibPropertyRuntimeSymbols = map[string]map[string]string{
	"process": {
		"argv":     "jayess_process_argv",
		"env":      "jayess_process_env",
		"stdin":    "jayess_process_stdin",
		"stdout":   "jayess_process_stdout",
		"stderr":   "jayess_process_stderr",
		"pid":      "jayess_process_pid",
		"platform": "jayess_process_platform",
	},
}

var processStreamRuntimeSymbols = map[string]map[string]string{
	"stdin": {
		"read": "jayess_process_stdin_read",
	},
	"stdout": {
		"write": "jayess_process_stdout_write",
	},
	"stderr": {
		"write": "jayess_process_stderr_write",
	},
}

var stdlibImportAliases = map[string]string{
	"child_process": "childProcess",
	"fs":            "fs",
	"process":       "process",
	"stream":        "stream",
	"terminal":      "terminal",
}

func (emitter *ExpressionEmitter) RegisterStdlibNamespace(local string, module string) {
	if local == "" || module == "" {
		return
	}
	if emitter.stdlibNamespaces == nil {
		emitter.stdlibNamespaces = map[string]string{}
	}
	emitter.stdlibNamespaces[local] = module
}

func (emitter *ExpressionEmitter) RegisterStdlibBinding(local string, module string, member string) {
	if local == "" || module == "" || member == "" {
		return
	}
	if emitter.stdlibBindings == nil {
		emitter.stdlibBindings = map[string]stdlibBinding{}
	}
	emitter.stdlibBindings[local] = stdlibBinding{Module: module, Member: member}
}

func (emitter *ExpressionEmitter) stdlibModuleForIdentifier(name string) (string, bool) {
	if module, ok := emitter.stdlibNamespaces[name]; ok {
		return module, true
	}
	if _, ok := stdlibRuntimeSymbols[name]; ok {
		return name, true
	}
	if _, ok := stdlibPropertyRuntimeSymbols[name]; ok {
		return name, true
	}
	return "", false
}

func (emitter *ExpressionEmitter) emitStdlibLocalCall(name string, arguments []ast.Expression) (string, bool, error) {
	binding, ok := emitter.stdlibBindings[name]
	if !ok {
		return "", false, nil
	}
	return emitter.emitStdlibRuntimeCall(binding.Module, binding.Member, arguments)
}

func (emitter *ExpressionEmitter) emitStdlibMemberInvoke(member *ast.MemberExpression, arguments []ast.Expression) (string, bool, error) {
	module, ok := emitter.stdlibMemberModule(member)
	if !ok {
		return "", false, nil
	}
	return emitter.emitStdlibRuntimeCall(module, member.Property, arguments)
}

func (emitter *ExpressionEmitter) emitProcessStreamInvoke(member *ast.MemberExpression, arguments []ast.Expression) (string, bool, error) {
	target, ok := member.Target.(*ast.MemberExpression)
	if !ok {
		return "", false, nil
	}
	root, ok := target.Target.(*ast.Identifier)
	if !ok || root.Name != "process" {
		return "", false, nil
	}
	methods, ok := processStreamRuntimeSymbols[target.Property]
	if !ok {
		return "", false, nil
	}
	symbol, ok := methods[member.Property]
	if !ok {
		return "", false, nil
	}
	args, err := emitter.runtimeValueArgs(arguments)
	if err != nil {
		return "", true, err
	}
	value, err := emitter.emitRuntimeValueCall(symbol, args)
	return value, true, err
}

func (emitter *ExpressionEmitter) emitStdlibProperty(member *ast.MemberExpression) (string, bool, error) {
	module, ok := emitter.stdlibMemberModule(member)
	if !ok {
		return "", false, nil
	}
	symbols, ok := stdlibPropertyRuntimeSymbols[module]
	if !ok {
		return "", false, nil
	}
	symbol, ok := symbols[member.Property]
	if !ok {
		return "", false, nil
	}
	value, err := emitter.emitRuntimeValueCall(symbol, nil)
	return value, true, err
}

func (emitter *ExpressionEmitter) stdlibMemberModule(member *ast.MemberExpression) (string, bool) {
	if member == nil || member.Optional || member.Private {
		return "", false
	}
	identifier, ok := member.Target.(*ast.Identifier)
	if !ok {
		return "", false
	}
	return emitter.stdlibModuleForIdentifier(identifier.Name)
}

func (emitter *ExpressionEmitter) emitStdlibRuntimeCall(module string, member string, arguments []ast.Expression) (string, bool, error) {
	symbols, ok := stdlibRuntimeSymbols[module]
	if !ok {
		return "", false, nil
	}
	symbol, ok := symbols[member]
	if !ok {
		return "", false, nil
	}
	args, err := emitter.runtimeValueArgs(arguments)
	if err != nil {
		return "", true, err
	}
	value, err := emitter.emitRuntimeValueCall(symbol, args)
	return value, true, err
}

func (emitter *ExpressionEmitter) runtimeValueArgs(expressions []ast.Expression) ([]RuntimeCallArg, error) {
	args := make([]RuntimeCallArg, 0, len(expressions))
	for _, expression := range expressions {
		value, err := emitter.EmitExpression(expression)
		if err != nil {
			return nil, err
		}
		args = append(args, RuntimeCallArg{IRType: runtimeValueIRType, Value: value})
	}
	return args, nil
}

func stdlibModuleAlias(importPath string) (string, bool) {
	module, ok := stdlibImportAliases[importPath]
	return module, ok
}

func stdlibBindingForImport(importPath string, specifier ast.ImportSpecifier) (stdlibBinding, bool) {
	module, ok := stdlibModuleAlias(importPath)
	if !ok || specifier.Default {
		return stdlibBinding{}, false
	}
	if specifier.Namespace {
		return stdlibBinding{Module: module}, true
	}
	if specifier.Imported == "" {
		return stdlibBinding{}, false
	}
	return stdlibBinding{Module: module, Member: specifier.Imported}, true
}

func stdlibUnsupportedCallError(module string, member string) error {
	return fmt.Errorf("unsupported stdlib runtime call %s.%s", module, member)
}
