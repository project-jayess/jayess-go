package mongoose

type ServerFeature string

const (
	CreateManager ServerFeature = "create-manager"
	BindListen    ServerFeature = "bind-listen"
	AcceptRequest ServerFeature = "accept-http-request"
	ReadRequest   ServerFeature = "read-method-path-query-headers-body"
	SendResponse  ServerFeature = "send-status-headers-body"
	CleanShutdown ServerFeature = "clean-shutdown"
)

func ServerFeatures() []ServerFeature {
	return []ServerFeature{
		CreateManager,
		BindListen,
		AcceptRequest,
		ReadRequest,
		SendResponse,
		CleanShutdown,
	}
}
