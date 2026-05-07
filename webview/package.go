package webview

type PublicAPIKind string

const (
	AppLifecycleAPI  PublicAPIKind = "app-lifecycle"
	WindowSurfaceAPI PublicAPIKind = "window-surface"
	MountSurfaceAPI  PublicAPIKind = "mount-surface"
	EventSurfaceAPI  PublicAPIKind = "event-surface"
	DialogSurfaceAPI PublicAPIKind = "dialog-surface"
	DropSurfaceAPI   PublicAPIKind = "drop-surface"
	RawHostAPI       PublicAPIKind = "raw-host-escape-hatch"
)

type PackageModel struct {
	Import              string
	RuntimeImport       string
	UsesInternalRuntime bool
	APIs                []PublicAPIKind
}

type PackageDiagnostic struct {
	Field   string
	Message string
}

func DefaultPackage() PackageModel {
	return PackageModel{
		Import:              "@jayess/webview",
		RuntimeImport:       "jayess-go/runtime/webview",
		UsesInternalRuntime: true,
		APIs: []PublicAPIKind{
			AppLifecycleAPI,
			WindowSurfaceAPI,
			MountSurfaceAPI,
			EventSurfaceAPI,
			DialogSurfaceAPI,
			DropSurfaceAPI,
			RawHostAPI,
		},
	}
}

func SupportsPublicAPI(model PackageModel, api PublicAPIKind) bool {
	for _, available := range model.APIs {
		if available == api {
			return true
		}
	}
	return false
}

func ValidatePackage(model PackageModel) []PackageDiagnostic {
	var diagnostics []PackageDiagnostic
	if model.Import == "" {
		diagnostics = append(diagnostics, PackageDiagnostic{Field: "webview.import", Message: "webview package import must not be empty"})
	}
	if model.RuntimeImport == "" {
		diagnostics = append(diagnostics, PackageDiagnostic{Field: "webview.runtime", Message: "webview package runtime import must not be empty"})
	}
	if !model.UsesInternalRuntime {
		diagnostics = append(diagnostics, PackageDiagnostic{Field: "webview.runtime", Message: "webview package must use an internal runtime layer"})
	}
	for _, required := range []PublicAPIKind{
		AppLifecycleAPI,
		WindowSurfaceAPI,
		MountSurfaceAPI,
		EventSurfaceAPI,
		DialogSurfaceAPI,
		DropSurfaceAPI,
	} {
		if !SupportsPublicAPI(model, required) {
			diagnostics = append(diagnostics, PackageDiagnostic{
				Field:   "webview.apis",
				Message: "webview package must expose " + string(required),
			})
		}
	}
	return diagnostics
}
