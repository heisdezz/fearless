package tls

import (
	"crypto/tls"
	"strings"

	"fearless/internal/core/model"
)

// CertType denotes the format of client certificate storage.
type CertType string

const (
	// CertTypePEM represents separate PEM certificate and private key files.
	CertTypePEM CertType = "cert"
	// CertTypePFX represents a PKCS#12 (.p12 / .pfx) certificate and key bundle.
	CertTypePFX CertType = "pfx"
)

// CertSource identifies where a selected client certificate originated.
type CertSource string

const (
	// CertSourceNone indicates no client certificate was selected.
	CertSourceNone CertSource = "none"
	// CertSourceCollection indicates the certificate was chosen from collection config.
	CertSourceCollection CertSource = "collection"
	// CertSourceGlobal indicates the certificate was chosen from global preferences.
	CertSourceGlobal CertSource = "global"
)

// ClientCertConfig specifies a single client certificate rule.
type ClientCertConfig struct {
	Domain       string   `json:"domain,omitempty"`
	Type         CertType `json:"type,omitempty"`
	CertFilePath string   `json:"certFilePath,omitempty"`
	KeyFilePath  string   `json:"keyFilePath,omitempty"`
	PFXFilePath  string   `json:"pfxFilePath,omitempty"`
	Passphrase   string   `json:"passphrase,omitempty"`
	Disabled     bool     `json:"disabled,omitempty"`
}

// NormalizedType returns the resolved CertType, defaulting to CertTypePEM.
func (c ClientCertConfig) NormalizedType() CertType {
	if strings.EqualFold(string(c.Type), string(CertTypePFX)) {
		return CertTypePFX
	}
	return CertTypePEM
}

// ClientCertificates groups a collection of client certificates with an overall enabled switch.
type ClientCertificates struct {
	Enabled bool               `json:"enabled"`
	Certs   []ClientCertConfig `json:"certs,omitempty"`
}

// FromModelClientCertRef converts a core model.ClientCertRef into a ClientCertConfig.
func FromModelClientCertRef(ref model.ClientCertRef) ClientCertConfig {
	return ClientCertConfig{
		Domain:       ref.Domain,
		Type:         CertTypePEM,
		CertFilePath: ref.CertFilePath,
		KeyFilePath:  ref.KeyFilePath,
		Passphrase:   ref.Passphrase,
	}
}

// FromModelClientCertificates converts a core model.ClientCertificates into a ClientCertificates set.
func FromModelClientCertificates(m *model.ClientCertificates) *ClientCertificates {
	if m == nil {
		return nil
	}
	certs := make([]ClientCertConfig, 0, len(m.Certificates))
	for _, ref := range m.Certificates {
		certs = append(certs, FromModelClientCertRef(ref))
	}
	return &ClientCertificates{
		Enabled: m.Enabled,
		Certs:   certs,
	}
}

// CAConfig defines how the certificate authority trust bundle is constructed.
type CAConfig struct {
	// CustomFilePath points to a user-provided CA certificate file (.pem or .crt).
	CustomFilePath string `json:"customFilePath,omitempty"`
	// CustomBytes provides raw CA certificate bytes in PEM format.
	CustomBytes []byte `json:"-"`
	// KeepDefaultCerts controls whether system roots are retained. Defaults to true.
	KeepDefaultCerts bool `json:"keepDefaultCerts"`
	// ExtraFilePath points to an optional extra CA bundle (or falls back to env vars).
	ExtraFilePath string `json:"extraFilePath,omitempty"`
}

// CACertCounts reports the number of loaded CA certificates by origin.
type CACertCounts struct {
	System int `json:"system"`
	Root   int `json:"root"`
	Custom int `json:"custom"`
	Extra  int `json:"extra"`
}

// Total returns the total count of certificates loaded.
func (c CACertCounts) Total() int {
	return c.System + c.Root + c.Custom + c.Extra
}

// BuildOptions parameters used to construct a *tls.Config for an outbound target.
type BuildOptions struct {
	// TargetURL or target hostname/address (e.g. "https://api.example.com/v1" or "api.example.com:443").
	TargetURL string

	// VerifyServerCertificates turns off TLS server verification when explicitly false.
	// If nil, defaults to true.
	VerifyServerCertificates *bool

	// InsecureSkipVerify directly forces server certificate verification off when true.
	InsecureSkipVerify bool

	// CA configures the trust pool.
	CA CAConfig

	// CollectionCerts are collection-scoped client certificates (higher precedence).
	CollectionCerts *ClientCertificates

	// GlobalCerts are app preferences client certificates (fallback).
	GlobalCerts *ClientCertificates

	// CollectionPath is the base directory used to resolve relative certificate paths.
	CollectionPath string

	// MinVersion is the minimum TLS version. Defaults to tls.VersionTLS12.
	MinVersion uint16

	// MaxVersion is the maximum TLS version. 0 uses Go's default.
	MaxVersion uint16

	// ServerName explicitly overrides the SNI hostname if desired.
	ServerName string

	// DisableSessionCache disables TLS session resumption caching when true.
	DisableSessionCache bool
}

// ResolutionResult returns the built tls.Config and diagnostic metadata.
type ResolutionResult struct {
	// TLSConfig is the configured crypto/tls.Config ready for http.Transport.
	TLSConfig *tls.Config

	// SelectedCert is the matched client certificate config, if any.
	SelectedCert *ClientCertConfig

	// CertSource indicates whether the cert was chosen from collection or global.
	CertSource CertSource

	// CACounts reflects the numbers of CA certificates loaded.
	CACounts CACertCounts

	// VerificationUsed indicates whether server certificate validation was active.
	VerificationUsed bool
}
