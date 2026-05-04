package codegen

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"jayess-go/ir"
	"jayess-go/typesys"
)

type LLVMIRGenerator struct{}

func NewLLVMIRGenerator() *LLVMIRGenerator {
	return &LLVMIRGenerator{}
}

type variableSlot struct {
	kind        ir.ValueKind
	ptr         string
	ownsCleanup bool
}

type boxedUse struct {
	ref     string
	cleanup bool
	shallow bool
}

const indirectApplyMaxArgs = 16

type emittedValue struct {
	kind         ir.ValueKind
	ref          string
	staticString bool
}

type functionState struct {
	tempCounter           int
	labelCounter          int
	slots                 map[string]variableSlot
	hoistedVarSlots       map[string]variableSlot
	scopeStack            []cleanupScope
	stringRefs            map[string]string
	controlStack          []controlLabels
	functions             map[string]ir.Function
	externs               map[string]ir.ExternFunction
	globals               map[string]ir.ValueKind
	classNames            map[string]bool
	functionName          string
	eligibleLocals        map[string]ir.LocalLifetimeClassification
	isMain                bool
	exceptionTarget       string
	exceptionCleanupDepth int
}

type cleanupScope struct {
	cleanups []variableSlot
	shadowed []shadowedSlot
}

type shadowedSlot struct {
	name string
	slot variableSlot
	ok   bool
}

type debugMetadataState struct {
	enabled          bool
	fileName         string
	directory        string
	compileUnitID    int
	fileID           int
	subroutineTypeID int
	functionIDs      map[string]int
	locationIDs      map[string]int
}

type controlLabels struct {
	label                string
	breakTarget          string
	continueTarget       string
	cleanupScopeDepth    int
	allowsUnlabeledBreak bool
	allowsContinue       bool
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
	debugState := buildDebugMetadataState(module)
	sourceFilename := module.SourcePath
	if sourceFilename == "" {
		sourceFilename = "jayess"
	}
	sourceFilename = filepath.Base(sourceFilename)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "; jayess module\n")
	fmt.Fprintf(&buf, "target triple = %q\n\n", targetTriple)
	fmt.Fprintf(&buf, "source_filename = %q\n\n", sourceFilename)
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
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_tmpdir()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_hostname()\n")
	fmt.Fprintf(&buf, "declare double @jayess_std_process_uptime()\n")
	fmt.Fprintf(&buf, "declare double @jayess_std_process_hrtime()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_cpu_info()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_memory_info()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_user_info()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_thread_pool_size()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_on_signal(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_once_signal(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_off_signal(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_raise(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_process_exit(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compile(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compile_file(ptr, ptr)\n")
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
	fmt.Fprintf(&buf, "declare ptr @jayess_std_url_parse(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_url_format(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_querystring_parse(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_querystring_stringify(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_dns_lookup(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_dns_lookup_all(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_dns_reverse(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_dns_set_resolver(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_dns_clear_resolver()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_child_process_exec(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_child_process_spawn(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_child_process_kill(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_worker_create(ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_load(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_store(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_add(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_sub(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_and(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_or(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_xor(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_exchange(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare double @jayess_atomics_compareExchange(ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_random_bytes(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_hash(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_hmac(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_secure_compare(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_encrypt(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_decrypt(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_generate_key_pair(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_public_encrypt(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_private_decrypt(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_sign(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_crypto_verify(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_gzip(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_gunzip(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_deflate(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_inflate(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_brotli(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_unbrotli(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_gzip_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_gunzip_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_deflate_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_inflate_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_brotli_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_compression_create_unbrotli_stream()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_net_is_ip(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_net_create_datagram_socket(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_net_connect(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_net_listen(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_parse_request(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_format_request(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_parse_response(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_format_response(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_request(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_create_server(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_request_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_request_stream_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_get(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_get_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_get_stream_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_request_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_http_get_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_request(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_request_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_request_stream_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_get(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_get_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_get_stream_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_request_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_get_async(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_tls_is_available()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_tls_backend()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_tls_connect(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_tls_create_server(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_is_available()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_backend()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_https_create_server(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_read_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_read_file_async(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_write_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_append_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_write_file_async(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_create_read_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_create_write_stream(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_exists(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_read_dir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_stat(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_mkdir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_remove(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_copy_file(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_copy_dir(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_rename(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_symlink(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_fs_watch(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_line(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_key(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_line_value(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_read_key_value(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_sleep_ms(i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_sleep_async(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_set_timeout(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_clear_timeout(ptr)\n")
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
	fmt.Fprintf(&buf, "declare ptr @jayess_value_object_symbols(ptr)\n")
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
	fmt.Fprintf(&buf, "declare ptr @jayess_std_weak_map_new()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_weak_set_new()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_for(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_key_for(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_iterator()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_async_iterator()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_to_string_tag()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_has_instance()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_species()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_match()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_replace()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_search()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_split()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_symbol_to_primitive()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_date_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_date_now()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_regexp_new(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_error_new(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_aggregate_error_new(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_array_buffer_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_shared_array_buffer_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_int8_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint8_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint16_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_int16_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint32_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_int32_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_float32_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_float64_array_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_data_view_new(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint8_array_from_string(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint8_array_concat(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint8_array_equals(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_uint8_array_compare(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_iterator_from(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_async_iterator_from(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_resolve(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_reject(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_all(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_race(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_all_settled(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_std_promise_any(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_await(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_run_microtasks()\n")
	fmt.Fprintf(&buf, "declare void @jayess_runtime_shutdown()\n")
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
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_static_string(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_owned_string(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_from_bigint(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_not(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_and(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_or(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_xor(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_shl(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_shr(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_bitwise_ushr(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_stringify(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_template_string(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_concat_values(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_add(ptr, ptr)\n")
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
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function2(ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function3(ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function4(ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function5(ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function6(ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function7(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function8(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function9(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function10(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function11(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function12(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_call_function13(ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_call_with_this(ptr, ptr, ptr, i32)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_bound_this(ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_value_function_bound_arg_count(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_function_bound_arg(ptr, i32)\n")
	fmt.Fprintf(&buf, "declare double @jayess_value_to_number(ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_value_function_param_count(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_function_has_rest(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i32 @jayess_value_kind_of(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_value_is_nullish(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_string_eq(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_args_is_truthy(ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_merge_bound_args(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_value_constructor_return(ptr, ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_throw(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_throw_not_function()\n")
	fmt.Fprintf(&buf, "declare i1 @jayess_has_exception()\n")
	fmt.Fprintf(&buf, "declare ptr @jayess_take_exception()\n")
	fmt.Fprintf(&buf, "declare void @jayess_report_uncaught_exception()\n")
	fmt.Fprintf(&buf, "declare void @jayess_push_call_frame(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_pop_call_frame()\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_free_unshared(ptr)\n")
	fmt.Fprintf(&buf, "declare void @jayess_value_free_array_shallow(ptr)\n")
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
	eligibleLocalsByFunction := map[string]map[string]ir.LocalLifetimeClassification{}
	for _, item := range module.LifetimeEligible {
		locals := eligibleLocalsByFunction[item.Function]
		if locals == nil {
			locals = map[string]ir.LocalLifetimeClassification{}
			eligibleLocalsByFunction[item.Function] = locals
		}
		locals[localEligibilityKey(item.Name, item.Line, item.Column)] = item
	}

	if err := g.emitGlobalInit(&buf, module.Globals, stringRefs, functionsByName, externsByName, globalKinds, classNames); err != nil {
		return nil, err
	}

	for _, fn := range module.Functions {
		if err := g.emitFunction(&buf, fn, stringRefs, functionsByName, externsByName, globalKinds, classNames, eligibleLocalsByFunction[fn.Name], debugState); err != nil {
			return nil, err
		}
	}

	if err := g.emitEntryWrapper(&buf, findMain(module.Functions)); err != nil {
		return nil, err
	}
	emitDebugMetadata(&buf, module, debugState)

	return []byte(applyFunctionDebugLocations(buf.String(), module, debugState)), nil
}

func (g *LLVMIRGenerator) emitFunction(buf *bytes.Buffer, fn ir.Function, stringRefs map[string]string, functionsByName map[string]ir.Function, externsByName map[string]ir.ExternFunction, globalKinds map[string]ir.ValueKind, classNames map[string]bool, eligibleLocals map[string]ir.LocalLifetimeClassification, debugState *debugMetadataState) error {
	headerName := emittedFunctionName(fn.Name)
	returnType := "ptr"
	if fn.Name == "main" {
		returnType = "double"
	}
	emitFunctionSourceComment(buf, fn, headerName)
	fmt.Fprintf(buf, "define %s @%s(", returnType, headerName)
	for i, param := range fn.Params {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(buf, "ptr %%%s", param.Name)
	}
	buf.WriteString(")")
	if debugState != nil && debugState.enabled {
		if id, ok := debugState.functionIDs[fn.Name]; ok {
			fmt.Fprintf(buf, " !dbg !%d", id)
		}
	}
	buf.WriteString(" {\n")
	buf.WriteString("entry:\n")

	state := &functionState{
		slots:           map[string]variableSlot{},
		hoistedVarSlots: map[string]variableSlot{},
		stringRefs:      stringRefs,
		functions:       functionsByName,
		externs:         externsByName,
		globals:         globalKinds,
		classNames:      classNames,
		functionName:    fn.Name,
		eligibleLocals:  eligibleLocals,
		isMain:          fn.Name == "main",
	}
	uncaughtLabel := state.nextLabel("throw.uncaught")
	state.exceptionTarget = uncaughtLabel
	state.exceptionCleanupDepth = 0
	debugLocationSuffix := ""
	if debugState != nil && debugState.enabled {
		if id, ok := debugState.locationIDs[fn.Name]; ok {
			debugLocationSuffix = fmt.Sprintf(", !dbg !%d", id)
		}
	}
	fmt.Fprintf(buf, "  call void @jayess_push_call_frame(ptr %s)%s\n", state.stringRefs[stackFrameLabel(fn)], debugLocationSuffix)

	state.pushScope()
	for _, param := range fn.Params {
		slot := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", slot)
		fmt.Fprintf(buf, "  store ptr %%%s, ptr %s\n", param.Name, slot)
		state.slots[param.Name] = variableSlot{kind: param.Kind, ptr: slot}
		if param.CleanupEligible && param.Kind == ir.ValueDynamic {
			state.scopeStack[len(state.scopeStack)-1].cleanups = append(state.scopeStack[len(state.scopeStack)-1].cleanups, variableSlot{
				kind: ir.ValueDynamic,
				ptr:  slot,
			})
		}
	}
	g.emitHoistedVarSlots(buf, state, fn)

	terminated, err := g.emitScopedStatements(buf, state, fn.Body)
	if err != nil {
		return err
	}
	if !terminated {
		g.emitCurrentScopeCleanup(buf, state)
		state.popScope()
		buf.WriteString("  call void @jayess_pop_call_frame()\n")
		if state.isMain {
			buf.WriteString("  ret double 0.000000\n")
		} else {
			buf.WriteString("  %tmp.default = call ptr @jayess_value_undefined()\n")
			buf.WriteString("  ret ptr %tmp.default\n")
		}
	} else {
		state.popScope()
	}
	fmt.Fprintf(buf, "%s:\n", uncaughtLabel)
	buf.WriteString("  call void @jayess_pop_call_frame()\n")
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
	if fn.Line > 0 && fn.Column > 0 {
		fmt.Fprintf(buf, "; native entry wrapper for %s (%d:%d)\n", fn.Name, fn.Line, fn.Column)
	} else {
		fmt.Fprintf(buf, "; native entry wrapper for %s\n", fn.Name)
	}
	buf.WriteString("define i32 @main(i32 %argc, ptr %argv) {\n")
	buf.WriteString("entry:\n")
	buf.WriteString("  call void @jayess_init_globals()\n")
	if len(fn.Params) == 1 {
		buf.WriteString("  %args = call ptr @jayess_make_args(i32 %argc, ptr %argv)\n")
		buf.WriteString("  %result = call double @jayess_user_main(ptr %args)\n")
	} else {
		buf.WriteString("  %result = call double @jayess_user_main()\n")
	}
	buf.WriteString("  call void @jayess_run_microtasks()\n")
	buf.WriteString("  %thrown = call i1 @jayess_has_exception()\n")
	buf.WriteString("  br i1 %thrown, label %uncaught, label %exit.ok\n")
	buf.WriteString("uncaught:\n")
	buf.WriteString("  call void @jayess_report_uncaught_exception()\n")
	buf.WriteString("  call void @jayess_runtime_shutdown()\n")
	buf.WriteString("  ret i32 1\n")
	buf.WriteString("exit.ok:\n")
	buf.WriteString("  call void @jayess_runtime_shutdown()\n")
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
			storageKind := value.kind
			var slot string
			if stmt.Kind == ir.DeclarationVar {
				storageKind = ir.ValueDynamic
				if hoisted, ok := state.hoistedVarSlots[stmt.Name]; ok {
					slot = hoisted.ptr
					state.slots[stmt.Name] = hoisted
				} else {
					slot = state.nextTemp()
				}
			} else {
				slot = state.nextTemp()
			}
			typ := llvmStorageType(storageKind)
			if stmt.Kind != ir.DeclarationVar || state.hoistedVarSlots[stmt.Name].ptr == "" {
				fmt.Fprintf(buf, "  %s = alloca %s\n", slot, typ)
			}
			slotState := variableSlot{kind: storageKind, ptr: slot}
			if stmt.Kind == ir.DeclarationVar {
				if hoisted, ok := state.hoistedVarSlots[stmt.Name]; ok && hoisted.ptr == slot {
					slotState = hoisted
				}
			}
			if stmt.Kind == ir.DeclarationVar && storageKind == ir.ValueDynamic && state.hoistedVarSlots[stmt.Name].ptr != "" && shouldScheduleDynamicLocalCleanup(stmt.Value, value) {
				previous := state.nextTemp()
				fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", previous, slot)
				fmt.Fprintf(buf, "  call void @jayess_value_free_unshared(ptr %s)\n", previous)
			}
			slotState, err = g.emitStoreIntoVariableSlot(buf, state, slotState, value, stmt.Value)
			if err != nil {
				return false, err
			}
			if stmt.Kind == ir.DeclarationVar {
				state.slots[stmt.Name] = slotState
				if _, ok := state.hoistedVarSlots[stmt.Name]; ok {
					state.hoistedVarSlots[stmt.Name] = slotState
				}
			} else {
				state.declareSlot(stmt.Name, slotState)
			}
			if storageKind == ir.ValueDynamic && state.isEligibleLocal(stmt.Name, stmt.Line, stmt.Column) && slotState.ownsCleanup {
				if stmt.Kind == ir.DeclarationVar {
					if state.isFunctionScopedVarCleanupEligible(stmt.Name, stmt.Line, stmt.Column) && state.hoistedVarSlots[stmt.Name].ptr == "" {
						state.addRootCleanup(variableSlot{kind: storageKind, ptr: slot, ownsCleanup: true})
					}
				} else {
					state.addCleanup(variableSlot{kind: storageKind, ptr: slot, ownsCleanup: true})
				}
			}
		case *ir.AssignmentStatement:
			if err := g.emitAssignment(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.ExpressionStatement:
			value, err := g.emitExpression(buf, state, stmt.Expression)
			if err != nil {
				return false, err
			}
			if state.shouldCleanupDiscardedExpression(stmt.Expression, value) {
				boxed, err := g.emitBoxedValue(buf, state, value)
				if err != nil {
					return false, err
				}
				g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: true})
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
			g.emitCleanupAllScopes(buf, state)
			buf.WriteString("  call void @jayess_pop_call_frame()\n")
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
		case *ir.BlockStatement:
			terminated, err := g.emitScopedStatements(buf, state, stmt.Body)
			if err != nil {
				return false, err
			}
			if terminated {
				return true, nil
			}
		case *ir.WhileStatement:
			if err := g.emitWhile(buf, state, stmt, ""); err != nil {
				return false, err
			}
		case *ir.DoWhileStatement:
			if err := g.emitDoWhile(buf, state, stmt, ""); err != nil {
				return false, err
			}
		case *ir.ForStatement:
			if err := g.emitFor(buf, state, stmt, ""); err != nil {
				return false, err
			}
		case *ir.SwitchStatement:
			if err := g.emitSwitch(buf, state, stmt, ""); err != nil {
				return false, err
			}
		case *ir.LabeledStatement:
			if err := g.emitLabeled(buf, state, stmt); err != nil {
				return false, err
			}
		case *ir.BreakStatement:
			target, depth, ok := state.resolveBreak(stmt.Label)
			if !ok {
				return false, fmt.Errorf("break used outside a valid target")
			}
			g.emitCleanupScopesToDepth(buf, state, depth)
			fmt.Fprintf(buf, "  br label %%%s\n", target)
			return true, nil
		case *ir.ContinueStatement:
			target, depth, ok := state.resolveContinue(stmt.Label)
			if !ok {
				return false, fmt.Errorf("continue used outside loop")
			}
			g.emitCleanupScopesToDepth(buf, state, depth)
			fmt.Fprintf(buf, "  br label %%%s\n", target)
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

func (g *LLVMIRGenerator) emitScopedStatements(buf *bytes.Buffer, state *functionState, statements []ir.Statement) (bool, error) {
	state.pushScope()
	terminated, err := g.emitStatements(buf, state, statements)
	if err == nil && !terminated {
		g.emitCurrentScopeCleanup(buf, state)
	}
	state.popScope()
	return terminated, err
}

func (state *functionState) resolveBreak(label string) (string, int, bool) {
	for i := len(state.controlStack) - 1; i >= 0; i-- {
		target := state.controlStack[i]
		if label == "" {
			if target.allowsUnlabeledBreak && target.breakTarget != "" {
				return target.breakTarget, target.cleanupScopeDepth, true
			}
			continue
		}
		if target.label == label && target.breakTarget != "" {
			return target.breakTarget, target.cleanupScopeDepth, true
		}
	}
	return "", 0, false
}

func (state *functionState) resolveContinue(label string) (string, int, bool) {
	for i := len(state.controlStack) - 1; i >= 0; i-- {
		target := state.controlStack[i]
		if label == "" {
			if target.label == "" && target.allowsContinue && target.continueTarget != "" {
				return target.continueTarget, target.cleanupScopeDepth, true
			}
			continue
		}
		if target.label == label && target.allowsContinue && target.continueTarget != "" {
			return target.continueTarget, target.cleanupScopeDepth, true
		}
	}
	return "", 0, false
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
	thenTerminated, err := g.emitScopedStatements(buf, state, stmt.Consequence)
	if err != nil {
		return err
	}
	if !thenTerminated {
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	}
	fmt.Fprintf(buf, "%s:\n", elseLabel)
	elseTerminated, err := g.emitScopedStatements(buf, state, stmt.Alternative)
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
	g.emitCleanupScopesToDepth(buf, state, state.exceptionCleanupDepth)
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
	outerDepth := state.exceptionCleanupDepth
	state.exceptionTarget = tryExceptionTarget
	state.exceptionCleanupDepth = len(state.scopeStack)
	tryTerminated, err := g.emitScopedStatements(buf, state, stmt.TryBody)
	if err != nil {
		return err
	}
	state.exceptionTarget = outerTarget
	state.exceptionCleanupDepth = outerDepth
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
		state.exceptionCleanupDepth = len(state.scopeStack)
		catchTerminated, err := g.emitScopedStatements(buf, state, stmt.CatchBody)
		if stmt.CatchName != "" {
			delete(state.slots, stmt.CatchName)
		}
		if err != nil {
			return err
		}
		state.exceptionTarget = outerTarget
		state.exceptionCleanupDepth = outerDepth
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
		state.exceptionCleanupDepth = outerDepth
		finallyTerminated, err := g.emitScopedStatements(buf, state, stmt.FinallyBody)
		if err != nil {
			return err
		}
		state.exceptionTarget = outerTarget
		state.exceptionCleanupDepth = outerDepth
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

func (g *LLVMIRGenerator) emitWhile(buf *bytes.Buffer, state *functionState, stmt *ir.WhileStatement, label string) error {
	condLabel := state.nextLabel("while.cond")
	bodyLabel := state.nextLabel("while.body")
	endLabel := state.nextLabel("while.end")
	if label != "" {
		state.controlStack = append(state.controlStack, controlLabels{label: label, breakTarget: endLabel, continueTarget: condLabel, cleanupScopeDepth: len(state.scopeStack), allowsContinue: true})
	}
	state.controlStack = append(state.controlStack, controlLabels{breakTarget: endLabel, continueTarget: condLabel, cleanupScopeDepth: len(state.scopeStack), allowsUnlabeledBreak: true, allowsContinue: true})
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", condLabel)
	cond, err := g.emitCondition(buf, state, stmt.Condition)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	terminated, err := g.emitScopedStatements(buf, state, stmt.Body)
	if err != nil {
		return err
	}
	if !terminated {
		fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	}
	fmt.Fprintf(buf, "%s:\n", endLabel)
	state.controlStack = state.controlStack[:len(state.controlStack)-1]
	if label != "" {
		state.controlStack = state.controlStack[:len(state.controlStack)-1]
	}
	return nil
}

func (g *LLVMIRGenerator) emitDoWhile(buf *bytes.Buffer, state *functionState, stmt *ir.DoWhileStatement, label string) error {
	bodyLabel := state.nextLabel("dowhile.body")
	condLabel := state.nextLabel("dowhile.cond")
	endLabel := state.nextLabel("dowhile.end")
	if label != "" {
		state.controlStack = append(state.controlStack, controlLabels{label: label, breakTarget: endLabel, continueTarget: condLabel, cleanupScopeDepth: len(state.scopeStack), allowsContinue: true})
	}
	state.controlStack = append(state.controlStack, controlLabels{breakTarget: endLabel, continueTarget: condLabel, cleanupScopeDepth: len(state.scopeStack), allowsUnlabeledBreak: true, allowsContinue: true})
	fmt.Fprintf(buf, "  br label %%%s\n", bodyLabel)
	fmt.Fprintf(buf, "%s:\n", bodyLabel)
	terminated, err := g.emitScopedStatements(buf, state, stmt.Body)
	if err != nil {
		return err
	}
	if !terminated {
		fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	}
	fmt.Fprintf(buf, "%s:\n", condLabel)
	cond, err := g.emitCondition(buf, state, stmt.Condition)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, bodyLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	state.controlStack = state.controlStack[:len(state.controlStack)-1]
	if label != "" {
		state.controlStack = state.controlStack[:len(state.controlStack)-1]
	}
	return nil
}

func (g *LLVMIRGenerator) emitFor(buf *bytes.Buffer, state *functionState, stmt *ir.ForStatement, label string) error {
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

	if label != "" {
		state.controlStack = append(state.controlStack, controlLabels{label: label, breakTarget: endLabel, continueTarget: continueTarget, cleanupScopeDepth: len(state.scopeStack), allowsContinue: true})
	}
	state.controlStack = append(state.controlStack, controlLabels{breakTarget: endLabel, continueTarget: continueTarget, cleanupScopeDepth: len(state.scopeStack), allowsUnlabeledBreak: true, allowsContinue: true})
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
	terminated, err := g.emitScopedStatements(buf, state, stmt.Body)
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
	state.controlStack = state.controlStack[:len(state.controlStack)-1]
	if label != "" {
		state.controlStack = state.controlStack[:len(state.controlStack)-1]
	}
	return nil
}

func (g *LLVMIRGenerator) emitSwitch(buf *bytes.Buffer, state *functionState, stmt *ir.SwitchStatement, label string) error {
	endLabel := state.nextLabel("switch.end")
	defaultLabel := state.nextLabel("switch.default")
	caseLabels := make([]string, len(stmt.Cases))
	nextLabels := make([]string, len(stmt.Cases))
	for i := range stmt.Cases {
		caseLabels[i] = state.nextLabel("switch.case")
		if i == len(stmt.Cases)-1 {
			nextLabels[i] = defaultLabel
		} else {
			nextLabels[i] = state.nextLabel("switch.next")
		}
	}

	discriminant, err := g.emitExpression(buf, state, stmt.Discriminant)
	if err != nil {
		return err
	}
	if state.exceptionTarget != "" {
		g.emitExceptionCheck(buf, state, state.exceptionTarget)
	}

	if label != "" {
		state.controlStack = append(state.controlStack, controlLabels{label: label, breakTarget: endLabel, cleanupScopeDepth: len(state.scopeStack)})
	}
	state.controlStack = append(state.controlStack, controlLabels{breakTarget: endLabel, cleanupScopeDepth: len(state.scopeStack), allowsUnlabeledBreak: true})

	if len(stmt.Cases) == 0 {
		fmt.Fprintf(buf, "  br label %%%s\n", defaultLabel)
	}

	for i, switchCase := range stmt.Cases {
		match, err := g.emitEqualityComparisonValue(buf, state, discriminant, switchCase.Test)
		if err != nil {
			return err
		}
		if state.exceptionTarget != "" {
			g.emitExceptionCheck(buf, state, state.exceptionTarget)
		}
		cond, err := g.emitTruthyFromValue(buf, state, match)
		if err != nil {
			return err
		}
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, caseLabels[i], nextLabels[i])
		fmt.Fprintf(buf, "%s:\n", caseLabels[i])
		terminated, err := g.emitScopedStatements(buf, state, switchCase.Consequent)
		if err != nil {
			return err
		}
		if !terminated {
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		}
		if i < len(stmt.Cases)-1 {
			fmt.Fprintf(buf, "%s:\n", nextLabels[i])
		}
	}

	fmt.Fprintf(buf, "%s:\n", defaultLabel)
	terminated, err := g.emitScopedStatements(buf, state, stmt.Default)
	if err != nil {
		return err
	}
	if !terminated {
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	}
	fmt.Fprintf(buf, "%s:\n", endLabel)
	if state.shouldCleanupDiscardedExpression(stmt.Discriminant, discriminant) {
		boxed, err := g.emitBoxedValue(buf, state, discriminant)
		if err != nil {
			return err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: true})
	}

	state.controlStack = state.controlStack[:len(state.controlStack)-1]
	if label != "" {
		state.controlStack = state.controlStack[:len(state.controlStack)-1]
	}
	return nil
}

func (g *LLVMIRGenerator) emitLabeled(buf *bytes.Buffer, state *functionState, stmt *ir.LabeledStatement) error {
	switch inner := stmt.Statement.(type) {
	case *ir.WhileStatement:
		return g.emitWhile(buf, state, inner, stmt.Label)
	case *ir.DoWhileStatement:
		return g.emitDoWhile(buf, state, inner, stmt.Label)
	case *ir.ForStatement:
		return g.emitFor(buf, state, inner, stmt.Label)
	case *ir.SwitchStatement:
		return g.emitSwitch(buf, state, inner, stmt.Label)
	default:
		endLabel := state.nextLabel("label.end")
		state.controlStack = append(state.controlStack, controlLabels{label: stmt.Label, breakTarget: endLabel, cleanupScopeDepth: len(state.scopeStack)})
		terminated, err := g.emitScopedStatements(buf, state, []ir.Statement{stmt.Statement})
		if err != nil {
			return err
		}
		if !terminated {
			fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		}
		fmt.Fprintf(buf, "%s:\n", endLabel)
		state.controlStack = state.controlStack[:len(state.controlStack)-1]
		return nil
	}
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
			slot, err = g.emitStoreIntoVariableSlot(buf, state, slot, value, stmt.Value)
			if err != nil {
				return err
			}
			state.slots[target.Name] = slot
			if _, hoisted := state.hoistedVarSlots[target.Name]; hoisted {
				state.hoistedVarSlots[target.Name] = slot
			}
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
			boxedObject := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxedObject, objectValue.ref)
			fmt.Fprintf(buf, "  call void @jayess_value_set_member(ptr %s, ptr %s, ptr %s)\n", boxedObject, state.stringRefs[target.Property], boxed)
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
				boxedObject := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxedObject, arrayValue.ref)
				fmt.Fprintf(buf, "  call void @jayess_value_set_member(ptr %s, ptr %s, ptr %s)\n", boxedObject, indexValue.ref, boxed)
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
	case *ir.BigIntLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_bigint(ptr %s)\n", tmp, state.stringRefs[expr.Value])
		return emittedValue{kind: ir.ValueBigInt, ref: tmp}, nil
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
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs[expr.Value], staticString: true}, nil
	case *ir.ObjectLiteral:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_object_new()\n", tmp)
		for _, property := range expr.Properties {
			if property.Spread {
				value, err := g.emitExpression(buf, state, property.Value)
				if err != nil {
					return emittedValue{}, err
				}
				boxedSource, err := g.emitBoxedValue(buf, state, value)
				if err != nil {
					return emittedValue{}, err
				}
				boxedObject := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxedObject, tmp)
				fmt.Fprintf(buf, "  call ptr @jayess_value_object_assign(ptr %s, ptr %s)\n", boxedObject, boxedSource)
				continue
			}
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
				key := property.Key
				if property.Getter {
					key = accessorStorageKeyForCodegen(true, property.Key)
				} else if property.Setter {
					key = accessorStorageKeyForCodegen(false, property.Key)
				}
				fmt.Fprintf(buf, "  call void @jayess_object_set_value(ptr %s, ptr %s, ptr %s)\n", tmp, state.stringRefs[key], boxed)
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
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_static_string(ptr %s)\n", boxedPart, state.stringRefs[part])
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
		raw := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_template_string(ptr %s, ptr %s)\n", raw, partsBoxed, valuesBoxed)
		g.emitCleanupBoxedUse(buf, boxedUse{ref: partsBoxed, cleanup: true})
		g.emitCleanupBoxedUse(buf, boxedUse{ref: valuesBoxed, cleanup: true})
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_owned_string(ptr %s)\n", tmp, raw)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		if expr.Environment != nil && paramCount > 0 {
			paramCount--
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr %s, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, emittedFunctionName(expr.Name), envRef, state.stringRefs[expr.Name], classRef, paramCount, hasRest)
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
			raw := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_concat_values(ptr %s, ptr %s)\n", raw, leftBoxed, rightBoxed)
			g.emitCleanupBoxedUse(buf, boxedUse{ref: leftBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Left, left)})
			g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_owned_string(ptr %s)\n", tmp, raw)
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		if expr.Operator == ir.OperatorAdd && (left.kind == ir.ValueDynamic || right.kind == ir.ValueDynamic) {
			leftBoxed, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			rightBoxed, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_add(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			g.emitCleanupBoxedUse(buf, boxedUse{ref: leftBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Left, left)})
			g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
			return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
		}
		switch expr.Operator {
		case ir.OperatorBitAnd, ir.OperatorBitOr, ir.OperatorBitXor, ir.OperatorShl, ir.OperatorShr, ir.OperatorUShr:
			leftBoxed, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			rightBoxed, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			switch expr.Operator {
			case ir.OperatorBitAnd:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_and(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			case ir.OperatorBitOr:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_or(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			case ir.OperatorBitXor:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_xor(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			case ir.OperatorShl:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_shl(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			case ir.OperatorShr:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_shr(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			default:
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_ushr(ptr %s, ptr %s)\n", tmp, leftBoxed, rightBoxed)
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: leftBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Left, left)})
			g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
			switch expr.Kind {
			case ir.ValueNumber:
				numberResult := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call double @jayess_value_to_number(ptr %s)\n", numberResult, tmp)
				g.emitCleanupBoxedUse(buf, boxedUse{ref: tmp, cleanup: true})
				return emittedValue{kind: ir.ValueNumber, ref: numberResult}, nil
			case ir.ValueBigInt, ir.ValueDynamic:
				return emittedValue{kind: expr.Kind, ref: tmp}, nil
			default:
				return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
			}
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
		useLeftLabel := state.nextLabel("nullish.left")
		useRightLabel := state.nextLabel("nullish.right")
		endLabel := state.nextLabel("nullish.end")
		leftNullish, err := g.emitNullishCheck(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", leftNullish, useRightLabel, useLeftLabel)
		fmt.Fprintf(buf, "%s:\n", useLeftLabel)
		boxedLeft, err := g.emitBoxedValue(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxedLeft, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
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
	case *ir.CommaExpression:
		left, err := g.emitExpression(buf, state, expr.Left)
		if err != nil {
			return emittedValue{}, err
		}
		if state.shouldCleanupDiscardedExpression(expr.Left, left) {
			boxedLeft, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedLeft, cleanup: true})
		}
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		return emittedValue{kind: expr.Kind, ref: right.ref}, nil
	case *ir.ConditionalExpression:
		condition, err := g.emitExpression(buf, state, expr.Condition)
		if err != nil {
			return emittedValue{}, err
		}
		cond, err := g.emitTruthyFromValue(buf, state, condition)
		if err != nil {
			return emittedValue{}, err
		}
		resultPtr := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
		trueLabel := state.nextLabel("cond.true")
		falseLabel := state.nextLabel("cond.false")
		endLabel := state.nextLabel("cond.end")
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", cond, trueLabel, falseLabel)
		fmt.Fprintf(buf, "%s:\n", trueLabel)
		consequent, err := g.emitExpression(buf, state, expr.Consequent)
		if err != nil {
			return emittedValue{}, err
		}
		boxedConsequent, err := g.emitBoxedValue(buf, state, consequent)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxedConsequent, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", falseLabel)
		alternative, err := g.emitExpression(buf, state, expr.Alternative)
		if err != nil {
			return emittedValue{}, err
		}
		boxedAlternative, err := g.emitBoxedValue(buf, state, alternative)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", boxedAlternative, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", endLabel)
		result := state.nextTemp()
		fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", result, resultPtr)
		return emittedValue{kind: expr.Kind, ref: result}, nil
	case *ir.UnaryExpression:
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		if expr.Operator == ir.OperatorBitNot {
			boxedRight, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_bitwise_not(ptr %s)\n", tmp, boxedRight)
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedRight, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
			switch expr.Kind {
			case ir.ValueNumber:
				numberResult := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call double @jayess_value_to_number(ptr %s)\n", numberResult, tmp)
				g.emitCleanupBoxedUse(buf, boxedUse{ref: tmp, cleanup: true})
				return emittedValue{kind: ir.ValueNumber, ref: numberResult}, nil
			case ir.ValueBigInt, ir.ValueDynamic:
				return emittedValue{kind: expr.Kind, ref: tmp}, nil
			default:
				return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
			}
		}
		cond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		if state.shouldCleanupDiscardedExpression(expr.Right, right) {
			boxedRight, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedRight, cleanup: true})
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = xor i1 %s, true\n", tmp, cond)
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	case *ir.NewTargetExpression:
		if state.classNames[state.functionName] {
			paramCount, hasRest := functionMetadata(state.functionName, state.functions, state.externs)
			tmp := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr null, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, emittedFunctionName(state.functionName), state.stringRefs[state.functionName], state.stringRefs[state.functionName], paramCount, hasRest)
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
		if state.shouldCleanupDiscardedExpression(expr.Left, left) {
			boxedLeft, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedLeft, cleanup: true})
		}
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		rightCond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		if state.shouldCleanupDiscardedExpression(expr.Right, right) {
			boxedRight, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedRight, cleanup: true})
		}
		fmt.Fprintf(buf, "  store i1 %s, ptr %s\n", rightCond, resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", shortLabel)
		if state.shouldCleanupDiscardedExpression(expr.Left, left) {
			boxedLeft, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedLeft, cleanup: true})
		}
		fmt.Fprintf(buf, "  store i1 false, ptr %s\n", resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	} else {
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", leftCond, shortLabel, rightLabel)
		fmt.Fprintf(buf, "%s:\n", shortLabel)
		if state.shouldCleanupDiscardedExpression(expr.Left, left) {
			boxedLeft, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedLeft, cleanup: true})
		}
		fmt.Fprintf(buf, "  store i1 true, ptr %s\n", resultPtr)
		fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
		fmt.Fprintf(buf, "%s:\n", rightLabel)
		if state.shouldCleanupDiscardedExpression(expr.Left, left) {
			boxedLeft, err := g.emitBoxedValue(buf, state, left)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedLeft, cleanup: true})
		}
		right, err := g.emitExpression(buf, state, expr.Right)
		if err != nil {
			return emittedValue{}, err
		}
		rightCond, err := g.emitTruthyFromValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		if state.shouldCleanupDiscardedExpression(expr.Right, right) {
			boxedRight, err := g.emitBoxedValue(buf, state, right)
			if err != nil {
				return emittedValue{}, err
			}
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedRight, cleanup: true})
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
	case ir.ValueBigInt:
		return emittedValue{kind: ir.ValueString, ref: state.stringRefs["bigint"]}, nil
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
		g.emitCleanupBoxedUse(buf, boxedUse{ref: value.ref, cleanup: shouldCleanupBoxedValueAfterUse(expr.Value, value)})
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
		g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
	}
	tmp := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_value_instanceof(ptr %s, ptr %s)\n", tmp, leftBoxed, classNameRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: leftBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Left, left)})
	return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
}

func boolLiteralRef(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (g *LLVMIRGenerator) emitKindEquals(buf *bytes.Buffer, state *functionState, boxedRef string, kind int) string {
	kindRef := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_kind_of(ptr %s)\n", kindRef, boxedRef)
	result := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, %d\n", result, kindRef, kind)
	return result
}

func (g *LLVMIRGenerator) emitLiteralTypeCheck(buf *bytes.Buffer, state *functionState, boxedRef string, literal string) (string, error) {
	var literalBoxed string
	switch literal {
	case "true", "false":
		literalBoxed = state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_bool(i1 %s)\n", literalBoxed, literal)
	case "null":
		literalBoxed = state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_null()\n", literalBoxed)
	default:
		if strings.HasPrefix(literal, "\"") {
			text, err := strconv.Unquote(literal)
			if err != nil {
				return "", err
			}
			literalBoxed = state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_static_string(ptr %s)\n", literalBoxed, state.stringRefs[text])
		} else {
			number, err := strconv.ParseFloat(literal, 64)
			if err != nil {
				return "", err
			}
			literalBoxed = state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_number(double %s)\n", literalBoxed, formatFloat(number))
		}
	}
	result := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_value_eq(ptr %s, ptr %s)\n", result, boxedRef, literalBoxed)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: literalBoxed, cleanup: true})
	return result, nil
}

func (g *LLVMIRGenerator) emitGuardedTypeCheck(buf *bytes.Buffer, state *functionState, guardRef string, build func() (string, error)) (string, error) {
	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca i1\n", resultPtr)
	fmt.Fprintf(buf, "  store i1 false, ptr %s\n", resultPtr)
	thenLabel := state.nextLabel("typecheck.then")
	endLabel := state.nextLabel("typecheck.end")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", guardRef, thenLabel, endLabel)
	fmt.Fprintf(buf, "%s:\n", thenLabel)
	innerRef, err := build()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(buf, "  store i1 %s, ptr %s\n", innerRef, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	result := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load i1, ptr %s\n", result, resultPtr)
	return result, nil
}

func (g *LLVMIRGenerator) emitRuntimeTypeCheck(buf *bytes.Buffer, state *functionState, boxedRef string, typeExpr *typesys.Expr) (string, error) {
	if typeExpr == nil {
		return boolLiteralRef(true), nil
	}
	switch typeExpr.Kind {
	case typesys.KindAny:
		return boolLiteralRef(true), nil
	case typesys.KindSimple:
		switch typeExpr.Name {
		case "", "any", "dynamic", "unknown":
			return boolLiteralRef(true), nil
		case "never":
			return boolLiteralRef(false), nil
		case "number":
			return g.emitKindEquals(buf, state, boxedRef, 2), nil
		case "bigint":
			return g.emitKindEquals(buf, state, boxedRef, 3), nil
		case "boolean":
			return g.emitKindEquals(buf, state, boxedRef, 4), nil
		case "string":
			return g.emitKindEquals(buf, state, boxedRef, 1), nil
		case "symbol":
			return g.emitKindEquals(buf, state, boxedRef, 9), nil
		case "function":
			return g.emitKindEquals(buf, state, boxedRef, 8), nil
		case "null":
			return g.emitKindEquals(buf, state, boxedRef, 0), nil
		case "undefined", "void":
			return g.emitKindEquals(buf, state, boxedRef, 7), nil
		case "array":
			return g.emitKindEquals(buf, state, boxedRef, 6), nil
		case "object":
			return g.emitKindEquals(buf, state, boxedRef, 5), nil
		default:
			result := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i1 @jayess_value_instanceof(ptr %s, ptr %s)\n", result, boxedRef, state.stringRefs[typeExpr.Name])
			return result, nil
		}
	case typesys.KindLiteral:
		return g.emitLiteralTypeCheck(buf, state, boxedRef, typeExpr.Name)
	case typesys.KindUnion:
		if len(typeExpr.Elements) == 0 {
			return boolLiteralRef(false), nil
		}
		var current string
		for i, element := range typeExpr.Elements {
			match, err := g.emitRuntimeTypeCheck(buf, state, boxedRef, element)
			if err != nil {
				return "", err
			}
			if i == 0 {
				current = match
				continue
			}
			combined := state.nextTemp()
			fmt.Fprintf(buf, "  %s = or i1 %s, %s\n", combined, current, match)
			current = combined
		}
		return current, nil
	case typesys.KindIntersection:
		if len(typeExpr.Elements) == 0 {
			return boolLiteralRef(true), nil
		}
		var current string
		for i, element := range typeExpr.Elements {
			match, err := g.emitRuntimeTypeCheck(buf, state, boxedRef, element)
			if err != nil {
				return "", err
			}
			if i == 0 {
				current = match
				continue
			}
			combined := state.nextTemp()
			fmt.Fprintf(buf, "  %s = and i1 %s, %s\n", combined, current, match)
			current = combined
		}
		return current, nil
	case typesys.KindFunction:
		return g.emitKindEquals(buf, state, boxedRef, 8), nil
	case typesys.KindTuple:
		guardRef := g.emitKindEquals(buf, state, boxedRef, 6)
		return g.emitGuardedTypeCheck(buf, state, guardRef, func() (string, error) {
			lengthRef := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call i32 @jayess_value_array_length(ptr %s)\n", lengthRef, boxedRef)
			lengthOk := state.nextTemp()
			fmt.Fprintf(buf, "  %s = icmp eq i32 %s, %d\n", lengthOk, lengthRef, len(typeExpr.Elements))
			current := lengthOk
			for index, element := range typeExpr.Elements {
				itemRef := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_index(ptr %s, i32 %d)\n", itemRef, boxedRef, index)
				itemOk, err := g.emitRuntimeTypeCheck(buf, state, itemRef, element)
				if err != nil {
					return "", err
				}
				combined := state.nextTemp()
				fmt.Fprintf(buf, "  %s = and i1 %s, %s\n", combined, current, itemOk)
				current = combined
			}
			return current, nil
		})
	case typesys.KindObject:
		guardRef := g.emitKindEquals(buf, state, boxedRef, 5)
		return g.emitGuardedTypeCheck(buf, state, guardRef, func() (string, error) {
			current := boolLiteralRef(true)
			for _, property := range typeExpr.Properties {
				memberRef := state.nextTemp()
				fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", memberRef, boxedRef, state.stringRefs[property.Name])
				memberKindRef := g.emitKindEquals(buf, state, memberRef, 7)
				if property.Optional {
					matchesRef, err := g.emitRuntimeTypeCheck(buf, state, memberRef, property.Type)
					if err != nil {
						return "", err
					}
					optionalRef := state.nextTemp()
					fmt.Fprintf(buf, "  %s = or i1 %s, %s\n", optionalRef, memberKindRef, matchesRef)
					matchesRef = optionalRef
					combined := state.nextTemp()
					fmt.Fprintf(buf, "  %s = and i1 %s, %s\n", combined, current, matchesRef)
					current = combined
					continue
				}
				matchesRef, err := g.emitRuntimeTypeCheck(buf, state, memberRef, property.Type)
				if err != nil {
					return "", err
				}
				combined := state.nextTemp()
				fmt.Fprintf(buf, "  %s = and i1 %s, %s\n", combined, current, matchesRef)
				current = combined
			}
			return current, nil
		})
	default:
		return boolLiteralRef(false), nil
	}
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
		if left.kind == ir.ValueDynamic || right.kind == ir.ValueDynamic || left.kind == ir.ValueFunction || right.kind == ir.ValueFunction || left.kind == ir.ValueBigInt || right.kind == ir.ValueBigInt {
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
			g.emitCleanupBoxedUse(buf, boxedUse{ref: leftBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Left, left)})
			g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(expr.Right, right)})
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

func (g *LLVMIRGenerator) emitEqualityComparisonValue(buf *bytes.Buffer, state *functionState, left emittedValue, rightExpr ir.Expression) (emittedValue, error) {
	right, err := g.emitExpression(buf, state, rightExpr)
	if err != nil {
		return emittedValue{}, err
	}

	if left.kind == ir.ValueBoolean && right.kind == ir.ValueBoolean {
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = icmp eq i1 %s, %s\n", tmp, left.ref, right.ref)
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	}
	if left.kind == ir.ValueDynamic || right.kind == ir.ValueDynamic || left.kind == ir.ValueFunction || right.kind == ir.ValueFunction || left.kind == ir.ValueBigInt || right.kind == ir.ValueBigInt {
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
		g.emitCleanupBoxedUse(buf, boxedUse{ref: rightBoxed, cleanup: shouldCleanupBoxedValueAfterUse(rightExpr, right)})
		return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
	}
	if left.kind == ir.ValueString && right.kind == ir.ValueString {
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_string_eq(ptr %s, ptr %s)\n", tmp, left.ref, right.ref)
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
	fmt.Fprintf(buf, "  %s = fcmp oeq double %s, %s\n", tmp, leftRef, rightRef)
	return emittedValue{kind: ir.ValueBoolean, ref: tmp}, nil
}

func (g *LLVMIRGenerator) emitCall(buf *bytes.Buffer, state *functionState, call *ir.CallExpression) (emittedValue, error) {
	switch call.Callee {
	case "__jayess_type_is":
		if len(call.Arguments) != 2 {
			return emittedValue{}, fmt.Errorf("__jayess_type_is expects 2 arguments")
		}
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		annotationLiteral, ok := call.Arguments[1].(*ir.StringLiteral)
		if !ok {
			return emittedValue{}, fmt.Errorf("__jayess_type_is expects a string literal type annotation")
		}
		typeExpr, err := typesys.Parse(annotationLiteral.Value)
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		matchRef, err := g.emitRuntimeTypeCheck(buf, state, boxed, typeExpr)
		if err != nil {
			return emittedValue{}, err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[0], value)})
		return emittedValue{kind: ir.ValueBoolean, ref: matchRef}, nil
	case "print":
		if len(call.Arguments) == 1 {
			arg, err := g.emitExpression(buf, state, call.Arguments[0])
			if err != nil {
				return emittedValue{}, err
			}
			switch arg.kind {
			case ir.ValueNumber:
				fmt.Fprintf(buf, "  call void @jayess_print_number(double %s)\n", arg.ref)
			case ir.ValueBigInt:
				fmt.Fprintf(buf, "  call void @jayess_print_value(ptr %s)\n", arg.ref)
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
	case "compile", "compileFile":
		source, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedSource, err := g.emitBoxedValue(buf, state, source)
		if err != nil {
			return emittedValue{}, err
		}
		var boxedOutput string
		if len(call.Arguments) > 1 {
			output, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			boxedOutput, err = g.emitBoxedValue(buf, state, output)
			if err != nil {
				return emittedValue{}, err
			}
		} else {
			boxedOutput = state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", boxedOutput)
		}
		tmp := state.nextTemp()
		runtimeName := "jayess_std_compile"
		if call.Callee == "compileFile" {
			runtimeName = "jayess_std_compile_file"
		}
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedSource, boxedOutput)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_console_log", "__jayess_console_warn", "__jayess_console_error":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  call void @jayess_%s(ptr %s)\n", runtimeName, argsBoxed)
		fmt.Fprintf(buf, "  call void @jayess_value_free_unshared(ptr %s)\n", argsBoxed)
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_arch":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_arch()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_tmpdir":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_tmpdir()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_hostname":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_hostname()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_uptime":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_std_process_uptime()\n", tmp)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_process_hrtime":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_std_process_hrtime()\n", tmp)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_process_cpu_info":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_cpu_info()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_memory_info":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_memory_info()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_user_info":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_user_info()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_thread_pool_size":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_thread_pool_size()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_resolve":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_resolve(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_sep":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_sep()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_path_delimiter":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_path_delimiter()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_tls_is_available":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_tls_is_available()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_tls_backend":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_tls_backend()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_tls_connect":
		argValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedArg, err := g.emitBoxedValue(buf, state, argValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_tls_connect(ptr %s)\n", tmp, boxedArg)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_tls_create_server":
		optionsValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedOptions, err := g.emitBoxedValue(buf, state, optionsValue)
		if err != nil {
			return emittedValue{}, err
		}
		handlerValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedHandler, err := g.emitBoxedValue(buf, state, handlerValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_tls_create_server(ptr %s, ptr %s)\n", tmp, boxedOptions, boxedHandler)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_https_create_server":
		optionsValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedOptions, err := g.emitBoxedValue(buf, state, optionsValue)
		if err != nil {
			return emittedValue{}, err
		}
		handlerValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedHandler, err := g.emitBoxedValue(buf, state, handlerValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_https_create_server(ptr %s, ptr %s)\n", tmp, boxedOptions, boxedHandler)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_https_is_available":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_https_is_available()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_https_backend":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_https_backend()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_compression_create_gzip_stream", "__jayess_compression_create_gunzip_stream", "__jayess_compression_create_deflate_stream", "__jayess_compression_create_inflate_stream", "__jayess_compression_create_brotli_stream", "__jayess_compression_create_unbrotli_stream", "__jayess_dns_clear_resolver":
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s()\n", tmp, runtimeName)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s(ptr %s)\n", tmp, runtimeName, boxedPath)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_url_parse", "__jayess_url_format", "__jayess_querystring_parse", "__jayess_querystring_stringify", "__jayess_dns_lookup", "__jayess_dns_lookup_all", "__jayess_dns_reverse", "__jayess_dns_set_resolver", "__jayess_child_process_exec", "__jayess_child_process_spawn", "__jayess_child_process_kill", "__jayess_worker_create", "__jayess_shared_array_buffer_new", "__jayess_crypto_random_bytes", "__jayess_crypto_encrypt", "__jayess_crypto_decrypt", "__jayess_crypto_generate_key_pair", "__jayess_crypto_public_encrypt", "__jayess_crypto_private_decrypt", "__jayess_crypto_sign", "__jayess_crypto_verify", "__jayess_compression_gzip", "__jayess_compression_gunzip", "__jayess_compression_deflate", "__jayess_compression_inflate", "__jayess_compression_brotli", "__jayess_compression_unbrotli", "__jayess_net_is_ip", "__jayess_net_create_datagram_socket", "__jayess_net_connect", "__jayess_net_listen", "__jayess_http_parse_request", "__jayess_http_format_request", "__jayess_http_parse_response", "__jayess_http_format_response", "__jayess_http_request", "__jayess_http_create_server", "__jayess_http_request_stream", "__jayess_http_request_stream_async", "__jayess_http_get", "__jayess_http_get_stream", "__jayess_http_get_stream_async", "__jayess_http_request_async", "__jayess_http_get_async", "__jayess_https_request", "__jayess_https_request_stream", "__jayess_https_request_stream_async", "__jayess_https_get", "__jayess_https_get_stream", "__jayess_https_get_stream_async", "__jayess_https_request_async", "__jayess_https_get_async":
		argValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedArg, err := g.emitBoxedValue(buf, state, argValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s(ptr %s)\n", tmp, runtimeName, boxedArg)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_on_signal", "__jayess_process_once_signal":
		signalValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		callbackValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedSignal, err := g.emitBoxedValue(buf, state, signalValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedCallback, err := g.emitBoxedValue(buf, state, callbackValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedSignal, boxedCallback)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_off_signal":
		signalValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedSignal, err := g.emitBoxedValue(buf, state, signalValue)
		if err != nil {
			return emittedValue{}, err
		}
		callbackRef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", callbackRef)
		if len(call.Arguments) > 1 {
			callbackValue, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			callbackRef, err = g.emitBoxedValue(buf, state, callbackValue)
			if err != nil {
				return emittedValue{}, err
			}
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_off_signal(ptr %s, ptr %s)\n", tmp, boxedSignal, callbackRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_process_raise":
		signalValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedSignal, err := g.emitBoxedValue(buf, state, signalValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_process_raise(ptr %s)\n", tmp, boxedSignal)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_crypto_hash", "__jayess_crypto_secure_compare":
		leftValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		rightValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedLeft, err := g.emitBoxedValue(buf, state, leftValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedRight, err := g.emitBoxedValue(buf, state, rightValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedLeft, boxedRight)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_crypto_hmac":
		algorithmValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		keyValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		valueValue, err := g.emitExpression(buf, state, call.Arguments[2])
		if err != nil {
			return emittedValue{}, err
		}
		boxedAlgorithm, err := g.emitBoxedValue(buf, state, algorithmValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedKey, err := g.emitBoxedValue(buf, state, keyValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedValue, err := g.emitBoxedValue(buf, state, valueValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_crypto_hmac(ptr %s, ptr %s, ptr %s)\n", tmp, boxedAlgorithm, boxedKey, boxedValue)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_atomics_load":
		left, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		right, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedLeft, err := g.emitBoxedValue(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		boxedRight, err := g.emitBoxedValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_atomics_load(ptr %s, ptr %s)\n", tmp, boxedLeft, boxedRight)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_atomics_store", "__jayess_atomics_add", "__jayess_atomics_sub", "__jayess_atomics_and", "__jayess_atomics_or", "__jayess_atomics_xor", "__jayess_atomics_exchange":
		a0, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		a1, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		a2, err := g.emitExpression(buf, state, call.Arguments[2])
		if err != nil {
			return emittedValue{}, err
		}
		b0, err := g.emitBoxedValue(buf, state, a0)
		if err != nil {
			return emittedValue{}, err
		}
		b1, err := g.emitBoxedValue(buf, state, a1)
		if err != nil {
			return emittedValue{}, err
		}
		b2, err := g.emitBoxedValue(buf, state, a2)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call double @jayess_%s(ptr %s, ptr %s, ptr %s)\n", tmp, runtimeName, b0, b1, b2)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_atomics_compareExchange":
		a0, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		a1, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		a2, err := g.emitExpression(buf, state, call.Arguments[2])
		if err != nil {
			return emittedValue{}, err
		}
		a3, err := g.emitExpression(buf, state, call.Arguments[3])
		if err != nil {
			return emittedValue{}, err
		}
		b0, err := g.emitBoxedValue(buf, state, a0)
		if err != nil {
			return emittedValue{}, err
		}
		b1, err := g.emitBoxedValue(buf, state, a1)
		if err != nil {
			return emittedValue{}, err
		}
		b2, err := g.emitBoxedValue(buf, state, a2)
		if err != nil {
			return emittedValue{}, err
		}
		b3, err := g.emitBoxedValue(buf, state, a3)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_atomics_compareExchange(ptr %s, ptr %s, ptr %s, ptr %s)\n", tmp, b0, b1, b2, b3)
		return emittedValue{kind: ir.ValueNumber, ref: tmp}, nil
	case "__jayess_fs_read_file", "__jayess_fs_read_file_async":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		encodingRef := "null"
		encodingValue := emittedValue{kind: ir.ValueUndefined}
		if len(call.Arguments) > 1 {
			encodingValue, err = g.emitExpression(buf, state, call.Arguments[1])
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
		runtimeName := "jayess_std_fs_read_file"
		if call.Callee == "__jayess_fs_read_file_async" {
			runtimeName = "jayess_std_fs_read_file_async"
		}
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedPath, encodingRef)
		g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedPath, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[0], pathValue)})
		if len(call.Arguments) > 1 {
			g.emitCleanupBoxedUse(buf, boxedUse{ref: encodingRef, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], encodingValue)})
		} else {
			g.emitCleanupBoxedUse(buf, boxedUse{ref: encodingRef, cleanup: true})
		}
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_write_file", "__jayess_fs_append_file", "__jayess_fs_write_file_async":
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
		runtimeName := "jayess_std_fs_write_file"
		if call.Callee == "__jayess_fs_append_file" {
			runtimeName = "jayess_std_fs_append_file"
		} else if call.Callee == "__jayess_fs_write_file_async" {
			runtimeName = "jayess_std_fs_write_file_async"
		}
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedPath, boxedContent)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_fs_create_read_stream", "__jayess_fs_create_write_stream":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := "jayess_std_fs_create_read_stream"
		if call.Callee == "__jayess_fs_create_write_stream" {
			runtimeName = "jayess_std_fs_create_write_stream"
		}
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s)\n", tmp, runtimeName, boxedPath)
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
	case "__jayess_fs_watch":
		pathValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPath, err := g.emitBoxedValue(buf, state, pathValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_fs_watch(ptr %s)\n", tmp, boxedPath)
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
	case "__jayess_fs_copy_file", "__jayess_fs_copy_dir", "__jayess_fs_rename", "__jayess_fs_symlink":
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
		runtimeName := strings.TrimPrefix(call.Callee, "__jayess_")
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_%s(ptr %s, ptr %s)\n", tmp, runtimeName, boxedFrom, boxedTo)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "readLine":
		prompt, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPrompt, err := g.emitBoxedValue(buf, state, prompt)
		if err != nil {
			return emittedValue{}, err
		}
		raw := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_read_line_value(ptr %s)\n", raw, boxedPrompt)
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_owned_string(ptr %s)\n", tmp, raw)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "readKey":
		prompt, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedPrompt, err := g.emitBoxedValue(buf, state, prompt)
		if err != nil {
			return emittedValue{}, err
		}
		raw := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_read_key_value(ptr %s)\n", raw, boxedPrompt)
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_owned_string(ptr %s)\n", tmp, raw)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
	case "sleepAsync", "__jayess_timers_sleep":
		delay, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedDelay, err := g.emitBoxedValue(buf, state, delay)
		if err != nil {
			return emittedValue{}, err
		}
		valueRef := ""
		if len(call.Arguments) > 1 {
			value, err := g.emitExpression(buf, state, call.Arguments[1])
			if err != nil {
				return emittedValue{}, err
			}
			valueRef, err = g.emitBoxedValue(buf, state, value)
			if err != nil {
				return emittedValue{}, err
			}
		} else {
			valueRef = state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", valueRef)
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_sleep_async(ptr %s, ptr %s)\n", tmp, boxedDelay, valueRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "setTimeout", "__jayess_timers_set_timeout":
		callback, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		delay, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedCallback, err := g.emitBoxedValue(buf, state, callback)
		if err != nil {
			return emittedValue{}, err
		}
		boxedDelay, err := g.emitBoxedValue(buf, state, delay)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_set_timeout(ptr %s, ptr %s)\n", tmp, boxedCallback, boxedDelay)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "clearTimeout", "__jayess_timers_clear_timeout":
		id, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedID, err := g.emitBoxedValue(buf, state, id)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_clear_timeout(ptr %s)\n", tmp, boxedID)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
		startRef, err := g.emitNumberOperand(buf, state, start)
		if err != nil {
			return emittedValue{}, err
		}
		startInt := state.nextTemp()
		fmt.Fprintf(buf, "  %s = fptosi double %s to i32\n", startInt, startRef)
		endInt := "0"
		hasEnd := "false"
		if _, omittedEnd := call.Arguments[2].(*ir.UndefinedLiteral); !omittedEnd {
			end, err := g.emitExpression(buf, state, call.Arguments[2])
			if err != nil {
				return emittedValue{}, err
			}
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
	case "__jayess_object_symbols":
		target, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxedTarget, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_object_symbols(ptr %s)\n", tmp, boxedTarget)
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
	case "__jayess_std_weak_map_new":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_weak_map_new()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_weak_set_new":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_weak_set_new()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_symbol":
		argRef := state.nextTemp()
		if len(call.Arguments) == 0 {
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
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_symbol(ptr %s)\n", tmp, argRef)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_symbol_for", "__jayess_std_symbol_key_for":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := "jayess_std_symbol_for"
		if call.Callee == "__jayess_std_symbol_key_for" {
			runtimeName = "jayess_std_symbol_key_for"
		}
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s)\n", tmp, runtimeName, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_symbol_iterator", "__jayess_std_symbol_async_iterator", "__jayess_std_symbol_to_string_tag", "__jayess_std_symbol_has_instance", "__jayess_std_symbol_species", "__jayess_std_symbol_match", "__jayess_std_symbol_replace", "__jayess_std_symbol_search", "__jayess_std_symbol_split", "__jayess_std_symbol_to_primitive":
		tmp := state.nextTemp()
		runtimeName := map[string]string{
			"__jayess_std_symbol_iterator":       "jayess_std_symbol_iterator",
			"__jayess_std_symbol_async_iterator": "jayess_std_symbol_async_iterator",
			"__jayess_std_symbol_to_string_tag":  "jayess_std_symbol_to_string_tag",
			"__jayess_std_symbol_has_instance":   "jayess_std_symbol_has_instance",
			"__jayess_std_symbol_species":        "jayess_std_symbol_species",
			"__jayess_std_symbol_match":          "jayess_std_symbol_match",
			"__jayess_std_symbol_replace":        "jayess_std_symbol_replace",
			"__jayess_std_symbol_search":         "jayess_std_symbol_search",
			"__jayess_std_symbol_split":          "jayess_std_symbol_split",
			"__jayess_std_symbol_to_primitive":   "jayess_std_symbol_to_primitive",
		}[call.Callee]
		fmt.Fprintf(buf, "  %s = call ptr @%s()\n", tmp, runtimeName)
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
	case "__jayess_std_error_new":
		name, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		message, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedName, err := g.emitBoxedValue(buf, state, name)
		if err != nil {
			return emittedValue{}, err
		}
		boxedMessage, err := g.emitBoxedValue(buf, state, message)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_error_new(ptr %s, ptr %s)\n", tmp, boxedName, boxedMessage)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_aggregate_error_new":
		errorsValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		message, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedErrors, err := g.emitBoxedValue(buf, state, errorsValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedMessage, err := g.emitBoxedValue(buf, state, message)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_aggregate_error_new(ptr %s, ptr %s)\n", tmp, boxedErrors, boxedMessage)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_array_buffer_new", "__jayess_std_int8_array_new", "__jayess_std_uint8_array_new", "__jayess_std_uint16_array_new", "__jayess_std_int16_array_new", "__jayess_std_uint32_array_new", "__jayess_std_int32_array_new", "__jayess_std_float32_array_new", "__jayess_std_float64_array_new", "__jayess_std_data_view_new", "__jayess_std_iterator_from", "__jayess_std_async_iterator_from", "__jayess_std_promise_resolve", "__jayess_std_promise_reject", "__jayess_std_promise_all", "__jayess_std_promise_race", "__jayess_std_promise_all_settled", "__jayess_std_promise_any", "__jayess_await":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		runtimeName := strings.TrimPrefix(call.Callee, "__")
		fmt.Fprintf(buf, "  %s = call ptr @%s(ptr %s)\n", tmp, runtimeName, boxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_uint8_array_from_string":
		value, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		encoding, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedValue, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return emittedValue{}, err
		}
		boxedEncoding, err := g.emitBoxedValue(buf, state, encoding)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_uint8_array_from_string(ptr %s, ptr %s)\n", tmp, boxedValue, boxedEncoding)
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
	case "__jayess_constructor_return":
		selfValue, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		returnValue, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedSelf, err := g.emitBoxedValue(buf, state, selfValue)
		if err != nil {
			return emittedValue{}, err
		}
		boxedReturn, err := g.emitBoxedValue(buf, state, returnValue)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_constructor_return(ptr %s, ptr %s)\n", tmp, boxedSelf, boxedReturn)
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
		fmt.Fprintf(buf, "  %s = call double @%s(double %s)\n", tmp, strings.TrimPrefix(call.Callee, "__"), number)
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
		fmt.Fprintf(buf, "  %s = call double @%s(double %s, double %s)\n", tmp, strings.TrimPrefix(call.Callee, "__"), leftRef, rightRef)
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
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
	case "__jayess_std_uint8_array_concat":
		argsBoxed, err := g.emitBoxedArrayFromExpressions(buf, state, call.Arguments)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_uint8_array_concat(ptr %s)\n", tmp, argsBoxed)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_uint8_array_equals":
		left, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		right, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedLeft, err := g.emitBoxedValue(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		boxedRight, err := g.emitBoxedValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_uint8_array_equals(ptr %s, ptr %s)\n", tmp, boxedLeft, boxedRight)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	case "__jayess_std_uint8_array_compare":
		left, err := g.emitExpression(buf, state, call.Arguments[0])
		if err != nil {
			return emittedValue{}, err
		}
		right, err := g.emitExpression(buf, state, call.Arguments[1])
		if err != nil {
			return emittedValue{}, err
		}
		boxedLeft, err := g.emitBoxedValue(buf, state, left)
		if err != nil {
			return emittedValue{}, err
		}
		boxedRight, err := g.emitBoxedValue(buf, state, right)
		if err != nil {
			return emittedValue{}, err
		}
		boxedResult := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_std_uint8_array_compare(ptr %s, ptr %s)\n", boxedResult, boxedLeft, boxedRight)
		numberResult := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call double @jayess_value_to_number(ptr %s)\n", numberResult, boxedResult)
		return emittedValue{kind: ir.ValueNumber, ref: numberResult}, nil
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
		g.emitCleanupBoxedUse(buf, boxedUse{ref: callbackBoxed, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], callback)})
		if target.kind == ir.ValueArgsArray {
			fmt.Fprintf(buf, "  call void @jayess_value_free_array_shallow(ptr %s)\n", itemsRef)
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
		result, err := g.emitArrayMapCall(buf, state, itemsRef, callbackBoxed)
		if err != nil {
			return emittedValue{}, err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: callbackBoxed, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], callback)})
		if target.kind == ir.ValueArgsArray {
			fmt.Fprintf(buf, "  call void @jayess_value_free_array_shallow(ptr %s)\n", itemsRef)
		}
		return result, nil
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
		result, err := g.emitArrayFilterCall(buf, state, itemsRef, callbackBoxed)
		if err != nil {
			return emittedValue{}, err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: callbackBoxed, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], callback)})
		if target.kind == ir.ValueArgsArray {
			fmt.Fprintf(buf, "  call void @jayess_value_free_array_shallow(ptr %s)\n", itemsRef)
		}
		return result, nil
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
		result, err := g.emitArrayFindCall(buf, state, itemsRef, callbackBoxed)
		if err != nil {
			return emittedValue{}, err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: callbackBoxed, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], callback)})
		if target.kind == ir.ValueArgsArray {
			fmt.Fprintf(buf, "  call void @jayess_value_free_array_shallow(ptr %s)\n", itemsRef)
		}
		return result, nil
	case "__jayess_current_this":
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_current_this()\n", tmp)
		return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
	default:
		if ext, ok := state.externs[call.Callee]; ok {
			var args []string
			var cleanupUses []boxedUse
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
				cleanupUses = append(cleanupUses, boxedUse{
					ref:     boxed,
					cleanup: ext.BorrowsArgs && shouldCleanupBoxedValueAfterUse(argExpr, argValue),
				})
			}
			tmp := state.nextTemp()
			if len(args) == 0 {
				fmt.Fprintf(buf, "  %s = call ptr @%s()\n", tmp, ext.SymbolName)
			} else {
				fmt.Fprintf(buf, "  %s = call ptr @%s(%s)\n", tmp, ext.SymbolName, strings.Join(args, ", "))
			}
			for _, use := range cleanupUses {
				g.emitCleanupBoxedUse(buf, use)
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
		var cleanupUses []boxedUse
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
			cleanupUses = append(cleanupUses, boxedUse{ref: boxed, cleanup: shouldCleanupTransientParserArgAfterUse(call.Callee, argExpr, argValue)})
		}
		undefThis := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefThis)
		return g.emitDirectJayessCallWithThis(buf, state, call.Callee, fn, undefThis, args, cleanupUses)
	}
}

func (g *LLVMIRGenerator) emitInvoke(buf *bytes.Buffer, state *functionState, call *ir.InvokeExpression) (emittedValue, error) {
	if member, ok := call.Callee.(*ir.MemberExpression); ok && !member.Optional && isDirectFunctionReceiver(member.Target) {
		switch member.Property {
		case "bind":
			thisArg := ir.Expression(&ir.UndefinedLiteral{})
			if len(call.Arguments) > 0 {
				thisArg = call.Arguments[0]
			}
			boundArgs := &ir.ArrayLiteral{Elements: append([]ir.Expression(nil), call.Arguments[1:]...)}
			return g.emitBind(buf, state, &ir.CallExpression{
				Callee:    "__jayess_bind",
				Arguments: []ir.Expression{member.Target, thisArg, boundArgs},
			})
		case "apply":
			thisArg := ir.Expression(&ir.UndefinedLiteral{})
			if len(call.Arguments) > 0 {
				thisArg = call.Arguments[0]
			}
			argsArray := ir.Expression(&ir.ArrayLiteral{})
			if len(call.Arguments) > 1 {
				argsArray = call.Arguments[1]
			}
			return g.emitApply(buf, state, &ir.CallExpression{
				Callee:    "__jayess_apply",
				Arguments: []ir.Expression{member.Target, thisArg, argsArray},
			})
		case "call":
			thisArg := ir.Expression(&ir.UndefinedLiteral{})
			if len(call.Arguments) > 0 {
				thisArg = call.Arguments[0]
			}
			argsArray := &ir.ArrayLiteral{Elements: append([]ir.Expression(nil), call.Arguments[1:]...)}
			return g.emitApply(buf, state, &ir.CallExpression{
				Callee:    "__jayess_apply",
				Arguments: []ir.Expression{member.Target, thisArg, argsArray},
			})
		}
	}
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
		g.emitCleanupBoxedUse(buf, boxedUse{ref: directArgsBoxed, cleanup: true, shallow: true})
		g.emitCleanupBoxedUse(buf, boxedUse{ref: mergedArgs, cleanup: true, shallow: true})
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
	result, err := g.emitApplyFromValues(buf, state, boxed, boundThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
	if err != nil {
		return emittedValue{}, err
	}
	g.emitCleanupBoxedUse(buf, boxedUse{ref: directArgsBoxed, cleanup: true, shallow: true})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: mergedArgs, cleanup: true, shallow: true})
	return result, nil
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
	result, err := g.emitApplyFromValues(buf, state, boxedCallee, boxedThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
	if err != nil {
		return emittedValue{}, err
	}
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedCallee, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[0], callee)})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedThis, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[1], thisValue)})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedArgs, cleanup: argsValue.kind != ir.ValueDynamic || shouldCleanupBoxedValueAfterUse(call.Arguments[2], argsValue), shallow: true})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: mergedArgs, cleanup: true, shallow: true})
	return result, nil
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
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedCallee, cleanup: shouldCleanupBoxedValueAfterUse(call.Arguments[0], callee)})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedArgs, cleanup: boundArgs.kind != ir.ValueDynamic || shouldCleanupBoxedValueAfterUse(call.Arguments[2], boundArgs)})
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
	if err := g.emitArrayCallbackInvocationDiscardingResult(buf, state, callbackRef, item); err != nil {
		return err
	}
	nextIndex := state.nextTemp()
	fmt.Fprintf(buf, "  %s = add i32 %s, 1\n", nextIndex, index)
	fmt.Fprintf(buf, "  store i32 %s, ptr %s\n", nextIndex, indexPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", condLabel)
	fmt.Fprintf(buf, "%s:\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitArrayCallbackInvocationDiscardingResult(buf *bytes.Buffer, state *functionState, callbackRef string, itemRef string) error {
	undefinedThis := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
	boundCount := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_function_bound_arg_count(ptr %s)\n", boundCount, callbackRef)
	hasBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp sgt i32 %s, 0\n", hasBound, boundCount)
	fastLabel := state.nextLabel("array.callback.fast")
	oneBoundLabel := state.nextLabel("array.callback.onebound")
	oneBoundCallLabel := state.nextLabel("array.callback.onebound.call")
	twoBoundCallLabel := state.nextLabel("array.callback.twobound.call")
	threeBoundCallLabel := state.nextLabel("array.callback.threebound.call")
	fourBoundCallLabel := state.nextLabel("array.callback.fourbound.call")
	fiveBoundCallLabel := state.nextLabel("array.callback.fivebound.call")
	sixBoundCallLabel := state.nextLabel("array.callback.sixbound.call")
	sevenBoundCallLabel := state.nextLabel("array.callback.sevenbound.call")
	eightBoundCallLabel := state.nextLabel("array.callback.eightbound.call")
	nineBoundCallLabel := state.nextLabel("array.callback.ninebound.call")
	tenBoundCallLabel := state.nextLabel("array.callback.tenbound.call")
	elevenBoundCallLabel := state.nextLabel("array.callback.elevenbound.call")
	twelveBoundCallLabel := state.nextLabel("array.callback.twelvebound.call")
	slowLabel := state.nextLabel("array.callback.slow")
	endLabel := state.nextLabel("array.callback.end")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasBound, oneBoundLabel, fastLabel)

	fmt.Fprintf(buf, "%s:\n", fastLabel)
	fastResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_call_with_this(ptr %s, ptr %s, ptr %s, i32 1)\n", fastResult, callbackRef, undefinedThis, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: fastResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", oneBoundLabel)
	isOneBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 1\n", isOneBound, boundCount)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isOneBound, oneBoundCallLabel, twoBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", oneBoundCallLabel)
	boundArg := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArg, callbackRef)
	oneBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function2(ptr %s, ptr %s, ptr %s)\n", oneBoundResult, callbackRef, boundArg, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: oneBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", twoBoundCallLabel)
	isTwoBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 2\n", isTwoBound, boundCount)
	twoBoundFastLabel := state.nextLabel("array.callback.twobound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTwoBound, twoBoundFastLabel, threeBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", twoBoundFastLabel)
	boundArg0 := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArg0, callbackRef)
	boundArg1 := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArg1, callbackRef)
	twoBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function3(ptr %s, ptr %s, ptr %s, ptr %s)\n", twoBoundResult, callbackRef, boundArg0, boundArg1, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: twoBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", threeBoundCallLabel)
	isThreeBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 3\n", isThreeBound, boundCount)
	threeBoundFastLabel := state.nextLabel("array.callback.threebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isThreeBound, threeBoundFastLabel, fourBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", threeBoundFastLabel)
	boundArgA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgA, callbackRef)
	boundArgB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgB, callbackRef)
	boundArgC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgC, callbackRef)
	threeBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function4(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", threeBoundResult, callbackRef, boundArgA, boundArgB, boundArgC, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: threeBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", fourBoundCallLabel)
	isFourBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 4\n", isFourBound, boundCount)
	fourBoundFastLabel := state.nextLabel("array.callback.fourbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isFourBound, fourBoundFastLabel, fiveBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", fourBoundFastLabel)
	boundArgD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgD, callbackRef)
	boundArgE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgE, callbackRef)
	boundArgF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgF, callbackRef)
	boundArgG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgG, callbackRef)
	fourBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function5(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", fourBoundResult, callbackRef, boundArgD, boundArgE, boundArgF, boundArgG, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: fourBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", fiveBoundCallLabel)
	isFiveBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 5\n", isFiveBound, boundCount)
	fiveBoundFastLabel := state.nextLabel("array.callback.fivebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isFiveBound, fiveBoundFastLabel, sixBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", fiveBoundFastLabel)
	boundArgH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgH, callbackRef)
	boundArgI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgI, callbackRef)
	boundArgJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgJ, callbackRef)
	boundArgK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgK, callbackRef)
	boundArgL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgL, callbackRef)
	fiveBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function6(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", fiveBoundResult, callbackRef, boundArgH, boundArgI, boundArgJ, boundArgK, boundArgL, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: fiveBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", sixBoundCallLabel)
	isSixBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 6\n", isSixBound, boundCount)
	sixBoundFastLabel := state.nextLabel("array.callback.sixbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isSixBound, sixBoundFastLabel, sevenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", sixBoundFastLabel)
	boundArgM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgM, callbackRef)
	boundArgN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgN, callbackRef)
	boundArgO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgO, callbackRef)
	boundArgP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgP, callbackRef)
	boundArgQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgQ, callbackRef)
	boundArgR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgR, callbackRef)
	sixBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function7(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", sixBoundResult, callbackRef, boundArgM, boundArgN, boundArgO, boundArgP, boundArgQ, boundArgR, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: sixBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", sevenBoundCallLabel)
	isSevenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 7\n", isSevenBound, boundCount)
	sevenBoundFastLabel := state.nextLabel("array.callback.sevenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isSevenBound, sevenBoundFastLabel, eightBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", sevenBoundFastLabel)
	boundArgS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgS, callbackRef)
	boundArgT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgT, callbackRef)
	boundArgU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgU, callbackRef)
	boundArgV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgV, callbackRef)
	boundArgW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgW, callbackRef)
	boundArgX := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgX, callbackRef)
	boundArgY := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgY, callbackRef)
	sevenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function8(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", sevenBoundResult, callbackRef, boundArgS, boundArgT, boundArgU, boundArgV, boundArgW, boundArgX, boundArgY, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: sevenBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", eightBoundCallLabel)
	isEightBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 8\n", isEightBound, boundCount)
	eightBoundFastLabel := state.nextLabel("array.callback.eightbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isEightBound, eightBoundFastLabel, nineBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", eightBoundFastLabel)
	boundArgZ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgZ, callbackRef)
	boundArgAA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAA, callbackRef)
	boundArgAB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAB, callbackRef)
	boundArgAC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAC, callbackRef)
	boundArgAD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAD, callbackRef)
	boundArgAE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAE, callbackRef)
	boundArgAF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAF, callbackRef)
	boundArgAG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAG, callbackRef)
	eightBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function9(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", eightBoundResult, callbackRef, boundArgZ, boundArgAA, boundArgAB, boundArgAC, boundArgAD, boundArgAE, boundArgAF, boundArgAG, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: eightBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", nineBoundCallLabel)
	isNineBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 9\n", isNineBound, boundCount)
	nineBoundFastLabel := state.nextLabel("array.callback.ninebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isNineBound, nineBoundFastLabel, tenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", nineBoundFastLabel)
	boundArgAH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgAH, callbackRef)
	boundArgAI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAI, callbackRef)
	boundArgAJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAJ, callbackRef)
	boundArgAK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAK, callbackRef)
	boundArgAL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAL, callbackRef)
	boundArgAM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAM, callbackRef)
	boundArgAN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAN, callbackRef)
	boundArgAO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAO, callbackRef)
	boundArgAP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgAP, callbackRef)
	nineBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function10(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", nineBoundResult, callbackRef, boundArgAH, boundArgAI, boundArgAJ, boundArgAK, boundArgAL, boundArgAM, boundArgAN, boundArgAO, boundArgAP, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: nineBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", tenBoundCallLabel)
	isTenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 10\n", isTenBound, boundCount)
	tenBoundFastLabel := state.nextLabel("array.callback.tenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTenBound, tenBoundFastLabel, elevenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", tenBoundFastLabel)
	boundArgAQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgAQ, callbackRef)
	boundArgAR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAR, callbackRef)
	boundArgAS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAS, callbackRef)
	boundArgAT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAT, callbackRef)
	boundArgAU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAU, callbackRef)
	boundArgAV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAV, callbackRef)
	boundArgAW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAW, callbackRef)
	boundArgAX := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAX, callbackRef)
	boundArgAY := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgAY, callbackRef)
	boundArgAZ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgAZ, callbackRef)
	tenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function11(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", tenBoundResult, callbackRef, boundArgAQ, boundArgAR, boundArgAS, boundArgAT, boundArgAU, boundArgAV, boundArgAW, boundArgAX, boundArgAY, boundArgAZ, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: tenBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", elevenBoundCallLabel)
	isElevenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 11\n", isElevenBound, boundCount)
	elevenBoundFastLabel := state.nextLabel("array.callback.elevenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isElevenBound, elevenBoundFastLabel, twelveBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", elevenBoundFastLabel)
	boundArgBA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgBA, callbackRef)
	boundArgBB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgBB, callbackRef)
	boundArgBC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgBC, callbackRef)
	boundArgBD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgBD, callbackRef)
	boundArgBE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgBE, callbackRef)
	boundArgBF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgBF, callbackRef)
	boundArgBG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgBG, callbackRef)
	boundArgBH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgBH, callbackRef)
	boundArgBI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgBI, callbackRef)
	boundArgBJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgBJ, callbackRef)
	boundArgBK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 10)\n", boundArgBK, callbackRef)
	elevenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function12(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", elevenBoundResult, callbackRef, boundArgBA, boundArgBB, boundArgBC, boundArgBD, boundArgBE, boundArgBF, boundArgBG, boundArgBH, boundArgBI, boundArgBJ, boundArgBK, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: elevenBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", twelveBoundCallLabel)
	isTwelveBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 12\n", isTwelveBound, boundCount)
	twelveBoundFastLabel := state.nextLabel("array.callback.twelvebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTwelveBound, twelveBoundFastLabel, slowLabel)

	fmt.Fprintf(buf, "%s:\n", twelveBoundFastLabel)
	boundArgBL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgBL, callbackRef)
	boundArgBM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgBM, callbackRef)
	boundArgBN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgBN, callbackRef)
	boundArgBO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgBO, callbackRef)
	boundArgBP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgBP, callbackRef)
	boundArgBQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgBQ, callbackRef)
	boundArgBR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgBR, callbackRef)
	boundArgBS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgBS, callbackRef)
	boundArgBT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgBT, callbackRef)
	boundArgBU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgBU, callbackRef)
	boundArgBV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 10)\n", boundArgBV, callbackRef)
	boundArgBW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 11)\n", boundArgBW, callbackRef)
	twelveBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function13(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", twelveBoundResult, callbackRef, boundArgBL, boundArgBM, boundArgBN, boundArgBO, boundArgBP, boundArgBQ, boundArgBR, boundArgBS, boundArgBT, boundArgBU, boundArgBV, boundArgBW, itemRef)
	g.emitCleanupBoxedUse(buf, boxedUse{ref: twelveBoundResult, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", slowLabel)
	argsArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
	fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, itemRef)
	boxedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
	mergedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_merge_bound_args(ptr %s, ptr %s)\n", mergedArgs, callbackRef, boxedArgs)
	result, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
	if err != nil {
		return err
	}
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedArgs, cleanup: true, shallow: true})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: mergedArgs, cleanup: true, shallow: true})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: result.ref, cleanup: true})
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", endLabel)
	return nil
}

func (g *LLVMIRGenerator) emitArrayMapCall(buf *bytes.Buffer, state *functionState, itemsRef string, callbackRef string) (emittedValue, error) {
	resultArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", resultArray)
	if err := g.emitArrayCallbackLoop(buf, state, itemsRef, callbackRef, func(item string) error {
		result, err := g.emitArrayCallbackInvocation(buf, state, callbackRef, item)
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
		result, err := g.emitArrayCallbackInvocation(buf, state, callbackRef, item)
		if err != nil {
			return err
		}
		cond, err := g.emitTruthyFromValue(buf, state, result)
		if err != nil {
			return err
		}
		g.emitCleanupBoxedUse(buf, boxedUse{ref: result.ref, cleanup: result.kind == ir.ValueDynamic})
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
	result, err := g.emitArrayCallbackInvocation(buf, state, callbackRef, item)
	if err != nil {
		return emittedValue{}, err
	}
	cond, err := g.emitTruthyFromValue(buf, state, result)
	if err != nil {
		return emittedValue{}, err
	}
	g.emitCleanupBoxedUse(buf, boxedUse{ref: result.ref, cleanup: result.kind == ir.ValueDynamic})
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

func (g *LLVMIRGenerator) emitArrayCallbackInvocation(buf *bytes.Buffer, state *functionState, callbackRef string, itemRef string) (emittedValue, error) {
	resultPtr := state.nextTemp()
	fmt.Fprintf(buf, "  %s = alloca ptr\n", resultPtr)
	undefinedThis := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undefinedThis)
	boundCount := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i32 @jayess_value_function_bound_arg_count(ptr %s)\n", boundCount, callbackRef)
	hasBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp sgt i32 %s, 0\n", hasBound, boundCount)
	fastLabel := state.nextLabel("array.callback.fast")
	oneBoundLabel := state.nextLabel("array.callback.onebound")
	oneBoundCallLabel := state.nextLabel("array.callback.onebound.call")
	twoBoundCallLabel := state.nextLabel("array.callback.twobound.call")
	threeBoundCallLabel := state.nextLabel("array.callback.threebound.call")
	fourBoundCallLabel := state.nextLabel("array.callback.fourbound.call")
	fiveBoundCallLabel := state.nextLabel("array.callback.fivebound.call")
	sixBoundCallLabel := state.nextLabel("array.callback.sixbound.call")
	sevenBoundCallLabel := state.nextLabel("array.callback.sevenbound.call")
	eightBoundCallLabel := state.nextLabel("array.callback.eightbound.call")
	nineBoundCallLabel := state.nextLabel("array.callback.ninebound.call")
	tenBoundCallLabel := state.nextLabel("array.callback.tenbound.call")
	elevenBoundCallLabel := state.nextLabel("array.callback.elevenbound.call")
	twelveBoundCallLabel := state.nextLabel("array.callback.twelvebound.call")
	slowLabel := state.nextLabel("array.callback.slow")
	endLabel := state.nextLabel("array.callback.end")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasBound, oneBoundLabel, fastLabel)

	fmt.Fprintf(buf, "%s:\n", fastLabel)
	fastResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_call_with_this(ptr %s, ptr %s, ptr %s, i32 1)\n", fastResult, callbackRef, undefinedThis, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", fastResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", oneBoundLabel)
	isOneBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 1\n", isOneBound, boundCount)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isOneBound, oneBoundCallLabel, twoBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", oneBoundCallLabel)
	boundArg := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArg, callbackRef)
	oneBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function2(ptr %s, ptr %s, ptr %s)\n", oneBoundResult, callbackRef, boundArg, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", oneBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", twoBoundCallLabel)
	isTwoBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 2\n", isTwoBound, boundCount)
	twoBoundFastLabel := state.nextLabel("array.callback.twobound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTwoBound, twoBoundFastLabel, threeBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", twoBoundFastLabel)
	boundArg0 := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArg0, callbackRef)
	boundArg1 := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArg1, callbackRef)
	twoBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function3(ptr %s, ptr %s, ptr %s, ptr %s)\n", twoBoundResult, callbackRef, boundArg0, boundArg1, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", twoBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", threeBoundCallLabel)
	isThreeBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 3\n", isThreeBound, boundCount)
	threeBoundFastLabel := state.nextLabel("array.callback.threebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isThreeBound, threeBoundFastLabel, fourBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", threeBoundFastLabel)
	boundArgA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgA, callbackRef)
	boundArgB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgB, callbackRef)
	boundArgC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgC, callbackRef)
	threeBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function4(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", threeBoundResult, callbackRef, boundArgA, boundArgB, boundArgC, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", threeBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", fourBoundCallLabel)
	isFourBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 4\n", isFourBound, boundCount)
	fourBoundFastLabel := state.nextLabel("array.callback.fourbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isFourBound, fourBoundFastLabel, fiveBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", fourBoundFastLabel)
	boundArgD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgD, callbackRef)
	boundArgE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgE, callbackRef)
	boundArgF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgF, callbackRef)
	boundArgG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgG, callbackRef)
	fourBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function5(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", fourBoundResult, callbackRef, boundArgD, boundArgE, boundArgF, boundArgG, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", fourBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", fiveBoundCallLabel)
	isFiveBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 5\n", isFiveBound, boundCount)
	fiveBoundFastLabel := state.nextLabel("array.callback.fivebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isFiveBound, fiveBoundFastLabel, sixBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", fiveBoundFastLabel)
	boundArgH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgH, callbackRef)
	boundArgI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgI, callbackRef)
	boundArgJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgJ, callbackRef)
	boundArgK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgK, callbackRef)
	boundArgL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgL, callbackRef)
	fiveBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function6(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", fiveBoundResult, callbackRef, boundArgH, boundArgI, boundArgJ, boundArgK, boundArgL, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", fiveBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", sixBoundCallLabel)
	isSixBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 6\n", isSixBound, boundCount)
	sixBoundFastLabel := state.nextLabel("array.callback.sixbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isSixBound, sixBoundFastLabel, sevenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", sixBoundFastLabel)
	boundArgM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgM, callbackRef)
	boundArgN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgN, callbackRef)
	boundArgO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgO, callbackRef)
	boundArgP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgP, callbackRef)
	boundArgQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgQ, callbackRef)
	boundArgR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgR, callbackRef)
	sixBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function7(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", sixBoundResult, callbackRef, boundArgM, boundArgN, boundArgO, boundArgP, boundArgQ, boundArgR, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", sixBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", sevenBoundCallLabel)
	isSevenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 7\n", isSevenBound, boundCount)
	sevenBoundFastLabel := state.nextLabel("array.callback.sevenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isSevenBound, sevenBoundFastLabel, eightBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", sevenBoundFastLabel)
	boundArgS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgS, callbackRef)
	boundArgT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgT, callbackRef)
	boundArgU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgU, callbackRef)
	boundArgV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgV, callbackRef)
	boundArgW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgW, callbackRef)
	boundArgX := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgX, callbackRef)
	boundArgY := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgY, callbackRef)
	sevenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function8(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", sevenBoundResult, callbackRef, boundArgS, boundArgT, boundArgU, boundArgV, boundArgW, boundArgX, boundArgY, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", sevenBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", eightBoundCallLabel)
	isEightBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 8\n", isEightBound, boundCount)
	eightBoundFastLabel := state.nextLabel("array.callback.eightbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isEightBound, eightBoundFastLabel, nineBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", eightBoundFastLabel)
	boundArgZ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgZ, callbackRef)
	boundArgAA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAA, callbackRef)
	boundArgAB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAB, callbackRef)
	boundArgAC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAC, callbackRef)
	boundArgAD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAD, callbackRef)
	boundArgAE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAE, callbackRef)
	boundArgAF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAF, callbackRef)
	boundArgAG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAG, callbackRef)
	eightBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function9(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", eightBoundResult, callbackRef, boundArgZ, boundArgAA, boundArgAB, boundArgAC, boundArgAD, boundArgAE, boundArgAF, boundArgAG, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", eightBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", nineBoundCallLabel)
	isNineBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 9\n", isNineBound, boundCount)
	nineBoundFastLabel := state.nextLabel("array.callback.ninebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isNineBound, nineBoundFastLabel, tenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", nineBoundFastLabel)
	boundArgAH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgAH, callbackRef)
	boundArgAI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAI, callbackRef)
	boundArgAJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAJ, callbackRef)
	boundArgAK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAK, callbackRef)
	boundArgAL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAL, callbackRef)
	boundArgAM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAM, callbackRef)
	boundArgAN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAN, callbackRef)
	boundArgAO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAO, callbackRef)
	boundArgAP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgAP, callbackRef)
	nineBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function10(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", nineBoundResult, callbackRef, boundArgAH, boundArgAI, boundArgAJ, boundArgAK, boundArgAL, boundArgAM, boundArgAN, boundArgAO, boundArgAP, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", nineBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", tenBoundCallLabel)
	isTenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 10\n", isTenBound, boundCount)
	tenBoundFastLabel := state.nextLabel("array.callback.tenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTenBound, tenBoundFastLabel, elevenBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", tenBoundFastLabel)
	boundArgAQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgAQ, callbackRef)
	boundArgAR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgAR, callbackRef)
	boundArgAS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgAS, callbackRef)
	boundArgAT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgAT, callbackRef)
	boundArgAU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgAU, callbackRef)
	boundArgAV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgAV, callbackRef)
	boundArgAW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgAW, callbackRef)
	boundArgAX := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgAX, callbackRef)
	boundArgAY := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgAY, callbackRef)
	boundArgAZ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgAZ, callbackRef)
	tenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function11(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", tenBoundResult, callbackRef, boundArgAQ, boundArgAR, boundArgAS, boundArgAT, boundArgAU, boundArgAV, boundArgAW, boundArgAX, boundArgAY, boundArgAZ, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", tenBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", elevenBoundCallLabel)
	isElevenBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 11\n", isElevenBound, boundCount)
	elevenBoundFastLabel := state.nextLabel("array.callback.elevenbound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isElevenBound, elevenBoundFastLabel, twelveBoundCallLabel)

	fmt.Fprintf(buf, "%s:\n", elevenBoundFastLabel)
	boundArgBA := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgBA, callbackRef)
	boundArgBB := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgBB, callbackRef)
	boundArgBC := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgBC, callbackRef)
	boundArgBD := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgBD, callbackRef)
	boundArgBE := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgBE, callbackRef)
	boundArgBF := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgBF, callbackRef)
	boundArgBG := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgBG, callbackRef)
	boundArgBH := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgBH, callbackRef)
	boundArgBI := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgBI, callbackRef)
	boundArgBJ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgBJ, callbackRef)
	boundArgBK := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 10)\n", boundArgBK, callbackRef)
	elevenBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function12(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", elevenBoundResult, callbackRef, boundArgBA, boundArgBB, boundArgBC, boundArgBD, boundArgBE, boundArgBF, boundArgBG, boundArgBH, boundArgBI, boundArgBJ, boundArgBK, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", elevenBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", twelveBoundCallLabel)
	isTwelveBound := state.nextTemp()
	fmt.Fprintf(buf, "  %s = icmp eq i32 %s, 12\n", isTwelveBound, boundCount)
	twelveBoundFastLabel := state.nextLabel("array.callback.twelvebound.fast")
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", isTwelveBound, twelveBoundFastLabel, slowLabel)

	fmt.Fprintf(buf, "%s:\n", twelveBoundFastLabel)
	boundArgBL := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 0)\n", boundArgBL, callbackRef)
	boundArgBM := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 1)\n", boundArgBM, callbackRef)
	boundArgBN := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 2)\n", boundArgBN, callbackRef)
	boundArgBO := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 3)\n", boundArgBO, callbackRef)
	boundArgBP := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 4)\n", boundArgBP, callbackRef)
	boundArgBQ := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 5)\n", boundArgBQ, callbackRef)
	boundArgBR := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 6)\n", boundArgBR, callbackRef)
	boundArgBS := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 7)\n", boundArgBS, callbackRef)
	boundArgBT := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 8)\n", boundArgBT, callbackRef)
	boundArgBU := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 9)\n", boundArgBU, callbackRef)
	boundArgBV := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 10)\n", boundArgBV, callbackRef)
	boundArgBW := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_function_bound_arg(ptr %s, i32 11)\n", boundArgBW, callbackRef)
	twelveBoundResult := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_call_function13(ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s, ptr %s)\n", twelveBoundResult, callbackRef, boundArgBL, boundArgBM, boundArgBN, boundArgBO, boundArgBP, boundArgBQ, boundArgBR, boundArgBS, boundArgBT, boundArgBU, boundArgBV, boundArgBW, itemRef)
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", twelveBoundResult, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", slowLabel)
	argsArray := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_array_new()\n", argsArray)
	fmt.Fprintf(buf, "  call void @jayess_array_set_value(ptr %s, i32 0, ptr %s)\n", argsArray, itemRef)
	boxedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_array(ptr %s)\n", boxedArgs, argsArray)
	mergedArgs := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_merge_bound_args(ptr %s, ptr %s)\n", mergedArgs, callbackRef, boxedArgs)
	result, err := g.emitApplyFromValues(buf, state, callbackRef, undefinedThis, emittedValue{kind: ir.ValueDynamic, ref: mergedArgs})
	if err != nil {
		return emittedValue{}, err
	}
	g.emitCleanupBoxedUse(buf, boxedUse{ref: boxedArgs, cleanup: true, shallow: true})
	g.emitCleanupBoxedUse(buf, boxedUse{ref: mergedArgs, cleanup: true, shallow: true})
	fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", result.ref, resultPtr)
	fmt.Fprintf(buf, "  br label %%%s\n", endLabel)

	fmt.Fprintf(buf, "%s:\n", endLabel)
	final := state.nextTemp()
	fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", final, resultPtr)
	return emittedValue{kind: ir.ValueDynamic, ref: final}, nil
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

	caseLabels := make([]string, indirectApplyMaxArgs+1)
	checkLabels := make([]string, indirectApplyMaxArgs+1)
	for i := 0; i <= indirectApplyMaxArgs; i++ {
		caseLabels[i] = state.nextLabel(fmt.Sprintf("apply.%d", i))
		checkLabels[i] = state.nextLabel(fmt.Sprintf("apply.check.%d", i))
	}

	fmt.Fprintf(buf, "  br label %%%s\n", checkLabels[0])
	for i := 0; i <= indirectApplyMaxArgs; i++ {
		next := defaultLabel
		if i < indirectApplyMaxArgs {
			next = checkLabels[i+1]
		}
		fmt.Fprintf(buf, "%s:\n", checkLabels[i])
		match := state.nextTemp()
		fmt.Fprintf(buf, "  %s = icmp eq i32 %s, %d\n", match, lenRef, i)
		fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", match, caseLabels[i], next)
	}

	for i := 0; i <= indirectApplyMaxArgs; i++ {
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
	maxParams := indirectApplyMaxArgs
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

func (g *LLVMIRGenerator) emitDirectJayessCallWithThis(buf *bytes.Buffer, state *functionState, callee string, fn ir.Function, thisRef string, args []string, cleanupUses []boxedUse) (emittedValue, error) {
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
	calleeName := emittedFunctionName(callee)
	if len(callArgs) == 0 {
		fmt.Fprintf(buf, "  %s = call ptr @%s()\n", result, calleeName)
	} else {
		fmt.Fprintf(buf, "  %s = call ptr @%s(%s)\n", result, calleeName, strings.Join(callArgs, ", "))
	}
	for _, use := range cleanupUses {
		g.emitCleanupBoxedUse(buf, use)
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
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_function(ptr @%s, ptr null, ptr %s, ptr %s, i32 %d, i1 %s)\n", tmp, emittedFunctionName(name), state.stringRefs[name], classRef, paramCount, hasRest)
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
			boxed := state.nextTemp()
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxed, target.ref)
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, boxed, index.ref)
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: true})
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
	raw := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call ptr @jayess_args_get(ptr %s, i32 %s)\n", raw, target.ref, indexInt)
	fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_string(ptr %s)\n", tmp, raw)
	return emittedValue{kind: ir.ValueDynamic, ref: tmp}, nil
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
			g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: true})
			num := state.nextTemp()
			fmt.Fprintf(buf, "  %s = sitofp i32 %s to double\n", num, tmp)
			return emittedValue{kind: ir.ValueNumber, ref: num}, nil
		}
	}
	tmp := state.nextTemp()
	if target.kind == ir.ValueObject {
		boxed := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_object(ptr %s)\n", boxed, target.ref)
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, boxed, state.stringRefs[property])
		g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: true})
	} else if target.kind == ir.ValueDynamic {
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, target.ref, state.stringRefs[property])
	} else {
		boxed, err := g.emitBoxedValue(buf, state, target)
		if err != nil {
			return emittedValue{}, err
		}
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_get_member(ptr %s, ptr %s)\n", tmp, boxed, state.stringRefs[property])
		g.emitCleanupBoxedUse(buf, boxedUse{ref: boxed, cleanup: target.kind != ir.ValueFunction})
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

func accessorStorageKeyForCodegen(getter bool, name string) string {
	if getter {
		return "__jayess_get_" + name
	}
	return "__jayess_set_" + name
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
	fmt.Fprintf(buf, "  call void @jayess_throw_not_function()\n")
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
	raisedLabel := state.nextLabel("throw.raise")
	continueLabel := state.nextLabel("throw.cont")
	hasException := state.nextTemp()
	fmt.Fprintf(buf, "  %s = call i1 @jayess_has_exception()\n", hasException)
	fmt.Fprintf(buf, "  br i1 %s, label %%%s, label %%%s\n", hasException, raisedLabel, continueLabel)
	fmt.Fprintf(buf, "%s:\n", raisedLabel)
	g.emitCleanupScopesToDepth(buf, state, state.exceptionCleanupDepth)
	fmt.Fprintf(buf, "  br label %%%s\n", target)
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
	case ir.ValueBigInt:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call i1 @jayess_value_is_truthy(ptr %s)\n", tmp, value.ref)
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
		if value.staticString {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_static_string(ptr %s)\n", tmp, value.ref)
		} else {
			fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_string(ptr %s)\n", tmp, value.ref)
		}
		return tmp, nil
	case ir.ValueNumber:
		tmp := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_from_number(double %s)\n", tmp, value.ref)
		return tmp, nil
	case ir.ValueBigInt:
		return value.ref, nil
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

func localEligibilityKey(name string, line, column int) string {
	return fmt.Sprintf("%s:%d:%d", name, line, column)
}

func (s *functionState) isEligibleLocal(name string, line, column int) bool {
	if s.eligibleLocals == nil {
		return false
	}
	if strings.HasPrefix(name, "__jayess_") {
		return false
	}
	_, ok := s.eligibleLocals[localEligibilityKey(name, line, column)]
	return ok
}

func (s *functionState) isFunctionScopedVarCleanupEligible(name string, line, column int) bool {
	if s.eligibleLocals == nil {
		return false
	}
	item, ok := s.eligibleLocals[localEligibilityKey(name, line, column)]
	if !ok {
		return false
	}
	return item.Kind == ir.DeclarationVar
}

func collectHoistedVarNames(statements []ir.Statement, names map[string]bool) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ir.VariableDecl:
			if stmt.Kind == ir.DeclarationVar && stmt.Name != "" {
				names[stmt.Name] = true
			}
		case *ir.IfStatement:
			collectHoistedVarNames(stmt.Consequence, names)
			collectHoistedVarNames(stmt.Alternative, names)
		case *ir.WhileStatement:
			collectHoistedVarNames(stmt.Body, names)
		case *ir.DoWhileStatement:
			collectHoistedVarNames(stmt.Body, names)
		case *ir.BlockStatement:
			collectHoistedVarNames(stmt.Body, names)
		case *ir.ForStatement:
			if stmt.Init != nil {
				collectHoistedVarNames([]ir.Statement{stmt.Init}, names)
			}
			if stmt.Update != nil {
				collectHoistedVarNames([]ir.Statement{stmt.Update}, names)
			}
			collectHoistedVarNames(stmt.Body, names)
		case *ir.SwitchStatement:
			for _, switchCase := range stmt.Cases {
				collectHoistedVarNames(switchCase.Consequent, names)
			}
			collectHoistedVarNames(stmt.Default, names)
		case *ir.TryStatement:
			collectHoistedVarNames(stmt.TryBody, names)
			collectHoistedVarNames(stmt.CatchBody, names)
			collectHoistedVarNames(stmt.FinallyBody, names)
		case *ir.LabeledStatement:
			if stmt.Statement != nil {
				collectHoistedVarNames([]ir.Statement{stmt.Statement}, names)
			}
		}
	}
}

func (g *LLVMIRGenerator) emitHoistedVarSlots(buf *bytes.Buffer, state *functionState, fn ir.Function) {
	names := map[string]bool{}
	collectHoistedVarNames(fn.Body, names)
	for _, param := range fn.Params {
		delete(names, param.Name)
	}
	for name := range names {
		slot := state.nextTemp()
		fmt.Fprintf(buf, "  %s = alloca ptr\n", slot)
		undef := state.nextTemp()
		fmt.Fprintf(buf, "  %s = call ptr @jayess_value_undefined()\n", undef)
		fmt.Fprintf(buf, "  store ptr %s, ptr %s\n", undef, slot)
		state.hoistedVarSlots[name] = variableSlot{kind: ir.ValueDynamic, ptr: slot}
		state.slots[name] = variableSlot{kind: ir.ValueDynamic, ptr: slot}
		if state.hasEligibleVarByName(name) {
			state.addRootCleanup(variableSlot{kind: ir.ValueDynamic, ptr: slot})
		}
	}
}

func (s *functionState) hasEligibleVarByName(name string) bool {
	if s.eligibleLocals == nil {
		return false
	}
	for _, item := range s.eligibleLocals {
		if item.Name == name && item.Kind == ir.DeclarationVar {
			return true
		}
	}
	return false
}

func (s *functionState) pushScope() {
	s.scopeStack = append(s.scopeStack, cleanupScope{})
}

func (s *functionState) popScope() {
	if len(s.scopeStack) == 0 {
		return
	}
	scope := s.scopeStack[len(s.scopeStack)-1]
	s.scopeStack = s.scopeStack[:len(s.scopeStack)-1]
	for i := len(scope.shadowed) - 1; i >= 0; i-- {
		item := scope.shadowed[i]
		if item.ok {
			s.slots[item.name] = item.slot
		} else {
			delete(s.slots, item.name)
		}
	}
}

func (s *functionState) declareSlot(name string, slot variableSlot) {
	if len(s.scopeStack) > 0 {
		scope := &s.scopeStack[len(s.scopeStack)-1]
		alreadyTracked := false
		for _, item := range scope.shadowed {
			if item.name == name {
				alreadyTracked = true
				break
			}
		}
		if !alreadyTracked {
			previous, ok := s.slots[name]
			scope.shadowed = append(scope.shadowed, shadowedSlot{name: name, slot: previous, ok: ok})
		}
	}
	s.slots[name] = slot
}

func (s *functionState) addCleanup(slot variableSlot) {
	if len(s.scopeStack) == 0 {
		return
	}
	scope := &s.scopeStack[len(s.scopeStack)-1]
	scope.cleanups = append(scope.cleanups, slot)
}

func (s *functionState) addRootCleanup(slot variableSlot) {
	if len(s.scopeStack) == 0 {
		return
	}
	scope := &s.scopeStack[0]
	scope.cleanups = append(scope.cleanups, slot)
}

func isTransientDynamicExpression(expr ir.Expression) bool {
	switch expr.(type) {
	case *ir.CallExpression, *ir.InvokeExpression, *ir.BinaryExpression, *ir.TemplateLiteral, *ir.NullishCoalesceExpression, *ir.LogicalExpression:
		return true
	default:
		return false
	}
}

func shouldCleanupBoxedValueAfterUse(expr ir.Expression, value emittedValue) bool {
	if value.kind != ir.ValueDynamic {
		return true
	}
	return isTransientDynamicExpression(expr)
}

func shouldCleanupTransientParserArgAfterUse(callee string, expr ir.Expression, value emittedValue) bool {
	switch callee {
	case "parseHtml", "parseHtmlFragment", "tokenizeHtml", "parseXml", "tokenizeXml", "parseCss", "tokenizeCss":
		return shouldCleanupBoxedValueAfterUse(expr, value)
	default:
		return false
	}
}

func isBorrowedAliasExpression(expr ir.Expression) bool {
	switch expr.(type) {
	case *ir.VariableRef, *ir.MemberExpression, *ir.IndexExpression:
		return true
	default:
		return false
	}
}

func shouldScheduleDynamicLocalCleanup(expr ir.Expression, value emittedValue) bool {
	if value.kind != ir.ValueDynamic {
		return false
	}
	return !isBorrowedAliasExpression(expr)
}

func isDirectFunctionReceiver(expr ir.Expression) bool {
	switch expr := expr.(type) {
	case *ir.VariableRef:
		return expr.Kind == ir.ValueFunction
	case *ir.FunctionValue:
		return true
	default:
		return false
	}
}

func (s *functionState) functionExpressionReturnsFresh(expr ir.Expression) bool {
	switch expr := expr.(type) {
	case *ir.VariableRef:
		if expr.Kind != ir.ValueFunction {
			return false
		}
		fn, ok := s.functions[expr.Name]
		return ok && fn.ReturnFresh
	case *ir.FunctionValue:
		fn, ok := s.functions[expr.Name]
		return ok && fn.ReturnFresh
	case *ir.CallExpression:
		if expr.Callee == "__jayess_bind" && len(expr.Arguments) > 0 {
			return s.functionExpressionReturnsFresh(expr.Arguments[0])
		}
		return false
	default:
		return false
	}
}

func (s *functionState) shouldCleanupDiscardedFreshCall(call *ir.CallExpression, value emittedValue) bool {
	if value.ref == "" {
		return false
	}
	switch call.Callee {
	case "__jayess_bind":
		return value.kind == ir.ValueFunction
	case "__jayess_apply":
		return value.kind == ir.ValueDynamic && len(call.Arguments) > 0 && s.functionExpressionReturnsFresh(call.Arguments[0])
	case "__jayess_array_map", "__jayess_array_filter":
		return value.kind == ir.ValueArray
	}
	if value.kind != ir.ValueDynamic {
		return false
	}
	fn, ok := s.functions[call.Callee]
	if !ok {
		return false
	}
	return fn.ReturnFresh
}

func (s *functionState) shouldCleanupDiscardedExpression(expr ir.Expression, value emittedValue) bool {
	if value.ref == "" {
		return false
	}
	switch expr := expr.(type) {
	case *ir.CallExpression:
		return s.shouldCleanupDiscardedFreshCall(expr, value)
	case *ir.TemplateLiteral, *ir.ObjectLiteral, *ir.ArrayLiteral, *ir.FunctionValue, *ir.BigIntLiteral, *ir.NullLiteral, *ir.UndefinedLiteral, *ir.NewTargetExpression:
		return true
	case *ir.BinaryExpression:
		switch expr.Operator {
		case ir.OperatorAdd, ir.OperatorBitAnd, ir.OperatorBitOr, ir.OperatorBitXor, ir.OperatorShl, ir.OperatorShr, ir.OperatorUShr:
			return value.kind == ir.ValueDynamic || value.kind == ir.ValueBigInt
		default:
			return false
		}
	case *ir.ConditionalExpression:
		return isDiscardableFreshExpression(expr.Consequent) && isDiscardableFreshExpression(expr.Alternative)
	case *ir.NullishCoalesceExpression:
		return isDiscardableFreshExpression(expr.Left) && isDiscardableFreshExpression(expr.Right)
	case *ir.CommaExpression:
		return isDiscardableFreshExpression(expr.Right)
	default:
		return false
	}
}

func isDiscardableFreshExpression(expr ir.Expression) bool {
	switch expr := expr.(type) {
	case *ir.CallExpression:
		return false
	case *ir.TemplateLiteral, *ir.ObjectLiteral, *ir.ArrayLiteral, *ir.FunctionValue, *ir.BigIntLiteral, *ir.NullLiteral, *ir.UndefinedLiteral, *ir.NewTargetExpression:
		return true
	case *ir.BinaryExpression:
		switch expr.Operator {
		case ir.OperatorAdd, ir.OperatorBitAnd, ir.OperatorBitOr, ir.OperatorBitXor, ir.OperatorShl, ir.OperatorShr, ir.OperatorUShr:
			return true
		default:
			return false
		}
	case *ir.ConditionalExpression:
		return isDiscardableFreshExpression(expr.Consequent) && isDiscardableFreshExpression(expr.Alternative)
	case *ir.NullishCoalesceExpression:
		return isDiscardableFreshExpression(expr.Left) && isDiscardableFreshExpression(expr.Right)
	case *ir.CommaExpression:
		return isDiscardableFreshExpression(expr.Right)
	default:
		return false
	}
}

func (g *LLVMIRGenerator) emitCleanupBoxedUse(buf *bytes.Buffer, use boxedUse) {
	if !use.cleanup || use.ref == "" {
		return
	}
	if use.shallow {
		fmt.Fprintf(buf, "  call void @jayess_value_free_array_shallow(ptr %s)\n", use.ref)
		return
	}
	fmt.Fprintf(buf, "  call void @jayess_value_free_unshared(ptr %s)\n", use.ref)
}

func (s *functionState) nextLabel(prefix string) string {
	name := fmt.Sprintf("%s.%d", prefix, s.labelCounter)
	s.labelCounter++
	return name
}

func (g *LLVMIRGenerator) emitCleanupSlots(buf *bytes.Buffer, state *functionState, scope cleanupScope) {
	for i := len(scope.cleanups) - 1; i >= 0; i-- {
		slot := scope.cleanups[i]
		switch slot.kind {
		case ir.ValueDynamic:
			if !state.slotOwnsCleanup(slot) {
				continue
			}
			loaded := state.nextTemp()
			fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", loaded, slot.ptr)
			fmt.Fprintf(buf, "  call void @jayess_value_free_unshared(ptr %s)\n", loaded)
		}
	}
}

func (g *LLVMIRGenerator) emitCurrentScopeCleanup(buf *bytes.Buffer, state *functionState) {
	if len(state.scopeStack) == 0 {
		return
	}
	g.emitCleanupSlots(buf, state, state.scopeStack[len(state.scopeStack)-1])
}

func (g *LLVMIRGenerator) emitCleanupScopesToDepth(buf *bytes.Buffer, state *functionState, depth int) {
	if depth < 0 {
		depth = 0
	}
	for i := len(state.scopeStack) - 1; i >= depth; i-- {
		g.emitCleanupSlots(buf, state, state.scopeStack[i])
	}
}

func (g *LLVMIRGenerator) emitCleanupAllScopes(buf *bytes.Buffer, state *functionState) {
	for i := len(state.scopeStack) - 1; i >= 0; i-- {
		g.emitCleanupSlots(buf, state, state.scopeStack[i])
	}
}

func (s *functionState) slotOwnsCleanup(slot variableSlot) bool {
	for _, current := range s.slots {
		if current.ptr == slot.ptr && current.kind == slot.kind {
			return current.ownsCleanup
		}
	}
	for _, current := range s.hoistedVarSlots {
		if current.ptr == slot.ptr && current.kind == slot.kind {
			return current.ownsCleanup
		}
	}
	return slot.ownsCleanup
}

func (g *LLVMIRGenerator) emitStoreIntoVariableSlot(buf *bytes.Buffer, state *functionState, slot variableSlot, value emittedValue, expr ir.Expression) (variableSlot, error) {
	if slot.kind == ir.ValueDynamic && slot.ownsCleanup {
		previous := state.nextTemp()
		fmt.Fprintf(buf, "  %s = load ptr, ptr %s\n", previous, slot.ptr)
		fmt.Fprintf(buf, "  call void @jayess_value_free_unshared(ptr %s)\n", previous)
	}
	storeRef := value.ref
	if slot.kind == ir.ValueDynamic {
		boxed, err := g.emitBoxedValue(buf, state, value)
		if err != nil {
			return slot, err
		}
		storeRef = boxed
		slot.ownsCleanup = shouldScheduleDynamicLocalCleanup(expr, value)
	}
	fmt.Fprintf(buf, "  store %s %s, ptr %s\n", llvmStorageType(slot.kind), storeRef, slot.ptr)
	return slot, nil
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
		addString(fn.Name, seen, &out)
		addString(stackFrameLabel(fn), seen, &out)
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
	case *ir.ThrowStatement:
		collectStringsFromExpression(stmt.Value, seen, out)
	case *ir.IfStatement:
		collectStringsFromExpression(stmt.Condition, seen, out)
		for _, child := range stmt.Consequence {
			collectStringsFromStatement(child, seen, out)
		}
		for _, child := range stmt.Alternative {
			collectStringsFromStatement(child, seen, out)
		}
	case *ir.BlockStatement:
		for _, child := range stmt.Body {
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
	case *ir.SwitchStatement:
		collectStringsFromExpression(stmt.Discriminant, seen, out)
		for _, switchCase := range stmt.Cases {
			collectStringsFromExpression(switchCase.Test, seen, out)
			for _, child := range switchCase.Consequent {
				collectStringsFromStatement(child, seen, out)
			}
		}
		for _, child := range stmt.Default {
			collectStringsFromStatement(child, seen, out)
		}
	case *ir.LabeledStatement:
		collectStringsFromStatement(stmt.Statement, seen, out)
	case *ir.TryStatement:
		for _, child := range stmt.TryBody {
			collectStringsFromStatement(child, seen, out)
		}
		for _, child := range stmt.CatchBody {
			collectStringsFromStatement(child, seen, out)
		}
		for _, child := range stmt.FinallyBody {
			collectStringsFromStatement(child, seen, out)
		}
	}
}

func collectStringsFromExpression(expr ir.Expression, seen map[string]bool, out *[]string) {
	switch expr := expr.(type) {
	case *ir.StringLiteral:
		addString(expr.Value, seen, out)
	case *ir.BigIntLiteral:
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
				key := property.Key
				if property.Getter {
					key = accessorStorageKeyForCodegen(true, property.Key)
				} else if property.Setter {
					key = accessorStorageKeyForCodegen(false, property.Key)
				}
				addString(key, seen, out)
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
		addString("bigint", seen, out)
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
	case *ir.CommaExpression:
		collectStringsFromExpression(expr.Left, seen, out)
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.ConditionalExpression:
		collectStringsFromExpression(expr.Condition, seen, out)
		collectStringsFromExpression(expr.Consequent, seen, out)
		collectStringsFromExpression(expr.Alternative, seen, out)
	case *ir.UnaryExpression:
		collectStringsFromExpression(expr.Right, seen, out)
	case *ir.IndexExpression:
		collectStringsFromExpression(expr.Target, seen, out)
		collectStringsFromExpression(expr.Index, seen, out)
	case *ir.CallExpression:
		if expr.Callee == "__jayess_type_is" && len(expr.Arguments) == 2 {
			if annotation, ok := expr.Arguments[1].(*ir.StringLiteral); ok {
				if typeExpr, err := typesys.Parse(annotation.Value); err == nil {
					collectStringsFromTypeExpr(typeExpr, seen, out)
				}
			}
		}
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

func collectStringsFromTypeExpr(expr *typesys.Expr, seen map[string]bool, out *[]string) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case typesys.KindSimple:
		switch expr.Name {
		case "", "any", "dynamic", "unknown", "never", "number", "bigint", "boolean", "string", "function", "object", "array", "null", "undefined", "void":
		default:
			addString(expr.Name, seen, out)
		}
	case typesys.KindLiteral:
		if strings.HasPrefix(expr.Name, "\"") {
			if text, err := strconv.Unquote(expr.Name); err == nil {
				addString(text, seen, out)
			}
		}
	case typesys.KindUnion, typesys.KindIntersection, typesys.KindTuple:
		for _, element := range expr.Elements {
			collectStringsFromTypeExpr(element, seen, out)
		}
	case typesys.KindObject:
		for _, property := range expr.Properties {
			addString(property.Name, seen, out)
			collectStringsFromTypeExpr(property.Type, seen, out)
		}
		for _, signature := range expr.IndexSignatures {
			collectStringsFromTypeExpr(signature.KeyType, seen, out)
			collectStringsFromTypeExpr(signature.ValueType, seen, out)
		}
	case typesys.KindFunction:
		for _, param := range expr.Params {
			collectStringsFromTypeExpr(param, seen, out)
		}
		collectStringsFromTypeExpr(expr.Return, seen, out)
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
