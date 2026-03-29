#ifndef JAYESS_RUNTIME_H
#define JAYESS_RUNTIME_H

typedef struct jayess_args jayess_args;
typedef struct jayess_value jayess_value;
typedef struct jayess_object jayess_object;
typedef struct jayess_array jayess_array;

typedef enum jayess_value_kind {
    JAYESS_VALUE_NULL = 0,
    JAYESS_VALUE_STRING = 1,
    JAYESS_VALUE_NUMBER = 2,
    JAYESS_VALUE_BOOL = 3,
    JAYESS_VALUE_OBJECT = 4,
    JAYESS_VALUE_ARRAY = 5,
    JAYESS_VALUE_UNDEFINED = 6
} jayess_value_kind;

jayess_value *jayess_value_null(void);
jayess_value *jayess_value_undefined(void);

void jayess_print_string(const char *text);
void jayess_print_number(double value);
void jayess_print_bool(int value);
void jayess_print_object(jayess_object *object);
void jayess_print_array(jayess_array *array);
void jayess_print_args(jayess_args *args);
void jayess_print_value(jayess_value *value);

char *jayess_read_line(const char *prompt);
char *jayess_read_key(const char *prompt);
void jayess_sleep_ms(int milliseconds);

jayess_args *jayess_make_args(int argc, char **argv);
char *jayess_args_get(jayess_args *args, int index);

jayess_object *jayess_object_new(void);
void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value);
jayess_value *jayess_object_get(jayess_object *object, const char *key);

void jayess_value_set_member(jayess_value *target, const char *key, jayess_value *value);
jayess_value *jayess_value_get_member(jayess_value *target, const char *key);

jayess_array *jayess_array_new(void);
void jayess_array_set_value(jayess_array *array, int index, jayess_value *value);
jayess_value *jayess_array_get(jayess_array *array, int index);

void jayess_value_set_index(jayess_value *target, int index, jayess_value *value);
jayess_value *jayess_value_get_index(jayess_value *target, int index);

jayess_value *jayess_value_from_string(const char *value);
jayess_value *jayess_value_from_number(double value);
jayess_value *jayess_value_from_bool(int value);
jayess_value *jayess_value_from_object(jayess_object *value);
jayess_value *jayess_value_from_array(jayess_array *value);

double jayess_value_to_number(jayess_value *value);
int jayess_value_eq(jayess_value *left, jayess_value *right);
int jayess_string_is_truthy(const char *value);
int jayess_string_eq(const char *left, const char *right);
int jayess_args_is_truthy(jayess_args *args);
int jayess_value_is_truthy(jayess_value *value);

jayess_value_kind jayess_value_kind_of(jayess_value *value);
const char *jayess_value_as_string(jayess_value *value);
int jayess_value_as_bool(jayess_value *value);
jayess_object *jayess_value_as_object(jayess_value *value);
jayess_array *jayess_value_as_array(jayess_value *value);

#endif
