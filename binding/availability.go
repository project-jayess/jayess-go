package binding

import (
	"os"
	"path/filepath"
)

type SymbolInventory map[string][]string

func ValidateBuildAvailability(plan BuildPlan, symbols SymbolInventory) []Diagnostic {
	var diagnostics []Diagnostic
	diagnostics = append(diagnostics, validateCompileUnitFiles(plan.CompileUnits)...)
	diagnostics = append(diagnostics, validateRuntimeHeader(plan.RuntimeHeaderDir)...)
	diagnostics = append(diagnostics, validateLibraryDirs(plan.LibraryDirs)...)
	diagnostics = append(diagnostics, validateSharedLibraryFiles(plan.SharedLibraryFiles)...)
	diagnostics = append(diagnostics, validateExpectedSymbols(plan.ExpectedSymbols, symbols)...)
	return diagnostics
}

func validateCompileUnitFiles(units []CompileUnit) []Diagnostic {
	var diagnostics []Diagnostic
	for _, unit := range units {
		path := normalizeSourceKey(unit.ModulePath, unit.Source)
		if !fileExists(path) {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "sources",
				Message: "missing native source " + path,
			})
		}
		for _, dir := range unit.IncludeDirs {
			if !dirExists(dir) {
				diagnostics = append(diagnostics, Diagnostic{
					Field:   "includeDirs",
					Message: "missing header directory " + dir,
				})
			}
		}
	}
	return diagnostics
}

func validateRuntimeHeader(runtimeHeaderDir string) []Diagnostic {
	if runtimeHeaderDir == "" {
		return nil
	}
	header := filepath.Join(runtimeHeaderDir, RuntimeHeader)
	if fileExists(header) {
		return nil
	}
	return []Diagnostic{{
		Field:   "runtimeHeaderDir",
		Message: "missing runtime header " + header,
	}}
}

func validateLibraryDirs(dirs []string) []Diagnostic {
	var diagnostics []Diagnostic
	for _, dir := range dirs {
		if !dirExists(dir) {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "libraryDirs",
				Message: "missing library directory " + dir,
			})
		}
	}
	return diagnostics
}

func validateSharedLibraryFiles(libraries []string) []Diagnostic {
	var diagnostics []Diagnostic
	for _, library := range libraries {
		if !fileExists(library) {
			diagnostics = append(diagnostics, Diagnostic{
				Field:   "sharedLibraries",
				Message: "missing shared library " + library,
			})
		}
	}
	return diagnostics
}

func validateExpectedSymbols(expected []ExpectedSymbol, inventories SymbolInventory) []Diagnostic {
	if len(inventories) == 0 {
		return nil
	}
	var diagnostics []Diagnostic
	for _, symbol := range expected {
		if hasSymbol(inventories[symbol.ModulePath], symbol.Symbol) {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Field:   "exports." + symbol.ExportName,
			Message: "missing native symbol " + symbol.Symbol + " for " + symbol.ExportName,
		})
	}
	return diagnostics
}

func hasSymbol(symbols []string, want string) bool {
	for _, symbol := range symbols {
		if symbol == want {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
