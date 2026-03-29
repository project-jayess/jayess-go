#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <conio.h>
#include <windows.h>
#else
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
    JAYESS_VALUE_UNDEFINED = 6
} jayess_value_kind;

typedef struct jayess_value jayess_value;
typedef struct jayess_object_entry jayess_object_entry;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;

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

struct jayess_value {
    jayess_value_kind kind;
    union {
        char *string_value;
        double number_value;
        int bool_value;
        jayess_object *object_value;
        jayess_array *array_value;
    } as;
};

static jayess_value jayess_null_singleton = {JAYESS_VALUE_NULL, {0}};
static jayess_value jayess_undefined_singleton = {JAYESS_VALUE_UNDEFINED, {0}};

static char *jayess_strdup(const char *value) {
#ifdef _WIN32
    return _strdup(value);
#else
    return strdup(value);
#endif
}

static void jayess_print_value_inline(jayess_value *value);

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
    return args;
}

char *jayess_args_get(jayess_args *args, int index) {
    if (args == NULL || index < 0 || index >= args->count) {
        return "";
    }
    return args->values[index];
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

void jayess_value_set_member(jayess_value *target, const char *key, jayess_value *value) {
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT) {
        return;
    }
    jayess_object_set_value(target->as.object_value, key, value);
}

jayess_value *jayess_value_get_member(jayess_value *target, const char *key) {
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT) {
        return NULL;
    }
    return jayess_object_get(target->as.object_value, key);
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
    default:
        return 0;
    }
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
