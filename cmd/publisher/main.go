package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	stan "github.com/nats-io/stan.go"
)

func main() {
	var (
		file      = flag.String("f", "model.json", "path to JSON payload")
		clusterID = flag.String("cluster", "test-cluster", "NATS Streaming cluster ID")
		clientID  = flag.String("client", "publisher", "NATS Streaming client ID")
		subject   = flag.String("subject", "orders", "subject to publish to")
		natsURL   = flag.String("url", "nats://localhost:4222", "NATS Streaming server URL")
	)
	flag.Parse() // разбираем переданные флаги

	payload, err := os.ReadFile(*file) // читаем JSON из файла
	if err != nil {
		log.Fatalf("read payload: %v", err)
	}

	if !json.Valid(payload) { // проверяем корректность JSON
		log.Fatal("file does not contain valid JSON")
	}

	sc, err := stan.Connect(*clusterID, fmt.Sprintf("%s-%d", *clientID, randInt()), stan.NatsURL(*natsURL)) // подключаемся к NATS Streaming, добавляя случайный хвост к clientID
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer sc.Close() // гарантируем закрытие соединения после завершения

	if err := sc.Publish(*subject, payload); err != nil { // отправляем сообщение
		log.Fatalf("publish: %v", err)
	}

	log.Printf("published %d bytes to %s", len(payload), *subject) // логируем факт отправки и размер
}

func randInt() int64 { // randInt выдаёт случайное неотрицательное число < 2^31
	n, err := rand.Int(rand.Reader, big.NewInt(1<<31)) // используем криптографический генератор
	if err != nil {
		panic(err)
	}
	return n.Int64()
}
