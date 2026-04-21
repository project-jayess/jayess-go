#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <stdint.h>
#include <math.h>
#include <time.h>

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
#include <security.h>
#include <schannel.h>
#else
#include <arpa/inet.h>
#include <dirent.h>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/tcp.h>
#include <pthread.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>
#include <errno.h>
#include <openssl/ssl.h>
#include <openssl/err.h>
#include <openssl/x509v3.h>
#endif

typedef struct jayess_args {
    int count;
    char **values;
} jayess_args;

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

typedef struct jayess_value jayess_value;
typedef struct jayess_object_entry jayess_object_entry;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;
typedef struct jayess_function jayess_function;
typedef struct jayess_microtask jayess_microtask;
typedef struct jayess_promise_dependent jayess_promise_dependent;

#ifdef _WIN32
typedef SOCKET jayess_socket_handle;
#define JAYESS_INVALID_SOCKET INVALID_SOCKET
typedef struct jayess_tls_socket_state {
    jayess_socket_handle handle;
    CredHandle credentials;
    CtxtHandle context;
    int has_credentials;
    int has_context;
    int reject_unauthorized;
    char *host;
    SecPkgContext_StreamSizes stream_sizes;
    unsigned char *encrypted_buffer;
    size_t encrypted_length;
    size_t encrypted_capacity;
    unsigned char *plaintext_buffer;
    size_t plaintext_offset;
    size_t plaintext_length;
} jayess_tls_socket_state;
typedef struct jayess_winhttp_stream_state {
    HINTERNET request;
    HINTERNET connection;
    HINTERNET session;
} jayess_winhttp_stream_state;
#else
typedef int jayess_socket_handle;
#define JAYESS_INVALID_SOCKET (-1)
typedef struct jayess_tls_socket_state {
    jayess_socket_handle handle;
    SSL_CTX *ctx;
    SSL *ssl;
    int reject_unauthorized;
    char *host;
} jayess_tls_socket_state;
#endif

struct jayess_object_entry {
    char *key;
    jayess_value *value;
    jayess_object_entry *next;
};

struct jayess_object {
    jayess_object_entry *head;
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

typedef struct jayess_this_frame {
    jayess_value *value;
    struct jayess_this_frame *previous;
} jayess_this_frame;

typedef jayess_value *(*jayess_callback0)(void);
typedef jayess_value *(*jayess_callback1)(jayess_value *);
typedef jayess_value *(*jayess_callback2)(jayess_value *, jayess_value *);

typedef enum jayess_task_kind {
    JAYESS_TASK_PROMISE_CALLBACK = 0,
    JAYESS_TASK_FS_READ = 1,
    JAYESS_TASK_TIMER = 2,
    JAYESS_TASK_FS_WRITE = 3,
    JAYESS_TASK_SOCKET_READ = 4,
    JAYESS_TASK_SOCKET_WRITE = 5,
    JAYESS_TASK_SERVER_ACCEPT = 6,
    JAYESS_TASK_HTTP_REQUEST = 7,
    JAYESS_TASK_HTTP_GET = 8,
    JAYESS_TASK_HTTPS_REQUEST = 9,
    JAYESS_TASK_HTTPS_GET = 10,
    JAYESS_TASK_HTTP_REQUEST_STREAM = 11,
    JAYESS_TASK_HTTP_GET_STREAM = 12,
    JAYESS_TASK_HTTPS_REQUEST_STREAM = 13,
    JAYESS_TASK_HTTPS_GET_STREAM = 14
} jayess_task_kind;

typedef enum jayess_promise_action {
    JAYESS_PROMISE_ACTION_THEN = 0,
    JAYESS_PROMISE_ACTION_ALL = 1,
    JAYESS_PROMISE_ACTION_RACE = 2,
    JAYESS_PROMISE_ACTION_ALL_SETTLED = 3,
    JAYESS_PROMISE_ACTION_ANY = 4,
    JAYESS_PROMISE_ACTION_FINALLY = 5
} jayess_promise_action;

struct jayess_microtask {
    jayess_task_kind kind;
    jayess_promise_action promise_action;
    volatile int completed;
    jayess_value *source;
    jayess_value *result;
    jayess_value *on_fulfilled;
    jayess_value *on_rejected;
    jayess_value *path;
    jayess_value *encoding;
    jayess_value *content;
    jayess_value *worker_result;
    jayess_socket_handle socket_handle;
    int worker_bytes;
    int worker_emit_error;
    int worker_emit_close;
    double due_ms;
    int timer_id;
    struct jayess_microtask *next;
    struct jayess_microtask *worker_next;
    int dependency_count;
    int queued;
    int finished;
};

struct jayess_promise_dependent {
    jayess_microtask *task;
    jayess_promise_dependent *next;
};

typedef struct jayess_task_queue {
    jayess_microtask *head;
    jayess_microtask *tail;
} jayess_task_queue;

typedef struct jayess_scheduler {
    jayess_task_queue promise_callbacks;
    jayess_task_queue timers;
    jayess_task_queue io_pending;
    jayess_task_queue io_completions;
} jayess_scheduler;

#define JAYESS_IO_WORKER_COUNT 4

typedef struct jayess_io_worker_pool {
    int started;
    int stopping;
    int worker_count;
    jayess_microtask *head;
    jayess_microtask *tail;
#ifdef _WIN32
    CRITICAL_SECTION lock;
    CONDITION_VARIABLE available;
    HANDLE workers[JAYESS_IO_WORKER_COUNT];
#else
    pthread_mutex_t lock;
    pthread_cond_t available;
    pthread_t workers[JAYESS_IO_WORKER_COUNT];
#endif
} jayess_io_worker_pool;

struct jayess_value {
    jayess_value_kind kind;
    union {
        char *string_value;
        double number_value;
        int bool_value;
        jayess_object *object_value;
        jayess_array *array_value;
        jayess_function *function_value;
    } as;
};

char *jayess_value_stringify(jayess_value *value);
double jayess_value_to_number(jayess_value *value);
int jayess_value_eq(jayess_value *left, jayess_value *right);
int jayess_value_is_nullish(jayess_value *value);
const char *jayess_value_as_string(jayess_value *value);
int jayess_value_as_bool(jayess_value *value);
int jayess_string_eq(const char *left, const char *right);
jayess_value *jayess_value_undefined(void);
jayess_value *jayess_value_from_string(const char *value);
jayess_value *jayess_value_from_number(double value);
jayess_value *jayess_value_from_bool(int value);
jayess_value *jayess_value_from_object(jayess_object *value);
jayess_value *jayess_value_from_array(jayess_array *value);
jayess_value *jayess_value_from_args(jayess_args *args);
jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest);
jayess_value *jayess_value_get_member(jayess_value *target, const char *key);
jayess_object *jayess_value_as_object(jayess_value *value);
jayess_value *jayess_value_array_includes(jayess_value *target, jayess_value *value);
jayess_value *jayess_value_array_join(jayess_value *target, jayess_value *separator);
jayess_value *jayess_std_path_dirname(jayess_value *path);
jayess_value *jayess_std_path_extname(jayess_value *path);
jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message);
jayess_value *jayess_std_fs_read_file(jayess_value *path, jayess_value *encoding);
jayess_value *jayess_std_fs_write_file(jayess_value *path, jayess_value *content);
jayess_object *jayess_object_new(void);
void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value);
jayess_value *jayess_object_get(jayess_object *object, const char *key);
jayess_array *jayess_array_new(void);
void jayess_array_set_value(jayess_array *array, int index, jayess_value *value);
int jayess_array_push_value(jayess_array *array, jayess_value *value);
jayess_value *jayess_array_pop_value(jayess_array *array);
jayess_value *jayess_array_get(jayess_array *array, int index);
jayess_value *jayess_value_iterable_values(jayess_value *target);
void jayess_push_this(jayess_value *value);
void jayess_pop_this(void);
void jayess_throw(jayess_value *value);
int jayess_has_exception(void);
jayess_value *jayess_take_exception(void);
jayess_value *jayess_set_timeout(jayess_value *callback, jayess_value *delay);
jayess_value *jayess_clear_timeout(jayess_value *id);
void jayess_run_microtasks(void);
void jayess_throw_not_function(void);

static jayess_value jayess_null_singleton = {JAYESS_VALUE_NULL, {0}};
static jayess_value jayess_undefined_singleton = {JAYESS_VALUE_UNDEFINED, {0}};
static jayess_this_frame *jayess_this_stack = NULL;
static jayess_value *jayess_current_exception = NULL;
static jayess_args *jayess_current_args = NULL;
static jayess_scheduler jayess_runtime_scheduler = {{NULL, NULL}, {NULL, NULL}, {NULL, NULL}, {NULL, NULL}};
static jayess_io_worker_pool jayess_runtime_io_pool = {0};
static int jayess_next_timer_id = 1;

static jayess_value *jayess_std_promise_pending(void);
static void jayess_enqueue_microtask(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected);
static void jayess_enqueue_promise_task(jayess_value *source, jayess_value *result, jayess_value *on_fulfilled, jayess_value *on_rejected, jayess_promise_action action);
static void jayess_requeue_microtask(jayess_microtask *task);
static void jayess_append_microtask(jayess_microtask *task);
static jayess_array *jayess_std_bytes_slot(jayess_value *target);
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
static jayess_value *jayess_std_server_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_server_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_server_accept_method(jayess_value *env);
static jayess_value *jayess_std_server_close_method(jayess_value *env);
static jayess_value *jayess_std_server_address_method(jayess_value *env);
static jayess_value *jayess_std_server_set_timeout_method(jayess_value *env, jayess_value *timeout_ms);
static jayess_value *jayess_std_server_accept_async_method(jayess_value *env);
static void jayess_std_stream_emit(jayess_value *env, const char *event, jayess_value *argument);
static int jayess_std_socket_configure_timeout(jayess_socket_handle handle, int timeout);
jayess_value *jayess_std_http_parse_request(jayess_value *input);
jayess_value *jayess_std_http_format_request(jayess_value *parts);
jayess_value *jayess_std_http_parse_response(jayess_value *input);
jayess_value *jayess_std_http_format_response(jayess_value *parts);
jayess_value *jayess_std_url_parse(jayess_value *input);
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
static int jayess_std_socket_runtime_ready(void);
static jayess_value *jayess_std_read_stream_read_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_read_stream_read_bytes_method(jayess_value *env, jayess_value *size_value);
static jayess_value *jayess_std_read_stream_close_method(jayess_value *env);
static jayess_value *jayess_std_read_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_read_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_stream_off_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_stream_listener_count_method(jayess_value *env, jayess_value *event);
static jayess_value *jayess_std_stream_event_names_method(jayess_value *env);
static jayess_value *jayess_std_read_stream_pipe_method(jayess_value *env, jayess_value *destination);
static jayess_value *jayess_std_write_stream_write_method(jayess_value *env, jayess_value *value);
static jayess_value *jayess_std_write_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_write_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback);
static jayess_value *jayess_std_write_stream_end_method(jayess_value *env);
static int jayess_std_stream_requested_size(jayess_value *size_value, int default_size);
static void jayess_std_stream_emit_error(jayess_value *env, const char *message);
static int jayess_std_socket_close_handle(jayess_socket_handle handle);
static jayess_value *jayess_std_socket_value_from_handle(jayess_socket_handle handle, const char *remote_address, int remote_port);
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

#ifdef _WIN32
static jayess_tls_socket_state *jayess_std_tls_state(jayess_value *env);
static int jayess_std_tls_send_all(jayess_socket_handle handle, const unsigned char *buffer, size_t length);
static int jayess_std_tls_state_free(jayess_tls_socket_state *state, int close_handle);
static int jayess_std_tls_read_bytes(jayess_value *env, unsigned char *buffer, int max_count, int *did_timeout);
static int jayess_std_tls_write_bytes(jayess_value *env, const unsigned char *buffer, int length, int *did_timeout);
static jayess_value *jayess_std_tls_connect_socket(jayess_value *options);
static jayess_value *jayess_std_tls_normalize_alpn_protocols(jayess_value *value);
static int jayess_std_tls_build_alpn_wire(jayess_value *protocols_value, unsigned char **out_buffer, size_t *out_length);
static void jayess_std_https_copy_tls_request_settings(jayess_object *target, jayess_object *source);
static jayess_value *jayess_std_tls_subject_alt_names(jayess_value *env);
#ifdef _WIN32
static int jayess_std_windows_load_certificates_from_file(HCERTSTORE store, const char *path);
static int jayess_std_windows_load_certificates_from_path(HCERTSTORE store, const char *path);
static int jayess_std_windows_validate_tls_certificate(jayess_tls_socket_state *state, const char *server_name, const char *ca_file, const char *ca_path, int trust_system);
static void *jayess_std_tls_build_schannel_alpn_buffer(const unsigned char *wire, size_t wire_length, unsigned long *buffer_length);
static const char *jayess_std_tls_windows_protocol_name(DWORD protocol);
#endif
static jayess_value *jayess_std_tls_peer_certificate(jayess_value *env);
#endif

static char *jayess_strdup(const char *value) {
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

static char *jayess_number_to_string(double value) {
    char buffer[64];
    snprintf(buffer, sizeof(buffer), "%g", value);
    return jayess_strdup(buffer);
}

static int jayess_path_is_separator(char ch) {
#ifdef _WIN32
    return ch == '\\' || ch == '/';
#else
    return ch == '/';
#endif
}

static const char *jayess_path_last_separator(const char *text) {
    const char *last = NULL;
    while (text != NULL && *text != '\0') {
        if (jayess_path_is_separator(*text)) {
            last = text;
        }
        text++;
    }
    return last;
}

static int jayess_path_is_absolute(const char *text) {
    if (text == NULL || text[0] == '\0') {
        return 0;
    }
#ifdef _WIN32
    if ((text[0] == '\\' || text[0] == '/') || (isalpha((unsigned char)text[0]) && text[1] == ':')) {
        return 1;
    }
    return 0;
#else
    return text[0] == '/';
#endif
}

static char jayess_path_separator_char(void) {
#ifdef _WIN32
    return '\\';
#else
    return '/';
#endif
}

static const char *jayess_path_separator_string(void) {
#ifdef _WIN32
    return "\\";
#else
    return "/";
#endif
}

static const char *jayess_path_delimiter_string(void) {
#ifdef _WIN32
    return ";";
#else
    return ":";
#endif
}

static int jayess_path_root_length(const char *text) {
    if (text == NULL || text[0] == '\0') {
        return 0;
    }
#ifdef _WIN32
    if (isalpha((unsigned char)text[0]) && text[1] == ':') {
        if (jayess_path_is_separator(text[2])) {
            return 3;
        }
        return 2;
    }
    if (jayess_path_is_separator(text[0])) {
        return 1;
    }
    return 0;
#else
    return text[0] == '/' ? 1 : 0;
#endif
}

static jayess_array *jayess_path_split_segments(const char *text) {
    int root_length = jayess_path_root_length(text);
    const char *cursor = text != NULL ? text + root_length : "";
    jayess_array *segments = jayess_array_new();
    while (*cursor != '\0') {
        const char *start = cursor;
        size_t length;
        char *segment;
        while (*cursor != '\0' && !jayess_path_is_separator(*cursor)) {
            cursor++;
        }
        length = (size_t)(cursor - start);
        if (length > 0) {
            segment = (char *)malloc(length + 1);
            if (segment == NULL) {
                return segments;
            }
            memcpy(segment, start, length);
            segment[length] = '\0';
            if (strcmp(segment, ".") == 0) {
                free(segment);
            } else if (strcmp(segment, "..") == 0) {
                if (segments->count > 0) {
                    jayess_array_pop_value(segments);
                }
                free(segment);
            } else {
                jayess_array_push_value(segments, jayess_value_from_string(segment));
                free(segment);
            }
        }
        while (*cursor != '\0' && jayess_path_is_separator(*cursor)) {
            cursor++;
        }
    }
    return segments;
}

static char *jayess_path_join_segments_with_root(const char *root, jayess_array *segments) {
    char sep = jayess_path_separator_char();
    size_t total = 1;
    int i;
    int root_len = root != NULL ? (int)strlen(root) : 0;
    char *out;
    if (root_len > 0) {
        total += (size_t)root_len;
    }
    if (segments != NULL) {
        for (i = 0; i < segments->count; i++) {
            const char *piece = jayess_value_as_string(jayess_array_get(segments, i));
            total += strlen(piece);
            if ((root_len > 0 || i > 0) && piece[0] != '\0') {
                total += 1;
            }
        }
    }
    out = (char *)malloc(total);
    if (out == NULL) {
        return NULL;
    }
    out[0] = '\0';
    if (root_len > 0) {
        strcpy(out, root);
    }
    if (segments != NULL) {
        for (i = 0; i < segments->count; i++) {
            const char *piece = jayess_value_as_string(jayess_array_get(segments, i));
            size_t current_len = strlen(out);
            if (piece[0] == '\0') {
                continue;
            }
            if (current_len > 0 && !jayess_path_is_separator(out[current_len - 1])) {
                out[current_len] = sep;
                out[current_len + 1] = '\0';
            }
            strcat(out, piece);
        }
    }
    if (out[0] == '\0') {
        strcpy(out, ".");
    }
    return out;
}

static int jayess_path_exists_text(const char *path_text) {
#ifdef _WIN32
    DWORD attributes = GetFileAttributesA(path_text);
    return attributes != INVALID_FILE_ATTRIBUTES;
#else
    struct stat info;
    return path_text != NULL && stat(path_text, &info) == 0;
#endif
}

static int jayess_path_is_dir_text(const char *path_text) {
#ifdef _WIN32
    DWORD attributes = GetFileAttributesA(path_text);
    if (attributes == INVALID_FILE_ATTRIBUTES) {
        return 0;
    }
    return (attributes & FILE_ATTRIBUTE_DIRECTORY) != 0;
#else
    struct stat info;
    if (path_text == NULL || stat(path_text, &info) != 0) {
        return 0;
    }
    return S_ISDIR(info.st_mode);
#endif
}

static int jayess_path_mkdir_single(const char *path_text) {
    if (path_text == NULL || path_text[0] == '\0') {
        return 0;
    }
    if (jayess_path_exists_text(path_text)) {
        return jayess_path_is_dir_text(path_text);
    }
#ifdef _WIN32
    return _mkdir(path_text) == 0;
#else
    return mkdir(path_text, 0755) == 0;
#endif
}

static int jayess_fs_remove_path_recursive(const char *path_text) {
    if (path_text == NULL || path_text[0] == '\0') {
        return 0;
    }
    if (!jayess_path_is_dir_text(path_text)) {
#ifdef _WIN32
        return DeleteFileA(path_text) != 0;
#else
        return remove(path_text) == 0;
#endif
    }
#ifdef _WIN32
    {
        WIN32_FIND_DATAA find_data;
        HANDLE handle;
        size_t length = strlen(path_text);
        char *pattern = (char *)malloc(length + 3);
        int ok = 1;
        if (pattern == NULL) {
            return 0;
        }
        strcpy(pattern, path_text);
        if (length > 0 && !jayess_path_is_separator(pattern[length - 1])) {
            strcat(pattern, "\\");
        }
        strcat(pattern, "*");
        handle = FindFirstFileA(pattern, &find_data);
        free(pattern);
        if (handle != INVALID_HANDLE_VALUE) {
            do {
                char *full_path;
                if (strcmp(find_data.cFileName, ".") == 0 || strcmp(find_data.cFileName, "..") == 0) {
                    continue;
                }
                full_path = (char *)malloc(length + strlen(find_data.cFileName) + 3);
                if (full_path == NULL) {
                    ok = 0;
                    continue;
                }
                strcpy(full_path, path_text);
                if (length > 0 && !jayess_path_is_separator(full_path[length - 1])) {
                    strcat(full_path, "\\");
                }
                strcat(full_path, find_data.cFileName);
                if (!jayess_fs_remove_path_recursive(full_path)) {
                    ok = 0;
                }
                free(full_path);
            } while (FindNextFileA(handle, &find_data));
            FindClose(handle);
        }
        return ok && RemoveDirectoryA(path_text) != 0;
    }
#else
    {
        DIR *dir = opendir(path_text);
        struct dirent *entry;
        int ok = 1;
        size_t length = strlen(path_text);
        if (dir == NULL) {
            return 0;
        }
        while ((entry = readdir(dir)) != NULL) {
            char *full_path;
            if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
                continue;
            }
            full_path = (char *)malloc(length + strlen(entry->d_name) + 3);
            if (full_path == NULL) {
                ok = 0;
                continue;
            }
            strcpy(full_path, path_text);
            if (length > 0 && !jayess_path_is_separator(full_path[length - 1])) {
                strcat(full_path, "/");
            }
            strcat(full_path, entry->d_name);
            if (!jayess_fs_remove_path_recursive(full_path)) {
                ok = 0;
            }
            free(full_path);
        }
        closedir(dir);
        return ok && rmdir(path_text) == 0;
    }
#endif
}

static int jayess_fs_copy_dir_recursive(const char *from_text, const char *to_text);

static int jayess_object_option_bool(jayess_value *options, const char *key) {
    jayess_value *value;
    if (options == NULL || options->kind != JAYESS_VALUE_OBJECT || options->as.object_value == NULL) {
        return 0;
    }
    value = jayess_object_get(options->as.object_value, key);
    return jayess_value_as_bool(value);
}

static double jayess_path_file_size_text(const char *path_text) {
#ifdef _WIN32
    WIN32_FILE_ATTRIBUTE_DATA data;
    LARGE_INTEGER size;
    if (path_text == NULL || !GetFileAttributesExA(path_text, GetFileExInfoStandard, &data)) {
        return 0.0;
    }
    size.HighPart = (LONG)data.nFileSizeHigh;
    size.LowPart = data.nFileSizeLow;
    return (double)size.QuadPart;
#else
    struct stat info;
    if (path_text == NULL || stat(path_text, &info) != 0) {
        return 0.0;
    }
    return (double)info.st_size;
#endif
}

static double jayess_path_modified_time_ms_text(const char *path_text) {
#ifdef _WIN32
    WIN32_FILE_ATTRIBUTE_DATA data;
    ULARGE_INTEGER value;
    if (path_text == NULL || !GetFileAttributesExA(path_text, GetFileExInfoStandard, &data)) {
        return 0.0;
    }
    value.HighPart = data.ftLastWriteTime.dwHighDateTime;
    value.LowPart = data.ftLastWriteTime.dwLowDateTime;
    return (double)((value.QuadPart - 116444736000000000ULL) / 10000ULL);
#else
    struct stat info;
    if (path_text == NULL || stat(path_text, &info) != 0) {
        return 0.0;
    }
#if defined(__APPLE__)
    return (double)info.st_mtimespec.tv_sec * 1000.0 + (double)info.st_mtimespec.tv_nsec / 1000000.0;
#else
    return (double)info.st_mtim.tv_sec * 1000.0 + (double)info.st_mtim.tv_nsec / 1000000.0;
#endif
#endif
}

static const char *jayess_path_permissions_text(const char *path_text) {
#ifdef _WIN32
    (void)path_text;
    return "rwx";
#else
    static char perms[10];
    struct stat info;
    if (path_text == NULL || stat(path_text, &info) != 0) {
        return "";
    }
    perms[0] = (info.st_mode & S_IRUSR) ? 'r' : '-';
    perms[1] = (info.st_mode & S_IWUSR) ? 'w' : '-';
    perms[2] = (info.st_mode & S_IXUSR) ? 'x' : '-';
    perms[3] = (info.st_mode & S_IRGRP) ? 'r' : '-';
    perms[4] = (info.st_mode & S_IWGRP) ? 'w' : '-';
    perms[5] = (info.st_mode & S_IXGRP) ? 'x' : '-';
    perms[6] = (info.st_mode & S_IROTH) ? 'r' : '-';
    perms[7] = (info.st_mode & S_IWOTH) ? 'w' : '-';
    perms[8] = (info.st_mode & S_IXOTH) ? 'x' : '-';
    perms[9] = '\0';
    return perms;
#endif
}

static jayess_value *jayess_fs_dir_entry_value(const char *name, const char *full_path, int is_dir) {
    jayess_object *entry = jayess_object_new();
    if (entry == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(entry, "name", jayess_value_from_string(name != NULL ? name : ""));
    jayess_object_set_value(entry, "path", jayess_value_from_string(full_path != NULL ? full_path : ""));
    jayess_object_set_value(entry, "isDir", jayess_value_from_bool(is_dir));
    jayess_object_set_value(entry, "isFile", jayess_value_from_bool(!is_dir));
    jayess_object_set_value(entry, "size", jayess_value_from_number(jayess_path_file_size_text(full_path)));
    jayess_object_set_value(entry, "mtimeMs", jayess_value_from_number(jayess_path_modified_time_ms_text(full_path)));
    jayess_object_set_value(entry, "permissions", jayess_value_from_string(jayess_path_permissions_text(full_path)));
    return jayess_value_from_object(entry);
}

static void jayess_fs_read_dir_collect(jayess_array *entries, const char *path_text, int recursive) {
    if (entries == NULL || path_text == NULL) {
        return;
    }
#ifdef _WIN32
    WIN32_FIND_DATAA find_data;
    HANDLE handle;
    size_t length = strlen(path_text);
    char *pattern = (char *)malloc(length + 3);
    if (pattern == NULL) {
        return;
    }
    strcpy(pattern, path_text);
    if (length > 0 && !jayess_path_is_separator(pattern[length - 1])) {
        strcat(pattern, "\\");
    }
    strcat(pattern, "*");
    handle = FindFirstFileA(pattern, &find_data);
    free(pattern);
    if (handle == INVALID_HANDLE_VALUE) {
        return;
    }
    do {
        char *full_path;
        int is_dir;
        if (strcmp(find_data.cFileName, ".") == 0 || strcmp(find_data.cFileName, "..") == 0) {
            continue;
        }
        full_path = (char *)malloc(length + strlen(find_data.cFileName) + 3);
        if (full_path == NULL) {
            continue;
        }
        strcpy(full_path, path_text);
        if (length > 0 && !jayess_path_is_separator(full_path[length - 1])) {
            strcat(full_path, "\\");
        }
        strcat(full_path, find_data.cFileName);
        is_dir = (find_data.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0;
        jayess_array_push_value(entries, jayess_fs_dir_entry_value(find_data.cFileName, full_path, is_dir));
        if (recursive && is_dir) {
            jayess_fs_read_dir_collect(entries, full_path, recursive);
        }
        free(full_path);
    } while (FindNextFileA(handle, &find_data));
    FindClose(handle);
#else
    DIR *dir = opendir(path_text);
    if (dir != NULL) {
        struct dirent *entry;
        size_t path_len = strlen(path_text);
        while ((entry = readdir(dir)) != NULL) {
            char *full_path;
            int is_dir;
            if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
                continue;
            }
            full_path = (char *)malloc(path_len + strlen(entry->d_name) + 3);
            if (full_path == NULL) {
                continue;
            }
            strcpy(full_path, path_text);
            if (path_len > 0 && !jayess_path_is_separator(full_path[path_len - 1])) {
                strcat(full_path, "/");
            }
            strcat(full_path, entry->d_name);
            is_dir = jayess_path_is_dir_text(full_path);
            jayess_array_push_value(entries, jayess_fs_dir_entry_value(entry->d_name, full_path, is_dir));
            if (recursive && is_dir) {
                jayess_fs_read_dir_collect(entries, full_path, recursive);
            }
            free(full_path);
        }
        closedir(dir);
    }
#endif
}

static void jayess_print_value_inline(jayess_value *value);
static jayess_array *jayess_array_clone(jayess_array *array);
static jayess_array *jayess_array_concat(jayess_array *left, jayess_array *right);
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

void jayess_print_string(const char *text) {
    if (text == NULL) {
        return;
    }
    puts(text);
}

void jayess_print_number(double value) {
    printf("%g\n", value);
}

void jayess_print_bool(int value) {
    puts(value ? "true" : "false");
}

void jayess_print_object(jayess_object *object) {
    int first = 1;
    jayess_object_entry *current;

    if (object == NULL) {
        puts("{}");
        return;
    }

    putchar('{');
    current = object->head;
    while (current != NULL) {
        if (!first) {
            fputs(", ", stdout);
        }
        fputs(current->key, stdout);
        fputs(": ", stdout);
        jayess_print_value_inline(current->value);
        first = 0;
        current = current->next;
    }
    puts("}");
}

void jayess_print_array(jayess_array *array) {
    int i;

    if (array == NULL) {
        puts("[]");
        return;
    }

    putchar('[');
    for (i = 0; i < array->count; i++) {
        if (i > 0) {
            fputs(", ", stdout);
        }
        jayess_print_value_inline(array->values[i]);
    }
    puts("]");
}

void jayess_print_args(jayess_args *args) {
    int i;

    if (args == NULL) {
        puts("[]");
        return;
    }

    putchar('[');
    for (i = 0; i < args->count; i++) {
        if (i > 0) {
            fputs(", ", stdout);
        }
        fputs(args->values[i] != NULL ? args->values[i] : "", stdout);
    }
    puts("]");
}

void jayess_print_value(jayess_value *value) {
    jayess_print_value_inline(value);
    putchar('\n');
}

void jayess_print_many(jayess_value *values) {
    int i;
    if (values == NULL || values->kind != JAYESS_VALUE_ARRAY || values->as.array_value == NULL) {
        putchar('\n');
        return;
    }
    for (i = 0; i < values->as.array_value->count; i++) {
        if (i > 0) {
            putchar(' ');
        }
        jayess_print_value_inline(values->as.array_value->values[i]);
    }
    putchar('\n');
}

static void jayess_console_write(jayess_value *values, FILE *stream) {
    int i;
    if (values == NULL || values->kind != JAYESS_VALUE_ARRAY || values->as.array_value == NULL) {
        fputc('\n', stream);
        return;
    }
    for (i = 0; i < values->as.array_value->count; i++) {
        char *text = jayess_value_stringify(values->as.array_value->values[i]);
        if (i > 0) {
            fputc(' ', stream);
        }
        fputs(text != NULL ? text : "", stream);
        free(text);
    }
    fputc('\n', stream);
}

void jayess_console_log(jayess_value *values) { jayess_console_write(values, stdout); }
void jayess_console_warn(jayess_value *values) { jayess_console_write(values, stderr); }
void jayess_console_error(jayess_value *values) { jayess_console_write(values, stderr); }

char *jayess_value_stringify(jayess_value *value) {
    if (value == NULL) {
        return jayess_strdup("null");
    }
    switch (value->kind) {
    case JAYESS_VALUE_NULL:
        return jayess_strdup("null");
    case JAYESS_VALUE_UNDEFINED:
        return jayess_strdup("undefined");
    case JAYESS_VALUE_STRING:
        return jayess_strdup(value->as.string_value != NULL ? value->as.string_value : "");
    case JAYESS_VALUE_NUMBER:
        return jayess_number_to_string(value->as.number_value);
    case JAYESS_VALUE_BOOL:
        return jayess_strdup(value->as.bool_value ? "true" : "false");
    case JAYESS_VALUE_FUNCTION:
        if (value->as.function_value != NULL && value->as.function_value->name != NULL) {
            size_t length = strlen(value->as.function_value->name) + 12;
            char *out = (char *)malloc(length);
            if (out == NULL) {
                return NULL;
            }
            snprintf(out, length, "[Function %s]", value->as.function_value->name);
            return out;
        }
        return jayess_strdup("[Function]");
    case JAYESS_VALUE_OBJECT:
        return jayess_strdup("[object Object]");
    case JAYESS_VALUE_ARRAY: {
        int i;
        size_t total = 3;
        char *out;
        if (value->as.array_value == NULL) {
            return jayess_strdup("[]");
        }
        for (i = 0; i < value->as.array_value->count; i++) {
            char *item = jayess_value_stringify(value->as.array_value->values[i]);
            total += item != NULL ? strlen(item) : 0;
            if (i > 0) {
                total += 2;
            }
            free(item);
        }
        out = (char *)malloc(total);
        if (out == NULL) {
            return NULL;
        }
        out[0] = '\0';
        strcat(out, "[");
        for (i = 0; i < value->as.array_value->count; i++) {
            char *item = jayess_value_stringify(value->as.array_value->values[i]);
            if (i > 0) {
                strcat(out, ", ");
            }
            if (item != NULL) {
                strcat(out, item);
                free(item);
            }
        }
        strcat(out, "]");
        return out;
    }
    default:
        return jayess_strdup("");
    }
}

char *jayess_template_string(jayess_value *parts, jayess_value *values) {
    int i;
    int part_count = 0;
    int value_count = 0;
    size_t total = 1;
    char *result;
    if (parts != NULL && parts->kind == JAYESS_VALUE_ARRAY && parts->as.array_value != NULL) {
        part_count = parts->as.array_value->count;
    }
    if (values != NULL && values->kind == JAYESS_VALUE_ARRAY && values->as.array_value != NULL) {
        value_count = values->as.array_value->count;
    }
    for (i = 0; i < part_count; i++) {
        char *text = jayess_value_stringify(parts->as.array_value->values[i]);
        if (text != NULL) {
            total += strlen(text);
            free(text);
        }
        if (i < value_count) {
            char *value_text = jayess_value_stringify(values->as.array_value->values[i]);
            if (value_text != NULL) {
                total += strlen(value_text);
                free(value_text);
            }
        }
    }
    result = (char *)malloc(total);
    if (result == NULL) {
        return NULL;
    }
    result[0] = '\0';
    for (i = 0; i < part_count; i++) {
        char *text = jayess_value_stringify(parts->as.array_value->values[i]);
        if (text != NULL) {
            strcat(result, text);
            free(text);
        }
        if (i < value_count) {
            char *value_text = jayess_value_stringify(values->as.array_value->values[i]);
            if (value_text != NULL) {
                strcat(result, value_text);
                free(value_text);
            }
        }
    }
    return result;
}

char *jayess_concat_values(jayess_value *left, jayess_value *right) {
    char *left_text = jayess_value_stringify(left);
    char *right_text = jayess_value_stringify(right);
    size_t left_len = left_text != NULL ? strlen(left_text) : 0;
    size_t right_len = right_text != NULL ? strlen(right_text) : 0;
    char *result = (char *)malloc(left_len + right_len + 1);
    if (result == NULL) {
        free(left_text);
        free(right_text);
        return NULL;
    }
    if (left_len > 0) {
        memcpy(result, left_text, left_len);
    }
    if (right_len > 0) {
        memcpy(result + left_len, right_text, right_len);
    }
    result[left_len + right_len] = '\0';
    free(left_text);
    free(right_text);
    return result;
}

static void jayess_print_value_inline(jayess_value *value) {
    if (value == NULL) {
        fputs("null", stdout);
        return;
    }

    switch (value->kind) {
    case JAYESS_VALUE_NULL:
        fputs("null", stdout);
        break;
    case JAYESS_VALUE_STRING:
        fputs(value->as.string_value != NULL ? value->as.string_value : "", stdout);
        break;
    case JAYESS_VALUE_NUMBER:
        printf("%g", value->as.number_value);
        break;
    case JAYESS_VALUE_BOOL:
        fputs(value->as.bool_value ? "true" : "false", stdout);
        break;
    case JAYESS_VALUE_OBJECT:
        if (value->as.object_value == NULL) {
            fputs("{}", stdout);
        } else {
            int first = 1;
            jayess_object_entry *current = value->as.object_value->head;
            putchar('{');
            while (current != NULL) {
                if (!first) {
                    fputs(", ", stdout);
                }
                fputs(current->key, stdout);
                fputs(": ", stdout);
                jayess_print_value_inline(current->value);
                first = 0;
                current = current->next;
            }
            putchar('}');
        }
        break;
    case JAYESS_VALUE_ARRAY:
        if (value->as.array_value == NULL) {
            fputs("[]", stdout);
        } else {
            int i;
            putchar('[');
            for (i = 0; i < value->as.array_value->count; i++) {
                if (i > 0) {
                    fputs(", ", stdout);
                }
                jayess_print_value_inline(value->as.array_value->values[i]);
            }
            putchar(']');
        }
        break;
    case JAYESS_VALUE_UNDEFINED:
        fputs("undefined", stdout);
        break;
    case JAYESS_VALUE_FUNCTION:
        if (value->as.function_value != NULL && value->as.function_value->name != NULL) {
            printf("[Function %s]", value->as.function_value->name);
        } else {
            fputs("[Function]", stdout);
        }
        break;
    default:
        fputs("", stdout);
        break;
    }
}

char *jayess_read_line(const char *prompt) {
    char buffer[1024];
    size_t length;
    char *result;

    if (prompt != NULL) {
        fputs(prompt, stdout);
        fflush(stdout);
    }

    if (fgets(buffer, sizeof(buffer), stdin) == NULL) {
        result = (char *)malloc(1);
        if (result != NULL) {
            result[0] = '\0';
        }
        return result;
    }

    length = strlen(buffer);
    if (length > 0 && buffer[length - 1] == '\n') {
        buffer[length - 1] = '\0';
        length--;
    }

    result = (char *)malloc(length + 1);
    if (result == NULL) {
        return NULL;
    }

    memcpy(result, buffer, length + 1);
    return result;
}

char *jayess_read_key(const char *prompt) {
    int value;
    char *result;

    if (prompt != NULL) {
        fputs(prompt, stdout);
        fflush(stdout);
    }

#ifdef _WIN32
    value = _getch();
#else
    struct termios original;
    struct termios raw;

    if (tcgetattr(STDIN_FILENO, &original) != 0) {
        value = getchar();
    } else {
        raw = original;
        raw.c_lflag &= (tcflag_t) ~(ICANON | ECHO);
        raw.c_cc[VMIN] = 1;
        raw.c_cc[VTIME] = 0;

        if (tcsetattr(STDIN_FILENO, TCSANOW, &raw) != 0) {
            value = getchar();
        } else {
            unsigned char ch = 0;
            ssize_t bytes_read = read(STDIN_FILENO, &ch, 1);
            tcsetattr(STDIN_FILENO, TCSANOW, &original);
            if (bytes_read == 1) {
                value = (int)ch;
            } else {
                value = EOF;
            }
        }
    }
#endif

    result = (char *)malloc(2);
    if (result == NULL) {
        return NULL;
    }
    if (value == EOF) {
        result[0] = '\0';
        result[1] = '\0';
        return result;
    }
    result[0] = (char)value;
    result[1] = '\0';
    return result;
}

void jayess_sleep_ms(int milliseconds) {
    if (milliseconds <= 0) {
        return;
    }
#ifdef _WIN32
    Sleep((DWORD)milliseconds);
#else
    usleep((useconds_t)milliseconds * 1000);
#endif
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

jayess_object *jayess_object_new(void) {
    jayess_object *object = (jayess_object *)malloc(sizeof(jayess_object));
    if (object == NULL) {
        return NULL;
    }
    object->head = NULL;
    object->promise_dependents = NULL;
    object->stream_file = NULL;
    object->socket_handle = JAYESS_INVALID_SOCKET;
    object->native_handle = NULL;
    return object;
}

static jayess_object_entry *jayess_object_find(jayess_object *object, const char *key) {
    jayess_object_entry *current = object != NULL ? object->head : NULL;
    while (current != NULL) {
        if (strcmp(current->key, key) == 0) {
            return current;
        }
        current = current->next;
    }
    return NULL;
}

void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value) {
    jayess_object_entry *entry;

    if (object == NULL || key == NULL || value == NULL) {
        return;
    }

    entry = jayess_object_find(object, key);
    if (entry == NULL) {
        entry = (jayess_object_entry *)malloc(sizeof(jayess_object_entry));
        if (entry == NULL) {
            return;
        }
        entry->key = jayess_strdup(key);
        entry->value = NULL;
        entry->next = object->head;
        object->head = entry;
    }
    entry->value = value;
}

jayess_value *jayess_object_get(jayess_object *object, const char *key) {
    jayess_object_entry *entry;

    if (object == NULL || key == NULL) {
        return NULL;
    }

    entry = jayess_object_find(object, key);
    if (entry == NULL) {
        return NULL;
    }

    return entry->value;
}

void jayess_object_delete(jayess_object *object, const char *key) {
    jayess_object_entry *current;
    jayess_object_entry *previous;

    if (object == NULL || key == NULL) {
        return;
    }

    previous = NULL;
    current = object->head;
    while (current != NULL) {
        if (strcmp(current->key, key) == 0) {
            if (previous == NULL) {
                object->head = current->next;
            } else {
                previous->next = current->next;
            }
            free(current->key);
            free(current);
            return;
        }
        previous = current;
        current = current->next;
    }
}

jayess_array *jayess_object_keys(jayess_object *object) {
    int index = 0;
    jayess_object_entry *current;
    jayess_array *keys = jayess_array_new();
    if (keys == NULL || object == NULL) {
        return keys;
    }
    current = object->head;
    while (current != NULL) {
        jayess_array_set_value(keys, index++, jayess_value_from_string(current->key));
        current = current->next;
    }
    return keys;
}

static int jayess_std_kind_is(jayess_value *target, const char *kind) {
    jayess_value *kind_value;
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return 0;
    }
    kind_value = jayess_object_get(target->as.object_value, "__jayess_std_kind");
    return kind_value != NULL && kind_value->kind == JAYESS_VALUE_STRING && strcmp(kind_value->as.string_value, kind) == 0;
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

jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message) {
    jayess_object *object = jayess_object_new();
    const char *name_text = "Error";
    char *message_text = NULL;
    if (name != NULL && name->kind == JAYESS_VALUE_STRING && name->as.string_value != NULL) {
        name_text = name->as.string_value;
    }
    if (message != NULL && message->kind != JAYESS_VALUE_UNDEFINED && message->kind != JAYESS_VALUE_NULL) {
        message_text = jayess_value_stringify(message);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(name_text));
    jayess_object_set_value(object, "name", jayess_value_from_string(name_text));
    jayess_object_set_value(object, "message", jayess_value_from_string(message_text != NULL ? message_text : ""));
    if (message_text != NULL) {
        free(message_text);
    }
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_aggregate_error_new(jayess_value *errors, jayess_value *message) {
    jayess_object *object = jayess_object_new();
    char *message_text = NULL;
    jayess_value *error_values = jayess_value_iterable_values(errors);
    if (message != NULL && message->kind != JAYESS_VALUE_UNDEFINED && message->kind != JAYESS_VALUE_NULL) {
        message_text = jayess_value_stringify(message);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("AggregateError"));
    jayess_object_set_value(object, "name", jayess_value_from_string("AggregateError"));
    jayess_object_set_value(object, "message", jayess_value_from_string(message_text != NULL ? message_text : ""));
    jayess_object_set_value(object, "errors", error_values != NULL ? error_values : jayess_value_from_array(jayess_array_new()));
    if (message_text != NULL) {
        free(message_text);
    }
    return jayess_value_from_object(object);
}

static jayess_value *jayess_type_error_value(const char *message) {
    return jayess_std_error_new(jayess_value_from_string("TypeError"), jayess_value_from_string(message != NULL ? message : ""));
}

jayess_value *jayess_std_array_buffer_new(jayess_value *length_value) {
    jayess_object *object = jayess_object_new();
    jayess_array *bytes = jayess_array_new();
    int length = (int)jayess_value_to_number(length_value);
    int i;
    if (length < 0) {
        length = 0;
    }
    for (i = 0; i < length; i++) {
        jayess_array_push_value(bytes, jayess_value_from_number(0));
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("ArrayBuffer"));
    jayess_object_set_value(object, "__jayess_bytes", jayess_value_from_array(bytes));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_uint8_array_new(jayess_value *source) {
	jayess_object *object = jayess_object_new();
	jayess_value *buffer = NULL;
    jayess_array *bytes = NULL;
    if (source != NULL && source->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(source, "ArrayBuffer")) {
        buffer = source;
        jayess_value *stored = jayess_object_get(source->as.object_value, "__jayess_bytes");
        if (stored != NULL && stored->kind == JAYESS_VALUE_ARRAY) {
            bytes = stored->as.array_value;
        }
    }
    if (bytes == NULL) {
        int length = (int)jayess_value_to_number(source);
        if (length < 0) {
            length = 0;
        }
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)length));
        if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
            jayess_value *stored = jayess_object_get(buffer->as.object_value, "__jayess_bytes");
            if (stored != NULL && stored->kind == JAYESS_VALUE_ARRAY) {
                bytes = stored->as.array_value;
            }
        }
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Uint8Array"));
    jayess_object_set_value(object, "__jayess_bytes", jayess_value_from_array(bytes));
    jayess_object_set_value(object, "buffer", buffer != NULL ? buffer : jayess_std_array_buffer_new(jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0))));
    jayess_object_set_value(object, "length", jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0)));
	jayess_object_set_value(object, "byteLength", jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0)));
	return jayess_value_from_object(object);
}

jayess_value *jayess_std_data_view_new(jayess_value *buffer) {
	jayess_object *object = jayess_object_new();
	jayess_array *bytes = NULL;
	if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(buffer, "ArrayBuffer")) {
		jayess_value *stored = jayess_object_get(buffer->as.object_value, "__jayess_bytes");
		if (stored != NULL && stored->kind == JAYESS_VALUE_ARRAY) {
			bytes = stored->as.array_value;
		}
	}
	if (object == NULL) {
		return jayess_value_from_object(NULL);
	}
	if (bytes == NULL) {
		buffer = jayess_std_array_buffer_new(jayess_value_from_number(0));
		if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
			jayess_value *stored = jayess_object_get(buffer->as.object_value, "__jayess_bytes");
			if (stored != NULL && stored->kind == JAYESS_VALUE_ARRAY) {
				bytes = stored->as.array_value;
			}
		}
	}
	jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("DataView"));
	jayess_object_set_value(object, "__jayess_bytes", jayess_value_from_array(bytes));
	jayess_object_set_value(object, "buffer", buffer);
	jayess_object_set_value(object, "byteLength", jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0)));
	return jayess_value_from_object(object);
}

static jayess_value *jayess_std_uint8_array_from_bytes(const unsigned char *bytes, size_t length) {
    jayess_value *buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)length));
    jayess_value *view = jayess_std_uint8_array_new(buffer);
    jayess_array *out = jayess_std_bytes_slot(view);
    size_t i;
    if (out == NULL) {
        return view;
    }
    for (i = 0; i < length; i++) {
        jayess_array_set_value(out, (int)i, jayess_value_from_number((double)bytes[i]));
    }
    return view;
}

static int jayess_std_bytes_encoding_is_hex(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 0;
    }
    text = jayess_value_stringify(encoding);
    ok = text != NULL && strcmp(text, "hex") == 0;
    free(text);
    return ok;
}

static int jayess_std_bytes_encoding_is_base64(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 0;
    }
    text = jayess_value_stringify(encoding);
    ok = text != NULL && (strcmp(text, "base64") == 0 || strcmp(text, "base-64") == 0);
    free(text);
    return ok;
}

static int jayess_std_bytes_encoding_is_text(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 1;
    }
    text = jayess_value_stringify(encoding);
    ok = text == NULL || strcmp(text, "utf8") == 0 || strcmp(text, "utf-8") == 0 || strcmp(text, "text") == 0;
    free(text);
    return ok;
}

static int jayess_std_hex_digit(char value) {
    if (value >= '0' && value <= '9') {
        return value - '0';
    }
    if (value >= 'a' && value <= 'f') {
        return value - 'a' + 10;
    }
    if (value >= 'A' && value <= 'F') {
        return value - 'A' + 10;
    }
    return -1;
}

static int jayess_std_base64_digit(char value) {
    if (value >= 'A' && value <= 'Z') {
        return value - 'A';
    }
    if (value >= 'a' && value <= 'z') {
        return value - 'a' + 26;
    }
    if (value >= '0' && value <= '9') {
        return value - '0' + 52;
    }
    if (value == '+') {
        return 62;
    }
    if (value == '/') {
        return 63;
    }
    return -1;
}

jayess_value *jayess_std_uint8_array_from_string(jayess_value *source, jayess_value *encoding) {
    char *text = jayess_value_stringify(source);
    size_t length;
    jayess_value *buffer;
    jayess_value *view;
    jayess_array *bytes;
    size_t i;
    if (text == NULL) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
    length = strlen(text);
    if (jayess_std_bytes_encoding_is_hex(encoding)) {
        if (length % 2 != 0) {
            free(text);
            return jayess_value_undefined();
        }
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)(length / 2)));
        view = jayess_std_uint8_array_new(buffer);
        bytes = jayess_std_bytes_slot(view);
        if (bytes == NULL) {
            free(text);
            return jayess_value_undefined();
        }
        for (i = 0; i < length; i += 2) {
            int high = jayess_std_hex_digit(text[i]);
            int low = jayess_std_hex_digit(text[i + 1]);
            if (high < 0 || low < 0) {
                free(text);
                return jayess_value_undefined();
            }
            jayess_array_set_value(bytes, (int)(i / 2), jayess_value_from_number((double)((high << 4) | low)));
        }
        free(text);
        return view;
    }
    if (jayess_std_bytes_encoding_is_base64(encoding)) {
        size_t clean_length = 0;
        size_t padding = 0;
        size_t out_length;
        size_t out_index = 0;
        int quartet[4];
        int quartet_count = 0;
        for (i = 0; i < length; i++) {
            if (isspace((unsigned char)text[i])) {
                continue;
            }
            if (text[i] == '=') {
                padding++;
            } else if (padding > 0 || jayess_std_base64_digit(text[i]) < 0) {
                free(text);
                return jayess_value_undefined();
            }
            clean_length++;
        }
        if (clean_length == 0) {
            free(text);
            return jayess_std_uint8_array_new(jayess_value_from_number(0));
        }
        if (clean_length % 4 != 0 || padding > 2) {
            free(text);
            return jayess_value_undefined();
        }
        out_length = (clean_length / 4) * 3;
        out_length -= padding;
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)out_length));
        view = jayess_std_uint8_array_new(buffer);
        bytes = jayess_std_bytes_slot(view);
        if (bytes == NULL) {
            free(text);
            return jayess_value_undefined();
        }
        for (i = 0; i < length; i++) {
            int value;
            if (isspace((unsigned char)text[i])) {
                continue;
            }
            value = text[i] == '=' ? 0 : jayess_std_base64_digit(text[i]);
            if (value < 0) {
                free(text);
                return jayess_value_undefined();
            }
            quartet[quartet_count++] = value;
            if (quartet_count == 4) {
                unsigned int triple = ((unsigned int)quartet[0] << 18) | ((unsigned int)quartet[1] << 12) | ((unsigned int)quartet[2] << 6) | (unsigned int)quartet[3];
                if (out_index < out_length) {
                    jayess_array_set_value(bytes, (int)out_index++, jayess_value_from_number((double)((triple >> 16) & 255)));
                }
                if (out_index < out_length) {
                    jayess_array_set_value(bytes, (int)out_index++, jayess_value_from_number((double)((triple >> 8) & 255)));
                }
                if (out_index < out_length) {
                    jayess_array_set_value(bytes, (int)out_index++, jayess_value_from_number((double)(triple & 255)));
                }
                quartet_count = 0;
            }
        }
        free(text);
        return view;
    }
    if (!jayess_std_bytes_encoding_is_text(encoding)) {
        free(text);
        return jayess_value_undefined();
    }
    buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)length));
    view = jayess_std_uint8_array_new(buffer);
    bytes = jayess_std_bytes_slot(view);
    if (bytes == NULL) {
        free(text);
        return jayess_value_undefined();
    }
    for (i = 0; i < length; i++) {
        jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)(unsigned char)text[i]));
    }
    free(text);
    return view;
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

static jayess_value *jayess_value_call_one(jayess_value *callback, jayess_value *argument) {
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
        guard++;
    }
    if (guard >= 100000) {
        jayess_throw(jayess_type_error_value("microtask queue did not settle"));
    }
}

void jayess_runtime_shutdown(void) {
    jayess_io_pool_shutdown();
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
    if (target == NULL) {
        return;
    }
    if (target->kind == JAYESS_VALUE_OBJECT) {
        jayess_object_set_value(target->as.object_value, key, value);
        return;
    }
    if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        jayess_object_set_value(target->as.function_value->properties, key, value);
    }
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

static int jayess_std_bytes_read(jayess_array *bytes, int offset) {
	if (bytes == NULL || offset < 0 || offset >= bytes->count) {
		return 0;
	}
	return (int)jayess_value_to_number(jayess_array_get(bytes, offset)) & 255;
}

static void jayess_std_bytes_write(jayess_array *bytes, int offset, int value) {
	if (bytes == NULL || offset < 0 || offset >= bytes->count) {
		return;
	}
	jayess_array_set_value(bytes, offset, jayess_value_from_number((double)(value & 255)));
}

static unsigned int jayess_std_data_view_read_u32(jayess_array *bytes, int offset, int little_endian) {
	unsigned int b0 = (unsigned int)jayess_std_bytes_read(bytes, offset);
	unsigned int b1 = (unsigned int)jayess_std_bytes_read(bytes, offset + 1);
	unsigned int b2 = (unsigned int)jayess_std_bytes_read(bytes, offset + 2);
	unsigned int b3 = (unsigned int)jayess_std_bytes_read(bytes, offset + 3);
	return little_endian ? (b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)) : ((b0 << 24) | (b1 << 16) | (b2 << 8) | b3);
}

static void jayess_std_data_view_write_u32(jayess_array *bytes, int offset, unsigned int number, int little_endian) {
	if (little_endian) {
		jayess_std_bytes_write(bytes, offset, (int)(number & 255));
		jayess_std_bytes_write(bytes, offset+1, (int)((number >> 8) & 255));
		jayess_std_bytes_write(bytes, offset+2, (int)((number >> 16) & 255));
		jayess_std_bytes_write(bytes, offset+3, (int)((number >> 24) & 255));
	} else {
		jayess_std_bytes_write(bytes, offset, (int)((number >> 24) & 255));
		jayess_std_bytes_write(bytes, offset+1, (int)((number >> 16) & 255));
		jayess_std_bytes_write(bytes, offset+2, (int)((number >> 8) & 255));
		jayess_std_bytes_write(bytes, offset+3, (int)(number & 255));
	}
}

static unsigned long long jayess_std_data_view_read_u64(jayess_array *bytes, int offset, int little_endian) {
	unsigned long long value = 0;
	int i;
	if (little_endian) {
		for (i = 7; i >= 0; i--) {
			value = (value << 8) | (unsigned long long)jayess_std_bytes_read(bytes, offset+i);
		}
	} else {
		for (i = 0; i < 8; i++) {
			value = (value << 8) | (unsigned long long)jayess_std_bytes_read(bytes, offset+i);
		}
	}
	return value;
}

static void jayess_std_data_view_write_u64(jayess_array *bytes, int offset, unsigned long long number, int little_endian) {
	int i;
	if (little_endian) {
		for (i = 0; i < 8; i++) {
			jayess_std_bytes_write(bytes, offset+i, (int)((number >> (i * 8)) & 255ULL));
		}
	} else {
		for (i = 0; i < 8; i++) {
			jayess_std_bytes_write(bytes, offset+i, (int)((number >> ((7 - i) * 8)) & 255ULL));
		}
	}
}

static jayess_array *jayess_std_bytes_slot(jayess_value *target) {
	jayess_value *stored;
	if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return NULL;
    }
    stored = jayess_object_get(target->as.object_value, "__jayess_bytes");
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
	return stored->as.array_value;
}

static jayess_value *jayess_std_data_view_get_uint8_method(jayess_value *env, jayess_value *offset_value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	return jayess_value_from_number((double)jayess_std_bytes_read(bytes, offset));
}

static jayess_value *jayess_std_data_view_set_uint8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int byte_value = (int)jayess_value_to_number(value) & 255;
	jayess_std_bytes_write(bytes, offset, byte_value);
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_int8_method(jayess_value *env, jayess_value *offset_value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int value = jayess_std_bytes_read(bytes, offset);
	if (value >= 128) {
		value -= 256;
	}
	return jayess_value_from_number((double)value);
}

static jayess_value *jayess_std_data_view_set_int8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int byte_value = (int)jayess_value_to_number(value);
	jayess_std_bytes_write(bytes, offset, byte_value);
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int b0 = jayess_std_bytes_read(bytes, offset);
	int b1 = jayess_std_bytes_read(bytes, offset + 1);
	int value = jayess_value_as_bool(little_endian) ? (b0 | (b1 << 8)) : ((b0 << 8) | b1);
	return jayess_value_from_number((double)value);
}

static jayess_value *jayess_std_data_view_set_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int number = (int)jayess_value_to_number(value) & 65535;
	if (jayess_value_as_bool(little_endian)) {
		jayess_std_bytes_write(bytes, offset, number & 255);
		jayess_std_bytes_write(bytes, offset+1, (number >> 8) & 255);
	} else {
		jayess_std_bytes_write(bytes, offset, (number >> 8) & 255);
		jayess_std_bytes_write(bytes, offset+1, number & 255);
	}
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int b0 = jayess_std_bytes_read(bytes, offset);
	int b1 = jayess_std_bytes_read(bytes, offset + 1);
	int value = jayess_value_as_bool(little_endian) ? (b0 | (b1 << 8)) : ((b0 << 8) | b1);
	if (value >= 32768) {
		value -= 65536;
	}
	return jayess_value_from_number((double)value);
}

static jayess_value *jayess_std_data_view_set_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int number = (int)jayess_value_to_number(value) & 65535;
	if (jayess_value_as_bool(little_endian)) {
		jayess_std_bytes_write(bytes, offset, number & 255);
		jayess_std_bytes_write(bytes, offset+1, (number >> 8) & 255);
	} else {
		jayess_std_bytes_write(bytes, offset, (number >> 8) & 255);
		jayess_std_bytes_write(bytes, offset+1, number & 255);
	}
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	unsigned int value = jayess_std_data_view_read_u32(bytes, offset, jayess_value_as_bool(little_endian));
	return jayess_value_from_number((double)value);
}

static jayess_value *jayess_std_data_view_set_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	unsigned int number = (unsigned int)jayess_value_to_number(value);
	jayess_std_data_view_write_u32(bytes, offset, number, jayess_value_as_bool(little_endian));
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	unsigned int value = jayess_std_data_view_read_u32(bytes, offset, jayess_value_as_bool(little_endian));
	long long signed_value = value >= 2147483648U ? (long long)value - 4294967296LL : (long long)value;
	return jayess_value_from_number((double)signed_value);
}

static jayess_value *jayess_std_data_view_set_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	int signed_number = (int)jayess_value_to_number(value);
	unsigned int number = (unsigned int)signed_number;
	jayess_std_data_view_write_u32(bytes, offset, number, jayess_value_as_bool(little_endian));
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	unsigned int bits = jayess_std_data_view_read_u32(bytes, offset, jayess_value_as_bool(little_endian));
	float value = 0.0f;
	memcpy(&value, &bits, sizeof(value));
	return jayess_value_from_number((double)value);
}

static jayess_value *jayess_std_data_view_set_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	float number = (float)jayess_value_to_number(value);
	unsigned int bits = 0;
	memcpy(&bits, &number, sizeof(bits));
	jayess_std_data_view_write_u32(bytes, offset, bits, jayess_value_as_bool(little_endian));
	return jayess_value_undefined();
}

static jayess_value *jayess_std_data_view_get_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	unsigned long long bits = jayess_std_data_view_read_u64(bytes, offset, jayess_value_as_bool(little_endian));
	double value = 0.0;
	memcpy(&value, &bits, sizeof(value));
	return jayess_value_from_number(value);
}

static jayess_value *jayess_std_data_view_set_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
	jayess_array *bytes = jayess_std_bytes_slot(env);
	int offset = (int)jayess_value_to_number(offset_value);
	double number = jayess_value_to_number(value);
	unsigned long long bits = 0;
	memcpy(&bits, &number, sizeof(bits));
	jayess_std_data_view_write_u64(bytes, offset, bits, jayess_value_as_bool(little_endian));
	return jayess_value_undefined();
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

static jayess_value *jayess_std_uint8_to_string_method(jayess_value *env, jayess_value *encoding) {
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

jayess_value *jayess_value_get_member(jayess_value *target, const char *key) {
    if (target == NULL) {
        return NULL;
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
		if (jayess_std_kind_is(target, "ArrayBuffer")) {
			if (strcmp(key, "byteLength") == 0) {
				jayess_array *bytes = jayess_std_bytes_slot(target);
				return jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0));
			}
		}
		if (jayess_std_kind_is(target, "DataView")) {
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
		if (jayess_std_kind_is(target, "Uint8Array")) {
            if (strcmp(key, "length") == 0 || strcmp(key, "byteLength") == 0) {
                jayess_array *bytes = jayess_std_bytes_slot(target);
                return jayess_value_from_number((double)(bytes != NULL ? bytes->count : 0));
            }
            if (strcmp(key, "fill") == 0) {
                return jayess_value_from_function((void *)jayess_std_uint8_fill_method, target, "fill", NULL, 1, 0);
            }
			if (strcmp(key, "includes") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_includes_method, target, "includes", NULL, 1, 0);
			}
			if (strcmp(key, "indexOf") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_index_of_method, target, "indexOf", NULL, 1, 0);
			}
			if (strcmp(key, "startsWith") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_starts_with_method, target, "startsWith", NULL, 1, 0);
			}
			if (strcmp(key, "endsWith") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_ends_with_method, target, "endsWith", NULL, 1, 0);
			}
			if (strcmp(key, "set") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_set_method, target, "set", NULL, 2, 0);
			}
			if (strcmp(key, "copyWithin") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_copy_within_method, target, "copyWithin", NULL, 3, 0);
			}
			if (strcmp(key, "slice") == 0) {
				return jayess_value_from_function((void *)jayess_std_uint8_slice_method, target, "slice", NULL, 2, 0);
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
            if (strcmp(key, "writableEnded") == 0) {
                return jayess_object_get(target->as.object_value, "writableEnded");
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
        if (jayess_std_kind_is(target, "Socket")) {
            if (strcmp(key, "connected") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "readable") == 0 || strcmp(key, "writable") == 0 || strcmp(key, "timeout") == 0 || strcmp(key, "remoteAddress") == 0 || strcmp(key, "remotePort") == 0 || strcmp(key, "remoteFamily") == 0 || strcmp(key, "localAddress") == 0 || strcmp(key, "localPort") == 0 || strcmp(key, "localFamily") == 0 || strcmp(key, "bytesRead") == 0 || strcmp(key, "bytesWritten") == 0 || strcmp(key, "errored") == 0 || strcmp(key, "error") == 0 || strcmp(key, "secure") == 0 || strcmp(key, "authorized") == 0 || strcmp(key, "backend") == 0 || strcmp(key, "protocol") == 0 || strcmp(key, "alpnProtocol") == 0 || strcmp(key, "alpnProtocols") == 0) {
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
        if (jayess_std_kind_is(target, "Server")) {
            if (strcmp(key, "listening") == 0 || strcmp(key, "closed") == 0 || strcmp(key, "host") == 0 || strcmp(key, "port") == 0 || strcmp(key, "timeout") == 0 || strcmp(key, "connectionsAccepted") == 0) {
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
        return jayess_object_get(target->as.object_value, key);
    }
    if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        return jayess_object_get(target->as.function_value->properties, key);
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

jayess_value *jayess_value_object_rest(jayess_value *target, jayess_value *excluded_keys) {
    jayess_object *source;
    jayess_array *keys;
    jayess_object *copy;
    int i;

    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return jayess_value_from_object(jayess_object_new());
    }

    source = target->as.object_value;
    keys = jayess_object_keys(source);
    copy = jayess_object_new();
    if (copy == NULL) {
        return jayess_value_from_object(NULL);
    }

    for (i = 0; keys != NULL && i < keys->count; i++) {
        jayess_value *key_value = keys->values[i];
        const char *key;
        int skip = 0;
        int j;

        if (key_value == NULL || key_value->kind != JAYESS_VALUE_STRING) {
            continue;
        }
        key = key_value->as.string_value;
        if (excluded_keys != NULL && excluded_keys->kind == JAYESS_VALUE_ARRAY && excluded_keys->as.array_value != NULL) {
            for (j = 0; j < excluded_keys->as.array_value->count; j++) {
                jayess_value *excluded = excluded_keys->as.array_value->values[j];
                if (excluded != NULL && excluded->kind == JAYESS_VALUE_STRING && jayess_string_eq(key, excluded->as.string_value)) {
                    skip = 1;
                    break;
                }
            }
        }
        if (!skip) {
            jayess_object_set_value(copy, key, jayess_object_get(source, key));
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
    if (target->kind == JAYESS_VALUE_OBJECT) {
        if (jayess_std_kind_is(target, "Map")) {
            return jayess_std_map_entries_method(target);
        }
        if (jayess_std_kind_is(target, "Set")) {
            return jayess_std_set_values_method(target);
        }
        if (jayess_std_kind_is(target, "Uint8Array")) {
            jayess_array *bytes = jayess_std_bytes_slot(target);
            return jayess_value_from_array(jayess_array_clone(bytes));
        }
        if (jayess_std_kind_is(target, "Iterator")) {
            jayess_value *values = jayess_object_get(target->as.object_value, "__jayess_iterator_values");
            if (values != NULL && values->kind == JAYESS_VALUE_ARRAY) {
                return jayess_value_from_array(jayess_array_clone(values->as.array_value));
            }
        }
    }
    return jayess_value_from_array(jayess_array_new());
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
    jayess_value *keys_value;
    int i;
    if (target == NULL || source == NULL) {
        return target != NULL ? target : jayess_value_undefined();
    }
    keys_value = jayess_value_object_keys(source);
    if (keys_value == NULL || keys_value->kind != JAYESS_VALUE_ARRAY || keys_value->as.array_value == NULL) {
        return target;
    }
    for (i = 0; i < keys_value->as.array_value->count; i++) {
        jayess_value *key = keys_value->as.array_value->values[i];
        if (key != NULL && key->kind == JAYESS_VALUE_STRING) {
            jayess_value_set_member(target, key->as.string_value, jayess_value_get_member(source, key->as.string_value));
        }
    }
    return target;
}

jayess_value *jayess_value_object_has_own(jayess_value *target, jayess_value *key) {
    char *text;
    jayess_value *value = NULL;
    if (target == NULL || key == NULL) {
        return jayess_value_from_bool(0);
    }
    text = jayess_value_stringify(key);
    if (text == NULL) {
        return jayess_value_from_bool(0);
    }
    if (target->kind == JAYESS_VALUE_OBJECT && target->as.object_value != NULL) {
        value = jayess_object_get(target->as.object_value, text);
    } else if (target->kind == JAYESS_VALUE_FUNCTION && target->as.function_value != NULL) {
        value = jayess_object_get(target->as.function_value->properties, text);
    }
    free(text);
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

static FILE *jayess_std_stream_file(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    return env->as.object_value->stream_file;
}

static jayess_socket_handle jayess_std_socket_handle(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return JAYESS_INVALID_SOCKET;
    }
    return env->as.object_value->socket_handle;
}

static void jayess_std_socket_set_handle(jayess_value *env, jayess_socket_handle handle) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        env->as.object_value->socket_handle = handle;
    }
}

static jayess_tls_socket_state *jayess_std_tls_state(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    if (!jayess_std_kind_is(env, "Socket")) {
        return NULL;
    }
    return (jayess_tls_socket_state *)env->as.object_value->native_handle;
}

static jayess_value *jayess_std_tls_peer_certificate(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (state == NULL) {
        return jayess_value_undefined();
    }
#ifdef _WIN32
    {
        PCCERT_CONTEXT cert = NULL;
        char subject[512];
        char issuer[512];
        char subject_cn[256];
        char issuer_cn[256];
        char serial[256];
        char valid_from[64];
        char valid_to[64];
        jayess_object *result;
        SECURITY_STATUS status = QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert);
        if (status != SEC_E_OK || cert == NULL) {
            return jayess_value_undefined();
        }
        subject[0] = '\0';
        issuer[0] = '\0';
        subject_cn[0] = '\0';
        issuer_cn[0] = '\0';
        serial[0] = '\0';
        valid_from[0] = '\0';
        valid_to[0] = '\0';
        CertGetNameStringA(cert, CERT_NAME_SIMPLE_DISPLAY_TYPE, 0, NULL, subject, (DWORD)sizeof(subject));
        CertGetNameStringA(cert, CERT_NAME_SIMPLE_DISPLAY_TYPE, CERT_NAME_ISSUER_FLAG, NULL, issuer, (DWORD)sizeof(issuer));
        CertGetNameStringA(cert, CERT_NAME_ATTR_TYPE, 0, szOID_COMMON_NAME, subject_cn, (DWORD)sizeof(subject_cn));
        CertGetNameStringA(cert, CERT_NAME_ATTR_TYPE, CERT_NAME_ISSUER_FLAG, szOID_COMMON_NAME, issuer_cn, (DWORD)sizeof(issuer_cn));
        {
            int i;
            size_t offset = 0;
            for (i = (int)cert->pCertInfo->SerialNumber.cbData - 1; i >= 0 && offset + 2 < sizeof(serial); i--) {
                offset += (size_t)snprintf(serial + offset, sizeof(serial) - offset, "%02X", cert->pCertInfo->SerialNumber.pbData[i]);
            }
        }
        {
            SYSTEMTIME from_system;
            SYSTEMTIME to_system;
            if (FileTimeToSystemTime(&cert->pCertInfo->NotBefore, &from_system)) {
                snprintf(valid_from, sizeof(valid_from), "%04u-%02u-%02uT%02u:%02u:%02uZ",
                    (unsigned int)from_system.wYear, (unsigned int)from_system.wMonth, (unsigned int)from_system.wDay,
                    (unsigned int)from_system.wHour, (unsigned int)from_system.wMinute, (unsigned int)from_system.wSecond);
            }
            if (FileTimeToSystemTime(&cert->pCertInfo->NotAfter, &to_system)) {
                snprintf(valid_to, sizeof(valid_to), "%04u-%02u-%02uT%02u:%02u:%02uZ",
                    (unsigned int)to_system.wYear, (unsigned int)to_system.wMonth, (unsigned int)to_system.wDay,
                    (unsigned int)to_system.wHour, (unsigned int)to_system.wMinute, (unsigned int)to_system.wSecond);
            }
        }
        result = jayess_object_new();
        if (result == NULL) {
            CertFreeCertificateContext(cert);
            return jayess_value_from_object(NULL);
        }
        jayess_object_set_value(result, "subject", jayess_value_from_string(subject));
        jayess_object_set_value(result, "issuer", jayess_value_from_string(issuer));
        jayess_object_set_value(result, "subjectCN", jayess_value_from_string(subject_cn));
        jayess_object_set_value(result, "issuerCN", jayess_value_from_string(issuer_cn));
        jayess_object_set_value(result, "serialNumber", jayess_value_from_string(serial));
        jayess_object_set_value(result, "validFrom", jayess_value_from_string(valid_from));
        jayess_object_set_value(result, "validTo", jayess_value_from_string(valid_to));
        jayess_object_set_value(result, "subjectAltNames", jayess_std_tls_subject_alt_names(env));
        jayess_object_set_value(result, "backend", jayess_value_from_string("schannel"));
        jayess_object_set_value(result, "authorized", jayess_object_get(env->as.object_value, "authorized"));
        CertFreeCertificateContext(cert);
        return jayess_value_from_object(result);
    }
#else
    {
        X509 *cert = SSL_get_peer_certificate(state->ssl);
        char subject[512];
        char issuer[512];
        char subject_cn[256];
        char issuer_cn[256];
        char serial[256];
        char valid_from[64];
        char valid_to[64];
        jayess_object *result;
        if (cert == NULL) {
            return jayess_value_undefined();
        }
        subject[0] = '\0';
        issuer[0] = '\0';
        subject_cn[0] = '\0';
        issuer_cn[0] = '\0';
        serial[0] = '\0';
        valid_from[0] = '\0';
        valid_to[0] = '\0';
        X509_NAME_oneline(X509_get_subject_name(cert), subject, (int)sizeof(subject));
        X509_NAME_oneline(X509_get_issuer_name(cert), issuer, (int)sizeof(issuer));
        X509_NAME_get_text_by_NID(X509_get_subject_name(cert), NID_commonName, subject_cn, (int)sizeof(subject_cn));
        X509_NAME_get_text_by_NID(X509_get_issuer_name(cert), NID_commonName, issuer_cn, (int)sizeof(issuer_cn));
        {
            ASN1_INTEGER *serial_number = X509_get_serialNumber(cert);
            BIGNUM *bn = ASN1_INTEGER_to_BN(serial_number, NULL);
            if (bn != NULL) {
                char *hex = BN_bn2hex(bn);
                if (hex != NULL) {
                    snprintf(serial, sizeof(serial), "%s", hex);
                    OPENSSL_free(hex);
                }
                BN_free(bn);
            }
        }
        {
            const ASN1_TIME *not_before = X509_get0_notBefore(cert);
            const ASN1_TIME *not_after = X509_get0_notAfter(cert);
            BIO *bio = BIO_new(BIO_s_mem());
            if (bio != NULL) {
                if (not_before != NULL && ASN1_TIME_print(bio, not_before)) {
                    int len = BIO_read(bio, valid_from, (int)sizeof(valid_from) - 1);
                    if (len > 0) {
                        valid_from[len] = '\0';
                    }
                }
                (void)BIO_reset(bio);
                if (not_after != NULL && ASN1_TIME_print(bio, not_after)) {
                    int len = BIO_read(bio, valid_to, (int)sizeof(valid_to) - 1);
                    if (len > 0) {
                        valid_to[len] = '\0';
                    }
                }
                BIO_free(bio);
            }
        }
        result = jayess_object_new();
        if (result == NULL) {
            X509_free(cert);
            return jayess_value_from_object(NULL);
        }
        jayess_object_set_value(result, "subject", jayess_value_from_string(subject));
        jayess_object_set_value(result, "issuer", jayess_value_from_string(issuer));
        jayess_object_set_value(result, "subjectCN", jayess_value_from_string(subject_cn));
        jayess_object_set_value(result, "issuerCN", jayess_value_from_string(issuer_cn));
        jayess_object_set_value(result, "serialNumber", jayess_value_from_string(serial));
        jayess_object_set_value(result, "validFrom", jayess_value_from_string(valid_from));
        jayess_object_set_value(result, "validTo", jayess_value_from_string(valid_to));
        jayess_object_set_value(result, "subjectAltNames", jayess_std_tls_subject_alt_names(env));
        jayess_object_set_value(result, "backend", jayess_value_from_string("openssl"));
        jayess_object_set_value(result, "authorized", jayess_object_get(env->as.object_value, "authorized"));
        X509_free(cert);
        return jayess_value_from_object(result);
    }
#endif
}

static int jayess_std_tls_send_all(jayess_socket_handle handle, const unsigned char *buffer, size_t length) {
    size_t offset = 0;
    while (offset < length) {
        int sent = (int)send(handle, (const char *)buffer + offset, (int)(length - offset), 0);
        if (sent <= 0) {
            return 0;
        }
        offset += (size_t)sent;
    }
    return 1;
}

static int jayess_std_tls_state_free(jayess_tls_socket_state *state, int close_handle) {
    if (state == NULL) {
        return 1;
    }
#ifdef _WIN32
    if (state->has_context) {
        DeleteSecurityContext(&state->context);
        state->has_context = 0;
    }
    if (state->has_credentials) {
        FreeCredentialHandle(&state->credentials);
        state->has_credentials = 0;
    }
    free(state->encrypted_buffer);
    free(state->plaintext_buffer);
    free(state->host);
    state->encrypted_buffer = NULL;
    state->plaintext_buffer = NULL;
    state->host = NULL;
    state->encrypted_length = 0;
    state->encrypted_capacity = 0;
    state->plaintext_offset = 0;
    state->plaintext_length = 0;
#else
    if (state->ssl != NULL) {
        SSL_free(state->ssl);
        state->ssl = NULL;
    }
    if (state->ctx != NULL) {
        SSL_CTX_free(state->ctx);
        state->ctx = NULL;
    }
    free(state->host);
    state->host = NULL;
#endif
    if (close_handle && state->handle != JAYESS_INVALID_SOCKET) {
        jayess_std_socket_close_handle(state->handle);
        state->handle = JAYESS_INVALID_SOCKET;
    }
    free(state);
    return 1;
}

static int jayess_std_tls_read_bytes(jayess_value *env, unsigned char *buffer, int max_count, int *did_timeout) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (state == NULL || buffer == NULL || max_count <= 0) {
        return -1;
    }
#ifdef _WIN32
    while (1) {
        if (state->plaintext_offset < state->plaintext_length) {
            size_t available = state->plaintext_length - state->plaintext_offset;
            size_t count = available < (size_t)max_count ? available : (size_t)max_count;
            memcpy(buffer, state->plaintext_buffer + state->plaintext_offset, count);
            state->plaintext_offset += count;
            if (state->plaintext_offset >= state->plaintext_length) {
                state->plaintext_offset = 0;
                state->plaintext_length = 0;
            }
            return (int)count;
        }
        {
            SecBuffer buffers[4];
            SecBufferDesc descriptor;
            SECURITY_STATUS status;
            int i;
            if (state->encrypted_length == 0) {
                if (state->encrypted_capacity < 16384) {
                    unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, 16384);
                    if (grown == NULL) {
                        return -1;
                    }
                    state->encrypted_buffer = grown;
                    state->encrypted_capacity = 16384;
                }
                {
                    int read_count = (int)recv(state->handle, (char *)state->encrypted_buffer, (int)state->encrypted_capacity, 0);
                    if (read_count == 0) {
                        return 0;
                    }
                    if (read_count < 0) {
                        int error_code = WSAGetLastError();
                        if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                            *did_timeout = 1;
                        }
                        return -1;
                    }
                    state->encrypted_length = (size_t)read_count;
                }
            }
            buffers[0].pvBuffer = state->encrypted_buffer;
            buffers[0].cbBuffer = (unsigned long)state->encrypted_length;
            buffers[0].BufferType = SECBUFFER_DATA;
            buffers[1].pvBuffer = NULL;
            buffers[1].cbBuffer = 0;
            buffers[1].BufferType = SECBUFFER_EMPTY;
            buffers[2].pvBuffer = NULL;
            buffers[2].cbBuffer = 0;
            buffers[2].BufferType = SECBUFFER_EMPTY;
            buffers[3].pvBuffer = NULL;
            buffers[3].cbBuffer = 0;
            buffers[3].BufferType = SECBUFFER_EMPTY;
            descriptor.ulVersion = SECBUFFER_VERSION;
            descriptor.cBuffers = 4;
            descriptor.pBuffers = buffers;
            status = DecryptMessage(&state->context, &descriptor, 0, NULL);
            if (status == SEC_E_INCOMPLETE_MESSAGE) {
                if (state->encrypted_length >= state->encrypted_capacity) {
                    size_t new_capacity = state->encrypted_capacity > 0 ? state->encrypted_capacity * 2 : 32768;
                    unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, new_capacity);
                    if (grown == NULL) {
                        return -1;
                    }
                    state->encrypted_buffer = grown;
                    state->encrypted_capacity = new_capacity;
                }
                {
                    int read_count = (int)recv(state->handle, (char *)state->encrypted_buffer + state->encrypted_length, (int)(state->encrypted_capacity - state->encrypted_length), 0);
                    if (read_count == 0) {
                        return 0;
                    }
                    if (read_count < 0) {
                        int error_code = WSAGetLastError();
                        if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                            *did_timeout = 1;
                        }
                        return -1;
                    }
                    state->encrypted_length += (size_t)read_count;
                }
                continue;
            }
            if (status == SEC_I_CONTEXT_EXPIRED) {
                return 0;
            }
            if (status != SEC_E_OK) {
                return -1;
            }
            for (i = 0; i < 4; i++) {
                if (buffers[i].BufferType == SECBUFFER_DATA && buffers[i].cbBuffer > 0) {
                    unsigned char *plain = (unsigned char *)buffers[i].pvBuffer;
                    unsigned long plain_len = buffers[i].cbBuffer;
                    if (state->plaintext_buffer == NULL || state->plaintext_length < plain_len) {
                        unsigned char *grown = (unsigned char *)realloc(state->plaintext_buffer, (size_t)plain_len);
                        if (grown == NULL) {
                            return -1;
                        }
                        state->plaintext_buffer = grown;
                    }
                    memcpy(state->plaintext_buffer, plain, plain_len);
                    state->plaintext_offset = 0;
                    state->plaintext_length = plain_len;
                    break;
                }
            }
            for (i = 0; i < 4; i++) {
                if (buffers[i].BufferType == SECBUFFER_EXTRA) {
                    memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - buffers[i].cbBuffer), buffers[i].cbBuffer);
                    state->encrypted_length = buffers[i].cbBuffer;
                    break;
                }
            }
            if (i == 4) {
                state->encrypted_length = 0;
            }
        }
    }
#else
    {
        int read_count = SSL_read(state->ssl, buffer, max_count);
        if (read_count > 0) {
            return read_count;
        }
        {
            int ssl_error = SSL_get_error(state->ssl, read_count);
            if (ssl_error == SSL_ERROR_ZERO_RETURN) {
                return 0;
            }
            if (ssl_error == SSL_ERROR_WANT_READ || ssl_error == SSL_ERROR_WANT_WRITE || (ssl_error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) {
                if (did_timeout != NULL) {
                    *did_timeout = 1;
                }
            }
            return -1;
        }
    }
#endif
}

static int jayess_std_tls_write_bytes(jayess_value *env, const unsigned char *buffer, int length, int *did_timeout) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    int offset = 0;
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (state == NULL || buffer == NULL || length < 0) {
        return -1;
    }
#ifdef _WIN32
    while (offset < length) {
        int chunk_size = length - offset;
        int total_size;
        unsigned char *packet;
        SecBuffer buffers[4];
        SecBufferDesc descriptor;
        SECURITY_STATUS status;
        if (chunk_size > (int)state->stream_sizes.cbMaximumMessage) {
            chunk_size = (int)state->stream_sizes.cbMaximumMessage;
        }
        total_size = (int)(state->stream_sizes.cbHeader + chunk_size + state->stream_sizes.cbTrailer);
        packet = (unsigned char *)malloc((size_t)total_size);
        if (packet == NULL) {
            return -1;
        }
        memcpy(packet + state->stream_sizes.cbHeader, buffer + offset, (size_t)chunk_size);
        buffers[0].pvBuffer = packet;
        buffers[0].cbBuffer = state->stream_sizes.cbHeader;
        buffers[0].BufferType = SECBUFFER_STREAM_HEADER;
        buffers[1].pvBuffer = packet + state->stream_sizes.cbHeader;
        buffers[1].cbBuffer = (unsigned long)chunk_size;
        buffers[1].BufferType = SECBUFFER_DATA;
        buffers[2].pvBuffer = packet + state->stream_sizes.cbHeader + chunk_size;
        buffers[2].cbBuffer = state->stream_sizes.cbTrailer;
        buffers[2].BufferType = SECBUFFER_STREAM_TRAILER;
        buffers[3].pvBuffer = NULL;
        buffers[3].cbBuffer = 0;
        buffers[3].BufferType = SECBUFFER_EMPTY;
        descriptor.ulVersion = SECBUFFER_VERSION;
        descriptor.cBuffers = 4;
        descriptor.pBuffers = buffers;
        status = EncryptMessage(&state->context, 0, &descriptor, 0);
        if (status != SEC_E_OK) {
            free(packet);
            return -1;
        }
        if (!jayess_std_tls_send_all(state->handle, packet, buffers[0].cbBuffer + buffers[1].cbBuffer + buffers[2].cbBuffer)) {
            int error_code = WSAGetLastError();
            if (did_timeout != NULL && error_code == WSAETIMEDOUT) {
                *did_timeout = 1;
            }
            free(packet);
            return -1;
        }
        free(packet);
        offset += chunk_size;
    }
    return length;
#else
    while (offset < length) {
        int written = SSL_write(state->ssl, buffer + offset, length - offset);
        if (written > 0) {
            offset += written;
            continue;
        }
        {
            int ssl_error = SSL_get_error(state->ssl, written);
            if (ssl_error == SSL_ERROR_WANT_READ || ssl_error == SSL_ERROR_WANT_WRITE || (ssl_error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) {
                if (did_timeout != NULL) {
                    *did_timeout = 1;
                }
            }
            return -1;
        }
    }
    return length;
#endif
}

static jayess_value *jayess_std_tls_connect_socket(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    jayess_value *reject_value = object_options != NULL ? jayess_object_get(object_options, "rejectUnauthorized") : NULL;
    jayess_value *timeout_value = object_options != NULL ? jayess_object_get(object_options, "timeout") : NULL;
    jayess_value *alpn_value = object_options != NULL ? jayess_object_get(object_options, "alpnProtocols") : NULL;
    jayess_value *server_name_value = object_options != NULL ? jayess_object_get(object_options, "serverName") : NULL;
    jayess_value *ca_file_value = object_options != NULL ? jayess_object_get(object_options, "caFile") : NULL;
    jayess_value *ca_path_value = object_options != NULL ? jayess_object_get(object_options, "caPath") : NULL;
    jayess_value *trust_system_value = object_options != NULL ? jayess_object_get(object_options, "trustSystem") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    char *server_name_text = NULL;
    char *ca_file_text = NULL;
    char *ca_path_text = NULL;
    int port = (int)jayess_value_to_number(port_value);
    int reject_unauthorized = reject_value == NULL || reject_value->kind == JAYESS_VALUE_UNDEFINED ? 1 : jayess_value_as_bool(reject_value);
    int timeout = (int)jayess_value_to_number(timeout_value);
    int trust_system = trust_system_value == NULL || trust_system_value->kind == JAYESS_VALUE_UNDEFINED ? 1 : jayess_value_as_bool(trust_system_value);
    jayess_value *normalized_alpn = jayess_value_undefined();
    unsigned char *alpn_wire = NULL;
    size_t alpn_wire_length = 0;
    char negotiated_alpn[256];
    const char *negotiated_protocol = "";
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;
    jayess_tls_socket_state *state = NULL;
#ifdef _WIN32
    SCHANNEL_CRED credentials;
    TimeStamp expiry;
    DWORD flags = ISC_REQ_SEQUENCE_DETECT | ISC_REQ_REPLAY_DETECT | ISC_REQ_CONFIDENTIALITY |
        ISC_REQ_EXTENDED_ERROR | ISC_REQ_ALLOCATE_MEMORY | ISC_REQ_STREAM;
    SecBuffer out_buffer;
    SecBufferDesc out_desc;
    SecBuffer in_buffers[2];
    SecBufferDesc in_desc;
    SecBuffer initial_in_buffers[1];
    SecBufferDesc initial_in_desc;
    DWORD context_flags = 0;
    SECURITY_STATUS sec_status;
    int first_call = 1;
    void *alpn_buffer = NULL;
    unsigned long alpn_buffer_length = 0;
#else
    int authorized = 0;
#endif
    negotiated_alpn[0] = '\0';
    if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }
    if (server_name_value != NULL && server_name_value->kind != JAYESS_VALUE_UNDEFINED && server_name_value->kind != JAYESS_VALUE_NULL) {
        server_name_text = jayess_value_stringify(server_name_value);
    } else {
        server_name_text = jayess_strdup(host_text);
    }
    if (ca_file_value != NULL && ca_file_value->kind != JAYESS_VALUE_UNDEFINED && ca_file_value->kind != JAYESS_VALUE_NULL) {
        ca_file_text = jayess_value_stringify(ca_file_value);
    }
    if (ca_path_value != NULL && ca_path_value->kind != JAYESS_VALUE_UNDEFINED && ca_path_value->kind != JAYESS_VALUE_NULL) {
        ca_path_text = jayess_value_stringify(ca_path_value);
    }
    if (server_name_text == NULL || server_name_text[0] == '\0') {
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        jayess_throw(jayess_type_error_value("tls.connect serverName must be a non-empty string"));
        return jayess_value_undefined();
    }
    if (alpn_value != NULL && alpn_value->kind != JAYESS_VALUE_UNDEFINED && alpn_value->kind != JAYESS_VALUE_NULL) {
        normalized_alpn = jayess_std_tls_normalize_alpn_protocols(alpn_value);
        if (jayess_has_exception()) {
            free(host_text);
            return jayess_value_undefined();
        }
    }
    if (normalized_alpn != NULL && normalized_alpn->kind == JAYESS_VALUE_ARRAY && normalized_alpn->as.array_value != NULL && normalized_alpn->as.array_value->count > 0) {
        if (!jayess_std_tls_build_alpn_wire(normalized_alpn, &alpn_wire, &alpn_wire_length)) {
            jayess_throw(jayess_type_error_value("tls.connect failed to encode ALPN protocols"));
            free(host_text);
            return jayess_value_undefined();
        }
    }
    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        jayess_throw(jayess_type_error_value("tls.connect failed to resolve host"));
        free(host_text);
        return jayess_value_undefined();
    }
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
        if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }
    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_throw(jayess_type_error_value("tls.connect failed to connect socket"));
        free(host_text);
        return jayess_value_undefined();
    }
    if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
        jayess_std_socket_close_handle(handle);
        jayess_throw(jayess_type_error_value("tls.connect failed to configure timeout"));
        free(host_text);
        return jayess_value_undefined();
    }
    state = (jayess_tls_socket_state *)calloc(1, sizeof(jayess_tls_socket_state));
    if (state == NULL) {
        jayess_std_socket_close_handle(handle);
        jayess_throw(jayess_type_error_value("tls.connect failed to allocate TLS state"));
        free(host_text);
        return jayess_value_undefined();
    }
    state->handle = handle;
    state->reject_unauthorized = reject_unauthorized;
    state->host = jayess_strdup(server_name_text);
#ifdef _WIN32
    int custom_trust_requested = ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0') || !trust_system);
    if ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0') || !trust_system) {
        if (!reject_unauthorized) {
            /* Custom trust settings are ignored when certificate verification is disabled. */
            custom_trust_requested = 0;
        }
    }
    if (alpn_wire != NULL && alpn_wire_length > 0) {
        alpn_buffer = jayess_std_tls_build_schannel_alpn_buffer(alpn_wire, alpn_wire_length, &alpn_buffer_length);
        if (alpn_buffer == NULL) {
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to prepare ALPN protocols"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
    }
    memset(&credentials, 0, sizeof(credentials));
    credentials.dwVersion = SCHANNEL_CRED_VERSION;
    credentials.dwFlags = SCH_USE_STRONG_CRYPTO | SCH_CRED_NO_DEFAULT_CREDS |
        ((reject_unauthorized && !custom_trust_requested) ? SCH_CRED_AUTO_CRED_VALIDATION : SCH_CRED_MANUAL_CRED_VALIDATION);
    sec_status = AcquireCredentialsHandleA(NULL, UNISP_NAME_A, SECPKG_CRED_OUTBOUND, NULL, &credentials, NULL, NULL, &state->credentials, &expiry);
    if (sec_status != SEC_E_OK) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to acquire TLS credentials"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    state->has_credentials = 1;
    while (1) {
        out_buffer.pvBuffer = NULL;
        out_buffer.cbBuffer = 0;
        out_buffer.BufferType = SECBUFFER_TOKEN;
        out_desc.ulVersion = SECBUFFER_VERSION;
        out_desc.cBuffers = 1;
        out_desc.pBuffers = &out_buffer;
        if (first_call) {
            if (alpn_buffer != NULL) {
                initial_in_buffers[0].pvBuffer = alpn_buffer;
                initial_in_buffers[0].cbBuffer = alpn_buffer_length;
                initial_in_buffers[0].BufferType = SECBUFFER_APPLICATION_PROTOCOLS;
                initial_in_desc.ulVersion = SECBUFFER_VERSION;
                initial_in_desc.cBuffers = 1;
                initial_in_desc.pBuffers = initial_in_buffers;
                sec_status = InitializeSecurityContextA(&state->credentials, NULL, state->host, flags, 0, SECURITY_NATIVE_DREP, &initial_in_desc, 0, &state->context, &out_desc, &context_flags, &expiry);
            } else {
                sec_status = InitializeSecurityContextA(&state->credentials, NULL, state->host, flags, 0, SECURITY_NATIVE_DREP, NULL, 0, &state->context, &out_desc, &context_flags, &expiry);
            }
        } else {
            in_buffers[0].pvBuffer = state->encrypted_buffer;
            in_buffers[0].cbBuffer = (unsigned long)state->encrypted_length;
            in_buffers[0].BufferType = SECBUFFER_TOKEN;
            in_buffers[1].pvBuffer = NULL;
            in_buffers[1].cbBuffer = 0;
            in_buffers[1].BufferType = SECBUFFER_EMPTY;
            in_desc.ulVersion = SECBUFFER_VERSION;
            in_desc.cBuffers = 2;
            in_desc.pBuffers = in_buffers;
            sec_status = InitializeSecurityContextA(&state->credentials, &state->context, state->host, flags, 0, SECURITY_NATIVE_DREP, &in_desc, 0, &state->context, &out_desc, &context_flags, &expiry);
        }
        if (sec_status == SEC_E_OK || sec_status == SEC_I_CONTINUE_NEEDED || sec_status == SEC_I_COMPLETE_NEEDED || sec_status == SEC_I_COMPLETE_AND_CONTINUE || sec_status == SEC_E_INCOMPLETE_MESSAGE) {
            state->has_context = 1;
        }
        if (sec_status == SEC_I_COMPLETE_NEEDED || sec_status == SEC_I_COMPLETE_AND_CONTINUE) {
            if (CompleteAuthToken(&state->context, &out_desc) != SEC_E_OK) {
                if (out_buffer.pvBuffer != NULL) {
                    FreeContextBuffer(out_buffer.pvBuffer);
                }
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to complete TLS handshake token"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        }
        if (out_buffer.pvBuffer != NULL && out_buffer.cbBuffer > 0) {
            int sent_ok = jayess_std_tls_send_all(handle, (const unsigned char *)out_buffer.pvBuffer, out_buffer.cbBuffer);
            FreeContextBuffer(out_buffer.pvBuffer);
            if (!sent_ok) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to send handshake bytes"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        }
        if (sec_status == SEC_E_OK || sec_status == SEC_I_COMPLETE_NEEDED) {
            if (!first_call && in_buffers[1].BufferType == SECBUFFER_EXTRA && in_buffers[1].cbBuffer > 0) {
                memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - in_buffers[1].cbBuffer), in_buffers[1].cbBuffer);
                state->encrypted_length = in_buffers[1].cbBuffer;
            } else {
                state->encrypted_length = 0;
            }
            break;
        }
        if (sec_status != SEC_I_CONTINUE_NEEDED && sec_status != SEC_I_COMPLETE_AND_CONTINUE && sec_status != SEC_E_INCOMPLETE_MESSAGE) {
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect handshake failed"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        if (!first_call) {
            if (in_buffers[1].BufferType == SECBUFFER_EXTRA && in_buffers[1].cbBuffer > 0 && in_buffers[1].cbBuffer < state->encrypted_length) {
                memmove(state->encrypted_buffer, state->encrypted_buffer + (state->encrypted_length - in_buffers[1].cbBuffer), in_buffers[1].cbBuffer);
                state->encrypted_length = in_buffers[1].cbBuffer;
            } else if (sec_status != SEC_E_INCOMPLETE_MESSAGE) {
                state->encrypted_length = 0;
            }
        }
        if (state->encrypted_capacity - state->encrypted_length < 4096) {
            size_t new_capacity = state->encrypted_capacity > 0 ? state->encrypted_capacity * 2 : 32768;
            unsigned char *grown = (unsigned char *)realloc(state->encrypted_buffer, new_capacity);
            if (grown == NULL) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to grow handshake buffer"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
            state->encrypted_buffer = grown;
            state->encrypted_capacity = new_capacity;
        }
        {
            int read_count = (int)recv(handle, (char *)state->encrypted_buffer + state->encrypted_length, (int)(state->encrypted_capacity - state->encrypted_length), 0);
            if (read_count <= 0) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed while reading handshake bytes"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
            state->encrypted_length += (size_t)read_count;
        }
        first_call = 0;
        state->has_context = 1;
    }
    if (QueryContextAttributes(&state->context, SECPKG_ATTR_STREAM_SIZES, &state->stream_sizes) != SEC_E_OK) {
        free(alpn_buffer);
        free(alpn_wire);
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to query TLS stream sizes"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    {
        SecPkgContext_ConnectionInfo connection_info;
        SecPkgContext_ApplicationProtocol application_protocol;
        int authorized = 0;
        if (QueryContextAttributes(&state->context, SECPKG_ATTR_CONNECTION_INFO, &connection_info) == SEC_E_OK) {
            negotiated_protocol = jayess_std_tls_windows_protocol_name(connection_info.dwProtocol);
        }
        if (QueryContextAttributes(&state->context, SECPKG_ATTR_APPLICATION_PROTOCOL, &application_protocol) == SEC_E_OK &&
            application_protocol.ProtoNegoStatus == SecApplicationProtocolNegotiationStatus_Success &&
            application_protocol.ProtocolIdSize > 0) {
            size_t copy_length = application_protocol.ProtocolIdSize;
            if (copy_length >= sizeof(negotiated_alpn)) {
                copy_length = sizeof(negotiated_alpn) - 1;
            }
            memcpy(negotiated_alpn, application_protocol.ProtocolId, copy_length);
            negotiated_alpn[copy_length] = '\0';
        }
        authorized = reject_unauthorized ? (!custom_trust_requested ? 1 : jayess_std_windows_validate_tls_certificate(state, server_name_text, ca_file_text, ca_path_text, trust_system)) : 0;
        if (reject_unauthorized && !authorized) {
            free(alpn_buffer);
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect certificate validation failed"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        if (result == NULL || result->kind != JAYESS_VALUE_OBJECT || result->as.object_value == NULL) {
            free(alpn_buffer);
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to create socket object"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        result->as.object_value->native_handle = state;
        jayess_object_set_value(result->as.object_value, "secure", jayess_value_from_bool(1));
        jayess_object_set_value(result->as.object_value, "authorized", jayess_value_from_bool(authorized));
        jayess_object_set_value(result->as.object_value, "backend", jayess_value_from_string("schannel"));
        jayess_object_set_value(result->as.object_value, "protocol", jayess_value_from_string(negotiated_protocol));
        jayess_object_set_value(result->as.object_value, "alpnProtocol", jayess_value_from_string(negotiated_alpn));
        jayess_object_set_value(result->as.object_value, "alpnProtocols", normalized_alpn != NULL && normalized_alpn->kind != JAYESS_VALUE_UNDEFINED ? normalized_alpn : jayess_value_from_array(jayess_array_new()));
        jayess_object_set_value(result->as.object_value, "timeout", jayess_value_from_number((double)timeout));
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(alpn_buffer);
        free(alpn_wire);
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return result;
    }
#else
    OPENSSL_init_ssl(0, NULL);
    state->ctx = SSL_CTX_new(TLS_client_method());
    if (state->ctx == NULL) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to create TLS context"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    if (reject_unauthorized) {
        SSL_CTX_set_verify(state->ctx, SSL_VERIFY_PEER, NULL);
        if (trust_system) {
            SSL_CTX_set_default_verify_paths(state->ctx);
        }
        if ((ca_file_text != NULL && ca_file_text[0] != '\0') || (ca_path_text != NULL && ca_path_text[0] != '\0')) {
            if (SSL_CTX_load_verify_locations(state->ctx,
                    (ca_file_text != NULL && ca_file_text[0] != '\0') ? ca_file_text : NULL,
                    (ca_path_text != NULL && ca_path_text[0] != '\0') ? ca_path_text : NULL) != 1) {
                jayess_std_tls_state_free(state, 1);
                jayess_throw(jayess_type_error_value("tls.connect failed to load custom trust configuration"));
                free(host_text);
                free(server_name_text);
                free(ca_file_text);
                free(ca_path_text);
                return jayess_value_undefined();
            }
        } else if (!trust_system) {
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect requires caFile or caPath when trustSystem is false"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
    } else {
        SSL_CTX_set_verify(state->ctx, SSL_VERIFY_NONE, NULL);
    }
    state->ssl = SSL_new(state->ctx);
    if (state->ssl == NULL) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to create TLS session"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    SSL_set_fd(state->ssl, handle);
    SSL_set_tlsext_host_name(state->ssl, server_name_text);
    if (alpn_wire != NULL && alpn_wire_length > 0 && SSL_set_alpn_protos(state->ssl, alpn_wire, (unsigned int)alpn_wire_length) != 0) {
        free(alpn_wire);
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect failed to configure ALPN protocols"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    if (reject_unauthorized) {
        X509_VERIFY_PARAM *param = SSL_get0_param(state->ssl);
        if (param != NULL) {
            X509_VERIFY_PARAM_set1_host(param, server_name_text, 0);
        }
    }
    if (SSL_connect(state->ssl) != 1) {
        jayess_std_tls_state_free(state, 1);
        jayess_throw(jayess_type_error_value("tls.connect handshake failed"));
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return jayess_value_undefined();
    }
    negotiated_protocol = SSL_get_version(state->ssl);
    {
        const unsigned char *selected = NULL;
        unsigned int selected_length = 0;
        SSL_get0_alpn_selected(state->ssl, &selected, &selected_length);
        if (selected != NULL && selected_length > 0) {
            size_t copy_length = selected_length;
            if (copy_length >= sizeof(negotiated_alpn)) {
                copy_length = sizeof(negotiated_alpn) - 1;
            }
            memcpy(negotiated_alpn, selected, copy_length);
            negotiated_alpn[copy_length] = '\0';
        }
    }
    authorized = reject_unauthorized ? (SSL_get_verify_result(state->ssl) == X509_V_OK) : 0;
    {
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        if (result == NULL || result->kind != JAYESS_VALUE_OBJECT || result->as.object_value == NULL) {
            free(alpn_wire);
            jayess_std_tls_state_free(state, 1);
            jayess_throw(jayess_type_error_value("tls.connect failed to create socket object"));
            free(host_text);
            free(server_name_text);
            free(ca_file_text);
            free(ca_path_text);
            return jayess_value_undefined();
        }
        result->as.object_value->native_handle = state;
        jayess_object_set_value(result->as.object_value, "secure", jayess_value_from_bool(1));
        jayess_object_set_value(result->as.object_value, "authorized", jayess_value_from_bool(authorized));
        jayess_object_set_value(result->as.object_value, "backend", jayess_value_from_string("openssl"));
        jayess_object_set_value(result->as.object_value, "protocol", jayess_value_from_string(negotiated_protocol != NULL ? negotiated_protocol : ""));
        jayess_object_set_value(result->as.object_value, "alpnProtocol", jayess_value_from_string(negotiated_alpn));
        jayess_object_set_value(result->as.object_value, "alpnProtocols", normalized_alpn != NULL && normalized_alpn->kind != JAYESS_VALUE_UNDEFINED ? normalized_alpn : jayess_value_from_array(jayess_array_new()));
        jayess_object_set_value(result->as.object_value, "timeout", jayess_value_from_number((double)timeout));
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(alpn_wire);
        free(host_text);
        free(server_name_text);
        free(ca_file_text);
        free(ca_path_text);
        return result;
    }
#endif
}

static void jayess_std_socket_mark_closed(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "connected", jayess_value_from_bool(0));
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "readable", jayess_value_from_bool(0));
        jayess_object_set_value(env->as.object_value, "writable", jayess_value_from_bool(0));
    }
}

static void jayess_std_socket_emit_close(jayess_value *env) {
    jayess_value *already_emitted;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    already_emitted = jayess_object_get(env->as.object_value, "__jayess_socket_close_emitted");
    if (jayess_value_as_bool(already_emitted)) {
        return;
    }
    jayess_object_set_value(env->as.object_value, "__jayess_socket_close_emitted", jayess_value_from_bool(1));
    jayess_std_stream_emit(env, "close", jayess_value_undefined());
}

static void jayess_std_socket_close_native(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    if (state != NULL) {
        jayess_std_tls_state_free(state, 0);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            env->as.object_value->native_handle = NULL;
        }
    }
}

static int jayess_std_socket_close_handle(jayess_socket_handle handle) {
    if (handle == JAYESS_INVALID_SOCKET) {
        return 1;
    }
#ifdef _WIN32
    return closesocket(handle) == 0;
#else
    return close(handle) == 0;
#endif
}

static jayess_value *jayess_std_socket_value_from_handle(jayess_socket_handle handle, const char *remote_address, int remote_port) {
    jayess_object *socket_object;
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    socket_object = jayess_object_new();
    if (socket_object == NULL) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_from_object(NULL);
    }
    socket_object->socket_handle = handle;
    jayess_object_set_value(socket_object, "__jayess_std_kind", jayess_value_from_string("Socket"));
    jayess_object_set_value(socket_object, "connected", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "readable", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "writable", jayess_value_from_bool(1));
    jayess_object_set_value(socket_object, "remoteAddress", jayess_value_from_string(remote_address != NULL ? remote_address : ""));
    jayess_object_set_value(socket_object, "remotePort", jayess_value_from_number((double)remote_port));
    jayess_object_set_value(socket_object, "remoteFamily", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "localAddress", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "localPort", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "localFamily", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesRead", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "bytesWritten", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "timeout", jayess_value_from_number(0));
    jayess_object_set_value(socket_object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "secure", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "authorized", jayess_value_from_bool(0));
    jayess_object_set_value(socket_object, "backend", jayess_value_from_string("tcp"));
    jayess_object_set_value(socket_object, "protocol", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "alpnProtocol", jayess_value_from_string(""));
    jayess_object_set_value(socket_object, "alpnProtocols", jayess_value_from_array(jayess_array_new()));
    return jayess_value_from_object(socket_object);
}

static void jayess_std_socket_set_local_endpoint(jayess_value *socket_value, jayess_socket_handle handle) {
    struct sockaddr_storage local_addr;
    char address[INET6_ADDRSTRLEN];
    int port = 0;
    int family = 0;
    void *addr_ptr = NULL;
#ifdef _WIN32
    int local_len = sizeof(local_addr);
#else
    socklen_t local_len = sizeof(local_addr);
#endif
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || socket_value->as.object_value == NULL || handle == JAYESS_INVALID_SOCKET) {
        return;
    }
    memset(&local_addr, 0, sizeof(local_addr));
    address[0] = '\0';
    if (getsockname(handle, (struct sockaddr *)&local_addr, &local_len) != 0) {
        return;
    }
    if (local_addr.ss_family == AF_INET) {
        struct sockaddr_in *ipv4 = (struct sockaddr_in *)&local_addr;
        addr_ptr = &(ipv4->sin_addr);
        port = ntohs(ipv4->sin_port);
        family = 4;
    } else if (local_addr.ss_family == AF_INET6) {
        struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)&local_addr;
        addr_ptr = &(ipv6->sin6_addr);
        port = ntohs(ipv6->sin6_port);
        family = 6;
    }
    if (addr_ptr == NULL || inet_ntop(local_addr.ss_family, addr_ptr, address, sizeof(address)) == NULL) {
        return;
    }
    jayess_object_set_value(socket_value->as.object_value, "localAddress", jayess_value_from_string(address));
    jayess_object_set_value(socket_value->as.object_value, "localPort", jayess_value_from_number((double)port));
    jayess_object_set_value(socket_value->as.object_value, "localFamily", jayess_value_from_number((double)family));
}

static void jayess_std_socket_set_remote_family(jayess_value *socket_value, int family) {
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || socket_value->as.object_value == NULL) {
        return;
    }
    jayess_object_set_value(socket_value->as.object_value, "remoteFamily", jayess_value_from_number((double)family));
}

static jayess_value *jayess_std_tls_normalize_alpn_protocols(jayess_value *value) {
    jayess_array *result;
    int i;
    if (value == NULL || value->kind == JAYESS_VALUE_UNDEFINED || value->kind == JAYESS_VALUE_NULL) {
        return jayess_value_undefined();
    }
    result = jayess_array_new();
    if (result == NULL) {
        return jayess_value_undefined();
    }
    if (value->kind == JAYESS_VALUE_STRING) {
        const char *text = jayess_value_as_string(value);
        if (text == NULL || text[0] == '\0' || strlen(text) > 255) {
            jayess_throw(jayess_type_error_value("tls.connect alpnProtocols entries must be non-empty strings up to 255 bytes"));
            return NULL;
        }
        jayess_array_push_value(result, jayess_value_from_string(text));
        return jayess_value_from_array(result);
    }
    if (value->kind != JAYESS_VALUE_ARRAY || value->as.array_value == NULL) {
        jayess_throw(jayess_type_error_value("tls.connect alpnProtocols must be a string or array of strings"));
        return NULL;
    }
    for (i = 0; i < value->as.array_value->count; i++) {
        char *text = jayess_value_stringify(value->as.array_value->values[i]);
        size_t length = text != NULL ? strlen(text) : 0;
        if (text == NULL || text[0] == '\0' || length > 255) {
            free(text);
            jayess_throw(jayess_type_error_value("tls.connect alpnProtocols entries must be non-empty strings up to 255 bytes"));
            return NULL;
        }
        jayess_array_push_value(result, jayess_value_from_string(text));
        free(text);
    }
    return jayess_value_from_array(result);
}

static int jayess_std_tls_build_alpn_wire(jayess_value *protocols_value, unsigned char **out_buffer, size_t *out_length) {
    size_t total_length = 0;
    int i;
    jayess_array *protocols;
    unsigned char *buffer;
    size_t offset = 0;
    if (out_buffer == NULL || out_length == NULL) {
        return 0;
    }
    *out_buffer = NULL;
    *out_length = 0;
    if (protocols_value == NULL || protocols_value->kind != JAYESS_VALUE_ARRAY || protocols_value->as.array_value == NULL) {
        return 1;
    }
    protocols = protocols_value->as.array_value;
    if (protocols->count == 0) {
        return 1;
    }
    for (i = 0; i < protocols->count; i++) {
        const char *text = jayess_value_as_string(protocols->values[i]);
        size_t length = text != NULL ? strlen(text) : 0;
        if (text == NULL || text[0] == '\0' || length > 255) {
            return 0;
        }
        total_length += 1 + length;
    }
    buffer = (unsigned char *)malloc(total_length);
    if (buffer == NULL) {
        return 0;
    }
    for (i = 0; i < protocols->count; i++) {
        const char *text = jayess_value_as_string(protocols->values[i]);
        size_t length = strlen(text);
        buffer[offset++] = (unsigned char)length;
        memcpy(buffer + offset, text, length);
        offset += length;
    }
    *out_buffer = buffer;
    *out_length = total_length;
    return 1;
}

static void jayess_std_https_copy_tls_request_settings(jayess_object *target, jayess_object *source) {
    static const char *keys[] = {
        "rejectUnauthorized",
        "serverName",
        "caFile",
        "caPath",
        "trustSystem"
    };
    int i;
    if (target == NULL || source == NULL) {
        return;
    }
    for (i = 0; i < (int)(sizeof(keys) / sizeof(keys[0])); i++) {
        jayess_value *value = jayess_object_get(source, keys[i]);
        if (value != NULL) {
            jayess_object_set_value(target, keys[i], value);
        }
    }
}

static void jayess_std_tls_array_push_prefixed(jayess_array *array, const char *prefix, const char *value) {
    size_t prefix_len;
    size_t value_len;
    char *text;
    if (array == NULL || prefix == NULL || value == NULL || value[0] == '\0') {
        return;
    }
    prefix_len = strlen(prefix);
    value_len = strlen(value);
    text = (char *)malloc(prefix_len + value_len + 1);
    if (text == NULL) {
        return;
    }
    memcpy(text, prefix, prefix_len);
    memcpy(text + prefix_len, value, value_len + 1);
    jayess_array_push_value(array, jayess_value_from_string(text));
    free(text);
}

static jayess_value *jayess_std_tls_subject_alt_names(jayess_value *env) {
    jayess_tls_socket_state *state = jayess_std_tls_state(env);
    jayess_array *names = jayess_array_new();
    if (names == NULL) {
        return jayess_value_from_array(NULL);
    }
    if (state == NULL) {
        return jayess_value_from_array(names);
    }
#ifdef _WIN32
    {
        PCCERT_CONTEXT cert = NULL;
        SECURITY_STATUS status = QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert);
        if (status == SEC_E_OK && cert != NULL) {
            PCERT_EXTENSION extension = CertFindExtension(szOID_SUBJECT_ALT_NAME2, cert->pCertInfo->cExtension, cert->pCertInfo->rgExtension);
            if (extension != NULL && extension->Value.pbData != NULL && extension->Value.cbData > 0) {
                DWORD decoded_size = 0;
                if (CryptDecodeObject(X509_ASN_ENCODING | PKCS_7_ASN_ENCODING, X509_ALTERNATE_NAME, extension->Value.pbData, extension->Value.cbData, 0, NULL, &decoded_size) && decoded_size > 0) {
                    CERT_ALT_NAME_INFO *info = (CERT_ALT_NAME_INFO *)malloc(decoded_size);
                    if (info != NULL) {
                        if (CryptDecodeObject(X509_ASN_ENCODING | PKCS_7_ASN_ENCODING, X509_ALTERNATE_NAME, extension->Value.pbData, extension->Value.cbData, 0, info, &decoded_size)) {
                            DWORD i;
                            for (i = 0; i < info->cAltEntry; i++) {
                                CERT_ALT_NAME_ENTRY *entry = &info->rgAltEntry[i];
                                if (entry->dwAltNameChoice == CERT_ALT_NAME_DNS_NAME && entry->pwszDNSName != NULL) {
                                    char *dns = jayess_wide_to_utf8(entry->pwszDNSName);
                                    if (dns != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "DNS:", dns);
                                        free(dns);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_URL && entry->pwszURL != NULL) {
                                    char *uri = jayess_wide_to_utf8(entry->pwszURL);
                                    if (uri != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "URI:", uri);
                                        free(uri);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_RFC822_NAME && entry->pwszRfc822Name != NULL) {
                                    char *email = jayess_wide_to_utf8(entry->pwszRfc822Name);
                                    if (email != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "EMAIL:", email);
                                        free(email);
                                    }
                                } else if (entry->dwAltNameChoice == CERT_ALT_NAME_IP_ADDRESS && entry->IPAddress.pbData != NULL) {
                                    char address[INET6_ADDRSTRLEN];
                                    const void *addr_ptr = NULL;
                                    int family = 0;
                                    address[0] = '\0';
                                    if (entry->IPAddress.cbData == 4) {
                                        family = AF_INET;
                                        addr_ptr = entry->IPAddress.pbData;
                                    } else if (entry->IPAddress.cbData == 16) {
                                        family = AF_INET6;
                                        addr_ptr = entry->IPAddress.pbData;
                                    }
                                    if (addr_ptr != NULL && inet_ntop(family, addr_ptr, address, sizeof(address)) != NULL) {
                                        jayess_std_tls_array_push_prefixed(names, "IP:", address);
                                    }
                                }
                            }
                        }
                        free(info);
                    }
                }
            }
            CertFreeCertificateContext(cert);
        }
    }
#else
    {
        X509 *cert = SSL_get_peer_certificate(state->ssl);
        if (cert != NULL) {
            GENERAL_NAMES *general_names = X509_get_ext_d2i(cert, NID_subject_alt_name, NULL, NULL);
            if (general_names != NULL) {
                int count = sk_GENERAL_NAME_num(general_names);
                int i;
                for (i = 0; i < count; i++) {
                    GENERAL_NAME *name = sk_GENERAL_NAME_value(general_names, i);
                    if (name == NULL) {
                        continue;
                    }
                    if (name->type == GEN_DNS || name->type == GEN_URI || name->type == GEN_EMAIL) {
                        const unsigned char *data = ASN1_STRING_get0_data(name->d.ia5);
                        int length = ASN1_STRING_length(name->d.ia5);
                        char *text;
                        const char *prefix = name->type == GEN_DNS ? "DNS:" : (name->type == GEN_URI ? "URI:" : "EMAIL:");
                        if (data == NULL || length <= 0) {
                            continue;
                        }
                        text = (char *)malloc((size_t)length + 1);
                        if (text == NULL) {
                            continue;
                        }
                        memcpy(text, data, (size_t)length);
                        text[length] = '\0';
                        jayess_std_tls_array_push_prefixed(names, prefix, text);
                        free(text);
                    } else if (name->type == GEN_IPADD) {
                        char address[INET6_ADDRSTRLEN];
                        const unsigned char *data = ASN1_STRING_get0_data(name->d.ip);
                        int length = ASN1_STRING_length(name->d.ip);
                        int family = 0;
                        address[0] = '\0';
                        if (data == NULL) {
                            continue;
                        }
                        if (length == 4) {
                            family = AF_INET;
                        } else if (length == 16) {
                            family = AF_INET6;
                        }
                        if (family != 0 && inet_ntop(family, data, address, sizeof(address)) != NULL) {
                            jayess_std_tls_array_push_prefixed(names, "IP:", address);
                        }
                    }
                }
                GENERAL_NAMES_free(general_names);
            }
            X509_free(cert);
        }
    }
#endif
    return jayess_value_from_array(names);
}

#ifdef _WIN32
static int jayess_std_windows_add_encoded_certificate(HCERTSTORE store, const unsigned char *data, DWORD length) {
    if (store == NULL || data == NULL || length == 0) {
        return 0;
    }
    return CertAddEncodedCertificateToStore(
        store,
        X509_ASN_ENCODING | PKCS_7_ASN_ENCODING,
        data,
        length,
        CERT_STORE_ADD_REPLACE_EXISTING,
        NULL);
}

static int jayess_std_windows_load_certificates_from_file(HCERTSTORE store, const char *path) {
    FILE *file;
    long length;
    char *buffer;
    char *cursor;
    int loaded = 0;
    if (store == NULL || path == NULL || path[0] == '\0') {
        return 0;
    }
    file = fopen(path, "rb");
    if (file == NULL) {
        return 0;
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        return 0;
    }
    length = ftell(file);
    if (length <= 0) {
        fclose(file);
        return 0;
    }
    if (fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return 0;
    }
    buffer = (char *)malloc((size_t)length + 1);
    if (buffer == NULL) {
        fclose(file);
        return 0;
    }
    if (fread(buffer, 1, (size_t)length, file) != (size_t)length) {
        free(buffer);
        fclose(file);
        return 0;
    }
    buffer[length] = '\0';
    fclose(file);

    cursor = buffer;
    while (1) {
        char *begin = strstr(cursor, "-----BEGIN CERTIFICATE-----");
        if (begin == NULL) {
            break;
        }
        char *end = strstr(begin, "-----END CERTIFICATE-----");
        DWORD decoded_length = 0;
        BYTE *decoded = NULL;
        if (end == NULL) {
            break;
        }
        end += (int)strlen("-----END CERTIFICATE-----");
        while (*end == '\r' || *end == '\n') {
            end++;
        }
        if (CryptStringToBinaryA(begin, (DWORD)(end - begin), CRYPT_STRING_BASE64HEADER, NULL, &decoded_length, NULL, NULL) && decoded_length > 0) {
            decoded = (BYTE *)malloc(decoded_length);
            if (decoded != NULL && CryptStringToBinaryA(begin, (DWORD)(end - begin), CRYPT_STRING_BASE64HEADER, decoded, &decoded_length, NULL, NULL)) {
                if (jayess_std_windows_add_encoded_certificate(store, decoded, decoded_length)) {
                    loaded++;
                }
            }
            free(decoded);
        }
        cursor = end;
    }
    if (loaded == 0) {
        if (jayess_std_windows_add_encoded_certificate(store, (const unsigned char *)buffer, (DWORD)length)) {
            loaded = 1;
        }
    }
    free(buffer);
    return loaded > 0;
}

static int jayess_std_windows_load_certificates_from_path(HCERTSTORE store, const char *path) {
    char pattern[MAX_PATH];
    WIN32_FIND_DATAA find_data;
    HANDLE find_handle;
    int loaded = 0;
    if (store == NULL || path == NULL || path[0] == '\0') {
        return 0;
    }
    if (jayess_path_is_separator(path[strlen(path) - 1])) {
        snprintf(pattern, sizeof(pattern), "%s*", path);
    } else {
        snprintf(pattern, sizeof(pattern), "%s\\*", path);
    }
    find_handle = FindFirstFileA(pattern, &find_data);
    if (find_handle == INVALID_HANDLE_VALUE) {
        return 0;
    }
    do {
        char full_path[MAX_PATH];
        if (strcmp(find_data.cFileName, ".") == 0 || strcmp(find_data.cFileName, "..") == 0) {
            continue;
        }
        if ((find_data.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0) {
            continue;
        }
        if (jayess_path_is_separator(path[strlen(path) - 1])) {
            snprintf(full_path, sizeof(full_path), "%s%s", path, find_data.cFileName);
        } else {
            snprintf(full_path, sizeof(full_path), "%s\\%s", path, find_data.cFileName);
        }
        if (jayess_std_windows_load_certificates_from_file(store, full_path)) {
            loaded = 1;
        }
    } while (FindNextFileA(find_handle, &find_data));
    FindClose(find_handle);
    return loaded;
}

static int jayess_std_windows_validate_tls_certificate(jayess_tls_socket_state *state, const char *server_name, const char *ca_file, const char *ca_path, int trust_system) {
    PCCERT_CONTEXT cert = NULL;
    HCERTSTORE custom_store = NULL;
    HCERTSTORE collection_store = NULL;
    HCERTSTORE system_root = NULL;
    HCERTSTORE system_trusted_people = NULL;
    HCERTCHAINENGINE engine = NULL;
    CERT_CHAIN_ENGINE_CONFIG engine_config;
    CERT_CHAIN_PARA chain_para;
    PCCERT_CHAIN_CONTEXT chain = NULL;
    HTTPSPolicyCallbackData policy_data;
    CERT_CHAIN_POLICY_PARA policy_para;
    CERT_CHAIN_POLICY_STATUS policy_status;
    wchar_t *server_name_wide = NULL;
    int ok = 0;
    int has_custom_trust = 0;

    if (state == NULL || server_name == NULL || server_name[0] == '\0') {
        return 0;
    }
    if (QueryContextAttributes(&state->context, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert) != SEC_E_OK || cert == NULL) {
        return 0;
    }
    if ((ca_file != NULL && ca_file[0] != '\0') || (ca_path != NULL && ca_path[0] != '\0') || !trust_system) {
        custom_store = CertOpenStore(CERT_STORE_PROV_MEMORY, 0, 0, CERT_STORE_CREATE_NEW_FLAG, NULL);
        if (custom_store == NULL) {
            goto cleanup;
        }
        if (ca_file != NULL && ca_file[0] != '\0') {
            has_custom_trust = jayess_std_windows_load_certificates_from_file(custom_store, ca_file) || has_custom_trust;
        }
        if (ca_path != NULL && ca_path[0] != '\0') {
            has_custom_trust = jayess_std_windows_load_certificates_from_path(custom_store, ca_path) || has_custom_trust;
        }
        if (!trust_system && !has_custom_trust) {
            goto cleanup;
        }
        collection_store = CertOpenStore(CERT_STORE_PROV_COLLECTION, 0, 0, CERT_STORE_CREATE_NEW_FLAG, NULL);
        if (collection_store == NULL) {
            goto cleanup;
        }
        if (trust_system) {
            system_root = CertOpenSystemStoreA(0, "ROOT");
            system_trusted_people = CertOpenSystemStoreA(0, "TrustedPeople");
            if (system_root != NULL) {
                CertAddStoreToCollection(collection_store, system_root, 0, 0);
            }
            if (system_trusted_people != NULL) {
                CertAddStoreToCollection(collection_store, system_trusted_people, 0, 0);
            }
        }
        if (custom_store != NULL) {
            CertAddStoreToCollection(collection_store, custom_store, 0, 0);
        }
        memset(&engine_config, 0, sizeof(engine_config));
        engine_config.cbSize = sizeof(engine_config);
        engine_config.hExclusiveRoot = collection_store;
#if (NTDDI_VERSION >= NTDDI_WIN8)
        engine_config.dwExclusiveFlags = CERT_CHAIN_EXCLUSIVE_ENABLE_CA_FLAG;
#endif
        if (!CertCreateCertificateChainEngine(&engine_config, &engine)) {
            goto cleanup;
        }
    }

    memset(&chain_para, 0, sizeof(chain_para));
    chain_para.cbSize = sizeof(chain_para);
    if (!CertGetCertificateChain(engine, cert, NULL, cert->hCertStore, &chain_para, 0, NULL, &chain)) {
        goto cleanup;
    }

    server_name_wide = jayess_utf8_to_wide(server_name);
    if (server_name_wide == NULL) {
        goto cleanup;
    }
    memset(&policy_data, 0, sizeof(policy_data));
    policy_data.cbStruct = sizeof(policy_data);
    policy_data.dwAuthType = AUTHTYPE_SERVER;
    policy_data.pwszServerName = server_name_wide;
    memset(&policy_para, 0, sizeof(policy_para));
    policy_para.cbSize = sizeof(policy_para);
    policy_para.pvExtraPolicyPara = &policy_data;
    memset(&policy_status, 0, sizeof(policy_status));
    policy_status.cbSize = sizeof(policy_status);
    if (!CertVerifyCertificateChainPolicy(CERT_CHAIN_POLICY_SSL, chain, &policy_para, &policy_status)) {
        goto cleanup;
    }
    ok = policy_status.dwError == 0;

cleanup:
    free(server_name_wide);
    if (chain != NULL) {
        CertFreeCertificateChain(chain);
    }
    if (engine != NULL) {
        CertFreeCertificateChainEngine(engine);
    }
    if (system_trusted_people != NULL) {
        CertCloseStore(system_trusted_people, 0);
    }
    if (system_root != NULL) {
        CertCloseStore(system_root, 0);
    }
    if (collection_store != NULL) {
        CertCloseStore(collection_store, 0);
    }
    if (custom_store != NULL) {
        CertCloseStore(custom_store, 0);
    }
    if (cert != NULL) {
        CertFreeCertificateContext(cert);
    }
    return ok;
}
#endif

#ifdef _WIN32
static void *jayess_std_tls_build_schannel_alpn_buffer(const unsigned char *wire, size_t wire_length, unsigned long *buffer_length) {
    size_t total_size;
    SEC_APPLICATION_PROTOCOLS *protocols;
    if (buffer_length == NULL) {
        return NULL;
    }
    *buffer_length = 0;
    if (wire == NULL || wire_length == 0) {
        return NULL;
    }
    total_size = FIELD_OFFSET(SEC_APPLICATION_PROTOCOLS, ProtocolLists) +
        FIELD_OFFSET(SEC_APPLICATION_PROTOCOL_LIST, ProtocolList) + wire_length;
    protocols = (SEC_APPLICATION_PROTOCOLS *)calloc(1, total_size);
    if (protocols == NULL) {
        return NULL;
    }
    protocols->ProtocolListsSize = (unsigned long)(FIELD_OFFSET(SEC_APPLICATION_PROTOCOL_LIST, ProtocolList) + wire_length);
    protocols->ProtocolLists[0].ProtoNegoExt = SecApplicationProtocolNegotiationExt_ALPN;
    protocols->ProtocolLists[0].ProtocolListSize = (unsigned short)wire_length;
    memcpy(protocols->ProtocolLists[0].ProtocolList, wire, wire_length);
    *buffer_length = (unsigned long)total_size;
    return protocols;
}

static const char *jayess_std_tls_windows_protocol_name(DWORD protocol) {
#ifdef SP_PROT_TLS1_3_CLIENT
    if (protocol & (SP_PROT_TLS1_3_CLIENT | SP_PROT_TLS1_3_SERVER)) {
        return "TLSv1.3";
    }
#endif
    if (protocol & (SP_PROT_TLS1_2_CLIENT | SP_PROT_TLS1_2_SERVER)) {
        return "TLSv1.2";
    }
    if (protocol & (SP_PROT_TLS1_1_CLIENT | SP_PROT_TLS1_1_SERVER)) {
        return "TLSv1.1";
    }
    if (protocol & (SP_PROT_TLS1_CLIENT | SP_PROT_TLS1_SERVER)) {
        return "TLSv1.0";
    }
#ifdef SP_PROT_SSL3_CLIENT
    if (protocol & (SP_PROT_SSL3_CLIENT | SP_PROT_SSL3_SERVER)) {
        return "SSLv3";
    }
#endif
    return "";
}
#endif

static void jayess_std_stream_set_file(jayess_value *env, FILE *file) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        env->as.object_value->stream_file = file;
    }
}

static int jayess_std_stream_event_key(const char *event, char *buffer, size_t buffer_size) {
    int written;
    if (event == NULL || buffer == NULL || buffer_size == 0) {
        return 0;
    }
    written = snprintf(buffer, buffer_size, "__jayess_stream_event_%s", event);
    return written > 0 && (size_t)written < buffer_size;
}

static int jayess_std_stream_once_key(const char *event, char *buffer, size_t buffer_size) {
    int written;
    if (event == NULL || buffer == NULL || buffer_size == 0) {
        return 0;
    }
    written = snprintf(buffer, buffer_size, "__jayess_stream_once_%s", event);
    return written > 0 && (size_t)written < buffer_size;
}

static int jayess_string_has_prefix(const char *value, const char *prefix) {
    size_t prefix_len;
    if (value == NULL || prefix == NULL) {
        return 0;
    }
    prefix_len = strlen(prefix);
    return strncmp(value, prefix, prefix_len) == 0;
}

static void jayess_std_stream_add_listener(jayess_value *env, const char *event, jayess_value *callback, int once) {
    char key[128];
    jayess_value *stored;
    jayess_array *listeners;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    if (!(once ? jayess_std_stream_once_key(event, key, sizeof(key)) : jayess_std_stream_event_key(event, key, sizeof(key)))) {
        return;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        listeners = jayess_array_new();
        if (listeners == NULL) {
            return;
        }
        jayess_object_set_value(env->as.object_value, key, jayess_value_from_array(listeners));
    } else {
        listeners = stored->as.array_value;
    }
    jayess_array_push_value(listeners, callback);
}

static void jayess_std_stream_on(jayess_value *env, const char *event, jayess_value *callback) {
    jayess_std_stream_add_listener(env, event, callback, 0);
}

static void jayess_std_stream_once(jayess_value *env, const char *event, jayess_value *callback) {
    jayess_std_stream_add_listener(env, event, callback, 1);
}

static void jayess_std_stream_off_key(jayess_value *env, const char *key, jayess_value *callback) {
    jayess_value *stored;
    jayess_array *listeners;
    int i;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || key == NULL) {
        return;
    }
    if (callback == NULL || jayess_value_is_nullish(callback)) {
        jayess_object_delete(env->as.object_value, key);
        return;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        return;
    }
    listeners = stored->as.array_value;
    for (i = 0; i < listeners->count; i++) {
        if (listeners->values[i] == callback || jayess_value_eq(listeners->values[i], callback)) {
            jayess_array_remove_at(listeners, i);
            break;
        }
    }
    if (listeners->count == 0) {
        jayess_object_delete(env->as.object_value, key);
    }
}

static void jayess_std_stream_off(jayess_value *env, const char *event, jayess_value *callback) {
    char key[128];
    if (jayess_std_stream_event_key(event, key, sizeof(key))) {
        jayess_std_stream_off_key(env, key, callback);
    }
    if (jayess_std_stream_once_key(event, key, sizeof(key))) {
        jayess_std_stream_off_key(env, key, callback);
    }
}

static jayess_array *jayess_std_stream_listeners_for_key(jayess_value *env, const char *key) {
    jayess_value *stored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || key == NULL) {
        return NULL;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        return NULL;
    }
    return stored->as.array_value;
}

static jayess_array *jayess_std_stream_listeners(jayess_value *env, const char *event, int once) {
    char key[128];
    if (!(once ? jayess_std_stream_once_key(event, key, sizeof(key)) : jayess_std_stream_event_key(event, key, sizeof(key)))) {
        return NULL;
    }
    return jayess_std_stream_listeners_for_key(env, key);
}

static int jayess_std_stream_listener_count(jayess_value *env, const char *event) {
    jayess_array *listeners = jayess_std_stream_listeners(env, event, 0);
    jayess_array *once_listeners = jayess_std_stream_listeners(env, event, 1);
    int count = 0;
    if (listeners != NULL) {
        count += listeners->count;
    }
    if (once_listeners != NULL) {
        count += once_listeners->count;
    }
    return count;
}

static int jayess_std_stream_event_names_has(jayess_array *names, const char *event) {
    int i;
    if (names == NULL || event == NULL) {
        return 0;
    }
    for (i = 0; i < names->count; i++) {
        jayess_value *name = jayess_array_get(names, i);
        if (name != NULL && name->kind == JAYESS_VALUE_STRING && name->as.string_value != NULL && strcmp(name->as.string_value, event) == 0) {
            return 1;
        }
    }
    return 0;
}

static jayess_value *jayess_std_stream_listener_count_method(jayess_value *env, jayess_value *event) {
    char *event_text = jayess_value_stringify(event);
    int count;
    if (event_text == NULL) {
        return jayess_value_from_number(0);
    }
    count = jayess_std_stream_listener_count(env, event_text);
    free(event_text);
    return jayess_value_from_number((double)count);
}

static jayess_value *jayess_std_stream_event_names_method(jayess_value *env) {
    const char *event_prefix = "__jayess_stream_event_";
    const char *once_prefix = "__jayess_stream_once_";
    size_t event_prefix_len = strlen(event_prefix);
    size_t once_prefix_len = strlen(once_prefix);
    jayess_array *names = jayess_array_new();
    jayess_object_entry *current;
    if (names == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_from_array(names);
    }
    current = env->as.object_value->head;
    while (current != NULL) {
        const char *event_name = NULL;
        if (jayess_string_has_prefix(current->key, event_prefix)) {
            event_name = current->key + event_prefix_len;
        } else if (jayess_string_has_prefix(current->key, once_prefix)) {
            event_name = current->key + once_prefix_len;
        }
        if (event_name != NULL && event_name[0] != '\0' && !jayess_std_stream_event_names_has(names, event_name)) {
            jayess_array_push_value(names, jayess_value_from_string(event_name));
        }
        current = current->next;
    }
    return jayess_value_from_array(names);
}

static void jayess_std_stream_emit(jayess_value *env, const char *event, jayess_value *argument) {
    jayess_array *listeners = jayess_std_stream_listeners(env, event, 0);
    jayess_array *once_listeners;
    char once_key[128];
    int i;
    if (listeners != NULL) {
        for (i = 0; i < listeners->count; i++) {
            jayess_value *callback = listeners->values[i];
            if (callback != NULL && callback->kind == JAYESS_VALUE_FUNCTION) {
                jayess_value_call_one(callback, argument != NULL ? argument : jayess_value_undefined());
                if (jayess_has_exception()) {
                    return;
                }
            }
        }
    }
    if (!jayess_std_stream_once_key(event, once_key, sizeof(once_key))) {
        return;
    }
    once_listeners = jayess_std_stream_listeners_for_key(env, once_key);
    if (once_listeners == NULL) {
        return;
    }
    jayess_object_delete(env->as.object_value, once_key);
    for (i = 0; i < once_listeners->count; i++) {
        jayess_value *callback = once_listeners->values[i];
        if (callback != NULL && callback->kind == JAYESS_VALUE_FUNCTION) {
            jayess_value_call_one(callback, argument != NULL ? argument : jayess_value_undefined());
            if (jayess_has_exception()) {
                return;
            }
        }
    }
}

static void jayess_std_stream_emit_error(jayess_value *env, const char *message) {
    jayess_value *error;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    error = jayess_std_error_new(jayess_value_from_string("Error"), jayess_value_from_string(message != NULL ? message : "stream error"));
    jayess_object_set_value(env->as.object_value, "errored", jayess_value_from_bool(1));
    jayess_object_set_value(env->as.object_value, "error", error);
    jayess_std_stream_emit(env, "error", error);
}

static void jayess_std_stream_register_error_handler(jayess_value *env, jayess_value *callback) {
    jayess_value *errored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    jayess_std_stream_on(env, "error", callback);
    errored = jayess_object_get(env->as.object_value, "errored");
    if (jayess_value_as_bool(errored)) {
        jayess_value_call_one(callback, jayess_object_get(env->as.object_value, "error"));
    }
}

static void jayess_std_stream_register_error_once(jayess_value *env, jayess_value *callback) {
    jayess_value *errored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    errored = jayess_object_get(env->as.object_value, "errored");
    if (jayess_value_as_bool(errored)) {
        jayess_value_call_one(callback, jayess_object_get(env->as.object_value, "error"));
        return;
    }
    jayess_std_stream_once(env, "error", callback);
}

static void jayess_std_write_stream_register_finish_handler(jayess_value *env, jayess_value *callback) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    jayess_std_stream_on(env, "finish", callback);
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded")) && !jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        jayess_value_call_one(callback, jayess_value_undefined());
    }
}

static void jayess_std_write_stream_register_finish_once(jayess_value *env, jayess_value *callback) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded")) && !jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        jayess_value_call_one(callback, jayess_value_undefined());
        return;
    }
    jayess_std_stream_once(env, "finish", callback);
}

static void jayess_std_write_stream_emit_finish(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        return;
    }
    jayess_std_stream_emit(env, "finish", jayess_value_undefined());
}

static jayess_value *jayess_std_fs_stream_open_error(const char *kind, const char *message) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(kind != NULL ? kind : "Stream"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(1));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(1));
    jayess_object_set_value(object, "error", jayess_std_error_new(jayess_value_from_string("Error"), jayess_value_from_string(message != NULL ? message : "failed to open stream")));
    if (kind != NULL && strcmp(kind, "ReadStream") == 0) {
        jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(1));
    }
    if (kind != NULL && strcmp(kind, "WriteStream") == 0) {
        jayess_object_set_value(object, "writableEnded", jayess_value_from_bool(1));
    }
    return jayess_value_from_object(object);
}

static void jayess_std_read_stream_mark_ended(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "readableEnded", jayess_value_from_bool(1));
    }
}

static void jayess_std_read_stream_emit_end(jayess_value *env) {
    jayess_std_read_stream_mark_ended(env);
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    jayess_std_stream_emit(env, "end", jayess_value_undefined());
}

static int jayess_std_stream_requested_size(jayess_value *size_value, int default_size) {
    int requested = default_size;
    if (size_value != NULL && !jayess_value_is_nullish(size_value)) {
        requested = (int)jayess_value_to_number(size_value);
        if (requested <= 0) {
            requested = 1;
        }
        if (requested > 1048576) {
            requested = 1048576;
        }
    }
    return requested;
}

static jayess_value *jayess_std_read_stream_read_chunk(jayess_value *env, jayess_value *size_value) {
    FILE *file = jayess_std_stream_file(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    char *buffer;
    size_t read_count;
    jayess_value *result;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
        return jayess_value_undefined();
    }
    buffer = (char *)malloc((size_t)requested + 1);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate stream read buffer");
        return jayess_value_undefined();
    }
    read_count = fread(buffer, 1, (size_t)requested, file);
    if (read_count == 0) {
        free(buffer);
        if (feof(file)) {
            jayess_std_read_stream_emit_end(env);
            return jayess_value_null();
        }
        jayess_std_stream_emit_error(env, "failed to read from stream");
        return jayess_value_undefined();
    }
    buffer[read_count] = '\0';
    result = jayess_value_from_string(buffer);
    free(buffer);
    return result;
}

static jayess_value *jayess_std_read_stream_read_method(jayess_value *env, jayess_value *size_value) {
    return jayess_std_read_stream_read_chunk(env, size_value);
}

static jayess_value *jayess_std_read_stream_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    FILE *file = jayess_std_stream_file(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer;
    size_t read_count;
    jayess_value *array_buffer;
    jayess_value *view;
    jayess_array *bytes;
    int i;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
        return jayess_value_undefined();
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate stream read buffer");
        return jayess_value_undefined();
    }
    read_count = fread(buffer, 1, (size_t)requested, file);
    if (read_count == 0) {
        free(buffer);
        if (feof(file)) {
            jayess_std_read_stream_emit_end(env);
            return jayess_value_null();
        }
        jayess_std_stream_emit_error(env, "failed to read from stream");
        return jayess_value_undefined();
    }
    array_buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)read_count));
    view = jayess_std_uint8_array_new(array_buffer);
    bytes = jayess_std_bytes_slot(view);
    if (bytes == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    for (i = 0; i < (int)read_count; i++) {
        jayess_array_set_value(bytes, i, jayess_value_from_number((double)buffer[i]));
    }
    free(buffer);
    return view;
}

static jayess_value *jayess_std_read_stream_close_method(jayess_value *env) {
    FILE *file = jayess_std_stream_file(env);
    if (file != NULL) {
        fclose(file);
        jayess_std_stream_set_file(env, NULL);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_std_read_stream_mark_ended(env);
        }
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_read_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        jayess_std_stream_on(env, "end", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        while (1) {
            jayess_value *chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
            if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
                break;
            }
            jayess_value_call_one(callback, chunk);
            if (jayess_has_exception()) {
                break;
            }
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_read_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "end", callback);
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        jayess_value *chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
        if (chunk != NULL && chunk->kind != JAYESS_VALUE_NULL && chunk->kind != JAYESS_VALUE_UNDEFINED) {
            jayess_value_call_one(callback, chunk);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_stream_off_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    jayess_std_stream_off(env, event_text, callback);
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_read_stream_pipe_method(jayess_value *env, jayess_value *destination) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(destination, "WriteStream")) {
        return destination != NULL ? destination : jayess_value_undefined();
    }
    while (1) {
        jayess_value *chunk = jayess_std_read_stream_read_chunk(env, jayess_value_undefined());
        if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
            break;
        }
        jayess_std_write_stream_write_method(destination, chunk);
        if (jayess_has_exception()) {
            break;
        }
    }
    jayess_std_write_stream_end_method(destination);
    return destination;
}

static jayess_value *jayess_std_http_body_stream_read_method(jayess_value *env, jayess_value *size_value) {
    return jayess_http_body_stream_read_chunk(env, size_value, 0);
}

static jayess_value *jayess_std_http_body_stream_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    return jayess_http_body_stream_read_chunk(env, size_value, 1);
}

static jayess_value *jayess_std_http_body_stream_close_method(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_http_body_stream_mark_ended(env);
        jayess_http_body_stream_close_socket(env);
        jayess_http_body_stream_close_native(env);
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_http_body_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        jayess_std_stream_on(env, "end", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        while (1) {
            jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 0);
            if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
                break;
            }
            jayess_value_call_one(callback, chunk);
            if (jayess_has_exception()) {
                break;
            }
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_http_body_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "end") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "end", callback);
        }
    } else if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "data") == 0) {
        jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 0);
        if (chunk != NULL && chunk->kind != JAYESS_VALUE_NULL && chunk->kind != JAYESS_VALUE_UNDEFINED) {
            jayess_value_call_one(callback, chunk);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_http_body_stream_pipe_method(jayess_value *env, jayess_value *destination) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(destination, "WriteStream")) {
        return destination != NULL ? destination : jayess_value_undefined();
    }
    while (1) {
        jayess_value *chunk = jayess_http_body_stream_read_chunk(env, jayess_value_undefined(), 1);
        if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
            break;
        }
        jayess_std_write_stream_write_method(destination, chunk);
        if (jayess_has_exception()) {
            break;
        }
    }
    jayess_std_write_stream_end_method(destination);
    return destination;
}

static jayess_value *jayess_std_write_stream_write_method(jayess_value *env, jayess_value *value) {
    FILE *file = jayess_std_stream_file(env);
    char *text;
    size_t length;
    int ok;
    jayess_array *bytes;
    int i;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        }
        return jayess_value_from_bool(0);
    }
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Uint8Array")) {
        bytes = jayess_std_bytes_slot(value);
        if (bytes == NULL) {
            return jayess_value_from_bool(0);
        }
        for (i = 0; i < bytes->count; i++) {
            unsigned char byte_value = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
            if (fwrite(&byte_value, 1, 1, file) != 1) {
                jayess_std_stream_emit_error(env, "failed to write to stream");
                return jayess_value_from_bool(0);
            }
        }
        return jayess_value_from_bool(1);
    }
    text = jayess_value_stringify(value);
    if (text == NULL) {
        return jayess_value_from_bool(0);
    }
    length = strlen(text);
    ok = fwrite(text, 1, length, file) == length;
    free(text);
    if (!ok) {
        jayess_std_stream_emit_error(env, "failed to write to stream");
    }
    return jayess_value_from_bool(ok);
}

static jayess_value *jayess_std_write_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "finish") == 0) {
        jayess_std_write_stream_register_finish_handler(env, callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_write_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "finish") == 0) {
        jayess_std_write_stream_register_finish_once(env, callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_write_stream_end_method(jayess_value *env) {
    FILE *file = jayess_std_stream_file(env);
    if (file != NULL) {
        if (fflush(file) != 0) {
            jayess_std_stream_emit_error(env, "failed to flush stream");
        }
        fclose(file);
        jayess_std_stream_set_file(env, NULL);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "writableEnded", jayess_value_from_bool(1));
            jayess_std_write_stream_emit_finish(env);
        }
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_socket_read_method(jayess_value *env, jayess_value *size_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    char *buffer;
    int read_count;
    int did_timeout = 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_undefined();
    }
    buffer = (char *)malloc((size_t)requested + 1);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    #ifdef _WIN32
    if (jayess_std_tls_state(env) != NULL) {
        read_count = jayess_std_tls_read_bytes(env, (unsigned char *)buffer, requested, &did_timeout);
    } else {
        read_count = (int)recv(handle, buffer, requested, 0);
    }
    #else
    read_count = (int)recv(handle, buffer, requested, 0);
    #endif
    if (read_count <= 0) {
        free(buffer);
        if (read_count < 0) {
            jayess_std_stream_emit_error(env, did_timeout ? "socket read timed out" : "failed to read from socket");
        }
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
        jayess_std_socket_close_native(env);
        jayess_std_socket_mark_closed(env);
        jayess_std_socket_emit_close(env);
        if (read_count == 0) {
            return jayess_value_null();
        }
        return jayess_value_undefined();
    }
    buffer[read_count] = '\0';
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)read_count;
        jayess_object_set_value(env->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    {
        jayess_value *result = jayess_value_from_string(buffer);
        free(buffer);
        return result;
    }
}

static jayess_value *jayess_std_socket_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer;
    int read_count;
    int did_timeout = 0;
    jayess_value *array_buffer;
    jayess_value *view;
    jayess_array *bytes;
    int i;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_undefined();
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    #ifdef _WIN32
    if (jayess_std_tls_state(env) != NULL) {
        read_count = jayess_std_tls_read_bytes(env, buffer, requested, &did_timeout);
    } else {
        read_count = (int)recv(handle, (char *)buffer, requested, 0);
    }
    #else
    read_count = (int)recv(handle, (char *)buffer, requested, 0);
    #endif
    if (read_count <= 0) {
        free(buffer);
        if (read_count < 0) {
            jayess_std_stream_emit_error(env, did_timeout ? "socket read timed out" : "failed to read from socket");
        }
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
        jayess_std_socket_close_native(env);
        jayess_std_socket_mark_closed(env);
        jayess_std_socket_emit_close(env);
        if (read_count == 0) {
            return jayess_value_null();
        }
        return jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_value *current = jayess_object_get(env->as.object_value, "bytesRead");
        double total = jayess_value_to_number(current) + (double)read_count;
        jayess_object_set_value(env->as.object_value, "bytesRead", jayess_value_from_number(total));
    }
    array_buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)read_count));
    view = jayess_std_uint8_array_new(array_buffer);
    bytes = jayess_std_bytes_slot(view);
    if (bytes == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    for (i = 0; i < read_count; i++) {
        bytes->values[i] = jayess_value_from_number((double)buffer[i]);
    }
    free(buffer);
    return view;
}

static jayess_value *jayess_std_socket_write_method(jayess_value *env, jayess_value *value) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int did_timeout = 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return jayess_value_from_bool(0);
    }
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(value);
        int offset = 0;
        if (bytes == NULL) {
            return jayess_value_from_bool(0);
        }
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
            if (jayess_std_tls_state(env) != NULL) {
                sent = jayess_std_tls_write_bytes(env, chunk, chunk_len, &did_timeout);
            } else {
                sent = (int)send(handle, (const char *)chunk, chunk_len, 0);
            }
            #else
            sent = (int)send(handle, (const char *)chunk, chunk_len, 0);
            #endif
            if (sent <= 0) {
                jayess_std_stream_emit_error(env, did_timeout ? "socket write timed out" : "failed to write to socket");
                return jayess_value_from_bool(0);
            }
            if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
                jayess_value *current = jayess_object_get(env->as.object_value, "bytesWritten");
                double total = jayess_value_to_number(current) + (double)sent;
                jayess_object_set_value(env->as.object_value, "bytesWritten", jayess_value_from_number(total));
            }
            offset += sent;
        }
        return jayess_value_from_bool(1);
    }
    {
        char *text = jayess_value_stringify(value);
        size_t length;
        size_t offset = 0;
        if (text == NULL) {
            return jayess_value_from_bool(0);
        }
        length = strlen(text);
        while (offset < length) {
            int sent;
            #ifdef _WIN32
            if (jayess_std_tls_state(env) != NULL) {
                sent = jayess_std_tls_write_bytes(env, (const unsigned char *)text + offset, (int)(length - offset), &did_timeout);
            } else {
                sent = (int)send(handle, text + offset, (int)(length - offset), 0);
            }
            #else
            sent = (int)send(handle, text + offset, (int)(length - offset), 0);
            #endif
            if (sent <= 0) {
                jayess_std_stream_emit_error(env, did_timeout ? "socket write timed out" : "failed to write to socket");
                free(text);
                return jayess_value_from_bool(0);
            }
            if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
                jayess_value *current = jayess_object_get(env->as.object_value, "bytesWritten");
                double total = jayess_value_to_number(current) + (double)sent;
                jayess_object_set_value(env->as.object_value, "bytesWritten", jayess_value_from_number(total));
            }
            offset += (size_t)sent;
        }
        free(text);
        return jayess_value_from_bool(1);
    }
}

static jayess_value *jayess_std_socket_read_async_method(jayess_value *env, jayess_value *size_value) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_socket_read_task(promise, env, size_value);
    return promise;
}

static jayess_value *jayess_std_socket_write_async_method(jayess_value *env, jayess_value *value) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_socket_write_task(promise, env, value);
    return promise;
}

static jayess_value *jayess_std_socket_set_no_delay_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, IPPROTO_TCP, TCP_NODELAY, (const char *)&flag, sizeof(flag)) != 0) {
#else
    if (setsockopt(handle, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure TCP_NODELAY");
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_set_keep_alive_method(jayess_value *env, jayess_value *enabled) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int flag = jayess_value_as_bool(enabled) ? 1 : 0;
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    if (setsockopt(handle, SOL_SOCKET, SO_KEEPALIVE, (const char *)&flag, sizeof(flag)) != 0) {
#else
    if (setsockopt(handle, SOL_SOCKET, SO_KEEPALIVE, &flag, sizeof(flag)) != 0) {
#endif
        jayess_std_stream_emit_error(env, "failed to configure SO_KEEPALIVE");
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_set_timeout_method(jayess_value *env, jayess_value *timeout_ms) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int timeout = (int)jayess_value_to_number(timeout_ms);
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        jayess_std_socket_mark_closed(env);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (!jayess_std_socket_configure_timeout(handle, timeout)) {
        jayess_std_stream_emit_error(env, "failed to configure socket timeout");
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "timeout", jayess_value_from_number((double)timeout));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_close_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
#ifdef _WIN32
        shutdown(handle, SD_BOTH);
#else
        shutdown(handle, SHUT_RDWR);
#endif
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
    jayess_std_socket_close_native(env);
    jayess_std_socket_mark_closed(env);
    jayess_std_socket_emit_close(env);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_socket_address_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "localAddress"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "localPort"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "localFamily"));
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_socket_remote_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "remoteAddress"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "remotePort"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "remoteFamily"));
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_socket_get_peer_certificate_method(jayess_value *env) {
    if (jayess_std_tls_state(env) == NULL) {
        return jayess_value_undefined();
    }
    return jayess_std_tls_peer_certificate(env);
}

static jayess_value *jayess_std_socket_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "connect") == 0) {
        jayess_std_stream_on(env, "connect", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "connected"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "close") == 0) {
        jayess_std_stream_on(env, "close", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_socket_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "connect") == 0) {
        jayess_std_stream_once(env, "connect", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "connected"))) {
            jayess_std_stream_off(env, "connect", callback);
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "close") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "close", callback);
        }
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        jayess_std_stream_on(env, "close", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "listening") == 0) {
        jayess_std_stream_on(env, "listening", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "listening"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "connection") == 0) {
        jayess_std_stream_on(env, "connection", callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "close") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "closed"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "close", callback);
        }
    } else if (strcmp(event_text, "listening") == 0) {
        jayess_std_stream_once(env, "listening", callback);
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "listening"))) {
            jayess_std_stream_off(env, "listening", callback);
            jayess_value_call_one(callback, jayess_value_undefined());
        }
    } else if (strcmp(event_text, "connection") == 0) {
        jayess_std_stream_once(env, "connection", callback);
    }
    free(event_text);
    return env;
}

static jayess_value *jayess_std_server_accept_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
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
    if (handle == JAYESS_INVALID_SOCKET) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
        }
        return jayess_value_undefined();
    }
    memset(&client_addr, 0, sizeof(client_addr));
    client_handle = accept(handle, (struct sockaddr *)&client_addr, &client_len);
    if (client_handle == JAYESS_INVALID_SOCKET) {
        jayess_std_stream_emit_error(env, "failed to accept socket connection");
        return jayess_value_undefined();
    }
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
        return jayess_value_undefined();
    }
    {
        jayess_value *result = jayess_std_socket_value_from_handle(client_handle, address, port);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_value *current = jayess_object_get(env->as.object_value, "connectionsAccepted");
            double total = jayess_value_to_number(current) + 1.0;
            jayess_object_set_value(env->as.object_value, "connectionsAccepted", jayess_value_from_number(total));
        }
        jayess_std_socket_set_remote_family(result, client_addr.ss_family == AF_INET6 ? 6 : 4);
        jayess_std_socket_set_local_endpoint(result, client_handle);
        jayess_std_stream_emit(env, "connection", result);
        return result;
    }
}

static jayess_value *jayess_std_server_accept_async_method(jayess_value *env) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_server_accept_task(promise, env);
    return promise;
}

static jayess_value *jayess_std_server_close_method(jayess_value *env) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
#ifdef _WIN32
        shutdown(handle, SD_BOTH);
#else
        shutdown(handle, SHUT_RDWR);
#endif
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
    }
    jayess_std_socket_emit_close(env);
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_server_set_timeout_method(jayess_value *env, jayess_value *timeout_ms) {
    jayess_socket_handle handle = jayess_std_socket_handle(env);
    int timeout = (int)jayess_value_to_number(timeout_ms);
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "listening", jayess_value_from_bool(0));
        }
        return env != NULL ? env : jayess_value_undefined();
    }
#ifdef _WIN32
    {
        DWORD timeout_value = (DWORD)timeout;
        if (setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) != 0) {
            jayess_std_stream_emit_error(env, "failed to configure server timeout");
            return env != NULL ? env : jayess_value_undefined();
        }
    }
#else
    {
        struct timeval timeout_value;
        timeout_value.tv_sec = timeout / 1000;
        timeout_value.tv_usec = (timeout % 1000) * 1000;
        if (setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, &timeout_value, sizeof(timeout_value)) != 0) {
            jayess_std_stream_emit_error(env, "failed to configure server timeout");
            return env != NULL ? env : jayess_value_undefined();
        }
    }
#endif
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "timeout", jayess_value_from_number((double)timeout));
    }
    return env != NULL ? env : jayess_value_undefined();
}

static jayess_value *jayess_std_server_address_method(jayess_value *env) {
    jayess_object *result;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    if (result == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(result, "address", jayess_object_get(env->as.object_value, "host"));
    jayess_object_set_value(result, "port", jayess_object_get(env->as.object_value, "port"));
    jayess_object_set_value(result, "family", jayess_object_get(env->as.object_value, "family"));
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_process_exit(jayess_value *code) {
    int exit_code = (int)jayess_value_to_number(code);
    jayess_runtime_shutdown();
    exit(exit_code);
    return jayess_value_undefined();
}

jayess_value *jayess_std_process_argv(void) {
    if (jayess_current_args == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    return jayess_value_from_args(jayess_current_args);
}

jayess_value *jayess_std_process_platform(void) {
#ifdef _WIN32
    return jayess_value_from_string("windows");
#elif __APPLE__
    return jayess_value_from_string("darwin");
#else
    return jayess_value_from_string("linux");
#endif
}

jayess_value *jayess_std_process_arch(void) {
#if defined(__aarch64__) || defined(_M_ARM64)
    return jayess_value_from_string("arm64");
#elif defined(__x86_64__) || defined(_M_X64)
    return jayess_value_from_string("x64");
#elif defined(__i386__) || defined(_M_IX86)
    return jayess_value_from_string("x86");
#else
    return jayess_value_from_string("unknown");
#endif
}

jayess_value *jayess_std_process_thread_pool_size(void) {
    return jayess_value_from_number((double)JAYESS_IO_WORKER_COUNT);
}

jayess_value *jayess_std_tls_is_available(void) {
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_tls_backend(void) {
#ifdef _WIN32
    return jayess_value_from_string("schannel");
#else
    return jayess_value_from_string("openssl");
#endif
}

jayess_value *jayess_std_tls_connect(jayess_value *options) {
    return jayess_std_tls_connect_socket(options);
}

jayess_value *jayess_std_https_is_available(void) {
    return jayess_std_tls_is_available();
}

jayess_value *jayess_std_https_backend(void) {
    return jayess_std_tls_backend();
}

static char *jayess_shell_quote(const char *value) {
    size_t len;
    size_t out_len = 2;
    size_t i;
    size_t j = 0;
    char *out;
    if (value == NULL) {
        value = "";
    }
    len = strlen(value);
    for (i = 0; i < len; i++) {
        out_len += (value[i] == '"' || value[i] == '\\') ? 2 : 1;
    }
    out = (char *)malloc(out_len + 1);
    if (out == NULL) {
        return NULL;
    }
    out[j++] = '"';
    for (i = 0; i < len; i++) {
        if (value[i] == '"' || value[i] == '\\') {
            out[j++] = '\\';
        }
        out[j++] = value[i];
    }
    out[j++] = '"';
    out[j] = '\0';
    return out;
}

static char *jayess_compile_flag(const char *name, const char *value) {
    size_t len;
    char *out;
    if (name == NULL || value == NULL || value[0] == '\0') {
        return NULL;
    }
    len = strlen(name) + strlen(value) + 1;
    out = (char *)malloc(len + 1);
    if (out == NULL) {
        return NULL;
    }
    sprintf(out, "%s=%s", name, value);
    return out;
}

#ifdef _WIN32
static int jayess_spawn_compiler(const char *compiler, const char *emit_arg, const char *target_arg, const char *warnings_arg, const char *output_path, const char *source_path, const char *stdout_path, const char *stderr_path) {
    char *quoted_compiler = jayess_shell_quote(compiler);
    char *quoted_output = jayess_shell_quote(output_path);
    char *quoted_source = jayess_shell_quote(source_path);
    char *command;
    STARTUPINFOA startup;
    PROCESS_INFORMATION process;
    SECURITY_ATTRIBUTES security;
    HANDLE stdout_handle;
    HANDLE stderr_handle;
    DWORD exit_code = 1;
    if (quoted_compiler == NULL || quoted_output == NULL || quoted_source == NULL) {
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        return -1;
    }
    command = (char *)malloc(strlen(quoted_compiler) + strlen(emit_arg) + (target_arg != NULL ? strlen(target_arg) + 1 : 0) + (warnings_arg != NULL ? strlen(warnings_arg) + 1 : 0) + strlen(quoted_output) + strlen(quoted_source) + 16);
    if (command == NULL) {
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        return -1;
    }
    sprintf(command, "%s %s%s%s%s%s -o %s %s",
            quoted_compiler,
            emit_arg,
            target_arg != NULL ? " " : "",
            target_arg != NULL ? target_arg : "",
            warnings_arg != NULL ? " " : "",
            warnings_arg != NULL ? warnings_arg : "",
            quoted_output,
            quoted_source);
    security.nLength = sizeof(security);
    security.lpSecurityDescriptor = NULL;
    security.bInheritHandle = TRUE;
    stdout_handle = CreateFileA(stdout_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    stderr_handle = CreateFileA(stderr_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (stdout_handle == INVALID_HANDLE_VALUE || stderr_handle == INVALID_HANDLE_VALUE) {
        if (stdout_handle != INVALID_HANDLE_VALUE) {
            CloseHandle(stdout_handle);
        }
        if (stderr_handle != INVALID_HANDLE_VALUE) {
            CloseHandle(stderr_handle);
        }
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        free(command);
        return -1;
    }
    ZeroMemory(&startup, sizeof(startup));
    ZeroMemory(&process, sizeof(process));
    startup.cb = sizeof(startup);
    startup.dwFlags = STARTF_USESTDHANDLES;
    startup.hStdOutput = stdout_handle;
    startup.hStdError = stderr_handle;
    startup.hStdInput = GetStdHandle(STD_INPUT_HANDLE);
    if (!CreateProcessA(NULL, command, NULL, NULL, TRUE, 0, NULL, NULL, &startup, &process)) {
        CloseHandle(stdout_handle);
        CloseHandle(stderr_handle);
        free(quoted_compiler);
        free(quoted_output);
        free(quoted_source);
        free(command);
        return -1;
    }
    WaitForSingleObject(process.hProcess, INFINITE);
    GetExitCodeProcess(process.hProcess, &exit_code);
    CloseHandle(process.hProcess);
    CloseHandle(process.hThread);
    CloseHandle(stdout_handle);
    CloseHandle(stderr_handle);
    free(quoted_compiler);
    free(quoted_output);
    free(quoted_source);
    free(command);
    return (int)exit_code;
}
#else
static int jayess_spawn_compiler(const char *compiler, const char *emit_arg, const char *target_arg, const char *warnings_arg, const char *output_path, const char *source_path, const char *stdout_path, const char *stderr_path) {
    int stdout_fd = open(stdout_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int stderr_fd = open(stderr_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int status = 1;
    pid_t pid;
    if (stdout_fd < 0 || stderr_fd < 0) {
        if (stdout_fd >= 0) {
            close(stdout_fd);
        }
        if (stderr_fd >= 0) {
            close(stderr_fd);
        }
        return -1;
    }
    pid = fork();
    if (pid < 0) {
        close(stdout_fd);
        close(stderr_fd);
        return -1;
    }
    if (pid == 0) {
        char *argv[9];
        int argc = 0;
        dup2(stdout_fd, STDOUT_FILENO);
        dup2(stderr_fd, STDERR_FILENO);
        close(stdout_fd);
        close(stderr_fd);
        argv[argc++] = (char *)compiler;
        argv[argc++] = (char *)emit_arg;
        if (target_arg != NULL) {
            argv[argc++] = (char *)target_arg;
        }
        if (warnings_arg != NULL) {
            argv[argc++] = (char *)warnings_arg;
        }
        argv[argc++] = "-o";
        argv[argc++] = (char *)output_path;
        argv[argc++] = (char *)source_path;
        argv[argc] = NULL;
        execvp(compiler, argv);
        _exit(127);
    }
    close(stdout_fd);
    close(stderr_fd);
    if (waitpid(pid, &status, 0) < 0) {
        return -1;
    }
    if (WIFEXITED(status)) {
        return WEXITSTATUS(status);
    }
    return -1;
}
#endif

static const char *jayess_temp_dir(void) {
#ifdef _WIN32
    const char *tmp = getenv("TEMP");
    if (tmp == NULL || tmp[0] == '\0') {
        tmp = getenv("TMP");
    }
    return (tmp != NULL && tmp[0] != '\0') ? tmp : ".";
#else
    const char *tmp = getenv("TMPDIR");
    return (tmp != NULL && tmp[0] != '\0') ? tmp : "/tmp";
#endif
}

static char *jayess_read_text_file_or_empty(const char *path) {
    FILE *file;
    long size;
    char *text;
    size_t read_count;
    if (path == NULL) {
        return jayess_strdup("");
    }
    file = fopen(path, "rb");
    if (file == NULL) {
        return jayess_strdup("");
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fclose(file);
        return jayess_strdup("");
    }
    size = ftell(file);
    if (size < 0) {
        fclose(file);
        return jayess_strdup("");
    }
    rewind(file);
    text = (char *)malloc((size_t)size + 1);
    if (text == NULL) {
        fclose(file);
        return jayess_strdup("");
    }
    read_count = fread(text, 1, (size_t)size, file);
    text[read_count] = '\0';
    fclose(file);
    return text;
}

static char *jayess_compile_option_string(jayess_value *options, const char *key) {
    jayess_value *value;
    if (options == NULL || options->kind != JAYESS_VALUE_OBJECT || options->as.object_value == NULL || key == NULL) {
        return NULL;
    }
    value = jayess_object_get(options->as.object_value, key);
    if (value == NULL || jayess_value_is_nullish(value)) {
        return NULL;
    }
    return jayess_value_stringify(value);
}

static int jayess_compile_is_safe_flag_value(const char *value) {
    size_t i;
    if (value == NULL || value[0] == '\0') {
        return 1;
    }
    for (i = 0; value[i] != '\0'; i++) {
        unsigned char ch = (unsigned char)value[i];
        if (!(isalnum(ch) || ch == '-' || ch == '_' || ch == '.')) {
            return 0;
        }
    }
    return 1;
}

static int jayess_compile_emit_is_valid(const char *value) {
    return value == NULL || value[0] == '\0' || strcmp(value, "exe") == 0 || strcmp(value, "llvm") == 0;
}

static int jayess_compile_warnings_is_valid(const char *value) {
    return value == NULL || value[0] == '\0' || strcmp(value, "default") == 0 || strcmp(value, "none") == 0 || strcmp(value, "error") == 0;
}

static jayess_value *jayess_compile_invalid_options_result(char *source_text, char *output_text, char *target_text, char *emit_text, char *warnings_text, const char *message) {
    jayess_object *result = jayess_object_new();
    jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
    jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
    jayess_object_set_value(result, "status", jayess_value_from_number(-1));
    jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
    jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
    jayess_object_set_value(result, "error", jayess_value_from_string(message != NULL ? message : "invalid compile options"));
    free(source_text);
    free(output_text);
    free(target_text);
    free(emit_text);
    free(warnings_text);
    return jayess_value_from_object(result);
}

static jayess_value *jayess_std_compile_impl(jayess_value *input, jayess_value *options, int input_is_path) {
    char *source_text = input_is_path ? NULL : jayess_value_stringify(input);
    char *input_path_text = input_is_path ? jayess_value_stringify(input) : NULL;
    char *output_text = NULL;
    char *target_text = NULL;
    char *emit_text = NULL;
    char *warnings_text = NULL;
    const char *compiler = getenv("JAYESS_COMPILER");
    const char *tmp_dir = jayess_temp_dir();
    char temp_source_path[4096];
    char default_output[4096];
    char stdout_path[4096];
    char stderr_path[4096];
    char *emit_arg = NULL;
    char *target_arg = NULL;
    char *warnings_arg = NULL;
    char *stdout_text = NULL;
    char *stderr_text = NULL;
    FILE *file;
    int status;
    jayess_object *result = jayess_object_new();
    long stamp = (long)time(NULL);
#ifdef _WIN32
    const char *exe_suffix = ".exe";
    const char sep = '\\';
#else
    const char *exe_suffix = "";
    const char sep = '/';
#endif
    if (compiler == NULL || compiler[0] == '\0') {
        compiler = "jayess";
    }
    if (options != NULL && !jayess_value_is_nullish(options)) {
        if (options->kind == JAYESS_VALUE_OBJECT && options->as.object_value != NULL) {
            output_text = jayess_compile_option_string(options, "output");
            target_text = jayess_compile_option_string(options, "target");
            emit_text = jayess_compile_option_string(options, "emit");
            warnings_text = jayess_compile_option_string(options, "warnings");
        } else {
            output_text = jayess_value_stringify(options);
        }
    }
    if (input_is_path && (input_path_text == NULL || input_path_text[0] == '\0')) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compileFile expects a non-empty input path");
    }
    if (!input_is_path && source_text == NULL) {
        source_text = jayess_strdup("");
    }
    if (!jayess_compile_emit_is_valid(emit_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option emit must be \"exe\" or \"llvm\"");
    }
    if (!jayess_compile_warnings_is_valid(warnings_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option warnings must be \"default\", \"none\", or \"error\"");
    }
    if (!jayess_compile_is_safe_flag_value(target_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile option target contains unsupported characters");
    }
    if (!jayess_compile_is_safe_flag_value(emit_text) || !jayess_compile_is_safe_flag_value(warnings_text)) {
        return jayess_compile_invalid_options_result(source_text, output_text, target_text, emit_text, warnings_text, "compile options contain unsupported characters");
    }
    snprintf(temp_source_path, sizeof(temp_source_path), "%s%cjayess-runtime-%ld-%d.js", tmp_dir, sep, stamp, rand());
    snprintf(stdout_path, sizeof(stdout_path), "%s%cjayess-runtime-%ld-%d.stdout", tmp_dir, sep, stamp, rand());
    snprintf(stderr_path, sizeof(stderr_path), "%s%cjayess-runtime-%ld-%d.stderr", tmp_dir, sep, stamp, rand());
    if (output_text == NULL || output_text[0] == '\0') {
        snprintf(default_output, sizeof(default_output), "%s%cjayess-runtime-%ld-%d%s", tmp_dir, sep, stamp, rand(), exe_suffix);
        if (output_text != NULL) {
            free(output_text);
        }
        output_text = jayess_strdup(default_output);
    }
    if (!input_is_path) {
        file = fopen(temp_source_path, "wb");
        if (file == NULL) {
            jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
            jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
            jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
            jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
            jayess_object_set_value(result, "error", jayess_value_from_string("failed to create temporary source file"));
            free(source_text);
            free(input_path_text);
            free(output_text);
            free(target_text);
            free(emit_text);
            free(warnings_text);
            return jayess_value_from_object(result);
        }
        fwrite(source_text, 1, strlen(source_text), file);
        fclose(file);
    }
    emit_arg = jayess_compile_flag("--emit", emit_text != NULL && emit_text[0] != '\0' ? emit_text : "exe");
    target_arg = target_text != NULL && target_text[0] != '\0' ? jayess_compile_flag("--target", target_text) : NULL;
    warnings_arg = warnings_text != NULL && warnings_text[0] != '\0' ? jayess_compile_flag("--warnings", warnings_text) : NULL;
    if (emit_arg == NULL || (target_text != NULL && target_text[0] != '\0' && target_arg == NULL) || (warnings_text != NULL && warnings_text[0] != '\0' && warnings_arg == NULL)) {
        jayess_object_set_value(result, "ok", jayess_value_from_bool(0));
        jayess_object_set_value(result, "output", output_text != NULL ? jayess_value_from_string(output_text) : jayess_value_undefined());
        jayess_object_set_value(result, "stdout", jayess_value_from_string(""));
        jayess_object_set_value(result, "stderr", jayess_value_from_string(""));
        jayess_object_set_value(result, "error", jayess_value_from_string("failed to build compiler command"));
        if (!input_is_path) {
            remove(temp_source_path);
        }
        free(source_text);
        free(input_path_text);
        free(output_text);
        free(target_text);
        free(emit_text);
        free(warnings_text);
        free(emit_arg);
        free(target_arg);
        free(warnings_arg);
        return jayess_value_from_object(result);
    }
    status = jayess_spawn_compiler(compiler, emit_arg, target_arg, warnings_arg, output_text, input_is_path ? input_path_text : temp_source_path, stdout_path, stderr_path);
    stdout_text = jayess_read_text_file_or_empty(stdout_path);
    stderr_text = jayess_read_text_file_or_empty(stderr_path);
    if (!input_is_path) {
        remove(temp_source_path);
    }
    remove(stdout_path);
    remove(stderr_path);
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status == 0));
    jayess_object_set_value(result, "output", jayess_value_from_string(output_text));
    jayess_object_set_value(result, "status", jayess_value_from_number((double)status));
    jayess_object_set_value(result, "stdout", jayess_value_from_string(stdout_text != NULL ? stdout_text : ""));
    jayess_object_set_value(result, "stderr", jayess_value_from_string(stderr_text != NULL ? stderr_text : ""));
    jayess_object_set_value(result, "error", status == 0 ? jayess_value_undefined() : jayess_value_from_string((stderr_text != NULL && stderr_text[0] != '\0') ? stderr_text : "compiler command failed"));
    free(source_text);
    free(input_path_text);
    free(output_text);
    free(target_text);
    free(emit_text);
    free(warnings_text);
    free(emit_arg);
    free(target_arg);
    free(warnings_arg);
    free(stdout_text);
    free(stderr_text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_compile(jayess_value *source, jayess_value *options) {
    return jayess_std_compile_impl(source, options, 0);
}

jayess_value *jayess_std_compile_file(jayess_value *input_path, jayess_value *options) {
    return jayess_std_compile_impl(input_path, options, 1);
}

jayess_value *jayess_std_path_join(jayess_value *parts) {
    const char *sep = jayess_path_separator_string();
    size_t total = 1;
    char *out;
    int i;
    if (parts == NULL || parts->kind != JAYESS_VALUE_ARRAY || parts->as.array_value == NULL) {
        return jayess_value_from_string("");
    }
    for (i = 0; i < parts->as.array_value->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        total += strlen(piece != NULL ? piece : "");
        if (i > 0) {
            total += strlen(sep);
        }
        free(piece);
    }
    out = (char *)malloc(total);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    for (i = 0; i < parts->as.array_value->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        if (i > 0) {
            strcat(out, sep);
        }
        strcat(out, piece != NULL ? piece : "");
        free(piece);
    }
    parts = jayess_value_from_string(out);
    free(out);
    return parts;
}

jayess_value *jayess_std_path_normalize(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *sep = jayess_path_separator_string();
    int absolute = jayess_path_is_absolute(path_text);
    jayess_array *segments = jayess_array_new();
    const char *cursor = path_text != NULL ? path_text : "";
    while (*cursor != '\0') {
        const char *start = cursor;
        while (*cursor != '\0' && !jayess_path_is_separator(*cursor)) {
            cursor++;
        }
        if (cursor > start) {
            size_t length = (size_t)(cursor - start);
            char *segment = (char *)malloc(length + 1);
            jayess_value *value;
            if (segment == NULL) {
                free(path_text);
                return jayess_value_from_string(path_text != NULL ? path_text : "");
            }
            memcpy(segment, start, length);
            segment[length] = '\0';
            if (strcmp(segment, ".") == 0) {
                free(segment);
            } else if (strcmp(segment, "..") == 0) {
                if (segments->count > 0) {
                    jayess_array_pop_value(segments);
                }
                free(segment);
            } else {
                value = jayess_value_from_string(segment);
                jayess_array_push_value(segments, value);
                free(segment);
            }
        }
        while (*cursor != '\0' && jayess_path_is_separator(*cursor)) {
            cursor++;
        }
    }
    {
        jayess_value *joined = jayess_std_path_join(jayess_value_from_array(segments));
        char *joined_text = jayess_value_stringify(joined);
        jayess_value *result;
        if (joined_text == NULL) {
            free(path_text);
            return jayess_value_from_string(absolute ? sep : ".");
        }
        if (absolute && !jayess_path_is_absolute(joined_text)) {
            size_t total = strlen(sep) + strlen(joined_text) + 1;
            char *prefixed = (char *)malloc(total);
            if (prefixed == NULL) {
                result = jayess_value_from_string(joined_text);
                free(joined_text);
                free(path_text);
                return result;
            }
            strcpy(prefixed, sep);
            strcat(prefixed, joined_text);
            free(joined_text);
            joined_text = prefixed;
        }
        if (!absolute && joined_text[0] == '\0') {
            free(joined_text);
            joined_text = jayess_strdup(".");
        }
        result = jayess_value_from_string(joined_text);
        free(joined_text);
        free(path_text);
        return result;
    }
}

jayess_value *jayess_std_path_resolve(jayess_value *parts) {
    jayess_array *values = jayess_array_new();
    int i;
    int start = 0;
    if (parts == NULL || parts->kind != JAYESS_VALUE_ARRAY || parts->as.array_value == NULL || parts->as.array_value->count == 0) {
        return jayess_std_process_cwd();
    }
    for (i = parts->as.array_value->count - 1; i >= 0; i--) {
        jayess_value *part = jayess_array_get(parts->as.array_value, i);
        char *text = jayess_value_stringify(part);
        if (text != NULL && text[0] != '\0') {
            if (jayess_path_is_absolute(text)) {
                start = i;
                free(text);
                break;
            }
        }
        free(text);
    }
    if (i < 0) {
        jayess_array_push_value(values, jayess_std_process_cwd());
        start = 0;
    }
    for (i = start; i < parts->as.array_value->count; i++) {
        char *text = jayess_value_stringify(jayess_array_get(parts->as.array_value, i));
        if (text != NULL && text[0] != '\0') {
            jayess_array_push_value(values, jayess_value_from_string(text));
        }
        free(text);
    }
    return jayess_std_path_normalize(jayess_std_path_join(jayess_value_from_array(values)));
}

jayess_value *jayess_std_path_relative(jayess_value *from, jayess_value *to) {
    jayess_array *from_parts = jayess_array_new();
    jayess_array *to_parts = jayess_array_new();
    jayess_value *from_resolved;
    jayess_value *to_resolved;
    char *from_text;
    char *to_text;
    jayess_array *from_segments;
    jayess_array *to_segments;
    jayess_array *relative_segments = jayess_array_new();
    int common = 0;
    int i;
    char *joined;
    if (relative_segments == NULL) {
        return jayess_value_from_string(".");
    }
    jayess_array_push_value(from_parts, from);
    jayess_array_push_value(to_parts, to);
    from_resolved = jayess_std_path_resolve(jayess_value_from_array(from_parts));
    to_resolved = jayess_std_path_resolve(jayess_value_from_array(to_parts));
    from_text = jayess_value_stringify(from_resolved);
    to_text = jayess_value_stringify(to_resolved);
    if (from_text == NULL || to_text == NULL) {
        free(from_text);
        free(to_text);
        return jayess_value_from_string(".");
    }
    from_segments = jayess_path_split_segments(from_text);
    to_segments = jayess_path_split_segments(to_text);
    if (jayess_path_root_length(from_text) != jayess_path_root_length(to_text)) {
        free(from_text);
        free(to_text);
        return to_resolved;
    }
#ifdef _WIN32
    if (_strnicmp(from_text, to_text, (size_t)jayess_path_root_length(from_text)) != 0) {
#else
    if (strncmp(from_text, to_text, (size_t)jayess_path_root_length(from_text)) != 0) {
#endif
        free(from_text);
        free(to_text);
        return to_resolved;
    }
    while (common < from_segments->count && common < to_segments->count) {
        const char *left = jayess_value_as_string(jayess_array_get(from_segments, common));
        const char *right = jayess_value_as_string(jayess_array_get(to_segments, common));
        if (strcmp(left, right) != 0) {
            break;
        }
        common++;
    }
    for (i = common; i < from_segments->count; i++) {
        jayess_array_push_value(relative_segments, jayess_value_from_string(".."));
    }
    for (i = common; i < to_segments->count; i++) {
        jayess_array_push_value(relative_segments, jayess_array_get(to_segments, i));
    }
    joined = jayess_path_join_segments_with_root("", relative_segments);
    free(from_text);
    free(to_text);
    if (joined == NULL) {
        return jayess_value_from_string(".");
    }
    from_resolved = jayess_value_from_string(joined);
    free(joined);
    return from_resolved;
}

jayess_value *jayess_std_path_parse(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    int root_length;
    const char *last_sep;
    const char *base;
    jayess_object *parsed = jayess_object_new();
    jayess_value *result;
    char *dir_text;
    char *base_text;
    char *ext_text;
    char *name_text;
    if (parsed == NULL) {
        free(path_text);
        return jayess_value_undefined();
    }
    if (path_text == NULL) {
        path_text = jayess_strdup("");
    }
    root_length = jayess_path_root_length(path_text);
    last_sep = jayess_path_last_separator(path_text);
    base = last_sep != NULL ? last_sep + 1 : path_text;
    dir_text = jayess_value_stringify(jayess_std_path_dirname(jayess_value_from_string(path_text)));
    base_text = jayess_strdup(base);
    ext_text = jayess_value_stringify(jayess_std_path_extname(jayess_value_from_string(path_text)));
    if (ext_text != NULL && ext_text[0] != '\0' && strlen(base_text) >= strlen(ext_text)) {
        size_t name_len = strlen(base_text) - strlen(ext_text);
        name_text = (char *)malloc(name_len + 1);
        if (name_text != NULL) {
            memcpy(name_text, base_text, name_len);
            name_text[name_len] = '\0';
        }
    } else {
        name_text = jayess_strdup(base_text != NULL ? base_text : "");
    }
    if (root_length > 0) {
        char *root_text = (char *)malloc((size_t)root_length + 1);
        if (root_text != NULL) {
            memcpy(root_text, path_text, (size_t)root_length);
            root_text[root_length] = '\0';
            jayess_object_set_value(parsed, "root", jayess_value_from_string(root_text));
            free(root_text);
        } else {
            jayess_object_set_value(parsed, "root", jayess_value_from_string(""));
        }
    } else {
        jayess_object_set_value(parsed, "root", jayess_value_from_string(""));
    }
    jayess_object_set_value(parsed, "dir", jayess_value_from_string(dir_text != NULL ? dir_text : "."));
    jayess_object_set_value(parsed, "base", jayess_value_from_string(base_text != NULL ? base_text : ""));
    jayess_object_set_value(parsed, "ext", jayess_value_from_string(ext_text != NULL ? ext_text : ""));
    jayess_object_set_value(parsed, "name", jayess_value_from_string(name_text != NULL ? name_text : ""));
    free(path_text);
    free(dir_text);
    free(base_text);
    free(ext_text);
    free(name_text);
    result = jayess_value_from_object(parsed);
    return result;
}

jayess_value *jayess_std_path_is_absolute(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    int absolute = jayess_path_is_absolute(path_text);
    free(path_text);
    return jayess_value_from_bool(absolute);
}

jayess_value *jayess_std_path_format(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    jayess_value *dirValue;
    jayess_value *rootValue;
    jayess_value *baseValue;
    jayess_value *nameValue;
    jayess_value *extValue;
    char *dirText;
    char *rootText;
    char *baseText;
    char *nameText;
    char *extText;
    char *out;
    size_t total;
    char sep = jayess_path_separator_char();
    if (object == NULL) {
        return jayess_value_from_string("");
    }
    dirValue = jayess_object_get(object, "dir");
    rootValue = jayess_object_get(object, "root");
    baseValue = jayess_object_get(object, "base");
    nameValue = jayess_object_get(object, "name");
    extValue = jayess_object_get(object, "ext");
    dirText = jayess_value_stringify(dirValue);
    rootText = jayess_value_stringify(rootValue);
    baseText = jayess_value_stringify(baseValue);
    nameText = jayess_value_stringify(nameValue);
    extText = jayess_value_stringify(extValue);
    if ((baseText == NULL || baseText[0] == '\0') && nameText != NULL) {
        size_t nameLen = strlen(nameText);
        size_t extLen = extText != NULL ? strlen(extText) : 0;
        baseText = (char *)realloc(baseText, nameLen + extLen + 1);
        if (baseText != NULL) {
            strcpy(baseText, nameText);
            if (extText != NULL) {
                strcat(baseText, extText);
            }
        }
    }
    total = strlen(dirText != NULL ? dirText : "") + strlen(rootText != NULL ? rootText : "") + strlen(baseText != NULL ? baseText : "") + 2;
    out = (char *)malloc(total);
    if (out == NULL) {
        free(dirText); free(rootText); free(baseText); free(nameText); free(extText);
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    if (dirText != NULL && dirText[0] != '\0') {
        strcpy(out, dirText);
        if (!jayess_path_is_separator(out[strlen(out)-1]) && baseText != NULL && baseText[0] != '\0') {
            size_t len = strlen(out);
            out[len] = sep;
            out[len+1] = '\0';
        }
    } else if (rootText != NULL && rootText[0] != '\0') {
        strcpy(out, rootText);
    }
    if (baseText != NULL) {
        strcat(out, baseText);
    }
    parts = jayess_value_from_string(out);
    free(out);
    free(dirText); free(rootText); free(baseText); free(nameText); free(extText);
    return parts;
}

jayess_value *jayess_std_path_sep(void) {
    return jayess_value_from_string(jayess_path_separator_string());
}

jayess_value *jayess_std_path_delimiter(void) {
    return jayess_value_from_string(jayess_path_delimiter_string());
}

jayess_value *jayess_std_path_basename(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *start;
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_string("");
    }
    start = jayess_path_last_separator(path_text);
    if (start == NULL) {
        result = jayess_value_from_string(path_text);
    } else {
        result = jayess_value_from_string(start + 1);
    }
    free(path_text);
    return result;
}

jayess_value *jayess_std_path_dirname(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *last;
    jayess_value *result;
    if (path_text == NULL || path_text[0] == '\0') {
        free(path_text);
        return jayess_value_from_string(".");
    }
    last = jayess_path_last_separator(path_text);
    if (last == NULL) {
        free(path_text);
        return jayess_value_from_string(".");
    }
    if (last == path_text) {
        path_text[1] = '\0';
    } else {
        path_text[last - path_text] = '\0';
    }
    result = jayess_value_from_string(path_text);
    free(path_text);
    return result;
}

jayess_value *jayess_std_path_extname(jayess_value *path) {
    char *path_text = jayess_value_stringify(path);
    const char *last_sep;
    const char *last_dot;
    jayess_value *result;
    if (path_text == NULL) {
        return jayess_value_from_string("");
    }
    last_sep = jayess_path_last_separator(path_text);
    last_dot = strrchr(path_text, '.');
    if (last_dot == NULL || (last_sep != NULL && last_dot < last_sep + 1)) {
        result = jayess_value_from_string("");
    } else {
        result = jayess_value_from_string(last_dot);
    }
    free(path_text);
    return result;
}

static char *jayess_substring(const char *text, size_t start, size_t end) {
    size_t len;
    char *out;
    if (text == NULL || end < start) {
        return jayess_strdup("");
    }
    len = end - start;
    out = (char *)malloc(len + 1);
    if (out == NULL) {
        return jayess_strdup("");
    }
    memcpy(out, text + start, len);
    out[len] = '\0';
    return out;
}

static int jayess_hex_value(char ch) {
    if (ch >= '0' && ch <= '9') {
        return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
        return ch - 'a' + 10;
    }
    if (ch >= 'A' && ch <= 'F') {
        return ch - 'A' + 10;
    }
    return -1;
}

static char *jayess_percent_decode(const char *text) {
    size_t len = text != NULL ? strlen(text) : 0;
    char *out = (char *)malloc(len + 1);
    size_t i;
    size_t j = 0;
    if (out == NULL) {
        return jayess_strdup("");
    }
    for (i = 0; i < len; i++) {
        if (text[i] == '%' && i + 2 < len) {
            int hi = jayess_hex_value(text[i + 1]);
            int lo = jayess_hex_value(text[i + 2]);
            if (hi >= 0 && lo >= 0) {
                out[j++] = (char)((hi << 4) | lo);
                i += 2;
                continue;
            }
        }
        out[j++] = text[i] == '+' ? ' ' : text[i];
    }
    out[j] = '\0';
    return out;
}

static int jayess_url_should_encode(unsigned char ch) {
    return !(isalnum(ch) || ch == '-' || ch == '_' || ch == '.' || ch == '~');
}

static char *jayess_percent_encode(const char *text) {
    static const char *hex = "0123456789ABCDEF";
    size_t len = text != NULL ? strlen(text) : 0;
    size_t out_len = 0;
    size_t i;
    size_t j = 0;
    char *out;
    for (i = 0; i < len; i++) {
        out_len += jayess_url_should_encode((unsigned char)text[i]) ? 3 : 1;
    }
    out = (char *)malloc(out_len + 1);
    if (out == NULL) {
        return jayess_strdup("");
    }
    for (i = 0; i < len; i++) {
        unsigned char ch = (unsigned char)text[i];
        if (jayess_url_should_encode(ch)) {
            out[j++] = '%';
            out[j++] = hex[(ch >> 4) & 15];
            out[j++] = hex[ch & 15];
        } else {
            out[j++] = (char)ch;
        }
    }
    out[j] = '\0';
    return out;
}

static char *jayess_http_trim_copy(const char *text) {
    const char *start = text != NULL ? text : "";
    const char *end = start + strlen(start);
    while (start < end && isspace((unsigned char)*start)) {
        start++;
    }
    while (end > start && isspace((unsigned char)*(end - 1))) {
        end--;
    }
    return jayess_substring(start, 0, (size_t)(end - start));
}

static const char *jayess_http_line_end(const char *cursor) {
    while (cursor != NULL && *cursor != '\0' && *cursor != '\r' && *cursor != '\n') {
        cursor++;
    }
    return cursor;
}

static const char *jayess_http_next_line(const char *cursor) {
    if (cursor == NULL) {
        return NULL;
    }
    if (*cursor == '\r' && *(cursor + 1) == '\n') {
        return cursor + 2;
    }
    if (*cursor == '\r' || *cursor == '\n') {
        return cursor + 1;
    }
    return cursor;
}

static const char *jayess_http_header_boundary(const char *text) {
    const char *cursor = text != NULL ? text : "";
    while (*cursor != '\0') {
        if (cursor[0] == '\r' && cursor[1] == '\n' && cursor[2] == '\r' && cursor[3] == '\n') {
            return cursor;
        }
        if (cursor[0] == '\n' && cursor[1] == '\n') {
            return cursor;
        }
        cursor++;
    }
    return NULL;
}

static jayess_object *jayess_http_parse_header_object(const char *text) {
    jayess_object *headers = jayess_object_new();
    const char *cursor = text != NULL ? text : "";
    while (*cursor != '\0') {
        const char *line_end = jayess_http_line_end(cursor);
        const char *colon = cursor;
        while (colon < line_end && *colon != ':') {
            colon++;
        }
        if (colon < line_end) {
            char *key_raw = jayess_substring(cursor, 0, (size_t)(colon - cursor));
            char *value_raw = jayess_substring(colon + 1, 0, (size_t)(line_end - colon - 1));
            char *key = jayess_http_trim_copy(key_raw);
            char *value = jayess_http_trim_copy(value_raw);
            if (key != NULL && key[0] != '\0') {
                jayess_object_set_value(headers, key, jayess_value_from_string(value != NULL ? value : ""));
            }
            free(key_raw);
            free(value_raw);
            free(key);
            free(value);
        }
        cursor = jayess_http_next_line(line_end);
    }
    return headers;
}

static int jayess_http_text_contains_ci(const char *text, const char *token) {
    size_t text_len = text != NULL ? strlen(text) : 0;
    size_t token_len = token != NULL ? strlen(token) : 0;
    size_t i;
    if (token_len == 0 || text_len < token_len) {
        return 0;
    }
    for (i = 0; i + token_len <= text_len; i++) {
        size_t j = 0;
        while (j < token_len && tolower((unsigned char)text[i + j]) == tolower((unsigned char)token[j])) {
            j++;
        }
        if (j == token_len) {
            return 1;
        }
    }
    return 0;
}

static int jayess_http_text_eq_ci(const char *left, const char *right) {
    size_t i = 0;
    if (left == NULL || right == NULL) {
        return left == right;
    }
    while (left[i] != '\0' && right[i] != '\0') {
        if (tolower((unsigned char)left[i]) != tolower((unsigned char)right[i])) {
            return 0;
        }
        i++;
    }
    return left[i] == '\0' && right[i] == '\0';
}

static int jayess_http_is_redirect_status(int status) {
    return status == 301 || status == 302 || status == 303 || status == 307 || status == 308;
}

static char *jayess_http_request_current_url(jayess_object *request_object) {
    char *scheme_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "scheme")) : jayess_strdup("http");
    char *host_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "host")) : jayess_strdup("");
    char *path_text = request_object != NULL ? jayess_value_stringify(jayess_object_get(request_object, "path")) : jayess_strdup("/");
    int port = (int)jayess_value_to_number(request_object != NULL ? jayess_object_get(request_object, "port") : jayess_value_from_number(80));
    size_t total;
    char *url;
    const char *scheme = scheme_text != NULL && scheme_text[0] != '\0' ? scheme_text : "http";
    int default_port = strcmp(scheme, "https") == 0 ? 443 : 80;
    if (host_text == NULL || host_text[0] == '\0') {
        free(scheme_text);
        free(host_text);
        free(path_text);
        return jayess_strdup("");
    }
    total = strlen(scheme) + strlen(host_text) + strlen(path_text != NULL && path_text[0] != '\0' ? path_text : "/") + 32;
    url = (char *)malloc(total);
    if (url == NULL) {
        free(scheme_text);
        free(host_text);
        free(path_text);
        return jayess_strdup("");
    }
    if (port > 0 && port != default_port) {
        snprintf(url, total, "%s://%s:%d%s", scheme, host_text, port, path_text != NULL && path_text[0] != '\0' ? path_text : "/");
    } else {
        snprintf(url, total, "%s://%s%s", scheme, host_text, path_text != NULL && path_text[0] != '\0' ? path_text : "/");
    }
    free(scheme_text);
    free(host_text);
    free(path_text);
    return url;
}

static int jayess_std_socket_configure_timeout(jayess_socket_handle handle, int timeout) {
    if (timeout < 0) {
        timeout = 0;
    }
    if (handle == JAYESS_INVALID_SOCKET) {
        return 0;
    }
#ifdef _WIN32
    {
        DWORD timeout_value = (DWORD)timeout;
        return setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) == 0 &&
            setsockopt(handle, SOL_SOCKET, SO_SNDTIMEO, (const char *)&timeout_value, sizeof(timeout_value)) == 0;
    }
#else
    {
        struct timeval timeout_value;
        timeout_value.tv_sec = timeout / 1000;
        timeout_value.tv_usec = (timeout % 1000) * 1000;
        return setsockopt(handle, SOL_SOCKET, SO_RCVTIMEO, &timeout_value, sizeof(timeout_value)) == 0 &&
            setsockopt(handle, SOL_SOCKET, SO_SNDTIMEO, &timeout_value, sizeof(timeout_value)) == 0;
    }
#endif
}

static jayess_value *jayess_http_header_get_ci(jayess_object *headers, const char *key) {
    jayess_object_entry *entry = headers != NULL ? headers->head : NULL;
    while (entry != NULL) {
        if (entry->key != NULL && jayess_http_text_eq_ci(entry->key, key)) {
            return entry->value;
        }
        entry = entry->next;
    }
    return NULL;
}

static int jayess_http_headers_transfer_chunked(jayess_object *headers) {
    jayess_value *value = jayess_http_header_get_ci(headers, "Transfer-Encoding");
    if (value != NULL) {
        char *text = jayess_value_stringify(value);
        int matches = jayess_http_text_contains_ci(text, "chunked");
        free(text);
        if (matches) {
            return 1;
        }
    }
    return 0;
}

static long jayess_http_headers_content_length(jayess_object *headers) {
    jayess_value *value = jayess_http_header_get_ci(headers, "Content-Length");
    if (value != NULL) {
        char *text = jayess_value_stringify(value);
        char *trimmed = jayess_http_trim_copy(text);
        char *end_ptr;
        long length = -1;
        if (trimmed != NULL && trimmed[0] != '\0') {
            length = strtol(trimmed, &end_ptr, 10);
            if (end_ptr == trimmed || *end_ptr != '\0' || length < 0) {
                length = -1;
            }
        }
        free(text);
        free(trimmed);
        return length;
    }
    return -1;
}

static char *jayess_http_decode_chunked_body(const char *body) {
    const char *cursor = body != NULL ? body : "";
    char *out = jayess_strdup("");
    size_t out_len = 0;
    if (out == NULL) {
        return jayess_strdup("");
    }
    while (*cursor != '\0') {
        const char *line_end = jayess_http_line_end(cursor);
        const char *size_end = cursor;
        size_t chunk_size = 0;
        char *size_raw;
        char *size_text;
        char *end_ptr;
        char *next;
        if (line_end == cursor) {
            break;
        }
        while (size_end < line_end && *size_end != ';') {
            size_end++;
        }
        size_raw = jayess_substring(cursor, 0, (size_t)(size_end - cursor));
        size_text = jayess_http_trim_copy(size_raw);
        free(size_raw);
        if (size_text == NULL || size_text[0] == '\0') {
            free(size_text);
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        chunk_size = (size_t)strtoul(size_text, &end_ptr, 16);
        if (end_ptr == size_text || *end_ptr != '\0') {
            free(size_text);
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        free(size_text);
        cursor = jayess_http_next_line(line_end);
        if (chunk_size == 0) {
            return out;
        }
        if (strlen(cursor) < chunk_size) {
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
        next = (char *)realloc(out, out_len + chunk_size + 1);
        if (next == NULL) {
            free(out);
            return jayess_strdup("");
        }
        out = next;
        memcpy(out + out_len, cursor, chunk_size);
        out_len += chunk_size;
        out[out_len] = '\0';
        cursor += chunk_size;
        if (cursor[0] == '\r' && cursor[1] == '\n') {
            cursor += 2;
        } else if (cursor[0] == '\n') {
            cursor += 1;
        } else if (cursor[0] != '\0') {
            free(out);
            return jayess_strdup(body != NULL ? body : "");
        }
    }
    return out;
}

static int jayess_http_chunked_body_complete(const char *body, size_t available) {
    const char *cursor = body != NULL ? body : "";
    const char *end = cursor + available;
    while (cursor < end) {
        const char *line_end = cursor;
        const char *size_end = cursor;
        char *size_raw;
        char *size_text;
        char *end_ptr;
        size_t chunk_size;
        while (line_end < end && *line_end != '\r' && *line_end != '\n') {
            line_end++;
        }
        if (line_end >= end) {
            return 0;
        }
        while (size_end < line_end && *size_end != ';') {
            size_end++;
        }
        size_raw = jayess_substring(cursor, 0, (size_t)(size_end - cursor));
        size_text = jayess_http_trim_copy(size_raw);
        free(size_raw);
        if (size_text == NULL || size_text[0] == '\0') {
            free(size_text);
            return 0;
        }
        chunk_size = (size_t)strtoul(size_text, &end_ptr, 16);
        free(size_text);
        if (end_ptr == NULL || *end_ptr != '\0') {
            return 0;
        }
        cursor = jayess_http_next_line(line_end);
        if ((size_t)(end - cursor) < chunk_size) {
            return 0;
        }
        cursor += chunk_size;
        if (cursor >= end) {
            return 0;
        }
        if (cursor[0] == '\r') {
            if (cursor + 1 >= end || cursor[1] != '\n') {
                return 0;
            }
            cursor += 2;
        } else if (cursor[0] == '\n') {
            cursor += 1;
        } else {
            return 0;
        }
        if (chunk_size == 0) {
            if (cursor < end && cursor[0] == '\r') {
                return cursor + 1 < end && cursor[1] == '\n';
            }
            if (cursor < end && cursor[0] == '\n') {
                return 1;
            }
            return cursor >= end;
        }
    }
    return 0;
}

static void jayess_http_body_stream_mark_ended(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "readableEnded", jayess_value_from_bool(1));
        jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
    }
}

static jayess_value *jayess_http_body_stream_socket_value(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    return jayess_object_get(env->as.object_value, "__jayess_http_body_socket");
}

static void jayess_http_body_stream_close_socket(jayess_value *env) {
    jayess_value *socket_value = jayess_http_body_stream_socket_value(env);
    jayess_socket_handle handle;
    if (socket_value != NULL && socket_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(socket_value, "Socket")) {
        jayess_std_socket_close_method(socket_value);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
            jayess_object_set_value(env->as.object_value, "__jayess_http_body_socket", jayess_value_undefined());
        }
        return;
    }
    handle = jayess_std_socket_handle(env);
    if (handle != JAYESS_INVALID_SOCKET) {
        jayess_std_socket_close_handle(handle);
        jayess_std_socket_set_handle(env, JAYESS_INVALID_SOCKET);
    }
}

static void jayess_http_body_stream_close_native(jayess_value *env) {
#ifdef _WIN32
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL && env->as.object_value->native_handle != NULL) {
        jayess_winhttp_stream_state *state = (jayess_winhttp_stream_state *)env->as.object_value->native_handle;
        if (state->request != NULL) {
            WinHttpCloseHandle(state->request);
        }
        if (state->connection != NULL) {
            WinHttpCloseHandle(state->connection);
        }
        if (state->session != NULL) {
            WinHttpCloseHandle(state->session);
        }
        free(state);
        env->as.object_value->native_handle = NULL;
    }
#else
    (void)env;
#endif
}

static void jayess_http_body_stream_emit_end(jayess_value *env) {
    jayess_http_body_stream_mark_ended(env);
    jayess_http_body_stream_close_socket(env);
    jayess_http_body_stream_close_native(env);
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_std_stream_emit(env, "end", jayess_value_undefined());
    }
}

static jayess_array *jayess_http_body_stream_prebuffer_bytes(jayess_value *env) {
    jayess_value *prebuffer;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    prebuffer = jayess_object_get(env->as.object_value, "__jayess_http_body_prebuffer");
    if (prebuffer == NULL || prebuffer->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(prebuffer, "Uint8Array")) {
        return NULL;
    }
    return jayess_std_bytes_slot(prebuffer);
}

static int jayess_http_body_stream_take_prebuffer(jayess_value *env, unsigned char *buffer, int max_count) {
    jayess_array *bytes = jayess_http_body_stream_prebuffer_bytes(env);
    int offset;
    int available;
    int count;
    int i;
    if (bytes == NULL || max_count <= 0 || env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return 0;
    }
    offset = (int)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_prebuffer_offset"));
    if (offset < 0) {
        offset = 0;
    }
    if (offset >= bytes->count) {
        return 0;
    }
    available = bytes->count - offset;
    count = available < max_count ? available : max_count;
    for (i = 0; i < count; i++) {
        buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, offset + i)) & 255);
    }
    jayess_object_set_value(env->as.object_value, "__jayess_http_body_prebuffer_offset", jayess_value_from_number((double)(offset + count)));
    return count;
}

static int jayess_http_body_stream_read_raw(jayess_value *env, unsigned char *buffer, int max_count) {
    int count = 0;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || buffer == NULL || max_count <= 0) {
        return -1;
    }
    count = jayess_http_body_stream_take_prebuffer(env, buffer, max_count);
    if (count > 0) {
        return count;
    }
    {
        jayess_value *socket_value = jayess_http_body_stream_socket_value(env);
        if (socket_value != NULL && socket_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(socket_value, "Socket")) {
            int did_timeout = 0;
            if (jayess_std_tls_state(socket_value) != NULL) {
                return jayess_std_tls_read_bytes(socket_value, buffer, max_count, &did_timeout);
            }
        }
        jayess_socket_handle handle = jayess_std_socket_handle(env);
        if (handle != JAYESS_INVALID_SOCKET) {
            int read_count = (int)recv(handle, (char *)buffer, max_count, 0);
            if (read_count < 0) {
                jayess_std_stream_emit_error(env, "failed to read from HTTP body stream");
                jayess_http_body_stream_close_socket(env);
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (read_count == 0) {
                jayess_http_body_stream_close_socket(env);
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            return read_count;
        }
#ifdef _WIN32
        if (env->as.object_value->native_handle != NULL) {
            jayess_winhttp_stream_state *state = (jayess_winhttp_stream_state *)env->as.object_value->native_handle;
            DWORD available = 0;
            DWORD read_now = 0;
            DWORD to_read = 0;
            if (state == NULL || state->request == NULL) {
                return 0;
            }
            if (!WinHttpQueryDataAvailable(state->request, &available)) {
                jayess_std_stream_emit_error(env, "failed to query HTTPS body availability");
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (available == 0) {
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            to_read = available < (DWORD)max_count ? available : (DWORD)max_count;
            if (!WinHttpReadData(state->request, buffer, to_read, &read_now)) {
                jayess_std_stream_emit_error(env, "failed to read from HTTPS body stream");
                jayess_http_body_stream_close_native(env);
                jayess_http_body_stream_mark_ended(env);
                return -1;
            }
            if (read_now == 0) {
                jayess_http_body_stream_close_native(env);
                return 0;
            }
            return (int)read_now;
        }
#endif
        return 0;
    }
}

static int jayess_http_body_stream_read_byte(jayess_value *env, unsigned char *out) {
    unsigned char byte_value = 0;
    int read_count = jayess_http_body_stream_read_raw(env, &byte_value, 1);
    if (read_count > 0 && out != NULL) {
        *out = byte_value;
    }
    return read_count;
}

static char *jayess_http_body_stream_read_line(jayess_value *env) {
    size_t capacity = 32;
    size_t length = 0;
    char *line = (char *)malloc(capacity);
    if (line == NULL) {
        return NULL;
    }
    for (;;) {
        unsigned char byte_value = 0;
        int read_count = jayess_http_body_stream_read_byte(env, &byte_value);
        if (read_count <= 0) {
            free(line);
            return NULL;
        }
        if (byte_value == '\n') {
            if (length > 0 && line[length - 1] == '\r') {
                length--;
            }
            line[length] = '\0';
            return line;
        }
        if (length + 1 >= capacity) {
            size_t next_capacity = capacity * 2;
            char *next = (char *)realloc(line, next_capacity);
            if (next == NULL) {
                free(line);
                return NULL;
            }
            line = next;
            capacity = next_capacity;
        }
        line[length++] = (char)byte_value;
    }
}

static int jayess_http_body_stream_consume_crlf(jayess_value *env) {
    unsigned char first = 0;
    int read_first = jayess_http_body_stream_read_byte(env, &first);
    if (read_first <= 0) {
        return 0;
    }
    if (first == '\n') {
        return 1;
    }
    if (first == '\r') {
        unsigned char second = 0;
        int read_second = jayess_http_body_stream_read_byte(env, &second);
        return read_second > 0 && second == '\n';
    }
    return 0;
}

static jayess_value *jayess_http_body_stream_make_string(const unsigned char *buffer, int count) {
    char *text;
    jayess_value *result;
    if (buffer == NULL || count <= 0) {
        return jayess_value_from_string("");
    }
    text = (char *)malloc((size_t)count + 1);
    if (text == NULL) {
        return jayess_value_undefined();
    }
    memcpy(text, buffer, (size_t)count);
    text[count] = '\0';
    result = jayess_value_from_string(text);
    free(text);
    return result;
}

static jayess_value *jayess_http_body_stream_read_non_chunked(jayess_value *env, jayess_value *size_value, int as_bytes) {
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    long remaining = (long)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_remaining"));
    unsigned char *buffer;
    int total = 0;
    if (remaining == 0) {
        jayess_http_body_stream_emit_end(env);
        return jayess_value_null();
    }
    if (remaining > 0 && requested > remaining) {
        requested = (int)remaining;
    }
    buffer = (unsigned char *)malloc((size_t)requested);
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate HTTP body buffer");
        return jayess_value_undefined();
    }
    while (total < requested) {
        int read_count = jayess_http_body_stream_read_raw(env, buffer + total, requested - total);
        if (read_count < 0) {
            free(buffer);
            return jayess_value_undefined();
        }
        if (read_count == 0) {
            if (remaining < 0) {
                break;
            }
            jayess_std_stream_emit_error(env, "HTTP body stream ended before expected Content-Length");
            free(buffer);
            return jayess_value_undefined();
        }
        total += read_count;
        if (remaining < 0) {
            break;
        }
    }
    if (total == 0) {
        free(buffer);
        jayess_http_body_stream_emit_end(env);
        return jayess_value_null();
    }
    if (remaining > 0) {
        remaining -= total;
        if (remaining < 0) {
            remaining = 0;
        }
        jayess_object_set_value(env->as.object_value, "__jayess_http_body_remaining", jayess_value_from_number((double)remaining));
        if (remaining == 0) {
            jayess_http_body_stream_mark_ended(env);
            jayess_http_body_stream_close_socket(env);
        }
    }
    if (as_bytes) {
        jayess_value *result = jayess_std_uint8_array_from_bytes(buffer, (size_t)total);
        free(buffer);
        if (remaining == 0) {
            jayess_std_stream_emit(env, "end", jayess_value_undefined());
        }
        return result;
    }
    {
        jayess_value *result = jayess_http_body_stream_make_string(buffer, total);
        free(buffer);
        if (remaining == 0) {
            jayess_std_stream_emit(env, "end", jayess_value_undefined());
        }
        return result;
    }
}

static jayess_value *jayess_http_body_stream_read_chunked(jayess_value *env, jayess_value *size_value, int as_bytes) {
    int requested = jayess_std_stream_requested_size(size_value, 4095);
    unsigned char *buffer = (unsigned char *)malloc((size_t)requested);
    int total = 0;
    if (buffer == NULL) {
        jayess_std_stream_emit_error(env, "failed to allocate HTTP chunk buffer");
        return jayess_value_undefined();
    }
    for (;;) {
        long chunk_remaining = (long)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_http_body_chunk_remaining"));
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "__jayess_http_body_chunk_finished"))) {
            free(buffer);
            jayess_http_body_stream_emit_end(env);
            return jayess_value_null();
        }
        if (chunk_remaining < 0) {
            char *line = jayess_http_body_stream_read_line(env);
            char *trimmed;
            char *end_ptr;
            unsigned long chunk_size;
            if (line == NULL) {
                free(buffer);
                jayess_std_stream_emit_error(env, "failed to read HTTP chunk size");
                return jayess_value_undefined();
            }
            trimmed = jayess_http_trim_copy(line);
            free(line);
            if (trimmed == NULL || trimmed[0] == '\0') {
                free(trimmed);
                continue;
            }
            {
                char *semi = strchr(trimmed, ';');
                if (semi != NULL) {
                    *semi = '\0';
                }
            }
            chunk_size = strtoul(trimmed, &end_ptr, 16);
            free(trimmed);
            if (end_ptr == NULL || *end_ptr != '\0') {
                free(buffer);
                jayess_std_stream_emit_error(env, "invalid HTTP chunk size");
                return jayess_value_undefined();
            }
            if (chunk_size == 0) {
                for (;;) {
                    char *trailer = jayess_http_body_stream_read_line(env);
                    if (trailer == NULL) {
                        free(buffer);
                        jayess_std_stream_emit_error(env, "failed to read HTTP chunk trailer");
                        return jayess_value_undefined();
                    }
                    if (trailer[0] == '\0') {
                        free(trailer);
                        break;
                    }
                    free(trailer);
                }
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_finished", jayess_value_from_bool(1));
                if (total == 0) {
                    free(buffer);
                    jayess_http_body_stream_emit_end(env);
                    return jayess_value_null();
                }
                break;
            }
            chunk_remaining = (long)chunk_size;
            jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number((double)chunk_remaining));
        }
        if (chunk_remaining > 0) {
            int need = requested - total;
            int take = (int)chunk_remaining;
            while (need > 0 && take > 0) {
                int read_target = need < take ? need : take;
                int read_count = jayess_http_body_stream_read_raw(env, buffer + total, read_target);
                if (read_count <= 0) {
                    free(buffer);
                    jayess_std_stream_emit_error(env, "HTTP chunk body ended unexpectedly");
                    return jayess_value_undefined();
                }
                total += read_count;
                need -= read_count;
                take -= read_count;
                chunk_remaining -= read_count;
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number((double)chunk_remaining));
                if (total >= requested) {
                    break;
                }
            }
            if (chunk_remaining == 0) {
                if (!jayess_http_body_stream_consume_crlf(env)) {
                    free(buffer);
                    jayess_std_stream_emit_error(env, "invalid HTTP chunk terminator");
                    return jayess_value_undefined();
                }
                jayess_object_set_value(env->as.object_value, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
            }
            if (total > 0) {
                break;
            }
        }
    }
    if (total == 0) {
        free(buffer);
        return jayess_value_null();
    }
    if (as_bytes) {
        jayess_value *result = jayess_std_uint8_array_from_bytes(buffer, (size_t)total);
        free(buffer);
        return result;
    }
    {
        jayess_value *result = jayess_http_body_stream_make_string(buffer, total);
        free(buffer);
        return result;
    }
}

static jayess_value *jayess_http_body_stream_read_chunk(jayess_value *env, jayess_value *size_value, int as_bytes) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "readableEnded"))) {
        return jayess_value_null();
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "__jayess_http_body_chunked"))) {
        return jayess_http_body_stream_read_chunked(env, size_value, as_bytes);
    }
    return jayess_http_body_stream_read_non_chunked(env, size_value, as_bytes);
}

static jayess_value *jayess_http_body_stream_new(jayess_socket_handle handle, const unsigned char *prebuffer, size_t prebuffer_len, jayess_object *headers) {
    jayess_object *object;
    long content_length;
    int chunked;
    if (handle == JAYESS_INVALID_SOCKET) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        jayess_std_socket_close_handle(handle);
        return jayess_value_undefined();
    }
    object->socket_handle = handle;
    chunked = jayess_http_headers_transfer_chunked(headers);
    content_length = jayess_http_headers_content_length(headers);
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("HttpBodyStream"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_chunked", jayess_value_from_bool(chunked));
    jayess_object_set_value(object, "__jayess_http_body_remaining", jayess_value_from_number(chunked ? -1 : (double)content_length));
    jayess_object_set_value(object, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
    jayess_object_set_value(object, "__jayess_http_body_chunk_finished", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer", jayess_std_uint8_array_from_bytes(prebuffer != NULL ? prebuffer : (const unsigned char *)"", prebuffer_len));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer_offset", jayess_value_from_number(0));
    if (!chunked && content_length == 0 && prebuffer_len == 0) {
        jayess_value *stream_value = jayess_value_from_object(object);
        jayess_http_body_stream_mark_ended(stream_value);
        jayess_http_body_stream_close_socket(stream_value);
        return stream_value;
    }
    return jayess_value_from_object(object);
}

static jayess_value *jayess_http_body_stream_new_from_socket(jayess_value *socket_value, const unsigned char *prebuffer, size_t prebuffer_len, jayess_object *headers) {
    jayess_value *stream_value = jayess_http_body_stream_new(jayess_std_socket_handle(socket_value), prebuffer, prebuffer_len, headers);
    if (stream_value != NULL && stream_value->kind == JAYESS_VALUE_OBJECT && stream_value->as.object_value != NULL) {
        jayess_object_set_value(stream_value->as.object_value, "__jayess_http_body_socket", socket_value != NULL ? socket_value : jayess_value_undefined());
    }
    return stream_value;
}

static int jayess_http_response_complete(const char *buffer, size_t length) {
    const char *header_end;
    const char *body_start;
    size_t header_bytes;
    size_t body_bytes;
    char *headers_text;
    jayess_object *headers;
    long content_length;
    int chunked;
    if (buffer == NULL || length == 0) {
        return 0;
    }
    header_end = jayess_http_header_boundary(buffer);
    if (header_end == NULL) {
        return 0;
    }
    body_start = (header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2;
    header_bytes = (size_t)(body_start - buffer);
    if (length < header_bytes) {
        return 0;
    }
    body_bytes = length - header_bytes;
    {
        const char *line_end = jayess_http_line_end(buffer);
        const char *header_start = jayess_http_next_line(line_end);
        headers_text = jayess_substring(header_start, 0, (size_t)(header_end - header_start));
    }
    headers = jayess_http_parse_header_object(headers_text);
    free(headers_text);
    chunked = jayess_http_headers_transfer_chunked(headers);
    if (chunked) {
        return jayess_http_chunked_body_complete(body_start, body_bytes);
    }
    content_length = jayess_http_headers_content_length(headers);
    if (content_length >= 0) {
        return body_bytes >= (size_t)content_length;
    }
    return 0;
}

static char *jayess_http_format_header_lines(jayess_object *headers) {
    jayess_object_entry *entry = headers != NULL ? headers->head : NULL;
    char *out = jayess_strdup("");
    while (entry != NULL) {
        char *value = jayess_value_stringify(entry->value);
        size_t current_len = strlen(out != NULL ? out : "");
        size_t key_len = strlen(entry->key != NULL ? entry->key : "");
        size_t value_len = strlen(value != NULL ? value : "");
        char *next = (char *)malloc(current_len + key_len + value_len + 5);
        if (next == NULL) {
            free(value);
            break;
        }
        sprintf(next, "%s%s: %s\r\n", out != NULL ? out : "", entry->key != NULL ? entry->key : "", value != NULL ? value : "");
        free(out);
        out = next;
        free(value);
        entry = entry->next;
    }
    return out;
}

static int jayess_http_socket_read_raw(jayess_value *socket_value, unsigned char *buffer, int max_count, int *did_timeout) {
    jayess_socket_handle handle;
    if (did_timeout != NULL) {
        *did_timeout = 0;
    }
    if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket_value, "Socket")) {
        return -1;
    }
    if (jayess_std_tls_state(socket_value) != NULL) {
        return jayess_std_tls_read_bytes(socket_value, buffer, max_count, did_timeout);
    }
    handle = jayess_std_socket_handle(socket_value);
    if (handle == JAYESS_INVALID_SOCKET) {
        return 0;
    }
    return (int)recv(handle, (char *)buffer, max_count, 0);
}

static char *jayess_http_read_all_socket_value(jayess_value *socket_value) {
    size_t capacity = 1024;
    size_t length = 0;
    char *buffer = (char *)malloc(capacity + 1);
    if (buffer == NULL) {
        return jayess_strdup("");
    }
    for (;;) {
        unsigned char chunk[1024];
        int read_count = jayess_http_socket_read_raw(socket_value, chunk, (int)sizeof(chunk), NULL);
        if (read_count <= 0) {
            break;
        }
        if (length + (size_t)read_count >= capacity) {
            size_t next_capacity = capacity;
            while (length + (size_t)read_count >= next_capacity) {
                next_capacity *= 2;
            }
            {
                char *next = (char *)realloc(buffer, next_capacity + 1);
                if (next == NULL) {
                    break;
                }
                buffer = next;
                capacity = next_capacity;
            }
        }
        memcpy(buffer + length, chunk, (size_t)read_count);
        length += (size_t)read_count;
        buffer[length] = '\0';
        if (jayess_http_response_complete(buffer, length)) {
            break;
        }
    }
    buffer[length] = '\0';
    return buffer;
}

static char *jayess_http_read_all_socket(jayess_socket_handle handle) {
    jayess_value *socket_value = jayess_std_socket_value_from_handle(handle, "", 0);
    if (socket_value == NULL) {
        return jayess_strdup("");
    }
    return jayess_http_read_all_socket_value(socket_value);
}

static jayess_value *jayess_http_read_response_stream_socket(jayess_value *socket_value) {
    size_t capacity = 1024;
    size_t length = 0;
    char *buffer = (char *)malloc(capacity + 1);
    const char *header_end;
    const char *line_end;
    const char *sp1;
    const char *sp2;
    const char *header_start;
    const char *body_start;
    char *version;
    char *status_text;
    char *reason;
    char *headers_text;
    jayess_object *headers;
    jayess_object *result;
    jayess_value *body_stream;
    double status_number;
    size_t body_len;
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    for (;;) {
        unsigned char chunk[1024];
        int read_count = jayess_http_socket_read_raw(socket_value, chunk, (int)sizeof(chunk), NULL);
        if (read_count <= 0) {
            free(buffer);
            return jayess_value_undefined();
        }
        if (length + (size_t)read_count >= capacity) {
            size_t next_capacity = capacity;
            while (length + (size_t)read_count >= next_capacity) {
                next_capacity *= 2;
            }
            {
                char *next = (char *)realloc(buffer, next_capacity + 1);
                if (next == NULL) {
                    free(buffer);
                    return jayess_value_undefined();
                }
                buffer = next;
                capacity = next_capacity;
            }
        }
        memcpy(buffer + length, chunk, (size_t)read_count);
        length += (size_t)read_count;
        buffer[length] = '\0';
        header_end = jayess_http_header_boundary(buffer);
        if (header_end != NULL) {
            break;
        }
    }
    header_end = jayess_http_header_boundary(buffer);
    if (header_end == NULL) {
        free(buffer);
        return jayess_value_undefined();
    }
    line_end = jayess_http_line_end(buffer);
    sp1 = buffer;
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(buffer);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    version = jayess_substring(buffer, 0, (size_t)(sp1 - buffer));
    status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
    reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
    header_start = jayess_http_next_line(line_end);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end - header_start));
    headers = jayess_http_parse_header_object(headers_text);
    body_start = (header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2;
    body_len = length >= (size_t)(body_start - buffer) ? length - (size_t)(body_start - buffer) : 0;
    body_stream = jayess_http_body_stream_new_from_socket(socket_value, (const unsigned char *)body_start, body_len, headers);
    status_number = atof(status_text != NULL ? status_text : "0");
    result = jayess_object_new();
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "status", jayess_value_from_number(status_number));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_number >= 200.0 && status_number < 300.0));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers));
    jayess_object_set_value(result, "bodyStream", body_stream);
    free(version);
    free(status_text);
    free(reason);
    free(headers_text);
    free(buffer);
    return jayess_value_from_object(result);
}

static jayess_value *jayess_http_read_response_stream(jayess_socket_handle handle) {
    jayess_value *socket_value = jayess_std_socket_value_from_handle(handle, "", 0);
    if (socket_value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_http_read_response_stream_socket(socket_value);
}

static void jayess_http_split_host_port(const char *input, char **host_out, int *port_out, int default_port) {
    const char *value = input != NULL ? input : "";
    const char *last_colon = strrchr(value, ':');
    if (host_out != NULL) {
        *host_out = NULL;
    }
    if (port_out != NULL) {
        *port_out = default_port;
    }
    if (last_colon != NULL && strchr(last_colon + 1, ':') == NULL) {
        char *host = jayess_substring(value, 0, (size_t)(last_colon - value));
        int port = atoi(last_colon + 1);
        if (host_out != NULL) {
            *host_out = host;
        } else {
            free(host);
        }
        if (port_out != NULL && port > 0) {
            *port_out = port;
        }
        return;
    }
    if (host_out != NULL) {
        *host_out = jayess_strdup(value);
    }
}

static jayess_object *jayess_http_request_object_from_url_value(jayess_value *input, const char *default_method) {
    jayess_value *parsed = jayess_std_url_parse(input);
    jayess_object *parsed_object = jayess_value_as_object(parsed);
    char *protocol = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "protocol")) : jayess_strdup("");
    char *host_raw = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "host")) : jayess_strdup("");
    char *host = NULL;
    int port = protocol != NULL && strcmp(protocol, "https:") == 0 ? 443 : 80;
    char *pathname = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "pathname")) : jayess_strdup("/");
    char *query = parsed_object != NULL ? jayess_value_stringify(jayess_object_get(parsed_object, "query")) : jayess_strdup("");
    char *full_path;
    jayess_object *request_object = jayess_object_new();
    size_t path_len;
    if (protocol != NULL && strcmp(protocol, "https:") == 0) {
        jayess_http_split_host_port(host_raw, &host, &port, 443);
    } else {
        jayess_http_split_host_port(host_raw, &host, &port, 80);
    }
    path_len = strlen(pathname != NULL && pathname[0] != '\0' ? pathname : "/") + strlen(query != NULL && query[0] != '\0' ? query : "") + 2;
    full_path = (char *)malloc(path_len);
    if (full_path == NULL) {
        free(protocol);
        free(host_raw);
        free(host);
        free(pathname);
        free(query);
        return NULL;
    }
    sprintf(full_path, "%s%s%s", pathname != NULL && pathname[0] != '\0' ? pathname : "/", query != NULL && query[0] != '\0' ? "?" : "", query != NULL ? query : "");
    jayess_object_set_value(request_object, "method", jayess_value_from_string(default_method != NULL && default_method[0] != '\0' ? default_method : "GET"));
    jayess_object_set_value(request_object, "path", jayess_value_from_string(full_path));
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string(protocol != NULL && strcmp(protocol, "https:") == 0 ? "https" : "http"));
    jayess_object_set_value(request_object, "version", jayess_value_from_string("HTTP/1.1"));
    jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
    jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
    jayess_object_set_value(request_object, "host", jayess_value_from_string(host != NULL ? host : ""));
    jayess_object_set_value(request_object, "port", jayess_value_from_number((double)port));
    free(protocol);
    free(host_raw);
    free(host);
    free(pathname);
    free(query);
    free(full_path);
    return request_object;
}

static void jayess_http_prepare_request_headers(jayess_object *request_object, const char *host_text, int port) {
    jayess_object *headers;
    char *body_text;
    char *host_header;
    char body_len_text[32];
    if (request_object == NULL) {
        return;
    }
    headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
    if (headers == NULL) {
        headers = jayess_object_new();
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(headers));
    }
    if (jayess_http_header_get_ci(headers, "Host") == NULL) {
        jayess_object_set_value(headers, "Host", jayess_value_from_string(host_text != NULL ? host_text : ""));
    }
    if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
        jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
    }
    if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
        body_text = jayess_value_stringify(jayess_object_get(request_object, "body"));
        if (body_text != NULL && body_text[0] != '\0') {
            snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)strlen(body_text));
            jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
        }
        free(body_text);
    }
}

static jayess_value *jayess_http_request_from_parts(jayess_object *request_object) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *request_text_value;
        char *request_text;
        char port_text[32];
        struct addrinfo hints;
        struct addrinfo *results = NULL;
        struct addrinfo *entry;
        jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
        int status;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;

        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }

        jayess_http_prepare_request_headers(request_object, host_text, port);
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);

        if (request_text == NULL) {
            free(host_text);
            return jayess_value_undefined();
        }

        snprintf(port_text, sizeof(port_text), "%d", port);
        memset(&hints, 0, sizeof(hints));
        hints.ai_family = AF_UNSPEC;
        hints.ai_socktype = SOCK_STREAM;
        status = getaddrinfo(host_text, port_text, &hints, &results);
        if (status != 0 || results == NULL) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }

        for (entry = results; entry != NULL; entry = entry->ai_next) {
            handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
            if (handle == JAYESS_INVALID_SOCKET) {
                continue;
            }
            if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
                break;
            }
            jayess_std_socket_close_handle(handle);
            handle = JAYESS_INVALID_SOCKET;
        }
        freeaddrinfo(results);
        if (handle == JAYESS_INVALID_SOCKET) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
            jayess_std_socket_close_handle(handle);
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }

        {
            size_t length = strlen(request_text);
            size_t offset = 0;
            while (offset < length) {
                int sent = (int)send(handle, request_text + offset, (int)(length - offset), 0);
                if (sent <= 0) {
                    jayess_std_socket_close_handle(handle);
                    free(request_text);
                    free(host_text);
                    return jayess_value_undefined();
                }
                offset += (size_t)sent;
            }
        }

#ifdef _WIN32
        shutdown(handle, SD_SEND);
#else
        shutdown(handle, SHUT_WR);
#endif
        {
            char *response_text = jayess_http_read_all_socket(handle);
            response = jayess_std_http_parse_response(jayess_value_from_string(response_text));
            free(response_text);
        }
        jayess_std_socket_close_handle(handle);
        free(request_text);
        free(host_text);

        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

static jayess_value *jayess_http_request_stream_from_parts(jayess_object *request_object) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *request_text_value;
        char *request_text;
        char port_text[32];
        struct addrinfo hints;
        struct addrinfo *results = NULL;
        struct addrinfo *entry;
        jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
        int status;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;

        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }

        jayess_http_prepare_request_headers(request_object, host_text, port);
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);
        if (request_text == NULL) {
            free(host_text);
            return jayess_value_undefined();
        }

        snprintf(port_text, sizeof(port_text), "%d", port);
        memset(&hints, 0, sizeof(hints));
        hints.ai_family = AF_UNSPEC;
        hints.ai_socktype = SOCK_STREAM;
        status = getaddrinfo(host_text, port_text, &hints, &results);
        if (status != 0 || results == NULL) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        for (entry = results; entry != NULL; entry = entry->ai_next) {
            handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
            if (handle == JAYESS_INVALID_SOCKET) {
                continue;
            }
            if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
                break;
            }
            jayess_std_socket_close_handle(handle);
            handle = JAYESS_INVALID_SOCKET;
        }
        freeaddrinfo(results);
        if (handle == JAYESS_INVALID_SOCKET) {
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        if (timeout > 0 && !jayess_std_socket_configure_timeout(handle, timeout)) {
            jayess_std_socket_close_handle(handle);
            free(request_text);
            free(host_text);
            return jayess_value_undefined();
        }
        {
            size_t length = strlen(request_text);
            size_t offset = 0;
            while (offset < length) {
                int sent = (int)send(handle, request_text + offset, (int)(length - offset), 0);
                if (sent <= 0) {
                    jayess_std_socket_close_handle(handle);
                    free(request_text);
                    free(host_text);
                    return jayess_value_undefined();
                }
                offset += (size_t)sent;
            }
        }
#ifdef _WIN32
        shutdown(handle, SD_SEND);
#else
        shutdown(handle, SHUT_WR);
#endif
        response = jayess_http_read_response_stream(handle);
        free(request_text);
        free(host_text);
        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            jayess_std_socket_close_handle(handle);
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        {
            jayess_value *body_stream = jayess_object_get(response_object, "bodyStream");
            if (body_stream != NULL && body_stream->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_stream, "HttpBodyStream")) {
                jayess_std_http_body_stream_close_method(body_stream);
            } else {
                jayess_std_socket_close_handle(handle);
            }
        }
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

static jayess_value *jayess_https_request_via_tls_from_parts(jayess_object *request_object, int stream_response) {
    int redirects = 0;
    int max_redirects = (int)jayess_value_to_number(jayess_object_get(request_object, "maxRedirects"));
    jayess_array *http11_alpn;
    if (max_redirects < 0) {
        max_redirects = 0;
    }
    if (max_redirects == 0 && jayess_object_get(request_object, "maxRedirects") == NULL) {
        max_redirects = 5;
    }
    http11_alpn = jayess_array_new();
    if (http11_alpn != NULL) {
        jayess_array_push_value(http11_alpn, jayess_value_from_string("http/1.1"));
    }
    for (;;) {
        char *host_text = jayess_value_stringify(jayess_object_get(request_object, "host"));
        int port = (int)jayess_value_to_number(jayess_object_get(request_object, "port"));
        int timeout = (int)jayess_value_to_number(jayess_object_get(request_object, "timeout"));
        jayess_value *reject_value = jayess_object_get(request_object, "rejectUnauthorized");
        jayess_object *tls_options;
        jayess_value *socket_value;
        jayess_value *request_text_value;
        char *request_text;
        jayess_value *response;
        jayess_object *response_object;
        int response_status;
        jayess_value *location_value;
        char *location_text;
        if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
            free(host_text);
            return jayess_value_undefined();
        }
        jayess_http_prepare_request_headers(request_object, host_text, port);
        tls_options = jayess_object_new();
        jayess_object_set_value(tls_options, "host", jayess_value_from_string(host_text));
        jayess_object_set_value(tls_options, "port", jayess_value_from_number((double)port));
        jayess_object_set_value(tls_options, "rejectUnauthorized", reject_value != NULL ? reject_value : jayess_value_from_bool(1));
        jayess_object_set_value(tls_options, "timeout", jayess_value_from_number((double)timeout));
        if (http11_alpn != NULL) {
            jayess_object_set_value(tls_options, "alpnProtocols", jayess_value_from_array(http11_alpn));
        }
        jayess_std_https_copy_tls_request_settings(tls_options, request_object);
        socket_value = jayess_std_tls_connect(jayess_value_from_object(tls_options));
        if (jayess_has_exception()) {
            free(host_text);
            return jayess_value_undefined();
        }
        if (socket_value == NULL || socket_value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(socket_value, "Socket")) {
            free(host_text);
            return jayess_value_undefined();
        }
        request_text_value = jayess_std_http_format_request(jayess_value_from_object(request_object));
        request_text = jayess_value_stringify(request_text_value);
        if (request_text == NULL) {
            jayess_std_socket_close_method(socket_value);
            free(host_text);
            return jayess_value_undefined();
        }
        if (!jayess_value_as_bool(jayess_std_socket_write_method(socket_value, jayess_value_from_string(request_text)))) {
            free(request_text);
            jayess_std_socket_close_method(socket_value);
            free(host_text);
            return jayess_value_undefined();
        }
        free(request_text);
        if (stream_response) {
            response = jayess_http_read_response_stream_socket(socket_value);
        } else {
            char *response_text = jayess_http_read_all_socket_value(socket_value);
            response = jayess_std_http_parse_response(jayess_value_from_string(response_text != NULL ? response_text : ""));
            free(response_text);
        }
        if (!stream_response) {
            jayess_std_socket_close_method(socket_value);
        }
        free(host_text);
        response_object = jayess_value_as_object(response);
        if (response_object == NULL) {
            return response;
        }
        response_status = (int)jayess_value_to_number(jayess_object_get(response_object, "status"));
        if (!jayess_http_is_redirect_status(response_status) || redirects >= max_redirects) {
            char *final_url = jayess_http_request_current_url(request_object);
            jayess_object_set_value(response_object, "redirected", jayess_value_from_bool(redirects > 0));
            jayess_object_set_value(response_object, "redirectCount", jayess_value_from_number((double)redirects));
            jayess_object_set_value(response_object, "url", jayess_value_from_string(final_url != NULL ? final_url : ""));
            free(final_url);
            return response;
        }
        location_value = jayess_http_header_get_ci(jayess_value_as_object(jayess_object_get(response_object, "headers")), "Location");
        location_text = jayess_value_stringify(location_value);
        if (stream_response) {
            jayess_value *body_stream = jayess_object_get(response_object, "bodyStream");
            if (body_stream != NULL && body_stream->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_stream, "HttpBodyStream")) {
                jayess_std_http_body_stream_close_method(body_stream);
            }
        }
        if (location_text == NULL || location_text[0] == '\0') {
            free(location_text);
            return response;
        }
        if (strncmp(location_text, "https://", 8) == 0 || strncmp(location_text, "http://", 7) == 0) {
            jayess_object *redirect_object = jayess_http_request_object_from_url_value(jayess_value_from_string(location_text), jayess_value_as_string(jayess_object_get(request_object, "method")));
            if (redirect_object == NULL) {
                free(location_text);
                return response;
            }
            jayess_object_set_value(redirect_object, "timeout", jayess_object_get(request_object, "timeout"));
            jayess_object_set_value(redirect_object, "rejectUnauthorized", jayess_object_get(request_object, "rejectUnauthorized"));
            jayess_std_https_copy_tls_request_settings(redirect_object, request_object);
            request_object = redirect_object;
        } else if (location_text[0] == '/') {
            jayess_object_set_value(request_object, "path", jayess_value_from_string(location_text));
        } else {
            free(location_text);
            return response;
        }
        if (response_status == 301 || response_status == 302 || response_status == 303) {
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
        }
        jayess_object_set_value(request_object, "headers", jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "maxRedirects", jayess_value_from_number((double)max_redirects));
        redirects++;
        free(location_text);
    }
}

jayess_value *jayess_std_querystring_parse(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *cursor = text != NULL ? text : "";
    jayess_object *object = jayess_object_new();
    while (*cursor != '\0') {
        const char *part_start = cursor;
        const char *part_end;
        const char *eq;
        char *key_raw;
        char *value_raw;
        char *key;
        char *value;
        while (*cursor != '\0' && *cursor != '&') {
            cursor++;
        }
        part_end = cursor;
        eq = part_start;
        while (eq < part_end && *eq != '=') {
            eq++;
        }
        key_raw = jayess_substring(part_start, 0, (size_t)(eq - part_start));
        value_raw = eq < part_end ? jayess_substring(eq + 1, 0, (size_t)(part_end - eq - 1)) : jayess_strdup("");
        key = jayess_percent_decode(key_raw);
        value = jayess_percent_decode(value_raw);
        if (key != NULL && key[0] != '\0') {
            jayess_object_set_value(object, key, jayess_value_from_string(value != NULL ? value : ""));
        }
        free(key_raw);
        free(value_raw);
        free(key);
        free(value);
        if (*cursor == '&') {
            cursor++;
        }
    }
    free(text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_querystring_stringify(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    jayess_object_entry *entry = object != NULL ? object->head : NULL;
    char *out = jayess_strdup("");
    size_t out_len = 0;
    int first = 1;
    while (entry != NULL) {
        char *key = jayess_percent_encode(entry->key);
        char *value_text = jayess_value_stringify(entry->value);
        char *value = jayess_percent_encode(value_text != NULL ? value_text : "");
        size_t key_len = strlen(key != NULL ? key : "");
        size_t value_len = strlen(value != NULL ? value : "");
        char *next = (char *)malloc(out_len + key_len + value_len + (first ? 2 : 3));
        if (next == NULL) {
            free(key);
            free(value_text);
            free(value);
            break;
        }
        sprintf(next, "%s%s%s=%s", out != NULL ? out : "", first ? "" : "&", key != NULL ? key : "", value != NULL ? value : "");
        free(out);
        out = next;
        out_len = strlen(out);
        first = 0;
        free(key);
        free(value_text);
        free(value);
        entry = entry->next;
    }
    {
        jayess_value *result = jayess_value_from_string(out != NULL ? out : "");
        free(out);
        return result;
    }
}

jayess_value *jayess_std_url_parse(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *scheme = strstr(value, "://");
    const char *after_scheme = scheme != NULL ? scheme + 3 : value;
    const char *path_start = strchr(after_scheme, '/');
    const char *query_start = strchr(after_scheme, '?');
    const char *hash_start = strchr(after_scheme, '#');
    const char *host_end = after_scheme + strlen(after_scheme);
    const char *path_end;
    const char *query_end;
    char *protocol;
    char *host;
    char *pathname;
    char *query;
    char *hash;
    jayess_object *object = jayess_object_new();
    if (path_start != NULL && path_start < host_end) {
        host_end = path_start;
    }
    if (query_start != NULL && query_start < host_end) {
        host_end = query_start;
    }
    if (hash_start != NULL && hash_start < host_end) {
        host_end = hash_start;
    }
    path_end = value + strlen(value);
    if (query_start != NULL && query_start < path_end) {
        path_end = query_start;
    }
    if (hash_start != NULL && hash_start < path_end) {
        path_end = hash_start;
    }
    query_end = hash_start != NULL ? hash_start : value + strlen(value);
    protocol = scheme != NULL ? jayess_substring(value, 0, (size_t)(scheme - value + 1)) : jayess_strdup("");
    host = jayess_substring(after_scheme, 0, (size_t)(host_end - after_scheme));
    pathname = path_start != NULL ? jayess_substring(path_start, 0, (size_t)(path_end - path_start)) : jayess_strdup("");
    query = query_start != NULL ? jayess_substring(query_start + 1, 0, (size_t)(query_end - query_start - 1)) : jayess_strdup("");
    hash = hash_start != NULL ? jayess_strdup(hash_start) : jayess_strdup("");
    jayess_object_set_value(object, "href", jayess_value_from_string(value));
    jayess_object_set_value(object, "protocol", jayess_value_from_string(protocol));
    jayess_object_set_value(object, "host", jayess_value_from_string(host));
    jayess_object_set_value(object, "pathname", jayess_value_from_string(pathname));
    jayess_object_set_value(object, "query", jayess_value_from_string(query));
    jayess_object_set_value(object, "hash", jayess_value_from_string(hash));
    jayess_object_set_value(object, "queryObject", jayess_std_querystring_parse(jayess_value_from_string(query)));
    free(protocol);
    free(host);
    free(pathname);
    free(query);
    free(hash);
    free(text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_url_format(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *protocol = object != NULL ? jayess_value_stringify(jayess_object_get(object, "protocol")) : jayess_strdup("");
    char *host = object != NULL ? jayess_value_stringify(jayess_object_get(object, "host")) : jayess_strdup("");
    char *pathname = object != NULL ? jayess_value_stringify(jayess_object_get(object, "pathname")) : jayess_strdup("");
    char *query = object != NULL ? jayess_value_stringify(jayess_object_get(object, "query")) : jayess_strdup("");
    char *hash = object != NULL ? jayess_value_stringify(jayess_object_get(object, "hash")) : jayess_strdup("");
    size_t len = strlen(protocol != NULL ? protocol : "") + strlen(host != NULL ? host : "") + strlen(pathname != NULL ? pathname : "") + strlen(query != NULL ? query : "") + strlen(hash != NULL ? hash : "") + 8;
    char *out = (char *)malloc(len);
    if (out == NULL) {
        return jayess_value_from_string("");
    }
    out[0] = '\0';
    if (protocol != NULL && protocol[0] != '\0') {
        strcat(out, protocol);
        if (strstr(protocol, "://") == NULL) {
            strcat(out, "//");
        }
    }
    strcat(out, host != NULL ? host : "");
    strcat(out, pathname != NULL ? pathname : "");
    if (query != NULL && query[0] != '\0') {
        strcat(out, "?");
        strcat(out, query);
    }
    if (hash != NULL && hash[0] != '\0') {
        if (hash[0] != '#') {
            strcat(out, "#");
        }
        strcat(out, hash);
    }
    {
        jayess_value *result = jayess_value_from_string(out);
        free(protocol);
        free(host);
        free(pathname);
        free(query);
        free(hash);
        free(out);
        return result;
    }
}

jayess_value *jayess_std_http_parse_request(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *line_end = jayess_http_line_end(value);
    const char *sp1 = value;
    const char *sp2;
    const char *header_start;
    const char *header_end;
    const char *body_start;
    char *method;
    char *path;
    char *version;
    char *headers_text;
    char *body;
    jayess_object *result;
    if (line_end == value) {
        free(text);
        return jayess_value_undefined();
    }
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    if (sp2 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    method = jayess_substring(value, 0, (size_t)(sp1 - value));
    path = jayess_substring(sp1 + 1, 0, (size_t)(sp2 - sp1 - 1));
    version = jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1));
    header_start = jayess_http_next_line(line_end);
    header_end = jayess_http_header_boundary(header_start);
    body_start = header_end != NULL ? ((header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2) : value + strlen(value);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end != NULL ? (size_t)(header_end - header_start) : 0));
    body = jayess_strdup(body_start != NULL ? body_start : "");
    result = jayess_object_new();
    jayess_object_set_value(result, "method", jayess_value_from_string(method));
    jayess_object_set_value(result, "path", jayess_value_from_string(path));
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "headers", jayess_value_from_object(jayess_http_parse_header_object(headers_text)));
    jayess_object_set_value(result, "body", jayess_value_from_string(body));
    free(method);
    free(path);
    free(version);
    free(headers_text);
    free(body);
    free(text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_http_format_request(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *method = object != NULL ? jayess_value_stringify(jayess_object_get(object, "method")) : jayess_strdup("GET");
    char *path = object != NULL ? jayess_value_stringify(jayess_object_get(object, "path")) : jayess_strdup("/");
    char *version = object != NULL ? jayess_value_stringify(jayess_object_get(object, "version")) : jayess_strdup("HTTP/1.1");
    jayess_object *headers = object != NULL ? jayess_value_as_object(jayess_object_get(object, "headers")) : NULL;
    char *headers_text = jayess_http_format_header_lines(headers);
    char *body = object != NULL ? jayess_value_stringify(jayess_object_get(object, "body")) : jayess_strdup("");
    size_t total = strlen(method != NULL ? method : "") + strlen(path != NULL ? path : "") + strlen(version != NULL ? version : "") + strlen(headers_text != NULL ? headers_text : "") + strlen(body != NULL ? body : "") + 8;
    char *out = (char *)malloc(total);
    jayess_value *result;
    if (out == NULL) {
        free(method);
        free(path);
        free(version);
        free(headers_text);
        free(body);
        return jayess_value_from_string("");
    }
    sprintf(out, "%s %s %s\r\n%s\r\n%s", method != NULL && method[0] != '\0' ? method : "GET", path != NULL && path[0] != '\0' ? path : "/", version != NULL && version[0] != '\0' ? version : "HTTP/1.1", headers_text != NULL ? headers_text : "", body != NULL ? body : "");
    result = jayess_value_from_string(out);
    free(method);
    free(path);
    free(version);
    free(headers_text);
    free(body);
    free(out);
    return result;
}

jayess_value *jayess_std_http_parse_response(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    const char *value = text != NULL ? text : "";
    const char *line_end = jayess_http_line_end(value);
    const char *sp1 = value;
    const char *sp2;
    const char *header_start;
    const char *header_end;
    const char *body_start;
    char *version;
    char *status_text;
    char *reason;
    char *headers_text;
    char *body;
    char *decoded_body;
    jayess_object *headers;
    jayess_object *result;
    double status_number;
    if (line_end == value) {
        free(text);
        return jayess_value_undefined();
    }
    while (sp1 < line_end && *sp1 != ' ') {
        sp1++;
    }
    if (sp1 >= line_end) {
        free(text);
        return jayess_value_undefined();
    }
    sp2 = sp1 + 1;
    while (sp2 < line_end && *sp2 != ' ') {
        sp2++;
    }
    version = jayess_substring(value, 0, (size_t)(sp1 - value));
    status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
    reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
    header_start = jayess_http_next_line(line_end);
    header_end = jayess_http_header_boundary(header_start);
    body_start = header_end != NULL ? ((header_end[0] == '\r' && header_end[1] == '\n') ? header_end + 4 : header_end + 2) : value + strlen(value);
    headers_text = jayess_substring(header_start, 0, (size_t)(header_end != NULL ? (size_t)(header_end - header_start) : 0));
    body = jayess_strdup(body_start != NULL ? body_start : "");
    headers = jayess_http_parse_header_object(headers_text);
    decoded_body = jayess_http_headers_transfer_chunked(headers) ? jayess_http_decode_chunked_body(body) : jayess_strdup(body != NULL ? body : "");
    status_number = atof(status_text != NULL ? status_text : "0");
    result = jayess_object_new();
    jayess_object_set_value(result, "version", jayess_value_from_string(version));
    jayess_object_set_value(result, "status", jayess_value_from_number(status_number));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_number >= 200.0 && status_number < 300.0));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers));
    jayess_object_set_value(result, "body", jayess_value_from_string(decoded_body != NULL ? decoded_body : ""));
    jayess_object_set_value(result, "bodyBytes", jayess_std_uint8_array_from_bytes((const unsigned char *)(decoded_body != NULL ? decoded_body : ""), decoded_body != NULL ? strlen(decoded_body) : 0));
    free(version);
    free(status_text);
    free(reason);
    free(headers_text);
    free(body);
    free(decoded_body);
    free(text);
    return jayess_value_from_object(result);
}

jayess_value *jayess_std_http_format_response(jayess_value *parts) {
    jayess_object *object = jayess_value_as_object(parts);
    char *version = object != NULL ? jayess_value_stringify(jayess_object_get(object, "version")) : jayess_strdup("HTTP/1.1");
    char *reason = object != NULL ? jayess_value_stringify(jayess_object_get(object, "reason")) : jayess_strdup("OK");
    char *status_text = object != NULL ? jayess_value_stringify(jayess_object_get(object, "status")) : jayess_strdup("200");
    jayess_object *headers = object != NULL ? jayess_value_as_object(jayess_object_get(object, "headers")) : NULL;
    char *headers_text = jayess_http_format_header_lines(headers);
    char *body = object != NULL ? jayess_value_stringify(jayess_object_get(object, "body")) : jayess_strdup("");
    size_t total = strlen(version != NULL ? version : "") + strlen(status_text != NULL ? status_text : "") + strlen(reason != NULL ? reason : "") + strlen(headers_text != NULL ? headers_text : "") + strlen(body != NULL ? body : "") + 8;
    char *out = (char *)malloc(total);
    jayess_value *result;
    if (out == NULL) {
        free(version);
        free(reason);
        free(status_text);
        free(headers_text);
        free(body);
        return jayess_value_from_string("");
    }
    sprintf(out, "%s %s %s\r\n%s\r\n%s", version != NULL && version[0] != '\0' ? version : "HTTP/1.1", status_text != NULL && status_text[0] != '\0' ? status_text : "200", reason != NULL ? reason : "", headers_text != NULL ? headers_text : "", body != NULL ? body : "");
    result = jayess_value_from_string(out);
    free(version);
    free(reason);
    free(status_text);
    free(headers_text);
    free(body);
    free(out);
    return result;
}

jayess_value *jayess_std_http_request(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(80));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_from_parts(request_object);
}

jayess_value *jayess_std_http_get(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_http_request_object_from_url_value(input, "GET");
    } else {
        jayess_object *input_object = jayess_value_as_object(input);
        if (input_object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_object_get(input_object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(input_object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(input_object, "url"), "GET");
            if (request_object != NULL) {
                if (jayess_object_get(input_object, "version") != NULL) {
                    jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version"));
                }
                if (jayess_object_get(input_object, "headers") != NULL) {
                    jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers"));
                }
                if (jayess_object_get(input_object, "host") != NULL) {
                    jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
                }
                if (jayess_object_get(input_object, "port") != NULL) {
                    jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port"));
                }
                if (jayess_object_get(input_object, "timeout") != NULL) {
                    jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout"));
                }
            }
        } else {
            request_object = jayess_object_new();
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "path", jayess_object_get(input_object, "path") != NULL ? jayess_object_get(input_object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version") != NULL ? jayess_object_get(input_object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers") != NULL ? jayess_object_get(input_object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port") != NULL ? jayess_object_get(input_object, "port") : jayess_value_from_number(80));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout") != NULL ? jayess_object_get(input_object, "timeout") : jayess_value_from_number(0));
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_from_parts(request_object);
}

jayess_value *jayess_std_http_request_stream(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(80));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_http_get_stream(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_http_request_object_from_url_value(input, "GET");
    } else {
        jayess_object *input_object = jayess_value_as_object(input);
        if (input_object == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_object_get(input_object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(input_object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(input_object, "url"), "GET");
            if (request_object != NULL) {
                if (jayess_object_get(input_object, "version") != NULL) {
                    jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version"));
                }
                if (jayess_object_get(input_object, "headers") != NULL) {
                    jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers"));
                }
                if (jayess_object_get(input_object, "host") != NULL) {
                    jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
                }
                if (jayess_object_get(input_object, "port") != NULL) {
                    jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port"));
                }
                if (jayess_object_get(input_object, "timeout") != NULL) {
                    jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout"));
                }
            }
        } else {
            request_object = jayess_object_new();
            jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
            jayess_object_set_value(request_object, "path", jayess_object_get(input_object, "path") != NULL ? jayess_object_get(input_object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "version", jayess_object_get(input_object, "version") != NULL ? jayess_object_get(input_object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(input_object, "headers") != NULL ? jayess_object_get(input_object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "host", jayess_object_get(input_object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(input_object, "port") != NULL ? jayess_object_get(input_object, "port") : jayess_value_from_number(80));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(input_object, "timeout") != NULL ? jayess_object_get(input_object, "timeout") : jayess_value_from_number(0));
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_http_text_eq_ci(jayess_value_as_string(jayess_object_get(request_object, "scheme")), "https")) {
        jayess_throw(jayess_type_error_value("https URLs must use https.request(...) or https.get(...)"));
        return jayess_value_undefined();
    }
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_object_get(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_object_get(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
    return jayess_http_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_http_request_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 0, 0);
    return promise;
}

jayess_value *jayess_std_http_get_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 0, 0);
    return promise;
}

jayess_value *jayess_std_http_request_stream_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 0, 1);
    return promise;
}

jayess_value *jayess_std_http_get_stream_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 0, 1);
    return promise;
}

static unsigned char *jayess_http_request_body_bytes(jayess_value *body_value, size_t *length_out) {
    unsigned char *buffer = NULL;
    size_t length = 0;
    if (length_out != NULL) {
        *length_out = 0;
    }
    if (body_value == NULL || jayess_value_is_nullish(body_value)) {
        return NULL;
    }
    if (body_value != NULL && body_value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(body_value, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(body_value);
        int i;
        if (bytes == NULL || bytes->count <= 0) {
            return NULL;
        }
        length = (size_t)bytes->count;
        buffer = (unsigned char *)malloc(length);
        if (buffer == NULL) {
            return NULL;
        }
        for (i = 0; i < bytes->count; i++) {
            buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
        }
        if (length_out != NULL) {
            *length_out = length;
        }
        return buffer;
    }
    {
        char *text = jayess_value_stringify(body_value);
        if (text == NULL) {
            return NULL;
        }
        length = strlen(text);
        if (length == 0) {
            free(text);
            return NULL;
        }
        buffer = (unsigned char *)malloc(length);
        if (buffer == NULL) {
            free(text);
            return NULL;
        }
        memcpy(buffer, text, length);
        free(text);
        if (length_out != NULL) {
            *length_out = length;
        }
        return buffer;
    }
}

#ifdef _WIN32
static jayess_value *jayess_http_body_stream_new_winhttp(HINTERNET request, HINTERNET connection, HINTERNET session, jayess_object *headers) {
    jayess_object *object;
    jayess_winhttp_stream_state *state;
    long content_length;
    if (request == NULL || connection == NULL || session == NULL) {
        if (request != NULL) {
            WinHttpCloseHandle(request);
        }
        if (connection != NULL) {
            WinHttpCloseHandle(connection);
        }
        if (session != NULL) {
            WinHttpCloseHandle(session);
        }
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        WinHttpCloseHandle(request);
        WinHttpCloseHandle(connection);
        WinHttpCloseHandle(session);
        return jayess_value_undefined();
    }
    state = (jayess_winhttp_stream_state *)malloc(sizeof(jayess_winhttp_stream_state));
    if (state == NULL) {
        WinHttpCloseHandle(request);
        WinHttpCloseHandle(connection);
        WinHttpCloseHandle(session);
        return jayess_value_undefined();
    }
    state->request = request;
    state->connection = connection;
    state->session = session;
    object->native_handle = state;
    content_length = jayess_http_headers_content_length(headers);
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("HttpBodyStream"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_chunked", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_remaining", jayess_value_from_number((double)content_length));
    jayess_object_set_value(object, "__jayess_http_body_chunk_remaining", jayess_value_from_number(-1));
    jayess_object_set_value(object, "__jayess_http_body_chunk_finished", jayess_value_from_bool(0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer", jayess_std_uint8_array_from_bytes((const unsigned char *)"", 0));
    jayess_object_set_value(object, "__jayess_http_body_prebuffer_offset", jayess_value_from_number(0));
    if (content_length == 0) {
        jayess_value *stream_value = jayess_value_from_object(object);
        jayess_http_body_stream_mark_ended(stream_value);
        jayess_http_body_stream_close_native(stream_value);
        return stream_value;
    }
    return jayess_value_from_object(object);
}

static jayess_value *jayess_https_read_response_stream(HINTERNET request, HINTERNET connection, HINTERNET session) {
    DWORD header_bytes = 0;
    wchar_t *raw_headers_w = NULL;
    char *raw_headers = NULL;
    char *version = NULL;
    char *status_text = NULL;
    char *reason = NULL;
    char *header_lines = NULL;
    jayess_object *headers = NULL;
    jayess_object *result = NULL;
    DWORD status_code = 0;
    DWORD status_size = sizeof(status_code);

    WinHttpQueryHeaders(request, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, WINHTTP_NO_OUTPUT_BUFFER, &header_bytes, WINHTTP_NO_HEADER_INDEX);
    if (GetLastError() != ERROR_INSUFFICIENT_BUFFER || header_bytes == 0) {
        goto cleanup;
    }
    raw_headers_w = (wchar_t *)malloc((size_t)header_bytes);
    if (raw_headers_w == NULL) {
        goto cleanup;
    }
    if (!WinHttpQueryHeaders(request, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, raw_headers_w, &header_bytes, WINHTTP_NO_HEADER_INDEX)) {
        goto cleanup;
    }
    raw_headers = jayess_wide_to_utf8(raw_headers_w);
    if (raw_headers == NULL) {
        goto cleanup;
    }
    {
        const char *line_end = jayess_http_line_end(raw_headers);
        const char *sp1 = raw_headers;
        const char *sp2;
        const char *header_start;
        while (sp1 < line_end && *sp1 != ' ') {
            sp1++;
        }
        if (sp1 >= line_end) {
            goto cleanup;
        }
        sp2 = sp1 + 1;
        while (sp2 < line_end && *sp2 != ' ') {
            sp2++;
        }
        version = jayess_substring(raw_headers, 0, (size_t)(sp1 - raw_headers));
        status_text = jayess_substring(sp1 + 1, 0, (size_t)((sp2 < line_end ? sp2 : line_end) - sp1 - 1));
        reason = sp2 < line_end ? jayess_substring(sp2 + 1, 0, (size_t)(line_end - sp2 - 1)) : jayess_strdup("");
        header_start = jayess_http_next_line(line_end);
        header_lines = jayess_substring(header_start, 0, strlen(header_start));
    }
    headers = jayess_http_parse_header_object(header_lines != NULL ? header_lines : "");
    if (!WinHttpQueryHeaders(request, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER, WINHTTP_HEADER_NAME_BY_INDEX, &status_code, &status_size, WINHTTP_NO_HEADER_INDEX)) {
        status_code = (DWORD)atoi(status_text != NULL ? status_text : "0");
    }
    result = jayess_object_new();
    if (result == NULL) {
        goto cleanup;
    }
    jayess_object_set_value(result, "version", jayess_value_from_string(version != NULL ? version : "HTTP/1.1"));
    jayess_object_set_value(result, "status", jayess_value_from_number((double)status_code));
    jayess_object_set_value(result, "reason", jayess_value_from_string(reason != NULL ? reason : ""));
    jayess_object_set_value(result, "statusText", jayess_value_from_string(reason != NULL ? reason : ""));
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status_code >= 200 && status_code < 300));
    jayess_object_set_value(result, "headers", jayess_value_from_object(headers != NULL ? headers : jayess_object_new()));
    jayess_object_set_value(result, "bodyStream", jayess_http_body_stream_new_winhttp(request, connection, session, headers));
    request = NULL;
    connection = NULL;
    session = NULL;
    free(raw_headers_w);
    free(raw_headers);
    free(version);
    free(status_text);
    free(reason);
    free(header_lines);
    return jayess_value_from_object(result);

cleanup:
    if (request != NULL) {
        WinHttpCloseHandle(request);
    }
    if (connection != NULL) {
        WinHttpCloseHandle(connection);
    }
    if (session != NULL) {
        WinHttpCloseHandle(session);
    }
    free(raw_headers_w);
    free(raw_headers);
    free(version);
    free(status_text);
    free(reason);
    free(header_lines);
    return jayess_value_undefined();
}

static jayess_value *jayess_https_request_stream_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 1);
}

static jayess_value *jayess_https_request_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 0);
}
#else
static jayess_value *jayess_https_request_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 0);
}
#endif

#ifndef _WIN32
static jayess_value *jayess_https_request_stream_from_parts(jayess_object *request_object) {
    return jayess_https_request_via_tls_from_parts(request_object, 1);
}
#endif

jayess_value *jayess_std_https_request(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    {
        const char *method = jayess_value_as_string(jayess_object_get(object, "method"));
        size_t body_length = 0;
        unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(object, "body"), &body_length);
        free(body_bytes);
        if ((method == NULL || method[0] == '\0' || jayess_http_text_eq_ci(method, "GET")) && body_length == 0) {
            return jayess_std_https_get(options);
        }
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
            if (jayess_object_get(object, "maxRedirects") != NULL) {
                jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
            }
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
        jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
        jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
        jayess_std_https_copy_tls_request_settings(request_object, object);
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
        if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
            size_t body_len = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(request_object, "body"), &body_len);
            if (body_bytes != NULL || body_len > 0) {
                char body_len_text[32];
                snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)body_len);
                jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
            }
            free(body_bytes);
        }
    }
    return jayess_https_request_from_parts(request_object);
}

jayess_value *jayess_std_https_request_stream(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_object *request_object = NULL;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
        request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), jayess_value_as_string(jayess_object_get(object, "method")) != NULL ? jayess_value_as_string(jayess_object_get(object, "method")) : "GET");
        if (request_object != NULL) {
            if (jayess_object_get(object, "version") != NULL) {
                jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
            }
            if (jayess_object_get(object, "headers") != NULL) {
                jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
            }
            if (jayess_object_get(object, "body") != NULL) {
                jayess_object_set_value(request_object, "body", jayess_object_get(object, "body"));
            }
            if (jayess_object_get(object, "host") != NULL) {
                jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            }
            if (jayess_object_get(object, "port") != NULL) {
                jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
            }
            if (jayess_object_get(object, "timeout") != NULL) {
                jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
            }
            if (jayess_object_get(object, "maxRedirects") != NULL) {
                jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
            }
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    } else {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_object_get(object, "method") != NULL ? jayess_object_get(object, "method") : jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
        jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
        jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
        jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
        jayess_object_set_value(request_object, "body", jayess_object_get(object, "body") != NULL ? jayess_object_get(object, "body") : jayess_value_from_string(""));
        jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
        jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
        jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
        jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
        jayess_std_https_copy_tls_request_settings(request_object, object);
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
        if (jayess_http_header_get_ci(headers, "Content-Length") == NULL && !jayess_http_headers_transfer_chunked(headers)) {
            size_t body_len = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(request_object, "body"), &body_len);
            if (body_bytes != NULL || body_len > 0) {
                char body_len_text[32];
                snprintf(body_len_text, sizeof(body_len_text), "%u", (unsigned int)body_len);
                jayess_object_set_value(headers, "Content-Length", jayess_value_from_string(body_len_text));
            }
            free(body_bytes);
        }
    }
    return jayess_https_request_stream_from_parts(request_object);
}

jayess_value *jayess_std_https_get(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "url", input);
    }
    if (request_object == NULL) {
        jayess_object *object = jayess_value_as_object(input);
        if (object == NULL) {
            return jayess_value_undefined();
        }
        {
            const char *method = jayess_value_as_string(jayess_object_get(object, "method"));
            size_t body_length = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(object, "body"), &body_length);
            free(body_bytes);
            if ((method != NULL && method[0] != '\0' && !jayess_http_text_eq_ci(method, "GET")) || body_length > 0) {
                jayess_throw(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
                return jayess_value_undefined();
            }
        }
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), "GET");
        } else {
            jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
            jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
            jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
        if (request_object != NULL && jayess_object_get(object, "headers") != NULL) {
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
        }
        if (request_object != NULL && jayess_object_get(object, "version") != NULL) {
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
        }
        if (request_object != NULL && jayess_object_get(object, "host") != NULL) {
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        }
        if (request_object != NULL && jayess_object_get(object, "port") != NULL) {
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
        }
        if (request_object != NULL && jayess_object_get(object, "timeout") != NULL) {
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
        }
        if (request_object != NULL && jayess_object_get(object, "maxRedirects") != NULL) {
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
        }
        if (request_object != NULL) {
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
    }
    return jayess_https_request_from_parts(request_object);
}

jayess_value *jayess_std_https_get_stream(jayess_value *input) {
    jayess_object *request_object = NULL;
    if (input != NULL && input->kind == JAYESS_VALUE_STRING) {
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        jayess_object_set_value(request_object, "url", input);
    }
    if (request_object == NULL) {
        jayess_object *object = jayess_value_as_object(input);
        if (object == NULL) {
            return jayess_value_undefined();
        }
        {
            const char *method = jayess_value_as_string(jayess_object_get(object, "method"));
            size_t body_length = 0;
            unsigned char *body_bytes = jayess_http_request_body_bytes(jayess_object_get(object, "body"), &body_length);
            free(body_bytes);
            if ((method != NULL && method[0] != '\0' && !jayess_http_text_eq_ci(method, "GET")) || body_length > 0) {
                jayess_throw(jayess_type_error_value("HTTPS request bodies and custom methods are not supported yet"));
                return jayess_value_undefined();
            }
        }
        request_object = jayess_object_new();
        jayess_object_set_value(request_object, "method", jayess_value_from_string("GET"));
        if (jayess_object_get(object, "url") != NULL && !jayess_value_is_nullish(jayess_object_get(object, "url"))) {
            request_object = jayess_http_request_object_from_url_value(jayess_object_get(object, "url"), "GET");
        } else {
            jayess_object_set_value(request_object, "path", jayess_object_get(object, "path") != NULL ? jayess_object_get(object, "path") : jayess_value_from_string("/"));
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port") != NULL ? jayess_object_get(object, "port") : jayess_value_from_number(443));
            jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version") != NULL ? jayess_object_get(object, "version") : jayess_value_from_string("HTTP/1.1"));
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers") != NULL ? jayess_object_get(object, "headers") : jayess_value_from_object(jayess_object_new()));
            jayess_object_set_value(request_object, "body", jayess_value_from_string(""));
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout") != NULL ? jayess_object_get(object, "timeout") : jayess_value_from_number(0));
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects") != NULL ? jayess_object_get(object, "maxRedirects") : jayess_value_from_number(5));
            jayess_object_set_value(request_object, "rejectUnauthorized", jayess_object_get(object, "rejectUnauthorized") != NULL ? jayess_object_get(object, "rejectUnauthorized") : jayess_value_from_bool(1));
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
        if (request_object != NULL && jayess_object_get(object, "headers") != NULL) {
            jayess_object_set_value(request_object, "headers", jayess_object_get(object, "headers"));
        }
        if (request_object != NULL && jayess_object_get(object, "version") != NULL) {
            jayess_object_set_value(request_object, "version", jayess_object_get(object, "version"));
        }
        if (request_object != NULL && jayess_object_get(object, "host") != NULL) {
            jayess_object_set_value(request_object, "host", jayess_object_get(object, "host"));
        }
        if (request_object != NULL && jayess_object_get(object, "port") != NULL) {
            jayess_object_set_value(request_object, "port", jayess_object_get(object, "port"));
        }
        if (request_object != NULL && jayess_object_get(object, "timeout") != NULL) {
            jayess_object_set_value(request_object, "timeout", jayess_object_get(object, "timeout"));
        }
        if (request_object != NULL && jayess_object_get(object, "maxRedirects") != NULL) {
            jayess_object_set_value(request_object, "maxRedirects", jayess_object_get(object, "maxRedirects"));
        }
        if (request_object != NULL) {
            jayess_std_https_copy_tls_request_settings(request_object, object);
        }
    }
    if (request_object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(request_object, "scheme", jayess_value_from_string("https"));
    if (jayess_value_as_object(jayess_object_get(request_object, "headers")) != NULL) {
        jayess_object *headers = jayess_value_as_object(jayess_object_get(request_object, "headers"));
        if (jayess_http_header_get_ci(headers, "Host") == NULL && jayess_object_get(request_object, "host") != NULL) {
            jayess_object_set_value(headers, "Host", jayess_object_get(request_object, "host"));
        }
        if (jayess_http_header_get_ci(headers, "Connection") == NULL) {
            jayess_object_set_value(headers, "Connection", jayess_value_from_string("close"));
        }
    }
#ifdef _WIN32
    return jayess_https_request_stream_from_parts(request_object);
#else
    jayess_throw(jayess_type_error_value("HTTPS streaming is not available on this platform"));
    return jayess_value_undefined();
#endif
}

jayess_value *jayess_std_https_request_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 1, 0);
    return promise;
}

jayess_value *jayess_std_https_get_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 1, 0);
    return promise;
}

jayess_value *jayess_std_https_request_stream_async(jayess_value *options) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, options, 0, 1, 1);
    return promise;
}

jayess_value *jayess_std_https_get_stream_async(jayess_value *input) {
    jayess_value *promise = jayess_std_promise_pending();
    jayess_enqueue_http_request_task(promise, input, 1, 1, 1);
    return promise;
}

static int jayess_std_socket_runtime_ready(void) {
#ifdef _WIN32
    static int winsock_initialized = 0;
    if (!winsock_initialized) {
        WSADATA data;
        if (WSAStartup(MAKEWORD(2, 2), &data) != 0) {
            return 0;
        }
        winsock_initialized = 1;
    }
#endif
    return 1;
}

jayess_value *jayess_std_dns_lookup(jayess_value *host) {
    char *host_text = jayess_value_stringify(host);
    const char *lookup_host = host_text != NULL ? host_text : "";
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    char address[INET6_ADDRSTRLEN];
    int family = 0;
    int status;
    jayess_object *object;

    if (lookup_host[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    if (!jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(lookup_host, NULL, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    address[0] = '\0';
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        void *addr = NULL;
        if (entry->ai_family == AF_INET) {
            struct sockaddr_in *ipv4 = (struct sockaddr_in *)entry->ai_addr;
            addr = &(ipv4->sin_addr);
            family = 4;
        } else if (entry->ai_family == AF_INET6) {
            struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)entry->ai_addr;
            addr = &(ipv6->sin6_addr);
            family = 6;
        }
        if (addr != NULL && inet_ntop(entry->ai_family, addr, address, sizeof(address)) != NULL) {
            break;
        }
    }

    freeaddrinfo(results);
    if (address[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    object = jayess_object_new();
    jayess_object_set_value(object, "host", jayess_value_from_string(lookup_host));
    jayess_object_set_value(object, "address", jayess_value_from_string(address));
    jayess_object_set_value(object, "family", jayess_value_from_number((double)family));
    free(host_text);
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_dns_lookup_all(jayess_value *host) {
    char *host_text = jayess_value_stringify(host);
    const char *lookup_host = host_text != NULL ? host_text : "";
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_array *records;
    int status;

    if (lookup_host[0] == '\0') {
        free(host_text);
        return jayess_value_undefined();
    }

    if (!jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(lookup_host, NULL, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    records = jayess_array_new();
    for (entry = results; entry != NULL; entry = entry->ai_next) {
        char address[INET6_ADDRSTRLEN];
        void *addr = NULL;
        int family = 0;
        jayess_object *record;
        if (entry->ai_family == AF_INET) {
            struct sockaddr_in *ipv4 = (struct sockaddr_in *)entry->ai_addr;
            addr = &(ipv4->sin_addr);
            family = 4;
        } else if (entry->ai_family == AF_INET6) {
            struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)entry->ai_addr;
            addr = &(ipv6->sin6_addr);
            family = 6;
        }
        if (addr == NULL || inet_ntop(entry->ai_family, addr, address, sizeof(address)) == NULL) {
            continue;
        }
        record = jayess_object_new();
        jayess_object_set_value(record, "host", jayess_value_from_string(lookup_host));
        jayess_object_set_value(record, "address", jayess_value_from_string(address));
        jayess_object_set_value(record, "family", jayess_value_from_number((double)family));
        jayess_array_push_value(records, jayess_value_from_object(record));
    }

    freeaddrinfo(results);
    free(host_text);
    if (records->count == 0) {
        return jayess_value_undefined();
    }
    return jayess_value_from_array(records);
}

jayess_value *jayess_std_dns_reverse(jayess_value *address) {
    char *address_text = jayess_value_stringify(address);
    const char *lookup_address = address_text != NULL ? address_text : "";
    char host[NI_MAXHOST];
    unsigned char buffer[sizeof(struct in6_addr)];
    int status;

    if (lookup_address[0] == '\0' || !jayess_std_socket_runtime_ready()) {
        free(address_text);
        return jayess_value_undefined();
    }

    host[0] = '\0';
    if (inet_pton(AF_INET, lookup_address, buffer) == 1) {
        struct sockaddr_in addr;
        memset(&addr, 0, sizeof(addr));
        addr.sin_family = AF_INET;
        memcpy(&addr.sin_addr, buffer, sizeof(struct in_addr));
        status = getnameinfo((struct sockaddr *)&addr, sizeof(addr), host, sizeof(host), NULL, 0, 0);
    } else if (inet_pton(AF_INET6, lookup_address, buffer) == 1) {
        struct sockaddr_in6 addr;
        memset(&addr, 0, sizeof(addr));
        addr.sin6_family = AF_INET6;
        memcpy(&addr.sin6_addr, buffer, sizeof(struct in6_addr));
        status = getnameinfo((struct sockaddr *)&addr, sizeof(addr), host, sizeof(host), NULL, 0, 0);
    } else {
        free(address_text);
        return jayess_value_undefined();
    }

    free(address_text);
    if (status != 0 || host[0] == '\0') {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(host);
}

jayess_value *jayess_std_net_is_ip(jayess_value *input) {
    char *text = jayess_value_stringify(input);
    unsigned char buffer[sizeof(struct in6_addr)];
    int family = 0;

    if (text != NULL && inet_pton(AF_INET, text, buffer) == 1) {
        family = 4;
    } else if (text != NULL && inet_pton(AF_INET6, text, buffer) == 1) {
        family = 6;
    }

    free(text);
    return jayess_value_from_number((double)family);
}

jayess_value *jayess_std_net_connect(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;

    if (host_text == NULL || host_text[0] == '\0' || port <= 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
        if (connect(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }

    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(host_text);
        return jayess_value_undefined();
    }

    {
        jayess_value *result = jayess_std_socket_value_from_handle(handle, host_text, port);
        jayess_std_socket_set_remote_family(result, family);
        jayess_std_socket_set_local_endpoint(result, handle);
        free(host_text);
        return result;
    }
}

jayess_value *jayess_std_net_listen(jayess_value *options) {
    jayess_object *object_options = jayess_value_as_object(options);
    jayess_value *host_value = object_options != NULL ? jayess_object_get(object_options, "host") : NULL;
    jayess_value *port_value = object_options != NULL ? jayess_object_get(object_options, "port") : NULL;
    char *host_text = jayess_value_stringify(host_value);
    int port = (int)jayess_value_to_number(port_value);
    char port_text[32];
    struct addrinfo hints;
    struct addrinfo *results = NULL;
    struct addrinfo *entry;
    jayess_socket_handle handle = JAYESS_INVALID_SOCKET;
    int family = 0;
    int status;
    jayess_object *server_object;
    int yes = 1;

    if (host_text == NULL || host_text[0] == '\0' || port < 0 || !jayess_std_socket_runtime_ready()) {
        free(host_text);
        return jayess_value_undefined();
    }

    snprintf(port_text, sizeof(port_text), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_flags = AI_PASSIVE;
    status = getaddrinfo(host_text, port_text, &hints, &results);
    if (status != 0 || results == NULL) {
        free(host_text);
        return jayess_value_undefined();
    }

    for (entry = results; entry != NULL; entry = entry->ai_next) {
        handle = socket(entry->ai_family, entry->ai_socktype, entry->ai_protocol);
        if (handle == JAYESS_INVALID_SOCKET) {
            continue;
        }
#ifdef _WIN32
        setsockopt(handle, SOL_SOCKET, SO_REUSEADDR, (const char *)&yes, sizeof(yes));
#else
        setsockopt(handle, SOL_SOCKET, SO_REUSEADDR, &yes, sizeof(yes));
#endif
        if (bind(handle, entry->ai_addr, (int)entry->ai_addrlen) == 0 && listen(handle, 16) == 0) {
            family = entry->ai_family == AF_INET6 ? 6 : 4;
            break;
        }
        jayess_std_socket_close_handle(handle);
        handle = JAYESS_INVALID_SOCKET;
    }
    freeaddrinfo(results);
    if (handle == JAYESS_INVALID_SOCKET) {
        free(host_text);
        return jayess_value_undefined();
    }

    if (port == 0) {
        struct sockaddr_storage local_addr;
#ifdef _WIN32
        int local_len = sizeof(local_addr);
#else
        socklen_t local_len = sizeof(local_addr);
#endif
        memset(&local_addr, 0, sizeof(local_addr));
        if (getsockname(handle, (struct sockaddr *)&local_addr, &local_len) == 0) {
            if (local_addr.ss_family == AF_INET) {
                port = ntohs(((struct sockaddr_in *)&local_addr)->sin_port);
            } else if (local_addr.ss_family == AF_INET6) {
                port = ntohs(((struct sockaddr_in6 *)&local_addr)->sin6_port);
            }
        }
    }

    server_object = jayess_object_new();
    if (server_object == NULL) {
        jayess_std_socket_close_handle(handle);
        free(host_text);
        return jayess_value_from_object(NULL);
    }
    server_object->socket_handle = handle;
    jayess_object_set_value(server_object, "__jayess_std_kind", jayess_value_from_string("Server"));
    jayess_object_set_value(server_object, "listening", jayess_value_from_bool(1));
    jayess_object_set_value(server_object, "closed", jayess_value_from_bool(0));
    jayess_object_set_value(server_object, "host", jayess_value_from_string(host_text));
    jayess_object_set_value(server_object, "port", jayess_value_from_number((double)port));
    jayess_object_set_value(server_object, "family", jayess_value_from_number((double)family));
    jayess_object_set_value(server_object, "timeout", jayess_value_from_number(0));
    jayess_object_set_value(server_object, "connectionsAccepted", jayess_value_from_number(0));
    jayess_object_set_value(server_object, "errored", jayess_value_from_bool(0));
    free(host_text);
    return jayess_value_from_object(server_object);
}

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
            current = jayess_path_join_segments_with_root(root, built);
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
            char *key = jayess_value_stringify(jayess_array_get(entry->as.array_value, 0));
            jayess_value *value = jayess_array_get(entry->as.array_value, 1);
            jayess_object_set_value(object, key != NULL ? key : "", value);
            free(key);
        }
    }
    return jayess_value_from_object(object);
}

void jayess_value_set_computed_member(jayess_value *target, jayess_value *key, jayess_value *value) {
    char *key_text;
    if (target == NULL || key == NULL || value == NULL) {
        return;
    }
    key_text = jayess_value_stringify(key);
    if (key_text == NULL) {
        return;
    }
    jayess_value_set_member(target, key_text, value);
    free(key_text);
}

jayess_array *jayess_array_new(void) {
    jayess_array *array = (jayess_array *)malloc(sizeof(jayess_array));
    if (array == NULL) {
        return NULL;
    }
    array->count = 0;
    array->values = NULL;
    return array;
}

static int jayess_array_ensure(jayess_array *array, int index) {
    int i;
    jayess_value **values;

    if (array == NULL || index < 0) {
        return 0;
    }
    if (index < array->count) {
        return 1;
    }

    values = (jayess_value **)realloc(array->values, sizeof(jayess_value *) * (size_t)(index + 1));
    if (values == NULL) {
        return 0;
    }
    for (i = array->count; i <= index; i++) {
        values[i] = NULL;
    }
    array->values = values;
    array->count = index + 1;
    return 1;
}

void jayess_array_set_value(jayess_array *array, int index, jayess_value *value) {
    if (!jayess_array_ensure(array, index)) {
        return;
    }
    array->values[index] = value;
}

jayess_value *jayess_array_get(jayess_array *array, int index) {
    if (array == NULL || index < 0 || index >= array->count) {
        return NULL;
    }
    return array->values[index];
}

int jayess_array_length(jayess_array *array) {
    if (array == NULL) {
        return 0;
    }
    return array->count;
}

int jayess_array_push_value(jayess_array *array, jayess_value *value) {
    if (array == NULL) {
        return 0;
    }
    jayess_array_set_value(array, array->count, value);
    return array->count;
}

jayess_value *jayess_array_pop_value(jayess_array *array) {
    jayess_value *value;

    if (array == NULL || array->count == 0) {
        return jayess_value_undefined();
    }

    value = array->values[array->count - 1];
    array->count--;
    if (array->count == 0) {
        free(array->values);
        array->values = NULL;
    } else {
        jayess_value **shrunk = (jayess_value **)realloc(array->values, sizeof(jayess_value *) * (size_t)array->count);
        if (shrunk != NULL) {
            array->values = shrunk;
        }
    }
    return value != NULL ? value : jayess_value_undefined();
}

jayess_value *jayess_array_shift_value(jayess_array *array) {
    int i;
    jayess_value *value;

    if (array == NULL || array->count == 0) {
        return jayess_value_undefined();
    }
    value = array->values[0];
    for (i = 1; i < array->count; i++) {
        array->values[i - 1] = array->values[i];
    }
    array->count--;
    if (array->count == 0) {
        free(array->values);
        array->values = NULL;
    } else {
        jayess_value **shrunk = (jayess_value **)realloc(array->values, sizeof(jayess_value *) * (size_t)array->count);
        if (shrunk != NULL) {
            array->values = shrunk;
        }
    }
    return value != NULL ? value : jayess_value_undefined();
}

int jayess_array_unshift_value(jayess_array *array, jayess_value *value) {
    int i;
    jayess_value **values;

    if (array == NULL) {
        return 0;
    }
    values = (jayess_value **)realloc(array->values, sizeof(jayess_value *) * (size_t)(array->count + 1));
    if (values == NULL) {
        return array->count;
    }
    array->values = values;
    for (i = array->count; i > 0; i--) {
        array->values[i] = array->values[i - 1];
    }
    array->values[0] = value;
    array->count++;
    return array->count;
}

jayess_array *jayess_array_slice_values(jayess_array *array, int start, int end, int has_end) {
    int i;
    int begin;
    int finish;
    int out_index = 0;
    jayess_array *copy = jayess_array_new();
    if (copy == NULL || array == NULL) {
        return copy;
    }
    begin = start < 0 ? 0 : start;
    finish = has_end ? end : array->count;
    if (finish > array->count) {
        finish = array->count;
    }
    if (begin > finish) {
        begin = finish;
    }
    for (i = begin; i < finish; i++) {
        jayess_array_set_value(copy, out_index++, array->values[i]);
    }
    return copy;
}

void jayess_array_append_array(jayess_array *array, jayess_array *other) {
    int i;
    if (array == NULL || other == NULL) {
        return;
    }
    for (i = 0; i < other->count; i++) {
        jayess_array_set_value(array, array->count, other->values[i]);
    }
}

void jayess_value_set_index(jayess_value *target, int index, jayess_value *value) {
    if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(target, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(target);
        int byte_value = (int)jayess_value_to_number(value) & 255;
        if (bytes != NULL) {
            jayess_array_set_value(bytes, index, jayess_value_from_number((double)byte_value));
        }
        return;
    }
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY) {
        return;
    }
    jayess_array_set_value(target->as.array_value, index, value);
}

jayess_value *jayess_value_get_index(jayess_value *target, int index) {
    if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(target, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(target);
        jayess_value *value = bytes != NULL ? jayess_array_get(bytes, index) : NULL;
        return value != NULL ? value : jayess_value_undefined();
    }
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
    return jayess_array_get(target->as.array_value, index);
}

void jayess_value_set_dynamic_index(jayess_value *target, jayess_value *index, jayess_value *value) {
    if (target == NULL || index == NULL || value == NULL) {
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

    if (index->kind == JAYESS_VALUE_OBJECT || index->kind == JAYESS_VALUE_ARRAY) {
        return;
    }
}

jayess_value *jayess_value_get_dynamic_index(jayess_value *target, jayess_value *index) {
    if (target == NULL || index == NULL) {
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

    return NULL;
}

void jayess_value_delete_dynamic_index(jayess_value *target, jayess_value *index) {
    if (target == NULL || index == NULL) {
        return;
    }

    if (index->kind == JAYESS_VALUE_STRING) {
        jayess_value_delete_member(target, index->as.string_value);
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
    if (target->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(target, "Uint8Array")) {
        jayess_array *bytes = jayess_std_bytes_slot(target);
        return bytes != NULL ? bytes->count : 0;
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
        if (target != NULL && target->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(target, "Uint8Array")) {
            return jayess_std_uint8_slice_values(target, start, end, has_end);
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
    return boxed;
}

jayess_value *jayess_value_from_number(double value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_NUMBER;
    boxed->as.number_value = value;
    return boxed;
}

jayess_value *jayess_value_from_bool(int value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_BOOL;
    boxed->as.bool_value = value ? 1 : 0;
    return boxed;
}

jayess_value *jayess_value_from_object(jayess_object *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_OBJECT;
    boxed->as.object_value = value;
    return boxed;
}

jayess_value *jayess_value_from_array(jayess_array *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_ARRAY;
    boxed->as.array_value = value;
    return boxed;
}

jayess_value *jayess_value_from_args(jayess_args *args) {
    int i;
    jayess_array *array = jayess_array_new();
    if (array == NULL) {
        return NULL;
    }
    if (args != NULL) {
        for (i = 0; i < args->count; i++) {
            jayess_array_set_value(array, i, jayess_value_from_string(args->values[i]));
        }
    }
    return jayess_value_from_array(array);
}

jayess_value *jayess_value_from_function(void *callee, jayess_value *env, const char *name, const char *class_name, int param_count, int has_rest) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    jayess_function *function_value;
    if (boxed == NULL) {
        return NULL;
    }
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
    boxed->kind = JAYESS_VALUE_FUNCTION;
    boxed->as.function_value = function_value;
    return boxed;
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
    bound->bound_args = jayess_array_concat(original->bound_args, tail);

    boxed->kind = JAYESS_VALUE_FUNCTION;
    boxed->as.function_value = bound;
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

    merged = jayess_array_concat(value->as.function_value->bound_args, tail);
    return jayess_value_from_array(merged);
}

void jayess_throw(jayess_value *value) {
    jayess_current_exception = value != NULL ? value : jayess_value_undefined();
}

void jayess_throw_not_function(void) {
    jayess_throw(jayess_type_error_value("value is not a function"));
}

int jayess_has_exception(void) {
    return jayess_current_exception != NULL;
}

jayess_value *jayess_take_exception(void) {
    jayess_value *value = jayess_current_exception;
    jayess_current_exception = NULL;
    return value != NULL ? value : jayess_value_undefined();
}

void jayess_report_uncaught_exception(void) {
    if (jayess_current_exception == NULL) {
        return;
    }
    fputs("Uncaught exception: ", stderr);
    jayess_print_value_inline(jayess_current_exception);
    fputc('\n', stderr);
    jayess_current_exception = NULL;
}

void jayess_push_this(jayess_value *value) {
    jayess_this_frame *frame = (jayess_this_frame *)malloc(sizeof(jayess_this_frame));
    if (frame == NULL) {
        return;
    }
    frame->value = value != NULL ? value : jayess_value_undefined();
    frame->previous = jayess_this_stack;
    jayess_this_stack = frame;
}

void jayess_pop_this(void) {
    jayess_this_frame *current = jayess_this_stack;
    if (current == NULL) {
        return;
    }
    jayess_this_stack = current->previous;
    free(current);
}

jayess_value *jayess_current_this(void) {
    if (jayess_this_stack == NULL || jayess_this_stack->value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_this_stack->value;
}

const char *jayess_value_typeof(jayess_value *value) {
    if (value == NULL) {
        return "object";
    }
    switch (value->kind) {
    case JAYESS_VALUE_UNDEFINED:
        return "undefined";
    case JAYESS_VALUE_STRING:
        return "string";
    case JAYESS_VALUE_NUMBER:
        return "number";
    case JAYESS_VALUE_BOOL:
        return "boolean";
    case JAYESS_VALUE_FUNCTION:
        return "function";
    case JAYESS_VALUE_NULL:
    case JAYESS_VALUE_OBJECT:
    case JAYESS_VALUE_ARRAY:
    default:
        return "object";
    }
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

double jayess_value_to_number(jayess_value *value) {
    if (value == NULL) {
        return 0.0;
    }
    switch (value->kind) {
    case JAYESS_VALUE_NULL:
        return 0.0;
    case JAYESS_VALUE_NUMBER:
        return value->as.number_value;
    case JAYESS_VALUE_BOOL:
        return value->as.bool_value ? 1.0 : 0.0;
    case JAYESS_VALUE_STRING:
        return strtod(value->as.string_value != NULL ? value->as.string_value : "0", NULL);
    case JAYESS_VALUE_UNDEFINED:
        return 0.0;
    default:
        return 0.0;
    }
}

int jayess_value_eq(jayess_value *left, jayess_value *right) {
    if (left == NULL || right == NULL) {
        return left == right;
    }
    if (left->kind != right->kind) {
        return 0;
    }

    switch (left->kind) {
    case JAYESS_VALUE_NULL:
    case JAYESS_VALUE_UNDEFINED:
        return 1;
    case JAYESS_VALUE_STRING:
        return strcmp(left->as.string_value != NULL ? left->as.string_value : "",
                      right->as.string_value != NULL ? right->as.string_value : "") == 0;
    case JAYESS_VALUE_NUMBER:
        return left->as.number_value == right->as.number_value;
    case JAYESS_VALUE_BOOL:
        return left->as.bool_value == right->as.bool_value;
    case JAYESS_VALUE_OBJECT:
        return left->as.object_value == right->as.object_value;
    case JAYESS_VALUE_ARRAY:
        return left->as.array_value == right->as.array_value;
    case JAYESS_VALUE_FUNCTION:
        return left->as.function_value != NULL && right->as.function_value != NULL &&
               left->as.function_value->callee == right->as.function_value->callee &&
               left->as.function_value->env == right->as.function_value->env;
    default:
        return 0;
    }
}

int jayess_value_is_nullish(jayess_value *value) {
    if (value == NULL) {
        return 1;
    }
    return value->kind == JAYESS_VALUE_NULL || value->kind == JAYESS_VALUE_UNDEFINED;
}

int jayess_string_is_truthy(const char *value) {
    return value != NULL && value[0] != '\0';
}

int jayess_string_eq(const char *left, const char *right) {
    const char *lhs = left != NULL ? left : "";
    const char *rhs = right != NULL ? right : "";
    return strcmp(lhs, rhs) == 0;
}

int jayess_args_is_truthy(jayess_args *args) {
    return args != NULL && args->count > 0;
}

int jayess_value_is_truthy(jayess_value *value) {
    if (value == NULL) {
        return 0;
    }
    switch (value->kind) {
    case JAYESS_VALUE_NULL:
    case JAYESS_VALUE_UNDEFINED:
        return 0;
    case JAYESS_VALUE_STRING:
        return value->as.string_value != NULL && value->as.string_value[0] != '\0';
    case JAYESS_VALUE_NUMBER:
        return value->as.number_value != 0.0;
    case JAYESS_VALUE_BOOL:
        return value->as.bool_value != 0;
    case JAYESS_VALUE_OBJECT:
        return value->as.object_value != NULL;
    case JAYESS_VALUE_ARRAY:
        return value->as.array_value != NULL && value->as.array_value->count > 0;
    case JAYESS_VALUE_FUNCTION:
        return value->as.function_value != NULL && value->as.function_value->callee != NULL;
    default:
        return 0;
    }
}

jayess_value_kind jayess_value_kind_of(jayess_value *value) {
    if (value == NULL) {
        return JAYESS_VALUE_NULL;
    }
    return value->kind;
}

const char *jayess_value_as_string(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_STRING) {
        return "";
    }
    return value->as.string_value != NULL ? value->as.string_value : "";
}

int jayess_value_as_bool(jayess_value *value) {
    if (value == NULL) {
        return 0;
    }
    if (value->kind == JAYESS_VALUE_BOOL) {
        return value->as.bool_value;
    }
    return jayess_value_is_truthy(value);
}

jayess_object *jayess_value_as_object(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT) {
        return NULL;
    }
    return value->as.object_value;
}

jayess_array *jayess_value_as_array(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_ARRAY) {
        return NULL;
    }
    return value->as.array_value;
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
