#include "jayess_runtime_internal.h"

#include <ctype.h>

#ifdef _WIN32
#include <direct.h>
#include <io.h>
#include <windows.h>
#else
#include <dirent.h>
#include <sys/stat.h>
#include <unistd.h>
#endif

int jayess_path_is_separator(char ch) {
    return ch == '\\' || ch == '/';
}

const char *jayess_path_last_separator(const char *text) {
    const char *last = NULL;
    while (text != NULL && *text != '\0') {
        if (jayess_path_is_separator(*text)) {
            last = text;
        }
        text++;
    }
    return last;
}

int jayess_path_is_absolute(const char *text) {
    if (text == NULL || text[0] == '\0') {
        return 0;
    }
    if ((text[0] == '\\' || text[0] == '/') || (isalpha((unsigned char)text[0]) && text[1] == ':')) {
        return 1;
    }
    return 0;
}

char jayess_path_separator_char(void) {
#ifdef _WIN32
    return '\\';
#else
    return '/';
#endif
}

const char *jayess_path_separator_string(void) {
#ifdef _WIN32
    return "\\";
#else
    return "/";
#endif
}

const char *jayess_path_delimiter_string(void) {
#ifdef _WIN32
    return ";";
#else
    return ":";
#endif
}

int jayess_path_root_length(const char *text) {
    if (text == NULL || text[0] == '\0') {
        return 0;
    }
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
}

char jayess_path_preferred_separator_char(const char *text) {
    if (text != NULL && strchr(text, '\\') != NULL && strchr(text, '/') == NULL) {
        return '\\';
    }
    return jayess_path_separator_char();
}

jayess_array *jayess_path_split_segments(const char *text) {
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

char *jayess_path_join_segments_with_root(const char *root, jayess_array *segments, char sep) {
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

int jayess_path_exists_text(const char *path_text) {
#ifdef _WIN32
    DWORD attributes = GetFileAttributesA(path_text);
    return attributes != INVALID_FILE_ATTRIBUTES;
#else
    struct stat info;
    return path_text != NULL && stat(path_text, &info) == 0;
#endif
}

int jayess_path_is_dir_text(const char *path_text) {
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

int jayess_path_mkdir_single(const char *path_text) {
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

int jayess_fs_remove_path_recursive(const char *path_text) {
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

int jayess_object_option_bool(jayess_value *options, const char *key) {
    jayess_value *value;
    if (options == NULL || options->kind != JAYESS_VALUE_OBJECT || options->as.object_value == NULL) {
        return 0;
    }
    value = jayess_object_get(options->as.object_value, key);
    return jayess_value_as_bool(value);
}

double jayess_path_file_size_text(const char *path_text) {
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

double jayess_path_modified_time_ms_text(const char *path_text) {
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

void jayess_fs_watch_snapshot_text(const char *path_text, int *exists, int *is_dir, double *size, double *mtime_ms) {
    int found = jayess_path_exists_text(path_text);
    int dir = 0;
    double current_size = 0.0;
    double current_mtime = 0.0;
    if (found) {
        dir = jayess_path_is_dir_text(path_text);
        current_size = jayess_path_file_size_text(path_text);
        current_mtime = jayess_path_modified_time_ms_text(path_text);
    }
    if (exists != NULL) {
        *exists = found;
    }
    if (is_dir != NULL) {
        *is_dir = dir;
    }
    if (size != NULL) {
        *size = current_size;
    }
    if (mtime_ms != NULL) {
        *mtime_ms = current_mtime;
    }
}

const char *jayess_path_permissions_text(const char *path_text) {
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

jayess_value *jayess_fs_dir_entry_value(const char *name, const char *full_path, int is_dir) {
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

void jayess_fs_read_dir_collect(jayess_array *entries, const char *path_text, int recursive) {
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
    struct dirent *entry;
    size_t path_len = strlen(path_text);
    if (dir == NULL) {
        return;
    }
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
#endif
}

jayess_value *jayess_std_fs_stream_open_error(const char *kind, const char *message) {
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
