package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/wb-order-service/internal/adapter/cache"
	"github.com/example/wb-order-service/internal/adapter/httpapi"
	"github.com/example/wb-order-service/internal/adapter/natsstan"
	"github.com/example/wb-order-service/internal/adapter/repo"
	"github.com/example/wb-order-service/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Точка входа сервиса: сборка зависимостей, запуск HTTP и подписки NATS/STAN
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbURL := getEnv("DATABASE_URL", "postgres://wbuser:wbpass@localhost:5433/wborders")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// Гарантируем наличие схемы БД (idempotent)
	if err := repo.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	// Сборка зависимостей (репозиторий, кэш, юзкейсы)
	orderRepo := repo.NewPostgresOrderRepo(pool)
	orderCache := cache.NewMemoryOrderCache()
	ucGet := usecase.GetOrderByID{Cache: orderCache}
	ucLoad := usecase.LoadCache{Repo: orderRepo, Cache: orderCache}
	ucIn := usecase.ProcessIncomingOrder{Repo: orderRepo, Cache: orderCache}

	if err := ucLoad.Execute(ctx); err != nil {
		log.Fatalf("load cache: %v", err)
	}

	// HTTP‑сервер через адаптер (REST + статика из web/)
	srv := &http.Server{Addr: ":8080", Handler: httpapi.NewServer(ucGet).Router}
	go func() {
		log.Printf("http listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// Подписка NATS/STAN через адаптер (обработка входящих заказов)
	go func() {
		sub := &natsstan.Subscriber{
			ClusterID: getEnv("STAN_CLUSTER_ID", "wb-cluster"),
			ClientID:  getEnv("STAN_CLIENT_ID", ""),
			URL:       getEnv("NATS_URL", "nats://localhost:4223"),
			Subject:   getEnv("STAN_SUBJECT", "orders"),
			Durable:   "wb-durable",
		}
		_ = sub.Subscribe(ctx, func(c context.Context, raw []byte) error {
			c2, cancel := context.WithTimeout(c, 3*time.Second)
			defer cancel()
			return ucIn.Execute(c2, raw)
		})
	}()

	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

// getEnv — получить переменную окружения с дефолтом
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
