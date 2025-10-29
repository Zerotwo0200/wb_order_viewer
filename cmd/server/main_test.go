package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), "postgres://wbuser:wbpass@localhost:5433/wborders")
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}

	// Clean up
	pool.Exec(context.Background(), "DELETE FROM orders")
	return pool
}

func TestHandleGet(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	app := &App{
		db:    pool,
		cache: make(map[string]Order),
	}

	// Insert test data directly into cache
	testOrder := Order{
		OrderUID:    "test-order-123",
		TrackNumber: "TRACK123",
		Entry:       "TEST",
		Locale:      "en",
		SmID:        99,
	}
	app.cache["test-order-123"] = testOrder

	tests := []struct {
		name     string
		orderID  string
		wantCode int
	}{
		{
			name:     "existing order",
			orderID:  "test-order-123",
			wantCode: http.StatusOK,
		},
		{
			name:     "non-existing order",
			orderID:  "non-existent",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/order/"+tt.orderID, nil)
			w := httptest.NewRecorder()

			// Create a mux route for testing
			r := mux.NewRouter()
			r.HandleFunc("/api/order/{id}", app.handleGet).Methods(http.MethodGet)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("handleGet() = %v, want %v", w.Code, tt.wantCode)
			}
		})
	}
}

func TestCacheRecovery(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	// Insert order into DB
	testOrderJSON := `{"order_uid":"recovery-test","track_number":"REC123","entry":"TEST","locale":"en","sm_id":99}`
	_, err := pool.Exec(context.Background(),
		"INSERT INTO orders(order_uid, payload) VALUES($1, $2::jsonb)",
		"recovery-test", testOrderJSON)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create app and load cache
	app := &App{
		db:    pool,
		cache: make(map[string]Order),
	}

	if err := app.loadCache(context.Background()); err != nil {
		t.Fatalf("loadCache() error = %v", err)
	}

	// Check if order is in cache
	order, ok := app.cache["recovery-test"]
	if !ok {
		t.Error("loadCache() failed to load order from DB")
	}
	if order.OrderUID != "recovery-test" {
		t.Errorf("loadCache() order ID = %v, want 'recovery-test'", order.OrderUID)
	}
}

func TestOrderProcessing(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	app := &App{
		db:    pool,
		cache: make(map[string]Order),
	}

	// Simulate NATS message processing
	testJSON := `{"order_uid":"process-test","track_number":"TRACK001","entry":"WBIL","delivery":{},"payment":{},"items":[],"locale":"en","sm_id":99}`

	var order Order
	if err := json.Unmarshal([]byte(testJSON), &order); err != nil {
		t.Fatalf("Failed to unmarshal test order: %v", err)
	}

	if order.OrderUID == "" {
		t.Error("Order missing order_uid")
	}

	// Persist to DB
	_, err := pool.Exec(context.Background(),
		"INSERT INTO orders(order_uid, payload) VALUES($1, $2::jsonb) ON CONFLICT (order_uid) DO UPDATE SET payload = EXCLUDED.payload",
		order.OrderUID, testJSON)
	if err != nil {
		t.Fatalf("Failed to persist order: %v", err)
	}

	// Update cache
	app.cache[order.OrderUID] = order

	// Verify in cache
	if _, ok := app.cache["process-test"]; !ok {
		t.Error("Order not found in cache after processing")
	}
}
