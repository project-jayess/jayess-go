#include "jayess_runtime_internal.h"

#ifdef _WIN32
#include <conio.h>
#else
#include <termios.h>
#include <unistd.h>
#endif

static char *jayess_number_to_string(double value) {
    char buffer[64];
    snprintf(buffer, sizeof(buffer), "%g", value);
    return jayess_strdup(buffer);
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
        jayess_print_property_key_inline(current);
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
    case JAYESS_VALUE_BIGINT:
        return jayess_strdup(value->as.bigint_value != NULL ? value->as.bigint_value : "0");
    case JAYESS_VALUE_BOOL:
        return jayess_strdup(value->as.bool_value ? "true" : "false");
    case JAYESS_VALUE_SYMBOL: {
        const char *description = NULL;
        size_t length;
        char *out;
        if (value->as.symbol_value != NULL) {
            description = value->as.symbol_value->description;
        }
        length = strlen("Symbol(") + (description != NULL ? strlen(description) : 0) + strlen(")") + 1;
        out = (char *)malloc(length);
        if (out == NULL) {
            return NULL;
        }
        snprintf(out, length, "Symbol(%s)", description != NULL ? description : "");
        return out;
    }
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

jayess_value *jayess_value_add(jayess_value *left, jayess_value *right) {
    if ((left != NULL && left->kind == JAYESS_VALUE_STRING) || (right != NULL && right->kind == JAYESS_VALUE_STRING)) {
        char *text = jayess_concat_values(left, right);
        jayess_value *result = jayess_value_from_string(text != NULL ? text : "");
        free(text);
        return result;
    }
    return jayess_value_from_number(jayess_value_to_number(left) + jayess_value_to_number(right));
}

void jayess_print_value_inline(jayess_value *value) {
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
    case JAYESS_VALUE_BIGINT:
        fputs(value->as.bigint_value != NULL ? value->as.bigint_value : "0", stdout);
        fputc('n', stdout);
        break;
    case JAYESS_VALUE_BOOL:
        fputs(value->as.bool_value ? "true" : "false", stdout);
        break;
    case JAYESS_VALUE_SYMBOL:
        if (value->as.symbol_value != NULL && value->as.symbol_value->description != NULL) {
            printf("Symbol(%s)", value->as.symbol_value->description);
        } else {
            fputs("Symbol()", stdout);
        }
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
                jayess_print_property_key_inline(current);
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

char *jayess_read_line_value(jayess_value *prompt) {
    char *prompt_text = jayess_value_stringify(prompt);
    char *result = jayess_read_line(prompt_text);
    free(prompt_text);
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

char *jayess_read_key_value(jayess_value *prompt) {
    char *prompt_text = jayess_value_stringify(prompt);
    char *result = jayess_read_key(prompt_text);
    free(prompt_text);
    return result;
}
