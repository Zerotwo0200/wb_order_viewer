package main

import (
	"encoding/json"
	"log"
	"os"

	stan "github.com/nats-io/stan.go"
)

// Точка входа утилиты публикации: читает JSON из stdin и отправляет в NATS Streaming
func main() {
	// Конфигурация подключения
	clusterID := getenv("STAN_CLUSTER_ID", "wb-cluster")
	clientID := getenv("STAN_PUB_ID", "wb-publisher")
	natsURL := getenv("NATS_URL", "nats://localhost:4223")
	subject := getenv("STAN_SUBJECT", "orders")

	// Подключаемся к STAN
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		log.Fatalf("stan connect: %v", err)
	}
	defer sc.Close()

	// Читаем JSON‑заказ из стандартного ввода
	var payload map[string]any
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&payload); err != nil {
		log.Fatalf("read json from stdin: %v", err)
	}
	// Переупаковываем и публикуем
	b, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if err := sc.Publish(subject, b); err != nil {
		log.Fatalf("publish: %v", err)
	}
	log.Printf("published %d bytes to %s", len(b), subject)
}

// getenv — получить переменную окружения с дефолтом
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
