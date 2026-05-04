package audio

type LibraryTarget string

const (
	SDLAudioTarget       LibraryTarget = "sdl-audio"
	OpenALTarget         LibraryTarget = "openal"
	MiniaudioTarget      LibraryTarget = "miniaudio"
	PortAudioTarget      LibraryTarget = "portaudio"
	PlatformNativeTarget LibraryTarget = "platform-native"
)

func LibraryTargets() []LibraryTarget {
	return []LibraryTarget{
		SDLAudioTarget,
		OpenALTarget,
		MiniaudioTarget,
		PortAudioTarget,
		PlatformNativeTarget,
	}
}

func SupportsLibraryTarget(target LibraryTarget) bool {
	for _, supported := range LibraryTargets() {
		if supported == target {
			return true
		}
	}
	return false
}
