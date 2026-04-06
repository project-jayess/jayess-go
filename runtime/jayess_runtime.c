#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <math.h>
#include <time.h>

#ifdef _WIN32
#include <conio.h>
#include <direct.h>
#include <io.h>
#include <windows.h>
#else
#include <dirent.h>
#include <sys/stat.h>
#include <termios.h>
#include <unistd.h>
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

struct jayess_object_entry {
    char *key;
    jayess_value *value;
    jayess_object_entry *next;
};

struct jayess_object {
    jayess_object_entry *head;
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

static jayess_value jayess_null_singleton = {JAYESS_VALUE_NULL, {0}};
static jayess_value jayess_undefined_singleton = {JAYESS_VALUE_UNDEFINED, {0}};
static jayess_this_frame *jayess_this_stack = NULL;
static jayess_value *jayess_current_exception = NULL;
static jayess_args *jayess_current_args = NULL;

static char *jayess_strdup(const char *value) {
#ifdef _WIN32
    return _strdup(value);
#else
    return strdup(value);
#endif
}

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
    case JAYESS_VALUE_ARRAY:
        return jayess_strdup("[array]");
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
    return (double)time(NULL) * 1000.0;
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

jayess_value *jayess_std_process_exit(jayess_value *code) {
    int exit_code = (int)jayess_value_to_number(code);
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
    if (target == NULL || target->kind != JAYESS_VALUE_ARRAY) {
        return;
    }
    jayess_array_set_value(target->as.array_value, index, value);
}

jayess_value *jayess_value_get_index(jayess_value *target, int index) {
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
