package cisco

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBuildSetup(t *testing.T) {
	now := time.Date(2026, time.July, 11, 19, 0, 0, 0, time.UTC)

	setup, err := BuildSetup(SetupInput{
		Username:            "test-user",
		CertificatePassword: "test-password",
		ConnectionName:      "Breeze Plus",
		ServerAddress:       "vpn.example.com",
		ServerPort:          9443,
		PublicAPIBaseURL:    "https://panel.example.com:3443/",
		SecretKey:           "test-secret",
		Now:                 now,
	})
	if err != nil {
		t.Fatalf("BuildSetup() error = %v", err)
	}

	if setup.CertificatePassword != "test-password" {
		t.Fatalf(
			"BuildSetup() CertificatePassword = %q, want %q",
			setup.CertificatePassword,
			"test-password",
		)
	}

	if setup.ConnectionName != "Breeze Plus" {
		t.Fatalf(
			"BuildSetup() ConnectionName = %q, want %q",
			setup.ConnectionName,
			"Breeze Plus",
		)
	}

	if setup.ServerAddress != "vpn.example.com" {
		t.Fatalf(
			"BuildSetup() ServerAddress = %q, want %q",
			setup.ServerAddress,
			"vpn.example.com",
		)
	}

	if setup.ServerPort != 9443 {
		t.Fatalf(
			"BuildSetup() ServerPort = %d, want %d",
			setup.ServerPort,
			9443,
		)
	}

	expectedExpiresAt := now.Add(CertificateTokenTTL)
	if !setup.ExpiresAt.Equal(expectedExpiresAt) {
		t.Fatalf(
			"BuildSetup() ExpiresAt = %v, want %v",
			setup.ExpiresAt,
			expectedExpiresAt,
		)
	}

	certificateImportURLPrefix :=
		"https://panel.example.com:3443" +
			certificateImportLaunchPath

	if !strings.HasPrefix(
		setup.CertificateImportURL,
		certificateImportURLPrefix,
	) {
		t.Fatalf(
			"BuildSetup() CertificateImportURL = %q, expected prefix %q",
			setup.CertificateImportURL,
			certificateImportURLPrefix,
		)
	}

	connectionCreateURLPrefix :=
		"https://panel.example.com:3443" +
			connectionCreateLaunchPath

	if !strings.HasPrefix(
		setup.ConnectionCreateURL,
		connectionCreateURLPrefix,
	) {
		t.Fatalf(
			"BuildSetup() ConnectionCreateURL = %q, expected prefix %q",
			setup.ConnectionCreateURL,
			connectionCreateURLPrefix,
		)
	}

	certificateLaunchToken := strings.TrimPrefix(
		setup.CertificateImportURL,
		certificateImportURLPrefix,
	)
	connectionLaunchToken := strings.TrimPrefix(
		setup.ConnectionCreateURL,
		connectionCreateURLPrefix,
	)

	if certificateLaunchToken == "" {
		t.Fatal("CertificateImportURL token must not be empty")
	}

	if certificateLaunchToken != connectionLaunchToken {
		t.Fatalf(
			"launcher tokens differ: certificate = %q, connection = %q",
			certificateLaunchToken,
			connectionLaunchToken,
		)
	}

	username, err := ParseCertificateToken(
		certificateLaunchToken,
		now,
		"test-secret",
	)
	if err != nil {
		t.Fatalf("ParseCertificateToken() error = %v", err)
	}

	if username != "test-user" {
		t.Fatalf(
			"launcher token username = %q, want %q",
			username,
			"test-user",
		)
	}

	assertCertificateImportURI(t, setup.CertificateImportURI, now)
	assertConnectionCreateURI(t, setup.ConnectionCreateURI)

	assertCertificateImportURI(t, setup.CertificateImportURI, now)
	assertConnectionCreateURI(t, setup.ConnectionCreateURI)
}

func TestBuildSetupValidatesProfileConfiguration(t *testing.T) {
	now := time.Date(2026, time.July, 11, 19, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		change    func(*SetupInput)
		wantError string
	}{
		{
			name: "missing connection name",
			change: func(input *SetupInput) {
				input.ConnectionName = ""
			},
			wantError: "client profile connection name is required",
		},
		{
			name: "invalid server address",
			change: func(input *SetupInput) {
				input.ServerAddress = "https://vpn.example.com"
			},
			wantError: "client profile server address must not include a URL scheme",
		},
		{
			name: "invalid server port",
			change: func(input *SetupInput) {
				input.ServerPort = 0
			},
			wantError: "client profile server port must be between 1 and 65535",
		},
		{
			name: "non HTTPS public API URL",
			change: func(input *SetupInput) {
				input.PublicAPIBaseURL = "http://panel.example.com:3443"
			},
			wantError: "certificate URL must use HTTPS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := SetupInput{
				Username:            "test-user",
				CertificatePassword: "test-password",
				ConnectionName:      "Breeze Plus",
				ServerAddress:       "vpn.example.com",
				ServerPort:          9443,
				PublicAPIBaseURL:    "https://panel.example.com:3443",
				SecretKey:           "test-secret",
				Now:                 now,
			}

			test.change(&input)

			_, err := BuildSetup(input)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf(
					"BuildSetup() error = %v, want %q",
					err,
					test.wantError,
				)
			}
		})
	}
}

func assertCertificateImportURI(
	t *testing.T,
	importURI string,
	now time.Time,
) {
	t.Helper()

	parsedImportURI, err := url.Parse(importURI)
	if err != nil {
		t.Fatalf("url.Parse(certificate import URI) error = %v", err)
	}

	if parsedImportURI.Scheme != "anyconnect" {
		t.Fatalf(
			"certificate import URI scheme = %q, want %q",
			parsedImportURI.Scheme,
			"anyconnect",
		)
	}

	if parsedImportURI.Host != "import" {
		t.Fatalf(
			"certificate import URI host = %q, want %q",
			parsedImportURI.Host,
			"import",
		)
	}

	if parsedImportURI.Query().Get("type") != "pkcs12" {
		t.Fatalf(
			"certificate import type = %q, want %q",
			parsedImportURI.Query().Get("type"),
			"pkcs12",
		)
	}

	certificateURL := parsedImportURI.Query().Get("uri")

	parsedCertificateURL, err := url.Parse(certificateURL)
	if err != nil {
		t.Fatalf("url.Parse(certificate URL) error = %v", err)
	}

	if parsedCertificateURL.Scheme != "https" {
		t.Fatalf(
			"certificate URL scheme = %q, want %q",
			parsedCertificateURL.Scheme,
			"https",
		)
	}

	if parsedCertificateURL.Host != "panel.example.com:3443" {
		t.Fatalf(
			"certificate URL host = %q, want %q",
			parsedCertificateURL.Host,
			"panel.example.com:3443",
		)
	}

	token := strings.TrimPrefix(
		parsedCertificateURL.Path,
		certificateDownloadPath,
	)

	if token == parsedCertificateURL.Path {
		t.Fatalf(
			"certificate URL path %q does not start with %q",
			parsedCertificateURL.Path,
			certificateDownloadPath,
		)
	}

	username, err := ParseCertificateToken(
		token,
		now,
		"test-secret",
	)
	if err != nil {
		t.Fatalf("ParseCertificateToken() error = %v", err)
	}

	if username != "test-user" {
		t.Fatalf(
			"certificate token username = %q, want %q",
			username,
			"test-user",
		)
	}
}

func assertConnectionCreateURI(t *testing.T, createURI string) {
	t.Helper()

	parsedCreateURI, err := url.Parse(createURI)
	if err != nil {
		t.Fatalf("url.Parse(connection create URI) error = %v", err)
	}

	if parsedCreateURI.Scheme != "anyconnect" {
		t.Fatalf(
			"connection create URI scheme = %q, want %q",
			parsedCreateURI.Scheme,
			"anyconnect",
		)
	}

	if parsedCreateURI.Host != "create" {
		t.Fatalf(
			"connection create URI host = %q, want %q",
			parsedCreateURI.Host,
			"create",
		)
	}

	query := parsedCreateURI.Query()

	if query.Get("name") != "Breeze Plus" {
		t.Fatalf(
			"connection name = %q, want %q",
			query.Get("name"),
			"Breeze Plus",
		)
	}

	if query.Get("host") != "vpn.example.com:9443" {
		t.Fatalf(
			"connection host = %q, want %q",
			query.Get("host"),
			"vpn.example.com:9443",
		)
	}

	if query.Get("usecert") != "true" {
		t.Fatalf(
			"usecert = %q, want %q",
			query.Get("usecert"),
			"true",
		)
	}

	if query.Get("certcommonname") != "test-user" {
		t.Fatalf(
			"certcommonname = %q, want %q",
			query.Get("certcommonname"),
			"test-user",
		)
	}
}
