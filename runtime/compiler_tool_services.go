package runtime

type CompilerToolService string

const (
	SourceService       CompilerToolService = "source"
	PathService         CompilerToolService = "path"
	DiagnosticService   CompilerToolService = "diagnostic"
	DataService         CompilerToolService = "data"
	LLVMService         CompilerToolService = "llvm"
	LinkerService       CompilerToolService = "linker"
	DistributionService CompilerToolService = "distribution"
)

type CompilerToolRuntime struct {
	Services []CompilerToolService
}

func DefaultCompilerToolRuntime() CompilerToolRuntime {
	return CompilerToolRuntime{Services: []CompilerToolService{
		SourceService,
		PathService,
		DiagnosticService,
		DataService,
		LLVMService,
		LinkerService,
		DistributionService,
	}}
}

func (runtime CompilerToolRuntime) Has(service CompilerToolService) bool {
	for _, available := range runtime.Services {
		if available == service {
			return true
		}
	}
	return false
}

func ValidateCompilerToolRuntime(runtime CompilerToolRuntime) []string {
	required := DefaultCompilerToolRuntime().Services
	var diagnostics []string
	for _, service := range required {
		if !runtime.Has(service) {
			diagnostics = append(diagnostics, "missing compiler tool runtime service: "+string(service))
		}
	}
	return diagnostics
}
