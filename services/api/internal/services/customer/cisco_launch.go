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

// LaunchCiscoSetupCertificateImport redirects a short-lived HTTPS setup link to the Cisco certificate import URI.
//
// @Summary      Launch Cisco certificate import
// @Description  Redirect a short-lived signed HTTPS setup link to the Cisco Secure Client certificate import URI
// @Tags         Customers
// @Produce      text/html
// @Param        token path string true "Cisco Secure Client setup token"
// @Failure      400 {object} request.ErrorResponse
// @Failure      429 {object} middlewares.TooManyRequests
// @Success      302
// @Router       /customers/setup/cisco/launch/certificate/{token} [get]
func (ctl *Controller) LaunchCiscoSetupCertificateImport(
	c echo.Context,
) error {
	token, _, err := parseCiscoSetupLaunchToken(c)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	targetURI, err := ciscoSetup.BuildCertificateImportURI(
		ciscoSetup.PublicAPIBaseURL(c.Request()),
		token,
	)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	return redirectToCiscoSetupURI(c, targetURI)
}

// LaunchCiscoSetupConnectionCreate redirects a short-lived HTTPS setup link to the Cisco connection creation URI.
//
// @Summary      Launch Cisco connection creation
// @Description  Redirect a short-lived signed HTTPS setup link to the Cisco Secure Client connection creation URI
// @Tags         Customers
// @Produce      text/html
// @Param        token path string true "Cisco Secure Client setup token"
// @Failure      400 {object} request.ErrorResponse
// @Failure      429 {object} middlewares.TooManyRequests
// @Success      302
// @Router       /customers/setup/cisco/launch/connection/{token} [get]
func (ctl *Controller) LaunchCiscoSetupConnectionCreate(
	c echo.Context,
) error {
	_, username, err := parseCiscoSetupLaunchToken(c)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	systemConfig, err := ctl.systemRepo.System(
		c.Request().Context(),
	)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	targetURI, err := ciscoSetup.BuildConnectionCreateURI(
		systemConfig.ClientProfileConnectionName,
		systemConfig.ClientProfileServerAddress,
		systemConfig.ClientProfileServerPort,
		username,
	)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	return redirectToCiscoSetupURI(c, targetURI)
}

func parseCiscoSetupLaunchToken(
	c echo.Context,
) (string, string, error) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		return "", "", errors.New("token is required")
	}

	username, err := ciscoSetup.ParseCertificateToken(
		token,
		time.Now(),
		config.Get().SecretKey,
	)
	if err != nil {
		return "", "", err
	}

	return token, username, nil
}

func redirectToCiscoSetupURI(
	c echo.Context,
	targetURI string,
) error {
	c.Response().Header().Set(
		echo.HeaderCacheControl,
		"no-store",
	)
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set(
		"Referrer-Policy",
		"no-referrer",
	)
	c.Response().Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)

	return c.Redirect(http.StatusFound, targetURI)
}
