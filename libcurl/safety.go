package libcurl

type DiagnosticKind string

const (
	MissingHeaders DiagnosticKind = "missing-curl-headers"
	MissingLibrary DiagnosticKind = "missing-curl-library"
	TransferError  DiagnosticKind = "transfer-error"
)

func DiagnosticKinds() []DiagnosticKind {
	return []DiagnosticKind{
		MissingHeaders,
		MissingLibrary,
		TransferError,
	}
}
