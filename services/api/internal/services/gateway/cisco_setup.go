package gateway

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	ciscoSetup "github.com/mmtaee/ocserv-dashboard/api/internal/services/cisco"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/request"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/config"
	"gorm.io/gorm"
)

// CiscoSetup creates Cisco Secure Client setup data for an external gateway.
//
// @Summary      Gateway Cisco Secure Client setup
// @Description  Creates Cisco Secure Client certificate import and connection creation URIs for an existing ocserv user.
// @Tags         Gateway
// @Produce      json
// @Param        Authorization header string true "Bearer GATEWAY_API_TOKEN"
// @Param        username path string true "Ocserv username"
// @Failure      400 {object} request.ErrorResponse
// @Failure      401 {object} middlewares.Unauthorized
// @Failure      404 {object} request.ErrorResponse
// @Success      200 {object} CiscoSetupResponse
// @Router       /gateway/users/{username}/cisco-setup [post]
func (ctl *Controller) CiscoSetup(c echo.Context) error {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		return ctl.request.BadRequest(c, errors.New("username is required"))
	}

	ctx := c.Request().Context()

	user, err := ctl.ocservUserRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, request.ErrorResponse{
				Error:   []string{"user not found"},
				Message: []string{},
			})
		}

		return ctl.request.BadRequest(c, err)
	}

	systemConfig, err := ctl.systemRepo.System(ctx)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	setup, err := ciscoSetup.BuildSetup(ciscoSetup.SetupInput{
		Username:            user.Username,
		CertificatePassword: user.Password,
		ConnectionName:      systemConfig.ClientProfileConnectionName,
		ServerAddress:       systemConfig.ClientProfileServerAddress,
		ServerPort:          systemConfig.ClientProfileServerPort,
		PublicAPIBaseURL:    ciscoSetup.PublicAPIBaseURL(c.Request()),
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
