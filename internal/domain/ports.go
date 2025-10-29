package domain

import "context"

// OrderRepository — порт для операций персистентности заказов.
type OrderRepository interface {
	Upsert(ctx context.Context, id string, raw []byte) error
	LoadAll(ctx context.Context, fn func(id string, raw []byte) error) error
}

// OrderCache — порт быстрого доступа к заказам (кэш).
type OrderCache interface {
	Get(id string) (Order, bool)
	Set(id string, o Order)
}

// MessageSubscriber — порт подписчика на входящие сообщения заказов.
type MessageSubscriber interface {
	// Subscribe регистрирует обработчик; ack/повторные доставки реализует адаптер.
	Subscribe(ctx context.Context, handler func(ctx context.Context, raw []byte) error) error
}

// Общие доменные ошибки
var (
	ErrNotFound   = notFoundError("not found")
	ErrValidation = validationError("invalid data")
)

type notFoundError string

func (e notFoundError) Error() string { return string(e) }

type validationError string

func (e validationError) Error() string { return string(e) }
