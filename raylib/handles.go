package raylib

type HandleKind string

const (
	WindowHandle        HandleKind = "window"
	ImageHandle         HandleKind = "image"
	TextureHandle       HandleKind = "texture"
	RenderTextureHandle HandleKind = "render-texture"
	FontHandle          HandleKind = "font"
	SoundHandle         HandleKind = "sound"
	MusicHandle         HandleKind = "music"
	AudioStreamHandle   HandleKind = "audio-stream"
)

func SupportsHandle(kind HandleKind) bool {
	for _, available := range HandleKinds() {
		if available == kind {
			return true
		}
	}
	return false
}

func HandleKinds() []HandleKind {
	return []HandleKind{
		WindowHandle,
		ImageHandle,
		TextureHandle,
		RenderTextureHandle,
		FontHandle,
		SoundHandle,
		MusicHandle,
		AudioStreamHandle,
	}
}

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func RGBA(r uint8, g uint8, b uint8, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}
