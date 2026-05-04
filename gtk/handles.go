package gtk

type HandleKind string

const (
	ApplicationHandle HandleKind = "GtkApplication"
	WindowHandle      HandleKind = "GtkWindow"
	WidgetHandle      HandleKind = "GtkWidget"
	LayoutHandle      HandleKind = "GtkLayout"
	SignalHandle      HandleKind = "GtkSignal"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: ApplicationHandle, Managed: true, Closable: true},
		{Kind: WindowHandle, Managed: true, Closable: true},
		{Kind: WidgetHandle, Managed: true, Closable: true, Nullable: true},
		{Kind: LayoutHandle, Managed: true, Closable: true, Nullable: true},
		{Kind: SignalHandle, Managed: true, Closable: true},
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
