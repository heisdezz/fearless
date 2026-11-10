package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"software.sslmate.com/src/go-pkcs12"
)

// LoadCertificate reads and decodes a client certificate and private key.
func LoadCertificate(cfg ClientCertConfig, basePath string) (tls.Certificate, error) {
	switch cfg.NormalizedType() {
	case CertTypePFX:
		return loadPFXCertificate(cfg, basePath)
	default:
		return loadPEMCertificate(cfg, basePath)
	}
}

// loadPEMCertificate loads a standard PEM certificate and private key pair.
func loadPEMCertificate(cfg ClientCertConfig, basePath string) (tls.Certificate, error) {
	certPath := resolvePath(cfg.CertFilePath, basePath)
	keyPath := resolvePath(cfg.KeyFilePath, basePath)

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Error reading cert/key file: %w", err)
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Error reading cert/key file: %w", err)
	}

	// Handle encrypted private key if passphrase is provided
	if cfg.Passphrase != "" {
		decryptedKeyBytes, err := decryptPEMPrivateKey(keyBytes, cfg.Passphrase)
		if err == nil && len(decryptedKeyBytes) > 0 {
			keyBytes = decryptedKeyBytes
		}
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Error parsing cert/key pair: %w", err)
	}

	if len(cert.Certificate) > 0 {
		cert.Leaf, _ = x509.ParseCertificate(cert.Certificate[0])
	}
	return cert, nil
}

// loadPFXCertificate decodes a PKCS#12 (.p12 / .pfx) archive.
func loadPFXCertificate(cfg ClientCertConfig, basePath string) (tls.Certificate, error) {
	pfxPath := resolvePath(cfg.PFXFilePath, basePath)

	pfxBytes, err := os.ReadFile(pfxPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Error reading pfx file: %w", err)
	}

	// Use DecodeChain from go-pkcs12 for complete support of modern AES/SHA-256 and legacy algorithms
	priv, leaf, caChain, decodeErr := pkcs12.DecodeChain(pfxBytes, cfg.Passphrase)
	if decodeErr == nil && leaf != nil && priv != nil {
		certList := [][]byte{leaf.Raw}
		for _, ca := range caChain {
			certList = append(certList, ca.Raw)
		}
		tlsCert := tls.Certificate{
			Certificate: certList,
			PrivateKey:  priv,
			Leaf:        leaf,
		}
		return tlsCert, nil
	}

	// Fallback to ToPEM
	blocks, pemErr := pkcs12.ToPEM(pfxBytes, cfg.Passphrase)
	if pemErr == nil && len(blocks) > 0 {
		var certPEM []byte
		var keyPEM []byte
		for _, b := range blocks {
			if strings.EqualFold(b.Type, "CERTIFICATE") {
				certPEM = append(certPEM, pem.EncodeToMemory(b)...)
			} else if strings.Contains(strings.ToUpper(b.Type), "PRIVATE KEY") {
				keyPEM = append(keyPEM, pem.EncodeToMemory(b)...)
			}
		}
		if len(certPEM) > 0 && len(keyPEM) > 0 {
			cert, err := tls.X509KeyPair(certPEM, keyPEM)
			if err == nil {
				if len(cert.Certificate) > 0 {
					cert.Leaf, _ = x509.ParseCertificate(cert.Certificate[0])
				}
				return cert, nil
			}
		}
	}

	if decodeErr != nil {
		return tls.Certificate{}, fmt.Errorf("Error decoding pfx file: %w", decodeErr)
	}
	return tls.Certificate{}, fmt.Errorf("Error decoding pfx file: %w", pemErr)
}

// decryptPEMPrivateKey attempts to decrypt legacy PEM-encrypted private keys.
func decryptPEMPrivateKey(keyBytes []byte, passphrase string) ([]byte, error) {
	block, rest := pem.Decode(keyBytes)
	if block == nil {
		return keyBytes, nil
	}

	// Legacy x509 PEM decryption
	if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck
		der, err := x509.DecryptPEMBlock(block, []byte(passphrase)) //nolint:staticcheck
		if err != nil {
			return nil, err
		}
		decryptedBlock := &pem.Block{
			Type:  block.Type,
			Bytes: der,
		}
		return append(pem.EncodeToMemory(decryptedBlock), rest...), nil
	}

	return keyBytes, nil
}

// resolvePath joins non-absolute paths with the base path if provided.
func resolvePath(p, basePath string) string {
	if filepath.IsAbs(p) || basePath == "" {
		return p
	}
	return filepath.Join(basePath, p)
}
