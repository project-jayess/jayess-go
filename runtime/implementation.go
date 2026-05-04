package runtime

type ImplementationLanguage string

const (
	GoRuntime ImplementationLanguage = "go"
)

type BoundaryLanguage string

const (
	CBoundary   BoundaryLanguage = "c"
	CPPBoundary BoundaryLanguage = "c++"
)

type ImplementationModel struct {
	RuntimeLanguage       ImplementationLanguage
	CompilerLanguage      ImplementationLanguage
	ExternalBoundaries    []BoundaryLanguage
	NativeBindingsAreCore bool
}

func DefaultImplementationModel() ImplementationModel {
	return ImplementationModel{
		RuntimeLanguage:       GoRuntime,
		CompilerLanguage:      GoRuntime,
		ExternalBoundaries:    []BoundaryLanguage{CBoundary, CPPBoundary},
		NativeBindingsAreCore: false,
	}
}

func RuntimeIsGoFirst(model ImplementationModel) bool {
	return model.RuntimeLanguage == GoRuntime && !model.NativeBindingsAreCore
}

func SupportsExternalBoundary(model ImplementationModel, language BoundaryLanguage) bool {
	for _, boundary := range model.ExternalBoundaries {
		if boundary == language {
			return true
		}
	}
	return false
}
