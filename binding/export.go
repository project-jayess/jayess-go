package binding

type ExportKind string

const (
	FunctionExport ExportKind = "function"
	ValueExport    ExportKind = "value"
)

type Export struct {
	Name   string
	Symbol string
	Kind   ExportKind
}

func (kind ExportKind) Valid() bool {
	return kind == FunctionExport || kind == ValueExport
}

func (manifest Manifest) ExportByName(name string) (Export, bool) {
	for _, export := range manifest.Exports {
		if export.Name == name {
			return export, true
		}
	}
	return Export{}, false
}
