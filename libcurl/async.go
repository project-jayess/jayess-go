package libcurl

type AsyncFeature string

const (
	MultiHandleSupport AsyncFeature = "multi-handle"
	JayessAsyncModel   AsyncFeature = "jayess-async-model"
	StreamingBody      AsyncFeature = "streaming-body"
)

func AsyncFeatures() []AsyncFeature {
	return []AsyncFeature{
		MultiHandleSupport,
		JayessAsyncModel,
		StreamingBody,
	}
}
