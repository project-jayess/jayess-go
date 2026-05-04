package binding

type PlaceholderKind string

const (
	DirectPlaceholder PlaceholderKind = "direct"
	SharedPlaceholder PlaceholderKind = "shared"
)

type PlaceholderExport struct {
	Name string
	Stub string
	Kind PlaceholderKind
}

func ValidatePlaceholderExports(manifest Manifest, placeholders []PlaceholderExport) []Diagnostic {
	declared := map[string]PlaceholderExport{}
	for _, placeholder := range placeholders {
		declared[placeholder.Name] = placeholder
	}
	var diagnostics []Diagnostic
	for _, export := range manifest.Exports {
		placeholder, ok := declared[export.Name]
		if !ok {
			continue
		}
		if placeholder.Kind == SharedPlaceholder && placeholder.Stub == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "placeholder." + export.Name,
				Message: "shared placeholder export must name its stub binding",
			})
		}
	}
	return diagnostics
}
