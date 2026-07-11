package cisco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func CreateCertificateToken(username string, expiresAt time.Time, secretKey string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", errors.New("username is required")
	}

	if strings.Contains(username, "|") {
		return "", errors.New("username contains invalid characters")
	}

	payload := username + "|" + strconv.FormatInt(expiresAt.Unix(), 10)

	signature, err := signCertificatePayload(payload, secretKey)
	if err != nil {
		return "", err
	}

	rawToken := payload + "|" + signature

	return base64.RawURLEncoding.EncodeToString([]byte(rawToken)), nil
}

func ParseCertificateToken(token string, now time.Time, secretKey string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("token is required")
	}

	rawToken, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", errors.New("invalid token")
	}

	parts := strings.Split(string(rawToken), "|")
	if len(parts) != 3 {
		return "", errors.New("invalid token")
	}

	username := parts[0]
	expiresAtUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", errors.New("invalid token expiry")
	}

	if now.After(time.Unix(expiresAtUnix, 0)) {
		return "", errors.New("token has expired")
	}

	payload := username + "|" + parts[1]

	expectedSignature, err := signCertificatePayload(payload, secretKey)
	if err != nil {
		return "", err
	}

	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return "", errors.New("invalid token signature")
	}

	return username, nil
}

func signCertificatePayload(payload string, secretKey string) (string, error) {
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		return "", errors.New("secret key is not configured")
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
