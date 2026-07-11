package cisco

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicAPIBaseURLUsesForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"http://api:8080/api/customers/setup/cisco",
		nil,
	)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "panel.example.com:3443")

	got := PublicAPIBaseURL(req)
	want := "https://panel.example.com:3443"

	if got != want {
		t.Fatalf(
			"PublicAPIBaseURL() = %q, want %q",
			got,
			want,
		)
	}
}

func TestPublicAPIBaseURLUsesHTTPSRequest(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"https://panel.example.com:3443/api/customers/setup/cisco",
		nil,
	)

	got := PublicAPIBaseURL(req)
	want := "https://panel.example.com:3443"

	if got != want {
		t.Fatalf(
			"PublicAPIBaseURL() = %q, want %q",
			got,
			want,
		)
	}
}

func TestPublicAPIBaseURLFallsBackToHTTP(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"http://panel.example.com:3443/api/customers/setup/cisco",
		nil,
	)

	got := PublicAPIBaseURL(req)
	want := "http://panel.example.com:3443"

	if got != want {
		t.Fatalf(
			"PublicAPIBaseURL() = %q, want %q",
			got,
			want,
		)
	}
}
