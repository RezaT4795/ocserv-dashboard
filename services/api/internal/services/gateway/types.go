package gateway

type CreateUserData struct {
	Username       string  `json:"username" validate:"required,min=2,max=64"`
	Password       string  `json:"password" validate:"required,min=4,max=64"`
	Group          string  `json:"group" validate:"required"`
	Unlimited      bool    `json:"unlimited" validate:"omitempty"`
	ExpireAt       *string `json:"expire_at" validate:"omitempty" example:"2026-12-31"`
	TrafficType    string  `json:"traffic_type" validate:"required,oneof=Free MonthlyTransmit MonthlyReceive MonthlyRxTx TotallyTransmit TotallyReceive TotallyRxTx"`
	TrafficLimitGB int     `json:"traffic_limit_gb" validate:"omitempty,min=0,max=100000"`
	Description    string  `json:"description" validate:"omitempty,max=1024"`
}

type CreateUserResponse struct {
	RemoteUserID   string  `json:"remote_user_id"`
	Username       string  `json:"username"`
	Password       string  `json:"password"`
	Group          string  `json:"group"`
	Unlimited      bool    `json:"unlimited"`
	ExpireAt       *string `json:"expire_at,omitempty"`
	TrafficType    string  `json:"traffic_type"`
	TrafficLimitGB int     `json:"traffic_limit_gb"`
}

type UserStatusResponse struct {
	RemoteUserID          string  `json:"remote_user_id"`
	Username              string  `json:"username"`
	Group                 string  `json:"group"`
	Active                bool    `json:"active"`
	Locked                bool    `json:"locked"`
	Deactivated           bool    `json:"deactivated"`
	Unlimited             bool    `json:"unlimited"`
	ExpireAt              *string `json:"expire_at,omitempty"`
	DeactivatedAt         *string `json:"deactivated_at,omitempty"`
	TrafficType           string  `json:"traffic_type"`
	TrafficLimitGB        float64 `json:"traffic_limit_gb"`
	TrafficConsumedGB     float64 `json:"traffic_consumed_gb"`
	TrafficRemainingGB    float64 `json:"traffic_remaining_gb"`
	RxGB                  float64 `json:"rx_gb"`
	TxGB                  float64 `json:"tx_gb"`
	TrafficLimitBytes     int64   `json:"traffic_limit_bytes"`
	TrafficConsumedBytes  int64   `json:"traffic_consumed_bytes"`
	TrafficRemainingBytes int64   `json:"traffic_remaining_bytes"`
	RxBytes               int64   `json:"rx_bytes"`
	TxBytes               int64   `json:"tx_bytes"`
}

type UpdateUserSubscriptionData struct {
	TrafficLimitGB    *int    `json:"traffic_limit_gb" validate:"omitempty,min=1,max=100000"`
	ExpireAt          *string `json:"expire_at" validate:"omitempty" example:"2026-12-31"`
	Unlimited         bool    `json:"unlimited" validate:"omitempty"`
	ResetTrafficUsage bool    `json:"reset_traffic_usage" validate:"omitempty"`
	Activate          bool    `json:"activate" validate:"omitempty"`
}

type DeleteUserResponse struct {
	RemoteUserID string `json:"remote_user_id"`
	Username     string `json:"username"`
	Deleted      bool   `json:"deleted"`
}
