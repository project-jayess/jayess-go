package binding

import (
	"sort"

	"jayess-go/ast"
)

func ManifestFromProgram(program *ast.Program) (Manifest, []Diagnostic) {
	call := bindingExportCall(program)
	if call == nil {
		return Manifest{}, []Diagnostic{{Field: "default", Message: "binding module must export default bind(...) from ffi"}}
	}
	if len(call.Arguments) != 1 {
		return Manifest{}, []Diagnostic{{Field: "default", Message: "bind(...) expects exactly one manifest object"}}
	}
	object, ok := call.Arguments[0].(*ast.ObjectLiteral)
	if !ok {
		return Manifest{}, []Diagnostic{{Field: "default", Message: "bind(...) manifest must be an object literal"}}
	}
	return manifestFromObject(object)
}

func manifestFromObject(object *ast.ObjectLiteral) (Manifest, []Diagnostic) {
	var manifest Manifest
	var diagnostics []Diagnostic
	for _, property := range object.Properties {
		key, ok := literalPropertyKey(property, "manifest")
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "manifest", Message: "binding manifest keys must be literal names"})
			continue
		}
		switch key {
		case "sources":
			manifest.Sources, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "includeDirs":
			manifest.IncludeDirs, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "libraryDirs":
			manifest.LibraryDirs, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "sharedLibraries":
			manifest.SharedLibraries, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "licenseFiles":
			manifest.LicenseFiles, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "cflags":
			manifest.CFlags, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "ldflags":
			manifest.LDFlags, diagnostics = readStringArrayField(key, property.Value, diagnostics)
		case "exports":
			manifest.Exports, diagnostics = readExports(property.Value, diagnostics)
		case "platforms":
			manifest.Platforms, diagnostics = readPlatforms(property.Value, diagnostics)
		}
	}
	return manifest, diagnostics
}

func readStringArrayField(field string, expression ast.Expression, diagnostics []Diagnostic) ([]string, []Diagnostic) {
	array, ok := expression.(*ast.ArrayLiteral)
	if !ok {
		return nil, append(diagnostics, Diagnostic{Field: field, Message: "binding manifest field must be a string array literal"})
	}
	values := make([]string, 0, len(array.Elements))
	for _, element := range array.Elements {
		literal, ok := element.(*ast.StringLiteral)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: field, Message: "binding manifest array entries must be string literals"})
			continue
		}
		values = append(values, literal.Value)
	}
	return values, diagnostics
}

func readExports(expression ast.Expression, diagnostics []Diagnostic) ([]Export, []Diagnostic) {
	object, ok := expression.(*ast.ObjectLiteral)
	if !ok {
		return nil, append(diagnostics, Diagnostic{Field: "exports", Message: "binding exports must be an object literal"})
	}
	exports := make([]Export, 0, len(object.Properties))
	for _, property := range object.Properties {
		name, ok := literalPropertyKey(property, "exports")
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports", Message: "binding export names must be literal names"})
			continue
		}
		exportObject, ok := property.Value.(*ast.ObjectLiteral)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + name, Message: "binding export must be an object literal"})
			continue
		}
		export, exportDiagnostics := readExport(name, exportObject)
		diagnostics = append(diagnostics, exportDiagnostics...)
		exports = append(exports, export)
	}
	sort.SliceStable(exports, func(i, j int) bool {
		return exports[i].Name < exports[j].Name
	})
	return exports, diagnostics
}

func readExport(name string, object *ast.ObjectLiteral) (Export, []Diagnostic) {
	export := Export{Name: name}
	var diagnostics []Diagnostic
	for _, property := range object.Properties {
		key, ok := literalPropertyKey(property, "exports."+name)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + name, Message: "binding export keys must be literal names"})
			continue
		}
		value, ok := property.Value.(*ast.StringLiteral)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{Field: "exports." + name + "." + key, Message: "binding export fields must be string literals"})
			continue
		}
		switch key {
		case "symbol":
			export.Symbol = value.Value
		case "type":
			export.Kind = ExportKind(value.Value)
		}
	}
	return export, diagnostics
}

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
		case "cflags":
			options.CFlags, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		case "ldflags":
			options.LDFlags, diagnostics = readStringArrayField(prefix+key, property.Value, diagnostics)
		}
	}
	return options, diagnostics
}

func literalPropertyKey(property ast.ObjectProperty, field string) (string, bool) {
	if property.Computed || property.Spread || property.Method || property.Getter || property.Setter {
		return field, false
	}
	if property.Key == "" {
		return field, false
	}
	return property.Key, true
}
