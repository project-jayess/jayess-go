package binding

type WrapperExpectation struct {
	ExportName    string
	NativeSymbol  string
	WrapperSymbol string
	Kind          ExportKind
}

func WrapperExpectations(manifest Manifest) []WrapperExpectation {
	expectations := make([]WrapperExpectation, 0, len(manifest.Exports))
	for _, export := range manifest.Exports {
		expectations = append(expectations, WrapperExpectation{
			ExportName:    export.Name,
			NativeSymbol:  export.Symbol,
			WrapperSymbol: WrapperSymbolForExport(export.Name),
			Kind:          export.Kind,
		})
	}
	return expectations
}

func WrapperSymbolForExport(name string) string {
	suffix := cSymbolSuffix(name)
	if suffix == "" {
		suffix = "anonymous"
	}
	return "jayess_binding_export_" + suffix
}

func ValidateExportedSymbols(manifest Manifest) []Diagnostic {
	var diagnostics []Diagnostic
	seenNativeSymbols := map[string]string{}
	for _, expectation := range WrapperExpectations(manifest) {
		field := "exports." + expectation.ExportName
		if !isCSymbol(expectation.NativeSymbol) {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   field,
				Message: "export symbol must be a valid C symbol for generated wrappers",
			})
		}
		if owner, exists := seenNativeSymbols[expectation.NativeSymbol]; exists && expectation.NativeSymbol != "" {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   field,
				Message: "export symbol duplicates native symbol used by " + owner,
			})
		}
		seenNativeSymbols[expectation.NativeSymbol] = expectation.ExportName
		if expectation.NativeSymbol == expectation.WrapperSymbol {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   field,
				Message: "export symbol must not collide with generated wrapper symbol",
			})
		}
		if RuntimeHeaderHasFunction(expectation.NativeSymbol) {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   field,
				Message: "export symbol must not collide with Jayess runtime header functions",
			})
		}
	}
	return diagnostics
}

func cSymbolSuffix(value string) string {
	out := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if isCSymbolPart(character) {
			out = append(out, character)
			continue
		}
		out = append(out, '_')
	}
	return string(out)
}

func isCSymbol(value string) bool {
	if value == "" || !isCSymbolStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isCSymbolPart(value[index]) {
			return false
		}
	}
	return true
}

func isCSymbolStart(character byte) bool {
	return character == '_' ||
		character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z'
}

func isCSymbolPart(character byte) bool {
	return isCSymbolStart(character) ||
		character >= '0' && character <= '9'
}
