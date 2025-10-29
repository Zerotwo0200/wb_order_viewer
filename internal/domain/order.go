package domain

import "encoding/json"

// Order — доменная сущность заказа.
type Order struct {
	OrderUID     string          `json:"order_uid"`
	TrackNumber  string          `json:"track_number"`
	Entry        string          `json:"entry"`
	Delivery     json.RawMessage `json:"delivery"`
	Payment      json.RawMessage `json:"payment"`
	Items        json.RawMessage `json:"items"`
	Locale       string          `json:"locale"`
	InternalSign string          `json:"internal_signature"`
	CustomerID   string          `json:"customer_id"`
	DeliverySrv  string          `json:"delivery_service"`
	Shardkey     string          `json:"shardkey"`
	SmID         int             `json:"sm_id"`
	DateCreated  string          `json:"date_created"`
	OofShard     string          `json:"oof_shard"`
}
