package diagnostics

import "strings"

type StackFrame struct {
	Function string
	Module   string
	Location SourceLocation
}

type StackTrace struct {
	Frames []StackFrame
}

func (trace StackTrace) Format() string {
	var out strings.Builder
	for index, frame := range trace.Frames {
		if index > 0 {
			out.WriteByte('\n')
		}
		out.WriteString("at ")
		if frame.Function == "" {
			out.WriteString("<anonymous>")
		} else {
			out.WriteString(frame.Function)
		}
		if frame.Module != "" {
			out.WriteString(" (")
			out.WriteString(frame.Module)
			out.WriteString(" ")
			out.WriteString(frame.Location.String())
			out.WriteString(")")
		} else {
			out.WriteString(" (")
			out.WriteString(frame.Location.String())
			out.WriteString(")")
		}
	}
	return out.String()
}

func (trace StackTrace) TopLocation() (SourceLocation, bool) {
	if len(trace.Frames) == 0 {
		return SourceLocation{}, false
	}
	return trace.Frames[0].Location, true
}
