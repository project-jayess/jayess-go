package audio

type PlaybackFeature string

const (
	OpenPlaybackDevice PlaybackFeature = "open-playback-device"
	OpenCaptureDevice  PlaybackFeature = "open-capture-device"
	ConfigureFormat    PlaybackFeature = "configure-format"
	StartPlayback      PlaybackFeature = "start-playback"
	StopPlayback       PlaybackFeature = "stop-playback"
	PausePlayback      PlaybackFeature = "pause-playback"
	SubmitAudioBuffer  PlaybackFeature = "submit-audio-buffer"
	StreamingPlayback  PlaybackFeature = "streaming-playback"
)

func PlaybackFeatures() []PlaybackFeature {
	return []PlaybackFeature{
		OpenPlaybackDevice,
		OpenCaptureDevice,
		ConfigureFormat,
		StartPlayback,
		StopPlayback,
		PausePlayback,
		SubmitAudioBuffer,
		StreamingPlayback,
	}
}
