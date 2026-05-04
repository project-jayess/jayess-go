package binding

type UseCase string

const (
	EngineAPIUseCase           UseCase = "engine-api"
	PlatformAPIUseCase         UseCase = "platform-api"
	RenderingAudioInputUseCase UseCase = "rendering-audio-input"
	ThirdPartyCUseCase         UseCase = "third-party-c"
	PerformanceCriticalUseCase UseCase = "performance-critical"
)

func SupportedUseCases() []UseCase {
	return []UseCase{
		EngineAPIUseCase,
		PlatformAPIUseCase,
		RenderingAudioInputUseCase,
		ThirdPartyCUseCase,
		PerformanceCriticalUseCase,
	}
}
