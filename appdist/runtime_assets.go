package appdist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jayess-go/binding"
)

func ResolveRuntimeAssets(plan binding.BuildPlan, targetName string) ([]RuntimeAsset, []string) {
	var assets []RuntimeAsset
	var diagnostics []string
	seen := map[string]struct{}{}
	for _, library := range plan.SharedLibraryFiles {
		if !isSharedLibraryFile(library, targetName) {
			continue
		}
		if _, err := os.Stat(library); err != nil {
			diagnostics = append(diagnostics, "missing runtime shared library: "+library)
			continue
		}
		assets = appendRuntimeAsset(assets, seen, library)
	}
	for _, library := range plan.SharedLibraries {
		if isLibraryPath(library) && len(plan.SharedLibraryFiles) != 0 {
			continue
		}
		resolved, libraryDiagnostics := resolveRuntimeLibrary(library, plan.LibraryDirs, targetName)
		diagnostics = append(diagnostics, libraryDiagnostics...)
		for _, path := range resolved {
			assets = appendRuntimeAsset(assets, seen, path)
		}
	}
	for _, license := range plan.LicenseFiles {
		if _, err := os.Stat(license); err != nil {
			diagnostics = append(diagnostics, "missing binding license file: "+license)
			continue
		}
		assets = appendLicenseAsset(assets, seen, license)
	}
	return assets, diagnostics
}

func appendRuntimeAsset(assets []RuntimeAsset, seen map[string]struct{}, path string) []RuntimeAsset {
	key := filepath.Clean(path)
	if _, ok := seen[key]; ok {
		return assets
	}
	seen[key] = struct{}{}
	return append(assets, RuntimeAsset{SourcePath: key, OutputName: filepath.Base(key)})
}

func appendLicenseAsset(assets []RuntimeAsset, seen map[string]struct{}, path string) []RuntimeAsset {
	key := filepath.Clean(path)
	if _, ok := seen[key]; ok {
		return assets
	}
	seen[key] = struct{}{}
	return append(assets, RuntimeAsset{SourcePath: key, OutputName: filepath.Join("licenses", filepath.Base(key))})
}

func resolveRuntimeLibrary(library string, libraryDirs []string, targetName string) ([]string, []string) {
	if isLibraryPath(library) {
		if !isSharedLibraryFile(library, targetName) {
			return nil, nil
		}
		if _, err := os.Stat(library); err != nil {
			return nil, []string{"missing runtime shared library: " + library}
		}
		return []string{library}, nil
	}
	names := sharedLibraryNames(library, targetName)
	for _, dir := range libraryDirs {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return []string{candidate}, nil
			}
		}
	}
	return nil, []string{fmt.Sprintf("runtime shared library %q was not found in libraryDirs", library)}
}

func isLibraryPath(library string) bool {
	return strings.Contains(library, "/") || strings.Contains(library, string(filepath.Separator)) || filepath.Ext(library) != ""
}

func isSharedLibraryFile(path string, targetName string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch targetName {
	case "windows-x64":
		return ext == ".dll"
	case "macos-x64", "macos-arm64":
		return ext == ".dylib"
	default:
		return ext == ".so"
	}
}

func sharedLibraryNames(library string, targetName string) []string {
	name := strings.TrimPrefix(library, "-l")
	switch targetName {
	case "windows-x64":
		return []string{name + ".dll", "lib" + name + ".dll"}
	case "macos-x64", "macos-arm64":
		return []string{"lib" + name + ".dylib", name + ".dylib"}
	default:
		return []string{"lib" + name + ".so", name + ".so"}
	}
}
