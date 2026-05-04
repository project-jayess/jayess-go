package picohttpparser

type DiagnosticKind string

const (
	MissingHeaders DiagnosticKind = "missing-picohttpparser-headers"
	MissingSource  DiagnosticKind = "missing-picohttpparser-source"
	BuildFailure   DiagnosticKind = "picohttpparser-build-failure"
	MalformedInput DiagnosticKind = "malformed-http-input"
	IncompleteData DiagnosticKind = "incomplete-http-data"
)

func DiagnosticKinds() []DiagnosticKind {
	return []DiagnosticKind{
		MissingHeaders,
		MissingSource,
		BuildFailure,
		MalformedInput,
		IncompleteData,
	}
}
