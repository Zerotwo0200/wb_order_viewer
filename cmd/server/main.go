package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	stan "github.com/nats-io/stan.go"
)

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

type App struct {
	db    *pgxpool.Pool
	cache map[string]Order
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbURL := getEnv("DATABASE_URL", "postgres://wbuser:wbpass@localhost:5433/wborders")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := initSchema(ctx, pool); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	app := &App{db: pool, cache: make(map[string]Order)}
	if err := app.loadCache(ctx); err != nil {
		log.Fatalf("load cache: %v", err)
	}

	go subscribeNATS(app)

	r := mux.NewRouter()
	r.HandleFunc("/api/order/{id}", app.handleGet).Methods(http.MethodGet)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("web")))

	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		log.Printf("http listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS orders (
  order_uid text PRIMARY KEY,
  payload jsonb NOT NULL
);
`)
	return err
}

func (a *App) loadCache(ctx context.Context) error {
	rows, err := a.db.Query(ctx, `SELECT order_uid, payload FROM orders`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		var o Order
		if err := json.Unmarshal(raw, &o); err != nil {
			// skip corrupted row
			continue
		}
		a.cache[id] = o
	}
	return rows.Err()
}

func (a *App) handleGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	o, ok := a.cache[id]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(o)
}

func subscribeNATS(a *App) {
	clusterID := getEnv("STAN_CLUSTER_ID", "wb-cluster")
	clientID := getEnv("STAN_CLIENT_ID", fmt.Sprintf("wb-svc-%d", time.Now().UnixNano()))
	natsURL := getEnv("NATS_URL", "nats://localhost:4223")
	subject := getEnv("STAN_SUBJECT", "orders")

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		log.Printf("stan connect: %v", err)
		return
	}
	defer sc.Close()

	_, err = sc.QueueSubscribe(subject, "wb-workers", func(m *stan.Msg) {
		var o Order
		if err := json.Unmarshal(m.Data, &o); err != nil {
			log.Printf("invalid message: %v", err)
			// Don't ACK, let it timeout and redeliver
			return
		}
		if o.OrderUID == "" {
			log.Printf("missing order_uid")
			// Don't ACK
			return
		}
		// persist
		raw := m.Data
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := a.db.Exec(ctx, `INSERT INTO orders(order_uid, payload) VALUES($1, $2)
                                   ON CONFLICT (order_uid) DO UPDATE SET payload = EXCLUDED.payload`, o.OrderUID, raw)
		if err != nil {
			log.Printf("db upsert: %v", err)
			// Don't ACK, NATS will redeliver
			return
		}
		// cache
		a.cache[o.OrderUID] = o
		log.Printf("processed order %s", o.OrderUID)
		if err := m.Ack(); err != nil {
			log.Printf("ack failed: %v", err)
		}
	}, stan.DurableName("wb-durable"), stan.SetManualAckMode(), stan.AckWait(10*time.Second), stan.DeliverAllAvailable())
	if err != nil {
		log.Printf("stan subscribe: %v", err)
		return
	}

	// keep connection
	select {}
}
