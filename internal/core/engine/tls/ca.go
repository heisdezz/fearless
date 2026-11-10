package tls

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// LoadCACertPool builds an x509.CertPool and tallies loaded certificates.
func LoadCACertPool(cfg CAConfig) (*x509.CertPool, CACertCounts, error) {
	var counts CACertCounts
	seen := make(map[[32]byte]bool)

	var pool *x509.CertPool
	hasCustom := cfg.CustomFilePath != "" || len(cfg.CustomBytes) > 0

	// Determine whether to start from system root certificates.
	// If custom CA is provided and KeepDefaultCerts is false, start clean.
	keepDefaults := cfg.KeepDefaultCerts
	if !hasCustom {
		// When no custom CA is specified, default roots are always kept.
		keepDefaults = true
	}

	if keepDefaults {
		sysPool, err := x509.SystemCertPool()
		if err == nil && sysPool != nil {
			pool = sysPool
			counts.System = 1 // Flag system pool as active
		} else {
			pool = x509.NewCertPool()
		}
	} else {
		pool = x509.NewCertPool()
	}

	// 1. Process custom CA certificate
	if hasCustom {
		customCerts, err := loadPEMCertificates(cfg.CustomFilePath, cfg.CustomBytes)
		if err != nil {
			return nil, counts, err
		}
		if len(customCerts) == 0 {
			src := cfg.CustomFilePath
			if src == "" {
				src = "custom CA bytes"
			}
			return nil, counts, fmt.Errorf("unable to load custom CA certificate from %s: no valid certificates found", src)
		}
		for _, cert := range customCerts {
			fingerprint := sha256.Sum256(cert.Raw)
			if !seen[fingerprint] {
				seen[fingerprint] = true
				pool.AddCert(cert)
				counts.Custom++
			}
		}
	}

	// 2. Process extra CA certificates (explicit path or NODE_EXTRA_CA_CERTS / SSL_CERT_FILE)
	extraPath := cfg.ExtraFilePath
	if extraPath == "" {
		if p := os.Getenv("NODE_EXTRA_CA_CERTS"); p != "" {
			extraPath = p
		} else if p := os.Getenv("SSL_CERT_FILE"); p != "" {
			extraPath = p
		}
	}

	if extraPath != "" {
		extraCerts, err := loadPEMCertificates(extraPath, nil)
		if err == nil {
			for _, cert := range extraCerts {
				fingerprint := sha256.Sum256(cert.Raw)
				if !seen[fingerprint] {
					seen[fingerprint] = true
					pool.AddCert(cert)
					counts.Extra++
				}
			}
		}
	}

	return pool, counts, nil
}

// loadPEMCertificates reads and parses all x509 certificates from a file or byte slice.
func loadPEMCertificates(filePath string, data []byte) ([]*x509.Certificate, error) {
	if filePath != "" {
		if _, err := os.Stat(filePath); err != nil {
			return nil, fmt.Errorf("invalid custom CA certificate path: %s", filePath)
		}
		var err error
		data, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read CA certificate from %s: %w", filePath, err)
		}
	}

	if len(data) == 0 {
		return nil, nil
	}

	var certs []*x509.Certificate
	rest := data
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if strings.EqualFold(block.Type, "CERTIFICATE") || strings.Contains(strings.ToUpper(block.Type), "CERTIFICATE") {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil && cert != nil {
				certs = append(certs, cert)
			}
		}
	}

	return certs, nil
}
