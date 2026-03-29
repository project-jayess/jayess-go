#include "jayess_runtime.h"
#include <stdlib.h>
#include <string.h>

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}

jayess_value *jayess_greet(jayess_value *name) {
    const char *prefix = "Hello, ";
    const char *value = jayess_value_as_string(name);
    size_t prefix_len = strlen(prefix);
    size_t value_len = strlen(value);
    char *buffer = (char *)malloc(prefix_len + value_len + 1);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    memcpy(buffer, prefix, prefix_len);
    memcpy(buffer + prefix_len, value, value_len + 1);
    return jayess_value_from_string(buffer);
}

jayess_value *jayess_toggle(jayess_value *value) {
    return jayess_value_from_bool(!jayess_value_as_bool(value));
}

jayess_value *jayess_make_profile(jayess_value *name, jayess_value *score) {
    jayess_object *profile = jayess_object_new();
    jayess_object_set_value(profile, "name", name);
    jayess_object_set_value(profile, "score", score);
    return jayess_value_from_object(profile);
}
