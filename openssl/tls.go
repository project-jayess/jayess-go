package openssl

type TLSFeature string

const (
	TLSClient            TLSFeature = "tls-client"
	TLSServer            TLSFeature = "tls-server"
	CertificateLoading   TLSFeature = "certificate-loading"
	TrustStoreConfig     TLSFeature = "trust-store-config"
	HostnameVerification TLSFeature = "hostname-verification"
	ALPN                 TLSFeature = "alpn"
)

func TLSFeatures() []TLSFeature {
	return []TLSFeature{
		TLSClient,
		TLSServer,
		CertificateLoading,
		TrustStoreConfig,
		HostnameVerification,
		ALPN,
	}
}
