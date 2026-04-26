#ifndef JAYESS_RUNTIME_H
#define JAYESS_RUNTIME_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct jayess_args jayess_args;
typedef struct jayess_value jayess_value;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;
typedef void (*jayess_native_handle_finalizer)(void *);

typedef enum jayess_value_kind {
    JAYESS_VALUE_NULL = 0,
    JAYESS_VALUE_STRING = 1,
    JAYESS_VALUE_NUMBER = 2,
    JAYESS_VALUE_BIGINT = 3,
    JAYESS_VALUE_BOOL = 4,
    JAYESS_VALUE_OBJECT = 5,
    JAYESS_VALUE_ARRAY = 6,
    JAYESS_VALUE_UNDEFINED = 7,
    JAYESS_VALUE_FUNCTION = 8,
    JAYESS_VALUE_SYMBOL = 9
} jayess_value_kind;

jayess_value *jayess_value_null(void);
jayess_value *jayess_value_undefined(void);

void jayess_print_string(const char *text);
void jayess_print_number(double value);
void jayess_print_bool(int value);
void jayess_print_object(jayess_object *object);
void jayess_print_array(jayess_array *array);
void jayess_print_args(jayess_args *args);
void jayess_print_value(jayess_value *value);
void jayess_print_many(jayess_value *values);
void jayess_console_log(jayess_value *values);
void jayess_console_warn(jayess_value *values);
void jayess_console_error(jayess_value *values);
char *jayess_value_stringify(jayess_value *value);
char *jayess_template_string(jayess_value *parts, jayess_value *values);
char *jayess_concat_values(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_add(jayess_value *left, jayess_value *right);

char *jayess_read_line(const char *prompt);
char *jayess_read_key(const char *prompt);
void jayess_sleep_ms(int milliseconds);

jayess_args *jayess_make_args(int argc, char **argv);
char *jayess_args_get(jayess_args *args, int index);
int jayess_args_length(jayess_args *args);

jayess_object *jayess_object_new(void);
void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value);
jayess_value *jayess_object_get(jayess_object *object, const char *key);
void jayess_object_delete(jayess_object *object, const char *key);
jayess_array *jayess_object_keys(jayess_object *object);

void jayess_value_set_member(jayess_value *target, const char *key, jayess_value *value);
jayess_value *jayess_value_get_member(jayess_value *target, const char *key);
void jayess_value_delete_member(jayess_value *target, const char *key);
jayess_value *jayess_value_object_keys(jayess_value *target);
jayess_value *jayess_value_object_symbols(jayess_value *target);
void jayess_value_set_computed_member(jayess_value *target, jayess_value *key, jayess_value *value);
jayess_value *jayess_value_object_rest(jayess_value *target, jayess_value *excluded_keys);
jayess_value *jayess_std_map_new(void);
jayess_value *jayess_std_set_new(void);
jayess_value *jayess_std_weak_map_new(void);
jayess_value *jayess_std_weak_set_new(void);
jayess_value *jayess_std_symbol(jayess_value *description);
jayess_value *jayess_std_symbol_for(jayess_value *key);
jayess_value *jayess_std_symbol_key_for(jayess_value *symbol);
jayess_value *jayess_std_symbol_iterator(void);
jayess_value *jayess_std_symbol_async_iterator(void);
jayess_value *jayess_std_symbol_to_string_tag(void);
jayess_value *jayess_std_symbol_has_instance(void);
jayess_value *jayess_std_symbol_species(void);
jayess_value *jayess_std_symbol_match(void);
jayess_value *jayess_std_symbol_replace(void);
jayess_value *jayess_std_symbol_search(void);
jayess_value *jayess_std_symbol_split(void);
jayess_value *jayess_std_symbol_to_primitive(void);
jayess_value *jayess_std_date_new(jayess_value *value);
jayess_value *jayess_std_date_now(void);
jayess_value *jayess_std_regexp_new(jayess_value *pattern, jayess_value *flags);
jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message);
jayess_value *jayess_std_aggregate_error_new(jayess_value *errors, jayess_value *message);
jayess_value *jayess_std_array_buffer_new(jayess_value *length);
jayess_value *jayess_std_shared_array_buffer_new(jayess_value *length);
jayess_value *jayess_std_int8_array_new(jayess_value *source);
jayess_value *jayess_std_uint8_array_new(jayess_value *source);
jayess_value *jayess_std_uint16_array_new(jayess_value *source);
jayess_value *jayess_std_int16_array_new(jayess_value *source);
jayess_value *jayess_std_uint32_array_new(jayess_value *source);
jayess_value *jayess_std_int32_array_new(jayess_value *source);
jayess_value *jayess_std_float32_array_new(jayess_value *source);
jayess_value *jayess_std_float64_array_new(jayess_value *source);
jayess_value *jayess_std_data_view_new(jayess_value *buffer);
jayess_value *jayess_std_uint8_array_from_string(jayess_value *source, jayess_value *encoding);
jayess_value *jayess_std_uint8_array_concat(jayess_value *values);
jayess_value *jayess_std_uint8_array_equals(jayess_value *left, jayess_value *right);
jayess_value *jayess_std_uint8_array_compare(jayess_value *left, jayess_value *right);
jayess_value *jayess_std_iterator_from(jayess_value *target);
jayess_value *jayess_std_async_iterator_from(jayess_value *target);
jayess_value *jayess_std_promise_resolve(jayess_value *value);
jayess_value *jayess_std_promise_reject(jayess_value *reason);
jayess_value *jayess_std_promise_all(jayess_value *values);
jayess_value *jayess_std_promise_race(jayess_value *values);
jayess_value *jayess_std_promise_all_settled(jayess_value *values);
jayess_value *jayess_std_promise_any(jayess_value *values);
jayess_value *jayess_await(jayess_value *value);
jayess_value *jayess_set_timeout(jayess_value *callback, jayess_value *delay);
jayess_value *jayess_clear_timeout(jayess_value *id);
jayess_value *jayess_sleep_async(jayess_value *delay, jayess_value *value);
void jayess_run_microtasks(void);
void jayess_runtime_shutdown(void);
void jayess_throw_not_function(void);
jayess_value *jayess_std_json_stringify(jayess_value *value);
jayess_value *jayess_std_json_parse(jayess_value *value);
jayess_value *jayess_value_iterable_values(jayess_value *target);
jayess_value *jayess_value_object_values(jayess_value *target);
jayess_value *jayess_value_object_entries(jayess_value *target);
jayess_value *jayess_value_object_assign(jayess_value *target, jayess_value *source);
jayess_value *jayess_value_object_has_own(jayess_value *target, jayess_value *key);
double jayess_math_floor(double value);
double jayess_math_ceil(double value);
double jayess_math_round(double value);
double jayess_math_min(double left, double right);
double jayess_math_max(double left, double right);
double jayess_math_abs(double value);
double jayess_math_pow(double left, double right);
double jayess_math_sqrt(double value);
double jayess_math_random(void);
jayess_value *jayess_std_number_is_nan(jayess_value *value);
jayess_value *jayess_std_number_is_finite(jayess_value *value);
jayess_value *jayess_std_string_from_char_code(jayess_value *codes);
jayess_value *jayess_std_array_is_array(jayess_value *value);
jayess_value *jayess_std_array_from(jayess_value *value);
jayess_value *jayess_std_array_of(jayess_value *values);
jayess_value *jayess_std_object_from_entries(jayess_value *entries);
jayess_value *jayess_std_process_cwd(void);
jayess_value *jayess_std_process_env(jayess_value *name);
jayess_value *jayess_std_process_exit(jayess_value *code);
jayess_value *jayess_std_process_argv(void);
jayess_value *jayess_std_process_platform(void);
jayess_value *jayess_std_process_arch(void);
jayess_value *jayess_std_process_tmpdir(void);
jayess_value *jayess_std_process_hostname(void);
double jayess_std_process_uptime(void);
double jayess_std_process_hrtime(void);
jayess_value *jayess_std_process_cpu_info(void);
jayess_value *jayess_std_process_memory_info(void);
jayess_value *jayess_std_process_user_info(void);
jayess_value *jayess_std_process_thread_pool_size(void);
jayess_value *jayess_std_process_on_signal(jayess_value *signal, jayess_value *callback);
jayess_value *jayess_std_process_once_signal(jayess_value *signal, jayess_value *callback);
jayess_value *jayess_std_process_off_signal(jayess_value *signal, jayess_value *callback);
jayess_value *jayess_std_process_raise(jayess_value *signal);
jayess_value *jayess_std_compile(jayess_value *source, jayess_value *output_path);
jayess_value *jayess_std_compile_file(jayess_value *input_path, jayess_value *options);
jayess_value *jayess_std_path_join(jayess_value *parts);
jayess_value *jayess_std_path_normalize(jayess_value *path);
jayess_value *jayess_std_path_resolve(jayess_value *parts);
jayess_value *jayess_std_path_relative(jayess_value *from, jayess_value *to);
jayess_value *jayess_std_path_parse(jayess_value *path);
jayess_value *jayess_std_path_is_absolute(jayess_value *path);
jayess_value *jayess_std_path_format(jayess_value *parts);
jayess_value *jayess_std_path_sep(void);
jayess_value *jayess_std_path_delimiter(void);
jayess_value *jayess_std_path_basename(jayess_value *path);
jayess_value *jayess_std_path_dirname(jayess_value *path);
jayess_value *jayess_std_path_extname(jayess_value *path);
jayess_value *jayess_std_url_parse(jayess_value *input);
jayess_value *jayess_std_url_format(jayess_value *parts);
jayess_value *jayess_std_querystring_parse(jayess_value *input);
jayess_value *jayess_std_querystring_stringify(jayess_value *parts);
jayess_value *jayess_std_dns_lookup(jayess_value *host);
jayess_value *jayess_std_dns_lookup_all(jayess_value *host);
jayess_value *jayess_std_dns_reverse(jayess_value *address);
jayess_value *jayess_std_child_process_exec(jayess_value *options);
jayess_value *jayess_std_child_process_spawn(jayess_value *options);
jayess_value *jayess_std_child_process_kill(jayess_value *options);
jayess_value *jayess_std_worker_create(jayess_value *handler);
double jayess_atomics_load(jayess_value *target, jayess_value *index);
double jayess_atomics_store(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_add(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_sub(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_and(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_or(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_xor(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_exchange(jayess_value *target, jayess_value *index, jayess_value *value);
double jayess_atomics_compareExchange(jayess_value *target, jayess_value *index, jayess_value *expected, jayess_value *replacement);
jayess_value *jayess_std_crypto_random_bytes(jayess_value *length);
jayess_value *jayess_std_crypto_hash(jayess_value *algorithm, jayess_value *value);
jayess_value *jayess_std_crypto_hmac(jayess_value *algorithm, jayess_value *key, jayess_value *value);
jayess_value *jayess_std_crypto_secure_compare(jayess_value *left, jayess_value *right);
jayess_value *jayess_std_crypto_encrypt(jayess_value *options);
jayess_value *jayess_std_crypto_decrypt(jayess_value *options);
jayess_value *jayess_std_crypto_generate_key_pair(jayess_value *options);
jayess_value *jayess_std_crypto_public_encrypt(jayess_value *options);
jayess_value *jayess_std_crypto_private_decrypt(jayess_value *options);
jayess_value *jayess_std_crypto_sign(jayess_value *options);
jayess_value *jayess_std_crypto_verify(jayess_value *options);
jayess_value *jayess_std_compression_gzip(jayess_value *value);
jayess_value *jayess_std_compression_gunzip(jayess_value *value);
jayess_value *jayess_std_compression_deflate(jayess_value *value);
jayess_value *jayess_std_compression_inflate(jayess_value *value);
jayess_value *jayess_std_compression_brotli(jayess_value *value);
jayess_value *jayess_std_compression_unbrotli(jayess_value *value);
jayess_value *jayess_std_compression_create_gzip_stream(void);
jayess_value *jayess_std_compression_create_gunzip_stream(void);
jayess_value *jayess_std_compression_create_deflate_stream(void);
jayess_value *jayess_std_compression_create_inflate_stream(void);
jayess_value *jayess_std_compression_create_brotli_stream(void);
jayess_value *jayess_std_compression_create_unbrotli_stream(void);
jayess_value *jayess_std_net_is_ip(jayess_value *input);
jayess_value *jayess_std_net_create_datagram_socket(jayess_value *options);
jayess_value *jayess_std_net_connect(jayess_value *options);
jayess_value *jayess_std_net_listen(jayess_value *options);
jayess_value *jayess_std_tls_is_available(void);
jayess_value *jayess_std_tls_backend(void);
jayess_value *jayess_std_tls_connect(jayess_value *options);
jayess_value *jayess_std_tls_create_server(jayess_value *options, jayess_value *handler);
jayess_value *jayess_std_https_is_available(void);
jayess_value *jayess_std_https_backend(void);
jayess_value *jayess_std_http_parse_request(jayess_value *input);
jayess_value *jayess_std_http_format_request(jayess_value *parts);
jayess_value *jayess_std_http_parse_response(jayess_value *input);
jayess_value *jayess_std_http_format_response(jayess_value *parts);
jayess_value *jayess_std_http_request(jayess_value *options);
jayess_value *jayess_std_http_create_server(jayess_value *handler);
jayess_value *jayess_std_https_create_server(jayess_value *options, jayess_value *handler);
jayess_value *jayess_std_http_get(jayess_value *input);
jayess_value *jayess_std_http_request_async(jayess_value *options);
jayess_value *jayess_std_http_get_async(jayess_value *input);
jayess_value *jayess_std_fs_read_file(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_read_file_async(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_write_file(jayess_value *path, jayess_value *content);
jayess_value *jayess_std_fs_append_file(jayess_value *path, jayess_value *content);
jayess_value *jayess_std_fs_write_file_async(jayess_value *path, jayess_value *content);
jayess_value *jayess_std_fs_create_read_stream(jayess_value *path);
jayess_value *jayess_std_fs_create_write_stream(jayess_value *path);
jayess_value *jayess_std_fs_exists(jayess_value *path);
jayess_value *jayess_std_fs_read_dir(jayess_value *path, jayess_value *options);
jayess_value *jayess_std_fs_stat(jayess_value *path);
jayess_value *jayess_std_fs_mkdir(jayess_value *path, jayess_value *options);
jayess_value *jayess_std_fs_remove(jayess_value *path, jayess_value *options);
jayess_value *jayess_std_fs_copy_file(jayess_value *from, jayess_value *to);
jayess_value *jayess_std_fs_copy_dir(jayess_value *from, jayess_value *to);
jayess_value *jayess_std_fs_rename(jayess_value *from, jayess_value *to);
jayess_value *jayess_std_fs_symlink(jayess_value *target, jayess_value *path);
jayess_value *jayess_std_fs_watch(jayess_value *path);

jayess_array *jayess_array_new(void);
void jayess_array_set_value(jayess_array *array, int index, jayess_value *value);
jayess_value *jayess_array_get(jayess_array *array, int index);
int jayess_array_length(jayess_array *array);
int jayess_array_push_value(jayess_array *array, jayess_value *value);
jayess_value *jayess_array_pop_value(jayess_array *array);
jayess_value *jayess_array_shift_value(jayess_array *array);
int jayess_array_unshift_value(jayess_array *array, jayess_value *value);
jayess_array *jayess_array_slice_values(jayess_array *array, int start, int end, int has_end);

void jayess_value_set_index(jayess_value *target, int index, jayess_value *value);
jayess_value *jayess_value_get_index(jayess_value *target, int index);
void jayess_value_set_dynamic_index(jayess_value *target, jayess_value *index, jayess_value *value);
jayess_value *jayess_value_get_dynamic_index(jayess_value *target, jayess_value *index);
void jayess_value_delete_dynamic_index(jayess_value *target, jayess_value *index);
int jayess_value_array_length(jayess_value *target);
jayess_value *jayess_value_array_push(jayess_value *target, jayess_value *value);
jayess_value *jayess_value_array_pop(jayess_value *target);
jayess_value *jayess_value_array_shift(jayess_value *target);
jayess_value *jayess_value_array_unshift(jayess_value *target, jayess_value *value);
jayess_value *jayess_value_array_slice(jayess_value *target, int start, int end, int has_end);
jayess_value *jayess_value_array_includes(jayess_value *target, jayess_value *value);
jayess_value *jayess_value_array_join(jayess_value *target, jayess_value *separator);
void jayess_array_append_array(jayess_array *array, jayess_array *other);

jayess_value *jayess_value_from_string(const char *value);
jayess_value *jayess_value_from_number(double value);
jayess_value *jayess_value_from_bigint(const char *value);
jayess_value *jayess_value_from_bool(int value);
jayess_value *jayess_value_from_symbol(const char *description);
jayess_value *jayess_value_bitwise_not(jayess_value *value);
jayess_value *jayess_value_bitwise_and(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_bitwise_or(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_bitwise_xor(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_bitwise_shl(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_bitwise_shr(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_bitwise_ushr(jayess_value *left, jayess_value *right);
jayess_value *jayess_value_from_object(jayess_object *value);
jayess_value *jayess_value_from_array(jayess_array *value);
jayess_value *jayess_value_from_args(jayess_args *args);
jayess_value *jayess_value_from_bytes_copy(const unsigned char *bytes, size_t length);
unsigned char *jayess_value_to_bytes_copy(jayess_value *value, size_t *length_out);
char *jayess_value_to_string_copy(jayess_value *value);
void jayess_string_free(char *text);
void jayess_bytes_free(void *bytes);
jayess_value *jayess_value_from_native_handle(const char *kind, void *handle);
jayess_value *jayess_value_from_managed_native_handle(const char *kind, void *handle, jayess_native_handle_finalizer finalizer);
void *jayess_value_as_native_handle(jayess_value *value, const char *kind);
void jayess_value_clear_native_handle(jayess_value *value);
int jayess_value_close_native_handle(jayess_value *value);
jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest);
jayess_value *jayess_call_function(jayess_value *callback, jayess_value *argument);
jayess_value *jayess_call_function2(jayess_value *callback, jayess_value *first, jayess_value *second);
void *jayess_value_function_ptr(jayess_value *value);
jayess_value *jayess_value_function_env(jayess_value *value);
jayess_value *jayess_value_bind(jayess_value *value, jayess_value *bound_this, jayess_value *bound_args);
jayess_value *jayess_value_function_bound_this(jayess_value *value);
const char *jayess_value_function_class_name(jayess_value *value);
int jayess_value_function_param_count(jayess_value *value);
int jayess_value_function_has_rest(jayess_value *value);
int jayess_value_function_bound_arg_count(jayess_value *value);
jayess_value *jayess_value_function_bound_arg(jayess_value *value, int index);
jayess_value *jayess_value_merge_bound_args(jayess_value *value, jayess_value *tail_args);
jayess_value *jayess_error_value(const char *name, const char *message);
void jayess_throw(jayess_value *value);
void jayess_throw_error(const char *message);
void jayess_throw_type_error(const char *message);
void jayess_throw_named_error(const char *name, const char *message);
int jayess_has_exception(void);
jayess_value *jayess_take_exception(void);
void jayess_report_uncaught_exception(void);
void jayess_push_call_frame(const char *name);
void jayess_pop_call_frame(void);
void jayess_push_this(jayess_value *value);
void jayess_pop_this(void);
jayess_value *jayess_current_this(void);
const char *jayess_value_typeof(jayess_value *value);
int jayess_value_instanceof(jayess_value *target, const char *class_name);

double jayess_value_to_number(jayess_value *value);
int jayess_value_eq(jayess_value *left, jayess_value *right);
int jayess_value_is_nullish(jayess_value *value);
int jayess_string_is_truthy(const char *value);
int jayess_string_eq(const char *left, const char *right);
int jayess_args_is_truthy(jayess_args *args);
int jayess_value_is_truthy(jayess_value *value);

jayess_value_kind jayess_value_kind_of(jayess_value *value);
const char *jayess_value_as_string(jayess_value *value);
int jayess_value_as_bool(jayess_value *value);
jayess_object *jayess_value_as_object(jayess_value *value);
jayess_array *jayess_value_as_array(jayess_value *value);
const char *jayess_expect_string(jayess_value *value, const char *context);
jayess_object *jayess_expect_object(jayess_value *value, const char *context);
jayess_array *jayess_expect_array(jayess_value *value, const char *context);
unsigned char *jayess_expect_bytes_copy(jayess_value *value, size_t *length_out, const char *context);
void *jayess_expect_native_handle(jayess_value *value, const char *kind, const char *context);

#ifdef __cplusplus
}
#endif

#endif
