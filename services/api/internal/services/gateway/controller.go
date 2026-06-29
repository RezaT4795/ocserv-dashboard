package gateway

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mmtaee/ocserv-dashboard/api/internal/repository"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/request"
	"github.com/mmtaee/ocserv-dashboard/common/models"
)

const bytesInGiB int64 = 1024 * 1024 * 1024

type Controller struct {
	request        request.CustomRequestInterface
	ocservUserRepo repository.OcservUserRepositoryInterface
}

func New() *Controller {
	return &Controller{
		request:        request.NewCustomRequest(),
		ocservUserRepo: repository.NewtOcservUserRepository(),
	}
}

// CreateUser creates an ocserv user from an external gateway.
//
// @Summary      Gateway ocserv user creation
// @Description  Creates a local ocserv user for an authenticated external gateway.
// @Tags         Gateway
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer GATEWAY_API_TOKEN"
// @Param        request body CreateUserData true "gateway user create data"
// @Failure      400 {object} request.ErrorResponse
// @Failure      401 {object} middlewares.Unauthorized
// @Success      201 {object} CreateUserResponse
// @Router       /gateway/users [post]
func (ctl *Controller) CreateUser(c echo.Context) error {
	var data CreateUserData
	if err := ctl.request.DoValidate(c, &data); err != nil {
		return ctl.request.BadRequest(c, err)
	}

	var expireAt *time.Time
	if !data.Unlimited {
		if data.ExpireAt == nil || strings.TrimSpace(*data.ExpireAt) == "" {
			return ctl.request.BadRequest(c, errors.New("expire_at is required unless unlimited is true"))
		}

		parsedExpireAt, err := time.Parse("2006-01-02", strings.TrimSpace(*data.ExpireAt))
		if err != nil {
			return ctl.request.BadRequest(c, errors.New("expire_at must use YYYY-MM-DD format"))
		}

		expireAt = &parsedExpireAt
	}

	trafficLimitGB := data.TrafficLimitGB
	trafficSize := int64(trafficLimitGB) * bytesInGiB
	if data.TrafficType == models.Free {
		trafficLimitGB = 0
		trafficSize = 0
	}

	owner := strings.TrimSpace(os.Getenv("GATEWAY_API_OWNER"))
	if owner == "" {
		owner = "gateway"
	}

	description := strings.TrimSpace(data.Description)
	if description == "" {
		description = "created via external gateway"
	}

	user := &models.OcservUser{
		Owner:       owner,
		Username:    strings.TrimSpace(data.Username),
		Password:    data.Password,
		Group:       strings.TrimSpace(data.Group),
		ExpireAt:    expireAt,
		TrafficType: data.TrafficType,
		TrafficSize: trafficSize,
		Description: description,
	}

	created, err := ctl.ocservUserRepo.Create(c.Request().Context(), user)
	if err != nil {
		return ctl.request.BadRequest(c, fmt.Errorf("failed to create ocserv user: %w", err))
	}

	var expireAtResponse *string
	if created.ExpireAt != nil {
		formatted := created.ExpireAt.Format("2006-01-02")
		expireAtResponse = &formatted
	}

	return c.JSON(http.StatusCreated, CreateUserResponse{
		RemoteUserID:   created.UID,
		Username:       created.Username,
		Password:       created.Password,
		Group:          created.Group,
		Unlimited:      created.ExpireAt == nil,
		ExpireAt:       expireAtResponse,
		TrafficType:    created.TrafficType,
		TrafficLimitGB: trafficLimitGB,
	})
}
