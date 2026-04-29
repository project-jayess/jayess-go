#include "jayess_runtime_internal.h"

/*
 * Container writes currently store aliased jayess_value pointers.
 *
 * They do not:
 * - deep-clone the stored value
 * - consume the caller's box
 * - free a previously stored pointer on replacement
 *
 * That matches the current Jayess ownership model: replacement must preserve
 * any surviving external alias to the old value, even though it means the
 * broad "release previous stored value" proof remains a separate future task.
 */
static void jayess_container_store_alias(jayess_value **slot, jayess_value *value) {
    if (slot == NULL) {
        return;
    }
    *slot = value;
}

static void jayess_object_remove_entry_preserving_alias(jayess_object *object, jayess_object_entry *previous, jayess_object_entry *current) {
    if (object == NULL || current == NULL) {
        return;
    }
    if (previous == NULL) {
        object->head = current->next;
    } else {
        previous->next = current->next;
    }
    if (object->tail == current) {
        object->tail = previous;
    }
    if (jayess_runtime_accounting_state.object_entries > 0) {
        jayess_runtime_accounting_state.object_entries--;
    }
    free(current->key);
    free(current);
}

static void jayess_array_remove_slot_preserving_alias(jayess_array *array, int index) {
    int i;
    if (array == NULL || index < 0 || index >= array->count) {
        return;
    }
    for (i = index + 1; i < array->count; i++) {
        array->values[i - 1] = array->values[i];
    }
    array->count--;
    if (jayess_runtime_accounting_state.array_slots > 0) {
        jayess_runtime_accounting_state.array_slots--;
    }
    if (array->count == 0) {
        free(array->values);
        array->values = NULL;
    } else {
        jayess_value **shrunk = (jayess_value **)realloc(array->values, sizeof(jayess_value *) * (size_t)array->count);
        if (shrunk != NULL) {
            array->values = shrunk;
        }
    }
}

jayess_object *jayess_object_new(void) {
    jayess_object *object = (jayess_object *)malloc(sizeof(jayess_object));
    if (object == NULL) {
        return NULL;
    }
    object->head = NULL;
    object->tail = NULL;
    object->promise_dependents = NULL;
    object->stream_file = NULL;
    object->socket_handle = JAYESS_INVALID_SOCKET;
    object->native_handle = NULL;
    jayess_runtime_accounting_state.objects++;
    return object;
}

static jayess_object_entry *jayess_object_find(jayess_object *object, const char *key) {
    jayess_object_entry *current = object != NULL ? object->head : NULL;
    while (current != NULL) {
        if (jayess_object_entry_matches_string(current, key)) {
            return current;
        }
        current = current->next;
    }
    return NULL;
}

void jayess_object_set_value(jayess_object *object, const char *key, jayess_value *value) {
    jayess_value temp_key;

    if (object == NULL || key == NULL || value == NULL) {
        return;
    }
    temp_key.kind = JAYESS_VALUE_STRING;
    temp_key.as.string_value = (char *)key;
    jayess_object_set_key_value(object, &temp_key, value);
}

jayess_value *jayess_object_get(jayess_object *object, const char *key) {
    jayess_value temp_key;

    if (object == NULL || key == NULL) {
        return NULL;
    }
    temp_key.kind = JAYESS_VALUE_STRING;
    temp_key.as.string_value = (char *)key;
    return jayess_object_get_key_value(object, &temp_key);
}

void jayess_object_delete(jayess_object *object, const char *key) {
    jayess_value temp_key;
    if (object == NULL || key == NULL) {
        return;
    }
    temp_key.kind = JAYESS_VALUE_STRING;
    temp_key.as.string_value = (char *)key;
    jayess_object_delete_key_value(object, &temp_key);
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
        if (current->key != NULL && strncmp(current->key, "__jayess_", 10) != 0) {
            jayess_array_set_value(keys, index++, jayess_value_from_string(current->key));
        }
        current = current->next;
    }
    return keys;
}

int jayess_object_entry_is_symbol(jayess_object_entry *entry) {
    return entry != NULL && entry->key == NULL && entry->key_value != NULL && entry->key_value->kind == JAYESS_VALUE_SYMBOL;
}

int jayess_object_entry_matches_string(jayess_object_entry *entry, const char *key) {
    return entry != NULL && entry->key != NULL && key != NULL && strcmp(entry->key, key) == 0;
}

int jayess_object_entry_matches_value(jayess_object_entry *entry, jayess_value *key) {
    if (entry == NULL || key == NULL) {
        return 0;
    }
    if (key->kind == JAYESS_VALUE_STRING) {
        return jayess_object_entry_matches_string(entry, key->as.string_value);
    }
    if (key->kind == JAYESS_VALUE_SYMBOL) {
        return jayess_object_entry_is_symbol(entry) && jayess_value_eq(entry->key_value, key);
    }
    return 0;
}

jayess_object_entry *jayess_object_find_value(jayess_object *object, jayess_value *key) {
    jayess_object_entry *current = object != NULL ? object->head : NULL;
    while (current != NULL) {
        if (jayess_object_entry_matches_value(current, key)) {
            return current;
        }
        current = current->next;
    }
    return NULL;
}

void jayess_object_set_key_value(jayess_object *object, jayess_value *key, jayess_value *value) {
    jayess_object_entry *entry;
    if (object == NULL || key == NULL || value == NULL) {
        return;
    }
    if (key->kind != JAYESS_VALUE_STRING && key->kind != JAYESS_VALUE_SYMBOL) {
        return;
    }
    entry = jayess_object_find_value(object, key);
    if (entry == NULL) {
        entry = (jayess_object_entry *)malloc(sizeof(jayess_object_entry));
        if (entry == NULL) {
            return;
        }
        entry->key = NULL;
        entry->key_value = NULL;
        entry->value = NULL;
        entry->next = NULL;
        if (key->kind == JAYESS_VALUE_STRING) {
            entry->key = jayess_strdup(key->as.string_value != NULL ? key->as.string_value : "");
        } else {
            entry->key_value = key;
        }
        if (object->tail != NULL) {
            object->tail->next = entry;
        } else {
            object->head = entry;
        }
        object->tail = entry;
        jayess_runtime_accounting_state.object_entries++;
    }
    jayess_container_store_alias(&entry->value, value);
}

jayess_value *jayess_object_get_key_value(jayess_object *object, jayess_value *key) {
    jayess_object_entry *entry = jayess_object_find_value(object, key);
    if (entry == NULL) {
        return NULL;
    }
    return entry->value;
}

void jayess_object_delete_key_value(jayess_object *object, jayess_value *key) {
    jayess_object_entry *current;
    jayess_object_entry *previous;
    if (object == NULL || key == NULL) {
        return;
    }
    previous = NULL;
    current = object->head;
    while (current != NULL) {
        if (jayess_object_entry_matches_value(current, key)) {
            jayess_object_remove_entry_preserving_alias(object, previous, current);
            return;
        }
        previous = current;
        current = current->next;
    }
}

jayess_array *jayess_array_new(void) {
    jayess_array *array = (jayess_array *)malloc(sizeof(jayess_array));
    if (array == NULL) {
        return NULL;
    }
    array->count = 0;
    array->values = NULL;
    jayess_runtime_accounting_state.arrays++;
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
    jayess_runtime_accounting_state.array_slots += (size_t)((index + 1) - array->count);
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
    jayess_container_store_alias(&array->values[index], value);
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
    jayess_array_remove_slot_preserving_alias(array, array->count - 1);
    return value != NULL ? value : jayess_value_undefined();
}

jayess_value *jayess_array_shift_value(jayess_array *array) {
    jayess_value *value;

    if (array == NULL || array->count == 0) {
        return jayess_value_undefined();
    }
    value = array->values[0];
    jayess_array_remove_slot_preserving_alias(array, 0);
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
