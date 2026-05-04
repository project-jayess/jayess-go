package libuv

type HandleKind string

const (
	LoopHandle    HandleKind = "uv_loop_t"
	TimerHandle   HandleKind = "uv_timer_t"
	TCPHandle     HandleKind = "uv_tcp_t"
	UDPHandle     HandleKind = "uv_udp_t"
	FSReqHandle   HandleKind = "uv_fs_t"
	ProcessHandle HandleKind = "uv_process_t"
	SignalHandle  HandleKind = "uv_signal_t"
)

type HandleRule struct {
	Kind          HandleKind
	Managed       bool
	Closable      bool
	LoopOwned     bool
	CallbackOwned bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: LoopHandle, Managed: true, Closable: true},
		{Kind: TimerHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
		{Kind: TCPHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
		{Kind: UDPHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
		{Kind: FSReqHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
		{Kind: ProcessHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
		{Kind: SignalHandle, Managed: true, Closable: true, LoopOwned: true, CallbackOwned: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind && rule.Managed {
			return true
		}
	}
	return false
}
