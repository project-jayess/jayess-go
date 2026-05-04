package webparse

type IntegrationFeature string

const (
	FileSystemParsing      IntegrationFeature = "file-system-parsing"
	ModuleSystemParsing    IntegrationFeature = "module-system-parsing"
	UserProgramASTNodes    IntegrationFeature = "user-program-ast-nodes"
	CompilerDiagnostics    IntegrationFeature = "compiler-diagnostics"
	CompilerSpanAlignment  IntegrationFeature = "compiler-span-alignment"
	PredictableMemoryModel IntegrationFeature = "predictable-memory-model"
)

func IntegrationFeatures() []IntegrationFeature {
	return []IntegrationFeature{
		FileSystemParsing,
		ModuleSystemParsing,
		UserProgramASTNodes,
		CompilerDiagnostics,
		CompilerSpanAlignment,
		PredictableMemoryModel,
	}
}

func ParseHTMLFromFile(path string, readFile func(string) (string, error)) (Document, error) {
	source, err := readFile(path)
	if err != nil {
		return Document{}, err
	}
	return ParseHTMLFile(source, path), nil
}
