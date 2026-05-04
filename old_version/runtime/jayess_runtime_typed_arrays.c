#include <string.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <pthread.h>
#endif

#include "jayess_runtime_internal.h"

double jayess_std_typed_array_get_number(jayess_value *target, int index);
void jayess_std_typed_array_set_number(jayess_value *target, int index, double number);
jayess_value *jayess_std_typed_array_slice_values(jayess_value *env, int start, int end, int has_end);

static jayess_buffer_state *jayess_std_buffer_state_new(int length, int shared) {
    jayess_buffer_state *state = (jayess_buffer_state *)calloc(1, sizeof(jayess_buffer_state));
    int i;
    if (state == NULL) {
        return NULL;
    }
    state->bytes = jayess_array_new();
    if (state->bytes == NULL) {
        free(state);
        return NULL;
    }
    state->refcount = 1;
    state->shared = shared ? 1 : 0;
#ifdef _WIN32
    InitializeCriticalSection(&state->lock);
#else
    pthread_mutex_init(&state->lock, NULL);
#endif
    if (length < 0) {
        length = 0;
    }
    for (i = 0; i < length; i++) {
        jayess_array_push_value(state->bytes, jayess_value_from_number(0));
    }
    return state;
}

void jayess_std_buffer_state_retain(jayess_buffer_state *state) {
    if (state != NULL) {
        state->refcount++;
    }
}

void jayess_std_buffer_state_release(jayess_buffer_state *state) {
    if (state == NULL) {
        return;
    }
    if (state->refcount > 1) {
        state->refcount--;
        return;
    }
    if (state->bytes != NULL) {
        jayess_array_free_unshared(state->bytes);
    }
#ifdef _WIN32
    DeleteCriticalSection(&state->lock);
#else
    pthread_mutex_destroy(&state->lock);
#endif
    free(state);
}

jayess_buffer_state *jayess_std_bytes_state(jayess_value *target) {
    const char *kind;
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return NULL;
    }
    kind = jayess_std_typed_array_kind(target);
    if (kind != NULL || jayess_std_kind_is(target, "DataView") || jayess_std_kind_is(target, "ArrayBuffer") || jayess_std_kind_is(target, "SharedArrayBuffer")) {
        return (jayess_buffer_state *)target->as.object_value->native_handle;
    }
    return NULL;
}

jayess_value *jayess_std_buffer_value_from_state(jayess_buffer_state *state) {
    jayess_object *object;
    if (state == NULL) {
        return jayess_value_from_object(NULL);
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_std_buffer_state_retain(state);
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(state->shared ? "SharedArrayBuffer" : "ArrayBuffer"));
    return jayess_value_from_object(object);
}

static int jayess_std_is_typed_array_kind(const char *kind) {
    return kind != NULL && (
        strcmp(kind, "Uint8Array") == 0 ||
        strcmp(kind, "Int8Array") == 0 ||
        strcmp(kind, "Uint16Array") == 0 ||
        strcmp(kind, "Int16Array") == 0 ||
        strcmp(kind, "Uint32Array") == 0 ||
        strcmp(kind, "Int32Array") == 0 ||
        strcmp(kind, "Float32Array") == 0 ||
        strcmp(kind, "Float64Array") == 0
    );
}

static int jayess_std_typed_array_element_size(const char *kind) {
    if (kind == NULL) {
        return 0;
    }
    if (strcmp(kind, "Uint8Array") == 0 || strcmp(kind, "Int8Array") == 0) {
        return 1;
    }
    if (strcmp(kind, "Uint16Array") == 0 || strcmp(kind, "Int16Array") == 0) {
        return 2;
    }
    if (strcmp(kind, "Uint32Array") == 0 || strcmp(kind, "Int32Array") == 0 || strcmp(kind, "Float32Array") == 0) {
        return 4;
    }
    if (strcmp(kind, "Float64Array") == 0) {
        return 8;
    }
    return 0;
}

static int jayess_std_base64_decode_char(char ch) {
    if (ch >= 'A' && ch <= 'Z') {
        return ch - 'A';
    }
    if (ch >= 'a' && ch <= 'z') {
        return ch - 'a' + 26;
    }
    if (ch >= '0' && ch <= '9') {
        return ch - '0' + 52;
    }
    if (ch == '+') {
        return 62;
    }
    if (ch == '/') {
        return 63;
    }
    return -1;
}

static int jayess_std_uint8_clamped_index(jayess_value *value, int length, int default_value) {
    int index;
    if (value == NULL || jayess_value_is_nullish(value)) {
        return default_value;
    }
    index = (int)jayess_value_to_number(value);
    if (index < 0) {
        index += length;
    }
    if (index < 0) {
        index = 0;
    }
    if (index > length) {
        index = length;
    }
    return index;
}

static int jayess_std_bytes_read(jayess_array *bytes, int offset) {
    if (bytes == NULL || offset < 0 || offset >= bytes->count) {
        return 0;
    }
    return (int)jayess_value_to_number(jayess_array_get(bytes, offset)) & 255;
}

static void jayess_std_bytes_write(jayess_array *bytes, int offset, int value) {
    if (bytes == NULL || offset < 0 || offset >= bytes->count) {
        return;
    }
    jayess_array_set_value(bytes, offset, jayess_value_from_number((double)(value & 255)));
}

static unsigned int jayess_std_data_view_read_u32(jayess_array *bytes, int offset, int little_endian) {
    unsigned int b0 = (unsigned int)jayess_std_bytes_read(bytes, offset);
    unsigned int b1 = (unsigned int)jayess_std_bytes_read(bytes, offset + 1);
    unsigned int b2 = (unsigned int)jayess_std_bytes_read(bytes, offset + 2);
    unsigned int b3 = (unsigned int)jayess_std_bytes_read(bytes, offset + 3);
    return little_endian ? (b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)) : ((b0 << 24) | (b1 << 16) | (b2 << 8) | b3);
}

static void jayess_std_data_view_write_u32(jayess_array *bytes, int offset, unsigned int number, int little_endian) {
    if (little_endian) {
        jayess_std_bytes_write(bytes, offset, (int)(number & 255));
        jayess_std_bytes_write(bytes, offset + 1, (int)((number >> 8) & 255));
        jayess_std_bytes_write(bytes, offset + 2, (int)((number >> 16) & 255));
        jayess_std_bytes_write(bytes, offset + 3, (int)((number >> 24) & 255));
    } else {
        jayess_std_bytes_write(bytes, offset, (int)((number >> 24) & 255));
        jayess_std_bytes_write(bytes, offset + 1, (int)((number >> 16) & 255));
        jayess_std_bytes_write(bytes, offset + 2, (int)((number >> 8) & 255));
        jayess_std_bytes_write(bytes, offset + 3, (int)(number & 255));
    }
}

static unsigned long long jayess_std_data_view_read_u64(jayess_array *bytes, int offset, int little_endian) {
    unsigned long long value = 0;
    int i;
    if (little_endian) {
        for (i = 7; i >= 0; i--) {
            value = (value << 8) | (unsigned long long)jayess_std_bytes_read(bytes, offset + i);
        }
    } else {
        for (i = 0; i < 8; i++) {
            value = (value << 8) | (unsigned long long)jayess_std_bytes_read(bytes, offset + i);
        }
    }
    return value;
}

static void jayess_std_data_view_write_u64(jayess_array *bytes, int offset, unsigned long long number, int little_endian) {
    int i;
    if (little_endian) {
        for (i = 0; i < 8; i++) {
            jayess_std_bytes_write(bytes, offset + i, (int)((number >> (i * 8)) & 255ULL));
        }
    } else {
        for (i = 0; i < 8; i++) {
            jayess_std_bytes_write(bytes, offset + i, (int)((number >> ((7 - i) * 8)) & 255ULL));
        }
    }
}

static jayess_buffer_state *jayess_std_shared_bytes_state(jayess_value *target) {
    jayess_buffer_state *state = jayess_std_bytes_state(target);
    if (state == NULL || !state->shared) {
        return NULL;
    }
    return state;
}

int jayess_std_byte_length(jayess_value *target) {
    jayess_array *bytes = jayess_std_bytes_slot(target);
    return bytes != NULL ? bytes->count : 0;
}

static int jayess_std_byte_read(jayess_value *target, int offset) {
    jayess_buffer_state *shared = jayess_std_shared_bytes_state(target);
    jayess_array *bytes = jayess_std_bytes_slot(target);
    int value = 0;
    if (bytes == NULL || offset < 0 || offset >= bytes->count) {
        return 0;
    }
    if (shared != NULL) {
#ifdef _WIN32
        EnterCriticalSection(&shared->lock);
#else
        pthread_mutex_lock(&shared->lock);
#endif
    }
    value = (int)jayess_value_to_number(jayess_array_get(bytes, offset)) & 255;
    if (shared != NULL) {
#ifdef _WIN32
        LeaveCriticalSection(&shared->lock);
#else
        pthread_mutex_unlock(&shared->lock);
#endif
    }
    return value;
}

static void jayess_std_byte_write(jayess_value *target, int offset, int value) {
    jayess_buffer_state *shared = jayess_std_shared_bytes_state(target);
    jayess_array *bytes = jayess_std_bytes_slot(target);
    if (bytes == NULL || offset < 0 || offset >= bytes->count) {
        return;
    }
    if (shared != NULL) {
#ifdef _WIN32
        EnterCriticalSection(&shared->lock);
#else
        pthread_mutex_lock(&shared->lock);
#endif
    }
    jayess_array_set_value(bytes, offset, jayess_value_from_number((double)(value & 255)));
    if (shared != NULL) {
#ifdef _WIN32
        LeaveCriticalSection(&shared->lock);
#else
        pthread_mutex_unlock(&shared->lock);
#endif
    }
}

static unsigned int jayess_std_data_view_read_u32_target(jayess_value *target, int offset, int little_endian) {
    unsigned int b0 = (unsigned int)jayess_std_byte_read(target, offset);
    unsigned int b1 = (unsigned int)jayess_std_byte_read(target, offset + 1);
    unsigned int b2 = (unsigned int)jayess_std_byte_read(target, offset + 2);
    unsigned int b3 = (unsigned int)jayess_std_byte_read(target, offset + 3);
    return little_endian ? (b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)) : ((b0 << 24) | (b1 << 16) | (b2 << 8) | b3);
}

static void jayess_std_data_view_write_u32_target(jayess_value *target, int offset, unsigned int number, int little_endian) {
    if (little_endian) {
        jayess_std_byte_write(target, offset, (int)(number & 255));
        jayess_std_byte_write(target, offset + 1, (int)((number >> 8) & 255));
        jayess_std_byte_write(target, offset + 2, (int)((number >> 16) & 255));
        jayess_std_byte_write(target, offset + 3, (int)((number >> 24) & 255));
    } else {
        jayess_std_byte_write(target, offset, (int)((number >> 24) & 255));
        jayess_std_byte_write(target, offset + 1, (int)((number >> 16) & 255));
        jayess_std_byte_write(target, offset + 2, (int)((number >> 8) & 255));
        jayess_std_byte_write(target, offset + 3, (int)(number & 255));
    }
}

static unsigned long long jayess_std_data_view_read_u64_target(jayess_value *target, int offset, int little_endian) {
    unsigned long long value = 0;
    int i;
    if (little_endian) {
        for (i = 7; i >= 0; i--) {
            value = (value << 8) | (unsigned long long)jayess_std_byte_read(target, offset + i);
        }
    } else {
        for (i = 0; i < 8; i++) {
            value = (value << 8) | (unsigned long long)jayess_std_byte_read(target, offset + i);
        }
    }
    return value;
}

static void jayess_std_data_view_write_u64_target(jayess_value *target, int offset, unsigned long long number, int little_endian) {
    int i;
    if (little_endian) {
        for (i = 0; i < 8; i++) {
            jayess_std_byte_write(target, offset + i, (int)((number >> (i * 8)) & 255ULL));
        }
    } else {
        for (i = 0; i < 8; i++) {
            jayess_std_byte_write(target, offset + i, (int)((number >> ((7 - i) * 8)) & 255ULL));
        }
    }
}

jayess_value *jayess_std_array_buffer_new(jayess_value *length_value) {
    jayess_object *object = jayess_object_new();
    jayess_buffer_state *state;
    int length = (int)jayess_value_to_number(length_value);
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    state = jayess_std_buffer_state_new(length, 0);
    if (state == NULL) {
        free(object);
        return jayess_value_from_object(NULL);
    }
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("ArrayBuffer"));
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_shared_array_buffer_new(jayess_value *length_value) {
    jayess_object *object = jayess_object_new();
    jayess_buffer_state *state;
    int length = (int)jayess_value_to_number(length_value);
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    state = jayess_std_buffer_state_new(length, 1);
    if (state == NULL) {
        free(object);
        return jayess_value_from_object(NULL);
    }
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("SharedArrayBuffer"));
    return jayess_value_from_object(object);
}

const char *jayess_std_typed_array_kind(jayess_value *target) {
    jayess_value *kind_value;
    if (target == NULL || target->kind != JAYESS_VALUE_OBJECT || target->as.object_value == NULL) {
        return NULL;
    }
    kind_value = jayess_object_get(target->as.object_value, "__jayess_std_kind");
    if (kind_value == NULL || kind_value->kind != JAYESS_VALUE_STRING) {
        return NULL;
    }
    return jayess_std_is_typed_array_kind(kind_value->as.string_value) ? kind_value->as.string_value : NULL;
}

int jayess_std_is_typed_array(jayess_value *target) {
    return jayess_std_typed_array_kind(target) != NULL;
}

int jayess_std_typed_array_length_from_bytes(jayess_array *bytes, const char *kind) {
    int size = jayess_std_typed_array_element_size(kind);
    if (bytes == NULL || size <= 0) {
        return 0;
    }
    return bytes->count / size;
}

jayess_value *jayess_std_typed_array_new(const char *kind, jayess_value *source) {
    jayess_object *object = jayess_object_new();
    jayess_value *buffer = NULL;
    jayess_buffer_state *state = NULL;
    jayess_array *bytes = NULL;
    int element_size = jayess_std_typed_array_element_size(kind);
    int length = 0;
    int i;
    int owned_buffer = 0;
    jayess_value view_value;
    if (element_size <= 0) {
        return jayess_value_from_object(NULL);
    }
    if (source != NULL && source->kind == JAYESS_VALUE_OBJECT && (jayess_std_kind_is(source, "ArrayBuffer") || jayess_std_kind_is(source, "SharedArrayBuffer"))) {
        state = jayess_std_bytes_state(source);
        bytes = jayess_std_bytes_slot(source);
        length = jayess_std_typed_array_length_from_bytes(bytes, kind);
        jayess_std_buffer_state_retain(state);
    } else if (jayess_std_is_typed_array(source)) {
        int source_length = jayess_value_array_length(source);
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)(source_length * element_size)));
        if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
            owned_buffer = 1;
            state = jayess_std_bytes_state(buffer);
            bytes = jayess_std_bytes_slot(buffer);
            length = source_length;
            jayess_std_buffer_state_retain(state);
        }
    } else if (source != NULL && source->kind == JAYESS_VALUE_ARRAY && source->as.array_value != NULL) {
        length = source->as.array_value->count;
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)(length * element_size)));
        if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
            owned_buffer = 1;
            state = jayess_std_bytes_state(buffer);
            bytes = jayess_std_bytes_slot(buffer);
            jayess_std_buffer_state_retain(state);
        }
    } else {
        length = (int)jayess_value_to_number(source);
        if (length < 0) {
            length = 0;
        }
        buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)(length * element_size)));
        if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
            owned_buffer = 1;
            state = jayess_std_bytes_state(buffer);
            bytes = jayess_std_bytes_slot(buffer);
            jayess_std_buffer_state_retain(state);
        }
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(kind));
    view_value.kind = JAYESS_VALUE_OBJECT;
    view_value.as.object_value = object;
    if (source != NULL && jayess_std_is_typed_array(source)) {
        for (i = 0; i < length; i++) {
            jayess_std_typed_array_set_number(&view_value, i, jayess_std_typed_array_get_number(source, i));
        }
    } else if (source != NULL && source->kind == JAYESS_VALUE_ARRAY && source->as.array_value != NULL) {
        for (i = 0; i < length; i++) {
            jayess_std_typed_array_set_number(&view_value, i, jayess_value_to_number(jayess_array_get(source->as.array_value, i)));
        }
    }
    if (owned_buffer && buffer != NULL) {
        jayess_value_free_unshared(buffer);
    }
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_int8_array_new(jayess_value *source) { return jayess_std_typed_array_new("Int8Array", source); }
jayess_value *jayess_std_uint8_array_new(jayess_value *source) { return jayess_std_typed_array_new("Uint8Array", source); }
jayess_value *jayess_std_uint8_array_from_string(jayess_value *source, jayess_value *encoding) {
    char *text = jayess_value_stringify(source);
    jayess_value *out;
    jayess_array *bytes;
    size_t count = 0;
    size_t i;
    if (text == NULL) {
        return jayess_std_uint8_array_new(jayess_value_from_number(0));
    }
    if (jayess_std_bytes_encoding_is_hex(encoding)) {
        size_t len = strlen(text);
        count = len / 2;
        out = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)count)));
        bytes = jayess_std_bytes_slot(out);
        if (bytes == NULL) {
            free(text);
            return jayess_value_undefined();
        }
        for (i = 0; i < count; i++) {
            char pair[3];
            unsigned long value;
            pair[0] = text[i * 2];
            pair[1] = text[(i * 2) + 1];
            pair[2] = '\0';
            value = strtoul(pair, NULL, 16);
            jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)(value & 255UL)));
        }
        free(text);
        return out;
    }
    if (jayess_std_bytes_encoding_is_base64(encoding)) {
        size_t len = strlen(text);
        size_t capacity = (len / 4) * 3 + 3;
        unsigned char *decoded = (unsigned char *)malloc(capacity > 0 ? capacity : 1);
        size_t out_len = 0;
        if (decoded == NULL) {
            free(text);
            return jayess_value_undefined();
        }
        for (i = 0; i < len;) {
            int vals[4];
            int j;
            int pad = 0;
            for (j = 0; j < 4;) {
                char ch = text[i++];
                if (ch == '\0') {
                    vals[j++] = -2;
                    pad++;
                    continue;
                }
                if (ch == '=') {
                    vals[j++] = -2;
                    pad++;
                    continue;
                }
                if (ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t') {
                    continue;
                }
                vals[j++] = jayess_std_base64_decode_char(ch);
            }
            if (vals[0] < 0 || vals[1] < 0) {
                free(decoded);
                free(text);
                return jayess_std_uint8_array_new(jayess_value_from_number(0));
            }
            decoded[out_len++] = (unsigned char)(((vals[0] & 63) << 2) | ((vals[1] & 48) >> 4));
            if (vals[2] >= 0) {
                decoded[out_len++] = (unsigned char)(((vals[1] & 15) << 4) | ((vals[2] & 60) >> 2));
                if (vals[3] >= 0) {
                    decoded[out_len++] = (unsigned char)(((vals[2] & 3) << 6) | (vals[3] & 63));
                }
            }
            if (pad > 0) {
                break;
            }
        }
        out = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)out_len)));
        bytes = jayess_std_bytes_slot(out);
        if (bytes == NULL) {
            free(decoded);
            free(text);
            return jayess_value_undefined();
        }
        for (i = 0; i < out_len; i++) {
            jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)decoded[i]));
        }
        free(decoded);
        free(text);
        return out;
    }
    count = strlen(text);
    out = jayess_std_uint8_array_new(jayess_std_array_buffer_new(jayess_value_from_number((double)count)));
    bytes = jayess_std_bytes_slot(out);
    if (bytes == NULL) {
        free(text);
        return jayess_value_undefined();
    }
    for (i = 0; i < count; i++) {
        jayess_array_set_value(bytes, (int)i, jayess_value_from_number((double)((unsigned char)text[i])));
    }
    free(text);
    return out;
}
jayess_value *jayess_std_uint16_array_new(jayess_value *source) { return jayess_std_typed_array_new("Uint16Array", source); }
jayess_value *jayess_std_int16_array_new(jayess_value *source) { return jayess_std_typed_array_new("Int16Array", source); }
jayess_value *jayess_std_uint32_array_new(jayess_value *source) { return jayess_std_typed_array_new("Uint32Array", source); }
jayess_value *jayess_std_int32_array_new(jayess_value *source) { return jayess_std_typed_array_new("Int32Array", source); }
jayess_value *jayess_std_float32_array_new(jayess_value *source) { return jayess_std_typed_array_new("Float32Array", source); }
jayess_value *jayess_std_float64_array_new(jayess_value *source) { return jayess_std_typed_array_new("Float64Array", source); }

jayess_value *jayess_std_data_view_new(jayess_value *buffer) {
    jayess_object *object = jayess_object_new();
    jayess_buffer_state *state = NULL;
    int owned_buffer = 0;
    if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT && (jayess_std_kind_is(buffer, "ArrayBuffer") || jayess_std_kind_is(buffer, "SharedArrayBuffer"))) {
        state = jayess_std_bytes_state(buffer);
        jayess_std_buffer_state_retain(state);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    if (state == NULL) {
        buffer = jayess_std_array_buffer_new(jayess_value_from_number(0));
        if (buffer != NULL && buffer->kind == JAYESS_VALUE_OBJECT) {
            owned_buffer = 1;
            state = jayess_std_bytes_state(buffer);
            jayess_std_buffer_state_retain(state);
        }
    }
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("DataView"));
    if (owned_buffer && buffer != NULL) {
        jayess_value_free_unshared(buffer);
    }
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_uint8_array_from_bytes(const unsigned char *bytes, size_t length) {
    jayess_value *buffer = jayess_std_array_buffer_new(jayess_value_from_number((double)length));
    jayess_value *view = jayess_std_uint8_array_new(buffer);
    jayess_array *out = jayess_std_bytes_slot(view);
    size_t i;
    if (out == NULL) {
        return view;
    }
    for (i = 0; i < length; i++) {
        jayess_array_set_value(out, (int)i, jayess_value_from_number((double)bytes[i]));
    }
    return view;
}

jayess_value *jayess_std_data_view_get_uint8_method(jayess_value *env, jayess_value *offset_value) {
    int offset = (int)jayess_value_to_number(offset_value);
    return jayess_value_from_number((double)jayess_std_byte_read(env, offset));
}

jayess_value *jayess_std_data_view_set_uint8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value) {
    int offset = (int)jayess_value_to_number(offset_value);
    int byte_value = (int)jayess_value_to_number(value) & 255;
    jayess_std_byte_write(env, offset, byte_value);
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_int8_method(jayess_value *env, jayess_value *offset_value) {
    int offset = (int)jayess_value_to_number(offset_value);
    int value = jayess_std_byte_read(env, offset);
    if (value >= 128) {
        value -= 256;
    }
    return jayess_value_from_number((double)value);
}

jayess_value *jayess_std_data_view_set_int8_method(jayess_value *env, jayess_value *offset_value, jayess_value *value) {
    int offset = (int)jayess_value_to_number(offset_value);
    int byte_value = (int)jayess_value_to_number(value);
    jayess_std_byte_write(env, offset, byte_value);
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    int b0 = jayess_std_byte_read(env, offset);
    int b1 = jayess_std_byte_read(env, offset + 1);
    int value = jayess_value_as_bool(little_endian) ? (b0 | (b1 << 8)) : ((b0 << 8) | b1);
    return jayess_value_from_number((double)value);
}

jayess_value *jayess_std_data_view_set_uint16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    int number = (int)jayess_value_to_number(value) & 65535;
    if (jayess_value_as_bool(little_endian)) {
        jayess_std_byte_write(env, offset, number & 255);
        jayess_std_byte_write(env, offset + 1, (number >> 8) & 255);
    } else {
        jayess_std_byte_write(env, offset, (number >> 8) & 255);
        jayess_std_byte_write(env, offset + 1, number & 255);
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    int b0 = jayess_std_byte_read(env, offset);
    int b1 = jayess_std_byte_read(env, offset + 1);
    int value = jayess_value_as_bool(little_endian) ? (b0 | (b1 << 8)) : ((b0 << 8) | b1);
    if (value >= 32768) {
        value -= 65536;
    }
    return jayess_value_from_number((double)value);
}

jayess_value *jayess_std_data_view_set_int16_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    int number = (int)jayess_value_to_number(value) & 65535;
    if (jayess_value_as_bool(little_endian)) {
        jayess_std_byte_write(env, offset, number & 255);
        jayess_std_byte_write(env, offset + 1, (number >> 8) & 255);
    } else {
        jayess_std_byte_write(env, offset, (number >> 8) & 255);
        jayess_std_byte_write(env, offset + 1, number & 255);
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    unsigned int value = jayess_std_data_view_read_u32_target(env, offset, jayess_value_as_bool(little_endian));
    return jayess_value_from_number((double)value);
}

jayess_value *jayess_std_data_view_set_uint32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    unsigned int number = (unsigned int)jayess_value_to_number(value);
    jayess_std_data_view_write_u32_target(env, offset, number, jayess_value_as_bool(little_endian));
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    unsigned int value = jayess_std_data_view_read_u32_target(env, offset, jayess_value_as_bool(little_endian));
    long long signed_value = value >= 2147483648U ? (long long)value - 4294967296LL : (long long)value;
    return jayess_value_from_number((double)signed_value);
}

jayess_value *jayess_std_data_view_set_int32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    int signed_number = (int)jayess_value_to_number(value);
    unsigned int number = (unsigned int)signed_number;
    jayess_std_data_view_write_u32_target(env, offset, number, jayess_value_as_bool(little_endian));
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    unsigned int bits = jayess_std_data_view_read_u32_target(env, offset, jayess_value_as_bool(little_endian));
    float value = 0.0f;
    memcpy(&value, &bits, sizeof(value));
    return jayess_value_from_number((double)value);
}

jayess_value *jayess_std_data_view_set_float32_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    float number = (float)jayess_value_to_number(value);
    unsigned int bits = 0;
    memcpy(&bits, &number, sizeof(bits));
    jayess_std_data_view_write_u32_target(env, offset, bits, jayess_value_as_bool(little_endian));
    return jayess_value_undefined();
}

jayess_value *jayess_std_data_view_get_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    unsigned long long bits = jayess_std_data_view_read_u64_target(env, offset, jayess_value_as_bool(little_endian));
    double value = 0.0;
    memcpy(&value, &bits, sizeof(value));
    return jayess_value_from_number(value);
}

jayess_value *jayess_std_data_view_set_float64_method(jayess_value *env, jayess_value *offset_value, jayess_value *value, jayess_value *little_endian) {
    int offset = (int)jayess_value_to_number(offset_value);
    double number = jayess_value_to_number(value);
    unsigned long long bits = 0;
    memcpy(&bits, &number, sizeof(bits));
    jayess_std_data_view_write_u64_target(env, offset, bits, jayess_value_as_bool(little_endian));
    return jayess_value_undefined();
}

double jayess_std_typed_array_get_number(jayess_value *target, int index) {
    const char *kind = jayess_std_typed_array_kind(target);
    int size = jayess_std_typed_array_element_size(kind);
    int offset;
    if (size <= 0 || index < 0 || index >= jayess_value_array_length(target)) {
        return 0.0;
    }
    offset = index * size;
    if (strcmp(kind, "Uint8Array") == 0) {
        return (double)jayess_std_byte_read(target, offset);
    }
    if (strcmp(kind, "Int8Array") == 0) {
        int value = jayess_std_byte_read(target, offset);
        return (double)(value >= 128 ? value - 256 : value);
    }
    if (strcmp(kind, "Uint16Array") == 0) {
        int b0 = jayess_std_byte_read(target, offset);
        int b1 = jayess_std_byte_read(target, offset + 1);
        return (double)(b0 | (b1 << 8));
    }
    if (strcmp(kind, "Int16Array") == 0) {
        int value = jayess_std_byte_read(target, offset) | (jayess_std_byte_read(target, offset + 1) << 8);
        if (value >= 32768) {
            value -= 65536;
        }
        return (double)value;
    }
    if (strcmp(kind, "Uint32Array") == 0) {
        return (double)jayess_std_data_view_read_u32_target(target, offset, 1);
    }
    if (strcmp(kind, "Int32Array") == 0) {
        unsigned int value = jayess_std_data_view_read_u32_target(target, offset, 1);
        long long signed_value = value >= 2147483648U ? (long long)value - 4294967296LL : (long long)value;
        return (double)signed_value;
    }
    if (strcmp(kind, "Float32Array") == 0) {
        unsigned int bits = jayess_std_data_view_read_u32_target(target, offset, 1);
        float value = 0.0f;
        memcpy(&value, &bits, sizeof(value));
        return (double)value;
    }
    if (strcmp(kind, "Float64Array") == 0) {
        unsigned long long bits = jayess_std_data_view_read_u64_target(target, offset, 1);
        double value = 0.0;
        memcpy(&value, &bits, sizeof(value));
        return value;
    }
    return 0.0;
}

void jayess_std_typed_array_set_number(jayess_value *target, int index, double number) {
    const char *kind = jayess_std_typed_array_kind(target);
    int size = jayess_std_typed_array_element_size(kind);
    int offset;
    if (size <= 0 || index < 0 || index >= jayess_value_array_length(target)) {
        return;
    }
    offset = index * size;
    if (strcmp(kind, "Uint8Array") == 0) {
        jayess_std_byte_write(target, offset, (int)number & 255);
        return;
    }
    if (strcmp(kind, "Int8Array") == 0) {
        jayess_std_byte_write(target, offset, (int)number);
        return;
    }
    if (strcmp(kind, "Uint16Array") == 0 || strcmp(kind, "Int16Array") == 0) {
        int value = (int)number;
        jayess_std_byte_write(target, offset, value & 255);
        jayess_std_byte_write(target, offset + 1, (value >> 8) & 255);
        return;
    }
    if (strcmp(kind, "Uint32Array") == 0) {
        jayess_std_data_view_write_u32_target(target, offset, (unsigned int)number, 1);
        return;
    }
    if (strcmp(kind, "Int32Array") == 0) {
        jayess_std_data_view_write_u32_target(target, offset, (unsigned int)((int)number), 1);
        return;
    }
    if (strcmp(kind, "Float32Array") == 0) {
        float value = (float)number;
        unsigned int bits = 0;
        memcpy(&bits, &value, sizeof(bits));
        jayess_std_data_view_write_u32_target(target, offset, bits, 1);
        return;
    }
    if (strcmp(kind, "Float64Array") == 0) {
        unsigned long long bits = 0;
        memcpy(&bits, &number, sizeof(bits));
        jayess_std_data_view_write_u64_target(target, offset, bits, 1);
    }
}

jayess_value *jayess_std_typed_array_fill_method(jayess_value *env, jayess_value *value) {
    int length = jayess_value_array_length(env);
    int i;
    double number = jayess_value_to_number(value);
    for (i = 0; i < length; i++) {
        jayess_std_typed_array_set_number(env, i, number);
    }
    return env != NULL ? env : jayess_value_undefined();
}

jayess_value *jayess_std_typed_array_includes_method(jayess_value *env, jayess_value *value) {
    int length = jayess_value_array_length(env);
    double needle = jayess_value_to_number(value);
    int i;
    for (i = 0; i < length; i++) {
        if (jayess_std_typed_array_get_number(env, i) == needle) {
            return jayess_value_from_bool(1);
        }
    }
    return jayess_value_from_bool(0);
}

jayess_value *jayess_std_typed_array_index_of_method(jayess_value *env, jayess_value *needle) {
    int length = jayess_value_array_length(env);
    double value = jayess_value_to_number(needle);
    int i;
    for (i = 0; i < length; i++) {
        if (jayess_std_typed_array_get_number(env, i) == value) {
            return jayess_value_from_number((double)i);
        }
    }
    return jayess_value_from_number(-1);
}

jayess_value *jayess_std_typed_array_set_method(jayess_value *env, jayess_value *source, jayess_value *offset_value) {
    int length = jayess_value_array_length(env);
    int offset = jayess_std_uint8_clamped_index(offset_value, length, 0);
    int count = 0;
    int i;
    if (source == NULL) {
        return jayess_value_undefined();
    }
    if (jayess_std_is_typed_array(source)) {
        count = jayess_value_array_length(source);
        if (count > length - offset) {
            count = length - offset;
        }
        for (i = 0; i < count; i++) {
            jayess_std_typed_array_set_number(env, offset + i, jayess_std_typed_array_get_number(source, i));
        }
        return jayess_value_undefined();
    }
    if (source->kind == JAYESS_VALUE_ARRAY && source->as.array_value != NULL) {
        count = source->as.array_value->count;
        if (count > length - offset) {
            count = length - offset;
        }
        for (i = 0; i < count; i++) {
            jayess_std_typed_array_set_number(env, offset + i, jayess_value_to_number(jayess_array_get(source->as.array_value, i)));
        }
    }
    return jayess_value_undefined();
}

jayess_value *jayess_std_typed_array_slice_values(jayess_value *env, int start, int end, int has_end) {
    int length = jayess_value_array_length(env);
    jayess_array *values = jayess_array_new();
    const char *kind = jayess_std_typed_array_kind(env);
    int i;
    if (start < 0) {
        start = length + start;
    }
    if (start < 0) {
        start = 0;
    }
    if (start > length) {
        start = length;
    }
    if (!has_end) {
        end = length;
    } else if (end < 0) {
        end = length + end;
    }
    if (end < start) {
        end = start;
    }
    if (end > length) {
        end = length;
    }
    for (i = start; i < end; i++) {
        jayess_array_push_value(values, jayess_value_from_number(jayess_std_typed_array_get_number(env, i)));
    }
    return jayess_std_typed_array_new(kind != NULL ? kind : "Uint8Array", jayess_value_from_array(values));
}

jayess_value *jayess_std_typed_array_slice_method(jayess_value *env, jayess_value *start_value, jayess_value *end_value) {
    int length = jayess_value_array_length(env);
    int start = jayess_std_uint8_clamped_index(start_value, length, 0);
    int end = jayess_std_uint8_clamped_index(end_value, length, length);
    return jayess_std_typed_array_slice_values(env, start, end, 1);
}

static int jayess_atomics_is_supported_kind(const char *kind) {
    return kind != NULL && (
        strcmp(kind, "Int8Array") == 0 ||
        strcmp(kind, "Uint8Array") == 0 ||
        strcmp(kind, "Int16Array") == 0 ||
        strcmp(kind, "Uint16Array") == 0 ||
        strcmp(kind, "Int32Array") == 0 ||
        strcmp(kind, "Uint32Array") == 0
    );
}

static jayess_buffer_state *jayess_atomics_state(jayess_value *target, int *index_out, const char **kind_out) {
    const char *kind = jayess_std_typed_array_kind(target);
    jayess_buffer_state *state;
    int index;
    if (!jayess_atomics_is_supported_kind(kind)) {
        jayess_throw(jayess_type_error_value("Atomics requires an integer typed array"));
        return NULL;
    }
    state = jayess_std_shared_bytes_state(target);
    if (state == NULL) {
        jayess_throw(jayess_type_error_value("Atomics requires a SharedArrayBuffer-backed typed array"));
        return NULL;
    }
    index = index_out != NULL ? *index_out : 0;
    if (index < 0 || index >= jayess_value_array_length(target)) {
        jayess_throw(jayess_type_error_value("Atomics index out of range"));
        return NULL;
    }
    if (kind_out != NULL) {
        *kind_out = kind;
    }
    return state;
}

static double jayess_atomics_apply(jayess_value *target, jayess_value *index_value, jayess_value *operand_value, const char *op, jayess_value *expected) {
    const char *kind = NULL;
    int index = (int)jayess_value_to_number(index_value);
    jayess_buffer_state *state = jayess_atomics_state(target, &index, &kind);
    jayess_array *bytes = jayess_std_bytes_slot(target);
    int size = jayess_std_typed_array_element_size(kind);
    int offset = index * size;
    double previous;
    double operand = operand_value != NULL ? jayess_value_to_number(operand_value) : 0.0;
    double expected_number = expected != NULL ? jayess_value_to_number(expected) : 0.0;
    if (state == NULL) {
        return 0.0;
    }
#ifdef _WIN32
    EnterCriticalSection(&state->lock);
#else
    pthread_mutex_lock(&state->lock);
#endif
    if (strcmp(kind, "Uint8Array") == 0) {
        previous = (double)jayess_std_bytes_read(bytes, offset);
    } else if (strcmp(kind, "Int8Array") == 0) {
        int value = jayess_std_bytes_read(bytes, offset);
        previous = (double)(value >= 128 ? value - 256 : value);
    } else if (strcmp(kind, "Uint16Array") == 0) {
        previous = (double)(jayess_std_bytes_read(bytes, offset) | (jayess_std_bytes_read(bytes, offset + 1) << 8));
    } else if (strcmp(kind, "Int16Array") == 0) {
        int value = jayess_std_bytes_read(bytes, offset) | (jayess_std_bytes_read(bytes, offset + 1) << 8);
        previous = (double)(value >= 32768 ? value - 65536 : value);
    } else if (strcmp(kind, "Uint32Array") == 0) {
        previous = (double)jayess_std_data_view_read_u32(bytes, offset, 1);
    } else {
        unsigned int raw = jayess_std_data_view_read_u32(bytes, offset, 1);
        previous = (double)(raw >= 2147483648U ? (long long)raw - 4294967296LL : (long long)raw);
    }
    if (op != NULL) {
        double next = previous;
        if (strcmp(op, "store") == 0 || strcmp(op, "exchange") == 0) {
            next = operand;
        } else if (strcmp(op, "add") == 0) {
            next = previous + operand;
        } else if (strcmp(op, "sub") == 0) {
            next = previous - operand;
        } else if (strcmp(op, "and") == 0) {
            next = (double)(((int64_t)previous) & ((int64_t)operand));
        } else if (strcmp(op, "or") == 0) {
            next = (double)(((int64_t)previous) | ((int64_t)operand));
        } else if (strcmp(op, "xor") == 0) {
            next = (double)(((int64_t)previous) ^ ((int64_t)operand));
        } else if (strcmp(op, "compareExchange") == 0) {
            next = previous == expected_number ? operand : previous;
        }
        if (strcmp(op, "compareExchange") != 0 || previous == expected_number) {
            if (strcmp(kind, "Uint8Array") == 0) {
                jayess_std_bytes_write(bytes, offset, (int)next & 255);
            } else if (strcmp(kind, "Int8Array") == 0) {
                jayess_std_bytes_write(bytes, offset, (int)next);
            } else if (strcmp(kind, "Uint16Array") == 0 || strcmp(kind, "Int16Array") == 0) {
                int value = (int)next;
                jayess_std_bytes_write(bytes, offset, value & 255);
                jayess_std_bytes_write(bytes, offset + 1, (value >> 8) & 255);
            } else if (strcmp(kind, "Uint32Array") == 0) {
                jayess_std_data_view_write_u32(bytes, offset, (unsigned int)next, 1);
            } else {
                jayess_std_data_view_write_u32(bytes, offset, (unsigned int)((int)next), 1);
            }
        }
    }
#ifdef _WIN32
    LeaveCriticalSection(&state->lock);
#else
    pthread_mutex_unlock(&state->lock);
#endif
    return previous;
}

double jayess_atomics_load(jayess_value *target, jayess_value *index) {
    return jayess_atomics_apply(target, index, NULL, NULL, NULL);
}

double jayess_atomics_store(jayess_value *target, jayess_value *index, jayess_value *value) {
    jayess_atomics_apply(target, index, value, "store", NULL);
    return jayess_value_to_number(value);
}

double jayess_atomics_add(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "add", NULL);
}

double jayess_atomics_sub(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "sub", NULL);
}

double jayess_atomics_and(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "and", NULL);
}

double jayess_atomics_or(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "or", NULL);
}

double jayess_atomics_xor(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "xor", NULL);
}

double jayess_atomics_exchange(jayess_value *target, jayess_value *index, jayess_value *value) {
    return jayess_atomics_apply(target, index, value, "exchange", NULL);
}

double jayess_atomics_compareExchange(jayess_value *target, jayess_value *index, jayess_value *expected, jayess_value *replacement) {
    return jayess_atomics_apply(target, index, replacement, "compareExchange", expected);
}
