package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BenchmarkHandleGet(b *testing.B) {
	pool, err := pgxpool.New(context.Background(), "postgres://wbuser:wbpass@localhost:5433/wborders")
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	app := &App{
		db:    pool,
		cache: make(map[string]Order),
	}

	// Populate cache
	for i := 0; i < 1000; i++ {
		orderID := fmt.Sprintf("order-%d", i)
		app.cache[orderID] = Order{
			OrderUID:    orderID,
			TrackNumber: fmt.Sprintf("TRACK-%d", i),
			Entry:       "TEST",
			Locale:      "en",
			SmID:        i,
		}
	}

	// Setup handler
	r := mux.NewRouter()
	r.HandleFunc("/api/order/{id}", app.handleGet).Methods(http.MethodGet)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			orderID := fmt.Sprintf("order-%d", i%1000)
			req := httptest.NewRequest(http.MethodGet, "/api/order/"+orderID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			i++
		}
	})
}

func BenchmarkCacheGet(b *testing.B) {
	app := &App{
		cache: make(map[string]Order),
	}

	// Populate cache
	for i := 0; i < 10000; i++ {
		orderID := fmt.Sprintf("order-%d", i)
		app.cache[orderID] = Order{
			OrderUID: orderID,
			Locale:   "en",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orderID := fmt.Sprintf("order-%d", i%10000)
		_, _ = app.cache[orderID]
	}
}
