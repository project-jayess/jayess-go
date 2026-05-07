package runtime

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type CryptoKey struct {
	Algorithm string
	Public    any
	Private   any
	Secret    []byte
}

type CryptoCertificateInfo struct {
	Subject    string
	Issuer     string
	DNSNames   []string
	NotBefore  time.Time
	NotAfter   time.Time
	IsCA       bool
	SerialText string
}

func CryptoEncrypt(algorithm string, key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	block, err := aesBlockForAlgorithm(algorithm, key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("nonce size must be %d bytes", aead.NonceSize())
	}
	return aead.Seal(nil, nonce, plaintext, nil), nil
}

func CryptoDecrypt(algorithm string, key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aesBlockForAlgorithm(algorithm, key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("nonce size must be %d bytes", aead.NonceSize())
	}
	return aead.Open(nil, nonce, ciphertext, nil)
}

func CryptoGenerateKey(algorithm string) (*CryptoKey, error) {
	switch normalizeCryptoAlgorithm(algorithm) {
	case "aes256gcm":
		secret, err := CryptoRandomBytes(32)
		if err != nil {
			return nil, err
		}
		return &CryptoKey{Algorithm: "aes-256-gcm", Secret: secret}, nil
	case "ed25519":
		public, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		return &CryptoKey{Algorithm: "ed25519", Public: public, Private: private}, nil
	case "rsaoaep", "rsa":
		private, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		return &CryptoKey{Algorithm: "rsa-oaep", Public: &private.PublicKey, Private: private}, nil
	default:
		return nil, fmt.Errorf("unsupported internal key algorithm %q", algorithm)
	}
}

func CryptoSign(key *CryptoKey, data []byte) ([]byte, error) {
	if private, ok := keyPrivateEd25519(key); ok {
		return ed25519.Sign(private, data), nil
	}
	return nil, fmt.Errorf("unsupported signing key")
}

func CryptoVerify(key *CryptoKey, data []byte, signature []byte) (bool, error) {
	public, ok := keyPublicEd25519(key)
	if !ok {
		return false, fmt.Errorf("unsupported verification key")
	}
	return ed25519.Verify(public, data, signature), nil
}

func CryptoPublicEncrypt(key *CryptoKey, data []byte) ([]byte, error) {
	public, ok := keyPublicRSA(key)
	if !ok {
		return nil, fmt.Errorf("unsupported public encryption key")
	}
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, public, data, nil)
}

func CryptoPrivateDecrypt(key *CryptoKey, data []byte) ([]byte, error) {
	private, ok := keyPrivateRSA(key)
	if !ok {
		return nil, fmt.Errorf("unsupported private decryption key")
	}
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, private, data, nil)
}

func CryptoExportKeyPEM(key *CryptoKey, private bool) ([]byte, error) {
	if key == nil {
		return nil, fmt.Errorf("key is required")
	}
	if private && key.Private != nil {
		der, err := x509.MarshalPKCS8PrivateKey(key.Private)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
	}
	public := key.Public
	if public == nil {
		if rsaPrivate, ok := key.Private.(*rsa.PrivateKey); ok {
			public = &rsaPrivate.PublicKey
		}
		if edPrivate, ok := key.Private.(ed25519.PrivateKey); ok {
			public = edPrivate.Public().(ed25519.PublicKey)
		}
	}
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func CryptoImportKeyPEM(data []byte) (*CryptoKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("PEM key is required")
	}
	if strings.Contains(block.Type, "PRIVATE") {
		private, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		switch typed := private.(type) {
		case ed25519.PrivateKey:
			return &CryptoKey{Algorithm: "ed25519", Public: typed.Public().(ed25519.PublicKey), Private: typed}, nil
		case *rsa.PrivateKey:
			return &CryptoKey{Algorithm: "rsa-oaep", Public: &typed.PublicKey, Private: typed}, nil
		default:
			return nil, fmt.Errorf("unsupported private key type %T", typed)
		}
	}
	public, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	switch typed := public.(type) {
	case ed25519.PublicKey:
		return &CryptoKey{Algorithm: "ed25519", Public: typed}, nil
	case *rsa.PublicKey:
		return &CryptoKey{Algorithm: "rsa-oaep", Public: typed}, nil
	default:
		return nil, fmt.Errorf("unsupported public key type %T", typed)
	}
}

func CryptoParseCertificatePEM(data []byte) (CryptoCertificateInfo, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return CryptoCertificateInfo{}, fmt.Errorf("PEM certificate is required")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CryptoCertificateInfo{}, err
	}
	return CryptoCertificateInfo{
		Subject:    cert.Subject.String(),
		Issuer:     cert.Issuer.String(),
		DNSNames:   append([]string(nil), cert.DNSNames...),
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		IsCA:       cert.IsCA,
		SerialText: serialText(cert.SerialNumber),
	}, nil
}

func aesBlockForAlgorithm(algorithm string, key []byte) (cipher.Block, error) {
	switch normalizeCryptoAlgorithm(algorithm) {
	case "aes128gcm":
		if len(key) != 16 {
			return nil, fmt.Errorf("aes-128-gcm key must be 16 bytes")
		}
	case "aes192gcm":
		if len(key) != 24 {
			return nil, fmt.Errorf("aes-192-gcm key must be 24 bytes")
		}
	case "aes256gcm", "aesgcm":
		if len(key) != 32 {
			return nil, fmt.Errorf("aes-256-gcm key must be 32 bytes")
		}
	default:
		return nil, fmt.Errorf("unsupported internal symmetric algorithm %q", algorithm)
	}
	return aes.NewCipher(key)
}

func keyPrivateEd25519(key *CryptoKey) (ed25519.PrivateKey, bool) {
	if key == nil {
		return nil, false
	}
	private, ok := key.Private.(ed25519.PrivateKey)
	return private, ok
}

func keyPublicEd25519(key *CryptoKey) (ed25519.PublicKey, bool) {
	if key == nil {
		return nil, false
	}
	if public, ok := key.Public.(ed25519.PublicKey); ok {
		return public, true
	}
	if private, ok := key.Private.(ed25519.PrivateKey); ok {
		return private.Public().(ed25519.PublicKey), true
	}
	return nil, false
}

func keyPrivateRSA(key *CryptoKey) (*rsa.PrivateKey, bool) {
	if key == nil {
		return nil, false
	}
	private, ok := key.Private.(*rsa.PrivateKey)
	return private, ok
}

func keyPublicRSA(key *CryptoKey) (*rsa.PublicKey, bool) {
	if key == nil {
		return nil, false
	}
	if public, ok := key.Public.(*rsa.PublicKey); ok {
		return public, true
	}
	if private, ok := key.Private.(*rsa.PrivateKey); ok {
		return &private.PublicKey, true
	}
	return nil, false
}

func serialText(serial *big.Int) string {
	if serial == nil {
		return ""
	}
	return serial.String()
}
