package natsstan

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/example/wb-order-service/internal/domain"
	stan "github.com/nats-io/stan.go"
)

type Subscriber struct {
	ClusterID string
	ClientID  string
	URL       string
	Subject   string
	Durable   string
}

func (s *Subscriber) Subscribe(ctx context.Context, handler func(ctx context.Context, raw []byte) error) error {
	clientID := s.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("rwb-svc-%d", time.Now().UnixNano())
	}
	sc, err := stan.Connect(s.ClusterID, clientID, stan.NatsURL(s.URL))
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		sc.Close()
	}()
	_, err = sc.QueueSubscribe(s.Subject, "rwb-workers", func(m *stan.Msg) {
		hCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := handler(hCtx, m.Data); err != nil {
			// не подтверждаем, даём сообщению переотправиться
			log.Printf("handler error: %v", err)
			return
		}
		if err := m.Ack(); err != nil {
			log.Printf("ack failed: %v", err)
		}
	}, stan.DurableName(s.Durable), stan.SetManualAckMode(), stan.AckWait(10*time.Second), stan.DeliverAllAvailable())
	return err
}

var _ domain.MessageSubscriber = (*Subscriber)(nil)
