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
