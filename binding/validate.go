package binding

import "strings"

func ValidateManifest(manifest Manifest) []Diagnostic {
	var diagnostics []Diagnostic
	diagnostics = append(diagnostics, validatePathList("sources", manifest.Sources)...)
	diagnostics = append(diagnostics, validatePathList("includeDirs", manifest.IncludeDirs)...)
	diagnostics = append(diagnostics, validatePathList("libraryDirs", manifest.LibraryDirs)...)
	diagnostics = append(diagnostics, validatePathList("licenseFiles", manifest.LicenseFiles)...)
	diagnostics = append(diagnostics, validatePathList("runtimeAssets", manifest.RuntimeAssets)...)
	diagnostics = append(diagnostics, validatePathList("helperAssets", manifest.HelperAssets)...)
	diagnostics = append(diagnostics, validateSharedLibraries("sharedLibraries", manifest.SharedLibraries)...)
	diagnostics = append(diagnostics, validateFlagList("cflags", manifest.CFlags)...)
	diagnostics = append(diagnostics, validateFlagList("ldflags", manifest.LDFlags)...)
	diagnostics = append(diagnostics, validatePlatforms(manifest.Platforms)...)
	diagnostics = append(diagnostics, validateExports(manifest.Exports)...)
	diagnostics = append(diagnostics, ValidateExportedSymbols(manifest)...)
	return diagnostics
}

func validatePlatforms(platforms map[string]PlatformOptions) []Diagnostic {
	var diagnostics []Diagnostic
	for platform, options := range platforms {
		if strings.TrimSpace(platform) != platform || platform == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "platforms",
				Message: "platform name must not be empty or padded",
			})
		}
		prefix := "platforms." + platform + "."
		diagnostics = append(diagnostics, validatePathList(prefix+"sources", options.Sources)...)
		diagnostics = append(diagnostics, validatePathList(prefix+"includeDirs", options.IncludeDirs)...)
		diagnostics = append(diagnostics, validatePathList(prefix+"libraryDirs", options.LibraryDirs)...)
		diagnostics = append(diagnostics, validatePathList(prefix+"licenseFiles", options.LicenseFiles)...)
		diagnostics = append(diagnostics, validatePathList(prefix+"runtimeAssets", options.RuntimeAssets)...)
		diagnostics = append(diagnostics, validatePathList(prefix+"helperAssets", options.HelperAssets)...)
		diagnostics = append(diagnostics, validateSharedLibraries(prefix+"sharedLibraries", options.SharedLibraries)...)
		diagnostics = append(diagnostics, validateFlagList(prefix+"cflags", options.CFlags)...)
		diagnostics = append(diagnostics, validateFlagList(prefix+"ldflags", options.LDFlags)...)
	}
	return diagnostics
}

func validateSharedLibraries(field string, values []string) []Diagnostic {
	var diagnostics []Diagnostic
	for _, value := range values {
		if strings.TrimSpace(value) != value || value == "" {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "shared library entries must not be empty or padded"})
		}
		if strings.ContainsAny(value, "?#") {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "shared library entries must not include query strings or fragments"})
		}
		if strings.Contains(value, "\\") {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "shared library entries must use / as the separator"})
		}
	}
	return diagnostics
}

func validateExports(exports []Export) []Diagnostic {
	if len(exports) == 0 {
		return []Diagnostic{{Field: "exports", Message: "binding manifest must export at least one symbol"}}
	}
	var diagnostics []Diagnostic
	names := map[string]struct{}{}
	for _, export := range exports {
		if strings.TrimSpace(export.Name) != export.Name || export.Name == "" {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports", Message: "export name must not be empty or padded"})
		}
		if strings.TrimSpace(export.Symbol) != export.Symbol || export.Symbol == "" {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + export.Name, Message: "export symbol must not be empty or padded"})
		}
		if !export.Kind.Valid() {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + export.Name, Message: "export type must be function or value"})
		}
		if _, exists := names[export.Name]; exists {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + export.Name, Message: "duplicate export name"})
		}
		names[export.Name] = struct{}{}
	}
	return diagnostics
}

func validatePathList(field string, values []string) []Diagnostic {
	var diagnostics []Diagnostic
	for _, value := range values {
		if strings.TrimSpace(value) != value || value == "" {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "path entries must not be empty or padded"})
		}
		if strings.ContainsAny(value, "?#") {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "path entries must not include query strings or fragments"})
		}
		if strings.Contains(value, "\\") {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "path entries must use / as the separator"})
		}
	}
	return diagnostics
}

func validateFlagList(field string, values []string) []Diagnostic {
	var diagnostics []Diagnostic
	for _, value := range values {
		if strings.TrimSpace(value) != value || value == "" {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "flag entries must not be empty or padded"})
		}
	}
	return diagnostics
}
