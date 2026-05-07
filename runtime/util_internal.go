package runtime

import (
	"fmt"
	"sort"
	"strings"
)

func UtilFormat(format string, args ...any) string {
	if strings.Contains(format, "%") {
		return fmt.Sprintf(format, args...)
	}
	parts := []string{format}
	for _, arg := range args {
		parts = append(parts, UtilInspect(arg))
	}
	return strings.Join(parts, " ")
}

func UtilInspect(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return fmt.Sprintf("%q", typed)
	case []byte:
		return fmt.Sprintf("<Buffer %x>", typed)
	case *Buffer:
		if typed == nil {
			return "<Buffer nil>"
		}
		return fmt.Sprintf("<Buffer %x>", typed.Data)
	case map[string]any:
		return inspectStringAnyMap(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func inspectStringAnyMap(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+UtilInspect(values[key]))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
