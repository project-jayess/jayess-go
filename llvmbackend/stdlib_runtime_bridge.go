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
	"Buffer": {
		"create":            "jayess_buffer_create",
		"fromString":        "jayess_buffer_from_string",
		"toString":          "jayess_buffer_to_string",
		"slice":             "jayess_buffer_slice",
		"copy":              "jayess_buffer_copy",
		"readUInt16LE":      "jayess_buffer_read_uint16_le",
		"writeUInt16LE":     "jayess_buffer_write_uint16_le",
		"typedArrayView":    "jayess_buffer_typed_array_view",
		"createReadStream":  "jayess_buffer_create_read_stream",
		"createWriteStream": "jayess_buffer_create_write_stream",
	},
	"crypto": {
		"randomBytes":    "jayess_crypto_random_bytes",
		"hash":           "jayess_crypto_hash",
		"hmac":           "jayess_crypto_hmac",
		"encrypt":        "jayess_crypto_encrypt",
		"decrypt":        "jayess_crypto_decrypt",
		"publicEncrypt":  "jayess_crypto_public_encrypt",
		"privateDecrypt": "jayess_crypto_private_decrypt",
		"sign":           "jayess_crypto_sign",
		"verify":         "jayess_crypto_verify",
		"generateKey":    "jayess_crypto_generate_key",
		"secureCompare":  "jayess_crypto_secure_compare",
	},
	"compression": {
		"gzip":                   "jayess_compression_gzip",
		"gunzip":                 "jayess_compression_gunzip",
		"deflate":                "jayess_compression_deflate",
		"inflate":                "jayess_compression_inflate",
		"brotliCompress":         "jayess_compression_brotli_compress",
		"brotliDecompress":       "jayess_compression_brotli_decompress",
		"createCompressStream":   "jayess_compression_create_compress_stream",
		"createDecompressStream": "jayess_compression_create_decompress_stream",
	},
	"dns": {
		"lookup":   "jayess_dns_lookup",
		"reverse":  "jayess_dns_reverse",
		"resolver": "jayess_dns_resolver",
		"isIP":     "jayess_dns_is_ip",
		"parseIP":  "jayess_dns_parse_ip",
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
	"http": {
		"createServer":   "jayess_http_create_server",
		"request":        "jayess_http_request",
		"requestObject":  "jayess_http_request_object",
		"responseObject": "jayess_http_response_object",
		"headers":        "jayess_http_headers",
		"status":         "jayess_http_status",
		"readBody":       "jayess_http_read_body",
		"writeBody":      "jayess_http_write_body",
		"streamBody":     "jayess_http_stream_body",
		"keepAlive":      "jayess_http_keep_alive",
		"withTimeout":    "jayess_http_with_timeout",
	},
	"https": {
		"createServer":      "jayess_https_create_server",
		"request":           "jayess_https_request",
		"loadCertificate":   "jayess_https_load_certificate",
		"loadPrivateKey":    "jayess_https_load_private_key",
		"trustStore":        "jayess_https_trust_store",
		"verifyCertificate": "jayess_https_verify_certificate",
		"secureDefaults":    "jayess_https_secure_defaults",
	},
	"process": {
		"cwd":    "jayess_process_cwd",
		"exit":   "jayess_process_exit",
		"hrtime": "jayess_process_hrtime",
		"on":     "jayess_process_on",
	},
	"storage": {
		"open":   "jayess_storage_open",
		"close":  "jayess_storage_close",
		"get":    "jayess_storage_get",
		"put":    "jayess_storage_put",
		"delete": "jayess_storage_delete",
		"scan":   "jayess_storage_scan",
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
	"tcp": {
		"client":      "jayess_tcp_client",
		"server":      "jayess_tcp_server",
		"connect":     "jayess_tcp_connect",
		"listen":      "jayess_tcp_listen",
		"accept":      "jayess_tcp_accept",
		"read":        "jayess_tcp_read",
		"write":       "jayess_tcp_write",
		"close":       "jayess_tcp_close",
		"lastError":   "jayess_tcp_last_error",
		"withTimeout": "jayess_tcp_with_timeout",
		"awaitDrain":  "jayess_tcp_await_drain",
	},
	"tls": {
		"client":         "jayess_tls_client",
		"server":         "jayess_tls_server",
		"certificate":    "jayess_tls_certificate",
		"withALPN":       "jayess_tls_with_alpn",
		"verifyHostname": "jayess_tls_verify_hostname",
	},
	"udp": {
		"socket":        "jayess_udp_socket",
		"send":          "jayess_udp_send",
		"receive":       "jayess_udp_receive",
		"bind":          "jayess_udp_bind",
		"joinMulticast": "jayess_udp_join_multicast",
		"setBroadcast":  "jayess_udp_set_broadcast",
	},
	"url": {
		"parse":          "jayess_url_parse",
		"format":         "jayess_url_format",
		"parseQuery":     "jayess_url_parse_query",
		"stringifyQuery": "jayess_url_stringify_query",
		"encode":         "jayess_url_encode",
		"decode":         "jayess_url_decode",
		"fileURLToPath":  "jayess_url_file_url_to_path",
		"pathToFileURL":  "jayess_url_path_to_file_url",
	},
	"util": {
		"format":  "jayess_util_format",
		"inspect": "jayess_util_inspect",
	},
	"worker": {
		"thread":       "jayess_worker_thread",
		"postMessage":  "jayess_worker_post_message",
		"onMessage":    "jayess_worker_on_message",
		"sharedMemory": "jayess_worker_shared_memory",
		"atomicLoad":   "jayess_worker_atomic_load",
		"atomicStore":  "jayess_worker_atomic_store",
	},
}

var builtinRuntimeSymbols = map[string]string{
	"clearInterval":  "jayess_timer_clear_interval",
	"clearTimeout":   "jayess_timer_clear_timeout",
	"queueMicrotask": "jayess_queue_microtask",
	"setInterval":    "jayess_timer_set_interval",
	"setTimeout":     "jayess_timer_set_timeout",
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
	"Buffer":        "Buffer",
	"compression":   "compression",
	"crypto":        "crypto",
	"dns":           "dns",
	"fs":            "fs",
	"http":          "http",
	"https":         "https",
	"process":       "process",
	"storage":       "storage",
	"stream":        "stream",
	"tcp":           "tcp",
	"terminal":      "terminal",
	"tls":           "tls",
	"udp":           "udp",
	"url":           "url",
	"util":          "util",
	"worker":        "worker",
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
	if ok {
		return emitter.emitStdlibRuntimeCall(binding.Module, binding.Member, arguments)
	}
	symbol, ok := builtinRuntimeSymbols[name]
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
