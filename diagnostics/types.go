package diagnostics

type TypeErrorKind string

const (
	TypeMismatch        TypeErrorKind = "type-mismatch"
	MissingProperty     TypeErrorKind = "missing-property"
	InvalidCallTarget   TypeErrorKind = "invalid-call-target"
	InvalidAssignment   TypeErrorKind = "invalid-assignment"
	UnsupportedTypeForm TypeErrorKind = "unsupported-type-form"
)

func TypeError(location SourceLocation, kind TypeErrorKind, message string) Diagnostic {
	return Diagnostic{
		Code:     "JY-TYPE-" + string(kind),
		Message:  message,
		Severity: ErrorSeverity,
		Location: location,
	}
}

func TypeErrorKinds() []TypeErrorKind {
	return []TypeErrorKind{
		TypeMismatch,
		MissingProperty,
		InvalidCallTarget,
		InvalidAssignment,
		UnsupportedTypeForm,
	}
}
