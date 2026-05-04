package libcurl

type AdvancedFeature string

const (
	HTTPS          AdvancedFeature = "https"
	Redirects      AdvancedFeature = "redirects"
	Timeouts       AdvancedFeature = "timeouts"
	Upload         AdvancedFeature = "upload"
	DownloadToFile AdvancedFeature = "download-to-file"
	Cookies        AdvancedFeature = "cookies"
	Proxy          AdvancedFeature = "proxy"
)

func AdvancedFeatures() []AdvancedFeature {
	return []AdvancedFeature{
		HTTPS,
		Redirects,
		Timeouts,
		Upload,
		DownloadToFile,
		Cookies,
		Proxy,
	}
}
