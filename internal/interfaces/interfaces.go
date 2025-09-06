package emsub

import (
	"context"
	"time"

	model "github.com/glkeru/EM_Subscriptions/internal/model"
	"github.com/google/uuid"
)

type RepoSubcription interface {
	SubscriptionCreate(ctx context.Context, s model.Subscription) (uuid.UUID, error)
	SubscriptionRead(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
	SubscriptionUpdate(ctx context.Context, s model.Subscription) error
	SubscriptionPatch(ctx context.Context, id uuid.UUID, fields map[string]any) error
	SubscriptionDelete(ctx context.Context, id uuid.UUID) error
	SubscriptionList(ctx context.Context, user uuid.UUID, service_name string, start *time.Time, end *time.Time, limit int, offset int) ([]model.Subscription, error)
	SubscriptionTotal(ctx context.Context, user uuid.UUID, service_name string, start *time.Time, end *time.Time) (uint, error)
}
