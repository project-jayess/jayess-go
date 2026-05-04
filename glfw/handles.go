package glfw

type HandleKind string

const (
	WindowHandle   HandleKind = "GLFWwindow"
	MonitorHandle  HandleKind = "GLFWmonitor"
	CursorHandle   HandleKind = "GLFWcursor"
	JoystickHandle HandleKind = "GLFWjoystick"
	VulkanSurface  HandleKind = "VkSurfaceKHR"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: WindowHandle, Managed: true, Closable: true},
		{Kind: MonitorHandle, Managed: false, Nullable: true},
		{Kind: CursorHandle, Managed: true, Closable: true, Nullable: true},
		{Kind: JoystickHandle, Managed: false, Nullable: true},
		{Kind: VulkanSurface, Managed: true, Closable: true, Nullable: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind {
			return true
		}
	}
	return false
}
