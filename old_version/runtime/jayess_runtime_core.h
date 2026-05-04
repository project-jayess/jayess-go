#ifndef JAYESS_RUNTIME_CORE_H
#define JAYESS_RUNTIME_CORE_H

/* Runtime-only concrete state and implementation scaffolding for jayess_runtime.c.
 * This header is intentionally included only from jayess_runtime.c after platform
 * system headers are available.
 */

#include "jayess_runtime_internal.h"

#ifdef _WIN32
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
typedef struct jayess_tls_socket_state {
    jayess_socket_handle handle;
    SSL_CTX *ctx;
    SSL *ssl;
    int reject_unauthorized;
    char *host;
} jayess_tls_socket_state;
#endif

struct jayess_crypto_key_state {
#ifdef _WIN32
    BCRYPT_KEY_HANDLE handle;
#else
    EVP_PKEY *pkey;
#endif
    int is_private;
    char *type;
};

typedef struct jayess_worker_message {
    jayess_value *value;
    struct jayess_worker_message *next;
} jayess_worker_message;

typedef struct jayess_worker_state {
    jayess_value *handler;
    jayess_worker_message *inbound_head;
    jayess_worker_message *inbound_tail;
    jayess_worker_message *outbound_head;
    jayess_worker_message *outbound_tail;
    int terminate_requested;
    int closed;
#ifdef _WIN32
    CRITICAL_SECTION lock;
    CONDITION_VARIABLE inbound_available;
    CONDITION_VARIABLE outbound_available;
    HANDLE thread;
#else
    pthread_mutex_t lock;
    pthread_cond_t inbound_available;
    pthread_cond_t outbound_available;
    pthread_t thread;
#endif
} jayess_worker_state;

typedef struct jayess_managed_native_handle {
    void *handle;
    jayess_native_handle_finalizer finalizer;
    int closed;
} jayess_managed_native_handle;

typedef struct jayess_fs_watch_state {
    char *path;
    int exists;
    int is_dir;
    double size;
    double mtime_ms;
    int closed;
} jayess_fs_watch_state;

typedef struct jayess_http_server_state {
    jayess_value *handler;
    jayess_value *tls_options;
    jayess_value *backend_server;
    int secure;
    int http_mode;
    int closed;
} jayess_http_server_state;

typedef struct jayess_http_response_state {
    jayess_value *socket;
    int headers_sent;
    int finished;
    int keep_alive;
    int chunked;
} jayess_http_response_state;

typedef jayess_value *(*jayess_callback0)(void);
typedef jayess_value *(*jayess_callback1)(jayess_value *);
typedef jayess_value *(*jayess_callback2)(jayess_value *, jayess_value *);

typedef struct jayess_symbol_registry_entry {
    char *key;
    jayess_value *symbol;
    struct jayess_symbol_registry_entry *next;
} jayess_symbol_registry_entry;

typedef struct jayess_microtask jayess_microtask;

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

#endif
