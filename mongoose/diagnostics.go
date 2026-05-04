package mongoose

type DiagnosticKind string

const (
	MissingHeaders DiagnosticKind = "missing-mongoose-headers"
	MissingSource  DiagnosticKind = "missing-mongoose-source"
	BuildFailure   DiagnosticKind = "mongoose-build-failure"
	RuntimeError   DiagnosticKind = "mongoose-runtime-error"
)

func DiagnosticKinds() []DiagnosticKind {
	return []DiagnosticKind{
		MissingHeaders,
		MissingSource,
		BuildFailure,
		RuntimeError,
	}
}
