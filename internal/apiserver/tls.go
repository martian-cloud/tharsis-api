package apiserver

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func loadTLSConfig(cfg *config.Config, logger logger.Logger) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if cfg.TLSEnabled {
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile == "" {
			return nil, errors.New("TLS key file is required when using a custom certificate")
		}

		if cfg.TLSCertFile == "" && cfg.TLSKeyFile != "" {
			return nil, errors.New("TLS certificate file is required when using a custom key")
		}

		var cert tls.Certificate
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			logger.Infof("Starting server with TLS enabled: certPath=%s, keyPath=%s", cfg.TLSCertFile, cfg.TLSKeyFile)

			c, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load tls key pair: %w", err)
			}
			cert = c
		} else {
			logger.Info("Starting server with TLS enabled and self signed certificate")

			c, err := generateSelfSignedCert()
			if err != nil {
				return nil, fmt.Errorf("failed to generate self signed tls cert: %w", err)
			}
			cert = *c
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}
	return tlsConfig, nil
}

func generateSelfSignedCert() (*tls.Certificate, error) {
	// Generate a new private key
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ecdsa key: %w", err)
	}

	// Create certificate template
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Tharsis"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create cert
	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create x509: %w", err)
	}

	// Encode the cert to PEM format
	var buffer bytes.Buffer
	err = pem.Encode(&buffer, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return nil, fmt.Errorf("failed to encode cert to pem format: %w", err)
	}
	certPEM := buffer.Bytes()

	// Create PEM block
	privKeyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	// Write the key
	buffer = bytes.Buffer{}
	if err = pem.Encode(&buffer, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privKeyBytes}); err != nil {
		return nil, fmt.Errorf("failed to encode key: %w", err)
	}
	keyPEM := buffer.Bytes()

	// Create key pair
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create x509 key pair: %w", err)
	}

	return &cert, nil
}
