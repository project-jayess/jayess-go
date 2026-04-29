#ifndef JAYESS_RUNTIME_INTERNAL_H
#define JAYESS_RUNTIME_INTERNAL_H

/* Shared concrete runtime types for split runtime implementation files. */

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <pthread.h>
#endif

#include "jayess_runtime.h"

#define JAYESS_STD_STREAM_DEFAULT_HIGH_WATER_MARK 4096

#ifdef _WIN32
#define JAYESS_THREAD_LOCAL __declspec(thread)
#else
#define JAYESS_THREAD_LOCAL __thread
#endif

typedef struct jayess_function jayess_function;
typedef struct jayess_symbol jayess_symbol;
typedef struct jayess_object_entry jayess_object_entry;
typedef struct jayess_promise_dependent jayess_promise_dependent;
typedef struct jayess_crypto_key_state jayess_crypto_key_state;
typedef struct jayess_runtime_accounting jayess_runtime_accounting;
typedef struct jayess_buffer_state jayess_buffer_state;

#ifdef _WIN32
typedef uintptr_t jayess_socket_handle;
#define JAYESS_INVALID_SOCKET ((jayess_socket_handle)(~(uintptr_t)0))
#else
typedef int jayess_socket_handle;
#define JAYESS_INVALID_SOCKET (-1)
#endif

typedef struct jayess_args {
    int count;
    char **values;
} jayess_args;

struct jayess_object_entry {
    char *key;
    jayess_value *key_value;
    jayess_value *value;
    jayess_object_entry *next;
};

struct jayess_object {
    jayess_object_entry *head;
    jayess_object_entry *tail;
    jayess_promise_dependent *promise_dependents;
    FILE *stream_file;
    jayess_socket_handle socket_handle;
    void *native_handle;
};

struct jayess_array {
    int count;
    jayess_value **values;
};

struct jayess_function {
    void *callee;
    jayess_value *env;
    const char *name;
    const char *class_name;
    int param_count;
    int has_rest;
    jayess_object *properties;
    jayess_value *bound_this;
    jayess_array *bound_args;
};

struct jayess_symbol {
    uint64_t id;
    char *description;
};

typedef struct jayess_this_frame {
    jayess_value *value;
    struct jayess_this_frame *previous;
} jayess_this_frame;

typedef struct jayess_call_frame {
    const char *name;
    struct jayess_call_frame *previous;
} jayess_call_frame;

struct jayess_value {
    jayess_value_kind kind;
    union {
        char *string_value;
        double number_value;
        char *bigint_value;
        int bool_value;
        jayess_object *object_value;
        jayess_array *array_value;
        jayess_function *function_value;
        jayess_symbol *symbol_value;
    } as;
};

struct jayess_runtime_accounting {
    size_t boxed_values;
    size_t objects;
    size_t object_entries;
    size_t arrays;
    size_t array_slots;
    size_t functions;
    size_t strings;
    size_t bigints;
    size_t symbols;
    size_t native_handle_wrappers;
};

struct jayess_buffer_state {
    jayess_array *bytes;
    size_t refcount;
    int shared;
#ifdef _WIN32
    CRITICAL_SECTION lock;
#else
    pthread_mutex_t lock;
#endif
};

extern jayess_runtime_accounting jayess_runtime_accounting_state;

char *jayess_strdup(const char *value);
void jayess_runtime_free_static_strings(void);
int jayess_object_entry_is_symbol(jayess_object_entry *entry);
int jayess_object_entry_matches_string(jayess_object_entry *entry, const char *key);
int jayess_object_entry_matches_value(jayess_object_entry *entry, jayess_value *key);
jayess_object_entry *jayess_object_find_value(jayess_object *object, jayess_value *key);
void jayess_object_set_key_value(jayess_object *object, jayess_value *key, jayess_value *value);
jayess_value *jayess_object_get_key_value(jayess_object *object, jayess_value *key);
void jayess_object_delete_key_value(jayess_object *object, jayess_value *key);
void jayess_print_property_key_inline(jayess_object_entry *entry);
void jayess_print_value_inline(jayess_value *value);
extern jayess_value *jayess_process_signal_bus;
char *jayess_read_text_file_or_empty(const char *path);
int jayess_std_kind_is(jayess_value *target, const char *kind);
jayess_array *jayess_std_bytes_slot(jayess_value *target);
jayess_buffer_state *jayess_std_bytes_state(jayess_value *target);
jayess_value *jayess_std_buffer_value_from_state(jayess_buffer_state *state);
void jayess_std_buffer_state_retain(jayess_buffer_state *state);
void jayess_std_buffer_state_release(jayess_buffer_state *state);
jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message);
jayess_value *jayess_std_array_buffer_new(jayess_value *length_value);
jayess_value *jayess_std_uint8_array_new(jayess_value *source);
jayess_value *jayess_std_uint8_to_string_method(jayess_value *env, jayess_value *encoding);
const char *jayess_std_typed_array_kind(jayess_value *target);
int jayess_std_is_typed_array(jayess_value *target);
jayess_value *jayess_std_typed_array_new(const char *kind, jayess_value *source);
char *jayess_compile_option_string(jayess_value *options, const char *key);
int jayess_std_child_process_signal_number(const char *signal_name);
const char *jayess_std_process_signal_name(int signal_number);
jayess_value *jayess_std_process_signal_bus_value(void);
int jayess_std_process_install_signal(int signal_number);
void jayess_runtime_note_signal(int signal_number);
void jayess_runtime_dispatch_pending_signals(void);
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
jayess_value *jayess_std_write_stream_write_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_write_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_write_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_write_stream_end_method(jayess_value *env);
jayess_value *jayess_std_writable_write(jayess_value *destination, jayess_value *chunk);
jayess_value *jayess_std_writable_end(jayess_value *destination);
jayess_value *jayess_std_compression_stream_write_method(jayess_value *env, jayess_value *value);
jayess_value *jayess_std_compression_stream_end_method(jayess_value *env);
jayess_value *jayess_std_compression_stream_read_method(jayess_value *env, jayess_value *size_value);
jayess_value *jayess_std_compression_stream_read_bytes_method(jayess_value *env, jayess_value *size_value);
jayess_value *jayess_std_compression_stream_pipe_method(jayess_value *env, jayess_value *destination);
jayess_value *jayess_std_compression_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
jayess_value *jayess_std_compression_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
const char *jayess_temp_dir(void);
jayess_value *jayess_value_call_with_this(jayess_value *callback, jayess_value *this_value, jayess_value *argument, int argument_count);
jayess_value *jayess_value_call_one(jayess_value *callback, jayess_value *argument);
int jayess_std_crypto_copy_bytes(jayess_value *value, unsigned char **out_bytes, size_t *out_length);
char *jayess_std_crypto_hex_encode(const unsigned char *bytes, size_t length);
int jayess_std_crypto_equal_name(const char *left, const char *right);
void jayess_std_crypto_normalize_name(char *text);
int jayess_std_crypto_cipher_key_length(const char *algorithm);
int jayess_std_crypto_option_bytes(jayess_value *options, const char *key, unsigned char **out_bytes, size_t *out_length, int required);
jayess_value *jayess_std_crypto_key_value(const char *type, int is_private);
jayess_crypto_key_state *jayess_std_crypto_key_state_from_value(jayess_value *value);
int jayess_std_bytes_encoding_is_hex(jayess_value *encoding);
int jayess_std_bytes_encoding_is_base64(jayess_value *encoding);
int jayess_std_bytes_encoding_is_text(jayess_value *encoding);
jayess_value *jayess_std_worker_post_message_method(jayess_value *env, jayess_value *message);
jayess_value *jayess_std_worker_receive_method(jayess_value *env, jayess_value *timeout);
jayess_value *jayess_std_worker_terminate_method(jayess_value *env);
#ifdef _WIN32
LPCWSTR jayess_std_crypto_algorithm_id(const char *name);
int jayess_std_crypto_sha256_bytes(const unsigned char *input, size_t input_length, unsigned char *output, DWORD *output_length);
#endif

#endif
