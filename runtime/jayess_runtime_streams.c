#include "jayess_runtime_internal.h"

#include <zlib.h>
#include <brotli/encode.h>
#include <brotli/decode.h>

static double jayess_std_stream_number_property(jayess_value *env, const char *key) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return 0.0;
    }
    return jayess_value_to_number(jayess_object_get(env->as.object_value, key));
}

static void jayess_std_stream_set_number_property(jayess_value *env, const char *key, double value) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    jayess_object_set_value(env->as.object_value, key, jayess_value_from_number(value));
}

int jayess_std_stream_bool_property(jayess_value *env, const char *key) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return 0;
    }
    return jayess_value_as_bool(jayess_object_get(env->as.object_value, key));
}

static void jayess_std_stream_set_bool_property(jayess_value *env, const char *key, int value) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    jayess_object_set_value(env->as.object_value, key, jayess_value_from_bool(value));
}

static double jayess_std_stream_high_water_mark(jayess_value *env) {
    double high_water_mark = jayess_std_stream_number_property(env, "writableHighWaterMark");
    if (high_water_mark <= 0.0) {
        high_water_mark = (double)JAYESS_STD_STREAM_DEFAULT_HIGH_WATER_MARK;
    }
    return high_water_mark;
}

static void jayess_std_stream_set_writable_length(jayess_value *env, double length) {
    if (length < 0.0) {
        length = 0.0;
    }
    jayess_std_stream_set_number_property(env, "writableLength", length);
}

int jayess_std_stream_backpressure_note_pending(jayess_value *env, double pending_length) {
    double high_water_mark = jayess_std_stream_high_water_mark(env);
    jayess_std_stream_set_writable_length(env, pending_length);
    if (pending_length > high_water_mark) {
        jayess_std_stream_set_bool_property(env, "writableNeedDrain", 1);
        return 0;
    }
    return 1;
}

void jayess_std_stream_backpressure_maybe_drain(jayess_value *env, double pending_length) {
    double high_water_mark = jayess_std_stream_high_water_mark(env);
    int need_drain = jayess_std_stream_bool_property(env, "writableNeedDrain");
    jayess_std_stream_set_writable_length(env, pending_length);
    if (need_drain && pending_length <= high_water_mark) {
        jayess_std_stream_set_bool_property(env, "writableNeedDrain", 0);
        jayess_std_stream_emit(env, "drain", jayess_value_undefined());
    }
}

FILE *jayess_std_stream_file(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return NULL;
    }
    return env->as.object_value->stream_file;
}

void jayess_std_stream_set_file(jayess_value *env, FILE *file) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        env->as.object_value->stream_file = file;
    }
}

static int jayess_std_stream_event_key(const char *event, char *buffer, size_t buffer_size) {
    int written;
    if (event == NULL || buffer == NULL || buffer_size == 0) {
        return 0;
    }
    written = snprintf(buffer, buffer_size, "__jayess_stream_event_%s", event);
    return written > 0 && (size_t)written < buffer_size;
}

static int jayess_std_stream_once_key(const char *event, char *buffer, size_t buffer_size) {
    int written;
    if (event == NULL || buffer == NULL || buffer_size == 0) {
        return 0;
    }
    written = snprintf(buffer, buffer_size, "__jayess_stream_once_%s", event);
    return written > 0 && (size_t)written < buffer_size;
}

static int jayess_string_has_prefix(const char *value, const char *prefix) {
    size_t prefix_len;
    if (value == NULL || prefix == NULL) {
        return 0;
    }
    prefix_len = strlen(prefix);
    return strncmp(value, prefix, prefix_len) == 0;
}

static void jayess_stream_array_remove_at(jayess_array *array, int index) {
    int i;
    if (array == NULL || index < 0 || index >= array->count) {
        return;
    }
    for (i = index + 1; i < array->count; i++) {
        array->values[i - 1] = array->values[i];
    }
    array->count--;
}

static void jayess_std_stream_add_listener(jayess_value *env, const char *event, jayess_value *callback, int once) {
    char key[128];
    jayess_value *stored;
    jayess_array *listeners;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    if (!(once ? jayess_std_stream_once_key(event, key, sizeof(key)) : jayess_std_stream_event_key(event, key, sizeof(key)))) {
        return;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        listeners = jayess_array_new();
        if (listeners == NULL) {
            return;
        }
        jayess_object_set_value(env->as.object_value, key, jayess_value_from_array(listeners));
    } else {
        listeners = stored->as.array_value;
    }
    jayess_array_push_value(listeners, callback);
}

void jayess_std_stream_on(jayess_value *env, const char *event, jayess_value *callback) {
    jayess_std_stream_add_listener(env, event, callback, 0);
}

void jayess_std_stream_once(jayess_value *env, const char *event, jayess_value *callback) {
    jayess_std_stream_add_listener(env, event, callback, 1);
}

static void jayess_std_stream_off_key(jayess_value *env, const char *key, jayess_value *callback) {
    jayess_value *stored;
    jayess_array *listeners;
    int i;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || key == NULL) {
        return;
    }
    if (callback == NULL || jayess_value_is_nullish(callback)) {
        jayess_object_delete(env->as.object_value, key);
        return;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        return;
    }
    listeners = stored->as.array_value;
    for (i = 0; i < listeners->count; i++) {
        if (listeners->values[i] == callback || jayess_value_eq(listeners->values[i], callback)) {
            jayess_stream_array_remove_at(listeners, i);
            break;
        }
    }
    if (listeners->count == 0) {
        jayess_object_delete(env->as.object_value, key);
    }
}

void jayess_std_stream_off(jayess_value *env, const char *event, jayess_value *callback) {
    char key[128];
    if (jayess_std_stream_event_key(event, key, sizeof(key))) {
        jayess_std_stream_off_key(env, key, callback);
    }
    if (jayess_std_stream_once_key(event, key, sizeof(key))) {
        jayess_std_stream_off_key(env, key, callback);
    }
}

static jayess_array *jayess_std_stream_listeners_for_key(jayess_value *env, const char *key) {
    jayess_value *stored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || key == NULL) {
        return NULL;
    }
    stored = jayess_object_get(env->as.object_value, key);
    if (stored == NULL || stored->kind != JAYESS_VALUE_ARRAY || stored->as.array_value == NULL) {
        return NULL;
    }
    return stored->as.array_value;
}

static jayess_array *jayess_std_stream_listeners(jayess_value *env, const char *event, int once) {
    char key[128];
    if (!(once ? jayess_std_stream_once_key(event, key, sizeof(key)) : jayess_std_stream_event_key(event, key, sizeof(key)))) {
        return NULL;
    }
    return jayess_std_stream_listeners_for_key(env, key);
}

static int jayess_std_stream_listener_count(jayess_value *env, const char *event) {
    jayess_array *listeners = jayess_std_stream_listeners(env, event, 0);
    jayess_array *once_listeners = jayess_std_stream_listeners(env, event, 1);
    int count = 0;
    if (listeners != NULL) {
        count += listeners->count;
    }
    if (once_listeners != NULL) {
        count += once_listeners->count;
    }
    return count;
}

static int jayess_std_stream_event_names_has(jayess_array *names, const char *event) {
    int i;
    if (names == NULL || event == NULL) {
        return 0;
    }
    for (i = 0; i < names->count; i++) {
        jayess_value *name = jayess_array_get(names, i);
        if (name != NULL && name->kind == JAYESS_VALUE_STRING && name->as.string_value != NULL && strcmp(name->as.string_value, event) == 0) {
            return 1;
        }
    }
    return 0;
}

jayess_value *jayess_std_stream_listener_count_method(jayess_value *env, jayess_value *event) {
    char *event_text = jayess_value_stringify(event);
    int count;
    if (event_text == NULL) {
        return jayess_value_from_number(0);
    }
    count = jayess_std_stream_listener_count(env, event_text);
    free(event_text);
    return jayess_value_from_number((double)count);
}

jayess_value *jayess_std_stream_event_names_method(jayess_value *env) {
    const char *event_prefix = "__jayess_stream_event_";
    const char *once_prefix = "__jayess_stream_once_";
    size_t event_prefix_len = strlen(event_prefix);
    size_t once_prefix_len = strlen(once_prefix);
    jayess_array *names = jayess_array_new();
    jayess_object_entry *current;
    if (names == NULL) {
        return jayess_value_from_array(jayess_array_new());
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_from_array(names);
    }
    current = env->as.object_value->head;
    while (current != NULL) {
        const char *event_name = NULL;
        if (jayess_string_has_prefix(current->key, event_prefix)) {
            event_name = current->key + event_prefix_len;
        } else if (jayess_string_has_prefix(current->key, once_prefix)) {
            event_name = current->key + once_prefix_len;
        }
        if (event_name != NULL && event_name[0] != '\0' && !jayess_std_stream_event_names_has(names, event_name)) {
            jayess_array_push_value(names, jayess_value_from_string(event_name));
        }
        current = current->next;
    }
    return jayess_value_from_array(names);
}

jayess_value *jayess_std_stream_off_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    jayess_std_stream_off(env, event_text, callback);
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

void jayess_std_stream_emit(jayess_value *env, const char *event, jayess_value *argument) {
    jayess_array *listeners = jayess_std_stream_listeners(env, event, 0);
    jayess_array *once_listeners;
    char once_key[128];
    int i;
    if (listeners != NULL) {
        for (i = 0; i < listeners->count; i++) {
            jayess_value *callback = listeners->values[i];
            if (callback != NULL && callback->kind == JAYESS_VALUE_FUNCTION) {
                jayess_value_call_one(callback, argument != NULL ? argument : jayess_value_undefined());
                if (jayess_has_exception()) {
                    return;
                }
            }
        }
    }
    if (!jayess_std_stream_once_key(event, once_key, sizeof(once_key))) {
        return;
    }
    once_listeners = jayess_std_stream_listeners_for_key(env, once_key);
    if (once_listeners == NULL) {
        return;
    }
    jayess_object_delete(env->as.object_value, once_key);
    for (i = 0; i < once_listeners->count; i++) {
        jayess_value *callback = once_listeners->values[i];
        if (callback != NULL && callback->kind == JAYESS_VALUE_FUNCTION) {
            jayess_value_call_one(callback, argument != NULL ? argument : jayess_value_undefined());
            if (jayess_has_exception()) {
                return;
            }
        }
    }
}

void jayess_std_stream_emit_error(jayess_value *env, const char *message) {
    jayess_value *error;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    error = jayess_std_error_new(jayess_value_from_string("Error"), jayess_value_from_string(message != NULL ? message : "stream error"));
    jayess_object_set_value(env->as.object_value, "errored", jayess_value_from_bool(1));
    jayess_object_set_value(env->as.object_value, "error", error);
    jayess_std_stream_emit(env, "error", error);
}

void jayess_std_stream_register_error_handler(jayess_value *env, jayess_value *callback) {
    jayess_value *errored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    jayess_std_stream_on(env, "error", callback);
    errored = jayess_object_get(env->as.object_value, "errored");
    if (jayess_value_as_bool(errored)) {
        jayess_value_call_one(callback, jayess_object_get(env->as.object_value, "error"));
    }
}

void jayess_std_stream_register_error_once(jayess_value *env, jayess_value *callback) {
    jayess_value *errored;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    errored = jayess_object_get(env->as.object_value, "errored");
    if (jayess_value_as_bool(errored)) {
        jayess_value_call_one(callback, jayess_object_get(env->as.object_value, "error"));
        return;
    }
    jayess_std_stream_once(env, "error", callback);
}

static void jayess_std_write_stream_register_finish_handler(jayess_value *env, jayess_value *callback) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    jayess_std_stream_on(env, "finish", callback);
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded")) && !jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        jayess_value_call_one(callback, jayess_value_undefined());
    }
}

static void jayess_std_write_stream_register_finish_once(jayess_value *env, jayess_value *callback) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        return;
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded")) && !jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        jayess_value_call_one(callback, jayess_value_undefined());
        return;
    }
    jayess_std_stream_once(env, "finish", callback);
}

static void jayess_std_write_stream_emit_finish(jayess_value *env) {
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return;
    }
    if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "errored"))) {
        return;
    }
    jayess_std_stream_emit(env, "finish", jayess_value_undefined());
}

int jayess_std_stream_requested_size(jayess_value *size_value, int default_size) {
    int requested = default_size;
    if (size_value != NULL && !jayess_value_is_nullish(size_value)) {
        requested = (int)jayess_value_to_number(size_value);
        if (requested <= 0) {
            requested = 1;
        }
        if (requested > 1048576) {
            requested = 1048576;
        }
    }
    return requested;
}

jayess_value *jayess_std_write_stream_write_method(jayess_value *env, jayess_value *value) {
    FILE *file = jayess_std_stream_file(env);
    char *text;
    size_t length;
    int ok;
    jayess_array *bytes;
    int i;
    if (file == NULL) {
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
        }
        return jayess_value_from_bool(0);
    }
    if (value != NULL && value->kind == JAYESS_VALUE_OBJECT && jayess_std_kind_is(value, "Uint8Array")) {
        bytes = jayess_std_bytes_slot(value);
        if (bytes == NULL) {
            return jayess_value_from_bool(0);
        }
        for (i = 0; i < bytes->count; i++) {
            unsigned char byte_value = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, i)) & 255);
            if (fwrite(&byte_value, 1, 1, file) != 1) {
                jayess_std_stream_emit_error(env, "failed to write to stream");
                return jayess_value_from_bool(0);
            }
        }
        if (!jayess_std_stream_backpressure_note_pending(env, (double)bytes->count)) {
            if (fflush(file) != 0) {
                jayess_std_stream_emit_error(env, "failed to flush stream");
                return jayess_value_from_bool(0);
            }
            jayess_std_stream_backpressure_maybe_drain(env, 0.0);
            return jayess_value_from_bool(0);
        }
        return jayess_value_from_bool(1);
    }
    text = jayess_value_stringify(value);
    if (text == NULL) {
        return jayess_value_from_bool(0);
    }
    length = strlen(text);
    ok = fwrite(text, 1, length, file) == length;
    free(text);
    if (!ok) {
        jayess_std_stream_emit_error(env, "failed to write to stream");
        return jayess_value_from_bool(0);
    }
    if (!jayess_std_stream_backpressure_note_pending(env, (double)length)) {
        if (fflush(file) != 0) {
            jayess_std_stream_emit_error(env, "failed to flush stream");
            return jayess_value_from_bool(0);
        }
        jayess_std_stream_backpressure_maybe_drain(env, 0.0);
        return jayess_value_from_bool(0);
    }
    return jayess_value_from_bool(ok);
}

jayess_value *jayess_std_write_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "finish") == 0) {
        jayess_std_write_stream_register_finish_handler(env, callback);
    } else if (strcmp(event_text, "drain") == 0) {
        jayess_std_stream_on(env, "drain", callback);
    }
    free(event_text);
    return env;
}

jayess_value *jayess_std_write_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        free(event_text);
        return jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "finish") == 0) {
        jayess_std_write_stream_register_finish_once(env, callback);
    } else if (strcmp(event_text, "drain") == 0) {
        if (!jayess_std_stream_bool_property(env, "writableNeedDrain")) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "drain", callback);
        }
    }
    free(event_text);
    return env;
}

jayess_value *jayess_std_write_stream_end_method(jayess_value *env) {
    FILE *file = jayess_std_stream_file(env);
    if (file != NULL) {
        if (fflush(file) != 0) {
            jayess_std_stream_emit_error(env, "failed to flush stream");
        }
        jayess_std_stream_backpressure_maybe_drain(env, 0.0);
        fclose(file);
        jayess_std_stream_set_file(env, NULL);
        if (env != NULL && env->kind == JAYESS_VALUE_OBJECT) {
            jayess_object_set_value(env->as.object_value, "closed", jayess_value_from_bool(1));
            jayess_object_set_value(env->as.object_value, "writableEnded", jayess_value_from_bool(1));
            jayess_std_write_stream_emit_finish(env);
        }
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_writable_write(jayess_value *destination, jayess_value *chunk) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT) {
        return jayess_value_from_bool(0);
    }
    if (jayess_std_kind_is(destination, "WriteStream")) {
        return jayess_std_write_stream_write_method(destination, chunk);
    }
    if (jayess_std_kind_is(destination, "CompressionStream")) {
        return jayess_std_compression_stream_write_method(destination, chunk);
    }
    return jayess_value_from_bool(0);
}

jayess_value *jayess_std_writable_end(jayess_value *destination) {
    if (destination == NULL || destination->kind != JAYESS_VALUE_OBJECT) {
        return jayess_value_undefined();
    }
    if (jayess_std_kind_is(destination, "WriteStream")) {
        return jayess_std_write_stream_end_method(destination);
    }
    if (jayess_std_kind_is(destination, "CompressionStream")) {
        return jayess_std_compression_stream_end_method(destination);
    }
    return jayess_value_undefined();
}

static jayess_value *jayess_std_compression_transform(jayess_value *value, int window_bits, int mode) {
    unsigned char *input = NULL;
    size_t input_length = 0;
    z_stream stream;
    unsigned char chunk[4096];
    unsigned char *output = NULL;
    size_t output_length = 0;
    size_t output_capacity = 0;
    int status;
    jayess_value *result = NULL;
    if (!jayess_std_crypto_copy_bytes(value, &input, &input_length)) {
        free(input);
        return jayess_value_undefined();
    }
    memset(&stream, 0, sizeof(stream));
    if (mode == 0) {
        status = deflateInit2(&stream, Z_DEFAULT_COMPRESSION, Z_DEFLATED, window_bits, 8, Z_DEFAULT_STRATEGY);
    } else {
        status = inflateInit2(&stream, window_bits);
    }
    if (status != Z_OK) {
        free(input);
        return jayess_value_undefined();
    }
    stream.next_in = input;
    stream.avail_in = (uInt)input_length;
    do {
        int flush = mode == 0 ? (stream.avail_in == 0 ? Z_FINISH : Z_NO_FLUSH) : Z_NO_FLUSH;
        stream.next_out = chunk;
        stream.avail_out = sizeof(chunk);
        status = mode == 0 ? deflate(&stream, flush) : inflate(&stream, Z_NO_FLUSH);
        if (!(status == Z_OK || status == Z_STREAM_END || (mode == 0 && status == Z_BUF_ERROR))) {
            if (mode == 0) {
                deflateEnd(&stream);
            } else {
                inflateEnd(&stream);
            }
            free(input);
            free(output);
            return jayess_value_undefined();
        }
        {
            size_t produced = sizeof(chunk) - stream.avail_out;
            if (produced > 0) {
                if (output_length + produced > output_capacity) {
                    size_t new_capacity = output_capacity == 0 ? produced : output_capacity * 2;
                    unsigned char *grown;
                    while (new_capacity < output_length + produced) {
                        new_capacity *= 2;
                    }
                    grown = (unsigned char *)realloc(output, new_capacity);
                    if (grown == NULL) {
                        if (mode == 0) {
                            deflateEnd(&stream);
                        } else {
                            inflateEnd(&stream);
                        }
                        free(input);
                        free(output);
                        return jayess_value_undefined();
                    }
                    output = grown;
                    output_capacity = new_capacity;
                }
                memcpy(output + output_length, chunk, produced);
                output_length += produced;
            }
        }
        if (mode == 0 && status == Z_BUF_ERROR && flush == Z_FINISH) {
            status = Z_STREAM_END;
        }
    } while (status != Z_STREAM_END);
    if (mode == 0) {
        deflateEnd(&stream);
    } else {
        inflateEnd(&stream);
    }
    result = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)output_length)));
    if (result != NULL && result->kind == JAYESS_VALUE_OBJECT) {
        jayess_array *bytes = jayess_std_bytes_slot(result);
        size_t i;
        if (bytes != NULL) {
            for (i = 0; i < output_length; i++) {
                jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)output[i]));
            }
        } else {
            result = jayess_value_undefined();
        }
    }
    free(input);
    free(output);
    return result != NULL ? result : jayess_value_undefined();
}

jayess_value *jayess_std_compression_gzip(jayess_value *value) {
    return jayess_std_compression_transform(value, 15 + 16, 0);
}

jayess_value *jayess_std_compression_gunzip(jayess_value *value) {
    return jayess_std_compression_transform(value, 15 + 16, 1);
}

jayess_value *jayess_std_compression_deflate(jayess_value *value) {
    return jayess_std_compression_transform(value, 15, 0);
}

jayess_value *jayess_std_compression_inflate(jayess_value *value) {
    return jayess_std_compression_transform(value, 15, 1);
}

jayess_value *jayess_std_compression_brotli(jayess_value *value) {
    unsigned char *input = NULL;
    size_t input_length = 0;
    size_t encoded_size;
    unsigned char *encoded = NULL;
    jayess_value *result = NULL;
    if (!jayess_std_crypto_copy_bytes(value, &input, &input_length)) {
        free(input);
        return jayess_value_undefined();
    }
    encoded_size = BrotliEncoderMaxCompressedSize(input_length);
    encoded = (unsigned char *)malloc(encoded_size > 0 ? encoded_size : 1);
    if (encoded == NULL) {
        free(input);
        return jayess_value_undefined();
    }
    if (BrotliEncoderCompress(BROTLI_DEFAULT_QUALITY, BROTLI_DEFAULT_WINDOW, BROTLI_MODE_GENERIC, input_length, input, &encoded_size, encoded) == BROTLI_FALSE) {
        free(input);
        free(encoded);
        return jayess_value_undefined();
    }
    result = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)encoded_size)));
    if (result != NULL && result->kind == JAYESS_VALUE_OBJECT) {
        jayess_array *bytes = jayess_std_bytes_slot(result);
        size_t i;
        if (bytes != NULL) {
            for (i = 0; i < encoded_size; i++) {
                jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)encoded[i]));
            }
        } else {
            result = jayess_value_undefined();
        }
    }
    free(input);
    free(encoded);
    return result != NULL ? result : jayess_value_undefined();
}

jayess_value *jayess_std_compression_unbrotli(jayess_value *value) {
    unsigned char *input = NULL;
    size_t input_length = 0;
    size_t output_capacity;
    unsigned char *output = NULL;
    size_t output_length = output_capacity;
    BrotliDecoderResult status;
    jayess_value *result = NULL;
    if (!jayess_std_crypto_copy_bytes(value, &input, &input_length)) {
        free(input);
        return jayess_value_undefined();
    }
    output_capacity = input_length > 0 ? input_length * 6 : 64;
    if (output_capacity < 64) {
        output_capacity = 64;
    }
    output = (unsigned char *)malloc(output_capacity);
    if (output == NULL) {
        free(input);
        return jayess_value_undefined();
    }
    for (;;) {
        output_length = output_capacity;
        status = BrotliDecoderDecompress(input_length, input, &output_length, output);
        if (status == BROTLI_DECODER_RESULT_SUCCESS) {
            result = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)output_length)));
            if (result != NULL && result->kind == JAYESS_VALUE_OBJECT) {
                jayess_array *bytes = jayess_std_bytes_slot(result);
                size_t i;
                if (bytes != NULL) {
                    for (i = 0; i < output_length; i++) {
                        jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)output[i]));
                    }
                } else {
                    result = jayess_value_undefined();
                }
            }
            free(input);
            free(output);
            return result != NULL ? result : jayess_value_undefined();
        }
        if (status != BROTLI_DECODER_RESULT_NEEDS_MORE_OUTPUT || output_capacity > (1u << 26)) {
            free(input);
            free(output);
            return jayess_value_undefined();
        }
        output_capacity *= 2;
        {
            unsigned char *grown = (unsigned char *)realloc(output, output_capacity);
            if (grown == NULL) {
                free(input);
                free(output);
                return jayess_value_undefined();
            }
            output = grown;
        }
    }
}

static jayess_value *jayess_std_compression_stream_new(const char *mode) {
    jayess_object *object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("CompressionStream"));
    jayess_object_set_value(object, "__jayess_compression_mode", jayess_value_from_string(mode != NULL ? mode : ""));
    jayess_object_set_value(object, "__jayess_bytes", jayess_value_from_array(jayess_array_new()));
    jayess_object_set_value(object, "__jayess_compression_offset", jayess_value_from_number(0));
    jayess_object_set_value(object, "readableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableEnded", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableNeedDrain", jayess_value_from_bool(0));
    jayess_object_set_value(object, "writableLength", jayess_value_from_number(0));
    jayess_object_set_value(object, "writableHighWaterMark", jayess_value_from_number(JAYESS_STD_STREAM_DEFAULT_HIGH_WATER_MARK));
    jayess_object_set_value(object, "errored", jayess_value_from_bool(0));
    jayess_object_set_value(object, "error", jayess_value_null());
    return jayess_value_from_object(object);
}

static void jayess_std_compression_stream_mark_ended(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_object_set_value(env->as.object_value, "readableEnded", jayess_value_from_bool(1));
    }
}

static jayess_value *jayess_std_compression_stream_transform(jayess_value *env, jayess_value *value) {
    jayess_value *mode_value;
    char *mode;
    jayess_value *result = NULL;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_undefined();
    }
    mode_value = jayess_object_get(env->as.object_value, "__jayess_compression_mode");
    mode = jayess_value_stringify(mode_value);
    if (mode == NULL) {
        return jayess_value_undefined();
    }
    if (strcmp(mode, "gzip") == 0) {
        result = jayess_std_compression_gzip(value);
    } else if (strcmp(mode, "gunzip") == 0) {
        result = jayess_std_compression_gunzip(value);
    } else if (strcmp(mode, "deflate") == 0) {
        result = jayess_std_compression_deflate(value);
    } else if (strcmp(mode, "inflate") == 0) {
        result = jayess_std_compression_inflate(value);
    } else if (strcmp(mode, "brotli") == 0) {
        result = jayess_std_compression_brotli(value);
    } else if (strcmp(mode, "unbrotli") == 0) {
        result = jayess_std_compression_unbrotli(value);
    }
    free(mode);
    return result != NULL ? result : jayess_value_undefined();
}

jayess_value *jayess_std_compression_stream_write_method(jayess_value *env, jayess_value *value) {
    jayess_value *chunk;
    jayess_array *target_bytes;
    jayess_array *chunk_bytes;
    int offset;
    int pending;
    int i;
    if (env == NULL || env->kind != JAYESS_VALUE_OBJECT || env->as.object_value == NULL) {
        return jayess_value_from_bool(0);
    }
    chunk = jayess_std_compression_stream_transform(env, value);
    if (chunk == NULL || chunk->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(chunk, "Uint8Array")) {
        jayess_std_stream_emit_error(env, "compression stream transform failed");
        return jayess_value_from_bool(0);
    }
    target_bytes = jayess_std_bytes_slot(env);
    chunk_bytes = jayess_std_bytes_slot(chunk);
    if (target_bytes == NULL || chunk_bytes == NULL) {
        jayess_std_stream_emit_error(env, "compression stream buffer is unavailable");
        return jayess_value_from_bool(0);
    }
    for (i = 0; i < chunk_bytes->count; i++) {
        jayess_array_push_value(target_bytes, jayess_array_get(chunk_bytes, i));
    }
    offset = (int)jayess_std_stream_number_property(env, "__jayess_compression_offset");
    pending = target_bytes->count - offset;
    return jayess_value_from_bool(jayess_std_stream_backpressure_note_pending(env, (double)pending));
}

jayess_value *jayess_std_compression_stream_read_bytes_method(jayess_value *env, jayess_value *size_value) {
    jayess_array *bytes = jayess_std_bytes_slot(env);
    int offset;
    int requested;
    int available;
    int count;
    jayess_value *out;
    jayess_array *out_bytes;
    int i;
    if (bytes == NULL) {
        jayess_std_compression_stream_mark_ended(env);
        return jayess_value_undefined();
    }
    offset = (int)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_compression_offset"));
    if (offset >= bytes->count) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded"))) {
            jayess_std_compression_stream_mark_ended(env);
        }
        return jayess_value_undefined();
    }
    requested = jayess_std_stream_requested_size(size_value, bytes->count - offset);
    available = bytes->count - offset;
    count = requested < available ? requested : available;
    out = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)count)));
    out_bytes = jayess_std_bytes_slot(out);
    if (out_bytes == NULL) {
        return jayess_value_undefined();
    }
    for (i = 0; i < count; i++) {
        jayess_array_set_value(out_bytes, i, jayess_array_get(bytes, offset + i));
    }
    jayess_object_set_value(env->as.object_value, "__jayess_compression_offset", jayess_value_from_number((double)(offset + count)));
    jayess_std_stream_backpressure_maybe_drain(env, (double)(bytes->count - (offset + count)));
    if (offset + count >= bytes->count && jayess_value_as_bool(jayess_object_get(env->as.object_value, "writableEnded"))) {
        jayess_std_compression_stream_mark_ended(env);
        jayess_std_stream_emit(env, "end", jayess_value_undefined());
    }
    return out;
}

jayess_value *jayess_std_compression_stream_read_method(jayess_value *env, jayess_value *size_value) {
    jayess_value *chunk = jayess_std_compression_stream_read_bytes_method(env, size_value);
    if (chunk == NULL || chunk->kind == JAYESS_VALUE_UNDEFINED || chunk->kind == JAYESS_VALUE_NULL) {
        return chunk != NULL ? chunk : jayess_value_undefined();
    }
    return jayess_std_uint8_to_string_method(chunk, jayess_value_undefined());
}

jayess_value *jayess_std_compression_stream_pipe_method(jayess_value *env, jayess_value *destination) {
    while (1) {
        jayess_value *chunk = jayess_std_compression_stream_read_bytes_method(env, jayess_value_undefined());
        if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
            break;
        }
        jayess_std_writable_write(destination, chunk);
        if (jayess_has_exception()) {
            break;
        }
    }
    jayess_std_writable_end(destination);
    return destination != NULL ? destination : jayess_value_undefined();
}

jayess_value *jayess_std_compression_stream_end_method(jayess_value *env) {
    if (env != NULL && env->kind == JAYESS_VALUE_OBJECT && env->as.object_value != NULL) {
        jayess_array *bytes = jayess_std_bytes_slot(env);
        int offset = (int)jayess_value_to_number(jayess_object_get(env->as.object_value, "__jayess_compression_offset"));
        jayess_object_set_value(env->as.object_value, "writableEnded", jayess_value_from_bool(1));
        jayess_std_stream_emit(env, "finish", jayess_value_undefined());
        if (bytes == NULL || offset >= bytes->count) {
            jayess_std_compression_stream_mark_ended(env);
            jayess_std_stream_emit(env, "end", jayess_value_undefined());
        }
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_compression_stream_on_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_handler(env, callback);
    } else if (strcmp(event_text, "end") == 0 || strcmp(event_text, "finish") == 0) {
        jayess_std_stream_on(env, strcmp(event_text, "finish") == 0 ? "finish" : "end", callback);
    } else if (strcmp(event_text, "drain") == 0) {
        jayess_std_stream_on(env, "drain", callback);
    } else if (strcmp(event_text, "data") == 0) {
        while (1) {
            jayess_value *chunk = jayess_std_compression_stream_read_method(env, jayess_value_undefined());
            if (chunk == NULL || chunk->kind == JAYESS_VALUE_NULL || chunk->kind == JAYESS_VALUE_UNDEFINED) {
                break;
            }
            jayess_value_call_one(callback, chunk);
            if (jayess_has_exception()) {
                break;
            }
        }
    }
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

jayess_value *jayess_std_compression_stream_once_method(jayess_value *env, jayess_value *event, jayess_value *callback) {
    char *event_text = jayess_value_stringify(event);
    if (event_text == NULL || callback == NULL || callback->kind != JAYESS_VALUE_FUNCTION) {
        free(event_text);
        return env != NULL ? env : jayess_value_undefined();
    }
    if (strcmp(event_text, "error") == 0) {
        jayess_std_stream_register_error_once(env, callback);
    } else if (strcmp(event_text, "end") == 0 || strcmp(event_text, "finish") == 0) {
        if (jayess_value_as_bool(jayess_object_get(env->as.object_value, strcmp(event_text, "finish") == 0 ? "writableEnded" : "readableEnded"))) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, strcmp(event_text, "finish") == 0 ? "finish" : "end", callback);
        }
    } else if (strcmp(event_text, "drain") == 0) {
        if (!jayess_std_stream_bool_property(env, "writableNeedDrain")) {
            jayess_value_call_one(callback, jayess_value_undefined());
        } else {
            jayess_std_stream_once(env, "drain", callback);
        }
    } else if (strcmp(event_text, "data") == 0) {
        jayess_value *chunk = jayess_std_compression_stream_read_method(env, jayess_value_undefined());
        if (chunk != NULL && chunk->kind != JAYESS_VALUE_NULL && chunk->kind != JAYESS_VALUE_UNDEFINED) {
            jayess_value_call_one(callback, chunk);
        }
    }
    free(event_text);
    return env != NULL ? env : jayess_value_undefined();
}

jayess_value *jayess_std_compression_create_gzip_stream(void) {
    return jayess_std_compression_stream_new("gzip");
}

jayess_value *jayess_std_compression_create_gunzip_stream(void) {
    return jayess_std_compression_stream_new("gunzip");
}

jayess_value *jayess_std_compression_create_deflate_stream(void) {
    return jayess_std_compression_stream_new("deflate");
}

jayess_value *jayess_std_compression_create_inflate_stream(void) {
    return jayess_std_compression_stream_new("inflate");
}

jayess_value *jayess_std_compression_create_brotli_stream(void) {
    return jayess_std_compression_stream_new("brotli");
}

jayess_value *jayess_std_compression_create_unbrotli_stream(void) {
    return jayess_std_compression_stream_new("unbrotli");
}
