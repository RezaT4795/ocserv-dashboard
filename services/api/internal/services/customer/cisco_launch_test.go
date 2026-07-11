package customer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	apiModels "github.com/mmtaee/ocserv-dashboard/api/internal/models"
	"github.com/mmtaee/ocserv-dashboard/api/internal/repository"
	ciscoSetup "github.com/mmtaee/ocserv-dashboard/api/internal/services/cisco"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/request"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/config"
)

type fakeCiscoLaunchSystemRepository struct {
	repository.SystemRepositoryInterface

	system *apiModels.System
	err    error
}

func (repo *fakeCiscoLaunchSystemRepository) System(
	_ context.Context,
) (*apiModels.System, error) {
	if repo.err != nil {
		return nil, repo.err
	}

	return repo.system, nil
}

func TestLaunchCiscoSetupCertificateImport(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	config.Init(false, "", 0)

	token, err := ciscoSetup.CreateCertificateToken(
		"test-user",
		time.Now().Add(10*time.Minute),
		config.Get().SecretKey,
	)
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	ctl := &Controller{
		request: request.NewCustomRequest(),
	}

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"http://api:8080/api/customers/setup/cisco/launch/certificate/"+token,
		nil,
	)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set(
		"X-Forwarded-Host",
		"panel.example.com:3443",
	)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetPath(
		"/api/customers/setup/cisco/launch/certificate/:token",
	)
	c.SetParamNames("token")
	c.SetParamValues(token)

	if err := ctl.LaunchCiscoSetupCertificateImport(c); err != nil {
		t.Fatalf(
			"LaunchCiscoSetupCertificateImport() error = %v",
			err,
		)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusFound,
			rec.Body.String(),
		)
	}

	location := rec.Header().Get(echo.HeaderLocation)

	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}

	if parsedLocation.Scheme != "anyconnect" {
		t.Fatalf(
			"Location scheme = %q, want %q",
			parsedLocation.Scheme,
			"anyconnect",
		)
	}

	if parsedLocation.Host != "import" {
		t.Fatalf(
			"Location host = %q, want %q",
			parsedLocation.Host,
			"import",
		)
	}

	if parsedLocation.Query().Get("type") != "pkcs12" {
		t.Fatalf(
			"import type = %q, want %q",
			parsedLocation.Query().Get("type"),
			"pkcs12",
		)
	}

	certificateURL := parsedLocation.Query().Get("uri")

	expectedCertificateURL :=
		"https://panel.example.com:3443" +
			"/api/customers/setup/cisco/certificate/" +
			token

	if certificateURL != expectedCertificateURL {
		t.Fatalf(
			"certificate URL = %q, want %q",
			certificateURL,
			expectedCertificateURL,
		)
	}

	if rec.Header().Get(echo.HeaderCacheControl) != "no-store" {
		t.Fatalf(
			"Cache-Control = %q, want %q",
			rec.Header().Get(echo.HeaderCacheControl),
			"no-store",
		)
	}
}

func TestLaunchCiscoSetupConnectionCreate(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	config.Init(false, "", 0)

	token, err := ciscoSetup.CreateCertificateToken(
		"test-user",
		time.Now().Add(10*time.Minute),
		config.Get().SecretKey,
	)
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	ctl := &Controller{
		request: request.NewCustomRequest(),
		systemRepo: &fakeCiscoLaunchSystemRepository{
			system: &apiModels.System{
				ClientProfileConnectionName: "Breeze Plus",
				ClientProfileServerAddress:  "vpn.example.com",
				ClientProfileServerPort:     9443,
			},
		},
	}

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"https://panel.example.com/api/customers/setup/cisco/launch/connection/"+token,
		nil,
	)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetPath(
		"/api/customers/setup/cisco/launch/connection/:token",
	)
	c.SetParamNames("token")
	c.SetParamValues(token)

	if err := ctl.LaunchCiscoSetupConnectionCreate(c); err != nil {
		t.Fatalf(
			"LaunchCiscoSetupConnectionCreate() error = %v",
			err,
		)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusFound,
			rec.Body.String(),
		)
	}

	location := rec.Header().Get(echo.HeaderLocation)

	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}

	if parsedLocation.Scheme != "anyconnect" {
		t.Fatalf(
			"Location scheme = %q, want %q",
			parsedLocation.Scheme,
			"anyconnect",
		)
	}

	if parsedLocation.Host != "create" {
		t.Fatalf(
			"Location host = %q, want %q",
			parsedLocation.Host,
			"create",
		)
	}

	query := parsedLocation.Query()

	if query.Get("name") != "Breeze Plus" {
		t.Fatalf(
			"name = %q, want %q",
			query.Get("name"),
			"Breeze Plus",
		)
	}

	if query.Get("host") != "vpn.example.com:9443" {
		t.Fatalf(
			"host = %q, want %q",
			query.Get("host"),
			"vpn.example.com:9443",
		)
	}

	if query.Get("certcommonname") != "test-user" {
		t.Fatalf(
			"certcommonname = %q, want %q",
			query.Get("certcommonname"),
			"test-user",
		)
	}

	if query.Get("usecert") != "true" {
		t.Fatalf(
			"usecert = %q, want %q",
			query.Get("usecert"),
			"true",
		)
	}
}

func TestLaunchCiscoSetupRejectsExpiredToken(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	config.Init(false, "", 0)

	token, err := ciscoSetup.CreateCertificateToken(
		"test-user",
		time.Now().Add(-time.Second),
		config.Get().SecretKey,
	)
	if err != nil {
		t.Fatalf("CreateCertificateToken() error = %v", err)
	}

	ctl := &Controller{
		request: request.NewCustomRequest(),
	}

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"https://panel.example.com/api/customers/setup/cisco/launch/certificate/"+token,
		nil,
	)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetPath(
		"/api/customers/setup/cisco/launch/certificate/:token",
	)
	c.SetParamNames("token")
	c.SetParamValues(token)

	if err := ctl.LaunchCiscoSetupCertificateImport(c); err != nil {
		t.Fatalf(
			"LaunchCiscoSetupCertificateImport() error = %v",
			err,
		)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusBadRequest,
			rec.Body.String(),
		)
	}
}
