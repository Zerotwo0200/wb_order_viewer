package main

import (
    "encoding/json"
    "log"
    "os"

    stan "github.com/nats-io/stan.go"
)

func main() {
    clusterID := getenv("STAN_CLUSTER_ID", "wb-cluster")
    clientID := getenv("STAN_PUB_ID", "wb-publisher")
    natsURL := getenv("NATS_URL", "nats://localhost:4223")
    subject := getenv("STAN_SUBJECT", "orders")

    sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
    if err != nil {
        log.Fatalf("stan connect: %v", err)
    }
    defer sc.Close()

    var payload map[string]any
    dec := json.NewDecoder(os.Stdin)
    if err := dec.Decode(&payload); err != nil {
        log.Fatalf("read json from stdin: %v", err)
    }
    b, err := json.Marshal(payload)
    if err != nil {
        log.Fatalf("marshal: %v", err)
    }
    if err := sc.Publish(subject, b); err != nil {
        log.Fatalf("publish: %v", err)
    }
    log.Printf("published %d bytes to %s", len(b), subject)
}

func getenv(k, d string) string {
    if v := os.Getenv(k); v != "" { return v }
    return d
}


