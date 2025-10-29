package main

import (
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/example/wb-order-service/internal/adapter/cache"
    "github.com/example/wb-order-service/internal/adapter/httpapi"
    "github.com/example/wb-order-service/internal/domain"
    "github.com/example/wb-order-service/internal/usecase"
)

func BenchmarkHandleGet(b *testing.B) {
    // Build HTTP adapter with in-memory cache and seeded data
    orderCache := cache.NewMemoryOrderCache()
    for i := 0; i < 1000; i++ {
        orderCache.Set(fmt.Sprintf("order-%d", i), domain.Order{OrderUID: fmt.Sprintf("order-%d", i)})
    }
    ucGet := usecase.GetOrderByID{Cache: orderCache}
    router := httpapi.NewServer(ucGet).Router

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            orderID := fmt.Sprintf("order-%d", i%1000)
            req := httptest.NewRequest(http.MethodGet, "/api/order/"+orderID, nil)
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            i++
        }
    })
}

func BenchmarkCacheGet(b *testing.B) {
    c := cache.NewMemoryOrderCache()
    for i := 0; i < 10000; i++ {
        c.Set(fmt.Sprintf("order-%d", i), domain.Order{OrderUID: fmt.Sprintf("order-%d", i)})
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = c.Get(fmt.Sprintf("order-%d", i%10000))
    }
}
