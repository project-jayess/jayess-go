package tooling

type DiagnosticFormat struct {
	Name        string
	ShowsFile   bool
	ShowsLine   bool
	ShowsColumn bool
	ShowsDetail bool
}

func DefaultDiagnosticFormat() DiagnosticFormat {
	return DiagnosticFormat{
		Name:        "default",
		ShowsFile:   true,
		ShowsLine:   true,
		ShowsColumn: true,
		ShowsDetail: true,
	}
}
