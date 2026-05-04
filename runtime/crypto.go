package runtime

type CryptoCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func CryptoCapabilities() []CryptoCapability {
	return []CryptoCapability{
		{Name: "randomBytes", RuntimeSymbol: "jayess_crypto_random_bytes", Kind: "function"},
		{Name: "hash", RuntimeSymbol: "jayess_crypto_hash", Kind: "function"},
		{Name: "hmac", RuntimeSymbol: "jayess_crypto_hmac", Kind: "function"},
		{Name: "encrypt", RuntimeSymbol: "jayess_crypto_encrypt", Kind: "function"},
		{Name: "decrypt", RuntimeSymbol: "jayess_crypto_decrypt", Kind: "function"},
		{Name: "publicEncrypt", RuntimeSymbol: "jayess_crypto_public_encrypt", Kind: "function"},
		{Name: "privateDecrypt", RuntimeSymbol: "jayess_crypto_private_decrypt", Kind: "function"},
		{Name: "sign", RuntimeSymbol: "jayess_crypto_sign", Kind: "function"},
		{Name: "verify", RuntimeSymbol: "jayess_crypto_verify", Kind: "function"},
		{Name: "generateKey", RuntimeSymbol: "jayess_crypto_generate_key", Kind: "function"},
		{Name: "secureCompare", RuntimeSymbol: "jayess_crypto_secure_compare", Kind: "function"},
	}
}

func HasCryptoCapability(name string) bool {
	for _, capability := range CryptoCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
