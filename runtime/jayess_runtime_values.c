#include "jayess_runtime_internal.h"

jayess_runtime_accounting jayess_runtime_accounting_state = {0};

static jayess_value jayess_bool_false_singleton = {
    .kind = JAYESS_VALUE_BOOL,
    .as.bool_value = 0,
};

static jayess_value jayess_bool_true_singleton = {
    .kind = JAYESS_VALUE_BOOL,
    .as.bool_value = 1,
};

static jayess_value jayess_number_neg_one_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = -1.0,
};

static jayess_value jayess_number_zero_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 0.0,
};

static jayess_value jayess_number_one_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 1.0,
};

static jayess_value jayess_number_two_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 2.0,
};

static jayess_value jayess_number_three_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 3.0,
};

static jayess_value jayess_number_four_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 4.0,
};

static jayess_value jayess_number_five_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 5.0,
};

static jayess_value jayess_number_six_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 6.0,
};

static jayess_value jayess_number_seven_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 7.0,
};

static jayess_value jayess_number_eight_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 8.0,
};

static jayess_value jayess_number_nine_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 9.0,
};

static jayess_value jayess_number_ten_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 10.0,
};

static jayess_value jayess_number_eleven_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 11.0,
};

static jayess_value jayess_number_twelve_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 12.0,
};

static jayess_value jayess_number_thirteen_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 13.0,
};

static jayess_value jayess_number_fourteen_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 14.0,
};

static jayess_value jayess_number_fifteen_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 15.0,
};

static jayess_value jayess_number_sixteen_singleton = {
    .kind = JAYESS_VALUE_NUMBER,
    .as.number_value = 16.0,
};

typedef struct jayess_static_string_entry {
    char *text;
    jayess_value *boxed;
    struct jayess_static_string_entry *next;
} jayess_static_string_entry;

static jayess_static_string_entry *jayess_static_strings = NULL;

static int jayess_value_is_static_string_box(jayess_value *value) {
    jayess_static_string_entry *current = jayess_static_strings;
    while (current != NULL) {
        if (current->boxed == value) {
            return 1;
        }
        current = current->next;
    }
    return 0;
}

static int jayess_value_is_immortal_singleton(jayess_value *value) {
    return value == jayess_value_null() ||
           value == jayess_value_undefined() ||
           jayess_value_is_static_string_box(value) ||
           value == &jayess_bool_false_singleton ||
           value == &jayess_bool_true_singleton ||
           value == &jayess_number_neg_one_singleton ||
           value == &jayess_number_zero_singleton ||
           value == &jayess_number_one_singleton ||
           value == &jayess_number_two_singleton ||
           value == &jayess_number_three_singleton ||
           value == &jayess_number_four_singleton ||
           value == &jayess_number_five_singleton ||
           value == &jayess_number_six_singleton ||
           value == &jayess_number_seven_singleton ||
           value == &jayess_number_eight_singleton ||
           value == &jayess_number_nine_singleton ||
           value == &jayess_number_ten_singleton ||
           value == &jayess_number_eleven_singleton ||
           value == &jayess_number_twelve_singleton ||
           value == &jayess_number_thirteen_singleton ||
           value == &jayess_number_fourteen_singleton ||
           value == &jayess_number_fifteen_singleton ||
           value == &jayess_number_sixteen_singleton;
}

static jayess_value *jayess_number_singleton(double value) {
    if (value == -1.0) {
        return &jayess_number_neg_one_singleton;
    }
    if (value == 0.0) {
        return &jayess_number_zero_singleton;
    }
    if (value == 1.0) {
        return &jayess_number_one_singleton;
    }
    if (value == 2.0) {
        return &jayess_number_two_singleton;
    }
    if (value == 3.0) {
        return &jayess_number_three_singleton;
    }
    if (value == 4.0) {
        return &jayess_number_four_singleton;
    }
    if (value == 5.0) {
        return &jayess_number_five_singleton;
    }
    if (value == 6.0) {
        return &jayess_number_six_singleton;
    }
    if (value == 7.0) {
        return &jayess_number_seven_singleton;
    }
    if (value == 8.0) {
        return &jayess_number_eight_singleton;
    }
    if (value == 9.0) {
        return &jayess_number_nine_singleton;
    }
    if (value == 10.0) {
        return &jayess_number_ten_singleton;
    }
    if (value == 11.0) {
        return &jayess_number_eleven_singleton;
    }
    if (value == 12.0) {
        return &jayess_number_twelve_singleton;
    }
    if (value == 13.0) {
        return &jayess_number_thirteen_singleton;
    }
    if (value == 14.0) {
        return &jayess_number_fourteen_singleton;
    }
    if (value == 15.0) {
        return &jayess_number_fifteen_singleton;
    }
    if (value == 16.0) {
        return &jayess_number_sixteen_singleton;
    }
    return NULL;
}

jayess_value *jayess_value_from_static_string(const char *value) {
    const char *text = value != NULL ? value : "";
    jayess_static_string_entry *current = jayess_static_strings;
    jayess_static_string_entry *entry;
    jayess_value *boxed;

    while (current != NULL) {
        if (strcmp(current->text, text) == 0) {
            return current->boxed;
        }
        current = current->next;
    }

    entry = (jayess_static_string_entry *)malloc(sizeof(jayess_static_string_entry));
    if (entry == NULL) {
        return jayess_value_from_string(text);
    }
    entry->text = jayess_strdup(text);
    if (entry->text == NULL) {
        free(entry);
        return jayess_value_from_string(text);
    }
    boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        free(entry->text);
        free(entry);
        return jayess_value_from_string(text);
    }
    boxed->kind = JAYESS_VALUE_STRING;
    boxed->as.string_value = entry->text;
    entry->boxed = boxed;
    entry->next = jayess_static_strings;
    jayess_static_strings = entry;
    return boxed;
}

void jayess_runtime_free_static_strings(void) {
    jayess_static_string_entry *current = jayess_static_strings;
    while (current != NULL) {
        jayess_static_string_entry *next = current->next;
        free(current->boxed);
        free(current->text);
        free(current);
        current = next;
    }
    jayess_static_strings = NULL;
}

jayess_value *jayess_value_from_owned_string(char *value) {
    jayess_value *boxed;
    if (value == NULL) {
        return jayess_value_from_string("");
    }
    boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        free(value);
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_STRING;
    boxed->as.string_value = value;
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.strings++;
    return boxed;
}

void jayess_value_free_unshared(jayess_value *value) {
    if (value == NULL || jayess_value_is_immortal_singleton(value)) {
        return;
    }
    switch (value->kind) {
        case JAYESS_VALUE_STRING:
            if (jayess_runtime_accounting_state.strings > 0) {
                jayess_runtime_accounting_state.strings--;
            }
            free(value->as.string_value);
            break;
        case JAYESS_VALUE_BIGINT:
            if (jayess_runtime_accounting_state.bigints > 0) {
                jayess_runtime_accounting_state.bigints--;
            }
            free(value->as.bigint_value);
            break;
        case JAYESS_VALUE_OBJECT:
            jayess_object_free_unshared(value->as.object_value);
            break;
        case JAYESS_VALUE_ARRAY:
            jayess_array_free_unshared(value->as.array_value);
            break;
        case JAYESS_VALUE_FUNCTION:
            if (value->as.function_value != NULL) {
                if (jayess_runtime_accounting_state.functions > 0) {
                    jayess_runtime_accounting_state.functions--;
                }
                if (value->as.function_value->properties != NULL) {
                    jayess_object_free_unshared(value->as.function_value->properties);
                }
                if (value->as.function_value->bound_args != NULL) {
                    jayess_array_free_unshared(value->as.function_value->bound_args);
                }
                free(value->as.function_value);
            }
            break;
        case JAYESS_VALUE_SYMBOL:
            if (value->as.symbol_value != NULL) {
                if (jayess_runtime_accounting_state.symbols > 0) {
                    jayess_runtime_accounting_state.symbols--;
                }
                free(value->as.symbol_value->description);
                free(value->as.symbol_value);
            }
            break;
        case JAYESS_VALUE_NULL:
        case JAYESS_VALUE_NUMBER:
        case JAYESS_VALUE_BOOL:
        case JAYESS_VALUE_UNDEFINED:
        default:
            break;
    }
    if (jayess_runtime_accounting_state.boxed_values > 0) {
        jayess_runtime_accounting_state.boxed_values--;
    }
	free(value);
}

void jayess_value_free_array_shallow(jayess_value *value) {
    if (value == NULL || jayess_value_is_immortal_singleton(value)) {
        return;
    }
    if (value->kind != JAYESS_VALUE_ARRAY) {
        jayess_value_free_unshared(value);
        return;
    }
    if (value->as.array_value != NULL) {
        free(value->as.array_value->values);
        free(value->as.array_value);
    }
    free(value);
}

jayess_value *jayess_value_from_number(double value) {
    jayess_value *singleton = jayess_number_singleton(value);
    if (singleton != NULL) {
        return singleton;
    }
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_NUMBER;
    boxed->as.number_value = value;
    jayess_runtime_accounting_state.boxed_values++;
    return boxed;
}

jayess_value *jayess_value_from_bigint(const char *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_BIGINT;
    boxed->as.bigint_value = jayess_strdup(value != NULL ? value : "0");
    jayess_runtime_accounting_state.boxed_values++;
    jayess_runtime_accounting_state.bigints++;
    return boxed;
}

jayess_value *jayess_value_from_bool(int value) {
    return value ? &jayess_bool_true_singleton : &jayess_bool_false_singleton;
}

jayess_value *jayess_value_from_object(jayess_object *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_OBJECT;
    boxed->as.object_value = value;
    jayess_runtime_accounting_state.boxed_values++;
    return boxed;
}

jayess_value *jayess_value_from_array(jayess_array *value) {
    jayess_value *boxed = (jayess_value *)malloc(sizeof(jayess_value));
    if (boxed == NULL) {
        return NULL;
    }
    boxed->kind = JAYESS_VALUE_ARRAY;
    boxed->as.array_value = value;
    jayess_runtime_accounting_state.boxed_values++;
    return boxed;
}

jayess_value *jayess_value_constructor_return(jayess_value *self, jayess_value *value) {
    if (value != NULL && value != self) {
        jayess_value_free_unshared(self);
        return value;
    }
    if (self != NULL) {
        return self;
    }
    if (value != NULL) {
        return value;
    }
    return jayess_value_undefined();
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
    case JAYESS_VALUE_BIGINT:
        return "bigint";
    case JAYESS_VALUE_BOOL:
        return "boolean";
    case JAYESS_VALUE_SYMBOL:
        return "symbol";
    case JAYESS_VALUE_FUNCTION:
        return "function";
    case JAYESS_VALUE_NULL:
    case JAYESS_VALUE_OBJECT:
    case JAYESS_VALUE_ARRAY:
    default:
        return "object";
    }
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
    case JAYESS_VALUE_BIGINT:
        return strtod(value->as.bigint_value != NULL ? value->as.bigint_value : "0", NULL);
    case JAYESS_VALUE_BOOL:
        return value->as.bool_value ? 1.0 : 0.0;
    case JAYESS_VALUE_STRING:
        return strtod(value->as.string_value != NULL ? value->as.string_value : "0", NULL);
    case JAYESS_VALUE_SYMBOL:
        return 0.0;
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
    case JAYESS_VALUE_BIGINT:
        return strcmp(left->as.bigint_value != NULL ? left->as.bigint_value : "0",
                      right->as.bigint_value != NULL ? right->as.bigint_value : "0") == 0;
    case JAYESS_VALUE_BOOL:
        return left->as.bool_value == right->as.bool_value;
    case JAYESS_VALUE_SYMBOL:
        return left->as.symbol_value != NULL && right->as.symbol_value != NULL &&
               left->as.symbol_value->id == right->as.symbol_value->id;
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
    case JAYESS_VALUE_BIGINT:
        return value->as.bigint_value != NULL && strcmp(value->as.bigint_value, "0") != 0;
    case JAYESS_VALUE_BOOL:
        return value->as.bool_value != 0;
    case JAYESS_VALUE_SYMBOL:
        return value->as.symbol_value != NULL;
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

const char *jayess_expect_string(jayess_value *value, const char *context) {
    if (value == NULL || value->kind != JAYESS_VALUE_STRING) {
        char message[256];
        snprintf(message, sizeof(message), "%s expects a string", context != NULL && context[0] != '\0' ? context : "native wrapper");
        jayess_throw_type_error(message);
        return "";
    }
    return value->as.string_value != NULL ? value->as.string_value : "";
}

jayess_object *jayess_expect_object(jayess_value *value, const char *context) {
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || value->as.object_value == NULL) {
        char message[256];
        snprintf(message, sizeof(message), "%s expects an object", context != NULL && context[0] != '\0' ? context : "native wrapper");
        jayess_throw_type_error(message);
        return NULL;
    }
    return value->as.object_value;
}

jayess_array *jayess_expect_array(jayess_value *value, const char *context) {
    if (value == NULL || value->kind != JAYESS_VALUE_ARRAY || value->as.array_value == NULL) {
        char message[256];
        snprintf(message, sizeof(message), "%s expects an array", context != NULL && context[0] != '\0' ? context : "native wrapper");
        jayess_throw_type_error(message);
        return NULL;
    }
    return value->as.array_value;
}
