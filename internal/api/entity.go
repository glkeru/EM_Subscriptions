package emsub

import "github.com/google/uuid"

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

type SubscriptionTotalRequest struct {
	ServiceName string    `json:"service_name"`
	UserId      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date,omitempty"`
	EndDate     string    `json:"end_date,omitempty"`
}

type SubscriptionTotalResponse struct {
	Price uint `json:"price"`
}
