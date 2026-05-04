package libcurl

type CoreFeature string

const (
	CreateEasyHandle  CoreFeature = "create-easy-handle"
	ConfigureRequest  CoreFeature = "configure-url-method-headers-body"
	PerformTransfer   CoreFeature = "perform-transfer"
	ReadResponse      CoreFeature = "read-status-headers-body"
	CleanupEasyHandle CoreFeature = "cleanup-easy-handle"
)

func CoreFeatures() []CoreFeature {
	return []CoreFeature{
		CreateEasyHandle,
		ConfigureRequest,
		PerformTransfer,
		ReadResponse,
		CleanupEasyHandle,
	}
}
