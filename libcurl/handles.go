package libcurl

type HandleKind string

const (
	EasyHandle  HandleKind = "CURL"
	MultiHandle HandleKind = "CURLM"
	HeaderList  HandleKind = "curl_slist"
	MimeHandle  HandleKind = "curl_mime"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: EasyHandle, Managed: true, Closable: true},
		{Kind: MultiHandle, Managed: true, Closable: true, Nullable: true},
		{Kind: HeaderList, Managed: true, Closable: true, Nullable: true},
		{Kind: MimeHandle, Managed: true, Closable: true, Nullable: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind && rule.Managed {
			return true
		}
	}
	return false
}
