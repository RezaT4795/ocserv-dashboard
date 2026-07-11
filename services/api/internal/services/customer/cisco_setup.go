package customer

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	ciscoSetup "github.com/mmtaee/ocserv-dashboard/api/internal/services/cisco"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/config"
)

// CiscoSetup creates Cisco Secure Client setup URIs for the customer.
//
// @Summary      Create customer Cisco Secure Client setup links
// @Description  Create Cisco Secure Client certificate import and connection creation URIs using ocserv username/password
// @Tags         Customers
// @Accept       json
// @Produce      json
// @Param        request body SummaryData true "customer username and password (same ocserv account)."
// @Failure      400 {object} request.ErrorResponse
// @Failure      429 {object} middlewares.TooManyRequests
// @Success      200 {object} CiscoSetupResponse
// @Router       /customers/setup/cisco [post]
func (ctl *Controller) CiscoSetup(c echo.Context) error {
	var data SummaryData

	if err := ctl.request.DoValidate(c, &data); err != nil {
		return ctl.request.BadRequest(c, err)
	}

	if data.Password == "Secret-Ocpasswd" {
		return ctl.request.BadRequest(c, errors.New("invalid username or password"))
	}

	user, err := ctl.ocservUserRepo.GetByUsername(
		c.Request().Context(),
		data.Username,
	)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	if user.Password != data.Password {
		return ctl.request.BadRequest(c, errors.New("invalid username or password"))
	}

	systemConfig, err := ctl.systemRepo.System(c.Request().Context())
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	setup, err := ciscoSetup.BuildSetup(ciscoSetup.SetupInput{
		Username:            user.Username,
		CertificatePassword: user.Password,
		ConnectionName:      systemConfig.ClientProfileConnectionName,
		ServerAddress:       systemConfig.ClientProfileServerAddress,
		ServerPort:          systemConfig.ClientProfileServerPort,
		PublicAPIBaseURL:    publicAPIBaseURL(c),
		SecretKey:           config.Get().SecretKey,
		Now:                 time.Now(),
	})
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	return c.JSON(http.StatusOK, CiscoSetupResponse{
		CertificateImportURI: setup.CertificateImportURI,
		ConnectionCreateURI:  setup.ConnectionCreateURI,
		CertificatePassword:  setup.CertificatePassword,
		ConnectionName:       setup.ConnectionName,
		ServerAddress:        setup.ServerAddress,
		ServerPort:           setup.ServerPort,
		ExpiresAt:            setup.ExpiresAt,
	})
}

// DownloadCiscoSetupCertificate downloads the customer's certificate through a short-lived signed setup token.
//
// @Summary      Download customer Cisco Secure Client setup certificate
// @Description  Download customer's PKCS#12 certificate using a short-lived Cisco Secure Client setup token
// @Tags         Customers
// @Produce      application/x-pkcs12
// @Param        token path string true "Cisco Secure Client setup certificate token"
// @Failure      400 {object} request.ErrorResponse
// @Failure      429 {object} middlewares.TooManyRequests
// @Success      200 {file} file "user.p12"
// @Router       /customers/setup/cisco/certificate/{token} [get]
func (ctl *Controller) DownloadCiscoSetupCertificate(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return ctl.request.BadRequest(c, errors.New("token is required"))
	}

	username, err := ciscoSetup.ParseCertificateToken(
		token,
		time.Now(),
		config.Get().SecretKey,
	)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	ctx := c.Request().Context()

	user, err := ctl.ocservUserRepo.GetByUsername(ctx, username)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	path, err := ctl.ocservUserRepo.CertificatePathByUsername(
		ctx,
		user.Username,
	)
	if err != nil {
		if err := ctl.ocservUserRepo.CreateCertificate(
			ctx,
			user.UID,
		); err != nil {
			return ctl.request.BadRequest(c, err)
		}

		path, err = ctl.ocservUserRepo.CertificatePathByUsername(
			ctx,
			user.Username,
		)
		if err != nil {
			return ctl.request.BadRequest(c, err)
		}
	}

	c.Response().Header().Set(
		echo.HeaderContentType,
		"application/x-pkcs12",
	)
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")

	return c.Attachment(path, user.Username+".p12")
}

func publicAPIBaseURL(c echo.Context) string {
	req := c.Request()

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
