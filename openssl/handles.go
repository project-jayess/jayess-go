package openssl

type HandleKind string

const (
	SSLContextHandle HandleKind = "SSL_CTX"
	SSLHandle        HandleKind = "SSL"
	KeyHandle        HandleKind = "EVP_PKEY"
	CertHandle       HandleKind = "X509"
	CipherHandle     HandleKind = "EVP_CIPHER_CTX"
	DigestHandle     HandleKind = "EVP_MD_CTX"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: SSLContextHandle, Managed: true, Closable: true},
		{Kind: SSLHandle, Managed: true, Closable: true},
		{Kind: KeyHandle, Managed: true, Closable: true},
		{Kind: CertHandle, Managed: true, Closable: true},
		{Kind: CipherHandle, Managed: true, Closable: true},
		{Kind: DigestHandle, Managed: true, Closable: true},
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
