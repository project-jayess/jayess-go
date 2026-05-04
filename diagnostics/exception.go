package diagnostics

type Exception struct {
	Name    string
	Message string
	Stack   StackTrace
}

func NewException(name string, message string, frames []StackFrame) Exception {
	return Exception{
		Name:    name,
		Message: message,
		Stack:   StackTrace{Frames: append([]StackFrame{}, frames...)},
	}
}

func (exception Exception) UncaughtDiagnostic() Diagnostic {
	location := SourceLocation{}
	if len(exception.Stack.Frames) > 0 {
		location = exception.Stack.Frames[0].Location
	}
	return Diagnostic{
		Code:     "JY-RUNTIME-uncaught-exception",
		Message:  "uncaught " + exception.Name + ": " + exception.Message,
		Severity: ErrorSeverity,
		Location: location,
	}
}
