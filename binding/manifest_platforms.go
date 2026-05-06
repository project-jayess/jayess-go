package binding

import "jayess-go/ast"

func readPlatforms(expression ast.Expression, diagnostics []Diagnostic) (map[string]PlatformOptions, []Diagnostic) {
	object, ok := expression.(*ast.ObjectLiteral)
	if !ok {
		return nil, append(diagnostics, Diagnostic{Field: "platforms", Message: "binding platforms must be an object literal"})
	}
	platforms := map[string]PlatformOptions{}
	for _, property := range object.Properties {
		name, ok := literalPropertyKey(property, "platforms")
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "platforms", Message: "binding platform names must be literal names"})
			continue
		}
		platformObject, ok := property.Value.(*ast.ObjectLiteral)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "platforms." + name, Message: "binding platform options must be an object literal"})
			continue
		}
		options, optionDiagnostics := readPlatformOptions(name, platformObject)
		diagnostics = append(diagnostics, optionDiagnostics...)
		platforms[name] = options
	}
	return platforms, diagnostics
}

func readPlatformOptions(name string, object *ast.ObjectLiteral) (PlatformOptions, []Diagnostic) {
	var options PlatformOptions
	var diagnostics []Diagnostic
	prefix := "platforms." + name + "."
	for _, property := range object.Properties {
		key, ok := literalPropertyKey(property, "platforms."+name)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "platforms." + name, Message: "binding platform option keys must be literal names"})
			continue
		}
		switch key {
		case "sources":
			options.Sources, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "includeDirs":
			options.IncludeDirs, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "libraryDirs":
			options.LibraryDirs, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "sharedLibraries":
			options.SharedLibraries, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "licenseFiles":
			options.LicenseFiles, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "runtimeAssets":
			options.RuntimeAssets, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "helperAssets":
			options.HelperAssets, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "cflags":
			options.CFlags, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "ldflags":
			options.LDFlags, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		}
	}
	return options, diagnostics
}
