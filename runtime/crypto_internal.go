package runtime

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

func CryptoRandomBytes(size int) ([]byte, error) {
	if size < 0 {
		return nil, fmt.Errorf("random byte size must not be negative")
	}
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func CryptoHash(algorithm string, data []byte) ([]byte, error) {
	builder, err := cryptoHashBuilder(algorithm)
	if err != nil {
		return nil, err
	}
	digest := builder()
	if _, err := digest.Write(data); err != nil {
		return nil, err
	}
	return digest.Sum(nil), nil
}

func CryptoHashHex(algorithm string, data []byte) (string, error) {
	digest, err := CryptoHash(algorithm, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

func CryptoHMAC(algorithm string, key []byte, data []byte) ([]byte, error) {
	builder, err := cryptoHashBuilder(algorithm)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(builder, key)
	if _, err := mac.Write(data); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func CryptoHMACHex(algorithm string, key []byte, data []byte) (string, error) {
	digest, err := CryptoHMAC(algorithm, key, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

func CryptoSecureCompare(left []byte, right []byte) bool {
	return hmac.Equal(left, right)
}

func cryptoHashBuilder(algorithm string) (func() hash.Hash, error) {
	switch normalizeCryptoAlgorithm(algorithm) {
	case "md5":
		return md5.New, nil
	case "sha1":
		return sha1.New, nil
	case "sha224":
		return sha256.New224, nil
	case "sha256":
		return sha256.New, nil
	case "sha384":
		return sha512.New384, nil
	case "sha512":
		return sha512.New, nil
	default:
		return nil, fmt.Errorf("unsupported internal crypto algorithm %q", algorithm)
	}
}

func normalizeCryptoAlgorithm(algorithm string) string {
	algorithm = strings.ToLower(strings.TrimSpace(algorithm))
	algorithm = strings.ReplaceAll(algorithm, "-", "")
	algorithm = strings.ReplaceAll(algorithm, "_", "")
	return algorithm
}
