package nats

import (
	"log"

	stan "github.com/nats-io/stan.go" // без /v2
)

// Subscribe подключается к NATS Streaming и возвращает соединение и подписку.
// Вызывать Close на subscription не обязательно, но позволяет отписаться аккуратно
// во время graceful shutdown.
func Subscribe(clusterID, clientID, channel string, handler func(msg *stan.Msg)) (stan.Conn, stan.Subscription, error) {
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		return nil, nil, err
	}

	sub, err := sc.Subscribe(
		channel,
		handler,
		stan.DeliverAllAvailable(),
		stan.DurableName("orders-svc"),
		stan.MaxInflight(25),
	)
	if err != nil {
		return nil, nil, err
	}

	log.Println("Subscribed to", channel)
	return sc, sub, nil
}
