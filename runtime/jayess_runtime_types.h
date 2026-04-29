#ifndef JAYESS_RUNTIME_TYPES_H
#define JAYESS_RUNTIME_TYPES_H

typedef struct jayess_args jayess_args;
typedef struct jayess_value jayess_value;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;
typedef void (*jayess_native_handle_finalizer)(void *);

typedef enum jayess_value_kind {
    JAYESS_VALUE_NULL = 0,
    JAYESS_VALUE_STRING = 1,
    JAYESS_VALUE_NUMBER = 2,
    JAYESS_VALUE_BIGINT = 3,
    JAYESS_VALUE_BOOL = 4,
    JAYESS_VALUE_OBJECT = 5,
    JAYESS_VALUE_ARRAY = 6,
    JAYESS_VALUE_UNDEFINED = 7,
    JAYESS_VALUE_FUNCTION = 8,
    JAYESS_VALUE_SYMBOL = 9
} jayess_value_kind;

#endif
