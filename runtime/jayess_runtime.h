#ifndef JAYESS_RUNTIME_H
#define JAYESS_RUNTIME_H

typedef struct jayess_args jayess_args;
typedef struct jayess_value jayess_value;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;

typedef enum jayess_value_kind {
    JAYESS_VALUE_NULL = 0,
    JAYESS_VALUE_STRING = 1,
    JAYESS_VALUE_NUMBER = 2,
    JAYESS_VALUE_BOOL = 3,
    JAYESS_VALUE_OBJECT = 4,
    JAYESS_VALUE_ARRAY = 5,
    JAYESS_VALUE_UNDEFINED = 6,
    JAYESS_VALUE_FUNCTION = 7
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
void jayess_value_set_computed_member(jayess_value *target, jayess_value *key, jayess_value *value);
jayess_value *jayess_value_object_rest(jayess_value *target, jayess_value *excluded_keys);
jayess_value *jayess_std_map_new(void);
jayess_value *jayess_std_set_new(void);
jayess_value *jayess_std_date_new(jayess_value *value);
jayess_value *jayess_std_date_now(void);
jayess_value *jayess_std_regexp_new(jayess_value *pattern, jayess_value *flags);
jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message);
jayess_value *jayess_std_aggregate_error_new(jayess_value *errors, jayess_value *message);
jayess_value *jayess_std_array_buffer_new(jayess_value *length);
jayess_value *jayess_std_uint8_array_new(jayess_value *source);
jayess_value *jayess_std_data_view_new(jayess_value *buffer);
jayess_value *jayess_std_uint8_array_from_string(jayess_value *source, jayess_value *encoding);
jayess_value *jayess_std_uint8_array_concat(jayess_value *values);
jayess_value *jayess_std_uint8_array_equals(jayess_value *left, jayess_value *right);
jayess_value *jayess_std_uint8_array_compare(jayess_value *left, jayess_value *right);
jayess_value *jayess_std_iterator_from(jayess_value *target);
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
jayess_value *jayess_std_process_thread_pool_size(void);
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
jayess_value *jayess_std_net_is_ip(jayess_value *input);
jayess_value *jayess_std_net_connect(jayess_value *options);
jayess_value *jayess_std_net_listen(jayess_value *options);
jayess_value *jayess_std_http_parse_request(jayess_value *input);
jayess_value *jayess_std_http_format_request(jayess_value *parts);
jayess_value *jayess_std_http_parse_response(jayess_value *input);
jayess_value *jayess_std_http_format_response(jayess_value *parts);
jayess_value *jayess_std_http_request(jayess_value *options);
jayess_value *jayess_std_http_get(jayess_value *input);
jayess_value *jayess_std_http_request_async(jayess_value *options);
jayess_value *jayess_std_http_get_async(jayess_value *input);
jayess_value *jayess_std_fs_read_file(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_read_file_async(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_write_file(jayess_value *path, jayess_value *content);
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
jayess_value *jayess_value_from_bool(int value);
jayess_value *jayess_value_from_object(jayess_object *value);
jayess_value *jayess_value_from_array(jayess_array *value);
jayess_value *jayess_value_from_args(jayess_args *args);
jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest);
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
void jayess_throw(jayess_value *value);
int jayess_has_exception(void);
jayess_value *jayess_take_exception(void);
void jayess_report_uncaught_exception(void);
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

#endif
