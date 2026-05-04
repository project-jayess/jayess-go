package diagnostics

type RuntimeErrorKind string

const (
	ThrownException   RuntimeErrorKind = "thrown-exception"
	UncaughtException RuntimeErrorKind = "uncaught-exception"
	InvalidRuntimeUse RuntimeErrorKind = "invalid-runtime-use"
	NativeCallFailure RuntimeErrorKind = "native-call-failure"
	AsyncTaskFailure  RuntimeErrorKind = "async-task-failure"
)

func RuntimeErrorKinds() []RuntimeErrorKind {
	return []RuntimeErrorKind{
		ThrownException,
		UncaughtException,
		InvalidRuntimeUse,
		NativeCallFailure,
		AsyncTaskFailure,
	}
}
