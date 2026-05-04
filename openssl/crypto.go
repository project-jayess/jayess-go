package openssl

type CryptoFeature string

const (
	Hashing              CryptoFeature = "hashing"
	HMAC                 CryptoFeature = "hmac"
	SymmetricEncryption  CryptoFeature = "symmetric-encryption"
	AsymmetricEncryption CryptoFeature = "asymmetric-encryption"
	DigitalSignatures    CryptoFeature = "digital-signatures"
	KeyGeneration        CryptoFeature = "key-generation"
	RandomBytes          CryptoFeature = "random-bytes"
)

func CryptoFeatures() []CryptoFeature {
	return []CryptoFeature{
		Hashing,
		HMAC,
		SymmetricEncryption,
		AsymmetricEncryption,
		DigitalSignatures,
		KeyGeneration,
		RandomBytes,
	}
}
