package openssl

type SafetyRule string

const (
	KeyCertificateLifetimeSafe SafetyRule = "key-certificate-lifetime-safe"
	OpenSSLErrorDiagnostics    SafetyRule = "openssl-error-diagnostics"
	VersionFeatureSafe         SafetyRule = "version-feature-safe"
)

func SafetyRules() []SafetyRule {
	return []SafetyRule{
		KeyCertificateLifetimeSafe,
		OpenSSLErrorDiagnostics,
		VersionFeatureSafe,
	}
}

type VersionPolicy struct {
	MinimumVersion string
	FeatureGates   []string
}

func DefaultVersionPolicy() VersionPolicy {
	return VersionPolicy{
		MinimumVersion: "1.1.1",
		FeatureGates:   []string{"tls1.3", "alpn", "evp"},
	}
}
