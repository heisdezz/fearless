package tls

import (
	"crypto/tls"
	"path/filepath"
)

// Resolve constructs a crypto/tls.Config according to the specified options,
// implementing Bruno-compatible CA loading, verification rules, and the
// four-way client certificate precedence matrix.
func Resolve(opts BuildOptions) (*ResolutionResult, error) {
	tlsConfig := &tls.Config{}

	// 1. Determine verification setting
	verify := true
	if opts.InsecureSkipVerify {
		verify = false
	} else if opts.VerifyServerCertificates != nil {
		verify = *opts.VerifyServerCertificates
	}

	result := &ResolutionResult{
		CertSource:       CertSourceNone,
		VerificationUsed: verify,
	}

	// 2. Configure CA trust bundle
	if !verify {
		// Verification disabled: CA loading is skipped entirely
		tlsConfig.InsecureSkipVerify = true
	} else {
		caCfg := opts.CA
		if caCfg.CustomFilePath != "" && !filepath.IsAbs(caCfg.CustomFilePath) && opts.CollectionPath != "" {
			caCfg.CustomFilePath = filepath.Join(opts.CollectionPath, caCfg.CustomFilePath)
		}
		if caCfg.ExtraFilePath != "" && !filepath.IsAbs(caCfg.ExtraFilePath) && opts.CollectionPath != "" {
			caCfg.ExtraFilePath = filepath.Join(opts.CollectionPath, caCfg.ExtraFilePath)
		}
		pool, counts, err := LoadCACertPool(caCfg)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = pool
		result.CACounts = counts
	}

	// 3. Client certificate resolution: Collection takes precedence over Global
	var (
		matchedCert   *ClientCertConfig
		matchedSource CertSource
	)

	// Check collection certificates first
	if opts.CollectionCerts != nil && opts.CollectionCerts.Enabled {
		for i := range opts.CollectionCerts.Certs {
			certRef := &opts.CollectionCerts.Certs[i]
			if certRef.Disabled {
				continue
			}
			if MatchDomain(certRef.Domain, opts.TargetURL) {
				matchedCert = certRef
				matchedSource = CertSourceCollection
				break
			}
		}
	}

	// If no collection certificate matched, check global certificates
	if matchedCert == nil && opts.GlobalCerts != nil && opts.GlobalCerts.Enabled {
		for i := range opts.GlobalCerts.Certs {
			certRef := &opts.GlobalCerts.Certs[i]
			if certRef.Disabled {
				continue
			}
			if MatchDomain(certRef.Domain, opts.TargetURL) {
				matchedCert = certRef
				matchedSource = CertSourceGlobal
				break
			}
		}
	}

	// Load matched client certificate if found
	if matchedCert != nil {
		basePath := ""
		if matchedSource == CertSourceCollection {
			basePath = opts.CollectionPath
		}

		loadedCert, err := LoadCertificate(*matchedCert, basePath)
		if err != nil {
			return nil, err
		}

		tlsConfig.Certificates = []tls.Certificate{loadedCert}
		tlsConfig.GetClientCertificate = func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &loadedCert, nil
		}
		result.SelectedCert = matchedCert
		result.CertSource = matchedSource
	}

	// 4. Protocol versions
	if opts.MinVersion != 0 {
		tlsConfig.MinVersion = opts.MinVersion
	} else {
		// Default to modern TLS 1.2 minimum
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	if opts.MaxVersion != 0 {
		tlsConfig.MaxVersion = opts.MaxVersion
	}

	// 5. Hostname / ServerName for SNI
	if opts.ServerName != "" {
		tlsConfig.ServerName = opts.ServerName
	} else if host := extractHostname(opts.TargetURL); host != "" {
		tlsConfig.ServerName = host
	}

	// 6. Session resumption caching
	if !opts.DisableSessionCache {
		tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(64)
	}

	result.TLSConfig = tlsConfig
	return result, nil
}

// ResolveVerifyTLS calculates the effective TLS verification boolean using standard precedence:
// Request > Collection > App Preferences > default true.
func ResolveVerifyTLS(requestVerify, collectionVerify, settingsVerify *bool) bool {
	if requestVerify != nil {
		return *requestVerify
	}
	if collectionVerify != nil {
		return *collectionVerify
	}
	if settingsVerify != nil {
		return *settingsVerify
	}
	return true
}
