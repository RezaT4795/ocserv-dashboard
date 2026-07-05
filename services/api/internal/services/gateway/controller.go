package gateway

import (
	"context"
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
	"gorm.io/gorm"
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

	expireAtResponse := formatDate(created.ExpireAt)

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

// UserStatus returns live ocserv user status and traffic usage for an external gateway.
//
// @Summary      Gateway ocserv user status
// @Description  Returns live status, expiry, traffic limit, consumed traffic, and remaining traffic for a gateway-created user.
// @Tags         Gateway
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer GATEWAY_API_TOKEN"
// @Param        username path string true "Ocserv username"
// @Failure      400 {object} request.ErrorResponse
// @Failure      401 {object} middlewares.Unauthorized
// @Failure      404 {object} request.ErrorResponse
// @Success      200 {object} UserStatusResponse
// @Router       /gateway/users/{username}/status [get]
func (ctl *Controller) UserStatus(c echo.Context) error {
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

	response, err := ctl.buildUserStatusResponse(ctx, user)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateUserSubscription updates traffic, expiry, activation state, and group for an existing gateway user.
//
// @Summary      Gateway ocserv user subscription update
// @Description  Updates traffic limit, expiry date, traffic usage reset, activation state, and group for a gateway-created user.
// @Tags         Gateway
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer GATEWAY_API_TOKEN"
// @Param        username path string true "Ocserv username"
// @Param        request body UpdateUserSubscriptionData true "gateway user subscription update data"
// @Failure      400 {object} request.ErrorResponse
// @Failure      401 {object} middlewares.Unauthorized
// @Failure      404 {object} request.ErrorResponse
// @Success      200 {object} UserStatusResponse
// @Router       /gateway/users/{username}/subscription [patch]
func (ctl *Controller) UpdateUserSubscription(c echo.Context) error {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		return ctl.request.BadRequest(c, errors.New("username is required"))
	}

	var data UpdateUserSubscriptionData
	if err := ctl.request.DoValidate(c, &data); err != nil {
		return ctl.request.BadRequest(c, err)
	}

	update := repository.GatewaySubscriptionUpdate{
		ResetTraffic: data.ResetTrafficUsage,
		Activate:     data.Activate,
	}

	if data.Group != nil {
		group := strings.TrimSpace(*data.Group)
		if group == "" {
			return ctl.request.BadRequest(c, errors.New("group cannot be empty"))
		}

		update.Group = &group
	}

	if data.TrafficLimitGB != nil {
		trafficSize := int64(*data.TrafficLimitGB) * bytesInGiB
		update.TrafficSize = &trafficSize
	}

	if data.Unlimited {
		update.SetExpireAt = true
		update.ExpireAt = nil
	} else if data.ExpireAt != nil {
		expireAtText := strings.TrimSpace(*data.ExpireAt)
		if expireAtText == "" {
			return ctl.request.BadRequest(c, errors.New("expire_at cannot be empty"))
		}

		expireAt, err := time.Parse("2006-01-02", expireAtText)
		if err != nil {
			return ctl.request.BadRequest(c, errors.New("expire_at must use YYYY-MM-DD format"))
		}

		update.SetExpireAt = true
		update.ExpireAt = &expireAt
	}

	if update.TrafficSize == nil &&
		!update.SetExpireAt &&
		!update.ResetTraffic &&
		!update.Activate &&
		update.Group == nil {
		return ctl.request.BadRequest(c, errors.New("at least one subscription field is required"))
	}

	ctx := c.Request().Context()

	user, err := ctl.ocservUserRepo.UpdateGatewaySubscription(
		ctx,
		username,
		update)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, request.ErrorResponse{
				Error:   []string{"user not found"},
				Message: []string{},
			})
		}

		return ctl.request.BadRequest(c, err)
	}

	response, err := ctl.buildUserStatusResponse(ctx, user)
	if err != nil {
		return ctl.request.BadRequest(c, err)
	}

	return c.JSON(http.StatusOK, response)
}

// DeleteUser deletes an existing gateway-created ocserv user.
//
// @Summary      Gateway ocserv user deletion
// @Description  Deletes an existing ocserv user for an authenticated external gateway.
// @Tags         Gateway
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer GATEWAY_API_TOKEN"
// @Param        username path string true "Ocserv username"
// @Failure      400 {object} request.ErrorResponse
// @Failure      401 {object} middlewares.Unauthorized
// @Failure      404 {object} request.ErrorResponse
// @Success      200 {object} DeleteUserResponse
// @Router       /gateway/users/{username} [delete]
func (ctl *Controller) DeleteUser(c echo.Context) error {
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

	_, err = ctl.ocservUserRepo.Delete(ctx, user.UID)
	if err != nil {
		return ctl.request.BadRequest(c, fmt.Errorf("failed to delete ocserv user: %w", err))
	}

	return c.JSON(http.StatusOK, DeleteUserResponse{
		RemoteUserID: user.UID,
		Username:     user.Username,
		Deleted:      true,
	})
}

func (ctl *Controller) buildUserStatusResponse(
	ctx context.Context,
	user *models.OcservUser,
) (UserStatusResponse, error) {
	rxBytes := int64(user.Rx)
	txBytes := int64(user.Tx)

	if isMonthlyTrafficType(user.TrafficType) {
		currentCycleTraffic, err := ctl.ocservUserRepo.CurrentCycleTraffic(
			ctx,
			user.ID,
			user.UsageResetAt,
		)
		if err != nil {
			return UserStatusResponse{}, err
		}

		rxBytes = currentCycleTraffic.RX
		txBytes = currentCycleTraffic.TX
	}

	consumedBytes := consumedTrafficBytes(user.TrafficType, rxBytes, txBytes)

	remainingBytes := user.TrafficSize - consumedBytes
	if remainingBytes < 0 {
		remainingBytes = 0
	}

	return UserStatusResponse{
		RemoteUserID:          user.UID,
		Username:              user.Username,
		Group:                 user.Group,
		Active:                user.DeactivatedAt == nil && !user.IsLocked,
		Locked:                user.IsLocked,
		Deactivated:           user.DeactivatedAt != nil,
		Unlimited:             user.ExpireAt == nil,
		ExpireAt:              formatDate(user.ExpireAt),
		DeactivatedAt:         formatDate(user.DeactivatedAt),
		TrafficType:           user.TrafficType,
		TrafficLimitGB:        bytesToGiB(user.TrafficSize),
		TrafficConsumedGB:     bytesToGiB(consumedBytes),
		TrafficRemainingGB:    bytesToGiB(remainingBytes),
		RxGB:                  bytesToGiB(rxBytes),
		TxGB:                  bytesToGiB(txBytes),
		TrafficLimitBytes:     user.TrafficSize,
		TrafficConsumedBytes:  consumedBytes,
		TrafficRemainingBytes: remainingBytes,
		RxBytes:               rxBytes,
		TxBytes:               txBytes,
	}, nil
}

func isMonthlyTrafficType(trafficType string) bool {
	switch trafficType {
	case models.MonthlyTransmit, models.MonthlyReceive, models.MonthlyRxTx:
		return true
	default:
		return false
	}
}

func consumedTrafficBytes(trafficType string, rxBytes int64, txBytes int64) int64 {
	switch trafficType {
	case models.TotallyTransmit, models.MonthlyTransmit:
		return txBytes

	case models.TotallyReceive, models.MonthlyReceive:
		return rxBytes

	case models.TotallyRxTx, models.MonthlyRxTx, models.Free:
		return rxBytes + txBytes

	default:
		return rxBytes + txBytes
	}
}

func bytesToGiB(value int64) float64 {
	return float64(value) / float64(bytesInGiB)
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format("2006-01-02")
	return &formatted
}
