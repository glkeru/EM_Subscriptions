package emsub

import "github.com/google/uuid"

const DateFormat = "01-2006"

type SubscriptionFull struct {
	Id          uuid.UUID `json:"id"`
	ServiceName string    `json:"service_name"`
	UserId      uuid.UUID `json:"user_id"`
	Price       uint      `json:"price"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date,omitempty"`
}

type SubscriptionCreateResponse struct {
	Id uuid.UUID `json:"id"`
}

type SubscriptionListResponse struct {
	Data   []SubscriptionFull `json:"data"`
	Limit  int                `json:"limit,omitempty"`
	Offset int                `json:"offset,omitempty"`
}

type SubscriptionTotalResponse struct {
	Price uint `json:"total"`
}
