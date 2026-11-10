package tls_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	coretls "fearless/internal/core/engine/tls"
	"fearless/internal/core/model"
)

type testCerts struct {
	dir              string
	caCertPath       string
	serverCertPath   string
	serverKeyPath    string
	colClientCert    string
	colClientKey     string
	globalClientCert string
	globalClientKey  string
	untrustedCert    string
	untrustedKey     string
	clientPfxPath    string
	clientPfxPass    string
	encryptedKeyPath string
	encryptedKeyPass string
}

func generateTestCertificates(t *testing.T) *testCerts {
	t.Helper()
	dir := t.TempDir()

	// 1. Generate CA
	caKey := filepath.Join(dir, "ca-key.pem")
	caCert := filepath.Join(dir, "ca-cert.pem")
	runCmd(t, dir, "openssl", "req", "-x509", "-newkey", "rsa:2048", "-days", "1", "-nodes",
		"-keyout", caKey, "-out", caCert, "-subj", "/CN=Fearless Test CA")

	// 2. Generate Server cert (signed by CA, SAN=localhost, 127.0.0.1)
	serverKey := filepath.Join(dir, "server-key.pem")
	serverCsr := filepath.Join(dir, "server.csr")
	serverCert := filepath.Join(dir, "server-cert.pem")
	extFile := filepath.Join(dir, "server.ext")
	if err := os.WriteFile(extFile, []byte("subjectAltName=DNS:localhost,IP:127.0.0.1\nextendedKeyUsage=serverAuth\n"), 0o600); err != nil {
		t.Fatalf("failed to write server.ext: %v", err)
	}
	runCmd(t, dir, "openssl", "req", "-newkey", "rsa:2048", "-nodes",
		"-keyout", serverKey, "-out", serverCsr, "-subj", "/CN=localhost")
	runCmd(t, dir, "openssl", "x509", "-req", "-in", serverCsr, "-CA", caCert, "-CAkey", caKey,
		"-CAcreateserial", "-out", serverCert, "-days", "1", "-extfile", extFile)

	// 3. Generate Collection Client Cert (signed by CA, CN=collection-client)
	colKey := filepath.Join(dir, "col-client-key.pem")
	colCsr := filepath.Join(dir, "col-client.csr")
	colCert := filepath.Join(dir, "col-client-cert.pem")
	clientExt := filepath.Join(dir, "client.ext")
	if err := os.WriteFile(clientExt, []byte("extendedKeyUsage=clientAuth\n"), 0o600); err != nil {
		t.Fatalf("failed to write client.ext: %v", err)
	}
	runCmd(t, dir, "openssl", "req", "-newkey", "rsa:2048", "-nodes",
		"-keyout", colKey, "-out", colCsr, "-subj", "/CN=collection-client")
	runCmd(t, dir, "openssl", "x509", "-req", "-in", colCsr, "-CA", caCert, "-CAkey", caKey,
		"-CAcreateserial", "-out", colCert, "-days", "1", "-extfile", clientExt)

	// 4. Generate Global Client Cert (signed by CA, CN=global-client)
	globalKey := filepath.Join(dir, "global-client-key.pem")
	globalCsr := filepath.Join(dir, "global-client.csr")
	globalCert := filepath.Join(dir, "global-client-cert.pem")
	runCmd(t, dir, "openssl", "req", "-newkey", "rsa:2048", "-nodes",
		"-keyout", globalKey, "-out", globalCsr, "-subj", "/CN=global-client")
	runCmd(t, dir, "openssl", "x509", "-req", "-in", globalCsr, "-CA", caCert, "-CAkey", caKey,
		"-CAcreateserial", "-out", globalCert, "-days", "1", "-extfile", clientExt)

	// 5. Generate Untrusted Client Cert (self-signed, rejected by server)
	untrustedKey := filepath.Join(dir, "untrusted-key.pem")
	untrustedCert := filepath.Join(dir, "untrusted-cert.pem")
	runCmd(t, dir, "openssl", "req", "-x509", "-newkey", "rsa:2048", "-days", "1", "-nodes",
		"-keyout", untrustedKey, "-out", untrustedCert, "-subj", "/CN=untrusted-client")

	// 6. Generate PKCS#12 bundle (client.pfx, passphrase: "bruno")
	clientPfx := filepath.Join(dir, "client.pfx")
	pfxPass := "bruno"
	runCmd(t, dir, "openssl", "pkcs12", "-export", "-out", clientPfx,
		"-inkey", colKey, "-in", colCert, "-certfile", caCert,
		"-password", "pass:"+pfxPass)

	// 7. Generate Encrypted PEM Private Key (passphrase: "secretpass")
	encKey := filepath.Join(dir, "encrypted-key.pem")
	encPass := "secretpass"
	runCmd(t, dir, "openssl", "rsa", "-in", colKey, "-des3", "-traditional", "-out", encKey,
		"-passout", "pass:"+encPass)

	return &testCerts{
		dir:              dir,
		caCertPath:       caCert,
		serverCertPath:   serverCert,
		serverKeyPath:    serverKey,
		colClientCert:    colCert,
		colClientKey:     colKey,
		globalClientCert: globalCert,
		globalClientKey:  globalKey,
		untrustedCert:    untrustedCert,
		untrustedKey:     untrustedKey,
		clientPfxPath:    clientPfx,
		clientPfxPass:    pfxPass,
		encryptedKeyPath: encKey,
		encryptedKeyPass: encPass,
	}
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\nOutput: %s", name, args, err, string(out))
	}
}

// startMTLSServer runs a TLS test server requiring client certificate signed by caCert.
func startMTLSServer(t *testing.T, certs *testCerts) *httptest.Server {
	t.Helper()
	serverCert, err := tls.LoadX509KeyPair(certs.serverCertPath, certs.serverKeyPath)
	if err != nil {
		t.Fatalf("failed to load server cert: %v", err)
	}

	caBytes, err := os.ReadFile(certs.caCertPath)
	if err != nil {
		t.Fatalf("failed to read CA: %v", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caBytes)

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			peer := r.TLS.PeerCertificates[0]
			w.Header().Set("X-Peer-CN", peer.Subject.CommonName)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "mTLS OK CN=%s", peer.Subject.CommonName)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("No client cert presented"))
	}))

	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}
	ts.StartTLS()
	return ts
}

func TestDomainMatcher(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"localhost", "https://localhost:8443/api", true},
		{"localhost", "localhost", true},
		{"localhost", "http://localhost", true},
		{"localhost", "https://localhost", true},
		{"*.example.com", "https://api.example.com/v1", true},
		{"*.example.com", "sub.api.example.com", true},
		{"*.example.com", "example.com", true},
		{"api.example.com", "https://api.example.com", true},
		{"api.example.com", "https://other.example.com", false},
		{"grpc://api.service", "grpc://api.service:50051", true},
		{"wss://stream.example.com", "wss://stream.example.com/feed", true},
		{"127.0.0.1", "https://127.0.0.1:9090", true},
		{"", "https://example.com", false},
		{"*.example.com", "", false},
		{"*", "https://anything.com", true},
		{"*.foo.bar.com", "baz.foo.bar.com", true},
		{"foo.com:8443", "https://foo.com:8443/test", true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.pattern, tc.target), func(t *testing.T) {
			got := coretls.MatchDomain(tc.pattern, tc.target)
			if got != tc.want {
				t.Errorf("MatchDomain(%q, %q) = %v; want %v", tc.pattern, tc.target, got, tc.want)
			}
		})
	}
}

func TestResolveVerifyTLS(t *testing.T) {
	tr := true
	fa := false

	tests := []struct {
		name     string
		req      *bool
		col      *bool
		settings *bool
		want     bool
	}{
		{"all nil defaults true", nil, nil, nil, true},
		{"settings false", nil, nil, &fa, false},
		{"col overrides settings", nil, &tr, &fa, true},
		{"req overrides all", &fa, &tr, &tr, false},
		{"req true overrides col false", &tr, &fa, &fa, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := coretls.ResolveVerifyTLS(tc.req, tc.col, tc.settings)
			if got != tc.want {
				t.Errorf("ResolveVerifyTLS() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestFourWayPrecedenceMatrix(t *testing.T) {
	certs := generateTestCertificates(t)

	// Case 1: Both collection and global define matching certs. Collection MUST take precedence!
	t.Run("Case1_CollectionPrecedenceOverGlobal", func(t *testing.T) {
		opts := coretls.BuildOptions{
			TargetURL: "https://localhost:8443",
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.colClientCert,
						KeyFilePath:  certs.colClientKey,
					},
				},
			},
			GlobalCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.globalClientCert,
						KeyFilePath:  certs.globalClientKey,
					},
				},
			},
		}

		res, err := coretls.Resolve(opts)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if res.CertSource != coretls.CertSourceCollection {
			t.Errorf("CertSource = %v; want %v", res.CertSource, coretls.CertSourceCollection)
		}
		if res.SelectedCert == nil || res.SelectedCert.CertFilePath != certs.colClientCert {
			t.Errorf("expected collection cert to be selected, got: %+v", res.SelectedCert)
		}
		if len(res.TLSConfig.Certificates) != 1 {
			t.Fatalf("expected 1 client certificate configured")
		}
		if res.TLSConfig.Certificates[0].Leaf.Subject.CommonName != "collection-client" {
			t.Errorf("expected CN=collection-client, got %s", res.TLSConfig.Certificates[0].Leaf.Subject.CommonName)
		}
	})

	// Case 2: Collection cert enabled, Global disabled -> Collection used
	t.Run("Case2_CollectionEnabled_GlobalDisabled", func(t *testing.T) {
		opts := coretls.BuildOptions{
			TargetURL: "https://localhost:8443",
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.colClientCert,
						KeyFilePath:  certs.colClientKey,
					},
				},
			},
			GlobalCerts: &coretls.ClientCertificates{
				Enabled: false,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.globalClientCert,
						KeyFilePath:  certs.globalClientKey,
					},
				},
			},
		}

		res, err := coretls.Resolve(opts)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if res.CertSource != coretls.CertSourceCollection {
			t.Errorf("expected collection cert, got: %v", res.CertSource)
		}
		if res.TLSConfig.Certificates[0].Leaf.Subject.CommonName != "collection-client" {
			t.Errorf("expected CN=collection-client, got %s", res.TLSConfig.Certificates[0].Leaf.Subject.CommonName)
		}
	})

	// Case 3: Collection cert disabled, Global cert enabled -> Global wins
	t.Run("Case3_CollectionDisabled_GlobalEnabled", func(t *testing.T) {
		opts := coretls.BuildOptions{
			TargetURL: "https://localhost:8443",
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.colClientCert,
						KeyFilePath:  certs.colClientKey,
						Disabled:     true, // Disabled!
					},
				},
			},
			GlobalCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "localhost",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.globalClientCert,
						KeyFilePath:  certs.globalClientKey,
					},
				},
			},
		}

		res, err := coretls.Resolve(opts)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if res.CertSource != coretls.CertSourceGlobal {
			t.Errorf("expected global cert source, got: %v", res.CertSource)
		}
		if res.SelectedCert.CertFilePath != certs.globalClientCert {
			t.Errorf("expected global cert path, got: %s", res.SelectedCert.CertFilePath)
		}
		if res.TLSConfig.Certificates[0].Leaf.Subject.CommonName != "global-client" {
			t.Errorf("expected CN=global-client, got %s", res.TLSConfig.Certificates[0].Leaf.Subject.CommonName)
		}
	})

	// Case 4: Both collection and global disabled -> No client cert configured
	t.Run("Case4_BothDisabled", func(t *testing.T) {
		opts := coretls.BuildOptions{
			TargetURL: "https://localhost:8443",
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: false,
			},
			GlobalCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:   "localhost",
						Disabled: true,
					},
				},
			},
		}

		res, err := coretls.Resolve(opts)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if res.CertSource != coretls.CertSourceNone {
			t.Errorf("expected CertSourceNone, got %v", res.CertSource)
		}
		if len(res.TLSConfig.Certificates) != 0 {
			t.Errorf("expected 0 client certs, got %d", len(res.TLSConfig.Certificates))
		}
	})
}

func TestLiveMTLSServerRoundTrip(t *testing.T) {
	certs := generateTestCertificates(t)
	server := startMTLSServer(t, certs)
	defer server.Close()

	// 1. Successful mTLS round-trip with PEM collection cert
	t.Run("PEM_mTLS_Success", func(t *testing.T) {
		res, err := coretls.Resolve(coretls.BuildOptions{
			TargetURL: server.URL,
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "127.0.0.1",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.colClientCert,
						KeyFilePath:  certs.colClientKey,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: res.TLSConfig,
			},
		}
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("mTLS GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if cn := resp.Header.Get("X-Peer-CN"); cn != "collection-client" {
			t.Errorf("expected peer CN collection-client, got %s", cn)
		}
	})

	// 2. Successful mTLS round-trip with PKCS#12 (.pfx) bundle
	t.Run("PFX_mTLS_Success", func(t *testing.T) {
		res, err := coretls.Resolve(coretls.BuildOptions{
			TargetURL: server.URL,
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:      "127.0.0.1",
						Type:        coretls.CertTypePFX,
						PFXFilePath: certs.clientPfxPath,
						Passphrase:  certs.clientPfxPass,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: res.TLSConfig,
			},
		}
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("PFX mTLS GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if cn := resp.Header.Get("X-Peer-CN"); cn != "collection-client" {
			t.Errorf("expected peer CN collection-client, got %s", cn)
		}
	})

	// 3. Handshake rejection when untrusted client cert is presented
	t.Run("Untrusted_Client_Cert_Rejected", func(t *testing.T) {
		res, err := coretls.Resolve(coretls.BuildOptions{
			TargetURL: server.URL,
			CA: coretls.CAConfig{
				CustomFilePath: certs.caCertPath,
			},
			CollectionCerts: &coretls.ClientCertificates{
				Enabled: true,
				Certs: []coretls.ClientCertConfig{
					{
						Domain:       "127.0.0.1",
						Type:         coretls.CertTypePEM,
						CertFilePath: certs.untrustedCert,
						KeyFilePath:  certs.untrustedKey,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: res.TLSConfig,
			},
		}
		_, err = client.Get(server.URL)
		if err == nil {
			t.Fatalf("expected mTLS handshake error for untrusted client certificate, but request succeeded")
		}
	})
}

func TestEncryptedPEMPrivateKey(t *testing.T) {
	certs := generateTestCertificates(t)

	cert, err := coretls.LoadCertificate(coretls.ClientCertConfig{
		Type:         coretls.CertTypePEM,
		CertFilePath: certs.colClientCert,
		KeyFilePath:  certs.encryptedKeyPath,
		Passphrase:   certs.encryptedKeyPass,
	}, "")
	if err != nil {
		t.Fatalf("failed to load encrypted key: %v", err)
	}

	if cert.Leaf.Subject.CommonName != "collection-client" {
		t.Errorf("expected CommonName collection-client, got %s", cert.Leaf.Subject.CommonName)
	}
}

func TestCABundleLoading(t *testing.T) {
	certs := generateTestCertificates(t)

	// Valid custom CA
	pool, counts, err := coretls.LoadCACertPool(coretls.CAConfig{
		CustomFilePath:   certs.caCertPath,
		KeepDefaultCerts: true,
	})
	if err != nil {
		t.Fatalf("LoadCACertPool failed: %v", err)
	}
	if pool == nil {
		t.Fatalf("expected non-nil pool")
	}
	if counts.Custom != 1 {
		t.Errorf("counts.Custom = %d; want 1", counts.Custom)
	}
	if counts.Total() < 1 {
		t.Errorf("counts.Total() = %d; want >= 1", counts.Total())
	}

	// Custom CA via bytes with KeepDefaultCerts false
	caBytes, err := os.ReadFile(certs.caCertPath)
	if err != nil {
		t.Fatalf("failed to read CA bytes: %v", err)
	}
	poolBytes, countsBytes, err := coretls.LoadCACertPool(coretls.CAConfig{
		CustomBytes:      caBytes,
		KeepDefaultCerts: false,
	})
	if err != nil {
		t.Fatalf("LoadCACertPool via bytes failed: %v", err)
	}
	if poolBytes == nil || countsBytes.Custom != 1 || countsBytes.System != 0 {
		t.Errorf("unexpected counts: %+v", countsBytes)
	}

	// Extra CA bundle path
	poolExtra, countsExtra, err := coretls.LoadCACertPool(coretls.CAConfig{
		ExtraFilePath: certs.caCertPath,
	})
	if err != nil {
		t.Fatalf("LoadCACertPool with extra failed: %v", err)
	}
	if poolExtra == nil || countsExtra.Extra != 1 {
		t.Errorf("unexpected extra count: %+v", countsExtra)
	}

	// Missing custom CA file path errors
	_, _, err = coretls.LoadCACertPool(coretls.CAConfig{
		CustomFilePath: filepath.Join(certs.dir, "nonexistent-ca.pem"),
	})
	if err == nil {
		t.Errorf("expected error for missing CA file, got nil")
	}

	// Empty custom CA file errors
	emptyCA := filepath.Join(certs.dir, "empty.pem")
	if err := os.WriteFile(emptyCA, []byte("# no cert here\n"), 0o600); err == nil {
		_, _, err = coretls.LoadCACertPool(coretls.CAConfig{
			CustomFilePath: emptyCA,
		})
		if err == nil {
			t.Errorf("expected error for empty CA file, got nil")
		}
	}

	// InsecureSkipVerify skips CA loading entirely
	f := false
	res, err := coretls.Resolve(coretls.BuildOptions{
		TargetURL:                "https://localhost:8443",
		VerifyServerCertificates: &f,
	})
	if err != nil {
		t.Fatalf("Resolve with skip verify failed: %v", err)
	}
	if !res.TLSConfig.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify = true")
	}
	if res.TLSConfig.RootCAs != nil {
		t.Errorf("expected RootCAs to be nil when verification skipped")
	}
}

func TestRelativePathsAndCollectionPath(t *testing.T) {
	certs := generateTestCertificates(t)

	// Make relative cert config
	relCert, err := filepath.Rel(certs.dir, certs.colClientCert)
	if err != nil {
		t.Fatalf("failed to make rel cert: %v", err)
	}
	relKey, err := filepath.Rel(certs.dir, certs.colClientKey)
	if err != nil {
		t.Fatalf("failed to make rel key: %v", err)
	}

	res, err := coretls.Resolve(coretls.BuildOptions{
		TargetURL:      "https://localhost:8443",
		CollectionPath: certs.dir,
		CollectionCerts: &coretls.ClientCertificates{
			Enabled: true,
			Certs: []coretls.ClientCertConfig{
				{
					Domain:       "localhost",
					Type:         coretls.CertTypePEM,
					CertFilePath: relCert,
					KeyFilePath:  relKey,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Resolve with relative paths failed: %v", err)
	}
	if res.CertSource != coretls.CertSourceCollection {
		t.Errorf("expected collection cert source, got: %v", res.CertSource)
	}
	if len(res.TLSConfig.Certificates) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(res.TLSConfig.Certificates))
	}
}

func TestResolveOptions(t *testing.T) {
	opts := coretls.BuildOptions{
		TargetURL:           "https://test.example.com:443/api",
		MinVersion:          tls.VersionTLS13,
		MaxVersion:          tls.VersionTLS13,
		ServerName:          "override.example.com",
		DisableSessionCache: true,
	}

	res, err := coretls.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if res.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %x, want TLS 1.3", res.TLSConfig.MinVersion)
	}
	if res.TLSConfig.MaxVersion != tls.VersionTLS13 {
		t.Errorf("MaxVersion = %x, want TLS 1.3", res.TLSConfig.MaxVersion)
	}
	if res.TLSConfig.ServerName != "override.example.com" {
		t.Errorf("ServerName = %q, want override.example.com", res.TLSConfig.ServerName)
	}
	if res.TLSConfig.ClientSessionCache != nil {
		t.Errorf("expected nil ClientSessionCache when DisableSessionCache is true")
	}
}

func TestFromModelConversions(t *testing.T) {
	ref := model.ClientCertRef{
		Domain:       "*.example.org",
		CertFilePath: "/path/to/cert.pem",
		KeyFilePath:  "/path/to/key.pem",
		Passphrase:   "pw",
	}

	cfg := coretls.FromModelClientCertRef(ref)
	if cfg.Domain != ref.Domain || cfg.CertFilePath != ref.CertFilePath || cfg.KeyFilePath != ref.KeyFilePath || cfg.Passphrase != ref.Passphrase {
		t.Errorf("mismatch in FromModelClientCertRef conversion: %+v", cfg)
	}

	modelCerts := &model.ClientCertificates{
		Enabled:      true,
		Certificates: []model.ClientCertRef{ref},
	}
	tlsCerts := coretls.FromModelClientCertificates(modelCerts)
	if !tlsCerts.Enabled || len(tlsCerts.Certs) != 1 || tlsCerts.Certs[0].Domain != ref.Domain {
		t.Errorf("mismatch in FromModelClientCertificates conversion: %+v", tlsCerts)
	}

	if coretls.FromModelClientCertificates(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
}

func TestErrorHandling(t *testing.T) {
	certs := generateTestCertificates(t)

	// Missing cert file
	_, err := coretls.LoadCertificate(coretls.ClientCertConfig{
		Type:         coretls.CertTypePEM,
		CertFilePath: "/missing/cert.pem",
		KeyFilePath:  "/missing/key.pem",
	}, "")
	if err == nil {
		t.Fatalf("expected error on missing cert file")
	}

	// Missing pfx file
	_, err = coretls.LoadCertificate(coretls.ClientCertConfig{
		Type:        coretls.CertTypePFX,
		PFXFilePath: "/missing/bundle.pfx",
	}, "")
	if err == nil {
		t.Fatalf("expected error on missing pfx file")
	}

	// Invalid PFX passphrase
	_, err = coretls.LoadCertificate(coretls.ClientCertConfig{
		Type:        coretls.CertTypePFX,
		PFXFilePath: certs.clientPfxPath,
		Passphrase:  "wrong-passphrase",
	}, "")
	if err == nil {
		t.Fatalf("expected error for invalid PFX passphrase")
	}

	// Resolution error propagated when cert file missing
	_, err = coretls.Resolve(coretls.BuildOptions{
		TargetURL: "https://localhost:8443",
		CollectionCerts: &coretls.ClientCertificates{
			Enabled: true,
			Certs: []coretls.ClientCertConfig{
				{
					Domain:       "localhost",
					CertFilePath: "/missing/cert.pem",
					KeyFilePath:  "/missing/key.pem",
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error from Resolve when client cert cannot be loaded")
	}
}
