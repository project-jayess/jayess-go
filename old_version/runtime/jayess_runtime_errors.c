#include "jayess_runtime_internal.h"

/* Split from jayess_runtime.c: error construction, exception state, and call-stack helpers. */

static JAYESS_THREAD_LOCAL jayess_this_frame *jayess_this_stack = NULL;
static JAYESS_THREAD_LOCAL jayess_call_frame *jayess_call_stack = NULL;
static JAYESS_THREAD_LOCAL jayess_value *jayess_current_exception = NULL;

void jayess_print_value_inline(jayess_value *value);

static jayess_value *jayess_std_error_new_text(const char *name_text, const char *message_text) {
    jayess_object *object = jayess_object_new();
    const char *safe_name = (name_text != NULL && name_text[0] != '\0') ? name_text : "Error";
    const char *safe_message = message_text != NULL ? message_text : "";
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(safe_name));
    jayess_object_set_value(object, "name", jayess_value_from_string(safe_name));
    jayess_object_set_value(object, "message", jayess_value_from_string(safe_message));
    return jayess_value_from_object(object);
}

jayess_value *jayess_type_error_value(const char *message) {
    return jayess_std_error_new_text("TypeError", message != NULL ? message : "");
}

jayess_value *jayess_std_error_new(jayess_value *name, jayess_value *message) {
    jayess_object *object = jayess_object_new();
    const char *name_text = "Error";
    char *message_text = NULL;
    if (name != NULL && name->kind == JAYESS_VALUE_STRING && name->as.string_value != NULL) {
        name_text = name->as.string_value;
    }
    if (message != NULL && message->kind != JAYESS_VALUE_UNDEFINED && message->kind != JAYESS_VALUE_NULL) {
        message_text = jayess_value_stringify(message);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string(name_text));
    jayess_object_set_value(object, "name", jayess_value_from_string(name_text));
    jayess_object_set_value(object, "message", jayess_value_from_string(message_text != NULL ? message_text : ""));
    if (message_text != NULL) {
        free(message_text);
    }
    return jayess_value_from_object(object);
}

jayess_value *jayess_std_aggregate_error_new(jayess_value *errors, jayess_value *message) {
    jayess_object *object = jayess_object_new();
    char *message_text = NULL;
    jayess_value *error_values = jayess_value_iterable_values(errors);
    if (message != NULL && message->kind != JAYESS_VALUE_UNDEFINED && message->kind != JAYESS_VALUE_NULL) {
        message_text = jayess_value_stringify(message);
    }
    if (object == NULL) {
        return jayess_value_from_object(NULL);
    }
    jayess_object_set_value(object, "__jayess_std_kind", jayess_value_from_string("AggregateError"));
    jayess_object_set_value(object, "name", jayess_value_from_string("AggregateError"));
    jayess_object_set_value(object, "message", jayess_value_from_string(message_text != NULL ? message_text : ""));
    jayess_object_set_value(object, "errors", error_values != NULL ? error_values : jayess_value_from_array(jayess_array_new()));
    if (message_text != NULL) {
        free(message_text);
    }
    return jayess_value_from_object(object);
}

jayess_value *jayess_error_value(const char *name, const char *message) {
    return jayess_std_error_new_text(name != NULL ? name : "Error", message != NULL ? message : "");
}

static char *jayess_capture_stack_trace_text(void) {
    jayess_call_frame *frame = jayess_call_stack;
    size_t total = 1;
    char *text;
    if (frame == NULL) {
        text = (char *)malloc(1);
        if (text != NULL) {
            text[0] = '\0';
        }
        return text;
    }
    while (frame != NULL) {
        const char *name = (frame->name != NULL && frame->name[0] != '\0') ? frame->name : "<anonymous>";
        total += strlen("  at ") + strlen(name) + 1;
        frame = frame->previous;
    }
    text = (char *)malloc(total);
    if (text == NULL) {
        return NULL;
    }
    text[0] = '\0';
    frame = jayess_call_stack;
    while (frame != NULL) {
        const char *name = (frame->name != NULL && frame->name[0] != '\0') ? frame->name : "<anonymous>";
        strcat(text, "  at ");
        strcat(text, name);
        strcat(text, "\n");
        frame = frame->previous;
    }
    return text;
}

static void jayess_attach_exception_stack(jayess_value *value) {
    char *stack_text;
    if (value == NULL) {
        return;
    }
    if (value->kind != JAYESS_VALUE_OBJECT || value->as.object_value == NULL) {
        return;
    }
    stack_text = jayess_capture_stack_trace_text();
    if (stack_text == NULL) {
        return;
    }
    jayess_object_set_value(value->as.object_value, "stack", jayess_value_from_string(stack_text));
    free(stack_text);
}

void jayess_throw(jayess_value *value) {
    jayess_current_exception = value != NULL ? value : jayess_value_undefined();
    jayess_attach_exception_stack(jayess_current_exception);
}

void jayess_throw_error(const char *message) {
    jayess_throw(jayess_error_value("Error", message));
}

void jayess_throw_type_error(const char *message) {
    jayess_throw(jayess_type_error_value(message));
}

void jayess_throw_named_error(const char *name, const char *message) {
    jayess_throw(jayess_error_value(name, message));
}

void jayess_throw_not_function(void) {
    jayess_throw(jayess_type_error_value("value is not a function"));
}

int jayess_has_exception(void) {
    return jayess_current_exception != NULL;
}

jayess_value *jayess_take_exception(void) {
    jayess_value *value = jayess_current_exception;
    jayess_current_exception = NULL;
    return value != NULL ? value : jayess_value_undefined();
}

void jayess_report_uncaught_exception(void) {
    jayess_value *stack;
    jayess_value *current;
    if (jayess_current_exception == NULL) {
        return;
    }
    current = jayess_current_exception;
    fputs("Uncaught exception: ", stderr);
    jayess_print_value_inline(current);
    fputc('\n', stderr);
    if (current->kind == JAYESS_VALUE_OBJECT && current->as.object_value != NULL) {
        stack = jayess_object_get(current->as.object_value, "stack");
        if (stack != NULL && stack->kind == JAYESS_VALUE_STRING && stack->as.string_value != NULL && stack->as.string_value[0] != '\0') {
            fputs(stack->as.string_value, stderr);
        }
    }
    jayess_current_exception = NULL;
    if (current != jayess_value_null() && current != jayess_value_undefined()) {
        jayess_value_free_unshared(current);
    }
}

void jayess_push_call_frame(const char *name) {
    jayess_call_frame *frame = (jayess_call_frame *)malloc(sizeof(jayess_call_frame));
    if (frame == NULL) {
        return;
    }
    frame->name = name;
    frame->previous = jayess_call_stack;
    jayess_call_stack = frame;
}

void jayess_pop_call_frame(void) {
    jayess_call_frame *current = jayess_call_stack;
    if (current == NULL) {
        return;
    }
    jayess_call_stack = current->previous;
    free(current);
}

void jayess_push_this(jayess_value *value) {
    jayess_this_frame *frame = (jayess_this_frame *)malloc(sizeof(jayess_this_frame));
    if (frame == NULL) {
        return;
    }
    frame->value = value != NULL ? value : jayess_value_undefined();
    frame->previous = jayess_this_stack;
    jayess_this_stack = frame;
}

void jayess_pop_this(void) {
    jayess_this_frame *current = jayess_this_stack;
    if (current == NULL) {
        return;
    }
    jayess_this_stack = current->previous;
    free(current);
}

jayess_value *jayess_current_this(void) {
    if (jayess_this_stack == NULL || jayess_this_stack->value == NULL) {
        return jayess_value_undefined();
    }
    return jayess_this_stack->value;
}

void jayess_runtime_error_state_shutdown(void) {
    while (jayess_this_stack != NULL) {
        jayess_this_frame *next = jayess_this_stack->previous;
        free(jayess_this_stack);
        jayess_this_stack = next;
    }
    while (jayess_call_stack != NULL) {
        jayess_call_frame *next = jayess_call_stack->previous;
        free(jayess_call_stack);
        jayess_call_stack = next;
    }
    if (jayess_current_exception != NULL &&
        jayess_current_exception != jayess_value_null() &&
        jayess_current_exception != jayess_value_undefined()) {
        jayess_value_free_unshared(jayess_current_exception);
    }
    jayess_current_exception = NULL;
}
