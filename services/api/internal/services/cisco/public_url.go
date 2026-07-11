package cisco

import (
	"net/http"
	"strings"
)

func PublicAPIBaseURL(req *http.Request) string {
	scheme := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = strings.TrimSpace(req.URL.Scheme)
	}
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(req.Host)
	}

	return scheme + "://" + host
}
