package cisco

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestCreateAndParseCertificateToken(t *testing.T) {
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	token, err := CreateCertificateToken("test-user", expiresAt, "test-secret")
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	username, err := ParseCertificateToken(token, now, "test-secret")
	if err != nil {
		t.Fatalf("ParseCertificateToken() error = %v", err)
	}

	if username != "test-user" {
		t.Fatalf(
			"ParseCertificateToken() username = %q, want %q",
			username,
			"test-user",
		)
	}
}

func TestParseCertificateTokenRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)

	token, err := CreateCertificateToken(
		"test-user",
		now.Add(-time.Second),
		"test-secret",
	)
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	_, err = ParseCertificateToken(token, now, "test-secret")
	if err == nil || err.Error() != "token has expired" {
		t.Fatalf(
			"ParseCertificateToken() error = %v, want %q",
			err,
			"token has expired",
		)
	}
}

func TestParseCertificateTokenRejectsTamperedToken(t *testing.T) {
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)

	token, err := CreateCertificateToken(
		"test-user",
		now.Add(10*time.Minute),
		"test-secret",
	)
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	rawToken, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	tamperedRawToken := strings.Replace(
		string(rawToken),
		"test-user|",
		"other-user|",
		1,
	)
	tamperedToken := base64.RawURLEncoding.EncodeToString(
		[]byte(tamperedRawToken),
	)

	_, err = ParseCertificateToken(tamperedToken, now, "test-secret")
	if err == nil || err.Error() != "invalid token signature" {
		t.Fatalf(
			"ParseCertificateToken() error = %v, want %q",
			err,
			"invalid token signature",
		)
	}
}

func TestCreateCertificateTokenValidatesInput(t *testing.T) {
	expiresAt := time.Date(2026, time.July, 11, 12, 10, 0, 0, time.UTC)

	tests := []struct {
		name      string
		username  string
		secretKey string
		wantError string
	}{
		{
			name:      "empty username",
			username:  "",
			secretKey: "test-secret",
			wantError: "username is required",
		},
		{
			name:      "invalid username character",
			username:  "invalid|user",
			secretKey: "test-secret",
			wantError: "username contains invalid characters",
		},
		{
			name:      "empty secret key",
			username:  "test-user",
			secretKey: "",
			wantError: "secret key is not configured",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CreateCertificateToken(
				test.username,
				expiresAt,
				test.secretKey,
			)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf(
					"CreateCertificateToken() error = %v, want %q",
					err,
					test.wantError,
				)
			}
		})
	}
}
