#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <stdint.h>
#include <math.h>
#include <time.h>
#include <signal.h>

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef SECURITY_WIN32
#define SECURITY_WIN32
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#include <conio.h>
#include <direct.h>
#include <io.h>
#include <windows.h>
#include <winhttp.h>
#include <wincrypt.h>
#include <bcrypt.h>
#include <security.h>
#include <schannel.h>
#include <zlib.h>
#include <brotli/encode.h>
#include <brotli/decode.h>
#else
#include <arpa/inet.h>
#include <dirent.h>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/tcp.h>
#include <pthread.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>
#include <errno.h>
#include <pwd.h>
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>
#include <openssl/rsa.h>
#include <openssl/ssl.h>
#include <openssl/err.h>
#include <openssl/x509v3.h>
#include <zlib.h>
#include <brotli/encode.h>
#include <brotli/decode.h>
#endif

#include "jayess_runtime_core.h"

char *jayess_value_stringify(jayess_value *value);
double jayess_value_to_number(jayess_value *value);
jayess_value *jayess_value_from_bigint(const char *value);
jayess_value *jayess_value_from_symbol(const char *description);
int jayess_value_eq(jayess_value *left, jayess_value *right);
int jayess_value_is_nullish(jayess_value *value);
const char *jayess_value_as_string(jayess_value *value);
int jayess_value_as_bool(jayess_value *value);
int jayess_string_eq(const char *left, const char *right);
jayess_value *jayess_value_null(void);
jayess_value *jayess_value_undefined(void);
jayess_value *jayess_value_from_string(const char *value);
jayess_value *jayess_value_from_number(double value);
jayess_value *jayess_value_from_bool(int value);
jayess_value *jayess_value_from_object(jayess_object *value);
jayess_value *jayess_value_from_array(jayess_array *value);
jayess_value *jayess_value_from_args(jayess_args *args);
jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest);
jayess_value *jayess_value_get_member(jayess_value *target, const char *key);
jayess_value *jayess_value_get_dynamic_index(jayess_value *target, jayess_value *index);
jayess_object *jayess_value_as_object(jayess_value *value);
jayess_value *jayess_value_array_includes(jayess_value *target, jayess_value *value);
jayess_value *jayess_value_array_join(jayess_value *target, jayess_value *separator);
jayess_value *jayess_value_object_symbols(jayess_value *target);
int jayess_value_array_length(jayess_value *target);
jayess_value *jayess_std_path_dirname(jayess_value *path);
jayess_value *jayess_std_path_extname(jayess_value *path);
jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message);
jayess_value *jayess_std_aggregate_error_new(jayess_value *errors, jayess_value *message);
jayess_value *jayess_std_fs_read_file(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_write_file(jayess_value *path, jayess_value *content);
jayess_value *jayess_std_fs_append_file(jayess_value *path, jayess_value *content);
jayess_value *jayess_std_fs_symlink(jayess_value *target, jayess_value *path);
jayess_value *jayess_std_fs_watch(jayess_value *path);
jayess_object *jayess_object_new(void);
void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value);
jayess_value *jayess_object_get(jayess_object *object, const char *key);
void jayess_object_delete(jayess_object *object, const char *key);
jayess_array *jayess_object_keys(jayess_object *object);
jayess_array *jayess_array_new(void);
void jayess_array_set_value(jayess_array *array, int index, jayess_value *value);
int jayess_array_push_value(jayess_array *array, jayess_value *value);
jayess_value *jayess_array_pop_value(jayess_array *array);
jayess_value *jayess_array_shift_value(jayess_array *array);
int jayess_array_unshift_value(jayess_array *array, jayess_value *value);
jayess_array *jayess_array_slice_values(jayess_array *array, int start, int end, int has_end);
jayess_value *jayess_array_get(jayess_array *array, int index);
jayess_value *jayess_value_iterable_values(jayess_value *target);
void jayess_push_this(jayess_value *value);
void jayess_pop_this(void);
void jayess_push_call_frame(const char *name);
void jayess_pop_call_frame(void);
jayess_value *jayess_type_error_value(const char *message);
void jayess_runtime_error_state_shutdown(void);
void jayess_throw(jayess_value *value);
void jayess_throw_type_error(const char *message);
int jayess_has_exception(void);
jayess_value *jayess_take_exception(void);
int jayess_std_child_process_signal_number(const char *signal_name);
const char *jayess_std_process_signal_name(int signal_number);
jayess_value *jayess_std_process_signal_bus_value(void);
int jayess_std_process_install_signal(int signal_number);
void jayess_runtime_note_signal(int signal_number);
void jayess_runtime_dispatch_pending_signals(void);
char *jayess_compile_option_string(jayess_value *options, const char *key);
jayess_value *jayess_std_worker_post_message_method(jayess_value *env, jayess_value *message);
jayess_value *jayess_std_worker_receive_method(jayess_value *env, jayess_value *timeout);
jayess_value *jayess_std_worker_terminate_method(jayess_value *env);
jayess_value *jayess_set_timeout(jayess_value *callback, jayess_value *delay);
jayess_value *jayess_clear_timeout(jayess_value *id);
void jayess_run_microtasks(void);
void jayess_throw_not_function(void);

static jayess_value jayess_null_singleton = {JAYESS_VALUE_NULL, {0}};
static jayess_value jayess_undefined_singleton = {JAYESS_VALUE_UNDEFINED, {0}};
static JAYESS_THREAD_LOCAL jayess_args *jayess_current_args = NULL;
static JAYESS_THREAD_LOCAL jayess_scheduler jayess_runtime_scheduler = {{NULL, NULL}, {NULL, NULL}, {NULL, NULL}, {NULL, NULL}};
static jayess_io_worker_pool jayess_runtime_io_pool = {0};
static JAYESS_THREAD_LOCAL int jayess_next_timer_id = 1;
static uint64_t jayess_next_symbol_id = 1;
static jayess_symbol_registry_entry *jayess_symbol_registry = NULL;
static jayess_value *jayess_symbol_iterator_singleton = NULL;
static jayess_value *jayess_symbol_async_iterator_singleton = NULL;
static jayess_value *jayess_symbol_to_string_tag_singleton = NULL;
static jayess_value *jayess_symbol_has_instance_singleton = NULL;
static jayess_value *jayess_symbol_species_singleton = NULL;
static jayess_value *jayess_symbol_match_singleton = NULL;
static jayess_value *jayess_symbol_replace_singleton = NULL;
static jayess_value *jayess_symbol_search_singleton = NULL;
static jayess_value *jayess_symbol_split_singleton = NULL;
static jayess_value *jayess_symbol_to_primitive_singleton = NULL;
jayess_value *jayess_process_signal_bus = NULL;
static jayess_value *jayess_dns_custom_resolver = NULL;

static jayess_value *jayess_std_promise_pending(void);
static void jayess_enqueue_microtask(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected);
static void jayess_enqueue_promise_task(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected, jayess_promise_action action);
static void jayess_requeue_microtask(jayess_microtask *task);
static void jayess_append_microtask(jayess_microtask *task);
int jayess_object_entry_is_symbol(jayess_object_entry *entry);
int jayess_object_entry_matches_string(jayess_object_entry *entry, const char *key);
int jayess_object_entry_matches_value(jayess_object_entry *entry, jayess_value *key);
jayess_object_entry *jayess_object_find_value(jayess_object *object, jayess_value *key);
void jayess_object_set_key_value(jayess_object *object, jayess_value *key, jayess_value *value);
jayess_value *jayess_object_get_key_value(jayess_object *object, jayess_value *key);
void jayess_object_delete_key_value(jayess_object *object, jayess_value *key);
void jayess_print_property_key_inline(jayess_object_entry *entry);
static jayess_value *jayess_value_to_property_key(jayess_value *key);
static jayess_value *jayess_std_symbol_to_string_method(jayess_value *env);
static jayess_symbol_registry_entry *jayess_symbol_registry_find(const char *key);
static jayess_value *jayess_symbol_singleton(jayess_value **slot, const char *description);
static jayess_value *jayess_std_async_iterator_next_method(jayess_value *env);
static jayess_value *jayess_std_async_iterator_identity_method(jayess_value *env);
static jayess_value *jayess_std_iterator_protocol_values(jayess_value *iterator);
static jayess_value *jayess_std_iterable_protocol_values(jayess_value *target);
void jayess_value_free_unshared(jayess_value *value);
void jayess_object_free_unshared(jayess_object *object);
void jayess_array_free_unshared(jayess_array *array);
jayess_array *jayess_std_bytes_slot(jayess_value *target);
jayess_value *jayess_std_array_buffer_new(jayess_value *length_value);
jayess_value *jayess_std_shared_array_buffer_new(jayess_value *length_value);
jayess_value *jayess_std_data_view_new(jayess_value *buffer);
const char *jayess_std_typed_array_kind(jayess_value *target);
int jayess_std_byte_length(jayess_value *target);
int jayess_std_is_typed_array(jayess_value *target);
int jayess_std_typed_array_length_from_bytes(jayess_array *bytes, const char *kind);
jayess_value *jayess_std_uint8_array_from_bytes(const unsigned char *bytes, size_t length);
double jayess_std_typed_array_get_number(jayess_value *target, int index);
void jayess_std_typed_array_set_number(jayess_value *target, int index, double number);
jayess_value *jayess_std_typed_array_new(const char *kind, jayess_value *source);
jayess_value *jayess_std_uint8_array_new(jayess_value *source);
jayess_value *jayess_std_data_view_get_uint8_method(jayess_value *env, jayess_value *offset_value);
jayess_value *jayess_std_data_view_set_uint8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value);
jayess_value *jayess_std_data_view_get_int8_method(jayess_value *env, jayess_value *offset_value);
jayess_value *jayess_std_data_view_set_int8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value);
jayess_value *jayess_std_data_view_get_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_get_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_get_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_get_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_get_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_get_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian);
jayess_value *jayess_std_data_view_set_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian);
jayess_value *jayess_std_typed_array_fill_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_typed_array_includes_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_typed_array_index_of_method(jayess_value *env, jayess_value *needle);
jayess_value *jayess_std_typed_array_set_method(jayess_value *env, jayess_value *source, jayess_value *offset_value);
jayess_value *jayess_std_typed_array_slice_method(jayess_value *env, jayess_value *start_value, jayess_value *end_value);
jayess_value *jayess_std_typed_array_slice_values(jayess_value *env, int start, int end, int has_end);
static jayess_value *jayess_std_uint8_index_of_method(jayess_value *env, jayess_value *needle);
static int jayess_std_uint8_clamped_index(jayess_value *value, int length, int default_value);
int jayess_std_bytes_encoding_is_hex(jayess_value *encoding);
int jayess_std_bytes_encoding_is_base64(jayess_value *encoding);
int jayess_std_bytes_encoding_is_text(jayess_value *encoding);
static jayess_value *jayess_std_socket_read_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_socket_read_bytes_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_socket_write_method(jayess_value *env, jayess_value *value);
static jayess_value *jayess_std_socket_close_method(jayess_value *env);
static jayess_value *jayess_std_socket_read_async_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_socket_write_async_method(jayess_value *env, jayess_value *value);
static jayess_value *jayess_std_socket_set_no_delay_method(jayess_value *env, jayess_value *enabled);
static jayess_value *jayess_std_socket_set_keep_alive_method(jayess_value *env, jayess_value *enabled);
static jayess_value *jayess_std_socket_set_timeout_method(jayess_value *env, jayess_value *timeout_ms);
static jayess_value *jayess_std_socket_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_socket_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_socket_address_method(jayess_value *env);
static jayess_value *jayess_std_socket_remote_method(jayess_value *env);
static jayess_value *jayess_std_socket_get_peer_certificate_method(jayess_value *env);
static jayess_value *jayess_std_datagram_socket_send_method(jayess_value *env, jayess_value *value, jayess_value *port_value, jayess_value *host_value);
static jayess_value *jayess_std_datagram_socket_receive_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_datagram_socket_set_broadcast_method(jayess_value *env, jayess_value *enabled);
static jayess_value *jayess_std_datagram_socket_join_group_method(jayess_value *env, jayess_value *group_value, jayess_value *interface_value);
static jayess_value *jayess_std_datagram_socket_leave_group_method(jayess_value *env, jayess_value *group_value, jayess_value *interface_value);
static jayess_value *jayess_std_datagram_socket_set_multicast_interface_method(jayess_value *env, jayess_value *interface_value);
static jayess_value *jayess_std_datagram_socket_set_multicast_loopback_method(jayess_value *env, jayess_value *enabled);
static jayess_value *jayess_std_server_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_server_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_server_accept_method(jayess_value *env);
static jayess_value *jayess_std_server_close_method(jayess_value *env);
static jayess_value *jayess_std_server_address_method(jayess_value *env);
static jayess_value *jayess_std_server_set_timeout_method(jayess_value *env, jayess_value *timeout_ms);
static jayess_value *jayess_std_server_accept_async_method(jayess_value *env);
FILE *jayess_std_stream_file(jayess_value *env);
void jayess_std_stream_set_file(jayess_value *env, FILE *file);
int jayess_std_stream_bool_property(jayess_value *env, const char *key);
int jayess_std_stream_backpressure_note_pending(jayess_value *env, double pending_length);
void jayess_std_stream_backpressure_maybe_drain(jayess_value *env, double pending_length);
void jayess_std_stream_on(jayess_value *env, const char *event, jayess_value *callback);
void jayess_std_stream_once(jayess_value *env, const char *event, jayess_value *callback);
void jayess_std_stream_off(jayess_value *env, const char *event, jayess_value *callback);
jayess_value *jayess_std_stream_off_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_stream_listener_count_method(jayess_value *env, jayess_value *event);
jayess_value *jayess_std_stream_event_names_method(jayess_value *env);
void jayess_std_stream_emit(jayess_value *env, const char *event, jayess_value *argument);
void jayess_std_stream_emit_error(jayess_value *env, const char *message);
void jayess_std_stream_register_error_handler(jayess_value *env, jayess_value *callback);
void jayess_std_stream_register_error_once(jayess_value *env, jayess_value *callback);
int jayess_std_stream_requested_size(jayess_value *size_value, int default_size);
static int jayess_std_socket_configure_timeout(jayess_socket_handle handle, int timeout);
static jayess_value *jayess_std_http_server_listen_method(jayess_value *env, jayess_value *port_value, jayess_value *host_value);
static jayess_value *jayess_std_http_server_close_method(jayess_value *env);
static jayess_value *jayess_std_http_response_set_header_method(jayess_value *env, jayess_value *name, jayess_value *value);
static jayess_value *jayess_std_http_response_write_method(jayess_value *env, jayess_value *chunk);
static jayess_value *jayess_std_http_response_end_method(jayess_value *env, jayess_value *chunk);
int jayess_path_is_separator(char ch);
const char *jayess_path_last_separator(const char *text);
int jayess_path_is_absolute(const char *text);
char jayess_path_separator_char(void);
const char *jayess_path_separator_string(void);
const char *jayess_path_delimiter_string(void);
int jayess_path_root_length(const char *text);
char jayess_path_preferred_separator_char(const char *text);
jayess_array *jayess_path_split_segments(const char *text);
char *jayess_path_join_segments_with_root(const char *root, jayess_array *segments, char sep);
int jayess_path_exists_text(const char *path_text);
int jayess_path_is_dir_text(const char *path_text);
int jayess_path_mkdir_single(const char *path_text);
int jayess_fs_remove_path_recursive(const char *path_text);
int jayess_object_option_bool(jayess_value *options, const char *key);
double jayess_path_file_size_text(const char *path_text);
double jayess_path_modified_time_ms_text(const char *path_text);
void jayess_fs_watch_snapshot_text(const char *path_text, int *exists, int *is_dir, double *size, double *mtime_ms);
const char *jayess_path_permissions_text(const char *path_text);
jayess_value *jayess_fs_dir_entry_value(const char *name, const char *full_path, int is_dir);
void jayess_fs_read_dir_collect(jayess_array *entries, const char *path_text, int recursive);
jayess_value *jayess_std_fs_stream_open_error(const char *kind, const char *message);
jayess_value *jayess_std_https_create_server(jayess_value *options, jayess_value *handler);
jayess_value *jayess_std_tls_create_server(jayess_value *options, jayess_value *handler);
jayess_value *jayess_std_http_parse_request(jayess_value *input);
jayess_value *jayess_std_http_format_request(jayess_value *parts);
jayess_value *jayess_std_http_parse_response(jayess_value *input);
jayess_value *jayess_std_http_format_response(jayess_value *parts);
jayess_value *jayess_std_url_parse(jayess_value *input);
jayess_value *jayess_std_net_listen(jayess_value *options);
jayess_value *jayess_std_http_request(jayess_value *options);
jayess_value *jayess_std_http_get(jayess_value *input);
jayess_value *jayess_std_http_request_stream(jayess_value *options);
jayess_value *jayess_std_http_get_stream(jayess_value *input);
jayess_value *jayess_std_http_request_stream_async(jayess_value *options);
jayess_value *jayess_std_http_get_stream_async(jayess_value *input);
jayess_value *jayess_std_http_request_async(jayess_value *options);
jayess_value *jayess_std_http_get_async(jayess_value *input);
jayess_value *jayess_std_https_request(jayess_value *options);
jayess_value *jayess_std_https_request_stream(jayess_value *options);
jayess_value *jayess_std_https_get(jayess_value *input);
jayess_value *jayess_std_https_get_stream(jayess_value *input);
jayess_value *jayess_std_https_request_stream_async(jayess_value *options);
jayess_value *jayess_std_https_get_stream_async(jayess_value *input);
jayess_value *jayess_std_https_request_async(jayess_value *options);
jayess_value *jayess_std_https_get_async(jayess_value *input);
jayess_value *jayess_std_tls_is_available(void);
jayess_value *jayess_std_tls_backend(void);
jayess_value *jayess_std_tls_connect(jayess_value *options);
jayess_value *jayess_std_https_is_available(void);
jayess_value *jayess_std_https_backend(void);
jayess_array *jayess_std_bytes_slot(jayess_value *target);
int jayess_std_crypto_copy_bytes(jayess_value *value, unsigned char **out_bytes, size_t *out_length);
char *jayess_std_crypto_hex_encode(const unsigned char *bytes, size_t length);
int jayess_std_crypto_equal_name(const char *left, const char *right);
void jayess_std_crypto_normalize_name(char *text);
int jayess_std_crypto_cipher_key_length(const char *algorithm);
int jayess_std_crypto_option_bytes(jayess_value *options, const char *key, unsigned char **out_bytes, size_t *out_length, int required);
jayess_value *jayess_std_crypto_key_value(const char *type, int is_private);
jayess_crypto_key_state *jayess_std_crypto_key_state_from_value(jayess_value *value);
jayess_value *jayess_std_compression_gzip(jayess_value *value);
jayess_value *jayess_std_compression_gunzip(jayess_value *value);
jayess_value *jayess_std_compression_deflate(jayess_value *value);
jayess_value *jayess_std_compression_inflate(jayess_value *value);
jayess_value *jayess_std_compression_brotli(jayess_value *value);
jayess_value *jayess_std_compression_unbrotli(jayess_value *value);
jayess_value *jayess_std_compression_stream_write_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_compression_stream_end_method(jayess_value *env);
jayess_value *jayess_std_compression_stream_read_method(jayess_value *env, jayess_value *size_value);
jayess_value *jayess_std_compression_stream_read_bytes_method(jayess_value *env, jayess_value *size_value);
jayess_value *jayess_std_compression_stream_pipe_method(jayess_value *env, jayess_value *destination);
jayess_value *jayess_std_compression_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_compression_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_writable_write(jayess_value *destination, jayess_value *chunk);
jayess_value *jayess_std_writable_end(jayess_value *destination);
static int jayess_std_socket_runtime_ready(void);
static jayess_value *jayess_std_read_stream_read_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_read_stream_read_bytes_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_read_stream_close_method(jayess_value *env);
static jayess_value *jayess_std_read_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_read_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_read_stream_pipe_method(jayess_value *env, jayess_value *destination);
jayess_value *jayess_std_write_stream_write_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_write_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_write_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_write_stream_end_method(jayess_value *env);
int jayess_std_kind_is(jayess_value *target, const char *kind);
static jayess_value *jayess_std_fs_watch_poll_method(jayess_value *env);
static jayess_value *jayess_std_fs_watch_poll_async_method(jayess_value *env, jayess_value *timeout_ms);
static jayess_value *jayess_std_fs_watch_poll_async_tick(jayess_value *env);
static jayess_value *jayess_std_fs_watch_close_method(jayess_value *env);
static jayess_value *jayess_std_fs_watch_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_fs_watch_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static int jayess_std_socket_close_handle(jayess_socket_handle handle);
static jayess_value *jayess_std_socket_value_from_handle(jayess_socket_handle handle, const char *remote_address, int remote_port);
static jayess_value *jayess_std_datagram_socket_value_from_handle(jayess_socket_handle handle);
static void jayess_std_socket_set_local_endpoint(jayess_value *socket_value, jayess_socket_handle handle);
static void jayess_std_socket_set_remote_family(jayess_value *socket_value, int family);
static jayess_socket_handle jayess_std_socket_handle(jayess_value *env);
static void jayess_std_socket_set_handle(jayess_value *env, jayess_socket_handle handle);
static void jayess_std_socket_mark_closed(jayess_value *env);
static void jayess_std_socket_emit_close(jayess_value *env);
static void jayess_std_socket_close_native(jayess_value *env);
static jayess_value *jayess_std_http_body_stream_read_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_http_body_stream_read_bytes_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_http_body_stream_close_method(jayess_value *env);
static jayess_value *jayess_std_http_body_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_http_body_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_http_body_stream_pipe_method(jayess_value *env, jayess_value *destination);
static void jayess_http_body_stream_mark_ended(jayess_value *env);
static void jayess_http_body_stream_close_socket(jayess_value *env);
static void jayess_http_body_stream_close_native(jayess_value *env);
static jayess_value *jayess_http_body_stream_read_chunk(jayess_value *env, jayess_value *size_value, int as_bytes);
static int jayess_http_text_eq_ci(const char *left, const char *right);
static jayess_value *jayess_std_tls_normalize_alpn_protocols(jayess_value *value);
static int jayess_std_tls_build_alpn_wire(jayess_value *protocols_value, unsigned char **out_buffer, size_t *out_length);
static jayess_value *jayess_std_tls_subject_alt_names(jayess_value *env);

#ifdef _WIN32
static jayess_tls_socket_state *jayess_std_tls_state(jayess_value *env);
LPCWSTR jayess_std_crypto_algorithm_id(const char *name);
int jayess_std_crypto_sha256_bytes(const unsigned char *input, size_t input_length, unsigned char *output, DWORD *output_length);
static int jayess_std_tls_send_all(jayess_socket_handle handle, const unsigned char *buffer, size_t length);
static int jayess_std_tls_state_free(jayess_tls_socket_state *state, int close_handle);
static int jayess_std_tls_read_bytes(jayess_value *env, unsigned char *buffer, int max_count, int *did_timeout);
static int jayess_std_tls_write_bytes(jayess_value *env, const unsigned char *buffer, int length, int *did_timeout);
static jayess_value *jayess_std_tls_connect_socket(jayess_value *options);
static jayess_value *jayess_std_tls_accept_socket(jayess_value *socket_value, jayess_value *options);
static void jayess_std_https_copy_tls_request_settings(jayess_object *target, jayess_object *source);
#ifdef _WIN32
static int jayess_std_windows_load_certificates_from_file(HCERTSTORE store, const char *path);
static int jayess_std_windows_load_certificates_from_path(HCERTSTORE store, const char *path);
static int jayess_std_windows_validate_tls_certificate(jayess_tls_socket_state *state, const char *server_name, const char *ca_file, const char *ca_path, int trust_system);
static void *jayess_std_tls_build_schannel_alpn_buffer(const unsigned char *wire, size_t wire_length, unsigned long *buffer_length);
static const char *jayess_std_tls_windows_protocol_name(DWORD protocol);
#endif
static jayess_value *jayess_std_tls_peer_certificate(jayess_value *env);
#endif

char *jayess_strdup(const char *value) {
#ifdef _WIN32
    return _strdup(value);
#else
    return strdup(value);
#endif
}

#ifdef _WIN32
static wchar_t *jayess_utf8_to_wide(const char *value) {
    const char *text = value != NULL ? value : "";
    int needed = MultiByteToWideChar(CP_UTF8, 0, text, -1, NULL, 0);
    wchar_t *wide;
    if (needed <= 0) {
        return NULL;
    }
    wide = (wchar_t *)malloc((size_t)needed * sizeof(wchar_t));
    if (wide == NULL) {
        return NULL;
    }
    if (MultiByteToWideChar(CP_UTF8, 0, text, -1, wide, needed) <= 0) {
        free(wide);
        return NULL;
    }
    return wide;
}

static char *jayess_wide_to_utf8(const wchar_t *value) {
    const wchar_t *text = value != NULL ? value : L"";
    int needed = WideCharToMultiByte(CP_UTF8, 0, text, -1, NULL, 0, NULL, NULL);
    char *utf8;
    if (needed <= 0) {
        return NULL;
    }
    utf8 = (char *)malloc((size_t)needed);
    if (utf8 == NULL) {
        return NULL;
    }
    if (WideCharToMultiByte(CP_UTF8, 0, text, -1, utf8, needed, NULL, NULL) <= 0) {
        free(utf8);
        return NULL;
    }
    return utf8;
}

static int jayess_winhttp_add_headers(HINTERNET request, jayess_object *headers) {
    jayess_object_entry *entry = headers != NULL ? headers->head : NULL;
    while (entry != NULL) {
        if (!jayess_http_text_eq_ci(entry->key, "Host") && !jayess_http_text_eq_ci(entry->key, "Connection") && !jayess_http_text_eq_ci(entry->key, "Content-Length")) {
            char *value = jayess_value_stringify(entry->value);
            size_t line_len = strlen(entry->key != NULL ? entry->key : "") + strlen(value != NULL ? value : "") + 4;
            char *line = (char *)malloc(line_len);
            wchar_t *line_w;
            int ok = 1;
            if (line == NULL) {
                free(value);
                return 0;
            }
            snprintf(line, line_len, "%s: %s", entry->key != NULL ? entry->key : "", value != NULL ? value : "");
            line_w = jayess_utf8_to_wide(line);
            free(line);
            free(value);
            if (line_w == NULL) {
                return 0;
            }
            ok = WinHttpAddRequestHeaders(request, line_w, (DWORD)-1, WINHTTP_ADDREQ_FLAG_ADD | WINHTTP_ADDREQ_FLAG_REPLACE);
            free(line_w);
            if (!ok) {
                return 0;
            }
        }
        entry = entry->next;
    }
    return 1;
}
#endif

static int jayess_fs_copy_dir_recursive(const char *from_text, const char *to_text);

static jayess_fs_watch_state *jayess_std_fs_watch_state(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || !jayess_std_kind_is(env, "Watcher")) {
        return NULL;
    }
    return (jayess_fs_watch_state *)env->as.object_value->native_handle;
}

static void jayess_std_fs_watch_apply_snapshot(jayess_value *env, int exists, int is_dir, double size, double mtime_ms) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    jayess_object_set_value(env->as.object_value, "exists", jayess_value_from_bool(exists));
    jayess_object_set_value(env->as.object_value, "isDir", jayess_value_from_bool(is_dir));
    jayess_object_set_value(env->as.object_value, "isFile", jayess_value_from_bool(exists && !is_dir));
    jayess_object_set_value(env->as.object_value, "size", jayess_value_from_number(size));
    jayess_object_set_value(env->as.object_value, "mtimeMs", jayess_value_from_number(mtime_ms));
}

static jayess_value *jayess_std_fs_watch_event_value(jayess_fs_watch_state *state) {
    jayess_object *event;
    if (state == NULL) {
        return jayess_value_null();
    }
    event = jayess_object_new();
    if (event == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(event, "type", jayess_value_from_string("change"));
    jayess_object_set_value(event, "path", jayess_value_from_string(state->path != NULL ? state->path : ""));
    jayess_object_set_value(event, "exists", jayess_value_from_bool(state->exists));
    jayess_object_set_value(event, "isDir", jayess_value_from_bool(state->is_dir));
    jayess_object_set_value(event, "isFile", jayess_value_from_bool(state->exists && !state->is_dir));
    jayess_object_set_value(event, "size", jayess_value_from_number(state->size));
    jayess_object_set_value(event, "mtimeMs", jayess_value_from_number(state->mtime_ms));
    return jayess_value_from_object(event);
}

void jayess_print_value_inline(jayess_value *value);
static jayess_array *jayess_array_clone(jayess_array *array);
static jayess_array *jayess_array_concat(jayess_array *left, jayess_array *right);
static jayess_value *jayess_value_clone_bound_arg(jayess_value *value);
static jayess_array *jayess_array_concat_bound_args_owned(jayess_array *left, jayess_array *right);
jayess_array *jayess_array_new(void);
void jayess_array_set_value(jayess_array *array, int index, jayess_value *value);
jayess_value *jayess_value_from_string(const char *value);
jayess_value *jayess_value_from_array(jayess_array *value);
double jayess_value_to_number(jayess_value *value);
jayess_value *jayess_value_iterable_values(jayess_value *target);

jayess_value *jayess_value_null(void) {
    return &jayess_null_singleton;
}

jayess_value *jayess_value_undefined(void) {
    return &jayess_undefined_singleton;
}

void jayess_sleep_ms(int milliseconds) {
    int remaining;
    if (milliseconds <= 0) {
        jayess_runtime_dispatch_pending_signals();
        return;
    }
    remaining = milliseconds;
    while (remaining > 0) {
        int step = remaining > 10 ? 10 : remaining;
        jayess_runtime_dispatch_pending_signals();
#ifdef _WIN32
        Sleep((DWORD)step);
#else
        usleep((useconds_t)step * 1000);
#endif
        jayess_runtime_dispatch_pending_signals();
        if (jayess_has_exception()) {
            return;
        }
        remaining -= step;
    }
}

jayess_args *jayess_make_args(int argc, char **argv) {
    int i;
    jayess_args *args = (jayess_args *)malloc(sizeof(jayess_args));
    if (args == NULL) {
        return NULL;
    }
    if (argc <= 1) {
        args->count = 0;
        args->values = NULL;
        jayess_current_args = args;
        return args;
    }
    args->count = argc - 1;
    args->values = (char **)malloc(sizeof(char *) * (size_t)args->count);
    if (args->values == NULL) {
        free(args);
        return NULL;
    }
    for (i = 1; i < argc; i++) {
        args->values[i - 1] = argv[i];
    }
    jayess_current_args = args;
    return args;
}

char *jayess_args_get(jayess_args *args, int index) {
    if (args == NULL || index < 0 || index >= args->count) {
        return "";
    }
    return args->values[index];
}

int jayess_args_length(jayess_args *args) {
    if (args == NULL) {
        return 0;
    }
    return args->count;
}

void jayess_object_free_unshared(jayess_object *object) {
    jayess_object_entry *current;
    int managed_native_handle = 0;
    int native_handle_wrapper = 0;
    int bytes_backed_object = 0;
    if (object == NULL) {
        return;
    }
    current = object->head;
    while (current != NULL) {
        if (current->key != NULL && strcmp(current->key, "__jayess_std_kind") == 0 &&
            current->value != NULL && current->value->kind == JAYESS_VALUE_STRING &&
            current->value->as.string_value != NULL) {
            if (strcmp(current->value->as.string_value, "ManagedNativeHandle") == 0) {
                managed_native_handle = 1;
                native_handle_wrapper = 1;
                break;
            }
            if (strcmp(current->value->as.string_value, "NativeHandle") == 0) {
                native_handle_wrapper = 1;
            }
            if (strcmp(current->value->as.string_value, "ArrayBuffer") == 0 ||
                strcmp(current->value->as.string_value, "SharedArrayBuffer") == 0 ||
                strcmp(current->value->as.string_value, "DataView") == 0 ||
                strcmp(current->value->as.string_value, "Uint8Array") == 0 ||
                strcmp(current->value->as.string_value, "Int8Array") == 0 ||
                strcmp(current->value->as.string_value, "Uint16Array") == 0 ||
                strcmp(current->value->as.string_value, "Int16Array") == 0 ||
                strcmp(current->value->as.string_value, "Uint32Array") == 0 ||
                strcmp(current->value->as.string_value, "Int32Array") == 0 ||
                strcmp(current->value->as.string_value, "Float32Array") == 0 ||
                strcmp(current->value->as.string_value, "Float64Array") == 0) {
                bytes_backed_object = 1;
            }
        }
        current = current->next;
    }
    if (managed_native_handle && object->native_handle != NULL) {
        jayess_managed_native_handle *managed = (jayess_managed_native_handle *)object->native_handle;
        if (managed->finalizer != NULL && managed->handle != NULL && !managed->closed) {
            managed->finalizer(managed->handle);
        }
        free(managed);
        object->native_handle = NULL;
    }
    current = object->head;
    while (current != NULL) {
        jayess_object_entry *next = current->next;
        if (current->key_value != NULL) {
            jayess_value_free_unshared(current->key_value);
        }
        if (current->value != NULL) {
            jayess_value_free_unshared(current->value);
        }
        if (jayess_runtime_accounting_state.object_entries > 0) {
            jayess_runtime_accounting_state.object_entries--;
        }
        free(current->key);
        free(current);
        current = next;
    }
    if (native_handle_wrapper && jayess_runtime_accounting_state.native_handle_wrappers > 0) {
        jayess_runtime_accounting_state.native_handle_wrappers--;
    }
    if (bytes_backed_object && object->native_handle != NULL) {
        jayess_std_buffer_state_release((jayess_buffer_state *)object->native_handle);
        object->native_handle = NULL;
    }
    if (jayess_runtime_accounting_state.objects > 0) {
        jayess_runtime_accounting_state.objects--;
    }
    free(object);
}

static char *jayess_accessor_key(const char *prefix, const char *key) {
    size_t prefix_len;
    size_t key_len;
    char *out;
    if (prefix == NULL || key == NULL) {
        return NULL;
    }
    prefix_len = strlen(prefix);
    key_len = strlen(key);
    out = (char *)malloc(prefix_len + key_len + 1);
    if (out == NULL) {
        return NULL;
    }
    memcpy(out, prefix, prefix_len);
    memcpy(out + prefix_len, key, key_len + 1);
    return out;
}

jayess_value *jayess_value_call_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *argument, int argument_count) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (argument_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else {
            result = ((jayess_callback2)fn->callee)(fn->env, argument != NULL ? argument : jayess_value_undefined());
        }
    } else if (argument_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else {
        result = ((jayess_callback1)fn->callee)(argument != NULL ? argument : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_two_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_three_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_four_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_five_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_six_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_seven_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_eight_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_nine_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback10)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else if (fn->param_count == 8) {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        } else {
            result = ((jayess_callback10)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else if (fn->param_count == 8) {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    } else {
        result = ((jayess_callback9)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_ten_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback10)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback11)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else if (fn->param_count == 8) {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        } else if (fn->param_count == 9) {
            result = ((jayess_callback10)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
        } else {
            result = ((jayess_callback11)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else if (fn->param_count == 8) {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    } else if (fn->param_count == 9) {
        result = ((jayess_callback9)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
    } else {
        result = ((jayess_callback10)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_eleven_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback10)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback11)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback12)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else if (fn->param_count == 8) {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        } else if (fn->param_count == 9) {
            result = ((jayess_callback10)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
        } else if (fn->param_count == 10) {
            result = ((jayess_callback11)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
        } else {
            result = ((jayess_callback12)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else if (fn->param_count == 8) {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    } else if (fn->param_count == 9) {
        result = ((jayess_callback9)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
    } else if (fn->param_count == 10) {
        result = ((jayess_callback10)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
    } else {
        result = ((jayess_callback11)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_twelve_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh, jayess_value *twelfth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback10)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback11)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback12)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback13)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else if (fn->param_count == 8) {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        } else if (fn->param_count == 9) {
            result = ((jayess_callback10)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
        } else if (fn->param_count == 10) {
            result = ((jayess_callback11)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
        } else if (fn->param_count == 11) {
            result = ((jayess_callback12)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
        } else {
            result = ((jayess_callback13)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else if (fn->param_count == 8) {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    } else if (fn->param_count == 9) {
        result = ((jayess_callback9)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
    } else if (fn->param_count == 10) {
        result = ((jayess_callback10)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
    } else if (fn->param_count == 11) {
        result = ((jayess_callback11)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
    } else {
        result = ((jayess_callback12)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static jayess_value *jayess_value_call_thirteen_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh, jayess_value *twelfth, jayess_value *thirteenth) {
    jayess_function *fn;
    jayess_value *result = NULL;
    jayess_value *bound_this = NULL;
    typedef jayess_value *(*jayess_callback3)(jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback4)(jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback5)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback6)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback7)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback8)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback9)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback10)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback11)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback12)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback13)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    typedef jayess_value *(*jayess_callback14)(jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *, jayess_value *);
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    if (fn->bound_this != NULL && fn->bound_this->kind != JAYESS_VALUE_UNDEFINED) {
        bound_this = fn->bound_this;
    }
    jayess_push_this(bound_this != NULL ? bound_this : (this_value != NULL ? this_value : jayess_value_undefined()));
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else if (fn->param_count == 1) {
            result = ((jayess_callback2)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined());
        } else if (fn->param_count == 2) {
            result = ((jayess_callback3)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
        } else if (fn->param_count == 3) {
            result = ((jayess_callback4)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
        } else if (fn->param_count == 4) {
            result = ((jayess_callback5)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
        } else if (fn->param_count == 5) {
            result = ((jayess_callback6)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
        } else if (fn->param_count == 6) {
            result = ((jayess_callback7)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
        } else if (fn->param_count == 7) {
            result = ((jayess_callback8)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
        } else if (fn->param_count == 8) {
            result = ((jayess_callback9)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
        } else if (fn->param_count == 9) {
            result = ((jayess_callback10)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
        } else if (fn->param_count == 10) {
            result = ((jayess_callback11)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
        } else if (fn->param_count == 11) {
            result = ((jayess_callback12)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
        } else if (fn->param_count == 12) {
            result = ((jayess_callback13)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined());
        } else {
            result = ((jayess_callback14)fn->callee)(fn->env, first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined(), thirteenth != NULL ? thirteenth : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else if (fn->param_count == 1) {
        result = ((jayess_callback1)fn->callee)(first != NULL ? first : jayess_value_undefined());
    } else if (fn->param_count == 2) {
        result = ((jayess_callback2)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined());
    } else if (fn->param_count == 3) {
        result = ((jayess_callback3)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined());
    } else if (fn->param_count == 4) {
        result = ((jayess_callback4)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined());
    } else if (fn->param_count == 5) {
        result = ((jayess_callback5)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined());
    } else if (fn->param_count == 6) {
        result = ((jayess_callback6)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined());
    } else if (fn->param_count == 7) {
        result = ((jayess_callback7)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined());
    } else if (fn->param_count == 8) {
        result = ((jayess_callback8)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined());
    } else if (fn->param_count == 9) {
        result = ((jayess_callback9)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined());
    } else if (fn->param_count == 10) {
        result = ((jayess_callback10)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined());
    } else if (fn->param_count == 11) {
        result = ((jayess_callback11)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined());
    } else if (fn->param_count == 12) {
        result = ((jayess_callback12)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined());
    } else {
        result = ((jayess_callback13)fn->callee)(first != NULL ? first : jayess_value_undefined(), second != NULL ? second : jayess_value_undefined(), third != NULL ? third : jayess_value_undefined(), fourth != NULL ? fourth : jayess_value_undefined(), fifth != NULL ? fifth : jayess_value_undefined(), sixth != NULL ? sixth : jayess_value_undefined(), seventh != NULL ? seventh : jayess_value_undefined(), eighth != NULL ? eighth : jayess_value_undefined(), ninth != NULL ? ninth : jayess_value_undefined(), tenth != NULL ? tenth : jayess_value_undefined(), eleventh != NULL ? eleventh : jayess_value_undefined(), twelfth != NULL ? twelfth : jayess_value_undefined(), thirteenth != NULL ? thirteenth : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}


int jayess_std_kind_is(jayess_value *target, const char *kind) {
    jayess_value *kind_value;
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return 0;
    }
    kind_value = jayess_object_get(target->as.object_value, "__jayess_std_kind");
    return kind_value != NULL && kind_value->kind == JAYESS_VALUE_STRING && strcmp(kind_value->as.string_value, kind) == 0;
}

void jayess_print_property_key_inline(jayess_object_entry *entry) {
    if (entry == NULL) {
        return;
    }
    if (entry->key != NULL) {
        fputs(entry->key, stdout);
        return;
    }
    if (entry->key_value != NULL) {
        putchar('[');
        jayess_print_value_inline(entry->key_value);
        putchar(']');
    }
}

static jayess_value *jayess_value_to_property_key(jayess_value *key) {
    char *text;
    jayess_value *property_key;
    if (key == NULL) {
        return NULL;
    }
    if (key->kind == JAYESS_VALUE_SYMBOL) {
        return key;
    }
    text = jayess_value_stringify(key);
    if (text == NULL) {
        return NULL;
    }
    property_key = jayess_value_from_string(text);
    free(text);
    return property_key;
}

static jayess_symbol_registry_entry *jayess_symbol_registry_find(const char *key) {
    jayess_symbol_registry_entry *current = jayess_symbol_registry;
    const char *text = key != NULL ? key : "";
    while (current != NULL) {
        if (current->key != NULL && strcmp(current->key, text) == 0) {
            return current;
        }
        current = current->next;
    }
    return NULL;
}

static jayess_value *jayess_symbol_singleton(jayess_value **slot, const char *description) {
    if (slot == NULL) {
        return jayess_value_undefined();
    }
    if (*slot == NULL) {
        *slot = jayess_value_from_symbol(description);
    }
    return *slot != NULL ? *slot : jayess_value_undefined();
}

static jayess_array *jayess_std_array_slot(jayess_value *target, const char *key) {
    jayess_value *value;
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return NULL;
    }
    value = jayess_object_get(target->as.object_value, key);
    if (value == NULL || value->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
    return value->as.array_value;
}

static int jayess_std_map_index_of(jayess_value *target, jayess_value *key) {
    jayess_array *keys = jayess_std_array_slot(target, "__jayess_map_keys");
    int i;
    if (keys == NULL) {
        return -1;
    }
    for (i = 0; i < keys->count; i++) {
        if (jayess_value_eq(keys->values[i], key)) {
            return i;
        }
    }
    return -1;
}

static int jayess_std_set_index_of(jayess_value *target, jayess_value *value) {
    jayess_array *values = jayess_std_array_slot(target, "__jayess_set_values");
    int i;
    if (values == NULL) {
        return -1;
    }
    for (i = 0; i < values->count; i++) {
        if (jayess_value_eq(values->values[i], value)) {
            return i;
        }
    }
    return -1;
}

static int jayess_std_weak_key_valid(jayess_value *key) {
    if (key == NULL) {
        return 0;
    }
    return key->kind == JAYESS_VALUE_OBJECT || key->kind == JAYESS_VALUE_FUNCTION;
}

static int jayess_std_weak_map_index_of(jayess_value *target, jayess_value *key) {
    jayess_array *keys = jayess_std_array_slot(target, "__jayess_weak_map_keys");
    int i;
    if (keys == NULL || !jayess_std_weak_key_valid(key)) {
        return -1;
    }
    for (i = 0; i < keys->count; i++) {
        if (keys->values[i] == key) {
            return i;
        }
    }
    return -1;
}

static int jayess_std_weak_set_index_of(jayess_value *target, jayess_value *value) {
    jayess_array *values = jayess_std_array_slot(target, "__jayess_weak_set_values");
    int i;
    if (values == NULL || !jayess_std_weak_key_valid(value)) {
        return -1;
    }
    for (i = 0; i < values->count; i++) {
        if (values->values[i] == value) {
            return i;
        }
    }
    return -1;
}

static void jayess_array_remove_at(jayess_array *array, int index) {
    int i;
    if (array == NULL || index < 0 || index >= array->count) {
        return;
    }
    for (i = index + 1; i < array->count; i++) {
        array->values[i - 1] = array->values[i];
    }
    array->count--;
}

static double jayess_now_ms(void) {
#ifdef _WIN32
    return (double)GetTickCount64();
#else
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (double)ts.tv_sec * 1000.0 + (double)ts.tv_nsec / 1000000.0;
#endif
}

jayess_value *jayess_std_map_new(void) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Map"));
    jayess_object_set_value(object, "__jayess_map_keys", jayess_value_from_array(jayess_array_new()));
    jayess_object_set_value(object, "__jayess_map_values", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_set_new(void) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Set"));
    jayess_object_set_value(object, "__jayess_set_values", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_weak_map_new(void) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("WeakMap"));
    jayess_object_set_value(object, "__jayess_weak_map_keys", jayess_value_from_array(jayess_array_new()));
    jayess_object_set_value(object, "__jayess_weak_map_values", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_weak_set_new(void) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("WeakSet"));
    jayess_object_set_value(object, "__jayess_weak_set_values", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_symbol(jayess_value *description) {
    char *text = NULL;
    if (description != NULL && description->kind != JAYESS_VALUE_UNDEFINED) {
        text = jayess_value_stringify(description);
    }
    return jayess_value_from_symbol(text);
}

jayess_value *jayess_std_symbol_for(jayess_value *key) {
    char *text;
    jayess_symbol_registry_entry *entry;
    if (key == NULL) {
        text = jayess_strdup("");
    } else {
        text = jayess_value_stringify(key);
    }
    if (text == NULL) {
        return jayess_value_undefined();
    }
    entry = jayess_symbol_registry_find(text);
    if (entry != NULL) {
        free(text);
        return entry->symbol != NULL ? entry->symbol : jayess_value_undefined();
    }
    entry = (jayess_symbol_registry_entry *)malloc(sizeof(jayess_symbol_registry_entry));
    if (entry == NULL) {
        free(text);
        return jayess_value_undefined();
    }
    entry->key = text;
    entry->symbol = jayess_value_from_symbol(text);
    entry->next = jayess_symbol_registry;
    jayess_symbol_registry = entry;
    return entry->symbol != NULL ? entry->symbol : jayess_value_undefined();
}

jayess_value *jayess_std_symbol_key_for(jayess_value *symbol) {
    jayess_symbol_registry_entry *current;
    if (symbol == NULL || symbol->kind != JAYESS_VALUE_SYMBOL) {
        return jayess_value_undefined();
    }
    current = jayess_symbol_registry;
    while (current != NULL) {
        if (current->symbol != NULL && jayess_value_eq(current->symbol, symbol)) {
            return jayess_value_from_string(current->key != NULL ? current->key : "");
        }
        current = current->next;
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_symbol_iterator(void) {
    return jayess_symbol_singleton(&jayess_symbol_iterator_singleton, "Symbol.iterator");
}

jayess_value *jayess_std_symbol_async_iterator(void) {
    return jayess_symbol_singleton(&jayess_symbol_async_iterator_singleton, "Symbol.asyncIterator");
}

jayess_value *jayess_std_symbol_to_string_tag(void) {
    return jayess_symbol_singleton(&jayess_symbol_to_string_tag_singleton, "Symbol.toStringTag");
}

jayess_value *jayess_std_symbol_has_instance(void) {
    return jayess_symbol_singleton(&jayess_symbol_has_instance_singleton, "Symbol.hasInstance");
}

jayess_value *jayess_std_symbol_species(void) {
    return jayess_symbol_singleton(&jayess_symbol_species_singleton, "Symbol.species");
}

jayess_value *jayess_std_symbol_match(void) {
    return jayess_symbol_singleton(&jayess_symbol_match_singleton, "Symbol.match");
}

jayess_value *jayess_std_symbol_replace(void) {
    return jayess_symbol_singleton(&jayess_symbol_replace_singleton, "Symbol.replace");
}

jayess_value *jayess_std_symbol_search(void) {
    return jayess_symbol_singleton(&jayess_symbol_search_singleton, "Symbol.search");
}

jayess_value *jayess_std_symbol_split(void) {
    return jayess_symbol_singleton(&jayess_symbol_split_singleton, "Symbol.split");
}

jayess_value *jayess_std_symbol_to_primitive(void) {
    return jayess_symbol_singleton(&jayess_symbol_to_primitive_singleton, "Symbol.toPrimitive");
}

jayess_value *jayess_std_date_new(jayess_value *value) {
    jayess_object *object = jayess_object_new();
    double ms = jayess_now_ms();
    if (value != NULL && value->kind != JAYESS_VALUE_UNDEFINED && value->kind != JAYESS_VALUE_NULL) {
        ms = jayess_value_to_number(value);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Date"));
    jayess_object_set_value(object, "__jayess_date_ms", jayess_value_from_number(ms));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_date_now(void) {
    return jayess_value_from_number(jayess_now_ms());
}

jayess_value *jayess_std_regexp_new(jayess_value *pattern, jayess_value *flags) {
    jayess_object *object = jayess_object_new();
    const char *pattern_text = "";
    const char *flags_text = "";
    if (pattern != NULL) {
        if (pattern->kind == JAYESS_VALUE_STRING && pattern->as.string_value != NULL) {
            pattern_text = pattern->as.string_value;
        }
    }
    if (flags != NULL) {
        if (flags->kind == JAYESS_VALUE_STRING && flags->as.string_value != NULL) {
            flags_text = flags->as.string_value;
        }
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("RegExp"));
    jayess_object_set_value(object, "__jayess_regexp_pattern", jayess_value_from_string(pattern_text));
    jayess_object_set_value(object, "__jayess_regexp_flags", jayess_value_from_string(flags_text));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_iterator_from(jayess_value *target) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Iterator"));
    jayess_object_set_value(object, "__jayess_iterator_values", jayess_value_iterable_values(target));
    jayess_object_set_value(object, "__jayess_iterator_index", jayess_value_from_number(0));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_async_iterator_from(jayess_value *target) {
    jayess_object *object = jayess_object_new();
    jayess_value *boxed;
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("AsyncIterator"));
    jayess_object_set_value(object, "__jayess_iterator_values", jayess_value_iterable_values(target));
    jayess_object_set_value(object, "__jayess_iterator_index", jayess_value_from_number(0));
    boxed = jayess_value_from_object(object);
    jayess_object_set_key_value(object, jayess_std_symbol_async_iterator(), jayess_value_from_function((void *)jayess_std_async_iterator_identity_method, boxed, "[Symbol.asyncIterator]", NULL, 0, 0));
    return boxed;
}

jayess_value *jayess_std_promise_resolve(jayess_value *value) {
    jayess_object *object = jayess_object_new();
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Promise")) {
        return value;
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Promise"));
    jayess_object_set_value(object, "__jayess_promise_state", jayess_value_from_string("fulfilled"));
    jayess_object_set_value(object, "__jayess_promise_value", value != NULL ? value : jayess_value_undefined());
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_promise_reject(jayess_value *reason) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Promise"));
    jayess_object_set_value(object, "__jayess_promise_state", jayess_value_from_string("rejected"));
    jayess_object_set_value(object, "__jayess_promise_value", reason != NULL ? reason : jayess_value_undefined());
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_promise_all(jayess_value *values) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_promise_task(values, promise, jayess_value_undefined(), jayess_value_undefined(), JAYESS_PROMISE_ACTION_ALL);
    return promise;
}

jayess_value *jayess_std_promise_race(jayess_value *values) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_promise_task(values, promise, jayess_value_undefined(), jayess_value_undefined(), JAYESS_PROMISE_ACTION_RACE);
    return promise;
}

jayess_value *jayess_std_promise_all_settled(jayess_value *values) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_promise_task(values, promise, jayess_value_undefined(), jayess_value_undefined(), JAYESS_PROMISE_ACTION_ALL_SETTLED);
    return promise;
}

jayess_value *jayess_std_promise_any(jayess_value *values) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_promise_task(values, promise, jayess_value_undefined(), jayess_value_undefined(), JAYESS_PROMISE_ACTION_ANY);
    return promise;
}

static jayess_value *jayess_std_promise_pending(void) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Promise"));
    jayess_object_set_value(object, "__jayess_promise_state", jayess_value_from_string("pending"));
    jayess_object_set_value(object, "__jayess_promise_value", jayess_value_undefined());
    return jayess_value_from_object(object);
}

static int jayess_promise_is_state(jayess_value *value, const char *state) {
    jayess_value *stored;
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(value, "Promise")) {
        return 0;
    }
    stored = jayess_object_get(value->as.object_value, "__jayess_promise_state");
    return stored != NULL && stored->kind == JAYESS_VALUE_STRING && stored->as.string_value != NULL && strcmp(stored->as.string_value, state) == 0;
}

static jayess_value *jayess_promise_value(jayess_value *value) {
    jayess_value *stored;
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(value, "Promise")) {
        return jayess_value_undefined();
    }
    stored = jayess_object_get(value->as.object_value, "__jayess_promise_value");
    return stored != NULL ? stored : jayess_value_undefined();
}

static int jayess_value_is_string(jayess_value *value, const char *text) {
    return value != NULL && value->kind == JAYESS_VALUE_STRING && value->as.string_value != NULL && text != NULL && strcmp(value->as.string_value, text) == 0;
}

static void jayess_promise_settle(jayess_value *promise, const char *state, jayess_value *value) {
    jayess_promise_dependent *dependent;
    if (promise == NULL || promise->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(promise, "Promise")) {
        return;
    }
    jayess_object_set_value(promise->as.object_value, "__jayess_promise_state", jayess_value_from_string(state));
    jayess_object_set_value(promise->as.object_value, "__jayess_promise_value", value != NULL ? value : jayess_value_undefined());
    dependent = promise->as.object_value->promise_dependents;
    promise->as.object_value->promise_dependents = NULL;
    while (dependent != NULL) {
        jayess_promise_dependent *next = dependent->next;
        jayess_microtask *task = dependent->task;
        if (task != NULL && task->dependency_count > 0) {
            task->dependency_count--;
        }
        if (task != NULL && task->finished) {
            if (task->dependency_count <= 0) {
                free(task);
            }
        } else if (task != NULL && !task->queued) {
            jayess_append_microtask(task);
        }
        free(dependent);
        dependent = next;
    }
}

static void jayess_io_worker_run_task(jayess_microtask *task) {
    if (task == NULL) {
        return;
    }
    if (task->kind == JAYESS_TASK_FS_READ) {
        task->worker_result = jayess_std_fs_read_file(task->path, task->encoding);
    } else if (task->kind == JAYESS_TASK_FS_WRITE) {
        task->worker_result = jayess_std_fs_write_file(task->path, task->content);
    } else if (task->kind == JAYESS_TASK_SOCKET_READ) {
        int requested = jayess_std_stream_requested_size(task->path, 4095);
        char *buffer;
        int read_count;
        int did_timeout = 0;
        if (task->socket_handle == JAYESS_INVALID_SOCKET) {
            task->worker_result = jayess_value_undefined();
            task->worker_emit_close = 1;
        } else {
            buffer = (char *)malloc((size_t)requested + 1);
            if (buffer == NULL) {
                task->worker_result = jayess_value_undefined();
            } else {
                #ifdef _WIN32
                if (jayess_std_tls_state(task->source) != NULL) {
                    read_count = jayess_std_tls_read_bytes(task->source, (unsigned char *)buffer, requested, &did_timeout);
                } else {
                    read_count = (int)recv(task->socket_handle, buffer, requested, 0);
                }
                #else
                read_count = (int)recv(task->socket_handle, buffer, requested, 0);
                #endif
                if (read_count <= 0) {
                    free(buffer);
                    task->worker_result = read_count == 0 ? jayess_value_null() : jayess_value_undefined();
                    task->worker_emit_error = read_count < 0 && !did_timeout;
                    task->worker_emit_close = 1;
                    jayess_std_socket_close_handle(task->socket_handle);
                } else {
                    buffer[read_count] = '\0';
                    task->worker_bytes = read_count;
                    task->worker_result = jayess_value_from_string(buffer);
                    free(buffer);
                }
            }
        }
    } else if (task->kind == JAYESS_TASK_SOCKET_WRITE) {
        if (task->socket_handle == JAYESS_INVALID_SOCKET) {
            task->worker_result = jayess_value_from_bool(0);
            task->worker_emit_close = 1;
        } else if (task->content != NULL && task->content->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(task->content, "Uint8Array")) {
            jayess_array *bytes = jayess_std_bytes_slot(task->content);
            int offset = 0;
            int ok = 1;
            if (bytes == NULL) {
                task->worker_result = jayess_value_from_bool(0);
            } else {
                while (offset < bytes->count) {
                    unsigned char chunk[1024];
                    int chunk_len = bytes->count - offset;
                    int i;
                    int sent;
                    if (chunk_len > (int)sizeof(chunk)) {
                        chunk_len = (int)sizeof(chunk);
                    }
                    for (i = 0; i < chunk_len; i++) {
                        chunk[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, offset + i)) & 255);
                    }
                    #ifdef _WIN32
                    if (jayess_std_tls_state(task->source) != NULL) {
                        sent = jayess_std_tls_write_bytes(task->source, chunk, chunk_len, NULL);
                    } else {
                        sent = (int)send(task->socket_handle, (const char *)chunk, chunk_len, 0);
                    }
                    #else
                    sent = (int)send(task->socket_handle, (const char *)chunk, chunk_len, 0);
                    #endif
                    if (sent <= 0) {
                        ok = 0;
                        task->worker_emit_error = 1;
                        break;
                    }
                    task->worker_bytes += sent;
                    offset += sent;
                }
                task->worker_result = jayess_value_from_bool(ok);
            }
        } else {
            char *text = jayess_value_stringify(task->content);
            size_t length;
            size_t offset = 0;
            int ok = 1;
            if (text == NULL) {
                task->worker_result = jayess_value_from_bool(0);
            } else {
                length = strlen(text);
                while (offset < length) {
                    int sent;
                    #ifdef _WIN32
                    if (jayess_std_tls_state(task->source) != NULL) {
                        sent = jayess_std_tls_write_bytes(task->source, (const unsigned char *)text + offset, (int)(length - offset), NULL);
                    } else {
                        sent = (int)send(task->socket_handle, text + offset, (int)(length - offset), 0);
                    }
                    #else
                    sent = (int)send(task->socket_handle, text + offset, (int)(length - offset), 0);
                    #endif
                    if (sent <= 0) {
                        ok = 0;
                        task->worker_emit_error = 1;
                        break;
                    }
                    task->worker_bytes += sent;
                    offset += (size_t)sent;
                }
                free(text);
                task->worker_result = jayess_value_from_bool(ok);
            }
        }
    } else if (task->kind == JAYESS_TASK_SERVER_ACCEPT) {
        struct sockaddr_storage client_addr;
#ifdef _WIN32
        int client_len = sizeof(client_addr);
#else
        socklen_t client_len = sizeof(client_addr);
#endif
        jayess_socket_handle client_handle;
        char address[INET6_ADDRSTRLEN];
        int port = 0;
        void *addr_ptr = NULL;
        if (task->socket_handle == JAYESS_INVALID_SOCKET) {
            task->worker_result = jayess_value_undefined();
        } else {
            memset(&client_addr, 0, sizeof(client_addr));
            client_handle = accept(task->socket_handle, (struct sockaddr *)&client_addr, &client_len);
            if (client_handle == JAYESS_INVALID_SOCKET) {
                task->worker_result = jayess_value_undefined();
                task->worker_emit_error = 1;
            } else {
                address[0] = '\0';
                if (client_addr.ss_family == AF_INET) {
                    struct sockaddr_in *ipv4 = (struct sockaddr_in *)&client_addr;
                    addr_ptr = &(ipv4->sin_addr);
                    port = ntohs(ipv4->sin_port);
                } else if (client_addr.ss_family == AF_INET6) {
                    struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)&client_addr;
                    addr_ptr = &(ipv6->sin6_addr);
                    port = ntohs(ipv6->sin6_port);
                }
                if (addr_ptr == NULL || inet_ntop(client_addr.ss_family, addr_ptr, address, sizeof(address)) == NULL) {
                    jayess_std_socket_close_handle(client_handle);
                    task->worker_result = jayess_value_undefined();
                } else {
                    task->worker_result = jayess_std_socket_value_from_handle(client_handle, address, port);
                    jayess_std_socket_set_remote_family(task->worker_result, client_addr.ss_family == AF_INET6 ? 6 : 4);
                    jayess_std_socket_set_local_endpoint(task->worker_result, client_handle);
                }
            }
        }
    } else if (task->kind == JAYESS_TASK_HTTP_REQUEST) {
        task->worker_result = jayess_std_http_request(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTP_GET) {
        task->worker_result = jayess_std_http_get(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTP_REQUEST_STREAM) {
        task->worker_result = jayess_std_http_request_stream(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTP_GET_STREAM) {
        task->worker_result = jayess_std_http_get_stream(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTPS_REQUEST) {
        task->worker_result = jayess_std_https_request(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTPS_GET) {
        task->worker_result = jayess_std_https_get(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTPS_REQUEST_STREAM) {
        task->worker_result = jayess_std_https_request_stream(task->content != NULL ? task->content : jayess_value_undefined());
    } else if (task->kind == JAYESS_TASK_HTTPS_GET_STREAM) {
        task->worker_result = jayess_std_https_get_stream(task->content != NULL ? task->content : jayess_value_undefined());
    } else {
        task->worker_result = jayess_type_error_value("unknown async I/O task");
    }
    task->completed = 1;
}

static void jayess_io_pool_push(jayess_microtask *task) {
    jayess_io_worker_pool *pool = &jayess_runtime_io_pool;
    if (task == NULL) {
        return;
    }
    task->worker_next = NULL;
#ifdef _WIN32
    EnterCriticalSection(&pool->lock);
    if (pool->tail != NULL) {
        pool->tail->worker_next = task;
    } else {
        pool->head = task;
    }
    pool->tail = task;
    WakeConditionVariable(&pool->available);
    LeaveCriticalSection(&pool->lock);
#else
    pthread_mutex_lock(&pool->lock);
    if (pool->tail != NULL) {
        pool->tail->worker_next = task;
    } else {
        pool->head = task;
    }
    pool->tail = task;
    pthread_cond_signal(&pool->available);
    pthread_mutex_unlock(&pool->lock);
#endif
}

static jayess_microtask *jayess_io_pool_pop(void) {
    jayess_io_worker_pool *pool = &jayess_runtime_io_pool;
    jayess_microtask *task;
#ifdef _WIN32
    EnterCriticalSection(&pool->lock);
    while (pool->head == NULL && !pool->stopping) {
        SleepConditionVariableCS(&pool->available, &pool->lock, INFINITE);
    }
    if (pool->head == NULL && pool->stopping) {
        LeaveCriticalSection(&pool->lock);
        return NULL;
    }
    task = pool->head;
    pool->head = task->worker_next;
    if (pool->head == NULL) {
        pool->tail = NULL;
    }
    task->worker_next = NULL;
    LeaveCriticalSection(&pool->lock);
#else
    pthread_mutex_lock(&pool->lock);
    while (pool->head == NULL && !pool->stopping) {
        pthread_cond_wait(&pool->available, &pool->lock);
    }
    if (pool->head == NULL && pool->stopping) {
        pthread_mutex_unlock(&pool->lock);
        return NULL;
    }
    task = pool->head;
    pool->head = task->worker_next;
    if (pool->head == NULL) {
        pool->tail = NULL;
    }
    task->worker_next = NULL;
    pthread_mutex_unlock(&pool->lock);
#endif
    return task;
}

#ifdef _WIN32
static DWORD WINAPI jayess_io_worker_main(LPVOID raw) {
    (void)raw;
    for (;;) {
        jayess_microtask *task = jayess_io_pool_pop();
        if (task == NULL) {
            break;
        }
        jayess_io_worker_run_task(task);
    }
    return 0;
}
#else
static void *jayess_io_worker_main(void *raw) {
    (void)raw;
    for (;;) {
        jayess_microtask *task = jayess_io_pool_pop();
        if (task == NULL) {
            break;
        }
        jayess_io_worker_run_task(task);
    }
    return NULL;
}
#endif

static int jayess_io_pool_start(void) {
    jayess_io_worker_pool *pool = &jayess_runtime_io_pool;
    int i;
    if (pool->started) {
        return 1;
    }
    pool->stopping = 0;
    pool->worker_count = 0;
#ifdef _WIN32
    InitializeCriticalSection(&pool->lock);
    InitializeConditionVariable(&pool->available);
    pool->started = 1;
    for (i = 0; i < JAYESS_IO_WORKER_COUNT; i++) {
        pool->workers[i] = CreateThread(NULL, 0, jayess_io_worker_main, NULL, 0, NULL);
        if (pool->workers[i] == NULL) {
            if (i == 0) {
                DeleteCriticalSection(&pool->lock);
                pool->started = 0;
            }
            return i > 0;
        }
        pool->worker_count++;
    }
#else
    if (pthread_mutex_init(&pool->lock, NULL) != 0) {
        return 0;
    }
    if (pthread_cond_init(&pool->available, NULL) != 0) {
        pthread_mutex_destroy(&pool->lock);
        return 0;
    }
    pool->started = 1;
    for (i = 0; i < JAYESS_IO_WORKER_COUNT; i++) {
        if (pthread_create(&pool->workers[i], NULL, jayess_io_worker_main, NULL) != 0) {
            if (i == 0) {
                pthread_cond_destroy(&pool->available);
                pthread_mutex_destroy(&pool->lock);
                pool->started = 0;
            }
            return i > 0;
        }
        pool->worker_count++;
    }
#endif
    return 1;
}

static void jayess_io_pool_shutdown(void) {
    jayess_io_worker_pool *pool = &jayess_runtime_io_pool;
    int i;
    if (!pool->started) {
        return;
    }
#ifdef _WIN32
    EnterCriticalSection(&pool->lock);
    pool->stopping = 1;
    WakeAllConditionVariable(&pool->available);
    LeaveCriticalSection(&pool->lock);
    for (i = 0; i < pool->worker_count; i++) {
        if (pool->workers[i] != NULL) {
            WaitForSingleObject(pool->workers[i], INFINITE);
            CloseHandle(pool->workers[i]);
            pool->workers[i] = NULL;
        }
    }
    DeleteCriticalSection(&pool->lock);
#else
    pthread_mutex_lock(&pool->lock);
    pool->stopping = 1;
    pthread_cond_broadcast(&pool->available);
    pthread_mutex_unlock(&pool->lock);
    for (i = 0; i < pool->worker_count; i++) {
        pthread_join(pool->workers[i], NULL);
    }
    pthread_cond_destroy(&pool->available);
    pthread_mutex_destroy(&pool->lock);
#endif
    pool->head = NULL;
    pool->tail = NULL;
    pool->worker_count = 0;
    pool->started = 0;
    pool->stopping = 0;
}

static jayess_task_queue *jayess_scheduler_queue_for(jayess_microtask *task) {
    if (task != NULL && task->kind == JAYESS_TASK_PROMISE_CALLBACK) {
        return &jayess_runtime_scheduler.promise_callbacks;
    }
    if (task != NULL && task->kind == JAYESS_TASK_TIMER) {
        return &jayess_runtime_scheduler.timers;
    }
    return &jayess_runtime_scheduler.io_pending;
}

static int jayess_scheduler_has_tasks(void) {
    return jayess_runtime_scheduler.promise_callbacks.head != NULL || jayess_runtime_scheduler.timers.head != NULL || jayess_runtime_scheduler.io_pending.head != NULL || jayess_runtime_scheduler.io_completions.head != NULL;
}

static void jayess_queue_append(jayess_task_queue *queue, jayess_microtask *task) {
    if (queue == NULL || task == NULL) {
        return;
    }
    task->next = NULL;
    task->queued = 1;
    if (queue->tail != NULL) {
        queue->tail->next = task;
    } else {
        queue->head = task;
    }
    queue->tail = task;
}

static jayess_microtask *jayess_queue_pop_head(jayess_task_queue *queue) {
    jayess_microtask *task;
    if (queue == NULL || queue->head == NULL) {
        return NULL;
    }
    task = queue->head;
    queue->head = task->next;
    if (queue->head == NULL) {
        queue->tail = NULL;
    }
    task->next = NULL;
    task->queued = 0;
    return task;
}

static int jayess_promise_combinator_ready(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    int i;
    int saw_pending = 0;

    if (task == NULL) {
        return 0;
    }
    if (task->promise_action == JAYESS_PROMISE_ACTION_THEN || task->promise_action == JAYESS_PROMISE_ACTION_FINALLY) {
        return !jayess_promise_is_state(task->source, "pending");
    }

    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        return 1;
    }
    items = items_value->as.array_value;

    if (task->promise_action == JAYESS_PROMISE_ACTION_RACE) {
        if (items->count == 0) {
            return 1;
        }
        for (i = 0; i < items->count; i++) {
            jayess_value *item = items->values[i];
            if (item == NULL || item->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(item, "Promise")) {
                return 1;
            }
            if (!jayess_promise_is_state(item, "pending")) {
                return 1;
            }
        }
        return 0;
    }

    if (task->promise_action == JAYESS_PROMISE_ACTION_ANY) {
        if (items->count == 0) {
            return 1;
        }
        for (i = 0; i < items->count; i++) {
            jayess_value *item = items->values[i];
            if (item == NULL || item->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(item, "Promise")) {
                return 1;
            }
            if (jayess_promise_is_state(item, "fulfilled")) {
                return 1;
            }
            if (jayess_promise_is_state(item, "pending")) {
                saw_pending = 1;
            }
        }
        return !saw_pending;
    }

    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_is_state(item, "rejected") && task->promise_action == JAYESS_PROMISE_ACTION_ALL) {
                return 1;
            }
            if (jayess_promise_is_state(item, "pending")) {
                return 0;
            }
        }
    }
    return 1;
}

static jayess_microtask *jayess_queue_remove_ready_promise_callback(jayess_task_queue *queue) {
    jayess_microtask *previous = NULL;
    jayess_microtask *current = queue != NULL ? queue->head : NULL;
    while (current != NULL) {
        if (jayess_promise_combinator_ready(current)) {
            if (previous != NULL) {
                previous->next = current->next;
            } else {
                queue->head = current->next;
            }
            if (queue->tail == current) {
                queue->tail = previous;
            }
            current->next = NULL;
            current->queued = 0;
            return current;
        }
        previous = current;
        current = current->next;
    }
    return NULL;
}

static jayess_microtask *jayess_queue_remove_due_timer(jayess_task_queue *queue, double now_ms) {
    jayess_microtask *previous = NULL;
    jayess_microtask *current = queue != NULL ? queue->head : NULL;
    while (current != NULL) {
        if (current->due_ms <= now_ms) {
            if (previous != NULL) {
                previous->next = current->next;
            } else {
                queue->head = current->next;
            }
            if (queue->tail == current) {
                queue->tail = previous;
            }
            current->next = NULL;
            current->queued = 0;
            return current;
        }
        previous = current;
        current = current->next;
    }
    return NULL;
}

static int jayess_scheduler_next_timer_delay_ms(void) {
    jayess_microtask *current = jayess_runtime_scheduler.timers.head;
    double now = jayess_now_ms();
    double best = -1;
    while (current != NULL) {
        if (best < 0 || current->due_ms < best) {
            best = current->due_ms;
        }
        current = current->next;
    }
    if (best < 0) {
        return 1;
    }
    if (best <= now) {
        return 0;
    }
    {
        int delay = (int)(best - now);
        if (delay < 1) {
            return 1;
        }
        if (delay > 50) {
            return 50;
        }
        return delay;
    }
}

static void jayess_scheduler_promote_completed_io(void) {
    jayess_task_queue *pending = &jayess_runtime_scheduler.io_pending;
    jayess_task_queue *completions = &jayess_runtime_scheduler.io_completions;
    jayess_microtask *previous = NULL;
    jayess_microtask *current = pending->head;
    while (current != NULL) {
        jayess_microtask *next = current->next;
        if (current->completed) {
            if (previous != NULL) {
                previous->next = next;
            } else {
                pending->head = next;
            }
            if (pending->tail == current) {
                pending->tail = previous;
            }
            current->next = NULL;
            current->queued = 0;
            jayess_queue_append(completions, current);
        } else {
            previous = current;
        }
        current = next;
    }
}

static void jayess_append_microtask(jayess_microtask *task) {
    jayess_queue_append(jayess_scheduler_queue_for(task), task);
}

static int jayess_promise_add_dependent(jayess_value *promise, jayess_microtask *task) {
    jayess_promise_dependent *dependent;
    if (promise == NULL || promise->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(promise, "Promise") || task == NULL) {
        return 0;
    }
    if (!jayess_promise_is_state(promise, "pending")) {
        return 0;
    }
    dependent = (jayess_promise_dependent *)malloc(sizeof(jayess_promise_dependent));
    if (dependent == NULL) {
        return 0;
    }
    dependent->task = task;
    dependent->next = promise->as.object_value->promise_dependents;
    promise->as.object_value->promise_dependents = dependent;
    task->dependency_count++;
    task->queued = 0;
    return 1;
}

static int jayess_promise_add_combinator_dependents(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    int i;
    int added = 0;
    if (task == NULL) {
        return 0;
    }
    if (jayess_promise_combinator_ready(task)) {
        return 0;
    }
    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        return 0;
    }
    items = items_value->as.array_value;
    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_add_dependent(item, task)) {
                added = 1;
            }
        }
    }
    return added;
}

static void jayess_enqueue_microtask(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected) {
    jayess_enqueue_promise_task(source, result, on_fulfilled, on_rejected, JAYESS_PROMISE_ACTION_THEN);
}

static void jayess_enqueue_promise_task(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected, jayess_promise_action action) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue promise callback"));
        return;
    }
    task->kind = JAYESS_TASK_PROMISE_CALLBACK;
    task->promise_action = action;
    task->completed = 0;
    task->source = source;
    task->result = result;
    task->on_fulfilled = on_fulfilled;
    task->on_rejected = on_rejected;
    task->path = NULL;
    task->encoding = NULL;
    task->content = NULL;
    task->worker_result = NULL;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if ((action == JAYESS_PROMISE_ACTION_THEN || action == JAYESS_PROMISE_ACTION_FINALLY) && jayess_promise_add_dependent(source, task)) {
        return;
    }
    if (action != JAYESS_PROMISE_ACTION_THEN && action != JAYESS_PROMISE_ACTION_FINALLY && jayess_promise_add_combinator_dependents(task)) {
        return;
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_fs_read_file_task(jayess_value *result, jayess_value *path, jayess_value *encoding) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue file read"));
        return;
    }
    task->kind = JAYESS_TASK_FS_READ;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = NULL;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = path;
    task->encoding = encoding;
    task->content = NULL;
    task->worker_result = NULL;
    task->socket_handle = JAYESS_INVALID_SOCKET;
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_undefined();
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_fs_write_file_task(jayess_value *result, jayess_value *path, jayess_value *content) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue file write"));
        return;
    }
    task->kind = JAYESS_TASK_FS_WRITE;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = NULL;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = path;
    task->encoding = NULL;
    task->content = content;
    task->worker_result = NULL;
    task->socket_handle = JAYESS_INVALID_SOCKET;
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_from_bool(0);
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_timer_task(jayess_value *callback, int delay_ms) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_throw(jayess_type_error_value("failed to enqueue timer"));
        return;
    }
    task->kind = JAYESS_TASK_TIMER;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = NULL;
    task->result = NULL;
    task->on_fulfilled = callback;
    task->on_rejected = NULL;
    task->path = NULL;
    task->encoding = NULL;
    task->content = NULL;
    task->worker_result = NULL;
    task->socket_handle = JAYESS_INVALID_SOCKET;
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = jayess_now_ms() + (delay_ms > 0 ? delay_ms : 0);
    task->timer_id = jayess_next_timer_id++;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    jayess_append_microtask(task);
}

static void jayess_enqueue_sleep_async_task(jayess_value *result, int delay_ms, jayess_value *value) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue async sleep"));
        return;
    }
    task->kind = JAYESS_TASK_TIMER;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = NULL;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = NULL;
    task->encoding = NULL;
    task->content = NULL;
    task->worker_result = value != NULL ? value : jayess_value_undefined();
    task->socket_handle = JAYESS_INVALID_SOCKET;
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = jayess_now_ms() + (delay_ms > 0 ? delay_ms : 0);
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    jayess_append_microtask(task);
}

static void jayess_enqueue_socket_read_task(jayess_value *result, jayess_value *socket, jayess_value *size_value) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue socket read"));
        return;
    }
    task->kind = JAYESS_TASK_SOCKET_READ;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = socket;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = size_value;
    task->encoding = NULL;
    task->content = NULL;
    task->worker_result = NULL;
    task->socket_handle = jayess_std_socket_handle(socket);
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_undefined();
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_socket_write_task(jayess_value *result, jayess_value *socket, jayess_value *value) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue socket write"));
        return;
    }
    task->kind = JAYESS_TASK_SOCKET_WRITE;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = socket;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = NULL;
    task->encoding = NULL;
    task->content = value;
    task->worker_result = NULL;
    task->socket_handle = jayess_std_socket_handle(socket);
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_from_bool(0);
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_server_accept_task(jayess_value *result, jayess_value *server) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue server accept"));
        return;
    }
    task->kind = JAYESS_TASK_SERVER_ACCEPT;
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = server;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = NULL;
    task->encoding = NULL;
    task->content = NULL;
    task->worker_result = NULL;
    task->socket_handle = jayess_std_socket_handle(server);
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_undefined();
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_enqueue_http_request_task(jayess_value *result, jayess_value *options, int is_get, int is_https, int is_stream) {
    jayess_microtask *task = (jayess_microtask *)malloc(sizeof(jayess_microtask));
    if (task == NULL) {
        jayess_promise_settle(result, "rejected", jayess_type_error_value("failed to enqueue http request"));
        return;
    }
    if (is_https) {
        if (is_stream) {
            task->kind = is_get ? JAYESS_TASK_HTTPS_GET_STREAM : JAYESS_TASK_HTTPS_REQUEST_STREAM;
        } else {
            task->kind = is_get ? JAYESS_TASK_HTTPS_GET : JAYESS_TASK_HTTPS_REQUEST;
        }
    } else {
        if (is_stream) {
            task->kind = is_get ? JAYESS_TASK_HTTP_GET_STREAM : JAYESS_TASK_HTTP_REQUEST_STREAM;
        } else {
            task->kind = is_get ? JAYESS_TASK_HTTP_GET : JAYESS_TASK_HTTP_REQUEST;
        }
    }
    task->promise_action = JAYESS_PROMISE_ACTION_THEN;
    task->completed = 0;
    task->source = NULL;
    task->result = result;
    task->on_fulfilled = NULL;
    task->on_rejected = NULL;
    task->path = NULL;
    task->encoding = NULL;
    task->content = options;
    task->worker_result = NULL;
    task->socket_handle = JAYESS_INVALID_SOCKET;
    task->worker_bytes = 0;
    task->worker_emit_error = 0;
    task->worker_emit_close = 0;
    task->due_ms = 0;
    task->timer_id = 0;
    task->worker_next = NULL;
    task->dependency_count = 0;
    task->queued = 0;
    task->finished = 0;
    if (!jayess_io_pool_start()) {
        task->worker_result = jayess_value_undefined();
        task->completed = 1;
    } else {
        jayess_io_pool_push(task);
    }
    jayess_append_microtask(task);
}

static void jayess_requeue_microtask(jayess_microtask *task) {
    if (task == NULL) {
        return;
    }
    jayess_append_microtask(task);
}

static jayess_microtask *jayess_dequeue_microtask(void) {
    jayess_microtask *task = jayess_queue_remove_ready_promise_callback(&jayess_runtime_scheduler.promise_callbacks);
    if (task != NULL) {
        return task;
    }
    jayess_scheduler_promote_completed_io();
    task = jayess_queue_pop_head(&jayess_runtime_scheduler.io_completions);
    if (task != NULL) {
        return task;
    }
    task = jayess_queue_remove_due_timer(&jayess_runtime_scheduler.timers, jayess_now_ms());
    if (task != NULL) {
        return task;
    }
    return jayess_queue_pop_head(&jayess_runtime_scheduler.promise_callbacks);
}

static int jayess_run_promise_all_task(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    jayess_array *resolved;
    int i;

    if (task == NULL) {
        return 1;
    }
    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise.all expects an iterable value"));
        return 1;
    }
    items = items_value->as.array_value;
    resolved = jayess_array_new();
    if (resolved == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("failed to allocate Promise.all result"));
        return 1;
    }
    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_is_state(item, "pending")) {
                if (!jayess_promise_add_combinator_dependents(task)) {
                    jayess_requeue_microtask(task);
                }
                return 0;
            }
            if (jayess_promise_is_state(item, "rejected")) {
                jayess_promise_settle(task->result, "rejected", jayess_promise_value(item));
                return 1;
            }
            jayess_array_push_value(resolved, jayess_promise_value(item));
        } else {
            jayess_array_push_value(resolved, item != NULL ? item : jayess_value_undefined());
        }
    }
    jayess_promise_settle(task->result, "fulfilled", jayess_value_from_array(resolved));
    return 1;
}

static int jayess_run_promise_race_task(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    int i;
    int saw_pending = 0;

    if (task == NULL) {
        return 1;
    }
    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise.race expects an iterable value"));
        return 1;
    }
    items = items_value->as.array_value;
    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_is_state(item, "pending")) {
                saw_pending = 1;
                continue;
            }
            jayess_promise_settle(task->result, jayess_promise_is_state(item, "rejected") ? "rejected" : "fulfilled", jayess_promise_value(item));
            return 1;
        }
        jayess_promise_settle(task->result, "fulfilled", item != NULL ? item : jayess_value_undefined());
        return 1;
    }
    if (saw_pending) {
        if (!jayess_promise_add_combinator_dependents(task)) {
            jayess_requeue_microtask(task);
        }
        return 0;
    }
    jayess_promise_settle(task->result, "fulfilled", jayess_value_undefined());
    return 1;
}

static jayess_value *jayess_promise_settled_record(const char *status, const char *slot, jayess_value *value) {
    jayess_object *record = jayess_object_new();
    if (record == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(record, "status", jayess_value_from_string(status));
    jayess_object_set_value(record, slot, value != NULL ? value : jayess_value_undefined());
    return jayess_value_from_object(record);
}

static int jayess_run_promise_all_settled_task(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    jayess_array *settled;
    int i;

    if (task == NULL) {
        return 1;
    }
    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise.allSettled expects an iterable value"));
        return 1;
    }
    items = items_value->as.array_value;
    settled = jayess_array_new();
    if (settled == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("failed to allocate Promise.allSettled result"));
        return 1;
    }
    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_is_state(item, "pending")) {
                if (!jayess_promise_add_combinator_dependents(task)) {
                    jayess_requeue_microtask(task);
                }
                return 0;
            }
            if (jayess_promise_is_state(item, "rejected")) {
                jayess_array_push_value(settled, jayess_promise_settled_record("rejected", "reason", jayess_promise_value(item)));
            } else {
                jayess_array_push_value(settled, jayess_promise_settled_record("fulfilled", "value", jayess_promise_value(item)));
            }
        } else {
            jayess_array_push_value(settled, jayess_promise_settled_record("fulfilled", "value", item != NULL ? item : jayess_value_undefined()));
        }
    }
    jayess_promise_settle(task->result, "fulfilled", jayess_value_from_array(settled));
    return 1;
}

static int jayess_run_promise_any_task(jayess_microtask *task) {
    jayess_value *items_value;
    jayess_array *items;
    jayess_array *errors;
    int i;

    if (task == NULL) {
        return 1;
    }
    items_value = jayess_value_iterable_values(task->source);
    if (items_value == NULL || items_value->kind != JAYESS_VALUE_ARRAY || items_value->as.array_value == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise.any expects an iterable value"));
        return 1;
    }
    items = items_value->as.array_value;
    errors = jayess_array_new();
    if (errors == NULL) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("failed to allocate Promise.any errors"));
        return 1;
    }
    for (i = 0; i < items->count; i++) {
        jayess_value *item = items->values[i];
        if (item != NULL && item->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(item, "Promise")) {
            if (jayess_promise_is_state(item, "pending")) {
                if (!jayess_promise_add_combinator_dependents(task)) {
                    jayess_requeue_microtask(task);
                }
                return 0;
            }
            if (jayess_promise_is_state(item, "fulfilled")) {
                jayess_promise_settle(task->result, "fulfilled", jayess_promise_value(item));
                return 1;
            }
            jayess_array_push_value(errors, jayess_promise_value(item));
        } else {
            jayess_promise_settle(task->result, "fulfilled", item != NULL ? item : jayess_value_undefined());
            return 1;
        }
    }
    jayess_promise_settle(task->result, "rejected", jayess_std_aggregate_error_new(jayess_value_from_array(errors), jayess_value_from_string("All promises were rejected")));
    return 1;
}

jayess_value *jayess_value_call_one(jayess_value *callback, jayess_value *argument) {
    jayess_function *fn;
    jayess_value *this_value;
    jayess_value *result = NULL;
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION || callback->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    fn = callback->as.function_value;
    if (fn->callee == NULL) {
        return jayess_value_undefined();
    }
    this_value = fn->bound_this != NULL ? fn->bound_this : jayess_value_undefined();
    jayess_push_this(this_value);
    if (fn->env != NULL) {
        if (fn->param_count <= 0) {
            result = ((jayess_callback1)fn->callee)(fn->env);
        } else {
            result = ((jayess_callback2)fn->callee)(fn->env, argument != NULL ? argument : jayess_value_undefined());
        }
    } else if (fn->param_count <= 0) {
        result = ((jayess_callback0)fn->callee)();
    } else {
        result = ((jayess_callback1)fn->callee)(argument != NULL ? argument : jayess_value_undefined());
    }
    jayess_pop_this();
    return result != NULL ? result : jayess_value_undefined();
}

static void jayess_join_async_worker_task(jayess_microtask *task) {
    (void)task;
}

static int jayess_run_async_worker_task(jayess_microtask *task) {
    if (task == NULL) {
        return 1;
    }
    if (!task->completed) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("internal async I/O task ran before completion"));
        return 1;
    }
    jayess_join_async_worker_task(task);
    if (task->kind == JAYESS_TASK_SOCKET_READ && task->worker_bytes > 0 && task->source != NULL && task->source->kind == JAYESS_VALUE_OBJECT && task->source->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(task->source->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)task->worker_bytes;
        jayess_object_set_value(task->source->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    if (task->kind == JAYESS_TASK_SOCKET_WRITE && task->worker_bytes > 0 && task->source != NULL && task->source->kind == JAYESS_VALUE_OBJECT && task->source->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(task->source->as.object_value, "bytesWritten");
        double total = jayess_value_to_number(current) + (double)task->worker_bytes;
        jayess_object_set_value(task->source->as.object_value, "bytesWritten", jayess_value_from_number(total));
    }
    if (task->kind == JAYESS_TASK_SERVER_ACCEPT && task->worker_result != NULL && task->worker_result->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(task->worker_result, "Socket") && task->source != NULL && task->source->kind == JAYESS_VALUE_OBJECT && task->source->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(task->source->as.object_value, "connectionsAccepted");
        double total = jayess_value_to_number(current) + 1.0;
        jayess_object_set_value(task->source->as.object_value, "connectionsAccepted", jayess_value_from_number(total));
        jayess_std_stream_emit(task->source, "connection", task->worker_result);
    }
    if (task->worker_emit_error && task->source != NULL) {
        if (task->kind == JAYESS_TASK_SOCKET_READ) {
            jayess_std_stream_emit_error(task->source, "failed to read from socket");
        } else if (task->kind == JAYESS_TASK_SOCKET_WRITE) {
            jayess_std_stream_emit_error(task->source, "failed to write to socket");
        } else if (task->kind == JAYESS_TASK_SERVER_ACCEPT) {
            jayess_std_stream_emit_error(task->source, "failed to accept socket connection");
        }
    }
    if (task->worker_emit_close && task->source != NULL) {
        jayess_std_socket_set_handle(task->source, JAYESS_INVALID_SOCKET);
        jayess_std_socket_close_native(task->source);
        jayess_std_socket_mark_closed(task->source);
        jayess_std_socket_emit_close(task->source);
    }
    if (jayess_has_exception()) {
        jayess_promise_settle(task->result, "rejected", jayess_take_exception());
        return 1;
    }
    jayess_promise_settle(task->result, "fulfilled", task->worker_result);
    return 1;
}

static int jayess_run_microtask(jayess_microtask *task) {
    jayess_value *stored;
    jayess_value *callback;
    jayess_value *result;
    int rejected;
    if (task == NULL) {
        return 1;
    }
    if (task->kind == JAYESS_TASK_FS_READ || task->kind == JAYESS_TASK_FS_WRITE || task->kind == JAYESS_TASK_SOCKET_READ || task->kind == JAYESS_TASK_SOCKET_WRITE || task->kind == JAYESS_TASK_SERVER_ACCEPT || task->kind == JAYESS_TASK_HTTP_REQUEST || task->kind == JAYESS_TASK_HTTP_GET || task->kind == JAYESS_TASK_HTTP_REQUEST_STREAM || task->kind == JAYESS_TASK_HTTP_GET_STREAM || task->kind == JAYESS_TASK_HTTPS_REQUEST || task->kind == JAYESS_TASK_HTTPS_GET || task->kind == JAYESS_TASK_HTTPS_REQUEST_STREAM || task->kind == JAYESS_TASK_HTTPS_GET_STREAM) {
        return jayess_run_async_worker_task(task);
    }
    if (task->kind == JAYESS_TASK_TIMER) {
        if (task->result != NULL) {
            jayess_promise_settle(task->result, "fulfilled", task->worker_result);
            return 1;
        }
        if (task->on_fulfilled == NULL || task->on_fulfilled->kind != JAYESS_VALUE_FUNCTION) {
            jayess_throw(jayess_type_error_value("setTimeout callback must be a function"));
            return 1;
        }
        jayess_value_call_one(task->on_fulfilled, jayess_value_undefined());
        return 1;
    }
    if (task->promise_action == JAYESS_PROMISE_ACTION_ALL) {
        return jayess_run_promise_all_task(task);
    }
    if (task->promise_action == JAYESS_PROMISE_ACTION_RACE) {
        return jayess_run_promise_race_task(task);
    }
    if (task->promise_action == JAYESS_PROMISE_ACTION_ALL_SETTLED) {
        return jayess_run_promise_all_settled_task(task);
    }
    if (task->promise_action == JAYESS_PROMISE_ACTION_ANY) {
        return jayess_run_promise_any_task(task);
    }
    if (jayess_promise_is_state(task->source, "pending")) {
        if (jayess_promise_add_dependent(task->source, task)) {
            return 0;
        }
        jayess_enqueue_microtask(task->source, task->result, task->on_fulfilled, task->on_rejected);
        return 1;
    }
    stored = jayess_promise_value(task->source);
    rejected = jayess_promise_is_state(task->source, "rejected");
    callback = rejected ? task->on_rejected : task->on_fulfilled;
    if (task->promise_action == JAYESS_PROMISE_ACTION_FINALLY) {
        if (task->on_fulfilled != NULL && task->on_fulfilled->kind != JAYESS_VALUE_UNDEFINED && task->on_fulfilled->kind != JAYESS_VALUE_NULL) {
            if (task->on_fulfilled->kind != JAYESS_VALUE_FUNCTION) {
                jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise.finally callback must be a function"));
                return 1;
            }
            result = jayess_value_call_one(task->on_fulfilled, jayess_value_undefined());
            if (jayess_has_exception()) {
                jayess_promise_settle(task->result, "rejected", jayess_take_exception());
                return 1;
            }
            if (result != NULL && result->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(result, "Promise")) {
                jayess_run_microtasks();
                if (jayess_promise_is_state(result, "pending")) {
                    if (!jayess_promise_add_dependent(result, task)) {
                        jayess_requeue_microtask(task);
                    }
                    jayess_sleep_ms(1);
                    return 0;
                }
                if (jayess_promise_is_state(result, "rejected")) {
                    jayess_promise_settle(task->result, "rejected", jayess_promise_value(result));
                    return 1;
                }
            }
        }
        jayess_promise_settle(task->result, rejected ? "rejected" : "fulfilled", stored);
        return 1;
    }
    if (callback == NULL || callback->kind == JAYESS_VALUE_UNDEFINED || callback->kind == JAYESS_VALUE_NULL) {
        jayess_promise_settle(task->result, rejected ? "rejected" : "fulfilled", stored);
        return 1;
    }
    if (callback->kind != JAYESS_VALUE_FUNCTION) {
        jayess_promise_settle(task->result, "rejected", jayess_type_error_value("Promise callback must be a function"));
        return 1;
    }
    result = jayess_value_call_one(callback, stored);
    if (jayess_has_exception()) {
        jayess_promise_settle(task->result, "rejected", jayess_take_exception());
        return 1;
    }
    if (result != NULL && result->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(result, "Promise")) {
        jayess_run_microtasks();
        if (jayess_promise_is_state(result, "pending")) {
            jayess_enqueue_microtask(result, task->result, jayess_value_undefined(), jayess_value_undefined());
            return 1;
        }
        stored = jayess_promise_value(result);
        jayess_promise_settle(task->result, jayess_promise_is_state(result, "rejected") ? "rejected" : "fulfilled", stored);
        return 1;
    }
    jayess_promise_settle(task->result, "fulfilled", result);
    return 1;
}

void jayess_run_microtasks(void) {
    int guard = 0;
    while (jayess_scheduler_has_tasks() && guard < 100000) {
        jayess_runtime_dispatch_pending_signals();
        if (jayess_has_exception()) {
            return;
        }
        jayess_microtask *task = jayess_dequeue_microtask();
        if (task == NULL) {
            jayess_sleep_ms(jayess_scheduler_next_timer_delay_ms());
            guard++;
            continue;
        }
        if (jayess_run_microtask(task)) {
            if (task->dependency_count > 0) {
                task->finished = 1;
            } else {
                free(task);
            }
        }
        if (jayess_has_exception()) {
            return;
        }
        guard++;
    }
    if (guard >= 100000) {
        jayess_throw(jayess_type_error_value("microtask queue did not settle"));
    }
}

static void jayess_runtime_free_args(void) {
    if (jayess_current_args == NULL) {
        return;
    }
    free(jayess_current_args->values);
    free(jayess_current_args);
    jayess_current_args = NULL;
}

static void jayess_runtime_free_symbol_registry(void) {
    jayess_symbol_registry_entry *current = jayess_symbol_registry;
    while (current != NULL) {
        jayess_symbol_registry_entry *next = current->next;
        free(current->key);
        if (current->symbol != NULL) {
            jayess_value_free_unshared(current->symbol);
        }
        free(current);
        current = next;
    }
    jayess_symbol_registry = NULL;
}

static void jayess_runtime_free_root_value(jayess_value **slot) {
    if (slot == NULL || *slot == NULL) {
        return;
    }
    if (*slot != jayess_value_null() && *slot != jayess_value_undefined()) {
        jayess_value_free_unshared(*slot);
    }
    *slot = NULL;
}

void jayess_runtime_shutdown(void) {
    jayess_io_pool_shutdown();
    jayess_runtime_free_args();
    jayess_runtime_error_state_shutdown();
    jayess_runtime_free_static_strings();
    jayess_runtime_free_root_value(&jayess_process_signal_bus);
    jayess_runtime_free_root_value(&jayess_dns_custom_resolver);
    jayess_runtime_free_root_value(&jayess_symbol_iterator_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_async_iterator_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_to_string_tag_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_has_instance_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_species_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_match_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_replace_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_search_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_split_singleton);
    jayess_runtime_free_root_value(&jayess_symbol_to_primitive_singleton);
    jayess_runtime_free_symbol_registry();
}

jayess_value *jayess_set_timeout(jayess_value *callback, jayess_value *delay) {
    int delay_ms = (int)jayess_value_to_number(delay);
    int timer_id;
    if (callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        jayess_throw(jayess_type_error_value("setTimeout callback must be a function"));
        return jayess_value_undefined();
    }
    timer_id = jayess_next_timer_id;
    jayess_enqueue_timer_task(callback, delay_ms);
    return jayess_value_from_number((double)timer_id);
}

jayess_value *jayess_clear_timeout(jayess_value *id) {
    int timer_id = (int)jayess_value_to_number(id);
    jayess_microtask *previous = NULL;
    jayess_task_queue *queue = &jayess_runtime_scheduler.timers;
    jayess_microtask *current = queue->head;
    while (current != NULL) {
        if (current->kind == JAYESS_TASK_TIMER && current->timer_id == timer_id) {
            if (previous != NULL) {
                previous->next = current->next;
            } else {
                queue->head = current->next;
            }
            if (queue->tail == current) {
                queue->tail = previous;
            }
            free(current);
            break;
        }
        previous = current;
        current = current->next;
    }
    return jayess_value_undefined();
}

jayess_value *jayess_sleep_async(jayess_value *delay, jayess_value *value) {
    jayess_value *promise = jayess_std_promise_pending();
    int delay_ms = (int)jayess_value_to_number(delay);
    jayess_enqueue_sleep_async_task(promise, delay_ms, value);
    return promise;
}

static jayess_value *jayess_std_promise_then_method(jayess_value *env, jayess_value *on_fulfilled, jayess_value *on_rejected) {
    jayess_value *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(env, "Promise")) {
        return jayess_std_promise_reject(jayess_type_error_value("Promise.then called on non-promise value"));
    }
    result = jayess_std_promise_pending();
    jayess_enqueue_microtask(env, result, on_fulfilled, on_rejected);
    return result;
}

static jayess_value *jayess_std_promise_catch_method(jayess_value *env, jayess_value *on_rejected) {
    return jayess_std_promise_then_method(env, jayess_value_undefined(), on_rejected);
}

static jayess_value *jayess_std_promise_finally_method(jayess_value *env, jayess_value *on_finally) {
    jayess_value *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(env, "Promise")) {
        return jayess_std_promise_reject(jayess_type_error_value("Promise.finally called on non-promise value"));
    }
    result = jayess_std_promise_pending();
    jayess_enqueue_promise_task(env, result, on_finally, jayess_value_undefined(), JAYESS_PROMISE_ACTION_FINALLY);
    return result;
}

jayess_value *jayess_await(jayess_value *value) {
    jayess_value *resolved;
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Promise")) {
        jayess_run_microtasks();
        resolved = jayess_promise_value(value);
        if (jayess_promise_is_state(value, "pending")) {
            jayess_throw(jayess_type_error_value("awaited promise did not settle"));
            return jayess_value_undefined();
        }
        if (jayess_promise_is_state(value, "rejected")) {
            jayess_throw(resolved != NULL ? resolved : jayess_value_undefined());
            return jayess_value_undefined();
        }
        return resolved != NULL ? resolved : jayess_value_undefined();
    }
    return value != NULL ? value : jayess_value_undefined();
}

typedef struct jayess_json_parser {
    const char *cursor;
} jayess_json_parser;

static void jayess_json_skip_ws(jayess_json_parser *parser) {
    while (parser->cursor != NULL && *parser->cursor != '\0' && isspace((unsigned char)*parser->cursor)) {
        parser->cursor++;
    }
}

static int jayess_regex_atom_length(const char *pattern) {
    if (pattern == NULL || pattern[0] == '\0') {
        return 0;
    }
    if (pattern[0] == '\\' && pattern[1] != '\0') {
        return 2;
    }
    return 1;
}

static int jayess_regex_atom_matches(const char *pattern, char value) {
    if (pattern == NULL || pattern[0] == '\0' || value == '\0') {
        return 0;
    }
    if (pattern[0] == '\\' && pattern[1] != '\0') {
        return value == pattern[1];
    }
    return pattern[0] == '.' || pattern[0] == value;
}

static int jayess_regex_match_here(const char *pattern, const char *text, const char **end) {
    int atom_len;
    char quantifier;
    const char *cursor;
    if (pattern == NULL || text == NULL) {
        return 0;
    }
    if (pattern[0] == '\0') {
        *end = text;
        return 1;
    }
    if (pattern[0] == '$' && pattern[1] == '\0') {
        if (*text == '\0') {
            *end = text;
            return 1;
        }
        return 0;
    }
    atom_len = jayess_regex_atom_length(pattern);
    quantifier = pattern[atom_len];
    if (quantifier == '*') {
        cursor = text;
        do {
            if (jayess_regex_match_here(pattern + atom_len + 1, cursor, end)) {
                return 1;
            }
        } while (*cursor != '\0' && jayess_regex_atom_matches(pattern, *cursor++));
        return 0;
    }
    if (quantifier == '+') {
        if (*text == '\0' || !jayess_regex_atom_matches(pattern, *text)) {
            return 0;
        }
        cursor = text + 1;
        do {
            if (jayess_regex_match_here(pattern + atom_len + 1, cursor, end)) {
                return 1;
            }
        } while (*cursor != '\0' && jayess_regex_atom_matches(pattern, *cursor++));
        return 0;
    }
    if (quantifier == '?') {
        if (jayess_regex_match_here(pattern + atom_len + 1, text, end)) {
            return 1;
        }
        if (*text != '\0' && jayess_regex_atom_matches(pattern, *text)) {
            return jayess_regex_match_here(pattern + atom_len + 1, text + 1, end);
        }
        return 0;
    }
    if (*text != '\0' && jayess_regex_atom_matches(pattern, *text)) {
        return jayess_regex_match_here(pattern + atom_len, text + 1, end);
    }
    return 0;
}

static int jayess_regex_search(const char *pattern, const char *text, int *start_out, int *end_out) {
    const char *end = NULL;
    const char *cursor;
    const char *search_pattern = pattern != NULL ? pattern : "";
    const char *search_text = text != NULL ? text : "";
    if (search_pattern[0] == '^') {
        if (jayess_regex_match_here(search_pattern + 1, search_text, &end)) {
            *start_out = 0;
            *end_out = (int)(end - search_text);
            return 1;
        }
        return 0;
    }
    for (cursor = search_text; ; cursor++) {
        if (jayess_regex_match_here(search_pattern, cursor, &end)) {
            *start_out = (int)(cursor - search_text);
            *end_out = (int)(end - search_text);
            return 1;
        }
        if (*cursor == '\0') {
            break;
        }
    }
    return 0;
}

static const char *jayess_regex_pattern_from_value(jayess_value *value) {
    jayess_value *pattern;
    if (value == NULL) {
        return "";
    }
    if (value->kind == JAYESS_VALUE_STRING) {
        return value->as.string_value != NULL ? value->as.string_value : "";
    }
    if (value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "RegExp")) {
        pattern = jayess_object_get(value->as.object_value, "__jayess_regexp_pattern");
        if (pattern != NULL && pattern->kind == JAYESS_VALUE_STRING) {
            return pattern->as.string_value != NULL ? pattern->as.string_value : "";
        }
    }
    return "";
}

static jayess_value *jayess_json_parse_value(jayess_json_parser *parser);

static char *jayess_json_parse_string_raw(jayess_json_parser *parser) {
    size_t cap = 16;
    size_t len = 0;
    char *out;
    if (parser->cursor == NULL || *parser->cursor != '"') {
        return NULL;
    }
    parser->cursor++;
    out = (char *)malloc(cap);
    if (out == NULL) {
        return NULL;
    }
    while (*parser->cursor != '\0' && *parser->cursor != '"') {
        char ch = *parser->cursor++;
        if (ch == '\\') {
            ch = *parser->cursor++;
            switch (ch) {
                case '"': break;
                case '\\': break;
                case '/': break;
                case 'b': ch = '\b'; break;
                case 'f': ch = '\f'; break;
                case 'n': ch = '\n'; break;
                case 'r': ch = '\r'; break;
                case 't': ch = '\t'; break;
                default:
                    free(out);
                    return NULL;
            }
        }
        if (len + 2 > cap) {
            char *grown;
            cap *= 2;
            grown = (char *)realloc(out, cap);
            if (grown == NULL) {
                free(out);
                return NULL;
            }
            out = grown;
        }
        out[len++] = ch;
    }
    if (*parser->cursor != '"') {
        free(out);
        return NULL;
    }
    parser->cursor++;
    out[len] = '\0';
    return out;
}

static jayess_value *jayess_json_parse_string(jayess_json_parser *parser) {
    char *text = jayess_json_parse_string_raw(parser);
    jayess_value *value;
    if (text == NULL) {
        return jayess_value_undefined();
    }
    value = jayess_value_from_string(text);
    free(text);
    return value;
}

static jayess_value *jayess_json_parse_number(jayess_json_parser *parser) {
    char *end = NULL;
    double value = strtod(parser->cursor, &end);
    if (end == parser->cursor) {
        return jayess_value_undefined();
    }
    parser->cursor = end;
    return jayess_value_from_number(value);
}

static jayess_value *jayess_json_parse_array(jayess_json_parser *parser) {
    jayess_array *array = jayess_array_new();
    if (*parser->cursor != '[') {
        return jayess_value_undefined();
    }
    parser->cursor++;
    jayess_json_skip_ws(parser);
    if (*parser->cursor == ']') {
        parser->cursor++;
        return jayess_value_from_array(array);
    }
    while (*parser->cursor != '\0') {
        jayess_value *item = jayess_json_parse_value(parser);
        jayess_array_push_value(array, item);
        jayess_json_skip_ws(parser);
        if (*parser->cursor == ']') {
            parser->cursor++;
            return jayess_value_from_array(array);
        }
        if (*parser->cursor != ',') {
            return jayess_value_undefined();
        }
        parser->cursor++;
        jayess_json_skip_ws(parser);
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_json_parse_object(jayess_json_parser *parser) {
    jayess_object *object = jayess_object_new();
    if (*parser->cursor != '{') {
        return jayess_value_undefined();
    }
    parser->cursor++;
    jayess_json_skip_ws(parser);
    if (*parser->cursor == '}') {
        parser->cursor++;
        return jayess_value_from_object(object);
    }
    while (*parser->cursor != '\0') {
        char *key;
        jayess_value *value;
        if (*parser->cursor != '"') {
            return jayess_value_undefined();
        }
        key = jayess_json_parse_string_raw(parser);
        if (key == NULL) {
            return jayess_value_undefined();
        }
        jayess_json_skip_ws(parser);
        if (*parser->cursor != ':') {
            free(key);
            return jayess_value_undefined();
        }
        parser->cursor++;
        jayess_json_skip_ws(parser);
        value = jayess_json_parse_value(parser);
        jayess_object_set_value(object, key, value);
        free(key);
        jayess_json_skip_ws(parser);
        if (*parser->cursor == '}') {
            parser->cursor++;
            return jayess_value_from_object(object);
        }
        if (*parser->cursor != ',') {
            return jayess_value_undefined();
        }
        parser->cursor++;
        jayess_json_skip_ws(parser);
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_json_parse_value(jayess_json_parser *parser) {
    jayess_json_skip_ws(parser);
    if (parser->cursor == NULL || *parser->cursor == '\0') {
        return jayess_value_undefined();
    }
    switch (*parser->cursor) {
        case '"':
            return jayess_json_parse_string(parser);
        case '{':
            return jayess_json_parse_object(parser);
        case '[':
            return jayess_json_parse_array(parser);
        case 't':
            if (strncmp(parser->cursor, "true", 4) == 0) {
                parser->cursor += 4;
                return jayess_value_from_bool(1);
            }
            break;
        case 'f':
            if (strncmp(parser->cursor, "false", 5) == 0) {
                parser->cursor += 5;
                return jayess_value_from_bool(0);
            }
            break;
        case 'n':
            if (strncmp(parser->cursor, "null", 4) == 0) {
                parser->cursor += 4;
                return jayess_value_null();
            }
            break;
        default:
            if (*parser->cursor == '-' || isdigit((unsigned char)*parser->cursor)) {
                return jayess_json_parse_number(parser);
            }
            break;
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_json_stringify(jayess_value *value) {
    char *text = jayess_value_stringify(value);
    jayess_value *result = jayess_value_from_string(text != NULL ? text : "");
    free(text);
    return result;
}

jayess_value *jayess_std_json_parse(jayess_value *value) {
    jayess_json_parser parser;
    if (value == NULL || value->kind != JAYESS_VALUE_STRING || value->as.string_value == NULL) {
        return jayess_value_undefined();
    }
    parser.cursor = value->as.string_value;
    return jayess_json_parse_value(&parser);
}

void jayess_value_set_member(jayess_value *target, const char *key, jayess_value *value) {
    jayess_object *properties = NULL;
    jayess_value *setter = NULL;
    char *setter_key;
    if (target == NULL) {
        return;
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        properties = target->as.object_value;
    } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        properties = target->as.function_value->properties;
    }
    if (properties == NULL) {
        return;
    }
    setter_key = jayess_accessor_key("__jayess_set_", key);
    if (setter_key != NULL) {
        setter = jayess_object_get(properties, setter_key);
    }
    if (setter != NULL && setter->kind == JAYESS_VALUE_FUNCTION) {
        (void)jayess_value_call_with_this(setter, target, value, 1);
        free(setter_key);
        return;
    }
    free(setter_key);
    jayess_object_set_value(properties, key, value);
}

static jayess_value *jayess_std_map_get_method(jayess_value *env, jayess_value *key) {
    int index = jayess_std_map_index_of(env, key);
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    if (index < 0 || values == NULL || index >= values->count) {
        return jayess_value_undefined();
    }
    return values->values[index] != NULL ? values->values[index] : jayess_value_undefined();
}

static jayess_value *jayess_std_map_keys_method(jayess_value *env) {
    jayess_array *keys = jayess_std_array_slot(env, "__jayess_map_keys");
    if (keys == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_array(jayess_array_clone(keys));
}

static jayess_value *jayess_std_map_values_method(jayess_value *env) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    if (values == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_array(jayess_array_clone(values));
}

static jayess_value *jayess_std_map_entries_method(jayess_value *env) {
    jayess_array *keys = jayess_std_array_slot(env, "__jayess_map_keys");
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    jayess_array *entries = jayess_array_new();
    int i;
    int count;
    if (keys == NULL || values == NULL) {
        return jayess_value_from_array(entries);
    }
    count = keys->count < values->count ? keys->count : values->count;
    for (i = 0; i < count; i++) {
        jayess_array *pair = jayess_array_new();
        jayess_array_push_value(pair, jayess_array_get(keys, i));
        jayess_array_push_value(pair, jayess_array_get(values, i));
        jayess_array_push_value(entries, jayess_value_from_array(pair));
    }
    return jayess_value_from_array(entries);
}

static jayess_value *jayess_std_map_set_method(jayess_value *env, jayess_value *key, jayess_value *value) {
    jayess_array *keys = jayess_std_array_slot(env, "__jayess_map_keys");
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    int index = jayess_std_map_index_of(env, key);
    if (keys == NULL || values == NULL) {
        return env != NULL ? env : jayess_value_undefined();
    }
    if (index < 0) {
        jayess_array_push_value(keys, key);
        jayess_array_push_value(values, value);
    } else {
        values->values[index] = value;
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_map_has_method(jayess_value *env, jayess_value *key) {
    return jayess_value_from_bool(jayess_std_map_index_of(env, key) >= 0);
}

static jayess_value *jayess_std_map_clear_method(jayess_value *env) {
    jayess_array *keys = jayess_std_array_slot(env, "__jayess_map_keys");
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    if (keys != NULL) {
        keys->count = 0;
    }
    if (values != NULL) {
        values->count = 0;
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_map_delete_method(jayess_value *env, jayess_value *key) {
    jayess_array *keys = jayess_std_array_slot(env, "__jayess_map_keys");
    jayess_array *values = jayess_std_array_slot(env, "__jayess_map_values");
    int index = jayess_std_map_index_of(env, key);
    if (index < 0 || keys == NULL || values == NULL) {
        return jayess_value_from_bool(0);
    }
    jayess_array_remove_at(keys, index);
    jayess_array_remove_at(values, index);
    return jayess_value_from_bool(1);
}

static jayess_value *jayess_std_set_add_method(jayess_value *env, jayess_value *value) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_set_values");
    if (values != NULL && jayess_std_set_index_of(env, value) < 0) {
        jayess_array_push_value(values, value);
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_set_values_method(jayess_value *env) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_set_values");
    if (values == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_array(jayess_array_clone(values));
}

static jayess_value *jayess_std_set_entries_method(jayess_value *env) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_set_values");
    jayess_array *entries = jayess_array_new();
    int i;
    if (values == NULL) {
        return jayess_value_from_array(entries);
    }
    for (i = 0; i < values->count; i++) {
        jayess_value *value = jayess_array_get(values, i);
        jayess_array *pair = jayess_array_new();
        jayess_array_push_value(pair, value);
        jayess_array_push_value(pair, value);
        jayess_array_push_value(entries, jayess_value_from_array(pair));
    }
    return jayess_value_from_array(entries);
}

static jayess_value *jayess_std_set_has_method(jayess_value *env, jayess_value *value) {
    return jayess_value_from_bool(jayess_std_set_index_of(env, value) >= 0);
}

static jayess_value *jayess_std_set_clear_method(jayess_value *env) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_set_values");
    if (values != NULL) {
        values->count = 0;
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_set_delete_method(jayess_value *env, jayess_value *value) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_set_values");
    int index = jayess_std_set_index_of(env, value);
    if (index < 0 || values == NULL) {
        return jayess_value_from_bool(0);
    }
    jayess_array_remove_at(values, index);
    return jayess_value_from_bool(1);
}

static jayess_value *jayess_std_weak_map_get_method(jayess_value *env, jayess_value *key) {
    int index;
    jayess_array *values;
    if (!jayess_std_weak_key_valid(key)) {
        jayess_throw(jayess_type_error_value("WeakMap keys must be objects or functions"));
        return jayess_value_undefined();
    }
    index = jayess_std_weak_map_index_of(env, key);
    values = jayess_std_array_slot(env, "__jayess_weak_map_values");
    if (index < 0 || values == NULL || index >= values->count) {
        return jayess_value_undefined();
    }
    return values->values[index] != NULL ? values->values[index] : jayess_value_undefined();
}

static jayess_value *jayess_std_weak_map_set_method(jayess_value *env, jayess_value *key, jayess_value *value) {
    jayess_array *keys;
    jayess_array *values;
    int index;
    if (!jayess_std_weak_key_valid(key)) {
        jayess_throw(jayess_type_error_value("WeakMap keys must be objects or functions"));
        return jayess_value_undefined();
    }
    keys = jayess_std_array_slot(env, "__jayess_weak_map_keys");
    values = jayess_std_array_slot(env, "__jayess_weak_map_values");
    index = jayess_std_weak_map_index_of(env, key);
    if (keys == NULL || values == NULL) {
        return env != NULL ? env : jayess_value_undefined();
    }
    if (index < 0) {
        jayess_array_push_value(keys, key);
        jayess_array_push_value(values, value);
    } else {
        values->values[index] = value;
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_weak_map_has_method(jayess_value *env, jayess_value *key) {
    if (!jayess_std_weak_key_valid(key)) {
        jayess_throw(jayess_type_error_value("WeakMap keys must be objects or functions"));
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(jayess_std_weak_map_index_of(env, key) >= 0);
}

static jayess_value *jayess_std_weak_map_delete_method(jayess_value *env, jayess_value *key) {
    jayess_array *keys;
    jayess_array *values;
    int index;
    if (!jayess_std_weak_key_valid(key)) {
        jayess_throw(jayess_type_error_value("WeakMap keys must be objects or functions"));
        return jayess_value_undefined();
    }
    keys = jayess_std_array_slot(env, "__jayess_weak_map_keys");
    values = jayess_std_array_slot(env, "__jayess_weak_map_values");
    index = jayess_std_weak_map_index_of(env, key);
    if (index < 0 || keys == NULL || values == NULL) {
        return jayess_value_from_bool(0);
    }
    jayess_array_remove_at(keys, index);
    jayess_array_remove_at(values, index);
    return jayess_value_from_bool(1);
}

static jayess_value *jayess_std_weak_set_add_method(jayess_value *env, jayess_value *value) {
    jayess_array *values;
    if (!jayess_std_weak_key_valid(value)) {
        jayess_throw(jayess_type_error_value("WeakSet values must be objects or functions"));
        return jayess_value_undefined();
    }
    values = jayess_std_array_slot(env, "__jayess_weak_set_values");
    if (values != NULL && jayess_std_weak_set_index_of(env, value) < 0) {
        jayess_array_push_value(values, value);
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_weak_set_has_method(jayess_value *env, jayess_value *value) {
    if (!jayess_std_weak_key_valid(value)) {
        jayess_throw(jayess_type_error_value("WeakSet values must be objects or functions"));
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(jayess_std_weak_set_index_of(env, value) >= 0);
}

static jayess_value *jayess_std_weak_set_delete_method(jayess_value *env, jayess_value *value) {
    jayess_array *values;
    int index;
    if (!jayess_std_weak_key_valid(value)) {
        jayess_throw(jayess_type_error_value("WeakSet values must be objects or functions"));
        return jayess_value_undefined();
    }
    values = jayess_std_array_slot(env, "__jayess_weak_set_values");
    index = jayess_std_weak_set_index_of(env, value);
    if (index < 0 || values == NULL) {
        return jayess_value_from_bool(0);
    }
    jayess_array_remove_at(values, index);
    return jayess_value_from_bool(1);
}

static jayess_value *jayess_std_date_get_time_method(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_object_get(env->as.object_value, "__jayess_date_ms");
}

static jayess_value *jayess_std_date_to_string_method(jayess_value *env) {
    jayess_value *ms = jayess_std_date_get_time_method(env);
    time_t seconds = (time_t)(jayess_value_to_number(ms) / 1000.0);
    struct tm tm_value;
    char buffer[64];
#ifdef _WIN32
    localtime_s(&tm_value, &seconds);
#else
    localtime_r(&seconds, &tm_value);
#endif
    strftime(buffer, sizeof(buffer), "%a %b %d %Y %H:%M:%S", &tm_value);
    return jayess_value_from_string(buffer);
}

static jayess_value *jayess_std_date_to_iso_string_method(jayess_value *env) {
    jayess_value *ms = jayess_std_date_get_time_method(env);
    double millis = jayess_value_to_number(ms);
    time_t seconds = (time_t)(millis / 1000.0);
    int ms_part = ((int)millis) % 1000;
    struct tm tm_value;
    char base[32];
    char buffer[40];
    if (ms_part < 0) {
        ms_part += 1000;
    }
#ifdef _WIN32
    gmtime_s(&tm_value, &seconds);
#else
    gmtime_r(&seconds, &tm_value);
#endif
    strftime(base, sizeof(base), "%Y-%m-%dT%H:%M:%S", &tm_value);
    snprintf(buffer, sizeof(buffer), "%s.%03dZ", base, ms_part);
    return jayess_value_from_string(buffer);
}

static jayess_value *jayess_std_regexp_test_method(jayess_value *env, jayess_value *text) {
    const char *pattern = jayess_regex_pattern_from_value(env);
    const char *value = jayess_value_as_string(text);
    int start = 0;
    int end = 0;
    return jayess_value_from_bool(jayess_regex_search(pattern, value != NULL ? value : "", &start, &end));
}

static jayess_value *jayess_std_error_to_string_method(jayess_value *env) {
	jayess_value *name = jayess_value_get_member(env, "name");
    jayess_value *message = jayess_value_get_member(env, "message");
    const char *name_text = name != NULL && name->kind == JAYESS_VALUE_STRING ? name->as.string_value : "Error";
    const char *message_text = message != NULL && message->kind == JAYESS_VALUE_STRING ? message->as.string_value : "";
    size_t name_len = strlen(name_text != NULL ? name_text : "Error");
    size_t message_len = strlen(message_text != NULL ? message_text : "");
    char *combined;
    if (message_len == 0) {
        return jayess_value_from_string(name_text);
    }
    combined = (char *)malloc(name_len + message_len + 3);
    if (combined == NULL) {
        return jayess_value_from_string(name_text);
    }
    sprintf(combined, "%s: %s", name_text, message_text);
    {
        jayess_value *out = jayess_value_from_string(combined);
        free(combined);
        return out;
	}
}

jayess_array *jayess_std_bytes_slot(jayess_value *target) {
	jayess_buffer_state *state;
	jayess_value *stored;
	if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return NULL;
    }
    state = jayess_std_bytes_state(target);
    if (state != NULL) {
        return state->bytes;
    }
    stored = jayess_object_get(target->as.object_value, "__jayess_bytes");
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
	return stored->as.array_value;
}
static jayess_value *jayess_std_uint8_fill_method(jayess_value *env, jayess_value *value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
    int byte_value = (int)jayess_value_to_number(value) & 255;
    int i;
    if (bytes == NULL) {
        return env;
    }
    for (i = 0; i < bytes->count; i++) {
        jayess_array_set_value(bytes, i, jayess_value_from_number((double)byte_value));
    }
    return env;
}

static jayess_value *jayess_std_uint8_includes_method(jayess_value *env, jayess_value *value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int byte_value = (int)jayess_value_to_number(value) & 255;
    int i;
    if (bytes == NULL) {
        return jayess_value_from_bool(0);
    }
    for (i = 0; i < bytes->count; i++) {
        if (((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255) == byte_value) {
            return jayess_value_from_bool(1);
        }
    }
	return jayess_value_from_bool(0);
}

static int jayess_std_uint8_index_of_value(jayess_array *bytes, jayess_value *needle) {
	jayess_array *needle_bytes = jayess_std_bytes_slot(needle);
	int i;
	int j;
	if (bytes == NULL) {
		return -1;
	}
	if (needle_bytes != NULL) {
		if (needle_bytes->count == 0) {
			return 0;
		}
		if (needle_bytes->count > bytes->count) {
			return -1;
		}
		for (i = 0; i <= bytes->count - needle_bytes->count; i++) {
			int matched = 1;
			for (j = 0; j < needle_bytes->count; j++) {
				int left_byte = (int)jayess_value_to_number(jayess_array_get(bytes, i+j)) & 255;
				int right_byte = (int)jayess_value_to_number(jayess_array_get(needle_bytes, j)) & 255;
				if (left_byte != right_byte) {
					matched = 0;
					break;
				}
			}
			if (matched) {
				return i;
			}
		}
		return -1;
	}
	{
		int byte_value = (int)jayess_value_to_number(needle) & 255;
		for (i = 0; i < bytes->count; i++) {
			if (((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255) == byte_value) {
				return i;
			}
		}
	}
	return -1;
}

static jayess_value *jayess_std_uint8_index_of_method(jayess_value *env, jayess_value *needle) {
	return jayess_value_from_number((double)jayess_std_uint8_index_of_value(jayess_std_bytes_slot(env), needle));
}

static jayess_value *jayess_std_uint8_starts_with_method(jayess_value *env, jayess_value *needle) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	jayess_array *needle_bytes = jayess_std_bytes_slot(needle);
	if (bytes == NULL) {
		return jayess_value_from_bool(0);
	}
	if (needle_bytes != NULL) {
		if (needle_bytes->count > bytes->count) {
			return jayess_value_from_bool(0);
		}
		return jayess_value_from_bool(jayess_std_uint8_index_of_value(bytes, needle) == 0);
	}
	if (bytes->count == 0) {
		return jayess_value_from_bool(0);
	}
	return jayess_value_from_bool(((int)jayess_value_to_number(jayess_array_get(bytes, 0)) & 255) == ((int)jayess_value_to_number(needle) & 255));
}

static jayess_value *jayess_std_uint8_ends_with_method(jayess_value *env, jayess_value *needle) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	jayess_array *needle_bytes = jayess_std_bytes_slot(needle);
	int start;
	int i;
	if (bytes == NULL) {
		return jayess_value_from_bool(0);
	}
	if (needle_bytes != NULL) {
		if (needle_bytes->count > bytes->count) {
			return jayess_value_from_bool(0);
		}
		start = bytes->count - needle_bytes->count;
		for (i = 0; i < needle_bytes->count; i++) {
			int left_byte = (int)jayess_value_to_number(jayess_array_get(bytes, start+i)) & 255;
			int right_byte = (int)jayess_value_to_number(jayess_array_get(needle_bytes, i)) & 255;
			if (left_byte != right_byte) {
				return jayess_value_from_bool(0);
			}
		}
		return jayess_value_from_bool(1);
	}
	if (bytes->count == 0) {
		return jayess_value_from_bool(0);
	}
	return jayess_value_from_bool(((int)jayess_value_to_number(jayess_array_get(bytes, bytes->count-1)) & 255) == ((int)jayess_value_to_number(needle) & 255));
}

static int jayess_std_uint8_clamped_index(jayess_value *value, int length, int default_value) {
	int index;
	if (value == NULL || jayess_value_is_nullish(value)) {
		index = default_value;
	} else {
		index = (int)jayess_value_to_number(value);
	}
	if (index < 0) {
		index = length + index;
	}
	if (index < 0) {
		index = 0;
	}
	if (index > length) {
		index = length;
	}
	return index;
}

static jayess_array *jayess_std_uint8_source_array(jayess_value *source) {
	jayess_array *bytes = jayess_std_bytes_slot(source);
	if (bytes != NULL) {
		return bytes;
	}
	if (source != NULL && source->kind == JAYESS_VALUE_ARRAY) {
		return source->as.array_value;
	}
	return NULL;
}

static jayess_value *jayess_std_uint8_set_method(jayess_value *env, jayess_value *source, jayess_value *offset_value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	jayess_array *source_bytes = jayess_std_uint8_source_array(source);
	int offset;
	int count;
	int i;
	if (bytes == NULL || source_bytes == NULL) {
		return jayess_value_undefined();
	}
	offset = jayess_std_uint8_clamped_index(offset_value, bytes->count, 0);
	count = source_bytes->count;
	if (count > bytes->count - offset) {
		count = bytes->count - offset;
	}
	for (i = 0; i < count; i++) {
		int byte_value = (int)jayess_value_to_number(jayess_array_get(source_bytes, i)) & 255;
		jayess_array_set_value(bytes, offset+i, jayess_value_from_number((double)byte_value));
	}
	return jayess_value_undefined();
}

static jayess_value *jayess_std_uint8_copy_within_method(jayess_value *env, jayess_value *target_value, jayess_value *start_value, jayess_value *end_value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int target;
	int start;
	int end;
	int count;
	int i;
	if (bytes == NULL) {
		return env;
	}
	target = jayess_std_uint8_clamped_index(target_value, bytes->count, 0);
	start = jayess_std_uint8_clamped_index(start_value, bytes->count, 0);
	end = jayess_std_uint8_clamped_index(end_value, bytes->count, bytes->count);
	if (end < start) {
		end = start;
	}
	count = end - start;
	if (count > bytes->count - target) {
		count = bytes->count - target;
	}
	if (count <= 0) {
		return env;
	}
	if (target < start) {
		for (i = 0; i < count; i++) {
			jayess_array_set_value(bytes, target+i, jayess_array_get(bytes, start+i));
		}
	} else {
		for (i = count - 1; i >= 0; i--) {
			jayess_array_set_value(bytes, target+i, jayess_array_get(bytes, start+i));
		}
	}
	return env;
}

static jayess_value *jayess_std_uint8_slice_values(jayess_value *env, int start, int end, int has_end) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int i;
    int out_index = 0;
    jayess_value *buffer;
    jayess_value *view;
    jayess_array *out_bytes;
    if (bytes == NULL) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
    if (start < 0) {
        start = bytes->count + start;
    }
    if (start < 0) {
        start = 0;
    }
    if (start > bytes->count) {
        start = bytes->count;
    }
    if (has_end) {
        if (end < 0) {
            end = bytes->count + end;
        }
    } else {
        end = bytes->count;
    }
    if (end < 0) {
        end = 0;
    }
    if (end > bytes->count) {
        end = bytes->count;
    }
    if (end < start) {
        end = start;
    }
    buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)(end - start)));
    view = jayess_std_uint8_array_new(buffer);
    out_bytes = jayess_std_bytes_slot(view);
    if (out_bytes == NULL) {
        return jayess_value_undefined();
    }
    for (i = start; i < end; i++) {
        jayess_array_set_value(out_bytes, out_index++, jayess_array_get(bytes, i));
    }
    return view;
}

static jayess_value *jayess_std_uint8_slice_method(jayess_value *env, jayess_value *start_value, jayess_value *end_value) {
    int start = (int)jayess_value_to_number(start_value);
    int has_end = end_value != NULL && !jayess_value_is_nullish(end_value);
    int end = has_end ? (int)jayess_value_to_number(end_value) : 0;
    return jayess_std_uint8_slice_values(env, start, end, has_end);
}

jayess_value *jayess_std_uint8_to_string_method(jayess_value *env, jayess_value *encoding) {
    jayess_array *bytes = jayess_std_bytes_slot(env);
    char *text;
    jayess_value *result;
    static const char *hex = "0123456789abcdef";
    static const char *base64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    int i;
    if (bytes == NULL) {
        return jayess_value_from_string("");
    }
    if (jayess_std_bytes_encoding_is_hex(encoding)) {
        text = (char *)malloc(((size_t)bytes->count * 2) + 1);
        if (text == NULL) {
            return jayess_value_from_string("");
        }
        for (i = 0; i < bytes->count; i++) {
            int byte_value = (int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255;
            text[i * 2] = hex[(byte_value >> 4) & 15];
            text[(i * 2) + 1] = hex[byte_value & 15];
        }
        text[bytes->count * 2] = '\0';
        result = jayess_value_from_string(text);
        free(text);
        return result;
    }
    if (jayess_std_bytes_encoding_is_base64(encoding)) {
        size_t out_len = ((size_t)(bytes->count + 2) / 3) * 4;
        size_t out_index = 0;
        text = (char *)malloc(out_len + 1);
        if (text == NULL) {
            return jayess_value_from_string("");
        }
        for (i = 0; i < bytes->count; i += 3) {
            int b0 = (int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255;
            int b1 = i + 1 < bytes->count ? ((int)jayess_value_to_number(jayess_array_get(bytes, i + 1)) & 255) : 0;
            int b2 = i + 2 < bytes->count ? ((int)jayess_value_to_number(jayess_array_get(bytes, i + 2)) & 255) : 0;
            unsigned int triple = ((unsigned int)b0 << 16) | ((unsigned int)b1 << 8) | (unsigned int)b2;
            text[out_index++] = base64[(triple >> 18) & 63];
            text[out_index++] = base64[(triple >> 12) & 63];
            text[out_index++] = i + 1 < bytes->count ? base64[(triple >> 6) & 63] : '=';
            text[out_index++] = i + 2 < bytes->count ? base64[triple & 63] : '=';
        }
        text[out_index] = '\0';
        result = jayess_value_from_string(text);
        free(text);
        return result;
    }
    if (!jayess_std_bytes_encoding_is_text(encoding)) {
        return jayess_value_undefined();
    }
    text = (char *)malloc((size_t)bytes->count + 1);
    if (text == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < bytes->count; i++) {
        text[i] = (char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
    }
    text[bytes->count] = '\0';
    result = jayess_value_from_string(text);
    free(text);
    return result;
}

static int jayess_std_uint8_concat_length(jayess_value *values) {
    int total = 0;
    int i;
    if (values == NULL || values->kind != JAYESS_VALUE_ARRAY || values->as.array_value == NULL) {
        return 0;
    }
    for (i = 0; i < values->as.array_value->count; i++) {
        jayess_value *item = jayess_array_get(values->as.array_value, i);
        jayess_array *bytes = jayess_std_bytes_slot(item);
        if (bytes != NULL) {
            total += bytes->count;
        }
    }
    return total;
}

jayess_value *jayess_std_uint8_array_concat(jayess_value *values) {
    int total = jayess_std_uint8_concat_length(values);
    jayess_value *buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)total));
    jayess_value *view = jayess_std_uint8_array_new(buffer);
    jayess_array *out = jayess_std_bytes_slot(view);
    int out_index = 0;
    int i;
    if (out == NULL || values == NULL || values->kind != JAYESS_VALUE_ARRAY || values->as.array_value == NULL) {
        return view;
    }
    for (i = 0; i < values->as.array_value->count; i++) {
        jayess_value *item = jayess_array_get(values->as.array_value, i);
        jayess_array *bytes = jayess_std_bytes_slot(item);
        int j;
        if (bytes == NULL) {
            continue;
        }
        for (j = 0; j < bytes->count; j++) {
            jayess_array_set_value(out, out_index++, jayess_array_get(bytes, j));
        }
    }
    return view;
}

jayess_value *jayess_std_uint8_array_equals(jayess_value *left, jayess_value *right) {
    jayess_array *left_bytes = jayess_std_bytes_slot(left);
    jayess_array *right_bytes = jayess_std_bytes_slot(right);
    int i;
    if (left_bytes == NULL || right_bytes == NULL) {
        return jayess_value_from_bool(0);
    }
    if (left_bytes->count != right_bytes->count) {
        return jayess_value_from_bool(0);
    }
    for (i = 0; i < left_bytes->count; i++) {
        int left_byte = (int)jayess_value_to_number(jayess_array_get(left_bytes, i)) & 255;
        int right_byte = (int)jayess_value_to_number(jayess_array_get(right_bytes, i)) & 255;
        if (left_byte != right_byte) {
            return jayess_value_from_bool(0);
        }
    }
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_uint8_array_compare(jayess_value *left, jayess_value *right) {
    jayess_array *left_bytes = jayess_std_bytes_slot(left);
    jayess_array *right_bytes = jayess_std_bytes_slot(right);
    int left_count;
    int right_count;
    int count;
    int i;
    if (left_bytes == NULL || right_bytes == NULL) {
        return jayess_value_from_number(0);
    }
    left_count = left_bytes->count;
    right_count = right_bytes->count;
    count = left_count < right_count ? left_count : right_count;
    for (i = 0; i < count; i++) {
        int left_byte = (int)jayess_value_to_number(jayess_array_get(left_bytes, i)) & 255;
        int right_byte = (int)jayess_value_to_number(jayess_array_get(right_bytes, i)) & 255;
        if (left_byte < right_byte) {
            return jayess_value_from_number(-1);
        }
        if (left_byte > right_byte) {
            return jayess_value_from_number(1);
        }
    }
    if (left_count < right_count) {
        return jayess_value_from_number(-1);
    }
    if (left_count > right_count) {
        return jayess_value_from_number(1);
    }
    return jayess_value_from_number(0);
}

static jayess_value *jayess_std_uint8_equals_method(jayess_value *env, jayess_value *other) {
    return jayess_std_uint8_array_equals(env, other);
}

static jayess_value *jayess_std_uint8_compare_method(jayess_value *env, jayess_value *other) {
    return jayess_std_uint8_array_compare(env, other);
}

static jayess_value *jayess_std_uint8_concat_method(jayess_value *env, jayess_value *values) {
    jayess_array *items = jayess_array_new();
    int i;
    if (items == NULL) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
    jayess_array_push_value(items, env);
    if (values != NULL && values->kind == JAYESS_VALUE_ARRAY && values->as.array_value != NULL) {
        for (i = 0; i < values->as.array_value->count; i++) {
            jayess_array_push_value(items, jayess_array_get(values->as.array_value, i));
        }
    }
    return jayess_std_uint8_array_concat(jayess_value_from_array(items));
}

static jayess_value *jayess_std_iterator_next_method(jayess_value *env) {
    jayess_array *values = jayess_std_array_slot(env, "__jayess_iterator_values");
    jayess_value *index_value = jayess_object_get(env->as.object_value, "__jayess_iterator_index");
    int index = (int)jayess_value_to_number(index_value);
    jayess_object *result = jayess_object_new();
    if (values == NULL || index >= values->count) {
        jayess_object_set_value(result, "value", jayess_value_undefined());
        jayess_object_set_value(result, "done", jayess_value_from_bool(1));
        return jayess_value_from_object(result);
    }
    jayess_object_set_value(result, "value", jayess_array_get(values, index));
    jayess_object_set_value(result, "done", jayess_value_from_bool(0));
    jayess_object_set_value(env->as.object_value, "__jayess_iterator_index", jayess_value_from_number((double)(index + 1)));
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_async_iterator_next_method(jayess_value *env) {
    return jayess_std_promise_resolve(jayess_std_iterator_next_method(env));
}

static jayess_value *jayess_std_async_iterator_identity_method(jayess_value *env) {
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_iterator_protocol_values(jayess_value *iterator) {
    jayess_array *values = jayess_array_new();
    int guard = 0;
    if (iterator == NULL) {
        return jayess_value_from_array(values);
    }
    while (guard++ < 1000000) {
        jayess_value *next_method = jayess_value_get_member(iterator, "next");
        jayess_value *step;
        jayess_value *done;
        jayess_value *value;
        if (next_method == NULL || next_method->kind != JAYESS_VALUE_FUNCTION) {
            break;
        }
        step = jayess_value_call_with_this(next_method, iterator, NULL, 0);
        if (step == NULL || step->kind != JAYESS_VALUE_OBJECT) {
            break;
        }
        done = jayess_value_get_member(step, "done");
        if (jayess_value_as_bool(done)) {
            break;
        }
        value = jayess_value_get_member(step, "value");
        jayess_array_push_value(values, value != NULL ? value : jayess_value_undefined());
    }
    return jayess_value_from_array(values);
}

static jayess_value *jayess_std_iterable_protocol_values(jayess_value *target) {
    jayess_value *iterator_symbol;
    jayess_value *iterator_method;
    jayess_value *iterator;
    if (target == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    if (target->kind == JAYESS_VALUE_STRING && target->as.string_value != NULL) {
        jayess_array *values = jayess_array_new();
        const char *text = target->as.string_value;
        int i;
        for (i = 0; text[i] != '\0'; i++) {
            char piece[2] = {text[i], '\0'};
            jayess_array_push_value(values, jayess_value_from_string(piece));
        }
        return jayess_value_from_array(values);
    }
    iterator_symbol = jayess_std_symbol_iterator();
    iterator_method = jayess_value_get_dynamic_index(target, iterator_symbol);
    if (iterator_method != NULL && iterator_method->kind == JAYESS_VALUE_FUNCTION) {
        iterator = jayess_value_call_with_this(iterator_method, target, NULL, 0);
        return jayess_std_iterator_protocol_values(iterator);
    }
    if (jayess_value_get_member(target, "next") != NULL) {
        return jayess_std_iterator_protocol_values(target);
    }
    return jayess_value_from_array(jayess_array_new());
}

static const char *jayess_string_env(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_STRING || env->as.string_value == NULL) {
        return "";
    }
    return env->as.string_value;
}

static jayess_value *jayess_std_string_includes_method(jayess_value *env, jayess_value *needle) {
    char *text = jayess_value_stringify(needle);
    int found = strstr(jayess_string_env(env), text != NULL ? text : "") != NULL;
    free(text);
    return jayess_value_from_bool(found);
}

static jayess_value *jayess_std_string_starts_with_method(jayess_value *env, jayess_value *prefix) {
    char *text = jayess_value_stringify(prefix);
    const char *value = jayess_string_env(env);
    size_t prefix_len = strlen(text != NULL ? text : "");
    int found = strncmp(value, text != NULL ? text : "", prefix_len) == 0;
    free(text);
    return jayess_value_from_bool(found);
}

static jayess_value *jayess_std_string_ends_with_method(jayess_value *env, jayess_value *suffix) {
    char *text = jayess_value_stringify(suffix);
    const char *value = jayess_string_env(env);
    size_t value_len = strlen(value);
    size_t suffix_len = strlen(text != NULL ? text : "");
    int found = value_len >= suffix_len && strcmp(value + value_len - suffix_len, text != NULL ? text : "") == 0;
    free(text);
    return jayess_value_from_bool(found);
}

static jayess_value *jayess_std_string_slice_method(jayess_value *env, jayess_value *start_value, jayess_value *end_value) {
    const char *value = jayess_string_env(env);
    int length = (int)strlen(value);
    int start = (int)jayess_value_to_number(start_value);
    int end = jayess_value_is_nullish(end_value) ? length : (int)jayess_value_to_number(end_value);
    char *out;
    if (start < 0) start = 0;
    if (end < start) end = start;
    if (end > length) end = length;
    out = (char *)malloc((size_t)(end - start + 1));
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    memcpy(out, value + start, (size_t)(end - start));
    out[end - start] = '\0';
    start_value = jayess_value_from_string(out);
    free(out);
    return start_value;
}

static jayess_value *jayess_std_string_trim_method(jayess_value *env) {
    const char *value = jayess_string_env(env);
    int start = 0;
    int end = (int)strlen(value);
    char *out;
    while (start < end && isspace((unsigned char)value[start])) start++;
    while (end > start && isspace((unsigned char)value[end - 1])) end--;
    out = (char *)malloc((size_t)(end - start + 1));
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    memcpy(out, value + start, (size_t)(end - start));
    out[end - start] = '\0';
    env = jayess_value_from_string(out);
    free(out);
    return env;
}

static jayess_value *jayess_std_string_upper_method(jayess_value *env) {
    const char *value = jayess_string_env(env);
    size_t length = strlen(value);
    char *out = (char *)malloc(length + 1);
    size_t i;
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < length; i++) out[i] = (char)toupper((unsigned char)value[i]);
    out[length] = '\0';
    env = jayess_value_from_string(out);
    free(out);
    return env;
}

static jayess_value *jayess_std_string_lower_method(jayess_value *env) {
    const char *value = jayess_string_env(env);
    size_t length = strlen(value);
    char *out = (char *)malloc(length + 1);
    size_t i;
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < length; i++) out[i] = (char)tolower((unsigned char)value[i]);
    out[length] = '\0';
    env = jayess_value_from_string(out);
    free(out);
    return env;
}

static jayess_value *jayess_std_string_split_method(jayess_value *env, jayess_value *separator) {
    const char *value = jayess_string_env(env);
    char *sep = separator != NULL ? jayess_value_stringify(separator) : jayess_strdup("");
    jayess_array *parts = jayess_array_new();
    if (sep == NULL || strlen(sep) == 0) {
        int i;
        for (i = 0; value[i] != '\0'; i++) {
            char piece[2] = { value[i], '\0' };
            jayess_array_push_value(parts, jayess_value_from_string(piece));
        }
        free(sep);
        return jayess_value_from_array(parts);
    }
    {
        const char *cursor = value;
        const char *found;
        while ((found = strstr(cursor, sep)) != NULL) {
            size_t len = (size_t)(found - cursor);
            char *piece = (char *)malloc(len + 1);
            memcpy(piece, cursor, len);
            piece[len] = '\0';
            jayess_array_push_value(parts, jayess_value_from_string(piece));
            free(piece);
            cursor = found + strlen(sep);
        }
        jayess_array_push_value(parts, jayess_value_from_string(cursor));
    }
    free(sep);
    return jayess_value_from_array(parts);
}

static jayess_value *jayess_std_string_match_method(jayess_value *env, jayess_value *pattern_value) {
    const char *value = jayess_string_env(env);
    const char *pattern = jayess_regex_pattern_from_value(pattern_value);
    int start = 0;
    int end = 0;
    jayess_array *matches;
    char *piece;
    if (!jayess_regex_search(pattern, value, &start, &end)) {
        return jayess_value_undefined();
    }
    matches = jayess_array_new();
    piece = (char *)malloc((size_t)(end - start + 1));
    if (piece == NULL) {
        return jayess_value_from_array(matches);
    }
    memcpy(piece, value + start, (size_t)(end - start));
    piece[end - start] = '\0';
    jayess_array_push_value(matches, jayess_value_from_string(piece));
    free(piece);
    return jayess_value_from_array(matches);
}

static jayess_value *jayess_std_string_search_method(jayess_value *env, jayess_value *pattern_value) {
    int start = 0;
    int end = 0;
    if (jayess_regex_search(jayess_regex_pattern_from_value(pattern_value), jayess_string_env(env), &start, &end)) {
        return jayess_value_from_number((double)start);
    }
    return jayess_value_from_number(-1);
}

static jayess_value *jayess_std_string_replace_method(jayess_value *env, jayess_value *pattern_value, jayess_value *replacement_value) {
    const char *value = jayess_string_env(env);
    const char *pattern = jayess_regex_pattern_from_value(pattern_value);
    char *replacement = jayess_value_stringify(replacement_value);
    int start = 0;
    int end = 0;
    size_t value_len = strlen(value);
    size_t replacement_len = strlen(replacement != NULL ? replacement : "");
    char *out;
    jayess_value *result;
    if (!jayess_regex_search(pattern, value, &start, &end)) {
        free(replacement);
        return jayess_value_from_string(value);
    }
    out = (char *)malloc(value_len - (size_t)(end - start) + replacement_len + 1);
    if (out == NULL) {
        free(replacement);
        return jayess_value_from_string(value);
    }
    memcpy(out, value, (size_t)start);
    memcpy(out + start, replacement != NULL ? replacement : "", replacement_len);
    strcpy(out + start + replacement_len, value + end);
    result = jayess_value_from_string(out);
    free(out);
    free(replacement);
    return result;
}

static jayess_value *jayess_std_string_regex_split_method(jayess_value *env, jayess_value *pattern_value) {
    const char *value = jayess_string_env(env);
    const char *pattern = jayess_regex_pattern_from_value(pattern_value);
    jayess_array *parts = jayess_array_new();
    int offset = 0;
    int length = (int)strlen(value);
    while (offset <= length) {
        int start = 0;
        int end = 0;
        const char *cursor = value + offset;
        char *piece;
        if (!jayess_regex_search(pattern, cursor, &start, &end)) {
            jayess_array_push_value(parts, jayess_value_from_string(cursor));
            break;
        }
        piece = (char *)malloc((size_t)start + 1);
        if (piece == NULL) {
            break;
        }
        memcpy(piece, cursor, (size_t)start);
        piece[start] = '\0';
        jayess_array_push_value(parts, jayess_value_from_string(piece));
        free(piece);
        offset += end;
        if (end == 0) {
            offset += 1;
        }
        if (offset > length) {
            jayess_array_push_value(parts, jayess_value_from_string(""));
            break;
        }
    }
    return jayess_value_from_array(parts);
}

static jayess_value *jayess_std_symbol_to_string_method(jayess_value *env) {
    char *text = jayess_value_stringify(env);
    jayess_value *result = jayess_value_from_string(text != NULL ? text : "");
    free(text);
    return result;
}

jayess_value *jayess_value_get_member(jayess_value *target, const char *key) {
    jayess_object *properties = NULL;
    jayess_value *getter = NULL;
    char *getter_key;
    if (target == NULL) {
        return NULL;
    }
    if (target->kind == JAYESS_VALUE_SYMBOL) {
        if (strcmp(key, "description") == 0) {
            if (target->as.symbol_value != NULL && target->as.symbol_value->description != NULL) {
                return jayess_value_from_string(target->as.symbol_value->description);
            }
            return jayess_value_undefined();
        }
        if (strcmp(key, "toString") == 0) {
            return jayess_value_from_function((void *)jayess_std_symbol_to_string_method, target, "toString", NULL, 0, 0);
        }
    }
    if (target->kind == JAYESS_VALUE_STRING) {
        if (strcmp(key, "length") == 0) {
            return jayess_value_from_number((double)strlen(target->as.string_value));
        }
        if (strcmp(key, "includes") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_includes_method, target, "includes", NULL, 1, 0);
        }
        if (strcmp(key, "startsWith") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_starts_with_method, target, "startsWith", NULL, 1, 0);
        }
        if (strcmp(key, "endsWith") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_ends_with_method, target, "endsWith", NULL, 1, 0);
        }
        if (strcmp(key, "slice") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_slice_method, target, "slice", NULL, 2, 0);
        }
        if (strcmp(key, "trim") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_trim_method, target, "trim", NULL, 0, 0);
        }
        if (strcmp(key, "toUpperCase") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_upper_method, target, "toUpperCase", NULL, 0, 0);
        }
        if (strcmp(key, "toLowerCase") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_lower_method, target, "toLowerCase", NULL, 0, 0);
        }
        if (strcmp(key, "split") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_split_method, target, "split", NULL, 1, 0);
        }
        if (strcmp(key, "match") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_match_method, target, "match", NULL, 1, 0);
        }
        if (strcmp(key, "search") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_search_method, target, "search", NULL, 1, 0);
        }
        if (strcmp(key, "replace") == 0) {
            return jayess_value_from_function((void *)jayess_std_string_replace_method, target, "replace", NULL, 2, 0);
        }
    }
    if (target->kind == JAYESS_VALUE_ARRAY) {
        if (strcmp(key, "includes") == 0) {
            return jayess_value_from_function((void *)jayess_value_array_includes, target, "includes", NULL, 1, 0);
        }
        if (strcmp(key, "join") == 0) {
            return jayess_value_from_function((void *)jayess_value_array_join, target, "join", NULL, 1, 0);
        }
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        if (jayess_std_kind_is(target, "Map")) {
            if (strcmp(key, "size") == 0) {
                jayess_array *keys = jayess_std_array_slot(target, "__jayess_map_keys");
                return jayess_value_from_number((double)(keys != NULL ? keys->count : 0));
            }
            if (strcmp(key, "get") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_get_method, target, "get", NULL, 1, 0);
            }
            if (strcmp(key, "set") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_set_method, target, "set", NULL, 2, 0);
            }
            if (strcmp(key, "has") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_has_method, target, "has", NULL, 1, 0);
            }
            if (strcmp(key, "delete") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_delete_method, target, "delete", NULL, 1, 0);
            }
            if (strcmp(key, "keys") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_keys_method, target, "keys", NULL, 0, 0);
            }
            if (strcmp(key, "values") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_values_method, target, "values", NULL, 0, 0);
            }
            if (strcmp(key, "entries") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_entries_method, target, "entries", NULL, 0, 0);
            }
            if (strcmp(key, "clear") == 0) {
                return jayess_value_from_function((void *)jayess_std_map_clear_method, target, "clear", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Set")) {
            if (strcmp(key, "size") == 0) {
                jayess_array *values = jayess_std_array_slot(target, "__jayess_set_values");
                return jayess_value_from_number((double)(values != NULL ? values->count : 0));
            }
            if (strcmp(key, "add") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_add_method, target, "add", NULL, 1, 0);
            }
            if (strcmp(key, "has") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_has_method, target, "has", NULL, 1, 0);
            }
            if (strcmp(key, "delete") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_delete_method, target, "delete", NULL, 1, 0);
            }
            if (strcmp(key, "values") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_values_method, target, "values", NULL, 0, 0);
            }
            if (strcmp(key, "entries") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_entries_method, target, "entries", NULL, 0, 0);
            }
            if (strcmp(key, "clear") == 0) {
                return jayess_value_from_function((void *)jayess_std_set_clear_method, target, "clear", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "WeakMap")) {
            if (strcmp(key, "get") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_map_get_method, target, "get", NULL, 1, 0);
            }
            if (strcmp(key, "set") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_map_set_method, target, "set", NULL, 2, 0);
            }
            if (strcmp(key, "has") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_map_has_method, target, "has", NULL, 1, 0);
            }
            if (strcmp(key, "delete") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_map_delete_method, target, "delete", NULL, 1, 0);
            }
        }
        if (jayess_std_kind_is(target, "WeakSet")) {
            if (strcmp(key, "add") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_set_add_method, target, "add", NULL, 1, 0);
            }
            if (strcmp(key, "has") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_set_has_method, target, "has", NULL, 1, 0);
            }
            if (strcmp(key, "delete") == 0) {
                return jayess_value_from_function((void *)jayess_std_weak_set_delete_method, target, "delete", NULL, 1, 0);
            }
        }
        if (jayess_std_kind_is(target, "Date")) {
            if (strcmp(key, "getTime") == 0) {
                return jayess_value_from_function((void *)jayess_std_date_get_time_method, target, "getTime", NULL, 0, 0);
            }
            if (strcmp(key, "toString") == 0) {
                return jayess_value_from_function((void *)jayess_std_date_to_string_method, target, "toString", NULL, 0, 0);
            }
            if (strcmp(key, "toISOString") == 0) {
                return jayess_value_from_function((void *)jayess_std_date_to_iso_string_method, target, "toISOString", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "RegExp")) {
            if (strcmp(key, "source") == 0) {
                return jayess_object_get(target->as.object_value, "__jayess_regexp_pattern");
            }
            if (strcmp(key, "flags") == 0) {
                return jayess_object_get(target->as.object_value, "__jayess_regexp_flags");
            }
            if (strcmp(key, "test") == 0) {
                return jayess_value_from_function((void *)jayess_std_regexp_test_method, target, "test", NULL, 1, 0);
            }
        }
        if (jayess_std_kind_is(target, "Error") || jayess_std_kind_is(target, "TypeError") || jayess_std_kind_is(target, "AggregateError")) {
            if (strcmp(key, "toString") == 0) {
                return jayess_value_from_function((void *)jayess_std_error_to_string_method, target, "toString", NULL, 0, 0);
            }
        }
		if (jayess_std_kind_is(target, "ArrayBuffer") || jayess_std_kind_is(target, "SharedArrayBuffer")) {
			if (strcmp(key, "byteLength") == 0) {
				return jayess_value_from_number((double)jayess_std_byte_length(target));
			}
		}
		if (jayess_std_kind_is(target, "DataView")) {
			if (strcmp(key, "buffer") == 0) {
				return jayess_std_buffer_value_from_state(jayess_std_bytes_state(target));
			}
			if (strcmp(key, "byteLength") == 0) {
				jayess_array *bytes = jayess_std_bytes_slot(target);
				return jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0));
			}
			if (strcmp(key, "getUint8") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_uint8_method, target, "getUint8", NULL, 1, 0);
			}
			if (strcmp(key, "setUint8") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_uint8_method, target, "setUint8", NULL, 2, 0);
			}
			if (strcmp(key, "getInt8") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_int8_method, target, "getInt8", NULL, 1, 0);
			}
			if (strcmp(key, "setInt8") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_int8_method, target, "setInt8", NULL, 2, 0);
			}
			if (strcmp(key, "getUint16") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_uint16_method, target, "getUint16", NULL, 2, 0);
			}
			if (strcmp(key, "setUint16") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_uint16_method, target, "setUint16", NULL, 3, 0);
			}
			if (strcmp(key, "getInt16") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_int16_method, target, "getInt16", NULL, 2, 0);
			}
			if (strcmp(key, "setInt16") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_int16_method, target, "setInt16", NULL, 3, 0);
			}
			if (strcmp(key, "getUint32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_uint32_method, target, "getUint32", NULL, 2, 0);
			}
			if (strcmp(key, "setUint32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_uint32_method, target, "setUint32", NULL, 3, 0);
			}
			if (strcmp(key, "getInt32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_int32_method, target, "getInt32", NULL, 2, 0);
			}
			if (strcmp(key, "setInt32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_int32_method, target, "setInt32", NULL, 3, 0);
			}
			if (strcmp(key, "getFloat32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_float32_method, target, "getFloat32", NULL, 2, 0);
			}
			if (strcmp(key, "setFloat32") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_float32_method, target, "setFloat32", NULL, 3, 0);
			}
			if (strcmp(key, "getFloat64") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_get_float64_method, target, "getFloat64", NULL, 2, 0);
			}
			if (strcmp(key, "setFloat64") == 0) {
				return jayess_value_from_function((void *)jayess_std_data_view_set_float64_method, target, "setFloat64", NULL, 3, 0);
			}
		}
		if (jayess_std_is_typed_array(target)) {
            if (strcmp(key, "buffer") == 0) {
                return jayess_std_buffer_value_from_state(jayess_std_bytes_state(target));
            }
            if (strcmp(key, "length") == 0 || strcmp(key, "byteLength") == 0) {
                if (strcmp(key, "length") == 0) {
                    return jayess_value_from_number((double)jayess_value_array_length(target));
                }
                {
                    jayess_array *bytes = jayess_std_bytes_slot(target);
                    return jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0));
                }
            }
            if (strcmp(key, "fill") == 0) {
                return jayess_value_from_function((void *)jayess_std_typed_array_fill_method, target, "fill", NULL, 1, 0);
            }
			if (strcmp(key, "includes") == 0) {
				return jayess_value_from_function((void *)jayess_std_typed_array_includes_method, target, "includes", NULL, 1, 0);
			}
			if (strcmp(key, "indexOf") == 0) {
				if (jayess_std_kind_is(target, "Uint8Array")) {
					return jayess_value_from_function((void *)jayess_std_uint8_index_of_method, target, "indexOf", NULL, 1, 0);
				}
				return jayess_value_from_function((void *)jayess_std_typed_array_index_of_method, target, "indexOf", NULL, 1, 0);
			}
			if (strcmp(key, "set") == 0) {
				return jayess_value_from_function((void *)jayess_std_typed_array_set_method, target, "set", NULL, 2, 0);
			}
			if (strcmp(key, "slice") == 0) {
				return jayess_value_from_function((void *)jayess_std_typed_array_slice_method, target, "slice", NULL, 2, 0);
			}
        }
		if (jayess_std_kind_is(target, "Uint8Array")) {
			if (strcmp(key, "startsWith") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_starts_with_method, target, "startsWith", NULL, 1, 0);
			}
			if (strcmp(key, "endsWith") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_ends_with_method, target, "endsWith", NULL, 1, 0);
			}
			if (strcmp(key, "copyWithin") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_copy_within_method, target, "copyWithin", NULL, 3, 0);
			}
            if (strcmp(key, "toString") == 0) {
                return jayess_value_from_function((void *)jayess_std_uint8_to_string_method, target, "toString", NULL, 1, 0);
            }
            if (strcmp(key, "concat") == 0) {
                return jayess_value_from_function((void *)jayess_std_uint8_concat_method, target, "concat", NULL, 1, 1);
            }
            if (strcmp(key, "equals") == 0) {
                return jayess_value_from_function((void *)jayess_std_uint8_equals_method, target, "equals", NULL, 1, 0);
            }
            if (strcmp(key, "compare") == 0) {
                return jayess_value_from_function((void *)jayess_std_uint8_compare_method, target, "compare", NULL, 1, 0);
            }
        }
        if (jayess_std_kind_is(target, "Iterator")) {
            if (strcmp(key, "next") == 0) {
                return jayess_value_from_function((void *)jayess_std_iterator_next_method, target, "next", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "AsyncIterator")) {
            if (strcmp(key, "next") == 0) {
                return jayess_value_from_function((void *)jayess_std_async_iterator_next_method, target, "next", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Promise")) {
            if (strcmp(key, "then") == 0) {
                return jayess_value_from_function((void *)jayess_std_promise_then_method, target, "then", NULL, 2, 0);
            }
            if (strcmp(key, "catch") == 0) {
                return jayess_value_from_function((void *)jayess_std_promise_catch_method, target, "catch", NULL, 1, 0);
            }
            if (strcmp(key, "finally") == 0) {
                return jayess_value_from_function((void *)jayess_std_promise_finally_method, target, "finally", NULL, 1, 0);
            }
        }
        if (jayess_std_kind_is(target, "ReadStream")) {
            if (strcmp(key, "readableEnded") == 0) {
                return jayess_object_get(target->as.object_value, "readableEnded");
            }
            if (strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "read") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_read_method, target, "read", NULL, 1, 0);
            }
            if (strcmp(key, "readBytes") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_read_bytes_method, target, "readBytes", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "pipe") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_pipe_method, target, "pipe", NULL, 1, 0);
            }
            if (strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_read_stream_close_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "HttpBodyStream")) {
            if (strcmp(key, "readableEnded") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "read") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_read_method, target, "read", NULL, 1, 0);
            }
            if (strcmp(key, "readBytes") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_read_bytes_method, target, "readBytes", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "pipe") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_pipe_method, target, "pipe", NULL, 1, 0);
            }
            if (strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_http_body_stream_close_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "WriteStream")) {
            if (strcmp(key, "writableEnded") == 0 || strcmp(key, "writableNeedDrain") == 0 || strcmp(key, "writableLength") == 0 || strcmp(key, "writableHighWaterMark") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_write_stream_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_write_stream_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "write") == 0) {
                return jayess_value_from_function((void *)jayess_std_write_stream_write_method, target, "write", NULL, 1, 0);
            }
            if (strcmp(key, "end") == 0 || strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_write_stream_end_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "CompressionStream")) {
            if (strcmp(key, "readableEnded") == 0 || strcmp(key, "writableEnded") == 0 || strcmp(key, "writableNeedDrain") == 0 || strcmp(key, "writableLength") == 0 || strcmp(key, "writableHighWaterMark") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "read") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_read_method, target, "read", NULL, 1, 0);
            }
            if (strcmp(key, "readBytes") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_read_bytes_method, target, "readBytes", NULL, 1, 0);
            }
            if (strcmp(key, "write") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_write_method, target, "write", NULL, 1, 0);
            }
            if (strcmp(key, "end") == 0 || strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_end_method, target, key, NULL, 0, 0);
            }
            if (strcmp(key, "pipe") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_pipe_method, target, "pipe", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_compression_stream_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Worker")) {
            if (strcmp(key, "closed") == 0) {
                return jayess_object_get(target->as.object_value, "closed");
            }
            if (strcmp(key, "postMessage") == 0) {
                return jayess_value_from_function((void *)jayess_std_worker_post_message_method, target, "postMessage", NULL, 1, 0);
            }
            if (strcmp(key, "receive") == 0) {
                return jayess_value_from_function((void *)jayess_std_worker_receive_method, target, "receive", NULL, 1, 0);
            }
            if (strcmp(key, "terminate") == 0 || strcmp(key, "close") == 0) {
                return jayess_value_from_function((void *)jayess_std_worker_terminate_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Watcher")) {
            if (strcmp(key, "path") == 0 || strcmp(key, "exists") == 0 || strcmp(key, "isDir") == 0 || strcmp(key, "isFile") == 0 || strcmp(key, "size") == 0 || strcmp(key, "mtimeMs") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "poll") == 0) {
                return jayess_value_from_function((void *)jayess_std_fs_watch_poll_method, target, "poll", NULL, 0, 0);
            }
            if (strcmp(key, "pollAsync") == 0) {
                return jayess_value_from_function((void *)jayess_std_fs_watch_poll_async_method, target, "pollAsync", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_fs_watch_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_fs_watch_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_fs_watch_close_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Socket")) {
            if (strcmp(key, "connected") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "readable") == 0 || strcmp(key, "writable") == 0 || strcmp(key, "timeout") == 0 || strcmp(key, "remoteAddress") == 0 || strcmp(key, "remotePort") == 0 || strcmp(key, "remoteFamily") == 0 || strcmp(key, "localAddress") == 0 || strcmp(key, "localPort") == 0 || strcmp(key, "localFamily") == 0 || strcmp(key, "bytesRead") == 0 || strcmp(key, "bytesWritten") == 0 || strcmp(key, "writableLength") == 0 || strcmp(key, "writableHighWaterMark") == 0 || strcmp(key, "writableNeedDrain") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0 || strcmp(key, "secure") == 0 || strcmp(key, "authorized") == 0 || strcmp(key, "backend") == 0 || strcmp(key, "protocol") == 0 || strcmp(key, "alpnProtocol") == 0 || strcmp(key, "alpnProtocols") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "address") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_address_method, target, "address", NULL, 0, 0);
            }
            if (strcmp(key, "remote") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_remote_method, target, "remote", NULL, 0, 0);
            }
            if (strcmp(key, "getPeerCertificate") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_get_peer_certificate_method, target, "getPeerCertificate", NULL, 0, 0);
            }
            if (strcmp(key, "read") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_read_method, target, "read", NULL, 1, 0);
            }
            if (strcmp(key, "readAsync") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_read_async_method, target, "readAsync", NULL, 1, 0);
            }
            if (strcmp(key, "readBytes") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_read_bytes_method, target, "readBytes", NULL, 1, 0);
            }
            if (strcmp(key, "write") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_write_method, target, "write", NULL, 1, 0);
            }
            if (strcmp(key, "writeAsync") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_write_async_method, target, "writeAsync", NULL, 1, 0);
            }
            if (strcmp(key, "setNoDelay") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_set_no_delay_method, target, "setNoDelay", NULL, 1, 0);
            }
            if (strcmp(key, "setKeepAlive") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_set_keep_alive_method, target, "setKeepAlive", NULL, 1, 0);
            }
            if (strcmp(key, "setTimeout") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_set_timeout_method, target, "setTimeout", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "end") == 0 || strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_close_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "DatagramSocket")) {
            if (strcmp(key, "closed") == 0 || strcmp(key, "timeout") == 0 || strcmp(key, "localAddress") == 0 || strcmp(key, "localPort") == 0 || strcmp(key, "localFamily") == 0 || strcmp(key, "bytesRead") == 0 || strcmp(key, "bytesWritten") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0 || strcmp(key, "backend") == 0 || strcmp(key, "protocol") == 0 || strcmp(key, "broadcast") == 0 || strcmp(key, "multicastLoopback") == 0 || strcmp(key, "multicastInterface") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "address") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_address_method, target, "address", NULL, 0, 0);
            }
            if (strcmp(key, "send") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_send_method, target, "send", NULL, 3, 0);
            }
            if (strcmp(key, "receive") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_receive_method, target, "receive", NULL, 1, 0);
            }
            if (strcmp(key, "setBroadcast") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_set_broadcast_method, target, "setBroadcast", NULL, 1, 0);
            }
            if (strcmp(key, "joinGroup") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_join_group_method, target, "joinGroup", NULL, 2, 0);
            }
            if (strcmp(key, "leaveGroup") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_leave_group_method, target, "leaveGroup", NULL, 2, 0);
            }
            if (strcmp(key, "setMulticastInterface") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_set_multicast_interface_method, target, "setMulticastInterface", NULL, 1, 0);
            }
            if (strcmp(key, "setMulticastLoopback") == 0) {
                return jayess_value_from_function((void *)jayess_std_datagram_socket_set_multicast_loopback_method, target, "setMulticastLoopback", NULL, 1, 0);
            }
            if (strcmp(key, "setTimeout") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_set_timeout_method, target, "setTimeout", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
            if (strcmp(key, "close") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_close_method, target, key, NULL, 0, 0);
            }
        }
        if (jayess_std_kind_is(target, "Server")) {
            if (strcmp(key, "listening") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "host") == 0 || strcmp(key, "port") == 0 || strcmp(key, "timeout") == 0 || strcmp(key, "connectionsAccepted") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0) {
                return jayess_object_get(target->as.object_value, key);
            }
            if (strcmp(key, "address") == 0) {
                return jayess_value_from_function((void *)jayess_std_server_address_method, target, "address", NULL, 0, 0);
            }
            if (strcmp(key, "accept") == 0) {
                return jayess_value_from_function((void *)jayess_std_server_accept_method, target, "accept", NULL, 0, 0);
            }
            if (strcmp(key, "acceptAsync") == 0) {
                return jayess_value_from_function((void *)jayess_std_server_accept_async_method, target, "acceptAsync", NULL, 0, 0);
            }
            if (strcmp(key, "close") == 0 || strcmp(key, "end") == 0 || strcmp(key, "destroy") == 0) {
                return jayess_value_from_function((void *)jayess_std_server_close_method, target, key, NULL, 0, 0);
            }
            if (strcmp(key, "setTimeout") == 0) {
                return jayess_value_from_function((void *)jayess_std_server_set_timeout_method, target, "setTimeout", NULL, 1, 0);
            }
            if (strcmp(key, "on") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_on_method, target, "on", NULL, 2, 0);
            }
            if (strcmp(key, "once") == 0) {
                return jayess_value_from_function((void *)jayess_std_socket_once_method, target, "once", NULL, 2, 0);
            }
            if (strcmp(key, "off") == 0 || strcmp(key, "removeListener") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_off_method, target, key, NULL, 2, 0);
            }
            if (strcmp(key, "listenerCount") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_listener_count_method, target, "listenerCount", NULL, 1, 0);
            }
            if (strcmp(key, "eventNames") == 0) {
                return jayess_value_from_function((void *)jayess_std_stream_event_names_method, target, "eventNames", NULL, 0, 0);
            }
        }
        properties = target->as.object_value;
    } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        properties = target->as.function_value->properties;
    }
    if (properties != NULL) {
        getter_key = jayess_accessor_key("__jayess_get_", key);
        if (getter_key != NULL) {
            getter = jayess_object_get(properties, getter_key);
        }
        if (getter != NULL && getter->kind == JAYESS_VALUE_FUNCTION) {
            free(getter_key);
            return jayess_value_call_with_this(getter, target, NULL, 0);
        }
        free(getter_key);
        getter = jayess_object_get(properties, key);
        return getter != NULL ? getter : jayess_value_undefined();
    }
    return NULL;
}

void jayess_value_delete_member(jayess_value *target, const char *key) {
    if (target == NULL) {
        return;
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        jayess_object_delete(target->as.object_value, key);
        return;
    }
    if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        jayess_object_delete(target->as.function_value->properties, key);
    }
}

jayess_value *jayess_value_object_keys(jayess_value *target) {
    if (target == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        return jayess_value_from_array(jayess_object_keys(target->as.object_value));
    }
    if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        return jayess_value_from_array(jayess_object_keys(target->as.function_value->properties));
    }
    return jayess_value_from_array(jayess_array_new());
}

jayess_value *jayess_value_object_symbols(jayess_value *target) {
    jayess_object *properties = NULL;
    jayess_array *symbols = jayess_array_new();
    jayess_object_entry *current;
    if (target == NULL) {
        return jayess_value_from_array(symbols);
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        properties = target->as.object_value;
    } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        properties = target->as.function_value->properties;
    }
    if (properties == NULL) {
        return jayess_value_from_array(symbols);
    }
    for (current = properties->head; current != NULL; current = current->next) {
        if (jayess_object_entry_is_symbol(current)) {
            jayess_array_push_value(symbols, current->key_value);
        }
    }
    return jayess_value_from_array(symbols);
}

jayess_value *jayess_value_object_rest(jayess_value *target, jayess_value *excluded_keys) {
    jayess_object *source;
    jayess_object *copy;
    jayess_object_entry *current;

    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return jayess_value_from_object(jayess_object_new());
    }

    source = target->as.object_value;
    copy = jayess_object_new();
    if (copy == NULL) {
        return jayess_value_from_object(NULL);
    }

    for (current = source->head; current != NULL; current = current->next) {
        jayess_value *key_value;
        const char *key;
        int skip = 0;
        int j;

        if (current->key != NULL && strncmp(current->key, "__jayess_", 10) == 0) {
            continue;
        }
        if (current->key != NULL) {
            key_value = jayess_value_from_string(current->key);
        } else {
            key_value = current->key_value;
        }
        if (key_value == NULL) {
            continue;
        }
        key = key_value->kind == JAYESS_VALUE_STRING ? key_value->as.string_value : NULL;
        if (excluded_keys != NULL && excluded_keys->kind == JAYESS_VALUE_ARRAY && excluded_keys->as.array_value != NULL) {
            for (j = 0; j < excluded_keys->as.array_value->count; j++) {
                jayess_value *excluded = excluded_keys->as.array_value->values[j];
                if (excluded != NULL && jayess_value_eq(key_value, excluded)) {
                    skip = 1;
                    break;
                }
            }
        }
        if (!skip) {
            if (key != NULL) {
                jayess_object_set_value(copy, key, current->value);
            } else {
                jayess_object_set_key_value(copy, key_value, current->value);
            }
        }
    }

    return jayess_value_from_object(copy);
}

jayess_value *jayess_value_iterable_values(jayess_value *target) {
    if (target == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    if (target->kind == JAYESS_VALUE_ARRAY) {
        return jayess_value_from_array(jayess_array_clone(target->as.array_value));
    }
    if (target->kind == JAYESS_VALUE_STRING) {
        return jayess_std_iterable_protocol_values(target);
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        if (jayess_std_kind_is(target, "Map")) {
            return jayess_std_map_entries_method(target);
        }
        if (jayess_std_kind_is(target, "Set")) {
            return jayess_std_set_values_method(target);
        }
        if (jayess_std_is_typed_array(target)) {
            jayess_array *values = jayess_array_new();
            int i;
            int length = jayess_value_array_length(target);
            for (i = 0; i < length; i++) {
                jayess_array_push_value(values, jayess_value_from_number(jayess_std_typed_array_get_number(target, i)));
            }
            return jayess_value_from_array(values);
        }
        if (jayess_std_kind_is(target, "Iterator")) {
            jayess_value *values = jayess_object_get(target->as.object_value, "__jayess_iterator_values");
            if (values != NULL && values->kind == JAYESS_VALUE_ARRAY) {
                return jayess_value_from_array(jayess_array_clone(values->as.array_value));
            }
        }
    }
    return jayess_std_iterable_protocol_values(target);
}

jayess_value *jayess_value_object_values(jayess_value *target) {
    jayess_value *keys_value = jayess_value_object_keys(target);
    jayess_array *values = jayess_array_new();
    int i;
    if (keys_value == NULL || keys_value->kind != JAYESS_VALUE_ARRAY || keys_value->as.array_value == NULL) {
        return jayess_value_from_array(values);
    }
    for (i = 0; i < keys_value->as.array_value->count; i++) {
        jayess_value *key = keys_value->as.array_value->values[i];
        if (key != NULL && key->kind == JAYESS_VALUE_STRING) {
            jayess_array_push_value(values, jayess_value_get_member(target, key->as.string_value));
        }
    }
    return jayess_value_from_array(values);
}

jayess_value *jayess_value_object_entries(jayess_value *target) {
    jayess_value *keys_value = jayess_value_object_keys(target);
    jayess_array *entries = jayess_array_new();
    int i;
    if (keys_value == NULL || keys_value->kind != JAYESS_VALUE_ARRAY || keys_value->as.array_value == NULL) {
        return jayess_value_from_array(entries);
    }
    for (i = 0; i < keys_value->as.array_value->count; i++) {
        jayess_value *key = keys_value->as.array_value->values[i];
        if (key != NULL && key->kind == JAYESS_VALUE_STRING) {
            jayess_array *pair = jayess_array_new();
            jayess_array_push_value(pair, key);
            jayess_array_push_value(pair, jayess_value_get_member(target, key->as.string_value));
            jayess_array_push_value(entries, jayess_value_from_array(pair));
        }
    }
    return jayess_value_from_array(entries);
}

jayess_value *jayess_value_object_assign(jayess_value *target, jayess_value *source) {
    jayess_object *properties = NULL;
    jayess_object_entry *current;
    if (target == NULL || source == NULL) {
        return target != NULL ? target : jayess_value_undefined();
    }
    if (source->kind == JAYESS_VALUE_OBJECT) {
        properties = source->as.object_value;
    } else if (source->kind == JAYESS_VALUE_FUNCTION && source->as.function_value != NULL) {
        properties = source->as.function_value->properties;
    }
    if (properties == NULL) {
        return target;
    }
    for (current = properties->head; current != NULL; current = current->next) {
        if (current->key != NULL) {
            if (strncmp(current->key, "__jayess_", 10) == 0) {
                continue;
            }
            jayess_value_set_member(target, current->key, current->value);
        } else if (jayess_object_entry_is_symbol(current)) {
            if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
                jayess_object_set_key_value(target->as.object_value, current->key_value, current->value);
            } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
                jayess_object_set_key_value(target->as.function_value->properties, current->key_value, current->value);
            }
        }
    }
    return target;
}

jayess_value *jayess_value_object_has_own(jayess_value *target, jayess_value *key) {
    jayess_value *property_key;
    jayess_value *value = NULL;
    if (target == NULL || key == NULL) {
        return jayess_value_from_bool(0);
    }
    property_key = jayess_value_to_property_key(key);
    if (property_key == NULL) {
        return jayess_value_from_bool(0);
    }
    if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
        value = jayess_object_get_key_value(target->as.object_value, property_key);
    } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        value = jayess_object_get_key_value(target->as.function_value->properties, property_key);
    }
    return jayess_value_from_bool(value != NULL);
}

double jayess_math_floor(double value) { return floor(value); }
double jayess_math_ceil(double value) { return ceil(value); }
double jayess_math_round(double value) { return floor(value + 0.5); }
double jayess_math_min(double left, double right) { return left < right ? left : right; }
double jayess_math_max(double left, double right) { return left > right ? left : right; }
double jayess_math_abs(double value) { return fabs(value); }
double jayess_math_pow(double left, double right) { return pow(left, right); }
double jayess_math_sqrt(double value) { return sqrt(value); }
double jayess_math_random(void) {
    static int seeded = 0;
    if (!seeded) {
        srand((unsigned int)time(NULL));
        seeded = 1;
    }
    return (double)rand() / (double)RAND_MAX;
}

jayess_value *jayess_std_process_cwd(void) {
    char buffer[4096];
#ifdef _WIN32
    if (_getcwd(buffer, sizeof(buffer)) == NULL) {
#else
    if (getcwd(buffer, sizeof(buffer)) == NULL) {
#endif
        return jayess_value_undefined();
    }
    return jayess_value_from_string(buffer);
}

jayess_value *jayess_std_process_env(jayess_value *name) {
    char *key = jayess_value_stringify(name);
    char *value;
    jayess_value *result;
    if (key == NULL) {
        return jayess_value_undefined();
    }
    value = getenv(key);
    free(key);
    if (value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_value_from_string(value);
    return result;
}

#include "jayess_runtime_network.c"
jayess_value *jayess_std_fs_read_file(jayess_value *path, jayess_value *encoding) {
    char *path_text = jayess_value_stringify(path);
    FILE *file;
    long length;
    char *buffer;
    jayess_value *result;
    char *encoding_text = NULL;
    if (path_text == NULL) {
        return jayess_value_undefined();
    }
    if (encoding != NULL && !jayess_value_is_nullish(encoding)) {
        encoding_text = jayess_value_stringify(encoding);
        if (encoding_text != NULL &&
            strcmp(encoding_text, "utf8") != 0 &&
            strcmp(encoding_text, "utf-8") != 0 &&
            strcmp(encoding_text, "text") != 0) {
            free(path_text);
            free(encoding_text);
            return jayess_value_undefined();
        }
    }
    file = fopen(path_text, "rb");
    free(path_text);
    free(encoding_text);
    if (file == NULL) {
        return jayess_value_undefined();
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        return jayess_value_undefined();
    }
    length = ftell(file);
    if (length < 0 || fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return jayess_value_undefined();
    }
    buffer = (char *)malloc((size_t)length + 1);
    if (buffer == NULL) {
        fclose(file);
        return jayess_value_undefined();
    }
    if (fread(buffer, 1, (size_t)length, file) != (size_t)length) {
        free(buffer);
        fclose(file);
        return jayess_value_undefined();
    }
    buffer[length] = '\0';
    fclose(file);
    result = jayess_value_from_string(buffer);
    free(buffer);
    return result;
}

jayess_value *jayess_std_fs_read_file_async(jayess_value *path, jayess_value *encoding) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_fs_read_file_task(promise, path, encoding);
    return promise;
}

jayess_value *jayess_std_fs_write_file(jayess_value *path, jayess_value *content) {
    char *path_text = jayess_value_stringify(path);
    char *text = jayess_value_stringify(content);
    FILE *file;
    size_t length;
    jayess_value *result;
    if (path_text == NULL || text == NULL) {
        free(path_text);
        free(text);
        return jayess_value_from_bool(0);
    }
    file = fopen(path_text, "wb");
    free(path_text);
    if (file == NULL) {
        free(text);
        return jayess_value_from_bool(0);
    }
    length = strlen(text);
    result = jayess_value_from_bool(fwrite(text, 1, length, file) == length);
    fclose(file);
    free(text);
    return result;
}

jayess_value *jayess_std_fs_append_file(jayess_value *path, jayess_value *content) {
    char *path_text = jayess_value_stringify(path);
    char *text = jayess_value_stringify(content);
    FILE *file;
    size_t length;
    jayess_value *result;
    if (path_text == NULL || text == NULL) {
        free(path_text);
        free(text);
        return jayess_value_from_bool(0);
    }
    file = fopen(path_text, "ab");
    free(path_text);
    if (file == NULL) {
        free(text);
        return jayess_value_from_bool(0);
    }
    length = strlen(text);
    result = jayess_value_from_bool(fwrite(text, 1, length, file) == length);
    fclose(file);
    free(text);
    return result;
}

jayess_value *jayess_std_fs_write_file_async(jayess_value *path, jayess_value *content) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_fs_write_file_task(promise, path, content);
    return promise;
}

jayess_value *jayess_std_fs_create_read_stream(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    FILE *file;
    jayess_object *object;
    if (path_text == NULL) {
        return jayess_std_fs_stream_open_error("ReadStream", "stream path must be convertible to text");
    }
    file = fopen(path_text, "rb");
    free(path_text);
    if (file == NULL) {
        return jayess_std_fs_stream_open_error("ReadStream", "failed to open read stream");
    }
    object = jayess_object_new();
    if (object == NULL) {
        fclose(file);
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("ReadStream"));
    object->stream_file = file;
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "error", jayess_value_null());
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_fs_create_write_stream(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    FILE *file;
    jayess_object *object;
    if (path_text == NULL) {
        return jayess_std_fs_stream_open_error("WriteStream", "stream path must be convertible to text");
    }
    file = fopen(path_text, "wb");
    free(path_text);
    if (file == NULL) {
        return jayess_std_fs_stream_open_error("WriteStream", "failed to open write stream");
    }
    object = jayess_object_new();
    if (object == NULL) {
        fclose(file);
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("WriteStream"));
    object->stream_file = file;
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableNeedDrain", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableLength", jayess_value_from_number(0));
    jayess_object_set_value(object, "writableHighWaterMark", jayess_value_from_number(JAYESS_STD_STREAM_DEFAULT_HIGH_WATER_MARK));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "error", jayess_value_null());
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_fs_exists(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    FILE *file;
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_bool(0);
    }
    file = fopen(path_text, "rb");
    free(path_text);
    result = jayess_value_from_bool(file != NULL);
    if (file != NULL) {
        fclose(file);
    }
    return result;
}

jayess_value *jayess_std_fs_read_dir(jayess_value *path, jayess_value *options) {
    char *path_text = jayess_value_stringify(path);
    jayess_array *entries = jayess_array_new();
    int recursive = jayess_object_option_bool(options, "recursive");
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_array(entries);
    }
    jayess_fs_read_dir_collect(entries, path_text, recursive);
    free(path_text);
    result = jayess_value_from_array(entries);
    return result;
}

jayess_value *jayess_std_fs_stat(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    jayess_object *entry;
    int is_dir;
    if (path_text == NULL || !jayess_path_exists_text(path_text)) {
        free(path_text);
        return jayess_value_undefined();
    }
    entry = jayess_object_new();
    if (entry == NULL) {
        free(path_text);
        return jayess_value_undefined();
    }
    is_dir = jayess_path_is_dir_text(path_text);
    jayess_object_set_value(entry, "path", jayess_value_from_string(path_text));
    jayess_object_set_value(entry, "isDir", jayess_value_from_bool(is_dir));
    jayess_object_set_value(entry, "isFile", jayess_value_from_bool(!is_dir));
    jayess_object_set_value(entry, "size", jayess_value_from_number(jayess_path_file_size_text(path_text)));
    jayess_object_set_value(entry, "mtimeMs", jayess_value_from_number(jayess_path_modified_time_ms_text(path_text)));
    jayess_object_set_value(entry, "permissions", jayess_value_from_string(jayess_path_permissions_text(path_text)));
    free(path_text);
    return jayess_value_from_object(entry);
}

jayess_value *jayess_std_fs_mkdir(jayess_value *path, jayess_value *options) {
    char *path_text = jayess_value_stringify(path);
    int ok = 0;
    int recursive = jayess_object_option_bool(options, "recursive");
    if (path_text == NULL) {
        return jayess_value_from_bool(0);
    }
    if (!recursive) {
        ok = jayess_path_mkdir_single(path_text);
    } else {
        int root_length = jayess_path_root_length(path_text);
        jayess_array *segments = jayess_path_split_segments(path_text);
        jayess_array *built = jayess_array_new();
        char root[4] = {0};
        int i;
        if (root_length > 0) {
            memcpy(root, path_text, (size_t)root_length < sizeof(root) - 1 ? (size_t)root_length : sizeof(root) - 1);
        }
        ok = 1;
        for (i = 0; i < segments->count; i++) {
            char *current;
            jayess_array_push_value(built, jayess_array_get(segments, i));
            current = jayess_path_join_segments_with_root(root, built, jayess_path_preferred_separator_char(path_text));
            if (current == NULL || !jayess_path_mkdir_single(current)) {
                ok = 0;
                free(current);
                break;
            }
            free(current);
        }
    }
    free(path_text);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_fs_remove(jayess_value *path, jayess_value *options) {
    char *path_text = jayess_value_stringify(path);
    int ok = 0;
    int recursive = jayess_object_option_bool(options, "recursive");
    if (path_text == NULL) {
        return jayess_value_from_bool(0);
    }
    if (recursive) {
        ok = jayess_fs_remove_path_recursive(path_text);
    } else if (jayess_path_is_dir_text(path_text)) {
#ifdef _WIN32
        ok = (_rmdir(path_text) == 0);
#else
        ok = (rmdir(path_text) == 0);
#endif
    } else {
        ok = (remove(path_text) == 0);
    }
    free(path_text);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_fs_copy_file(jayess_value *from, jayess_value *to) {
    char *from_text = jayess_value_stringify(from);
    char *to_text = jayess_value_stringify(to);
    FILE *source;
    FILE *dest;
    char buffer[4096];
    size_t read_bytes;
    int ok = 1;
    if (from_text == NULL || to_text == NULL) {
        free(from_text);
        free(to_text);
        return jayess_value_from_bool(0);
    }
    source = fopen(from_text, "rb");
    if (source == NULL) {
        free(from_text);
        free(to_text);
        return jayess_value_from_bool(0);
    }
    dest = fopen(to_text, "wb");
    if (dest == NULL) {
        fclose(source);
        free(from_text);
        free(to_text);
        return jayess_value_from_bool(0);
    }
    while ((read_bytes = fread(buffer, 1, sizeof(buffer), source)) > 0) {
        if (fwrite(buffer, 1, read_bytes, dest) != read_bytes) {
            ok = 0;
            break;
        }
    }
    if (ferror(source)) {
        ok = 0;
    }
    fclose(source);
    fclose(dest);
    free(from_text);
    free(to_text);
    return jayess_value_from_bool(ok);
}

static int jayess_fs_copy_dir_recursive(const char *from_text, const char *to_text) {
    if (from_text == NULL || to_text == NULL || !jayess_path_is_dir_text(from_text)) {
        return 0;
    }
    if (!jayess_path_mkdir_single(to_text) && !jayess_path_is_dir_text(to_text)) {
        return 0;
    }
#ifdef _WIN32
    {
        WIN32_FIND_DATAA find_data;
        HANDLE handle;
        size_t from_len = strlen(from_text);
        size_t to_len = strlen(to_text);
        char *pattern = (char *)malloc(from_len + 3);
        int ok = 1;
        if (pattern == NULL) {
            return 0;
        }
        strcpy(pattern, from_text);
        if (from_len > 0 && !jayess_path_is_separator(pattern[from_len - 1])) {
            strcat(pattern, "\\");
        }
        strcat(pattern, "*");
        handle = FindFirstFileA(pattern, &find_data);
        free(pattern);
        if (handle == INVALID_HANDLE_VALUE) {
            return 0;
        }
        do {
            char *from_path;
            char *to_path;
            int is_dir;
            if (strcmp(find_data.cFileName, ".") == 0 || strcmp(find_data.cFileName, "..") == 0) {
                continue;
            }
            from_path = (char *)malloc(from_len + strlen(find_data.cFileName) + 3);
            to_path = (char *)malloc(to_len + strlen(find_data.cFileName) + 3);
            if (from_path == NULL || to_path == NULL) {
                free(from_path);
                free(to_path);
                ok = 0;
                continue;
            }
            strcpy(from_path, from_text);
            if (from_len > 0 && !jayess_path_is_separator(from_path[from_len - 1])) {
                strcat(from_path, "\\");
            }
            strcat(from_path, find_data.cFileName);
            strcpy(to_path, to_text);
            if (to_len > 0 && !jayess_path_is_separator(to_path[to_len - 1])) {
                strcat(to_path, "\\");
            }
            strcat(to_path, find_data.cFileName);
            is_dir = (find_data.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0;
            if (is_dir) {
                if (!jayess_fs_copy_dir_recursive(from_path, to_path)) {
                    ok = 0;
                }
            } else if (!jayess_value_as_bool(jayess_std_fs_copy_file(jayess_value_from_string(from_path), jayess_value_from_string(to_path)))) {
                ok = 0;
            }
            free(from_path);
            free(to_path);
        } while (FindNextFileA(handle, &find_data));
        FindClose(handle);
        return ok;
    }
#else
    {
        DIR *dir = opendir(from_text);
        struct dirent *entry;
        size_t from_len = strlen(from_text);
        size_t to_len = strlen(to_text);
        int ok = 1;
        if (dir == NULL) {
            return 0;
        }
        while ((entry = readdir(dir)) != NULL) {
            char *from_path;
            char *to_path;
            int is_dir;
            if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
                continue;
            }
            from_path = (char *)malloc(from_len + strlen(entry->d_name) + 3);
            to_path = (char *)malloc(to_len + strlen(entry->d_name) + 3);
            if (from_path == NULL || to_path == NULL) {
                free(from_path);
                free(to_path);
                ok = 0;
                continue;
            }
            strcpy(from_path, from_text);
            if (from_len > 0 && !jayess_path_is_separator(from_path[from_len - 1])) {
                strcat(from_path, "/");
            }
            strcat(from_path, entry->d_name);
            strcpy(to_path, to_text);
            if (to_len > 0 && !jayess_path_is_separator(to_path[to_len - 1])) {
                strcat(to_path, "/");
            }
            strcat(to_path, entry->d_name);
            is_dir = jayess_path_is_dir_text(from_path);
            if (is_dir) {
                if (!jayess_fs_copy_dir_recursive(from_path, to_path)) {
                    ok = 0;
                }
            } else if (!jayess_value_as_bool(jayess_std_fs_copy_file(jayess_value_from_string(from_path), jayess_value_from_string(to_path)))) {
                ok = 0;
            }
            free(from_path);
            free(to_path);
        }
        closedir(dir);
        return ok;
    }
#endif
}

jayess_value *jayess_std_fs_copy_dir(jayess_value *from, jayess_value *to) {
    char *from_text = jayess_value_stringify(from);
    char *to_text = jayess_value_stringify(to);
    int ok = jayess_fs_copy_dir_recursive(from_text, to_text);
    free(from_text);
    free(to_text);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_fs_rename(jayess_value *from, jayess_value *to) {
    char *from_text = jayess_value_stringify(from);
    char *to_text = jayess_value_stringify(to);
    int ok = 0;
    if (from_text != NULL && to_text != NULL) {
        ok = rename(from_text, to_text) == 0;
    }
    free(from_text);
    free(to_text);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_fs_symlink(jayess_value *target, jayess_value *path) {
    char *target_text = jayess_value_stringify(target);
    char *path_text = jayess_value_stringify(path);
    int ok = 0;
    if (target_text == NULL || path_text == NULL) {
        free(target_text);
        free(path_text);
        return jayess_value_from_bool(0);
    }
#ifdef _WIN32
    {
        DWORD flags = 0;
        if (jayess_path_is_dir_text(target_text)) {
            flags |= SYMBOLIC_LINK_FLAG_DIRECTORY;
        }
#ifdef SYMBOLIC_LINK_FLAG_ALLOW_UNPRIVILEGED_CREATE
        flags |= SYMBOLIC_LINK_FLAG_ALLOW_UNPRIVILEGED_CREATE;
#endif
        ok = CreateSymbolicLinkA(path_text, target_text, flags) != 0;
    }
#else
    ok = symlink(target_text, path_text) == 0;
#endif
    free(target_text);
    free(path_text);
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_fs_watch(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    jayess_object *object;
    jayess_value *watcher;
    jayess_fs_watch_state *state;
    int exists;
    int is_dir;
    double size;
    double mtime_ms;
    if (path_text == NULL) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        free(path_text);
        return jayess_value_undefined();
    }
    state = (jayess_fs_watch_state *)malloc(sizeof(jayess_fs_watch_state));
    if (state == NULL) {
        free(path_text);
        return jayess_value_undefined();
    }
    jayess_fs_watch_snapshot_text(path_text, &exists, &is_dir, &size, &mtime_ms);
    state->path = path_text;
    state->exists = exists;
    state->is_dir = is_dir;
    state->size = size;
    state->mtime_ms = mtime_ms;
    state->closed = 0;
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Watcher"));
    jayess_object_set_value(object, "path", jayess_value_from_string(path_text));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "error", jayess_value_null());
    watcher = jayess_value_from_object(object);
    jayess_std_fs_watch_apply_snapshot(watcher, exists, is_dir, size, mtime_ms);
    return watcher;
}

jayess_value *jayess_std_number_is_nan(jayess_value *value) {
    return jayess_value_from_bool(isnan(jayess_value_to_number(value)));
}

jayess_value *jayess_std_number_is_finite(jayess_value *value) {
    return jayess_value_from_bool(isfinite(jayess_value_to_number(value)));
}

jayess_value *jayess_std_string_from_char_code(jayess_value *codes) {
    int count = 0;
    char *out;
    int i;
    if (codes != NULL && codes->kind == JAYESS_VALUE_ARRAY && codes->as.array_value != NULL) {
        count = codes->as.array_value->count;
    }
    out = (char *)malloc((size_t)count + 1);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < count; i++) {
        jayess_value *code = jayess_array_get(codes->as.array_value, i);
        int numeric = (int)jayess_value_to_number(code);
        out[i] = (char)(numeric & 0xFF);
    }
    out[count] = '\0';
    codes = jayess_value_from_string(out);
    free(out);
    return codes;
}

jayess_value *jayess_std_array_is_array(jayess_value *value) {
    return jayess_value_from_bool(value != NULL && value->kind == JAYESS_VALUE_ARRAY);
}

jayess_value *jayess_std_array_from(jayess_value *value) {
    return jayess_value_iterable_values(value);
}

jayess_value *jayess_std_array_of(jayess_value *values) {
    if (values != NULL && values->kind == JAYESS_VALUE_ARRAY && values->as.array_value != NULL) {
        return jayess_value_from_array(jayess_array_clone(values->as.array_value));
    }
    return jayess_value_from_array(jayess_array_new());
}

jayess_value *jayess_std_object_from_entries(jayess_value *entries) {
    jayess_object *object = jayess_object_new();
    int i;
    if (entries == NULL || entries->kind != JAYESS_VALUE_ARRAY || entries->as.array_value == NULL) {
        return jayess_value_from_object(object);
    }
    for (i = 0; i < entries->as.array_value->count; i++) {
        jayess_value *entry = jayess_array_get(entries->as.array_value, i);
        if (entry != NULL && entry->kind == JAYESS_VALUE_ARRAY && entry->as.array_value != NULL && entry->as.array_value->count >= 2) {
            jayess_value *key = jayess_value_to_property_key(jayess_array_get(entry->as.array_value, 0));
            jayess_value *value = jayess_array_get(entry->as.array_value, 1);
            if (key != NULL) {
                jayess_object_set_key_value(object, key, value);
            }
        }
    }
    return jayess_value_from_object(object);
}

void jayess_value_set_computed_member(jayess_value *target, jayess_value *key, jayess_value *value) {
    jayess_value *property_key;
    if (target == NULL || key == NULL || value == NULL) {
        return;
    }
    property_key = jayess_value_to_property_key(key);
    if (property_key == NULL) {
        return;
    }
    if (property_key->kind == JAYESS_VALUE_STRING) {
        jayess_value_set_member(target, property_key->as.string_value, value);
        return;
    }
    if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
        jayess_object_set_key_value(target->as.object_value, property_key, value);
        return;
    }
    if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        jayess_object_set_key_value(target->as.function_value->properties, property_key, value);
    }
}

void jayess_array_free_unshared(jayess_array *array) {
    int index;
    if (array == NULL) {
        return;
    }
    for (index = 0; index < array->count; index++) {
        if (array->values[index] != NULL) {
            jayess_value_free_unshared(array->values[index]);
        }
    }
    if (jayess_runtime_accounting_state.array_slots >= (size_t)array->count) {
        jayess_runtime_accounting_state.array_slots -= (size_t)array->count;
    } else {
        jayess_runtime_accounting_state.array_slots = 0;
    }
    if (jayess_runtime_accounting_state.arrays > 0) {
        jayess_runtime_accounting_state.arrays--;
    }
    free(array->values);
    free(array);
}

void jayess_value_set_index(jayess_value *target, int index, jayess_value *value) {
    if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_is_typed_array(target)) {
        jayess_std_typed_array_set_number(target, index, jayess_value_to_number(value));
        return;
    }
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY) {
        return;
    }
    jayess_array_set_value(target->as.array_value, index, value);
}

jayess_value *jayess_value_get_index(jayess_value *target, int index) {
    if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_is_typed_array(target)) {
        return jayess_value_from_number(jayess_std_typed_array_get_number(target, index));
    }
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
    return jayess_array_get(target->as.array_value, index);
}

void jayess_value_set_dynamic_index(jayess_value *target, jayess_value *index, jayess_value *value) {
    jayess_value *property_key;
    if (target == NULL || index == NULL || value == NULL) {
        return;
    }

    if (index->kind == JAYESS_VALUE_SYMBOL) {
        if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
            jayess_object_set_key_value(target->as.object_value, index, value);
        } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
            jayess_object_set_key_value(target->as.function_value->properties, index, value);
        }
        return;
    }

    if (index->kind == JAYESS_VALUE_STRING) {
        jayess_value_set_member(target, index->as.string_value, value);
        return;
    }

    if (index->kind == JAYESS_VALUE_NUMBER) {
        jayess_value_set_index(target, (int)index->as.number_value, value);
        return;
    }

    if (index->kind == JAYESS_VALUE_BOOL) {
        jayess_value_set_index(target, index->as.bool_value ? 1 : 0, value);
        return;
    }

    if (index->kind == JAYESS_VALUE_NULL || index->kind == JAYESS_VALUE_UNDEFINED) {
        return;
    }

    property_key = jayess_value_to_property_key(index);
    if (property_key == NULL || property_key->kind != JAYESS_VALUE_STRING) {
        return;
    }
    if (target->kind == JAYESS_VALUE_OBJECT || target->kind == JAYESS_VALUE_FUNCTION) {
        jayess_value_set_member(target, property_key->as.string_value, value);
    }
}

jayess_value *jayess_value_get_dynamic_index(jayess_value *target, jayess_value *index) {
    jayess_value *property_key;
    if (target == NULL || index == NULL) {
        return NULL;
    }

    if (index->kind == JAYESS_VALUE_SYMBOL) {
        if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
            jayess_value *value = jayess_object_get_key_value(target->as.object_value, index);
            return value != NULL ? value : jayess_value_undefined();
        }
        if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
            jayess_value *value = jayess_object_get_key_value(target->as.function_value->properties, index);
            return value != NULL ? value : jayess_value_undefined();
        }
        return NULL;
    }

    if (index->kind == JAYESS_VALUE_STRING) {
        return jayess_value_get_member(target, index->as.string_value);
    }

    if (index->kind == JAYESS_VALUE_NUMBER) {
        return jayess_value_get_index(target, (int)index->as.number_value);
    }

    if (index->kind == JAYESS_VALUE_BOOL) {
        return jayess_value_get_index(target, index->as.bool_value ? 1 : 0);
    }

    property_key = jayess_value_to_property_key(index);
    if (property_key != NULL && property_key->kind == JAYESS_VALUE_STRING && (target->kind == JAYESS_VALUE_OBJECT || target->kind == JAYESS_VALUE_FUNCTION)) {
        return jayess_value_get_member(target, property_key->as.string_value);
    }
    return NULL;
}

void jayess_value_delete_dynamic_index(jayess_value *target, jayess_value *index) {
    jayess_value *property_key;
    if (target == NULL || index == NULL) {
        return;
    }

    if (index->kind == JAYESS_VALUE_SYMBOL) {
        if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
            jayess_object_delete_key_value(target->as.object_value, index);
        } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
            jayess_object_delete_key_value(target->as.function_value->properties, index);
        }
        return;
    }

    if (index->kind == JAYESS_VALUE_STRING) {
        jayess_value_delete_member(target, index->as.string_value);
        return;
    }
    property_key = jayess_value_to_property_key(index);
    if (property_key != NULL && property_key->kind == JAYESS_VALUE_STRING) {
        jayess_value_delete_member(target, property_key->as.string_value);
    }
}

int jayess_value_array_length(jayess_value *target) {
    if (target == NULL) {
        return 0;
    }
    if (target->kind == JAYESS_VALUE_STRING && target->as.string_value != NULL) {
        return (int)strlen(target->as.string_value);
    }
    if (target->kind == JAYESS_VALUE_ARRAY && target->as.array_value != NULL) {
        return target->as.array_value->count;
    }
    if (target->kind == JAYESS_VALUE_OBJECT && jayess_std_is_typed_array(target)) {
        return jayess_std_typed_array_length_from_bytes(jayess_std_bytes_slot(target), jayess_std_typed_array_kind(target));
    }
    return 0;
}

jayess_value *jayess_value_array_push(jayess_value *target, jayess_value *value) {
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)jayess_array_push_value(target->as.array_value, value));
}

jayess_value *jayess_value_array_pop(jayess_value *target) {
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_array_pop_value(target->as.array_value);
}

jayess_value *jayess_value_array_shift(jayess_value *target) {
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_array_shift_value(target->as.array_value);
}

jayess_value *jayess_value_array_unshift(jayess_value *target, jayess_value *value) {
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)jayess_array_unshift_value(target->as.array_value, value));
}

jayess_value *jayess_value_array_slice(jayess_value *target, int start, int end, int has_end) {
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_is_typed_array(target)) {
            return jayess_std_typed_array_slice_values(target, start, end, has_end);
        }
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_array(jayess_array_slice_values(target->as.array_value, start, end, has_end));
}

jayess_value *jayess_value_array_includes(jayess_value *target, jayess_value *value) {
    int i;
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_from_bool(0);
    }
    for (i = 0; i < target->as.array_value->count; i++) {
        if (jayess_value_eq(target->as.array_value->values[i], value)) {
            return jayess_value_from_bool(1);
        }
    }
    return jayess_value_from_bool(0);
}

jayess_value *jayess_value_array_join(jayess_value *target, jayess_value *separator) {
    const char *sep = ",";
    size_t total = 1;
    char *out;
    int i;
    if (separator != NULL && separator->kind == JAYESS_VALUE_STRING) {
        sep = separator->as.string_value;
    }
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY || target->as.array_value == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < target->as.array_value->count; i++) {
        char *text = jayess_value_stringify(target->as.array_value->values[i]);
        total += strlen(text);
        if (i > 0) {
            total += strlen(sep);
        }
        free(text);
    }
    out = (char *)malloc(total);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    for (i = 0; i < target->as.array_value->count; i++) {
        char *text = jayess_value_stringify(target->as.array_value->values[i]);
        if (i > 0) {
            strcat(out, sep);
        }
        strcat(out, text);
        free(text);
    }
    separator = jayess_value_from_string(out);
    free(out);
    return separator;
}

jayess_value *jayess_value_from_string(const char *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_STRING;
    boxed->as.string_value = jayess_strdup(value != NULL ? value : "");
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.strings++;
    return boxed;
}

jayess_value *jayess_value_from_symbol(const char *description) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    jayess_symbol *symbol_value;
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_SYMBOL;
    symbol_value = (jayess_symbol *)malloc(sizeof(jayess_symbol));
    if (symbol_value == NULL) {
        free(boxed);
        return NULL;
    }
    symbol_value->id = jayess_next_symbol_id++;
    symbol_value->description = description != NULL ? jayess_strdup(description) : NULL;
    boxed->as.symbol_value = symbol_value;
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.symbols++;
    return boxed;
}

jayess_value *jayess_value_from_bytes_copy(const unsigned char *bytes, size_t length) {
    return jayess_std_uint8_array_from_bytes(bytes != NULL ? bytes : (const unsigned char *)"", length);
}

unsigned char *jayess_value_to_bytes_copy(jayess_value *value, size_t *length_out) {
    jayess_array *bytes = jayess_std_bytes_slot(value);
    unsigned char *copy;
    size_t i;
    if (length_out != NULL) {
        *length_out = 0;
    }
    if (bytes == NULL || bytes->count <= 0) {
        return NULL;
    }
    copy = (unsigned char *)malloc((size_t)bytes->count);
    if (copy == NULL) {
        return NULL;
    }
    for (i = 0; i < (size_t)bytes->count; i++) {
        copy[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, (int)i)) & 255);
    }
    if (length_out != NULL) {
        *length_out = (size_t)bytes->count;
    }
    return copy;
}

char *jayess_value_to_string_copy(jayess_value *value) {
    if (value == NULL) {
        return jayess_strdup("");
    }
    if (value->kind == JAYESS_VALUE_STRING) {
        return jayess_strdup(value->as.string_value != NULL ? value->as.string_value : "");
    }
    return jayess_value_stringify(value);
}

void jayess_string_free(char *text) {
    free(text);
}

void jayess_bytes_free(void *bytes) {
    free(bytes);
}

jayess_value *jayess_value_from_native_handle(const char *kind, void *handle) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    object->native_handle = handle;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("NativeHandle"));
    jayess_object_set_value(object, "kind", jayess_value_from_string(kind != NULL ? kind : ""));
    jayess_runtime_accounting_state.native_handle_wrappers++;
    return jayess_value_from_object(object);
}

jayess_value *jayess_value_from_managed_native_handle(const char *kind, void *handle, jayess_native_handle_finalizer finalizer) {
    jayess_object *object = jayess_object_new();
    jayess_managed_native_handle *managed;
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    managed = (jayess_managed_native_handle *)malloc(sizeof(jayess_managed_native_handle));
    if (managed == NULL) {
        return jayess_value_from_object(NULL);
    }
    managed->handle = handle;
    managed->finalizer = finalizer;
    managed->closed = 0;
    object->native_handle = managed;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("ManagedNativeHandle"));
    jayess_object_set_value(object, "kind", jayess_value_from_string(kind != NULL ? kind : ""));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_runtime_accounting_state.native_handle_wrappers++;
    return jayess_value_from_object(object);
}

void *jayess_value_as_native_handle(jayess_value *value, const char *kind) {
    const char *actual_kind;
    jayess_managed_native_handle *managed;
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || value->as.object_value == NULL) {
        return NULL;
    }
    if (!(jayess_std_kind_is(value, "NativeHandle") || jayess_std_kind_is(value, "ManagedNativeHandle"))) {
        return NULL;
    }
    actual_kind = jayess_value_as_string(jayess_object_get(value->as.object_value, "kind"));
    if (kind != NULL && *kind != '\0' && !jayess_string_eq(actual_kind, kind)) {
        return NULL;
    }
    if (jayess_std_kind_is(value, "ManagedNativeHandle")) {
        managed = (jayess_managed_native_handle *)value->as.object_value->native_handle;
        if (managed == NULL || managed->closed) {
            return NULL;
        }
        return managed->handle;
    }
    return value->as.object_value->native_handle;
}

void jayess_value_clear_native_handle(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || value->as.object_value == NULL) {
        return;
    }
    if (jayess_std_kind_is(value, "ManagedNativeHandle")) {
        jayess_managed_native_handle *managed = (jayess_managed_native_handle *)value->as.object_value->native_handle;
        if (managed != NULL) {
            managed->handle = NULL;
            managed->closed = 1;
        }
        jayess_object_set_value(value->as.object_value, "closed", jayess_value_from_bool(1));
        return;
    }
    value->as.object_value->native_handle = NULL;
}

int jayess_value_close_native_handle(jayess_value *value) {
    jayess_managed_native_handle *managed;
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || value->as.object_value == NULL) {
        return 0;
    }
    if (!jayess_std_kind_is(value, "ManagedNativeHandle")) {
        value->as.object_value->native_handle = NULL;
        return 1;
    }
    managed = (jayess_managed_native_handle *)value->as.object_value->native_handle;
    if (managed == NULL || managed->closed) {
        jayess_object_set_value(value->as.object_value, "closed", jayess_value_from_bool(1));
        return 0;
    }
    if (managed->finalizer != NULL && managed->handle != NULL) {
        managed->finalizer(managed->handle);
    }
    managed->handle = NULL;
    managed->closed = 1;
    jayess_object_set_value(value->as.object_value, "closed", jayess_value_from_bool(1));
    return 1;
}

jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    jayess_function *function_value;
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_FUNCTION;
    function_value = (jayess_function *)malloc(sizeof(jayess_function));
    if (function_value == NULL) {
        free(boxed);
        return NULL;
    }
    function_value->callee = callee;
    function_value->env = env;
    function_value->name = name;
    function_value->class_name = class_name;
    function_value->param_count = param_count;
    function_value->has_rest = has_rest ? 1 : 0;
    function_value->properties = jayess_object_new();
    function_value->bound_this = jayess_value_undefined();
    function_value->bound_args = jayess_array_new();
    boxed->as.function_value = function_value;
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.functions++;
    return boxed;
}

jayess_value *jayess_call_function(jayess_value *callback, jayess_value *argument) {
    return jayess_value_call_one(callback, argument);
}

jayess_value *jayess_call_function2(jayess_value *callback, jayess_value *first, jayess_value *second) {
    return jayess_value_call_two_with_this(callback, jayess_value_undefined(), first, second);
}

jayess_value *jayess_call_function3(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third) {
    return jayess_value_call_three_with_this(callback, jayess_value_undefined(), first, second, third);
}

jayess_value *jayess_call_function4(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth) {
    return jayess_value_call_four_with_this(callback, jayess_value_undefined(), first, second, third, fourth);
}

jayess_value *jayess_call_function5(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth) {
    return jayess_value_call_five_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth);
}

jayess_value *jayess_call_function6(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth) {
    return jayess_value_call_six_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth);
}

jayess_value *jayess_call_function7(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh) {
    return jayess_value_call_seven_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh);
}

jayess_value *jayess_call_function8(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth) {
    return jayess_value_call_eight_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth);
}

jayess_value *jayess_call_function9(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth) {
    return jayess_value_call_nine_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth, ninth);
}

jayess_value *jayess_call_function10(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth) {
    return jayess_value_call_ten_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth, ninth, tenth);
}

jayess_value *jayess_call_function11(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh) {
    return jayess_value_call_eleven_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth, ninth, tenth, eleventh);
}

jayess_value *jayess_call_function12(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh, jayess_value *twelfth) {
    return jayess_value_call_twelve_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth, ninth, tenth, eleventh, twelfth);
}

jayess_value *jayess_call_function13(jayess_value *callback, jayess_value *first, jayess_value *second, jayess_value *third, jayess_value *fourth, jayess_value *fifth, jayess_value *sixth, jayess_value *seventh, jayess_value *eighth, jayess_value *ninth, jayess_value *tenth, jayess_value *eleventh, jayess_value *twelfth, jayess_value *thirteenth) {
    return jayess_value_call_thirteen_with_this(callback, jayess_value_undefined(), first, second, third, fourth, fifth, sixth, seventh, eighth, ninth, tenth, eleventh, twelfth, thirteenth);
}

void *jayess_value_function_ptr(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return NULL;
    }
    return value->as.function_value->callee;
}

jayess_value *jayess_value_function_env(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return NULL;
    }
    return value->as.function_value->env;
}

jayess_value *jayess_value_bind(jayess_value *value, jayess_value *bound_this, jayess_value *bound_args) {
    jayess_value *boxed;
    jayess_function *original;
    jayess_function *bound;
    jayess_array *tail = NULL;

    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return jayess_value_undefined();
    }

    original = value->as.function_value;
    if (bound_args != NULL && bound_args->kind == JAYESS_VALUE_ARRAY) {
        tail = bound_args->as.array_value;
    }

    boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_FUNCTION;
    bound = (jayess_function *)malloc(sizeof(jayess_function));
    if (bound == NULL) {
        free(boxed);
        return NULL;
    }

    bound->callee = original->callee;
    bound->env = original->env;
    bound->name = original->name;
    bound->class_name = original->class_name;
    bound->param_count = original->param_count;
    bound->has_rest = original->has_rest;
    bound->properties = jayess_object_new();
    bound->bound_this = bound_this != NULL ? bound_this : original->bound_this;
    bound->bound_args = jayess_array_concat_bound_args_owned(original->bound_args, tail);

    boxed->as.function_value = bound;
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.functions++;
    return boxed;
}

jayess_value *jayess_value_function_bound_this(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    if (value->as.function_value->bound_this == NULL) {
        return jayess_value_undefined();
    }
    return value->as.function_value->bound_this;
}

const char *jayess_value_function_class_name(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL || value->as.function_value->class_name == NULL) {
        return "";
    }
    return value->as.function_value->class_name;
}

int jayess_value_function_param_count(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return 0;
    }
    return value->as.function_value->param_count;
}

int jayess_value_function_has_rest(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return 0;
    }
    return value->as.function_value->has_rest;
}

int jayess_value_function_bound_arg_count(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return 0;
    }
    if (value->as.function_value->bound_args == NULL) {
        return 0;
    }
    return value->as.function_value->bound_args->count;
}

jayess_value *jayess_value_function_bound_arg(jayess_value *value, int index) {
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return jayess_value_undefined();
    }
    if (value->as.function_value->bound_args == NULL) {
        return jayess_value_undefined();
    }
    return jayess_array_get(value->as.function_value->bound_args, index);
}

jayess_value *jayess_value_merge_bound_args(jayess_value *value, jayess_value *tail_args) {
    jayess_array *tail = NULL;
    jayess_array *merged;

    if (tail_args != NULL && tail_args->kind == JAYESS_VALUE_ARRAY) {
        tail = tail_args->as.array_value;
    }
    if (value == NULL || value->kind != JAYESS_VALUE_FUNCTION || value->as.function_value == NULL) {
        return jayess_value_from_array(jayess_array_clone(tail));
    }

    merged = jayess_array_concat_bound_args_owned(value->as.function_value->bound_args, tail);
    return jayess_value_from_array(merged);
}

int jayess_value_instanceof(jayess_value *target, const char *class_name) {
    char key[512];
    jayess_value *marker;

    if (target == NULL || class_name == NULL || class_name[0] == '\0') {
        return 0;
    }
    snprintf(key, sizeof(key), "__jayess_is_%s", class_name);
    marker = jayess_value_get_member(target, key);
    if (marker == NULL) {
        return 0;
    }
    return jayess_value_as_bool(marker) != 0;
}

unsigned char *jayess_expect_bytes_copy(jayess_value *value, size_t *length_out, const char *context) {
    unsigned char *bytes = jayess_value_to_bytes_copy(value, length_out);
    if (bytes == NULL && (length_out == NULL || *length_out == 0)) {
        char message[256];
        snprintf(message, sizeof(message), "%s expects a Uint8Array or byte buffer value", context != NULL && context[0] != '\0' ? context : "native wrapper");
        jayess_throw_type_error(message);
    }
    return bytes;
}

void *jayess_expect_native_handle(jayess_value *value, const char *kind, const char *context) {
    void *handle = jayess_value_as_native_handle(value, kind);
    if (handle == NULL) {
        char message[256];
        if (kind != NULL && kind[0] != '\0') {
            snprintf(message, sizeof(message), "%s expects a %s native handle", context != NULL && context[0] != '\0' ? context : "native wrapper", kind);
        } else {
            snprintf(message, sizeof(message), "%s expects a native handle", context != NULL && context[0] != '\0' ? context : "native wrapper");
        }
        jayess_throw_type_error(message);
    }
    return handle;
}

static jayess_array *jayess_array_clone(jayess_array *array) {
    int i;
    jayess_array *copy = jayess_array_new();
    if (copy == NULL) {
        return NULL;
    }
    if (array == NULL) {
        return copy;
    }
    for (i = 0; i < array->count; i++) {
        jayess_array_set_value(copy, i, array->values[i]);
    }
    return copy;
}

static jayess_array *jayess_array_concat(jayess_array *left, jayess_array *right) {
    int i;
    jayess_array *merged = jayess_array_new();
    if (merged == NULL) {
        return NULL;
    }
    if (left != NULL) {
        for (i = 0; i < left->count; i++) {
            jayess_array_set_value(merged, merged->count, left->values[i]);
        }
    }
    if (right != NULL) {
        for (i = 0; i < right->count; i++) {
            jayess_array_set_value(merged, merged->count, right->values[i]);
        }
    }
    return merged;
}

static jayess_value *jayess_value_clone_bound_arg(jayess_value *value) {
    if (value == NULL) {
        return NULL;
    }
    switch (value->kind) {
        case JAYESS_VALUE_STRING:
            return jayess_value_from_string(value->as.string_value);
        case JAYESS_VALUE_NUMBER:
            return jayess_value_from_number(value->as.number_value);
        case JAYESS_VALUE_BIGINT:
            return jayess_value_from_bigint(value->as.bigint_value);
        case JAYESS_VALUE_BOOL:
            return jayess_value_from_bool(value->as.bool_value);
        case JAYESS_VALUE_NULL:
            return jayess_value_null();
        case JAYESS_VALUE_UNDEFINED:
            return jayess_value_undefined();
        case JAYESS_VALUE_SYMBOL:
            if (value->as.symbol_value == NULL) {
                return NULL;
            }
            return jayess_value_from_symbol(value->as.symbol_value->description);
        case JAYESS_VALUE_OBJECT:
        case JAYESS_VALUE_ARRAY:
        case JAYESS_VALUE_FUNCTION:
        default:
            return value;
    }
}

static jayess_array *jayess_array_concat_bound_args_owned(jayess_array *left, jayess_array *right) {
    int i;
    jayess_array *merged = jayess_array_new();
    if (merged == NULL) {
        return NULL;
    }
    if (left != NULL) {
        for (i = 0; i < left->count; i++) {
            jayess_array_set_value(merged, merged->count, jayess_value_clone_bound_arg(left->values[i]));
        }
    }
    if (right != NULL) {
        for (i = 0; i < right->count; i++) {
            jayess_array_set_value(merged, merged->count, jayess_value_clone_bound_arg(right->values[i]));
        }
    }
    return merged;
}
