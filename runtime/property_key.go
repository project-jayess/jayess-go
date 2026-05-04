package runtime

import "strconv"

func PropertyKey(value Value) string {
	switch value.Kind() {
	case StringValue:
		return value.Text()
	case NumberValue:
		return strconv.FormatFloat(value.Number(), 'f', -1, 64)
	case BooleanValue:
		if value.Bool() {
			return "true"
		}
		return "false"
	case NullValue:
		return "null"
	case UndefinedValue:
		return "undefined"
	default:
		return string(value.Kind())
	}
}
