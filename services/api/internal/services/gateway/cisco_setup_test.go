package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	apiModels "github.com/mmtaee/ocserv-dashboard/api/internal/models"
	"github.com/mmtaee/ocserv-dashboard/api/internal/repository"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/request"
	commonModels "github.com/mmtaee/ocserv-dashboard/common/models"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/config"
	"gorm.io/gorm"
)

type fakeCiscoSetupOcservUserRepository struct {
	repository.OcservUserRepositoryInterface

	user *commonModels.OcservUser
	err  error
}

func (repo *fakeCiscoSetupOcservUserRepository) GetByUsername(
	_ context.Context,
	username string,
) (*commonModels.OcservUser, error) {
	if repo.err != nil {
		return nil, repo.err
	}

	if repo.user == nil || repo.user.Username != username {
		return nil, gorm.ErrRecordNotFound
	}

	return repo.user, nil
}

type fakeCiscoSetupSystemRepository struct {
	repository.SystemRepositoryInterface

	system *apiModels.System
	err    error
}

func (repo *fakeCiscoSetupSystemRepository) System(
	_ context.Context,
) (*apiModels.System, error) {
	if repo.err != nil {
		return nil, repo.err
	}

	return repo.system, nil
}

func TestCiscoSetup(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	config.Init(false, "", 0)

	ctl := &Controller{
		request: request.NewCustomRequest(),
		ocservUserRepo: &fakeCiscoSetupOcservUserRepository{
			user: &commonModels.OcservUser{
				Username: "test-user",
				Password: "test-password",
			},
		},
		systemRepo: &fakeCiscoSetupSystemRepository{
			system: &apiModels.System{
				ClientProfileConnectionName: "Breeze Plus",
				ClientProfileServerAddress:  "vpn.example.com",
				ClientProfileServerPort:     9443,
			},
		},
	}

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodPost,
		"http://api:8080/api/gateway/users/test-user/cisco-setup",
		nil,
	)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "panel.example.com:3443")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetPath("/api/gateway/users/:username/cisco-setup")
	c.SetParamNames("username")
	c.SetParamValues("test-user")

	if err := ctl.CiscoSetup(c); err != nil {
		t.Fatalf("CiscoSetup() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"CiscoSetup() status = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var response CiscoSetupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.CertificatePassword != "test-password" {
		t.Fatalf(
			"CertificatePassword = %q, want %q",
			response.CertificatePassword,
			"test-password",
		)
	}

	if response.ConnectionName != "Breeze Plus" {
		t.Fatalf(
			"ConnectionName = %q, want %q",
			response.ConnectionName,
			"Breeze Plus",
		)
	}

	if response.ServerAddress != "vpn.example.com" {
		t.Fatalf(
			"ServerAddress = %q, want %q",
			response.ServerAddress,
			"vpn.example.com",
		)
	}

	if response.ServerPort != 9443 {
		t.Fatalf(
			"ServerPort = %d, want %d",
			response.ServerPort,
			9443,
		)
	}

	if !strings.Contains(
		response.CertificateImportURI,
		"panel.example.com%3A3443",
	) {
		t.Fatalf(
			"CertificateImportURI = %q, expected public panel host",
			response.CertificateImportURI,
		)
	}

	if !strings.Contains(
		response.ConnectionCreateURI,
		"certcommonname=test-user",
	) {
		t.Fatalf(
			"ConnectionCreateURI = %q, expected certificate common name",
			response.ConnectionCreateURI,
		)
	}

	if response.ExpiresAt.IsZero() {
		t.Fatal("ExpiresAt must not be zero")
	}
}

func TestCiscoSetupReturnsNotFound(t *testing.T) {
	ctl := &Controller{
		request: request.NewCustomRequest(),
		ocservUserRepo: &fakeCiscoSetupOcservUserRepository{
			err: gorm.ErrRecordNotFound,
		},
		systemRepo: &fakeCiscoSetupSystemRepository{},
	}

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodPost,
		"https://panel.example.com/api/gateway/users/missing-user/cisco-setup",
		nil,
	)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.SetPath("/api/gateway/users/:username/cisco-setup")
	c.SetParamNames("username")
	c.SetParamValues("missing-user")

	if err := ctl.CiscoSetup(c); err != nil {
		t.Fatalf("CiscoSetup() error = %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"CiscoSetup() status = %d, want %d; body = %s",
			rec.Code,
			http.StatusNotFound,
			rec.Body.String(),
		)
	}

	var response request.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(response.Error) != 1 || response.Error[0] != "user not found" {
		t.Fatalf(
			"Error = %#v, want %#v",
			response.Error,
			[]string{"user not found"},
		)
	}
}
