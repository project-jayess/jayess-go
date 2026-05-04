package raylib

type DiagnosticKind string

const (
	MissingHeaders DiagnosticKind = "missing-raylib-headers"
	MissingSource  DiagnosticKind = "missing-raylib-source"
	MissingLibrary DiagnosticKind = "missing-raylib-library"
	BuildFailure   DiagnosticKind = "raylib-build-failure"
	RuntimeError   DiagnosticKind = "raylib-runtime-error"
)

func DiagnosticKinds() []DiagnosticKind {
	return []DiagnosticKind{
		MissingHeaders,
		MissingSource,
		MissingLibrary,
		BuildFailure,
		RuntimeError,
	}
}
