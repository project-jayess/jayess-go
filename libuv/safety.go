package libuv

type SafetyRule string

const (
	HandleLifetimeSafe   SafetyRule = "handle-lifetime-safe"
	CallbackLifetimeSafe SafetyRule = "callback-lifetime-safe"
	ThreadLoopOwnership  SafetyRule = "thread-loop-ownership"
	ErrorDiagnostics     SafetyRule = "libuv-error-diagnostics"
)

func SafetyRules() []SafetyRule {
	return []SafetyRule{
		HandleLifetimeSafe,
		CallbackLifetimeSafe,
		ThreadLoopOwnership,
		ErrorDiagnostics,
	}
}
