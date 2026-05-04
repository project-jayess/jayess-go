package audio

type Codec string

const (
	WAVCodec  Codec = "wav"
	OGGCodec  Codec = "ogg"
	MP3Codec  Codec = "mp3"
	FLACCodec Codec = "flac"
	PCMCodec  Codec = "pcm"
)

type AssetSupport struct {
	Codec                 Codec
	DecodedAsJayessBuffer bool
}

func AssetFormats() []AssetSupport {
	return []AssetSupport{
		{Codec: WAVCodec, DecodedAsJayessBuffer: true},
		{Codec: OGGCodec, DecodedAsJayessBuffer: true},
		{Codec: MP3Codec, DecodedAsJayessBuffer: true},
		{Codec: FLACCodec, DecodedAsJayessBuffer: true},
		{Codec: PCMCodec, DecodedAsJayessBuffer: true},
	}
}

func SupportsCodec(codec Codec) bool {
	for _, format := range AssetFormats() {
		if format.Codec == codec {
			return true
		}
	}
	return false
}
