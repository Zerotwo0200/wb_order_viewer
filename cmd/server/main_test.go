package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/wb-order-service/internal/adapter/cache"
	"github.com/example/wb-order-service/internal/adapter/httpapi"
	"github.com/example/wb-order-service/internal/adapter/repo"
	"github.com/example/wb-order-service/internal/domain"
	"github.com/example/wb-order-service/internal/usecase"
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
	// Use in-memory cache and HTTP adapter
	orderCache := cache.NewMemoryOrderCache()
	orderCache.Set("test-order-123", domain.Order{
		OrderUID:    "test-order-123",
		TrackNumber: "TRACK123",
		Entry:       "TEST",
		Locale:      "en",
		SmID:        99,
	})
	ucGet := usecase.GetOrderByID{Cache: orderCache}
	srv := httpapi.NewServer(ucGet)

	tests := []struct {
		name     string
		orderID  string
		wantCode int
	}{
		{name: "existing order", orderID: "test-order-123", wantCode: http.StatusOK},
		{name: "non-existing order", orderID: "non-existent", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/order/"+tt.orderID, nil)
			w := httptest.NewRecorder()
			srv.Router.ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Errorf("GET status = %v, want %v", w.Code, tt.wantCode)
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

	// Use usecase.LoadCache
	orderCache := cache.NewMemoryOrderCache()
	uc := usecase.LoadCache{Repo: repo.NewPostgresOrderRepo(pool), Cache: orderCache}
	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("LoadCache error = %v", err)
	}
	if _, ok := orderCache.Get("recovery-test"); !ok {
		t.Error("LoadCache failed to load order from DB")
	}
}

func TestOrderProcessing(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	orderCache := cache.NewMemoryOrderCache()
	uc := usecase.ProcessIncomingOrder{Repo: repo.NewPostgresOrderRepo(pool), Cache: orderCache}

	testJSON := `{"order_uid":"process-test","track_number":"TRACK001","entry":"WBIL","delivery":{},"payment":{},"items":[],"locale":"en","sm_id":99}`
	if err := uc.Execute(context.Background(), []byte(testJSON)); err != nil {
		t.Fatalf("ProcessIncomingOrder error: %v", err)
	}
	if _, ok := orderCache.Get("process-test"); !ok {
		t.Error("Order not found in cache after processing")
	}
}

// poolForNoop is a helper for creating a pool when not used (kept to satisfy constructors if needed)
func poolForNoop() *pgxpool.Pool { return nil }
