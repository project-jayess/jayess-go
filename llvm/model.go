package llvm

type APIKind string

const (
	ContextAPI APIKind = "context"
	ModuleAPI  APIKind = "module"
	BuilderAPI APIKind = "builder"
	TypeAPI    APIKind = "type"
	ValueAPI   APIKind = "value"
	TargetAPI  APIKind = "target"
	ObjectAPI  APIKind = "object"
	LinkerAPI  APIKind = "linker"
)

type BackendKind string

const (
	LLVMCBackend BackendKind = "llvm-c-api"
	LLDBackend   BackendKind = "lld-cpp-shim"
)

type PackageModel struct {
	Import   string
	APIs     []APIKind
	Backends []BackendKind
}

type Diagnostic struct {
	Field   string
	Message string
}

func DefaultPackage() PackageModel {
	return PackageModel{
		Import: "llvm",
		APIs: []APIKind{
			ContextAPI,
			ModuleAPI,
			BuilderAPI,
			TypeAPI,
			ValueAPI,
			TargetAPI,
			ObjectAPI,
			LinkerAPI,
		},
		Backends: []BackendKind{LLVMCBackend, LLDBackend},
	}
}

func SupportsAPI(model PackageModel, api APIKind) bool {
	for _, available := range model.APIs {
		if available == api {
			return true
		}
	}
	return false
}

func SupportsBackend(model PackageModel, backend BackendKind) bool {
	for _, available := range model.Backends {
		if available == backend {
			return true
		}
	}
	return false
}

func ValidatePackage(model PackageModel) []Diagnostic {
	var diagnostics []Diagnostic
	if model.Import == "" {
		diagnostics = append(diagnostics, Diagnostic{Field: "llvm.import", Message: "LLVM package import must not be empty"})
	}
	if len(model.APIs) == 0 {
		diagnostics = append(diagnostics, Diagnostic{Field: "llvm.apis", Message: "LLVM package must expose at least one API group"})
	}
	if !SupportsAPI(model, ModuleAPI) {
		diagnostics = append(diagnostics, Diagnostic{Field: "llvm.apis", Message: "LLVM package must expose module construction"})
	}
	if !SupportsAPI(model, ObjectAPI) {
		diagnostics = append(diagnostics, Diagnostic{Field: "llvm.apis", Message: "LLVM package must expose object emission"})
	}
	if len(model.Backends) == 0 {
		diagnostics = append(diagnostics, Diagnostic{Field: "llvm.backends", Message: "LLVM package must declare at least one backend"})
	}
	return diagnostics
}
