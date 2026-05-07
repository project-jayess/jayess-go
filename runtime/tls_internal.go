package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

type TLSCertificate struct {
	Certificate tls.Certificate
	CertPEM     []byte
	KeyPEM      []byte
}

type TLSTrustStore struct {
	Pool      *x509.CertPool
	PEM       []byte
	System    bool
	Subjects  int
	LoadError string
}

type TLSRuntimeConfig struct {
	ServerName         string
	Certificates       []TLSCertificate
	TrustStore         *TLSTrustStore
	ALPN               []string
	InsecureSkipVerify bool
	MinVersion         uint16
}

func NewTLSCertificate(certPEM []byte, keyPEM []byte) (TLSCertificate, error) {
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return TLSCertificate{}, err
	}
	return TLSCertificate{Certificate: certificate, CertPEM: append([]byte(nil), certPEM...), KeyPEM: append([]byte(nil), keyPEM...)}, nil
}

func NewTLSTrustStore(caPEM []byte) (*TLSTrustStore, error) {
	pool := x509.NewCertPool()
	if len(caPEM) > 0 && !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("trust store did not contain any PEM certificates")
	}
	return &TLSTrustStore{Pool: pool, PEM: append([]byte(nil), caPEM...), Subjects: len(pool.Subjects())}, nil
}

func NewSystemTLSTrustStore() *TLSTrustStore {
	pool, err := x509.SystemCertPool()
	store := &TLSTrustStore{System: true}
	if err != nil || pool == nil {
		store.Pool = x509.NewCertPool()
		if err != nil {
			store.LoadError = err.Error()
		}
		return store
	}
	store.Pool = pool
	store.Subjects = len(pool.Subjects())
	return store
}

func TLSClientConfig(config TLSRuntimeConfig) *tls.Config {
	tlsConfig := baseTLSConfig(config)
	tlsConfig.ServerName = config.ServerName
	if config.TrustStore != nil {
		tlsConfig.RootCAs = config.TrustStore.Pool
	}
	tlsConfig.InsecureSkipVerify = config.InsecureSkipVerify
	return tlsConfig
}

func TLSServerConfig(config TLSRuntimeConfig) *tls.Config {
	tlsConfig := baseTLSConfig(config)
	for _, certificate := range config.Certificates {
		tlsConfig.Certificates = append(tlsConfig.Certificates, certificate.Certificate)
	}
	return tlsConfig
}

func TLSWithALPN(config TLSRuntimeConfig, protocols []string) TLSRuntimeConfig {
	config.ALPN = append([]string(nil), protocols...)
	return config
}

func TLSVerifyHostname(certificate *x509.Certificate, hostname string) error {
	if certificate == nil {
		return fmt.Errorf("certificate is required")
	}
	return certificate.VerifyHostname(hostname)
}

func baseTLSConfig(config TLSRuntimeConfig) *tls.Config {
	minVersion := config.MinVersion
	if minVersion == 0 {
		minVersion = tls.VersionTLS12
	}
	return &tls.Config{
		MinVersion: minVersion,
		NextProtos: append([]string(nil), config.ALPN...),
	}
}
