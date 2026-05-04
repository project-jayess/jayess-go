package audio

type DeviceKind string

const (
	OutputDevice DeviceKind = "output"
	InputDevice  DeviceKind = "input"
)

type SampleFormat string

const (
	Float32Format SampleFormat = "float32"
	Int16Format   SampleFormat = "int16"
	Int32Format   SampleFormat = "int32"
)

type DeviceCapability struct {
	Kind        DeviceKind
	Enumerable  bool
	Openable    bool
	SampleRates []int
	Channels    []int
	Formats     []SampleFormat
}

func DefaultDeviceCapabilities() []DeviceCapability {
	return []DeviceCapability{
		{
			Kind:        OutputDevice,
			Enumerable:  true,
			Openable:    true,
			SampleRates: []int{44100, 48000},
			Channels:    []int{1, 2},
			Formats:     []SampleFormat{Float32Format, Int16Format},
		},
		{
			Kind:        InputDevice,
			Enumerable:  true,
			Openable:    true,
			SampleRates: []int{44100, 48000},
			Channels:    []int{1, 2},
			Formats:     []SampleFormat{Float32Format, Int16Format},
		},
	}
}

func SupportsDeviceKind(kind DeviceKind) bool {
	for _, capability := range DefaultDeviceCapabilities() {
		if capability.Kind == kind && capability.Enumerable && capability.Openable {
			return true
		}
	}
	return false
}
