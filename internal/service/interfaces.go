package service

import (
	"context"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
)

type OrderCache interface {
	GiveOrderByUID(UID string) (entity.Order, error)
	SaveOrder(ctx context.Context, o entity.Order) error
	LoadCache(ctx context.Context) error
}
