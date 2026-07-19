package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/mmtaee/ocserv-dashboard/api/internal/repository"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/request"
	"github.com/mmtaee/ocserv-dashboard/common/models"
)

type fakeGatewaySubscriptionRepository struct {
	repository.OcservUserRepositoryInterface

	updateCalled bool
	update       repository.GatewaySubscriptionUpdate
}

func (repo *fakeGatewaySubscriptionRepository) UpdateGatewaySubscription(
	_ context.Context,
	username string,
	update repository.GatewaySubscriptionUpdate,
) (*models.OcservUser, error) {
	repo.updateCalled = true
	repo.update = update

	trafficSize := int64(0)
	if update.TrafficSize != nil {
		trafficSize = *update.TrafficSize
	}

	return &models.OcservUser{
		UID:         "remote-id",
		Username:    username,
		Group:       "default",
		TrafficType: models.TotallyRxTx,
		TrafficSize: trafficSize,
	}, nil
}

func TestUpdateUserSubscriptionAcceptsExactTrafficLimitBytes(
	t *testing.T,
) {
	const trafficLimitBytes int64 = 50*bytesInGiB + bytesInGiB/2

	repo := &fakeGatewaySubscriptionRepository{}
	ctl := &Controller{
		request:        request.NewCustomRequest(),
		ocservUserRepo: repo,
	}

	recorder := performGatewaySubscriptionUpdate(
		t,
		ctl,
		fmt.Sprintf(
			`{"traffic_limit_bytes":%d}`,
			trafficLimitBytes,
		),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"UpdateUserSubscription() status = %d, want %d; body = %s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	if repo.update.TrafficSize == nil ||
		*repo.update.TrafficSize != trafficLimitBytes {
		t.Fatalf(
			"TrafficSize = %v, want %d",
			repo.update.TrafficSize,
			trafficLimitBytes,
		)
	}

	var response UserStatusResponse
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"json.Unmarshal() error = %v",
			err,
		)
	}

	if response.TrafficLimitBytes !=
		trafficLimitBytes {
		t.Fatalf(
			"TrafficLimitBytes = %d, want %d",
			response.TrafficLimitBytes,
			trafficLimitBytes,
		)
	}

	if response.TrafficLimitGB != 50.5 {
		t.Fatalf(
			"TrafficLimitGB = %v, want 50.5",
			response.TrafficLimitGB,
		)
	}
}

func TestUpdateUserSubscriptionKeepsTrafficLimitGBCompatibility(
	t *testing.T,
) {
	repo := &fakeGatewaySubscriptionRepository{}
	ctl := &Controller{
		request:        request.NewCustomRequest(),
		ocservUserRepo: repo,
	}

	recorder := performGatewaySubscriptionUpdate(
		t,
		ctl,
		`{"traffic_limit_gb":50}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"UpdateUserSubscription() status = %d, want %d; body = %s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	const want = 50 * bytesInGiB

	if repo.update.TrafficSize == nil ||
		*repo.update.TrafficSize != want {
		t.Fatalf(
			"TrafficSize = %v, want %d",
			repo.update.TrafficSize,
			want,
		)
	}
}

func TestUpdateUserSubscriptionRejectsBothTrafficLimitUnits(
	t *testing.T,
) {
	repo := &fakeGatewaySubscriptionRepository{}
	ctl := &Controller{
		request:        request.NewCustomRequest(),
		ocservUserRepo: repo,
	}

	recorder := performGatewaySubscriptionUpdate(
		t,
		ctl,
		`{
			"traffic_limit_gb": 50,
			"traffic_limit_bytes": 53687091200
		}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"UpdateUserSubscription() status = %d, want %d; body = %s",
			recorder.Code,
			http.StatusBadRequest,
			recorder.Body.String(),
		)
	}

	if repo.updateCalled {
		t.Fatal(
			"UpdateGatewaySubscription() was called for an invalid request",
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		"traffic_limit_gb and traffic_limit_bytes cannot be used together",
	) {
		t.Fatalf(
			"response body = %q, want conflicting traffic limit error",
			recorder.Body.String(),
		)
	}
}

func performGatewaySubscriptionUpdate(
	t *testing.T,
	ctl *Controller,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/gateway/users/test-user/subscription",
		strings.NewReader(body),
	)
	req.Header.Set(
		echo.HeaderContentType,
		echo.MIMEApplicationJSON,
	)

	recorder := httptest.NewRecorder()
	echoContext := e.NewContext(req, recorder)

	echoContext.SetPath(
		"/api/gateway/users/:username/subscription",
	)
	echoContext.SetParamNames("username")
	echoContext.SetParamValues("test-user")

	if err := ctl.UpdateUserSubscription(
		echoContext,
	); err != nil {
		t.Fatalf(
			"UpdateUserSubscription() error = %v",
			err,
		)
	}

	return recorder
}
