package runtime

type PackageRole string

const (
	CoreRuntimeRole PackageRole = "core-runtime"
	StdlibRole      PackageRole = "stdlib"
	SystemRole      PackageRole = "system"
)

type PackageModel struct {
	Name     string
	Import   string
	Role     PackageRole
	Language ImplementationLanguage
}

type PackageDiagnostic struct {
	Package string
	Message string
}

func GoRuntimePackages() []PackageModel {
	return []PackageModel{
		{Name: "mvp-globals", Import: "jayess-go/runtime", Role: CoreRuntimeRole, Language: GoRuntime},
		{Name: "filesystem", Import: "jayess-go/runtime", Role: StdlibRole, Language: GoRuntime},
		{Name: "process", Import: "jayess-go/runtime", Role: SystemRole, Language: GoRuntime},
		{Name: "network", Import: "jayess-go/runtime", Role: StdlibRole, Language: GoRuntime},
		{Name: "worker", Import: "jayess-go/runtime", Role: SystemRole, Language: GoRuntime},
	}
}

func CoreRuntimePackagesAreGo(packages []PackageModel) bool {
	for _, pkg := range packages {
		if pkg.Role == CoreRuntimeRole && pkg.Language != GoRuntime {
			return false
		}
	}
	return true
}

func HasGoRuntimePackage(name string) bool {
	for _, pkg := range GoRuntimePackages() {
		if pkg.Name == name {
			return pkg.Language == GoRuntime && pkg.Import != ""
		}
	}
	return false
}

func ValidateGoRuntimePackages(packages []PackageModel) []PackageDiagnostic {
	var diagnostics []PackageDiagnostic
	seen := map[string]struct{}{}
	hasCore := false
	for _, pkg := range packages {
		if pkg.Name == "" {
			diagnostics = append(diagnostics, PackageDiagnostic{Message: "runtime package name must not be empty"})
			continue
		}
		if _, exists := seen[pkg.Name]; exists {
			diagnostics = append(diagnostics, PackageDiagnostic{Package: pkg.Name, Message: "duplicate runtime package"})
		}
		seen[pkg.Name] = struct{}{}
		if pkg.Import == "" {
			diagnostics = append(diagnostics, PackageDiagnostic{Package: pkg.Name, Message: "runtime package import path must not be empty"})
		}
		if pkg.Role == "" {
			diagnostics = append(diagnostics, PackageDiagnostic{Package: pkg.Name, Message: "runtime package role must not be empty"})
		} else if !isPackageRole(pkg.Role) {
			diagnostics = append(diagnostics, PackageDiagnostic{Package: pkg.Name, Message: "unknown runtime package role"})
		}
		if pkg.Role == CoreRuntimeRole {
			hasCore = true
		}
		if pkg.Language != GoRuntime {
			diagnostics = append(diagnostics, PackageDiagnostic{Package: pkg.Name, Message: "runtime package must be implemented in Go"})
		}
	}
	if !hasCore {
		diagnostics = append(diagnostics, PackageDiagnostic{Message: "runtime package registry must include a core runtime package"})
	}
	return diagnostics
}

func isPackageRole(role PackageRole) bool {
	switch role {
	case CoreRuntimeRole, StdlibRole, SystemRole:
		return true
	default:
		return false
	}
}
