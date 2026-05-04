package binding

import "path/filepath"

type Module struct {
	Path     string
	Manifest Manifest
}

type CompileUnit struct {
	ModulePath  string
	Source      string
	IncludeDirs []string
	CFlags      []string
}

type BuildPlan struct {
	CompileUnits       []CompileUnit
	LibraryDirs        []string
	SharedLibraries    []string
	SharedLibraryFiles []string
	LicenseFiles       []string
	ExpectedSymbols    []ExpectedSymbol
	LDFlags            []string
	RuntimeHeaderDir   string
	Diagnostics        []Diagnostic
}

type ExpectedSymbol struct {
	ModulePath string
	ExportName string
	Symbol     string
}

func PlanBuild(modules []Module, platform string, runtimeHeaderDir string) BuildPlan {
	plan := BuildPlan{RuntimeHeaderDir: runtimeHeaderDir}
	seenSources := map[string]string{}
	seenLibraryDirs := map[string]struct{}{}
	seenSharedLibraries := map[string]struct{}{}
	seenLicenseFiles := map[string]struct{}{}
	seenLDFlags := map[string]struct{}{}
	for _, module := range modules {
		if err := ValidateBindingTarget(module.Path); err != nil {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Field: "module", Message: err.Error()})
			continue
		}
		for _, diagnostic := range ValidateManifest(module.Manifest) {
			plan.Diagnostics = append(plan.Diagnostics, diagnostic)
		}
		for _, expectation := range WrapperExpectations(module.Manifest) {
			plan.ExpectedSymbols = append(plan.ExpectedSymbols, ExpectedSymbol{
				ModulePath: module.Path,
				ExportName: expectation.ExportName,
				Symbol:     expectation.NativeSymbol,
			})
		}
		inputs := module.Manifest.BuildInputsFor(platform)
		includeDirs := includeRuntimeHeader(resolveBindingPaths(module.Path, inputs.IncludeDirs), runtimeHeaderDir)
		for _, source := range inputs.Sources {
			key := normalizeSourceKey(module.Path, source)
			if owner, exists := seenSources[key]; exists {
				plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
					Field:   "sources",
					Message: "duplicate native source " + source + " already compiled for " + owner,
				})
				continue
			}
			seenSources[key] = module.Path
			plan.CompileUnits = append(plan.CompileUnits, CompileUnit{
				ModulePath:  module.Path,
				Source:      source,
				IncludeDirs: append([]string{}, includeDirs...),
				CFlags:      append([]string{}, inputs.CFlags...),
			})
		}
		for _, rawDir := range inputs.LibraryDirs {
			dir := normalizeSourceKey(module.Path, rawDir)
			if _, exists := seenLibraryDirs[dir]; exists {
				continue
			}
			seenLibraryDirs[dir] = struct{}{}
			plan.LibraryDirs = append(plan.LibraryDirs, dir)
			appendUniqueLDFlag(&plan, seenLDFlags, "-L"+dir)
		}
		for _, library := range inputs.SharedLibraries {
			linkArg := sharedLibraryLinkArg(module.Path, library)
			if _, exists := seenSharedLibraries[linkArg]; exists {
				continue
			}
			seenSharedLibraries[linkArg] = struct{}{}
			plan.SharedLibraries = append(plan.SharedLibraries, library)
			if looksLikeLibraryPath(library) {
				plan.SharedLibraryFiles = append(plan.SharedLibraryFiles, linkArg)
			}
			appendUniqueLDFlag(&plan, seenLDFlags, linkArg)
		}
		for _, rawLicense := range inputs.LicenseFiles {
			license := normalizeSourceKey(module.Path, rawLicense)
			if _, exists := seenLicenseFiles[license]; exists {
				continue
			}
			seenLicenseFiles[license] = struct{}{}
			plan.LicenseFiles = append(plan.LicenseFiles, license)
		}
		for _, flag := range inputs.LDFlags {
			appendUniqueLDFlag(&plan, seenLDFlags, flag)
		}
	}
	return plan
}

func appendUniqueLDFlag(plan *BuildPlan, seen map[string]struct{}, flag string) {
	if _, exists := seen[flag]; exists {
		return
	}
	seen[flag] = struct{}{}
	plan.LDFlags = append(plan.LDFlags, flag)
}

func includeRuntimeHeader(includeDirs []string, runtimeHeaderDir string) []string {
	merged := append([]string{}, includeDirs...)
	if runtimeHeaderDir == "" {
		return merged
	}
	for _, dir := range merged {
		if dir == runtimeHeaderDir {
			return merged
		}
	}
	return append(merged, runtimeHeaderDir)
}

func resolveBindingPaths(modulePath string, paths []string) []string {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved = append(resolved, normalizeSourceKey(modulePath, path))
	}
	return resolved
}

func normalizeSourceKey(modulePath string, source string) string {
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}
	base := filepath.Dir(modulePath)
	return filepath.Clean(filepath.Join(base, filepath.FromSlash(source)))
}
