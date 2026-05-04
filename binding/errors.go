package binding

type BoundaryErrorKind string

const (
	InvalidImportError      BoundaryErrorKind = "invalid-import"
	SymbolResolutionError   BoundaryErrorKind = "symbol-resolution"
	TypeMismatchError       BoundaryErrorKind = "type-mismatch"
	BindingThrownError      BoundaryErrorKind = "binding-thrown"
	NativeBuildFailureError BoundaryErrorKind = "native-build-failure"
)

type BoundaryError struct {
	Kind    BoundaryErrorKind
	Message string
}

func BoundaryErrorKinds() []BoundaryErrorKind {
	return []BoundaryErrorKind{
		InvalidImportError,
		SymbolResolutionError,
		TypeMismatchError,
		BindingThrownError,
		NativeBuildFailureError,
	}
}
