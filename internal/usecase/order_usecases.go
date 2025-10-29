package usecase

import (
	"context"
	"encoding/json"

	"github.com/example/wb-order-service/internal/domain"
)

// GetOrderByID — получить заказ из кэша по идентификатору.
type GetOrderByID struct {
	Cache domain.OrderCache
}

func (uc GetOrderByID) Execute(id string) (domain.Order, bool) {
	return uc.Cache.Get(id)
}

// LoadCache — загрузить все заказы из репозитория в кэш при старте.
type LoadCache struct {
	Repo  domain.OrderRepository
	Cache domain.OrderCache
}

func (uc LoadCache) Execute(ctx context.Context) error {
	return uc.Repo.LoadAll(ctx, func(id string, raw []byte) error {
		var o domain.Order
		if err := json.Unmarshal(raw, &o); err != nil {
			// пропускаем битые записи, не прерывая полную загрузку
			return nil
		}
		uc.Cache.Set(id, o)
		return nil
	})
}

// ProcessIncomingOrder — сохранить входящее сообщение заказа и обновить кэш.
type ProcessIncomingOrder struct {
	Repo  domain.OrderRepository
	Cache domain.OrderCache
}

func (uc ProcessIncomingOrder) Execute(ctx context.Context, raw []byte) error {
	var o domain.Order
	if err := json.Unmarshal(raw, &o); err != nil {
		return err
	}
	if o.OrderUID == "" {
		return domain.ErrValidation
	}
	if err := uc.Repo.Upsert(ctx, o.OrderUID, raw); err != nil {
		return err
	}
	uc.Cache.Set(o.OrderUID, o)
	return nil
}
