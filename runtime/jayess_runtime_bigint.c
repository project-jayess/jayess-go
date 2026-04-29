#include <math.h>

#include "jayess_runtime_internal.h"

typedef struct jayess_bigint_words {
    size_t length;
    uint32_t *words;
} jayess_bigint_words;

static jayess_bigint_words jayess_bigint_words_new(size_t length) {
    jayess_bigint_words value;
    value.length = length;
    value.words = length > 0 ? (uint32_t *)calloc(length, sizeof(uint32_t)) : NULL;
    return value;
}

static void jayess_bigint_words_free(jayess_bigint_words value) {
    free(value.words);
}

static size_t jayess_bigint_words_trimmed_length(jayess_bigint_words value) {
    size_t length = value.length;
    while (length > 0 && value.words[length-1] == 0) {
        length--;
    }
    return length;
}

static int jayess_bigint_words_is_zero(jayess_bigint_words value) {
    return jayess_bigint_words_trimmed_length(value) == 0;
}

static jayess_bigint_words jayess_bigint_words_clone(jayess_bigint_words value) {
    jayess_bigint_words copy = jayess_bigint_words_new(value.length);
    if (copy.words != NULL && value.words != NULL && value.length > 0) {
        memcpy(copy.words, value.words, value.length * sizeof(uint32_t));
    }
    return copy;
}

static jayess_bigint_words jayess_bigint_words_resized(jayess_bigint_words value, size_t length) {
    jayess_bigint_words resized = jayess_bigint_words_new(length);
    size_t copy_count = value.length < length ? value.length : length;
    if (resized.words != NULL && value.words != NULL && copy_count > 0) {
        memcpy(resized.words, value.words, copy_count * sizeof(uint32_t));
    }
    return resized;
}

static jayess_bigint_words jayess_bigint_words_from_u32(uint32_t value) {
    jayess_bigint_words out;
    if (value == 0) {
        out.length = 0;
        out.words = NULL;
        return out;
    }
    out = jayess_bigint_words_new(1);
    if (out.words != NULL) {
        out.words[0] = value;
    }
    return out;
}

static void jayess_bigint_words_mul_small(jayess_bigint_words *value, uint32_t factor) {
    size_t i;
    uint64_t carry = 0;
    if (factor == 0 || value == NULL || value->length == 0 || value->words == NULL) {
        if (value != NULL && factor == 0) {
            free(value->words);
            value->words = NULL;
            value->length = 0;
        }
        return;
    }
    for (i = 0; i < value->length; i++) {
        uint64_t product = (uint64_t)value->words[i] * factor + carry;
        value->words[i] = (uint32_t)product;
        carry = product >> 32;
    }
    if (carry != 0) {
        uint32_t *grown = (uint32_t *)realloc(value->words, (value->length + 1) * sizeof(uint32_t));
        if (grown == NULL) {
            return;
        }
        value->words = grown;
        value->words[value->length] = (uint32_t)carry;
        value->length++;
    }
}

static void jayess_bigint_words_add_small(jayess_bigint_words *value, uint32_t addend) {
    size_t i = 0;
    uint64_t carry = addend;
    if (value == NULL || carry == 0) {
        return;
    }
    if (value->length == 0 || value->words == NULL) {
        *value = jayess_bigint_words_from_u32(addend);
        return;
    }
    while (carry != 0 && i < value->length) {
        uint64_t sum = (uint64_t)value->words[i] + carry;
        value->words[i] = (uint32_t)sum;
        carry = sum >> 32;
        i++;
    }
    if (carry != 0) {
        uint32_t *grown = (uint32_t *)realloc(value->words, (value->length + 1) * sizeof(uint32_t));
        if (grown == NULL) {
            return;
        }
        value->words = grown;
        value->words[value->length] = (uint32_t)carry;
        value->length++;
    }
}

static int jayess_bigint_words_sub_small(jayess_bigint_words *value, uint32_t subtrahend) {
    size_t i = 0;
    uint64_t borrow = subtrahend;
    if (value == NULL || value->length == 0 || value->words == NULL) {
        return 0;
    }
    while (borrow != 0 && i < value->length) {
        uint64_t current = value->words[i];
        if (current >= borrow) {
            value->words[i] = (uint32_t)(current - borrow);
            borrow = 0;
        } else {
            value->words[i] = (uint32_t)((UINT64_C(1) << 32) + current - borrow);
            borrow = 1;
        }
        i++;
    }
    return borrow == 0;
}

static int jayess_bigint_words_cmp(jayess_bigint_words left, jayess_bigint_words right) {
    size_t left_len = jayess_bigint_words_trimmed_length(left);
    size_t right_len = jayess_bigint_words_trimmed_length(right);
    size_t i;
    if (left_len != right_len) {
        return left_len > right_len ? 1 : -1;
    }
    for (i = left_len; i > 0; i--) {
        uint32_t lhs = left.words[i-1];
        uint32_t rhs = right.words[i-1];
        if (lhs != rhs) {
            return lhs > rhs ? 1 : -1;
        }
    }
    return 0;
}

static void jayess_bigint_words_add_words(jayess_bigint_words *left, jayess_bigint_words right) {
    size_t max_len;
    size_t i;
    uint64_t carry = 0;
    if (right.length == 0 || right.words == NULL || left == NULL) {
        return;
    }
    if (left->length < right.length) {
        uint32_t *grown = (uint32_t *)realloc(left->words, right.length * sizeof(uint32_t));
        if (grown == NULL) {
            return;
        }
        memset(grown + left->length, 0, (right.length - left->length) * sizeof(uint32_t));
        left->words = grown;
        left->length = right.length;
    }
    max_len = left->length > right.length ? left->length : right.length;
    for (i = 0; i < max_len; i++) {
        uint64_t lhs = i < left->length ? left->words[i] : 0;
        uint64_t rhs = i < right.length ? right.words[i] : 0;
        uint64_t sum = lhs + rhs + carry;
        if (i < left->length) {
            left->words[i] = (uint32_t)sum;
        }
        carry = sum >> 32;
    }
    if (carry != 0) {
        uint32_t *grown = (uint32_t *)realloc(left->words, (left->length + 1) * sizeof(uint32_t));
        if (grown == NULL) {
            return;
        }
        left->words = grown;
        left->words[left->length] = (uint32_t)carry;
        left->length++;
    }
}

static int jayess_bigint_words_divmod_small(jayess_bigint_words *value, uint32_t divisor, uint32_t *remainder) {
    size_t i;
    uint64_t rem = 0;
    if (value == NULL || divisor == 0 || value->length == 0 || value->words == NULL) {
        if (remainder != NULL) {
            *remainder = 0;
        }
        return 0;
    }
    for (i = value->length; i > 0; i--) {
        uint64_t current = (rem << 32) | value->words[i-1];
        value->words[i-1] = (uint32_t)(current / divisor);
        rem = current % divisor;
    }
    if (remainder != NULL) {
        *remainder = (uint32_t)rem;
    }
    return 1;
}

static size_t jayess_bigint_words_bit_length(jayess_bigint_words value) {
    size_t length = jayess_bigint_words_trimmed_length(value);
    uint32_t top;
    size_t bits = 0;
    if (length == 0) {
        return 0;
    }
    top = value.words[length-1];
    while (top != 0) {
        bits++;
        top >>= 1;
    }
    return (length - 1) * 32 + bits;
}

static jayess_bigint_words jayess_bigint_parse_magnitude(const char *text) {
    const char *cursor = text;
    jayess_bigint_words value = {0, NULL};
    if (cursor == NULL) {
        return value;
    }
    if (*cursor == '+' || *cursor == '-') {
        cursor++;
    }
    while (*cursor == '0') {
        cursor++;
    }
    for (; *cursor != '\0'; cursor++) {
        if (*cursor < '0' || *cursor > '9') {
            jayess_bigint_words_free(value);
            value.length = 0;
            value.words = NULL;
            return value;
        }
        jayess_bigint_words_mul_small(&value, 10);
        jayess_bigint_words_add_small(&value, (uint32_t)(*cursor - '0'));
    }
    return value;
}

static int jayess_bigint_is_negative_text(const char *text) {
    return text != NULL && text[0] == '-';
}

static size_t jayess_bigint_signed_bit_length(const char *text) {
    jayess_bigint_words magnitude = jayess_bigint_parse_magnitude(text);
    size_t bits = 1;
    if (!jayess_bigint_words_is_zero(magnitude)) {
        if (jayess_bigint_is_negative_text(text)) {
            jayess_bigint_words adjusted = jayess_bigint_words_clone(magnitude);
            jayess_bigint_words_sub_small(&adjusted, 1);
            bits = jayess_bigint_words_bit_length(adjusted) + 1;
            jayess_bigint_words_free(adjusted);
        } else {
            bits = jayess_bigint_words_bit_length(magnitude) + 1;
        }
    }
    jayess_bigint_words_free(magnitude);
    return bits;
}

static jayess_bigint_words jayess_bigint_to_twos_complement(const char *text, size_t width_bits) {
    size_t word_count = (width_bits + 31) / 32;
    size_t i;
    jayess_bigint_words magnitude = jayess_bigint_parse_magnitude(text);
    jayess_bigint_words words = jayess_bigint_words_resized(magnitude, word_count);
    jayess_bigint_words_free(magnitude);
    if (jayess_bigint_is_negative_text(text) && word_count > 0) {
        uint64_t carry = 1;
        for (i = 0; i < word_count; i++) {
            uint64_t value = (uint64_t)(~words.words[i]) + carry;
            words.words[i] = (uint32_t)value;
            carry = value >> 32;
        }
    }
    if (word_count > 0 && (width_bits % 32) != 0) {
        uint32_t mask = (uint32_t)((UINT64_C(1) << (width_bits % 32)) - 1);
        words.words[word_count-1] &= mask;
    }
    return words;
}

static char *jayess_bigint_from_twos_complement(jayess_bigint_words words, size_t width_bits) {
    size_t word_count = (width_bits + 31) / 32;
    int negative;
    size_t bit_index;
    if (word_count == 0 || jayess_bigint_words_is_zero(words)) {
        return jayess_strdup("0");
    }
    bit_index = width_bits - 1;
    negative = ((words.words[bit_index / 32] >> (bit_index % 32)) & 1U) != 0;
    if (!negative) {
        jayess_bigint_words magnitude = jayess_bigint_words_resized(words, word_count);
        if ((width_bits % 32) != 0) {
            uint32_t mask = (uint32_t)((UINT64_C(1) << (width_bits % 32)) - 1);
            magnitude.words[word_count-1] &= mask;
        }
        if (jayess_bigint_words_is_zero(magnitude)) {
            jayess_bigint_words_free(magnitude);
            return jayess_strdup("0");
        }
        {
            static const uint32_t base = 1000000000U;
            uint32_t parts[256];
            size_t part_count = 0;
            char *out;
            size_t capacity;
            jayess_bigint_words work = jayess_bigint_words_clone(magnitude);
            jayess_bigint_words_free(magnitude);
            while (!jayess_bigint_words_is_zero(work) && part_count < (sizeof(parts) / sizeof(parts[0]))) {
                uint32_t remainder = 0;
                jayess_bigint_words_divmod_small(&work, base, &remainder);
                part_count++;
                parts[part_count-1] = remainder;
            }
            capacity = part_count * 10 + 2;
            out = (char *)malloc(capacity);
            if (out == NULL) {
                jayess_bigint_words_free(work);
                return jayess_strdup("0");
            }
            snprintf(out, capacity, "%u", parts[part_count-1]);
            while (part_count > 1) {
                char chunk[16];
                part_count--;
                snprintf(chunk, sizeof(chunk), "%09u", parts[part_count-1]);
                strcat(out, chunk);
            }
            jayess_bigint_words_free(work);
            return out;
        }
    }
    {
        size_t i;
        jayess_bigint_words magnitude = jayess_bigint_words_resized(words, word_count);
        uint64_t carry = 1;
        char *positive;
        char *out;
        if ((width_bits % 32) != 0) {
            uint32_t mask = (uint32_t)((UINT64_C(1) << (width_bits % 32)) - 1);
            magnitude.words[word_count-1] &= mask;
        }
        for (i = 0; i < word_count; i++) {
            uint64_t value = (uint64_t)(~magnitude.words[i]) + carry;
            magnitude.words[i] = (uint32_t)value;
            carry = value >> 32;
        }
        positive = jayess_bigint_from_twos_complement(magnitude, width_bits);
        jayess_bigint_words_free(magnitude);
        if (positive == NULL || strcmp(positive, "0") == 0) {
            free(positive);
            return jayess_strdup("0");
        }
        out = (char *)malloc(strlen(positive) + 2);
        if (out == NULL) {
            free(positive);
            return jayess_strdup("0");
        }
        out[0] = '-';
        strcpy(out + 1, positive);
        free(positive);
        return out;
    }
}

static jayess_bigint_words jayess_bigint_shift_left_words(jayess_bigint_words value, size_t shift) {
    size_t word_shift = shift / 32;
    size_t bit_shift = shift % 32;
    size_t i;
    jayess_bigint_words out;
    if (jayess_bigint_words_is_zero(value)) {
        out.length = 0;
        out.words = NULL;
        return out;
    }
    out = jayess_bigint_words_new(value.length + word_shift + 1);
    for (i = 0; i < value.length; i++) {
        uint64_t current = (uint64_t)value.words[i] << bit_shift;
        out.words[i + word_shift] |= (uint32_t)current;
        out.words[i + word_shift + 1] |= (uint32_t)(current >> 32);
    }
    return out;
}

static jayess_bigint_words jayess_bigint_shift_right_words(jayess_bigint_words value, size_t shift) {
    size_t word_shift = shift / 32;
    size_t bit_shift = shift % 32;
    size_t i;
    jayess_bigint_words out;
    if (word_shift >= value.length || jayess_bigint_words_is_zero(value)) {
        out.length = 0;
        out.words = NULL;
        return out;
    }
    out = jayess_bigint_words_new(value.length - word_shift);
    for (i = value.length; i > word_shift; i--) {
        uint32_t current = value.words[i-1];
        size_t target = i - 1 - word_shift;
        out.words[target] |= bit_shift == 0 ? current : (current >> bit_shift);
        if (bit_shift != 0 && i - 1 > word_shift) {
            out.words[target] |= value.words[i-2] << (32 - bit_shift);
        }
    }
    return out;
}

static int jayess_bigint_parse_shift_count(const char *text, int *negative, size_t *count) {
    const char *cursor = text;
    size_t value = 0;
    if (negative != NULL) {
        *negative = 0;
    }
    if (count != NULL) {
        *count = 0;
    }
    if (cursor == NULL) {
        return 0;
    }
    if (*cursor == '+' || *cursor == '-') {
        if (*cursor == '-' && negative != NULL) {
            *negative = 1;
        }
        cursor++;
    }
    while (*cursor == '0') {
        cursor++;
    }
    for (; *cursor != '\0'; cursor++) {
        if (*cursor < '0' || *cursor > '9') {
            return 0;
        }
        if (value > (SIZE_MAX - (size_t)(*cursor - '0')) / 10) {
            return 0;
        }
        value = value * 10 + (size_t)(*cursor - '0');
    }
    if (count != NULL) {
        *count = value;
    }
    return 1;
}

static char *jayess_bigint_bitwise_unary_not(const char *text) {
    jayess_bigint_words magnitude = jayess_bigint_parse_magnitude(text);
    char *out;
    if (jayess_bigint_is_negative_text(text)) {
        jayess_bigint_words_sub_small(&magnitude, 1);
        out = jayess_bigint_from_twos_complement(magnitude, jayess_bigint_words_bit_length(magnitude) + 1);
        jayess_bigint_words_free(magnitude);
        return out;
    }
    jayess_bigint_words_add_small(&magnitude, 1);
    out = jayess_bigint_from_twos_complement(magnitude, jayess_bigint_words_bit_length(magnitude) + 1);
    jayess_bigint_words_free(magnitude);
    if (strcmp(out, "0") == 0) {
        free(out);
        return jayess_strdup("-1");
    }
    {
        char *negative = (char *)malloc(strlen(out) + 2);
        if (negative == NULL) {
            free(out);
            return jayess_strdup("-1");
        }
        negative[0] = '-';
        strcpy(negative + 1, out);
        free(out);
        return negative;
    }
}

static char *jayess_bigint_bitwise_binary(const char *left_text, const char *right_text, char op) {
    size_t width = jayess_bigint_signed_bit_length(left_text);
    size_t right_width = jayess_bigint_signed_bit_length(right_text);
    size_t word_count;
    size_t i;
    jayess_bigint_words left_words;
    jayess_bigint_words right_words;
    if (right_width > width) {
        width = right_width;
    }
    word_count = (width + 31) / 32;
    left_words = jayess_bigint_to_twos_complement(left_text, width);
    right_words = jayess_bigint_to_twos_complement(right_text, width);
    for (i = 0; i < word_count; i++) {
        switch (op) {
        case '&':
            left_words.words[i] &= right_words.words[i];
            break;
        case '|':
            left_words.words[i] |= right_words.words[i];
            break;
        default:
            left_words.words[i] ^= right_words.words[i];
            break;
        }
    }
    jayess_bigint_words_free(right_words);
    {
        char *out = jayess_bigint_from_twos_complement(left_words, width);
        jayess_bigint_words_free(left_words);
        return out;
    }
}

static char *jayess_bigint_shift_left_text(const char *value_text, const char *shift_text) {
    jayess_bigint_words magnitude = jayess_bigint_parse_magnitude(value_text);
    int negative_shift = 0;
    size_t shift = 0;
    char *out;
    if (!jayess_bigint_parse_shift_count(shift_text, &negative_shift, &shift)) {
        jayess_bigint_words_free(magnitude);
        return NULL;
    }
    if (negative_shift) {
        jayess_bigint_words_free(magnitude);
        return NULL;
    }
    magnitude = jayess_bigint_shift_left_words(magnitude, shift);
    out = jayess_bigint_from_twos_complement(magnitude, jayess_bigint_words_bit_length(magnitude) + 1);
    jayess_bigint_words_free(magnitude);
    if (jayess_bigint_is_negative_text(value_text) && strcmp(out, "0") != 0) {
        char *negative = (char *)malloc(strlen(out) + 2);
        if (negative == NULL) {
            free(out);
            return jayess_strdup("0");
        }
        negative[0] = '-';
        strcpy(negative + 1, out);
        free(out);
        return negative;
    }
    return out;
}

static char *jayess_bigint_shift_right_text(const char *value_text, const char *shift_text) {
    jayess_bigint_words magnitude = jayess_bigint_parse_magnitude(value_text);
    int negative_shift = 0;
    size_t shift = 0;
    char *out;
    if (!jayess_bigint_parse_shift_count(shift_text, &negative_shift, &shift)) {
        jayess_bigint_words_free(magnitude);
        return NULL;
    }
    if (negative_shift) {
        jayess_bigint_words_free(magnitude);
        return NULL;
    }
    if (!jayess_bigint_is_negative_text(value_text)) {
        magnitude = jayess_bigint_shift_right_words(magnitude, shift);
        out = jayess_bigint_from_twos_complement(magnitude, jayess_bigint_words_bit_length(magnitude) + 1);
        jayess_bigint_words_free(magnitude);
        return out;
    }
    if (jayess_bigint_words_is_zero(magnitude)) {
        jayess_bigint_words_free(magnitude);
        return jayess_strdup("0");
    }
    jayess_bigint_words_sub_small(&magnitude, 1);
    magnitude = jayess_bigint_shift_right_words(magnitude, shift);
    jayess_bigint_words_add_small(&magnitude, 1);
    out = jayess_bigint_from_twos_complement(magnitude, jayess_bigint_words_bit_length(magnitude) + 1);
    jayess_bigint_words_free(magnitude);
    if (strcmp(out, "0") == 0) {
        free(out);
        return jayess_strdup("-1");
    }
    {
        char *negative = (char *)malloc(strlen(out) + 2);
        if (negative == NULL) {
            free(out);
            return jayess_strdup("-1");
        }
        negative[0] = '-';
        strcpy(negative + 1, out);
        free(out);
        return negative;
    }
}

static uint32_t jayess_number_to_uint32(double value) {
    if (!isfinite(value) || value == 0.0) {
        return 0;
    }
    {
        double integer = trunc(value);
        double modulo = fmod(integer, 4294967296.0);
        if (modulo < 0.0) {
            modulo += 4294967296.0;
        }
        return (uint32_t)modulo;
    }
}

static int32_t jayess_number_to_int32(double value) {
    uint32_t uint_value = jayess_number_to_uint32(value);
    if (uint_value >= 2147483648U) {
        return (int32_t)(uint_value - 4294967296U);
    }
    return (int32_t)uint_value;
}

static jayess_value *jayess_bitwise_type_error(const char *message) {
    jayess_throw(jayess_type_error_value(message));
    return jayess_value_undefined();
}

static jayess_value *jayess_value_bitwise_number_binary(jayess_value *left, jayess_value *right, char op) {
    int32_t lhs = jayess_number_to_int32(jayess_value_to_number(left));
    uint32_t rhs_u32 = jayess_number_to_uint32(jayess_value_to_number(right));
    int32_t rhs = (int32_t)rhs_u32;
    int32_t result = 0;
    switch (op) {
    case '&':
        result = lhs & rhs;
        break;
    case '|':
        result = lhs | rhs;
        break;
    case '^':
        result = lhs ^ rhs;
        break;
    case '<':
        result = lhs << (rhs_u32 & 31U);
        break;
    case '>':
        result = lhs >> (rhs_u32 & 31U);
        break;
    default:
        return jayess_value_from_number((double)(jayess_number_to_uint32(jayess_value_to_number(left)) >> (rhs_u32 & 31U)));
    }
    return jayess_value_from_number((double)result);
}

jayess_value *jayess_value_bitwise_not(jayess_value *value) {
    if (value != NULL && value->kind == JAYESS_VALUE_BIGINT) {
        char *result_text = jayess_bigint_bitwise_unary_not(value->as.bigint_value != NULL ? value->as.bigint_value : "0");
        jayess_value *result = jayess_value_from_bigint(result_text != NULL ? result_text : "0");
        free(result_text);
        return result;
    }
    return jayess_value_from_number((double)(~jayess_number_to_int32(jayess_value_to_number(value))));
}

static jayess_value *jayess_value_bitwise_binary(jayess_value *left, jayess_value *right, char op) {
    int left_is_bigint = left != NULL && left->kind == JAYESS_VALUE_BIGINT;
    int right_is_bigint = right != NULL && right->kind == JAYESS_VALUE_BIGINT;
    if (left_is_bigint || right_is_bigint) {
        char *result_text;
        jayess_value *result;
        if (!left_is_bigint || !right_is_bigint) {
            return jayess_bitwise_type_error("cannot mix number and bigint in bitwise expressions");
        }
        result_text = jayess_bigint_bitwise_binary(left->as.bigint_value != NULL ? left->as.bigint_value : "0",
                                                   right->as.bigint_value != NULL ? right->as.bigint_value : "0",
                                                   op);
        result = jayess_value_from_bigint(result_text != NULL ? result_text : "0");
        free(result_text);
        return result;
    }
    return jayess_value_bitwise_number_binary(left, right, op);
}

jayess_value *jayess_value_bitwise_and(jayess_value *left, jayess_value *right) {
    return jayess_value_bitwise_binary(left, right, '&');
}

jayess_value *jayess_value_bitwise_or(jayess_value *left, jayess_value *right) {
    return jayess_value_bitwise_binary(left, right, '|');
}

jayess_value *jayess_value_bitwise_xor(jayess_value *left, jayess_value *right) {
    return jayess_value_bitwise_binary(left, right, '^');
}

jayess_value *jayess_value_bitwise_shl(jayess_value *left, jayess_value *right) {
    int left_is_bigint = left != NULL && left->kind == JAYESS_VALUE_BIGINT;
    int right_is_bigint = right != NULL && right->kind == JAYESS_VALUE_BIGINT;
    if (left_is_bigint || right_is_bigint) {
        char *result_text;
        jayess_value *result;
        if (!left_is_bigint || !right_is_bigint) {
            return jayess_bitwise_type_error("cannot mix number and bigint in bitwise expressions");
        }
        result_text = jayess_bigint_shift_left_text(left->as.bigint_value != NULL ? left->as.bigint_value : "0",
                                                    right->as.bigint_value != NULL ? right->as.bigint_value : "0");
        if (result_text == NULL) {
            return jayess_bitwise_type_error("bigint shift count must be a non-negative bigint within runtime limits");
        }
        result = jayess_value_from_bigint(result_text);
        free(result_text);
        return result;
    }
    return jayess_value_bitwise_number_binary(left, right, '<');
}

jayess_value *jayess_value_bitwise_shr(jayess_value *left, jayess_value *right) {
    int left_is_bigint = left != NULL && left->kind == JAYESS_VALUE_BIGINT;
    int right_is_bigint = right != NULL && right->kind == JAYESS_VALUE_BIGINT;
    if (left_is_bigint || right_is_bigint) {
        char *result_text;
        jayess_value *result;
        if (!left_is_bigint || !right_is_bigint) {
            return jayess_bitwise_type_error("cannot mix number and bigint in bitwise expressions");
        }
        result_text = jayess_bigint_shift_right_text(left->as.bigint_value != NULL ? left->as.bigint_value : "0",
                                                     right->as.bigint_value != NULL ? right->as.bigint_value : "0");
        if (result_text == NULL) {
            return jayess_bitwise_type_error("bigint shift count must be a non-negative bigint within runtime limits");
        }
        result = jayess_value_from_bigint(result_text);
        free(result_text);
        return result;
    }
    return jayess_value_bitwise_number_binary(left, right, '>');
}

jayess_value *jayess_value_bitwise_ushr(jayess_value *left, jayess_value *right) {
    if ((left != NULL && left->kind == JAYESS_VALUE_BIGINT) || (right != NULL && right->kind == JAYESS_VALUE_BIGINT)) {
        return jayess_bitwise_type_error("operator >>> does not support bigint operands");
    }
    return jayess_value_bitwise_number_binary(left, right, 'u');
}
