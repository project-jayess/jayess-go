#include <ctype.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#else
#include <errno.h>
#include <fcntl.h>
#include <pthread.h>
#include <sys/wait.h>
#include <unistd.h>
#endif

#include "jayess_runtime_internal.h"

#define JAYESS_SIGNAL_MAX 32

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
    HANDLE thread;
    CRITICAL_SECTION lock;
    CONDITION_VARIABLE inbound_available;
    CONDITION_VARIABLE outbound_available;
#else
    pthread_t thread;
    pthread_mutex_t lock;
    pthread_cond_t inbound_available;
    pthread_cond_t outbound_available;
#endif
} jayess_worker_state;

static volatile sig_atomic_t jayess_pending_signals[JAYESS_SIGNAL_MAX] = {0};
static sig_atomic_t jayess_installed_signals[JAYESS_SIGNAL_MAX] = {0};

static jayess_value *jayess_std_child_process_result(int status, int pid, const char *stdout_text, const char *stderr_text);
static char *jayess_shell_quote(const char *value);
static int jayess_spawn_shell_command(const char *command, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id);
static int jayess_spawn_process_argv(const char *file, jayess_array *args, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id);
static jayess_worker_message *jayess_worker_message_new(jayess_value *value);
static void jayess_worker_queue_push(jayess_worker_message **head, jayess_worker_message **tail, jayess_worker_message *message);
static jayess_worker_message *jayess_worker_queue_pop(jayess_worker_message **head, jayess_worker_message **tail);
static void jayess_worker_queue_free(jayess_worker_message **head, jayess_worker_message **tail);
static jayess_value *jayess_worker_clone_value(jayess_value *value, int depth, int *ok);
static jayess_value *jayess_worker_make_envelope(int ok, jayess_value *value, jayess_value *error);
static void jayess_worker_execute_message(jayess_worker_state *state, jayess_value *message);
static int jayess_worker_wait_outbound(jayess_worker_state *state, double timeout_ms);
static jayess_value *jayess_std_process_signal_event(int signal_number);
static void jayess_runtime_signal_handler(int signal_number);
#ifdef _WIN32
static DWORD WINAPI jayess_worker_thread_main(LPVOID raw);
#else
static void *jayess_worker_thread_main(void *raw);
#endif

int jayess_std_child_process_signal_number(const char *signal_name) {
    char normalized[32];
    size_t i = 0;
    if (signal_name == NULL || signal_name[0] == '\0') {
        return 15;
    }
    while (signal_name[i] != '\0' && i + 1 < sizeof(normalized)) {
        normalized[i] = (char)toupper((unsigned char)signal_name[i]);
        i++;
    }
    normalized[i] = '\0';
    if (strcmp(normalized, "TERM") == 0 || strcmp(normalized, "SIGTERM") == 0) return 15;
    if (strcmp(normalized, "KILL") == 0 || strcmp(normalized, "SIGKILL") == 0) return 9;
    if (strcmp(normalized, "INT") == 0 || strcmp(normalized, "SIGINT") == 0) return 2;
    if (strcmp(normalized, "HUP") == 0 || strcmp(normalized, "SIGHUP") == 0) return 1;
    if (strcmp(normalized, "QUIT") == 0 || strcmp(normalized, "SIGQUIT") == 0) return 3;
    if (strcmp(normalized, "STOP") == 0 || strcmp(normalized, "SIGSTOP") == 0) return 19;
    if (strcmp(normalized, "CONT") == 0 || strcmp(normalized, "SIGCONT") == 0) return 18;
    if (strcmp(normalized, "USR1") == 0 || strcmp(normalized, "SIGUSR1") == 0) return 10;
    if (strcmp(normalized, "USR2") == 0 || strcmp(normalized, "SIGUSR2") == 0) return 12;
    return 0;
}

const char *jayess_std_process_signal_name(int signal_number) {
    switch (signal_number) {
    case 1: return "SIGHUP";
    case 2: return "SIGINT";
    case 3: return "SIGQUIT";
    case 9: return "SIGKILL";
    case 10: return "SIGUSR1";
    case 12: return "SIGUSR2";
    case 15: return "SIGTERM";
    case 18: return "SIGCONT";
    case 19: return "SIGSTOP";
    default: return NULL;
    }
}

jayess_value *jayess_std_process_signal_bus_value(void) {
    if (jayess_process_signal_bus == NULL || jayess_process_signal_bus->kind != JAYESS_VALUE_OBJECT || jayess_process_signal_bus->as.object_value == NULL) {
        jayess_object *object = jayess_object_new();
        if (object == NULL) {
            return jayess_value_undefined();
        }
        jayess_process_signal_bus = jayess_value_from_object(object);
    }
    return jayess_process_signal_bus;
}

void jayess_runtime_note_signal(int signal_number) {
    if (signal_number <= 0 || signal_number >= JAYESS_SIGNAL_MAX) {
        return;
    }
    jayess_pending_signals[signal_number] = 1;
}

static void jayess_runtime_signal_handler(int signal_number) {
    jayess_runtime_note_signal(signal_number);
}

int jayess_std_process_install_signal(int signal_number) {
    if (signal_number <= 0 || signal_number >= JAYESS_SIGNAL_MAX) {
        return 0;
    }
    if (jayess_installed_signals[signal_number]) {
        return 1;
    }
    if (signal(signal_number, jayess_runtime_signal_handler) == SIG_ERR) {
        return 0;
    }
    jayess_installed_signals[signal_number] = 1;
    return 1;
}

static jayess_value *jayess_std_process_signal_event(int signal_number) {
    jayess_object *event;
    const char *signal_name = jayess_std_process_signal_name(signal_number);
    if (signal_name == NULL) {
        signal_name = "UNKNOWN";
    }
    event = jayess_object_new();
    if (event == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(event, "signal", jayess_value_from_string(signal_name));
    jayess_object_set_value(event, "number", jayess_value_from_number((double)signal_number));
    return jayess_value_from_object(event);
}

void jayess_runtime_dispatch_pending_signals(void) {
    int signal_number;
    jayess_value *bus = jayess_std_process_signal_bus_value();
    if (bus == NULL || bus->kind != JAYESS_VALUE_OBJECT || bus->as.object_value == NULL) {
        return;
    }
    for (signal_number = 1; signal_number < JAYESS_SIGNAL_MAX; signal_number++) {
        sig_atomic_t count = jayess_pending_signals[signal_number];
        if (count <= 0) {
            continue;
        }
        jayess_pending_signals[signal_number] = 0;
        while (count-- > 0) {
            const char *signal_name = jayess_std_process_signal_name(signal_number);
            jayess_value *event;
            if (signal_name == NULL) {
                continue;
            }
            event = jayess_std_process_signal_event(signal_number);
            jayess_std_stream_emit(bus, signal_name, event);
            if (jayess_has_exception()) {
                return;
            }
        }
    }
}

char *jayess_compile_option_string(jayess_value *options, const char *key) {
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

static jayess_value *jayess_std_child_process_result(int status, int pid, const char *stdout_text, const char *stderr_text) {
    jayess_object *result = jayess_object_new();
    jayess_object_set_value(result, "ok", jayess_value_from_bool(status == 0));
    jayess_object_set_value(result, "status", jayess_value_from_number((double)status));
    jayess_object_set_value(result, "stdout", jayess_value_from_string(stdout_text != NULL ? stdout_text : ""));
    jayess_object_set_value(result, "stderr", jayess_value_from_string(stderr_text != NULL ? stderr_text : ""));
    jayess_object_set_value(result, "pid", jayess_value_from_number((double)pid));
    return jayess_value_from_object(result);
}

static char *jayess_shell_quote(const char *value) {
    size_t length = 2;
    size_t i;
    char *quoted;
    const char *text = value != NULL ? value : "";
    for (i = 0; text[i] != '\0'; i++) {
        length += text[i] == '"' ? 2 : 1;
    }
    quoted = (char *)malloc(length + 1);
    if (quoted == NULL) {
        return NULL;
    }
    quoted[0] = '"';
    length = 1;
    for (i = 0; text[i] != '\0'; i++) {
        if (text[i] == '"') {
            quoted[length++] = '\\';
        }
        quoted[length++] = text[i];
    }
    quoted[length++] = '"';
    quoted[length] = '\0';
    return quoted;
}

#ifdef _WIN32
static int jayess_spawn_shell_command(const char *command, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id) {
    const char *comspec = getenv("COMSPEC");
    char *quoted_shell = NULL;
    char *quoted_command = NULL;
    char *command_line = NULL;
    SECURITY_ATTRIBUTES security;
    STARTUPINFOA startup;
    PROCESS_INFORMATION process;
    HANDLE stdin_handle = INVALID_HANDLE_VALUE;
    HANDLE stdout_handle = INVALID_HANDLE_VALUE;
    HANDLE stderr_handle = INVALID_HANDLE_VALUE;
    DWORD exit_code = 1;
    if (comspec == NULL || comspec[0] == '\0') {
        comspec = "cmd.exe";
    }
    quoted_shell = jayess_shell_quote(comspec);
    quoted_command = jayess_shell_quote(command != NULL ? command : "");
    if (quoted_shell == NULL || quoted_command == NULL) {
        free(quoted_shell);
        free(quoted_command);
        return -1;
    }
    command_line = (char *)malloc(strlen(quoted_shell) + strlen(quoted_command) + 8);
    if (command_line == NULL) {
        free(quoted_shell);
        free(quoted_command);
        return -1;
    }
    sprintf(command_line, "%s /C %s", quoted_shell, quoted_command);
    security.nLength = sizeof(security);
    security.lpSecurityDescriptor = NULL;
    security.bInheritHandle = TRUE;
    if (stdin_path != NULL && stdin_path[0] != '\0') {
        stdin_handle = CreateFileA(stdin_path, GENERIC_READ, FILE_SHARE_READ, &security, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
    }
    stdout_handle = CreateFileA(stdout_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    stderr_handle = CreateFileA(stderr_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (stdout_handle == INVALID_HANDLE_VALUE || stderr_handle == INVALID_HANDLE_VALUE || (stdin_path != NULL && stdin_path[0] != '\0' && stdin_handle == INVALID_HANDLE_VALUE)) {
        if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
        if (stdout_handle != INVALID_HANDLE_VALUE) CloseHandle(stdout_handle);
        if (stderr_handle != INVALID_HANDLE_VALUE) CloseHandle(stderr_handle);
        free(quoted_shell);
        free(quoted_command);
        free(command_line);
        return -1;
    }
    ZeroMemory(&startup, sizeof(startup));
    ZeroMemory(&process, sizeof(process));
    startup.cb = sizeof(startup);
    startup.dwFlags = STARTF_USESTDHANDLES;
    startup.hStdInput = stdin_handle != INVALID_HANDLE_VALUE ? stdin_handle : GetStdHandle(STD_INPUT_HANDLE);
    startup.hStdOutput = stdout_handle;
    startup.hStdError = stderr_handle;
    if (!CreateProcessA(NULL, command_line, NULL, NULL, TRUE, 0, NULL, NULL, &startup, &process)) {
        if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
        CloseHandle(stdout_handle);
        CloseHandle(stderr_handle);
        free(quoted_shell);
        free(quoted_command);
        free(command_line);
        return -1;
    }
    if (process_id != NULL) {
        *process_id = (int)process.dwProcessId;
    }
    WaitForSingleObject(process.hProcess, INFINITE);
    GetExitCodeProcess(process.hProcess, &exit_code);
    CloseHandle(process.hProcess);
    CloseHandle(process.hThread);
    if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
    CloseHandle(stdout_handle);
    CloseHandle(stderr_handle);
    free(quoted_shell);
    free(quoted_command);
    free(command_line);
    return (int)exit_code;
}

static int jayess_spawn_process_argv(const char *file, jayess_array *args, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id) {
    char *quoted_file = NULL;
    char *command_line = NULL;
    size_t command_len = 0;
    SECURITY_ATTRIBUTES security;
    STARTUPINFOA startup;
    PROCESS_INFORMATION process;
    HANDLE stdin_handle = INVALID_HANDLE_VALUE;
    HANDLE stdout_handle = INVALID_HANDLE_VALUE;
    HANDLE stderr_handle = INVALID_HANDLE_VALUE;
    DWORD exit_code = 1;
    int i;
    if (file == NULL || file[0] == '\0') {
        return -1;
    }
    quoted_file = jayess_shell_quote(file);
    if (quoted_file == NULL) {
        return -1;
    }
    command_len = strlen(quoted_file) + 1;
    for (i = 0; args != NULL && i < args->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(args, i));
        char *quoted_piece = jayess_shell_quote(piece != NULL ? piece : "");
        command_len += (quoted_piece != NULL ? strlen(quoted_piece) : 0) + 1;
        free(piece);
        free(quoted_piece);
    }
    command_line = (char *)malloc(command_len + 1);
    if (command_line == NULL) {
        free(quoted_file);
        return -1;
    }
    strcpy(command_line, quoted_file);
    for (i = 0; args != NULL && i < args->count; i++) {
        char *piece = jayess_value_stringify(jayess_array_get(args, i));
        char *quoted_piece = jayess_shell_quote(piece != NULL ? piece : "");
        strcat(command_line, " ");
        strcat(command_line, quoted_piece != NULL ? quoted_piece : "\"\"");
        free(piece);
        free(quoted_piece);
    }
    security.nLength = sizeof(security);
    security.lpSecurityDescriptor = NULL;
    security.bInheritHandle = TRUE;
    if (stdin_path != NULL && stdin_path[0] != '\0') {
        stdin_handle = CreateFileA(stdin_path, GENERIC_READ, FILE_SHARE_READ, &security, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
    }
    stdout_handle = CreateFileA(stdout_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    stderr_handle = CreateFileA(stderr_path, GENERIC_WRITE, FILE_SHARE_READ, &security, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (stdout_handle == INVALID_HANDLE_VALUE || stderr_handle == INVALID_HANDLE_VALUE || (stdin_path != NULL && stdin_path[0] != '\0' && stdin_handle == INVALID_HANDLE_VALUE)) {
        if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
        if (stdout_handle != INVALID_HANDLE_VALUE) CloseHandle(stdout_handle);
        if (stderr_handle != INVALID_HANDLE_VALUE) CloseHandle(stderr_handle);
        free(quoted_file);
        free(command_line);
        return -1;
    }
    ZeroMemory(&startup, sizeof(startup));
    ZeroMemory(&process, sizeof(process));
    startup.cb = sizeof(startup);
    startup.dwFlags = STARTF_USESTDHANDLES;
    startup.hStdInput = stdin_handle != INVALID_HANDLE_VALUE ? stdin_handle : GetStdHandle(STD_INPUT_HANDLE);
    startup.hStdOutput = stdout_handle;
    startup.hStdError = stderr_handle;
    if (!CreateProcessA(NULL, command_line, NULL, NULL, TRUE, 0, NULL, NULL, &startup, &process)) {
        if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
        CloseHandle(stdout_handle);
        CloseHandle(stderr_handle);
        free(quoted_file);
        free(command_line);
        return -1;
    }
    if (process_id != NULL) {
        *process_id = (int)process.dwProcessId;
    }
    WaitForSingleObject(process.hProcess, INFINITE);
    GetExitCodeProcess(process.hProcess, &exit_code);
    CloseHandle(process.hProcess);
    CloseHandle(process.hThread);
    if (stdin_handle != INVALID_HANDLE_VALUE) CloseHandle(stdin_handle);
    CloseHandle(stdout_handle);
    CloseHandle(stderr_handle);
    free(quoted_file);
    free(command_line);
    return (int)exit_code;
}
#else
static int jayess_spawn_shell_command(const char *command, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id) {
    int stdin_fd = -1;
    int stdout_fd = open(stdout_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int stderr_fd = open(stderr_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int status = 1;
    pid_t pid;
    if (stdin_path != NULL && stdin_path[0] != '\0') {
        stdin_fd = open(stdin_path, O_RDONLY);
    }
    if (stdout_fd < 0 || stderr_fd < 0 || (stdin_path != NULL && stdin_path[0] != '\0' && stdin_fd < 0)) {
        if (stdin_fd >= 0) close(stdin_fd);
        if (stdout_fd >= 0) close(stdout_fd);
        if (stderr_fd >= 0) close(stderr_fd);
        return -1;
    }
    pid = fork();
    if (pid < 0) {
        if (stdin_fd >= 0) close(stdin_fd);
        close(stdout_fd);
        close(stderr_fd);
        return -1;
    }
    if (pid == 0) {
        if (stdin_fd >= 0) {
            dup2(stdin_fd, STDIN_FILENO);
            close(stdin_fd);
        }
        dup2(stdout_fd, STDOUT_FILENO);
        dup2(stderr_fd, STDERR_FILENO);
        close(stdout_fd);
        close(stderr_fd);
        execl("/bin/sh", "sh", "-c", command != NULL ? command : "", (char *)NULL);
        _exit(127);
    }
    if (process_id != NULL) {
        *process_id = (int)pid;
    }
    if (stdin_fd >= 0) close(stdin_fd);
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

static int jayess_spawn_process_argv(const char *file, jayess_array *args, const char *stdin_path, const char *stdout_path, const char *stderr_path, int *process_id) {
    int stdin_fd = -1;
    int stdout_fd = open(stdout_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int stderr_fd = open(stderr_path, O_CREAT | O_TRUNC | O_WRONLY, 0600);
    int status = 1;
    pid_t pid;
    char **argv = NULL;
    int argc = 0;
    int i;
    if (file == NULL || file[0] == '\0') {
        return -1;
    }
    if (stdin_path != NULL && stdin_path[0] != '\0') {
        stdin_fd = open(stdin_path, O_RDONLY);
    }
    if (stdout_fd < 0 || stderr_fd < 0 || (stdin_path != NULL && stdin_path[0] != '\0' && stdin_fd < 0)) {
        if (stdin_fd >= 0) close(stdin_fd);
        if (stdout_fd >= 0) close(stdout_fd);
        if (stderr_fd >= 0) close(stderr_fd);
        return -1;
    }
    argc = 1 + (args != NULL ? args->count : 0);
    argv = (char **)calloc((size_t)argc + 1, sizeof(char *));
    if (argv == NULL) {
        if (stdin_fd >= 0) close(stdin_fd);
        close(stdout_fd);
        close(stderr_fd);
        return -1;
    }
    argv[0] = (char *)file;
    for (i = 0; args != NULL && i < args->count; i++) {
        argv[i + 1] = jayess_value_stringify(jayess_array_get(args, i));
    }
    pid = fork();
    if (pid < 0) {
        if (stdin_fd >= 0) close(stdin_fd);
        close(stdout_fd);
        close(stderr_fd);
        for (i = 1; i < argc; i++) free(argv[i]);
        free(argv);
        return -1;
    }
    if (pid == 0) {
        if (stdin_fd >= 0) {
            dup2(stdin_fd, STDIN_FILENO);
            close(stdin_fd);
        }
        dup2(stdout_fd, STDOUT_FILENO);
        dup2(stderr_fd, STDERR_FILENO);
        close(stdout_fd);
        close(stderr_fd);
        execvp(file, argv);
        _exit(127);
    }
    if (process_id != NULL) {
        *process_id = (int)pid;
    }
    if (stdin_fd >= 0) close(stdin_fd);
    close(stdout_fd);
    close(stderr_fd);
    if (waitpid(pid, &status, 0) < 0) {
        status = -1;
    } else if (WIFEXITED(status)) {
        status = WEXITSTATUS(status);
    } else {
        status = -1;
    }
    for (i = 1; i < argc; i++) free(argv[i]);
    free(argv);
    return status;
}
#endif

jayess_value *jayess_std_child_process_exec(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *command = NULL;
    char *input = NULL;
    const char *tmp_dir = jayess_temp_dir();
    char stdin_path[4096];
    char stdout_path[4096];
    char stderr_path[4096];
    FILE *stdin_file = NULL;
    char *stdout_text = NULL;
    char *stderr_text = NULL;
    int status;
    int pid = -1;
    long stamp = (long)time(NULL);
#ifdef _WIN32
    const char sep = '\\';
#else
    const char sep = '/';
#endif
    if (options != NULL && options->kind == JAYESS_VALUE_STRING) {
        command = jayess_value_stringify(options);
    } else if (object != NULL) {
        command = jayess_compile_option_string(options, "command");
        input = jayess_compile_option_string(options, "input");
    }
    if (command == NULL || command[0] == '\0') {
        free(command);
        free(input);
        return jayess_std_child_process_result(-1, -1, "", "childProcess.exec requires a non-empty command");
    }
    snprintf(stdin_path, sizeof(stdin_path), "%s%cjayess-child-%ld-%d.stdin", tmp_dir, sep, stamp, rand());
    snprintf(stdout_path, sizeof(stdout_path), "%s%cjayess-child-%ld-%d.stdout", tmp_dir, sep, stamp, rand());
    snprintf(stderr_path, sizeof(stderr_path), "%s%cjayess-child-%ld-%d.stderr", tmp_dir, sep, stamp, rand());
    if (input != NULL) {
        stdin_file = fopen(stdin_path, "wb");
        if (stdin_file == NULL) {
            free(command);
            free(input);
            return jayess_std_child_process_result(-1, -1, "", "failed to create child stdin pipe");
        }
        fwrite(input, 1, strlen(input), stdin_file);
        fclose(stdin_file);
    }
    status = jayess_spawn_shell_command(command, input != NULL ? stdin_path : NULL, stdout_path, stderr_path, &pid);
    stdout_text = jayess_read_text_file_or_empty(stdout_path);
    stderr_text = jayess_read_text_file_or_empty(stderr_path);
    if (input != NULL) {
        remove(stdin_path);
    }
    remove(stdout_path);
    remove(stderr_path);
    {
        jayess_value *result = jayess_std_child_process_result(status, pid, stdout_text, stderr_text);
        free(command);
        free(input);
        free(stdout_text);
        free(stderr_text);
        return result;
    }
}

jayess_value *jayess_std_child_process_spawn(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    char *file = NULL;
    char *input = NULL;
    jayess_array *args = NULL;
    const char *tmp_dir = jayess_temp_dir();
    char stdin_path[4096];
    char stdout_path[4096];
    char stderr_path[4096];
    FILE *stdin_file = NULL;
    char *stdout_text = NULL;
    char *stderr_text = NULL;
    int status;
    int pid = -1;
    long stamp = (long)time(NULL);
#ifdef _WIN32
    const char sep = '\\';
#else
    const char sep = '/';
#endif
    if (options != NULL && options->kind == JAYESS_VALUE_STRING) {
        file = jayess_value_stringify(options);
    } else if (object != NULL) {
        file = jayess_compile_option_string(options, "file");
        input = jayess_compile_option_string(options, "input");
        {
            jayess_value *args_value = jayess_object_get(object, "args");
            if (args_value != NULL && args_value->kind == JAYESS_VALUE_ARRAY) {
                args = args_value->as.array_value;
            }
        }
    }
    if (file == NULL || file[0] == '\0') {
        free(file);
        free(input);
        return jayess_std_child_process_result(-1, -1, "", "childProcess.spawn requires a non-empty file");
    }
    snprintf(stdin_path, sizeof(stdin_path), "%s%cjayess-child-%ld-%d.stdin", tmp_dir, sep, stamp, rand());
    snprintf(stdout_path, sizeof(stdout_path), "%s%cjayess-child-%ld-%d.stdout", tmp_dir, sep, stamp, rand());
    snprintf(stderr_path, sizeof(stderr_path), "%s%cjayess-child-%ld-%d.stderr", tmp_dir, sep, stamp, rand());
    if (input != NULL) {
        stdin_file = fopen(stdin_path, "wb");
        if (stdin_file == NULL) {
            free(file);
            free(input);
            return jayess_std_child_process_result(-1, -1, "", "failed to create child stdin pipe");
        }
        fwrite(input, 1, strlen(input), stdin_file);
        fclose(stdin_file);
    }
    status = jayess_spawn_process_argv(file, args, input != NULL ? stdin_path : NULL, stdout_path, stderr_path, &pid);
    stdout_text = jayess_read_text_file_or_empty(stdout_path);
    stderr_text = jayess_read_text_file_or_empty(stderr_path);
    if (input != NULL) {
        remove(stdin_path);
    }
    remove(stdout_path);
    remove(stderr_path);
    {
        jayess_value *result = jayess_std_child_process_result(status, pid, stdout_text, stderr_text);
        free(file);
        free(input);
        free(stdout_text);
        free(stderr_text);
        return result;
    }
}

jayess_value *jayess_std_child_process_kill(jayess_value *options) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_value *pid_value = options;
    char *signal_name = NULL;
    int pid;
    int ok = 0;
    if (object != NULL) {
        pid_value = jayess_object_get(object, "pid");
        signal_name = jayess_compile_option_string(options, "signal");
    }
    if (pid_value == NULL || jayess_value_is_nullish(pid_value)) {
        free(signal_name);
        return jayess_value_from_bool(0);
    }
    pid = (int)jayess_value_to_number(pid_value);
    if (pid <= 0) {
        free(signal_name);
        return jayess_value_from_bool(0);
    }
#ifdef _WIN32
    {
        HANDLE handle = OpenProcess(PROCESS_TERMINATE, FALSE, (DWORD)pid);
        if (handle != NULL) {
            ok = TerminateProcess(handle, 1) ? 1 : 0;
            CloseHandle(handle);
        }
    }
#else
    {
        int signal_number = jayess_std_child_process_signal_number(signal_name);
        if (signal_number != 0) {
            ok = kill((pid_t)pid, signal_number) == 0 ? 1 : 0;
        }
    }
#endif
    free(signal_name);
    return jayess_value_from_bool(ok);
}

static jayess_worker_message *jayess_worker_message_new(jayess_value *value) {
    jayess_worker_message *message = (jayess_worker_message *)malloc(sizeof(jayess_worker_message));
    if (message == NULL) {
        return NULL;
    }
    message->value = value != NULL ? value : jayess_value_undefined();
    message->next = NULL;
    return message;
}

static void jayess_worker_queue_push(jayess_worker_message **head, jayess_worker_message **tail, jayess_worker_message *message) {
    if (head == NULL || tail == NULL || message == NULL) {
        return;
    }
    message->next = NULL;
    if (*tail != NULL) {
        (*tail)->next = message;
    } else {
        *head = message;
    }
    *tail = message;
}

static jayess_worker_message *jayess_worker_queue_pop(jayess_worker_message **head, jayess_worker_message **tail) {
    jayess_worker_message *message;
    if (head == NULL || tail == NULL || *head == NULL) {
        return NULL;
    }
    message = *head;
    *head = message->next;
    if (*head == NULL) {
        *tail = NULL;
    }
    message->next = NULL;
    return message;
}

static void jayess_worker_queue_free(jayess_worker_message **head, jayess_worker_message **tail) {
    jayess_worker_message *current;
    if (head == NULL || tail == NULL) {
        return;
    }
    current = *head;
    while (current != NULL) {
        jayess_worker_message *next = current->next;
        free(current);
        current = next;
    }
    *head = NULL;
    *tail = NULL;
}

static jayess_value *jayess_worker_clone_value(jayess_value *value, int depth, int *ok) {
    int i;
    jayess_object *clone_object;
    jayess_array *clone_array;
    jayess_value *clone_value;
    jayess_object_entry *entry;
    if (ok != NULL) {
        *ok = 1;
    }
    if (depth > 64) {
        if (ok != NULL) {
            *ok = 0;
        }
        return jayess_value_undefined();
    }
    if (value == NULL) {
        return jayess_value_undefined();
    }
    switch (value->kind) {
    case JAYESS_VALUE_NULL:
        return jayess_value_null();
    case JAYESS_VALUE_UNDEFINED:
        return jayess_value_undefined();
    case JAYESS_VALUE_BOOL:
        return jayess_value_from_bool(value->as.bool_value);
    case JAYESS_VALUE_NUMBER:
        return jayess_value_from_number(value->as.number_value);
    case JAYESS_VALUE_STRING:
        return jayess_value_from_string(value->as.string_value != NULL ? value->as.string_value : "");
    case JAYESS_VALUE_BIGINT: {
        char *text = jayess_value_stringify(value);
        jayess_value *out = jayess_value_from_bigint(text != NULL ? text : "0");
        free(text);
        return out != NULL ? out : jayess_value_undefined();
    }
    case JAYESS_VALUE_SYMBOL:
        return value;
    case JAYESS_VALUE_FUNCTION:
        if (ok != NULL) {
            *ok = 0;
        }
        return jayess_value_undefined();
    case JAYESS_VALUE_ARRAY:
        clone_array = jayess_array_new();
        if (clone_array == NULL) {
            if (ok != NULL) {
                *ok = 0;
            }
            return jayess_value_undefined();
        }
        for (i = 0; value->as.array_value != NULL && i < value->as.array_value->count; i++) {
            int item_ok = 1;
            jayess_array_set_value(clone_array, i, jayess_worker_clone_value(jayess_array_get(value->as.array_value, i), depth + 1, &item_ok));
            if (!item_ok) {
                if (ok != NULL) {
                    *ok = 0;
                }
                return jayess_value_undefined();
            }
        }
        return jayess_value_from_array(clone_array);
    case JAYESS_VALUE_OBJECT:
        if (value->as.object_value == NULL) {
            return jayess_value_undefined();
        }
        if (jayess_std_kind_is(value, "SharedArrayBuffer")) {
            return jayess_std_buffer_value_from_state(jayess_std_bytes_state(value));
        }
        if (jayess_std_kind_is(value, "DataView") || jayess_std_is_typed_array(value)) {
            jayess_value *buffer = jayess_std_buffer_value_from_state(jayess_std_bytes_state(value));
            if (buffer != NULL && jayess_std_kind_is(buffer, "SharedArrayBuffer")) {
                if (jayess_std_kind_is(value, "DataView")) {
                    jayess_value *clone = jayess_std_data_view_new(buffer);
                    jayess_value_free_unshared(buffer);
                    return clone;
                }
                clone_value = jayess_std_typed_array_new(jayess_std_typed_array_kind(value), buffer);
                jayess_value_free_unshared(buffer);
                return clone_value;
            }
            jayess_value_free_unshared(buffer);
        }
        if (value->as.object_value->stream_file != NULL || value->as.object_value->native_handle != NULL || value->as.object_value->socket_handle != JAYESS_INVALID_SOCKET) {
            if (ok != NULL) {
                *ok = 0;
            }
            return jayess_value_undefined();
        }
        clone_object = jayess_object_new();
        if (clone_object == NULL) {
            if (ok != NULL) {
                *ok = 0;
            }
            return jayess_value_undefined();
        }
        for (entry = value->as.object_value->head; entry != NULL; entry = entry->next) {
            int key_ok = 1;
            int value_ok = 1;
            jayess_value *cloned_key = entry->key_value != NULL ? jayess_worker_clone_value(entry->key_value, depth + 1, &key_ok) : jayess_value_from_string(entry->key != NULL ? entry->key : "");
            jayess_value *cloned_entry_value = jayess_worker_clone_value(entry->value, depth + 1, &value_ok);
            if (!key_ok || !value_ok) {
                if (ok != NULL) {
                    *ok = 0;
                }
                return jayess_value_undefined();
            }
            jayess_object_set_key_value(clone_object, cloned_key, cloned_entry_value);
        }
        clone_value = jayess_value_from_object(clone_object);
        return clone_value != NULL ? clone_value : jayess_value_undefined();
    default:
        if (ok != NULL) {
            *ok = 0;
        }
        return jayess_value_undefined();
    }
}

static jayess_value *jayess_worker_make_envelope(int ok, jayess_value *value, jayess_value *error) {
    jayess_object *envelope = jayess_object_new();
    if (envelope == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(envelope, "ok", jayess_value_from_bool(ok));
    jayess_object_set_value(envelope, "value", value != NULL ? value : jayess_value_undefined());
    jayess_object_set_value(envelope, "error", error != NULL ? error : jayess_value_undefined());
    return jayess_value_from_object(envelope);
}

static void jayess_worker_execute_message(jayess_worker_state *state, jayess_value *message) {
    jayess_value *result;
    jayess_value *envelope;
    jayess_worker_message *outbound;
    int clone_ok = 1;
    if (state == NULL || state->handler == NULL) {
        return;
    }
    result = jayess_value_call_with_this(state->handler, jayess_value_undefined(), message, 1);
    if (jayess_has_exception()) {
        jayess_value *error_value = jayess_worker_clone_value(jayess_take_exception(), 0, &clone_ok);
        if (!clone_ok) {
            error_value = jayess_type_error_value("worker failed to clone thrown value");
        }
        envelope = jayess_worker_make_envelope(0, jayess_value_undefined(), error_value);
    } else {
        jayess_value *cloned_result = jayess_worker_clone_value(result, 0, &clone_ok);
        if (!clone_ok) {
            envelope = jayess_worker_make_envelope(0, jayess_value_undefined(), jayess_type_error_value("worker failed to clone result value"));
        } else {
            envelope = jayess_worker_make_envelope(1, cloned_result, jayess_value_undefined());
        }
    }
    outbound = jayess_worker_message_new(envelope);
    if (outbound == NULL) {
        return;
    }
#ifdef _WIN32
    EnterCriticalSection(&state->lock);
    jayess_worker_queue_push(&state->outbound_head, &state->outbound_tail, outbound);
    WakeConditionVariable(&state->outbound_available);
    LeaveCriticalSection(&state->lock);
#else
    pthread_mutex_lock(&state->lock);
    jayess_worker_queue_push(&state->outbound_head, &state->outbound_tail, outbound);
    pthread_cond_signal(&state->outbound_available);
    pthread_mutex_unlock(&state->lock);
#endif
}

#ifdef _WIN32
static DWORD WINAPI jayess_worker_thread_main(LPVOID raw) {
    jayess_worker_state *state = (jayess_worker_state *)raw;
    for (;;) {
        jayess_worker_message *message;
        EnterCriticalSection(&state->lock);
        while (!state->terminate_requested && state->inbound_head == NULL) {
            SleepConditionVariableCS(&state->inbound_available, &state->lock, INFINITE);
        }
        if (state->terminate_requested && state->inbound_head == NULL) {
            state->closed = 1;
            WakeConditionVariable(&state->outbound_available);
            LeaveCriticalSection(&state->lock);
            break;
        }
        message = jayess_worker_queue_pop(&state->inbound_head, &state->inbound_tail);
        LeaveCriticalSection(&state->lock);
        if (message != NULL) {
            jayess_worker_execute_message(state, message->value);
            free(message);
        }
    }
    return 0;
}
#else
static void *jayess_worker_thread_main(void *raw) {
    jayess_worker_state *state = (jayess_worker_state *)raw;
    for (;;) {
        jayess_worker_message *message;
        pthread_mutex_lock(&state->lock);
        while (!state->terminate_requested && state->inbound_head == NULL) {
            pthread_cond_wait(&state->inbound_available, &state->lock);
        }
        if (state->terminate_requested && state->inbound_head == NULL) {
            state->closed = 1;
            pthread_cond_broadcast(&state->outbound_available);
            pthread_mutex_unlock(&state->lock);
            break;
        }
        message = jayess_worker_queue_pop(&state->inbound_head, &state->inbound_tail);
        pthread_mutex_unlock(&state->lock);
        if (message != NULL) {
            jayess_worker_execute_message(state, message->value);
            free(message);
        }
    }
    return NULL;
}
#endif

static int jayess_worker_wait_outbound(jayess_worker_state *state, double timeout_ms) {
#ifdef _WIN32
    DWORD wait_ms = timeout_ms < 0 ? INFINITE : (DWORD)timeout_ms;
    while (state->outbound_head == NULL && !state->closed) {
        if (!SleepConditionVariableCS(&state->outbound_available, &state->lock, wait_ms)) {
            return 0;
        }
        if (timeout_ms >= 0) {
            break;
        }
    }
    return state->outbound_head != NULL;
#else
    if (timeout_ms < 0) {
        while (state->outbound_head == NULL && !state->closed) {
            pthread_cond_wait(&state->outbound_available, &state->lock);
        }
        return state->outbound_head != NULL;
    }
    while (state->outbound_head == NULL && !state->closed) {
        struct timespec deadline;
        clock_gettime(CLOCK_REALTIME, &deadline);
        deadline.tv_sec += (time_t)(timeout_ms / 1000.0);
        deadline.tv_nsec += (long)((long long)(timeout_ms * 1000000.0) % 1000000000LL);
        if (deadline.tv_nsec >= 1000000000L) {
            deadline.tv_sec += 1;
            deadline.tv_nsec -= 1000000000L;
        }
        if (pthread_cond_timedwait(&state->outbound_available, &state->lock, &deadline) != 0) {
            break;
        }
    }
    return state->outbound_head != NULL;
#endif
}

jayess_value *jayess_std_worker_create(jayess_value *handler) {
    jayess_object *object;
    jayess_worker_state *state;
    if (handler == NULL || handler->kind != JAYESS_VALUE_FUNCTION || handler->as.function_value == NULL) {
        return jayess_type_error_value("worker.create expects a function");
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_worker_state *)calloc(1, sizeof(jayess_worker_state));
    if (state == NULL) {
        return jayess_value_undefined();
    }
    state->handler = handler;
#ifdef _WIN32
    InitializeCriticalSection(&state->lock);
    InitializeConditionVariable(&state->inbound_available);
    InitializeConditionVariable(&state->outbound_available);
    state->thread = CreateThread(NULL, 0, jayess_worker_thread_main, state, 0, NULL);
    if (state->thread == NULL) {
        DeleteCriticalSection(&state->lock);
        free(state);
        return jayess_value_undefined();
    }
#else
    if (pthread_mutex_init(&state->lock, NULL) != 0 ||
        pthread_cond_init(&state->inbound_available, NULL) != 0 ||
        pthread_cond_init(&state->outbound_available, NULL) != 0 ||
        pthread_create(&state->thread, NULL, jayess_worker_thread_main, state) != 0) {
        pthread_cond_destroy(&state->outbound_available);
        pthread_cond_destroy(&state->inbound_available);
        pthread_mutex_destroy(&state->lock);
        free(state);
        return jayess_value_undefined();
    }
#endif
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("Worker"));
    jayess_object_set_value(object, "closed", jayess_value_from_bool(0));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_worker_post_message_method(jayess_value *env, jayess_value *message) {
    jayess_worker_state *state;
    jayess_worker_message *queued;
    int clone_ok = 1;
    jayess_value *cloned_message;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(env, "Worker") || env->as.object_value == NULL) {
        return jayess_value_from_bool(0);
    }
    state = (jayess_worker_state *)env->as.object_value->native_handle;
    if (state == NULL || state->closed) {
        return jayess_value_from_bool(0);
    }
    cloned_message = jayess_worker_clone_value(message, 0, &clone_ok);
    if (!clone_ok) {
        jayess_throw(jayess_type_error_value("worker.postMessage only supports cloneable values"));
        return jayess_value_undefined();
    }
    queued = jayess_worker_message_new(cloned_message);
    if (queued == NULL) {
        return jayess_value_from_bool(0);
    }
#ifdef _WIN32
    EnterCriticalSection(&state->lock);
    if (state->closed || state->terminate_requested) {
        LeaveCriticalSection(&state->lock);
        free(queued);
        return jayess_value_from_bool(0);
    }
    jayess_worker_queue_push(&state->inbound_head, &state->inbound_tail, queued);
    WakeConditionVariable(&state->inbound_available);
    LeaveCriticalSection(&state->lock);
#else
    pthread_mutex_lock(&state->lock);
    if (state->closed || state->terminate_requested) {
        pthread_mutex_unlock(&state->lock);
        free(queued);
        return jayess_value_from_bool(0);
    }
    jayess_worker_queue_push(&state->inbound_head, &state->inbound_tail, queued);
    pthread_cond_signal(&state->inbound_available);
    pthread_mutex_unlock(&state->lock);
#endif
    return jayess_value_from_bool(1);
}

jayess_value *jayess_std_worker_receive_method(jayess_value *env, jayess_value *timeout) {
    jayess_worker_state *state;
    jayess_worker_message *message;
    double timeout_ms = -1;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(env, "Worker") || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_worker_state *)env->as.object_value->native_handle;
    if (state == NULL) {
        return jayess_value_undefined();
    }
    if (timeout != NULL && !jayess_value_is_nullish(timeout)) {
        timeout_ms = jayess_value_to_number(timeout);
    }
#ifdef _WIN32
    EnterCriticalSection(&state->lock);
    if (!jayess_worker_wait_outbound(state, timeout_ms)) {
        LeaveCriticalSection(&state->lock);
        return jayess_value_undefined();
    }
    message = jayess_worker_queue_pop(&state->outbound_head, &state->outbound_tail);
    jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(state->closed));
    LeaveCriticalSection(&state->lock);
#else
    pthread_mutex_lock(&state->lock);
    if (!jayess_worker_wait_outbound(state, timeout_ms)) {
        pthread_mutex_unlock(&state->lock);
        return jayess_value_undefined();
    }
    message = jayess_worker_queue_pop(&state->outbound_head, &state->outbound_tail);
    jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(state->closed));
    pthread_mutex_unlock(&state->lock);
#endif
    if (message == NULL) {
        return jayess_value_undefined();
    }
    {
        jayess_value *result = message->value != NULL ? message->value : jayess_value_undefined();
        free(message);
        return result;
    }
}

jayess_value *jayess_std_worker_terminate_method(jayess_value *env) {
    jayess_worker_state *state;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(env, "Worker") || env->as.object_value == NULL) {
        return jayess_value_from_bool(0);
    }
    state = (jayess_worker_state *)env->as.object_value->native_handle;
    if (state == NULL) {
        return jayess_value_from_bool(1);
    }
#ifdef _WIN32
    EnterCriticalSection(&state->lock);
    state->terminate_requested = 1;
    WakeConditionVariable(&state->inbound_available);
    WakeConditionVariable(&state->outbound_available);
    LeaveCriticalSection(&state->lock);
    WaitForSingleObject(state->thread, INFINITE);
    CloseHandle(state->thread);
    DeleteCriticalSection(&state->lock);
#else
    pthread_mutex_lock(&state->lock);
    state->terminate_requested = 1;
    pthread_cond_broadcast(&state->inbound_available);
    pthread_cond_broadcast(&state->outbound_available);
    pthread_mutex_unlock(&state->lock);
    pthread_join(state->thread, NULL);
    pthread_cond_destroy(&state->outbound_available);
    pthread_cond_destroy(&state->inbound_available);
    pthread_mutex_destroy(&state->lock);
#endif
    state->closed = 1;
    jayess_worker_queue_free(&state->inbound_head, &state->inbound_tail);
    jayess_worker_queue_free(&state->outbound_head, &state->outbound_tail);
    env->as.object_value->native_handle = NULL;
    jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
    free(state);
    return jayess_value_from_bool(1);
}
