package raylib

type SafetyFeature string

const (
	CallbackLifetimeSafety SafetyFeature = "callback-lifetime-safety"
	ResourceHandleLifetime SafetyFeature = "resource-handle-lifetime"
	AsyncRuntimeCoexist    SafetyFeature = "async-runtime-coexistence"
	DiagnosticPropagation  SafetyFeature = "diagnostic-propagation"
)

func SafetyFeatures() []SafetyFeature {
	return []SafetyFeature{
		CallbackLifetimeSafety,
		ResourceHandleLifetime,
		AsyncRuntimeCoexist,
		DiagnosticPropagation,
	}
}
