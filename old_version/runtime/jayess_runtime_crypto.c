#include <ctype.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <windows.h>
#include <wincrypt.h>
#include <bcrypt.h>
#else
#include <openssl/evp.h>
#endif

#include "jayess_runtime_internal.h"

struct jayess_crypto_key_state {
#ifdef _WIN32
    BCRYPT_KEY_HANDLE handle;
#else
    EVP_PKEY *pkey;
#endif
    int is_private;
    char *type;
};

void jayess_std_crypto_normalize_name(char *text) {
    size_t i;
    if (text == NULL) {
        return;
    }
    for (i = 0; text[i] != '\0'; i++) {
        text[i] = (char)tolower((unsigned char)text[i]);
    }
}

int jayess_std_crypto_equal_name(const char *left, const char *right) {
    if (left == NULL || right == NULL) {
        return 0;
    }
    while (*left != '\0' && *right != '\0') {
        if (tolower((unsigned char)*left) != tolower((unsigned char)*right)) {
            return 0;
        }
        left++;
        right++;
    }
    return *left == '\0' && *right == '\0';
}

char *jayess_std_crypto_hex_encode(const unsigned char *bytes, size_t length) {
    static const char *hex = "0123456789abcdef";
    char *out;
    size_t i;
    out = (char *)malloc((length * 2) + 1);
    if (out == NULL) {
        return NULL;
    }
    for (i = 0; i < length; i++) {
        out[i * 2] = hex[(bytes[i] >> 4) & 15];
        out[(i * 2) + 1] = hex[bytes[i] & 15];
    }
    out[length * 2] = '\0';
    return out;
}

int jayess_std_crypto_copy_bytes(jayess_value *value, unsigned char **out_bytes, size_t *out_length) {
    jayess_array *bytes = jayess_std_bytes_slot(value);
    unsigned char *buffer = NULL;
    size_t length = 0;
    size_t i;
    if (out_bytes == NULL || out_length == NULL) {
        return 0;
    }
    *out_bytes = NULL;
    *out_length = 0;
    if (bytes != NULL) {
        length = (size_t)bytes->count;
        buffer = (unsigned char *)malloc(length > 0 ? length : 1);
        if (buffer == NULL) {
            return 0;
        }
        for (i = 0; i < length; i++) {
            buffer[i] = (unsigned char)((int)jayess_value_to_number(jayess_array_get(bytes, (int)i)) & 255);
        }
        *out_bytes = buffer;
        *out_length = length;
        return 1;
    }
    {
        char *text = jayess_value_stringify(value);
        if (text == NULL) {
            buffer = (unsigned char *)malloc(1);
            if (buffer == NULL) {
                return 0;
            }
            *out_bytes = buffer;
            *out_length = 0;
            return 1;
        }
        length = strlen(text);
        buffer = (unsigned char *)malloc(length > 0 ? length : 1);
        if (buffer == NULL) {
            free(text);
            return 0;
        }
        if (length > 0) {
            memcpy(buffer, text, length);
        }
        free(text);
        *out_bytes = buffer;
        *out_length = length;
        return 1;
    }
}

int jayess_std_crypto_cipher_key_length(const char *algorithm) {
    if (jayess_std_crypto_equal_name(algorithm, "aes-128-gcm")) {
        return 16;
    }
    if (jayess_std_crypto_equal_name(algorithm, "aes-192-gcm")) {
        return 24;
    }
    if (jayess_std_crypto_equal_name(algorithm, "aes-256-gcm")) {
        return 32;
    }
    return 0;
}

int jayess_std_crypto_option_bytes(jayess_value *options, const char *key, unsigned char **out_bytes, size_t *out_length, int required) {
    jayess_object *object = jayess_value_as_object(options);
    jayess_value *value;
    if (out_bytes == NULL || out_length == NULL) {
        return 0;
    }
    *out_bytes = NULL;
    *out_length = 0;
    if (object == NULL) {
        return required ? 0 : 1;
    }
    value = jayess_object_get(object, key);
    if (value == NULL || jayess_value_is_nullish(value)) {
        return required ? 0 : 1;
    }
    return jayess_std_crypto_copy_bytes(value, out_bytes, out_length);
}

jayess_value *jayess_std_crypto_key_value(const char *type, int is_private) {
    jayess_object *object = jayess_object_new();
    jayess_crypto_key_state *state;
    if (object == NULL) {
        return jayess_value_undefined();
    }
    state = (jayess_crypto_key_state *)calloc(1, sizeof(jayess_crypto_key_state));
    if (state == NULL) {
        return jayess_value_undefined();
    }
    state->is_private = is_private ? 1 : 0;
    state->type = jayess_strdup(type != NULL ? type : "");
    object->native_handle = state;
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("CryptoKey"));
    jayess_object_set_value(object, "type", jayess_value_from_string(type != NULL ? type : ""));
    jayess_object_set_value(object, "private", jayess_value_from_bool(is_private));
    return jayess_value_from_object(object);
}

jayess_crypto_key_state *jayess_std_crypto_key_state_from_value(jayess_value *value) {
    if (value == NULL || value->kind != JAYESS_VALUE_OBJECT || !jayess_std_kind_is(value, "CryptoKey") || value->as.object_value == NULL) {
        return NULL;
    }
    return (jayess_crypto_key_state *)value->as.object_value->native_handle;
}

int jayess_std_bytes_encoding_is_hex(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 0;
    }
    text = jayess_value_stringify(encoding);
    ok = text != NULL && strcmp(text, "hex") == 0;
    free(text);
    return ok;
}

int jayess_std_bytes_encoding_is_base64(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 0;
    }
    text = jayess_value_stringify(encoding);
    ok = text != NULL && (strcmp(text, "base64") == 0 || strcmp(text, "base-64") == 0);
    free(text);
    return ok;
}

int jayess_std_bytes_encoding_is_text(jayess_value *encoding) {
    char *text;
    int ok;
    if (encoding == NULL || jayess_value_is_nullish(encoding)) {
        return 1;
    }
    text = jayess_value_stringify(encoding);
    ok = text == NULL || strcmp(text, "utf8") == 0 || strcmp(text, "utf-8") == 0 || strcmp(text, "text") == 0;
    free(text);
    return ok;
}

#ifdef _WIN32
LPCWSTR jayess_std_crypto_algorithm_id(const char *name) {
    if (jayess_std_crypto_equal_name(name, "sha1") || jayess_std_crypto_equal_name(name, "sha-1")) {
        return BCRYPT_SHA1_ALGORITHM;
    }
    if (jayess_std_crypto_equal_name(name, "sha256") || jayess_std_crypto_equal_name(name, "sha-256")) {
        return BCRYPT_SHA256_ALGORITHM;
    }
    if (jayess_std_crypto_equal_name(name, "sha384") || jayess_std_crypto_equal_name(name, "sha-384")) {
        return BCRYPT_SHA384_ALGORITHM;
    }
    if (jayess_std_crypto_equal_name(name, "sha512") || jayess_std_crypto_equal_name(name, "sha-512")) {
        return BCRYPT_SHA512_ALGORITHM;
    }
    if (jayess_std_crypto_equal_name(name, "md5")) {
        return BCRYPT_MD5_ALGORITHM;
    }
    return NULL;
}

int jayess_std_crypto_sha256_bytes(const unsigned char *input, size_t input_length, unsigned char *output, DWORD *output_length) {
    BCRYPT_ALG_HANDLE provider = NULL;
    BCRYPT_HASH_HANDLE hash = NULL;
    DWORD object_length = 0;
    DWORD digest_length = 0;
    DWORD bytes_written = 0;
    PUCHAR object_buffer = NULL;
    int ok = 0;
    if (output == NULL || output_length == NULL) {
        return 0;
    }
    if (BCryptOpenAlgorithmProvider(&provider, BCRYPT_SHA256_ALGORITHM, NULL, 0) < 0 ||
        BCryptGetProperty(provider, BCRYPT_OBJECT_LENGTH, (PUCHAR)&object_length, sizeof(object_length), &bytes_written, 0) < 0 ||
        BCryptGetProperty(provider, BCRYPT_HASH_LENGTH, (PUCHAR)&digest_length, sizeof(digest_length), &bytes_written, 0) < 0 ||
        digest_length > *output_length) {
        if (provider != NULL) {
            BCryptCloseAlgorithmProvider(provider, 0);
        }
        return 0;
    }
    object_buffer = (PUCHAR)malloc(object_length > 0 ? object_length : 1);
    if (object_buffer == NULL ||
        BCryptCreateHash(provider, &hash, object_buffer, object_length, NULL, 0, 0) < 0 ||
        (input_length > 0 && BCryptHashData(hash, (PUCHAR)input, (ULONG)input_length, 0) < 0) ||
        BCryptFinishHash(hash, output, digest_length, 0) < 0) {
        if (hash != NULL) {
            BCryptDestroyHash(hash);
        }
        BCryptCloseAlgorithmProvider(provider, 0);
        free(object_buffer);
        return 0;
    }
    *output_length = digest_length;
    ok = 1;
    BCryptDestroyHash(hash);
    BCryptCloseAlgorithmProvider(provider, 0);
    free(object_buffer);
    return ok;
}
#endif
