package emsub

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	Id          uuid.UUID
	ServiceName string
	UserId      uuid.UUID
	Price       uint
	StartDate   time.Time
	EndDate     *time.Time
}
