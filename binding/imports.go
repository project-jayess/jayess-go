package binding

type ImportKind string

const (
	NamedImport      ImportKind = "named"
	DefaultImport    ImportKind = "default"
	NamespaceImport  ImportKind = "namespace"
	SideEffectImport ImportKind = "side-effect"
)

type ImportSpec struct {
	Source string
	Kind   ImportKind
	Names  []string
}

func ValidateImportSpec(spec ImportSpec, manifest Manifest) []Diagnostic {
	if err := ValidateBindingTarget(spec.Source); err != nil {
		return []Diagnostic{{Field: "import", Message: err.Error()}}
	}
	if spec.Kind != NamedImport {
		return []Diagnostic{{Field: "import", Message: "binding modules only support named imports"}}
	}
	if len(spec.Names) == 0 {
		return []Diagnostic{{Field: "import", Message: "binding import must request at least one exported name"}}
	}
	var diagnostics []Diagnostic
	for _, name := range spec.Names {
		if _, ok := manifest.ExportByName(name); !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "import." + name,
				Message: "binding export " + name + " was not declared",
			})
		}
	}
	return diagnostics
}
