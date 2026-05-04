package mongoose

type EventRule string

const (
	JayessCallbackSafe    EventRule = "jayess-callback-safe"
	CallbackLifetimeSafe  EventRule = "callback-lifetime-safe"
	ServerLoopIntegration EventRule = "server-loop-integration"
	ErrorDiagnostics      EventRule = "mongoose-error-diagnostics"
)

func EventRules() []EventRule {
	return []EventRule{
		JayessCallbackSafe,
		CallbackLifetimeSafe,
		ServerLoopIntegration,
		ErrorDiagnostics,
	}
}
